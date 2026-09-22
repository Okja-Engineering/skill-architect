# Gate lens: RELEASE MECHANICS — PR #22 @ `9c8ba53`

> **Persisted by the chief of staff.** The hunter had no Write tool, said so, verified by
> `ls`, and did not fall back to a heredoc. This is that report.

**VERDICT: NOT CLEAN — 3 medium (M1, M2, M3), 2 low (M4, M5), 3 informational.**

## Coverage

| Surface | Enumerated from | Walked |
|---|---|---|
| DoD conditions | `landing-plan.md` §8, lines 1059–1066 | 7/7, twice |
| Diff files | `git diff --stat 10326b3 9c8ba53` | 10/10 |
| Build/test commands | `ci.yml` steps, verbatim | 6/6 |
| Shell suites | `ls tests/test_*.sh` = 7; CI names the same 7; `test_harness` asserts set equality | 7 × 2 shells = **14/14** |
| Version surfaces | `grep -rn version --include=*.json` + `types.go` + `cmd/main.go:48` + both SKILL.md | 8/8 |
| Manifests | the 5 named | 5/5 JSON-validated + cross-checked |
| Install routes | every route in README §Install (700–767) | 7 enumerated, **2 run**, 5 unrunnable/excluded |
| skillscore revisions | v0.4.1, v0.4.2, v0.4.3, `10326b3`, `9c8ba53` | 5 × 2 skills = 10/10 |
| Tags | local + `ls-remote` + diff scan | 3/3 |
| Attribution | 2 diff commits + all 34 on branch + PR body | 3/3 |
| Remote refs (durability) | every `refs/remotes/origin/*` + every tag, programmatically | 17 heads + 4 tags |

**Every command, with verdict:** `go build` **0** · `go vet` **0** · `gofmt -l` **empty**
· `go test` **ok ×3** · `go test -race -count=1` **ok ×3** · `go test -cover`
**97.9/94.5/93.5** · 7 suites × bash 3.2.57 → **2649/0** · 7 suites × bash 5.3.15 →
**2649/0, identical per-suite** · `skill-validator check` ×2 → **passed, zero errors zero
warnings** · `skillscore` ×10 → table below · 5 manifests `json.tool` → **all valid** ·
Claude Code local-checkout install → **0.5.0** · Claude Code GitHub install → **0.4.3**
(M6) · `git tag` → **exactly 4** · `gh pr checks 22` → **pass** · end-to-end `capture` →
`"schema":"skill-architect/profile/v1"`, `"adapter_version":"0.5.0"`.

## Definition of done — all 7 re-verified at head

| # | Condition | Verdict |
|---|---|---|
| 1 | S1–S10 squash-merged, linear, tagged | **Partial, by design.** Linear (`log --merges` empty), carries S1–S9. S10 is this unmerged PR; tag deliberately the user's. |
| 2 | no `skillgate\|skill-gate\|go\.work\|^NOTICE$` | **PASS**, 0 matches |
| 3 | no `cache_write\|CacheWrite` | **PASS**, 0 matches |
| 4 | `{cursor,codex,devin}.go` absent; README "Planned" | **PASS**, 0 files, 3 `\| Planned \|` rows (`README.md:35–37`) |
| 5 | `AdapterVersion`/`ProfileSchema` | **PASS** (`types.go:377`, `:367`) — confirmed end-to-end in a real captured profile, not just source |
| 6 | `wip/0.5.0-profiler` tagged `archive/0.5.0-wip` | **Deviation.** Zero `archive/*` tags. "Referenced nowhere else" **passes** (0 hits tree-wide). → **M3** |
| 7 | changelog names what was not measured | **PASS** — `CHANGELOG.md:15`, `:247`, `:304`, `:71` |

**Independent verification of S10's numbers — every figure reproduced exactly, none
carried:** 2649 (174+53+48+20+136+1868+350) ✓ · 0.4.3 baseline 2499 from a v0.4.3
worktree ✓ · 275 (233/34/8) ✓ · 669 (636 depth-1, 33 depth-2), `--- SKIP:
TestHomeBarrierChild` ✓ · 0.4.3's 93 and 452 ✓ · coverage 97.9/94.5/93.5 ✓ · 64 fixtures
(48 json + 16 ndjson) vs 59 ✓ · 8 queries ✓ · skill-audit 92.5 (A-) / skill-rewrite 89.5
(B+) ✓.

