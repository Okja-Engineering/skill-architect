package profiler

// What `doctor` is allowed to say about this machine.
//
// `doctor` is the single most likely place in this product to overstate,
// because its whole job is making claims about capability. Three adapters were
// kept out of this release for advertising signals their captures returned
// nothing for; a tier is the same claim with a shorter name. So the assertions
// here are as much about what the report may not say as about what it says, and
// every tier it can report is proved present with a real input — see
// adapter_contract_test.go, where the tiers are a table over the same fixtures
// the adapter contract uses.
//
// Every home below comes from homesafe.SandboxHome. That is not tidiness: these
// tests seed a `.cursor/hooks.json` and a spool *inside* the home they then ask
// about, so a home that resolved to the real one would rewrite the machine's
// own Cursor configuration.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Okja-Engineering/skill-architect/profiler/internal/homesafe"
)

// TestDetectEnvironment_ABareMachineMeasuresNothing is the honest floor. No
// export, nothing registered, no spool: the answer is none, and it says what
// would change the answer.
func TestDetectEnvironment_ABareMachineMeasuresNothing(t *testing.T) {
	home := homesafe.SandboxHome(t)

	rep := DetectEnvironment(EnvironmentQuery{Home: home})

	if rep.Measurement.Tier != TierNone {
		t.Errorf("tier = %q, want %q", rep.Measurement.Tier, TierNone)
	}
	if rep.Measurement.Reason == "" {
		t.Error("a tier of none with no reason tells a user nothing about what to do next")
	}
	if len(rep.Measurement.Signals) != 0 {
		t.Errorf("signals = %v on a machine where nothing was probed", rep.Measurement.Signals)
	}
	if rep.Observed.HooksJSON.RegisteredEvents != nil {
		t.Errorf("registered_hook_events = %v with no hooks.json", rep.Observed.HooksJSON.RegisteredEvents)
	}
	if rep.Observed.Spool.Exists {
		t.Error("a spool was reported on a machine that has none")
	}
	if _, err := time.Parse(time.RFC3339, rep.DetectedAt); err != nil {
		t.Errorf("detected_at %q is not RFC3339: %v — a report that cannot be dated cannot be told it is stale", rep.DetectedAt, err)
	}
	if rep.AdapterVersion != AdapterVersion {
		t.Errorf("adapter_version = %q, want %q — a report is quoted later and has to say which build said it", rep.AdapterVersion, AdapterVersion)
	}
}

// TestDetectEnvironment_ReportsTheExportItProbed is the one tier that can be
// present, proved present with the same fixture the adapter contract uses.
//
// The signals are the probe's own, not a list this function keeps: a tier above
// none means a registered harness read something out of a real file, and the
// evidence for that is the capability report it produced.
func TestDetectEnvironment_ReportsTheExportItProbed(t *testing.T) {
	export := fixture("full_export.ndjson")

	rep := DetectEnvironment(EnvironmentQuery{
		Home:       homesafe.SandboxHome(t),
		Harness:    "claude_code",
		ExportFile: export,
	})

	if rep.Measurement.Tier != TierExport {
		t.Fatalf("tier = %q, want %q (reason %q)", rep.Measurement.Tier, TierExport, rep.Measurement.Reason)
	}
	if rep.Measurement.Harness != "claude_code" {
		t.Errorf("harness = %q, want the one that was probed", rep.Measurement.Harness)
	}
	if rep.Measurement.Export != export {
		t.Errorf("export = %q, want %q", rep.Measurement.Export, export)
	}

	// At least one signal from a real source, and every signal the report
	// carries is one the probe named. The set is the adapter's, so this asserts
	// agreement rather than restating it.
	adapter, ok := NewAdapter("claude_code", export)
	if !ok {
		t.Fatal("the registry has no claude_code adapter, so this test proves nothing")
	}
	want := adapter.Probe().Capabilities
	if len(want) == 0 {
		t.Fatal("the probe advertises nothing over this fixture, so a tier above none could not be justified by it")
	}
	real := 0
	for metric, source := range want {
		if got := rep.Measurement.Signals[metric]; got != source {
			t.Errorf("signals[%s] = %q, want the probe's %q", metric, got, source)
		}
		if source != SourceNone {
			real++
		}
	}
	if real == 0 {
		t.Error("no signal came from a real source, so this fixture cannot justify a tier above none")
	}
}

