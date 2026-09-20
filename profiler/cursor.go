package profiler

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// CursorAdapter captures runtime signals from Cursor via OTel or SQLite session data.
//
// Cursor emits enterprise OTel (tokens, api.request timing, skill.activation,
// hook.execution_complete for tool calls). When OTel is not available, the adapter
// recognises a Cursor state.vscdb SQLite file but does not yet parse it — it reports
// the capability as `sqlite` and honestly returns `unknown` until the schema is mapped.
// Attribution is not supported for any Cursor surface in this adapter version.
type CursorAdapter struct {
	// OtelExportFile is the path to a JSON file containing OTel-exported data.
	OtelExportFile string
	// ExportFile is the path to local session data (Cursor state.vscdb for SQLite fallback).
	ExportFile string
	// SpoolDir is the hook-capture spool directory (~/.cursor-profiler/spool).
	// When it contains events it is the preferred source: the only local
	// surface where skill_activation and attribution can be present.
	SpoolDir string
}

// Name returns the harness identifier.
func (a CursorAdapter) Name() string { return "cursor" }

// Probe inspects the environment and returns what metrics this adapter can produce.
func (a CursorAdapter) Probe() CapabilityReport {
	caps := map[MetricName]MetricSource{
		MetricTokens:          SourceNone,
		MetricToolCalls:       SourceNone,
		MetricSkillActivation: SourceNone,
		MetricTiming:          SourceNone,
		MetricAttribution:     SourceNone,
	}

	if a.OtelExportFile != "" {
		if _, err := os.Stat(a.OtelExportFile); err == nil {
			if tokens, toolCalls, timing, activation := hasCursorOtelSignals(a.OtelExportFile); tokens || toolCalls || timing || activation {
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

	if a.SpoolDir != "" && spoolHasEvents(a.SpoolDir) {
		caps[MetricToolCalls] = SourceHooks
		caps[MetricSkillActivation] = SourceHooks
		caps[MetricTiming] = SourceHooks
		caps[MetricAttribution] = SourceHooks
		// Tokens intentionally stay unset: hooks carry no billed counts.
	}

	if caps[MetricTokens] == SourceNone && a.ExportFile != "" {
		if _, err := os.Stat(a.ExportFile); err == nil {
			// Cursor SQLite file detected. The schema is not yet verified, but the
			// surface is reported so the capability is auditable.
			caps[MetricTokens] = SourceSQLite
		}
	}

	return CapabilityReport{
		Harness:      a.Name(),
		AdapterVer:   AdapterVersion,
		ProbedAt:     time.Now().UTC().Format(time.RFC3339),
		Capabilities: caps,
	}
}

// spoolHasEvents reports whether the spool directory holds any JSONL files.
func spoolHasEvents(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			return true
		}
	}
	return false
}

// Capture reads telemetry for a specific Cursor session and produces a Profile.
// A populated spool is the preferred source (hooks); OTel fills tokens when
// both are present.
func (a CursorAdapter) Capture(sessionID string, opts CaptureOpts) (Profile, error) {
	if a.SpoolDir != "" && spoolHasEvents(a.SpoolDir) {
		events, err := LoadSessionEvents(a.SpoolDir, sessionID)
		if err != nil {
			return Profile{}, err
		}
		if len(events) > 0 {
			profile := profileFromSpool(events, sessionID, opts, a.Probe())
			// OTel tokens are strictly additive when an export is also present.
			if a.OtelExportFile != "" {
				if data, err := os.ReadFile(a.OtelExportFile); err == nil {
					var otelData cursorOtelExport
					if json.Unmarshal(data, &otelData) == nil {
						if tok := extractCursorTokenCounts(otelData); tok.State == MetricPresent {
							profile.Tokens = tok
						}
					}
				}
			}
			return profile, nil
		}
		if a.OtelExportFile == "" {
			// Spool exists but holds nothing for this session, and no OTel
			// fallback — report honestly rather than emitting zero-value metrics.
			reason := "no events for session " + sessionID + " in spool"
			now := time.Now().UTC().Format(time.RFC3339)
			return Profile{
				Schema:                 ProfileSchema,
				ProfiledAt:             now,
				Harness:                a.Name(),
				SessionID:              sessionID,
				SnapshotHash:           opts.SnapshotHash,
				SkillDir:               opts.SkillDir,
				Capability:             a.Probe(),
				Tokens:                 UnknownTokenResult("Cursor hooks do not expose billed token counts"),
				ToolCalls:              UnknownToolCallResult(reason),
				Timing:                 UnknownTimingResult(reason),
				SkillActivation:        UnknownActivationResult(reason),
				Attribution:            UnknownAttributionResult(reason),
				EstimatedContextTokens: UnknownEstimatedTokensResult(reason),
			}, nil
		}
	}
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

	// Attribution is not supported for Cursor.
	profile.Attribution = UnknownAttributionResult("Cursor does not attribute outputs to skills at this adapter version")

	if cap.Capabilities[MetricTokens] == SourceNone &&
		cap.Capabilities[MetricToolCalls] == SourceNone &&
		cap.Capabilities[MetricTiming] == SourceNone &&
		cap.Capabilities[MetricSkillActivation] == SourceNone {
		noSourceReason := "No Cursor telemetry source detected. Provide an OTel export file or a SQLite state.vscdb file."
		profile.Tokens = UnknownTokenResult(noSourceReason)
		profile.ToolCalls = UnknownToolCallResult(noSourceReason)
		profile.Timing = UnknownTimingResult(noSourceReason)
		profile.SkillActivation = UnknownActivationResult(noSourceReason)
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
			errReason := "OTel enabled but no export file path provided."
			profile.Tokens = ErrorTokenResult(errReason)
			profile.ToolCalls = UnknownToolCallResult(errReason)
			profile.Timing = UnknownTimingResult(errReason)
			profile.SkillActivation = UnknownActivationResult(errReason)
			return profile, nil
		}

		data, err := os.ReadFile(exportFile)
		if err != nil {
			errReason := fmt.Sprintf("failed to read OTel export file: %v", err)
			profile.Tokens = ErrorTokenResult(errReason)
			profile.ToolCalls = UnknownToolCallResult(errReason)
			profile.Timing = UnknownTimingResult(errReason)
			profile.SkillActivation = UnknownActivationResult(errReason)
			return profile, nil
		}

		var otelData cursorOtelExport
		if err := json.Unmarshal(data, &otelData); err != nil {
			errReason := fmt.Sprintf("failed to parse OTel export JSON: %v", err)
			profile.Tokens = ErrorTokenResult(errReason)
			profile.ToolCalls = UnknownToolCallResult(errReason)
			profile.Timing = UnknownTimingResult(errReason)
			profile.SkillActivation = UnknownActivationResult(errReason)
			return profile, nil
		}

		profile.Tokens = extractCursorTokenCounts(otelData)
		profile.ToolCalls = extractCursorToolCalls(otelData)
		profile.Timing = extractCursorTiming(otelData)
		profile.SkillActivation = extractCursorSkillActivation(otelData)
		return profile, nil
	}

	// SQLite source.
	if cap.Capabilities[MetricTokens] == SourceSQLite {
		profile.Tokens = UnknownTokenResult("Cursor SQLite token schema not yet verified")
		profile.ToolCalls = UnknownToolCallResult("Cursor SQLite has no tool-call events in this adapter")
		profile.Timing = UnknownTimingResult("Cursor SQLite timing schema not yet verified")
		profile.SkillActivation = UnknownActivationResult("Cursor SQLite has no skill activation events")
		return profile, nil
	}

	return profile, nil
}

