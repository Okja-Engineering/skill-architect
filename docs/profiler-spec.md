# Profiler adapter interface spec (F03)

**Status:** spec · **Date:** 2026-09-10 · **Architecture:** Option C — adapter-per-harness with capability negotiation

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
    MetricPresent MetricState = "present" // value captured from telemetry
    MetricUnknown MetricState = "unknown" // no telemetry source available
    MetricError   MetricState = "error"   // source existed but failed
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
hand: `PresentTokenResult`/`UnknownTokenResult`/`ErrorTokenResult` and the
matching `Present…`/`Unknown…` pair for each other category.

**Rules:**
- `State == "present"` → `Value` must be populated, `Reason` must be empty.
- `State == "unknown"` → `Value` is nil and its key is absent from the JSON; `Reason` explains why (e.g., "no `claude_code.token.usage` metric found in OTel export", "Claude Code has no skill-level activation events").
- `State == "error"` → `Value` is nil and its key is absent from the JSON; `Reason` describes the failure (e.g., "failed to parse OTel export JSON: …").
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

// AnySource reports whether probing found any telemetry source at all. Capture
// uses it to decide whether there is an export worth reading; no adapter may use
// a single metric's capability as a proxy for the whole export.
func (c CapabilityReport) AnySource() bool

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

**Capability vs result:** `CapabilityReport` says what the adapter *can* produce. A metric result says what it *did* produce for a specific session. An adapter may declare `tokens: "otel"` in its capability report but return `MetricError` for a specific session if OTel was misconfigured.

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
    // opts carries optional configuration (OTel endpoint, export file path, API credentials).
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

type TokenCounts struct {
    Input         int `json:"input"`
    Output        int `json:"output"`
    CacheRead     int `json:"cache_read,omitempty"`
    CacheCreation int `json:"cache_creation,omitempty"`
    Reasoning     int `json:"reasoning,omitempty"`
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

**Probe logic:**
1. If `OtelExportFile` is set and the file exists, parse it. For each metric category, set `otel` only if the file contains the corresponding signal:
   - `claude_code.token.usage` metric → `tokens: otel`
   - `claude_code.tool_decision` log events → `tool_calls: otel`
   - `claude_code.api_request` log events → `timing: otel`
   If the file is empty, malformed, or missing the relevant signal, that capability stays `none`.
2. Skill activation and attribution are always `none` for Claude Code: its OTel surface has no activation event and no output-to-skill mapping. The `skill.name` attribute on token and cost metrics is not read yet.

**Capture logic:**
1. Probe first. If `AnySource()` is false there is nothing to read: every metric is `unknown` with the fallback reason below, and capture stops. Otherwise read the export file.
2. Extract `claude_code.token.usage` metrics → `TokenCounts`.
3. Extract `claude_code.tool_decision` log events → `ToolCallEntry` list.
4. Extract timing from `claude_code.api_request` log events → `TimingData`.
5. Steps 2–4 settle their own signal independently: `present` with `Source: "otel"` when the export carries that signal, `unknown` naming the missing signal when it does not. A partial export never discards the signals it does carry.
6. `SkillActivation` → `unknown` with reason "Claude Code has no skill-level activation events".
7. `Attribution` → `unknown` with reason "Claude Code does not attribute outputs to skills".
8. Serialize to `Profile` JSON.

**Fallback:** If no export file is configured, or the file is missing, unreadable, malformed, or carries none of the three signals, all metrics are `unknown` with reason "OTel export not configured. Provide an OTel export file via --otel-file or OtelExportFile." The adapter reads only the file it is given — it does not inspect `CLAUDE_CODE_ENABLE_TELEMETRY` or the `OTEL_*` env vars itself.

## Acceptance criteria (Slice 1)

1. Metric result serialization: a `present` result includes value and source; an `unknown` result includes reason and no value key; an `error` result includes reason and no value key.
2. `CapabilityReport` for Claude Code is per signal: `tokens`, `tool_calls`, and `timing` are each `otel` only when the export file actually contains that signal, and `none` otherwise; `skill_activation` and `attribution` are always `none`. An export carrying all three therefore reports `tokens: otel`, `tool_calls: otel`, `timing: otel`.
3. `CapabilityReport` for Claude Code without OTel: all metrics `none`. Same for an export file that is empty, malformed, or carries none of the three signals.
4. A Claude Code session whose export carries all three signals produces a profile where tokens, tool_calls, and timing are `present`; skill_activation and attribution are `unknown` with reason.
5. A session with no OTel produces a profile where all metrics are `unknown` with the reason quoted under **Fallback** above, which begins "OTel export not configured."
6. Profile JSON round-trips: `Marshal → Unmarshal → Marshal` is identical.
7. Profile `schema` field is `"skill-architect/profile/v1"`.
8. Profile `snapshot_hash` matches the input.
9. Probe and capture agree: every signal the capability report marks available is `present` in the profile with that source, and every signal it marks `none` is `unknown` with a reason. This holds for partial exports — an export with tool calls and timing but no token metric captures both, with tokens `unknown`.
