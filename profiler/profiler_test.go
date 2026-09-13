package profiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Slice 1 acceptance tests — verify the adapter interface, Claude Code adapter,
// and profile serialization against the spec in docs/profiler-spec.md.

// --- Acceptance criterion 1: MetricResult serialization ---

func TestMetricResultSerialization_Present(t *testing.T) {
	r := PresentTokenResult(TokenCounts{Input: 100, Output: 50}, "otel")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var parsed TokenResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.State != MetricPresent {
		t.Errorf("state = %q, want %q", parsed.State, MetricPresent)
	}
	if parsed.Source != "otel" {
		t.Errorf("source = %q, want %q", parsed.Source, "otel")
	}
	if parsed.Value == nil {
		t.Fatal("value is nil for present result")
	}
	if parsed.Value.Input != 100 || parsed.Value.Output != 50 {
		t.Errorf("value = %+v, want Input=100 Output=50", parsed.Value)
	}
	if parsed.Reason != "" {
		t.Errorf("reason = %q, want empty for present", parsed.Reason)
	}
}

func TestMetricResultSerialization_Unknown(t *testing.T) {
	r := UnknownTokenResult("no telemetry configured")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var parsed TokenResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.State != MetricUnknown {
		t.Errorf("state = %q, want %q", parsed.State, MetricUnknown)
	}
	if parsed.Reason != "no telemetry configured" {
		t.Errorf("reason = %q, want %q", parsed.Reason, "no telemetry configured")
	}
	if parsed.Value != nil {
		t.Errorf("value = %+v, want nil for unknown", parsed.Value)
	}
	if parsed.Source != "" {
		t.Errorf("source = %q, want empty for unknown", parsed.Source)
	}
}

func TestMetricResultSerialization_Error(t *testing.T) {
	r := ErrorTokenResult("collector unreachable")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var parsed TokenResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.State != MetricError {
		t.Errorf("state = %q, want %q", parsed.State, MetricError)
	}
	if parsed.Reason != "collector unreachable" {
		t.Errorf("reason = %q, want %q", parsed.Reason, "collector unreachable")
	}
	if parsed.Value != nil {
		t.Errorf("value = %+v, want nil for error", parsed.Value)
	}
}

// --- Acceptance criterion 2 & 3: CapabilityReport for Claude Code ---

func TestCapabilityReport_ClaudeCode_WithOtel(t *testing.T) {
	// Create a temp OTel export file so the adapter detects it as available.
	tmp := t.TempDir()
	otelFile := filepath.Join(tmp, "otel.json")
	otelData := `{
		"metrics": [
			{"name": "claude_code.token.usage", "attributes": {"token_type": "input"}, "value": 100}
		],
		"logs": [
			{"event_name": "claude_code.api_request", "timestamp": "2026-09-10T22:00:00Z"},
			{"event_name": "claude_code.tool_decision", "attributes": {"tool_name": "Bash", "decision": "approved"}, "timestamp": "2026-09-10T22:00:05Z"}
		]
	}`
	os.WriteFile(otelFile, []byte(otelData), 0644)

	adapter := ClaudeCodeAdapter{OtelExportFile: otelFile}
	cap := adapter.Probe()

	if cap.Harness != "claude_code" {
		t.Errorf("harness = %q, want claude_code", cap.Harness)
	}
	if cap.Capabilities[MetricTokens] != SourceOtel {
		t.Errorf("tokens = %q, want otel", cap.Capabilities[MetricTokens])
	}
	if cap.Capabilities[MetricToolCalls] != SourceOtel {
		t.Errorf("tool_calls = %q, want otel", cap.Capabilities[MetricToolCalls])
	}
	if cap.Capabilities[MetricTiming] != SourceOtel {
		t.Errorf("timing = %q, want otel", cap.Capabilities[MetricTiming])
	}
	if cap.Capabilities[MetricSkillActivation] != SourceNone {
		t.Errorf("skill_activation = %q, want none", cap.Capabilities[MetricSkillActivation])
	}
	if cap.Capabilities[MetricAttribution] != SourceNone {
		t.Errorf("attribution = %q, want none", cap.Capabilities[MetricAttribution])
	}
}

