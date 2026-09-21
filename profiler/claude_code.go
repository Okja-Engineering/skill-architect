package profiler

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// OTel signal names emitted by Claude Code. Named once and read once: the same
// extractor decides both what the probe can advertise and what the capture
// returns, so there is no second copy of a name to drift.
const (
	otelTokenUsageMetric = "claude_code.token.usage"
	otelToolResultLog    = "claude_code.tool_result"
	otelToolDecisionLog  = "claude_code.tool_decision"
	otelAPIRequestLog    = "claude_code.api_request"
	// otelSkillActivatedLog is logged when a skill is invoked, through the
	// Skill tool or a / command, and only then — so one record is one
	// activation. A skill.name attribute also rides along on request-scoped
	// signals (token.usage, cost.usage, api_request, api_error, api_refusal),
	// marking the skill active for that request; those are not read as
	// activations, because a skill used across five requests carries the
	// attribute five times and reporting five activations would be a count
	// nobody measured.
	otelSkillActivatedLog = "claude_code.skill_activated"

	// Every Claude Code event name is qualified with this prefix in a record's
	// body and unqualified in its event.name attribute.
	otelEventPrefix = "claude_code."

	// otelSessionAttr is the attribute Claude Code puts the run's identity on.
	// Every metric data point and every log record carries it — see README,
	// "A capture identifies you" — which is what makes a session-scoped read
	// possible at all.
	otelSessionAttr = "session.id"
)

// Token type attribute values on claude_code.token.usage. They are camelCase on
// the wire and snake_case in the profile: the profile's keys are schema v1 and
// do not follow the harness.
const (
	tokenTypeInput         = "input"
	tokenTypeOutput        = "output"
	tokenTypeCacheRead     = "cacheRead"
	tokenTypeCacheCreation = "cacheCreation"
)

// ClaudeCodeAdapter captures runtime signals from Claude Code via OTel export.
//
// Claude Code emits OTel when CLAUDE_CODE_ENABLE_TELEMETRY=1 and OTEL_*_EXPORTER
// env vars are set. It has no file exporter, so the file this adapter reads is
// written by something downstream: a local OTLP/HTTP receiver or an OpenTelemetry
// collector's file exporter. Either way the format is OTLP/JSON — one
// ExportMetricsServiceRequest or ExportLogsServiceRequest per JSON object — and
// README "Capturing an OTel export" documents both routes. Reading a file rather
// than a live endpoint keeps the adapter self-contained and testable without a
// running collector.
//
// Attribution is always unknown for Claude Code, because there is nothing to
// read: no signal maps an output back to the skill that produced it, and
// reporting one would mean inventing it. Skill activation is a different case
// and is read — from claude_code.skill_activated, the event the harness logs
// once per invocation.
type ClaudeCodeAdapter struct {
	// OtelExportFile is the path to a file containing an OTLP/JSON export.
	// It is the adapter's only input: Probe and Capture both resolve this one
	// path, and nothing else supplies one.
	OtelExportFile string
}

// Name returns the harness identifier.
func (a ClaudeCodeAdapter) Name() string { return "claude_code" }

// otelSignals is one export file resolved into the signals it carries.
//
// Probe and Capture both derive from this single resolution, so "the adapter can
// produce this signal" and "the adapter produced this signal" are by construction
// the same question, answered by the same code. Two separate predicates — one
// scanning for structure, one extracting values — are what let the probe
// advertise data the capture then discarded.
type otelSignals struct {
	Tokens     TokenResult
	ToolCalls  ToolCallResult
	Activation ActivationResult
	Timing     TimingResult
}

// resolvedSignal is one settled signal under the name the capability report and
// the profile key it by.
type resolvedSignal struct {
	metric MetricName
	raw    RawMetricResult
}

// resolved is every signal this adapter reads out of the export.
//
// It is the single enumeration of that set: the capability report and the probe
// diagnostics are both derived from it, so neither can fall a signal behind the
// other — which is how a signal comes to be advertised by one and explained by
// neither. Ordered rather than a map, because the diagnostics are printed and a
// reader comparing two runs should not have to sort them first.
func (s otelSignals) resolved() []resolvedSignal {
	return []resolvedSignal{
		{MetricTokens, s.Tokens.RawMetricResult},
		{MetricToolCalls, s.ToolCalls.RawMetricResult},
		{MetricSkillActivation, s.Activation.RawMetricResult},
		{MetricTiming, s.Timing.RawMetricResult},
	}
}

