# C7 gate — PR #6 `fix/0.4.3-verification-integrity` @ `be1b06c`

**Verdict: NOT CLEAN. G1 blocks.**

All five scoped entries are genuinely fixed and every claim in `c7-fixes.md` reproduced,
including the byte-identical mutation output. What blocks is that **the class G7-01 exists to
close is provably still open in `tests/test_walk.sh`**, which the C7 root analysis treats as
already conforming.

> Persisted by the chief of staff: the hunter's session had Write disabled.
> **Verified by CoS** notes are independent confirmation.

## Coverage

5/5 diff files · 9/9 hunks (199+/76-) · 5/5 worklist entries with each stated non-vacuity
proof re-run independently · **27/27 `assert` call sites in the rewritten `tests/test_skill.sh`
individually mutation-probed** · 4/4 suites x 2 bash versions (3.2.57 and 5.3.15) · 6
additional adversarial mutations. Second full pass added nothing.

## Findings

| # | file:line | Sev | Class |
|---|---|---|---|
| G1 | `tests/test_walk.sh:32,162,164,170,172,174,176` + `:21` | **P1** | REAL — same defect as G7-01, unclosed |
| G2 | `test_f01.sh:158`, `test_f02.sh:246`, `test_walk.sh:21` | P2 | REAL — abort guard in 1 of 4 suites |
| G3 | `tests/test_skill.sh:17-27` | P3 | REAL — vacuous call site still constructible |
| G4 | `tests/lib/masked-path.sh:32` | P3 | REAL — CDPATH subverts the new resolution |
| G5 | `tests/test_skill.sh:45-58` | P3 | REAL — the abort guard is not abort-proof |
| G6 | `skills/skill-audit/scripts/audit-report.sh:24-26` | P3 | REAL — new comment states an untruth |
| G7 | `tests/test_f01.sh:838-873` | P3 | REAL — 5 census blind spots, comment overclaims |
| G8 | `tests/test_f01.sh:889` | P3 | REAL — prose token counted as a registered rule |
| G9 | `c7-fixes.md` vs `masked-path.sh:40,46` | P3 | REAL — record drift, not a code defect |
| G10 | `.github/workflows/ci.yml:13,14,16,32` | P3 | SUSPECTED — pin class narrowed, not closed |

---

## G1 — the class is open in `tests/test_walk.sh` · P1 · BLOCKS

`test_walk.sh:9-18` **already carries** the command-shaped `assert` this PR installs in
`test_skill.sh` — and it has **7 call sites passing the literal `true`**, with the real check
on the preceding line as a bare command:

```
161  skills/skill-audit/scripts/check-frontmatter.sh "$fixed" >/dev/null
162  assert "fixed skill frontmatter passes" true
```

The preceding commands are not `[[ ]]`, so errexit **does** fire on bash 3.2 — producing the
bash-5 half of the C7 symptom on **both** bashes. `test_walk.sh:21`'s trap is `rm -rf "$tmp"`
only, with no summary guard.

Reproduced by breaking a `license:` field so the bare check at `:169` exits 2:

```
test_walk.sh (mutated) bash 3.2 : rc=2  FAIL lines: 0  summary: ''
test_walk.sh (mutated) bash 5.3 : rc=2  FAIL lines: 0  summary: ''
```

Those 7 labels can never print FAIL on any bash the project supports. The worklist's C7 root —
"`tests/test_skill.sh` is the one suite never brought up to the other three's conventions" —
is **factually wrong about `test_walk.sh`**, so this is in-scope drift, not new scope.
`test_f01.sh` and `test_f02.sh` are clean on this axis.

> **Verified by CoS.** `test_walk.sh` at head has the `if "$@"` helper and exactly 7
> `assert … true` call sites at the lines named. Confirmed REAL.

## G2 — the abort guard reaches 1 of 4 suites · P2

`summary_printed` and `cleanup` exist only in `test_skill.sh`. Injecting a bare `false` into
`test_f01.sh`: `rc=1, FAIL lines=0, summary=''`. A suite that dies before its summary still
reports nothing in three of four suites.

## G3 — "passing a literal becomes impossible" is false · P3

The new `assert` runs `"$@"`, and **a literal truthy command is still a literal**:

```
PASS: the moon is made of green cheese          (assert "…" true)
PASS: every user is authenticated               (assert "…" :)
PASS: the database is encrypted at rest         (assert "…" echo checking...)
PASS: CI is green                               (assert "…" quietly true)

32 passed, 0 failed        exit=0
```

Nothing in the repo detects this. Not hypothetical — **G1 is 7 live instances.**

Minor related: `assert "x" ! test -e y` yields `!: command not found` and FAILs for the wrong
reason. `test ! -e` is the correct form.

> **Verified by CoS.** The helper at `test_skill.sh:17-27` is `if "$@"`, identical in shape to
> the one in `test_walk.sh`. `assert "label" true` passes. Confirmed REAL.

### Shared root of G1 + G2 + G3

