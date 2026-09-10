package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/auraprix/skill-architect/profiler"
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
	fs.Parse(args)

	adapter := getAdapter(*harness, *otelFile, *exportFile)
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
	fs.Parse(args)

	if *sessionID == "" || *snapshotHash == "" || *skillDir == "" {
		fmt.Fprintln(os.Stderr, "required: --session, --snapshot, --skill-dir")
		os.Exit(1)
	}

	adapter := getAdapter(*harness, *otelFile, *exportFile)
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

func getAdapter(harness, otelFile, exportFile string) profiler.ProfilerAdapter {
	switch harness {
	case "claude_code":
		return profiler.ClaudeCodeAdapter{OtelExportFile: otelFile, ExportFile: exportFile}
	case "cursor":
		return profiler.CursorAdapter{OtelExportFile: otelFile, ExportFile: exportFile}
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