// resolve reads and parses the export file once, projects it onto the
// provenance it was given, and settles every export-backed signal from the
// projection.
//
// The extractors are handed the projection and never the file, so a signal is
// scoped to the named session by construction rather than by each extractor
// remembering to scope itself — which is what lets a new signal join the set
// without reopening the defect that scoping exists to close.
//
// It owns file resolution, provenance and failure classification for this
// adapter; nothing below it re-decides any of them, so every signal shares one
// error path:
//
//   - no file configured — unknown, naming what to configure;
//   - the file could not be read as an OTLP/JSON export — error, naming the
//     failure. The export was supplied, so "not configured" would send the
//     caller to fix the one thing that is not wrong;
//   - it parsed but carries no OTLP envelope — unknown, naming the format
//     expected. Nothing failed; the file simply is not an export;
//   - parsed — the export is projected onto prov, and each extractor settles
//     its own signal from what it can read of the projection. An export that
//     carries nothing of prov's is not a failure of the export: each signal
//     reports for itself that it read nothing, and says what was there instead.
func (a ClaudeCodeAdapter) resolve(prov provenance) otelSignals {
	if a.OtelExportFile == "" {
		return unknownSignals("OTel export not configured. Provide an OTel export file via --otel-file or OtelExportFile.")
	}

	f, err := os.Open(a.OtelExportFile)
	if err != nil {
		return erroredSignals(fileReadError(err).Error())
	}
	defer f.Close()

	export, err := readOTLP(f)
	if err != nil {
		// The format layer's error is already written for the reader: it names
		// the shape that could not be read, never a Go type.
		return erroredSignals(err.Error())
	}
	if !export.hasEnvelope() {
		return unknownSignals("OTel export is not OTLP/JSON: no resourceMetrics or resourceLogs found. " +
			"Claude Code emits OTLP via OTEL_EXPORTER_OTLP_PROTOCOL=http/json; see README 'Capturing an OTel export'.")
	}

	scoped := export.scopedTo(prov)

	return otelSignals{
		Tokens:     extractTokenCounts(scoped),
		ToolCalls:  extractToolCalls(scoped),
		Activation: extractActivations(scoped),
		Timing:     extractTiming(scoped),
	}
}

// provenanceFor is the test a record must pass to be read into a profile of
// sessionID.
//
// The namespace is the adapter's own Name(), which is the same token its signal
// names are qualified with, so there is no second copy of the harness's
// identity in the code to drift from the first.
func (a ClaudeCodeAdapter) provenanceFor(sessionID string) provenance {
	return provenance{
		namespace:   a.Name(),
		sessionAttr: otelSessionAttr,
		sessionID:   sessionID,
	}
}

// probeProvenance is the same test with the session half dropped, because probe
// is asked what an export can yield without being told which session. It is the
// one place anySession is set, and it is not reachable from Capture: Capture
// refuses an empty session id rather than falling back to this.
func (a ClaudeCodeAdapter) probeProvenance() provenance {
	return provenance{
		namespace:   a.Name(),
		sessionAttr: otelSessionAttr,
		anySession:  true,
	}
}

// unknownSignals and erroredSignals settle every signal the same way, for the
// failures that belong to the export as a whole rather than to one signal.

func unknownSignals(reason string) otelSignals {
	return otelSignals{
		Tokens:     UnknownTokenResult(reason),
		ToolCalls:  UnknownToolCallResult(reason),
		Activation: UnknownActivationResult(reason),
		Timing:     UnknownTimingResult(reason),
	}
}

func erroredSignals(reason string) otelSignals {
	return otelSignals{
		Tokens:     ErrorTokenResult(reason),
		ToolCalls:  ErrorToolCallResult(reason),
		Activation: ErrorActivationResult(reason),
		Timing:     ErrorTimingResult(reason),
	}
}

