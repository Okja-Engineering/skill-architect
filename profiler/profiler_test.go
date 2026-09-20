package profiler

import (
	"encoding/json"
	"fmt"
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

// --- Wire format: upstream-aligned field names ---

func TestTokenCounts_CacheWriteFieldName(t *testing.T) {
	r := PresentTokenResult(TokenCounts{Input: 10, CacheWrite: 5}, "otel")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	json.Unmarshal(data, &raw)
	value, _ := raw["value"].(map[string]any)
	if _, ok := value["cache_write"]; !ok {
		t.Errorf("missing cache_write key: %s", string(data))
	}
	if _, ok := value["cache_creation"]; ok {
		t.Errorf("stale cache_creation key present: %s", string(data))
	}
}

func TestToolCallEntry_ErrorType(t *testing.T) {
	// Failure is the presence of an error type, matching the standard's model.
	entry := ToolCallEntry{Name: "Bash", Timestamp: "2026-09-11T00:00:00Z", Success: false, ErrorType: "timeout"}
	data, _ := json.Marshal(entry)
	var raw map[string]any
	json.Unmarshal(data, &raw)
	if raw["error_type"] != "timeout" {
		t.Errorf("error_type = %v, want timeout", raw["error_type"])
	}
	// Omitted when absent.
	ok := ToolCallEntry{Name: "Bash", Timestamp: "t", Success: true}
	data, _ = json.Marshal(ok)
	var raw2 map[string]any
	json.Unmarshal(data, &raw2)
	if _, has := raw2["error_type"]; has {
		t.Errorf("error_type should be omitted on success: %s", string(data))
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

// --- F03 Slice 1.2: Claude Code session_data (JSONL transcript) adapter ---

func TestCapabilityReport_ClaudeCode_WithSessionData(t *testing.T) {
	tmp := t.TempDir()
	jsonl := filepath.Join(tmp, "session.jsonl")
	data := `{"timestamp":"2026-09-10T22:00:00.000Z","message":{"id":"msg_1","content":[{"type":"tool_use","name":"Bash","id":"toolu_1"}],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":20,"output_tokens_details":{"thinking_tokens":5}}}}
{"timestamp":"2026-09-10T22:00:05.000Z","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_1","is_error":false}]}}
`
	os.WriteFile(jsonl, []byte(data), 0644)

	adapter := ClaudeCodeAdapter{ExportFile: jsonl}
	cap := adapter.Probe()

	if cap.Capabilities[MetricTokens] != SourceSessionData {
		t.Errorf("tokens = %q, want session_data", cap.Capabilities[MetricTokens])
	}
	if cap.Capabilities[MetricToolCalls] != SourceSessionData {
		t.Errorf("tool_calls = %q, want session_data", cap.Capabilities[MetricToolCalls])
	}
	if cap.Capabilities[MetricTiming] != SourceSessionData {
		t.Errorf("timing = %q, want session_data", cap.Capabilities[MetricTiming])
	}
	if cap.Capabilities[MetricSkillActivation] != SourceNone {
		t.Errorf("skill_activation = %q, want none", cap.Capabilities[MetricSkillActivation])
	}
	if cap.Capabilities[MetricAttribution] != SourceNone {
		t.Errorf("attribution = %q, want none", cap.Capabilities[MetricAttribution])
	}
}

func TestCapabilityReport_ClaudeCode_SessionDataEmptyFile(t *testing.T) {
	tmp := t.TempDir()
	jsonl := filepath.Join(tmp, "session.jsonl")
	os.WriteFile(jsonl, []byte(""), 0644)

	adapter := ClaudeCodeAdapter{ExportFile: jsonl}
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

func TestCapture_ClaudeCode_WithSessionData(t *testing.T) {
	tmp := t.TempDir()
	jsonl := filepath.Join(tmp, "session.jsonl")
	data := `{"timestamp":"2026-09-10T22:00:00.000Z","message":{"id":"msg_1","content":[{"type":"tool_use","name":"Bash","id":"toolu_1"}],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":20,"output_tokens_details":{"thinking_tokens":5}}}}
{"timestamp":"2026-09-10T22:00:02.000Z","message":{"id":"msg_1","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":20,"output_tokens_details":{"thinking_tokens":5}}}}
{"timestamp":"2026-09-10T22:00:05.000Z","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_1","is_error":false}]}}
`
	os.WriteFile(jsonl, []byte(data), 0644)

	adapter := ClaudeCodeAdapter{ExportFile: jsonl}
	opts := CaptureOpts{SnapshotHash: "sha123", SkillDir: "/skills/my-skill"}
	profile, err := adapter.Capture("session-001", opts)
	if err != nil {
		t.Fatal(err)
	}

	if profile.Tokens.State != MetricPresent {
		t.Errorf("tokens state = %q, want present", profile.Tokens.State)
	}
	if profile.Tokens.Value == nil {
		t.Fatal("tokens value is nil")
	}
	if profile.Tokens.Value.Input != 100 {
		t.Errorf("tokens input = %d, want 100", profile.Tokens.Value.Input)
	}
	if profile.Tokens.Value.Output != 50 {
		t.Errorf("tokens output = %d, want 50", profile.Tokens.Value.Output)
	}
	if profile.Tokens.Value.CacheRead != 20 {
		t.Errorf("tokens cache_read = %d, want 20", profile.Tokens.Value.CacheRead)
	}
	if profile.Tokens.Value.CacheWrite != 10 {
		t.Errorf("tokens cache_write = %d, want 10", profile.Tokens.Value.CacheWrite)
	}
	if profile.Tokens.Value.Reasoning != 5 {
		t.Errorf("tokens reasoning = %d, want 5", profile.Tokens.Value.Reasoning)
	}
	if profile.Tokens.Source != "session_data" {
		t.Errorf("tokens source = %q, want session_data", profile.Tokens.Source)
	}

	if profile.ToolCalls.State != MetricPresent {
		t.Errorf("tool_calls state = %q, want present", profile.ToolCalls.State)
	}
	if len(profile.ToolCalls.Value) != 1 {
		t.Fatalf("tool_calls count = %d, want 1", len(profile.ToolCalls.Value))
	}
	if profile.ToolCalls.Value[0].Name != "Bash" {
		t.Errorf("tool_calls[0] name = %q, want Bash", profile.ToolCalls.Value[0].Name)
	}
	if !profile.ToolCalls.Value[0].Success {
		t.Errorf("tool_calls[0] success = %v, want true", profile.ToolCalls.Value[0].Success)
	}
	if profile.ToolCalls.Source != "session_data" {
		t.Errorf("tool_calls source = %q, want session_data", profile.ToolCalls.Source)
	}

	if profile.Timing.State != MetricPresent {
		t.Errorf("timing state = %q, want present", profile.Timing.State)
	}
	if profile.Timing.Value == nil {
		t.Fatal("timing value is nil")
	}
	if profile.Timing.Value.TotalMs != 5000 {
		t.Errorf("timing total_ms = %d, want 5000", profile.Timing.Value.TotalMs)
	}
	if profile.Timing.Source != "session_data" {
		t.Errorf("timing source = %q, want session_data", profile.Timing.Source)
	}
}

func TestCapture_ClaudeCode_SessionDataToolError(t *testing.T) {
	tmp := t.TempDir()
	jsonl := filepath.Join(tmp, "session.jsonl")
	data := `{"timestamp":"2026-09-10T22:00:00.000Z","message":{"id":"msg_1","content":[{"type":"tool_use","name":"Bash","id":"toolu_1"}],"usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens_details":{"thinking_tokens":0}}}}
{"timestamp":"2026-09-10T22:00:01.000Z","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_1","is_error":true}]}}
`
	os.WriteFile(jsonl, []byte(data), 0644)

	adapter := ClaudeCodeAdapter{ExportFile: jsonl}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "sha123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}

	if profile.ToolCalls.State != MetricPresent {
		t.Fatalf("tool_calls state = %q, want present", profile.ToolCalls.State)
	}
	if profile.ToolCalls.Value[0].Success {
		t.Errorf("tool_calls[0] success = %v, want false for is_error=true", profile.ToolCalls.Value[0].Success)
	}
}

// --- Cursor adapter tests ---

func TestCapabilityReport_Cursor_WithOtel(t *testing.T) {
	tmp := t.TempDir()
	otelFile := filepath.Join(tmp, "otel.json")
	otelData := `{
		"metrics": [
			{"name": "cursor.token.usage", "attributes": {"token_type": "input"}, "value": 100},
			{"name": "cursor.tool.calls", "attributes": {"cursor.tool.name": "Bash", "cursor.tool.status": "success"}, "value": 3}
		],
		"logs": [
			{"event_name": "cursor.api.request", "timestamp": "2026-09-10T22:00:00Z"},
			{"event_name": "cursor.skill.activated", "attributes": {"skill_name": "audit"}, "timestamp": "2026-09-10T22:00:06Z"}
		]
	}`
	os.WriteFile(otelFile, []byte(otelData), 0644)

	adapter := CursorAdapter{OtelExportFile: otelFile}
	cap := adapter.Probe()

	if cap.Harness != "cursor" {
		t.Errorf("harness = %q, want cursor", cap.Harness)
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
	if cap.Capabilities[MetricSkillActivation] != SourceOtel {
		t.Errorf("skill_activation = %q, want otel", cap.Capabilities[MetricSkillActivation])
	}
	if cap.Capabilities[MetricAttribution] != SourceNone {
		t.Errorf("attribution = %q, want none", cap.Capabilities[MetricAttribution])
	}
}

func TestCapture_Cursor_WithOtelData(t *testing.T) {
	tmp := t.TempDir()
	otelFile := filepath.Join(tmp, "otel.json")
	otelData := `{
		"metrics": [
			{"name": "cursor.token.usage", "attributes": {"token_type": "input"}, "value": 1000},
			{"name": "cursor.token.usage", "attributes": {"token_type": "output"}, "value": 500},
			{"name": "cursor.token.usage", "attributes": {"token_type": "reasoning"}, "value": 150},
			{"name": "cursor.tool.calls", "attributes": {"cursor.tool.name": "Bash", "cursor.tool.status": "success"}, "value": 4},
			{"name": "cursor.tool.calls", "attributes": {"cursor.tool.name": "Read", "cursor.tool.status": "failure"}, "value": 1}
		],
		"logs": [
			{"event_name": "cursor.api.request", "timestamp": "2026-09-10T22:00:00Z"},
			{"event_name": "cursor.skill.activated", "attributes": {"skill_name": "audit", "trigger": "keyword"}, "timestamp": "2026-09-10T22:00:06Z"},
			{"event_name": "cursor.api.request", "timestamp": "2026-09-10T22:00:10Z"}
		]
	}`
	os.WriteFile(otelFile, []byte(otelData), 0644)

	adapter := CursorAdapter{OtelExportFile: otelFile}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "sha123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}

	if profile.Tokens.State != MetricPresent {
		t.Fatalf("tokens state = %q, want present", profile.Tokens.State)
	}
	if profile.Tokens.Value.Input != 1000 {
		t.Errorf("tokens input = %d, want 1000", profile.Tokens.Value.Input)
	}
	if profile.Tokens.Value.Output != 500 {
		t.Errorf("tokens output = %d, want 500", profile.Tokens.Value.Output)
	}
	if profile.Tokens.Value.Reasoning != 150 {
		t.Errorf("tokens reasoning = %d, want 150", profile.Tokens.Value.Reasoning)
	}
	if profile.Tokens.Source != "otel" {
		t.Errorf("tokens source = %q, want otel", profile.Tokens.Source)
	}
	if profile.ToolCalls.State != MetricPresent {
		t.Fatalf("tool_calls state = %q, want present", profile.ToolCalls.State)
	}
	if len(profile.ToolCalls.Value) != 2 {
		t.Fatalf("tool_calls count = %d, want 2", len(profile.ToolCalls.Value))
	}
	if profile.ToolCalls.Value[0].Name != "Bash" || !profile.ToolCalls.Value[0].Success {
		t.Errorf("tool_calls[0] = %+v, want Bash/success", profile.ToolCalls.Value[0])
	}
	if profile.ToolCalls.Value[0].Count != 4 {
		t.Errorf("tool_calls[0] count = %d, want 4", profile.ToolCalls.Value[0].Count)
	}
	if profile.ToolCalls.Value[1].Success || profile.ToolCalls.Value[1].ErrorType != "failure" {
		t.Errorf("tool_calls[1] = %+v, want Read/failure with error_type", profile.ToolCalls.Value[1])
	}
	if profile.Timing.State != MetricPresent {
		t.Fatalf("timing state = %q, want present", profile.Timing.State)
	}
	if profile.SkillActivation.State != MetricPresent {
		t.Fatalf("skill_activation state = %q, want present", profile.SkillActivation.State)
	}
	if len(profile.SkillActivation.Value) != 1 {
		t.Fatalf("skill_activation count = %d, want 1", len(profile.SkillActivation.Value))
	}
	if profile.SkillActivation.Value[0].SkillName != "audit" {
		t.Errorf("skill_activation skill = %q, want audit", profile.SkillActivation.Value[0].SkillName)
	}
	if profile.Attribution.State != MetricUnknown {
		t.Errorf("attribution state = %q, want unknown", profile.Attribution.State)
	}
}

func TestCapture_Cursor_SqliteUnknownSchema(t *testing.T) {
	tmp := t.TempDir()
	sqliteFile := filepath.Join(tmp, "state.vscdb")
	os.WriteFile(sqliteFile, []byte("fake sqlite bytes"), 0644)

	adapter := CursorAdapter{ExportFile: sqliteFile}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "sha123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}

	if profile.Tokens.State != MetricUnknown {
		t.Errorf("tokens state = %q, want unknown", profile.Tokens.State)
	}
	if profile.ToolCalls.State != MetricUnknown {
		t.Errorf("tool_calls state = %q, want unknown", profile.ToolCalls.State)
	}
	if profile.Timing.State != MetricUnknown {
		t.Errorf("timing state = %q, want unknown", profile.Timing.State)
	}
	if profile.SkillActivation.State != MetricUnknown {
		t.Errorf("skill_activation state = %q, want unknown", profile.SkillActivation.State)
	}
}

// --- Codex adapter tests ---

func TestCapabilityReport_Codex_WithOtel(t *testing.T) {
	tmp := t.TempDir()
	otelFile := filepath.Join(tmp, "otel.json")
	otelData := `{
		"logs": [
			{"event_name": "codex.sse_event", "attributes": {"input_token_count": 10}, "timestamp": "2026-09-10T22:00:00Z"},
			{"event_name": "codex.tool_decision", "attributes": {"tool_name": "Bash", "decision": "approved"}, "timestamp": "2026-09-10T22:00:05Z"},
			{"event_name": "codex.api.request", "timestamp": "2026-09-10T22:00:10Z"}
		]
	}`
	os.WriteFile(otelFile, []byte(otelData), 0644)

	adapter := CodexAdapter{OtelExportFile: otelFile}
	cap := adapter.Probe()

	if cap.Harness != "codex" {
		t.Errorf("harness = %q, want codex", cap.Harness)
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
}

func TestCapture_Codex_WithOtelData(t *testing.T) {
	tmp := t.TempDir()
	otelFile := filepath.Join(tmp, "otel.json")
	otelData := `{
		"logs": [
			{"event_name": "codex.sse_event", "attributes": {"input_token_count": 1000, "output_token_count": 500, "reasoning_tokens": 50}, "timestamp": "2026-09-10T22:00:00Z"},
			{"event_name": "codex.tool_decision", "attributes": {"tool_name": "Bash", "decision": "approved"}, "timestamp": "2026-09-10T22:00:05Z"},
			{"event_name": "codex.sse_event", "attributes": {"input_token_count": 10}, "timestamp": "2026-09-10T22:00:07Z"},
			{"event_name": "codex.api.request", "timestamp": "2026-09-10T22:00:00Z"},
			{"event_name": "codex.api.request", "timestamp": "2026-09-10T22:00:10Z"}
		]
	}`
	os.WriteFile(otelFile, []byte(otelData), 0644)

	adapter := CodexAdapter{OtelExportFile: otelFile}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "sha123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}

	if profile.Tokens.State != MetricPresent {
		t.Fatalf("tokens state = %q, want present", profile.Tokens.State)
	}
	if profile.Tokens.Value.Input != 1010 {
		t.Errorf("tokens input = %d, want 1010", profile.Tokens.Value.Input)
	}
	if profile.Tokens.Value.Output != 500 {
		t.Errorf("tokens output = %d, want 500", profile.Tokens.Value.Output)
	}
	if profile.Tokens.Value.Reasoning != 50 {
		t.Errorf("tokens reasoning = %d, want 50", profile.Tokens.Value.Reasoning)
	}
	if profile.Tokens.Source != "otel" {
		t.Errorf("tokens source = %q, want otel", profile.Tokens.Source)
	}
	if profile.ToolCalls.State != MetricPresent {
		t.Fatalf("tool_calls state = %q, want present", profile.ToolCalls.State)
	}
	if len(profile.ToolCalls.Value) != 1 {
		t.Fatalf("tool_calls count = %d, want 1", len(profile.ToolCalls.Value))
	}
	if !profile.ToolCalls.Value[0].Success {
		t.Errorf("tool_calls[0] success = %v, want true", profile.ToolCalls.Value[0].Success)
	}
	if profile.Timing.State != MetricPresent {
		t.Fatalf("timing state = %q, want present", profile.Timing.State)
	}
	if profile.SkillActivation.State != MetricUnknown {
		t.Errorf("skill_activation state = %q, want unknown", profile.SkillActivation.State)
	}
}

// --- Devin adapter tests ---

func TestCapabilityReport_Devin_WithExport(t *testing.T) {
	tmp := t.TempDir()
	exportFile := filepath.Join(tmp, "atif.json")
	exportData := `{"steps": [{"type": "message"}], "messages": []}`
	os.WriteFile(exportFile, []byte(exportData), 0644)

	adapter := DevinAdapter{ExportFile: exportFile}
	cap := adapter.Probe()

	if cap.Harness != "devin" {
		t.Errorf("harness = %q, want devin", cap.Harness)
	}
	if cap.Capabilities[MetricTokens] != SourceSessionData {
		t.Errorf("tokens = %q, want session_data", cap.Capabilities[MetricTokens])
	}
	if cap.Capabilities[MetricToolCalls] != SourceSessionData {
		t.Errorf("tool_calls = %q, want session_data", cap.Capabilities[MetricToolCalls])
	}
	if cap.Capabilities[MetricTiming] != SourceSessionData {
		t.Errorf("timing = %q, want session_data", cap.Capabilities[MetricTiming])
	}
}

func TestCapture_Devin_WithExport(t *testing.T) {
	tmp := t.TempDir()
	exportFile := filepath.Join(tmp, "atif.json")
	exportData := `{"steps": [{"type": "message"}], "messages": []}`
	os.WriteFile(exportFile, []byte(exportData), 0644)

	adapter := DevinAdapter{ExportFile: exportFile}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "sha123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}

	if profile.Tokens.State != MetricUnknown {
		t.Errorf("tokens state = %q, want unknown", profile.Tokens.State)
	}
	if profile.ToolCalls.State != MetricUnknown {
		t.Errorf("tool_calls state = %q, want unknown", profile.ToolCalls.State)
	}
	if profile.Timing.State != MetricUnknown {
		t.Errorf("timing state = %q, want unknown", profile.Timing.State)
	}
	if profile.SkillActivation.State != MetricUnknown {
		t.Errorf("skill_activation state = %q, want unknown", profile.SkillActivation.State)
	}
	if profile.Attribution.State != MetricUnknown {
		t.Errorf("attribution state = %q, want unknown", profile.Attribution.State)
	}
}

// --- F04 comparison tests ---

func TestCompareProfiles_TokensAndToolCalls(t *testing.T) {
	tmp := t.TempDir()
	baseFile := filepath.Join(tmp, "base.json")
	candFile := filepath.Join(tmp, "cand.json")

	baseline := Profile{
		Schema:       ProfileSchema,
		Harness:      "claude_code",
		SessionID:    "base-001",
		SnapshotHash: "sha-base",
		SkillDir:     "skills/skill-audit",
		Tokens:       PresentTokenResult(TokenCounts{Input: 100, Output: 50, CacheRead: 10}, "otel"),
		ToolCalls:    PresentToolCallResult([]ToolCallEntry{{Name: "Bash", Success: true}}, "otel"),
		Timing:       PresentTimingResult(TimingData{TotalMs: 1000}, "otel"),
	}
	candidate := Profile{
		Schema:       ProfileSchema,
		Harness:      "claude_code",
		SessionID:    "cand-001",
		SnapshotHash: "sha-base",
		SkillDir:     "skills/skill-audit",
		Tokens:       PresentTokenResult(TokenCounts{Input: 120, Output: 60, CacheRead: 5}, "otel"),
		ToolCalls:    PresentToolCallResult([]ToolCallEntry{{Name: "Bash", Success: true}, {Name: "Read", Success: false}}, "otel"),
		Timing:       PresentTimingResult(TimingData{TotalMs: 1500}, "otel"),
	}

	baseJSON, _ := json.Marshal(baseline)
	candJSON, _ := json.Marshal(candidate)
	os.WriteFile(baseFile, baseJSON, 0644)
	os.WriteFile(candFile, candJSON, 0644)

	base, err := LoadProfile(baseFile)
	if err != nil {
		t.Fatal(err)
	}
	cand, err := LoadProfile(candFile)
	if err != nil {
		t.Fatal(err)
	}

	report := CompareProfiles(base, cand)
	if report.Schema != ComparisonSchema {
		t.Errorf("schema = %q, want %q", report.Schema, ComparisonSchema)
	}
	if !report.Comparable {
		t.Fatal("report should be comparable")
	}
	if report.Metrics["tokens"].Comparable != true {
		t.Fatalf("tokens not comparable")
	}
	delta := report.Metrics["tokens"].Delta.(TokenCounts)
	if delta.Input != 20 {
		t.Errorf("token input delta = %d, want 20", delta.Input)
	}
	if delta.Output != 10 {
		t.Errorf("token output delta = %d, want 10", delta.Output)
	}

	toolDelta := report.Metrics["tool_calls"].Delta.(map[string]any)
	if toolDelta["count"].(int) != 1 {
		t.Errorf("tool call count delta = %v, want 1", toolDelta["count"])
	}

	timingDelta := report.Metrics["timing"].Delta.(map[string]int64)
	if timingDelta["total_ms"] != 500 {
		t.Errorf("timing total_ms delta = %d, want 500", timingDelta["total_ms"])
	}
}

func TestCompareProfiles_HarnessMismatch(t *testing.T) {
	baseline := Profile{
		Schema:    ProfileSchema,
		Harness:   "claude_code",
		SessionID: "base-001",
		Tokens:    PresentTokenResult(TokenCounts{Input: 100}, "otel"),
	}
	candidate := Profile{
		Schema:    ProfileSchema,
		Harness:   "cursor",
		SessionID: "cand-001",
		Tokens:    PresentTokenResult(TokenCounts{Input: 120}, "otel"),
	}

	report := CompareProfiles(baseline, candidate)
	if len(report.Notes) == 0 {
		t.Error("expected harness mismatch note")
	}
	if !report.Comparable {
		t.Error("metrics should still be comparable despite harness mismatch")
	}
}

func TestCompareProfiles_UnknownMetric(t *testing.T) {
	baseline := Profile{
		Schema:    ProfileSchema,
		Harness:   "claude_code",
		SessionID: "base-001",
		Tokens:    UnknownTokenResult("no data"),
	}
	candidate := Profile{
		Schema:    ProfileSchema,
		Harness:   "claude_code",
		SessionID: "cand-001",
		Tokens:    PresentTokenResult(TokenCounts{Input: 120}, "otel"),
	}

	report := CompareProfiles(baseline, candidate)
	if report.Metrics["tokens"].Comparable {
		t.Error("expected tokens to not be comparable when one is unknown")
	}
	if report.Comparable {
		t.Error("report should not be comparable when no metric is present in both")
	}
}

// --- F04 experiment plan tests ---

func TestNormalizeDesign_AppliesDefaults(t *testing.T) {
	d := ExperimentDesign{
		Schema:       ExperimentSchema,
		Name:         "skill-rewrite-efficiency",
		TaskFamilies: []string{"refactor-audit", "refactor-rewrite"},
		Repetitions:  3,
		Baseline: Condition{
			Name:         "no-skill",
			Command:      "./capture-baseline $TASK $REP $PROFILE",
			SnapshotHash: "sha-base",
			SkillDir:     "skills/skill-audit",
		},
		Candidate: Condition{
			Name:         "with-skill",
			Command:      "./capture-candidate $TASK $REP $PROFILE",
			SnapshotHash: "sha-cand",
			SkillDir:     "skills/skill-audit",
		},
	}
	n, err := NormalizeDesign(d)
	if err != nil {
		t.Fatal(err)
	}
	if n.Ordering != "blocked" {
		t.Errorf("ordering = %q, want blocked", n.Ordering)
	}
	if n.AnalysisMethod != "difference" {
		t.Errorf("analysis_method = %q, want difference", n.AnalysisMethod)
	}
	if n.StoppingRule != "fixed" {
		t.Errorf("stopping_rule = %q, want fixed", n.StoppingRule)
	}
	if n.Baseline.Harness != "claude_code" {
		t.Errorf("baseline harness = %q, want claude_code", n.Baseline.Harness)
	}
	if n.Candidate.Harness != "claude_code" {
		t.Errorf("candidate harness = %q, want claude_code", n.Candidate.Harness)
	}
}

func TestGeneratePlan(t *testing.T) {
	d := ExperimentDesign{
		Schema:       ExperimentSchema,
		Name:         "skill-rewrite-efficiency",
		TaskFamilies: []string{"refactor-audit", "refactor-rewrite"},
		Repetitions:  2,
		Ordering:     "blocked",
		Baseline: Condition{
			Name:         "no-skill",
			Command:      "./capture-baseline $TASK $REP $PROFILE",
			SnapshotHash: "sha-base",
			SkillDir:     "skills/skill-audit",
			Harness:      "claude_code",
		},
		Candidate: Condition{
			Name:         "with-skill",
			Command:      "./capture-candidate $TASK $REP $PROFILE",
			SnapshotHash: "sha-cand",
			SkillDir:     "skills/skill-audit",
			Harness:      "claude_code",
		},
	}
	plan, err := GeneratePlan(d)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Schema != ExperimentPlanSchema {
		t.Errorf("schema = %q, want %q", plan.Schema, ExperimentPlanSchema)
	}
	wantRuns := len(d.TaskFamilies) * d.Repetitions
	if len(plan.Runs) != wantRuns {
		t.Fatalf("runs = %d, want %d", len(plan.Runs), wantRuns)
	}
	for i, run := range plan.Runs {
		if len(run.Steps) != 2 {
			t.Errorf("run %d steps = %d, want 2", i, len(run.Steps))
		}
		if run.Steps[0].Condition != "baseline" {
			t.Errorf("run %d first step condition = %q, want baseline", i, run.Steps[0].Condition)
		}
		if run.Steps[1].Condition != "candidate" {
			t.Errorf("run %d second step condition = %q, want candidate", i, run.Steps[1].Condition)
		}
		if !strings.Contains(run.Steps[0].Command, "capture-baseline") {
			t.Errorf("run %d baseline command = %q, want capture-baseline", i, run.Steps[0].Command)
		}
		if !strings.HasSuffix(run.Comparison, "-comparison.json") {
			t.Errorf("run %d comparison output = %q, want -comparison.json suffix", i, run.Comparison)
		}
	}
}

func TestLoadExperimentDesign(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "design.json")
	d := ExperimentDesign{
		Schema:       ExperimentSchema,
		Name:         "skill-rewrite-efficiency",
		TaskFamilies: []string{"refactor-audit"},
		Baseline: Condition{
			Command:      "cmd",
			SnapshotHash: "sha-base",
			SkillDir:     "skills/skill-audit",
		},
		Candidate: Condition{
			Command:      "cmd",
			SnapshotHash: "sha-cand",
			SkillDir:     "skills/skill-audit",
		},
	}
	data, _ := json.Marshal(d)
	os.WriteFile(file, data, 0644)

	loaded, err := LoadExperimentDesign(file)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Name != d.Name {
		t.Errorf("name = %q, want %q", loaded.Name, d.Name)
	}
}

// --- F04 experiment run tests ---

func TestRunPlan(t *testing.T) {
	tmp := t.TempDir()
	baseFixture := filepath.Join(tmp, "base-fixture.json")
	candFixture := filepath.Join(tmp, "cand-fixture.json")

	baseline := Profile{
		Schema:       ProfileSchema,
		Harness:      "claude_code",
		SessionID:    "base-001",
		SnapshotHash: "sha-base",
		SkillDir:     "skills/skill-audit",
		Tokens:       PresentTokenResult(TokenCounts{Input: 100, Output: 50}, "otel"),
	}
	candidate := Profile{
		Schema:       ProfileSchema,
		Harness:      "claude_code",
		SessionID:    "cand-001",
		SnapshotHash: "sha-cand",
		SkillDir:     "skills/skill-audit",
		Tokens:       PresentTokenResult(TokenCounts{Input: 120, Output: 60}, "otel"),
	}
	bJSON, _ := json.Marshal(baseline)
	cJSON, _ := json.Marshal(candidate)
	os.WriteFile(baseFixture, bJSON, 0644)
	os.WriteFile(candFixture, cJSON, 0644)

	design := ExperimentDesign{
		Schema:       ExperimentSchema,
		Name:         "run-test",
		TaskFamilies: []string{"audit"},
		Repetitions:  1,
		OutputDir:    tmp,
		Baseline: Condition{
			Name:         "no-skill",
			Command:      fmt.Sprintf("cp %s $PROFILE", baseFixture),
			SnapshotHash: "sha-base",
			SkillDir:     "skills/skill-audit",
			Harness:      "claude_code",
		},
		Candidate: Condition{
			Name:         "with-skill",
			Command:      fmt.Sprintf("cp %s $PROFILE", candFixture),
			SnapshotHash: "sha-cand",
			SkillDir:     "skills/skill-audit",
			Harness:      "claude_code",
		},
	}
	plan, err := GeneratePlan(design)
	if err != nil {
		t.Fatal(err)
	}

	reports, err := RunPlan(plan, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 {
		t.Fatalf("reports = %d, want 1", len(reports))
	}
	if !reports[0].Comparable {
		t.Fatal("report should be comparable")
	}
	delta := reports[0].Metrics["tokens"].Delta.(TokenCounts)
	if delta.Input != 20 {
		t.Errorf("token input delta = %d, want 20", delta.Input)
	}

	outFile := filepath.Join(tmp, "audit-r1-comparison.json")
	if _, err := os.Stat(outFile); err != nil {
		t.Fatalf("comparison file not written: %v", err)
	}
}

func TestLoadPlan(t *testing.T) {
	tmp := t.TempDir()
	d := ExperimentDesign{
		Schema:       ExperimentSchema,
		Name:         "load-plan",
		TaskFamilies: []string{"audit"},
		Baseline: Condition{
			Command:      "cmd",
			SnapshotHash: "sha-base",
			SkillDir:     "skills/skill-audit",
		},
		Candidate: Condition{
			Command:      "cmd",
			SnapshotHash: "sha-cand",
			SkillDir:     "skills/skill-audit",
		},
	}
	plan, err := GeneratePlan(d)
	if err != nil {
		t.Fatal(err)
	}
	planFile := filepath.Join(tmp, "plan.json")
	planJSON, _ := json.Marshal(plan)
	os.WriteFile(planFile, planJSON, 0644)

	loaded, err := LoadPlan(planFile)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Schema != ExperimentPlanSchema {
		t.Errorf("schema = %q, want %q", loaded.Schema, ExperimentPlanSchema)
	}
	if len(loaded.Runs) != 1 {
		t.Errorf("runs = %d, want 1", len(loaded.Runs))
	}
}
