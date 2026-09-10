# Hunter report — slice verifiability lens (spec gate, round 5 final confirming pass)

**Artifact:** `roadmap.md` (354 ln) + 17 slice files · **Date:** 2026-09-10 · **Verdict:** NOT-CLEAN — 2 HIGH, 3 MED, 9 LOW, 1 SUSPECTED, 0 INVALID

**Coverage:** 17/17 slices × 6 checks · roadmap §A–§H · 65/65 requirement IDs both directions · 35/35 edges recomputed · 5/5 S8a rules run against the tree · 24/24 code cites resolved · 2 sweeps. `go vet` clean; `go test` ok, 34 tests.

## REAL

| # | file:line | Sev | Detail (proven) |
|---|---|---|---|
| 1 | `S8a.status.md:27` | **HIGH** | Rule 5(a) `grep -rn '\.Tokens[[:space:]]*=' profiler/` minus `tokens_select.go`/`*_test.go` must be zero. Ran it: 24 hits. After S0 clears `cursor.go`'s 6, 18 remain in `claude_code.go` (8), `codex.go` (5), `devin.go` (5) — adapters no E-CS slice touches. S8a is in both MVEs, so CI goes red the day it merges. Same class as C4-#1; rule 5's grep was never executed at the gate. |
| 2 | `S8a.status.md:28` vs `SC.status.md:46` | **HIGH** | Rule 5(b) AST walk fails any `PresentTokenResult` with `Source` in `SourceHooks|SourceSessionData|SourceSQLite|SourceNone|""`. `claude_code.go:465` is `PresentTokenResult(tc, string(SourceSessionData))` — legitimate, and SC item 15 maps `session_data` to `harness_reported`. Package-wide, so ships red and contradicts SC's own honesty table. D12's justification supports banning `hooks` on the Cursor path only. |
| 3 | `SC.status.md:22-24`; `S0:16`, `S1b:32`, `S5:27`, `S6:19`, `S9:15` | MED | The single assignment site is unowned. `SelectTokenSource(cands, sessionRequested) TokenResult` has no `*Profile` param, so a caller must write `p.Tokens = …`; all five producers assert they assign nowhere; the only allowlisted file `tokens_select.go` structurally cannot contain the line. Either branch fails. Nobody owns the candidate accumulator either. |
| 4 | `S0.status.md:17` vs `SC.status.md:26`, `S7:17` | MED | Session-eligibility mechanized for `tokens` only. `cursor.tool.calls` is a delta counter with no `cursor.conversation.id`, yet `capture --session X` emits export-wide rows as that session's `tool_calls` with no eligibility rule. S7 DoD 5's zero-delta claim holds only for single-session exports. |
| 5 | `SC.status.md:36` (item 12 rule c) vs `:42-45` (item 14) | MED | `ValidateProfile` errors on `Status` outside `success|failure|aborted`; `""` is outside, and every pre-v1.1 row has `Status: ""` while item 14 declares the Status-less encoding legitimate. `CompareProfiles` calls the validator → every legacy comparison floods `Notes`. |
| 6 | `S8a.status.md:24` | LOW | Rule 3 heading wider than mechanism (callers of `appendSpoolLine`). |
| 7 | `S8a.status.md:28` | LOW | Rule 5(b) names no test function. |
| 8–10 | `SC.status.md:34,36,44`, `roadmap.md:351`, `S7:26`, `S5:49` | LOW | `compare.go` citation ranges drifted by 1–2 lines (`Notes` field is `:40`; comparators `:90-94`; `MetricComparison` `:23-30`). |
| 11 | `SC.status.md:48` | LOW | Item 17 omitempty/declared-last scoped to items 1–9, omitting item 11's `BaselineSource`/`CandidateSource`. |
| 12 | `SC.status.md:59` | LOW | `compareContextWindow` nil case unstated (`ContextWindow` is a pointer, nil on hooks-only and legacy profiles). |
| 13 | `S1a.status.md:27` | LOW | Latency ceiling "2× measured p95" names no machine; flaky on `ubuntu-latest`. |
| 14 | `roadmap.md:46` vs `SC.status.md:67` | LOW | R-CS-05 and R-SA-16 register↔slice asymmetries. |

**Shared root (1–4):** D12's single-writer invariant was specified at repo width (SC item 8 "the only function in the repo"; rule 5 tree-wide grep, package-wide AST walk) while its justification and the epic's file scope are Cursor-only — and rule 5's mechanism was never run against `main`. The same widening left the assignment site homeless (3) and stopped at `tokens` when the identical unscopable-aggregate problem exists on `tool_calls` (4). Invariant: the guardrail's width, the contract's admissible-source set, and the set of files the epic owns must be the same set, and every grep/AST rule must be run against `main` before its slice claims red→green.

## DEFERRED / recommendation
- **Split SC** (per §H2): SC-a = `types.go` + `tokens_select.go` + `validate.go` + spec doc (unblocks S0, S1b, S3, S8a, S9, S10); SC-b = `compare.go` + comparison/experiment goldens (needed only by S5, S7). No edge inverts; both MVEs still close. Verified.
- §G 1–16, §H 1, 3–7 honestly scoped; §H hides no REAL finding but is incomplete (findings 1–5 absent).

## SUSPECTED
- `roadmap.md:350` (H3, LOW): with `GOTOOLCHAIN=auto`, Go 1.23 downloads and runs 1.27.1, so the "latent CI break" likely never fires. Confirm from a CI run before acting.

## V4 fix table
1 fixed at type level, mechanism defective (→ V5-1/2/3) · 2, 3, 4, 6, 7, 8, 10–16 fixed at root · 5 fixed, one over-wide rule (→ V5-5) · 9 fixed on paper, ships red (→ V5-2). 16/16 addressed.

## Graph / matrix
35 edges, identical across slice files / §C / Mermaid / §F; acyclic. Primary MVE closes; SP1=no fallback closes. Same-wave collision: wave 2 `cursor.go` S0‖S1b only. Matrix 65/65, 0 orphans, 0 phantoms. 17/17 name a test file. S8a rules 1, 2, 4 green (or green after their named slice); rule 3 heading wide; rules 5(a)/5(b) ship RED.
