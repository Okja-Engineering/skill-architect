package skillgate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture reads a per-style frontmatter fixture. The fixtures are whole
// SKILL.md files on disk, not one-line strings, so the multi-line paths
// this slice adds are the paths the assertions actually travel.
func fixture(t *testing.T, style string) *Frontmatter {
	t.Helper()
	p := filepath.Join("testdata", "frontmatter", style, "SKILL.md")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return ParseFrontmatter(string(b))
}

// scalarAt resolves a path and requires a readable scalar there, naming the
// state it found instead when there is not one. Absent and unreadable are
// distinct failures, never the same one.
func scalarAt(t *testing.T, fm *Frontmatter, path ...string) string {
	t.Helper()
	n := fm.Lookup(path...)
	v, ok := n.Scalar()
	if !ok {
		t.Errorf("%s: want a readable scalar, got kind=%s reason=%q",
			strings.Join(path, "."), n.Kind(), n.Reason())
		return ""
	}
	return v
}

func wantScalar(t *testing.T, fm *Frontmatter, want string, path ...string) {
	t.Helper()
	if got := scalarAt(t, fm, path...); got != want {
		t.Errorf("%s = %q, want %q", strings.Join(path, "."), got, want)
	}
}

// TestGateReadsItsOwnCarrierDeclaration is the slice's headline: the gate
// could not read the frontmatter of the skill it ships, because `version`
// and the S01 carrier keys live under a nested `metadata:` map and the
// reader skipped nested keys by design.
func TestGateReadsItsOwnCarrierDeclaration(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "skills", "skill-gate", "SKILL.md"))
	if err != nil {
		t.Skipf("own SKILL.md not present: %v", err)
	}
	fm := ParseFrontmatter(string(b))
	wantScalar(t, fm, "0.1.0", "metadata", "version")
	wantScalar(t, fm, "installed-binary", "metadata", "carrier")
	wantScalar(t, fm, "skillgate", "metadata", "carrier-command")
	wantScalar(t, fm, "./skillgate/cmd/skillgate", "metadata", "carrier-source")
}

// TestReadsEveryOwnSkillVersion — all three shipped skills declare
// metadata.version, and none of them was readable before this slice.
func TestReadsEveryOwnSkillVersion(t *testing.T) {
	for _, skill := range []string{"skill-audit", "skill-rewrite", "skill-gate"} {
		b, err := os.ReadFile(filepath.Join("..", "skills", skill, "SKILL.md"))
		if err != nil {
			t.Skipf("%s not present: %v", skill, err)
		}
		fm := ParseFrontmatter(string(b))
		v := scalarAt(t, fm, "metadata", "version")
		if !strings.HasPrefix(v, "0.") {
			t.Errorf("%s metadata.version = %q, want a 0.x version string", skill, v)
		}
		if c := scalarAt(t, fm, "metadata", "carrier"); c == "" {
			t.Errorf("%s metadata.carrier unreadable", skill)
		}
	}
}

// --- one test per scalar style ------------------------------------------

func TestBlockScalarStyle(t *testing.T) {
	fm := fixture(t, "block-scalar")
	wantScalar(t, fm, "Clip chomping keeps one trailing newline.\nA second content line.\n", "description")
	wantScalar(t, fm, "Strip chomping keeps none.\nSecond line.", "summary")
	wantScalar(t, fm, "Keep chomping keeps them all.\n\n", "notes")
	wantScalar(t, fm, "   two-space indicator, so three spaces survive.\n", "indented")
	wantScalar(t, fm, "0.3.0", "metadata", "version")
}

func TestFoldedScalarStyle(t *testing.T) {
	fm := fixture(t, "folded-scalar")
	wantScalar(t, fm, "Folded lines join with a single space.\nA blank line becomes one newline.\n", "description")
	wantScalar(t, fm, "Folded and stripped, so no trailing newline.", "summary")
	wantScalar(t, fm, "folded text\n  indented literal\nmore folded\n", "literal-run")
	wantScalar(t, fm, "0.4.0", "metadata", "version")
}

func TestNestedMapStyle(t *testing.T) {
	fm := fixture(t, "nested-map")
	wantScalar(t, fm, "1.2.3", "metadata", "version")
	wantScalar(t, fm, "installed-binary", "metadata", "carrier")
	wantScalar(t, fm, "in-repo", "metadata", "provenance", "origin")
	wantScalar(t, fm, "imagineux", "metadata", "provenance", "review", "reviewer")
	if k := fm.Lookup("metadata").Kind(); k != KindMapping {
		t.Errorf("metadata kind = %s, want %s", k, KindMapping)
	}
	if got := fm.Lookup("metadata").Keys(); len(got) != 4 {
		t.Errorf("metadata keys = %v, want 4", got)
	}
}

