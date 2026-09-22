# Barrier repair — round 2. Record. PR #22, `release/0.5.0`

Received head `011defa`. Two commits on top, both authored `imagineux
<imagineux@gmail.com>`, no `Co-Authored-By` and no "Generated with" line
(`git log 011defa..HEAD --format='%B' | grep -inE '^Co-Authored-By|Generated with
\[Claude|noreply@anthropic'` → no output; `git log 011defa..HEAD --format='%an
<%ae>' | sort -u` → one line, `imagineux <imagineux@gmail.com>`).

```
5d04f14  The bound stops being a table and starts being a generator, and goes red
619cab7  Containment is decided by identity, not by a name
```

Worktree: `…/scratchpad/bfix`, created with `git worktree add --force … release/0.5.0`.
A second throwaway worktree at base, `…/scratchpad/bfix-base` (detached at `10326b3`),
for the base-behaviour half of the F3 regression proof. Nothing was written in the
primary tree except this file. No `checkout`/`reset`/`clean`/`stash` in any tree but my
own; the two `git show 011defa:… > …` restores used for the revert-to-RED check were in
my own worktree and were undone from a copy taken first.

Lane 2's files are untouched: `git diff --name-only 011defa..HEAD` lists no
`CHANGELOG.md`, `RELEASE_NOTES.md`, `README.md` or `docs/profiler-spec.md`. Six files,
**+2155 −519** (`git diff --shortstat 011defa..HEAD`).

Pushed: `011defa..619cab7  release/0.5.0 -> release/0.5.0`. Not merged, not tagged.

**PR threads: there are none to reply to or resolve.** `gh api graphql … pullRequest(22)
{ reviewThreads }` returns `threads: 0`, and `gh pr view 22` shows `reviewDecision: ""` —
every finding this repairs came from the internal confirming pass the chief of staff
persisted at `.scuba/teams/release-0.5.0/confirm-barriers.md`, not from a review comment.
Nothing outside my mandate was touched.

---

## The root as I found it

The confirming pass named it and I re-derived it. It is one root, and it is not the one
the previous round fixed.

**All three barriers decided containment by comparing the bytes of a string they had
built from the caller's spelling, against the bytes of a string they had built for the
root — while the kernel performs the write on the caller's original spelling.** The
previous round replaced a lexical `..` fold with a component walk, which was the right
direction and did close the original escape. It left the comparison a name comparison.

A name is not an identity, and on this platform three independent mechanisms make it not
one. Each is measured, not assumed:

| mechanism | measurement |
|---|---|
| the volume is case-insensitive | `: > d/probe; [ d/probe -ef d/PROBE ]` → true |
| APFS is normalisation-insensitive | `mkdir "caf\303\251"; [ "caf\303\251" -ef "cafe\314\201" ]` → true |
| bash's **builtin** `pwd -P` returns the caller's spelling | `( cd -P -- "dir"$'\n' && p="$(pwd -P)"; [ "$p" -ef . ] )` → **false**: the captured value names a different entry |

That third row is two facts at once, and the second of them is F3. `$( )` strips every
trailing newline, so a resolved path carried back out of a shell function has lost the
last bytes of its leaf name. The path that was *decided* and the path that was *written*
were two different entries.

So: **F1, F2, F3 and F4 are one repair.** Stop resolving to a name.

---

## What I fixed

### The invariant, as implemented

> For every spelling `s` and root `r`, the verdict equals whether the object a write on
> `s` creates-or-truncates is the same filesystem object as `r`, or is reachable from `r`
> without leaving it — decided by identity for every component that exists, and by name
> only for components that do not yet exist.

It reduces to one question, and that reduction is the whole design: **the object a write
creates or truncates lives in the deepest directory the kernel reaches while resolving
the spelling**, so the verdict is whether *that directory* is the root or is under it.

