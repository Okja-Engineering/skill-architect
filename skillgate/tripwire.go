package skillgate

import (
	"path"
	"regexp"
	"strings"
)

// Pack B — the tripwire floor, hard-capped at 20 rules. These are the checks
// that must block a dangerous bundle with no Python and no Rust present;
// deeper coverage (SkillSpector's 113 IDs, agnix's 455) is delegated to
// opt-in shell-outs that degrade to checks_skipped.
//
// Which rules those are is tripwireGroups below, and RuleCatalog() reads it —
// not a range written out here, which is the comment that goes on naming a
// rule after it is gone. The cap is enforced in catalog_test.go.
//
// Every rule ships two tests: fires on a real malicious fixture and does not
// fire on the clean corpus.

// rule is one tripwire check. match returns evidence strings; each becomes a
// finding at the rule's severity.
type rule struct {
	id      string
	sev     string
	quality string
	effort  int
	msg     string
	// scan inspects one view of one file and returns evidence snippets.
	// It receives a View rather than a FileContent because a view is what a
	// lexical rule scans: the same rule runs over the raw text and over each
	// normalised rendering of it, and run() maps whatever it finds back to
	// the raw source.
	scan func(v *View) []string
	// rawOnly, when non-empty, is the written reason this rule must not run
	// on normalised views — it runs on raw text alone. Empty, the default,
	// means the rule runs on every registered view, so a view added later
	// covers it with no edit here and a rule added later is view-covered by
	// construction. ViewCoverage() derives the published list from this
	// field; the reason is the whole justification, so there is no way to
	// opt out silently.
	rawOnly string
	// scanBundle inspects cross-file state (e.g. config files, symlinks).
	scanBundle func(l *Ledger) []Finding
	// files limits which bundle paths the scan applies to (empty = all
	// inspected text files).
	files func(path string) bool
}

// views returns the views this rule scans for a file: every view, or the raw
// one alone when the rule has opted out with a reason.
func (r rule) views(f *FileContent) []*View {
	all := f.Views()
	if r.rawOnly == "" {
		return all
	}
	return all[:1] // the raw view is always first
}

func (r rule) run(l *Ledger) []Finding {
	var out []Finding
	if r.scanBundle != nil {
		out = append(out, r.scanBundle(l)...)
	}
	if r.scan == nil {
		return out
	}
	for i := range l.Files {
		f := &l.Files[i]
		if f.Entry.Outcome != "inspected" {
			continue
		}
		if r.files != nil && !r.files(f.Entry.Path) {
			continue
		}
		// Dedup is *between* views, raw first and raw wins: a derived view
		// rediscovering a raw hit is the same finding wearing a different
		// spelling, and reporting it twice would double every existing
		// finding the day a view was added. Within one view nothing is
		// dropped, so a rule that reports two unlocatable hits in the same
		// file still reports both, exactly as it did before views existed.
		reported := map[int]bool{}
		for _, v := range r.views(f) {
			var batch []Finding
			for _, ev := range r.scan(v) {
				line, evidence, ok := locate(f, v, ev)
				if !ok || reported[line] {
					continue
				}
				batch = append(batch, Finding{
					RuleID:        r.id,
					Severity:      r.sev,
					Quality:       r.quality,
					Message:       r.msg,
					File:          f.Entry.Path,
					Line:          line,
					Evidence:      truncate(evidence, 160),
					View:          v.Tag(),
					EffortMinutes: r.effort,
					Source:        "skillgate",
				})
			}
			for _, fd := range batch {
				reported[fd.Line] = true
			}
			out = append(out, batch...)
		}
	}
	return out
}

// locate turns evidence a rule found in a view into a raw source position and
// raw evidence.
//
// On the raw view this is the behaviour the gate has always had, including
// the case where a rule synthesises evidence that is not a substring of the
// file (T005 and T009 report a pair of lines): those report at line 0 and
// carry their synthesised text, unchanged.
//
// On a derived view that case cannot be reported at all — evidence that is
// nowhere in the view cannot be mapped back to raw, and a finding at line 0
// citing text that is in no file is not a report, it is noise. Such a hit is
// dropped; the raw view still finds it. This is the only coverage a view
// loses and docs/skillgate-spec.md states it.
func locate(f *FileContent, v *View, ev string) (line int, evidence string, ok bool) {
	off := strings.Index(v.Text, ev)
	if v.IsRaw() {
		if off < 0 {
			return 0, ev, true
		}
		return f.LineNumber(off), ev, true
	}
	if off < 0 {
		return 0, "", false
	}
	start, end := v.SourceOffset(off), v.SourceEnd(off+len(ev))
	if start > end || end > len(f.Text) {
		return 0, "", false
	}
	return f.LineNumber(start), strings.TrimSpace(f.Text[start:end]), true
}