// TestDetectEnvironment_RefusesToNameATierForAnExportThatReadsNothing is the
// direction that matters most.
//
// An export supplied is not a capability. A file that parses and carries
// nothing this adapter can read leaves the tier at none, with the reason naming
// the file — because "you gave me a file" and "I read a measurement out of it"
// are the two things this product has repeatedly confused.
func TestDetectEnvironment_RefusesToNameATierForAnExportThatReadsNothing(t *testing.T) {
	for _, tc := range []struct{ name, export string }{
		{"a file that is not an export at all", fixture("no_envelope.json")},
		{"an export carrying no data points", fixture("empty_envelope.json")},
		{"a file that cannot be read as an export", filepath.Join(t.TempDir(), "absent.json")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rep := DetectEnvironment(EnvironmentQuery{
				Home:       homesafe.SandboxHome(t),
				Harness:    "claude_code",
				ExportFile: tc.export,
			})
			if rep.Measurement.Tier != TierNone {
				t.Errorf("tier = %q, want %q", rep.Measurement.Tier, TierNone)
			}
			if !strings.Contains(rep.Measurement.Reason, tc.export) {
				t.Errorf("reason = %q, want it to name the file it could not read a signal out of", rep.Measurement.Reason)
			}
		})
	}
}

// TestDetectEnvironment_NamesTheHarnessesItAcceptsWhenGivenOne is the refusal
// for a harness nobody ships, and it reads the set from the registry rather
// than restating it — the same rule the CLI's own refusal keeps.
func TestDetectEnvironment_NamesTheHarnessesItAcceptsWhenGivenOne(t *testing.T) {
	rep := DetectEnvironment(EnvironmentQuery{
		Home:       homesafe.SandboxHome(t),
		Harness:    "cursor",
		ExportFile: fixture("full_export.ndjson"),
	})

	if rep.Measurement.Tier != TierNone {
		t.Errorf("tier = %q, want %q", rep.Measurement.Tier, TierNone)
	}
	for _, want := range []string{"cursor", SupportedHarnesses()} {
		if !strings.Contains(rep.Measurement.Reason, want) {
			t.Errorf("reason = %q, want it to name %q", rep.Measurement.Reason, want)
		}
	}
}

// TestDetectEnvironment_AnExportWithNoHarnessIsNotATier covers the caller who
// supplies half an invocation. A file with nobody asked to read it is not a
// measurement surface, and the reason has to say which half is missing.
func TestDetectEnvironment_AnExportWithNoHarnessIsNotATier(t *testing.T) {
	rep := DetectEnvironment(EnvironmentQuery{
		Home:       homesafe.SandboxHome(t),
		ExportFile: fixture("full_export.ndjson"),
	})

	if rep.Measurement.Tier != TierNone {
		t.Errorf("tier = %q, want %q", rep.Measurement.Tier, TierNone)
	}
	if !strings.Contains(rep.Measurement.Reason, "harness") {
		t.Errorf("reason = %q, want it to say that no harness was named", rep.Measurement.Reason)
	}
}

// --- The spool is an observation, never a capability --------------------------

