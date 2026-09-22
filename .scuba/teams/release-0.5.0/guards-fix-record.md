# GUARDS lane — verification and completion. Record. PR #22, `release/0.5.0`

Received head `e2fb39d` (four recovered commits on `619cab7`, verification unrecorded).
Delivered head **`c3a565c`**, four further commits, author `imagineux
<imagineux@gmail.com>` on every one, no `Co-Authored-By` and no "Generated with" line
(`git log e2fb39d..HEAD --format='%B' | grep -inE '^Co-Authored-By|Generated with
\[Claude|noreply@anthropic'` → NONE; `git log e2fb39d..HEAD --format='%an <%ae>' | sort
-u` → one line).

```
3df3760  Both sides of the audit's record count are the same program, and the shipped-skill floor gets git
1bcddf0  The harness's own controls run under the shell the suite is running under
18dd0f1  The derived name set refuses to come back empty, and a control says so
c3a565c  Two sentences the round wrote are measured and corrected
```

Worktree: `…/scratchpad/gfin`, `git worktree add --force … release/0.5.0`. One throwaway
detached worktree at `619cab7` (`…/scratchpad/gbase`) to derive the pre-guards audit
figures, since removed. The stale `…/scratchpad/guards-lane` was not touched. Nothing was
written in the primary tree except this file; it is still at `0a83615` with its ~2,900
uncommitted lines. No `checkout`/`reset`/`clean`/`stash` in any tree but my own. Every
injection below was made in my own worktree and restored from a copy taken first; the tree
was confirmed clean (`git status --short` empty) after each.

Scope: `git diff --name-only e2fb39d..c3a565c` → `tests/lib/audit-suites.sh`,
`tests/test_harness.sh`, `tests/test_skill.sh`. **Lane 2's four files are untouched** — no
`CHANGELOG.md`, `RELEASE_NOTES.md`, `README.md` or `docs/profiler-spec.md`. The barrier
lane's areas were not re-opened. `tests/test_f02.sh` was read and deliberately not
changed (F7d, below).

Pushed per unit: `e2fb39d..3df3760`, `..1bcddf0`, `..18dd0f1`, `..c3a565c`. Not merged,
not tagged.

**PR threads: there are none.** `gh api graphql … pullRequest(number:22)` →
`reviewThreads.totalCount 0`, `hasNextPage false`, `reviews.totalCount 0`,
`comments.totalCount 0`. Nothing to reply to and nothing to resolve.

---

## 1 · F1–F9, each re-verified by injecting the report's own regression

Every injection was driven on **both** `/bin/bash` 3.2.57 and `/opt/homebrew/bin/bash`
5.3.15 unless the row says the reader was exercised directly, in which case both were still
run. Every one was restored and the tree re-confirmed green afterwards.

### F1 · CLOSED · both shells — a quoted `<<WORD` no longer swallows the file

Injected at `tests/test_install.sh:2019`, exactly as the report describes:

```sh
assert "the install route writes no heredoc terminator of its own" \
  quietly grep -qv '<<INSTALL_ROUTE' tests/test_install.sh
assert "the destination is contained" true
assert_value "the copy landed" true
```

Audit **rc=1**, and the site count **did not collapse**:

```
tests/test_install.sh:2022: verdict is the constant command 'true', so this assertion cannot fail: assert "the destination is contained" true
tests/test_install.sh:2023: verdict is the literal 'true', so this assertion cannot fail: assert_value "the copy landed" true
read=tests/test_install.sh sites=55 lines=2847 accounted=2847
```

Against the report's `sites=1 files=1 lines=2133 accounted=2133`. **This is the point that
mattered and it holds: the repair did not make both sides agree again.** `sites` went to 55,
not to 1, because `heredoc_term` now reads `<<` from the *tokens* and a `<<` inside a quoted
word is text. The two vacuous assertions are named individually.

Assertions that fire, `tests/test_harness.sh` 274 passed / 2 failed on both shells:
`tests/test_install.sh has no assertion that cannot fail, and no harness of its own`, and
`the quoted <<WORD is not itself a finding, so the refusal above is the vacuous
assertions'` (the second is an artifact of injecting into the real file the control copies).

The second half of the repair is independent of any count: an unterminated heredoc is a
violation in its own right (`audit-suites.sh` `close_file`), with fixture controls at
`test_harness.sh:855-858` in both directions. Re-verified at the final head `c3a565c`:
same two findings, `sites=55`.

