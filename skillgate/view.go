package skillgate

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Views — lexical rules run over normalised renderings of a file, never over
// raw text alone.
//
// The defect this closes: a rule that matches English phrasing against raw
// bytes is defeated by how the payload is *spelled*. `SK-T002` is blocker
// severity and `Ｉｇｎｏｒｅ ａｌｌ ｐｒｅｖｉｏｕｓ ｉｎｓｔｒｕｃｔｉｏｎｓ`
// walked past it, as did the Cyrillic homoglyph spelling. Answering that by
// adding fullwidth and Cyrillic alternations to ten regexes is the
// enumerate-the-forms defect; normalising the text before matching is the
// grammar.
//
// The property that makes this non-trivial: **a finding discovered on a
// derived view must report the raw source location.** A gate that reports an
// offset into a string the user cannot see is unusable, and one that silently
// points at the wrong line is worse than no report. So a view is not just
// transformed text — it carries the map from its own offsets back to the
// bytes that produced them.
//
// Contract, from docs/research/rule-language.md §4.3 and restated normatively
// in docs/skillgate-spec.md:
//
//   - a view is (name, text, source offsets), where the offset map answers
//     "which raw byte produced derived byte i";
//   - a hit found on a view is reported at the raw line the map names, with
//     the raw text as evidence, tagged with the view's name;
//   - a hit that cannot be anchored back to raw is dropped rather than
//     reported at an invented position;
//   - findings dedup **raw-wins**: the same rule at the same place in the
//     same file is reported once, preferring the raw discovery.
//
// Which rules run on views is derived, not listed: every rule with a `scan`
// func runs on every view unless it carries a written `rawOnly` reason. See
// ViewCoverage.

// View names. The raw view is always present and always first.
const (
	viewRaw      = "raw"
	viewSkeleton = "skeleton"
)

// View is one rendering of a file's text together with the map back to the
// raw bytes that produced it.
//
// Rules receive a View rather than a FileContent because that is what they
// actually scan: handing a rule a FileContent whose Text is derived but whose
// Entry describes the raw bytes would be a lie in the type, and the first
// rule to use one of the other fields would be silently wrong.
type View struct {
	// Name identifies the transform; viewRaw for the untransformed text.
	Name string
	// Path is the bundle-relative path of the file this is a view of.
	Path string
	// Text is the view's own text — what a rule matches against.
	Text string
	// segs maps Text offsets back to raw offsets. nil means identity, which
	// is what the raw view has.
	segs []viewSeg
	// rawLen is len(raw), the clamp for offsets past the end.
	rawLen int
}

// viewSeg maps a contiguous range of view bytes onto the raw bytes that
// produced it.
//
// A run of characters the transform left alone is one segment with linear
// set: derived and raw advance together, so the whole run costs one entry
// however long it is (a pure-ASCII file is a single segment). A character the
// transform rewrote gets its own segment: every derived byte in it came from
// the same raw character, so the mapping inside it is not positional and the
// segment reports its own bounds. A character the transform deleted produces
// no segment at all — the raw bytes it occupied simply belong to no derived
// byte, which is the honest answer.
type viewSeg struct {
	dStart, dEnd int // half-open range in View.Text
	sStart, sEnd int // half-open range in the raw text
	linear       bool
}

// SourceOffset returns the raw byte offset that produced view byte d.
//
// For an offset past the end of the view it returns the end of the raw text,
// so a caller mapping a half-open range never has to special-case the tail.
func (v *View) SourceOffset(d int) int {
	if v.segs == nil {
		return clamp(d, 0, v.rawLen)
	}
	i := v.segAt(d)
	if i < 0 {
		return 0
	}
	s := v.segs[i]
	if d >= s.dEnd {
		// d fell in a gap or past the last segment: the next raw byte
		// not yet consumed is this segment's end.
		if i+1 < len(v.segs) {
			return v.segs[i+1].sStart
		}
		return v.rawLen
	}
	if s.linear {
		return s.sStart + (d - s.dStart)
	}
	return s.sStart
}

