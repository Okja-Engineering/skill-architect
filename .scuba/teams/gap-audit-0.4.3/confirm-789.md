# Confirming pass — PRs #7, #8, #9 fix rounds, plus the four-way integration map

> Persisted by the chief of staff: the hunter's session had Write disabled.
> **Verified by CoS** notes are independent confirmation.

## Verdict

| PR | head | verdict | blocking |
|---|---|---|---|
| **#7** | `d787977` | **CLEAN** | one LOW prose residue, non-blocking |
| **#8** | `f7cacbe` | **NOT CLEAN** | **F-8a (P1)** — B1's class is not closed by construction |
| **#9** | `3df3207` | **CLEAN** | — |

**Merge order: 9 → 10 → 8 → 7.** The order holds; **the fix rounds did not change the
conflict map.**

## Coverage

**PR #7** — 2/2 blockers; 2/2 F1 mutations re-run, each reddening its own assertion and
leaving the other's green; 3/3 F2 pin neuterings; **55/55 fixtures re-compared three ways by
execution** against a base binary; the unnamed-scope count re-measured at 3 revisions under 4
denominators; 11/11 diff files and 10/10 dispositions walked.
**PR #8** — 5/5 blockers; **6 escape spellings attempted** (3 escaped, 3 refused, all proven by
decoy state); 3/3 command forms re-reverted plus 2 standalone staging measurements; 3/3
vacuity cases; 5/5 vendor prose claims executed or fetched; 17/17 diff files; 952/0 on both
shells.
**PR #9** — 3/3 blockers (4 defect halves); **6 document mutations x 2 shells**; the reader's
status codes exercised directly; 3/3 assertion directions; 94/0 on both shells.
**Integration** — 6/6 pairs by trial merge; **one full four-way sequence merged, resolved and
rebuilt**; 3/3 conflicts characterised by resolution; 2/2 named conditions measured; 7/7
suites on the integrated tree.

---

## F-8a — BLOCKER — P1 — `tests/test_install.sh:99-133`

**B1's verb is fixed; its class is not.**

The refusal mechanism is sound and was proven: the new `require` halts through the summary,
exits 1, is listed in the harness readiness check, and the assertion auditor refuses a vacuous
one. The substitution is gone and the block runs verbatim under a stripped environment.

**What does not hold is the containment argument.** The guard enumerates exactly two escapes,
an absolute path and a tilde with a user name, and claims:

> "This is the whole set of escapes from the containment above, which is what makes it a proof
> rather than a list of the spellings that happened to be wrong once."

**Parent-directory traversal is absent**, and the guard inspects only token *shape*, so
anything beginning `..`, `$HOME/..` or `$(…)/..` passes. Proven at head, suite unmodified,
only the readme's destination respelled:

| respelling | the five containment requires | where the block wrote |
|---|---|---|
| `$HOME/../../../../CONFIRM-ESCAPE-A/.claude/skills` | **all PASS** | outside the scratch root |
| `../../../../CONFIRM-ESCAPE-B/.claude/skills` | **all PASS** | outside the scratch root |
| 15 x `../` then an absolute tail to a decoy home | **all PASS** | **deleted the decoy's files and installed over them** |

That is arbitrary filesystem reach from a readme edit. With the operator's real home
substituted it reaches a live skills directory — the outcome B1 was raised for, by a spelling
the guard does not know. The control confirms the two covered spellings **do** halt.

**Invariant**: no path the extracted block writes to may *resolve* outside the harness scratch
root, for any spelling — decided after resolution, not from the block's text. **A denylist over
spellings cannot be a containment proof; only the resolved write can.**

Two more expressions of the same root, so one repair closes all three:

- `:179-181` — the assertion named for what the block can reach evaluates a relation that is
  **true by construction** and never looks at the block. The require carrying the load-bearing
  name decides nothing.
- `:119` — scanning stops at a line's first `#` token, so anything after a `#` is never
  inspected.

> **Verified by CoS.** The awk tests only `^/` and `^~[^/]`. Nothing examines `..`. Confirmed
> REAL.

## F-8b — P3 — `tests/test_harness.sh:138-139`

The derived count is correct and both live directions are proven. But the precondition written
to close the empty-workflow case is **unreachable**: the count is computed earlier, the pipe
matches nothing, `pipefail` propagates status 1 out of the command substitution, and `set -e`
kills the shell first. Observed output is a single abort-guard line.

