# PR #8 — rebase onto `origin/main` and live re-verification

**Steward closeout.** Scope: rebase + re-verify only. Not merged, not tagged — the user merges.

| field | value |
| --- | --- |
| PR | #8 `fix/0.4.3-install-truth` |
| Base | `origin/main` = `2c155e7` |
| Head before | `63dc8c8` (34 commits) |
| **Head after** | **`c174113`** (35 commits) |
| Worktree | `scratchpad/wt-c6` (own worktree; primary tree never touched) |
| Mergeable | **MERGEABLE / CLEAN** against `2c155e7` (per-PR query, not list endpoint) |
| Review threads | **0** (`totalCount: 0`, `hasNextPage: false`) |
| CI on head | `test` → SUCCESS |

Commit count went 34 → 35: the 34 are preserved unsquashed and unreworded; the
35th is the README line the rebase surfaced (below). Author and committer are
`imagineux <imagineux@gmail.com>` on all 35. Zero attribution trailers
(`Co-Authored-By`, "Generated with", `noreply@anthropic` all grep to 0).

## Conflict resolutions — three files

### 1 + 2. `tests/lib/harness.sh` and `tests/lib/audit-suites.sh` — UNION, not a side

Each side added one name to the same two name-lists: this branch added the
refusal verb `require`, PR #10's exit-witness repair added `witness_exit`.
Both lists now carry **both** names.

`tests/lib/harness.sh`, `harness_ready()`:

```sh
  for n in harness_init harness_exit assert assert_value require quietly \
           witness_exit harness_summary; do
```

`tests/lib/audit-suites.sh` (the awk `harness_name` census):

```awk
  n = split("assert assert_value require quietly witness_exit harness_init harness_exit harness_summary harness_ready", list, " ")
```

Both names verified present in both lists at head `c174113`, and both are
defined in `harness.sh` (`witness_exit()` L104, `require()` L148). The union is
load-bearing rather than tidy: `witness_exit` has real call sites in
`tests/lib/masked-path.sh` and `tests/test_f01.sh`, so dropping it from the
census would have stopped the auditor detecting a suite's private copy, and
dropping it from the readiness check would have stopped a suite proving the
harness loaded it. Either loss is silent — suite still green, guard disabled.

### 3. `.github/workflows/ci.yml` — additive steps, count-free comment

The **steps auto-merged**; the conflict was only the hand-maintained comment.
Result has **seven** steps and seven suites: `test_harness`, `test_skill`,
`test_install`, `test_walk`, `test_rewrite`, `test_f01`, `test_f02`.

Two commits conflicted here:

- Commit 1/34 (`G6-03`) — its wording said "the five suites below". Took main's
  count-free wording, which states the same invariant this commit introduced.
- Commit 15/34 (`B5 — derive the suite count`) — took this branch's wording,
  which is a strict **superset** of main's: "the suites in `tests/` are exactly
  the ones named in this file and in the README" subsumes main's "holds this
  workflow to the suite glob", and it is count-free. Nothing from main is lost.

A hand-maintained count is exactly what drifted, so the surviving comment
carries no number.

