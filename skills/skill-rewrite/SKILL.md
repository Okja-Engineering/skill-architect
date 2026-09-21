---
name: skill-rewrite
description: Draft a rewritten SKILL.md for an Agent Skill based on a skill-audit report. Use when a skill has failed or weak audit dimensions and you need a concrete rewrite plan before editing. The skill does not apply changes without explicit approval.
license: MIT
compatibility: bash 3.2+.
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

- `skill_root`: the directory holding this `SKILL.md` — the `skill-rewrite` skill directory. Every command below is anchored on it, directly or through `audit_root`, so the commands run from any working directory rather than only from the skill's own.
- `audit_root`: the sibling `skill-audit` skill directory, `"$skill_root/../skill-audit"`. `skill-rewrite` runs `skill-audit`'s check scripts and reads its evaluation matrix; it bundles neither.
- `target_skill`: the skill directory to rewrite.
- `audit_report`: optional path to an existing audit report. If omitted, run `skill-audit` first.
- `output`: optional path to write the draft to. If omitted, the draft goes beside the target's `SKILL.md`. See "Where the draft is written" under Stage 2 for the four destinations it will not write to.

Prerequisites:

Required tools: `awk`, `cat`, `grep`, `jq`, `mktemp`, `skill-validator`, `wc`.

Stage 1 and `draft-rewrite.sh` both run `skill-audit`'s `check-frontmatter.sh`, which validates the spec with `skill-validator`. Install it with `brew install agent-ecosystem/tap/skill-validator`, because without it neither path will draft for you:

Without `skill-validator`: `check-frontmatter.sh` exits `3`; `draft-rewrite.sh` exits `3`.

Stage 1 runs the check directly, so it exits 3 with `required tool not found: skill-validator` on stderr and reports no verdict rather than one it could not compute. `draft-rewrite.sh` is inside that same guard now: it reads each check's exit status instead of discarding it, captures each check's stdout on its own instead of merging the check's stderr into it, and a check that reached no verdict stops the draft rather than becoming its content. So it exits 3 too, with `required tool not found: skill-validator` on stderr, and it writes no `REWRITE-DRAFT.md` — the partial one it had opened is removed on the way out, so there is no half-draft in your skill directory either. **What earlier releases warned about here — a drafter that exited 0 and presented `required tool not found: skill-validator` as the draft's own `Current state` — is closed, not something to work around.** A draft you are handed is a draft built from an audit that ran.

The other six required tools refuse the same way, so this is the skill's behaviour and not one tool's special case. `awk`, `grep` and `cat` are what this skill computes with, and `mktemp` holds the audit when you do not hand it one with `-a`; `jq` and `wc` are preconditions of the two sibling checks. Mask any one of the seven and `draft-rewrite.sh` exits 3 naming that tool and leaves no draft. Nothing on this skill's path needs `skillscore`, which scores quality this skill does not run; add it if you extend the stages to use it.

This skill is not self-contained. It bundles `draft-rewrite.sh` and runs no check of its own: every check it runs comes from `$audit_root/scripts/` — `check-frontmatter.sh` and `check-structure.sh`, which both load `verdict-guard.sh` and refuse to compute a verdict without it. `draft-rewrite.sh` loads that guard too, which is how it refuses in the guard's own words rather than in the shell's; it sources the guard rather than running it, which is why the guard is not one of the checks named above. A pruner who removes the guard as an unused file, or who installs this skill alone, gets no draft at all. Install or prune the two skills together.

### Stage 1: Run audit if needed

If no audit report is provided, run the audit:

```bash
skill_root="<path-to-skill-architect>/skills/skill-rewrite"
audit_root="$skill_root/../skill-audit"
target_skill="<target-skill-dir>"
"$audit_root/scripts/check-frontmatter.sh" "$target_skill"
"$audit_root/scripts/check-structure.sh" "$target_skill"
```

Capture the output and score the 10 dimensions against `"$audit_root/references/evaluation-matrix.md"`.

### Stage 2: Generate rewrite draft

Run the rewrite drafter. Without `-a` it runs the Stage 1 checks itself; with `-a` it reads the report you already have; with `-o` it writes the draft where you say instead of into the target skill's directory:

```bash
skill_root="<path-to-skill-architect>/skills/skill-rewrite"
target_skill="<target-skill-dir>"
"$skill_root/scripts/draft-rewrite.sh" -t "$target_skill"
"$skill_root/scripts/draft-rewrite.sh" -t "$target_skill" -a "<audit-report-path>"
"$skill_root/scripts/draft-rewrite.sh" -t "$target_skill" -o "<output-path>"
```

