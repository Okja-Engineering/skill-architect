package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/Okja-Engineering/skill-architect/profiler"
)

// Usage:
//
//	profiler capture --harness claude_code --session <id> --snapshot <sha> --skill-dir <path> [--otel-file <path>]
//	profiler probe --harness claude_code [--otel-file <path>]
//	profiler compare --baseline <profile.json> --candidate <profile.json>
func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "probe":
		cmdProbe(os.Args[2:])
	case "capture":
		cmdCapture(os.Args[2:])
	case "compare":
		cmdCompare(os.Args[2:])
	case "version":
		fmt.Println("profiler " + profiler.AdapterVersion)
	case "-h", "--help", "help":
		// Help that was asked for is not a usage error. It used to reach
		// `default` and exit 1 with it, because both were the same path.
		usage()
	default:
		usage()
		os.Exit(1)
	}
}

// parseFlags parses a subcommand's flags, exiting 0 on a requested help and 1
// if they do not parse.
//
// ContinueOnError rather than ExitOnError, because ExitOnError exits 2 — the
// status capture uses for "this capture read nothing", which is the one a
// wrapping script is meant to act on. A mistyped flag and an unusable export
// behind the same number is a status nobody can branch on, so the CLI picks its
// own: 1 for anything the caller can fix by retyping the command.
//
// ContinueOnError also returns flag.ErrHelp for `-h` and `--help`, and having
// printed the flag defaults it was asked for. Reading that as "the flags do not
// parse" is what made every subcommand's help exit 1. The two errors are
// separated here rather than at each call site, because both subcommands ask
// the same question and one of them would eventually be the copy that forgot.
func parseFlags(fs *flag.FlagSet, args []string) {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		os.Exit(1)
	}
}

// harnessFlag registers --harness, whose description offers the set of
// harnesses the profiler ships. Both subcommands take the flag and it is
// defined once: a description naming a different set from the one
// profiler.NewAdapter accepts would advertise a harness the CLI cannot build.
func harnessFlag(fs *flag.FlagSet) *string {
	return fs.String("harness", "", "harness name ("+profiler.SupportedHarnesses()+")")
}

// unknownHarness reports a name no adapter answers to. It names the set from
// the registry rather than restating it, so the refusal and the help text
// cannot come to disagree about what is supported.
func unknownHarness(harness string) {
	fmt.Fprintf(os.Stderr, "unknown harness: %s (supported: %s)\n", harness, profiler.SupportedHarnesses())
	os.Exit(1)
}

func cmdProbe(args []string) {
	fs := flag.NewFlagSet("probe", flag.ContinueOnError)
	harness := harnessFlag(fs)
	otelFile := fs.String("otel-file", "", "path to OTel export file")
	parseFlags(fs, args)

	adapter, ok := profiler.NewAdapter(*harness, *otelFile)
	if !ok {
		unknownHarness(*harness)
	}

	cap, diags := probeWithDiagnostics(adapter)
	out, err := json.MarshalIndent(cap, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))

	// After the report, because the report is the answer and these explain it.
	// On stderr, because stdout is the report and a consumer parses it.
	//
	// The exit status stays 0. README documents no exit codes for `probe` at
	// all, so a script wrapping it today can only be relying on 0, and giving
	// `probe` an exit contract is new surface rather than a repair. Making the
	// failure audible is the repair; making it branchable is 0.5.0.
	for _, d := range diags {
		fmt.Fprintf(os.Stderr, "probe: %s\n", d)
	}
}

// probeWithDiagnostics probes an adapter and takes its diagnostics too when it
// has them.
//
// Probe is on the adapter interface and every adapter has it; explaining a
// "none" is optional, so an adapter that cannot still probes. The alternative —
// widening ProfilerAdapter — would make every future adapter implement an
// explanation before it could report a capability.
func probeWithDiagnostics(a profiler.ProfilerAdapter) (profiler.CapabilityReport, []string) {
	if d, ok := a.(profiler.ProbeDiagnoser); ok {
		return d.ProbeWithDiagnostics()
	}
	return a.Probe(), nil
}

func cmdCapture(args []string) {
	fs := flag.NewFlagSet("capture", flag.ContinueOnError)
	harness := harnessFlag(fs)
	sessionID := fs.String("session", "", "session ID")
	snapshotHash := fs.String("snapshot", "", "snapshot hash (git SHA or content hash)")
	skillDir := fs.String("skill-dir", "", "path to the skill being profiled")
	otelFile := fs.String("otel-file", "", "path to OTel export file")
	exportFile := fs.String("export-file", "", "path to a non-OTel session export (e.g. Devin ATIF); reserved — no shipped adapter reads it, and passing it is an error")
	parseFlags(fs, args)

	// The three flags a capture cannot be asked for without. This is the flag
	// layer's check, and it stays here rather than deferring to the adapter:
	// it runs before a harness is resolved (there may not be an adapter to ask)
	// and it names all three in one message, so a caller missing two of them
	// learns both at once instead of one per run.
	//
	// --session is also refused by the adapter itself, as
	// profiler.SessionIDRequiredError — it decides which records are read, so
	// an empty one is no assertion rather than a session that matched nothing,
	// and a library caller who never goes through this file is told too. The
	// two agree deliberately and this one is earlier; the adapter's is the
	// contract, and this is the usage message for it.
	if *sessionID == "" || *snapshotHash == "" || *skillDir == "" {
		fmt.Fprintln(os.Stderr, "required: --session, --snapshot, --skill-dir")
		os.Exit(1)
	}

	adapter, ok := profiler.NewAdapter(*harness, *otelFile)
	if !ok {
		unknownHarness(*harness)
	}

	if err := captureFlagError(*harness, *exportFile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	opts := profiler.CaptureOpts{
		SnapshotHash: *snapshotHash,
		SkillDir:     *skillDir,
	}

	profile, err := adapter.Capture(*sessionID, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "capture error: %v\n", err)
		os.Exit(1)
	}

	out, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
	os.Exit(captureExitCode(profile))
}

