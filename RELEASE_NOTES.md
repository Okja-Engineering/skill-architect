# Release notes

## v0.3.1

**Dogfooding fixes from self-audit.**

- Added dependency-verification guards before script invocations — missing `skill-validator` or `skillscore` now fails loudly.
- Added inline evaluation checklists and scoring dimensions table directly in `skill-audit` SKILL.md body, keeping external tool integration and reference files intact.
- Added Inputs/Outputs stage contracts to the Orchestration section.
- Fixed hardcoded `/tmp` path in `skill-rewrite` example.
- Quality scores improved: `skill-audit` 89→92.5 (A-), `skill-rewrite` 86.5→89.5 (B+).
- 131 tests pass; no spec regressions.

## v0.3.0

**Unified machine-readable audit reports.**

- `audit-report.sh` composes `skill-validator`, `skillscore`, and house-policy checks into a single JSON document with a top-level summary for quick pass/fail checks.
- `check-paths.sh` and `check-structure.sh` now support `--json` for structured output; text mode is unchanged.
- 131 tests pass across layout, end-to-end walk, F01 format/policy separation, and F02 unified report.

## v0.2.0

**Two-skill workflow: audit, then draft a rewrite.**

- `skill-audit` evaluates a skill directory and reports pass/fail on 10 dimensions.
- `skill-rewrite` consumes an audit report and drafts a `REWRITE-DRAFT.md` with section templates and action items. It does not apply changes without approval.
- Tests now cover both skills and an end-to-end rewrite draft run.

## v0.1.0

**First release: a spec-aware auditor for Agent Skills.**

- `skill-audit` evaluates a skill directory on 10 dimensions and produces a pass/fail report with fixes.
- Deterministic checks cover frontmatter, file layout, required headings, code blocks, and bundled resources.
- Scoring matrix and best-practices references draw from the Agent Skills spec, Anthropic guidance, NVIDIA SkillEvaluator tiers, SkillsBench, and ICM.
- Available as a Devin plugin and via manual copy for Claude Code, Cursor, and Codex.
