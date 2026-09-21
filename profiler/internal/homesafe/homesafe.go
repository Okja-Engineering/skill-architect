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
// # The four properties, each of which was proved by reverting it
//
//   - **Containment is decided by identity, not by any name.** The verdict is
//     whether the object a write creates-or-truncates is the protected
//     directory or is reachable from it without leaving it, asked with
//     [os.SameFile] over the components the kernel itself walked. Two earlier
//     versions of this decided it on names, and each was a live escape.
//     Comparing *unresolved* names is wrong because a symlink into the
//     protected directory is outside by every string comparison and inside in
//     fact, and because a sibling sharing a name prefix is the reverse.
//     Comparing *resolved* names is also wrong, and that is the half that is
//     easy to miss: this volume is case-insensitive and APFS is
//     normalisation-insensitive, so `.cursor`, `.CURSOR` and an NFD spelling
//     of a non-ASCII home are all one directory with several names, and a
//     resolved name is only as canonical as the spelling it was built from.
//     Folding `..` textually is wrong for a third reason again — a `..` that
//     follows a symlink folds against the link's own name — which is why
//     resolution is a component walk and no `filepath` function that calls
//     `Clean` is used anywhere in this file. See [PathContains] and [resolve].
//   - **A name is compared in exactly one place**: when the protected
//     directory does not exist yet it has no identity, so the components it
//     would be made of are compared as text — folded the way a
//     case-insensitive volume folds them, and refused rather than guessed at
//     when a name carries a byte outside ASCII. See [tailContains].
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

// PathContains reports whether child is parent, or something inside it,
// deciding by **identity** and never by comparing the two paths as names.
//
// # Why not a comparison of resolved names
//
// It used to be one, and that was the second defect in this file rather than
// the first. Resolution is necessary — the question is which directory will be
// written, not how the path was spelled — but it is not sufficient, because a
// resolved name is still a name. On the platform this repository is developed
// on, three separate mechanisms make a name not an identity: the volume is
// case-insensitive, so `.cursor` and `.CURSOR` are one directory with two
// names; APFS is normalisation-insensitive, so an NFC and an NFD spelling are
// one directory with two names; and on the shell side of the same barrier the
// resolved name came back through a command substitution, which strips trailing
// newlines. Every one of them was a live escape, reproduced against a write:
// PathContains said "outside" and the bytes landed inside.
//
// So the question is asked the only way that no spelling can change. The object
// a write creates or truncates lives in the deepest directory the kernel reaches
// while resolving the spelling, so containment is whether *that directory* is
// the parent or is under it — decided with os.SameFile, walking up through the
// resolved prefix's own components. Not filepath.Rel, which compares names, and
// not strings.HasPrefix, which additionally calls /Users/alice-backup a part of
// /Users/alice.
//
// Names are compared in exactly one place, and only where there is nothing else
// to compare: when the parent does not exist yet it has no identity, so the
// components it would be made of are compared as names — with three answers,
// the third being that there is no answer. See [tailContains] for why guessing
// at the filesystem's own folding is not available to a walk that serves two
// barriers of opposite polarity. The shell copies draw the same line in the same
// place.
func PathContains(parent, child string) (bool, error) {
	p, err := resolve(parent)
	if err != nil {
		return false, err
	}
	c, err := resolve(child)
	if err != nil {
		return false, err
	}

	// One stat of the parent's own directory, and then one per level of the
	// climb. Every path stat'ed here is a directory the walk has already
	// reached, so a failure means the tree changed underneath the decision,
	// and that is an error rather than a verdict.
	parentDir, err := os.Stat(joinComponents(p.dir))
	if err != nil {
		return false, err
	}

	if len(p.tail) > 0 {
		// The parent does not exist yet. Nothing can exist below a directory
		// that does not exist, so the child's own deepest directory has to be
		// the same object as the parent's, and what is left over is the name
		// comparison the invariant allows for names that are not yet anything.
		childDir, err := os.Stat(joinComponents(c.dir))
		if err != nil {
			return false, err
		}
		if !os.SameFile(childDir, parentDir) {
			return false, nil
		}
		return tailContains(p.tail, c.tail)
	}

	// The parent exists, so the question is pure identity: climb from the
	// child's own directory towards the filesystem root, and the child is
	// inside iff the parent is one of the directories passed on the way.
	// Climbing is popping components, which is exactly `..` here because the
	// prefix these components came from holds no symlink and no `..` any more.
	for climb := c.dir; ; climb = climb[:len(climb)-1] {
		here, err := os.Stat(joinComponents(climb))
		if err != nil {
			return false, err
		}
		if os.SameFile(here, parentDir) {
			return true, nil
		}
		if len(climb) == 0 {
			return false, nil
		}
	}
}

