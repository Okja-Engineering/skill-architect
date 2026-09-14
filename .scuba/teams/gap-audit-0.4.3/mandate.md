# Gap audit of delivered state — toward 0.4.3

## User directive (2026-09-14)

> "we are incrementing towards 1.0 and 0.5 represents additional features — if we have
> gaps in anything that is currently delivered we should exercise patience and
> addressed those now"

0.5.0 is **additional features**. Before those features start, close the gaps in what
is **already delivered**. This audit finds them.

## Scope: the delivered product only

**Audit `origin/main` at `5c847e1`** — v0.4.1 plus both 0.4.2 squashes. That is the
delivered state.

Explicitly **out of scope**:

- Local `main` (`0a83615`) and its 7 unpushed commits. Unlanded work is not delivered.
- The 21 modified tracked files and all untracked files in the primary working tree.
- `skillgate/` and `skills/skill-gate/`. The user confirmed these stay out of 0.5.0;
  they are 0.6.0 material and are not delivered.
- The in-flight v0.4.2 version-bump PR. Version surfaces are its job, not a gap.

A gap is a defect in what a user who installs the released product today would get.

## Ground rules

- Work in your **own worktree cut from `origin/main`**. Never touch the primary
  working tree: it holds uncommitted 0.5.0 work. No `checkout`, `reset`, `clean`,
  `stash`, or `rebase` there.
- **Read and run only.** Hunters never fix. Report findings; do not edit product code.
- Prove findings by **running** the code and its tests, not by reasoning about them.
- Cite `file:line`. Classify every finding REAL or SUSPECTED.
- Return a **coverage line**: the denominator you enumerated and walked. A findings
  list with no denominator is an early stop and will be rejected.
- Enumerate the **whole class**, not a few salient hits, so any fix can be holistic.
- Note that local `bash` is 3.2 on macOS while CI runs bash 5 on Linux. A finding that
  reproduces only on one is still real; say which.
- Write your report to `.scuba/teams/gap-audit-0.4.3/<lens>.md` by absolute path.

## Severity, for triage into a release

- **P1** — the delivered product produces a wrong answer, or a documented flow fails.
- **P2** — a documented claim is false, or a real gap with a workaround.
- **P3** — cosmetic, stale prose, or a gap with no user-visible effect.

Say plainly which findings belong in a 0.4.3 patch and which are honest 0.5.0
deferrals. Patch means a fix to delivered behavior with no new capability and no
schema change.

## Lenses

1. **docs-truth** — every claim in `README.md`, `RELEASE_NOTES.md`, `CHANGELOG.md`,
   `docs/profiler-spec.md` against actual behavior on main.
2. **deferred-ledger** — every item 0.4.1 and 0.4.2 explicitly deferred, still open?
3. **install-truth** — the documented install and first-run path, actually executed.
4. **adapter-honesty** — each adapter's capability report vs what Capture produces.
5. **skills-function** — skill-audit and skill-rewrite vs their own SKILL.md.

## Definition of done

One deduped, classified, severity-ranked worklist that a `bug-fixer` could act on
without re-deriving anything, separating 0.4.3 patch items from 0.5.0 deferrals.
