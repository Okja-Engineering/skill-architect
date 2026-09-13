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

// fixture resolves an OTLP/JSON export fixture. The files are real OTLP — one
// Export*ServiceRequest per JSON object — so these tests exercise the format
// Claude Code emits rather than one the adapter invented. Each file's purpose,
// and which of its fields were observed on the wire, is in
// testdata/otlp/README.md.
func fixture(name string) string { return filepath.Join("testdata", "otlp", name) }

func TestCapabilityReport_ClaudeCode_WithOtel(t *testing.T) {
	adapter := ClaudeCodeAdapter{OtelExportFile: fixture("full_export.ndjson")}
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
	// Well-formed JSON carrying no OTLP envelope: parsed fine, carries no
	// telemetry, so every signal is "none".
	adapter := ClaudeCodeAdapter{OtelExportFile: fixture("no_envelope.json")}
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
	adapter := ClaudeCodeAdapter{OtelExportFile: fixture("full_export.ndjson")}
	opts := CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"}
	profile, err := adapter.Capture("session-001", opts)
	if err != nil {
		t.Fatal(err)
	}

	// Tokens should be present.
	if profile.Tokens.State != MetricPresent {
		t.Errorf("tokens state = %q (reason %q), want present", profile.Tokens.State, profile.Tokens.Reason)
	}
	if profile.Tokens.Value == nil {
		t.Fatal("tokens value is nil")
	}
	want := TokenCounts{Input: 1523, Output: 412, CacheRead: 20480, CacheCreation: 3072}
	if *profile.Tokens.Value != want {
		t.Errorf("tokens = %+v, want %+v", *profile.Tokens.Value, want)
	}
	// Claude Code has no reasoning token type, so the field stays unpopulated
	// and its key is absent from the profile.
	if profile.Tokens.Value.Reasoning != 0 {
		t.Errorf("tokens reasoning = %d, want 0 — Claude Code emits no reasoning token type",
			profile.Tokens.Value.Reasoning)
	}
	if profile.Tokens.Source != "otel" {
		t.Errorf("tokens source = %q, want otel", profile.Tokens.Source)
	}

	// Tool calls should be present: the rejected decision and the completed
	// result, in timestamp order.
	if profile.ToolCalls.State != MetricPresent {
		t.Errorf("tool_calls state = %q (reason %q), want present", profile.ToolCalls.State, profile.ToolCalls.Reason)
	}
	wantCalls := []ToolCallEntry{
		{Name: "Bash", Timestamp: "2026-09-13T20:49:55.3Z", Success: false},
		{Name: "Read", Timestamp: "2026-09-13T20:49:55.46Z", Success: true},
	}
	assertToolCalls(t, profile.ToolCalls.Value, wantCalls)

	// Timing should be present.
	if profile.Timing.State != MetricPresent {
		t.Errorf("timing state = %q (reason %q), want present", profile.Timing.State, profile.Timing.Reason)
	}
	if profile.Timing.Value == nil {
		t.Fatal("timing value is nil")
	}
	if profile.Timing.Value.StartTime != "2026-09-13T20:49:55.1Z" {
		t.Errorf("timing start = %q, want 2026-09-13T20:49:55.1Z", profile.Timing.Value.StartTime)
	}
	if profile.Timing.Value.EndTime != "2026-09-13T20:49:56.272Z" {
		t.Errorf("timing end = %q, want 2026-09-13T20:49:56.272Z", profile.Timing.Value.EndTime)
	}
	if profile.Timing.Value.TotalMs != 1172 {
		t.Errorf("timing total_ms = %d, want 1172", profile.Timing.Value.TotalMs)
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

// assertToolCalls compares the captured entries with what the export carried,
// in order: the list is the session's tool calls, and both its contents and its
// order are part of the profile.
func assertToolCalls(t *testing.T, got, want []ToolCallEntry) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("tool_calls = %+v, want %d entries: %+v", got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tool_calls[%d] = %+v, want %+v", i, got[i], want[i])
		}
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

// A supplied CaptureOpts.ExportFile is refused by the adapter that cannot read
// it, not only by the CLI. The adapter owns its input contract: a library caller
// who passes a session export must be told it is not read, rather than handed a
// profile that looks like missing telemetry.
func TestCapture_ClaudeCode_RefusesAnExportFileItCannotRead(t *testing.T) {
	adapter := ClaudeCodeAdapter{OtelExportFile: fixture("full_export.ndjson")}
	_, err := adapter.Capture("session-001", CaptureOpts{
		ExportFile:   "/var/tmp/session.atif.json",
		SnapshotHash: "abc123",
		SkillDir:     "/skills/my-skill",
	})
	if err == nil {
		t.Fatal("--export-file accepted by an adapter that cannot read it: the path would be silently ignored")
	}
	if !strings.Contains(err.Error(), "--export-file") || !strings.Contains(err.Error(), "--otel-file") {
		t.Errorf("error = %q, want it to name the flag that was refused and the one that works", err)
	}
	if !strings.Contains(err.Error(), "claude_code") {
		t.Errorf("error = %q, want it to name the adapter that cannot honour the input", err)
	}
}

// --- Acceptance criterion 6: Profile JSON round-trips ---

func TestProfileRoundTrip(t *testing.T) {
	adapter := ClaudeCodeAdapter{OtelExportFile: fixture("full_export.ndjson")}
	opts := CaptureOpts{SnapshotHash: "sha123", SkillDir: "/skills/test"}
	profile, err := adapter.Capture("sess-1", opts)
	if err != nil {
		t.Fatal(err)
	}

	// A profile of nothing round-trips trivially, so the round trip must be
	// exercised on one that carries values.
	if profile.Tokens.State != MetricPresent || profile.ToolCalls.State != MetricPresent || profile.Timing.State != MetricPresent {
		t.Fatalf("round trip is vacuous: tokens %q, tool_calls %q, timing %q — want all present",
			profile.Tokens.State, profile.ToolCalls.State, profile.Timing.State)
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

func TestCaptureDeliversEverySignalProbeAdvertises(t *testing.T) {
	cases := []struct {
		name string
		// fixture is the OTLP export the adapter is pointed at, unless one of
		// the two flags below says otherwise.
		fixture      string
		unconfigured bool         // adapter has no export file at all
		missingFile  bool         // adapter points at a path that does not exist
		present      []MetricName // signals that must be "present"
		absent       MetricState  // state required of tokens/tool_calls/timing when not present
	}{
		{name: "no export file configured", unconfigured: true, absent: MetricUnknown},
		{name: "export file path set but file missing", missingFile: true, absent: MetricError},

		// Exports that carry values.
		{name: "every signal", fixture: "full_export.ndjson",
			present: []MetricName{MetricTokens, MetricToolCalls, MetricTiming}, absent: MetricUnknown},
		{name: "tokens only", fixture: "tokens_only.json",
			present: []MetricName{MetricTokens}, absent: MetricUnknown},
		{name: "tool calls only", fixture: "tool_calls_only.json",
			present: []MetricName{MetricToolCalls}, absent: MetricUnknown},
		{name: "timing only", fixture: "timing_only.json",
			present: []MetricName{MetricTiming}, absent: MetricUnknown},
		{name: "tool calls and timing, no token metric", fixture: "partial_no_tokens.json",
			present: []MetricName{MetricToolCalls, MetricTiming}, absent: MetricUnknown},
		{name: "delta sums across batches", fixture: "multi_batch_delta.ndjson",
			present: []MetricName{MetricTokens}, absent: MetricUnknown},
		{name: "cumulative takes the last value per series", fixture: "cumulative.ndjson",
			present: []MetricName{MetricTokens}, absent: MetricUnknown},
		{name: "numbers and strings both decode", fixture: "number_string_variants.json",
			present: []MetricName{MetricTokens, MetricToolCalls, MetricTiming}, absent: MetricUnknown},
		{name: "readable points alongside an unreadable one", fixture: "as_double_rounding.json",
			present: []MetricName{MetricTokens}, absent: MetricUnknown},
		{name: "an accept and its result are one call", fixture: "accept_then_result.json",
			present: []MetricName{MetricToolCalls}, absent: MetricUnknown},
		{name: "a failed tool call is still a call", fixture: "tool_failure.json",
			present: []MetricName{MetricToolCalls}, absent: MetricUnknown},
		{name: "a tool call with no timestamp is still a call", fixture: "no_timestamp_tool_call.json",
			present: []MetricName{MetricToolCalls}, absent: MetricUnknown},
		{name: "skill.name is carried but not read", fixture: "skill_name_present.json",
			present: []MetricName{MetricTokens, MetricTiming}, absent: MetricUnknown},
		{name: "api_request records out of order", fixture: "out_of_order.ndjson",
			present: []MetricName{MetricTiming}, absent: MetricUnknown},

		// Exports that parse but carry nothing readable for any signal.
		{name: "token.usage arrives as a gauge", fixture: "gauge_not_sum.json", absent: MetricUnknown},
		{name: "temporality reads as neither delta nor cumulative", fixture: "unreadable_temporality.json", absent: MetricUnknown},
		{name: "tool events with nothing readable", fixture: "unreadable_tool_events.json", absent: MetricUnknown},
		{name: "accepts with no results yet", fixture: "accepts_no_results.json", absent: MetricUnknown},
		{name: "a reject with no tool name", fixture: "unnamed_reject.json", absent: MetricUnknown},
		{name: "api_request with no timeUnixNano", fixture: "untimed_api_request.json", absent: MetricUnknown},
		{name: "only events the adapter does not read", fixture: "unknown_events.json", absent: MetricUnknown},
		{name: "no OTLP envelope", fixture: "no_envelope.json", absent: MetricUnknown},
		{name: "the envelope the adapter used to invent", fixture: "bespoke_envelope.json", absent: MetricUnknown},

		// Exports that cannot be read as OTLP/JSON at all.
		{name: "malformed JSON", fixture: "malformed.json", absent: MetricError},
		{name: "a partial final line", fixture: "truncated_final_line.ndjson", absent: MetricError},
		{name: "a value that does not fit the schema", fixture: "type_mismatch.json", absent: MetricError},
		{name: "an array of exports", fixture: "top_level_array.json", absent: MetricError},
		{name: "an empty file", fixture: "empty.json", absent: MetricError},
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
				adapter = ClaudeCodeAdapter{OtelExportFile: fixture(tc.fixture)}
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

// Every reason is true of the input that produced it. A reason that names a
// cause the export does not have sends the reader to the wrong place, and one
// that leaks the adapter's Go types tells them nothing they can act on.
func TestCapture_ReasonNamesWhatTheExportActuallyCarried(t *testing.T) {
	cases := []struct {
		fixture string
		metric  MetricName
		want    string   // the whole reason, when the wording is the contract
		wantIn  []string // substrings the reason must carry
		wantOut []string // substrings it must not
	}{
		// File-level: the file could not be read as an OTLP/JSON export.
		{fixture: "empty.json", metric: MetricTokens,
			want: "OTel export file is empty"},
		{fixture: "top_level_array.json", metric: MetricToolCalls,
			wantIn:  []string{"top-level JSON value is an array", "ExportMetricsServiceRequest"},
			wantOut: []string{"profiler.", "Go struct"}},
		{fixture: "malformed.json", metric: MetricTiming,
			wantIn:  []string{"malformed JSON at byte", "in batch 1"},
			wantOut: []string{"delete the final partial line", "profiler."}},
		{fixture: "truncated_final_line.ndjson", metric: MetricTokens,
			wantIn: []string{"malformed JSON at byte", "in batch 4",
				"No data from earlier batches was used", "delete the final partial line and retry"}},
		{fixture: "type_mismatch.json", metric: MetricTokens,
			wantIn:  []string{"OTel export does not fit the OTLP schema", "resourceMetrics", "in batch 1"},
			wantOut: []string{"profiler.", "otlpBatch", "Go struct"}},

		// File-level: parsed fine, carried no telemetry.
		{fixture: "no_envelope.json", metric: MetricTokens,
			wantIn:  []string{"no resourceMetrics or resourceLogs found", "OTEL_EXPORTER_OTLP_PROTOCOL=http/json"},
			wantOut: []string{"no claude_code.token.usage metric found"}},
		{fixture: "bespoke_envelope.json", metric: MetricTokens,
			wantIn:  []string{"OTel export is not OTLP/JSON", "no resourceMetrics or resourceLogs found"},
			wantOut: []string{"no claude_code.token.usage metric found"}},

		// Per signal: the counters the walk kept.
		{fixture: "unknown_events.json", metric: MetricTokens,
			want: "no claude_code.token.usage metric found in OTel export"},
		{fixture: "unknown_events.json", metric: MetricToolCalls,
			want: "no claude_code.tool_result or claude_code.tool_decision log events found in OTel export"},
		{fixture: "unknown_events.json", metric: MetricTiming,
			want: "no claude_code.api_request log events found in OTel export"},
		{fixture: "gauge_not_sum.json", metric: MetricTokens,
			want: "no readable claude_code.token.usage metric in OTel export: it carried no sum data points"},
		{fixture: "unreadable_temporality.json", metric: MetricTokens,
			wantIn: []string{"no readable claude_code.token.usage metric in OTel export:",
				"2 data points declared an aggregationTemporality that is neither 1 (delta) nor 2 (cumulative)"},
			wantOut: []string{"no data point carried both a recognised type attribute"}},
		{fixture: "untimed_api_request.json", metric: MetricTiming,
			want: "no readable claude_code.api_request log events in OTel export: none carried a parseable timeUnixNano"},

		// The tool-call reason is composed from the counters the walk kept, so
		// it can name every defect it saw and cannot state a count nothing
		// observed.
		{fixture: "unreadable_tool_events.json", metric: MetricToolCalls,
			want: "no tool call outcomes in OTel export: " +
				"1 claude_code.tool_result events carried no tool_name; " +
				"1 claude_code.tool_result events carried no readable success value; " +
				"1 claude_code.tool_decision events carried no recognised decision"},
		{fixture: "accepts_no_results.json", metric: MetricToolCalls,
			want: "no tool call outcomes in OTel export: " +
				"2 claude_code.tool_decision events were accepts, and only claude_code.tool_result " +
				"reports an outcome — the export may have been captured before those tools completed"},
		{fixture: "unnamed_reject.json", metric: MetricToolCalls,
			want: "no tool call outcomes in OTel export: " +
				"1 claude_code.tool_decision events recorded a reject with no tool_name"},

		// The mandate boundary: skill.name is carried verbatim for a
		// user-defined skill, and the adapter says it does not read it — not
		// that the name was redacted.
		{fixture: "skill_name_present.json", metric: MetricSkillActivation,
			wantIn:  []string{"skill.name", "0.5.0"},
			wantOut: []string{"redact", "OTEL_LOG_TOOL_DETAILS", "custom_skill"}},
		{fixture: "skill_name_present.json", metric: MetricAttribution,
			want: "Claude Code telemetry carries no output-to-skill mapping"},
	}

	for _, tc := range cases {
		t.Run(tc.fixture+"/"+string(tc.metric), func(t *testing.T) {
			adapter := ClaudeCodeAdapter{OtelExportFile: fixture(tc.fixture)}
			profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
			if err != nil {
				t.Fatal(err)
			}
			got, ok := capturedSignals(profile)[tc.metric]
			if !ok {
				t.Fatalf("no result for %s in the profile", tc.metric)
			}
			if tc.want != "" && got.raw.Reason != tc.want {
				t.Errorf("reason =\n  %q\nwant\n  %q", got.raw.Reason, tc.want)
			}
			for _, want := range tc.wantIn {
				if !strings.Contains(got.raw.Reason, want) {
					t.Errorf("reason = %q, want it to name %q", got.raw.Reason, want)
				}
			}
			for _, unwanted := range tc.wantOut {
				if strings.Contains(got.raw.Reason, unwanted) {
					t.Errorf("reason = %q, must not contain %q", got.raw.Reason, unwanted)
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
		fixture string
		content string // written to a temp file instead, when set
		write   bool
		wantIn  string
	}{
		{name: "malformed JSON", fixture: "malformed.json", wantIn: "malformed JSON"},
		{name: "not JSON at all", content: "resourceMetrics: none\n", write: true, wantIn: "malformed JSON"},
		{name: "an array of exports", fixture: "top_level_array.json", wantIn: "top-level JSON value"},
		{name: "a value that does not fit the schema", fixture: "type_mismatch.json", wantIn: "OTLP schema"},
		{name: "an empty file", fixture: "empty.json", wantIn: "empty"},
		{name: "file does not exist", wantIn: "failed to read OTel export file"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "otel.json")
			switch {
			case tc.write:
				if err := os.WriteFile(path, []byte(tc.content), 0644); err != nil {
					t.Fatal(err)
				}
			case tc.fixture != "":
				path = fixture(tc.fixture)
			}
			adapter := ClaudeCodeAdapter{OtelExportFile: path}
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
// however the exporter ordered them — across batches as well as within one.
func TestTimingIsASpanNotFileOrder(t *testing.T) {
	adapter := ClaudeCodeAdapter{OtelExportFile: fixture("out_of_order.ndjson")}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Timing.Value == nil {
		t.Fatalf("timing value is nil (state %q, reason %q)", profile.Timing.State, profile.Timing.Reason)
	}
	got := *profile.Timing.Value
	if got.StartTime != "2026-09-13T20:49:55.1Z" {
		t.Errorf("start_time = %q, want the earliest event 2026-09-13T20:49:55.1Z", got.StartTime)
	}
	if got.EndTime != "2026-09-13T20:49:56.272Z" {
		t.Errorf("end_time = %q, want the latest event 2026-09-13T20:49:56.272Z", got.EndTime)
	}
	if got.TotalMs < 0 {
		t.Errorf("total_ms = %d, a session cannot take negative time", got.TotalMs)
	}
	if got.TotalMs != 1172 {
		t.Errorf("total_ms = %d, want 1172", got.TotalMs)
	}
}

// A single api_request is a zero-length span, which is a value that was read —
// not an absence.
func TestTiming_SingleRequestIsAZeroLengthSpan(t *testing.T) {
	adapter := ClaudeCodeAdapter{OtelExportFile: fixture("timing_only.json")}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Timing.State != MetricPresent {
		t.Fatalf("timing state = %q (reason %q), want present", profile.Timing.State, profile.Timing.Reason)
	}
	got := *profile.Timing.Value
	want := TimingData{StartTime: "2026-09-13T20:49:55.1Z", EndTime: "2026-09-13T20:49:55.1Z", TotalMs: 0}
	if got != want {
		t.Errorf("timing = %+v, want %+v", got, want)
	}
}

// Token counts are the sum of what was read, never a zero standing in for what
// could not be read — and a present result carries no reason, because profile/v1
// has nowhere to report the points that were skipped.
func TestTokens_OnlyReadableMetricsAreCounted(t *testing.T) {
	adapter := ClaudeCodeAdapter{OtelExportFile: fixture("as_double_rounding.json")}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Tokens.Value == nil {
		t.Fatalf("tokens value is nil (state %q, reason %q)", profile.Tokens.State, profile.Tokens.Reason)
	}
	// 1522.7 rounds to 1523; truncation would lose a token that was counted.
	want := TokenCounts{Input: 1523}
	if *profile.Tokens.Value != want {
		t.Errorf("tokens = %+v, want %+v — only the readable point counts, rounded", *profile.Tokens.Value, want)
	}
	if profile.Tokens.Reason != "" {
		t.Errorf("tokens reason = %q, want empty — a present result carries no reason", profile.Tokens.Reason)
	}
}

// Aggregation temporality decides whether data points for one series add up or
// supersede each other. Getting it backwards double-counts or undercounts every
// token in the session, silently.
func TestTokens_TemporalityDecidesSumOrSupersede(t *testing.T) {
	cases := []struct {
		name    string
		fixture string
		want    TokenCounts
	}{
		{"delta sums across batches", "multi_batch_delta.ndjson", TokenCounts{Input: 600}},
		// The series reports a running total: 900 is the total, not 500+900.
		// A second series on another model is a different series and adds.
		{"cumulative keeps the last value per series", "cumulative.ndjson", TokenCounts{Input: 1150}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := ClaudeCodeAdapter{OtelExportFile: fixture(tc.fixture)}
			profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
			if err != nil {
				t.Fatal(err)
			}
			if profile.Tokens.Value == nil {
				t.Fatalf("tokens value is nil (state %q, reason %q)", profile.Tokens.State, profile.Tokens.Reason)
			}
			if *profile.Tokens.Value != tc.want {
				t.Errorf("tokens = %+v, want %+v", *profile.Tokens.Value, tc.want)
			}
		})
	}
}

// The OTLP spec accepts every 64-bit integer as a number or a decimal string,
// and the two producers in the documented capture routes disagree about which
// they emit. Both must read.
func TestOTLP_NumbersAndStringsBothDecode(t *testing.T) {
	adapter := ClaudeCodeAdapter{OtelExportFile: fixture("number_string_variants.json")}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Tokens.Value == nil {
		t.Fatalf("tokens value is nil (state %q, reason %q)", profile.Tokens.State, profile.Tokens.Reason)
	}
	// asInt as a quoted string and as a bare number.
	want := TokenCounts{Input: 1523, Output: 412}
	if *profile.Tokens.Value != want {
		t.Errorf("tokens = %+v, want %+v", *profile.Tokens.Value, want)
	}
	// timeUnixNano as an unquoted number.
	if profile.Timing.State != MetricPresent {
		t.Errorf("timing state = %q (reason %q), want present", profile.Timing.State, profile.Timing.Reason)
	}
	// success as a real JSON boolean rather than the documented string.
	assertToolCalls(t, profile.ToolCalls.Value, []ToolCallEntry{
		{Name: "Read", Timestamp: "2026-09-13T20:49:55.46Z", Success: true},
	})
}

// What a tool-call entry means: the tool ran and succeeded, read from
// claude_code.tool_result, plus the calls that were rejected and never ran.
func TestToolCalls_EntriesRecordExecutionOutcomes(t *testing.T) {
	cases := []struct {
		name    string
		fixture string
		want    []ToolCallEntry
	}{
		{
			name:    "the three ways a record names its event",
			fixture: "tool_calls_only.json",
			want: []ToolCallEntry{
				{Name: "Read", Timestamp: "2026-09-13T20:49:55.46Z", Success: true},
				{Name: "Bash", Timestamp: "2026-09-13T20:49:55.56Z", Success: true},
				{Name: "Write", Timestamp: "2026-09-13T20:49:55.66Z", Success: false},
			},
		},
		{
			// The accept's outcome comes from its result, so the two sources
			// cannot describe the same call and no de-duplication is needed.
			name:    "an accept and its result are one call",
			fixture: "accept_then_result.json",
			want: []ToolCallEntry{
				{Name: "Read", Timestamp: "2026-09-13T20:49:55.46Z", Success: true},
			},
		},
		{
			// The second result carries no success value: it is not assimilated
			// and never serialises as a failed call.
			name:    "a failure is listed, an unreadable outcome is not",
			fixture: "tool_failure.json",
			want: []ToolCallEntry{
				{Name: "Write", Timestamp: "2026-09-13T20:49:55.46Z", Success: false},
			},
		},
		{
			// The call was read; only its timestamp was not. Dropping it would
			// make tool_calls lie about how many calls the session made.
			name:    "a call with no timestamp sorts last",
			fixture: "no_timestamp_tool_call.json",
			want: []ToolCallEntry{
				{Name: "Read", Timestamp: "2026-09-13T20:49:55.46Z", Success: true},
				{Name: "Bash", Timestamp: "", Success: true},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := ClaudeCodeAdapter{OtelExportFile: fixture(tc.fixture)}
			profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
			if err != nil {
				t.Fatal(err)
			}
			if profile.ToolCalls.State != MetricPresent {
				t.Fatalf("tool_calls state = %q (reason %q), want present", profile.ToolCalls.State, profile.ToolCalls.Reason)
			}
			assertToolCalls(t, profile.ToolCalls.Value, tc.want)
		})
	}
}

// Multiple resourceMetrics and scopeMetrics entries per object are legal, and a
// walk that indexes [0] loses most of the file. The `type` attribute is read
// only on claude_code.token.usage, because other metrics carry their own.
func TestTokens_EveryResourceAndScopeIsWalked(t *testing.T) {
	adapter := ClaudeCodeAdapter{OtelExportFile: fixture("tokens_only.json")}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Tokens.Value == nil {
		t.Fatalf("tokens value is nil (state %q, reason %q)", profile.Tokens.State, profile.Tokens.Reason)
	}
	want := TokenCounts{Input: 1523, Output: 412, CacheRead: 20480, CacheCreation: 3072}
	if *profile.Tokens.Value != want {
		t.Errorf("tokens = %+v, want %+v", *profile.Tokens.Value, want)
	}
}

// Values, not just states, survive a partial export — the concrete regression
// behind the contract test above.
func TestCapture_ClaudeCode_PartialExport_KeepsToolCallsAndTiming(t *testing.T) {
	adapter := ClaudeCodeAdapter{OtelExportFile: fixture("partial_no_tokens.json")}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}

	assertToolCalls(t, profile.ToolCalls.Value, []ToolCallEntry{
		{Name: "Bash", Timestamp: "2026-09-13T20:49:55.46Z", Success: true},
	})
	if profile.Timing.Value == nil {
		t.Fatalf("timing value is nil (state %q, reason %q)", profile.Timing.State, profile.Timing.Reason)
	}
	if profile.Timing.Value.TotalMs != 1172 {
		t.Errorf("timing total_ms = %d, want 1172", profile.Timing.Value.TotalMs)
	}

	// The one signal that genuinely is absent stays honest.
	if profile.Tokens.State != MetricUnknown {
		t.Errorf("tokens state = %q, want %q", profile.Tokens.State, MetricUnknown)
	}
	if profile.Tokens.Value != nil {
		t.Errorf("tokens value = %+v, want nil when no token metric was exported", profile.Tokens.Value)
	}
}
