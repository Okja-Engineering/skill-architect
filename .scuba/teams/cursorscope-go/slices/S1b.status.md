# S1b — Spool → `Profile` (the capture path)

- **Stage:** parked on SC + S1a
- **Owner:** unassigned
- **Branch:** none (target: `epic/skill-scope`)
- **Depends on:** **SC, S1a**
- **Blocks:** S3, S5, S7, S9, S10

## Goal
`profiler capture --harness cursor --hook-spool <f>` turns a spool into a schema-valid `Profile`. First consumer of `SourceHooks`, which exists in `types.go` and is used by no adapter today (R-SA-05). This is the read half of the old S1.

## Definition of done
1. `tool_calls` and `timing` are `present` with `Source: "hooks"` from a spool.
2. Events correlate `conversation_id` → `generation_id` → `tool_use_id`, and subagent invocations branch rather than flatten (R-CS-05).
3. An un-closed session is closed by a **reader-side** heuristic — no long-lived process, no TTL sweep, no leak class (R-CS-06 adapted; the original in-process TTL form is deleted by Fork (i) and is not re-claimed by S11).
4. `snapshot_hash` from `CaptureOpts` lands in the `Profile` and survives `Marshal→Unmarshal→Marshal` byte-identically (R-SA-01 **and R-SA-04**, which the old plan assigned to S1 and then never mentioned again).
5. `--session <conversation_id>` selects one conversation — `conversation_id` is the right `SessionID` because it is the join key to both the Admin API and Enterprise OTel (R-SA-08).
6. With no spool and no Cursor present, the adapter returns all-`unknown` and exits 0 (R-SA-13).

## Test approach
`profiler/capture_hooks_test.go` (new, this slice's own file): a golden spool JSONL → expected `Profile` JSON comparison; a round-trip test (R-SA-01); an explicit `snapshot_hash` assertion (R-SA-04); a two-conversation spool proving `--session` selects one; a no-Cursor/no-spool degradation test asserting all-`unknown` and exit 0 (R-SA-13).
`ToolCallEntry.Status`/`Success` derive from the **documented** `afterShellExecution` fields (`command`, `output`, `duration`, `sandbox`). **`exit_code` is a Local hypothesis** — Cursor's docs do not list it and the ledger's payload table (§A5.1:124) omits it; it traces only to cursorscope 0.3.7. If SP1's real capture shows it, use it; otherwise `Status` is `unknown` and the test pins that. No DoD depends on an undocumented field.

## Requirements
R-CS-05, R-CS-06 (adapted), R-CS-15 (adapted — see above), R-SA-01, R-SA-04, R-SA-05, R-SA-08, R-SA-13, R-SA-02

## Files
`profiler/cursor.go`, `profiler/hooks.go`, `profiler/cmd/main.go`, `profiler/capture_hooks_test.go` (new)

## Next
`profiler/cursor.go` is also written by S0 (same wave), S6, S7 and S9 — the roadmap's old claim that "only S0 and S1 touch it" was false. S0 owns the `extractCursor*` helpers and `Probe`; this slice adds a hook-spool source branch. Expect **one** conflict with S0 and rebase on it rather than serializing the epic.
