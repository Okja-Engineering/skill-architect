# S1 record — G0 gate, D3 snapshot, and slice S1

**Implementer** senior-implementer · **Date** 2026-09-20 · **Base** `origin/main` = `5b60fca`
**Branch** `fix/0.5.0-boundary-hygiene` · **Commit** `a864646` · **PR** [#13](https://github.com/Okja-Engineering/skill-architect/pull/13) (open, not draft, base `main`)
**Status** delivered, not merged. The user merges.

---

## 1. G0 — the gate

### 1.1 Preconditions: met

| Precondition | Evidence |
|---|---|
| 0.4.3 fully merged | `gh pr list --state open` → `[]`. PRs #7–#12 all merged. |
| `origin/main` = `5b60fca` | `git rev-parse origin/main` |
| `AdapterVersion` = `"0.4.3"` | `git show origin/main:profiler/types.go` → `const AdapterVersion = "0.4.3"` |
| `ProfileSchema` unchanged | `"skill-architect/profile/v1"` — D5 holds |

**R1 is closed.** The plan's sharpest escalation — "`AdapterVersion` is `0.4.2` on all four
branches; if 0.4.3 ships without its bump, a 0.4.3 profile is permanently
indistinguishable from a 0.4.2 one" — did not happen. The bump landed. S5's `compare` will
have a version to refuse on.

**The merge order the plan predicted was wrong, harmlessly.** Plan: "#9 → #8 → #7; #10 after
#9". Actual: #9 `a06af3f` → #10 `2c155e7` → #8 `505a34b` → **#11** `8342b21` → #7 `be5c0aa`
→ **#12** `5b60fca`. **#11 and #12 were not in the plan at all** (#11 a C5 prose correction,
#12 the release). Four merges after `a06af3f`, not three.

### 1.2 §1.1 re-derived — the committed side

| Figure | Plan (`a06af3f`) | Now (`5b60fca`) | Moved? |
|---|---|---|---|
| wip diffstat `30f374c..0a83615` | 2,534 ins / 118 del, 12 files | 2,534 / 118, 12 files | no |
| **Colliding files** | 7 | **7** | **no** |
| Disjoint (wip-only) | 5 | 5 | no |
| Denominator: files `main` changed since `30f374c` | 94 | **106** | **+12** |

**R3 did not materialise — the collision set did not grow.** The plan expected it to, since
#7/#8/#10 each touched files on the list. But every file they touched was *already*
colliding, so the intersection held at 7 while the denominator moved 94 → 106. With `wip`
frozen and 0.4.3 closed, **the set is now stable and cannot grow before the slices land.**
That removes the urgency R3 attached to cutting slices fast.

Colliding set unchanged: `profiler/{claude_code.go,cmd/main.go,profiler_test.go}`,
`skills/skill-rewrite/{SKILL.md,scripts/draft-rewrite.sh}`, `tests/{test_skill.sh,test_walk.sh}`.
Disjoint unchanged: `profiler/{codex,compare,cursor,devin,experiment}.go`.

### 1.3 §1.4 re-derived — the uncommitted side

| Figure | Plan | Now | Moved? |
|---|---|---|---|
| Tracked modified | 21 files, 687 ins / 175 del | 21, 687 / 175 | no |
| Overlapping with `main`'s changes | 13 | 13 | no |
| Untracked paths | 26 | 26 | no |
| Untracked files (recursive) | — | **210** | new |
| Untracked `profiler/*.go` | 2,230 lines / 12 files | 2,230 / 12 | no |
| `skillgate/` | 139 | 139 | no |
| `docs/research/` | 40 | 40 | no |
| `profiler/queries/` | 8 | 8 | no |
| `skills/skill-gate/` | 1 | 1 | no |

**Nothing moved.** Nobody edited the primary tree between plan and gate. It is now frozen.

### 1.4 The flagged risk: the rewrite suite **survived**

The plan warned that PRs #7 and #8 showed `tests/test_rewrite.sh` and
`tests/fixtures/rewrite/` as deletions, and that a real deletion would cost 0.5.0 a
1,047-line suite. **It was stale-base noise and it resolved on rebase, exactly as judged.**

| Check at `5b60fca` | Result |
|---|---|
| `tests/test_rewrite.sh` present | **Yes** |
| `tests/fixtures/rewrite/` present | **Yes** — 2 files, the same 2 as at `a06af3f` |
| Suite executes and reports | **85 assertions, 0 failed**, both shells |
| All seven suites present | Yes |

**One number in the plan is now wrong:** S9 calls `test_rewrite.sh` "1,047 lines on `main`".
It is **939** at `5b60fca`. The 108-line reduction is deliberate and attributable —
`2c155e7` (C1/C3/C4) and `8342b21` (C5, "claims that derive instead of restating") took 166
lines out and put 58 back. **No assertion was lost**: 85 run. S9 should cite 939.

---

## 2. D3 — the snapshot

| | |
|---|---|
| Snapshot | `<scratch>/snapshot-0.5.0/` |
| Manifest (full) | `<scratch>/snapshot-0.5.0.manifest` — sha256, **15,452 lines** |
| Manifest (repo content, durable) | `.scuba/teams/release-0.5.0/snapshot-0.5.0.repo-files.manifest` — **78 lines** |
| Total files | **15,452** |
| **Of which repo content** | **78** |
| Of which `.venv-skillspector/` | **15,374 — 99.5%** |
| Size | 195 MB |
| Read-only | `chmod -R a-w`; a write attempt was made and **refused** |
| Excluded paths absent | **All 11 verified absent**, and 0 matching lines in the manifest |
| Primary tree after | **untouched** — still `0a83615`, 21 modified / 687 ins / 175 del, 26 untracked |

Excluded and confirmed absent: `.git`, `.scuba`, `tmp`, `skillgate`, `skills/skill-gate`,
`go.work`, `go.work.sum`, `NOTICE`, `docs/research`, `docs/skillgate-intent.md`,
`docs/skillgate-spec.md`.

No `checkout`, `reset`, `clean`, `stash` or `rebase` was run in the primary tree. `git
stash create` was not used, per D3.

**The provenance check works, and S1 used it.** `AGENTS.md` landed byte-identical to the
snapshot and its sha256 matched the manifest line exactly:
`a77f5a98d1199e202391368293e75092ed2b9853afcb04a89db8e7e97ba38e8d`. That is D3 earning its
cost on the first slice.

---

## 3. S1 — what landed

**Two files, not three.** `.gitignore` and `AGENTS.md`. `git show --stat` names exactly
those two. The third is a plan finding, §4.1 below.

### `.gitignore` (+17/-1)

Adds `go.work`, `go.work.sum`, `skillgate/`, `skills/skill-gate/`, `.scuba/`; widens
`.venv/` → `.venv*/`. Each group carries a short comment saying why, because D4 frames
these as load-bearing enforcement and an uncommented list is the kind a later slice tidies
away.

### `AGENTS.md` (+6/-0)

The four Operating principles, appended as a new section. Byte-identical to the snapshot.

### Verification

S1 changes no behaviour, so green is the proof of no regression.

| | bash 3.2 (3.2.57) | bash 5 (5.3.15) |
|---|---|---|
| test_harness | 75 | 75 |
| test_skill | 53 | 53 |
| test_install | 48 | 48 |
| test_walk | 20 | 20 |
| test_rewrite | 85 | 85 |
| test_f01 | 1868 | 1868 |
| test_f02 | 350 | 350 |
| **Total** | **2499 passed, 0 failed** | **2499 passed, 0 failed** |

The 2499 orientation figure is **current, not stale** — it reproduces exactly on both shells.

Go, in `profiler/`: `gofmt -l .` empty; `go build ./...` clean; `go vet ./...` clean;
`go test -race -count=1 ./...` green — **93 top-level tests, 545 PASS lines with subtests,
0 failures.**

**The `.gitignore` change was driven test-first.** A 20-check acceptance script was written
first and run **RED** against the unmodified `.gitignore`: 6 failures, precisely the six
missing entries, with `.venv/` and `tmp/` already passing. After the edit: **20/20 GREEN**.
The checks run in both directions — 8 paths that must be ignored, and 12 that must stay
stageable (`profiler/`, `profiler/queries/`, `skills/skill-audit/`, `skills/skill-rewrite/`,
`tests/test_rewrite.sh`, `docs/profiler-spec.md`, `.github/workflows/ci.yml`, …), because an
over-broad pattern is the more expensive mistake and would silently block a later slice.

Script: `<scratch>/s1-check-gitignore.sh`. **It is not committed** — see §4.3.

### Hygiene

- `git add .gitignore AGENTS.md` — explicit paths. No `git add -A`, no `commit -a`.
- D4 barrier 3 assertion run before the commit: **PASS**, no out-of-scope path staged.
- Author and committer `imagineux <imagineux@gmail.com>`.
- AI-attribution grep over commit message, author, committer, PR title and body: **0**.
- No version surface touched. `AdapterVersion` still `"0.4.3"`.

---

## 4. Findings — things that should change before slice two

### 4.1 The `.out-of-scope.md` bullet is a false claim at S1. Moved to S5. **[ratify]**

S1 was told to land the "Run live paired comparisons end-to-end" bullet, and §6.2 marks the
row "—", no correction required. **That is the error.** The bullet reads:

> The profiler captures runtime signals and `profiler compare` (F04) reports paired
> differences; scheduling and executing the paired runs remains the caller's job.

`profiler compare` does not exist at `5b60fca`. The CLI dispatches `probe|capture|version|
help`; `git ls-tree -r 5b60fca` has no `compare.go`; `git grep compare` over `profiler/`
returns only comments. Landing that sentence would contradict two lines in the *same
commit*: `README.md:612` ("the comparison engine is not yet built") and `README.md:303`
("future paired comparisons (F04)").

This is the defect §6.1 rejects the `CHANGELOG`/`RELEASE_NOTES` edits for — asserting a
capability the source says does not exist — so the **plan's own standard decides it**. The
bullet is not wrong, it is *early*.

**Done:** bullet deferred to **S5**, which lands `compare` and already edits `README.md` and
`docs/profiler-spec.md`. `.out-of-scope.md` added to S5's `git add` list in the plan. S5
must also fix `README.md:612` and `:303`, which become false in the same commit.
**Manager to ratify.**

### 4.2 D4 barrier 3's assertion reports failure on a clean index. **[fix before S2]**

The snippet every slice is told to run before committing:

```
git diff --cached --name-only \
  | grep -E '^(skillgate/|...)' \
  && { echo "OUT-OF-SCOPE PATH STAGED"; exit 1; }
```

Measured, with a clean index:

- as a **non-final** statement in a script → script exits **0**. Correct.
- as the **final** statement of a script → script exits **1**. **Inverted.**

`grep` exits 1 when it matches nothing, and nothing resets it. A pre-commit hook, a CI step,
or any wrapper that reads `$?` sees "out-of-scope path staged" **when the staging area is
clean** — and the natural fix under pressure is to delete the check. It also cannot be
`source`d, since `exit 1` would kill the caller's shell.

**Recommendation:** ship it as a small committed script with an explicit verdict, e.g.

```sh
if git diff --cached --name-only | grep -qE '^(skillgate/|...)'; then
  echo "OUT-OF-SCOPE PATH STAGED"; git diff --cached --name-only | grep -E '...'; exit 1
fi
echo "staged paths clean"; exit 0
```

A barrier whose clean state looks like a failure is a barrier that gets removed at slice
five.

### 4.3 S1's content has **no committed test**, and green proves nothing about it. **[gap]**

Measured: **none** of the seven suites reads `.gitignore`, `AGENTS.md` or
`.out-of-scope.md` — `grep -c 'out-of-scope|AGENTS.md|.gitignore'` is 0 in all seven. So
"green is the proof of no regression" is true and also the whole story: green says nothing
positive about what S1 landed.

The consequence is specific: **a later slice can delete a `.gitignore` entry and all 2499
assertions still pass.** D4 calls these entries the mechanism that turns discipline into
enforcement; nothing currently enforces the enforcement. The repo's own `test_harness.sh`
exists to refuse checks that cannot fail — this is the same class, one level up.

**Recommendation:** fold the 20-check script into `tests/test_harness.sh` or
`tests/test_walk.sh` as a committed suite. I did **not** do it in S1: S1's `git add` list is
three named paths and a new `tests/` file is outside it. This is the kind of scope decision
the manager should make rather than the implementer.

### 4.4 D3's exclusion list misses `.venv*/` — 99.5% of the snapshot is noise. **[fix before S2]**

Run literally, D3's rsync copies `.venv-skillspector/` — **15,374 files, 195 MB** — into the
snapshot. Of 15,452 files, **78 are actual repo content**. D3's own instruction to "confirm
the snapshot's file count" therefore tells a reader nothing, and the manifest is 3.1 MB.

I executed D3 literally rather than improvising the exclusion, and mitigated by writing the
78-line repo-content manifest into the control plane. **Recommendation: add
`--exclude '.venv*/'` to D3 before S2.** S1's own `.gitignore` already declares that path
out of scope, so this is consistent with the release, not a new decision. The snapshot as
taken is correct and usable — no re-take needed.

### 4.5 The snapshot is in `/tmp` and is a sole copy. **[decide]**

`<scratch>` is `/private/tmp/claude-501/<session-uuid>/scratchpad` — session-scoped and
subject to OS temp cleanup. It now holds one of only two copies of 2,230 untracked lines;
the other is the primary working tree, which D3 exists because we do not trust. R3 named
"the 2,230 untracked lines have no backup but this filesystem" as a live risk, and the
snapshot as specified does not retire it — it duplicates within the same filesystem, into
its most volatile directory.

**Recommendation:** a second copy outside `/tmp`, or commit the snapshot to a
`snapshot/0.5.0-worktree` branch (read-only, never merged). Cheap now; unrecoverable later.

Minor, same area: `chmod -R a-w` strips write permission from **directories**, so removing
the snapshot later needs `chmod -R u+w` first. Worth one line in the slice-ten cleanup.

### 4.6 Worker worktrees must NOT go in `skill-architect-wt/`. **[blocks S2–S8]**

The enforcement hook (`~/.claude/hooks/scuba-guard.sh`) recognises exactly two safe write
locations: a path with a `.scuba` component, and `/tmp` / `/private/tmp` / `/var/folders`.
Its worktree convention is `<project>/.claude/worktrees/agent-*`.

`/Users/.../skill-architect-wt/<name>/` — **the convention the whole 0.4.x release used** —
matches neither. A write there falls to the "no worktree anchor" branch, which allows `.md`
and **denies tracked non-`.md` code**. S1 would have been half-blocked there: `AGENTS.md`
allowed, `.gitignore` denied. **S2–S8 are all Go slices and would be denied outright.**

**Done:** S1 used the session scratchpad, which is whitelisted. **Every later slice must do
the same** (or `.claude/worktrees/agent-*`). This is worth stating in the plan, because the
0.4.x convention is the wrong answer now and it is the first thing an implementer will copy.

### 4.7 Smaller notes

- `zsh` is the tool shell, and it does not word-split unquoted `$VAR`. A `for s in $SUITES`
  loop silently becomes one iteration with a garbage filename. Drive multi-suite loops from
  a `bash` script, not an inline zsh command.
- Reading git objects needs `git show "${rev}":"path"`, not `git show $rev:path` — zsh eats
  `:t` and friends as history modifiers and corrupts the revision.
- Both shells produced **identical** counts on all seven suites. For S1 the bash-3.2 run
  found nothing bash 5 missed — but S1 touches no shell code. R4's point stands unchanged
  for S9, which does.
- `skillscore` 2.0.2 and `skill-validator` v1.6.1 are both present and intact. The
  previously destroyed `skillscore` install is healthy; nothing was installed or
  reconfigured during this slice.

---

## 5. Not done, and why

| Not done | Why |
|---|---|
| `.out-of-scope.md` bullet | False claim at `5b60fca`. Moved to S5. §4.1. |
| A committed test for the `.gitignore` entries | Outside S1's named `git add` list. Recommended, §4.3. |
| Adding a `macos-latest` bash-3.2 CI job | R4 suggests it "as an addition to S1". It is not in S1's Lands/`git add` list, and `.github/workflows/ci.yml` is not one of S1's three paths. Left for the manager: S10 or its own slice. |
| Tagging `archive/0.5.0-wip` | D1's, not S1's. Still outstanding. |
| Merging | The user merges. |
