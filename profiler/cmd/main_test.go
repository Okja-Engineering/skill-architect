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
