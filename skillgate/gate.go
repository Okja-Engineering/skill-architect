package skillgate

import (
	"sort"
	"time"
)

// Version is the gate's semver, stamped into every report.
const Version = "0.1.0"

// Target describes the bundle under gate: its ledger plus context derived
// from the frontmatter and config files (populated lazily by helpers).
type Target struct {
	Ledger     *Ledger
	Untrusted  bool // true when the input was fetched/quarantined
	Provenance *Provenance
}

// Options configure a gate run.
type Options struct {
	BaselinePath     string
	FailOnIncomplete bool
	SkipChecks       []string // check names to force-skip (testing, degraded envs)
	OnlyChecks       []string // when non-empty, the only checks to run; every other registered check is a named skip
}

// Engine is the gate pipeline: G1 ledger → registered checks → G7 verdict.
type Engine struct {
	checks []Check
}

// NewEngine returns an engine with every built-in check registered.
func NewEngine() *Engine {
	return &Engine{checks: builtinChecks()}
}

// Gate runs the pipeline over an already-local target directory. Fetching a
// remote target into quarantine (G0) happens before this call.
func (e *Engine) Gate(dir string, opts Options) (*Report, error) {
	// Which checks run, and the reason for each that does not, come from one
	// walk of one registry — so the run set and checks_skipped partition the
	// same list and a narrowed run cannot claim coverage it did not get (F13).
	// Refusing an unknown check name before the ledger is built keeps a
	// mistyped --only from costing a full scan.
	sel, err := selectChecks(e.checks, opts)
	if err != nil {
		return nil, err
	}
	ledger, err := BuildLedger(dir)
	if err != nil {
		return nil, err
	}
	t := &Target{
		Ledger:    ledger,
		Untrusted: false,
		Provenance: &Provenance{
			Path:             ledger.Root,
			SHA256OfSet:      ManifestSHA256(ledger.Entries()),
			PackageManifests: packageManifests(ledger),
		},
	}

	results, budget := e.runChecks(sel, checkInput{Ledger: ledger, Target: t})

	// Collected in registry order, never in completion order: this is what
	// makes a concurrent run emit the same report a serial one did.
	var findings []Finding
	var skipped []SkippedCheck
	for i, c := range e.checks {
		if !sel[i].Run {
			skipped = append(skipped, SkippedCheck{Check: c.Name, Reason: sel[i].Reason})
			continue
		}
		findings = append(findings, results[i].Findings...)
		skipped = append(skipped, results[i].Skipped...)
	}

	// Deterministic emission order, always: a finding's report position and
	// its fingerprint are computed after this sort, so no check's internal
	// iteration order (map ranges, graph walks) can drift between runs.
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.RuleID != b.RuleID {
			return a.RuleID < b.RuleID
		}
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Evidence < b.Evidence
	})

	base, err := LoadBaseline(opts.BaselinePath)
	if err != nil {
		return nil, err
	}
	for i := range findings {
		if findings[i].Fingerprint == "" {
			findings[i].Fingerprint = Fingerprint(findings[i].RuleID, findings[i].File, findings[i].Evidence, findings[i].Severity)
		}
	}
	base.Apply(findings)

	cov := ledger.Coverage()
	rep := &Report{
		Schema:        ReportSchema,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Tool:          ToolInfo{Name: "skillgate", Version: Version},
		Target:        dir,
		Provenance:    t.Provenance,
		Findings:      findings,
		Ledger:        ledger.Entries(),
		Coverage:      cov,
		ChecksSkipped: skipped,
		Refusals:      []string{},
		Tokens:        budget,
		Verdict:       computeVerdict(findings, cov, skipped, t.Untrusted),
	}
	if rep.Findings == nil {
		rep.Findings = []Finding{}
	}
	if rep.Ledger == nil {
		rep.Ledger = []LedgerEntry{}
	}
	if rep.ChecksSkipped == nil {
		rep.ChecksSkipped = []SkippedCheck{}
	}
	return rep, nil
}

// ExitCode maps a report to the CLI exit contract: 1 on REJECT or, under
// --fail-on-incomplete, on any incomplete coverage — an uninspected file or
// a skipped check (an absent optional scanner is incomplete coverage too).
// The standing F14 boundary skip is excluded: --fail-on-incomplete asks
// "was everything that could be examined, examined?" — the bash leg is a
// claim boundary no gate can close, not coverage the operator can complete.
func ExitCode(rep *Report, opts Options) int {
	if rep.Verdict == VerdictReject {
		return ExitGate
	}
	if opts.FailOnIncomplete {
		runnableSkips := 0
		for _, s := range rep.ChecksSkipped {
			if s.Check != CheckPreactivationBashLeg {
				runnableSkips++
			}
		}
		if !rep.Coverage.Complete || runnableSkips > 0 {
			return ExitGate
		}
	}
	return ExitPass
}
