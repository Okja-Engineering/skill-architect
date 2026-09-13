# otlp-plan.md — plan gate, round 1 (2026-09-13 16:25)

Verdict on the plan as written: NOT CLEAN. 27 findings, 9 roots. Chief-of-staff decisions inline so the revision doesn't re-open them.

## A. Typed-struct decode turns OTLP-legal type variation into a whole-file error (REAL, high)
- A1 `timeUnixNano` typed `string`: an unquoted number (spec-legal: "either numbers or strings are accepted") fails the decode → all signals `error`. `UseNumber()` never reaches a `string`-typed field, so the plan's rationale at :37 is wrong.
- A2 `AggregationTemporality int` hard-fails on `"1"` or enum-name strings. Low exposure, wrong blast radius.
- A3 `Body struct{StringValue}` hard-fails on a plain-string body; research :939-940 requires guarding it.
- A4 `asInt64` routes int64 through float64, discarding the precision `UseNumber()` was added for.
**Decision:** leaf positions that the spec allows to vary (`timeUnixNano`, `startTimeUnixNano`, `asInt`, `intValue`, `aggregationTemporality`, `body`) decode as `any`/`json.RawMessage` with tolerant helpers; a leaf that cannot be read degrades that data point or record, never the file. int64 parsing goes through `strconv.ParseInt` on the string form or `json.Number.Int64()`, float only for `asDouble`/`doubleValue`. Unknown field names ignored (already).

## B. Helpers, fixtures, and pins inherited from a wider mapping the plan removed (REAL, medium)
- B1 `otlpAttrs.Int` has no caller. Delete or use.
- B2/B3 `int_value_as_string.json` pins `duration_ms`, which nothing reads; R2 is half-pinned.
- B4 `as_double_integers.json` at 999999 is vacuous (round == trunc). Use a value where they differ, or a non-integer double that must be rejected/rounded per the rule you choose, and state the rule.
**Decision:** every helper has a caller; every fixture pins a read path; every risk row cites a fixture that would go RED without the behaviour.

## C. `fold`'s signature cannot implement §2.4's cumulative rule (REAL, high)
Per-`otlpSum` call with `map[string]float64` cannot express "last value by timeUnixNano per series across batches", nor regroup to the four `type` buckets.
**Decision:** accumulator is keyed by series (full attribute set) and carries `(type, value, timeUnixNano, temporality)`, merged across all batches, then reduced to TokenCounts. Delta sums; cumulative last-by-time. `cumulative.ndjson` keeps its 900-not-1400 expectation.

## D. Reason-string and classification holes (REAL)
- D1 "none carried a tool_name" is false when the only events are accepted `tool_decision`s with no `tool_result` yet.
- D2 nil-`Sum` skip vs `seen` ordering makes `gauge_not_sum.json` produce one of two false reasons.
- D3 zero-byte / whitespace-only file: today `error` "unexpected end of JSON input"; under the plan → `unknown` "not OTLP/JSON", a misleading diagnosis. `empty.json` is `{}`, not empty.
- D4 `ToolCallEntry.Timestamp` when `timeUnixNano` absent: value and sort position unstated.
- D5 top-level array leaks a Go type name in the reason.
**Decision:** reasons are true of the input that produced them; enumerate the reason per input shape in a table (found nothing / found but no sum / found sum but no recognised type / etc.). Empty or whitespace-only file → `error` "export is empty". Top-level array → `error` "top-level JSON array is not an OTLP export". Entries with no timestamp sort last with `timestamp: ""` and the spec says so, or are dropped with the reason counting them; pick one and pin it.

## E. §6 misses five sentences that go false (REAL, medium)
README:65 tool_calls row; spec AC3 (:230, names `token_type`, `tool_decision`, no-timestamp); README:75-76 skill sentences; CHANGELOG:36 and RELEASE_NOTES:17 test counts; README:71-73 "all three come back error" now incomplete because §4 adds an `unknown` shape.
**Decision:** §6 lists every sentence by content, not by line number (see F), including the AC block :226-238 and both test-count sites.

## F. All §6–§7 line cites are stale at head (REAL, low-med)
Two docs commits landed after the plan. types.go:98 is `SkillName` (Reasoning is :86); RELEASE_NOTES:21 is the 0.4.0 heading (claim at :24); spec cites drift 3–5 lines and two name the wrong sentence; W7/W10 conflict note already resolved; test envelope is :163-176.
**Decision:** cite by quoted sentence plus file; line numbers only as hints marked "at 6205fdc". Implementer rebases and re-greps.

## G. Test-plan gaps (REAL)
- G1 no fixture pairs an `accept` `tool_decision` with its `tool_result` — the only evidence for "no tool_use_id dedup needed".
- G2 no `success:"false"` from a failed tool — the semantic change (mandate item-19 decision) is unpinned.
- G3 no redacted-skill-name fixture (`custom_skill`/`third-party` on token.usage → skill_activation `unknown` with the redaction reason). This is a mandate boundary.
- G4 no zero-byte fixture (see D3).
- G5 `TestProfileRoundTrip` not in the replace list; it goes vacuous silently after C2.
- G6 RED criterion "every OTLP fixture → unknown" is false for the .ndjson fixtures (today they `error` on concatenated JSON). State the true RED expectation per fixture.
**Decision:** add all six. Push C1 and C2 together (CI runs on the pushed head; a test-only push is red by design).

## H. Mandate drift and collector-config errors (REAL)
- H1 D5 is already decided in mandate.md (item-19 decisions block): success = execution outcome from `tool_result`, rejects listed from `tool_decision`. Remove the "manager ack" branch and the fallback.
- H2 one bare `file` exporter with two pipelines is inconsistent; either one shared file (nothing to concatenate) or two named instances `file/metrics`, `file/logs`.
- H3 fileexporter default `append: false` truncates on every collector start; the documented config must set `append: true`.
- H4 `OTEL_LOG_TOOL_DETAILS=1` is named in the new skill_activation reason but absent from the README env list.
**Decision:** one shared `file` exporter, `append: true`, `format: json`; env list includes `OTEL_LOG_TOOL_DETAILS` with one sentence on what it exposes.

## I. Risks the table omits (REAL, medium)
- I1 truncated final NDJSON line (collector stopped mid-write, `flush_interval`) → whole capture `error`. Most likely real-world complaint.
- I2 the documented collector route was read, not run.
**Decision:** I1: a parse failure is an `error` (never swallowed) whose reason names the batch index and byte offset and says the earlier batches were not used; README says restart the collector cleanly and how to trim a partial line. I2: the implementer attempts route (b) end to end only if `otelcol-contrib` is already installed or installs without a permission prompt; otherwise the README marks the snippet "derived from the fileexporter README; run route (a) to verify locally" and status.md records that no end-to-end run happened. Route (a) (http/json to a local receiver that persists bodies) is documented as the primary route because it was observed.

## Size / sequencing notes (accepted)
C2 will land nearer +320–360 than +260 at this package's comment density; the .ndjson-vs-pretty-printed fixture question needs a deliberate call (keep at least one true one-object-per-line file to pin framing); verification path nit at :220 (`testdata/otlp/…` after `cd profiler`, binary from `./cmd`).

## Confirmed clean (keep as is)
D1 file split and deletion list; D2 stream decode; §4 error-vs-unknown owner in `resolve`; token mapping rows; D4 `Reasoning` handling; D5 decision-vs-source reasoning; eventName normalisation; D6 timing span; helper semantics; no new deps; §9 out-of-scope.
