package difftest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"testing"
)

// confusableTableDigest pins the contents of this module's copy of the ported
// UTS #39 confusable skeleton table (ae6_skeleton.go).
//
// The production module carries a second copy at skillgate/confusables.go,
// because difftest is its own module and the production module may not import
// it (s02-record.md §7.4). Two copies of a generated artifact are two places
// to drift, so both are pinned to the same digest of the table's *value*: a
// hand-edit to either copy, or divergence between them, fails a test by name.
//
// The constant is duplicated rather than shared for the same reason the table
// is: a shared constant would be a cross-module import.
const confusableTableDigest = "e22863cc56f40b795731d28c951868f650dc8bfb8057eadbbcec0adeaf1bd9e4"

// confusableDigest serializes a confusable table canonically and hashes it.
func confusableDigest(table map[rune]string) string {
	keys := make([]rune, 0, len(table))
	for k := range table {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	h := sha256.New()
	for _, k := range keys {
		fmt.Fprintf(h, "%04X=%s\n", k, table[k])
	}
	return hex.EncodeToString(h.Sum(nil))
}

func TestConfusableTableMatchesPortedDigest(t *testing.T) {
	if got := confusableDigest(asciiConfusableSkeleton); got != confusableTableDigest {
		t.Errorf("this module's confusable table has changed:\n  digest now %s\n  pinned     %s\n"+
			"it must stay byte-equal in value to skillgate/confusables.go — if the change is "+
			"a re-port from upstream, update the digest in both tests; if it is a hand-edit, "+
			"it is the enumerate-the-forms defect and must be reverted",
			got, confusableTableDigest)
	}
}
