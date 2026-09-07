# Release notes

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
