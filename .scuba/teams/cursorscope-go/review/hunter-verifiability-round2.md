# Hunter report — slice verifiability lens (spec gate, round 2 confirming pass)

**Artifact:** revised `roadmap.md` + 16 slice files · **Date:** 2026-09-10 · **Verdict:** NOT-CLEAN (9 REAL + 1 SUSPECTED above LOW)

Baseline `go vet ./...` + `go test ./...` green. Two findings reproduced in an isolated post-SC copy (scratchpad `sc/`).

**Coverage:** 16/16 slice files × 6 checks; 65/65 requirement IDs both directions; 28/28 graph edges vs 16 table rows vs 16 slice `Depends on`; 6 wave rows × 9 shared files; 18/18 round-1 findings re-verified against `profiler/types.go`, `compare.go`, `experiment.go`, `cursor.go`, `docs/profiler-spec.md`, `AGENTS.md`; second sweep run.

## Root A — R1 not closed at the root (SC scoped to the 4 channels round 1 named; downstream DoDs and consumers of SC's new fields never swept)

1. **HIGH REAL, reproduced.** `profiler/compare.go:143-155` — `countToolCalls` is `len(calls)`. SC (`roadmap.md:136` DoD 5) fixes source/class only, never cardinality, so S0's `Count`-bearing aggregate rows (`S0.status.md:17`) vs S1b's per-call rows for the same 42 calls → `comparable:true, count delta 40, success_rate delta +0.45`. Same class as V-5, introduced by SC's own `Count`. `S7.status.md:15` DoD 3 asserts `comparable:true` — satisfied while wrong. *Invariant: two profiles of one session must yield zero delta regardless of row encoding.*
2. **HIGH REAL.** `S1b.status.md:14` DoD 2 ("correlate `conversation_id`→`generation_id`→`tool_use_id`, subagents branch") is not representable: `ToolCallEntry` (`types.go:88-92`) has no ID/parent, `Profile` has no hierarchy, SC adds none. `Attribution.Target` (`types.go:114-118`, "tool call ID") dangles. `S3.status.md:17` DoD 3 inherits it.
3. **MEDIUM REAL, reproduced.** `SC.status.md:23` DoD 1 (pre-v1.1 golden re-marshals byte-identically) contradicts DoD 1/2/4 (`:13,14,16`), which specify `omitempty` only for `Count`. `Source`/`Status`/`Notes`/`Available` emit `""`/`null` and break byte-identity.
4. **MEDIUM REAL.** `S5.status.md:21` / R-CS-13 (`roadmap.md:52`, keep) / R-RL-18 require `context_window_size` + `context_usage_percent`; `TokenCounts` (`types.go:80-86`) has no field and SC adds none. S5 doesn't own `types.go`.
5. **LOW SUSPECTED.** Honesty class of an unknown/legacy `Source` string is unspecified (`roadmap.md:120` D3 enumerates Go constants; wire field is free `string`; D1 keeps `.../v1`). Fail-open reinstates V-5.

## Root B — dependency soundness not re-swept after the V-6 call

6. **MEDIUM REAL.** `S8.status.md:6` omits **S0**. Guardrail 4 (`:19`) allows `cursor.` only in `cursor_wirekeys.go`, which S0 creates; `profiler/cursor.go:175,182,184,186,197,224,242,262` holds 8 live `cursor.` literals → CI red until S0 merges.
7. **MEDIUM REAL.** `roadmap.md:159` fallback MVE (SP1=no) = SC+S0+S1a+S1b+S2+S4+S8 does **not close**: S8 depends on S3 (`roadmap.md:147`, `S8.status.md:6`, Mermaid `:210`). S3 is the only thing S8's four rules don't need.
8. **MEDIUM REAL.** Fork (vi) flip (`roadmap.md:238`, `S5.status.md:36`) has S5's PR delete SC's `estimated` constant — the constant `S8.status.md:31` says guardrail 2 needs. Q3=option(1) breaks an MVE slice.
9. **LOW REAL.** `S11.status.md:15` DoD requires SP1's fixtures; `:6` declares only S1a, S2.

## Root C — slice-file/§F completeness not swept

10. **MEDIUM REAL.** `roadmap.md:276` claims hooks.go writers "land in different waves except S1b‖S4" — false: wave 3 (`:271`) runs S3 ‖ S5, both listing `profiler/hooks.go`. §F omits `docs/profiler-spec.md` (SC w1, S0 w2, S5 w3), `profiler/cmd/main.go` (S1b, S7) and S11's `hooks.go` from its ownership map.
11. **MEDIUM REAL.** `S9`, `S10`, `S11`, `T1` have no `## Files` section; `T1` also has no `## Requirements`, so R-RL-09/10/11 and R-SA-10's T1 half are unclaimed slice-side.
12. **MEDIUM SUSPECTED.** `S8.status.md:22` rule 2's enforcement ("a Go test over the package's declarations, using the same source scan") names no analyzer; the only rule not reducible to grep.
13. **LOW REAL.** `S11.status.md:18` Test approach names no test file, violating `roadmap.md:132`.

## LOW

14. R-CS-30's "bounded memory" half asserted by no S1a DoD item. 15. `S11.status.md:15` builds a TTL sweep while `:22` disclaims R-CS-06. 16. `compare.go:157-162` `successRate` conflates `error` and `aborted` once `Status` is tri-state. 17. `SP1.status.md:25` commits `.scuba/…/spike-r-rl-17.md` onto `epic/skill-scope`. 18. `revision-log-round1.md:71` says "26 edges"; actual 28. 19. `S10.status.md:24` claims R-SA-14, absent from `roadmap.md:88`'s Slice column.

## Round-1 fix status

V-1..4 fixed · **V-5 partial** (→#1) · **V-6 partial** (→#6,#7) · **V-7 partial** (→#2,#14) · V-8 fixed · **V-9 partial** (→#10) · **V-10 partial** (→#9) · V-11 fixed · **V-12 partial** (→#11,#13) · V-13..18 fixed. No finding regressed outright; five fixed for the named instance only.

## Partial-disagreement adjudication

- **V-6 (S8→SC not S5): groomer CORRECT** — but the same test was not applied to S8's other edges (#6, #7), and #8 shows the SC-only dep is fragile under the Q3 flip.
- **V-7 (two slices not three): groomer CORRECT** — residue is #2, not slice count.

## Matrix and graph

65/65 IDs walked: 0 orphans, 0 phantoms; residual defect is T1's missing `## Requirements` (#11). Graph acyclic; table ⟷ slice `Depends on` ⟷ 28 Mermaid edges agree; primary MVE {SC,S0,S1a,S1b,S2,S3,S4,S8} closes; SP1=no fallback does not (#7); SP1 hard-gates S3 in all three places.
