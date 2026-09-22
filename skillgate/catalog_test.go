package skillgate

// The rule catalog, checked rather than read.
//
// `docs/skillgate-spec.md` publishes a table of every rule the gate can fire.
// Until this file that table was maintained by hand, which is the defect that
// ran through the previous release: a set asserted complete by a person rather
// than read off the artifact. Four slices of this release add or move rows in
// it, and a conflicting row survives a merge by being read rather than run.
//
// So the table is compared, in both directions, against the gate's own rule
// registry — and the registry is compared against the source that declares the
// rules, so no side is derived from the side it is checked against:
//
//	docs/skillgate-spec.md §Rule catalog  ──parsed──▶  documentedRules()
//	tripwireGroups + icmRules + harnessRules  ─────▶   RuleCatalog()
//	the package's .go sources, by AST      ──scanned─▶ declaredRules()
//
// Deriving both sides from one place proves nothing: a guard whose denominator
// comes from its own subject shrinks with it and stays green. Prose is read out
// of the Markdown, the registry out of the running gate, and the registry's own
// claim to be complete out of the source text the compiler reads.
//
// Every diagnostic below names the rule ID, and a severity mismatch names both
// values. "The two counts differ" is not a diagnostic — it sends the next
// reader off to diff two lists by hand, which is how the stale row survived in
// the first place.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// specPath is the published spec, relative to the package directory the tests
// run in.
const specPath = "../docs/skillgate-spec.md"

// specCatalogHeading is the section whose first table is the rule catalog.
const specCatalogHeading = "## Rule catalog"

// reRuleID is the rule identifier shape: SK-, one pack letter, three digits.
var reRuleID = regexp.MustCompile(`^SK-[A-Z][0-9]{3}$`)

// --- comparing two independent enumerations of one set --------------------
//
// The comparison is a pure function over two maps rather than a body of
// t.Errorf calls, for the reason profiler's adapter contract gives: a check
// that can only be shown to work by breaking the repository is a check nobody
// has watched work. TestTheCatalogCheckCanFail feeds it divergences on every
// run.
//
// It is written over `map[string]string` — id to the single claim being
// compared — rather than over rules, because this release has a second
// enumeration to make true the same way: S13's ceded-lane sentence, the list of
// gaps the release does not close. That list has the same shape (a named set,
// declared in one place and enumerated in prose in another) and can call this
// with its own two derivations; only the derivations are rule-specific.

// enumerationSubject names what is being compared and where each side came
// from, so a violation reads as an instruction rather than as a diff.
type enumerationSubject struct {
	// noun is the singular thing being enumerated, e.g. "rule".
	noun string
	// claim is what the value in each map asserts, e.g. "severity".
	claim string
	// artifact is where the derived side came from, in words.
	artifact string
	// documented is where the prose side came from, in words.
	documented string
	// orphanHint is the instruction given when the documented side has an
	// entry the artifact does not. It is a field because the cause differs by
	// subject and the wrong instruction sends the reader the wrong way: a
	// catalog row with no rule really is a row that outlived its rule, while a
	// stated limit with no lane is one nobody has driven yet. Empty means the
	// default below.
	orphanHint string
}

// defaultOrphanHint is what an extra documented entry usually means: the thing
// it describes was removed or renamed and the entry did not follow.
const defaultOrphanHint = "it was removed or renamed, and the entry outlived it"

func (s enumerationSubject) orphan() string {
	if s.orphanHint == "" {
		return defaultOrphanHint
	}
	return s.orphanHint
}

var ruleCatalogSubject = enumerationSubject{
	noun:       "rule",
	claim:      "severity",
	artifact:   "the gate's rule registry",
	documented: specPath + " §Rule catalog",
}

