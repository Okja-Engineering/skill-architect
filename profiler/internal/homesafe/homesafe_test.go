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
	"errors"
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

// boundAxis is one arrangement of the protected root: the component it really
// is, whether it exists yet, and the way the destination spells that same
// component.
//
// The spelling axis is the half the previous round had no row for. Where the
// root exists, every spelling in a group names one object, so the verdict has
// to be the same for all of them — and it was not: a case-folded or NFD
// spelling resolved to a string that did not match and the write landed inside
// anyway. Where the root does not exist there is no identity on either side,
// which is the one place a name comparison is the right answer, so it is
// generated over too rather than assumed.
// mayRefuse is whether this arrangement is one the barrier is permitted to have
// no answer for, and it is exactly the arrangements in which the root does not
// exist yet. Such a root has no identity, so its name is the whole comparison,
// and this package compares a not-yet-existing name byte for byte or not at all
// — see [tailContains]. No caller ever passes such a root: the homes and scratch
// directories the barrier is asked about are all real directories. The axis is
// generated anyway, because a generator narrowed to what the callers do today is
// how the next gap gets in.
//
// Every arrangement in which the root *exists* must produce an answer, with no
// tolerance at all, and that is where every escape this repairs lived. A
// refusal is still only tolerated and never required: a wrong definite answer
// fails on these axes too, so a barrier that said "outside" for a write that
// landed inside is caught here as everywhere.
type boundAxis struct {
	component string
	exists    bool
	mayRefuse bool
	spelling  string
}

// The non-ASCII component is spelled NFC on disk and NFD in one of the
// destination spellings. APFS is normalisation-insensitive, so the two name one
// directory; a comparison of the strings says they are two.
const (
	nfcName = "café"
	nfdName = "café"
)

func boundAxes() []boundAxis {
	return []boundAxis{
		{"root", true, false, "root"},
		{"root", true, false, "ROOT"},
		{"root", true, false, "Root"},
		{nfcName, true, false, nfcName},
		{nfcName, true, false, nfdName},
		{nfcName, true, false, "CAF" + nfcName[3:]},
		{"notyet", false, true, "notyet"},
		{"notyet", false, true, "NOTYET"},
		{nfcName, false, true, nfcName},
		{nfcName, false, true, nfdName},
	}
}

// boundOperators is one line per mechanism by which a spelling can name an
// object other than the one it appears to name. `@R` is whatever the
// destination calls the root's own component, so each of these is applied to
// every axis above: adding a mechanism covers every arrangement of the root,
// and adding an arrangement covers every mechanism.
//
// Both directions are here. A path that resolves *into* the protected directory
// while its name says otherwise is the hole; one that resolves *out* of it while
// its name says inside is the false refusal, which costs a legitimate
// destination. A repair closing only the first passes half of this.
//
// Spelled with plain concatenation and never filepath.Join, because Join calls
// Clean, which folds the `..` half of these cases away before the test has run.
var boundOperators = []string{
	"@R/inner/d-exact",
	"@R",
	"@R/not-created-yet/d-new",
	"@RX/d-sibling",
	"@R-backup/d-sibling",
	"away/d-outside",
	"@R/aside/d-out-through-link",
	"@R/aside/../d-climb-out",
	"@R/rel/../d-climb-out-rel",
	"away/back/d-back-in",
	"away/back/../d-back-out",
	"away/chain/d-through-chain",
	"away/chain/../d-climb-out-of-chain",
	"@R/../away/d-climb",
	"@R/not-created-yet/../../d-fold-past-missing",
	"@R/inner/../aside/../d-interleaved",
	"away/leaflink",
	"away/leaflink/d-under-leaflink",
	"away/leaffile",
	"away/dangling",
	"@R/outleaf",
	"@R/outleaf/d-under-outleaf",
	"@R/outfile",
	"//@R///inner//d-doubled",
	"./@R/./inner/./d-dots",
	"@R/inner/d-trailing-nl\n",
	"away/d-trailing-nl\n",
	"@R/nl-out\n",
	"away/nl-in\n",
	"@R/inner/d-trailing-tab\t",
	// A non-directory mid-path. The kernel answers ENOTDIR, so no write
	// happens and the barrier has no verdict to give — which it reports rather
	// than guessing, and which the oracle records as a write it could not
	// perform.
	"@R/inner/leaf-target/d-through-a-file",
	"away/out-target/d-through-a-file",
	"away/via-nl-target",
}

