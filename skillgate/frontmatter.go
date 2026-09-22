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
// The projection loses structure — a nested map flattens to nothing, and
// map[string]string has no spelling for "unreadable" — but it never loses a
// key. A key the reader refused is present in Keys under the text that stood
// on its line, because dropping it would make a key that is there
// indistinguishable from one that is not, in the view every rule reads. What
// the projection cannot say, Root and Unreadable can: see Lookup.
type Frontmatter struct {
	// Keys holds each top-level entry whose value is a scalar, under that
	// value, and every other top-level entry under the text that stood on
	// the key's own line — empty for a block collection, the bracket text
	// for a flow one, the refused text for an unreadable one.
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

// Absent reports whether key k is genuinely not in the document — as
// opposed to unreadable, which is a different answer and must not become the
// same finding.
//
// Keys is a flat projection and `map[string]string` has no spelling for
// "unknown", so every rule that reached for a missing key got "" from a
// document the reader had refused whole and concluded the key was missing.
// A user told their keys are missing goes and adds keys that are already
// there; this project has shipped that defect twice and paid to fix it both
// times. Root can say what Keys cannot, so the question is asked here, once,
// and answered off Root:
//
//   - no frontmatter block at all (KindAbsent) — the keys really are not
//     there, and SK-I001 must still say so;
//   - a document the reader refused (KindUnreadable) — nothing is known
//     about any key, so nothing is absent, and SK-I006 reports the refusal
//     itself;
//   - a document that was read — ask it, and a key whose own value the
//     reader refused is present-but-unreadable, never missing.
//
// A key read as a blank scalar still counts as missing: "declared and left
// empty" is the same defect as "not declared", and that is the behaviour
// SK-I001 already had.
func (fm *Frontmatter) Absent(k string) bool {
	switch fm.Root.Kind() {
	case KindAbsent:
		return true
	case KindMapping:
		if fm.Root.Get(k).Kind() == KindUnreadable {
			return false
		}
		return strings.TrimSpace(fm.Keys[k]) == ""
	default:
		return false
	}
}

// Readable reports whether the reader read the whole document. False is what
// SK-I006 reports, and it is the state in which no rule may conclude that
// anything is absent.
func (fm *Frontmatter) Readable() bool { return len(fm.Unreadable) == 0 }

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
		case KindMapping, KindUnreadable:
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

// scriptLangs maps a bundled script's extension to the token that begins a
// line comment in that language.
//
// One table, two questions. "Is this file executable class?" and "which of
// its bytes does its interpreter never run?" are both properties of the same
// language, and answering them from two lists is how the two drift apart —
// a language added to one and not the other is a file the gate either scans
// with the wrong grammar or does not scan at all. isScript reads the keys;
// commentMarker reads the values.
var scriptLangs = map[string]string{
	".sh": "#", ".bash": "#", ".zsh": "#",
	".py": "#", ".rb": "#", ".pl": "#", ".ps1": "#",
	".js": "//", ".mjs": "//", ".ts": "//",
}

// isScript reports whether a path is an executable-class bundled file.
func isScript(path string) bool {
	for ext := range scriptLangs {
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

// isHarnessEntry reports whether a harness loads this file *by convention*
// rather than through a reference from another file: the skill manifest, a
// memory file it reads on session start, or a Cursor rule file it discovers
// and applies by glob. These are the doors into a bundle — they need no
// inbound reference to be reached, which is what makes them G003's entry
// set and never its candidates.
func isHarnessEntry(path string) bool {
	base := path[strings.LastIndex(path, "/")+1:]
	switch base {
	case "SKILL.md", "AGENTS.md", "CLAUDE.md", "GEMINI.md", "AGENT.md":
		return true
	}
	return strings.HasSuffix(path, ".mdc")
}

// isLoadedText reports whether a file's text enters an agent's context:
// markdown instruction files, harness memory files, and Cursor rule files.
func isLoadedText(path string) bool {
	return isHarnessEntry(path) || strings.HasSuffix(path, ".md")
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

// ruleH002 — the per-harness frontmatter leg. Declared here rather than in a
// tripwire group because its finding is built inline against the parsed
// frontmatter; the check below reads its identity out of this declaration, so
// the severity is stated once and RuleCatalog publishes the same one.
var ruleH002 = CatalogRule{
	ID: "SK-H002", Severity: SeverityInfo, Quality: "maintainability",
	Check: CheckHarnessFrontmatter,
}

// harnessRules is the pack, as RuleCatalog reads it.
var harnessRules = []CatalogRule{ruleH002}

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
				RuleID: ruleH002.ID, Severity: ruleH002.Severity, Quality: ruleH002.Quality,
				Message: "frontmatter keys valid for Claude Code, silently ignored by Cursor",
				File:    sf.Entry.Path, Evidence: strings.Join(ignored, ", "),
				EffortMinutes: 0, Source: "skillgate",
			})
		}
	}
	return out
}