// enumerationDivergences returns one message per way the two sides disagree,
// and nothing when they agree. Each message names the id; a claim mismatch
// names both values.
func enumerationDivergences(sub enumerationSubject, artifact, documented map[string]string) []string {
	var out []string
	for _, id := range sortedKeys(artifact) {
		got, ok := documented[id]
		if !ok {
			out = append(out, fmt.Sprintf(
				"%s %s is in %s (%s %q) and %s has no entry for it — add the entry naming %s, or remove the %s",
				sub.noun, id, sub.artifact, sub.claim, artifact[id], sub.documented, id, sub.noun))
			continue
		}
		if got != artifact[id] {
			out = append(out, fmt.Sprintf(
				"%s %s: %s says %s %q, %s says %q — one of the two is stale",
				sub.noun, id, sub.artifact, sub.claim, artifact[id], sub.documented, got))
		}
	}
	for _, id := range sortedKeys(documented) {
		if _, ok := artifact[id]; !ok {
			out = append(out, fmt.Sprintf(
				"%s has an entry for %s %s (%s %q) and %s holds no such %s — %s",
				sub.documented, sub.noun, id, sub.claim, documented[id], sub.artifact, sub.noun, sub.orphan()))
		}
	}
	return out
}

// --- the three derivations -------------------------------------------------

// catalogSeverities is the running gate's side: every rule the engine has
// registered, with the severity it fires at.
func catalogSeverities() map[string]string {
	out := map[string]string{}
	for _, r := range RuleCatalog() {
		out[r.ID] = r.Severity
	}
	return out
}

// publishedTable describes one two-column claim table in the spec: which
// section it sits under, what its key column must look like, and how to name
// it when a row cannot be read.
//
// It is a type rather than one function because the spec now publishes more
// than one such table — the rule catalog here, the view axis in view_test.go —
// and a second hand-written parser is a second place for the prose side to be
// read wrongly.
type publishedTable struct {
	// heading is the exact heading line the table sits under.
	heading string
	// shape is the column layout, quoted back in a malformed-row error.
	shape string
	// headerKey is the first cell of the header row.
	headerKey string
	// keyNoun names what the first column holds, e.g. "rule id".
	keyNoun string
	// validKey reports whether a first cell is a well-formed key.
	validKey func(string) bool
}

var specCatalogTable = publishedTable{
	heading:   specCatalogHeading,
	shape:     "Rule | Sev | What it catches",
	headerKey: "Rule",
	keyNoun:   "rule id",
	validKey:  reRuleID.MatchString,
}

// specLine is one line of a spec section, carrying the line number it came
// from so a diagnostic can point at the file rather than quote it.
type specLine struct {
	n    int // 1-based line number in specPath
	text string
}

// specSection returns the lines under a heading, up to the next heading of any
// level.
//
// One reader for every section-scoped check in the package: the tables here and
// in view_test.go, and the limit lists in ceded_test.go. Stopping at any `#`
// rather than at `## ` matters — the view tables are `###` siblings, and a
// reader that only stopped at `## ` ran into the next subsection and was saved
// only by the table happening to end first.
func specSection(t *testing.T, heading string) []specLine {
	t.Helper()
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read %s: %v — the spec is the documented side; without it nothing below is checked", specPath, err)
	}
	lines := strings.Split(string(data), "\n")

	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = i + 1
			break
		}
	}
	if start < 0 {
		t.Fatalf("%s has no %q section: without it the comparison below holds over nothing", specPath, heading)
	}

	var out []specLine
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "#") {
			break
		}
		out = append(out, specLine{n: i + 1, text: lines[i]})
	}
	return out
}

// parse reads the first table under the section into key → second column.
//
// The prose side, and it is read as prose: the section heading is located, the
// first table under it is the one meant, and every row of it must be readable.
// A row this parser cannot read is an error rather than a skip — a silently
// dropped row is a row the comparison never sees.
func (pt publishedTable) parse(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	inTable := false
	for _, sl := range specSection(t, pt.heading) {
		line := strings.TrimSpace(sl.text)
		if !strings.HasPrefix(line, "|") {
			if inTable {
				break // the table ended
			}
			continue
		}
		inTable = true
		cells := tableCells(line)
		if len(cells) < 2 {
			t.Errorf("%s:%d is a table row with %d cells: the %q table is %s",
				specPath, sl.n, len(cells), pt.heading, pt.shape)
			continue
		}
		if cells[0] == pt.headerKey || isRuleSeparator(cells[0]) {
			continue // header, or the |---|---| rule under it
		}
		key := cells[0]
		if !pt.validKey(key) {
			t.Errorf("%s:%d: %q is in the %q column of the %q table and is not a %s — a row this "+
				"parser cannot read is a row the check never compares",
				specPath, sl.n, key, pt.headerKey, pt.heading, pt.keyNoun)
			continue
		}
		if prev, dup := out[key]; dup {
			t.Errorf("%s: the %q table has two rows for %s (%q and %q)", specPath, pt.heading, key, prev, cells[1])
			continue
		}
		out[key] = cells[1]
	}
	if len(out) == 0 {
		t.Fatalf("%s %s yielded no rows: every comparison over it would hold vacuously", specPath, pt.heading)
	}
	return out
}

