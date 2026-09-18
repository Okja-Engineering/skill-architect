# PR #9 gate — cluster C8, skill-rewrite integration

Gated `fix/0.4.3-skill-rewrite` @ `483903a` against base `71cc866`.

**Verdict: NOT CLEAN.** Three diff blockers (B1, B2, B3) and three **release-level routing**
blockers (R1, R2, M-1) that do not block the diff.

> Persisted by the chief of staff: the hunter's session had Write disabled.
> **Verified by CoS** notes are independent confirmation.

## Coverage

6/6 diff files and 15/15 hunks walked line by line · **12/12 C8 entries** including both
returns and the deliberately-open half · **70/70 runtime assertions individually classified**
(62 source call sites plus 8 from a flag loop) · **23 mutations executed** · 12 suite runs
(6 suites x two bash versions) · 4 drafter runs against clean, minimal, faulty and empty
targets plus 2 missing-target paths · **4 interpreters** (bash 3.2, bash 5.3, zsh, sh) ·
3 masked tools via the repo's own helper, read-only.

Counts reproduce on both bashes: 50 / 27 / 20 / 70 / 575 / 243 = **985 per bash, 1970 total,
zero failures.**

---

## B3 — BLOCKER — three headline assertions pass over a document that no longer contains the invocation

`tests/test_rewrite.sh:132-136`. `doc_block` anchors on a heading regex and prints only when
it finds both the heading and a fence. When either is absent it prints **nothing**, so
`run_doc_block` writes a script containing only `set -eu`, `bash` exits 0, and the suite
reports **"the documented invocation ran"**.

| Mutation | Document change | Suite result |
|---|---|---|
| M18 | rename `### Generate a rewrite draft` heading | **70 passed, 0 failed** |
| M19 | rename `### Stage 1: Run audit if needed` | **70 passed, 0 failed** |
| M22 | delete the Stage 1 fence, keep the heading | **70 passed, 0 failed** |
| M23 | delete the whole section | **70 passed, 0 failed** |
| M20 | break a command *inside* the block | 2 FAIL — machinery works when found |
| M21 | rename the Stage 2 heading | 1 FAIL — only from a backstop |

Affected: three assertions. Stage 2 is partly backstopped; Stage 1 and the worked example are
not, which is why two mutations are fully green.

**This is the assertion-that-cannot-fail class PR #6 exists to refuse, landing on G8-01 — the
entry the cluster is named for.** The existing control covers *a block whose command fails*,
not *no block found*. The meta-check cannot see it by its own documented boundary. The author
got this right in the sibling reader, which is guarded, so the root is local to this helper.

**Invariant**: an assertion that a document's invocation runs must fail when the document no
longer carries that invocation. A rendered-empty block is a missing document, not a passing one.

> **Verified by CoS.** The awk sets `seen` only on a heading match and prints only inside a
> fence, and the runner prepends `set -eu` to whatever it returns. Confirmed REAL.

## B2 — BLOCKER — the new Prerequisites paragraph describes guarded behaviour on the one path that is not guarded

`skills/skill-rewrite/SKILL.md:40` says Stage 1 **and `draft-rewrite.sh`** both exit 3 with
the tool named on stderr "rather than reporting a verdict it could not compute".

True of the check script run directly. **Not true of `draft-rewrite.sh`** — the command this
paragraph's readers run. Reproduced at head with the validator masked:

```
exit=0
stdout: Rewrite draft written to: .../REWRITE-DRAFT.md
## Current state
ERROR: required tool not found: skill-validator
```

Exit 0, draft written, diagnostic presented as the skill's own `Current state`. That is the
half correctly left open for the concurrent guard lane — **and this new sentence tells the
reader the opposite of what happens**, which is the one thing the mandate asked to rule out.
The release now contradicts itself: the readme states the honest behaviour for the adjacent
failure mode. Nothing in the 70 assertions constrains it.

## B1 — BLOCKER — a newly written sentence is false

`skills/skill-rewrite/SKILL.md:42` claims `verdict-guard.sh` is a file "**which no other file
references by name**". It is one of the most-named files in the repo.

The covering assertion only greps that the filename appears somewhere in the document, so it
passes over the falsehood. The warning's load-bearing point survives without the clause; it is
the **scope** that is wrong.

> **Verified by CoS.** `git grep -l` finds it named in **12 files**, including four check
> scripts, the readme, two test suites and the drafter. Confirmed REAL.

---

## Release-level routing blockers — three entries are owned by nobody

