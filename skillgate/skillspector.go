package skillgate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// G3 — SkillSpector deep scan, opt-in. The gate shells out to the unmodified
// CLI (`skillspector scan <dir> --no-llm --format json`) and maps its issues
// into findings. It is never load-bearing for REJECT: when the binary is
// absent or fails, the check lands in checks_skipped and the verdict caps at
// CAUTION (F13).
//
// SkillSpector issues use severity names CRITICAL/HIGH/MEDIUM/LOW; rule IDs
// are its own analyzer codes (e.g. AE1). Install is git-URL, Python
// ≥3.12,<3.15 — see docs/research/recommendation.md §4.

const skillSpectorTimeout = 120 * time.Second

type ssLocation struct {
	File      string `json:"file"`
	StartLine int    `json:"start_line"`
	EndLine   *int   `json:"end_line"`
}

type ssIssue struct {
	ID          string       `json:"id"`
	Category    string       `json:"category"`
	Severity    string       `json:"severity"`
	Finding     string       `json:"finding"`
	Explanation string       `json:"explanation"`
	Remediation string       `json:"remediation"`
	Snippet     string       `json:"code_snippet"`
	Location    ssLocation   `json:"location"`
	Occurrences []ssLocation `json:"occurrences"`
}

type ssCompleteness struct {
	Status      string `json:"status"`
	IsComplete  bool   `json:"is_complete"`
	TotalFiles  int    `json:"total_components"`
	Inspected   int    `json:"fully_inspected_files"`
	Uninspected int    `json:"entirely_uninspected_files"`
}

type ssReport struct {
	Issues              []ssIssue      `json:"issues"`
	ExecutionSuccessful bool           `json:"execution_successful"`
	Completeness        ssCompleteness `json:"analysis_completeness"`
}

// runSkillSpector shells out. Findings keep SkillSpector's own rule IDs and
// true severities, but are all Advisory — REJECT stays a Pack-B-owned
// decision and never depends on an optional binary (D2).
func runSkillSpector(l *Ledger, t *Target) ([]Finding, *SkippedCheck) {
	bin, err := exec.LookPath("skillspector")
	if err != nil {
		return nil, &SkippedCheck{
			Check:  "skillspector",
			Reason: "binary not found — install is opt-in (Python ≥3.12,<3.15, git-URL install); verdict capped at CAUTION",
		}
	}

	out, err := os.CreateTemp("", "skillspector-*.json")
	if err != nil {
		return nil, &SkippedCheck{Check: "skillspector", Reason: "temp file: " + err.Error()}
	}
	outPath := out.Name()
	out.Close()
	defer os.Remove(outPath)

	ctx, cancel := context.WithTimeout(context.Background(), skillSpectorTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "scan", l.Root,
		"--no-llm", "--format", "json", "--output", outPath)
	cmd.Dir = l.Root
	// Same conflation as the other two, one step removed: skillspector
	// writes its report to --output rather than stdout, so a non-zero exit
	// never destroyed the file — but the branch on the exit status meant the
	// gate never opened it. A scanner that exits non-zero *because it found
	// something* is the case that matters. See external.go.
	runErr := cmd.Run()
	data, readErr := os.ReadFile(outPath)
	if readErr != nil {
		// No report file at all: nothing to decide from, so the run error is
		// the whole story.
		reason := "scan produced no report: " + readErr.Error()
		if runErr != nil {
			reason = "scan failed (" + runErr.Error() + ") and produced no report: " + readErr.Error()
		}
		if ctx.Err() == context.DeadlineExceeded {
			reason = "scan timed out after " + skillSpectorTimeout.String()
		}
		return nil, &SkippedCheck{Check: "skillspector", Reason: reason}
	}
	var rep ssReport
	if skip := externalOutcome(ctx, "skillspector", "scan", skillSpectorTimeout, runErr, data, &rep); skip != nil {
		return nil, skip
	}

	var findings []Finding
	for _, is := range rep.Issues {
		sev := ssSeverity(is.Severity)
		if sev == "" {
			continue
		}
		locs := is.Occurrences
		if len(locs) == 0 {
			locs = []ssLocation{is.Location}
		}
		for _, loc := range locs {
			file := loc.File
			if rel, err := filepath.Rel(l.Root, file); err == nil {
				file = filepath.ToSlash(rel)
			}
			findings = append(findings, Finding{
				RuleID:        "SS-" + is.ID,
				Severity:      sev,
				Quality:       "security",
				Message:       firstNonEmpty(is.Explanation, is.Category, "skillspector issue"),
				File:          file,
				Line:          loc.StartLine,
				Evidence:      truncate(firstNonEmpty(is.Snippet, is.Finding), 160),
				EffortMinutes: 15, // uncalibrated default — minutes only, no grade (D6)
				Source:        "skillspector",
				Advisory:      true,
			})
		}
	}
	return findings, nil
}

// ssSeverity maps SkillSpector severities onto ours. CRITICAL maps to
// Blocker severity for display, but Advisory findings never block.
func ssSeverity(s string) string {
	switch strings.ToUpper(s) {
	case "CRITICAL":
		return SeverityBlocker
	case "HIGH":
		return SeverityHigh
	case "MEDIUM":
		return SeverityMedium
	case "LOW":
		return SeverityLow
	case "INFO", "INFORMATIONAL":
		return SeverityInfo
	}
	return ""
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}