The repair changed `assert`'s **shape** in one file. The **invariant** — an assertion's verdict
is a function of repository state, and a suite that does not reach its summary says so — is
enforced nowhere mechanically. One root fix (a meta-check over every suite, plus the guard and
shape applied uniformly) closes all three. **Patching `test_walk.sh`'s 7 sites alone re-opens
as the next round's finding.**

---

## The advisory P3s

- **G4** `masked-path.sh:32`'s `cd "$d" && pwd -P` is CDPATH-sensitive. With `CDPATH` exported
  and a colliding basename, `cd` resolves elsewhere and echoes the path, so the value is two
  lines and the binary is dropped — reintroducing the vacuous exit-127 outcome G7-04 exists to
  prevent, through a narrower door.
- **G5** errexit is live inside `cleanup`, so a failing step before `exit "$code"` skips the
  diagnostic the guard exists to print. Direction is safe, never green-on-red, but a green run
  can exit nonzero silently.
- **G6** the new `audit-report.sh` header says it raises DEP001 for itself when jq is absent.
  Measured: exit 3, **empty stdout**, no rule ID. `require_tool jq false` means
  `cannot_compute` never emits the ID, and it precedes the child call so it cannot arrive by
  relay. Root: the census's new rule ignores `require_tool`'s `emit_json` argument, and that
  over-attribution is what forced the sentence.
- **G7** five census blind spots, all reproduced: variable indirection, non-literal level,
  non-adjacent `--json`, a `printf` without exactly `": "`, and a non-line-initial
  `require_tool`. The first two fall under the deliberate 0.5.0 deferral; **the last three are
  inside the forms the widening claims to have closed.**
- **G8** the registry regex counts any uppercase token, so prose on the `Exit codes:` line
  registers as a rule. Fail-safe but a false-positive generator.
- **G9** `c7-fixes.md` claims every new `[[ ]]` gating control flow is an explicit `if`.
  `masked-path.sh:40,46` are `&& continue` / `|| continue`. The code is correct; the claim is
  not.
- **G10 SUSPECTED** `skillscore@2.0.2` is a correct and justified pin, since a major shipped
  under the old floating install. But CI still floats on action major tags and a floating node
  minor — the same mechanism G7-05 cites. A scope call for the manager.

## Observation, not a finding

Four early `test_f01.sh` runs showed a quality-JSON failure because `skillscore` returned a
stub `{"overallScore":80}` — the damaged install, since repaired. Identical at base and head,
so not a PR regression. But `test_f01.sh:131` cannot distinguish "skillscore scored it" from
"skillscore returned a stub."

## Confirmed by making things fail — no finding

- **The bash premise is correct.** 3.2.57: `[[ -f /nope ]]` under errexit reaches the next
  line; `[ -f /nope ]`, `grep -q`, `false` all exit 1. 5.3.15: all four exit 1.
- **The mutation evidence is exact.** Before, bash 3.2 prints PASS for a deleted file and bash
  5 aborts, both with zero FAIL lines and no summary. After, both give 4 FAIL lines,
  `23 passed, 4 failed`, exit 1, and `cmp` of the two full transcripts reports **IDENTICAL**.
- **27/27 assertions non-vacuous.** Every call site mutated independently; each produced its
  own labelled FAIL, a summary, and exit 1.
- **G7-02**: passes from `/`, `/var/tmp`, a scratch dir and `tests/`, on both bashes; base dies
  from `/`.
- **Census proven both directions.** A quoted new ID goes red across the relay transitively; a
  registered-but-unemitted ID also goes red; base stays green for both. Delta accounting from
  label sets: **0 removed, exactly 15 added, all attribution.** Emitted and registered sets are
  the same 10 IDs. The cycle guard terminates on an induced A-to-B relay.
- **Masked-PATH**: head absolutises, 0 broken links, payload returned, and a dangling entry in
  an earlier PATH dir no longer shadows a working one later.
- **Scratch dir**: measured against the Darwin user temp dir, the leak is gone at head on both
  bashes. The one remaining leftover is `draft-rewrite.sh`'s own file, correctly out of scope.
- **No regressions.** `audit-report.sh` has 0 non-comment changed lines with byte-identical
  non-comment text and identical runtime JSON, stderr and exit — comment-only confirmed. Label
  sets base to head: three suites IDENTICAL, `test_f01` 0 removed and 15 added. Nothing
  weakened. All four suites at head: 27/0, 19/0, 576/0, 243/0, identical on both bashes.
- **bash 3.2 safe.** No bash-5-only construct in any changed file; 10/10 shell files parse
  under 3.2.

## What blocks

**G1 alone.** The cluster's stated purpose is that the verification substrate could not report
a failure. That is still true of `tests/test_walk.sh`, on both supported bashes, at 7 call
sites, in a suite this PR's own CI runs. **G2 and G3 share its root and should close in the
same pass** so the class does not return. G4 through G10 are advisory and would not hold the
merge on their own.
