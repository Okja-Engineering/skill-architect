# Hunter report — telemetry-name and factual conformance lens (spec gate, round 1)

**Artifact:** `.scuba/teams/cursorscope-go/roadmap.md` + 13 slice files · **Date:** 2026-09-10 · **Verdict:** NOT-CLEAN (15 REAL, 4 low)

**Coverage:** 14/14 artifact files (roadmap + S0–S11 + T1) walked twice. Enumerated **85 telemetry/field/endpoint/identifier/version strings** (21 `cursor.*`, 2 `gen_ai.*`, 1 `skill_architect.*`, 6 hook events + the 21-event set claim, 15 payload fields, 9 Admin-API strings, 10 package/version, 20 Go identifiers): **84 verified, 1 untraceable, 7 mislabelled.** Enumerated **37 numeric claims: 33 verified, 2 wrong, 1 unsupported by cited source, 1 unmeasured-but-gating.** 39/39 cited paths resolve; 10/10 "(new)" files absent. Verified by fetching the Cursor hooks / OTel Wire Reference / Admin API / Claude Code skills docs, grepping the cursorscope clone, and running `go test` (34 PASS), `go vet` (clean), all 4 bash suites (131 assertions, 0 fail), and `go list -deps` on `otelspike2`.

## Root A — thesis written from the ledger narrative, not the adapter's states

1. `roadmap.md:13` HIGH REAL (reproduced): "all four return `skill_activation: unknown`" (**Repository fact**) is false. `profiler/cursor.go:259-273` returns `PresentActivationResult`; `profiler/profiler_test.go:644-652` passes asserting `present` + `SkillName=="audit"`. Wrong *keys* yield present-but-empty, not `unknown`. (`attribution: unknown` half is correct.)
2. `roadmap.md:123`, `S3.status.md:10` MED REAL: "first non-`unknown` activation any adapter has ever emitted" — same root; weakens R-SA-07's framing.

## Root B — labels assigned per row by topic; ledger hedges upgraded on transcription

3. `roadmap.md:99`+`S9.status.md:24` MED REAL: "20 req/min/team" for `filtered-usage-events`; docs say **60/min**. 20/min belongs to `daily-usage-data`/`audit-logs`.
4. Same lines MED REAL: "30-day max range" is documented for `daily-usage-data`/`audit-logs`, **not** `filtered-usage-events`.
5. `roadmap.md:50` (R-CS-15) MED REAL — **the one untraceable name**: `exit_code` labelled *Specification*, but Cursor's `afterShellExecution` documents only `command`/`output`/`duration`/`sandbox`; ledger §A5.1 omits it too. Traces only to cursorscope CHANGELOG 0.3.7 + `src/telemetry.js:530`. S1's DoD depends on it against a hand-authored fixture.
6. `roadmap.md:103` REAL: `skillListingBudgetFraction` 0.01 labelled *Specification (code.claude.com/docs/en/skills)* — that page doesn't mention it; ledger §B1 files it as External evidence. Drops the ~200K-reference caveat.
7. `roadmap.md:44` (R-CS-09) MED REAL: labelled *Specification* citing ledger §A5.1 finding 3, which says the opposite — **Local hypothesis**, no skill hook exists.
8. `roadmap.md:47` (R-CS-12) LOW REAL: `result_json` carrying usage is cursorscope's probe (External), not Specification.
9. `roadmap.md:101`/`T1.status.md:23` LOW REAL: "real `o200k_base` BPE" labelled *Repository fact*; it's External (ledger §A4). The `audit-report.sh:36` / `test_f02.sh:100` half is correct — verified exactly.
10. `roadmap.md:42` (R-CS-07) LOW REAL: "no dotenv" cited to AGENTS.md as *Repository fact*; AGENTS.md (23 lines) has no env/dotenv rule.

## Root C — S0's DoD written without checking the current type surface

11. `roadmap.md:120`/`S0.status.md:19` MED REAL: adds `ActivationEntry.Source` but omits `docs/profiler-spec.md`, where the schema is normative (`:135-139`, enum at `:28`) — code/doc drift by construction.
12. Same MED REAL: "`Probe()` reports `tokens: none` **with a reason**" — `CapabilityReport` is `map[MetricName]MetricSource` (`types.go`, spec `:52-58`); no reason channel exists.
13. `roadmap.md:120,126` MED REAL: `cursor.tool.calls` is a delta **counter** — `otelMetric` (`claude_code.go:197`) has no timestamp, `Value` is N calls not one entry, `cursor.tool.status` is tri-state vs `Success bool` (`aborted` unrepresentable), and metric attributes carry no `cursor.conversation.id` (a *log* join key). "tool_calls present with correct values" overstates the wire shape. Diagnosis at `cursor.go:224-231` is correct.
14. `roadmap.md:86,229` LOW-MED REAL: counts right (131 = 32+58+22+19; 34 Go), but "stay green" is unachievable for S0, and "**extend** existing Cursor tests" should be **replace** — `profiler_test.go:560,564,565,605,626` pin the defect, including `"trigger":"keyword"` (not in the real enum) and `Reasoning==150` that R-RL-01 deletes.

## Root D — two requirement registers merged without a conflict pass

15. `roadmap.md:51` (R-CS-16, reuse `cursor_*_total`) vs `:105` (R-RL-13, `skill_architect.skill.*`), both in `S10.status.md:16`, unresolved — emitting `cursor_*` from a non-Cursor tool is exactly the R-SA-14 failure. Also S8 guardrail #4 (`S8.status.md:22`, one file may hold `cursor.*`) contradicts S10's DoD requiring wire-correct keys in a new `profiler/otlp.go`.

## Low

- `roadmap.md:256` (§G6 "unmeasured") vs `:121`/`S1.status.md:13` gating on "<50 ms".
- `S4.status.md:13` uses a 4 KB prompt to prove truncation at 8192.
- `roadmap.md:23,107` overreads AGENTS.md:6 (scoped to "skill scripts").
- Ledger §A4's flagged `AGENTS.md:11` skillscore error (npm/7-dimension → Dart/six) silently dropped while the CI go-version note was carried.

## Clean

All 21 `cursor.*` names/keys confirmed against the Wire Reference (nothing invented); 21-event hook surface and every payload field confirmed; Admin API verb/path/auth/`conversationId`/`tokenUsage{...}` confirmed; cursorscope's wrong poller confirmed at `src/cursor-api-poller.js:44,47`; Fork (v) numbers reproduced exactly (2+22 modules, 408 packages, 66 grpc, 17.7 MB / 6.4 MB / 0 modules); 4,030 = 1,814+2,216, 55%, 8192, 104 lines, 0.3.7, `_testHooks`, `SKILL_PATH_RE`, go 1.27.1 vs CI 1.23, `~/.cursor` absent, 2.1.221, ACES figures, 1,536/500, −7%…+15%.

Second sweep over the concatenated files produced no string or number outside the enumerated set and no new finding.
