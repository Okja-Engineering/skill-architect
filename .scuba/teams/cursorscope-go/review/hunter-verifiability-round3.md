# Hunter report — slice verifiability lens (spec gate, round 3 confirming pass)

**Artifact:** `roadmap.md` + 17 slice files · **Date:** 2026-09-10 · **Verdict:** NOT-CLEAN (5 HIGH, 7 MED, 4 LOW)

Baseline: `go vet ./...` clean, `go test ./...` ok (34 Go tests green).

**Coverage:** 17/17 slices × 6 checks (DoD-carrier, deps, requirements, test-file, Files, contract-fit); 65/65 requirement IDs both directions; 32/32 Mermaid edges recomputed vs 17 `depends_on` + roadmap table + revlog §3; 6 wave rows × all 17 `Files` sets pairwise; V2-1…19; 2 sweeps.

## REAL

| # | file:line | Sev | Detail |
|---|---|---|---|
| 1 | `SC.status.md:21` vs `:22` | HIGH | **Reproduced.** DoD 9 asserts zero delta "regardless of row encoding"; DoD 10 keeps the `Success`-over-all-rows denominator when `Status` empty — which S1b (`S1b.status.md:23`) guarantees. One 6-call session: per-call rate 0.50, aggregate 0.75. SC's test uses homogeneous fixtures, so it cannot catch it. Poisons `S7.status.md:15`. |
| 2 | `S5.status.md:17-24` | HIGH | One `TokenResult` = one `Source` (`types.go:23`; spec:35 "Source always populated"). Four labels cannot ride one `TokenCounts`. SC adds value fields, no per-value source channel. S9:13 silently overwrites. |
| 3 | `S1b.status.md:13,15` | HIGH | No hook event carries a timestamp (ledger:113 base fields). `TimingData.StartTime/EndTime` have no hook carrier; §1 rows "S1b 1/3" marked ok on field-existence alone. |
| 4 | `S1b.status.md:23` | HIGH | `Success`/`Status` "derive from `afterShellExecution` (`command,output,duration,sandbox`)" — none is a success signal. Real carriers `postToolUse`/`postToolUseFailure` named nowhere. |
| 5 | `S3.status.md:17` | HIGH | `beforeReadFile` carries no `tool_use_id` (ledger:128). `Attribution.Target`→`ToolCallEntry.ID` unsatisfiable from the inferring event; join unspecified. SP1's DoD never asks. |
| 6 | `S1a.status.md:13` | MED | DoD pins only base fields; S1b/S3/S5 read `tool_use_id`, `file_path`, `context_tokens` from the spool. Same root as 3. |
| 7 | `S0.status.md:19` | MED | "adapter capability table" does not exist in `docs/profiler-spec.md`; `:135-139` is `ActivationEntry`. §1/§F repeat it. |
| 8 | `S6.status.md:14` | MED | `cursor.token.usage` has no conversation id (ledger:158) — the `--session` fallback is unscopable; nothing pins it. |
| 9 | `S8a.status.md:16-17` | MED | Rule says "inside **or returned from**"; mechanism is a body-walk. Seeded `estimateBogus` proves only the narrow half. |
| 10 | `S11.status.md:17` | MED | DoD 3 binds S4's redactor; deps = {S1a,S2}. V2-#9's own invariant, reintroduced. |
| 11 | `S8a.status.md:18` vs `S11.status.md:32` | MED | serve.go calls the spool writer → guardrail 3's grep fires. Unnamed exception (cf. D4/rule 4). |
| 12 | `SC:15` vs `S1b:14` | MED | `ParentID` = parent tool call vs `parent_conversation_id`. "Chains" untestable. |
| 13 | `S8b.status.md:22,28` | LOW | `tests/test_guardrails.sh` absent from `## Files`; fallback unnamed; "claim line" undefined. |
| 14 | `SC.status.md:20` | LOW | `MetricComparison` + tool_calls map shape change `comparison/v1` with no version rule; D1 covers profiles only. |
| 15 | `SC.status.md:30` | LOW | Golden provenance unpinned — indented/reordered golden proves formatting, not additivity. |
| 16 | `roadmap.md:44,88` | LOW | R-CS-04 Slice omits S11; R-SA-13 omits S9. |

**Roots:** (A) §1 proved *field exists*, never *source supplies + producer asserts* → 2,3,4,5,6,7,8. (B) DoD 9/10 define `successes` twice → 1. (C) rule/mechanism and DoD/deps scope mismatch → 9,10,11,13.

**V2 status:** fixed — 2 (partial: see 12), 6, 7, 8, 10, 11, 13, 14, 15, 17, 18, 19; **not fixed** — 1 (F1), 3 (F15), 4 (F2), 12 (F9); 5 fixed and verified fail-closed; 16 fixed but collides with 9. **V2-#9: concur** with the groomer's rebinding on the merits — but see F10.

**Matrix/graph:** 65 IDs, 0 orphans, 0 phantoms; T1 has `## Requirements`; S9/S10/S11/T1 have `## Files`; all 17 name a test file except S8b. 32 edges, acyclic, file/table/Mermaid identical. Primary MVE closes; fallback closes. Wave collisions: `profiler/cursor.go`, S0‖S1b only — groomer's claim verified.
