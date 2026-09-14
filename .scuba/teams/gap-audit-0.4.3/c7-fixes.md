# Cluster C7 — Verification integrity — fix record

**Branch** `fix/0.4.3-verification-integrity`, cut from `origin/main` at `5c847e1`.
**Worktree** `/Users/matthewvandusen/Development/Auraprix/skill-architect-wt/c7-verification`.
**PR** https://github.com/Okja-Engineering/skill-architect/pull/6 — open, not draft, base `main`. **Not merged, not tagged.**

Primary working tree never touched. No `checkout`, `reset`, `clean`, `stash` or `rebase` there.

## Disposition

| ID | Sev | Disposition | Commit | File(s) |
|---|---|---|---|---|
| G7-01 | P1 | **REAL, fixed** | `848ebe7` (+ `be1b06c` self-review) | `tests/test_skill.sh` |
| G7-02 | P2 | **REAL, fixed** | `848ebe7` (same repair as G7-01) | `tests/test_skill.sh` |
| G7-03 | P2 | **REAL, fixed** (widening; behavioural census still 0.5.0) | `3a3224e` | `tests/test_f01.sh`, `skills/skill-audit/scripts/audit-report.sh` |
| G7-04 | P3 | **REAL, fixed** | `16ad204` | `tests/lib/masked-path.sh` |
| G7-05 | P2 | **REAL, fixed** | `2ec1899` | `.github/workflows/ci.yml` |

Nothing classified INVALID. All five reproduced against `5c847e1` before being touched.

Commits, oldest first:

```
848ebe7 test(skill): make assert run a command so a failed check can report FAIL
3a3224e test(f01): census the rule IDs a script can emit, not the ones it spells
16ad204 test(lib): build a masked-PATH farm that masks one binary and shadows none
2ec1899 ci: pin skillscore, the one test dependency npm could change under us
be1b06c test(skill): give the scratch directory to the shell that installed the trap
```

Author and committer `imagineux <imagineux@gmail.com>` on all five. AI-attribution grep over commit messages, diff and PR body: **0**.

## The root, and why one repair covers G7-01 and G7-02

`assert` took a *value*, not a condition. That single shape is what produced both the 16 literal-`true` call sites and the 11 inert `[[ ]]` guards, and it is why the missing `cd` was invisible: from the wrong cwd the guards failed and the assertions printed PASS anyway. Making `assert` take a command removes the class — there is no value to hand in — rather than patching 16 sites, which would have left the shape to re-open as the next round's finding.

Verified premise, measured rather than assumed:

```
$ cat probe.sh
set -euo pipefail
[[ -f /nonexistent-xyz ]]
echo "REACHED-AFTER-BRACKET"

bash 3.2.57 : REACHED-AFTER-BRACKET     <- errexit does NOT fire for [[ ]]
bash 5.3.15 : (nothing)                 <- errexit DOES fire
```

Both are broken, differently: 3.2 runs on and prints PASS; 5 aborts with no verdict. Neither reports.

## Mutation evidence — G7-01, both bash versions

Mutation: `rm skills/skill-audit/references/best-practices.md`.

### BEFORE (suite at `5c847e1`)

bash 3.2.57 — a PASS for a file that does not exist:

```
PASS: skill-audit check-structure.sh is executable
PASS: skill-audit best-practices reference exists      <- the file is deleted
PASS: skill-audit evaluation-matrix reference exists
EXIT=1        (from an unrelated downstream abort, not from the deleted file)
```

bash 5.3.15 — aborts at the `[[ ]]`, never mentions it:

```
PASS: skill-audit check-frontmatter.sh is executable
PASS: skill-audit check-structure.sh is executable
EXIT=1
```

`FAIL:`-or-summary lines in either run: **0**. Requirement was "print FAIL, reach the summary, exit 1". Zero of three, on both.

### AFTER (at HEAD) — byte-identical on both

```
FAIL: skill-audit best-practices reference exists
FAIL: skill-audit passes its own frontmatter check
FAIL: skill-audit passes its own structure check
FAIL: skills/skill-audit validates with zero errors and zero warnings

23 passed, 4 failed
exit=1
```

Three of three, on bash 3.2.57 and bash 5.3.15.

### Abort guard

Injecting a bare `false` mid-suite, both bashes:

```
FAIL: suite aborted before reaching its summary (exit 1)
```

## G7-02 evidence

Before, by absolute path from `/`: `PASS: .devin-plugin/plugin.json exists`, then a Python `FileNotFoundError`. After, from `/` and from `/var/tmp`, both bashes: `27 passed, 0 failed`, exit 0.

## G7-03 evidence

| mutation on `5c847e1` | before | after |
|---|---|---|
| `cannot_compute "XX003" "mutant" true` in `check-paths.sh`, ID **quoted** | `561 passed, 0 failed` | `575 passed, 4 failed` |
| `ZZ009` registered in `SKILL.md`, emitted nowhere | `561 passed, 0 failed` | `575 passed, 1 failed` |

The second is the mirror direction: the registry side matched a fixed prefix whitelist, so it could not grow either. Both sides now read a rule-shaped token.

Per-script attribution, identical on both bashes:

```
                       before                          after
audit-report.sh        PL001                           DEP001 DEP002 PATH PL001 PL002 PL003 PL004 PL005 PT001 PT002
check-frontmatter.sh   DEP002 PL001                    DEP001 DEP002 PL001
check-paths.sh         PATH PT001 PT002                DEP001 PATH PT001 PT002
check-quality.sh       (none)                          (none)
check-structure.sh     DEP002 PL002 PL003 PL004 PL005  DEP001 DEP002 PATH PL002 PL003 PL004 PL005 PT001 PT002
verdict-guard.sh       DEP001                          DEP001
```

`DEP001` now reaches all four indirect emitters via `require_tool`; the `findings+=` relay at `check-structure.sh:158` and the jq splice at `audit-report.sh:189` are both followed, transitively and cycle-guarded.

**Scope grew by one file, deliberately.** The honest census immediately failed `audit-report.sh`'s header for `DEP002`, `PATH`, `PL002`, `PL003`, `PL004` — it relays them and named none of them, writing the policy rules as ranges. Fixed in the header comment. Comment only, no behaviour. Chosen over teaching the check to parse range notation, which would have weakened the check to preserve a false header.

**Outstanding, as instructed.** The census is still vacuous over `check-quality.sh`: it emits no rule IDs at all because it is not inside the shared guard. **That is C3's to close, and it is not closed here.** Separately, a genuinely behavioural census (run the scripts, harvest emitted IDs) remains the 0.5.0 item worklist item P-c scoped it as; this commit is the cheap widening P-c assigned to 0.4.3.

## G7-04 evidence

Fixture: `relbin/` holding a working `grep`, placed first on `PATH` as a relative entry.

```
                       BEFORE                  AFTER
farm/grep           -> relbin/grep          -> /abs/.../relbin/grep
farm/grep resolves?    NO-BROKEN               yes
grep under farm        exit=127                exit=0
broken links in farm   4                       0
tool asked to hide     still hidden            still hidden
tests/test_f01.sh      556 passed, 5 failed    576 passed, 0 failed
```

All five before-failures were jq-masked cases that died at 127 before reaching the script they were written to test. Both invariants land: `[[ -e … || -L … ]]` for the dedupe test, and an absolutised link target. Nothing enters the farm under a name that does not resolve, which also drops the literal `dir/*` an empty directory's glob leaves behind.

## G7-05 evidence

CI-only, so proved by exercising the coupling locally rather than by running the workflow. Same tree, no repo change, only the dependency version:

```
skillscore 2.0.2  ->  243 passed, 0 failed
skillscore 1.2.1  ->  240 passed, 3 failed
    FAIL: valid-full: summary.quality_score is 80
    FAIL: valid-full: summary.quality_grade is B-
    FAIL: valid-full: quality has 7 categories
```

The registry has already shipped a major version (1.0.0 … 2.0.2). Pinned to `skillscore@2.0.2`, the version the suite is green against. One line.

## Self-review finding, fixed in `be1b06c`

`quietly` captures output through a command substitution, so a helper that created its own scratch directory assigned it inside a subshell and the EXIT trap never saw it — one `skill-rewrite-test` directory survived every run. Fixed by giving the directory to the shell that installed the trap, so no helper can lose a handle it does not own. Measured: zero `skill-rewrite-test` directories remain after a run on either bash.

The one temp directory still left behind per run is `draft-rewrite.sh`'s own — **G8-04 / cluster M7**, not touched here.

## Final verification, at HEAD

```
                bash 3.2.57            bash 5.3.15
test_skill.sh   27 passed, 0 failed    27 passed, 0 failed
test_walk.sh    19 passed, 0 failed    19 passed, 0 failed
test_f01.sh     576 passed, 0 failed   576 passed, 0 failed
test_f02.sh     243 passed, 0 failed   243 passed, 0 failed
```

```
cd profiler
go build ./...        OK
go vet ./...          OK
gofmt -l .            (clean)
go test -race ./...   ok  github.com/Okja-Engineering/skill-architect/profiler       1.380s
                      ok  github.com/Okja-Engineering/skill-architect/profiler/cmd   2.363s
```

Assertion deltas: `test_f01.sh` 561 -> 576 (attribution, not new rules); `test_skill.sh` 27 -> 27; `test_f02.sh` and `test_walk.sh` unchanged.

## Constraint compliance

- `profiler/` untouched; `AdapterVersion` untouched; no version surface changed. All five `0.4.1` literals in `tests/test_skill.sh` preserved verbatim, so the in-flight version-bump PR still finds them.
- bash 3.2 safe: no associative arrays, no `mapfile`/`readarray`, no `declare -A`, no `local -n`, no `wait -n`, no globstar. Every new `[[ ]]` that gates control flow is written as an explicit `if`, because a failing `[[ ]] && continue` is exempt from errexit only by position and that is the exact class this cluster is about.
- Committed and pushed per entry, five commits, never held uncommitted.
- Not merged. Not tagged.

## For the next cluster

The substrate now reports. A green run from `tests/test_skill.sh` means something it did not mean before, and `tests/test_f01.sh` will now notice a rule ID added under any of the three indirect forms. Two known holes remain, both owned elsewhere: `check-quality.sh` outside the guard (C3), and `draft-rewrite.sh`'s mktemp leak (M7 / G8-04).
