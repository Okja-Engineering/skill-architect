# Item 19 — implementation plan v3: parse real OTLP/JSON in the Claude Code adapter

_Architect, 2026-09-13. **v3**, planned against worktree head **`527ba4e`**. Ground truth: `otlp-research.md`. Gate decisions closed here: `otlp-plan-gate-round1.md` roots A–I and `otlp-plan-gate-round2.md` REAL-1…13 + nits 1–13. Invariants: `worklist-round1.md` Roots 1–2. Round-1 residuals R-F2…R-F7 are allocated in §6/§7. Every doc sentence quoted in §7 was re-grepped at `527ba4e` for this revision; line numbers corrected where they were off._

## 0. Preconditions

1. **Rebase onto the branch head first.** Round 1 is fully landed through `527ba4e` (docs W10–W18 included). `git -C /Users/matthewvandusen/Development/Auraprix/skill-architect-wt/release-0.4.1 fetch && git rebase origin/release/0.4.1`. Rebase again before pushing.
2. **Do not re-litigate round 1's shape.** `resolve()` / `otelSignals` / `capabilityReport()` / `sourceOf()` (`claude_code.go:40-139`) are the invariants: one file resolution, one failure classifier, one predicate per signal shared by Probe and Capture, `present` only when a value was assimilated. This change replaces **what `resolve` decodes and what the three extractors walk** — nothing above that line moves.
3. **Line numbers are hints at `527ba4e`.** Edit by the quoted sentence; re-grep after rebase. Verify RED before GREEN and record the failure output in `status.md`.

## 1. The integration seam (this is a swap, not an addition)

The bug is not "the adapter lacks an OTLP parser". It is that **wire decoding and signal semantics were the same code**: `claudeCodeOtelExport` (`claude_code.go:165-182`) is simultaneously a JSON shape and the adapter's model of Claude Code, so an invented shape was invisible. Round 1 cut the seam correctly — extractors take a decoded value and return a settled `*Result`. Item 19 puts a real format layer beneath that seam and deletes the bespoke one.

**D1 — two files, one package.** New `profiler/otlp.go` holds the format layer (envelope structs, leaf readers, stream reader, token accumulator). `profiler/claude_code.go` keeps the adapter and the three extractors, which speak only `otlpMetric` / `otlpLogRecord`.
- _Rejected:_ rewrite in place in `claude_code.go` (~500 lines mixing wire format with harness semantics — the shape that hid the envelope). _Rejected:_ a `profiler/otlp` sub-package — one consumer; a package boundary would export ~12 identifiers to buy nothing. Note the future Cursor/Codex consumers in the file header; do not build it now.

**Authorized refactor:** delete `claudeCodeOtelExport`, `otelMetric`, `otelLog`, `toInt`, `getString`, `parseTime` (`:165-182`, `:294-325`) outright. **As Go identifiers they are referenced only inside `claude_code.go`** (verified by grep at `527ba4e`); `profiler_test.go` does not name them — it depends on the same shape through JSON string fragments (`:449-458`) and inline exports, which C1 replaces with fixture files. No shim, no second accepted input format.

## 2. Parser design (`profiler/otlp.go`)

### 2.1 Reading the file — one prelude, one stream

**D2 — classify the stream prelude once, then stream objects.** `os.Open` replaces `os.ReadFile` at `:68`. Wrap in `bufio.Reader`; **strip a leading UTF-8 BOM (`EF BB BF`)** (nit 8); `Peek` the first non-whitespace byte **before** constructing the decoder:

| First non-space byte (post-BOM) | Outcome |
|---|---|
| EOF (file empty or all whitespace) | `error`, reason `"OTel export file is empty"` |
| `{` | stream with `json.Decoder` over the same `bufio.Reader` |
| `[` `"` `t` `f` `n` digit `-` | `error`, reason `"top-level JSON value is a <kind>; an OTLP export is a JSON object (one ExportMetricsServiceRequest or ExportLogsServiceRequest per object)"`, `<kind>` ∈ array/string/boolean/null/number from the byte |
| anything else | hand it to the decoder; its syntax error names the offending byte |

