package profiler

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"time"
)

// The comparison answers one question: what changed about the skill between two
// runs. Everything here exists to stop it answering a different question — what
// changed about the *reader* — and presenting that answer as if it were about
// the skill. A stored 0.4.x profile against a fresh 0.5.0 one differs by four
// releases of fixes to how the export is read; subtracted, those fixes read as
// the skill's regression.
//
// So a difference is reported only when both sides can be shown to have been
// measured the same way. Two things decide that, at two levels: the adapter
// version, which is a property of the pair and refuses the whole comparison,
// and the metric source, which is a property of one signal and refuses that
// signal. Neither is a warning beside a delta — a refusal means no delta is
// computed at all, because a number in the report is a number somebody will
// read.

// ComparisonSchema is the version string embedded in every comparison report.
const ComparisonSchema = "skill-architect/comparison/v1"

// ProfileRef identifies one side of a comparison.
//
// AdapterVersion is part of a side's identity rather than only of a refusal
// message: a stored comparison has to stay interpretable, and "which reader
// produced this" is the question the refusal below is about.
type ProfileRef struct {
	Harness        string `json:"harness"`
	SessionID      string `json:"session_id"`
	SnapshotHash   string `json:"snapshot_hash"`
	SkillDir       string `json:"skill_dir"`
	AdapterVersion string `json:"adapter_version"`
}

// MetricComparison is one signal's answer.
//
// Both sources are always carried — they have no `omitempty` — so a delta is
// never laundered into a single source and a reader can always see what each
// side was read from. A side that read nothing reports the source "none", which
// is the vocabulary's own word for it; an empty string is not a member of that
// vocabulary, and a reader walking it would be wrong rather than ignorant.
//
// Baseline, Candidate and Delta are absent together. A comparison that is not
// comparable carries a Reason and no numbers at all.
type MetricComparison struct {
	Comparable      bool   `json:"comparable"`
	Reason          string `json:"reason,omitempty"`
	BaselineSource  string `json:"baseline_source"`
	CandidateSource string `json:"candidate_source"`
	Note            string `json:"note,omitempty"`
	Baseline        any    `json:"baseline,omitempty"`
	Candidate       any    `json:"candidate,omitempty"`
	Delta           any    `json:"delta,omitempty"`
}

// ComparisonReport is the serialized result of a paired comparison.
//
// Refusal and Notes are different kinds of statement and are kept apart.
// A refusal is the reason *no* metric in this pair may be compared, and when it
// is set every MetricComparison carries it too — a report that said "not
// comparable" at the top and carried a delta underneath would still hand the
// fabricated number to anything reading one metric out of it. A note is a
// difference worth stating that does not stop the comparison: comparing two
// snapshots of a skill is what this tool is for, so a different snapshot hash
// is a note and never a refusal.
type ComparisonReport struct {
	Schema      string                          `json:"schema"`
	GeneratedAt string                          `json:"generated_at"` // ISO 8601 UTC
	Baseline    ProfileRef                      `json:"baseline"`
	Candidate   ProfileRef                      `json:"candidate"`
	Comparable  bool                            `json:"comparable"`
	Refusal     string                          `json:"refusal,omitempty"`
	Metrics     map[MetricName]MetricComparison `json:"metrics"`
	Notes       []string                        `json:"notes,omitempty"`
}

// LoadProfile reads a profile from a file and refuses anything this comparator
// cannot honestly read: another schema, or a signal marked "present" with no
// value behind it. Every error names the path, because a caller comparing two
// files needs to know which one it was.
func LoadProfile(path string) (Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, fmt.Errorf("read profile %s: %w", path, err)
	}
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return Profile{}, fmt.Errorf("read profile %s: %w", path, err)
	}
	if p.Schema != ProfileSchema {
		return Profile{}, fmt.Errorf("read profile %s: schema is %q, want %q", path, p.Schema, ProfileSchema)
	}
	if err := validateProfileValues(p); err != nil {
		return Profile{}, fmt.Errorf("read profile %s: %w", path, err)
	}
	return p, nil
}

// validateProfileValues refuses a profile whose signal says "present" and
// carries no value. Left to the comparator, that shape is a nil dereference;
// refused here, it is a file somebody can go and look at.
//
// The comparator refuses it a second time, per signal, because CompareProfiles
// is a library entry point that a caller can reach without going through a
// file at all.
func validateProfileValues(p Profile) error {
	sides := profileSides(p)
	for _, name := range sortedMetricNames(sides) {
		if s := sides[name]; s.state == MetricPresent && s.missing {
			return fmt.Errorf("%s is marked %q and the profile carries no value for it", name, MetricPresent)
		}
	}
	return nil
}

// signalValue is one signal's result reduced to what a comparison needs: what
// it says about itself, and whether there is anything behind it.
type signalValue struct {
	state   MetricState
	source  string
	missing bool
}

