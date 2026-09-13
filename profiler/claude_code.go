package profiler

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// OTel signal names emitted by Claude Code. Named once and read once: the same
// extractor decides both what the probe can advertise and what the capture
// returns, so there is no second copy of a name to drift.
const (
	otelTokenUsageMetric = "claude_code.token.usage"
	otelToolDecisionLog  = "claude_code.tool_decision"
	otelAPIRequestLog    = "claude_code.api_request"
)

// ClaudeCodeAdapter captures runtime signals from Claude Code via OTel export.
//
// Claude Code emits OTel when CLAUDE_CODE_ENABLE_TELEMETRY=1 and OTEL_*_EXPORTER
// env vars are set. This adapter reads from a file the OTel collector writes
// (or a file-based exporter), not a live network endpoint, to keep the adapter
// self-contained and testable without a running collector.
//
// Skill activation and attribution are always unknown for Claude Code: it emits
// no skill activation event, and this adapter reads none of the skill-level
// attributes Claude Code does attach elsewhere in its OTel surface. Reporting
// either signal would mean inventing it.
type ClaudeCodeAdapter struct {
	// OtelExportFile is the path to a JSON file containing OTel-exported data.
	// It is the adapter's only input: Probe and Capture both resolve this one
	// path, and nothing else supplies one.
	OtelExportFile string
}

// Name returns the harness identifier.
func (a ClaudeCodeAdapter) Name() string { return "claude_code" }

// otelSignals is one export file resolved into the three signals it carries.
//
// Probe and Capture both derive from this single resolution, so "the adapter can
// produce this signal" and "the adapter produced this signal" are by construction
// the same question, answered by the same code. Two separate predicates — one
// scanning for structure, one extracting values — are what let the probe
// advertise data the capture then discarded.
type otelSignals struct {
	Tokens    TokenResult
	ToolCalls ToolCallResult
	Timing    TimingResult
}

// resolve reads and parses the export file once and settles all three signals.
//
// It owns file resolution and failure classification for this adapter; nothing
// below it re-decides either, so all three signals share one error path:
//
//   - no file configured — unknown, naming what to configure;
//   - file unreadable or unparseable — error, naming the failure. The export was
//     supplied, so "not configured" would send the caller to fix the one thing
//     that is not wrong;
//   - parsed — each extractor settles its own signal from what it can read.
func (a ClaudeCodeAdapter) resolve() otelSignals {
	if a.OtelExportFile == "" {
		return unknownSignals("OTel export not configured. Provide an OTel export file via --otel-file or OtelExportFile.")
	}

	data, err := os.ReadFile(a.OtelExportFile)
	if err != nil {
		return erroredSignals(fmt.Sprintf("failed to read OTel export file: %v", err))
	}

	var export claudeCodeOtelExport
	if err := json.Unmarshal(data, &export); err != nil {
		return erroredSignals(fmt.Sprintf("failed to parse OTel export JSON: %v", err))
	}

	return otelSignals{
		Tokens:    extractTokenCounts(export),
		ToolCalls: extractToolCalls(export),
		Timing:    extractTiming(export),
	}
}

// unknownSignals and erroredSignals settle every signal the same way, for the
// failures that belong to the export as a whole rather than to one signal.

func unknownSignals(reason string) otelSignals {
	return otelSignals{
		Tokens:    UnknownTokenResult(reason),
		ToolCalls: UnknownToolCallResult(reason),
		Timing:    UnknownTimingResult(reason),
	}
}

func erroredSignals(reason string) otelSignals {
	return otelSignals{
		Tokens:    ErrorTokenResult(reason),
		ToolCalls: ErrorToolCallResult(reason),
		Timing:    ErrorTimingResult(reason),
	}
}

// capabilityReport derives the capability report from resolved signals, so a
// capability is advertised exactly when a value was read for it.
func (a ClaudeCodeAdapter) capabilityReport(sig otelSignals) CapabilityReport {
	return CapabilityReport{
		Harness:    a.Name(),
		AdapterVer: AdapterVersion,
		ProbedAt:   time.Now().UTC().Format(time.RFC3339),
		Capabilities: map[MetricName]MetricSource{
			MetricTokens:    sourceOf(sig.Tokens.RawMetricResult),
			MetricToolCalls: sourceOf(sig.ToolCalls.RawMetricResult),
			MetricTiming:    sourceOf(sig.Timing.RawMetricResult),
			// This adapter reads no skill-level attributes, so it can offer no
			// activation or attribution signal whatever the OTel configuration.
			MetricSkillActivation: SourceNone,
			MetricAttribution:     SourceNone,
		},
	}
}

// sourceOf reports where a signal's value was read from, or "none" when no value
// was read. Availability is a fact about a value in hand, not about structure
// spotted in a file.
func sourceOf(r RawMetricResult) MetricSource {
	if r.State != MetricPresent {
		return SourceNone
	}
	return MetricSource(r.Source)
}

