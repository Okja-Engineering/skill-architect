# E-CS — Cursor hook telemetry for the profiler (skill-scope)

## Status
- **Stage:** spec — **presented to user 2026-09-10 ~03:55; awaiting the user's answers to §E Q1–Q8**. Five spec-gate rounds run; the round-5 revision is itself unreviewed.
- **Owner:** chief of staff (manager hat)
- **Branch:** none yet (target integration branch: `epic/skill-scope`)
- **Worktree:** none yet
- **Last SHA:** `0a83615` (main)
- **Blocker:** none in-thread; user fork calls and Cursor availability remain open in the roadmap's §E

## Artifacts
- `.scuba/teams/cursorscope-go/mandate-draft.md` — mandate draft (forks in §7, open questions in §10)
- `.scuba/teams/cursorscope-go/roadmap.md` — groomed epic roadmap, **revised round 5** (381 lines): thesis, requirements register, **§B.4 Design decisions D1–D13**, slices, forks (§D), user-only questions (§E), integration plan with a rebuilt ownership map, §G (16 unverified items), **§H Known residue (H1–H8; H2 closed, H3 downgraded)**
- `.scuba/teams/cursorscope-go/slices/` — **18** slice status files: **SC-a / SC-b** (the contract, split at round 5; SC-a ships first), **SP1** (gating spike, four questions), S0, **S1a/S1b**, S2–S7, **S8a** (guardrails) / **S8b** (docs), S9–S11, plus T1 (separate parallel thread). `SC.status.md` is retained as a one-line pointer to the two halves. `S8.status.md` deleted in round 2; `S1.status.md` deleted in round 1.
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
- `.scuba/teams/cursorscope-go/slices/SC-a.status.md` — contract part A: channels, selector (`ApplyTokenSelection`), validator, caveat carrier — **ships first**
- `.scuba/teams/cursorscope-go/slices/SC-b.status.md` — contract part B: comparators, comparability gate, wire-format goldens — gated on SC-a
- `review/revision-log-round4.md` — **25 dispositions plus three class-sweep tables**: (1) every slice that produces token data → its label, its scope flag and the fact that it assigns nothing, (2) every SC rule → the mechanism that now enforces it, (3) every slice adding a bash test → how CI runs it; plus a re-derived citation register and the residue list
- `review/hunter-conformance-round5.md` — round 5: NOT-CLEAN, **3 MED (one shared root) + 8 LOW**; guardrail rules 1–5 executed against today's tree, 5(a)/5(b) found shipping red
- `review/hunter-verifiability-round5.md` — round 5: NOT-CLEAN, **2 HIGH + 3 MED + 9 LOW + 1 SUSPECTED** — the HIGHs are S8a rule 5(a)/(b) scoped repo-wide over files no slice owns
- `review/revision-log-round5.md` — round-5 dispositions: rule 5 scoped to a named 8-file Cursor path set with executed baselines, `ApplyTokenSelection` owns the assignment site, `tool_calls` session eligibility, `ValidateProfile` accepts legacy `Status: ""`, **SC split into SC-a/SC-b** (edges 35 → 39)

