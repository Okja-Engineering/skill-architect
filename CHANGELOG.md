# Changelog

All notable changes to `skill-architect`.

## Unreleased

## 0.3.0 — 2026-09-09

- Added `audit-report.sh` — produces a unified machine-readable JSON report composing `skill-validator` (spec, structure, content, contamination), `skillscore` (7-dimension quality scoring), and house-policy checks (PL001–PL005, PT001–PT002) into one document with a top-level summary.
- Added `--json` flag to `check-paths.sh` and `check-structure.sh` for structured output; text mode is unchanged.
- Documented `audit-report.sh` and `--json` flags in `skill-audit` SKILL.md.
- Added 58 tests covering the unified report shape, summary fields, all three sources, policy/path findings, spec failures, self-audit, and backward compatibility.
- Wired `test_f02.sh` into CI.

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