### F2 · CLOSED · both shells — `assert ( ) {` is a redefinition

Injected after `harness_init` in `tests/test_walk.sh`, verbatim from the report. Three
independent routes fire:

```
tests/test_walk.sh:6: a private copy of the harness name 'assert', free to drift from the shared one: assert ( ) {
```
audit **rc=1**; and the suite, **20 passed / 2 failed identically on both shells**:

```
the live 'assert' was defined by tests/test_walk.sh, not by …/tests/lib/harness.sh: this suite is running a private copy of the harness
FAIL: the harness this suite is running is not the one in …/tests/lib/harness.sh
FAIL: an assertion ran at a line the assertion audit never examined
```

The third one is the census failing **closed** on emptiness — "this suite recorded no
assertion call site at all, so the census has nothing to hold and cannot report" — because
the shadowed `assert` never calls `harness_note_site`.

The reader now has no list of spellings at all (`defined_name_at` reads an optional
`function`, a name, and an optional parenthesis pair that may contain whitespace). Held
against bash: `test_harness.sh:657-685` generates 14 spellings, sources each into a real
shell, asks `declare -F` what that shell defined, and requires the reader to agree — with
the audit's expected verdict **computed from the oracle** rather than written beside the
case, and both directions counted (`spelling_shadowing -gt 0`, `spelling_innocent -gt 0`).

### F3 · CLOSED · both shells — `trap cleanup 0` and `trap cleanup exit`

The reader driven over nine one-line fixtures:

| spelling | rc | reported as |
|---|---|---|
| `trap cleanup EXIT` | 1 | `spelled 'EXIT'` |
| `trap 'cleanup' EXIT` | 1 | `spelled 'EXIT'` |
| **`trap cleanup 0`** | **1** | **`spelled '0'`** |
| **`trap cleanup exit`** | **1** | **`spelled 'exit'`** |
| `trap cleanup Exit` | 1 | `spelled 'Exit'` |
| `trap - EXIT` | 1 | `spelled 'EXIT'` |
| `trap '' EXIT` | 1 | `spelled 'EXIT'` |
| `trap cleanup INT` | 0 | no finding |
| `trap -p EXIT` | 0 | no finding |

Identical on bash 5 for the three that decide it. End-to-end, `trap cleanup 0` planted in
`tests/test_walk.sh`: audit names it at `:7`, and the suite reports **20 passed / 1 failed**
on both shells via bash's own `trap -p EXIT` — "the EXIT trap is no longer the harness abort
guard but `trap -- 'cleanup' EXIT`".

There is no list of spellings: `is_exit_spec` is `tolower(d) == "exit" || d == "0"`.
`test_harness.sh:741-770` drives 13 trap spellings and derives the required verdict by
**firing a real guard in a real shell** and asking whether it survived, both directions
counted.

### F4 · CLOSED · both shells — the audit reads the closure under `source`

The report's exact injection: `assert_value() { echo "PASS: $1"; pass=$((pass + 1)); }`
appended to `tests/lib/masked-path.sh`, `skills/skill-audit/SKILL.md` replaced with
`BROKEN`.

```
tests/lib/masked-path.sh:89: a private copy of the harness name 'assert_value', free to drift from the shared one: …
read=tests/lib/masked-path.sh sites=0 lines=89 accounted=89
read=tests/lib/mutation-runner.sh sites=0 lines=486 accounted=486
sites=817 files=9 lines=11655 accounted=11655
audit rc=1
```

**`files=9`, not 7.** The audited set is now computed here, following every `source` it can
resolve and refusing every one it cannot.

- `tests/test_f01.sh`: **1912 passed, 1 failed** — no longer its published figure over a
  broken tree, on both shells. The failure is `harness_ready`'s: "the live 'assert_value'
  was defined by tests/lib/masked-path.sh, not by …/harness.sh".
- `tests/test_harness.sh`: **270 passed, 4 failed** on both shells — one per suite whose
  closure reaches the library: `FAIL: tests/test_f01.sh has no assertion that cannot fail,
  and no harness of its own`, and the same for `test_f02`, `test_rewrite`, `test_skill`.

Re-verified at the final head `c3a565c`: `sites=821 files=9`, f01 1912/1, harness 272/4.
Restored → `audit rc=0`, f01 1912/0, harness 276/0.

### F5 · CLOSED · both shells — both halves

