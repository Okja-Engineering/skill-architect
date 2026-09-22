package skillgate

import (
	"sort"
	"strings"
	"testing"
)

// The check registry is published by --list-checks and consumed by
// --skip-checks and --only. These tests hold the one invariant that matters
// about it: the set it publishes is the set the run actually accounts for.
//
// No side here is derived from the side it checks:
//
//   - the published side is CheckCatalog(), read off the engine's registry;
//   - the reported side is checks_skipped in a real report/v1, produced by a
//     real Gate() run over a real bundle;
//   - the rule side is RuleCatalog(), which S03 already checks against the
//     package source by AST.
//
// A check that runs outside the registry emits no skip entry when it is
// skipped, so the two sets diverge and the divergence names it.

// namesOf is the published set, as names.
func namesOf(cs []CheckInfo) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Name)
	}
	sort.Strings(out)
	return out
}

// skippedNames is the reported set, as names.
func skippedNames(rep *Report) []string {
	out := make([]string, 0, len(rep.ChecksSkipped))
	for _, s := range rep.ChecksSkipped {
		out = append(out, s.Check)
	}
	sort.Strings(out)
	return out
}

func diffSets(want, got []string) (missing, extra []string) {
	w := map[string]bool{}
	for _, s := range want {
		w[s] = true
	}
	g := map[string]bool{}
	for _, s := range got {
		g[s] = true
	}
	for _, s := range want {
		if !g[s] {
			missing = append(missing, s)
		}
	}
	for _, s := range got {
		if !w[s] {
			extra = append(extra, s)
		}
	}
	return missing, extra
}

// demoBundle is a well-formed skill with nothing to report, so a divergence
// in these tests is about the registry and never about a rule.
func demoBundle(t *testing.T) string {
	t.Helper()
	return writeBundle(t, map[string]string{
		"SKILL.md": "---\nname: demo\ndescription: a demo skill\n---\nUse it.\n",
	})
}

// TestEveryRuleNamesARegisteredCheck is the tie between the two registries.
// CatalogRule.Check is the name --skip-checks matches and checks_skipped
// reports; a rule whose check the engine does not register is a rule that
// cannot be skipped by name and cannot be published by --list-checks.
func TestEveryRuleNamesARegisteredCheck(t *testing.T) {
	registered := map[string]bool{}
	for _, c := range NewEngine().Checks() {
		registered[c.Name] = true
	}
	if len(registered) == 0 {
		t.Fatal("the engine registers no checks — the comparison would be vacuous")
	}
	rules := RuleCatalog()
	if len(rules) == 0 {
		t.Fatal("RuleCatalog() is empty — the comparison would be vacuous")
	}
	for _, r := range rules {
		if !registered[r.Check] {
			t.Errorf("rule %s names check %q and the engine registers no such check: "+
				"the check runs outside the registry, so --list-checks cannot publish it "+
				"and --skip-checks cannot name it", r.ID, r.Check)
		}
	}
}

// TestPublishedChecksAreTheChecksTheRunAccountsFor drives the whole set
// through a real run. Skipping every published name must produce a report
// whose checks_skipped is exactly that set: a check that runs outside the
// registry does not honour the skip and contributes no entry, and a name
// published for a check the engine never dispatches contributes none either.
//
// icm-token-budget — the named leg inside icm that needs a measured counter —
// cannot appear here because icm itself is skipped. The leg is covered by
// TestEverySkipReportedNamesSomethingPublished, which runs icm for real.
func TestPublishedChecksAreTheChecksTheRunAccountsFor(t *testing.T) {
	published := namesOf(NewEngine().Checks())
	if len(published) == 0 {
		t.Fatal("--list-checks would publish nothing — the comparison would be vacuous")
	}
	rep, err := NewEngine().Gate(demoBundle(t), Options{SkipChecks: published})
	if err != nil {
		t.Fatal(err)
	}
	missing, extra := diffSets(published, skippedNames(rep))
	for _, n := range missing {
		t.Errorf("check %q is published by --list-checks and skipping it produced no "+
			"checks_skipped entry — it is published but never dispatched, or it runs "+
			"outside the registry and ignores the skip", n)
	}
	for _, n := range extra {
		t.Errorf("check %q appears in checks_skipped and --list-checks does not publish "+
			"it — the gate accounts for a check no published list names", n)
	}
}

