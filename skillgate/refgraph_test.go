package skillgate

import (
	"sort"
	"strings"
	"testing"
)

// g003 returns the sorted bundle-relative paths SK-G003 accused, for a
// bundle written from files. Every reachability test asserts the *whole*
// set, never "at least one" — an unreachability rule that is right about
// the orphan and wrong about everything else has not passed.
func g003(t *testing.T, files map[string]string) []string {
	t.Helper()
	root := writeBundle(t, files)
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range rep.Findings {
		if f.RuleID == "SK-G003" {
			out = append(out, f.File)
		}
	}
	sort.Strings(out)
	return out
}

func assertG003(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("SK-G003 = %v, want %v", got, want)
	}
}

// The headline case: a file in the skill's own tree that nothing references
// is unreachable, and the gate says so.
func TestG003OrphanFires(t *testing.T) {
	got := g003(t, map[string]string{
		"SKILL.md":                   "---\nname: x\ndescription: x\n---\nRead `references/guide.md`.\n",
		"references/guide.md":        "the guide\n",
		"references/zzquuxorphan.md": "nothing points here\n",
	})
	assertG003(t, got, []string{"references/zzquuxorphan.md"})
}

// The control that decides whether this is reachability at all: a file two
// and three hops from the entry point is reachable and must not fire. The
// orphan in the same fixture keeps the assertion non-vacuous — a rule that
// never fires would pass the transitive half on its own.
func TestG003TransitiveReachabilityDoesNotFire(t *testing.T) {
	got := g003(t, map[string]string{
		"SKILL.md":                   "---\nname: x\ndescription: x\n---\nStart at [a](references/a.md).\n",
		"references/a.md":            "then [b](b.md)\n",
		"references/b.md":            "then [c](c.md)\n",
		"references/c.md":            "the end\n",
		"references/zzquuxorphan.md": "nothing points here\n",
	})
	assertG003(t, got, []string{"references/zzquuxorphan.md"})
}

// Cycle ruling, both directions. Reachability is a forward walk from the
// entry set: a cycle hanging off the entry point is reachable through its
// entry edge, and an island cycle referencing only itself is not — mutual
// reference is not reachability, so every member of an unreached cycle is
// reported.
func TestG003ReachableCycleDoesNotFire(t *testing.T) {
	got := g003(t, map[string]string{
		"SKILL.md":                   "---\nname: x\ndescription: x\n---\nSee [a](references/a.md).\n",
		"references/a.md":            "to [b](b.md)\n",
		"references/b.md":            "back to [a](a.md)\n",
		"references/zzquuxorphan.md": "nothing points here\n",
	})
	assertG003(t, got, []string{"references/zzquuxorphan.md"})
}

func TestG003IslandCycleFiresForEveryMember(t *testing.T) {
	got := g003(t, map[string]string{
		"SKILL.md":        "---\nname: x\ndescription: x\n---\nbody with no references\n",
		"references/p.md": "to [q](q.md)\n",
		"references/q.md": "back to [p](p.md)\n",
	})
	assertG003(t, got, []string{"references/p.md", "references/q.md"})
}

// Out-of-directory ruling: an edge may leave the referencing skill's
// directory. The audited unit is the package, so a file another skill
// reaches is reachable, and only the file nothing reaches is accused.
func TestG003EdgeAcrossSkillDirectoriesIsFollowed(t *testing.T) {
	got := g003(t, map[string]string{
		"skills/a/SKILL.md":                   "---\nname: a\ndescription: a\n---\nShared: [s](../b/references/shared.md).\n",
		"skills/b/SKILL.md":                   "---\nname: b\ndescription: b\n---\nbody\n",
		"skills/b/references/shared.md":       "shared\n",
		"skills/b/references/zzquuxorphan.md": "nothing points here\n",
	})
	assertG003(t, got, []string{"skills/b/references/zzquuxorphan.md"})
}

