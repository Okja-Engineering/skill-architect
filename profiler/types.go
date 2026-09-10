// Package profiler defines the harness-agnostic profiler adapter interface
// and the serialized profile format. Each harness implements ProfilerAdapter;
// the comparison engine (F04) reads profiles without knowing which adapter produced them.
package profiler

import "encoding/json"

// MetricState represents the availability state of a metric.
type MetricState string

const (
	MetricPresent MetricState = "present" // value captured from telemetry
	MetricUnknown MetricState = "unknown" // no telemetry source available
	MetricError   MetricState = "error"   // source existed but failed
)

// MetricResult wraps every metric so unavailable data is explicit, not silent.
// Generics are avoided for JSON marshal/unmarshal simplicity; callers use typed
// wrappers (TokenResult, ToolCallResult, etc.) that embed RawMetricResult.
type RawMetricResult struct {
	State  MetricState `json:"state"`
	Reason string      `json:"reason,omitempty"` // present when State != "present"
	Source string      `json:"source,omitempty"` // "otel" | "hooks" | "session_data" | "server_api" | "sqlite"
}

// MetricName identifies a runtime signal category.
type MetricName string

const (
	MetricTokens          MetricName = "tokens"
	MetricToolCalls       MetricName = "tool_calls"
	MetricSkillActivation MetricName = "skill_activation"
	MetricTiming          MetricName = "timing"
	MetricAttribution     MetricName = "attribution"
)

// MetricSource identifies where a metric value came from.
type MetricSource string

const (
	SourceOtel        MetricSource = "otel"
	SourceHooks       MetricSource = "hooks"
	SourceSessionData MetricSource = "session_data"
	SourceServerAPI   MetricSource = "server_api"
	SourceSQLite      MetricSource = "sqlite"
	SourceNone        MetricSource = "none"
)

// CapabilityReport declares what an adapter can produce after probing its environment.
type CapabilityReport struct {
	Harness      string                      `json:"harness"` // "cursor" | "claude_code" | "codex" | "devin"
	AdapterVer   string                      `json:"adapter_version"`
	ProbedAt     string                      `json:"probed_at"` // ISO 8601 UTC
	Capabilities map[MetricName]MetricSource `json:"capabilities"`
}

// CaptureOpts carries optional configuration for a capture session.
type CaptureOpts struct {
	OtelEndpoint string `json:"otel_endpoint,omitempty"` // OTLP receiver URL
	ExportFile   string `json:"export_file,omitempty"`   // ATIF export or session transcript path
	APIKey       string `json:"api_key,omitempty"`       // server API auth (Devin)
	SnapshotHash string `json:"snapshot_hash"`           // git SHA or content hash of the skill being profiled
	SkillDir     string `json:"skill_dir"`               // path to the skill being profiled
}

// ProfilerAdapter is implemented by each harness adapter.
type ProfilerAdapter interface {
	// Name returns the harness identifier.
	Name() string

	// Probe inspects the environment and returns what metrics this adapter can produce.
	// Must not fail — if nothing is available, returns a report with all "none".
	Probe() CapabilityReport

	// Capture reads telemetry for a specific session and produces a Profile.
	Capture(sessionID string, opts CaptureOpts) (Profile, error)
}

// TokenCounts holds per-session token usage.
type TokenCounts struct {
	Input         int `json:"input"`
	Output        int `json:"output"`
	CacheRead     int `json:"cache_read,omitempty"`
	CacheCreation int `json:"cache_creation,omitempty"`
	Reasoning     int `json:"reasoning,omitempty"`
}

// ToolCallEntry records a single tool invocation.
type ToolCallEntry struct {
	Name      string `json:"name"`
	Timestamp string `json:"timestamp"`
	Success   bool   `json:"success"`
}

// ActivationEntry records a skill activation event.
type ActivationEntry struct {
	SkillName string `json:"skill_name"`
	Timestamp string `json:"timestamp"`
	Trigger   string `json:"trigger,omitempty"`
}

// TimingData holds session timing.
type TimingData struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	TotalMs   int64  `json:"total_ms"`
}

// Attribution maps outputs to the skills that produced them.
type AttributionData struct {
	Attributions []Attribution `json:"attributions,omitempty"`
}

// Attribution links a target (tool call ID or output) to a skill.
type Attribution struct {
	Target    string `json:"target"`
	SkillName string `json:"skill_name"`
}

// TokenResult is a typed MetricResult for token counts.
// Value is a pointer so omitempty works — nil means no value (unknown/error states).
type TokenResult struct {
	RawMetricResult
	Value *TokenCounts `json:"value,omitempty"`
}

// ToolCallResult is a typed MetricResult for tool call entries.
// Value is a pointer so omitempty works — nil means no value (unknown/error states).
type ToolCallResult struct {
	RawMetricResult
	Value []ToolCallEntry `json:"value,omitempty"`
}

