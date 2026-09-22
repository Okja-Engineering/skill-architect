package skillgate

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"time"
)

// Pack E — always-on token budget via skill-validator (`check -o json`),
// which returns exact o200k_base counts per file. Conformance errors and
// warnings map to advisory findings; token counts land in report.tokens
// with basis "measured". The per-item leg (skill description lines, tool
// description/guideline text, tool input schemas — the two [pi] classes a
// file list misses) is computed in-process in budget.go and attached to the
// same report.tokens. Named skip when absent; advisory only.

const skillValidatorTimeout = 60 * time.Second

type svTokenFile struct {
	File   string `json:"file"`
	Tokens int    `json:"tokens"`
}

type svReport struct {
	Passed   bool `json:"passed"`
	Errors   int  `json:"errors"`
	Warnings int  `json:"warnings"`
	Tokens   struct {
		Files []svTokenFile `json:"files"`
		Total int           `json:"total"`
	} `json:"token_counts"`
}

// runSkillValidator returns findings, a token budget (may be nil), and a
// skip. It feeds both Pack E (budget) and the conformance floor.
func runSkillValidator(l *Ledger, t *Target) ([]Finding, *TokenBudget, *SkippedCheck) {
	bin, err := exec.LookPath("skill-validator")
	if err != nil {
		return nil, nil, &SkippedCheck{
			Check:  "skill-validator",
			Reason: "binary not found — brew install; token budget and conformance unmeasured; verdict capped at CAUTION",
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), skillValidatorTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "check", "-o", "json", l.Root)
	out, err := cmd.Output()
	if err != nil {
		reason := "check failed: " + err.Error()
		if ctx.Err() == context.DeadlineExceeded {
			reason = "check timed out after " + skillValidatorTimeout.String()
		}
		return nil, nil, &SkippedCheck{Check: "skill-validator", Reason: reason}
	}
	var rep svReport
	if err := json.Unmarshal(out, &rep); err != nil {
		return nil, nil, &SkippedCheck{Check: "skill-validator", Reason: "report unparseable: " + err.Error()}
	}

	var findings []Finding
	if rep.Errors > 0 {
		findings = append(findings, Finding{
			RuleID:        "SV-CONFORM",
			Severity:      SeverityMedium,
			Quality:       "maintainability",
			Message:       "skill-validator reports spec conformance errors",
			Evidence:      strconv.Itoa(rep.Errors) + " errors, " + strconv.Itoa(rep.Warnings) + " warnings",
			EffortMinutes: 15,
			Source:        "skill-validator",
			Advisory:      true,
		})
	}
	var budget *TokenBudget
	if rep.Tokens.Total > 0 {
		budget = &TokenBudget{
			TotalTokens: rep.Tokens.Total,
			Counter:     "skill-validator/o200k_base",
			Basis:       "measured",
		}
		for _, f := range rep.Tokens.Files {
			budget.Files = append(budget.Files, TokenPerFile{File: f.File, Tokens: f.Tokens})
		}
	}
	return findings, budget, nil
}
