# Hunter report — telemetry-name and factual conformance lens (spec gate, round 4 confirming pass)

**Artifact:** `roadmap.md` (335 ln) + 17 slice files · **Date:** 2026-09-10 · **Verdict:** NOT-CLEAN — 3 MED + 6 LOW REAL, 0 INVALID. No HIGH remains.

**Coverage:** 18/18 artifact files walked twice · 29/29 §1 source-supply rows spot-checked against ledger §A5.1/§A5.2/§A5.3 and live re-fetch of cursor.com hooks + OTel Wire Reference + Admin API · 21/21 events, 10/10 base fields, 16 distinct `cursor.*`, 28 snake_case field names — 0 invented, 0 misspelled · 24 labels sampled (all carried by cited line) · 13/13 C3 dispositions · guardrail greps 1 and 4 executed · 34 Go + 131 bash tests run green.

**C3 fix table:** C3-1…12 + SUSPECTED all fixed at root. Re-derived by running: literals=12 at `cursor.go:175,182,184,186,197,216,224,234,242,250,262,271`; `gen-ai-semconv.js:34` (`TOOL_PAYLOAD_MAX_LEN`), `:223,226` slice+marker; `privacy.test.js` 6 cases, no truncation; −6.6% at ledger :90, rounded at :92/Q1:300; 33 graph edges reconcile both directions.

## REAL
1. **MED** `S8a.status.md:17,24` — rule 1 ships red. Executed: `scripts/` absent (exit 2); `tests/test_skill.sh:23`, `tests/test_walk.sh:25` already invoke `python3`. Red→green DoD unsatisfiable. Rule text also wider than `AGENTS.md:6` ("skill scripts"). Same root as C3-1, fixed for rule 4 only.
2. **MED** `S6.status.md:15-16` vs `:19`, `S5:25`, `roadmap:135` — S6 emits `Source:"otel_aggregate"` but "registers an `otel` candidate"; `selectTokenSource`'s order (`server_api>otel>mcp_reported>estimated`) has no rank for `otel_aggregate`, though D3 classes it. Invariant: every source a slice can emit for `tokens` must be rankable.
3. **MED** `S1b.status.md:24` — "The profile … states the caveat verbatim" has no carrier: `RawMetricResult` is `{State,Value,Reason,Source}` and `Reason` must be empty when present (`profiler-spec.md:33`, retained by D2); `Notes` is on `CapabilityReport` only (`SC:19`). Root R1 recurrence.
4. **LOW** `roadmap:122`, `SC:20` — "six `Unknown*Result`" but five exist; the sixth `Source==""` constructor is `ErrorTokenResult` (`types.go:201`), cited nowhere.
5. **LOW** `S9:34` — self-contradicts on 30-day cap; live docs: 60 req/min, and the 30-day cap does apply to `filtered-usage-events`.
6. **LOW** `roadmap:42`, `S1a:15` — "§A5.1:115-137, 21 table rows"; rows are `:117-137`.
7. **LOW** `S4:14` — "1-byte marker … `[+]`"; `[+]` is 3 bytes.
8. **LOW** `SC:16`, `S3:17`, `roadmap:134` — "tool call ID or output identifier" cited to `types.go:114`, which reads "or output".
9. **LOW** `roadmap:285` — "Every file any slice touches appears here" omits SP1's fixture and 12 `_test.go` files.

**Shared root (1,4,5,6,8):** a command/count/quote asserted from the topic of the cited block rather than executed or read verbatim — surviving in five new places.
