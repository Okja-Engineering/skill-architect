# PR #9 closeout + PRs 7/8/9 conflict map

Steward closeout. Written 2026-09-18. Two mandates: unblock PR #9's CI (Task 1) and
establish the 7/8/9 integration map by trial merge (Task 2).

**No merge and no tag were performed. The user merges.**

## Pinned state

| Fact | Value |
| --- | --- |
| PR | #9 `fix/0.4.3-skill-rewrite` -> `main` |
| Head before | `d85ea7d193ca382c3848635c18744963170f985a` (14 commits) |
| **Head after** | **`483903a659c49fae23a68f45184158de058b65c1`** (15 commits) |
| Base at time of closeout | `origin/main` = `71cc8667ba2aac1550ebb0dce8418b81a1bd7319` |
| CI at new head | **success** — run `35387632344`, all 19 steps green, read live |
| `mergeable` / `mergeStateStatus` (per-PR, live) | `MERGEABLE` / `CLEAN` (was `BLOCKED`) |
| Review threads (live, paginated) | `totalCount` 0, nodes returned 0, `hasNextPage` false |
| Reviews / issue comments | 0 / 0 — `reviewDecision` null |

Rebase: **not required.** PR #9 was already current — `origin/main` has not moved off
`71cc866` since the branch forked, and the branch merges clean against it. Confirmed by
`git merge-tree --write-tree` rather than assumed.

Thread tally is live-verified, not taken from the control-plane claim: `totalCount` equals
nodes returned and `hasNextPage` is false, so the zero is a real zero and not a page-boundary
early stop. Nothing to triage, route, resolve or defer. PR #9's findings arrived through the
control plane, as stated.

## Task 1 — the CI blocker

### Confirmed diagnosis

Read live from the failing run `35387125895` at `d85ea7d1`:

```
/home/runner/work/_temp/....sh: line 1: tests/test_rewrite.sh: Permission denied
##[error]Process completed with exit code 126.
```

`tests/test_rewrite.sh` was committed **100644**; every other suite in `tests/` is 100755.
`ci.yml` invokes each suite as a bare path (`run: tests/test_rewrite.sh`), so the step died
at exit 126 in 9s, before one assertion ran.

### One correction to the brief

The brief said it passed locally "because the worktree copy was executable." It was not.
On-disk mode in the lane worktree was **600** — not executable either. The lane's local pass
came from invoking through `bash tests/test_rewrite.sh`, which needs no x bit. The
file-writing helper wrote mode 0600 (`.github/workflows/ci.yml` is also 600 on disk); git
records only the x bit, so 600 lands as 100644. Same root cause, different mechanism than
described: nothing ever had the bit, rather than the bit being dropped on commit.

### The fix

Commit `483903a`, mode-only:

```
:100644 100755 a96bfe5 a96bfe5 M	tests/test_rewrite.sh
```

Blob hash identical on both sides (`a96bfe50b236b04281c2efaa2ce9b5fe94430372`) — content
untouched. On-disk mode also corrected. `bash -n` clean. Authored and committed
`imagineux <imagineux@gmail.com>`; trailer grep for co-author / generated-with / AI
attribution returned zero across all 15 commits before pushing.

### Full mode audit

PR #9's complete add/modify set is six files. All correct except the one fixed:

| File | Mode | Verdict |
| --- | --- | --- |
| `.github/workflows/ci.yml` | 100644 | correct |
| `skills/skill-rewrite/SKILL.md` | 100644 | correct |
| `skills/skill-rewrite/scripts/draft-rewrite.sh` | 100755 | correct (`test_skill.sh:121` asserts `-x`) |
| `tests/fixtures/rewrite/all-sections/SKILL.md` | 100644 | correct |
| `tests/fixtures/rewrite/all-sections/scripts/run.sh` | 100755 | cosmetic inconsistency — see below |
| `tests/test_rewrite.sh` | 100644 -> **100755** | **the defect, fixed** |

**No other wrong mode blocks anything.** Three near-misses, all examined and cleared:

