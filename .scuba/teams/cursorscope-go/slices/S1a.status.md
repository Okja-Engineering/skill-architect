# S1a — Hook entrypoint: stdin JSON → JSONL spool

- **Stage:** ready — dispatchable now
- **Owner:** unassigned
- **Branch:** none (target: `epic/skill-scope`)
- **Depends on:** none
- **Blocks:** S1b, S2, S4, S11

## Goal
One static Go binary reads a Cursor hook payload on stdin and appends a normalized line to a JSONL spool. No daemon, no port, no long-lived state (Fork i). This is the write half of the old S1, split at the spec gate because the original claimed 13 requirements and its DoD proved about three.

## Definition of done
1. For **each of the 21** documented hook events (**Specification**, ledger §A5.1:115-137 — 21 table rows), a payload on stdin appends exactly **one** normalized JSONL line carrying **whichever of the 10 documented base fields that event actually supplies**. The ten are `conversation_id`, `generation_id`, `model`, `model_id` (opt), `model_params` (opt), `hook_event_name`, `cursor_version`, `workspace_roots`, `user_email`, `transcript_path` (ledger §A5.1:113). **`workspaceOpen` supplies only four** — `hook_event_name`, `cursor_version`, `workspace_roots`, `user_email` — omitting `conversation_id`, `generation_id`, `model`, `model_id` and `transcript_path` (ledger §A5.1:137). Round 1 asserted a flat "eight base fields on all 21 events", which is false in two directions: the documented set is ten, and one event carries four. Absent fields are written empty, **never invented** (R-SA-02).
2. Malformed JSON, an unwritable spool path, and an unrecognized `hook_event_name` each exit **0** with empty stdout — a hook must never break the Cursor session (R-CS-03).
3. **Bounded memory (R-CS-30's other half).** stdin is consumed through a bounded reader with a documented cap, so payload size cannot grow the process: a **10 MB** payload on stdin still exits 0, writes exactly one line no larger than the cap, and does not buffer the whole input. Round 1 asserted "bounded memory" in the register and proved it nowhere.
4. Spool path resolves from a flag or an environment variable. **No dotenv** — a Design decision, not a repo rule: `AGENTS.md` (23 lines) says nothing about env files.
5. Single static Go build; `profiler/go.mod` still declares zero `require`s (R-CS-28 — the port's actual justification over Node ≥20 + `npm install`).

## Test approach
`profiler/hooks_test.go` (new, this slice's own file): a table over **21** payload fixtures — one per event — asserting the exact spool line each produces, with `workspaceOpen` asserted as the four-field case rather than being forced through the blanket rule. That table is what actually proves R-CS-01 and R-CS-02 rather than asserting them in prose. Three failure-mode tests (malformed JSON, read-only spool dir, unknown event) each assert exit code 0 and empty stdout. One bounded-memory test feeds 10 MB on stdin and asserts exit 0 plus a capped line.
`tests/test_hook_latency.sh` (new): runs the binary N times over a representative payload, **records** p50/p95 as a **Repository fact** in the PR body, then pins the CI ceiling at **2× the measured p95** in `.github/workflows/ci.yml`. §G7 states the latency budget is unmeasured; the old DoD gated on an invented "<50 ms". This slice produces the number first and derives the gate from it.

## Requirements
R-CS-01, R-CS-02, R-CS-03, R-CS-04 (adapted — the spool *is* the ingestor, D9), R-CS-07, R-CS-28, R-CS-30 (both halves: latency measured, memory bounded), R-SA-13, R-SA-12, R-SA-02

## Files
`profiler/cmd/cursor-hook/main.go` (new), `profiler/hooks.go` (new), `profiler/hooks_test.go` (new), `tests/test_hook_latency.sh` (new), `.github/workflows/ci.yml` (the latency ceiling — S8a is the only other writer, three waves later)

## Next
Dispatchable in parallel with SC — they share no files. Hook field names are **Specification** from Cursor's docs, not captured payloads (§G1): the 21 fixtures are hand-authored, and **SP1's real capture is what upgrades them**. If SP1 lands first, use its capture to author the fixtures instead.

`profiler/hooks.go` is created here and edited by exactly one other slice: **S4**, which inserts the redaction call on the write path in wave 2. The read path is a separate file (`hooks_read.go`, S1b), and S3/S5 fill stubs in their own files — so this file has no same-wave contention at any point.
