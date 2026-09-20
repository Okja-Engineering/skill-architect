package profiler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)

// ClaudeCodeAdapter captures runtime signals from Claude Code via OTel or session transcript.
//
// OTel requires CLAUDE_CODE_ENABLE_TELEMETRY=1 and an export file. Session data reads the
// local JSONL transcript that Claude Code writes, with no collector required.
//
// Skill activation is present when the OTel export carries `skill.name`
// attributes on claude_code.token.usage / claude_code.cost.usage metrics —
// Claude Code's first-party per-skill attribution surface (documented as
// "Skill active for the request"). Attribution remains unknown: skill.name
// marks which skill was active, not which output it produced.
type ClaudeCodeAdapter struct {
	// OtelExportFile is the path to a JSON file containing OTel-exported data.
	OtelExportFile string
	// ExportFile is the path to a JSONL session transcript (session_data source).
	ExportFile string
}

// Name returns the harness identifier.
func (a ClaudeCodeAdapter) Name() string { return "claude_code" }

// Probe inspects the environment and returns what metrics this adapter can produce.
// OTel is reported as available only when an export file exists and contains
// the relevant signals; session_data is reported only when a JSONL transcript
// contains the corresponding signals. Empty or malformed files report "none".
func (a ClaudeCodeAdapter) Probe() CapabilityReport {
	caps := map[MetricName]MetricSource{
		MetricTokens:          SourceNone,
		MetricToolCalls:       SourceNone,
		MetricSkillActivation: SourceNone,
		MetricTiming:          SourceNone,
		MetricAttribution:     SourceNone,
	}

	otelProbed := false
	if a.OtelExportFile != "" {
		if _, err := os.Stat(a.OtelExportFile); err == nil {
			if tokens, toolCalls, timing, activation := hasOtelSignals(a.OtelExportFile); tokens || toolCalls || timing || activation {
				otelProbed = true
				if tokens {
					caps[MetricTokens] = SourceOtel
				}
				if toolCalls {
					caps[MetricToolCalls] = SourceOtel
				}
				if timing {
					caps[MetricTiming] = SourceOtel
				}
				if activation {
					caps[MetricSkillActivation] = SourceOtel
				}
			}
		}
	}

	if !otelProbed && a.ExportFile != "" {
		if _, err := os.Stat(a.ExportFile); err == nil {
			if tokens, toolCalls, timing := hasSessionDataSignals(a.ExportFile); tokens || toolCalls || timing {
				if tokens {
					caps[MetricTokens] = SourceSessionData
				}
				if toolCalls {
					caps[MetricToolCalls] = SourceSessionData
				}
				if timing {
					caps[MetricTiming] = SourceSessionData
				}
			}
		}
	}

	// Claude Code exposes skill.name on OTel metrics — activation is
	// probe-dependent (set above when present). Output attribution per skill
	// remains unavailable; no stable convention exists.

	return CapabilityReport{
		Harness:      a.Name(),
		AdapterVer:   AdapterVersion,
		ProbedAt:     time.Now().UTC().Format(time.RFC3339),
		Capabilities: caps,
	}
}

