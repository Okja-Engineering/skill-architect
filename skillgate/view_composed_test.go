package skillgate

import (
	"os"
	"strings"
	"testing"
	"time"
	"unicode"
)

// --- fixtures, derived rather than typed ----------------------------------

// interpolateIntoRun spells a payload that is *both* evasions at once: the
// text is already letter-spaced, and an inline-markup construct is opened and
// closed around two of the run's own letters.
//
// It is derived from the two parent slices' derivations rather than typed
// out, so the composed fixture cannot drift into being a different sentence
// or a different construct than the ones S10 and S11 close. The delimiters go
// *inside* a run, which is the whole point: markup outside a run is already
// reached by the compaction, because a delimiter outside a run is either
// untouched or absorbed into a gap. What no single view reaches is a
// delimiter sitting where the compaction has to read a gap width.
func interpolateIntoRun(spaced string, w markupWrap) string {
	rs := []rune(spaced)
	var letters []int
	for i, r := range rs {
		if unicode.IsLetter(r) {
			letters = append(letters, i)
		}
	}
	if len(letters) < 3 {
		panic("the payload has too few letters to interpolate inside its run")
	}
	open, closeAt := letters[1], letters[2]
	var sb strings.Builder
	for i, r := range rs {
		if i == open {
			sb.WriteString(w.open)
		}
		sb.WriteRune(r)
		if i == closeAt {
			sb.WriteString(w.close)
		}
	}
	return sb.String()
}

// composedSeparators are the letter-spacing separators the composed view is
// measured against. They are S10's separator rows minus the two that are
// themselves inline-markup delimiters (`*` and `_`) — those are a stated
// limit, pinned by name in TestComposedViewMissesMarkupCharacterSeparators
// rather than quietly dropped from the table.
var composedSeparators = []struct{ name, sep string }{
	{"space", " "},
	{"dot", "."},
	{"hyphen", "-"},
	{"nbsp", " "},
	{"wide-unit", "  "},
}

// composedViewOf returns the composed view of one text, built through the
// same registry the ledger uses.
func composedViewOf(t *testing.T, raw string) *View {
	t.Helper()
	for _, v := range newViews("probe", raw) {
		if v.Name == viewMarkupCompact {
			return v
		}
	}
	t.Fatalf("no %s view in the registry", viewMarkupCompact)
	return nil
}

// --- the slice's reason to exist ------------------------------------------

// TestComposedViewFiresOnLetterSpacedInterpolatedPayloads is the GREEN of this
// slice, and it is the gap S10 §7.6 and S11 §7.6 each raised independently: a
// payload that is letter-spaced *and* markup-interpolated is reached by no
// view built from the fold plus one stage.
//
// Every row asserts the whole contract rather than merely that something
// fired: the view is named, the line is the *raw* line, and the evidence is
// the raw line as it is written on disk, markup and spacing and all. The view
// tag is also what proves no earlier view reached it — an earlier discovery
// wins dedup and would carry that view's name instead.
func TestComposedViewFiresOnLetterSpacedInterpolatedPayloads(t *testing.T) {
	for _, sc := range composedSeparators {
		for _, w := range markupWraps {
			t.Run(sc.name+"/"+w.name, func(t *testing.T) {
				spelled := interpolateIntoRun(toLetterSpaced(overridePayload, sc.sep), w)
				if spelled == overridePayload {
					t.Fatal("the fixture derivation produced the plain payload")
				}
				root, wantLine := bundleWithBody(t, "x", spelled)
				got := findingsFor(t, root, "SK-T002")
				if len(got) != 1 {
					t.Fatalf("want exactly 1 SK-T002, got %d: %+v", len(got), got)
				}
				f := got[0]
				if f.View != viewMarkupCompact {
					t.Errorf("view = %q, want %q", f.View, viewMarkupCompact)
				}
				if f.Line != wantLine {
					t.Errorf("line = %d, want %d (the raw source line)", f.Line, wantLine)
				}
				// The evidence is the raw source line as the report renders
				// it — the gate caps evidence at 160 bytes, and a composed
				// spelling of this payload runs past that in several of the
				// rows below. Comparing against the gate's own rendering is
				// replaying the engine rather than relaxing the assertion:
				// what is asserted is still that every byte of the evidence
				// came from the raw line, markup and spacing intact.
				if want := truncate(spelled, 160); f.Evidence != want {
					t.Errorf("evidence is not the raw source line\n got: %q\nwant: %q",
						f.Evidence, want)
				}
				if f.Severity != SeverityBlocker {
					t.Errorf("severity = %q, want %q", f.Severity, SeverityBlocker)
				}
			})
		}
	}
}

