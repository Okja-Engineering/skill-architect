# Changelog

All notable changes to `skill-architect`.

## Unreleased

## 0.4.1 — 2026-09-13

Makes 0.4.0's promises true. No new features: one family of profiler bugs — a
signal reported available on structural evidence rather than on a value read —
and documentation corrected to match the code.

**Fixed**

- The Claude Code adapter parses real OTLP/JSON. It read a bespoke `{"metrics": [...], "logs": [...]}` envelope carrying `token_type`, `cache_read`, `reasoning` and `decision: "approved"` — a shape nothing has ever emitted. Claude Code emits OTLP: `resourceMetrics[].scopeMetrics[].metrics[]` and `resourceLogs[].scopeLogs[].logRecords[]`, with `type` ∈ `input`/`output`/`cacheRead`/`cacheCreation` and `decision` ∈ `accept`/`reject`. A real export therefore produced an all-`unknown` profile while only a hand-written file produced values — the truth inverted. The adapter now reads one `Export*ServiceRequest` per JSON object (a single object, NDJSON, or concatenated objects), accepts every 64-bit integer as a JSON number or a decimal string as the OTLP spec requires, sums delta data points and supersedes cumulative ones per time series, and refuses a temporality that reads as neither rather than guessing a direction. Three further consequences: `ToolCallEntry.Success` now means the tool ran and succeeded, read from `claude_code.tool_result`, with rejected calls listed from `claude_code.tool_decision` as `success: false`; `TokenCounts.Reasoning` is no longer populated, because Claude Code has no reasoning token type and a zero there would claim a measurement nobody made; and `CaptureOpts.ExportFile` is refused by the adapter itself, not only by the CLI, so a library caller is told rather than handed an all-unknown profile. No profile schema keys changed, and `capability.adapter_version` already records this release. The bullets below stay as written — they are true as history of what the `token_type` and `decision` handling did, inside the envelope this release replaced.
- A profiler signal is `present` only when a value was actually read from the export. `Probe` scanned the export for structure while `Capture` extracted values from it — two predicates per signal, over two reads of the same file — and they disagreed in both directions. A `claude_code.token.usage` metric whose `token_type` the adapter does not recognise, or whose value is not a number, produced `present` with all-zero counts; a `tool_decision` with no `tool_name` and an `api_request` with no timestamp did the same. Both predicates are now one: the export is read and parsed once, each extractor is its signal's only predicate, and the capability report is derived from what those extractors returned, so probe and capture cannot disagree.
- Profiler capture no longer discards signals a partial OTel export does carry. `Capture` used the token capability as a proxy for "is any OTel data available", so an export with `claude_code.tool_decision` and `claude_code.api_request` events but no `claude_code.token.usage` metric was reported as "OTel export not configured" and its tool-call and timing data was dropped.
- An OTel export that was supplied but cannot be read or parsed is now `error` for all three signals, naming the read or parse failure. It previously reported "OTel export not configured", sending the caller to fix the one thing that was not wrong, because the probe's parse error was swallowed. `MetricError` was unemittable by this adapter — every branch that produced it was unreachable, and `ErrorToolCallResult` and `ErrorTimingResult` did not exist at all — and is now emitted and covered by tests.
- Session timing spans the earliest `claude_code.api_request` event to the latest, not the first in file order to the last. An out-of-order export produced a negative `total_ms`.
- `capture --export-file` now fails loudly instead of accepting a path nothing reads. No shipped adapter consults `CaptureOpts.ExportFile`, and the Claude Code adapter's fallback to it was unreachable, so the flag silently produced an all-unknown profile that looked like missing telemetry.
- Added `.claude-plugin/marketplace.json`, without which neither documented Claude Code install command could work: `claude plugin install` resolves a plugin name against configured marketplaces, not a repo path or `.`.
- `AdapterVersion` moved from `0.1.0` to `0.4.1`. It is recorded in every profile, and this release changed what a profile contains for the same export.
- Corrected the `profiler` module path from `github.com/auraprix/skill-architect/profiler` to `github.com/Okja-Engineering/skill-architect/profiler`, which is where the repo actually lives.
- CI now reads the Go toolchain version from `profiler/go.mod` instead of a hardcoded `1.23`, four minors behind the `1.27.1` the module requires, and disables `setup-go`'s cache, which warned on every run looking for a `go.sum` this dependency-free module does not have.
- Removed `CaptureOpts.OtelEndpoint`, which nothing read, and `CapabilityReport.AnySource`, whose only caller went away with the single-resolution fix.

