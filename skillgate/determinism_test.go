package skillgate

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

// TestReportDeterministic is the permanent repeated-run probe required by
// rule-language.md §8 before any rule migrates: report/v1 output must be
// byte-identical across runs once the wall clock is excluded. The
// testdata/determinism fixture exercises every historically nondeterministic
// emitter — Tarjan's map-seeded DFS (reference cycle), T015's mcpServers and
// env map ranges, T017's `bin` object map, and the SK-I002 same-name
// collision map. External scanners are force-skipped: this test asserts OUR
// determinism, not an environment's, so it runs green with no Python or Rust
// present.
func TestReportDeterministic(t *testing.T) {
	const runs = 50
	dir := filepath.Join("testdata", "determinism")
	opts := Options{SkipChecks: []string{"skillspector", "agnix", "skill-validator"}}

	var want []byte
	var wantFindings int
	for i := 0; i < runs; i++ {
		rep, err := NewEngine().Gate(dir, opts)
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
		rep.GeneratedAt = "" // the wall clock is the only legitimate drift
		got, err := json.MarshalIndent(rep, "", "  ")
		if err != nil {
			t.Fatalf("run %d: marshal: %v", i, err)
		}
		if i == 0 {
			want = got
			wantFindings = len(rep.Findings)
			if wantFindings == 0 {
				t.Fatal("fixture produced no findings — probe is not exercising the nondeterministic emitters")
			}
			continue
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("run %d: report/v1 output differs from run 0 (%d bytes)", i, len(got))
		}
	}
}