// Capture reads telemetry for a specific Claude Code session and produces a Profile.
func (a ClaudeCodeAdapter) Capture(sessionID string, opts CaptureOpts) (Profile, error) {
	cap := a.Probe()
	now := time.Now().UTC().Format(time.RFC3339)

	profile := Profile{
		Schema:       ProfileSchema,
		ProfiledAt:   now,
		Harness:      a.Name(),
		SessionID:    sessionID,
		SnapshotHash: opts.SnapshotHash,
		SkillDir:     opts.SkillDir,
		Capability:   cap,
	}

	// Attribution is always unknown; skill activation is populated below when
	// the OTel export carries skill.name attributes.
	profile.SkillActivation = UnknownActivationResult("no skill.name attributes in the available telemetry source")
	profile.Attribution = UnknownAttributionResult("Claude Code does not attribute outputs to skills")

	// If no telemetry source is available, all telemetry metrics are unknown.
	if cap.Capabilities[MetricTokens] == SourceNone &&
		cap.Capabilities[MetricToolCalls] == SourceNone &&
		cap.Capabilities[MetricTiming] == SourceNone &&
		cap.Capabilities[MetricSkillActivation] == SourceNone {
		noSourceReason := "No telemetry source detected. Provide an OTel export file or a JSONL session transcript via --otel-file or --export-file."
		profile.Tokens = UnknownTokenResult(noSourceReason)
		profile.ToolCalls = UnknownToolCallResult(noSourceReason)
		profile.Timing = UnknownTimingResult(noSourceReason)
		return profile, nil
	}

	// OTel source.
	if cap.Capabilities[MetricTokens] == SourceOtel ||
		cap.Capabilities[MetricToolCalls] == SourceOtel ||
		cap.Capabilities[MetricTiming] == SourceOtel ||
		cap.Capabilities[MetricSkillActivation] == SourceOtel {
		exportFile := a.OtelExportFile
		if exportFile == "" {
			exportFile = opts.ExportFile
		}

		if exportFile == "" {
			errReason := "OTel enabled but no export file path provided. Set OtelExportFile or opts.ExportFile."
			profile.Tokens = ErrorTokenResult(errReason)
			profile.ToolCalls = UnknownToolCallResult(errReason)
			profile.Timing = UnknownTimingResult(errReason)
			return profile, nil
		}

		data, err := os.ReadFile(exportFile)
		if err != nil {
			errReason := fmt.Sprintf("failed to read OTel export file: %v", err)
			profile.Tokens = ErrorTokenResult(errReason)
			profile.ToolCalls = UnknownToolCallResult(errReason)
			profile.Timing = UnknownTimingResult(errReason)
			return profile, nil
		}

		var otelData claudeCodeOtelExport
		if err := json.Unmarshal(data, &otelData); err != nil {
			errReason := fmt.Sprintf("failed to parse OTel export JSON: %v", err)
			profile.Tokens = ErrorTokenResult(errReason)
			profile.ToolCalls = UnknownToolCallResult(errReason)
			profile.Timing = UnknownTimingResult(errReason)
			return profile, nil
		}

		profile.Tokens = extractTokenCounts(otelData)
		profile.ToolCalls = extractToolCalls(otelData)
		profile.Timing = extractTiming(otelData)
		profile.SkillActivation = extractSkillActivation(otelData)

		return profile, nil
	}

	// session_data source.
	exportFile := a.ExportFile
	if exportFile == "" {
		exportFile = opts.ExportFile
	}

	if exportFile == "" {
		errReason := "session_data enabled but no transcript file path provided. Set ExportFile or opts.ExportFile."
		profile.Tokens = ErrorTokenResult(errReason)
		profile.ToolCalls = UnknownToolCallResult(errReason)
		profile.Timing = UnknownTimingResult(errReason)
		return profile, nil
	}

	tokens, toolCalls, timing, err := parseClaudeCodeTranscript(exportFile)
	if err != nil {
		errReason := fmt.Sprintf("failed to parse session transcript: %v", err)
		profile.Tokens = ErrorTokenResult(errReason)
		profile.ToolCalls = UnknownToolCallResult(errReason)
		profile.Timing = UnknownTimingResult(errReason)
		return profile, nil
	}

	profile.Tokens = tokens
	profile.ToolCalls = toolCalls
	profile.Timing = timing

	return profile, nil
}

// claudeCodeOtelExport is the JSON structure written by a file-based OTel exporter
// for Claude Code sessions. It contains metrics and log events.
type claudeCodeOtelExport struct {
	Metrics []otelMetric `json:"metrics,omitempty"`
	Logs    []otelLog    `json:"logs,omitempty"`
}

type otelMetric struct {
	Name       string         `json:"name"`
	Attributes map[string]any `json:"attributes,omitempty"`
	Value      any            `json:"value"`
}

type otelLog struct {
	EventName  string         `json:"event_name"`
	Attributes map[string]any `json:"attributes,omitempty"`
	Timestamp  string         `json:"timestamp"`
}

func extractTokenCounts(data claudeCodeOtelExport) TokenResult {
	var tc TokenCounts
	found := false
	for _, m := range data.Metrics {
		if m.Name == "claude_code.token.usage" {
			found = true
			tokenType, _ := m.Attributes["token_type"].(string)
			val := toInt(m.Value)
			switch tokenType {
			case "input":
				tc.Input += val
			case "output":
				tc.Output += val
			case "cache_read":
				tc.CacheRead += val
			case "cache_creation", "cache_write":
				tc.CacheWrite += val
			case "reasoning", "reasoning_output":
				tc.Reasoning += val
			}
		}
	}
	if !found {
		return UnknownTokenResult("no claude_code.token.usage metric found in OTel export")
	}
	return PresentTokenResult(tc, string(SourceOtel))
}