// documentedRules parses the rule catalog table out of the published spec.
func documentedRules(t *testing.T) map[string]string {
	t.Helper()
	return specCatalogTable.parse(t)
}

// tableCells splits a Markdown table row into its trimmed cells.
func tableCells(line string) []string {
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	cells := strings.Split(line, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

// isRuleSeparator reports whether a cell is the |---| rule under a header.
func isRuleSeparator(cell string) bool {
	return cell != "" && strings.Trim(cell, ":-") == ""
}

// declaredRules reads every rule declaration out of the package's own Go
// sources, by AST.
//
// Out of the source rather than out of the registry it checks, for the reason
// profiler's declaredTiers reads doctor.go: a registry that names its own
// members cannot report the member somebody declared and forgot to register.
// This is the artifact side, and the only side the compiler also reads.
//
// A rule declares itself by a composite literal carrying a rule id as a string
// literal — `rule{id: "SK-T001", sev: SeverityBlocker, …}`, the tripwire form;
// `CatalogRule{ID: …, Severity: …}`, the form used where findings are built
// inline; and `Finding{RuleID: …, Severity: …}`, which is a rule emitted with
// no declaration at all and is exactly what this scan exists to catch.
func declaredRules(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	forEachProductionFile(t, func(path string, file *ast.File) {
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			fields := map[string]ast.Expr{}
			for _, elt := range lit.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				if key, ok := kv.Key.(*ast.Ident); ok {
					fields[key.Name] = kv.Value
				}
			}
			id, ok := ruleIDField(fields)
			if !ok {
				return true
			}
			sev, ok := severityField(fields)
			if !ok {
				t.Errorf("%s declares %s in a %s literal with no severity this scan can read: the rule's "+
					"severity is half of what the catalog publishes", path, id, litTypeName(lit))
				return true
			}
			if prev, dup := out[id]; dup && prev != sev {
				t.Errorf("%s declares %s at severity %q, and it is declared at %q elsewhere in the package",
					path, id, sev, prev)
				return true
			}
			out[id] = sev
			return true
		})
	})
	return out
}

// ruleIDField returns the rule id a literal declares, if any.
func ruleIDField(fields map[string]ast.Expr) (string, bool) {
	for _, key := range []string{"id", "ID", "RuleID"} {
		expr, ok := fields[key]
		if !ok {
			continue
		}
		s, ok := stringLiteral(expr)
		if !ok || !reRuleID.MatchString(s) {
			continue
		}
		return s, true
	}
	return "", false
}

// severityValues maps the severity constant identifiers to their values. It
// references the constants rather than repeating their strings, so renaming one
// breaks this file at compile time; an identifier that is not here is reported,
// never skipped.
var severityValues = map[string]string{
	"SeverityBlocker": SeverityBlocker,
	"SeverityHigh":    SeverityHigh,
	"SeverityMedium":  SeverityMedium,
	"SeverityLow":     SeverityLow,
	"SeverityInfo":    SeverityInfo,
}

// severityField returns the severity a rule literal declares.
func severityField(fields map[string]ast.Expr) (string, bool) {
	for _, key := range []string{"sev", "Severity"} {
		expr, ok := fields[key]
		if !ok {
			continue
		}
		if ident, ok := expr.(*ast.Ident); ok {
			if sev, known := severityValues[ident.Name]; known {
				return sev, true
			}
			return "", false
		}
		if s, ok := stringLiteral(expr); ok {
			return s, true
		}
		return "", false
	}
	return "", false
}

func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return s, true
}

func litTypeName(lit *ast.CompositeLit) string {
	switch typ := lit.Type.(type) {
	case *ast.Ident:
		return typ.Name
	case *ast.SelectorExpr:
		return typ.Sel.Name
	}
	return "composite"
}

