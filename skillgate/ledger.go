package skillgate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Ledger limits. A file or bundle past these is recorded as skipped, never
// silently dropped.
const (
	maxFileBytes  = 1 << 20  // 1 MiB per file
	maxTotalBytes = 64 << 20 // 64 MiB per bundle
	maxFiles      = 4096
)

// FileContent pairs a ledger entry with the bytes that were inspected, so
// rules never re-read the filesystem.
type FileContent struct {
	Entry LedgerEntry
	Text  string // empty when the file was skipped
	Lines []int  // 1-based byte offset of each line start
	// views is the raw view plus every registered normalised view, built
	// once by BuildLedger. Read through Views(), never directly.
	views []*View
}

// Ledger is the G1 coverage ledger: every file under the bundle root gets a
// terminal outcome — inspected with its content, or skipped with an
// allow-listed reason.
type Ledger struct {
	Root  string
	Files []FileContent
}

// BuildLedger walks root, hashes every file, and classifies each entry. It
// never follows a symlink outside root and never reads past the limits.
func BuildLedger(root string) (*Ledger, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	// Resolve the root itself so in-root symlinks compare against real paths —
	// EvalSymlinks canonicalizes (e.g. /var → /private/var on macOS), and a
	// naive root would make every in-root link look like an escape.
	if resolved, rerr := filepath.EvalSymlinks(abs); rerr == nil {
		abs = resolved
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", root)
	}

	l := &Ledger{Root: abs}
	var total int64
	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		rel, relErr := filepath.Rel(abs, path)
		if relErr != nil {
			rel = path
		}
		if err != nil {
			l.record(rel, 0, "", SkipUnreadable)
			return nil
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if len(l.Files) >= maxFiles {
			l.record(rel, 0, "", SkipLimit)
			return filepath.SkipAll
		}

		// Resolve symlinks; a link that leaves the bundle root is a skip, and
		// the escape itself is also a T019 tripwire finding. A link that stays
		// inside root is followed and inspected like the file it points at.
		isLink := d.Type()&fs.ModeSymlink != 0
		if isLink {
			target, err := filepath.EvalSymlinks(path)
			if err != nil || !within(abs, target) {
				l.record(rel, 0, "", SkipSymlinkEscape)
				return nil
			}
		}

		// d.Info() describes the link itself, not its target; stat through.
		var fi fs.FileInfo
		if isLink {
			fi, err = os.Stat(path)
		} else {
			fi, err = d.Info()
		}
		if err != nil {
			l.record(rel, 0, "", SkipUnreadable)
			return nil
		}
		if !fi.Mode().IsRegular() {
			l.record(rel, fi.Size(), "", SkipUnsupported)
			return nil
		}
		if fi.Size() > maxFileBytes || total+fi.Size() > maxTotalBytes {
			l.record(rel, fi.Size(), hashFile(path), SkipTooLarge)
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			l.record(rel, fi.Size(), "", SkipUnreadable)
			return nil
		}
		total += fi.Size()
		sum := sha256.Sum256(data)
		if isBinary(data) {
			l.record(rel, fi.Size(), hex.EncodeToString(sum[:]), SkipBinary)
			return nil
		}
		l.Files = append(l.Files, FileContent{
			Entry: LedgerEntry{
				Path:    filepath.ToSlash(rel),
				Outcome: "inspected",
				Bytes:   fi.Size(),
				SHA256:  hex.EncodeToString(sum[:]),
			},
			Text:  string(data),
			Lines: lineOffsets(data),
		})
		return nil
	})
	if err != nil {
		return l, err
	}
	sort.Slice(l.Files, func(i, j int) bool { return l.Files[i].Entry.Path < l.Files[j].Entry.Path })
	// After the file list is final: views hold pointers into it, and checks
	// run concurrently over a ledger they treat as read-only.
	l.buildViews()
	return l, nil
}

func (l *Ledger) record(rel string, size int64, sum string, reason SkipReason) {
	l.Files = append(l.Files, FileContent{
		Entry: LedgerEntry{
			Path:    filepath.ToSlash(rel),
			Outcome: "skipped",
			Reason:  reason,
			Bytes:   size,
			SHA256:  sum,
		},
	})
}

// Entries returns the ledger's serializable form.
func (l *Ledger) Entries() []LedgerEntry {
	out := make([]LedgerEntry, len(l.Files))
	for i, f := range l.Files {
		out[i] = f.Entry
	}
	return out
}

// Coverage computes the ledger summary.
func (l *Ledger) Coverage() Coverage {
	c := Coverage{Total: len(l.Files)}
	for _, f := range l.Files {
		if f.Entry.Outcome == "inspected" {
			c.Inspected++
		} else {
			c.Skipped++
		}
	}
	c.Complete = c.Skipped == 0
	return c
}

// Get returns the inspected content for a bundle-relative path, or nil.
func (l *Ledger) Get(path string) *FileContent {
	for i := range l.Files {
		if l.Files[i].Entry.Path == path {
			return &l.Files[i]
		}
	}
	return nil
}

// hasDir reports whether any ledger entry lives under the given directory.
func (l *Ledger) hasDir(dir string) bool {
	prefix := strings.TrimSuffix(dir, "/") + "/"
	for _, f := range l.Files {
		if strings.HasPrefix(f.Entry.Path, prefix) {
			return true
		}
	}
	return false
}

// packageManifests returns the sorted bundle-relative paths of inspected
// manifest files — the package unit's self-declarations (G0).
func packageManifests(l *Ledger) []string {
	var out []string
	for _, f := range l.Files {
		if f.Entry.Outcome == "inspected" && isManifestFile(f.Entry.Path) {
			out = append(out, f.Entry.Path)
		}
	}
	sort.Strings(out)
	return out
}

// within reports whether target resolves inside root.
func within(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// isBinary reports whether the content looks binary: a NUL byte in the first
// 8 KiB is the standard heuristic.
func isBinary(data []byte) bool {
	n := len(data)
	if n > 8192 {
		n = 8192
	}
	for i := 0; i < n; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

func hashFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

// lineOffsets returns the 1-based byte offset of every line start, used to
// map byte offsets to line numbers for findings.
func lineOffsets(data []byte) []int {
	offs := []int{0}
	for i, b := range data {
		if b == '\n' {
			offs = append(offs, i+1)
		}
	}
	return offs
}

// LineNumber maps a byte offset to a 1-based line number.
func (f *FileContent) LineNumber(offset int) int {
	lo, hi := 0, len(f.Lines)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if f.Lines[mid] <= offset {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo + 1
}
