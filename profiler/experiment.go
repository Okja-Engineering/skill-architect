package profiler

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// An experiment is the thing that *makes* the pair the comparison reads: it
// declares two conditions, materializes the runs that produce a profile for
// each, executes them, and hands each pair to CompareProfiles.
//
// Because it makes the pair, it is also the only place that can refuse to make
// a bad one. Three rules, at three times:
//
//  1. The design refuses any declaration the runner cannot honour — a
//     randomised order nothing randomises, a ratio nothing computes, an early
//     stop nothing guards, two harnesses measured by different meters. Refused
//     before anything is executed, which is before anything is spent.
//  2. The run refuses a profile that is not the one the step said it would
//     produce: a step that wrote nothing, a leftover file from an earlier run,
//     another snapshot, another skill, another harness.
//  3. What is left is the comparison's own refusal. The two commands belong to
//     the caller and nothing here can promise they are the same build of the
//     profiler, so a pair read by two adapter versions is still possible — and
//     it is refused by CompareProfiles, upstream of every subtraction. That
//     rule is not restated here; there is one copy of it and this defers to it.

// The three documents an experiment passes through, each with its own schema
// because each is stored, read back, and outlives the run that produced it.
const (
	// ExperimentSchema identifies a design: what a human declared.
	ExperimentSchema = "skill-architect/experiment/v1"
	// ExperimentPlanSchema identifies a plan: the runs that design expands to.
	ExperimentPlanSchema = "skill-architect/experiment-plan/v1"
	// ExperimentResultSchema identifies a result: what running that plan found.
	ExperimentResultSchema = "skill-architect/experiment-result/v1"
)

// The single supported value of each declared field. Each is a singleton
// because the runner honours exactly one, and a design may declare only what
// the runner does.
const (
	orderingBlocked    = "blocked"
	analysisDifference = "difference"
	stoppingFixed      = "fixed"
)

// The placeholders a condition's command may use. The command is the caller's,
// and these are how the plan tells it which task, which repetition and — above
// all — which file to write.
const (
	placeholderTask       = "$TASK"
	placeholderRepetition = "$REP"
	placeholderProfile    = "$PROFILE"
)

// Condition is one side of a paired comparison.
//
// Command is run by a shell and is expected to write a profile to the path it
// is handed as $PROFILE. Harness, SnapshotHash and SkillDir are what the
// command is being asked to produce, and the run checks the profile it gets
// back against them — so they are a promise that is kept rather than metadata
// nobody reads.
type Condition struct {
	Name         string `json:"name"`
	Harness      string `json:"harness"`
	SnapshotHash string `json:"snapshot_hash"`
	SkillDir     string `json:"skill_dir"`
	Command      string `json:"command"`
}

// ExperimentDesign is the human-declared specification of a paired comparison:
// what is being compared, on which tasks, how many times, and how it will be
// judged. Declaring it before running is what keeps the result honest — and
// every field here is honoured or refused, never accepted and ignored.
type ExperimentDesign struct {
	Schema         string    `json:"schema"`
	Name           string    `json:"name"`
	TaskFamilies   []string  `json:"task_families"`
	Repetitions    int       `json:"repetitions"`
	Ordering       string    `json:"ordering"`
	AnalysisMethod string    `json:"analysis_method"`
	StoppingRule   string    `json:"stopping_rule"`
	Baseline       Condition `json:"baseline"`
	Candidate      Condition `json:"candidate"`
	OutputDir      string    `json:"output_dir,omitempty"`
}

// ExperimentStep is one execution: one command, producing one profile.
type ExperimentStep struct {
	Command      string `json:"command"`
	ProfilePath  string `json:"profile_path"`
	Harness      string `json:"harness"`
	SnapshotHash string `json:"snapshot_hash"`
	SkillDir     string `json:"skill_dir"`
}

// ExperimentRun is one repetition of one task family: a pair, not a list.
//
// The two sides are named fields rather than a slice of steps, so a run with
// one step — or three — cannot be written down at all. The alternative carries
// a length check into every consumer, and a plan read from disk that failed it
// would already have been half executed.
type ExperimentRun struct {
	TaskFamily string         `json:"task_family"`
	Repetition int            `json:"repetition"`
	Baseline   ExperimentStep `json:"baseline"`
	Candidate  ExperimentStep `json:"candidate"`
}