// mentionedRuleIDs returns every rule id that appears as a string literal
// anywhere in the package's production sources, with the files it appears in.
//
// A rule id can reach a finding without passing through a declaration — an id
// handed to a helper that builds the finding is the form already in this
// package. Those ids are not declarations and this scan cannot read a severity
// off them, but a gate that emits an id the catalog does not publish is the
// same defect in the other direction.
func mentionedRuleIDs(t *testing.T) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	forEachProductionFile(t, func(path string, file *ast.File) {
		seen := map[string]bool{}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, err := strconv.Unquote(lit.Value)
			if err != nil || !reRuleID.MatchString(s) || seen[s] {
				return true
			}
			seen[s] = true
			out[s] = append(out[s], path)
			return true
		})
	})
	return out
}

// forEachProductionFile parses every non-test .go file in the package
// directory. Comments are excluded: a rule named in a comment is prose, and
// prose is what the catalog is being checked against.
func forEachProductionFile(t *testing.T, fn func(path string, file *ast.File)) {
	t.Helper()
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob the package sources: %v", err)
	}
	parsed := 0
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		parsed++
		fn(path, file)
	}
	if parsed == 0 {
		t.Fatal("no production source was parsed: every scan over the package's sources would come back empty")
	}
}

// --- the checks ------------------------------------------------------------

// TestSpecCatalogMatchesTheRuleSet is the table this slice exists for: the
// published catalog and the gate's rules are the same set at the same
// severities, in both directions.
func TestSpecCatalogMatchesTheRuleSet(t *testing.T) {
	code := catalogSeverities()
	if len(code) == 0 {
		t.Fatal("RuleCatalog() is empty, so the spec table would be checked against nothing")
	}
	for _, v := range enumerationDivergences(ruleCatalogSubject, code, documentedRules(t)) {
		t.Error(v)
	}
}

// TestRuleCatalogIsWhatTheSourceDeclares checks the registry against the
// artifact. Without it the catalog is one more hand-maintained list, and the
// spec table would agree with a list that had quietly stopped matching the
// code.
func TestRuleCatalogIsWhatTheSourceDeclares(t *testing.T) {
	declared := declaredRules(t)
	if len(declared) == 0 {
		t.Fatal("the source scan read no rule declaration, so the catalog is checked against nothing")
	}
	subject := enumerationSubject{
		noun:       "rule",
		claim:      "severity",
		artifact:   "the package's rule declarations, read from source",
		documented: "RuleCatalog()",
	}
	for _, v := range enumerationDivergences(subject, declared, catalogSeverities()) {
		t.Error(v)
	}
}

// TestEveryRuleIDInTheSourceIsInTheCatalog closes the third way a rule reaches
// a report: an id handed to a helper, carrying no declaration of its own.
func TestEveryRuleIDInTheSourceIsInTheCatalog(t *testing.T) {
	catalog := catalogSeverities()
	mentioned := mentionedRuleIDs(t)
	if len(mentioned) == 0 {
		t.Fatal("no rule id was found in the package's sources, so this check holds over nothing")
	}
	ids := make([]string, 0, len(mentioned))
	for id := range mentioned {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if _, ok := catalog[id]; !ok {
			t.Errorf("%s appears in %s and RuleCatalog() does not publish it: the gate can emit a rule the "+
				"catalog and the spec table both omit", id, strings.Join(mentioned[id], ", "))
		}
	}
}

// TestTheRuleScanReadsEveryDeclarationSite is the control on the derivation.
//
// The rules are declared in more than one place — the tripwire groups, the ICM
// statics, the per-harness leg — and a scan that had silently stopped reading
// one of them would leave the catalog agreeing with a smaller set and the spec
// table agreeing with both. So the scan is required to find a rule from each
// pack that ships, with the packs taken from the catalog rather than listed.
func TestTheRuleScanReadsEveryDeclarationSite(t *testing.T) {
	declared := declaredRules(t)
	packs := map[string]int{}
	for id := range declared {
		packs[id[3:4]]++
	}
	for _, pack := range []string{"T", "G", "I", "H"} {
		if packs[pack] == 0 {
			t.Errorf("the source scan found no SK-%s rule: the pack is declared somewhere this scan does not read",
				pack)
		}
	}
}