> **Note for the record:** `-cover` only yields 94.5% for `./cmd` under the **documented**
> command. `go test -race -cover ./...` gives **3.6%** — `-race` breaks the `GOCOVERDIR`
> propagation the subprocess coverage depends on (`cmd/main_test.go:68`). The notes say
> `go test -cover`, so they are precise; flagged only so nobody "confirms" it with
> `-race` and reports a false regression.

## The skillscore claim — independently confirmed; 0.5.0 changes nothing

| rev | skill-audit | clarity | skill-rewrite |
|---|---|---|---|
| v0.4.1 | 94 (A) | 9 | 89.5 (B+) |
| v0.4.2 | 94 (A) | **9** | 89.5 (B+) |
| v0.4.3 | 92.5 (A-) | **8** | 89.5 (B+) |
| `10326b3` | 92.5 (A-) | 8 | 89.5 (B+) |
| `9c8ba53` | 92.5 (A-) | 8 | 89.5 (B+) |

Drop is **entirely inside 0.4.3**, `clarity` 9→8, finding `{"type":"warning","message":"1
synonym pair(s) used interchangeably","points":1}`. **0.5.0 changed neither score.**
`CHANGELOG.md:222–225` and `RELEASE_NOTES.md:26` are correct including "v0.4.1 scores 94".
No unrecorded in-release regression. Correctly filed as history.

**`skillscore` global install intact: `--version` → `2.0.2`**, via
`~/.nvm/versions/node/v25.9.0/bin/skillscore -> ../lib/node_modules/skillscore/dist/index.js`
(symlink dated Sep 9, untouched). `npm ls -g` → `skillscore@2.0.2`. No PATH mirror built,
no stub written. `skill-validator` → `v1.6.1`. The user's real `~/.claude/plugins` has
**zero** files modified today.

## Manifests and install

All five valid JSON. `marketplace.json`'s embedded entry agrees with
`.claude-plugin/plugin.json` on `name`, `version`, `description`, `source: "./"`.
Installed for real: cache at `.../skill-architect/skill-architect/0.5.0/`, both SKILL.md
present, `claude plugin list` → **`Version: 0.5.0`**. README's trailing-slash claim holds
(`add .` → `✘ Invalid marketplace source format`; `add ./` → success).

**Routes run:** Claude Code local-checkout (**0.5.0** ✓), Claude Code GitHub native
(**0.4.3**, M6).
**Routes NOT run — do not read as verified:** Codex native and local (`codex` not
installed); Cursor copy/symlink (`cursor` not installed); Devin native and local
(**excluded by mandate** — and `devin` **is** installed at `/opt/homebrew/bin/devin`, so
the exclusion was load-bearing; not invoked).

## Tag and attribution — both clean

`git tag` → exactly `v0.2.0 v0.4.1 v0.4.2 v0.4.3`. `ls-remote --tags` → the same 4. No
`v0.5.0` anywhere. The diff contains no tag instruction. **PASS.**

All 34 commits: `grep -icE 'co-authored|generated with|noreply@anthropic'` over `%H%n%B` →
**0**. The 2 diff commits: author and committer `imagineux <imagineux@gmail.com>`,
`%(trailers)` **empty on both**, full bodies read. PR body: no trailer, no "Generated
with", no 🤖. The only `claude` strings are `claude plugin …` prose and `.claude-plugin/`
paths. **PASS.**

The release's own test-first claim driven RED: reverting all five manifests to `0.4.3`
gives **exactly 5 `FAIL:` lines**, `48 passed, 5 failed`, identically on both shells.
Non-vacuous.

---

## M1 · REAL · medium — the release documents are a version surface with no guard, and this PR is 98% release documents

**Proven, not reasoned.** Deleting the *entire* `## 0.5.0 — 2026-09-21` section from
`CHANGELOG.md` and the *entire* `## v0.5.0` section from `RELEASE_NOTES.md` in a copy of
the head tree, then running all seven suites:

```
test_skill     53 passed, 0 failed      test_f01   1868 passed, 0 failed
test_harness  174 passed, 0 failed      test_f02    350 passed, 0 failed
test_install   48 passed, 0 failed      test_walk    20 passed, 0 failed
test_rewrite  136 passed, 0 failed      →  2649 passed, 0 failed
```

**Byte-identical to the unmodified tree.** `grep -rn 'CHANGELOG\|RELEASE_NOTES'
tests/*.sh tests/lib/*.sh` returns **nothing**.

`tests/test_skill.sh:93-96` guards the fifth manifest, and its own comment states the
reason: *"a release that bumps the four plugin.json files and forgets this one advertises
the previous release to anyone installing by name."* That lesson was applied to
`marketplace.json` and **not** to the two documents that **are** this release. A release
that bumps five manifests and ships no changelog section is green on CI and green on all
2649 assertions.