func TestCapabilityReport_ClaudeCode_WithoutOtel(t *testing.T) {
	// No OTel file — all capabilities should be "none".
	adapter := ClaudeCodeAdapter{}
	cap := adapter.Probe()

	for metric, source := range cap.Capabilities {
		if source != SourceNone {
			t.Errorf("%s = %q, want none", metric, source)
		}
	}
}

func TestCapabilityReport_ClaudeCode_EmptyOtelFile(t *testing.T) {
	tmp := t.TempDir()
	otelFile := filepath.Join(tmp, "otel.json")
	os.WriteFile(otelFile, []byte(`{}`), 0644)

	adapter := ClaudeCodeAdapter{OtelExportFile: otelFile}
	cap := adapter.Probe()

	for metric, source := range cap.Capabilities {
		if metric == MetricSkillActivation || metric == MetricAttribution {
			continue
		}
		if source != SourceNone {
			t.Errorf("%s = %q, want none for empty file", metric, source)
		}
	}
}

// --- Acceptance criterion 4: Claude Code session with OTel produces a profile ---

func TestCapture_ClaudeCode_WithOtelData(t *testing.T) {
	tmp := t.TempDir()
	otelFile := filepath.Join(tmp, "otel.json")
	otelData := `{
		"metrics": [
			{"name": "claude_code.token.usage", "attributes": {"token_type": "input"}, "value": 1500},
			{"name": "claude_code.token.usage", "attributes": {"token_type": "output"}, "value": 800},
			{"name": "claude_code.token.usage", "attributes": {"token_type": "cache_read"}, "value": 200},
			{"name": "claude_code.token.usage", "attributes": {"token_type": "reasoning"}, "value": 350}
		],
		"logs": [
			{"event_name": "claude_code.api_request", "timestamp": "2026-09-10T22:00:00Z"},
			{"event_name": "claude_code.tool_decision", "attributes": {"tool_name": "Bash", "decision": "approved"}, "timestamp": "2026-09-10T22:00:05Z"},
			{"event_name": "claude_code.tool_decision", "attributes": {"tool_name": "Read", "decision": "approved"}, "timestamp": "2026-09-10T22:00:10Z"},
			{"event_name": "claude_code.api_request", "timestamp": "2026-09-10T22:00:15Z"}
		]
	}`
	os.WriteFile(otelFile, []byte(otelData), 0644)

	adapter := ClaudeCodeAdapter{OtelExportFile: otelFile}
	opts := CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"}
	profile, err := adapter.Capture("session-001", opts)
	if err != nil {
		t.Fatal(err)
	}

	// Tokens should be present.
	if profile.Tokens.State != MetricPresent {
		t.Errorf("tokens state = %q, want present", profile.Tokens.State)
	}
	if profile.Tokens.Value == nil {
		t.Fatal("tokens value is nil")
	}
	if profile.Tokens.Value.Input != 1500 {
		t.Errorf("tokens input = %d, want 1500", profile.Tokens.Value.Input)
	}
	if profile.Tokens.Value.Output != 800 {
		t.Errorf("tokens output = %d, want 800", profile.Tokens.Value.Output)
	}
	if profile.Tokens.Value.CacheRead != 200 {
		t.Errorf("tokens cache_read = %d, want 200", profile.Tokens.Value.CacheRead)
	}
	if profile.Tokens.Value.Reasoning != 350 {
		t.Errorf("tokens reasoning = %d, want 350", profile.Tokens.Value.Reasoning)
	}
	if profile.Tokens.Source != "otel" {
		t.Errorf("tokens source = %q, want otel", profile.Tokens.Source)
	}

	// Tool calls should be present.
	if profile.ToolCalls.State != MetricPresent {
		t.Errorf("tool_calls state = %q, want present", profile.ToolCalls.State)
	}
	if len(profile.ToolCalls.Value) != 2 {
		t.Errorf("tool_calls count = %d, want 2", len(profile.ToolCalls.Value))
	}
	if profile.ToolCalls.Value[0].Name != "Bash" {
		t.Errorf("tool_calls[0] name = %q, want Bash", profile.ToolCalls.Value[0].Name)
	}

	// Timing should be present.
	if profile.Timing.State != MetricPresent {
		t.Errorf("timing state = %q, want present", profile.Timing.State)
	}
	if profile.Timing.Value == nil {
		t.Fatal("timing value is nil")
	}
	if profile.Timing.Value.StartTime != "2026-09-10T22:00:00Z" {
		t.Errorf("timing start = %q, want 2026-09-10T22:00:00Z", profile.Timing.Value.StartTime)
	}
	if profile.Timing.Value.TotalMs != 15000 {
		t.Errorf("timing total_ms = %d, want 15000", profile.Timing.Value.TotalMs)
	}

	// Skill activation should be unknown.
	if profile.SkillActivation.State != MetricUnknown {
		t.Errorf("skill_activation state = %q, want unknown", profile.SkillActivation.State)
	}
	if profile.SkillActivation.Reason == "" {
		t.Error("skill_activation reason should not be empty")
	}
	if profile.SkillActivation.Value != nil {
		t.Errorf("skill_activation value = %+v, want nil for unknown", profile.SkillActivation.Value)
	}

	// Attribution should be unknown.
	if profile.Attribution.State != MetricUnknown {
		t.Errorf("attribution state = %q, want unknown", profile.Attribution.State)
	}
	if profile.Attribution.Value != nil {
		t.Errorf("attribution value = %+v, want nil for unknown", profile.Attribution.Value)
	}
}

