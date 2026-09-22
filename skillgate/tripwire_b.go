package skillgate

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

// Pack B rules T010–T020: cross-harness snooping, excessive agency, MCP and
// hook smuggling, persistence.

var (
	// T010/T011/T012 — reads of other harnesses' config and credential
	// stores. Applied to script files and code blocks, not prose.
	reClaudePaths = regexp.MustCompile(`(?i)(~/|~?/?\.?/)?\.claude/|\$HOME/\.claude|~/\.codex|\.codex/|\.gemini/|\.continue/`)
	reCursorPaths = regexp.MustCompile(`(?i)(~/|~?/?\.?/)?\.cursor/|\$HOME/\.cursor|\.cursor/mcp\.json|\.cursor/hooks\.json`)
	reCursorCreds = regexp.MustCompile(`(?i)state\.vscdb|cursorAuth|ItemTable`)
	reAccessToken = regexp.MustCompile(`(?i)\baccessToken\b`)
	reCursorCtx   = regexp.MustCompile(`(?i)cursor|state\.vscdb|ItemTable`)

	// T020 — persistence / self-modification targets. [pi]: harness state
	// roots include `.pi/` — `~/.pi/agent/trust.json` is pi's only
	// input-loading guard, so writing it auto-answers the trust prompt —
	// and the whole `.cursor/` state root, not only hooks/rules.
	rePersistPath = regexp.MustCompile(`(?i)(~/\.(bashrc|zshrc|profile|bash_profile|zprofile)|\.claude/settings|\.claude/CLAUDE|\.cursor/|\.pi/|agent/trust\.json|/etc/(profile|crontab)|LaunchAgents|systemd|crontab\s|launchctl|systemctl\s+(enable|start)|defaults\s+write)`)
	// The redirect legs need a non-word preceding char (`x > file`) so `<cwd>`
	// placeholders and `>=` comparisons don't fire; the target must look like
	// a path (/, ~, . or quote) so `a > b` comparisons don't. Echo/cat bodies
	// exclude `<>` for the same placeholder reason. writeFileSync/appendFile/
	// createWriteStream/open-w cover programmatic persistence — the pi repo's
	// real trust-file writes never touch shell.
	rePersistVerb = regexp.MustCompile(`(?im)((^|[\s;|0-9)])>>?\s*["']?[/~.]|tee\s+\S|cp\s+(-\S+\s+)?\S+\s|mv\s+(-\S+\s+)?\S+\s|sed\s+-i\b|install\s+(-\S+\s+)?\S+\s|cat\s+[^<>\n]*>|echo\s+[^<>\n]*>>?\s*["']?[/~.]|(fs\.)?(write|append)File(Sync)?\s*\(|createWriteStream|open\s*\([^)\n]*['"][wax]\b)`)
	// reProgWrite identifies the programmatic-write legs of rePersistVerb —
	// for those, the persist path must sit in the call's first argument
	// (the target), not in a later string-literal argument.
	reProgWrite = regexp.MustCompile(`(write|append)File|createWriteStream|open\s*\(`)

	// T016 — MCP auto-approval without consent.
	reAutoApprove = regexp.MustCompile(`(?i)"(autoApprove|alwaysAllow|auto-approve|auto_approve)"\s*:\s*(true|\[)|"(requireApproval|require_approval|confirm)"\s*:\s*false|"disabled"\s*:\s*false[^}]*autoApprove`)

	// T015 — wildcard HTTP binding.
	reWildcardBind = regexp.MustCompile(`(?i)"(url|host|bind|listen)"\s*:\s*"[^"]*(0\.0\.0\.0|http://\*|::/0)|0\.0\.0\.0:\d+`)
)