func tripwireChecks() []Check {
	var checks []Check
	for _, gr := range tripwireGroups {
		rules := gr.rules
		checks = append(checks, Check{
			Name: gr.name,
			Run: func(l *Ledger, _ *Target) []Finding {
				var out []Finding
				for _, r := range rules {
					out = append(out, r.run(l)...)
				}
				return out
			},
		})
	}
	return checks
}

// tripwireGroup is a named bundle of rules — one entry in checks_skipped if
// it cannot run.
type tripwireGroup struct {
	name  string
	rules []rule
}

var tripwireGroups = []tripwireGroup{
	{name: "tripwire-injection", rules: []rule{ruleT001, ruleT002, ruleT003}},
	{name: "tripwire-exfil", rules: []rule{ruleT004, ruleT005, ruleT006, ruleT007, ruleT008, ruleT009}},
	{name: "tripwire-snoop", rules: []rule{ruleT010, ruleT011, ruleT012}},
	{name: "tripwire-agency", rules: []rule{ruleT013, ruleT014}},
	{name: "tripwire-config", rules: []rule{ruleT015, ruleT016, ruleT017, ruleT018}},
	{name: "tripwire-integrity", rules: []rule{ruleT019, ruleT020}},
	{name: "refgraph", rules: []rule{ruleG001, ruleG002, ruleG003}},
}

// --- helpers shared by the rule implementations ---

// rePathRef captures a whole path-ish token containing a `..` segment.
var rePathRef = regexp.MustCompile(`[\w.\-/\\]*\.\.([\\/][\w.\-/\\]*)?`)

// ruleT019 — path escape: a `../` reference that *resolves outside* the
// bundle root, or a symlink whose target leaves it (the ledger records the
// latter as symlink_escape skips). References that stay inside the bundle —
// e.g. references/x.md → ../SKILL.md — are legal and must not fire.
var ruleT019 = rule{
	id: "SK-T019", sev: SeverityBlocker, quality: "security", effort: 15,
	msg: "path escape: reference resolves outside the bundle root",
	rawOnly: "its subject is path syntax, and a normalised view manufactures path syntax " +
		"out of prose: NFKC deliberately maps typographic punctuation onto ASCII, so " +
		"U+2025 TWO DOT LEADER becomes `..`. Measured — a bare `‥` in an ordinary " +
		"sentence fires this blocker on the Skeleton view. The cost is that a " +
		"fullwidth-spelled `../` escape is not caught; a false blocker against correct " +
		"prose is the worse direction, which is the ruling SK-G003 already took",
	scan: func(v *View) []string {
		dir := v.Path
		if i := strings.LastIndex(dir, "/"); i >= 0 {
			dir = dir[:i]
		} else {
			dir = ""
		}
		var ev []string
		seen := map[string]bool{}
		for _, m := range rePathRef.FindAllStringIndex(v.Text, -1) {
			ref := strings.ReplaceAll(v.Text[m[0]:m[1]], "\\", "/")
			joined := path.Join(dir, ref)
			if joined != ".." && !strings.HasPrefix(joined, "../") {
				continue // resolves inside the bundle — legal
			}
			line := lineAt(v.Text, m[0])
			if !seen[line] {
				seen[line] = true
				ev = append(ev, line)
			}
		}
		return ev
	},
	scanBundle: func(l *Ledger) []Finding {
		var out []Finding
		for _, f := range l.Files {
			if f.Entry.Reason == SkipSymlinkEscape {
				out = append(out, Finding{
					RuleID: "SK-T019", Severity: SeverityBlocker, Quality: "security",
					Message: "symlink target escapes the bundle root",
					File:    f.Entry.Path, EffortMinutes: 15, Source: "skillgate",
				})
			}
		}
		return out
	},
}

func lineAt(text string, offset int) string {
	start := strings.LastIndex(text[:offset], "\n") + 1
	end := strings.Index(text[offset:], "\n")
	if end < 0 {
		end = len(text)
	} else {
		end += offset
	}
	return strings.TrimSpace(text[start:end])
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// normPathSep folds Windows separators so path regexes can't be evaded by
// `.cursor\hooks.json`-style spelling.
func normPathSep(s string) string { return strings.ReplaceAll(s, "\\", "/") }
