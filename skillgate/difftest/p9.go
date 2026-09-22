package difftest

// SK-P-P9 — port of whitespace_padding.detect_whitespace_padding.
// Offsets are CHAR offsets (Python str indices), matching upstream records.

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	p9VerticalBlankLines = 20
	p9HorizontalRunChars = 80
	p9BlockByteBudget    = 2048
	p9RatioThreshold     = 0.90
	p9RatioMinFileBytes  = 4096
	p9RepeatedCharThresh = 512
	p9RepeatedLineThresh = 64
	p9ReplacementDensity = 0.30
	p9ReplacementChar    = '\uFFFD'
)

// upstream _LINE_BOUNDARY_RE — narrower than logicalLineBreak (no \v\f\x1c-1e).
var p9LineBreak = regexp.MustCompile(`\r\n|\r|\n|\x{2028}|\x{2029}|\x85`)

var p9FenceRE = regexp.MustCompile(`^\s*(` + "```" + `|~~~)`)

var p9ZeroWidth = map[rune]bool{'\u200B': true, '\u200C': true, '\u200D': true, '\u2060': true, '\uFEFF': true}

// isPaddingChar mirrors upstream is_padding_char.
func isPaddingChar(r rune) bool {
	if r < 0x80 {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f'
	}
	if p9ZeroWidth[r] || r == '\u0085' || r == '\u180E' {
		return true
	}
	return unicode.Is(unicode.Zs, r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r)
}

// p9SplitLines mirrors _split_lines: separator-free lines + char offsets.
func p9SplitLines(content string) ([]string, []int) {
	var lines []string
	offsets := []int{0}
	pos := 0
	for _, br := range p9LineBreak.FindAllStringIndex(content, -1) {
		lines = append(lines, content[pos:br[0]])
		pos = br[1]
		offsets = append(offsets, utf8.RuneCountInString(content[:pos]))
	}
	lines = append(lines, content[pos:])
	offsets = append(offsets, utf8.RuneCountInString(content))
	return lines, offsets
}

func p9IsBlankLine(line string) bool {
	for _, r := range line {
		if !isPaddingChar(r) {
			return false
		}
	}
	return true
}

func p9FenceFlags(lines []string) []bool {
	inside := false
	flags := make([]bool, len(lines))
	for i, line := range lines {
		if p9FenceRE.MatchString(line) {
			flags[i] = true
			inside = !inside
		} else {
			flags[i] = inside
		}
	}
	return flags
}

func runeOffsetOf(text string, byteOff int) int {
	return utf8.RuneCountInString(text[:byteOff])
}

type p9Run struct {
	kind      string
	startOff  int // char offset
	startLine int
	length    int
	endOff    int
	followed  bool
}

func p9DetectVertical(content string, lines []string, offsets []int) []p9Run {
	var runs []p9Run
	blank := make([]bool, len(lines))
	for i, l := range lines {
		blank[i] = p9IsBlankLine(l)
	}
	i, n := 0, len(lines)
	for i < n {
		if !blank[i] {
			i++
			continue
		}
		j := i
		for j < n && blank[j] {
			j++
		}
		if j-i >= p9VerticalBlankLines {
			runs = append(runs, p9Run{
				kind: "vertical", startOff: offsets[i], startLine: i + 1,
				length: j - i, followed: j < n && !blank[j], endOff: offsets[j],
			})
		}
		i = j
	}
	return runs
}

func p9DetectHorizontal(lines []string, offsets []int, fileType string) []p9Run {
	var runs []p9Run
	var fences []bool
	if fileType == "markdown" {
		fences = p9FenceFlags(lines)
	}
	for idx, line := range lines {
		if fences != nil && fences[idx] {
			continue
		}
		rs := []rune(line)
		k, n := 0, len(rs)
		for k < n {
			if !isPaddingChar(rs[k]) {
				k++
				continue
			}
			start := k
			for k < n && isPaddingChar(rs[k]) {
				k++
			}
			if k-start >= p9HorizontalRunChars {
				runs = append(runs, p9Run{
					kind: "horizontal", startOff: offsets[idx] + start,
					startLine: idx + 1, length: k - start,
					followed: k < n, endOff: offsets[idx] + k,
				})
			}
		}
	}
	return runs
}

