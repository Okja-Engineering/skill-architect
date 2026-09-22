// Package skillgate implements the safety gate for Agent Skills: quarantine,
// coverage ledger, tripwire rules, optional external scanners, and a verdict
// that never claims more than the evidence supports.
//
// Invariants (see docs/research/recommendation.md §5):
//   - F12: the tool never says "safe" or "clean". Verdicts are APPROVE,
//     CAUTION, or REJECT, always beside the coverage ledger.
//   - F13: no verdict above CAUTION when the ledger is incomplete or any
//     check was skipped; every skip is named in checks_skipped.
//   - F14: the gate audits before load and never contains what has loaded.
//     Every gating claim is scoped to the read path and the moment before
//     load; the bash leg is a standing entry in checks_skipped, never
//     claimed closed — so CAUTION is the reachable ceiling today.
package skillgate

// ReportSchema is the machine-readable contract emitted by `skillgate gate`.
const ReportSchema = "skillgate/report/v1"

// Verdict values. These are the only three verdict strings the tool emits.
const (
	VerdictApprove = "APPROVE"
	VerdictCaution = "CAUTION"
	VerdictReject  = "REJECT"
)

// Severity levels, ordered most to least severe.
const (
	SeverityBlocker = "blocker"
	SeverityHigh    = "high"
	SeverityMedium  = "medium"
	SeverityLow     = "low"
	SeverityInfo    = "info"
)

// Exit codes for the CLI.
const (
	ExitPass      = 0 // verdict APPROVE or CAUTION, ledger complete (unless tolerated)
	ExitGate      = 1 // verdict REJECT, or ledger incomplete under --fail-on-incomplete
	ExitExecError = 2 // the gate itself failed to run
)

// Finding is one rule firing on the target bundle.
type Finding struct {
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"`
	Quality  string `json:"quality"` // security | reliability | maintainability
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Evidence string `json:"evidence,omitempty"`
	// View names the normalised rendering the rule matched on, when it was
	// not the raw text — e.g. "skeleton" for a payload spelled in fullwidth
	// or confusable characters. File, Line and Evidence are always the raw
	// source, whatever View says, so a reader can find the text in the file.
	// Empty for a raw hit, which keeps every pre-view finding unchanged.
	View           string `json:"view,omitempty"`
	EffortMinutes  int    `json:"effort_minutes"`
	Source         string `json:"source"` // skillgate | skillspector | agnix
	Fingerprint    string `json:"fingerprint"`
	Suppressed     bool   `json:"suppressed,omitempty"`
	SuppressReason string `json:"suppress_reason,omitempty"`
	// Advisory marks findings from opt-in external scanners: they report at
	// their true severity but can never force REJECT on their own — the block
	// set is Pack-B-owned so the gate's floor has no external dependency (D2).
	Advisory bool `json:"advisory,omitempty"`
}

// Blocks reports whether this finding participates in the block set: any
// Blocker or High finding that is neither suppressed nor advisory forces
// REJECT.
func (f Finding) Blocks() bool {
	if f.Suppressed || f.Advisory {
		return false
	}
	return f.Severity == SeverityBlocker || f.Severity == SeverityHigh
}

// SkipReason is the allow-listed enum for ledger skips. Every file in the
// bundle gets a terminal outcome; "skipped" is only ever one of these.
type SkipReason string

const (
	SkipNone          SkipReason = ""
	SkipTooLarge      SkipReason = "too_large"
	SkipBinary        SkipReason = "binary_unparsed"
	SkipUnreadable    SkipReason = "unreadable"
	SkipOutsideRoot   SkipReason = "outside_root"   // resolved path escapes the bundle root
	SkipSymlinkEscape SkipReason = "symlink_escape" // symlink target leaves the bundle root
	SkipUnsupported   SkipReason = "unsupported_type"
	SkipLimit         SkipReason = "file_count_limit"
)

// LedgerEntry is one file's terminal outcome in the coverage ledger.
type LedgerEntry struct {
	Path    string     `json:"path"`    // bundle-relative, forward slashes
	Outcome string     `json:"outcome"` // inspected | skipped
	Reason  SkipReason `json:"reason,omitempty"`
	Bytes   int64      `json:"bytes"`
	SHA256  string     `json:"sha256"`
}

// Coverage summarizes the ledger.
type Coverage struct {
	Total     int  `json:"total"`
	Inspected int  `json:"inspected"`
	Skipped   int  `json:"skipped"`
	Complete  bool `json:"complete"` // skipped == 0
}

// SkippedCheck names a stage or external scanner that did not run.
type SkippedCheck struct {
	Check  string `json:"check"`  // e.g. "skillspector", "agnix"
	Reason string `json:"reason"` // e.g. "binary not found"
}

