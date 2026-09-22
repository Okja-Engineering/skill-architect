package difftest

// SK-P-AE6 — port of the letter-spacing lane of artifact_integrity's AE6
// (_concealed_instruction_run_spans + _spacing_span_has_security_signal).
// All offsets are CHAR offsets. NFKC + casefold use golang.org/x/text.

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

const (
	ae6MinRunLetters     = 6
	ae6MaxSecurityPhrase = 79
	ae6MaxSecurityTerm   = 20
	ae6RuntimeInterval   = 4096
	ae6MaxBenignRunChars = 96
)

var ae6SecurityTerms = []string{
	"bypass", "disregard", "ignore", "instructions", "jailbreak", "override",
	"previousinstructions", "restrictions", "securityconstraints",
	"silentlysend", "sshkey", "unfiltered", "unrestricted", "userdata",
}

var ae6ExactTerms = map[string]bool{
	"accesstoken": true, "apikey": true, "credential": true,
	"credentials": true, "password": true, "privatekey": true,
	"secrettoken": true, "systemprompt": true,
}

// Upstream _LETTER_SPACING_COMMAND_PATTERNS, verbatim alternations (fullmatch).
var ae6CommandPatterns = mustREs(
	`^(?:(?:immediately|covertly|secretly|silently|quietly|please|now))?(?:disclosing|displaying|outputting|disclosed|discloses|displayed|outputted|returning|revealing|disclose|displays|exposing|printing|returned|revealed|display|dumping|echoing|exposed|exposes|outputs|printed|returns|reveals|showing|dumped|echoed|echoes|expose|output|prints|return|reveal|showed|dumps|print|shown|shows|dump|echo|show)(?:(?:available|sensitive|private|remote|secret|stored|system|local|users|full|user|your|all|any|the|a)){0,3}(?:systeminstructions|systemprompts|instructions|systemprompt|prompts|prompt|rules)(?:(?:immediately|now))?$`,
	`^(?:(?:immediately|covertly|secretly|silently|quietly|please|now))?(?:exfiltrating|exfiltrated|exfiltrates|exfiltrate|stealing|leaking|leaked|steals|stolen|leaks|steal|stole|leak)(?:(?:available|sensitive|private|remote|secret|stored|system|local|users|full|user|your|all|any|the|a)){0,3}(?:systeminstructions|systemprompts|accesstokens|secrettokens|systemprompt|accesstoken|credentials|privatekeys|secrettoken|credential|privatekey|passwords|password|userdata|apikeys|secrets|sshkeys|apikey|secret|sshkey|tokens|token|files|data|file)(?:(?:immediately|now))?$`,
	`^(?:(?:immediately|covertly|secretly|silently|quietly|please|now))?(?:transmitting|transmitted|forwarding|forwarded|transmits|uploading|forwards|transmit|uploaded|forward|posting|sending|uploads|posted|upload|posts|sends|post|send|sent)(?:(?:available|sensitive|private|remote|secret|stored|system|local|users|full|user|your|all|any|the|a)){0,3}(?:systeminstructions|systemprompts|accesstokens|secrettokens|systemprompt|accesstoken|credentials|privatekeys|secrettoken|credential|privatekey|passwords|password|userdata|apikeys|secrets|sshkeys|apikey|secret|sshkey|tokens|token)(?:(?:immediately|now))?$`,
	`^(?:(?:immediately|covertly|secretly|silently|quietly|please|now))?(?:destroying|destroyed|deleting|destroys|removing|deleted|deletes|destroy|erasing|removed|removes|delete|erased|erases|remove|wiping|erase|wiped|wipes|wipe)(?:(?:available|sensitive|private|remote|secret|stored|system|local|users|full|user|your|all|any|the|a)){0,3}(?:credentials|directories|credential|directory|passwords|workspace|password|userdata|history|secrets|memory|secret|tokens|files|token|data|file)(?:(?:immediately|now))?$`,
)

// [^\W\d_] — Python \w minus digits minus underscore (unicode classes + the
// non-underscore connector punctuation Pc\{U+005F}).
const ae6LetterCls = `\p{L}\p{Mn}\p{Mc}\p{Nl}\p{No}‿⁀⁔ﳍﳎﳏﳠﳡﳢ`

// [^\w]|_ — everything that is not a Python word char, plus underscore.
const ae6GapCls = `\p{L}\p{N}\p{Mn}\p{Mc}‿⁀⁔ﳍﳎﳏﳠﳡﳢ`

var ae6CandidateRE = regexp.MustCompile(`(?:[` + ae6LetterCls + `](?:[^` + ae6GapCls + `]|_)+){5}[` + ae6LetterCls + `]`)