// TestDetectEnvironment_ASpoolIsCountedAndClaimsNothing is the ruling this
// slice was handed, in the assertions.
//
// After the hook-spool slice there is no adapter that reads a spool, so the
// hook surface produces no `present` signal at all. `doctor` may report that a
// spool exists and how much is in it. It may not report that this machine can
// capture anything from Cursor, and the tier is where that claim would be made.
func TestDetectEnvironment_ASpoolIsCountedAndClaimsNothing(t *testing.T) {
	home := homesafe.SandboxHome(t)
	spool := filepath.Join(home, ".skill-architect", "spool")
	homesafe.MustBeOutside(t, spool)

	for _, ev := range []struct {
		ts      string
		payload string
	}{
		{"2026-09-10T01:00:00Z", `{"hook_event_name":"sessionStart","conversation_id":"c1"}`},
		{"2026-09-11T02:00:00Z", `{"hook_event_name":"stop","conversation_id":"c1"}`},
	} {
		line, err := NormalizeHookPayload([]byte(ev.payload), mustTime(t, ev.ts))
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		if err := AppendSpool(spool, line); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	rep := DetectEnvironment(EnvironmentQuery{Home: home})

	if !rep.Observed.Spool.Exists {
		t.Fatal("the spool was not reported at all")
	}
	if rep.Observed.Spool.Dir != spool {
		t.Errorf("spool dir = %q, want %q", rep.Observed.Spool.Dir, spool)
	}
	if rep.Observed.Spool.Files != 2 {
		t.Errorf("spool files = %d, want 2", rep.Observed.Spool.Files)
	}
	if rep.Observed.Spool.Lines != 2 {
		t.Errorf("spool lines = %d, want 2", rep.Observed.Spool.Lines)
	}
	if rep.Observed.Spool.LastCaptureAt != "2026-09-11T02:00:00Z" {
		t.Errorf("last_capture_at = %q, want the most recent capture time", rep.Observed.Spool.LastCaptureAt)
	}

	// And the claim it does not make.
	if rep.Measurement.Tier != TierNone {
		t.Errorf("tier = %q: a spool with lines in it raised the tier, and no adapter in this release reads one", rep.Measurement.Tier)
	}
	if rep.Observed.Spool.Yields == "" {
		t.Error("the report counts spool lines and says nothing about what they yield, which reads as a capability")
	}
	for _, want := range []string{"no adapter", "parsed later"} {
		if !strings.Contains(rep.Observed.Spool.Yields, want) {
			t.Errorf("spool yields = %q, want it to say %q", rep.Observed.Spool.Yields, want)
		}
	}
}

// TestSpoolYields_IsTheSentenceTheSpecQuotes holds the wording to the one place
// a reader looks it up.
//
// The sentence is the whole of what a spool is allowed to say about itself, and
// docs/profiler-spec.md quotes it verbatim under "The hook surface produces no
// signal in this release". A change to either without the other is a defect
// nobody notices — the code would say one thing and the document a user reads
// would say another, about exactly the distinction this release exists to keep.
// Copied from the spec rather than from the code, for the same reason
// fallbackReasonInSpec is.
func TestSpoolYields_IsTheSentenceTheSpecQuotes(t *testing.T) {
	// docs/profiler-spec.md, "The hook surface produces no signal in this
	// release, and the report says so".
	const quotedInSpec = "no measurement: no adapter in this build reads the spool, so these lines are " +
		"a capture to be parsed later and no profile, token count or tool call can be produced from them"

	if spoolYieldsNothingMeasured != quotedInSpec {
		t.Errorf("the spool says\n  %q\nand the spec quotes\n  %q", spoolYieldsNothingMeasured, quotedInSpec)
	}

	spec, err := os.ReadFile(filepath.Join("..", "docs", "profiler-spec.md"))
	if err != nil {
		t.Fatalf("read the spec: %v", err)
	}
	// The spec carries it as a wrapped block quote, so the comparison is over
	// the words rather than over the line breaks a markdown file chose.
	if !strings.Contains(unwrapped(string(spec)), quotedInSpec) {
		t.Errorf("docs/profiler-spec.md does not carry\n  %q\nso the sentence in the code is quoted nowhere a user reads", quotedInSpec)
	}
}

// unwrapped collapses a markdown block quote's line breaks and `>` markers into
// single spaces, so a sentence can be looked for as a sentence.
func unwrapped(text string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(text, "\n>", "\n")), " ")
}

