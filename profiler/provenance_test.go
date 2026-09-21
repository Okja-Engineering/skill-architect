package profiler

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// A profile names one session, and both documented capture routes append to one
// file by design: route (a) is a receiver on a loopback OTLP port, which
// anything on the machine may post to, and route (b) is a collector's file
// exporter. So one export legitimately carries several sessions, and whatever
// else on the machine was exporting. These tests hold the adapter to the
// invariant that makes a session-stamped profile mean anything:
//
//	a profile stamped with a session id contains only signal derived from
//	records carrying it, or it is not present.
//
// They are written against the invariant rather than against the filter, so a
// later repair is free to move where the filter lives.

const (
	// The two sessions in two_sessions.ndjson and in
	// two_sessions_one_metric.json. sessionA is the profiled one.
	sessionA = "22222222-2222-4222-8222-222222222222"
	sessionB = "33333333-3333-4333-8333-333333333333"

	// A session id that appears nowhere in any fixture. Asserting it must
	// produce no value at all — this is the half that catches a filter which
	// silently matches everything, because such a filter passes every other
	// case in this file.
	sessionAbsent = "44444444-4444-4444-8444-444444444444"
)

func profileOf(t *testing.T, fixtureName, sessionID string) Profile {
	t.Helper()
	adapter := ClaudeCodeAdapter{OtelExportFile: fixture(fixtureName)}
	profile, err := adapter.Capture(sessionID, CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

func toolCallNames(entries []ToolCallEntry) []string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}
	return names
}

// --- G2-01: the session the profile names is the session it reports ---

// Two sessions in one export, which is what the documented capture produces.
// Every number in the profile must be session A's. Before the repair the totals
// were both sessions' — 49200/9440 against a real 48000/9100 — and the span ran
// from A's first request to B's last, 99999000 ms of "session".
func TestProvenance_TwoSessionExportReportsOnlyTheProfiledSession(t *testing.T) {
	profile := profileOf(t, "two_sessions.ndjson", sessionA)

	if profile.Tokens.State != MetricPresent {
		t.Fatalf("tokens state = %q (%s), want present — session A's own points are readable",
			profile.Tokens.State, profile.Tokens.Reason)
	}
	assertTokenJSON(t, profile.Tokens.Value, `{"input":48000,"output":9100}`)

	if got, want := toolCallNames(profile.ToolCalls.Value), []string{"Read", "Bash"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("tool_calls = %v, want %v — GrepInAnotherSession belongs to session B", got, want)
	}

	if profile.Timing.State != MetricPresent {
		t.Fatalf("timing state = %q (%s), want present", profile.Timing.State, profile.Timing.Reason)
	}
	if got := profile.Timing.Value.TotalMs; got != 12000 {
		t.Errorf("timing total_ms = %d, want 12000 — session A ran 12 seconds; 99999000 is the span to session B's request", got)
	}
}

// The same export, profiled as session B. Asserting a different identity must
// give that identity's numbers, which is what proves the filter reads the
// argument rather than any fixed value.
func TestProvenance_TheSameExportProfiledAsTheOtherSession(t *testing.T) {
	profile := profileOf(t, "two_sessions.ndjson", sessionB)

	assertTokenJSON(t, profile.Tokens.Value, `{"input":1200,"output":340}`)
	if got, want := toolCallNames(profile.ToolCalls.Value), []string{"GrepInAnotherSession"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("tool_calls = %v, want %v", got, want)
	}
	if got := profile.Timing.Value.TotalMs; got != 0 {
		t.Errorf("timing total_ms = %d, want 0 — session B made one request", got)
	}
}

