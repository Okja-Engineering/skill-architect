package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Okja-Engineering/skill-architect/profiler"
)

// Usage:
//
//	profiler capture --harness claude_code --session <id> --snapshot <sha> --skill-dir <path> [--otel-file <path>]
//	profiler probe --harness claude_code [--otel-file <path>]
//	profiler compare --baseline <profile.json> --candidate <profile.json>
//	profiler experiment {design,plan,run} …
//	profiler ingest [--spool-dir <dir>] [--strict]        (reads one hook payload on stdin)
//	profiler hooks {install,uninstall} [--home <dir>] [--command <cmd>]
//	profiler analyze [--spool-dir <dir>]
//	profiler doctor [--home <dir>] [--spool-dir <dir>] [--command <cmd>] [--harness <name>] [--otel-file <path>]
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
	case "experiment":
		cmdExperiment(os.Args[2:])
	case "ingest":
		cmdIngest(os.Args[2:])
	case "hooks":
		cmdHooks(os.Args[2:])
	case "analyze":
		cmdAnalyze(os.Args[2:])
	case "doctor":
		cmdDoctor(os.Args[2:])
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
	printJSON(cap)

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

	printJSON(profile)
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
	printJSON(report)

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
	return exitForComparable(r.Comparable)
}

// experimentExitCode is the same contract one level up: an experiment that ran
// and produced nothing comparable is a run a wrapper must not store as a
// result. It is 2 unless *every* run of the experiment compared something.
func experimentExitCode(r profiler.ExperimentResult) int {
	return exitForComparable(r.Comparable)
}

// exitForComparable is the one place the meaning of 2 lives. Two commands
// answer to it, and a third would be the one that drifted.
func exitForComparable(comparable bool) int {
	if comparable {
		return 0
	}
	return 2
}

// cmdExperiment dispatches the three steps of a paired experiment: say what the
// design means, say what will be run, run it.
//
// The help words are taken before the switch, so the switch holds the
// subcommands and nothing else — which is what the help/dispatcher derivation
// in the tests reads.
func cmdExperiment(args []string) {
	if len(args) == 0 {
		experimentUsage(os.Stderr)
		os.Exit(1)
	}
	if helpRequested(args[0]) {
		experimentUsage(os.Stdout)
		return
	}
	switch args[0] {
	case "design":
		cmdExperimentDesign(args[1:])
	case "plan":
		cmdExperimentPlan(args[1:])
	case "run":
		cmdExperimentRun(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown experiment subcommand: %s\n", args[0])
		experimentUsage(os.Stderr)
		os.Exit(1)
	}
}

func helpRequested(word string) bool {
	return word == "-h" || word == "--help" || word == "help"
}

// cmdExperimentDesign prints the design as the runner will read it, with the
// defaults filled in and the refusals applied. It is the step before anything
// is executed, which is the only point at which a refusal is free.
func cmdExperimentDesign(args []string) {
	fs := flag.NewFlagSet("experiment design", flag.ContinueOnError)
	file := fs.String("file", "", "path to the experiment design JSON")
	parseFlags(fs, args)
	if *file == "" {
		fmt.Fprintln(os.Stderr, "required: --file")
		os.Exit(1)
	}
	design, err := profiler.LoadExperimentDesign(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "experiment design error: %v\n", err)
		os.Exit(1)
	}
	printJSON(design)
}

// cmdExperimentPlan expands a design into the runs that would be executed, and
// executes none of them.
func cmdExperimentPlan(args []string) {
	fs := flag.NewFlagSet("experiment plan", flag.ContinueOnError)
	file := fs.String("file", "", "path to the experiment design JSON")
	parseFlags(fs, args)
	if *file == "" {
		fmt.Fprintln(os.Stderr, "required: --file")
		os.Exit(1)
	}
	design, err := profiler.LoadExperimentDesign(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "experiment plan error: %v\n", err)
		os.Exit(1)
	}
	plan, err := profiler.GeneratePlan(design)
	if err != nil {
		fmt.Fprintf(os.Stderr, "experiment plan error: %v\n", err)
		os.Exit(1)
	}
	printJSON(plan)
}

