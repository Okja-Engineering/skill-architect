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