// ExperimentPlan is a design expanded into the runs that will be executed. It
// carries the normalized design, because a stored plan is the record of what
// was run and the defaults are part of that.
type ExperimentPlan struct {
	Schema      string           `json:"schema"`
	GeneratedAt string           `json:"generated_at"`
	Design      ExperimentDesign `json:"design"`
	Runs        []ExperimentRun  `json:"runs"`
}

// RunResult is one run's answer: which pair was compared, and the comparison.
type RunResult struct {
	TaskFamily       string           `json:"task_family"`
	Repetition       int              `json:"repetition"`
	BaselineProfile  string           `json:"baseline_profile"`
	CandidateProfile string           `json:"candidate_profile"`
	Comparison       ComparisonReport `json:"comparison"`
}

// ExperimentResult is what running a plan produced.
//
// Comparable is derived, never asserted: an experiment is comparable when it
// ran at least one run and every run of it compared something. The "at least
// one" is not defensive — "every run compared" is vacuously true of no runs,
// and an experiment that executed nothing would otherwise report itself a
// success.
type ExperimentResult struct {
	Schema      string           `json:"schema"`
	GeneratedAt string           `json:"generated_at"`
	Design      ExperimentDesign `json:"design"`
	Runs        []RunResult      `json:"runs"`
	Comparable  bool             `json:"comparable"`
}

func (r ExperimentResult) comparable() bool {
	if len(r.Runs) == 0 {
		return false
	}
	for _, run := range r.Runs {
		if !run.Comparison.Comparable {
			return false
		}
	}
	return true
}

// LoadExperimentDesign reads a design and normalizes it, so a caller cannot get
// an unchecked design by going through a file.
func LoadExperimentDesign(path string) (ExperimentDesign, error) {
	var d ExperimentDesign
	if err := readDocument(path, "experiment design", &d); err != nil {
		return ExperimentDesign{}, err
	}
	if d.Schema != ExperimentSchema {
		return ExperimentDesign{}, fmt.Errorf("read experiment design %s: schema is %q, want %q", path, d.Schema, ExperimentSchema)
	}
	d, err := NormalizeDesign(d)
	if err != nil {
		return ExperimentDesign{}, fmt.Errorf("read experiment design %s: %w", path, err)
	}
	return d, nil
}

// LoadPlan reads a materialized plan and refuses one that cannot be executed as
// written.
func LoadPlan(path string) (ExperimentPlan, error) {
	var p ExperimentPlan
	if err := readDocument(path, "experiment plan", &p); err != nil {
		return ExperimentPlan{}, err
	}
	if p.Schema != ExperimentPlanSchema {
		return ExperimentPlan{}, fmt.Errorf("read experiment plan %s: schema is %q, want %q", path, p.Schema, ExperimentPlanSchema)
	}
	if err := validatePlan(p); err != nil {
		return ExperimentPlan{}, fmt.Errorf("read experiment plan %s: %w", path, err)
	}
	return p, nil
}

// readDocument reads one JSON document, naming the path in every error. A
// caller holding three files needs to know which one it was.
func readDocument(path, what string, into any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s %s: %w", what, path, err)
	}
	if err := json.Unmarshal(data, into); err != nil {
		return fmt.Errorf("read %s %s: %w", what, path, err)
	}
	return nil
}

// validatePlan refuses a plan that would run to a meaningless answer.
//
// A plan with no runs is the one that matters: it executes nothing, and
// "every run compared something" is true of no runs, so the experiment would
// report itself entirely comparable having measured nothing at all.
func validatePlan(p ExperimentPlan) error {
	if len(p.Runs) == 0 {
		return fmt.Errorf("the plan has no runs: it would execute nothing and report an experiment in which everything was comparable")
	}
	for _, run := range p.Runs {
		where := fmt.Sprintf("%s r%d", run.TaskFamily, run.Repetition)
		for _, side := range []struct {
			label string
			step  ExperimentStep
		}{{"baseline", run.Baseline}, {"candidate", run.Candidate}} {
			if side.step.Command == "" {
				return fmt.Errorf("%s: the %s step has no command", where, side.label)
			}
			if side.step.ProfilePath == "" {
				return fmt.Errorf("%s: the %s step has no profile path, so nothing says where its profile goes", where, side.label)
			}
		}
		if run.Baseline.ProfilePath == run.Candidate.ProfilePath {
			return fmt.Errorf("%s: both steps write the same profile path %q, so the second would overwrite the first and the comparison would subtract a profile from itself",
				where, run.Baseline.ProfilePath)
		}
	}
	return nil
}

