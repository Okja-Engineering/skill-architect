package skillgate

import (
	"strconv"
	"strings"
)

// A reader for the block-style YAML subset that SKILL.md frontmatter is
// written in.
//
// The set of shapes this reads is derived from the YAML 1.2 node kinds it
// claims, one production each, not from a list of layouts anyone thought
// of. In block context (§8.2):
//
//	document    ::= node(0)
//	node(c)     ::= block-mapping(c) | block-sequence(c)
//	              | block-scalar | flow-node | plain-scalar(c)
//	block-mapping(c)  ::= ( key ":" value(c) )+      every entry at column c
//	block-sequence(c) ::= ( "-" value(c) )+          every entry at column c
//	value(c)    ::= block-scalar                     header "|" or ">" (§8.1)
//	              | flow-node                        "[", "{", or a quoted
//	                                                 scalar (§7.3, §7.4),
//	                                                 which may span lines
//	              | plain-scalar(c)                  folding its more-indented
//	                                                 continuation lines (§8.1.3)
//	              | node(c') where c' > c            on the following lines
//	              | empty
//
// A block sequence entry's value is parsed as a node at the column of the
// text after its "-", which is what makes "- key: value" a compact mapping
// and "- - a" a compact nested sequence without either being its own case.
//
// Everything else is REFUSED BY NAME, never skipped. A construct the reader
// does not implement produces an unreadable node carrying a reason, so a key
// that is present but unreadable is never the same answer as a key that is
// absent. The refused set, and it is the complement of the grammar above:
// anchors, aliases, tags, the merge key, explicit keys, directives, document
// markers inside the block, tabs used for indentation, malformed flow
// collections, malformed block scalar headers, duplicate keys, and any line
// whose indentation opens no block.
//
// A refusal is scoped by one rule: it is entry-scope when the reader can
// still locate the end of the entry that carries it, and document-scope when
// it cannot. An entry-scope refusal leaves the rest of the document readable;
// a document-scope refusal makes the whole document unreadable, because a
// reader that cannot find the boundaries cannot honestly report what it did
// find. Partial parses are never handed out.
//
// Not implemented, and refused as above rather than approximated: multiple
// documents, anchors and aliases, explicit and complex keys, and tag
// resolution — a scalar's value is its text, never a typed bool or int.

// NodeKind is the vocabulary a lookup answers in. It is total: every path
// resolves to exactly one of these, and KindAbsent and KindUnreadable are
// deliberately different answers.
type NodeKind string

const (
	// KindAbsent — no node at this path. The nil *Node's kind.
	KindAbsent NodeKind = "absent"
	// KindScalar — a value that was read. Scalar() hands it over.
	KindScalar NodeKind = "scalar"
	// KindMapping — a mapping. Get()/Keys() walk it.
	KindMapping NodeKind = "mapping"
	// KindSequence — a sequence. Items() walks it.
	KindSequence NodeKind = "sequence"
	// KindUnreadable — a node is here and the reader would not read it.
	// Reason() says which construct; Scalar() refuses.
	KindUnreadable NodeKind = "unreadable"
)

type mapEntry struct {
	key  string
	node *Node
}

// Node is one node of a parsed frontmatter document. Its fields are
// unexported and kind-dependent, reachable only through accessors that check
// the kind first, so no caller can read a value out of a node that has none.
// Every accessor is nil-safe: the nil *Node is the absent node.
type Node struct {
	kind    NodeKind
	line    int // 1-based line in the containing file
	value   string
	entries []mapEntry
	items   []*Node
	reason  string
	// src is the text that stood after the key's colon on the key's own
	// line. It is empty for a block collection and holds the bracket text
	// for a flow one, which is what the compatibility projection needs.
	src string
}

// Kind reports what is at this path. The nil node is KindAbsent.
func (n *Node) Kind() NodeKind {
	if n == nil {
		return KindAbsent
	}
	return n.kind
}

