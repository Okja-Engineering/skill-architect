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
	viewRaw           = "raw"
	viewSkeleton      = "skeleton"
	viewCompactLetter = "compactLetter"
	viewMarkup        = "markup"
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

// viewStage is one transform in a view's pipeline: it takes the text of the
// stage before it and returns its own text with the map back to that text's
// offsets. A stage that changes nothing returns a nil map, which is the same
// identity convention View.segs uses.
//
// Views are pipelines rather than monoliths because that is what they are:
// the compact view *is* the skeleton fold with letter-spacing separators
// removed, and writing it as `{foldSkeleton, compactLetterSpacing}` is the
// contract rather than a paraphrase of it. composeSegs is what lets a stage
// be written against the text it actually sees while its findings still
// anchor to raw.
type viewStage func(text string) (string, []viewSeg)

// viewBuilder is one registered view. Adding a view is adding an entry here;
// every rule that has not opted out then runs on it, with no edit at the rule.
type viewBuilder struct {
	name   string
	stages []viewStage
}

// build runs the pipeline over a file's raw text, composing each stage's
// offset map onto the one before it, so the view that comes out maps its own
// offsets straight back to raw however many stages produced it.
func (b viewBuilder) build(raw string) (string, []viewSeg) {
	text := raw
	var segs []viewSeg
	for _, stage := range b.stages {
		next, stageSegs := stage(text)
		segs = composeSegs(stageSegs, segs, len(raw))
		text = next
	}
	return text, segs
}

// viewBuilders is the ordered registry of normalised views. Order is the
// order findings are discovered in, which matters only for which view's
// rendering survives dedup when two views find the same place.
var viewBuilders = []viewBuilder{
	{name: viewSkeleton, stages: []viewStage{foldSkeleton}},
	{name: viewCompactLetter, stages: []viewStage{foldSkeleton, compactLetterSpacing}},
	{name: viewMarkup, stages: []viewStage{foldSkeleton, stripInlineMarkup}},
}

// composeSegs composes two offset maps: outer maps a stage's own offsets onto
// the text it was given, inner maps that text's offsets onto raw. The result
// maps the stage's offsets onto raw.
//
// A nil map means the identity, exactly as View.segs does, so a stage that
// changed nothing costs nothing here.
//
// A linear outer segment is split at the inner map's boundaries rather than
// carried whole: inside the range it covers, the inner map may be linear in
// one part and a rewrite in the next, and a composition that kept the outer
// segment's linearity would interpolate positions through a character that
// has no interior.
func composeSegs(outer, inner []viewSeg, rawLen int) []viewSeg {
	if outer == nil {
		return inner
	}
	if inner == nil {
		return outer
	}
	in := &View{segs: inner, rawLen: rawLen}
	var out segList
	for _, o := range outer {
		if !o.linear {
			// The whole of o's output came from o's input range as a unit;
			// that range's raw bounds are the composition.
			out.add(viewSeg{
				dStart: o.dStart, dEnd: o.dEnd,
				sStart: in.SourceOffset(o.sStart), sEnd: in.SourceEnd(o.sEnd),
			})
			continue
		}
		i := in.segAt(o.sStart)
		if i < 0 {
			i = 0
		}
		for ; i < len(inner) && inner[i].dStart < o.sEnd; i++ {
			s := inner[i]
			lo, hi := max(s.dStart, o.sStart), min(s.dEnd, o.sEnd)
			if lo >= hi {
				continue
			}
			seg := viewSeg{dStart: o.dStart + (lo - o.sStart), dEnd: o.dStart + (hi - o.sStart)}
			if s.linear {
				seg.sStart, seg.sEnd, seg.linear = s.sStart+(lo-s.dStart), s.sStart+(hi-s.dStart), true
			} else {
				seg.sStart, seg.sEnd = s.sStart, s.sEnd
			}
			out.add(seg)
		}
	}
	return out.segs
}