// SourceEnd maps an exclusive view offset to an exclusive raw offset, so that
// SourceOffset(a):SourceEnd(b) brackets the raw text that produced Text[a:b].
//
// It is a separate method because the two questions genuinely differ inside a
// rewritten character: every derived byte of `Ｉ`→`I` starts at the same raw
// offset, but the range that produced it ends three bytes later, not one.
func (v *View) SourceEnd(d int) int {
	if v.segs == nil {
		return clamp(d, 0, v.rawLen)
	}
	if d <= 0 {
		return 0
	}
	i := v.segAt(d - 1)
	if i < 0 {
		return 0
	}
	s := v.segs[i]
	if s.linear && d <= s.dEnd {
		return s.sStart + (d - s.dStart)
	}
	return s.sEnd
}

// segAt returns the index of the last segment starting at or before d.
func (v *View) segAt(d int) int {
	if d < 0 {
		return -1
	}
	i := sort.Search(len(v.segs), func(i int) bool { return v.segs[i].dStart > d })
	return i - 1
}

func clamp(n, lo, hi int) int {
	if n < lo {
		return lo
	}
	if n > hi {
		return hi
	}
	return n
}

// viewBuilder is one registered transform. Adding a view is adding an entry
// here; every rule that has not opted out then runs on it, with no edit at
// the rule.
type viewBuilder struct {
	name  string
	build func(raw string) (string, []viewSeg)
}

// viewBuilders is the ordered registry of normalised views. Order is the
// order findings are discovered in, which matters only for which view's
// rendering survives dedup when two views find the same place.
var viewBuilders = []viewBuilder{
	{name: viewSkeleton, build: buildSkeleton},
}

// Views returns the raw view followed by every registered normalised view.
//
// Views are built once, when the ledger is built, because the check registry
// runs concurrently over a ledger it treats as read-only — building them
// lazily on first scan would be a write behind a shared pointer. A
// FileContent assembled outside BuildLedger has no prebuilt views and gets
// the raw view alone.
func (f *FileContent) Views() []*View {
	if f.views != nil {
		return f.views
	}
	return []*View{f.rawView()}
}

func (f *FileContent) rawView() *View {
	return &View{Name: viewRaw, Path: f.Entry.Path, Text: f.Text, rawLen: len(f.Text)}
}

// buildViews materialises every view of every inspected file. Called by
// BuildLedger after the file list is final, so the views are built once and
// read many times.
func (l *Ledger) buildViews() {
	for i := range l.Files {
		f := &l.Files[i]
		if f.Entry.Outcome != "inspected" {
			continue
		}
		views := make([]*View, 0, 1+len(viewBuilders))
		views = append(views, f.rawView())
		for _, b := range viewBuilders {
			text, segs := b.build(f.Text)
			views = append(views, &View{
				Name: b.name, Path: f.Entry.Path, Text: text, segs: segs, rawLen: len(f.Text),
			})
		}
		f.views = views
	}
}

// ViewNames returns every view the gate builds, raw first, in the order they
// are scanned.
//
// Derived from the registry, so the published list of views is the list that
// runs. Adding a view is one registry entry and the spec comparison in
// view_test.go then demands its row.
func ViewNames() []string {
	out := make([]string, 0, 1+len(viewBuilders))
	out = append(out, viewRaw)
	for _, b := range viewBuilders {
		out = append(out, b.name)
	}
	return out
}

// ViewRuleCoverage is one text-scanning rule's position on the view axis.
type ViewRuleCoverage struct {
	// ID is the rule identifier.
	ID string `json:"id"`
	// RawOnly is the written reason the rule runs on raw text alone, or
	// empty when it runs on every view.
	RawOnly string `json:"raw_only,omitempty"`
}

