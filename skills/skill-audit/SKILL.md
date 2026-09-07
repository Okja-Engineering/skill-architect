---
name: skill-audit
description: Evaluate an Agent Skill directory against the Agent Skills spec, Anthropic best practices, and the Interpretable Context Methodology (ICM). Use when reviewing a SKILL.md before release, after a major edit, or when a skill is not triggering or executing reliably.
license: MIT
compatibility: POSIX shell (bash 3.2+ or zsh), git.
metadata:
  version: "0.1.0"
---

# skill-audit

Evaluate an Agent Skill directory against the Agent Skills spec, Anthropic best practices, and the Interpretable Context Methodology (ICM). Produce a pass/fail report with actionable fixes.

## When to use

Use this skill when:

- You are about to release a skill and want a final quality check.
- You have made a major edit to a `SKILL.md` or bundled resource.
- A skill is not triggering reliably (likely a description problem).
- A skill is triggering but executing incorrectly (likely a structure or script problem).
- You are onboarding a skill from prose notes into a spec-compliant format.

Do not use this skill to rewrite the audited skill without explicit approval; it reports and recommends.

## Deterministic actions (60%)

### Stage 0: Orient

Identify the skill to audit. `target_skill` must be one skill directory containing a `SKILL.md`, not a repository root or collection.

### Stage 1: Inspect files

Read these in order:

1. `<skill-dir>/SKILL.md`
2. `<skill-dir>/scripts/*`
3. `<skill-dir>/references/*`
4. `<skill-dir>/assets/*`

### Stage 2: Run structural checks

Resolve `skill_root` to the directory containing this `SKILL.md` and run the bundled scripts:

```bash
"$skill_root/scripts/check-frontmatter.sh" "$target_skill"
"$skill_root/scripts/check-structure.sh" "$target_skill"
```

List bundled resources:

```bash
ls -la "$target_skill/scripts" 2>/dev/null || echo "no scripts/"
ls -la "$target_skill/references" 2>/dev/null || echo "no references/"
ls -la "$target_skill/assets" 2>/dev/null || echo "no assets/"
```

Treat a nonzero script exit as a deterministic failure. Preserve individual findings in the report rather than replacing them with a generic failure.

### Stage 3: Evaluate the 10 dimensions

Score each dimension 0–2 using the matrix in `references/evaluation-matrix.md`:

- 0 = missing or broken
- 1 = present but weak
- 2 = strong

Dimensions: Spec compliance, Trigger, Scope, Body structure, Determinism, Context management, Token discipline, Validation, Self-contained, Examples.

### Stage 4: Produce the report

Use the format in `references/evaluation-matrix.md`. Include:

- Every scoring dimension and its 0–2 score.
- Deterministic command results.
- The 60/30/10 ratio.
- Ordered, concrete fixes tied to failed or weak criteria.

## Orchestration (30%)

### Audit process

1. Confirm `target_skill` is a single skill directory, not a repository or `SKILL.md` file.
2. Inspect `SKILL.md`, then `scripts/`, `references/`, and `assets/` when present.
3. Run the structural scripts and record outputs.
4. Consult `references/best-practices.md` only for criteria requiring interpretation.
5. Score each dimension 0–2.
6. Estimate the deterministic / orchestration / AI-judgment ratio from the body text.
7. Produce the required report.
8. Modify the audited skill only after explicit approval.

## AI judgment (10%)

Use judgment for:

- Scoring the description's trigger quality when it is borderline.
- Deciding whether a multi-step skill needs stage contracts or a simpler linear process.
- Recommending whether a missing element should be added or whether the skill should be split.
- Interpreting "present but weak" vs "strong" for dimensions without hard fail conditions.
- Estimating the 60/30/10 ratio from the body text.

When in doubt, mark the dimension as 1 (present but weak) and explain what would make it a 2.

## Constraints

- Do not modify the audited skill without explicit approval.
- Be specific in recommendations: quote the missing section or give exact text to add.
- Prefer suggesting cuts over additions when a skill is trying to do too many jobs.
- Note honestly where ICM or Agent Skills conventions do not apply.

## Examples

### Audit a skill in the current repo

```bash
skill_root=".devin/skills/skill-audit"
target_skill=".devin/skills/release-check"
"$skill_root/scripts/check-frontmatter.sh" "$target_skill"
"$skill_root/scripts/check-structure.sh" "$target_skill"
```

### Example report opening

```markdown
# Skill audit: release-check

| Dimension | Score | Notes |
|---|---|---|
| Spec compliance | 2 | Directory name matches `name`, description 180 chars, license MIT. |
| Trigger | 2 | Imperative "Use when...", specific contexts, negative scope. |
| ... | ... | ... |

## Ratio

- Deterministic: 60%
- Orchestration: 30%
- AI judgment: 10%

## Fixes

1. Add an `Examples` section with a concrete command and expected output.
```
