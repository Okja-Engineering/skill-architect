package profiler

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

// ComparisonSchema is the canonical identifier for F04 comparison reports.
const ComparisonSchema = "skill-architect/comparison/v1"

// ProfileRef identifies the source of a side in a comparison.
type ProfileRef struct {
	Harness      string `json:"harness"`
	SessionID    string `json:"session_id"`
	SnapshotHash string `json:"snapshot_hash"`
	SkillDir     string `json:"skill_dir"`
}

// MetricComparison reports whether a single metric can be compared across two
// profiles and, if so, the baseline, candidate, and delta values. Both sides'
// MetricSource are always carried so a delta is never laundered into a single
// source; a comparison across different sources is refused outright (R-AT-01).
type MetricComparison struct {
	Comparable      bool   `json:"comparable"`
	Reason          string `json:"reason,omitempty"`
	BaselineSource  string `json:"baseline_source,omitempty"`
	CandidateSource string `json:"candidate_source,omitempty"`
	Note            string `json:"note,omitempty"` // e.g. "relative estimate, not billed"
	Baseline        any    `json:"baseline,omitempty"`
	Candidate       any    `json:"candidate,omitempty"`
	Delta           any    `json:"delta,omitempty"`
}

// ComparisonReport is the serialized result of a paired (F04) comparison.
type ComparisonReport struct {
	Schema      string                      `json:"schema"`
	GeneratedAt string                      `json:"generated_at"`
	Baseline    ProfileRef                  `json:"baseline"`
	Candidate   ProfileRef                  `json:"candidate"`
	Metrics     map[string]MetricComparison `json:"metrics"`
	Comparable  bool                        `json:"comparable"`
	Notes       []string                    `json:"notes,omitempty"`
}

// LoadProfile reads a profile JSON file and validates its schema.
func LoadProfile(path string) (Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, err
	}
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return Profile{}, err
	}
	if p.Schema != ProfileSchema {
		return Profile{}, fmt.Errorf("unknown profile schema %q, want %q", p.Schema, ProfileSchema)
	}
	if err := validateProfileValues(&p); err != nil {
		return Profile{}, err
	}
	return p, nil
}

// validateProfileValues rejects profiles that mark a metric "present" without a
// value — otherwise the comparator would dereference a nil pointer (D-3).
func validateProfileValues(p *Profile) error {
	type named struct {
		name    string
		state   MetricState
		missing bool
	}
	checks := []named{
		{"tokens", p.Tokens.State, p.Tokens.Value == nil},
		{"tool_calls", p.ToolCalls.State, p.ToolCalls.Value == nil},
		{"skill_activation", p.SkillActivation.State, p.SkillActivation.Value == nil},
		{"timing", p.Timing.State, p.Timing.Value == nil},
		{"attribution", p.Attribution.State, p.Attribution.Value == nil},
		{"estimated_context_tokens", p.EstimatedContextTokens.State, p.EstimatedContextTokens.Value == nil},
	}
	for _, c := range checks {
		if c.state == MetricPresent && c.missing {
			return fmt.Errorf("metric %q marked present but has no value", c.name)
		}
	}
	return nil
}

// guardPair returns a non-comparable MetricComparison when the two sides cannot
// be honestly compared: either side not present, a present state with a missing
// value, or mismatched metric sources. ok is true only when a delta is legal.
// Both sources are always recorded regardless of outcome.
func guardPair(metric string, bState, cState MetricState, bMissing, cMissing bool, bSrc, cSrc string) (MetricComparison, bool) {
	mc := MetricComparison{BaselineSource: bSrc, CandidateSource: cSrc}
	if bState != MetricPresent || cState != MetricPresent {
		mc.Reason = metric + " data not present in both profiles"
		return mc, false
	}
	if bMissing || cMissing {
		mc.Reason = metric + " marked present but missing value"
		return mc, false
	}
	if bSrc != cSrc {
		mc.Reason = fmt.Sprintf("source mismatch: baseline %q vs candidate %q — cross-source deltas are not comparable", bSrc, cSrc)
		return mc, false
	}
	return mc, true
}

