# Final convergence check — PR #8 @ `63dc8c8` · PR #10 @ `523b061`

> Persisted by the chief of staff: the reviewing agent had no Write tool, said so explicitly,
> verified the absence by `ls`, and returned its report inline without falling back to a heredoc.

## Verdicts

- **PR #8 — CLEAN to merge.** All three mandated items closed, each with a control proved to
  discriminate by reverting the mechanism. One P3 residual, in prose this round wrote.
- **PR #10 — the class is closed at 36 of 37 sites.** The one remaining is **P3, not a blocker**:
  it needs a shadowed or failing `rm` and does not corrupt output.
- **Merge order 9 → 10 → 8 → 7, unchanged.** The map gained a **fourth conflict**, introduced by
  PR #10's own fix round.
- **Concurrency incident: coherent.** No orphan, complete removal, linear history, nothing lost.

## Coverage

**PR #8** — 4/4 clauses of the new statement driven against code · 3 renames plus the new
admits-control with **all 4 admit rows mutated independently** · both remaining mechanisms
reverted in isolation · quadratic cost measured at three lengths and end-to-end at 400k · 6
suites x 2 shells (**983/0**) · Go checks · residual-claim sweep over every markdown file in the
tree, the PR body, and the suite's own prose.

**PR #10** — **all 7 shell files parsed for command-position words**, with quoted strings and
heredoc bodies blanked so program text cannot masquerade as a call · **35 real external call
sites plus 11 probe invocations** enumerated and classified · **288 break positions walked
independently of the suite** · both deletions compared byte-for-byte against the base over 8
fixture runs plus a boundary case · assertion labels diffed **label by label** (1615 base, 2307
head, all 37 removals enumerated) · meta-check and test library diffed against `main` · 4
discrimination mutations · 5 suites x 2 shells (**2307/0**) · Go checks.

**Cross-cutting** — 6/6 PR pairs trial-merged at these heads, plus the four-step sequence.

---

## PR #8 — all three mandated items closed

**The new statement is true on all four clauses**, each driven against the code.

The clause worth proving is that **the block is executed and can do anything a shell can**. The
reviewer added one line to the readme block with a backslash-escaped leading slash, ran it at
head, and the block wrote outside the scratch root **with the suite fully green**. That is
exactly what the prose now says, which is the point of retiring the old claim.

The destination bound is real: a shape-clean destination that resolves outside is refused with
both fences named, and a climb aimed at a decoy leaves its contents untouched — a run that did
not happen.

**The demotion's control discriminates on every row.** Widening the diagnostic four separate
ways, one row each, reddens the matching row every time and nothing is carried by another.
**Nothing anywhere claims the pass is complete**, swept across every markdown file, the PR body
and the suite text.

**The guard that had no control now has one**, and the length bound reddens when reverted.

### C7-3r — P3 — REAL — the durability clause is false

The suite claims its fence order keeps a pathological document from ending the run in a CI
timeout with no summary. **Only the destination is length-bounded.** The path-word pass is
quadratic over every word:

```
diagnostic alone, long non-destination word:  100k → 1s    400k → 14s
whole suite, same word in the readme block:   400k → 193s, and it returns 48 passed, 0 failed
```

About a megabyte on that line is roughly twenty minutes, which is the timeout-with-no-summary the
bound exists to refuse, **and it passes on the way there**. The file already knows the pass is
quadratic and bounded only the expansion.

---

## PR #10 — 36 of 37, with the one exception found by exhaustive enumeration

The complete external surface across all seven files was enumerated and each site classified,
then **288 break positions walked**, forwarding the first k−1 calls to the real binary and
failing from k. Every one refuses with the external named and leaves no draft behind.

### A5 — P3 — REAL — `rm`'s status is unread in the drafter's EXIT trap

`rm` is the last command of an AND-list, so errexit is **not** exempt from it. When it runs and
fails, the trap dies before its `return 0` and the script exits with `rm`'s status. Reproduced on
both shells with `rm` shadowed by one directory holding one real file, which is this project's
own stated threat model:

```
exit=7
stderr: nothing but the normal progress line
draft present: YES        temp audit left behind: yes
```

Three things wrong, all the original signature: the status is the tool's, nothing names the tool,
and **7 is outside the set the script's own header states** — so the exit-table witness added
this round cannot see it either.

**Shared root, already named honestly by the lane**: both the census and the break-mode walk
derive their tool denominator from the guard's registration sites. `rm` is in neither, so it is
simultaneously missing from the readme census and never driven by the walk. The lane stated that
boundary plainly; the **rule** is still asserted as implemented, and it is not.

**One line of this is new this round** — the line giving the trap ownership of the draft, which
was the partial-draft repair. **The fix round added one new member to the class it was closing.**

### The two deletions are clean

Both tools appear nowhere except the known-answers table and comments. Output is **byte-identical
to the base** across eight fixture runs plus a boundary case, exit codes included. The
replacement for one is strictly safer than what it replaced.

### The diagnostic names the right component

A probe-answering tool now yields a refusal naming **that tool**, across every consuming script,
including the non-section read the walk found. The old failure blaming a tool that answered
perfectly is gone. The sibling-script case names the tool at all 31 positions, and the child's
sentence now reaches the caller.

### Nothing was weakened to reach 2307

**Zero assertions deleted or loosened.** The arithmetic is exact: 2307 = 1615 − 37 + 729. All 37
removals enumerated: 35 are cases retired with the deleted tools, 2 are relabels into strictly
stronger forms, one of which was **proved** stronger — the old floor passed a shrink the new form
catches. **The meta-check refusing a vacuous assertion is unchanged** against `main`.

---

## M1 — P2, merge-resolution — a new fourth conflict, from PR #10

| pair | result |
|---|---|
| 9 x 10 | CONFLICT — the drafter |
| 8 x 9 | CONFLICT — the CI workflow |
| 8 x 7 | CONFLICT — readme and the profiler |
| 7 x 9, 7 x 10 | clean |
| **8 x 10** | **CONFLICT — the harness and the suite auditor — NEW** |

Isolated to PR #10's fix round: its exit-witness repair added a name to the same two lists PR #8
added its refusal verb to.

**Resolution invariant: the union — both names in both lists.** Dropping either is silent. Out of
the readiness check, a suite stops proving the harness loaded that primitive. Out of the auditor,
a suite's private copy of that name stops being detected. **It must not be resolved by picking a
side.**

## The concurrency incident — coherent

Nothing orphaned survives at head. What the mis-described commit's message *does* describe is
present and reddens when reverted. The function inventory differs by exactly the documented
renames plus the new control. **34 commits, zero merges; linear.** Both authoring worktrees clean
at the pushed heads, so nothing is stranded. The only residue is commit-message accuracy on two
pushed commits, disclosed in the lane record, **not worth a force-push on a shared branch**.
