package skillgate

import (
	"encoding/json"
	"os"
	"path/filepath"
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
