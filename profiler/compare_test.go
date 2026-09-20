package profiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Measurement-contract tests (cursor-profiler spec R-AT-01..R-AT-04) ---

func TestCompareProfiles_SourceMismatchRefused(t *testing.T) {
	baseline := Profile{
		Schema:    ProfileSchema,
		Harness:   "claude_code",
		Tokens:    PresentTokenResult(TokenCounts{Input: 100}, string(SourceSessionData)),
		ToolCalls: PresentToolCallResult([]ToolCallEntry{{Name: "Bash", Success: true}}, string(SourceSessionData)),
	}
	candidate := Profile{
		Schema:    ProfileSchema,
		Harness:   "claude_code",
		Tokens:    PresentTokenResult(TokenCounts{Input: 120}, string(SourceOtel)),
		ToolCalls: PresentToolCallResult([]ToolCallEntry{{Name: "Bash", Success: true}}, string(SourceOtel)),
	}

	report := CompareProfiles(baseline, candidate)

	tok := report.Metrics["tokens"]
	if tok.Comparable {
		t.Error("tokens must not be comparable across different sources (session_data vs otel)")
	}
	if !strings.Contains(tok.Reason, "session_data") || !strings.Contains(tok.Reason, "otel") {
		t.Errorf("reason should name both sources, got %q", tok.Reason)
	}
	if tok.Delta != nil {
		t.Error("a cross-source delta must never be produced")
	}
	if tok.BaselineSource != string(SourceSessionData) || tok.CandidateSource != string(SourceOtel) {
		t.Errorf("report must carry both sources, got %q/%q", tok.BaselineSource, tok.CandidateSource)
	}
}

func TestCompareProfiles_SameSourceCarriesBoth(t *testing.T) {
	baseline := Profile{
		Schema:  ProfileSchema,
		Harness: "claude_code",
		Tokens:  PresentTokenResult(TokenCounts{Input: 100}, string(SourceOtel)),
	}
	candidate := Profile{
		Schema:  ProfileSchema,
		Harness: "claude_code",
		Tokens:  PresentTokenResult(TokenCounts{Input: 120}, string(SourceOtel)),
	}

	report := CompareProfiles(baseline, candidate)
	tok := report.Metrics["tokens"]
	if !tok.Comparable {
		t.Fatal("same-source comparison should be comparable")
	}
	if tok.BaselineSource != string(SourceOtel) || tok.CandidateSource != string(SourceOtel) {
		t.Errorf("both sources must be carried, got %q/%q", tok.BaselineSource, tok.CandidateSource)
	}
}

func TestCompareProfiles_EstimatedContextTokens(t *testing.T) {
	baseline := Profile{
		Schema:                 ProfileSchema,
		Harness:                "cursor",
		EstimatedContextTokens: PresentEstimatedTokensResult(EstimatedTokens{Total: 1000}, string(SourceHooksEstimated)),
	}
	candidate := Profile{
		Schema:                 ProfileSchema,
		Harness:                "cursor",
		EstimatedContextTokens: PresentEstimatedTokensResult(EstimatedTokens{Total: 800}, string(SourceHooksEstimated)),
	}

	report := CompareProfiles(baseline, candidate)
	est := report.Metrics["estimated_context_tokens"]
	if !est.Comparable {
		t.Fatal("estimated_context_tokens must be compared when both sides carry it")
	}
	delta, ok := est.Delta.(EstimatedTokens)
	if !ok {
		t.Fatalf("delta type = %T, want EstimatedTokens", est.Delta)
	}
	if delta.Total != -200 {
		t.Errorf("estimated token delta = %d, want -200", delta.Total)
	}
	if !strings.Contains(est.Note, "relative estimate") {
		t.Errorf("estimate comparison must be labelled, got note %q", est.Note)
	}
	if strings.Contains(est.Note, "$") {
		t.Error("tier-4 comparisons must never render a dollar figure")
	}
}

func TestLoadProfile_PresentWithoutValueFails(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "bad.json")
	// Hand-written profile: state present, no value key — previously loaded
	// cleanly then panicked the comparator (defect D-3).
	raw := `{"schema":"skill-architect/profile/v1","harness":"cursor","tokens":{"state":"present"}}`
	if err := os.WriteFile(file, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadProfile(file); err == nil {
		t.Fatal("LoadProfile must reject a present metric with no value")
	}
}

