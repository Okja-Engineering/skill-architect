package skillgate

import (
	"path"
	"regexp"
	"strings"
)

// Pack B — the tripwire floor. SK-T001..T020, hard-capped at 20 rules. These
// are the checks that must block a dangerous bundle with no Python and no
// Rust present; deeper coverage (SkillSpector's 113 IDs, agnix's 455) is
// delegated to opt-in shell-outs that degrade to checks_skipped.
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
	// scan inspects one file's text and returns evidence snippets.
	scan func(f *FileContent) []string
	// scanBundle inspects cross-file state (e.g. config files, symlinks).
	scanBundle func(l *Ledger) []Finding
	// files limits which bundle paths the scan applies to (empty = all
	// inspected text files).
	files func(path string) bool
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
		for _, ev := range r.scan(f) {
			line := 0
			if off := strings.Index(f.Text, ev); off >= 0 {
				line = f.LineNumber(off)
			}
			out = append(out, Finding{
				RuleID:        r.id,
				Severity:      r.sev,
				Quality:       r.quality,
				Message:       r.msg,
				File:          f.Entry.Path,
				Line:          line,
				Evidence:      truncate(ev, 160),
				EffortMinutes: r.effort,
				Source:        "skillgate",
			})
		}
	}
	return out
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
	scan: func(f *FileContent) []string {
		dir := f.Entry.Path
		if i := strings.LastIndex(dir, "/"); i >= 0 {
			dir = dir[:i]
		} else {
			dir = ""
		}
		var ev []string
		seen := map[string]bool{}
		for _, m := range rePathRef.FindAllStringIndex(f.Text, -1) {
			ref := strings.ReplaceAll(f.Text[m[0]:m[1]], "\\", "/")
			joined := path.Join(dir, ref)
			if joined != ".." && !strings.HasPrefix(joined, "../") {
				continue // resolves inside the bundle — legal
			}
			line := lineAt(f.Text, m[0])
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