// --- Acceptance criterion 5: Session with no OTel produces all-unknown profile ---

func TestCapture_ClaudeCode_WithoutOtel(t *testing.T) {
	adapter := ClaudeCodeAdapter{}
	opts := CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"}
	profile, err := adapter.Capture("session-001", opts)
	if err != nil {
		t.Fatal(err)
	}

	if profile.Tokens.State != MetricUnknown {
		t.Errorf("tokens state = %q, want unknown", profile.Tokens.State)
	}
	if profile.Tokens.Value != nil {
		t.Errorf("tokens value = %+v, want nil for unknown", profile.Tokens.Value)
	}
	if profile.ToolCalls.State != MetricUnknown {
		t.Errorf("tool_calls state = %q, want unknown", profile.ToolCalls.State)
	}
	if profile.Timing.State != MetricUnknown {
		t.Errorf("timing state = %q, want unknown", profile.Timing.State)
	}
	if profile.Timing.Value != nil {
		t.Errorf("timing value = %+v, want nil for unknown", profile.Timing.Value)
	}
	if profile.SkillActivation.State != MetricUnknown {
		t.Errorf("skill_activation state = %q, want unknown", profile.SkillActivation.State)
	}
	if profile.Attribution.State != MetricUnknown {
		t.Errorf("attribution state = %q, want unknown", profile.Attribution.State)
	}
}

// --- Acceptance criterion 6: Profile JSON round-trips ---

func TestProfileRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	otelFile := filepath.Join(tmp, "otel.json")
	otelData := `{
		"metrics": [
			{"name": "claude_code.token.usage", "attributes": {"token_type": "input"}, "value": 100},
			{"name": "claude_code.token.usage", "attributes": {"token_type": "output"}, "value": 50}
		],
		"logs": [
			{"event_name": "claude_code.api_request", "timestamp": "2026-09-10T22:00:00Z"},
			{"event_name": "claude_code.api_request", "timestamp": "2026-09-10T22:00:10Z"}
		]
	}`
	os.WriteFile(otelFile, []byte(otelData), 0644)

	adapter := ClaudeCodeAdapter{OtelExportFile: otelFile}
	opts := CaptureOpts{SnapshotHash: "sha123", SkillDir: "/skills/test"}
	profile, err := adapter.Capture("sess-1", opts)
	if err != nil {
		t.Fatal(err)
	}

	// Marshal → Unmarshal → Marshal must be identical.
	first, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	var reparsed Profile
	if err := json.Unmarshal(first, &reparsed); err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(reparsed)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Errorf("round-trip mismatch:\nfirst:  %s\nsecond: %s", first, second)
	}
}

// --- Acceptance criterion 7: Profile schema field ---

func TestProfileSchemaField(t *testing.T) {
	adapter := ClaudeCodeAdapter{}
	profile, _ := adapter.Capture("s", CaptureOpts{SnapshotHash: "h", SkillDir: "/d"})
	if profile.Schema != ProfileSchema {
		t.Errorf("schema = %q, want %q", profile.Schema, ProfileSchema)
	}

	// Verify via JSON marshal too (custom MarshalJSON).
	data, _ := json.Marshal(profile)
	var raw map[string]any
	json.Unmarshal(data, &raw)
	if raw["schema"] != ProfileSchema {
		t.Errorf("json schema = %v, want %q", raw["schema"], ProfileSchema)
	}
}

