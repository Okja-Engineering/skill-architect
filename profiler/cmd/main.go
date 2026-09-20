package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Okja-Engineering/skill-architect/profiler"
)

// Usage:
//
//	profiler capture --harness claude_code --session <id> --snapshot <sha> --skill-dir <path> [--otel-file <path>] [--export-file <path>]
//	profiler probe --harness claude_code [--otel-file <path>]
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
	case "ingest":
		cmdIngest(os.Args[2:])
	case "doctor":
		cmdDoctor(os.Args[2:])
	case "hooks":
		cmdHooks(os.Args[2:])
	case "analyze":
		cmdAnalyze(os.Args[2:])
	case "experiment":
		cmdExperiment(os.Args[2:])
	case "version":
		fmt.Println("profiler " + profiler.AdapterVersion)
	default:
		usage()
		os.Exit(1)
	}
}

func cmdProbe(args []string) {
	fs := flag.NewFlagSet("probe", flag.ExitOnError)
	harness := fs.String("harness", "", "harness name (claude_code, cursor, codex, devin)")
	otelFile := fs.String("otel-file", "", "path to OTel export file")
	exportFile := fs.String("export-file", "", "path to JSONL session transcript")
	spoolDir := fs.String("spool-dir", "", "hook spool dir (cursor; default ~/.cursor-profiler/spool)")
	fs.Parse(args)

	adapter := getAdapter(*harness, *otelFile, *exportFile, *spoolDir)
	if adapter == nil {
		fmt.Fprintf(os.Stderr, "unknown harness: %s (supported: claude_code, cursor, codex, devin)\n", *harness)
		os.Exit(1)
	}

	cap := adapter.Probe()
	out, err := json.MarshalIndent(cap, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}

func cmdCapture(args []string) {
	fs := flag.NewFlagSet("capture", flag.ExitOnError)
	harness := fs.String("harness", "", "harness name (claude_code, cursor, codex, devin)")
	sessionID := fs.String("session", "", "session ID")
	snapshotHash := fs.String("snapshot", "", "snapshot hash (git SHA or content hash)")
	skillDir := fs.String("skill-dir", "", "path to the skill being profiled")
	otelFile := fs.String("otel-file", "", "path to OTel export file")
	exportFile := fs.String("export-file", "", "path to session export file (e.g. Devin ATIF)")
	spoolDir := fs.String("spool-dir", "", "hook spool dir (cursor; default ~/.cursor-profiler/spool)")
	fs.Parse(args)

	if *sessionID == "" || *snapshotHash == "" || *skillDir == "" {
		fmt.Fprintln(os.Stderr, "required: --session, --snapshot, --skill-dir")
		os.Exit(1)
	}

	adapter := getAdapter(*harness, *otelFile, *exportFile, *spoolDir)
	if adapter == nil {
		fmt.Fprintf(os.Stderr, "unknown harness: %s (supported: claude_code, cursor, codex, devin)\n", *harness)
		os.Exit(1)
	}

	opts := profiler.CaptureOpts{
		ExportFile:   *exportFile,
		SnapshotHash: *snapshotHash,
		SkillDir:     *skillDir,
	}
	// --otel-file takes precedence for OTel-based adapters; --export-file
	// is for non-OTel export formats (e.g. Devin ATIF in future adapters).
	if *otelFile != "" {
		opts.ExportFile = *otelFile
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
}

// cmdIngest reads one Cursor hook payload from stdin and appends it to the
// daily spool file. Registered in ~/.cursor/hooks.json as
// "profiler ingest || true" so a failure can never break the Cursor session.
func cmdIngest(args []string) {
	fs := flag.NewFlagSet("ingest", flag.ExitOnError)
	spoolDir := fs.String("spool-dir", "", "spool directory (default ~/.cursor-profiler/spool)")
	strict := fs.Bool("strict", os.Getenv("CURSOR_PROFILER_STRICT") != "", "strip prompt/tool content to size stubs (also: CURSOR_PROFILER_STRICT)")
	fs.Parse(args)

	dir := *spoolDir
	if dir == "" {
		var err error
		dir, err = profiler.DefaultSpoolDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot resolve home dir: %v\n", err)
			os.Exit(1)
		}
	}
	if _, err := profiler.IngestMode(profiler.StdinSource{In: os.Stdin}, dir, time.Now(), *strict); err != nil {
		fmt.Fprintf(os.Stderr, "ingest error: %v\n", err)
		os.Exit(1)
	}
	// Quiet on success: a hook must not echo payloads back.
}

// cmdDoctor reports which telemetry surfaces this machine can reach — the
// environment "Modernizr": Cursor installed, our hook registered, spool state,
// Admin API key, OTel config, and the resulting tier.
func cmdDoctor(args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	home := fs.String("home", "", "home directory to inspect (default: current user)")
	fs.Parse(args)

	h := *home
	if h == "" {
		var err error
		h, err = os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot resolve home dir: %v\n", err)
			os.Exit(1)
		}
	}
	rep := profiler.DetectEnvironment(h, os.Getenv)
	out, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}

