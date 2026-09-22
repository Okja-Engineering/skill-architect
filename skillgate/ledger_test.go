package skillgate

import (
	"os"
	"path/filepath"
	"testing"
)

// writeBundle creates a skill directory in a temp dir and returns its root.
func writeBundle(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestLedgerCoversEveryFile(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"SKILL.md":           "---\nname: demo\ndescription: a demo skill\n---\nBody.\n",
		"scripts/run.sh":     "#!/bin/sh\necho ok\n",
		"references/deep.md": "details\n",
	})
	l, err := BuildLedger(root)
	if err != nil {
		t.Fatal(err)
	}
	cov := l.Coverage()
	if cov.Total != 3 || cov.Inspected != 3 || !cov.Complete {
		t.Fatalf("coverage = %+v, want 3/3 complete", cov)
	}
	for _, f := range l.Files {
		if f.Entry.SHA256 == "" {
			t.Fatalf("no sha for %s", f.Entry.Path)
		}
	}
	if got := l.Get("scripts/run.sh"); got == nil || got.Text == "" {
		t.Fatal("expected inspected content for scripts/run.sh")
	}
}

func TestLedgerSkipsBinary(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"SKILL.md": "body\n",
		"bin.dat":  "a\x00b\n",
	})
	l, err := BuildLedger(root)
	if err != nil {
		t.Fatal(err)
	}
	cov := l.Coverage()
	if cov.Complete || cov.Skipped != 1 {
		t.Fatalf("coverage = %+v, want 1 skip", cov)
	}
	if l.Files[1].Entry.Reason != SkipBinary {
		t.Fatalf("reason = %q, want %q", l.Files[1].Entry.Reason, SkipBinary)
	}
}

func TestLedgerSkipsSymlinkEscape(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := writeBundle(t, map[string]string{"SKILL.md": "body\n"})
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	l, err := BuildLedger(root)
	if err != nil {
		t.Fatal(err)
	}
	var sawSkip bool
	for _, f := range l.Files {
		if f.Entry.Path == "link.txt" && f.Entry.Reason == SkipSymlinkEscape {
			sawSkip = true
		}
	}
	if !sawSkip {
		t.Fatal("symlink escape not recorded as skipped")
	}
}

func TestLineNumber(t *testing.T) {
	f := &FileContent{Text: "ab\ncd\nef\n", Lines: lineOffsets([]byte("ab\ncd\nef\n"))}
	for offset, want := range map[int]int{0: 1, 2: 1, 3: 2, 6: 3, 8: 3} {
		if got := f.LineNumber(offset); got != want {
			t.Fatalf("LineNumber(%d) = %d, want %d", offset, got, want)
		}
	}
}
