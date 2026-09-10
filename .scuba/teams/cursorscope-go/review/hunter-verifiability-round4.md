# Hunter report — slice verifiability lens (spec gate, round 4 confirming pass)

**Artifact:** `roadmap.md` + 17 slice files · **Date:** 2026-09-10 · **Verdict:** NOT-CLEAN — 1 HIGH, 8 MED, 7 LOW

**Coverage:** 17/17 slices × 6 checks · 65/65 requirement IDs both directions · 33/33 edges recomputed (identical, acyclic, both MVEs close) · 29/29 §1 source-supply rows re-verified against `ledger.md` · every cite resolved · V3-1…16 + C3-1…12+S · 2 sweeps. `go vet` clean; `go test ./...` ok (34 tests).

**V3 fix table:** #3–13, #15, #16 fixed at root, verified. #1, #2, #14 fixed but class not swept. Round-3 citation register 100% accurate (12/12 re-derived).

## REAL

| # | file:line | Sev | Detail |
|---|---|---|---|
| 1 | `S0.status.md:16` vs `S6.status.md:14-17` | **HIGH** | S0 reads the same `cursor.token.usage` aggregate and labels it `otel`, assigning `Profile.Tokens` directly. S6 DoD 2 says that data must be `otel_aggregate`; S5:25 says `selectTokenSource` is the only writer of `Profile.Tokens`. D12/§2 swept S5/S6/S9 and missed S0 — the fourth writer, which ships in the MVE while S6 is conditional. |
| 2 | `S6.status.md:6,19` | MED | DoD 4 calls `selectTokenSource` (defined by S5, in `tokens.go` created by S1b); `Depends on` = SC, S0 only. Same wave 3 as S5 ⇒ won't compile. Edge missing in slice file, table, Mermaid. |
| 3 | `S5.status.md:25` | MED | Rank order `server_api>otel>mcp_reported>estimated` is not total — `otel_aggregate` unranked and absent from S5's candidate table. |
| 4 | `S7.status.md:25` vs `SC.status.md:12-30` | MED | S7 asserts `tokens`/`context_window` "compared as separate metrics"; `CompareProfiles` (`compare.go:88-93`) has 5 comparators and no `context_window`. Only SC may edit `compare.go`; no SC DoD adds it. |
| 5 | `SC.status.md:16,41` | MED | Item 4's referential rule is "a contract violation the test catches" — no validator exists (`grep -n 'func.*Validate' profiler/*.go` → none) and no DoD creates one. |
| 6 | `experiment.go:14,17` | MED | `ExperimentSchema` + `ExperimentPlanSchema` are two more strictly-validated wire strings. S7 DoD 2 adds `HookSpool` inside both. D13 closed only `ComparisonSchema`; no omitempty/last/golden rule for these. |
| 7 | `S2:29`, `SP1:31`, `S8b:28` | MED | `ci.yml` enumerates the four bash scripts by name (`.github/workflows/ci.yml:24-31`); no glob. `test_hooks_install.sh`, `test_spike_activation.sh`, `test_doc_labels.sh` never run — none of the three lists `ci.yml` in `## Files`; §F:300 claims only S1a+S8a write it. |
| 8 | `SC.status.md:26` vs `:40` | MED | "zero delta regardless of row encoding" is falsified by SC's own next bullet (Status-less encoding → 0.50 vs 0.75). No comparability gate for Status-bearing vs Status-less profiles of one session. |
| 9 | `S1b:31`, `S5:34` | MED | D12's invariant "`hooks` is never a token-flow label" has no test and no guardrail in any slice. |
| 10 | `S3.status.md:20` | LOW | Both attribution forms come from `beforeReadFile` ⇒ same `Trigger`; only the scheme prefix distinguishes them. |
| 11 | `roadmap.md:285-301` | LOW | Ownership map incomplete: `guardrails_test.go`, `cursor_admin_api.go`, `otlp.go`, ~10 `_test.go`, SP1's fixture absent. All single-owner. |
| 12 | `S8a.status.md:24` | LOW | Rule-2 seed placement unstated; `ParseDir` sees `_test.go` by default ⇒ naive seed makes CI permanently red. Walked types omit `ContextResult`. |
| 13 | `S4.status.md:14` | LOW | "1-byte marker `[+]`" — 3 bytes. |
| 14 | `S6.status.md:25`; `roadmap.md:89` | LOW | R-SA-13 assigned to S6 in register; S6's `## Requirements` omits it. SC claims R-RL-18; register row lists only S5. |
| 15 | `S1a.status.md:21` | LOW | Permitted-field list omits `stop.status` (`ledger:135`), named in revlog §1 row 25. |
| 16 | `S10:6`, `S9:6` | LOW | DoDs name SC's fields/rules; SC not declared (transitively reachable). |

**Roots:** (A) D12's sweep stopped at S5/S6/S9 — S0 unswept, `otel_aggregate` unranked, S6's edge unadded → 1,2,3,9. (B) SC's DoD adds channels/rules but not the code that enforces them → 4,5. (C) one instance patched, class not swept (D13 vs experiment schemas; ci.yml wiring; ownership map) → 6,7,11. (D) rule stated wider than mechanism → 8,10,12,13.

**Matrix/graph:** 65 IDs, 0 orphans, 0 phantoms (2 LOW asymmetries). 33 edges, acyclic, file/table/Mermaid identical; both MVEs close; wave collisions = `profiler/cursor.go` S0‖S1b only. 17/17 name a test file.
