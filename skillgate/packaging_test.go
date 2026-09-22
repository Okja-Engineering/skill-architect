package skillgate

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Slice-1 acceptance — the manifest trap [pi §6 #1]: a package manifest
// short-circuits the convention-directory walk, so a manifest that lists
// executables but not `skills` ships the tool and silently drops the skill
// that is the interface (SkillSpector ships exactly this defect). The test
// simulates install via every `*-plugin/plugin.json` in the repo: the
// manifest must declare a skills root, and skills/skill-gate/SKILL.md must
// be discoverable under it — fail the build otherwise.
func TestSkillGateDiscoverableViaPluginManifests(t *testing.T) {
	repoRoot := filepath.Join("..")
	manifests, err := filepath.Glob(filepath.Join(repoRoot, "*-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifests) == 0 {
		t.Skip("no plugin manifests — not running from the skill-architect repo")
	}
	for _, mpath := range manifests {
		data, err := os.ReadFile(mpath)
		if err != nil {
			t.Fatalf("%s: %v", mpath, err)
		}
		var doc map[string]any
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatalf("%s: manifest unparseable: %v", mpath, err)
		}
		skills, _ := doc["skills"].(string)
		if skills == "" {
			t.Errorf("%s: manifest declares no skills root — a skill-gate install would ship the binary and silently drop the skill", mpath)
			continue
		}
		// The skills root is declared relative to the package root — the
		// manifest's parent directory (repo/.claude-plugin/plugin.json →
		// repo/), matching how the harness resolves it at install.
		packageRoot := filepath.Dir(filepath.Dir(mpath))
		skillMD := filepath.Join(packageRoot, filepath.FromSlash(skills), "skill-gate", "SKILL.md")
		text, err := os.ReadFile(skillMD)
		if err != nil {
			t.Errorf("%s: skills/skill-gate/SKILL.md not discoverable under declared skills root %q: %v", mpath, skills, err)
			continue
		}
		// Cursor's rule too: frontmatter name must equal the folder name.
		if n := ParseFrontmatter(string(text)).Keys["name"]; n != "skill-gate" {
			t.Errorf("%s: skills/skill-gate/SKILL.md frontmatter name = %q, want skill-gate", mpath, n)
		}
	}
}

// The second manifest trap: a module manifest that misstates what the code
// actually needs. The plugin manifest above ships a tool and drops a skill;
// this one ships a dependency nobody imports, or an import nobody declared.
//
// The assertion is deliberately *not* "zero requires". Zero is a numeral about
// today's imports — the moment a slice takes a dependency on purpose, a
// zero-assertion is deleted rather than answered, which is the same defect as
// an enumeration asserted complete. The invariant that survives a new
// dependency is mutual containment between manifest and source:
//
//   - every direct require of a module is imported by some package in it, and
//   - every non-standard import of its packages resolves to one of its requires.
//
// Both sides are derived from the artifacts — the require set is parsed out of
// go.mod, the import set out of the Go source with go/parser — so neither is a
// list anyone maintains by hand. Taking a dependency means stating it; losing
// the last importer of one means removing it. Either omission names itself
// here, in either module.
//
// An import is resolved to its *longest* matching module path, sibling modules
// in this tree included. That is load-bearing rather than tidy: difftest's
// module path is a subpath of the production module's, so a production import
// of difftest looks like a self-import to a naive prefix test and builds
// cleanly under go.work — while failing for anyone who resolves the production
// module on its own. The workspace is exactly what hides this, so the check
// cannot lean on the toolchain to find it.
func TestModuleDependencySurfacesAreDeclared(t *testing.T) {
	roots := goModuleRoots(t, ".")
	if len(roots) == 0 {
		t.Fatal("no go.mod found under the skillgate tree — the walk found nothing to check")
	}
	mods := make(map[string]goModuleManifest, len(roots))
	for _, dir := range roots {
		mods[dir] = parseGoModManifest(t, dir)
	}

	for _, dir := range roots {
		mod := mods[dir]
		imports, files := collectGoImports(t, dir, roots)
		if files == 0 {
			t.Errorf("%s: module %s has no Go source — nothing was checked against its require block", dir, mod.Path)
			continue
		}

		// Candidate providers, longest match wins: the module itself, each of
		// its requires, and every sibling module in the tree.
		const (
			self     = "self"
			required = "require"
			sibling  = "workspace sibling"
		)
		provider := map[string]string{mod.Path: self}
		for _, r := range mod.Requires {
			provider[r.Path] = required
		}
		for other, om := range mods {
			if other != dir {
				if _, ok := provider[om.Path]; !ok {
					provider[om.Path] = sibling
				}
			}
		}

		used := map[string]bool{}
		for _, imp := range sortedKeys(imports) {
			if isStandardImportPath(imp) {
				continue
			}
			owner := ""
			for path := range provider {
				if importCoveredBy(imp, path) && len(path) > len(owner) {
					owner = path
				}
			}
			switch provider[owner] {
			case self:
				continue
			case required:
				used[owner] = true
			case sibling:
				t.Errorf("%s/go.mod: %q is imported by %s and is satisfied only by go.work — module %s must require %s, or anyone resolving it outside this workspace cannot build it",
					dir, imp, strings.Join(imports[imp], ", "), mod.Path, owner)
			default:
				t.Errorf("%s/go.mod: %q is imported by %s but no require declares it — the module's dependency surface is wider than it says",
					dir, imp, strings.Join(imports[imp], ", "))
			}
		}

		// The other direction: every direct require is actually imported.
		// Indirect requires are exempt by definition — they exist to pin a
		// transitive module the source never names.
		for _, r := range mod.Requires {
			if r.Indirect || used[r.Path] {
				continue
			}
			t.Errorf("%s/go.mod:%d: require %s %s is declared but no package in module %s imports it — the module claims a dependency it does not have",
				dir, r.Line, r.Path, r.Version, mod.Path)
		}

		t.Logf("%s (%s): %d Go files, %d requires, %d imported", dir, mod.Path, files, len(mod.Requires), len(used))
	}
}

type goModuleManifest struct {
	Path     string
	Requires []goModuleRequire
}

type goModuleRequire struct {
	Path     string
	Version  string
	Indirect bool
	Line     int
}

// goModuleRoots returns every directory at or under start that holds a go.mod,
// nested modules included — so a module carved out of another is discovered
// rather than listed.
func goModuleRoots(t *testing.T, start string) []string {
	t.Helper()
	var roots []string
	err := filepath.WalkDir(start, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipGoDir(p, start, d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if d.Name() == "go.mod" {
			roots = append(roots, filepath.Dir(p))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", start, err)
	}
	sort.Strings(roots)
	return roots
}

func skipGoDir(p, start, name string) bool {
	if p == start {
		return false
	}
	switch name {
	case ".git", "testdata", "vendor", "node_modules":
		return true
	}
	return strings.HasPrefix(name, "_")
}

func parseGoModManifest(t *testing.T, dir string) goModuleManifest {
	t.Helper()
	name := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	var m goModuleManifest
	block := ""
	for i, raw := range strings.Split(string(data), "\n") {
		line, indirect := raw, false
		if c := strings.Index(line, "//"); c >= 0 {
			indirect = strings.Contains(line[c:], "indirect")
			line = line[:c]
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if block != "" {
			if fields[0] == ")" {
				block = ""
				continue
			}
			if block == "require" && len(fields) >= 2 {
				m.Requires = append(m.Requires, goModuleRequire{fields[0], fields[1], indirect, i + 1})
			}
			continue
		}
		switch {
		case fields[0] == "module" && len(fields) >= 2:
			m.Path = fields[1]
		case len(fields) >= 2 && fields[1] == "(":
			block = fields[0]
		case fields[0] == "require" && len(fields) >= 3:
			m.Requires = append(m.Requires, goModuleRequire{fields[1], fields[2], indirect, i + 1})
		}
	}
	if m.Path == "" {
		t.Fatalf("%s: no module directive", name)
	}
	return m
}

// collectGoImports returns import path → the files importing it, for every Go
// file belonging to the module rooted at dir. Directories that are themselves
// module roots are pruned, so a nested module's imports are never attributed to
// its parent — which is what makes the carve-out measurable.
func collectGoImports(t *testing.T, dir string, allRoots []string) (map[string][]string, int) {
	t.Helper()
	nested := map[string]bool{}
	for _, r := range allRoots {
		if r != dir {
			nested[r] = true
		}
	}
	imports := map[string][]string{}
	files := 0
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipGoDir(p, dir, d.Name()) || nested[p] {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			return nil
		}
		files++
		for _, spec := range f.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Errorf("%s: unquotable import %s", p, spec.Path.Value)
				continue
			}
			imports[path] = append(imports[path], filepath.ToSlash(p))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return imports, files
}

// isStandardImportPath applies the toolchain's own rule rather than a list of
// standard packages: an import path whose first element carries no dot is
// resolved from the standard library, because a module path's first element is
// a domain.
func isStandardImportPath(p string) bool {
	first, _, _ := strings.Cut(p, "/")
	return !strings.Contains(first, ".")
}

// importCoveredBy reports whether import path imp is provided by module path
// mod — equal, or below it.
func importCoveredBy(imp, mod string) bool {
	return imp == mod || strings.HasPrefix(imp, mod+"/")
}