// ViewCoverage returns every rule that scans text, with the reason it is
// raw-only when it is one.
//
// This is the view axis of the ceded-lane sentence, and it is *derived*: a
// rule is view-covered unless it carries a reason not to be, so a rule added
// later is covered by default and a rule taken off the views cannot be taken
// off silently. Nothing here is a list anyone maintains — the registry is the
// list, and view_test.go compares this to the spec in both directions.
//
// Rules that only inspect cross-file state (scanBundle) have no view axis and
// are absent.
func ViewCoverage() []ViewRuleCoverage {
	var out []ViewRuleCoverage
	for _, gr := range tripwireGroups {
		for _, r := range gr.rules {
			if r.scan == nil {
				continue
			}
			out = append(out, ViewRuleCoverage{ID: r.id, RawOnly: r.rawOnly})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// IsRaw reports whether this is the untransformed view.
func (v *View) IsRaw() bool { return v.Name == viewRaw }

// Tag is the value a finding carries in its `view` field: empty for the raw
// view, so raw findings — every finding the gate emitted before views existed
// — are byte-identical in the report, and the view's name otherwise.
func (v *View) Tag() string {
	if v.IsRaw() {
		return ""
	}
	return v.Name
}

// ---- the Skeleton view ----

// buildSkeleton renders the Unicode fold: per-character NFKC, then the UTS #39
// ASCII confusable skeleton, then removal of format and control characters and
// default-ignorables.
//
// Every step is a Unicode property or a generated table, never a list of
// spellings someone thought of:
//
//   - NFKC closes the compatibility forms — fullwidth, halfwidth, circled,
//     superscript, ligature, and the rest of the block — as a class.
//   - The confusable skeleton (confusables.go, generated upstream from
//     Unicode 17 confusables.txt) closes homoglyphs as a class.
//   - Cf, non-whitespace Cc and Other_Default_Ignorable_Code_Point close the
//     invisible interleaving channel as a class. Whitespace controls are
//     kept, because line structure is what every rule's evidence is cut on.
//
// NFKC is applied per character rather than to the whole string, deliberately:
// whole-string NFKC composes across character boundaries, which destroys the
// one-character-to-one-span property the offset map is built on. The cost is
// that a combining sequence spelled as base + mark is not composed; that is a
// stated limit, not an oversight, and it costs nothing for the ASCII-phrase
// rules this view exists to serve.
func buildSkeleton(raw string) (string, []viewSeg) {
	var b strings.Builder
	b.Grow(len(raw))
	var segs []viewSeg

	// emit records one raw character's contribution. repl == "" deletes it.
	emit := func(repl string, sStart, sLen int, identity bool) {
		dStart := b.Len()
		b.WriteString(repl)
		if repl == "" {
			return
		}
		if identity && len(segs) > 0 {
			if last := &segs[len(segs)-1]; last.linear && last.dEnd == dStart && last.sEnd == sStart {
				last.dEnd += len(repl)
				last.sEnd += sLen
				return
			}
		}
		segs = append(segs, viewSeg{
			dStart: dStart, dEnd: dStart + len(repl),
			sStart: sStart, sEnd: sStart + sLen,
			linear: identity,
		})
	}

	for i := 0; i < len(raw); {
		r, n := utf8.DecodeRuneInString(raw[i:])
		if r == utf8.RuneError && n <= 1 {
			// Invalid UTF-8: pass the byte through untouched. Normalising
			// bytes that are not characters would invent text.
			emit(raw[i:i+1], i, 1, true)
			i++
			continue
		}
		if r < utf8.RuneSelf {
			// ASCII is already NFKC and has no confusable mapping; only the
			// C0/DEL controls are dropped, and not the whitespace ones.
			if isStrippableControl(r) {
				emit("", i, n, false)
			} else {
				emit(raw[i:i+n], i, n, true)
			}
			i += n
			continue
		}
		folded := foldRune(r)
		emit(folded, i, n, folded == raw[i:i+n])
		i += n
	}
	return b.String(), segs
}

// foldRune is NFKC then confusable-skeleton then ignorable-removal, for one
// character. It returns the replacement text, which may be empty, one
// character, or several.
func foldRune(r rune) string {
	var b strings.Builder
	for _, c := range norm.NFKC.String(string(r)) {
		if rep, ok := asciiConfusableSkeleton[c]; ok {
			b.WriteString(rep)
			continue
		}
		if isStrippableControl(c) {
			continue
		}
		b.WriteRune(c)
	}
	return b.String()
}

// isStrippableControl reports whether a character carries no visible text and
// is therefore an interleaving channel rather than content: format characters,
// default-ignorables, and the non-whitespace controls. Whitespace controls —
// newline, tab, carriage return, NEL — are content: they carry line structure.
func isStrippableControl(r rune) bool {
	if unicode.IsSpace(r) {
		return false
	}
	return unicode.Is(unicode.Cf, r) ||
		unicode.Is(unicode.Cc, r) ||
		unicode.Is(unicode.Other_Default_Ignorable_Code_Point, r)
}
