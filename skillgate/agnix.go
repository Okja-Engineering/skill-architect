package skillgate

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// G4 — agnix conformance scan, opt-in. Shells out to the unmodified Rust CLI
// (`agnix --target generic --format json <dir>`) covering CC/Cursor/MCP/
// AGENTS.md rules. Same contract as SkillSpector: advisory-only findings,
// named skip when absent, never load-bearing for REJECT (D3).
//
// Coverage note [pi]: agnix's CUR-* rules cover Cursor `.mdc` files only —
// Cursor's stock SKILL.md conformance (eight roots, name-matches-folder,
// undocumented precedence) is checked natively in icm.go/frontmatter.go,
// not delegated here.

const agnixTimeout = 60 * time.Second

type agnixDiag struct {
	Level        string `json:"level"`
	Rule         string `json:"rule"`
	File         string `json:"file"`
	Line         int    `json:"line"`
	Message      string `json:"message"`
	Suggestion   string `json:"suggestion"`
	Category     string `json:"category"`
	RuleSeverity string `json:"rule_severity"`
}

type agnixReport struct {
	Version      string      `json:"version"`
	FilesChecked int         `json:"files_checked"`
	Diagnostics  []agnixDiag `json:"diagnostics"`
}

// runAgnix shells out. Diagnostics map by level: error→High, warning→Medium,
// info→Low — all Advisory, so agnix can warn but never REJECT.
func runAgnix(l *Ledger, t *Target) ([]Finding, *SkippedCheck) {
	bin, err := exec.LookPath("agnix")
	if err != nil {
		return nil, &SkippedCheck{
			Check:  "agnix",
			Reason: "binary not found — opt-in Rust CLI (github.com/agent-sh/agnix); verdict capped at CAUTION",
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), agnixTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--target", "generic", "--format", "json", l.Root)
	// Same shape as skill-validator's, and repaired for the same reason: a
	// linter that reports diagnostics by exiting non-zero would have had its
	// report discarded by the branch that collected it. See external.go.
	out, runErr := cmd.Output()
	var rep agnixReport
	if skip := externalOutcome(ctx, "agnix", "scan", agnixTimeout, runErr, out, &rep); skip != nil {
		return nil, skip
	}
	var findings []Finding
	for _, d := range rep.Diagnostics {
		sev := SeverityInfo
		switch strings.ToLower(d.Level) {
		case "error":
			sev = SeverityHigh
		case "warning":
			sev = SeverityMedium
		case "info":
			sev = SeverityLow
		}
		file := d.File
		if rel, err := filepath.Rel(l.Root, file); err == nil {
			file = filepath.ToSlash(rel)
		}
		msg := d.Message
		if d.Suggestion != "" {
			msg += " — " + d.Suggestion
		}
		findings = append(findings, Finding{
			RuleID:        "AG-" + d.Rule,
			Severity:      sev,
			Quality:       "maintainability",
			Message:       msg,
			File:          file,
			Line:          d.Line,
			EffortMinutes: 15,
			Source:        "agnix",
			Advisory:      true,
		})
	}
	return findings, nil
}
