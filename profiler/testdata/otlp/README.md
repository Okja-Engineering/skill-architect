# OTLP/JSON fixtures for the Claude Code adapter

Every file here is an input to `ClaudeCodeAdapter`, read through `--otel-file`.
They are real OTLP/JSON: one `ExportMetricsServiceRequest` or
`ExportLogsServiceRequest` object per JSON object, exactly what a local OTLP
receiver (README route (a)) or the collector's `file` exporter (route (b))
writes. `.json` files hold a single object, pretty-printed for review; `.ndjson`
files hold one object per line, which is the framing both capture routes
produce.

JSON has no comments, so what is real and what is constructed is recorded here.

**[OBSERVED]** — taken from a live Claude Code v2.1.221 capture: the envelope
(`resourceMetrics` → `scopeMetrics` → `metrics` → `sum.dataPoints`,
`resourceLogs` → `scopeLogs` → `logRecords`), `asDouble` as a JSON number,
`"aggregationTemporality": 1`, `isMonotonic`, nanosecond timestamps as decimal
strings, `body.stringValue` carrying the fully-qualified event name while the
`event.name` attribute carries the short one, `intValue` as a bare number,
scope names, and the resource attribute set.

**[DOCS]** — from Anthropic's monitoring reference: the attribute names and
values (`type` ∈ `input`/`output`/`cacheRead`/`cacheCreation`, `tool_name`,
`success` as the string `"true"`/`"false"`, `decision` ∈ `accept`/`reject`,
`skill.name`).

**Constructed** — the specific numbers, ids, model and tool names, and every
malformed or degraded shape. Those exist to pin a failure path; a producer
emitting them would be unusual or broken, which is the point. Where a fixture
contains something no producer emits, it is called out below.

Session ids and user ids here are placeholders. A real capture carries
`user.email`, `organization.id` and `session.id`; scrub before sharing.

