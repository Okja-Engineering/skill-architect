package skillgate

import (
	"sort"
	"time"
)

// Version is the gate's semver, stamped into every report.
const Version = "0.1.0"

// Check is a named, registered stage of the gate. A check that is not
// registered — or that declines to run — must appear in checks_skipped, so
// the verdict can never claim coverage it did not get (F13).
type Check struct {
	Name string
	Run  func(l *Ledger, t *Target) []Finding
}

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
}

// Engine is the gate pipeline: G1 ledger → registered checks → G7 verdict.
type Engine struct {
	checks []Check
}

// NewEngine returns an engine with every built-in check registered.
func NewEngine() *Engine {
	e := &Engine{}
	e.checks = append(e.checks, tripwireChecks()...)
	e.checks = append(e.checks, Check{Name: "harness-frontmatter", Run: harnessFrontmatterCheck})
	return e
}

// Gate runs the pipeline over an already-local target directory. Fetching a
// remote target into quarantine (G0) happens before this call.
func (e *Engine) Gate(dir string, opts Options) (*Report, error) {
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

	skip := map[string]bool{}
	for _, s := range opts.SkipChecks {
		skip[s] = true
	}

	var findings []Finding
	var skipped []SkippedCheck
	for _, c := range e.checks {
		if skip[c.Name] {
			skipped = append(skipped, SkippedCheck{Check: c.Name, Reason: "skipped by option"})
			continue
		}
		findings = append(findings, c.Run(ledger, t)...)
	}
	extFindings, extSkipped, budget := externalChecks(ledger, t, skip)
	findings = append(findings, extFindings...)
	skipped = append(skipped, extSkipped...)

	// Pack E per-item leg: char-exact, computed in-process so it never
	// depends on an external binary being present.
	if items := budgetItems(ledger); len(items) > 0 {
		if budget == nil {
			budget = &TokenBudget{Counter: "skillgate/exact-chars", Basis: "measured"}
		}
		budget.Items = items
	}

	// ICM statics run after externals so the measured token budget feeds
	// SK-I005; a missing counter is a named skip, never silence (F13).
	if !skip["icm"] {
		icmF, icmS := icmCheck(ledger, budget)
		findings = append(findings, icmF...)
		skipped = append(skipped, icmS...)
	} else {
		skipped = append(skipped, SkippedCheck{Check: "icm", Reason: "skipped by option"})
	}

	// F14 — the standing boundary skip. A gate audits the bundle before
	// load; its verdict covers the install decision and the read path only.
	// The bash leg (e.g. `bash cat SKILL.md`) is open on every harness
	// examined — pi's own docs instruct bash when read is unavailable — so
	// "no un-pinned SKILL.md enters context by any tool" is not achievable
	// today. It is named here, never claimed closed, and caps every verdict
	// at CAUTION: APPROVE is unreachable until a harness closes that leg.
	skipped = append(skipped, SkippedCheck{
		Check:  CheckPreactivationBashLeg,
		Reason: "pre-activation claims cover the read path and the moment before load only; the bash leg is open on every harness (F14)",
	})

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

// externalChecks runs the opt-in external scanners (G3/G4 + Pack E). Absent
// binaries are named skips, never failures — and their absence caps the
// verdict at CAUTION via F13.
func externalChecks(l *Ledger, t *Target, skip map[string]bool) ([]Finding, []SkippedCheck, *TokenBudget) {
	var findings []Finding
	var skipped []SkippedCheck
	var budget *TokenBudget

	run := func(name string, fn func() ([]Finding, *SkippedCheck)) {
		if skip[name] {
			skipped = append(skipped, SkippedCheck{Check: name, Reason: "skipped by option"})
			return
		}
		f, s := fn()
		findings = append(findings, f...)
		if s != nil {
			skipped = append(skipped, *s)
		}
	}
	run("skillspector", func() ([]Finding, *SkippedCheck) { return runSkillSpector(l, t) })
	run("agnix", func() ([]Finding, *SkippedCheck) { return runAgnix(l, t) })

	if skip["skill-validator"] {
		skipped = append(skipped, SkippedCheck{Check: "skill-validator", Reason: "skipped by option"})
	} else {
		f, b, s := runSkillValidator(l, t)
		findings = append(findings, f...)
		budget = b
		if s != nil {
			skipped = append(skipped, *s)
		}
	}
	return findings, skipped, budget
}
