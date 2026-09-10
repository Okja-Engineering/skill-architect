package profiler

import (
	"encoding/json"
	"fmt"
	"os"
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
// profiles and, if so, the baseline, candidate, and delta values.
type MetricComparison struct {
	Comparable bool   `json:"comparable"`
	Reason     string `json:"reason,omitempty"`
	Source     string `json:"source,omitempty"`
	Baseline   any    `json:"baseline,omitempty"`
	Candidate  any    `json:"candidate,omitempty"`
	Delta      any    `json:"delta,omitempty"`
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
	return p, nil
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

	for _, mc := range report.Metrics {
		if mc.Comparable {
			report.Comparable = true
			break
		}
	}

	return report
}

func compareTokens(baseline, candidate TokenResult) MetricComparison {
	if baseline.State != MetricPresent || candidate.State != MetricPresent {
		return MetricComparison{
			Comparable: false,
			Reason:     "token data not present in both profiles",
		}
	}
	return MetricComparison{
		Comparable: true,
		Source:     candidate.Source,
		Baseline:   baseline.Value,
		Candidate:  candidate.Value,
		Delta: TokenCounts{
			Input:         candidate.Value.Input - baseline.Value.Input,
			Output:        candidate.Value.Output - baseline.Value.Output,
			CacheRead:     candidate.Value.CacheRead - baseline.Value.CacheRead,
			CacheCreation: candidate.Value.CacheCreation - baseline.Value.CacheCreation,
			Reasoning:     candidate.Value.Reasoning - baseline.Value.Reasoning,
		},
	}
}

func compareToolCalls(baseline, candidate ToolCallResult) MetricComparison {
	if baseline.State != MetricPresent || candidate.State != MetricPresent {
		return MetricComparison{
			Comparable: false,
			Reason:     "tool call data not present in both profiles",
		}
	}
	baseCount, baseOK := countToolCalls(baseline.Value)
	candCount, candOK := countToolCalls(candidate.Value)
	baseRate := successRate(baseOK, baseCount)
	candRate := successRate(candOK, candCount)
	return MetricComparison{
		Comparable: true,
		Source:     candidate.Source,
		Baseline:   map[string]any{"count": baseCount, "successes": baseOK, "success_rate": baseRate},
		Candidate:  map[string]any{"count": candCount, "successes": candOK, "success_rate": candRate},
		Delta:      map[string]any{"count": candCount - baseCount, "successes": candOK - baseOK, "success_rate": round2(candRate - baseRate)},
	}
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
	if baseline.State != MetricPresent || candidate.State != MetricPresent {
		return MetricComparison{
			Comparable: false,
			Reason:     "timing data not present in both profiles",
		}
	}
	return MetricComparison{
		Comparable: true,
		Source:     candidate.Source,
		Baseline:   baseline.Value,
		Candidate:  candidate.Value,
		Delta:      map[string]int64{"total_ms": candidate.Value.TotalMs - baseline.Value.TotalMs},
	}
}

func compareSkillActivation(baseline, candidate ActivationResult) MetricComparison {
	if baseline.State != MetricPresent || candidate.State != MetricPresent {
		return MetricComparison{
			Comparable: false,
			Reason:     "skill activation data not present in both profiles",
		}
	}
	return MetricComparison{
		Comparable: true,
		Source:     candidate.Source,
		Baseline:   map[string]int{"count": len(baseline.Value)},
		Candidate:  map[string]int{"count": len(candidate.Value)},
		Delta:      map[string]int{"count": len(candidate.Value) - len(baseline.Value)},
	}
}

func compareAttribution(baseline, candidate AttributionResult) MetricComparison {
	if baseline.State != MetricPresent || candidate.State != MetricPresent {
		return MetricComparison{
			Comparable: false,
			Reason:     "attribution data not present in both profiles",
		}
	}
	return MetricComparison{
		Comparable: true,
		Source:     candidate.Source,
		Baseline:   map[string]int{"count": len(baseline.Value.Attributions)},
		Candidate:  map[string]int{"count": len(candidate.Value.Attributions)},
		Delta:      map[string]int{"count": len(candidate.Value.Attributions) - len(baseline.Value.Attributions)},
	}
}