func TestListOfMapsStyle(t *testing.T) {
	fm := fixture(t, "list-of-maps")
	tools := fm.Lookup("tools")
	if k := tools.Kind(); k != KindSequence {
		t.Fatalf("tools kind = %s, want %s", k, KindSequence)
	}
	if n := len(tools.Items()); n != 2 {
		t.Fatalf("tools has %d items, want 2", n)
	}
	wantScalar(t, fm, "Read", "tools", "0", "name")
	wantScalar(t, fm, "repo", "tools", "0", "scope")
	wantScalar(t, fm, "Bash", "tools", "1", "name")
	wantScalar(t, fm, "--no-network", "tools", "1", "args", "0")
	wantScalar(t, fm, "--read-only", "tools", "1", "args", "1")
	wantScalar(t, fm, "a", "nested-sequence", "0", "0")
	wantScalar(t, fm, "b", "nested-sequence", "0", "1")
	wantScalar(t, fm, "c", "nested-sequence", "1", "0")
	wantScalar(t, fm, "2.0.0", "metadata", "version")
}

func TestPerValueLineStyle(t *testing.T) {
	fm := fixture(t, "per-value-lines")
	wantScalar(t, fm,
		"A plain scalar that begins on the line after its key and continues over "+
			"several lines, folding each break into a single space.\n"+
			"A blank line inside it folds to one newline.", "description")
	wantScalar(t, fm, "bash 3.2+", "compatibility")
	wantScalar(t, fm, "starts on the key line and continues below it", "inline-continuation")
	wantScalar(t, fm, "3.1.0", "metadata", "version")
}

func TestFlowStyle(t *testing.T) {
	fm := fixture(t, "flow")
	wantScalar(t, fm, "Read", "allowed-tools", "0")
	wantScalar(t, fm, "Bash", "allowed-tools", "1")
	wantScalar(t, fm, "bash", "matrix", "shell")
	wantScalar(t, fm, "3.2", "matrix", "version")
	wantScalar(t, fm, "Read", "spanning", "0")
	wantScalar(t, fm, "Bash", "spanning", "1")
	wantScalar(t, fm, "Bash", "deep", "tools", "1", "name")
	wantScalar(t, fm, "a value with: a colon, and a # hash", "quoted")
	wantScalar(t, fm, "it's quoted", "single")
	wantScalar(t, fm, "tab\there\nand a newline", "escaped")
	wantScalar(t, fm, "4.0.0", "metadata", "version")
}

// --- refusal: the subset's edge is named, never silent -------------------

