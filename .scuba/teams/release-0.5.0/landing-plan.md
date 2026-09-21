# 0.5.0 landing plan — how the unlanded work reaches `main`

**Author** architect · **Date** 2026-09-19 · **Status** plan, for manager QA
**Measured against** `origin/main` = **`a06af3f`** ("0.4.3 C8: bring skill-rewrite up to
skill-audit's conventions (#9)"), fetched 2026-09-19. Fork point of the unlanded work =
**`30f374c`**. Every number below is as of those two commits and will move as PRs #7, #8
and #10 land; each number states how to re-derive it.

> **Superseded in part — see §0.5 (G0 re-derivation, 2026-09-20).** 0.4.3 has shipped.
> `origin/main` is now **`5b60fca`**. §1.1 and §1.4 have been re-run against it and the
> results recorded below. Read §0.5 before trusting any figure in §1.

---

## 0.5. G0 re-derivation — measured 2026-09-20 against `5b60fca`

Run by the S1 implementer as the gate requires. **`origin/main` = `5b60fca`** ("release:
v0.4.3 — a profile reports the session it names, and every claim answers to something that
runs"). Fork point unchanged at `30f374c`.

### Gate preconditions — all met

| Precondition | Result |
|---|---|
| 0.4.3 fully merged | Yes — PRs #7, #8, #9, #10, #11, #12 all merged; **no PR open** (`gh pr list --state open` → `[]`) |
| `AdapterVersion` reaches `"0.4.3"` | **Yes.** `git show origin/main:profiler/types.go` → `const AdapterVersion = "0.4.3"`. **R1 is closed.** |
| `ProfileSchema` | `"skill-architect/profile/v1"`, unchanged — D5 holds |

**The merge order the plan predicted was wrong, harmlessly.** Plan said "#9 done → #8 → #7;
#10 after #9". Actual: #9 (`a06af3f`) → #10 (`2c155e7`) → #8 (`505a34b`) → **#11**
(`8342b21`) → #7 (`be5c0aa`) → **#12** (`5b60fca`, the release). PRs **#11 and #12 were not
in the plan at all** — #11 is a C5 prose correction, #12 the release commit. Four merges
landed after `a06af3f`, not three.

### §1.1 re-derived — the committed side

```
comm -12 <(git diff --name-only 30f374c wip/0.5.0-profiler | sort) \
         <(git diff --name-only 30f374c origin/main | sort)
```

| Figure | Plan (`a06af3f`) | Now (`5b60fca`) | Moved? |
|---|---|---|---|
| wip diffstat `30f374c..0a83615` | 2,534 ins / 118 del, 12 files | **2,534 / 118, 12 files** | no (wip is frozen) |
| **Colliding files** | 7 | **7** | **no** |
| Disjoint (wip-only) | 5 | **5** | no |
| Denominator: files `main` changed since `30f374c` | 94 | **106** | **+12** |

**R3 did not materialise. The collision set did not grow.** The plan predicted it would,
because #7/#8/#10 each touched files on the list — but every file they touched was *already*
colliding, so the intersection stayed at 7 while the denominator grew 94 → 106. The
collision set is now stable: `wip` is frozen and 0.4.3 is closed, so it cannot grow again
before the slices land.

Colliding set, with the `main` commits that have touched each since `30f374c` (new commits
in **bold**):

| Colliding file | `main` commits since `30f374c` |
|---|---|
| `profiler/claude_code.go` | `2ed34b8`, `bd32b26`, **`505a34b`**, **`be5c0aa`** |
| `profiler/cmd/main.go` | `2ed34b8`, **`2c155e7`**, **`505a34b`**, **`be5c0aa`** |
| `profiler/profiler_test.go` | `2ed34b8`, `bd32b26`, `a90f632`, **`be5c0aa`**, **`5b60fca`** |
| `skills/skill-rewrite/SKILL.md` | `a06af3f`, **`2c155e7`** |
| `skills/skill-rewrite/scripts/draft-rewrite.sh` | `71cc866`, `a06af3f`, **`2c155e7`** |
| `tests/test_skill.sh` | `2ed34b8`, `5c847e1`, `a90f632`, `71cc866`, **`2c155e7`**, **`8342b21`**, **`5b60fca`** |
| `tests/test_walk.sh` | `71cc866`, **`505a34b`** |

Disjoint, unchanged: `profiler/{codex,compare,cursor,devin,experiment}.go`.

### §1.4 re-derived — the uncommitted side

```
comm -12 <(git diff --name-only | sort) <(git diff --name-only 30f374c origin/main | sort)
```

| Figure | Plan | Now | Moved? |
|---|---|---|---|
| Tracked modified | 21 files, 687 ins / 175 del | **21, 687 / 175** | no |
| Overlapping with `main`'s changes | 13 | **13** | no |
| Untracked paths | 26 | **26** | no |
| Untracked files (recursive) | — | **210** | new figure |
| Untracked `profiler/*.go` | 2,230 lines / 12 files | **2,230 / 12** | no |
| `skillgate/` | 139 files | **139** | no |
| `docs/research/` | 40 files | **40** | no |
| `profiler/queries/` | 8 files | **8** | no |
| `skills/skill-gate/` | 1 file | **1** | no |

**Nothing on the uncommitted side moved**, because nobody edited the primary tree between
the plan and the gate. That is the good outcome D3 exists to protect, and it is now frozen
(see §0.6).

### G0 risk check — the rewrite suite and its fixtures **survived**

The plan flagged that PRs #7 and #8 showed `tests/test_rewrite.sh` and
`tests/fixtures/rewrite/` as *deletions*, and that if it were real, 0.5.0 would inherit a
deleted suite. **It was stale-base noise, exactly as the plan judged, and it resolved.**

| Check at `5b60fca` | Result |
|---|---|
| `tests/test_rewrite.sh` present | **Yes** |
| `tests/fixtures/rewrite/` present | **Yes** — 2 files (`all-sections/SKILL.md`, `all-sections/scripts/run.sh`), the same 2 as at `a06af3f` |
| Suite runs and reports | **Yes — 85 assertions, 0 failed**, on both bash 3.2 and bash 5 |
| All seven suites present in `tests/` | Yes |

**One number to correct in the plan:** §3/S9 calls `test_rewrite.sh` "1,047 lines on
`main`". It is **939 lines** at `5b60fca`. The 108-line reduction is deliberate and
attributable — `2c155e7` (C1/C3/C4) and `8342b21` (C5, "claims that derive instead of
restating") rewrote 166 lines out and 58 in. **No assertion was lost**: the suite reports 85
at runtime. S9 should cite 939.

### Suite and test totals at `5b60fca` (the S1 baseline)

| | bash 3.2 | bash 5 |
|---|---|---|
| test_harness | 75 | 75 |
| test_skill | 53 | 53 |
| test_install | 48 | 48 |
| test_walk | 20 | 20 |
| test_rewrite | 85 | 85 |
| test_f01 | 1868 | 1868 |
| test_f02 | 350 | 350 |
| **Total** | **2499 passed, 0 failed** | **2499 passed, 0 failed** |

Go: `go build`, `go vet`, `gofmt -l .` (empty) all clean; `go test -race -count=1 ./...`
green — **93 top-level tests, 545 PASS lines including subtests, 0 failures.**

## 0.6. The D3 snapshot — taken 2026-09-20, frozen

| | |
|---|---|
| Snapshot | `<scratch>/snapshot-0.5.0/` |
| Manifest | `<scratch>/snapshot-0.5.0.manifest` (sha256, 15,452 lines) |
| Repo-content copy of manifest | `.scuba/teams/release-0.5.0/snapshot-0.5.0.repo-files.manifest` (78 lines) |
| Total files | **15,452** |
| **Of which actual repo content** | **78** |
| Of which `.venv-skillspector/` | **15,374 (99.5%)** |
| Read-only | Yes — `chmod -R a-w`, write attempt refused |
| Excluded paths absent | Verified, all 11 |

`<scratch>` = `/private/tmp/claude-501/-Users-matthewvandusen-Development-Auraprix-skill-architect/31edaf48-6b19-4833-82e7-6cb52dac691f/scratchpad`.

**D3's exclusion list has a hole: `.venv-skillspector/`.** Run literally, the rsync copies a
195 MB / 15,374-file Python virtualenv into the snapshot, so 99.5% of both the snapshot and
the manifest is noise and D3's own "confirm the file count" check tells a reader nothing.
**Recommendation before S2: add `--exclude '.venv*/'` to D3.** S1's `.gitignore` already
declares that path out of scope for the repo, so the exclusion is consistent with the
release, not a new decision. The snapshot as taken is still correct and usable — the 78
repo files are intact and hash-verified — so this is a cleanup, not a re-take.

**D3's provenance check works.** S1 used it: `AGENTS.md` landed byte-identical to the
snapshot and its sha256 matched the manifest line
(`a77f5a98d1199e202391368293e75092ed2b9853afcb04a89db8e7e97ba38e8d`).

---

## 0. Headline

**This is not a rebase, and it is not a cherry-pick either.** I trialled all seven
`wip/0.5.0-profiler` commits onto `a06af3f` in a scratch worktree: **7 of 7 conflict.**
Two of the five "disjoint" files do not even compile against current `main` — for two
independent reasons, each introduced by a release the branch predates.

The route is **re-apply, file by file, against the post-0.4.3 contract**, taking content
from the *working tree* (not from the wip commits — see D2), through ten small PRs.

Three bodies of work in the unlanded set should **not land at all**: the Codex and Devin
adapters, the skill-rewrite script vendoring, and four documentation edits that current
`main` has already superseded with better text. Deleting them is the cheapest correct
outcome and costs no capability.

---

## 1. Re-derived measurements

### 1.1 The committed side — `wip/0.5.0-profiler` @ `0a83615`

Seven commits, `30f374c..0a83615`: **2,534 insertions / 118 deletions across 12 files.**
Confirms the manager's figure.

**Colliding files: seven, not four.** Denominator walked: all 12 files the seven commits
touch, intersected with the 94 files `origin/main` changed over `30f374c..a06af3f`.

| Colliding file | `main` commits that touched it since `30f374c` |
|---|---|
| `profiler/claude_code.go` | `2ed34b8` (v0.4.1), `bd32b26` (#3) |
| `profiler/cmd/main.go` | `2ed34b8` (v0.4.1) |
| `profiler/profiler_test.go` | `2ed34b8`, `bd32b26`, `a90f632` (v0.4.2) |
| `skills/skill-rewrite/SKILL.md` | `a06af3f` (0.4.3 C8, PR #9) |
| `skills/skill-rewrite/scripts/draft-rewrite.sh` | `71cc866` (C7), `a06af3f` (C8) |
| `tests/test_skill.sh` | `2ed34b8`, `5c847e1`, `a90f632`, `71cc866` |
| `tests/test_walk.sh` | `71cc866` (C7) |

The manager measured four. The extra three are `skill-rewrite/SKILL.md`,
`skill-rewrite/scripts/draft-rewrite.sh` and `tests/test_walk.sh` — PR #9 merged as
`a06af3f` *after* that measurement, and C7 (`71cc866`) touched `test_walk.sh`. **The
collision set grew by 75% in one merge.** It will grow again: PR #10 touches
`skill-rewrite/SKILL.md`, `draft-rewrite.sh` and `test_skill.sh`; PR #8 touches
`test_walk.sh`; PR #7 touches `claude_code.go`, `cmd/main.go` and `profiler_test.go`.
Assume all seven stay in collision after 0.4.3 and re-run the command below at G0.

Re-derive:
```
git fetch --all --prune
comm -12 <(git diff --name-only 30f374c wip/0.5.0-profiler | sort) \
         <(git diff --name-only 30f374c origin/main | sort)
```

**Disjoint on the committed side: five files** — `profiler/{codex,compare,cursor,devin,experiment}.go`.
None exists on `main`, and none is touched by PR #7, #8 or #10.

### 1.2 Cherry-pick trial — 7 of 7 fail

In `scratchpad/wt-landing` (detached at `a06af3f`), each commit was cherry-picked
`--no-commit`, its conflicts recorded, then aborted:

| Commit | Conflicts |
|---|---|
| `fd2cfe5` Claude Code session_data | `claude_code.go`, `cmd/main.go`, `profiler_test.go` |
| `b73326d` skill-rewrite patch | `skill-rewrite/SKILL.md`, `draft-rewrite.sh`, `test_skill.sh`, `test_walk.sh` |
| `1e4e845` Cursor/Codex/Devin adapters | `claude_code.go`, `cmd/main.go`, `profiler_test.go` |
| `f13173e` skill-rewrite `/tmp` fix | `skill-rewrite/SKILL.md` |
| `e0e2a52` F04 compare | `cmd/main.go`, `profiler_test.go` |
| `236152c` F04 experiment design/plan | `cmd/main.go` |
| `0a83615` F04 experiment run | `cmd/main.go`, `experiment.go` |

Note the last row: `0a83615` conflicts on `profiler/experiment.go`, a *new* file, because
its parent `236152c` did not apply. The commits are not independently pickable even from
each other.

**Conclusion: cherry-pick is dead for all seven. Denominator: 7 of 7 trialled.**

### 1.3 The adapters do not compile — measured twice

I copied the three adapter files onto a clean `a06af3f` worktree and ran `go build ./...`.

Wip-branch versions:
```
./codex.go:126:12:  undefined: otelMetric
./codex.go:127:12:  undefined: otelLog
./codex.go:164:16:  undefined: toInt
./codex.go:194:16:  undefined: getString
./cursor.go:161:12: undefined: otelMetric
```
Stubbing those six symbols back in to reach the next layer:
```
./codex.go:164:4:  invalid operation: tc.Input += toInt(v) (mismatched types *int and int)
./cursor.go:203:5: invalid operation: tc.Input += val     (mismatched types *int and int)
```
Working-tree (later) versions, same test:
```
./codex.go:176:7:   tc.CacheWrite undefined (type TokenCounts has no field or method CacheWrite)
./compare.go:298:49: undefined: EstimatedTokensResult
```

Two independent breakages, one per release:
- **v0.4.1** deleted the entire bespoke-envelope layer. `otelMetric`, `otelLog`, `toInt`,
  `getString`, `parseTime` all existed inside `claude_code.go` at `30f374c`
  (lines 147, 153, 251, 266, 273) and exist nowhere on `main`. `main` even ships a fixture,
  `profiler/testdata/otlp/bespoke_envelope.json`, whose job is to prove that shape is now
  *rejected*.
- **v0.4.2** made every `TokenCounts` field a `*int` so "unread" and "zero" are different
  answers. `tc.Input += val` is not portable to that; it is the exact fabrication v0.4.2
  removed.

The worktree was restored to a clean `a06af3f` and rebuilt green after each experiment.

### 1.4 The uncommitted side

**Tracked: 21 modified files, 687 insertions / 175 deletions.**
**Overlapping with `main`'s changes since `30f374c`: thirteen, not eleven** — the manager's
eleven plus `skills/skill-rewrite/SKILL.md` and `skills/skill-rewrite/scripts/draft-rewrite.sh`,
again from PR #9. Re-derive: `comm -12 <(git diff --name-only | sort) <(git diff --name-only 30f374c origin/main | sort)`.

**Untracked: 26 paths.** Of those, `profiler/*.go` is 2,230 lines across 12 files
(`analyze`, `doctor`, `hooks`, `hooks_install`, `spool_profile`, `compare_test`,
`strict_test` and their tests). `skillgate/` is **139 files**, not ~30;
`skills/skill-gate/` is 1. `docs/research/` is 40 files. `profiler/queries/` is 8.

**Coverage I actually walked, stated as a denominator:**

| Set | Denominator | Read in full | Read as diffstat / listing only |
|---|---|---|---|
| wip commits | 7 | 7 (message + stat); 3 adapter files line-by-line | 4 (content sampled via targeted grep) |
| tracked modified | 21 | **12** | **9** — `claude_code.go`, `cmd/main.go`, `codex.go`, `compare.go`, `cursor.go`, `devin.go`, `experiment.go`, `profiler_test.go`, `README.md` |
| untracked paths | 26 | `NOTICE`, `go.work`, 5 Go file headers, dir listings | `docs/research/` (40 files), `profiler/queries/` (8), 7 Go test files, `skillgate/` (139), `skills/skill-gate/` (1) |
| open PR branches | 3 | PR #7 `types.go` + `claude_code.go`; PR #8 `types.go`; PR #10 `skill-audit/SKILL.md` | PR #7 `provenance.go` (268 lines) and `otlp.go` changes **unread**; PR #8 and #10 read as diffstat |

**Everything below is sound on what I walked. The nine diffstat-only tracked files and
PR #7's `provenance.go` are the places where my account is thinnest; each slice that
touches them re-reads them in full as its first step.** That is written into the slices.

---

## 2. Decisions

### D1 — Route: re-apply as new work. Do not rebase, cherry-pick, or merge `wip`.

**Taken:** treat `wip/0.5.0-profiler` as *prior art*, not as a landing path. Tag it
`archive/0.5.0-wip` (annotated, pushed) so it stays addressable, then land the content as
fresh commits on branches cut from post-0.4.3 `main`.

**Rejected — rebase `wip` onto `main`.** Same seven conflicts (§1.2), plus every
intermediate commit would be a non-building tree, so no rebase step could be verified.
Rebasing a branch that was never pushed loses nothing, but it also gains nothing: the
conflict work is identical and the result is a fake history claiming the code evolved that
way.

**Rejected — merge `wip` into `main`.** Produces a merge commit. The repo's ruleset
requires linear history, re-verified at `71cc866` ("zero merge commits"). Not available.

**Constraint check:** nothing here rewrites published history or force-pushes a shared
branch. `wip/0.5.0-profiler` was never pushed; the tag is additive.

### D2 — The working tree, not the wip commits, is the content of record.

Local `main` is at `0a83615`, which *is* `wip/0.5.0-profiler`'s head, so the uncommitted
diff sits directly on top of the wip work. For every file present in both, the working-tree
version is the **later revision** — `compare.go` is +233/-… over its wip form, `cursor.go`
+116, `experiment.go` +9. An implementer who ports from the wip commits ports a superseded
draft.

**Rule: port from the working tree. Cite the tag only for history.**

### D3 — Snapshot the working tree once. Never check out over it.

The primary tree holds the only copy of ~2,900 lines of unlanded work, half of it
untracked. No slice may `checkout`, `reset`, `clean`, `stash` or `rebase` there.

**Mechanism:** before slice 1, copy the tree once to a read-only snapshot:

```
rsync -a --exclude '.git/' --exclude '.scuba/' --exclude 'tmp/' \
      --exclude 'skillgate/' --exclude 'skills/skill-gate/' \
      --exclude 'docs/skillgate-*' --exclude 'go.work*' --exclude 'NOTICE' \
      --exclude 'docs/research/' \
      <primary>/ <scratch>/snapshot-0.5.0/
( cd <scratch>/snapshot-0.5.0 && find . -type f -print0 | xargs -0 shasum -a 256 ) \
  > <scratch>/snapshot-0.5.0.manifest
chmod -R a-w <scratch>/snapshot-0.5.0
```

Every slice copies files *out of* the snapshot into its own worktree and checks the file's
hash against the manifest before editing. That gives three things: the primary tree is
never touched, every slice draws from one frozen state, and a slice can prove it took the
content it meant to. **Rejected alternative:** `git stash create` — it does not capture
untracked files, which is where 2,230 of the 2,900 lines live.

### D4 — Keeping `skillgate/` out: three independent barriers, not one habit.

A blanket commit sweeps in 140 files. One rule that everyone must remember is the failure
mode; three mechanical barriers is the fix.

1. **Exclusion at the snapshot.** `skillgate/`, `skills/skill-gate/`, `docs/skillgate-*.md`,
   `go.work`, `go.work.sum`, `NOTICE`, `docs/research/` are excluded by the rsync in D3, so
   they are **not present** in any writer's worktree. A writer cannot commit a file that is
   not there.
2. **Explicit path lists, never `git add -A`.** Each slice below names its exact `git add`
   argument list. `git add -A` and `git commit -a` are forbidden for the whole release.
3. **A pre-commit assertion in every slice's checklist:**
   ```
   git diff --cached --name-only \
     | grep -E '^(skillgate/|skills/skill-gate/|docs/skillgate|go\.work|NOTICE$|docs/research/)' \
     && { echo "OUT-OF-SCOPE PATH STAGED"; exit 1; }
   ```
4. **`.gitignore` hardening lands first** (slice S1): add `go.work`, `go.work.sum`,
   `skillgate/`, `skills/skill-gate/` and `.scuba/` to `.gitignore`. That turns barrier 2
   from discipline into enforcement even in the primary tree, where the files *do* exist.
   (`.scuba/` is already in the uncommitted `.gitignore` diff and is correct: the control
   plane is mirrored to the `scuba-state/…` branch, not committed to `main`. Verified: `.scuba`
   is absent from `git ls-tree origin/main`.)

Note the ordering consequence: **S1 must be the first PR**, because it is what makes
barriers 2 and 4 real for everything after it.

### D5 — Schema stays `profile/v1`. `AdapterVersion` moves to `0.5.0`, once.

**`ProfileSchema` = `"skill-architect/profile/v1"`, unchanged at 0.5.0.**
Rule applied: v1 stays while a v1 reader is merely *ignorant* of a new key; v2 is required
when a v1 reader would be *wrong*. Everything 0.5.0 adds — `ToolCallEntry.{ErrorType,Count,ID}`,
`Attribution.{Category,Detail,OperationName,Confidence}`, `estimated_context_tokens` — is
optional and absent when unmeasured, so a v1 reader is ignorant, not wrong.

**One condition makes that false, and it must be fixed in slice S3.** The uncommitted
`types.go` declares:
```go
EstimatedContextTokens EstimatedTokensResult `json:"estimated_context_tokens,omitempty"`
```
`omitempty` **does nothing on a struct in Go.** As written, *every* profile — including one
that estimated nothing — emits `"estimated_context_tokens":{"state":""}`, and `""` is not a
member of `{present, unknown, error}`. That is a v1 reader being wrong, and it would force
v2. **Make the field `*EstimatedTokensResult` and the problem disappears.** S3 carries a
test that a profile with no estimate has no such key.

**Rejected — bump to `profile/v2`.** It invalidates every stored profile in the same
release that first ships `compare`, the tool whose entire job is reading stored profiles.
Maximum churn, zero reader benefit, since the additions are optional.

**`AdapterVersion` = `"0.5.0"`, set exactly once, in slice S3.** The repo's own rule
(`types.go`): bump whenever the adapter changes what a profile contains for the same input.
0.5.0 does that twice over — `skill_activation` goes from `unknown` to `present` for an
export 0.4.3 reported as `unknown` (S4), and the hook spool produces profiles from a source
no prior version could read (S7). Bumping per slice would make `adapter_version`
un-interpretable; bumping once, early, in the slice that also changes the type surface,
keeps it a statement about the release.

**⚠ Precondition that is not mine to fix.** `AdapterVersion` is `"0.4.2"` on **all four**
of `origin/main`, `fix/0.4.3-profiler-provenance`, `fix/0.4.3-install-truth` and
`fix/0.4.3-body-guard-exits`, verified 2026-09-19. The 0.4.3 worklist names the bump to
`"0.4.3"` as its definition of done (worklist §642, §689). PR #7 changes what a profile
contains for the same input and still says `0.4.2`. **If 0.4.3 ships without its bump, a
0.4.3 profile is permanently indistinguishable from a 0.4.2 one, and no 0.5.0 slice can
repair that retroactively.** Escalated in §7.

**`cache_creation` → `cache_write` is DROPPED (user decision).** The uncommitted diff
performs that rename in `profiler/types.go` and `docs/profiler-spec.md`, and `codex.go` and
`cursor.go` consume it. **Those hunks must be reverted, not ported.** Acceptance check on
every 0.5.0 branch: `grep -rn 'cache_write\|CacheWrite' profiler/ docs/ README.md` returns
nothing. The compile error `tc.CacheWrite undefined` measured in §1.3 is the marker; if it
reappears, the rename came back.

**Related trap in the same file:** the uncommitted `types.go` diff is computed against the
*pre-0.4.2* file, so it also shows `Input int` where `main` has `Input *int`. **Applying
that file wholesale silently reverts v0.4.2's zero-vs-unread fix.** S3 must hand-port the
additions onto `main`'s `types.go`, never replace it.

### D6 — The three adapters: none of them lands. Two are deleted.

Answered in full in §4. This is the `subtract-before-you-add` outcome: ~640 lines removed
from the landing set, zero capability lost, and the README's "Planned" markers stay true.

### D7 — Reintegration, not addition: the contract gets a test, not more prose.

The reason three adapters could be written that lie about what they measure is that the
adapter contract lived in **prose** (`docs/profiler-spec.md`) and in **one adapter's
implementation** (`claude_code.go`). Nothing mechanical stood between a new adapter and a
false capability claim. The adapter-honesty lens found exactly that (F1, 335
signal-outcomes over the one adapter that exists).

So slice **S2 refactors the contract into a table-driven test over the adapter registry**
before any new capture path is added. That refactor is *authorized by this plan*: the
implementer of S2 is expected to extract the AC9 probe/capture agreement, the session-id
refusal and the pointer-count discipline into enforcement that iterates every registered
adapter, rather than adding a fourth hand-written adapter beside three hand-written ones.
Without S2, slices S7–S8 bolt a second capture source onto a contract that cannot check it.

---

## 3. The slicing

**Conventions honoured:** one writer per branch; never open a draft PR; squash-merge only
(linear history); branch names follow `main`'s existing pattern (`feat(...)` / `fix(...)` /
`release:` subject lines, `0.5.0 Sn:` prefix for slice commits). No agent merges to `main`.

**One gate, then ten PRs.**

### G0 — Gate (not a PR, not mine to execute)

0.4.3 finishes: PRs #8, #7, #10 merge in the recorded order (#9 done → #8 → #7; #10 after
#9), and `AdapterVersion` reaches `"0.4.3"`.

**No 0.5.0 slice may be cut before PR #7 is on `main`**, because every profiler slice must
be written against the provenance-scoped `resolve(prov)` read path, the `scopedExport` type
and `SessionIDRequiredError`. Cutting earlier guarantees a second round of the same
conflicts.

At G0, **re-run §1.1's `comm` command and §1.4's, and record the new numbers in this file.**
My seven and thirteen are as of `a06af3f` and will be wrong by then.

Also at G0: note that PR #7 and PR #8 currently show `tests/test_rewrite.sh` and
`tests/fixtures/rewrite/` as *deletions* against `main` — that is stale-base noise from
branching before PR #9, not intent. It resolves on their rebase. If it does not, 0.5.0
inherits a deleted 1,047-line suite.

---

### S1 — Boundary and hygiene · `fix/0.5.0-boundary-hygiene`

**Lands:** `.gitignore` (add `.venv*/`, `.scuba/`, `go.work`, `go.work.sum`, `skillgate/`,
`skills/skill-gate/`), `AGENTS.md` (the four Operating principles), `.out-of-scope.md`
(**only** the "Run live paired comparisons end-to-end" bullet).

**Explicitly excluded:** the `.out-of-scope.md` "Certify skills safe" bullet — it names
`skillgate` and is 0.6.0 content. Leave `main`'s "Guarantee security" bullet alone.

**`git add`:** `.gitignore AGENTS.md .out-of-scope.md` — three paths, nothing else.

> **⚠ IMPLEMENTER FINDING, 2026-09-20 — the third path was NOT landed. It moves to S5.**
>
> The bullet S1 was told to land reads: *"The profiler captures runtime signals and
> `profiler compare` (F04) reports paired differences; scheduling and executing the paired
> runs remains the caller's job."* **`profiler compare` does not exist at `5b60fca`.** The
> CLI dispatches `probe|capture|version|help` only; `git ls-tree -r 5b60fca` has no
> `compare.go`. Landing that sentence in S1 puts a present-tense capability claim in the
> repo six slices before the capability, and it contradicts two lines in the same commit:
> `README.md:612` ("the comparison engine is not yet built") and `README.md:303` ("future
> paired comparisons (F04)").
>
> This is the same defect §6.1 rejects the `CHANGELOG`/`RELEASE_NOTES` edits for — asserting
> a capability the source says does not exist — so the plan's own standard decides it. §6.2
> marks this row "—" (no correction required); **that is the error.** The bullet is not
> wrong, it is *early*: it becomes true when S5 lands `compare`.
>
> **Resolution: S1 landed two paths (`.gitignore`, `AGENTS.md`). The bullet is deferred to
> S5**, which already edits `README.md` and `docs/profiler-spec.md` for the comparison
> contract and is the slice that makes the sentence true. Add `.out-of-scope.md` to S5's
> `git add` list. Manager to ratify.

**Why first:** it installs barriers 2 and 4 of D4. Every later slice depends on it. It is
also the smallest possible end-to-end exercise of the pipeline (snapshot → worktree →
explicit add → squash), so if the process is wrong it is wrong cheaply.

**Verified by:** all seven shell suites + the Go job, all unchanged, so green *is* the
proof of no regression. Plus `git show --stat` naming exactly three files.

**Disjointness:** `AGENTS.md` is untouched by `main` since `30f374c` (verified:
`git log 30f374c..origin/main -- AGENTS.md` is empty) and by all three open PRs.
`.gitignore` and `.out-of-scope.md` likewise.

---

### S2 — The adapter contract, enforced · `feat(profiler)/0.5.0-adapter-contract`

**Lands:** no new capability. A `profiler/adapter_contract_test.go` that iterates the
harness registry and asserts, for every registered adapter:
1. `Capture("")` returns `SessionIDRequiredError` (PR #7's rule, generalised off
   `ClaudeCodeAdapter`).
2. **AC9 holds**: every signal `Probe()` advertises as a source other than `none` is
   `present` in the `Capture()` of a session the export contains — and every signal it
   advertises `none` for is not `present`. Both directions.
3. `TokenCounts` fields are `*int`, and a token result built from an export carrying only
   cache counts has **no** `input`/`output` keys (v0.4.2's invariant, tested at the
   contract rather than in one adapter).

Plus a section in `docs/profiler-spec.md` stating the three obligations, and a
deliberately-lying stub adapter *inside the test file* proving the table can fail.

**`git add`:** `profiler/adapter_contract_test.go docs/profiler-spec.md`
(+ `profiler/types.go` only if the registry needs exporting).

**Why this exists:** D7. This is the integration step, not an addition. It is what makes
S7–S8 safe and what would have caught the Devin adapter in §4 mechanically.

**Verified by:** `go test -race -count=1 ./...`; a **mutation proof** — the stub adapter
that over-advertises must turn the suite red, and its removal green. A green run alone
proves nothing here; that is the defect the test exists to catch, and the repo has learned
this lesson twice already (worklist G7-01).

Also: `tests/test_walk.sh` — it walks documented claims, and the spec gains claims.

---

### S3 — Profile type surface for 0.5.0 · `feat(profiler)/0.5.0-profile-surface`

**Lands, hand-ported onto `main`'s `types.go` (never by replacing the file — D5):**
`ToolCallEntry.{ErrorType,Count,ID}`; `Attribution.{Category,Detail,OperationName,Confidence}`;
`SourceHooksEstimated`; `EstimatedTokens` + `EstimatedTokensResult` and its two
constructors; `Profile.EstimatedContextTokens` as **`*EstimatedTokensResult`**;
`SignalStates()` extended to six; `AdapterVersion = "0.5.0"`; matching
`docs/profiler-spec.md` edits.

**Reverted from the uncommitted diff:** the `CacheCreation` → `CacheWrite` rename in both
`types.go` and `docs/profiler-spec.md`; the non-pointer `TokenCounts` fields.

**`git add`:** `profiler/types.go profiler/profiler_test.go docs/profiler-spec.md`

**Verified by:** `go test -race -count=1`, with four new tests that are the acceptance
criteria of this slice:
- A profile that estimated nothing has **no** `estimated_context_tokens` key at all
  (this is the `omitempty`-on-struct trap, D5).
- `SignalStates()` is **derived from, and asserted equal to, the set of result-typed fields
  on `Profile` by reflection** — not a hand-written list of six. `main`'s own comment on
  `SignalStates` says it exists so a caller "cannot walk four of them and believe it walked
  the set"; a hand-maintained six-entry literal recreates the defect one field later.
- Round-trip (AC6) over a profile carrying every new key.
- `grep -rn 'cache_write\|CacheWrite' profiler/ docs/ README.md` is empty.

**Collides with:** `profiler/types.go` (PR #7 and PR #8 both add to it), `profiler_test.go`,
`docs/profiler-spec.md`. Rebase after G0 and re-read all three before editing.

---

### S4 — Claude Code reads `skill.name` activation · `feat(profiler)/0.5.0-skill-activation`

**Why it is in scope and not already done:** `main`'s `claude_code.go:205` currently says
`UnknownActivationResult("This adapter does not yet read Claude Code's skill telemetry: …
carries skill.name, invocation_trigger, skill.source and skill.kind. **Reading it is
0.5.0.**")` — the repo has explicitly deferred this to this release. `main` already ships
the fixture `profiler/testdata/otlp/skill_name_present.json`.

**Lands:** activation extraction through PR #7's provenance projection (so an activation is
scoped to the named session like every other signal); the `docs/profiler-spec.md` AC2/AC4
updates the uncommitted diff already drafts; README capability row.

**`git add`:** `profiler/claude_code.go profiler/profiler_test.go profiler/testdata/otlp/… docs/profiler-spec.md README.md`

**Depends on:** S3 (one `AdapterVersion` bump, already made) and PR #7.

**Verified by:** `go test -race`; fixtures covering present / absent / foreign-scope
`skill.name`; the S2 contract table (activation is now a signal it checks both directions
of); `tests/test_walk.sh` for the README and spec claims.

**Collides with:** `profiler/claude_code.go` — PR #7 rewrites `resolve()` here. Read PR #7's
`provenance.go` in full first; it is one of my thinnest coverage points (§1.4).

---

### S5 — `profiler compare` · `feat(profiler)/0.5.0-compare`

**Lands:** `profiler/compare.go` + `profiler/compare_test.go` (251 lines, from the snapshot),
the `compare` CLI subcommand, and the **Comparison contract** section of
`docs/profiler-spec.md` (`skill-architect/comparison/v1`).

**Integration requirement, not optional:** `compare` must **refuse a comparison between
profiles with different `capability.adapter_version`**, and say both versions in the reason.
That is the entire purpose of `AdapterVersion` and `compare` is its first consumer. Without
it, 0.5.0 ships a comparator whose first real use — a stored 0.4.x profile against a fresh
0.5.0 one — reports the reader's four releases of fixes as the skill's regression. The
adapter-honesty lens measured exactly this failure at 50% on one fixture (F2). It must also
refuse a cross-`source` delta and carry both sides' sources on every `MetricComparison`, as
the uncommitted spec draft already states.

**`git add`:** `profiler/compare.go profiler/compare_test.go profiler/cmd/main.go profiler/cmd/main_test.go docs/profiler-spec.md README.md .out-of-scope.md`

> **Added 2026-09-20 by S1:** `.out-of-scope.md` joins this list. The "Run live paired
> comparisons end-to-end" bullet was assigned to S1, but it asserts that `profiler compare`
> reports paired differences — true only once *this* slice lands. See the finding in S1.
> S5 must also update `README.md:612` ("the comparison engine is not yet built") and
> `README.md:303` ("future paired comparisons"), which become false in the same commit.

**Verified by:** `go test -race` (`compare_test.go`);
**⚠ verification gap** — the CLI half does not exist. `profiler/cmd/main_test.go` is 191
lines on `main` and `profiler/cmd` coverage is 12.6% (deferred-ledger L21). **This slice
must write the CLI-level tests for `compare`; it may not rely on `compare_test.go` alone.**

**Collides with:** `profiler/cmd/main.go` (PR #7 and PR #8 both touch it; the roadmap
already records a `resolve()`-signature collision between #7 and #8 at `cmd/main.go`).

---

### S6 — `profiler experiment` (design / plan / run) · `feat(profiler)/0.5.0-experiment`

**Lands:** `profiler/experiment.go` + the `experiment` subcommand with `plan` and `run`.

**Depends on:** S5 — `run` produces the profiles `compare` reads, and the two must agree on
the comparison contract.

**`git add`:** `profiler/experiment.go profiler/experiment_test.go profiler/cmd/main.go profiler/cmd/main_test.go docs/profiler-spec.md README.md`

**Verified by:** `go test -race`.
**⚠ verification gap** — **there is no `experiment_test.go` anywhere** in the untracked set
or on the wip branch as a separate file; the wip branch's coverage of it is inside
`profiler_test.go`'s 766 added lines, which I did not read line by line. **The implementer
must first measure what coverage actually exists rather than assume the 766 lines cover
this**, then write what is missing. Do not accept "the wip branch had tests" as a proof.

---

### S7 — Cursor hook spool: capture only · `feat(profiler)/0.5.0-hook-spool`

**Lands:** `profiler/hooks.go`, `hooks_install.go`, `spool_profile.go` and their three test
files; the `ingest` and `hooks install|uninstall` subcommands.

**Does NOT land:** any `CursorAdapter`. See §4.

**Naming:** the spool schema constant is currently `"cursor-profiler/spool/v1"` — a
namespace from the sibling `../cursor-profiler` repo, not this product. Rename to
`"skill-architect/spool/v1"` in this slice, before anything writes a file with it.
This is cheap now and a migration later.

**`git add`:** `profiler/hooks.go profiler/hooks_install.go profiler/spool_profile.go profiler/{hooks,hooks_install,spool_profile}_test.go profiler/cmd/main.go profiler/cmd/main_test.go docs/profiler-spec.md README.md`

**Boundary discipline already present — preserve it.** `InstallHooks(home, command)` and
`UninstallHooks(home, command)` take `home` as a parameter; `DetectEnvironment(home, getenv)`
takes both. Only `DefaultSpoolDir()` reads the real `os.UserHomeDir()`, and only as a
default. That is the right shape. **The risk is in the CLI wiring**, where the real home is
supplied: PR #8's gate already caught "a live-directory destroyer" in this repo. Tests must
prove that no test invocation touches the real `~/.cursor` or `~/.cursor-profiler`.

**Verified by:** `go test -race`; an idempotence test (`InstallHooks` twice = one entry);
a foreign-hook-preservation test; a test that `~` is never written under test.
**⚠ Cannot be verified against a real Cursor.** Cursor is not installed on this machine
(session memory: E-CS research lives in `../cursor-profiler`; Cursor not installed locally).
Every hook-payload shape this code parses is `[DOCS]`, not `[OBSERVED]`. That is the same
evidence label the repo already flags as a known limit (deferred-ledger L24). **Manager
decision required — §7, risk R2.**

---

### S8 — `profiler analyze` and `doctor` · `feat(profiler)/0.5.0-analyze-doctor`

**Lands:** `profiler/analyze.go`, `doctor.go`, their tests, `profiler/queries/` (8 SQL
files + README), the `analyze` and `doctor` subcommands.

**Depends on:** S7 (both read the spool S7 writes).

**Integration requirement:** `doctor`'s `EnvironmentTier` is a capability claim by another
name — it says which telemetry surface is reachable. **Extend S2's contract table to cover
`DetectEnvironment`**: a tier it reports as reachable must be a tier from which a capture
actually produces a `present` signal. Otherwise `doctor` becomes the fourth place in this
codebase that advertises what it cannot deliver.

**`git add`:** `profiler/analyze.go profiler/doctor.go profiler/{analyze,doctor}_test.go profiler/queries/ profiler/cmd/main.go profiler/cmd/main_test.go docs/profiler-spec.md README.md`

**Verified by:** `go test -race`; the extended S2 table; CLI tests for both subcommands.
I did not read `profiler/queries/*.sql` — the implementer must check they reference the
spool schema names S7 lands, not the sibling repo's.

---

### S9 — `skill-rewrite` gains `-o/--output` · `fix(skill-rewrite)/0.5.0-output-flag`

**Lands:** the `-o|--output` flag on `scripts/draft-rewrite.sh`, plus the SKILL.md usage
line. This is the **one surviving piece** of wip commit `b73326d` — see §5.

**Verified:** `main`'s `draft-rewrite.sh` parses only `-t|--target` and `-a|--audit`
(line 133) and hardcodes `output="$target_skill/REWRITE-DRAFT.md"` (line 181). The flag is
genuinely absent and genuinely wanted.

**`git add`:** `skills/skill-rewrite/scripts/draft-rewrite.sh skills/skill-rewrite/SKILL.md tests/test_rewrite.sh`

**Independent of S2–S8.** Can run in parallel with the profiler chain any time after S1.

**Verified by:** `tests/test_rewrite.sh` (1,047 lines on `main`) and `tests/test_skill.sh`.
Must use `main`'s `require_arg` helper (PR #9 named it), not reintroduce `b73326d`'s own.

**Collides with:** `skill-rewrite/SKILL.md` and `draft-rewrite.sh` — PR #10 touches both.
Sequence after #10 merges.

---

### S10 — Release 0.5.0 · `release/0.5.0`

**Lands:** a **new** `## 0.5.0` section in `CHANGELOG.md` and `RELEASE_NOTES.md` (not edits
to the 0.4.0 entry — §5); README capability tables; the version string in the five plugin
manifests; the version asserts in `tests/test_skill.sh`; annotated tag `v0.5.0`.

**Verified by:** all seven shell suites + the Go job, plus the repo's own release ritual —
dogfooding (AGENTS.md: "Dogfooding is a release gate") and the prose gate that found six
false claims in PR #5. **The 0.5.0 changelog must state what is *not* measured**: the
hook-spool payload shapes are `[DOCS]`, no Cursor/Codex/Devin adapter ships, and the
deferred-ledger's ~15 open 0.5.0 items (L5–L25) that this release does not close.

---

**Order:** G0 → S1 → S2 → S3 → S4 → S5 → S6 → S7 → S8 → S10, with **S9 in parallel**
after S1 and after PR #10.
**Ten PRs.** Each is independently shippable: `main` is releasable after every one of them,
because none leaves a half-built subcommand or a type nobody constructs.

---

## 4. The three adapters — the sharpest question

**Answer: no. None of Cursor, Codex or Devin satisfies the contract it would land into, and
the gap is not a porting gap. Codex and Devin are deleted. Cursor's value survives, but not
as an adapter.**

Five independent failures, each measured:

**(i) They do not compile.** §1.3. Six identifiers deleted by v0.4.1; `*int` arithmetic
broken by v0.4.2. Measured on both the wip and the working-tree versions.

**(ii) Ported mechanically, their semantics are still pre-0.4.1.** `tc.Input += val` on a
value field means a Cursor export carrying only cache tokens reports `"input": 0` — a
measurement nobody made. That is the proto3-zero fabrication class v0.4.1 and v0.4.2 spent
two releases closing, reintroduced once per harness. None of the three has resource+scope
series identity, cumulative-vs-delta temporality handling, run identity by
`startTimeUnixNano`, or mixed-temporality refusal. Each would be a fresh copy of every bug
`a90f632`'s changelog spends ten paragraphs describing.

**(iii) They violate AC9 — probe advertises what capture cannot deliver.** This is the
class the manager named, and it is worse than "the Devin adapter":

- **Devin.** `Probe()` sets `tokens`, `tool_calls` and `timing` to `session_data` whenever
  `hasATIFSignals()` finds a `steps`, `messages` or `transcript` key in the file. `Capture()`
  then returns, **unconditionally on every path that reaches it**,
  `UnknownTokenResult("Devin ATIF schema not yet verified; token, tool, and timing fields
  are unmapped")`. The adapter's own doc comment admits it: *"does not yet parse the schema;
  it reports the capability as `session_data`."* There is no input for which it can deliver
  what it advertises. It measures nothing.
- **Cursor.** Same pattern on the SQLite branch: `caps[MetricTokens] = SourceSQLite` for any
  stat-able `ExportFile`, then `UnknownTokenResult("Cursor SQLite token schema not yet
  verified")`. The comment says the quiet part — *"the surface is reported so the capability
  is auditable."* A `CapabilityReport` is not an audit log; it is embedded in every profile
  as a claim about what was read.
- **Codex.** Does not over-advertise in the same way, but reads token counts from
  `codex.sse_event` **log attributes** — a bespoke shape, not OTLP — and would have to be
  rewritten against `otlp.go` regardless.

The adapter-honesty lens, which ran 335 signal-outcomes over the single adapter that exists,
recorded this exactly: *"The placeholder Devin adapter is unpushed local work and should be
audited before it lands in 0.5.0."* This plan is that audit's conclusion.

**(iv) They fail PR #7's provenance contract, three more times.** None of the three passes
`sessionID` anywhere except into `Profile.SessionID`; none refuses an empty one. That is
finding **F1 — the P1 that 0.4.3 exists to close** — replicated per harness. Landing them
re-opens the exact class four releases were spent closing, in triplicate.

**(v) The README is currently correct and would become false.** `main` marks all three
harnesses "Planned", and `git ls-tree origin/main profiler/` returns no cursor, codex or
devin file. Landing placeholders makes a true document false.

### Disposition

- **`profiler/codex.go` — delete.** 228 lines. A Codex adapter that reads real OTLP through
  `otlp.go` is a rewrite, not a port; there is nothing in this file to carry forward except
  the event names, which belong in a design note. Cost of keeping it: a from-scratch rewrite
  *plus* the work of understanding a dead draft.
- **`profiler/devin.go` — delete.** 136 lines that measure nothing and claim otherwise.
- **`profiler/cursor.go` — delete as an adapter.** Its OTel branch is the bespoke envelope;
  its SQLite branch is a capability claim with no implementation.

**~640 lines deleted, zero capability lost.** The Cursor work that is real — the hook spool,
`spool_profile`, `analyze`, `doctor` — is a different and better source, and lands as
S7–S8 as **capture infrastructure**, not as an adapter. A `CursorAdapter` is constructed
later, on top of the spool, and only when it can pass S2's contract table. That gate is
mechanical, so it does not depend on anyone remembering this decision.

---

## 5. The colliding files, one by one

Seven files. For each: what the old work wanted, what `main` does now, and the call.

### 5.1 `profiler/claude_code.go` — **port one intent, drop the rest**

**Old intent (`fd2cfe5`, `1e4e845`, +337 lines):** add a JSONL-transcript `session_data`
reader beside the OTel reader, and share the bespoke envelope helpers with the three new
adapters.

**What `main` does instead:** v0.4.1 replaced the whole reader with `otlp.go` (1,086 lines,
absent from the wip branch) and deleted the envelope helpers. v0.4.2 re-keyed series
identity. PR #7 adds provenance projection and refuses an empty session id.

**Call: mostly obsolete.** The bespoke envelope is superseded. The session_data transcript
reader is a *different* source that the current design would express as a second source
behind the same `resolve(prov)` shape — that is new design work, not a port, and it is **not
in 0.5.0's landing set**. Record it as a 0.5.0-or-later backlog item.
**One intent survives and is genuinely deferred to this release:** reading `skill.name`
activation, which `main:205` names as 0.5.0 work in its own source. → **S4.**

### 5.2 `profiler/cmd/main.go` — **port the subcommand surface, not the file**

**Old intent:** grow the CLI from `probe|capture|version` to add `compare`, `ingest`,
`doctor`, `hooks`, `analyze`, `experiment`, and register four harnesses.

**What `main` does instead:** `probe|capture|version|help`, one harness (`claude_code`),
with `--export-file` and `--session` refusals owned by the adapter and restated by the CLI
so they cannot drift. PR #7 and PR #8 both modify it; the roadmap already records a
`resolve()`-signature collision between them here.

**Call: intent wanted, file not portable.** Six of the seven new subcommands are still the
0.5.0 product. The four-harness registry is **obsolete** — three of the four adapters are
deleted (§4). Each subcommand lands in its own slice (S5–S8), each adding one `case` to
`main`'s current dispatcher, never replacing it.

### 5.3 `profiler/profiler_test.go` — **do not port; re-derive**

**Old intent:** +766 lines covering the new adapters and commands.

**What `main` does instead:** +1,251 lines over the same file across three releases, plus
`otlp_test.go` (1,359 lines) and `cmd/main_test.go` (191), plus 55 committed OTLP fixtures.
PR #7 adds `provenance_test.go` (364).

**Call: obsolete as a diff, and dangerous as one.** A large fraction of the wip 766 lines
tests the three deleted adapters. Tests are written fresh per slice, against `main`'s
harness conventions. **Do not port this file, and do not credit its line count as coverage**
(S6's gap, §3).

### 5.4 `skills/skill-rewrite/SKILL.md` — **one line survives**

**Old intent (`b73326d`, `f13173e`):** rewrite the SKILL.md around a new preflight, an `-o`
output flag, audit-JSON parsing, and remove `/tmp/` path examples that tripped the
portability checks.

**What `main` does instead:** PR #9 (`a06af3f`, C8) rewrote 69 lines of it — made the
sibling dependency explicit and named (`audit_root`, *"it bundles neither"*), documented the
`skill-validator` requirement, and restructured the stages. `main` contains **no `/tmp`
path** (verified by grep).

**Call:**
- The `/tmp` fix (`f13173e`) is **obsolete** — already true on `main`, by a different route.
- The preflight and audit-parsing rewrites are **obsolete** — superseded by C8, which is
  better: it names the dependency instead of hiding it.
- The **`-o/--output` flag is NOT superseded** — `main` parses only `-t` and `-a`. → **S9.**

### 5.5 `skills/skill-rewrite/scripts/draft-rewrite.sh` — **the vendoring is a regression; discard it**

**Old intent (uncommitted, plus 5 untracked files):** make `skill-rewrite` self-contained
(audit finding T019) by **vendoring copies** of `skill-audit`'s `audit-report.sh`,
`check-frontmatter.sh`, `check-paths.sh`, `check-structure.sh` and
`references/evaluation-matrix.md` into `skill-rewrite/`, and repointing every call at the
local copies.

**What `main` does instead:** C7 and C8 wired `draft-rewrite.sh` into `skill-audit`'s
`scripts/verdict-guard.sh` with a load-check on both sides (`bash -n` before,
`declare -F verdict_guard_ready` after), a `CDPATH=`-hardened path resolution, `require_tool
mktemp`, and a deliberate, documented sibling dependency.

**Measured:** the vendored `audit-report.sh` is **117 lines**; `main`'s is **380**. The
vendored `audit-report.sh` and `check-frontmatter.sh` contain **zero** references to
`verdict_guard`.

**Call: discard the vendoring outright.** It forks four scripts away from four releases of
verdict-integrity work — 263 lines of it in one file — and would give `skill-rewrite` a
copy of `audit-report.sh` that reports a clean bill of health from a check that never ran.
That is the precise defect `a90f632` exists to fix, reintroduced under a different path.
The uncommitted diff also reintroduces `/tmp/draft-rewrite-audit-*.json`, which `f13173e`
removed and `main` no longer has.

**But the underlying question is real and does not go away:** "self-contained" is one of
this product's own audit dimensions, and `skill-rewrite` fails it. `main`'s answer is to
make the dependency explicit rather than remove it. The clean third option — extract the
audit scripts into a shared bundle both skills reference — is a real design and is **0.6.0
work, not a 0.5.0 port**. Escalated in §7.

### 5.6 `tests/test_skill.sh` — **obsolete**

**Old intent:** four lines adjusting asserts for the skill-rewrite changes.
**What `main` does instead:** +155 lines across four releases, and C7 rebuilt the assertion
mechanism entirely — worklist G7-01 found 16 assertions passing a literal and 11 inert
`[[ ]]` guards under bash 3.2, and the fix was a shared harness (`tests/lib/harness.sh`),
not a patch. PR #10 adds another 151 lines.
**Call: obsolete.** Whatever S9 needs is written against the current harness.

### 5.7 `tests/test_walk.sh` — **obsolete**

**Old intent:** one line.
**What `main` does instead:** +69 lines via C7; PR #8 adds 16 more.
**Call: obsolete.** Each slice adds its own claim-walk entries.

---

## 6. What the uncommitted work needs before it can be committed

### 6.1 Discard outright — superseded or out of scope (verified, one by one)

| Path / hunk | Why |
|---|---|
| `CHANGELOG.md` (the 0.4.0-entry edit) | `main`'s 0.4.0 bullet already reads *"Skill activation and attribution are honestly `unknown` (the adapter does not yet read skill-level attributes)"* — the 0.4.2 prose gate's verified wording. The uncommitted edit asserts 0.4.0 **did** read `skill.name`, which `main`'s own `claude_code.go:205` ("Reading it is 0.5.0") says is false. Landing it would put a false claim in the changelog. |
| `RELEASE_NOTES.md` (same edit) | Identical reason. |
| `profiler/go.mod` | The `auraprix` → `Okja-Engineering` module rename is **already on `main`**. Verified: `git show origin/main:profiler/go.mod`. |
| `.github/workflows/ci.yml` | Both hunks are skillgate-coupled: `go-version-file: go.work` (and `go.work` names `./skillgate`) and `cd skillgate && go test`. `main`'s ci.yml is a different file now — C7 rewrote it around a derived suite list, and PR #8 touches it again. Any 0.5.0 CI change is written fresh against post-0.4.3 ci.yml. |
| `go.work`, `go.work.sum` | `go.work` declares `use (./profiler ./skillgate)`. Never commit. `.gitignore`d in S1. |
| `NOTICE` | Its entire content attributes NVIDIA SkillSpector material used in `skillgate/difftest/`. Landing it in 0.5.0 would attribute code the repo does not contain. **0.6.0, with skillgate.** |
| `docs/skillgate-intent.md`, `docs/skillgate-spec.md`, `skillgate/` (139 files), `skills/skill-gate/` (1) | 0.6.0, user-confirmed. |
| `docs/research/` (40 files) | Roadmap: PR-0 `docs/research-and-repairs` is *"superseded; folds into 0.6.0."* No collision with `main`'s `docs/research.md` (different path). **I did not read these 40 files** — see §1.4; if any of it documents the hook spool, S7 may want one or two, named individually. |
| `skills/skill-rewrite/scripts/{audit-report,check-frontmatter,check-paths,check-structure}.sh`, `skills/skill-rewrite/references/evaluation-matrix.md` | The vendoring fork. §5.5. |
| `.out-of-scope.md` "Certify skills safe" bullet | Names `skillgate`. 0.6.0. |
| `skills/skill-audit/SKILL.md` (both hunks) | The dependency-guard downgrade: the *intent* is right — the SKILL.md preflight `exit 1` is now redundant with `verdict-guard.sh`'s `DEP001` / exit 3 — but the replacement prose is **wrong**: it says the scripts "will report 'unknown'", and they report a `fail`-level `DEP001` at exit 3. And PR #10 rewrites this file. **Do not port; raise as a finding against post-0.4.3 `main`.** `allowed-tools: Read, Bash` — see §7, U1. |

### 6.2 Port, with the named corrections

| Path | Slice | Correction required before commit |
|---|---|---|
| `.gitignore`, `AGENTS.md`, `.out-of-scope.md` (F04 bullet) | S1 | — |
| `profiler/types.go` | S3 | Revert the `CacheWrite` rename (user decision). Keep `*int`. Make `EstimatedContextTokens` a pointer. Extend `SignalStates()` by reflection. Set `AdapterVersion = "0.5.0"`. |
| `docs/profiler-spec.md` | S3/S4/S5 | Revert the `cache_write` line. Split: type surface → S3, activation ACs → S4, comparison contract → S5. |
| `profiler/claude_code.go` | S4 | Activation only. Everything else is superseded. |
| `profiler/cmd/main.go` | S5–S8 | One `case` per slice onto `main`'s dispatcher. Drop the cursor/codex/devin harness registry. |
| `profiler/compare.go`, `compare_test.go` | S5 | Add the `adapter_version` refusal. `EstimatedTokensResult` reference follows S3's pointer type. |
| `profiler/experiment.go` | S6 | Write real tests; do not credit the wip file's line count. |
| `profiler/hooks.go`, `hooks_install.go`, `spool_profile.go` + tests | S7 | Rename the `cursor-profiler/spool/v1` namespace. Prove no test writes to a real home. |
| `profiler/analyze.go`, `doctor.go` + tests, `profiler/queries/` | S8 | `doctor`'s tier claim goes under S2's contract table. Check the SQL against S7's schema names. |
| `skills/skill-rewrite/{SKILL.md,scripts/draft-rewrite.sh}` | S9 | `-o/--output` only. Use `main`'s `require_arg`. No vendoring. |
| `profiler/{cursor,codex,devin}.go`, `profiler/strict_test.go` | — | **Deleted.** §4. (`strict_test.go`, 64 lines, unread by me — if it tests the deleted adapters it goes with them; if it tests the contract, it belongs in S2. The S2 implementer reads it and decides.) |

### 6.3 The commit hygiene that makes 6.1 enforceable

Per D4: snapshot excludes the out-of-scope paths at copy time; every slice uses an explicit
`git add` path list; every slice runs the staged-path grep before committing; S1 lands the
`.gitignore` entries so the primary tree is protected too.

---

## 7. Risks, unknowns and escalations

### R1 — 0.4.3's own `AdapterVersion` bump is still not done. **[escalate]**

`AdapterVersion == "0.4.2"` on `origin/main` **and on all three open PR branches**, verified
2026-09-19. PR #7 changes what a profile contains for the same input. The 0.4.3 worklist
names the bump to `"0.4.3"` as its definition of done and flags "the bump is forgotten" as a
named risk (§689).
**Impact on 0.5.0:** if 0.4.3 ships without it, a 0.4.3 profile is permanently
indistinguishable from a 0.4.2 one. S5's `compare` cannot refuse a comparison it cannot
detect. **Nothing in 0.5.0 can repair it retroactively.**
**Resolved by:** the manager confirming the bump is in PR #7's or PR #10's final head before
0.4.3 merges. One `grep AdapterVersion profiler/types.go` on the merge candidate.

### R2 — The hook spool cannot be verified against a real Cursor. **[decision needed]**

S7 lands ~700 lines that parse Cursor hook payloads whose shapes are documentary, not
observed. Cursor is not installed on this machine. The repo already carries this exact
evidence-label gap as a known limit (deferred-ledger L24, L22).
**This is the same shape as the Devin adapter's defect, one level up:** code that claims to
read a source nobody here has run.
**Options:** (a) ship S7–S8 with the README and changelog stating plainly that every payload
shape is `[DOCS]` and no capture has been observed; (b) hold S7–S8 until someone can run
Cursor and capture one real spool; (c) ship the spool writer but not `doctor`'s tier claim,
which is the part that asserts reachability.
**My recommendation: (a) plus S2's contract table**, because the spool is lossless-by-design
(it writes the untouched payload), so a wrong *parse* is recoverable from the same files,
where a wrong *capability claim* is not. **Manager/user decides.**

### R3 — The collision set is growing faster than it is being cleared. **[monitor]**

It went 4 → 7 on one merge (§1.1). PR #10 will touch `skill-rewrite/SKILL.md`,
`draft-rewrite.sh` and `test_skill.sh`; PR #8 `test_walk.sh`; PR #7 `claude_code.go`,
`cmd/main.go`, `profiler_test.go`. Every day the unlanded work sits, the port gets more
expensive, and the 2,230 untracked lines have no backup but this filesystem.
**Resolved by:** taking the D3 snapshot **today**, before G0 — it is read-only and costs
nothing — and cutting S1 the hour PR #7 merges.

### R4 — Verification that does not exist yet

- **S5, S6, S8: CLI-level tests.** `profiler/cmd` coverage is 12.6% (ledger L21) and
  `cmd/main_test.go` is 191 lines for two subcommands. 0.5.0 adds six. The tests must be
  written in-slice; no existing suite proves a subcommand's behaviour.
- **S6 specifically: there is no `experiment_test.go`.** Do not assume the wip branch's 766
  test lines cover it. Measure first.
- **The bash 3.2 half of "seven suites on both bashes" is not in CI.** `main`'s ci.yml runs
  `ubuntu-latest` only — bash 5. The bash 3.2 requirement is a local macOS step in the
  release checklist, and C7's whole finding was that bash 3.2 exempts `[[ ]]` from `errexit`
  while bash 5 does not. **A slice whose proof is "green on CI" has not been proven on bash
  3.2.** Resolved by adding a `macos-latest` job running `/bin/bash tests/*.sh` — I
  recommend it as an addition to S1, or as an explicitly accepted risk recorded in S10.

### R5 — The `omitempty`-on-a-struct trap

`Profile.EstimatedContextTokens` as a value type with `omitempty` emits
`{"state":""}` on **every** profile, out of the `MetricState` vocabulary, and forces a schema
v2 nobody wanted. D5 fixes it with a pointer; S3 carries the test. Flagged here because it
is a one-character defect with a release-scoped consequence, and it is the single most
likely thing in this plan to be re-introduced by a later edit.

### U1 — `allowed-tools: Read, Bash` — **undetermined from the repo**

The uncommitted diff adds this frontmatter key to both SKILL.md files. I could find no
handling of it in `main`'s `check-frontmatter.sh`, and this repo does not tell me whether
`skill-validator` 1.6.1 accepts, ignores or warns on it, nor what it does to `skill-audit`'s
own runtime (which shells out). `tests/test_skill.sh` asserts both skills validate with
**zero errors and zero warnings** (added by 0.4.2), so a warning here turns the suite red.
**Resolved by:** running `skill-validator` against a skill carrying the key. One command.
Until then, **do not land it.**

### U2 — Things I did not read, stated so they are not mistaken for cleared

Nine tracked modified files read as diffstat only; PR #7's `provenance.go` (268 lines) and
its `otlp.go` changes; `docs/research/`'s 40 files; `profiler/queries/`'s 8 SQL files; seven
Go test files including `strict_test.go`; `skillgate/`'s 139 files. Each slice that touches
one of these reads it in full as its first step, and that instruction is written into the
slice. **This plan's enumeration of what lands is complete over the denominators in §1.4 and
nowhere wider.**

---

## 8. Definition of done for this plan

- `origin/main` carries S1–S10, squash-merged, linear, tagged `v0.5.0`.
- `git ls-tree -r origin/main --name-only | grep -E 'skillgate|skill-gate|go\.work|^NOTICE$'`
  returns nothing.
- `grep -rn 'cache_write\|CacheWrite'` over the repo returns nothing.
- `profiler/{cursor,codex,devin}.go` do not exist, and the README still says "Planned".
- `AdapterVersion == "0.5.0"`, `ProfileSchema == "skill-architect/profile/v1"`.
- `wip/0.5.0-profiler` is tagged `archive/0.5.0-wip` and referenced nowhere else.
- The 0.5.0 changelog names what was **not** measured.