// TestTheCatalogCheckCanFail runs the comparison over inputs built to break it,
// so its ability to fail — and to name what diverged — is under test on every
// run rather than asserted once by whoever wrote it.
//
// The victim is taken from the set rather than named here: an id written down
// in this file is one more hand-maintained enumeration, and the first one to go
// stale would turn these controls into no-ops.
func TestTheCatalogCheckCanFail(t *testing.T) {
	code := catalogSeverities()
	doc := documentedRules(t)
	if v := enumerationDivergences(ruleCatalogSubject, code, doc); len(v) != 0 {
		t.Fatalf("the two sides disagree before any mutation, so the controls below prove nothing: %v", v)
	}
	victim := sortedKeys(code)[0]

	t.Run("a row deleted from the spec names the rule", func(t *testing.T) {
		mutated := copyOf(doc)
		delete(mutated, victim)
		v := enumerationDivergences(ruleCatalogSubject, code, mutated)
		if len(v) != 1 {
			t.Fatalf("deleting the row for %s produced %d violations, want 1: %v", victim, len(v), v)
		}
		if !strings.Contains(v[0], victim) {
			t.Errorf("the missing row is reported as %q, which does not name %s — a diagnostic that does not "+
				"name the rule sends the reader off to diff two lists by hand", v[0], victim)
		}
	})

	t.Run("a rule with no row names the rule", func(t *testing.T) {
		const added = "SK-Z999"
		mutated := copyOf(code)
		mutated[added] = SeverityHigh
		v := enumerationDivergences(ruleCatalogSubject, mutated, doc)
		if len(v) != 1 {
			t.Fatalf("a rule the spec does not document produced %d violations, want 1: %v", len(v), v)
		}
		if !strings.Contains(v[0], added) {
			t.Errorf("the undocumented rule is reported as %q, which does not name %s", v[0], added)
		}
	})

	t.Run("a severity mismatch names both values", func(t *testing.T) {
		was := doc[victim]
		other := SeverityLow
		if was == other {
			other = SeverityInfo
		}
		mutated := copyOf(doc)
		mutated[victim] = other
		v := enumerationDivergences(ruleCatalogSubject, code, mutated)
		if len(v) != 1 {
			t.Fatalf("a severity the spec disagrees with produced %d violations, want 1: %v", len(v), v)
		}
		for _, want := range []string{victim, was, other} {
			if !strings.Contains(v[0], want) {
				t.Errorf("the mismatch is reported as %q, which does not name %q", v[0], want)
			}
		}
	})
}

func copyOf(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// reTripwireCap reads the cap the spec publishes on the tripwire pack.
var reTripwireCap = regexp.MustCompile(`[Rr]ule count is capped at ([0-9]+) tripwires`)

// TestTripwirePackIsWithinTheDocumentedCap is the numeral in the spec's prose,
// made a claim the artifact settles.
//
// A cap is a bound rather than a count, so it may carry a numeral — but only
// one that something enforces. The number is read out of the sentence that
// publishes it, and the pack is counted out of the catalog, so raising the cap
// in prose does not quietly raise it in fact, and the twenty-first tripwire
// fails here rather than in review.
func TestTripwirePackIsWithinTheDocumentedCap(t *testing.T) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read %s: %v", specPath, err)
	}
	m := reTripwireCap.FindSubmatch(data)
	if m == nil {
		t.Fatalf("%s publishes no tripwire cap: the sentence this check enforces is gone, and with it the "+
			"only bound on the pack", specPath)
	}
	published, err := strconv.Atoi(string(m[1]))
	if err != nil {
		t.Fatalf("the published cap %q is not a number: %v", m[1], err)
	}

	var tripwires []string
	for _, r := range RuleCatalog() {
		if strings.HasPrefix(r.ID, "SK-T") {
			tripwires = append(tripwires, r.ID)
		}
	}
	if len(tripwires) > published {
		t.Errorf("the gate registers %d tripwires and %s caps the pack at %d: %s — an extension folds into an "+
			"existing leg, or the cap is a different number and the spec says so",
			len(tripwires), specPath, published, strings.Join(tripwires, ", "))
	}
}