// TestRefusalsAreNamedAndVisible pins the other half of the contract: a
// construct outside the declared subset is refused by name, and the key
// that carries it stays visible as unreadable. Silence is the defect this
// slice exists to remove.
func TestRefusalsAreNamedAndVisible(t *testing.T) {
	cases := []struct {
		name   string
		text   string
		path   []string // the path that must read as unreadable
		reason string   // substring the refusal must name
		scope  string   // "entry" or "document"
	}{
		{
			name:   "anchor",
			text:   "---\nname: x\ndescription: &d anchored\n---\nbody\n",
			path:   []string{"description"},
			reason: "anchor",
			scope:  "entry",
		},
		{
			name:   "alias",
			text:   "---\nname: x\ndescription: *d\n---\nbody\n",
			path:   []string{"description"},
			reason: "alias",
			scope:  "entry",
		},
		{
			name:   "tag",
			text:   "---\nname: x\ndescription: !!str tagged\n---\nbody\n",
			path:   []string{"description"},
			reason: "tag",
			scope:  "entry",
		},
		{
			name:   "merge key",
			text:   "---\nname: x\nmetadata:\n  <<: *base\n  version: \"1\"\n---\nbody\n",
			path:   []string{"metadata", "<<"},
			reason: "merge key",
			scope:  "entry",
		},
		{
			// Document scope: an explicit key may itself span lines, so the
			// reader cannot locate the entries after it.
			name:   "explicit key",
			text:   "---\nname: x\n? description\n: value\n---\nbody\n",
			reason: "explicit key",
			scope:  "document",
		},
		{
			name:   "duplicate key",
			text:   "---\nname: x\ndescription: first\ndescription: second\n---\nbody\n",
			path:   []string{"description"},
			reason: "duplicate key",
			scope:  "entry",
		},
		{
			name:   "tab indentation",
			text:   "---\nname: x\nmetadata:\n\tversion: \"1\"\n---\nbody\n",
			reason: "tab",
			scope:  "document",
		},
		{
			name:   "directive",
			text:   "---\n%YAML 1.2\nname: x\n---\nbody\n",
			reason: "directive",
			scope:  "document",
		},
		{
			name:   "unclosed flow collection",
			text:   "---\nname: x\nallowed-tools: [Read, Bash\n---\nbody\n",
			reason: "unclosed flow",
			scope:  "document",
		},
		{
			name:   "document end marker",
			text:   "---\nname: x\n...\ndescription: y\n---\nbody\n",
			reason: "document end",
			scope:  "document",
		},
		{
			name:   "indentation matching no open block",
			text:   "---\nname: x\nmetadata:\n    version: \"1\"\n  carrier: y\n---\nbody\n",
			reason: "indentation",
			scope:  "document",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fm := ParseFrontmatter(c.text)

			if len(fm.Unreadable) == 0 {
				t.Fatalf("refusal not recorded; parse returned Keys=%v", fm.Keys)
			}
			var named bool
			for _, r := range fm.Unreadable {
				if strings.Contains(strings.ToLower(r.Reason), c.reason) {
					named = true
				}
				if r.Line <= 0 {
					t.Errorf("refusal %q carries no line number", r.Reason)
				}
			}
			if !named {
				t.Errorf("no refusal names %q; got %v", c.reason, fm.Unreadable)
			}

			switch c.scope {
			case "entry":
				n := fm.Lookup(c.path...)
				if n.Kind() != KindUnreadable {
					t.Errorf("%s kind = %s, want %s — a key that is present but "+
						"unreadable must not read as absent",
						strings.Join(c.path, "."), n.Kind(), KindUnreadable)
				}
				if _, ok := n.Scalar(); ok {
					t.Errorf("%s handed out a value it could not read", strings.Join(c.path, "."))
				}
				// The rest of the document still parses: an entry-scope
				// refusal is local to its entry.
				if got := fm.Keys["name"]; got != "x" {
					t.Errorf("name = %q, want x — an entry refusal must not lose the document", got)
				}
			case "document":
				if k := fm.Root.Kind(); k != KindUnreadable {
					t.Errorf("root kind = %s, want %s", k, KindUnreadable)
				}
				if len(fm.Keys) != 0 || len(fm.Lists) != 0 {
					t.Errorf("a refused document handed out a partial parse: Keys=%v Lists=%v",
						fm.Keys, fm.Lists)
				}
			}
		})
	}
}

// TestUnreadableIsNotAbsent is the distinction stated on its own, because
// collapsing it is how the reader arrived here: both yield no value, and
// they must not yield the same answer.
func TestUnreadableIsNotAbsent(t *testing.T) {
	fm := ParseFrontmatter("---\nname: x\ndescription: !!str tagged\n---\nbody\n")

	present := fm.Lookup("name")
	unreadable := fm.Lookup("description")
	absent := fm.Lookup("license")

	if present.Kind() != KindScalar {
		t.Errorf("name kind = %s, want %s", present.Kind(), KindScalar)
	}
	if unreadable.Kind() != KindUnreadable {
		t.Errorf("description kind = %s, want %s", unreadable.Kind(), KindUnreadable)
	}
	if absent.Kind() != KindAbsent {
		t.Errorf("license kind = %s, want %s", absent.Kind(), KindAbsent)
	}
	if unreadable.Kind() == absent.Kind() {
		t.Error("unreadable and absent are the same answer")
	}
	if unreadable.Reason() == "" {
		t.Error("an unreadable node carries no reason")
	}
	if absent.Reason() != "" {
		t.Error("an absent node invented a reason")
	}
	// And the refusal is enumerable without walking the tree.
	if len(fm.Unreadable) != 1 || len(fm.Unreadable[0].Path) != 1 ||
		fm.Unreadable[0].Path[0] != "description" {
		t.Errorf("Unreadable = %v, want one entry at [description]", fm.Unreadable)
	}
}

