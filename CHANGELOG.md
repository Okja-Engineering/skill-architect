# Changelog

All notable changes to `skill-architect`.

## Unreleased

## 0.2.0 — 2026-09-07

- Added `skill-rewrite` skill for drafting rewrites from audit reports.
- Added `draft-rewrite.sh` helper that runs the audit and generates a `REWRITE-DRAFT.md` with templates for missing sections.
- Updated README and tests to cover both skills.

## 0.1.0 — 2026-09-07

**Initial release.**

- Added `skill-audit` skill for evaluating Agent Skills against the spec, Anthropic best practices, and ICM context-management criteria.
- Bundled `check-frontmatter.sh` and `check-structure.sh` deterministic checks.
- Added `references/best-practices.md` and `references/evaluation-matrix.md` for interpretation and scoring.
- Added plugin manifests for Devin, Claude Code, Cursor, and Codex.
- Added layout and script tests in `tests/`.