**Invariant:** the version the manifests advertise must have a section in both release
documents, asserted by a suite CI runs. **Shared root with M2:** the repo enforces
version/boundary invariants by author memory wherever the assert was not written, and it
has done so on the three surfaces it learned the lesson from and no others.

## M2 · REAL · medium — `.gitignore`'s boundary block claims tool enforcement it does not have, for 2 of 4 forbidden-path classes

`.gitignore:10-16` states the mechanism explicitly: *"They are ignored rather than merely
avoided so that the boundary is enforced by the tool instead of remembered by the author:
a `git add -A` cannot stage what git will not see."*

```
go.work                      IGNORED
go.work.sum                  IGNORED
skillgate/x.go               IGNORED
skills/skill-gate/SKILL.md   IGNORED
NOTICE                       NOT IGNORED     ← in the DoD grep
docs/skillgate-spec.md       NOT IGNORED     ← matches the DoD grep 'skillgate'
docs/skillgate-intent.md     NOT IGNORED     ← matches the DoD grep 'skillgate'
```

All three exist untracked in the working tree **right now** and all three match the DoD's
own pattern. `git add -A --dry-run` in the primary tree stages `NOTICE`,
`docs/skillgate-intent.md`, `docs/skillgate-spec.md`. The `skillgate/` *directory* is
ignored; the skillgate *docs* are not, because the pattern is `skillgate/` and they are
`docs/skillgate-*.md`.

**Compounding, and operationally the live state:** the primary tree's *working*
`.gitignore` is the three-line pre-boundary version (`tmp/`, `.venv*/`, `.scuba/`) — that
tree sits on `0a83615`. So in the tree the maintainer merges from, the block is absent
entirely and `git add -A --dry-run` stages **all 150+** forbidden files, `skillgate/`
included. Resolves on checkout of `main`; reported because it is the state today and the
merge happens from there.

**Invariant:** every path class the DoD grep forbids must be *unstageable*, not merely
un-staged.

## M3 · REAL · medium — DoD #6's durability rests on two mutable branches, and this repo has already lost a pushed 0.5.0 slice branch

**Durability intent for the 0.5.0 WIP: satisfied today.** Extracting
`origin/archive/0.5.0-wip-tree` (`a3673b8`) and comparing every one of the 47 `git status`
entries in the maintainer's tree: **zero `DIFFERS`.** Every modified-tracked and untracked
profiler file is byte-identical to the pushed archive. The three deleted adapter drafts are
present on **both** `origin/archive/0.5.0-wip-tree` and `origin/wip/0.5.0-profiler`, so
the changelog's "copied all three into `profiler/` at head and built" measurement is
reproducible by anyone with a clone.

**But the mechanism substituted for the tag is demonstrably not durable, and the
counter-example is in this repo:**

**`origin/feat(profiler)/0.5.0-analyze-doctor` was pushed and is now gone from the
remote.** `ls-remote --heads origin` lists its six sibling slice branches and not it. It
was pushed: `.git/config` carries
`branch."feat(profiler)/0.5.0-analyze-doctor".remote=origin` and `.merge=refs/heads/…`,
and the remote-tracking ref existed locally until this session's `fetch --prune` reported
`[deleted] (none) -> origin/feat(profiler)/0.5.0-analyze-doctor`. Its three commits
(`5365a46`, `5eae640`, `d540aff`) are reachable from **no** remote ref. **No data was
lost** — every unique file is byte-identical in `10326b3`
(`internal/homesafe/homesafe{,_test}.go`, `doctor.go`, `analyze.go`,
`queries/payload_{bytes,keys}.sql` all `SAME`) — but it is proof that a branch in this
repo gets deleted while a tag would not have.

Two further exposures, same root:
- `origin/wip/0.5.0-profiler` == `0a83615`, the tip of the branch the maintainer's
  **working tree is currently on**. A routine commit + push moves the "frozen" point. An
  annotated tag cannot be moved by a routine push.
- `git tag --points-at a3673b8` → **empty**. The archive commit is held by a branch ref only.

`s10-record.md:385-386` records this as "satisfied as a branch, not a tag"; `:479` declines
conversion as "the maintainer's". The letter is a deviation and the intent holds *today* —
but the substitution is not equivalent, and the repo supplies its own counter-example.

**Invariant:** the frozen 0.5.0 WIP must be addressable by a ref that a routine push
cannot move and a routine branch-cleanup cannot delete.