// isHarnessConfig reports whether a bundled path is harness configuration:
// hook configs, MCP configs, and settings files a skill should never ship.
func isHarnessConfig(path string) bool {
	p := strings.ToLower(path)
	base := p[strings.LastIndex(p, "/")+1:]
	if strings.HasPrefix(p, ".claude/") || strings.HasPrefix(p, ".cursor/") ||
		strings.HasPrefix(p, ".codex/") || strings.Contains(p, "/.claude/") ||
		strings.Contains(p, "/.cursor/") || strings.Contains(p, "/.codex/") ||
		strings.HasPrefix(p, "hooks/") || strings.Contains(p, "/hooks/") {
		return strings.HasSuffix(base, ".json") || strings.HasSuffix(base, ".toml") ||
			base == "settings.json" || base == "settings.local.json"
	}
	switch base {
	case "settings.json", "settings.local.json", "hooks.json", "mcp.json", ".mcp.json":
		return true
	}
	return false
}

// ownHarnessDir returns the harness dotdir the file itself lives under.
// Self-references to it (e.g. `$CLAUDE_PROJECT_DIR/.claude/hooks/x` from a
// file inside `.claude/`) are bundle home plumbing, not cross-harness snoop.
func ownHarnessDir(p string) string {
	for _, d := range []string{".claude/", ".codex/", ".cursor/", ".gemini/", ".continue/"} {
		if strings.HasPrefix(p, d) || strings.Contains(p, "/"+d) {
			return d
		}
	}
	return ""
}

// suppressSameDirRefs strips bare references to the file's own harness dir
// while keeping ~/ and $HOME refs, which reach the user's real home — the
// actual cross-boundary case.
func suppressSameDirRefs(line, own string) string {
	const home = "\x00HOMEDIR\x00"
	l := strings.ReplaceAll(line, "~/"+own, home)
	l = strings.ReplaceAll(l, "$HOME/"+own, home)
	l = strings.ReplaceAll(l, own, "SELFREF/")
	return strings.ReplaceAll(l, home, "~/"+own)
}

// harnessPathFindings runs a harness-path regex over a file's lines,
// suppressing same-dotdir self-references.
func harnessPathFindings(f *FileContent, re *regexp.Regexp) []string {
	own := ownHarnessDir(f.Entry.Path)
	var out []string
	for _, line := range strings.Split(normPathSep(f.Text), "\n") {
		l := line
		if own != "" {
			l = suppressSameDirRefs(l, own)
		}
		if re.MatchString(l) {
			out = append(out, strings.TrimSpace(line))
		}
	}
	return out
}

// T010 — reads of other harnesses' config directories.
var ruleT010 = rule{
	id: "SK-T010", sev: SeverityHigh, quality: "security", effort: 15,
	msg:   "touches another harness's config directory (.claude/.codex/.gemini/.continue)",
	files: isScript,
	scan:  func(f *FileContent) []string { return harnessPathFindings(f, reClaudePaths) },
}

// T011 — reads of Cursor config paths.
var ruleT011 = rule{
	id: "SK-T011", sev: SeverityHigh, quality: "security", effort: 15,
	msg:   "reads Cursor config or state paths (.cursor/)",
	files: isScript,
	scan:  func(f *FileContent) []string { return harnessPathFindings(f, reCursorPaths) },
}

// T012 — reads of the Cursor credential store. `state.vscdb`, `cursorAuth`,
// and `ItemTable` have no legitimate use in a skill → Blocker anywhere.
// Bare `accessToken` is generic OAuth vocabulary — it only counts with
// Cursor context in the same file.
var ruleT012 = rule{
	id: "SK-T012", sev: SeverityBlocker, quality: "security", effort: 10,
	msg:   "references the Cursor credential store (state.vscdb / cursorAuth / accessToken)",
	files: nil,
	scan: func(f *FileContent) []string {
		out := lineMatches(f.Text, reCursorCreds)
		if reCursorCtx.MatchString(f.Text) {
			out = append(out, lineMatches(f.Text, reAccessToken)...)
		}
		return out
	},
}

