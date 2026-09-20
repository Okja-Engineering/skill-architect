---
name: skill-rewrite
description: Draft a rewritten SKILL.md for an Agent Skill from a skill-audit report. Use when a skill scored 0 or 1 on any audit dimension, when skillscore or skill-validator reports warnings or failures, or when asked to refactor, restructure, fix, or improve an existing SKILL.md before editing it. Produces a reviewable draft and rewrite plan only; never modifies the target skill without explicit approval. Not for creating a new skill from scratch.
license: MIT
compatibility: POSIX shell (bash 3.2+ or zsh), git, jq. skill-validator and skillscore only when no audit report is supplied.
allowed-tools: Read, Bash
metadata:
  version: "0.2.0"
---

# skill-rewrite

Draft a rewritten `SKILL.md` for an Agent Skill from a `skill-audit` report. This skill turns audit findings into a concrete rewrite plan and a draft; it does not modify the audited skill without explicit approval.

## When to use

Use this skill when:

- A skill has one or more dimensions scored 0 or 1 in a `skill-audit` report.
- You need a structured rewrite plan before editing `SKILL.md`.
- You want to preserve the skill's intent while fixing structural, trigger, or context-management issues.

Do not use this skill to apply changes silently; the draft must be reviewed and approved first. Do not use it to create a skill from scratch.

## Deterministic actions (60%)

### Stage 0: Orient

Inputs:

- `target_skill`: one skill directory containing `SKILL.md`.
- `audit_report`: path to an existing `skill-audit` report (markdown). If omitted, Stage 1 generates one.
- `skill_root`: the directory containing this `SKILL.md` and `scripts/`.
- `draft_out`: where the draft is written. Must be outside `target_skill`.

Preflight:

```bash
[[ -f "$target_skill/SKILL.md" ]] || { echo "not a skill dir: $target_skill" >&2; exit 1; }
[[ -x "$skill_root/scripts/audit-report.sh" ]] || { echo "vendored audit scripts missing under $skill_root/scripts" >&2; exit 3; }
```

Outputs: resolved `target_skill`, `skill_root`, `draft_out`, and either `audit_report` or a decision to run Stage 1.

### Stage 1: Run the audit if no report was supplied

```bash
audit_report="$draft_out.audit.json"
"$skill_root/scripts/audit-report.sh" "$target_skill" > "$audit_report" || echo "audit exited $?" >&2
jq '.summary' "$audit_report"
```

Score the 10 dimensions using `references/evaluation-matrix.md` (a vendored copy of skill-audit's matrix).

Outputs: `audit_report` with a 10-dimension score table.

### Stage 2: Generate the rewrite draft

```bash
"$skill_root/scripts/draft-rewrite.sh" -t "$target_skill" -a "$audit_report" -o "$draft_out"
```

The script writes `draft_out` containing:

- The audit report inlined under `## Current state`.
- The proposed section skeleton (Agent Skills spec + ICM).
- Templates for any of `When to use`, `Examples`, `Validation` that the target lacks.
- One checklist item per dimension scored 0 or 1 when the report contains the audit score table; otherwise generic action items.

Never write the draft inside `target_skill`; that dirties the tree being audited before approval.

Outputs: `draft_out`.

## Orchestration (30%)

### Inputs

- `target_skill`, `audit_report` (or none), `skill_root`, `draft_out`.

### Rewrite process

1. Confirm `target_skill` is a single skill directory and resolve `draft_out` outside it.
2. Read the current `SKILL.md`, then `scripts/`, `references/`, and `assets/` when present.
3. Order fixes by impact:
   - Spec compliance failures (frontmatter, layout).
   - Trigger failures (description missing what/when/keywords).
   - Body structure failures (missing headings, >500 lines, no examples).
   - Determinism failures (missing scripts, bad paths, no error handling).
4. Run Stage 2.
5. Produce the rewrite plan from the draft: sections added/moved/removed; scripts or references to create, rename, or delete; content preserved unchanged; items needing human judgment.
6. Present the plan and draft to the maintainer and ask for approval.

### After approval

7. Apply only the approved changes to `target_skill`.
8. Re-run `skill-audit` and confirm no dimension scored lower than before.

### Outputs

- A rewrite plan (four parts, step 5) and a rewritten `SKILL.md` draft, both at `draft_out`.
- Nothing inside `target_skill` until step 7.

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

## Validation

Before presenting the draft, confirm:

- [ ] Every audit dimension scored 0 or 1 maps to at least one change in the draft.
- [ ] No content was added that is not traceable to an audit finding or the original `SKILL.md`.
- [ ] Every existing working script is preserved or listed as removed with a reason.
- [ ] `git status --porcelain` inside `target_skill` is empty.

After approval and apply:

- [ ] Re-audit scores are >= the original on every dimension.

## Constraints

- Do not overwrite the original `SKILL.md` without explicit approval.
- Do not guess the maintainer's intent for ambiguous fixes.
- Preserve all working scripts unless they are provably broken or duplicated.
- Keep the skill focused on one job; prefer splitting over adding scope.
- Every rewrite must be followed by a re-audit.

## Examples

### Generate a rewrite draft from an existing report

```bash
skill_root="skills/skill-rewrite"
target_skill="skills/release-check"
draft_out="REWRITE-DRAFT-release-check.md"
"$skill_root/scripts/draft-rewrite.sh" -t "$target_skill" -a ./release-check-audit.md -o "$draft_out"
```

Output: `Rewrite draft written to: REWRITE-DRAFT-release-check.md`

### Example draft excerpt

```markdown
# Rewrite draft: release-check

Generated from audit report: ./release-check-audit.md

## Current state
| Trigger | 1 | No synonyms; no negative scope. |
| Examples | 0 | No concrete command shown. |

## Missing section templates
### Examples
#### Example 1: <scenario>
...

## Action items
- [ ] Trigger (score 1): add synonyms and a "Not for..." clause to the description.
- [ ] Examples (score 0): add one runnable command with expected output.
```

### Rewrite plan outline

```text
REWRITE-DRAFT-release-check.md
- Preserve: frontmatter name, description intent, existing scripts.
- Add: When to use, Examples, Validation checklist.
- Move: deep reference content to references/<topic>.md.
- Remove: duplicate Process section under Orchestration.
- Rename: scripts/<old>.sh to scripts/<new>.sh to match SKILL.md text.
```
