package profiler

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// The half of the probe/capture contract that a capability report cannot carry.
//
// README: "capture delivers exactly what probe advertised, because both read
// the export through the same extractor." A capability's vocabulary is a source
// or "none", so a report has no way to say "the export you named could not be
// read" — it says none, which is also what it says when no telemetry was
// configured at all. capture keeps those apart: the first is an error state
// carrying a reason, and exit 2; the second is unknown, and exit 0. README's own
// warning for capture is that "exiting 0 after reading nothing is how a broken
// --otel-file path becomes a row of zeros".
//
// probe discarded the distinction, so `probe --otel-file ./typo.json` printed
// five nones and exited 0 — a report indistinguishable from a correctly
// configured session that happened to be silent. That is not a gap in the
// vocabulary, it is information the adapter held and threw away.
//
// These tests are driven off captureCases, the table that already drives the
// capture contract, so the denominator is every export shape in the suite
// rather than the one input that prompted the fix.

// TestProbeIsNotSilentWhereCaptureReportsAnError pins the pairing, not the
// wording: wherever capture reports an error for an input, probe must say
// something about it, and wherever capture does not, probe must stay quiet. A
// diagnostic on a healthy export would be its own defect.
func TestProbeIsNotSilentWhereCaptureReportsAnError(t *testing.T) {
	for _, tc := range captureCases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := adapterForCase(t, tc)

			_, diags := adapter.ProbeWithDiagnostics()
			wantDiagnostic := tc.absent == MetricError

			switch {
			case wantDiagnostic && len(diags) == 0:
				t.Errorf("capture reports an error for this input and probe said nothing; " +
					"a caller cannot tell a broken --otel-file from a silent session")
			case !wantDiagnostic && len(diags) != 0:
				t.Errorf("capture reports no error for this input, but probe said %q", diags)
			}
		})
	}
}

// TestProbeDiagnosticsAreCapturesOwnReasons keeps the two commands from drifting
// into two vocabularies for one failure. A message probe invents is worse than
// none: it is a second description of the same fault, free to go stale. Every
// diagnostic has to be a reason capture puts in the profile for the same input.
func TestProbeDiagnosticsAreCapturesOwnReasons(t *testing.T) {
	for _, tc := range captureCases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := adapterForCase(t, tc)

			_, diags := adapter.ProbeWithDiagnostics()
			profile, err := adapter.Capture("session-001", CaptureOpts{
				SnapshotHash: "abc123", SkillDir: "/skills/my-skill",
			})
			if err != nil {
				t.Fatal(err)
			}

			reasons := make(map[string]bool)
			for _, sig := range capturedSignals(profile) {
				if sig.raw.State == MetricError {
					reasons[sig.raw.Reason] = true
				}
			}

			for _, d := range diags {
				if strings.TrimSpace(d) == "" {
					t.Errorf("probe emitted an empty diagnostic")
					continue
				}
				if !reasons[d] {
					t.Errorf("probe diagnostic %q is not a reason capture gives for the same input", d)
				}
			}
		})
	}
}

// TestProbeAndProbeWithDiagnosticsAgreeOnTheReport is one half of the guard on
// the shape of the fix: the two entry points return the same report, so a
// caller that switched to the diagnosing one sees no change in what it parses.
//
// It is only that half, and it used to claim to be both. Comparing Probe()
// against ProbeWithDiagnostics() cannot notice a *new field* on
// CapabilityReport, because both entry points return the same struct and would
// both grow it — the suite stayed fully green when the report was widened,
// while the comment here said this held the diagnostics off the report. The
// claim was sound; the assertion was mislabelled. The claim is asserted by
// TestCapabilityReportCarriesOnlyTheKeysConsumersRead, and this test now says
// only what it checks.
func TestProbeAndProbeWithDiagnosticsAgreeOnTheReport(t *testing.T) {
	for _, tc := range captureCases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := adapterForCase(t, tc)

			plain := adapter.Probe()
			withDiags, _ := adapter.ProbeWithDiagnostics()

			if len(plain.Capabilities) != len(withDiags.Capabilities) {
				t.Fatalf("capability count differs: %d vs %d",
					len(plain.Capabilities), len(withDiags.Capabilities))
			}
			for metric, source := range plain.Capabilities {
				if got := withDiags.Capabilities[metric]; got != source {
					t.Errorf("%s: Probe says %q, ProbeWithDiagnostics says %q", metric, source, got)
				}
			}
			if plain.Harness != withDiags.Harness || plain.AdapterVer != withDiags.AdapterVer {
				t.Errorf("identity differs: %+v vs %+v", plain, withDiags)
			}
		})
	}
}