// TestEverySkipReportedNamesSomethingPublished is the same invariant on an
// ordinary run, where every check that can run does. Every name the report
// puts in checks_skipped must be a published check or a published leg of one,
// so a reader who sees a name can look it up.
func TestEverySkipReportedNamesSomethingPublished(t *testing.T) {
	known := map[string]bool{}
	for _, c := range NewEngine().Checks() {
		known[c.Name] = true
		for _, leg := range c.Legs {
			known[leg] = true
		}
	}
	// The external scanners are skipped so the run does not depend on what is
	// installed; icm runs, and with no measured counter it names its own leg.
	rep, err := NewEngine().Gate(demoBundle(t), optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.ChecksSkipped) == 0 {
		t.Fatal("no check was skipped — the comparison would be vacuous")
	}
	sawLeg := false
	for _, s := range rep.ChecksSkipped {
		if !known[s.Check] {
			t.Errorf("checks_skipped names %q and --list-checks publishes no check or leg "+
				"by that name (reason: %s)", s.Check, s.Reason)
		}
		if s.Check == CheckICMTokenBudget {
			sawLeg = true
		}
	}
	if !sawLeg {
		t.Fatalf("this run was meant to exercise the %s leg and did not — "+
			"the leg half of the assertion is vacuous", CheckICMTokenBudget)
	}
}

// TestOnlyNamesEveryCheckItDidNotRun is F13 applied to the narrowest possible
// run: a verdict may never claim coverage it did not get. --only is the
// easiest way in the whole program to break that.
func TestOnlyNamesEveryCheckItDidNotRun(t *testing.T) {
	published := NewEngine().Checks()
	const selected = "tripwire-injection"
	var want []string
	for _, c := range published {
		if c.Name != selected {
			want = append(want, c.Name)
		}
	}
	sort.Strings(want)

	rep, err := NewEngine().Gate(demoBundle(t), Options{OnlyChecks: []string{selected}})
	if err != nil {
		t.Fatal(err)
	}
	missing, extra := diffSets(want, skippedNames(rep))
	for _, n := range missing {
		t.Errorf("--only %s did not run check %q and checks_skipped does not name it: "+
			"the report claims coverage the run did not get", selected, n)
	}
	for _, n := range extra {
		t.Errorf("--only %s reports %q as skipped and it is neither the selection nor a "+
			"published check", selected, n)
	}
	for _, s := range rep.ChecksSkipped {
		if s.Reason == "" {
			t.Errorf("checks_skipped entry %q carries no reason", s.Check)
		}
	}
	// A narrowed run can never be complete coverage, so it can never APPROVE.
	if rep.Verdict == VerdictApprove {
		t.Errorf("verdict = %s after --only narrowed the run to one check", rep.Verdict)
	}
}

// TestOnlyAndSkipComposeWithoutLosingACheck holds the ran-xor-skipped
// invariant under both selectors at once: every registered check either ran
// or is named in checks_skipped, and never both.
func TestOnlyAndSkipComposeWithoutLosingACheck(t *testing.T) {
	rep, err := NewEngine().Gate(demoBundle(t), Options{
		OnlyChecks: []string{"tripwire-injection", "tripwire-exfil", CheckICM},
		SkipChecks: []string{"tripwire-exfil"},
	})
	if err != nil {
		t.Fatal(err)
	}
	reported := map[string]string{}
	for _, s := range rep.ChecksSkipped {
		if prev, dup := reported[s.Check]; dup {
			t.Errorf("check %q is reported skipped twice (%q and %q)", s.Check, prev, s.Reason)
		}
		reported[s.Check] = s.Reason
	}
	ran := map[string]bool{"tripwire-injection": true, CheckICM: true}
	for _, c := range NewEngine().Checks() {
		_, skipped := reported[c.Name]
		if ran[c.Name] && skipped {
			t.Errorf("check %q was selected to run and is also reported skipped: %q", c.Name, reported[c.Name])
		}
		if !ran[c.Name] && !skipped {
			t.Errorf("check %q neither ran nor is reported skipped — it fell out of both "+
				"sides of the selection", c.Name)
		}
	}
	if r := reported["tripwire-exfil"]; !strings.Contains(r, "skipped by option") {
		t.Errorf("tripwire-exfil was named by both --only and --skip-checks; reason = %q, "+
			"want the explicit skip to win", r)
	}
}

