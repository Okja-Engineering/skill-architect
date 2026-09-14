# Profiler adapter interface spec (F03)

**Status:** spec · **Date:** 2026-09-13 · **Architecture:** Option C — adapter-per-harness with capability negotiation

## Purpose

A harness-agnostic profiler that captures runtime signals from any target harness (Cursor, Claude Code, Codex, Devin), degrades gracefully to `unknown` for unavailable metrics, and produces a serialized profile labelled with a caller-supplied snapshot id that F04 (paired comparisons) can read and group by.

## Core types

### Metric results

Every metric is wrapped so unavailable data is explicit, not silent. There is no
generic `MetricResult[T]`. Type parameters are avoided so results marshal and
unmarshal as plain JSON, so the shared state lives in `RawMetricResult` and each
metric category has its own wrapper embedding it.

```go
type MetricState string

const (
    MetricPresent MetricState = "present" // a value was read from telemetry
    MetricUnknown MetricState = "unknown" // no telemetry source, or nothing readable in it
    MetricError   MetricState = "error"   // the source existed and failed
)

type RawMetricResult struct {
    State  MetricState `json:"state"`
    Reason string      `json:"reason,omitempty"` // present when State != "present"
    Source string      `json:"source,omitempty"` // "otel" | "hooks" | "session_data" | "server_api" | "sqlite"
}

// One wrapper per metric category. Value is a pointer wherever the payload is a
// struct, so omitempty drops the key entirely for unknown and error states.
type TokenResult struct {
    RawMetricResult
    Value *TokenCounts `json:"value,omitempty"`
}

type ToolCallResult struct {
    RawMetricResult
    Value []ToolCallEntry `json:"value,omitempty"`
}

type ActivationResult struct {
    RawMetricResult
    Value []ActivationEntry `json:"value,omitempty"`
}

type TimingResult struct {
    RawMetricResult
    Value *TimingData `json:"value,omitempty"`
}

type AttributionResult struct {
    RawMetricResult
    Value *AttributionData `json:"value,omitempty"`
}
```

Adapters build results through the constructors rather than setting `State` by
hand. The three OTel signals have all three constructors —
`PresentTokenResult`/`UnknownTokenResult`/`ErrorTokenResult`, and the same for
tool calls and timing. `ActivationResult` and `AttributionResult` have only
`Present…` and `Unknown…`: they are a property of the harness and of the
adapter, not of any export, so no export can fail them.

**Rules:**
- `State == "present"` → `Value` must be populated, `Reason` must be empty.
- `State == "unknown"` → `Value` is nil and its key is absent from the JSON; `Reason` explains why (e.g., "no `claude_code.token.usage` metric found in OTel export", "This adapter does not yet read Claude Code's skill telemetry: the `claude_code.skill_activated` event …").
- `State == "error"` → `Value` is nil and its key is absent from the JSON; `Reason` describes the failure (e.g., "malformed JSON at byte 87 in batch 1: unexpected end of JSON input", "OTel export file is empty").
- `Source` is always populated when `State == "present"` so the profile is auditable.

### CapabilityReport

Declared upfront by each adapter after probing its environment:

```go
type MetricName string

const (
    MetricTokens          MetricName = "tokens"
    MetricToolCalls       MetricName = "tool_calls"
    MetricSkillActivation MetricName = "skill_activation"
    MetricTiming          MetricName = "timing"
    MetricAttribution     MetricName = "attribution"
)

type CapabilityReport struct {
    Harness    string                    `json:"harness"`     // "cursor" | "claude_code" | "codex" | "devin"
    AdapterVer  string                    `json:"adapter_version"`
    ProbedAt   string                    `json:"probed_at"`   // ISO 8601 UTC
    Capabilities map[MetricName]MetricSource `json:"capabilities"`
}

type MetricSource string

const (
    SourceOtel       MetricSource = "otel"
    SourceHooks      MetricSource = "hooks"
    SourceSessionData MetricSource = "session_data"
    SourceServerAPI  MetricSource = "server_api"
    SourceSQLite     MetricSource = "sqlite"
    SourceNone       MetricSource = "none"
)
```

