# Profiler adapter interface spec (F03)

**Status:** reference for shipped behaviour · **Date:** 2026-09-21 · **Covers:** v0.5.0 · **Architecture:** Option C — adapter-per-harness with capability negotiation

> This document began as a forward-looking spec and is now the reference for what
> ships: every contract below is asserted by a test, and the sections that describe
> something unbuilt say so in the sentence that describes it.

## Purpose

A harness-agnostic profiler, designed so that an adapter per harness can be added without changing the profile schema, which degrades gracefully to `unknown` for unavailable metrics and produces a serialized profile labelled with a caller-supplied snapshot id that `compare` can read and group by. **One adapter ships: Claude Code.** `capture --harness cursor|codex|devin` answers "unknown harness" and exits 1; the architecture is written for four and the registry holds one, which is the distinction this document keeps rather than collapsing.

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
    ExportFile    string `json:"export_file,omitempty"`      // ATIF export or session transcript path; reserved, read by nothing in this release
    APIKey        string `json:"api_key,omitempty"`          // server API auth for a Devin or Cursor adapter; reserved, read by nothing in this release
    SnapshotHash  string `json:"snapshot_hash"`              // git SHA or content hash of the skill being profiled
    SkillDir      string `json:"skill_dir"`                 // path to the skill being profiled
}
```

`ExportFile` and `APIKey` are the input contract for adapters that do not exist, and **no
shipped adapter reads either**: the Claude Code adapter refuses an `ExportFile` rather than
ignoring it, and there is no CLI flag for an API key at all. They are kept because they are
the shape a Devin or Cursor adapter would need and removing them would break this struct
twice over. No release is named as the one that will read them, deliberately: naming the
release being cut reads as a schedule and becomes a false claim the moment that release
ships.

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

## Experiment contract (`skill-architect/experiment/v1`)

`compare` is handed two profiles. The **experiment** is the layer that produces
them: it declares two conditions, expands them into the runs that will be
executed, executes those runs, and hands each pair to `CompareProfiles`. Three
documents, each with its own schema because each is stored and read back:

| document | schema | produced by |
|---|---|---|
| design | `skill-architect/experiment/v1` | a human, by hand |
| plan | `skill-architect/experiment-plan/v1` | `GeneratePlan(design)` |
| result | `skill-architect/experiment-result/v1` | `RunPlan(plan)` |

Because the experiment is what *makes* the pair, it is the only layer that can
refuse to make a bad one. It refuses at three times, earliest first.

### 1. The design refuses any declaration the runner cannot honour

A design may declare only what the runner actually does. Running a design that
asked for a randomised order in blocked order, or for a ratio by subtraction,
produces a document that is the record of an experiment nobody ran — and the
document is what outlives the run.

- `ordering` — only `blocked` (every repetition of one task family, then the
  next). `random` is **refused**, not accepted-and-noted: nothing randomises.
- `analysis_method` — only `difference`. The comparison subtracts; nothing here
  computes a ratio.
- `stopping_rule` — only `fixed`. Stopping early on a peek needs alpha-spending
  or a minimum-N guard to stay honest; until that machinery exists, accepting
  the word would permit peeking and call it a stopping rule.
- `name`, `task_families` (no empty names), and each condition's `command`,
  `snapshot_hash` and `skill_dir` are required.
- Each `command` must mention **`$PROFILE`**. The plan chooses where each
  profile goes; a command that ignores it writes somewhere nothing reads.
- **Both conditions must name the same harness**, and it must be one the
  registry answers to. Two harnesses measure with different meters, so a
  difference between them is not a difference in the skill. `compare` treats a
  differing `harness` as a *note* — a human handing it two profiles may have a
  reason — but the experiment is deciding what to capture, so it refuses. This
  is why a run can never produce a cross-harness pair.

Defaults are applied before the refusals: `repetitions` defaults to 1, the three
declarations to their single supported values, and `harness` to the registered
adapter **only while exactly one is registered** — with two, there is no longer
one thing the design could have meant, and it has to say which.

`LoadExperimentDesign` normalizes as it reads, so a design cannot enter through
a file unchecked.

### 2. The plan is a document, expanded once and executed later

One run per repetition of each task family; each run is a **pair** — a
`baseline` step and a `candidate` step as named fields, so a run with one step
cannot be written down at all.

- `$TASK`, `$REP` and `$PROFILE` are substituted into the command at plan time.
- Environment variables are **not** expanded. A plan is kept and run later,
  possibly elsewhere: expanding the planning machine's environment would bake
  its values into the record, and an unset variable would become an empty string
  at plan time instead of being the shell's problem at run time. The shell that
  runs the step expands what is left.
- A task family becomes one path component, with everything but letters, digits,
  `-` and `_` replaced. There is no `.` left to make a `..` out of and no
  separator left to start a component, so a task family cannot name a file
  outside `output_dir`.
- Every step in a plan writes its own profile path. `LoadPlan` refuses a plan
  with no runs, a step with no command or no profile path, and two steps of one
  run writing the same path.

### 3. The run refuses a profile that is not the one the step declared

- The step's command is run by `sh -c`. Its **stdout is redirected to stderr**,
  because stdout belongs to the result document a wrapper parses.
- The profile file is stamped before and after. A command that exits 0 and
  writes nothing is refused — the path is then either empty or, worse, holds a
  profile from an earlier run, which would be compared and reported as this
  run's numbers.
- The profile is read with `LoadProfile` and checked against what the step
  declared: `harness`, `snapshot_hash`, `skill_dir`. A profile that disagrees is
  the wrong file.
- What is left is **the comparison's own refusal**, which is not restated here.
  The two commands are the caller's and nothing can promise they are the same
  build of the profiler, so a pair read by two adapter versions is still
  possible — and `CompareProfiles` refuses it, upstream of every subtraction.

A refused comparison is a **result, not an error**: every run is an independent
observation, and one pair the comparator will not subtract does not discard the
runs that worked. A step that fails, or that produces a profile the plan did not
ask for, **is** an error and stops the experiment; the setup is wrong and every
run after it would be wrong the same way.

The result's own `comparable` is **derived**: true when the experiment ran at
least one run and *every* run of it compared something. The "at least one" is
not defensive — "every run compared" is vacuously true of no runs, and an
experiment that executed nothing would otherwise report itself a success.

`experiment run` exits **0** when every run compared something, **2** when it
ran and some run did not, and **1** for every usage error and every failed step.
The result is on stdout for 0 and 2; a failed step leaves no result to print.
`design` and `plan` execute nothing and exit **0** or **1**.

## Hook spool contract (`skill-architect/spool/v1`)

Cursor can be configured to run a command on each of its lifecycle events,
handing it a payload. `profiler ingest` is that command: it reads one payload
and appends one line to a daily file. `profiler hooks install` registers it.

This is **capture only**. Nothing in this release reads a spool and reports a
measurement: there is no Cursor adapter, `profiler capture --harness cursor`
does not exist, and no `Profile` is produced from a spool line. What the spool
does is put on disk what was handed to it, so that a reader written later — once
somebody has run Cursor and seen what it actually sends — works from files that
already exist instead of needing the sessions captured again.

### What has been observed, and what has not

**No part of this has been checked against a running Cursor.** Cursor was not
installed on the machine where it was written, no spool line in this repository
was produced by Cursor, and no `hooks.json` this code wrote has been read by
Cursor. Every statement below about *Cursor* is taken from Cursor's published
hooks documentation, and that documentation has been re-read against this code
field by field — which closes the docs-versus-code gap and not the
docs-versus-running-Cursor one. Specifically, these remain unverified:

- that its `hooks` member is keyed by event name, and that an entry is an object
  with a `command` string;
- that a hook is invoked with one JSON document on stdin;
- that the document carries `hook_event_name`, `cursor_version` and
  `conversation_id`;
- every payload shape for the tool-call events (`preToolUse`, `postToolUse`),
  which were registered on a live hook emitter here and never fired.

Two assumptions this section used to list have since been settled against
Cursor's published reference, and they went in opposite directions:

- **Confirmed.** `~/.cursor/hooks.json` is the reference's User-scope location
  (Enterprise, Team and Project scopes sit above it and this tool writes none of
  them), and the 21 names in `CursorHookEvents` are exactly the documented event
  set, with no twenty-second. The reference also marks a top-level `version`
  **required** — *"Config schema version. Must be a positive integer (use 1)"* —
  and `InstallHooks` now writes `1` when the key is absent, which is a stronger
  statement than the old "unverified" note: without it the file most likely
  fails Cursor's schema validation and is ignored whole, while `doctor` reads
  the same file back and reports all 21 events registered.
- **Contradicted.** `cwd` is **not** a field every payload carries. The
  reference documents it on `preToolUse`, `postToolUse` and
  `beforeShellExecution`, and the field carried on every payload for workspace
  location is `workspace_roots`, which this build does not promote. So `cwd` is
  promoted when present, and blank otherwise — which is the "treated as absent
  rather than rendered" rule working, not a symptom of anything. Promoting
  `workspace_roots` is a candidate for a later release; `raw` carries it today.

If any of the remaining assumptions is wrong, `hooks install` writes a file
Cursor ignores and the spool stays empty, or lines arrive whose promoted
envelope fields are blank. **Neither loses data**, which is the reason this ships ahead of an adapter: a
wrong guess about a field name costs a re-read of files that are still on disk,
where a wrong capability claim in a stored profile is subtracted and reported by
callers who never see this file. The three unlanded adapters are what that looks
like, and none of them lands.

Everything below about *this code* is verified by its tests.

### The line

One JSONL line per invocation, appended to `<spool>/<YYYY-MM-DD>.jsonl`, where
the day is the UTC day of the capture time. Directory `0700`, file `0600`: a
line can carry prompt text and paths, and it outlives the session. An event
carrying no capture time is **refused** rather than filed — there is no daily
file for it, and `.jsonl` is hidden from every reader listing the spool by date.

| field | |
|---|---|
| `ts` | capture time, RFC3339 UTC. Read by this tool, not taken from the payload |
| `event` | the payload's `hook_event_name`, verbatim, or `"unknown"` |
| `schema_version` | `skill-architect/spool/v1` |
| `cursor_version`, `cwd`, `conversation_id` | promoted from the payload when present |
| `strict` | present and true when content fields were replaced by their sizes |
| `raw` | the payload |

The promoted fields are a **convenience for filtering a large spool without
decoding every line, and not the record**: `raw` keeps the payload's own copy of
everything they were taken from. A field that is present but not a string is
treated as absent rather than rendered, because a rendering of some other value
is not the payload's text.

### What "nothing is lost" means, exactly

- **No field is dropped.** An event name this build has never heard of is kept
  verbatim rather than mapped into the known set; a field nobody reads is
  carried through.
- **No value is changed.** Numeric literals are decoded with `UseNumber` and
  re-encoded as they arrived. Decoding into `any` without it sends every number
  through `float64`, so an integer above 2⁵³ comes back rounded — and a
  nanosecond epoch timestamp is about 1.7×10¹⁸, which is exactly the kind of
  field a payload carries and the kind this repo already pairs runs on.
- **A payload that is not one JSON document is still captured**, stored as a
  JSON string holding the bytes as received. That covers input that is not JSON
  at all, a truncated document, and two documents concatenated.
- **What is not preserved**, stated rather than glossed: the order of an
  object's keys, and a duplicate key, which collapses to its last occurrence.
  Both are consequences of decoding and re-encoding, neither changes the value
  of any field, and the JSON object model gives neither meaning.
- **A credential is removed**, and that is the one deliberate loss. Fields whose
  normalized name ends in `token`, `secret`, `password`, `passwd`, `apikey`,
  `privatekey`, `accesskey` or `sessionkey`, and the names `auth`,
  `authorization`, `credential`, `credentials`, are replaced wholesale.
  Credential-shaped substrings — a bearer token, an `sk-`, `AKIA` or `ghp_`
  key, a PEM header — are replaced in place, so the text around one survives: a
  command with a token in it is still the command that ran. Measurement names
  like `context_tokens` and `tokenUsage` survive, and so do file paths and
  prompts, which on this tool are the data. Redaction runs over every decoded
  shape and not only objects, because a credential inside a JSON array is still
  a credential.

`--strict` additionally replaces each content field — `prompt`, `text`,
`content`, `tool_input`, `tool_output`, `output`, `command`, `edits`,
`result_json`, `description`, `summary`, `agent_message`, `task`,
`attachments` — with `{"_stripped_bytes": N}`, keeping metadata. It is the
metadata-only mode for a shared machine, and it is a deliberate loss, so the
line records `strict: true`: a stripped line that did not say so would be
indistinguishable from one that simply carried no prompt.

### Reading it back

`LoadSessionEvents(dir, sessionID)` returns the lines belonging to one session,
ordered by capture time across every `*.jsonl` file in the directory. Files are
read oldest-first, which `os.ReadDir`'s filename ordering already gives.

- **An empty `sessionID` is refused.** It is not a session that matched nothing:
  every comparison is an equality against a field that reads as `""` when
  absent, so an empty id would select every line naming no session and hand
  back a machine's whole spool as one session. That is the session-scoping
  defect this release spent four versions closing.
- A session is matched on `conversation_id`, `session_id` or `generation_id`,
  because which of the three a payload carries is documentary.
- **A line that cannot be decoded is skipped and the file keeps being read.** A
  hook killed mid-write leaves a truncated last line, and losing the rest of the
  day over it would break the spool's one promise exactly when it matters.
- Only `*.jsonl` files are read, and not subdirectories. The spool sits under
  the user's home and collects other things.
- A spool directory that is not there is an **error**, not an empty session: a
  caller who cannot tell them apart reports "nothing happened" for a capture
  that never ran.

### Registration

`hooks install` merges the command into every event in `CursorHookEvents`;
`hooks uninstall` removes the entries whose command is exactly ours. `hooks.json`
is somebody else's file — Cursor owns the format, the user owns the contents —
so:

- **Idempotent.** A second install registers nothing and does not rewrite the
  file. It also leaves no backup: a backup named for a write that did not happen
  is a claim the result cannot support, and rerunning the install would
  otherwise drop one beside the file each time.
- **Additive.** Foreign entries on an event we also register, events we do not
  register, and top-level fields this build has never heard of all survive. The
  document is decoded into a map rather than a struct of the known fields, which
  is what makes the merge safe against a file written by a newer Cursor.
- **Backed up before any write**, install and uninstall alike, to
  `hooks.json.bak-<timestamp>`. Removing entries from a user's configuration is
  the more destructive of the two.
- **Refused, not replaced, when the file cannot be parsed.** A file this build
  cannot read is one whose contents it cannot preserve, so the error names the
  path and the file is left exactly as it was.
- An entry is matched by its `command` **as a string**. An entry whose `command`
  is a number that reads the same is not the same entry.
- An event left with no entries is removed with them: an empty registration is a
  trace of us in a file we are meant to have left as we found it.
- **The schema version is supplied when absent and never overwritten.**
  `ensureSchemaVersion` sets a top-level `"version": 1` if the key is missing,
  because Cursor's reference marks it required; an existing value, including a
  future `2`, is left alone. It is called from `InstallHooks` and deliberately
  **not** from the save path `uninstall` shares: a required field added on the
  way out would be something of ours left in a file we are meant to have
  vacated. The result reports `schema_version_added` so a write that only
  supplied the version does not report a backup and name nothing it did.
- **Uninstall on a home that had no `.cursor` directory does not restore that
  state.** It leaves the directory, a `hooks.json` holding `{"hooks": {},
  "version": 1}`, and the timestamped backup. The residue is schema-valid, so it
  does not break a hook config the user adds by hand later, but it is not
  as-found and the bullet above's reasoning applies one level up.
- **The backup suffix is second-resolution, so two writes in one second
  collide.** Install then uninstall inside one second leaves one
  `hooks.json.bak-<timestamp>` holding the post-install state, so the file the
  user started with is not recoverable from it.

The command registered by default is this binary's **absolute path** plus
`ingest --spool-dir <home>/.skill-architect/spool || true`. Absolute because
`PATH` inside a hook's environment is not something this tool gets to assume;
`--spool-dir` for the same reason applied to `$HOME`, so that `--home` scopes
the capture and not only the registration and `doctor --home X` describes the
spool the registered hook actually writes to; `|| true` so a failure of ours
cannot take the user's session down. `--command` still wins outright, and it is
the only way to register `--strict`, which nothing else supplies and no
environment variable reaches.

### The home directory

`InstallHooks(home, …)`, `UninstallHooks(home, …)`, `AppendSpool(dir, …)` and
`Ingest(…, dir, …)` all take the directory they write as a parameter.
`DefaultSpoolDir()` is the only function that reads the real user's home, and it
only computes `~/.skill-architect/spool` — nothing in the library creates it.
The CLI resolves the home once, from `--home`/`--spool-dir` or from the user's
own, and passes it in.

### Exit statuses

`ingest` exits **0** and prints nothing at all on success: a hook runs inside
the user's session, and a capture tool that echoes the payload back is one the
user can see in the thing it is capturing. It exits **1** and says why on stderr
when the spool cannot be written. Cursor never sees that status — the
registration ends in `|| true` — but a human running it by hand does, and a
spool that has been silently empty for a week is otherwise found a week late.

`hooks install|uninstall` print their result on stdout and exit **0**, or exit
**1** with the reason on stderr for a usage error and for a file that could not
be parsed.

## Summarising a spool (`profiler analyze`)

`AnalyzeSpool(dir)` reports what a spool directory contains. `profiler analyze`
prints it as JSON.

**It describes the files, and nothing about what the harness did.** No payload
in this repository has been produced by a running Cursor, so a reader that turns
`tool_name` into a tool call or `context_tokens` into a token count is asserting
the documentation is right — in the data, where a hedge in prose cannot follow
it, and a summary travels. Concretely, the summary carries no metric state, no
source and no signal result of any kind, and
`TestSpoolSummary_MakesNoSignalClaim` refuses those types by reflection so the
claim cannot be reintroduced later.

| field | |
|---|---|
| `dir` | the directory summarised |
| `files`, `lines` | `*.jsonl` files read, and non-blank lines in them |
| `envelopes`, `unreadable_lines` | the two parts `lines` divides into. A line that could not be decoded is **counted**, not skipped in silence: a corpus reported smaller than it is reads as a capture that did not happen |
| `stripped_envelopes` | lines written in metadata-only mode. Their payload bytes are not comparable with an unstripped line's |
| `schema_versions`, `event_names` | envelope field values, counted verbatim. A field that was absent counts under the empty string rather than under `unknown`, which is the writer's word for a payload that named no event |
| `capture_days` | lines per UTC day of the capture time — when the hook ran here, not a time the harness reported |
| `last_capture_at` | the newest capture time, as the maximum across every file rather than the last line of the last one |
| `conversation_ids` | distinct non-empty `conversation_id` values |
| `payload_bytes` | the stored size of the payloads. **Bytes, not tokens**: the draft divided them by four and called the result estimated context tokens |
| `object_payloads`, `other_payloads` | payloads the key census could walk, and those it could not. Input that was not JSON is stored as a JSON string on purpose, and is still a payload that arrived |
| `payload_keys` | how many payloads carried each top-level key. **This is the census**, and it is what a later adapter should be written against |
| `payload_key_values` | the values of those keys, for keys that are not content and whose value is a string of at most 64 characters |

Two bounds on the value census. **Content keys are counted and never quoted** —
the same set `--strict` replaces with sizes, asked of the one place it is
declared, because a summary is a document a user pastes into an issue. **Numbers
are not valued**, because a histogram of every number in a spool is a table of
measurements nobody made.

A spool directory that is not there is an **error**, not an empty summary, for
the reason `LoadSessionEvents` gives. `analyze` exits **0** with the summary on
stdout, or **1** with the reason on stderr.

`profiler/queries/` asks the same corpus bigger questions in DuckDB SQL, under
the same rule. Its column list is derived from `SpoolEvent` and its default
spool directory from `DefaultSpoolDir`, so neither can fall behind the writer.

## Reporting what a machine can measure (`profiler doctor`)

`DetectEnvironment(EnvironmentQuery)` reports what this build can turn into a
measured profile here. `profiler doctor` prints it as JSON and exits **0**
whatever it finds — detection is a report, not a gate — or **1** for a usage
error.

The document has two halves, and a reader cannot confuse them:

- **`measurement`** — what a registered harness actually read. `tier`,
  `harness`, `export`, the probe's own `signals` and a `reason`.
- **`observed`** — what is on this machine. `hooks_json` and `spool`: paths,
  counts, and why a read failed when it did. No tier, no state, no source.

### The tier is what a probe read, and nothing else

| tier | what it asserts |
|---|---|
| `none` | nothing was read that this build can turn into a profile |
| `export` | a registered harness probed the export supplied and reports at least one signal from a real source |

**The only input to a tier is a capability report from a probe over a real
file.** Nothing found on the machine can raise it, because nothing found on a
machine is a measurement: a file existing, a hook being registered and an
environment variable being set are all true of installations that measure
nothing. So `doctor` reports a measurement surface only when it is given
`--harness` and `--otel-file`, and everything else it prints is an observation.

`TestEveryEnvironmentTierIsOneACaptureDelivers` reads the tier constants out of
`doctor.go` and requires, for each one above `none`, an input from which
`DetectEnvironment` reports it **and** from which a capture of the same export
produces a `present` signal. A tier added without one turns that table red. It
is the fourth rule of the adapter contract, applied to the one capability claim
in this codebase that is not made by an adapter.

### The hook surface produces no signal in this release, and the report says so

There is no adapter that reads a spool. So there is **no hook tier**, and there
is no signal the spool can contribute to one. What `doctor` may say is that a
spool exists and how much is in it — `files`, `lines`, `unreadable_lines`,
`payload_bytes`, `last_capture_at` — and every spool observation ships this
sentence beside those counts, in the document rather than in the prose:

> no measurement: no adapter in this build reads the spool, so these lines are a
> capture to be parsed later and no profile, token count or tool call can be
> produced from them

What it may **not** say is that this machine can capture tokens, tool calls or
anything else from Cursor.

### Our registration, matched the way the install writes it

`hooks_json.registered_hook_events` is the events whose entries carry **exactly**
the command the query named, read through the same `readHooksDoc` and matched by
the same `entryCommand` that `hooks install` and `hooks uninstall` use. The CLI
resolves that command in one place for both subcommands, so `doctor` cannot come
to look for a string `install` does not write.

A `hooks.json` that exists and cannot be parsed is reported as **present with a
reason**, not as a machine with nothing registered; an unreadable spool
directory likewise. Either reported as absent would be the reader's own failure
told to the user as a fact about their machine.

### Three things it does not report

- **A Cursor user-data directory.** The draft read
  `~/Library/Application Support/Cursor` and reported `cursor_installed`. That
  path is macOS's only, so `false` means "not on a Mac" as often as "no Cursor"
  — and whether Cursor is installed is not a statement about what can be
  measured.
- **`CURSOR_ADMIN_API_KEY` as a `server_api` tier.** There is no Admin API
  client in this repository.
- **`OTEL_EXPORTER_OTLP_ENDPOINT` as an `enterprise` tier.** An endpoint is
  where a harness sends telemetry, not a file this tool can read.

`doctor` writes nothing, anywhere. It is a read of a home and a read of a file.

## Adapter implementations

### Claude Code adapter

**Telemetry surface:** OTel export via `CLAUDE_CODE_ENABLE_TELEMETRY=1` + `OTEL_*` env vars.

**Input format:** OTLP/JSON, as Claude Code emits it over `OTEL_EXPORTER_OTLP_PROTOCOL=http/json`. One `ExportMetricsServiceRequest` or `ExportLogsServiceRequest` per JSON object; a file may hold one object, one per line (NDJSON), or several concatenated, and all three read the same way. A top-level value that is not an object is not an export. README "Capturing an OTel export" documents the two routes that produce the file.

Within that envelope:
- Metrics are walked at every level — `resourceMetrics[].scopeMetrics[].metrics[]` — not just the first of each. Token counts come from `claude_code.token.usage` sum data points: the `type` attribute (`input`, `output`, `cacheRead`, `cacheCreation`, camelCase on the wire, snake_case in the profile) and the value from `asDouble`, falling back to `asInt`. Every 64-bit integer is accepted as a JSON number or a decimal string. That is narrower than the proto3 JSON mapping OTLP mandates, which also accepts exponent notation for an integer field; refusing exponent form is a deliberate deviation, recorded as such under "Deliberate deviations" below rather than claimed as conformance. A fractional `asDouble` is rounded, not truncated.
- **A value reaches the profile only if it is a count**: a whole number from 0 to 2⁶³−1. The rule is applied to the *rounded* value, so a `-0.4` that is float drift on a true zero is the zero it was, while `-0.5` is a number the exporter meant to be negative. A data point that fails this splits two ways, counted apart because they are two different things to go and look at in a capture. A leaf that **decoded as a finite number** whose rounded value is negative or 2⁶³ or greater *is not a count*: it is refused rather than converted — converting it is undefined in Go and gave the same export two different profiles on two architectures — and `asInt` is not consulted after it, because `asDouble` already answered. A leaf that **did not decode as a finite number at all** is *unreadable*, not "not a count": `NaN`, `Infinity` and a literal that overflows `float64` are refused by the float reader itself, so `asDouble` said nothing and `asInt` is read next — `{"asDouble":"NaN","asInt":"9"}` is the count 9, not a refusal. A point is unreadable only when neither leaf yields a number, which is the clause the reason uses: "carried no asDouble or asInt value that reads as a number". An `asDouble` is bounded at 2⁶³ and not at 2⁶³−1 because no `float64` is 2⁶³−1: the literal `9223372036854775807` parses to exactly 2⁶³, and only `asInt` can carry that count exactly. A token type that no data point carried has no key in the profile at all; a type read as zero has its key, with `0` in it.
- **Data points are merged per time series, and a series is what OTel says it is**: the resource it was exported from, the instrumentation scope that recorded it, the metric's name, and the data point's full attribute set — every attribute, of every `AnyValue` kind, and not the subset that renders as text. **Each `AnyValue` kind is canonicalised by what that kind is.** An `arrayValue`'s member order is part of its value and is kept; a `kvlistValue` is a map, and the OTel common data model defines two maps as equal irrespective of the order their members arrive in, so its members are sorted. Both recurse, so a map nested in an array or in another map normalises too, and a `null` members list is the empty one it decodes to under the proto3 JSON mapping. A *kind* written `null` is that kind **unset** rather than that kind holding its zero value, because ProtoJSON reads a null as the field being unset: `{"kvlistValue":null}` is an `AnyValue` of no kind — the same answer as `{}` — while `{"kvlistValue":{}}` is an empty map somebody set, and the two are two series. That is the rule the envelope is read by as well (`{"resourceMetrics":null}` says nothing about metrics), and it holds for every kind, scalars included; it does not reach a members list, because a repeated field's `null` *is* its default. Taking the compacted JSON as a composite's identity instead made one series read as two, and under cumulative temporality the two running totals **added** into a number no data point in the export carried — an over-count, which is the one direction "never invented" forbids. Two points differing in any of the four are two series and do not merge, so a capture that aggregates several resources or several scopes keeps them apart: two `resourceMetrics` entries whose data points carry identical attributes — a collector fanning in two `service.instance.id`s — are two running totals that add. A **resource** is its attribute set, so an entry carrying no `resource` and one carrying a resource with no attributes are one resource rather than two; `schemaUrl` is not part of it, because it declares which version of the semantic conventions the attributes follow, not a different origin. A **scope** is its name, its version and its attributes. Each of the four parts is encoded behind its byte length, so no part can spell another and no attribute value — a tool name, a prompt, a model id, all arbitrary text — can forge a foreign series' identity.
- **Temporality decides how the points of one series merge, and they are opposite instructions.** `aggregationTemporality` 1 (delta) means the points are the increments since the last export and add up; 2 (cumulative) means each point is a running total. A temporality that reads as neither is refused rather than assumed, and absent, unreadable and declared-but-neither are counted apart, because they are three different things to look at in a capture. **A series carrying both temporalities is refused whole**, for the same reason and by the same rule: the two are opposite instructions, so every way of resolving the mix invents a number the export does not contain — adding the increments to a running total counts them twice, and dropping them counts them not at all. The series contributes nothing and has no key in the profile, the refusal is counted per *series* rather than per data point (the points are individually fine; it is their company that is malformed), and the `tokens` reason names it: "1 time series carried both delta (1) and cumulative (2) aggregationTemporality points". Refusing one series is not refusing the export — the well-formed series beside it still count, and `tokens` is `unknown` only when no series survives. This is a change from v0.4.1, where a series carrying either temporality was treated as cumulative and the point with the latest `timeUnixNano` won — so which number a mixed series produced depended on the timestamps, not on write order. (Checked against the v0.4.1 binary: a cumulative 50 at t9000 written before a delta 100 at t1000 gave 50.) **What the profile can say about a refusal depends on what else survived.** When no series survives, `tokens` is `unknown` and the reason names the refused series. When a well-formed series survives beside it, `tokens` is `present` with a total the refused series is missing from, and schema v1 has no field to say so: a `present` result carries no reason. That is a gap in the schema rather than in the count, and it is not closed by 0.5.0: it needs a caveat channel schema v1 has not got, and it belongs beside the skipped-record count.
- **A cumulative series is a set of runs, and `startTimeUnixNano` identifies one.** The points sharing a start are one run, and what that run holds is the **greatest running total any of them reported**. These are monotonic counters: a running total only goes up, so the values carry their own order, every point of a run is a prefix of the greatest, and a flush reporting less than an earlier one on the same run is a capture contradicting itself rather than tokens given back. `timeUnixNano` is **not read** by the merge. Ordering by it lets such a flush take the run down with it, and it is also what rules out the obvious repair: a merge that kept the ordering and let each run hold its latest point by `timeUnixNano` would let supplying `startTimeUnixNano` — the field that says which run a point is from — make the profile report *fewer* tokens than leaving it out, which is the property below inverted. (v0.4.1 does not exhibit that, because it does not read `startTimeUnixNano` at all; it is a property of the candidate, measured against a build of it.) A point carrying a *different* start is a counter that restarted: its run sits beside the earlier one rather than replacing it, and the series contributes the sum of its runs, so a capture spanning a restart reports both — 100 over `startTimeUnixNano` 1–10 followed by 20 over 11–20 is 120. Runs are identified by their start, not by file order, so batches written out of order still add up correctly. **A point whose `startTimeUnixNano` is absent, zero or unreadable** cannot say which run it came from — but it came from one, either a run that named itself or a run nothing else in the capture observed, so it is neither a run of its own nor free. What the capture guarantees is the **least total over every run it could have come from**, and the cheapest such run is the one that already reached furthest: the series contributes `(its runs added) + max(0, the unplaceable total − its largest run)`. Both ends of that are wrong in a direction already shipped. Counting the point as a run of its own adds mass no point ever reported — one flush that omitted one field then doubles a session. Taking the series to be "the greatest running total observed anywhere on it" throws away the runs the point did not join, which are tokens the session really spent. Four consequences worth stating: a capture carrying no start times at all has no runs to place its points against, so it holds the greatest running total they reported (v0.4.1 held the latest by `timeUnixNano` instead, which is the same number wherever an exporter wrote its flushes in time order and a lower one where it did not — `900` at t1001 then `500` at t1002 gave 500 at v0.4.1 and gives 900 here); a session whose last flush omits `startTimeUnixNano` — one series, one run, 800000 then 1200000 then an unplaced 1200050 — reports 1200050, not 2400050; a capture whose runs reach 100 and 20 beside an unplaced point reporting 500 reports **520**, because the 500 is a running total of the run that reached 100 and the other run's 20 is still beside it — 500 drops a run that named itself and 620 invents a third; and a capture whose runs reach 100, 100 and 100 beside an unplaced 150 reports **350**, the case where the unplaceable total is above the largest run but below their sum, so a rule that compares it only with the sum never fires and loses 50 tokens the points reported. The property behind all four: taking information away — erasing a start time — may lower the total and may never raise it. `startTimeUnixNano` **`0` is `startTimeUnixNano` absent**: it is a proto3 `fixed64` and OTLP mandates the proto3 JSON mapping, whose deviations do not touch default values, so an explicit `0` and an omitted field are two encodings of one message — `protojson` writes `"0"` with `EmitUnpopulated` on and omits the field with it off — and the same holds for `timeUnixNano`. An **exponent-form** number (`1.7893e18`) does not read as a start time even when it is an exact integer: reading it means going through a `float64`, which cannot represent every nanosecond instant, and two points of one run that round apart would become two runs. That refusal is a deliberate deviation from the proto3 JSON mapping, recorded under "Deliberate deviations" below. Since an unreadable start now costs only what its point exceeds the largest run by, refusing it is the cheap direction. A **delta** series is one sum whatever its points' start times say: a delta point's `startTimeUnixNano` is the start of that point's own interval rather than of a run, and it is not read.
- Log records are identified by their `body`, which carries the fully-qualified event name (e.g. `claude_code.tool_result`) and is **taken as it stands**. A body that names an *unqualified* event — a bare `tool_result` — is refused rather than re-qualified: the qualified name is what this harness emits, so adding the prefix would manufacture an identity the record never had and attribute another product's log record to this profile. There is no fallback out of a body that said something. The `event.name` attribute is read only when the body says nothing at all — absent, not readable as a string, or an empty one — and only that attribute is qualified, because the short form is its documented spelling. A body may be an object, a bare string, or absent. Unknown event names are ignored.
- Timestamps are read from `timeUnixNano`, nanoseconds since the epoch as a decimal string or a number, and recorded in the profile as RFC 3339 timestamps.
- **Data points and log records that cannot be read are skipped, and the profile does not report how many.** A `present` result carries no reason in schema v1; surfacing a count of skipped records needs a channel 0.5.0 did not add, and two more cases belong in that same channel. A **refused time series**: a `present` total reduced by a malformed series beside a healthy one is invisible today. And **records the provenance projection removed while others survived**: when *some* of a signal's records lack the asserted `session.id` — or carry another session's, or arrived under a foreign scope — the surviving ones make the signal `present`, so the value is silently reduced and carries no reason at all, because the "not read" clause is built only on the `unknown` paths. All three are one gap with one shape: a `present` result has no field in which to say what it did not count. When *no* record survives, every signal is `unknown` and the clause does name how many were passed over and why, which is why this is a schema gap rather than a counting one.

**Deliberate deviations from the proto3 JSON mapping.** Two, stated here so this is not mistaken for a conformance claim. They run in opposite directions: one refuses a form the mapping accepts, the other accepts a form the mapping refuses.

**Narrower than the mapping — exponent form is refused.** A 64-bit integer field written in **exponent form** (`1.7893e18`) is refused, though the mapping accepts it. Reading it means going through a `float64`, whose spacing at that magnitude is 256ns, and `startTimeUnixNano` is a run identity — two points of one run that round apart would become two runs, and an approximate identity is worse than none. No exporter going through `protojson` emits integers that way, so the cost is theoretical and the risk is not. The same rule is applied to every 64-bit integer leaf rather than to timestamps alone, because one number-reading rule is easier to hold than two.

**Wider than the mapping — a quoted decimal enum is accepted.** `aggregationTemporality` is read from a bare number, from the enum name, *and* from a **quoted decimal** (`"aggregationTemporality": "2"`). The mapping takes the first two and refuses the third: it reads a JSON string as an enum *name*, and `"2"` is not the name of anything. Checked against the reference `protojson` on the OTLP `Sum` message — `2` and `"AGGREGATION_TEMPORALITY_CUMULATIVE"` unmarshal, `"2"` fails with `invalid value for enum field aggregationTemporality`. Nothing observed produces the quoted form, and the reader's own comment puts it no higher than a *may*; the same reference `protojson` writes the enum *name*, not the digit. The acceptance rests on what refusing would cost rather than on a sighting: a temporality that does not read is expensive, because it decides how an entire series merges, and the alternative is refusing a series over its punctuation. `cumulative.ndjson` carries the quoted form so the acceptance stays pinned, and is the measurement of that cost — making just that batch's temporality unreadable instead drops the capture's `input` total from 1150 to 750, the series holding the 500 it had already reached rather than the 900 the quoted batch carries. It is a fixture written to pin the acceptance, not an observation of a producer. This one widens what is read and refuses nothing the mapping accepts, which is why it is the safer direction of the two.

**Failure classification:** `error` means the file could not be read as an OTLP/JSON export — unreadable, empty or whitespace-only, a top-level value that is not a JSON object, malformed JSON, or a value that does not fit the OTLP schema. No reason quotes a Go type. Two of those five sit inside a batch and name it (1-based): malformed JSON, and a value that does not fit the OTLP schema — which also names the field path it found the wrong type at, when the decoder reports one. A batch whose own top level is the wrong shape (an array where the envelope should be) has no field path inside it to name, and that reason names the batch alone. Malformed JSON is the only one with a single byte to point at, so it is the only reason that carries an offset; the rest name the shape or the failure, because no offset would mean anything for them. A parse failure is never swallowed: nothing from earlier batches is used, and past batch 1 the reason says what to do about a capture stopped mid-write.

**One convention for byte offsets:** the 0-based offset, in the capture file, of the first byte the decoder could not accept — counting the byte-order mark and any leading whitespace, because the reader's editor counts them. When the file ended mid-object there is no such byte, and the reason names the file's length, which is where the byte the decoder wanted would have been.

`unknown` means a well-formed OTLP/JSON export that carries no telemetry — for one signal, or, when no batch carries an OTLP envelope, for all three. **The envelope test is a value, not a key**: a batch carries an envelope when `resourceMetrics` or `resourceLogs` has a *value* under it. An empty list is a value, so `{"resourceMetrics":[]}` *is* an OTLP export of a session that emitted nothing, and each signal is `unknown` with its own reason. A JSON `null` is not a value — ProtoJSON reads `null` as the field's default, so a batch written `{"resourceMetrics":null}` said nothing about metrics — and a file whose batches carry a value for neither field is not a broken export, it is not an export: all three signals get the "this is not OTLP/JSON" reason. Naming the key is not enough, and `{"resourceMetrics":null}` is the case that tells the two apart.

**Timing is a span over API requests.** `start_time`, `end_time` and `total_ms` cover the earliest to the latest `claude_code.api_request` record, so they exclude the prompt before the first request and any tool activity after the last one. Per-request `duration_ms` and the `claude_code.active_time.total` metric measure different quantities and have no field in schema v1; 0.5.0 added neither field.

**Probe logic:**
0. Probe is **not session-scoped**, because it is not given a session: it reports what the export can yield for the session the export belongs to. The instrumentation-scope half of the provenance test still applies. A capture names a session and reads only that session's records, so an export carrying several sessions can probe `otel` and capture `unknown` for a session it does not contain. A profile cannot disagree with itself: the `capability` block in a profile is derived from that capture's own scoped resolution, not from this one.
1. Probe resolves the export file exactly as capture does — one read, one parse, one extractor per signal — and reports a capability as `otel` only when that extractor produced a value:
   - a `claude_code.token.usage` sum declaring an `aggregationTemporality` of 1 or 2 — spelled as a bare number, as a quoted digit, or as the protobuf JSON enum name (`AGGREGATION_TEMPORALITY_DELTA`, `AGGREGATION_TEMPORALITY_CUMULATIVE`; `AGGREGATION_TEMPORALITY_UNSPECIFIED` reads as 0) — with a data point carrying a recognised `type` (`input`, `output`, `cacheRead`, `cacheCreation`) and an `asDouble` or `asInt` value that is a count → `tokens: otel`
   - `claude_code.tool_result` log events carrying a `tool_name` and a readable `success`, or `claude_code.tool_decision` events recording a reject with a `tool_name` → `tool_calls: otel`
   - `claude_code.skill_activated` log events carrying a `skill.name` → `skill_activation: otel`
   - `claude_code.api_request` log events carrying a parseable `timeUnixNano` → `timing: otel`
   If the file is absent, unreadable, malformed, missing the signal, or carrying the signal with nothing readable inside it, that capability stays `none`. Availability is a fact about a value in hand, not about a name matched in a file — a probe that reports structure is how it comes to advertise data the capture cannot deliver.

   A capability's vocabulary is a source or `none`, so the report structurally cannot distinguish "no telemetry was configured" from "the export you named could not be read": both are `none`, while capture keeps them apart as `unknown` and as `error` with a reason. That distinction is information `resolve` already computed, so probe reports it — on **stderr**, through `ProbeDiagnoser`, one message per distinct reason, using capture's own wording. Adding it to `CapabilityReport` instead would change what every `Profile` that embeds the report contains for the same input, which is a schema and adapter-version question; a second channel changes nothing anyone parses. `Probe()` delegates to `ProbeWithDiagnostics()` so there is one read of one file and the report and its explanation cannot describe different files.
2. Attribution is always `none` for Claude Code, because there is nothing to read: its telemetry carries no output-to-skill mapping at all. Skill activation was `none` for the same-shaped but different reason — the telemetry existed and this adapter did not read it — until 0.5.0, which reads it. **The source is the event, not the attribute.** `claude_code.skill_activated` is logged when a skill is invoked, through the Skill tool or a `/` command, and only then, so one record is one activation and the timestamp on it is the time the skill was invoked. `skill.name` also rides along on `token.usage`, `cost.usage`, `api_request`, `api_error` and `api_refusal`, where it marks the skill active *for that request* — a skill used across five requests carries it five times, and those records carry flush and request times rather than invocation times. Reading them as activations would report a count and a set of times the harness never recorded, so an export carrying the attribute and no event gives `skill_activation: "none"` in the capability report and `{"state": "unknown", "reason": …}` in the profile, the reason naming the event that was looked for — a capability value carries no reason, and a signal state does, which is why both have to be said. A reason may say what this adapter does not read; it may not say what the harness does not emit unless that is true.

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
it one is new surface rather than a repair, and **0.5.0 did not add one**; a caller that
must branch on a bad export uses `capture`, which does have one.

`profiler capture` writes the profile to stdout in every case, and exits **2** when no signal was read and at least one is `error` — a supplied export that could not be used — and **0** otherwise, including an all-`unknown` profile from a session with no telemetry configured. The status is what a wrapping script branches on; exiting 0 after reading nothing would have it store the all-unknown profile as a successful capture. Every usage error exits **1** before any capture happens: an unknown command or harness, a missing required flag, `--export-file`, and an unrecognised flag. The last one is why the flag sets use `ContinueOnError` — `flag.ExitOnError` exits 2 on its own, and 2 has to mean exactly one thing for a script to branch on it.

**Fallback:** With no export file configured, `tokens`, `tool_calls`, `skill_activation` and `timing` are `unknown` with reason "OTel export not configured. Provide an OTel export file via --otel-file or OtelExportFile." A file that is configured but cannot be read as an OTLP/JSON export must not borrow that reason: missing, unreadable, empty, non-object at the top level, malformed, or not fitting the OTLP schema are all `error`, naming the failure, because the caller did supply a file and "not configured" would send them to fix the one thing that is not wrong. A file that parses but carries nothing readable for a signal leaves that signal `unknown`, naming what was missing; a file carrying no OTLP envelope at all leaves all four `unknown`, naming the format expected. `attribution` is `unknown` with its own reason (step 7 above) in every one of these cases — it is a property of the harness and of this adapter, not of the export. The adapter reads only the file it is given: it does not inspect `CLAUDE_CODE_ENABLE_TELEMETRY` or the `OTEL_*` env vars itself.

## Acceptance criteria

These are the normative criteria for the Claude Code adapter as it ships, spanning
every release that has touched it — AC2 and AC4's skill-activation clauses are
0.5.0's, AC3's mixed-temporality clause is 0.4.3's — and not, as an earlier
heading said, the scope of one slice.

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
