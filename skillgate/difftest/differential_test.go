package difftest

// The E2 differ: runs the upstream Python runner and the Go ports over the
// same corpus, then diffs (rule, file, line) keyed records.
//
// Upstream requires the SkillSpector venv interpreter: set SKILLSPECTOR_PYTHON
// (default /tmp/skillspector-venv/venv/bin/python). If the interpreter or the
// skillspector module is absent the test SKIPS — an honest gap, never a pass.
//
// The report is written only when SKILLGATE_E2_REPORT_DIR names an absolute
// directory outside the repository; otherwise it is logged and nothing is
// written. See TestMain in tree_guard_test.go, which fails the package if any
// test here changes the tree.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var fileTypeByExt = map[string]string{
	".md": "markdown", ".markdown": "markdown", ".py": "python",
	".sh": "shell", ".bash": "shell", ".zsh": "shell", ".json": "json",
	".yaml": "yaml", ".yml": "yaml", ".toml": "toml", ".txt": "text",
	".js": "javascript", ".ts": "typescript", ".rb": "ruby",
}

type diffResult struct {
	Matches  int
	Drift    int // same (rule,file,line), different evidence
	Misses   []Match
	Extras   []Match
	UpErrors []Match // upstream ANALYZER_ERROR rows
}

func key(m Match) string { return m.Rule + "|" + m.File + "|" + fmt.Sprint(m.Line) }

func diffRecords(upstream, ours []Match) map[string]*diffResult {
	up := map[string][]Match{}
	go_ := map[string][]Match{}
	rules := map[string]bool{}
	for _, m := range upstream {
		up[key(m)] = append(up[key(m)], m)
		rules[strings.SplitN(m.Rule, ":", 2)[0]] = true
	}
	for _, m := range ours {
		go_[key(m)] = append(go_[key(m)], m)
		rules[strings.SplitN(m.Rule, ":", 2)[0]] = true
	}
	out := map[string]*diffResult{}
	for r := range rules {
		out[r] = &diffResult{}
	}
	count := func(src map[string][]Match, m Match) {
		base := strings.SplitN(m.Rule, ":", 2)[0]
		if _, ok := out[base]; !ok {
			out[base] = &diffResult{}
		}
	}
	for k, ms := range up {
		for _, m := range ms {
			count(up, m)
			r := out[strings.SplitN(m.Rule, ":", 2)[0]]
			_ = k
			if strings.HasPrefix(m.Evidence, "ANALYZER_ERROR:") {
				r.UpErrors = append(r.UpErrors, m)
				continue
			}
			if _, ok := go_[k]; ok {
				r.Matches++
				if go_[k][0].Evidence != m.Evidence {
					r.Drift++
				}
			} else {
				r.Misses = append(r.Misses, m)
			}
		}
	}
	for k, ms := range go_ {
		for _, m := range ms {
			r := out[strings.SplitN(m.Rule, ":", 2)[0]]
			if _, ok := up[k]; !ok {
				r.Extras = append(r.Extras, m)
			}
		}
	}
	return out
}