var ae6BypassSumRE = regexp.MustCompile(`^b *\+ *y *\+ *p *\+ *a *\+ *s *\+ *s$`)
var ae6SpellPreRE = regexp.MustCompile(`^(?:the\s+)?spelling\s+(?:example|exercise)\s*$`)
var ae6SpellSufRE = regexp.MustCompile(`^\s*(?:demonstrates|illustrates|shows)\s+(?:the\s+)?letter\s+order[.!?]?\s*$`)
var ae6ExprPreRE = regexp.MustCompile(`^(?:the\s+)?(?:expression|formula)\s*$`)
var ae6ExprSufRE = regexp.MustCompile(`^\s*(?:is|equals)\s+(?:a\s+)?spelling\s+(?:example|exercise)[.!?]?\s*$`)

var ae6BenignTerms = map[string]bool{"bypass": true, "restrictions": true}

// common.py LINE_BREAK_CHARS.
const lineBreakChars = "\r\n\v\f\x1c\x1d\x1e\x85  "

func isAlphaRune(r rune) bool { return unicode.IsLetter(r) }

func isAlnumRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

// ae6Fold mirrors NFKC → ASCII_CONFUSABLE_SKELETON translate → casefold →
// keep letters. Multi-rune folds are kept (upstream keeps len>1 folds here).
func ae6Fold(r rune) string {
	nfkc := norm.NFKC.String(string(r))
	var b strings.Builder
	for _, c := range nfkc {
		if rep, ok := asciiConfusableSkeleton[c]; ok {
			b.WriteString(rep)
		} else {
			b.WriteRune(c)
		}
	}
	folded := cases.Fold().String(b.String())
	var out []rune
	for _, c := range folded {
		if unicode.IsLetter(c) {
			out = append(out, c)
		}
	}
	return string(out)
}

// concealedInstructionRunSpans ports _concealed_instruction_run_spans.
// Operates on runes; yields char-offset spans.
func concealedInstructionRunSpans(text string) [][2]int {
	if !ae6CandidateRE.MatchString(text) {
		return nil
	}
	rs := []rune(text)
	n := len(rs)
	var spans [][2]int
	offset := 0
	for offset < n {
		if !isAlphaRune(rs[offset]) || (offset > 0 && isAlphaRune(rs[offset-1])) {
			offset++
			continue
		}
		runStart := offset
		lastLetterEnd := offset + 1
		letterCount := 1
		cursor := lastLetterEnd
		for cursor < n {
			gapStart := cursor
			for cursor < n && !isAlnumRune(rs[cursor]) {
				cursor++
			}
			if gapStart == cursor || cursor >= n || !isAlphaRune(rs[cursor]) {
				break
			}
			letterCount++
			lastLetterEnd = cursor + 1
			cursor = lastLetterEnd
			if cursor < n && isAlphaRune(rs[cursor]) {
				break
			}
		}
		if letterCount >= ae6MinRunLetters {
			spans = append(spans, [2]int{runStart, lastLetterEnd})
			offset = lastLetterEnd
		} else {
			offset = runStart + 1
		}
	}
	return spans
}

func ae6PhraseHasSignal(phrase string) bool {
	if ae6ExactTerms[phrase] {
		return true
	}
	for _, re := range ae6CommandPatterns {
		if re.MatchString(phrase) {
			return true
		}
	}
	return false
}

// ae6BoundedLineContext mirrors _bounded_same_line_context over runes.
func ae6BoundedLineContext(rs []rune, start, end int) (string, string, bool) {
	prefixStart := start - ae6MaxBenignRunChars
	if prefixStart < 0 {
		prefixStart = 0
	}
	prefix := string(rs[prefixStart:start])
	if idx := lastBreakIndex(prefix); idx >= 0 {
		prefix = prefix[idx:]
	} else if prefixStart > 0 {
		return "", "", false
	}
	suffixEnd := end + ae6MaxBenignRunChars
	if suffixEnd > len(rs) {
		suffixEnd = len(rs)
	}
	suffix := string(rs[end:suffixEnd])
	if idx := firstBreakIndex(suffix); idx >= 0 {
		suffix = suffix[:idx]
	} else if suffixEnd < len(rs) {
		return "", "", false
	}
	return strings.ToLower(prefix), strings.ToLower(suffix), true
}

// lastBreakIndex returns the byte index just past the last line break.
func lastBreakIndex(s string) int {
	locs := logicalLineBreak.FindAllStringIndex(s, -1)
	if len(locs) == 0 {
		return -1
	}
	return locs[len(locs)-1][1]
}