// NormalizeDesign applies the defaults and then refuses anything the runner
// cannot honour. Every entry point to a design goes through it.
func NormalizeDesign(d ExperimentDesign) (ExperimentDesign, error) {
	d = designDefaults(d)
	if err := designRefusal(d); err != nil {
		return d, err
	}
	return d, nil
}

// designDefaults fills in what a design need not say. A default is only
// legitimate where there is one thing the design could have meant.
func designDefaults(d ExperimentDesign) ExperimentDesign {
	if d.Repetitions <= 0 {
		d.Repetitions = 1
	}
	d.Ordering = orDefault(d.Ordering, orderingBlocked)
	d.AnalysisMethod = orDefault(d.AnalysisMethod, analysisDifference)
	d.StoppingRule = orDefault(d.StoppingRule, stoppingFixed)
	harness := defaultHarness(HarnessNames())
	d.Baseline.Harness = orDefault(d.Baseline.Harness, harness)
	d.Candidate.Harness = orDefault(d.Candidate.Harness, harness)
	return d
}

// orDefault lowercases a declared value and supplies the default for an empty
// one. Case is not a distinction any of these fields makes.
func orDefault(declared, fallback string) string {
	if declared == "" {
		return fallback
	}
	return strings.ToLower(declared)
}

// defaultHarness is the harness a design need not name, derived from the
// registry rather than written down.
//
// It exists only while the answer is unambiguous. With one adapter registered
// there is nothing else the design could have meant; with two, guessing would
// pick the meter for the maintainer, and the design has to say which.
func defaultHarness(registered []string) string {
	if len(registered) != 1 {
		return ""
	}
	return registered[0]
}

// designRefusal is the whole of the design-level rule set, in one place and in
// one shape, so a new rule is a row rather than another branch in a function
// that grows with every release.
//
// The three declaration fields — ordering, analysis method, stopping rule —
// share one principle: a design may declare only what the runner actually does.
// Running a design that asked for a randomised order in blocked order, or for a
// ratio by subtraction, produces a document that is the record of an experiment
// nobody ran.
func designRefusal(d ExperimentDesign) error {
	for _, rule := range []struct {
		broken bool
		reason string
	}{
		{d.Name == "", "name is required: an experiment nobody named cannot be referred to afterwards"},
		{len(d.TaskFamilies) == 0, "task_families is required: an experiment with no task has nothing to measure"},
		{hasEmpty(d.TaskFamilies), "task_families contains an empty name, and a task nobody named cannot be reported"},

		{d.Baseline.Command == "", "baseline.command is required: nothing would produce the baseline profile"},
		{d.Candidate.Command == "", "candidate.command is required: nothing would produce the candidate profile"},
		{!strings.Contains(d.Baseline.Command, placeholderProfile),
			"baseline.command does not mention " + placeholderProfile + ": the plan chooses where each profile goes, and a command that ignores it writes somewhere nothing reads"},
		{!strings.Contains(d.Candidate.Command, placeholderProfile),
			"candidate.command does not mention " + placeholderProfile + ": the plan chooses where each profile goes, and a command that ignores it writes somewhere nothing reads"},

		{d.Baseline.SnapshotHash == "", "baseline.snapshot_hash is required: a result that cannot say which revision it measured is not recomputable"},
		{d.Candidate.SnapshotHash == "", "candidate.snapshot_hash is required: a result that cannot say which revision it measured is not recomputable"},
		{d.Baseline.SkillDir == "", "baseline.skill_dir is required: nothing would say which skill was profiled"},
		{d.Candidate.SkillDir == "", "candidate.skill_dir is required: nothing would say which skill was profiled"},

		// Before the registration check, because it is the reason that survives
		// the day a second adapter is registered and both names become valid.
		{d.Baseline.Harness != d.Candidate.Harness, fmt.Sprintf(
			"the two conditions must name the same harness: baseline %q vs candidate %q — two harnesses measure with different meters, so a difference between them is not a difference in the skill",
			d.Baseline.Harness, d.Candidate.Harness)},
		{!isRegisteredHarness(d.Baseline.Harness), fmt.Sprintf(
			"unknown harness %q (supported: %s): nothing here can profile it",
			d.Baseline.Harness, SupportedHarnesses())},

		{d.Ordering != orderingBlocked, fmt.Sprintf(
			"unsupported ordering %q: only %q is implemented, and a design run in an order it did not declare is a different experiment",
			d.Ordering, orderingBlocked)},
		{d.AnalysisMethod != analysisDifference, fmt.Sprintf(
			"unsupported analysis_method %q: only %q is implemented — the comparison subtracts, and nothing here computes a ratio",
			d.AnalysisMethod, analysisDifference)},
		// "threshold" — peek and stop early — needs alpha-spending or a minimum
		// N to stay honest. Until that machinery exists, accepting the word
		// would permit peeking and call it a stopping rule.
		{d.StoppingRule != stoppingFixed, fmt.Sprintf(
			"unsupported stopping_rule %q: only %q is implemented — stopping early needs alpha-spending guards that do not exist here, and without them it is peeking",
			d.StoppingRule, stoppingFixed)},
	} {
		if rule.broken {
			return fmt.Errorf("%s", rule.reason)
		}
	}
	return nil
}