// profileSides reduces every signal the profile carries, once. It is the single
// place the comparison and the file validation agree about what a signal is, so
// a signal cannot be compared by one and skipped by the other — and it is the
// one place an unread signal's empty source becomes "none", which is the
// vocabulary's word for it. An empty string is not a member of that vocabulary,
// and a reader walking it would be wrong rather than ignorant.
//
// For the two list-valued signals, "no value" is an empty list and not only a
// nil one, because that is what the profile format says they mean: a result
// with no entries is never present (see ToolCallResult), and `null` and `[]`
// are the same absence once a file has been through a JSON decoder. An adapter
// reporting a session that called no tool says so with `unknown` and a reason.
func profileSides(p Profile) map[MetricName]signalValue {
	estimate := p.EstimatedContextTokens.orAbsent()
	return map[MetricName]signalValue{
		MetricTokens:                 side(p.Tokens.RawMetricResult, p.Tokens.Value == nil),
		MetricToolCalls:              side(p.ToolCalls.RawMetricResult, len(p.ToolCalls.Value) == 0),
		MetricSkillActivation:        side(p.SkillActivation.RawMetricResult, len(p.SkillActivation.Value) == 0),
		MetricTiming:                 side(p.Timing.RawMetricResult, p.Timing.Value == nil),
		MetricAttribution:            side(p.Attribution.RawMetricResult, p.Attribution.Value == nil),
		MetricEstimatedContextTokens: side(estimate.RawMetricResult, estimate.Value == nil),
	}
}

func side(raw RawMetricResult, missing bool) signalValue {
	source := raw.Source
	if source == "" {
		source = string(SourceNone)
	}
	return signalValue{state: raw.State, source: source, missing: missing}
}