// Line is the 1-based line the node starts on, or 0 when absent.
func (n *Node) Line() int {
	if n == nil {
		return 0
	}
	return n.line
}

// Reason names the refused construct, and is empty for every kind but
// KindUnreadable — an absent node never invents one.
func (n *Node) Reason() string {
	if n == nil {
		return ""
	}
	return n.reason
}

// Scalar hands over the value, and reports false for every other kind
// including KindUnreadable. There is no way to read "" out of a key the
// reader could not parse.
func (n *Node) Scalar() (string, bool) {
	if n == nil || n.kind != KindScalar {
		return "", false
	}
	return n.value, true
}

// Get resolves a key in a mapping. Absent, and not-a-mapping, are both nil.
func (n *Node) Get(key string) *Node {
	if n == nil || n.kind != KindMapping {
		return nil
	}
	for _, e := range n.entries {
		if e.key == key {
			return e.node
		}
	}
	return nil
}

// Keys lists a mapping's keys in document order.
func (n *Node) Keys() []string {
	if n == nil || n.kind != KindMapping {
		return nil
	}
	out := make([]string, 0, len(n.entries))
	for _, e := range n.entries {
		out = append(out, e.key)
	}
	return out
}

// Items lists a sequence's items in order.
func (n *Node) Items() []*Node {
	if n == nil || n.kind != KindSequence {
		return nil
	}
	return n.items
}

// Refusal names one construct the reader would not read, and where. An empty
// refusal list is the only way to say everything present was read.
type Refusal struct {
	Path   []string
	Line   int
	Reason string
}

// --- the parser ---------------------------------------------------------

type yamlLine struct {
	num    int    // 1-based line number in the containing file
	indent int    // leading spaces
	text   string // the line past its indentation
	raw    string // the line verbatim, for block scalar content
	blank  bool
	tabbed bool // a tab appears in the indentation
}

type yamlParser struct {
	lines    []yamlLine
	i        int
	path     []string
	refusals []Refusal
	fatal    *Refusal
	quiet    int // >0 while parsing a region whose refusals are already named
}

// parseYAMLSubset parses raw, whose first line is file line firstLine.
func parseYAMLSubset(raw string, firstLine int) (*Node, []Refusal) {
	p := &yamlParser{lines: splitYAMLLines(raw, firstLine)}
	root := p.parseBlockNode(0)
	if p.fatal == nil {
		p.skipIgnorable()
		if p.i < len(p.lines) {
			p.fail(p.lines[p.i], "indentation matches no open block")
		}
	}
	if p.fatal != nil {
		return &Node{kind: KindUnreadable, line: p.fatal.Line, reason: p.fatal.Reason}, p.refusals
	}
	return root, p.refusals
}

func splitYAMLLines(raw string, firstLine int) []yamlLine {
	var out []yamlLine
	for n, s := range strings.Split(raw, "\n") {
		l := yamlLine{num: firstLine + n, raw: s}
		i := 0
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			if s[i] == '\t' {
				l.tabbed = true
			} else if !l.tabbed {
				l.indent++
			}
			i++
		}
		l.text = s[i:]
		l.blank = l.text == ""
		out = append(out, l)
	}
	return out
}

// refuse records an entry-scope refusal and returns the unreadable node that
// stands in for the entry. src is the text that stood on the entry's own
// line, which the compatibility projection reports so that a key the reader
// refused is never missing from the flat view — missing is how a key that is
// present becomes indistinguishable from one that is absent.
func (p *yamlParser) refuse(l yamlLine, reason, src string) *Node {
	if p.quiet == 0 {
		p.refusals = append(p.refusals, Refusal{
			Path: append([]string(nil), p.path...), Line: l.num, Reason: reason,
		})
	}
	return &Node{kind: KindUnreadable, line: l.num, reason: reason, src: src}
}