// An edge whose target leaves the package root resolves to no ledger file
// and therefore to no node: SK-T019 owns that reference, and a sibling
// escaping must not turn a reachable file into an orphan.
func TestG003EscapingEdgeIsNotOurs(t *testing.T) {
	got := g003(t, map[string]string{
		"SKILL.md":                   "---\nname: x\ndescription: x\n---\nSee `../../outside/secret.md` and [a](references/a.md).\n",
		"references/a.md":            "reachable\n",
		"references/zzquuxorphan.md": "nothing points here\n",
	})
	assertG003(t, got, []string{"references/zzquuxorphan.md"})
}

// The false-orphan guard. `extractRefs` follows markdown links, backticked
// path tokens, and bare paths under the five conventional payload
// directories — a bare prose mention of a file under any other directory is
// a spelling it does not follow, and the missing edge must not become an
// accusation. The orphan in the same fixture proves the rule is still live.
func TestG003UnfollowedSpellingIsNotAnOrphan(t *testing.T) {
	got := g003(t, map[string]string{
		"SKILL.md":                   "---\nname: x\ndescription: x\n---\nRender with templates/report.md before writing.\n",
		"templates/report.md":        "template body\n",
		"references/zzquuxorphan.md": "nothing points here\n",
	})
	assertG003(t, got, []string{"references/zzquuxorphan.md"})
}

// Naming a directory discloses what is in it. Found by dogfooding against
// externally authored skills: `workflow-package-builder/SKILL.md` says "use
// the templates in `assets/contracts/`" and names no file inside, so a
// file-only graph accused both templates. A directory reference reaches
// every ledger file under it, and the reached files' own references are
// followed onward from there.
func TestG003DirectoryReferenceReachesItsContents(t *testing.T) {
	got := g003(t, map[string]string{
		"SKILL.md":                    "---\nname: x\ndescription: x\n---\nUse the templates in `assets/contracts/`.\n",
		"assets/contracts/stage.md":   "stage, and see [g](../../references/guide.md)\n",
		"assets/contracts/deep/ws.md": "workspace\n",
		"references/guide.md":         "guide\n",
		"references/zzquuxorphan.md":  "nothing points here\n",
	})
	assertG003(t, got, []string{"references/zzquuxorphan.md"})
}

// The other side of the same ruling, stated as a limit rather than left
// implicit: a directory reference is indistinguishable from layout prose,
// so a bundle that names its payload directory as a whole cannot produce an
// SK-G003 for a file inside it. The gap is deliberate — a missed orphan is
// a gap, a false orphan is an accusation against a correct skill.
func TestG003DirectoryReferenceSuppressesOrphansInsideIt(t *testing.T) {
	got := g003(t, map[string]string{
		"SKILL.md":                   "---\nname: x\ndescription: x\n---\nLayout: `references/`.\n",
		"references/zzquuxorphan.md": "nothing points here\n",
		"other/zzquuxorphan2.md":     "nor here\n",
	})
	assertG003(t, got, []string{"other/zzquuxorphan2.md"})
}

// Stated limit 2: scripts and other non-loaded-text files are a ceded lane.
// The resolver reads references out of text and cannot see a script's own
// imports or a hook config's command target, so an unreferenced script is
// never called unreachable — SK-T013 and SK-T017 own that surface.
func TestG003UnreferencedScriptIsNotReported(t *testing.T) {
	got := g003(t, map[string]string{
		"SKILL.md":                   "---\nname: x\ndescription: x\n---\nbody with no references\n",
		"scripts/dead.sh":            "#!/bin/sh\necho nobody runs me\n",
		"hooks/hooks.json":           `{"hooks":[]}`,
		"references/zzquuxorphan.md": "nothing points here\n",
	})
	assertG003(t, got, []string{"references/zzquuxorphan.md"})
}

