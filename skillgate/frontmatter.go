package skillgate

import (
	"regexp"
	"strconv"
	"strings"
)

// Frontmatter is the parsed `---` block of a SKILL.md. Root is the document
// as the reader in yaml.go read it; Keys and Lists are a flat projection of
// its top-level entries, kept because every rule reads through them.
//
// The projection is lossy by construction and in exactly one way: Keys is a
// map[string]string, so it has no spelling for a key that is present and
// unreadable, and such a key is missing from it. Root and Unreadable are
// where that distinction lives — see Lookup.
type Frontmatter struct {
	// Keys holds each top-level entry whose value is a scalar, under that
	// value; each entry whose value is a collection is here too, under the
	// text that stood on the key's own line — empty for a block collection,
	// the bracket text for a flow one.
	Keys map[string]string
	// Lists holds each top-level entry whose value is a sequence of
	// scalars, flow or block alike.
	Lists map[string][]string
	Raw   string

	// Root is the parsed document. Its Kind is KindAbsent when the file
	// carries no frontmatter block at all.
	Root *Node
	// Unreadable names every construct the reader refused, in document
	// order. Empty is the only way to say "everything present was read".
	Unreadable []Refusal
}

// Lookup resolves a path through the document. Sequence steps are decimal
// indices. The zero answer is a nil *Node, whose Kind is KindAbsent — so a
// key that is present but unreadable (KindUnreadable, carrying a Reason) is
// never the same answer as a key that is not there.
func (fm *Frontmatter) Lookup(path ...string) *Node {
	n := fm.Root
	for _, step := range path {
		switch n.Kind() {
		case KindMapping:
			n = n.Get(step)
		case KindSequence:
			i, err := strconv.Atoi(step)
			if err != nil || i < 0 || i >= len(n.Items()) {
				return nil
			}
			n = n.Items()[i]
		default:
			return nil
		}
	}
	return n
}

// ParseFrontmatter extracts the `---`-delimited block from a SKILL.md.
func ParseFrontmatter(text string) *Frontmatter {
	fm := &Frontmatter{Keys: map[string]string{}, Lists: map[string][]string{}}
	if !strings.HasPrefix(text, "---") {
		return fm
	}
	end := strings.Index(text[3:], "\n---")
	if end < 0 {
		return fm
	}
	// The block begins on line 2 of the file, past the opening delimiter.
	fm.Raw = text[4 : 3+end]
	fm.Root, fm.Unreadable = parseYAMLSubset(fm.Raw, 2)
	fm.project()
	return fm
}

// project flattens the document's top-level entries into Keys and Lists. A
// refused document projects to nothing: a reader that lost the boundaries
// has no honest partial answer to give.
func (fm *Frontmatter) project() {
	if fm.Root.Kind() != KindMapping {
		return
	}
	for _, k := range fm.Root.Keys() {
		n := fm.Root.Get(k)
		switch n.Kind() {
		case KindScalar:
			v, _ := n.Scalar()
			fm.Keys[k] = v
		case KindMapping:
			fm.Keys[k] = n.src
		case KindSequence:
			fm.Keys[k] = n.src
			var items []string
			for _, it := range n.Items() {
				if v, ok := it.Scalar(); ok && v != "" {
					items = append(items, v)
				}
			}
			if len(items) > 0 {
				fm.Lists[k] = items
			}
		}
	}
}

// Body returns the text after the frontmatter block.
func Body(text string) string {
	if !strings.HasPrefix(text, "---") {
		return text
	}
	end := strings.Index(text[3:], "\n---")
	if end < 0 {
		return text
	}
	return text[3+end:]
}

// skillFiles returns the inspected files that carry skill frontmatter:
// SKILL.md and Cursor .mdc rule files.
func skillFiles(l *Ledger) []*FileContent {
	var out []*FileContent
	for i := range l.Files {
		f := &l.Files[i]
		p := f.Entry.Path
		if f.Entry.Outcome != "inspected" {
			continue
		}
		base := p[strings.LastIndex(p, "/")+1:]
		// SKILL.md only — a .mdc Cursor rule file has no `name`/`description`
		// frontmatter contract and can't carry `allowed-tools`; including it
		// makes I001/T013 false-fire on every Cursor rules bundle (agnix's
		// CUR-* rules own .mdc conformance).
		if base == "SKILL.md" {
			out = append(out, f)
		}
	}
	return out
}

// isScript reports whether a path is an executable-class bundled file.
func isScript(path string) bool {
	for _, ext := range []string{".sh", ".bash", ".py", ".js", ".mjs", ".ts", ".rb", ".pl", ".ps1", ".zsh"} {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	// scripts/, bin/, and hooks/ are executable dirs by convention —
	// including extensionless entries (bin/skill-recon with a shebang is
	// still a script; a hooks/ payload file is run by the hook config).
	// Obvious non-executables in those dirs don't count.
	for _, d := range []string{"scripts/", "bin/", "hooks/"} {
		if strings.HasPrefix(path, d) || strings.Contains(path, "/"+d) {
			for _, doc := range []string{".json", ".md", ".mdc", ".toml", ".txt", ".lock", ".map"} {
				if strings.HasSuffix(path, doc) {
					return false
				}
			}
			return true
		}
	}
	return false
}

// isLoadedText reports whether a file's text enters an agent's context:
// markdown instruction files, harness memory files, and Cursor rule files.
func isLoadedText(path string) bool {
	base := path[strings.LastIndex(path, "/")+1:]
	switch base {
	case "AGENTS.md", "CLAUDE.md", "GEMINI.md", "AGENT.md":
		return true
	}
	return strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".mdc")
}

// reSkillName is the spec name charset — lowercase alphanumerics and
// hyphens, ≤64 chars. Cursor's stock skills system additionally requires
// the name to match the parent folder (SK-I002's other leg).
var reSkillName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$|^[a-z0-9]$`)

func validSkillName(n string) bool { return reSkillName.MatchString(n) }

// ccOnlyFrontmatterKeys are SKILL.md frontmatter keys Claude Code honors
// that Cursor's stock skills system silently ignores. Cursor walks eight
// roots for SKILL.md (four native plus .claude/ and .codex/ compatibility
// roots) and honors name + description; the rest of the CC surface is a
// silent no-op there. agnix's CUR-* rules cover .mdc only, so this
// per-harness validity leg is ours — reported informationally, never a
// downgrade of the T013 boundary check.
var ccOnlyFrontmatterKeys = []string{"allowed-tools", "disable-model-invocation", "context", "when_to_use"}

// harnessFrontmatterCheck is the per-harness validity dimension's slice-1
// leg (Pack D's SK-H002): informational findings only.
func harnessFrontmatterCheck(l *Ledger, _ *Target) []Finding {
	var out []Finding
	for _, sf := range skillFiles(l) {
		if !strings.HasSuffix(sf.Entry.Path, "SKILL.md") {
			continue
		}
		fm := ParseFrontmatter(sf.Text)
		var ignored []string
		for _, k := range ccOnlyFrontmatterKeys {
			if _, ok := fm.Keys[k]; ok {
				ignored = append(ignored, k)
			}
		}
		if len(ignored) > 0 {
			out = append(out, Finding{
				RuleID: "SK-H002", Severity: SeverityInfo, Quality: "maintainability",
				Message: "frontmatter keys valid for Claude Code, silently ignored by Cursor",
				File:    sf.Entry.Path, Evidence: strings.Join(ignored, ", "),
				EffortMinutes: 0, Source: "skillgate",
			})
		}
	}
	return out
}
