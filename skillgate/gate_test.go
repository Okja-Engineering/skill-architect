package skillgate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// optsForTest force-skips external scanners so verdicts depend only on the
// deterministic floor.
func optsForTest() Options {
	return Options{SkipChecks: []string{"skillspector", "agnix", "skill-validator"}}
}

func TestGateCleanBundleApprovesOnlyWithCompleteLedger(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"SKILL.md": "---\nname: demo\ndescription: a demo skill\n---\nUse it.\n",
	})
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	// skillspector is force-skipped → F13 caps the verdict at CAUTION even
	// with zero findings.
	if rep.Verdict != VerdictCaution {
		t.Fatalf("verdict = %s, want CAUTION (skipped check caps)", rep.Verdict)
	}
	if ExitCode(rep, optsForTest()) != ExitPass {
		t.Fatal("exit code should be 0 for CAUTION")
	}
}

func TestGateRejectsOnBlocker(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"SKILL.md": "Body referencing ../../outside/secret.txt\n",
	})
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Verdict != VerdictReject {
		t.Fatalf("verdict = %s, want REJECT", rep.Verdict)
	}
	if ExitCode(rep, optsForTest()) != ExitGate {
		t.Fatal("exit code should be 1 for REJECT")
	}
	var found bool
	for _, f := range rep.Findings {
		if f.RuleID == "SK-T019" && f.Blocks() {
			found = true
		}
	}
	if !found {
		t.Fatal("expected a blocking SK-T019 finding")
	}
}

func TestVerdictNeverAboveCautionOnIncompleteLedger(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"SKILL.md": "body\n",
		"bin.dat":  "a\x00b\n",
	})
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Verdict == VerdictApprove {
		t.Fatal("incomplete ledger must cap verdict below APPROVE (F13)")
	}
	if ExitCode(rep, Options{FailOnIncomplete: true, SkipChecks: []string{"skillspector"}}) != ExitGate {
		t.Fatal("--fail-on-incomplete should exit 1 on incomplete ledger")
	}
}

func TestVerdictRejectsIncompleteLedgerOnUntrusted(t *testing.T) {
	findings := []Finding{}
	cov := Coverage{Total: 2, Inspected: 1, Skipped: 1, Complete: false}
	if v := computeVerdict(findings, cov, nil, true); v != VerdictReject {
		t.Fatalf("untrusted + incomplete = %s, want REJECT", v)
	}
	if v := computeVerdict(findings, cov, nil, false); v != VerdictCaution {
		t.Fatalf("local + incomplete = %s, want CAUTION", v)
	}
}

func TestReportNeverSaysSafeOrClean(t *testing.T) {
	root := writeBundle(t, map[string]string{"SKILL.md": "body\n"})
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{VerdictApprove, VerdictCaution, VerdictReject} {
		_ = v
	}
	if strings.Contains(strings.ToLower(rep.Verdict), "safe") || strings.Contains(strings.ToLower(rep.Verdict), "clean") {
		t.Fatalf("verdict %q violates F12", rep.Verdict)
	}
}

