package skillgate

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ---- the Markup view ----
//
// The channel this closes is *interpolation*: `Ignore **all previous**
// instructions` is the blocker-severity SK-T002 payload and it produced zero
// findings, because ordinary markdown bold puts `**` exactly where the rule's
// `\s+` has to match. It is the headline case of the release and no other
// view reaches it. The skeleton fold does not touch `*` — NFKC and the
// confusable skeleton have no mapping for it, and it is neither a format
// character nor a default-ignorable. The compaction strips separators only
// inside runs of six or more *isolated letters*, and `**all previous**` is
// whole words. Folding the payload the skeleton's way yields
// `Ignoreallpreviousinstructions`, which then fails the same regex for want
// of whitespace — a different miss, not a fix. Reconstructing the word
// boundary is the whole problem, and only the markup grammar knows where it
// is.
//
// **The rule is a grammar, not a list of evasions.** Upstream closes this
// case with a filler-stripping view scoped to five instruction verbs
// (`ignore|override|bypass|disregard|forget`, docs/research/rule-language.md
// §26) — a matcher enumerating forms where the vocabulary is grammar, which
// is the defect this release exists to end. This stage instead resolves
// CommonMark's *inline* productions: emphasis and strong-emphasis delimiter
// runs, code spans, inline links and images, and HTML comments. It leaves the
// text they wrap. So it closes `**bold**`, `_italic_`, `` `code` ``,
// `[text](url)`, `![alt](url)` and `<!-- comment -->` with one derivation,
// and it closes the next markup spelling nobody has thought of, because the
// grammar already contains it. Nothing here knows what a threat looks like —
// the rules hold the vocabulary and this stage holds the syntax.
//
// **The false-positive risk is the whole difficulty, and the grammar is the
// answer to it.** `*`, `_`, `` ` `` and the bracket characters are ordinary
// prose punctuation: they appear as themselves in globs (`rm *.sh`),
// identifiers (`snake_case`), pointer types (`v *View`), arithmetic
// (`2 * 3`), shell expansions (`${a[*]}`), tables (`| Python | https://… |`)
// and regexes, constantly. A stripper that removed them everywhere would fuse
// unrelated words and manufacture findings out of prose. What tells a
// delimiter from a literal asterisk is not a list of exceptions: it is
// CommonMark's **delimiter run** rule, which asks only what sits on either
// side of the run, plus the requirement that a delimiter actually *pair* with
// another one. `rm *.sh` has an opener and no closer, so nothing is stripped;
// `2 * 3` is flanked by whitespace on both sides, so it is not a delimiter at
// all. Both fall out of the grammar, neither is a special case.
//
// What the stage may *not* do is as load-bearing as what it does:
//
//   - **It only ever deletes.** Its output is a subsequence of its input, so
//     it can bring two characters that were already on the line together, but
//     it can never produce a character the line did not contain. That is the
//     bound on the failure mode the skeleton view measured, where NFKC turned
//     a bare `‥` into `..` and fired a path-traversal blocker — and SK-T019,
//     the rule that fired, is already raw-only for exactly this reason.
//   - **It never crosses a line break.** CommonMark lets emphasis span the
//     lines of a paragraph; this stage deliberately stops at the newline,
//     because every rule cuts its evidence on lines and a pair resolved
//     across a break would let two lines' text form a match that is in
//     neither. Bounding the scan to the line is also what makes the view's
//     lines the file's lines, which the whole reporting contract rests on.
//
// Stated limits, so they are contract rather than discovery:
//
//   - **Reference links** (`[text][ref]`, `[text][]`) are not resolved. They
//     need the document's link-reference definitions, which a line-bounded
//     stage does not have. Inline links and images are.
//   - **A code span's content is literal, delimiters and all.** That is
//     CommonMark precedence — a renderer displays `` `**x**` `` as `**x**`,
//     so a view that stripped the `**` inside it would be reading the
//     document differently from every renderer and every human. An HTML
//     comment is the opposite case: it displays nothing at all, so there is
//     no literal rendering to preserve, its content is text the model reads,
//     and markup inside it is markup.
//   - **Strikethrough (`~~x~~`) is not resolved.** It is a GFM extension
//     rather than a CommonMark production, and it is outside the set this
//     slice was planned around. Measured over this repository and 300 real
//     markdown documents it would have been inert either way; `~` here is
//     overwhelmingly the home-directory shorthand SK-T010 is written around.