1. `tests/fixtures/rewrite/all-sections/scripts/run.sh` is 100755 while its pre-existing
   twin `tests/fixtures/f01/valid-full/scripts/run.sh` — byte-identical, blob `588829f` —
   is 100644. Inconsistent in the *opposite* direction from the bug. Harmless: fixture
   scripts exist so path-resolution checks can find them and are never executed; nothing
   asserts `-x` on a fixture. Left alone as out of scope for a mode-only push.
2. `skills/skill-audit/scripts/verdict-guard.sh` is 100644 while its five siblings are
   100755. Correct as-is — it is *sourced* as a shared guard, not executed. Pre-existing
   on `main`, not in PR #9.
3. `tests/lib/harness.sh` and `tests/lib/masked-path.sh` are 100644 while
   `tests/lib/audit-suites.sh` is 100755. Correct — the first two are sourced, the third is
   a runnable entry point. Pre-existing on `main`.

**The helper hypothesis does not generalise.** PR #8's new suite `tests/test_install.sh`
came out 100755, and PR #7 adds only 100644 JSON testdata. The defect is isolated to this
one file.

### Recommended follow-up (not done here — needs a lane, not a mode fix)

Nothing in the repo asserts that the suites themselves are executable. `test_skill.sh`
asserts `test -x` on *skill* scripts (lines 90, 92, 121), and `test_harness.sh` already
walks the suite glob asserting each suite is a step in `ci.yml` (lines 109-111) — but no
check covers the x bit on the suites. That is why CI was the first thing to notice, at the
cost of a full round trip. The root repair is one added assertion inside `test_harness.sh`'s
existing per-suite loop, where the glob is already in hand. Recommend routing to a lane.

### Verification at the new head

Every suite in PR #9's tree, both bash versions, run in the lane worktree. CI's order.

| Suite | bash 3.2.57 | bash 5.3.15 | Lane reported |
| --- | --- | --- | --- |
| `test_harness` | 50 passed, 0 failed | 50 passed, 0 failed | 50 |
| `test_skill` | 27 passed, 0 failed | 27 passed, 0 failed | 27 |
| `test_walk` | 20 passed, 0 failed | 20 passed, 0 failed | 20 |
| `test_rewrite` | 70 passed, 0 failed | 70 passed, 0 failed | 70 |
| `test_f01` | 575 passed, 0 failed | 575 passed, 0 failed | 575 |
| `test_f02` | 243 passed, 0 failed | 243 passed, 0 failed | 243 |

All `rc=0`. **985 assertions per bash, 1970 total, zero failures. The lane's
50/27/20/70/575/243 reproduces exactly on both.**

Go gate in PR #9's tree: `go build ./...` OK, `go vet ./...` OK,
`go test -race -count=1 ./...` OK (`profiler` 1.431s, `profiler/cmd` 1.836s),
`gofmt -l .` empty. PR #9 touches no Go, as expected.

## Task 2 — the integration map

Established by trial merge and rebase in throwaway worktrees at base `71cc866`. All four
throwaways have been removed. **No resolution was pushed anywhere.**

Baseline re-confirmed: each of #7, #8, #9 merges **clean** against `origin/main` on its own.
All three now pass CI and read `MERGEABLE` / `CLEAN` per-PR.

### Conflict matrix

| Pair | Result | Files | Size |
| --- | --- | --- | --- |
| **#7 x #9** | **CLEAN** | — | — |
| **#8 x #9** | CONFLICT | `.github/workflows/ci.yml` | 1 hunk, 5 vs 6 lines, **YAML comment only** |
| **#7 x #8** | CONFLICT | `README.md` | 1 hunk, 30 vs 22 lines of prose |
| | | `profiler/claude_code.go` | 1 hunk, 9 vs 45 lines, **semantic** |

`profiler/types.go` auto-merges between #7 and #8 despite both adding to it (+12 / +28).

### #8 x #9 is purely additive — and the harness premise in the brief is wrong

The brief expected both PRs to change `tests/test_harness.sh`'s per-suite floor and asked
whether the two disagree. They cannot disagree, because **PR #9 does not touch
`tests/test_harness.sh` at all.**

