# E-CS — Cursor hook telemetry for the profiler (skill-scope)

## Status
- **Stage:** spec (**round-2 revision complete** — round-3 confirming hunters running)
- **Owner:** chief of staff (manager hat)
- **Branch:** none yet (target integration branch: `epic/skill-scope`)
- **Worktree:** none yet
- **Last SHA:** `0a83615` (main)
- **Blocker:** none in-thread; user fork calls and Cursor availability remain open in the roadmap's §E

## Artifacts
- `.scuba/teams/cursorscope-go/mandate-draft.md` — mandate draft (forks in §7, open questions in §10)
- `.scuba/teams/cursorscope-go/roadmap.md` — groomed epic roadmap, **revised round 2** (325 lines): thesis, requirements register, **§B.4 Design decisions D1–D10**, slices, forks (§D), user-only questions (§E), integration plan with a rebuilt ownership map, §G (16 unverified items)
- `.scuba/teams/cursorscope-go/slices/` — **17** slice status files: **SC** (contract, ships first), **SP1** (gating spike), S0, **S1a/S1b**, S2–S7, **S8a** (guardrails) / **S8b** (docs), S9–S11, plus T1 (separate parallel thread). `S8.status.md` deleted in round 2; `S1.status.md` deleted in round 1.
- `review/hunter-conformance.md` · `review/hunter-verifiability.md` — round 1: NOT-CLEAN, 33 REAL + 4 low
- `review/revision-log-round1.md` — 37 dispositions, five roots repaired
- `review/hunter-conformance-round2.md` — round 2: NOT-CLEAN, 12 REAL; root = labels/enums transcribed by topic, not re-derived from source
- `review/hunter-verifiability-round2.md` — round 2: NOT-CLEAN, 9 REAL + 1 SUSPECTED + 6 LOW; root = SC never swept downstream
- `review/revision-log-round2.md` — **31 dispositions plus three proof tables**: (1) every DoD assertion in all 17 slices → its carrier field, (2) every enum literal and evidence label → its source line, (3) every `Depends on` recomputed from DoD + Files

## Round 2 revision summary
Fixed by **class**, not by instance, because both hunters named "round 1 fixed the named instances" as the meta-defect.
- **SC widened** 4 channels → 8: tool-call identity/hierarchy (`ID`/`GenerationID`/`ParentID`), the three `preCompact` context fields on `TokenCounts`, `SourceNone` → honesty class `none`, fail-closed `unrecognized` for any unknown wire `Source`, **encoding-invariant** tool-call counting, tri-state-aware `successRate`. Every added field is `omitempty` and declared last, which makes the byte-identity DoD derivable instead of self-contradictory.
- **S8 split into S8a (guardrails, deps {SC, S0, S4}) + S8b (docs)** — the *SP1 = no* fallback MVE now closes.
- **S1b lays an enrichment seam** so S3 and S5 own their own files; the wave-3 `hooks.go` collision is removed, not documented. One same-wave collision remains in the whole epic: S0 ‖ S1b on `cursor.go`.
- **D10** records the mandatory reuse-vs-build check: **build**, with `dash0hq/dash0-agent-plugin` as a reference implementation (9 of 21 hooks, per-turn spans, no skill span, no subagent span), `o11y-dev/opentelemetry-hooks` excluded as Python.
- One invented enum value corrected (`cursor.tool.status` is `success|failure|aborted`); base fields corrected to 10 with `workspaceOpen`'s four-field exception; six evidence labels re-derived and cited.

**MVE:** SC + S0 + S1a + S1b + S2 + S3 + S4 + S8a + S8b (9 PRs) + the SP1 gate. **Fallback (SP1 = no):** the same minus S3 — 8 PRs, and it closes. **Ships first: SC**, with S1a and SP1 in parallel; T1 as its own thread today.

## Note — sibling repo
`/Users/matthewvandusen/Development/Auraprix/cursor-profiler` holds `intent.md` and `spec.md` mirroring this mandate; research findings live under `docs/research/` (add-only), including the deep-research report `existing-per-skill-cost-tools.md` that D10 is built on. **Contention:** that directory's `research-contradictions.md` (A4) calls the `cursor.*` telemetry names unverified, while round-1 conformance verified all of them against Cursor's Wire Reference (the *keys* were wrong; the names are real) — surfaced as a decision on the roadmap.

## Next
If round 3 is **CLEAN** → present forks (§D) and User Questions 1–8 to the user. Else → round-3 revision.
