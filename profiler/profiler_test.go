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
	r := PresentTokenResult(TokenCounts{Input: Count(100), Output: Count(50)}, "otel")
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
	// Counts survive the round trip as counts, and the two the caller never
	// set stay unset rather than arriving as zeros.
	assertTokenJSON(t, parsed.Value, `{"input":100,"output":50}`)
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

// fixtureSession is the session.id every export under testdata/otlp carries.
// A capture reads only the records that carry the session it was asked for, so
// a test reading a fixture's numbers has to ask for the session that produced
// them.
const fixtureSession = "00000000-0000-4000-8000-000000000001"

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

// Well-formed JSON that carries no OTLP envelope parsed fine and carries no
// telemetry, so every signal — including the two that are always none — is
// none. There is nothing about skill_activation or attribution that this input
// makes different, so there is nothing to skip.
func TestCapabilityReport_ClaudeCode_NoOtlpEnvelope(t *testing.T) {
	adapter := ClaudeCodeAdapter{OtelExportFile: fixture("no_envelope.json")}
	cap := adapter.Probe()

	if len(cap.Capabilities) != 5 {
		t.Fatalf("capability report covers %d signals, want 5", len(cap.Capabilities))
	}
	for metric, source := range cap.Capabilities {
		if source != SourceNone {
			t.Errorf("%s = %q, want none for a file carrying no OTLP envelope", metric, source)
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
	// All four token types, and no reasoning key: Claude Code has no reasoning
	// token type, so nothing was read for it and the profile says nothing.
	assertTokenJSON(t, profile.Tokens.Value,
		`{"input":1523,"output":412,"cache_read":20480,"cache_creation":3072}`)
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

	// The three OTel signals carry the fallback reason the spec quotes, word
	// for word. A substring check passes against a sentence that says the
	// opposite; the spec quotes this one, so the test has to hold it to it.
	for metric, got := range map[MetricName]string{
		MetricTokens:    profile.Tokens.Reason,
		MetricToolCalls: profile.ToolCalls.Reason,
		MetricTiming:    profile.Timing.Reason,
	} {
		if got != fallbackReasonInSpec {
			t.Errorf("%s reason =\n  %q\nwant the reason docs/profiler-spec.md quotes under Fallback:\n  %q",
				metric, got, fallbackReasonInSpec)
		}
	}
}

// The two reasons docs/profiler-spec.md quotes verbatim, copied from the spec
// rather than from the code, so that a change to either without a change to the
// other is a failing test rather than a documentation defect nobody notices.
const (
	// docs/profiler-spec.md, "Fallback".
	fallbackReasonInSpec = "OTel export not configured. Provide an OTel export file via --otel-file or OtelExportFile."
	// docs/profiler-spec.md, "Capture logic" step 6.
	activationReasonInSpec = "This adapter does not yet read Claude Code's skill telemetry: the " +
		"claude_code.skill_activated event, logged when a skill is invoked through the Skill tool or a / command, " +
		"carries skill.name, invocation_trigger, skill.source and skill.kind. Reading it is 0.5.0."
)

// skill_activation and attribution are a property of the harness and of this
// adapter, not of any export, so their reasons are the same in every case the
// adapter can be in — and both are quoted in the spec.
func TestCapture_TheHarnessLevelReasonsAreTheSpecsWordForWord(t *testing.T) {
	for _, name := range []string{"full_export.ndjson", "skill_name_present.json", "malformed.json", "no_envelope.json"} {
		t.Run(name, func(t *testing.T) {
			profile := capturedProfile(t, name)
			if got := profile.SkillActivation.Reason; got != activationReasonInSpec {
				t.Errorf("skill_activation reason =\n  %q\nwant\n  %q", got, activationReasonInSpec)
			}
			if got := profile.Attribution.Reason; got != "Claude Code telemetry carries no output-to-skill mapping" {
				t.Errorf("attribution reason = %q", got)
			}
		})
	}
	// And with no export configured at all.
	adapter := ClaudeCodeAdapter{}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}
	if got := profile.SkillActivation.Reason; got != activationReasonInSpec {
		t.Errorf("skill_activation reason with no export =\n  %q\nwant\n  %q", got, activationReasonInSpec)
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

// The schema string is the contract F04 reads profiles by, so the literal is
// what the test holds it to. Comparing the profile's field to the constant that
// produced it passes whatever the constant says, including a typo.
func TestProfileSchemaField(t *testing.T) {
	const want = "skill-architect/profile/v1"

	if ProfileSchema != want {
		t.Errorf("ProfileSchema = %q, want %q — schema v1 is what consumers are reading", ProfileSchema, want)
	}

	adapter := ClaudeCodeAdapter{}
	profile, _ := adapter.Capture("s", CaptureOpts{SnapshotHash: "h", SkillDir: "/d"})
	if profile.Schema != want {
		t.Errorf("schema = %q, want %q", profile.Schema, want)
	}

	// Verify via JSON marshal too (custom MarshalJSON).
	data, _ := json.Marshal(profile)
	var raw map[string]any
	json.Unmarshal(data, &raw)
	if raw["schema"] != want {
		t.Errorf("json schema = %v, want %q", raw["schema"], want)
	}
}

// The adapter version is a release surface: it goes into every profile, and
// tests/test_skill.sh asserts the same number beside the five plugin manifests.
// Pinning the literal here is what makes a forgotten bump fail rather than
// quietly ship a 0.4.2 profile labelled as something else.
func TestAdapterVersionIsThisRelease(t *testing.T) {
	const want = "0.4.2"
	if AdapterVersion != want {
		t.Errorf("AdapterVersion = %q, want %q", AdapterVersion, want)
	}
	report := ClaudeCodeAdapter{}.Probe()
	if report.AdapterVer != want {
		t.Errorf("capability.adapter_version = %q, want %q", report.AdapterVer, want)
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

// captureCase is one export shape the contract above is asserted over. The
// table is package-level so the fixture directory can be checked against it:
// see TestEveryFixtureIsAContractCase, which is what keeps "add a fixture" and
// "assert the contract for it" from being two things someone has to remember.
type captureCase struct {
	name string
	// fixture is the OTLP export the adapter is pointed at, unless one of
	// the two flags below says otherwise.
	fixture      string
	unconfigured bool         // adapter has no export file at all
	missingFile  bool         // adapter points at a path that does not exist
	present      []MetricName // signals that must be "present"
	absent       MetricState  // state required of tokens/tool_calls/timing when not present
}

var captureCases = []captureCase{
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
	{name: "only a cache count was exported", fixture: "cache_only.json",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "counts read as zero", fixture: "zero_token_count.json",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "two series told apart by a non-string attribute", fixture: "array_attribute_series.ndjson",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "two resources' cumulative series", fixture: "multi_resource_cumulative.json",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "the same two resources under delta", fixture: "multi_resource_delta.json",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "several instrumentation scopes", fixture: "multi_scope_cumulative.json",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "two resources told apart by a non-string attribute", fixture: "resource_identity_kinds.ndjson",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "a cumulative counter that reset", fixture: "counter_reset.ndjson",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "a cumulative counter that did not reset", fixture: "counter_no_reset.ndjson",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "cumulative points carrying no start time", fixture: "absent_start_time.json",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "cumulative points whose start time does not read", fixture: "unreadable_start_time.json",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "a run beside a flush that carried no start time", fixture: "mixed_start_time.json",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "one run whose flushes disagree", fixture: "run_flushes_disagree.ndjson",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "an unplaceable total above every run", fixture: "unplaced_above_every_run.json",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "an unplaceable total under the runs' sum", fixture: "unplaced_under_the_runs_sum.json",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "start times that are zero and start times that are absent", fixture: "zero_start_time.json",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "a series mixing temporalities beside a well-formed one", fixture: "mixed_temporality.ndjson",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "the temporality enum spelled out by name", fixture: "temporality_enum_names.json",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "asDouble beside asInt on one point", fixture: "as_double_wins_over_as_int.json",
		present: []MetricName{MetricTokens}, absent: MetricUnknown},
	{name: "log records spread over several resources and scopes", fixture: "log_record_shapes.json",
		present: []MetricName{MetricToolCalls}, absent: MetricUnknown},

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
	{name: "an envelope carrying no telemetry", fixture: "empty_envelope.json", absent: MetricUnknown},
	{name: "a sum that declared no temporality", fixture: "absent_temporality.json", absent: MetricUnknown},
	{name: "a token type the adapter does not recognise", fixture: "unrecognised_token_type.json", absent: MetricUnknown},
	{name: "every series mixes delta and cumulative", fixture: "mixed_temporality_only.ndjson", absent: MetricUnknown},
	{name: "two series each mixing delta and cumulative", fixture: "mixed_temporality_two_series.ndjson", absent: MetricUnknown},
	{name: "values no count can hold", fixture: "value_not_a_count.json", absent: MetricUnknown},

	// Exports that cannot be read as OTLP/JSON at all.
	{name: "malformed JSON", fixture: "malformed.json", absent: MetricError},
	{name: "a partial final line", fixture: "truncated_final_line.ndjson", absent: MetricError},
	{name: "a stray closing brace between two batches", fixture: "stray_close_then_batch.ndjson", absent: MetricError},
	{name: "a value that does not fit the schema", fixture: "type_mismatch.json", absent: MetricError},
	{name: "an array of exports", fixture: "top_level_array.json", absent: MetricError},
	{name: "an empty file", fixture: "empty.json", absent: MetricError},
}

// TestEveryFixtureIsAContractCase makes the fixture directory the coverage
// denominator. A fixture nobody wrote a case for is a shape this contract was
// never asserted over, and the only way to notice is to count.
func TestEveryFixtureIsAContractCase(t *testing.T) {
	covered := make(map[string]bool, len(captureCases))
	for _, tc := range captureCases {
		if tc.fixture != "" {
			covered[tc.fixture] = true
		}
	}
	found, err := filepath.Glob(filepath.Join("testdata", "otlp", "*"))
	if err != nil {
		t.Fatal(err)
	}
	files := 0
	for _, path := range found {
		name := filepath.Base(path)
		if filepath.Ext(name) != ".json" && filepath.Ext(name) != ".ndjson" {
			continue // the directory's own README
		}
		files++
		if !covered[name] {
			t.Errorf("%s has no case in captureCases: the probe/capture contract was never asserted over it", name)
		}
	}
	if files != len(covered) {
		t.Errorf("%d fixtures on disk, %d named in captureCases — one of them names a file that is not there", files, len(covered))
	}
}

func TestCaptureDeliversEverySignalProbeAdvertises(t *testing.T) {
	for _, tc := range captureCases {
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

		// File-level: the JSON object is not an OTLP export at all — neither
		// envelope key is there to be empty.
		{fixture: "no_envelope.json", metric: MetricTokens,
			wantIn:  []string{"no resourceMetrics or resourceLogs found", "OTEL_EXPORTER_OTLP_PROTOCOL=http/json"},
			wantOut: []string{"no claude_code.token.usage metric found"}},
		{fixture: "bespoke_envelope.json", metric: MetricTokens,
			wantIn:  []string{"OTel export is not OTLP/JSON", "no resourceMetrics or resourceLogs found"},
			wantOut: []string{"no claude_code.token.usage metric found"}},

		// An export that carried the envelope and no telemetry inside it is an
		// OTLP export — an empty one. Telling its owner it is "not OTLP/JSON"
		// sends them to fix their exporter protocol, when what they have is a
		// session that emitted nothing yet.
		{fixture: "empty_envelope.json", metric: MetricTokens,
			want: "no claude_code.token.usage metric found in OTel export"},
		{fixture: "empty_envelope.json", metric: MetricToolCalls,
			want: "no claude_code.tool_result or claude_code.tool_decision log events found in OTel export"},
		{fixture: "empty_envelope.json", metric: MetricTiming,
			want: "no claude_code.api_request log events found in OTel export"},

		// Per signal: the counters the walk kept.
		{fixture: "unknown_events.json", metric: MetricTokens,
			want: "no claude_code.token.usage metric found in OTel export"},
		{fixture: "unknown_events.json", metric: MetricToolCalls,
			want: "no claude_code.tool_result or claude_code.tool_decision log events found in OTel export"},
		{fixture: "unknown_events.json", metric: MetricTiming,
			want: "no claude_code.api_request log events found in OTel export"},
		{fixture: "gauge_not_sum.json", metric: MetricTokens,
			want: "no readable claude_code.token.usage metric in OTel export: it carried no sum data points"},
		// Absent, unreadable, and declared-but-neither are three different
		// things to go and look at in a capture, so they are three clauses. A
		// sum that declared nothing did not declare a wrong temporality.
		{fixture: "unreadable_temporality.json", metric: MetricTokens,
			want: "no readable claude_code.token.usage metric in OTel export: " +
				"1 data point carried an aggregationTemporality that could not be read; " +
				"1 data point declared an aggregationTemporality that is neither 1 (delta) nor 2 (cumulative)",
			wantOut: []string{"carried no aggregationTemporality"}},
		{fixture: "absent_temporality.json", metric: MetricTokens,
			want: "no readable claude_code.token.usage metric in OTel export: " +
				"1 data point carried no aggregationTemporality",
			wantOut: []string{"declared an aggregationTemporality"}},
		{fixture: "unrecognised_token_type.json", metric: MetricTokens,
			want: "no readable claude_code.token.usage metric in OTel export: " +
				"2 data points carried no recognised type attribute (input/output/cacheRead/cacheCreation)"},
		// A series carrying both temporalities is refused whole, and the reason
		// counts series rather than data points because that is the unit the
		// defect belongs to: the points are individually fine and it is their
		// company that is malformed.
		{fixture: "mixed_temporality_only.ndjson", metric: MetricTokens,
			want: "no readable claude_code.token.usage metric in OTel export: " +
				"1 time series carried both delta (1) and cumulative (2) aggregationTemporality points",
			wantOut: []string{"data point"}},
		// "time series" is its own plural, and this is the case the helper that
		// knows it exists for: two refused series must not read "2 time seriess".
		{fixture: "mixed_temporality_two_series.ndjson", metric: MetricTokens,
			want: "no readable claude_code.token.usage metric in OTel export: " +
				"2 time series carried both delta (1) and cumulative (2) aggregationTemporality points",
			wantOut: []string{"data point", "seriess"}},
		{fixture: "value_not_a_count.json", metric: MetricTokens,
			want: "no readable claude_code.token.usage metric in OTel export: " +
				"4 data points carried a value that is not a token count: " +
				"a count is a whole number from 0 to 9223372036854775807"},
		{fixture: "untimed_api_request.json", metric: MetricTiming,
			want: "no readable claude_code.api_request log events in OTel export: none carried a parseable timeUnixNano"},

		// The tool-call reason is composed from the counters the walk kept, so
		// it can name every defect it saw and cannot state a count nothing
		// observed.
		{fixture: "unreadable_tool_events.json", metric: MetricToolCalls,
			want: "no tool call outcomes in OTel export: " +
				"1 claude_code.tool_result event carried no tool_name; " +
				"1 claude_code.tool_result event carried no readable success value; " +
				"1 claude_code.tool_decision event carried no recognised decision"},
		{fixture: "accepts_no_results.json", metric: MetricToolCalls,
			want: "no tool call outcomes in OTel export: " +
				"2 accepted claude_code.tool_decision events, and only claude_code.tool_result " +
				"reports an outcome — the export may have been captured before those tools completed"},
		{fixture: "unnamed_reject.json", metric: MetricToolCalls,
			want: "no tool call outcomes in OTel export: " +
				"1 claude_code.tool_decision event recorded a reject with no tool_name"},

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
	// 1522.7 rounds to 1523; truncation would lose a token that was counted.
	// The unreadable output point leaves its key out, rather than reporting a
	// zero for a count the export never delivered.
	assertTokenJSON(t, profile.Tokens.Value, `{"input":1523}`)
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
		want    string
	}{
		{"delta sums across batches", "multi_batch_delta.ndjson", `{"input":600}`},
		// The series reports a running total: 900 is the total, not 500+900.
		// A second series on another model is a different series and adds.
		{"cumulative keeps the last value per series", "cumulative.ndjson", `{"input":1150}`},
		// Both enum names, as the standard protobuf JSON mapping emits them.
		{"the temporality enum spelled out by name", "temporality_enum_names.json", `{"input":100,"output":900}`},
		// One series declares both temporalities, which are opposite
		// instructions: no total it could contribute is in the export, so it
		// contributes none. The well-formed series beside it is untouched —
		// 640, not 690 (its 640 plus the running total) and not 740 (plus the
		// increment). Refusing the series is not refusing the file.
		{"a series carrying both temporalities is refused", "mixed_temporality.ndjson", `{"input":640}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			profile := capturedProfile(t, tc.fixture)
			assertTokenJSON(t, profile.Tokens.Value, tc.want)
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
	// asInt as a quoted string and as a bare number.
	assertTokenJSON(t, profile.Tokens.Value, `{"input":1523,"output":412}`)
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
	profile := capturedProfile(t, "tokens_only.json")
	assertTokenJSON(t, profile.Tokens.Value,
		`{"input":1523,"output":412,"cache_read":20480,"cache_creation":3072}`)
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

// --- What may reach a profile: representability, series identity, absence ---

// capturedProfile runs the adapter over a fixture and fails the test if the
// adapter itself refuses the call, which is never what these cases are about.
func capturedProfile(t *testing.T, name string) Profile {
	t.Helper()
	adapter := ClaudeCodeAdapter{OtelExportFile: fixture(name)}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

// assertTokenJSON compares the captured counts with the JSON the profile
// carries, because absence is the assertion: a count no data point carried has
// no key at all, and a count read as zero has one whose value is 0.
func assertTokenJSON(t *testing.T, got *TokenCounts, want string) {
	t.Helper()
	if counts := tokenJSON(t, got); counts != want {
		t.Errorf("tokens JSON =\n  %s\nwant\n  %s", counts, want)
	}
}

// tokenJSON is the counts as the profile serialises them. It is separate from
// the assertion above so a case that has something to say about why its total
// is what it is can say it in its own failure message.
func tokenJSON(t *testing.T, got *TokenCounts) string {
	t.Helper()
	if got == nil {
		t.Fatalf("tokens value is nil, want counts")
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// A token count is a whole number of tokens: finite, not negative, and small
// enough to be an int64. A data point carrying anything else carried no count,
// and the reader must say so rather than substitute what the conversion
// happens to produce — which Go leaves up to the architecture, so the same
// capture read on two machines produced two different profiles.
func TestTokens_AValueThatIsNotACountIsRefused(t *testing.T) {
	profile := capturedProfile(t, "value_not_a_count.json")

	if profile.Tokens.State != MetricUnknown {
		t.Errorf("tokens state = %q, want %q — no data point carried a count",
			profile.Tokens.State, MetricUnknown)
	}
	if profile.Tokens.Value != nil {
		data, _ := json.Marshal(profile.Tokens.Value)
		t.Errorf("tokens value = %s, want none — every point was out of the range of a count", data)
	}
	if want := "4 data points carried a value that is not a token count"; !strings.Contains(profile.Tokens.Reason, want) {
		t.Errorf("tokens reason = %q, want it to name %q", profile.Tokens.Reason, want)
	}
	if profile.Capability.Capabilities[MetricTokens] != SourceNone {
		t.Errorf("probe advertised tokens = %q with no readable count in the export",
			profile.Capability.Capabilities[MetricTokens])
	}
}

// OTel identifies a time series by its whole attribute set, not by the subset
// this adapter happens to read as text. Two series that merge are one session's
// tokens reported as a fraction of themselves.
func TestTokens_SeriesDifferingOnlyByANonStringAttributeDoNotMerge(t *testing.T) {
	profile := capturedProfile(t, "array_attribute_series.ndjson")

	// Two cumulative series, 100 and 200, distinguished only by an arrayValue
	// attribute. Merged, they are one series and hold the greatest running
	// total reported on it, which is 200 whichever order the points arrive in;
	// kept apart, they are two running totals and add.
	assertTokenJSON(t, profile.Tokens.Value, `{"input":300}`)
}

// The attribute set is only part of a series' identity. OTel identifies a time
// series by the resource it was exported from, the instrumentation scope that
// recorded it, the metric's name and the data point's attributes — so two points
// carrying identical attributes under different resources or scopes are two
// series. Merging them is an undercount under cumulative temporality, where one
// running total replaces another instead of adding to it.
func TestTokens_ASeriesIsResourceScopeMetricAndAttributes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fixture string
		want    string
		reason  string
	}{
		{
			name:    "two resources under cumulative are two running totals",
			fixture: "multi_resource_cumulative.json",
			want:    `{"input":300}`,
			reason:  "a collector fanning in two service.instance.ids carries two series, 100 and 200 — not one reporting 200",
		},
		{
			name:    "the same two resources under delta are unchanged",
			fixture: "multi_resource_delta.json",
			want:    `{"input":300}`,
			reason:  "delta points add up whatever series they belong to, and identity must not change that",
		},
		{
			name:    "scopes are told apart by name, version and attributes",
			fixture: "multi_scope_cumulative.json",
			want:    `{"input":400}`,
			reason:  "four scopes differing by name, by version and by attributes are four series: 100 + 200 + 40 + 60",
		},
		{
			name:    "resource identity does not collapse two AnyValue kinds",
			fixture: "resource_identity_kinds.ndjson",
			want:    `{"input":300}`,
			reason:  `an intValue 5 and the string "5" are two resources, as they are two attribute sets`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile := capturedProfile(t, tc.fixture)
			if profile.Tokens.State != MetricPresent {
				t.Fatalf("tokens state = %q (reason %q), want present", profile.Tokens.State, profile.Tokens.Reason)
			}
			if got := tokenJSON(t, profile.Tokens.Value); got != tc.want {
				t.Errorf("tokens JSON =\n  %s\nwant\n  %s\n— %s", got, tc.want, tc.reason)
			}
		})
	}
}

// A cumulative counter that restarts inside one capture reports a running total
// from zero again, and startTimeUnixNano is what says so: a point whose start
// differs from the points before it belongs to a new run. The run before the
// reset is tokens the session really spent, so it is kept and added rather than
// replaced — replacing it reports a fraction of the capture as the whole of it.
// What one run holds is the greatest running total its points reported, which
// is the only reading that does not depend on an order the points may not
// carry and cannot be talked below a total the export states outright.
//
// A point whose start time is absent, zero or unreadable cannot say which run
// it came from — but it came from one, so it is neither a run of its own nor
// free. It joins the run it is cheapest to have come from, which is the largest
// one, and costs the series only what it reports over and above that run. A
// capture that carries start times on some points and not others is therefore
// never reported at twice its size, and never below the runs that did name
// themselves either.
func TestTokens_ACumulativeResetKeepsTheRunBeforeIt(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fixture string
		want    string
		reason  string
	}{
		{
			name:    "a reset keeps the run before it",
			fixture: "counter_reset.ndjson",
			want:    `{"input":120}`,
			reason:  "the first run ended at 100 and the second reached 20, so the capture holds 120",
		},
		{
			name:    "with no reset the greatest running total is the whole of it",
			fixture: "counter_no_reset.ndjson",
			want:    `{"input":120}`,
			reason:  "one run reporting 60, 100 then 120 is 120 — runs that are one run must not add",
		},
		{
			name:    "one run whose flushes disagree holds the greatest of them",
			fixture: "run_flushes_disagree.ndjson",
			want:    `{"input":900}`,
			reason: "one run reported 500, then 900, then 700: it reached 900, and the 700 after it " +
				"is a capture contradicting itself rather than tokens given back — 700 is the " +
				"answer that lets a later flush unsee a total the export carries",
		},
		{
			name:    "points carrying no start time name no run",
			fixture: "absent_start_time.json",
			want:    `{"input":120}`,
			reason: "nothing places these points against a run, and with no runs to place them " +
				"against the greatest running total they reported is the whole of what the " +
				"capture guarantees",
		},
		{
			name:    "points whose start time cannot be read name no run either",
			fixture: "unreadable_start_time.json",
			want:    `{"input":120}`,
			reason:  "a start time that does not read is not evidence of a reset, and inventing one double-counts",
		},
		{
			name:    "an unplaceable total above every run keeps the runs it did not join",
			fixture: "unplaced_above_every_run.json",
			want:    `{"input":520}`,
			reason: "runs of 100 and 20 beside an unplaceable 500: the 500 is a running total of the " +
				"run that reached 100, and the other run's 20 is still beside it. 500 drops a run " +
				"that named itself; 620 invents a third run",
		},
		{
			name:    "an unplaceable total under the runs' sum still costs what it exceeds them by",
			fixture: "unplaced_under_the_runs_sum.json",
			want:    `{"input":350}`,
			reason: "three runs of 100 beside an unplaceable 150: whichever run the 150 came from " +
				"reached 150 and the other two still hold 100 each — comparing the point with " +
				"the runs' sum instead never fires here and loses 50 tokens the points reported",
		},
		{
			name:    "a run and a flush that omitted its start time are one series",
			fixture: "mixed_start_time.json",
			want:    `{"input":1200050}`,
			reason: "one session, one series: a final flush with no start time is the running total it " +
				"reports, not a second session's worth of tokens on top of the first",
		},
		{
			name:    "a start time of zero is a start time that is absent",
			fixture: "zero_start_time.json",
			want:    `{"input":120}`,
			reason: "startTimeUnixNano is a proto3 fixed64: an explicit 0 and an absent field are two " +
				"encodings of one message, and a reader that tells them apart splits one series in two",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile := capturedProfile(t, tc.fixture)
			if profile.Tokens.State != MetricPresent {
				t.Fatalf("tokens state = %q (reason %q), want present", profile.Tokens.State, profile.Tokens.Reason)
			}
			if got := tokenJSON(t, profile.Tokens.Value); got != tc.want {
				t.Errorf("tokens JSON =\n  %s\nwant\n  %s\n— %s", got, tc.want, tc.reason)
			}
		})
	}
}

// A count the export did not carry is not a zero. The profile omits it, so a
// reader — and F04 — can tell "the session used no output tokens" from "this
// export said nothing about output tokens". A count that was read as zero is a
// measurement and keeps its key.
func TestTokens_AnUnreadCountIsAbsentAndAReadZeroIsZero(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fixture string
		want    string
	}{
		{
			name:    "a cache-only export says nothing about input or output",
			fixture: "cache_only.json",
			want:    `{"cache_read":20480}`,
		},
		{
			name:    "a count read as zero is reported as zero",
			fixture: "zero_token_count.json",
			want:    `{"input":0,"cache_creation":0}`,
		},
		{
			name:    "an unreadable point leaves its type absent, not zero",
			fixture: "as_double_rounding.json",
			want:    `{"input":1523}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile := capturedProfile(t, tc.fixture)
			if profile.Tokens.State != MetricPresent {
				t.Fatalf("tokens state = %q (reason %q), want present", profile.Tokens.State, profile.Tokens.Reason)
			}
			assertTokenJSON(t, profile.Tokens.Value, tc.want)
		})
	}
}

// A log record's identity comes from its body, and the event.name attribute is
// the fallback for a record whose body was dropped. When the two disagree the
// body wins, because event.name is the one attribute cardinality limits can
// replace — and getting it backwards files a tool call under timing.
//
// The same fixture spreads its records over two resourceLogs with two
// scopeLogs each: a walk that indexes [0] loses three quarters of them.
func TestToolCalls_TheBodyNamesTheEventAndEveryScopeIsWalked(t *testing.T) {
	profile := capturedProfile(t, "log_record_shapes.json")

	if profile.ToolCalls.State != MetricPresent {
		t.Fatalf("tool_calls state = %q (reason %q), want present", profile.ToolCalls.State, profile.ToolCalls.Reason)
	}
	// Timed calls first in timestamp order, then the untimed ones in file
	// order — which here means across resources and scopes, in the order the
	// walk visits them.
	assertToolCalls(t, profile.ToolCalls.Value, []ToolCallEntry{
		{Name: "Read", Timestamp: "2026-09-13T20:49:55.46Z", Success: true},
		{Name: "Write", Timestamp: "2026-09-13T20:49:55.56Z", Success: true},
		{Name: "Bash", Timestamp: "", Success: true},
		{Name: "Grep", Timestamp: "", Success: false},
	})

	// The first record's event.name attribute says api_request. Its body says
	// tool_result, and the body is what counts — so there is no api_request in
	// this export at all.
	if profile.Timing.State != MetricUnknown {
		t.Errorf("timing state = %q, want unknown — event.name overrode the body", profile.Timing.State)
	}
	if want := "no claude_code.api_request log events found in OTel export"; profile.Timing.Reason != want {
		t.Errorf("timing reason = %q, want %q", profile.Timing.Reason, want)
	}
}

// A data point may carry both asDouble and asInt. Claude Code's own exporter
// sets asDouble, and a collector re-serialising the same number may add asInt,
// so asDouble is the one that is read — consistently, not by whichever the
// decoder saw first.
func TestTokens_AsDoubleIsReadBeforeAsInt(t *testing.T) {
	profile := capturedProfile(t, "as_double_wins_over_as_int.json")
	assertTokenJSON(t, profile.Tokens.Value, `{"input":1523}`)
}
