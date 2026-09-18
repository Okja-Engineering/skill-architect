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
- bash 3.2 safe: no associative arrays, no `mapfile`/`readarray`, no `declare -A`, no `local -n`, no `wait -n`, no globstar. ~~Every new `[[ ]]` that gates control flow is written as an explicit `if`, because a failing `[[ ]] && continue` is exempt from errexit only by position and that is the exact class this cluster is about.~~

  **Corrected at the C7 gate (G9), 2026-09-18.** That sentence was not true of the diff it described. `tests/lib/masked-path.sh:40,46` — now `:46,52` after the CDPATH repair — gate control flow as `[[ … ]] && continue` and `[[ … ]] || continue`, not as an explicit `if`. Most of the new `[[ ]]` was written as an `if`; "every" was not.

  The code is correct as written, and that is measured rather than assumed. A failing `&&`/`||` list is exempt from errexit on both supported bashes, list status included:

  ```
  bash -c 'set -euo pipefail; for i in 1 2; do [[ -e /nope ]] && continue; echo "reached body $i"; done; echo "loop finished"'

  bash 3.2.57 : reached body 1 / reached body 2 / loop finished   rc=0
  bash 5.3.15 : reached body 1 / reached body 2 / loop finished   rc=0
  ```

  So the record was wrong and the code was not, and the record is what changed. Rewriting two working lines to match a claim that overstated what had been done would have been the wrong direction.
- Committed and pushed per entry, five commits, never held uncommitted.
- Not merged. Not tagged.

## For the next cluster

The substrate now reports. A green run from `tests/test_skill.sh` means something it did not mean before, and `tests/test_f01.sh` will now notice a rule ID added under any of the three indirect forms. Two known holes remain, both owned elsewhere: `check-quality.sh` outside the guard (C3), and `draft-rewrite.sh`'s mktemp leak (M7 / G8-04).

---

# Round 2 — the C7 gate's findings, 2026-09-18

The gate returned **NOT CLEAN, G1 blocks**. Round 1 changed `assert`'s *shape* in one file; the *invariant* was enforced nowhere mechanically, and it was still open in `tests/test_walk.sh`. Round 1's own root sentence — "`tests/test_skill.sh` is the one suite never brought up to the other three's conventions" — was factually wrong about `test_walk.sh`.

Branch and worktree unchanged. Primary working tree never touched; no `checkout`, `reset`, `clean`, `stash` or `rebase` there. `.scuba/` is gitignored, so this record is outside the repository's history either way.

## Disposition

| ID | Sev | Disposition | Commit |
|---|---|---|---|
| G1 | P1 | **REAL, fixed at the root** | `17a7a93` + `8e9f9ec` (+ `216ee06`) |
| G2 | P2 | **REAL, fixed at the root** | `17a7a93` + `8e9f9ec` (+ `216ee06`) |
| G3 | P3 | **REAL, fixed at the root** | `17a7a93` |
| G4 | P3 | **REAL, fixed — and the class widened past the named line** | `8e9f9ec`, `06c5c78` |
| G5 | P3 | **REAL, fixed** | `8e9f9ec` |
| G6 | P3 | **REAL, fixed at the root (attribution, then the header)** | `8d33f4a` |
| G7 | P3 | **REAL, 3 of 5 fixed; 2 remain the stated 0.5.0 deferral** | `8d33f4a` |
| G8 | P3 | **REAL, fixed** | `8d33f4a` |
| G9 | P3 | **REAL, record corrected above** | n/a — record only |
| G10 | P3 | **Not touched.** Out of scope by instruction; a scope call for the manager. | n/a |

```
17a7a93 test(harness): assert the invariant the suites are supposed to hold, over all of them
8e9f9ec test: put all four suites on the shared harness, and repair the seven that could not fail
8d33f4a test(f01): attribute a rule ID to the script that can carry it to a consumer
06c5c78 fix(skill-audit): resolve a script's own directory without asking the caller's CDPATH
216ee06 test(harness): hold the suite glob and the CI workflow to the same list
```

Author and committer `imagineux <imagineux@gmail.com>` on all five. AI-attribution grep over commit messages, diff and both PR comments: **0**.

## The root, and why it is not seven call sites

G1, G2 and G3 are one defect: **four hand-maintained copies of the harness.** Each suite carried its own `assert`, its own counters and its own EXIT handling, so round 1's repair to `test_skill.sh` could not reach the other three, and a suite whose copy predated it kept the old shape. Patching `test_walk.sh`'s seven sites would have left the copies, and the copies are what re-open the class.

