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
    Source string      `json:"source,omitempty"` // "otel" | "hooks" | "hooks_estimated" | "session_data" | "server_api" | "sqlite"
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

type EstimatedTokensResult struct {
    RawMetricResult
    Value *EstimatedTokens `json:"value,omitempty"`
}
```

Adapters build results through the constructors rather than setting `State` by
hand. The four signals read out of an export have all three constructors —
`PresentTokenResult`/`UnknownTokenResult`/`ErrorTokenResult`, and the same for
tool calls, skill activation and timing. `AttributionResult` and
`EstimatedTokensResult` have only `Present…` and `Unknown…`: they are a property
of the harness and of the adapter, not of any export, so no export can fail
them. `ActivationResult` gained its `Error…` constructor in 0.5.0, when the
activation began being read out of the export — a source that exists can fail,
and the rule the constructor set encodes is "is there a source", not "which
signal is it".

**Rules:**
- `State == "present"` → `Value` must be populated, `Reason` must be empty.
- `State == "unknown"` → `Value` is nil and its key is absent from the JSON; `Reason` explains why (e.g., "no `claude_code.token.usage` metric found in OTel export", "no `claude_code.skill_activated` log events found in OTel export").
- `State == "error"` → `Value` is nil and its key is absent from the JSON; `Reason` describes the failure (e.g., "malformed JSON at byte 87 in batch 1: unexpected end of JSON input", "OTel export file is empty").
- `Source` is always populated when `State == "present"` so the profile is auditable.

### CapabilityReport

Declared upfront by each adapter after probing its environment:

```go
type MetricName string

const (
    MetricTokens                 MetricName = "tokens"
    MetricToolCalls              MetricName = "tool_calls"
    MetricSkillActivation        MetricName = "skill_activation"
    MetricTiming                 MetricName = "timing"
    MetricAttribution            MetricName = "attribution"
    MetricEstimatedContextTokens MetricName = "estimated_context_tokens"
)

type CapabilityReport struct {
    Harness    string                    `json:"harness"`     // "cursor" | "claude_code" | "codex" | "devin"
    AdapterVer  string                    `json:"adapter_version"`
    ProbedAt   string                    `json:"probed_at"`   // ISO 8601 UTC
    Capabilities map[MetricName]MetricSource `json:"capabilities"`
}

type MetricSource string

const (
    SourceOtel           MetricSource = "otel"
    SourceHooks          MetricSource = "hooks"
    SourceHooksEstimated MetricSource = "hooks_estimated"  // estimated over hook payloads, never measured
    SourceSessionData    MetricSource = "session_data"
    SourceServerAPI      MetricSource = "server_api"
    SourceSQLite         MetricSource = "sqlite"
    SourceNone           MetricSource = "none"
)
```

`hooks_estimated` is a source of its own rather than a note on `hooks` because
an estimate and a measurement must not pool. A reader grouping profiles by
source has to be able to exclude derived numbers without knowing which signal
they came from.

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
    // sessionID is harness-specific (Claude Code session ID, Cursor conversation ID, etc.)
    // and is what the capture reads by, not only what the profile is stamped with:
    // an export may carry several sessions, so only records carrying this identity
    // contribute. An empty sessionID is refused.
    // opts carries optional configuration (session export path, API credentials,
    // and the snapshot id and skill dir recorded in the profile).
    Capture(sessionID string, opts CaptureOpts) (Profile, error)
}

// ProbeDiagnoser is optional, and asserted at the call site rather than folded
// into ProfilerAdapter: an adapter that cannot say why a capability came back
// "none" is still a complete adapter.
type ProbeDiagnoser interface {
    // ProbeWithDiagnostics probes once and returns the report together with one
    // message per distinct reason a signal could not be read. Empty when
    // nothing failed, including when no export was supplied at all — that is an
    // answer about the session, not a fault of the run.
    ProbeWithDiagnostics() (CapabilityReport, []string)
}

type CaptureOpts struct {
    ExportFile    string `json:"export_file,omitempty"`      // ATIF export or session transcript path
    APIKey        string `json:"api_key,omitempty"`          // server API auth (Devin)
    SnapshotHash  string `json:"snapshot_hash"`              // git SHA or content hash of the skill being profiled
    SkillDir      string `json:"skill_dir"`                 // path to the skill being profiled
}
```

## Adapter contract

Implementing `ProfilerAdapter` is not only satisfying the method set. Three
obligations hold for every adapter, and an adapter that breaks any of them
produces a profile that claims a measurement nobody made.

1. **A capture names one session.** `Capture("")` returns
   `SessionIDRequiredError(Name())`. A capture reads only the records carrying
   the session it was asked for, so an empty id is not a session that matched
   nothing — it is no assertion, and a profile stamped with it could only report
   every session the export happens to carry. The CLI requires `--session`
   before it gets that far; the adapter's refusal is what makes the rule true
   for a library caller too, and the two say the same thing deliberately.

2. **Probe and capture answer the same question, in both directions** (AC9).
   Every signal the capability report advertises from a source other than `none`
   is `present` in the capture of a session the export carries, with that source
   and with the value that was read; every signal it marks `none` is not
   `present`. The forward direction refuses an adapter that advertises a
   capability its capture cannot deliver. The reverse refuses one that reports a
   value it never claimed a source for. The capability report is the
   denominator, so it must enumerate every signal the profile carries: an
   adapter cannot satisfy this by advertising nothing.

