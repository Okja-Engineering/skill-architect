package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/Okja-Engineering/skill-architect/profiler"
)

// Usage:
//
//	profiler capture --harness claude_code --session <id> --snapshot <sha> --skill-dir <path> [--otel-file <path>]
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
	case "version":
		fmt.Println("profiler " + profiler.AdapterVersion)
	default:
		usage()
		os.Exit(1)
	}
}

func cmdProbe(args []string) {
	fs := flag.NewFlagSet("probe", flag.ExitOnError)
	harness := fs.String("harness", "", "harness name (claude_code)")
	otelFile := fs.String("otel-file", "", "path to OTel export file")
	fs.Parse(args)

	adapter := getAdapter(*harness, *otelFile)
	if adapter == nil {
		fmt.Fprintf(os.Stderr, "unknown harness: %s (supported: claude_code)\n", *harness)
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
	harness := fs.String("harness", "", "harness name (claude_code)")
	sessionID := fs.String("session", "", "session ID")
	snapshotHash := fs.String("snapshot", "", "snapshot hash (git SHA or content hash)")
	skillDir := fs.String("skill-dir", "", "path to the skill being profiled")
	otelFile := fs.String("otel-file", "", "path to OTel export file")
	exportFile := fs.String("export-file", "", "path to a non-OTel session export (e.g. Devin ATIF); reserved — no shipped adapter reads it, and passing it is an error")
	fs.Parse(args)

	if *sessionID == "" || *snapshotHash == "" || *skillDir == "" {
		fmt.Fprintln(os.Stderr, "required: --session, --snapshot, --skill-dir")
		os.Exit(1)
	}

	adapter := getAdapter(*harness, *otelFile)
	if adapter == nil {
		fmt.Fprintf(os.Stderr, "unknown harness: %s (supported: claude_code)\n", *harness)
		os.Exit(1)
	}

	if err := captureFlagError(*harness, *exportFile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	opts := profiler.CaptureOpts{
		ExportFile:   *exportFile,
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
}

// captureFlagError reports why the selected adapter cannot honour the flags it
// was given, or nil when it can.
//
// --export-file fills CaptureOpts.ExportFile, which only an adapter reading a
// session export (Devin's ATIF, a transcript) consults. No such adapter ships
// yet, so no selectable harness can honour the flag. Refuse it: a flag the
// adapter will ignore must fail loudly, not accept a path and produce an
// all-unknown profile that looks like missing telemetry.
func captureFlagError(harness, exportFile string) error {
	if exportFile != "" {
		return fmt.Errorf("--export-file is not read by the %s adapter; supply an OTel export with --otel-file", harness)
	}
	return nil
}

func getAdapter(harness, otelFile string) profiler.ProfilerAdapter {
	switch harness {
	case "claude_code":
		return profiler.ClaudeCodeAdapter{OtelExportFile: otelFile}
	}
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: profiler <command> [flags]

commands:
  probe     Probe environment and report capabilities
  capture   Capture a session profile
  version   Print version

examples:
  profiler probe --harness claude_code --otel-file ./otel-export.json
  profiler capture --harness claude_code --session abc123 --snapshot sha123 --skill-dir ./skills/my-skill --otel-file ./otel-export.json`)
}