**Capability vs result:** `CapabilityReport` says what the adapter can produce for the input it was given; a metric result says what it did produce. For an adapter whose capability is derived from one resolution of one input — which is what the Claude Code adapter does, and what AC9 requires — the two cannot disagree: a capability is `otel` exactly when that signal is `present` with that source, and `none` exactly when it is `unknown` or `error`. An adapter that probed one thing and captured another is free to differ, and that is the defect AC9 exists to catch, not a licence.

### ProfilerAdapter interface

```go
type ProfilerAdapter interface {
    // Name returns the harness identifier ("cursor" | "claude_code" | "codex" | "devin").
    Name() string

    // Probe inspects the environment and returns what metrics this adapter can produce.
    // Must not fail — if nothing is available, returns a report with all "none".
    Probe() CapabilityReport

    // Capture reads telemetry for a specific session and produces a Profile.
    // sessionID is harness-specific (Claude Code session ID, Cursor conversation ID, etc.).
    // opts carries optional configuration (session export path, API credentials,
    // and the snapshot id and skill dir recorded in the profile).
    Capture(sessionID string, opts CaptureOpts) (Profile, error)
}

type CaptureOpts struct {
    ExportFile    string `json:"export_file,omitempty"`      // ATIF export or session transcript path
    APIKey        string `json:"api_key,omitempty"`          // server API auth (Devin)
    SnapshotHash  string `json:"snapshot_hash"`              // git SHA or content hash of the skill being profiled
    SkillDir      string `json:"skill_dir"`                 // path to the skill being profiled
}
```

## Serialized profile format

The profile is a JSON document carrying the caller-supplied snapshot id. It is the integration point for F04.

```go
type Profile struct {
    Schema       string           `json:"schema"`          // "skill-architect/profile/v1"
    ProfiledAt   string           `json:"profiled_at"`      // ISO 8601 UTC
    Harness      string           `json:"harness"`
    SessionID    string           `json:"session_id"`
    SnapshotHash string           `json:"snapshot_hash"`    // caller-supplied id for the skill version
    SkillDir     string           `json:"skill_dir"`
    Capability   CapabilityReport `json:"capability"`

    Tokens          TokenResult       `json:"tokens"`
    ToolCalls       ToolCallResult    `json:"tool_calls"`
    SkillActivation ActivationResult  `json:"skill_activation"`
    Timing          TimingResult      `json:"timing"`
    Attribution     AttributionResult `json:"attribution"`
}

// Each count is a pointer: a count the export said nothing about has no key in
// the JSON at all, and a count read as zero has its key with 0 in it. Use
// profiler.Count(n) to set one. The key names are schema v1 and unchanged.
type TokenCounts struct {
    Input         *int `json:"input,omitempty"`
    Output        *int `json:"output,omitempty"`
    CacheRead     *int `json:"cache_read,omitempty"`
    CacheCreation *int `json:"cache_creation,omitempty"`
    Reasoning     *int `json:"reasoning,omitempty"`
}

type ToolCallEntry struct {
    Name      string `json:"name"`
    Timestamp string `json:"timestamp"`
    Success   bool   `json:"success"`
}

type ActivationEntry struct {
    SkillName string `json:"skill_name"`
    Timestamp string `json:"timestamp"`
    Trigger   string `json:"trigger,omitempty"`  // what triggered activation
}

type TimingData struct {
    StartTime string `json:"start_time"`
    EndTime   string `json:"end_time"`
    TotalMs   int64  `json:"total_ms"`
}

type AttributionData struct {
    // Maps each tool call or output to the skill that produced it.
    // For harnesses without skill-level attribution, this is unknown.
    Attributions []Attribution `json:"attributions,omitempty"`
}

type Attribution struct {
    Target    string `json:"target"`     // tool call ID or output identifier
    SkillName string `json:"skill_name"`
}
```