## Round 4 revision summary (final planned revision)
Round 3 proved every "present from hooks" assertion had a **supplier**; round 4 asked whether **any code enforces the rule the DoD states**, and whether the fix that closed one instance closed its **class**. Three classes:
- **Token single-writer (V4-#1, HIGH).** D12's invariant had **five** producers — S0, S1b, S5, S6, S9 — and no mechanism, and its selector sat in S5 while **S0 ships two waves earlier, inside the MVE, gated on nothing**. `SelectTokenSource` + `TokenCandidate` **moved to SC** (`profiler/tokens_select.go`) as the only assigner of `Profile.Tokens`; the rank became **total over five labels — `server_api` > `otel` > `otel_aggregate` > `mcp_reported` > `estimated`** (class order, then scope precision); an unscoped candidate is **ineligible** under `--session`; and **S8a guardrail 5** (grep + `go/ast`) enforces both the single writer and "`hooks` is never a token-flow label". The S6 → S5 edge dissolved with the move and was **not** invented back.
- **SC enforcement.** Five rules had no code: a `context_window` comparator, **`ValidateProfile`** (called from `CompareProfiles`), a closed-set **`RawMetricResult.Caveat`** carrying D11's timing caveat *in the profile*, a **comparability gate** replacing the false "zero delta regardless of row encoding" claim, and the additive rule extended from **two** wire strings to **four** (the experiment schemas, which S7 changes).
- **CI and ownership.** `ci.yml` enumerated four scripts by name, so four slices' bash tests would never have run. **S1a DoD 7 switches CI to a `tests/test_*.sh` glob once and everyone inherits**; `ci.yml` becomes single-writer; S8a drops it. S8a rule 1 is scoped to `skills/*/scripts/` (the width `AGENTS.md:6` states) so it **ships green** — `tests/test_skill.sh:23` and `test_walk.sh:25` already invoke `python3`, and that cleanup is unowned residue.

Edges **33 → 35** (SC → S9, SC → S10). Slice count unchanged at **17**. Roadmap **335 → 354** lines, +8 of which is the new **§H Known residue** (H1–H7, each with its finding id).

### Round 5 (`review/revision-log-round5.md`)

Tightly scoped: 2 HIGH, 3 MED, 9 LOW, 1 suspected, plus the deferred SC split. **One invariant closes the HIGHs — the guardrail's width, the contract's admissible-source set and the files the epic owns are one set, and every grep/AST rule is executed against `main` before its slice claims red→green.** S8a rule 5 was repo-wide and shipped red both ways (24 `.Tokens =` hits, 18 in adapters no slice touches; one legitimate `session_data` label); it is now scoped to a named **Cursor-path file set**, with every rule's executed baseline stated. The homeless assignment site got an owner — **`ApplyTokenSelection`** in `tokens_select.go`, with an accumulator (`TokenCandidates`) and two named call sites (S0's OTel branch, S1b's `enrich()`), collapsed to one by S7. Session eligibility extended from `tokens` to **`tool_calls`**, with a single-conversation escape and a third caveat constant. `ValidateProfile` stopped rejecting the legacy `Status: ""`. **SC split into SC-a + SC-b** (§H2 closed): edges **35 → 39**, depth still five waves, both MVEs close (primary **10** PRs, fallback **9** — SC-b is in both because S0 is). New residue **H8** (the invariant is Cursor-wide, 18 assignments remain in three legacy adapters); **H3 downgraded** (`GOTOOLCHAIN=auto` likely makes it inert — confirm from a CI run).

## Round 3 revision summary
Both hunters named the same remaining root: round 2 proved every DoD assertion had a contract **field**, never that a hook event **supplies the value**. Round 3 answers with a source-supply table over all 29 hook-sourced assertions — **4 removed or downgraded, 5 derived under a named decision, 1 requirement half marked not-consumed.**
- **D11 — what hooks do not supply.** (1) **Timestamps:** none exist anywhere in the surface; S1a's binary stamps `received_at` into the spool envelope and `TimingData` is `Source: hooks` **with a stated caveat** (hook-receipt wall-clock, not model time), cross-checked in test against `sessionEnd.duration_ms`. (2) **Outcome:** `afterShellExecution` has no success signal; `Status` derives from `postToolUse` / `postToolUseFailure` (`is_interrupt` → `aborted`) and is **always set**. (3) **Identifiers:** `ID`/`ParentID` hold tool-call ids only — `subagentStart` spells it `tool_call_id`, and `parent_conversation_id` is barred. (4) **Attribution join:** `beforeReadFile` has no `tool_use_id`; two forms specified, **SP1 decides which**.
- **D12 — one result, one source, one writer.** `preCompact`'s occupancy fields leave `TokenCounts` for a new **`context_window`** metric; `tokens` carries the single best-honesty flow source over the total order `server_api` > `otel` > `otel_aggregate` > `mcp_reported` > `estimated`, lower candidates discarded not merged. **The contract's diff to `TokenCounts` is zero**; the admissible-source set is an **allowlist** of those five, and the one assigner is **`ApplyTokenSelection`** (SC-a).
- **D13 — the comparison report** (`skill-architect/comparison/v1`) versions independently and stays `v1` under D1's additive rule, proven by a pinned golden.
- **`successes` defined exactly once** (SC-b) and `Status` always set (S1b) — the 0.75-vs-0.50 divergence is fixed at the root, and both fixtures are now required to be **heterogeneous**.
- **SP1 widened 1 → 4 questions** so the spike that is already happening validates all three blind decisions in D11. Edges 32 → 33 (**S4 → S11**). Slice count unchanged at 17.

**MVE:** SC-a + SC-b + S0 + S1a + S1b + S2 + S3 + S4 + S8a + S8b (**10** PRs) + the SP1 gate. **Fallback (SP1 = no):** the same minus S3 — **9** PRs, and it closes. **Ships first: SC-a**, with S1a and SP1 in parallel; **SC-b second**, gated on nothing but SC-a; T1 as its own thread today.

## Note — sibling repo
`/Users/matthewvandusen/Development/Auraprix/cursor-profiler` holds `intent.md` and `spec.md` mirroring this mandate; research findings live under `docs/research/` (add-only), including the deep-research report `existing-per-skill-cost-tools.md` that D10 is built on. **Contention:** that directory's `research-contradictions.md` (A4) calls the `cursor.*` telemetry names unverified, while round-1 conformance verified all of them against Cursor's Wire Reference (the *keys* were wrong; the names are real) — surfaced as a decision on the roadmap.

## Upstream defect found in round 3 (another team's artifact — not fixed here)
`.scuba/teams/research-profiling/ledger.md:111` claims cursorscope's registered hook list "matches exactly" Cursor's — it registers **19**, missing `beforeTabFileRead` and `workspaceOpen`. R-CS-01 is already correct; the ledger line is not. Recorded as unowned note (c) in the roadmap's §F.

## Next
**On the user's answers:** (a) dispatch a bug-fixer for **S0** if approved; (b) dispatch **T1** as its own thread if approved; (c) **one confirming hunter pass over the round-5 revision before any code dispatch** — round 5's own repairs have not been reviewed.

**Presented to user 2026-09-10 ~03:55.** Round 5 was the tightly scoped revision after the fifth verifiability gate; the forks (§D) and user questions (§E) go to the user next. Every round-5 finding was fixed at the root and the tables are in `review/revision-log-round5.md`. What a further hunter should look at first: the **executed-baseline table** in `S8a.status.md` (the step round 4 skipped, and the one that caught both HIGHs), the two new call sites for `ApplyTokenSelection` and S7's collapse of them, the `tool_calls` eligibility rule and its single-conversation escape, and **§H** — H8 is new, H2 is closed, H3 is downgraded pending a CI log.