// indicatorToken reports the length of the token an anchor ("&"), alias
// ("*") or tag ("!") indicator introduces. It is zero when nothing follows
// the indicator, and an indicator that introduces no token starts no node:
// the text is then read as the plain scalar it looks like rather than
// refused. That is the one deliberate leniency here, and it is load-bearing.
// "allowed-tools: *" is not well-formed YAML, but a harness reading it
// leniently grants every tool, so a gate that declined to read it would be
// silent about the grant it exists to catch.
func indicatorToken(s string) int {
	n := 0
	for n+1 < len(s) {
		switch s[n+1] {
		case ' ', '\t', ',', '[', ']', '{', '}':
			return n
		}
		n++
	}
	return n
}

func indicatorName(c byte) string {
	switch c {
	case '&':
		return "an anchor"
	case '*':
		return "an alias"
	}
	return "a tag"
}

// fail records a document-scope refusal: the reader has lost the boundaries
// and will hand out nothing rather than a partial parse.
func (p *yamlParser) fail(l yamlLine, reason string) {
	if p.fatal != nil {
		return
	}
	r := Refusal{Path: append([]string(nil), p.path...), Line: l.num, Reason: reason}
	p.refusals = append(p.refusals, r)
	p.fatal = &r
}

func (p *yamlParser) skipIgnorable() {
	for p.i < len(p.lines) {
		l := p.lines[p.i]
		if l.blank || strings.HasPrefix(l.text, "#") {
			p.i++
			continue
		}
		return
	}
}

// structuralRefusal names the constructs that can only be recognised where a
// line is being read as structure — inside a block scalar they are content.
func (p *yamlParser) structuralRefusal(l yamlLine) bool {
	switch {
	case l.tabbed:
		p.fail(l, "a tab is used for indentation")
	case strings.HasPrefix(l.text, "%"):
		p.fail(l, "a YAML directive")
	case l.text == "..." || strings.HasPrefix(l.text, "... "):
		p.fail(l, "a document end marker inside the block")
	case l.text == "---" || strings.HasPrefix(l.text, "--- "):
		p.fail(l, "a document start marker inside the block")
	case l.text == "?" || strings.HasPrefix(l.text, "? "):
		p.fail(l, "an explicit key")
	default:
		return false
	}
	return true
}

// parseBlockNode parses the node whose first line is the next line indented
// at least minIndent. It returns nil when there is no such line.
func (p *yamlParser) parseBlockNode(minIndent int) *Node {
	p.skipIgnorable()
	if p.i >= len(p.lines) {
		return nil
	}
	l := p.lines[p.i]
	if l.indent < minIndent && !l.tabbed {
		return nil
	}
	if p.structuralRefusal(l) {
		return nil
	}
	if isSeqEntry(l.text) {
		return p.parseBlockSequence(l.indent)
	}
	if _, _, ok := splitYAMLKey(l.text); ok {
		return p.parseBlockMapping(l.indent)
	}
	p.i++
	return p.parsePlainScalar(l.indent-1, l.text, l)
}

func isSeqEntry(text string) bool {
	return text == "-" || strings.HasPrefix(text, "- ")
}

func (p *yamlParser) parseBlockMapping(col int) *Node {
	node := &Node{kind: KindMapping, line: p.lines[p.i].num}
	seen := map[string]int{}
	for {
		p.skipIgnorable()
		if p.i >= len(p.lines) || p.fatal != nil {
			return node
		}
		l := p.lines[p.i]
		if l.indent < col && !l.tabbed {
			return node
		}
		if p.structuralRefusal(l) {
			return node
		}
		if l.indent > col {
			p.fail(l, "indentation matches no open block")
			return node
		}
		key, rest, ok := splitYAMLKey(l.text)
		if !ok {
			p.fail(l, "a line that is neither a mapping entry nor a sequence entry")
			return node
		}

		p.path = append(p.path, key)
		var value *Node
		if key == "<<" {
			// The merge key resolves against an anchor this reader does not
			// track, so its entry is refused whole; the value is consumed
			// only to find where the entry ends.
			value = p.refuse(l, "the merge key (<<)", rest)
			p.quiet++
			p.parseValue(col, rest, l)
			p.quiet--
		} else {
			value = p.parseValue(col, rest, l)
		}
		if at, dup := seen[key]; dup {
			// Which value the key holds is exactly what the reader cannot
			// say, so it holds neither.
			// Which value the key holds is what the reader cannot say, so
			// the flat view reports no text for it either — but the key is
			// still there, which is what keeps it apart from an absent one.
			node.entries[at].node = p.refuse(l, "a duplicate key", "")
		} else {
			seen[key] = len(node.entries)
			node.entries = append(node.entries, mapEntry{key: key, node: value})
		}
		p.path = p.path[:len(p.path)-1]
	}
}

