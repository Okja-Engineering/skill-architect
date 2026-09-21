package profiler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The experiment is the thing that *manufactures* the pair `compare` reads, so
// every rule here exists to stop it manufacturing a pair whose comparison
// cannot be trusted. Two kinds of rule, at two times:
//
//   - the design refuses a declaration the runner cannot honour, before
//     anything is executed and before any money is spent;
//   - the run refuses a profile that is not the one the step said it would
//     produce, before the two are ever subtracted.
//
// What is left — two profiles read by different adapter versions, because the
// two commands are the caller's and may not be the same binary — is `compare`'s
// refusal, reached through CompareProfiles rather than restated here. There is
// one copy of that rule and this is not it.

// validDesign is the control every refusal below is measured against: a design
// that passes. Without it, a NormalizeDesign that refused everything would
// satisfy every refusal case in this file.
func validDesign() ExperimentDesign {
	return ExperimentDesign{
		Schema:       ExperimentSchema,
		Name:         "skill-rewrite-efficiency",
		TaskFamilies: []string{"refactor-audit", "refactor-rewrite"},
		Repetitions:  2,
		Baseline: Condition{
			Name:         "no-skill",
			Harness:      "claude_code",
			Command:      "./capture-baseline $TASK $REP $PROFILE",
			SnapshotHash: "sha-base",
			SkillDir:     "skills/skill-audit",
		},
		Candidate: Condition{
			Name:         "with-skill",
			Harness:      "claude_code",
			Command:      "./capture-candidate $TASK $REP $PROFILE",
			SnapshotHash: "sha-cand",
			SkillDir:     "skills/skill-audit",
		},
	}
}

func TestNormalizeDesign_TheControlPasses(t *testing.T) {
	if _, err := NormalizeDesign(validDesign()); err != nil {
		t.Fatalf("the control design was refused: %v — every refusal case in this file would pass vacuously", err)
	}
}

func TestNormalizeDesign_AppliesTheDefaultsAndSaysSo(t *testing.T) {
	d := validDesign()
	d.Repetitions = 0
	d.Ordering = ""
	d.AnalysisMethod = ""
	d.StoppingRule = ""
	d.Baseline.Harness = ""
	d.Candidate.Harness = ""

	n, err := NormalizeDesign(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ field, got, want string }{
		{"ordering", n.Ordering, "blocked"},
		{"analysis_method", n.AnalysisMethod, "difference"},
		{"stopping_rule", n.StoppingRule, "fixed"},
		{"baseline.harness", n.Baseline.Harness, "claude_code"},
		{"candidate.harness", n.Candidate.Harness, "claude_code"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.field, tc.got, tc.want)
		}
	}
	if n.Repetitions != 1 {
		t.Errorf("repetitions = %d, want 1 — an experiment with no repetition is not an experiment", n.Repetitions)
	}
}

// A declaration is normalized case-insensitively before it is judged, so
// "Blocked" is the ordering it obviously means rather than an unsupported one.
func TestNormalizeDesign_JudgesADeclarationCaseInsensitively(t *testing.T) {
	d := validDesign()
	d.Ordering = "BLOCKED"
	d.AnalysisMethod = "Difference"
	d.StoppingRule = "FIXED"
	n, err := NormalizeDesign(d)
	if err != nil {
		t.Fatalf("a design declaring the supported values in capitals was refused: %v", err)
	}
	if n.Ordering != "blocked" || n.AnalysisMethod != "difference" || n.StoppingRule != "fixed" {
		t.Errorf("normalized to %q/%q/%q, want the lowercase spellings", n.Ordering, n.AnalysisMethod, n.StoppingRule)
	}
}