func firstBreakIndex(s string) int {
	loc := logicalLineBreak.FindStringIndex(s)
	if loc == nil {
		return -1
	}
	return loc[0]
}

// ae6BenignNotation mirrors _spacing_span_is_benign_notation (rune-based).
func ae6BenignNotation(rs []rune, span [2]int) bool {
	start, end := span[0], span[1]
	semanticEnd := end
	if end < len(rs) && isAlphaRune(rs[end]) {
		semanticEnd = end - 1
	}
	for semanticEnd > start && strings.ContainsRune(lineBreakChars, rs[semanticEnd-1]) {
		semanticEnd--
	}
	if semanticEnd <= start || semanticEnd-start > ae6MaxBenignRunChars {
		return false
	}
	rawRun := string(rs[start:semanticEnd])
	var letters strings.Builder
	for _, r := range rs[start:semanticEnd] {
		if isAlphaRune(r) {
			letters.WriteString(ae6Fold(r))
		}
	}
	phrase := letters.String()
	if !ae6BenignTerms[phrase] {
		return false
	}
	prefix, suffix, ok := ae6BoundedLineContext(rs, start, semanticEnd)
	if !ok {
		return false
	}
	if phrase == "bypass" &&
		ae6BypassSumRE.MatchString(strings.ToLower(rawRun)) &&
		start <= ae6MaxBenignRunChars &&
		len(rs)-semanticEnd <= ae6MaxBenignRunChars &&
		strings.TrimSpace(string(rs[:start])) == "" &&
		strings.Trim(string(rs[semanticEnd:]), " \t.!?"+lineBreakChars) == "" {
		return true
	}
	return (ae6SpellPreRE.MatchString(prefix) && ae6SpellSufRE.MatchString(suffix)) ||
		(ae6ExprPreRE.MatchString(prefix) && ae6ExprSufRE.MatchString(suffix))
}

// ae6SpanHasSignal mirrors _spacing_span_has_security_signal.
func ae6SpanHasSignal(rs []rune, span [2]int) bool {
	if ae6BenignNotation(rs, span) {
		return false
	}
	var letters []string
	var phraseParts []string
	letterChars, phraseChars := 0, 0
	phraseOverflow := false
	overlap := ""
	for offset := span[0]; offset < span[1]; offset++ {
		if !isAlphaRune(rs[offset]) {
			continue
		}
		folded := ae6Fold(rs[offset])
		if folded == "" {
			continue
		}
		letters = append(letters, folded)
		letterChars += len([]rune(folded))
		if !phraseOverflow {
			phraseChars += len([]rune(folded))
			if phraseChars <= ae6MaxSecurityPhrase {
				phraseParts = append(phraseParts, folded)
			} else {
				phraseParts = nil
				phraseOverflow = true
			}
		}
		if letterChars < ae6RuntimeInterval {
			continue
		}
		block := overlap + strings.Join(letters, "")
		if ae6AnyTerm(block) {
			return true
		}
		brs := []rune(block)
		if len(brs) > ae6MaxSecurityTerm-1 {
			overlap = string(brs[len(brs)-(ae6MaxSecurityTerm-1):])
		} else {
			overlap = block
		}
		letters = nil
		letterChars = 0
	}
	block := overlap + strings.Join(letters, "")
	if ae6AnyTerm(block) {
		return true
	}
	if phraseOverflow {
		return false
	}
	if ae6PhraseHasSignal(strings.Join(phraseParts, "")) {
		return true
	}
	return len(phraseParts) > 0 &&
		span[1] < len(rs) &&
		isAlphaRune(rs[span[1]]) &&
		ae6PhraseHasSignal(strings.Join(phraseParts[:len(phraseParts)-1], ""))
}

func ae6AnyTerm(block string) bool {
	for _, term := range ae6SecurityTerms {
		if strings.Contains(block, term) {
			return true
		}
	}
	return false
}

func detectAE6Spacing(d *Doc) []Match {
	rs := []rune(d.Text)
	var out []Match
	for _, span := range concealedInstructionRunSpans(d.Text) {
		if ae6SpanHasSignal(rs, span) {
			// char offset → byte offset for lineOf
			byteOff := len(string(rs[:span[0]]))
			out = append(out, Match{Rule: "AE6:spacing", File: d.Path,
				Line:     lineOf(d.Text, byteOff),
				Evidence: capRunes(string(rs[span[0]:span[1]]), 200)})
			break // upstream emits at most one spacing finding per file
		}
	}
	return out
}