func (p *yamlParser) parseBlockSequence(col int) *Node {
	node := &Node{kind: KindSequence, line: p.lines[p.i].num}
	for {
		p.skipIgnorable()
		if p.i >= len(p.lines) || p.fatal != nil {
			return node
		}
		l := p.lines[p.i]
		if l.indent < col && !l.tabbed {
			return node
		}
		if p.structuralRefusal(l) {
			return node
		}
		if l.indent > col || !isSeqEntry(l.text) {
			p.fail(l, "indentation matches no open block")
			return node
		}

		p.path = append(p.path, strconv.Itoa(len(node.items)))
		var item *Node
		if l.text == "-" {
			p.i++
			item = p.parseBlockNode(col + 1)
			if item == nil {
				item = &Node{kind: KindScalar, line: l.num}
			}
		} else {
			// The item is a node at the column of the text after the dash,
			// which is what makes "- key: v" a compact mapping and "- - a"
			// a compact sequence without either being a separate case.
			inner := l
			inner.indent = l.indent + 2
			inner.text = strings.TrimLeft(l.text[2:], " ")
			inner.indent += len(l.text[2:]) - len(inner.text)
			p.lines[p.i] = inner
			item = p.parseBlockNode(inner.indent)
			if item == nil {
				item = &Node{kind: KindScalar, line: l.num}
			}
		}
		node.items = append(node.items, item)
		p.path = p.path[:len(p.path)-1]
	}
}

// parseValue parses the value of a mapping entry at column col whose key
// line is l and whose text after the colon is rest.
func (p *yamlParser) parseValue(col int, rest string, l yamlLine) *Node {
	rest = strings.TrimSpace(rest)
	if rest == "" || strings.HasPrefix(rest, "#") {
		p.i++
		if n := p.parseBlockNode(col + 1); n != nil {
			return n
		}
		return &Node{kind: KindScalar, line: l.num}
	}
	switch rest[0] {
	case '|', '>':
		return p.parseBlockScalar(col, rest, l)
	case '&', '*', '!':
		if indicatorToken(rest) > 0 {
			return p.refuseEntry(col, l, indicatorName(rest[0]), rest)
		}
		// No token follows, so no node begins here: fall through to the
		// plain scalar below.
	case '[', '{', '"', '\'':
		return p.parseFlowValue(rest, l)
	}
	p.i++
	return p.parsePlainScalar(col, rest, l)
}

// refuseEntry refuses a value whose extent the reader can still find: the
// key's own line plus any block indented under it.
func (p *yamlParser) refuseEntry(col int, l yamlLine, what, src string) *Node {
	n := p.refuse(l, what, src)
	p.i++
	for p.i < len(p.lines) && (p.lines[p.i].blank || p.lines[p.i].indent > col) {
		p.i++
	}
	return n
}