## M4 · REAL · low — 186 untracked files exist on no remote ref or tag

Walking **every** `refs/remotes/origin/*` and **every** tag programmatically: not one
contains `skillgate/` (139 files), `docs/research/` (41 files),
`docs/skillgate-{spec,intent}.md`, `NOTICE`, `go.work`, `go.work.sum`. **They exist on
exactly one disk.**

This is the 0.6.0 work the release deliberately excludes, and DoD #6's letter covers only
the 0.5.0 profiler WIP — so it is **outside the condition**. Reported because the question
asked was "is all the unlanded work actually recoverable from the remote?", and for this
set the answer is no. Same root as M3.

## M5 · REAL · low — the PR body attaches the grade letter to the wrong number

PR #22 body: *"`skill-audit`'s quality score fell 94 (A-) → 92.5 in 0.4.3"*.

Measured: **94 → `letterGrade: "A"`**, **92.5 → `letterGrade: "A-"`**. The `(A-)` is on the
from-number. `CHANGELOG.md:222` and `RELEASE_NOTES.md:26` both state it correctly as "94
(A) to 92.5 (A-)" — so this is the PR body alone, and it inverts the very fact the bullet
exists to record. **Invariant:** a grade travels with its own number.

## M6 · Informational, not a defect

`README.md:713` — "`claude plugin list` then shows `skill-architect@skill-architect` at
version 0.5.0" — sits directly under the **GitHub** route. Running that route against the
live remote in an isolated `CLAUDE_CONFIG_DIR` reports **`Version: 0.4.3`**, because
`origin/main` is `10326b3`. The sentence becomes true at merge. Correct for a release
commit; recorded so the local-checkout **0.5.0** result is not read as covering the GitHub
route at this SHA.

## M7 / M8 · Informational, pre-existing, unchanged by this diff

- `profiler/go.mod:3` declares `go 1.27.1` — a **patch-pinned** directive on a release
  artifact, so a user on any Go below 1.27.1 cannot `go build ./...`. Not in the diff.
- `.claude-plugin/plugin.json` carries `displayName`; the codex/cursor/devin manifests do
  not. Untestable here. Not in the diff.

---

## Second sweep — dryness

All 7 DoD conditions re-run at `9c8ba53` (identical verdicts), all 10 diff files re-read,
all 8 version surfaces re-enumerated, every `.sh` checked for the executable bit (the five
at `100644` — `verdict-guard.sh` and `tests/lib/{harness,masked-path,mutation-runner}.sh`
— are all *sourced*, never invoked by path, confirmed at `skills/skill-audit/SKILL.md:108`
and `tests/test_skill.sh:3,18`; **clean**), `out-of-scope-check.sh` run directly
(`out-of-scope check: clean`) and confirmed wired at `tests/test_harness.sh:632`,
`profiler version` confirmed to read `profiler.AdapterVersion` at `cmd/main.go:48` rather
than a seventh literal, and the plan's **D5** re-checked against a real captured profile
(`grep '"state": *""'` → empty; `estimated_context_tokens` has no key at all — the pointer
held).

**The second pass added no new findings.**

## Summary

The release **does** install, run and validate as 0.5.0: it builds, vets, formats, passes
2649 shell assertions identically on both shells, passes Go tests under `-race`, installs
as a real plugin reporting `0.5.0`, emits `adapter_version: 0.5.0` in a real profile,
validates both skills with zero errors and zero warnings, reproduces all nine of its own
numbers and all four 0.4.3 baselines exactly, carries no tag, and carries no attribution.
It satisfies 6 of 7 DoD conditions outright and the 7th only because the user has not
merged yet.

**What it does not do is protect any of that.** M1 and M2 are one root: the invariants this
release depends on are enforced by author memory on every surface where the assert was not
written, and the release commit is the one artifact whose correctness is 98% in the two
files nothing checks. M3 and M4 are the same root in the ref namespace.

**Fix directions advisory — `bug-fixer` owns re-deriving:**
- M1/M2 → one assertion in `tests/test_skill.sh` beside the existing five, reading the
  manifest version and requiring a matching section in both release documents; and extend
  `.gitignore` so the DoD grep's own pattern set is exactly the ignore set (`NOTICE` and
  `docs/skillgate-*.md` are the two the current block misses).
- M3 → an immutable ref for the frozen tree, per the plan's original letter.
- M5 → a one-word move in the PR body.

**None of the seven seeded Go-comment findings are re-litigated here.**
