package profiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// The comparison exists to answer one question: what changed about the skill
// between two runs. Everything in this file is a guard against it answering a
// different question — what changed about the *reader* — and presenting the
// answer as if it were about the skill.

const (
	testAdapterVersion  = "0.5.0"
	otherAdapterVersion = "0.4.3"
)

// presentProfile is a profile in which every signal was read, so that a test
// about a refusal cannot pass merely because there was nothing to compare.
func presentProfile(adapterVer, source string) Profile {
	p := Profile{
		Schema:       ProfileSchema,
		ProfiledAt:   "2026-09-20T10:00:00Z",
		Harness:      "claude_code",
		SessionID:    "session-1",
		SnapshotHash: "sha-baseline",
		SkillDir:     "/skills/my-skill",
		Capability: CapabilityReport{
			Harness:      "claude_code",
			AdapterVer:   adapterVer,
			ProbedAt:     "2026-09-20T10:00:00Z",
			Capabilities: map[MetricName]MetricSource{},
		},
		Tokens: PresentTokenResult(TokenCounts{
			Input:         Count(100),
			Output:        Count(20),
			CacheRead:     Count(5),
			CacheCreation: Count(1),
		}, source),
		ToolCalls: PresentToolCallResult([]ToolCallEntry{
			{Name: "Read", Timestamp: "2026-09-20T10:00:01Z", Success: true},
			{Name: "Bash", Timestamp: "2026-09-20T10:00:02Z", Success: false, ErrorType: "timeout"},
		}, source),
		SkillActivation: PresentActivationResult([]ActivationEntry{
			{SkillName: "skill-audit", Timestamp: "2026-09-20T10:00:00Z", Trigger: "skill_tool"},
		}, source),
		Timing: PresentTimingResult(TimingData{
			StartTime: "2026-09-20T10:00:00Z",
			EndTime:   "2026-09-20T10:00:10Z",
			TotalMs:   10000,
		}, source),
		Attribution: PresentAttributionResult(AttributionData{
			Attributions: []Attribution{{Target: "call-1", SkillName: "skill-audit", Confidence: "observed"}},
		}, source),
	}
	est := PresentEstimatedTokensResult(EstimatedTokens{Total: 4000}, string(SourceHooksEstimated))
	p.EstimatedContextTokens = &est
	return p
}

// comparablePair is the control: two profiles that differ only in what the
// session did. Every refusal below is asserted against it, because a comparator
// that refused everything would satisfy each refusal test on its own.
func comparablePair() (baseline, candidate Profile) {
	baseline = presentProfile(testAdapterVersion, string(SourceOtel))
	candidate = presentProfile(testAdapterVersion, string(SourceOtel))
	candidate.SessionID = "session-2"
	candidate.Tokens.Value.Input = Count(60)
	candidate.Timing.Value.TotalMs = 6000
	return baseline, candidate
}

func metricNames(m map[MetricName]MetricComparison) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, string(k))
	}
	sort.Strings(names)
	return names
}

// --- The adapter version refusal: the reason this slice exists ---