| Fixture | What it pins |
|---|---|
| `full_export.ndjson` | The happy path in the shape a capture actually has: a metrics batch and a logs batch concatenated as NDJSON. All four token types, a `tool_result`, a rejected `tool_decision`, two `api_request`s. All three signals `present`. |
| `tokens_only.json` | Two `resourceMetrics` entries with two `scopeMetrics` each — legal, and a parser that indexes `[0]` loses three quarters of the file. Also two non-token metrics carrying their own `type` attribute: `claude_code.lines_of_code.count` here carries `type: "output"`, which it never does in reality, so that if the `type` lookup were not scoped to `claude_code.token.usage` the output count would be wrong by 999999. |
| `tool_calls_only.json` | The three ways a record names its event: `body.stringValue`, a bare-string `body`, and no `body` at all with only the `event.name` attribute. |
| `partial_no_tokens.json` | A logs-only capture: tool calls and timing are `present`, tokens `unknown` with its own reason. A partial export never costs the caller the signals it does carry. |
| `timing_only.json` | A single `api_request`: a zero-length span is a value that was read, so `total_ms: 0` is `present`, not `unknown`. |
| `multi_batch_delta.ndjson` | Delta temporality across three batches: 100 + 200 + 300 = 600. The default temporality, so this is the common case. |
| `cumulative.ndjson` | Cumulative temporality: one series reports 500 then 900 (last by `timeUnixNano` wins, not 1400), a second series on a different `model` adds its own 250, and one batch declares the enum as the string `"2"`. |
| `array_attribute_series.ndjson` | Two cumulative series distinguished only by an `arrayValue` attribute. OTel identifies a series by its whole attribute set, so these are two running totals that add to 300 — not one series whose later point supersedes the earlier and reports 200. |
| `temporality_enum_names.json` | `aggregationTemporality` as the proto enum *name* rather than the integer — what a producer going through the standard protobuf JSON mapping emits. `AGGREGATION_TEMPORALITY_DELTA` and `AGGREGATION_TEMPORALITY_CUMULATIVE`, one metric each. |
| `number_string_variants.json` | The spec's "either numbers or strings are accepted" for 64-bit ints: `asInt` as a quoted string and as a bare number, `timeUnixNano` as an unquoted number, and `success` as a real JSON boolean rather than the documented string. |
| `as_double_rounding.json` | `asDouble: 1522.7` rounds to 1523 rather than truncating to 1522, and one unreadable point (`"asDouble": "n/a"`) among readable ones is skipped — the signal stays `present` with the readable total and carries no reason, because profile/v1 has nowhere to report a skipped record. |
| `value_not_a_count.json` | Four points whose values decode but are not token counts: `9.3e18` and `1e300` are past `int64`, `-500` and `asInt: "-1"` are negative. Go leaves an out-of-range float-to-integer conversion to the architecture, so before 0.4.1 this file produced `MaxInt64` on arm64 and `MinInt64` on amd64 — two profiles from one capture. Every point is refused and counted. |
| `cache_only.json` | An export that carried only a `cacheRead` point. `input` and `output` were not read, so the profile has no key for them: "nothing was said about output" is not "no output tokens". |
| `zero_token_count.json` | A count read as zero is a measurement, not an absence: `input: 0` and `cache_creation: 0` are both in the profile. |
| `gauge_not_sum.json` | `claude_code.token.usage` arriving as a `gauge` instead of a `sum`: counted as seen, carries no sum data points, no panic. |
| `accept_then_result.json` | An accepted `tool_decision` and its `tool_result` on one `tool_use_id` yield exactly one entry. The two entry sources are disjoint, so no de-duplication is needed. |
| `tool_failure.json` | `success: "false"` is a listed call that failed; a `tool_result` with no `success` key at all is not a call that failed — it is not listed, and it is counted in the reason instead. |
| `unreadable_tool_events.json` | One defect of each kind (no `tool_name`, no readable `success`, an unrecognised `decision`), so the `calls == 0` reason names all three with the count the walk observed. |
| `accepts_no_results.json` | Accepts with no results yet — an export captured mid-run. The reason says that, and does not claim the events carried no tool name. |
| `unnamed_reject.json` | A rejected `tool_decision` with no `tool_name`: not an entry, and the reason names it rather than falling through to an empty clause list. |
| `no_timestamp_tool_call.json` | A `tool_result` with no `timeUnixNano` is still a tool call. It is kept, sorted last, with `timestamp: ""`. |
| `skill_name_present.json` | `skill.name` verbatim (`"my-skill"`) on `token.usage` and `"third-party"` on `api_request`: user-defined skill names are not redacted. Tokens and timing are `present`; `skill_activation` stays `unknown` because this adapter does not read the attribute yet. |
| `unreadable_temporality.json` | A sum whose `aggregationTemporality` reads as neither 1 (delta) nor 2 (cumulative) — `0`/unspecified in one metric, unparseable text in the other. Neither is assimilated, and the reason names how many points were refused rather than silently picking a direction. |
| `untimed_api_request.json` | An `api_request` with no `timeUnixNano`. The `event.timestamp` attribute beside it duplicates the record field in RFC 3339, and is deliberately not a second time source: timing is `unknown`, naming what was unreadable. |
| `out_of_order.ndjson` | Timing is a span over `api_request` records, not file order: batches arrive newest-first and `total_ms` is still positive. |
| `unknown_events.json` | Only `hook_registered` and `user_prompt`. Unknown event names are ignored silently: all three signals `unknown`, never `error`. |
| `malformed.json` | Truncated mid-object in the first and only batch: `error`, naming the byte offset, with no mid-write advice (there were no earlier batches to lose). |
| `truncated_final_line.ndjson` | Three complete batches and a partial fourth line, as a collector killed mid-write leaves behind: `error` naming batch 4 and the offset, plus what to do about it. Earlier batches are not used — a swallowed parse error is how the adapter used to lie. |
| `type_mismatch.json` | `resourceMetrics` as an object where the schema wants an array: `error`, naming the JSON path, never a Go type name. |
| `top_level_array.json` | A JSON array of export objects — a plausible mistake, and not OTLP: `error` saying so in those words. |
| `empty.json` | Zero bytes: `error` saying the file is empty, not a decoder message about the end of input. |
| `no_envelope.json` | `{}` — well-formed JSON, no OTLP envelope: `unknown` naming the expected format. Parsed fine, carried no telemetry. |
| `bespoke_envelope.json` | The `{"metrics": [...], "logs": [...]}` envelope the adapter invented before 0.4.1 and nothing ever emitted. It must now read as `unknown`, naming the format the adapter expects. Before 0.4.1 this file — and only a hand-written file like it — produced values. |
