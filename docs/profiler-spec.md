# Profiler adapter interface spec (F03)

**Status:** spec · **Date:** 2026-09-10 · **Architecture:** Option C — adapter-per-harness with capability negotiation

## Purpose

A harness-agnostic profiler that captures runtime signals from any target harness (Cursor, Claude Code, Codex, Devin), degrades gracefully to `unknown` for unavailable metrics, and produces a serialized profile pinned to a snapshot hash that F04 (paired comparisons) can read and recompute.

## Core types

### MetricResult<T>

Every metric is wrapped so unavailable data is explicit, not silent:

```go
type MetricState string

const (
    MetricPresent  MetricState = "present"   // value captured from telemetry
    MetricUnknown  MetricState = "unknown"   // no telemetry source available
    MetricError    MetricState = "error"     // source existed but failed
)

type MetricResult[T any] struct {
    State   MetricState `json:"state"`
    Value   T           `json:"value,omitempty"`     // present when State == "present"
    Reason  string      `json:"reason,omitempty"`    // present when State != "present"
    Source  string      `json:"source,omitempty"`    // "otel" | "hooks" | "session_data" | "server_api" | "sqlite"
}
```

**Rules:**
- `State == "present"` → `Value` must be populated, `Reason` must be empty.
- `State == "unknown"` → `Value` must be zero-valued, `Reason` explains why (e.g., "no OTel export configured", "harness has no skill activation events").
- `State == "error"` → `Value` must be zero-valued, `Reason` describes the failure (e.g., "OTel collector unreachable: connection refused").
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

**Capability vs result:** `CapabilityReport` says what the adapter *can* produce. `MetricResult` says what it *did* produce for a specific session. An adapter may declare `tokens: "otel"` in its capability report but return `MetricError` for a specific session if OTel was misconfigured.

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
    OtelEndpoint  string `json:"otel_endpoint,omitempty"`   // OTLP receiver URL
    ExportFile    string `json:"export_file,omitempty"`      // ATIF export or session transcript path
    APIKey        string `json:"api_key,omitempty"`          // server API auth (Devin)
    SnapshotHash  string `json:"snapshot_hash"`              // git SHA or content hash of the skill being profiled
    SkillDir      string `json:"skill_dir"`                 // path to the skill being profiled
}
```

## Serialized profile format

The profile is a JSON document pinned to a snapshot hash. It is the integration point for F04.

```go
type Profile struct {
    Schema       string           `json:"schema"`          // "skill-architect/profile/v1"
    ProfiledAt   string           `json:"profiled_at"`      // ISO 8601 UTC
    Harness      string           `json:"harness"`
    SessionID    string           `json:"session_id"`
    SnapshotHash string           `json:"snapshot_hash"`    // pins to skill version
    SkillDir     string           `json:"skill_dir"`
    Capability   CapabilityReport `json:"capability"`

    Tokens          MetricResult[TokenCounts]      `json:"tokens"`
    ToolCalls       MetricResult[[]ToolCallEntry]   `json:"tool_calls"`
    SkillActivation MetricResult[[]ActivationEntry] `json:"skill_activation"`
    Timing          MetricResult[TimingData]        `json:"timing"`
    Attribution     MetricResult[AttributionData]   `json:"attribution"`
}

type TokenCounts struct {
    Input      int `json:"input"`
    Output     int `json:"output"`
    CacheRead  int `json:"cache_read,omitempty"`
    CacheWrite int `json:"cache_write,omitempty"`   // upstream "cache_write"; "cache_creation" accepted on input
    Reasoning  int `json:"reasoning,omitempty"`
}

type ToolCallEntry struct {
    Name      string `json:"name"`
    Timestamp string `json:"timestamp"`
    Success   bool   `json:"success"`
    ErrorType string `json:"error_type,omitempty"`    // failure is the presence of an error type
    Count     int    `json:"count,omitempty"`         // aggregate count for delta metrics; absent = 1 call
    ID        string `json:"id,omitempty"`            // tool_use_id when the source supplies one
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
    Target        string `json:"target"`     // tool call ID or output identifier
    SkillName     string `json:"skill_name"`
    Category      string `json:"category,omitempty"`       // skill | mcp | cli | subagent | tool
    Detail        string `json:"detail,omitempty"`         // e.g. "file" for path-inferred activation
    OperationName string `json:"operation_name,omitempty"` // e.g. "execute_tool"
    Confidence    string `json:"confidence,omitempty"`     // inferred | observed
}