func walkCorpus(t *testing.T, root string) []Doc {
	t.Helper()
	var docs []Doc
	err := filepath.WalkDir(root, func(p string, e os.DirEntry, err error) error {
		if err != nil || e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		docs = append(docs, Doc{
			Path:     rel,
			FileType: fileTypeByExt[strings.ToLower(filepath.Ext(p))],
			Text:     string(data),
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return docs
}

func walkBundles(t *testing.T, root string) map[string]map[string]string {
	t.Helper()
	bundles := map[string]map[string]string{}
	entries, err := os.ReadDir(root)
	if err != nil {
		return bundles
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		files := map[string]string{}
		bdir := filepath.Join(root, e.Name())
		filepath.WalkDir(bdir, func(p string, de os.DirEntry, err error) error {
			if err != nil || de.IsDir() {
				return nil
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(bdir, p)
			files[filepath.ToSlash(rel)] = string(data)
			return nil
		})
		bundles[e.Name()] = files
	}
	return bundles
}

func repoRoot(t *testing.T) string {
	// difftest/ → skillgate/ → repo root
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(filepath.Dir(wd))
}

const (
	// envInterpreter names the SkillSpector venv interpreter to run upstream with.
	envInterpreter = "SKILLSPECTOR_PYTHON"
	// defaultInterpreter is where the E2 setup notes put the venv.
	defaultInterpreter = "/tmp/skillspector-venv/venv/bin/python"
	// envReportDir opts into writing the E2 report, and says where. There is no
	// default: the report is an experiment artefact, and the only destination a
	// default could name is somewhere the runner did not ask for.
	envReportDir = "SKILLGATE_E2_REPORT_DIR"
)

func skillspectorInterpreter() string {
	if py := os.Getenv(envInterpreter); py != "" {
		return py
	}
	return defaultInterpreter
}

// upstreamUnavailable reports why the upstream runner cannot be used with py,
// or "" if it can. Both legs are probed — the interpreter file and the
// skillspector module inside it — because an interpreter that exists but
// cannot import the module is the ordinary case, and probing only the file
// turns the documented skip into a traceback.
func upstreamUnavailable(py string) string {
	if _, err := os.Stat(py); err != nil {
		return fmt.Sprintf("interpreter not found at %s (set %s)", py, envInterpreter)
	}
	out, err := exec.Command(py, "-c", "import skillspector").CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(lastLine(string(out)))
		if detail == "" {
			detail = err.Error()
		}
		return fmt.Sprintf("%s cannot import the skillspector module — %s (set %s to the venv interpreter)", py, detail, envInterpreter)
	}
	return ""
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return lines[len(lines)-1]
}

// e2ReportDir resolves the opt-in destination for the E2 report. It returns ""
// when no report was asked for, and refuses any destination inside root: the
// differential reads the repository as part of its corpus, so writing its
// findings back into that corpus both corrupts the next run's input and makes
// `git status` — which the release's boundary checks read — report work nobody
// did.
func e2ReportDir(root string) (string, error) {
	dir := os.Getenv(envReportDir)
	if dir == "" {
		return "", nil
	}
	if !filepath.IsAbs(dir) {
		return "", fmt.Errorf("%s=%q must be an absolute path: a relative one resolves inside the repository", envReportDir, dir)
	}
	resolvedDir, err := resolveExisting(dir)
	if err != nil {
		return "", err
	}
	resolvedRoot, err := resolveExisting(root)
	if err != nil {
		return "", err
	}
	if resolvedDir == resolvedRoot || strings.HasPrefix(resolvedDir, resolvedRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("%s=%q is inside the repository at %s: the E2 report is a generated artefact and may not land in the tree", envReportDir, dir, root)
	}
	return dir, nil
}

// resolveExisting canonicalises p through its nearest existing ancestor, so a
// destination that has not been created yet is still compared against the
// repository by real path rather than by spelling. On macOS /tmp is a symlink
// to /private/tmp, and a containment check that skipped this would be trivially
// evaded by the spelling of the path.
func resolveExisting(p string) (string, error) {
	p = filepath.Clean(p)
	rest := ""
	for {
		if resolved, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Join(resolved, rest), nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(p)
		if parent == p {
			return "", fmt.Errorf("cannot resolve %s: no existing ancestor", p)
		}
		rest = filepath.Join(filepath.Base(p), rest)
		p = parent
	}
}

// The documented contract at the top of this file: the differential skips when
// upstream cannot be run, and never fails for its absence. Both legs have to be
// probed, because an interpreter that exists but cannot import skillspector is
// the common case — a stock python3 is on every machine this runs on, and the
// venv is on almost none.
func TestUpstreamUnavailableNamesEitherMissingLeg(t *testing.T) {
	dir := t.TempDir()
	importable := stubInterpreter(t, dir, "importable", 0)
	bare := stubInterpreter(t, dir, "bare", 1)
	absent := filepath.Join(dir, "no-such-interpreter")

	if got := upstreamUnavailable(importable); got != "" {
		t.Errorf("interpreter that imports skillspector: got reason %q, want the differential to run", got)
	}
	for _, tc := range []struct{ name, py, want string }{
		{"absent interpreter", absent, absent},
		{"interpreter cannot import the module", bare, "skillspector"},
	} {
		got := upstreamUnavailable(tc.py)
		if got == "" {
			t.Errorf("%s: no reason returned — the differential would run and fail instead of skipping", tc.name)
			continue
		}
		if !strings.Contains(got, tc.want) {
			t.Errorf("%s: reason %q does not name %q", tc.name, got, tc.want)
		}
		if !strings.Contains(got, envInterpreter) {
			t.Errorf("%s: reason %q does not say which variable fixes it", tc.name, got)
		}
	}
}

// stubInterpreter writes an executable that exits with code, standing in for an
// interpreter with and without the skillspector module. A stub rather than a
// real python because the contract under test is "what do we do when the
// interpreter cannot do the job", and that must be decidable on a machine with
// no Python at all.
func stubInterpreter(t *testing.T, dir, name string, code int) string {
	t.Helper()
	p := filepath.Join(dir, name)
	script := fmt.Sprintf("#!/bin/sh\nexit %d\n", code)
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// Where the E2 report may land. The shipped version wrote it into
// docs/research/experiments/ on every run, which is how two generated files
// came to sit in the working copy with no commit behind them.
func TestE2ReportDirRefusesTheRepositoryTree(t *testing.T) {
	root := repoRoot(t)
	outside := t.TempDir()

	t.Run("unset means no report", func(t *testing.T) {
		t.Setenv(envReportDir, "")
		dir, err := e2ReportDir(root)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != "" {
			t.Errorf("got %q, want no destination — the report is opt-in", dir)
		}
	})

	t.Run("inside the repository is refused", func(t *testing.T) {
		inside := filepath.Join(root, "docs", "research", "experiments")
		t.Setenv(envReportDir, inside)
		dir, err := e2ReportDir(root)
		if err == nil {
			t.Fatalf("got destination %q, want refusal — a test may not write into the repository", dir)
		}
		if !strings.Contains(err.Error(), inside) {
			t.Errorf("error %q does not name the refused path", err)
		}
	})

	t.Run("relative is refused", func(t *testing.T) {
		t.Setenv(envReportDir, "reports")
		if dir, err := e2ReportDir(root); err == nil {
			t.Fatalf("got destination %q, want refusal — a relative path resolves inside the repository", dir)
		}
	})

	t.Run("outside the repository is accepted", func(t *testing.T) {
		t.Setenv(envReportDir, outside)
		dir, err := e2ReportDir(root)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != outside {
			t.Errorf("got %q, want %q", dir, outside)
		}
	})
}

func TestDifferential(t *testing.T) {
	py := skillspectorInterpreter()
	if reason := upstreamUnavailable(py); reason != "" {
		t.Skip("skipping the upstream differential: " + reason)
	}
	runner := filepath.Join(".", "run_upstream.py")
	root := repoRoot(t)

	corpusRoots := []string{
		"testdata/upstream",
		"testdata/hostile",
		filepath.Join(root, "skills", "skill-audit"),
		filepath.Join(root, "skills", "skill-rewrite"),
		filepath.Join(root, "skills", "skill-gate"),
	}
	bundleRoots := []string{
		"testdata/hostile/bh2",
		"testdata/hostile/ssr1",
		"testdata/upstream",
	}

	// --- upstream side ---
	args := []string{runner}
	for _, c := range corpusRoots {
		args = append(args, "--corpus-root", c)
	}
	for _, b := range bundleRoots {
		args = append(args, "--bundle-root", b)
	}
	cmd := exec.Command(py, args...)
	cmd.Dir = "."
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("upstream runner failed: %v\n%s", err, stderr.String())
	}
	var upstream []Match
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if line == "" {
			continue
		}
		var m Match
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("bad upstream record %q: %v", line, err)
		}
		upstream = append(upstream, m)
	}

	// --- Go side ---
	var ours []Match
	for _, c := range corpusRoots {
		for _, doc := range walkCorpus(t, c) {
			for _, r := range FileRules {
				if r.Detect != nil {
					ours = append(ours, r.Detect(&doc)...)
				}
			}
		}
	}
	for _, b := range bundleRoots {
		for name, files := range walkBundles(t, b) {
			for _, r := range BundleRules {
				ours = append(ours, r.Detect(name, files)...)
			}
		}
	}

	results := diffRecords(upstream, ours)

	// --- report ---
	var sb strings.Builder
	sb.WriteString("# E2 differential — Python re vs Go RE2\n\n")
	sb.WriteString("| rule | matches | misses | extras | evidence drift | upstream errors |\n")
	sb.WriteString("|---|---|---|---|---|---|\n")
	type row struct {
		rule string
		*diffResult
	}
	var rows []row
	for r, res := range results {
		rows = append(rows, row{r, res})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].rule < rows[j].rule })
	for _, rw := range rows {
		fmt.Fprintf(&sb, "| %s | %d | %d | %d | %d | %d |\n",
			rw.rule, rw.Matches, len(rw.Misses), len(rw.Extras), rw.Drift, len(rw.UpErrors))
	}
	sb.WriteString("\n## misses (upstream only)\n")
	for _, rw := range rows {
		for _, m := range rw.Misses {
			fmt.Fprintf(&sb, "- %s %s:%d `%s`\n", m.Rule, m.File, m.Line, m.Evidence)
		}
	}
	sb.WriteString("\n## extras (go only)\n")
	for _, rw := range rows {
		for _, m := range rw.Extras {
			fmt.Fprintf(&sb, "- %s %s:%d `%s`\n", m.Rule, m.File, m.Line, m.Evidence)
		}
	}
	sb.WriteString("\n## upstream analyzer errors\n")
	for _, rw := range rows {
		for _, m := range rw.UpErrors {
			fmt.Fprintf(&sb, "- %s %s `%s`\n", m.Rule, m.File, m.Evidence)
		}
	}

	raw, err := json.MarshalIndent(map[string]any{
		"upstream_records": len(upstream),
		"go_records":       len(ours),
		"results":          results,
	}, "", "  ")
	if err != nil {
		t.Fatalf("encode report: %v", err)
	}

	// The report is always in the test log; the files are the opt-in.
	t.Logf("upstream=%d go=%d\n%s", len(upstream), len(ours), sb.String())

	outDir, err := e2ReportDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if outDir == "" {
		t.Logf("no %s set — report not written to disk", envReportDir)
		return
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string][]byte{
		"E2-differential.md":   []byte(sb.String()),
		"E2-differential.json": raw,
	} {
		if err := os.WriteFile(filepath.Join(outDir, name), body, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	t.Logf("report written to %s", outDir)
}
