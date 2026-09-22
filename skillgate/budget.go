package skillgate

import (
	"encoding/json"
	"strings"
)

// Pack E — the always-on budget's per-item leg. Files are counted by an
// external tokenizer (skill-validator); items are the two [pi] classes that
// file counts miss, measured in exact characters with per-item attribution:
//
//   - skill_description: a SKILL.md frontmatter description — the L1 line
//     injected into the prompt listing every turn while the skill is
//     installed; unlike a loaded body it survives compaction.
//   - tool_description / tool_input_schema: text and schema a manifest
//     declares for a bundled tool. Both ride the request's `tools`
//     parameter and are billed every turn — pi measured the schema leg at
//     ~3.9× the injected prose on one real tool (1,007 vs 255 chars).
//
// Chars are the honest unit: exact, zero-dep, and never rounded into a
// billed count. Token attribution lands only when a measured counter covers
// the item — today none does at sub-file granularity, so Items carry chars.
//
// Slice-2 note (rec §6 #2): the named acceptance bench for this arithmetic
// is pi's before_agent_start assembled system-prompt string — exact in
// characters. It needs a pi install plus a fixture extension; deferred past
// the one-day bar for this slice (D9 keeps pi a bench fixture, not a target).

// isManifestFile reports whether a bundle path can declare tools or
// executable extensions: package manifests and plugin/harness manifests.
func isManifestFile(p string) bool {
	base := p[strings.LastIndex(p, "/")+1:]
	switch base {
	case "package.json", "plugin.json", "mcp.json", ".mcp.json":
		return true
	}
	return strings.HasSuffix(base, ".plugin.json")
}

// budgetItems extracts the always-on line items with per-item attribution.
func budgetItems(l *Ledger) []BudgetItem {
	var items []BudgetItem
	for i := range l.Files {
		f := &l.Files[i]
		if f.Entry.Outcome != "inspected" {
			continue
		}
		p := f.Entry.Path
		base := p[strings.LastIndex(p, "/")+1:]

		if base == "SKILL.md" {
			fm := ParseFrontmatter(f.Text)
			if d := fm.Keys["description"]; d != "" {
				name := fm.Keys["name"]
				if name == "" {
					name = p
				}
				items = append(items, BudgetItem{
					Item: "skill description: " + name, Class: "skill_description",
					File: p, Chars: len(d),
				})
			}
			continue
		}
		if isManifestFile(p) {
			items = append(items, manifestBudgetItems(p, f.Text)...)
		}
	}
	return items
}

// manifestBudgetItems walks a manifest's tool declarations. Any object with
// a `name` plus injected text (`description`, `guidelines`, `instructions`,
// `when_to_use`) is a tool_description item; an `input_schema`/`inputSchema`
// object is a tool_input_schema item counted at its serialized size plus
// the name and description that ride beside it in the tools parameter.
func manifestBudgetItems(file, text string) []BudgetItem {
	var doc any
	if json.Unmarshal([]byte(text), &doc) != nil {
		return nil
	}
	var items []BudgetItem
	var walk func(v any)
	walk = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			name, _ := t["name"].(string)
			var descChars int
			for _, k := range []string{"description", "guidelines", "instructions", "when_to_use"} {
				if s, ok := t[k].(string); ok {
					descChars += len(s)
				}
			}
			if name != "" && descChars > 0 {
				items = append(items, BudgetItem{
					Item: "tool description: " + name, Class: "tool_description",
					File: file, Chars: descChars,
				})
			}
			for _, k := range []string{"input_schema", "inputSchema"} {
				if schema, ok := t[k]; ok {
					if raw, err := json.Marshal(schema); err == nil {
						items = append(items, BudgetItem{
							Item: "tool input schema: " + name, Class: "tool_input_schema",
							File: file, Chars: len(raw) + len(name) + descChars,
						})
					}
				}
			}
			for _, v2 := range t {
				walk(v2)
			}
		case []any:
			for _, e := range t {
				walk(e)
			}
		}
	}
	walk(doc)
	return items
}
