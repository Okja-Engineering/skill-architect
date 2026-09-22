// Package difftest holds the Go re-expressions of the 15 sampled SkillSpector
// rules, built for experiment E2: a differential fidelity harness that runs
// upstream Python (re) over a corpus and these ports (RE2 + scanners) over the
// same corpus, then diffs matches/misses/extras/evidence.
//
// Pattern text and analyzer semantics are derived from NVIDIA SkillSpector,
// Apache-2.0, pinned at 8421a2eb4e7bb98af2557bd4b54325d6ad90a3da (see NOTICE).
// Upstream rule IDs are recorded in Origin; ports use the SK-P* namespace.
package difftest

import (
	"encoding/base64"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Match is one normalized detection record, identical in shape to the records
// emitted by run_upstream.py.
type Match struct {
	Rule     string `json:"rule"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Evidence string `json:"evidence"`
}

// Doc is one corpus document under test.
type Doc struct {
	Path     string // corpus-relative path
	FileType string // upstream FILE_TYPES mapping
	Text     string
}

// PortedRule pairs a Go detector with its upstream provenance.
type PortedRule struct {
	SKID     string // SK-P-* rule id
	Origin   string // upstream SkillSpector rule id / lane
	Upstream string // record key used on the upstream side
	Detect   func(*Doc) []Match
	Note     string
}

// logicalLineBreak mirrors common.py LOGICAL_LINE_BREAK.
var logicalLineBreak = regexp.MustCompile(`\r\n|[\r\n\v\f\x1c-\x1e\x85\x{2028}\x{2029}]`)

// lineOf mirrors get_line_number: 1-based logical line for a char offset.
func lineOf(content string, off int) int {
	return len(logicalLineBreak.FindAllStringIndex(content[:off], -1)) + 1
}

// splitLogical mirrors Python str.splitlines() (same boundary set).
func splitLogical(content string) []string {
	parts := logicalLineBreak.Split(content, -1)
	return parts
}

// getContext mirrors common.get_context: ±contextLines logical lines joined.
func getContext(content string, start, contextLines int) string {
	lines := splitLogical(content)
	matchLine := lineOf(content, start) - 1
	lo := matchLine - contextLines
	if lo < 0 {
		lo = 0
	}
	hi := matchLine + contextLines + 1
	if hi > len(lines) {
		hi = len(lines)
	}
	return strings.Join(lines[lo:hi], "\n")
}

// isPyWord approximates Python's \w (unicode letters, numbers, marks, Pc, _).
func isPyWord(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) ||
		unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) ||
		unicode.Is(unicode.Pc, r) || r == '_'
}

// capRunes truncates s to n code points (Python s[:n] semantics).
func capRunes(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return string(rs[:n])
}

func emitRule(text, file, rule string, re *regexp.Regexp) []Match {
	var out []Match
	for _, loc := range re.FindAllStringIndex(text, -1) {
		out = append(out, Match{Rule: rule, File: file,
			Line: lineOf(text, loc[0]), Evidence: capRunes(text[loc[0]:loc[1]], 200)})
	}
	return out
}

func mustREs(patterns ...string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		out[i] = regexp.MustCompile(p)
	}
	return out
}

// ---------------------------------------------------------------------------
// SK-P-EA1 — upstream EA1_PATTERNS (excessive agency).
// Pattern 1's (?!\*|\w) lookahead becomes a post-check on the '*' capture.
// ---------------------------------------------------------------------------

var ea1StarRE = regexp.MustCompile(`(?i)(?:tools?|permissions?)\s*:[ \t]*\[?[ \t]*['"]?(\*)['"]?[ \t]*\]?`)

var ea1REs = mustREs(
	`(?i)(?:allow|grant|enable)\s+(?:access\s+to\s+)?(?:all|any|every)\s+tools?`,
	`(?i)(?:no|without)\s+(?:tool|permission|access|capability)\s+(?:restrictions?|constraints?|limitations?)`,
	`(?i)(?:call|invoke|use|execute)\s+(?:any|all|every)\s+(?:available\s+)?tools?`,
	`(?i)(?:unrestricted|unlimited|unconstrained)\s+(?:tool|function|api)\s+(?:access|use|calls?)`,
	`(?i)(?:can|may|should)\s+(?:freely|always)\s+(?:use|call|invoke)\s+(?:any|all)\s+(?:tools?|functions?|apis?)`,
	`(?i)tools?\s*:\s*\[\s*['"]shell['"].*?['"](?:file_write|network|http)['"]`,
	`(?i)(?:grant|give)\s+(?:full|complete|total)\s+(?:tool|function|api)\s+access`,
	`(?i)(?:execute|run)\s+(?:arbitrary|any)\s+(?:commands?|code|scripts?)`,
	`(?i)(?:no\s+)?(?:tool\s+)?(?:allow|block|deny)\s*(?:list|listing)\s*(?:is\s+)?(?:empty|disabled|off)`,
)

func detectEA1(d *Doc) []Match {
	var out []Match
	for _, loc := range ea1StarRE.FindAllStringSubmatchIndex(d.Text, -1) {
		starEnd := loc[3] // offset just past the '*' capture
		if starEnd < len(d.Text) {
			next, _ := utf8.DecodeRuneInString(d.Text[starEnd:])
			if next == '*' || isPyWord(next) {
				continue // upstream (?!\*|\w) rejects
			}
		}
		out = append(out, Match{Rule: "EA1", File: d.Path,
			Line: lineOf(d.Text, loc[0]), Evidence: capRunes(d.Text[loc[0]:loc[1]], 200)})
	}
	for _, re := range ea1REs {
		out = append(out, emitRule(d.Text, d.Path, "EA1", re)...)
	}
	return out
}

// ---------------------------------------------------------------------------
// SK-P-P5 — upstream DANGEROUS_ACTIONS (lane 1) + SUBSTANCE weight ladder.
// ---------------------------------------------------------------------------

var p5Lane1 = mustREs(
	`(?is)add\s+(?:a\s+)?(?:dash|pinch|bit|drop|amount)\s+of\s+(cyanide|arsenic|ricin|botulinum|strychnine|thallium|polonium|sarin|vx gas|mustard gas|anthrax|deadly nightshade|aconite|hemlock|oleander)`,
	`(?is)(?:include|incorporate|mix\s+in|stir\s+in)\s+(?:some\s+)?(cyanide|arsenic|ricin|botulinum|strychnine|thallium|polonium|sarin|vx gas|mustard gas|anthrax|deadly nightshade|aconite|hemlock|oleander)`,
	`(?is)for\s+(?:every|each|all)\s+(?:recipe|dish|meal|food).*?add.*?(cyanide|arsenic|ricin|botulinum|strychnine|thallium|polonium|sarin|vx gas|mustard gas|anthrax|deadly nightshade|aconite|hemlock|oleander)`,
	`(?is)mix\s+bleach\s+(?:and|with)\s+ammonia`,
	`(?is)mix\s+ammonia\s+(?:and|with)\s+bleach`,
	`(?is)(?:drink|consume|ingest)\s+(?:the\s+)?(?:bleach|antifreeze|drain\s+cleaner)`,
	`(?is)(?:you\s+should|try\s+to|please)\s+(?:hurt|harm|injure)\s+yourself`,
	`(?is)(?:cut|burn|starve)\s+yourself`,
	`(?is)(?:instructions?\s+(?:for|to|on)\s+)?(?:make|build|create)\s+(?:a\s+)?(?:bomb|explosive|weapon)`,
	`(?is)(?:how\s+to\s+)?(?:poison|kill|murder)\s+(?:someone|a\s+person|people)`,
)

var p5Substances = []string{
	"cyanide", "arsenic", "ricin", "botulinum", "strychnine", "thallium",
	"polonium", "sarin", "vx gas", "mustard gas", "anthrax",
	"deadly nightshade", "aconite", "hemlock", "oleander",
}

var p5InstructionalMarkers = []string{
	"step ", "recipe", "ingredient", "add ", "mix ", "stir ",
	"instructions", "how to", "directions", "prepare", "cook", "bake",
}

var p5EducationalMarkers = []string{
	"history of", "discovered by", "chemical compound", "periodic table",
	"forensic", "detective", "mystery", "agatha christie", "toxicology",
	"poison control", "symptoms of", "treatment for", "antidote",
	"emergency", "call 911", "warning", "danger", "never", "do not",
}

var p5WarningREs = mustREs(
	`(?i)do\s+not`, `(?i)never\s+(?:use|add|consume|eat|drink)`, `(?i)warning`,
	`(?i)danger`, `(?i)toxic`, `(?i)lethal`, `(?i)deadly`, `(?i)fatal`,
	`(?i)avoid`, `(?i)keep\s+away`,
)

func p5IsInstructional(content string, matchStart int) bool {
	lo := matchStart - 250
	if lo < 0 {
		lo = 0
	}
	hi := matchStart + 250
	if hi > len(content) {
		hi = len(content)
	}
	surrounding := strings.ToLower(content[lo:hi])
	for _, m := range p5InstructionalMarkers {
		if strings.Contains(surrounding, m) {
			return true
		}
	}
	return false
}

func detectP5(d *Doc) []Match {
	// lane 1: direct phrases — verbatim (?is) patterns
	type cand struct {
		line int
		conf float64
		ev   string
	}
	byLine := map[int]cand{}
	add := func(line int, conf float64, ev string) {
		if c, ok := byLine[line]; !ok || conf > c.conf {
			byLine[line] = cand{line, conf, ev}
		}
	}
	for _, re := range p5Lane1 {
		for _, loc := range re.FindAllStringIndex(d.Text, -1) {
			add(lineOf(d.Text, loc[0]), 0.95, capRunes(d.Text[loc[0]:loc[1]], 200))
		}
	}
	// lane 2: substance ladder — confidence gate >= 0.5
	for _, sub := range p5Substances {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(sub) + `\b`)
		for _, loc := range re.FindAllStringIndex(d.Text, -1) {
			conf := 0.7
			ctx := getContext(d.Text, loc[0], 5)
			if p5IsInstructional(d.Text, loc[0]) {
				conf = 0.95
			} else {
				lower := strings.ToLower(ctx)
				edu := false
				for _, m := range p5EducationalMarkers {
					if strings.Contains(lower, m) {
						edu = true
						break
					}
				}
				warn := false
				if !edu {
					for _, wre := range p5WarningREs {
						if wre.MatchString(ctx) {
							warn = true
							break
						}
					}
				}
				if edu {
					conf = 0.3
				} else if warn {
					conf = 0.2
				}
			}
			if conf >= 0.5 {
				add(lineOf(d.Text, loc[0]), conf, capRunes(d.Text[loc[0]:loc[1]], 200))
			}
		}
	}
	out := make([]Match, 0, len(byLine))
	for _, c := range byLine {
		out = append(out, Match{Rule: "P5", File: d.Path, Line: c.line, Evidence: c.ev})
	}
	return out
}

// ---------------------------------------------------------------------------
// SK-P-MP2 — upstream MP2_PATTERNS. Pattern 1 (.{2,20}?)\1{20,} is a
// backreference — ported as a bounded repeat scanner with lazy-unit semantics.
// ---------------------------------------------------------------------------

func mp2RepeatSpans(text string) [][2]int {
	var spans [][2]int
	n := len(text)
	for start := 0; start < n; {
		matched := false
		for ul := 2; ul <= 20 && start+ul <= n; ul++ {
			unit := text[start : start+ul]
			if strings.ContainsRune(unit, '\n') {
				continue // '.' (no DOTALL) excludes \n
			}
			reps := 1
			pos := start + ul
			for pos+ul <= n && strings.EqualFold(text[pos:pos+ul], unit) {
				reps++
				pos += ul
			}
			if reps >= 21 { // unit + \1{20,}
				spans = append(spans, [2]int{start, pos})
				start = pos
				matched = true
				break
			}
		}
		if !matched {
			start++
		}
	}
	return spans
}

var mp2LayoutASCII = "|-_=+"

func mp2IsLayoutOnly(span string) bool {
	if len([]rune(span)) > 256 {
		return false
	}
	var compact []rune
	for _, r := range span {
		if !unicode.IsSpace(r) {
			compact = append(compact, r)
		}
	}
	if len(compact) == 0 {
		return true
	}
	for _, r := range compact {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return false
		}
	}
	for _, r := range compact {
		if strings.ContainsRune(mp2LayoutASCII, r) {
			continue
		}
		if !(r >= 0x2500 && r <= 0x257F) && !(r >= 0x2580 && r <= 0x259F) {
			return false
		}
	}
	return true
}

func mp2SingleCharNoSpace(span string) bool {
	distinct := map[rune]bool{}
	for _, r := range span {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			distinct[r] = true
		}
	}
	return len(distinct) <= 1 && !strings.ContainsAny(span, " \t")
}

var mp2REs = mustREs(
	`(?im)(?:repeat|duplicate|echo)\s+(?:this|the\s+following)\s+(?:\d{3,}|many|hundreds?|thousands?)\s+times?`,
	`(?im)(?:fill|pad|stuff|flood|saturate)\s+(?:the\s+)?(?:context|memory|input|prompt|window|buffer)`,
	`(?im)(?:generate|produce|output|write)\s+(?:\d{4,}|thousands?\s+of|millions?\s+of)\s+(?:words?|characters?|tokens?|lines?)`,
	`(?im)(?:include|add|insert)\s+(?:enough|sufficient)\s+(?:text|content|padding|filler)\s+(?:to|until)\s+(?:fill|overflow|exhaust|push\s+out)`,
	`(?im)(?:displace|push\s+out|overwrite|crowd\s+out|evict)\s+(?:the\s+)?(?:original|system|previous|existing|safety)\s+(?:instructions?|prompt|context|rules?)`,
	`(?im)(?:exhaust|overflow|exceed)\s+(?:the\s+)?(?:context|token|memory)\s+(?:window|limit|budget|capacity)`,
)

func detectMP2(d *Doc) []Match {
	var out []Match
	keep := func(start, end int) {
		span := d.Text[start:end]
		if mp2IsLayoutOnly(span) || mp2SingleCharNoSpace(span) {
			return
		}
		out = append(out, Match{Rule: "MP2", File: d.Path,
			Line: lineOf(d.Text, start), Evidence: capRunes(span, 200)})
	}
	for _, sp := range mp2RepeatSpans(d.Text) {
		keep(sp[0], sp[1])
	}
	for _, re := range mp2REs {
		for _, loc := range re.FindAllStringIndex(d.Text, -1) {
			keep(loc[0], loc[1])
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// SK-P-OH1 — upstream OH1_PATTERNS + the subprocess fallback lane
// (non-python files only). Lookbehind/lookahead become boundary post-checks.
// ---------------------------------------------------------------------------

var oh1REs = mustREs(
	`(?im)exec\s*\(\s*(?:response|output|result|answer|completion|reply|generated)`,
	`(?im)eval\s*\(\s*(?:response|output|result|answer|completion|reply|generated)`,
	`(?im)os\.system\s*\(\s*(?:response|output|result|answer|completion)`,
	`(?im)os\.popen\s*\(\s*(?:response|output|result|answer|completion)`,
	`(?im)innerHTML\s*=\s*(?:response|output|result|answer|completion)`,
	`(?im)document\.write\s*\(\s*(?:response|output|result|answer|completion)`,
	`(?im)\.html\s*\(\s*(?:response|output|result|answer|completion)`,
	`(?im)dangerouslySetInnerHTML\s*=\s*\{`,
	`(?im)(?:execute|cursor\.execute|query)\s*\([^)]*(?:\+|%|\.format|f['"])\s*.*?(?:response|output|result)`,
	`(?im)f['"](?:SELECT|INSERT|UPDATE|DELETE)\s+.*?\{(?:response|output|result)`,
	`(?im)(?:run|execute|shell)\s+(?:the\s+)?(?:generated|model|llm|ai)\s+(?:output|response|code|command)`,
	`(?im)(?:pipe|pass|feed)\s+(?:the\s+)?(?:output|response|result)\s+(?:directly\s+)?(?:to|into)\s+(?:the\s+)?(?:shell|terminal|command|interpreter)`,
	`(?im)(?:use|insert|embed)\s+(?:the\s+)?(?:raw|unfiltered|unescaped|unsanitized)\s+(?:output|response)`,
)

var (
	oh1AnchorRE = regexp.MustCompile(`(?i)\bsubprocess\s*\.\s*(?:Popen|call|check_call|check_output|getoutput|getstatusoutput|run)\s*\(`)
	oh1KwRE     = regexp.MustCompile(`(?i)(answer|completion|generated|output|reply|response|result)`)
)

// oh1Fallback mirrors _SUBPROCESS_FALLBACK_PATTERN (re.VERBOSE): the keyword
// must be preceded by a char NOT in [-\w'"] and followed by a non-\w char.
func oh1Fallback(text string) [][2]int {
	var spans [][2]int
	cursor := 0
	for cursor < len(text) {
		anchor := oh1AnchorRE.FindStringIndex(text[cursor:])
		if anchor == nil {
			break
		}
		aStart, aEnd := cursor+anchor[0], cursor+anchor[1]
		// [^)]{0,1000}? — argument window ends at next ')' or 1000 chars.
		winEnd := aEnd + 1000
		if winEnd > len(text) {
			winEnd = len(text)
		}
		if idx := strings.IndexByte(text[aEnd:winEnd], ')'); idx >= 0 {
			winEnd = aEnd + idx
		}
		emitted := false
		for _, kw := range oh1KwRE.FindAllStringIndex(text[aEnd:winEnd], -1) {
			ks, ke := aEnd+kw[0], aEnd+kw[1]
			before := byte(0)
			if ks > 0 {
				before = text[ks-1]
			}
			// (?<![-\w'"]) — reject if preceding char is -, word char, or quote.
			if ks > 0 && (before == '-' || before == '\'' || before == '"' ||
				isPyWord(rune(before))) {
				continue
			}
			// (?!\w) — reject if a word char follows the keyword.
			if ke < len(text) && isPyWord(rune(text[ke])) {
				continue
			}
			spans = append(spans, [2]int{aStart, ke})
			cursor = ke
			emitted = true
			break // lazy [^)]{0,1000}? — earliest valid keyword wins
		}
		if !emitted {
			cursor = aEnd
		}
	}
	return spans
}

func detectOH1(d *Doc) []Match {
	var out []Match
	for _, re := range oh1REs {
		out = append(out, emitRule(d.Text, d.Path, "OH1", re)...)
	}
	if d.FileType != "python" {
		for _, sp := range oh1Fallback(d.Text) {
			out = append(out, Match{Rule: "OH1", File: d.Path,
				Line: lineOf(d.Text, sp[0]), Evidence: capRunes(d.Text[sp[0]:sp[1]], 200)})
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// SK-P-TM1 — upstream TM1_PATTERNS (regex lane; shell-lexer lane ceded).
// Pattern 3's atomic groups collapse to the equivalent non-atomic form (RE2
// is automata-based); dedup is (line, whitespace-normalized match).
// ---------------------------------------------------------------------------

var tm1REs = mustREs(
	`(?im)subprocess\.\w+\s*\([^)]*shell\s*=\s*True`,
	`(?im)Popen\s*\([^)]*shell\s*=\s*True`,
	// Upstream's {0,8192} window exceeds RE2's 1000-repeat cap; `*` is the
	// port (linear-time engine, no backtracking risk) — extras-only drift.
	`(?im)\b(?:rm|del|erase)\b(?:\\\r?\n)*[ \t](?:[ \t]|\\\r?\n)*(?:\\(?:\r?\n|[^\n])|[^\\\n|;&])*(?:\\)?-(?:rf|fr|r)(?:\\\r?\n)*[ \t](?:[ \t]|\\\r?\n)*[/~]`,
	`(?im)--force\s+(?:delete|remove|push|reset|clean)`,
	`(?im)--no-?(?:check|validate|confirm|protect|safe)\b`,
	`(?im)--skip-?(?:validation|verification|checks?|auth|tests?)`,
	`(?im)--allow-?(?:root|unrelated|unsafe)\b`,
	`(?im)\brm\b\s+(?:-[A-Za-z]+\s+)*(?:--\s+)?(?:['"][^'"]*/[^'"]*['"]|[^\s|;&>]*/[^\s|;&>]*)`,
	`(?im)\bshutil\.rmtree\s*\(\s*['"]\s*/`,
	`(?im)(?:chmod|chown)\s+[^|]*(?:777|666|a\+rwx)`,
	`(?im)git\s+push\s+[^|]*--force`,
	`(?im)git\s+reset\s+--hard`,
	`(?im)git\s+clean\s+-[fd]+x`,
	`(?im)\bgit\s+(?:am|commit|merge|push)\b[^\n|]*--no-verify\b`,
	`(?im)curl\s+[^|]*-k\b`,
	`(?im)curl\s+[^|]*--insecure\b`,
	`(?im)wget\s+[^|]*--no-check-certificate`,
	`(?im)\b(?:delete|remove)\s+['"]?/[^\s'"]{1,100}`,
	`(?im)(?:execute|query)\s*\(\s*f?['"].*?\{.*?\}.*?\b(?:DROP|DELETE|UPDATE|INSERT|ALTER|TRUNCATE)\b`,
	`(?im)(?:set|pass|use)\s+(?:the\s+)?(?:parameter|argument|flag|option)\s+(?:to\s+)?(?:shell\s*=\s*True|--force|-rf)\b`,
)

func detectTM1(d *Doc) []Match {
	seen := map[[2]any]bool{}
	var out []Match
	for _, re := range tm1REs {
		for _, loc := range re.FindAllStringIndex(d.Text, -1) {
			line := lineOf(d.Text, loc[0])
			norm := strings.Join(strings.Fields(strings.TrimSpace(d.Text[loc[0]:loc[1]])), " ")
			key := [2]any{line, norm}
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, Match{Rule: "TM1", File: d.Path, Line: line,
				Evidence: capRunes(d.Text[loc[0]:loc[1]], 200)})
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// SK-P-RA1 — upstream RA1_PATTERNS + _is_negated_safety_constraint.
// ---------------------------------------------------------------------------

var ra1REs = mustREs(
	`(?im)open\s*\(\s*__file__\s*,\s*['"]w`,
	`(?im)(?:Path|pathlib)\s*\(\s*__file__\s*\)\s*\.\s*write_text`,
	`(?im)(?:write|modify|edit|update|overwrite|patch)\s+(?:this\s+)?(?:skill(?:'s)?|SKILL\.md|skill\.md)`,
	`(?im)(?:modify|edit|change|rewrite|update|alter)\s+(?:your\s+own|its\s+own|this\s+skill(?:'s)?)\s+(?:code|source|file|script|config|configuration|instructions?|rules?)`,
	`(?im)(?:self[_-]?modify|self[_-]?update|self[_-]?rewrite|self[_-]?patch|self[_-]?evolve)`,
	`(?im)(?:rewrite|replace|overwrite)\s+(?:the\s+)?(?:current|existing|original)\s+(?:code|script|file|config|skill)`,
	`(?im)(?:append|add|inject|insert)\s+(?:code|instructions?|rules?)\s+(?:to|into)\s+(?:this|the\s+current|your\s+own)\s+(?:file|skill|script)`,
	`(?im)(?:generate|create|write)\s+(?:new|updated|replacement)\s+(?:code|version)\s+(?:for|of)\s+(?:this|itself|the\s+skill)`,
	`(?im)open\s*\(\s*['"](?:SKILL\.md|skill\.md)['"].*?['"]w`,
	`(?im)(?:update|modify|change|edit)\s+(?:the\s+)?(?:config|configuration|settings|parameters)\s+(?:of\s+)?(?:this|the\s+current)\s+(?:skill|agent|tool)`,
	`(?im)(?:disable|remove|delete|bypass)\s+(?:the\s+)?(?:safety|security|guard|protection|constraint)\s+(?:check|rule|mechanism|feature)`,
	`(?im)(?:comment\s+out|remove)\s+(?:the\s+)?(?:validation|verification|check|guard|assertion)`,
)

var ra1NegationRE = regexp.MustCompile(`(?i)(?:must\s+not|do\s+not|don't|never|should\s+not)\s+(?:(?:ever|again|directly|intentionally|explicitly|attempt\s+to|try\s+to)\s+){0,2}PHRASE$`)

// ra1Negated mirrors _is_negated_safety_constraint: the match phrase ends the
// clause and a negation+optional-gap prefix immediately precedes it.
func ra1Negated(content string, start, end int) bool {
	lineStart := strings.LastIndex(content[:start], "\n") + 1
	lineEnd := strings.Index(content[end:], "\n")
	if lineEnd < 0 {
		lineEnd = len(content)
	} else {
		lineEnd += end
	}
	line := content[lineStart:lineEnd]
	localStart := start - lineStart
	phrase := line[localStart : localStart+(end-start)]
	escaped := regexp.QuoteMeta(strings.TrimSpace(phrase))
	if escaped == "" {
		return false
	}
	clauseStart := -1
	for _, sep := range ".;:" {
		if i := strings.LastIndex(line[:localStart], string(sep)); i > clauseStart {
			clauseStart = i
		}
	}
	prefix := line[clauseStart+1 : localStart]
	re := regexp.MustCompile(strings.Replace(ra1NegationRE.String(), "PHRASE", escaped, 1))
	return re.MatchString(prefix + phrase)
}

func detectRA1(d *Doc) []Match {
	var out []Match
	for _, re := range ra1REs {
		for _, loc := range re.FindAllStringIndex(d.Text, -1) {
			if ra1Negated(d.Text, loc[0], loc[1]) {
				continue
			}
			out = append(out, Match{Rule: "RA1", File: d.Path,
				Line: lineOf(d.Text, loc[0]), Evidence: capRunes(d.Text[loc[0]:loc[1]], 200)})
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// SK-P-P6 — upstream P6_PATTERNS + heading suppressor + benign taxonomy.
// ---------------------------------------------------------------------------

var p6REs = mustREs(
	`(?im)(?:print|output|show|display|reveal|expose|return|echo)\s+(?:your\s+)?(?:full\s+)?(?:system\s+)?(?:prompt|instructions?|rules?|guidelines?|directives?)`,
	`(?im)(?:what\s+(?:are|is)\s+your|tell\s+me\s+your)\s+(?:system\s+)?(?:prompt|instructions?|rules?|guidelines?|directives?)`,
	`(?im)(?:output|print|display)\s+(?:the\s+)?(?:text|content|message)\s+(?:above|before)\s+(?:this|the\s+user)`,
	`(?im)(?:share|disclose|leak|give\s+me)\s+(?:your\s+)?(?:internal|hidden|system|secret)\s+(?:prompt|instructions?|rules?|config)`,
	`(?im)(?:dump|extract|copy)\s+(?:the\s+)?(?:system\s+)?(?:prompt|message|instructions?)\s+(?:to|into|as)`,
	`(?im)(?:begin|start)\s+(?:your\s+)?(?:response|output|reply)\s+with\s+(?:your\s+)?(?:system\s+)?(?:prompt|instructions?)`,
	`(?im)(?:first|before\s+anything)\s+(?:output|print|show|display)\s+(?:your\s+)?(?:full\s+)?(?:system\s+)?(?:prompt|instructions?)`,
	`(?im)(?:verbatim|exactly|word\s+for\s+word)\s+(?:repeat|output|print)\s+(?:your\s+)?(?:system\s+)?(?:prompt|instructions?)`,
)

// Anchored body of _BENIGN_PRINT_RULES_TAXONOMY, applied at line-start
// positions; the trailing (?=\Z|break) lookahead is a post-check.
var p6TaxonomyRE = regexp.MustCompile(`(?i)^[ \t]*["'` + "`" + `]{0,3}[ \t]*` +
	`(?:single-class[ \t]+selectors[ \t]+are[ \t]+honored[ \t]+(?:—|--|-)[ \t]+)?` +
	`descendant[ \t]*/[ \t]*compound[ \t]*/[ \t]*` +
	`(print[ \t]+rules)[ \t]+are` +
	`(?:[ \t]+|[ \t]*(?:\r\n|[\r\n\v\f\x1c-\x1e\x85\x{2028}\x{2029}])[ \t]+)(?:not|never)[ \t]+evaluated` +
	`(?:[ \t]+\((?:avoids?|to[ \t]+avoid)[ \t]+over-stripping[ \t]+content[ \t]+behind[ \t]+e\.g\.[ \t]+` + "`" + `?\.a[ \t]+\.b` + "`" + `?[ \t]+rules\))?` +
	`[ \t]*(?:[.!?][ \t]*)?["'` + "`" + `]{0,3}[ \t]*`)

var p6PrecedingDirectiveRE = regexp.MustCompile(`(?i)\b(?:you|your|agents?|assistants?|models?|llms?|bots?|must|shall|should|required|mandatory)\b|\bbefore[ \t]+(?:replying|responding)\b|\b(?:following|below|above|next|this|that|it|them|these|those|so|prior|previous|preceding|everything|all|former|latter|content|text|output|configuration|material)\b|\bthe[ \t]+same\b|\bwhat[ \t]+follows\b|:[ \t]*$`)

var p6NextLineRefRE = regexp.MustCompile(`(?i)\b(?:it|them|this|these|those|so|same|above|below|prior|previous|preceding|following|foregoing|everything|all|former|latter|content|text|output|configuration|material)\b|\b(?:the|this|that|these|those|same)[ \t]+(?:rules?|instructions?|prompts?|guidelines?|directives?|operations?|actions?)\b|\b(?:do|execute|perform|apply|follow|obey|use|print|output|show|display|reveal|expose|return|echo|repeat|share|disclose|publish|provide|send|copy|extract|dump|recite|summarize|translate|encode|write|save|forward|pipe)[ \t]+that\b`)

// prevNonblank mirrors _bounded_previous_nonblank_line (512-char window).
func prevNonblank(content string, offset int) (string, bool) {
	winStart := offset - 512
	if winStart < 0 {
		winStart = 0
	}
	parts := logicalLineBreak.Split(content[winStart:offset], -1)
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.TrimSpace(parts[i]) != "" {
			return parts[i], i > 0 || winStart == 0
		}
	}
	return "", winStart == 0
}

// nextNonblank mirrors _bounded_next_nonblank_line (512-char window).
func nextNonblank(content string, offset int) (string, bool) {
	winEnd := offset + 512
	if winEnd > len(content) {
		winEnd = len(content)
	}
	window := content[offset:winEnd]
	cursor := 0
	for _, br := range logicalLineBreak.FindAllStringIndex(window, -1) {
		line := window[cursor:br[0]]
		if strings.TrimSpace(line) != "" {
			return line, true
		}
		cursor = br[1]
	}
	if winEnd == len(content) {
		return window[cursor:], true
	}
	return "", false
}

// p6BenignTaxonomy mirrors _is_benign_print_rules_taxonomy.
func p6BenignTaxonomy(content string, start, end int) bool {
	winStart := start - 256
	if winStart < 0 {
		winStart = 0
	}
	winEnd := end + 256
	if winEnd > len(content) {
		winEnd = len(content)
	}
	// Candidate anchors: positions 0 or just past a logical line break,
	// within the window.
	var anchors []int
	if winStart == 0 {
		anchors = append(anchors, 0)
	}
	for _, br := range logicalLineBreak.FindAllStringIndex(content[:winEnd], -1) {
		if br[1] >= winStart {
			anchors = append(anchors, br[1])
		}
	}
	for _, anchor := range anchors {
		m := p6TaxonomyRE.FindStringSubmatchIndex(content[anchor:])
		if m == nil {
			continue
		}
		// target group must span exactly the P6 match
		if anchor+m[2] != start || anchor+m[3] != end {
			continue
		}
		candEnd := anchor + m[1]
		if candEnd != len(content) {
			// (?=\Z|break): the candidate must be followed by a line break
			br := logicalLineBreak.FindStringIndex(content[candEnd:])
			if br == nil || br[0] != 0 {
				continue
			}
			nextLine, nextComplete := nextNonblank(content, candEnd+br[1])
			if !nextComplete || p6NextLineRefRE.MatchString(nextLine) {
				continue
			}
		}
		if anchor == 0 {
			return true
		}
		prevLine, prevComplete := prevNonblank(content, anchor)
		if !prevComplete {
			return false
		}
		return !p6PrecedingDirectiveRE.MatchString(prevLine)
	}
	return false
}

func detectP6(d *Doc) []Match {
	var out []Match
	for _, re := range p6REs {
		for _, loc := range re.FindAllStringIndex(d.Text, -1) {
			// heading suppressor: exact "Output Rules" match inside the
			// benign markdown heading line only
			if d.FileType == "markdown" && d.Text[loc[0]:loc[1]] == "Output Rules" {
				ls := strings.LastIndex(d.Text[:loc[0]], "\n") + 1
				le := strings.Index(d.Text[loc[1]:], "\n")
				if le < 0 {
					le = len(d.Text)
				} else {
					le += loc[1]
				}
				if strings.TrimSpace(d.Text[ls:le]) == "## Output Rules (Both Modes)" {
					continue
				}
			}
			if p6BenignTaxonomy(d.Text, loc[0], loc[1]) {
				continue
			}
			out = append(out, Match{Rule: "P6", File: d.Path,
				Line: lineOf(d.Text, loc[0]), Evidence: capRunes(d.Text[loc[0]:loc[1]], 200)})
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// SK-P-TP1 — upstream _check_tp1, base64 lane only.
// ---------------------------------------------------------------------------

var (
	tp1DataURIRE = regexp.MustCompile(`data:text/[^;]+;base64,`)
	tp1B64RE     = regexp.MustCompile(`[A-Za-z0-9+/]{50,}={0,2}`)
)

func detectTP1Base64(d *Doc) []Match {
	var uriRanges [][2]int
	for _, m := range tp1DataURIRE.FindAllStringIndex(d.Text, -1) {
		uriRanges = append(uriRanges, [2]int{m[0], m[1]})
	}
	var out []Match
	for _, m := range tp1B64RE.FindAllStringIndex(d.Text, -1) {
		overlaps := false
		for _, r := range uriRanges {
			if m[0] >= r[0] && m[1] <= r[1]+200 {
				overlaps = true
				break
			}
		}
		if overlaps {
			continue
		}
		raw := d.Text[m[0]:m[1]]
		pad := (4 - len(raw)%4) % 4
		padded := raw + strings.Repeat("=", pad)
		decoded, err := base64.StdEncoding.DecodeString(padded)
		if err != nil || !utf8.Valid(decoded) {
			continue
		}
		ev := raw
		if len(ev) > 80 {
			ev = ev[:80] + "..."
		}
		line := strings.Count(d.Text[:m[0]], "\n") + 1
		out = append(out, Match{Rule: "TP1:base64", File: d.Path, Line: line, Evidence: ev})
	}
	return out
}

// ---------------------------------------------------------------------------
// SK-P-AS3 — upstream AS3_PATTERNS; #6's (?!CURRENT) becomes a capture reject.
// ---------------------------------------------------------------------------

var as3REs = mustREs(
	`(?im)(?:os\.listdir|os\.scandir|glob\.glob|Path\.iterdir)\s*\([^)]*\.(?:claude|codex|gemini)/skills?`,
	`(?im)(?:ls|find|dir)\s+[^|&;\n]*\.(?:claude|codex|gemini)/skills?`,
	`(?im)open\s*\(\s*['"][^'"]*SKILL\.md['"].*?\bother\b`,
	`(?im)(?:read|access|inspect|enumerate)\s+(?:all\s+)?(?:installed|available|other)\s+skills?(?:\s+in\s+(?:the\s+)?(?:skills?|agent)\s+(?:directory|folder))?`,
	`(?im)(?:list|discover|find|enumerate|identify)\s+(?:all\s+)?(?:other|installed|available)\s+(?:skills?|agents?|tools?)\s+(?:in\s+)?(?:the\s+)?(?:\.(?:claude|codex|gemini)|\$HOME)`,
	`(?im)(?:read|access|load)\s+(?:the\s+)?(?:SKILL|skill)\.md\s+(?:file\s+)?(?:of|from|for)\s+(?:another|other|different|all)\s+(?:skill|agent|tool)`,
)

var as3SegRE = regexp.MustCompile(`(?im)skills?/([A-Z][A-Za-z0-9_-]+)/SKILL\.md`)

func detectAS3(d *Doc) []Match {
	var out []Match
	for _, re := range as3REs {
		out = append(out, emitRule(d.Text, d.Path, "AS3", re)...)
	}
	for _, m := range as3SegRE.FindAllStringSubmatchIndex(d.Text, -1) {
		seg := d.Text[m[2]:m[3]]
		if strings.HasPrefix(seg, "CURRENT") {
			continue
		}
		out = append(out, Match{Rule: "AS3", File: d.Path,
			Line: lineOf(d.Text, m[0]), Evidence: capRunes(d.Text[m[0]:m[1]], 200)})
	}
	return out
}

// ---------------------------------------------------------------------------
// SK-P-AR2 — upstream AR2_PATTERNS. Evidence is the directive line (the
// annex's declared evidence rewrite), not the raw match.
// ---------------------------------------------------------------------------

var ar2REs = mustREs(
	`(?im)\b(?:do\s+not|don'?t|never|no)\s+(?:include|add|give|provide|use|append)?\s*(?:any\s+)?(?:warnings?|disclaimers?|caveats?)\b`,
	`(?im)\bwithout\s+(?:any\s+)?(?:warnings?|disclaimers?|caveats?|moralizing|moralising|judg(?:e?ment|ing))\b`,
	`(?im)\b(?:do\s+not|don'?t|never)\s+(?:lecture|moralize|moralise|preach|judge|warn)\b`,
	`(?im)\b(?:skip|omit|leave\s+out|drop)\s+(?:the\s+)?(?:disclaimer|warning|ethics|safety)\b`,
	`(?im)\b(?:no|without)\s+(?:ethical|moral)\s+(?:judg(?:e?ment)|considerations?|commentary|objections?)\b`,
	`(?im)\b(?:do\s+not|don'?t)\s+(?:apologize|apologise|say\s+sorry)\b`,
)

func detectAR2(d *Doc) []Match {
	var out []Match
	for _, re := range ar2REs {
		for _, loc := range re.FindAllStringIndex(d.Text, -1) {
			ls := strings.LastIndex(d.Text[:loc[0]], "\n") + 1
			le := strings.Index(d.Text[loc[1]:], "\n")
			if le < 0 {
				le = len(d.Text)
			} else {
				le += loc[1]
			}
			out = append(out, Match{Rule: "AR2", File: d.Path,
				Line:     lineOf(d.Text, loc[0]),
				Evidence: capRunes(strings.TrimSpace(d.Text[ls:le]), 200)})
		}
	}
	return out
}