// cmdExperimentRun executes a plan and compares each pair it produced.
//
// A design or a plan, never both: they are two ways of saying which runs to
// execute, and accepting both would mean silently preferring one while the
// caller believes the other is what ran.
func cmdExperimentRun(args []string) {
	fs := flag.NewFlagSet("experiment run", flag.ContinueOnError)
	planFile := fs.String("plan", "", "path to a materialized experiment plan JSON")
	designFile := fs.String("design", "", "path to an experiment design JSON, planned and then run")
	parseFlags(fs, args)
	if (*planFile == "") == (*designFile == "") {
		fmt.Fprintln(os.Stderr, "required: exactly one of --plan or --design")
		os.Exit(1)
	}

	plan, err := loadOrGeneratePlan(*planFile, *designFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "experiment run error: %v\n", err)
		os.Exit(1)
	}

	// A step that failed, or that produced a profile the plan did not ask for,
	// leaves no result to print: the setup is wrong, and that is something the
	// caller fixes — 1, like every other error they can act on. It is not an
	// experiment that ran and compared nothing, which is 2.
	result, err := profiler.RunPlan(plan)
	if err != nil {
		fmt.Fprintf(os.Stderr, "experiment run error: %v\n", err)
		os.Exit(1)
	}

	printJSON(result)

	// The refusals, on stderr, one line per run that produced no number. A
	// human running an experiment must not have to parse the document to learn
	// that it measured nothing.
	for _, run := range result.Runs {
		if run.Comparison.Comparable {
			continue
		}
		reason := run.Comparison.Refusal
		if reason == "" {
			reason = "no signal was present in both profiles, so there was nothing to subtract"
		}
		fmt.Fprintf(os.Stderr, "experiment: %s r%d: %s\n", run.TaskFamily, run.Repetition, reason)
	}
	os.Exit(experimentExitCode(result))
}

// loadOrGeneratePlan takes the plan from whichever of the two documents the
// caller named. Exactly one of them is set; the flag layer has already refused
// the other three combinations.
func loadOrGeneratePlan(planFile, designFile string) (profiler.ExperimentPlan, error) {
	if planFile != "" {
		return profiler.LoadPlan(planFile)
	}
	design, err := profiler.LoadExperimentDesign(designFile)
	if err != nil {
		return profiler.ExperimentPlan{}, err
	}
	return profiler.GeneratePlan(design)
}

// --- ingest and hooks ---
//
// These two are the only subcommands that write outside the working directory,
// so they are the only two that need a home directory. Where it comes from is
// the same in both: the flag when the caller gave one, and otherwise the user's
// own, resolved here and passed in. The library never reads it — every function
// that writes takes the directory as a parameter — which is what keeps the one
// risky resolution in one place instead of one per call.

// cmdIngest reads one Cursor hook payload from stdin and appends it to the
// daily spool file.
//
// It says nothing on success. A hook runs inside the user's session, and a
// capture tool that echoes the payload back is one the user can see in the
// thing it is capturing.
func cmdIngest(args []string) {
	fs := flag.NewFlagSet("ingest", flag.ContinueOnError)
	spoolDir := fs.String("spool-dir", "", "spool directory (default: <home>/.skill-architect/spool)")
	strict := fs.Bool("strict", false, "replace prompt and tool content with its size, keeping only metadata")
	parseFlags(fs, args)

	dir := resolveSpoolDir("ingest", *spoolDir)

	if _, err := profiler.IngestMode(profiler.StdinSource{In: os.Stdin}, dir, time.Now(), *strict); err != nil {
		// Registered as `… ingest || true`, so Cursor will never look at this
		// status. A human running it by hand is the one who will, and a spool
		// that has been silently empty for a week is found a week late.
		fmt.Fprintf(os.Stderr, "ingest error: %v\n", err)
		os.Exit(1)
	}
}