// The names of the registered checks that are not tripwire groups — the
// groups take theirs from tripwireGroups. Named constants because the name is
// load-bearing three times over: it is what --skip-checks and --only match,
// what checks_skipped reports, and the check a CatalogRule names. A literal
// repeated across the registry and the catalog is a literal that can be
// renamed in one of them.
//
// Every name here is registered in the one place that knows the set —
// builtinChecks() in checks.go — and published by --list-checks.
const (
	CheckICM                = "icm"
	CheckHarnessFrontmatter = "harness-frontmatter"
	CheckSkillSpector       = "skillspector"
	CheckAgnix              = "agnix"
	CheckSkillValidator     = "skill-validator"

	// CheckPreactivationBashLeg is the standing F14 boundary: the one surface
	// no gate can close — reading a skill through the shell instead of the
	// harness's read path. Registered as a check that can never run, so it is
	// published like the rest and reported skipped on every report.
	CheckPreactivationBashLeg = "preactivation-bash-leg"

	// CheckICMTokenBudget is a leg inside icm, not a check of its own: it is
	// reported skipped when no token counter measured the body, and there is
	// nothing to select or skip independently. Declared on icm's Legs so a
	// reader who meets the name in checks_skipped can look it up.
	CheckICMTokenBudget = "icm-token-budget"
)

// Provenance records where the bundle came from (G0). For a local directory
// only Path is set. For a fetched URL, the quarantine fields are populated.
// PackageManifests lists the manifest files found in the package — the
// audited unit is the whole package, and a manifest that names executables
// makes that explicit.
type Provenance struct {
	Path             string   `json:"path,omitempty"`              // local input path
	URL              string   `json:"url,omitempty"`               // origin URL, when fetched
	Commit           string   `json:"commit,omitempty"`            // resolved commit SHA
	Quarantine       string   `json:"quarantine,omitempty"`        // quarantine directory
	SHA256OfSet      string   `json:"sha256_manifest,omitempty"`   // digest over sorted per-file hashes
	PackageManifests []string `json:"package_manifests,omitempty"` // manifest files found in the package
}

// TokenBudget is the always-on context cost: exact per-file token counts
// with the counter and measurement basis named, plus per-item attribution
// for the content a bundle injects outside the file list. "measured" means
// an exact count (o200k_base via skill-validator for files; exact chars for
// items); estimated/inferred bases are labeled as such and never presented
// as billed usage. Per the don't-build list, no figure here derives from
// cache-miss/prefix-invalidation signals.
type TokenBudget struct {
	TotalTokens int            `json:"total_tokens"`
	Files       []TokenPerFile `json:"files"`
	Items       []BudgetItem   `json:"items,omitempty"`
	Counter     string         `json:"counter"` // e.g. "skill-validator/o200k_base"
	Basis       string         `json:"basis"`   // measured | estimated | inferred | billed
}

// BudgetItem is one always-on context line item that is not a whole file —
// per [pi] the two classes bigger than the manifest file itself: (a) the
// description/guideline text an installed tool or skill injects into the
// prompt every turn (which survives compaction while a loaded skill body
// does not) and (b) the tool input schema billed via the `tools` request
// parameter every turn. Chars are measured exactly; tokens are reported
// only when a real tokenizer covered the item — never chars/4.
type BudgetItem struct {
	Item   string `json:"item"`             // e.g. "skill description: skill-audit"
	Class  string `json:"class"`            // skill_description | tool_description | tool_input_schema
	File   string `json:"file"`             // bundle file carrying the item
	Chars  int    `json:"chars"`            // exact character count
	Tokens int    `json:"tokens,omitempty"` // only when derived from a measured count
}

// TokenPerFile attributes tokens to one loaded file.
type TokenPerFile struct {
	File   string `json:"file"`
	Tokens int    `json:"tokens"`
}

// Report is the top-level skillgate/report/v1 document.
type Report struct {
	Schema        string         `json:"schema"`
	GeneratedAt   string         `json:"generated_at"`
	Tool          ToolInfo       `json:"tool"`
	Target        string         `json:"target"`
	Provenance    *Provenance    `json:"provenance,omitempty"`
	Verdict       string         `json:"verdict"`
	Findings      []Finding      `json:"findings"`
	Ledger        []LedgerEntry  `json:"ledger"`
	Coverage      Coverage       `json:"coverage"`
	ChecksSkipped []SkippedCheck `json:"checks_skipped"`
	Refusals      []string       `json:"refusals"`
	Tokens        *TokenBudget   `json:"tokens,omitempty"`
}

// ToolInfo identifies the emitting binary and version.
type ToolInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