func extractToolCalls(data claudeCodeOtelExport) ToolCallResult {
	var calls []ToolCallEntry
	for _, log := range data.Logs {
		if log.EventName == "claude_code.tool_decision" {
			entry := ToolCallEntry{
				Name:      getString(log.Attributes, "tool_name"),
				Timestamp: log.Timestamp,
				Success:   getString(log.Attributes, "decision") == "approved",
			}
			calls = append(calls, entry)
		}
	}
	if len(calls) == 0 {
		return UnknownToolCallResult("no claude_code.tool_decision log events found in OTel export")
	}
	return PresentToolCallResult(calls, string(SourceOtel))
}

func extractTiming(data claudeCodeOtelExport) TimingResult {
	var start, end string
	for _, log := range data.Logs {
		if log.EventName == "claude_code.api_request" {
			if start == "" {
				start = log.Timestamp
			}
			end = log.Timestamp
		}
	}
	if start == "" {
		return UnknownTimingResult("no claude_code.api_request log events found in OTel export")
	}
	td := TimingData{StartTime: start, EndTime: end}
	if t1, e1 := parseTime(start), parseTime(end); !t1.IsZero() && !e1.IsZero() {
		td.TotalMs = e1.Sub(t1).Milliseconds()
	}
	return PresentTimingResult(td, string(SourceOtel))
}

// extractSkillActivation reads the `skill.name` attribute that Claude Code
// stamps on claude_code.token.usage and claude_code.cost.usage metrics. Each
// distinct name yields one activation entry — the attribute means "skill active
// for the request", which is cost-when-active attribution, not causation.
func extractSkillActivation(data claudeCodeOtelExport) ActivationResult {
	seen := map[string]bool{}
	for _, m := range data.Metrics {
		if m.Name != "claude_code.token.usage" && m.Name != "claude_code.cost.usage" {
			continue
		}
		if name := getString(m.Attributes, "skill.name"); name != "" {
			seen[name] = true
		}
	}
	if len(seen) == 0 {
		return UnknownActivationResult("no skill.name attributes on token/cost usage metrics in OTel export")
	}
	entries := make([]ActivationEntry, 0, len(seen))
	for name := range seen {
		entries = append(entries, ActivationEntry{SkillName: name})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].SkillName < entries[j].SkillName })
	return PresentActivationResult(entries, string(SourceOtel))
}

func hasOtelSignals(file string) (tokens, toolCalls, timing, activation bool) {
	data, err := os.ReadFile(file)
	if err != nil {
		return false, false, false, false
	}
	var otelData claudeCodeOtelExport
	if err := json.Unmarshal(data, &otelData); err != nil {
		return false, false, false, false
	}
	for _, m := range otelData.Metrics {
		if m.Name == "claude_code.token.usage" || m.Name == "claude_code.cost.usage" {
			if getString(m.Attributes, "skill.name") != "" {
				activation = true
			}
			if m.Name == "claude_code.token.usage" {
				tokens = true
			}
		}
	}
	for _, log := range otelData.Logs {
		switch log.EventName {
		case "claude_code.tool_decision":
			toolCalls = true
		case "claude_code.api_request":
			timing = true
		}
	}
	return tokens, toolCalls, timing, activation
}

