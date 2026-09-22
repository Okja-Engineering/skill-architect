package skillgate

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Pack B rules T001–T009: injection, exfiltration, supply chain, obfuscation.

var (
	// T002 — instruction-override phrasing. Tight enough to avoid firing on
	// prose that *discusses* injection (our own docs say "prompt injection"
	// freely); each pattern names the act, not the topic.
	reOverride = []*regexp.Regexp{
		regexp.MustCompile(`(?i)ignore\s+(all\s+|any\s+|the\s+)?(previous|prior|above|preceding|earlier)\s+(instructions|prompts|rules|directives|context)`),
		regexp.MustCompile(`(?i)disregard\s+(all\s+|the\s+)?(previous|prior|above|earlier)\s+(instructions|prompts|rules|directives)`),
		regexp.MustCompile(`(?i)forget\s+(your|all|everything|anything)\s+(you\s+(know|were|learned)|previous|prior)`),
		regexp.MustCompile(`(?i)(new|real|actual|true)\s+instructions?\s*:`),
		regexp.MustCompile(`(?i)you\s+are\s+now\s+(a|an|in)\b`),
		regexp.MustCompile(`(?i)(do\s+not|don't|never)\s+(follow|obey|trust)\s+(your|the|any)\s+(original|previous|system|prior)\b`),
		regexp.MustCompile(`(?i)override\s+(your|the|all|any)\s+(safety|security|guidelines|rules|instructions|restrictions)`),
		// "system prompt:" alone is normal doc prose — require the smuggled
		// channel to carry instruction-like content after the colon.
		regexp.MustCompile(`(?i)(system|hidden)\s+(prompt|message|instruction)\s*:\s*(you|ignore|never|always|do not|forget|reveal)`),
		regexp.MustCompile(`(?i)reveal\s+(your|the)\s+(system|initial|original)\s+(prompt|instructions)`),
		regexp.MustCompile(`(?i)important:\s*(ignore|override|forget|disregard)`),
	}

	// T004 — transmit to a literal remote host.
	reNetCmd = regexp.MustCompile(`(?i)\b(curl|wget|nc|ncat|socat|ftp|scp|rsync|fetch|Invoke-WebRequest|iwr)\b[^\n]*?(https?|ftp|tcp)://[^\s"'\\)]+`)
	reDevTCP = regexp.MustCompile(`/dev/(tcp|udp)/`)
	rePyNet  = regexp.MustCompile(`(?i)\b(requests\.(get|post|put)|urllib\.request|httpx\.|fetch\()\s*\(?[^)\n]*https?://`)
	// Loopback URLs are dev/test plumbing (ollama, local health checks), not
	// a transmission to a remote host — masked before T004's net legs run.
	reLoopbackURL = regexp.MustCompile(`(?i)https?://(localhost|127\.[0-9.]+|0\.0\.0\.0|\[?::1\]?)(:[0-9]+)?`)

	// T005 — env harvest patterns and their sinks. The dump leg is
	// *wholesale* enumeration only — `os.environ` iterated/copied, a
	// `process.env` spread/serialize, printenv — not a single-var read like
	// `process.env.HOME`, which is ordinary code.
	reEnvDump = regexp.MustCompile(`(?i)(\bprintenv\b|\benv\s*\|\s|^\s*env\s*$|\bset\s*\|\s|os\.environ(\s*[,)}\]\|]|\.(items|copy|keys|values)\s*\()|=\s*os\.environ\b|\.\.\.process\.env|JSON\.stringify\(process\.env|compgen\s+-e|Get-ChildItem\s+Env)`)
	// envAssign matches `env:`/`env =`/`envBackup =` — a process.env spread
	// bound to an env-named option or variable is child-env plumbing, not a
	// payload. Only fires to suppress when the dump IS the env value.
	reEnvAssign = regexp.MustCompile(`(?i)\b\w*env\w*\s*[:=]`)
	reSink      = regexp.MustCompile(`(?i)(curl|wget|nc\b|ncat|socat|/dev/tcp|requests\.post|urllib|fetch\(|https?://|>\s*[/~]|\|\s*(base64|sh|bash)|scp\b|rsync\b)`)

	// T006 — credential-shaped literals. Evidence is masked before it lands
	// in a finding — the gate never re-publishes a secret.
	reCreds = []*regexp.Regexp{
		regexp.MustCompile(`AKIA[0-9A-Z]{16}`),                                                // AWS access key
		regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`),                              // PEM key
		regexp.MustCompile(`\b(sk|pk|rk)_(live|test)_[A-Za-z0-9]{16,}`),                       // Stripe-style
		regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{20,}`),                                     // Anthropic
		regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{20,}`),                                         // OpenAI-style
		regexp.MustCompile(`\b(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9]{20,}`),                        // GitHub tokens
		regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{22,}`),                                  // GitHub fine-grained
		regexp.MustCompile(`\bxox[bapors]-[A-Za-z0-9-]{10,}`),                                 // Slack
		regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}`),                                         // Google API
		regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`), // JWT
		regexp.MustCompile(`(?i)(api[_-]?key|secret|token|passwd|password)\s*[:=]\s*["'][A-Za-z0-9_./+=-]{16,}["']`),
	}

	// T007 — network output piped to a shell or interpreter.
	// The second alternative (interpreter then URL) must not cross a `|` —
	// otherwise a markdown table row like `| Python | https://…` fires.
	rePipeToShell  = regexp.MustCompile(`(?i)(curl|wget|fetch|Invoke-WebRequest|iwr)\b[^\n|]*\|\s*(sudo\s+)?(ba|z|fi|da)?sh\b|\|\s*(sudo\s+)?(python[0-9.]*|perl|ruby|node)\b[^\n|]*https?://`)
	rePipeToShell2 = regexp.MustCompile(`(?i)(curl|wget)[^\n|]*\|\s*(sudo\s+)?(python[0-9.]*|perl|ruby|node)\b`)
	reIEX          = regexp.MustCompile(`(?i)\biex\s*\(.*(iwr|Invoke-WebRequest|wget|curl)`)

	// T008 — remote fetch / install without a pinned ref. The command must
	// start the line (modulo whitespace/sudo) so install *advice inside echo
	// strings or prose* is not flagged.
	reUnpinned = []*regexp.Regexp{
		regexp.MustCompile(`(?im)^\s*(sudo\s+)?(curl|wget)\s+[^\n|]*https?://[^\s"']*/(main|master|HEAD|latest|trunk)/`),
		regexp.MustCompile(`(?im)^\s*(sudo\s+)?npm\s+(i|install)\s+(-g\s+)?[a-z0-9@][^\s@]*\s*$`),
		regexp.MustCompile(`(?im)^\s*(sudo\s+)?npm\s+(i|install)\s+(-g\s+)?[a-z0-9][^\s@=]*\s+`),
		regexp.MustCompile(`(?im)^\s*(sudo\s+)?pip[0-9.]*\s+install\s+[a-z0-9_-]+\s*$`),
		regexp.MustCompile(`(?im)^\s*go\s+install\s+\S+@latest`),
		regexp.MustCompile(`(?im)^\s*gem\s+install\s+\S+\s*$`),
	}

	// T009 — decode-then-execute. The hex leg needs 4+ consecutive \xNN —
	// two-escape runs are ANSI/regex charset classes (`[\x07\x1b]`), a real
	// payload is a long run. `eval`/`exec` must not be dot-prefixed method
	// calls (`re.exec(str)` is a regex match, not an interpreter); qualified
	// exec sinks (child_process.exec, execSync/execFile) stay.
	reDecode = regexp.MustCompile(`(?i)(base64\s+(-d|--decode|-D)|b64decode|atob\(|xxd\s+-r|hex\.decode|fromhex|(\\x[0-9a-fA-F]{2}){4,})`)
	reExec   = regexp.MustCompile(`(?i)(^|[^\w.])(eval|exec)\b|child_process\.exec|\.exec(Sync|File|FileSync)\s*\(|os\.system|subprocess|popen|Invoke-Expression|\biex\s*\(|\|\s*(ba|z|fi)?sh\b`)
)