// segList accumulates segments, coalescing a linear segment onto the linear
// segment before it when the two are contiguous on both sides — so a stretch
// the pipeline left alone costs one entry however long it is.
type segList struct{ segs []viewSeg }

func (sl *segList) add(s viewSeg) {
	if s.linear && len(sl.segs) > 0 {
		if last := &sl.segs[len(sl.segs)-1]; last.linear && last.dEnd == s.dStart && last.sEnd == s.sStart {
			last.dEnd, last.sEnd = s.dEnd, s.sEnd
			return
		}
	}
	sl.segs = append(sl.segs, s)
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

// newViews builds the raw view and every registered view of one file's text.
// It is the single place the registry is walked, so a view the ledger builds
// and a view a test builds are the same object by construction.
func newViews(path, raw string) []*View {
	views := make([]*View, 0, 1+len(viewBuilders))
	views = append(views, &View{Name: viewRaw, Path: path, Text: raw, rawLen: len(raw)})
	for _, b := range viewBuilders {
		text, segs := b.build(raw)
		views = append(views, &View{
			Name: b.name, Path: path, Text: text, segs: segs, rawLen: len(raw),
		})
	}
	return views
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
		f.views = newViews(f.Entry.Path, f.Text)
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

// ---- building a stage's text and its offset map ----

// segBuilder accumulates a stage's output text together with the map back to
// the text it was given. Both stages build their map through it, so there is
// one implementation of the three cases a transform can produce.
type segBuilder struct {
	b    strings.Builder
	segs segList
}

// emit records one contribution of the input to the output.
//
//   - repl == "" is a deletion, and produces no segment at all: the input
//     bytes it occupied belong to no output byte, which is the honest answer
//     rather than a fabricated one.
//   - identity means the output is the input verbatim, so consecutive
//     identity contributions coalesce into one linear segment however long
//     the run — a stage that changes one character in a megabyte costs three
//     segments, not a million.
//   - otherwise the contribution is a rewrite and gets its own segment:
//     every output byte in it came from the same input range, so the mapping
//     inside it is not positional and the segment reports its own bounds.
func (sb *segBuilder) emit(repl string, sStart, sLen int, identity bool) {
	dStart := sb.b.Len()
	sb.b.WriteString(repl)
	if repl == "" {
		return
	}
	sb.segs.add(viewSeg{
		dStart: dStart, dEnd: dStart + len(repl),
		sStart: sStart, sEnd: sStart + sLen,
		linear: identity,
	})
}

func (sb *segBuilder) done() (string, []viewSeg) { return sb.b.String(), sb.segs.segs }

// ---- the Skeleton view ----

// foldSkeleton renders the Unicode fold: per-character NFKC, then the UTS #39
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
func foldSkeleton(raw string) (string, []viewSeg) {
	var sb segBuilder
	sb.b.Grow(len(raw))

	for i := 0; i < len(raw); {
		r, n := utf8.DecodeRuneInString(raw[i:])
		if r == utf8.RuneError && n <= 1 {
			// Invalid UTF-8: pass the byte through untouched. Normalising
			// bytes that are not characters would invent text.
			sb.emit(raw[i:i+1], i, 1, true)
			i++
			continue
		}
		if r < utf8.RuneSelf {
			// ASCII is already NFKC and has no confusable mapping; only the
			// C0/DEL controls are dropped, and not the whitespace ones.
			if isStrippableControl(r) {
				sb.emit("", i, n, false)
			} else {
				sb.emit(raw[i:i+n], i, n, true)
			}
			i += n
			continue
		}
		folded := foldRune(r)
		sb.emit(folded, i, n, folded == raw[i:i+n])
		i += n
	}
	return sb.done()
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

// ---- the CompactLetter view ----
//
// The channel this closes is *visible* interleaving: `i g n o r e  a l l
// p r e v i o u s  i n s t r u c t i o n s` is the blocker-severity SK-T002
// payload, and no amount of Unicode folding reaches it, because every
// character in it is already exactly what it looks like. The skeleton view
// removes the *invisible* interleaving channel — Cf, the non-whitespace Cc,
// the default-ignorables — and it can remove them everywhere, unconditionally,
// because an invisible character is never content. A visible character is
// content, so removing one needs evidence that interleaving is happening at
// all. That evidence is the run.
//
// **The rule is a grammar, not a list of spellings.** A letter-spacing run is
// six or more isolated letters in a row — each one separated from the next by
// a non-empty gap that contains no letter and no digit. Prose is not written
// that way; letter-spaced payloads are, whatever they are spelled with. So
// the separator is not enumerated: space, NBSP, `.`, `-`, `_`, `*`, a
// non-ASCII Z-separator, U+FFFD, and the filler nobody has thought of yet are
// all simply "not a letter", and all close together or none does.
//
// This is where the gate parts company with upstream's AE6
// (docs/research/rule-language.md §2, ss:artifacts.py:1662-1700), which
// carries four enumerations beside the same run detector: a security-term
// list, four command-phrase alternations, a benign-term list and a pair of
// spelling-example context regexes. It needs them because AE6 *emits a
// finding* from the run itself, so it has to decide whether the run spells
// something dangerous. A view emits nothing. It changes what the rules can
// see, and the rules already hold the vocabulary — so the whole enumeration
// layer has no job here and is not ported.
//
// What compaction may *not* do is as load-bearing as what it does:
//
//   - **It never crosses a line break.** Letter-spacing is a within-line
//     typographic device, and line structure is what every rule's evidence is
//     cut on: a compaction that swallowed a newline would move every finding
//     after it and would let two lines' text form a match that is in neither.
//   - **It only ever brings letters together.** Gaps inside a run are
//     removed or become one space; text outside a run is untouched. So,
//     unlike a fold, it cannot manufacture syntax out of prose — the failure
//     the skeleton view measured when NFKC turned a bare `‥` into `..` and
//     fired a path-traversal blocker.

// compactMinRunLetters is the shortest sequence of isolated letters the view
// reads as letter-spacing rather than as prose. Upstream's AE6 uses the same
// bound (rule-language.md §2, ss:artifacts.py:204). It is a threshold on the
// shape of the run, not a vocabulary: nothing here knows or asks what the run
// might spell.
const compactMinRunLetters = 6

// letterSpacingRun is one maximal stretch of isolated letters, in rune
// indices, half-open and always ending on a letter.
type letterSpacingRun struct{ start, end int }

// compactLetterSpacing removes the separators inside letter-spacing runs.
//
// Within a run, the *narrowest* gap is the unit separator — the one that
// stands between the letters of a word — and it is removed. A wider gap is
// where the words divide and becomes a single space, because that is what a
// wider gap means: letter-spaced text has always set the word boundary wider
// than the letter boundary, since a word boundary rendered the same as a
// letter boundary is unreadable. Taking the unit from the run rather than
// fixing it at one character is what makes `i  g  n  o  r  e    a  l  l`
// compact as correctly as `i g n o r e  a l l`, with no constant to tune.
//
// Preserving the word boundary is not a nicety: the gate's phrase rules are
// written with `\s+` between their words, so a compaction that collapsed the
// whole run to `ignoreallpreviousinstructions` would close nothing at all.
func compactLetterSpacing(text string) (string, []viewSeg) {
	rs, bo := runeOffsets(text)
	runs := letterSpacingRuns(rs)
	if len(runs) == 0 {
		// Nothing interleaved: the identity, with the nil map that says so.
		return text, nil
	}

	var sb segBuilder
	sb.b.Grow(len(text))
	pass := func(from, to int) { // rune range, verbatim
		if from < to {
			sb.emit(text[bo[from]:bo[to]], bo[from], bo[to]-bo[from], true)
		}
	}

	prev := 0
	for _, run := range runs {
		pass(prev, run.start)
		unit := unitGap(rs, run)
		for i := run.start; i < run.end; {
			if isRunLetter(rs[i]) {
				pass(i, i+1)
				i++
				continue
			}
			gap := i
			for i < run.end && !isRunLetter(rs[i]) {
				i++
			}
			repl := ""
			if i-gap > unit {
				repl = " "
			}
			sb.emit(repl, bo[gap], bo[i]-bo[gap], false)
		}
		prev = run.end
	}
	pass(prev, len(rs))
	return sb.done()
}

// letterSpacingRuns finds every maximal run of isolated letters of at least
// compactMinRunLetters. A run is: a letter that does not follow a letter,
// then repeatedly a non-empty gap of separators followed by a single letter.
// Two adjacent letters end the run — the spacing has stopped — and so does a
// gap that reaches a line break or the end of the text.
//
// Ported from upstream's `_concealed_instruction_run_spans`
// (ss:artifacts.py:1662-1700) by way of difftest's `concealedInstructionRunSpans`,
// which is its measured-equivalent Go transliteration; the departure is the
// separator class, which excludes line terminators here.
func letterSpacingRuns(rs []rune) []letterSpacingRun {
	var runs []letterSpacingRun
	for i := 0; i < len(rs); {
		if !isRunLetter(rs[i]) || (i > 0 && isRunLetter(rs[i-1])) {
			i++
			continue
		}
		start, lastLetterEnd, letters := i, i+1, 1
		c := i + 1
		for c < len(rs) {
			gap := c
			for c < len(rs) && isSeparator(rs[c]) {
				c++
			}
			if c == gap || c >= len(rs) || !isRunLetter(rs[c]) {
				break
			}
			letters++
			lastLetterEnd = c + 1
			c = lastLetterEnd
			if c < len(rs) && isRunLetter(rs[c]) {
				break
			}
		}
		if letters >= compactMinRunLetters {
			runs = append(runs, letterSpacingRun{start: start, end: lastLetterEnd})
			i = lastLetterEnd
			continue
		}
		i = start + 1
	}
	return runs
}

// unitGap is the width, in runes, of the narrowest gap in a run: the run's
// own letter separator, from which every wider gap is read as a word
// boundary. A run whose gaps are all the same width is one word.
func unitGap(rs []rune, run letterSpacingRun) int {
	unit := 0
	for i := run.start; i < run.end; {
		if isRunLetter(rs[i]) {
			i++
			continue
		}
		gap := i
		for i < run.end && !isRunLetter(rs[i]) {
			i++
		}
		if w := i - gap; unit == 0 || w < unit {
			unit = w
		}
	}
	return unit
}

// isRunLetter reports whether a character counts as one of a run's letters.
// Combining marks count, so a base character and its mark are two adjacent
// "letters" and end the run rather than being read as a letter and a
// separator — the conservative reading, and the one Python's `[^\W\d_]` gives
// upstream.
func isRunLetter(r rune) bool {
	return unicode.IsLetter(r) || unicode.In(r, unicode.Mn, unicode.Mc)
}

// isSeparator reports whether a character can sit inside a run's gap:
// anything that is not a letter, a digit or a line terminator.
//
// Line terminators are excluded so that a run cannot cross a line, which is
// what keeps a view's lines the file's lines. The set is the Unicode vertical
// whitespace that survives the fold — upstream's LOGICAL_LINE_BREAK minus the
// C0 file/group/record separators, which the skeleton has already stripped as
// non-whitespace controls.
func isSeparator(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.In(r, unicode.Mn, unicode.Mc) {
		return false
	}
	switch r {
	case '\n', '\r', '\v', '\f', '', ' ', ' ':
		return false
	}
	return true
}

// runeOffsets decodes text into runes beside the byte offset of each, with a
// final entry holding len(text) so a half-open rune range always has a byte
// range. Invalid bytes decode to one RuneError each, matching the way the
// fold passes them through.
func runeOffsets(text string) ([]rune, []int) {
	rs := make([]rune, 0, len(text))
	bo := make([]int, 0, len(text)+1)
	for i, r := range text {
		rs = append(rs, r)
		bo = append(bo, i)
	}
	return rs, append(bo, len(text))
}