// T013 — allowed-tools wildcard, or absent on a bundle that ships
// executables. allowed-tools grants ambient tool access; a wildcard grant or
// no declared boundary on an executable bundle is the EA1 floor.
var ruleT013 = rule{
	id: "SK-T013", sev: SeverityBlocker, quality: "security", effort: 10,
	msg: "tool boundary: wildcard allowed-tools, or executable bundle with none declared",
	scanBundle: func(l *Ledger) []Finding {
		var out []Finding
		var execPaths []string
		for _, f := range l.Files {
			if f.Entry.Outcome == "inspected" && isScript(f.Entry.Path) {
				execPaths = append(execPaths, f.Entry.Path)
			}
		}
		for _, sf := range skillFiles(l) {
			fm := ParseFrontmatter(sf.Text)
			at, ok := fm.Keys["allowed-tools"]
			if ok {
				for _, item := range fm.Lists["allowed-tools"] {
					if isWildcardGrant(item) {
						out = append(out, Finding{RuleID: "SK-T013", Severity: SeverityBlocker,
							Quality: "security", Message: "allowed-tools wildcard grant",
							File: sf.Entry.Path, Evidence: "allowed-tools: " + item,
							EffortMinutes: 10, Source: "skillgate"})
					}
				}
				if isWildcardGrant(at) {
					out = append(out, Finding{RuleID: "SK-T013", Severity: SeverityBlocker,
						Quality: "security", Message: "allowed-tools wildcard grant",
						File: sf.Entry.Path, Evidence: "allowed-tools: " + at,
						EffortMinutes: 10, Source: "skillgate"})
				}
				continue
			}
			// Executable scope is the skill's own directory tree, not the
			// whole package — one script deep in plugins/x/ must not flag
			// every doc-only SKILL.md in a monorepo (dogfood: a single
			// recon.mjs flagged 9 clean skills).
			if e := execUnder(execPaths, skillScope(sf.Entry.Path)); e != "" {
				out = append(out, Finding{RuleID: "SK-T013", Severity: SeverityBlocker,
					Quality: "security", Message: "executable bundle declares no allowed-tools boundary",
					File: sf.Entry.Path, Evidence: "skill ships " + e + " but SKILL.md has no allowed-tools",
					EffortMinutes: 10, Source: "skillgate"})
			}
		}
		return out
	},
}

func isWildcardGrant(s string) bool {
	s = strings.TrimSpace(s)
	return s == "*" || strings.Contains(s, "(*)") || s == "all"
}

// skillScope returns the directory a SKILL.md governs — "" when the file is
// at the package root (its scope is the whole bundle).
func skillScope(skillPath string) string {
	if i := strings.LastIndex(skillPath, "/"); i >= 0 {
		return skillPath[:i]
	}
	return ""
}

// execUnder returns the first script-class path under scope, or "" when the
// skill ships no executables.
func execUnder(execPaths []string, scope string) string {
	for _, e := range execPaths {
		if scope == "" || strings.HasPrefix(e, scope+"/") {
			return e
		}
	}
	return ""
}

// T014 — declared tool set greatly exceeds what the bundle invokes (Medium
// warn, not a blocker): over-declared agency without corresponding use.
var ruleT014 = rule{
	id: "SK-T014", sev: SeverityMedium, quality: "security", effort: 10,
	msg: "allowed-tools declares far more tools than the bundle invokes",
	scanBundle: func(l *Ledger) []Finding {
		var out []Finding
		var allText strings.Builder
		for _, f := range l.Files {
			if f.Entry.Outcome == "inspected" {
				// Strip frontmatter: the declaration itself must not count as
				// an invocation of the tools it names.
				allText.WriteString(Body(f.Text))
				allText.WriteByte('\n')
			}
		}
		bundle := allText.String()
		for _, sf := range skillFiles(l) {
			fm := ParseFrontmatter(sf.Text)
			declared := fm.Lists["allowed-tools"]
			if v := fm.Keys["allowed-tools"]; v != "" && !strings.HasPrefix(v, "[") {
				for _, item := range strings.Split(v, ",") {
					if s := strings.TrimSpace(item); s != "" {
						declared = append(declared, s)
					}
				}
			}
			if len(declared) < 3 {
				continue
			}
			invoked := 0
			for _, d := range declared {
				// Tool name is the token before any '(' arg spec.
				name := d
				if i := strings.Index(name, "("); i >= 0 {
					name = name[:i]
				}
				if name != "" && strings.Contains(bundle, name) {
					invoked++
				}
			}
			if invoked*2 < len(declared) {
				out = append(out, Finding{RuleID: "SK-T014", Severity: SeverityMedium,
					Quality: "security", Message: "allowed-tools declares far more tools than the bundle invokes",
					File: sf.Entry.Path, Evidence: "declared " + strings.Join(declared, ", "),
					EffortMinutes: 10, Source: "skillgate"})
			}
		}
		return out
	},
}