// Separately-labelled estimate field — never a billed count:
type EstimatedTokens struct {
    Total int64 `json:"total"`              // chars/4 over hook payloads, source "hooks_estimated"
}
// Profile.EstimatedContextTokens json:"estimated_context_tokens,omitempty"
```

**Serialization rules:**
- Profile is JSON. Pretty-printed for human readability, but parsing is canonical.
- `schema: "skill-architect/profile/v1"` is required. Future versions bump the suffix.
- `snapshot_hash` pins the profile to a specific skill version. F04 revalidates against this before comparing.
- A profile with all metrics `unknown` is valid — it honestly reports that no telemetry was available.
- A profile must round-trip: `Marshal → Unmarshal → Marshal` produces identical JSON (modulo key ordering).
- A profile that marks a metric `present` without a `value` is **invalid**; `LoadProfile` rejects it.

**Comparison contract** (`skill-architect/comparison/v1`):
- Every `MetricComparison` carries `baseline_source` and `candidate_source` — both sides, always.
- A delta across different sources is refused (`comparable: false`, reason names both sources).
  Cross-tier subtraction is never a delta.
- `estimated_context_tokens` deltas carry `note: "relative estimate, not billed — ordinal signal
  only"` and must never be rendered as dollars.
- `skill_activation` is compared as a set of skill names (`only_in_baseline` /
  `only_in_candidate`), not a bare count.

## Adapter implementations (Slice 1 scope)

### Claude Code adapter (Slice 1)

**Telemetry surface:** OTel export via `CLAUDE_CODE_ENABLE_TELEMETRY=1` + `OTEL_*` env vars.

**Probe logic:**
1. If `OtelExportFile` is set and the file exists, parse it. For each metric category, set `otel` only if the file contains the corresponding signal:
   - `claude_code.token.usage` metric → `tokens: otel`
   - `claude_code.tool_decision` log events → `tool_calls: otel`
   - `claude_code.api_request` log events → `timing: otel`
   - `skill.name` attribute on `claude_code.token.usage`/`claude_code.cost.usage` → `skill_activation: otel`
   If the file is empty, malformed, or missing the relevant signal, that capability stays `none`.
2. Attribution is always `none` for Claude Code — `skill.name` marks which skill was active for a request (cost-when-active), not which output it produced.

**Capture logic:**
1. Read OTel metrics from the configured endpoint (or a file the collector writes).
2. Extract `claude_code.token.usage` metrics → `TokenCounts`.
3. Extract `claude_code.tool_decision` log events → `ToolCallEntry` list.
4. Extract timing from `claude_code.api_request` log events → `TimingData`.
5. `SkillActivation` → one `ActivationEntry` per distinct `skill.name` on token/cost usage metrics, `present`/`otel`; `unknown` when no `skill.name` attributes exist.
6. `Attribution` → `MetricUnknown` with reason "Claude Code does not attribute outputs to skills".
7. Wrap all results in `MetricResult` with `Source: "otel"`.
8. Serialize to `Profile` JSON.

**Fallback:** If OTel is not configured, all metrics are `unknown` with reason "OTel export not configured. Set CLAUDE_CODE_ENABLE_TELEMETRY=1 and OTEL_METRICS_EXPORTER=otlp."

## Acceptance criteria (Slice 1)

1. `MetricResult[T]` serialization: a `present` result includes value and source; an `unknown` result includes reason and no value; an `error` result includes reason and no value.
2. `CapabilityReport` for Claude Code with OTel configured: `tokens: otel`, `tool_calls: otel`, `timing: otel`, `skill_activation: otel` when `skill.name` attributes are present (else `none`), `attribution: none`.
3. `CapabilityReport` for Claude Code without OTel: all metrics `none`.
4. A Claude Code session with OTel produces a profile where tokens, tool_calls, and timing are `present`; skill_activation is `present` when the export carries `skill.name`, otherwise `unknown`; attribution is `unknown` with reason.
5. A session with no OTel produces a profile where all metrics are `unknown` with reason "OTel export not configured."
6. Profile JSON round-trips: `Marshal → Unmarshal → Marshal` is identical.
7. Profile `schema` field is `"skill-architect/profile/v1"`.
8. Profile `snapshot_hash` matches the input.