// --- Acceptance criterion 8: Profile snapshot_hash matches input ---

func TestProfileSnapshotHash(t *testing.T) {
	adapter := ClaudeCodeAdapter{}
	profile, _ := adapter.Capture("s", CaptureOpts{SnapshotHash: "deadbeef", SkillDir: "/d"})
	if profile.SnapshotHash != "deadbeef" {
		t.Errorf("snapshot_hash = %q, want deadbeef", profile.SnapshotHash)
	}
}

// --- No value field in JSON for unknown/error states ---

func TestNoValueInJSONForUnknown(t *testing.T) {
	r := UnknownTokenResult("test reason")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	json.Unmarshal(data, &raw)
	if _, hasValue := raw["value"]; hasValue {
		t.Error("unknown result should not have a 'value' key in JSON")
	}
}

func TestNoValueInJSONForError(t *testing.T) {
	r := ErrorTokenResult("test error")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	json.Unmarshal(data, &raw)
	if _, hasValue := raw["value"]; hasValue {
		t.Error("error result should not have a 'value' key in JSON")
	}
}

func TestNoTimingValueInJSONForUnknown(t *testing.T) {
	r := UnknownTimingResult("no timing data")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	json.Unmarshal(data, &raw)
	if _, hasValue := raw["value"]; hasValue {
		t.Error("unknown timing result should not have a 'value' key in JSON")
	}
}

func TestNoAttributionValueInJSONForUnknown(t *testing.T) {
	r := UnknownAttributionResult("no attribution data")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	json.Unmarshal(data, &raw)
	if _, hasValue := raw["value"]; hasValue {
		t.Error("unknown attribution result should not have a 'value' key in JSON")
	}
}

// --- Probe/Capture consistency: the adapter contract ---
//
// Two invariants, stated once here and asserted over every export shape below.
//
//  1. "Present" means a value was read. A signal is present only when the export
//     carried something the adapter could actually assimilate. Structural
//     evidence alone — a metric with the right name but a token type or value
//     the adapter cannot read, a log event with no usable payload — is not a
//     value, and calling it "present" invents data the export never carried.
//  2. Probe and Capture apply the same predicate, per signal. Every capability
//     the report marks available is "present" in the profile with that source;
//     every capability it marks "none" is not present, carries a reason, and
//     carries no value.
//
// Both are pinned to the contract rather than to any particular gating,
// detection, or parsing implementation, so a rewrite of the adapter's internals
// still has to satisfy them.

// capturedSignal is one signal's state paired with whether the profile actually
// carries a value for it, so the contract can be checked over every capability
// the report enumerates instead of a hand-maintained subset.
type capturedSignal struct {
	raw      RawMetricResult
	hasValue bool
}

func capturedSignals(p Profile) map[MetricName]capturedSignal {
	return map[MetricName]capturedSignal{
		MetricTokens:          {p.Tokens.RawMetricResult, p.Tokens.Value != nil},
		MetricToolCalls:       {p.ToolCalls.RawMetricResult, len(p.ToolCalls.Value) > 0},
		MetricSkillActivation: {p.SkillActivation.RawMetricResult, len(p.SkillActivation.Value) > 0},
		MetricTiming:          {p.Timing.RawMetricResult, p.Timing.Value != nil},
		MetricAttribution:     {p.Attribution.RawMetricResult, p.Attribution.Value != nil},
	}
}

// Export fragments. The "usable" ones carry a value the adapter can read; the
// rest carry the right structure and nothing readable inside it.
const (
	tokenInput        = `{"name": "claude_code.token.usage", "attributes": {"token_type": "input"}, "value": 1500}`
	tokenOutput       = `{"name": "claude_code.token.usage", "attributes": {"token_type": "output"}, "value": 300}`
	tokenNoAttributes = `{"name": "claude_code.token.usage", "value": 1500}`
	tokenUnknownType  = `{"name": "claude_code.token.usage", "attributes": {"token_type": "mystery"}, "value": 1500}`
	tokenNonNumeric   = `{"name": "claude_code.token.usage", "attributes": {"token_type": "input"}, "value": "lots"}`
	toolLogBash       = `{"event_name": "claude_code.tool_decision", "attributes": {"tool_name": "Bash", "decision": "approved"}, "timestamp": "2026-09-10T22:00:05Z"}`
	toolLogNoName     = `{"event_name": "claude_code.tool_decision", "attributes": {"decision": "approved"}, "timestamp": "2026-09-10T22:00:05Z"}`
	apiLogStart       = `{"event_name": "claude_code.api_request", "timestamp": "2026-09-10T22:00:00Z"}`
	apiLogNoTimestamp = `{"event_name": "claude_code.api_request", "attributes": {"model": "claude"}}`
)