// TestNoSingleStageViewReachesTheComposedSpelling is the non-vacuity guard for
// the rows above: it asserts the composed fixtures are genuinely out of reach
// of every view that existed before this slice.
//
// Without it, a fixture that drifted into being reachable by the compaction
// alone would leave this slice's headline rows passing on a finding another
// view had already made — and the view tag assertion would fail confusingly
// rather than saying what is wrong. Here it says exactly what is wrong.
func TestNoSingleStageViewReachesTheComposedSpelling(t *testing.T) {
	for _, sc := range composedSeparators {
		for _, w := range markupWraps {
			t.Run(sc.name+"/"+w.name, func(t *testing.T) {
				spelled := interpolateIntoRun(toLetterSpaced(overridePayload, sc.sep), w)
				for _, v := range newViews("probe", spelled) {
					if v.Name == viewMarkupCompact {
						continue
					}
					if reOverride[0].MatchString(v.Text) {
						t.Errorf("the %s view already reaches this spelling: %q — the composed "+
							"fixture is no longer a composed case", v.Name, v.Text)
					}
				}
			})
		}
	}
}

// TestComposedStageOrderIsMarkupThenCompaction pins the order decision, which
// is the part of this slice that is a decision rather than a registry line.
//
// Strip-then-compact and compact-then-strip are different transforms with
// different outputs, and the choice was measured rather than inherited. Over
// 18,480 spellings of the payload — every combination of unit separator, word
// gap and interpolated construct — markup-then-compaction reached 10,968 that
// no existing view reached, and compaction-then-markup reached **zero** that
// markup-then-compaction did not. The reason is structural: the compaction
// normalises every gap in a run to "" or " ", so a delimiter inside a run is
// consumed as gap material and can never pair afterwards; the markup stage
// run first still sees the delimiters as delimiters.
//
// This test is that measurement as an executable claim, so a later reordering
// of the registry fails by name rather than silently losing the closure.
func TestComposedStageOrderIsMarkupThenCompaction(t *testing.T) {
	reached := 0
	for _, sc := range composedSeparators {
		for _, w := range markupWraps {
			spelled := interpolateIntoRun(toLetterSpaced(overridePayload, sc.sep), w)
			folded, _ := foldSkeleton(spelled)

			stripped, _ := stripInlineMarkup(folded)
			markupFirst, _ := compactLetterSpacing(stripped)

			compacted, _ := compactLetterSpacing(folded)
			compactFirst, _ := stripInlineMarkup(compacted)

			if !reOverride[0].MatchString(markupFirst) {
				t.Errorf("%s/%s: markup-then-compaction does not reach it: %q",
					sc.name, w.name, markupFirst)
				continue
			}
			reached++
			if reOverride[0].MatchString(compactFirst) {
				t.Errorf("%s/%s: compaction-then-markup reaches it too — the order measurement "+
					"this registry rests on no longer holds", sc.name, w.name)
			}

			// And the registry really is in that order: the view the ledger
			// builds is the text markup-then-compaction produces.
			if got := composedViewOf(t, spelled).Text; got != markupFirst {
				t.Errorf("%s/%s: the registered %s view is not markup-then-compaction\n got: %q\nwant: %q",
					sc.name, w.name, viewMarkupCompact, got, markupFirst)
			}
		}
	}
	if reached == 0 {
		t.Fatal("no fixture was reached by either order: this measurement is proving nothing")
	}
}