// sortedMetricNames keeps a walk over a map from reporting a different signal
// on two runs of the same input.
func sortedMetricNames(sides map[MetricName]signalValue) []MetricName {
	names := make([]MetricName, 0, len(sides))
	for name := range sides {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}

// adapterVersionRefusal is the whole of the pair-level rule: a comparison is
// allowed only between two profiles that name the same, non-empty adapter
// version.
//
// AdapterVersion is bumped whenever the adapter changes what a profile contains
// for the same input, which is precisely the condition under which a delta
// stops being about the session. This is that constant's first consumer and the
// reason it exists.
//
// Two profiles that name no version at all are not two profiles that agree:
// "" == "" is agreement between two unknowns, and neither of them can show it
// was read the way the other was.
func adapterVersionRefusal(baseline, candidate string) string {
	if baseline != candidate {
		return fmt.Sprintf("adapter version mismatch: baseline %q vs candidate %q — the two profiles were read by different adapters, so a difference between them may be the reader changing rather than the skill",
			baseline, candidate)
	}
	if baseline == "" {
		return "neither profile names an adapter version: capability.adapter_version is empty on both sides, so nothing shows the two were read the same way"
	}
	return ""
}

// CompareProfiles compares a baseline and a candidate profile.
func CompareProfiles(baseline, candidate Profile) ComparisonReport {
	report := ComparisonReport{
		Schema:      ComparisonSchema,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Baseline:    profileRef(baseline),
		Candidate:   profileRef(candidate),
		Refusal: adapterVersionRefusal(
			baseline.Capability.AdapterVer, candidate.Capability.AdapterVer),
		Notes:   pairNotes(baseline, candidate),
		Metrics: make(map[MetricName]MetricComparison),
	}

	c := comparator{
		refusal:   report.Refusal,
		baseline:  profileSides(baseline),
		candidate: profileSides(candidate),
	}
	report.Metrics[MetricTokens] = c.tokens(baseline.Tokens, candidate.Tokens)
	report.Metrics[MetricToolCalls] = c.toolCalls(baseline.ToolCalls, candidate.ToolCalls)
	report.Metrics[MetricSkillActivation] = c.activation(baseline.SkillActivation, candidate.SkillActivation)
	report.Metrics[MetricTiming] = c.timing(baseline.Timing, candidate.Timing)
	report.Metrics[MetricAttribution] = c.attribution(baseline.Attribution, candidate.Attribution)
	report.Metrics[MetricEstimatedContextTokens] = c.estimate(
		baseline.EstimatedContextTokens.orAbsent(), candidate.EstimatedContextTokens.orAbsent())

	// Derived, not asserted: the report is comparable when some signal was.
	// A refusal makes every signal incomparable by construction, so it needs no
	// clause of its own here — which is what keeps the two from disagreeing.
	for _, mc := range report.Metrics {
		if mc.Comparable {
			report.Comparable = true
			break
		}
	}
	return report
}

func profileRef(p Profile) ProfileRef {
	return ProfileRef{
		Harness:        p.Harness,
		SessionID:      p.SessionID,
		SnapshotHash:   p.SnapshotHash,
		SkillDir:       p.SkillDir,
		AdapterVersion: p.Capability.AdapterVer,
	}
}

// pairNotes states the differences that are worth knowing and do not stop a
// comparison. A different snapshot hash is the *point* of a paired run — a
// skill before and after a change — so it is said and not refused.
func pairNotes(baseline, candidate Profile) []string {
	var notes []string
	for _, n := range []struct {
		field               string
		baseline, candidate string
	}{
		{"harness", baseline.Harness, candidate.Harness},
		{"snapshot_hash", baseline.SnapshotHash, candidate.SnapshotHash},
		{"skill_dir", baseline.SkillDir, candidate.SkillDir},
	} {
		if n.baseline != n.candidate {
			notes = append(notes, fmt.Sprintf("%s differs: baseline %q vs candidate %q", n.field, n.baseline, n.candidate))
		}
	}
	return notes
}

// comparator carries the pair-level verdict so that every signal answers to it.
// It is held here rather than applied to the finished report because a refusal
// has to be upstream of the subtraction: a delta computed and then marked
// incomparable is still a delta in the document.
type comparator struct {
	refusal   string
	baseline  map[MetricName]signalValue
	candidate map[MetricName]signalValue
}

// pair is the gate every signal passes through, in the order the questions stop
// mattering: if the pair cannot be compared, what one signal says about itself
// is irrelevant; if a signal was not read on both sides there is nothing to
// subtract; and a delta across two sources is not a measurement of one thing.
//
// Both sources are recorded whatever the outcome.
func (c comparator) pair(metric MetricName) (MetricComparison, bool) {
	baseline, candidate := c.baseline[metric], c.candidate[metric]
	mc := MetricComparison{
		BaselineSource:  baseline.source,
		CandidateSource: candidate.source,
	}
	switch {
	case c.refusal != "":
		mc.Reason = c.refusal
	case baseline.state != MetricPresent || candidate.state != MetricPresent:
		mc.Reason = fmt.Sprintf("%s is not present in both profiles: baseline %q, candidate %q",
			metric, baseline.state, candidate.state)
	case baseline.missing || candidate.missing:
		mc.Reason = fmt.Sprintf("%s is marked %q and one of the profiles carries no value for it",
			metric, MetricPresent)
	case baseline.source != candidate.source:
		mc.Reason = fmt.Sprintf("source mismatch: baseline %q vs candidate %q — a delta across two sources is not a measurement of the same thing",
			baseline.source, candidate.source)
	default:
		return mc, true
	}
	return mc, false
}

func (c comparator) tokens(baseline, candidate TokenResult) MetricComparison {
	mc, ok := c.pair(MetricTokens)
	if !ok {
		return mc
	}
	delta, shared := tokenDelta(*baseline.Value, *candidate.Value)
	if shared == 0 {
		// Every key of the delta would be absent, and an object with no keys
		// reads as "no change" rather than as "nothing was subtracted".
		mc.Reason = "the two profiles share no token count that both of them read, so there is nothing to subtract"
		return mc
	}
	mc.Comparable = true
	mc.Baseline = baseline.Value
	mc.Candidate = candidate.Value
	mc.Delta = delta
	return mc
}

// tokenDelta subtracts the counts both profiles read, and says how many there
// were.
//
// Every count is a pointer because "the export said nothing about this" and
// "the export said zero" are different answers. A delta inherits that: a count
// only one side read has no key in the delta either, because the difference
// between a number and an absence is not a number. Its absence is legible
// beside the two values, which the comparison carries in full.
func tokenDelta(baseline, candidate TokenCounts) (TokenCounts, int) {
	shared := 0
	sub := func(b, c *int) *int {
		if b == nil || c == nil {
			return nil
		}
		shared++
		return Count(*c - *b)
	}
	return TokenCounts{
		Input:         sub(baseline.Input, candidate.Input),
		Output:        sub(baseline.Output, candidate.Output),
		CacheRead:     sub(baseline.CacheRead, candidate.CacheRead),
		CacheCreation: sub(baseline.CacheCreation, candidate.CacheCreation),
		Reasoning:     sub(baseline.Reasoning, candidate.Reasoning),
	}, shared
}

func (c comparator) toolCalls(baseline, candidate ToolCallResult) MetricComparison {
	mc, ok := c.pair(MetricToolCalls)
	if !ok {
		return mc
	}
	baseTotal, baseOK := countToolCalls(baseline.Value)
	candTotal, candOK := countToolCalls(candidate.Value)
	baseRate := successRate(baseOK, baseTotal)
	candRate := successRate(candOK, candTotal)
	mc.Comparable = true
	mc.Baseline = map[string]any{"count": baseTotal, "successes": baseOK, "success_rate": baseRate}
	mc.Candidate = map[string]any{"count": candTotal, "successes": candOK, "success_rate": candRate}
	mc.Delta = map[string]any{
		"count":        candTotal - baseTotal,
		"successes":    candOK - baseOK,
		"success_rate": round2(candRate - baseRate),
	}
	return mc
}

// countToolCalls counts calls, not entries. An entry carrying an aggregate
// Count is that many calls — that is what the field means for a source that
// reports a delta metric rather than one record per call — and counting entries
// would under-report every such source by however much it aggregated.
func countToolCalls(calls []ToolCallEntry) (total, success int) {
	for _, c := range calls {
		n := c.Count
		if n == 0 {
			n = 1 // an absent aggregate is one call
		}
		total += n
		if c.Success {
			success += n
		}
	}
	return total, success
}

// successRate is successes over calls, to two places.
//
// The zero guard is not reachable through CompareProfiles — a present tool-call
// result must carry entries, and profileSides refuses one that does not — but a
// rate is the one number here that divides, and an unguarded division would put
// a NaN into a document somebody reads. It is asserted directly rather than
// through the comparator, because that is the only way to reach it.
func successRate(success, total int) float64 {
	if total == 0 {
		return 0
	}
	return round2(float64(success) / float64(total))
}

// round2 rounds half away from zero, in both directions. Truncating after
// adding a half rounds negatives towards zero instead, and a success-rate delta
// is negative exactly when the candidate did worse — the direction a reader
// most needs to be right.
func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

func (c comparator) timing(baseline, candidate TimingResult) MetricComparison {
	mc, ok := c.pair(MetricTiming)
	if !ok {
		return mc
	}
	mc.Comparable = true
	mc.Baseline = baseline.Value
	mc.Candidate = candidate.Value
	mc.Delta = map[string]int64{"total_ms": candidate.Value.TotalMs - baseline.Value.TotalMs}
	return mc
}

// activation compares the *sets* of skill names on each side as well as the
// count. A bare count cannot say which skill stopped firing, which is the
// question a paired run is usually asking.
func (c comparator) activation(baseline, candidate ActivationResult) MetricComparison {
	mc, ok := c.pair(MetricSkillActivation)
	if !ok {
		return mc
	}
	baseNames := skillNameSet(baseline.Value)
	candNames := skillNameSet(candidate.Value)
	mc.Comparable = true
	mc.Baseline = map[string]any{"count": len(baseline.Value), "skills": sortedNames(baseNames)}
	mc.Candidate = map[string]any{"count": len(candidate.Value), "skills": sortedNames(candNames)}
	mc.Delta = map[string]any{
		"count":             len(candidate.Value) - len(baseline.Value),
		"only_in_baseline":  missingFrom(baseNames, candNames),
		"only_in_candidate": missingFrom(candNames, baseNames),
	}
	return mc
}

func skillNameSet(entries []ActivationEntry) map[string]bool {
	set := make(map[string]bool, len(entries))
	for _, e := range entries {
		set[e.SkillName] = true
	}
	return set
}

// missingFrom is the names in one set and not the other, sorted. It returns an
// empty slice rather than nil so the key is an empty list in the JSON and never
// `null`: "no skill stopped firing" and "this was not computed" are different
// answers.
func missingFrom(set, other map[string]bool) []string {
	only := []string{}
	for name := range set {
		if !other[name] {
			only = append(only, name)
		}
	}
	sort.Strings(only)
	return only
}

func sortedNames(set map[string]bool) []string {
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c comparator) attribution(baseline, candidate AttributionResult) MetricComparison {
	mc, ok := c.pair(MetricAttribution)
	if !ok {
		return mc
	}
	mc.Comparable = true
	mc.Baseline = map[string]int{"count": len(baseline.Value.Attributions)}
	mc.Candidate = map[string]int{"count": len(candidate.Value.Attributions)}
	mc.Delta = map[string]int{"count": len(candidate.Value.Attributions) - len(baseline.Value.Attributions)}
	return mc
}

// estimateNote rides on every estimate comparison. The estimate is chars/4 over
// payload bytes: comparable between two runs of one harness, and never a billed
// count. The note is part of the answer rather than documentation of it, so it
// survives into anything that renders the report.
const estimateNote = "relative estimate, not billed — an ordinal signal only, and never a cost"

func (c comparator) estimate(baseline, candidate EstimatedTokensResult) MetricComparison {
	mc, ok := c.pair(MetricEstimatedContextTokens)
	if !ok {
		return mc
	}
	mc.Comparable = true
	mc.Note = estimateNote
	mc.Baseline = baseline.Value
	mc.Candidate = candidate.Value
	mc.Delta = EstimatedTokens{Total: candidate.Value.Total - baseline.Value.Total}
	return mc
}
