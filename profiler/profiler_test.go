package profiler

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	os.WriteFile(otelFile, []byte(`{}`), 0644)

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
