package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/Okja-Engineering/skill-architect/skillgate"
)

// The gate command is exercised as a subprocess so the assertions can be made
// against the real entry point and the real exit status. os.Exit is the
// command's contract (0 = APPROVE/CAUTION, 1 = REJECT, 2 = the gate failed),
// and it cannot be observed in-process.
const subprocessEnv = "SKILLGATE_TEST_SUBPROCESS"

func TestMain(m *testing.M) {
	if os.Getenv(subprocessEnv) == "1" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// runCLI re-executes the test binary as `skillgate <args...>` and returns what
// a shell would see.
func runCLI(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), subprocessEnv+"=1")
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	err := cmd.Run()
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("running %v: %v", args, err)
		}
		code = ee.ExitCode()
	}
	return so.String(), se.String(), code
}

// skipList force-skips the three optional external scanners so a run depends
// only on the deterministic floor, never on what is installed on the machine.
const skipList = "skillspector,agnix,skill-validator"

// hermeticSkips is skipList in flag form.
var hermeticSkips = []string{"--skip-checks", skipList}

// cleanBundle is a minimal well-formed skill: no findings, so the verdict is
// CAUTION (capped by the skipped scanners) and the exit status is 0.
func cleanBundle(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	body := "---\nname: demo\ndescription: a demo skill\n---\nUse it.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// generatedAt is the one field that legitimately differs between two runs.
var generatedAt = regexp.MustCompile(`"generated_at": "[^"]*"`)

func maskClock(b []byte) []byte {
	return generatedAt.ReplaceAll(b, []byte(`"generated_at": "MASKED"`))
}

func readReport(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"generated_at"`)) {
		t.Fatalf("%s carries no generated_at — the clock mask would be vacuous", path)
	}
	return maskClock(b)
}

// TestArgumentOrderDoesNotChangeParse asserts the parse outcome — flag values
// and operands — not the permuted slice, so the invariant survives any later
// change of mechanism. The flag set comes from the command's own registration,
// so a flag added to gateFlags is covered here without a second list.
func TestArgumentOrderDoesNotChangeParse(t *testing.T) {
	for _, tc := range []struct {
		name     string
		args     []string
		want     gateFlags
		operands []string
		wantErr  bool
	}{
		{
			name:     "flags before the target",
			args:     []string{"-o", "r.json", "--format", "sarif", "dir"},
			want:     gateFlags{out: "r.json", format: "sarif"},
			operands: []string{"dir"},
		},
		{
			name:     "flags after the target — the published form",
			args:     []string{"dir", "-o", "r.json", "--format", "sarif"},
			want:     gateFlags{out: "r.json", format: "sarif"},
			operands: []string{"dir"},
		},
		{
			name:     "interleaved",
			args:     []string{"--format", "sarif", "dir", "-o", "r.json"},
			want:     gateFlags{out: "r.json", format: "sarif"},
			operands: []string{"dir"},
		},
		{
			name:     "equals form after the target",
			args:     []string{"dir", "-o=r.json", "--format=sarif"},
			want:     gateFlags{out: "r.json", format: "sarif"},
			operands: []string{"dir"},
		},
		{
			name:     "boolean flag does not swallow the target that follows it",
			args:     []string{"--fail-on-incomplete", "dir"},
			want:     gateFlags{failInc: true, format: "json"},
			operands: []string{"dir"},
		},
		{
			name:     "boolean flag after the target",
			args:     []string{"dir", "--fail-on-incomplete"},
			want:     gateFlags{failInc: true, format: "json"},
			operands: []string{"dir"},
		},
		{
			name:     "-- ends flag parsing: what follows is an operand, dash or not",
			args:     []string{"-o", "r.json", "--", "-weird-dir"},
			want:     gateFlags{out: "r.json", format: "json"},
			operands: []string{"-weird-dir"},
		},
		{
			name:     "a flag value that looks like a flag is still a value",
			args:     []string{"dir", "--baseline", "--format"},
			want:     gateFlags{baseline: "--format", format: "json"},
			operands: []string{"dir"},
		},
		{
			name:     "operand order is preserved, so two targets still refuse",
			args:     []string{"a", "-o", "r.json", "b"},
			want:     gateFlags{out: "r.json", format: "json"},
			operands: []string{"a", "b"},
		},
		{
			name:    "an undefined flag before the target is refused",
			args:    []string{"--nope", "dir"},
			wantErr: true,
		},
		{
			name:    "an undefined flag after the target is refused, not absorbed",
			args:    []string{"dir", "--nope"},
			wantErr: true,
		},
		{
			name:    "a flag left without its value is refused, before the target",
			args:    []string{"-o"},
			wantErr: true,
		},
		{
			name:    "a flag left without its value is refused, after the target",
			args:    []string{"dir", "-o"},
			wantErr: true,
		},
		{
			// What flag.Parse does with `-baseline -- dir` today: an option
			// argument is taken literally, so `--` is a value here and not a
			// separator. Permuting must not change that reading.
			name:     "the separator in value position is a value, as it is with flags first",
			args:     []string{"dir", "--baseline", "--"},
			want:     gateFlags{baseline: "--", format: "json"},
			operands: []string{"dir"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := flag.NewFlagSet("gate", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			var got gateFlags
			got.register(fs)

			// The two steps cmdGate takes. Which of them refuses a
			// malformed invocation is an implementation detail; that it
			// is refused is the contract.
			permuted, err := permuteArgs(fs, tc.args)
			if err == nil {
				err = fs.Parse(permuted)
			}
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parse succeeded, want refusal (operands %v)", fs.Args())
				}
				return
			}
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got != tc.want {
				t.Errorf("flags = %+v, want %+v", got, tc.want)
			}
			if !slices.Equal(fs.Args(), tc.operands) {
				t.Errorf("operands = %v, want %v", fs.Args(), tc.operands)
			}
		})
	}
}

// TestDocumentedInvocationSucceeds drives the form published by
// skills/skill-gate/SKILL.md, docs/skillgate-spec.md and this command's own doc
// comment: the flags follow the target directory.
func TestDocumentedInvocationSucceeds(t *testing.T) {
	dir := cleanBundle(t)
	out := filepath.Join(t.TempDir(), "report.json")

	args := append([]string{"gate", dir, "-o", out}, hermeticSkips...)
	_, stderr, code := runCLI(t, args...)
	if code != skillgate.ExitPass {
		t.Fatalf("exit = %d, want %d\nstderr: %s", code, skillgate.ExitPass, stderr)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("documented invocation wrote no report: %v", err)
	}
}

// TestFlagOrderIsImmaterial is the invariant: where a flag sits relative to the
// target changes nothing at all. Not merely "both orders run" — one report, one
// code path, byte-identical output modulo the wall clock.
func TestFlagOrderIsImmaterial(t *testing.T) {
	dir := cleanBundle(t)
	tmp := t.TempDir()

	orders := []struct {
		name string
		args func(out string) []string
	}{
		{"flags-before", func(out string) []string {
			return []string{"gate", "--skip-checks", skipList, "-o", out, dir}
		}},
		{"flags-after", func(out string) []string {
			return []string{"gate", dir, "-o", out, "--skip-checks", skipList}
		}},
		{"interleaved", func(out string) []string {
			return []string{"gate", "--format", "json", dir, "-o", out, "--skip-checks", skipList}
		}},
		{"separated", func(out string) []string {
			return []string{"gate", "--skip-checks", skipList, "-o", out, "--", dir}
		}},
		{"equals-form-after", func(out string) []string {
			return []string{"gate", dir, "-o=" + out, "--skip-checks=" + skipList}
		}},
	}

	var want []byte
	var wantName string
	for _, order := range orders {
		out := filepath.Join(tmp, order.name+".json")
		args := order.args(out)
		_, stderr, code := runCLI(t, args...)
		if code != skillgate.ExitPass {
			t.Fatalf("%s: exit = %d, want %d\nstderr: %s", order.name, code, skillgate.ExitPass, stderr)
		}
		got := readReport(t, out)
		if want == nil {
			want, wantName = got, order.name
			continue
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s produced a different report from %s (%d vs %d bytes) — argument order must not select a code path",
				order.name, wantName, len(got), len(want))
		}
	}
}

// TestBoolFlagMeansTheSameInEitherPosition guards the hazard specific to
// permuting arguments: a boolean flag takes no value, so a target that follows
// one must stay a target — and the flag's effect (here: exit 1 because the
// three skipped scanners are incomplete coverage) must not shift with position.
func TestBoolFlagMeansTheSameInEitherPosition(t *testing.T) {
	dir := cleanBundle(t)
	tmp := t.TempDir()

	before := filepath.Join(tmp, "before.json")
	after := filepath.Join(tmp, "after.json")
	cases := []struct {
		name string
		args []string
		out  string
	}{
		{"before the target",
			[]string{"gate", "--skip-checks", skipList, "--fail-on-incomplete", "-o", before, dir}, before},
		{"after the target",
			[]string{"gate", dir, "-o", after, "--fail-on-incomplete", "--skip-checks", skipList}, after},
	}
	var want []byte
	for _, tc := range cases {
		_, stderr, code := runCLI(t, tc.args...)
		if code != skillgate.ExitGate {
			t.Fatalf("%s: exit = %d, want %d (incomplete coverage under --fail-on-incomplete)\nstderr: %s",
				tc.name, code, skillgate.ExitGate, stderr)
		}
		got := readReport(t, tc.out)
		if want == nil {
			want = got
			continue
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s produced a different report from the flags-before order", tc.name)
		}
	}
}

// TestTwoPositionalsStillRefuse pins the refusal that accepting flags after the
// target must not weaken.
func TestTwoPositionalsStillRefuse(t *testing.T) {
	dir := cleanBundle(t)
	_, stderr, code := runCLI(t, "gate", dir, dir)
	if code != skillgate.ExitExecError {
		t.Fatalf("exit = %d, want %d", code, skillgate.ExitExecError)
	}
	if want := "gate: exactly one target directory required\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}

// TestDanglingFlagRefusesInEitherPosition: `-o` with nothing after it is
// malformed wherever it sits. Accepting flags after the target must not turn a
// refusal into a report written to a file named after a separator.
func TestDanglingFlagRefusesInEitherPosition(t *testing.T) {
	dir := cleanBundle(t)
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"no target at all", []string{"gate", "-o"}},
		{"after the target", []string{"gate", dir, "-o"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, stderr, code := runCLI(t, tc.args...)
			if code != skillgate.ExitExecError {
				t.Fatalf("exit = %d, want %d\nstderr: %s", code, skillgate.ExitExecError, stderr)
			}
			if !bytes.Contains([]byte(stderr), []byte("flag needs an argument: -o")) {
				t.Fatalf("stderr = %q, want the missing-value refusal", stderr)
			}
		})
	}
}

// TestUnknownFlagRefusesInEitherPosition: an undefined flag is an undefined
// flag wherever it sits. It must not become a positional and must not be
// silently ignored.
func TestUnknownFlagRefusesInEitherPosition(t *testing.T) {
	dir := cleanBundle(t)
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"before the target", []string{"gate", "--nope", dir}},
		{"after the target", []string{"gate", dir, "--nope"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, stderr, code := runCLI(t, tc.args...)
			if code != skillgate.ExitExecError {
				t.Fatalf("exit = %d, want %d", code, skillgate.ExitExecError)
			}
			if !bytes.Contains([]byte(stderr), []byte("flag provided but not defined: -nope")) {
				t.Fatalf("stderr = %q, want the undefined-flag refusal", stderr)
			}
		})
	}
}