func TestBaselineSuppressesWithReasonAndDriftFailsClosed(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"SKILL.md": "Body referencing ../../outside/x\n",
	})
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	// Locate the T019 finding by rule, not position — report order is sorted
	// (rule, file, line, evidence), and other findings' fingerprints do not
	// cover the drifting text.
	var f *Finding
	for i := range rep.Findings {
		if rep.Findings[i].RuleID == "SK-T019" {
			f = &rep.Findings[i]
			break
		}
	}
	if f == nil {
		t.Fatal("expected an SK-T019 finding")
	}
	basePath := filepath.Join(t.TempDir(), "baseline.json")
	good := `{"entries":[{"rule_id":"` + f.RuleID + `","fingerprint":"` + f.Fingerprint + `","reason":"reviewed — test fixture"}]}`
	if err := os.WriteFile(basePath, []byte(good), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := optsForTest()
	opts.BaselinePath = basePath
	rep2, err := NewEngine().Gate(root, opts)
	if err != nil {
		t.Fatal(err)
	}
	suppressed := func(rep *Report) bool {
		for _, g := range rep.Findings {
			if g.RuleID == "SK-T019" {
				return g.Suppressed && g.SuppressReason != ""
			}
		}
		return false
	}
	if !suppressed(rep2) {
		t.Fatal("finding should be suppressed with the recorded reason")
	}

	// Drift: same rule, different evidence → fingerprint mismatch → no suppression.
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("Body referencing ../../outside/CHANGED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rep3, err := NewEngine().Gate(root, opts)
	if err != nil {
		t.Fatal(err)
	}
	if suppressed(rep3) {
		t.Fatal("content drift must fail closed: no suppression")
	}
}

func TestBaselineEntryWithoutReasonRejected(t *testing.T) {
	basePath := filepath.Join(t.TempDir(), "baseline.json")
	if err := os.WriteFile(basePath, []byte(`{"entries":[{"rule_id":"SK-T019","fingerprint":"abc","reason":""}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBaseline(basePath); err == nil {
		t.Fatal("baseline entry without reason must be rejected")
	}
}

func TestCleanCorpusHasNoTripwireFindings(t *testing.T) {
	// Our own skills must pass our own gate on merit — the T013 spec
	// tension was resolved by fixing the skills (allowed-tools declared),
	// never by re-scoping the rule.
	//
	// The second half of that sentence used to read "skill-rewrite
	// self-contained". It is not, it never was on any shipped commit, and
	// the skill says so itself: `skills/skill-rewrite/SKILL.md` line 50
	// reads "This skill is not self-contained ... Install or prune the two
	// skills together", and README.md, RELEASE_NOTES.md and CHANGELOG.md
	// each repeat it. So SK-T019 is *right* about this bundle — it reaches
	// into its sibling for `verdict-guard.sh` — and the assertion as
	// written demanded the opposite of what the release ships.
	//
	// It is recorded where a reviewed-and-accepted finding belongs, in a
	// baseline: fail-closed on content drift, reason mandatory, finding
	// still present in the report and still reported at full severity to
	// anyone who gates this directory without it. `!f.Suppressed` below is
	// original and was written for exactly this. The rule was not touched
	// to make this pass.
	corpus := []struct{ path, baseline string }{
		{"../skills/skill-audit", ""},
		{"../skills/skill-rewrite", "testdata/dogfood/skill-rewrite-baseline.json"},
		{"../skills/skill-gate", ""},
	}
	ran := false
	for _, entry := range corpus {
		skill := entry.path
		if _, err := os.Stat(skill); err != nil {
			continue
		}
		ran = true
		opts := optsForTest()
		opts.BaselinePath = entry.baseline
		rep, err := NewEngine().Gate(skill, opts)
		if err != nil {
			t.Fatal(err)
		}
		// A baseline that suppresses nothing is a baseline that has gone
		// stale, and a stale one hides the finding it was written for by
		// simply not matching. Assert it did its job.
		if entry.baseline != "" {
			var suppressed int
			for _, f := range rep.Findings {
				if f.Suppressed {
					suppressed++
				}
			}
			if suppressed == 0 {
				t.Errorf("%s: the baseline at %s suppressed nothing — it has drifted off the "+
					"findings it records, and is no longer an acceptance of anything", skill, entry.baseline)
			}
		}
		for _, f := range rep.Findings {
			if strings.HasPrefix(f.RuleID, "SK-T") && !f.Suppressed {
				t.Errorf("%s: unexpected tripwire finding %s at %s:%d (%s)", skill, f.RuleID, f.File, f.Line, f.Evidence)
			}
		}
	}
	if !ran {
		t.Skip("clean corpus not present")
	}
}

// TestSameNameCollisionFires pins the SK-I002 collision leg: two SKILL.md
// files in one package claiming the same name shadow each other silently
// under Cursor's undocumented root precedence — the gate must name it.
func TestSameNameCollisionFires(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"skills/one/SKILL.md": "---\nname: dup\ndescription: first copy\n---\nA.\n",
		"skills/two/SKILL.md": "---\nname: dup\ndescription: second copy\n---\nB.\n",
	})
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range rep.Findings {
		if f.RuleID == "SK-I002" && strings.Contains(f.Message, "multiple skills") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected an SK-I002 same-name collision finding")
	}

	// Distinct names in the same package must not collide.
	root2 := writeBundle(t, map[string]string{
		"skills/one/SKILL.md": "---\nname: one\ndescription: first\n---\nA.\n",
		"skills/two/SKILL.md": "---\nname: two\ndescription: second\n---\nB.\n",
	})
	rep2, err := NewEngine().Gate(root2, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range rep2.Findings {
		if f.RuleID == "SK-I002" && strings.Contains(f.Message, "multiple skills") {
			t.Fatal("distinct names must not produce a collision finding")
		}
	}
}

// TestReportCarriesBashLegSkip pins F14: every report names the open
// pre-activation bash leg in checks_skipped — never claimed closed — and
// the verdict therefore never exceeds CAUTION today.
func TestReportCarriesBashLegSkip(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"SKILL.md": "---\nname: demo\ndescription: a demo skill\n---\nUse it.\n",
	})
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, s := range rep.ChecksSkipped {
		if s.Check == CheckPreactivationBashLeg {
			found = true
		}
	}
	if !found {
		t.Fatal("checks_skipped must name the open bash leg (F14)")
	}
	if rep.Verdict == VerdictApprove {
		t.Fatal("APPROVE is unreachable while the bash leg is open (F14)")
	}
	// --fail-on-incomplete covers bundle coverage, not the standing
	// boundary skip — it must not force exit 1 on its own.
	rep.ChecksSkipped = []SkippedCheck{{Check: CheckPreactivationBashLeg, Reason: "boundary"}}
	if ExitCode(rep, Options{FailOnIncomplete: true}) != ExitPass {
		t.Fatal("the standing F14 boundary skip must not trip --fail-on-incomplete")
	}
}
