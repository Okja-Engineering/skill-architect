# PR #10 confirming pass — C1, C3, C4 · `fix/0.4.3-body-guard-exits` @ `8a72f73`

**Verdict: NOT CLEAN.** The gate's **shared root is not fully closed.**

> Persisted by the chief of staff: the reviewing agent had no Write tool and returned its report
> inline. **Verified by CoS** notes are independent confirmation.

The lane wrote the correct rule down and applied it to the three instances the gate named. It did
not apply it to the class. **23 external call sites across 5 files still leave their status
unread**, and one is reachable with the real `cat` and leaves a half-written draft in the
caller's skill directory. Everything else the gate raised is genuinely closed, reproduced by
execution.

## Coverage

17/17 diff files and every hunk · **16/16 worklist entries** · **8/8 gate findings** (the brief
said six; the gate raised eight) · **every external invocation in all 7 shell files mechanically
enumerated — 31 call sites across 15 tool identities — each classified covered or not** · the
60-case broken-tool cross product **re-driven independently of the suite**, 60/60 clean · 8
probed tools x 3 break modes x 6 scripts · 5 selectively-answering stubs · 26 odd-argument paths ·
**16 mutations driven red and green** · all 5 non-vacuity revert rows re-measured to the digit ·
5 suites x 2 shells · 8 profiler exit paths plus 4 controls, both halves of the help fix reverted
in isolation. Two full sweeps.

---

## The shared root — NOT closed

Under `set -euo pipefail`, an external command whose status is not read at the call site leaks
its own status as the script's exit status, with no diagnostic and no verdict.

**The lane's stated rule is exactly right**: every external is either inside `require_tool` with
its status read **where it is called**, or it carries an explicit refusal. **The code implements
only the first conjunct as presence plus one probe**, and leaves the status unread at every call
site. A tool that answers that one question and fails another leaks straight through.

**The suite is structurally blind to this**: all three of its break modes fail the probe, so
`require_tool` catches them before any call site is reached.

### A1 — BLOCKER — P2 — `jq`'s status unread at 16 sites

Across `check-frontmatter.sh:115`, `audit-report.sh:195,278,279,303,317`, `check-paths.sh:167,170`,
and `check-structure.sh:194,195,207,210,211,212,241,244`. With a `jq` that forwards the probe and
the conformance call to the real binary and exits 5 otherwise: **exit 5, zero bytes on both
channels, no verdict.** That is the gate's own F4 signature verbatim.

A selectively-answering tool is **inside this project's own stated threat model** — one script
names "a build with a broken formatter, a wrapper on PATH under the same name", and the gate's
own earlier entry used a jq answering only one call.

### A2 — BLOCKER — P2 — `sed`'s status unread

`check-frontmatter.sh:128`, immediately before `exit 1`. With a sed that answers the probe and
fails otherwise, a genuine spec failure becomes **exit 5 with nothing on either channel**.
**The computed verdict is destroyed, not relabelled.**

### A3 — BLOCKER — P2 — reachable with the real `cat`, and it leaves a partial draft

Seven sites in `draft-rewrite.sh`. **No stub needed**: passing an existing but unreadable audit
report passes the file-exists test, `cat` fails, errexit fires, and a **six-line partial
`REWRITE-DRAFT.md` is left in the target**, with exit 1 — the status the contract reserves for
the caller's mistake — and a raw permission error on stderr.

It breaks the invariant the script states in its own comments: everything that can stop it must
stop it before the first byte is written, or "no draft was written" stops being true.

> **Verified by CoS** at the actual head with real tools: exit 1, raw `cat:` error, six-line
> partial draft left in the target. Confirmed REAL.

### A4 — P3 — the diagnostic names the wrong component

`json_document_conforms` reads **any** nonzero jq as "not that shape", so a jq that fails the
conformance call makes the diagnostic name **skill-validator** or **skillscore** when those
answered perfectly, and the word `jq` appears nowhere. That is the earlier entry's invariant
recurring one layer above where the lane closed it.

---

## P3 findings

- **B** — the readme's required-tools census is derived from `require_tool` sites only, so `date`
  and `dirname` are **absent** although both are now hard preconditions by the readme's own
  criterion — because the fix moved them outside the mechanism the census derives from. Short by two.
- **C** — the documentation says "the four house-policy checks"; there are **three**. An
  off-by-one in the exact section whose subject is documentation that drifted.
- **D** — the exit table pins document against header, **never header against behaviour**. Both
  sides moving together is uncaught: adding a status a script can never emit, or dropping one it
  emits on every failure, each leaves the suite fully green. **The original symptom was a stated
  code untrue of the script, so the mechanism installed to close it does not detect its own
  shape.** All five rows are true today, so this is durability, not a live wrong row.
- **E** — a comment says the adverse-script list is read off the tree; three lines later it is a
  literal. The behaviour is genuinely derived and could not be made to shrink, so this is
  accuracy — but it is the comment the next reader trusts instead of checking.
- **F** — a denominator guard asserts at least 45 broken cases while the real count is **60**, so
  a 25% shrink passes its own non-vacuity check.
- **G** — the lane's record asserts the audit skill's documentation was **not touched**. It edits
  20 lines. The edit is legitimate by CoS ruling; the **record** is what matters, because the
  final cluster will read that line as its handoff.

## Confirmed closed, by execution

**Both temp-file findings.** Nonexistent and unwritable temp directories and a garbage-answering
probe all give exit 3 with the tool named, **no draft, nothing created in the caller's directory**.
A probe that passes and a call that then fails is reported distinctly.

**The matrix derivation is real**, 45 to 60 cases, driven independently of the suite at 60/60
clean, and four mutations each redden it.

**The spec-source finding is closed and the lane's scope extension is correct and complete.** All
three gate rows re-measured with exit 0 preserved, plus a control proving non-vacuity. The
extension verified: one source, two readers, **one answer**.

**The named instances are closed** — broken `date`, `dirname` and `cat`-in-usage all refuse
correctly across all six scripts.

**The delivered help entry is non-vacuous** across **seven** paths with controls unmoved, proven
by reverting each half separately and disjointly.

> The reviewer disclosed that its **first measurement of these paths was wrong**, because zsh does
> not word-split, so a two-word argument arrived as one and read as an unknown command. It
> re-measured under bash and flagged it so the number would not be trusted from a bad harness.

**Both P1s remain closed**, with byte-identical payloads on the path-checker fixture pair and only
one frontmatter-delimiter reader left in the whole tree.

**Nothing was weakened to reach 1615.** The test library and the meta-check are **not in the
diff**. **Zero deleted assertions**, 168 added. The 15 deletions are loop scaffolding replaced by
something stronger.

**Every non-vacuity revert row reproduces to the digit** across five rows.

**The split decision is sound**: deleting the drafter's own header reddens **14** assertions, so
its contract is no longer compared against nothing, which was the actual defect.

**The final state is coherent** — the worktree is clean with zero diff against the pushed head, so
the discarded-edit incident left nothing behind.

## What blocks

A1, A2 and A3 — one root, 23 unread-status call sites across five files, of which A3 is reachable
with real tools and leaves a partial document. A4, B, C, D, E, F and G are P3 and share roots with
the above; G is record-only.