// buildBoundTree lays out one instance of the tree the operators are spelled
// against. Everything is inside the test's own scratch directory — what points
// "out" of the root points at a sibling under the scratch root, never at
// anything of the reader's — so the oracle can perform the escape rather than
// reason about it. A case whose write must not be allowed to happen cannot be
// measured.
func buildBoundTree(t *testing.T, base string, axis boundAxis) {
	t.Helper()
	if err := os.RemoveAll(base); err != nil {
		t.Fatalf("clearing %q: %v", base, err)
	}
	root := base + "/" + axis.component
	mk := func(dir string) {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatalf("mkdir %q: %v", dir, err)
		}
	}
	ln := func(target, name string) {
		if err := os.Symlink(target, name); err != nil {
			t.Fatalf("symlink %q -> %q: %v", name, target, err)
		}
	}
	file := func(path string) {
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatalf("write %q: %v", path, err)
		}
	}
	mk(base + "/away")
	mk(base + "/" + axis.component + "X")
	mk(base + "/" + axis.component + "-backup")
	if !axis.exists {
		return
	}
	mk(root + "/inner")
	// Inside the root, pointing out of it — absolute and relative targets,
	// since a relative target resolves against the link's own directory and
	// that is a second thing to get wrong.
	ln(base+"/away", root+"/aside")
	ln("../away", root+"/rel")
	// Outside the root, pointing back into it: the other direction of the same
	// defect, where the verdict refuses a path that is in fact contained. And a
	// chain of two, so one hop of resolution is not mistaken for all of it.
	ln(root, base+"/away/back")
	ln(base+"/away/back", base+"/away/chain")
	// At the leaf, in both directions and in all three states a leaf link can
	// be in: to a directory, to an existing file, and dangling. A write follows
	// a leaf link, so these are writes into the root under names outside it —
	// the case the two shell copies of this walk appended as named and never
	// followed.
	ln(root+"/inner", base+"/away/leaflink")
	file(root + "/inner/leaf-target")
	ln(root+"/inner/leaf-target", base+"/away/leaffile")
	ln(root+"/inner/never-created", base+"/away/dangling")
	ln(base+"/away", root+"/outleaf")
	file(base + "/away/out-target")
	ln(base+"/away/out-target", root+"/outfile")
	// And a link with a trailing newline in its own name, at each end. A
	// resolved name carried back through a shell's command substitution loses
	// that byte, which is how the shell copies of this decided the verdict for
	// one entry and performed the write on another; Go keeps the bytes, and the
	// case is here so that the three walks are held to one invariant rather
	// than to one language's accidents.
	ln(base+"/away", root+"/nl-out\n")
	ln(root+"/inner", base+"/away/nl-in\n")
	// And a link whose *target* ends with a newline, which is a different
	// defect from a link whose name does. Go keeps the bytes and the shells
	// have to work for it, but the case is on the shared list so that the three
	// walks are held to one invariant rather than to one language's accidents.
	file(base + "/twin")
	ln(root+"/inner/leaf-target", base+"/twin\n")
	ln(base+"/twin\n", base+"/away/via-nl-target")
}

// boundLanded performs the write the way a caller would and reports where its
// bytes ended up: inside the root, outside it, or nowhere because the kernel
// refused the write altogether.
//
// Answered by identity throughout. The object is located with os.SameFile over
// a walk that does not follow symlinks, and "the object *is* the root" is asked
// with os.SameFile too. A name-matching oracle would have agreed with a
// name-matching barrier about every case in which both were wrong, which is
// exactly how this class survived a round of this very test.
func boundLanded(root, spelled string) (landed string, where string) {
	marker := []byte("where-did-this-land")
	object := spelled
	if err := os.WriteFile(spelled, marker, 0o600); err != nil {
		// A destination that is a directory, or that needs one made, is the
		// other shape a caller writes: mkdir and then write inside it.
		if err := os.MkdirAll(spelled, 0o700); err != nil {
			return "nowrite", spelled
		}
		object = spelled + "/marker"
		if err := os.WriteFile(object, marker, 0o600); err != nil {
			return "nowrite", spelled
		}
	}
	target, err := os.Stat(object)
	if err != nil {
		return "nowrite", spelled
	}
	if rootInfo, err := os.Stat(root); err == nil && os.SameFile(rootInfo, target) {
		return "inside", root
	}
	found := ""
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		if os.SameFile(fi, target) {
			found = path
		}
		return nil
	})
	if found != "" {
		return "inside", found
	}
	if real, err := filepath.EvalSymlinks(object); err == nil {
		return "outside", real
	}
	return "outside", object
}