// T015 — MCP config carries a plaintext secret or a wildcard HTTP binding.
var ruleT015 = rule{
	id: "SK-T015", sev: SeverityBlocker, quality: "security", effort: 15,
	msg:   "MCP config carries a plaintext secret or wildcard HTTP binding",
	files: isHarnessConfig,
	scan: func(f *FileContent) []string {
		if !strings.Contains(f.Entry.Path, "mcp") {
			return nil
		}
		var out []string
		out = append(out, lineMatches(f.Text, reWildcardBind)...)
		var doc map[string]any
		if json.Unmarshal([]byte(f.Text), &doc) != nil {
			return out
		}
		servers, _ := doc["mcpServers"].(map[string]any)
		if servers == nil {
			servers, _ = doc["servers"].(map[string]any)
		}
		for _, name := range sortedKeys(servers) {
			env, _ := servers[name].(map[string]any)["env"].(map[string]any)
			for _, k := range sortedKeys(env) {
				v := env[k]
				val, _ := v.(string)
				if isSecretKey(k) && isPlaintextLiteral(val) {
					out = append(out, "mcp server "+name+": env "+k+" is a plaintext literal")
				}
			}
		}
		return out
	},
}

func isSecretKey(k string) bool {
	u := strings.ToUpper(k)
	return strings.Contains(u, "KEY") || strings.Contains(u, "TOKEN") ||
		strings.Contains(u, "SECRET") || strings.Contains(u, "PASSWORD")
}

func isPlaintextLiteral(v string) bool {
	v = strings.TrimSpace(v)
	return v != "" && !strings.HasPrefix(v, "$") && !strings.HasPrefix(v, "env:")
}

// T016 — MCP tool invocable without consent (auto-approve / alwaysAllow).
var ruleT016 = rule{
	id: "SK-T016", sev: SeverityBlocker, quality: "security", effort: 10,
	msg:   "MCP tool auto-approved without user consent",
	files: isHarnessConfig,
	scan:  func(f *FileContent) []string { return lineMatches(f.Text, reAutoApprove) },
}

// T017/T018 — a hook config inside the bundle that executes bundled content.
// The config and the payload travel together; the skill self-installs an
// executor. Any hook command that is not an absolute system path is treated
// as bundled (relative, ./, $VAR-prefixed, or bare names that resolve in the
// bundle).
func hookExecFindings(ruleID, file, text string) []Finding {
	var out []Finding
	var doc map[string]any
	if json.Unmarshal([]byte(text), &doc) != nil {
		// Unparseable hook config still trips if it mentions a hook command.
		for _, line := range lineMatches(text, regexp.MustCompile(`(?i)"command"\s*:`)) {
			out = append(out, Finding{RuleID: ruleID, Severity: SeverityBlocker,
				Quality: "security", Message: "hook config executes bundled content",
				File: file, Evidence: line, EffortMinutes: 15, Source: "skillgate"})
		}
		return out
	}
	for _, m := range reHookCmdValue.FindAllStringSubmatch(text, -1) {
		if isBundledCommand(m[1]) {
			out = append(out, Finding{RuleID: ruleID, Severity: SeverityBlocker,
				Quality: "security", Message: "hook config executes bundled content",
				File: file, Evidence: "command: " + m[1], EffortMinutes: 15, Source: "skillgate"})
		}
	}
	return out
}

var reHookCmdValue = regexp.MustCompile(`"command"\s*:\s*"([^"]+)"`)