// parseBlockScalar reads a literal ("|") or folded (">") scalar, §8.1.
func (p *yamlParser) parseBlockScalar(col int, header string, l yamlLine) *Node {
	folded := header[0] == '>'
	chomp := byte('c') // clip
	explicit := 0
	rest := header[1:]
	if i := strings.IndexByte(rest, '#'); i >= 0 {
		rest = rest[:i]
	}
	rest = strings.TrimSpace(rest)
	for rest != "" {
		switch {
		case rest[0] == '+' || rest[0] == '-':
			if chomp != 'c' {
				return p.refuseEntry(col, l, "a malformed block scalar header", header)
			}
			chomp = rest[0]
		case rest[0] >= '1' && rest[0] <= '9':
			if explicit != 0 {
				return p.refuseEntry(col, l, "a malformed block scalar header", header)
			}
			explicit = int(rest[0] - '0')
		default:
			return p.refuseEntry(col, l, "a malformed block scalar header", header)
		}
		rest = rest[1:]
	}

	p.i++
	contentIndent := 0
	if explicit > 0 {
		contentIndent = col + explicit
	} else {
		for j := p.i; j < len(p.lines); j++ {
			if p.lines[j].blank {
				continue
			}
			if p.lines[j].indent <= col {
				break
			}
			contentIndent = p.lines[j].indent
			break
		}
	}
	if contentIndent == 0 {
		// No content line is indented under the header: an empty scalar.
		return &Node{kind: KindScalar, line: l.num, value: chompBlock("", chomp)}
	}
	var content []string
	for p.i < len(p.lines) {
		cur := p.lines[p.i]
		if cur.blank {
			content = append(content, "")
			p.i++
			continue
		}
		if cur.indent < contentIndent {
			break
		}
		text := cur.raw
		if len(text) >= contentIndent {
			text = text[contentIndent:]
		} else {
			text = strings.TrimLeft(text, " ")
		}
		content = append(content, text)
		p.i++
	}
	// Trailing blank lines past the last content line belong to the scalar
	// only under keep chomping; either way they are blank, so consuming
	// them here costs the block parser nothing.
	var body string
	if folded {
		body = foldLines(content, true)
	} else {
		body = strings.Join(content, "\n")
		if len(content) > 0 {
			body += "\n"
		}
	}
	return &Node{kind: KindScalar, line: l.num, value: chompBlock(body, chomp)}
}

func chompBlock(s string, chomp byte) string {
	switch chomp {
	case '+':
		return s
	case '-':
		return strings.TrimRight(s, "\n")
	default:
		s = strings.TrimRight(s, "\n")
		if s == "" {
			return ""
		}
		return s + "\n"
	}
}

// foldLines applies YAML's line folding (§8.1.3) to already-dedented lines.
// A break between two ordinary lines folds to a space; an empty line
// contributes a line break; and when moreIndentAware, a break adjacent to a
// more-indented line is kept rather than folded. The result carries the
// final line break, which chomping then decides about.
func foldLines(lines []string, moreIndentAware bool) string {
	if len(lines) == 0 {
		return ""
	}
	moreIndented := func(s string) bool {
		return moreIndentAware && s != "" && (s[0] == ' ' || s[0] == '\t')
	}
	var b strings.Builder
	b.WriteString(lines[0])
	for i := 1; i < len(lines); i++ {
		prev, cur := lines[i-1], lines[i]
		switch {
		case cur == "":
			// The break before an empty line is folded away; the empty line
			// itself contributes the break.
			b.WriteString("\n")
		case prev == "":
			b.WriteString(cur)
		case moreIndented(prev) || moreIndented(cur):
			b.WriteString("\n")
			b.WriteString(cur)
		default:
			b.WriteString(" ")
			b.WriteString(cur)
		}
	}
	b.WriteString("\n")
	return b.String()
}