**The name list.** `assert_all()` added to `tests/lib/harness.sh` as a new shared primitive
and shadowed in `tests/test_walk.sh`. `audit-suites.sh --names tests/lib/harness.sh`
immediately returns it:

```
assert assert_all assert_value harness_census harness_closure_covers harness_definition_file
harness_exit harness_init harness_loaded_files harness_note_site harness_ready
harness_summary quietly require witness_exit
```

audit **rc=1**, `tests/test_walk.sh:6: a private copy of the harness name 'assert_all'`, and
the suite **20 passed / 1 failed** via `declare -F`. **No list was edited anywhere.** Held
for equality against bash at `test_harness.sh:605`, with `require`s on both sides being
non-empty.

**`harness_ready` has callers.** `grep -rnE '(^|[^a-z_#])harness_ready( |$|;|\))' tests/`
→ `tests/lib/harness.sh:186` (`harness_init`) and `:480` (`harness_summary`), plus one
comment in `mutation-runner.sh`. And the claim it always made is now true: truncating
`harness.sh` to 345 of 514 lines at a function boundary (`bash -n` clean) gives

```
the harness recorded none of the names it provides, so it did not finish loading
FAIL: the harness this suite loaded is not the one in …/harness.sh
FAIL: suite aborted before reaching its summary (exit 1)
```

### F6 · CLOSED · both shells — the promise reader's record count has an other side

The report's injection, in two stages.

**Stage A**, promise appended to `README.md` (line 992): reader finds it, **rc=1** —
`README.md:992: [deferral verb] - the count of skipped records is deferred to 0.5.0`,
`files=162 lines=43210 examined=42845 skipped=365`.

**Stage B**, `FNR > 400 { next }` inserted **below** `{ lines++ }` — the regression that hid
it. The reader now refuses **on its own terms**, identically on both shells:

```
the reader saw 43211 records, looked at 21488 and deliberately skipped 125: 21598 went past the walk unaccounted for
```
and `tests/test_skill.sh` reddens with the named assertion **`every record the reader read,
it either looked at or said it was skipping`** (97 passed / 2 failed, both shells).

**The other end too.** `FNR > 400 { next }` **above** `{ lines++ }`: the reader's own
accounting is self-consistent and it exits 0 — and the planted README violation is hidden —
but `test_skill.sh` catches it through the independently derived side, `grep -ac ''` over
the same files: **`the forward-promise reader read every record in those files, not a
prefix of each`** (96 passed / 3 failed).

**The F6-shaped control defect is closed too.** The 18 single-file one-line fixtures are
still there, and two new controls are driven over the **real tracked tree** (43,209 records,
files of 2,842 lines), one with a limit at each end a limit can be introduced at, each with
the inverse direction asserted over the same copy with the rule taken back out.

### F7 · CLOSED in three of four parts; the fourth is **INVALID** and was not changed

The four floors the enumeration listed are **gone rather than re-listed**, and each has a
derived neighbour. Live `-ge`/`-le` comparisons in `tests/*.sh` and `tests/lib/*.sh` at head,
comments stripped:

```
tests/test_f01.sh:1715   probe_calls_of >= 1        the stated requirement itself, reduced to an equality below
tests/test_f01.sh:2547   loop bound (i -le 520)     a fixture generator
tests/test_f01.sh:2788   grep -c … -ge 1            emptiness of a fixture's own content
tests/test_f01.sh:2790   grep -c … -ge 1            same
tests/test_f01.sh:2896   loop bound (i -le $body_lines)
tests/test_install.sh:484, :655                     containment internals (barrier lane's)
tests/lib/mutation-runner.sh:267                    a deadline
```

`f01:2132` → `-gt 0`; `f01:2517` → `-gt 0`; `f01:2828` → `-gt 0`; and the omission the
enumeration never listed, `.summary.policy_failures -ge 8`, is now an **equality against
the findings the same payload reports**, with `is_a_count` on both sides so a run that
emitted no JSON cannot pass as `[[ "" -eq "" ]]`.

**F7d — `tests/test_f02.sh:33-39` `assert_jq_gt` "with zero call sites" — INVALID, pushing
back with evidence.** It has two call sites, and it had them at `011defa` when the report
was written:

```
git show 011defa:tests/test_f02.sh | grep -n assert_jq_gt
33:assert_jq_gt() {
106:assert_jq_gt "no-license: policy_failures > 0" '.summary.policy_failures' 0
135:assert_jq_gt "missing-script-ref: summary.path_failures > 0" '.summary.path_failures' 0
```