// capabilityReport derives the capability report from resolved signals, so a
// capability is advertised exactly when a value was read for it.
func (a ClaudeCodeAdapter) capabilityReport(sig otelSignals) CapabilityReport {
	caps := map[MetricName]MetricSource{
		// Nothing Claude Code emits maps an output back to the skill that
		// produced it, so this adapter can offer no attribution signal whatever
		// the OTel configuration.
		MetricAttribution: SourceNone,
		// An estimate is derived from hook payloads, and this adapter reads an
		// OTel export. It has nothing to estimate over, so it says so rather
		// than leaving the signal out: the report enumerates every signal a
		// profile carries, and a signal nobody advertised is one no caller can
		// tell was considered.
		MetricEstimatedContextTokens: SourceNone,
	}
	// The signals that come out of the export are advertised from the values
	// resolve actually read, walked rather than listed a second time here.
	for _, s := range sig.resolved() {
		caps[s.metric] = sourceOf(s.raw)
	}
	return CapabilityReport{
		Harness:      a.Name(),
		AdapterVer:   AdapterVersion,
		ProbedAt:     time.Now().UTC().Format(time.RFC3339),
		Capabilities: caps,
	}
}

// sourceOf reports where a signal's value was read from, or "none" when no value
// was read. Availability is a fact about a value in hand, not about structure
// spotted in a file.
func sourceOf(r RawMetricResult) MetricSource {
	if r.State != MetricPresent {
		return SourceNone
	}
	return MetricSource(r.Source)
}

// Probe inspects the environment and returns what metrics this adapter can
// produce. A capability is "otel" only when the export file yielded a value for
// that signal; a missing, unreadable, malformed, or signal-less file reports
// "none".
//
// "none" is all the report can say, so a caller who needs to know whether a
// supplied export was the problem wants ProbeWithDiagnostics instead.
//
// Probe is not session-scoped, because it is not given a session: it answers
// what the export can yield for the session the export belongs to. A capture
// names a session and reads only that session's records, so an export carrying
// several sessions can probe "otel" and capture "unknown" for a session it does
// not contain. The profile's own capability block is derived from the capture's
// scoped resolution, not from this one, so a profile can never disagree with
// itself.
func (a ClaudeCodeAdapter) Probe() CapabilityReport {
	report, _ := a.ProbeWithDiagnostics()
	return report
}

// ProbeWithDiagnostics resolves the environment once and returns the capability
// report alongside the reasons any signal failed, implementing ProbeDiagnoser.
//
// This is the whole probe path now, and Probe delegates to it, so the report
// and its explanation are always computed from one read of one file and cannot
// disagree. That one read is scoped by probeProvenance — every session, but no
// foreign instrumentation scope — so the reasons account for exactly the
// records the report was derived from.
func (a ClaudeCodeAdapter) ProbeWithDiagnostics() (CapabilityReport, []string) {
	sig := a.resolve(a.probeProvenance())
	return a.capabilityReport(sig), failureReasons(sig)
}

// failureReasons collects the distinct reasons signals came back MetricError —
// a source that existed and failed, which is this run's problem rather than the
// session's.
//
// MetricUnknown is deliberately not included. "No export was configured" and
// "the export carried nothing for this signal" are answers about the session;
// capture exits 0 on them for the same reason. A probe that complained about
// those would be noise on every ordinary run, and noise is how a real
// diagnostic gets ignored.
//
// Deduplicated because a whole-export failure settles every signal with one
// reason, and saying it four times reads as four faults.
func failureReasons(sig otelSignals) []string {
	resolved := sig.resolved()
	seen := make(map[string]bool, len(resolved))
	var reasons []string
	for _, s := range resolved {
		r := s.raw
		if r.State != MetricError || r.Reason == "" || seen[r.Reason] {
			continue
		}
		seen[r.Reason] = true
		reasons = append(reasons, r.Reason)
	}
	return reasons
}

