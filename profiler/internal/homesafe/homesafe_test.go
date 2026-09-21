package homesafe

// The barrier's own proofs, in the one place the barrier now lives.
//
// These are the assertions that used to exist twice — once in package
// profiler's tests and once in package cmd's — carried over on the move and
// re-run against the moved code rather than assumed to have survived it. A
// refactor is exactly where a guard quietly stops guarding, so every property
// the two copies proved is proved again here, and two of them are proved harder
// than either copy managed:
//
//   - the abort is observed in-process as well as across a process boundary
//     (TestMustBeOutside_AbortsAndSaysWhy), where the old copies could only
//     watch it from a child `go test`;
//   - SandboxHome is shown to actually ask the barrier
//     (TestSandboxHome_RefusesADirectoryInsideTheRealHome), which neither copy
//     could show: a guard that never fires in a healthy tree is unobservable,
//     and a mutation removing that call survived the hook-spool slice for
//     exactly that reason. Driving SandboxHome with a recorder that hands back
//     the real home makes the call observable, so the mutation now dies.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// recorder stands in for *testing.T so that an abort can be watched instead of
// aborting the test doing the watching. It records rather than stops, which is
// why the cross-process proof below still has to exist: this can show that
// Fatalf was called and only a real *testing.T can show that the statement
// after it never ran.
type recorder struct {
	messages []string
	tempDir  string
}

func (r *recorder) Helper() {}

func (r *recorder) Fatalf(format string, args ...any) {
	r.messages = append(r.messages, fmt.Sprintf(format, args...))
}

func (r *recorder) TempDir() string { return r.tempDir }

// TestPathContains_DecidesAfterResolution pins the decision to where a path
// lands rather than to how it is spelled.
//
// The two cases a spelling test gets wrong are the point of the table. A
// symlink pointing into the protected directory is *outside* by every string
// test and inside by every filesystem test; a sibling whose name begins with
// the same letters is *inside* by strings.HasPrefix and outside in fact. Both
// are reachable by accident — macOS hands t.TempDir() a symlinked path of its
// own — so both are asserted.
//
// Every path here is one this test created, so no case depends on what happens
// to exist beside the real home and no case is skipped. A skipped case would be
// the barrier's own proof passing by not running.
func TestPathContains_DecidesAfterResolution(t *testing.T) {
	root := t.TempDir()
	stands := filepath.Join(root, "alice") // stands for the home
	inside := filepath.Join(stands, ".cursor")
	sibling := filepath.Join(root, "alice-backup") // shares its name as a prefix
	for _, d := range []string{stands, inside, sibling} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	link := filepath.Join(root, "looks-harmless")
	if err := os.Symlink(inside, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	for _, tc := range []struct {
		name  string
		child string
		want  bool
	}{
		{"the directory itself", stands, true},
		{"something inside it", inside, true},
		{"a symlink from elsewhere pointing inside it", link, true},
		{"something inside it that does not exist yet", filepath.Join(inside, "hooks.json"), true},
		{"a path climbing back into it", filepath.Join(root, "alice-backup", "..", "alice", ".cursor"), true},
		{"a sibling whose name starts with the same letters", sibling, false},
		{"the parent of both", root, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := PathContains(stands, tc.child)
			if err != nil {
				t.Fatalf("PathContains(%q, %q): %v", stands, tc.child, err)
			}
			if got != tc.want {
				t.Errorf("PathContains(%q, %q) = %v, want %v", stands, tc.child, got, tc.want)
			}
		})
	}
}

