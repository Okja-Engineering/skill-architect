package profiler

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// ExperimentSchema is the canonical identifier for F04 experiment designs.
const ExperimentSchema = "skill-architect/experiment/v1"

// ExperimentPlanSchema is the canonical identifier for generated plans.
const ExperimentPlanSchema = "skill-architect/experiment-plan/v1"

// Budget caps spend for the whole experiment or a single run.
type Budget struct {
	MaxTokens int   `json:"max_tokens,omitempty"`
	MaxTimeMs int64 `json:"max_time_ms,omitempty"`
}

// Condition describes one side of a paired comparison (baseline or candidate).
type Condition struct {
	Name         string `json:"name"`
	Harness      string `json:"harness"`
	SnapshotHash string `json:"snapshot_hash"`
	SkillDir     string `json:"skill_dir"`
	Command      string `json:"command"`
	ExportFile   string `json:"export_file,omitempty"`
	OtelFile     string `json:"otel_file,omitempty"`
}

// ExperimentDesign is the human-declared specification of a paired comparison.
//
// Before running, the maintainer must declare what is being compared, on which
// tasks, how many times, and under what budget. This keeps the experiment
// recomputable and the results honest.
type ExperimentDesign struct {
	Schema           string    `json:"schema"`
	Name             string    `json:"name"`
	TaskFamilies     []string  `json:"task_families"`
	Repetitions      int       `json:"repetitions"`
	Ordering         string    `json:"ordering"` // blocked, random
	Budget           Budget    `json:"budget,omitempty"`
	QualityTolerance float64   `json:"quality_tolerance"`
	EfficiencyTarget float64   `json:"efficiency_target"`
	AnalysisMethod   string    `json:"analysis_method"` // difference, ratio
	StoppingRule     string    `json:"stopping_rule"`   // fixed, threshold
	Baseline         Condition `json:"baseline"`
	Candidate        Condition `json:"candidate"`
	OutputDir        string    `json:"output_dir,omitempty"`
}

// ExperimentStep is a single execution required to produce one profile.
type ExperimentStep struct {
	Condition    string `json:"condition"`
	TaskFamily   string `json:"task_family"`
	Repetition   int    `json:"repetition"`
	Command      string `json:"command"`
	ProfilePath  string `json:"profile_path"`
	SnapshotHash string `json:"snapshot_hash"`
	SkillDir     string `json:"skill_dir"`
	Harness      string `json:"harness"`
	ExportFile   string `json:"export_file,omitempty"`
	OtelFile     string `json:"otel_file,omitempty"`
}

// ExperimentRun is one repetition of one task family.
type ExperimentRun struct {
	TaskFamily string           `json:"task_family"`
	Repetition int              `json:"repetition"`
	Steps      []ExperimentStep `json:"steps"`
	Comparison string           `json:"comparison_output"`
}

// ExperimentPlan is the materialized sequence of steps to execute.
type ExperimentPlan struct {
	Schema      string           `json:"schema"`
	GeneratedAt string           `json:"generated_at"`
	Design      ExperimentDesign `json:"design"`
	Runs        []ExperimentRun  `json:"runs"`
	Notes       []string         `json:"notes,omitempty"`
}

// LoadExperimentDesign reads and validates an experiment design file.
func LoadExperimentDesign(path string) (ExperimentDesign, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ExperimentDesign{}, err
	}
	var d ExperimentDesign
	if err := json.Unmarshal(data, &d); err != nil {
		return ExperimentDesign{}, err
	}
	if d.Schema != ExperimentSchema {
		return ExperimentDesign{}, fmt.Errorf("unknown experiment schema %q, want %q", d.Schema, ExperimentSchema)
	}
	return NormalizeDesign(d)
}