// markupDelims are the two emphasis delimiter characters of the CommonMark
// inline grammar. They are a property of the grammar, not a list anyone
// tunes: what makes an occurrence a delimiter is the flanking rule below,
// never which character it is.
const markupDelims = "*_"

// stripInlineMarkup removes inline markup syntax and keeps the text it wraps,
// returning its own offsets into the text it was given. composeSegs maps
// those back to raw, so a finding discovered here still reports the raw line.
//
// It returns (text, nil) when it changed nothing — the identity convention
// View.segs uses, which costs nothing for the many files that contain no
// resolvable markup at all.
func stripInlineMarkup(text string) (string, []viewSeg) {
	var sb segBuilder
	sb.b.Grow(len(text))

	changed := false
	off := 0
	for _, piece := range strings.SplitAfter(text, "\n") {
		if piece == "" {
			continue
		}
		body := strings.TrimSuffix(piece, "\n")
		del := inlineMarkupDeletions(body)

		i := 0
		for i < len(body) {
			if del[i] {
				j := i
				for j < len(body) && del[j] {
					j++
				}
				sb.emit("", off+i, j-i, false)
				changed = true
				i = j
				continue
			}
			j := i
			for j < len(body) && !del[j] {
				j++
			}
			sb.emit(body[i:j], off+i, j-i, true)
			i = j
		}
		if len(piece) > len(body) {
			sb.emit("\n", off+len(body), 1, true)
		}
		off += len(piece)
	}

	if !changed {
		return text, nil
	}
	return sb.done()
}

// inlineMarkupDeletions returns, for one line, which bytes are markup syntax
// rather than content.
//
// The three passes are the grammar's precedence order. Code spans and HTML
// comments bind tightest and are resolved first, because a delimiter inside a
// code span is not a delimiter. Links are resolved next, because a link's
// destination is an attribute rather than content and must be out of the way
// before emphasis is considered. Emphasis is resolved last, over whatever is
// left, which is how emphasis inside a link's text — `[**a b**](u)` — comes
// out right with no special case.
func inlineMarkupDeletions(line string) []bool {
	sc := newInlineScan(line)
	sc.scanCodeSpansAndComments()
	sc.scanLinks()
	sc.scanEmphasis()
	return sc.del
}

// inlineScan is one line under inspection, with three masks over its bytes.
//
// del marks syntax to remove. opaque marks bytes a later pass must not read
// as markup — a code span's literal content and a link's destination — which
// is how precedence is expressed as data rather than as a nest of conditions.
// esc marks bytes a backslash made literal, precomputed in one pass rather
// than counted backwards on demand: the gate reads files up to a megabyte and
// a hostile bundle is free to put all of one on a single line, so every pass
// here has to stay linear in the length of the line.
type inlineScan struct {
	s      string
	del    []bool
	opaque []bool
	esc    []bool
}

// newInlineScan builds the masks for one line, resolving backslash escapes up
// front. A backslash that is not itself escaped makes the next byte literal,
// which is what makes `\*not emphasis\*` stay as written.
func newInlineScan(line string) *inlineScan {
	sc := &inlineScan{
		s:      line,
		del:    make([]bool, len(line)),
		opaque: make([]bool, len(line)),
		esc:    make([]bool, len(line)),
	}
	for i := 0; i < len(line); i++ {
		if line[i] == '\\' && !sc.esc[i] && i+1 < len(line) {
			sc.esc[i+1] = true
		}
	}
	return sc
}

func (sc *inlineScan) markDel(start, n int) {
	for i := start; i < start+n && i < len(sc.del); i++ {
		sc.del[i] = true
	}
}

func (sc *inlineScan) markOpaque(start, end int) {
	for i := start; i < end && i < len(sc.opaque); i++ {
		sc.opaque[i] = true
	}
}