// A stored 0.4.x profile compared against a fresh 0.5.0 one reports four
// releases of reader fixes as the skill's regression. The adapter version is
// what makes that detectable, and this is its first consumer.
func TestCompareRefusesAPairReadByDifferentAdapterVersions(t *testing.T) {
	baseline, candidate := comparablePair()
	baseline.Capability.AdapterVer = otherAdapterVersion

	report := CompareProfiles(baseline, candidate)

	if report.Comparable {
		t.Error("a comparison across adapter versions reported itself comparable: the difference may be the reader, not the skill")
	}
	if report.Refusal == "" {
		t.Fatal("the report carries no refusal, so nothing tells a reader why it is not comparable")
	}
	for _, want := range []string{otherAdapterVersion, testAdapterVersion} {
		if !strings.Contains(report.Refusal, want) {
			t.Errorf("the refusal %q does not name version %q — a reader cannot tell which two readers disagreed", report.Refusal, want)
		}
	}

	// The refusal has to reach every metric. A report that says "not
	// comparable" at the top and carries a delta underneath still hands the
	// fabricated number to anything that reads one metric out of it.
	if len(report.Metrics) == 0 {
		t.Fatal("the report compared no metric at all, so this proves nothing about the refusal reaching them")
	}
	for name, mc := range report.Metrics {
		if mc.Comparable {
			t.Errorf("%s reported comparable under a refused pair", name)
		}
		if mc.Delta != nil {
			t.Errorf("%s carries a delta %v under a refused pair — this is the number that reads as a regression", name, mc.Delta)
		}
		if mc.Baseline != nil || mc.Candidate != nil {
			t.Errorf("%s carries values under a refused pair: baseline %v candidate %v", name, mc.Baseline, mc.Candidate)
		}
		if !strings.Contains(mc.Reason, otherAdapterVersion) || !strings.Contains(mc.Reason, testAdapterVersion) {
			t.Errorf("%s reason = %q, want it to name both adapter versions", name, mc.Reason)
		}
	}

	// The control. Without it, a comparator that refused every pair would
	// pass everything above.
	if control := CompareProfiles(comparablePair()); !control.Comparable {
		t.Fatal("the same pair at one adapter version is not comparable either — the refusal above proves nothing")
	} else if control.Refusal != "" {
		t.Errorf("a pair at one adapter version carries a refusal %q", control.Refusal)
	}
}

// A profile naming no adapter version cannot show it was read the same way as
// the other side, and "" == "" is agreement between two unknowns.
func TestCompareRefusesAPairThatNamesNoAdapterVersion(t *testing.T) {
	baseline, candidate := comparablePair()
	baseline.Capability.AdapterVer = ""
	candidate.Capability.AdapterVer = ""

	report := CompareProfiles(baseline, candidate)

	if report.Comparable {
		t.Error("two profiles naming no adapter version compared as if they agreed")
	}
	if !strings.Contains(report.Refusal, "adapter_version") {
		t.Errorf("refusal = %q, want it to name the field that is missing", report.Refusal)
	}
}

// Both versions are part of each side's identity in the report, not only of the
// refusal string: a reader that stores comparisons must be able to tell later
// which readers produced them.
func TestTheReportNamesEachSidesAdapterVersion(t *testing.T) {
	baseline, candidate := comparablePair()
	baseline.Capability.AdapterVer = otherAdapterVersion

	report := CompareProfiles(baseline, candidate)

	if report.Baseline.AdapterVersion != otherAdapterVersion {
		t.Errorf("baseline ref adapter_version = %q, want %q", report.Baseline.AdapterVersion, otherAdapterVersion)
	}
	if report.Candidate.AdapterVersion != testAdapterVersion {
		t.Errorf("candidate ref adapter_version = %q, want %q", report.Candidate.AdapterVersion, testAdapterVersion)
	}
}

// --- The source refusal, and both sources on every comparison ---

func TestCompareRefusesADeltaAcrossSources(t *testing.T) {
	baseline, candidate := comparablePair()
	candidate.Tokens.Source = string(SourceSQLite)

	report := CompareProfiles(baseline, candidate)

	tokens := report.Metrics[MetricTokens]
	if tokens.Comparable {
		t.Error("tokens compared across two sources: a cross-tier subtraction is not a delta")
	}
	if tokens.Delta != nil {
		t.Errorf("a cross-source comparison carries a delta %v", tokens.Delta)
	}
	for _, want := range []string{string(SourceOtel), string(SourceSQLite)} {
		if !strings.Contains(tokens.Reason, want) {
			t.Errorf("reason = %q, want it to name source %q", tokens.Reason, want)
		}
	}
	if tokens.BaselineSource != string(SourceOtel) || tokens.CandidateSource != string(SourceSQLite) {
		t.Errorf("sources = %q/%q, want %q/%q", tokens.BaselineSource, tokens.CandidateSource, SourceOtel, SourceSQLite)
	}

	// One refused metric is not a refused pair: the signals that do agree on a
	// source are still comparable, and the report still says so.
	if !report.Comparable {
		t.Error("one cross-source signal made the whole report incomparable")
	}
	if timing := report.Metrics[MetricTiming]; !timing.Comparable {
		t.Errorf("timing, which agrees on its source, was not compared: %q", timing.Reason)
	}
}

