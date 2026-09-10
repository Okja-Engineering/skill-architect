# SC — Profile contract v1.1 (additive Source/Reason channels, source-honest compare)

- **Stage:** ready — **ships first**
- **Owner:** unassigned
- **Branch:** none (target: `epic/skill-scope`)
- **Depends on:** none
- **Blocks:** S0, S1b, S3, S5, S6, S7, S8

## Goal
One slice owns `profiler/types.go`, `profiler/compare.go` and `docs/profiler-spec.md` and gives the contract the vocabulary the rest of the epic asserts against. Created by spec-gate root **R1**: four DoDs were asserting distinctions the F03 contract cannot express.

## Definition of done
1. `ActivationEntry` gains `Source string` (carries `cursor.skill.source`: `unspecified|workspace|user|builtin|plugin|claude`) — R-RL-05.
2. `ToolCallEntry` gains `Status string` (tri-state pass-through `success|error|aborted`) and `Count int` (`omitempty`, >1 when the row derives from a counter). `Success bool` stays and equals `Status == "success"`.
3. `MetricSource` gains `estimated`, `otel_aggregate`, `mcp_reported`.
4. `CapabilityReport` gains `Notes map[MetricName]string` (**why** a metric is `none` — today `Capabilities` is `map[MetricName]MetricSource`, `types.go:50-55`, with no reason channel at all) and `Available map[MetricName][]MetricSource` (a metric may have two sources; today the second assignment silently overwrites the first, `types.go:54`).
5. Every comparator carries **both** sides' sources (`BaselineSource`, `CandidateSource`) and **refuses** a cross-honesty-class comparison, naming both classes. Today `compareTokens` (`compare.go:106-126`) gates only on `State == present` and reports the *candidate*'s source, so an `estimated` baseline against a `server_api` candidate yields `comparable:true, source:"server_api"` and the estimate's honesty label vanishes.
6. `MetricPresent`'s doc comment stops saying "value captured from telemetry" (`types.go:12`, `profiler-spec.md:19,33`) — D2.
7. `docs/profiler-spec.md` revised to **v1.1** with a changelog, the honesty-class table (D3), the `Trigger` vocabulary (D6), and a "considered, not in v1.1" note for R-CS-14. Wire `schema` string stays `skill-architect/profile/v1` (D1).

## Test approach
`profiler/contract_test.go` (new, this slice's own file):
- a golden **pre-v1.1** profile JSON unmarshals and re-marshals byte-identically → proves every change is additive (D1);
- a table asserting every `MetricSource` value maps to exactly **one** honesty class, and that adding a source without a class fails the test;
- `compareTokens(baseline: estimated, candidate: server_api)` → `comparable:false` with a reason naming both classes;
- `compareTokens(baseline: hooks, candidate: otel)` → `comparable:true` reporting **both** sources;
- `Probe()`-shaped table asserting `Notes` and `Available` round-trip through JSON.
All **34 existing Go tests stay green** — no field is removed, no default changes.

## Requirements
R-RL-05, R-RL-06, R-RL-03 (the `Notes` channel it needs), R-SA-02, R-SA-03 (superseded form), R-SA-06 · Design decisions D1, D2, D3, D6, D8

## Files
`profiler/types.go`, `profiler/compare.go`, `docs/profiler-spec.md`, `profiler/contract_test.go` (new)

## Next
Dispatchable **now**: no dependencies, no Cursor, no user answer, and it unblocks seven slices. Not gated on User Question 3 — Q3 decides whether S5 *produces* an estimate, not whether the constant exists; if Q3 rejects estimates entirely, S5's PR deletes the unused constant. Do not let this slice grow into behaviour: it adds channels and one comparison rule, nothing that reads a hook or an export.
