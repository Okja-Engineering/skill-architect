package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Okja-Engineering/skill-architect/profiler"
)

// The invariant: a flag the selected adapter cannot honour fails loudly. It is
// never accepted and silently dropped, which turns a supplied export into a
// profile that looks like the user configured no telemetry at all.

func TestCaptureFlagError_RefusesAnExportFileNoAdapterReads(t *testing.T) {
	err := captureFlagError("claude_code", "/var/tmp/session.atif.json")
	if err == nil {
		t.Fatal("--export-file accepted by an adapter that cannot read it: the path would be silently ignored")
	}
	if !strings.Contains(err.Error(), "--otel-file") {
		t.Errorf("error = %q, want it to name the flag that does work", err)
	}
	if !strings.Contains(err.Error(), "claude_code") {
		t.Errorf("error = %q, want it to name the adapter that cannot honour the flag", err)
	}
}

func TestCaptureFlagError_AllowsTheOtelOnlyInvocation(t *testing.T) {
	if err := captureFlagError("claude_code", ""); err != nil {
		t.Fatalf("capture without --export-file rejected: %v", err)
	}
}

// --- Exit status ---
//
// The exit status is the only thing a script wrapping `profiler capture` can
// branch on. Exiting 0 after reading nothing at all tells that script the
// capture succeeded, and the all-unknown profile on stdout is stored as a
// result — which is how a broken `--otel-file` path becomes a row of zeros in
// somebody's comparison.

// buildProfiler builds the CLI the way a user does, because the exit status is
// a property of the process and not of any function inside it.
func buildProfiler(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "profiler")
	build := exec.Command("go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

func fixture(name string) string { return filepath.Join("..", "testdata", "otlp", name) }

func TestCapture_ExitStatusSaysWhetherAnythingWasRead(t *testing.T) {
	bin := buildProfiler(t)
	missing := filepath.Join(t.TempDir(), "absent.json")

	for _, tc := range []struct {
		name  string
		otel  string
		want  int
		about string
	}{
		{name: "an export that cannot be read at all", otel: fixture("malformed.json"), want: 2,
			about: "every signal is error; nothing was read"},
		{name: "an export file that is not there", otel: missing, want: 2,
			about: "every signal is error; nothing was read"},
		{name: "no export configured", want: 0,
			about: "nothing failed — the caller configured no telemetry"},
		{name: "an export carrying every signal", otel: fixture("full_export.ndjson"), want: 0,
			about: "three signals were read"},
		{name: "an export carrying no telemetry", otel: fixture("no_envelope.json"), want: 0,
			about: "nothing failed; the file simply is not an export"},
		{name: "an export carrying only tool calls", otel: fixture("tool_calls_only.json"), want: 0,
			about: "a signal was read, so the capture produced something"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"capture", "--harness", "claude_code", "--session", "s",
				"--snapshot", "h", "--skill-dir", "/skills/my-skill"}
			if tc.otel != "" {
				args = append(args, "--otel-file", tc.otel)
			}
			cmd := exec.Command(bin, args...)
			out, err := cmd.Output()
			got := cmd.ProcessState.ExitCode()
			if got != tc.want {
				t.Errorf("exit status = %d, want %d — %s (err %v)", got, tc.want, tc.about, err)
			}
			// Whatever the status, the profile is still written: a caller that
			// wants the reasons must be able to read them.
			var profile profiler.Profile
			if err := json.Unmarshal(out, &profile); err != nil {
				t.Fatalf("stdout is not a profile: %v\n%s", err, out)
			}
			if profile.Schema != profiler.ProfileSchema {
				t.Errorf("profile schema = %q, want %q", profile.Schema, profiler.ProfileSchema)
			}
		})
	}
}

