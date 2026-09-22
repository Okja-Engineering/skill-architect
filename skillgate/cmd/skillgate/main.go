// Command skillgate is the safety gate for Agent Skills.
//
//	skillgate gate <dir> [--baseline f] [--fail-on-incomplete] [--format json|sarif] [-o file]
//
// Flags may appear before or after the target, and `--` ends flag parsing.
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

// gateFlags is the gate command's flag set, declared in one place so every
// caller — the command and its tests — sees the same flags, and so a flag
// added later is picked up by argument permutation without further edits.
type gateFlags struct {
	baseline string
	failInc  bool
	format   string
	out      string
	skip     string
}

func (g *gateFlags) register(fs *flag.FlagSet) {
	fs.StringVar(&g.baseline, "baseline", "", "baseline JSON file (entries need a mandatory reason)")
	fs.BoolVar(&g.failInc, "fail-on-incomplete", false, "exit 1 when the coverage ledger is incomplete")
	fs.StringVar(&g.format, "format", "json", "json | sarif")
	fs.StringVar(&g.out, "o", "", "write report to file instead of stdout")
	fs.StringVar(&g.skip, "skip-checks", "", "comma-separated check names to force-skip")
}

// permuteArgs reorders args into the single order Go's flag package accepts —
// flags first, operands last — so a flag may sit either side of the target.
// flag.Parse stops at the first operand, which is why the published form
// `gate <dir> -o report.json` otherwise fails.
//
// Whether a flag consumes the next argument as its value is read from fs
// itself: any non-boolean flag does, in the separate-argument form. Nothing is
// hard-coded here, so a flag added to gateFlags later is permuted correctly
// without touching this function. An argument that looks like a flag but is
// not defined is passed through untouched, so flag.Parse refuses it exactly as
// it always has. `--` ends flag parsing, POSIX-style; the rest are operands.
//
// A flag that needs a value and has none ends the grammar here rather than in
// flag.Parse, because after reordering there is always a `--` behind which
// flag.Parse would happily read as that value — a malformed invocation must
// still be refused, not absorbed.
func permuteArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	flags := make([]string, 0, len(args))
	operands := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			operands = append(operands, args[i+1:]...)
			break
		}
		if len(arg) < 2 || arg[0] != '-' {
			operands = append(operands, arg)
			continue
		}
		flags = append(flags, arg)
		if takesSeparateValue(fs, arg) {
			if i+1 == len(args) {
				return nil, fmt.Errorf("flag needs an argument: %s", arg)
			}
			i++
			flags = append(flags, args[i])
		}
	}
	if len(operands) == 0 {
		return flags, nil
	}
	// The operands are re-emitted behind `--` so one that begins with a dash
	// is still an operand after the round trip.
	return append(append(flags, "--"), operands...), nil
}

// takesSeparateValue reports whether arg is a defined non-boolean flag written
// without `=`, and so claims the following argument as its value. It strips
// dashes the way flag.Parse does: one or two, never more.
func takesSeparateValue(fs *flag.FlagSet, arg string) bool {
	name := arg[1:]
	if name[0] == '-' {
		name = name[1:]
	}
	name, _, hasValue := strings.Cut(name, "=")
	if hasValue {
		return false
	}
	f := fs.Lookup(name)
	if f == nil {
		return false
	}
	b, ok := f.Value.(interface{ IsBoolFlag() bool })
	return !ok || !b.IsBoolFlag()
}

func cmdGate(args []string) {
	fs := flag.NewFlagSet("gate", flag.ExitOnError)
	var g gateFlags
	g.register(fs)
	permuted, err := permuteArgs(fs, args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gate:", err)
		os.Exit(skillgate.ExitExecError)
	}
	fs.Parse(permuted)
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "gate: exactly one target directory required")
		os.Exit(skillgate.ExitExecError)
	}
	target := fs.Arg(0)

	opts := skillgate.Options{
		BaselinePath:     g.baseline,
		FailOnIncomplete: g.failInc,
	}
	if g.skip != "" {
		opts.SkipChecks = strings.Split(g.skip, ",")
	}

	rep, err := skillgate.NewEngine().GateDir(target, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gate:", err)
		os.Exit(skillgate.ExitExecError)
	}

	var data []byte
	switch g.format {
	case "json":
		data, err = json.MarshalIndent(rep, "", "  ")
	case "sarif":
		data, err = skillgate.MarshalSARIF(rep)
	default:
		err = fmt.Errorf("unknown --format %q", g.format)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gate:", err)
		os.Exit(skillgate.ExitExecError)
	}

	if g.out != "" {
		err = os.WriteFile(g.out, append(data, '\n'), 0o644)
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
