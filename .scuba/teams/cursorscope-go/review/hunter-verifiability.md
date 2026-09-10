# Hunter report — slice verifiability lens (spec gate, round 1)

**Artifact:** `.scuba/teams/cursorscope-go/roadmap.md` + 13 slice files · **Date:** 2026-09-10 · **Verdict:** NOT-CLEAN

**Coverage:** walked 13/13 slices (S0–S11, T1) against 6 checks each (DoD testability, test approach, files, one-PR, dep-fit, table⟷file⟷Mermaid); 65/65 requirement IDs (R-CS-01…30, R-SA-01…17, R-RL-01…18); 21 graph edges, 6 wave rows, 3 contract files (`profiler/types.go`, `compare.go`, `experiment.go`) + `docs/profiler-spec.md`. 4 orphan requirements, 6 slices with untestable DoD.

**Shared root (R1):** the plan treats `profiler/types.go` + `docs/profiler-spec.md` as incidental edits owned by whoever happens to touch them, so four DoDs assert distinctions the F03 contract cannot express, and only S5 owns the spec doc.

## REAL — contract (root R1), all reproduced (probe in scratchpad `probe/main.go`)

1. `roadmap.md:120` / `S0.status.md:13` — "`Probe()` … reports `tokens: none` **with a reason**". `CapabilityReport` is `map[MetricName]MetricSource` (`types.go:50-55`); there is no reason field. Untestable as written. HIGH.
2. `roadmap.md:127` / `S7.status.md:13` — "`Probe()` reports hooks and OTel **independently**". Proven impossible: one source per metric; second assignment silently overwrites (`types.go:54`). S7's files exclude `types.go`. HIGH.
3. `roadmap.md:126` / `S6.status.md:13` — fallback "says so in `Source`": no aggregate-fallback member in the `MetricSource` enum (`types.go:41-47`), and S6 doesn't own `types.go`. MEDIUM.
4. `S0.status.md:19` — `ActivationEntry.Source` added while `docs/profiler-spec.md:135-139` documents the v1 shape; S0's files exclude the spec doc. MEDIUM.
5. **`SourceEstimated` uncovered downstream.** Proven: `compareTokens` (`compare.go:107,115`) gates only on `State==present` and reports **candidate**'s source only. An `estimated` baseline vs `server_api` candidate yields `comparable:true, source:"server_api", delta:{input:3000}` — the estimate's honesty label vanishes from the report. No slice owns this; S7's DoD (`roadmap.md:127`) only asserts `tool_calls`/`skill_activation`. Also contradicts `types.go:12` / `profiler-spec.md:19,33` ("present = value captured from telemetry"). HIGH.

## REAL — verifiability/deps

6. `S8.status.md:20` guardrail rule 2 ("`present` from a function named `estimate*`") forbids exactly what S5's Fork (vi) option 2 mandates; S8 depends on S3,S4 — **not S5**. Direct design conflict + missing dep. HIGH.
7. `S1.status.md:16` claims 13 requirements; its DoD (`:13`) proves ~3. R-CS-01 (21 events), 03, 05, 06, 07, R-SA-08, R-SA-13 untested. Secretly 3 PRs. HIGH.
8. `S5.status.md:6,13` — dep on S3 is phantom (no attribution needed); it parks S5 behind the slice most likely to collapse. Its "three labels" also mismatch Fork (vi) (`roadmap.md:204`): MCP-self-reported has no assigned label anywhere → DoD not executable. HIGH.
9. `roadmap.md:243` "Only S0 and S1 touch `profiler/cursor.go`" is false — S6 (`:126`), S7 (`:127`), S9 (`:129`) list it. Every parallel wave also has ≥2 slices writing the single `profiler/profiler_test.go`. MEDIUM.
10. `S10.status.md:19`, `S11.status.md:19` need `profiler/cmd/skill-scope/`, created by **S2**; both declare only S1 — undeclared transitive block on User Q1. `S11.status.md:13` needs S2's fixtures too. MEDIUM.
11. `S2.status.md:13,22` — the R-RL-17 spike is manual, hardware-gated, non-CI, and self-admittedly splittable; a spike that can void S3/S5/S7 sits *downstream* of S1 instead of gating it. HIGH (sequencing).
12. All 13 slice files omit the roadmap's **Test approach** column entirely — a dispatched worker reading only its slice file gets a DoD with no named test. MEDIUM, systemic.
13. `S3.status.md:13` — "`Trigger` explicitly marked as *inferred*" names no vocabulary, and collides with S0's `cursor.skill.trigger` enum (`roadmap.md:93`); no slice reconciles them. MEDIUM.

## REAL — coverage matrix (orphans both directions)

14. **R-SA-04** → `roadmap.md:74` assigns S1; `S1.status.md:16` omits it and the DoD never mentions `snapshot_hash`.
15. **R-CS-14** (tagged *adapt*) → slice "deferred" (`roadmap.md:49`): no slice. **R-CS-04** (*adapt*) → S11 only (`:39`), which `S11.status.md:22` recommends never ships. **R-RL-16** (*keep*) → S7, which `S7.status.md:24` explicitly declines to implement.
16. Phantom claims: `S11.status.md:16` claims R-CS-06 (roadmap→S1, and in the *original TTL* form Fork (i) deleted) and R-CS-23 (roadmap→S2); `S9.status.md:16` claims R-RL-08 (tagged **drop**); `S3.status.md:16` claims R-SA-11 with no doc file in its Files list.
17. `roadmap.md:181-183` Mermaid has no `S1 --> S7` edge though the table and `S7.status.md:6` both declare S1 a dependency. LOW.
18. `roadmap.md:122` says `jq` does the hooks.json merge while `S2.status.md:19` puts `hooks` in a Go cmd — merge owner ambiguous. LOW.

## DEFERRED (honestly flagged, not defects)

S5 gated on Q3 with two materially different DoDs; S8 partly gated on Q8; S2/S6/S9/S10/S11 conditionals.

## INVALID

None. Graph is **acyclic**; MVE {S0,S1,S2,S3,S4,S8} **closes** under the declared deps (it stops closing once finding 6 adds S8→S5); AGENTS.md conventions (no Python, grep-not-semgrep, no trailers) are clean across all 13 slices; `go vet`/`go test ./...` green at baseline.
