# OTLP/JSON fixtures for the Claude Code adapter

Every file here is an input to `ClaudeCodeAdapter`, read through `--otel-file`.
Most are real OTLP/JSON: one `ExportMetricsServiceRequest` or
`ExportLogsServiceRequest` object per JSON object, exactly what a local OTLP
receiver (README route (a)) or the collector's `file` exporter (route (b))
writes. `.ndjson` files hold one object per line, which is the framing both
capture routes produce. `.json` files hold a single object, pretty-printed for
review — except for the three that exist to be unreadable, which are whatever
their failure needs them to be: `empty.json` is zero bytes, `malformed.json` is
one truncated line, and `top_level_array.json` is an array rather than an
object. `no_envelope.json` is listed here for its formatting too — it is `{}` —
but it is not unreadable: it parses, carries no envelope, and so reports
`unknown` rather than `error`.

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
`user.email`, `user.id`, `user.account_id`, `user.account_uuid`,
`organization.id` and `session.id` on every metric data point and every log
record — and that list is what was observed on one version, so treat it as the
floor rather than the whole of it. See the privacy note in the repository
README ("A capture identifies you"). Read the file before you share it.

| Fixture | What it pins |
|---|---|
| `full_export.ndjson` | The happy path in the shape a capture actually has: a metrics batch and a logs batch concatenated as NDJSON. All four token types, a `tool_result`, a rejected `tool_decision`, two `api_request`s. All three signals `present`. |
| `tokens_only.json` | Two `resourceMetrics` entries with two `scopeMetrics` each — legal, and a parser that indexes `[0]` loses three quarters of the file. Also two non-token metrics carrying their own `type` attribute: `claude_code.lines_of_code.count` here carries `type: "output"`, which it never does in reality, so that if the `type` lookup were not scoped to `claude_code.token.usage` the output count would be wrong by 999999. |
| `tool_calls_only.json` | The three ways a record names its event: `body.stringValue`, a bare-string `body`, and no `body` at all with only the `event.name` attribute. |
| `log_record_shapes.json` | Four `tool_result` records spread over two `resourceLogs` with two `scopeLogs` each — a walk that indexes `[0]` loses three quarters of them. The first record's `body` says `tool_result` while its `event.name` attribute says `api_request`: the body wins, so this export has no `api_request` in it at all. Two of the four carry no `timeUnixNano` and sort last, in file order. |
| `as_double_wins_over_as_int.json` | One point carrying both `asDouble` and `asInt`, with different numbers in them. `asDouble` is what Claude Code's own exporter sets, so it is what is read — consistently, not by whichever the decoder reached first. |
| `partial_no_tokens.json` | A logs-only capture: tool calls and timing are `present`, tokens `unknown` with its own reason. A partial export never costs the caller the signals it does carry. |
| `timing_only.json` | A single `api_request`: a zero-length span is a value that was read, so `total_ms: 0` is `present`, not `unknown`. |
| `multi_batch_delta.ndjson` | Delta temporality across three batches: 100 + 200 + 300 = 600. The default temporality, so this is the common case. |
| `cumulative.ndjson` | Cumulative temporality: one series reports 500 then 900 (last by `timeUnixNano` wins, not 1400), a second series on a different `model` adds its own 250, and one batch declares the enum as the string `"2"`. |
| `array_attribute_series.ndjson` | Two cumulative series distinguished only by an `arrayValue` attribute. OTel identifies a series by its whole attribute set, so these are two running totals that add to 300 — not one series whose later point supersedes the earlier and reports 200. |
| `multi_resource_cumulative.json` | Two `resourceMetrics` entries whose resources differ only by `service.instance.id`, carrying cumulative points with *identical* data-point attributes — what a collector fanning in two agents writes. They are two series and add to 300; keyed on the attributes alone they merged, and the later point reported 200 as the whole capture. |
| `multi_resource_delta.json` | The same two resources and the same numbers under delta temporality, which is Claude Code's default. Delta points add up whatever series they belong to, so this reported 300 before series identity included the resource and must still report 300: it is the control that catches an identity change paid for by breaking the common case. |
| `multi_scope_cumulative.json` | One resource, four `scopeMetrics` under it: two scopes differing only by `version`, one by `name`, and one carrying a scope `attributes` entry the others do not. All four carry identical data-point attributes, so a series key that ignores the scope reports the latest point, 60, instead of 100 + 200 + 40 + 60 = 400. |
| `resource_identity_kinds.ndjson` | Two resources differing only in the *kind* of one attribute's value — `deployment.replica` as `intValue: 5` and as `stringValue: "5"`. Resource identity is derived the way attribute identity is, tagged by kind, so these are two series adding to 300; collapsing the kinds merges them and reports 200. |
| `counter_reset.ndjson` | A cumulative counter that restarts inside one capture: `startTimeUnixNano` 1789332594304000000 reaching 60 then 100, then a new run from 1789332597000000000 carrying 20. The runs add to 120. Reading only the latest point — the file's 20 — reports the tail of a session as the whole of it. |
| `counter_no_reset.ndjson` | The same three points with one start time throughout: one run of 60, 100, 120, so the total is 120 and not 280. The control that keeps the reset rule from adding up points that never restarted. |
| `absent_start_time.json` | Cumulative points carrying no `startTimeUnixNano` at all. None of them can say which run it is from, so they are one run and the latest running total, 120, stands — the behaviour every capture had before runs existed. |
| `unreadable_start_time.json` | Cumulative points whose `startTimeUnixNano` is present and does not read: text in one, a fractional number in the other. Unreadable is treated as absent — not as a run boundary of its own — so this is one run reporting 120. A start time that does not read is not evidence that a counter restarted. |
| `temporality_enum_names.json` | `aggregationTemporality` as the proto enum *name* rather than the integer — what a producer going through the standard protobuf JSON mapping emits. `AGGREGATION_TEMPORALITY_DELTA` and `AGGREGATION_TEMPORALITY_CUMULATIVE`, one metric each. |
| `number_string_variants.json` | The spec's "either numbers or strings are accepted" for 64-bit ints: `asInt` as a quoted string and as a bare number, `timeUnixNano` as an unquoted number, and `success` as a real JSON boolean rather than the documented string. |
| `as_double_rounding.json` | `asDouble: 1522.7` rounds to 1523 rather than truncating to 1522, and one unreadable point (`"asDouble": "n/a"`) among readable ones is skipped — the signal stays `present` with the readable total and carries no reason, because profile/v1 has nowhere to report a skipped record. |
| `value_not_a_count.json` | Four points whose values decode but are not token counts: `9.3e18` and `1e300` are past `int64`, `-500` and `asInt: "-1"` are negative. Go leaves an out-of-range float-to-integer conversion to the architecture, so before 0.4.1 this file produced `MaxInt64` on arm64 and `MinInt64` on amd64 — two profiles from one capture. Every point is refused and counted. |
| `cache_only.json` | An export that carried only a `cacheRead` point. `input` and `output` were not read, so the profile has no key for them: "nothing was said about output" is not "no output tokens". |
| `zero_token_count.json` | A count read as zero is a measurement, not an absence: `input: 0` and `cache_creation: 0` are both in the profile. |
| `gauge_not_sum.json` | `claude_code.token.usage` arriving as a `gauge` instead of a `sum`: counted as seen, carries no sum data points, no panic. |
| `accept_then_result.json` | An accepted `tool_decision` and its `tool_result` on one `tool_use_id` yield exactly one entry. The two entry sources are disjoint, so no de-duplication is needed. |
| `tool_failure.json` | `success: "false"` is a listed call that failed; a `tool_result` with no `success` key at all is not a call that failed — it is not listed at all. It is counted, but the count goes nowhere here: the other call was read, so the result is `present`, and a `present` result carries no reason in profile/v1. The count only reaches a reason when nothing was read (see `unreadable_tool_events.json`). |
| `unreadable_tool_events.json` | One defect of each kind (no `tool_name`, no readable `success`, an unrecognised `decision`), so the `calls == 0` reason names all three with the count the walk observed. |
| `accepts_no_results.json` | Accepts with no results yet — an export captured mid-run. The reason says that, and does not claim the events carried no tool name. |
| `unnamed_reject.json` | A rejected `tool_decision` with no `tool_name`: not an entry, and the reason names it rather than falling through to an empty clause list. |
| `no_timestamp_tool_call.json` | A `tool_result` with no `timeUnixNano` is still a tool call. It is kept, sorted last, with `timestamp: ""`. |
| `skill_name_present.json` | `skill.name` verbatim (`"my-skill"`) on `token.usage` and `"third-party"` on `api_request`: user-defined skill names are not redacted. Tokens and timing are `present`; `skill_activation` stays `unknown` because this adapter does not read the attribute yet. |
| `unreadable_temporality.json` | A sum whose `aggregationTemporality` reads as neither 1 (delta) nor 2 (cumulative) — `0`/unspecified in one metric, unparseable text in the other. Neither is assimilated, and the reason names how many points were refused rather than silently picking a direction. |
| `absent_temporality.json` | A `sum` that declares no `aggregationTemporality` at all. Absent is not the same defect as unreadable, and neither is the same as a temporality that was declared and is neither delta nor cumulative: the reason has a clause for each, because they send the reader to three different places in their capture. |
| `unrecognised_token_type.json` | One point whose `type` is `reasoning` — a type Claude Code does not emit — and one with no `type` attribute at all. Neither contributes, and the reason says exactly that and nothing about the values, which were fine. |
| `empty_envelope.json` | `{"resourceMetrics": [], "resourceLogs": []}` — an OTLP export that carried no telemetry, which a session that emitted nothing produces. It is an export, so each signal gets its own "nothing of mine is in here" reason rather than all three being told the file is not OTLP/JSON. |
| `untimed_api_request.json` | An `api_request` with no `timeUnixNano`. The `event.timestamp` attribute beside it duplicates the record field in RFC 3339, and is deliberately not a second time source: timing is `unknown`, naming what was unreadable. |
| `out_of_order.ndjson` | Timing is a span over `api_request` records, not file order: batches arrive newest-first and `total_ms` is still positive. |
| `unknown_events.json` | Only `hook_registered` and `user_prompt`. Unknown event names are ignored silently: all three signals `unknown`, never `error`. |
| `malformed.json` | Truncated mid-object in the first and only batch: `error` naming byte 87, which is the file's length — the file ended, so the byte the decoder wanted is the one past the end. No mid-write advice, because there were no earlier batches to lose. |
| `truncated_final_line.ndjson` | Three complete batches and a partial fourth line, as a collector killed mid-write leaves behind: `error` naming batch 4 and the offset, plus what to do about it. Earlier batches are not used — a swallowed parse error is how the adapter used to lie. |
| `stray_close_then_batch.ndjson` | A complete batch, a line holding a lone `}`, then a second complete batch carrying 200. The stray byte is what a half-unwrapped JSON array or a doubled write leaves behind, and it is the shape a reader that stops at "no next element" reports as `present` with 100 in it — a measurement of part of the file, with the 200 dropped in silence. It is `error`, naming batch 2, and neither number reaches the profile. |
| `type_mismatch.json` | `resourceMetrics` as an object where the schema wants an array: `error`, naming the JSON path, never a Go type name. |
| `top_level_array.json` | A JSON array of export objects — a plausible mistake, and not OTLP: `error` saying so in those words. |
| `empty.json` | Zero bytes: `error` saying the file is empty, not a decoder message about the end of input. |
| `no_envelope.json` | `{}` — well-formed JSON, no OTLP envelope: `unknown` naming the expected format. Parsed fine, carried no telemetry. |
| `bespoke_envelope.json` | The `{"metrics": [...], "logs": [...]}` envelope the adapter invented before 0.4.1 and nothing ever emitted. It must now read as `unknown`, naming the format the adapter expects. Before 0.4.1 this file — and only a hand-written file like it — produced values. |
