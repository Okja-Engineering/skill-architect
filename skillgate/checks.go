package skillgate

import (
	"fmt"
	"sort"
	"sync"
)

// The check registry — what the gate runs, answered by the gate.
//
// A check's name is load-bearing three times over: --skip-checks and --only
// match it, checks_skipped reports it, and a CatalogRule names it. So there
// can only be one place that knows the set, and this is it.
//
// Until this file there were four. The tripwire groups and harness-frontmatter
// were registered on the engine; the three external scanners were run by a
// function beside it; icm was an `if` in Gate; the F14 boundary was an append.
// A --list-checks that walked the engine's registry would have published a set
// missing five of them — an enumeration asserted complete that is not, which
// is what the catalog side (catalog.go) was repaired for one slice earlier.
//
// So every check is registered here, including the one that can never run, and
// both --list-checks and checks_skipped are derived from this one walk. A check
// that is not in builtinChecks() does not run at all, rather than running
// unpublished.

// checkInput is everything a check may read. The ledger and the target are
// read-only for the whole run — views are materialised when the ledger is
// built, so nothing here is written behind a shared pointer.
type checkInput struct {
	Ledger *Ledger
	Target *Target
	// Budget is the measured token budget. It is nil until the stage that
	// measures it has run, which is why a check that needs it declares
	// stageDerive rather than assuming an order.
	Budget *TokenBudget
}

// checkResult is everything a check may produce: findings, the named skips it
// reports about itself (an absent binary, a leg it could not measure), and a
// token budget when the check is the one that measures it.
type checkResult struct {
	Findings []Finding
	Skipped  []SkippedCheck
	Budget   *TokenBudget
}

// checkStage orders the run. Every runnable check in a stage is dispatched
// concurrently and all of them complete before the next stage begins, so a
// check that reads what another measured declares the later stage instead of
// relying on where it happens to sit in the registry.
type checkStage int

const (
	// stageScan depends on nothing but the ledger.
	stageScan checkStage = iota
	// stageDerive reads what stageScan measured — today, the token budget.
	stageDerive
)

// Check is a named, registered stage of the gate. A check that does not run —
// skipped by option, unselected by --only, unable to run at all, or crashed —
// must appear in checks_skipped, so the verdict can never claim coverage it
// did not get (F13).
type Check struct {
	// Name is what --skip-checks and --only match and what checks_skipped
	// reports.
	Name string
	// Stage is when the check may run relative to the others.
	Stage checkStage
	// Run executes the check. It is nil exactly when Standing is set.
	Run func(checkInput) checkResult
	// Standing, when non-empty, is the written reason this check can never
	// run. It is a claim boundary rather than coverage an operator can
	// complete: it is reported skipped on every run, and --only cannot select
	// it into one. F14's pre-activation bash leg is the only such entry.
	Standing string
	// Legs are named sub-checks inside this check that can be reported skipped
	// on their own, for want of an input the check cannot supply itself. They
	// are not selectable — they have no independent run — but they appear in
	// checks_skipped, so they are published here and a reader who sees one can
	// look it up.
	Legs []string
}

// Runnable reports whether the check can ever run.
func (c Check) Runnable() bool { return c.Standing == "" }

// CheckInfo is one registered check, as the gate declares it — the shape
// --list-checks publishes.
type CheckInfo struct {
	// Name is the name --skip-checks and --only match.
	Name string `json:"name"`
	// Runs is false for a standing boundary, which is reported skipped on
	// every run and can never be selected.
	Runs bool `json:"runs"`
	// Standing is the reason a non-running check can never run.
	Standing string `json:"standing,omitempty"`
	// Legs are the named sub-skips this check can report (see Check.Legs).
	Legs []string `json:"legs,omitempty"`
	// Rules are the rule ids this check can report, taken from RuleCatalog().
	// Empty for the external scanners, whose findings carry their own source
	// and are advisory rather than catalogued SK-* rules.
	Rules []string `json:"rules,omitempty"`
}

// Checks returns every registered check, in the order the report accounts for
// them. This is the list --list-checks publishes, and it is read off the
// registry the run dispatches — never typed out beside it.
func (e *Engine) Checks() []CheckInfo {
	byCheck := map[string][]string{}
	for _, r := range RuleCatalog() {
		byCheck[r.Check] = append(byCheck[r.Check], r.ID)
	}
	out := make([]CheckInfo, 0, len(e.checks))
	for _, c := range e.checks {
		out = append(out, CheckInfo{
			Name:     c.Name,
			Runs:     c.Runnable(),
			Standing: c.Standing,
			Legs:     c.Legs,
			Rules:    byCheck[c.Name],
		})
	}
	return out
}