`git log 011defa..e2fb39d -- tests/test_f02.sh` is empty, so nothing moved. Both sites
execute (`PASS: no-license: policy_failures > 0`, `PASS: missing-script-ref:
summary.path_failures > 0`) and both compare against **0**, so they are emptiness guards and
not floors. There is nothing to remove and no floor waiting for a caller. `tests/test_f02.sh`
is unchanged.

### F8 · CLOSED · both shells — and it is now guarded

The false "819 assertions" figure is gone; the sentence that replaced it records that it was
false under every derivation and that nothing read it. All four references to
`tests/lib/audit-assertions.sh` are gone (`grep -rn 'audit-assertions' tests/` returns one
hit, the comment in `test_harness.sh:950` explaining the defect).

And it is no longer only prose. `test_harness.sh:957-969` derives every `tests/lib/*.sh`
path named anywhere in the audited closure plus the harness and the audit, and requires each
to exist. Injected — one sentence added to `harness.sh` naming
`tests/lib/audit-assertions.sh` — both shells:

```
FAIL: tests/lib/audit-assertions.sh, named in the harness cluster, is a file that exists
274 passed, 1 failed
```

### F9 · Two proven halves CLOSED; the SUSPECTED half is **REAL and open, deliberately**

**Half one, the fail-closed reader — CLOSED, non-vacuously.** Re-spelling
`verdict_guard_ready` in the guard as `function verdict_guard_ready {`:

- pre-repair `test_f01.sh` (from `619cab7`), run against the re-spelled guard:
  **1885 passed, 1 failed** — `FAIL: every primitive the guard defines is one
  verdict_guard_ready requires, and none it does not`
- at head: **1912 passed, 0 failed**

**Half two, the fail-open reader — CLOSED, non-vacuously.** All 16 primitives re-spelled as
`name ( ) {` (`bash -n` clean):

```
new reader (grammar, audit-suites.sh --names):  16
old reader (the three-spelling sed):             0
```

And the sweep that holds it is non-vacuous: putting the old sed back into
`guard_definition_names` reddens `test_f01.sh` on exactly the three spellings the old reader
missed — `1909 passed, 3 failed`, naming `` `@N ( ) {` ``, `` `@N (  ) {` `` and
`` `function @N ( ) {` ``. There is now **one** reader of shell function definitions in the
repository, `guard_definition_names` delegates to it, and `test_harness.sh` holds that
reader against bash.

**Half three, the coordinated change — REAL, reproduced, and deliberately not patched.**
Adding a seventeenth, dead primitive to `verdict-guard.sh` *and* to `verdict_guard_ready`'s
list in one edit:

```
guard primitives examined: … nobody_calls_this …
1916 passed, 0 failed
```

**+4 assertions, still green.** The symmetric removal is the same thing at 1908. So the
finding is real as stated.

**I am not closing it, and this is the argument.** It is not a fail-open of any guard. In
F1, F4 and F6 the denominator could collapse **while the artifact was unchanged** — a reader
bug, a narrow closure, an inserted limit. Here it moves only when `verdict-guard.sh`
actually changes, and the loop's coverage is not "16 things" but "every primitive there is",
which tracks the artifact exactly. There is no external authority on what
`verdict-guard.sh` ought to define: I derived the candidate other side — every primitive
called somewhere in the guard's closure — and it does not work, because five of the sixteen
(`json_string`, `mktemp_answers`, `skill_read`, `skill_section`, `tool_answers`) are called
only from inside the guard, so that side is read out of the same file and shrinks with it.
The dangerous version — a primitive removed while a caller remains — is already caught, by
the caller dying. The only remaining pin would be
`GUARD_PRIMITIVES_EXPECTED=16`, which is the hand-kept number this whole round removed four
of, and which would be raised by hand in the very commit that changed the guard. **Prefer no
number to a number** says leave it. Recorded here rather than patched.

The reader-spelling fragility the same finding named in `guard_names_ready_requires` is
closed by `guard_ready_body`, and both readers are held against every spelling by the
re-spelling sweep at `test_f01.sh:836-880`, whose own denominator is asserted non-empty.

---

## 2 · Derived versus enumerated, per guard

