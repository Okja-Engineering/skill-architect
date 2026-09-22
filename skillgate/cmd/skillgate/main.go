// Command skillgate is the safety gate for Agent Skills.
//
//	skillgate gate <dir> [--baseline f] [--fail-on-incomplete] [--format json|sarif] [-o file]
//
// Exit codes: 0 = APPROVE/CAUTION · 1 = REJECT or incomplete ledger under
// --fail-on-incomplete · 2 = the gate itself failed.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Okja-Engineering/skill-architect/skillgate"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(skillgate.ExitExecError)
	}
	switch os.Args[1] {
	case "gate":
		cmdGate(os.Args[2:])
	case "version":
		fmt.Println("skillgate", skillgate.Version)
	default:
		usage()
		os.Exit(skillgate.ExitExecError)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: skillgate <command>

  gate <dir|url>  run the safety gate over a local bundle or git URL
  version         print version

The gate never prints "safe" or "clean": verdicts are APPROVE, CAUTION,
or REJECT, always beside the coverage ledger.
`)
}

func cmdGate(args []string) {
	fs := flag.NewFlagSet("gate", flag.ExitOnError)
	baseline := fs.String("baseline", "", "baseline JSON file (entries need a mandatory reason)")
	failInc := fs.Bool("fail-on-incomplete", false, "exit 1 when the coverage ledger is incomplete")
	format := fs.String("format", "json", "json | sarif")
	out := fs.String("o", "", "write report to file instead of stdout")
	skip := fs.String("skip-checks", "", "comma-separated check names to force-skip")
	fs.Parse(args)
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "gate: exactly one target directory required")
		os.Exit(skillgate.ExitExecError)
	}
	target := fs.Arg(0)

	opts := skillgate.Options{
		BaselinePath:     *baseline,
		FailOnIncomplete: *failInc,
	}
	if *skip != "" {
		opts.SkipChecks = strings.Split(*skip, ",")
	}

	rep, err := skillgate.NewEngine().GateDir(target, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gate:", err)
		os.Exit(skillgate.ExitExecError)
	}

	var data []byte
	switch *format {
	case "json":
		data, err = json.MarshalIndent(rep, "", "  ")
	case "sarif":
		data, err = skillgate.MarshalSARIF(rep)
	default:
		err = fmt.Errorf("unknown --format %q", *format)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gate:", err)
		os.Exit(skillgate.ExitExecError)
	}

	if *out != "" {
		err = os.WriteFile(*out, append(data, '\n'), 0o644)
	} else {
		_, err = os.Stdout.Write(append(data, '\n'))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gate:", err)
		os.Exit(skillgate.ExitExecError)
	}

	fmt.Fprintf(os.Stderr, "verdict: %s · findings: %d · coverage: %d/%d · checks_skipped: %d\n",
		rep.Verdict, countLive(rep.Findings), rep.Coverage.Inspected, rep.Coverage.Total, len(rep.ChecksSkipped))
	os.Exit(skillgate.ExitCode(rep, opts))
}

func countLive(fs []skillgate.Finding) int {
	n := 0
	for _, f := range fs {
		if !f.Suppressed {
			n++
		}
	}
	return n
}