**Serialization rules:**
- Profile is JSON. Pretty-printed for human readability, but parsing is canonical.
- `schema: "skill-architect/profile/v1"` is required. Future versions bump the suffix.
- `snapshot_hash` is whatever the caller passed to `--snapshot`, copied into the profile verbatim. The profiler neither hashes nor validates `--skill-dir` against it; the caller owns that correspondence. F04 is expected to compare only profiles carrying the same `snapshot_hash`.
- A profile with all metrics `unknown` is valid — it honestly reports that no telemetry was available.
- A profile must round-trip: `Marshal → Unmarshal → Marshal` produces identical JSON (modulo key ordering).

## Adapter implementations (Slice 1 scope)

### Claude Code adapter (Slice 1)

**Telemetry surface:** OTel export via `CLAUDE_CODE_ENABLE_TELEMETRY=1` + `OTEL_*` env vars.

**Input format:** OTLP/JSON, as Claude Code emits it over `OTEL_EXPORTER_OTLP_PROTOCOL=http/json`. One `ExportMetricsServiceRequest` or `ExportLogsServiceRequest` per JSON object; a file may hold one object, one per line (NDJSON), or several concatenated, and all three read the same way. A top-level value that is not an object is not an export. README "Capturing an OTel export" documents the two routes that produce the file.

Within that envelope:
- Metrics are walked at every level — `resourceMetrics[].scopeMetrics[].metrics[]` — not just the first of each. Token counts come from `claude_code.token.usage` sum data points: the `type` attribute (`input`, `output`, `cacheRead`, `cacheCreation`, camelCase on the wire, snake_case in the profile) and the value from `asDouble`, falling back to `asInt`. Every 64-bit integer is accepted as a JSON number or a decimal string, as the OTLP spec requires. A fractional `asDouble` is rounded, not truncated.
- **A value reaches the profile only if it is a count**: a whole number from 0 to 2⁶³−1. The rule is applied to the *rounded* value, so a `-0.4` that is float drift on a true zero is the zero it was, while `-0.5` is a number the exporter meant to be negative. A data point that fails this splits two ways, counted apart because they are two different things to go and look at in a capture. A leaf that **decoded as a finite number** whose rounded value is negative or 2⁶³ or greater *is not a count*: it is refused rather than converted — converting it is undefined in Go and gave the same export two different profiles on two architectures — and `asInt` is not consulted after it, because `asDouble` already answered. A leaf that **did not decode as a finite number at all** is *unreadable*, not "not a count": `NaN`, `Infinity` and a literal that overflows `float64` are refused by the float reader itself, so `asDouble` said nothing and `asInt` is read next — `{"asDouble":"NaN","asInt":"9"}` is the count 9, not a refusal. A point is unreadable only when neither leaf yields a number, which is the clause the reason uses: "carried no asDouble or asInt value that reads as a number". An `asDouble` is bounded at 2⁶³ and not at 2⁶³−1 because no `float64` is 2⁶³−1: the literal `9223372036854775807` parses to exactly 2⁶³, and only `asInt` can carry that count exactly. A token type that no data point carried has no key in the profile at all; a type read as zero has its key, with `0` in it.
- Data points are merged per time series, keyed by their full attribute set — every attribute, of every `AnyValue` kind, and not the subset that renders as text. `aggregationTemporality` 1 (delta) means the points add up; 2 (cumulative) means each is a running total, so the latest by `timeUnixNano` supersedes the rest. A temporality that reads as neither is refused rather than assumed, and absent, unreadable and declared-but-neither are counted apart, because they are three different things to look at in a capture.
- Log records are identified by their `body` (the fully-qualified event name, e.g. `claude_code.tool_result`), falling back to the `event.name` attribute (the short form). A body may be an object, a bare string, or absent. Unknown event names are ignored.
- Timestamps are read from `timeUnixNano`, nanoseconds since the epoch as a decimal string or a number, and recorded in the profile as RFC 3339 timestamps.
- **Data points and log records that cannot be read are skipped, and the profile does not report how many.** A `present` result carries no reason in schema v1; surfacing a count of skipped records is tracked for 0.5.0.

