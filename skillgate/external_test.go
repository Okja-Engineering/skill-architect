package skillgate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// External scanners — what a non-zero exit means.
//
// The defect: SV-CONFORM was gated on `rep.Errors > 0`, and skill-validator
// reports errors *by exiting 1*. The only path that could set rep.Errors was
// the path that returned a skip and threw the report away, so the finding
// was unreachable by construction. Measured across 63 bundles in four
// estates (S14): zero SV-CONFORM findings ever, seven bundles with errors
// the gate had already read and dropped.
//
// None of this repository's own three skills exits non-zero, so the defect
// was invisible to self-dogfooding — which is the argument for gating skills
// you did not author, proved on the gate itself.

// fakeScanner puts an executable named `name` at the front of PATH for the
// duration of the test. script is a /bin/sh body.
func fakeScanner(t *testing.T, name, script string) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// a report skill-validator would emit for a bundle with two conformance
// errors — and it emits it on stdout while exiting 1.
const svErrorReport = `{"passed":false,"errors":2,"warnings":1,` +
	`"token_counts":{"files":[{"file":"SKILL.md body","tokens":42}],"total":42}}`

func svBundle(t *testing.T) *Ledger {
	t.Helper()
	root := writeBundle(t, map[string]string{
		"SKILL.md": "---\nname: demo\ndescription: a demo skill\n---\nUse it.\n",
	})
	l, err := BuildLedger(root)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// TestExitStatusDoesNotDecide walks all four combinations of exit status and
// report readability. The output decides; the exit status only ever colours
// the skip's wording.
func TestExitStatusDoesNotDecide(t *testing.T) {
	report := `printf '` + svErrorReport + `'; `
	cases := []struct {
		name        string
		script      string
		wantSkip    bool
		wantFinding bool
	}{
		{
			name:        "exit 0, parseable — a result, as it always was",
			script:      report + "exit 0",
			wantSkip:    false,
			wantFinding: true,
		},
		{
			name:        "exit 1, parseable — a result, and this is the case that was lost",
			script:      report + "exit 1",
			wantSkip:    false,
			wantFinding: true,
		},
		{
			name:        "exit 0, unparseable — a skip, because nothing was reported",
			script:      `printf 'not json at all'; exit 0`,
			wantSkip:    true,
			wantFinding: false,
		},
		{
			name:        "exit 127, unparseable — a skip naming the exit",
			script:      `exit 127`,
			wantSkip:    true,
			wantFinding: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fakeScanner(t, "skill-validator", c.script)
			findings, budget, skip := runSkillValidator(svBundle(t), nil)

			if c.wantSkip && skip == nil {
				t.Fatal("want a named skip, got none — a scanner that reported nothing " +
					"must never read as a clean pass")
			}
			if !c.wantSkip && skip != nil {
				t.Fatalf("want a result, got skip %q", skip.Reason)
			}
			if skip != nil && strings.TrimSpace(skip.Reason) == "" {
				t.Error("a skip with no reason is silence, not a skip")
			}
			var sv bool
			for _, f := range findings {
				if f.RuleID == "SV-CONFORM" {
					sv = true
				}
			}
			if sv != c.wantFinding {
				t.Errorf("SV-CONFORM present = %v, want %v", sv, c.wantFinding)
			}
			// The token budget rides in the same report. Losing the report
			// lost the measurement too, on exactly the bundles that were
			// non-conformant.
			if c.wantFinding && (budget == nil || budget.TotalTokens != 42) {
				t.Errorf("token budget = %+v, want the 42 tokens the same report carried", budget)
			}
		})
	}
}

// TestDeadlineIsAlwaysASkip pins the one rule that outranks the output: a
// truncated report is not a report, however well the bytes that arrived
// happen to parse. "Did not finish" is never "clean".
func TestDeadlineIsAlwaysASkip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()

	var rep svReport
	skip := externalOutcome(ctx, "skill-validator", "check", time.Second, nil,
		[]byte(svErrorReport), &rep)
	if skip == nil {
		t.Fatal("a deadline must be a skip even when the bytes that arrived parse cleanly")
	}
	if !strings.Contains(skip.Reason, "timed out") {
		t.Errorf("skip reason %q does not say it timed out", skip.Reason)
	}
}

// TestBrokenScannerNeverReadsAsClean is the direction that matters most: a
// gate reporting "clean" because its scanner crashed is worse than the bug
// this fixes. Every failure mode must leave a named skip behind.
func TestBrokenScannerNeverReadsAsClean(t *testing.T) {
	for _, c := range []struct{ name, script string }{
		{"crashes with no output", `exit 3`},
		{"prints a stack trace to stderr", `echo "Traceback..." >&2; exit 1`},
		{"prints half a JSON document", `printf '{"errors":2,'; exit 1`},
		{"prints a shell error", `printf 'skill-validator: unknown flag\n'; exit 2`},
		{"succeeds but prints nothing at all", `exit 0`},
	} {
		t.Run(c.name, func(t *testing.T) {
			fakeScanner(t, "skill-validator", c.script)
			findings, _, skip := runSkillValidator(svBundle(t), nil)
			if skip == nil {
				t.Fatal("no skip — the gate would report this bundle as having been checked")
			}
			if strings.TrimSpace(skip.Reason) == "" {
				t.Error("skip carries no reason")
			}
			if skip.Check != "skill-validator" {
				t.Errorf("skip names %q, want skill-validator", skip.Check)
			}
			if len(findings) != 0 {
				t.Errorf("a scanner that produced no report must produce no findings, got %v", findings)
			}
		})
	}
}

// TestSVConformIsReachableThroughTheGate proves it end to end rather than at
// the function boundary: the published rule fires in a real report.
func TestSVConformIsReachableThroughTheGate(t *testing.T) {
	fakeScanner(t, "skill-validator", `printf '`+svErrorReport+`'; exit 1`)
	root := writeBundle(t, map[string]string{
		"SKILL.md": "---\nname: demo\ndescription: a demo skill\n---\nUse it.\n",
	})
	rep, err := NewEngine().Gate(root, Options{SkipChecks: []string{"skillspector", "agnix"}})
	if err != nil {
		t.Fatal(err)
	}
	var found *Finding
	for i := range rep.Findings {
		if rep.Findings[i].RuleID == "SV-CONFORM" {
			found = &rep.Findings[i]
		}
	}
	if found == nil {
		t.Fatal("SV-CONFORM did not reach the report — it is published in the rule " +
			"catalog and must be reachable")
	}
	if !strings.Contains(found.Evidence, "2 errors") {
		t.Errorf("evidence %q does not carry the validator's own count", found.Evidence)
	}
	for _, s := range rep.ChecksSkipped {
		if s.Check == "skill-validator" {
			t.Errorf("skill-validator both reported and was recorded as skipped: %q", s.Reason)
		}
	}
	if rep.Tokens == nil || rep.Tokens.Basis != "measured" {
		t.Errorf("the token budget rides in the same report and must survive with it: %+v", rep.Tokens)
	}
}
