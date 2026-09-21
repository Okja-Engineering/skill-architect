# S9 record — `skill-rewrite` gains `-o/--output`

**Implementer** S9 · **Date** 2026-09-21 · **Branch** `fix(skill-rewrite)/0.5.0-output-flag`
**Base** `origin/main` = **`9c3ff3d`** ("0.5.0 S8: analyze and doctor, with three tiers removed")
**Head** `5e45809` · **PR** [#21](https://github.com/Okja-Engineering/skill-architect/pull/21), open, **not draft**, base `main`
**Worktree** `<scratch>/wt-s9-output-flag`. The primary tree was never touched; it is still at `0a83615`.

Three commits:

| commit | what |
|---|---|
| `296e6cf` | `spool_profile.go` → `spool_read.go` |
| `60393d7` | the mutation runner into `tests/lib/`, with its proof |
| `5e45809` | the `-o/--output` flag and its containment |

---

## 1. The containment decision, and the reasoning

**Decision: the flag needs containment, and it is a refusal of a named set, decided after resolution.**

### Why there is one at all

The drafter writes exactly one artifact. Before `-o`, the destination was
`$target_skill/REWRITE-DRAFT.md` and **the `-t` checks were the containment**:
the caller had named a directory this script verifies exists and holds a
`SKILL.md`, so the only thing it could overwrite was a previous draft of its own.
`-o` removes that bound. The bound is therefore **restated, not dropped** —
which is the whole of the argument, and it is not abstract: this repository has
already shipped a documented command that `rm -rf`'d a live skills directory,
and a test suite whose containment guard computed "no", printed FAIL, and ran
the install on the next line.

### What it may not write to

**`-o` may write anywhere the caller can write**, except four destinations.
Stating it as a refusal-set rather than a permitted-set is deliberate: a rule
that confined `-o` to the target directory would be `-o` not existing, and
"the caller owns the destination" is *mostly* true. This is the short list of
where it is not.

| refused | the harm |
|---|---|
| an existing directory | the redirect fails of its own accord and the caller is told "could not open" when the truth is they named a folder. A wrong diagnostic is a wrong answer |
| a symbolic link | `: >` follows the link and truncates the far end, so the destination's name and its landing place are two different things. The caller is told to name the file it points at |
| a `SKILL.md` | a rewrite draft is not a skill. Written over one it destroys the document it was built by reading, and leaves a draft where an agent next reads a skill |
| anywhere inside `$HOME/{.claude,.cursor,.codex,.devin,.config}` | an agent reads `~/.claude/skills/` **as skills**, so a draft dropped in one is not a stray file but a document that may be loaded as instructions |

**The `SKILL.md` refusal is an integration, not an addition.** This skill's
`Constraints` section already said *"Do not overwrite the original `SKILL.md`
without explicit approval"* and had no mechanism behind it — because with the
destination hardcoded there was no way to reach the mistake. `-o` is what makes
it reachable, so `-o` is where the existing constraint becomes enforceable. That
is the one place where the slice makes the design more coherent rather than
merely larger.

### Decided after resolution, and why that is the load-bearing part

Both properties `profiler/internal/homesafe` names, each asserted in the
direction it fails:

- **A symlink into a protected directory is outside by every string test and
  inside in fact.** `$HOME/drafts/x.md` where `drafts` links into
  `.claude/skills` is refused, and nothing in its spelling looks wrong. This is
  the same shape `tests/test_install.sh` already found for the install block
  (`~/checkout/skills`).
- **A sibling sharing a name prefix is the reverse.** `$HOME/.claude-notes/x.md`
  is **accepted**, though `$HOME/.claude` is a string prefix of it. A
  `HasPrefix` test refuses a perfectly good destination — and a refusal-only
  suite would never notice, which is why the acceptance is its own assertion
  with its own mutation (A5).

And homesafe's third property, taken with them: **a path that cannot be
resolved counts as protected.** "I could not tell where this would land" is not
"go ahead", so it is `cannot_compute DEP002` and **exit 3**, not exit 1 — the
caller's command line was not what was wrong.

`path_inside` is `"$parent"/*` on two resolved, normalised, absolute paths. The
separator is the entire difference between that and the broken prefix test, and
on resolved paths it is the same answer `filepath.Rel` gives.

### The scope of the decision — stated, because it is what a reader gets wrong

**The four refusals apply to the destination `-o` names, not to every
destination.** With no `-o` the draft still goes beside the target's `SKILL.md`,
and a target that itself lives inside `~/.claude/skills/` still drafts. The
asymmetry is in the evidence, not the spelling: the caller who named `-t` named a
directory the drafter then verified holds a `SKILL.md`; the caller who names
`-o` has asserted nothing the drafter can check. Refusing the default would have
been a silent behaviour regression beyond this slice. Asserted both ways.

### No new exit status

A destination the script will not write to is the caller naming a bad path — the
`1` this header already registers for "usage or target error". A destination it
cannot resolve is the `3` it registers for "execution error". The existing
`statuses_emitted == statuses_registered` assertion therefore still holds
without being touched, and the header's sentence describing a usage error was
extended to name the destination.

### Where the code lives

In `draft-rewrite.sh`, not in `verdict-guard.sh`. The guard's primitives compute
a verdict over a skill; this is about where this script writes. **The drafter is
the only script in either skill with a destination to decide** — the sibling
checks read a directory and print to stdout. If a second one ever takes a
destination, this moves to the guard rather than being copied. (It is also
outside my `git add` list, but the design reason is the primary one.)

### Nothing was written to a live configuration directory

The protected roots are `$HOME`-relative, which is what makes the decision
askable at all: `tests/test_rewrite.sh` redirects `HOME` into the harness scratch
directory for every one of these cases. **The `$HOME` anchoring is itself
asserted** (mutation A19: a literal `/Users/nobody` home kills 7 checks), because
a drafter that resolved a literal home would pass every refusal assertion on the
machine that wrote it and would be reading a live config directory to do it.
Nothing was installed, reconfigured, or mirrored; no PATH symlink farm was
written into.

An unset or empty `$HOME` **accepts**: there is no home, therefore no live
configuration directory for a destination to be inside. That is a different
statement from "a path that could not be resolved" and is documented as such in
the script.

---

## 2. The mutation runner's five-direction proof

`tests/lib/mutation-runner.sh`, 486 lines, a sourced library beside `harness.sh`
and `masked-path.sh`. **Seven verdicts, each with its own exit status and its own
reason on the same line as the verdict.** The invariant: *a verdict is printed in
every direction, and only KILLED is a success.* There is no path through
`mutation_run` that returns without naming what it decided, including the paths
where it decided nothing could be decided.

The five the manager named, plus the two carried from S5–S8, each observed
**twice**: once in `tests/test_harness.sh` against fabricated suite output (the
only way to reach all seven on demand), and once through a real suite run.

| direction | the release incident it is for | in `test_harness.sh` | live |
|---|---|---|---|
| **KILLED** | the desired outcome | `suite_kills` | 39 mutations, groups A/B/C/E |
| **SURVIVED** | a check green against a pattern too wide to fail; a scan that refused itself | `suite_survives` | **A8** before the test was fixed; and D4's first form |
| **DID NOT BUILD** | a mutant that would not compile, so no test ran and a failure-line search found nothing | `suite_unloadable`, `suite_go_unbuildable` | **D2** (syntax error in the suite), **E2** (`go test` → `[build failed]`) |
| **SKIPPED** | a safety barrier's own proof that skipped, having built a path from a directory that did not exist | `suite_refuses`, `suite_go_skips` | **B10**, **E1** (the by-design `TestHomeBarrierChild`) |
| **DID NOT END** | one mutation that never terminated | `suite_never_ends`, `suite_go_times_out` | **D3** (stopped at a 45s deadline) |
| **DID NOT RUN** | a suite that reached its verdict counting nothing | `suite_asserts_nothing` | **D4** |
| **NOT APPLIED** | a `sed` whose anchor had moved, so the green was the unmutated tree's | `mutant_noop` | **D1** |

Plus the control that makes the seven mean something: **the seven directions
reach seven different verdicts**, asserted as a loop over all seven with a
duplicate check. A runner that had collapsed two of them would satisfy every
single-verdict assertion above.

`test_harness.sh` gained **43 assertions** (131 → 174) and is green on both
shells.

### Three corrections the move carried

1. **The absence of a verdict is a verdict, and never a pass.** The scratch
   runners searched the output for a failure line, which asks "did anything
   fail" of a suite that may never have started. The readers here ask whether
   the suite reported a tally *at all*, and "no" is `DID NOT BUILD`. Mutation
   **B1** reverts it: 3 checks red.
2. **`[build failed]` instead of a phrase list.** S8's runner recognised a
   non-building Go mutant by searching for a dozen compiler-error phrases —
   `undefined:`, `declared and not used`, `has no field or method` — a list that
   has to grow with every error Go learns to emit and whose failure mode is to
   fall through to SURVIVED. `go test` marks it once, in a line of its own, for
   every reason a package might not build. Mutation **B8**.
3. **SKIPPED is a verdict and outranks KILLED**, rather than a note beside one.
   S8's runner printed the skip count next to the verdict, which means nobody
   ever had to say *which* skip it was and a second one would have read the same
   way. A failure count taken from a run with unexplained skips is a count of
   the checks that happened to run. The allowance is a **declared number**,
   which is what keeps the strictness usable — and **the number for
   `go test ./...` over `profiler/` is 1**, written into the runner's header with
   the name of the skip (`TestHomeBarrierChild`, the body a parent test runs in
   a child process). Mutations **B2** and **B9**.

`mutation_format` is a parameter (`harness` | `go`) rather than a grep inlined at
the classification, because the *reading* is what differs between consumers and
the seven meanings are what does not.

### The restore

Unconditional and **verified by byte comparison** on every path including the
refusals, and it is `cp` from a backup — never `checkout`, `reset`, `clean`,
`stash` or `rebase`, because the file may be one this release is holding
uncommitted and a git operation would take the rest of the tree with it. A
restore that did not restore reports `NOT RESTORED` and returns 7, which is the
one thing in the file louder than a verdict. Mutation **B6** reverts it: 4 checks
red. Every mutation run also compared `git status --porcelain` against a
baseline afterwards; **all 42 left the tree byte-identical.**

---

## 3. Mutation ledger — 42 runs, a verdict in every one

Driven by `<scratch>/s9-mutrun.sh`, which **holds no verdict logic of its own**:
it sources the committed `tests/lib/mutation-runner.sh` and does nothing but
choose the mutations. That is the slice's own dogfooding. Mutation table in
`<scratch>/s9-mut.py`, one exact find/replace per name, which exits 1 if the
anchor has moved.

| verdict | runs |
|---|---|
| KILLED | 39 |
| SURVIVED | 1 (A8, a real gap — §3.2) |
| SKIPPED | 2 (B10, E1 — both correct; §3.3) |
| DID NOT BUILD / DID NOT END / DID NOT RUN / NOT APPLIED | the four group-D/E runs, each the intended direction |

### 3.1 Group A — the `-o` flag and its containment (suite: `test_rewrite.sh`)

| | mutation | verdict |
|---|---|---|
| A1 | the `-o\|--output)` parser arm removed | KILLED 16 |
| A2 | `-o` parsed and then ignored | KILLED 7 |
| A3 | the usage line's `-o` row removed | KILLED 1 |
| A4 | `require_value` dropped from the `-o` arm | KILLED 4 |
| A5 | the separator dropped from `path_inside` — **the broken HasPrefix** | KILLED 2 |
| A6 | no containment at all | KILLED 6 |
| A7 | resolution removed; decide on the folded string | KILLED 4 |
| A8 | unresolvable reads as permitted | **SURVIVED → KILLED 3** after the test was fixed |
| A9 | the leaf-symlink refusal dropped | KILLED 2 |
| A10 | the `SKILL.md` refusal dropped | KILLED 2 |
| A11 | the directory refusal dropped | KILLED 1 |
| A12 | **the refusal reports and continues** — the `test_install.sh` defect exactly | KILLED 9 |
| A13 | the refusal exits 3 instead of 1 | KILLED 1 |
| A14 | the unresolvable destination reported as the caller's mistake (exit 1) | KILLED 2 |
| A15 | a root dropped from the script's list only | KILLED 1 |
| A16 | a root added to the document's list only | KILLED 1 |
| A17 | the `-o` invocation dropped from the documented Stage 2 block | KILLED 1 |
| A18 | `-o` writes to the named destination **and** the target directory | KILLED 13 |
| A19 | protected roots resolved from a literal home, not `$HOME` | KILLED 7 |
| A20 | "exists but cannot be entered" collapsed into "does not exist" | KILLED 1 |
| A21 | the `-L` half of that pair dropped (`-e` alone) | KILLED 1 |

### 3.2 The one survival was my own test gap — and fixing it found a real hole

**A8** removed the resolution refusal (`|| resolved="$spelled"`) and **survived
all 131 checks.** The reason is the point: with the refusal gone the decision
lets the path through, the audit runs, and then `: >` fails to open it and the
drafter reports an execution error from *there* — **the same exit 3, for a
different reason.** My assertion read only the status, so it passed by not
running, which is the third of this release's five ways and is the exact shape
S8 found four of in its own fresh tests.

Fixed by pinning the mechanism on both channels available: the diagnostic must
say the destination could not be **resolved**, and **the audit must not have
run** (it has, on the other path, because the write is the last thing the script
does and the decision is one of the first).

**Fixing the test then exposed a real defect in my own resolver.** The new
symlink-cycle case failed: `path_resolved` climbed while `! -d "$head"`, so a
component that *exists and cannot be entered* — a directory with no execute bit,
a symlink cycle — was indistinguishable from one nobody had created yet, got
climbed past, and the path came back "resolved" with the unresolvable part still
in it. **That is homesafe's second property quietly missing from the shell
version I wrote it from**, and the Go original gets it right
(`if !os.IsNotExist(err) { return "", err }`). Now `! -e && ! -L`, with `-L`
paired for the reason `tests/lib/masked-path.sh` pairs them: `-e` alone is false
for a broken symlink and for a cycle, which are exactly the entries it must stop
on. Two mutations over it (A20, A21), and a second cause reaching the same rule
(ELOOP as well as EACCES) so it is the rule being held and not one instance.

### 3.3 Two SKIPPED verdicts, both correct

- **B10** renames `mutation_report`, so `mutation_runner_ready` refuses and the
  whole block never runs. The runner reported `SKIPPED — 1 of the suite's checks
  did not run and 0 were accounted for, so its tally of 1 failed counts only the
  checks that ran`. **That is the runner catching the release's own second
  failure mode on its own file.** Re-run with the skip declared
  (`mutation_allow_skips=1`) it is `KILLED 1`, and the one red check is exactly
  `the mutation runner defined every name it exists to provide`. Both readings
  recorded; neither is forced.
- **E1** removed `LoadSessionEvents`' empty-session refusal and reported SKIPPED
  on the first run, because the Go suite carries one by-design skip. Declared, it
  is `KILLED 1 of 943`. **This is how the allowance number got measured**, and it
  is now written into the runner's header.

### 3.4 Group B — the runner itself (suite: `test_harness.sh`)

B1 KILLED 3 · B2 KILLED 6 · B3 KILLED 3 · B4 KILLED 3 · B5 KILLED 4 ·
B6 KILLED 4 · B7 KILLED 11 (the verdict reached and not printed) · B8 KILLED 2 ·
B9 KILLED 1 · B10 SKIPPED → KILLED 1 declared · B11 KILLED 2.

The B group works because the driver has the runner **sourced** while the suite
under it re-sources the **mutated** file from disk: the outer runner stays sane
while the inner one is broken.

### 3.5 Group C, D, E

- **C1** — the prefix-sibling destination respelled to be genuinely inside
  `.claude`: KILLED 2. The acceptance assertion is about what it names.
- **D1–D4** — `NOT APPLIED`, `DID NOT BUILD`, `DID NOT END`, `DID NOT RUN`
  through a real run of `test_rewrite.sh`. D4's first form inserted an early
  second `harness_summary` and the runner answered **SURVIVED**, correctly: the
  suite went on to count 136 and the reader takes the *last* tally. Recorded
  because it is the reader's rule made visible, not a miss.
- **E1–E2** — the `go` reader against an actual `go test -v` run, not a
  transcript: KILLED and DID NOT BUILD.

### 3.6 What has no mutation, and why

**The rename.** It has no behaviour and therefore no check to revert; TDD's
mechanical-refactor exception applies. Its proof is `go build`, `go vet`,
`gofmt -l`, the full race suite green either side of it, and a grep that no
reference to the old name survives anywhere in the tree (one did, in the test
file's header; it is fixed).

---

## 4. The rename

`profiler/spool_profile.go` → **`profiler/spool_read.go`**, and
`spool_profile_test.go` → **`spool_read_test.go`**.

The name is what S7 §9.3 and S8 §8.1 both proposed and both left as outside
their ruling. The file holds `LoadSessionEvents` (the read of one named session),
`forEachSpoolLine` (the walk it and `AnalyzeSpool` share), `sessionMatches`,
`spoolFileSuffix` and `ErrSessionIDRequired`. Not one of them is a profile, and
S8 is right that the file got slightly worse when the shared walk arrived in it.

The file's own header said *"Why there is no Profile in this file"*, which is the
name being wrong stated as a question. It now says what the file is, and why a
file named for a type it does not build is the same defect as a profile claiming
a capability nobody captured — one layer down, in the tooling rather than in the
data. The substance of the "no Profile here" note is kept verbatim under a
heading that no longer apologises for the filename.

Git recorded both as renames (`R093`, `R099`), so the history follows.

---

## 5. Measured counts — re-derive these, do not carry them

**The figure handed to this slice was wrong again.** The brief said 274
top-level Go tests; it is **275**, and it was 275 at `9c3ff3d` before I touched
anything. That is the **fifth** stale figure of the release, and the fourth
consecutive slice to find one. The fix is the per-package breakdown below, so the
next reader can re-derive it in one command rather than trusting a total.

```
cd profiler && go test -list '.*' ./... | grep -cE '^(Test|Example|Fuzz|Benchmark)'
```

| package | top-level test functions |
|---|---|
| `profiler` | 233 |
| `profiler/cmd` | 34 |
| `profiler/internal/homesafe` | 8 |
| **Total** | **275** |

Shell suites, re-derived at head on both shells:

| suite | at `9c3ff3d` | at `5e45809` | delta |
|---|---|---|---|
| test_harness | 131 | **174** | +43 |
| test_rewrite | 85 | **136** | +51 |
| test_f01 | 1868 | 1868 | — |
| test_f02 | 350 | 350 | — |
| test_install | 48 | 48 | — |
| test_skill | 53 | 53 | — |
| test_walk | 20 | 20 | — |
| **Total** | **2555** | **2649** | **+94** |

**2649 passed, 0 failed on bash 3.2 (3.2.57) and on bash 5. Identical on both.**
The brief's 2555 baseline was correct; I verified it at `9c3ff3d` before
starting.

`go build ./...`, `go vet ./...`, `gofmt -l .` (empty), `go test -race -count=1
./...` — all green, before and after.

---

## 6. `git add` — the list, and the one file beyond it

Ten paths, all staged by explicit name. **`git add -A` and `git commit -a` were
never run.** `tests/lib/out-of-scope-check.sh` was run against the staged index
before each of the three commits: **`out-of-scope check: clean`, exit 0** each
time.

| path | on whose list |
|---|---|
| `skills/skill-rewrite/scripts/draft-rewrite.sh` | plan §3/S9 |
| `skills/skill-rewrite/SKILL.md` | plan §3/S9 |
| `tests/test_rewrite.sh` | plan §3/S9 |
| `tests/lib/mutation-runner.sh` | manager, addition A |
| `profiler/spool_read.go` (from `spool_profile.go`) | manager, addition B |
| `profiler/spool_read_test.go` (from `spool_profile_test.go`) | manager, addition B |
| **`tests/test_harness.sh`** | **forced beyond the list — see below** |

### The one file forced beyond the list, named as required

**`tests/test_harness.sh`.** Addition A says "give it a test", and this is where
a `tests/lib/` component's proof belongs: `test_harness.sh` is by its own
opening line *"the suite for the substrate the other suites are written on"*, and
`harness.sh`, `audit-suites.sh` and `out-of-scope-check.sh` are each already
proved in it and nowhere else.

The alternative was a new `tests/test_*.sh` suite, and that is worse on two
counts: it would take the shell suite count from seven to eight, and
`test_harness.sh` derives that count from `.github/workflows/ci.yml` and
`README.md` rather than writing it down — so a new suite forces **two more
files**, one of them the CI workflow, and breaks the "all seven shell suites"
constraint in the brief. Putting the proof in `test_rewrite.sh` was the other
option and it is wrong on the merits: the runner has nothing to do with
skill-rewrite.

**The barrier's non-vacuity.** S7 and S8 each proved it live by staging an
out-of-scope path. I did not, because as of this branch the barrier is held by
`test_harness.sh` in both directions over seven one-pattern fixture
repositories plus two clean-index controls and a near-miss control — the live
demonstration is now a test, which is the better place for it. The live run over
my own index returned clean, and `test_harness.sh` proves it can say otherwise.

---

## 7. Everything the release commit (S10) needs to know

### 7.1 The four items already queued, restated with what this slice adds

1. **The removed `doctor` tiers need their own release-note line.** S8 §8.2 is
   right and nothing here changes it: a user upgrading from a preview that
   advertised `hooks`, `server_api` and `enterprise` will notice they are gone,
   and *the reason* is the point of the release. Still owed.
2. **`experiment run` shells out with no time limit.** Left deliberately rather
   than shipping a cap that does not cap (S6 §9.1, still open). **This slice has
   a bearing on it**: `tests/lib/mutation-runner.sh` now ships a portable
   bash-3.2-safe deadline (`mutation_bounded` — background job, `kill -0` poll,
   TERM then KILL, no `wait -n`, no external `timeout`) and it is proved to stop
   a non-terminating child at its deadline. It is a *test* library, so nothing in
   `experiment run` may source it — but if 0.6.0 decides to cap the step, the
   mechanism has now been written and proved once in this repository and the Go
   side has `exec.CommandContext`. **S10 should still state plainly that there is
   no cap**, and may note the deadline the test library gained.
3. **The skill metadata versions are still undecided.** Untouched.
   `skills/skill-rewrite/SKILL.md` still carries `metadata.version: "0.1.0"`, and
   this slice changed that file substantially without touching that field. It is
   the user's decision and it is now slightly more conspicuous: a skill whose
   documented interface gained a flag still declares 0.1.0.
4. **`AppendSpool` has no locking** (S7 §9.5). Still wants a line. Untouched.

### 7.2 New for the changelog, from this slice

- **`skill-rewrite`'s `draft-rewrite.sh` takes `-o|--output`.** The
  user-visible sentence is the flag plus its bound: *it writes anywhere you can
  write, except a directory, a symlink, a `SKILL.md`, or inside
  `$HOME/{.claude,.cursor,.codex,.devin,.config}`, and it decides that after
  resolving the path.* The four refusals are worth naming in the notes rather
  than leaving in the SKILL.md: a caller scripting `-o` will hit one.
- **`tests/lib/mutation-runner.sh`** is new committed surface in the test
  library. Not user-facing; worth one line in the changelog's verification
  section if there is one, because it is the mechanism five slices of this
  release each rebuilt by hand.
- **`profiler/spool_profile.go` is now `spool_read.go`.** Internal, unexported,
  same package, no API change — **nothing shipped the old filename**, so no
  migration note is needed. Mention only if the notes track file moves.

### 7.3 Version surfaces — confirmed untouched

| surface | value at head | who owns it |
|---|---|---|
| `profiler/types.go` `AdapterVersion` | `"0.5.0"` | set once in S3; untouched here |
| `ProfileSchema` | `"skill-architect/profile/v1"` | D5; untouched |
| the five plugin manifests | `0.4.3` | **S10's, per the brief** |
| `tests/test_skill.sh` version asserts | `0.4.3` | S10's; untouched |
| skill `metadata.version` fields | `0.1.0` etc. | **open user decision**; left alone |

I touched no version surface. `test_skill.sh` is unmodified in this branch.

### 7.4 State S10 should verify rather than assume

- `origin/main` was `9c3ff3d` when this branch was cut. PR #21 is the only open
  PR I created; check `gh pr list --state open` before the release commit.
- The primary tree is still at `0a83615` with its ~2,900 lines of unlanded work
  intact. **Nothing in this slice touched it**, and no `checkout`, `reset`,
  `clean`, `stash` or `rebase` was run in any tree.
- The D3 snapshot was not needed: none of this slice's content came from the
  uncommitted working tree. The `-o` flag was written fresh against post-S8
  `main`, as §5.4 of the plan requires (the wip version of `draft-rewrite.sh` is
  the vendoring fork and is discarded).
- The suite total to quote is **2649 on both shells**, and the Go total is
  **275** with the per-package breakdown in §5. **Re-derive both.** Five figures
  have now gone stale in this release; quoting a total without the command that
  produced it is how.

---

## 8. Open, and not mine

### 8.1 A second script with a destination would move the containment. **[note]**

`output_is_permitted`, `path_resolved`, `path_absolute` and `path_inside` live in
`draft-rewrite.sh` because it is the only script in either skill that writes to a
caller-named path. If a second one gains one, they belong in
`skills/skill-audit/scripts/verdict-guard.sh` — which is not on this slice's
`git add` list, and which is the right home only once there are two callers.
Written into the script as a comment so the next person does not have to
rediscover the reasoning.

### 8.2 The shell containment primitive now exists three times. **[decide — 0.6.0]**

`tests/test_install.sh` has `path_normalized` / `path_resolved` /
`path_resolves_inside`; `profiler/internal/homesafe` has the Go original; and
`draft-rewrite.sh` now has a third. **They are not copies of one library and
cannot be** — one is a test suite, one is a Go package, one is a shipped script
that may only depend on its sibling skill — but they are three implementations of
one decision, and **the shell one I wrote was already weaker than the Go one it
was read from** (§3.2). That is the drift this release keeps finding, in a form
no `tests/lib/` file can fix. Worth a 0.6.0 decision; I did not widen the slice
to chase it.

### 8.3 The `-t --target -a --audit -o --output` flag loop has one vacuous half. **[note]**

`assert "$flag with no value says which option is missing a value"` passes on an
*unknown* option too, because "Unknown option: -o" also contains `-o`. That
shape predates this slice and applies to all six flags. It is not vacuous for the
distinction that matters — the "names the option rather than a shell positional"
assertion is what pins the `require_value` path, and mutation A4 kills through
it — but the label promises slightly more than its command decides. Named, not
changed: tightening it touches assertions for `-t` and `-a` that are not mine.

### 8.4 `mutation_read_go` requires `-v`. **[documented, not enforced]**

Without it a passing package prints one `ok` line and the reader has nothing to
count, which lands as `DID NOT RUN` rather than as a false `SURVIVED`. Failing
closed was the choice and it is documented in the function. It is not *enforced*
— the runner cannot see the caller's flags. A future slice could have the reader
refuse output with no `--- ` lines at all and say why, which is one more verdict
and nobody asked.

### 8.5 The runner reads the *last* tally a suite printed. **[recorded]**

D4's first form proved it: a suite that printed an early summary and then went on
to count 136 reports SURVIVED. That is the right rule — the last verdict is the
one the suite reached — but it is a choice, and it is recorded here because it
was discovered rather than designed.

---

## 9. Not done, and why

| not done | why |
|---|---|
| Merging or tagging | The user merges. Nothing tagged. Do not merge. |
| Touching any version surface | §7.3. The manifests are S10's; the skill metadata versions are the user's. |
| `tests/test_walk.sh` claim-walk entries | Not on the plan's S9 list and the plan says each slice adds its own; the `-o` claims are held by `test_rewrite.sh`'s two-sided comparisons, which is where this skill's claims live. |
| A `README.md` or `CHANGELOG.md` line for the flag | S10's, per the plan. The content S10 needs is in §7.2. |
| Moving the containment into `verdict-guard.sh` | §8.1 — one caller, and the file is outside the `git add` list. |
| Unifying the three shell/Go containment implementations | §8.2 — larger than this slice; surfaced rather than started. |
| A live staging test of the out-of-scope barrier | §6 — it is now a test, in both directions, over nine fixture repositories. |
| Resolving a leaf symlink rather than refusing it | The refusal closes the same hole with less machinery and no new required tool (`readlink` would have to join the SKILL.md's `Required tools:` line). Documented in the script. |
| Reading the PR's own review findings | Ship-gate work, and the `bug-fixer`'s. Handed off. |