// Every MetricComparison carries both sides' sources — always, including when
// the metric was not compared at all. A signal nobody read has the source
// "none", which is the vocabulary's own word for it: an empty string is not a
// member of the source vocabulary, and a reader walking it would be wrong
// rather than ignorant.
func TestEveryMetricComparisonCarriesBothSources(t *testing.T) {
	present := presentProfile(testAdapterVersion, string(SourceOtel))
	unread := presentProfile(testAdapterVersion, string(SourceOtel))
	unread.Tokens = UnknownTokenResult("no OTel export configured")
	unread.ToolCalls = UnknownToolCallResult("no OTel export configured")
	unread.SkillActivation = UnknownActivationResult("no OTel export configured")
	unread.Timing = UnknownTimingResult("no OTel export configured")
	unread.Attribution = UnknownAttributionResult("this harness does not attribute outputs to skills")
	unread.EstimatedContextTokens = nil

	for _, tc := range []struct {
		name                string
		baseline, candidate Profile
		wantBase, wantCand  string
	}{
		{"both read", present, present, string(SourceOtel), string(SourceOtel)},
		{"the candidate read nothing", present, unread, string(SourceOtel), string(SourceNone)},
		{"the baseline read nothing", unread, present, string(SourceNone), string(SourceOtel)},
		{"neither read anything", unread, unread, string(SourceNone), string(SourceNone)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := CompareProfiles(tc.baseline, tc.candidate)
			if len(report.Metrics) == 0 {
				t.Fatal("the report compared no metric, so this walk asserts nothing")
			}
			for name, mc := range report.Metrics {
				// The estimate is carried by a different source even when it
				// was read, so it is checked for being non-empty only.
				if name == MetricEstimatedContextTokens {
					if mc.BaselineSource == "" || mc.CandidateSource == "" {
						t.Errorf("%s carries an empty source: %q/%q", name, mc.BaselineSource, mc.CandidateSource)
					}
					continue
				}
				if mc.BaselineSource != tc.wantBase {
					t.Errorf("%s baseline_source = %q, want %q", name, mc.BaselineSource, tc.wantBase)
				}
				if mc.CandidateSource != tc.wantCand {
					t.Errorf("%s candidate_source = %q, want %q", name, mc.CandidateSource, tc.wantCand)
				}
			}
		})
	}
}