// backtickRun is one maximal run of unescaped backticks, with the index of
// the next run of exactly its own length.
type backtickRun struct {
	start, n int
	nextEq   int // index into the run list, or -1
}

// backtickRuns lists every backtick run on the line and links each to the
// next run of equal length, in two linear passes.
//
// The link is precomputed rather than searched for because searching is
// quadratic on a line of runs of increasing length — a megabyte of
// “ `a“a```a… “ took half a second before this table existed, and the
// gate's whole job is reading files an attacker chose.
func (sc *inlineScan) backtickRuns() []backtickRun {
	var runs []backtickRun
	for i := 0; i < len(sc.s); {
		if sc.s[i] != '`' || sc.esc[i] {
			i++
			continue
		}
		n := 0
		for i+n < len(sc.s) && sc.s[i+n] == '`' && !sc.esc[i+n] {
			n++
		}
		runs = append(runs, backtickRun{start: i, n: n, nextEq: -1})
		i += n
	}
	lastOfLen := map[int]int{}
	for i := len(runs) - 1; i >= 0; i-- {
		if j, ok := lastOfLen[runs[i].n]; ok {
			runs[i].nextEq = j
		}
		lastOfLen[runs[i].n] = i
	}
	return runs
}

// scanCodeSpansAndComments resolves the two productions that bind tighter
// than emphasis, in one left-to-right pass.
//
// A code span is a backtick run closed by the next backtick run of exactly
// the same length — that length rule is why “ `x` “ and ``` “a`b“ ```
// both work and why a lone backtick in prose stays literal. Its content is
// marked opaque: a renderer shows it verbatim, so no later pass may read
// markup inside it.
//
// An HTML comment is `<!--` … `-->`. Its markers are removed and its content
// is *not* opaque, because a comment renders as nothing at all: there is no
// literal display to preserve, the content is text the model still reads, and
// a payload hidden in a comment is the channel the production exists to
// close. So the scan continues *into* the comment rather than over it, and
// markup inside one is resolved like markup anywhere else.
func (sc *inlineScan) scanCodeSpansAndComments() {
	runs := sc.backtickRuns()
	runAt := make(map[int]int, len(runs))
	for i, r := range runs {
		runAt[r.start] = i
	}

	// A monotonic cursor for the next `-->`, so a line of a million unclosed
	// `<!--` costs one scan of the line rather than one per opener: the scan
	// only ever moves forward, and once there is no closer left there is none
	// for any later opener either. Searching from each opener took fifty
	// seconds on a megabyte of them.
	closeCursor := 0
	nextCommentClose := func(from int) int {
		if closeCursor < from {
			closeCursor = from
		}
		for closeCursor+3 <= len(sc.s) {
			if sc.s[closeCursor] == '-' && strings.HasPrefix(sc.s[closeCursor:], "-->") {
				return closeCursor
			}
			closeCursor++
		}
		return -1
	}

	for i := 0; i < len(sc.s); {
		if sc.esc[i] || sc.del[i] {
			i++
			continue
		}
		switch sc.s[i] {
		case '`':
			ri, ok := runAt[i]
			if !ok {
				i++
				continue
			}
			r := runs[ri]
			if r.nextEq < 0 {
				i += r.n // an unpaired backtick run is ordinary text
				continue
			}
			close := runs[r.nextEq]
			sc.markDel(r.start, r.n)
			sc.markOpaque(r.start+r.n, close.start)
			sc.markDel(close.start, close.n)
			i = close.start + close.n
		case '<':
			if strings.HasPrefix(sc.s[i:], "<!--") {
				if end := nextCommentClose(i + 4); end >= 0 {
					sc.markDel(i, 4)
					sc.markDel(end, 3)
					i += 4
					continue
				}
			}
			i++
		default:
			i++
		}
	}
}