// The rule the three "declaration" fields share: the design may declare only
// what the runner actually does. A design that asks for random ordering, a
// ratio analysis or an early stop and is then run blocked, by subtraction, to a
// fixed N is a *different experiment* from the one somebody wrote down — and
// the document would be the record of it. Accepting a request and ignoring it
// is the failure mode this repo has already found three times in its own prose.
func TestNormalizeDesign_RefusesEveryDeclarationTheRunnerCannotHonour(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ExperimentDesign)
		wants  []string
	}{
		{"a name nobody gave it", func(d *ExperimentDesign) { d.Name = "" }, []string{"name"}},
		{"no task family", func(d *ExperimentDesign) { d.TaskFamilies = nil }, []string{"task_families"}},
		{"an empty task family", func(d *ExperimentDesign) { d.TaskFamilies = []string{"audit", ""} }, []string{"task_families"}},
		{"no baseline command", func(d *ExperimentDesign) { d.Baseline.Command = "" }, []string{"baseline", "command"}},
		{"no candidate command", func(d *ExperimentDesign) { d.Candidate.Command = "" }, []string{"candidate", "command"}},
		{"a baseline command that ignores the profile path", func(d *ExperimentDesign) { d.Baseline.Command = "./capture" }, []string{"baseline", "$PROFILE"}},
		{"a candidate command that ignores the profile path", func(d *ExperimentDesign) { d.Candidate.Command = "./capture" }, []string{"candidate", "$PROFILE"}},
		{"no baseline snapshot", func(d *ExperimentDesign) { d.Baseline.SnapshotHash = "" }, []string{"baseline", "snapshot_hash"}},
		{"no candidate snapshot", func(d *ExperimentDesign) { d.Candidate.SnapshotHash = "" }, []string{"candidate", "snapshot_hash"}},
		{"no baseline skill dir", func(d *ExperimentDesign) { d.Baseline.SkillDir = "" }, []string{"baseline", "skill_dir"}},
		{"no candidate skill dir", func(d *ExperimentDesign) { d.Candidate.SkillDir = "" }, []string{"candidate", "skill_dir"}},
		{"randomised ordering, which nothing randomises", func(d *ExperimentDesign) { d.Ordering = "random" }, []string{"ordering", "random", "blocked"}},
		{"a ratio analysis, which nothing computes", func(d *ExperimentDesign) { d.AnalysisMethod = "ratio" }, []string{"analysis_method", "ratio", "difference"}},
		{"stopping early, which nothing guards", func(d *ExperimentDesign) { d.StoppingRule = "threshold" }, []string{"stopping_rule", "threshold", "fixed"}},
		{"a harness no adapter answers to", func(d *ExperimentDesign) {
			d.Baseline.Harness, d.Candidate.Harness = "hal9000", "hal9000"
		}, []string{"harness", "hal9000"}},
		{"two harnesses", func(d *ExperimentDesign) { d.Candidate.Harness = "cursor" }, []string{"harness", "claude_code", "cursor"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := validDesign()
			tc.mutate(&d)
			_, err := NormalizeDesign(d)
			if err == nil {
				t.Fatalf("the design was accepted: %s", tc.name)
			}
			for _, want := range tc.wants {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("refusal = %q, want it to name %q", err, want)
				}
			}
		})
	}
}

// Comparing two harnesses is refused where the pair is *made*, not where it is
// read. `compare` treats a differing harness as a note, because a human handing
// it two profiles may have a reason; the experiment is the thing that decides
// what to capture, so it may not decide to capture two things that were
// measured by different meters. This is the one place that rule lives, and it
// is the reason a run can never produce a cross-harness pair.
func TestNormalizeDesign_RefusesACrossHarnessPairBeforeItIsEverRun(t *testing.T) {
	d := validDesign()
	d.Candidate.Harness = "cursor"
	_, err := NormalizeDesign(d)
	if err == nil {
		t.Fatal("a design pairing two harnesses was accepted")
	}
	if !strings.Contains(err.Error(), "same harness") {
		t.Errorf("refusal = %q, want it to say the two sides must name the same harness", err)
	}
}

// An A/A design — the same condition on both sides — is a noise-floor
// measurement and a legitimate experiment. Refusing it would be the tool
// deciding what question the maintainer is allowed to ask.
func TestNormalizeDesign_AcceptsAnAADesign(t *testing.T) {
	d := validDesign()
	d.Candidate = d.Baseline
	if _, err := NormalizeDesign(d); err != nil {
		t.Fatalf("an A/A design was refused: %v — measuring the noise floor is an experiment", err)
	}
}

// The harness default is derived from the registry rather than written down. It
// exists only while the answer is unambiguous: with one adapter registered
// there is nothing else the design could have meant, and with two there is.
// Tested through the helper because the second case is unreachable through
// NormalizeDesign while one adapter ships, and an unreachable branch that is
// never asserted is a branch nobody has checked.
func TestDefaultHarness_OnlyDefaultsWhileTheAnswerIsUnambiguous(t *testing.T) {
	if got := defaultHarness([]string{"claude_code"}); got != "claude_code" {
		t.Errorf("defaultHarness with one registered = %q, want claude_code", got)
	}
	if got := defaultHarness([]string{"claude_code", "cursor"}); got != "" {
		t.Errorf("defaultHarness with two registered = %q, want no default — the design has to say which", got)
	}
	if got := defaultHarness(nil); got != "" {
		t.Errorf("defaultHarness with none registered = %q, want no default", got)
	}
	// The production registry is what NormalizeDesign actually asks, and the
	// defaults test above depends on it having exactly one entry today.
	if names := HarnessNames(); len(names) != 1 {
		t.Logf("the registry now has %d harnesses (%v) — the design must name one explicitly", len(names), names)
	}
}

// --- The plan ---