// tailContains reports whether a path whose not-yet-existing components are
// child begins with a parent whose not-yet-existing components are parent.
//
// This is the one comparison in this package made on names, and it is made only
// where both sides are names and nothing else: neither path exists, so there is
// no inode to ask about. It has three answers and not two, which is the whole of
// what makes it honest.
//
// **The same bytes** are the same name on every filesystem, so a byte match is
// an answer anywhere: contained.
//
// **Names that could not be one name however the volume compares them** are an
// answer too: not contained. This is the common case by far and it has to stay
// definite — the shell copy of this walk is asked about six protected roots for
// every destination it is given, and on a machine where one of them does not
// exist, answering "I cannot tell" would turn every destination into a refusal.
// Two names are definitely different when both are ASCII and they differ by more
// than case, because ASCII case is the only folding a filesystem applies to an
// ASCII name.
//
// **Anything left is undecidable, and it says so.** Two ASCII names differing
// only in case are one directory on a case-insensitive volume and two on a
// case-sensitive one; two names either of which carries a byte at or above 0x80
// may be an NFC and an NFD spelling of one name, and normalising Unicode needs
// tables this package has no business carrying. The temptation is to guess, and
// the guess was written twice before this comment was. It cannot be right:
// "this name may be the parent's name" is a *refusal* for a barrier that
// protects the parent and a *pass* for one that keeps writes inside it, and this
// one walk serves both — homesafe protects the user's home, while the suites'
// own fence keeps writes inside a scratch root. No guess is fail-closed for
// both. The error is, because every caller reports an error as contained.
func tailContains(parent, child []string) (bool, error) {
	if len(child) < len(parent) {
		return false, nil
	}
	head := child[:len(parent)]
	decidable := true
	for i, want := range parent {
		got := head[i]
		if got == want {
			continue
		}
		if isASCII(got) && isASCII(want) && !strings.EqualFold(got, want) {
			return false, nil
		}
		decidable = false
	}
	if decidable {
		return true, nil
	}
	return false, &os.PathError{
		Op:   "contains",
		Path: joinComponents(child),
		Err:  errUncomparableName,
	}
}

// isASCII reports whether every byte of s is below 0x80. Bytes and not runes,
// because what matters is whether the filesystem's own comparison of this name
// can differ from a byte comparison, and only bytes at or above 0x80 take part
// in Unicode canonical equivalence.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// maxSymlinkHops bounds a symlink chain so that a cycle terminates. It is not
// the kernel's limit and no longer claims to be: this said it bounded "the way
// the kernel does" and the kernel on this platform refuses at 32, not 40, so
// the comment was wrong about the one fact it asserted. It does not need to be
// the kernel's limit either — a chain the kernel refuses produces no write, so
// such a path has no object for a verdict to be about, and refusing is the safe
// answer for it. What the counter is actually for is a cycle, where Readlink
// succeeds for ever. The shell copies of this barrier use the same number for
// the same reason.
const maxSymlinkHops = 32

// target is where a write on a spelling would land: the deepest directory the
// kernel reaches while resolving it, as the components of a symlink-free
// absolute path, and the names after that which do not exist.
//
// Two lists and not one string, because a string would have to be taken apart
// again to climb it and `filepath.Dir` — which is what did that — calls Clean.
// A textual `..` folder in the file whose one invariant forbids textual `..`
// folding is the next bug, which is why the others were deleted rather than
// left unused, and this one is now gone the same way.
type target struct {
	dir  []string
	tail []string
}