func TestCaptureDeliversEverySignalProbeAdvertises(t *testing.T) {
	cases := []struct {
		name string
		// export is written to a file the adapter is pointed at, unless one of
		// the two flags below says otherwise.
		export       string
		unconfigured bool         // adapter has no export file at all
		missingFile  bool         // adapter points at a path that does not exist
		present      []MetricName // signals that must be "present"
		absent       MetricState  // state required of tokens/tool_calls/timing when not present
	}{
		{name: "no export file configured", unconfigured: true, absent: MetricUnknown},
		{name: "export file path set but file missing", missingFile: true, absent: MetricError},
		{name: "empty export", export: `{}`, absent: MetricUnknown},
		{name: "malformed JSON", export: `{"metrics": [`, absent: MetricError},
		{name: "truncated after a valid prefix", export: `{"metrics": [{"name": "claude_code.token.usage"`, absent: MetricError},
		{name: "tokens only", export: `{"metrics": [` + tokenInput + `]}`,
			present: []MetricName{MetricTokens}, absent: MetricUnknown},
		{name: "tool calls only", export: `{"logs": [` + toolLogBash + `]}`,
			present: []MetricName{MetricToolCalls}, absent: MetricUnknown},
		{name: "timing only", export: `{"logs": [` + apiLogStart + `]}`,
			present: []MetricName{MetricTiming}, absent: MetricUnknown},
		{name: "tool calls and timing, no token metric", export: `{"logs": [` + toolLogBash + `, ` + apiLogStart + `]}`,
			present: []MetricName{MetricToolCalls, MetricTiming}, absent: MetricUnknown},
		{name: "all signals", export: `{"metrics": [` + tokenInput + `], "logs": [` + toolLogBash + `, ` + apiLogStart + `]}`,
			present: []MetricName{MetricTokens, MetricToolCalls, MetricTiming}, absent: MetricUnknown},
		{name: "token metric with no attributes", export: `{"metrics": [` + tokenNoAttributes + `]}`,
			absent: MetricUnknown},
		{name: "token metric with an unrecognised token_type", export: `{"metrics": [` + tokenUnknownType + `]}`,
			absent: MetricUnknown},
		{name: "token metric with a non-numeric value", export: `{"metrics": [` + tokenNonNumeric + `]}`,
			absent: MetricUnknown},
		{name: "readable token metric alongside unreadable ones",
			export:  `{"metrics": [` + tokenUnknownType + `, ` + tokenNonNumeric + `, ` + tokenOutput + `]}`,
			present: []MetricName{MetricTokens}, absent: MetricUnknown},
		{name: "tool_decision with no tool_name", export: `{"logs": [` + toolLogNoName + `]}`,
			absent: MetricUnknown},
		{name: "api_request with no timestamp", export: `{"logs": [` + apiLogNoTimestamp + `]}`,
			absent: MetricUnknown},
		{name: "api_request events out of chronological order",
			export:  `{"logs": [{"event_name": "claude_code.api_request", "timestamp": "2026-09-10T22:00:15Z"}, ` + apiLogStart + `]}`,
			present: []MetricName{MetricTiming}, absent: MetricUnknown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var adapter ClaudeCodeAdapter
			switch {
			case tc.unconfigured:
				adapter = ClaudeCodeAdapter{}
			case tc.missingFile:
				adapter = ClaudeCodeAdapter{OtelExportFile: filepath.Join(t.TempDir(), "absent.json")}
			default:
				otelFile := filepath.Join(t.TempDir(), "otel.json")
				if err := os.WriteFile(otelFile, []byte(tc.export), 0644); err != nil {
					t.Fatal(err)
				}
				adapter = ClaudeCodeAdapter{OtelExportFile: otelFile}
			}

			report := adapter.Probe()
			profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
			if err != nil {
				t.Fatal(err)
			}

			// The capability report is the denominator: every signal the
			// adapter knows about is walked, none assumed.
			if len(report.Capabilities) != 5 {
				t.Fatalf("capability report covers %d signals, want 5", len(report.Capabilities))
			}

			signals := capturedSignals(profile)
			for metric, advertised := range report.Capabilities {
				got, ok := signals[metric]
				if !ok {
					t.Fatalf("%s: capability reported but no result in the profile", metric)
				}
				wantPresent := false
				for _, m := range tc.present {
					if m == metric {
						wantPresent = true
					}
				}

				if wantPresent {
					if advertised == SourceNone {
						t.Errorf("%s: export carries a readable value, probe advertised %q", metric, advertised)
					}
					if got.raw.State != MetricPresent {
						t.Errorf("%s: probe advertised %q, capture state = %q (reason %q), want %q",
							metric, advertised, got.raw.State, got.raw.Reason, MetricPresent)
					}
					if got.raw.Source != string(advertised) {
						t.Errorf("%s: capture source = %q, want %q", metric, got.raw.Source, advertised)
					}
					if !got.hasValue {
						t.Errorf("%s: state %q with no value — a present signal must carry the value that was read",
							metric, got.raw.State)
					}
					if got.raw.Reason != "" {
						t.Errorf("%s: present result carries reason %q", metric, got.raw.Reason)
					}
					continue
				}

				if advertised != SourceNone {
					t.Errorf("%s: nothing readable in the export, probe advertised %q", metric, advertised)
				}
				if got.raw.State == MetricPresent {
					t.Errorf("%s: state = present with no readable value in the export", metric)
				}
				if got.hasValue {
					t.Errorf("%s: state %q carries a value; unknown and error results must carry none",
						metric, got.raw.State)
				}
				if got.raw.Reason == "" {
					t.Errorf("%s: state %q must carry a reason", metric, got.raw.State)
				}
				// Skill activation and attribution are a property of the
				// harness, not of this export, so they are always unknown.
				want := tc.absent
				if metric == MetricSkillActivation || metric == MetricAttribution {
					want = MetricUnknown
				}
				if got.raw.State != want {
					t.Errorf("%s: state = %q, want %q", metric, got.raw.State, want)
				}
			}
		})
	}
}

