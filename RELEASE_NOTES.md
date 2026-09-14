# Release notes

## v0.4.1

**Makes 0.4.0's promises true.** One family of profiler bugs — signals reported available on structural evidence rather than on a value actually read — and documentation corrected to match the code. No new features.

- **The adapter reads the telemetry Claude Code actually emits.** It parsed an envelope nothing has ever produced, so a real OTel export came back all-`unknown` and only a hand-written file came back with numbers. It now parses OTLP/JSON — one `ExportMetricsServiceRequest` or `ExportLogsServiceRequest` per JSON object, one per line or concatenated — with the real attribute names, delta and cumulative temporality, and the spec's number-or-string integers. Two consequences worth knowing before you compare a 0.4.0 profile with a 0.4.1 one: a tool call's `success` now says whether the tool ran and succeeded rather than whether it was permitted, and the reasoning token count is gone, because Claude Code has no reasoning token type. Claude Code has no file exporter of its own, so the README now has a "Capturing an OTel export" section with two ways to produce the file.
- **A signal is `present` only when a value was read.** `probe` scanned the export for structure while `capture` extracted values from it, and the two predicates disagreed in both directions: a partial export lost the signals it did carry, while a token metric with an unreadable type or value produced `present` with all-zero counts. There is now one predicate per signal, used by both, and the capability report is derived from what was actually read.
- **A broken export says so.** An export file you supplied that cannot be read or parsed is `error` naming the failure, not "OTel export not configured". Session timing spans the earliest API request to the latest, so an out-of-order export can no longer report negative elapsed time.
- **`capture --export-file` fails instead of ignoring you.** No shipped adapter reads that flag; it used to accept the path and return an all-unknown profile.
- **Install prerequisites are documented.** `skill-validator`, `skillscore`, and `jq`, with what breaks without each.
- **`skill-rewrite` needs `skill-audit` beside it.** Its `draft-rewrite.sh` resolves the audit scripts at `../skill-audit`; copying it alone quietly produces a draft with no audit in it. Now said out loud in the README.
- **The documented install command works.** Neither `claude plugins install Okja-Engineering/skill-architect` nor `claude plugins install .` could: `install` resolves a plugin name against configured marketplaces. This release ships `.claude-plugin/marketplace.json` and documents `claude plugin marketplace add` followed by `claude plugin install skill-architect@skill-architect`, verified live against a local clone. The same commands pointed at the GitHub remote work once this release is on `main`, which is where `marketplace add` looks for the manifest.
- **Three delivered-but-undocumented profiler features are now in the README**: the `version` subcommand and the `--export-file` flag, both from 0.4.0 (the flag is reserved for a future session-export adapter), and per-signal probe detection, which landed after 0.4.0 and was rewritten here.
- **A number that is not a token count no longer becomes one.** A data point's value was carried through an out-of-range float-to-integer conversion, which Go leaves to the machine: the same export read as `input: 9223372036854775807` on arm64 and `input: -9223372036854775808` on amd64. Negative values, `NaN` and infinity were all assimilated too. Each is refused and counted now, and the reason names it.
- **Two token series that differ only by an attribute the adapter does not interpret stay two series.** They used to merge, so a session's tokens were reported as a fraction of themselves.
- **A count the export said nothing about is absent from the profile, not zero.** A cache-only export reported `input: 0, output: 0` beside its real numbers. A count read as zero still reports zero. No profile keys changed names.
- **`capture` exits 2 when it read nothing and something failed.** It exited 0 for a missing or unreadable `--otel-file`, so a script wrapping it filed the all-unknown profile as a success. The profile is still printed.
- **A capture identifies you, and the README says which fields.** The capture recipe no longer sets `OTEL_LOG_TOOL_DETAILS=1`: the adapter reads nothing it adds, and it widens the export.
- **The Devin, Codex and Cursor install rows are marked unverified.** Only the Claude Code route was run against a clean install.
- **The spec now describes the code.** The `MetricResult[T any]` generic never existed; the acceptance criteria described the probe one commit out of date; the fallback reason named env vars the adapter never reads and claimed every metric carries it.
- **"Snapshot-pinned" was an overstatement.** `--snapshot` is a label you supply and the profiler records verbatim. It does not hash or verify the skill directory.
- Module path corrected to `github.com/Okja-Engineering/skill-architect/profiler`; CI takes its Go version from `profiler/go.mod` instead of a stale pin four minors behind. The profiler adapter version moves to 0.4.1, since a profile's contents changed.
- 51 test functions pass — 47 in the profiler package, 4 in the CLI — and 135 subtests, under `-race`. The shell suites are at 134: 0.4.0's 131 plus three new version-surface assertions. CI now runs `gofmt -l`, `go vet` and `go test -race` rather than `go test` alone.
- v0.3.1's note that `skill-audit` scores 92.5 (A-) was true of the 0.3.1 commit. It has been **94 (A)** since 3e2efd8, and still is; `skill-rewrite` is unchanged at 89.5 (B+).

## v0.4.0

**Profiler preview: harness-agnostic runtime signal capture.**

- New `profiler/` Go module with adapter interface (`ProfilerAdapter`), per-metric results (present/unknown/error), `CapabilityReport`, and serialized `Profile` format.
- Claude Code adapter reads OTel export data for token counts, tool calls, and timing. Skill activation and attribution are honestly `unknown` — the adapter does not yet read skill-level attributes.
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
