package difftest

// The E2 differ: runs the upstream Python runner and the Go ports over the
// same corpus, then diffs (rule, file, line) keyed records.
//
// Upstream requires the SkillSpector venv interpreter: set SKILLSPECTOR_PYTHON
// (default /tmp/skillspector-venv/venv/bin/python). If the interpreter or the
// skillspector module is absent the test SKIPS — an honest gap, never a pass.

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

func TestDifferential(t *testing.T) {
	py := os.Getenv("SKILLSPECTOR_PYTHON")
	if py == "" {
		py = "/tmp/skillspector-venv/venv/bin/python"
	}
	if _, err := os.Stat(py); err != nil {
		t.Skipf("skillspector interpreter not found at %s (set SKILLSPECTOR_PYTHON)", py)
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

	outDir := filepath.Join(root, "docs", "research", "experiments")
	os.MkdirAll(outDir, 0o755)
	os.WriteFile(filepath.Join(outDir, "E2-differential.md"), []byte(sb.String()), 0o644)
	raw, _ := json.MarshalIndent(map[string]any{
		"upstream_records": len(upstream),
		"go_records":       len(ours),
		"results":          results,
	}, "", "  ")
	os.WriteFile(filepath.Join(outDir, "E2-differential.json"), raw, 0o644)

	t.Logf("upstream=%d go=%d\n%s", len(upstream), len(ours), sb.String())
}
