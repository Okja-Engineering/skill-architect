# OTLP/JSON for the Claude Code profiler adapter — research notes

**For:** release-0.4.1, senior-implementer rebuilding `profiler/claude_code.go`
**Date:** 2026-09-13
**Status:** Answered. Ground truth captured from a live Claude Code **v2.1.221** binary on this machine, not just docs.

## 0. Evidence grade (read this first)

Three tiers of evidence are used below. Every claim is tagged.

- **[OBSERVED]** — captured from real Claude Code v2.1.221 OTLP/JSON output on this machine
  (local HTTP receiver on `127.0.0.1:4318`, `OTEL_EXPORTER_OTLP_PROTOCOL=http/json`).
  Raw captures in scratchpad: `capture_002_v1_logs.json`, `capture_003_v1_logs.json`,
  `capture_004_v1_metrics.json`, `console_out.txt`. IDs redacted in this doc.
- **[BINARY]** — read out of the shipped executable
  `/Users/matthewvandusen/.local/share/claude/versions/2.1.221` (Bun single-file bundle; embedded JS
  recovered with `strings`/byte-offset extraction).
- **[DOCS]** — <https://code.claude.com/docs/en/monitoring-usage>. The `.md` source
  (`https://code.claude.com/docs/en/monitoring-usage.md`, 153 KB) was used rather than the rendered
  HTML, so quotes are verbatim.

**Caveat on coverage.** The nested `claude -p` sessions could not authenticate
(`Failed to authenticate: OAuth session expired and could not be refreshed`), so the live capture
contains `session.count`, `active_time.total`, `hook_registered`, `user_prompt`, and `api_error`
but **not** `token.usage`, `api_request`, `tool_result`, or `tool_decision`. The *envelope and
encoding* conventions — which are the load-bearing unknowns — are [OBSERVED] and apply uniformly to
all signals, because they are produced by one exporter code path. The *catalogue* of the missing
signals is [DOCS], and the int-vs-double question for `token.usage` is settled independently at
[BINARY] level (§2.4). This is called out again per-row in §4.

---

## 1. How a user gets an OTLP/JSON file out of Claude Code

Claude Code has **no file exporter**. [DOCS], env-var table:

> | `OTEL_METRICS_EXPORTER` | Metrics exporter types, comma-separated. Use `none` to disable | `console`, `otlp`, `prometheus`, `none` |
> | `OTEL_LOGS_EXPORTER` | Logs/events exporter types, comma-separated. Use `none` to disable | `console`, `otlp`, `none` |