// An export that was supplied but cannot be used is diagnosable as exactly
// that. Telling the user to provide an export file they already provided sends
// them to fix the one thing that is not wrong.
func TestCapture_UnusableExport_IsDiagnosable(t *testing.T) {
	cases := []struct {
		name    string
		content string
		write   bool
		wantIn  string
	}{
		{"malformed JSON", `{"metrics": [`, true, "parse"},
		{"not JSON at all", "resourceMetrics: none\n", true, "parse"},
		{"file does not exist", "", false, "read"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			otelFile := filepath.Join(t.TempDir(), "otel.json")
			if tc.write {
				if err := os.WriteFile(otelFile, []byte(tc.content), 0644); err != nil {
					t.Fatal(err)
				}
			}
			adapter := ClaudeCodeAdapter{OtelExportFile: otelFile}
			profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
			if err != nil {
				t.Fatal(err)
			}

			for metric, got := range map[MetricName]RawMetricResult{
				MetricTokens:    profile.Tokens.RawMetricResult,
				MetricToolCalls: profile.ToolCalls.RawMetricResult,
				MetricTiming:    profile.Timing.RawMetricResult,
			} {
				if got.State != MetricError {
					t.Errorf("%s: state = %q (reason %q), want %q — the export existed and failed",
						metric, got.State, got.Reason, MetricError)
				}
				if !strings.Contains(got.Reason, tc.wantIn) {
					t.Errorf("%s: reason = %q, want it to name the %s failure", metric, got.Reason, tc.wantIn)
				}
				if strings.Contains(got.Reason, "not configured") {
					t.Errorf("%s: reason = %q — the export was configured; this sends the user to fix the wrong thing",
						metric, got.Reason)
				}
			}
		})
	}
}