// Two sessions inside one metric's dataPoints array, which is what a collector
// flushing both at once writes — and the layout two_sessions.ndjson cannot
// reach, because it gives each session its own batch. The filter runs *within*
// the array here: whichever session is asserted, the profile holds that
// session's points and only those.
//
// The array's four points total 95000 input and 5000 output. Neither session
// holds that, so a filter that reads the array whole is loud rather than
// plausible — and a filter that refuses a metric because a foreign point sits
// beside a good one turns the profiled session's tokens into unknown.
func TestProvenance_OneMetricCarryingTwoSessionsContributesOnlyTheProfiledSession(t *testing.T) {
	for _, tc := range []struct {
		name    string
		session string
		want    string
	}{
		{"session A", sessionA, `{"input":5000,"output":700}`},
		{"session B", sessionB, `{"input":90000,"output":4300}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile := profileOf(t, "two_sessions_one_metric.json", tc.session)

			if profile.Tokens.State != MetricPresent {
				t.Fatalf("tokens state = %q (%s), want present — this session's own points sit in the array",
					profile.Tokens.State, profile.Tokens.Reason)
			}
			assertTokenJSON(t, profile.Tokens.Value, tc.want)
		})
	}
}

// The other half of the same array: a metric that arrived with data points and
// kept none of them is not a metric this session has, so it is dropped rather
// than kept as an empty sum.
//
// Keeping it produces a reason that states something false of the input — that
// the metric "carried no sum data points", of an export carrying four — which
// is the same shape of false statement as the number this repair removed, and
// the reason the drop rule exists. The true answer is that the export holds no
// token metric of *this* session's, beside a count of what it does hold.
func TestProvenance_AMetricEmptiedByTheFilterIsNotReportedAsCarryingNoDataPoints(t *testing.T) {
	profile := profileOf(t, "two_sessions_one_metric.json", sessionAbsent)

	if profile.Tokens.State != MetricUnknown || profile.Tokens.Value != nil {
		t.Fatalf("tokens state = %q with value %v, want unknown and no value — no point in the array is this session's",
			profile.Tokens.State, profile.Tokens.Value)
	}

	reason := profile.Tokens.Reason
	if !strings.Contains(reason, "no "+otelTokenUsageMetric+" metric found in OTel export") {
		t.Errorf("tokens reason = %q, want it to say the export holds no token metric of this session's", reason)
	}
	if strings.Contains(reason, "carried no sum data points") {
		t.Errorf("tokens reason = %q: the export carries four sum data points, so this reason is false of it", reason)
	}
	// And the reason accounts for what is in there, in the unit the reader can
	// count in their own export.
	if want := "4 data points not carrying session.id " + sessionAbsent; !strings.Contains(reason, want) {
		t.Errorf("tokens reason = %q, want it to carry %q", reason, want)
	}
}

// A session id that appears nowhere in the export must yield no value for any
// signal. This is the case that fails against a filter matching everything, and
// the reason must not claim the export lacks a signal it plainly carries.
func TestProvenance_ASessionAbsentFromTheExportYieldsNoValue(t *testing.T) {
	profile := profileOf(t, "two_sessions.ndjson", sessionAbsent)

	// The signals walked are derived rather than listed. Every signal this
	// adapter reads out of the export has to answer this way, and a
	// hand-written three stopped asserting that the moment a fourth was read —
	// which is what happened when skill_activation began coming out of the
	// export. otelBackedSignals is that set.
	signals := capturedSignals(profile)
	walked := 0
	for _, metric := range sortedMetrics(profile.Capability.Capabilities) {
		if !otelBackedSignals[metric] {
			continue
		}
		walked++
		sig := signals[metric]
		if sig.raw.State != MetricUnknown {
			data, _ := json.Marshal(profile)
			t.Errorf("%s state = %q for a session the export does not contain, want unknown\n%s",
				metric, sig.raw.State, data)
		}
		if sig.hasValue {
			data, _ := json.Marshal(profile)
			t.Errorf("%s carries a value, want none\n%s", metric, data)
		}
		// The export does carry token metrics, tool results and api_requests.
		// A reason saying it carries none of them would be a second false
		// statement in place of the first.
		if !strings.Contains(sig.raw.Reason, sessionAbsent) {
			t.Errorf("%s reason = %q, want it to name the session that was not found", metric, sig.raw.Reason)
		}
	}
	// A derivation that found nothing would make every assertion above
	// vacuously true, which is the failure mode a derived denominator has and a
	// written-down one does not. The two guards are not one guard twice:
	// comparing walked with len(otelBackedSignals) catches the capability
	// report falling behind that set, and is satisfied by *both* being empty —
	// so it cannot also be what notices that nothing was walked.
	if walked == 0 {
		t.Error("no export-backed signal was walked, so every assertion above passed by reading nothing")
	}
	if want := len(otelBackedSignals); walked != want {
		t.Errorf("walked %d export-backed signals, want %d — the capability report and otelBackedSignals disagree",
			walked, want)
	}

	if got := profile.Capability.Capabilities[MetricTokens]; got != SourceNone {
		t.Errorf("capability tokens = %q for a session the export does not contain, want none", got)
	}
}

// --session is required by the CLI. After the repair it decides what is read,
// so an empty one is not a session that matched nothing — it is no assertion at
// all, and a profile cannot be stamped with it. The adapter owns the contract,
// as it does for CaptureOpts.ExportFile, so a library caller is told too.
func TestProvenance_CaptureRefusesAnEmptySessionID(t *testing.T) {
	adapter := ClaudeCodeAdapter{OtelExportFile: fixture("two_sessions.ndjson")}
	_, err := adapter.Capture("", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err == nil {
		t.Fatal("Capture accepted an empty session id: the profile would be stamped with no identity and report every session in the export")
	}
	if !strings.Contains(err.Error(), "--session") {
		t.Errorf("error = %q, want it to name the flag that supplies the identity", err)
	}
}

// --- G2-02: a record is this harness's only if it says so ---

// One export, one session id on every record, and three of those records are
// not Claude Code's: two arrived under an instrumentation scope naming another
// product, and one names an unqualified event that only re-qualification in the
// reader turned into a claude_code one.
func TestProvenance_ForeignScopeAndUnqualifiedEventsAreNotAttributed(t *testing.T) {
	profile := profileOf(t, "foreign_scope.json", sessionA)

	// 7000 of the 8000 input tokens in this file were recorded by
	// some.other.product.
	assertTokenJSON(t, profile.Tokens.Value, `{"input":1000}`)

	// ViaEventName carries no body and is named by its event.name attribute in
	// the short form, which is the documented spelling: it is read. BareBodyTool
	// names an unqualified event in its body, where the qualified name is what
	// is emitted, so it is not this harness's record.
	if got, want := toolCallNames(profile.ToolCalls.Value), []string{"Read", "ViaEventName"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("tool_calls = %v, want %v — SomeOtherProductTool is another product's and BareBodyTool names no claude_code event",
			got, want)
	}

	// The foreign api_request sits 27.8 hours after this session's one.
	if got := profile.Timing.Value.TotalMs; got != 0 {
		t.Errorf("timing total_ms = %d, want 0 — this session made one api_request", got)
	}
}

// The counterpart to the test above: the scope check must not cost a record
// whose scope says nothing. Most fixtures carry no scope at all, and a
// collector or a hand-rolled receiver is free to drop it; refusing those would
// trade a wrong number for no number on every pipeline that does.
func TestProvenance_AnAbsentScopeIsNotAForeignScope(t *testing.T) {
	profile := profileOf(t, "cumulative.ndjson", fixtureSession)

	if profile.Tokens.State != MetricPresent {
		t.Fatalf("tokens state = %q (%s), want present — no record in this export names a foreign scope",
			profile.Tokens.State, profile.Tokens.Reason)
	}
}

// --- the same two rules, over the signal this release added ---
//
// skill_activation is read out of the export like tokens, tool calls and
// timing, so it answers to the same provenance. These cases are the ones that
// would pass against an extractor handed the file instead of the projection —
// which is the defect 0.4.3 closed, and the one most likely to reappear in a
// new extractor.

func activationNames(entries []ActivationEntry) []string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.SkillName)
	}
	return names
}

// One export, two sessions' activations, and a profile that names one of them.
// Asserting each identity in turn is what proves the filter reads its argument:
// a reader that returned every activation passes neither case, and one that
// returned the first record's passes neither.
func TestProvenance_ActivationsAreTheProfiledSessionsOnly(t *testing.T) {
	for _, tc := range []struct {
		name    string
		session string
		want    []string
	}{
		{"session A", sessionA, []string{"session-a-skill"}},
		{"session B", sessionB, []string{"session-b-first", "session-b-second"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile := profileOf(t, "two_sessions_activation.json", tc.session)

			if profile.SkillActivation.State != MetricPresent {
				t.Fatalf("skill_activation state = %q (%s), want present — this session activated a skill",
					profile.SkillActivation.State, profile.SkillActivation.Reason)
			}
			got := activationNames(profile.SkillActivation.Value)
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Errorf("skill_activation = %v, want %v — the other session's activations are not this profile's",
					got, tc.want)
			}
			// The export also carries an activation on a record that names no
			// session at all. An activation the adapter cannot attribute
			// belongs to no profile: reporting it under whichever session was
			// asked for is a guess, and the answer to a guess is to say nothing.
			for _, name := range got {
				if name == "unattributable-skill" {
					t.Errorf("skill_activation = %v, want the record carrying no session.id left out — "+
						"it cannot be attributed to any session", got)
				}
			}
		})
	}
}

// A session that activated no skill reports no activation, and the reason
// accounts for the records the projection removed rather than claiming the
// export carries no activation event at all — it carries four.
func TestProvenance_ASessionWithNoActivationIsNotGivenAnothersOrToldTheExportHasNone(t *testing.T) {
	profile := profileOf(t, "two_sessions_activation.json", sessionAbsent)

	if profile.SkillActivation.State != MetricUnknown || profile.SkillActivation.Value != nil {
		t.Fatalf("skill_activation state = %q with value %+v, want unknown and no value",
			profile.SkillActivation.State, profile.SkillActivation.Value)
	}
	reason := profile.SkillActivation.Reason
	if want := "4 log records not carrying session.id " + sessionAbsent; !strings.Contains(reason, want) {
		t.Errorf("skill_activation reason = %q, want it to carry %q — the export holds four activation records,"+
			" none of them this session's", reason, want)
	}
}

// A record naming another product's instrumentation scope is that product's,
// whatever session id it carries — so its activation is not this profile's even
// though every other test of the identity would pass it.
func TestProvenance_AForeignScopesActivationIsNotAttributed(t *testing.T) {
	profile := profileOf(t, "foreign_scope_activation.json", fixtureSession)

	if profile.SkillActivation.State != MetricPresent {
		t.Fatalf("skill_activation state = %q (%s), want present — this harness's scope logged one activation",
			profile.SkillActivation.State, profile.SkillActivation.Reason)
	}
	if got, want := activationNames(profile.SkillActivation.Value), []string{"my-skill"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("skill_activation = %v, want %v — some.other.product's activation carries this session's id and is still not ours",
			got, want)
	}
}

// The other half of it: when the foreign scope's record is the only activation
// in the file, the signal is unknown and says what it passed over. Without this
// case, a scope check could be deleted and the case above would still pass on
// the surviving record.
func TestProvenance_AnExportWhoseOnlyActivationIsForeignYieldsNone(t *testing.T) {
	profile := profileOf(t, "foreign_scope_activation_only.json", fixtureSession)

	if profile.SkillActivation.State != MetricUnknown || profile.SkillActivation.Value != nil {
		t.Fatalf("skill_activation state = %q with value %+v, want unknown — the one activation here is another product's",
			profile.SkillActivation.State, profile.SkillActivation.Value)
	}
	reason := profile.SkillActivation.Reason
	if want := "1 log record recorded by an instrumentation scope that is not claude_code's"; !strings.Contains(reason, want) {
		t.Errorf("skill_activation reason = %q, want it to carry %q", reason, want)
	}
	if got := profile.Capability.Capabilities[MetricSkillActivation]; got != SourceNone {
		t.Errorf("capability skill_activation = %q, want none", got)
	}
}

// --- probe is unscoped by session, and by nothing else ---

// probe and capture are asked different questions, so they are allowed to give
// different answers about one file, and this is the divergence that follows:
// probe is not told a session, so it answers for the export as a whole, while a
// capture answers for the session it names.
//
// The pair is asserted together because the divergence is only honest if the
// profile's own capability block comes from its own scoped read. An export
// holding two sessions probes `otel` and a capture of a session it does not
// hold reports `none` — a disagreement between two questions, never inside one
// profile.
func TestProvenance_ProbeAnswersForTheExportAndACaptureForItsSession(t *testing.T) {
	adapter := ClaudeCodeAdapter{OtelExportFile: fixture("two_sessions.ndjson")}

	report := adapter.Probe()
	for _, metric := range []MetricName{MetricTokens, MetricToolCalls, MetricTiming} {
		if got := report.Capabilities[metric]; got != SourceOtel {
			t.Errorf("probe capability %s = %q, want %q — probe was told no session, so it answers for the export",
				metric, got, SourceOtel)
		}
	}

	profile, err := adapter.Capture(sessionAbsent, CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}
	for _, metric := range []MetricName{MetricTokens, MetricToolCalls, MetricTiming} {
		if got := profile.Capability.Capabilities[metric]; got != SourceNone {
			t.Errorf("profile capability %s = %q, want %q — this profile's capability block is its own scoped read, not probe's",
				metric, got, SourceNone)
		}
	}
}

// The session test is the only one probe drops. A record naming another
// product's scope is that product's whatever question is being asked, so an
// export whose only token metric was recorded by some.other.product can yield
// nothing — and probe must say so rather than advertising a capability the
// capture could only ever refuse.
func TestProvenance_ProbeStillRefusesAForeignScope(t *testing.T) {
	const foreign = `{"resourceMetrics":[{"scopeMetrics":[{"scope":{"name":"some.other.product"},` +
		`"metrics":[{"name":"claude_code.token.usage","sum":{"aggregationTemporality":1,"dataPoints":[{` +
		`"attributes":[{"key":"session.id","value":{"stringValue":"` + fixtureSession + `"}},` +
		`{"key":"type","value":{"stringValue":"input"}}],"timeUnixNano":"1789332596272000000","asDouble":100}]}}]}]}]}`

	adapter := ClaudeCodeAdapter{OtelExportFile: writeExport(t, foreign)}

	if got := adapter.Probe().Capabilities[MetricTokens]; got != SourceNone {
		t.Errorf("probe capability tokens = %q, want %q — the one token metric in this export is another product's",
			got, SourceNone)
	}
}

// anySession, named and greppable, is what drops the session test, and probe is
// the only thing that sets it. Asserted on the projection rather than through a
// profile, because this is where the two provenances differ: one keeps every
// session's points, the other keeps one session's.
//
// The capture side is the load-bearing half. A capture provenance that dropped
// the session test would report every session in the export, which is the
// defect this whole repair exists for.
func TestProvenance_OnlyProbeDropsTheSessionTest(t *testing.T) {
	var adapter ClaudeCodeAdapter

	if !adapter.probeProvenance().anySession {
		t.Error("probeProvenance does not drop the session test: probe is asked what the export can yield, without a session")
	}
	if adapter.provenanceFor(sessionA).anySession {
		t.Fatal("a capture provenance drops the session test: the profile would report every session in the export")
	}

	export := readFixtureExport(t, "two_sessions_one_metric.json")
	if got, want := keptSessions(export.scopedTo(adapter.probeProvenance())), []string{sessionB, sessionA, sessionB, sessionA}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("probe's projection keeps sessions %v, want %v — every point in the array, whoever's it is", got, want)
	}
	if got, want := keptSessions(export.scopedTo(adapter.provenanceFor(sessionA))), []string{sessionA, sessionA}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("a capture of session A keeps sessions %v, want %v", got, want)
	}
}

// readFixtureExport is a fixture read but not projected, for the two tests that
// assert what a projection does to it.
func readFixtureExport(t *testing.T, name string) otlpExport {
	t.Helper()
	f, err := os.Open(fixture(name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	export, err := readOTLP(f)
	if err != nil {
		t.Fatal(err)
	}
	return export
}

// keptSessions is the session id of every token data point a projection kept,
// in walk order.
func keptSessions(scoped scopedExport) []string {
	ids := make([]string, 0, 4)
	for m := range scoped.metrics(otelTokenUsageMetric) {
		if m.Sum == nil {
			continue
		}
		for _, dp := range m.Sum.DataPoints {
			id, _ := dp.Attributes.String(otelSessionAttr)
			ids = append(ids, id)
		}
	}
	return ids
}