// TestPathContains_AgreesWithWhereTheWriteLands is the invariant the table
// above cannot state, and the one the barrier was found failing.
//
// The table above is a list of spellings somebody thought of. That is what let
// this class survive: `..` folded lexically *before* symlinks are resolved is
// correct for every spelling in which no `..` crosses a link, so a table
// assembled by hand agrees with a barrier that is wrong. This test does not
// name an expectation at all. It performs the write on the *spelled* path,
// finds out from the filesystem where the bytes actually landed, and requires
// the barrier's verdict to be that answer.
//
// So the oracle is the kernel, and a spelling nobody thought of is still
// judged. Every directory in each case already exists and only the leaf file
// is new, which is the shape a real caller is in: the write goes through the
// kernel's own resolution with nothing folded by anybody first.
//
// The two directions are both here and they fail differently. A path that
// resolves *into* the protected directory while its fold says otherwise is the
// hole — the barrier says "outside" and the write lands inside. A path that
// resolves *out* of it while its fold says inside is the false refusal, which
// costs a legitimate destination. A repair that only closed the first would
// pass half of this.
func TestPathContains_AgreesWithWhereTheWriteLands(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "r")
	home := filepath.Join(root, "home") // stands for the protected directory
	for _, d := range []string{
		filepath.Join(home, ".cursor"),
		filepath.Join(home, "sub"),
		filepath.Join(root, "outside"),
		filepath.Join(root, "home-backup"),
	} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	// aside and chain live beside the protected directory and point into it;
	// out and rel live inside it and point away. Each is the parent of a `..`
	// in the table, which is the component whose fold is the whole question.
	for _, ln := range []struct{ target, name string }{
		{filepath.Join(home, ".cursor"), filepath.Join(root, "aside")},
		{filepath.Join(root, "aside"), filepath.Join(root, "chain")},
		{filepath.Join(root, "outside"), filepath.Join(home, "out")},
		{filepath.Join("..", "outside"), filepath.Join(home, "rel")},
	} {
		if err := os.Symlink(ln.target, ln.name); err != nil {
			t.Fatalf("symlink %s: %v", ln.name, err)
		}
	}

	// Spelled with string concatenation, never filepath.Join: Join calls Clean,
	// which folds the `..` these cases are about before the test has even run.
	// A case assembled with Join would be testing a different path from the one
	// its name claims.
	cases := []struct{ name, spelled string }{
		{"named directly inside", home + "/.cursor/f1"},
		{"through a symlink into it", root + "/aside/f2"},
		{"through a chain of symlinks into it", root + "/chain/f3"},
		{"a `..` crossing a symlink whose target's parent is inside", root + "/aside/../f4"},
		{"a `..` crossing a chain of symlinks", root + "/chain/../f5"},
		{"two `..` crossing a symlink, climbing back down", root + "/aside/../../home/.cursor/f6"},
		{"a `..` crossing a symlink that points away", home + "/out/../f7"},
		{"a `..` crossing a symlink whose own target is relative", home + "/rel/../f8"},
		{"through a symlink that points away", home + "/out/f9"},
		{"a sibling whose name starts with the same letters", root + "/home-backup/f10"},
		{"a plain `..` inside it", home + "/sub/../f11"},
		{"a plain `..` out of it", home + "/../f12"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(tc.spelled, []byte(tc.name), 0o600); err != nil {
				t.Fatalf("the case could not be performed, so it proves nothing: "+
					"writing %s: %v", tc.spelled, err)
			}
			landedInside, where := writeLandedUnder(t, home, tc.spelled)

			got, err := PathContains(home, tc.spelled)
			if err != nil {
				t.Fatalf("PathContains(%q, %q): %v", home, tc.spelled, err)
			}
			if got != landedInside {
				t.Errorf("PathContains(%q, %q) = %v, but a write on that spelling landed at %s, "+
					"which is %s the protected directory",
					home, tc.spelled, got, where, insideOrOutside(landedInside))
			}
		})
	}
}

func insideOrOutside(inside bool) string {
	if inside {
		return "inside"
	}
	return "outside"
}

// writeLandedUnder answers the oracle's question: after a write on `spelled`,
// is the file that now exists somewhere under `dir`?
//
// It asks the filesystem and not a path library. os.Stat on the spelled path is
// the kernel's own resolution of it, and the walk compares device and inode, so
// no part of this answer comes from the code under test. Symlinks are not
// followed by the walk, which matters: the fixture deliberately hangs links out
// of the protected directory, and a walk that chased them would call a file
// outside the directory a file inside it.
func writeLandedUnder(t *testing.T, dir, spelled string) (bool, string) {
	t.Helper()
	target, err := os.Stat(spelled)
	if err != nil {
		t.Fatalf("the write on %s cannot be located, so this case has no oracle: %v", spelled, err)
	}
	found := ""
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		if os.SameFile(fi, target) {
			found = path
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", dir, err)
	}
	if found != "" {
		return true, found
	}
	// Not under dir. Name where it did land, so a failure reads as a fact
	// rather than as "somewhere else".
	if real, err := filepath.EvalSymlinks(spelled); err == nil {
		return false, real
	}
	return false, spelled
}