// TestNoFrontmatterIsAbsentNotUnreadable — a markdown file with no
// frontmatter block has no document, which is not a refusal.
func TestNoFrontmatterIsAbsentNotUnreadable(t *testing.T) {
	for _, text := range []string{"# just a body\n", "---\nname: x\nno terminator\n"} {
		fm := ParseFrontmatter(text)
		if k := fm.Root.Kind(); k != KindAbsent {
			t.Errorf("%q: root kind = %s, want %s", text, k, KindAbsent)
		}
		if len(fm.Unreadable) != 0 {
			t.Errorf("%q: invented refusals %v", text, fm.Unreadable)
		}
	}
}

// --- the compatibility projection ---------------------------------------

// TestLegacyProjectionUnchanged pins Keys/Lists — the view every rule reads
// through — to the answers the previous reader gave for the shapes it could
// already handle. The slice adds reach; it must not move the rules.
func TestLegacyProjectionUnchanged(t *testing.T) {
	fm := ParseFrontmatter("---\n" +
		"name: demo\n" +
		"description: 'a quoted description'\n" +
		"compatibility: \"bash 3.2+\"\n" +
		"inline: [Read, Bash]\n" +
		"block:\n" +
		"  - Read\n" +
		"  - Bash\n" +
		"metadata:\n" +
		"  version: \"9.9.9\"\n" +
		"---\nbody\n")

	wantKeys := map[string]string{
		"name":          "demo",
		"description":   "a quoted description",
		"compatibility": "bash 3.2+",
		"inline":        "[Read, Bash]",
		"block":         "",
		"metadata":      "",
	}
	for k, want := range wantKeys {
		got, ok := fm.Keys[k]
		if !ok {
			t.Errorf("Keys[%q] absent", k)
			continue
		}
		if got != want {
			t.Errorf("Keys[%q] = %q, want %q", k, got, want)
		}
	}
	if len(fm.Keys) != len(wantKeys) {
		t.Errorf("Keys = %v, want exactly %d entries", fm.Keys, len(wantKeys))
	}
	for _, k := range []string{"inline", "block"} {
		got := fm.Lists[k]
		if len(got) != 2 || got[0] != "Read" || got[1] != "Bash" {
			t.Errorf("Lists[%q] = %v, want [Read Bash]", k, got)
		}
	}
	// A nested map contributes no top-level list, and the nested key is not
	// promoted to the top level.
	if _, ok := fm.Keys["version"]; ok {
		t.Error("a nested key was promoted to the top level")
	}
}

// TestLegacyProjectionOmitsUnreadableKeys documents the one thing the
// compatibility view cannot say. Keys is map[string]string: it has no
// spelling for "present but unreadable", so an unreadable key is missing
// from it and visible only through Lookup and Unreadable. Pinned so the
// gap is a known boundary rather than a silent one.
func TestLegacyProjectionOmitsUnreadableKeys(t *testing.T) {
	fm := ParseFrontmatter("---\nname: x\ndescription: !!str tagged\n---\nbody\n")
	if _, ok := fm.Keys["description"]; ok {
		t.Error("Keys carried a value for an unreadable key")
	}
	if fm.Lookup("description").Kind() != KindUnreadable {
		t.Error("the unreadable key is not reachable through Lookup either")
	}
}

// --- the reader reaches the rules ---------------------------------------

// TestBlockScalarDescriptionReachesTheRules is the non-vacuity check the
// fixtures need: a per-style fixture that only ever visits ParseFrontmatter
// proves the parser, not the gate. A block-scalar description over the spec
// limit must reach SK-I003 — before this slice the reader handed the rule
// the single character "|".
func TestBlockScalarDescriptionReachesTheRules(t *testing.T) {
	long := strings.Repeat("  padding text that is long enough to matter.\n", 40)
	root := writeBundle(t, map[string]string{
		"SKILL.md": "---\nname: bundle\ndescription: |\n" + long + "---\n\nBody.\n",
	})
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	var fired bool
	for _, f := range rep.Findings {
		if f.RuleID == "SK-I003" {
			fired = true
		}
	}
	if !fired {
		t.Error("SK-I003 did not see a block-scalar description over the 1024-char limit")
	}
}

// TestMultiLinePlainDescriptionIsNotMissing — the same reach, in the other
// direction: a description written as a multi-line plain scalar is present,
// and SK-I001 must stop calling it missing.
func TestMultiLinePlainDescriptionIsNotMissing(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"SKILL.md": "---\nname: bundle\ndescription:\n  A description written over\n  two lines.\n---\n\nBody.\n",
	})
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range rep.Findings {
		if f.RuleID == "SK-I001" {
			t.Errorf("SK-I001 called a multi-line plain description missing: %s", f.Message)
		}
	}
}
