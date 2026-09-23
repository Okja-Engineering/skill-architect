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
// escapes the bundle is T019's; here we ask the graph three questions over
// one traversal: existence inside the bundle (G001, dangling), cycles
// (G002, file → file), and reachability from the harness's entry points
// (G003, orphaned).
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

// refToken is one path-shaped token a file names, before it is classified
// as pointing at a file or at a directory.
type refToken struct {
	text string
	// at is the byte offset in the file's text where the token was found,
	// so a finding about it can name the line the author has to open.
	at int
	// explicit marks a markdown link target — the one form where a bare
	// filename is a reference rather than a concept.
	explicit bool
}

// refTokens returns the path-shaped tokens a file names, in source order:
// markdown link targets, backtick-quoted paths, and bare paths under the
// conventional payload directories. Junk is filtered here — URLs, anchors,
// home-relative paths, placeholders, command strings — so that every
// consumer of the reference vocabulary reads the same token stream and
// only the terminal question (file or directory?) differs between them.
func refTokens(f *FileContent) []refToken {
	var out []refToken
	add := func(s string, at int, explicit bool) {
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
		out = append(out, refToken{text: s, at: at, explicit: explicit})
	}
	// The Index variants, so every token carries where it was found. The
	// three extractors each sweep the whole text, so their results interleave
	// rather than arrive in source order — the sort below makes the order
	// this function has always documented actually true, which is what lets
	// a finding cite the *first* place a reference appears rather than
	// whichever pattern happened to run first.
	for _, m := range reMdLink.FindAllStringSubmatchIndex(f.Text, -1) {
		add(f.Text[m[2]:m[3]], m[2], true)
	}
	for _, m := range reBacktick.FindAllStringSubmatchIndex(f.Text, -1) {
		add(f.Text[m[2]:m[3]], m[2], false)
	}
	for _, m := range reBarePath.FindAllStringIndex(f.Text, -1) {
		add(f.Text[m[0]:m[1]], m[0], false)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].at < out[j].at })
	return out
}

// isFileRef reports whether a token ends in one of the extensions that make
// a path-shaped token a *file* reference rather than layout prose
// (`references/`, `scripts/`).
func isFileRef(s string) bool {
	for _, ext := range fileRefExts {
		if strings.HasSuffix(s, ext) {
			return true
		}
	}
	return false
}

// extractRefs returns bundle-relative *file* candidates referenced by a
// file. Conservative on purpose: only single tokens that contain a slash
// and end in a file extension qualify. Directory mentions, placeholders,
// and command strings are prose — flagging them is the AE1 noise class this
// graph exists to avoid.
func extractRefs(f *FileContent) []refToken {
	var refs []refToken
	for _, tok := range refTokens(f) {
		if isFileRef(tok.text) {
			refs = append(refs, tok)
		}
	}
	return refs
}