// parsePlainScalar reads a plain scalar at column col, folding the
// continuation lines indented under it. Plain scalars carry no trailing
// break, and a " #" outside a quoted scalar begins a comment.
func (p *yamlParser) parsePlainScalar(col int, first string, l yamlLine) *Node {
	lines := []string{stripPlainComment(first)}
	for p.i < len(p.lines) {
		cur := p.lines[p.i]
		if cur.blank {
			lines = append(lines, "")
			p.i++
			continue
		}
		if cur.indent <= col || cur.tabbed || strings.HasPrefix(cur.text, "#") {
			break
		}
		lines = append(lines, stripPlainComment(cur.text))
		p.i++
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	value := strings.TrimRight(foldLines(lines, false), "\n")
	return &Node{kind: KindScalar, line: l.num, value: value}
}

// stripPlainComment removes a " #" comment from a plain scalar line and
// trims it. A '#' that is not preceded by a space is part of the scalar.
func stripPlainComment(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '#' && (i == 0 || s[i-1] == ' ' || s[i-1] == '\t') {
			return strings.TrimSpace(s[:i])
		}
	}
	return strings.TrimSpace(s)
}

// parseFlowValue reads a flow node — a flow mapping, a flow sequence, or a
// quoted scalar (§7.3, §7.4). A flow node may span lines, so the reader
// hands the scanner the rest of the block and converts what it consumed back
// into a line position.
func (p *yamlParser) parseFlowValue(rest string, l yamlLine) *Node {
	var b strings.Builder
	b.WriteString(rest)
	for j := p.i + 1; j < len(p.lines); j++ {
		b.WriteByte('\n')
		b.WriteString(p.lines[j].raw)
	}
	text := b.String()

	src := strings.TrimSpace(strings.SplitN(rest, "\n", 2)[0])
	s := &flowScanner{text: text, line: l.num}
	node := s.node()
	if s.err != "" {
		// A flow node whose delimiters balance still tells the reader where
		// the entry ends, so only its own entry is refused. One whose
		// delimiters do not balance takes the document with it.
		end, closed := skipFlowNode(text)
		if !closed {
			p.fail(yamlLine{num: s.line}, s.err)
			return nil
		}
		n := p.refuse(l, s.err, src)
		p.i += strings.Count(text[:end], "\n") + 1
		return n
	}
	// Whatever follows the flow node on its own line must be a comment. The
	// tail stops at the newline: skipping past it would read the next
	// entry's line as this value's trailing content.
	tail := s.text[s.pos:]
	if nl := strings.IndexByte(tail, '\n'); nl >= 0 {
		tail = tail[:nl]
	}
	if t := strings.TrimSpace(tail); t != "" && !strings.HasPrefix(t, "#") {
		p.fail(yamlLine{num: s.line}, "trailing content after a flow node")
		return nil
	}
	p.i += strings.Count(text[:s.pos], "\n") + 1
	if node.Kind() != KindScalar {
		// The compatibility projection reports a flow collection by the text
		// that stood on the key's line.
		node.src = src
	}
	return node
}

// skipFlowNode finds the end of the flow node starting at text[0] by
// matching its delimiters, without reading its content — which is how the
// reader locates the end of a flow value it could not read. It reports
// false when the delimiters never balance.
func skipFlowNode(text string) (int, bool) {
	depth := 0
	for i := 0; i < len(text); i++ {
		switch c := text[i]; c {
		case '[', '{':
			depth++
		case ']', '}':
			depth--
			if depth == 0 {
				return i + 1, true
			}
		case '"', '\'':
			j := i + 1
			for j < len(text) {
				if text[j] == '\\' && c == '"' {
					j += 2
					continue
				}
				if text[j] == c {
					if c == '\'' && j+1 < len(text) && text[j+1] == '\'' {
						j += 2
						continue
					}
					break
				}
				j++
			}
			if j >= len(text) {
				return 0, false
			}
			i = j
			if depth == 0 {
				return i + 1, true
			}
		case '\n':
			if depth == 0 {
				return i, true
			}
		}
	}
	return len(text), depth == 0
}

type flowScanner struct {
	text string
	pos  int
	line int
	err  string
}

func (s *flowScanner) skipSpace() {
	for s.pos < len(s.text) {
		switch s.text[s.pos] {
		case ' ', '\t':
			s.pos++
		case '\n':
			s.line++
			s.pos++
		case '#':
			for s.pos < len(s.text) && s.text[s.pos] != '\n' {
				s.pos++
			}
		default:
			return
		}
	}
}

