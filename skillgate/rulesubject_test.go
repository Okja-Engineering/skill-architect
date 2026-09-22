package skillgate

import (
	"strings"
	"testing"
)

// Rule-subject tests — the negative denominator.
//
// `firesTable` proves each rule fires on a fixture that carries its subject.
// It cannot prove the rule is *about* that subject: a rule that matched the
// empty string would pass every row. These tables are the other half — a
// fixture that carries the rule's *vocabulary* and not its subject, which
// the rule must therefore leave alone, and a fixture that carries the
// subject in a form the narrowing might have dropped, which it must still
// catch.
//
// Both halves are required for every rule this slice touches, because a fix
// proved only by what it silences is not proved.

// quietTable: the rule's words are present, its subject is not.
var quietTable = []struct {
	rule  string
	why   string
	files map[string]string
}{
	// SK-T010/T011 — the release's own B2. `draft-rewrite.sh` documents the
	// five home directories it refuses to write into; naming the bound in a
	// comment is the opposite of crossing it, and a rule that cannot tell
	// them apart flags the code that does the protecting.
	{"SK-T010", "whole-line shell comment naming a harness dir", map[string]string{
		"scripts/r.sh": "#!/bin/sh\n" +
			"# An agent reads `~/.claude/skills/` as skills, so a draft dropped\n" +
			"# there is loaded as instructions. Refuse $HOME/.claude outright.\n" +
			"protected=\".claude .cursor .codex\"\n" +
			"echo \"$protected\"\n",
	}},
	{"SK-T011", "whole-line shell comment naming the Cursor config dir", map[string]string{
		"scripts/r.sh": "#!/bin/sh\n# never touch ~/.cursor/mcp.json\necho ok\n",
	}},
	{"SK-T010", "whole-line // comment in a TypeScript bundle script", map[string]string{
		"scripts/r.ts": "// the installer must never read ~/.claude/settings.json\nexport const x = 1;\n",
	}},
	{"SK-T010", "python docstring-free comment naming the dir it avoids", map[string]string{
		"scripts/r.py": "# refuses to open $HOME/.codex/config.toml\nprint('ok')\n",
	}},

	// SK-T019 — `..` is a path *segment*. A sentence that quotes it is not a
	// reference to anywhere, and this rule is a blocker: the spec row S09
	// wrote to explain why T019 is raw-only contains a literal `..` and was
	// flagged by T019 itself.
	{"SK-T019", "prose quoting the two-dot segment, no path around it", map[string]string{
		"SKILL.md": "---\nname: x\ndescription: x\n---\n" +
			"NFKC maps U+2025 TWO DOT LEADER onto `..`, so a rule whose subject is\n" +
			"path syntax fires on ordinary prose. There is no `.` left to make a `..`\n" +
			"out of.\n",
	}},
}

// stillFiresTable: the subject is present in a form the narrowing could
// plausibly have dropped. Every row here is a true positive and must survive.
var stillFiresTable = []struct {
	rule  string
	why   string
	files map[string]string
}{
	// The comment marker only ends a line's code when it *starts* the line.
	// `echo "#"` does not comment out what follows it, and a rule that
	// stopped at the first `#` anywhere would be evadable in one character.
	{"SK-T010", "a `#` inside a string does not make the rest of the line a comment", map[string]string{
		"scripts/r.sh": "#!/bin/sh\necho \"#\" ; cat ~/.claude/settings.json\n",
	}},
	{"SK-T010", "live code with a trailing comment still touches the dir", map[string]string{
		"scripts/r.sh": "#!/bin/sh\ncat ~/.claude/settings.json  # slurp it\n",
	}},
	{"SK-T010", "indented live code inside a function", map[string]string{
		"scripts/r.sh": "#!/bin/sh\nf() {\n    cat \"$HOME/.claude/settings.json\"\n}\nf\n",
	}},
	{"SK-T010", "an extensionless bundled script keeps full reach", map[string]string{
		"bin/tool": "#!/bin/sh\n# cat ~/.claude/settings.json\n",
	}},
	{"SK-T011", "live read of the Cursor MCP config", map[string]string{
		"scripts/r.sh": "#!/bin/sh\ncat ~/.cursor/mcp.json\n",
	}},

	// T019's true positives all carry a separator — except a link whose
	// whole target is the parent, which is a reference by construction.
	{"SK-T019", "markdown link whose target is the bundle parent", map[string]string{
		"SKILL.md": "---\nname: x\ndescription: x\n---\nSee [the parent](..) for more.\n",
	}},
	{"SK-T019", "a climb spelled with separators in prose", map[string]string{
		"SKILL.md": "---\nname: x\ndescription: x\n---\nRead ../../etc/passwd\n",
	}},
	{"SK-T019", "a climb inside a bundled script", map[string]string{
		"scripts/r.sh": "#!/bin/sh\ncat ../../../etc/shadow\n",
	}},
	{"SK-T019", "backslash-spelled climb is still a climb", map[string]string{
		"SKILL.md": "---\nname: x\ndescription: x\n---\nRead ..\\..\\etc\\passwd\n",
	}},
}