**Failure classification:** `error` means the file could not be read as an OTLP/JSON export — unreadable, empty or whitespace-only, a top-level value that is not a JSON object, malformed JSON, or a value that does not fit the OTLP schema. No reason quotes a Go type. Two of those five sit inside a batch and name it (1-based): malformed JSON, and a value that does not fit the OTLP schema — which also names the field path it found the wrong type at, when the decoder reports one. A batch whose own top level is the wrong shape (an array where the envelope should be) has no field path inside it to name, and that reason names the batch alone. Malformed JSON is the only one with a single byte to point at, so it is the only reason that carries an offset; the rest name the shape or the failure, because no offset would mean anything for them. A parse failure is never swallowed: nothing from earlier batches is used, and past batch 1 the reason says what to do about a capture stopped mid-write.

**One convention for byte offsets:** the 0-based offset, in the capture file, of the first byte the decoder could not accept — counting the byte-order mark and any leading whitespace, because the reader's editor counts them. When the file ended mid-object there is no such byte, and the reason names the file's length, which is where the byte the decoder wanted would have been.

`unknown` means a well-formed OTLP/JSON export that carries no telemetry — for one signal, or, when no batch carries an OTLP envelope, for all three. **The envelope test is a value, not a key**: a batch carries an envelope when `resourceMetrics` or `resourceLogs` has a *value* under it. An empty list is a value, so `{"resourceMetrics":[]}` *is* an OTLP export of a session that emitted nothing, and each signal is `unknown` with its own reason. A JSON `null` is not a value — ProtoJSON reads `null` as the field's default, so a batch written `{"resourceMetrics":null}` said nothing about metrics — and a file whose batches carry a value for neither field is not a broken export, it is not an export: all three signals get the "this is not OTLP/JSON" reason. Naming the key is not enough, and `{"resourceMetrics":null}` is the case that tells the two apart.

**Timing is a span over API requests.** `start_time`, `end_time` and `total_ms` cover the earliest to the latest `claude_code.api_request` record, so they exclude the prompt before the first request and any tool activity after the last one. Per-request `duration_ms` and the `claude_code.active_time.total` metric measure different quantities and have no field in schema v1; both are tracked for 0.5.0.

**Probe logic:**
1. Probe resolves the export file exactly as capture does — one read, one parse, one extractor per signal — and reports a capability as `otel` only when that extractor produced a value:
   - a `claude_code.token.usage` sum declaring an `aggregationTemporality` of 1 or 2 — spelled as a bare number, as a quoted digit, or as the protobuf JSON enum name (`AGGREGATION_TEMPORALITY_DELTA`, `AGGREGATION_TEMPORALITY_CUMULATIVE`; `AGGREGATION_TEMPORALITY_UNSPECIFIED` reads as 0) — with a data point carrying a recognised `type` (`input`, `output`, `cacheRead`, `cacheCreation`) and an `asDouble` or `asInt` value that is a count → `tokens: otel`
   - `claude_code.tool_result` log events carrying a `tool_name` and a readable `success`, or `claude_code.tool_decision` events recording a reject with a `tool_name` → `tool_calls: otel`
   - `claude_code.api_request` log events carrying a parseable `timeUnixNano` → `timing: otel`
   If the file is absent, unreadable, malformed, missing the signal, or carrying the signal with nothing readable inside it, that capability stays `none`. Availability is a fact about a value in hand, not about a name matched in a file — a probe that reports structure is how it comes to advertise data the capture cannot deliver.
2. Skill activation and attribution are always `none` for Claude Code, for two different reasons. Activation is `none` because this adapter does not read the telemetry, not because the telemetry is absent: Claude Code logs a `claude_code.skill_activated` event whenever a skill is invoked — through the Skill tool or a `/` command — carrying `skill.name`, `invocation_trigger`, `skill.source` and `skill.kind`, and attaches `skill.name` to `token.usage`, `cost.usage`, `api_request`, `api_error` and `api_refusal` besides. Attribution is `none` because there is nothing to read: its telemetry carries no output-to-skill mapping at all. A reason may say what this adapter does not read; it may not say what the harness does not emit unless that is true.