func (s *flowScanner) fail(msg string) *Node {
	if s.err == "" {
		s.err = msg
	}
	return nil
}

func (s *flowScanner) node() *Node {
	s.skipSpace()
	if s.pos >= len(s.text) {
		return s.fail("an unclosed flow collection")
	}
	switch c := s.text[s.pos]; c {
	case '[':
		return s.sequence()
	case '{':
		return s.mapping()
	case '"', '\'':
		return s.quoted()
	case '&', '*', '!':
		if indicatorToken(s.text[s.pos:]) > 0 {
			return s.fail(indicatorName(c))
		}
		// No token follows, so no node begins here: read the plain scalar.
	}
	return s.plain()
}

func (s *flowScanner) sequence() *Node {
	node := &Node{kind: KindSequence, line: s.line}
	s.pos++ // '['
	for {
		s.skipSpace()
		if s.pos >= len(s.text) {
			return s.fail("an unclosed flow collection")
		}
		if s.text[s.pos] == ']' {
			s.pos++
			return node
		}
		item := s.node()
		if s.err != "" {
			return nil
		}
		node.items = append(node.items, item)
		s.skipSpace()
		if s.pos >= len(s.text) {
			return s.fail("an unclosed flow collection")
		}
		switch s.text[s.pos] {
		case ',':
			s.pos++
		case ']':
			s.pos++
			return node
		default:
			return s.fail("a malformed flow sequence")
		}
	}
}

func (s *flowScanner) mapping() *Node {
	node := &Node{kind: KindMapping, line: s.line}
	s.pos++ // '{'
	for {
		s.skipSpace()
		if s.pos >= len(s.text) {
			return s.fail("an unclosed flow collection")
		}
		if s.text[s.pos] == '}' {
			s.pos++
			return node
		}
		var key string
		if c := s.text[s.pos]; c == '"' || c == '\'' {
			k := s.quoted()
			if s.err != "" {
				return nil
			}
			key, _ = k.Scalar()
		} else {
			k := s.plainKey()
			if s.err != "" {
				return nil
			}
			key, _ = k.Scalar()
		}
		s.skipSpace()
		if s.pos >= len(s.text) || s.text[s.pos] != ':' {
			return s.fail("a flow mapping entry with no value")
		}
		s.pos++
		value := s.node()
		if s.err != "" {
			return nil
		}
		node.entries = append(node.entries, mapEntry{key: key, node: value})
		s.skipSpace()
		if s.pos >= len(s.text) {
			return s.fail("an unclosed flow collection")
		}
		switch s.text[s.pos] {
		case ',':
			s.pos++
		case '}':
			s.pos++
			return node
		default:
			return s.fail("a malformed flow mapping")
		}
	}
}

// quoted reads a single- or double-quoted scalar, which may span lines.
func (s *flowScanner) quoted() *Node {
	q := s.text[s.pos]
	start := s.line
	s.pos++
	var b strings.Builder
	for s.pos < len(s.text) {
		c := s.text[s.pos]
		switch {
		case c == q && q == '\'':
			if s.pos+1 < len(s.text) && s.text[s.pos+1] == '\'' {
				b.WriteByte('\'')
				s.pos += 2
				continue
			}
			s.pos++
			return &Node{kind: KindScalar, line: start, value: b.String()}
		case c == q:
			s.pos++
			return &Node{kind: KindScalar, line: start, value: b.String()}
		case c == '\\' && q == '"':
			s.pos++
			if s.pos >= len(s.text) {
				return s.fail("an unclosed quoted scalar")
			}
			s.writeEscape(&b)
		case c == '\n':
			// A quoted scalar folds its breaks like a plain one.
			s.line++
			s.pos++
			for s.pos < len(s.text) && (s.text[s.pos] == ' ' || s.text[s.pos] == '\t') {
				s.pos++
			}
			b.WriteByte(' ')
		default:
			b.WriteByte(c)
			s.pos++
		}
	}
	return s.fail("an unclosed quoted scalar")
}

