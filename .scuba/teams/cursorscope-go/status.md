# E-CS — Cursor hook telemetry for the profiler (skill-scope)

## Status
- **Stage:** spec (**round-4 revision complete**; **round-5 FINAL confirming hunters running** — present to user after round 5 regardless of verdict)
- **Owner:** chief of staff (manager hat)
- **Branch:** none yet (target integration branch: `epic/skill-scope`)
- **Worktree:** none yet
- **Last SHA:** `0a83615` (main)
- **Blocker:** none in-thread; user fork calls and Cursor availability remain open in the roadmap's §E

## Artifacts
- `.scuba/teams/cursorscope-go/mandate-draft.md` — mandate draft (forks in §7, open questions in §10)
- `.scuba/teams/cursorscope-go/roadmap.md` — groomed epic roadmap, **revised round 4** (354 lines): thesis, requirements register, **§B.4 Design decisions D1–D13**, slices, forks (§D), user-only questions (§E), integration plan with a rebuilt ownership map, §G (16 unverified items), **§H Known residue (H1–H7)**
- `.scuba/teams/cursorscope-go/slices/` — **17** slice status files: **SC** (contract, ships first), **SP1** (gating spike, now four questions), S0, **S1a/S1b**, S2–S7, **S8a** (guardrails) / **S8b** (docs), S9–S11, plus T1 (separate parallel thread). `S8.status.md` deleted in round 2; `S1.status.md` deleted in round 1.
- `review/hunter-conformance.md` · `review/hunter-verifiability.md` — round 1: NOT-CLEAN, 33 REAL + 4 low
- `review/revision-log-round1.md` — 37 dispositions, five roots repaired
- `review/hunter-conformance-round2.md` — round 2: NOT-CLEAN, 12 REAL; root = labels/enums transcribed by topic, not re-derived from source
- `review/hunter-verifiability-round2.md` — round 2: NOT-CLEAN, 9 REAL + 1 SUSPECTED + 6 LOW; root = SC never swept downstream
- `review/revision-log-round2.md` — **31 dispositions plus three proof tables**: (1) every DoD assertion in all 17 slices → its carrier field, (2) every enum literal and evidence label → its source line, (3) every `Depends on` recomputed from DoD + Files. *(Line 96's `§A4:300` corrected in place to `Q1:300` in round 3 — C3-#9.)*
- `review/hunter-conformance-round3.md` — round 3: confirmed all 12 round-2 fixes landed **at root**; 12 new findings + 1 SUSPECTED, mostly LOW
- `review/hunter-verifiability-round3.md` — round 3: NOT-CLEAN, **5 HIGH** — hooks carry no timestamp, no success signal on `afterShellExecution`, no `tool_use_id` on `beforeReadFile`, one `Source` per `TokenResult`
- `review/revision-log-round3.md` — **29 dispositions plus the source-supply table**: for every value asserted `present` from hooks, the `event.field` that supplies it (cited to a ledger line) or **NONE** with the consequence applied; plus the token-source decision and a re-derived citation register
- `review/hunter-conformance-round4.md` — round 4: NOT-CLEAN, **3 MED + 6 LOW, no HIGH**; all C3-1…12 + SUSPECTED confirmed fixed at root
- `review/hunter-verifiability-round4.md` — round 4: NOT-CLEAN, **1 HIGH + 8 MED** — the HIGH is S0 bypassing the single token writer (`selectTokenSource`), the fourth writer D12's sweep missed
- `review/revision-log-round4.md` — **25 dispositions plus three class-sweep tables**: (1) every slice that produces token data → its label, its scope flag and the fact that it assigns nothing, (2) every SC rule → the mechanism that now enforces it, (3) every slice adding a bash test → how CI runs it; plus a re-derived citation register and the residue list

## Round 4 revision summary (final planned revision)
Round 3 proved every "present from hooks" assertion had a **supplier**; round 4 asked whether **any code enforces the rule the DoD states**, and whether the fix that closed one instance closed its **class**. Three classes:
- **Token single-writer (V4-#1, HIGH).** D12's invariant had **five** producers — S0, S1b, S5, S6, S9 — and no mechanism, and its selector sat in S5 while **S0 ships two waves earlier, inside the MVE, gated on nothing**. `SelectTokenSource` + `TokenCandidate` **moved to SC** (`profiler/tokens_select.go`) as the only assigner of `Profile.Tokens`; the rank became **total over five labels — `server_api` > `otel` > `otel_aggregate` > `mcp_reported` > `estimated`** (class order, then scope precision); an unscoped candidate is **ineligible** under `--session`; and **S8a guardrail 5** (grep + `go/ast`) enforces both the single writer and "`hooks` is never a token-flow label". The S6 → S5 edge dissolved with the move and was **not** invented back.
- **SC enforcement.** Five rules had no code: a `context_window` comparator, **`ValidateProfile`** (called from `CompareProfiles`), a closed-set **`RawMetricResult.Caveat`** carrying D11's timing caveat *in the profile*, a **comparability gate** replacing the false "zero delta regardless of row encoding" claim, and the additive rule extended from **two** wire strings to **four** (the experiment schemas, which S7 changes).
- **CI and ownership.** `ci.yml` enumerated four scripts by name, so four slices' bash tests would never have run. **S1a DoD 7 switches CI to a `tests/test_*.sh` glob once and everyone inherits**; `ci.yml` becomes single-writer; S8a drops it. S8a rule 1 is scoped to `skills/*/scripts/` (the width `AGENTS.md:6` states) so it **ships green** — `tests/test_skill.sh:23` and `test_walk.sh:25` already invoke `python3`, and that cleanup is unowned residue.

Edges **33 → 35** (SC → S9, SC → S10). Slice count unchanged at **17**. Roadmap **335 → 354** lines, +8 of which is the new **§H Known residue** (H1–H7, each with its finding id).

## Round 3 revision summary
Both hunters named the same remaining root: round 2 proved every DoD assertion had a contract **field**, never that a hook event **supplies the value**. Round 3 answers with a source-supply table over all 29 hook-sourced assertions — **4 removed or downgraded, 5 derived under a named decision, 1 requirement half marked not-consumed.**
- **D11 — what hooks do not supply.** (1) **Timestamps:** none exist anywhere in the surface; S1a's binary stamps `received_at` into the spool envelope and `TimingData` is `Source: hooks` **with a stated caveat** (hook-receipt wall-clock, not model time), cross-checked in test against `sessionEnd.duration_ms`. (2) **Outcome:** `afterShellExecution` has no success signal; `Status` derives from `postToolUse` / `postToolUseFailure` (`is_interrupt` → `aborted`) and is **always set**. (3) **Identifiers:** `ID`/`ParentID` hold tool-call ids only — `subagentStart` spells it `tool_call_id`, and `parent_conversation_id` is barred. (4) **Attribution join:** `beforeReadFile` has no `tool_use_id`; two forms specified, **SP1 decides which**.
- **D12 — one result, one source.** `preCompact`'s occupancy fields leave `TokenCounts` for a new **`context_window`** metric; `tokens` carries the single best-honesty flow source via `selectTokenSource` (`server_api` > `otel` > `mcp_reported` > `estimated`), lower candidates discarded not merged. **SC's diff to `TokenCounts` is now zero**, and `hooks` is never a token-flow label.
- **D13 — the comparison report** (`skill-architect/comparison/v1`) versions independently and stays `v1` under D1's additive rule, proven by a pinned golden.
- **`successes` defined exactly once** (SC) and `Status` always set (S1b) — the 0.75-vs-0.50 divergence is fixed at the root, and both fixtures are now required to be **heterogeneous**.
- **SP1 widened 1 → 4 questions** so the spike that is already happening validates all three blind decisions in D11. Edges 32 → 33 (**S4 → S11**). Slice count unchanged at 17.

**MVE:** SC + S0 + S1a + S1b + S2 + S3 + S4 + S8a + S8b (9 PRs) + the SP1 gate. **Fallback (SP1 = no):** the same minus S3 — 8 PRs, and it closes. **Ships first: SC**, with S1a and SP1 in parallel; T1 as its own thread today.

## Note — sibling repo
`/Users/matthewvandusen/Development/Auraprix/cursor-profiler` holds `intent.md` and `spec.md` mirroring this mandate; research findings live under `docs/research/` (add-only), including the deep-research report `existing-per-skill-cost-tools.md` that D10 is built on. **Contention:** that directory's `research-contradictions.md` (A4) calls the `cursor.*` telemetry names unverified, while round-1 conformance verified all of them against Cursor's Wire Reference (the *keys* were wrong; the names are real) — surfaced as a decision on the roadmap.

## Upstream defect found in round 3 (another team's artifact — not fixed here)
`.scuba/teams/research-profiling/ledger.md:111` claims cursorscope's registered hook list "matches exactly" Cursor's — it registers **19**, missing `beforeTabFileRead` and `workspaceOpen`. R-CS-01 is already correct; the ledger line is not. Recorded as unowned note (c) in the roadmap's §F.

## Next
**Present to user after round 5.** The round-5 confirming hunters (conformance + verifiability) are running now; whatever they return, the forks (§D) and user questions (§E) go to the user next — round 4 was the last planned revision. Every finding was fixed at the class level and the sweep tables are in `review/revision-log-round4.md`. What a round-5 hunter should look at first: the three moved/new mechanisms (`SelectTokenSource` in SC, `ValidateProfile`, S8a guardrail 5), the CI glob, and §H — the residue list is where round 4 deliberately stopped.
