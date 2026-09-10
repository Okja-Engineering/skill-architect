# Hunter report — telemetry-name and factual conformance lens (spec gate, round 3 confirming pass)

**Artifact:** `roadmap.md` (325 ln) + 17 slice files (915 ln) · **Date:** 2026-09-10 · **Verdict:** NOT-CLEAN — 12 REAL, 1 SUSPECTED, 0 INVALID

## Coverage
18/18 files walked twice · 21 distinct `cursor.*` + 4 `gen_ai.*` / 1 `skill_architect.*` / 2 `cursor_*` — 0 invented (Wire Reference re-fetched live) · 15 enum sets / ~48 values · 21-event surface + 10 base fields + ~30 payload fields (hooks docs re-fetched) · 9 Admin-API strings + 3 limit claims (re-fetched) · 35/36 rows of `revision-log-round2.md` §2 spot-checked · ~50 numeric claims · ~70 `file:line` cites · 13 §G refs · 12 C2 dispositions. Baseline: `go vet` clean, 34/34 Go PASS, 131 bash PASS.

## C2 fix status
C2-1…12 all ✓ fixed at root, not reworded (C2-4 see R7; C2-6 see R8, R10).

## REAL (12)
1. **MED** `roadmap.md:149`, `S0:7`, `S8a:19` — "8 live `cursor.` literals"; actual **12** (`cursor.go:216,234,250,271` also, in `Unknown*Result` reason strings). Guardrail-4's own grep → 12. S8a stays RED after S0; S0 DoD 1 never covers reason strings.
2. **MED** `S7:15` — encoding-invariance claimed across hooks↔OTel. SC:22 makes `success_rate` denominator depend on `Status` being set; S1b:23 leaves hooks' `Status` empty. 3s/1f/2a → 0.75 vs 0.50.
3. **MED** `SC:15,37`, `roadmap:138` — `Attribution.Target` quoted "tool call ID"; `types.go:114` / `profiler-spec.md:154` say "tool call ID **or output identifier**". SC's referential rule narrows a shipped contract; contradicts D1 additive-only.
4. **MED** `S1b:14` — `ParentID` fed from `tool_call_id` *or* `parent_conversation_id` (two identifier kinds, one field); `ID`←`tool_use_id` but `subagentStart` spells it `tool_call_id` (ledger:131, live-confirmed).
5. **LOW-MED** `roadmap:60`, `S4:13,18` — 8192 attributed to `src/privacy.js`; it is `gen-ai-semconv.js:34` (`TOOL_PAYLOAD_MAX_LEN`, env-overridable). `privacy.test.js` has no truncation case. "exactly 8192 bytes with a marker" ≠ cursorscope's `slice(0,8192)+"…"`.
6. **LOW-MED** `S9:31`, `roadmap:104` — ledger §A5.3:177 does not pin the 30-day maximum to `daily-usage-data`/`audit-logs`. Facts right live; attribution invented.
7. **LOW** `roadmap:28` — "Two Go/hook prior-art tools"; o11y-dev is Python 3.12+ per source and per D10 itself.
8. **LOW** `roadmap:100` (R-RL-03) — cited ledger:200 is labelled Repository fact, not External.
9. **LOW** `revision-log-round2.md:96` — "§A4:300"; line 300 is Decision-questions Q1.
10. **LOW** `roadmap:51,107` — "−7%…+15%" cited to §A4:83; :85-90 measures −6.6%; the rounded phrase lives at ledger:300.
11. **LOW** `roadmap:98` — R-RL-01's `cursor.hook.name|.type|.outcome|.duration_ms` half has no S0 DoD carrier.
12. **LOW** `roadmap:65` — R-CS-25 keeps `--dry-run`/`--yes`; S2 claims R-CS-25 but neither flag appears in its DoD or tests.

**Shared root (1,5,6,8,9,10):** counts and labels still transcribed from the topic of the cited block rather than re-derived by executing/reading the cited line. Invariant: every count must be produced by the rule that will consume it; every label must be the cited line's own label.

## SUSPECTED (1)
`roadmap:121`, `SC:19,31` — D3 totality omits the empty string, the zero value the shipped constructors leave (`types.go:196,216,231,246,261`); SC tests only `"wat"`.

## Upstream / out-of-lens
- `ledger.md:111` claims cursorscope's list "matches exactly" — it is 19, missing `beforeTabFileRead`/`workspaceOpen`. R-CS-01 is correct.
- `S8b:22` edits `tests/test_guardrails.sh`, absent from its `## Files`.