// CompareProfiles compares a baseline and candidate profile, returning a report
// that only compares metrics present in both profiles.
func CompareProfiles(baseline, candidate Profile) ComparisonReport {
	report := ComparisonReport{
		Schema:      ComparisonSchema,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Baseline: ProfileRef{
			Harness:      baseline.Harness,
			SessionID:    baseline.SessionID,
			SnapshotHash: baseline.SnapshotHash,
			SkillDir:     baseline.SkillDir,
		},
		Candidate: ProfileRef{
			Harness:      candidate.Harness,
			SessionID:    candidate.SessionID,
			SnapshotHash: candidate.SnapshotHash,
			SkillDir:     candidate.SkillDir,
		},
		Metrics: make(map[string]MetricComparison),
	}

	if baseline.Harness != candidate.Harness {
		report.Notes = append(report.Notes, fmt.Sprintf("harness mismatch: baseline %q vs candidate %q", baseline.Harness, candidate.Harness))
	}
	if baseline.SnapshotHash != candidate.SnapshotHash {
		report.Notes = append(report.Notes, fmt.Sprintf("snapshot mismatch: baseline %q vs candidate %q", baseline.SnapshotHash, candidate.SnapshotHash))
	}
	if baseline.SkillDir != candidate.SkillDir {
		report.Notes = append(report.Notes, fmt.Sprintf("skill_dir mismatch: baseline %q vs candidate %q", baseline.SkillDir, candidate.SkillDir))
	}

	report.Metrics["tokens"] = compareTokens(baseline.Tokens, candidate.Tokens)
	report.Metrics["tool_calls"] = compareToolCalls(baseline.ToolCalls, candidate.ToolCalls)
	report.Metrics["timing"] = compareTiming(baseline.Timing, candidate.Timing)
	report.Metrics["skill_activation"] = compareSkillActivation(baseline.SkillActivation, candidate.SkillActivation)
	report.Metrics["attribution"] = compareAttribution(baseline.Attribution, candidate.Attribution)
	report.Metrics["estimated_context_tokens"] = compareEstimatedTokens(baseline.EstimatedContextTokens, candidate.EstimatedContextTokens)

	for _, mc := range report.Metrics {
		if mc.Comparable {
			report.Comparable = true
			break
		}
	}

	return report
}

func compareTokens(baseline, candidate TokenResult) MetricComparison {
	mc, ok := guardPair("tokens", baseline.State, candidate.State,
		baseline.Value == nil, candidate.Value == nil, baseline.Source, candidate.Source)
	if !ok {
		return mc
	}
	mc.Comparable = true
	mc.Baseline = baseline.Value
	mc.Candidate = candidate.Value
	mc.Delta = TokenCounts{
		Input:      candidate.Value.Input - baseline.Value.Input,
		Output:     candidate.Value.Output - baseline.Value.Output,
		CacheRead:  candidate.Value.CacheRead - baseline.Value.CacheRead,
		CacheWrite: candidate.Value.CacheWrite - baseline.Value.CacheWrite,
		Reasoning:  candidate.Value.Reasoning - baseline.Value.Reasoning,
	}
	return mc
}

func compareToolCalls(baseline, candidate ToolCallResult) MetricComparison {
	mc, ok := guardPair("tool call", baseline.State, candidate.State,
		baseline.Value == nil, candidate.Value == nil, baseline.Source, candidate.Source)
	if !ok {
		return mc
	}
	baseCount, baseOK := countToolCalls(baseline.Value)
	candCount, candOK := countToolCalls(candidate.Value)
	baseRate := successRate(baseOK, baseCount)
	candRate := successRate(candOK, candCount)
	mc.Comparable = true
	mc.Baseline = map[string]any{"count": baseCount, "successes": baseOK, "success_rate": baseRate}
	mc.Candidate = map[string]any{"count": candCount, "successes": candOK, "success_rate": candRate}
	mc.Delta = map[string]any{"count": candCount - baseCount, "successes": candOK - baseOK, "success_rate": round2(candRate - baseRate)}
	return mc
}

