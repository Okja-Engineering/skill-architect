# Changelog

All notable changes to `skill-architect`.

## Unreleased

## 0.4.1 — 2026-09-13

Makes 0.4.0's promises true. No new features: one real bug, and documentation
corrected to match the code.

**Fixed**

- Profiler capture no longer discards signals a partial OTel export does carry. `Capture` used the token capability as a proxy for "is any OTel data available", so an export with `claude_code.tool_decision` and `claude_code.api_request` events but no `claude_code.token.usage` metric was reported as "OTel export not configured" and its tool-call and timing data was dropped. Capture now returns early only when probing found no source at all, and each signal's state is settled by its own extractor. Pinned by a contract test over six export shapes: every signal the probe advertises is `present` in the profile, every signal it reports `none` is `unknown` with a reason.
- Corrected the `profiler` module path from `github.com/auraprix/skill-architect/profiler` to `github.com/Okja-Engineering/skill-architect/profiler`, which is where the repo actually lives.
- CI now reads the Go toolchain version from `profiler/go.mod` instead of a hardcoded `1.23` that was two minors behind what the module requires.
- Removed `CaptureOpts.OtelEndpoint`. No adapter read it and no flag set it.

**Documentation**

- `README.md` gains a Prerequisites block for `skill-validator`, `skillscore`, and `jq`, naming what fails without each. `skill-validator` and `skillscore` fail loudly; `jq` has no guard.
- `README.md` notes that `skill-rewrite` needs a sibling `skill-audit` directory — `draft-rewrite.sh` resolves the audit scripts at `../skill-audit`, and copying `skill-rewrite` alone produces a draft with "No such file or directory" where the audit should be.
- `README.md` documents three things 0.4.0 shipped without mentioning: the `profiler version` subcommand, per-signal probe detection, and the `capture --export-file` flag (reserved for non-OTel adapters; the Claude Code adapter ignores it).
- `docs/profiler-spec.md` replaces the `MetricResult[T any]` generic, which was never implemented, with the real `RawMetricResult` plus the typed wrapper per metric category.
- `docs/profiler-spec.md` AC2 described the all-or-nothing probe that 30f374c replaced, contradicting the probe-logic section above it. Both now say the same thing, and a new AC9 pins probe/capture agreement on partial exports.
- `docs/profiler-spec.md` fallback reason named `CLAUDE_CODE_ENABLE_TELEMETRY` and `OTEL_METRICS_EXPORTER`, which this adapter never reads. It now matches the string the code emits.
- "Snapshot-pinned" overstated what the profiler does: `--snapshot` is a caller-supplied label copied into the profile verbatim, and nothing hashes or validates `--skill-dir` against it. Reworded across the README, release notes, spec, and `Profile` doc comment. This also corrects 0.4.0's "`Profile` format pinned to a snapshot hash" below.
- 0.4.0's "14 profiler tests" was 15 after 30f374c, and is **17** at this release with the two added above.
- 0.4.0 cited the design-space survey at `tmp/teams/architect/profiler-design-space.md`, a gitignored path no reader can open. Path reference removed.
- 0.4.0's "Claude Code has no skill-level events" was too strong. Claude Code attaches a `skill.name` attribute to its token and cost metrics; this adapter does not read skill-level attributes yet. Reading them is future work.

**Known past changes, recorded late**

- 0.3.1's entries describe the evaluation checklists, scoring dimensions table, and report format template as inline in `skill-audit/SKILL.md`. They are not: 3e2efd8 moved the checklists to `references/checklists.md` and pointed the scoring table and report template at `references/evaluation-matrix.md`, applying progressive disclosure. The 0.3.1 entries are left as written; this is the correction.
- `ToolCallEntry.Duration` (`duration_ms`) was removed from profile schema v1 in 30f374c because nothing populated it. That went unrecorded at the time. The schema string is unchanged at `skill-architect/profile/v1`; consumers should not expect a `duration_ms` key on tool-call entries.

## 0.4.0 — 2026-09-10

Profiler preview: harness-agnostic runtime signal capture with graceful degradation.

- Added `profiler/` Go module — adapter interface (`ProfilerAdapter`), `MetricResult` (present/unknown/error), `CapabilityReport`, and serialized `Profile` format pinned to a snapshot hash.
- Added Claude Code adapter (Slice 1) — reads OTel export data for token counts, tool calls, and timing. Skill activation and attribution are honestly `unknown` (the adapter does not yet read skill-level attributes).
- Added `profiler` CLI — `probe` reports capabilities, `capture` produces a profile JSON.
- Added `docs/profiler-spec.md` — the adapter interface contract, profile format, and acceptance criteria.
- 15 profiler tests pass (serialization, capability reports, capture with/without OTel, round-trip, no-value-in-JSON for unknown/error states).
- All 131 existing tests still pass; no regressions.
- Design space survey of 4 harnesses (Cursor, Claude Code, Codex, Devin) informed the adapter-per-harness architecture.

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