// lineMatches returns each line matching any of the patterns.
func lineMatches(text string, res ...*regexp.Regexp) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		for _, re := range res {
			if re.MatchString(line) {
				out = append(out, strings.TrimSpace(line))
				break
			}
		}
	}
	return out
}

// T001 — zero-width / bidi / tag characters in loaded text.
var ruleT001 = rule{
	id: "SK-T001", sev: SeverityBlocker, quality: "security", effort: 10,
	msg:   "invisible or bidi control characters in loaded text (hidden-instruction channel)",
	files: isLoadedText,
	scan: func(v *View) []string {
		var out []string
		for i, line := range strings.Split(v.Text, "\n") {
			for _, r := range line {
				bad := (r >= 0x200B && r <= 0x200F) || // ZWSP…RLM
					(r >= 0x202A && r <= 0x202E) || // bidi embeds/overrides
					(r >= 0x2060 && r <= 0x2064) || // word joiner, invisible ops
					(r >= 0x2066 && r <= 0x2069) || // bidi isolates
					(r >= 0xE0000 && r <= 0xE007F) || // tag characters (steganographic injection)
					r == 0x00AD || r == 0x115F || r == 0x1160 || r == 0xFFA0 ||
					(r == 0xFEFF && i+1 > 0) // BOM mid-file
				if bad {
					out = append(out, fmt.Sprintf("line %d: contains U+%04X", i+1, r))
					break
				}
			}
		}
		return out
	},
}

