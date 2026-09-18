package profiler

import (
	"path/filepath"
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

// TestProbeDiagnosticsDoNotChangeWhatProbeAdvertises is the guard on the shape
// of the fix. The capability report is embedded in every profile, so widening
// it would change what a profile contains and would belong to a schema and
// adapter-version bump. The diagnostics ride a separate channel, and this holds
// them there: the capabilities a caller parses are the same either way.
func TestProbeDiagnosticsDoNotChangeWhatProbeAdvertises(t *testing.T) {
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
