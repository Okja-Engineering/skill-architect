// Command skillgate is the safety gate for Agent Skills.
//
//	skillgate gate <dir> [--baseline f] [--fail-on-incomplete] [--format json|sarif]
//	                     [-o file] [--skip-checks a,b] [--only a,b]
//	skillgate gate --list-checks
//
// Flags may appear before or after the target, and `--` ends flag parsing.
//
// --skip-checks and --only name registered checks; --list-checks is where
// those names come from, and a name that matches none of them is refused.
// Whatever --only leaves out is reported in checks_skipped, so a narrowed run
// never reads as a full one.
//
// Exit codes: 0 = APPROVE/CAUTION · 1 = REJECT or incomplete ledger under
// --fail-on-incomplete · 2 = the gate itself failed.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
  gate --list-checks
                  print the registered checks and the rules each runs
  version         print version

The gate never prints "safe" or "clean": verdicts are APPROVE, CAUTION,
or REJECT, always beside the coverage ledger.
`)
}

// gateFlags is the gate command's flag set, declared in one place so every
// caller — the command and its tests — sees the same flags, and so a flag
// added later is picked up by argument permutation without further edits.
type gateFlags struct {
	baseline   string
	failInc    bool
	format     string
	out        string
	skip       string
	only       string
	listChecks bool
}

func (g *gateFlags) register(fs *flag.FlagSet) {
	fs.StringVar(&g.baseline, "baseline", "", "baseline JSON file (entries need a mandatory reason)")
	fs.BoolVar(&g.failInc, "fail-on-incomplete", false, "exit 1 when the coverage ledger is incomplete")
	fs.StringVar(&g.format, "format", "json", "json | sarif")
	fs.StringVar(&g.out, "o", "", "write report to file instead of stdout")
	fs.StringVar(&g.skip, "skip-checks", "", "comma-separated check names to force-skip")
	fs.StringVar(&g.only, "only", "", "comma-separated check names to run; every other check is a named skip")
	fs.BoolVar(&g.listChecks, "list-checks", false, "print the registered checks and exit; no target is needed and no report is produced")
}

// splitCheckNames turns a comma-separated flag value into check names. Empty
// fields are dropped, so a trailing comma is not a check with no name.
func splitCheckNames(v string) []string {
	var out []string
	for _, n := range strings.Split(v, ",") {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// listChecks prints what the gate would run: every registered check, the rules
// it can report, and — for the standing boundary — why it never runs.
//
// The list is read off the engine's registry, which is the same list the run
// dispatches and checks_skipped accounts for. A list typed out here would be
// the enumeration that does not mention the check somebody added.
func listChecks(w io.Writer) {
	checks := skillgate.NewEngine().Checks()
	width := 0
	for _, c := range checks {
		if len(c.Name) > width {
			width = len(c.Name)
		}
	}
	for _, c := range checks {
		detail := strings.Join(c.Rules, " ")
		switch {
		case !c.Runs:
			detail = "never runs — " + c.Standing
		case detail == "":
			// A runnable check with no catalogued rule is an external
			// scanner: its findings carry their own source and are advisory,
			// so there is no SK-* id to print. Read off the empty rule set,
			// not off a list of which checks are external.
			detail = "reports under its own source; no SK-* rules of its own"
		}
		fmt.Fprintln(w, strings.TrimRight(fmt.Sprintf("%-*s  %s", width, c.Name, detail), " "))
		for _, leg := range c.Legs {
			fmt.Fprintf(w, "%-*s  reports the leg %s when it has no counter to read\n", width, "", leg)
		}
	}
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

	// Asking what the gate would run is not a run: no target, no report.
	if g.listChecks {
		listChecks(os.Stdout)
		os.Exit(skillgate.ExitPass)
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "gate: exactly one target directory required")
		os.Exit(skillgate.ExitExecError)
	}
	target := fs.Arg(0)

	opts := skillgate.Options{
		BaselinePath:     g.baseline,
		FailOnIncomplete: g.failInc,
		SkipChecks:       splitCheckNames(g.skip),
		OnlyChecks:       splitCheckNames(g.only),
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