**Capture logic:**
1. Refuse a supplied `CaptureOpts.ExportFile` before doing anything else, with the same error the CLI gives for `--export-file`, rather than silently ignoring it: the adapter owns its input contract, and a caller who supplies a session export is told it is not read instead of receiving a profile that looks like missing telemetry. Then resolve the export file once. That resolution owns failure classification for all three OTel signals: no file configured → each is `unknown` with the fallback reason below; the file cannot be read as an OTLP/JSON export → each is `error` naming the failure; it parses but carries no OTLP envelope → each is `unknown` naming the format expected; otherwise the parsed export goes to the extractors.
2. Extract `claude_code.token.usage` sum data points → `TokenCounts`, counting only those carrying a recognised `type` and a numeric value, merged per series according to the aggregation temporality above.
3. Extract `ToolCallEntry` list from two disjoint sources. `claude_code.tool_result` is emitted when a tool completes and only then, so it supplies the calls that ran: one entry per event carrying both a `tool_name` and a readable `success`, and `ToolCallEntry.Success` means the tool ran and succeeded. `claude_code.tool_decision` supplies the calls that were rejected and therefore never ran, listed with `success: false`. An accepted decision is not an entry — its outcome comes from its result, and an export captured mid-run simply does not list the calls whose results were not written yet. A `tool_result` whose `success` cannot be read is not an entry either: it is never recorded as a failed call. It is counted, and if the export yielded no tool calls at all that count reaches the `unknown` reason; where some calls were read the result is `present`, and a `present` result carries no reason, so the profile does not say how many records were skipped (see the bullet above). Entries sort by timestamp, and one whose `timeUnixNano` could not be read is kept, last, with an empty `timestamp`.
4. Extract timing from `claude_code.api_request` log events → `TimingData` spanning the earliest to the latest parseable timestamp, so `end_time >= start_time` and `total_ms >= 0` whatever order the exporter wrote the events in.
5. Steps 2–4 settle their own signal independently: `present` with `Source: "otel"` when the extractor read at least one usable value, `unknown` naming what was missing or unreadable when it did not. A partial export never discards the signals it does carry.
6. `SkillActivation` → `unknown` with reason "This adapter does not yet read Claude Code's skill telemetry: the claude_code.skill_activated event, logged when a skill is invoked through the Skill tool or a / command, carries skill.name, invocation_trigger, skill.source and skill.kind. Reading it is 0.5.0."
7. `Attribution` → `unknown` with reason "Claude Code telemetry carries no output-to-skill mapping".
8. The profile's `capability` block is derived from the same resolution, so probe and capture cannot disagree about what the export yielded.
9. Serialize to `Profile` JSON.

**CLI exit status.** `profiler capture` writes the profile to stdout in every case, and exits **2** when no signal was read and at least one is `error` — a supplied export that could not be used — and **0** otherwise, including an all-`unknown` profile from a session with no telemetry configured. The status is what a wrapping script branches on; exiting 0 after reading nothing would have it store the all-unknown profile as a successful capture. Every usage error exits **1** before any capture happens: an unknown command or harness, a missing required flag, `--export-file`, and an unrecognised flag. The last one is why the flag sets use `ContinueOnError` — `flag.ExitOnError` exits 2 on its own, and 2 has to mean exactly one thing for a script to branch on it.

**Fallback:** With no export file configured, `tokens`, `tool_calls`, and `timing` are `unknown` with reason "OTel export not configured. Provide an OTel export file via --otel-file or OtelExportFile." A file that is configured but cannot be read as an OTLP/JSON export must not borrow that reason: missing, unreadable, empty, non-object at the top level, malformed, or not fitting the OTLP schema are all `error`, naming the failure, because the caller did supply a file and "not configured" would send them to fix the one thing that is not wrong. A file that parses but carries nothing readable for a signal leaves that signal `unknown`, naming what was missing; a file carrying no OTLP envelope at all leaves all three `unknown`, naming the format expected. `skill_activation` and `attribution` are `unknown` with their own reasons (steps 6 and 7 above) in every one of these cases — they are a property of the harness and of this adapter, not of the export. The adapter reads only the file it is given: it does not inspect `CLAUDE_CODE_ENABLE_TELEMETRY` or the `OTEL_*` env vars itself.