// Stated limit 3: the base-name guard is blunt on purpose. A mention with
// nothing to do with the file still spares it, so a genuine orphan whose
// name appears in unrelated prose goes unreported.
func TestG003UnrelatedBaseNameMentionSparesAGenuineOrphan(t *testing.T) {
	got := g003(t, map[string]string{
		"SKILL.md":                   "---\nname: x\ndescription: x\n---\nNever name the payload here.\n",
		"references/setup.md":        "genuinely unreferenced\n",
		"references/zzquuxorphan.md": "Compare against the upstream setup.md in the vendor tree.\n",
	})
	assertG003(t, got, []string{"references/zzquuxorphan.md"})
}

// Harness-discovered files are entry points, not candidates: the agent
// loads AGENTS.md and a Cursor .mdc rule file by convention, so neither
// needs a reference to be reachable.
func TestG003HarnessDiscoveredFilesAreEntryPoints(t *testing.T) {
	got := g003(t, map[string]string{
		"SKILL.md":                   "---\nname: x\ndescription: x\n---\nbody\n",
		"references/AGENTS.md":       "memory file\n",
		"references/style.mdc":       "---\ndescription: style\n---\nrules\n",
		"references/zzquuxorphan.md": "nothing points here\n",
	})
	assertG003(t, got, []string{"references/zzquuxorphan.md"})
}

// Candidacy is the skill's disclosure payload — the subdirectories under a
// skill root. A file beside SKILL.md is skill or package furniture (README,
// CHANGELOG, a license) and is never accused of being unreachable.
func TestG003SkillRootFilesAreNotCandidates(t *testing.T) {
	got := g003(t, map[string]string{
		"SKILL.md":                   "---\nname: x\ndescription: x\n---\nbody\n",
		"README.md":                  "for humans\n",
		"CHANGELOG.md":               "history\n",
		"references/zzquuxorphan.md": "nothing points here\n",
	})
	assertG003(t, got, []string{"references/zzquuxorphan.md"})
}

// With no entry point there is no reachability question to answer. A
// bundle carrying no SKILL.md — a docs tree, a plugin fragment — is not a
// bundle of orphans; the rule is silent rather than accusing every file.
// The same file set *with* an entry point fires, so the silence is the
// ruling and not the rule being asleep.
func TestG003NoEntryPointNoFindings(t *testing.T) {
	payload := map[string]string{
		"references/zzquuxorphan.md": "nothing points here\n",
		"references/other.md":        "nor here\n",
	}
	withEntry := map[string]string{"SKILL.md": "---\nname: x\ndescription: x\n---\nbody\n"}
	for k, v := range payload {
		withEntry[k] = v
	}
	assertG003(t, g003(t, withEntry), []string{"references/other.md", "references/zzquuxorphan.md"})
	assertG003(t, g003(t, payload), nil)
}

// References resolved from a non-loaded-text file still count: a bundled
// script naming a template is a real edge an agent follows. The chain runs
// one hop *past* the script — SKILL.md → render.sh → template.md →
// partial.md — because only the hop past it distinguishes a followed edge
// from the mention guard sparing the script's own target.
func TestG003ScriptReferenceCountsAsAnEdge(t *testing.T) {
	got := g003(t, map[string]string{
		"SKILL.md":                   "---\nname: x\ndescription: x\n---\nRun `scripts/render.sh`.\n",
		"scripts/render.sh":          "#!/bin/sh\ncat references/template.md\n",
		"references/template.md":     "includes [partial](partial.md)\n",
		"references/partial.md":      "partial\n",
		"references/zzquuxorphan.md": "nothing points here\n",
	})
	assertG003(t, got, []string{"references/zzquuxorphan.md"})
}

// This repository's own skills are the dogfood corpus: they must be clean
// of SK-G003 on merit, not by the rule being asleep.
func TestG003CleanOnOwnSkills(t *testing.T) {
	ran := false
	for _, skill := range []string{"../skills/skill-audit", "../skills/skill-rewrite", "../skills/skill-gate"} {
		rep, err := NewEngine().Gate(skill, optsForTest())
		if err != nil {
			continue
		}
		ran = true
		for _, f := range rep.Findings {
			if f.RuleID == "SK-G003" {
				t.Errorf("%s: unexpected SK-G003 at %s (%s)", skill, f.File, f.Evidence)
			}
		}
	}
	if !ran {
		t.Skip("own skills not present")
	}
}

