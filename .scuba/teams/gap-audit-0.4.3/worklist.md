# Gap audit 0.4.3 — reconciled worklist

Reconciles `adapter-honesty.md`, `skills-function.md`, `deferred-ledger.md`,
`install-truth.md`, `docs-truth.md` into one deduped, root-clustered, executable list.

**Baseline audited:** `origin/main` @ `5c847e1`. **PR #5** = `origin/release/0.4.2-version`
@ `13b3141`, base `5c847e1`.

**Method.** Every P1 re-verified by the architect against `5c847e1` before entry — four by
running the code against purpose-built fixtures extracted with `git archive 5c847e1` into a
throwaway directory, two by reading the blob directly. The primary working tree was never
touched; no product file was edited. Where an entry says **[arch-verified]**, the architect
reproduced it independently. Where it says **[lens]**, the finding is carried on the
hunter's evidence and is unverified here — a fixer should re-run the lens's reproduction
before assuming it.

---

## 1. Headline numbers

| | |
|---|---|
| Findings enumerated from the five reports | **104** (adapter-honesty 10 · docs-truth 10 · skills-function 26 · install-truth 25 · deferred-ledger 33) |
| Collapsed by cross-lens dedup | **23** (18 merge groups) |
| Distinct defects | **81** |
| Disposed — closed, invalid, out-of-product, or deliberate | **10** |
| Discharged by PR #5 | **3** |
| **Actionable lens-findings** | **68** |
| **Worklist entries they collapse into** | **56** |
| Entries carrying 0.4.3 work | **49** (48 whole + `G9-01` split) |
| Entries carrying 0.5.0 work | **10** (6 whole + 4 split halves) |
| Root clusters | **9** |
| **P1 after re-verification** | **4** (6 claimed: 1 demoted, 1 discharged by PR #5) |

The last two rows overlap by three: `G9-01`, `G8-05` and `G5-03`/`G7-03` each carry a
0.4.3 half and a 0.5.0 half, itemised in §6. 68 actionable findings collapse to 56 entries
because eight late-cluster entries each carry more than one honestly-held deferral
(`G9-07` carries six, `G9-08` two) — those are tracked as one unit because they share one
0.5.0 design and must not be taken separately.

The manager's brief said "roughly 119". My denominator is 104 numbered findings. The
difference is sub-items counted separately in the briefing: `deferred-ledger` L19/L21/L22/
L23/L24/L25 are presented as one bullet list of six, and `skills-function`'s Roots C/D/E/F
carry lettered sub-items that can be read as findings or as one root each. Nothing was
dropped; the accounting is per numbered ID.

---

## 2. The P1 list, as verified

Four P1s enter 0.4.3. Two claimed P1s do not, for stated reasons.

### Confirmed — all four reproduced by the architect against `5c847e1`

| ID | Defect | Verification |
|---|---|---|
| **G1-01** | `check-structure.sh` never strips frontmatter, so the headline `audit-report.sh` returns `summary.passed: true` for a skill with an empty body | **[arch-verified]** reproduced end to end (below) |
| **G1-02** | `check-paths.sh`'s frontmatter toggle re-enters on a third `---`, so a markdown horizontal rule silently ends all path checking | **[arch-verified]** 3 findings → 1 across a rule/no-rule pair |
| **G2-01** | `Capture` ignores its `sessionID` argument; every number in a session-stamped profile is drawn from every session in the export | **[arch-verified]** `sessionID` occurs exactly twice in `claude_code.go`; `resolve()` is argumentless |
| **G7-01** | `tests/test_skill.sh`: 16 assertions pass a literal and 11 bare `[[ ]]` guards are inert under macOS bash 3.2 | **[arch-verified]** mutation survived with a printed PASS |

**G1-01 reproduction.** A `SKILL.md` whose six required headings are column-0 YAML
comments inside its own frontmatter (valid YAML — `#` at column 0 is a comment), whose code
fence sits in a block scalar (`PL004`'s regex is `^[[:space:]]*` and tolerates the indent),
and whose list item is the `---` delimiter itself (`PL005`'s regex `^[[:space:]]*[-*]`
matches `---`). Body: one prose sentence, no headings, no fence, no list.

```
$ skill-validator validate structure -o json fx/fmbleed2   -> {"passed":true,"errors":0}
$ check-structure.sh fx/fmbleed2                           -> exit=0, no findings
$ audit-report.sh fx/fmbleed2 | jq -c .summary
{"passed":true,"spec_passed":true,"spec_errors":0,"spec_warnings":1,
 "quality_score":73,"quality_grade":"C","policy_failures":0,
 "path_failures":0,"total_findings":0}
```

Healthy `jq`, healthy `skill-validator`, healthy `skillscore`, realistic input. The headline
command returns a clean bill of health for a skill that fails three of its own house rules.
This is the single highest-harm item in the audit.

Note one correction to the `skills-function` A2 write-up: headings indented inside a block
scalar do **not** bleed, because `PL002`'s regex is `^#{2,6}[[:space:]]+` with no whitespace
tolerance. They must be column-0 YAML comments. The finding stands; the mechanism is
narrower than stated and a fixer building a regression fixture needs the exact shape above.

**G7-01 reproduction.** On a scratch copy of `13b3141`, `skills/skill-audit/references/
best-practices.md` deleted — the file `tests/test_skill.sh:87-88` exists to guard:

```
PASS: skill-audit best-practices reference exists      <- the file does not exist
```

The mechanism is a bash 3.2 defect I characterised precisely, and more narrowly than the
lens did: **`[[ ]]` is exempt from `errexit` on bash 3.2; `[ ]` is not.**

```
$ /bin/bash -c 'set -euo pipefail; [[ -f /nope ]]; echo REACHED'   -> REACHED, exit 0
$ /bin/bash -c 'set -euo pipefail; [  -f /nope ]; echo REACHED'    -> exit 1
$ /bin/bash -c 'set -euo pipefail; grep -q zzz /etc/hosts; echo R' -> exit 1
```

So on macOS all 11 `[[ ]]` guards are silent no-ops *and* their paired assertion prints
PASS. On CI (bash 5) they abort — with no FAIL line and no summary. The suite's non-`[[ ]]`
enforcement (the python heredocs, `grep -q` at `:57`, the script invocations) **does** fire
on 3.2. That distinction is load-bearing for the PR #5 answer in §7.

### G3-01 — demoted from P1 to P2

`check-frontmatter.sh:44-63` reads only `$?` and never parses the `-o json` payload it
requested. **[arch-verified]** — line 44 captures `output`, the `case` at `:49-63` switches
on `$code` alone, and `output` is used only to `sed` into a failure message. Exit 0 or 2
falls through to the license gate and prints `frontmatter OK`. REAL, not in dispute.

Rated **P1** by install-truth (IT-01) and **P2** by docs-truth (F2). **Resolved to P2.**

The mandate's P1 bar is "the delivered product produces a wrong answer, or a documented flow
fails." On every healthy toolchain the exit status is a faithful proxy for the payload and
the answer is correct. Producing a wrong answer requires `skill-validator` to lie about its
own output — exit 0 while printing non-JSON. That is a second component misbehaving, not the
delivered product.

The decisive argument is install-truth's own: it held IT-02 and IT-03 at P2 explicitly
because "their trigger — a `jq` on PATH that runs but produces nothing — is rarer in the
field." IT-01's trigger is the same shape. Rating IT-01 P1 and IT-02/IT-03 P2 on a
rarity argument is internally inconsistent; docs-truth's P2 is the consistent rating.

**This has no scheduling consequence.** G3-01 is the top-ranked entry in cluster C3 and is
repaired in the same pass either way. The demotion is about the severity ledger being
defensible, not about deprioritising the work. It is also the *cheapest* item in the audit:
`verdict-guard.sh` already owns `json_document_conforms`, the exact predicate this needs.

### G-PR5 — discharged by PR #5, not 0.4.3's

`AdapterVersion` stale across a value-changing fix (`adapter-honesty` F2 = `deferred-ledger`
L29). Genuinely P1 on delivered `main`: a profile from `multi_resource_cumulative.json`
stamps `adapter_version: 0.4.1` while reporting 300 where real 0.4.1 reported 200.
**[arch-verified]** closed at `13b3141:profiler/types.go:233` (`AdapterVersion = "0.4.2"`).
Not a 0.4.3 item **provided PR #5 merges**. If PR #5 stalls, this becomes 0.4.3's most
urgent item and the release cannot ship without it.

---

## 3. Cross-lens dedup — the 18 merge groups

The manager named three collisions and asked me to confirm them and find the rest.

### The three named, confirmed

| # | Merged as | Lens findings | Resolution |
|---|---|---|---|
| M1 | **G3-01** | install-truth IT-01 (P1) · docs-truth F2 (P2) | Same defect, `check-frontmatter.sh:44-63`. **P2** — see §2. |
| M2 | **G5-01** | docs-truth F1 (P2, REAL) · adapter-honesty F9 (P3, SUSPECTED) | Same string. **P2, REAL.** See below. |
| M3 | **G3-05** | skills-function B4 (P2) · deferred-ledger L3 (P2) · docs-truth F8 (P3) · install-truth IT-11 (P3) · install-truth IT-19 (P3) · skills-function D4 (P2) | **Six lenses-findings, one script.** `check-quality.sh` is 20 lines, sources no guard, calls no `require_tool`, emits no `DEP001`, never checks `SKILL.md` exists, and passes `skillscore`'s status through. **[arch-verified]** by reading the whole file. **P2.** |

**M2 resolution in detail.** The reason string shipped in every Claude Code profile
(`claude_code.go:208`) and restated at `docs/profiler-spec.md:248` is `"Claude Code telemetry
carries no output-to-skill mapping"`. The code's own comment ten lines above
(`claude_code.go:197-204`) concedes that `skill.name` rides on `token.usage`, `cost.usage`,
`api_request`, `api_error` and `api_refusal`. `docs/profiler-spec.md:239` states the
project's own rule in the same paragraph: *"A reason may say what this adapter does not
read; it may not say what the harness does not emit unless that is true."*

The defect is the **assertion of an absence the project has not proven**, and that is REAL
regardless of whether adapter-honesty F9's specific `prompt.id` → `tool_use_id` join works.
The patch therefore does **not** depend on settling F9's SUSPECTED status: reword the reason
to the honest shape its sibling `SkillActivation` reason at `:205-207` already uses ("this
adapter does not read X"), which is true either way. F9's join is the 0.5.0 capability
question and can be settled then. docs-truth's REAL/P2 supersedes adapter-honesty's
SUSPECTED/P3 because the REAL half is the half the patch acts on.

### The fifteen further collisions found

| # | Merged as | Lens findings | Note |
|---|---|---|---|
| M4 | **G3-04** | docs-truth F3 (P2) · install-truth IT-09 (P2) | `audit-report.sh:70-75,185` renders a silent source's `""` as `null`. Agreed P2. |
| M5 | **G4-01** | install-truth IT-08 (P2) · skills-function D3 (P2) | `audit-report.sh` always exits 0 vs `SKILL.md:87` "0=pass". Agreed P2. |
| M6 | **G8-01** | install-truth IT-07 (P2) · skills-function C5 (P2) | `skill-rewrite/SKILL.md:52,119` bare `scripts/…`. Agreed P2. |
| M7 | **G8-04** | install-truth IT-14 (P3) · skills-function C7 (P3) | `draft-rewrite.sh:55,63` mktemp leak + provenance line. Agreed P3. |
| M8 | **G7-02** | install-truth IT-16 (P3) · deferred-ledger L1 (P2) | `test_skill.sh` has no `cd`. **P2** — L1's rating wins: three separate status files now carry the workaround, which is the signal the deferral outlived itself. |
| M9 | **G7-03** | docs-truth F4 (P2) · deferred-ledger L4 (P3) | `tests/test_f01.sh:824-833` rule-ID census. Two facets of one defect: F4 = the census misses *additions*; L4 = it also misses quoted literals and indirect emitters. **P2** for the prose half. |
| M10 | **G5-06** | install-truth IT-17 (P3) · docs-truth F5 (P2) | `OtelExportFile` in a user-facing reason string. **P2** — F5's framing is strictly stronger and correct: the published Go contract has *no* documented way to supply an export, so the reason names an identifier the contract does not define. |
| M11 | **G3-03** | install-truth IT-03 (P2) · install-truth IT-12 (P3) | Same file, same root, two contract breaches (silent pass; leaked status). Merged; one repair closes both. **P2.** |
| M12 | **G5-07** | docs-truth F7 (P3) · deferred-ledger L10 (P3) | Stale `v0.4.0`. L10 is the superset: `README.md:28`, `README.md:423`, `.out-of-scope.md:8`. **[arch-verified]** all three still present at `13b3141`. **P3.** |
| M13 | **G-PR5** | adapter-honesty F2 (P2, RESOLVED) · deferred-ledger L29 (P1) | See §2. Discharged by PR #5. |
| M14 | **G9-04** | adapter-honesty F5 (P3) · deferred-ledger L16 (P2) | Reject vs ran-and-failed both `success:false`. **P2**, L16's rating — it is a stated limit but the two are genuinely indistinguishable to a consumer. Needs a schema field → **0.5.0**. |
| M15 | **G5-05** | skills-function D5 (P2) · install-truth IT-24 (P3) | **Same line.** `skills/{skill-audit,skill-rewrite}/SKILL.md:5` — `compatibility: POSIX shell (bash 3.2+ or zsh), git.` **[arch-verified]** identical in both files. False three ways: not POSIX (bash-only), false for zsh (`BASH_SOURCE[0]` unset), and `git` is used by no script. **P2** — D5's rating wins because the zsh half is behavioural: `draft-rewrite.sh` writes a draft anyway and exits 0. |
| M16 | **G6-02** | install-truth IT-05 (P2) · docs-truth F9 (P3 SUSPECTED) | Same README Codex row. IT-05's execution-grade finding (`.codex/skills/` is not a Codex path; `.agents/skills/` is) subsumes F9's citation complaint. **P2.** |
| M17 | **G1-01** | skills-function A1 (P1) · skills-function A2 (P1) | Same file, same missing strip, one repair. A1 (`PL005` dead) is a strict subcase of A2. Merged to avoid a fixer treating them as two changes. **P1.** |
| M18 | **G8-05** | skills-function C2 · C3 · C4 (all P2) | Three sentences in one paragraph (`skill-rewrite/SKILL.md:57-60`) describing a draft the script does not produce. One doc repair. **P2.** |

### Disposed — do not enter the worklist

`deferred-ledger` L8 (payload-less exit 3 — documented, pinned by tests, deliberate) ·
L12 (INVALID at `5c847e1`; the substantial `skill-rewrite` change `b73326d` is unlanded) ·
L22, L23, L24, L25 (open, honestly disclosed, no action) · L31, L32 (proved CLOSED) ·
L33 (`scuba-guard` — tooling, outside the product) · `adapter-honesty` F10 (`skill_dir`
verbatim — explicitly documented as such). **10 disposed.**

Also disposed: `adapter-honesty`'s "Devin suspicion", REFUTED — **[arch-verified]**
`git ls-tree origin/main profiler/` carries no cursor, codex or devin adapter.

### One lens disagreement worth naming, resolved as *both right*

`adapter-honesty` lists "**`Probe` never fails**; file-access classification matches
`spec:254`/`:269` in every case" under **verified clean**. `install-truth` IT-10 says
`probe` cannot distinguish a wrong `--otel-file` from a silent session. These are not in
conflict: adapter-honesty asks whether `probe` conforms to its spec (it does — `{otel, none}`
is its whole vocabulary and `none` is correct for an unreadable file), and install-truth asks
whether a user can tell a typo from a real absence (they cannot). Both stand. Carried as
**G9-01**, P2, because a user who typos the path concludes `claude_code` has no token
capability — a wrong conclusion drawn from a technically true answer.

---

## 4. Root clusters

Nine clusters, ranked by **user-visible harm**. Each names the one repair that closes every
member. The lenses' proposed roots were verified against the code, not accepted.

### Ranking

| Rank | Cluster | Harm | Release |
|---|---|---|---|
| 1 | **C1 — Body extraction** | The headline command returns a clean pass for a failing skill, on a healthy toolchain, with a realistic input | 0.4.3 |
| 2 | **C2 — Profiler provenance** | Confident numbers stamped with a session ID that did not produce them | 0.4.3 |
| 3 | **C4 — Exit-status contract** | `audit-report.sh "$skill" && echo PASS` prints PASS for a failing skill and for an unchecked one, following the doc as written | 0.4.3 |
| 4 | **C3 — Guard coverage** | Fabricated verdicts and empty payloads at success status — but only when a tool is present-and-broken | 0.4.3 |
| 5 | **C6 — Install truth** | A user following the documented route gets a failed, misplaced or duplicated install | 0.4.3 |
| 6 | **C8 — skill-rewrite integration** | The skill documents a draft its script does not produce, and the documented invocation does not run | 0.4.3 (+0.5.0 capability) |
| 7 | **C9 — Profiler numbers** | One reachable over-count; the rest is a missing schema channel | 1 item 0.4.3, rest 0.5.0 |
| 8 | **C5 — Asserted completeness** | False claims; no behaviour is wrong | 0.4.3 |
| 9 | **C7 — Verification integrity** | ~zero direct user harm; total blast radius | 0.4.3, **sequenced first** |

---

### C1 — There is no single body-extraction primitive

**Root, verified.** Four scripts answer "where does the body start" and they do not agree.

| Script | Mechanism | Verdict |
|---|---|---|
| `check-frontmatter.sh:66-73` | awk, `exit` on the second `---` | correct |
| `audit-report.sh:140-147` | same shape | correct **[lens]** |
| `check-paths.sh:64-71` | awk **toggle** — `in_fm = 0; next` / `in_fm = 1; next` | wrong: a third `---` re-enters |
| `check-structure.sh:69-93` | **no stripping at all** | wrong: every check reads the whole file |

**[arch-verified]** — `check-structure.sh:70,77,83,89` each pass `"$skill_md"` directly;
grepping the file for `in_fm` or `frontmatter` returns zero matches.

**The one repair.** Add a `skill_body` primitive to `verdict-guard.sh` — frontmatter ends at
the second `---` and never re-enters — and route `check-structure.sh` and `check-paths.sh`
through it. Migrate `check-frontmatter.sh` and `audit-report.sh` onto it too and delete their
private copies, so the count of answers to this question goes from four to one. That last
step is the integration, not an optional tidy: leaving two correct private copies means the
next reader still sees a question with more than one answer, and this class recurs.

**Authorised refactor.** Adding the primitive to `verdict-guard.sh` and deleting the three
other extraction sites is in scope for 0.4.3 and the fixer is authorised to do it. It is a
larger diff than "fix the toggle" and it is the correct size.

**Members:** G1-01 (P1), G1-02 (P1), G1-03 (P3).

---

### C2 — `resolve()` takes no identity, so no extractor can scope anything

**Root, verified.** `profiler/claude_code.go:87` — `func (a ClaudeCodeAdapter) resolve()
otelSignals`. No argument. **[arch-verified]** `grep -n 'sessionID\|SessionID'` over
`claude_code.go` at `5c847e1` returns exactly two lines: `:173` (the parameter) and `:188`
(the struct field it is stamped into). Below `resolve()` nothing can be scoped — not by
session, not by instrumentation scope, not by resource — while the result is stamped with a
caller-asserted identity the data never justified. `cmd/main.go:79` **requires** `--session`
and then discards it.

The fix is feasible and non-vacuous: **[arch-verified]** `session.id` appears 70 times across
the committed fixtures, and `README.md:234-236` states that every metric data point and every
log record carries it.

**The one repair.** Give `resolve()` a provenance argument and apply it in the extractors:
a record contributes only if it carries the asserted `session.id`, and only if its
instrumentation scope is Claude Code's. "Required and ignored" must not survive the fix.
This closes G2-01 and G2-02 together; patching one signal leaves the other.

**Members:** G2-01 (P1), G2-02 (P2), G2-03 (P3).

**Release-gate note.** This changes what a profile contains for the same input, so 0.4.3
**must** bump `AdapterVersion` to `"0.4.3"`. That is the exact invariant PR #5 exists to
establish; missing it one release later would be the same defect recurring. Put it in the
0.4.3 definition of done now.

---

### C4 — One sentence asserts an exit contract over scripts that do not share one

**Root, verified.** `skills/skill-audit/SKILL.md:87` states `0=pass, 1=spec/path failure,
2=policy failure, 3=execution error` and sits directly beneath the `audit-report.sh` usage
block. But `audit-report.sh:26` states a *different* contract for itself (`0=report
generated, 3=execution error`) and only that one is implemented; `check-quality.sh:5` states
a third (`{0,3}`) and implements none of it. The doc **restates** rather than **derives**,
which is why it drifted.

**The one repair — and a decision the manager must make.** Two patch-shaped options, and
they are mutually exclusive. See §6, pair **P-d**. My recommendation is to correct the doc
and derive it, not to change `audit-report.sh`'s status.

**Members:** G4-01 (P2), G4-02 (P2), G4-03 (P2), G4-04 (P3).

---

### C3 — PR 4 established the right invariant and applied it to part of the surface

**Root, verified.** This is the manager's dominant candidate and it holds, with one important
correction to how the lenses stated it.

PR 4's invariant is right: *no script reports a verdict a source it could not read.*
`verdict-guard.sh` implements it with three primitives. **[arch-verified]**:

- `require_tool` (`verdict-guard.sh:130-137`) asks `command -v` and nothing else. It proves a
  tool is **present**, never that it **works**.
- `json_document_conforms` (`verdict-guard.sh:~140-200`) is the payload predicate — exactly
  one document, and a caller-supplied claim true of it. It exists and is well-designed.
- `cannot_compute` writes the reason to stderr, `passed:false` to stdout, exits 3.

The gap is coverage, in three distinct directions:

1. **Tools never routed through `require_tool` at all**: `grep`, `awk`, `wc`.
   `set -e` structurally cannot catch these — `! grep …` is exempt by definition.
2. **Scripts never brought inside the guard**: `check-quality.sh` (sources nothing),
   `draft-rewrite.sh` (discards status with `|| true`).
3. **`json_document_conforms` not applied to the spec source.** This is the correction:
   `check-frontmatter.sh` does not need a *new* predicate. The predicate already exists in
   the guard; the spec source is simply not put through it. That makes G3-01 a two-line
   integration, not a feature, and settles its patch-legality beyond argument.

**The one repair.** Extend the guard's contract from "the tool is present" to "the tool is
present **and** answered something I proved I could read", route every tool dependency
through it, and bring `check-quality.sh` and `draft-rewrite.sh` inside. Then delete
`|| true` and bare `!` from every path that carries a tool's status onto a verdict.

**Authorised refactor.** `check-quality.sh` currently has no guard, no rule IDs and no
`SKILL.md` existence check. Bringing it inside means rewriting it, not amending it. That is
authorised. It is also why it escaped every test: **[arch-verified]** it emits no rule
literals, so `tests/test_f01.sh:823-828` contributes the empty set for it and the registry
assertion at `:841` passes **vacuously** over it. Fixing the script without fixing the census
(C7) leaves it untested again.

**Members:** G3-01 (P2), G3-02 (P2), G3-03 (P2), G3-04 (P2), G3-05 (P2), G3-06 (P2),
G3-07 (P2), G3-08 (P3), G3-09 (P2).

---

### C6 — Install and update instructions were written, not executed

**Root.** `README.md:329-331` hedges that the three non-Claude routes were never run. Two of
them are wrong in ways a single execution catches, and the manual-copy command — which
`README.md:362` designates as the *update* path — corrupts the install on its second run.

**The one repair.** Execute each documented route once and write down what happened; where
the command itself is wrong, change the command, not the prose. G6-03 is the one member that
is not a doc fix: `cp -R src dst` copies *into* `dst` when `dst` exists, so the documented
update path nests a second `SKILL.md` and harnesses that walk the root recursively register
the skill twice.

**Members:** G6-01 … G6-06.

**Risk flagged.** This cluster cannot be verified in CI and cannot be verified by a session
without the Devin, Cursor and Codex CLIs. See §5.

---

### C8 — skill-rewrite was never brought up to skill-audit's conventions

**Root, verified.** `git diff bd32b26 5c847e1 -- skills/skill-rewrite/` is empty
**[lens]** — PR 4 did not reach it. Its SKILL.md describes a draft `draft-rewrite.sh` does
not produce, and its documented invocation is cwd-relative where every other invocation in
both skills is anchored.

**The one repair, in two halves.** (a) 0.4.3: correct `skill-rewrite/SKILL.md` to describe
the draft the script actually produces, anchor the invocations, and bring `draft-rewrite.sh`
inside the guard (that half belongs to C3). (b) 0.5.0: build the described capability. See
§6, pair **P-a**.

**Members:** G8-01 … G8-09.

---

### C9 — The profiler reports numbers the export does not contain

**Root.** Two unrelated mechanisms sharing one consequence. (a) `otlp.go:715` gives a
`kvlistValue` its identity as `"k" + compactJSON(...)` — **[arch-verified]** — so two data
points whose attribute maps are equal under the OTel data model become two series and their
totals **add**. The comment at `:695` states the governing rule, "the order of a composite's
members is part of its value," which is right for `arrayValue` and wrong for `kvlistValue`;
the OTel common spec says maps are equal irrespective of member order. This **over-counts**,
the one direction the product's own claim ("never invented") forbids. (b) Schema v1 has no
channel to say a series or a data point was excluded from a `present` total.

**The repairs.** (a) is patch-legal — behaviour fix, no schema change — and belongs in
0.4.3 alongside C2 under the same `AdapterVersion` bump. (b) is one 0.5.0 design that closes
G9-02, G9-03, G9-04, G9-05 and G9-06 together; designing them separately produces four
incompatible channels.

**Members:** G9-01 (P2, 0.4.3 partial), G9-02 (P2, 0.4.3), G9-03 … G9-08 (0.5.0).

**Ledger correction carried forward.** `deferred-ledger` L14's original deferral reason —
"a bound on the fix, not a known bug" — is now **false**, and the decision to defer should be
retaken on the corrected reason rather than inherited.

---

### C5 — Doc sentences assert a completeness the code does not deliver

**Root, verified.** The project already enforces the correct rule on reason strings, in two
places, in its own words: `CHANGELOG.md:49` and `docs/profiler-spec.md:239` — *a claim may
say what this code does not do; it may not assert an absence or an exhaustiveness it has not
proven.* The rule is enforced on strings emitted by Go and not on prose written in Markdown.

Each member is a sentence asserting completeness — "nothing maps", "does the same",
"names the source", "cannot fall behind", "each of the scripts", "are null" — where the code
delivers a narrower guarantee.

**The one repair.** Apply the reason-string rule to the prose as one pass: every sentence
either states the narrower guarantee or names its scope. Editing them one at a time
reproduces the class, because the class is "nobody applies this rule to prose."

**Members:** G5-01 … G5-09.

**Sequencing constraint.** C5 must run **last**. Several of its sentences describe behaviour
that C1, C3 and C4 will change; writing the prose first guarantees a second rewrite.

---

### C7 — The verification substrate cannot report a failure

**Root, verified.** `tests/test_skill.sh` is the one suite never brought up to the other
three's conventions. Its `assert` takes a *value*, not a condition, and 16 of 25 call sites
pass the literal `true`. The real check is a bare `[[ ]]` on the preceding line relying on
`errexit`, which bash 3.2 does not fire for `[[ ]]`. It also lacks the `cd
"$(dirname "$0")/.."` the other three carry. Around it, `tests/test_f01.sh:824-833`'s rule-ID
census is syntax-shaped and passes vacuously over `check-quality.sh`;
`tests/lib/masked-path.sh:32-33` can build a farm that shadows a real binary with a broken
symlink, converting a real test into a vacuous exit 127; and `.github/workflows/ci.yml:23`
installs `skillscore` unpinned while `skill-validator` is pinned at `v1.6.1`.

**The one repair.** Make `assert` take a condition and make passing a literal impossible —
the simplest form is `assert <label> <command...>` that runs the command, which removes the
class rather than fixing 16 call sites. Add the `cd`. Make the census behavioural. Fix the
farm. Pin `skillscore`.

**Members:** G7-01 (P1), G7-02 (P2), G7-03 (P2), G7-04 (P3), G7-05 (P2).

**This is the reason the cluster is sequenced first.** It ranks last on user-visible harm and
first on blast radius, and I want the manager to see both facts rather than one.

---

## 5. The worklist

Severity is the reconciled rating. `file:line` is at `5c847e1`.

### Cluster C1 — Body extraction · 0.4.3

| ID | Sev | Subsumes | file:line | Invariant violated | Proof of non-vacuity |
|---|---|---|---|---|---|
| **G1-01** | **P1** | sf A1, sf A2 | `check-structure.sh:69-93` (`:70,:77,:83,:89`) | A house-policy check about the body is computed over the body | The `fmbleed2` fixture in §2 must make `audit-report.sh` report `passed:false` with ≥8 findings. A fixture that only removes frontmatter does **not** prove this. |
| **G1-02** | **P1** | sf A3 | `check-paths.sh:64-71` | Frontmatter ends once; a horizontal rule is body | Paired fixtures identical but for one `---` rule, with **all** broken refs after the rule, must give identical findings and identical exit. Findings-count parity alone is insufficient — the with-rule case must not exit 0. |
| **G1-03** | P3 | sf A4 | `check-structure.sh:89` | `PL003`'s 500-line limit is a **body** limit, as `draft-rewrite.sh:78` states | A 497-line body + 5 frontmatter lines must not raise `PL003`; a 501-line body must. |

### Cluster C2 — Profiler provenance · 0.4.3

| ID | Sev | Subsumes | file:line | Invariant violated | Proof of non-vacuity |
|---|---|---|---|---|---|
| **G2-01** | **P1** | ah F1 | `claude_code.go:87,173,188`; `cmd/main.go:79` | A profile stamped with a session ID contains only signal derived from records carrying it, or is not `present` | A two-session export must give, for session A, exactly session A's totals — and a session ID appearing nowhere in the export must give `unknown`, not confident numbers. The second half is the one that catches a filter that silently matches everything. |
| **G2-02** | P2 | ah F3 | `claude_code.go:221-230`; `otlp.go:428` | A record is Claude Code's only if its scope says so | Records from scope `some.other.product` named bare `tool_result` / `api_request` must not appear in `tool_calls`. |
| **G2-03** | P3 | ah F4 | `claude_code.go:441-443` | A comment describes the code beneath it | The comment claims "the reason below says so" for mid-run accepts; the reason is built only when `len(calls)==0`. `spec:244` is correct; the comment contradicts it. Comment-only. |

### Cluster C4 — Exit-status contract · 0.4.3

| ID | Sev | Subsumes | file:line | Invariant violated | Proof of non-vacuity |
|---|---|---|---|---|---|
| **G4-01** | P2 | it IT-08, sf D3 | `SKILL.md:87` vs `audit-report.sh:26` | A stated exit contract holds for every script it is stated over | **Decision-gated — see §6 P-d.** Under the recommended option: `SKILL.md:87` must be per-script, and a test must assert the doc matches each script's own header. |
| **G4-02** | P2 | it IT-11, dl L3 | `check-quality.sh:20` | A script's status comes from its own documented set | `skillscore` shimmed to exit 9 must give exit 3, not 9. |
| **G4-03** | P2 | it IT-12, it IT-03 | `check-paths.sh:118,:121` | As above | `jq` shimmed to exit 5 must give exit 3, not 5. |
| **G4-04** | P3 | dl L18 | `profiler/cmd/main.go:42` + the top-level switch | An explicitly requested help exits 0 | All four of `profiler -h`, `--help`, `probe -h`, `capture -h` must exit 0. Exit 1 already means usage error and 2 already means nothing-was-read, so moving help to 0 disturbs neither. |

### Cluster C3 — Guard coverage · 0.4.3

| ID | Sev | Subsumes | file:line | Invariant violated | Proof of non-vacuity |
|---|---|---|---|---|---|
| **G3-01** | P2 | it IT-01, dt F2 | `check-frontmatter.sh:44-63` | A script emits a spec verdict only from a payload it proved it could interpret | `skill-validator` shimmed to **exit 0 printing non-JSON** must give exit 3 + `DEP002`, not `frontmatter OK`. Also the reverse: `{"errors":[]}` at exit 1. An absent-tool test does **not** prove this. |
| **G3-02** | P2 | it IT-02 | `audit-report.sh:60,155-192` | A report generator exits 0 only having written a report | `jq` shimmed silent (exit 0, no output) must not give exit 0 with empty stdout. |
| **G3-03** | P2 | it IT-03, it IT-12, sf B3 | `check-paths.sh:57,118,121,137`; `check-structure.sh:89` | A `--json` exit 0 carries a payload | `jq` silent → must give exit 3 + payload, not exit 0 + nothing. Separately, `chmod 000 SKILL.md` must not give `check-structure.sh --json` exit 1 with an empty payload channel. |
| **G3-04** | P2 | dt F3, it IT-09 | `audit-report.sh:70-75,88-101,185,187` | A failing verdict names its reason | A source exiting 0 printing nothing must give a non-null `spec_error` naming the source. The error message must stop being the source's own output. |
| **G3-05** | P2 | sf B4, dl L3, dt F8, it IT-11, it IT-19, sf D4 | `check-quality.sh` (whole file, 20 lines) | Every skill-audit script computes its verdict under the shared guard | With `verdict-guard.sh` deleted, `check-quality.sh` must exit 3 naming the guard with empty stdout, as its four siblings already do. With `skillscore` absent it must emit `DEP001`. On a directory with no `SKILL.md` it must exit 3, not 1. |
| **G3-06** | P2 | sf B1 | `check-frontmatter.sh:75`; `check-structure.sh:70,77,83`; `check-paths.sh:79,99`; `audit-report.sh:149` | A tool that errored did not answer "absent" | `grep` shimmed to exit 2 against a clean skill must give exit 3, not nine fabricated `PL*` failures with `policy_error: null`. |
| **G3-07** | P2 | sf B2 | `audit-report.sh:140-147` | `audit-report.sh` exits inside `{0,3}` | `chmod 000 SKILL.md` must not give exit 2 with 0 bytes of stdout. |
| **G3-08** | P3 | it IT-13 | `check-structure.sh:135-146`; `verdict-guard.sh:198-199` | A diagnostic names the component that failed | With a healthy `check-paths.sh` shim and a broken `jq`, the `DEP002` message must not blame `check-paths.sh`. |
| **G3-09** | P2 | sf C1 | `draft-rewrite.sh:53-58` | A drafter does not report success over a failed audit | With tools masked, `draft-rewrite.sh` must not write a draft and exit 0 with a "Current state" full of policy failures for a clean skill. Remove `2>&1` and `|| true`. |

### Cluster C6 — Install truth · 0.4.3

| ID | Sev | Subsumes | file:line | Invariant violated | Proof of non-vacuity |
|---|---|---|---|---|---|
| **G6-01** | P2 | it IT-04 | `README.md:349-350` | A documented command runs | `devin plugins install --local .` — the `--local` flag is missing. Re-run against the real Devin CLI. |
| **G6-02** | P2 | it IT-05, dt F9 | `README.md:376`, `:336` | A documented path is a path the harness reads | `.codex/skills/` is not a Codex location; `$CWD/.agents/skills`, `$REPO_ROOT/.agents/skills`, `$HOME/.agents/skills`, `/etc/codex/skills` are. Cite Codex's own docs, not a third-party domain. |
| **G6-03** | P2 | it IT-06 | `README.md:364-366` vs `:362` | The documented install/update command is safe to run twice | Run it twice; `~/.claude/skills/skill-audit/skill-audit/` must not exist. This is a **command** change, not prose. |
| **G6-04** | P3 | it IT-22 | `README.md:337` | An install row names what to copy and where | Cursor: destination `~/.cursor/plugins/local/`, source = the plugin root (the directory containing `.cursor-plugin/plugin.json`), not `.cursor-plugin/` itself. |
| **G6-05** | P3 | it IT-21, it IT-23 | four `plugin.json` files | Manifest hygiene is uniform | `.devin-plugin` says `"skills": "skills"` where the other three say `"./skills/"`; no manifest has `author`, which `claude plugin validate .` warns on. |
| **G6-06** | P3 | dl L11 | `skills/skill-audit/SKILL.md:7` | A skill's `metadata.version` tracks its consumer-observable surface | `+543/-168` and two new consumer-observable rule IDs since `0.2.0` was set. Bump to `0.3.0` in 0.4.3, whose script changes are larger still. Note `skill-rewrite` 0.1.0 is **correct** for the delivered tree (dl L12 is INVALID) — do not bundle. |

### Cluster C8 — skill-rewrite integration · 0.4.3 (+0.5.0 capability)

| ID | Sev | Subsumes | file:line | Invariant violated | Proof of non-vacuity |
|---|---|---|---|---|---|
| **G8-01** | P2 | it IT-07, sf C5 | `skill-rewrite/SKILL.md:52,:119,:45` | A documented invocation runs from the cwd the doc implies | Run verbatim from the repo root. `skill-rewrite/SKILL.md` defines no `skill_root` for itself at Stage 0 (`:29-32`) — that missing input is the shared root of both hits, so anchor by defining it, not by patching two strings. `:45` also points at a `references/` directory `skill-rewrite` does not have. |
| **G8-02** | P2 | sf D1, sf D2 | `skill-rewrite/SKILL.md:46-47` | A preflight prevents the failure it exists to prevent | The preflight still exits 1 for a missing tool (PR 4 standardised on `DEP001`/exit 3) and never checks `jq`, which PR 4 made unconditional. A user who passes the preflight and runs the headline command gets exit 3 and empty stdout. Test that sequence. |
| **G8-03** | P2 | sf D7, sf D6 | `skill-rewrite/SKILL.md` | A hard dependency is documented | `verdict-guard.sh` appears nowhere in either SKILL.md, yet every script hard-depends on it and exits 3 without it. |
| **G8-04** | P3 | it IT-14, sf C7 | `draft-rewrite.sh:55,:63` | A tool cleans up after itself and states a meaningful provenance | One orphaned `mktemp` per run, and its machine-local absolute path becomes the draft's stated provenance. |
| **G8-05** | P2 | sf C2, sf C3, sf C4 | `skill-rewrite/SKILL.md:57-60` | A skill describes the artifact it produces | **Doc-honesty in 0.4.3, capability in 0.5.0 — §6 pair P-a.** Frontmatter preservation does not exist; 3 of 5 documented templates do not exist and one emitted section is undocumented; the audit-dimension checklist is fixed boilerplate, byte-identical for a clean, a faulty and an empty skill, and `evaluation-matrix.md` is never loaded. |
| **G8-06** | P3 | sf C6 | `draft-rewrite.sh` `-a` handling | A silent fallback is not a silent lie | `-a <nonexistent>` runs a fresh audit and reports "No audit report provided", which is not what happened. |
| **G8-07** | P3 | sf C8 | `draft-rewrite.sh` arg parsing | A script has an exit-code contract | `-t` with no value gives a raw `$2: unbound variable`. |
| **G8-08** | P3 | sf C9 | `skill-rewrite/SKILL.md` | Tool prerequisites are stated | Stage 1 hard-requires `skill-validator`; neither it nor `jq` is mentioned. |
| **G8-09** | P3 | dl L26 | `draft-rewrite.sh:49` | A comment describes the code beneath it | Says "relative to this script"; `:50-51` resolve two levels above it. |

### Cluster C9 — Profiler numbers

| ID | Sev | Subsumes | file:line | Rel | Invariant / note |
|---|---|---|---|---|---|
| **G9-01** | P2 | it IT-10 | `claude_code.go:154-170` | **0.4.3 (partial)** | `probe` cannot distinguish a wrong `--otel-file` from a silent session. **Patch half:** emit the same diagnostic on stderr that `capture` emits, exit unchanged — making a silent failure loud is not new capability. **0.5.0 half:** give `probe` an exit contract (README documents none today, so inventing one is new surface). See §6 pair **P-e**. |
| **G9-02** | P2 | dl L14 | `otlp.go:715`, comment `:695` | **0.4.3** | Two data points whose attribute maps are equal under the OTel data model are one series; map members are unordered, array members are not. **Non-vacuity:** two cumulative points with the same two-entry `kvlistValue` in opposite order, each reporting 100, must give `{"input":100}`, not 200. Cumulative only. |
| **G9-03** | P2 | dl L9 | schema v1 | 0.5.0 | No channel to say a refused series was excluded from a `present` total. **Highest-harm 0.5.0 item** — the one place the product silently reports a number it knows is short. |
| **G9-04** | P2 | ah F5, dl L16 | `types.go:126-128`; `claude_code.go:466,:477` | 0.5.0 | Rejected and ran-and-failed both emit `success:false`. Needs a third field — same channel as G9-03. |
| **G9-05** | P2 | dl L7 | `otlp.go` (`flags` absent) | 0.5.0 | A `NO_RECORDED_VALUE` point carrying 0 reads as a running total. |
| **G9-06** | P3 | ah F7 | `otlp.go:1048,:1024`; `claude_code.go:331` | 0.5.0 | Reports `{"input": 9223372036854775807}` with no marker that it is a ceiling. Same channel as G9-03. |
| **G9-07** | P3 | dl L15, dl L13, dl L5, dl L19, dl L6, dl L21 | various | 0.5.0 | Six honestly-held deferrals whose stated reasons were re-checked and still hold. Do not re-open. |
| **G9-08** | P3 | ah F6, ah F8 | `claude_code.go:577-616`; `types.go:56,287,322` | 0.5.0 | Span excludes `api_error`/`api_refusal` where `spec:231` names two other exclusions; `PresentActivationResult` / `PresentAttributionResult` are unreachable, and four advertised harnesses have no producer. |

### Cluster C5 — Asserted completeness · 0.4.3

| ID | Sev | Subsumes | file:line | Invariant violated | Proof of non-vacuity |
|---|---|---|---|---|---|
| **G5-01** | P2 | dt F1, ah F9 | `claude_code.go:208`; `spec:239,:248`; `README.md:150-151` | No reason asserts an absence in the harness the harness does not have | Reword to the honest shape the sibling `SkillActivation` reason at `:205-207` already uses. **Does not depend on settling ah F9.** Pin the new string in `profiler_test.go:305,866` so it cannot regress. |
| **G5-02** | P2 | dt F2 (README half), it IT-18 | `README.md:278-282` | A claim is scoped to the mode it holds in | `:282` says `check-frontmatter.sh` "does the same"; it reads exit status only. `:279-281`'s DEP002 claim reads as unconditional but holds only in `--json`. **Order:** write after G3-01 lands, so the sentence describes the code that ships. |
| **G5-03** | P2 | dt F4, dl L4 (prose half) | `README.md:286-287`; `skill-audit/SKILL.md:87` | A claim states the direction it actually covers | "cannot fall behind what they produce" — the census catches **removal** of a registered ID, not **addition** of an unregistered one, which is the direction the sentence claims. |
| **G5-04** | P3 | dt F8, it IT-19 | `README.md:298-299` | "Each" means each | Four of five. **Order:** if G3-05 brings `check-quality.sh` inside the guard, this sentence becomes true and needs no edit — check before editing. |
| **G5-05** | P2 | sf D5, it IT-24 | `skills/skill-audit/SKILL.md:5`; `skills/skill-rewrite/SKILL.md:5` | A compatibility declaration is true | False three ways: bash-only not POSIX; zsh fails (`BASH_SOURCE[0]` unset → `script_dir` collapses to cwd → guard not found); `git` used by no script. Behavioural half: under zsh the four audit scripts correctly refuse with exit 3 but `draft-rewrite.sh` writes a draft and exits 0 — fix that too, or the corrected claim is still false. |
| **G5-06** | P2 | dt F5, it IT-17 | `spec:254`; `claude_code.go:56` (reason string) | A published contract can do what its own reason instructs | The reason names `OtelExportFile`, a Go struct field, to CLI users; and the published `ProfilerAdapter`/`CaptureOpts` surface has **no** documented way to supply an export at all, so a strict library caller can only ever build all-`unknown` profiles. Doc + string only. |
| **G5-07** | P3 | dt F7, dl L10 | `README.md:28`, `:423`; `.out-of-scope.md:8` | A version attribution is current | **[arch-verified]** all three still present at `13b3141`. The *substance* of `:423` and `.out-of-scope.md:8` is still true — `compare.go` and `experiment.go` do not exist on main — so only the label is stale. |
| **G5-08** | P3 | dt F6 | `README.md:270` | A documented key is named at the level it lives at | `quality_score` / `quality_grade` exist only under `summary`; the adjacent cell uses the qualified `summary.passed`, so the unqualified pair reads as top-level. `jq '.quality_score'` returns `null` both when skillscore is missing and when it scored 94 — the exact distinction the row teaches is undetectable at the path it names. |
| **G5-09** | P3 | dt F10 | `CHANGELOG.md:45` vs `spec:131`, `types.go:73`, `capture -h` | A completeness claim about a rewording is true | Claims the rewording landed across five surfaces; three are un-reworded. `Profile.SnapshotHash` **was** corrected, so the entry is right about what it names and wrong about being complete. |

### Cluster C7 — Verification integrity · 0.4.3 · **runs first**

| ID | Sev | Subsumes | file:line | Invariant violated | Proof of non-vacuity |
|---|---|---|---|---|---|
| **G7-01** | **P1** | dl L2 | `tests/test_skill.sh:7-17` (the `assert` shape); literals at `:22,31,39,51,58,64,66,79,84,86,88,90,94,96,119,124`; inert guards at `:21,38,63,65,78,83,85,87,89,118,123` | Every assertion can print FAIL and be counted, on every bash the project supports | **Mutation, not a green run.** Delete `references/best-practices.md`; the suite must print `FAIL:` for it, reach its summary, and exit 1 — **on bash 3.2 and on bash 5**. A green run proves nothing; that is the defect. |
| **G7-02** | P2 | dl L1, it IT-16 | `tests/test_skill.sh` (no `cd`) | A suite runs from any cwd, as its three siblings do | Run by absolute path from `/`; must pass. **Same file, same repair as G7-01** — closing G7-01 alone while leaving the `assert` shape re-opens as the next round's finding. |
| **G7-03** | P2 | dt F4, dl L4 | `tests/test_f01.sh:823-833,:841,:851-855` | The rule-ID census is a question about behaviour, not about syntax | Add `cannot_compute "XX003" "mutant" true` with the ID **quoted** to `check-paths.sh`; the census must go red. It must also attribute `DEP001` to the four scripts that emit it indirectly through `require_tool`, and must see `check-structure.sh:158`'s `findings+=(...)` relay. |
| **G7-04** | P3 | it IT-25 | `tests/lib/masked-path.sh:32-33` | A masked-PATH farm masks one binary and shadows none | `[[ -e ]]` is false for a **broken** symlink and `ln -s "$f"` copies the PATH entry verbatim, so a relative PATH entry becomes a broken link that permanently shadows the real binary. Two invariants: `[[ -L … ]]` in the dedupe test, and an absolutised link target. Prove by putting a relative entry on PATH and confirming the masked run exercises the script rather than dying at 127. |
| **G7-05** | P2 | dl L20 | `.github/workflows/ci.yml:23` | A test dependency is pinned | `skill-validator` is pinned at `v1.6.1` on `:19`; `skillscore` takes whatever npm serves. `tests/test_f02.sh` (243 assertions) composes reports over its output, so a shape change turns a green suite red — or a red one green — with no repo change. One line. |

---

## 6. The 0.4.3 / 0.5.0 line

**Patch** = a fix to delivered behaviour, no new capability, no schema change.

**0.4.3 — 49 entries.** All of C1, C2, C3, C4, C5, C6, C7 and C8 (C8's doc half), plus
`G9-02` whole and `G9-01`'s stderr half.

**0.5.0 — 10 entries.** `G9-03` … `G9-08` whole; `G9-01`'s exit-contract half; `G8-05`'s
capability half; the behavioural rule census behind `G5-03`/`G7-03` beyond the cheap
widening; and `install-truth` IT-20 (`$schema` in `.codex-plugin/plugin.json` — SUSPECTED,
needs a real Codex CLI to settle, and the manifest location and skills discovery are already
correct, so there is nothing to fix until it is settled).

### Patch-shaped alternatives — every pair, with a recommendation

The manager asked for every item a lens called 0.5.0 that has a legitimate patch-shaped
alternative. There are six. In five I recommend the patch; in one the choice is genuinely
open and belongs above my level.

| Pair | Lens called it | Patch-shaped alternative | Recommendation |
|---|---|---|---|
| **P-a** · G8-05 · `skill-rewrite/SKILL.md:57-60` describes frontmatter preservation, five templates and an audit-dimension checklist the script does not produce | 0.5.0 (build it) | Correct the SKILL.md to describe the draft the script actually produces | **Patch in 0.4.3, capability in 0.5.0.** Shipping a knowingly false description for another release is the worse option, and it is the only one of the two that costs nothing. `skills-function` recommends the same; I concur on independent grounds. |
| **P-b** · G5-01 · the `attribution` reason asserts an absence | ah F9 called it 0.5.0 | Reword the reason; do not build the read | **Patch in 0.4.3.** Decisive point: the patch does **not** require settling whether the `prompt.id` join works. The defect is an *unproven assertion of absence*, which is proven a defect either way by the project's own rule at `spec:239`. Building the join is 0.5.0 and stays there. |
| **P-c** · G5-03 / G7-03 · the rule-ID list "cannot fall behind" | dl L4 called the census 0.5.0-ish | Correct the prose; separately, widen the census's literal forms | **Both, split.** Prose → 0.4.3. The *cheap* census widening (recognise quoted IDs, attribute indirect `DEP001` emitters, see the `findings+=` relay) → 0.4.3, because C7 is opening that file anyway and doing it later means opening it twice. A genuinely behavioural census (run the scripts, harvest emitted IDs) → 0.5.0. |
| **P-d** · G4-01 · `audit-report.sh` always exits 0 vs `SKILL.md:87` "0=pass" | it IT-08 / sf D3, both 0.4.3 | **A:** correct the doc to be per-script. **B:** make `audit-report.sh` exit nonzero on a failing audit. | **Recommend A — but this is the manager's call.** `audit-report.sh:26` documents `{0, 3}`, that contract is implemented and pinned by tests, and "0 = report generated" is the right contract for a *report generator*: the verdict lives in `summary.passed`, which is what `SKILL.md:82`'s own `jq` pipeline reads. Option B changes a delivered, script-documented contract and breaks every caller wired to the implemented behaviour — that is not patch-shaped even though a lens called it 0.4.3. **Risk of A:** it leaves `audit-report.sh "$skill" && echo PASS` printing PASS for a failing skill, permanently, mitigated only by prose. If the manager weighs that harm higher than the compatibility break, B is defensible and belongs in 0.5.0 with the bump. |
| **P-e** · G9-01 · `probe` cannot distinguish a wrong `--otel-file` | it IT-10 called it 0.4.3 patch | Emit `capture`'s diagnostic on stderr, exit unchanged | **Split.** The stderr diagnostic is a patch: making a silent failure loud adds no capability and changes no schema. Giving `probe` an *exit code* is new surface — README documents no exit codes for `probe` at all — so that half is 0.5.0. IT-10's "0.4.3 patch" is right about half of itself. |
| **P-f** · G9-02 · kvlist over-count | dl L14 said "0.4.3 or early 0.5.0" | Fix the identity; no schema change | **0.4.3.** Patch-legal on the definition. Note it changes a number for the same input, so it rides the same `AdapterVersion = "0.4.3"` bump as C2. Also retake the deferral decision on the corrected reason: L14's original reason ("a bound on the fix, not a known bug") is now known false. |

### One 0.5.0 item I recommend **dropping** rather than carrying

`deferred-ledger` L17 — the `cache_creation` → `cache_write` rename. Not patch-legal (it is a
`profile/v1` output key). But the case *for* it has collapsed: `claude_code.go:32` reads the
OTLP attribute value `cacheCreation`, so `cache_creation` is a faithful snake_case of the
wire name and the rename would make the profile *less* faithful. **Recommend: take it off
the 0.5.0 list as a deliberate decision and record it in `decisions.md`**, rather than
carrying it as debt into a third release. Manager's call; it is one line either way.

---

## 7. Sequencing

### Lanes

```
  Lane 0 (first, alone) ──▶ C7  Verification integrity
                             │
       ┌─────────────────────┼─────────────────────┬──────────────┬──────────────┐
       ▼                     ▼                     ▼              ▼              ▼
  Lane A: C1            Lane B: C2 + G9-02    Lane C: C3     Lane D: C6     Lane E: C8
  body extraction       profiler              guard          install        skill-rewrite
       │                     │                     │              │              │
       │                     │                     ▼              │              │
       │                     │                  C4 exit           │              │
       │                     │                  contract          │              │
       └─────────────────────┴──────────┬──────────┴──────────────┴──────────────┘
                                        ▼
                                 Lane Z (last): C5  Asserted completeness
```

### Genuinely independent — run in parallel

**C1, C2(+G9-02), C3, C6, C8** touch disjoint files and disjoint invariants. C1 and C3 both
open `check-structure.sh` and `check-paths.sh`, so they are file-adjacent but not logically
dependent: C1 changes *what text is read*, C3 changes *what happens when a tool fails*. Give
them to one fixer in that order, or to two with a rebase, but do not serialise them on a
false dependency.

### Genuinely dependent

- **C7 → everything.** Not a code dependency; an evidence dependency. See below.
- **C3 → C4.** Same files, same lines. C4's repair (exit statuses drawn from each script's
  own set) is only meaningful once C3 has made the scripts *able* to know they failed.
  Attempting C4 first produces a contract the code cannot honour.
- **C1, C3, C4 → C5.** Several C5 sentences describe behaviour those clusters change.
  G5-02 and G5-04 in particular may need no edit at all once G3-01 and G3-05 land — check
  before editing, or the prose gets written twice and is wrong in between.
- **C3 → C7's G7-03.** The census must be widened *and* `check-quality.sh` must be brought
  inside the guard, or the census stays vacuous over it. Either order works; both must land.
- **C2 + G9-02 → the `AdapterVersion = "0.4.3"` bump.** One bump covers both; it must land
  in the same release, not the same commit.

### Non-vacuity, per cluster — and where `tests/test_skill.sh` makes a fix unverified

**`tests/test_skill.sh` proves nothing structural on macOS bash 3.2.** 16 assertions pass a
literal and cannot fail; 11 bare `[[ ]]` guards do not fire. **Any fix verified only by that
suite is unverified.** Per cluster:

| Cluster | Does it rely on `test_skill.sh`? | What actually proves it |
|---|---|---|
| **C7** | It **is** `test_skill.sh` | **Cannot be self-proved.** Must be mutation-proved from outside: delete a guarded file, confirm a printed `FAIL:` plus a summary plus exit 1, **on bash 3.2 and bash 5**. A green run is exactly the evidence that is worthless here. |
| **C1** | No | `tests/test_f01.sh`, which computes every condition. Needs the two new fixtures in §5 (`fmbleed2`; the paired horizontal-rule skills with **all** refs after the rule). Without new fixtures the fix is unverified — the existing suite is green *today*, over the defect. |
| **C2**, **G9-02** | No | `go test ./...` (`profiler` at 96.0%, `-race`). Sound. Needs a two-session fixture and a kvlist-order-pair fixture. |
| **C3** | Partly — but must not | Existing suites cover tool **absence** thoroughly (`test_f01` + `test_f02`, 804 assertions) and tool **presence-without-function** not at all. **Every C3 fix must arrive with a present-but-broken shim on PATH**, not an absent tool. An absent-tool test passes today against the unfixed code, so it proves nothing. This is the single most likely way C3 ships green and broken. |
| **C4** | G4-01's doc half, yes | Assert the doc matches each script's own header programmatically. Do not verify a doc claim by reading it. |
| **C5** | No | Grep the corrected sentence against the behaviour it describes; where the sentence is scoped, add the scoping case to `test_f01`. |
| **C6** | No — and **nothing in the repo can prove it** | Requires the real Devin, Cursor and Codex CLIs, and a second run of the copy command for G6-03. CI cannot do this (`ci.yml` never runs `claude plugin validate` and never exercises an install path). **Flagged as a risk in §8.** G6-03 alone is mechanically provable: run the command twice, assert no nested directory. |
| **C8** | No | Run the documented invocation verbatim from the repo root; diff the emitted draft's section list against the SKILL.md's list. |

**General instruction to the fixer, from `integrate-dont-bolt-on`:** write each test against
the invariant named in the entry, not against the patch. C1's test must be "a body check is
computed over the body," not "the toggle was replaced" — the former survives the refactor to
a shared primitive, the latter fights it.

---

## 8. Risks

Ranked by the chance the manager has to re-open this work later.

1. **C3 ships green and broken.** The existing 804 absence-assertions pass against the
   *unfixed* code. A fixer who adds more absence tests will see green and believe the guard
   is closed. The mitigation is stated per entry, but it is the one place where following
   the existing test conventions produces a false pass. **Highest re-work risk in the audit.**

2. **`check-quality.sh` gets amended rather than rewritten.** It has no guard, no rule IDs
   and no `SKILL.md` check. Six lens-findings land on it (M3). Amending it closes the six
   symptoms and leaves it structurally outside the class, which is exactly how it arrived
   here — PR 4 touched its four siblings and not it. The plan authorises rewriting it.

3. **C1's shared primitive gets added without the other three sites being migrated.** If
   `check-frontmatter.sh` and `audit-report.sh` keep their private (correct) extractors, the
   count of answers to "where does the body start" goes from four to three, and the next
   script added to this skill will invent a fourth. The migration is authorised; a reviewer
   should check it happened.

4. **The `AdapterVersion` bump is forgotten.** C2 and G9-02 both change what a profile
   contains for the same export. PR #5 exists to close precisely this defect one release
   earlier. Missing it in 0.4.3 is the same bug, recurring, in the release that was supposed
   to close the gaps. Put it in the definition of done, not in a fixer's memory.

5. **C6 cannot be verified here.** Three of five install routes need CLIs that are not on
   this machine (Cursor, Codex) or a live cloud account (Devin's non-`--local` route). If
   the fixer cannot execute them, the honest outcome is to **extend** `README.md:329-331`'s
   existing "not verified in this release" hedge to cover what was still not verified, not
   to write a confident correction from documentation. The hedge is the honest pattern this
   repo already uses; use it rather than replacing one unverified claim with another.

6. **P-d is a real decision, not a formality.** Recommending the doc fix leaves
   `audit-report.sh && echo PASS` printing PASS for a failing skill, permanently. I think
   that is the right trade because the alternative breaks a contract the script itself
   documents and tests pin — but it is a user-facing harm being accepted deliberately, and
   the manager should accept it explicitly rather than inherit it from a table.

7. **C5 written too early.** Low cost, high annoyance: the prose gets written twice and is
   wrong in between. Sequencing handles it if the sequencing is honoured.

---

## 9. PR #5 — the merge decision

**The manager's position is correct. PR #5 (`13b3141`) should merge as is. Nothing in this
audit blocks it.** Five reasons, then the one thing to carry forward and the one thing that
could have changed the answer.

1. **It closes a live P1.** `AdapterVersion` stale across a value-changing fix (M13) is a
   genuine P1 on delivered `main`: a profile stamps `0.4.1` while reporting 300 where real
   0.4.1 reported 200, so a stored profile compared against a fresh one shows a 50% token
   regression that is entirely the reader changing. Holding PR #5 keeps that P1 open for no
   gain. **[arch-verified]** closed at `13b3141:profiler/types.go:233`.

2. **Its diff does not collide with the P1 clusters.** **[arch-verified]** 12 files: five
   manifests, `CHANGELOG.md`, `README.md` (4 lines), `RELEASE_NOTES.md`, `profiler/otlp.go`
   (comment), `profiler/profiler_test.go` (comment), `profiler/types.go` (the constant), and
   `tests/test_skill.sh`. The only file a 0.4.3 cluster must rewrite is `tests/test_skill.sh`,
   and PR #5's change there is ten version literals — which C7's rewrite must preserve
   anyway. Trivially absorbed.

3. **I mutation-tested PR #5's own central claim under the defective suite, and it holds.**
   This is the thing that could have changed the answer, so I tested it rather than reasoning
   about it. On a scratch copy of `13b3141`, setting `.codex-plugin/plugin.json` to `9.9.9`
   **aborts the suite** on bash 3.2 with `AssertionError: version mismatch`. The reason is
   the characterisation in §2: bash 3.2 exempts `[[ ]]` from `errexit` but **not** python
   heredocs and **not** `grep -q`. Every one of PR #5's six version surfaces is enforced by a
   heredoc or by the `grep -q` at `:57`. **G7-01 does not undermine PR #5's verification.**

4. **PR #5 adds no false claim.** Its changelog entry is scoped to the token merge and the
   verdict guard, both of which this audit's clean-lists confirm. `README.md`'s two edits are
   exactly the two version references its changelog line 34 claims to move. It uses the
   project's own "Known past changes, recorded late" mechanism to correct 0.4.1's false
   claims without rewriting them, which is the right pattern.

5. **The one audit item that touches PR #5's content does not block it.**
   `deferred-ledger` L30: PR #5's `CHANGELOG.md:52` **Known limits, carried to 0.5.0** block
   names four items where this audit finds eighteen open. That gap is not a defect in PR #5 —
   **the complete list is this audit's output and post-dates the PR.** `release-commit-draft.md`
   §6b asked the manager in writing whether the scripts team's items belonged in that block,
   the question was never answered, and the implementer correctly declined to extend the list
   on its own judgement. Handle it in **0.4.3**, using the mechanism 0.4.2 itself established:
   0.4.3's changelog carries the full open set and, under "Known past changes, recorded late,"
   states that 0.4.2's block was the limits of the token merge and the verdict guard rather
   than of the product. **Do not rewrite a shipped entry.** Carried as a 0.4.3 item, not a
   PR #5 item.

**Carry forward, not a blocker.** PR #5 establishes the convention that changing what a
profile contains bumps `AdapterVersion`. C2 (G2-01, G2-02) and G9-02 all change numbers, so
**0.4.3 must bump to `"0.4.3"`**. Add it to the 0.4.3 definition of done at grooming, not at
implementation — see §8 risk 4.

**Two things I checked that are *not* problems, so they do not get re-raised:**

- `README.md:28` and `:423` still say `v0.4.0`, and `.out-of-scope.md:8` with them
  **[arch-verified]** at `13b3141`. Not PR #5's job: its changelog says "the README's two
  version references move with it," meaning the two `adapter_version` references, and those
  are exactly the two it moved. G5-07, 0.4.3, P3.
- `skills/skill-audit/SKILL.md:7` is still `version: "0.2.0"` and PR #5 is the version PR.
  Arguably in its remit, but it is P3 with no user-visible effect — nothing in `tests/` or the
  scripts reads `metadata.version` — and forcing it in re-opens a clean PR for a cosmetic.
  **Recommend 0.4.3, bumped to `0.3.0`**, whose script changes are larger than 0.4.2's were.
  G6-06.

---

## 10. Open questions for the manager

Three, in descending order of consequence.

1. **P-d (G4-01)** — does `audit-report.sh` keep its implemented `{0, 3}` contract and the
   doc gets corrected, or does it start exiting nonzero on a failing audit? I recommend the
   former and have argued it in §6, but it deliberately accepts a user-facing harm
   (`&& echo PASS` on a failing skill) and that acceptance should be yours, not a table's.

2. **L17 (`cache_creation` → `cache_write`)** — drop it as a deliberate decision, or carry it
   to the schema bump that G9-03/G9-04 will need anyway? I recommend dropping it: the wire
   name is `cacheCreation`, so the current key is the faithful one and the rename would make
   the profile less accurate, not more.

3. **C6 verification** — if the Devin, Cursor and Codex CLIs are not reachable to the fixer,
   do you want the corrections written from vendor documentation (fast, and re-introduces the
   exact class this cluster exists to close), or the existing "not verified in this release"
   hedge extended to cover them (honest, slower to close)? I recommend the hedge.
