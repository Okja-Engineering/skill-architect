# S8b — Docs and scope amendment

- **Stage:** parked — partly gated on User Question 8
- **Owner:** unassigned
- **Branch:** none (target: `epic/skill-scope`)
- **Depends on:** **SC, S1b, S4** · **+ S3 iff S3 ships** (documentation follows the code; if SP1 answers no there is no attribution to document)
- **Blocks:** none

## Goal
Say what the plugin now actually does, with an evidence label on every claim, and amend the two files that currently promise it does not do it.

**Why this is a separate slice.** Round 1's single S8 bundled the docs with the CI guardrails and therefore inherited a dependency on S3. The fallback MVE (SP1 = no) withdraws S3, so that MVE did not close. The guardrails are now `S8a` with deps {SC, S0, S4}; this slice keeps the S3-shaped dependency, and keeps it **conditional** — it documents whatever has merged.

## Definition of done
1. `docs/cursor-hooks-spec.md` exists and carries an evidence label on **every** claim, drawn from `AGENTS.md:18-23` (`Specification` / `External evidence` / `Repository fact` / `Design decision` / `Local hypothesis`).
2. It records **R-RL-16 as a constraint handed off to F04**, not implemented here: the comparison report must express Skill Lift with a confidence interval, not a point estimate (**External evidence**, arXiv:2608.20614 — ACES mean 0.2134, 95% CI [0.1967, 0.2301] over 947 paired cases, positive-lift rate 72.8%).
3. It records the four token honesty classes and, for the hook estimate, the structural undercount — so a reader cannot lift a `chars/4` number into a cost claim.
4. `.out-of-scope.md` and `README.md` reflect live capture (R-SA-17). Both currently promise the plugin does *not* run live comparisons.
5. **It documents only what has merged.** If SP1 answered **no**, the doc states plainly that skill attribution is not available from hooks, cites the spike's recorded answer as the **Repository fact** that settles it, and points at T1 as the epic's remaining value. A doc that describes a slice that was withdrawn is the same failure class as an invented telemetry name.

## Test approach
Docs are reviewed for a label on every claim; the label vocabulary is `AGENTS.md:18-23`. Mechanically: a `grep` assertion in `tests/test_guardrails.sh` (owned by S8a — this slice adds one case to it, or ships its own two-line check if S8a has not merged) that every claim line in `docs/cursor-hooks-spec.md` carries one of the five label strings. Absence checks use `grep`, per `AGENTS.md:12`.

## Requirements
R-SA-11, R-SA-17, R-RL-16 (**documented and handed off to F04**, not implemented here)

## Files
`docs/cursor-hooks-spec.md` (new), `.out-of-scope.md`, `README.md`

## Next
**Partly gated on User Question 8** — `.out-of-scope.md` and `README.md` currently promise the plugin does *not* run live comparisons; amending them is the user's call, not the implementer's.

Ships last in whichever MVE runs: wave 4 in the primary MVE (after S3), wave 3 in the fallback (S3 withdrawn, deps reduce to {SC, S1b, S4}). It is the slice that makes the epic legible to someone who did not read this roadmap.