// builtinChecks is the registry: every check the gate can account for, in the
// order the report accounts for them.
//
// The order is the emission order of findings and of checks_skipped, so it is
// part of the report's determinism, not a detail. Stage — not position — is
// what decides when a check may run.
func builtinChecks() []Check {
	checks := tripwireChecks()
	checks = append(checks,
		Check{
			Name: CheckHarnessFrontmatter, Stage: stageScan,
			Run: func(in checkInput) checkResult {
				return checkResult{Findings: harnessFrontmatterCheck(in.Ledger, in.Target)}
			},
		},
		// The opt-in external scanners (G3/G4 + Pack E). An absent binary is a
		// named skip, never a failure, and its absence caps the verdict at
		// CAUTION via F13.
		externalCheck(CheckSkillSpector, runSkillSpector),
		externalCheck(CheckAgnix, runAgnix),
		Check{
			Name: CheckSkillValidator, Stage: stageScan,
			Run: func(in checkInput) checkResult {
				f, b, s := runSkillValidator(in.Ledger, in.Target)
				return checkResult{Findings: f, Skipped: skipList(s), Budget: b}
			},
		},
		// icm is a deterministic, in-repo check, but it reads the measured
		// token budget for SK-I005 — so it declares stageDerive rather than
		// relying on sitting after the scanners in this list.
		Check{
			Name: CheckICM, Stage: stageDerive, Legs: []string{CheckICMTokenBudget},
			Run: func(in checkInput) checkResult {
				f, s := icmCheck(in.Ledger, in.Budget)
				return checkResult{Findings: f, Skipped: s}
			},
		},
		// F14 — the standing boundary. A gate audits the bundle before load;
		// its verdict covers the install decision and the read path only. The
		// bash leg (e.g. `bash cat SKILL.md`) is open on every harness
		// examined — pi's own docs instruct bash when read is unavailable — so
		// "no un-pinned SKILL.md enters context by any tool" is not achievable
		// today. It is named on every report, never claimed closed, and caps
		// every verdict at CAUTION: APPROVE is unreachable until a harness
		// closes that leg.
		Check{
			Name: CheckPreactivationBashLeg,
			Standing: "pre-activation claims cover the read path and the moment before load only; " +
				"the bash leg is open on every harness (F14)",
		},
	)
	return checks
}

// externalCheck adapts a scanner that reports findings and one skip about
// itself. Both external scanners have that shape; skill-validator also
// measures the token budget and is registered directly.
func externalCheck(name string, fn func(*Ledger, *Target) ([]Finding, *SkippedCheck)) Check {
	return Check{
		Name: name, Stage: stageScan,
		Run: func(in checkInput) checkResult {
			f, s := fn(in.Ledger, in.Target)
			return checkResult{Findings: f, Skipped: skipList(s)}
		},
	}
}

func skipList(s *SkippedCheck) []SkippedCheck {
	if s == nil {
		return nil
	}
	return []SkippedCheck{*s}
}

// checkSelection is the decision for one registered check: it runs, or it did
// not and this is the reason that goes in checks_skipped. Exactly one of the
// two is always true, because both come from the same walk of the same
// registry — which is the only way a narrowed run can be prevented from
// claiming coverage it did not get (F13).
type checkSelection struct {
	Run bool
	// Reason is why the check did not run. Never empty when Run is false.
	Reason string
}

// selectChecks decides, for every registered check, whether it runs.
//
// A name that matches no registered check is refused rather than absorbed:
// under --only it would silently narrow the run to nothing, and under
// --skip-checks it is an operator who believes they narrowed coverage and did
// not. --list-checks is where the valid names come from.
func selectChecks(checks []Check, opts Options) ([]checkSelection, error) {
	if err := checkNamesExist(checks, opts.OnlyChecks, "--only"); err != nil {
		return nil, err
	}
	if err := checkNamesExist(checks, opts.SkipChecks, "--skip-checks"); err != nil {
		return nil, err
	}
	only := map[string]bool{}
	for _, n := range opts.OnlyChecks {
		only[n] = true
	}
	skip := map[string]bool{}
	for _, n := range opts.SkipChecks {
		skip[n] = true
	}
	for _, c := range checks {
		if only[c.Name] && !c.Runnable() {
			return nil, fmt.Errorf("--only names check %q, which can never run: %s", c.Name, c.Standing)
		}
	}
	sel := make([]checkSelection, len(checks))
	for i, c := range checks {
		switch {
		case !c.Runnable():
			sel[i] = checkSelection{Reason: c.Standing}
		case skip[c.Name]:
			sel[i] = checkSelection{Reason: "skipped by option"}
		case len(only) > 0 && !only[c.Name]:
			sel[i] = checkSelection{Reason: "not selected by --only"}
		default:
			sel[i] = checkSelection{Run: true}
		}
	}
	return sel, nil
}