func TestGeneratePlan_MaterializesOneRunPerTaskFamilyPerRepetition(t *testing.T) {
	d := validDesign()
	plan, err := GeneratePlan(d)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Schema != ExperimentPlanSchema {
		t.Errorf("schema = %q, want %q", plan.Schema, ExperimentPlanSchema)
	}
	if plan.GeneratedAt == "" {
		t.Error("the plan does not say when it was generated")
	}
	want := len(d.TaskFamilies) * d.Repetitions
	if len(plan.Runs) != want {
		t.Fatalf("runs = %d, want %d", len(plan.Runs), want)
	}
	// The plan embeds the design it was generated from, normalized: a stored
	// plan is the record of what was run, and the defaults are part of that.
	if plan.Design.Ordering != "blocked" || plan.Design.StoppingRule != "fixed" {
		t.Errorf("the plan embeds design %+v, want the normalized one", plan.Design)
	}
	seen := map[string]string{}
	for _, run := range plan.Runs {
		if run.TaskFamily == "" || run.Repetition < 1 {
			t.Errorf("run %+v does not say which task family and repetition it is", run)
		}
		for label, step := range map[string]ExperimentStep{"baseline": run.Baseline, "candidate": run.Candidate} {
			if step.Command == "" || step.ProfilePath == "" {
				t.Errorf("%s step of %s r%d is empty: %+v", label, run.TaskFamily, run.Repetition, step)
			}
			// Two steps writing one path means the second overwrites the first
			// and the comparison subtracts a profile from itself.
			if prev, dup := seen[step.ProfilePath]; dup {
				t.Errorf("%s step of %s r%d writes %q, which %s already writes",
					label, run.TaskFamily, run.Repetition, step.ProfilePath, prev)
			}
			seen[step.ProfilePath] = fmt.Sprintf("%s of %s r%d", label, run.TaskFamily, run.Repetition)
		}
	}
	if len(seen) != 2*want {
		t.Errorf("the plan names %d profile paths, want %d — two per run", len(seen), 2*want)
	}
}