## Acceptance criteria (Slice 1)

1. Metric result serialization: a `present` result includes value and source; an `unknown` result includes reason and no value key; an `error` result includes reason and no value key.
2. `CapabilityReport` for Claude Code is per signal: `tokens`, `tool_calls`, and `timing` are each `otel` only when the export file yields a readable value for that signal, and `none` otherwise; `skill_activation` and `attribution` are always `none`. An export yielding all three therefore reports `tokens: otel`, `tool_calls: otel`, `timing: otel`.
3. `CapabilityReport` for Claude Code without OTel: all metrics `none`. Same for an export file that cannot be read as OTLP/JSON, that carries no OTLP envelope, that carries the envelope with nothing in it, that carries none of the three signals, or that carries one with nothing readable inside it — a `token.usage` data point with an unrecognised `type`, no readable value (absent, non-numeric, or `NaN`/`Infinity`, which the float leaf refuses), a value that is not a count (a finite number that rounds negative or to 2⁶³ or beyond), or a temporality that is absent, unreadable, or neither delta nor cumulative; a `token.usage` metric that arrives as a gauge rather than a sum; a `tool_result` with no `tool_name` or no readable `success`; a `tool_decision` with no recognised `decision`, or a reject with no `tool_name`; an `api_request` with no parseable `timeUnixNano`.
4. A Claude Code session whose export carries all three signals produces a profile where tokens, tool_calls, and timing are `present`; skill_activation and attribution are `unknown` with reason.
5. A session with no OTel export file produces a profile where `tokens`, `tool_calls`, and `timing` are `unknown` with the reason quoted under **Fallback** above, which begins "OTel export not configured"; `skill_activation` and `attribution` are `unknown` with their own reasons, as they are in every other case.
6. Profile JSON round-trips: `Marshal → Unmarshal → Marshal` is identical.
7. Profile `schema` field is `"skill-architect/profile/v1"`.
8. Profile `snapshot_hash` matches the input.
9. Probe and capture agree: every signal the capability report marks available is `present` in the profile with that source and carries a value, and every signal it marks `none` is not `present`, carries a reason, and carries no value. This holds for partial exports — an export with tool calls and timing but no token metric captures both, with tokens `unknown`.
10. A `present` signal means a value was read. An export carrying a signal's structure with nothing readable inside it yields `unknown` with a reason, never `present` without a value read from the export. A read zero is a value: a single `api_request` is a zero-length span and yields `total_ms: 0`, `present`, and a token type read as zero yields that key with `0` in it. A token type the export said nothing about has no key at all, so a reader can tell the two apart.
11. The same export produces the same profile on any architecture. A value that cannot be represented as a count is refused rather than converted, because an out-of-range float-to-integer conversion is undefined in Go and gave arm64 and amd64 different answers for one file.
12. An export file that is supplied but cannot be read as an OTLP/JSON export — unreadable, empty, a non-object top-level value, malformed, or carrying a value that does not fit the OTLP schema — yields `error` for `tokens`, `tool_calls`, and `timing`, each naming the failure. `MetricError` is reachable and covered by tests.
13. An export file that parses but whose batches carry a *value* for neither `resourceMetrics` nor `resourceLogs` yields `unknown` for all three, naming the format expected — never `error`, and never a silent zero. The test is value presence, not key presence: `{"resourceMetrics":null}` carries no value, because ProtoJSON reads `null` as the field default, and gets the format reason. An empty list is a value, so an export carrying either field with an empty list under it is an OTLP export of a session that emitted nothing: each signal is `unknown` with its own reason, not with the format one.
14. `profiler capture` exits 2 when no signal was read and at least one is `error`, and 0 otherwise; the profile is written to stdout either way.