func hasEmpty(names []string) bool {
	for _, n := range names {
		if n == "" {
			return true
		}
	}
	return false
}

// isRegisteredHarness asks the registry, so the set of harnesses a design may
// name and the set the CLI can dispatch cannot come to disagree.
func isRegisteredHarness(harness string) bool {
	for _, name := range HarnessNames() {
		if name == harness {
			return true
		}
	}
	return false
}

// GeneratePlan expands a design into the runs that will be executed: one run
// per repetition of each task family, blocked — every repetition of one task
// family before the next.
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
	for _, task := range d.TaskFamilies {
		for rep := 1; rep <= d.Repetitions; rep++ {
			plan.Runs = append(plan.Runs, ExperimentRun{
				TaskFamily: task,
				Repetition: rep,
				Baseline:   step(d.Baseline, d.OutputDir, "baseline", task, rep),
				Candidate:  step(d.Candidate, d.OutputDir, "candidate", task, rep),
			})
		}
	}
	return plan, nil
}

// step materializes one condition's command for one task and repetition.
//
// The placeholders are substituted here and the result is *not* passed through
// os.ExpandEnv. A plan is written down, kept, and run later — possibly on
// another machine — so expanding the planning machine's environment into it
// would bake one machine's values into the record, and an unset variable would
// become an empty string at plan time rather than being the shell's problem at
// run time. The shell that runs the step expands what is left.
func step(c Condition, outputDir, label, task string, rep int) ExperimentStep {
	path := filepath.Join(outputDir, fmt.Sprintf("%s-r%d-%s.json", safeName(task), rep, label))
	command := strings.NewReplacer(
		placeholderTask, task,
		placeholderRepetition, fmt.Sprintf("%d", rep),
		placeholderProfile, path,
	).Replace(c.Command)
	return ExperimentStep{
		Command:      command,
		ProfilePath:  path,
		Harness:      c.Harness,
		SnapshotHash: c.SnapshotHash,
		SkillDir:     c.SkillDir,
	}
}

// safeName turns a task family — a label a human typed — into one path
// component.
//
// It keeps letters, digits, '-' and '_' and replaces everything else. The
// separators and the dot are replaced rather than escaped, so no combination of
// them can climb out of the output directory: there is no '.' left to make a
// '..' out of, and no separator left to start a new component.
func safeName(s string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, s)
	if safe == "" {
		return "task"
	}
	return safe
}