// captureExitCode is what a script wrapping `capture` branches on.
//
// 2 when the capture read nothing and something failed: the export was supplied
// and could not be used, so the profile carries reasons and no values. Exiting
// 0 there tells the wrapper the capture succeeded, and a row of zeros gets
// stored as a result.
//
// 0 otherwise — including a profile that is entirely unknown because no
// telemetry was configured. That is an answer about the session, not a failure
// of this run, and a caller who wants to insist on telemetry can read the
// states out of the profile.
//
// The profile is printed either way. When the status is non-zero, the reasons
// in it are the whole point.
func captureExitCode(p profiler.Profile) int {
	read, failed := 0, 0
	for _, state := range p.SignalStates() {
		switch state {
		case profiler.MetricPresent:
			read++
		case profiler.MetricError:
			failed++
		}
	}
	if failed > 0 && read == 0 {
		return 2
	}
	return 0
}

func cmdCompare(args []string) {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	baseline := fs.String("baseline", "", "path to the baseline profile JSON")
	candidate := fs.String("candidate", "", "path to the candidate profile JSON")
	parseFlags(fs, args)

	// Both, always. A comparison of one profile against nothing is not a
	// comparison, and defaulting either side would mean guessing which stored
	// profile the caller meant.
	if *baseline == "" || *candidate == "" {
		fmt.Fprintln(os.Stderr, "required: --baseline, --candidate")
		os.Exit(1)
	}

	// A profile that cannot be read is the caller naming the wrong path or a
	// document that is not a profile of this schema — something to retype, so
	// 1, the same status capture gives its own errors. It is not a comparison
	// that produced nothing, which is 2 and is the status a wrapper acts on.
	base, err := profiler.LoadProfile(*baseline)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compare error: %v\n", err)
		os.Exit(1)
	}
	cand, err := profiler.LoadProfile(*candidate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compare error: %v\n", err)
		os.Exit(1)
	}

	report := profiler.CompareProfiles(base, cand)
	out, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))

	// A pair-level refusal is the one thing a human must not have to parse the
	// report to learn: it means every number they came for is absent, and why.
	// On stderr, because stdout is the report and a consumer parses it.
	if report.Refusal != "" {
		fmt.Fprintf(os.Stderr, "compare: %s\n", report.Refusal)
	}
	os.Exit(compareExitCode(report))
}

// compareExitCode is what a script wrapping `compare` branches on.
//
// 2 when the run produced a report and nothing in it is comparable — the two
// profiles were read by different adapters, or they share no signal that both
// of them read. Exiting 0 there tells the wrapper a comparison happened, and an
// empty one gets stored as "no change", which is the fabrication `compare`
// exists to refuse.
//
// 0 when something was compared. The report is printed either way; when the
// status is non-zero, the refusal and the per-signal reasons in it are the
// whole point.
func compareExitCode(r profiler.ComparisonReport) int {
	if r.Comparable {
		return 0
	}
	return 2
}

// captureFlagError reports why the selected adapter cannot honour the flags it
// was given, or nil when it can.
//
// --export-file fills CaptureOpts.ExportFile, which only an adapter reading a
// session export (Devin's ATIF, a transcript) consults. No such adapter ships
// yet, so no selectable harness can honour the flag. Refuse it: a flag the
// adapter will ignore must fail loudly, not accept a path and produce an
// all-unknown profile that looks like missing telemetry.
//
// The refusal itself belongs to the adapter, which owns the contract and gives
// the same error to a library caller. This guard is the earlier, cheaper
// version of it, so the CLI exits before doing any work.
func captureFlagError(harness, exportFile string) error {
	if exportFile != "" {
		return profiler.ExportFileUnsupportedError(harness)
	}
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: profiler <command> [flags]

commands:
  probe     Probe environment and report capabilities
  capture   Capture a session profile
  compare   Compare two captured profiles
  version   Print version
  help      Print this help (also -h, --help)

examples:
  profiler probe --harness claude_code --otel-file ./otel-export.json
  profiler capture --harness claude_code --session abc123 --snapshot sha123 --skill-dir ./skills/my-skill --otel-file ./otel-export.json
  profiler compare --baseline ./before.json --candidate ./after.json`)
}