The suite list there is a **glob** guarded by a floor assertion. On `main` the floor is
`-ge 5`. PR #8 raises it to `-ge 6` (it adds `test_install.sh`). PR #9 adds
`test_rewrite.sh` and changes nothing — the glob picks the new suite up for free, and 6 >= 5
still holds. Evidence that the glob is live: `test_harness` reports 50 assertions on PR #9
alone and **54** on the combined tree, because the loop gained a seventh suite.

So the only conflict is one YAML comment: PR #8 hard-codes a prose count ("the **five**
suites below"), PR #9 rewrites the same comment to be count-free ("the suites below ...
holds this workflow to the suite glob"). **Take PR #9's wording** — it is strictly better,
since it does not need editing again on the next suite. Both new CI steps auto-merge; the
merged workflow carries all seven suites in the right order with no intervention.

Verified by resolving it in a throwaway and running the combined tree — **all seven suites
green on bash 5**: harness 54, skill 27, install 8, walk 20, rewrite 70, f01 575, f02 243.
All seven suites 100755 in the merged tree.

**One latent gap the merge does not fix.** After both land there will be **7** suites but the
floor will still read **6**, whatever PR #8 set. It passes (7 >= 6) but the guard under-counts:
a suite could be deleted and the denominator check would stay green — the exact failure that
assertion exists to refuse. **Whoever merges second should bump the floor 6 -> 7.** This is
not automatic and no conflict will prompt for it.

### #7 x #8 is NOT additive — it is a Go signature conflict

`README.md` is an editorial union. Both sides extend the same "Anything else is `none`"
paragraph in different directions — #7 documents session scoping, #8 documents
probe/capture error reporting — and both sides end on a shared sentence about partial
exports that must be de-duplicated. Mechanical resolution is not available; it needs a
prose decision.

`profiler/claude_code.go` is the real problem. The two PRs change `resolve()` incompatibly:

- `main`: `func (a ClaudeCodeAdapter) resolve() otelSignals`
- **#7**: `func (a ClaudeCodeAdapter) resolve(prov provenance) otelSignals`, all callers updated
- **#8**: leaves the signature alone and adds a **new** no-arg caller `a.resolve()` inside a
  new `ProbeWithDiagnostics()`, plus a `ProbeDiagnoser` interface method in `types.go`, a
  `cmd/main.go` call site, and `probe_diagnostics_test.go`

The natural "keep both sides" union **does not compile**. Proven in the throwaway:

```
./claude_code.go:229:9: not enough arguments in call to a.resolve
	have ()
	want (provenance)
```

The correct resolution is a **one-line** thread-through — teach #8's new probe path to pass
#7's probe-scoped provenance:

```go
// in ProbeWithDiagnostics
sig := a.resolve(a.probeProvenance())   // was: a.resolve()
```

`probeProvenance()` is right on the merits: #7 defines it as "the same test with the session
half dropped", `Probe()` is not session-scoped, and `cmd/main.go`'s `probe` command takes no
`--session`. Verified in the throwaway: **build OK, vet OK, `go test -race -count=1 ./...`
OK** on both packages. Then discarded.

### Contested surface per branch — why order matters

| Branch | Commits | `ci.yml` | `README.md` | `claude_code.go` | `test_harness.sh` |
| --- | --- | --- | --- | --- | --- |
| #7 provenance | 6 | — | 1 | 2 | — |
| #8 install-truth | 9 | 1 | **6** | 1 | 1 |
| #9 skill-rewrite | 15 | 1 | — | — | — |

PR #8 has the broadest contested surface: `README.md` in 6 of its 9 commits. PR #9 is the
narrowest — one commit touching one file, and no collision with #7 at all.

### Rebase simulation (measured, every ordering)

| Scenario | Result |
| --- | --- |
| rebase #9 onto main+#7 | **CLEAN** |
| rebase #7 onto main+#9 | **CLEAN** |
| rebase #9 onto main+#8 | stops once — `ci.yml`, 1 comment hunk |
| rebase #8 onto main+#9 | stops once — `ci.yml`, 1 comment hunk |
| rebase #8 onto main+#7 | stops — `README.md` 1 hunk + `claude_code.go` 1 hunk, replaying 7 contested commits |
| rebase #7 onto main+#8 | stops — `README.md` 1 hunk + `claude_code.go` 1 hunk, replaying 3 contested commits |