// TestRealHomeContains_KnowsTheRealHome joins the decision to the directory it
// protects, and gives the reason that a refusal has to print.
//
// Only paths that already exist are asked about, and nothing is created: these
// are the only assertions in the package that look at the real home, and they
// look only.
func TestRealHomeContains_KnowsTheRealHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir: %v", err)
	}

	for _, tc := range []struct {
		name string
		dir  string
		want bool
	}{
		{"the real home itself", home, true},
		{"the Cursor config inside it", filepath.Join(home, ".cursor"), true},
		{"the spool inside it", filepath.Join(home, ".skill-architect", "spool"), true},
		{"a temp dir", t.TempDir(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, why, err := RealHomeContains(tc.dir)
			if err != nil {
				t.Fatalf("RealHomeContains(%q): %v", tc.dir, err)
			}
			if got != tc.want {
				t.Errorf("RealHomeContains(%q) = %v, want %v (reason %q)", tc.dir, got, tc.want, why)
			}
			if got && why == "" {
				t.Error("a path was called contained with no reason given, so a failure would not say what it found")
			}
			if !got && why != "" {
				t.Errorf("a path outside the home carries a reason %q, which reads as a refusal", why)
			}
		})
	}
}

// TestRealHomeContains_RefusesRatherThanGuessesWhenItCannotResolve pins the
// answer on the side of safety. A path the barrier cannot resolve is not a path
// it has shown to be outside the home, and "I could not tell" must not read as
// "go ahead".
func TestRealHomeContains_RefusesRatherThanGuessesWhenItCannotResolve(t *testing.T) {
	tmp := t.TempDir()

	// A directory the process cannot traverse: resolution of anything beneath
	// it fails with EACCES rather than ENOENT, which is the case that must not
	// be read as "does not exist, therefore fine". A path that merely does not
	// exist is *not* this case — resolve answers for it deliberately, so using
	// one here would make the test pass by not running.
	sealed := filepath.Join(tmp, "sealed")
	if err := os.Mkdir(sealed, 0o000); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(sealed, 0o700) })

	unresolvable := filepath.Join(sealed, "inner", ".cursor")

	// Whether this case is reachable at all is asked of the filesystem, not of
	// the function under test. Skipping on what RealHomeContains returned would
	// make the test pass by not running the moment the answer went wrong —
	// which is what happened once: a mutation that made an unresolvable path
	// read as "outside the home" turned this test green by skipping it.
	if _, err := filepath.EvalSymlinks(unresolvable); err == nil || os.IsNotExist(err) {
		t.Skipf("this filesystem resolves %s without a permission error, so the case is not reachable here", unresolvable)
	}

	contained, why, err := RealHomeContains(unresolvable)
	if err == nil {
		t.Error("RealHomeContains reported no error for a path the filesystem refused to resolve")
	}
	if !contained {
		t.Error("a path that could not be resolved was reported as outside the real home; \"I could not tell\" must not read as \"go ahead\"")
	}
	if why == "" {
		t.Error("no reason was given for the refusal")
	}
}

// TestMustBeOutside_AbortsAndSaysWhy watches the abort in-process, in both
// directions: a directory inside the real home is refused with a reason naming
// it, and one outside passes silently.
//
// The recorder is what makes the positive direction observable at all. With a
// real *testing.T the refusal would abort the test doing the asserting, which
// is why the old copies could only watch the barrier from a child process.
func TestMustBeOutside_AbortsAndSaysWhy(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir: %v", err)
	}

	inside := &recorder{}
	MustBeOutside(inside, filepath.Join(home, ".cursor"))
	if len(inside.messages) == 0 {
		t.Fatal("the barrier let a path inside the real home through without a word")
	}
	if !strings.Contains(inside.messages[0], "refusing") {
		t.Errorf("the refusal reads %q, and a reader scanning for a refusal will miss it", inside.messages[0])
	}
	if !strings.Contains(inside.messages[0], home) {
		t.Errorf("the refusal reads %q and does not name the home it is protecting", inside.messages[0])
	}

	outside := &recorder{}
	MustBeOutside(outside, t.TempDir())
	if len(outside.messages) != 0 {
		t.Errorf("a directory outside the real home was refused: %q", outside.messages)
	}
}

