# Mandate — skill-audit script guards (ships in 0.4.2)

_Chief of staff, 2026-09-14. Sibling of `teams/release-0.4.2/mandate.md`, running in parallel. Disjoint files, so no conflict: that PR touches `profiler/`, this one touches `skills/skill-audit/scripts/`._

## Goal

A skill-audit script must never report a passing result it did not verify. Two guards, both found during the 0.4.1 review and both reproduced.

## Commit rules (override any default guidance)

- **No `Co-Authored-By` trailer** for Claude or any AI agent, on any commit.
- **Never** pass `-c user.email=`, `--author=`, `GIT_AUTHOR_EMAIL` or `GIT_COMMITTER_EMAIL`. Let git use the configured identity, `imagineux <imagineux@gmail.com>`.
- Before opening the PR: `git log origin/main..HEAD --format='%an <%ae>' | sort | uniq -c` shows exactly one identity, and a grep for `Co-Authored-By` returns 0.

## Base

Branch from `origin/main` at `2ed34b8` (v0.4.1). **Not** from local `main`, which carries 7 unpushed commits and an uncommitted 0.5.0 prototype. Work in your own worktree; never touch the primary tree except `.scuba/`.

## The defects

### S1 — `check-structure.sh --json` reports a failing skill as a clean pass when `jq` is absent
`skills/skill-audit/scripts/check-structure.sh:71` pipes its child's path findings through `jq` with `2>/dev/null || true`, so a missing `jq` is swallowed and the findings vanish. The zero-findings branch at `:92-93` then emits `{"findings": [], "passed": true}` and exits 0. Reproduced on a fixture whose only faults are path findings:

```
jq present:  {"findings":[{"level":"fail","rule":"PT001",...}],"passed":false}  exit 1
jq masked :  {"findings": [], "passed": true}                                  exit 0
```

The child's 127 is lost too: `:68` captures it into `path_code` and `:86` only maps `path_code -eq 1` onto `fail`. Note the shape of the bug precisely — a policy finding (PL002 and friends) never round-trips through `jq`, so a skill with any policy fault correctly dies at `:102` with 127. Only a skill whose faults are *exclusively* path findings is silently passed. That is why the repo's own `tests/fixtures/f01/missing-script-ref` does **not** demonstrate it; it also raises PL002. Build a fixture whose only fault is a path fault.

### S2 — `check-frontmatter.sh` prints `frontmatter OK` and exits 0 when `skill-validator` is missing
`skills/skill-audit/scripts/check-frontmatter.sh:24`: a missing binary makes the command substitution exit **127**, and the script special-cases only 1 and 3, so 127 falls through the spec-failure branches to the license gate. On a licensed skill it prints `frontmatter OK` and exits 0 having validated nothing. On a license-less skill it exits 2 from the PL001 gate, which is why a casual check looks fine. The SKILL.md Stage 2 guard protects the documented workflow; running the script directly bypasses it.

## Invariants

1. A script exits non-zero and says which tool is missing when a tool it needs to reach its verdict is absent. It never emits a passing verdict it could not compute.
2. A missing tool is distinguishable from a clean result, in both the exit status and the `--json` payload.
3. The 0.3.1 "fail loudly" promise is kept: do not soften the existing `exit 1` dependency guards in `SKILL.md`.
4. Existing behaviour with the tools present is unchanged — every current assertion in the four shell suites still passes.

## Scope

`skills/skill-audit/scripts/check-structure.sh`, `skills/skill-audit/scripts/check-paths.sh` (same swallow pattern; verify whether it has the same hole and fix it if so), `skills/skill-audit/scripts/check-frontmatter.sh`, and `tests/` for the new cases. Update the README prerequisites row for `jq` if the behaviour it describes changes.

**Do NOT touch any version surface** — no plugin manifest, no `marketplace.json`, no `AdapterVersion`, no `tests/test_skill.sh` version asserts, no CHANGELOG or RELEASE_NOTES version heading. A separate release commit bumps them once after both 0.4.2 PRs merge.

Out of scope: anything under `profiler/` (the sibling PR owns it), `skillgate/`, `docs/research/`.

## Definition of done

Both reproductions above return a loud failure with the tool masked, and the unchanged result with it present. New shell-suite cases cover both, proven RED against the current scripts first. All four suites green. `profiler` untouched.

## Method

Test-first per defect: write the failing case, watch it fail for the right reason, fix, watch it pass, revert in a scratch copy to confirm RED again. Guard at the point the tool is needed rather than adding a condition at each call site, if one guard covers both scripts honestly.
