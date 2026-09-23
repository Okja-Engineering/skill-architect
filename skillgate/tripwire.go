package skillgate

import (
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
	// scan inspects one view of one file and returns what it found, each as
	// a span of the view's own text.
	//
	// It receives a View rather than a FileContent because a view is what a
	// lexical rule scans: the same rule runs over the raw text and over each
	// normalised rendering of it, and run() maps whatever it finds back to
	// the raw source.
	scan func(v *View) []hit
	// rawOnly, when non-empty, is the written reason this rule must not run
	// on normalised views — it runs on raw text alone. Empty, the default,
	// means the rule runs on every registered view, so a view added later
	// covers it with no edit here and a rule added later is view-covered by
	// construction. ViewCoverage() derives the published list from this
	// field; the reason is the whole justification, so there is no way to
	// opt out silently.
	rawOnly string
	// codeOnly, when non-empty, is the written reason this rule reads only
	// the parts of a file its interpreter executes — commentary is blanked
	// before the scan. Empty, the default, means the rule reads the whole
	// document, which is right for every rule whose subject is what a file
	// *says* rather than what it *does*.
	//
	// Same shape as rawOnly and for the same reason: the reason is the whole
	// justification, ViewCoverage publishes it, and the spec table is
	// compared to it in both directions, so a rule cannot quietly give up
	// reach. The two axes are genuinely different questions — rawOnly is
	// "in what spelling", codeOnly is "in what part" — and a rule may
	// answer either, both, or neither.
	codeOnly string
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
			if r.codeOnly != "" {
				v = v.code()
			}
			var batch []Finding
			for _, h := range r.scan(v) {
				line, evidence, ok := locate(f, v, h)
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

// hit is one thing a rule found in the view it was handed.
//
// A hit is a **span of the view's own text**, and that is the whole contract:
// a span is what run() can map back to the raw source, so a hit reports the
// line of the file it came from and evidence cut from the file as it is
// written on disk.
//
// Rules used to return evidence *strings*, and run() anchored one by searching
// the view for it. That works only while a rule matches against the exact text
// it was given — and five rules did not. A rule that needed a separator-
// normalised line, or one with `/dev/null` or a loopback URL masked out, built
// its own copy, matched that, and returned a slice **of the copy**. The copy's
// bytes are not in the view, so the search failed: on a derived view the hit
// was dropped and the rule silently lost the coverage it published, and on the
// raw view it was reported at line 0 carrying text that appears in no file.
// One backslash, one `2>/dev/null`, turned a blocker off.
//
// A span cannot fail that way. A rule may still read a different rendering of
// its line — it just may not *move the bytes* while doing so, because the
// offsets are what it reports. Masking is done in place, at equal length, and
// the reading is used for the match while the span comes from the view.
type hit struct {
	// at, end are the half-open byte span of View.Text the rule matched.
	// at < 0 means the rule has no span: see synthesised.
	at, end int
	// text, when non-empty, is the evidence to publish in place of the raw
	// bytes the span covers. A rule composes evidence when the raw text is
	// the wrong thing to print — a masked credential, or a statement about a
	// parsed structure rather than a quotation of it. The span still decides
	// *where* the finding is, which is what keeps the rule view-covered.
	text string
}

// found reports a hit at a span of the view's text, with the raw text of that
// span as its evidence.
func found(at, end int) hit { return hit{at: at, end: end} }

// foundAs reports a hit at a span of the view's text, publishing composed
// evidence instead of quoting the span.
func foundAs(at, end int, text string) hit { return hit{at: at, end: end, text: text} }

// synthesised reports a hit with no position at all: evidence the rule
// composed out of text that is nowhere contiguous in the file — T005's and
// T009's pair of distant lines, T006's masked secret.
//
// It is the one shape that cannot be anchored, and it is exactly the shape
// docs/skillgate-spec.md cedes: those three rules are raw-only *because* of
// it, and they report at line 0. A rule that is not raw-only must never
// return one — that is how view coverage is lost silently, and
// anchor_test.go fails on a fourth rule appearing at line 0.
func synthesised(text string) hit { return hit{at: -1, text: text} }

// locate turns a hit into a raw source position and the evidence to publish.
//
// The raw view and a derived view take the same path, because a raw view's
// offset map is the identity: a span is a span. The only branch is the
// positionless hit, which the raw view reports at line 0 with its composed
// text — the behaviour the gate has always had for the three rules that
// produce one — and which a derived view drops, because a finding at line 0
// citing text that is in no file is not a report, it is noise.
func locate(f *FileContent, v *View, h hit) (line int, evidence string, ok bool) {
	if h.at < 0 {
		if v.IsRaw() {
			return 0, h.text, true
		}
		return 0, "", false
	}
	start, end := v.SourceOffset(h.at), v.SourceEnd(h.end)
	if start > end || end > len(f.Text) {
		return 0, "", false
	}
	if h.text != "" {
		return f.LineNumber(start), h.text, true
	}
	return f.LineNumber(start), strings.TrimSpace(f.Text[start:end]), true
}

// eachLine calls fn for every line of text with its half-open byte span,
// excluding the line terminator. It yields exactly the pieces
// strings.Split(text, "\n") does, with the offset each piece came from — which
// is the whole reason it exists: a rule that reports a span cannot cut its
// lines out of a copy that has forgotten where they were.
func eachLine(text string, fn func(start, end int, line string)) {
	start := 0
	for {
		i := strings.IndexByte(text[start:], '\n')
		if i < 0 {
			fn(start, len(text), text[start:])
			return
		}
		fn(start, start+i, text[start:start+i])
		start += i + 1
	}
}

// lineSpanAt returns the half-open span of the line containing offset.
func lineSpanAt(text string, offset int) (int, int) {
	start := strings.LastIndexByte(text[:offset], '\n') + 1
	end := strings.IndexByte(text[offset:], '\n')
	if end < 0 {
		end = len(text)
	} else {
		end += offset
	}
	return start, end
}

// maskByte is what a mask writes over a token a rule has decided is not its
// subject.
//
// It is one byte per byte, so masking preserves every offset in the text and
// a hit found in the masked reading is still a span of the view. And it is a
// *word* character, so masking can neither create nor destroy a word boundary
// at the token's edges — a mask that blanked to spaces would introduce a `\b`
// in the middle of an identifier and could manufacture a match that the
// unmasked line does not contain.
const maskByte = 'X'

// maskSpans returns text with every match of re overwritten by maskByte,
// leaving every other byte and every offset exactly where it was.
//
// This is how a rule says "this token is not my subject" — a loopback URL is
// not a remote host; `2>/dev/null` discards output rather than persisting to
// it. Saying it by *substitution* was the defect: the replacement had a
// different length, so the line the rule then cut was not a line of the view
// and could not be anchored.
func maskSpans(text string, re *regexp.Regexp) string {
	locs := re.FindAllStringIndex(text, -1)
	if locs == nil {
		return text
	}
	b := []byte(text)
	for _, m := range locs {
		for i := m[0]; i < m[1]; i++ {
			b[i] = maskByte
		}
	}
	return string(b)
}

func tripwireChecks() []Check {
	var checks []Check
	for _, gr := range tripwireGroups {
		rules := gr.rules
		checks = append(checks, Check{
			Name: gr.name, Stage: stageScan,
			Run: func(in checkInput) checkResult {
				var out []Finding
				for _, r := range rules {
					out = append(out, r.run(in.Ledger)...)
				}
				return checkResult{Findings: out}
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

// pathRefs returns the references in a view that a `..` segment could carry
// outside the bundle, each with the offset it was found at.
//
// The distinction this draws is the one refgraph already drew and wrote
// down, in the same words, for the same reason: *a slash-containing token is
// path-shaped anywhere; a bare token is a concept unless it is an explicit
// link target* (`refTokens`, "`SKILL.md` in prose is a concept"). T019 never
// took delivery of it — refgraph hands it the escape question in three
// separate comments while T019 answers with its own naked regex — so `..`
// standing alone in a sentence was read as a path to the parent directory.
//
// It is one path *segment*, and a sentence that quotes it names nowhere.
// Measured on this repository: every T019 false positive was a bare `..` in
// prose, including the spec row S09 wrote to explain why T019 is raw-only,
// and every true positive carried a separator. The one bare form that really
// is a reference — a link whose whole target is the parent — is picked up by
// the explicit leg, which widens the rule rather than narrowing it.
func pathRefs(text string) []struct {
	at  int
	ref string
} {
	var out []struct {
		at  int
		ref string
	}
	add := func(at int, ref string) {
		out = append(out, struct {
			at  int
			ref string
		}{at, normPathSep(ref)})
	}
	for _, m := range rePathRef.FindAllStringIndex(text, -1) {
		if ref := normPathSep(text[m[0]:m[1]]); strings.Contains(ref, "/") {
			add(m[0], ref)
		}
	}
	// A markdown link target is a reference by construction, whatever its
	// shape: `[parent](..)` points out of the bundle and says so.
	for _, m := range reMdLink.FindAllStringSubmatchIndex(text, -1) {
		add(m[2], text[m[2]:m[3]])
	}
	return out
}

// ruleT019 — path escape: a `../` reference that *resolves outside* the
// bundle root, or a symlink whose target leaves it (the ledger records the
// latter as symlink_escape skips). References that stay inside the bundle —
// e.g. references/x.md → ../SKILL.md — are legal and must not fire.
//
// It reads the whole document, not the executed part: `# see ../other.md`
// in a comment is still a reference to a file outside the bundle, and the
// dependency is just as real for being documented. Its subject is what the
// bundle *points at*, which is why the codeOnly narrowing SK-T010 takes is
// wrong here.
//
// Not raw-only. It was, and the written reason was that NFKC folds U+2025
// TWO DOT LEADER onto `..` and a bare `‥` in prose fired this blocker on the
// skeleton view. pathRefs removed the premise — a bare `..` is one path
// segment and names nowhere — so the fold has nothing to manufacture, and
// the limit that opt-out had to accept is closed: a fullwidth-spelled climb
// is caught again. Both directions are pinned in rulesubject_test.go and the
// whole-repo differential over 330 files moved by zero lines.
//
// It does not model a process's working directory and never did: `cd ..` is
// not a reference and is not reported.
var ruleT019 = rule{
	id: "SK-T019", sev: SeverityBlocker, quality: "security", effort: 15,
	msg: "path escape: reference resolves outside the bundle root",
	scan: func(v *View) []hit {
		dir := v.Path
		if i := strings.LastIndex(dir, "/"); i >= 0 {
			dir = dir[:i]
		} else {
			dir = ""
		}
		var ev []hit
		// Deduped on the line's *text*, not its position, which is what this
		// rule has always done: the same escaping line written twice is one
		// reference reported once, at its first occurrence.
		seen := map[string]bool{}
		for _, r := range pathRefs(v.Text) {
			// resolveRef, not a second copy of it. The gate had two
			// resolvers that disagreed about a leading `/`: refgraph reads
			// it as package-root absolute, this rule read it as a relative
			// continuation, and the difference is a hole that widens with
			// the referencing file's depth. Measured — from `scripts/`,
			// `"$(dirname "$d")/../skill-audit"` resolved to `skill-audit`
			// and the escape went unreported, which is why this rule named
			// draft-rewrite.sh's shellcheck comment and not line 49, the
			// live code that performs the climb. The two readings can only
			// differ when the climb escapes, so agreeing with refgraph
			// widens the rule and never narrows it.
			if _, escaped := resolveRef(dir, r.ref); !escaped {
				continue // resolves inside the bundle — legal
			}
			start, end := lineSpanAt(v.Text, r.at)
			if line := strings.TrimSpace(v.Text[start:end]); !seen[line] {
				seen[line] = true
				ev = append(ev, found(start, end))
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
