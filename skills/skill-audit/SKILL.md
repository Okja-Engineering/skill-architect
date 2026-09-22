---
name: skill-audit
description: Evaluate an Agent Skill directory against the Agent Skills spec, Anthropic best practices, and the Interpretable Context Methodology (ICM). Use when reviewing a SKILL.md before release, after a major edit, or when a skill is not triggering or executing reliably.
license: MIT
compatibility: bash 3.2+.
allowed-tools: Read, Bash
metadata:
  version: "0.2.0"
  carrier: bundled-scripts
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
for tool in awk dirname grep jq skill-validator skillscore wc; do
  command -v "$tool" >/dev/null 2>&1 || { echo "required tool not found: $tool" >&2; exit 3; }
done

"$skill_root/scripts/check-frontmatter.sh" "$target_skill"
"$skill_root/scripts/check-structure.sh" "$target_skill"
"$skill_root/scripts/check-quality.sh" "$target_skill"
```

The preflight names every tool the three commands below it state as a precondition, and it exits **3** because 3 is the status they exit with: a missing dependency is an execution error, rule ID `DEP001`, and **1** in the exit table below is a spec or path failure — a verdict about the audited skill that a preflight has not computed. Neither the list nor the status is written down twice. `tests/test_f01.sh` masks each of those tools in turn, runs the three commands, and requires the preflight to refuse exactly when one of them refuses and with the status it refused with — so a tool that becomes a precondition without reaching this list fails that suite rather than reaching a reader. The scope is the stated preconditions: a tool a script shells out to without stating it as one is in neither this list nor that comparison, and each script's own `# Exit codes:` header is where its set is stated. `skill-validator` and `skillscore` are installed with the commands in the list below.

The checks use established tools, each for what it does best:

- **`skill-validator`** (from `agent-ecosystem/skill-validator`, Go binary) — spec validation: frontmatter format, name, description, YAML validity, code fence integrity, internal link resolution, token counts, content analysis, contamination detection. Install with `brew install agent-ecosystem/tap/skill-validator`.
- **`skillscore`** (npm package) — quality scoring across 7 Anthropic-aligned dimensions (identity, conciseness, clarity, routing, robustness, safety, portability). Install with `npm install -g skillscore`.
- **`grep`** — house-policy absence checks (missing headings, no code blocks, no lists, line count).
- **`check-paths.sh`** — resolves local script references in code blocks against the filesystem.

For direct access to individual checks:

```bash
skill-validator validate structure -o json "$target_skill"   # spec + structure (JSON)
skill-validator check -o json "$target_skill"                 # everything except LLM scoring (JSON)
"$skill_root/scripts/check-paths.sh" "$target_skill"          # script path resolution (text)
"$skill_root/scripts/check-paths.sh" --json "$target_skill"   # script path resolution (JSON)
"$skill_root/scripts/check-structure.sh" "$target_skill"      # policy + path checks (text)
"$skill_root/scripts/check-structure.sh" --json "$target_skill" # policy + path checks (JSON)
"$skill_root/scripts/check-quality.sh" "$target_skill"        # quality scoring (JSON)
```

For a unified machine-readable report combining all three sources (spec, quality, policy) into one JSON document:

```bash
"$skill_root/scripts/audit-report.sh" "$target_skill"
```

The report nests the full output of each source under `spec`, `quality`, and `policy`, with a top-level `summary` for quick pass/fail checks. A source the report could not read is named in `spec_error`, `quality_error` or `policy_error` rather than being treated as a source with nothing to report, and an unread `spec` or `policy` leaves `summary.passed` false. Parse with `jq`:

```bash
"$skill_root/scripts/audit-report.sh" "$target_skill" | jq '.summary.passed'
"$skill_root/scripts/audit-report.sh" "$target_skill" | jq '.summary.quality_grade'
"$skill_root/scripts/audit-report.sh" "$target_skill" | jq '.policy.findings[] | select(.level == "fail")'
```

**Exit codes are per script, and this table is the scripts' own statements of them.** One sentence used to assert a single contract — `0=pass, 1=spec/path failure, 2=policy failure, 3=execution error` — over five scripts that do not share one, and it drifted because it restated rather than derived. Each row below is copied from that script's own `# Exit codes:` header line, and `tests/test_f01.sh` compares every row against that header, so neither side can move without the other.