func ruleFired(t *testing.T, files map[string]string, rule string) bool {
	t.Helper()
	root := writeBundle(t, files)
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range rep.Findings {
		if f.RuleID == rule {
			return true
		}
	}
	return false
}

// TestRuleDoesNotFireOnItsOwnVocabulary pins B2's invariant: a rule fires on
// its subject, never on a document that merely names it.
func TestRuleDoesNotFireOnItsOwnVocabulary(t *testing.T) {
	for _, row := range quietTable {
		if ruleFired(t, row.files, row.rule) {
			t.Errorf("%s fired on a fixture that carries no subject: %s", row.rule, row.why)
		}
	}
}

// TestNarrowedRulesStillFire is the other direction, and it is the one that
// matters: under-firing a security rule is worse than over-firing it, and
// far harder to notice. Every row is a real positive.
func TestNarrowedRulesStillFire(t *testing.T) {
	for _, row := range stillFiresTable {
		if !ruleFired(t, row.files, row.rule) {
			t.Errorf("%s stopped firing on a true positive: %s", row.rule, row.why)
		}
	}
}

// TestUnreadableFrontmatterIsReportedAsItself pins the second half of the
// slice: a document the reader refused must be reported as unparseable, at
// the line and with the reason the reader recorded — never as a document
// whose keys are missing. A user told their keys are missing goes and adds
// keys that are already there.
func TestUnreadableFrontmatterIsReportedAsItself(t *testing.T) {
	// A tab in the indentation: YAML forbids it, the reader refuses the
	// document, and `name` and `description` are plainly written below.
	files := map[string]string{
		"SKILL.md": "---\nname: demo\ndescription: a demo skill\nmetadata:\n\tversion: \"1\"\n---\nBody.\n",
	}
	root := writeBundle(t, files)
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	fm := ParseFrontmatter(files["SKILL.md"])
	if len(fm.Unreadable) == 0 {
		t.Fatal("fixture no longer produces a refusal; pick another unreadable construct")
	}
	var unreadable, missing *Finding
	for i := range rep.Findings {
		switch rep.Findings[i].RuleID {
		case "SK-I006":
			unreadable = &rep.Findings[i]
		case "SK-I001":
			missing = &rep.Findings[i]
		}
	}
	if unreadable == nil {
		t.Fatal("a refused frontmatter must produce SK-I006")
	}
	if missing != nil {
		t.Errorf("SK-I001 must not claim keys are missing from a document that was never read: %q", missing.Message)
	}
	if unreadable.Line != fm.Unreadable[0].Line {
		t.Errorf("SK-I006 line = %d, reader recorded %d", unreadable.Line, fm.Unreadable[0].Line)
	}
	if !strings.Contains(unreadable.Message, fm.Unreadable[0].Reason) {
		t.Errorf("SK-I006 message %q drops the reader's reason %q", unreadable.Message, fm.Unreadable[0].Reason)
	}
}

// TestReadableFrontmatterStillReportsMissingKeys is the non-vacuous other
// side: silencing SK-I001 on a refused document must not silence it on a
// document that was read and genuinely lacks the keys.
func TestReadableFrontmatterStillReportsMissingKeys(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"SKILL.md": "---\nlicense: MIT\n---\nBody.\n",
	})
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range rep.Findings {
		if f.RuleID == "SK-I001" && strings.Contains(f.Message, "name") {
			found = true
		}
		if f.RuleID == "SK-I006" {
			t.Errorf("a readable document must not be reported as unparseable: %q", f.Message)
		}
	}
	if !found {
		t.Fatal("SK-I001 must still fire on a readable document with no name")
	}
}

// TestUnreadableFrontmatterDoesNotForgeAToolBoundaryFinding covers the same
// lie at blocker severity: SK-T013's no-boundary leg reads the absence of
// `allowed-tools`, and on a refused document absence is not something the
// reader knows.
func TestUnreadableFrontmatterDoesNotForgeAToolBoundaryFinding(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"SKILL.md":       "---\nname: demo\ndescription: d\nallowed-tools: Read\nmetadata:\n\tversion: \"1\"\n---\nBody.\n",
		"scripts/one.sh": "#!/bin/sh\necho hi\n",
	})
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range rep.Findings {
		if f.RuleID == "SK-T013" && strings.Contains(f.Evidence, "no allowed-tools") {
			t.Errorf("SK-T013 claimed no boundary is declared on a document it could not read: %q", f.Evidence)
		}
	}
}

// TestExecutableBundleWithNoBoundaryStillBlocks is that fix's true positive:
// a readable SKILL.md shipping a script and declaring nothing still blocks.
func TestExecutableBundleWithNoBoundaryStillBlocks(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"SKILL.md":       "---\nname: demo\ndescription: d\n---\nBody.\n",
		"scripts/one.sh": "#!/bin/sh\necho hi\n",
	})
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range rep.Findings {
		if f.RuleID == "SK-T013" && strings.Contains(f.Evidence, "no allowed-tools") {
			return
		}
	}
	t.Fatal("SK-T013 must still block an executable bundle that declares no boundary")
}
