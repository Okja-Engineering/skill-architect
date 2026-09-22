package skillgate

import (
	"path"
	"regexp"
	"sort"
	"strings"
)

// Reference graph — G5's static half. Every path-ish reference in loaded
// text is resolved against the ledger deterministically: markdown links,
// backtick-quoted paths, and bare bundle-root paths. A reference that
// escapes the bundle is T019's; here we check existence inside the bundle
// (dangling) and cycles (file → file).
//
// This is the deterministic answer to SkillSpector's AE1 class: refs are
// resolved for real, so "$skill_root/…" and prose-number noise never fire.

var (
	reMdLink   = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)\)`)
	reBacktick = regexp.MustCompile("`([^`\n]+)`")
	reBarePath = regexp.MustCompile(`\b(?:references|scripts|assets|examples|hooks)/[\w.\-/]+`)
)

// fileRefExts are the extensions that make a slash-containing token a
// *file* reference rather than layout prose (`references/`, `scripts/`) or
// a command argument (`brew install a/b`).
var fileRefExts = []string{".md", ".mdc", ".sh", ".bash", ".py", ".js", ".mjs", ".ts",
	".rb", ".pl", ".ps1", ".json", ".yaml", ".yml", ".toml", ".txt", ".csv", ".sql"}

// extractRefs returns bundle-relative *file* candidates referenced by a
// file. Conservative on purpose: only single tokens that contain a slash
// and end in a file extension qualify. Directory mentions, placeholders,
// and command strings are prose — flagging them is the AE1 noise class this
// graph exists to avoid.
func extractRefs(f *FileContent) []string {
	var refs []string
	add := func(s string, explicit bool) {
		s = strings.TrimSpace(s)
		s = strings.Trim(s, `'"`)
		s = strings.TrimRight(s, ".,;:)]}>")
		if s == "" || strings.ContainsAny(s, " \t") || strings.Contains(s, "://") ||
			strings.HasPrefix(s, "#") || strings.HasPrefix(s, "mailto:") ||
			strings.HasPrefix(s, "~/") { // home-relative, not a bundle path
			return
		}
		// Slash-containing tokens are path-shaped anywhere. Bare filenames
		// only count from explicit markdown link targets — `[x](a.md)` is a
		// reference; `SKILL.md` in prose is a concept.
		if !strings.Contains(s, "/") && !explicit {
			return
		}
		if strings.ContainsAny(s, "$<>*{}") || strings.HasPrefix(s, "-") {
			return
		}
		isFile := false
		for _, ext := range fileRefExts {
			if strings.HasSuffix(s, ext) {
				isFile = true
				break
			}
		}
		if isFile {
			refs = append(refs, s)
		}
	}
	for _, m := range reMdLink.FindAllStringSubmatch(f.Text, -1) {
		add(m[1], true)
	}
	for _, m := range reBacktick.FindAllStringSubmatch(f.Text, -1) {
		add(m[1], false)
	}
	for _, m := range reBarePath.FindAllString(f.Text, -1) {
		add(m, false)
	}
	return refs
}

// resolveRef maps a reference onto a ledger path. Relative refs resolve
// against the referencing file's directory; refs that escape return "" with
// escaped=true (T019's job, not ours).
func resolveRef(dir, ref string) (resolved string, escaped bool) {
	ref = strings.ReplaceAll(ref, "\\", "/")
	var joined string
	if strings.HasPrefix(ref, "/") {
		joined = path.Clean(strings.TrimPrefix(ref, "/"))
	} else {
		joined = path.Join(dir, ref)
	}
	if joined == ".." || strings.HasPrefix(joined, "../") {
		return "", true
	}
	return joined, false
}

// refTargets returns candidate ledger paths for a reference, in precedence
// order: the referencing file's directory first, then each ancestor up to
// the package root. Skill docs conventionally resolve paths relative to the
// enclosing skill root — a README nested under skills/x/lang/ refers to
// skills/x/shared/y.md as "shared/y.md" — so a file-dir miss is retried
// from each ancestor before a ref counts as dangling. A leading "/" is
// package-root absolute; a ref that escapes the root returns nil (T019's
// job).
func refTargets(dir, ref string) []string {
	ref = strings.ReplaceAll(ref, "\\", "/")
	if strings.HasPrefix(ref, "/") {
		j := path.Clean(strings.TrimPrefix(ref, "/"))
		if j == ".." || strings.HasPrefix(j, "../") {
			return nil
		}
		return []string{j}
	}
	var out []string
	for d := dir; ; {
		j := path.Join(d, ref)
		if j == ".." || strings.HasPrefix(j, "../") {
			// An ancestor interpretation escapes the root — keep the
			// candidates already collected at deeper levels. Only a ref
			// escaping at its own level means "outside the bundle" (T019).
			return out
		}
		out = append(out, j)
		if d == "" {
			return out
		}
		if i := strings.LastIndex(d, "/"); i >= 0 {
			d = d[:i]
		} else {
			d = ""
		}
	}
}

// ruleG001 — dangling reference: a bundle-internal path that resolves inside
// the root but names no ledger file. Medium: broken progressive disclosure,
// not a security hole.
var ruleG001 = rule{
	id: "SK-G001", sev: SeverityMedium, quality: "reliability", effort: 10,
	msg:   "dangling reference: path resolves inside the bundle but no file exists",
	files: isLoadedText,
	scan:  func(f *FileContent) []string { return nil }, // filled by scanBundle below
	scanBundle: func(l *Ledger) []Finding {
		var out []Finding
		for _, f := range l.Files {
			if f.Entry.Outcome != "inspected" || !isLoadedText(f.Entry.Path) {
				continue
			}
			dir := f.Entry.Path
			if i := strings.LastIndex(dir, "/"); i >= 0 {
				dir = dir[:i]
			} else {
				dir = ""
			}
			seen := map[string]bool{}
			for _, ref := range extractRefs(&f) {
				targets := refTargets(dir, ref)
				if len(targets) == 0 || seen[ref] {
					continue
				}
				seen[ref] = true
				found := false
				for _, t := range targets {
					if l.Get(t) != nil || l.hasDir(t) {
						found = true
						break
					}
				}
				if !found {
					out = append(out, Finding{
						RuleID: "SK-G001", Severity: SeverityMedium, Quality: "reliability",
						Message: "dangling reference: path resolves inside the bundle but no file exists",
						File:    f.Entry.Path, Evidence: ref, EffortMinutes: 10, Source: "skillgate",
					})
				}
			}
		}
		return out
	},
}

// ruleG002 — reference cycle. Tarjan over the file→file resolved graph.
// Info: a cycle is a progressive-disclosure smell, not a defect.
var ruleG002 = rule{
	id: "SK-G002", sev: SeverityInfo, quality: "maintainability", effort: 15,
	msg:   "reference cycle between bundle files",
	files: nil,
	scanBundle: func(l *Ledger) []Finding {
		graph := map[string][]string{}
		for _, f := range l.Files {
			if f.Entry.Outcome != "inspected" || !isLoadedText(f.Entry.Path) {
				continue
			}
			dir := f.Entry.Path
			if i := strings.LastIndex(dir, "/"); i >= 0 {
				dir = dir[:i]
			} else {
				dir = ""
			}
			for _, ref := range extractRefs(&f) {
				for _, t := range refTargets(dir, ref) {
					if l.Get(t) != nil && t != f.Entry.Path {
						graph[f.Entry.Path] = append(graph[f.Entry.Path], t)
						break
					}
				}
			}
		}
		var out []Finding
		for _, scc := range tarjan(graph) {
			if len(scc) < 2 {
				continue
			}
			out = append(out, Finding{
				RuleID: "SK-G002", Severity: SeverityInfo, Quality: "maintainability",
				Message: "reference cycle between bundle files",
				File:    scc[0], Evidence: strings.Join(scc, " <-> "), EffortMinutes: 15, Source: "skillgate",
			})
		}
		return out
	},
}

// tarjan returns the strongly connected components of a directed graph
// (iterative to keep it boring).
func tarjan(graph map[string][]string) [][]string {
	index := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	var sccs [][]string
	counter := 0

	var strongconnect func(v string)
	strongconnect = func(v string) {
		index[v] = counter
		low[v] = counter
		counter++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range graph[v] {
			if _, seen := index[w]; !seen {
				strongconnect(w)
				if low[w] < low[v] {
					low[v] = low[w]
				}
			} else if onStack[w] && index[w] < low[v] {
				low[v] = index[w]
			}
		}
		if low[v] == index[v] {
			var scc []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, scc)
		}
	}
	// Seed the DFS in sorted key order: map iteration order is nondeterministic
	// and would flip SCC discovery order, member order, and therefore the
	// finding's File/Evidence between runs (rule-language §8 prerequisite).
	roots := make([]string, 0, len(graph))
	for v := range graph {
		roots = append(roots, v)
	}
	sort.Strings(roots)
	for _, v := range roots {
		if _, seen := index[v]; !seen {
			strongconnect(v)
		}
	}
	// Canonicalize each SCC and their emission order so a cycle finding's
	// fingerprint and report position are stable across runs.
	for _, scc := range sccs {
		sort.Strings(scc)
	}
	sort.Slice(sccs, func(i, j int) bool { return sccs[i][0] < sccs[j][0] })
	return sccs
}