// cmdHooks registers or removes the ingest command in Cursor's hooks.json.
//
// The help words are taken before the switch, so the switch holds the
// subcommands and nothing else — which is what the help/dispatcher derivation
// in the tests reads.
func cmdHooks(args []string) {
	if len(args) == 0 {
		hooksUsage(os.Stderr)
		os.Exit(1)
	}
	if helpRequested(args[0]) {
		hooksUsage(os.Stdout)
		return
	}
	switch args[0] {
	case "install":
		cmdHooksInstall(args[1:])
	case "uninstall":
		cmdHooksUninstall(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown hooks subcommand: %s\n", args[0])
		hooksUsage(os.Stderr)
		os.Exit(1)
	}
}

func cmdHooksInstall(args []string) {
	home, command := hooksFlags("install", args)
	res, err := profiler.InstallHooks(home, command)
	reportHooks("install", res, err)
}

func cmdHooksUninstall(args []string) {
	home, command := hooksFlags("uninstall", args)
	res, err := profiler.UninstallHooks(home, command)
	reportHooks("uninstall", res, err)
}

// hooksFlags resolves the two things both subcommands need. One function, so
// the pair cannot come to disagree about what --home defaults to — which is the
// resolution that decides whether a run writes into the user's real
// configuration.
func hooksFlags(sub string, args []string) (home, command string) {
	fs := flag.NewFlagSet("hooks "+sub, flag.ContinueOnError)
	homeFlag := fs.String("home", "", "home directory holding .cursor/hooks.json (default: the current user's)")
	commandFlag := fs.String("command", "", "the hook command to register (default: this binary's own `ingest`)")
	parseFlags(fs, args)

	return resolveHome("hooks "+sub, *homeFlag), resolveHookCommand("hooks "+sub, *commandFlag)
}

// resolveHome is the home a command operates on: the flag, or the current
// user's.
//
// One function, so no two commands can come to disagree about what "the user's
// own" means — which is the resolution that decides whether a run touches the
// real Cursor configuration. `doctor` reads that directory and `hooks` writes
// it, and they have to be looking at the same one or the report describes a
// machine nobody configured.
func resolveHome(command, flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s error: cannot resolve the home directory, pass --home: %v\n", command, err)
		os.Exit(1)
	}
	return home
}

// resolveHookCommand is the registration this binary writes, and therefore the
// one `doctor` looks for.
//
// An absolute path, because PATH inside a hook's environment is not something
// this tool gets to assume; `|| true` so a failure of ours can never take the
// user's session down with it. Both spellings have to be the same string or
// `doctor` reports that nothing is registered on a machine `hooks install` has
// just configured — so there is one of them.
func resolveHookCommand(command, flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s error: cannot resolve this binary's path, pass --command: %v\n", command, err)
		os.Exit(1)
	}
	return self + " ingest || true"
}

// resolveSpoolDir is the spool a command reads or writes: the flag, or the one
// under the current user's home that `ingest` appends to.
func resolveSpoolDir(command, flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	dir, err := profiler.DefaultSpoolDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s error: cannot resolve the home directory, pass --spool-dir: %v\n", command, err)
		os.Exit(1)
	}
	return dir
}

// cmdAnalyze summarises a spool.
//
// It prints the summary and nothing else. A spool directory that is not there
// is an error rather than an empty summary: a caller who cannot tell them apart
// reports "nothing was captured" for a capture that never ran, and that is the
// conclusion a user of this command is most likely to draw and least able to
// check.
func cmdAnalyze(args []string) {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	spoolDir := fs.String("spool-dir", "", "spool directory (default: <home>/.skill-architect/spool)")
	parseFlags(fs, args)

	summary, err := profiler.AnalyzeSpool(resolveSpoolDir("analyze", *spoolDir))
	if err != nil {
		fmt.Fprintf(os.Stderr, "analyze error: %v\n", err)
		os.Exit(1)
	}
	printJSON(summary)
}