// claudeCodeTranscriptRecord is the JSON shape of a line in Claude Code's session JSONL.
type claudeCodeTranscriptRecord struct {
	Timestamp string `json:"timestamp"`
	Message   struct {
		ID      string `json:"id"`
		Content []struct {
			Type      string `json:"type"`
			Name      string `json:"name"`
			ID        string `json:"id"`
			ToolUseID string `json:"tool_use_id"`
			IsError   bool   `json:"is_error"`
		} `json:"content"`
		Usage *struct {
			InputTokens              int `json:"input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			OutputTokensDetails      struct {
				ThinkingTokens int `json:"thinking_tokens"`
			} `json:"output_tokens_details"`
		} `json:"usage"`
	} `json:"message"`
}

func hasSessionDataSignals(file string) (tokens, toolCalls, timing bool) {
	f, err := os.Open(file)
	if err != nil {
		return false, false, false
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			if line != "" {
				// Process the final line that lacks a trailing newline.
			} else {
				break
			}
		} else if err != nil {
			break
		}

		var rec claudeCodeTranscriptRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		if rec.Message.Usage != nil {
			tokens = true
		}
		for _, c := range rec.Message.Content {
			if c.Type == "tool_use" {
				toolCalls = true
			}
		}
		if rec.Timestamp != "" {
			timing = true
		}
		if tokens && toolCalls && timing {
			return
		}
	}
	return tokens, toolCalls, timing
}

func parseClaudeCodeTranscript(file string) (TokenResult, ToolCallResult, TimingResult, error) {
	f, err := os.Open(file)
	if err != nil {
		return ErrorTokenResult(fmt.Sprintf("failed to read session transcript: %v", err)),
			UnknownToolCallResult("session transcript not read"),
			UnknownTimingResult("session transcript not read"),
			err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	lastByMsgID := make(map[string]claudeCodeTranscriptRecord)
	toolUses := make(map[string]*ToolCallEntry)
	resultErrors := make(map[string]bool)
	var minTime, maxTime time.Time
	validRecords := 0

	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			if line == "" {
				break
			}
		} else if err != nil {
			break
		}

		var rec claudeCodeTranscriptRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		validRecords++

		if rec.Message.ID != "" {
			lastByMsgID[rec.Message.ID] = rec
		}
		for _, c := range rec.Message.Content {
			if c.Type == "tool_use" {
				toolUses[c.ID] = &ToolCallEntry{
					Name:      c.Name,
					Timestamp: rec.Timestamp,
					Success:   true,
				}
			}
			if c.Type == "tool_result" {
				resultErrors[c.ToolUseID] = c.IsError
			}
		}
		if t, err := parseClaudeCodeTimestamp(rec.Timestamp); err == nil {
			if minTime.IsZero() || t.Before(minTime) {
				minTime = t
			}
			if maxTime.IsZero() || t.After(maxTime) {
				maxTime = t
			}
		}

		if err == io.EOF {
			break
		}
	}

	if validRecords == 0 {
		return ErrorTokenResult("no valid JSONL records in session transcript"),
			UnknownToolCallResult("no valid JSONL records in session transcript"),
			UnknownTimingResult("no valid JSONL records in session transcript"),
			err
	}

	var tc TokenCounts
	tokenFound := false
	for _, rec := range lastByMsgID {
		if rec.Message.Usage != nil {
			tokenFound = true
			u := rec.Message.Usage
			tc.Input += u.InputTokens
			tc.Output += u.OutputTokens
			tc.CacheWrite += u.CacheCreationInputTokens
			tc.CacheRead += u.CacheReadInputTokens
			tc.Reasoning += u.OutputTokensDetails.ThinkingTokens
		}
	}

	var calls []ToolCallEntry
	for id, entry := range toolUses {
		if isErr, ok := resultErrors[id]; ok {
			entry.Success = !isErr
		}
		calls = append(calls, *entry)
	}
	callFound := len(calls) > 0

	sort.Slice(calls, func(i, j int) bool {
		return calls[i].Timestamp < calls[j].Timestamp
	})

	var tokenRes TokenResult
	if tokenFound {
		tokenRes = PresentTokenResult(tc, string(SourceSessionData))
	} else {
		tokenRes = UnknownTokenResult("no usage data in session transcript")
	}

	var toolRes ToolCallResult
	if callFound {
		toolRes = PresentToolCallResult(calls, string(SourceSessionData))
	} else {
		toolRes = UnknownToolCallResult("no tool_use events in session transcript")
	}

	var timingRes TimingResult
	if !minTime.IsZero() && !maxTime.IsZero() {
		td := TimingData{
			StartTime: minTime.UTC().Format(time.RFC3339),
			EndTime:   maxTime.UTC().Format(time.RFC3339),
			TotalMs:   maxTime.Sub(minTime).Milliseconds(),
		}
		timingRes = PresentTimingResult(td, string(SourceSessionData))
	} else {
		timingRes = UnknownTimingResult("no timestamps in session transcript")
	}

	return tokenRes, toolRes, timingRes, nil
}

func parseClaudeCodeTimestamp(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	if s, ok := m[key].(string); ok {
		return s == "true" || s == "1"
	}
	if n, ok := m[key].(float64); ok {
		return n != 0
	}
	return false
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
