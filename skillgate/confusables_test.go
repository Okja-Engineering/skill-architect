package skillgate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"testing"
)

// confusableTableDigest pins the contents of the ported UTS #39 confusable
// skeleton table.
//
// The table is a port of a generated upstream artifact (see confusables.go),
// so the thing that must not happen to it is a hand-edit — and the thing that
// must not happen to its two copies is drift. The digest is over the table's
// *value*, canonically serialized, so it is insensitive to formatting and
// sensitive to every entry added, removed or changed.
//
// difftest/confusables_test.go pins the same constant against its own copy.
// Changing one copy fails one test; changing both in the same way fails both.
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
		t.Errorf("the ported confusable table has changed:\n  digest now %s\n  pinned     %s\n"+
			"this table is a port of an upstream generated artifact — if the change is a "+
			"re-port, update the digest here and in difftest/confusables_test.go; if it is "+
			"a hand-edit, it is the enumerate-the-forms defect and must be reverted",
			got, confusableTableDigest)
	}
}