// Each step carries what the condition declared, because the run checks the
// profile it gets back against it. A step that did not carry the declaration
// would have nothing to check against.
func TestGeneratePlan_EveryStepCarriesWhatItsConditionDeclared(t *testing.T) {
	d := validDesign()
	plan, err := GeneratePlan(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, run := range plan.Runs {
		for _, tc := range []struct {
			label string
			step  ExperimentStep
			cond  Condition
		}{
			{"baseline", run.Baseline, d.Baseline},
			{"candidate", run.Candidate, d.Candidate},
		} {
			if tc.step.Harness != tc.cond.Harness {
				t.Errorf("%s step harness = %q, want %q", tc.label, tc.step.Harness, tc.cond.Harness)
			}
			if tc.step.SnapshotHash != tc.cond.SnapshotHash {
				t.Errorf("%s step snapshot = %q, want %q", tc.label, tc.step.SnapshotHash, tc.cond.SnapshotHash)
			}
			if tc.step.SkillDir != tc.cond.SkillDir {
				t.Errorf("%s step skill dir = %q, want %q", tc.label, tc.step.SkillDir, tc.cond.SkillDir)
			}
		}
	}
}

func TestGeneratePlan_ExpandsThePlaceholdersItDocuments(t *testing.T) {
	d := validDesign()
	d.TaskFamilies = []string{"audit"}
	d.Repetitions = 1
	d.OutputDir = "results"
	d.Baseline.Command = "capture --task $TASK --rep $REP --out $PROFILE"
	plan, err := GeneratePlan(d)
	if err != nil {
		t.Fatal(err)
	}
	step := plan.Runs[0].Baseline
	want := "capture --task audit --rep 1 --out " + step.ProfilePath
	if step.Command != want {
		t.Errorf("command = %q, want %q", step.Command, want)
	}
	if !strings.Contains(step.ProfilePath, "results") {
		t.Errorf("profile path = %q, want it under the design's output_dir", step.ProfilePath)
	}
}

// The plan is a document that is written down, kept, and run later, possibly
// elsewhere. Expanding the *planning* machine's environment into it would bake
// one machine's values into the record — and an unset variable would become an
// empty string at plan time instead of being the shell's problem at run time,
// which is how `--session $SESSION_ID` silently becomes `--session`.
func TestGeneratePlan_LeavesEnvironmentVariablesForTheShellThatRunsIt(t *testing.T) {
	t.Setenv("SKILL_ARCHITECT_S6_PROBE", "expanded-at-plan-time")
	d := validDesign()
	d.TaskFamilies = []string{"audit"}
	d.Repetitions = 1
	d.Baseline.Command = "capture --token $SKILL_ARCHITECT_S6_PROBE --out $PROFILE"
	plan, err := GeneratePlan(d)
	if err != nil {
		t.Fatal(err)
	}
	cmd := plan.Runs[0].Baseline.Command
	if !strings.Contains(cmd, "$SKILL_ARCHITECT_S6_PROBE") {
		t.Errorf("command = %q, want the variable left for the shell", cmd)
	}
	if strings.Contains(cmd, "expanded-at-plan-time") {
		t.Errorf("command = %q, want the planning machine's environment kept out of the stored plan", cmd)
	}
}

// A task family is a label a human typed, and it becomes part of a file path.
// It must not be able to name a file outside the output directory.
func TestGeneratePlan_ATaskFamilyCannotEscapeTheOutputDirectory(t *testing.T) {
	d := validDesign()
	d.Repetitions = 1
	d.OutputDir = "results"
	d.TaskFamilies = []string{"../../etc/passwd", "a b/c", "..", "with/slash"}
	plan, err := GeneratePlan(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Runs) != len(d.TaskFamilies) {
		t.Fatalf("runs = %d, want %d", len(plan.Runs), len(d.TaskFamilies))
	}
	for _, run := range plan.Runs {
		for _, step := range []ExperimentStep{run.Baseline, run.Candidate} {
			dir := filepath.Dir(filepath.Clean(step.ProfilePath))
			if dir != "results" {
				t.Errorf("task %q produced profile path %q, which is outside the output directory",
					run.TaskFamily, step.ProfilePath)
			}
		}
	}
}

// safeName's empty case is not reachable through a design — a design with an
// empty task family is refused — so it is asserted here rather than left as a
// branch nobody has checked. A name that mapped to nothing would put a path
// component of "" into the plan and two task families would collide there.
func TestSafeName_NeverProducesAnEmptyPathComponent(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"", "task"},
		{"..", "--"},
		{"refactor audit", "refactor-audit"},
		{"a/b", "a-b"},
		{"keeps_Digits-09", "keeps_Digits-09"},
	} {
		if got := safeName(tc.in); got != tc.want {
			t.Errorf("safeName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestGeneratePlan_RefusesTheDesignsNormalizeRefuses(t *testing.T) {
	d := validDesign()
	d.StoppingRule = "threshold"
	if _, err := GeneratePlan(d); err == nil {
		t.Fatal("a plan was generated from a design NormalizeDesign refuses")
	}
}

// --- Reading the two documents back ---

func writeJSON(t *testing.T, dir, name string, v any) string {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestLoadExperimentDesign(t *testing.T) {
	dir := t.TempDir()
	good := writeJSON(t, dir, "design.json", validDesign())

	loaded, err := LoadExperimentDesign(good)
	if err != nil {
		t.Fatalf("a valid design was refused: %v", err)
	}
	if loaded.Name != validDesign().Name {
		t.Errorf("name = %q, want %q", loaded.Name, validDesign().Name)
	}
	// Reading a design normalizes it, so a caller cannot get an un-defaulted
	// one by going through a file.
	if loaded.Ordering != "blocked" {
		t.Errorf("ordering = %q, want the default applied on load", loaded.Ordering)
	}

	otherSchema := validDesign()
	otherSchema.Schema = "skill-architect/experiment/v2"
	noSchema := validDesign()
	noSchema.Schema = ""
	refused := validDesign()
	refused.StoppingRule = "threshold"

	notJSON := filepath.Join(dir, "not-json.json")
	if err := os.WriteFile(notJSON, []byte("this is not a design"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, path, want string
	}{
		{"a file that is not there", filepath.Join(dir, "absent.json"), "absent.json"},
		{"a document that is not JSON", notJSON, "not-json.json"},
		{"another schema", writeJSON(t, dir, "v2.json", otherSchema), ExperimentSchema},
		{"no schema at all", writeJSON(t, dir, "none.json", noSchema), ExperimentSchema},
		{"a design the rules refuse", writeJSON(t, dir, "refused.json", refused), "stopping_rule"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadExperimentDesign(tc.path)
			if err == nil {
				t.Fatalf("accepted: %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to name %q", err, tc.want)
			}
		})
	}
}

func TestLoadPlan(t *testing.T) {
	dir := t.TempDir()
	plan, err := GeneratePlan(validDesign())
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadPlan(writeJSON(t, dir, "plan.json", plan))
	if err != nil {
		t.Fatalf("a generated plan was refused on reload: %v", err)
	}
	if len(loaded.Runs) != len(plan.Runs) {
		t.Errorf("runs = %d, want %d", len(loaded.Runs), len(plan.Runs))
	}

	empty := plan
	empty.Runs = nil
	otherSchema := plan
	otherSchema.Schema = ComparisonSchema
	noCommand := clonePlan(plan)
	noCommand.Runs[0].Candidate.Command = ""
	noPath := clonePlan(plan)
	noPath.Runs[0].Baseline.ProfilePath = ""
	onePath := clonePlan(plan)
	onePath.Runs[0].Candidate.ProfilePath = onePath.Runs[0].Baseline.ProfilePath

	notJSON := filepath.Join(dir, "not-a-plan.json")
	if err := os.WriteFile(notJSON, []byte("{["), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, path, want string
	}{
		{"a file that is not there", filepath.Join(dir, "absent.json"), "absent.json"},
		{"a document that is not JSON", notJSON, "not-a-plan.json"},
		{"another schema", writeJSON(t, dir, "other.json", otherSchema), ExperimentPlanSchema},
		// A plan with no runs would run nothing and then report itself as an
		// experiment in which everything was comparable, because "every run
		// compared something" is vacuously true of no runs.
		{"a plan with no runs", writeJSON(t, dir, "empty.json", empty), "no runs"},
		{"a step with no command", writeJSON(t, dir, "nocmd.json", noCommand), "command"},
		{"a step with no profile path", writeJSON(t, dir, "nopath.json", noPath), "profile path"},
		{"two steps writing one path", writeJSON(t, dir, "onepath.json", onePath), "same profile path"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadPlan(tc.path)
			if err == nil {
				t.Fatalf("accepted: %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to name %q", err, tc.want)
			}
		})
	}
}

func clonePlan(p ExperimentPlan) ExperimentPlan {
	runs := make([]ExperimentRun, len(p.Runs))
	copy(runs, p.Runs)
	p.Runs = runs
	return p
}

// --- Running it ---

// experimentProfile is a profile as a capture command would leave it on disk.
// The adapter version and the identity fields are parameters because they are
// what the run checks.
func experimentProfile(adapterVersion, harness, snapshot, skillDir string, input int) Profile {
	return Profile{
		Schema:       ProfileSchema,
		ProfiledAt:   "2026-09-20T10:00:00Z",
		Harness:      harness,
		SessionID:    "session-" + snapshot,
		SnapshotHash: snapshot,
		SkillDir:     skillDir,
		Capability: CapabilityReport{
			Harness:    harness,
			AdapterVer: adapterVersion,
			ProbedAt:   "2026-09-20T10:00:00Z",
		},
		Tokens: PresentTokenResult(TokenCounts{
			Input:  Count(input),
			Output: Count(20),
		}, string(SourceOtel)),
		ToolCalls:       UnknownToolCallResult("no tool call events in export"),
		SkillActivation: UnknownActivationResult("no skill activation events in export"),
		Timing:          UnknownTimingResult("no timing events in export"),
		Attribution:     UnknownAttributionResult("this harness does not attribute outputs to skills"),
	}
}

// runnableDesign is a design whose two commands copy prepared profiles into
// place: the shape of a real experiment, with the capture replaced by something
// this test can predict.
func runnableDesign(t *testing.T, dir string, baseline, candidate Profile) ExperimentDesign {
	t.Helper()
	basePath := writeJSON(t, dir, "baseline-fixture.json", baseline)
	candPath := writeJSON(t, dir, "candidate-fixture.json", candidate)
	d := validDesign()
	d.TaskFamilies = []string{"audit"}
	d.Repetitions = 1
	d.OutputDir = dir
	d.Baseline.Harness, d.Candidate.Harness = baseline.Harness, candidate.Harness
	d.Baseline.SnapshotHash, d.Candidate.SnapshotHash = baseline.SnapshotHash, candidate.SnapshotHash
	d.Baseline.SkillDir, d.Candidate.SkillDir = baseline.SkillDir, candidate.SkillDir
	d.Baseline.Command = fmt.Sprintf("cp %q $PROFILE", basePath)
	d.Candidate.Command = fmt.Sprintf("cp %q $PROFILE", candPath)
	return d
}

func runDesign(t *testing.T, d ExperimentDesign) (ExperimentResult, error) {
	t.Helper()
	plan, err := GeneratePlan(d)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	return RunPlan(plan)
}

func TestRunPlan_ExecutesThePairAndComparesWhatItProduced(t *testing.T) {
	dir := t.TempDir()
	d := runnableDesign(t, dir,
		experimentProfile(AdapterVersion, "claude_code", "sha-base", "skills/skill-audit", 1000),
		experimentProfile(AdapterVersion, "claude_code", "sha-cand", "skills/skill-audit", 600))

	result, err := runDesign(t, d)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Schema != ExperimentResultSchema {
		t.Errorf("schema = %q, want %q", result.Schema, ExperimentResultSchema)
	}
	if len(result.Runs) != 1 {
		t.Fatalf("runs = %d, want 1", len(result.Runs))
	}
	if !result.Comparable {
		t.Fatalf("the experiment reports nothing comparable: %+v", result.Runs[0].Comparison)
	}
	run := result.Runs[0]
	if run.TaskFamily != "audit" || run.Repetition != 1 {
		t.Errorf("run identity = %q r%d, want audit r1", run.TaskFamily, run.Repetition)
	}
	if run.BaselineProfile == "" || run.CandidateProfile == "" {
		t.Error("the result does not say which profiles were compared")
	}
	// The delta is the comparison's, computed by the one comparator: the
	// experiment does not subtract anything itself.
	delta, ok := run.Comparison.Metrics[MetricTokens].Delta.(TokenCounts)
	if !ok {
		t.Fatalf("token delta = %#v, want TokenCounts", run.Comparison.Metrics[MetricTokens].Delta)
	}
	if delta.Input == nil || *delta.Input != -400 {
		t.Errorf("input delta = %v, want -400", delta.Input)
	}
	// The design travels with the result: a stored result has to stay
	// interpretable without the design file beside it.
	if result.Design.Name != d.Name {
		t.Errorf("result design name = %q, want %q", result.Design.Name, d.Name)
	}
}

// The output directory is named in the design and is where the profiles go. A
// caller naming one that does not exist yet has not made a mistake — every
// capture command would otherwise fail on a path whose parent is missing, and
// the error would be about `cp` rather than about the experiment.
func TestRunPlan_CreatesTheOutputDirectoryTheDesignNames(t *testing.T) {
	dir := t.TempDir()
	d := runnableDesign(t, dir,
		experimentProfile(AdapterVersion, "claude_code", "sha-base", "skills/skill-audit", 1000),
		experimentProfile(AdapterVersion, "claude_code", "sha-cand", "skills/skill-audit", 500))
	d.OutputDir = filepath.Join(dir, "results", "2026-09-20")
	if _, err := os.Stat(d.OutputDir); !os.IsNotExist(err) {
		t.Fatalf("the output directory already exists, so this proves nothing: %v", err)
	}

	result, err := runDesign(t, d)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !result.Comparable {
		t.Errorf("the experiment did not compare: %+v", result.Runs[0].Comparison)
	}
	if _, err := os.Stat(result.Runs[0].BaselineProfile); err != nil {
		t.Errorf("the baseline profile is not where the result says it is: %v", err)
	}
}

// And an output directory that cannot be created is the experiment's error to
// report, not something the capture command is left to fail on obscurely.
func TestRunPlan_SaysSoWhenTheOutputDirectoryCannotBeCreated(t *testing.T) {
	dir := t.TempDir()
	d := runnableDesign(t, dir,
		experimentProfile(AdapterVersion, "claude_code", "sha-base", "skills/skill-audit", 1000),
		experimentProfile(AdapterVersion, "claude_code", "sha-cand", "skills/skill-audit", 500))
	// A regular file where a directory would have to be: MkdirAll cannot make a
	// directory underneath it, on any platform and whoever is running.
	blocker := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("this is a file"), 0o600); err != nil {
		t.Fatal(err)
	}
	d.OutputDir = filepath.Join(blocker, "results")

	_, err := runDesign(t, d)
	if err == nil {
		t.Fatal("an experiment whose output directory cannot exist ran anyway")
	}
	// Named as what it is. Left to the capture command, the same setup fails as
	// whatever `cp` says about a path it could not write, which is a sentence
	// about `cp`.
	if !strings.Contains(err.Error(), "could not be created") {
		t.Errorf("error = %q, want it to say the directory could not be created", err)
	}
}

// The headline. The two commands are the caller's, and nothing stops them being
// two different builds of the profiler — a fresh candidate against a baseline
// captured months ago by an older adapter. That pair is exactly what `compare`
// refuses, and the experiment must not launder it into a result that reads as
// the skill's regression.
func TestRunPlan_ReportsARefusedPairAsRefusedAndNeverAsADelta(t *testing.T) {
	dir := t.TempDir()
	d := runnableDesign(t, dir,
		experimentProfile("0.4.1", "claude_code", "sha-base", "skills/skill-audit", 1000),
		experimentProfile(AdapterVersion, "claude_code", "sha-cand", "skills/skill-audit", 500))

	result, err := runDesign(t, d)
	if err != nil {
		t.Fatalf("run: %v — a refused pair is a result, not an error", err)
	}
	if result.Comparable {
		t.Error("the experiment says it is comparable")
	}
	if len(result.Runs) != 1 {
		t.Fatalf("runs = %d, want 1 — the run happened and its refusal is the answer", len(result.Runs))
	}
	report := result.Runs[0].Comparison
	for _, want := range []string{"0.4.1", AdapterVersion} {
		if !strings.Contains(report.Refusal, want) {
			t.Errorf("refusal = %q, want it to name version %q", report.Refusal, want)
		}
	}
	if len(report.Metrics) == 0 {
		t.Fatal("the comparison covered no metric, so this proves nothing")
	}
	for name, mc := range report.Metrics {
		if mc.Comparable || mc.Delta != nil {
			t.Errorf("%s: comparable=%v delta=%v — a refused pair still produced a number", name, mc.Comparable, mc.Delta)
		}
	}
	// -500 is the delta this pair would have produced. It must be nowhere in
	// the document a wrapper stores.
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "-500") {
		t.Errorf("the refused delta is in the result anyway:\n%s", data)
	}

	// The control: the same two profiles at one adapter version do compare and
	// do produce that delta. Without it, a RunPlan that refused everything
	// would pass every assertion above.
	controlDir := t.TempDir()
	control := runnableDesign(t, controlDir,
		experimentProfile(AdapterVersion, "claude_code", "sha-base", "skills/skill-audit", 1000),
		experimentProfile(AdapterVersion, "claude_code", "sha-cand", "skills/skill-audit", 500))
	got, err := runDesign(t, control)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Comparable {
		t.Fatal("the control pair did not compare, so the refusal above proves nothing")
	}
	data, err = json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "-500") {
		t.Errorf("the control did not report the delta, so the refusal above proves nothing:\n%s", data)
	}
}