// TestComposedViewIsScannedLastSoItCannotTakeAnEarlierViewsPlace pins the
// registry-order property that makes "adding a view can only add findings"
// true of this view in particular.
//
// Dedup is raw-first and keyed on (rule, file, line), so the *first* view to
// reach a place is the one whose name the finding carries. The composed view
// is a superset transform of two views that are already registered: were it
// scanned before either of them, every finding they own would start arriving
// tagged `markupCompact` instead, which is a silent relabelling of existing
// output rather than an addition to it. Last is what keeps that from
// happening, and the whole-repository differential's empty REMOVED set is the
// measurement that agrees.
func TestComposedViewIsScannedLastSoItCannotTakeAnEarlierViewsPlace(t *testing.T) {
	names := ViewNames()
	if got := names[len(names)-1]; got != viewMarkupCompact {
		t.Errorf("the last view scanned is %q, want %q — a composed view scanned before its "+
			"own stages' views would relabel their findings rather than add any", got, viewMarkupCompact)
	}
	// Non-vacuity: both views it composes really are scanned before it.
	for _, earlier := range []string{viewCompactLetter, viewMarkup} {
		at := -1
		for i, n := range names {
			if n == earlier {
				at = i
			}
		}
		if at < 0 {
			t.Errorf("%s is not registered: this ordering claim is about a view that does not exist",
				earlier)
		}
	}
}

// TestComposedViewMissesMarkupCharacterSeparators states the limit rather than
// hiding it: when the letter-spacing separator is *itself* an inline-markup
// delimiter, the markup stage reads the run's own separators as delimiters,
// resolves some of them, and the compaction that follows no longer sees the
// run. The composed view misses that spelling in both stage orders.
//
// It is a limit and not a defect of this slice: the same payload *without*
// the extra interpolated construct is closed by the compaction alone (S10's
// `asterisk` and `underscore` rows), so reaching it costs an attacker the
// interpolation this view exists to close.
//
// Pinned so the limit is measured rather than asserted, and so that a later
// change which closes it fails here and prompts the spec line to be removed.
func TestComposedViewMissesMarkupCharacterSeparators(t *testing.T) {
	for _, sep := range []string{"*", "_"} {
		t.Run(sep, func(t *testing.T) {
			spelled := interpolateIntoRun(toLetterSpaced(overridePayload, sep), markupWraps[0])
			for _, v := range newViews("probe", spelled) {
				if reOverride[0].MatchString(v.Text) {
					t.Errorf("the %s view now reaches the %q-separated spelling: the stated limit "+
						"is out of date and the spec must lose that line", v.Name, sep)
				}
			}
			// Non-vacuity: the same spelling without the interpolation *is*
			// closed, so this row is about the combination and not about the
			// separator being unreachable in general.
			plain := toLetterSpaced(overridePayload, sep)
			hit := false
			for _, v := range newViews("probe", plain) {
				if reOverride[0].MatchString(v.Text) {
					hit = true
				}
			}
			if !hit {
				t.Errorf("the plain %q-separated payload is not reached either — this limit is "+
					"not about the combination and the reasoning above is wrong", sep)
			}
		})
	}
}

// --- the offset map, through two stages -----------------------------------