// Capture reads telemetry for a specific Claude Code session and produces a Profile.
//
// sessionID is what the capture reads by, not only what the profile is stamped
// with: an export legitimately carries several sessions, so only the records
// carrying this session.id contribute to the numbers below.
func (a ClaudeCodeAdapter) Capture(sessionID string, opts CaptureOpts) (Profile, error) {
	// The adapter owns its input contract. CaptureOpts.ExportFile is a session
	// export for an adapter that reads one; this adapter reads OTLP/JSON and
	// nothing else, so a caller who supplies one is told, rather than handed a
	// profile that looks like missing telemetry.
	if opts.ExportFile != "" {
		return Profile{}, ExportFileUnsupportedError(a.Name())
	}
	// The same rule for the identity. An empty session id is not a session that
	// matched nothing; it is no assertion at all, and a profile stamped with it
	// would have to either report every session in the export — the defect this
	// scoping exists to remove — or report a session nobody named. Refusing it
	// is what makes --session load-bearing in the library and not only in the
	// CLI that restates the requirement.
	if sessionID == "" {
		return Profile{}, SessionIDRequiredError(a.Name())
	}

	sig := a.resolve(a.provenanceFor(sessionID))

	return Profile{
		Schema:       ProfileSchema,
		ProfiledAt:   time.Now().UTC().Format(time.RFC3339),
		Harness:      a.Name(),
		SessionID:    sessionID,
		SnapshotHash: opts.SnapshotHash,
		SkillDir:     opts.SkillDir,
		Capability:   a.capabilityReport(sig),

		Tokens:          sig.Tokens,
		ToolCalls:       sig.ToolCalls,
		SkillActivation: sig.Activation,
		Timing:          sig.Timing,

		// Nothing in Claude Code's telemetry maps an output back to the skill
		// that produced it, so attribution has no source to read and is the one
		// signal here whose answer is the same for every export.
		Attribution: UnknownAttributionResult("Claude Code telemetry carries no output-to-skill mapping"),
	}, nil
}

// eventName is a log record's fully-qualified event name, or "" for a record
// whose identity cannot be read — which is not an error: an export carries
// events this adapter does not read, and new ones arrive with every release.
//
// The body is the primary identity and carries the qualified name; the
// event.name attribute carries the short one and can be dropped by attribute
// cardinality limits, so the body is preferred. Normalising to the qualified
// form keeps the names in the reasons below greppable against the user's own
// export.
func eventName(r otlpLogRecord) string {
	if name, ok := bodyName(r.Body); ok && name != "" {
		// The body carries the qualified name, and qualifying it here instead
		// would manufacture an identity: a body naming a bare tool_result is a
		// record of something that is not one of this harness's events, and
		// adding the prefix to it promoted another product's log record into
		// the profile's tool_calls.
		if !strings.HasPrefix(name, otelEventPrefix) {
			return ""
		}
		return name
	}
	// Only the event.name attribute is qualified here, because the short form
	// is the documented spelling of it and the body — the record's primary
	// identity — said nothing.
	name, _ := r.Attributes.String("event.name")
	if name == "" {
		return ""
	}
	return otelEventPrefix + strings.TrimPrefix(name, otelEventPrefix)
}

// Each extractor is the single predicate for its signal: present only when it
// read at least one usable value, unknown with a reason naming why not. A metric
// or event with the right name but nothing readable inside it is evidence that
// the harness was running, not evidence of a value.
//
// Data points and log records that cannot be read are skipped, and the profile
// does not report how many: a present result carries no reason in schema v1.