**These do not block this diff. They block 0.4.3.** All three belong to
`skills/skill-audit/SKILL.md`, which cluster **C5 (Lane Z)** owns and which has not been
dispatched.

### R1 — G8-02's return is correct; the record's replacement citation is wrong

Verified at base: `skill-rewrite/SKILL.md:46` is blank, `:47` a heading, and no preflight
exists anywhere in that file. The real preflight is at **`skill-audit/SKILL.md:46-47`** — the
*same line numbers*, which is how the mis-file happened.

**The lane record cites `skill-audit/SKILL.md:51-52`. That is wrong** — those are the quality
invocation and a closing fence. A lane sent there patches the wrong lines.

The return is substantively right. With no receiving lane it drops two P2 doc defects.

### R2 — G8-03's skill-audit half is correctly returned and also unowned

At base, `verdict-guard` appears in **neither** SKILL.md. At head only skill-rewrite's is
fixed, and see B1.

### M-1 — G5-05 is one Lane Z entry; half landed here, leaving its twin false and un-held

The worklist records it as **one** entry covering both SKILL.md files, arch-verified identical,
assigned to C5 and **sequenced last**. PR #9 is Lane E and edited one of the two.

`skill-audit/SKILL.md:5` still declares POSIX shell and zsh compatibility. **Two sibling
skills with identically bash-only scripts now declare contradictory compatibility.** The
machinery built to hold the claim is hard-wired to one file, sitting six lines from being
parameterised over both.

The worklist's own note is the risk that materialised: earlier clusters feed C5, so check
before editing or the prose gets written twice and is wrong in between.

**The fixer's decision is sound and is not being asked back.** What is left over is the
unowned twin and the single-file assertion.

---

## M-2 — label/assertion mismatch, 6 of 70, one root

The label states the claim the section is *about* rather than the check the line performs. No
capability gap, since a sibling assertion carries the real check in each case, but the labels
are what a CI reader trusts.

- Four flag assertions claim a value-less flag exits 1 "rather than aborting on an unbound
  variable" — but an unbound-variable abort under `set -u` also exits 1, so the assertion
  cannot distinguish the two states it names. Proven: reverting the fix leaves all four green
  while eight companions go red.
- One claims the path check refuses an unresolvable bare path, but tests a fixture premise.
- One claims the block runner reports an absent script, but tests the same.

## Low

- The drafter's header documents a helper as yielding the option's value; it yields none and
  each caller reads the argument itself. Same class as an entry this PR fixes elsewhere.
- "Every command below is anchored on it" is false for the Stage 1 block, and one input has
  two definitions.
- Two template-probe claims describe exact heading matches where the code prefix-matches. The
  PR's own new fixture exploits it.
- One list says three sections where there are four.
- "Verbatim" understates a stream merge that belongs with B2's root.
- A passing control prints its child's error to stderr, against the harness's own rule.

## Verified as claimed — do not re-litigate

**The enforcement claim has teeth, both directions.** Four mutations: an undocumented section
emitted, a phantom section listed, a simulated 0.5.0 capability landing (**4 FAIL**), and
simulated frontmatter preservation. The old description cannot ship again.

**The exit-3 claim is real**, proven both directions. **The PR #6 meta-check inspects this
suite and is not weakened** — injecting a constant assertion reddens it by name and line; its
blind spot for B3 is the behavioural one it documents, not a weakening. **The glob genuinely
picked the suite up** (base 46, head 50; the record's "45 to 50" is off by one).

**The zsh mechanism is exactly as described and the corrected claim is true.** Nothing declines
zsh on purpose; the audit scripts only appear to refuse because an unset variable collapses
their script directory. The drafter runs to a draft under bash 3.2, bash 5.3 and `sh`, with
byte-identical drafts, and no bash-5-only construct is present.

**The C3 half is genuinely still open** and nothing in the 70 assertions constrains it. **The
rebase notes are accurate** but for one off-by-one: the trap is one line above the first child
call, not two.

**Documentation against reality, run by run** across four targets plus two missing-target
paths: the emitted set matches the inventory exactly in every case, the tail is md5-identical
across all four, the evaluation matrix has exactly the claimed number of dimensions, and the
exit contract is complete over every literal exit.

## What blocks

**B1, B2, B3** on the diff. **R1, R2, M-1** on the release — three entries belonging to
skill-audit's document, owned by nobody, plus a wrong citation in the lane record that would
send a fixer to the wrong two lines.