// TestSandboxHome_RefusesADirectoryInsideTheRealHome is the assertion neither
// old copy could make.
//
// SandboxHome's guard never fires in a healthy tree, because t.TempDir() is
// never inside the home — so deleting the call changed nothing observable and a
// mutation that deleted it survived. Handing SandboxHome a TempDir that returns
// the real home makes the call observable, and the mutation dies.
//
// Nothing is written: SandboxHome only asks the barrier about the path.
func TestSandboxHome_RefusesADirectoryInsideTheRealHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir: %v", err)
	}

	pretending := &recorder{tempDir: filepath.Join(home, ".skill-architect", "spool")}
	got := SandboxHome(pretending)
	if len(pretending.messages) == 0 {
		t.Errorf("SandboxHome handed back %q inside the real home without asking the barrier", got)
	}

	honest := &recorder{tempDir: t.TempDir()}
	if got := SandboxHome(honest); got != honest.tempDir {
		t.Errorf("SandboxHome returned %q, want the directory it was given, %q", got, honest.tempDir)
	}
	if len(honest.messages) != 0 {
		t.Errorf("a sandbox outside the real home was refused: %q", honest.messages)
	}
}

// TestBarrier_RefusesWhenItCannotEvenNameTheHome covers the two refusal paths
// that fire when the barrier cannot get as far as comparing anything.
//
// Both are the same rule as the unresolvable-path case, one step earlier: if the
// real home cannot be named, or cannot be resolved, then nothing has been shown
// to be outside it and every directory has to be refused. A barrier that
// shrugged here would wave through every path on a machine with a broken $HOME,
// which is the one machine most likely to have a surprising one.
//
// HOME is set for the duration of the test only, and nothing is written.
func TestBarrier_RefusesWhenItCannotEvenNameTheHome(t *testing.T) {
	// A directory the process cannot traverse, so resolving the home itself
	// fails with EACCES rather than with "does not exist".
	sealed := filepath.Join(t.TempDir(), "sealed")
	if err := os.Mkdir(sealed, 0o000); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(sealed, 0o700) })
	unresolvableHome := filepath.Join(sealed, "home")

	elsewhere := t.TempDir()

	for _, tc := range []struct {
		name string
		home string
	}{
		{"the home cannot be named at all", ""},
		{"the home cannot be resolved", unresolvableHome},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.home == unresolvableHome {
				if _, err := filepath.EvalSymlinks(unresolvableHome); err == nil || os.IsNotExist(err) {
					t.Skipf("this filesystem resolves %s without a permission error, so the case is not reachable here", unresolvableHome)
				}
			}
			t.Setenv("HOME", tc.home)

			contained, why, err := RealHomeContains(elsewhere)
			if err == nil {
				t.Error("no error was reported for a home the barrier could not work with")
			}
			if !contained {
				t.Errorf("%s was reported as outside a home that could not be %s", elsewhere, tc.name)
			}
			if why == "" {
				t.Error("no reason was given for the refusal")
			}

			// And the abort path that carries the error, which is a second
			// branch from the one the refusal above returns on.
			rec := &recorder{}
			MustBeOutside(rec, elsewhere)
			if len(rec.messages) == 0 {
				t.Fatal("the barrier let a directory through while it could not name the home to compare it to")
			}
			if !strings.Contains(rec.messages[0], "refusing") {
				t.Errorf("the refusal reads %q", rec.messages[0])
			}
		})
	}
}

// The three statements in homesafe.go that no test reaches, and why none of
// them can be:
//
//   - `filepath.Rel` returning an error inside PathContains. It errors when one
//     path is relative and the other absolute, and resolve returns absolute
//     paths for both.
//   - `filepath.Abs` returning an error inside resolve. It fails only when the
//     working directory cannot be read.
//   - resolve reaching the filesystem root with nothing resolved. The root
//     always resolves, so the loop leaves before it.
//
// Written down rather than covered with a fake filesystem: each is a branch
// that returns the same refusal as the ones above, and the alternative is an
// indirection in production code existing only so a test can reach it.