3. **A count that was not read has no key.** Every `TokenCounts` field is
   `*int`, and a token result built from an export carrying only cache counts
   has no `input` and no `output` key at all — not a zero. "The export said
   nothing about this" and "the export said zero" are different answers, and
   arithmetic on a value field turns the first into the second. An adapter that
   advertises no token signal is exempt from the second half, and only that
   adapter.

These are enforced, not documented. `profiler/adapters.go` holds
`adapterRegistry`, the one list of harnesses the profiler ships;
`profiler.NewAdapter` is how the CLI resolves `--harness`, and
`profiler.HarnessNames` is where its help text and its refusal both get the set
they name. `profiler/adapter_contract_test.go` walks that registry and asserts
all three obligations over every entry, pairing each with the exports its claims
are asserted over — one the adapter reads, and one carrying only cache counts —
so an adapter with no input for which its own claims hold cannot ship.

Three checks keep the denominator honest. A registered harness with no fixtures
is a failure, not a skip. The registry is checked against the package's own
source for types implementing `ProfilerAdapter`, so an adapter cannot be added
without entering it. And the command's package is scanned for adapter types too:
an adapter defined there would be dispatchable and unreachable by the contract,
so it is refused — adapters live in `package profiler` and in the registry,
which is what makes them dispatchable at all. The table's ability to fail is
asserted on every run, against stub adapters built to break one obligation each.

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

    // A pointer, so a profile that estimated nothing has no key at all.
    // `omitempty` does nothing on a struct: held by value this would put
    // `"estimated_context_tokens":{"state":""}` on every profile, and `""` is
    // not a member of the state vocabulary.
    EstimatedContextTokens *EstimatedTokensResult `json:"estimated_context_tokens,omitempty"`
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
    ErrorType string `json:"error_type,omitempty"`  // the source's classification of a failure
    Count     int    `json:"count,omitempty"`       // aggregate for a delta metric; absent = one call
    ID        string `json:"id,omitempty"`          // tool_use_id when the source supplies one
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
    Category      string `json:"category,omitempty"`        // skill | mcp | cli | subagent | tool
    Detail        string `json:"detail,omitempty"`          // how it was derived, e.g. "file" for a path-inferred link
    OperationName string `json:"operation_name,omitempty"`  // the source's operation, e.g. "execute_tool"
    Confidence    string `json:"confidence,omitempty"`      // inferred | observed
}