// T002 — instruction-override phrasing in description or body.
var ruleT002 = rule{
	id: "SK-T002", sev: SeverityBlocker, quality: "security", effort: 15,
	msg:   "instruction-override phrasing in skill text",
	files: isLoadedText,
	scan:  func(v *View) []string { return lineMatches(v.Text, reOverride...) },
}

// T003 — homoglyph / mixed-script in description, name, or MCP tool name.
var ruleT003 = rule{
	id: "SK-T003", sev: SeverityBlocker, quality: "security", effort: 20,
	msg:   "confusable non-Latin characters mixed into a name or description",
	files: isLoadedText,
	scan: func(v *View) []string {
		fm := ParseFrontmatter(v.Text)
		var out []string
		for _, key := range []string{"name", "description", "when_to_use"} {
			if v := fm.Keys[key]; v != "" {
				if tok := mixedScriptToken(v); tok != "" {
					out = append(out, key+": "+tok)
				}
			}
		}
		return out
	},
}

// mixedScriptToken returns the first token mixing Latin with a confusable
// script (Cyrillic, Greek, Armenian, Cherokee), or "".
func mixedScriptToken(s string) string {
	for _, tok := range strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || r == ',' || r == ';'
	}) {
		latin, confus := false, false
		for _, r := range tok {
			switch {
			case r < 0x80:
				if unicode.IsLetter(r) {
					latin = true
				}
			case (r >= 0x0370 && r <= 0x03FF) || (r >= 0x0400 && r <= 0x04FF) ||
				(r >= 0x0530 && r <= 0x058F) || (r >= 0x13A0 && r <= 0x13FF):
				confus = true
			}
		}
		if latin && confus {
			return tok
		}
	}
	return ""
}

// T004 — bundled script transmits to a literal remote host.
var ruleT004 = rule{
	id: "SK-T004", sev: SeverityBlocker, quality: "security", effort: 20,
	msg:   "bundled script transmits to a literal remote host",
	files: isScript,
	scan: func(v *View) []string {
		return lineMatches(reLoopbackURL.ReplaceAllString(v.Text, "LOOPBACK"), reNetCmd, reDevTCP, rePyNet)
	},
}

