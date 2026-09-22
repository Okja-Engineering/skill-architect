package skillgate

import (
	"regexp"
	"strings"
)

// Frontmatter is a minimal YAML-frontmatter view: top-level keys to their
// raw values, plus list items for keys declared as lists. It is deliberately
// not a YAML parser — the gate needs `name`, `description`, `allowed-tools`,
// `disable-model-invocation`, `context`, `compatibility` and nothing else.
type Frontmatter struct {
	Keys  map[string]string
	Lists map[string][]string
	Raw   string
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
	fm.Raw = text[4 : 3+end]
	var lastKey string
	for _, line := range strings.Split(fm.Raw, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		// List items belong to the most recent key.
		if strings.HasPrefix(trim, "- ") && lastKey != "" {
			fm.Lists[lastKey] = append(fm.Lists[lastKey], unquote(trim[2:]))
			continue
		}
		// Nested keys (indented, e.g. "metadata:" → "version:") are skipped;
		// only top-level keys are tracked.
		if line != trim && lastKey != "" && !strings.Contains(trim, ":") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 || (len(line) > 0 && (line[0] == ' ' || line[0] == '\t')) {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := unquote(strings.TrimSpace(line[idx+1:]))
		fm.Keys[key] = val
		lastKey = key
		if val == "" {
			lastKey = key // may hold a block list below
		}
		// Inline lists: key: [a, b]
		if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
			inner := val[1 : len(val)-1]
			for _, item := range strings.Split(inner, ",") {
				if s := unquote(strings.TrimSpace(item)); s != "" {
					fm.Lists[key] = append(fm.Lists[key], s)
				}
			}
		}
	}
	return fm
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

func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
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
