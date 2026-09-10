# Hunter report — telemetry-name and factual conformance lens (spec gate, round 5 final confirming pass)

**Artifact:** `roadmap.md` (354 ln) + 17 slice files · **Date:** 2026-09-10 · **Verdict:** NOT-CLEAN — 3 MED (one shared root), 8 LOW, 1 INVALID (self-raised, cleared)

**Coverage:** 18/18 files walked twice · 25/25 round-4 dispositions traced · 13/13 revision-log §5 citation rows re-derived · 7/7 §H items · guardrail rules 1–5 all executed against the current tree · 35/35 Mermaid edges reconciled both directions, acyclic · 39 repo citations + 41 ledger lines compared verbatim · live re-fetch of Cursor Admin API docs · `go vet` clean, 34 Go green, 131 bash green.

## Guardrail status — run as written, today's tree

| # | Result today | Expected? |
|---|---|---|
| 1 | GREEN (exit 1; 6 scripts clean) | ✅ C4-#1 fixed at root |
| 2 | GREEN (no `(?i)estimate` func) | ✅ |
| 3 | GREEN (exit 1) | ✅ green by construction after S1a/S4/S11 |
| 4 | RED — 12 at `cursor.go:175,182,184,186,197,216,224,234,242,250,262,271` | ✅ declared red until S0 (`S8a:25`, `S0:7`) |
| 5(a) | RED — 24; DoD says zero | ❌ not declared; 18 permanent — REAL #1 |
| 5(b) | RED — `claude_code.go:465` | ❌ not declared; permanent — REAL #2 |

## C4 fix table
C4-1…9 all FIXED at root (verified by running/reading); C4-2 stray count → LOW #5; C4-9 row's own count wrong → LOW #4. C4-5 confirmed: live docs show both "60 requests per minute per team" and "Date range cannot exceed 30 days" under `/teams/filtered-usage-events`.

## REAL — MED (one shared root)
**Root.** D12's invariant is Cursor-scoped (ledger Q1:300 is about Cursor's hook surface). S8a rule 5 restates it tree-wide and enforces it over three shipped adapters no slice lists in `## Files`. Same class as C4-#1, which round 4 swept for rule 1 only.

1. **MED** `S8a.status.md:27` — guardrail 5(a) ships permanently red: 24 lines; 18 in `claude_code.go` (8), `codex.go` (5), `devin.go` (5). Latent: `[[:space:]]*=` also matches `==`.
2. **MED** `S8a.status.md:28` vs `profiler/claude_code.go:465` — 5(b) condemns legitimate `PresentTokenResult(tc, string(SourceSessionData))`, which D3 classes `harness_reported`. `ValidateProfile` rules (d)/(b) are correctly scoped; the over-reach is confined to rule 5(b).
3. **MED** `SC.status.md:23-24` — `SelectTokenSource(...) TokenResult` returns; it cannot be "the only function that assigns `Profile.Tokens`". The call site sits in a producer file, so 5(a) fires on the epic's own new code. Repeated at `roadmap:135`, `S0:16`, `S1b:32`, `S5:27`, `S6:19`, `S9` DoD 3.

## REAL — LOW
4. `roadmap.md:304` — "15 test files, 15 owners"; row enumerates 18 test files + 2 fixtures across 16 owners.
5. `SC.status.md:25` — "three slices could emit `otel_aggregate`"; round-4 table shows two.
6. `SC.status.md:12` — "four of its five writers"; S9 declares S5, so three.
7. Citation drift, 6 sites: `compare.go:88-93` → comparators are `:90-94` (`SC:34`, `S5:49`, `S7:140`, revlog:48,127); `ComparisonReport.Notes` field is `compare.go:40`, not `:83-88`.
8. `roadmap.md:344` — §H header claims every item carries a finding id; H3/H5/H7 carry section pointers. Items honest; header over-reaches.
9. `S5.status.md:49` — "S7 DoD 25"; S7 has five DoD items, referent is test bullet 6.
10. `revision-log-round4.md:15` — "+8 is §H"; §H is ~13 lines.
11. `revision-log-round4.md:93,112` — `roadmap:285`/`:301` should be `:287`/`:304`.

## INVALID (self-raised, cleared)
S9's Admin API figures: a summarizing fetch suggested the 30-day cap excluded `filtered-usage-events`; a verbatim fetch confirms it applies. C4-#5 stands; H6 mandates re-verification at build time.

**Verdict: NOT-CLEAN.** Three MED above LOW, one root: rule 5 scoped tree-wide while the invariant is Cursor-scoped, and the "only assigner" returns rather than assigns. One root fix closes 1–3. The LOWs share the standing root: a count, range, or universal claim asserted from the topic of the cited block rather than counted or read.
