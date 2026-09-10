package profiler

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

// CodexAdapter captures runtime signals from Codex via OTel logs.
//
// Codex `codex exec` has a known bug where token metrics are not emitted; token counts
// live in `codex.sse_event` span/log attributes instead. Tool calls come from
// `codex.tool_decision` log events. Skill activation and attribution are unavailable.
type CodexAdapter struct {
	// OtelExportFile is the path to a JSON file containing OTel-exported data.
	OtelExportFile string
}

// Name returns the harness identifier.
func (a CodexAdapter) Name() string { return "codex" }

// Probe inspects the environment and returns what metrics this adapter can produce.
func (a CodexAdapter) Probe() CapabilityReport {
	caps := map[MetricName]MetricSource{
		MetricTokens:          SourceNone,
		MetricToolCalls:       SourceNone,
		MetricSkillActivation: SourceNone,
		MetricTiming:          SourceNone,
		MetricAttribution:     SourceNone,
	}

	if a.OtelExportFile != "" {
		if _, err := os.Stat(a.OtelExportFile); err == nil {
			if tokens, toolCalls, timing := hasCodexOtelSignals(a.OtelExportFile); tokens || toolCalls || timing {
				if tokens {
					caps[MetricTokens] = SourceOtel
				}
				if toolCalls {
					caps[MetricToolCalls] = SourceOtel
				}
				if timing {
					caps[MetricTiming] = SourceOtel
				}
			}
		}
	}

	return CapabilityReport{
		Harness:      a.Name(),
		AdapterVer:   AdapterVersion,
		ProbedAt:     time.Now().UTC().Format(time.RFC3339),
		Capabilities: caps,
	}
}

// Capture reads telemetry for a specific Codex session and produces a Profile.
func (a CodexAdapter) Capture(sessionID string, opts CaptureOpts) (Profile, error) {
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

	// Codex has no skill-level activation or attribution events.
	profile.SkillActivation = UnknownActivationResult("Codex has no skill-level activation events")
	profile.Attribution = UnknownAttributionResult("Codex does not attribute outputs to skills")

	if cap.Capabilities[MetricTokens] == SourceNone &&
		cap.Capabilities[MetricToolCalls] == SourceNone &&
		cap.Capabilities[MetricTiming] == SourceNone {
		noSourceReason := "No Codex OTel source detected. Provide an OTel export file."
		profile.Tokens = UnknownTokenResult(noSourceReason)
		profile.ToolCalls = UnknownToolCallResult(noSourceReason)
		profile.Timing = UnknownTimingResult(noSourceReason)
		return profile, nil
	}

	exportFile := a.OtelExportFile
	if exportFile == "" {
		exportFile = opts.ExportFile
	}

	if exportFile == "" {
		errReason := "OTel enabled but no export file path provided."
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

	var otelData codexOtelExport
	if err := json.Unmarshal(data, &otelData); err != nil {
		errReason := fmt.Sprintf("failed to parse OTel export JSON: %v", err)
		profile.Tokens = ErrorTokenResult(errReason)
		profile.ToolCalls = UnknownToolCallResult(errReason)
		profile.Timing = UnknownTimingResult(errReason)
		return profile, nil
	}

	profile.Tokens = extractCodexTokenCounts(otelData)
	profile.ToolCalls = extractCodexToolCalls(otelData)
	profile.Timing = extractCodexTiming(otelData)

	return profile, nil
}

type codexOtelExport struct {
	Metrics []otelMetric `json:"metrics,omitempty"`
	Logs    []otelLog    `json:"logs,omitempty"`
}

func hasCodexOtelSignals(file string) (tokens, toolCalls, timing bool) {
	data, err := os.ReadFile(file)
	if err != nil {
		return false, false, false
	}
	var otelData codexOtelExport
	if err := json.Unmarshal(data, &otelData); err != nil {
		return false, false, false
	}
	for _, log := range otelData.Logs {
		switch log.EventName {
		case "codex.sse_event":
			// Token counts live in log attributes, not metrics.
			if _, ok := log.Attributes["input_token_count"]; ok {
				tokens = true
			}
		case "codex.tool_decision":
			toolCalls = true
		case "codex.api.request":
			timing = true
		}
	}
	return tokens, toolCalls, timing
}

func extractCodexTokenCounts(data codexOtelExport) TokenResult {
	var tc TokenCounts
	found := false
	for _, log := range data.Logs {
		if log.EventName != "codex.sse_event" {
			continue
		}
		if v, ok := log.Attributes["input_token_count"]; ok {
			found = true
			tc.Input += toInt(v)
		}
		if v, ok := log.Attributes["output_token_count"]; ok {
			found = true
			tc.Output += toInt(v)
		}
		if v, ok := log.Attributes["cache_read_input_tokens"]; ok {
			found = true
			tc.CacheRead += toInt(v)
		}
		if v, ok := log.Attributes["cache_creation_input_tokens"]; ok {
			found = true
			tc.CacheCreation += toInt(v)
		}
		if v, ok := log.Attributes["reasoning_tokens"]; ok {
			found = true
			tc.Reasoning += toInt(v)
		}
	}
	if !found {
		return UnknownTokenResult("no codex.sse_event token attributes found in OTel export")
	}
	return PresentTokenResult(tc, string(SourceOtel))
}

func extractCodexToolCalls(data codexOtelExport) ToolCallResult {
	var calls []ToolCallEntry
	for _, log := range data.Logs {
		if log.EventName == "codex.tool_decision" {
			entry := ToolCallEntry{
				Name:      getString(log.Attributes, "tool_name"),
				Timestamp: log.Timestamp,
				Success:   getString(log.Attributes, "decision") == "approved",
			}
			calls = append(calls, entry)
		}
	}
	if len(calls) == 0 {
		return UnknownToolCallResult("no codex.tool_decision log events found in OTel export")
	}
	sort.Slice(calls, func(i, j int) bool {
		return calls[i].Timestamp < calls[j].Timestamp
	})
	return PresentToolCallResult(calls, string(SourceOtel))
}

func extractCodexTiming(data codexOtelExport) TimingResult {
	var start, end string
	for _, log := range data.Logs {
		if log.EventName == "codex.api.request" {
			if start == "" {
				start = log.Timestamp
			}
			end = log.Timestamp
		}
	}
	if start == "" {
		return UnknownTimingResult("no codex.api.request log events found in OTel export")
	}
	td := TimingData{StartTime: start, EndTime: end}
	if t1, e1 := parseTime(start), parseTime(end); !t1.IsZero() && !e1.IsZero() {
		td.TotalMs = e1.Sub(t1).Milliseconds()
	}
	return PresentTimingResult(td, string(SourceOtel))
}