By default this writes a `REWRITE-DRAFT.md` next to the target skill's `SKILL.md`, and names it on stdout. The draft is a skeleton to work from, not a rewritten skill: the drafter reads the target's `SKILL.md` only to run three heading probes over it, and never writes to it.

#### Where the draft is written

`-o` makes the destination yours to name — a scratch directory, your notes, a review branch — and it writes there instead of into the skill you are auditing, not as well as.

Four destinations it refuses, and it refuses them before it runs the audit, so a refusal leaves nothing behind anywhere:

- **A directory.** Name the file to write, not the folder to write it in.
- **A symbolic link.** A redirect follows the link and truncates what is on the other end, so where the draft would go is not where you named it. Give the path the link points at.
- **A `SKILL.md`.** A rewrite draft is not a skill. This is the mechanism behind the Constraints section's "do not overwrite the original `SKILL.md`": until `-o` existed there was no way to reach that mistake, and now that there is, the drafter refuses it rather than trusting you not to make it.
- **Anywhere inside a live agent configuration directory, or inside a directory an agent reads skills from.** An agent reads a skills directory as skills, so a draft left in one is not a stray file but a document that may be loaded as instructions. The list is every `$HOME`-relative directory this repository's README documents an agent reading skills from — including `$HOME/.agents`, which belongs to no single harness and is read by more than one.

Protected destinations: `$HOME/.claude`, `$HOME/.cursor`, `$HOME/.codex`, `$HOME/.devin`, `$HOME/.config`, `$HOME/.agents`.

Each of those is decided **after the path is resolved**, which is the part worth knowing if you are writing the destination in a script. `$HOME/drafts/x.md` where `drafts` is a symlink into `~/.claude/skills` is refused, though nothing in its spelling looks wrong; `$HOME/.claude-notes/x.md` is accepted, though `$HOME/.claude` is a prefix of it. And a destination whose path the drafter cannot resolve at all — a directory it may not traverse — is refused as an execution error, exit 3, rather than being let through: "I could not tell where this would land" is not "go ahead".

Anywhere else you can write, it will write. The refusals are a short list, not a sandbox.

#### The sections the drafter writes

```text
## Current state
## Proposed structure
## Missing section templates
### When to use
### Examples
#### Example 1: <scenario>
### Validation checklist
## Action items
## Notes
```

Above them, a `# Rewrite draft: <target>` title. What each holds:

- `Current state` — the Stage 1 checks' stdout, captured one check at a time, or the contents of the report given with `-a`. Their stderr is not folded in: a check that could not compute a verdict stops the draft instead of appearing here as a finding about the skill, so nothing in this section is a diagnostic about the check itself; see Prerequisites.
- `Proposed structure` — the spec and ICM section list. Fixed text, the same for every target.
- `Missing section templates` — a blank template for each of `When to use`, `Examples` and `Validation checklist` the target has no heading for. The three probes read heading levels `##` to `######` and ignore case, and they are not all the same shape: `When to use` and `Example`/`Examples` must be the whole heading, while `Validation` is matched as a prefix, so a target's own `### Validation` section suppresses the `Validation checklist` template. A target that already has all three gets the heading and nothing under it.
- `Action items` and `Notes` — fixed text, the same for every target.

**What the drafter does not do**, and what Stage 3 is therefore for. It does not copy or correct the target's frontmatter. It writes no template for `Deterministic actions`, `Orchestration`, `AI judgment` or `Constraints`. That is four of the six sections its own `Proposed structure` requires. `Action items` is a five-item review checklist, not a mapping of the audit's findings: it is the same text for a skill that audits clean and one that fails, and the drafter never reads the evaluation matrix. Scoring the dimensions and turning them into fixes is Stage 1 and Stage 3 work, done by the reader.

`tests/test_rewrite.sh` compares the list above against the headings a run actually writes, so neither side can change without the other, and it pins each of the four gaps above as an absence — so if the drafter ever learns to close one, the suite goes red rather than leaving this section describing a draft that no longer exists.

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
skill_root="<path-to-skill-architect>/skills/skill-rewrite"
"$skill_root/scripts/draft-rewrite.sh" -t <target-skill-dir> -a <audit-report-path>
```

Output: `<target-skill-dir>/REWRITE-DRAFT.md`.

### Rewrite plan outline

```text
skills/release-check/REWRITE-DRAFT.md
- Preserve: frontmatter name, description intent, existing scripts.
- Add: When to use, Examples, Validation checklist.
- Move: deep reference content to references/validation-patterns.md.
- Remove: duplicate Process section under Orchestration.
- Rename: scripts/check.sh to scripts/validate.sh to match SKILL.md text.
```
