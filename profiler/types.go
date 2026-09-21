// Package profiler defines the harness-agnostic profiler adapter interface
// and the serialized profile format. Each harness implements ProfilerAdapter;
// the comparison engine (F04) reads profiles without knowing which adapter produced them.
package profiler

import (
	"encoding/json"
	"fmt"
)

// MetricState represents the availability state of a metric.
type MetricState string

const (
	MetricPresent MetricState = "present" // a value was read from telemetry
	MetricUnknown MetricState = "unknown" // no telemetry source, or nothing readable in it
	MetricError   MetricState = "error"   // the source existed and failed
)

// RawMetricResult is the state every metric result carries, so unavailable data
// is explicit rather than silent. There is no generic MetricResult[T]: type
// parameters are avoided for JSON marshal/unmarshal simplicity, and each metric
// category has its own wrapper (TokenResult, ToolCallResult, and the rest)
// embedding this.
type RawMetricResult struct {
	State  MetricState `json:"state"`
	Reason string      `json:"reason,omitempty"` // present when State != "present"
	Source string      `json:"source,omitempty"` // "otel" | "hooks" | "hooks_estimated" | "session_data" | "server_api" | "sqlite"
}

// MetricName identifies a runtime signal category.
type MetricName string

// Each name is also the profile's json key for that signal, so a capability
// report and a profile name the same signal the same way.
const (
	MetricTokens          MetricName = "tokens"
	MetricToolCalls       MetricName = "tool_calls"
	MetricSkillActivation MetricName = "skill_activation"
	MetricTiming          MetricName = "timing"
	MetricAttribution     MetricName = "attribution"
	// MetricEstimatedContextTokens is kept apart from MetricTokens because an
	// estimate is not a count: it is derived from payload sizes, never billed,
	// and a comparison that averaged the two together would report a number
	// with no unit.
	MetricEstimatedContextTokens MetricName = "estimated_context_tokens"
)

// MetricSource identifies where a metric value came from.
type MetricSource string

const (
	SourceOtel  MetricSource = "otel"
	SourceHooks MetricSource = "hooks"
	// SourceHooksEstimated marks a value derived by estimation over hook
	// payloads — chars/4 over what a hook carried — and never a measured or
	// billed count. It is a distinct source rather than a note on the reason
	// so that a reader grouping profiles by source cannot pool an estimate
	// with a measurement.
	SourceHooksEstimated MetricSource = "hooks_estimated"
	SourceSessionData    MetricSource = "session_data"
	SourceServerAPI      MetricSource = "server_api"
	SourceSQLite         MetricSource = "sqlite"
	SourceNone           MetricSource = "none"
)

// CapabilityReport declares what an adapter can produce after probing its environment.
type CapabilityReport struct {
	Harness      string                      `json:"harness"` // "cursor" | "claude_code" | "codex" | "devin"
	AdapterVer   string                      `json:"adapter_version"`
	ProbedAt     string                      `json:"probed_at"` // ISO 8601 UTC
	Capabilities map[MetricName]MetricSource `json:"capabilities"`
}

// CaptureOpts carries optional configuration for a capture session.
//
// ExportFile and APIKey are the input contract for adapters that do not exist
// yet, and no shipped adapter reads either: the Claude Code adapter refuses an
// ExportFile rather than ignoring it, and there is no CLI flag for an API key
// at all. They are kept because they are the shape the Devin and Cursor
// adapters need in 0.5.0, and removing them now would be a breaking change to
// this struct twice over.
type CaptureOpts struct {
	ExportFile   string `json:"export_file,omitempty"` // ATIF export or session transcript path; reserved for 0.5.0
	APIKey       string `json:"api_key,omitempty"`     // server API auth; reserved for the Devin and Cursor adapters in 0.5.0
	SnapshotHash string `json:"snapshot_hash"`         // git SHA or content hash of the skill being profiled
	SkillDir     string `json:"skill_dir"`             // path to the skill being profiled
}