The invariant fails closed, but by the abort guard rather than the stated mechanism, and the
diagnostic naming the cause is lost. **Invariant**: a denominator of zero must be reported as a
refused precondition, not surface as an unexplained abort.

---

## PR #7 — CLEAN

Both F1 mutations reddened on their own assertion and left the other's green. All three F2 pin
neuterings reddened, one of them plus roughly forty pre-existing subtests. **The 55 fixtures are
result-identical three ways**, comparing stdout, stderr **and** exit code.

**The unnamed-scope count is settled, and my gate's figure was wrong.** Measured at three
revisions under four denominators: **36 across 25 files** over content the reader reaches,
identical at all three. 38 is reachable only by counting entries in content never parsed, and
there the file count is 27, not 26. **No denominator yields the gate's 38/26.** The substance
reproduces exactly: a strict allowlist fails 53 tests and subtests, the gate's own number.

The one residue, LOW and non-blocking: the fixture readme's bolded claim excepts "the tail of"
one file, where the records lacking the id are that file's whole set. No behavioural
consequence, since the file resolves to an error and nothing in it is reached, and the wording
is inherited rather than introduced. One word.

## PR #9 — CLEAN

**Both halves of B3 are closed.** The reader now exits distinctly for no-heading, heading
without fence, and empty fence, and the bounded half stops at the next heading, so a block an
assertion names must come from the section it names. One reader replaced three hand-written
copies.

Four document mutations, each on both shells, restored and hash-verified after each: 93/1,
93/1, 92/2, 91/3, and 94/0 as written. The bounded half proven directly — with the fence
deleted the reader returns a distinct status and no output, where the old reader returned the
**next section's** block.

A methodology caution worth keeping: renaming a heading to a longer string that still matches
the assertion's prefix regex legitimately leaves the suite green. That is the regex being a
prefix, not a gap.

**B2's paragraph is true of the command a reader runs**, verified with the validator masked,
and non-vacuous both ways. The assertion reads the document's stated status and compares it to
reality, so it is pinned to "the document says what the command does" rather than to today's
wording — which is what makes it go red when PR #10 lands. Correct shape.

---

## Integration — re-tested at the current heads

| pair | result |
|---|---|
| 7 + 8 | **CONFLICT** — `README.md`, `profiler/claude_code.go` |
| 7 + 9 | clean |
| 7 + 10 | clean |
| 8 + 9 | **CONFLICT** — `.github/workflows/ci.yml` |
| 8 + 10 | clean |
| 9 + 10 | **CONFLICT** — `skills/skill-rewrite/scripts/draft-rewrite.sh` |

**Exactly the three conflicts already known, no more, no fewer.**

- **8+9, the workflow file: comment only.** Both rewrote the same suite-count comment; the
  steps merge cleanly and the merged file keeps **all seven** suite lines. Take PR #8's
  comment, which is the superset.
- **7+8, the readme: additive union.** Both expand the same paragraph in different directions.
  Nothing contradicts.
- **7+8, the profiler: semantic, and the naive union does not compile — confirmed**, giving two
  `Probe()` definitions and a zero-argument call. **The resolution is one line**, verified end
  to end on the integrated tree with build, vet and tests green, and both PRs' new test files
  passing together.
- **9+10, the drafter: semantic and the expensive one.** Three conflicting regions and
  **neither side can be taken whole** — a side-pick loses 14 of PR #9's own repairs and leaves
  19 failures. **Five of those failures are by design** and require the document to move in the
  same commit as the script. The lane record names three; **the tool declaration and the
  sibling-checks census are two more it does not name.** That coupling is exactly what PR #9's
  present-tense assertions were built to force, and they do force it.

**Both named conditions measured.** Rebasing PR #8 onto PR #9 reddens exactly one assertion,
naming the missing readme line; adding it gives green. And after all four land the derived
suite count equals the workflow's, with seven files and seven steps, so the workflow must keep
every step.

**Order rationale.** 9 before 8, because otherwise PR #9's branch stays green while post-merge
main goes red until someone notices; with 9 first, PR #8's rebase surfaces it loudly and by
name. 10 after 9, because PR #9's present-tense assertions exist to go red when the guard
lands, and landing 10 first inverts that and loses the coupling. 7 last, because it conflicts
only with 8 and the two resolutions are paid once wherever the pair meets.