// TestCapabilityReportCarriesOnlyTheKeysConsumersRead is the other half, and
// the one with teeth: the report is embedded in every Profile, so a new key
// changes what a profile contains for the same input, which is an
// adapter-version and schema question and not a probe fix. The reason a
// capability came back "none" therefore rides a separate channel.
//
// The key set is read off the *type*, not only off a marshalled report, and
// that was measured rather than assumed: a `Diagnostics []string` field tagged
// `json:"diagnostics,omitempty"` leaves an empty probe's JSON byte-identical,
// so a check that only marshalled one report passed the exact widening it
// exists to refuse. The declared fields catch that. The marshalled keys are
// checked against the same set as well, because a `json` tag renamed without
// touching the Go field is a different key to a caller.
//
// Listed rather than counted, so one key swapped for another is caught too.
func TestCapabilityReportCarriesOnlyTheKeysConsumersRead(t *testing.T) {
	want := []string{"adapter_version", "capabilities", "harness", "probed_at"}

	declared := []string{}
	reportType := reflect.TypeOf(CapabilityReport{})
	for i := 0; i < reportType.NumField(); i++ {
		field := reportType.Field(i)
		if field.PkgPath != "" {
			continue // unexported, so never on the wire
		}
		key := strings.Split(field.Tag.Get("json"), ",")[0]
		switch key {
		case "-":
			continue
		case "":
			key = field.Name // no tag: the field name is the key
		}
		declared = append(declared, key)
	}
	sort.Strings(declared)

	if strings.Join(declared, ",") != strings.Join(want, ",") {
		t.Errorf("CapabilityReport declares keys %v, want %v\n"+
			"a key added or renamed here changes every profile that embeds this "+
			"report, so it belongs to a schema and adapter-version bump and not "+
			"to a probe change", declared, want)
	}

	raw, err := json.Marshal(ClaudeCodeAdapter{}.Probe())
	if err != nil {
		t.Fatal(err)
	}
	var keyed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keyed); err != nil {
		t.Fatal(err)
	}
	marshalled := make([]string, 0, len(keyed))
	for key := range keyed {
		marshalled = append(marshalled, key)
	}
	sort.Strings(marshalled)

	if strings.Join(marshalled, ",") != strings.Join(want, ",") {
		t.Errorf("a marshalled capability report carries keys %v, want %v",
			marshalled, want)
	}
}

// TestProbeNamesTheUnreadableExportPath is the specific user-facing failure:
// README's first profiler example, run against a path that is not there. The
// diagnostic has to name the path, because "something could not be read" does
// not get anyone to the typo.
func TestProbeNamesTheUnreadableExportPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "otel-export.json")
	adapter := ClaudeCodeAdapter{OtelExportFile: missing}

	_, diags := adapter.ProbeWithDiagnostics()
	if len(diags) == 0 {
		t.Fatal("probe reported nothing for an --otel-file that does not exist")
	}

	named := false
	for _, d := range diags {
		if strings.Contains(d, missing) {
			named = true
		}
	}
	if !named {
		t.Errorf("no diagnostic names the path that could not be read; got %q", diags)
	}
}

// TestProbeSaysNothingWhenNoExportWasSupplied separates "you gave me a file I
// could not read" from "you gave me no file". The second is an answer about the
// session and not a fault of the run — capture exits 0 for it — so probe must
// not manufacture a complaint, or every unconfigured probe becomes noise a
// caller learns to ignore.
func TestProbeSaysNothingWhenNoExportWasSupplied(t *testing.T) {
	_, diags := ClaudeCodeAdapter{}.ProbeWithDiagnostics()
	if len(diags) != 0 {
		t.Errorf("probe complained with no export file configured: %q", diags)
	}
}

// TestClaudeCodeAdapterIsAProbeDiagnoser pins the optional interface to the
// adapter, so an adapter that stops satisfying it fails here rather than
// silently falling back to the quiet path in cmd/main.go.
func TestClaudeCodeAdapterIsAProbeDiagnoser(t *testing.T) {
	var a interface{} = ClaudeCodeAdapter{}
	if _, ok := a.(ProbeDiagnoser); !ok {
		t.Error("ClaudeCodeAdapter no longer implements ProbeDiagnoser; probe would go quiet again")
	}
	if _, ok := a.(ProfilerAdapter); !ok {
		t.Error("ClaudeCodeAdapter no longer implements ProfilerAdapter")
	}
}

// adapterForCase builds the adapter a captureCase describes. The three shapes
// are the same ones TestCaptureDeliversEverySignalProbeAdvertises uses; sharing
// the construction is what keeps the two contracts asserted over identical
// inputs.
func adapterForCase(t *testing.T, tc captureCase) ClaudeCodeAdapter {
	t.Helper()
	switch {
	case tc.unconfigured:
		return ClaudeCodeAdapter{}
	case tc.missingFile:
		return ClaudeCodeAdapter{OtelExportFile: filepath.Join(t.TempDir(), "absent.json")}
	default:
		return ClaudeCodeAdapter{OtelExportFile: fixture(tc.fixture)}
	}
}