// cmdDoctor reports what this machine can and cannot measure.
//
// It exits 0 whatever it finds. Detection is a report and not a gate: "nothing
// here measures anything" is the answer for most machines today, and a status
// that called it a failure would make the command unusable in the one situation
// it exists for. A usage error is still 1, because that is the caller's to fix.
//
// The measurement half only says something when an export is supplied, and that
// is the whole design: a capability is what a probe read, so the command has to
// be given something to read. Everything else it reports is an observation.
func cmdDoctor(args []string) {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	home := fs.String("home", "", "home directory holding .cursor/hooks.json and the spool (default: the current user's)")
	spoolDir := fs.String("spool-dir", "", "spool directory (default: <home>/.skill-architect/spool)")
	command := fs.String("command", "", "the registered hook command to look for (default: this binary's own `ingest`)")
	harness := harnessFlag(fs)
	otelFile := fs.String("otel-file", "", "an export to probe, which is the only way a measurement surface is reported")
	parseFlags(fs, args)

	printJSON(profiler.DetectEnvironment(profiler.EnvironmentQuery{
		Home:        resolveHome("doctor", *home),
		SpoolDir:    *spoolDir,
		HookCommand: resolveHookCommand("doctor", *command),
		Harness:     *harness,
		ExportFile:  *otelFile,
	}))
}

func reportHooks(sub string, res profiler.HookInstallResult, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "hooks %s error: %v\n", sub, err)
		os.Exit(1)
	}
	printJSON(res)
}

func hooksUsage(w *os.File) {
	fmt.Fprintln(w, `usage: profiler hooks <subcommand> [flags]

subcommands:
  install    Register this binary's `+"`ingest`"+` for every documented Cursor hook event
  uninstall  Remove the entries this binary registered, and only those

flags:
  --home <dir>     the directory holding .cursor/hooks.json (default: the current user's)
  --command <cmd>  the hook command to register or remove (default: this binary's own `+"`ingest`"+`)

The merge is additive and idempotent: hooks you or another tool registered are
preserved, the file is backed up before any write, and one this build cannot
parse is refused rather than replaced.

The file's location and format are taken from Cursor's published hooks
documentation and have not been checked against a running Cursor.`)
}

// printJSON writes the document this command exists to produce. Every
// subcommand prints one and only to stdout, so a shell redirect stores it and
// nothing else has to be offered to write a file.
func printJSON(doc any) {
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}

func experimentUsage(w *os.File) {
	fmt.Fprintln(w, `usage: profiler experiment <subcommand> [flags]

subcommands:
  design    Print a design with its defaults applied, or say why it is refused
  plan      Expand a design into the runs it would execute, and execute none
  run       Execute a plan and compare each pair it produces

flags:
  design --file <design.json>
  plan   --file <design.json>
  run    --plan <plan.json> | --design <design.json>

Each subcommand prints its document on stdout. `+"`run`"+` exits 2 when the
experiment ran and some run of it compared nothing.`)
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
  probe       Probe environment and report capabilities
  capture     Capture a session profile
  compare     Compare two captured profiles
  experiment  Design, plan or run a paired experiment
  hooks       Register or remove this binary's ingest in Cursor's hooks.json
  ingest      Append one Cursor hook payload, read on stdin, to the spool
  analyze     Summarise what a hook spool contains
  doctor      Report what this machine can and cannot measure
  version     Print version
  help        Print this help (also -h, --help)

examples:
  profiler probe --harness claude_code --otel-file ./otel-export.json
  profiler capture --harness claude_code --session abc123 --snapshot sha123 --skill-dir ./skills/my-skill --otel-file ./otel-export.json
  profiler compare --baseline ./before.json --candidate ./after.json
  profiler experiment plan --file ./design.json
  profiler experiment run --design ./design.json > ./result.json
  profiler hooks install
  echo '{"hook_event_name":"sessionStart"}' | profiler ingest
  profiler analyze
  profiler doctor --harness claude_code --otel-file ./otel-export.json`)
}