func extractTokenCounts(export scopedExport) TokenResult {
	acc := counterAccumulator{}
	var seen, points int
	var c tokenPointCounters

	for m := range export.metrics(otelTokenUsageMetric) {
		seen++
		if m.Sum == nil {
			// A gauge or a histogram: the name matched, but there are no sum
			// data points to visit.
			continue
		}
		temporality, temporalityState := m.Sum.temporality()
		for _, dp := range m.Sum.DataPoints {
			points++

			// Delta and cumulative are opposite instructions — add, or
			// supersede — so a temporality that reads as neither is not a third
			// instruction to guess at. The point is refused and counted, which
			// costs a number nobody could have trusted and states so in the
			// reason, rather than silently over- or under-counting the session.
			//
			// Three ways to get here, counted apart because they are three
			// different things to look at in the capture: the sum declared no
			// temporality, it declared one that does not read, or it declared
			// one that reads as neither delta nor cumulative.
			switch {
			case temporalityState == temporalityAbsent:
				c.absentTemporality++
				continue
			case temporalityState == temporalityUnreadable:
				c.unreadableTemporality++
				continue
			case temporality != temporalityDelta && temporality != temporalityCumulative:
				c.otherTemporality++
				continue
			}

			// type is overloaded across Claude Code's metrics, so it is read
			// only on this one; another metric's type contributes nothing.
			tokenType, ok := dp.Attributes.String("type")
			if !ok || !isTokenType(tokenType) {
				c.unrecognisedType++
				continue
			}
			value, kind := dp.count()
			switch kind {
			case valueUnreadable:
				c.unreadableValue++
				continue
			case valueNotACount:
				c.notACount++
				continue
			}
			// timeUnixNano is not read here. What a run holds is the greatest
			// running total its points reported, and an instant could only
			// stand in for an order those values already carry.
			acc.add(m.series(dp), counterPoint{
				label:      tokenType,
				value:      value,
				start:      readNanos(dp.StartTimeUnixNano),
				cumulative: temporality == temporalityCumulative,
			})
		}
	}

	// A point can also be refused for the company it keeps: delta and
	// cumulative on one series are opposite instructions, and the accumulator
	// refuses that series whole rather than picking a winner. That is a defect
	// of the series, not of any point in it, so it is counted here — after the
	// walk, where the series are what is left — and the totals are what
	// survived it, not what was read.
	//
	// The count reaches a reader only when nothing survived, because the reason
	// below is built only on that path. A profile/v1 present result carries no
	// reason at all, so a series refused beside a healthy one is a total
	// reduced by a defect the profile has no field to name — the same gap as a
	// skipped data point, tracked with it for 0.5.0. Inventing a channel for it
	// inside v1 would mean a reason on a present result, which is a schema
	// change wearing a bug fix's clothes.
	totals := acc.reduce()
	c.mixedTemporality = acc.refused()

	// Every unknown reason carries what the provenance projection removed, so a
	// reason saying this signal was not found cannot be read as saying the
	// export carries nothing of the kind. The clause is empty when nothing was
	// removed.
	switch {
	case seen == 0:
		return UnknownTokenResult("no " + otelTokenUsageMetric + " metric found in OTel export" + export.metricsNotRead())
	case points == 0:
		return UnknownTokenResult("no readable " + otelTokenUsageMetric + " metric in OTel export: it carried no sum data points" + export.metricsNotRead())
	case len(totals) == 0:
		return UnknownTokenResult("no readable " + otelTokenUsageMetric + " metric in OTel export: " + c.reason() + export.metricsNotRead())
	}
	return PresentTokenResult(tokenCounts(totals), string(SourceOtel))
}

// tokenPointCounters is what the walk observed over claude_code.token.usage
// data points, and the only thing the reason is built from — so a reason cannot
// state a number the walk did not count. At most one defect is counted per data
// point, in the order the walk tests them, so no point is counted twice.
//
// One counter counts series rather than data points, because one defect belongs
// to a series rather than to any point in it: the points of a mixed-temporality
// series are individually fine and it is their company that is malformed.
// Reporting it as a count of points would send the reader looking for a bad
// point that is not there.
//
// The reason is composed from the counters rather than fused into one sentence
// naming several defects at once, for the same reason the tool-call reason is:
// a sentence that names four independent defects is true of none of the inputs
// that carry only one of them.
type tokenPointCounters struct {
	absentTemporality     int
	unreadableTemporality int
	otherTemporality      int
	unrecognisedType      int
	unreadableValue       int
	notACount             int
	mixedTemporality      int // series, not data points
}

func (c tokenPointCounters) reason() string {
	clauses := make([]string, 0, 7)
	add := func(n int, rest string) {
		if n > 0 {
			clauses = append(clauses, quantity(n, "data point")+" "+rest)
		}
	}
	// The same, for the one counter whose unit is a series. "time series" is
	// its own plural, so it goes through quantityOf.
	addSeries := func(n int, rest string) {
		if n > 0 {
			clauses = append(clauses, quantityOf(n, "time series", "time series")+" "+rest)
		}
	}
	add(c.absentTemporality, "carried no aggregationTemporality")
	add(c.unreadableTemporality, "carried an aggregationTemporality that could not be read")
	add(c.otherTemporality, fmt.Sprintf("declared an aggregationTemporality that is neither %d (delta) nor %d (cumulative)",
		temporalityDelta, temporalityCumulative))
	add(c.unrecognisedType, "carried no recognised type attribute ("+
		tokenTypeInput+"/"+tokenTypeOutput+"/"+tokenTypeCacheRead+"/"+tokenTypeCacheCreation+")")
	add(c.unreadableValue, "carried no asDouble or asInt value that reads as a number")
	add(c.notACount, "carried a value that is not a token count: a count is a whole number from 0 to 9223372036854775807")
	addSeries(c.mixedTemporality, fmt.Sprintf("carried both delta (%d) and cumulative (%d) aggregationTemporality points",
		temporalityDelta, temporalityCumulative))
	return strings.Join(clauses, "; ")
}