// ExportFileUnsupportedError is the refusal for a CaptureOpts.ExportFile the
// selected adapter cannot read. The adapter owns the contract and returns this
// from Capture; the CLI raises the same error before it gets that far, so the
// two cannot drift into telling the caller different things.
func ExportFileUnsupportedError(harness string) error {
	return fmt.Errorf("--export-file is not read by the %s adapter; supply an OTel export with --otel-file", harness)
}

// SessionIDRequiredError is the refusal for a Capture with no session id.
//
// A capture reads only the records carrying the session it was asked for, so an
// empty id is not a session that matched nothing — it is no assertion, and a
// profile stamped with it could only report every session the export happens to
// carry. The adapter owns the contract and returns this from Capture; the CLI
// requires the same flag before it gets that far, so a library caller and a
// command-line caller are told the same thing.
func SessionIDRequiredError(harness string) error {
	return fmt.Errorf("--session is required by the %s adapter: a profile names one session and reads only the records carrying its session.id", harness)
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

// ProbeDiagnoser is implemented by an adapter that can say *why* a capability
// came back "none".
//
// A capability's vocabulary is a source or SourceNone, so a CapabilityReport
// structurally cannot distinguish "no telemetry was configured" from "the
// export you named could not be read" — both collapse to none. Capture keeps
// them apart, in a signal's MetricError state and its reason, and the two
// commands are documented to agree: capture delivers exactly what probe
// advertised, because both read the export through the same extractor. A probe
// that collapses an unreadable export into five nones and says nothing breaks
// that agreement in the one direction that misleads, since a caller reads it as
// a session with no telemetry rather than as a path they mistyped.
//
// The reason is reported here rather than added to CapabilityReport because
// that report is embedded in every Profile: widening it would change what a
// profile contains for the same input, which is an adapter-version and schema
// question. A diagnostic on a separate channel changes no output anyone parses.
type ProbeDiagnoser interface {
	// ProbeWithDiagnostics probes once and returns the report together with one
	// message per distinct reason a signal could not be read. It is empty when
	// nothing failed, including when no export was supplied at all — that is an
	// answer about the session, not a fault of the run.
	//
	// One resolve, not two: a diagnostic read separately could see a different
	// file from the report it is explaining and contradict it.
	ProbeWithDiagnostics() (CapabilityReport, []string)
}

// TokenCounts holds per-session token usage.
//
// Every count is a pointer because "the export said nothing about this" and
// "the export said zero" are different answers, and a profile that turns the
// first into the second reports a measurement nobody made — a cache-only
// export claiming the session used no input and no output tokens. A count that
// was not read has no key in the JSON at all; a count read as zero has its key,
// with 0 in it. The key names are schema v1 and unchanged.
type TokenCounts struct {
	Input         *int `json:"input,omitempty"`
	Output        *int `json:"output,omitempty"`
	CacheRead     *int `json:"cache_read,omitempty"`
	CacheCreation *int `json:"cache_creation,omitempty"`
	// Reasoning has no source in Claude Code's OTel surface: its token.usage
	// type attribute is exactly input, output, cacheRead and cacheCreation.
	// Left unpopulated there, so its key is absent rather than reporting a zero
	// that was never measured. The field is harness-agnostic and stays for an
	// adapter that does have the signal.
	Reasoning *int `json:"reasoning,omitempty"`
}

// Count is a token count a caller read, including a measured zero. Counts are
// pointers so an unread one can be told from a zero one; this is how a caller
// says "I read this".
func Count(n int) *int { return &n }

// ToolCallEntry records a single tool invocation.
//
// The three optional fields are absent when the source did not carry them, so
// a reader can tell "this harness does not report why a call failed" from "this
// call failed for no reason". Failure itself is Success, which every source can
// answer; ErrorType is the classification only some of them add.
type ToolCallEntry struct {
	Name      string `json:"name"`
	Timestamp string `json:"timestamp"`
	Success   bool   `json:"success"`
	// ErrorType classifies a failure the way the source did (e.g. "timeout"),
	// and is empty for a call that succeeded or a source that does not say.
	ErrorType string `json:"error_type,omitempty"`
	// Count carries the aggregate when the source is a delta-aggregated metric
	// rather than one record per call. Absent means one call, which is what
	// every per-record source reports.
	Count int `json:"count,omitempty"`
	// ID is the source's own identifier for the call (Claude Code's
	// tool_use_id), which is what an attribution targets.
	ID string `json:"id,omitempty"`
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
//
// Confidence is the honesty field: an attribution derived from a file path a
// skill happens to own is a guess, and one read from telemetry that names the
// skill is not. A reader that cannot tell them apart has to treat every
// attribution as either, which makes the whole signal unusable. It is optional
// because a source that only ever produces one kind has nothing to qualify.
type Attribution struct {
	Target        string `json:"target"`
	SkillName     string `json:"skill_name"`
	Category      string `json:"category,omitempty"`       // "skill" | "mcp" | "cli" | "subagent" | "tool"
	Detail        string `json:"detail,omitempty"`         // how it was derived, e.g. "file" for a path-inferred link
	OperationName string `json:"operation_name,omitempty"` // the source's operation, e.g. "execute_tool"
	Confidence    string `json:"confidence,omitempty"`     // "inferred" | "observed"
}

// TokenResult is the metric result for token counts.
// Value is a pointer so omitempty works — nil means no value (unknown/error states).
type TokenResult struct {
	RawMetricResult
	Value *TokenCounts `json:"value,omitempty"`
}

// ToolCallResult is the metric result for tool call entries. Value is a slice,
// and omitempty drops its key when it is nil or empty — which is the same thing
// here, because a result with no entries is never present.
type ToolCallResult struct {
	RawMetricResult
	Value []ToolCallEntry `json:"value,omitempty"`
}

// ActivationResult is the metric result for skill activation events. Value is a
// slice, like ToolCallResult's.
//
// It has all three constructors, unlike AttributionResult: an activation is
// read out of a source — Claude Code logs one event per invocation — so a
// source that exists and fails is a failure of this signal, the same as it is
// for tokens, tool calls and timing. It carried no Error constructor while no
// adapter read it, which was true until the read existed and is the kind of
// claim that has to move with the code rather than outlive it.
type ActivationResult struct {
	RawMetricResult
	Value []ActivationEntry `json:"value,omitempty"`
}

// TimingResult is the metric result for timing data.
// Value is a pointer so omitempty works — nil means no value (unknown/error states).
type TimingResult struct {
	RawMetricResult
	Value *TimingData `json:"value,omitempty"`
}

// AttributionResult is the metric result for attribution data. Value is a
// pointer so omitempty drops the key for unknown states, and there is no Error
// constructor: attribution is a property of the harness rather than of any
// export — no telemetry maps an output back to the skill that produced it — so
// there is no source that could exist and fail.
type AttributionResult struct {
	RawMetricResult
	Value *AttributionData `json:"value,omitempty"`
}

// EstimatedTokens is a chars/4 estimate over the payload bytes a hook carried.
// It is a relative signal — comparable between two runs of the same harness —
// and never a billed count.
type EstimatedTokens struct {
	Total int64 `json:"total"`
}

// EstimatedTokensResult is the metric result for an estimate.
//
// It is a signal of its own rather than a field inside TokenCounts, because
// TokenCounts is what was measured. An estimate folded in there would be
// averaged with measurements by anything reading the profile, and the result
// would be a number with no unit. Keeping it out is what lets `tokens` stay
// "unknown" on a harness that only exposes hook payloads, which is the true
// answer about that harness.
type EstimatedTokensResult struct {
	RawMetricResult
	Value *EstimatedTokens `json:"value,omitempty"`
}

// noEstimateReason is what an absent estimate says for itself.
const noEstimateReason = "no context-token estimate was made for this session"

// orAbsent is the result an estimate reports when none was made at all.
//
// Profile.EstimatedContextTokens is a pointer, so "nothing was estimated" is an
// absent key rather than a result — and a nil pointer has no state to read.
// Resolving it here, once, is what keeps the signal inside the closed
// {present, unknown, error} vocabulary instead of giving every caller a fourth
// case spelled "the pointer was nil".
func (r *EstimatedTokensResult) orAbsent() EstimatedTokensResult {
	if r == nil {
		return UnknownEstimatedTokensResult(noEstimateReason)
	}
	return *r
}

// Profile is the serialized artifact that F04 reads. SnapshotHash is the
// caller-supplied id labelling the skill version; it is recorded verbatim and
// is not derived from, or validated against, SkillDir.
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

	// EstimatedContextTokens is a pointer, and that is load-bearing rather than
	// stylistic: `omitempty` does nothing on a struct, so a value here would
	// put `"estimated_context_tokens":{"state":""}` on every profile ever
	// written — including every profile of a harness that estimates nothing —
	// and `""` is not a member of the state vocabulary. A v1 reader walking
	// the states would then be wrong rather than merely ignorant of a new key,
	// which is the one thing that would force schema v2.
	EstimatedContextTokens *EstimatedTokensResult `json:"estimated_context_tokens,omitempty"`
}

// SignalStates is the state of every signal in the profile, keyed by the name
// the capability report uses. It is the one place the profile's six results
// are enumerated together, so a caller asking "did this capture read anything"
// cannot walk five of them and believe it walked the set.
//
// Every result field on Profile appears here, and a test derives that set from
// the struct by reflection and requires this map to equal it — so the list
// below cannot be the one that was forgotten when a seventh signal arrives.
func (p Profile) SignalStates() map[MetricName]MetricState {
	return map[MetricName]MetricState{
		MetricTokens:          p.Tokens.State,
		MetricToolCalls:       p.ToolCalls.State,
		MetricSkillActivation: p.SkillActivation.State,
		MetricTiming:          p.Timing.State,
		MetricAttribution:     p.Attribution.State,
		// An estimate nobody made is "unknown", the same answer an export that
		// carried nothing gives for every other signal.
		MetricEstimatedContextTokens: p.EstimatedContextTokens.orAbsent().State,
	}
}

// ProfileSchema is the version string embedded in every profile.
const ProfileSchema = "skill-architect/profile/v1"

// AdapterVersion is the current adapter implementation version. It is
// recorded in every CapabilityReport, so it is bumped whenever the adapter
// changes what a profile contains for the same input — as 0.4.1, 0.4.2 and 0.4.3 all did.
//
// 0.5.0 is set once, here, for the whole release rather than per change:
// several of the release's slices change what a profile contains, and a value
// bumped by each of them would make adapter_version un-interpretable — a reader
// could no longer tell which set of behaviours produced a profile.
const AdapterVersion = "0.5.0"

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

// ErrorToolCallResult creates a ToolCallResult with state "error".
func ErrorToolCallResult(reason string) ToolCallResult {
	return ToolCallResult{RawMetricResult: RawMetricResult{State: MetricError, Reason: reason}}
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

// ErrorActivationResult creates an ActivationResult with state "error", for an
// activation source that existed and could not be read.
func ErrorActivationResult(reason string) ActivationResult {
	return ActivationResult{RawMetricResult: RawMetricResult{State: MetricError, Reason: reason}}
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

// ErrorTimingResult creates a TimingResult with state "error".
func ErrorTimingResult(reason string) TimingResult {
	return TimingResult{RawMetricResult: RawMetricResult{State: MetricError, Reason: reason}}
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

// --- EstimatedTokensResult constructors ---
//
// There is no Error constructor, for the same reason AttributionResult has
// none: an estimate is derived from payloads already in hand, so there is no
// separate source that can fail it. A payload that could not be read fails the
// signal that reads it.

// PresentEstimatedTokensResult creates an EstimatedTokensResult with state
// "present". The source names the estimation, not a meter: an adapter passing
// anything other than SourceHooksEstimated here is claiming a measurement.
func PresentEstimatedTokensResult(v EstimatedTokens, source string) EstimatedTokensResult {
	return EstimatedTokensResult{
		RawMetricResult: RawMetricResult{State: MetricPresent, Source: source},
		Value:           &v,
	}
}

// UnknownEstimatedTokensResult creates an EstimatedTokensResult with state "unknown".
func UnknownEstimatedTokensResult(reason string) EstimatedTokensResult {
	return EstimatedTokensResult{RawMetricResult: RawMetricResult{State: MetricUnknown, Reason: reason}}
}