**Documentation**

- `README.md` documents the Claude Code install route that works — `claude plugin marketplace add` followed by `claude plugin install skill-architect@skill-architect`. Verified live against a clean install from a local clone, which is what was exercised; the same commands against the GitHub remote work once this release is on `main`, because that is where `marketplace add` looks for `.claude-plugin/marketplace.json`.
- `README.md` gains a Prerequisites block for `skill-validator`, `skillscore`, and `jq`, naming what fails without each. `skill-validator` and `skillscore` fail loudly; `jq` has no guard. Without `skill-validator`, `check-frontmatter.sh` run on its own skips spec validation silently; only its license gate still runs.
- `README.md` notes that `skill-rewrite` needs a sibling `skill-audit` directory — `draft-rewrite.sh` resolves the audit scripts at `../skill-audit` from the `skill-rewrite` directory, and copying `skill-rewrite` alone produces a draft with "No such file or directory" where the audit should be.
- `README.md` documents three things 0.4.0 shipped without mentioning: the `profiler version` subcommand, per-signal probe detection, and the `capture --export-file` flag (reserved for a future session-export adapter; passing it today is an error).
- `docs/profiler-spec.md` replaces the `MetricResult[T any]` generic, which was never implemented, with the real `RawMetricResult` plus the typed wrapper per metric category. The same name in the `profiler/types.go` comments is corrected too.
- `docs/profiler-spec.md` AC2 described the all-or-nothing probe that 30f374c replaced, contradicting the probe-logic section above it. Probe logic, capture logic, and the acceptance criteria now state one predicate, and AC9–AC11 pin probe/capture agreement, "present means a value was read", and the error classification.
- `docs/profiler-spec.md` fallback reason named `CLAUDE_CODE_ENABLE_TELEMETRY` and `OTEL_METRICS_EXPORTER`, which this adapter never reads, and claimed every metric carries it. `skill_activation` and `attribution` never do, and a supplied-but-unreadable export no longer does either.
- "Snapshot-pinned" overstated what the profiler does: `--snapshot` is a caller-supplied label copied into the profile verbatim, and nothing hashes or validates `--skill-dir` against it. Reworded across the README, release notes, spec, `Profile` doc comment, and the 0.4.0 entry below.
- 0.4.0's profiler test count is stated per commit: 14 at 0.4.0 itself, 15 after 30f374c. This release has **33** test functions (31 in the profiler package, 2 in the CLI) and **65** subtests.
- 0.4.0 cited the design-space survey at `tmp/teams/architect/profiler-design-space.md`, a gitignored path no reader can open. Path reference removed.
- 0.4.0's "Claude Code has no skill-level events" was too strong: Claude Code attaches a `skill.name` attribute to parts of its OTel surface. This adapter does not read skill-level attributes yet, and the doc comments and reason strings that asserted otherwise now say that instead. Reading them is future work.

**Known past changes, recorded late**

- 0.3.1's entries describe the evaluation checklists, scoring dimensions table, and report format template as inline in `skill-audit/SKILL.md`. They are not: 3e2efd8 moved the checklists to `references/checklists.md` and pointed the scoring table and report template at `references/evaluation-matrix.md`, applying progressive disclosure. The 0.3.1 entries are left as written; this is the correction.
- `ToolCallEntry.Duration` (`duration_ms`) was removed from profile schema v1 in 30f374c because nothing populated it. That went unrecorded at the time. The schema string is unchanged at `skill-architect/profile/v1`; consumers should not expect a `duration_ms` key on tool-call entries.

## 0.4.0 — 2026-09-10

Profiler preview: harness-agnostic runtime signal capture with graceful degradation.

- Added `profiler/` Go module — adapter interface (`ProfilerAdapter`), per-metric results (present/unknown/error), `CapabilityReport`, and a serialized `Profile` format carrying the caller-supplied snapshot id.
- Added Claude Code adapter (Slice 1) — reads OTel export data for token counts, tool calls, and timing. Skill activation and attribution are honestly `unknown` (the adapter does not yet read skill-level attributes).
- Added `profiler` CLI — `probe` reports capabilities, `capture` produces a profile JSON.
- Added `docs/profiler-spec.md` — the adapter interface contract, profile format, and acceptance criteria.
- 14 profiler tests pass at this release, 15 after 30f374c (serialization, capability reports, capture with/without OTel, round-trip, no-value-in-JSON for unknown/error states).
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