func p9DetectBlockAndRatio(content string) []p9Run {
	var runs []p9Run
	rs := []rune(content)
	n := len(rs)
	bestByteLen, bestStart, bestEnd := 0, -1, -1
	i := 0
	for i < n {
		if !isPaddingChar(rs[i]) {
			i++
			continue
		}
		start := i
		for i < n && isPaddingChar(rs[i]) {
			i++
		}
		byteLen := len(string(rs[start:i]))
		if byteLen > bestByteLen {
			bestByteLen, bestStart, bestEnd = byteLen, start, i
		}
	}
	if bestByteLen > p9BlockByteBudget && bestStart >= 0 {
		runs = append(runs, p9Run{
			kind: "block", startOff: bestStart,
			startLine: strings.Count(string(rs[:bestStart]), "\n") + 1,
			length:    bestEnd - bestStart, endOff: bestEnd,
		})
	}
	fileBytes := len(content)
	if fileBytes > p9RatioMinFileBytes {
		padBytes := 0
		for _, r := range rs {
			if isPaddingChar(r) {
				padBytes += len(string(r))
			}
		}
		if fileBytes > 0 && float64(padBytes)/float64(fileBytes) > p9RatioThreshold {
			runs = append(runs, p9Run{
				kind: "ratio", startOff: 0, startLine: 1,
				length: padBytes, endOff: 0,
			})
		}
	}
	return runs
}

func p9DetectRepetition(content string) []p9Run {
	var runs []p9Run
	rs := []rune(content)
	n := len(rs)
	idx := 0
	for idx < n {
		end := idx + 1
		for end < n && rs[end] == rs[idx] {
			end++
		}
		if end-idx >= p9RepeatedCharThresh && !isPaddingChar(rs[idx]) {
			runs = append(runs, p9Run{
				kind: "repetition", startOff: idx,
				startLine: strings.Count(string(rs[:idx]), "\n") + 1,
				length:    end - idx, followed: end < n, endOff: end,
			})
		}
		idx = end
	}
	lines, offsets := p9SplitLines(content)
	idx = 0
	for idx < len(lines) {
		end := idx + 1
		for end < len(lines) && lines[end] == lines[idx] && strings.TrimSpace(lines[idx]) != "" {
			end++
		}
		if end-idx >= p9RepeatedLineThresh {
			runs = append(runs, p9Run{
				kind: "repetition", startOff: offsets[idx], startLine: idx + 1,
				length: end - idx, followed: end < len(lines), endOff: offsets[end],
			})
		}
		idx = end
	}
	return runs
}

func detectP9(d *Doc) []Match {
	content := d.Text
	if content == "" {
		return nil
	}
	if float64(strings.Count(content, string(p9ReplacementChar)))/float64(len([]rune(content))) > p9ReplacementDensity {
		return nil
	}
	lines, offsets := p9SplitLines(content)
	vertical := p9DetectVertical(content, lines, offsets)
	horizontal := p9DetectHorizontal(lines, offsets, d.FileType)
	blockRatio := p9DetectBlockAndRatio(content)

	primary := append(append([]p9Run{}, vertical...), horizontal...)
	overlapsPrimary := func(r p9Run) bool {
		for _, p := range primary {
			if r.startOff < p.endOff && p.startOff < r.endOff {
				return true
			}
		}
		return false
	}
	var deduped []p9Run
	blockKept := false
	for _, r := range blockRatio {
		if r.kind == "block" && overlapsPrimary(r) {
			continue
		}
		if r.kind == "ratio" && (len(primary) > 0 || blockKept) {
			continue
		}
		if r.kind == "block" {
			blockKept = true
		}
		deduped = append(deduped, r)
	}
	var repetition []p9Run
	for _, r := range p9DetectRepetition(content) {
		if !overlapsPrimary(r) {
			repetition = append(repetition, r)
		}
	}
	all := append(append(append(append([]p9Run{}, vertical...), horizontal...), deduped...), repetition...)
	out := make([]Match, 0, len(all))
	for _, r := range all {
		out = append(out, Match{
			Rule: "P9", File: d.Path, Line: r.startLine,
			Evidence: r.kind + "@" + strconv.Itoa(r.startOff) + ":" + strconv.Itoa(r.length),
		})
	}
	return out
}