- **Shell** (`skills/skill-rewrite/scripts/draft-rewrite.sh`, `tests/test_install.sh`).
  `__path_walk` performs the kernel's resolution as a sequence of `cd -P` — it *moves*
  rather than reading a spelling, so there is nothing for a spelling to change.
  `path_target_is_inside` then climbs from the destination's directory towards the
  filesystem root with `cd -P -- ..` and compares with `[ . -ef "$root_dir" ]`, which is
  device and inode. The whole of it is inside a subshell and **no path is ever carried
  back out**; the only values that cross the boundary are a status and, inside the
  subshell, `$PWD` — which is byte-exact because `cd` assigns it directly, and which is
  used only as a `stat` argument, never as a comparison operand.
- **Go** (`profiler/internal/homesafe/homesafe.go`). `resolve` returns a `target{dir,
  tail []string}` and `PathContains` compares with `os.SameFile`, climbing by popping
  components. `filepath.Rel`, `filepath.Dir` and `strings.HasPrefix` are gone from the
  file: `grep -n "filepath\.\(Clean\|Abs\|Join\|Rel\|Dir\|EvalSymlinks\)\|strings.HasPrefix"
  profiler/internal/homesafe/homesafe.go` returns comment lines only. That closes **F10**
  as a by-product rather than as a separate patch — the surviving textual folder is not
  commented as safe, it is absent.

`path_resolved`, `path_inside`, `path_absolute` and `path_normalized` no longer exist
anywhere, in code or in prose: `grep -rn "path_resolved\|path_absolute\|path_normalized"
tests/ skills/ docs/` → no output.

### The one place a name is still compared, and why it has three answers

A root that does not exist yet has no identity, so its components are compared as names.
That comparison is **three-valued**:

- the same bytes are the same name on every filesystem → inside;
- two names that are both ASCII and differ by more than case are different on every
  filesystem → outside;
- anything else has **no answer**, and every caller turns no answer into a refusal.

Two guesses were written before that rule, and both were wrong. Folding the case is what
a case-insensitive volume does; normalising is what APFS does. Neither can be right,
because **"this name may be the root's name" is a refusal for a fence that protects the
root and a pass for a fence that keeps writes inside it, and this one walk serves one of
each** — the drafter refuses a destination inside a protected directory, while
`tests/test_install.sh` runs a documented block only if it stays inside a scratch root.
No guess is fail-closed for both. Refusing is.

The middle answer is not an optimisation. A machine where `~/.devin` does not exist
reaches this comparison for every destination the drafter is ever given; answering "I
cannot tell" there turned the whole `-o` flag into exit 3, which is how I found out. It
was caught by the suite, not by review:

```
the destination @H/.claude/away/../climb-out.md: a write on that spelling lands outside
every protected directory, and the drafter refused it (exit 3) for a reason that is not
one of the three refusals it publishes
ERROR: the destination …/h/.claude/away/../climb-out.md cannot be shown to be outside the
live configuration directory …/h/.cursor, so where the draft would be written is unknown
```

This branch has **no production caller**: every root either consumer passes in exists —
`$HOME/.claude` and its five siblings, and a `mktemp -d` scratch directory. It is reached
only by the generated case set, which is deliberately wider than the consumers.

### F4 — the leaf symlink

`: > "$dest"` follows a leaf symlink, so a leaf link into a protected directory is a
write into that directory. Both shell copies appended an existing leaf as named while the
Go copy read it. Now all three follow it, and the mirror case (a leaf link *out* of the
root) is on the same generated list so the repair cannot be "follow the leaf and refuse".

The place this is observable is neither consumer's own surface, and that is the finding
behind the finding. The drafter refuses every leaf symlink by a published rule of its own
before containment is ever asked; `tests/test_install.sh`'s block does `mkdir -p`, which
fails on a link to a file and on a dangling link — which is exactly why the previous
round's oracle discarded those cases as *"the oracle could not create … so this case has
no verdict to compare against"*. So the suite gained a **second oracle over the same
generated operators**, `the_verdict_is_where_a_redirect_on_the_destination_lands`, which
performs a redirect instead of a `mkdir -p`. That is also the only place the `or-equal`
mode is exercised at all.

### F7, F8, F9 — and the reason they existed

- **F7**: an empty path now has no answer in all three. It used to return `$PWD` at
  status 0 in both shells while Go refused.
- **F8**: a root that resolves to `/` now contains everything in all three, which is the
  true answer for it. One shell copy accepted everything and the other refused
  everything. Neither root is ever `/`; `harness_scratch` is a `mktemp -d`.