// TestComposedViewOffsetMapIsLoadBearing is the mapping evidence, in S09's
// non-vacuity shape: for each fixture the mapped line must be right *and*
// reading the derived offset as a raw offset must give the wrong line, so a
// fixture that stopped changing length fails loudly instead of passing while
// proving nothing.
//
// The composed view is the first with **two** stages after the fold, which is
// a new way to be wrong: an offset can be correct in each stage's own map and
// wrong in the composition of the three. So every direction the composition
// can shift in is a row — the fold shrinking, the fold growing, the markup
// stage shrinking, and the compaction shrinking — and each is padding that a
// *different* stage is responsible for.
func TestComposedViewOffsetMapIsLoadBearing(t *testing.T) {
	spelled := interpolateIntoRun(toLetterSpaced(overridePayload, " "), markupWraps[0])
	cases := []struct {
		name string
		pad  []string
	}{
		{"shrinking-fold", []string{padFullwidth(60), padFullwidth(60)}},
		{"growing-fold", []string{padExpanding(80), padExpanding(80)}},
		{"shrinking-markup", []string{padMarkup(40), padMarkup(40)}},
		{"shrinking-compaction", []string{padLetterSpaced(40), padLetterSpaced(40)}},
		// Every stage changing length at once, in both directions, on the
		// same file: the composition has to be right in all three maps
		// simultaneously, which no single-stage fixture can demand.
		{"all-stages", []string{padExpanding(80), padFullwidth(60), padMarkup(40), padLetterSpaced(40)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := append(append([]string{}, tc.pad...), spelled)
			root, wantLine := bundleWithBody(t, "x", body...)
			l, err := BuildLedger(root)
			if err != nil {
				t.Fatal(err)
			}
			f := l.Get("SKILL.md")
			if f == nil {
				t.Fatal("SKILL.md not in the ledger")
			}
			var view *View
			for _, v := range f.Views() {
				if v.Name == viewMarkupCompact {
					view = v
				}
			}
			if view == nil {
				t.Fatalf("no %s view", viewMarkupCompact)
			}
			d := strings.Index(view.Text, overrideRun)
			if d < 0 {
				t.Fatalf("the composed payload is not in the %s view — the view does not close "+
					"this spelling: %q", viewMarkupCompact, view.Text)
			}
			if got := f.LineNumber(view.SourceOffset(d)); got != wantLine {
				t.Errorf("mapped line = %d, want %d", got, wantLine)
			}
			if naive := f.LineNumber(d); naive == wantLine {
				t.Errorf("the derived offset %d already lands on line %d: this fixture no longer "+
					"changes length, so it proves nothing about the mapping", d, naive)
			}
			// The mapped span must bracket the raw spelling exactly — the
			// letter spacing and the markup delimiters included.
			start, end := view.SourceOffset(d), view.SourceEnd(d+len(overrideRun))
			want := strings.TrimSuffix(spelled, " .")
			if got := f.Text[start:end]; got != want {
				t.Errorf("mapped span\n got: %q\nwant: %q", got, want)
			}
		})
	}

	// The end of the file maps to the end of the file, whichever direction
	// the pipeline moved the bytes: a composition that is right in the middle
	// and wrong at the boundary is a real shape, and nothing above sees it.
	for _, tc := range cases {
		body := append(append([]string{}, tc.pad...), spelled)
		root, _ := bundleWithBody(t, "x", body...)
		l, err := BuildLedger(root)
		if err != nil {
			t.Fatal(err)
		}
		f := l.Get("SKILL.md")
		v := composedViewOf(t, f.Text)
		if got := v.SourceEnd(len(v.Text)); got != len(f.Text) {
			t.Errorf("%s: the end of the view maps to raw %d, want %d", tc.name, got, len(f.Text))
		}
	}
}

// --- the false-positive sweep ---------------------------------------------