// cmdHooks merges (or removes) our ingest command in ~/.cursor/hooks.json.
// The registered command defaults to this binary's absolute path so the hook
// does not depend on PATH inside Cursor's hook environment.
func cmdHooks(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: hooks {install,uninstall} [--command <cmd>] [--home <dir>]")
		os.Exit(1)
	}
	fs := flag.NewFlagSet("hooks "+args[0], flag.ExitOnError)
	home := fs.String("home", "", "home directory (default: current user)")
	command := fs.String("command", "", "hook command (default: <this-binary> ingest || true)")
	fs.Parse(args[1:])

	h := *home
	if h == "" {
		var err error
		h, err = os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot resolve home dir: %v\n", err)
			os.Exit(1)
		}
	}
	cmd := *command
	if cmd == "" {
		self, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot resolve binary path, pass --command: %v\n", err)
			os.Exit(1)
		}
		cmd = self + " ingest || true"
	}

	var res profiler.HookInstallResult
	var err error
	switch args[0] {
	case "install":
		res, err = profiler.InstallHooks(h, cmd)
	case "uninstall":
		res, err = profiler.UninstallHooks(h, cmd)
	default:
		fmt.Fprintf(os.Stderr, "unknown hooks command: %s\n", args[0])
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "hooks %s error: %v\n", args[0], err)
		os.Exit(1)
	}
	out, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(out))
}

// cmdAnalyze summarizes the spool: what ran, how often, which models, and the
// honest token picture. DuckDB remains the engine for ad-hoc questions.
func cmdAnalyze(args []string) {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	spoolDir := fs.String("spool-dir", "", "spool directory (default ~/.cursor-profiler/spool)")
	fs.Parse(args)

	dir := *spoolDir
	if dir == "" {
		var err error
		dir, err = profiler.DefaultSpoolDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot resolve home dir: %v\n", err)
			os.Exit(1)
		}
	}
	stats, err := profiler.AnalyzeSpool(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "analyze error: %v\n", err)
		os.Exit(1)
	}
	out, _ := json.MarshalIndent(stats, "", "  ")
	fmt.Println(string(out))
}

func cmdCompare(args []string) {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	baseline := fs.String("baseline", "", "path to baseline profile JSON")
	candidate := fs.String("candidate", "", "path to candidate profile JSON")
	output := fs.String("output", "", "optional path to write comparison JSON (default: stdout)")
	fs.Parse(args)

	if *baseline == "" || *candidate == "" {
		fmt.Fprintln(os.Stderr, "required: --baseline and --candidate")
		os.Exit(1)
	}

	baseProfile, err := profiler.LoadProfile(*baseline)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load baseline profile: %v\n", err)
		os.Exit(1)
	}

	candProfile, err := profiler.LoadProfile(*candidate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load candidate profile: %v\n", err)
		os.Exit(1)
	}

	report := profiler.CompareProfiles(baseProfile, candProfile)
	out, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
		os.Exit(1)
	}

	if *output != "" {
		if err := os.WriteFile(*output, out, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("comparison written to: %s\n", *output)
		return
	}
	fmt.Println(string(out))
}