// Timing is the span the api_request events cover, so it cannot run backwards
// however the exporter ordered them.
func TestTimingIsASpanNotFileOrder(t *testing.T) {
	otelFile := filepath.Join(t.TempDir(), "otel.json")
	otelData := `{
		"logs": [
			{"event_name": "claude_code.api_request", "timestamp": "2026-09-10T22:00:15Z"},
			{"event_name": "claude_code.api_request", "timestamp": "2026-09-10T22:00:00Z"},
			{"event_name": "claude_code.api_request", "timestamp": "2026-09-10T22:00:07Z"}
		]
	}`
	if err := os.WriteFile(otelFile, []byte(otelData), 0644); err != nil {
		t.Fatal(err)
	}

	adapter := ClaudeCodeAdapter{OtelExportFile: otelFile}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Timing.Value == nil {
		t.Fatalf("timing value is nil (state %q, reason %q)", profile.Timing.State, profile.Timing.Reason)
	}
	got := *profile.Timing.Value
	if got.StartTime != "2026-09-10T22:00:00Z" {
		t.Errorf("start_time = %q, want the earliest event 2026-09-10T22:00:00Z", got.StartTime)
	}
	if got.EndTime != "2026-09-10T22:00:15Z" {
		t.Errorf("end_time = %q, want the latest event 2026-09-10T22:00:15Z", got.EndTime)
	}
	if got.TotalMs < 0 {
		t.Errorf("total_ms = %d, a session cannot take negative time", got.TotalMs)
	}
	if got.TotalMs != 15000 {
		t.Errorf("total_ms = %d, want 15000", got.TotalMs)
	}
}

// Token counts are the sum of what was read, never a zero standing in for what
// could not be read.
func TestTokens_OnlyReadableMetricsAreCounted(t *testing.T) {
	otelFile := filepath.Join(t.TempDir(), "otel.json")
	otelData := `{"metrics": [` + tokenUnknownType + `, ` + tokenNonNumeric + `, ` + tokenNoAttributes + `, ` + tokenOutput + `]}`
	if err := os.WriteFile(otelFile, []byte(otelData), 0644); err != nil {
		t.Fatal(err)
	}

	adapter := ClaudeCodeAdapter{OtelExportFile: otelFile}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Tokens.Value == nil {
		t.Fatalf("tokens value is nil (state %q, reason %q)", profile.Tokens.State, profile.Tokens.Reason)
	}
	want := TokenCounts{Output: 300}
	if *profile.Tokens.Value != want {
		t.Errorf("tokens = %+v, want %+v — only the one readable metric counts", *profile.Tokens.Value, want)
	}
}

// Values, not just states, survive a partial export — the concrete regression
// behind the contract test above.
func TestCapture_ClaudeCode_PartialExport_KeepsToolCallsAndTiming(t *testing.T) {
	otelFile := filepath.Join(t.TempDir(), "otel.json")
	otelData := `{
		"logs": [
			{"event_name": "claude_code.api_request", "timestamp": "2026-09-10T22:00:00Z"},
			{"event_name": "claude_code.tool_decision", "attributes": {"tool_name": "Bash", "decision": "approved"}, "timestamp": "2026-09-10T22:00:05Z"},
			{"event_name": "claude_code.api_request", "timestamp": "2026-09-10T22:00:15Z"}
		]
	}`
	if err := os.WriteFile(otelFile, []byte(otelData), 0644); err != nil {
		t.Fatal(err)
	}

	adapter := ClaudeCodeAdapter{OtelExportFile: otelFile}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}

	if len(profile.ToolCalls.Value) != 1 {
		t.Fatalf("tool_calls count = %d, want 1 (state %q, reason %q)",
			len(profile.ToolCalls.Value), profile.ToolCalls.State, profile.ToolCalls.Reason)
	}
	if profile.ToolCalls.Value[0].Name != "Bash" {
		t.Errorf("tool_calls[0] name = %q, want Bash", profile.ToolCalls.Value[0].Name)
	}
	if profile.Timing.Value == nil {
		t.Fatalf("timing value is nil (state %q, reason %q)", profile.Timing.State, profile.Timing.Reason)
	}
	if profile.Timing.Value.TotalMs != 15000 {
		t.Errorf("timing total_ms = %d, want 15000", profile.Timing.Value.TotalMs)
	}

	// The one signal that genuinely is absent stays honest.
	if profile.Tokens.State != MetricUnknown {
		t.Errorf("tokens state = %q, want %q", profile.Tokens.State, MetricUnknown)
	}
	if profile.Tokens.Value != nil {
		t.Errorf("tokens value = %+v, want nil when no token metric was exported", profile.Tokens.Value)
	}
}