// quantity renders a count with its noun, agreeing in number: "1 data point",
// "3 data points". Every clause in this file is built through it, so no reason
// can tell the reader about "1 data points".
func quantity(n int, noun string) string { return quantityOf(n, noun, noun+"s") }

// quantityOf is quantity for a noun whose plural is not the singular plus "s" —
// "time series" is its own plural, and a reason saying "1 time seriess" is a
// reason nobody finishes reading.
func quantityOf(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}

// tokenCounts carries the totals that were read into the profile's counts:
// camelCase attribute values in, snake_case keys out. A token type no data
// point carried is left unset, so its key is absent from the profile rather
// than reported as a measured zero — and a type that was read as zero keeps its
// key, because a read zero is a measurement. Reasoning has no source in Claude
// Code's surface at all (see TokenCounts) and is never set.
func tokenCounts(totals map[string]int64) TokenCounts {
	var counts TokenCounts
	for _, m := range []struct {
		wireType string
		count    **int
	}{
		{tokenTypeInput, &counts.Input},
		{tokenTypeOutput, &counts.Output},
		{tokenTypeCacheRead, &counts.CacheRead},
		{tokenTypeCacheCreation, &counts.CacheCreation},
	} {
		if total, ok := totals[m.wireType]; ok {
			*m.count = Count(clampToInt(total))
		}
	}
	return counts
}

func isTokenType(s string) bool {
	switch s {
	case tokenTypeInput, tokenTypeOutput, tokenTypeCacheRead, tokenTypeCacheCreation:
		return true
	}
	return false
}

// extractToolCalls lists the session's tool calls from two disjoint sources.
//
// claude_code.tool_result is emitted when a tool completes, and only then, so
// it is the execution outcome: ToolCallEntry.Success means the tool ran and
// succeeded. claude_code.tool_decision is the permission decision, and supplies
// the calls that were rejected and therefore never ran — they produce no
// result. Because an accepted call's outcome is read from its result and a
// rejected call has none, the two sources cannot describe the same call and no
// de-duplication by tool_use_id is needed. An export captured mid-run has
// accepts whose results have not been written yet, and those calls are not
// listed.
//
// The reason below says so only when no call was read at all, because that is
// the only path it is built on: a profile/v1 present result carries no reason,
// so an export mixing completed results with pending accepts lists the results
// and has no field in which to say how many accepts it passed over. That is the
// same schema v1 gap as a skipped data point, tracked with it for 0.5.0, and
// docs/profiler-spec.md states it. A comment claiming the reason always says so
// would describe a channel the schema does not have.
func extractToolCalls(export scopedExport) ToolCallResult {
	var calls []timedEntry[ToolCallEntry]
	var c toolCallCounters

	for r := range export.logRecords() {
		// At most one defect is counted per record, name before outcome, so no
		// record is counted twice and every number in the reason is a record.
		switch eventName(r) {
		case otelToolResultLog:
			c.seen++
			name, ok := r.Attributes.String("tool_name")
			if !ok || name == "" {
				c.unnamedResults++
				continue
			}
			success, ok := r.Attributes.Bool("success")
			if !ok {
				// An outcome that was not read is not an outcome. Recording it
				// as a failure would invent a failed call.
				c.unreadableOutcomes++
				continue
			}
			calls = append(calls, toolCall(name, success, r))
		case otelToolDecisionLog:
			c.seen++
			decision, _ := r.Attributes.String("decision")
			switch decision {
			case "reject":
				name, ok := r.Attributes.String("tool_name")
				if !ok || name == "" {
					c.unnamedRejects++
					continue
				}
				calls = append(calls, toolCall(name, false, r))
			case "accept":
				c.accepts++
			default:
				c.undecided++
			}
		}
	}

	switch {
	case c.seen == 0:
		return UnknownToolCallResult("no " + otelToolResultLog + " or " + otelToolDecisionLog +
			" log events found in OTel export" + export.logsNotRead())
	case len(calls) == 0:
		return UnknownToolCallResult("no tool call outcomes in OTel export: " + c.reason() + export.logsNotRead())
	}
	return PresentToolCallResult(orderedEntries(calls), string(SourceOtel))
}