| guard | sides | verdict |
|---|---|---|
| shadowing — **which names** | reader's `--names` over `harness.sh` **vs** bash's `declare -F` (`harness_provides`), equality, no number | **both derived** |
| shadowing — **which spellings** | grammar in `defined_name_at`, no list **vs** bash's `declare -F` per generated spelling, verdict computed from the oracle | **both derived**; the 14 *templates driven* are enumerated (stated plainly in the file) |
| shadowing — **EXIT-trap spellings** | `tolower(d)=="exit" \|\| d=="0"` **vs** firing a real guard in a real shell and asking if it survived | **both derived**; 13 templates enumerated |
| shadowing — **which files** | closure under `source`, computed, unresolvable directive is a violation **vs** bash's `harness_loaded_files`, **and** a grep second reader | **three derived sides** |
| audit `lines` | awk's record count **vs** `grep -ac ''`, per file, equality | **both derived** (was `wc -l`; changed in `3df3760`) |
| audit `accounted` | the four disposal paths **vs** `lines` | derived, and explicitly recorded as insufficient alone |
| audit `sites` | the reader **vs** bash's own record of the lines it ran an assertion on (`harness_note_site` → `harness_census`), span containment | **both derived**, no number |
| runaway heredoc | none needed — an unterminated heredoc is a violation | **no number at all** |
| promise reader `files` | the walk **vs** `git ls-files` from the repo root | **both derived** |
| promise reader `lines` | awk **vs** `grep -ac ''` from the repo root | **both derived** (new) |
| promise reader `examined`/`skipped` | `examined + skipped == lines`, checked in the reader **and** by the caller; skip *scope* held against the script's own `HISTORY_DOCS`/`EXEMPT_FILE` | **both derived** |
| suite list | `tests/test_*.sh` glob **vs** the CI workflow's steps, equality in both directions | **both derived** |
| guard primitives | `--names` over the guard **vs** `verdict_guard_ready`'s own list, equality | **both derived, from the same artifact** — F9's residual, argued above |
| shipped skills | `skills/*/` filesystem glob **vs** `git ls-files` from the repo root, equality | **both derived** (new; was `-ge 2`) |
| `*_EXPECTED` ×4 | `REQUIRED_TOOL_PRECONDITIONS=20`, `BROKEN_CASES=173`, `PREFLIGHT_CANDIDATES=8`, `PROMISE_CASES=18` | **enumerated, and defensible**: each is the number of *generated cases driven*, not a coverage denominator over the tree, and the report proved all four redden |

**Still enumerated, and why each is defensible:** the definition- and trap-spelling
*templates* (a new spelling mechanism is a line somebody writes, and the verdict for each is
still computed from bash, so a template nobody wrote is an untested spelling and not a
fail-open); the four `*_EXPECTED` case counts (proven to redden, and they pin a cross-product
this suite generates itself); the tokenizer's operator set; and the guard-primitive
denominator (F9, argued above). The file says all of this out loud rather than implying
otherwise.

---

## 3 · The defect shape in the repair itself — three found, all fixed

The brief named two properties to hunt for. All three findings below are one or the other of
them, in code the guards round wrote.

### (a) A derived side that can come back empty and say nothing — `audit-suites.sh`

The list of harness names was replaced by a derivation, and the derivation had **no floor
under it**. If `harness_names` comes back empty, `harness_name[]` is empty, every lookup in
the shadowing half misses, and the check is unconditionally silent. Measured, before the
fix — pointed at a harness defining no function, over a suite whose first line is
`assert() {`:

```
== with the REAL harness ==
…/shadows.sh:1: a private copy of the harness name 'assert', free to drift from the shared one: assert() {
…/shadows.sh:4: a private EXIT trap (spelled 'EXIT') …
rc=1
== with a harness that defines nothing ==
…/shadows.sh:4: a private EXIT trap (spelled 'EXIT') …
rc=1
```