// SK-G001 carries a line number.
//
// It did not, and it is 81–94% of everything the gate reports on bundles
// this project did not write (S14, measured across three estates), so the
// large majority of an operator's report could not be navigated to. Every
// other rule carried a line; this one carried `file` and nothing else.
//
// The property that made the repair safe is the one asserted hardest below:
// it changes *what a finding says*, never *which findings fire*. The
// reference is still deduped on (file, reference) alone — adding the offset
// to that key would turn one dangling reference mentioned three times into
// three findings, which is a change to the report, not to its addressing.
func TestG001CarriesTheLineTheReferenceIsOn(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"SKILL.md": "---\nname: demo\ndescription: d\n---\n" +
			"# demo\n" +
			"\n" +
			"Intro paragraph with no references at all.\n" +
			"\n" +
			"See [the guide](references/missing-guide.md) for details.\n" +
			"\n" +
			"And `scripts/absent.sh` runs it.\n",
	})
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{
		"references/missing-guide.md": 9,
		"scripts/absent.sh":           11,
	}
	got := map[string]int{}
	for _, f := range rep.Findings {
		if f.RuleID != "SK-G001" {
			continue
		}
		if f.Line == 0 {
			t.Errorf("SK-G001 on %q carries no line — the finding cannot be navigated to", f.Evidence)
		}
		got[f.Evidence] = f.Line
	}
	for ev, line := range want {
		if got[ev] != line {
			t.Errorf("SK-G001 for %q reported line %d, want %d", ev, got[ev], line)
		}
	}
}

// TestG001ReportsOneFindingPerReferenceNotPerMention is the guard on the
// property above: the offset rides on the finding, never on its identity.
func TestG001ReportsOneFindingPerReferenceNotPerMention(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"SKILL.md": "---\nname: demo\ndescription: d\n---\n" +
			"First mention of `scripts/absent.sh`.\n" +
			"Second mention of `scripts/absent.sh`.\n" +
			"Third mention of `scripts/absent.sh`.\n",
	})
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	var n, line int
	for _, f := range rep.Findings {
		if f.RuleID == "SK-G001" && f.Evidence == "scripts/absent.sh" {
			n++
			line = f.Line
		}
	}
	if n != 1 {
		t.Fatalf("one dangling reference mentioned three times produced %d findings, want 1", n)
	}
	// The earliest mention, because refTokens is sorted by offset — a report
	// that cited the third mention would send the reader to the wrong place
	// first.
	if line != 5 {
		t.Errorf("reported line %d, want 5 — the first mention in the file", line)
	}
}

// TestG001CitesTheEarliestMentionAcrossExtractors is what makes the offset
// sort load-bearing rather than decorative.
//
// refTokens runs three patterns over the whole text in turn — markdown
// links, then backticks, then bare payload paths — so their results
// interleave and the raw append order is "whichever pattern ran first",
// not "whichever came first in the file". The function's doc comment has
// always claimed source order; the sort is what makes that true.
//
// Here the *bare* mention is on line 5 and the *backticked* one on line 6,
// and the bare-path pattern runs last. Unsorted, the reader is sent to
// line 6 while line 5 is sitting above it.
func TestG001CitesTheEarliestMentionAcrossExtractors(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"SKILL.md": "---\nname: demo\ndescription: d\n---\n" +
			"Run scripts/absent.sh first.\n" +
			"Then `scripts/absent.sh` again.\n",
	})
	rep, err := NewEngine().Gate(root, optsForTest())
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range rep.Findings {
		if f.RuleID == "SK-G001" && f.Evidence == "scripts/absent.sh" {
			if f.Line != 5 {
				t.Fatalf("cited line %d, want 5 — the earliest mention, which a "+
					"later-running extractor found", f.Line)
			}
			return
		}
	}
	t.Fatal("no SK-G001 for scripts/absent.sh")
}