// The other half of the same refusal, and the one a first-time user meets: a
// capture command that is not this profiler writes profiles naming no adapter
// version at all. Two of those agree only in the sense that two unknowns agree.
func TestRunPlan_ReportsAPairThatNamesNoAdapterVersionAsRefused(t *testing.T) {
	dir := t.TempDir()
	d := runnableDesign(t, dir,
		experimentProfile("", "claude_code", "sha-base", "skills/skill-audit", 1000),
		experimentProfile("", "claude_code", "sha-cand", "skills/skill-audit", 500))

	result, err := runDesign(t, d)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Comparable {
		t.Error("two profiles naming no adapter version were compared anyway")
	}
	if !strings.Contains(result.Runs[0].Comparison.Refusal, "adapter_version") {
		t.Errorf("refusal = %q, want it to name the field that is empty", result.Runs[0].Comparison.Refusal)
	}
}

// Every run is an independent observation, so one refused run does not discard
// the ones that worked — but the experiment as a whole is comparable only when
// every run of it compared something, which is what the exit status is derived
// from.
func TestRunPlan_TheExperimentIsComparableOnlyWhenEveryRunWas(t *testing.T) {
	dir := t.TempDir()
	good := experimentProfile(AdapterVersion, "claude_code", "sha-base", "skills/skill-audit", 1000)
	d := runnableDesign(t, dir, good,
		experimentProfile(AdapterVersion, "claude_code", "sha-cand", "skills/skill-audit", 500))
	d.Repetitions = 2
	// The second repetition's candidate is written by a different adapter: one
	// run compares, one is refused.
	stale := writeJSON(t, dir, "stale.json",
		experimentProfile("0.4.1", "claude_code", "sha-cand", "skills/skill-audit", 500))
	fresh := writeJSON(t, dir, "fresh.json",
		experimentProfile(AdapterVersion, "claude_code", "sha-cand", "skills/skill-audit", 500))
	d.Candidate.Command = fmt.Sprintf("if [ $REP -eq 1 ]; then cp %q $PROFILE; else cp %q $PROFILE; fi", fresh, stale)

	result, err := runDesign(t, d)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(result.Runs) != 2 {
		t.Fatalf("runs = %d, want 2 — a refused run does not abort the ones after it", len(result.Runs))
	}
	if !result.Runs[0].Comparison.Comparable {
		t.Error("the first run did not compare, so the mixture below proves nothing")
	}
	if result.Runs[1].Comparison.Comparable {
		t.Error("the second run compared two adapter versions")
	}
	if result.Comparable {
		t.Error("the experiment reports itself comparable with a refused run in it")
	}
}