func (s *flowScanner) writeEscape(b *strings.Builder) {
	c := s.text[s.pos]
	s.pos++
	switch c {
	case 'n':
		b.WriteByte('\n')
	case 't':
		b.WriteByte('\t')
	case 'r':
		b.WriteByte('\r')
	case '0':
		b.WriteByte(0)
	case 'a':
		b.WriteByte(7)
	case 'b':
		b.WriteByte(8)
	case 'f':
		b.WriteByte(12)
	case 'v':
		b.WriteByte(11)
	case 'e':
		b.WriteByte(27)
	case 'x', 'u', 'U':
		n := map[byte]int{'x': 2, 'u': 4, 'U': 8}[c]
		if s.pos+n > len(s.text) {
			s.fail("a malformed escape in a quoted scalar")
			return
		}
		v, err := strconv.ParseUint(s.text[s.pos:s.pos+n], 16, 32)
		if err != nil {
			s.fail("a malformed escape in a quoted scalar")
			return
		}
		s.pos += n
		b.WriteRune(rune(v))
	default:
		// \\ \" \/ \<space> and the rest stand for themselves.
		b.WriteByte(c)
	}
}

// plain reads a flow-context plain scalar, which ends at a flow indicator.
func (s *flowScanner) plain() *Node {
	start := s.pos
	line := s.line
	for s.pos < len(s.text) {
		c := s.text[s.pos]
		if c == ',' || c == '[' || c == ']' || c == '{' || c == '}' || c == '\n' {
			break
		}
		if c == ':' && (s.pos+1 >= len(s.text) || s.text[s.pos+1] == ' ' || s.text[s.pos+1] == '\n') {
			break
		}
		if c == '#' && s.pos > start && (s.text[s.pos-1] == ' ' || s.text[s.pos-1] == '\t') {
			break
		}
		s.pos++
	}
	v := strings.TrimSpace(s.text[start:s.pos])
	if v == "" {
		return s.fail("an empty flow scalar")
	}
	return &Node{kind: KindScalar, line: line, value: v}
}

// plainKey reads a flow mapping key, which additionally ends at ':'.
func (s *flowScanner) plainKey() *Node {
	start := s.pos
	line := s.line
	for s.pos < len(s.text) {
		c := s.text[s.pos]
		if c == ':' || c == ',' || c == '[' || c == ']' || c == '{' || c == '}' || c == '\n' {
			break
		}
		s.pos++
	}
	v := strings.TrimSpace(s.text[start:s.pos])
	if v == "" {
		return s.fail("a flow mapping key with no name")
	}
	return &Node{kind: KindScalar, line: line, value: v}
}

// splitYAMLKey splits "key: rest" into its parts. A key is a quoted scalar
// or a plain one, and ends at the ":" that is followed by a space or the end
// of the line — which is why a value may itself contain a colon.
func splitYAMLKey(text string) (key, rest string, ok bool) {
	if text == "" {
		return "", "", false
	}
	if q := text[0]; q == '"' || q == '\'' {
		s := &flowScanner{text: text, line: 1}
		n := s.quoted()
		if s.err != "" {
			return "", "", false
		}
		after := strings.TrimLeft(text[s.pos:], " ")
		if !strings.HasPrefix(after, ":") {
			return "", "", false
		}
		key, _ = n.Scalar()
		return key, strings.TrimLeft(after[1:], " "), true
	}
	for i := 0; i < len(text); i++ {
		if text[i] != ':' {
			continue
		}
		if i+1 == len(text) {
			return strings.TrimSpace(text[:i]), "", i > 0
		}
		if text[i+1] == ' ' {
			return strings.TrimSpace(text[:i]), strings.TrimLeft(text[i+1:], " "), i > 0
		}
	}
	return "", "", false
}