// The rule is "no signal was read AND at least one failed", and the second half
// of it cannot be reached through the CLI. claude_code reads one export, so when
// that export cannot be used every OTel signal errors together and the other two
// are unknown: a capture that mixes a read signal with a failed one has never
// existed, which makes `failed > 0` indistinguishable from `failed > 0 && read
// == 0` in every end-to-end case above. The second adapter will mix them — that
// is what a second source is — so the mixed profile is pinned here, over a
// synthetic profile, where it can be reached at all.
func TestCaptureExitCode_TwoMeansNothingWasReadAndSomethingFailed(t *testing.T) {
	// In SignalStates order: tokens, tool calls, skill activation, timing,
	// attribution.
	profileWith := func(states ...profiler.MetricState) profiler.Profile {
		t.Helper()
		var p profiler.Profile
		p.Tokens.State = states[0]
		p.ToolCalls.State = states[1]
		p.SkillActivation.State = states[2]
		p.Timing.State = states[3]
		p.Attribution.State = states[4]
		if got := len(p.SignalStates()); got != len(states) {
			t.Fatalf("the profile carries %d signals and these cases set %d — a new signal must be set here too",
				got, len(states))
		}
		return p
	}
	const (
		present = profiler.MetricPresent
		unknown = profiler.MetricUnknown
		failed  = profiler.MetricError
	)
	for _, tc := range []struct {
		name  string
		p     profiler.Profile
		want  int
		about string
	}{
		{"every signal failed", profileWith(failed, failed, failed, failed, failed), 2,
			"nothing was read and the export was supplied"},
		{"the OTel signals failed and the rest are unknown", profileWith(failed, failed, unknown, failed, unknown), 2,
			"the shape a claude_code capture takes when its export cannot be used"},
		{"one signal read, one failed", profileWith(present, failed, unknown, unknown, unknown), 0,
			"the capture produced something, so a wrapper must not treat it as a dead run"},
		{"one signal read, the rest unknown", profileWith(unknown, present, unknown, unknown, unknown), 0,
			"a signal was read"},
		{"every signal read", profileWith(present, present, present, present, present), 0,
			"nothing failed"},
		{"every signal unknown", profileWith(unknown, unknown, unknown, unknown, unknown), 0,
			"no telemetry was configured — an answer about the session, not a failed run"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := captureExitCode(tc.p); got != tc.want {
				t.Errorf("captureExitCode = %d, want %d — %s", got, tc.want, tc.about)
			}
		})
	}
}

// Exit 2 has to mean one thing, or a script branching on it cannot act. The
// flag package exits 2 of its own accord on an unrecognised flag, which would
// put "you typed the flag wrong" and "the capture read nothing" behind the same
// status — and the second is the one a wrapper is supposed to retry or report.
func TestUsageErrorsExitOneSoThatTwoMeansOneThing(t *testing.T) {
	bin := buildProfiler(t)
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"no command", nil},
		{"an unknown command", []string{"frobnicate"}},
		{"an unknown flag on capture", []string{"capture", "--bogus-flag"}},
		{"an unknown flag on probe", []string{"probe", "--bogus-flag"}},
		{"an unknown harness", []string{"capture", "--harness", "nope",
			"--session", "s", "--snapshot", "h", "--skill-dir", "/d"}},
		{"a missing required flag", []string{"capture", "--harness", "claude_code"}},
		// --session decides which records are read, so an empty one is no
		// assertion rather than a session that matched nothing. Both this
		// layer and the adapter refuse it; whichever gets there first, it is a
		// usage error and never a profile.
		{"an empty --session", []string{"capture", "--harness", "claude_code",
			"--session", "", "--snapshot", "h", "--skill-dir", "/d"}},
		{"a flag no adapter reads", []string{"capture", "--harness", "claude_code",
			"--session", "s", "--snapshot", "h", "--skill-dir", "/d", "--export-file", "s.json"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(bin, tc.args...)
			out, _ := cmd.CombinedOutput()
			if got := cmd.ProcessState.ExitCode(); got != 1 {
				t.Errorf("exit status = %d, want 1 — a usage error is not a capture that read nothing\n%s", got, out)
			}
		})
	}
}