// Vacuity: "every run compared something" is true of no runs. LoadPlan refuses
// a plan with no runs, so this shape cannot be reached through the CLI — which
// is exactly why the derivation is asserted directly rather than left to be
// believed.
func TestExperimentResult_NoRunsIsNotComparable(t *testing.T) {
	if (ExperimentResult{}).comparable() {
		t.Error("an experiment with no runs reports itself comparable")
	}
}

func TestRunPlan_AStepThatFailsStopsTheExperiment(t *testing.T) {
	dir := t.TempDir()
	d := runnableDesign(t, dir,
		experimentProfile(AdapterVersion, "claude_code", "sha-base", "skills/skill-audit", 1000),
		experimentProfile(AdapterVersion, "claude_code", "sha-cand", "skills/skill-audit", 500))
	d.TaskFamilies = []string{"audit"}
	d.Candidate.Command = "echo $PROFILE >/dev/null; exit 3"

	_, err := runDesign(t, d)
	if err == nil {
		t.Fatal("a failing capture command produced a result")
	}
	for _, want := range []string{"candidate", "audit"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to name %q", err, want)
		}
	}
}

// A step that exits 0 and writes nothing is the trap this check exists for: the
// profile at that path is then either absent, or — worse — left over from an
// earlier run, and the experiment would compare yesterday's numbers and call
// them today's.
func TestRunPlan_AStepMustWriteTheProfileItWasGiven(t *testing.T) {
	for _, tc := range []struct {
		name      string
		preexists bool
	}{
		{"a step that writes no profile at all", false},
		{"a step that leaves an earlier run's profile in place", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			d := runnableDesign(t, dir,
				experimentProfile(AdapterVersion, "claude_code", "sha-base", "skills/skill-audit", 1000),
				experimentProfile(AdapterVersion, "claude_code", "sha-cand", "skills/skill-audit", 500))
			d.Baseline.Command = "true # $PROFILE"

			plan, err := GeneratePlan(d)
			if err != nil {
				t.Fatal(err)
			}
			path := plan.Runs[0].Baseline.ProfilePath
			if tc.preexists {
				data, err := json.Marshal(experimentProfile(AdapterVersion, "claude_code", "sha-base", "skills/skill-audit", 999))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, data, 0o600); err != nil {
					t.Fatal(err)
				}
			}

			_, err = RunPlan(plan)
			if err == nil {
				t.Fatal("a step that wrote no profile produced a result")
			}
			if !strings.Contains(err.Error(), path) {
				t.Errorf("error = %q, want it to name the path the step did not write: %q", err, path)
			}
		})
	}
}