`file` is not in either set. (`OTEL_LOG_RAW_API_BODIES=file:<dir>` does write files, but those are
raw Anthropic API request/response bodies, **not** OTLP — do not confuse the two. [DOCS] line 1385:
"Claude Code writes untruncated bodies to `.request.json` and `.response.json` files under that
directory, and the events carry a `body_ref` path instead of the inline body.")

So the file must be produced by something *downstream* of Claude Code. Three routes:

### Route (a) — local HTTP receiver that dumps request bodies ✅ RECOMMENDED

```bash
export CLAUDE_CODE_ENABLE_TELEMETRY=1
export OTEL_METRICS_EXPORTER=otlp
export OTEL_LOGS_EXPORTER=otlp
export OTEL_EXPORTER_OTLP_PROTOCOL=http/json
export OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4318
export OTEL_METRIC_EXPORT_INTERVAL=2000     # default 60000 — too slow for a short profiling run
export OTEL_LOGS_EXPORT_INTERVAL=1000       # default 5000
export OTEL_LOG_TOOL_DETAILS=1              # needed for skill_name in tool_parameters
claude -p "..."
```

**[OBSERVED] what actually arrives on the wire:**

- Two separate endpoints, exactly per OTLP spec: `POST /v1/metrics` and `POST /v1/logs`.
- `Content-Type: application/json`.
- No `Content-Encoding` (uncompressed) in the observed runs.
- **One complete `ExportMetricsServiceRequest` / `ExportLogsServiceRequest` JSON object per POST
  body.** Not NDJSON, not an array. Each body is a single `{...}`.
- Multiple POSTs per session (one per export interval). Observed: 2 log POSTs + 1 metric POST for a
  single short session.

Receiver log from the real run:

```
POST /v1/logs    ct=application/json bytes=2951 -> capture_002_v1_logs.json
POST /v1/logs    ct=application/json bytes=1986 -> capture_003_v1_logs.json
POST /v1/metrics ct=application/json bytes=2280 -> capture_004_v1_metrics.json
```

This route is **parseable, zero-dependency, and exactly reproducible**. A ~40-line Python
`http.server` is enough (see `receiver.py` in scratchpad). The natural file convention is: append
each POST body as one line → NDJSON, which is also what route (b) produces, so **one parser handles
both**.

OTLP spec confirms this is the contract
(<https://opentelemetry.io/docs/specs/otlp/>):

> "The client and the server MUST set 'Content-Type: application/json' request and response headers
> when sending JSON Protobuf encoded payload."

and the default paths are `/v1/traces`, `/v1/metrics`, `/v1/logs`, each carrying their respective
`Export*ServiceRequest` messages.

### Route (b) — OpenTelemetry Collector with the `file` exporter ✅ ALSO PARSEABLE

Claude Code → collector (OTLP in) → `file` exporter (JSON out).

Verified against **source**, not just the README, because the README does not state the per-line
message type.

`exporter/fileexporter/marshaller.go`:

```go
var metricsMarshalers = map[string]pmetric.Marshaler{
	formatTypeJSON:  &pmetric.JSONMarshaler{},
	formatTypeProto: &pmetric.ProtoMarshaler{},
}
var logsMarshalers = map[string]plog.Marshaler{
	formatTypeJSON:  &plog.JSONMarshaler{},
	formatTypeProto: &plog.ProtoMarshaler{},
}
```

`pmetric.JSONMarshaler` serialises a `pmetric.Metrics`, whose OTLP wire form is
`{"resourceMetrics":[...]}` — i.e. wire-identical to `ExportMetricsServiceRequest`. Same for
`plog` → `{"resourceLogs":[...]}`.

`exporter/fileexporter/file_writer.go` — one object per line:

```go
func exportMessageAsLine(w *fileWriter, buf []byte) error {
	...
	if _, err := w.file.Write(buf); err != nil { return err }
	if _, err := io.WriteString(w.file, "\n"); err != nil { return err }
	return nil
}

func buildExportFunc(cfg *Config) func(w *fileWriter, buf []byte) error {
	...
	if cfg.FormatType == formatTypeProto { return exportMessageAsBuffer }
	// if the data format is JSON and needs to be compressed, telemetry data can't be written to file in JSON format.
	if cfg.FormatType == formatTypeJSON && cfg.Compression != "" { return exportMessageAsBuffer }
	return exportMessageAsLine
}
```

README confirms: *"When `format` is json and `compression` is none, telemetry data is written to file
in JSON format. Each line in the file is a JSON object."*

So: **`format: json` + no compression ⇒ NDJSON, one `Export*ServiceRequest` per line.**
With `compression` set, it becomes length-prefixed binary framing — **not** line JSON. Parser should
document that compression must be off.

Caveat: metrics and logs go to **separate** file exporters (separate pipelines), so this yields two
files, or one file with mixed `resourceMetrics`/`resourceLogs` lines if both pipelines target the
same path. The parser should key off which top-level field is present per line, not off filename.

Cost: requires the user to install and configure a collector. Heavier than route (a).

### Route (c) — `console` exporter ❌ NOT PARSEABLE

[DOCS] suggests console for debugging. **It does not emit JSON.** [OBSERVED], real output:

```
{
  resource: {
    attributes: {
      "host.arch": "arm64",
      "os.type": "darwin",
      "service.name": "claude-code",
      "service.version": "2.1.221",
    },
  },
  instrumentationScope: {
    name: "com.anthropic.claude_code.events",
    version: "2.1.221",
    schemaUrl: undefined,
  },
  timestamp: 1789332793382000,
  traceId: undefined,
  spanId: undefined,
  severityText: undefined,
  severityNumber: undefined,
  body: "claude_code.hook_registered",
  attributes: {
    "user.id": "<redacted>",
    "event.name": "hook_registered",
    "event.sequence": 0,
    hook_event: "PreToolUse",
    hook_type: "command",
  },
}
```

This is Node/Bun `console.dir` object-inspection output. It is **invalid JSON**: unquoted keys
(`hook_event:`, `resource:`), bare `undefined` values, trailing commas. It is also **not the OTLP
envelope** — it is one flattened SDK-internal record per object, with no
`resourceLogs`/`scopeLogs` nesting, and `timestamp` in **microseconds** rather than
`timeUnixNano`/nanoseconds.

**Do not build a parser against console output.** Rule it out explicitly in the docs so users don't
try it after reading the Anthropic debugging tip.

### Recommendation for §1

**Document route (a)** as the primary, with a small bundled receiver script, and accept route (b)'s
NDJSON with the same parser. Route (a) needs nothing installed, is what the adapter's own test
fixtures can be generated from, and produces byte-identical `Export*ServiceRequest` objects to (b).
Ship the receiver as `profiler/scripts/otlp-capture.py` (or a Go `profiler otlp-capture` subcommand,
which would be more in keeping with the existing CLI) so the documented path is executable rather
than prose.

---

## 2. Exact OTLP/JSON shape

### 2.1 Encoding rules (OTLP spec, primary)

From <https://opentelemetry.io/docs/specs/otlp/> ("JSON Protobuf Encoding"):

> "The keys of JSON objects are field names converted to lowerCamelCase. Original field names are not
> valid to use as keys for JSON objects."

> **"64-bit integer numbers in JSON-encoded payloads are encoded as decimal strings, and either
> numbers or strings are accepted when decoding."**

> "Values of enum fields MUST be encoded as integer values. Unlike the standard Protobuf JSON
> Mapping, which allows values of enum fields to be encoded as either integer values or as enum name
> strings, only integer enum values are allowed in OTLP JSON Protobuf Encoding."

> "The `traceId` and `spanId` byte arrays are represented as case-insensitive hex-encoded strings;
> they are not base64-encoded as is defined in the standard Protobuf JSON Mapping."

The bolded sentence is the single most important line for this parser: **both numbers and strings
must be accepted for every 64-bit integer field.** This is not a tolerance we're inventing; the spec
mandates it. See §6.1 — Claude Code and the collector genuinely differ here.

### 2.2 Metrics: full path to a counter data point

[OBSERVED], real capture, ids redacted:

```json
{
  "resourceMetrics": [
    {
      "resource": {
        "attributes": [
          { "key": "host.arch",       "value": { "stringValue": "arm64" } },
          { "key": "os.type",         "value": { "stringValue": "darwin" } },
          { "key": "os.version",      "value": { "stringValue": "25.6.0" } },
          { "key": "service.name",    "value": { "stringValue": "claude-code" } },
          { "key": "service.version", "value": { "stringValue": "2.1.221" } }
        ],
        "droppedAttributesCount": 0
      },
      "scopeMetrics": [
        {
          "scope": { "name": "com.anthropic.claude_code", "version": "2.1.221" },
          "metrics": [
            {
              "name": "claude_code.session.count",
              "description": "Count of CLI sessions started",
              "unit": "",
              "sum": {
                "aggregationTemporality": 1,
                "isMonotonic": true,
                "dataPoints": [
                  {
                    "attributes": [
                      { "key": "session.id",  "value": { "stringValue": "<uuid>" } },
                      { "key": "start_type",  "value": { "stringValue": "fresh" } }
                    ],
                    "startTimeUnixNano": "1789332594304000000",
                    "timeUnixNano":      "1789332596272000000",
                    "asDouble": 1
                  }
                ]
              }
            }
          ]
        }
      ]
    }
  ]
}
```

Access path:
`resourceMetrics[].scopeMetrics[].metrics[].sum.dataPoints[].{attributes, asDouble|asInt, timeUnixNano, startTimeUnixNano}`

Confirmed facts:
- Claude Code counters are **`sum`** (not `gauge`, not `histogram`).
- `"isMonotonic": true` — JSON boolean.
- `"aggregationTemporality": 1` — **integer enum**, per spec. `1` = `AGGREGATION_TEMPORALITY_DELTA`
  (proto enum: 0 UNSPECIFIED, 1 DELTA, 2 CUMULATIVE). Matches [DOCS]:
  "`OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE` | Metrics temporality preference
  (default: `delta`)". **Because the default is DELTA, data points across multiple export batches
  must be SUMMED, not max-ed or last-wins.** If a user sets `cumulative`, summing would double-count
  — see §6.2.
- `timeUnixNano` / `startTimeUnixNano` are **strings** containing nanoseconds.
- `unit` is `""` for count metrics, `"s"` for `active_time.total`, `"tokens"` for `token.usage`,
  `"USD"` for `cost.usage` [DOCS/BINARY].
- **Values arrive as `asDouble`, not `asInt`** — see §2.4, this is definitive.

### 2.3 Logs: full path to a log record

[OBSERVED], real capture, ids redacted:

```json
{
  "resourceLogs": [
    {
      "resource": {
        "attributes": [
          { "key": "service.name",    "value": { "stringValue": "claude-code" } },
          { "key": "service.version", "value": { "stringValue": "2.1.221" } }
        ],
        "droppedAttributesCount": 0
      },
      "scopeLogs": [
        {
          "scope": { "name": "com.anthropic.claude_code.events", "version": "2.1.221" },
          "logRecords": [
            {
              "timeUnixNano":         "1789332594523000000",
              "observedTimeUnixNano": "1789332594523000000",
              "body": { "stringValue": "claude_code.user_prompt" },
              "attributes": [
                { "key": "session.id",      "value": { "stringValue": "<uuid>" } },
                { "key": "event.name",      "value": { "stringValue": "user_prompt" } },
                { "key": "event.timestamp", "value": { "stringValue": "2026-09-13T20:49:54.523Z" } },
                { "key": "event.sequence",  "value": { "intValue": 1 } },
                { "key": "prompt.id",       "value": { "stringValue": "<uuid>" } },
                { "key": "prompt_length",   "value": { "stringValue": "40" } },
                { "key": "prompt",          "value": { "stringValue": "<REDACTED>" } }
              ],
              "droppedAttributesCount": 0
            }
          ]
        }
      ]
    }
  ]
}
```

Access path:
`resourceLogs[].scopeLogs[].logRecords[].{timeUnixNano, observedTimeUnixNano, body, attributes}`

**Critical structural findings [OBSERVED]:**

1. **`body.stringValue` is the FULLY-QUALIFIED event name** — `"claude_code.user_prompt"`,
   `"claude_code.api_error"`, `"claude_code.hook_registered"`.
2. **Attribute `event.name` is the SHORT name** — `"user_prompt"`, `"api_error"`. No `claude_code.`
   prefix.
3. Both are present on every event record. **Match on `body.stringValue`** (it is the primary
   identity and is present even on records where `event.name` might be dropped by cardinality
   limits), and fall back to `event.name`. Belt and braces: normalise by stripping a leading
   `claude_code.` and compare the short name.
4. **`severityText` and `severityNumber` are ABSENT entirely.** Observed record key set is exactly
   `["attributes", "body", "droppedAttributesCount", "observedTimeUnixNano", "timeUnixNano"]`.
   Do not require, key off, or filter by severity. The `severityText` field mentioned in the task
   brief does not appear in practice.
5. `event.timestamp` attribute carries an **ISO 8601 string** that duplicates `timeUnixNano`. It is
   the friendlier of the two but is an attribute, not a record field.

### 2.4 `asInt` vs `asDouble` — SETTLED, and it is `asDouble`

This is the highest-risk detail and it is settled at source level.

[OBSERVED] — `claude_code.session.count`, an unambiguously integral counter, serialises as:

```json
"asDouble": 1
```

not `"asInt": "1"`. Likewise `active_time.total` → `"asDouble": 1.777`.

[BINARY] — the reason. Claude Code creates every metric through one generic helper that passes only
name + `{description, unit}`, never a `valueType`:

```js
CMi(t, (o, i) => {
  let s = t?.createCounter(o, i);
  return { add(a, l = {}) { let u = { ...O1t(), ...l }; s?.add(a, u) } }
}, { omitUnits: r.length > 0 && r.every((o) => o === "prometheus") })
```

and the bundled OTel JS SDK defaults `valueType` to `DOUBLE`:

```js
function Hib(e, t, r) {
  ...
  return {
    name: e, type: t,
    description: r?.description ?? "",
    unit: r?.unit ?? "",
    valueType: r?.valueType ?? vTp.ValueType.DOUBLE,
    advice: r?.advice ?? {}
  }
}
Z9e.createInstrumentDescriptor = Hib;
```

**Conclusion: every `claude_code.*` counter — including `claude_code.token.usage` — is a DOUBLE
instrument and serialises as `asDouble` with a JSON number.** This generalises from the observed
`session.count` to the unobserved `token.usage` because they are created by the same call site with
the same (absent) `valueType`.

**Parser requirement:** read `asDouble` **first**, fall back to `asInt`, and accept `asInt` as either
a JSON number or a decimal string. Round to int64 at the boundary (`int64(math.Round(v))`) — do not
truncate, or a float64 representation of 999999 could land as 999998. A collector in the path
(route b) may re-serialise, so `asInt`-as-string must still parse.

---

## 3. Claude Code metric and event catalogue

### 3.1 Metrics [DOCS], names+descriptions cross-checked [BINARY]

| Metric | Description | Unit |
|---|---|---|
| `claude_code.session.count` | Count of CLI sessions started | (none) |
| `claude_code.lines_of_code.count` | Count of lines of code modified | (none) |
| `claude_code.pull_request.count` | Number of pull requests created | (none) |
| `claude_code.commit.count` | Number of git commits created | (none) |
| `claude_code.cost.usage` | Cost of the Claude Code session | USD |
| `claude_code.token.usage` | Number of tokens used | tokens |
| `claude_code.code_edit_tool.decision` | Count of code editing tool permission decisions (accept/reject) for Edit, Write, and NotebookEdit tools | (none) |
| `claude_code.active_time.total` | Total active time in seconds | s |

Note: there is **no** `claude_code.tool_decision` *metric*. Tool decisions appear as an **event**
(§3.2) and, for edit tools only, as the `claude_code.code_edit_tool.decision` counter. The current
adapter's constant `otelToolDecisionLog = "claude_code.tool_decision"` is correctly a *log* name.

**`claude_code.token.usage` attributes** [DOCS] verbatim:

> * `type`: (`"input"`, `"output"`, `"cacheRead"`, `"cacheCreation"`)
> * `model`: Model identifier (for example, "claude-sonnet-5")
> * `query_source`: Category of the subsystem that issued the request. One of `"main"`, `"subagent"`, or `"auxiliary"`
> * `speed`: `"fast"` when the request used fast mode. Absent otherwise
> * `effort`: Effort level applied to the request.
> * `agent.name`, `skill.name`, `plugin.name`, `marketplace.name`, `mcp_server.name`, `mcp_tool.name`: Skill, plugin, agent, and MCP attribution for the request.

**The attribute key is `type`, not `token_type`.** The values are **camelCase**: `cacheRead`,
`cacheCreation` — *not* `cache_read`/`cache_creation`. The current adapter is wrong on both counts.

`claude_code.session.count` attribute: `start_type` ∈ `fresh`, `resume`, `continue`, `agents_view`.
`claude_code.active_time.total` attribute: `type` ∈ `user`, `cli`. (Note `type` is overloaded across
metrics — always scope the attribute lookup to the metric name.)
`claude_code.lines_of_code.count` attributes: `type` ∈ `added`, `removed`; `model`.
`claude_code.code_edit_tool.decision` attributes: `tool_name` ∈ `Edit`/`Write`/`NotebookEdit`;
`decision` ∈ `accept`/`reject`; `source`; `language`.

### 3.2 Events (log records) [DOCS]

All events carry the standard attributes plus `event.name`, `event.timestamp`, `event.sequence`.

**`claude_code.user_prompt`** — `prompt.id`, `prompt_length`, `prompt` (redacted to `<REDACTED>`
unless `OTEL_LOG_USER_PROMPTS=1`), `message.uuid`, `command_name`, `command_source`.

**`claude_code.assistant_response`** — `prompt.id`, `response_length`, `response` (redacted by
default), `model`, `request_id`, `message.uuid`, `query_source`. Requires v2.1.193+.

**`claude_code.api_request`** [DOCS] verbatim attribute list:

> * `model`: Model used (for example, "claude-sonnet-5")
> * `cost_usd`: Estimated cost in USD
> * `cost_usd_micros`: Estimated cost in millionths of a US dollar, emitted as an integer
> * `duration_ms`: Request duration in milliseconds
> * `input_tokens`: Number of input tokens
> * `output_tokens`: Number of output tokens
> * `cache_read_tokens`: Number of tokens read from cache
> * `cache_creation_tokens`: Number of tokens used for cache creation
> * `request_id`, `client_request_id`, `speed`, `query_source`, `effort`
> * `agent.name`, `skill.name`, `plugin.name`, `marketplace.name`, `mcp_server.name`, `mcp_tool.name`

**`claude_code.api_error`** — `model`, `error`, `status_code`, `duration_ms`, `attempt`,
`request_id`, `client_request_id`, plus the same attribution set.
[OBSERVED] real record: `duration_ms` = `{"intValue": 1570}`, `attempt` = `{"intValue": 1}`,
`model` = `{"stringValue": "claude-opus-5[1m]"}`.

**`claude_code.tool_result`** [DOCS] verbatim:

> * `tool_name`: Name of the tool
> * `tool_use_id`: Unique identifier for this tool invocation.
> * `success`: `"true"` or `"false"`
> * `duration_ms`: Execution time in milliseconds
> * `error_type`: Error category string when the tool failed, such as `"Error:ENOENT"` or `"ShellError"`
> * `decision_type`: Always `"accept"`, since this event is only emitted after the tool runs. Rejected calls don't produce a tool result
> * `decision_source`: ... One of `"config"`, `"hook"`, `"user_permanent"`, or `"user_temporary"`.
> * `tool_input_size_bytes`, `tool_result_size_bytes`, `mcp_server_scope`
> * `tool_parameters` (when `OTEL_LOG_TOOL_DETAILS=1`) ... For Skill tool: includes `skill_name`

Note `success` is the **string** `"true"`/`"false"`, not a JSON boolean. And on `tool_result` the
permission fields are named `decision_type`/`decision_source` — **different names** from the
`tool_decision` event's `decision`/`source`.

Also: *"Logged when a tool completes execution. Not emitted if the tool call was rejected."*

**`claude_code.tool_decision`** [DOCS] verbatim:

> * `tool_name`: Name of the tool (for example, "Read", "Edit", "Write", "NotebookEdit")
> * `tool_use_id`: Unique identifier for this tool invocation.
> * `decision`: Either `"accept"` or `"reject"`
> * `tool_source`: Always present. ... `"builtin"`, `"mcp"`, `"sdk_host_builtin_mcp"`
> * `source`: Where the decision came from: `"config"`, `"hook"`, `"user_permanent"`, `"user_temporary"`, `"user_abort"`, `"user_reject"`
> * `tool_parameters` (when `OTEL_LOG_TOOL_DETAILS=1`) ... For Skill tool: includes `skill_name`

Decision→accept/reject mapping [DOCS]: `config`, `hook`, `user_permanent`, `user_temporary` are
accepts; `user_abort` and `user_reject` are "Treated as a reject".

**The current adapter's `decision == "approved"` test is wrong — the value is `"accept"`.**

**`claude_code.api_refusal`** — `model`, `request_id`.
**`claude_code.hook_registered`** — [OBSERVED] but undocumented in the events table; attributes
`hook_event`, `hook_type`, `hook_source`, `hook_matcher`, `safe_mode`. Emitted at startup. Harmless;
the parser should ignore unknown event names rather than error.

### 3.3 Standard attributes (on every metric data point AND every log record)

[OBSERVED] on real records — note these are **data-point / log-record attributes**, not resource
attributes:

`user.id`, `session.id`, `organization.id`, `user.email`, `user.account_uuid`, `user.account_id`,
`terminal.type` (observed value `"non-interactive"` for `claude -p`).

[DOCS] adds, gated: `app.version` (`OTEL_METRICS_INCLUDE_VERSION`, default **false**),
`app.entrypoint` (`OTEL_METRICS_INCLUDE_ENTRYPOINT`, default false), `vcs.*`
(`OTEL_METRICS_INCLUDE_REPOSITORY`, default false, v2.1.269+).
Toggles: `OTEL_METRICS_INCLUDE_SESSION_ID` (default **true**),
`OTEL_METRICS_INCLUDE_ACCOUNT_UUID` (default true),
`OTEL_METRICS_INCLUDE_RESOURCE_ATTRIBUTES` (default true).

**Resource attributes** [OBSERVED] are a *different, smaller* set:
`service.name` = `"claude-code"`, `service.version` = `"2.1.221"`, `host.arch`, `os.type`,
`os.version`. **`session.id` is NOT a resource attribute** — do not look for it there.

**Scope names** [OBSERVED]: metrics → `com.anthropic.claude_code`; logs → `com.anthropic.claude_code.events`.
Useful as a cheap sanity check that a file came from Claude Code.

### 3.4 Where `skill.name` vs `skill_name` lives — important distinction

Two different keys, two different meanings:

- **`skill.name`** (dotted) — on `claude_code.token.usage`, `claude_code.cost.usage`, and the
  `api_request` / `api_error` events. [DOCS]: *"Skill active for the request, set by the Skill tool,
  a `/` command, or inherited by a spawned subagent. Built-in, bundled, user-defined, and
  official-marketplace plugin skill names appear verbatim. Third-party plugin skill names are
  replaced with `"third-party"`. Absent when no skill is active."*
  **Not gated** by `OTEL_LOG_TOOL_DETAILS` on these signals.
- **`skill_name`** (underscore) — only *inside the `tool_parameters` JSON string* on `tool_result` /
  `tool_decision`, and only when `OTEL_LOG_TOOL_DETAILS=1`, and only for the Skill tool.
  [DOCS] on the Skill-tool span attribute `skill.name`: *"For user-defined and third-party plugin
  skills the value is the placeholder `"custom_skill"` unless `OTEL_LOG_TOOL_DETAILS=1`"*.

So `tool_parameters` is a **nested JSON-encoded string**, not a structured attribute — it needs a
second `json.Unmarshal` to read `skill_name` out of it.

---

## 4. Mapping table: adapter's five signals → OTLP

| Signal | Source | Path / attribute | Honest state | Evidence |
|---|---|---|---|---|
| **Tokens** | metric `claude_code.token.usage` | `resourceMetrics[].scopeMetrics[].metrics[name=claude_code.token.usage].sum.dataPoints[]`; group by attribute `type` ∈ `input`/`output`/`cacheRead`/`cacheCreation`; value from `asDouble` (fallback `asInt`); **sum across all data points and all batches** (delta temporality) | **present** | catalogue [DOCS]; encoding [OBSERVED]+[BINARY] |
| **Tool calls** | log `claude_code.tool_result` (primary) + `claude_code.tool_decision` (rejections) | `body.stringValue`; `tool_name`, `tool_use_id`, `success` (`"true"`/`"false"` string), `duration_ms`, `timeUnixNano` | **present** | [DOCS]; envelope [OBSERVED] |
| **Timing** | log `claude_code.api_request` `duration_ms`; session bounds from min/max `timeUnixNano` across all records | see §4.2 | **present** | [DOCS]; envelope [OBSERVED] |
| **Skill activation** | attribute `skill.name` on `token.usage` / `cost.usage` / `api_request` | presence ⇒ a skill was active for that request | **present (weak)** — see §4.3 | [DOCS] |
| **Attribution** | — | no output→skill mapping exists | **unknown** — keep as-is | §4.3 |

### 4.1 Tokens — precise algorithm

```
for each line/object in file:
  for rm in resourceMetrics:
    for sm in rm.scopeMetrics:
      for m in sm.metrics where m.name == "claude_code.token.usage":
        for dp in m.sum.dataPoints:
          t := attr(dp, "type")           // "input" | "output" | "cacheRead" | "cacheCreation"
          v := value(dp)                  // asDouble first, then asInt (number OR string)
          totals[t] += int64(math.Round(v))
```

- `TokenCounts.Reasoning` has **no source** (§6.3). Leave it zero and do not claim it.
- Do **not** filter by `session.id` unless the caller asked for a specific session — but *do*
  support it, since `session.id` is on every data point and a capture file may span sessions.
- Guard `m.sum` being absent (a future version could switch an instrument to gauge); skip rather
  than panic.

### 4.2 Timing — what each timestamp actually means

Three distinct notions; be explicit about which one is reported:

1. **Per-request latency** — `duration_ms` attribute on `claude_code.api_request`. This is the
   Anthropic API call duration. Summing it gives *time spent in model calls*, not wall-clock.
2. **Wall-clock session span** — min and max `timeUnixNano` over **all** log records in the file
   (not just `api_request`). The current adapter uses only `api_request` records, which understates
   the span by excluding the `user_prompt` that precedes the first request and any `tool_result`
   after the last one. [OBSERVED] confirms `user_prompt` at `...594523000000` precedes `api_error`
   at a later ts, and `hook_registered` at `...594304000000` precedes both.
3. **`claude_code.active_time.total`** metric (unit `s`, attribute `type` ∈ `user`/`cli`) — Claude
   Code's own notion of active time. [OBSERVED] `asDouble: 1.777`. This is the most honest
   "how long was this session doing work" number and is cheap to read.

Recommendation: report wall-clock span from (2) as `StartTime`/`EndTime`/`TotalMs`, and additionally
surface (3) if the `TimingData` struct has room. `timeUnixNano` is a **nanosecond string** — parse
with `strconv.ParseInt` then `time.Unix(0, ns).UTC()`. Do **not** reuse the existing
`parseTime()` RFC3339 helper on it; it will silently return zero. (The `event.timestamp` *attribute*
is RFC3339 and would work with that helper — but prefer the record field.)

### 4.3 Skill activation vs attribution — the honest line

`skill.name` on `token.usage` / `cost.usage` / `api_request` genuinely tells you **which skill was
active for that request**, and because it is on the token metric's data points, you can sum tokens
*by skill*. That is more than the current adapter's blanket "unknown".

But note the ceiling, and it matters for the profiler's claims:

- It says a skill was **active during** the request, not that the skill **caused** the output. A
  skill loaded at turn 1 stays attributed across subsequent requests until it is no longer active.
- User-defined and third-party skill names are **redacted** (`"third-party"`, or `"custom_skill"`
  on the tool path) unless gates are set — and skill-architect's whole subject matter is
  *user-defined* skills. **This is the load-bearing limitation for this repo's use case.** Verify on
  a real authenticated run which placeholder appears for a local `skills/` skill before promoting
  activation to `present`.
- There is **no** skill-start/skill-end event pair, so activation is inferred from attribute
  presence on request-scoped signals, not observed directly.

**Recommendation:** promote **skill activation** to `present` *only if* the live check shows real
names for local skills; otherwise keep `unknown` with the redaction as the stated reason. Keep
**attribution** `unknown` regardless — nothing in the OTLP surface maps a specific output artifact
to a skill. This is a judgement call with product consequences; flagged to the manager in the
hand-off.

---

## 5. Sample payloads for test fixtures

Derived from [OBSERVED] envelopes with [DOCS] attribute sets substituted in. **Inferred fields are
marked `INFERRED` in comments here; strip comments for the actual fixture (JSON has no comments).**
Envelope, `asDouble`, `aggregationTemporality: 1`, `isMonotonic`, nanosecond string timestamps,
`body`+`event.name` duality, and `intValue`-as-number are all [OBSERVED], not inferred.

### 5.1 `ExportMetricsServiceRequest` — `testdata/otlp_metrics.json`

```json
{
  "resourceMetrics": [
    {
      "resource": {
        "attributes": [
          { "key": "host.arch", "value": { "stringValue": "arm64" } },
          { "key": "os.type", "value": { "stringValue": "darwin" } },
          { "key": "service.name", "value": { "stringValue": "claude-code" } },
          { "key": "service.version", "value": { "stringValue": "2.1.221" } }
        ],
        "droppedAttributesCount": 0
      },
      "scopeMetrics": [
        {
          "scope": { "name": "com.anthropic.claude_code", "version": "2.1.221" },
          "metrics": [
            {
              "name": "claude_code.token.usage",
              "description": "Number of tokens used",
              "unit": "tokens",
              "sum": {
                "aggregationTemporality": 1,
                "isMonotonic": true,
                "dataPoints": [
                  {
                    "attributes": [
                      { "key": "session.id", "value": { "stringValue": "00000000-0000-4000-8000-000000000001" } },
                      { "key": "type", "value": { "stringValue": "input" } },
                      { "key": "model", "value": { "stringValue": "claude-sonnet-5" } },
                      { "key": "query_source", "value": { "stringValue": "main" } },
                      { "key": "skill.name", "value": { "stringValue": "skill-audit" } }
                    ],
                    "startTimeUnixNano": "1789332594304000000",
                    "timeUnixNano": "1789332596272000000",
                    "asDouble": 1523
                  },
                  {
                    "attributes": [
                      { "key": "session.id", "value": { "stringValue": "00000000-0000-4000-8000-000000000001" } },
                      { "key": "type", "value": { "stringValue": "output" } },
                      { "key": "model", "value": { "stringValue": "claude-sonnet-5" } },
                      { "key": "query_source", "value": { "stringValue": "main" } },
                      { "key": "skill.name", "value": { "stringValue": "skill-audit" } }
                    ],
                    "startTimeUnixNano": "1789332594304000000",
                    "timeUnixNano": "1789332596272000000",
                    "asDouble": 412
                  },
                  {
                    "attributes": [
                      { "key": "session.id", "value": { "stringValue": "00000000-0000-4000-8000-000000000001" } },
                      { "key": "type", "value": { "stringValue": "cacheRead" } },
                      { "key": "model", "value": { "stringValue": "claude-sonnet-5" } }
                    ],
                    "startTimeUnixNano": "1789332594304000000",
                    "timeUnixNano": "1789332596272000000",
                    "asDouble": 20480
                  },
                  {
                    "attributes": [
                      { "key": "session.id", "value": { "stringValue": "00000000-0000-4000-8000-000000000001" } },
                      { "key": "type", "value": { "stringValue": "cacheCreation" } },
                      { "key": "model", "value": { "stringValue": "claude-sonnet-5" } }
                    ],
                    "startTimeUnixNano": "1789332594304000000",
                    "timeUnixNano": "1789332596272000000",
                    "asDouble": 3072
                  }
                ]
              }
            },
            {
              "name": "claude_code.session.count",
              "description": "Count of CLI sessions started",
              "unit": "",
              "sum": {
                "aggregationTemporality": 1,
                "isMonotonic": true,
                "dataPoints": [
                  {
                    "attributes": [
                      { "key": "session.id", "value": { "stringValue": "00000000-0000-4000-8000-000000000001" } },
                      { "key": "start_type", "value": { "stringValue": "fresh" } }
                    ],
                    "startTimeUnixNano": "1789332594304000000",
                    "timeUnixNano": "1789332596272000000",
                    "asDouble": 1
                  }
                ]
              }
            },
            {
              "name": "claude_code.active_time.total",
              "description": "Total active time in seconds",
              "unit": "s",
              "sum": {
                "aggregationTemporality": 1,
                "isMonotonic": true,
                "dataPoints": [
                  {
                    "attributes": [
                      { "key": "session.id", "value": { "stringValue": "00000000-0000-4000-8000-000000000001" } },
                      { "key": "type", "value": { "stringValue": "cli" } }
                    ],
                    "startTimeUnixNano": "1789332596270000000",
                    "timeUnixNano": "1789332596272000000",
                    "asDouble": 1.777
                  }
                ]
              }
            }
          ]
        }
      ]
    }
  ]
}
```

### 5.2 `ExportLogsServiceRequest` — `testdata/otlp_logs.json`

```json
{
  "resourceLogs": [
    {
      "resource": {
        "attributes": [
          { "key": "service.name", "value": { "stringValue": "claude-code" } },
          { "key": "service.version", "value": { "stringValue": "2.1.221" } }
        ],
        "droppedAttributesCount": 0
      },
      "scopeLogs": [
        {
          "scope": { "name": "com.anthropic.claude_code.events", "version": "2.1.221" },
          "logRecords": [
            {
              "timeUnixNano": "1789332594523000000",
              "observedTimeUnixNano": "1789332594523000000",
              "body": { "stringValue": "claude_code.user_prompt" },
              "attributes": [
                { "key": "session.id", "value": { "stringValue": "00000000-0000-4000-8000-000000000001" } },
                { "key": "event.name", "value": { "stringValue": "user_prompt" } },
                { "key": "event.timestamp", "value": { "stringValue": "2026-09-13T20:49:54.523Z" } },
                { "key": "event.sequence", "value": { "intValue": 1 } },
                { "key": "prompt.id", "value": { "stringValue": "00000000-0000-4000-8000-0000000000aa" } },
                { "key": "prompt_length", "value": { "stringValue": "40" } },
                { "key": "prompt", "value": { "stringValue": "<REDACTED>" } }
              ],
              "droppedAttributesCount": 0
            },
            {
              "timeUnixNano": "1789332595100000000",
              "observedTimeUnixNano": "1789332595100000000",
              "body": { "stringValue": "claude_code.api_request" },
              "attributes": [
                { "key": "session.id", "value": { "stringValue": "00000000-0000-4000-8000-000000000001" } },
                { "key": "event.name", "value": { "stringValue": "api_request" } },
                { "key": "event.timestamp", "value": { "stringValue": "2026-09-13T20:49:55.100Z" } },
                { "key": "event.sequence", "value": { "intValue": 2 } },
                { "key": "prompt.id", "value": { "stringValue": "00000000-0000-4000-8000-0000000000aa" } },
                { "key": "model", "value": { "stringValue": "claude-sonnet-5" } },
                { "key": "cost_usd", "value": { "doubleValue": 0.0123 } },
                { "key": "cost_usd_micros", "value": { "intValue": 12300 } },
                { "key": "duration_ms", "value": { "intValue": 1842 } },
                { "key": "input_tokens", "value": { "intValue": 1523 } },
                { "key": "output_tokens", "value": { "intValue": 412 } },
                { "key": "cache_read_tokens", "value": { "intValue": 20480 } },
                { "key": "cache_creation_tokens", "value": { "intValue": 3072 } },
                { "key": "query_source", "value": { "stringValue": "repl_main_thread" } },
                { "key": "speed", "value": { "stringValue": "normal" } },
                { "key": "skill.name", "value": { "stringValue": "skill-audit" } }
              ],
              "droppedAttributesCount": 0
            },
            {
              "timeUnixNano": "1789332595300000000",
              "observedTimeUnixNano": "1789332595300000000",
              "body": { "stringValue": "claude_code.tool_decision" },
              "attributes": [
                { "key": "session.id", "value": { "stringValue": "00000000-0000-4000-8000-000000000001" } },
                { "key": "event.name", "value": { "stringValue": "tool_decision" } },
                { "key": "event.timestamp", "value": { "stringValue": "2026-09-13T20:49:55.300Z" } },
                { "key": "event.sequence", "value": { "intValue": 3 } },
                { "key": "prompt.id", "value": { "stringValue": "00000000-0000-4000-8000-0000000000aa" } },
                { "key": "tool_name", "value": { "stringValue": "Read" } },
                { "key": "tool_use_id", "value": { "stringValue": "toolu_01AAAA" } },
                { "key": "decision", "value": { "stringValue": "accept" } },
                { "key": "tool_source", "value": { "stringValue": "builtin" } },
                { "key": "source", "value": { "stringValue": "config" } }
              ],
              "droppedAttributesCount": 0
            },
            {
              "timeUnixNano": "1789332595460000000",
              "observedTimeUnixNano": "1789332595460000000",
              "body": { "stringValue": "claude_code.tool_result" },
              "attributes": [
                { "key": "session.id", "value": { "stringValue": "00000000-0000-4000-8000-000000000001" } },
                { "key": "event.name", "value": { "stringValue": "tool_result" } },
                { "key": "event.timestamp", "value": { "stringValue": "2026-09-13T20:49:55.460Z" } },
                { "key": "event.sequence", "value": { "intValue": 4 } },
                { "key": "prompt.id", "value": { "stringValue": "00000000-0000-4000-8000-0000000000aa" } },
                { "key": "tool_name", "value": { "stringValue": "Read" } },
                { "key": "tool_use_id", "value": { "stringValue": "toolu_01AAAA" } },
                { "key": "success", "value": { "stringValue": "true" } },
                { "key": "duration_ms", "value": { "intValue": 160 } },
                { "key": "decision_type", "value": { "stringValue": "accept" } },
                { "key": "decision_source", "value": { "stringValue": "config" } },
                { "key": "tool_input_size_bytes", "value": { "intValue": 84 } },
                { "key": "tool_result_size_bytes", "value": { "intValue": 2048 } },
                { "key": "tool_parameters", "value": { "stringValue": "{\"skill_name\":\"skill-audit\"}" } }
              ],
              "droppedAttributesCount": 0
            }
          ]
        }
      ]
    }
  ]
}
```

**Inferred in these fixtures** (everything else is observed-shape): the specific attribute *values*
(token counts, durations, model name, tool name, skill name); `cost_usd` being `doubleValue`
(documented as "Estimated cost in USD", type not directly observed); `duration_ms` on `tool_result`
being `intValue` (observed as `intValue` on `api_error`, so consistent); the `tool_parameters`
nested-JSON-string form (documented as "JSON string", exact key set observed only in docs);
`speed: "normal"` on `api_request`.

**Also add a negative fixture:** the same logs file with `asInt` as a *string*
(`"asInt": "1523"`) and `event.sequence` as a string, to lock in the dual-form tolerance from §6.1.

---

## 6. Risks and required parser tolerances

### 6.1 Number-vs-string for 64-bit integers — REAL, not theoretical

The OTLP spec says int64 fields are "encoded as decimal strings, and **either numbers or strings are
accepted when decoding**". In practice the two producers disagree:

- **Claude Code direct** [OBSERVED] emits `intValue` as a **JSON number**: `{"intValue": 1570}`,
  `{"intValue": 0}`. It does *not* quote them.
- **Collector `pdata` JSON marshaler** (route b) follows canonical proto3 JSON and emits int64 as
  **quoted strings**.
- Meanwhile `timeUnixNano` / `startTimeUnixNano` **are** quoted strings from Claude Code [OBSERVED].
- And inconsistently, some *semantically numeric* attributes arrive as `stringValue`:
  `prompt_length` is `{"stringValue": "40"}` [OBSERVED], while `event.sequence` alongside it is
  `{"intValue": 1}`.

**Requirement:** every numeric read must go through one helper that accepts
`intValue` (number or string), `doubleValue`, **and** `stringValue`-containing-digits. Use
`json.Number` (via `decoder.UseNumber()`) or `any` + a type switch. A naive `int64` struct tag will
fail on route (b) *or* on route (a), depending on which you pick — this is the single most likely
source of a silently-zero token count.

### 6.2 Batching and temporality

- **Multiple POSTs/lines per session** [OBSERVED]: 3 requests for one trivial session. The parser
  must accumulate across **all** objects in the file, not read the first.
- **NDJSON vs single object**: route (a) gives one object per POST (append → NDJSON); route (b)
  gives NDJSON. Accept both: try whole-file `json.Unmarshal` first, and on failure fall back to
  line-by-line. Skip blank lines. A trailing newline is normal.
- **Multiple `resourceMetrics` / `resourceLogs` entries** per object are legal; iterate, don't index
  `[0]`.
- **Delta temporality means SUM.** Default is `delta` [DOCS], observed `aggregationTemporality: 1`.
  If a user sets `OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE=cumulative`, each batch carries a
  running total and summing **double-counts**. Mitigation: read
  `sum.aggregationTemporality` and, when it is `2` (CUMULATIVE), take the max / last value per
  attribute-set instead of summing. Cheap to implement, and without it the numbers are silently
  wrong for cumulative users. Treat `0` (UNSPECIFIED) as delta.

### 6.3 The `reasoning` token type does not exist

Searched the full docs source for `reasoning` / "thinking token": the **only** hits are
`stop_details.category` values `"reasoning_extraction"` (an API refusal category) — unrelated to
token accounting. The documented `type` enum is exactly
`"input"`, `"output"`, `"cacheRead"`, `"cacheCreation"`.

**There is no reasoning/thinking token type in Claude Code's OTel surface.** The current adapter's
`case "reasoning", "reasoning_output"` branch is dead code against invented data. Either drop
`TokenCounts.Reasoning` from the Claude Code path or leave it zero and document it as
not-emitted — do not report it as a real zero measurement, because zero and absent are different
claims.

### 6.4 Version drift

Telemetry is versioned and moving fast — the docs themselves gate features at v2.1.193, v2.1.214,
v2.1.216, v2.1.223, v2.1.268, v2.1.269. Local binary is **2.1.221**, so some documented attributes
(`vcs.*` on `tool_result`, `error_class`, `bash_command_class`) do **not** exist here yet.

Mitigations:
- Record `service.version` from resource attributes into the Profile so a capture is self-describing.
- Treat **every** attribute as optional. Missing attribute ⇒ zero value, never an error.
- Ignore unknown event names and unknown metric names silently (e.g. the undocumented
  `claude_code.hook_registered` [OBSERVED] would otherwise be a surprise).
- Do not assert on `scope.name`, but it is a good soft signal.

### 6.5 Other tolerances

- **`severityText`/`severityNumber` are absent** [OBSERVED]. Do not require them.
- **`body` may be absent or non-string** on records from other producers if a shared collector is
  used. Guard `body.stringValue`; fall back to the `event.name` attribute.
- **Redaction**: `prompt` and `response` are `"<REDACTED>"` by default [OBSERVED]. Never treat
  `<REDACTED>` as content.
- **PII**: real captures contain `user.email`, `organization.id`, `user.account_uuid`,
  `user.account_id`, `session.id` [OBSERVED]. If the profiler stores or ships capture files, scrub or
  hash these. Worth a note in the adapter docs — users will otherwise paste them into issues.
- **`success` is a string** `"true"`/`"false"`, not a bool. `strconv.ParseBool` handles it; a
  `bool` struct tag will not.
- **`tool_parameters` is nested JSON-in-a-string** — needs a second unmarshal, and must not fail the
  whole parse if malformed.
- **Empty/absent `sum`**: guard against a metric arriving as `gauge` or `histogram`.

---

## 7. Recommendation

**Document route (a)** — `OTEL_METRICS_EXPORTER=otlp` + `OTEL_LOGS_EXPORTER=otlp` +
`OTEL_EXPORTER_OTLP_PROTOCOL=http/json` pointed at a tiny local HTTP receiver that appends each POST
body as one line — and ship that receiver with the profiler so the documented path is executable
rather than prose; accept the OTel Collector `file` exporter's NDJSON with the same parser for free,
since both emit `Export*ServiceRequest` objects, and explicitly rule out the `console` exporter in
the docs because its `console.dir` output is not valid JSON at all. **Drop the bespoke
`{"metrics":[...],"logs":[...]}` envelope entirely rather than keeping it as a second accepted
input** — it was never emitted by anything, so there is no corpus of existing files to stay
compatible with, and every one of its field names (`token_type`, `cache_creation`, `reasoning`,
`decision: "approved"`) is not merely differently-spelled but *wrong*, so keeping it alive would
preserve a schema that quietly legitimises the invented `reasoning` token type and the invented
`"approved"` decision value. The smallest parser surface that makes the three signals honestly
`present` is: a tolerant value reader (`asDouble` → `asInt` → number-or-string) plus a tolerant
attribute lookup returning `(string, bool)`; walk `resourceMetrics[].scopeMetrics[].metrics[]` for
`claude_code.token.usage`, summing `sum.dataPoints[]` by the `type` attribute over the four camelCase
values (respecting `aggregationTemporality` for the cumulative case); walk
`resourceLogs[].scopeLogs[].logRecords[]` matching on `body.stringValue` with a
`claude_code.`-prefix-stripping fallback to the `event.name` attribute, taking tool calls from
`claude_code.tool_result` (`tool_name`, `success` as a *string*, `duration_ms`) plus rejections from
`claude_code.tool_decision` (`decision == "accept"`, **not** `"approved"`); and timing from the min/max
`timeUnixNano` across **all** log records — parsed as a nanosecond integer string, not RFC3339 — rather
than from `api_request` records alone. That is roughly three walk functions and two helpers, and it
needs no new dependency. The one judgement call above my level is §4.3: `skill.name` is genuinely
present on token metrics and would let the adapter report tokens-per-skill, but user-defined skill
names — which is exactly what this repo profiles — appear to be redacted to a placeholder unless
`OTEL_LOG_TOOL_DETAILS=1`, so whether skill activation graduates from `unknown` to `present` should
be decided after one authenticated run confirms what a local `skills/` skill actually reports.

---

## Appendix: sources

- Claude Code monitoring reference: <https://code.claude.com/docs/en/monitoring-usage>
  (markdown source: `https://code.claude.com/docs/en/monitoring-usage.md`)
- OTLP specification, JSON Protobuf Encoding: <https://opentelemetry.io/docs/specs/otlp/>
- opentelemetry-proto: <https://github.com/open-telemetry/opentelemetry-proto>
- Collector file exporter README: <https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/exporter/fileexporter/README.md>
- Collector file exporter source: `exporter/fileexporter/marshaller.go`, `file_writer.go`, `file_exporter.go`
- Live capture, Claude Code v2.1.221 (build `2026-08-03T03:19:26Z`, git sha `6efaf12e8b43dc7dbe50e0955c76dc4174a15876`)
- Scratchpad artifacts (session-local, not in repo):
  `/private/tmp/claude-501/-Users-matthewvandusen-Development-Auraprix-skill-architect/31edaf48-6b19-4833-82e7-6cb52dac691f/scratchpad/otlp/`
  — `capture_002_v1_logs.json`, `capture_003_v1_logs.json`, `capture_004_v1_metrics.json`,
  `console_out.txt`, `receiver.py`, `raw.md`
