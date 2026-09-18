# Additions routed into cluster C5 (Lane Z) by the chief of staff

C5 runs last and owns `skills/skill-audit/SKILL.md` prose. The PR #9 gate found three entries
that belong to that file and were **owned by nobody**. They do not block PR #9's diff; they
block the 0.4.3 release. **Whoever dispatches C5 must include these.**

## A1 — G8-02, returned from C8 as mis-filed, correctly

The original entry cites `skills/skill-rewrite/SKILL.md:46-47` for a Stage 2 preflight. Verified
at `71cc866`: line 46 is blank, line 47 is a heading, and no preflight exists anywhere in that
file.

**The real preflight is at `skills/skill-audit/SKILL.md:46-47`** — the same line numbers, which
is how the mis-file happened.

```
46  command -v skill-validator >/dev/null 2>&1 || { … exit 1; }
47  command -v skillscore      >/dev/null 2>&1 || { … exit 1; }
```

The defect: that preflight exits **1** for a missing tool, which is the condition PR #4
standardised on `DEP001` and exit **3**, and by the skill's own exit table 1 means a spec or
path failure. It also never checks `jq`, which PR #4 made unconditional — so a user who runs
the documented preflight, sees it pass, then runs the documented headline command gets exit 3
and empty stdout. The preflight exists to prevent exactly that.

**Beware**: `lane-d-fixes.md` originally cited `skill-audit/SKILL.md:51-52` for this. That is
wrong — those are the quality invocation and a closing fence. The C8 lane is correcting its own
record; use `:46-47`.

## A2 — G8-03's skill-audit half, returned from C8 as the wrong file

At `71cc866`, `verdict-guard.sh` appears in **neither** SKILL.md, though every audit script
hard-depends on it and exits 3 without it. C8 fixed skill-rewrite's side only (and its first
attempt at that sentence was itself false — see the PR #9 gate's B1). skill-audit's document
still omits the file entirely.

## A3 — G5-05's skill-audit half, which C8 half-landed out of order

The worklist records G5-05 as **one** entry covering both SKILL.md files, arch-verified
identical, assigned to C5 and sequenced last. The C8 lane edited skill-rewrite's copy.

Consequences to close here:

1. `skills/skill-audit/SKILL.md:5` still reads `compatibility: POSIX shell (bash 3.2+ or zsh),
   git.` **Two sibling skills with identically bash-only scripts now declare contradictory
   compatibility.**
2. The machinery built to hold the claim is hard-wired to one file: `tests/test_rewrite.sh`
   pins its own SKILL.md only, and all three compatibility checks read that one variable. The
   identical falsehood next door is un-held while the mechanism sits a few lines from being
   parameterised over both.
3. `git` is named in the compatibility line but no script in either skill runs it.

The C8 lane's *direction* was correct and is not being reopened: correct the claim rather than
make the scripts zsh-clean, because nothing declines zsh on purpose and the audit scripts only
appear to refuse because an unset variable collapses their script directory.

## Also for C5, from the PR #8 gate

The readme's "run the tests" block lists four suites and omits both the meta-check and the new
install suite, so a reader following the readme never runs either. Cluster C6 is fixing its own
half; confirm the final list is derived rather than hand-maintained, which is the shared root
the PR #8 gate named.