// The sources are on the wire too: omitempty would drop the key for the side
// that read nothing, and "carried on every comparison" would stop being true of
// the JSON a consumer actually reads.
func TestBothSourcesSurviveIntoTheJSON(t *testing.T) {
	baseline, candidate := comparablePair()
	candidate.Tokens = UnknownTokenResult("no OTel export configured")

	data, err := json.Marshal(CompareProfiles(baseline, candidate))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var doc struct {
		Metrics map[string]map[string]json.RawMessage `json:"metrics"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(doc.Metrics) == 0 {
		t.Fatal("the marshalled report carries no metrics, so this proves nothing")
	}
	for name, fields := range doc.Metrics {
		for _, key := range []string{"baseline_source", "candidate_source"} {
			if _, ok := fields[key]; !ok {
				t.Errorf("metric %q has no %q key in the JSON", name, key)
			}
		}
	}
}

// --- The comparison covers the profile it reads ---

// A signal the profile carries and the comparison does not is a signal that
// silently stops being compared. The set is derived from the profile rather
// than written down here, so a seventh signal fails this rather than passing
// unnoticed.
func TestTheComparisonCoversEverySignalTheProfileCarries(t *testing.T) {
	report := CompareProfiles(comparablePair())

	signals := make([]string, 0)
	for name := range (Profile{}).SignalStates() {
		signals = append(signals, string(name))
	}
	sort.Strings(signals)

	// The non-vacuity guard, and it is a `> 0` check rather than an equality
	// against another count of the same thing: two derivations that both went
	// empty would agree with each other and assert nothing.
	if len(signals) == 0 {
		t.Fatal("the profile derived no signals, so this check reads nothing")
	}
	if got := metricNames(report.Metrics); !reflect.DeepEqual(got, signals) {
		t.Errorf("the comparison covers %v, the profile carries %v", got, signals)
	}
}

// --- Token counts are pointers, and a delta is only ever over what both read ---

// A count one side never read has no delta. Subtracting a count from an absent
// one reports a number nobody measured, which is what the pointer exists to
// prevent everywhere else in the profile.
func TestATokenCountOnlyOneSideReadHasNoDelta(t *testing.T) {
	baseline, candidate := comparablePair()
	candidate.Tokens.Value.Output = nil

	tokens := CompareProfiles(baseline, candidate).Metrics[MetricTokens]
	if !tokens.Comparable {
		t.Fatalf("tokens not compared at all: %q — the counts both sides read are still comparable", tokens.Reason)
	}
	delta, ok := tokens.Delta.(TokenCounts)
	if !ok {
		t.Fatalf("token delta is %T, want TokenCounts", tokens.Delta)
	}
	if delta.Output != nil {
		t.Errorf("output delta = %d, want no delta at all: the candidate never read an output count", *delta.Output)
	}
	if delta.Input == nil || *delta.Input != -40 {
		t.Errorf("input delta = %v, want -40", delta.Input)
	}
}

// When the two sides share no count, there is nothing to subtract and the
// metric is not comparable — rather than a delta object with every key absent,
// which reads as "no change".
func TestTokensAreNotComparableWhenTheSidesShareNoCount(t *testing.T) {
	baseline, candidate := comparablePair()
	baseline.Tokens = PresentTokenResult(TokenCounts{Input: Count(10)}, string(SourceOtel))
	candidate.Tokens = PresentTokenResult(TokenCounts{CacheRead: Count(10)}, string(SourceOtel))

	tokens := CompareProfiles(baseline, candidate).Metrics[MetricTokens]
	if tokens.Comparable {
		t.Errorf("two profiles sharing no token count compared anyway, delta %v", tokens.Delta)
	}
	if tokens.Delta != nil {
		t.Errorf("delta = %v, want none", tokens.Delta)
	}
	if !strings.Contains(tokens.Reason, "no token count") {
		t.Errorf("reason = %q, want it to say the two share no count", tokens.Reason)
	}
}

func TestATokenCountReadAsZeroIsStillCompared(t *testing.T) {
	baseline, candidate := comparablePair()
	baseline.Tokens = PresentTokenResult(TokenCounts{Input: Count(0)}, string(SourceOtel))
	candidate.Tokens = PresentTokenResult(TokenCounts{Input: Count(7)}, string(SourceOtel))

	tokens := CompareProfiles(baseline, candidate).Metrics[MetricTokens]
	delta, ok := tokens.Delta.(TokenCounts)
	if !ok {
		t.Fatalf("token delta is %T, want TokenCounts", tokens.Delta)
	}
	if delta.Input == nil || *delta.Input != 7 {
		t.Errorf("input delta = %v, want 7 — a measured zero is a measurement", delta.Input)
	}
}

// --- Tool calls ---

// An entry carrying an aggregate count is that many calls. Counting entries
// instead under-reports every delta-aggregated source by however much it
// aggregated.
func TestAToolCallEntryCarryingAnAggregateCountsAsThatMany(t *testing.T) {
	baseline, candidate := comparablePair()
	baseline.ToolCalls = PresentToolCallResult([]ToolCallEntry{
		{Name: "Read", Timestamp: "2026-09-20T10:00:01Z", Success: true, Count: 4},
		{Name: "Bash", Timestamp: "2026-09-20T10:00:02Z", Success: false},
	}, string(SourceOtel))
	candidate.ToolCalls = PresentToolCallResult([]ToolCallEntry{
		{Name: "Read", Timestamp: "2026-09-20T10:00:01Z", Success: true},
	}, string(SourceOtel))

	calls := CompareProfiles(baseline, candidate).Metrics[MetricToolCalls]
	if !calls.Comparable {
		t.Fatalf("tool calls not compared: %q", calls.Reason)
	}
	base, ok := calls.Baseline.(map[string]any)
	if !ok {
		t.Fatalf("baseline is %T, want map[string]any", calls.Baseline)
	}
	if got := base["count"]; got != 5 {
		t.Errorf("baseline count = %v, want 5 — four aggregated calls and one more", got)
	}
	if got := base["successes"]; got != 4 {
		t.Errorf("baseline successes = %v, want 4", got)
	}
}

// The success-rate delta is negative whenever the candidate did worse, which is
// the direction a naive rounding gets wrong.
func TestASuccessRateDeltaRoundsAwayFromZeroInBothDirections(t *testing.T) {
	failing := func(successes, failures int) ToolCallResult {
		var entries []ToolCallEntry
		for i := 0; i < successes; i++ {
			entries = append(entries, ToolCallEntry{Name: "Read", Success: true})
		}
		for i := 0; i < failures; i++ {
			entries = append(entries, ToolCallEntry{Name: "Bash", Success: false})
		}
		return PresentToolCallResult(entries, string(SourceOtel))
	}
	baseline, candidate := comparablePair()
	baseline.ToolCalls = failing(7, 1) // 0.88 after rounding
	candidate.ToolCalls = failing(1, 7)

	calls := CompareProfiles(baseline, candidate).Metrics[MetricToolCalls]
	delta, ok := calls.Delta.(map[string]any)
	if !ok {
		t.Fatalf("delta is %T, want map[string]any", calls.Delta)
	}
	if got := delta["success_rate"]; got != -0.75 {
		t.Errorf("success_rate delta = %v, want -0.75", got)
	}
}

// The profile format says a list-valued result with no entries is never
// present (see ToolCallResult): a session that called no tool is reported
// `unknown` with a reason, not `present` with nothing in it. So an empty list
// is the same absence as a missing one, however it was spelled in the file.
func TestAListSignalMarkedPresentWithNoEntriesIsRefusedHoweverItIsSpelled(t *testing.T) {
	for _, tc := range []struct {
		name  string
		apply func(*Profile)
	}{
		{"a nil tool-call list", func(p *Profile) {
			p.ToolCalls = PresentToolCallResult(nil, string(SourceOtel))
		}},
		{"an empty tool-call list, as a JSON [] decodes", func(p *Profile) {
			p.ToolCalls = PresentToolCallResult([]ToolCallEntry{}, string(SourceOtel))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			baseline, candidate := comparablePair()
			tc.apply(&baseline)
			tc.apply(&candidate)

			calls := CompareProfiles(baseline, candidate).Metrics[MetricToolCalls]
			if calls.Comparable {
				t.Errorf("a present result carrying no entry was compared, delta %v", calls.Delta)
			}
			if !strings.Contains(calls.Reason, "no value") {
				t.Errorf("reason = %q, want it to name the missing value", calls.Reason)
			}
		})
	}
}

// The one number in the report that divides. The comparator cannot reach a zero
// denominator — the check above is what stops it — so the guard is asserted
// where it lives.
func TestASuccessRateOverNoCallsIsZeroAndNotANaN(t *testing.T) {
	if got := successRate(0, 0); got != 0 {
		t.Errorf("successRate(0, 0) = %v, want 0", got)
	}
}

// --- Skill activation is a set of names, not a count ---

func TestSkillActivationIsComparedByNameAndNotByCountAlone(t *testing.T) {
	baseline, candidate := comparablePair()
	baseline.SkillActivation = PresentActivationResult([]ActivationEntry{
		{SkillName: "skill-audit"}, {SkillName: "skill-rewrite"},
	}, string(SourceOtel))
	candidate.SkillActivation = PresentActivationResult([]ActivationEntry{
		{SkillName: "skill-audit"}, {SkillName: "skill-gate"},
	}, string(SourceOtel))

	activation := CompareProfiles(baseline, candidate).Metrics[MetricSkillActivation]
	if !activation.Comparable {
		t.Fatalf("activation not compared: %q", activation.Reason)
	}
	delta, ok := activation.Delta.(map[string]any)
	if !ok {
		t.Fatalf("delta is %T, want map[string]any", activation.Delta)
	}
	if got := delta["count"]; got != 0 {
		t.Errorf("count delta = %v, want 0 — the same number of activations", got)
	}
	if got, want := delta["only_in_baseline"], []string{"skill-rewrite"}; !reflect.DeepEqual(got, want) {
		t.Errorf("only_in_baseline = %v, want %v — a bare count cannot say which skill stopped firing", got, want)
	}
	if got, want := delta["only_in_candidate"], []string{"skill-gate"}; !reflect.DeepEqual(got, want) {
		t.Errorf("only_in_candidate = %v, want %v", got, want)
	}
}

// --- The estimate is ordinal, and may be absent entirely ---

func TestTheEstimateDeltaCarriesItsOrdinalNote(t *testing.T) {
	baseline, candidate := comparablePair()
	candidate.EstimatedContextTokens.Value.Total = 5000

	est := CompareProfiles(baseline, candidate).Metrics[MetricEstimatedContextTokens]
	if !est.Comparable {
		t.Fatalf("the estimate was not compared: %q", est.Reason)
	}
	if !strings.Contains(est.Note, "not billed") {
		t.Errorf("note = %q, want it to say the estimate is not a billed count", est.Note)
	}
	delta, ok := est.Delta.(EstimatedTokens)
	if !ok {
		t.Fatalf("delta is %T, want EstimatedTokens", est.Delta)
	}
	if delta.Total != 1000 {
		t.Errorf("estimate delta = %d, want 1000", delta.Total)
	}
}

// A profile that estimated nothing has no key at all, and comparing one must
// not dereference the absent result.
func TestAProfileThatEstimatedNothingIsComparedWithoutDereferencingIt(t *testing.T) {
	baseline, candidate := comparablePair()
	baseline.EstimatedContextTokens = nil
	candidate.EstimatedContextTokens = nil

	report := CompareProfiles(baseline, candidate)
	est := report.Metrics[MetricEstimatedContextTokens]
	if est.Comparable {
		t.Error("an estimate nobody made was compared")
	}
	if est.BaselineSource != string(SourceNone) || est.CandidateSource != string(SourceNone) {
		t.Errorf("sources = %q/%q, want none/none", est.BaselineSource, est.CandidateSource)
	}
	if !report.Comparable {
		t.Error("the rest of the profile is still comparable")
	}
}

// --- A profile that read nothing ---

func TestAPairThatReadNothingIsNotComparableAndDoesNotPanic(t *testing.T) {
	empty := Profile{Schema: ProfileSchema}
	empty.Capability.AdapterVer = testAdapterVersion
	empty.Tokens = UnknownTokenResult("no OTel export configured")
	empty.ToolCalls = UnknownToolCallResult("no OTel export configured")
	empty.SkillActivation = UnknownActivationResult("no OTel export configured")
	empty.Timing = UnknownTimingResult("no OTel export configured")
	empty.Attribution = UnknownAttributionResult("no attribution source")

	report := CompareProfiles(empty, empty)
	if report.Comparable {
		t.Error("two profiles that read nothing compared as if they had")
	}
	for name, mc := range report.Metrics {
		if mc.Delta != nil {
			t.Errorf("%s carries a delta over two profiles that read nothing", name)
		}
	}
}

// A result marked present with no value behind it would be dereferenced by the
// comparator. It is refused instead of trusted, wherever it came from.
func TestAResultMarkedPresentWithNoValueIsRefusedNotDereferenced(t *testing.T) {
	baseline, candidate := comparablePair()
	baseline.Timing.Value = nil // still says "present"

	timing := CompareProfiles(baseline, candidate).Metrics[MetricTiming]
	if timing.Comparable {
		t.Error("a present result with no value was compared")
	}
	if !strings.Contains(timing.Reason, "no value") {
		t.Errorf("reason = %q, want it to name the missing value", timing.Reason)
	}
}

// --- Notes: a difference that is worth saying and is not a refusal ---

func TestADifferentSnapshotOrHarnessIsNotedAndNotRefused(t *testing.T) {
	baseline, candidate := comparablePair()
	candidate.SnapshotHash = "sha-candidate"
	candidate.SkillDir = "/skills/other-skill"
	candidate.Harness = "cursor"

	report := CompareProfiles(baseline, candidate)
	if !report.Comparable {
		t.Fatal("a comparison of two snapshots was refused — comparing two versions of a skill is what this is for")
	}
	joined := strings.Join(report.Notes, "\n")
	for _, want := range []string{"sha-baseline", "sha-candidate", "cursor", "/skills/other-skill"} {
		if !strings.Contains(joined, want) {
			t.Errorf("notes %q do not mention %q", joined, want)
		}
	}
}

func TestAnIdenticalPairCarriesNoNotes(t *testing.T) {
	baseline := presentProfile(testAdapterVersion, string(SourceOtel))
	if notes := CompareProfiles(baseline, baseline).Notes; len(notes) != 0 {
		t.Errorf("notes = %v, want none — the two sides agree on everything a note is about", notes)
	}
}

// --- The report as an artifact ---

func TestTheReportIsSchemaComparisonV1(t *testing.T) {
	report := CompareProfiles(comparablePair())
	if report.Schema != ComparisonSchema {
		t.Errorf("schema = %q, want %q", report.Schema, ComparisonSchema)
	}
	if ComparisonSchema != "skill-architect/comparison/v1" {
		t.Errorf("ComparisonSchema = %q, want skill-architect/comparison/v1", ComparisonSchema)
	}
	if report.GeneratedAt == "" {
		t.Error("the report does not say when it was generated")
	}
}

// The report round-trips, modulo key ordering — the rule the profile format
// already states. Ordering is not preserved because a comparison value is a
// `any`: it leaves as a struct in field order and comes back as a map, which
// Go marshals sorted. The document is the same document either way.
func TestTheReportRoundTrips(t *testing.T) {
	asDocument := func(data []byte) map[string]any {
		t.Helper()
		var doc map[string]any
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, data)
		}
		return doc
	}

	first, err := json.Marshal(CompareProfiles(comparablePair()))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back ComparisonReport
	if err := json.Unmarshal(first, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	second, err := json.Marshal(back)
	if err != nil {
		t.Fatalf("remarshal: %v", err)
	}

	// Non-vacuity: a comparison whose metrics went missing would round-trip
	// perfectly and prove nothing about them.
	if doc := asDocument(first); len(doc["metrics"].(map[string]any)) == 0 {
		t.Fatal("the marshalled report carries no metrics, so the round trip asserts nothing")
	}
	if !reflect.DeepEqual(asDocument(first), asDocument(second)) {
		t.Errorf("round-trip mismatch\nfirst:  %s\nsecond: %s", first, second)
	}
}

// --- LoadProfile ---

func writeProfileFile(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestLoadProfileReadsAProfileThisToolWrote(t *testing.T) {
	data, err := json.Marshal(presentProfile(testAdapterVersion, string(SourceOtel)))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := LoadProfile(writeProfileFile(t, "profile.json", string(data)))
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if got.SessionID != "session-1" || got.Capability.AdapterVer != testAdapterVersion {
		t.Errorf("loaded profile = %+v, want the one that was written", got)
	}
}

func TestLoadProfileRefusesWhatIsNotAProfileOfThisSchema(t *testing.T) {
	presentWithoutValue := func() string {
		p := presentProfile(testAdapterVersion, string(SourceOtel))
		p.Tokens.Value = nil
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		return string(data)
	}

	for _, tc := range []struct {
		name, body, wantIn string
	}{
		{"not JSON at all", "this is not json", "invalid"},
		{"JSON that is not a profile", `{"hello":"world"}`, "schema"},
		{"a profile of another schema", `{"schema":"skill-architect/profile/v2"}`, "profile/v2"},
		{"a metric marked present with no value", presentWithoutValue(), "tokens"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadProfile(writeProfileFile(t, "profile.json", tc.body))
			if err == nil {
				t.Fatalf("LoadProfile accepted %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantIn) {
				t.Errorf("error = %q, want it to mention %q", err, tc.wantIn)
			}
		})
	}
}

func TestLoadProfileSaysWhichFileItCouldNotRead(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent.json")
	_, err := LoadProfile(missing)
	if err == nil {
		t.Fatal("LoadProfile accepted a file that is not there")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("error = %q, want it to name the path", err)
	}
}
