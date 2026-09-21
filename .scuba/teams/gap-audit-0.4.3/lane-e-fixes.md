# Lane E / Cluster C5 — asserted completeness · branch `fix/0.4.3-asserted-completeness`

Base `2c155e7`. Worktree
`/private/tmp/claude-501/-Users-matthewvandusen-Development-Auraprix-skill-architect/31edaf48-6b19-4833-82e7-6cb52dac691f/scratchpad/wt-c5`.
The primary tree was never touched. Author and committer `imagineux <imagineux@gmail.com>`
on every commit; attribution grep over `2c155e7..HEAD` returns 0.

**Every count and line number below was measured at `2c155e7`, not carried from a record.**

---

## Line numbers, re-measured

| record said | measured at `2c155e7` | verdict |
|---|---|---|
| preflight at `skill-audit/SKILL.md:46-47` | `:46-47` exactly — the two `command -v` guards | record correct |
| preflight at `skill-audit/SKILL.md:51-52` (lane-d's original) | `:51` is the `check-quality.sh` invocation, `:52` the closing fence | record wrong, not used |
| `skill-audit/SKILL.md:5` compatibility | `compatibility: POSIX shell (bash 3.2+ or zsh), git.` | record correct |
| `verdict-guard.sh` absent from `skill-audit/SKILL.md` | absent; named in 14 files tree-wide | record correct |
| "the four house-policy checks" | reads **three** at `:103` | **already closed** |
| suite baseline | harness 50 · skill 35 · walk 20 · rewrite 94 · f01 1856 · f02 350 = **2405** | measured, not assumed |

---

## Dispositions

### A1 — the Stage 2 preflight · **FIXED** · red `514ac8a`, fix `b92a0a2`

Two defects in one list: it exited **1** (which this skill's own exit table reserves for a
spec or path failure — a verdict about the audited skill the preflight has not computed)
and it named two tools where the commands it gates require seven.

Reproduced on the surface the defect lives on, `jq` masked with the repo's own
`tests/lib/masked-path.sh` (read-only, nothing installed or removed):

```
--- step 1: run the documented preflight with jq absent ---
preflight exit=0   (0 means it told the reader to go ahead)
--- step 2: run the first documented headline command with jq absent ---
check-frontmatter.sh exit=3
stdout bytes=0
stderr: ERROR: required tool not found: jq
```

**Root, and why the fix is not a longer list.** The preflight *restated* a dependency
contract the scripts already own, which is the identical shape the exit-code section of
this same document had already been repaired for ("it drifted because it restated rather
than derived"). So the holder derives instead of comparing against a list. Each candidate
tool is masked in turn, the three commands the block documents are run, and the preflight
must refuse **exactly when one of them refuses and with the status it refused with**.
Nothing is written down in the suite:

```
Stage 2 commands read from SKILL.md: check-frontmatter.sh check-quality.sh check-structure.sh
preflight candidate tools walked: awk date dirname grep jq skill-validator skillscore wc
of those, a Stage 2 command refuses without: awk dirname grep jq skill-validator skillscore wc
of those, no Stage 2 command needs: date
```

`date` is the negative control: required by `audit-report.sh`, which Stage 2 does not run,
so a preflight that refused unconditionally fails too. Both halves of the walk are asserted
populated.

Red → green → red, on the reproduction surface:

```
FAIL: the Stage 2 preflight refuses exactly when a command it gates refuses, and with that command's status
  preflight and the commands it gates disagree: awk(preflight:0 gated:3) dirname(preflight:0 gated:3)
  grep(preflight:0 gated:3) jq(preflight:0 gated:3) skill-validator(preflight:1 gated:3)
  skillscore(preflight:1 gated:3) wc(preflight:0 gated:3)
  1863 passed, 1 failed
```

```
with the fix      preflight exit=3, "required tool not found: jq"
fix reverted      preflight exit=0, then check-frontmatter.sh exit=3, stdout bytes=0
```

### A1b — the sentence A1 added was itself an over-claim · **FIXED** · `d0e3150`

The sentence said the preflight names every tool the commands "refuse to run without". That
is an exhaustiveness over every external a script might touch; the holder walks the tools
those scripts *state* as preconditions. The gap is not hypothetical — PR #10 found the same
shape in the drafter's `rm`. Reworded to "state as a precondition", with the scope named in
the sentence. This is the cluster's own rule applied to the cluster's own prose.

### A2 — `verdict-guard.sh` named nowhere · **FIXED** · red `331af2f`, fix `129651d`

All five scripts load it and refuse without it; nothing a reader runs names it, because it
is sourced rather than invoked. The document now names it where exit 3 is explained.

**Nothing in the new sentence asserts an absence** — the sibling's first attempt claimed the
guard was referenced by no other file, and a recursive grep finds it in 12. The claim held
is positive and derived from `$SCRIPTS_DIR`: five scripts load it, and five is computed.

Non-vacuous twice. Sentence reverted:

```
FAIL: SKILL.md names verdict-guard.sh, the file every one of them refuses to run without
FAIL: SKILL.md's count of the scripts that load the guard is the number that do (5)
  SKILL.md says: <no such sentence>
  1866 passed, 2 failed
```

Count alone mutated to "four":

```
FAIL: SKILL.md's count of the scripts that load the guard is the number that do (5)
  SKILL.md says: four
  the scripts say: five
  1867 passed, 1 failed
```

### A3 — the compatibility line, and the machinery that held only one file · **FIXED** · red `8d0895b`, fix `0cc69cb`

`skills/skill-audit/SKILL.md:5` declared POSIX shell with zsh, and `git`. False three ways,
each measured:

```
$ zsh skills/skill-audit/scripts/check-frontmatter.sh tests/fixtures/f01/valid-minimal
check-frontmatter.sh:32: BASH_SOURCE[0]: parameter not set
ERROR: cannot load <cwd>/verdict-guard.sh: missing or malformed; no verdict was computed
exit=3
```

no bundled shebang names `sh`, and no script in either skill runs `git`. Corrected to
`compatibility: bash 3.2+.`, matching the sibling. The sibling's direction — correct the
claim rather than make the scripts zsh-clean — was not reopened.

**The parameterised assertion.** The machinery read one `$SKILL`. It moved from
`tests/test_rewrite.sh` to `tests/test_skill.sh` — the suite over the shipped skills — and
is walked over a **derived denominator, `skills/*/`**, not a written pair. Nothing is
duplicated: both blocks are removed from the donor, which keeps its own `Required tools:`
census, and a cross-reference is left at the removal site. The same move carried the refusal
of any "named nowhere else" claim, now applied to both documents, which is what holds A2's
new sentence.

The per-skill behavioural witness is a run of that skill's own bundled script that reaches a
**verdict** — the `PL001` its fixture earns for skill-audit, the audit carried into the draft
for skill-rewrite — not an exit status, because under zsh the drafter exits 0 having written
a draft of two file-not-found lines. **A skill with no witness fails**, which is what makes
the glob a denominator rather than a loop.

Non-vacuous in both directions — the walk reaches each sibling independently:

```
this line reverted          skill-rewrite's line mutated to the old text
FAIL: skill-audit works under zsh …        FAIL: skill-rewrite works under zsh …
FAIL: skill-audit … names git …            FAIL: skill-rewrite … names git …
FAIL: skill-audit … and no other family    FAIL: skill-rewrite … and no other family
PASS: skill-rewrite … and no other family  PASS: skill-audit … and no other family
53 passed, 3 failed                        53 passed, 3 failed
```

Also proved: the witness itself refuses (`the witness refuses skill-audit under zsh`), and
the two line-readers still find what they must on the old literal line.

---

## Needed no edit — an earlier cluster had already closed them

### The "four house-policy checks" count — **closed by `2c155e7` (C1/C3/C4)**

At head the sentence reads **three**, and it is not merely correct, it is derived:
`tests/test_f01.sh` sorts the scripts into verdict-carriers and generators from their own
`# Exit codes:` headers and compares the prose numeral. Proved the holder reddens rather
than assuming it:

```
$ sed -i 's/^The three house-policy checks/The four house-policy checks/' …
FAIL: SKILL.md's count of the checks that carry a verdict in the exit status is the table's own (3)
  SKILL.md says: four
  the table says: three
```

### G5-03's skill-audit half, "cannot fall behind" — **closed by C7 (`71cc866`)**

`SKILL.md:110` now says the ten rule IDs "are the whole set these scripts emit … so that
neither side can grow without the other". The finding was that the census caught *removal*
and not *addition*, which is the direction the sentence claims. Measured with G7-03's own
stated proof — an unregistered `cannot_compute "XX003"` added to `check-paths.sh`:

```
FAIL: SKILL.md registers every rule ID the scripts emit, and registers no ID none of them emits
  emitted   : DEP001 DEP002 PATH PL001 PL002 PL003 PL004 PL005 PT001 PT002 XX003
FAIL: audit-report.sh header registers XX003, which it emits
FAIL: check-paths.sh header registers XX003, which it emits
FAIL: check-structure.sh header registers XX003, which it emits
  1867 passed, 4 failed
```

The direction the sentence claims is now held. **No edit.** Script restored byte-for-byte.

### `SKILL.md:90`, "neither side can move without the other" (exit table) — **true and held**

Measured with a phantom row for a script that does not exist:

```
  exit-table rows read from SKILL.md: 6
FAIL: SKILL.md's exit table was read, not matched as an empty set
FAIL: every row in SKILL.md's exit table names a script that exists
  rows naming no script: check-extra.sh
```

**No edit.**

---

## Left for the in-flight PRs

### The readme's "run the tests" block — **no edit; it would collide, and the root is closed in PR #8**

PR #8 rewrote that exact block and the sentence under it. It did not leave the list
hand-maintained: `tests/test_harness.sh` derives the denominator from the `tests/test_*.sh`
glob, requires the CI workflow to equal it, and requires every suite in it to appear in the
README block. The list is still written, but it **cannot fall behind**, and the holder
reddens. Editing README.md from this lane would be a head-on conflict for no gain.

**What must happen at merge, measured on a trial merge of PR #8 into this head.** My files
merge clean (no conflict in `skills/skill-audit/SKILL.md`, `tests/test_f01.sh`,
`tests/test_skill.sh`, `tests/test_rewrite.sh`); PR #8 conflicts with `main` in
`.github/workflows/ci.yml`, `tests/lib/audit-suites.sh` and `tests/lib/harness.sh` — the
already-recorded 8×9 conflicts, not mine. On the merged tree:

```
tests/ on disk : harness f01 f02 install rewrite skill walk   (7)
CI workflow    : harness f01 f02 install skill walk           (6)
README block   : harness skill install walk f01 f02           (6)

FAIL: tests/ holds exactly the suites the CI workflow runs
REFUSED: the precondition above failed, so nothing that depended on it ran
  1 passed, 1 failed
```

**`tests/test_rewrite.sh` must be added to both the CI workflow and the README block at
merge resolution** — it landed on `main` in C8 after PR #8 branched. This is the same
union-not-pick-a-side rule already recorded for the ci.yml conflict, and it fails *closed*:
the precondition refuses everything downstream, so the whole meta-check is lost, not one row.

With the workflow fixed and the README left short, the README half is isolated and reddens
on its own, so the mechanism is genuinely non-vacuous:

```
FAIL: tests/test_rewrite.sh is in the README's list of the tests to run
  74 passed, 1 failed
```

### Not touched, by mandate

- **No version surface.** The adapter-version bump to 0.4.3 and the release-note line for the
  new `jq` dependency are still owed by the release commit.
- **The skill metadata versions are an open user decision** — `skill-audit` `0.2.0` and
  `skill-rewrite` `0.1.0` are untouched and still need that decision.
- `README.md`, `tests/lib/`, `profiler/`, `docs/profiler-spec.md`: nothing written. The
  `docs/profiler-spec.md` entries in C5 (G5-06) belong to the profiler PRs' surface and were
  not in this lane's routed set.

---

## Verification

Nothing outside `skills/skill-audit/SKILL.md`, `tests/test_f01.sh`, `tests/test_skill.sh`
and `tests/test_rewrite.sh` was touched.

| suite | bash 3.2.57 | bash 5.3.15 | at base `2c155e7` |
|---|---|---|---|
| `test_harness.sh` | 50 / 0 | 50 / 0 | 50 |
| `test_skill.sh` | **53** / 0 | **53** / 0 | 35 |
| `test_walk.sh` | 20 / 0 | 20 / 0 | 20 |
| `test_rewrite.sh` | **85** / 0 | **85** / 0 | 94 |
| `test_f01.sh` | **1868** / 0 | **1868** / 0 | 1856 |
| `test_f02.sh` | 350 / 0 | 350 / 0 | 350 |
| **total** | **2426 / 0** | **2426 / 0** | 2405 |

`test_rewrite.sh` 94 → 85 and `test_skill.sh` 35 → 53 is the relocation: 9 assertions left
the donor, 18 arrived in the shipped-skills suite walked over both skills. Net +21 overall.

```
profiler$ go build ./...      OK
profiler$ go vet ./...        OK
profiler$ gofmt -l .          (no output)
profiler$ go test -race ./...
ok  github.com/Okja-Engineering/skill-architect/profiler      1.443s
ok  github.com/Okja-Engineering/skill-architect/profiler/cmd  2.256s
```

## Residual

One single-letter typo ("reden") in the body of pushed commit `0cc69cb`. Not force-pushed:
the branch is published and the cost of rewriting it exceeds the cost of the typo.