// TestComposedViewFiresOnlyOnHostileFixtures is the control that matters, in
// S10's shape: gate this whole repository and the composed view must account
// for every finding it is responsible for — the hostile fixtures that exist
// to be fired on, and nothing this project has written.
//
// The composed view is the riskiest of the four, and not because it is the
// union of its parents' risks. Markup stripping *fuses* text across removed
// delimiters, and the compaction then reads letter runs that exist neither in
// the raw document nor in either single view. Manufacturing a finding out of
// innocent prose is the failure this release keeps refusing to ship, so this
// sweep is the slice rather than a formality.
//
// Both halves bite: a hit outside the hostile corpus is that failure, and
// zero hits anywhere would mean the sweep had gone vacuous.
func TestComposedViewFiresOnlyOnHostileFixtures(t *testing.T) {
	if _, err := os.Stat("../README.md"); err != nil {
		t.Skip("not running inside the repository")
	}
	rep, err := NewEngine().Gate("..", optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	hostile := 0
	for _, f := range rep.Findings {
		if f.View != viewMarkupCompact {
			continue
		}
		if strings.Contains(f.File, "testdata/hostile/") {
			hostile++
			continue
		}
		t.Errorf("the composed view produced %s at %s:%d — %q",
			f.RuleID, f.File, f.Line, f.Evidence)
	}
	if hostile == 0 {
		t.Error("the composed view found nothing anywhere in the repository, not even in the " +
			"hostile fixtures: this sweep is proving nothing")
	}
}

// TestComposedViewManufacturesNoLettersOrPunctuation pins what the composition
// is allowed to do to a line, which is the property the sweep is a sample of.
//
// Each stage has a bound of its own — the markup stage only deletes, the
// compaction only brings letters together — but a composition's bound is not
// the conjunction of its parts, because the compaction runs on text the
// markup stage has already fused. So the bound is asserted on the composition
// itself, over real documents: every character of a composed line was on the
// raw line, in order. The composition can bring characters together; it can
// never produce one the line did not contain.
func TestComposedViewManufacturesNoLettersOrPunctuation(t *testing.T) {
	for path, raw := range realDocuments(t) {
		folded, _ := foldSkeleton(raw)
		stripped, _ := stripInlineMarkup(folded)
		composed, _ := compactLetterSpacing(stripped)
		fl, cl := strings.Split(folded, "\n"), strings.Split(composed, "\n")
		if len(fl) != len(cl) {
			t.Fatalf("%s: %d lines became %d", path, len(fl), len(cl))
		}
		for i := range fl {
			if !isSubsequenceAllowingSpace(cl[i], fl[i]) {
				t.Errorf("%s:%d the composition produced characters the folded line does not have\n"+
					"  composed: %q\n     folded: %q", path, i+1, cl[i], fl[i])
			}
		}
	}
}

// isSubsequenceAllowingSpace reports whether every character of got appears in
// want in order, treating a space in got as satisfied by any character — the
// compaction is allowed to render a word boundary as a space where the raw
// text spelled it some other way, and that is the only substitution either
// stage makes.
func isSubsequenceAllowingSpace(got, want string) bool {
	w := []rune(want)
	j := 0
	for _, r := range got {
		for j < len(w) && w[j] != r {
			if r == ' ' {
				break
			}
			j++
		}
		if j >= len(w) {
			return false
		}
		j++
	}
	return true
}

// --- worst-case cost on adversarial input ---------------------------------

// TestComposedPipelineStaysLinearOnHostileLines is a security test, not a
// performance note, and it is inherited work rather than new caution: S11
// found three quadratic partner searches in its own fresh markup stage, where
// a megabyte of `*a*a*a…` on one line took **81 seconds**. Its warning was
// that any later view matching paired constructs carries the same hazard.
//
// This view composes one of those with a second stage and a three-way offset
// composition, and the gate's input is chosen by an attacker: a file may be a
// megabyte on a single line. So the whole composed pipeline is measured on the
// shapes that stress each stage and the composition between them — the markup
// stage's partner searches, the compaction's run scan, and a maximally
// fragmented offset map where every character is its own segment.
//
// The budget is two orders of magnitude above the measured worst case, so it
// cannot flake on a slow machine, and any reintroduced quadratic search blows
// straight through it.
func TestComposedPipelineStaysLinearOnHostileLines(t *testing.T) {
	const n = 1 << 20
	const budget = 5 * time.Second

	ticksIncreasing := func() string {
		var b strings.Builder
		for i := 1; b.Len() < n; i++ {
			b.WriteString(strings.Repeat("`", i))
			b.WriteString("a")
		}
		return b.String()
	}
	cases := []struct{ name, line string }{
		// The markup stage's partner searches, at the file-size cap.
		{"unclosed brackets", strings.Repeat("[", n)},
		{"emphasis runs that open and close", strings.Repeat("*a", n/2)},
		{"unclosed html comments", strings.Repeat("<!--", n/4)},
		{"backtick runs of increasing length", ticksIncreasing()},
		// The compaction's run scan: one enormous run, and many short runs
		// that fail the threshold and force a rescan from the next letter.
		{"one enormous letter-spacing run", strings.Repeat("a ", n/2)},
		{"short runs that fail the threshold", strings.Repeat("a b c 1 ", n/8)},
		{"letters separated by long gaps", strings.Repeat("a"+strings.Repeat(".", 512), n/513)},
		// The composition itself: text where the fold rewrites every
		// character, so the inner map has a segment per character and the
		// outer maps have to be composed through all of them.
		{"every character rewritten by the fold", strings.Repeat("Ｘ", n/3)},
		{"fold rewrites and markup deletes together", strings.Repeat("**Ｘ**", n/9)},
		{"all three stages on every character", strings.Repeat("*Ｘ* ", n/8)},
	}
	composed, found := viewBuilder{}, false
	for _, b := range viewBuilders {
		if b.name == viewMarkupCompact {
			composed, found = b, true
		}
	}
	if !found {
		t.Fatalf("no %s view is registered", viewMarkupCompact)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			got, segs := composed.build(tc.line)
			elapsed := time.Since(start)
			if elapsed > budget {
				t.Errorf("%d bytes took %v, over the %v budget — the gate can be stalled by a file",
					len(tc.line), elapsed, budget)
			}
			t.Logf("%d bytes in %v, %d segments", len(tc.line), elapsed, len(segs))
			// Non-vacuity: the pipeline really did run on the whole line.
			if len(got) == 0 && len(tc.line) > 0 {
				t.Errorf("the pipeline produced nothing from %d bytes", len(tc.line))
			}
		})
	}
}