// isBundledCommand reports whether a hook command can resolve to bundled
// content: a relative path, a $VAR expansion, a script extension, or a bare
// command name (PATH-resolvable into the bundle). Fully absolute command
// lines are outside the bundle and do not fire.
func isBundledCommand(cmd string) bool {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return false
	}
	for _, tok := range fields {
		tok = strings.Trim(tok, `'"`)
		if strings.HasPrefix(tok, "$") || strings.HasPrefix(tok, ".") {
			return true
		}
		// A non-absolute token containing a slash is a relative path.
		if strings.Contains(tok, "/") && !strings.HasPrefix(tok, "/") {
			return true
		}
	}
	// Bare command name (no slash) — PATH resolution can land in the bundle.
	if !strings.Contains(fields[0], "/") {
		return true
	}
	return false
}

var ruleT017 = rule{
	id: "SK-T017", sev: SeverityBlocker, quality: "security", effort: 15,
	msg:   "hook config or package manifest in bundle registers bundled executable content",
	files: nil,
	scanBundle: func(l *Ledger) []Finding {
		var out []Finding
		for _, f := range l.Files {
			p := f.Entry.Path
			if f.Entry.Outcome != "inspected" {
				continue
			}
			// Hook-config legs apply to JSON only — a `.claude/hooks/CONFIG.md`
			// doc containing example `"command"` payloads is prose, not a
			// registration. `.codex/` is in scope: Codex hooks configs are an
			// equivalent exec-registration surface (dogfood: real TP on
			// `.codex/hooks.json` was missed).
			if isHarnessConfig(p) &&
				(strings.Contains(f.Text, "hooks") || strings.Contains(f.Text, `"command"`)) {
				out = append(out, hookExecFindings("SK-T017", p, f.Text)...)
			}
			// Convention-discovered executables: pi loads .pi/extensions/
			// without a manifest entry, and a top-level extensions/ dir of
			// scripts is the same shape — bundled code a loader will run
			// whether or not a manifest names it.
			if strings.HasPrefix(p, ".pi/extensions/") ||
				(strings.HasPrefix(p, "extensions/") && isScript(p)) {
				out = append(out, Finding{RuleID: "SK-T017", Severity: SeverityBlocker,
					Quality: "security", Message: "bundle ships convention-discovered extension executable",
					File: p, Evidence: "extension content: " + p,
					EffortMinutes: 15, Source: "skillgate"})
			}
			// [pi] extension leg — same rule, not a 21st: a package
			// manifest naming bundled executables. pi's `package.json#pi`
			// `extensions` entries are in-process TypeScript run with full
			// privileges and resolved provider auth, never type-checked by
			// the loader; npm lifecycle scripts are the install-time
			// supply-chain leg (the package manager runs them on install).
			if isManifestFile(p) {
				out = append(out, manifestExecFindings(p, f.Text)...)
			}
		}
		return out
	},
}

// manifestExecFindings fires for each manifest-declared executable that
// resolves inside the bundle: `pi.extensions`/`extensions` entries that are
// local paths, and package.json lifecycle scripts the package manager runs
// automatically at install time.
func manifestExecFindings(file, text string) []Finding {
	var doc map[string]any
	if json.Unmarshal([]byte(text), &doc) != nil {
		return nil
	}
	var out []Finding
	fire := func(evidence string) {
		out = append(out, Finding{RuleID: "SK-T017", Severity: SeverityBlocker,
			Quality: "security", Message: "package manifest registers bundled executable content",
			File: file, Evidence: evidence, EffortMinutes: 15, Source: "skillgate"})
	}
	var extEntries []string
	if pi, ok := doc["pi"].(map[string]any); ok {
		extEntries = append(extEntries, manifestStringList(pi["extensions"])...)
	}
	extEntries = append(extEntries, manifestStringList(doc["extensions"])...)
	for _, e := range extEntries {
		if isBundledManifestEntry(e) {
			fire("extension: " + e)
		}
	}
	if scripts, ok := doc["scripts"].(map[string]any); ok {
		for _, k := range []string{"preinstall", "install", "postinstall", "prepare"} {
			if s, ok := scripts[k].(string); ok && s != "" {
				fire("lifecycle " + k + ": " + s)
			}
		}
	}
	// `bin` entries are install-time executables on PATH — the same
	// registration surface as extensions.
	for _, e := range manifestStringList(doc["bin"]) {
		if isBundledManifestEntry(e) {
			fire("bin: " + e)
		}
	}
	if m, ok := doc["bin"].(map[string]any); ok {
		for _, name := range sortedKeys(m) {
			if s, ok := m[name].(string); ok && isBundledManifestEntry(s) {
				fire("bin " + name + ": " + s)
			}
		}
	}
	return out
}

