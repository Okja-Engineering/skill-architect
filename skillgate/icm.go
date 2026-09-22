package skillgate

import (
	"path"
	"sort"
	"strconv"
	"strings"
)

// Pack-ICM — the mechanical slice of ICM conformance, from
// skills/skill-audit/references/evaluation-matrix.md. Only checks that are
// deterministic get a rule; judgment dimensions (scope, trigger quality,
// examples) stay with skill-audit the skill, not the gate.
//
// SK-I001 frontmatter completeness   · name + description present
// SK-I002 name/dir agreement          · frontmatter name == bundle dir
// SK-I003 description length          · spec hard limit 1024 chars
// SK-I004 body size                   · progressive disclosure: ≤500 lines
// SK-I005 body token budget           · needs a measured counter (skipped named otherwise)

const icmDescriptionLimit = 1024
const icmBodyLineLimit = 500
const icmBodyTokenLimit = 8000 // stage context ceiling from the matrix (~2k–8k)

// icmCheck is a deterministic, in-repo check — not an external scanner. It
// runs after externals so the measured token budget can feed SK-I005; when
// no counter ran, that sub-check is a named skip, not silence.
func icmCheck(l *Ledger, budget *TokenBudget) ([]Finding, []SkippedCheck) {
	var findings []Finding
	var skipped []SkippedCheck

	skills := skillFiles(l)
	for _, sf := range skills {
		fm := ParseFrontmatter(sf.Text)
		base := path.Base(l.Root)
		if d := path.Dir(sf.Entry.Path); d != "." {
			base = path.Base(d)
		}

		// SK-I001
		var missing []string
		for _, k := range []string{"name", "description"} {
			if strings.TrimSpace(fm.Keys[k]) == "" {
				missing = append(missing, k)
			}
		}
		if len(missing) > 0 {
			findings = append(findings, Finding{
				RuleID: "SK-I001", Severity: SeverityMedium, Quality: "maintainability",
				Message: "frontmatter missing required keys: " + strings.Join(missing, ", "),
				File:    sf.Entry.Path, EffortMinutes: 10, Source: "skillgate",
			})
		}

		// SK-I002 — name/dir agreement plus the name charset. Cursor's stock
		// skills system requires `name` to match the parent folder and to be
		// lowercase-hyphenated; both legs are ours to check (agnix CUR-*
		// covers .mdc only).
		if n := fm.Keys["name"]; n != "" {
			if !validSkillName(n) {
				findings = append(findings, Finding{
					RuleID: "SK-I002", Severity: SeverityLow, Quality: "maintainability",
					Message: "frontmatter name must be lowercase alphanumeric + hyphens (≤64 chars)",
					File:    sf.Entry.Path, Evidence: n,
					EffortMinutes: 5, Source: "skillgate",
				})
			}
			if n != base {
				findings = append(findings, Finding{
					RuleID: "SK-I002", Severity: SeverityLow, Quality: "maintainability",
					Message: "frontmatter name does not match directory name",
					File:    sf.Entry.Path, Evidence: n + " vs " + base,
					EffortMinutes: 5, Source: "skillgate",
				})
			}
		}

		// SK-I003 — spec hard limit on description.
		if d := fm.Keys["description"]; len(d) > icmDescriptionLimit {
			findings = append(findings, Finding{
				RuleID: "SK-I003", Severity: SeverityMedium, Quality: "maintainability",
				Message: "description exceeds the 1024-char spec limit",
				File:    sf.Entry.Path, Evidence: strconv.Itoa(len(d)) + " chars",
				EffortMinutes: 10, Source: "skillgate",
			})
		}

		// SK-I004 — body line count as the progressive-disclosure floor.
		body := strings.TrimSpace(Body(sf.Text))
		if body != "" {
			lines := strings.Count(body, "\n") + 1
			if lines > icmBodyLineLimit {
				findings = append(findings, Finding{
					RuleID: "SK-I004", Severity: SeverityMedium, Quality: "maintainability",
					Message: "SKILL.md body exceeds 500 lines — move depth into references/",
					File:    sf.Entry.Path, Evidence: strconv.Itoa(lines) + " lines",
					EffortMinutes: 30, Source: "skillgate",
				})
			}
		}
	}

	// SK-I002 collision leg — two SKILL.md files in one package claiming the
	// same name. Cursor walks eight roots (native plus .claude/.codex
	// compatibility) with undocumented precedence, so a same-name pair
	// shadows silently; inside one package the collision is visible
	// statically. Cross-root collisions on a live estate are `skillgate
	// env`'s job (slice 2).
	byName := map[string][]string{}
	for _, sf := range skills {
		if !strings.HasSuffix(sf.Entry.Path, "SKILL.md") {
			continue
		}
		if n := ParseFrontmatter(sf.Text).Keys["name"]; n != "" {
			byName[n] = append(byName[n], sf.Entry.Path)
		}
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if paths := byName[name]; len(paths) > 1 {
			findings = append(findings, Finding{
				RuleID: "SK-I002", Severity: SeverityLow, Quality: "maintainability",
				Message: "name claimed by multiple skills in one package — silent shadow under Cursor's undocumented root precedence",
				File:    paths[1], Evidence: name + ": " + strings.Join(paths, ", "),
				EffortMinutes: 10, Source: "skillgate",
			})
		}
	}

	// SK-I005 — measured body tokens vs the stage-context ceiling. The skip
	// must key on per-file token counts, not budget != nil: items[]-only
	// budgets (exact chars, no tokenizer) leave Files empty — that case is
	// still "unmeasured" and must name the skip (F13).
	if budget == nil || len(budget.Files) == 0 {
		skipped = append(skipped, SkippedCheck{
			Check:  "icm-token-budget",
			Reason: "no token counter ran (skill-validator absent or failed) — body token cost unmeasured",
		})
	} else {
		for _, tf := range budget.Files {
			if strings.HasSuffix(tf.File, "SKILL.md") || strings.Contains(tf.File, "SKILL.md body") {
				if tf.Tokens > icmBodyTokenLimit {
					findings = append(findings, Finding{
						RuleID: "SK-I005", Severity: SeverityMedium, Quality: "maintainability",
						Message: "SKILL.md body exceeds the ~8k-token stage-context ceiling",
						File:    tf.File, Evidence: strconv.Itoa(tf.Tokens) + " tokens (measured)",
						EffortMinutes: 45, Source: "skillgate",
					})
				}
			}
		}
	}
	return findings, skipped
}