// TestDetectEnvironment_ReportsRegisteredHookEventsAndNotAForeignOnes asks the
// same code the install asks.
//
// "Our hook is registered" is a true and useful observation, and the draft made
// it with a string match for "profiler ingest" — which is not the command
// `hooks install` writes (that is an absolute path plus `ingest || true`), and
// which would count a foreign hook that happened to mention us. The
// registration is matched by its exact command, through the same reader and the
// same entry-matching the install and uninstall use, so there is one definition
// of what our entry is.
func TestDetectEnvironment_ReportsRegisteredHookEventsAndNotAForeignOne(t *testing.T) {
	home := homesafe.SandboxHome(t)
	const ours = "/opt/profiler ingest || true"

	res, err := InstallHooks(home, ours)
	if err != nil {
		t.Fatalf("InstallHooks: %v", err)
	}
	if res.EventsRegistered == 0 {
		t.Fatal("nothing was registered, so this test proves nothing")
	}

	rep := DetectEnvironment(EnvironmentQuery{Home: home, HookCommand: ours})

	if !rep.Observed.HooksJSON.Present {
		t.Error("hooks.json was reported absent after an install wrote it")
	}
	if rep.Observed.HooksJSON.Path != filepath.Join(home, ".cursor", "hooks.json") {
		t.Errorf("hooks.json path = %q", rep.Observed.HooksJSON.Path)
	}
	if len(rep.Observed.HooksJSON.RegisteredEvents) != res.EventsRegistered {
		t.Errorf("registered_hook_events has %d entries, want the %d the install wrote",
			len(rep.Observed.HooksJSON.RegisteredEvents), res.EventsRegistered)
	}
	if !sortedStrings(rep.Observed.HooksJSON.RegisteredEvents) {
		t.Errorf("registered_hook_events = %v, which is not sorted — two runs would report the same file differently",
			rep.Observed.HooksJSON.RegisteredEvents)
	}

	// Registering it is still not a capability.
	if rep.Measurement.Tier != TierNone {
		t.Errorf("tier = %q: a registered hook raised the tier, and nothing reads what it captures", rep.Measurement.Tier)
	}

	// A different command is somebody else's entry, whatever it looks like.
	foreign := DetectEnvironment(EnvironmentQuery{Home: home, HookCommand: "/opt/profiler ingest"})
	if len(foreign.Observed.HooksJSON.RegisteredEvents) != 0 {
		t.Errorf("a command that is not ours matched %v entries; an entry is ours only if its command is exactly ours",
			foreign.Observed.HooksJSON.RegisteredEvents)
	}
}