// TestHomeBarrier_StopsTheWriteThatWouldFollow is the reproduction of the
// defect this barrier exists for: not "does the guard notice", but "does the
// guard stop the next statement".
//
// It runs one test in a child `go test` whose HOME is a temp directory, so the
// home the child refuses to touch is a fake one and the real home is never in
// play. The child calls the barrier on a path inside that fake home and then
// writes a marker file. The assertions are that the child failed, that it said
// why, and that **the marker does not exist** — the last is the one that would
// have caught the shipped bug, where the guard reported and returned.
func TestHomeBarrier_StopsTheWriteThatWouldFollow(t *testing.T) {
	if os.Getenv(barrierChildEnv) != "" {
		t.Skip("running as the child of TestHomeBarrier_StopsTheWriteThatWouldFollow")
	}

	fakeHome := t.TempDir()
	marker := filepath.Join(t.TempDir(), "the-install-ran")

	cmd := exec.Command("go", "test", "-count=1", "-run", "^"+barrierChildTest+"$", "-v", ".")
	// HOME is replaced so the child's own idea of the home is the fake one.
	// GOCACHE is pinned to this run's cache because it otherwise derives from
	// HOME: the child would build into a cold cache under the temp directory
	// and rebuild the standard library to prove a path comparison.
	cmd.Env = append(envWithout(os.Environ(), "HOME", "GOCACHE", barrierChildEnv),
		"HOME="+fakeHome,
		"GOCACHE="+goCache(t),
		barrierChildEnv+"="+filepath.Join(fakeHome, ".cursor"),
		barrierChildMarkerEnv+"="+marker,
	)
	out, err := cmd.CombinedOutput()

	if err == nil {
		t.Errorf("the child test passed; the barrier let a path inside its own home through\n%s", out)
	}
	if !strings.Contains(string(out), "refusing") {
		t.Errorf("the child did not say it was refusing:\n%s", out)
	}
	if strings.Contains(string(out), "--- SKIP") {
		t.Errorf("the child skipped, so nothing was proved; a proof that passes by not running is the failure mode this release keeps finding\n%s", out)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatal("the statement after the barrier ran: the guard reported and returned instead of aborting, which is the defect this test exists for")
	}
}

const (
	barrierChildEnv       = "SKILL_ARCHITECT_HOME_BARRIER_DIR"
	barrierChildMarkerEnv = "SKILL_ARCHITECT_HOME_BARRIER_MARKER"
	barrierChildTest      = "TestHomeBarrierChild"
)

// TestHomeBarrierChild is the body the test above runs in a child process. It
// is skipped in every ordinary run; only the parent sets the two variables, and
// the home it is pointed at is the parent's temp directory.
func TestHomeBarrierChild(t *testing.T) {
	dir := os.Getenv(barrierChildEnv)
	if dir == "" {
		t.Skip("child of TestHomeBarrier_StopsTheWriteThatWouldFollow; not run on its own")
	}

	MustBeOutside(t, dir)

	// Unreachable when the barrier does its job. This stands for `InstallHooks`
	// — the line that, in the bug this reproduces, ran after the guard had
	// already reported the violation.
	if err := os.WriteFile(os.Getenv(barrierChildMarkerEnv), []byte("ran"), 0o600); err != nil {
		t.Fatalf("write marker: %v", err)
	}
}

// goCache is the build cache this run is using, so the child can share it.
func goCache(t *testing.T) string {
	t.Helper()
	if set := os.Getenv("GOCACHE"); set != "" {
		return set
	}
	out, err := exec.Command("go", "env", "GOCACHE").Output()
	if err != nil {
		t.Fatalf("go env GOCACHE: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// envWithout returns env with the named variables removed, so the caller's
// assignment of them is the only one the child sees.
func envWithout(env []string, names ...string) []string {
	var out []string
	for _, kv := range env {
		drop := false
		for _, name := range names {
			if strings.HasPrefix(kv, name+"=") {
				drop = true
			}
		}
		if !drop {
			out = append(out, kv)
		}
	}
	return out
}