// The plan says what each step will produce. A profile that does not match is
// the wrong file — the other condition's, another skill's, another snapshot's —
// and subtracting it would answer a question nobody asked.
func TestRunPlan_RefusesAProfileThatIsNotTheOneTheStepDeclared(t *testing.T) {
	for _, tc := range []struct {
		name     string
		profile  Profile
		wants    []string
		mismatch string
	}{
		{"another snapshot", experimentProfile(AdapterVersion, "claude_code", "sha-somewhere-else", "skills/skill-audit", 1000),
			[]string{"snapshot_hash", "sha-base", "sha-somewhere-else"}, "snapshot"},
		{"another skill", experimentProfile(AdapterVersion, "claude_code", "sha-base", "skills/other-skill", 1000),
			[]string{"skill_dir", "skills/skill-audit", "skills/other-skill"}, "skill dir"},
		{"another harness", experimentProfile(AdapterVersion, "cursor", "sha-base", "skills/skill-audit", 1000),
			[]string{"harness", "claude_code", "cursor"}, "harness"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			d := runnableDesign(t, dir,
				experimentProfile(AdapterVersion, "claude_code", "sha-base", "skills/skill-audit", 1000),
				experimentProfile(AdapterVersion, "claude_code", "sha-cand", "skills/skill-audit", 500))
			wrong := writeJSON(t, dir, "wrong.json", tc.profile)
			d.Baseline.Command = fmt.Sprintf("cp %q $PROFILE", wrong)

			_, err := runDesign(t, d)
			if err == nil {
				t.Fatalf("a %s profile was accepted for the baseline step", tc.mismatch)
			}
			for _, want := range tc.wants {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error = %q, want it to name %q", err, want)
				}
			}
		})
	}
}