func countToolCalls(calls []ToolCallEntry) (total, success int) {
	total = len(calls)
	for _, c := range calls {
		if c.Success {
			success++
		}
	}
	return total, success
}

func successRate(success, total int) float64 {
	if total == 0 {
		return 0
	}
	return round2(float64(success) / float64(total))
}

func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}

func compareTiming(baseline, candidate TimingResult) MetricComparison {
	mc, ok := guardPair("timing", baseline.State, candidate.State,
		baseline.Value == nil, candidate.Value == nil, baseline.Source, candidate.Source)
	if !ok {
		return mc
	}
	mc.Comparable = true
	mc.Baseline = baseline.Value
	mc.Candidate = candidate.Value
	mc.Delta = map[string]int64{"total_ms": candidate.Value.TotalMs - baseline.Value.TotalMs}
	return mc
}

// compareSkillActivation compares the *sets* of skill names activated on each
// side, not just counts — a bare count cannot express which skill fired (D-5).
// Per-task precision/recall lives in the experiment layer (R-AT-06), not here.
func compareSkillActivation(baseline, candidate ActivationResult) MetricComparison {
	mc, ok := guardPair("skill activation", baseline.State, candidate.State,
		baseline.Value == nil, candidate.Value == nil, baseline.Source, candidate.Source)
	if !ok {
		return mc
	}
	baseNames := skillNameSet(baseline.Value)
	candNames := skillNameSet(candidate.Value)
	var onlyBase, onlyCand []string
	for n := range baseNames {
		if !candNames[n] {
			onlyBase = append(onlyBase, n)
		}
	}
	for n := range candNames {
		if !baseNames[n] {
			onlyCand = append(onlyCand, n)
		}
	}
	sort.Strings(onlyBase)
	sort.Strings(onlyCand)
	mc.Comparable = true
	mc.Baseline = map[string]any{"count": len(baseline.Value), "skills": sortedKeys(baseNames)}
	mc.Candidate = map[string]any{"count": len(candidate.Value), "skills": sortedKeys(candNames)}
	mc.Delta = map[string]any{
		"count":             len(candidate.Value) - len(baseline.Value),
		"only_in_baseline":  onlyBase,
		"only_in_candidate": onlyCand,
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

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func compareAttribution(baseline, candidate AttributionResult) MetricComparison {
	mc, ok := guardPair("attribution", baseline.State, candidate.State,
		baseline.Value == nil, candidate.Value == nil, baseline.Source, candidate.Source)
	if !ok {
		return mc
	}
	mc.Comparable = true
	mc.Baseline = map[string]int{"count": len(baseline.Value.Attributions)}
	mc.Candidate = map[string]int{"count": len(candidate.Value.Attributions)}
	mc.Delta = map[string]int{"count": len(candidate.Value.Attributions) - len(baseline.Value.Attributions)}
	return mc
}

// compareEstimatedTokens compares the chars/4 estimate field. The delta is an
// ordinal signal only (tier 4): the note must survive into any UI and no dollar
// figure may be rendered from it (measurement contract, spec.md).
func compareEstimatedTokens(baseline, candidate EstimatedTokensResult) MetricComparison {
	mc, ok := guardPair("estimated_context_tokens", baseline.State, candidate.State,
		baseline.Value == nil, candidate.Value == nil, baseline.Source, candidate.Source)
	if !ok {
		return mc
	}
	mc.Comparable = true
	mc.Note = "relative estimate, not billed — ordinal signal only"
	mc.Baseline = baseline.Value
	mc.Candidate = candidate.Value
	mc.Delta = EstimatedTokens{Total: candidate.Value.Total - baseline.Value.Total}
	return mc
}
