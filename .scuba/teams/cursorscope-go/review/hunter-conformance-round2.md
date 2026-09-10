# Hunter report — telemetry-name and factual conformance lens (spec gate, round 2 confirming pass)

**Artifact:** revised `roadmap.md` + 16 slice files · **Date:** 2026-09-10 · **Verdict:** NOT-CLEAN — 12 REAL (1 HIGH, 3 MED, 8 LOW/LOW-MED), 0 DEFERRED, 1 INVALID

## Coverage
17/17 files walked twice · 28 telemetry names (22 `cursor.*`, 3 `gen_ai.*`, 1 `skill_architect.*`, 2 legacy `cursor_*`) — 0 invented · 6 enum value-sets / 33 values — 1 invented · 21-event hook surface + 34 payload/base fields · 9 Admin-API strings + 3 limit claims (re-fetched live) · 8 package/version strings · 66 cited paths · ~45 numeric claims · 19 round-1 items + 3 groomer partial-disagreements. Live-verified: Cursor Wire Reference, hooks docs, Admin API docs; ledger; cursorscope clone; `go test` (34 PASS), 4 bash suites (131, 0 fail).

## Round-1 fix status
C-1 ✅ · C-2 ⚠ partial (F-8) · C-3 ✅ (60/min confirmed live; groomer's hedge discharged) · C-4 ✅ (no 30-day limit on `filtered-usage-events` — confirmed) · C-5…C-10 ✅ · C-11 ⚠ partial (F-9) · C-12 ✅ · C-13 ⚠ **regressed** (F-1) · C-14 ✅ · C-15 ⚠ partial (F-9) · C-L-1…4 ✅

## REAL
1. **HIGH** `slices/SC.status.md:14` — `ToolCallEntry.Status` "tri-state pass-through `success|error|aborted`". Wire Reference: `cursor.tool.status` = `success | failure | aborted`. **`error` is invented.** C-13's fix introduced it. Invariant: a pass-through field's vocabulary is the source's vocabulary verbatim.
2. **MED** `roadmap.md:120` (D3) — "Every `MetricSource` maps to exactly one class" enumerates 8 of 9 members; `SourceNone` (`types.go:46`) is unmapped. `SC.status.md:24`'s own test goes RED on day one.
3. **MED** `roadmap.md:139` + `slices/S1a.status.md:13` — "each of the 21 events carrying the eight base fields" is false: docs state `workspaceOpen` omits `conversation_id`, `generation_id`, `model`, `model_id`, `transcript_path`. `roadmap.md:41` presents 8 of the 10 documented base fields as the base set.
4. **MED — Design-decision gap.** No reuse-vs-build decision against `dash0hq/dash0-agent-plugin` (Apache-2.0, Go+shell, 9 Cursor hooks → OTLP) or `o11y-dev/opentelemetry-hooks` anywhere in the 17 files. `roadmap.md:84` scopes R-SA-10 to `jq`/`skill-validator`/`skillscore`; `AGENTS.md:8-9` makes the check mandatory. Evidence: `cursor-profiler/docs/research/existing-per-skill-cost-tools.md:119-127`. Needs a D10 recording build-over-fork (9-of-21-event ceiling, Dash0-shaped span model, no skill span) and Python-exclusion for the MIT one.
5. **LOW-MED** `roadmap.md:69,139` cite "§G6" for the unmeasured latency budget; §G6 is the Admin API item, latency is §G7 (numbering shifted by the inserted §G3).
6. **LOW-MED** `roadmap.md:109` (R-RL-13) — `skill_architect.skill.*` labelled Specification (semconv 1.44.0); nothing specifies it; ledger:214 calls it a Design decision. Same class: `roadmap.md:50` (R-CS-11 as Repository fact), `roadmap.md:101` (R-RL-05).
7. **LOW** `roadmap.md:104` (R-RL-08) — `src/cursor-api-poller.js:44,47` labelled Repository fact; that file is in the cursorscope clone → External evidence per `AGENTS.md:21`.
8. **LOW-MED** `slices/S3.status.md:12` + `roadmap.md:142` — "the first activation that names a real skill" is still wrong: S0 (wave 2) emits `present` with a non-empty `cursor.skill.name` before S3 (wave 3). Only "first from a non-Enterprise surface" survives.
9. **LOW-MED** `slices/S6.status.md:23` + `roadmap.md:145` — S6 must introduce new `cursor.*` literals but its Files omit `profiler/cursor_wirekeys.go`, so D5 + S8 guardrail 4 make S6 unbuildable as scoped. `slices/S8.status.md:19` widens the allowance to `*_test.go`, which D5 (`roadmap.md:122`) does not grant.
10. **LOW** `roadmap.md:144` — "billed → `server_api`/`otel` (S6/S9)" is inverted; S9 is Admin API (`server_api`), S6 is Enterprise OTel (`otel`). `S5.status.md:22` has it right.
11. **LOW** `review/revision-log-round1.md:71` — "26 edges"; the graph has 28.
12. **LOW** `roadmap.md:155` — SP1 called "a ~40-line PR"; its DoD commits a full redacted session capture plus a test and a spike doc.

## Out-of-lens note
`roadmap.md:159` — the "if SP1 answers no" MVE retains S8, but S8 depends on S3, which that branch withdraws.

## Checked, clean
All 22 `cursor.*` strings trace to the Wire Reference (fetched live). `cursor.token.type`, `cursor.skill.trigger`, `cursor.skill.source`, `cursor.api.request.*_tokens` enums verbatim-correct. 21-event surface, `afterShellExecution` fields (no `exit_code`), `preCompact` fields, `SKILL_PATH_RE` confirmed. 60 req/min for `filtered-usage-events` and 20/min + 30-day for `daily-usage-data`/`audit-logs` confirmed live. `tokenUsage{inputTokens,outputTokens,cacheWriteTokens,cacheReadTokens}`, Basic auth, `conversationId` ✓. 131/34 test counts reproduced. All cited `types.go`/`compare.go`/`cursor.go`/`profiler_test.go`/`profiler-spec.md`/`AGENTS.md` lines verified. Fork (v) numbers, ACES figures, calibration range verified. `gen-ai-semconv.test.js` as an S3 oracle: INVALID as a finding — it does assert CLI/skill invocation labeling (`:92`).

## Shared root (1, 6, 7, 10)
Evidence labels and vocabularies are still transcribed from a row's topic rather than re-derived from the cited source at the moment of writing — the same Root B. One root fix: for every Specification/Repository-fact label and every enum literal, cite the source line that carries it, and treat "pass-through" as binding the vocabulary verbatim.