// T005 — env harvest reaching a sink (same file).
var ruleT005 = rule{
	id: "SK-T005", sev: SeverityBlocker, quality: "security", effort: 20,
	msg:   "environment dump reaches a network or file sink",
	files: isScript,
	// File-level correlation was the dominant dogfood FP (272/565 on the pi
	// monorepo): an env read *anywhere* plus any URL/sink token *anywhere*
	// fired. Require proximity — a wholesale env dump within 10 lines of a
	// sink is the exfil shape; a distant co-occurrence is ordinary code.
	scan: func(v *View) []string {
		lines := strings.Split(v.Text, "\n")
		var dumps, sinks []int
		for i, line := range lines {
			if reEnvDump.MatchString(line) {
				dumps = append(dumps, i)
			}
			if reSink.MatchString(line) {
				sinks = append(sinks, i)
			}
		}
		for _, d := range dumps {
			// A dump bound to an env-named option/var is plumbing — check
			// the dump line and the line above (multiline `env: {` objects).
			if strings.Contains(lines[d], "process.env") &&
				(reEnvAssign.MatchString(lines[d]) ||
					(d > 0 && reEnvAssign.MatchString(lines[d-1]) && strings.Contains(lines[d-1], "{"))) {
				continue
			}
			for _, s := range sinks {
				if d == s || absInt(d-s) <= 10 {
					return []string{fmt.Sprintf("line %d env dump: %s | line %d sink: %s",
						d+1, truncate(strings.TrimSpace(lines[d]), 60),
						s+1, truncate(strings.TrimSpace(lines[s]), 60))}
				}
			}
		}
		return nil
	},
}

// T006 — credential-shaped literal in a bundled file. Evidence is masked.
var ruleT006 = rule{
	id: "SK-T006", sev: SeverityBlocker, quality: "security", effort: 15,
	msg: "credential-shaped literal in a bundled file",
	scan: func(v *View) []string {
		var out []string
		for _, line := range strings.Split(v.Text, "\n") {
			for _, re := range reCreds {
				if re.MatchString(line) {
					masked := re.ReplaceAllString(strings.TrimSpace(line), "***")
					out = append(out, masked)
					break
				}
			}
		}
		return out
	},
}

// T007 — network output piped to a shell.
var ruleT007 = rule{
	id: "SK-T007", sev: SeverityBlocker, quality: "security", effort: 30,
	msg: "network output piped to a shell or interpreter",
	scan: func(v *View) []string {
		return lineMatches(v.Text, rePipeToShell, rePipeToShell2, reIEX)
	},
}

// T008 — remote fetch / install without a pinned ref.
var ruleT008 = rule{
	id: "SK-T008", sev: SeverityBlocker, quality: "security", effort: 10,
	msg:   "remote fetch or install without a pinned version",
	files: isScript,
	scan: func(v *View) []string {
		return lineMatches(v.Text, reUnpinned...)
	},
}

// T009 — obfuscation: decode construct feeding execution in the same file.
var ruleT009 = rule{
	id: "SK-T009", sev: SeverityHigh, quality: "security", effort: 25,
	msg:   "encoded payload feeding an interpreter (decode-then-execute)",
	files: isScript,
	// File-level co-occurrence was the FP amplifier (JWT `atob` + `re.exec`
	// anywhere in an auth file fired). Same-line, or decode within 10 lines
	// of exec — the decode-then-execute payload shape.
	scan: func(v *View) []string {
		lines := strings.Split(v.Text, "\n")
		var dec, ex []int
		for i, line := range lines {
			if reDecode.MatchString(line) {
				dec = append(dec, i)
			}
			if reExec.MatchString(line) {
				ex = append(ex, i)
			}
		}
		for _, d := range dec {
			for _, e := range ex {
				if absInt(d-e) <= 10 {
					return []string{fmt.Sprintf("line %d decode near line %d exec: %s",
						d+1, e+1, truncate(strings.TrimSpace(lines[d]), 90))}
				}
			}
		}
		return nil
	},
}
