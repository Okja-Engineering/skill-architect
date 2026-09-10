package profiler

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ClaudeCodeAdapter captures runtime signals from Claude Code via OTel export.
//
// Claude Code emits OTel when CLAUDE_CODE_ENABLE_TELEMETRY=1 and OTEL_*_EXPORTER
// env vars are set. This adapter reads from a file the OTel collector writes
// (or a file-based exporter), not a live network endpoint, to keep the adapter
// self-contained and testable without a running collector.
//
// Skill activation and attribution are always unknown for Claude Code — it has
// no skill-level events in its OTel surface, only tool_decision (tool-level).
type ClaudeCodeAdapter struct {
	// OtelExportFile is the path to a JSON file containing OTel-exported data.
	// Required for both Probe and Capture to report OTel capabilities.
	OtelExportFile string
}

// Name returns the harness identifier.
func (a ClaudeCodeAdapter) Name() string { return "claude_code" }

// Probe inspects the environment and returns what metrics this adapter can produce.
// OTel is reported as available only when an export file exists — the env var
// alone indicates intent but Capture cannot deliver without a file to read.
func (a ClaudeCodeAdapter) Probe() CapabilityReport {
	caps := map[MetricName]MetricSource{
		MetricTokens:          SourceNone,
		MetricToolCalls:       SourceNone,
		MetricSkillActivation: SourceNone,
		MetricTiming:          SourceNone,
		MetricAttribution:     SourceNone,
	}

	otelAvailable := false
	if a.OtelExportFile != "" {
		if _, err := os.Stat(a.OtelExportFile); err == nil {
			otelAvailable = true
		}
	}

	if otelAvailable {
		caps[MetricTokens] = SourceOtel
		caps[MetricToolCalls] = SourceOtel
		caps[MetricTiming] = SourceOtel
	}

	// Claude Code has no skill-level activation or attribution events.
	// These remain "none" regardless of OTel configuration.

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

	// Skill activation and attribution are always unknown for Claude Code.
	profile.SkillActivation = UnknownActivationResult("Claude Code has no skill-level activation events")
	profile.Attribution = UnknownAttributionResult("Claude Code does not attribute outputs to skills")

	// If no OTel source is available, telemetry metrics are unknown.
	if cap.Capabilities[MetricTokens] == SourceNone {
		noOtelReason := "OTel export not configured. Provide an OTel export file via --otel-file or OtelExportFile."
		profile.Tokens = UnknownTokenResult(noOtelReason)
		profile.ToolCalls = UnknownToolCallResult(noOtelReason)
		profile.Timing = UnknownTimingResult(noOtelReason)
		return profile, nil
	}

	// Read OTel export file.
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

	// Parse the OTel export data.
	var otelData claudeCodeOtelExport
	if err := json.Unmarshal(data, &otelData); err != nil {
		errReason := fmt.Sprintf("failed to parse OTel export JSON: %v", err)
		profile.Tokens = ErrorTokenResult(errReason)
		profile.ToolCalls = UnknownToolCallResult(errReason)
		profile.Timing = UnknownTimingResult(errReason)
		return profile, nil
	}

	// Extract token counts from metrics.
	profile.Tokens = extractTokenCounts(otelData)

	// Extract tool calls from log events.
	profile.ToolCalls = extractToolCalls(otelData)

	// Extract timing from API request log events.
	profile.Timing = extractTiming(otelData)

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
			case "cache_creation":
				tc.CacheCreation += val
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

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
