package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
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
//
// The binary is built with -cover when GOCOVERDIR is set, which `go test
// -cover` sets for the test process and plain `go test` does not. Without it
// the coverage figure for this package is an artifact: every test here runs the
// CLI as a *subprocess*, and the tool counts only statements executed in the
// test process — so adding a subcommand exercised end to end makes the reported
// percentage fall. With it, the subprocess writes its counters into the same
// directory and `go test -cover ./cmd` reports what the suite actually reaches:
// 8.5% becomes 91.5%, measured at e26b2a5.
func buildProfiler(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "profiler")
	args := []string{"build", "-o", bin}
	if os.Getenv("GOCOVERDIR") != "" {
		args = append(args, "-cover", "-coverpkg=github.com/Okja-Engineering/skill-architect/profiler/cmd")
	}
	build := exec.Command("go", append(args, ".")...)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

func fixture(name string) string { return filepath.Join("..", "testdata", "otlp", name) }

// fixtureSession is the session.id every export under testdata/otlp carries.
// A capture reads only the records carrying the session it was asked for, so a
// case that means to read something has to ask for this one.
const fixtureSession = "00000000-0000-4000-8000-000000000001"

func TestCapture_ExitStatusSaysWhetherAnythingWasRead(t *testing.T) {
	bin := buildProfiler(t)
	missing := filepath.Join(t.TempDir(), "absent.json")

	for _, tc := range []struct {
		name    string
		otel    string
		session string
		want    int
		about   string
	}{
		{name: "an export that cannot be read at all", otel: fixture("malformed.json"), want: 2,
			about: "every signal is error; nothing was read"},
		{name: "an export file that is not there", otel: missing, want: 2,
			about: "every signal is error; nothing was read"},
		{name: "no export configured", want: 0,
			about: "nothing failed — the caller configured no telemetry"},
		{name: "an export carrying every signal", otel: fixture("full_export.ndjson"), session: fixtureSession, want: 0,
			about: "four signals were read"},
		// The same export under a session it does not carry. Nothing is read,
		// and nothing failed either: a session absent from an export is an
		// answer about that session.
		{name: "an export that carries no record of the session asked for", otel: fixture("full_export.ndjson"), want: 0,
			about: "the projection removed every record, so nothing was read and nothing failed"},
		{name: "an export carrying no telemetry", otel: fixture("no_envelope.json"), want: 0,
			about: "nothing failed; the file simply is not an export"},
		{name: "an export carrying only tool calls", otel: fixture("tool_calls_only.json"), session: fixtureSession, want: 0,
			about: "a signal was read, so the capture produced something"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			session := tc.session
			if session == "" {
				session = "a-session-this-export-does-not-carry"
			}
			args := []string{"capture", "--harness", "claude_code", "--session", session,
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
	// In profile field order: tokens, tool calls, skill activation, timing,
	// attribution, estimated context tokens.
	profileWith := func(states ...profiler.MetricState) profiler.Profile {
		t.Helper()
		var p profiler.Profile
		p.Tokens.State = states[0]
		p.ToolCalls.State = states[1]
		p.SkillActivation.State = states[2]
		p.Timing.State = states[3]
		p.Attribution.State = states[4]
		// The estimate is the one signal a profile can leave out entirely. It
		// is given a result here so that every state below is reachable for
		// it; the absent shape is a case of its own.
		p.EstimatedContextTokens = &profiler.EstimatedTokensResult{}
		p.EstimatedContextTokens.State = states[5]
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

	// What every claude_code capture actually looks like: the estimate was
	// never made, so the key is absent rather than carrying a state somebody
	// assigned. An absent estimate must count as unknown and not as a failure,
	// or a capture that read nothing would exit 2 for the wrong reason.
	noEstimate := profileWith(failed, failed, unknown, failed, unknown, unknown)
	noEstimate.EstimatedContextTokens = nil

	for _, tc := range []struct {
		name  string
		p     profiler.Profile
		want  int
		about string
	}{
		{"every signal failed", profileWith(failed, failed, failed, failed, failed, failed), 2,
			"nothing was read and the export was supplied"},
		{"the OTel signals failed and the rest are unknown", profileWith(failed, failed, unknown, failed, unknown, unknown), 2,
			"the shape a claude_code capture takes when its export cannot be used"},
		{"the OTel signals failed and no estimate was ever made", noEstimate, 2,
			"an estimate nobody made is unknown, not a signal that failed"},
		{"one signal read, one failed", profileWith(present, failed, unknown, unknown, unknown, unknown), 0,
			"the capture produced something, so a wrapper must not treat it as a dead run"},
		{"one signal read, the rest unknown", profileWith(unknown, present, unknown, unknown, unknown, unknown), 0,
			"a signal was read"},
		{"only the estimate was read", profileWith(unknown, unknown, unknown, unknown, unknown, present), 0,
			"an estimate is a reading — a capture that produced one is not a dead run"},
		{"every signal read", profileWith(present, present, present, present, present, present), 0,
			"nothing failed"},
		{"every signal unknown", profileWith(unknown, unknown, unknown, unknown, unknown, unknown), 0,
			"no telemetry was configured — an answer about the session, not a failed run"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := captureExitCode(tc.p); got != tc.want {
				t.Errorf("captureExitCode = %d, want %d — %s", got, tc.want, tc.about)
			}
		})
	}
}

// Help that was asked for is not a usage error.
//
// Every help path exited 1, because help and usage-error were the same path:
// the top-level switch sends an unrecognised first word to usage() and exits 1,
// and `-h` is an unrecognised first word; the subcommands hand their args to
// flag.Parse, which returns flag.ErrHelp for `-h`, and parseFlags treated any
// non-nil error as a typo.
//
// The two cases are not the same event. A usage error is the caller getting it
// wrong, and 1 is right for it. `--help` is the caller getting exactly what
// they asked for, and a program that exits nonzero having done what it was
// asked cannot be scripted against: `profiler --help` in a shell with `set -e`
// takes the script down, and a wrapper cannot tell "I printed the help" from
// "your flags are wrong".
//
// 0 disturbs neither of the other two statuses. 1 already means usage error and
// 2 already means the capture read nothing, and help is neither.
func TestRequestedHelpExitsZero(t *testing.T) {
	bin := buildProfiler(t)
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"-h on the command itself", []string{"-h"}},
		{"--help on the command itself", []string{"--help"}},
		{"-h on probe", []string{"probe", "-h"}},
		{"-h on capture", []string{"capture", "-h"}},
		{"-h on compare", []string{"compare", "-h"}},
		{"--help on probe", []string{"probe", "--help"}},
		{"--help on capture", []string{"capture", "--help"}},
		{"--help on compare", []string{"compare", "--help"}},
		{"help as a word", []string{"help"}},
		{"-h on experiment", []string{"experiment", "-h"}},
		{"--help on experiment", []string{"experiment", "--help"}},
		{"help as a word under experiment", []string{"experiment", "help"}},
		{"-h on experiment design", []string{"experiment", "design", "-h"}},
		{"-h on experiment plan", []string{"experiment", "plan", "-h"}},
		{"-h on experiment run", []string{"experiment", "run", "-h"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(bin, tc.args...)
			out, _ := cmd.CombinedOutput()
			if got := cmd.ProcessState.ExitCode(); got != 0 {
				t.Errorf("exit status = %d, want 0 — help was asked for and given\n%s", got, out)
			}
			// Exiting 0 having printed nothing would satisfy the status and
			// still not be help.
			if len(out) == 0 {
				t.Error("help exited 0 and printed nothing at all")
			}
			if !strings.Contains(string(out), "usage:") && !strings.Contains(string(out), "Usage of") {
				t.Errorf("help printed no usage text:\n%s", out)
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

	// compare reads two files, so the ways of getting it wrong include the
	// files themselves. A profile that cannot be read is the caller naming the
	// wrong path or an unreadable document — not a comparison that produced
	// nothing, which is what 2 means.
	dir := t.TempDir()
	valid := writeProfileFile(t, dir, "valid.json", storedProfile(profiler.AdapterVersion, "a", 1000, 10000))
	notAProfile := filepath.Join(dir, "not-a-profile.json")
	if err := os.WriteFile(notAProfile, []byte(`{"hello":"world"}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	absent := filepath.Join(dir, "absent.json")

	// experiment reads a design or a plan, so its ways of getting it wrong
	// include the documents themselves — and the two flags that name them are
	// alternatives, not a pair.
	//
	// Both documents are *valid*, deliberately: a case that names two files one
	// of which would be refused anyway proves nothing about the refusal being
	// tested, because the command would exit 1 on the document either way.
	designDoc := experimentDesign(t, dir,
		storedProfile(profiler.AdapterVersion, "old", 1000, 10000),
		storedProfile(profiler.AdapterVersion, "new", 500, 6000))
	design := writeJSONFile(t, dir, "design.json", designDoc)
	validPlan, err := profiler.GeneratePlan(designDoc)
	if err != nil {
		t.Fatalf("generate a valid plan: %v", err)
	}
	plan := writeJSONFile(t, dir, "plan.json", validPlan)

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
		{"compare with neither profile", []string{"compare"}},
		{"compare with only a baseline", []string{"compare", "--baseline", valid}},
		{"compare with only a candidate", []string{"compare", "--candidate", valid}},
		{"an unknown flag on compare", []string{"compare", "--bogus-flag"}},
		{"a baseline that is not there", []string{"compare", "--baseline", absent, "--candidate", valid}},
		{"a candidate that is not there", []string{"compare", "--baseline", valid, "--candidate", absent}},
		{"a baseline that is not a profile", []string{"compare", "--baseline", notAProfile, "--candidate", valid}},
		{"a candidate that is not a profile", []string{"compare", "--baseline", valid, "--candidate", notAProfile}},
		{"experiment with no subcommand", []string{"experiment"}},
		{"an unknown experiment subcommand", []string{"experiment", "frobnicate"}},
		{"experiment design with no file", []string{"experiment", "design"}},
		{"experiment plan with no file", []string{"experiment", "plan"}},
		{"experiment run with neither a design nor a plan", []string{"experiment", "run"}},
		// Two sources for one plan: whichever were preferred, the other was
		// silently ignored, and the document that ran is not the one the caller
		// thinks they named.
		{"experiment run with both a design and a plan", []string{"experiment", "run", "--design", design, "--plan", plan}},
		{"an unknown flag on experiment plan", []string{"experiment", "plan", "--bogus-flag"}},
		{"an unknown flag on experiment run", []string{"experiment", "run", "--bogus-flag"}},
		{"a design that is not there", []string{"experiment", "plan", "--file", absent}},
		{"a design that is not a design", []string{"experiment", "plan", "--file", notAProfile}},
		{"a plan that is not a plan", []string{"experiment", "run", "--plan", notAProfile}},
		{"a design given where a plan is wanted", []string{"experiment", "run", "--plan", design}},
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

// The invariant: `probe` and `capture` read one export through one extractor, so
// a caller must not be able to learn from `capture` that the export was
// unreadable and from `probe` that everything is merely "none". A capability
// cannot say "error" — the vocabulary is a source or none — so the run says it,
// on stderr, and this holds the pairing end to end at the CLI, which is the
// surface the defect was found on.
//
// The exit status is deliberately not part of the pairing: `probe` has no
// documented exit contract, so it stays 0 and is asserted to stay 0. Changing it
// is new surface for 0.5.0, and a test that demanded a nonzero status here would
// be pinning the fix to a decision nobody has taken.
func TestProbe_SaysOnStderrWhyACapabilityIsNone(t *testing.T) {
	bin := buildProfiler(t)
	missing := filepath.Join(t.TempDir(), "otel-export.json")

	for _, tc := range []struct {
		name      string
		otel      string
		wantNoise bool
		about     string
	}{
		{name: "an export file that is not there", otel: missing, wantNoise: true,
			about: "capture calls this an error and exits 2; probe must not be silent about it"},
		{name: "an export that cannot be read at all", otel: fixture("malformed.json"), wantNoise: true,
			about: "capture calls this an error; probe must say the same thing"},
		{name: "no export configured", wantNoise: false,
			about: "an answer about the session, not a fault of the run — capture exits 0"},
		{name: "an export carrying every signal", otel: fixture("full_export.ndjson"), wantNoise: false,
			about: "nothing failed, so a diagnostic would be noise"},
		{name: "a file that is not an export at all", otel: fixture("no_envelope.json"), wantNoise: false,
			about: "parsed fine and simply is not an export; capture exits 0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"probe", "--harness", "claude_code"}
			if tc.otel != "" {
				args = append(args, "--otel-file", tc.otel)
			}
			cmd := exec.Command(bin, args...)
			var stderr strings.Builder
			cmd.Stderr = &stderr
			out, _ := cmd.Output()

			if got := cmd.ProcessState.ExitCode(); got != 0 {
				t.Errorf("exit status = %d, want 0 — probe has no exit contract to break", got)
			}

			// stdout is the report, and it is the same report either way: the
			// diagnostic must not have widened what a caller parses.
			var report profiler.CapabilityReport
			if err := json.Unmarshal(out, &report); err != nil {
				t.Fatalf("stdout is not a capability report: %v\n%s", err, out)
			}
			// One capability per signal a profile carries, asked of the profile
			// rather than written down: the report is the denominator the
			// probe/capture agreement is walked over, and a literal here would
			// stop noticing a report that fell behind.
			if want := len(profiler.Profile{}.SignalStates()); len(report.Capabilities) != want {
				t.Errorf("report covers %d capabilities, want %d — one per signal the profile carries",
					len(report.Capabilities), want)
			}

			noise := strings.TrimSpace(stderr.String())
			switch {
			case tc.wantNoise && noise == "":
				t.Errorf("stderr is empty — %s", tc.about)
			case !tc.wantNoise && noise != "":
				t.Errorf("stderr says %q — %s", noise, tc.about)
			}
			if noise == "" {
				return
			}
			for _, line := range strings.Split(noise, "\n") {
				if !strings.HasPrefix(line, "probe: ") {
					t.Errorf("diagnostic %q is not attributed to probe", line)
				}
			}
			if tc.otel == missing && !strings.Contains(noise, missing) {
				t.Errorf("the diagnostic does not name the path that could not be read: %q", noise)
			}
		})
	}
}

// --- compare ---
//
// The comparison's own tests live beside it in package profiler. These are the
// command, because that is what a user runs and what a script stores the output
// of. A subcommand nothing exercises end to end is how the adapter-version
// refusal comes to be silently not enforced — and that refusal is the whole
// reason `compare` can be trusted with a profile somebody stored months ago.

// storedProfile is a profile as it comes back off disk: one somebody captured
// and kept. The adapter version is a parameter because it is the thing under
// test.
func storedProfile(adapterVersion, sessionID string, input, totalMs int) profiler.Profile {
	p := profiler.Profile{
		Schema:       profiler.ProfileSchema,
		ProfiledAt:   "2026-09-20T10:00:00Z",
		Harness:      "claude_code",
		SessionID:    sessionID,
		SnapshotHash: "sha-" + sessionID,
		SkillDir:     "/skills/my-skill",
		Capability: profiler.CapabilityReport{
			Harness:    "claude_code",
			AdapterVer: adapterVersion,
			ProbedAt:   "2026-09-20T10:00:00Z",
		},
		Tokens: profiler.PresentTokenResult(profiler.TokenCounts{
			Input:  profiler.Count(input),
			Output: profiler.Count(20),
		}, string(profiler.SourceOtel)),
		Timing: profiler.PresentTimingResult(profiler.TimingData{
			StartTime: "2026-09-20T10:00:00Z",
			EndTime:   "2026-09-20T10:00:10Z",
			TotalMs:   int64(totalMs),
		}, string(profiler.SourceOtel)),
		ToolCalls:       profiler.UnknownToolCallResult("no tool call events in export"),
		SkillActivation: profiler.UnknownActivationResult("no skill activation events in export"),
		Attribution:     profiler.UnknownAttributionResult("this harness does not attribute outputs to skills"),
	}
	return p
}

func writeJSONFile(t *testing.T, dir, name string, v any) string {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", name, err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func writeProfileFile(t *testing.T, dir, name string, p profiler.Profile) string {
	t.Helper()
	return writeJSONFile(t, dir, name, p)
}

// runCLI runs the command and returns everything a caller can observe: the two
// streams apart, because which of them a thing was said on is part of the
// contract, and the status, because it is what a script branches on.
func runCLI(t *testing.T, bin string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	_ = cmd.Run()
	return outBuf.String(), errBuf.String(), cmd.ProcessState.ExitCode()
}

// runCompare runs the command and returns everything a caller can observe.
// The report is parsed only when stdout holds one; a usage error prints no
// report at all, and that is itself part of the contract.
func runCompare(t *testing.T, bin string, args ...string) (report profiler.ComparisonReport, stdout, stderr string, code int) {
	t.Helper()
	stdout, stderr, code = runCLI(t, bin, append([]string{"compare"}, args...)...)
	if strings.HasPrefix(strings.TrimSpace(stdout), "{") {
		if err := json.Unmarshal([]byte(stdout), &report); err != nil {
			t.Fatalf("stdout is not a comparison report: %v\n%s", err, stdout)
		}
	}
	return report, stdout, stderr, code
}

// The failure this slice exists to prevent: a profile stored by an older
// adapter compared against a fresh one, reporting four releases of fixes to the
// *reader* as the skill's regression. Measured at a 50% apparent token drop on
// one fixture, where the entire difference was the reader getting more honest.
func TestCompare_RefusesAProfileStoredByAnotherAdapterVersion(t *testing.T) {
	bin := buildProfiler(t)
	dir := t.TempDir()

	// The numbers are the ones that would be subtracted: a 50% drop.
	stored := writeProfileFile(t, dir, "stored.json", storedProfile("0.4.1", "old", 1000, 10000))
	fresh := writeProfileFile(t, dir, "fresh.json", storedProfile(profiler.AdapterVersion, "new", 500, 6000))

	report, stdout, stderr, code := runCompare(t, bin, "--baseline", stored, "--candidate", fresh)

	if code != 2 {
		t.Errorf("exit status = %d, want 2 — the comparison produced no comparable answer\n%s", code, stderr)
	}
	if report.Comparable {
		t.Error("the report says it is comparable")
	}
	for _, want := range []string{"0.4.1", profiler.AdapterVersion} {
		if !strings.Contains(report.Refusal, want) {
			t.Errorf("refusal = %q, want it to name version %q", report.Refusal, want)
		}
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr = %q, want it to name version %q — a human running this must not have to parse the JSON", stderr, want)
		}
	}
	if len(report.Metrics) == 0 {
		t.Fatal("the report compared no metric at all, so this proves nothing")
	}
	for name, mc := range report.Metrics {
		if mc.Comparable || mc.Delta != nil {
			t.Errorf("%s: comparable=%v delta=%v — a refused comparison still handed over a number", name, mc.Comparable, mc.Delta)
		}
	}
	// The number that would have been reported must appear nowhere in the
	// output. -500 is the delta this pair would have produced.
	if strings.Contains(stdout, "-500") {
		t.Errorf("the refused delta is in the output anyway:\n%s", stdout)
	}

	// The control: the same two profiles at one adapter version do compare,
	// and do produce that delta. Without it, a `compare` that refused
	// everything would pass every assertion above.
	sameVersion := writeProfileFile(t, dir, "same.json", storedProfile(profiler.AdapterVersion, "old", 1000, 10000))
	control, controlOut, _, controlCode := runCompare(t, bin, "--baseline", sameVersion, "--candidate", fresh)
	if controlCode != 0 {
		t.Fatalf("exit status = %d, want 0 — the same pair at one adapter version is comparable", controlCode)
	}
	if !control.Comparable || control.Refusal != "" {
		t.Fatalf("comparable=%v refusal=%q — the refusal above proves nothing", control.Comparable, control.Refusal)
	}
	if !strings.Contains(controlOut, "-500") {
		t.Errorf("the comparable pair did not report the delta, so the refusal proves nothing:\n%s", controlOut)
	}
}

// The exit status is what a script branches on. 0 is a comparison that produced
// something; 2 is a run that produced a report and nothing comparable in it;
// 1 is the caller getting the command wrong. The report is on stdout in both
// of the first two, because the reasons are the point when nothing compared.
func TestCompare_ExitStatusSaysWhetherAnythingWasCompared(t *testing.T) {
	bin := buildProfiler(t)
	dir := t.TempDir()

	comparableA := writeProfileFile(t, dir, "a.json", storedProfile(profiler.AdapterVersion, "a", 1000, 10000))
	comparableB := writeProfileFile(t, dir, "b.json", storedProfile(profiler.AdapterVersion, "b", 500, 6000))
	otherVersion := writeProfileFile(t, dir, "old.json", storedProfile("0.4.1", "a", 1000, 10000))

	readNothing := storedProfile(profiler.AdapterVersion, "c", 0, 0)
	readNothing.Tokens = profiler.UnknownTokenResult("no OTel export configured")
	readNothing.Timing = profiler.UnknownTimingResult("no OTel export configured")
	nothing := writeProfileFile(t, dir, "nothing.json", readNothing)

	otherSource := storedProfile(profiler.AdapterVersion, "d", 500, 6000)
	otherSource.Tokens.Source = string(profiler.SourceSQLite)
	otherSource.Timing.Source = string(profiler.SourceSQLite)
	crossSource := writeProfileFile(t, dir, "sqlite.json", otherSource)

	for _, tc := range []struct {
		name                string
		baseline, candidate string
		want                int
		about               string
	}{
		{"two profiles read the same way", comparableA, comparableB, 0,
			"tokens and timing were read from the same source on both sides"},
		{"two adapter versions", comparableA, otherVersion, 2,
			"nothing is comparable: the difference may be the reader"},
		{"a profile that read nothing", comparableA, nothing, 2,
			"no signal is present on both sides, so there is nothing to subtract"},
		{"two different sources", comparableA, crossSource, 2,
			"every signal these two share was read from a different source"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report, stdout, _, code := runCompare(t, bin, "--baseline", tc.baseline, "--candidate", tc.candidate)
			if code != tc.want {
				t.Errorf("exit status = %d, want %d — %s", code, tc.want, tc.about)
			}
			// Whatever the status, the report is written: a caller that wants
			// the reasons must be able to read them.
			if report.Schema != profiler.ComparisonSchema {
				t.Errorf("stdout schema = %q, want %q\n%s", report.Schema, profiler.ComparisonSchema, stdout)
			}
			if (code == 0) != report.Comparable {
				t.Errorf("exit status %d disagrees with the report's own comparable=%v", code, report.Comparable)
			}
		})
	}
}

// Both sources reach the report a caller parses, on every metric and whatever
// the outcome — including the signals neither side read, which say "none".
func TestCompare_EveryMetricInTheReportNamesBothSources(t *testing.T) {
	bin := buildProfiler(t)
	dir := t.TempDir()
	a := writeProfileFile(t, dir, "a.json", storedProfile(profiler.AdapterVersion, "a", 1000, 10000))
	b := writeProfileFile(t, dir, "b.json", storedProfile(profiler.AdapterVersion, "b", 500, 6000))

	_, stdout, _, _ := runCompare(t, bin, "--baseline", a, "--candidate", b)

	var doc struct {
		Metrics map[string]map[string]json.RawMessage `json:"metrics"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("stdout is not a report: %v\n%s", err, stdout)
	}
	if len(doc.Metrics) == 0 {
		t.Fatal("the report carries no metrics, so this walk asserts nothing")
	}
	for name, fields := range doc.Metrics {
		for _, key := range []string{"baseline_source", "candidate_source"} {
			if _, ok := fields[key]; !ok {
				t.Errorf("metric %q has no %q in the report", name, key)
			}
		}
	}
}

// --- experiment ---
//
// `experiment run` is the command that *makes* the pairs `compare` reads. Its
// rules live beside the runner in package profiler and are tested there; these
// are the command — the three subcommands a user types, what they print, which
// stream they print it on, and the status a wrapping script branches on.

// experimentDesign builds a design whose two commands copy prepared profiles
// into place: the shape of a real experiment, with the capture replaced by
// something these tests can predict.
func experimentDesign(t *testing.T, dir string, baseline, candidate profiler.Profile) profiler.ExperimentDesign {
	t.Helper()
	basePath := writeProfileFile(t, dir, "baseline-fixture.json", baseline)
	candPath := writeProfileFile(t, dir, "candidate-fixture.json", candidate)
	condition := func(name string, p profiler.Profile, fixture string) profiler.Condition {
		return profiler.Condition{
			Name:         name,
			Harness:      p.Harness,
			SnapshotHash: p.SnapshotHash,
			SkillDir:     p.SkillDir,
			Command:      fmt.Sprintf("cp %q \"$PROFILE\"", fixture),
		}
	}
	return profiler.ExperimentDesign{
		Schema:       profiler.ExperimentSchema,
		Name:         "skill-rewrite-efficiency",
		TaskFamilies: []string{"audit"},
		Repetitions:  1,
		OutputDir:    filepath.Join(dir, "results"),
		Baseline:     condition("no-skill", baseline, basePath),
		Candidate:    condition("with-skill", candidate, candPath),
	}
}

// runExperiment runs a subcommand and parses the document it printed, when it
// printed one. A usage error prints no document, and that is part of the
// contract too.
func runExperiment(t *testing.T, bin string, args ...string) (result profiler.ExperimentResult, stdout, stderr string, code int) {
	t.Helper()
	stdout, stderr, code = runCLI(t, bin, append([]string{"experiment"}, args...)...)
	if strings.HasPrefix(strings.TrimSpace(stdout), "{") {
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("stdout is not a JSON document: %v\n%s", err, stdout)
		}
	}
	return result, stdout, stderr, code
}

// The failure this command could introduce that `compare` cannot: the two
// capture commands are the caller's, and nothing stops one of them being an
// older build of the profiler. An experiment that subtracted that pair would
// report the reader's changes as the skill's, with the authority of a document
// that says an experiment was run.
func TestExperimentRun_RefusesAPairReadByTwoAdapterVersions(t *testing.T) {
	bin := buildProfiler(t)
	dir := t.TempDir()
	design := experimentDesign(t, dir,
		storedProfile("0.4.1", "old", 1000, 10000),
		storedProfile(profiler.AdapterVersion, "new", 500, 6000))
	designFile := writeJSONFile(t, dir, "design.json", design)

	result, stdout, stderr, code := runExperiment(t, bin, "run", "--design", designFile)

	if code != 2 {
		t.Errorf("exit status = %d, want 2 — no run of this experiment compared anything\n%s", code, stderr)
	}
	if result.Comparable {
		t.Error("the result says the experiment is comparable")
	}
	if len(result.Runs) != 1 {
		t.Fatalf("runs = %d, want 1 — the run happened and its refusal is the answer", len(result.Runs))
	}
	for _, want := range []string{"0.4.1", profiler.AdapterVersion} {
		if !strings.Contains(result.Runs[0].Comparison.Refusal, want) {
			t.Errorf("refusal = %q, want it to name version %q", result.Runs[0].Comparison.Refusal, want)
		}
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr = %q, want it to name version %q — a human running this must not have to parse the JSON", stderr, want)
		}
	}
	// -500 is the delta this pair would have produced.
	if strings.Contains(stdout, "-500") {
		t.Errorf("the refused delta is in the result anyway:\n%s", stdout)
	}

	// The control: the same experiment with both profiles read by one adapter
	// version does compare, and does produce that delta. Without it, a `run`
	// that refused everything would pass every assertion above.
	controlDir := t.TempDir()
	control := experimentDesign(t, controlDir,
		storedProfile(profiler.AdapterVersion, "old", 1000, 10000),
		storedProfile(profiler.AdapterVersion, "new", 500, 6000))
	controlFile := writeJSONFile(t, controlDir, "design.json", control)
	got, controlOut, _, controlCode := runExperiment(t, bin, "run", "--design", controlFile)
	if controlCode != 0 {
		t.Fatalf("exit status = %d, want 0 — one adapter version, so the pair compares", controlCode)
	}
	if !got.Comparable {
		t.Fatal("the control experiment did not compare, so the refusal above proves nothing")
	}
	if !strings.Contains(controlOut, "-500") {
		t.Errorf("the control did not report the delta, so the refusal above proves nothing:\n%s", controlOut)
	}
}

// The exit status is what a script branches on, and it means the same three
// things `capture` and `compare` made it mean: 0 the command did what it says,
// 2 it ran and produced nothing comparable, 1 the caller or the setup is wrong.
func TestExperimentRun_ExitStatusSaysWhetherEveryRunCompared(t *testing.T) {
	bin := buildProfiler(t)

	comparable := func(t *testing.T) string {
		dir := t.TempDir()
		d := experimentDesign(t, dir,
			storedProfile(profiler.AdapterVersion, "old", 1000, 10000),
			storedProfile(profiler.AdapterVersion, "new", 500, 6000))
		return writeJSONFile(t, dir, "design.json", d)
	}
	refused := func(t *testing.T) string {
		dir := t.TempDir()
		d := experimentDesign(t, dir,
			storedProfile("0.4.1", "old", 1000, 10000),
			storedProfile(profiler.AdapterVersion, "new", 500, 6000))
		return writeJSONFile(t, dir, "design.json", d)
	}
	stepFails := func(t *testing.T) string {
		dir := t.TempDir()
		d := experimentDesign(t, dir,
			storedProfile(profiler.AdapterVersion, "old", 1000, 10000),
			storedProfile(profiler.AdapterVersion, "new", 500, 6000))
		d.Candidate.Command = `echo "$PROFILE" >/dev/null; exit 3`
		return writeJSONFile(t, dir, "design.json", d)
	}
	wroteNothing := func(t *testing.T) string {
		dir := t.TempDir()
		d := experimentDesign(t, dir,
			storedProfile(profiler.AdapterVersion, "old", 1000, 10000),
			storedProfile(profiler.AdapterVersion, "new", 500, 6000))
		d.Baseline.Command = `true # "$PROFILE"`
		return writeJSONFile(t, dir, "design.json", d)
	}

	for _, tc := range []struct {
		name   string
		design func(*testing.T) string
		want   int
		about  string
	}{
		{"an experiment whose every run compared", comparable, 0, "the pair was read the same way on both sides"},
		{"an experiment whose run was refused", refused, 2, "the comparison produced no number, and 2 is how a wrapper learns that"},
		{"a capture command that fails", stepFails, 1, "the setup is wrong; there is no result to report"},
		{"a capture command that writes no profile", wroteNothing, 1, "the step did not produce what the plan asked for"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, stdout, stderr, code := runExperiment(t, bin, "run", "--design", tc.design(t))
			if code != tc.want {
				t.Errorf("exit status = %d, want %d — %s\n%s", code, tc.want, tc.about, stderr)
			}
			if tc.want == 1 {
				// An experiment that could not be run has no result to print,
				// and printing an empty one would be a document saying nothing
				// happened rather than that something failed.
				if strings.TrimSpace(stdout) != "" {
					t.Errorf("a failed experiment printed a document on stdout:\n%s", stdout)
				}
				return
			}
			// 0 and 2 both produced a result, because when nothing compared the
			// reasons in it are the point.
			if result.Schema != profiler.ExperimentResultSchema {
				t.Errorf("stdout schema = %q, want %q\n%s", result.Schema, profiler.ExperimentResultSchema, stdout)
			}
			if (code == 0) != result.Comparable {
				t.Errorf("exit status %d disagrees with the result's own comparable=%v", code, result.Comparable)
			}
		})
	}
}

// stdout is the result document a wrapper parses, and the capture commands are
// the caller's — they print whatever they print. Their output must not end up
// inside the document.
func TestExperimentRun_KeepsWhatTheCaptureCommandsPrintOffStdout(t *testing.T) {
	bin := buildProfiler(t)
	dir := t.TempDir()
	design := experimentDesign(t, dir,
		storedProfile(profiler.AdapterVersion, "old", 1000, 10000),
		storedProfile(profiler.AdapterVersion, "new", 500, 6000))
	noise := "capture-command-chatter"
	design.Baseline.Command = fmt.Sprintf("echo %s; %s", noise, design.Baseline.Command)
	designFile := writeJSONFile(t, dir, "design.json", design)

	result, stdout, stderr, code := runExperiment(t, bin, "run", "--design", designFile)
	if code != 0 {
		t.Fatalf("exit status = %d, want 0\n%s", code, stderr)
	}
	// The word appears in the document legitimately, inside the recorded
	// command — a result has to say what it ran. What must not appear is the
	// echoed *line*, which is what the command printed.
	for _, line := range strings.Split(stdout, "\n") {
		if strings.TrimSpace(line) == noise {
			t.Errorf("the capture command's own output is on stdout, where the document is:\n%s", stdout)
		}
	}
	if !strings.Contains(stderr, noise) {
		t.Errorf("the capture command's output went nowhere; it must still be visible on stderr:\n%s", stderr)
	}
	if result.Schema != profiler.ExperimentResultSchema {
		t.Errorf("stdout is not a result document:\n%s", stdout)
	}
	// And the document is the whole of stdout: a line before it would leave a
	// wrapper unable to parse what it stored.
	if trimmed := strings.TrimSpace(stdout); !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		t.Errorf("stdout is not one JSON document:\n%s", stdout)
	}
}

// `design` and `plan` are the two steps before anything is executed: one says
// what the design means once the defaults are applied, the other says what will
// be run. Both are refusals-first, because refusing after the money is spent is
// not refusing.
func TestExperimentDesignAndPlan_AnswerBeforeAnythingIsExecuted(t *testing.T) {
	bin := buildProfiler(t)
	dir := t.TempDir()
	design := experimentDesign(t, dir,
		storedProfile(profiler.AdapterVersion, "old", 1000, 10000),
		storedProfile(profiler.AdapterVersion, "new", 500, 6000))
	design.Repetitions = 3
	design.Ordering = ""
	designFile := writeJSONFile(t, dir, "design.json", design)

	t.Run("design prints the design with its defaults applied", func(t *testing.T) {
		stdout, stderr, code := runCLI(t, bin, "experiment", "design", "--file", designFile)
		if code != 0 {
			t.Fatalf("exit status = %d, want 0\n%s", code, stderr)
		}
		var got profiler.ExperimentDesign
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("stdout is not a design: %v\n%s", err, stdout)
		}
		if got.Ordering != "blocked" || got.StoppingRule != "fixed" || got.AnalysisMethod != "difference" {
			t.Errorf("design = %+v, want the defaults filled in", got)
		}
	})

	t.Run("plan materializes one run per repetition", func(t *testing.T) {
		stdout, stderr, code := runCLI(t, bin, "experiment", "plan", "--file", designFile)
		if code != 0 {
			t.Fatalf("exit status = %d, want 0\n%s", code, stderr)
		}
		var plan profiler.ExperimentPlan
		if err := json.Unmarshal([]byte(stdout), &plan); err != nil {
			t.Fatalf("stdout is not a plan: %v\n%s", err, stdout)
		}
		if plan.Schema != profiler.ExperimentPlanSchema {
			t.Errorf("schema = %q, want %q", plan.Schema, profiler.ExperimentPlanSchema)
		}
		if len(plan.Runs) != 3 {
			t.Errorf("runs = %d, want 3", len(plan.Runs))
		}
		// Nothing was executed: `plan` is the step you read before you spend.
		for _, run := range plan.Runs {
			if _, err := os.Stat(run.Baseline.ProfilePath); err == nil {
				t.Errorf("plan wrote a profile at %q — it must execute nothing", run.Baseline.ProfilePath)
			}
		}
	})

	t.Run("a plan can be run later from the file", func(t *testing.T) {
		stdout, _, code := runCLI(t, bin, "experiment", "plan", "--file", designFile)
		if code != 0 {
			t.Fatalf("plan exit status = %d, want 0", code)
		}
		planFile := filepath.Join(dir, "plan.json")
		if err := os.WriteFile(planFile, []byte(stdout), 0o600); err != nil {
			t.Fatal(err)
		}
		result, _, stderr, code := runExperiment(t, bin, "run", "--plan", planFile)
		if code != 0 {
			t.Fatalf("run exit status = %d, want 0\n%s", code, stderr)
		}
		if len(result.Runs) != 3 {
			t.Errorf("runs = %d, want 3 — the plan on disk is what was executed", len(result.Runs))
		}
	})

	t.Run("a design the rules refuse is refused by both, and says why", func(t *testing.T) {
		refused := design
		refused.StoppingRule = "threshold"
		file := writeJSONFile(t, dir, "refused.json", refused)
		for _, sub := range []string{"design", "plan"} {
			stdout, stderr, code := runCLI(t, bin, "experiment", sub, "--file", file)
			if code != 1 {
				t.Errorf("%s exit status = %d, want 1", sub, code)
			}
			if !strings.Contains(stderr, "stopping_rule") {
				t.Errorf("%s said %q on stderr, want the reason", sub, stderr)
			}
			if strings.TrimSpace(stdout) != "" {
				t.Errorf("%s printed a document for a design it refused:\n%s", sub, stdout)
			}
		}
	})
}

// --- The help and the dispatcher cannot come to disagree ---
//
// Every slice from here to the release adds a subcommand. A command the
// dispatcher accepts and the help does not list is one nobody can find, and a
// command the help lists and the dispatcher rejects is one that does not exist.
// Both sets are derived — one from main.go's own switch, one from the help text
// the binary prints — so neither can be updated without the other.
func TestTheHelpListsEveryCommandTheDispatcherAccepts(t *testing.T) {
	dispatched := caseLiteralsIn(t, "main")
	if len(dispatched) == 0 {
		t.Fatal("no command was read out of main.go's dispatcher, so this check reads nothing")
	}

	bin := buildProfiler(t)
	cmd := exec.Command(bin, "help")
	helpText, _ := cmd.CombinedOutput()
	listed := commandsUnder(string(helpText), "commands:")
	if len(listed) == 0 {
		t.Fatalf("no command was read out of the help text, so this check reads nothing:\n%s", helpText)
	}

	if !reflect.DeepEqual(dispatched, listed) {
		t.Errorf("the dispatcher accepts %v and the help lists %v", dispatched, listed)
	}
	// The two flag spellings are aliases of `help` rather than commands of
	// their own, so they are excluded above — which is only honest if the help
	// still says they work.
	for _, alias := range []string{"-h", "--help"} {
		if !strings.Contains(string(helpText), alias) {
			t.Errorf("the help does not mention %q, which the dispatcher accepts", alias)
		}
	}
}

// A subcommand with subcommands of its own has the same two halves and the same
// way of coming apart: `experiment` dispatches three words and prints a list of
// three words, and nothing but this holds them to each other.
func TestTheExperimentHelpListsEverySubcommandItDispatches(t *testing.T) {
	dispatched := caseLiteralsIn(t, "cmdExperiment")
	if len(dispatched) == 0 {
		t.Fatal("no subcommand was read out of cmdExperiment's switch, so this check reads nothing")
	}

	bin := buildProfiler(t)
	helpText, _, code := runCLI(t, bin, "experiment", "help")
	if code != 0 {
		t.Errorf("`experiment help` exited %d, want 0 — help that was asked for is not an error", code)
	}
	listed := commandsUnder(helpText, "subcommands:")
	if len(listed) == 0 {
		t.Fatalf("no subcommand was read out of the help text, so this check reads nothing:\n%s", helpText)
	}
	if !reflect.DeepEqual(dispatched, listed) {
		t.Errorf("experiment dispatches %v and its help lists %v", dispatched, listed)
	}
}

// caseLiteralsIn reads the string literals of the switch inside one named
// function of main.go, rather than out of a list written down here that would
// go stale the same way the help text does.
//
// Scoped to one function, not to the file: every subcommand that dispatches
// subcommands of its own adds a switch, and a file-wide walk would read
// `experiment`'s three words as three top-level commands and fail the check
// above for a reason that is not a defect.
func caseLiteralsIn(t *testing.T, funcName string) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, 0)
	if err != nil {
		t.Fatalf("parse main.go: %v", err)
	}
	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if d, ok := decl.(*ast.FuncDecl); ok && d.Name.Name == funcName {
			fn = d
		}
	}
	if fn == nil {
		t.Fatalf("main.go declares no function %q, so this check reads nothing", funcName)
	}
	var names []string
	ast.Inspect(fn, func(n ast.Node) bool {
		clause, ok := n.(*ast.CaseClause)
		if !ok {
			return true
		}
		for _, expr := range clause.List {
			lit, ok := expr.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			name, err := strconv.Unquote(lit.Value)
			if err != nil || strings.HasPrefix(name, "-") {
				continue // the flag spellings of help are aliases, not commands
			}
			names = append(names, name)
		}
		return true
	})
	sort.Strings(names)
	return names
}

// commandsUnder reads the first word of every line under a header.
func commandsUnder(help, header string) []string {
	var names []string
	inBlock := false
	for _, line := range strings.Split(help, "\n") {
		if strings.HasPrefix(line, header) {
			inBlock = true
			continue
		}
		if !inBlock {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			break // the block ends at the first blank line
		}
		names = append(names, fields[0])
	}
	sort.Strings(names)
	return names
}
