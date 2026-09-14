# Release notes

## v0.4.2

**Cumulative token counts are right, and no audit script reports a verdict it could not compute.** Two fixes, one release. No new features, and no profile schema keys changed.

- **A cumulative export's totals change, in the direction of being correct.** Three defects in one merge each read as "fewer tokens than the session spent". A capture spanning several resources or scopes merged their series and let one running total replace another; a counter that restarted inside the capture discarded everything before the restart; and a flush reporting less than an earlier one on the same run took that run down with it. A profile written by 0.4.2 and one written by 0.4.1 from the same cumulative export will disagree. Delta temporality — Claude Code's default — is unaffected throughout, and `capability.adapter_version` moves to 0.4.2, which is what tells the two profiles apart.
- **A time series is what OTel says it is.** The resource it came from, the instrumentation scope that recorded it, the metric's name, and the data point's whole attribute set. The adapter keyed on the attributes alone, so two resources reporting cumulative 100 and 200 gave 200 rather than 300 — the same pair under delta gave 300 all along, which is how it hid.
- **`startTimeUnixNano` says which run of a counter a point is from, and it is now read.** The points sharing a start are one run and the run holds the greatest running total any of them reported; a point carrying a different start is a counter that restarted, and its run sits beside the earlier one instead of replacing it, so 100 followed by a restart reaching 20 is 120 and not 20. A start time of `0` is a start time that is absent — OTLP uses the protobuf JSON mapping, in which an explicit zero and an omitted field are the same message.
- **`timeUnixNano` is no longer read by the merge at all.** These are counters, so their values carry their own order and a flush reporting less than an earlier one on the same run is a capture contradicting itself, not tokens given back. Ordering by the timestamp cost real tokens: a run reporting 900 at t1001 and then 500 at t1002 came back as 500 at v0.4.1, and comes back as 900 here. The apparatus is deleted rather than repaired, because the obvious repair — keep the ordering and let each run hold its latest point — would let supplying the field that says which run a point is from make the profile report *fewer* tokens than omitting it, which is the opposite of what more information should do.
- **A point that cannot say which run it is from joins the run it is cheapest to have come from.** It came from some run — one that named itself, or one nothing else in the capture observed — so it is neither a run of its own nor free. The series holds the least total its runs can account for: the runs added, plus whatever that point reported over and above the largest of them. Runs of 100 and 20 beside an unplaceable 500 give 520, not the 500 that drops a run which named itself and not the 620 that invents a third. One export flush that omits one field no longer reports a session at twice its size.
- **A series carrying both delta and cumulative points is refused rather than resolved.** They are opposite instructions, so every way of resolving the mix invents a number the export does not contain. The refusal is counted per series and named in the reason; the well-formed series beside it still count. What a `present` result cannot yet say is that a series was refused — schema v1 has no field for it, and that is tracked for 0.5.0.
- **No `skill-audit` script tells you a skill passed when it could not check.** With `jq` absent, `check-structure.sh --json` returned `{"findings": [], "passed": true}` at exit 0 and `check-paths.sh --json` died at exit 127 with no payload at all. With `skill-validator` absent, `check-frontmatter.sh` printed `frontmatter OK`. All three now exit 3 and name the missing tool on stderr, and the two `--json` writers carry a `passed: false` payload on stdout with it — `check-frontmatter.sh` has no `--json` mode, so for it the stderr line is the whole report. `--json` requires `jq` unconditionally; text mode is unchanged.
- **A child result a parent does not recognise is an execution error, not a pass.** `check-structure.sh` read `check-paths.sh`'s 127 as success, and read an unparseable payload as "no findings". Both parents now enumerate their child's documented statuses and treat anything else as an execution error, and `passed` is derived from the computed verdict in one place instead of being hard-coded on the zero-findings branch.
- **The report composer proves a source's shape before reading a verdict out of it.** `audit-report.sh` captured its guarded child with `2>&1`, so a failing payload never parsed and the composed report said `{"passed": true, "total_findings": 0}`. And a source emitting `0`, `[1,2]`, `true` or a two-document stream took `jq` down with no report at all. One guard now owns "exactly one document", each source claims only the shape it is actually read as, and a source yielding nothing readable is named in `policy_error` with `summary.passed` false.
- **A broken guard file is an execution error, not a policy failure.** A malformed `verdict-guard.sh` exited 2 — the status reserved for "policy failure" — from every script. Each script now checks the load on both sides and exits 3. Across every fixture, script, mode and masked tool, nothing exits outside `{0, 1, 2, 3}`.
- **The JSON encoder no longer depends on your locale.** It escaped through a character class that under a UTF-8 locale matched bytes with negative ordinals, corrupting them into a sixteen-hex-digit escape — including the valid C1 controls, which JSON asks no one to escape. The decision is the ordinal now, so the same bytes come out under `C` and under UTF-8.
- **`DEP001`, `DEP002` and `PATH` are registered rule IDs.** They are in the skill's rule-ID enumeration, in every emitting script's header, and in the README. `scripts/lib/` was flattened, which clears the `deep nesting detected` warning the skill had been carrying, and the layout suite now asserts both shipped skills validate with zero errors *and* zero warnings.
- 66 profiler test functions and 230 subtests pass under `-race`, against 0.4.1's 58 and 179. The shell suites are at 850, against 0.4.1's 134, and were run under both `LC_ALL=C` and a UTF-8 locale. 55 OTLP fixtures, against 39.
- **The two known limits named in v0.4.1 are fixed here, and that note needs two corrections.** Its heading said "both 0.4.2" and its body said "fixed together in 0.5.0" — one of those was always wrong, and 0.4.2 is the right one. And "Both are described in the README and the spec" is no longer true: fixing them deleted the paragraph that described them. The v0.4.1 note is left as written; this is the correction.

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
- **A number that is not a token count no longer becomes one.** A data point's value was carried through an out-of-range float-to-integer conversion, which Go leaves to the machine: the same export read as `input: 9223372036854775807` on arm64 and `input: -9223372036854775808` on amd64. Negative values, `NaN` and infinity were all assimilated too. None of them reaches a profile now, and the reason names the point — in one of two clauses, because they are two different defects. Which of the two you get turns on how the leaf *read*, not on how large the number is. A leaf that *decoded as a finite number* whose rounded value is negative or 2⁶³ or beyond *is not a count* and is refused. A leaf that never decoded as a finite number at all — `NaN`, infinity, or a literal too big for its own reader — says nothing, so the next leaf is read; only when neither leaf yields a number is the point reported as carrying no readable value. So `{"asInt":"9223372036854775808"}` lands in the second clause, not the first: 2⁶³ is finite and numeric, but the integer reader refused it outright.
- **Two token series that differ only by an attribute the adapter does not interpret stay two series.** They used to merge, so a session's tokens were reported as a fraction of themselves.
- **A count the export said nothing about is absent from the profile, not zero.** A cache-only export reported `input: 0, output: 0` beside its real numbers. A count read as zero still reports zero. No profile keys changed names.
- **`capture` exits 2 when it read nothing and something failed.** It exited 0 for a missing or unreadable `--otel-file`, so a script wrapping it filed the all-unknown profile as a success. The profile is still printed.
- **A capture is read to its end, or it is an error.** A stray `}` or `]` after a batch — what a doubled write or a half-unwrapped JSON array leaves behind — used to end the read silently: the batches before it came back `present`, and any batch after it was dropped without a word. That is a part of a capture reported as the whole of it, which a profile cannot show you. Such a file is now `error`, naming the batch and the byte, with nothing from earlier batches used. Blank lines and trailing whitespace after the last batch are still a clean end of file.
- **Two known limits, both 0.4.2.** Under cumulative temporality — not Claude Code's default, which is delta — token counts can undercount two ways: a capture that aggregates several resources merges their series, because resource and scope identity are not part of the series key, and a counter that resets inside one capture discards the run before the reset, because `startTimeUnixNano` is not read. Both are described in the README and the spec, and both are fixed together in 0.5.0.
- **A capture identifies you, and the README says which fields.** The capture recipe no longer sets `OTEL_LOG_TOOL_DETAILS=1`: the adapter reads nothing it adds, and it widens the export.
- **The Devin, Codex and Cursor install rows are marked unverified.** Only the Claude Code route was run against a clean install.
- **The spec now describes the code.** The `MetricResult[T any]` generic never existed; the acceptance criteria described the probe one commit out of date; the fallback reason named env vars the adapter never reads and claimed every metric carries it.
- **The `skill_activation` reason no longer says Claude Code emits no skill activation event.** It does: `claude_code.skill_activated` is logged whenever a skill is invoked, through the Skill tool or a `/` command, and carries `skill.name`, `invocation_trigger`, `skill.source` and `skill.kind`. The reason printed in every Claude Code profile now says what is actually true — this adapter does not read that telemetry yet — and names `claude_code.skill_activated` as the source it will read in 0.5.0. `skill_activation` is still `unknown` with capability `none`; only the justification changed.
- **"Snapshot-pinned" was an overstatement.** `--snapshot` is a label you supply and the profiler records verbatim. It does not hash or verify the skill directory.
- Module path corrected to `github.com/Okja-Engineering/skill-architect/profiler`; CI takes its Go version from `profiler/go.mod` instead of a stale pin four minors behind. The profiler adapter version moves to 0.4.1, since a profile's contents changed.
- 58 test functions pass — 53 in the profiler package, 5 in the CLI — and 179 subtests, under `-race`. The shell suites are at 134: 0.4.0's 131 plus three more — the two new marketplace-manifest assertions and the adapter-version one. CI now runs `gofmt -l`, `go vet` and `go test -race` rather than `go test` alone.
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