// extractDirRefs returns the other half of the same token stream: path-
// shaped tokens that name no file. Only reachability reads these. G001 must
// not — "is this a directory that exists" is a different question from "is
// this a dangling file", and answering the first through extractRefs would
// change what G001 reports.
func extractDirRefs(f *FileContent) []string {
	var refs []string
	for _, tok := range refTokens(f) {
		if !isFileRef(tok.text) {
			refs = append(refs, tok.text)
		}
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

// refDir returns the directory a file's relative references resolve from —
// "" for a file at the package root.
func refDir(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[:i]
	}
	return ""
}

// anyInspected admits every inspected file as a reference *source*. Used by
// G003: an edge found in a script or a manifest comment is still an edge a
// reader follows, and a missed edge is what turns a reachable file into a
// false orphan.
func anyInspected(string) bool { return true }

// walkRefs calls fn for every reference extracted from every inspected file
// whose path satisfies include, with that file's resolution candidates in
// precedence order. One traversal; each rule asks its own question of it.
func walkRefs(l *Ledger, include func(string) bool, fn func(f *FileContent, ref refToken, targets []string)) {
	for i := range l.Files {
		f := &l.Files[i]
		if f.Entry.Outcome != "inspected" || !include(f.Entry.Path) {
			continue
		}
		dir := refDir(f.Entry.Path)
		for _, ref := range extractRefs(f) {
			fn(f, ref, refTargets(dir, ref.text))
		}
	}
}

// resolveEdges builds the file→file graph: each reference contributes one
// edge, to the first candidate that names a real ledger file. A reference
// that resolves to nothing (G001's business), escapes the root (T019's), or
// points at its own file contributes none.
func resolveEdges(l *Ledger, include func(string) bool) map[string][]string {
	graph := map[string][]string{}
	walkRefs(l, include, func(f *FileContent, _ refToken, targets []string) {
		for _, t := range targets {
			if l.Get(t) != nil && t != f.Entry.Path {
				graph[f.Entry.Path] = append(graph[f.Entry.Path], t)
				break
			}
		}
	})
	return graph
}

// reachabilityEdges is the graph G003 walks: the resolved file→file edges
// plus one more relation a reader really follows — naming a directory
// discloses what is in it. "The templates are in `assets/contracts/`" puts
// every file under that directory within reach without naming any of them,
// so each directory reference contributes an edge to every ledger file
// beneath it, and those files' own references are followed onward.
//
// Only reachability reads directory references. The cost is stated as a
// limit of the rule rather than hidden: a directory reference is
// indistinguishable from layout prose ("Layout: `references/`"), so a
// bundle naming its payload directory as a whole can never produce an
// SK-G003 for a file inside it. That gap is the deliberate side of the
// trade — a missed orphan is a gap, a false orphan is the gate accusing a
// correct skill.
func reachabilityEdges(l *Ledger) map[string][]string {
	// Reference sources widen to every inspected file here: a script or a
	// manifest naming a template is an edge a reader follows, and dropping
	// it would orphan the template.
	graph := resolveEdges(l, anyInspected)
	for i := range l.Files {
		f := &l.Files[i]
		if f.Entry.Outcome != "inspected" {
			continue
		}
		dir := refDir(f.Entry.Path)
		for _, ref := range extractDirRefs(f) {
			for _, t := range refTargets(dir, ref) {
				if !l.hasDir(t) {
					continue
				}
				prefix := strings.TrimSuffix(t, "/") + "/"
				for j := range l.Files {
					if w := l.Files[j].Entry.Path; strings.HasPrefix(w, prefix) && w != f.Entry.Path {
						graph[f.Entry.Path] = append(graph[f.Entry.Path], w)
					}
				}
				break
			}
		}
	}
	return graph
}

// ruleG001 — dangling reference: a bundle-internal path that resolves inside
// the root but names no ledger file. Medium: broken progressive disclosure,
// not a security hole.
// It has **no view axis at all**, and says so by having no `scan`. Its
// question is not "what does this line spell" but "does this reference name a
// file the ledger holds", which is cross-file by construction and is answered
// in scanBundle over the raw text.
//
// It used to declare `scan: func(*View) []string { return nil }` — a no-op —
// and `ViewCoverage()` selects on `scan != nil`, so the gate published this
// rule as reaching every view with no reason given for giving anything up. It
// reached none of them. That falsifies the mechanism's central claim, which is
// that *a rule that gives up reach cannot do so silently*: the published table
// is derived from the registry precisely so nobody has to maintain it, and a
// no-op scan is how a rule lies to it. A rule with no view axis is absent from
// the table, which is the honest answer and the one the field already encodes.
var ruleG001 = rule{
	id: "SK-G001", sev: SeverityMedium, quality: "reliability", effort: 10,
	msg: "dangling reference: path resolves inside the bundle but no file exists",
	scanBundle: func(l *Ledger) []Finding {
		var out []Finding
		seen := map[string]bool{}
		walkRefs(l, isLoadedText, func(f *FileContent, ref refToken, targets []string) {
			// Deduped on (file, reference) and nothing else, deliberately.
			// Adding the offset to this key would turn one dangling
			// reference mentioned three times into three findings, which
			// would be a change to *what the gate reports* rather than to
			// where it points. The first occurrence wins, and refTokens is
			// sorted by offset, so that is the earliest one in the file.
			key := f.Entry.Path + "\x00" + ref.text
			if len(targets) == 0 || seen[key] {
				return
			}
			seen[key] = true
			for _, t := range targets {
				if l.Get(t) != nil || l.hasDir(t) {
					return
				}
			}
			out = append(out, Finding{
				RuleID: "SK-G001", Severity: SeverityMedium, Quality: "reliability",
				Message: "dangling reference: path resolves inside the bundle but no file exists",
				File:    f.Entry.Path, Line: f.LineNumber(ref.at),
				Evidence: ref.text, EffortMinutes: 10, Source: "skillgate",
			})
		})
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
		// Cycles are a progressive-disclosure smell in the *documentation*
		// chain, so the edge source stays loaded text — widening it here
		// would change what G002 reports.
		graph := resolveEdges(l, isLoadedText)
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

// ruleG003 — unreachable bundle file. The graph's third question: with the
// edges resolved, which files can an agent actually arrive at?
//
// Entry set: every inspected file a harness opens by convention
// (isHarnessEntry — SKILL.md, memory files, .mdc rules). Reachable set: the
// forward closure of those over reachabilityEdges. Candidate set: the
// inspected loaded-text files sitting in a *subdirectory* of some skill's
// root — the progressive-disclosure payload, which exists only to be
// pointed at. The finding set is candidates minus reachable: the complement
// of a computation, never a list of files that "look orphaned".
//
// Three consequences, each deliberate:
//
//   - A cycle is not reachability. An island of files referencing only each
//     other is reached by nobody, so every member is reported; a cycle
//     hanging off an entry point is reached through its entry edge and is
//     silent (it is G002's to mention, not ours).
//   - Edges may leave the referencing skill's directory. The audited unit is
//     the package, so a file another skill reaches is reachable. An edge
//     whose target leaves the package *root* resolves to no ledger file and
//     so contributes no edge at all — that reference is T019's.
//   - A missed edge is the dangerous direction: it accuses a file that is in
//     fact reached, in a spelling the resolver could not follow. So a
//     candidate mentioned by name in a file that produced no edge to it is
//     spared. The resolver's blind spot is not the skill's defect.
//
// Low, maintainability: dead weight and stale content, never a security
// claim and never a reason to REJECT.
var ruleG003 = rule{
	id: "SK-G003", sev: SeverityLow, quality: "maintainability", effort: 10,
	msg:   "unreachable file: no reference path from any harness entry point",
	files: nil,
	scanBundle: func(l *Ledger) []Finding {
		candidates := orphanCandidates(l)
		if len(candidates) == 0 {
			return nil
		}
		edges := reachabilityEdges(l)
		reached := map[string]bool{}
		var queue []string
		for i := range l.Files {
			if f := &l.Files[i]; f.Entry.Outcome == "inspected" && isHarnessEntry(f.Entry.Path) {
				if !reached[f.Entry.Path] {
					reached[f.Entry.Path] = true
					queue = append(queue, f.Entry.Path)
				}
			}
		}
		for len(queue) > 0 {
			v := queue[0]
			queue = queue[1:]
			for _, w := range edges[v] {
				if !reached[w] {
					reached[w] = true
					queue = append(queue, w)
				}
			}
		}
		var out []Finding
		for _, c := range candidates {
			if reached[c] || mentionedUnresolved(l, edges, c) {
				continue
			}
			out = append(out, Finding{
				RuleID: "SK-G003", Severity: SeverityLow, Quality: "maintainability",
				Message: "unreachable file: no reference path from any harness entry point",
				File:    c, Evidence: c, EffortMinutes: 10, Source: "skillgate",
			})
		}
		return out
	},
}

// orphanCandidates returns the files G003 may accuse, in ledger (sorted
// path) order: inspected loaded text, below a skill's own root rather than
// beside it. A harness entry point needs no exclusion here — it seeds the
// walk, so it is always in the reached set.
//
// The skill roots are derived from skillFiles, so a package with no
// SKILL.md has no candidates and the rule is silent — with no entry point
// there is no reachability question, and a docs tree is not a bundle of
// orphans. Files *beside* SKILL.md (README, CHANGELOG, a license) are skill
// furniture, not disclosure payload; scripts and other non-loaded-text
// files are a ceded lane, since the resolver reads references out of text
// and cannot see a script's own imports.
func orphanCandidates(l *Ledger) []string {
	var scopes []string
	for _, sf := range skillFiles(l) {
		scopes = append(scopes, skillScope(sf.Entry.Path))
	}
	var out []string
	for i := range l.Files {
		f := &l.Files[i]
		p := f.Entry.Path
		if f.Entry.Outcome != "inspected" || !isLoadedText(p) {
			continue
		}
		for _, s := range scopes {
			rest := p
			if s != "" {
				if !strings.HasPrefix(p, s+"/") {
					continue
				}
				rest = p[len(s)+1:]
			}
			if strings.Contains(rest, "/") {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

// mentionedUnresolved reports whether some other inspected file names the
// candidate's base name in text that produced no edge to it — the signature
// of a reference spelled in a form extractRefs does not follow. Mentions
// from files that *did* resolve an edge don't count: they are already in
// the graph, and counting them would spare every member of an island cycle.
func mentionedUnresolved(l *Ledger, edges map[string][]string, candidate string) bool {
	base := candidate[strings.LastIndex(candidate, "/")+1:]
	for i := range l.Files {
		f := &l.Files[i]
		if f.Entry.Outcome != "inspected" || f.Entry.Path == candidate {
			continue
		}
		if !strings.Contains(f.Text, base) {
			continue
		}
		resolved := false
		for _, w := range edges[f.Entry.Path] {
			if w == candidate {
				resolved = true
				break
			}
		}
		if !resolved {
			return true
		}
	}
	return false
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