The #7/#8 collision is unavoidable in either direction. What the order controls is **which
branch absorbs it**, and therefore how many commits get replayed against moved text.

### Recommended merge order: #9, then #8, then #7

1. **#9 first.** It is green right now at `483903a`, verified live, and costs the other two
   nothing: #7 rebases onto it perfectly clean, and #8's only cost is one YAML-comment hunk.
   Land the thing that is already cleared.
2. **#8 second.** Rebase onto main+#9 stops once on that one comment hunk — take #9's
   count-free wording. **Bump the harness floor 6 -> 7 in this same rebase**, since #8 owns
   that line and will be the seventh suite's landing point.
3. **#7 last.** It absorbs the #7/#8 collision, which is correct on two counts: the conflict
   *is* #7's own `resolve(provenance)` signature change, so #7's lane already knows where to
   thread it; and #7 replays 3 contested commits against a moved `README.md`/`claude_code.go`
   versus #8's 7 — less than half the rework. The fix is the one-line
   `a.resolve(a.probeProvenance())` above plus the README prose union.

Putting #8 *before* #7 is the load-bearing half of this. Reversing it (#7 before #8) forces
PR #8's six README-touching commits to replay, which is the highest-rework ordering. The
worst orderings are #9,#7,#8 and #7,#9,#8: both make PR #8 absorb the README/Go conflict
*and* the `ci.yml` comment in one rebase.

## Disposition ledger

| Item | Disposition |
| --- | --- |
| `tests/test_rewrite.sh` mode 100644, CI exit 126 | **RESOLVED** — `483903a`, mode-only; CI green live |
| Review threads on PR #9 | **NONE** — 0 live-verified, paginated, nothing to triage |
| Fixture `run.sh` 100755 vs twin's 100644 | **INVALID as a defect** — fixtures are never executed; noted only |
| `verdict-guard.sh` / `tests/lib/*` 100644 | **INVALID** — sourced, not executed; correct as-is |
| No suite asserts the suites are executable | **ROUTED / DEFERRED** — out of scope for a mode-only push; recommend a lane add one assertion to `test_harness.sh`'s existing glob loop |
| Harness floor stays 6 with 7 suites after #8+#9 | **ROUTED** — assign to whoever rebases second (#8 under the recommended order) |
| #7 x #8 `claude_code.go` signature conflict | **ROUTED to PR #7's lane** — one-line fix identified and verified, not applied or pushed |
| #7 x #8 `README.md` prose union | **ROUTED to PR #7's lane** — needs an editorial decision |
| Merge of #9 | **NOT DONE — the user merges.** Brief withheld merge and tag authority |

## Constraint compliance

- Primary working tree never touched — still on `main` with its 47 uncommitted 0.5.0 entries.
  No checkout, reset, clean, stash or rebase there. Only reads, `gh`, and
  `git worktree add/remove` (metadata only).
- Task 1 in the lane's own worktree `skill-architect-wt/c8-skill-rewrite` (clean at `d85ea7d`
  on entry). Task 2 in four throwaway worktrees under the session scratchpad, all removed.
- Pushed exactly one commit to exactly one branch: `d85ea7d..483903a` on
  `fix/0.4.3-skill-rewrite`. Fast-forward, no force needed. `origin/fix/0.4.3-install-truth`
  and `origin/fix/0.4.3-profiler-provenance` are untouched. No resolution pushed.
- `imagineux <imagineux@gmail.com>` author and committer. Zero AI-attribution trailers,
  grep-verified before push.
- No PATH mirror, no stubs, nothing installed or reconfigured on this host.
- The tracked-file change was a mode bit via `git update-index --chmod=+x` — no content edit,
  so no `Write`/`Edit` against a tracked non-markdown path and no heredoc-written file.
  This report written with the `Write` tool.

## Open loop for the manager

The external reviewer has produced **zero** reviews on PR #9 at any head — not a stale
review, none ever. `reviewDecision` is null. If the setup expects an external pass before
merge, PR #9 has not had one, and the green CI plus the 1970 live assertions are currently
the only gate it has cleared. Flagging rather than reading the silence as approval.