func cmdExperiment(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: experiment {plan,run} --file <design.json> | --plan <plan.json>")
		os.Exit(1)
	}
	switch args[0] {
	case "plan":
		fs := flag.NewFlagSet("experiment plan", flag.ExitOnError)
		file := fs.String("file", "", "path to experiment design JSON")
		output := fs.String("output", "", "optional path to write plan JSON")
		fs.Parse(args[1:])
		if *file == "" {
			fmt.Fprintln(os.Stderr, "required: --file")
			os.Exit(1)
		}
		design, err := profiler.LoadExperimentDesign(*file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load design: %v\n", err)
			os.Exit(1)
		}
		plan, err := profiler.GeneratePlan(design)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to generate plan: %v\n", err)
			os.Exit(1)
		}
		out, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
			os.Exit(1)
		}
		if *output != "" {
			if err := os.WriteFile(*output, out, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "failed to write plan: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("plan written to: %s\n", *output)
			return
		}
		fmt.Println(string(out))
	case "run":
		fs := flag.NewFlagSet("experiment run", flag.ExitOnError)
		planFile := fs.String("plan", "", "path to experiment plan JSON")
		designFile := fs.String("design", "", "path to experiment design JSON (generates plan before running)")
		outputDir := fs.String("output-dir", "", "optional output directory for comparison reports")
		fs.Parse(args[1:])
		if *planFile == "" && *designFile == "" {
			fmt.Fprintln(os.Stderr, "required: --plan or --design")
			os.Exit(1)
		}
		var plan profiler.ExperimentPlan
		var err error
		if *planFile != "" {
			plan, err = profiler.LoadPlan(*planFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to load plan: %v\n", err)
				os.Exit(1)
			}
		} else {
			design, err := profiler.LoadExperimentDesign(*designFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to load design: %v\n", err)
				os.Exit(1)
			}
			plan, err = profiler.GeneratePlan(design)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to generate plan: %v\n", err)
				os.Exit(1)
			}
		}
		reports, err := profiler.RunPlan(plan, *outputDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "experiment run failed: %v\n", err)
			os.Exit(1)
		}
		out, err := json.MarshalIndent(reports, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(out))
	default:
		fmt.Fprintf(os.Stderr, "unknown experiment command: %s\n", args[0])
		os.Exit(1)
	}
}

func getAdapter(harness, otelFile, exportFile, spoolDir string) profiler.ProfilerAdapter {
	if spoolDir == "" && harness == "cursor" {
		if d, err := profiler.DefaultSpoolDir(); err == nil {
			spoolDir = d
		}
	}
	switch harness {
	case "claude_code":
		return profiler.ClaudeCodeAdapter{OtelExportFile: otelFile, ExportFile: exportFile}
	case "cursor":
		return profiler.CursorAdapter{OtelExportFile: otelFile, ExportFile: exportFile, SpoolDir: spoolDir}
	case "codex":
		return profiler.CodexAdapter{OtelExportFile: otelFile}
	case "devin":
		return profiler.DevinAdapter{ExportFile: exportFile}
	}
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: profiler <command> [flags]

commands:
  probe       Probe environment and report capabilities
  capture     Capture a session profile
  compare     Compare two captured profiles
  ingest      Append one Cursor hook payload (stdin) to the spool
  doctor      Report reachable telemetry surfaces and the selected tier
  hooks       Install or remove our ingest hook in ~/.cursor/hooks.json
  analyze     Summarize the spool: events, tools, models, token picture
  experiment  Plan or run a paired experiment
  version     Print version

examples:
  profiler probe --harness claude_code --otel-file ./otel-export.json
  profiler probe --harness cursor --otel-file ./otel-export.json --export-file ./state.vscdb
  profiler capture --harness claude_code --session abc123 --snapshot sha123 --skill-dir ./skills/my-skill --otel-file ./otel-export.json
  profiler compare --baseline profile-baseline.json --candidate profile-candidate.json
  profiler experiment plan --file design.json
  profiler experiment run --plan plan.json --output-dir ./results`)
}