- **F9**: `maxSymlinkHops`/`__path_max_links` is **32** in all three and no longer claims
  to bound "the way the kernel does". Measured: macOS refuses a chain at 32, Linux at 40.
  The counter is there to terminate a cycle, where `readlink` succeeds for ever; a chain
  the kernel refuses produces no write, so such a path has no object for a verdict to be
  about and refusing is the safe answer.

**These three were divergences between two copies nobody could diff**, and prose was what
was holding them together. So `tests/test_install.sh` now extracts the block between
`# --- BEGIN THE PATH CONTAINMENT BARRIER` and its `END` from both itself and the drafter
and `diff`s them, refusing if either extraction is empty or short. The previous round's
sentence *"All three are now the same walk"* is now mechanically true of the two shell
copies and held for the Go one by the shared generated case set.

I did **not** unify the two shell copies into one file. It crosses a boundary: the
drafter can only source `skills/skill-audit/scripts/verdict-guard.sh`, a shipped skill
surface, and the suite can only source `tests/lib/`, which the shipped drafter cannot
reach. Lane 1 surfaced that and recommended deciding it in 0.6.0; that recommendation
stands, and the byte-identity diff is the cheap half of its benefit taken now.

### A bug the generated bound found in this fix

`readlink` has no portable terminator. GNU writes the target followed by a newline; the
**BSD `readlink` on macOS writes the target and nothing else** — measured:

```
$ ln -s 'target'$'\n' link
$ readlink link | od -An -c   ->   t a r g e t \n      (no second \n)
$ stat -f '%Y' link | od -An -c ->  t a r g e t \n
```

My first byte-exact read was `raw="$(readlink -- "$1"; printf X)"; raw="${raw%X}";
target="${raw%?}"` — correct on Linux and eating a byte of the target on macOS. The
generated bound reproduced a live escape against it: a link whose *target* ends in a
newline, pointing at a file inside the root, was judged outside while a redirect on it
truncated a file inside. It is now `readlink -n --` plus the guard, which both
implementations have a flag for.

That is worth recording as evidence about the test rather than about the code: the case
set was generated, so it covered a mechanism I had got wrong in the fix itself.

---

## Scoped out, with reasons

- **F5 · TOCTOU.** Out of scope. Nothing re-checks between the decision at
  `output_is_permitted` and the write at the bottom of the script, and the window is the
  whole audit run. Closing it means holding the opened handle and verifying its `dev/ino`
  against the decision (`O_NOFOLLOW`, or writing through the same fd), which is a
  different fix direction from this one and changes how the script writes rather than how
  it decides. It is pre-existing at base. **No prose claims otherwise**: the shipped
  document says *"The refusals are a short list, not a sandbox"*
  (`skills/skill-rewrite/SKILL.md`), and `tests/test_install.sh:149-155` says under its
  own heading *"This is not a sandbox"* — *"a check in this file cannot be a security
  boundary against a hostile README, because anyone who can edit README.md can edit
  tests/test_install.sh in the same commit"*. The barrier is a net under an accidental
  edit, and both surfaces say so.
- **F11 · hardlinks.** Out of scope, and it is not a path-resolution question: a hardlink
  inside the protected root to a file outside means a write judged "outside" mutates
  content inside, and no barrier that reasons about paths can see it. I also did **not**
  put a hardlink in any fixture, deliberately — the confirming pass observed one
  corrupting its own real-write oracle, and a content-searching oracle would report a
  hardlinked file as "inside" for a write that landed outside. Covered by the same
  not-a-sandbox prose.
- **Unifying the shell copies into one file.** Surfaced, not done. See above.

---

## Is the case set generated? Yes, on two axes, in all three suites

The previous round's oracle design was right and its driver was wrong: nine, twelve and
twelve spellings written out by hand. A table is complete only about what its author
enumerated, and two whole classes sat outside all three.

All three now take the **product of two axes**:

1. **the arrangement of the root** — which protected root, whether it exists yet, and how
   the destination spells that root's own component (exactly, case-folded two ways, NFC,
   NFD);
