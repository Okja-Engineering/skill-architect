# S1b — Spool → `Profile` (the capture path, and the enrichment seam)

- **Stage:** parked on SC + S1a
- **Owner:** unassigned
- **Branch:** none (target: `epic/skill-scope`)
- **Depends on:** **SC, S1a**
- **Blocks:** S3, S5, S7, S8b, S9, S10

## Goal
`profiler capture --harness cursor --hook-spool <f>` turns a spool into a schema-valid `Profile`. First consumer of `SourceHooks`, which exists in `types.go` and is used by no adapter today (R-SA-05). This is the read half of the old S1. It also lays the **enrichment seam** that keeps every later slice out of this slice's files.

## Definition of done
1. `tool_calls` and `timing` are `present` with `Source: "hooks"` from a spool.
2. **Correlation lands in the contract, not only in the code.** Each `ToolCallEntry` carries `ID` (`tool_use_id`), `GenerationID` (`generation_id`) and, for a subagent branch, `ParentID` (the parent tool call id — `subagentStart.tool_call_id` / `parent_conversation_id`, ledger §A5.1:131); `Profile.SessionID` carries `conversation_id`. The chain `SessionID → GenerationID → ID` **is** R-CS-05, and it is assertable only because SC adds those three fields — round 1's DoD asserted the correlation against a `ToolCallEntry` (`types.go:88-93`) that had `Name`, `Timestamp`, `Success` and nothing else.
3. An un-closed session is closed by a **reader-side** heuristic that sets `TimingData.EndTime` from the last event in the conversation — no long-lived process, no in-process TTL, no leak class (R-CS-06 adapted; the original in-process form is deleted by Fork (i) and is claimed by no slice, including S11).
4. `snapshot_hash` from `CaptureOpts` lands in the `Profile` and survives `Marshal→Unmarshal→Marshal` byte-identically (R-SA-01 **and R-SA-04**).
5. `--session <conversation_id>` selects one conversation — `conversation_id` is the right `SessionID` because it is the join key to both the Admin API and Enterprise OTel (R-SA-08).
6. With no spool and no Cursor present, the adapter returns all-`unknown` and exits 0 (R-SA-13).
7. **The enrichment seam.** `profiler/enrich.go` defines `enrich(p *Profile, ev []SpoolEvent)`, which calls `enrichAttribution` and `enrichTokens`. Both are created here as **no-op stubs** in `profiler/attribution.go` and `profiler/tokens.go` — the files **S3 and S5 then own outright**. Cost: two three-line functions. Benefit: S3 and S5 run in the same wave and edit no file this slice or each other owns, so wave 3 has zero merge contention. Round 1 had S3 and S5 both listing `profiler/hooks.go`, and §F claimed they were in different waves; they were not.

## Test approach
`profiler/capture_hooks_test.go` (new, this slice's own file): a golden spool JSONL → expected `Profile` JSON comparison; a round-trip test (R-SA-01); an explicit `snapshot_hash` assertion (R-SA-04); **a subagent spool asserting `ParentID` chains rather than flattens** (R-CS-05's "subagents branch" half, now checkable); a two-conversation spool proving `--session` selects one; a session with no terminal event proving the reader-side close sets `EndTime`; a no-Cursor/no-spool degradation test asserting all-`unknown` and exit 0 (R-SA-13); a test that `enrich` with both stubs in place leaves the profile unchanged.
`ToolCallEntry.Status`/`Success` derive from the **documented** `afterShellExecution` fields (`command`, `output`, `duration`, `sandbox`). **`exit_code` is a Local hypothesis** — Cursor's docs do not list it and the ledger's payload table (§A5.1:124) omits it; it traces only to cursorscope 0.3.7. If SP1's real capture shows it, use it; otherwise `Status` is left **empty**, which is exactly the case SC's `successRate` keeps on the legacy `Success`-based path. No DoD depends on an undocumented field.

## Requirements
R-CS-05, R-CS-06 (adapted), R-CS-15 (adapted — see above), R-SA-01, R-SA-04, R-SA-05, R-SA-08, R-SA-13, R-SA-02

## Files
`profiler/cursor.go`, `profiler/hooks_read.go` (new), `profiler/enrich.go` (new), `profiler/attribution.go` (new, stub — owned by S3 thereafter), `profiler/tokens.go` (new, stub — owned by S5 thereafter), `profiler/cmd/main.go`, `profiler/capture_hooks_test.go` (new)

## Next
`profiler/cursor.go` is also written by S0 (same wave), S6, S7 and S9 — the roadmap's old claim that "only S0 and S1 touch it" was false. S0 owns the `extractCursor*` helpers and `Probe`; this slice adds a hook-spool source branch. Expect **one** conflict with S0 and rebase on it rather than serializing the epic. That is the only same-wave file collision left in the epic; the seam in DoD 7 removed the other one.
