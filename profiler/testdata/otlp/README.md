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

**Every `sum` data point and every log record here carries
`session.id: 00000000-0000-4000-8000-000000000001`**, the one session these
fixtures are a capture of, except where a fixture exists to hold more than one
(`two_sessions.ndjson`, `two_sessions_one_metric.json`, `foreign_scope.json` —
all three use their own ids, listed below). A capture reads only the records carrying the session it was asked for,
so a fixture whose records name no session is an export no producer writes and
pins nothing; tests read these through `fixtureSession`.

Two kinds of record are deliberately left without one, because the session
filter never reaches them. `malformed.json` and the tail of
`truncated_final_line.ndjson` cannot be read as an export at all. And
`gauge_not_sum.json` carries a **gauge** point: only a sum's data points are
session-filtered, because a sum is the only shape a signal reads a value out
of, so that export is refused for its shape whoever's it is and a `session.id`
on it would pin nothing.

| Fixture | What it pins |
|---|---|
| `full_export.ndjson` | The happy path in the shape a capture actually has: a metrics batch and a logs batch concatenated as NDJSON. All four token types, a `skill_activated`, a `tool_result`, a rejected `tool_decision`, two `api_request`s. All four export-backed signals `present`. |
| `tokens_only.json` | Two `resourceMetrics` entries with two `scopeMetrics` each — legal, and a parser that indexes `[0]` loses three quarters of the file. Also two non-token metrics carrying their own `type` attribute: `claude_code.lines_of_code.count` here carries `type: "output"`, which it never does in reality, so that if the `type` lookup were not scoped to `claude_code.token.usage` the output count would be wrong by 999999. |
| `tool_calls_only.json` | The three ways a record names its event: `body.stringValue`, a bare-string `body`, and no `body` at all with only the `event.name` attribute. |
| `kvlist_attribute_series.ndjson` | Two cumulative data points whose `kvlistValue` attribute carries the same two members in opposite order, beside a second pair carrying that map nested inside an `arrayValue`. A map is equal irrespective of member order and an array is not, so this is two series holding 100 and 50; an identity taken from the compacted JSON reads four and reports 300. Cumulative, because that is the temporality under which two split series *add*. |
| `two_sessions.ndjson` | Two sessions in one export, which is what both documented capture routes produce: they append to one file by design, and route (a) listens on the standard OTLP port. Session `2222…` spent 48000/9100 tokens over 12 seconds and made two tool calls; session `3333…` sits beside it with 1200/340 and one call, 27.8 hours later. Read without scoping, the file reports 49200/9440 and a `total_ms` of 99999000 for whichever session it is labelled with. |
| `two_sessions_one_metric.json` | The other multi-session metric layout, and the one `two_sessions.ndjson` cannot hold: both sessions inside **one** metric's `dataPoints` array, interleaved, which is what a collector flushing them together writes. Session `2222…` holds 5000/700 and session `3333…` holds 90000/4300, so the filter has to run within the array — the four points total 95000/5000, which is neither session's. `two_sessions.ndjson` gives each session its own batch, so scoping it can drop a whole metric and never filter inside one. A session the file does not carry empties the metric, which must then be dropped: kept as an empty sum it makes the reason say the metric "carried no sum data points" of an export carrying four. |
| `foreign_scope.json` | One session id (`2222…`) on every record, so only the scope and the event naming can tell three of them apart: a `claude_code.token.usage` sum of 7000 and a `tool_result` under scope `some.other.product`, and a record under the harness's own scope whose `body` names a bare `tool_result` that only re-qualification in the reader turned into a `claude_code` event. Beside them, a genuine 1000-token point, a `tool_result`, an `api_request`, and a record named by its `event.name` attribute in the short form — which is the documented spelling and must still be read. A reader that attributes by name alone reports 8000 input tokens, four tool calls and a 99999000 ms span. |
| `log_record_shapes.json` | Four `tool_result` records spread over two `resourceLogs` with two `scopeLogs` each — a walk that indexes `[0]` loses three quarters of them. The first record's `body` says `tool_result` while its `event.name` attribute says `api_request`: the body wins, so this export has no `api_request` in it at all. Two of the four carry no `timeUnixNano` and sort last, in file order. |
| `as_double_wins_over_as_int.json` | One point carrying both `asDouble` and `asInt`, with different numbers in them. `asDouble` is what Claude Code's own exporter sets, so it is what is read — consistently, not by whichever the decoder reached first. |
| `partial_no_tokens.json` | A logs-only capture: tool calls and timing are `present`, tokens `unknown` with its own reason. A partial export never costs the caller the signals it does carry. |
| `timing_only.json` | A single `api_request`: a zero-length span is a value that was read, so `total_ms: 0` is `present`, not `unknown`. |
| `multi_batch_delta.ndjson` | Delta temporality across three batches: 100 + 200 + 300 = 600. The default temporality, so this is the common case. |
| `cumulative.ndjson` | Cumulative temporality: one series reports 500 then 900 (the run reached 900, not 1400 — they are running totals of each other), a second series on a different `model` adds its own 250, and one batch declares the enum as the string `"2"`. |
| `array_attribute_series.ndjson` | Two cumulative series distinguished only by an `arrayValue` attribute. OTel identifies a series by its whole attribute set, so these are two running totals that add to 300 — not one series holding the greater of them and reporting 200. |
| `multi_resource_cumulative.json` | Two `resourceMetrics` entries whose resources differ only by `service.instance.id`, carrying cumulative points with *identical* data-point attributes — what a collector fanning in two agents writes. They are two series and add to 300; keyed on the attributes alone they merged into one run holding 200, and 200 was reported as the whole capture. |
| `multi_resource_delta.json` | The same two resources and the same numbers under delta temporality, which is Claude Code's default. Delta points add up whatever series they belong to, so this reported 300 before series identity included the resource and must still report 300: it is the control that catches an identity change paid for by breaking the common case. |
| `multi_scope_cumulative.json` | One resource, four `scopeMetrics` under it: two scopes differing only by `version`, one by `name`, and one carrying a scope `attributes` entry the others do not. All four carry identical data-point attributes, so a series key that ignores the scope collapses them into one run holding 200, instead of 100 + 200 + 40 + 60 = 400. |
| `resource_identity_kinds.ndjson` | Two resources differing only in the *kind* of one attribute's value — `deployment.replica` as `intValue: 5` and as `stringValue: "5"`. Resource identity is derived the way attribute identity is, tagged by kind, so these are two series adding to 300; collapsing the kinds merges them and reports 200. |
| `counter_reset.ndjson` | A cumulative counter that restarts inside one capture: `startTimeUnixNano` 1789332594304000000 reaching 60 then 100, then a new run from 1789332597000000000 carrying 20. The runs add to 120. Reading only the latest point — the file's 20 — reports the tail of a session as the whole of it. |
| `counter_no_reset.ndjson` | The same three points with one start time throughout: one run of 60, 100, 120, so the total is 120 and not 280. The control that keeps the reset rule from adding up points that never restarted. |
| `absent_start_time.json` | Cumulative points carrying no `startTimeUnixNano` at all. None of them names a run, and there are no runs to place them against, so the greatest running total they reported — 120 — is the whole of what the capture guarantees. (v0.4.1 reported the latest by `timeUnixNano`, which is the same number here and a lower one where an exporter's flushes disagree.) |
| `unreadable_start_time.json` | Cumulative points whose `startTimeUnixNano` is present and does not read: text in one, a fractional number in the other. Unreadable is treated as absent — not as a run boundary of its own — so nothing here names a run and the capture holds 120. A start time that does not read is not evidence that a counter restarted. |
| `mixed_start_time.json` | One session, one series, three flushes of the same run: 800000 and 1200000 carrying `startTimeUnixNano`, then a last one that omits it. The unplaceable 1200050 joins the run that reached 1200000 and costs the 50 it exceeds it by, so the capture holds 1200050. Counted as a run of its own and added, it reports 2400050 — a session at twice its size, from one flush that dropped one field. |
| `run_flushes_disagree.ndjson` | One run, three batches, flushes of 500 then 900 then 700 with ascending `timeUnixNano`. The run reached 900; the 700 after it is a capture contradicting itself, not tokens given back. Reading the latest by `timeUnixNano` reports 700 — the answer that lets a later flush unsee a total the export states outright, and that made supplying `startTimeUnixNano` lower the profile's count. Every candidate rule gives a different number here: latest 700, first 500, greatest 900, summed 2100. |
| `unplaced_above_every_run.json` | Two runs reaching 100 and 20, beside a point carrying no `startTimeUnixNano` that reports 500. The 500 is a running total of one of the two runs; the cheapest reading puts it on the run that reached 100, which leaves the other run's 20 beside it — 520. Taking the series to be the greatest total observed on it reports 500 and drops a run that named itself; counting the point as a run of its own reports 620 and invents one. |
| `unplaced_under_the_runs_sum.json` | Three runs of 100 each beside an unplaceable 150 — the case where the unplaceable total is above the largest run but below the runs' sum. Whichever run the 150 came from reached 150, and the other two still hold 100 each: 350. A rule that compares the unplaceable point only with the runs' sum never fires here and loses 50 tokens the points reported, silently. |
| `zero_start_time.json` | The same series with `startTimeUnixNano` spelled three ways: the string `"0"`, absent, and the number `0`. `start_time_unix_nano` is a proto3 `fixed64`, and OTLP mandates the proto3 JSON mapping, so an explicit zero and an absent field are two encodings of one message — `protojson` writes `"0"` with `EmitUnpopulated` on and omits it with it off. All three are one unplaced bucket reporting 120; telling zero from absent splits the series and reports 230. |
| `mixed_temporality.ndjson` | One series carrying both temporalities — a cumulative 50 and a delta 100 on the same resource, scope, metric and attributes — beside a well-formed delta series of 640 on another model. Delta and cumulative are opposite instructions, so no total the mixed series could contribute is in the export: it is refused and the file still reports the 640 it does carry. |
| `mixed_temporality_only.ndjson` | The same malformed series with nothing else in the file, so the refusal is what the reader sees: `tokens` is `unknown` and the reason counts the series, not its points — the points are individually fine and it is their company that is malformed. |
| `mixed_temporality_two_series.ndjson` | Two series, told apart by `model`, each carrying both temporalities and nothing else in the file. The reason must read "2 time series": "time series" is its own plural, and this is the only fixture that reaches the plural branch of the helper that knows it. |
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
| `skill_activated.json` | Three `claude_code.skill_activated` events, written newest-first so file order is not timestamp order. One carries an `invocation_trigger` and one does not — the trigger is optional and its key is absent rather than empty. The third carries no `timeUnixNano`: an activation was still read, so it is kept and sorted last. `custom_skill` is the name Claude Code logs for a user-defined or third-party skill without `OTEL_LOG_TOOL_DETAILS=1`, and is reported as spelled. |
| `unreadable_activation_events.json` | Two `skill_activated` events, one with no `skill.name` attribute and one with an empty one. Neither is an activation anyone can act on, so both are counted and the `activations == 0` reason names the count. |
| `foreign_scope_activation.json` | One `skill_activated` under `some.other.product` and one under the harness's own scope, both carrying this session's id. Only the harness's is read — the identity test alone cannot tell them apart. |
| `foreign_scope_activation_only.json` | The other half: the file's only `skill_activated` is another product's. `skill_activation` is `unknown` and the reason carries the count of log records the scope test removed, so it cannot be read as saying the export logs no activation. |
| `two_sessions_activation.json` | Four `skill_activated` records interleaved: one session `2222…`, two session `3333…`, and one carrying no `session.id` at all. Each session's profile holds its own and neither holds the unattributable record — an activation the adapter cannot attribute belongs to no profile. A session the file does not carry gets `unknown` with all four counted in the reason. |
| `skill_name_present.json` | `skill.name` verbatim (`"my-skill"`) on `token.usage` and `"third-party"` on `api_request`: user-defined skill names are not redacted. Tokens and timing are `present`; `skill_activation` is `unknown`, because the attribute marks a skill active for a request and is not an activation record — this file logs no `skill_activated` event. It is the negative control for the activation extractor: a reader keyed on the attribute rather than on the event reports two activations here, at a metric flush time and a request time, neither of which is an invocation. |
| `unreadable_temporality.json` | A sum whose `aggregationTemporality` reads as neither 1 (delta) nor 2 (cumulative) — `0`/unspecified in one metric, unparseable text in the other. Neither is assimilated, and the reason names how many points were refused rather than silently picking a direction. |
| `absent_temporality.json` | A `sum` that declares no `aggregationTemporality` at all. Absent is not the same defect as unreadable, and neither is the same as a temporality that was declared and is neither delta nor cumulative: the reason has a clause for each, because they send the reader to three different places in their capture. |
| `unrecognised_token_type.json` | One point whose `type` is `reasoning` — a type Claude Code does not emit — and one with no `type` attribute at all. Neither contributes, and the reason says exactly that and nothing about the values, which were fine. |
| `empty_envelope.json` | `{"resourceMetrics": [], "resourceLogs": []}` — an OTLP export that carried no telemetry, which a session that emitted nothing produces. It is an export, so each signal gets its own "nothing of mine is in here" reason rather than all four being told the file is not OTLP/JSON. |
| `untimed_api_request.json` | An `api_request` with no `timeUnixNano`. The `event.timestamp` attribute beside it duplicates the record field in RFC 3339, and is deliberately not a second time source: timing is `unknown`, naming what was unreadable. |
| `out_of_order.ndjson` | Timing is a span over `api_request` records, not file order: batches arrive newest-first and `total_ms` is still positive. |
| `unknown_events.json` | Only `hook_registered` and `user_prompt`. Unknown event names are ignored silently: all four export-backed signals `unknown`, never `error`. |
| `malformed.json` | Truncated mid-object in the first and only batch: `error` naming byte 87, which is the file's length — the file ended, so the byte the decoder wanted is the one past the end. No mid-write advice, because there were no earlier batches to lose. |
| `truncated_final_line.ndjson` | Three complete batches and a partial fourth line, as a collector killed mid-write leaves behind: `error` naming batch 4 and the offset, plus what to do about it. Earlier batches are not used — a swallowed parse error is how the adapter used to lie. |
| `stray_close_then_batch.ndjson` | A complete batch, a line holding a lone `}`, then a second complete batch carrying 200. The stray byte is what a half-unwrapped JSON array or a doubled write leaves behind, and it is the shape a reader that stops at "no next element" reports as `present` with 100 in it — a measurement of part of the file, with the 200 dropped in silence. It is `error`, naming batch 2, and neither number reaches the profile. |
| `type_mismatch.json` | `resourceMetrics` as an object where the schema wants an array: `error`, naming the JSON path, never a Go type name. |
| `top_level_array.json` | A JSON array of export objects — a plausible mistake, and not OTLP: `error` saying so in those words. |
| `empty.json` | Zero bytes: `error` saying the file is empty, not a decoder message about the end of input. |
| `no_envelope.json` | `{}` — well-formed JSON, no OTLP envelope: `unknown` naming the expected format. Parsed fine, carried no telemetry. |
| `bespoke_envelope.json` | The `{"metrics": [...], "logs": [...]}` envelope the adapter invented before 0.4.1 and nothing ever emitted. It must now read as `unknown`, naming the format the adapter expects. Before 0.4.1 this file — and only a hand-written file like it — produced values. |