2. **a list of mutation operators** — one line per mechanism by which a name can name
   something other than what it appears to.

So a seventh protected root, or a newly understood mechanism, is covered without anybody
remembering these files exist. The drafter's root list is not even written in the test:
`protected_roots_in_script` reads the script's own `protected_home_dirs` line.

Operators, as generated (each applied to every root arrangement): exact-inside, the root
itself, a not-yet-existing subdirectory, a one-character-longer sibling, a name-prefix
sibling, outside-plain, mid-path symlink in, symlink chain in, `..` crossing a symlink in,
`..` crossing a chain in, symlink out, `..` crossing a symlink out, `..` crossing a
*relative* symlink out, leaf symlink to a directory in and out, leaf symlink to a **file**
in and out, **dangling** leaf symlink in, a link whose own **name** ends in a newline at
each end, a link whose **target** ends in a newline, `..` folding past a missing
component, `..` climbing out past a missing component, doubled separators, `.`
components, trailing newline, trailing tab, trailing space, two trailing newlines,
non-directory mid-path (ENOTDIR), and the NFD-home spellings of several of the above.

The oracles are **identity oracles**, which the previous round's were not: the object the
write created is compared with the root by `-ef` / `os.SameFile` and located under the
root by content, with a `find` that does not follow the links the tree hangs out of the
root. A name-matching oracle agrees with a name-matching barrier about every case in
which both are wrong, which is how this class survived a round of exactly this test.

Three guards keep a generated set from passing by not running:

- **both directions must have happened.** A count of cases that landed inside and were
  judged inside, and of cases that landed outside and were judged outside, each asserted
  non-zero. This replaced a proportion-of-unperformable bound deliberately: the
  proportion is a different number on a case-sensitive filesystem — CI is
  `ubuntu-latest`, where a folded spelling names a directory that is not there — and a
  bound that has to be retuned per platform is a bound nobody trusts.
- **a case the kernel will not write on is still asked.** It used to be skipped before the
  barrier was called, which left the ENOTDIR arm of the walk unexercised. Now the barrier
  is asked, its answer recorded, and only the comparison against a landing place is
  skipped — there is no object for a verdict to be about.
- **"no answer" is bounded to where it is allowed.** It is tolerated only on the
  arrangements where the root does not exist, and it is never *required*, so a wrong
  definite answer still fails there. On every arrangement where the root exists — which is
  every arrangement either consumer can produce — there is no tolerance at all.

### What is still enumerated, stated plainly

- The **operator list** is a list. The axis it multiplies is generated, so a new root is
  free, but a new *mechanism* is still a line somebody has to write. That is the honest
  limit of this design: it converts "did the author think of this spelling" into "did the
  author think of this mechanism", which is a much smaller and much more reviewable
  question, and it is not zero.
- The **`mayRefuse` flag** on the root axis is an enumeration of one documented limit, not
  of cases.
- The drafter suite asserts the safety direction without qualification and the other
  direction **up to the three refusals the script publishes** — a destination that lands
  outside is accepted unless it is refused as a symbolic link, as a directory, or as a
  `SKILL.md`, and a refusal naming none of those is a failure. That is a real weakening
  and it is why the leaf-symlink proof lives in the redirect sweep instead.
- The non-existent-root axis is asserted **for safety only** on this platform's terms; on
  a case-sensitive filesystem a folded spelling of a non-existent root is answered
  "no answer" where the exact answer would be "outside". Fail-closed, no consumer.

---

## RED → GREEN → RED, per finding

Every row was run on `/bin/bash` 3.2.57 and on `/opt/homebrew/bin/bash` 5.3.15 and
returned the same verdicts.

### F3 — the regression this PR introduced

Driven end to end on the shipped drafter, with a leaf symlink into `~/.claude/skills`
whose own name ends in a newline (`scratchpad/repro/drafter.sh`):

| tree | rc | landed in `.claude` |
|---|---|---|
| base `10326b3` | **3** | no — refused, *"the destination … could not be resolved"* |
| head `011defa` | **0** | **yes — ESCAPE** |
| this branch | **1** | no — refused, *"… is a symbolic link, so where the draft would be written is not where it is named"* |

