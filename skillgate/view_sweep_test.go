package skillgate

import (
	"os"
	"strings"
	"sync"
	"testing"
)

// The whole-repository view sweep, gated once.
//
// Three tests assert the same shape for three views: gate this repository,
// and every finding carrying that view's tag must come from the hostile
// fixtures and nowhere else. Each one used to call `Gate("..")` itself, so
// the same full-repository scan ran three times — **166 of the package's
// 223 seconds under `-race` on a ten-core machine**, against a three-core
// CI runner and a 600s budget. The package failed that budget once already
// and the failure named whichever test happened to be running.
//
// The scan is identical in all three — same root, same options — so it is
// done once here and shared. Nothing about what the three tests assert
// changes: same corpus, same filter, same both-directions check. This
// redistributes the cost rather than reducing the coverage, which is the
// only version of this worth doing.
//
// The report is **read-only** to its callers. They range over Findings and
// nothing else; a test that needs to mutate one must copy it first.

var (
	sweepOnce sync.Once
	sweepRep  *Report
	sweepErr  error
	sweepSkip bool
)

// repoReport gates this repository once per test binary.
func repoReport(t *testing.T) *Report {
	t.Helper()
	sweepOnce.Do(func() {
		if _, err := os.Stat("../README.md"); err != nil {
			sweepSkip = true
			return
		}
		sweepRep, sweepErr = NewEngine().Gate("..", optsForTest())
	})
	if sweepSkip {
		t.Skip("not running inside the repository")
	}
	if sweepErr != nil {
		t.Fatal(sweepErr)
	}
	return sweepRep
}

// assertViewFiresOnlyOnHostileFixtures is the assertion the three sweeps
// share, unchanged from when each spelled it out.
//
// Adding a view can only *add* findings — dedup is raw-first and derived
// views are scanned after — so the set of findings carrying a view's tag is
// exactly the set that view is responsible for, with no second binary
// needed. Both halves bite: zero hits would mean the sweep had gone vacuous,
// and a hit outside the hostile corpus is the false-positive generator the
// plan warned these views would be.
//
// If it fails on a real document that is a *result*, not a test to relax.
func assertViewFiresOnlyOnHostileFixtures(t *testing.T, tag, noun string) {
	t.Helper()
	rep := repoReport(t)
	hostile := 0
	for _, f := range rep.Findings {
		if f.View != tag {
			continue
		}
		if strings.Contains(f.File, "testdata/hostile/") {
			hostile++
			continue
		}
		t.Errorf("the %s view produced %s at %s:%d — %q",
			noun, f.RuleID, f.File, f.Line, f.Evidence)
	}
	if hostile == 0 {
		t.Errorf("the %s view found nothing anywhere in the repository, not even in the "+
			"hostile fixtures: this sweep is proving nothing", noun)
	}
}

// TestTheViewSweepIsNotVacuous guards the sharing itself. A shared fixture
// that silently produced an empty report would make all three sweeps pass
// while asserting nothing — the exact failure mode the `hostile == 0` check
// exists to catch within one view, applied to the corpus they all read.
func TestTheViewSweepIsNotVacuous(t *testing.T) {
	rep := repoReport(t)
	if len(rep.Findings) < 50 {
		t.Fatalf("the shared repository gate produced %d findings; the three view sweeps "+
			"read this report and would all pass vacuously on an empty one", len(rep.Findings))
	}
	seen := map[string]bool{}
	for _, f := range rep.Findings {
		seen[f.View] = true
	}
	for _, tag := range []string{viewCompactLetter, viewMarkup, viewMarkupCompact} {
		if !seen[tag] {
			t.Errorf("no finding in the shared report carries the %q tag — that view's "+
				"sweep has nothing to account for and passes without asserting", tag)
		}
	}
}
