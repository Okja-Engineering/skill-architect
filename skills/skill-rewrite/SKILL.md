---
name: skill-rewrite
description: Draft a rewritten SKILL.md for an Agent Skill based on a skill-audit report. Use when a skill has failed or weak audit dimensions and you need a concrete rewrite plan before editing. The skill does not apply changes without explicit approval.
license: MIT
compatibility: POSIX shell (bash 3.2+ or zsh), git.
metadata:
  version: "0.1.0"
---

# skill-rewrite

Draft a rewritten `SKILL.md` for an Agent Skill based on a `skill-audit` report. This skill turns audit findings into a concrete rewrite plan; it does not modify the audited skill without explicit approval.

## When to use

Use this skill when:

- A skill has one or more failed or weak dimensions in a `skill-audit` report.
- You are converting a prose workflow into a spec-compliant Agent Skill.
- You need a structured rewrite plan before editing `SKILL.md`.
- You want to preserve the skill's intent while fixing structural, trigger, or context-management issues.

Do not use this skill to apply changes silently; the draft must be reviewed and approved first.

## Deterministic actions (60%)

### Stage 0: Orient

Inputs:

- `target_skill`: the skill directory to rewrite.
- `audit_report`: optional path to an existing audit report. If omitted, run `skill-audit` first.

### Stage 1: Run audit if needed

If no audit report is provided, run the audit:

```bash
skill_root="<path-to-skill-architect>/skills/skill-audit"
target_skill="<target-skill-dir>"
"$skill_root/scripts/check-frontmatter.sh" "$target_skill"
"$skill_root/scripts/check-structure.sh" "$target_skill"
```

Capture the output and score the 10 dimensions using `references/evaluation-matrix.md` from `skill-audit`.

### Stage 2: Generate rewrite draft

Run the rewrite drafter:

```bash
scripts/draft-rewrite.sh -t <target-skill-dir> [-a <audit-report-path>]
```

This creates a `REWRITE-DRAFT.md` next to the target skill's `SKILL.md` with:

- Preserved frontmatter (with corrected `name` if mismatched).
- A proposed structure following the Agent Skills spec and ICM principles.
- Templates for missing sections: `When to use`, `Deterministic actions`, `Orchestration`, `Examples`, `Constraints`.
- A checklist mapping each failed/weak audit dimension to a concrete fix.

### Stage 3: Produce rewrite plan

Read the draft and produce a final rewrite plan that includes:

1. What sections will be added, moved, or removed.
2. What scripts or references need to be created, renamed, or deleted.
3. What content is preserved unchanged.
4. What requires human judgment (e.g., trigger phrasing, scope boundaries).

## Orchestration (30%)

### Rewrite process

1. Confirm the target skill directory and locate or generate the audit report.
2. Read the current `SKILL.md`, `scripts/`, `references/`, and `assets/`.
3. Identify the highest-impact fixes first:
   - Spec compliance failures (frontmatter, layout).
   - Trigger failures (description missing what/when/keywords).
   - Body structure failures (missing headings, >500 lines, no examples).
   - Determinism failures (missing scripts, bad paths, no error handling).
4. Generate the rewrite draft using `draft-rewrite.sh`.
5. Present the draft to the maintainer and ask for approval.
6. After approval, apply the rewrite. If approval is partial, apply only the approved changes.
7. Re-run `skill-audit` and confirm the scores improved.

### Content preservation rules

- Keep the skill's original intent and domain knowledge.
- Move mechanical work into `scripts/`; do not delete useful scripts unless they are duplicated or broken.
- Move deep reference material into `references/` or `assets/`.
- Do not inflate `SKILL.md` beyond ~500 lines; split instead of grow.

## AI judgment (10%)

Use judgment for:

- Deciding whether a missing `When to use` section should list three scenarios or five.
- Rewriting the description to be "pushy" without being misleading.
- Choosing whether a skill should be split rather than expanded.
- Summarizing domain-specific content into the new structure without losing nuance.
- Deciding which adjacent contexts belong in the description's trigger list.

When in doubt, keep the draft conservative and flag the uncertainty for the maintainer.

## Constraints

- Do not overwrite the original `SKILL.md` without explicit approval.
- Do not guess the maintainer's intent for ambiguous fixes.
- Preserve all working scripts unless they are provably broken or duplicated.
- Keep the skill focused on one job; prefer splitting over adding scope.
- Every rewrite must be followed by a re-audit.

## Examples

### Generate a rewrite draft

```bash
scripts/draft-rewrite.sh -t skills/release-check -a ./release-check-audit.md
```

Output: `skills/release-check/REWRITE-DRAFT.md`.

### Rewrite plan outline

```text
skills/release-check/REWRITE-DRAFT.md
- Preserve: frontmatter name, description intent, existing scripts.
- Add: When to use, Examples, Validation checklist.
- Move: deep reference content to references/validation-patterns.md.
- Remove: duplicate Process section under Orchestration.
- Rename: scripts/check.sh to scripts/validate.sh to match SKILL.md text.
```