// TestPathContains_AgreesWithWhereTheWriteLands is the invariant the table
// above cannot state, and the one the barrier has now been found failing twice.
//
// The table above is a list of spellings somebody thought of, and this test used
// to be one as well: twelve spellings, written out by hand, against a real-write
// oracle. The oracle was the right design and the table was the wrong driver.
// `..` folded lexically before symlinks are resolved is correct for every
// spelling in which no `..` crosses a link — and comparing two resolved *names*
// is correct for every spelling in which a name is an identity, which on this
// platform is every spelling somebody thinks to write down and none of the ones
// that matter. Two whole classes sat outside the twelve rows.
//
// So the case set is generated. One axis is the arrangement of the root —
// whether it exists, and how the destination spells its component — and the
// other is a list of mutation operators, one per mechanism by which a name can
// name something else. Their product is the case set, so a new mechanism is one
// line and a new arrangement is one line, and neither needs anybody to remember
// that the other exists.
//
// No case names an expected verdict. Each is performed twice over a tree built
// by one function: once as the write a caller would make, so the filesystem says
// where the bytes went, and once through the barrier. The two must agree.
//
// A spelling the kernel refuses to write on at all is skipped and counted: no
// object was created or truncated, so there is nothing for a verdict to be about.
// The count is asserted, because a generated set that had quietly become
// unperformable would otherwise look exactly like one that passed.
func TestPathContains_AgreesWithWhereTheWriteLands(t *testing.T) {
	cases, unperformable, noAnswer := 0, 0, 0
	agreedInside, agreedOutside := 0, 0
	for _, axis := range boundAxes() {
		for _, op := range boundOperators {
			axis, op := axis, op
			spelling := strings.ReplaceAll(op, "@R", axis.spelling)
			name := fmt.Sprintf("root %q exists=%v spelled %q: %q",
				axis.component, axis.exists, axis.spelling, spelling)
			cases++
			t.Run(name, func(t *testing.T) {
				// Two trees, built by one function. The oracle's write is
				// performed in one and the barrier is asked about the other,
				// because the write *changes the world the barrier is being
				// asked about*: for a root that does not exist yet, measuring
				// first would create it and the barrier would then be asked an
				// easier question than the one a caller asks.
				scratch := t.TempDir()
				oracleBase := scratch + "/oracle"
				verdictBase := scratch + "/verdict"
				buildBoundTree(t, oracleBase, axis)
				buildBoundTree(t, verdictBase, axis)

				landed, where := boundLanded(oracleBase+"/"+axis.component, oracleBase+"/"+spelling)

				root := verdictBase + "/" + axis.component
				spelled := verdictBase + "/" + spelling
				got, err := PathContains(root, spelled)

				if landed == "nowrite" {
					// No object was created or truncated, so there is no
					// landing place for a verdict to agree with. The barrier is
					// still *asked*, because a path the kernel refuses to write
					// on is a path a caller can still name: it has to come back
					// with an answer or with a refusal, and the answer it comes
					// back with is on the record.
					unperformable++
					t.Logf("the kernel performed no write on %q; the barrier answered %v, %v",
						spelled, got, err)
					return
				}
				if errors.Is(err, errUncomparableName) {
					// No answer is a refusal in every caller, so it is never
					// unsafe — but it costs a legitimate destination, so where
					// it is allowed is asserted rather than tolerated.
					noAnswer++
					if !axis.mayRefuse {
						t.Errorf("PathContains(%q, %q) reached no answer, and this arrangement is one it is meant to be able to answer: %v",
							root, spelled, err)
					}
					return
				}
				if err != nil {
					t.Fatalf("PathContains(%q, %q): %v", root, spelled, err)
				}
				if insideOrOutside(got) != landed {
					t.Errorf("PathContains(%q, %q) = %v, but a write on that spelling landed at %q, "+
						"which is %s the protected directory",
						root, spelled, got, where, landed)
					return
				}
				if landed == "inside" {
					agreedInside++
				} else {
					agreedOutside++
				}
			})
		}
	}
	if cases < 100 {
		t.Errorf("the generated bound produced only %d cases, so it is not the case set this claims to be", cases)
	}
	// Both directions have to have actually happened. This is the bound on the
	// generator rather than a proportion of unperformable cases, because a
	// proportion is a different number on a case-sensitive filesystem — where a
	// folded spelling names a directory that is not there — and a bound that has
	// to be retuned per platform is a bound nobody trusts. What matters is that
	// the sweep really saw a write land inside and agree, and really saw one
	// land outside and agree.
	if agreedInside == 0 || agreedOutside == 0 {
		t.Errorf("the generated bound agreed on %d cases that landed inside and %d that landed outside; it has to be both",
			agreedInside, agreedOutside)
	}
	t.Logf("the generated bound drove %d spellings: %d unperformable, %d with no answer, %d agreeing inside, %d agreeing outside",
		cases, unperformable, noAnswer, agreedInside, agreedOutside)
}