Then `for batchIdx := 1; dec.More(); batchIdx++ { off := dec.InputOffset(); var b otlpBatch; if err := dec.Decode(&b); err != nil { … } ; batches = append(…) }`. **Batch indices are 1-based** — stated at the loop in a comment and pinned by the `truncated_final_line.ndjson` fixture ("batch 4" for 3 good batches + 1 partial) (REAL-9). One path covers a single object, NDJSON, and concatenated objects with no newline. A decode failure is an `error` and is **never** swallowed (§4 for its typed reason).
- _Rejected:_ the research's "whole-file `Unmarshal`, fall back to line-by-line" (§6.2) — that fallback swallows the first parse error, the W5 defect round 1 just repaired, and is more code for less coverage. _Rejected:_ `dec.UseNumber()` — every numeric leaf is `json.RawMessage` (§2.2), so it would be a dead call.
- _Rejected:_ a dedicated `[`-only branch (v2's shape). One kind-from-byte mapping covers every non-object top level and removes the special case; a bare string or number gets the same honest sentence instead of a decoder message.

### 2.2 Envelope structs — tolerant at every varying leaf

```go
type otlpBatch struct {
    ResourceMetrics []struct{ ScopeMetrics []struct{ Metrics []otlpMetric } }
    ResourceLogs    []struct{ ScopeLogs    []struct{ LogRecords []otlpLogRecord } }
}
type otlpMetric    struct { Name string; Sum *otlpSum }   // Sum nil ⇒ gauge/histogram ⇒ counted, skipped, never panic
type otlpSum       struct { AggregationTemporality json.RawMessage; DataPoints []otlpDataPoint }
type otlpDataPoint struct { Attributes otlpAttrs; TimeUnixNano json.RawMessage; AsDouble, AsInt json.RawMessage }
type otlpLogRecord struct { TimeUnixNano json.RawMessage; Body json.RawMessage; Attributes otlpAttrs }
type otlpAttrs     []struct{ Key string; Value struct{ StringValue *string; IntValue, DoubleValue json.RawMessage; BoolValue *bool } }
```

**D3 — `json.RawMessage` at every leaf the OTLP spec lets vary; typed Go at every leaf it does not.** The spec mandates that 64-bit ints decode from "either numbers or strings"; `Body` may be an object, a bare string, or absent; enums arrive as `2` or `"2"` in the wild. A struct tag of `string` or `int` at those positions turns legal variation into a whole-file `error` — the A-root defect. The walk itself (`resourceMetrics` → `scopeMetrics` → `metrics`) stays fully type-checked, and a value that does not fit it is a file-level `error` with a typed reason (§4, nit 13).
- _Rejected:_ `map[string]any` throughout — untyped walk; the invented-shape failure mode returns. _Rejected:_ `any`-typed leaves with a type switch — `RawMessage` keeps the raw bytes, so `strconv.ParseInt` sees the exact digits and 1.79e18 cannot round-trip through a float64.

**Per-record degradation is the rule, per-file failure is the exception.** A leaf that cannot be read degrades exactly one data point or one log record and is skipped; it does not fail the file. **The profile does not report how many records were skipped** — a `present` result carries no reason in profile/v1, and inventing a channel for one is a schema change (0.5.0). This is stated in the spec (§7 item 7) and pinned by `as_double_rounding.json` (REAL-5). Iterate every level; never index `[0]` (multiple resource/scope entries are legal, research §6.2; pinned by `tokens_only.json`, nit 1). Struct tags are lowerCamelCase per the OTLP JSON encoding. Unknown field names, metric names and event names are ignored silently (research §6.4); do not assert on `scope.name` or `service.name`.

### 2.3 Leaf readers and accessors — every one has a caller

| Function | Accepts | Called by |
|---|---|---|
| `jsonString(raw) (string, bool)` | JSON string | `otlpAttrs.String`, `eventName` (body object + bare-string body) |
| `jsonInt64(raw) (int64, bool)` | JSON number or decimal string, via `strconv.ParseInt` on the unquoted digits; rejects a fractional value | `nanoTime`, `otlpDataPoint.value` (`asInt`), `otlpSum.temporality` |
| `jsonFloat64(raw) (float64, bool)` | JSON number or numeric string, `strconv.ParseFloat` | `otlpDataPoint.value` (`asDouble`) |
| `(otlpAttrs) String(key) (string, bool)` | `stringValue` | tokens `type`; tool calls `tool_name`, `decision`; `eventName` fallback `event.name` |
| `(otlpAttrs) Bool(key) (bool, bool)` | `boolValue`, or `stringValue` `"true"`/`"false"` via `strconv.ParseBool` | tool calls `success` |
| `(otlpAttrs) seriesKey() string` | every attribute, sorted by key, joined `key=<raw value bytes>\x00…` | token accumulator (nit 4) |
| `nanoTime(raw) (time.Time, bool)` | nanosecond string **or** number → `time.Unix(0, n).UTC()` | tool-call timestamp, timing span |
| `(otlpDataPoint) value() (int64, bool)` | `asDouble` first → `int64(math.Round(v))`; else `asInt` | token accumulator |
| `(otlpSum) temporality() (int64, bool)` | number, decimal string, or the three canonical enum names | token accumulator |
| `eventName(r otlpLogRecord) string` | §3.4 | tool calls, timing |
| `(otlpBatch) hasEnvelope() bool` | any `resourceMetrics` or `resourceLogs` entry | `resolve` (§4) |

The `bool` second return is round 1's assimilated-value rule made mechanical: absent and zero are different answers. `otlpAttrs.Int` from v1 is **deleted** — nothing reads an integer attribute (`duration_ms` is out of scope per D6), and a helper with no caller is the accretion this plan exists to remove.

`asDouble` first, `asInt` second: Claude Code emits `asDouble` even for integral counters ([BINARY], research §2.4); a collector in the path may re-serialise as `asInt`-as-string or as a bare number (nit 3). **Round, never truncate** — `1522.7` must yield `1523`, because a float64 representation of an integral count can land a hair low. Float arithmetic touches `asDouble`/`doubleValue` only; every other integer goes through `strconv.ParseInt` on the exact digits.

`nanoTime` **replaces** `parseTime` (`:319`), which returns zero silently on `"1789332594304000000"` and would make every timing signal falsely unknown. Delete it; do not branch between the two. The `event.timestamp` RFC3339 attribute is deliberately **not** a fallback — it duplicates `timeUnixNano`, and a second time source is a conditional that breeds. Attribute lookup is a linear scan (≤15 attrs, 2–4 lookups per record); _rejected:_ flattening to a map per record, an allocation and an intermediate representation buying nothing.

### 2.4 Token accumulator — series-keyed, merged across batches

**D7 — one accumulator for the whole file, keyed by series, reduced to `TokenCounts` at the end.** A per-`otlpSum` `map[string]float64` cannot express "last value by `timeUnixNano` per series across batches", which is exactly what cumulative temporality requires.

```go
type tokenSeries struct { tokenType string; value, timeNanos int64; hasTime, cumulative bool }
// key: attrs.seriesKey() — every attribute, sorted, raw value bytes — the OTel definition of a series
acc map[string]*tokenSeries
```

Per data point, after `type` and `value` both read, with `t, ok := sum.temporality(); cumulative := !(ok && t == 1)` — **delta only when temporality reads as exactly `1` (DELTA); unreadable, absent, or `0` (UNSPECIFIED) is treated as cumulative** (nit 6: the direction that cannot double-count; see §10): **new key** → store; **existing, delta** → `value += v`, keep the later `timeNanos`; **existing, cumulative** → keep the point with the greater `timeUnixNano`, and if neither has a readable time or they tie, the greater value (deterministic either way). A series seen with **both** temporalities (no producer emits this) is treated as cumulative for the rest of the file — same non-double-counting direction. One line, one comment, no global mode flag.

Reduce: accumulate per bucket in `int64`, then **clamp into `TokenCounts`'s `int` fields at `math.MaxInt`/`math.MinInt` — never wrap** (nit 5); a wrapped count would serialise as a negative token total, which is worse than a clamped one. Pinned by a direct unit test on reduce, not a fixture (a 2^63 fixture teaches a reader nothing). For each series, add `value` into the bucket named by its `tokenType`. Two `input` series on different models therefore both contribute — `cumulative.ndjson` yields **900 + the second series**, never 1400. Keying on `type` alone would collapse those series and undercount; ordering by file position would repeat W2's file-order bug.

## 3. Signal mapping — exact

### 3.1 Tokens (`extractTokenCounts`)

Walk `resourceMetrics[].scopeMetrics[].metrics[]` where `name == otelTokenUsageMetric`. Counters: `seen` (name matched, **including** `Sum == nil`), `points` (data points visited), `read` (yielded both a recognised `type` and a value).

| attribute `type` | `TokenCounts` field | output JSON key (unchanged) |
|---|---|---|
| `input` | `Input` | `input` |
| `output` | `Output` | `output` |
| `cacheRead` | `CacheRead` | `cache_read` |
| `cacheCreation` | `CacheCreation` | `cache_creation` |

camelCase attribute values, snake_case output keys — the rename to `cache_write` stays forbidden by the mandate. `token_type` is gone; the key is `type`, **read only on a metric whose name is `claude_code.token.usage`** — `type` is overloaded across Claude Code's metrics, so a second metric carrying it contributes nothing (nit 2, pinned in `tokens_only.json`).

**D4 — delete the `reasoning`/`reasoning_output` case (`:210`), keep the `TokenCounts.Reasoning` field.** Research §6.3 proves the type does not exist in Claude Code's surface, so the case is dead code against invented data. The field is harness-agnostic and `omitempty`, so leaving it unpopulated removes the key entirely — **no profile/v1 output change**. Add the comment at **`types.go:86`** (the `Reasoning` line; v1 cited `:98`, which is `ActivationEntry.SkillName`).

### 3.2 Tool calls (`extractToolCalls`)

- **`claude_code.tool_result`** — emitted only after a tool actually runs, never for a rejected call. Yields `tool_name` and `success` (documented as the string `"true"`/`"false"`; read through `attrs.Bool`, which also accepts a real `boolValue`). This is the **execution outcome**.
- **`claude_code.tool_decision`** — the **permission** decision, `decision` ∈ `accept`/`reject`.

**D5 — decided, not proposed.** Per `mandate.md` item-19 decisions block (chief of staff, 2026-09-13 16:12): `ToolCallEntry.Success` means "the tool ran and succeeded", sourced from `tool_result`; rejected calls are listed from `tool_decision` with `success: false`; spec, RELEASE_NOTES and the already-landed `AdapterVersion` bump carry the semantic change; no JSON key changes and profile/v1 stays. There is no manager-ack branch and no fallback shape — v1's §3.2/R3 ack request and its "tool_result only" fallback are removed.

**Entry rule:** one entry per `tool_result` carrying **both** a readable `tool_name` and a readable `success`; plus one entry per `tool_decision` whose `decision == "reject"` and which carries a `tool_name`.

**No `tool_use_id` de-duplication, and the reason is disjointness, not a claim about accepts** (REAL-8). Accepted calls are never listed from `tool_decision` — an accept's outcome is read from its `tool_result` — and a rejected call never runs, so it never produces a `tool_result`. The two entry sources therefore cannot describe the same call. Do **not** write "accepts always produce a `tool_result`": an export captured mid-run has accepts whose results have not been emitted yet, and those calls are simply not listed (the reason at `calls == 0` says exactly that). `accept_then_result.json` is the fixture that goes RED if the disjointness is ever false.

Do not add a second predicate on `source ∈ {user_abort, user_reject}`; `decision` already carries it. Sort by timestamp ascending; entries with no readable timestamp sort **last**, in file order among themselves, carrying `timestamp: ""` (§4 rule 4). Add `otelToolResultLog = "claude_code.tool_result"` beside the three constants at `:13-17`.

### 3.3 Timing (`extractTiming`)

**D6 — the span stays `claude_code.api_request`-scoped: min/max `timeUnixNano` over api_request records.** Not min/max over all log records (research §4.2's recommendation): that would make `timing: otel` true for a file of startup events only, and would change `total_ms`'s meaning between a 0.4.0 and a 0.4.1 profile — out of bounds for "make 0.4.0's promises true". Not a sum of `duration_ms` either: it measures time in model calls, a different quantity, and `TimingData` has no field for it; adding one is a profile/v1 schema change the mandate forbids. `active_time.total` likewise. Both recorded for 0.5.0. The cost is honesty, paid in prose: the spec must say the span excludes the prompt before the first request and any tool activity after the last.

`StartTime`/`EndTime` are `t.Format(time.RFC3339Nano)` of the parsed nanosecond values — the output field is a timestamp string and the raw nanosecond digits must not leak into the profile. `TotalMs = last.Sub(first).Milliseconds()`, min/max by time, so `total_ms >= 0` and `end >= start` hold however the exporter ordered records (W2, unchanged). A single api_request yields `total_ms: 0` and is still `present` — a zero-length span is a value that was read (R-F6).

### 3.4 Event identification

`body.stringValue` is fully qualified (`"claude_code.tool_result"`); the `event.name` attribute is the short name (`"tool_result"`). Prefer the body — it is the primary identity and survives attribute cardinality limits. `Body` is `json.RawMessage`, so all three observed shapes are handled without failing the record: an object with `stringValue`, a bare JSON string, or absent.

```go
func eventName(r otlpLogRecord) string {
    n, ok := bodyName(r.Body)                       // object{stringValue} → that; bare string → itself; else ""
    if !ok || n == "" { n, _ = r.Attributes.String("event.name") }
    if n == "" { return "" }                        // unknown record, ignored, never an error
    return "claude_code." + strings.TrimPrefix(n, "claude_code.")
}
```

Normalising **to the fully-qualified form** keeps the existing constants and reason strings greppable by a user against their own export.

## 4. Failure classification and the reason for each input shape

**The line, and it must be stated in the spec (REAL-6/7):** `error` means **the file could not be read as an OTLP/JSON export** — unreadable, empty, not a JSON object at top level, malformed JSON, or a value that does not fit the OTLP schema. `unknown` means **a well-formed OTLP/JSON export that carries no telemetry** (for that signal, or for all three). No reason string ever interpolates a Go type name: reasons are built from the error's *type*, never from `err.Error()` where that message names Go types.

**File-level (all three signals settle the same way, at `resolve`):**

| Input shape | State | Reason |
|---|---|---|
| no file configured | `unknown` | `"OTel export not configured. Provide an OTel export file via --otel-file or OtelExportFile."` (unchanged) |
| open fails | `error` | `"failed to read OTel export file: %v"` (unchanged) |
| zero bytes or whitespace only | `error` | `"OTel export file is empty"` |
| top level is not an object | `error` | `"top-level JSON value is a %s; an OTLP export is a JSON object (one ExportMetricsServiceRequest or ExportLogsServiceRequest per object)"` |
| `*json.SyntaxError` / unexpected EOF at batch *n* | `error` | `"malformed JSON at byte %d in batch %d: %s"` (`se.Error()` names characters, not Go types; offset from `se.Offset`, or `dec.InputOffset()` for an unexpected EOF). **Append** `" No data from earlier batches was used; if the collector was stopped mid-write, delete the final partial line and retry."` **only when batch index > 1** |
| `*json.UnmarshalTypeError` at batch *n* | `error` | `"OTel export does not fit the OTLP schema: value at %s in batch %d is not the expected type"`, path from `te.Struct`+`te.Field`, or `"a value in batch %d is not the expected type"` when `te.Field` is empty. **Never `te.Error()`** — it prints Go type names (nit 13) |
| parses, no `resourceMetrics` and no `resourceLogs` in any batch | `unknown` | `"OTel export is not OTLP/JSON: no resourceMetrics or resourceLogs found. Claude Code emits OTLP via OTEL_EXPORTER_OTLP_PROTOCOL=http/json; see README 'Capturing an OTel export'."` |

**Per-signal (each extractor, from its own counters — every reason is true of the input that produced it):**

| Signal | Counter state | Reason |
|---|---|---|
| tokens | `seen == 0` | `"no claude_code.token.usage metric found in OTel export"` |
| tokens | `seen > 0, points == 0` | `"no readable claude_code.token.usage metric in OTel export: it carried no sum data points"` (REAL-4: no asserted cause — a gauge, a histogram and an empty sum are indistinguishable here) |
| tokens | `points > 0, read == 0` | `"no readable claude_code.token.usage metric in OTel export: no data point carried both a recognised type attribute (input/output/cacheRead/cacheCreation) and a numeric asDouble or asInt value"` |
| tool calls | `seen == 0` | `"no claude_code.tool_result or claude_code.tool_decision log events found in OTel export"` |
| tool calls | `seen > 0`, `calls == 0` | `"no tool call outcomes in OTel export: "` + the composed clause list below |
| timing | `seen == 0` | `"no claude_code.api_request log events found in OTel export"` |
| timing | `seen > 0`, none timed | `"no readable claude_code.api_request log events in OTel export: none carried a parseable timeUnixNano"` |

**D8 — the tool-call `calls == 0` reason is composed from counters, not selected by a branch chain** (closes REAL-2/3/8 as a class). Counters: `seen`, `unnamed`, `unreadableOutcome`, `accepts`, `undecided`. Per record, count **at most one** defect, name first then outcome, so no record is counted twice. The reason is the prefix plus `strings.Join(clauses, "; ")`, appending one clause for each non-zero counter in this fixed order:

1. `"%d claude_code.tool_result events carried no tool_name"` (`unnamed`)
2. `"%d claude_code.tool_result events carried no readable success value"` (`unreadableOutcome`)
3. `"%d claude_code.tool_decision events were accepts, and only claude_code.tool_result reports an outcome — the export may have been captured before those tools completed"` (`accepts`)
4. `"%d claude_code.tool_decision events carried no recognised decision"` (`undecided`)

_Rejected:_ v2's row-per-combination table — with four independent defects it needs 15 rows, and REAL-2/3 were exactly the rows it was missing. A clause list has one row per counter and cannot go stale when a counter is added.

Four rules that fall out and must be pinned in the spec:
1. A metric whose name matches but whose `Sum` is nil increments `seen`, not `points` — so `gauge_not_sum.json` cannot produce either of the other two token reasons.
2. **An unreadable outcome is never assimilated and never serialises as `false`.** A `tool_result` with no readable `success` produces no entry; it is counted and named, not recorded as a failed call.
3. The `%d` counts come from the same counters the branch tests, so a reason cannot state a number the walk did not observe.
4. **A tool call with no readable `timeUnixNano` is kept**, with `timestamp: ""`, sorted last. The call was read; only its timestamp was not, and dropping it would undercount tool calls over a formatting defect. `ToolCallEntry.Timestamp` has no `omitempty`, so `""` is what serialises. _Rejected:_ dropping the entry and counting it in the reason — it makes `tool_calls` lie about how many calls the session made.

## 5. Test plan (RED first)

Fixtures as **files** under `profiler/testdata/otlp/` — an OTLP object is 50–100 lines and inline constants stop being reviewable. Add `profiler/testdata/otlp/README.md` listing each fixture, its purpose, and which fields are [OBSERVED] vs inferred (JSON has no comments). Point the adapter at the testdata path read-only; keep `t.TempDir()` only for the unconfigured/missing-file cases. Two things are pinned by **direct unit tests on `otlp.go`, not fixtures**: the BOM strip (invisible in a reviewed text file) and the int64→int clamp.

| Fixture | Pins (read path) | RED at `527ba4e` |
|---|---|---|
| `full_export.ndjson` | metrics batch + logs batch in one file; all four token types; tool_result, tool_decision(reject), 2×api_request. All three `present` with exact values | whole-file `Unmarshal` fails on concatenated objects → all three `error` |
| `tokens_only.json` | **two `resourceMetrics` entries, each with two `scopeMetrics`** (nit 1), four token types spread across them; plus a **second metric carrying its own `type` attribute** that must contribute nothing (nit 2); tool_calls/timing `unknown` with their own reasons | all three `unknown`, tokens reason is "no …metric found" |
| `tool_calls_only.json` | 3 tool_results: `body.stringValue`, bare-string `body`, and body-absent + `event.name` — the three `eventName` paths (closes A3) | all three `unknown` |
| `timing_only.json` | single api_request → `total_ms: 0`, still `present` | all three `unknown` |
| `multi_batch_delta.ndjson` | 3 batches, `type=input` 100/200/300, temporality `1` → **600** | `error` (concatenated) |
| `cumulative.ndjson` | one series 500 then 900 with temporality `2`, a second series on a different `model`, and one batch declaring `"2"` as a string (closes A2) → **900 + second series**, not 1400 | `error` (concatenated) |
| `number_string_variants.json` | `"asInt": "1523"` on one token point and **`"asInt": 1523` as a bare number** on another (nit 3); `success` as a real `boolValue`; `timeUnixNano` as an **unquoted number** on an api_request (closes A1) | all three `unknown` |
| `as_double_rounding.json` | `"asDouble": 1522.7` → **1523** (round-half-away-from-zero via `math.Round`; truncation gives 1522, closes B4); **plus one unreadable point (`"asDouble": "n/a"`) among the readable ones → `present` with the readable total only, and no reason on a present result** (REAL-5) | tokens `unknown` |
| `gauge_not_sum.json` | `token.usage` as `gauge` → tokens `unknown` with the **no-sum-data-points** reason, no panic (closes D2) | `unknown` with the wrong reason ("no …metric found") |
| `accept_then_result.json` | accept `tool_decision` + matching `tool_result` on one `tool_use_id` → exactly **one** entry, `success: true` (closes G1; the evidence for source disjointness, §3.2) | all three `unknown` |
| `tool_failure.json` | `tool_result` with `success: "false"` → one entry, `success: false`; **plus a second `tool_result` with no `success` key → no second entry** (REAL-3: an unreadable outcome never serialises as `false`) | all three `unknown` |
| `unreadable_tool_events.json` | one `tool_result` with no `tool_name`, one with no `success`, one `tool_decision` with `decision: "maybe"` → `calls == 0`, `unknown`, reason carries **all three clauses with count 1 each** in D8's order (REAL-2/3) | all three `unknown`, reason is round 1's wording |
| `accepts_no_results.json` | only accepted `tool_decision`s → `unknown` with the *accepts* clause only, **not** the unnamed clause (closes D1) | `unknown` with the wrong reason |
| `no_timestamp_tool_call.json` | one timed tool_result + one with no `timeUnixNano` → 2 entries, the untimed one last with `timestamp: ""` (closes D4) | all three `unknown` |
| `skill_name_present.json` | `skill.name: "my-skill"` **verbatim** on `token.usage` (a user-defined skill name is *not* redacted — research §3.4) and `"third-party"` on `api_request` → tokens/timing `present`, `skill_activation` still `unknown` **with item 11's new reason** (REAL-1; the mandate boundary) | all three `unknown`; skill_activation reason is round 1's wording |
| `out_of_order.ndjson` | descending api_request timestamps across batches → `total_ms >= 0` | `error` (concatenated) |
| `unknown_events.json` | only `hook_registered`/`user_prompt` → all `unknown`, **not** `error` | all three `unknown` (right state, right-ish reason — assert the reason text) |
| `malformed.json` | truncated mid-object → all three `error`, reason contains "malformed JSON at byte" and **no** mid-write advice (single batch) | `error` (passes today; keep it honest by asserting the new text) |
| `truncated_final_line.ndjson` | 3 whole batches + a partial 4th line → all three `error`, reason names **batch 4** (1-based) and a byte offset and appends the mid-write advice (closes I1, REAL-9) | `error` with `"invalid character '{' after top-level value"` — no batch, no offset |
| `type_mismatch.json` | `"resourceMetrics": {"…": …}` (object where the schema wants an array) → all three `error`, reason is the schema sentence naming a JSON path, and **contains no Go type name** — assert `!strings.Contains(reason, "profiler.")` and `!strings.Contains(reason, "Go struct")` (nit 13) | `error` leaking `profiler.claudeCodeOtelExport` |
| `top_level_array.json` | `[{…}]` → all three `error`, reason is `"top-level JSON value is an array; …"` (closes D5, REAL-6) | `error` leaking `profiler.claudeCodeOtelExport` |
| `empty.json` | **zero bytes** → all three `error`, reason `"OTel export file is empty"` (closes G4/D3) | `error` `"unexpected end of JSON input"` |
| `no_envelope.json` | `{}` — v1's `empty.json`, renamed → all three `unknown`, reason names OTLP/JSON | `unknown` with "no …metric found" |
| `bespoke_envelope.json` | the old `{"metrics":[…],"logs":[…]}` verbatim → all three `unknown`, reason **names the expected format**; assert the reason is not the generic one | tokens **`present`** — the inverted truth this release exists to fix |

**The RED criterion is per fixture, on (state, reason substring) — not "every OTLP fixture → unknown", which is false for the `.ndjson` fixtures.** Record the actual output in `status.md`.

Tests to rewrite (input only; assertions keep their meaning): `TestCapabilityReport_ClaudeCode_WithOtel` (`:90-102`), `TestCapabilityReport_ClaudeCode_EmptyOtelFile` (`:140`, repoint at `no_envelope.json`), `TestCapture_ClaudeCode_WithOtelData` (`:160-176`, drop the `reasoning` assertion at `:202-204`), **`TestProfileRoundTrip` (`:289-300`)** — repoint at `full_export.ndjson` **and add an assertion that at least one signal is `present`**, or it goes vacuously green after C2 (closes G5) — `TestCaptureDeliversEverySignalProbeAdvertises` (`:461`, the fragment constants at `:449-458` become fixture filenames and `export string` becomes `fixture string`; it keeps iterating `report.Capabilities` as the denominator, W3), `TestCapture_UnusableExport_IsDiagnosable` (`:597`), `TestTimingIsASpanNotFileOrder` (`:646`), `TestTokens_OnlyReadableMetricsAreCounted` (`:684`), `TestCapture_ClaudeCode_PartialExport_KeepsToolCallsAndTiming` (`:707`).

## 6. Commits, sequencing, verification

| # | Commit | Files | Est. lines |
|---|---|---|---|
| C1 | `test(profiler): pin the adapter against real OTLP/JSON exports` **[RED]** | `profiler/testdata/otlp/*` (23 + README), `profiler/profiler_test.go`, new `profiler/otlp_test.go` (BOM, clamp, reason-kind units) | +1080 / −140 |
| C2 | `fix(profiler): parse OTLP/JSON, not an envelope nothing emits` **[GREEN]** | new `profiler/otlp.go`, `profiler/claude_code.go`, `profiler/types.go` (one comment), **`profiler/cmd/main.go`** (R-F4) | +360 / −130 |
| C3 | `docs: describe the OTel capture route and the format the adapter reads` | `README.md`, `docs/profiler-spec.md` | +115 / −25 |
| C4 | `docs(changelog): record the OTLP parser and correct three stale claims` | `CHANGELOG.md`, `RELEASE_NOTES.md` | +18 / −6 |

**C1 and C2 are pushed together in one push.** CI runs on the pushed head, and a test-only push is red by design. Commit them separately (the RED must exist in history with its recorded output) but do not push between them. ≈ +1570 / −300, over half of it fixtures. Each commit ends `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>`.

**R-F3/R-F4 ride in C2** (REAL-13): `CaptureOpts.ExportFile` must not be silently ignored one layer below the CLI guard. Take the smaller honest option — `ClaudeCodeAdapter.Capture` returns the same loud error the CLI does when `opts.ExportFile != ""` (`"--export-file is not read by the claude_code adapter; supply an OTel export with --otel-file"`), so the adapter, not the CLI, owns the contract; keep the field on `CaptureOpts` for the future ATIF adapter. Delete the provably-dead `ExportFile: *exportFile` at `cmd/main.go:83`. One new test asserting the library-level refusal; state the refusal in the spec (§7 item 9).

Verification before push, from `profiler/`: `go build ./... && go vet ./... && gofmt -l . && go test ./...`, then the four shell suites, then **`BIN=$(mktemp -d)/profiler && go build -o "$BIN" ./cmd`** (nit 7 — never `profiler/profiler` inside the repo) and `"$BIN" capture --harness claude_code --session s1 --snapshot sha1 --skill-dir ../skills/skill-audit --otel-file testdata/otlp/full_export.ndjson` showing three `present` signals with real values, pasted into `status.md`.

**Recount (REAL-10/12, R-F5), run from `profiler/` at the C4 head:**
- test **functions**: `grep -c '^func Test' *_test.go cmd/*_test.go` prints one count per file — **sum them** (it does not print a total); cross-check with `go test ./... -list '.*' | grep -c '^Test'`. At `527ba4e` this is 20 + 2 = 22.
- **subtests**: `go test ./... -v 2>&1 | grep -c '^=== RUN .*/'` — the `/` restricts the count to subtests, so parents are not double-counted (R-F5: "42 cases" double-counted two container functions; distinct cases are 40).
- State the two numbers **separately and labelled** in both docs — functions (split profiler/CLI) and subtests. **Do not restate a conflated "N cases" figure.** Both sites are §7 item 15.

## 7. Documentation — every sentence that goes false

Edit by quoted content; the line is a hint at `527ba4e`, **re-verified by grep for v3**. Sentences explicitly **not** changed are listed too, so the implementer does not churn them.

1. **`README.md` (:65)** — `| `tool_calls` | `claude_code.tool_decision` events carrying a tool name |` → `claude_code.tool_result` events carrying a tool name and a readable outcome, plus rejected `claude_code.tool_decision` events.
2. **`README.md` (:64)** — `| `tokens` | a `claude_code.token.usage` metric with a recognised token type and a numeric value |` → name the attribute `type` and its four values.
3. **`README.md` (:66)** — `| `timing` | `claude_code.api_request` events carrying a parseable timestamp |` — **no change**; it names no format.
4. **`README.md` (:71-74)** — "An export that is missing, unreadable, or malformed is not "unconfigured": all three OTel signals come back `error` naming the failure, so a file you supplied but the profiler cannot use says so." Incomplete once §4 adds the `unknown` shape: add §4's line — `error` is "could not be read as an OTLP/JSON export" (empty, non-object top level, malformed, schema mismatch included), `unknown` is "parsed fine, carried no telemetry", including a file with no `resourceMetrics`/`resourceLogs`.
5. **`README.md` (:75-76)** — "`skill_activation` and `attribution` are always `none`: Claude Code emits no skill activation event, and this adapter does not read the skill-level attributes it does emit." Rewrite to match **item 11's** reasons verbatim. **It must not claim user-defined skill names are redacted** (REAL-1): research §3.4 quotes the docs — built-in, bundled, **user-defined**, and official-marketplace skill names appear verbatim on `skill.name`; only *third-party plugin* skill names become `"third-party"`; `skill.name` on these signals is **not** gated by `OTEL_LOG_TOOL_DETAILS`. The `"custom_skill"` placeholder is the *Skill-tool span* attribute, a different surface this adapter never reads. (Research §4.3's bullet says "user-defined … are redacted"; it is inconsistent with §3.4's verbatim doc quote in the same document, and §3.4 governs. Flagged in §10.)
6. **`README.md`, new subsection "Capturing an OTel export", inserted after the skill sentences at :76** (nit 10), before the `--export-file` paragraph. Claude Code has no file exporter — say that first. Then:
   - **Route (a), primary, observed:** the env block — `CLAUDE_CODE_ENABLE_TELEMETRY=1`, `OTEL_METRICS_EXPORTER=otlp`, `OTEL_LOGS_EXPORTER=otlp`, `OTEL_EXPORTER_OTLP_PROTOCOL=http/json`, `OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4318`, `OTEL_METRIC_EXPORT_INTERVAL=2000`, `OTEL_LOGS_EXPORT_INTERVAL=1000`, `OTEL_LOG_TOOL_DETAILS=1` — with the last one described as **"exposes tool parameters on `tool_result`/`tool_decision` events; this adapter does not read them"** and **nothing about skill-name redaction** (REAL-1) — and a ≤20-line `python3 http.server` snippet that appends each POST body as one line, giving NDJSON.
   - **Route (b), alternative:** collector with `receivers: otlp/protocols/http` and **one shared `file` exporter** — `format: json`, **`append: true`**, no compression — referenced by both the metrics and the logs pipeline. `append` defaults to `false` and would truncate the capture on every collector start; with compression on, the output is length-prefixed binary, not line JSON. One exporter means one file and nothing to concatenate, so v1's "concatenate the two pipeline files" sentence is dropped (the reader still accepts concatenated objects, which is why both routes share a parser).
   - **rule out the `console` exporter explicitly** — `console.dir` output is not valid JSON (unquoted keys, bare `undefined`); users will otherwise try it after Anthropic's debugging tip. A truncated final line means the collector was stopped mid-write: restart it cleanly, or drop the partial last line. Real captures contain `user.email`, `organization.id`, `session.id` — scrub before pasting into an issue. A bundled receiver / `otlp-capture` subcommand is 0.5.0; `--otel-file` remains the only input.
   - _Decision:_ the route (a) snippet stays **inline in the README**, not shipped as `profiler/scripts/otlp-capture.py` — the mandate defers a bundled receiver to 0.5.0, and a shipped script is new surface in a patch release. _Rejected:_ ship it now (scope); document route (b) only (v1's choice — it puts the unobserved route first).
7. **`docs/profiler-spec.md`, new "Input format" subsection under "Claude Code adapter" (after the Telemetry-surface line at :203).** One `Export*ServiceRequest` per object; NDJSON or concatenated accepted, non-object top level not; `asDouble`→`asInt`, number-or-string, rounded; delta sums, cumulative last-by-time per series, **temporality that does not read as DELTA is treated as cumulative**; `body.stringValue` vs `event.name`; §4's `error`-vs-`unknown` line and its file-level table; §4's four rules; **"data points or log records that cannot be read are skipped; the profile does not report how many (0.5.0)"** (REAL-5); and D6's timing-span caveat.
8. **`docs/profiler-spec.md` probe logic (:207-209)** — "a `claude_code.token.usage` metric carrying a recognised `token_type` and a numeric value" (→ `type`, four values, `asDouble`/`asInt`); "`claude_code.tool_decision` log events carrying a `tool_name`" (→ `tool_result` + rejects); "`claude_code.api_request` log events carrying a parseable RFC 3339 timestamp" (→ parseable `timeUnixNano`).
9. **`docs/profiler-spec.md` capture logic (:215-216)** — step 2 "summing only those carrying a recognised `token_type` and a numeric value" (→ `type`, plus the temporality rule); step 3 in full, including "`ToolCallEntry.Success` records the permission decision (`decision == "approved"`), not the execution outcome — the adapter reads no completion signal. Separating the two is a schema change tracked for 0.5.0." → D5's meaning, plus §4 rule 2 (an unreadable outcome is not an entry). Add one sentence for R-F3: a supplied `CaptureOpts.ExportFile` is refused by the adapter, not silently ignored. Step 4 (:217) is **unchanged**; it already says "earliest to the latest parseable timestamp".
10. **`docs/profiler-spec.md` steps 6 and 7 (:219-220)** quote the two reason strings verbatim — "Claude Code has no skill-level activation events" and "this adapter does not read Claude Code's skill attribution attributes" — so they go false with item 11 and must change with it.
11. **`profiler/claude_code.go`** — the adapter doc comment (:22-24) "This adapter reads from a file the OTel collector writes (or a file-based exporter), not a live network endpoint" names no format: name OTLP/JSON and point at the README section. Rewrite :26-29 and the two reasons at :160-161 to (REAL-1, R-F7):
    - skill_activation: `"Claude Code emits no skill activation event; this adapter does not yet read skill.name, which marks the skill active for a request on token.usage, cost.usage and api_request (third-party plugin skills appear as \"third-party\"). Reading it is 0.5.0."`
    - attribution: `"Claude Code telemetry carries no output-to-skill mapping"`
12. **`docs/profiler-spec.md` Fallback (:224)** — "A file that is configured but missing, unreadable, or malformed is a different case and must not borrow that reason: those three are `error`, naming the read or parse failure" → extend to §4's `error` set (empty, non-object top level, malformed, schema mismatch) and the no-envelope case as `unknown`. AC2 (:229) is **unchanged**.
13. **`docs/profiler-spec.md` AC3 (:230)** — rewrite in full: it names `token_type`, `tool_decision`, and "an `api_request` with no parseable timestamp", and its "empty" now means two different files. New text covers a `token.usage` with an unrecognised `type` or no readable value, a `tool_result` with no `tool_name` **or no readable `success`**, a `tool_decision` with no recognised `decision`, an `api_request` with no parseable `timeUnixNano`, and a file with no OTLP envelope.
14. **`docs/profiler-spec.md` AC11 (:238)** — "An export file that is supplied but cannot be read or parsed yields `error`…" → extend with the empty, non-object-top-level and schema-mismatch shapes. **Add AC12:** a file that parses but carries no `resourceMetrics`/`resourceLogs` is `unknown` for all three, naming the expected format — never `error`, and never a silent zero. **Amend AC10 (R-F6):** "never `present` with a zero value" → *never `present` without a value read from the export; a read zero is a value, and a single `api_request` yields `total_ms: 0` and is `present`.*
15. **`CHANGELOG.md` (:36)** — "This release has **22** test functions (20 in the profiler package, 2 in the CLI), 42 cases counting subtests." and **`RELEASE_NOTES.md` (:17)** — "22 profiler test functions pass, 42 cases counting subtests." Both stale after C1. Recount per §6 and state functions and subtests as two labelled numbers at both sites; no "N cases".
16. **`RELEASE_NOTES.md` (:24)** — delete "(including reasoning tokens)" from "Claude Code adapter reads OTel export data for token counts (including reasoning tokens), tool calls, and timing." It is the only "reasoning tokens" claim in the docs.
17. **`CHANGELOG.md` 0.4.1 "Fixed"** — one bullet: the adapter read a bespoke envelope nothing ever emitted (`token_type`, `cache_read`, `reasoning`, `decision: "approved"`), so a real export yielded all-`unknown` and only a hand-written file yielded values; it now parses OTLP/JSON. Plus: `Success` now means execution outcome (D5); the reasoning token type does not exist and is no longer populated; `CaptureOpts.ExportFile` is refused by the adapter, not only by the CLI (R-F3). **The 0.4.1 "Fixed" bullets already landed by round 1 stay exactly as written — they are true as history of what round 1 repaired** — and the new bullet carries the one-line note that the `token_type`/`decision` behaviour those bullets describe belonged to the bespoke envelope this release replaced (nit 11). `CHANGELOG.md:38` (the softened skill-level sentence) and `RELEASE_NOTES.md:8` are **unchanged** — both stay true under item 11's new reasons. **`RELEASE_NOTES.md` v0.4.1** — one bullet in the same voice plus the capture-route pointer.
18. **`profiler/types.go:86`** — comment on `Reasoning` stating Claude Code emits no reasoning token type (D4).
19. **`docs/profiler-spec.md` (:67)** — found in the v3 re-grep, missed by v2: the `State == "unknown"` bullet quotes **both** reason strings verbatim ("no `claude_code.token.usage` metric found in OTel export", "Claude Code has no skill-level activation events"). The first stays true; the second goes false with item 11 and must be updated with it.
20. **`CHANGELOG.md` / `RELEASE_NOTES.md` "verified live" (R-F2)** — the claim covers the *local* install route only; the marketplace route cannot work until `.claude-plugin/marketplace.json` is on `main`. Reword so the verified claim names what was exercised and the remote route reads "works once this release is on `main`".

## 8. Risks, and what pins each

| # | Risk | Pin |
|---|---|---|
| R1 | **Cumulative temporality is untested against real output.** Delta is the documented default and the only thing observed; the cumulative branch exists on spec reading alone — and nit 6 now routes *unreadable* temporality down it too. Most likely thing to need rework. | `cumulative.ndjson` (two series, integer and string temporality) + `multi_batch_delta.ndjson` (explicit `1` still sums). If it is wrong it is wrong in a fixture, not silently in a user's numbers. |
| R2 | **`tool_result`/`tool_decision` attributes are [DOCS], not [OBSERVED]** — the live capture could not authenticate, so no tool event was seen on the wire. `success` as a string is documented, not observed. | `number_string_variants.json` (real `boolValue`) and `tool_failure.json` (`"false"` string) read the same field two ways. A version that renames `success` fails loudly as `unknown` with a counted reason (D8 clause 2), never as a silent `false`. |
| R3 | **`ToolCallEntry.Success` changes meaning** (D5) between a 0.4.0 and a 0.4.1 profile with no key change to signal it. Pinned by prose, not by a test — this is the one that bites a consumer silently. | `tool_failure.json` + `accept_then_result.json` pin the new meaning; the spec sentence, the CHANGELOG bullet, and `capability.adapter_version` (already `0.4.1`) make the two distinguishable. Decided in the mandate; no ack outstanding. |
| R4 | **A truncated final NDJSON line fails the whole capture.** The most likely real-world complaint: a collector killed mid-write costs the user every earlier batch. | `truncated_final_line.ndjson` — the reason names batch and offset and tells the user what to do. The alternative, silently using the batches before the break, is W5's swallowed error returning; refused. |
| R5 | `token.usage` as `asDouble` is [BINARY]-generalised from `session.count`, not observed on the token metric itself. | Moot by tolerance: `asDouble`→`asInt`→number-or-string, pinned by `number_string_variants.json` and `as_double_rounding.json`. |
| R6 | **No live end-to-end run is possible** (OAuth expired), so item 19's DoD "a realistic OTLP/JSON export yields present values" is met by fixtures derived from observed envelopes, not by a confirming run. Route (b) was read from the fileexporter source, not run. | Stated as such in `status.md`. **Rule:** attempt route (b) end to end only if `otelcol-contrib` is already installed or installs without a permission prompt; otherwise mark the README snippet "derived from the fileexporter README and source; run route (a) to verify locally" and record in `status.md` that no end-to-end run happened. Route (a) is documented as primary because it was observed. |
| R7 | Concurrent writes to the same files from any late round-1 fix. | §0.1. Round 1 is landed through `527ba4e`; if the head moves, rebase, never merge — round 1 owns doc *sentences*, item 19 owns the *format facts* inside them, and item 19 owns test inputs while round 1 owns test assertions. |

## 9. Explicitly out of scope

Bundled `otlp-capture` receiver subcommand or shipped script; `skill.name` → `skill_activation` promotion; per-skill token attribution; reading `tool_parameters`/`skill_name`; `active_time.total` and `duration_ms` surfacing (no profile/v1 field); `cache_creation` → `cache_write`; repeatable `--otel-file`; session filtering by `session.id`; recording `service.version`; reporting a count of skipped records. All 0.5.0. If any looks necessary mid-build, stop and record it under "found, not fixed" rather than widening the release.

## 10. Disagreements with the gate decisions (planned to as decided anyway)

- **A truncated final line as a whole-file `error` (I1).** Planned as decided, and the reason is good, but I expect this to be the first post-release complaint: the user's three complete batches are discarded for a defect in bytes the producer never finished writing. The consistent alternative is to keep the `error` state and reason while also returning the earlier batches' values — which round 1 correctly forbids, because a swallowed parse error is W5. If it bites, the right repair is a separate explicit affordance (`--allow-truncated`), not a softening of the classifier. Recorded for 0.5.0, not built.
- **Unreadable temporality → cumulative (nit 6).** Planned as decided. Note that it is not a "safe" default so much as a choice of which silent error to take: a delta series whose temporality field is lost now *undercounts* (100/200/300 → 300, not 600) exactly as silently as the overcount it prevents. The loud option, consistent with round 1's assimilated-value rule, is to not assimilate a data point whose temporality cannot be read and to name it in the reason — no silent number in either direction. Recorded for 0.5.0.
- **A research inconsistency the implementer will hit (REAL-1).** `otlp-research.md` §3.4 (verbatim doc quote) and §4.3 (bullet 2) contradict each other on whether *user-defined* skill names are redacted on `skill.name`. §3.4 governs: they are not; only third-party plugin skills are. Every reason string and doc sentence in this plan follows §3.4. If a later live run shows otherwise, item 11's reasons change — nothing else in the plan depends on it.
- **Everything else** — round-1 roots A–I, REAL-2…13, nits 1–13 — I agree with as decided.
