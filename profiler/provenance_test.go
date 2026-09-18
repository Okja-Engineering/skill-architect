package profiler

import (
	"encoding/json"
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

	for _, sig := range []struct {
		name  string
		state MetricState
		value any
		nilly bool
		why   string
	}{
		{"tokens", profile.Tokens.State, profile.Tokens.Value, profile.Tokens.Value == nil, profile.Tokens.Reason},
		{"tool_calls", profile.ToolCalls.State, profile.ToolCalls.Value, profile.ToolCalls.Value == nil, profile.ToolCalls.Reason},
		{"timing", profile.Timing.State, profile.Timing.Value, profile.Timing.Value == nil, profile.Timing.Reason},
	} {
		if sig.state != MetricUnknown {
			data, _ := json.Marshal(sig.value)
			t.Errorf("%s state = %q with value %s for a session the export does not contain, want unknown",
				sig.name, sig.state, data)
		}
		if !sig.nilly {
			data, _ := json.Marshal(sig.value)
			t.Errorf("%s value = %s, want none", sig.name, data)
		}
		// The export does carry token metrics, tool results and api_requests.
		// A reason saying it carries none of them would be a second false
		// statement in place of the first.
		if !strings.Contains(sig.why, sessionAbsent) {
			t.Errorf("%s reason = %q, want it to name the session that was not found", sig.name, sig.why)
		}
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
