# PR #10 gate — C1, C3, C4 · `fix/0.4.3-body-guard-exits` @ `0d0c2fb`

**Verdict: NOT CLEAN.** Four P2s block; three P3s share a root with two of them; one C4
entry is undelivered and recorded nowhere.

> Persisted by the chief of staff: the hunter's session had Write disabled.

## Coverage

14/14 diff files and **all 95 hunks** · 16/16 entries across the C1 (3), C3 (9) and C4 (4)
tables · **18/18 `SKILL.md` in the tree** put through the new primitive against **both** base
implementations it replaced, plus 9 hand-built edge cases · **28 absent-tool and 155
present-but-broken cases across all 6 scripts** (the repo's own matrix covers four) · 60
adverse conditions swept for exit-set conformance · 10 mutations driven red and green · 5
suites on bash 3.2.57 and 5.3.15. Two full sweeps; the second added F4, F5, F6.

---

## The shared root under F1, F2, F4, F5 and F6

**An external command invoked outside `require_tool`'s coverage leaks its own status as the
script's exit status under `set -e`, with no diagnostic and no verdict** — either because it
runs before the precondition is stated, or because its probe does not actually check the
answer, or because it was reasoned off the list on a premise covering only its exit-0
failures.

## F1 — BLOCKER — `draft-rewrite.sh:118`, newly introduced

`mktemp`'s status is unread, and the failure is reachable with the **real** mktemp:

```
$ TMPDIR=/path/that/does/not/exist draft-rewrite.sh -t <skill>
mktemp: mkstemp failed on …: No such file or directory
exit 1
```

Exit 1 is "usage or target error" in this script's own contract, so a broken temp directory is
reported as the caller's mistake, with no sentence naming mktemp and no "no draft was written".

**Newly introduced by this diff**: the bare `mktemp` at base ignores `TMPDIR` on BSD, so this
mode did not exist before. The change itself is right; only its status is unread.

## F2 — BLOCKER — `verdict-guard.sh:219`, newly introduced

`mktemp` is the **only** arm of `tool_answers` that does not compare against a known answer:

```bash
mktemp) [[ -n "$(mktemp -u 2>/dev/null)" ]] ;;
```

Every sibling compares to a literal. This one asks only that *there is* an answer, against the
function's own stated contract: "ask one question this file already knows the answer to, and
succeed only when the answer is right."

Measured: a `mktemp` printing garbage and exiting 0 **passes the probe**, and the drafter then
writes a draft at exit 0 with a **fabricated provenance** naming that garbage, and creates a
file under that name in the caller's working directory.

**Why the repo's matrix misses it**: it has exactly the mode that catches this, but its cross
product drives only four scripts. `mktemp` is required by the drafter alone, and
`check-quality.sh` is likewise outside the product.

## F3 — BLOCKER — `audit-report.sh` spec source, pre-existing but in scope for G3-04

The payload claim is `type == "object"` while the caller reads three fields one level in, and
the source's exit status is captured but never read against its payload.

| skill-validator answers | the report says |
|---|---|
| `{"foo": 1}`, exit 0 | failing verdict with **`spec_error: null`** |
| `{"passed":"yes","errors":0}`, exit 0 | **`summary.passed: true`** — the verdict reads true off a string |
| `{"passed":true,"errors":0}`, exit **1** | `passed: true`, status contradiction unread |

Row 1 is G3-04's defect verbatim. Row 2 is worse. The final conformance check on the report
tests that `summary.passed` is a boolean, which holds, so nothing catches it.

The head file states the invariant it breaks: every source is proven to carry the shape it is
read as, **as far down as the read goes**. For contrast, `check-frontmatter.sh` refuses
`{"foo":1}` correctly. The test matrix has the same blind spot: it drives the spec source with
non-objects only.

## F7 — BLOCKER — `README.md:271` and `:265`

The `jq` trade is **honest in code** and stated in the script headers. The user-facing surface
is wrong.

Base `check-frontmatter.sh` required only `skill-validator`; head requires five tools. Base
`check-quality.sh` required nothing; head requires two. But the readme enumerates the
jq-requiring scripts as three others and closes:

> Text mode needs no jq and is unaffected.

**Both newly-jq-requiring scripts are text-mode-only** — one has no JSON mode at all. That
sentence is now directly false and the enumeration is short by two. "The skills shell out to
three external tools" is likewise short: seven are now hard preconditions that exit 3.

**No cluster owns this row.** C5's related entry scopes to different lines, and the roadmap's
note tracks the release notes rather than this table.

## F8 — a C4 member was dropped silently — needs a manager disposition

`worklist.md:484` lists **G4-04** with the criterion that all four profiler help paths exit 0.
Measured at head: **all four exit 1.** The diff contains no Go file, the lane record's C4
section names only the other three, its deferral list does not mention it, and grep finds the
identifier nowhere outside the worklist.

The lane delivered 15 of 16 entries and the 16th vanished without a line.

> **Chief of staff ruling: DELIVER it in the fix round.** A help flag exiting nonzero is an
> exit-status contract defect, and this PR owns the exit-status cluster. Deferring it inside
> the release whose subject is exit contracts would be the wrong call. It is four exit codes.

## F4, F5, F6 — P3, same root, close with F1 and F2

- **F4** `audit-report.sh:90` — `date` unguarded. A nonzero `date` exits 2 or 127 with no
  report, outside the stated set. The diff adds a comment concluding `date` needs no guard;
  that premise is correct for the two exit-0 rows and **does not cover a nonzero exit**.
- **F5** the `script_dir=` block — a broken `dirname` exits 1 with zero output. Outside the
  contract for two scripts, and **new** for `check-quality.sh`, which had no such block at
  base. `dirname` cannot be routed through the guard since it runs first, but it can carry the
  same explicit refusal the guard load-check three lines below already uses.
- **F6** `draft-rewrite.sh:54-63` — `usage()` is a heredoc that runs on argument-parsing paths
  **before** `require_tool cat`. A broken `cat` makes `-h` exit 2 or 127. Part of why it went
  unseen: the exit-table derivation loops the audit scripts only, so the drafter's stated
  contract is compared against nothing anywhere.

---

## Confirmed clean, by execution

**Both P1s, rebuilt from scratch with the gate's own fixtures.** The structure defect went from
`passed: true` with zero findings to `passed: false` with eight. The path defect's two
fixtures, differing by one `---`, now behave identically. **Mutating the new primitive reddens
each P1's assertions and nothing else**: the old toggle fails exactly the three path
assertions; removing body stripping fails exactly the nine structure and report ones. The
column-0 heading shape is genuinely pinned — moving the headings into an indented block scalar
reddens the fixture's own precondition.

**The refactor is byte-equivalent.** Frontmatter: **18/18 identical** to both base
implementations. Body: 17/18, the single difference being the P1 fixture and the intended fix.
All nine edge cases match except the two documented decisions. CRLF is byte-identical before
and after, and the validator rejects such a file earlier anyway.

**The broken-utility class is closed.** 110 audit-script cases, **zero bad**: every one exits
3, never reports a pass, emits exactly one document carrying a dependency rule, and names the
broken tool rather than another component. The original defect is gone: a `grep` exiting 2 now
gives exit 3 naming grep with **zero** fabricated policy findings, where it produced nine
beside a null error. Both self-found defects are genuinely closed and the shapes do not recur:
every probe arm now invokes exactly one binary, proven behaviourally by breaking each and
confirming zero misattributions.

**The exit-contract change holds all three ways.** Exit 0 is preserved and **asserted by 54
assertions**. The derivation is load-bearing in both directions: altering the doc row or the
script header each turns it red. Stated sets match what can be emitted across 60 adverse
conditions; the only violations need a broken unguarded external, which is F4, F5 and F6.

**The declined prescription is CORRECT, for a stronger reason than the record gave.** The
cited history is real, traceable to one commit. And measured: the quality payload predicate
**accepts** a dependency-findings object, so obeying the criterion literally would have had
the composer treat it as a valid report with no score and null error, **re-opening the very
invariant the criterion defends, one level up.** The prescription implemented literally would
have been a defect. No real gap remains; the information is present and distinguishable as
prose, uniformly across three scripts, documented in each header, and pre-existing.

**Both "already discharged by C1" claims confirmed by bisection** to the primitive commit,
before any C3 commit. **Non-vacuity reproduced to the digit**: 586/12 then 608/0, with all
twelve labels verbatim.

Suites both bashes on the restored head: 46 / 35 / 20 / 941 / 272 = **1314 passed, 0 failed**.

## Notes, not findings

- **The C3 non-vacuity integers do not verify at head, by construction.** They were taken at
  each commit's point in the series, and the broken-tool cross product is driven off the
  required-tools list, so reverting a script changes the case count. Direction and magnitude
  hold; the integers are not head measurements.
- The private-scanner census would not catch a scanner written with a different regex spelling.
  Marginal.
- The exit-table derivation scopes to the audit scripts, so the drafter states a contract
  nothing compares against. That is why F1 and F6 went unseen.
