package skillgate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Fingerprint is the content-bound identity of a finding: the rule, the
// normalized path, the matched text, and the severity. A baseline entry only
// suppresses when the fingerprint matches exactly — on any content drift it
// suppresses nothing (fail-closed).
func Fingerprint(ruleID, path, evidence, severity string) string {
	h := sha256.New()
	for _, s := range []string{ruleID, path, evidence, severity} {
		h.Write([]byte(s))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// BaselineEntry is one reviewed-and-accepted finding. Reason is mandatory:
// an entry without one is rejected at load time.
type BaselineEntry struct {
	RuleID      string `json:"rule_id"`
	Fingerprint string `json:"fingerprint"`
	Reason      string `json:"reason"`
}

// Baseline is the set of suppressed findings.
type Baseline struct {
	Entries []BaselineEntry `json:"entries"`
	byFP    map[string]BaselineEntry
	byRule  map[string][]BaselineEntry
}

// LoadBaseline reads a baseline JSON file. A missing file is an empty
// baseline. An entry with no reason is a hard error — baselines exist to
// record *why* a finding is accepted.
func LoadBaseline(path string) (*Baseline, error) {
	b := &Baseline{byFP: map[string]BaselineEntry{}, byRule: map[string][]BaselineEntry{}}
	if path == "" {
		return b, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return b, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, b); err != nil {
		return nil, fmt.Errorf("baseline %s: %w", path, err)
	}
	for i, e := range b.Entries {
		if strings.TrimSpace(e.Reason) == "" {
			return nil, fmt.Errorf("baseline %s: entry %d (%s) has no reason", path, i, e.RuleID)
		}
		b.byFP[e.Fingerprint] = e
		b.byRule[e.RuleID] = append(b.byRule[e.RuleID], e)
	}
	return b, nil
}

// Apply marks suppressed findings in place. Suppressed findings stay in the
// report — they are never deleted and never block.
func (b *Baseline) Apply(findings []Finding) {
	for i := range findings {
		if e, ok := b.byFP[findings[i].Fingerprint]; ok && e.RuleID == findings[i].RuleID {
			findings[i].Suppressed = true
			findings[i].SuppressReason = e.Reason
		}
	}
}

// ManifestSHA256 digests the sorted per-file hashes of a ledger — the
// "hash of the bundle" used to detect post-approval mutation (rug-pull class).
func ManifestSHA256(entries []LedgerEntry) string {
	var sums []string
	for _, e := range entries {
		if e.SHA256 != "" {
			sums = append(sums, e.Path+"="+e.SHA256)
		}
	}
	sort.Strings(sums)
	h := sha256.Sum256([]byte(strings.Join(sums, "\n")))
	return hex.EncodeToString(h[:])
}