// NormalizeDesign applies defaults and validates the experiment design.
func NormalizeDesign(d ExperimentDesign) (ExperimentDesign, error) {
	if d.Name == "" {
		return d, fmt.Errorf("name is required")
	}
	if len(d.TaskFamilies) == 0 {
		return d, fmt.Errorf("task_families is required")
	}
	if d.Repetitions <= 0 {
		d.Repetitions = 1
	}
	if d.Baseline.Command == "" {
		return d, fmt.Errorf("baseline.command is required")
	}
	if d.Candidate.Command == "" {
		return d, fmt.Errorf("candidate.command is required")
	}
	if d.Baseline.SnapshotHash == "" {
		return d, fmt.Errorf("baseline.snapshot_hash is required")
	}
	if d.Candidate.SnapshotHash == "" {
		return d, fmt.Errorf("candidate.snapshot_hash is required")
	}
	if d.Baseline.SkillDir == "" {
		return d, fmt.Errorf("baseline.skill_dir is required")
	}
	if d.Candidate.SkillDir == "" {
		return d, fmt.Errorf("candidate.skill_dir is required")
	}
	if d.Baseline.Harness == "" {
		d.Baseline.Harness = "claude_code"
	}
	if d.Candidate.Harness == "" {
		d.Candidate.Harness = "claude_code"
	}
	order := strings.ToLower(d.Ordering)
	if order == "" {
		order = "blocked"
	}
	if order != "blocked" && order != "random" {
		return d, fmt.Errorf("unsupported ordering %q", d.Ordering)
	}
	d.Ordering = order
	if d.AnalysisMethod == "" {
		d.AnalysisMethod = "difference"
	}
	if d.AnalysisMethod != "difference" && d.AnalysisMethod != "ratio" {
		return d, fmt.Errorf("unsupported analysis_method %q", d.AnalysisMethod)
	}
	if d.StoppingRule == "" {
		d.StoppingRule = "fixed"
	}
	if d.StoppingRule != "fixed" && d.StoppingRule != "threshold" {
		return d, fmt.Errorf("unsupported stopping_rule %q", d.StoppingRule)
	}
	return d, nil
}

// GeneratePlan turns a validated design into an executable sequence of runs.
func GeneratePlan(d ExperimentDesign) (ExperimentPlan, error) {
	d, err := NormalizeDesign(d)
	if err != nil {
		return ExperimentPlan{}, err
	}
	plan := ExperimentPlan{
		Schema:      ExperimentPlanSchema,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Design:      d,
	}
	outputDir := d.OutputDir
	if outputDir == "" {
		outputDir = "."
	}
	for _, task := range d.TaskFamilies {
		for r := 1; r <= d.Repetitions; r++ {
			basePath := fmt.Sprintf("%s/%s-r%d-baseline.json", outputDir, safeName(task), r)
			candPath := fmt.Sprintf("%s/%s-r%d-candidate.json", outputDir, safeName(task), r)
			baseStep := stepForCondition(d.Baseline, "baseline", task, r, basePath)
			candStep := stepForCondition(d.Candidate, "candidate", task, r, candPath)
			run := ExperimentRun{
				TaskFamily: task,
				Repetition: r,
				Comparison: fmt.Sprintf("%s/%s-r%d-comparison.json", outputDir, safeName(task), r),
			}
			run.Steps = []ExperimentStep{baseStep, candStep}
			plan.Runs = append(plan.Runs, run)
		}
	}
	if d.Ordering == "random" {
		plan.Notes = append(plan.Notes, "random ordering requested but not yet implemented; using blocked order")
	}
	return plan, nil
}

func stepForCondition(c Condition, condition, task string, r int, path string) ExperimentStep {
	cmd := os.ExpandEnv(expandPlaceholders(c.Command, task, r, path))
	return ExperimentStep{
		Condition:    condition,
		TaskFamily:   task,
		Repetition:   r,
		Command:      cmd,
		ProfilePath:  path,
		SnapshotHash: c.SnapshotHash,
		SkillDir:     c.SkillDir,
		Harness:      c.Harness,
		ExportFile:   c.ExportFile,
		OtelFile:     c.OtelFile,
	}
}

func expandPlaceholders(cmd, task string, r int, path string) string {
	cmd = strings.ReplaceAll(cmd, "$TASK", task)
	cmd = strings.ReplaceAll(cmd, "$REP", fmt.Sprintf("%d", r))
	cmd = strings.ReplaceAll(cmd, "$PROFILE", path)
	return cmd
}

func safeName(s string) string {
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	return s
}
