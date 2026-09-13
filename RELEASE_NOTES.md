# Release notes

## v0.4.1

**Makes 0.4.0's promises true.** One family of profiler bugs — signals reported available on structural evidence rather than on a value actually read — and documentation corrected to match the code. No new features.

- **A signal is `present` only when a value was read.** `probe` scanned the export for structure while `capture` extracted values from it, and the two predicates disagreed in both directions: a partial export lost the signals it did carry, while a token metric with an unreadable type or value produced `present` with all-zero counts. There is now one predicate per signal, used by both, and the capability report is derived from what was actually read.
- **A broken export says so.** An export file you supplied that cannot be read or parsed is `error` naming the failure, not "OTel export not configured". Session timing spans the earliest API request to the latest, so an out-of-order export can no longer report negative elapsed time.
- **`capture --export-file` fails instead of ignoring you.** No shipped adapter reads that flag; it used to accept the path and return an all-unknown profile.
- **Install prerequisites are documented.** `skill-validator`, `skillscore`, and `jq`, with what breaks without each.
- **`skill-rewrite` needs `skill-audit` beside it.** Its `draft-rewrite.sh` resolves the audit scripts at `../skill-audit`; copying it alone quietly produces a draft with no audit in it. Now said out loud in the README.
- **The documented install command works.** Neither `claude plugins install Okja-Engineering/skill-architect` nor `claude plugins install .` could: `install` resolves a plugin name against configured marketplaces. This release ships `.claude-plugin/marketplace.json` and documents `claude plugin marketplace add` followed by `claude plugin install skill-architect@skill-architect`, verified live.
- **Three delivered-but-undocumented profiler features are now in the README**: the `version` subcommand, per-signal probe detection, and the `--export-file` flag (reserved for a future session-export adapter).
- **The spec now describes the code.** The `MetricResult[T any]` generic never existed; the acceptance criteria described the probe one commit out of date; the fallback reason named env vars the adapter never reads and claimed every metric carries it.
- **"Snapshot-pinned" was an overstatement.** `--snapshot` is a label you supply and the profiler records verbatim. It does not hash or verify the skill directory.
- Module path corrected to `github.com/Okja-Engineering/skill-architect/profiler`; CI takes its Go version from `profiler/go.mod` instead of a stale pin four minors behind. The profiler adapter version moves to 0.4.1, since a profile's contents changed.
- 22 profiler test functions pass, 42 cases counting subtests. All 131 existing tests still pass.

## v0.4.0

**Profiler preview: harness-agnostic runtime signal capture.**

- New `profiler/` Go module with adapter interface (`ProfilerAdapter`), per-metric results (present/unknown/error), `CapabilityReport`, and serialized `Profile` format.
- Claude Code adapter reads OTel export data for token counts (including reasoning tokens), tool calls, and timing. Skill activation and attribution are honestly `unknown` — the adapter does not yet read skill-level attributes.
- `profiler` CLI: `probe` reports what the adapter can capture; `capture` produces a profile JSON carrying the `--snapshot` id the caller supplied. The id labels the profile; nothing hashes or validates `--skill-dir` against it.
- Graceful degradation: unavailable metrics are `unknown` with a reason, never silently invented.
- 14 profiler tests pass at this release, 15 after 30f374c. All 131 existing tests still pass.
- Design survey of 4 harnesses (Cursor, Claude Code, Codex, Devin) informed the adapter-per-harness architecture.

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
