# S8b — Docs and scope amendment

- **Stage:** parked — partly gated on User Question 8
- **Owner:** unassigned
- **Branch:** none (target: `epic/skill-scope`)
- **Depends on:** **SC-a, S1b, S4** · **+ S3 iff S3 ships** (documentation follows the code; if SP1 answers no there is no attribution to document)
- **Blocks:** none

## Goal
Say what the plugin now actually does, with an evidence label on every claim, and amend the two files that currently promise it does not do it.

**Why this is a separate slice.** Round 1's single S8 bundled the docs with the CI guardrails and therefore inherited a dependency on S3. The fallback MVE (SP1 = no) withdraws S3, so that MVE did not close. The guardrails are now `S8a` with deps {SC-a, S0, S4}; this slice keeps the S3-shaped dependency, and keeps it **conditional** — it documents whatever has merged.

## Definition of done
1. `docs/cursor-hooks-spec.md` exists and carries an evidence label on **every claim line**, drawn from `AGENTS.md:18-23` (`Specification` / `External evidence` / `Repository fact` / `Design decision` / `Local hypothesis`). **"Claim line" is defined here so the grep is writable (V3-#13):** a claim line is any line that is not blank, not a heading (`^#`), not a fence delimiter or a line inside a fenced block, not a table separator (`^\|[-: |]*\|$`), and not a bare list-continuation line. Every remaining line — including each table row — must contain one of the five label strings, or inherit one from a `**Label:**`-prefixed paragraph or table column it sits in, which the grep resolves by column, not by prose. State the definition in the doc's own header so a later reader can re-run the check.
2. It records **R-RL-16 as a constraint handed off to F04**, not implemented here: the comparison report must express Skill Lift with a confidence interval, not a point estimate (**External evidence**, arXiv:2608.20614 — ACES mean 0.2134, 95% CI [0.1967, 0.2301] over 947 paired cases, positive-lift rate 72.8%).
3. It records the honesty classes and, for the hook estimate, the structural undercount — so a reader cannot lift a `chars/4` number into a cost claim. It also records **D11's two caveats verbatim** (timing is hook-receipt wall-clock, not model time; `Status` is derived from `postToolUse`/`postToolUseFailure` and an unterminated call reads `aborted`) and **S8a rule 2's documented gap** (a mislabelled estimate built in a non-matching helper is not caught by the AST walk; naming convention is the mitigation).
4. `.out-of-scope.md` and `README.md` reflect live capture (R-SA-17). Both currently promise the plugin does *not* run live comparisons.
5. **It documents only what has merged.** If SP1 answered **no**, the doc states plainly that skill attribution is not available from hooks, cites the spike's recorded answer as the **Repository fact** that settles it, and points at T1 as the epic's remaining value. A doc that describes a slice that was withdrawn is the same failure class as an invented telemetry name.

## Test approach
Docs are reviewed for a label on every claim line; the label vocabulary is `AGENTS.md:18-23`. Mechanically: a `grep` case added to **`tests/test_guardrails.sh`** (created by S8a in wave 3; this slice appends one case in wave 4, one wave later, so there is no contention) implementing DoD 1's claim-line definition against `docs/cursor-hooks-spec.md`. **Named fallback (V3-#13):** if S8a has **not** merged when this slice is dispatched, this slice creates `tests/test_doc_labels.sh` — a standalone script with the same check and the same red→green seeded-violation pattern — and S8a's file is left alone. The fallback is a separate file, not an edit to a file that may not exist, so the choice is visible in the diff rather than resolved at merge time.

## Requirements
R-SA-11, R-SA-17, R-RL-16 (**documented and handed off to F04**, not implemented here)

## Files
`docs/cursor-hooks-spec.md` (new), `.out-of-scope.md`, `README.md`, **`tests/test_guardrails.sh`** (one appended case — S8a owns the file; **or** `tests/test_doc_labels.sh` (new) under the fallback above, never both)

## Next
**Partly gated on User Question 8** — `.out-of-scope.md` and `README.md` currently promise the plugin does *not* run live comparisons; amending them is the user's call, not the implementer's.

Ships last in whichever MVE runs: **wave 5** in the primary MVE (after S3, and one wave after S8a so the two never contend for `tests/test_guardrails.sh`), wave 3 in the fallback (S3 withdrawn, deps reduce to {SC-a, S1b, S4}, and S8a is not yet merged — which is exactly when the named fallback file applies). It is the slice that makes the epic legible to someone who did not read this roadmap.

**CI wiring (V4-#7).** Whichever file this slice ships — the appended case in `tests/test_guardrails.sh` or the standalone `tests/test_doc_labels.sh` — is discovered by the `tests/test_*.sh` glob **S1a DoD 7** installs in `.github/workflows/ci.yml`. This slice edits no workflow file, and the fallback path needs no extra wiring, which round 3 left unstated: under the pre-S1a workflow a new `test_doc_labels.sh` would never have run at all.