// toolCallCounters is what the walk observed, and the only thing the reason is
// built from — so a reason cannot state a number the walk did not count.
//
// The reason is composed from the counters rather than selected by a branch per
// combination: four independent defects need fifteen branches, and the ones
// that were missing were exactly the combinations nobody thought of. A clause
// list has one entry per counter and cannot go stale when a counter is added.
type toolCallCounters struct {
	seen               int
	unnamedResults     int
	unreadableOutcomes int
	accepts            int
	undecided          int
	unnamedRejects     int
}

func (c toolCallCounters) reason() string {
	clauses := make([]string, 0, 5)
	add := func(n int, noun, rest string) {
		if n > 0 {
			clauses = append(clauses, quantity(n, noun)+rest)
		}
	}
	add(c.unnamedResults, otelToolResultLog+" event", " carried no tool_name")
	add(c.unreadableOutcomes, otelToolResultLog+" event", " carried no readable success value")
	// "accepted …" rather than "… were accepts", so the clause reads the same
	// whether one accept or five are being reported.
	add(c.accepts, "accepted "+otelToolDecisionLog+" event", ", and only "+otelToolResultLog+
		" reports an outcome — the export may have been captured before those tools completed")
	add(c.undecided, otelToolDecisionLog+" event", " carried no recognised decision")
	add(c.unnamedRejects, otelToolDecisionLog+" event", " recorded a reject with no tool_name")
	return strings.Join(clauses, "; ")
}

// timedEntry keeps a profile entry beside the time it sorts by. An entry whose
// timestamp could not be read is kept, not dropped: the record was read, only
// its clock was not, and dropping it would make the list lie about how many
// things the session did.
//
// It is generic over the entry because the rule is the same for every list the
// profile carries in event order, and a second copy of it is a second place for
// the untimed case to be got wrong.
type timedEntry[T any] struct {
	entry T
	nanos int64
	timed bool
}

// recordTime is a log record's timestamp in the two forms an entry needs: the
// text the profile carries, and the integer the ordering compares. A record
// whose clock could not be read yields the empty string and false, which is an
// entry that sorts last rather than one that is dropped.
func recordTime(r otlpLogRecord) (string, int64, bool) {
	t, ok := nanoTime(r.TimeUnixNano)
	if !ok {
		return "", 0, false
	}
	return t.Format(time.RFC3339Nano), t.UnixNano(), true
}

func toolCall(name string, success bool, r otlpLogRecord) timedEntry[ToolCallEntry] {
	stamp, nanos, timed := recordTime(r)
	return timedEntry[ToolCallEntry]{
		entry: ToolCallEntry{Name: name, Timestamp: stamp, Success: success},
		nanos: nanos,
		timed: timed,
	}
}

// orderedEntries sorts entries by timestamp ascending, untimed entries last in
// file order. Neither ToolCallEntry.Timestamp nor ActivationEntry.Timestamp has
// omitempty, so an untimed entry serialises with an empty timestamp rather than
// borrowing a neighbour's.
func orderedEntries[T any](items []timedEntry[T]) []T {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].timed != items[j].timed {
			return items[i].timed
		}
		if !items[i].timed {
			return false
		}
		return items[i].nanos < items[j].nanos
	})
	entries := make([]T, 0, len(items))
	for _, it := range items {
		entries = append(entries, it.entry)
	}
	return entries
}

