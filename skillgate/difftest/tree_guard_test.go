package difftest

// Running the tests must leave the repository exactly as it found it.
//
// This is not housekeeping. The differential harness reads the repository's own
// skills as part of its corpus, so it already knows where the tree is; the
// version of it that shipped also wrote its report into that tree, which is how
// two generated files ended up in the working copy with no commit behind them.
// A test that writes into the repository makes `git status` a liar, and the
// release's boundary checks read `git status`.
//
// The guard is stated as the observable invariant — the tree is unchanged after
// the run — rather than as a rule about any one write. That way it holds for
// writes nobody has thought of yet, including ones a future fixture helper adds.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	root, err := repositoryRootDir()
	if err != nil {
		// Refusing to run is the honest response: a guard that quietly
		// disables itself when it cannot find the tree proves nothing.
		fmt.Fprintf(os.Stderr, "difftest: cannot locate the repository root, so the no-tree-writes guard cannot run: %v\n", err)
		os.Exit(1)
	}
	before, err := snapshotTree(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "difftest: cannot snapshot %s: %v\n", root, err)
		os.Exit(1)
	}

	code := m.Run()

	after, err := snapshotTree(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "difftest: cannot re-snapshot %s: %v\n", root, err)
		os.Exit(1)
	}
	if changes := diffTrees(before, after); len(changes) > 0 {
		fmt.Fprintf(os.Stderr, "difftest: the test run changed %d path(s) in the repository working tree, which no test may do:\n", len(changes))
		for _, c := range changes {
			fmt.Fprintf(os.Stderr, "  %s\n", c)
		}
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

// repositoryRootDir walks up from the working directory to the directory
// holding go.work — the marker that identifies this checkout rather than a
// module cache entry.
func repositoryRootDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.work at or above %s", dir)
		}
		dir = parent
	}
}

type treeEntry struct {
	Size    int64
	ModTime time.Time
	Mode    fs.FileMode
}

// snapshotTree records every file under root except git's own bookkeeping,
// which changes for reasons that have nothing to do with the tests.
func snapshotTree(root string) (map[string]treeEntry, error) {
	tree := map[string]treeEntry{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		tree[filepath.ToSlash(rel)] = treeEntry{info.Size(), info.ModTime(), info.Mode()}
		return nil
	})
	return tree, err
}

func diffTrees(before, after map[string]treeEntry) []string {
	var changes []string
	for p, a := range after {
		b, ok := before[p]
		switch {
		case !ok:
			changes = append(changes, "created:  "+p)
		case b != a:
			changes = append(changes, "modified: "+p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			changes = append(changes, "removed:  "+p)
		}
	}
	sort.Strings(changes)
	return changes
}