// TestDetectEnvironment_SaysWhenItCouldNotReadWhatItLookedAt keeps the two
// failures apart. A hooks.json that cannot be parsed is not a machine with
// nothing registered, and a spool directory that cannot be read is not an empty
// one: reporting either as absent is the reader's own failure told to the user
// as a fact about the machine.
func TestDetectEnvironment_SaysWhenItCouldNotReadWhatItLookedAt(t *testing.T) {
	home := homesafe.SandboxHome(t)
	cursorDir := filepath.Join(home, ".cursor")
	if err := os.MkdirAll(cursorDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cursorDir, "hooks.json"), []byte("{ not json"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	rep := DetectEnvironment(EnvironmentQuery{Home: home, HookCommand: "x ingest"})

	if !rep.Observed.HooksJSON.Present {
		t.Error("a hooks.json that exists and cannot be parsed was reported as absent")
	}
	if rep.Observed.HooksJSON.Unreadable == "" {
		t.Error("a hooks.json that could not be parsed carries no reason, so a user reads it as 'nothing registered'")
	}
	if len(rep.Observed.HooksJSON.RegisteredEvents) != 0 {
		t.Error("events were reported out of a file that could not be parsed")
	}
}

// TestDetectEnvironment_NeverFails pins detection as a report rather than a
// gate. A machine nothing can be read from still gets an answer, because "this
// command failed" is not something a user can act on.
func TestDetectEnvironment_NeverFails(t *testing.T) {
	// A spool path that is a file rather than a directory: the read of it
	// fails, and the report still has to come back.
	home := homesafe.SandboxHome(t)
	notADir := filepath.Join(home, "spool-is-a-file")
	if err := os.WriteFile(notADir, []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	rep := DetectEnvironment(EnvironmentQuery{Home: home, SpoolDir: notADir})

	if rep.Observed.Spool.Unreadable == "" {
		t.Error("a spool that could not be read carries no reason")
	}
	if rep.Observed.Spool.Exists {
		t.Error("a path that is not a readable spool directory was reported as a spool")
	}
	if rep.Measurement.Tier != TierNone {
		t.Errorf("tier = %q, want %q", rep.Measurement.Tier, TierNone)
	}
	if rep.DetectedAt == "" {
		t.Error("the report is undated")
	}
}

// TestDetectEnvironment_DefaultsTheSpoolToTheOneTheWriterUses keeps the two
// from drifting: a doctor reporting on a different directory from the one
// `ingest` appends to is a doctor that says the capture is not running while it
// is.
func TestDetectEnvironment_DefaultsTheSpoolToTheOneTheWriterUses(t *testing.T) {
	home := homesafe.SandboxHome(t)

	rep := DetectEnvironment(EnvironmentQuery{Home: home})

	want := filepath.Join(home, ".skill-architect", "spool")
	if rep.Observed.Spool.Dir != want {
		t.Errorf("spool dir = %q, want %q", rep.Observed.Spool.Dir, want)
	}

	// And that default is the writer's own, taken relative to the home rather
	// than spelled again here.
	def, err := DefaultSpoolDir()
	if err != nil {
		t.Fatalf("DefaultSpoolDir: %v", err)
	}
	realHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir: %v", err)
	}
	if got := strings.TrimPrefix(rep.Observed.Spool.Dir, home); got != strings.TrimPrefix(def, realHome) {
		t.Errorf("doctor's spool is %q under its home and the writer's is %q under the real one", got, strings.TrimPrefix(def, realHome))
	}
}

// TestEnvironmentReport_SeparatesTheCapabilityFromTheObservation is the shape
// rule, asserted over the serialized document because that is what travels.
//
// The two halves are separate members with separate names: a reader of the JSON
// cannot mistake a count of spool files for a statement about what can be
// measured, and nothing under `observed` is a tier, a state or a source.
func TestEnvironmentReport_SeparatesTheCapabilityFromTheObservation(t *testing.T) {
	rep := DetectEnvironment(EnvironmentQuery{
		Home:       homesafe.SandboxHome(t),
		Harness:    "claude_code",
		ExportFile: fixture("full_export.ndjson"),
	})

	body, err := json.Marshal(rep)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"detected_at", "adapter_version", "measurement", "observed"} {
		if _, ok := doc[key]; !ok {
			t.Errorf("the report has no %q member:\n%s", key, body)
		}
	}
	if len(doc) != 4 {
		t.Errorf("the report has %d top-level members, want the four above — a fifth is a claim in neither half:\n%s", len(doc), body)
	}

	var observed map[string]json.RawMessage
	if err := json.Unmarshal(doc["observed"], &observed); err != nil {
		t.Fatalf("unmarshal observed: %v", err)
	}
	for _, forbidden := range []string{"tier", "state", "source", "signals"} {
		if _, ok := observed[forbidden]; ok {
			t.Errorf("the observed half carries %q, which is the vocabulary of a capability claim:\n%s", forbidden, doc["observed"])
		}
	}
}

// sortedStrings reports whether the slice is in non-decreasing order.
func sortedStrings(values []string) bool {
	for i := 1; i < len(values); i++ {
		if values[i-1] > values[i] {
			return false
		}
	}
	return true
}