The shadow **vanishes from the report**; rc=1 came from the unrelated trap, so a suite that
shadowed a name and nothing else would have read clean. This is a short list with the list
taken out — and the file's own prose already states the rule for the other derivation it
made total ("the derivation has to be total, and where it cannot be, it says so instead of
returning a smaller world").

Fixed in `18dd0f1`: `audit` mode refuses with the usage status and names the harness.
Scoped to `audit` on purpose — `--names` *is* the derivation and is handed an empty set
deliberately, and `--sites`/`--closure` never consult a harness name; refusing in `--sites`
took the census's own diagnostic away from it, which the first attempt did and which
`test_harness.sh` caught immediately (`FAIL: a suite that ran an assertion the reader never
examined says which line`). Control at `test_harness.sh:608-640`, both directions.
RED→GREEN→revert-to-RED: **275/1 → 276/0 → 275/1**, both shells.

### (b) A control that cannot reach the condition it tests — the shell it runs under

Nine fixtures driving the harness's three runtime guards launched a bare `bash`, which is
whichever interpreter is first on PATH — `/opt/homebrew/bin/bash` 5.3.15 here. So those nine
controls ran under bash 5 **whichever shell drove the suite**: "the abort guard fires on both
shells" was a claim about one of them, and a fail-open in any of the three guards existing
only on 3.2.57 had no control in this repository that could reach it. `bash_defines_in` and
`abort_guard_survives`, 200 lines above, already used `"$BASH"` for exactly this reason.

Fixed in `1bcddf0`: all nine use `"$BASH"`. Driven under 3.2.57 for the first time — **all
nine pass**, so nothing was hiding, including `declare -F` under `extdebug` naming the
defining file, which every shadowing check rests on and which had only ever been exercised
on bash 5.

### (c) The `wc -l` / record-count mismatch — the denominator that replaced the `-ge 200` floor

`audit_read_whole_file` is the whole replacement for the old floor, and its other side was
`wc -l`. awk counts **records**; a file whose last line has no newline holds one more record
than `wc -l` reports, and `tests/lib/forward-promise-check.sh` — written in the same round,
twenty lines from the same choice — says so explicitly and uses `grep -ac ''` instead. Two
tracked files in this repository are already written that way
(`profiler/testdata/otlp/malformed.json` 0/1, `profiler/testdata/otlp/truncated_final_line.ndjson`
3/4), so the condition is reachable here; only the audit's closure happens to exclude them.

Measured, before the fix, appending `: ;` with no trailing newline to
`tests/lib/masked-path.sh`:

```
the audit read 90 lines of tests/lib/masked-path.sh, which has 89: a check over a file it did not finish reading says nothing about the rest of it
FAIL: the audit read every line of tests/lib/masked-path.sh, not a prefix of it
273 passed, 1 failed          (both shells)
```

The audit was right and its own denominator was wrong. A guard that refuses a legal file is
a guard somebody turns off. Fixed in `3df3760`; with the same injection in place the suite
returns **274 passed, 0 failed** on both shells, and the truncating-audit control still
fires.

### (d) Two false sentences the round wrote — corrected in `c3a565c`

- `test_harness.sh` claimed the grep second reader "finds at least as much as the audit's
  tokenizer does". It also finds **less**: it matches only a literal path on the directive,
  so `. "$mutation_runner"` at the bottom of that same file is invisible to it while the
  tokenizer resolves the variable and reads the library. Measured: grep sees **1** library
  (`tests/lib/masked-path.sh`), the closure holds **2** (plus
  `tests/lib/mutation-runner.sh`). The direction actually asserted — containment — is
  unaffected, and is now what the sentence says.
- `audit-suites.sh` said `wc -l` is the other side of `lines`. Corrected with (c).

### Also completed: the fourth floor==real case the round left behind

The report's own enumeration named four `floor == real` cases: `f01:2517` 5/5, `f01:2724`
8/8, `f01:2828` 6/6, and **`test_skill.sh:116` 2/2**. Three were repaired in
`tests/test_f01.sh`; the fourth was not, and it is in this lane's files. Its own sentence
claims only that the set was "not matched as an empty set", which is what `-gt 0` says and
what 2 never said — and a floor cannot see the denominator move the *other* way. Measured,
an untracked `skills/skill-untracked/SKILL.md`:

```
shipped skills: skill-audit skill-rewrite skill-untracked
PASS: the shipped skills were read from the tree, not matched as an empty set
106 passed, 2 failed     — and the only two failures are incidental content checks:
FAIL: skill-untracked's compatibility line names an interpreter at all
FAIL: skill-untracked's executable scripts each name an interpreter in their shebang
```

Seven assertions joined the published total, the floor said nothing, and a well-formed
untracked skill would have gone through green. Fixed in `3df3760`: `-gt 0`, plus an
equality against git's index read from the repository root, with a `require` on the second
side. With the same injection: **`FAIL: every skill directory in the tree is one this
repository tracks a SKILL.md for, and none it does not`**, 107/3 on both shells. Removed →
101/0 on both.

### Residuals reported, not changed

1. **F9's coordinated change** — argued above. Real, reproduced at +4/green, deliberately
   left with no number.
2. **`tests/test_harness.sh:131` says the site floor stood "against a real 740", and
   `:339` says `sites=767 files=7`.** Both are true of different commits (740 at
   `9c8ba53` per `lane1-record.md`, 767 at `619cab7`, derived here) and neither names its
   commit. Not false; will go stale.
3. **`harness_census` caps span expansion at `$2 + 500` physical lines.** A logical line
   longer than that would truncate its span and produce a false RED. No such line exists;
   the direction is fail-closed.
4. **`assert "$@"` as a variadic forward is a reader false-positive** ("assert has no
   verdict to run"), observed while injecting F5. Fail-closed, and no such call site exists
   in the tree.
5. **`harness_closure_covers` can only see libraries that define functions.** A library
   sourced purely for variables is invisible to bash's side; the grep second reader covers
   it only if the path is a literal. Stated in the file.

---

## 4 · Every number, with the command that produced it

At `c3a565c`, clean tree, both shells, from `…/scratchpad/gfin`.

| figure | command | value |
|---|---|---|
| shell assertions, **bash 3.2.57** | `for s in tests/test_*.sh; do /bin/bash "$s" \| tail -1; done` | **3112**, 0 failed |
| shell assertions, **bash 5.3.15** | same with `/opt/homebrew/bin/bash` | **3112**, 0 failed, identical per-suite |
| — per suite, both shells | as above | `test_f01` **1912** · `test_f02` **350** · `test_harness` **276** · `test_install` **52** · `test_rewrite` **401** · `test_skill` **101** · `test_walk` **20** |
| assertion audit | `tests/lib/audit-suites.sh tests/test_f01.sh … tests/test_walk.sh \| tail -1` | **`sites=821 files=9 lines=11749 accounted=11749`** |
| — the closure, named | `tests/lib/audit-suites.sh --closure tests/test_*.sh` | the 7 suites + `tests/lib/masked-path.sh` + `tests/lib/mutation-runner.sh` |
| forward-promise reader | `tests/lib/forward-promise-check.sh 0.5.0 \| tail -1` | **`files=162 lines=43333 examined=42968 skipped=365`** (42968+365=43333 ✓) |
| — cwd-independence | same, run from `profiler/` and from `tests/lib/` | **identical**, `files=162 lines=43333 …` |
| harness names, two sides | `tests/lib/audit-suites.sh --names tests/lib/harness.sh \| wc -l` and bash's `harness_provides` | **14 = 14**, equal |
| guard primitives, two sides | `tests/lib/audit-suites.sh --names skills/skill-audit/scripts/verdict-guard.sh \| wc -l` | **16 = 16**, equal |
| shipped skills, two sides | `skills/*/` glob; `git ls-files -- 'skills/*/SKILL.md'` | **2 = 2**, equal |
| go build | `cd profiler && go build ./...` | clean |
| go vet | `go vet ./...` | clean |
| gofmt | `gofmt -l .` | no output |
| go test | `go test ./...` | ok ×3 |
| go test -race | `go test -race ./...` | ok ×3 |
| top-level Go tests | `go test -list '.*' <pkg> \| grep -c '^Test'` | **286** = 239 + 35 + **12** |
| subtests under `-race` | `go test -race <pkg> -v \| grep -c '^=== RUN'` minus the top-level count | **1014** = 579 + 87 + **348** |
| coverage | `go test -cover ./...` | **97.9 / 94.7 / 95.9** |
| skill-validator | `skill-validator check skills/skill-audit`, `… skills/skill-rewrite` | exit 0 both; `-o json` → `"errors": 0, "warnings": 0, "passed": true` both |
| skillscore global install | `skillscore --version` | **2.0.2**, intact. No PATH mirror was built. |
| diff, whole guards round | `git diff --shortstat 619cab7..c3a565c` | 6 files, **+1873 −202** |
| diff, this lane only | `git diff --shortstat e2fb39d..c3a565c` | 3 files, **+151 −27** |

Pre-guards baseline at `619cab7`, re-derived here rather than quoted:
`tests/lib/audit-suites.sh tests/test_*.sh | tail -1` → **`sites=767 files=7 lines=10288
accounted=10288`**, and `tests/test_f01.sh` → **1886 passed, 0 failed**. So the report's
`sites=767 files=7` and `1886 passed, 0 failed` were both exactly right at the base. Go
figures (286 / 1014 / 97.9-94.7-95.9) are **unchanged** — this lane touched no Go.

---

## 5 · Prose the prose lane must move — every sentence this work falsifies

Only `CHANGELOG.md` and `RELEASE_NOTES.md`. `README.md` and `docs/profiler-spec.md` carry
none of these figures (`grep -nE 'assertion|shell suite|subtest|coverage|top-level|sites='`
finds only unrelated uses of "top-level" and "assertion"). Nothing here was edited.

**`CHANGELOG.md:367-370`** — "**2741 assertions across seven shell suites, 0 failed,
identical under bash 3.2.57 and bash 5.3.15**: `test_f01` 1886, `test_f02` 350,
`test_harness` 191, `test_rewrite` 158, `test_skill` 86, `test_install` 50, `test_walk` 20."
→ **3112**: `test_f01` **1912**, `test_f02` 350, `test_harness` **276**, `test_rewrite`
**401**, `test_skill` **101**, `test_install` **52**, `test_walk` 20. Six of the eight
numbers in that sentence are false. (`test_rewrite` 158 → 401 and `test_install` 50 → 52
were already falsified by the barrier lane.)

**`CHANGELOG.md:371-376`** — "**285 top-level Go test functions**, of which **284** pass and
one skips by design, and **696 subtests** pass under `-race` — 239 top-level in `profiler`,
35 in `profiler/cmd`, **11** in `profiler/internal/homesafe`; … with 87 in `profiler/cmd`
and **30** in `profiler/internal/homesafe`." → **286** (239 / 35 / **12**), **285** pass and
one skips, **1014** subtests (579 / 87 / **348**). 239, 35, 87 and 579 still hold.
(Barrier-lane movement, not this lane's; reported because it is what ships.)

**`CHANGELOG.md:386-393`** — "The assertion audit was guarded by `examined -ge 200` against
a real **761**. Driven at this release: injecting a reader limit into the audit takes it from
`sites=761 files=7 lines=9397 accounted=9397` to `sites=218 files=7 lines=2594
accounted=2594` … The audit now reports `lines` and `accounted` **per file**, each compared
against the other side (**`wc -l` over the seven suites is 9397, and the audit reports
9397**)". → Four things false. The audit no longer reads seven files but the **closure under
`source`**, nine of them: `sites=821 files=9 lines=11749 accounted=11749`. The `761`/`9397`
pair was the figure at `011defa`; the base of this round measured **767 / 10288**. The other
side is no longer `wc -l` but a second record count (`grep -ac ''`), for a stated reason. And
the account is incomplete: `sites` now also has an independent other side from outside the
file — bash's own record of the lines it ran an assertion on.

**`CHANGELOG.md:394-398`** — "its reader saw one of bash's **three** function-definition
spellings … **The reader now reads all three spellings**". → False. There is no list of
spellings: the reader reads the grammar, `name ( ) {` and `name (  ) {` were a live
fail-open that no list contained, and `tests/test_harness.sh` holds the reader against what
**bash** says each of 14 generated spellings defines. The same sentence should also record
that the harness-name list, the trap spellings and the set of files read were the other three
hand-enumerated sides, and that all four are now derived.

**`CHANGELOG.md:398-402`** — "Two more floors in the same file are exact counts now, and
**the blanket exemption that let three of them stand is an enumeration naming which floors
remain** and what two-directional comparison holds each one's set." → False. That
enumeration was itself stale within one release — it omitted
`.summary.policy_failures -ge 8`, stated a real value of 6 where the measured value was 5,
and generalised over `bcalls -gt bprobe`, which it did not cover. All four remaining floors
are **gone**, replaced by `-gt 0` beside a derived other side (and, for the policy count, by
an equality against the findings the same payload reports). What stands in its place is a
rule about shapes, not a list of lines.

**`RELEASE_NOTES.md:21`** — "**2741 assertions** across seven shell suites … **285**
top-level Go test functions — **284** passing and one skipping … **696 subtests** … `go
test -cover`: 97.9% for the profiler package, 94.7% for the CLI, **97.0%** for the
home-safety package." → **3112**; **286** (285 passing, one skipping); **1014**; and
home-safety coverage is **95.9%**, not 97.0%. The 0.4.3/0.4.2/0.4.1 rows at `:50`, `:71`
and `:98` are measured at tags and are unaffected.

Two sentences that remain **true** and should not be touched: "64 OTLP fixtures, against 59"
(`ls profiler/testdata/otlp/*.json profiler/testdata/otlp/*.ndjson | wc -l` → 64), and "Both
validate with zero errors and zero warnings, asserted by `tests/test_skill.sh`".