func TestRunPlan_RefusesAProfileItCannotRead(t *testing.T) {
	dir := t.TempDir()
	d := runnableDesign(t, dir,
		experimentProfile(AdapterVersion, "claude_code", "sha-base", "skills/skill-audit", 1000),
		experimentProfile(AdapterVersion, "claude_code", "sha-cand", "skills/skill-audit", 500))
	d.Candidate.Command = `printf '{"hello":"world"}' > $PROFILE`

	_, err := runDesign(t, d)
	if err == nil {
		t.Fatal("a document that is not a profile was compared")
	}
	if !strings.Contains(err.Error(), "schema") {
		t.Errorf("error = %q, want the profile reader's own refusal", err)
	}
}

// The result is an artifact somebody stores and reads later, so it has to
// survive the round trip a wrapper puts it through.
func TestExperimentResult_RoundTripsThroughJSON(t *testing.T) {
	dir := t.TempDir()
	d := runnableDesign(t, dir,
		experimentProfile(AdapterVersion, "claude_code", "sha-base", "skills/skill-audit", 1000),
		experimentProfile(AdapterVersion, "claude_code", "sha-cand", "skills/skill-audit", 500))
	result, err := runDesign(t, d)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var back ExperimentResult
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("the result does not read back: %v\n%s", err, data)
	}
	if back.Schema != result.Schema || len(back.Runs) != len(result.Runs) || back.Comparable != result.Comparable {
		t.Errorf("round trip lost something: %+v", back)
	}
	if back.Runs[0].Comparison.Schema != ComparisonSchema {
		t.Errorf("the comparison inside the result did not survive: %+v", back.Runs[0].Comparison)
	}
}