// A chars/4 estimate over the payload bytes a hook carried. A relative signal,
// comparable between two runs of the same harness, and never a billed count.
// It is its own signal rather than a field inside TokenCounts, because
// TokenCounts is what was measured: folded in there it would be averaged with
// measurements, and `tokens` could no longer stay honestly "unknown" on a
// harness that exposes only hook payloads.
type EstimatedTokens struct {
    Total int64 `json:"total"`
}
```

`Confidence` is the honesty field on an attribution: a link derived from a file
path a skill happens to own is `inferred`, and one read from telemetry naming
the skill is `observed`. A reader that cannot tell them apart has to treat every
attribution as a guess, which makes the whole signal unusable.

**Serialization rules:**
- Profile is JSON. Pretty-printed for human readability, but parsing is canonical.
- `schema: "skill-architect/profile/v1"` is required. Future versions bump the suffix. The rule for bumping it: v1 stays while a v1 reader is merely *ignorant* of a new key, and v2 is required when a v1 reader would be *wrong*. Every key 0.5.0 adds — `error_type`, `count`, `id`, the four attribution fields, `estimated_context_tokens` — is optional and absent when it was not read, so a v1 reader skips what it does not know and is right about everything it does. A signal that serialized an empty result on every profile would break that, because its `state` would be outside the vocabulary a v1 reader walks; that is why the estimate is a pointer.
- `snapshot_hash` is whatever the caller passed to `--snapshot`, copied into the profile verbatim. The profiler neither hashes nor validates `--skill-dir` against it; the caller owns that correspondence. `compare` reads it and **states a difference rather than refusing one**: comparing a skill before and after a change is the paired comparison's whole purpose, so two snapshot ids are a `note`, not a refusal. What `compare` does refuse is stated in the comparison contract below.
- A profile with all metrics `unknown` is valid — it honestly reports that no telemetry was available.
- A profile must round-trip: `Marshal → Unmarshal → Marshal` produces identical JSON (modulo key ordering).
- A profile that marks a signal `present` and carries no value for it is **invalid**, and `LoadProfile` refuses it rather than leaving the comparator to dereference it. For the two list-valued signals — `tool_calls` and `skill_activation` — "no value" is an empty list as well as an absent one: a result with no entries is never `present`, and an adapter reporting a session that called no tool says so with `unknown` and a reason.

## Comparison contract (`skill-architect/comparison/v1`)

`CompareProfiles(baseline, candidate)` reads two profiles and reports what
changed between them. `LoadProfile(path)` is how a stored one is read, and it
refuses anything of another schema, anything that is not a profile, and any
profile marking a signal `present` with no value behind it.

The comparison's job is to answer one question — what changed about the *skill*
— and every rule below exists to stop it answering a different one, what changed
about the *reader*, and presenting that answer as if it were the first.

**A refusal is not a warning beside a delta. No delta is computed at all.** A
number in the report is a number somebody will read, so a comparison that cannot
be made honestly carries a reason and no values.

1. **The pair is refused when the two profiles do not name the same, non-empty
   `capability.adapter_version`.** The report carries `refusal`, every
   `MetricComparison` carries the same reason, none is `comparable`, and the
   reason names **both versions**. `AdapterVersion` is bumped whenever the
   adapter changes what a profile contains for the same input, which is exactly
   the condition under which a delta stops being about the session — so a
   profile stored under 0.4.x subtracted from a fresh 0.5.0 one reports four
   releases of fixes to the reader as the skill's regression. This is that
   constant's first consumer and the reason it exists. Two profiles naming *no*
   version are refused too: `"" == ""` is agreement between two unknowns.

2. **One signal is refused when the two sides read it from different sources.**
   A delta across `otel` and `sqlite` is not a measurement of the same thing.
   The reason names both sources, and the signals that do agree still compare —
   a cross-source signal refuses itself, not the report.

3. **Every `MetricComparison` carries `baseline_source` and `candidate_source`,
   always**, whatever the outcome and including the signals neither side read,
   which report `none`. Neither key is `omitempty`: an empty string is not a
   member of the source vocabulary, and a reader walking it would be *wrong*
   rather than ignorant — the same rule that makes `estimated_context_tokens` a
   pointer on the profile.

4. **A delta is only ever over what both sides read.** A `TokenCounts` field
   one profile never read has no key in the delta either, because the difference
   between a number and an absence is not a number; both sides' values are
   carried in full beside it, so which side was missing it is visible. When the
   two share no count at all, `tokens` is not comparable and says so — a delta
   object with every key absent reads as "no change".

5. **`estimated_context_tokens` carries a `note` on the answer itself**: a
   relative estimate, not billed — an ordinal signal only, and never a cost. The
   note travels with the delta rather than being documented beside it, so it
   survives into anything that renders the report.

6. **`skill_activation` is compared as a set of skill names**
   (`only_in_baseline` / `only_in_candidate`) as well as a count, because a bare
   count cannot say which skill stopped firing. Per-task precision and recall
   belong to the experiment layer, not here.

7. **A difference that does not stop a comparison is a `note`, not a refusal.**
   `harness`, `snapshot_hash` and `skill_dir` are noted when they differ.
   Comparing two snapshots of a skill is the paired comparison's whole purpose.

The report's own `comparable` is **derived**: it is true when some signal was
compared. A refusal makes every signal incomparable by construction rather than
by a second rule, which is what keeps the two from ever disagreeing.

`compare` exits **0** when something was compared, **2** when the report is
complete and nothing in it is comparable, and **1** for every usage error
including a profile that could not be read. The report is written to stdout
whatever the status: when nothing compared, the reasons are the answer.

## Adapter implementations (Slice 1 scope)

### Claude Code adapter (Slice 1)

**Telemetry surface:** OTel export via `CLAUDE_CODE_ENABLE_TELEMETRY=1` + `OTEL_*` env vars.

**Input format:** OTLP/JSON, as Claude Code emits it over `OTEL_EXPORTER_OTLP_PROTOCOL=http/json`. One `ExportMetricsServiceRequest` or `ExportLogsServiceRequest` per JSON object; a file may hold one object, one per line (NDJSON), or several concatenated, and all three read the same way. A top-level value that is not an object is not an export. README "Capturing an OTel export" documents the two routes that produce the file.

Within that envelope:
- Metrics are walked at every level — `resourceMetrics[].scopeMetrics[].metrics[]` — not just the first of each. Token counts come from `claude_code.token.usage` sum data points: the `type` attribute (`input`, `output`, `cacheRead`, `cacheCreation`, camelCase on the wire, snake_case in the profile) and the value from `asDouble`, falling back to `asInt`. Every 64-bit integer is accepted as a JSON number or a decimal string. That is narrower than the proto3 JSON mapping OTLP mandates, which also accepts exponent notation for an integer field; refusing exponent form is a deliberate deviation, recorded as such under "Deliberate deviations" below rather than claimed as conformance. A fractional `asDouble` is rounded, not truncated.
- **A value reaches the profile only if it is a count**: a whole number from 0 to 2⁶³−1. The rule is applied to the *rounded* value, so a `-0.4` that is float drift on a true zero is the zero it was, while `-0.5` is a number the exporter meant to be negative. A data point that fails this splits two ways, counted apart because they are two different things to go and look at in a capture. A leaf that **decoded as a finite number** whose rounded value is negative or 2⁶³ or greater *is not a count*: it is refused rather than converted — converting it is undefined in Go and gave the same export two different profiles on two architectures — and `asInt` is not consulted after it, because `asDouble` already answered. A leaf that **did not decode as a finite number at all** is *unreadable*, not "not a count": `NaN`, `Infinity` and a literal that overflows `float64` are refused by the float reader itself, so `asDouble` said nothing and `asInt` is read next — `{"asDouble":"NaN","asInt":"9"}` is the count 9, not a refusal. A point is unreadable only when neither leaf yields a number, which is the clause the reason uses: "carried no asDouble or asInt value that reads as a number". An `asDouble` is bounded at 2⁶³ and not at 2⁶³−1 because no `float64` is 2⁶³−1: the literal `9223372036854775807` parses to exactly 2⁶³, and only `asInt` can carry that count exactly. A token type that no data point carried has no key in the profile at all; a type read as zero has its key, with `0` in it.
- **Data points are merged per time series, and a series is what OTel says it is**: the resource it was exported from, the instrumentation scope that recorded it, the metric's name, and the data point's full attribute set — every attribute, of every `AnyValue` kind, and not the subset that renders as text. **Each `AnyValue` kind is canonicalised by what that kind is.** An `arrayValue`'s member order is part of its value and is kept; a `kvlistValue` is a map, and the OTel common data model defines two maps as equal irrespective of the order their members arrive in, so its members are sorted. Both recurse, so a map nested in an array or in another map normalises too, and a `null` members list is the empty one it decodes to under the proto3 JSON mapping. A *kind* written `null` is that kind **unset** rather than that kind holding its zero value, because ProtoJSON reads a null as the field being unset: `{"kvlistValue":null}` is an `AnyValue` of no kind — the same answer as `{}` — while `{"kvlistValue":{}}` is an empty map somebody set, and the two are two series. That is the rule the envelope is read by as well (`{"resourceMetrics":null}` says nothing about metrics), and it holds for every kind, scalars included; it does not reach a members list, because a repeated field's `null` *is* its default. Taking the compacted JSON as a composite's identity instead made one series read as two, and under cumulative temporality the two running totals **added** into a number no data point in the export carried — an over-count, which is the one direction "never invented" forbids. Two points differing in any of the four are two series and do not merge, so a capture that aggregates several resources or several scopes keeps them apart: two `resourceMetrics` entries whose data points carry identical attributes — a collector fanning in two `service.instance.id`s — are two running totals that add. A **resource** is its attribute set, so an entry carrying no `resource` and one carrying a resource with no attributes are one resource rather than two; `schemaUrl` is not part of it, because it declares which version of the semantic conventions the attributes follow, not a different origin. A **scope** is its name, its version and its attributes. Each of the four parts is encoded behind its byte length, so no part can spell another and no attribute value — a tool name, a prompt, a model id, all arbitrary text — can forge a foreign series' identity.
- **Temporality decides how the points of one series merge, and they are opposite instructions.** `aggregationTemporality` 1 (delta) means the points are the increments since the last export and add up; 2 (cumulative) means each point is a running total. A temporality that reads as neither is refused rather than assumed, and absent, unreadable and declared-but-neither are counted apart, because they are three different things to look at in a capture. **A series carrying both temporalities is refused whole**, for the same reason and by the same rule: the two are opposite instructions, so every way of resolving the mix invents a number the export does not contain — adding the increments to a running total counts them twice, and dropping them counts them not at all. The series contributes nothing and has no key in the profile, the refusal is counted per *series* rather than per data point (the points are individually fine; it is their company that is malformed), and the `tokens` reason names it: "1 time series carried both delta (1) and cumulative (2) aggregationTemporality points". Refusing one series is not refusing the export — the well-formed series beside it still count, and `tokens` is `unknown` only when no series survives. This is a change from v0.4.1, where a series carrying either temporality was treated as cumulative and the point with the latest `timeUnixNano` won — so which number a mixed series produced depended on the timestamps, not on write order. (Checked against the v0.4.1 binary: a cumulative 50 at t9000 written before a delta 100 at t1000 gave 50.) **What the profile can say about a refusal depends on what else survived.** When no series survives, `tokens` is `unknown` and the reason names the refused series. When a well-formed series survives beside it, `tokens` is `present` with a total the refused series is missing from, and schema v1 has no field to say so: a `present` result carries no reason. That is a gap in the schema rather than in the count, and it is tracked for 0.5.0 with the skipped-record count it belongs beside.
- **A cumulative series is a set of runs, and `startTimeUnixNano` identifies one.** The points sharing a start are one run, and what that run holds is the **greatest running total any of them reported**. These are monotonic counters: a running total only goes up, so the values carry their own order, every point of a run is a prefix of the greatest, and a flush reporting less than an earlier one on the same run is a capture contradicting itself rather than tokens given back. `timeUnixNano` is **not read** by the merge. Ordering by it lets such a flush take the run down with it, and it is also what rules out the obvious repair: a merge that kept the ordering and let each run hold its latest point by `timeUnixNano` would let supplying `startTimeUnixNano` — the field that says which run a point is from — make the profile report *fewer* tokens than leaving it out, which is the property below inverted. (v0.4.1 does not exhibit that, because it does not read `startTimeUnixNano` at all; it is a property of the candidate, measured against a build of it.) A point carrying a *different* start is a counter that restarted: its run sits beside the earlier one rather than replacing it, and the series contributes the sum of its runs, so a capture spanning a restart reports both — 100 over `startTimeUnixNano` 1–10 followed by 20 over 11–20 is 120. Runs are identified by their start, not by file order, so batches written out of order still add up correctly. **A point whose `startTimeUnixNano` is absent, zero or unreadable** cannot say which run it came from — but it came from one, either a run that named itself or a run nothing else in the capture observed, so it is neither a run of its own nor free. What the capture guarantees is the **least total over every run it could have come from**, and the cheapest such run is the one that already reached furthest: the series contributes `(its runs added) + max(0, the unplaceable total − its largest run)`. Both ends of that are wrong in a direction already shipped. Counting the point as a run of its own adds mass no point ever reported — one flush that omitted one field then doubles a session. Taking the series to be "the greatest running total observed anywhere on it" throws away the runs the point did not join, which are tokens the session really spent. Four consequences worth stating: a capture carrying no start times at all has no runs to place its points against, so it holds the greatest running total they reported (v0.4.1 held the latest by `timeUnixNano` instead, which is the same number wherever an exporter wrote its flushes in time order and a lower one where it did not — `900` at t1001 then `500` at t1002 gave 500 at v0.4.1 and gives 900 here); a session whose last flush omits `startTimeUnixNano` — one series, one run, 800000 then 1200000 then an unplaced 1200050 — reports 1200050, not 2400050; a capture whose runs reach 100 and 20 beside an unplaced point reporting 500 reports **520**, because the 500 is a running total of the run that reached 100 and the other run's 20 is still beside it — 500 drops a run that named itself and 620 invents a third; and a capture whose runs reach 100, 100 and 100 beside an unplaced 150 reports **350**, the case where the unplaceable total is above the largest run but below their sum, so a rule that compares it only with the sum never fires and loses 50 tokens the points reported. The property behind all four: taking information away — erasing a start time — may lower the total and may never raise it. `startTimeUnixNano` **`0` is `startTimeUnixNano` absent**: it is a proto3 `fixed64` and OTLP mandates the proto3 JSON mapping, whose deviations do not touch default values, so an explicit `0` and an omitted field are two encodings of one message — `protojson` writes `"0"` with `EmitUnpopulated` on and omits the field with it off — and the same holds for `timeUnixNano`. An **exponent-form** number (`1.7893e18`) does not read as a start time even when it is an exact integer: reading it means going through a `float64`, which cannot represent every nanosecond instant, and two points of one run that round apart would become two runs. That refusal is a deliberate deviation from the proto3 JSON mapping, recorded under "Deliberate deviations" below. Since an unreadable start now costs only what its point exceeds the largest run by, refusing it is the cheap direction. A **delta** series is one sum whatever its points' start times say: a delta point's `startTimeUnixNano` is the start of that point's own interval rather than of a run, and it is not read.
- Log records are identified by their `body`, which carries the fully-qualified event name (e.g. `claude_code.tool_result`) and is **taken as it stands**. A body that names an *unqualified* event — a bare `tool_result` — is refused rather than re-qualified: the qualified name is what this harness emits, so adding the prefix would manufacture an identity the record never had and attribute another product's log record to this profile. There is no fallback out of a body that said something. The `event.name` attribute is read only when the body says nothing at all — absent, not readable as a string, or an empty one — and only that attribute is qualified, because the short form is its documented spelling. A body may be an object, a bare string, or absent. Unknown event names are ignored.
- Timestamps are read from `timeUnixNano`, nanoseconds since the epoch as a decimal string or a number, and recorded in the profile as RFC 3339 timestamps.
- **Data points and log records that cannot be read are skipped, and the profile does not report how many.** A `present` result carries no reason in schema v1; surfacing a count of skipped records is tracked for 0.5.0, and two more cases belong in that same channel. A **refused time series**: a `present` total reduced by a malformed series beside a healthy one is invisible today. And **records the provenance projection removed while others survived**: when *some* of a signal's records lack the asserted `session.id` — or carry another session's, or arrived under a foreign scope — the surviving ones make the signal `present`, so the value is silently reduced and carries no reason at all, because the "not read" clause is built only on the `unknown` paths. All three are one gap with one shape: a `present` result has no field in which to say what it did not count. When *no* record survives, every signal is `unknown` and the clause does name how many were passed over and why, which is why this is a schema gap rather than a counting one.

**Deliberate deviations from the proto3 JSON mapping.** Two, stated here so this is not mistaken for a conformance claim. They run in opposite directions: one refuses a form the mapping accepts, the other accepts a form the mapping refuses.

**Narrower than the mapping — exponent form is refused.** A 64-bit integer field written in **exponent form** (`1.7893e18`) is refused, though the mapping accepts it. Reading it means going through a `float64`, whose spacing at that magnitude is 256ns, and `startTimeUnixNano` is a run identity — two points of one run that round apart would become two runs, and an approximate identity is worse than none. No exporter going through `protojson` emits integers that way, so the cost is theoretical and the risk is not. The same rule is applied to every 64-bit integer leaf rather than to timestamps alone, because one number-reading rule is easier to hold than two.

**Wider than the mapping — a quoted decimal enum is accepted.** `aggregationTemporality` is read from a bare number, from the enum name, *and* from a **quoted decimal** (`"aggregationTemporality": "2"`). The mapping takes the first two and refuses the third: it reads a JSON string as an enum *name*, and `"2"` is not the name of anything. Checked against the reference `protojson` on the OTLP `Sum` message — `2` and `"AGGREGATION_TEMPORALITY_CUMULATIVE"` unmarshal, `"2"` fails with `invalid value for enum field aggregationTemporality`. Nothing observed produces the quoted form, and the reader's own comment puts it no higher than a *may*; the same reference `protojson` writes the enum *name*, not the digit. The acceptance rests on what refusing would cost rather than on a sighting: a temporality that does not read is expensive, because it decides how an entire series merges, and the alternative is refusing a series over its punctuation. `cumulative.ndjson` carries the quoted form so the acceptance stays pinned, and is the measurement of that cost — making just that batch's temporality unreadable instead drops the capture's `input` total from 1150 to 750, the series holding the 500 it had already reached rather than the 900 the quoted batch carries. It is a fixture written to pin the acceptance, not an observation of a producer. This one widens what is read and refuses nothing the mapping accepts, which is why it is the safer direction of the two.

**Failure classification:** `error` means the file could not be read as an OTLP/JSON export — unreadable, empty or whitespace-only, a top-level value that is not a JSON object, malformed JSON, or a value that does not fit the OTLP schema. No reason quotes a Go type. Two of those five sit inside a batch and name it (1-based): malformed JSON, and a value that does not fit the OTLP schema — which also names the field path it found the wrong type at, when the decoder reports one. A batch whose own top level is the wrong shape (an array where the envelope should be) has no field path inside it to name, and that reason names the batch alone. Malformed JSON is the only one with a single byte to point at, so it is the only reason that carries an offset; the rest name the shape or the failure, because no offset would mean anything for them. A parse failure is never swallowed: nothing from earlier batches is used, and past batch 1 the reason says what to do about a capture stopped mid-write.

**One convention for byte offsets:** the 0-based offset, in the capture file, of the first byte the decoder could not accept — counting the byte-order mark and any leading whitespace, because the reader's editor counts them. When the file ended mid-object there is no such byte, and the reason names the file's length, which is where the byte the decoder wanted would have been.

`unknown` means a well-formed OTLP/JSON export that carries no telemetry — for one signal, or, when no batch carries an OTLP envelope, for all three. **The envelope test is a value, not a key**: a batch carries an envelope when `resourceMetrics` or `resourceLogs` has a *value* under it. An empty list is a value, so `{"resourceMetrics":[]}` *is* an OTLP export of a session that emitted nothing, and each signal is `unknown` with its own reason. A JSON `null` is not a value — ProtoJSON reads `null` as the field's default, so a batch written `{"resourceMetrics":null}` said nothing about metrics — and a file whose batches carry a value for neither field is not a broken export, it is not an export: all three signals get the "this is not OTLP/JSON" reason. Naming the key is not enough, and `{"resourceMetrics":null}` is the case that tells the two apart.

**Timing is a span over API requests.** `start_time`, `end_time` and `total_ms` cover the earliest to the latest `claude_code.api_request` record, so they exclude the prompt before the first request and any tool activity after the last one. Per-request `duration_ms` and the `claude_code.active_time.total` metric measure different quantities and have no field in schema v1; both are tracked for 0.5.0.

**Probe logic:**
0. Probe is **not session-scoped**, because it is not given a session: it reports what the export can yield for the session the export belongs to. The instrumentation-scope half of the provenance test still applies. A capture names a session and reads only that session's records, so an export carrying several sessions can probe `otel` and capture `unknown` for a session it does not contain. A profile cannot disagree with itself: the `capability` block in a profile is derived from that capture's own scoped resolution, not from this one.
1. Probe resolves the export file exactly as capture does — one read, one parse, one extractor per signal — and reports a capability as `otel` only when that extractor produced a value:
   - a `claude_code.token.usage` sum declaring an `aggregationTemporality` of 1 or 2 — spelled as a bare number, as a quoted digit, or as the protobuf JSON enum name (`AGGREGATION_TEMPORALITY_DELTA`, `AGGREGATION_TEMPORALITY_CUMULATIVE`; `AGGREGATION_TEMPORALITY_UNSPECIFIED` reads as 0) — with a data point carrying a recognised `type` (`input`, `output`, `cacheRead`, `cacheCreation`) and an `asDouble` or `asInt` value that is a count → `tokens: otel`
   - `claude_code.tool_result` log events carrying a `tool_name` and a readable `success`, or `claude_code.tool_decision` events recording a reject with a `tool_name` → `tool_calls: otel`
   - `claude_code.skill_activated` log events carrying a `skill.name` → `skill_activation: otel`
   - `claude_code.api_request` log events carrying a parseable `timeUnixNano` → `timing: otel`
   If the file is absent, unreadable, malformed, missing the signal, or carrying the signal with nothing readable inside it, that capability stays `none`. Availability is a fact about a value in hand, not about a name matched in a file — a probe that reports structure is how it comes to advertise data the capture cannot deliver.

   A capability's vocabulary is a source or `none`, so the report structurally cannot distinguish "no telemetry was configured" from "the export you named could not be read": both are `none`, while capture keeps them apart as `unknown` and as `error` with a reason. That distinction is information `resolve` already computed, so probe reports it — on **stderr**, through `ProbeDiagnoser`, one message per distinct reason, using capture's own wording. Adding it to `CapabilityReport` instead would change what every `Profile` that embeds the report contains for the same input, which is a schema and adapter-version question; a second channel changes nothing anyone parses. `Probe()` delegates to `ProbeWithDiagnostics()` so there is one read of one file and the report and its explanation cannot describe different files.
2. Attribution is always `none` for Claude Code, because there is nothing to read: its telemetry carries no output-to-skill mapping at all. Skill activation was `none` for the same-shaped but different reason — the telemetry existed and this adapter did not read it — until 0.5.0, which reads it. **The source is the event, not the attribute.** `claude_code.skill_activated` is logged when a skill is invoked, through the Skill tool or a `/` command, and only then, so one record is one activation and the timestamp on it is the time the skill was invoked. `skill.name` also rides along on `token.usage`, `cost.usage`, `api_request`, `api_error` and `api_refusal`, where it marks the skill active *for that request* — a skill used across five requests carries it five times, and those records carry flush and request times rather than invocation times. Reading them as activations would report a count and a set of times the harness never recorded, so an export carrying the attribute and no event reports `skill_activation: none`, with a reason naming the event that was looked for. A reason may say what this adapter does not read; it may not say what the harness does not emit unless that is true.

**Provenance — which records a profile may be built from.** A capture file is not a session. Both documented capture routes append to one file by design, and route (a) is a receiver on the standard OTLP port that anything on the machine may post to, so one export legitimately carries several sessions and more than one product's telemetry. A profile names one session and is read as a measurement of it, so **a record contributes only when it carries the asserted `session.id` and was not recorded by another product's instrumentation scope.** Two independent tests, because either alone leaves a hole: Claude Code puts `session.id` on every metric data point and every log record, so a record either says which run it is from or cannot be attributed to one — carrying a different id and carrying none are the same answer; and a record naming another product's scope is that product's whatever identity it carries. The scope test is an **exclusion of a positively foreign scope, not an allowlist**: a scope that names no library is read, because the scope is optional in OTLP and a receiver or collector in the path may not carry one through, and refusing those would trade a wrong number for no number on every pipeline that drops it. The match is on the harness's own namespace component (`claude_code`), which is the same token its signal names are qualified with, so `com.anthropic.claude_code`, its `.events` and `.subagent` siblings and any later tail all read, while `some.other.product` does not. A log record's event identity is read the same way: the fully-qualified name in the body is taken as it stands rather than re-qualified, because adding the prefix to a body naming a bare `tool_result` manufactures an identity the record never had; only the `event.name` attribute is qualified, since the short form is the documented spelling of it. **A `present` signal is therefore always a measurement of the session the profile names.** When the projection removes records a signal reads from, that signal's `unknown` reason carries a clause naming how many and why, so a reason saying a signal was not found cannot be read as saying the export carries nothing of the kind; the clause is absent when nothing was removed, which is every capture of one session with nothing else on the port.

**Capture logic:**
1. Refuse a supplied `CaptureOpts.ExportFile` before doing anything else, with the same error the CLI gives for `--export-file`, rather than silently ignoring it: the adapter owns its input contract, and a caller who supplies a session export is told it is not read instead of receiving a profile that looks like missing telemetry. Refuse an empty `sessionID` on the same ground: a capture reads only the records carrying the session it was asked for, so an empty id is not a session that matched nothing but no assertion at all, and a profile stamped with it could only report every session the export happens to carry. `--session` is required by the CLI and load-bearing in the library, not required and discarded. Then resolve the export file once. That resolution owns failure classification for all three OTel signals: no file configured → each is `unknown` with the fallback reason below; the file cannot be read as an OTLP/JSON export → each is `error` naming the failure; it parses but carries no OTLP envelope → each is `unknown` naming the format expected; otherwise the parsed export is projected onto the provenance above and the projection goes to the extractors. An export that carries no record of the asserted session is **not** a failure of the export: each signal reports for itself that it read nothing and names what was there instead.
2. Extract `claude_code.token.usage` sum data points → `TokenCounts`, counting only those carrying a recognised `type` and a numeric value, merged per series according to the aggregation temporality above.
3. Extract `ToolCallEntry` list from two disjoint sources. `claude_code.tool_result` is emitted when a tool completes and only then, so it supplies the calls that ran: one entry per event carrying both a `tool_name` and a readable `success`, and `ToolCallEntry.Success` means the tool ran and succeeded. `claude_code.tool_decision` supplies the calls that were rejected and therefore never ran, listed with `success: false`. An accepted decision is not an entry — its outcome comes from its result, and an export captured mid-run simply does not list the calls whose results were not written yet. A `tool_result` whose `success` cannot be read is not an entry either: it is never recorded as a failed call. It is counted, and if the export yielded no tool calls at all that count reaches the `unknown` reason; where some calls were read the result is `present`, and a `present` result carries no reason, so the profile does not say how many records were skipped (see the bullet above). Entries sort by timestamp, and one whose `timeUnixNano` could not be read is kept, last, with an empty `timestamp`.
4. Extract an `ActivationEntry` list from `claude_code.skill_activated` log events: one entry per event carrying a `skill.name`, with `Trigger` from `invocation_trigger` when the event carried one and absent when it did not, ordered by timestamp ascending with untimed entries last in file order. An event carrying no `skill.name` is counted and not listed — an activation of no named skill is one a reader can do nothing with. Names are reported exactly as the export spelled them: Claude Code redacts user-defined and third-party plugin skills to `custom_skill` on this event unless `OTEL_LOG_TOOL_DETAILS=1`, and un-redacting would invent a name nobody recorded. `skill.name` on a request-scoped signal is **not** a source for this list; see Probe logic step 2.
5. Extract timing from `claude_code.api_request` log events → `TimingData` spanning the earliest to the latest parseable timestamp, so `end_time >= start_time` and `total_ms >= 0` whatever order the exporter wrote the events in.
6. Steps 2–5 settle their own signal independently: `present` with `Source: "otel"` when the extractor read at least one usable value, `unknown` naming what was missing or unreadable when it did not. A partial export never discards the signals it does carry.
7. `Attribution` → `unknown` with reason "Claude Code telemetry carries no output-to-skill mapping". It is the one signal here whose answer is the same for every export.
8. The profile's `capability` block is derived from the same resolution, so probe and capture cannot disagree about what the export yielded.
9. Serialize to `Profile` JSON.

**CLI exit status.** `profiler probe` writes the capability report to stdout, writes any
diagnostics to stderr, and exits **0** in every case, including one where the export it was
given could not be read. stdout is byte-identical with and without diagnostics, so a caller
already parsing it is unaffected. Probe has no documented exit contract to extend, so giving
it one is new surface rather than a repair, and it is **deferred to 0.5.0**; a caller that
must branch on a bad export uses `capture`, which does have one.

`profiler capture` writes the profile to stdout in every case, and exits **2** when no signal was read and at least one is `error` — a supplied export that could not be used — and **0** otherwise, including an all-`unknown` profile from a session with no telemetry configured. The status is what a wrapping script branches on; exiting 0 after reading nothing would have it store the all-unknown profile as a successful capture. Every usage error exits **1** before any capture happens: an unknown command or harness, a missing required flag, `--export-file`, and an unrecognised flag. The last one is why the flag sets use `ContinueOnError` — `flag.ExitOnError` exits 2 on its own, and 2 has to mean exactly one thing for a script to branch on it.

**Fallback:** With no export file configured, `tokens`, `tool_calls`, `skill_activation` and `timing` are `unknown` with reason "OTel export not configured. Provide an OTel export file via --otel-file or OtelExportFile." A file that is configured but cannot be read as an OTLP/JSON export must not borrow that reason: missing, unreadable, empty, non-object at the top level, malformed, or not fitting the OTLP schema are all `error`, naming the failure, because the caller did supply a file and "not configured" would send them to fix the one thing that is not wrong. A file that parses but carries nothing readable for a signal leaves that signal `unknown`, naming what was missing; a file carrying no OTLP envelope at all leaves all four `unknown`, naming the format expected. `attribution` is `unknown` with its own reason (step 7 above) in every one of these cases — it is a property of the harness and of this adapter, not of the export. The adapter reads only the file it is given: it does not inspect `CLAUDE_CODE_ENABLE_TELEMETRY` or the `OTEL_*` env vars itself.

## Acceptance criteria (Slice 1)

1. Metric result serialization: a `present` result includes value and source; an `unknown` result includes reason and no value key; an `error` result includes reason and no value key.
2. `CapabilityReport` for Claude Code is per signal: `tokens`, `tool_calls`, `skill_activation` and `timing` are each `otel` only when the export file yields a readable value for that signal, and `none` otherwise; `attribution` is always `none`. An export yielding all four therefore reports `tokens: otel`, `tool_calls: otel`, `skill_activation: otel`, `timing: otel`. An export carrying `skill.name` on request-scoped signals and no `claude_code.skill_activated` event reports `skill_activation: none`: the attribute is not the source.
3. `CapabilityReport` for Claude Code without OTel: all metrics `none`. Same for an export file that cannot be read as OTLP/JSON, that carries no OTLP envelope, that carries the envelope with nothing in it, that carries none of the four export-backed signals, or that carries one with nothing readable inside it — a `token.usage` data point with an unrecognised `type`, no readable value (neither leaf decoded as a finite number — absent, non-numeric, `NaN`/`Infinity`, which the float leaf refuses, or an `asInt` outside the range of an int64), a value that is not a count (a leaf that decoded as a finite number whose rounded value is negative or 2⁶³ or greater), or a temporality that is absent, unreadable, or neither delta nor cumulative; a `token.usage` sum whose every series carried *both* temporalities, each of which is refused whole; a `token.usage` metric that arrives as a gauge rather than a sum; a `tool_result` with no `tool_name` or no readable `success`; a `tool_decision` with no recognised `decision`, or a reject with no `tool_name`; an `api_request` with no parseable `timeUnixNano`.
4. A Claude Code session whose export carries all four export-backed signals produces a profile where tokens, tool_calls, skill_activation and timing are `present`; attribution is `unknown` with reason. `skill_activation` is `present` when, and only when, the export carries a `claude_code.skill_activated` event of that session carrying a `skill.name`.
5. A session with no OTel export file produces a profile where `tokens`, `tool_calls`, `skill_activation` and `timing` are `unknown` with the reason quoted under **Fallback** above, which begins "OTel export not configured"; `attribution` is `unknown` with its own reason, as it is in every other case.
6. Profile JSON round-trips: `Marshal → Unmarshal → Marshal` is identical.
7. Profile `schema` field is `"skill-architect/profile/v1"`.
8. Profile `snapshot_hash` matches the input.
9. Probe and capture agree: every signal the capability report marks available is `present` in the profile with that source and carries a value, and every signal it marks `none` is not `present`, carries a reason, and carries no value. This holds for partial exports — an export with tool calls and timing but no token metric captures both, with tokens `unknown`.
10. A `present` signal means a value was read. An export carrying a signal's structure with nothing readable inside it yields `unknown` with a reason, never `present` without a value read from the export. A read zero is a value: a single `api_request` is a zero-length span and yields `total_ms: 0`, `present`, and a token type read as zero yields that key with `0` in it. A token type the export said nothing about has no key at all, so a reader can tell the two apart.
11. The same export produces the same profile on any architecture. A value that cannot be represented as a count is refused rather than converted, because an out-of-range float-to-integer conversion is undefined in Go and gave arm64 and amd64 different answers for one file.
12. An export file that is supplied but cannot be read as an OTLP/JSON export — unreadable, empty, a non-object top-level value, malformed, or carrying a value that does not fit the OTLP schema — yields `error` for `tokens`, `tool_calls`, `skill_activation` and `timing`, each naming the failure. `MetricError` is reachable and covered by tests.
13. An export file that parses but whose batches carry a *value* for neither `resourceMetrics` nor `resourceLogs` yields `unknown` for all four, naming the format expected — never `error`, and never a silent zero. The test is value presence, not key presence: `{"resourceMetrics":null}` carries no value, because ProtoJSON reads `null` as the field default, and gets the format reason. An empty list is a value, so an export carrying either field with an empty list under it is an OTLP export of a session that emitted nothing: each signal is `unknown` with its own reason, not with the format one.
14. `profiler capture` exits 2 when no signal was read and at least one is `error`, and 0 otherwise; the profile is written to stdout either way.