// Probe inspects the environment and returns what metrics this adapter can
// produce. A capability is "otel" only when the export file yielded a value for
// that signal; a missing, unreadable, malformed, or signal-less file reports
// "none".
func (a ClaudeCodeAdapter) Probe() CapabilityReport {
	return a.capabilityReport(a.resolve())
}

// Capture reads telemetry for a specific Claude Code session and produces a Profile.
func (a ClaudeCodeAdapter) Capture(sessionID string, opts CaptureOpts) (Profile, error) {
	sig := a.resolve()

	return Profile{
		Schema:       ProfileSchema,
		ProfiledAt:   time.Now().UTC().Format(time.RFC3339),
		Harness:      a.Name(),
		SessionID:    sessionID,
		SnapshotHash: opts.SnapshotHash,
		SkillDir:     opts.SkillDir,
		Capability:   a.capabilityReport(sig),

		Tokens:    sig.Tokens,
		ToolCalls: sig.ToolCalls,
		Timing:    sig.Timing,

		// Claude Code emits no skill activation event, and this adapter reads
		// none of the skill-level attributes it does emit.
		SkillActivation: UnknownActivationResult("Claude Code has no skill-level activation events"),
		Attribution:     UnknownAttributionResult("this adapter does not read Claude Code's skill attribution attributes"),
	}, nil
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

// Each extractor is the single predicate for its signal: present only when it
// read at least one usable value, unknown with a reason naming why not. A metric
// or event with the right name but nothing readable inside it is evidence that
// the harness was running, not evidence of a value.

func extractTokenCounts(data claudeCodeOtelExport) TokenResult {
	var tc TokenCounts
	seen, read := 0, 0
	for _, m := range data.Metrics {
		if m.Name != otelTokenUsageMetric {
			continue
		}
		seen++
		val, ok := toInt(m.Value)
		if !ok {
			continue
		}
		switch getString(m.Attributes, "token_type") {
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
		default:
			continue
		}
		read++
	}
	switch {
	case seen == 0:
		return UnknownTokenResult("no " + otelTokenUsageMetric + " metric found in OTel export")
	case read == 0:
		return UnknownTokenResult("no readable " + otelTokenUsageMetric + " metric in OTel export: none carried both a recognised token_type and a numeric value")
	}
	return PresentTokenResult(tc, string(SourceOtel))
}

func extractToolCalls(data claudeCodeOtelExport) ToolCallResult {
	var calls []ToolCallEntry
	seen := 0
	for _, log := range data.Logs {
		if log.EventName != otelToolDecisionLog {
			continue
		}
		seen++
		name := getString(log.Attributes, "tool_name")
		if name == "" {
			continue
		}
		calls = append(calls, ToolCallEntry{
			Name:      name,
			Timestamp: log.Timestamp,
			// Success carries the permission decision, not the execution
			// outcome: tool_decision says whether the call was approved, and
			// this adapter reads no completion signal. Separating the two is a
			// schema change, tracked for 0.5.0.
			Success: getString(log.Attributes, "decision") == "approved",
		})
	}
	switch {
	case seen == 0:
		return UnknownToolCallResult("no " + otelToolDecisionLog + " log events found in OTel export")
	case len(calls) == 0:
		return UnknownToolCallResult("no readable " + otelToolDecisionLog + " log events in OTel export: none carried a tool_name")
	}
	return PresentToolCallResult(calls, string(SourceOtel))
}

func extractTiming(data claudeCodeOtelExport) TimingResult {
	var first, last time.Time
	var start, end string
	seen, have := 0, false
	for _, log := range data.Logs {
		if log.EventName != otelAPIRequestLog {
			continue
		}
		seen++
		t := parseTime(log.Timestamp)
		if t.IsZero() {
			continue
		}
		// The span runs from the earliest event to the latest, not from the
		// first line to the last: an exporter is free to write events out of
		// order, and a session must not end before it starts.
		if !have || t.Before(first) {
			first, start = t, log.Timestamp
		}
		if !have || t.After(last) {
			last, end = t, log.Timestamp
		}
		have = true
	}
	switch {
	case seen == 0:
		return UnknownTimingResult("no " + otelAPIRequestLog + " log events found in OTel export")
	case !have:
		return UnknownTimingResult("no readable " + otelAPIRequestLog + " log events in OTel export: none carried a parseable RFC 3339 timestamp")
	}
	return PresentTimingResult(TimingData{
		StartTime: start,
		EndTime:   end,
		TotalMs:   last.Sub(first).Milliseconds(),
	}, string(SourceOtel))
}

// toInt reads a numeric JSON value. The second return distinguishes "the value
// was zero" from "the value was not a number" — the difference between a count
// that was read and a count that was invented.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		return int(i), err == nil
	}
	return 0, false
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