// scanLinks resolves inline links and images: `[text](destination)` and
// `![alt](destination)` keep the text they wrap and lose the syntax and the
// destination.
//
// The destination is the one thing this stage deletes that is not syntax, and
// that is the grammar's own reading: a destination is an attribute of the
// link, not part of the document's prose. Keeping it would be worse than
// dropping it — `[click](https://x)` would become `clickhttps://x`, fusing
// two things that are not adjacent in any rendering. Nothing is lost by the
// deletion either way, because the raw view is always scanned and always wins
// dedup.
//
// A `[` with no `](…)` after it is left entirely alone: `[TODO]` and `- [ ]`
// are literal text in CommonMark too.
//
// Brackets are resolved with a **stack in one left-to-right pass**, which is
// both how CommonMark's own parser does it — a `]` looks for the innermost
// unmatched opener — and the only way to stay linear. Scanning forward from
// each `[` for its partner is quadratic, and a line of twenty thousand `[`
// took a quarter of a second before this was a stack; a bundle is free to put
// a megabyte of them on one line, so that shape is a denial of service on the
// gate rather than a performance note.
func (sc *inlineScan) scanLinks() {
	closer := sc.parenPartners()
	var open []int
	for i := 0; i < len(sc.s); i++ {
		if sc.opaque[i] || sc.del[i] || sc.esc[i] {
			continue
		}
		switch sc.s[i] {
		case '[':
			open = append(open, i)
		case ']':
			if len(open) == 0 {
				continue
			}
			start := open[len(open)-1]
			open = open[:len(open)-1]
			if i+1 >= len(sc.s) || sc.s[i+1] != '(' {
				continue // a bracket pair that is not a link stays literal
			}
			destEnd := closer[i+1]
			if destEnd < 0 {
				continue
			}
			text := start
			if start > 0 && sc.s[start-1] == '!' && !sc.opaque[start-1] && !sc.esc[start-1] {
				start-- // an image's `!` is part of the syntax
			}
			sc.markDel(start, text-start+1)
			sc.markDel(i, 1)
			sc.markDel(i+1, destEnd-i)
			sc.markOpaque(i+1, destEnd+1)
		}
	}
}

// parenPartners returns, for every byte of the line, the index of the `)`
// closing the `(` at that byte, or -1. One stack pass, so a line of unclosed
// parentheses costs what reading it costs.
func (sc *inlineScan) parenPartners() []int {
	out := make([]int, len(sc.s)+1)
	for i := range out {
		out[i] = -1
	}
	var open []int
	for i := 0; i < len(sc.s); i++ {
		if sc.esc[i] {
			continue
		}
		switch sc.s[i] {
		case '(':
			open = append(open, i)
		case ')':
			if len(open) > 0 {
				out[open[len(open)-1]] = i
				open = open[:len(open)-1]
			}
		}
	}
	return out
}

// ---- emphasis ----

// emphasisDelim is one delimiter run: a maximal sequence of the same
// delimiter character, with the flanking verdict the grammar computes from
// what sits on either side of it.
//
// used counts the characters already consumed by a match. An opener is
// consumed from its *end* and a closer from its *start*, because that is
// where the emphasis they mark actually begins and ends — which is what makes
// `***a***` come apart into a strong pair and an emphasis pair with the
// asterisks deleted in the right places.
type emphasisDelim struct {
	pos, end          int // byte range of the run
	ch                byte
	origN, used       int
	canOpen, canClose bool
}

func (d *emphasisDelim) rem() int { return d.origN - d.used }

// openersFloor keys the lower bound below which a closer of this class will
// never find an opener again: the delimiter character, the closer's length
// modulo three and whether it can also open. Those are exactly the inputs the
// rule of three reads, so two closers with the same key fail on the same
// openers.
type openersFloor struct {
	ch      byte
	mod3    int
	canOpen bool
}