// TestPathContains_AChildShorterThanAParentThatDoesNotExist is the one arm of
// the name comparison the generated set above cannot reach, because every
// operator it builds names something at or below the root rather than above it.
//
// A parent that does not exist has its components compared as names. A child
// with fewer of them than the parent cannot be inside it, whatever those names
// are, and saying so before comparing anything is what keeps the comparison
// from reading off the end of the shorter list.
func TestPathContains_AChildShorterThanAParentThatDoesNotExist(t *testing.T) {
	base := t.TempDir()
	contained, err := PathContains(base+"/notyet/deeper/deepest", base+"/notyet/deeper")
	if err != nil {
		t.Fatalf("PathContains: %v", err)
	}
	if contained {
		t.Error("a path above a parent that does not exist was called inside it")
	}
}

// TestPathContains_RefusesAPathItCannotResolve covers the refusal paths the
// component walk introduced, because "a path that cannot be resolved counts as
// contained" is one of this package's three stated properties and a refusal
// nobody has watched fire is not a refusal.
//
// Both cases are answers the old implementation got from EvalSymlinks and this
// one has to produce itself: an empty path is not a path, and a symlink chain
// that does not end has to be bounded rather than followed. An unbounded walk
// inside a safety barrier is a barrier that never answers, which is worse than
// one that refuses.
func TestPathContains_RefusesAPathItCannotResolve(t *testing.T) {
	root := t.TempDir()

	t.Run("an empty path", func(t *testing.T) {
		if _, err := PathContains(root, ""); err == nil {
			t.Error("PathContains accepted an empty child path instead of refusing to answer")
		}
		if _, err := PathContains("", root); err == nil {
			t.Error("PathContains accepted an empty parent path instead of refusing to answer")
		}
	})

	t.Run("a symlink chain that does not end", func(t *testing.T) {
		// Relative targets, so the cycle is two links and not two absolute
		// paths that happen to point at each other: the walk splices a
		// relative target in against the resolved prefix, which is the arm
		// that would loop.
		if err := os.Symlink("b", filepath.Join(root, "a")); err != nil {
			t.Fatalf("symlink: %v", err)
		}
		if err := os.Symlink("a", filepath.Join(root, "b")); err != nil {
			t.Fatalf("symlink: %v", err)
		}
		// The premise: the kernel cannot resolve it either, so this is a path
		// with no answer and not one this test made up.
		if _, err := os.Stat(filepath.Join(root, "a")); err == nil {
			t.Fatal("the cycle resolved, so this case is not about a cycle")
		}
		if _, err := PathContains(root, filepath.Join(root, "a", "draft.md")); err == nil {
			t.Error("PathContains answered for a path whose symlink chain does not end; " +
				"an unresolvable path must be a refusal, not a verdict")
		}
	})
}

// TestPathContains_ResolvesARelativePathAgainstTheWorkingDirectory pins the
// other half of making a path absolute.
//
// The working directory is prepended by concatenation rather than by
// filepath.Join, for the same reason nothing else here joins: Join cleans, and
// a caller's relative path may carry the `..` this package exists to fold
// correctly. The walk then resolves the working directory's own symlinks like
// any other components — which matters on this platform, where a scratch
// directory arrives through /var and lives at /private/var.
func TestPathContains_ResolvesARelativePathAgainstTheWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	protected := filepath.Join(root, ".claude")
	if err := os.MkdirAll(filepath.Join(protected, "skills"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Symlink(filepath.Join(protected, "skills"), filepath.Join(root, "aside")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	t.Chdir(root)

	for _, tc := range []struct {
		name  string
		child string
		want  bool
	}{
		{"a relative path into it", ".claude/skills/draft.md", true},
		{"a relative path beside it", "elsewhere/draft.md", false},
		{"a relative `..` crossing a symlink back into it", "aside/../draft.md", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := PathContains(protected, tc.child)
			if err != nil {
				t.Fatalf("PathContains(%q, %q): %v", protected, tc.child, err)
			}
			if got != tc.want {
				t.Errorf("PathContains(%q, %q) = %v, want %v", protected, tc.child, got, tc.want)
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
