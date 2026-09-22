package skillgate

import "sort"

// The rule catalog — what the gate can report, answered by the gate.
//
// `docs/skillgate-spec.md` publishes a table of every rule, and until this file
// that table was the only enumeration of them: maintained by hand, beside a
// code base that registers its rules in three different shapes. A list typed
// out beside the set it describes is the list that does not mention the rule
// somebody added, and four slices of this release add rows to that table.
//
// So the catalog is assembled from the registries themselves:
//
//   - tripwireGroups  — the scanning rules, SK-T* and SK-G*, which carry their
//     severity in the `rule` struct the group runs.
//   - icmRules        — the ICM statics, whose findings icmCheck builds inline
//     because they need the parsed frontmatter and the measured token budget,
//     not a file scan.
//   - harnessRules    — the per-harness leg, same reason.
//
// The second and third are declarations beside the code that emits their
// findings, and the emitting code reads its identity out of them, so a rule's
// severity is stated once and this catalog cannot disagree with what the
// finding carries. catalog_test.go checks all three against the source by AST:
// a rule declared and registered nowhere, or emitted with no declaration,
// fails there by name.

// CatalogRule is one rule the gate can report, as the gate declares it.
//
// It publishes what identifies a rule to a reader deciding whether to care:
// the id findings carry, the severity it fires at, the quality dimension it
// belongs to, and the registered check that runs it — which is also the name
// that would appear in checks_skipped if it did not run. Remediation effort is
// not here: it is a property of the finding, and one rule's legs legitimately
// differ (SK-I002's name legs cost five minutes, its collision leg ten).
type CatalogRule struct {
	// ID is the rule identifier carried on every finding it produces.
	ID string `json:"id"`
	// Severity is the level the rule fires at.
	Severity string `json:"severity"`
	// Quality is the dimension the rule belongs to: security, reliability,
	// or maintainability.
	Quality string `json:"quality"`
	// Check is the registered check that runs the rule.
	Check string `json:"check"`
}

// RuleCatalog returns every rule the gate can report, sorted by id.
//
// Sorted because a catalog is also an interface: `--list-checks` and the spec
// comparison both read it, and map order is not an order.
func RuleCatalog() []CatalogRule {
	var out []CatalogRule
	for _, gr := range tripwireGroups {
		for _, r := range gr.rules {
			out = append(out, CatalogRule{
				ID: r.id, Severity: r.sev, Quality: r.quality, Check: gr.name,
			})
		}
	}
	out = append(out, icmRules...)
	out = append(out, harnessRules...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
