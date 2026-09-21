# S6 record — the experiment, and the refusals that belong where the pair is made

**Implementer** senior-implementer · **Date** 2026-09-20 · **Base** `origin/main` = `e26b2a5`
**Branch** `feat(profiler)/0.5.0-experiment` · **Commit** `4a39d01` · **PR** [#18](https://github.com/Okja-Engineering/skill-architect/pull/18) (open, not draft, base `main`)
**Worktree** `<scratch>/wt-s6-experiment` (session scratchpad)
**Status** delivered, not merged. The user merges. Nothing tagged.

---

## 1. The verification gap, measured rather than assumed

The plan said there is no `experiment_test.go` anywhere and that whatever
coverage exists is buried in `profiler_test.go`'s 766 added lines, unread. Both
halves are true, and the measurement is worse than the plan's phrasing implies.

### 1.1 What coverage actually existed

**On `main`: none.** There is no `experiment.go` on `main` at all, so there was
nothing to cover.

**In the snapshot draft: five tests, and they are all happy path.** `grep -c -i
experiment` over every snapshot test file: 18 hits, all in `profiler_test.go`,
lines 985–1219 — `TestNormalizeDesign_AppliesDefaults`, `TestGeneratePlan`,
`TestLoadExperimentDesign`, `TestRunPlan`, `TestLoadPlan`.

I did not stop at reading them. The draft's `experiment.go`, `compare.go` and
`types.go` were copied into a scratch module (the rest of the snapshot package
does not compile — plan §1.3) and the five tests were run against them under
`-coverprofile`:

| | statements | covered | |
|---|---|---|---|
| draft `experiment.go` under the draft's own 5 tests | 130 | 100 | **76.9%** |
| this slice's `experiment.go` under this slice's tests | 139 | 139 | **100.0%** |

**All 30 uncovered blocks in the draft were refusals and error paths** — every
`is required`, the schema checks, the ordering/analysis/stopping refusals, every
`return nil, err` in `RunPlan`, and the step-failure message. Not one refusal in
the draft was ever executed by a test. The 76.9% is entirely the happy path.

Two further points the number alone does not show:

- `safeName` and `stepForCondition` show **100%** in the draft and assert
  nothing about what they do: no test there pins `$TASK` expansion to a value,
  and none pins that a task family cannot escape the output directory.
- **The draft's `TestRunPlan` cannot pass against `main`.** Its fixtures carry
  no `capability.adapter_version`, and S5's contract refuses two empty versions;
  it also type-asserts `Delta.(TokenCounts)` and reads `delta.Input` as an
  `int`, which S3 made a `*int`. "The wip branch had tests" would have been
  proof of nothing even if the tests had been ported unchanged.

### 1.2 The coverage figure for `profiler/cmd`, and S5's four lines

S5 was right and the fix is better than it expected. `go test -cover` **sets
`GOCOVERDIR` for the test process**; plain `go test` does not. So the four lines
in `buildProfiler` need no external harness at all:

```go
if os.Getenv("GOCOVERDIR") != "" {
    args = append(args, "-cover", "-coverpkg=…/profiler/cmd")
}
```

`go test -cover ./cmd` now instruments the subprocess and merges its counters
into the same run. Plain `go test` is untouched and pays nothing.

| `profiler/cmd` | base `e26b2a5` | head |
|---|---|---|
| `go test -cover`, as reported | 8.5% | **94.5%** |
| the honest figure (subprocess counted) | 91.5% | **94.5%** |

Base 91.5% reproduces S5's head exactly, re-derived in a scratch copy before any
edit. **L21's 12.6% in the deferred ledger is still wrong and should be
corrected to point at this instead.**

---

## 2. What landed

7 files, 798 insertions, 49 deletions. Two are new.

| File | What |
|---|---|
| `profiler/experiment.go` | **new** — design, plan, run, and the refusals |
| `profiler/experiment_test.go` | **new** — 28 tests over the experiment contract |
| `profiler/cmd/main.go` | `experiment {design,plan,run}`, the exit contract, `printJSON` |
| `profiler/cmd/main_test.go` | the CLI half; the `-cover` build; the derivation, scoped |
| `docs/profiler-spec.md` | `## Experiment contract` |
| `README.md` | "Running a paired experiment", and one false claim corrected |
| `.out-of-scope.md` | **outside the plan's `git add` list** — see §7.1 |

`git add` was those seven, named. No `git add -A`, no `commit -a`.

---

## 3. The design: three refusals at three times

Because the experiment is what *makes* the pair, it is the only layer that can
refuse to make a bad one.

### 3.1 The design refuses a declaration the runner cannot honour

One principle, one place (`designRefusal`, a table of rules rather than a
growing `if` chain): **a design may declare only what the runner actually
does.** `ordering` accepts only `blocked`, `analysis_method` only `difference`,
`stopping_rule` only `fixed`. The draft accepted `random` and attached a note
saying it would be run blocked — which is a request the tool ignores, the exact
failure this release has already found three times in its own prose.

Also required: `name`, non-empty `task_families` with no empty entries, both
commands, both `snapshot_hash`es and `skill_dir`s, and **`$PROFILE` in each
command** — the plan chooses where each profile goes, so a command that ignores
it writes somewhere nothing reads.

### 3.2 The run refuses a profile that is not the one the step declared

- The profile file is **stamped before and after** the command. A command that
  exits 0 and writes nothing is refused: the file at that path is then either
  absent or, worse, an earlier run's, and it would be compared and reported as
  this run's numbers. Stamping rather than deleting, because deleting a file at
  a path the caller named is not this tool's to do.
- The profile is read with `LoadProfile` and checked against the step's declared
  `harness`, `snapshot_hash` and `skill_dir`, so the plan's declaration is a
  promise that is kept rather than metadata nobody reads.

### 3.3 What is left is the comparison's refusal, and it is not restated

`RunPlan` calls `CompareProfiles`. There is one copy of the adapter-version rule
and this is not it.

---

## 4. Can `run` produce a pair `compare` would refuse? Yes — and here is what happens

**Yes, and it is unavoidable.** The two capture commands are the caller's, and
nothing can promise they are the same build of the profiler: a fresh candidate
against a baseline captured months ago is exactly the pair S5 refuses. The
"neither names a version" half is the one a first-time user meets, because any
capture command that is not this profiler writes profiles with no
`capability.adapter_version` at all.

**Handled deliberately, not silently:**

- The refused pair produces **a result, not an error**: the run happened, and
  the refusal is the answer. `comparable: false`, the refusal in the run's
  comparison, and **no delta computed anywhere** (S5's shape, inherited by
  deferring to it).
- The refusal is printed **on stderr**, one line per refused run, naming the
  task family and repetition.
- `experiment run` exits **2** — the same meaning `compare` gives it, through
  one shared `exitForComparable`.
- Aggregation is strict: the experiment is comparable only when **every** run
  compared something. A refused run does not abort the ones after it — each run
  is an independent observation — but it does make the experiment's verdict
  false. M33 proves the abort-instead shape is not what landed.

Live, against the built binary (§8.4): a `0.4.1` baseline against a `0.5.0`
candidate gives exit 2, the refusal on stderr, and `-400` appears **nowhere** in
the result document.

**What `run` cannot produce**, by construction: a cross-harness pair (§5), a
pair where one side is a leftover file (§3.2), a pair naming a different
snapshot or skill from the one declared (§3.2).

---

## 5. Cross-harness pairing — the open question S5 left, and it is mine

**It is mine, and the answer is: `experiment` refuses it; `compare` keeps the
note.** `Condition` carries a `Harness` per side, so a design *can* name two —
which is precisely the condition S5 §7.3 said stops being free.

`NormalizeDesign` refuses a design whose two conditions name different
harnesses, before the registration check, so the refusal survives the day a
second adapter lands and both names become valid. The harness must also be one
the **registry** answers to, asked of `HarnessNames()` rather than written down,
so the set a design may name cannot drift from the set the CLI can dispatch.

Why the asymmetry with `compare` is right rather than an inconsistency:

- `compare` is *handed* a pair by a human who may have a reason to cross
  harnesses and is entitled to the numbers with a note.
- `experiment` is *deciding what to capture*. A tool that chooses to measure
  with two different meters and then subtracts has produced the number itself.

**Consequence for S5 §7.3:** the cross-harness question is now closed for
anything this release produces, because the only path from this repo's own tools
to a pair is `experiment`, and it will not make one. The slice that registers a
second adapter still has to decide whether `compare` should refuse a
hand-assembled cross-harness pair — but it is no longer load-bearing.

**Reachability of the refusal today:** the same-harness rule is checked before
registration precisely so it is testable while one adapter ships
(`{claude_code, cursor}` is refused for the pairing, not for the name). The
harness **default** — take the registered adapter only when there is exactly one
— is a pure function tested in both directions, because the two-adapter branch
is unreachable through `NormalizeDesign` today and an unreachable branch nobody
asserts is a branch nobody has checked.

---

## 6. Mutation proofs — 37, with a verdict printed in every direction

S5's rule, carried: **any check whose pass signal is the absence of output can
pass by not running.** The runner prints one of four verdicts — `NOT APPLIED`,
`DID NOT BUILD`, `DID NOT RUN`, `SURVIVED`, `KILLED n` — and **was proved in
three directions before a single result from it was trusted**:

| harness proof | expected | got |
|---|---|---|
| H1 a mutation that changes behaviour | KILLED | **KILLED — 6** |
| H2 a reworded comment | SURVIVED | **SURVIVED** |
| H3 an undefined symbol | DID NOT BUILD | **DID NOT BUILD** |

Restores were `cp` from a scratch backup. **No `checkout`, `reset`, `clean`,
`stash` or `rebase` was run anywhere**, and all four files were diffed against
the backup afterwards and are byte-identical.

| # | Mutation | Result |
|---|---|---|
| M1 | the same-harness refusal removed | KILLED — 3 |
| M2 | the stopping-rule refusal removed | KILLED — 7, incl. the CLI |
| M3 | the analysis-method refusal removed | KILLED |
| M4 | the ordering refusal removed | KILLED |
| M5 | the `$PROFILE` requirement removed | KILLED |
| M6 | the registered-harness check removed | KILLED |
| M7 | an empty task family accepted | KILLED |
| M8 | `defaultHarness` guesses when two are registered | KILLED |
| M9 | `LoadExperimentDesign` skips normalization | KILLED — 5 |
| M10 | `LoadExperimentDesign` skips the schema check | KILLED — 3 |
| M11 | `LoadPlan` accepts a plan with no runs | KILLED |
| **M12** | **`comparable()` drops the no-runs guard (vacuous truth)** | **KILLED** |
| M13 | `comparable()` means *any* run rather than *every* run | KILLED |
| M14 | the step's write check removed | **DID NOT BUILD** — see below |
| M14b | the same, written so it compiles | KILLED — the stale-file subtest |
| M15 | the write check asks only whether the file exists | **DID NOT BUILD** |
| M15b | the same, written so it compiles | KILLED — the stale-file subtest |
| M16 | the declared-identity check removed | KILLED — 4 |
| M17 | the identity check covers only the harness | KILLED — 3 |
| M18 | the plan expands the planning machine's environment | KILLED — 21 |
| M19 | `safeName` keeps the separators and the dot | KILLED — the traversal |
| M20 | the step's stdout goes to the result document | KILLED |
| M21 | the output directory is not created | KILLED — 8 |
| M22 | `experiment` dispatched and left out of the help | KILLED |
| M23 | `experiment` in the help and not dispatched | KILLED — 19 |
| M24 | a subcommand in the experiment help that is not dispatched | KILLED |
| M25 | a subcommand dispatched and left out of the experiment help | KILLED |
| M26 | `experiment run` always exits 0 | KILLED — 3 |
| M27 | the per-run refusal is no longer said on stderr | KILLED |
| **M28** | **`run` accepts both `--design` and `--plan`** | **SURVIVED — a real gap. See §6.2** |
| M28b | the same, after the test was repaired | KILLED |
| M29 | a failed experiment prints an empty document anyway | KILLED — 3 |
| M30 | the profile path drops the condition label | KILLED — 4 |
| M31 | the profile path drops the repetition | KILLED |
| M32 | `LoadPlan` stops refusing two steps that write one path | KILLED |
| M33 | a refused run aborts the experiment | KILLED — 6 |
| M34 | `safeName` may return an empty path component | KILLED |
| M35 | an output directory that cannot be created is ignored | KILLED |
| **C1** | **control:** the dispatcher derivation names a missing function | KILLED by the guard |
| **C2** | **control:** the help derivation reads a header that is not there | KILLED by the guard |

### 6.1 The vacuity trap fired again, and the runner said so

**M14 and M15 did not build.** Both make `before` (and for M14 `after`) an
unused variable, which is the *exact* mechanism that gave S5 a false green.
Because the verdict is printed in every direction, they read `DID NOT BUILD — it
was never executed, so it proves nothing` rather than silence, and were re-run as
M14b/M15b in a form that compiles. Both then killed — and only the *stale-file*
subtest, because a step that writes nothing at all is caught downstream by
`LoadProfile`. The two subtests are not redundant; one of them is the only thing
holding the stamp comparison.

### 6.2 M28 found a vacuous test of mine, before the PR did

`run` refusing `--design` and `--plan` together was tested with the **design
file passed to both flags**. With the refusal mutated away, `LoadPlan` rejected
the design on its schema and the command exited 1 anyway — so the case passed
for a reason that had nothing to do with the rule under test. Repaired by
generating a **valid plan** alongside the valid design, so accepting both would
actually run something; M28b then killed it.

That is the fifth slice in a row to catch a vacuous check in its own work, and
the first to catch one because a *mutation survived* rather than because the
check was silent.

---

## 7. Deviations, and the one file outside the plan's list

### 7.1 `.out-of-scope.md` is staged, and the plan did not list it. **[flag]**

`.out-of-scope.md:8` said *"scheduling and executing the paired runs remains the
caller's job."* This commit makes that false: `experiment plan` schedules and
`experiment run` executes. `README.md:756` carried the same sentence, and
`README.md` **is** in the list — so leaving the other would have shipped a
commit whose two boundary documents contradict each other.

Precedent: S1 added `.out-of-scope.md` to S5's list for exactly this reason, and
S3 §7 records files forced outside the plan's list with the justification.

**Both now say the same thing**: what remains out of scope is *driving the
agent* — the command that runs it and writes each profile is still the
caller's, and the plugin never launches a model or spends tokens of its own.
That is true and stays true. The change is one bullet and one README line; if
the manager wants it out, it is separable.

### 7.2 `design` is a third subcommand the plan did not name, and I chose its meaning

The plan says "the `experiment` subcommand with `plan` and `run`"; the brief
says "design, plan and run". The snapshot has no `design` command, so its
meaning was mine to choose. **`experiment design --file d.json` prints the
design as the runner will read it** — defaults applied — or exits 1 with the
refusal. It is the step at which a refusal is free, and it makes the design
rules reachable on their own rather than only through `plan`.

The alternative reading — `design` scaffolds a template — is served instead by
the worked design document in the README, which costs no new surface. **If the
manager meant the scaffold, this is a small change and I should be told.**

### 7.3 Ported from the snapshot: nine changes, by hand

Hash-checked before reading: `profiler/experiment.go` =
`e23575d3…44f6`, matches the manifest. **Nothing was copied.** The warning held
for the fifth time.

| What the draft does | Why it could not be ported | What landed |
|---|---|---|
| `Budget{MaxTokens, MaxTimeMs}` | **nothing enforces either.** A declared cap that nothing applies is the fourth thing in this repo advertising what it cannot deliver | **dropped.** A real budget belongs to the slice that can enforce it |
| `QualityTolerance`, `EfficiencyTarget` | same: nothing judges a result against them | **dropped** |
| `ordering: random` accepted with a note | a request the runner ignores | **refused** |
| `analysis_method: ratio` accepted, differences computed | same, and worse: the number is wrong rather than absent | **refused** |
| `os.ExpandEnv` over the command at plan time | a plan is written down and run later, possibly elsewhere; and an unset variable becomes `""` at plan time rather than the shell's problem at run time | **dropped.** `sh` expands what is left |
| `Steps []ExperimentStep` with `[0]`/`[1]` and a `len < 2` check | a pair is not a list; the check would ride into every consumer, and a plan failing it mid-execution is already half run | `Baseline`/`Candidate` as named fields. The illegal state is unrepresentable and the check is gone |
| `RunPlan(plan, outputDir)` writing a comparison file per run, joining `outputDir` with a path that **already contains** `design.OutputDir` | a double-prefix defect, and `--output` was dropped by S5 for the same reason | one result document on stdout; a redirect stores it |
| `[]ComparisonReport` as the result | the array index is the only run identity, and a stored result has to stay interpretable | `ExperimentResult` with a schema, the design, and per-run identity and profile paths |
| `cmdExperiment` with `flag.ExitOnError` | exits **2** on a flag typo, and 2 means "nothing compared" | `flag.ContinueOnError` through `parseFlags`, as S5 established |
| `safeName` replacing only `' '` and `'/'` | `".."` survives it, so a task family could climb out of `output_dir` | everything but `[A-Za-z0-9_-]` replaced: no `.` left to make a `..` from |

### 7.4 Integrations rather than additions, in the files I touched

- **`dispatchedCommands` was file-wide and would have broken.** It walks *every*
  `CaseClause` in `main.go`, so `experiment`'s three subcommand literals would
  have been read as three top-level commands. Narrowed to the named function's
  body (`caseLiteralsIn(t, "main")`), which is what "the dispatcher" meant all
  along, and reused for the new sub-dispatcher check. C1 proves the guard.
- **`commandsInHelp` → `commandsUnder(help, header)`**, one derivation for both
  help blocks. C2 proves the guard.
- **`printJSON`** replaces the fourth copy of marshal-print-or-die; `probe`,
  `capture` and `compare` now share it rather than the new command adding a
  fifth.
- **`exitForComparable`** is the one place the meaning of exit 2 lives;
  `compareExitCode` and `experimentExitCode` both answer to it.
- **`writeProfileFile` → `writeJSONFile`**, and `runCompare` now goes through a
  shared `runCLI`.

---

## 8. Verification — measured, not carried

### 8.1 Shell suites, both shells

| suite | base | head, bash 3.2.57 | head, bash 5.3.15 |
|---|---|---|---|
| test_f01 | 1868 | 1868 | 1868 |
| test_f02 | 350 | 350 | 350 |
| test_harness | 131 | 131 | 131 |
| test_install | 48 | 48 | 48 |
| test_rewrite | 85 | 85 | 85 |
| test_skill | 53 | 53 | 53 |
| test_walk | 20 | 20 | 20 |
| **total** | **2555** | **2555 passed, 0 failed** | **2555 passed, 0 failed** |

Unchanged, and expected: this slice touches no shell surface, and no suite reads
the prose that changed (`test_harness.sh` names `README.md` and
`.out-of-scope.md` only in its stageability list; `test_f01`'s README assertions
are about jq tooling; `test_install`'s are about install paths).

### 8.2 Go, in `profiler/`

| gate | base `e26b2a5` | head |
|---|---|---|
| `gofmt -l .` | empty | empty |
| `go build ./...` | clean | clean |
| `go vet ./...` | clean | clean |
| `go test -race -count=1 ./...` | ok | **ok**, both packages |
| top-level tests | 144 | **175** (+31) |
| PASS lines incl. subtests | 662 | **752** (+90) |
| failures | 0 | **0** |
| `profiler` coverage | 96.7% | **97.8%** |
| `profiler/cmd`, honest | 91.5% | **94.5%** |
| `experiment.go` per-file | — | **100.0% (139/139)** |

The +31: 28 in `experiment_test.go` and 3 in `cmd/main_test.go` (the two
`experiment run` command tests and the sub-dispatcher derivation); the rest of
the new cases are rows on tables that already existed — 12 usage errors, 6 help
paths — which is why subtests rose by 90.

### 8.3 Test isolation

A stray `profiler/cmd/baseline-fixture.json` appeared in the tree at 21:18,
during the RED phase. **Chased, not ignored:** it does not reproduce — two full
`go test ./...` runs and a `-race` run leave `git status` showing only the two
intended new files — and every write in both test files goes to a path derived
from `t.TempDir()`. It was an intermediate edition of the test file, deleted.
Nothing was installed, nothing reconfigured, no PATH mirror built, no live
config directory written.

### 8.4 Live, against the built binary

| run | result |
|---|---|
| a clean experiment | exit **0**, `comparable: true`, `delta {input: -400}` |
| a baseline captured by `0.4.1` | exit **2**, refusal on stdout **and** stderr, `-400` appears **0 times** in the document |
| a step that writes nothing, over a leftover profile from the previous run | exit **1**, nothing on stdout, and the error names the path and says the file there would be an earlier run's |

### 8.5 Hygiene

- `git add` was **seven named paths**. No `git add -A`, no `commit -a`.
- `tests/lib/out-of-scope-check.sh` over the staged index: `out-of-scope check:
  clean`, exit 0 — the committed script. **Proved non-vacuous against this
  index**: staging `skillgate/probe.md` gave `BLOCKED: out-of-scope path staged`,
  exit 1, naming it; unstaged and deleted, the check returned clean.
- Author and committer `imagineux <imagineux@gmail.com>`. AI-attribution grep
  over the whole commit object: **0 matches**.
- **No version surface touched.** `AdapterVersion` stays `0.5.0`,
  `ProfileSchema` stays `skill-architect/profile/v1`, the five plugin manifests
  stay `0.4.3`. The three new schemas are new documents, not a bump of an
  existing one.
- No `checkout`, `reset`, `clean`, `stash` or `rebase` anywhere, including
  across 37 mutation runs.
- The snapshot was read-only and hash-checked; nothing written to it.

---

## 9. Findings — what should change before slice seven

### 9.1 `run` executes arbitrary shell with no time limit. **[decide — S7 or the release]**

`RunPlan` shells out to the caller's command with no timeout, and an experiment
is a loop of them. The draft declared a `Budget{MaxTokens, MaxTimeMs}` that
nothing read; I dropped it rather than ship a cap that does not cap (§7.3).
**The honest version of it is a per-step wall-clock timeout**, which is ~5 lines
on `exec.CommandContext` and is genuinely enforceable — unlike a token cap,
which cannot prevent the spend it names because the count is only known after
the step. Named rather than silently added: it is new behaviour and the slice
was not asked for it.

### 9.2 A stale profile is caught by its stamp, not by its provenance. **[note]**

The before/after stamp catches a step that did not write, and the declared-
identity check catches a profile from the other condition or another snapshot.
What neither catches is a *byte-identical* re-run of the same design into the
same output directory where the command silently fails to update the file and
the filesystem's mtime resolution hides it. The window is nanoseconds wide on
APFS and ext4. Closing it properly means the capture writing something unique
per run (a run id in the profile), which is a profile-format change and not
this slice's.

### 9.3 L21's coverage figure is still wrong in the ledger. **[decide — carries from S5 §7.2]**

12.6% is an artifact twice over. The fix landed here (§1.2): `go test -cover`
now reports the honest 94.5%. **The ledger entry should be replaced with a
pointer to `buildProfiler`'s comment**, or S8 will chase the same ghost. S8 adds
two subcommands and would have watched the number fall again.

### 9.4 Cross-harness is decided for anything this repo produces. **[closes S5 §7.3 halfway]**

§5. The second-adapter slice still owns `compare`'s note-vs-refusal question for
a hand-assembled pair, but it is no longer the only thing standing between two
meters and a subtraction.

### 9.5 The three "tracked for 0.5.0" deferrals and `README.md:28`. **[unchanged, S10's]**

Untouched by this slice; S5 §7.4's flag stands. **One addition for S10's prose
gate:** `README.md:28`'s signal summary and the harness table are now two
releases stale, and the README's "What this plugin does not do" list has moved
once (§7.1) — the release commit should re-read all three together.

### 9.6 Smaller notes

- `ExperimentResult` has no `refusals` summary field; the CLI derives the stderr
  lines by walking `Runs`. One place, and a summary field would be a second
  thing to keep in agreement with the first.
- `GeneratePlan` blocks by task family because that is the only ordering
  implemented. The refusal of `random` is written so that implementing it later
  is deleting a rule, not changing a meaning.
- `experiment` has no `--output` flag, by the same argument S5 used for
  `compare`: one document on stdout, and a redirect stores it. The per-run
  comparison files the draft wrote are gone with it.
- The result embeds the whole normalized design, including the commands. That is
  deliberate — a stored result must say what it ran — and it means the commands
  (and any secret spelled literally in one) travel with the document. Worth a
  line in the release notes if anything renders results publicly.

---

## 10. PR

[#18](https://github.com/Okja-Engineering/skill-architect/pull/18), open, **not
draft**, base `main`, head `feat(profiler)/0.5.0-experiment` at `4a39d01`.
CI: `gh pr checks 18` → **pass, 2m58s** (ubuntu-latest, bash 5, Linux git).
AI-attribution grep over the live PR title and body: **0 matches**.
The primary working tree was re-checked after the push: still `0a83615`, 47
`git status --porcelain` lines — the figure S3, S4 and S5 recorded.

---

## 11. Not done, and why

| Not done | Why |
|---|---|
| Merging or tagging | The user merges. Nothing tagged. |
| Touching any version surface | §8.5. S3 set `AdapterVersion`; the manifests are S10's. |
| A budget or a step timeout | §9.1 — named, not silently added or silently dropped. |
| `ordering: random` | Refused rather than implemented. Randomising is a decision about the experiment's statistics, not a port. |
| Refusing a cross-`harness` pair in `compare` | §5 — `experiment` cannot make one; the hand-assembled case is still the second adapter's call. |
| `profiler analyze` / `doctor` / the spool | S7 and S8. |
| `CHANGELOG.md` / `RELEASE_NOTES.md` | S10's. |
| Rebuilding the snapshot | Frozen, read-only, hash-checked, not written to. |