// TestUnknownCheckNameIsRefused: a name that matches no registered check is a
// mistake, and under --only it silently narrows the run to nothing. It is
// refused, by name, rather than absorbed.
func TestUnknownCheckNameIsRefused(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts Options
	}{
		{"only", Options{OnlyChecks: []string{"tripwire-injecton"}}},
		{"skip", Options{SkipChecks: []string{"tripwire-injecton"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewEngine().Gate(demoBundle(t), tc.opts)
			if err == nil {
				t.Fatal("an unknown check name was accepted")
			}
			if !strings.Contains(err.Error(), "tripwire-injecton") {
				t.Errorf("error does not name the unknown check: %v", err)
			}
		})
	}
}

// TestStandingBoundaryIsDeclaredInTheRegistry: F14's permanent skip is the one
// entry in checks_skipped that is not a check that could have run. It is a
// registry member so --list-checks can publish it and a reader can look it up,
// and it is declared unrunnable so --only cannot select it into a run.
func TestStandingBoundaryIsDeclaredInTheRegistry(t *testing.T) {
	var found *CheckInfo
	for _, c := range NewEngine().Checks() {
		c := c
		if c.Name == CheckPreactivationBashLeg {
			found = &c
		}
	}
	if found == nil {
		t.Fatalf("%s is reported on every run and the registry does not declare it",
			CheckPreactivationBashLeg)
	}
	if found.Runs {
		t.Errorf("%s is published as a check that runs; it can never run", found.Name)
	}
	if found.Standing == "" {
		t.Errorf("%s is published as unrunnable with no reason", found.Name)
	}
	if _, err := NewEngine().Gate(demoBundle(t), Options{
		OnlyChecks: []string{CheckPreactivationBashLeg},
	}); err == nil {
		t.Errorf("--only %s was accepted; a standing boundary cannot be selected into a run",
			CheckPreactivationBashLeg)
	}
}

// TestRegistryDeclarationsAreConsistent guards the registry's own shape: a
// check that can run has something to run, and one that cannot says why.
func TestRegistryDeclarationsAreConsistent(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range NewEngine().checks {
		if seen[c.Name] {
			t.Errorf("check %q is registered twice — --skip-checks would match both", c.Name)
		}
		seen[c.Name] = true
		if c.Runnable() != (c.Run != nil) {
			t.Errorf("check %q: Runnable() = %v with Run != nil = %v", c.Name, c.Runnable(), c.Run != nil)
		}
		if !c.Runnable() && c.Standing == "" {
			t.Errorf("check %q cannot run and gives no reason", c.Name)
		}
	}
}

// TestAPanickingCheckIsANamedSkipNotACrash: with the checks running
// concurrently, an unrecovered panic in one goroutine takes the whole process
// down and the operator gets no report at all. A check that panics is coverage
// the run did not get, so it becomes a named skip — loud in checks_skipped,
// capping the verdict at CAUTION — and every other check still reports.
func TestAPanickingCheckIsANamedSkipNotACrash(t *testing.T) {
	e := NewEngine()
	e.checks = append(e.checks, Check{
		Name: "boom",
		Run:  func(checkInput) checkResult { panic("kaboom") },
	})
	rep, err := e.Gate(demoBundle(t), optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	var reason string
	for _, s := range rep.ChecksSkipped {
		if s.Check == "boom" {
			reason = s.Reason
		}
	}
	if reason == "" {
		t.Fatal("a check panicked and checks_skipped does not name it")
	}
	if !strings.Contains(reason, "kaboom") {
		t.Errorf("the skip reason does not carry the panic value: %q", reason)
	}
	if rep.Verdict == VerdictApprove {
		t.Errorf("verdict = %s with a check that did not complete", rep.Verdict)
	}
	// The rest of the run is intact: the ledger was still built and inspected.
	if !rep.Coverage.Complete {
		t.Errorf("a panicking check damaged the file ledger: %+v", rep.Coverage)
	}
}