// resolve walks a path the way the kernel walks it and reports where a write on
// it would land.
//
// # Why this is a component walk and not filepath.Clean plus EvalSymlinks
//
// It used to be, and that was the first defect. `filepath.Abs` cleans, and
// cleaning folds `..` **against the text**: a `..` that follows a symlink is
// folded against the *link's* name instead of against its target. So a path
// spelled `<dir>/link/../x`, where `link` points at `<protected>/sub`, folds to
// `<dir>/x` and is judged outside the protected directory — while a write on
// that same spelling goes through the kernel, which resolves `link` first, and
// lands at `<protected>/x`. The barrier said "outside" and the bytes went
// inside. `filepath.Join` folds too, so appending an unresolved remainder to a
// resolved head had the same hole one level down.
//
// That cannot be got by reordering two library calls, because `..` and a symlink
// interleave — each `..` must be folded against however much of the path has
// been resolved *so far*. So the path is walked one component at a time, in the
// order the kernel walks it, and `..` pops the already-resolved prefix, where
// popping is correct precisely because that prefix holds no links and no `..`
// any more.
//
// # The rule, once, for every component
//
// A component that is a **symlink** has its target walked in its place, exactly
// as the kernel splices it in, including at the leaf: a write follows a leaf
// link, so a leaf link into a protected directory is a write into it. A
// component that is a **directory** is entered. A component that **does not
// exist** begins the tail, because a directory a test is about to create does
// not exist at the moment the barrier is asked about it, and refusing to answer
// for it would push every caller into checking a path only after the thing that
// creates it has run — which is after the write. A component that exists, is
// not a directory and is **not the last** has no answer: the kernel answers
// ENOTDIR, so no write happens, and no answer is a refusal rather than a guess.
//
// A `..` inside the tail folds against the tail and, past its start, climbs the
// resolved prefix. The kernel answers ENOENT for a `..` after a component that
// does not exist, so no write happens on such a path either way; folding is the
// conservative reading of it.
//
// A path that cannot be resolved for any other reason — a directory the process
// may not traverse, a cycle — is an error, and every caller above turns that
// into a refusal.
func resolve(path string) (target, error) {
	if path == "" {
		return target{}, &os.PathError{Op: "resolve", Path: path, Err: os.ErrInvalid}
	}
	remaining := splitPath(path)
	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err != nil {
			return target{}, err
		}
		// The working directory's own components are prepended and then walked
		// like any other, rather than joined — filepath.Join cleans, which is
		// the fold this function exists not to do.
		remaining = append(splitPath(cwd), remaining...)
	}

	var out target
	hops := 0
	for len(remaining) > 0 {
		name := remaining[0]
		remaining = remaining[1:]
		switch name {
		case "", ".":
			continue
		case "..":
			if len(out.tail) > 0 {
				out.tail = out.tail[:len(out.tail)-1]
			} else if len(out.dir) > 0 {
				out.dir = out.dir[:len(out.dir)-1]
			}
			continue
		}
		if len(out.tail) > 0 {
			out.tail = append(out.tail, name)
			continue
		}

		next := childPath(out.dir, name)
		info, err := os.Lstat(next)
		if err != nil {
			if os.IsNotExist(err) {
				out.tail = append(out.tail, name)
				continue
			}
			return target{}, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			hops++
			if hops > maxSymlinkHops {
				return target{}, &os.PathError{Op: "resolve", Path: path, Err: errTooManyLinks}
			}
			link, err := os.Readlink(next)
			if err != nil {
				return target{}, err
			}
			if filepath.IsAbs(link) {
				out.dir = nil
			}
			remaining = append(splitPath(link), remaining...)
			continue
		}
		if info.IsDir() {
			out.dir = append(out.dir, name)
			continue
		}
		// An existing entry that is neither a directory nor a link. It is the
		// object a write would truncate if it is the last component, and
		// ENOTDIR if it is not.
		if hasName(remaining) {
			return target{}, &os.PathError{Op: "resolve", Path: next, Err: syscall.ENOTDIR}
		}
		out.tail = append(out.tail, name)
	}
	return out, nil
}

// hasName reports whether any component left in a walk is a real name rather
// than an empty segment or a `.`, which is the difference between a path that
// ends at a file and one that tries to go through it.
func hasName(remaining []string) bool {
	for _, name := range remaining {
		if name != "" && name != "." {
			return true
		}
	}
	return false
}

// errTooManyLinks is the refusal for a chain that does not end. It is this
// package's own value rather than syscall.ELOOP so that the message reads as
// what happened on any platform.
var errTooManyLinks = errors.New("too many levels of symbolic links")

// errUncomparableName is the refusal for the one comparison this package makes
// on names, when the two names are not the same bytes and so cannot be compared
// without guessing at how the filesystem would fold them. Reported rather than
// guessed, and every caller turns it into a refusal.
var errUncomparableName = errors.New("a name that does not exist yet cannot be compared with another the way the filesystem would compare them")

// splitPath breaks a path into its components without folding anything. Empty
// components and "." are left in for the walk to skip, because removing them
// here would be the beginning of a clean.
func splitPath(path string) []string {
	return strings.Split(path, string(filepath.Separator))
}

// childPath spells one name under an already-resolved component list. Spelled
// rather than appended to the slice, so that nothing can write into the
// component list's spare capacity behind the walk's back.
func childPath(dir []string, name string) string {
	base := joinComponents(dir)
	if base == string(filepath.Separator) {
		return base + name
	}
	return base + string(filepath.Separator) + name
}

// joinComponents spells an absolute path from resolved components. Not
// filepath.Join, which cleans; there is nothing here to clean, and the result is
// only ever handed to the kernel as a syscall argument, never compared with
// another path.
func joinComponents(components []string) string {
	if len(components) == 0 {
		return string(filepath.Separator)
	}
	return string(filepath.Separator) + strings.Join(components, string(filepath.Separator))
}