**Parsing-hazard check (the failure mode the PR #10 rebase found):**
`workflow_suites()` greps `tests/test_[A-Za-z0-9_]+\.sh` over the *whole* of
`ci.yml`, comments included, and the per-suite `grep -qF -- "$suite" "$WORKFLOW"`
does too. A suite filename in a comment would inflate the census and force a
false step. Verified the resolved comment names **no** literal suite filename
(it says `tests/`, which the pattern does not match). Census = 7, glob = 7.

## The non-conflict the rebase surfaced — README

**Derived, not assumed.** Ran `tests/test_harness.sh` at the rebased head before
editing: exactly **one** failure, and it was exactly the predicted one.

```
FAIL: tests/test_rewrite.sh is in the README's list of the tests to run
74 passed, 1 failed
```

So the "only such step" claim holds at this head — re-established here rather
than carried from a record. Line added to the README's "Run the tests:" block,
in the order CI runs it (after `tests/test_walk.sh`):

```
tests/test_rewrite.sh
```

After: `75 passed, 0 failed`, rc=0. That is the 35th commit.

## Suite-count equality — holds at seven, reddens both ways

The derived check is an exact equality against the workflow, not a floor.
Confirmed it holds at seven, and that it still has teeth in **both** directions:

| direction | injection | result |
| --- | --- | --- |
| glob > workflow | added `tests/test_zzprobe.sh`, not a CI step | `FAIL: tests/ holds exactly the suites the CI workflow runs`, rc=1 |
| workflow > glob | added a CI step for `tests/test_phantom.sh` | `FAIL: tests/ holds exactly the suites the CI workflow runs`, rc=1 |

Both injections reverted; suite returns `75 passed, 0 failed`.

## Meta-check still has teeth — red then green

Injected a deliberately vacuous assertion at `tests/test_walk.sh:194`:

```sh
assert "this assertion cannot fail" true
```

`tests/test_harness.sh` reddened and named the file **and** the line:

```
tests/test_walk.sh:194: verdict is the constant command 'true', so this assertion cannot fail: assert "this assertion cannot fail" true
sites=21 files=1
FAIL: tests/test_walk.sh has no assertion that cannot fail, and no harness of its own
74 passed, 1 failed   (rc=1)
```

Removed; back to `75 passed, 0 failed` (rc=0), `tests/test_walk.sh` byte-identical
to its committed form. Assertion call sites examined across the seven suites: 605.

## Live suite results — all seven, both shells

Measured at head `c174113`. **Identical on both shells, 2478 passing, 0 failing.**

| suite | bash 3.2.57 | bash 5.3.15 |
| --- | --- | --- |
| `test_f01.sh` | 1856 | 1856 |
| `test_f02.sh` | 350 | 350 |
| `test_harness.sh` | 75 | 75 |
| `test_install.sh` | 48 | 48 |
| `test_rewrite.sh` | 94 | 94 |
| `test_skill.sh` | 35 | 35 |
| `test_walk.sh` | 20 | 20 |
| **total** | **2478** | **2478** |

### The arithmetic — why 2478 and not 983 + 2405

Both baselines re-measured here rather than quoted, since records pinned to
mid-flight heads have been stale twice this epic:

- PR #8 pre-rebase `63dc8c8`: **983** across six suites (measured).
- `origin/main` `2c155e7`: **2405** across six suites (measured). The brief's
  2307 was stale; 2405 is the live figure, and `test_rewrite` is 94, not 70.

The two totals do **not** add, because the branches *share* five suites rather
than contributing separate ones. PR #8 touches only `audit-suites.sh`,
`harness.sh`, `test_harness.sh`, `test_install.sh`, `test_walk.sh` — it never
touches `test_f01`, `test_f02`, `test_skill`, `test_rewrite`, so main's newer
copies of those simply supersede the older ones PR #8's 983 was counting.

Decomposition of 2478:

```
  2335   untouched by PR #8, taken from main: f01 1856 + f02 350 + skill 35 + rewrite 94
+   20   test_walk — 20 on both sides; PR #8 edits it but adds no assertion
+   48   test_install — PR #8's genuinely new suite, absent from main
+   75   test_harness — main's 50, grown by PR #8's +25
= 2478
```

Equivalently, against main's total: **2405 + 48 (new suite) + 25 (harness growth) = 2478.**

The 983 superseded: `983 − 48 (install) − 70 (its harness) = 865` = f01 575 +
f02 243 + skill 27 + walk 20, all replaced by main's 2261 for the same suites.

## Go

At `c174113`, in `profiler/`:

- `go build ./...` — rc=0
- `go vet ./...` — rc=0
- `go test -race -count=1 ./...` — both packages `ok` (uncached; the cached run was re-run to make "green" measured)
- `gofmt -l .` — no output

## Cluster behaviour re-proven (a merge can undo it)

All three, live at the rebased head:

1. **Documented destination still works** — `PASS: the README's block names a
   destination this suite can expand, which resolves inside the harness scratch
   root`, plus `the block's destination is the path the README's own list gives`.
2. **Outside-root destination still refused, decoy intact** — the containment
   decision and the resolved verdict each refuse the absolute tail, the tilde
   with a user name, both climbs, the symlink route and the not-yet-existing
   path. The decoy `/Users/decoy-operator/.claude/skills` **does not exist** on
   the real filesystem; no `ESCAPED-THE-SCRATCH-ROOT` anywhere real; the live
   `~/.claude/skills` is untouched and the global `skillscore` install is intact.
3. **Shape pass is still a diagnostic** — `PASS: the path-word diagnostic admits
   blocks that leave the scratch root, as documented`, over exactly **four**
   admitted blocks: a backslash before a leading slash; a brace expansion; a
   climb composed from two bound single dots; a command naming no path at all.
   Asserting admission (not refusal) is what keeps the prose and the mechanism
   coupled.

## Left alone, deliberately

- **Three P3 items knowingly shipping**, per `final-8-10.md`: the false
  durability clause in test prose (C7-3r), the unread `rm` status on the
  drafter's EXIT trap exit path (A5), and the fact that one arrived in the round
  that closed its own class. Not touched.
- **No version surface touched.** The four `plugin.json` files appear in the
  branch diff only for an added `author` field; `"version": "0.4.2"` is
  unchanged on both sides, so the 0.4.3 adapter bump and the `jq` release-note
  line remain owed by the release commit.
- **History not tidied.** The commit whose message under-describes its change is
  left as-is — it is pushed, a later commit explains the removal, and rewriting
  beyond the rebase risks more than it fixes.
- Force-push used `--force-with-lease=fix/0.4.3-install-truth:63dc8c8…`, never
  bare `--force`. Linear history preserved.

## Disposition

- Resolved: 0 threads (PR #8 has none — confirmed live, not assumed).
- Deferred: the 3 P3 items above, reason recorded in `final-8-10.md`.
- Routed to `bug-fixer`: nothing. The rebase surfaced no REAL bug — the single
  red was the README list, which is disposition, not root-cause repair.

**Blocked on:** nothing mechanical. PR #8 is MERGEABLE/CLEAN with CI green.
Merge is the user's.