// sortedKeys returns m's keys in lexical order — the only order a report
// may ever emit. Map iteration order is nondeterministic and would shuffle
// findings (and their fingerprints' report position) between runs.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func manifestStringList(v any) []string {
	var out []string
	switch t := v.(type) {
	case []any:
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
	case string:
		out = append(out, t)
	}
	return out
}

// isBundledManifestEntry reports whether an extension entry names content
// inside the bundle rather than a registry spec: local paths (./, /, or a
// relative path) and script filenames are bundled; `npm:`, `git:`, URLs,
// and `name@ver`/`@scope/name@ver` specs are external.
func isBundledManifestEntry(entry string) bool {
	e := strings.TrimSpace(entry)
	if e == "" {
		return false
	}
	for _, p := range []string{"http://", "https://", "npm:", "git:", "github:", "file://"} {
		if strings.HasPrefix(e, p) {
			return false
		}
	}
	if isScript(e) || strings.HasPrefix(e, ".") || strings.HasPrefix(e, "/") {
		return true
	}
	return strings.Contains(e, "/") && !strings.Contains(e, "@")
}

var ruleT018 = rule{
	id: "SK-T018", sev: SeverityBlocker, quality: "security", effort: 15,
	msg:   "Cursor hook config in bundle executes bundled content",
	files: nil,
	scanBundle: func(l *Ledger) []Finding {
		var out []Finding
		for _, f := range l.Files {
			p := f.Entry.Path
			if f.Entry.Outcome != "inspected" {
				continue
			}
			if p == ".cursor/hooks.json" || strings.HasSuffix(p, ".cursor/hooks.json") {
				out = append(out, hookExecFindings("SK-T018", p, f.Text)...)
			}
		}
		return out
	},
}

// T020 — writes into harness config, shell rc, or OS persistence surfaces.
var ruleT020 = rule{
	id: "SK-T020", sev: SeverityBlocker, quality: "security", effort: 20,
	msg:   "writes to a persistence or self-modification surface (shell rc, .claude/.cursor config, crontab, LaunchAgents)",
	files: isScript,
	scan: func(f *FileContent) []string {
		var out []string
		// Normalize Windows separators (.cursor\hooks.json evasion) and mask
		// /dev/null — `2>/dev/null` discards output; it isn't persistence.
		text := normPathSep(strings.ReplaceAll(f.Text, "/dev/null", "DEVNULL"))
		for _, line := range strings.Split(text, "\n") {
			ploc := rePersistPath.FindStringIndex(line)
			if ploc == nil {
				continue
			}
			verb := false
			if vloc := rePersistVerb.FindStringIndex(line); vloc != nil {
				verb = true
				if reProgWrite.MatchString(line[vloc[0]:vloc[1]]) {
					// The path must be the call's target (first arg) —
					// writeFileSync(x, ".pi/…") is content, not persistence.
					verb = false
					if open := strings.Index(line[vloc[0]:], "("); open >= 0 {
						argStart := vloc[0] + open + 1
						argEnd := len(line)
						if comma := strings.Index(line[argStart:], ","); comma >= 0 {
							argEnd = argStart + comma
						}
						verb = ploc[0] >= argStart && ploc[0] < argEnd
					}
				}
			}
			if verb || strings.Contains(line, "crontab") || strings.Contains(line, "launchctl") ||
				strings.Contains(line, "systemctl") || strings.Contains(line, "defaults") {
				out = append(out, strings.TrimSpace(line))
			}
		}
		return out
	},
}
