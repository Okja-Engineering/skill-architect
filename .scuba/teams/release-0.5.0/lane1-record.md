# Lane 1 — CODE. Record. PR #22, `release/0.5.0`

Base `9c8ba53`. Head `d904c30`. 11 commits, author `imagineux <imagineux@gmail.com>`
on every one, no `Co-Authored-By` and no "Generated with" line
(`git log 9c8ba53..HEAD --format='%B' | grep -inE '^Co-Authored-By|Generated with \[Claude|noreply@anthropic'` → NONE).

Worktree: `…/scratchpad/lane1-code`, `git worktree add --force … release/0.5.0`.
Nothing was written in the primary tree except this file. No `checkout`/`reset`/
`clean`/`stash` in any tree but my own and one throwaway detached worktree
(`scratchpad/redtree`, created at my HEAD for the two RED checks that require
editing CHANGELOG.md/RELEASE_NOTES.md, then removed — those two files are
Lane 2's and I did not modify them in my own tree at any point).

**Lane 2's files are untouched**: `git diff --name-only 9c8ba53..HEAD` lists no
`CHANGELOG.md`, `RELEASE_NOTES.md`, `README.md` or `docs/profiler-spec.md`.
21 files, +2280 −255.

---

## Verification, at head

| check | command | result |
|---|---|---|
| shell, bash 3.2.57 | `for s in tests/test_*.sh; do /bin/bash "$s"; done` | **2741 passed, 0 failed** |
| shell, bash 5.3.15 | same with `/opt/homebrew/bin/bash` | **2741 passed, 0 failed**, identical per-suite |
| go build | `cd profiler && go build ./...` | clean |
| go vet | `go vet ./...` | clean |
| gofmt | `gofmt -l .` | no output |
| go test | `go test ./...` | ok ×3 |
| go test -race | `go test -race ./...` | ok ×3 |
| skill-validator | `skill-validator check skills/skill-audit`, `… skills/skill-rewrite` | exit 0 both — 0 errors, 0 warnings (`-o json`: `errors=0 warnings=0 passed=True`) |
| skillscore global install | `skillscore --version` | **2.0.2**, intact. No PATH mirror was built. |

Per-suite, both shells:

```
tests/test_f01.sh        1886      tests/test_rewrite.sh     158
tests/test_f02.sh         350      tests/test_skill.sh        86
tests/test_harness.sh     191      tests/test_walk.sh         20
tests/test_install.sh      50      TOTAL                    2741
```

## Numbers Lane 2 must requote, each re-derived at head and at `9c8ba53`

The base figures were re-measured with the same commands, and they reproduce
S10's published values exactly — so these deltas are real and not a change of
method.

| figure | published (`9c8ba53`) | head (`d904c30`) | command |
|---|---|---|---|
| shell assertions | 2649 | **2741** | `for s in tests/test_*.sh; do /bin/bash "$s" \| tail -1; done` |
| — per suite | 1868/350/174/48/136/53/20 | **1886/350/191/50/158/86/20** | as above |
| top-level Go tests | 275 (233/34/8) | **285 (239/35/11)** | `go test -list '.*' <pkg> \| grep -c '^Test'` |
| subtests under `-race` | 669 (569/87/13) | **696 (579/87/30)** | `go test -race <pkg> -v \| grep -c '^=== RUN'` minus the top-level count |
| coverage | 97.9 / 94.5 / 93.5 | **97.9 / 94.7 / 97.0** | `go test -cover <pkg>` |
| assertion call sites audited | 727 (stated); 740 measured at base | **761** | `tests/lib/audit-suites.sh tests/test_*.sh \| tail -1` |
| OTLP fixtures | 64 | **64**, unchanged | `ls profiler/testdata/otlp/*.json profiler/testdata/otlp/*.ndjson \| wc -l` |
| DuckDB queries | 8 | **8**, unchanged | `ls profiler/queries/*.sql \| wc -l` |

Two notes on those.

**The 727 in `gate-numbers.md` was already stale when it was written.** The audit
reports 740 at `9c8ba53` (`sites=740 files=7`), not 727; the gate's per-suite
addition `292+101+108+48+120+38+20` is 727, and its `test_rewrite.sh` term of 120
is short — the audit reads 131 there at base. The lens's *conclusion* is
unaffected (a floor of 200 against either number is 3.6× low, and the injected
regression reproduced), but if Lane 2 quotes a site count, **740 → 761** are the
derived figures.

**`internal/homesafe` coverage moved twice.** The component walk took it to 89.6%
because the new refusal paths had no tests; the follow-up commit `d904c30` takes
it to 97.0%, above the 93.5% this release published. The two statements still
uncovered are `filepath.Rel`'s error return (cannot fail for two absolute paths)
and `os.Getwd`'s (not provokable in-process).

---

## C1 · CRITICAL · containment decided before symlink resolution

### The root

One invariant, and **three** implementations, not two. Every one of them made the
same two moves in the same order: fold `.` and `..` textually over the whole
string, then resolve symlinks over the deepest existing ancestor. A `..` that
crosses a symlink is therefore folded against the *link's own name* instead of
against its target, and by the time resolution runs the component that would
have answered the question is gone from the string.

`..` and a symlink **interleave** — a path may cross a link, climb out of its
target, and cross another — so no ordering of a whole-string fold and a
whole-string resolve is correct. That is why this is not a two-line reordering in
any of the three.

### Reproduced, before the fix

`skills/skill-rewrite/scripts/draft-rewrite.sh` — `$HOME/drafts/../skills/draft.md`
with `drafts` a link to `$HOME/.claude/skills`:

```
rc=0
--- landed inside protected? ---
/…/home/.claude/skills/draft.md
--- lexical location exists? ---
ls: /…/home/skills: No such file or directory
Rewrite draft written to: /…/home/drafts/../skills/draft.md
```

`profiler/internal/homesafe` — same spelling:

```
PathContains(home/.claude, spelled) = false err=<nil>
write landed at: "/…/home/.claude/skills/draft.md" (inside home/.claude = true)
```

`tests/test_install.sh` — **the third copy, not in the worklist.**
`git grep path_resolved` finds three. The third is the fence that decides whether
a README code block is *executed* by the suite, and its escape is the dangerous
direction: `~/checkout` is a symlink to the repository, which is not under the
harness scratch root, so `~/checkout/../ESCAPED-THE-SCRATCH-ROOT/.claude/skills`
folds to `~/ESCAPED-…` and compares as **inside** the scratch root. It was
accepted. The block's `mkdir -p`, `rm -rf` and `cp -R` would have landed beside
the repository. Every character in that spelling is inert and no rule in the file
names it.

```
  ~/checkout/../ESCAPED-THE-SCRATCH-ROOT/.claude/skills
FAIL: the resolved verdict alone refuses a climb out through a symlink
```

### The fix

All three are now one walk over the path's components in the order the kernel
walks them. A component that **is a directory** is entered (`cd -P` in shell,
`os.Readlink` + splice in Go), one that **is not there** is appended as named, a
**leaf that exists and is not a directory** is appended as named, and anything
else — a non-directory mid-path, a directory that cannot be entered, a cycle —
has **no answer** and stays a refusal. `..` pops the resolved prefix, which is
correct precisely because that prefix no longer holds a link or a `..`.

The textual folders are **deleted**, not left unused: `path_absolute` in the
drafter and `path_normalized` in `test_install.sh`. A lexical `..` folder sitting
in a file whose one invariant forbids lexical `..` folding is the next bug. In Go
the equivalent is that `resolve` no longer calls `filepath.Abs`, `filepath.Clean`
or `filepath.Join` at all; `filepath.Rel` survives in `PathContains` and is
commented as safe there and only there, because both of its arguments have been
through `resolve`.

A cycle is bounded at 40 hops in Go and needs no counter in shell (`-d` is false
for a cycle, and the walk consumes one component of a finite string per
iteration).

### Tests pinned to the invariant, not to the patch

No case in any of the three names an expected verdict. Each spelling is performed
twice over two trees built by one function: once as a real write, so the
**filesystem** says where the bytes landed, and once through the barrier. The two
must agree.

- Go: 12 spellings, device-and-inode comparison via `os.SameFile` over a walk
  that does not follow symlinks. 5 fail without the fix, in both directions.
- drafter: 12 spellings, a content marker located with `find -type f -exec grep`.
- `test_install.sh`: 9 spellings, `mkdir -p` plus a marker file; the tree's
  "outside" is a sibling directory *inside* the harness scratch, which is the
  only arrangement in which an oracle is possible at all — a case whose write
  must not be allowed to happen cannot be measured.

Both directions are in every table. A path that resolves *into* the protected
directory while its fold says otherwise is the hole; a path that resolves *out*
of it while its fold says inside is the false refusal, which costs a legitimate
destination. A repair closing only the first passes half of each table.

### RED / GREEN / RED

| barrier | RED | GREEN | RED again |
|---|---|---|---|
| `homesafe` | 5 of 12 subtests fail | all pass | `git show 9c8ba53:…/homesafe.go` swapped in → the same 5 fail |
| drafter | `143 passed, 11 failed` | `154 passed, 0 failed` | script reverted, tests kept → `143 passed, 11 failed` |
| `test_install.sh` | `22 passed, 1 failed` | `50 passed, 0 failed` | textual pre-fold reinjected into the new walk, spelling control removed → the oracle names `@B/root/home/aside/../d3` |

The third row is the one that matters most for non-vacuity: the RED was produced
by reintroducing **only** the pre-fold into the new walk, so the oracle is red
against the *defect* and not merely against the old code.

### Sized honestly, and one thing surfaced rather than done

Three copies of one barrier in two languages. Unifying the two shell copies means
either moving path resolution into `skills/skill-audit/scripts/verdict-guard.sh`
— a shipped skill surface, whose own comment reasons about that boundary and
names its trigger as "a second *script* that takes a destination", which a test
is not — or into `tests/lib/`, which the shipped drafter cannot source. That is a
refactor larger than the bug and it crosses a design boundary, so I did not do
it. What holds the three together instead is that all three now carry the same
oracle-shaped test, so a copy that degrades is caught rather than invisible.
**Recommend for 0.6.0: decide whether the shell resolver belongs in
`verdict-guard.sh`.**

---

## C3 · HIGH · `path_resolved` reported a resolvable path as unresolvable

Same root as C1, second half. Resolution ended in one `cd -P` over the deepest
existing ancestor, and you cannot `cd` into a file — so every destination that
already existed as a **regular file** came back unresolvable and exited 3, *"the
destination could not be resolved, so where the draft would be written is
unknown"*, for a path that is perfectly well defined.

Three consequences, all now closed by the one walk:

1. the undocumented fifth refusal is gone — an existing regular file is a
   destination, and the draft overwrites it;
2. `-o` is idempotent, so it can re-run over its own output the way the default
   destination has always overwritten its own `REWRITE-DRAFT.md`;
3. the leaf-symlink refusal at `:401` is **reachable**, and the rule
   `tests/test_rewrite.sh` documents as the one that fires now fires.

`refused_out` reads the **status** rather than `!= 0`, which is what reddens the
class from inside the suite:

```
the drafter refused …/skillmd-out/valid-full/SKILL.md with exit 3;
a refusal of the destination is exit 1, and 3 means no verdict was reached
FAIL: a destination that is the target's own SKILL.md is refused
FAIL: a destination that resolves onto a SKILL.md is refused
134 passed, 2 failed
```

Two refusals were passing on exit 3 while the rules they name had never run.
Also added: two runs to the same `-o` destination both exit 0 and neither says
"could not be resolved"; a destination that is somebody else's existing file is
written, not refused.

The exit-3 cases that must stay exit 3 still do — the unreadable directory
(`chmod 000`) and the symlink cycle both come back as no answer.

---

## C4 · HIGH · the protected-root list did not satisfy its own rationale

Reproduced: `rc=0`, draft written into `$HOME/.agents/skills/`. The README
documents `~/.agents/skills/` as a global skills directory for Codex (`:836`)
and for Cursor (`:842`), and it was not protected.

The root is not the missing entry. The suite compared the script's list only
against `skills/skill-rewrite/SKILL.md`, so the two agreed with each other and
both disagreed with the README — which is the document that says which
directories an agent reads skills from. The list had been assembled from harness
*names*, which is exactly why it missed the one shared directory belonging to no
single harness.

Fix: `.agents` added to `protected_home_dirs`; `SKILL.md`'s sentence widened,
since it said "a live agent configuration directory" while the thing protected is
anywhere an agent reads skills from, and `$HOME/.agents` is the second without
being the first; and the README added as the second side of the comparison.

**The judgment I made, stated.** The README side is read as a *path shape* —
every `$HOME`-relative path it spells with a `skills` component, reduced to its
first component — and it is deliberately over-inclusive: `~/.devin/skills/`
appears in the README only to say it is **not** the place, and this counts it
anyway. That is what makes an honest **equality** possible in both directions
without an exemption list, and over-inclusive is the safe direction for a refusal
list: protecting a directory nobody reads costs a destination, and failing to
protect one costs a document loaded as instructions. The alternative I rejected
was an exemption for `.devin`, because an exemption list is the shape that let
this through in the first place.

Derived, both sides:

```
  in the script  : .agents .claude .codex .config .cursor .devin
  in the README  : .agents .claude .codex .config .cursor .devin
```

RED `153 passed, 5 failed` → GREEN `158 passed, 0 failed` → RED again with the
root removed. A behavioural case drives it as well as the list comparison,
because a list comparison passes over a root the refusal loop never uses.

---

## C2 · CRITICAL · `hooks install` wrote a `hooks.json` with no `version`

Reproduced at the CLI. A virgin install produced `{"hooks":{…21 events…}}` and no
top-level `version`. Cursor's configuration reference marks it required: *"Config
schema version. Must be a positive integer (use 1)."*

**Confidence, kept where the lane put it.** The missing field is a fact. The
*consequence* — Cursor ignores the file, the spool stays permanently empty — is
inference from the field being documented required; no Cursor is installed here
and nothing was observed against a running one. That split is written into
`hooks_install.go`'s header rather than left to this record. The compounding harm
is **not** inference: `doctor` reads the same file we wrote and reports all 21
events registered, so the only diagnostic a user has would tell them it is
configured.

Why 31 hook tests missed it, which is the part worth repairing: every case that
mentions `version` seeds one into a file that already exists and then checks it
survived. It was tested as a field to *preserve* and never as a field to *write*.
Fixture and emitter came from the same document, agreed with each other, and both
disagreed with it.

Fix: `ensureSchemaVersion(root) bool` supplies `1` only when the key is absent,
called from `InstallHooks` and **deliberately not** from the `saveHooksDoc` that
uninstall shares — a required field added on the way out would be something of
ours left in a file we are meant to have vacated. The early return now has two
reasons to write, because a file carrying every registration and no version is
what every build before this left behind and re-running the install is the only
instruction a user has. `schema_version_added` is in the result (`omitempty`) so
that write does not report a backup it took and name nothing it did.

Four new tests: a virgin install writes a positive-integer `version`; a second
run adds nothing and changes the file's **bytes** not at all; a file registered
without one is repaired and says so; an existing `2` is preserved by install
*and* by uninstall, asserted on the value rather than on "not nil"; uninstall
invents none on a file that has none.

RED on two of the four with `ensureSchemaVersion`'s call neutered, GREEN with it.

At the CLI, after:

```
$ profiler hooks install --home <tmp> --command '/nonexistent/x ingest || true'
{ "hooks_json": "…/.cursor/hooks.json", "events_registered": 21,
  "events_removed": 0, "schema_version_added": true }
$ python3 -c "…"
top-level keys: ['hooks', 'version']
version: 1 int
```

---

## C6 · MEDIUM · `--home` scoped the registration but not the spool

Measured: `hooks install --home X` registered `<binary> ingest || true` with no
spool, so at hook time `ingest` resolved one from whatever `$HOME` was when the
hook fired, while `doctor --home X` reported `X/.skill-architect/spool` — a
directory the installed hook never writes to, reported as the spool with every
count in it zero.

Two roots, both repaired.

The **layout** was written out three times (`DefaultSpoolDir`, doctor's
`DetectEnvironment`, and implicitly in the CLI) and held together by a comment
saying a test held them. `profiler.SpoolDirIn(home)` is that layout, once, and
both callers call it.

The **resolution** was disjoint. `resolveHome` and `resolveHookCommand` each
carry a comment saying there is one of them so `install` and `doctor` cannot
disagree about the machine they describe — and then the command was resolved
without reference to the home. It now takes the home as a parameter, in
`hooksFlags` and in `cmdDoctor`, and names the spool for the same reason it
already named the binary by absolute path: a hook runs in an environment this
tool does not control. `--command` still wins outright.

The test never mentions the flag that implements it. It reads the command out of
the file `install` wrote, runs it through a shell the way a hook host would, and
requires the line to appear in the directory `doctor` names as the spool — with
`--home` and the environment's `HOME` pointed at two **different** sandboxes,
which is the only arrangement that can tell, since equal ones make the defect
invisible.

RED:

```
doctor reports 0 lines in "…/002/.skill-architect/spool" after the command it
says is registered wrote one; the registration and the report are not talking
about the same spool
```

GREEN after; RED again with only the `--spool-dir` dropped from the registration.

`runCLIIn` grew a `shellLine` mode rather than the case building its own
`exec.Command`, because `TestEverySubprocessRunsThroughTheSandboxedRunner` reads
this file with the AST and requires every subprocess to come through the
sandboxed runner — and a hook is exactly the case where a forgotten `HOME`
writes into the real spool. The allowlist was not widened.

---

## C5 · HIGH · the non-vacuity floors could not fire

### `tests/test_harness.sh` — floor 200, real 740

Re-measured at head: injecting `FNR > 400 { next }` into the audit takes the
examined sites from 740 to **225**, with `test_install.sh` audited at **zero**,
and every other suite green at exactly its published total on both shells. A
floor of 200 passes at 225. This audit is the only thing standing behind
non-vacuity for the `assert_value` sites `tests/lib/harness.sh` says rest on it.

A hand-written bound cannot ask the question, and **neither can the site count**:
the number of assertions found is not the amount of file examined. So the audit
now reports two figures a caller can compare against something.

- `lines` — every record the reader saw. Other side: `wc -l`.
- `accounted` — what the walk actually disposed of. Every record leaves the main
  rule through one of four paths and each one counts itself, so the two are equal
  unless something skipped a record without saying so. Other side: `lines`.

Held **per file** for equality, the pair refuses a limit introduced at either
end: above the walk (the measured regression — awk still reads every record, so
`lines` is untouched and `accounted` is not) or above the reader. Per file and
not in total, because one suite going unexamined is the case that happened and is
invisible in a sum. There is no number left in the file, and the site count is
now held as the sum of the per-suite counts.

RED, the measured regression against the new denominator:

```
sites=225 files=7 lines=9025 accounted=2594
the audit read 2708 lines of tests/test_f01.sh and accounted for 400: 2308 lines
went past the walk without being examined or deliberately skipped
FAIL × 6 suites + FAIL: the audit found an assertion call site in
tests/test_install.sh at all
```

and the same injection is driven **in-suite** as a control, over a copy of the
audit and over a real suite (a four-line fixture is under 400 lines, and a
control that cannot reach the limit it tests is the shape being refused), with
the rule stripped back out as the other direction.

### `tests/test_f01.sh:736` — floor 6, real 16

Two defects. The enumeration read one of bash's three definition spellings, and
re-spelling eight primitives as `name ()  {` dropped the derived set to 8 while
`verdict_guard_ready` still loaded — half the primitives lost their drop-one
coverage and 24 assertions vanished from the published total with zero failures.
`guard_broken_tree`'s `drop-*` `sed` had the **same** assumption one level down
and would have left a *working* guard in the tree, so every assertion over that
mode would have been about a guard that loads.

Both readers are now `guard_definition_names`, which reads all three spellings,
and each drop tree asserts the deletion actually happened.

The count is an **equality against the artifact's other side** —
`verdict_guard_ready`'s own name list, plus itself, since it is defined last and
is not in its own list — so there is no number here at all. It also catches a
primitive added to the file that the readiness check does not cover, which is a
bug in its own right.

Proven three ways:

```
old reader + the respelling → FAIL: every primitive the guard defines is one
  verdict_guard_ready requires, and none it does not
  defined : cannot_compute json_string mktemp_answers require_tool skill_body
            skill_frontmatter tool_answers verdict_guard_ready        (8)
  required: … (16)
  1853 passed, 1 failed      — where `-ge 6` passed at 8

fixed reader + the respelling → 1886 passed, 0 failed   (harmless, as it should be)

fixed reader + text_extract deleted → FAIL, defined 15 vs required 16
```

### `tests/test_f01.sh:1290` — floor 15, real 20, and `:2188` — floor 6, real 8

Both are exact counts now. The first one's neighbour is one-directional: the
emptiness of `unprobed_tools` means a *shrinking* walk finds fewer tools to be
unprobed and passes more easily. Nothing counts the second at all.

Proven: dropping one `require_tool` from `check-structure.sh` gives 19 —

```
  tool preconditions walked: 19
FAIL: the walk visited every one of the 20 tool preconditions the scripts state
  the scripts state 19 tool preconditions, and this file says 20
```

— which the floor of 15 accepted.

### The exemption that let two of them stand

`test_f01.sh` already recorded this ruling for `-ge 45`, and then exempted "the
other floors in this file" on the grounds that each sat beside an exact
comparison. **That was false for three of them.** The exemption is now an
enumeration: which floors were converted and why; which remain floors, with the
two-directional exact comparison that holds each one's set named
(`readme_missing`/`readme_stale`; `exit_undocumented`; `probe_call_floor`); and
what the rest of the numbers in the file are (`-gt 0` on a read that must not be
empty, a fixture's own content, a timing bound). A blanket exemption is how this
went another release.

---

## C7 · MEDIUM · eight false `0.5.0` forward promises in Go

Grouped by topic, three roots not eight edits, each re-pointed in the words S10
already used on its prose sibling.

- **probe's exit contract** — `cmd/main.go:118`, `cmd/main_test.go:369`. Both
  said making the status branchable is 0.5.0. It is not.
  `docs/profiler-spec.md` reads *"0.5.0 did not add one"*; `main.go` now cites it.
- **the schema-v1 caveat channel** — `claude_code.go:475`, `:611`, `otlp.go:1130`.
  All three said the gap is "tracked for 0.5.0". Each now says what the spec
  says: it needs a caveat channel schema v1 has not got, which 0.5.0 did not add.
- **the reserved `CaptureOpts` fields** — `types.go` doc comment and both field
  comments. *"The shape the Devin and Cursor adapters need in 0.5.0"* was the
  worst of the three, because 0.5.0 is the release that **removed three drafts of
  those adapters**. The comment now names no release as the one that will read
  them — naming the release being cut is precisely what turns a note into a false
  claim the moment that release ships — and says the true thing: reserved, and
  nothing in this release reads either one.

Untouched, as instructed: `profiler_test.go:325` (`retiredActivationDeferral`,
the negative control), `claude_code.go:837-838`, `docs/profiler-spec.md:72`, the
`doctor` tier machinery.

### The decision at `tests/test_rewrite.sh`

The test said building the drafter's four missing templates was scheduled into
this release. The shipped `skills/skill-rewrite/SKILL.md` frames the same gap as
a boundary — *"**What the drafter does not do**, and what Stage 3 is therefore
for … Scoring the dimensions and turning them into fixes is Stage 1 and Stage 3
work, done by the reader."* Neither cited the other.

**The test's sentence is the wrong one.** Reasoning, in order of weight:

1. It is a **false statement about a shipped release**. 0.5.0 did not build it.
   An arguable framing beats a false fact.
2. A comment in a test has no standing to schedule product scope. The shipped
   `SKILL.md` is the document that defines what the skill promises, and
   `AGENTS.md`'s 60/30/10 principle supports its framing: deterministic scripts
   carry mechanical work and judgment stays with the reader.
3. **Nothing in the repository commits to moving that boundary.** There is no
   open row for it in `.scuba/teams/gap-audit-0.4.3/deferred-ledger.md`, and the
   only two places that scheduled it — `CHANGELOG.md:400` and `:427` — are
   0.4.3's own text, which is history and stays byte-identical.

So the test now names no release, states what is true of the artifact, and cites
the `SKILL.md` sentence it holds; the `SKILL.md` says the suite pins each gap as
an absence, so closing one turns the suite red rather than leaving the document
describing a draft that no longer exists.

**Not decided, and flagged:** whether that boundary should ever move is a
product-scope question for the maintainer. Neither document now claims it will.
If the answer is "yes, build it", the place to record that is a ledger row, not a
comment. **Escalated.**

---

## C8 · MEDIUM · `MetricSource` declared five values no capture can produce

`SourceHooks`, `SourceHooksEstimated`, `SourceSessionData`, `SourceServerAPI`,
`SourceSQLite` — zero production references, and **no comment on any of them**, so
the claim was made silently by the declaration. A reader who sees `sqlite` in the
enumeration concludes some capture reads a SQLite database. `server_api` is
worse: its string is one of the three `doctor` tiers this release removed for
advertising a surface nothing reads, and `cmd/main_test.go:1704` asserts the
doctor report may not contain it *"which no capture in this release delivers"* —
while it stood 1,600 lines away as a valid profile source.

`declaredTiers` is the model, and this is it applied to the other vocabulary
(`git grep declaredSources` was empty).

- `declaredSources` reads the constants **and their doc comments** out of
  `types.go` with `parser.ParseComments`.
- `sourcesUsedInProduction` reads every non-test file in both packages and
  collects references, excluding each constant's own declaration. An AST walk and
  not a grep, precisely because `SourceHooksEstimated` is named twice in
  `types.go`'s *prose* and by no expression anywhere — a grep would have called
  it produced.
- The rule is both directions: produced, or the doc comment carries
  `reservedSourceNote` verbatim. A note left on a source that has *gained* a
  producer is also red.
- `TestTheSourceScanFindsTheSources` is the control on both derivations — the
  declared set against an exact list, the use scan against two sources production
  does refer to and one it does not.

RED on all five with the notes stripped and the constants left as they were;
GREEN with them; RED the other way with one fake reference added:

```
SourceSQLite ("sqlite") says "No capture in this release produces it." and is
referred to by [compare.go:159] — the note outlived the reservation
```

I did **not** delete the five constants. They are the profile schema's `source`
value domain and `docs/profiler-spec.md` documents them; removing them crosses
into Lane 2's file and into a schema decision. Making the claim explicit and
guarded is the repair the worklist asked for.

---

## C9 · GUARD · stop the fifth recurrence of the marker class

`tests/lib/forward-promise-check.sh`, driven from `tests/test_skill.sh`.

**Denominator is `git ls-files`.** The readers that ran before this one were
rooted at `.`, `../docs` and `../README.md` from inside `profiler/`, and two of
this round's eight markers lived in `tests/` and `skills/`, where none of those
roots reach.

**It reads a deferral vocabulary, not the version string.** The version appears
69 times in this tree and about sixty of those are correct — `did not add`,
`Until`, `every key it adds`, `this release is`, `AdapterVersion is`, a heading,
a manifest value — and a check that fires on forty correct lines is a check
somebody turns off. The words that make a line an assignment are few and this
repository has used the same ones every time: tracked, deferred, carried,
reserved, scheduled, planned, postponed; `is <version> work` and a bare
`is <version>`; `will … in <version>`; `needs … in <version>`; `new surface for
<version>`.

**Two scopes**, because the two release documents are the one place a forward
promise is legitimately *history*: 0.4.3's section says what 0.4.3 meant and is
byte-identical to the tag. In `CHANGELOG.md` and `RELEASE_NOTES.md` only the
section for the release being cut is in scope; everywhere else, every line is.

**One exemption, self-checking.** `profiler/profiler_test.go` carries the refused
sentence as a negative control; a check that fired on it would refuse the guard
standing against the thing it guards. If that exemption stops matching anything
the reader reports it as a finding of its own, so a stale exemption cannot sit
there widening the hole.

**Boundary, stated:** a line naming a version surface is exempt from the bare
`is <version>` shape, because `AdapterVersion is 0.5.0` and `building the
capability is 0.5.0` are the same three words doing opposite jobs. A forward
promise written into a line that also mentions a version gets past this. Closing
that needs the sentence parsed.

18 controls: nine shapes it must refuse and nine correct lines it must accept,
taken from lines actually in this tree with the version substituted so the
fixtures do not trip the reader over themselves. The file count is compared for
equality against `git ls-files` (non-empty, because awk's per-file counter cannot
fire for `profiler/testdata/otlp/empty.json`).

RED, four ways:

```
one C7 marker restored in types.go
  profiler/types.go:140: [deferral verb] … reserved for 0.5.0
  FAIL: no tracked file assigns work to the release being cut

markers planted where the old readers did not look
  skills/skill-rewrite/SKILL.md:195: [deferral verb] A template for `Constraints` is deferred to 0.5.0.
  tests/test_walk.sh:196: [deferral verb] … that is tracked for 0.5.0

the exemption renamed so it goes stale
  profiler/profiler_test.go: the exemption for the negative control matched
  nothing, so it is stale or the control is gone
```

**Also fixed while proving it:** the diagnostic that prints the findings is a
pipeline, `pipefail` passed the reader's exit 1 out of it, and `set -e` killed
the suite one line after the assertion written to report exactly that — the same
failure `tests/test_harness.sh` records against `workflow_suites`. With `|| true`
the suite reports FAIL and reaches its summary.

---

## C10 · GUARD · the release documents are a guarded version surface now

Confirmed: deleting the entire 0.5.0 section from both documents left every
assertion green and byte-identical, and `grep -rn 'CHANGELOG\|RELEASE_NOTES'
tests/*.sh tests/lib/*.sh` returned nothing. `tests/test_skill.sh:93-96` learned
this lesson for `marketplace.json` and did not apply it to the two documents that
*are* the release.

No number in the block. The release version is read out of
`.claude-plugin/plugin.json`; each document must carry a section for it; each
section must have a non-blank line under it, because a heading with nothing
beneath it is the same deletion one line later. Then the strong half: the two
documents must name the **same set** of releases, derived from both and compared
for equality — which holds every release rather than this one, and fires
whichever file a section was deleted from.

RED, both shapes, in a throwaway worktree so Lane 2's files were never edited in
mine:

```
both 0.5.0 sections deleted      → 82 passed, 4 failed
## v0.4.1 deleted from RN only   → FAIL: the changelog and the release notes
                                   carry a section for the same set of releases
  in the changelog    : 0.1.0 0.2.0 0.3.0 0.3.1 0.4.0 0.4.1 0.4.2 0.4.3 0.5.0
  in the release notes: 0.1.0 0.2.0 0.3.0 0.3.1 0.4.0 0.4.2 0.4.3 0.5.0
```

---

## Out of scope, left alone, escalated

**Left alone deliberately** (named by the worklist as correct):
`profiler/profiler_test.go:325`, `docs/profiler-spec.md:72`,
`profiler/claude_code.go:837-838`, the `doctor` tier machinery. None was edited.

**Lane 2's files**, untouched: `CHANGELOG.md`, `RELEASE_NOTES.md`, `README.md`,
`docs/profiler-spec.md`.

**Not mine, not done:** every DEFERRED item (M2/M3/M4, the `preview`/`Slice 1`
staleness, `-o ""` at `:394`, `profiler/go.mod`'s patch pin,
`main_test.go:63`'s 91.5% citation, `harness.sh:20`'s wrong filename, the 13 open
ledger rows). None of them is in C1–C10.

**Escalated to the manager:**

1. **The third containment barrier is a worklist gap, not just a fix.** C1 named
   two; there are three. If the reconciliation is being used as a coverage
   record, `git grep path_resolved` is the denominator it should have had.
2. **Whether `skill-rewrite`'s four-template boundary should ever move** (C7's
   decision). A product-scope question. Neither document now promises it; if the
   answer is yes, it wants a ledger row.
3. **Unifying the shell resolver into `verdict-guard.sh`** — larger than the bug
   and across a design boundary. Recommended for 0.6.0, not done.
4. **`gate-numbers.md`'s 727 is itself stale** (real 740 at base). Worth a note
   wherever gate figures are recorded, since this release has already had five
   numbers go stale.

## Threads replied to / resolved

None, because there are none. PR #22 has **0** review comments and **0** issue
comments (`gh api repos/Okja-Engineering/skill-architect/pulls/22/comments --jq
'length'` → `0`; the issue-comments query returns nothing). No external reviewer
thread exists to reply to or resolve. Nothing was resolved.

## Commits, in order

```
0bf2a34  C1: the home barrier resolves symlinks before it folds `..`
f9bc255  C1+C3: the drafter's `-o` bound resolves symlinks before it folds `..`
97131c1  C1, third copy: the install suite's own containment fence had the same defect
0302ea2  C4: the protected-root list now satisfies its own stated rationale
f9c9380  C2: `hooks install` writes the schema version Cursor documents as required
57685a9  C6: `--home` scopes the capture as well as the registration
c346a1d  C5: the non-vacuity floors are derived denominators compared for equality
76c56c7  C7: the eight false 0.5.0 forward promises in Go, by topic
2fd02c2  C8: MetricSource's five unproducible values now say so, and a guard holds them
5284652  C9+C10: the marker class and the release documents get guards
d904c30  C1 follow-up: the barrier's new refusal paths get the negative tests they need
```
