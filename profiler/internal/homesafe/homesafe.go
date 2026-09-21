// Package homesafe is the barrier between a test that hands a directory to
// code that writes and the directory the user actually lives in.
//
// # Why it is a package and not a helper in each test file
//
// Three test packages now hand a home to production code — `profiler` for
// `InstallHooks` and `AppendSpool`, `profiler/cmd` for the binary's own `$HOME`,
// and `profiler`'s `doctor` tests, which seed a `.cursor/hooks.json` and a spool
// inside the home they then ask about. The barrier existed twice by the end of
// the hook-spool slice and a third copy was the moment the duplication stopped
// being cheap: three copies of a guard are three chances for one of them to be
// the copy that degraded, and the degradation is invisible because a guard that
// never fires in a healthy tree changes nothing observable when it stops
// working. That is what happened once already — one copy became a string prefix
// test and the whole suite stayed green.
//
// It is an ordinary package rather than a test file because Go cannot share a
// `_test.go` file across packages, and `internal/` rather than exported surface
// because the promise is bounded to this module: nothing outside it can bind to
// this, so it can never become an API somebody depends on. It is imported only
// from tests, so it is linked into no binary this module ships.
//
// # The three properties, each of which was proved by reverting it
//
//   - **Containment is decided after both paths are resolved**, never from the
//     shape of the strings. A symlink pointing into the protected directory is
//     outside by every string comparison and inside in fact; a sibling sharing
//     the name as a prefix is the reverse.
//   - **A path that cannot be resolved counts as contained.** "I could not tell"
//     must not read as "go ahead".
//   - **The abort aborts.** [FatalTB] offers only `Fatalf`, so writing the
//     defect this replaces — a guard that called `Errorf`, marking the test
//     failed and then *returning*, so the install on the next line ran anyway —
//     does not compile.
package homesafe

import (
	"os"
	"path/filepath"
	"strings"
)

// FatalTB is the part of *testing.T the barrier is allowed to use.
//
// It is this narrow on purpose. The bug the barrier replaces was a guard that
// called Errorf, which marks the test failed and then *returns* — so the
// install on the next line ran anyway. An interface carrying only Fatalf makes
// writing that bug a compile error rather than something a reviewer has to
// catch again.
type FatalTB interface {
	Helper()
	Fatalf(format string, args ...any)
}

// TempDirTB is a FatalTB that can also hand out a scratch directory, which is
// what [SandboxHome] needs. Stated as an interface rather than *testing.T so
// that the barrier's own tests can drive it with a recorder and watch the abort
// happen.
type TempDirTB interface {
	FatalTB
	TempDir() string
}

// MustBeOutside aborts unless dir is somewhere other than the real user's home
// directory. Every test that hands a home, a spool directory or a hooks.json
// path to code that writes goes through it.
//
// The explicit return after each Fatalf is not redundant. With a *testing.T,
// Fatalf does not come back; the return says the barrier does not depend on
// that, so the guarantee is this function's own rather than borrowed from
// testing.
func MustBeOutside(tb FatalTB, dir string) {
	tb.Helper()
	contained, why, err := RealHomeContains(dir)
	if err != nil {
		tb.Fatalf("refusing to run: %s: %v", why, err)
		return
	}
	if contained {
		tb.Fatalf("refusing to run: %s", why)
		return
	}
}

// SandboxHome returns a directory that stands in for a home directory, proved
// to be outside the real one before it is handed back.
//
// This is how a test names a home. A test that built one itself would be the
// test the barrier could not see.
func SandboxHome(tb TempDirTB) string {
	tb.Helper()
	dir := tb.TempDir()
	MustBeOutside(tb, dir)
	return dir
}

// RealHomeContains reports whether dir is the real user's home directory or
// something inside it, and says why when it is.
//
// An error is reported with contained=true. A path that could not be resolved
// is not a path shown to be outside the home, and "I could not tell" must not
// read as "go ahead".
//
// It is split out from the abort so it can be asserted in both directions. A
// guard with no negative test is a guard nobody has watched work.
func RealHomeContains(dir string) (contained bool, why string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return true, "the real home directory could not be named, so nothing can be shown to be outside it", err
	}
	contained, err = PathContains(home, dir)
	if err != nil {
		return true, dir + " could not be placed relative to the real home " + home + ", so where it would be written is unknown", err
	}
	if !contained {
		return false, "", nil
	}
	return true, dir + " is inside the real home " + home, nil
}

// PathContains reports whether child is parent or something inside it, deciding
// after both are resolved.
//
// Resolution is the point. The question is which directory will be written, not
// how the path was spelled: on macOS t.TempDir() hands back /var/folders/…,
// which is a symlink to /private/var/folders/…, $HOME can itself be a symlink,
// and `..` inside a path says nothing about where it lands. A comparison of the
// unresolved strings answers a different question from the one being asked.
//
// Containment is filepath.Rel rather than a prefix test, because a parent's
// name is a prefix of every sibling that starts with the same letters —
// strings.HasPrefix calls /Users/alice-backup a part of /Users/alice.
func PathContains(parent, child string) (bool, error) {
	resolvedParent, err := resolve(parent)
	if err != nil {
		return false, err
	}
	resolvedChild, err := resolve(child)
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(resolvedParent, resolvedChild)
	if err != nil {
		return false, err
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}

// resolve makes a path absolute and follows every symlink in it, including when
// the leaf does not exist yet.
//
// A directory a test is about to create does not exist at the moment the
// barrier is asked about it, and refusing to answer for it would push every
// caller into checking a path only after the thing that creates it has run —
// which is after the write. So the deepest existing ancestor is resolved and
// the remainder appended: the place the path *would* be created is what the
// barrier is about.
//
// A path that cannot be resolved for any other reason — a directory the process
// may not traverse, say — is an error, and every caller above turns that into a
// refusal.
func resolve(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	current, rest := filepath.Clean(abs), ""
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			return filepath.Join(resolved, rest), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		rest = filepath.Join(filepath.Base(current), rest)
		current = parent
	}
}