func TestCompareProfiles_NilValueDoesNotPanic(t *testing.T) {
	// CompareProfiles is exported; profiles can arrive from anywhere, so the
	// comparator must defend itself even if LoadProfile was bypassed.
	baseline := Profile{
		Schema:  ProfileSchema,
		Harness: "cursor",
		Tokens:  TokenResult{RawMetricResult: RawMetricResult{State: MetricPresent, Source: "hooks"}},
		Timing:  TimingResult{RawMetricResult: RawMetricResult{State: MetricPresent, Source: "hooks"}},
	}
	candidate := Profile{
		Schema:  ProfileSchema,
		Harness: "cursor",
		Tokens:  TokenResult{RawMetricResult: RawMetricResult{State: MetricPresent, Source: "hooks"}},
		Timing:  TimingResult{RawMetricResult: RawMetricResult{State: MetricPresent, Source: "hooks"}},
	}

	report := CompareProfiles(baseline, candidate) // must not panic
	if report.Metrics["tokens"].Comparable {
		t.Error("present-without-value must not be comparable")
	}
	if report.Metrics["timing"].Comparable {
		t.Error("present-without-value must not be comparable")
	}
}

func TestCompareProfiles_SkillActivationByName(t *testing.T) {
	baseline := Profile{
		Schema:  ProfileSchema,
		Harness: "cursor",
		SkillActivation: PresentActivationResult([]ActivationEntry{
			{SkillName: "skill-audit"},
			{SkillName: "skill-rewrite"},
		}, string(SourceHooks)),
	}
	candidate := Profile{
		Schema:  ProfileSchema,
		Harness: "cursor",
		SkillActivation: PresentActivationResult([]ActivationEntry{
			{SkillName: "skill-audit"},
		}, string(SourceHooks)),
	}

	report := CompareProfiles(baseline, candidate)
	act := report.Metrics["skill_activation"]
	if !act.Comparable {
		t.Fatal("same-source activation should be comparable")
	}
	delta, ok := act.Delta.(map[string]any)
	if !ok {
		t.Fatalf("activation delta type = %T, want map", act.Delta)
	}
	onlyBase, _ := delta["only_in_baseline"].([]string)
	if len(onlyBase) != 1 || onlyBase[0] != "skill-rewrite" {
		t.Errorf("only_in_baseline = %v, want [skill-rewrite]", delta["only_in_baseline"])
	}
}

// --- D-4: Claude Code skill.name activation ---

func TestClaudeCode_SkillNameActivation(t *testing.T) {
	tmp := t.TempDir()
	otelFile := filepath.Join(tmp, "otel.json")
	export := map[string]any{
		"metrics": []map[string]any{
			{
				"name":       "claude_code.token.usage",
				"attributes": map[string]any{"token_type": "input", "skill.name": "skill-audit"},
				"value":      100,
			},
			{
				"name":       "claude_code.token.usage",
				"attributes": map[string]any{"token_type": "output", "skill.name": "skill-audit"},
				"value":      50,
			},
			{
				"name":       "claude_code.cost.usage",
				"attributes": map[string]any{"skill.name": "skill-rewrite"},
				"value":      0.02,
			},
		},
		"logs": []map[string]any{
			{"event_name": "claude_code.api_request", "timestamp": "2026-09-11T10:00:00Z"},
		},
	}
	data, _ := json.Marshal(export)
	if err := os.WriteFile(otelFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	adapter := ClaudeCodeAdapter{OtelExportFile: otelFile}
	cap := adapter.Probe()
	if cap.Capabilities[MetricSkillActivation] != SourceOtel {
		t.Errorf("probe skill_activation = %q, want otel when skill.name present", cap.Capabilities[MetricSkillActivation])
	}

	profile, err := adapter.Capture("sess-1", CaptureOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if profile.SkillActivation.State != MetricPresent {
		t.Fatalf("skill_activation = %q (%s), want present", profile.SkillActivation.State, profile.SkillActivation.Reason)
	}
	if profile.SkillActivation.Source != string(SourceOtel) {
		t.Errorf("activation source = %q, want otel", profile.SkillActivation.Source)
	}
	names := map[string]bool{}
	for _, e := range profile.SkillActivation.Value {
		names[e.SkillName] = true
	}
	if !names["skill-audit"] || !names["skill-rewrite"] {
		t.Errorf("activation names = %v, want skill-audit and skill-rewrite", names)
	}
}

func TestClaudeCode_NoSkillNameStillUnknown(t *testing.T) {
	tmp := t.TempDir()
	otelFile := filepath.Join(tmp, "otel.json")
	export := map[string]any{
		"metrics": []map[string]any{
			{
				"name":       "claude_code.token.usage",
				"attributes": map[string]any{"token_type": "input"},
				"value":      100,
			},
		},
	}
	data, _ := json.Marshal(export)
	if err := os.WriteFile(otelFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	adapter := ClaudeCodeAdapter{OtelExportFile: otelFile}
	profile, err := adapter.Capture("sess-1", CaptureOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if profile.SkillActivation.State != MetricUnknown {
		t.Errorf("skill_activation = %q, want unknown when no skill.name attributes", profile.SkillActivation.State)
	}
}