// scanEmphasis finds every delimiter run on the line and pairs them.
//
// The look-back keeps CommonMark's `openers_bottom` floor, which is part of
// the published algorithm rather than an optimisation bolted beside it: when
// a closer finds no opener, no later closer of the same class can find one
// below that point either, so the floor rises and the prefix is never
// rescanned. Without it the pass is quadratic, and `*a*a*a*a…` — a megabyte
// of which took eighty seconds — is a line any bundle may contain.
func (sc *inlineScan) scanEmphasis() {
	ds := sc.delimiterRuns()
	floor := map[openersFloor]int{}
	for ci := range ds {
		c := &ds[ci]
		if !c.canClose {
			continue
		}
		key := openersFloor{c.ch, c.origN % 3, c.canOpen}
		for c.rem() > 0 {
			oi := -1
			for k := ci - 1; k >= floor[key]; k-- {
				o := &ds[k]
				if o.ch == c.ch && o.canOpen && o.rem() > 0 && !skipByRuleOfThree(o, c) {
					oi = k
					break
				}
			}
			if oi < 0 {
				floor[key] = ci
				break
			}
			o := &ds[oi]
			use := 1
			if o.rem() >= 2 && c.rem() >= 2 {
				use = 2
			}
			sc.markDel(o.end-o.used-use, use)
			o.used += use
			sc.markDel(c.pos+c.used, use)
			c.used += use
			// Delimiters between a matched pair can no longer pair with
			// anything outside it: emphasis does not interleave.
			for k := oi + 1; k < ci; k++ {
				ds[k].used = ds[k].origN
			}
		}
	}
}

// skipByRuleOfThree is CommonMark's rule that a delimiter which can both open
// and close may not pair when the two run lengths sum to a multiple of three,
// unless both are themselves multiples of three. It is what keeps
// `*foo**bar*` from coming apart wrongly, and it is part of the grammar
// rather than a heuristic.
func skipByRuleOfThree(o, c *emphasisDelim) bool {
	if !c.canOpen && !o.canClose {
		return false
	}
	if (o.origN+c.origN)%3 != 0 {
		return false
	}
	return o.origN%3 != 0 || c.origN%3 != 0
}

// delimiterRuns collects every emphasis delimiter run on the line, skipping
// bytes a tighter production already claimed, and computes each run's
// flanking verdict.
func (sc *inlineScan) delimiterRuns() []emphasisDelim {
	var ds []emphasisDelim
	for i := 0; i < len(sc.s); {
		ch := sc.s[i]
		if strings.IndexByte(markupDelims, ch) < 0 || sc.opaque[i] || sc.del[i] || sc.esc[i] {
			i++
			continue
		}
		n := 0
		for i+n < len(sc.s) && sc.s[i+n] == ch && !sc.opaque[i+n] && !sc.del[i+n] {
			n++
		}
		d := emphasisDelim{pos: i, end: i + n, ch: ch, origN: n}
		d.canOpen, d.canClose = sc.flanking(i, i+n, ch)
		ds = append(ds, d)
		i += n
	}
	return ds
}

// flanking computes whether a run may open and may close, from CommonMark's
// left- and right-flanking definitions.
//
// A run is left-flanking when it is not followed by whitespace and either not
// followed by punctuation or preceded by whitespace or punctuation;
// right-flanking is the mirror. `*` opens when left-flanking and closes when
// right-flanking. `_` carries the extra condition that keeps `snake_case`
// literal: it may only open when it is left-flanking and either not
// right-flanking or preceded by punctuation, and mirror-wise to close.
//
// The start and the end of the line count as whitespace, which is how the
// stage's line bound enters the grammar rather than sitting beside it.
func (sc *inlineScan) flanking(start, end int, ch byte) (canOpen, canClose bool) {
	beforeWS, beforePunct := classifyRune(lastRune(sc.s[:start]))
	afterWS, afterPunct := classifyRune(firstRune(sc.s[end:]))

	left := !afterWS && (!afterPunct || beforeWS || beforePunct)
	right := !beforeWS && (!beforePunct || afterWS || afterPunct)

	if ch == '_' {
		return left && (!right || beforePunct), right && (!left || afterPunct)
	}
	return left, right
}

// classifyRune reports whether a character is whitespace or punctuation for
// the flanking rule. utf8.RuneError with no text stands for the start or the
// end of the line, which the grammar treats as whitespace.
func classifyRune(r rune, ok bool) (isWS, isPunct bool) {
	if !ok {
		return true, false
	}
	return unicode.IsSpace(r), unicode.IsPunct(r) || unicode.IsSymbol(r)
}

func lastRune(s string) (rune, bool) {
	if s == "" {
		return 0, false
	}
	r, _ := utf8.DecodeLastRuneInString(s)
	return r, true
}

func firstRune(s string) (rune, bool) {
	if s == "" {
		return 0, false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return r, true
}