type cursorOtelExport struct {
	Metrics []otelMetric `json:"metrics,omitempty"`
	Logs    []otelLog    `json:"logs,omitempty"`
}

func hasCursorOtelSignals(file string) (tokens, toolCalls, timing, activation bool) {
	data, err := os.ReadFile(file)
	if err != nil {
		return false, false, false, false
	}
	var otelData cursorOtelExport
	if err := json.Unmarshal(data, &otelData); err != nil {
		return false, false, false, false
	}
	for _, m := range otelData.Metrics {
		if m.Name == "cursor.token.usage" {
			tokens = true
			break
		}
	}
	for _, m := range otelData.Metrics {
		if m.Name == "cursor.tool.calls" {
			toolCalls = true
			break
		}
	}
	for _, log := range otelData.Logs {
		switch log.EventName {
		case "cursor.api.request":
			timing = true
		case "cursor.skill.activated":
			activation = true
		}
	}
	return tokens, toolCalls, timing, activation
}

func extractCursorTokenCounts(data cursorOtelExport) TokenResult {
	var tc TokenCounts
	found := false
	for _, m := range data.Metrics {
		if m.Name == "cursor.token.usage" {
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
		return UnknownTokenResult("no cursor.token.usage metric found in OTel export")
	}
	return PresentTokenResult(tc, string(SourceOtel))
}

func extractCursorToolCalls(data cursorOtelExport) ToolCallResult {
	// cursor.tool.calls is the real tool-call surface: a delta-aggregated
	// metric keyed by cursor.tool.name / cursor.tool.status. The
	// cursor.hook.execution_complete log event tracks hook executions, not
	// tool calls, and must not be read as one (see cursor-telemetry-surfaces §2).
	var calls []ToolCallEntry
	for _, m := range data.Metrics {
		if m.Name != "cursor.tool.calls" {
			continue
		}
		status := getString(m.Attributes, "cursor.tool.status")
		if status == "" {
			status = getString(m.Attributes, "tool_status")
		}
		entry := ToolCallEntry{
			Name:  getString(m.Attributes, "cursor.tool.name"),
			Count: toInt(m.Value),
		}
		if entry.Name == "" {
			entry.Name = getString(m.Attributes, "tool_name")
		}
		entry.Success = status == "success" || status == ""
		if !entry.Success {
			entry.ErrorType = status
		}
		calls = append(calls, entry)
	}
	if len(calls) == 0 {
		return UnknownToolCallResult("no cursor.tool.calls metric found in OTel export")
	}
	return PresentToolCallResult(calls, string(SourceOtel))
}

func extractCursorTiming(data cursorOtelExport) TimingResult {
	var start, end string
	for _, log := range data.Logs {
		if log.EventName == "cursor.api.request" {
			if start == "" {
				start = log.Timestamp
			}
			end = log.Timestamp
		}
	}
	if start == "" {
		return UnknownTimingResult("no cursor.api.request log events found in OTel export")
	}
	td := TimingData{StartTime: start, EndTime: end}
	if t1, e1 := parseTime(start), parseTime(end); !t1.IsZero() && !e1.IsZero() {
		td.TotalMs = e1.Sub(t1).Milliseconds()
	}
	return PresentTimingResult(td, string(SourceOtel))
}

func extractCursorSkillActivation(data cursorOtelExport) ActivationResult {
	var events []ActivationEntry
	for _, log := range data.Logs {
		if log.EventName == "cursor.skill.activated" {
			events = append(events, ActivationEntry{
				SkillName: getString(log.Attributes, "skill_name"),
				Timestamp: log.Timestamp,
				Trigger:   getString(log.Attributes, "trigger"),
			})
		}
	}
	if len(events) == 0 {
		return UnknownActivationResult("no cursor.skill.activated log events found in OTel export")
	}
	return PresentActivationResult(events, string(SourceOtel))
}
