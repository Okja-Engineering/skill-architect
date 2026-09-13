# PR #2 release/0.4.1 — reconciled worklist, round 1

_Chief of staff, 2026-09-13 15:50. Head reviewed: 7cb32e2. Streams: 3 internal hunters (code line-by-line, DoD conformance, docs truth). External reviewer: silent past the 20-minute window (last push 15:25); hunters are the gate._

## Verdict on 7cb32e2: NOT CLEAN. Do not merge.

## Root 1 — a signal is `present` on structural evidence, not on a value actually read (code, REAL)

Invariant: a signal is `present` only when a real value was assimilated from the source; otherwise `unknown` with a reason. Probe's availability predicate and Capture's extraction predicate are the same predicate per signal.

- **W1** `profiler/claude_code.go:171-197` — `found = true` on metric-name match before the `token_type` switch; unrecognized/missing type or non-numeric value ⇒ `present` with all-zero counts. Reproduced three ways. Violates README:28 "never invented" and spec "present ⇒ Value populated".
- **W2** `profiler/claude_code.go:217-235` — timing uses `start == ""` as its found-sentinel (tokens/tool calls use real found checks). Events present but no top-level `timestamp` ⇒ probe says available, capture says "no api_request events found". Falsifies AC9, README:66, RELEASE_NOTES:7. Also first/last in file order, not min/max by timestamp ⇒ negative `total_ms` on out-of-order logs. Invariant: `end_time >= start_time`, `total_ms >= 0`.
- **W3** `profiler/profiler_test.go:409-460` — contract test iterates a hardcoded 3-signal list, not `report.Capabilities`; never asserts `Value` non-empty on `present`; no malformed/no-timestamp/unknown-type/out-of-order shapes. Extend so W1/W2 are pinned to the invariant (RED before fix).
- **W4** `profiler/claude_code.go:206` `Success: decision == "approved"` conflates permission decision with execution outcome. DEFERRED to 0.5.0 (schema meaning change); document in spec that `success` currently means "approved", and list under found-not-fixed.

## Root 2 — Capture below the new gate still assumes the old token-proxy model (code, REAL)

Invariant: Probe or Capture owns file resolution and failure classification, not both; all three signals share the same error path; a provided-but-unusable export is diagnosable as such.

- **W5** `profiler/claude_code.go:99-105, 243` — malformed/truncated export ⇒ "OTel export not configured. Provide an OTel export file" (user did). Parse error swallowed in `hasOtelSignals`.
- **W6** `profiler/claude_code.go:113-138` — three unreachable error branches; tokens gets `ErrorTokenResult` while the other two get `Unknown*`; `MetricError` now unemittable by the only adapter. Spec self-contradicts (Rules say parse failure ⇒ `error`; Fallback says ⇒ `unknown`). Pick one and make code, spec, and test agree.
- **W7** `profiler/claude_code.go:108-111`, `cmd/main.go:14, 82-86`, README:63 — `--export-file` fallback is dead code; the flag is advertised and documented as a no-op. Invariant: a flag the selected adapter cannot honor fails loudly or is not offered.
- **W8** `profiler/claude_code.go:194, 212, 228` — signal names still literal inside reason strings after the "named once" refactor. Use the constants.
- **W9** `profiler/types.go:189` `AdapterVersion = "0.1.0"` — profile output behavior changed in this PR (partial exports now yield present data); bump to `0.4.1` so artifacts are distinguishable. Add nothing else.

## Root 3 — the release's truth bar was applied to targets, not to the new prose or their siblings (docs, REAL)

Invariant: every sentence added or edited on this branch, and every sibling line in a block it edited, is true of the code at head. Re-walk them; don't patch the three numbers.