func checkNamesExist(checks []Check, names []string, flag string) error {
	if len(names) == 0 {
		return nil
	}
	known := map[string]bool{}
	for _, c := range checks {
		known[c.Name] = true
	}
	for _, n := range names {
		if !known[n] {
			return fmt.Errorf("%s names %q and no such check is registered (see --list-checks)", flag, n)
		}
	}
	return nil
}

// runChecks dispatches the selected checks and returns one result per
// registered check, aligned by index with the registry.
//
// Checks within a stage run concurrently; every stage completes before the
// next begins. Two properties matter more than the speed:
//
//   - Determinism. No goroutine appends to a shared slice: each writes its own
//     results[i], and Gate reads them back in registry order. So the emission
//     order of findings and of checks_skipped is the registry's order, exactly
//     as it was when the checks ran one after another, whatever order they
//     finish in. The ledger and the target are read-only for the whole run —
//     views are materialised when the ledger is built — so a check cannot see
//     another check's progress.
//   - Failure isolation. An unrecovered panic in a goroutine takes the whole
//     process down, and the operator gets no report at all. runCheck converts
//     one into a named skip instead: see there.
func (e *Engine) runChecks(sel []checkSelection, in checkInput) ([]checkResult, *TokenBudget) {
	results := make([]checkResult, len(e.checks))
	var budget *TokenBudget
	for _, stage := range stageOrder(e.checks) {
		var wg sync.WaitGroup
		for i := range e.checks {
			if !sel[i].Run || e.checks[i].Stage != stage {
				continue
			}
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				results[i] = runCheck(e.checks[i], in)
			}(i)
		}
		wg.Wait()
		// The stage boundary is where what one stage measured becomes visible
		// to the next.
		budget = assembleBudget(results, in.Ledger)
		in.Budget = budget
	}
	return results, budget
}

// stageOrder is the distinct stages the registry declares, in order — read off
// the registry, so a check that declares a new stage is sequenced without an
// edit here.
func stageOrder(checks []Check) []checkStage {
	seen := map[checkStage]bool{}
	var out []checkStage
	for _, c := range checks {
		if !c.Runnable() || seen[c.Stage] {
			continue
		}
		seen[c.Stage] = true
		out = append(out, c.Stage)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// runCheck runs one check and turns a panic into a named skip.
//
// A check that crashed is coverage the run did not get, which is precisely
// what checks_skipped exists to say: the entry caps the verdict at CAUTION,
// counts as incomplete under --fail-on-incomplete, and leaves every other
// check's findings — including a blocker, which still REJECTs — intact. The
// alternatives are both worse: an unrecovered panic in a goroutine kills the
// process and reports nothing at all, and failing the whole gate throws away
// the findings the other checks did produce.
//
// Only the panic value is recorded. A stack trace carries goroutine ids and
// addresses, and report/v1 must be byte-identical across runs.
func runCheck(c Check, in checkInput) (res checkResult) {
	defer func() {
		if r := recover(); r != nil {
			res = checkResult{Skipped: []SkippedCheck{{
				Check:  c.Name,
				Reason: fmt.Sprintf("check panicked and did not complete: %v", r),
			}}}
		}
	}()
	return c.Run(in)
}

// assembleBudget is the token budget as it stands after a stage: whatever
// measured the per-file counts, plus the per-item leg computed in-process so
// it never depends on an external binary being present.
//
// The per-item leg is a measurement, not a check — it reports no finding and
// there is nothing about it to skip or select — so it is not in the registry.
func assembleBudget(results []checkResult, l *Ledger) *TokenBudget {
	var b *TokenBudget
	for _, r := range results {
		if r.Budget != nil {
			b = r.Budget
		}
	}
	if items := budgetItems(l); len(items) > 0 {
		if b == nil {
			b = &TokenBudget{Counter: "skillgate/exact-chars", Basis: "measured"}
		}
		b.Items = items
	}
	return b
}