Base refused it, head accepted it, and this branch refuses it — at exit **1**, a refusal
of the caller's destination, which is a better answer than base's exit 3.

### F1 and F2 — all six protected roots

Same driver, `-o "$HOME/.<ROOT-IN-CAPITALS>/skills/ESCAPED.md"` and an NFD spelling of an
NFC home, over `.claude .cursor .codex .devin .config .agents`:

```
head 011defa                          this branch
root .claude  .CLAUDE  rc=0  ESCAPE   root .claude  .CLAUDE  rc=1  refused
root .cursor  .CURSOR  rc=0  ESCAPE   root .cursor  .CURSOR  rc=1  refused
root .codex   .CODEX   rc=0  ESCAPE   root .codex   .CODEX   rc=1  refused
root .devin   .DEVIN   rc=0  ESCAPE   root .devin   .DEVIN   rc=1  refused
root .config  .CONFIG  rc=0  ESCAPE   root .config  .CONFIG  rc=1  refused
root .agents  .AGENTS  rc=0  ESCAPE   root .agents  .AGENTS  rc=1  refused

root .claude  NFD home rc=0  ESCAPE   root .claude  NFD home rc=1  refused
 … all six, both ways …                … all six, both ways …
```

Controls held throughout: the exact spelling returns rc=1 for all six at head and on this
branch, and `$HOME/.claude-notes/x.md` is still accepted.

Go, same two findings, with a real write as the oracle: at head
`PathContains(home/.cursor, home+"/.CURSOR/skills/f1")` = `false` while the write landed
inside; on this branch the two agree.

### The generated suites

| suite | at head `011defa` | on this branch |
|---|---|---|
| `tests/test_rewrite.sh` | **352 passed, 30 failed** | **401 passed, 0 failed** |
| `tests/test_install.sh` (dir sweep) | 260 spellings, **46 disagreeing** | 300 spellings, **0 disagreeing** |
| `tests/test_install.sh` (suite) | **23 passed, 1 failed** | **52 passed, 0 failed** |
| `homesafe_test.go` | 300 subtests, **58 failing** | 330 subtests, **0 failing** |

The 30 in `test_rewrite` at head were 12 F3 (the trailing-newline leaf link, through two
spellings, × 6 roots), 6 F1, 6 F2 and 6 F1+F2 combined. The 58 Go failures grouped by
spelling: 7 `ROOT`, 7 `Root`, 7 `CAFé`, 15 `NOTYET`, 22 NFD — and **zero** in the
exact-ASCII arrangements, which is the shape the class has.

### Reverted, and RED again

Reverting the whole fix makes the byte-identity assertion fire first and halt the suite,
so each mechanism was reverted on its own, in **both** shell copies so that the identity
diff stays green and the sweep is red against the *defect* rather than against a mismatch:

| mechanism reinjected | result |
|---|---|
| `-ef` → compare `$PWD` against the root's resolved name (the head's semantics) | `test_install` **20 disagreeing**, `test_rewrite` **18 FAIL** |
| the leaf link appended as named instead of followed (the head's semantics) | `test_install` redirect sweep red on `@B/away/leaffile`, `@B/away/dangling` **and** `@B/root/outfile` — both directions |
| the symlink target read back through a plain `$( )` | `test_install` redirect sweep red on `@B/away/via-nl-target` |
| one byte changed in the drafter's copy of the block (`__path_max_links=31`) | `FAIL: the two shell copies of the containment barrier are one text`, with the diff printed |

Restored after each; `52 passed, 0 failed` every time.

---

## Numbers, each with the command that produced it

Run in `…/scratchpad/bfix` at `619cab7`.

**Shell assertions — 2986, identical on both shells, 0 failed.**

```
for B in /bin/bash /opt/homebrew/bin/bash; do
  for s in tests/test_*.sh; do printf '%-24s %s\n' "$s" "$($B "$s" 2>&1 | tail -1)"; done
done
```

| suite | at head | now |
|---|---|---|
| `test_f01.sh` | 1886 | 1886 |
| `test_f02.sh` | 350 | 350 |
| `test_harness.sh` | 191 | 191 |
| `test_install.sh` | 50 | **52** |
| `test_rewrite.sh` | 158 | **401** |
| `test_skill.sh` | 86 | 86 |
| `test_walk.sh` | 20 | 20 |
| **total** | **2741** | **2986** |

**Generated case counts**, printed by the suites themselves:

```
tests/test_install.sh  ->  the generated bound drove 300 spellings, 12 of which the kernel
                           would not perform a write on at all, 32 the barrier reached no
                           answer for, 100 agreeing inside, 156 agreeing outside, 0 disagreeing
                           the redirect bound drove 360 spellings, 156 … would not perform a
                           redirect on at all, 2 … no answer, 74 agreeing inside,
                           128 agreeing outside, 0 disagreeing
tests/test_rewrite.sh  ->  the generated bound drove 252 spellings: 36 unwritable,
                           144 refused for landing inside, 66 accepted for landing outside,
                           6 refused by a published destination rule
homesafe_test.go       ->  the generated bound drove 330 spellings: 12 unperformable,
                           34 with no answer, 122 agreeing inside, 162 agreeing outside
```

**Go tests — 286 top-level, 1014 subtests.**

```
go test ./... -list '.*' | grep -c '^Test'        -> 286   (239 + 35 + 12)
go test ./... -v 2>&1 | grep -c '^=== RUN .*/'    -> 1014
```

At head: **285** (239 / 35 / **11**) and **696**. The extra top-level function is
`TestPathContains_AChildShorterThanAParentThatDoesNotExist`, which covers the one arm of
the name comparison the generated product cannot reach, because every operator it builds
names something at or below the root rather than above it.

**Coverage.**

```
go test ./... -cover
  profiler                    97.9%   (published 97.9 — unchanged)
  profiler/cmd                94.7%   (published 94.7 — unchanged)
  profiler/internal/homesafe  95.9%   (published 97.0 — DOWN 1.1 points)
```

`go tool cover -func` on `internal/homesafe`: `PathContains` **87.5%**, `resolve`
**95.8%**, everything else 100%. Five statements are uncovered and each is an error arm
that cannot be provoked in-process, listed so nobody has to re-derive it:

| line | arm | why not provokable |
|---|---|---|
| 189 | `os.Stat` of the parent's own directory fails | the walk has already reached it; only a rename underneath the decision gets here |
| 199 | `os.Stat` of the child's directory fails | same |
| 215 | `os.Stat` of a directory on the climb fails | same |
| 372 | `os.Getwd` fails | **measured**: `t.Chdir` into a directory then `os.RemoveAll` it, and `os.Getwd` still returns `<nil>` — Go falls back to `$PWD` |
| 417 | `os.Readlink` fails after `Lstat` reported a symlink | a race between the two calls |

I raised it from the 92.7% the rewrite first landed at by two legitimate tests — the
ENOTDIR operator (which also covers `hasName`) and the short-child arm — and by folding
three duplicated stat-error arms into the shape they now have. I did not restructure
further to chase the published figure; the remaining five are genuinely unreachable and
reporting the number is the honest move.

**Other gates.**

```
gofmt -l profiler/                 -> (no output)
cd profiler && go build ./...      -> clean
cd profiler && go vet ./...        -> clean
cd profiler && go test -race -count=1 ./...  -> ok x3
skill-validator check skills/skill-audit     -> exit 0, passed
skill-validator check skills/skill-rewrite   -> exit 0, 0 errors 0 warnings (-o json)
skillscore --version               -> 2.0.2   (intact; no PATH mirror was built)
```

---

## Prose that is now false, reported and not rewritten

I did not touch `CHANGELOG.md`, `RELEASE_NOTES.md`, `README.md` or
`docs/profiler-spec.md`. Five statements in them are now false or no longer the whole
truth. Exact text and location:

1. **`CHANGELOG.md:254`** — *"**97.0%** for `profiler/internal/homesafe`"*. It is now
   **95.9%**, by `go test ./... -cover`. Same sentence in
   **`RELEASE_NOTES.md:21`** — *"97.0% for the home-safety package"*.

2. **`CHANGELOG.md:258-259`** — *"The two only functions in `internal/homesafe` short of
   100% are `PathContains` at **90.0%** and `resolve` at **96.8%**, by `go tool
   cover -func`."* Still the same two functions; the figures are now **87.5%** and
   **95.8%**.

3. **`CHANGELOG.md:367-370`** — *"**2741 assertions across seven shell suites, 0 failed,
   identical under bash 3.2.57 and bash 5.3.15**: `test_f01` 1886, `test_f02` 350,
   `test_harness` 191, `test_rewrite` 158, `test_skill` 86, `test_install` 50, `test_walk`
   20."* Now **2986**, with `test_rewrite` **401** and `test_install` **52**; the other
   five are unchanged. Same figure in **`RELEASE_NOTES.md:21`** — *"2741 assertions across
   seven shell suites"*.

4. **`CHANGELOG.md:371-372`** — *"**285 top-level Go test functions, of which 284 pass and
   one skips by design, and 696 subtests pass under `-race`** — 239 top-level in
   `profiler`, 35 in `profiler/cmd`, 11 in …"*. Now **286** (239 / 35 / **12**) and
   **1014** subtests. Same figures in **`RELEASE_NOTES.md:21`**.

5. **`CHANGELOG.md:261-262`** — *"**Containment is decided on a path whose symlinks are
   resolved *before* any `..` is folded, in all three implementations of that
   decision.**"* Not false, but it is no longer the property that holds the barrier up,
   and a reader who takes it as the whole guarantee gets the escape this round repaired:
   containment is now decided by **identity**, on no path at all. The bullet's later
   sentence *"All three are now one walk over the path's components in the order the
   kernel walks them"* was **false at `011defa`** — the two shell copies and the Go copy
   disagreed about a leaf symlink, about an empty path, and about a root resolving to `/`
   — and is now true of the two shell copies by a diff the suite runs, and of the Go copy
   by the shared generated case set.

**The one claim the task asked about — `CHANGELOG.md:273-276`** — *"The tests are pinned
to the invariant and not to the patch: no case names an expected verdict, each spelling is
performed twice over two trees — once as a real write, so the **filesystem** says where
the bytes landed, and once through the barrier — and the two must agree"*: **true at head
as far as it goes, and it is now stronger.** No case named an expected verdict then and
none does now. What was missing is that the *spellings* were a hand-written table, so the
sentence was true of every case in it and silent about the two classes that were not.
They are now a generated product, and the sentence's named witness
`TestPathContains_AgreesWithWhereTheWriteLands` still exists and is the generator.
`TestPathContains_DecidesAfterResolution`, the other witness it names, also still exists.
The sentence needs no correction; if Lane 2 wants it to carry the round's actual claim,
the phrase to add is that the case set is generated from the protected-root list and a set
of mutation operators rather than enumerated.

---

## Suites and shells

Every shell figure was produced under both `/bin/bash` 3.2.57 and
`/opt/homebrew/bin/bash` 5.3.15 and is identical under each, including the generated
counts and the mutation-revert results. Two specific 3.2 hazards were watched:

- **`set -e` does not fire on `[[ ]]` under 3.2.** The barrier's conditionals are `[ ]`
  and its status checks are `|| verdict=$?`, never a bare call followed by `case $?`.
  That is not theoretical: a bare call was the first version, and errexit made the
  *script* exit **2** — a status its own header does not register — for every destination
  the barrier could not decide. The status-registry assertion in `tests/test_rewrite.sh`
  caught it. The barrier's third status is therefore **3**, the number the drafter already
  registers for "no verdict was reached", which is the same statement one layer down.
- **`${text:i:1}`, `shopt -s nocasematch` and `[$'\200'-$'\377']` under `LC_ALL=C`** were
  each measured on both shells before being relied on.

One platform note for whoever reads this next. CI is `ubuntu-latest`, which is
case-sensitive, so the case-folded and NFD arrangements there name directories that do
not exist and the sweeps assert their (different, correct) answers. The macOS-specific
half of the invariant — that a folded spelling of an *existing* root is refused — is
therefore only exercised on a developer machine. It is exercised there on every run, and
a regression in it turns the sweep red, but CI will not catch it.