| Script | Exit codes |
|---|---|
| `check-frontmatter.sh` | 0=pass, 1=spec failure, 2=policy failure, 3=execution error. |
| `check-paths.sh` | 0=pass, 1=path failure, 3=execution error. |
| `check-structure.sh` | 0=pass, 1=path failure, 2=policy failure, 3=execution error. |
| `check-quality.sh` | 0=report produced, 3=execution error. |
| `audit-report.sh` | 0=report generated, 3=execution error. |

**`audit-report.sh` and `check-quality.sh` report a verdict in their output, not in their exit status.** Both are generators: 0 means a document was produced, and it means that over a failing skill exactly as much as over a clean one. So `audit-report.sh "$skill" && echo PASS` prints PASS for a skill that failed every check — do not write it. Read the verdict out of the report:

```bash
"$skill_root/scripts/audit-report.sh" "$target_skill" | jq -e '.summary.passed'
```

The three house-policy checks do carry their verdict in the exit status, and each one's own set is in the table. Treat any status outside a script's set as a script that reached no verdict, not as a verdict you have not seen before.

All **five** of the scripts above load `scripts/verdict-guard.sh`, which is where "I could not compute a verdict" is decided: a required tool that is absent or not answering becomes a `DEP001` or `DEP002` finding and an exit 3, rather than a status a reader would take for a verdict. A script that cannot load it exits 3 on that alone. No command in this document invokes the guard — it is sourced, not run — so an installer that copies the files the commands name, or a pruner that removes what looks like an unused file, leaves every check in this skill exiting 3. Install and prune it with them. The count above is derived from the scripts by `tests/test_f01.sh`, not written down twice.

Rule IDs: every finding carries a level and a rule ID. At level `fail`: `PL001` for license, `PL002` for headings, `PL003` for line count, `PL004` for code blocks, `PL005` for lists, `PT001` for missing scripts, `PT002` for missing markdown links, and — on an exit 3, in the same findings array a `--json` consumer reads — `DEP001` when a required tool is absent and `DEP002` when a source returns a status or a payload the script cannot interpret. At level `unverified`: `PATH`, for a reference built from a glob or a variable, which the scripts cannot resolve and so report without judging; an unverified finding is not a failure and does not change the exit status. Those ten are the whole set these scripts emit, and `tests/test_f01.sh` compares this line against what they can emit so that neither side can grow without the other.

List bundled resources:

```bash
ls -la "$target_skill/scripts" 2>/dev/null || echo "no scripts/"
ls -la "$target_skill/references" 2>/dev/null || echo "no references/"
ls -la "$target_skill/assets" 2>/dev/null || echo "no assets/"
```

Treat a nonzero script exit as a deterministic failure. Preserve individual findings in the report rather than replacing them with a generic failure.

### Stage 3: Evaluate the 10 dimensions

Load [checklists](references/checklists.md) for criteria that the scripts do not cover mechanically. Consult [best practices](references/best-practices.md) for criteria requiring interpretation.

Score each dimension 0–2 using the dimensions and strong-signal criteria in [evaluation matrix](references/evaluation-matrix.md):

- 0 = missing or broken
- 1 = present but weak
- 2 = strong

### Stage 4: Produce the report

Use the report format in [evaluation matrix](references/evaluation-matrix.md). Include:

- Every scoring dimension and its 0–2 score.
- Deterministic command results.
- The 60/30/10 ratio.
- Ordered, concrete fixes tied to failed or weak criteria.

## Orchestration (30%)

### Inputs

- `target_skill`: one skill directory containing `SKILL.md`.
- `skill_root`: the directory containing this auditor's `SKILL.md` and scripts.

### Audit process

1. Confirm `target_skill` is a single skill directory, not a repository or `SKILL.md` file.
2. Inspect `SKILL.md`, then `scripts/`, `references/`, and `assets/` when present.
3. Run the structural scripts and record outputs.
4. Consult `references/best-practices.md` only for criteria requiring interpretation.
5. Score each dimension 0–2.
6. Estimate the deterministic / orchestration / AI-judgment ratio from the body text.
7. Produce the required report.
8. Modify the audited skill only after explicit approval.

### Outputs

One report containing:

- Every scoring dimension and its 0–2 score.
- Deterministic command results.
- A ratio totaling 100%.
- Ordered, concrete fixes tied to failed or weak criteria.

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