// extractActivations lists the skills this session activated.
//
// The source is claude_code.skill_activated, which the harness logs when a
// skill is invoked — through the Skill tool or a / command — and only then, so
// one record is one activation and no de-duplication is needed. skill.name also
// rides along on request-scoped signals, and those are deliberately *not* read
// here: that attribute marks a skill active for a request, so a skill used
// across five requests carries it five times, and one entry per record would
// report five activations the harness never logged. An export carrying the
// attribute and no event therefore reports unknown, and the reason names the
// event that was looked for.
//
// A name is required and a trigger is not. The name is what an activation is
// about, and an entry naming no skill is one nothing can be done with; the
// trigger is a detail the event does not always carry, and ActivationEntry
// leaves it out rather than guessing at it.
//
// The names themselves are reported exactly as the export spelled them. Claude
// Code redacts user-defined and third-party plugin skills to custom_skill on
// this event unless OTEL_LOG_TOOL_DETAILS=1, and un-redacting that would be
// inventing a name nobody recorded.
func extractActivations(export scopedExport) ActivationResult {
	var activations []timedEntry[ActivationEntry]
	var c activationCounters

	for r := range export.logRecords() {
		if eventName(r) != otelSkillActivatedLog {
			continue
		}
		c.seen++
		name, ok := r.Attributes.String("skill.name")
		if !ok || name == "" {
			c.unnamed++
			continue
		}
		trigger, _ := r.Attributes.String("invocation_trigger")
		stamp, nanos, timed := recordTime(r)
		activations = append(activations, timedEntry[ActivationEntry]{
			entry: ActivationEntry{SkillName: name, Timestamp: stamp, Trigger: trigger},
			nanos: nanos,
			timed: timed,
		})
	}

	// Every unknown reason carries what the provenance projection removed, so a
	// reason saying no activation was found cannot be read as saying the export
	// carries none — it may carry several, under another session or another
	// product's instrumentation scope.
	switch {
	case c.seen == 0:
		return UnknownActivationResult("no " + otelSkillActivatedLog + " log events found in OTel export" + export.logsNotRead())
	case len(activations) == 0:
		return UnknownActivationResult("no skill activations in OTel export: " + c.reason() + export.logsNotRead())
	}
	return PresentActivationResult(orderedEntries(activations), string(SourceOtel))
}

// activationCounters is what the walk observed over claude_code.skill_activated
// records, and the only thing the reason is built from — so a reason cannot
// state a number the walk did not count. Composed as a clause list for the same
// reason the token and tool-call reasons are: a second defect class adds an
// entry rather than reshaping a sentence.
type activationCounters struct {
	seen    int
	unnamed int
}

func (c activationCounters) reason() string {
	clauses := make([]string, 0, 1)
	if c.unnamed > 0 {
		clauses = append(clauses, quantity(c.unnamed, otelSkillActivatedLog+" event")+" carried no skill.name")
	}
	return strings.Join(clauses, "; ")
}

// extractTiming reports the span the API requests cover.
//
// It is scoped to claude_code.api_request rather than to every log record: a
// file of startup events would otherwise report a session span, and total_ms
// would mean something different than it did in 0.4.0. The cost is that the
// span excludes the prompt before the first request and any tool activity after
// the last one, which the spec says out loud. Per-request duration_ms and
// active_time.total measure different quantities and have no field in schema v1.
func extractTiming(export scopedExport) TimingResult {
	var first, last time.Time
	seen, have := 0, false

	for r := range export.logRecords() {
		if eventName(r) != otelAPIRequestLog {
			continue
		}
		seen++
		t, ok := nanoTime(r.TimeUnixNano)
		if !ok {
			continue
		}
		// The span runs from the earliest event to the latest, not from the
		// first record in the file to the last: an exporter is free to write
		// batches out of order, and a session must not end before it starts.
		if !have || t.Before(first) {
			first = t
		}
		if !have || t.After(last) {
			last = t
		}
		have = true
	}

	switch {
	case seen == 0:
		return UnknownTimingResult("no " + otelAPIRequestLog + " log events found in OTel export" + export.logsNotRead())
	case !have:
		return UnknownTimingResult("no readable " + otelAPIRequestLog +
			" log events in OTel export: none carried a parseable timeUnixNano" + export.logsNotRead())
	}
	// A single request is a zero-length span: a value that was read, not an
	// absence. The nanosecond digits stay out of the profile — these fields are
	// timestamps.
	return PresentTimingResult(TimingData{
		StartTime: first.Format(time.RFC3339Nano),
		EndTime:   last.Format(time.RFC3339Nano),
		TotalMs:   last.Sub(first).Milliseconds(),
	}, string(SourceOtel))
}