- **W10** `CHANGELOG.md:16` "two minors behind" ⇒ four (1.23 → 1.27). `RELEASE_NOTES.md:11` "two commits out of date" ⇒ one (only 30f374c between 0.4.0 and base).
- **W11** `docs/profiler-spec.md:125` interface comment still names deleted `OtelEndpoint`.
- **W12** `profiler/types.go:17,133,140,147,153,160` comments name `MetricResult`, the generic the spec now says never existed. `CHANGELOG.md:41`, `RELEASE_NOTES.md:20` still list "`MetricResult` (present/unknown/error)" as delivered — correct in place (the branch already corrects sibling 0.4.0 bullets in place, so the "history stays" rule is not what was applied).
- **W13** Item 15 as an invariant, not a substitution: no doc/comment asserts an absolute about Claude Code's OTel surface that the published reference contradicts. Sites: `docs/profiler-spec.md:215`, `profiler/claude_code.go:26` ("no output-to-skill mapping" — `skill.name` is on `api_request`, which this adapter reads, and `tool_result` carries `skill_name`); `profiler/claude_code.go:66` (untouched "no skill-level activation or attribution events"); `:94` and `spec:224` reason string ("does not attribute outputs to skills"); `README.md:70`, `CHANGELOG.md:30` scope `skill.name` to token/cost only. Mandate wording: "the adapter does not yet read skill-level attributes."
- **W14** `docs/profiler-spec.md:227, 235` — AC5 and Fallback say all metrics get the "OTel export not configured" reason; `skill_activation`/`attribution` get their own reasons. Make the sentence true.
- **W15** `README.md:105, 123` — `claude plugins install Okja-Engineering/skill-architect` and `claude plugins install .` both fail as written (`claude plugin install` takes `name[@marketplace]`; no `.claude-plugin/marketplace.json` exists). This is 0.4.0 promise P11/P13. Invariant: the documented install command works on a clean machine. Direction: document the route that works; a minimal `.claude-plugin/marketplace.json` is the standard mechanism if the shorthand is to be kept — decide and verify live with `claude plugin install`.
- **W16** small truths: `README.md:54` version prints `profiler 0.1.0` not `0.1.0`; `README.md:87` "without validating anything" overstated (PL001 license gate still runs, exits 2 on a license-less skill); `README.md:135-136` resolves relative to the skill dir, not the script's dir; `docs/profiler-spec.md:3` header still "spec · 2026-09-10"; `CHANGELOG.md:45`/`RELEASE_NOTES.md:24` "15 tests" for 0.4.0 vs `CHANGELOG.md:28` "15 after 30f374c" — say which commit the count is at.
- **W17** `status.md` additions list omits the AC4/AC5/preamble edits in 7cb32e2 and omits `:94`/`spec:224`, `RELEASE_NOTES.md:20`, `types.go` comments from found-not-fixed. Record them.
- **W18** `.github/workflows/ci.yml:36-37` — add `cache: false` to the touched `setup-go` block (module has no go.sum; permanent warning). `go vet`/`-race`/`gofmt` in CI and pinning the curl/npm installs: DEFERRED to 0.5.0, record.

## Fork — needs the user (ROOT A from the docs hunter)

The Claude Code adapter does not parse OTel. `profiler/claude_code.go:154-169` reads a bespoke envelope `{"metrics":[{name,attributes,value}],"logs":[{event_name,attributes,timestamp}]}`; real OTLP/JSON is `{"resourceMetrics":[{"scopeMetrics":[...]}]}`. Attribute names are invented: `token_type` (real: `type`), `cache_creation`/`cache_read` (real: `cacheCreation`/`cacheRead`), `reasoning` (no such type), `decision` (real: `decision_type` ∈ accept/reject). Anthropic's docs say Claude Code has no file exporter, so the spec's `CLAUDE_CODE_ENABLE_TELEMETRY=1` setup cannot produce a `--otel-file`. A realistic OTLP export ⇒ all signals `none`. The adapter's envelope with real attribute names ⇒ `present` with zeros (W1). Pre-existing at 0.4.0; this PR did not introduce it, but 0.4.0's headline promise rests on it. Nothing in the repo documents the envelope the adapter actually reads.

Options (user decides; W1–W18 proceed regardless):
- **A (recommended for a patch): truth only.** Name and document the envelope the adapter reads (schema, example), align attribute names and values with Claude Code's real ones (`type`, `cacheRead`/`cacheCreation`, `decision_type`), drop the "reasoning tokens" claim, state that Claude Code has no file exporter and that OTLP/JSON from a collector is not parsed until 0.5.0, and fix the spec's setup instructions. No new parser.
- **B: parse OTLP/JSON in 0.4.1.** Adds a real parser for `resourceMetrics`/`resourceLogs`; makes the 0.4.0 promise literally true; larger change, new test corpus.

## Dispositions
- REAL, route to bug-fixer now: W1, W2, W3, W5, W6, W7, W8, W9, W10, W11, W12, W13, W14, W15, W16, W17, W18 (cache: false only).
- DEFERRED (0.5.0, recorded in status found-not-fixed): W4 rename; CI vet/race/gofmt; unpinned CI installs; the two silent-failure scripts already recorded.
- Fork DECIDED (user, 2026-09-13 15:55): **Option B — parse real OTLP/JSON in 0.4.1.** Lands as its own unit after round-1 repair: researcher pins the wire shape (`otlp-research.md`), then a senior-implementer builds the parser (mandate item 19). Round-1 bug-fixer leaves envelope/attribute sentences untouched, listed as pending.
- INVALID: none.
