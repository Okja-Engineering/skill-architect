# Changelog

All notable changes to `skill-architect`.

## Unreleased

## 0.4.0 — 2026-09-10

Profiler preview: harness-agnostic runtime signal capture with graceful degradation.

- Added `profiler/` Go module — adapter interface (`ProfilerAdapter`), `MetricResult` (present/unknown/error), `CapabilityReport`, and serialized `Profile` format pinned to a snapshot hash.
- Added Claude Code adapter (Slice 1) — reads OTel export data for token counts, tool calls, and timing. Skill activation and attribution are honestly `unknown` (Claude Code has no skill-level events).
- Added `profiler` CLI — `probe` reports capabilities, `capture` produces a profile JSON.
- Added `docs/profiler-spec.md` — the adapter interface contract, profile format, and acceptance criteria.
- 14 profiler tests pass (serialization, capability reports, capture with/without OTel, round-trip, no-value-in-JSON for unknown/error states).
- All 131 existing tests still pass; no regressions.
- Design space survey of 4 harnesses (Cursor, Claude Code, Codex, Devin) in `tmp/teams/architect/profiler-design-space.md`.

## 0.3.1 — 2026-09-09

Dogfooding fixes: ran `skill-architect` against its own skills and addressed the findings.

- Added dependency-verification guards (`command -v skill-validator`, `command -v skillscore`) before script invocations in `skill-audit` SKILL.md — a missing dependency now fails loudly instead of silently.
- Added inline evaluation checklists (spec, trigger, body, determinism, ICM, validation, self-contained) directly in `skill-audit` SKILL.md body so the auditor is self-contained for criteria that scripts don't cover mechanically.
- Added scoring dimensions table with "Strong signal" column inline in `skill-audit` SKILL.md, keeping the `references/evaluation-matrix.md` reference for the tier model and output format.
- Added inline report format template to Stage 4 of `skill-audit` SKILL.md.
- Added Inputs/Outputs stage contracts to the Orchestration section of `skill-audit` SKILL.md.
- Fixed hardcoded `/tmp/release-check-audit.md` path in `skill-rewrite` SKILL.md example — replaced with a relative path.
- Bumped `skill-audit` metadata version to 0.2.0.
- Quality scores improved: `skill-audit` 89→92.5 (A-), `skill-rewrite` 86.5→89.5 (B+).
- All 131 tests pass; no spec regressions, no orphaned files.

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
