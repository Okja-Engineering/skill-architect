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
//     shape of the strings, and resolution comes **before any `..` is folded**.
//     A symlink pointing into the protected directory is outside by every
//     string comparison and inside in fact; a sibling sharing the name as a
//     prefix is the reverse. The ordering half of that property is not a
//     refinement of it: a barrier that resolved both paths and folded `..`
//     first — which is what `filepath.Clean` does, and what this did — is
//     wrong for every path whose `..` crosses a link, in both directions. See
//     [resolve].
//   - **A path that cannot be resolved counts as contained.** "I could not tell"
//     must not read as "go ahead".
//   - **The abort aborts.** [FatalTB] offers only `Fatalf`, so writing the
//     defect this replaces — a guard that called `Errorf`, marking the test
//     failed and then *returning*, so the install on the next line ran anyway —
//     does not compile.
package homesafe

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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
// strings.HasPrefix calls /Users/alice-backup a part of /Users/alice. Rel
// cleans both of its arguments, which is safe here and only here: both have
// been through resolve, so neither still carries a `..` for a clean to fold.
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

// maxSymlinkHops bounds the chain resolve will follow before calling a path
// unresolvable, the way the kernel does. Without it a cycle is an infinite
// loop, and an infinite loop inside a safety barrier is a barrier that never
// answers rather than one that refuses.
const maxSymlinkHops = 40

// resolve makes a path absolute and follows every symlink in it, including when
// the leaf does not exist yet.
//
// # Why this is a component walk and not filepath.Clean plus EvalSymlinks
//
// It used to be, and that was the defect. `filepath.Abs` cleans, and cleaning
// folds `..` **against the text**: a `..` that follows a symlink is folded
// against the *link's* name instead of against its target. So a path spelled
// `<dir>/link/../x`, where `link` points at `<protected>/sub`, folds to
// `<dir>/x` and is judged outside the protected directory — while a write on
// that same spelling goes through the kernel, which resolves `link` first, and
// lands at `<protected>/x`. The barrier said "outside" and the bytes went
// inside. `filepath.Join` folds too, so appending an unresolved remainder to a
// resolved head had the same hole one level down.
//
// The invariant, which the barrier's own header states and this is now the
// implementation of: **containment is decided on a path whose symlinks are
// resolved before any `..` is folded.** That cannot be got by reordering two
// library calls, because `..` and a symlink interleave — each `..` must be
// folded against however much of the path has been resolved *so far*. So the
// path is walked one component at a time, in the order the kernel walks it:
// a name is appended and followed if it is a link, and `..` pops the
// already-resolved prefix, where popping is correct precisely because that
// prefix holds no links and no `..` any more.
//
// A directory a test is about to create does not exist at the moment the
// barrier is asked about it, and refusing to answer for it would push every
// caller into checking a path only after the thing that creates it has run —
// which is after the write. So a component that does not exist is appended and
// the walk continues: the place the path *would* be created is what the barrier
// is about, and a `..` after a component that does not exist still folds
// against what is resolved, so a climb back into a symlinked region resolves
// that region rather than folding past it.
//
// A path that cannot be resolved for any other reason — a directory the process
// may not traverse, a cycle — is an error, and every caller above turns that
// into a refusal.
func resolve(path string) (string, error) {
	if path == "" {
		return "", &os.PathError{Op: "resolve", Path: path, Err: os.ErrInvalid}
	}
	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		// Concatenated rather than joined: filepath.Join cleans, which is the
		// fold this function exists to not do. The working directory's own
		// symlinks are resolved by the walk below like any other component.
		path = cwd + string(filepath.Separator) + path
	}

	resolved := string(filepath.Separator)
	remaining := splitPath(path)
	hops := 0
	for len(remaining) > 0 {
		name := remaining[0]
		remaining = remaining[1:]
		switch name {
		case "", ".":
			continue
		case "..":
			resolved = filepath.Dir(resolved)
			continue
		}

		next := childOf(resolved, name)
		target, err := os.Readlink(next)
		if err != nil {
			// Not a symlink (EINVAL) and not there at all (ENOENT) are both
			// "append it and carry on": the first is a real entry, the second
			// is a place a write would create. Anything else — a parent this
			// process may not traverse, a component that is not a directory —
			// is a path whose landing place is unknown, and unknown is not a
			// pass.
			if isNotSymlink(err) || os.IsNotExist(err) {
				resolved = next
				continue
			}
			return "", err
		}

		hops++
		if hops > maxSymlinkHops {
			return "", &os.PathError{Op: "resolve", Path: path, Err: errTooManyLinks}
		}
		if filepath.IsAbs(target) {
			resolved = string(filepath.Separator)
		}
		// The link's own target is walked before the rest of the path, exactly
		// as the kernel splices it in, so a target that is itself a link or
		// carries its own `..` resolves by the same rule.
		remaining = append(splitPath(target), remaining...)
	}
	return resolved, nil
}

// errTooManyLinks is the refusal for a chain that does not end. It is this
// package's own value rather than syscall.ELOOP so that the message reads as
// what happened on any platform.
var errTooManyLinks = errors.New("too many levels of symbolic links")

// splitPath breaks a path into its components without folding anything. Empty
// components and "." are left in for the walk to skip, because removing them
// here would be the beginning of a clean.
func splitPath(path string) []string {
	return strings.Split(path, string(filepath.Separator))
}

// childOf appends one name to an already-resolved absolute path. Not
// filepath.Join, which cleans; the prefix is resolved and the name is a single
// component, so there is nothing to clean and nothing that may be folded.
func childOf(dir, name string) string {
	if strings.HasSuffix(dir, string(filepath.Separator)) {
		return dir + name
	}
	return dir + string(filepath.Separator) + name
}

// isNotSymlink reports whether Readlink refused because the entry is a real
// file or directory rather than a link. That is an answer, not a failure, and
// it is the common case: every component of an ordinary path reaches it.
func isNotSymlink(err error) bool {
	return errors.Is(err, syscall.EINVAL)
}
