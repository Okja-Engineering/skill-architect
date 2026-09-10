# SC — Profile contract v1.1 (additive channels, identity, source-honest compare)

- **Stage:** ready — **ships first**
- **Owner:** unassigned
- **Branch:** none (target: `epic/skill-scope`)
- **Depends on:** none
- **Blocks:** S0, S1b, S3, S5, S6, S7, S8a, S8b

## Goal
One slice owns `profiler/types.go`, `profiler/compare.go` and `docs/profiler-spec.md` and gives the contract the vocabulary the rest of the epic asserts against. Created by spec-gate root **R1**. **Widened at round 2** by a representability sweep: every DoD assertion in every slice was walked against the F03 types, and four more assertions turned out to have no carrier at all (tool-call identity, context-window fields, `SourceNone`'s honesty, an unrecognized wire `Source`). Those channels are added here or the assertions are struck — nothing downstream may assert what this file cannot express. The sweep table is in `review/revision-log-round2.md`.

## Definition of done
1. `ActivationEntry` gains `Source string` — carries `cursor.skill.source`, whose six values `unspecified|workspace|user|builtin|plugin|claude` are **Specification** (ledger §A5.2:169, Cursor OTel Wire Reference). R-RL-05.
2. `ToolCallEntry` gains `Status string`, a **verbatim pass-through of `cursor.tool.status`, whose vocabulary is `success | failure | aborted`** (ledger §A5.2:159). *There is no `error` value* — round 1 invented one; a pass-through field's vocabulary is the source's vocabulary, unchanged. `Count int` is added for counter-derived rows. `Success bool` stays and equals `Status == "success"` when `Status` is set.
3. `ToolCallEntry` gains the identity/hierarchy fields **`ID string`** (`tool_use_id`), **`GenerationID string`**, **`ParentID string`** (the parent tool call for a subagent branch). Without them S1b's DoD 2 (`conversation_id` → `generation_id` → `tool_use_id`, subagents branch) is unrepresentable, and `Attribution.Target` (`types.go:114-118`, documented as "tool call ID") refers to nothing. `Profile.SessionID` carries `conversation_id`; the chain is `SessionID` → `GenerationID` → `ID`, with `ParentID` for branches.
4. `TokenCounts` gains **`ContextTokens int`**, **`ContextWindowSize int`**, **`ContextUsagePercent float64`** — `preCompact` supplies all three (ledger §A5.1:136) and R-CS-13/R-RL-18 assert them; `TokenCounts` (`types.go:80-86`) had no field for any of them and S5 does not own `types.go`.
5. `MetricSource` gains `estimated`, `otel_aggregate`, `mcp_reported`.
6. `CapabilityReport` gains `Notes map[MetricName]string` (**why** a metric is `none` — today `Capabilities` is `map[MetricName]MetricSource`, `types.go:50-55`, with no reason channel) and `Available map[MetricName][]MetricSource` (a metric may have two sources; today the second assignment silently overwrites the first, `types.go:54`).
7. **Honesty classes are total and fail-closed (D3).** All nine `MetricSource` constants map to exactly one class, **including `SourceNone` (`types.go:46`) → class `none`**; a `present` result carrying `none` is a contract violation. Because `RawMetricResult.Source` is a free `string` (`types.go:23`), any value outside the nine maps to class **`unrecognized`** and `compare` refuses it, naming the string — a legacy or typo'd source fails closed, never open.
8. Every comparator carries **both** sides' sources (`BaselineSource`, `CandidateSource`) and refuses a cross-class comparison, naming both classes. Today `compareTokens` (`compare.go:106-126`) gates only on `State == present` and reports the *candidate*'s source, so an `estimated` baseline against a `server_api` candidate yields `comparable:true, source:"server_api"` and the estimate's honesty label vanishes.
9. **Tool-call comparison is encoding-invariant.** `countToolCalls` (`compare.go:148-149`) is `len(calls)`; once S0 emits aggregate rows with `Count`, two profiles of the *same* session — one per-call, one aggregated — compare as `comparable:true` with a nonzero count delta. Fix: total sums `Count` (an absent or zero `Count` counts as 1) and successes sum `Count` over rows whose `Status == "success"`. **Invariant: two profiles of one session yield zero delta regardless of row encoding.**
10. **`successRate` handles the tri-state.** `successRate` (`compare.go:158-162`) divides successes by *all* rows, conflating `failure` and `aborted`. Fix: when `Status` is set, the report carries `successes`, `failures` and `aborted` separately and `success_rate = successes / (successes + failures)` — an aborted call is neither a tool success nor a tool failure. When `Status` is empty (pre-v1.1 rows, or hooks without a status) the old `Success`-over-all-rows behaviour is kept exactly, so the 34 existing tests do not move.
11. `MetricPresent`'s doc comment stops saying "value captured from telemetry" (`types.go:12`, `profiler-spec.md:19,33`) — D2.
12. **Every field added by items 1–6 carries `omitempty` and is declared after the existing fields of its struct.** This is what makes DoD item 13 true rather than contradictory: round 1 promised byte-identity while specifying `omitempty` for `Count` alone, so `Source`/`Status`/`Notes`/`Available` would have emitted `""`/`null` and broken it.
13. A pre-v1.1 golden profile unmarshals **and re-marshals byte-identically** (D1). Wire `schema` stays `skill-architect/profile/v1`.
14. `docs/profiler-spec.md` revised to **v1.1** with a changelog, the honesty-class table (D3), the two `Trigger` vocabularies (D6), the tool-call encoding-invariance rule, and a "considered, not in v1.1" note for R-CS-14 (D8).

## Test approach
`profiler/contract_test.go` (new, this slice's own file):
- a golden **pre-v1.1** profile JSON unmarshals and re-marshals **byte-identically** (`Marshal(Unmarshal(golden)) == golden`) → proves both additivity and DoD 12's `omitempty` discipline;
- a table asserting all **nine** `MetricSource` constants map to exactly one honesty class, that `SourceNone` maps to `none`, that adding a constant without a class fails the test, and that the wire string `"wat"` maps to `unrecognized`;
- `compareTokens(baseline: estimated, candidate: server_api)` → `comparable:false` with a reason naming both classes;
- `compareTokens(baseline: "wat", candidate: hooks)` → `comparable:false` naming the unrecognized string (fail-closed);
- `compareTokens(baseline: hooks, candidate: otel)` → `comparable:true` reporting **both** sources;
- **encoding invariance:** two profiles of one 42-call session — one with 42 per-call rows, one with aggregate rows carrying `Count` — compare to **zero** delta on `count`, `successes` and `success_rate`;
- tri-state `successRate`: a fixture of 3 `success` / 1 `failure` / 2 `aborted` yields `success_rate` 0.75 and a separate `aborted: 2`; a `Status`-less fixture reproduces today's number exactly;
- **referential integrity:** every `Attribution.Target` in a profile resolves to some `ToolCallEntry.ID` in the same profile (this is what makes S3's DoD 3 checkable);
- a `Probe()`-shaped table asserting `Notes` and `Available` round-trip through JSON.

All **34 existing Go tests stay green** — no field is removed, no default changes.

## Requirements
R-RL-05, R-RL-06, R-RL-03 (the `Notes` channel it needs), R-CS-05 and R-CS-13/R-RL-18 (the *carriers* only; the producers are S1b and S5), R-SA-02, R-SA-03 (superseded form), R-SA-06 · Design decisions D1, D2, D3, D6, D8

## Files
`profiler/types.go`, `profiler/compare.go`, `docs/profiler-spec.md`, `profiler/contract_test.go` (new)

## Next
Dispatchable **now**: no dependencies, no Cursor, no user answer, and it unblocks eight slices. Not gated on User Question 3 — Q3 decides whether S5 *produces* an estimate, not whether the constant exists. **If Q3 rejects estimates, the `estimated` constant is retained anyway**: S8a's guardrail 2 and D3's class table both bind to it, and the round-1 plan's "S5's PR deletes the unused constant" would have broken an MVE slice. Do not let this slice grow into behaviour: it adds channels, one comparison rule and two counting fixes — nothing that reads a hook or an export.