// RunPlan executes every run in the plan and compares each pair.
//
// A refused comparison is a result, not an error: every run is an independent
// observation, and one pair the comparator will not subtract does not discard
// the runs that worked. A step that fails, or that produces a profile the plan
// did not ask for, *is* an error and stops the experiment — the setup is wrong,
// and every run after it would be wrong the same way.
func RunPlan(plan ExperimentPlan) (ExperimentResult, error) {
	result := ExperimentResult{
		Schema:      ExperimentResultSchema,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Design:      plan.Design,
	}
	for _, run := range plan.Runs {
		baseline, err := produceProfile(run, "baseline", run.Baseline)
		if err != nil {
			return ExperimentResult{}, err
		}
		candidate, err := produceProfile(run, "candidate", run.Candidate)
		if err != nil {
			return ExperimentResult{}, err
		}
		result.Runs = append(result.Runs, RunResult{
			TaskFamily:       run.TaskFamily,
			Repetition:       run.Repetition,
			BaselineProfile:  run.Baseline.ProfilePath,
			CandidateProfile: run.Candidate.ProfilePath,
			// The comparison is the comparator's, including its refusals. The
			// experiment subtracts nothing itself.
			Comparison: CompareProfiles(baseline, candidate),
		})
	}
	result.Comparable = result.comparable()
	return result, nil
}

// produceProfile runs one step and returns the profile it produced, having
// checked that it is the profile the step said it would produce.
func produceProfile(run ExperimentRun, label string, s ExperimentStep) (Profile, error) {
	where := fmt.Sprintf("the %s step of %s r%d", label, run.TaskFamily, run.Repetition)
	if err := executeStep(s); err != nil {
		return Profile{}, fmt.Errorf("%s: %w", where, err)
	}
	p, err := LoadProfile(s.ProfilePath)
	if err != nil {
		return Profile{}, fmt.Errorf("%s: %w", where, err)
	}
	if err := profileMatchesStep(p, s); err != nil {
		return Profile{}, fmt.Errorf("%s produced %s: %w", where, s.ProfilePath, err)
	}
	return p, nil
}

// executeStep runs one command and insists that it wrote the profile it was
// given.
//
// The file is stamped before and after. A command that exits 0 and writes
// nothing is the trap: the path is then either empty — or, worse, holds a
// profile from an earlier run, which would be compared and reported as this
// run's numbers.
//
// The command's own stdout goes to stderr, because stdout here belongs to the
// result document a wrapper parses.
func executeStep(s ExperimentStep) error {
	if err := os.MkdirAll(filepath.Dir(s.ProfilePath), 0o755); err != nil {
		return fmt.Errorf("the directory for %s could not be created: %w", s.ProfilePath, err)
	}
	before := stamp(s.ProfilePath)

	cmd := exec.Command("sh", "-c", s.Command)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command %q failed: %w", s.Command, err)
	}

	after := stamp(s.ProfilePath)
	if !after.exists || after == before {
		return fmt.Errorf("the command exited 0 and wrote no profile at %s: a step must write the profile it is handed as %s, or the file left there is an earlier run's",
			s.ProfilePath, placeholderProfile)
	}
	return nil
}

// fileStamp is what a file looks like from outside. Comparing the stamp before
// and after is what distinguishes "this step wrote it" from "it was already
// there", without deleting anything the caller may have meant to keep.
type fileStamp struct {
	exists  bool
	size    int64
	modTime time.Time
}

func stamp(path string) fileStamp {
	info, err := os.Stat(path)
	if err != nil {
		return fileStamp{}
	}
	return fileStamp{exists: true, size: info.Size(), modTime: info.ModTime()}
}

// profileMatchesStep holds the profile to what the step declared. The plan says
// which harness, revision and skill the command was asked to profile; a profile
// that disagrees is the wrong file, and subtracting it would answer a question
// nobody asked.
func profileMatchesStep(p Profile, s ExperimentStep) error {
	for _, field := range []struct{ name, declared, got string }{
		{"harness", s.Harness, p.Harness},
		{"snapshot_hash", s.SnapshotHash, p.SnapshotHash},
		{"skill_dir", s.SkillDir, p.SkillDir},
	} {
		if field.declared != field.got {
			return fmt.Errorf("its %s is %q and the plan declared %q", field.name, field.got, field.declared)
		}
	}
	return nil
}
