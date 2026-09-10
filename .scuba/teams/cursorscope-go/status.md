# E-CS — Cursor hook telemetry for the profiler (skill-scope)

## Status
- **Stage:** spec (**round-2 revision running** — both confirming hunters returned NOT-CLEAN but converging)
- **Owner:** chief of staff (manager hat)
- **Branch:** none yet (target integration branch: `epic/skill-scope`)
- **Worktree:** none yet
- **Last SHA:** `0a83615` (main)
- **Blocker:** none in-thread; user fork calls and Cursor availability remain open in the roadmap's §E

## Artifacts
- `.scuba/teams/cursorscope-go/mandate-draft.md` — mandate draft (forks in §7, open questions in §10)
- `.scuba/teams/cursorscope-go/roadmap.md` — groomed epic roadmap, **revised round 1** (298 lines): thesis, requirements register, **§B.4 Design decisions D1–D9**, slices, forks (§D), user-only questions (§E), integration plan
- `.scuba/teams/cursorscope-go/slices/` — **16** slice status files: **SC** (contract, ships first), **SP1** (gating spike), S0, **S1a/S1b** (S1 split), S2–S11, plus T1 (recommended as a separate parallel thread). `S1.status.md` deleted.
- `.scuba/teams/cursorscope-go/review/hunter-conformance.md` — round 1, conformance lens: **NOT-CLEAN**, 15 REAL (+4 low)
- `.scuba/teams/cursorscope-go/review/hunter-verifiability.md` — round 1, verifiability lens: **NOT-CLEAN**, 18 REAL (shared root R1)
- `.scuba/teams/cursorscope-go/review/revision-log-round1.md` — **37 dispositions**, one line each; five roots repaired; nothing rejected outright, two partial disagreements and one evidence caveat recorded in place
- `.scuba/teams/cursorscope-go/review/hunter-conformance-round2.md` — round 2, conformance lens: **NOT-CLEAN**, 12 REAL (1 HIGH, 3 MED, 8 LOW/LOW-MED), 1 INVALID; the HIGH is an **invented enum value** (`error` in a pass-through tri-state whose source vocabulary is `success | failure | aborted`)
- `.scuba/teams/cursorscope-go/review/hunter-verifiability-round2.md` — round 2, verifiability lens: **NOT-CLEAN**, 9 REAL + 1 SUSPECTED; root A = **the contract slice was not swept downstream** (SC scoped to the four channels round 1 named; downstream DoDs and consumers of SC's new fields never swept)

## Round 1 revision summary
New **SC** slice owns `types.go` + `compare.go` + `profiler-spec.md` (root R1) · thesis rewritten from the adapter's real states (root A) · eight evidence labels corrected against the ledger (root B) · S0's DoD rewritten to the wire shape and its pinned tests marked **replace, not extend** (root C) · nine Design decisions close the merged-register conflicts (root D) · spike moved ahead of S3 as a hard gate · S1 split · every slice file has a Test approach and its own `_test.go`.

**MVE:** SC + S0 + S1a + S1b + S2 + S3 + S4 + S8 (8 PRs) + the SP1 gate. **Ships first: SC**, with S1a and SP1 in parallel.

## Note — sibling repo
`/Users/matthewvandusen/Development/Auraprix/cursor-profiler` now holds `intent.md` and `spec.md` mirroring this mandate; research findings are collected under its `docs/research/` (add-only): six research files plus the deep-research report `existing-per-skill-cost-tools.md`. **Contention:** that directory's `research-contradictions.md` (A4) calls the `cursor.*` telemetry names unverified, while round-1 conformance verified all 21 against Cursor's Wire Reference (the *keys* are wrong; the names are real) — surfaced as a decision on the roadmap.

## Next
Round-3 confirming pass after revision 2; then present forks + questions to the user.