Two things changed, and between them they leave nowhere for the class to live:

**`tests/lib/harness.sh`** — one `assert` (runs a command), one `assert_value` (judges a verdict already computed as a string, which is the shape the two large suites' 819 assertions are written in), one EXIT handler, one scratch directory, one summary. All four suites source it; none defines any of those names.

**`tests/test_harness.sh` + `tests/lib/audit-suites.sh`** — the mechanical check, run over `tests/test_*.sh` by glob so a fifth suite is covered the day it lands. `audit-suites.sh` reads shell words (skipping heredoc bodies, since two suites embed Python that starts lines with `assert`) and refuses: a verdict that is a constant command, including under a transparent wrapper; a value-shaped verdict written as a literal; a redefinition of any harness name in any of bash's three spellings; and a private `trap … EXIT`. New CI step, first, before the suites it underwrites.

### Where we pushed back on the gate's direction

The gate advised "the abort guard **and the assertion shape** applied uniformly across all four suites". The guard: done, and structurally. The **shape**: not converted, deliberately.

G3 is the argument. A command-shaped `assert "label" true` is exactly the live G1 defect, so shape uniformity is neither necessary nor sufficient for the invariant. What the invariant needs is a mechanism that makes a constant verdict *detectable*, and at runtime it provably is not — a computed `"true"` and a written one are the same bytes, which is *why* G3 exists. That question is decidable only in the source text, and there it is decidable exactly, for **both** shapes, at the cost of no call-site churn. Converting `test_f01.sh` and `test_f02.sh`'s 819 assertions would have been a refactor larger than the bug, in the two suites whose job is to catch silent semantic change, for no gain in the invariant. Surfaced rather than done; the deferral is written into both suites' headers where a reader meets it.

## G1 — reproduced independently, then closed

```
$ perl -0pi -e 's/^license: .*\n//m' skills/skill-rewrite/SKILL.md
$ skills/skill-audit/scripts/check-frontmatter.sh skills/skill-rewrite
POLICY FAIL [PL001]: missing license (house policy)     rc=2

tests/test_walk.sh  bash 3.2.57 : rc=2  FAIL lines: 0  summary: ''
tests/test_walk.sh  bash 5.3.15 : rc=2  FAIL lines: 0  summary: ''
```

The label `skill-audit can audit skill-rewrite frontmatter` never printed, on either bash. Confirmed exactly as the gate reported.

`audit-suites.sh` found the same seven sites at the same seven lines, independently:

```
tests/test_walk.sh:32,162,164,170,172,174,176:
  verdict is the constant command 'true', so this assertion cannot fail
sites=253 files=4
```

### Each of the seven, mutated, both bashes

Every site's own condition broken, one at a time; every run reverted; tree clean after.

| site | mutation | bash 3.2.57 | bash 5.3.15 |
|---|---|---|---|
| `:32` plugin points to canonical skill directory | `"skills"` → `"nowhere"` in `.devin-plugin/plugin.json` | FAIL + `19 passed, 1 failed` rc=1 | identical |
| `:162` fixed skill frontmatter passes | reconstructed skill loses `license:` | FAIL + `19 passed, 1 failed` rc=1 | identical |
| `:164` fixed skill structure passes | reconstructed skill loses `## Examples` | FAIL + `19 passed, 1 failed` rc=1 | identical |
| `:170` skill-audit can audit skill-rewrite frontmatter | skill-rewrite loses `license:` | FAIL + `19 passed, 1 failed` rc=1 | identical |
| `:172` skill-audit can audit skill-rewrite structure | skill-rewrite loses `## Constraints` | FAIL + `19 passed, 1 failed` rc=1 | identical |
| `:174` skill-audit can audit itself frontmatter | skill-audit loses `license:` | FAIL + `19 passed, 1 failed` rc=1 | identical |
| `:176` skill-audit can audit itself structure | skill-audit loses `## Constraints` | FAIL + `19 passed, 1 failed` rc=1 | identical |
| *added* rewrite drafts a plan for a broken skill | `chmod -x draft-rewrite.sh` | FAIL + `16 passed, 4 failed` rc=1 | identical |

In every case the suite printed **that site's own label**, reached its summary, and exited 1.

The eighth is the one added check (19 → 20): `draft-rewrite.sh` at `:69` was a bare command with its effect asserted separately — the same shape, in the suite's own subject under test, so it became an assertion too.

## G3 — the meta-check, red then green

```
-- before: tests/test_harness.sh over the untouched tree
bash 3.2.57  rc=0  41 passed, 0 failed
bash 5.3.15  rc=0  41 passed, 0 failed

-- inject `assert "the moon is made of green cheese" true` into tests/test_skill.sh
-- the suite itself still goes green on the lie:
bash 3.2.57  rc=0  PASS: the moon is made of green cheese   28 passed, 0 failed
bash 5.3.15  rc=0  PASS: the moon is made of green cheese   28 passed, 0 failed

-- and the meta-check refuses it:
bash 3.2.57  rc=1  40 passed, 1 failed
    tests/test_skill.sh:125: verdict is the constant command 'true', so this
      assertion cannot fail: assert "the moon is made of green cheese" true
    FAIL: tests/test_skill.sh has no assertion that cannot fail, and no harness of its own
bash 5.3.15  rc=1  (byte-identical)

-- remove it
bash 3.2.57  rc=0  41 passed, 0 failed
bash 5.3.15  rc=0  41 passed, 0 failed
```

The same red/green is pinned permanently as 16 controls inside `test_harness.sh`: seven shapes of vacuous verdict (`true`, `:`, `echo …`, `quietly true`, no verdict at all, a literal value verdict, an assertion after a separator), four shapes of private harness (`name()`, `name ()`, `function name`, `trap … EXIT`), and five negative controls (a real command, a computed value, a variable verdict, a heredoc body that must not be read as shell, a non-EXIT trap that must be left alone). A check that can only ever say "found nothing" carries a control that makes it say the opposite.

The denominator is asserted too: the suite glob must find at least five suites, and the audit must report at least 200 call sites examined, so neither an empty glob nor a tokenizer that matched nothing can make the per-suite checks vacuously green.

## G2 — the abort guard, injected `false` in every suite

```
                                                                   bash 3.2.57   bash 5.3.15
test_harness  FAIL: suite aborted before reaching its summary (exit 1)  rc=1          rc=1
test_skill    FAIL: suite aborted before reaching its summary (exit 1)  rc=1          rc=1
test_walk     FAIL: suite aborted before reaching its summary (exit 1)  rc=1          rc=1
test_f01      FAIL: suite aborted before reaching its summary (exit 1)  rc=1          rc=1
test_f02      FAIL: suite aborted before reaching its summary (exit 1)  rc=1          rc=1
```

No summary printed in any of them, which is correct — the summary is the thing that was not reached. Before, on the same injection: `test_walk`, `test_f01` and `test_f02` gave `rc=1, FAIL lines=0, summary=''` on both bashes.

## G5 — the guard's diagnostic does not wait on the guard's cleanup

`harness_exit` drops `errexit` and `nounset` on the way in, and prints before removing anything. Pinned by a control that makes the cleanup genuinely fail (a scratch subdirectory `chmod 500`, so `rm -rf` cannot unlink through it):

```
PASS: the abort diagnostic survives a cleanup step that fails
PASS: a suite whose cleanup fails still exits nonzero
PASS: the control's own scratch directory is not left behind
```

## G4 — fixed, and the class was wider than the finding

`masked-path.sh:32` was the named line, and it is fixed. Looking for the rest of the class found the same `$(cd … && pwd)` in six more places, including every check script's `script_dir`.

Measured with a scratch directory holding `skills/skill-audit/scripts`, exported on `CDPATH`:

```
                                           BEFORE                      AFTER
check-paths.sh --json valid-full           rc=3, no verdict            rc=0, verdict returned
  "cannot load …/verdict-guard.sh: missing or malformed;
   no verdict was computed"
tests/test_f01.sh                          517 passed, 58 failed       575 passed, 0 failed
tests/test_walk.sh (colliding tests/)      would resolve elsewhere     20 passed, 0 failed
```

All five suites, both bashes, with `CDPATH` exported: 41/27/20/575/243, zero failed.

The `script_dir` half **fails safe** — the guard installed for precisely this reason caught it every time, which is why nobody had seen it. It was fixed anyway: half a class fixed is what opened G1, and the repair is one line in six identical copies, now spelled the same way as `harness.sh` and `masked-path.sh` so there is one way to do this in the repository rather than two.

## G6 — the attribution, then the header

Measured, with the tool masked:

```
check-frontmatter.sh, skill-validator absent
  rc=3   stdout: []   stderr: [ERROR: required tool not found: skill-validator]
  DEP001 anywhere: 0

audit-report.sh, jq absent
  rc=3   stdout: 0 bytes   stderr: [ERROR: required tool not found: jq]
  DEP001 anywhere: 0

check-structure.sh --json, jq absent   (the emit_json=true caller, for contrast)
  {"findings": [{"level":"fail","rule":"DEP001","message":"required tool not found: jq"}], …}
  DEP001 in payload: 1
```

So `require_tool <tool> false` carries no rule ID to any consumer, and the census credited DEP001 to every script with a line-initial `require_tool` regardless. It then demanded DEP001 in the header of a script that cannot emit it, and the header was duly changed to say so. **Attribution that is only ever too generous still forces a lie.** The argument is now read; the header follows.

Per-script attribution, both bashes:

```
                       before                                          after
audit-report.sh        DEP001 DEP002 PATH PL001 PL002-PL005 PT001 PT002   unchanged (DEP001 by relay)
check-frontmatter.sh   DEP001 DEP002 PL001                                DEP002 PL001
check-paths.sh         DEP001 PATH PT001 PT002                            unchanged
check-quality.sh       (none)                                             unchanged
check-structure.sh     DEP001 DEP002 PATH PL002-PL005 PT001 PT002         unchanged
verdict-guard.sh       DEP001                                             unchanged
ALL                    DEP001 DEP002 PATH PL001 PL002-PL005 PT001 PT002   unchanged
```

576 → 575, and the one dropped assertion is the false one. The global set is unchanged, so the registry comparison is untouched.

`check-frontmatter.sh`'s header had the same untruth and is corrected in the same pass: its IDs classify its exits, they are not printed, and an exit 3 without `--json` carries the message and no ID. `audit-report.sh` now enumerates DEP001 with the rest of the relayed set and says plainly that it builds no finding of its own for it — and that the exit above gets there first, since the only tool either script requires is the same jq.

## G7 — three of five, each proved against the old census

Each mutation run against the census as it stood at `be1b06c` and as it stands now.

| mutation | old | new |
|---|---|---|
| unmutated control | `576 passed, 0 failed` | `575 passed, 0 failed` |
| relay as `check-paths.sh "$dir" --json` (flag not adjacent) | `576 passed, 0 failed` | **`575 passed, 4 failed`** |
| object as `"rule":"ZZ001"` (no space after colon) | `576 passed, 0 failed` | **`574 passed, 2 failed`** |
| `cd . && require_tool jq true` (not line-initial) | `576 passed, 0 failed` | **`575 passed, 1 failed`** |
| an emitted ID nobody registered | `575 passed, 2 failed` | `574 passed, 2 failed` |
| a registered ID nobody emits | `575 passed, 1 failed` | `574 passed, 1 failed` |

Both original directions still go red, so nothing was weakened to buy the widening.

**Not closed, and named as such in the code rather than left implied:** variable indirection and a non-literal level. Both need a behavioural census — run the scripts and harvest the IDs they actually emit — which is the 0.5.0 item worklist entry P-c scoped. `audit-suites.sh`'s header states its own equivalent boundary: a verdict whose command word is itself an expansion is accepted, and a variable holding a constant is not traced.

## G8 — a prose word is no longer a rule

```
old census, registry line reading "Exit codes: the JSON payload names each one. …"
  575 passed, 1 failed     <- "JSON" registered as a rule
new census, same line
  575 passed, 0 failed
```

The ten IDs on `skills/skill-audit/SKILL.md:87` are now set in backticks and read as backticked tokens. Still open to any new prefix — the whitelist the old comment rightly refused is not what replaced it.

**Also fixed while in there, unprompted and in this cluster's own class:** both reads that build the comparison exited 1 under `errexit`/`pipefail` when they found nothing, killing the suite at the assignment. "The registry line is gone" and "it names no rule" are two of the things the assertions below them exist to report. With the backticks stripped: `573 passed, 2 failed` and both labels print, where before the suite died with no verdict at all.

## One residual the mechanism did not reach — `216ee06`

Everything `test_harness.sh` asserts holds only over the suites that actually execute, and CI names its steps one by one. A suite added to `tests/` and not to the workflow reports nothing at all — the same outcome as an assertion that cannot fail, reached from the other side. The glob and the workflow are now held to the same list.

```
copy a suite in under a name the workflow does not mention
  FAIL: tests/test_unwired.sh is a step in the CI workflow
  49 passed, 1 failed   rc=1
remove it
  46 passed, 0 failed   rc=0   (both bashes)
```

## Final verification, at `216ee06`

```
                  bash 3.2.57            bash 5.3.15
test_harness.sh   46 passed, 0 failed    46 passed, 0 failed
test_skill.sh     27 passed, 0 failed    27 passed, 0 failed
test_walk.sh      20 passed, 0 failed    20 passed, 0 failed
test_f01.sh       575 passed, 0 failed   575 passed, 0 failed
test_f02.sh       243 passed, 0 failed   243 passed, 0 failed

audit-suites.sh tests/test_*.sh   ->  sites=284 files=5   rc=0
```

Identical run from `/` and from `/var/tmp`, and identical with `CDPATH` exported.

```
cd profiler
go build ./...        OK
go vet ./...          OK
gofmt -l .            (clean)
go test -race -count=1 ./...
                      ok  github.com/Okja-Engineering/skill-architect/profiler       1.301s
                      ok  github.com/Okja-Engineering/skill-architect/profiler/cmd   2.059s
```

Assertion deltas this round: `test_walk.sh` 19 → 20 (one bare command became an assertion); `test_f01.sh` 576 → 575 (the false DEP001 attribution dropped); `test_skill.sh` and `test_f02.sh` unchanged; `test_harness.sh` new at 46.

Temp-directory leaks, per suite, measured against the Darwin user temp dir: `test_harness` 0, `test_f01` 0, `test_f02` 0; `test_skill` and `test_walk` each leave one **file**, which is `draft-rewrite.sh`'s own `mktemp` — G8-04 / cluster M7, pre-existing, out of scope. Nothing leaves a directory behind, including the control that deliberately makes cleanup fail.

## Constraint compliance

- **No version surface touched.** All five `0.4.1` literals in `tests/test_skill.sh` preserved verbatim — `git diff be1b06c..HEAD -- tests/test_skill.sh | grep -c 0.4.1` is **0**, and the file still holds 5. PR #5's bump is not pre-empted.
- `profiler/` untouched. `AdapterVersion` untouched.
- bash 3.2 safe: no `declare -A`, `mapfile`, `readarray`, `local -n`, `wait -n`, globstar or `${v,,}` in any new file; 15/15 shell files parse under both 3.2.57 and 5.3.15.
- **No PATH mirror of symlinks was built and no stub was written anywhere.** For the two G6 measurements the repo's own `tests/lib/masked-path.sh` was used read-only. A directory of real file *copies* was attempted first, as instructed, and does not work on this machine: a copy of `/usr/bin/dirname` outside its original path silently produces empty output, so a real-file mirror of Apple system binaries cannot mask anything here. Noted for whoever writes that instruction next.
- Author and committer `imagineux <imagineux@gmail.com>`; AI-attribution grep 0 before each of the four pushes.
- Committed and pushed per finding, four commits, never held uncommitted.
- Not merged. Not tagged. G10 not touched.

## Environment note, not a finding

Part-way through this round `/usr/bin/git` and `/usr/bin/python3` began failing with *"You have not agreed to the Xcode license agreements"* — `xcode-select -p` points at `/Applications/Xcode.app`, whose version has moved past the agreed `26.4.1`. Every command here was run with `DEVELOPER_DIR=/Library/Developer/CommandLineTools` as a result. The four `python3` assertions in `test_skill.sh` and one in `test_walk.sh` fail without it, on an unmodified tree. `sudo xcodebuild -license` clears it. Nothing to do with this PR, but it will bite the next person on this machine.

## For the next cluster, revised

The invariant now has a keeper. A green run means the assertions were capable of failing, because a check that runs over every suite says so, and a suite that dies says it died. Three holes remain, each owned elsewhere and each named where a reader will meet it:

- the **behavioural** census — run the assertions and see which can be made to fail — 0.5.0, worklist P-c. `audit-suites.sh` reads text, and its header says what that cannot reach.
- `check-quality.sh` outside the shared guard — **C3**.
- `draft-rewrite.sh`'s `mktemp` leak — **M7 / G8-04**.
- `test_f01.sh:131` still cannot distinguish "skillscore scored it" from "skillscore returned a stub" — the gate's observation, not a finding, and not touched.
