---
name: skill-audit
description: Evaluate an Agent Skill directory against the Agent Skills spec, Anthropic best practices, and the Interpretable Context Methodology (ICM). Use when reviewing a SKILL.md before release, after a major edit, or when a skill is not triggering or executing reliably.
license: MIT
compatibility: POSIX shell (bash 3.2+ or zsh), git.
metadata:
  version: "0.2.0"
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
command -v skill-validator >/dev/null 2>&1 || { echo "skill-validator not found. Install with: brew install agent-ecosystem/tap/skill-validator" >&2; exit 1; }
command -v skillscore >/dev/null 2>&1 || { echo "skillscore not found. Install with: npm install -g skillscore" >&2; exit 1; }

"$skill_root/scripts/check-frontmatter.sh" "$target_skill"
"$skill_root/scripts/check-structure.sh" "$target_skill"
"$skill_root/scripts/check-quality.sh" "$target_skill"
```

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

The report nests the full output of each source under `spec`, `quality`, and `policy`, with a top-level `summary` for quick pass/fail checks. Parse with `jq`:

```bash
"$skill_root/scripts/audit-report.sh" "$target_skill" | jq '.summary.passed'
"$skill_root/scripts/audit-report.sh" "$target_skill" | jq '.summary.quality_grade'
"$skill_root/scripts/audit-report.sh" "$target_skill" | jq '.policy.findings[] | select(.level == "fail")'
```

Exit codes: 0=pass, 1=spec/path failure, 2=policy failure, 3=execution error. House-policy findings include rule IDs: PL001 for license, PL002 for headings, PL003 for line count, PL004 for code blocks, PL005 for lists, PT001 for missing scripts, PT002 for missing markdown links.

List bundled resources:

```bash
ls -la "$target_skill/scripts" 2>/dev/null || echo "no scripts/"
ls -la "$target_skill/references" 2>/dev/null || echo "no references/"
ls -la "$target_skill/assets" 2>/dev/null || echo "no assets/"
```

Treat a nonzero script exit as a deterministic failure. Preserve individual findings in the report rather than replacing them with a generic failure.

### Evaluation checklists

Use these inline checklists for criteria that the scripts do not cover mechanically. Consult [best practices](references/best-practices.md) for criteria requiring interpretation.

**Agent Skills spec checklist**

- Directory name matches `name` in frontmatter.
- `name` is lowercase, hyphenated, 1-64 chars.
- `description` ≤1024 chars, describes what + when to use.
- `license` present.
- `SKILL.md` is at the skill root.

**Trigger-quality checklist**

- Imperative phrasing: "Use when..."
- User intent, not implementation details.
- Specific contexts and trigger phrases.
- Synonym coverage for key terms.
- Negative scope boundaries if the edge is unclear.
- Third person; no "you" or "I".

**Body checklist**

- Imperative voice throughout.
- Under ~500 lines.
- Clear purpose / what this does.
- "When to use" with specific scenarios.
- Structured process or workflow.
- At least one concrete example with real paths or commands.
- Constraints or guardrails.

**Determinism checklist**

- Mechanical work is in scripts, not prose.
- Scripts have error handling (`set -euo pipefail` or equivalent).
- No hardcoded absolute paths.
- Exact commands and file references.
- Checklists instead of vague "review this" instructions.

**ICM context-management checklist**

- Progressive disclosure: frontmatter → body → references/assets.
- No "load everything" instructions.
- Multi-step skills use stage contracts: Inputs, Process, Outputs.
- Working artifacts (per-run) separated from stable reference material.
- L0–L2 catalog files stay small; L3 reference payloads are pointed at, not inlined.

**Validation checklist**

- Skill includes a checklist, evals, tests, or pass/fail criteria.
- Criteria are concrete enough to verify.
- Evals are specific and realistic if present.

**Self-contained checklist**

- All needed resources are in the skill directory or explicitly sourced.
- Templates are blank and reusable, not filled-in deployments.
- No hidden external dependencies.

### Stage 3: Evaluate the 10 dimensions

Score each dimension 0–2:

- 0 = missing or broken
- 1 = present but weak
- 2 = strong

| Dimension | Strong signal |
|---|---|
| **Spec compliance** | Directory name, frontmatter, file layout all correct |
| **Trigger** | Imperative, intent-focused, specific contexts, synonyms, negative scope, ≤1024 chars |
| **Scope** | One job; does not straddle categories |
| **Body structure** | Imperative, <500 lines, purpose, when-to-use, workflow, examples, constraints |
| **Determinism** | Scripts for mechanical work, exact paths, error handling, checklists |
| **Context mgmt** | Progressive disclosure, scoped loading, stage contracts, L3/L4 separation |
| **Token discipline** | L0–L2 small, no inlined L3 payloads, stage context ~2k–8k tokens |
| **Validation** | Checklist, evals, tests, or pass/fail criteria |
| **Self-contained** | All resources in directory or explicitly sourced |
| **Examples** | Concrete input/output examples, not only abstract descriptions |

See [evaluation matrix](references/evaluation-matrix.md) for the three-tier evaluation model and suggested output format.

### Stage 4: Produce the report

Use the format below. See [evaluation matrix](references/evaluation-matrix.md) for the full template with tier annotations. Include:

- Every scoring dimension and its 0–2 score.
- Deterministic command results.
- The 60/30/10 ratio.
- Ordered, concrete fixes tied to failed or weak criteria.

```markdown
# Skill audit: <skill-name>

| Dimension | Score | Notes |
|---|---|---|
| Spec compliance | 0/1/2 | ... |
| Trigger | 0/1/2 | ... |
| Scope | 0/1/2 | ... |
| Body structure | 0/1/2 | ... |
| Determinism | 0/1/2 | ... |
| Context mgmt | 0/1/2 | ... |
| Token discipline | 0/1/2 | ... |
| Validation | 0/1/2 | ... |
| Self-contained | 0/1/2 | ... |
| Examples | 0/1/2 | ... |

## Ratio

- Deterministic: X%
- Orchestration: Y%
- AI judgment: Z%

## Fixes

1. ...
```

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