// ActivationResult is a typed MetricResult for skill activation events.
type ActivationResult struct {
	RawMetricResult
	Value []ActivationEntry `json:"value,omitempty"`
}

// TimingResult is a typed MetricResult for timing data.
// Value is a pointer so omitempty works — nil means no value (unknown/error states).
type TimingResult struct {
	RawMetricResult
	Value *TimingData `json:"value,omitempty"`
}

// AttributionResult is a typed MetricResult for attribution data.
type AttributionResult struct {
	RawMetricResult
	Value *AttributionData `json:"value,omitempty"`
}

// Profile is the serialized, snapshot-pinned artifact that F04 reads.
type Profile struct {
	Schema       string           `json:"schema"` // "skill-architect/profile/v1"
	ProfiledAt   string           `json:"profiled_at"`
	Harness      string           `json:"harness"`
	SessionID    string           `json:"session_id"`
	SnapshotHash string           `json:"snapshot_hash"`
	SkillDir     string           `json:"skill_dir"`
	Capability   CapabilityReport `json:"capability"`

	Tokens          TokenResult       `json:"tokens"`
	ToolCalls       ToolCallResult    `json:"tool_calls"`
	SkillActivation ActivationResult  `json:"skill_activation"`
	Timing          TimingResult      `json:"timing"`
	Attribution     AttributionResult `json:"attribution"`
}

// ProfileSchema is the version string embedded in every profile.
const ProfileSchema = "skill-architect/profile/v1"

// AdapterVersion is the current adapter implementation version.
const AdapterVersion = "0.1.0"

// MarshalJSON for Profile ensures the schema field is always set.
func (p Profile) MarshalJSON() ([]byte, error) {
	type Alias Profile
	a := Alias(p)
	a.Schema = ProfileSchema
	return json.Marshal(a)
}

// --- TokenResult constructors ---

// PresentTokenResult creates a TokenResult with state "present".
func PresentTokenResult(v TokenCounts, source string) TokenResult {
	return TokenResult{
		RawMetricResult: RawMetricResult{State: MetricPresent, Source: source},
		Value:           &v,
	}
}

// UnknownTokenResult creates a TokenResult with state "unknown".
func UnknownTokenResult(reason string) TokenResult {
	return TokenResult{RawMetricResult: RawMetricResult{State: MetricUnknown, Reason: reason}}
}

// ErrorTokenResult creates a TokenResult with state "error".
func ErrorTokenResult(reason string) TokenResult {
	return TokenResult{RawMetricResult: RawMetricResult{State: MetricError, Reason: reason}}
}

// --- ToolCallResult constructors ---

// PresentToolCallResult creates a ToolCallResult with state "present".
func PresentToolCallResult(v []ToolCallEntry, source string) ToolCallResult {
	return ToolCallResult{
		RawMetricResult: RawMetricResult{State: MetricPresent, Source: source},
		Value:           v,
	}
}

// UnknownToolCallResult creates a ToolCallResult with state "unknown".
func UnknownToolCallResult(reason string) ToolCallResult {
	return ToolCallResult{RawMetricResult: RawMetricResult{State: MetricUnknown, Reason: reason}}
}

// --- ActivationResult constructors ---

// PresentActivationResult creates an ActivationResult with state "present".
func PresentActivationResult(v []ActivationEntry, source string) ActivationResult {
	return ActivationResult{
		RawMetricResult: RawMetricResult{State: MetricPresent, Source: source},
		Value:           v,
	}
}

// UnknownActivationResult creates an ActivationResult with state "unknown".
func UnknownActivationResult(reason string) ActivationResult {
	return ActivationResult{RawMetricResult: RawMetricResult{State: MetricUnknown, Reason: reason}}
}

// --- TimingResult constructors ---

// PresentTimingResult creates a TimingResult with state "present".
func PresentTimingResult(v TimingData, source string) TimingResult {
	return TimingResult{
		RawMetricResult: RawMetricResult{State: MetricPresent, Source: source},
		Value:           &v,
	}
}

// UnknownTimingResult creates a TimingResult with state "unknown".
func UnknownTimingResult(reason string) TimingResult {
	return TimingResult{RawMetricResult: RawMetricResult{State: MetricUnknown, Reason: reason}}
}

// --- AttributionResult constructors ---

// PresentAttributionResult creates an AttributionResult with state "present".
func PresentAttributionResult(v AttributionData, source string) AttributionResult {
	return AttributionResult{
		RawMetricResult: RawMetricResult{State: MetricPresent, Source: source},
		Value:           &v,
	}
}

// UnknownAttributionResult creates an AttributionResult with state "unknown".
func UnknownAttributionResult(reason string) AttributionResult {
	return AttributionResult{RawMetricResult: RawMetricResult{State: MetricUnknown, Reason: reason}}
}
