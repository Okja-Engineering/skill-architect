# confirm-promise-guard — final focused adversarial pass, PR #22, head `34976a2`

Scope: `tests/lib/forward-promise-check.sh`, its fixtures in `tests/test_skill.sh`,
and the sentences in `CHANGELOG.md` / `RELEASE_NOTES.md` that describe it.
Worktree: `/private/tmp/claude-501/-Users-matthewvandusen-Development-Auraprix-skill-architect/31edaf48-6b19-4833-82e7-6cb52dac691f/scratchpad/pg7/head` (detached at `34976a2`).
Drivers: `.../scratchpad/pgdrive/drive.sh` (head) and `.../pgdrive/drive-base.sh`
(the same file as at `bf045ce`) — each feeds one line at a time through
`forward-promise-check.sh --over <fixture> 0.5.0` and prints FIRE / PASS.
The primary tree at `/Users/matthewvandusen/Development/Auraprix/skill-architect` was
never touched; `scratchpad/pgfix` was never entered. Every mutation below was made in
my own worktree and reverted, with `git status --porcelain` confirmed empty after each.

**Driver fidelity check (so the denominator means something):** all 35 published
fixture cases extracted mechanically from `tests/test_skill.sh:822..857` and replayed
through the head driver. 24 refuse -> FIRE, 11 accept -> PASS, `agree=35`, zero
mismatches. The driver reproduces the suite's verdicts exactly.

---

## Verdict up front

**The seventh is there.** Block 2. Same class, one level in: the fix normalised the
*record* and left the *interior of every pattern* enumerating forms.

| # | finding | label | gates the tag? |
|---|---|---|---|
| 2 | Markup / punctuation / digits between verb and version defeat all five shapes (44 of 51 driven cases missed) | REAL, pre-existing, mis-described as closed | judgement call |
| 3 | The fold was applied to the *exemption* too, so 5 promises the guard caught at `bf045ce` it no longer catches | REAL, **fix-introduced**, undisclosed | **yes** |
| 4 | Residual 3 confirmed and worse than stated; its cited mitigation **does not exist in the suite** | REAL | **yes** (the false sentence) |
| 6 | CRLF silently disables one of five shapes | REAL but dormant (0 CRLF files, no `.gitattributes`) | no |
| 7 | `101`/`91` is a self-referential, unasserted numeral the release's own rule forbids | REAL prose defect | no |

Self-reported residuals: **#1 refuted**, **#2 confirmed and accurately described**,
**#3 confirmed, understated, and its mitigation clause is false**.

The headline repair itself is sound. Two independent sweeps over the case/position
axis (172 cases, then 76 different ones) are **completely dry**.

---

## The matcher under test

`tests/lib/forward-promise-check.sh:237-249`:

```awk
function shape_of(rec) {
  if (rec ~ ("(tracked|deferred|carried|reserved|scheduled|planned|postponed)[ a-z]* (for|to|until|with it for) v?" v))
    return "deferral verb"
  if (rec ~ ("is +`?v?" v "`?( work|\\.|,|$)") && rec !~ /version|release \*?is\*?/)
    return "is <version>"
  if (rec ~ ("will [a-z]+ [a-z ]*in v?" v))    return "will ... in <version>"
  if (rec ~ ("needs? [a-z ]*in v?" v))         return "needs ... in <version>"
  if (rec ~ ("new surface (for|in) v?" v))     return "new surface for <version>"
  return ""
}
```

Case is folded once by the caller at `:295` (`rec = tolower($0)`). Tests are unanchored.

---

## Attack block 1 — declared vocabulary x case x leading marker: DRY

Denominator: the 7 declared deferral verbs (`tracked deferred carried reserved
scheduled planned postponed`) x the 4 declared prepositions (`for`, `to`, `until`,
`with it for`) x 4 case spellings (lower / Title / UPPER / MiXeD) = 112 cases; plus
15 leading markers (`**`, `__`, `*`, `_`, `- `, `+ `, `> `, `# `, tab, four spaces,
`1. `, `  - `, `## `, `  * `, none) x 4 case spellings = 60 cases. **172 total.**

Result: **172/172 FIRE. Zero misses.** The case fold and the unanchored tests do hold
the whole declared vocabulary in every case and behind every leading marker I could
construct. The headline repair is real and complete on the axis it claims.

Files: `.../pgdrive/block1.txt`, `.../pgdrive/block1.out`.

---

## Attack block 2 — THE SEVENTH. REAL. Markup *between* the verb and the version

**Invariant broken:** "A vocabulary is a set of verbs, not a set of spellings"
(`forward-promise-check.sh:70`). The fix folds case at the *record* level but leaves
the *interior* of every shape pattern enumerating forms: `[ a-z]*` (lowercase letters
and spaces only), a literal single space before the version token, and a bare
unquoted version token. The recurring class is still live — it has only moved from
"the verb's capitalisation" to "anything sitting between the verb and the version".

Denominator: 51 cases crossing the declared verbs and the four non-deferral shapes
against interior markup (`**`, `__`, `*`, `_`, backtick), interior punctuation
(`,` `:` `;` `(...)` `[...]` `--` `/` `&`), interior digits, interior hyphenation,
markup *on the version token*, and non-single-space separators.

**Result: 44 of 51 PASS — the guard walks past 44 constructed forward promises.**

### 2a. Bolding / emphasising / coding the verb alone defeats the deferral shape

All PASS (guard silent) against version `0.5.0`:

```
**Deferred** to 0.5.0          *Deferred* to 0.5.0        _deferred_ to 0.5.0
`deferred` to 0.5.0            **deferred** to 0.5.0      __Deferred__ to 0.5.0
```

This is the sharpest form of the finding. `forward-promise-check.sh:236` picks as its
worked example `- **Deferred to <version>**:` — bold wrapping the *whole phrase* — and
that one does fire. Move the closing `**` two words left, to `- **Deferred** to 0.5.0:`
— bolding just the verb, an equally ordinary markdown idiom and the one this repo's
own release notes use for lead-ins — and the guard is silent. The comment's claim that
"a bold or emphasis lead-in ... needs no rule of its own" holds only for the exact
bracketing the fixture happens to use.

**Fixture blind spot confirmed:** every one of the 24 refuse cases at
`tests/test_skill.sh:822-846` places its markup *outside* the verb-to-version span
(`**Deferred to @V**:`, `- **Planned for @V**`, `*Deferred to @V.*`). No control
drives markup *inside* that span. That is precisely why rebuilding the fixtures from
18 to 35 cases did not reach this.

### 2b. Ordinary punctuation between verb and preposition defeats it

```
Deferred, to 0.5.0        Deferred: to 0.5.0        Deferred; to 0.5.0
Deferred (again) to 0.5.0 Deferred -- to 0.5.0      deferred (see #12) to 0.5.0
scheduled - for 0.5.0     planned, for 0.5.0        postponed again; to 0.5.0
reserved [see note] for 0.5.0                       Deferred once more, to 0.5.0
```

All PASS. Cause: `[ a-z]*` at `:238`. Two cases in this group *do* fire — `Deferred /
postponed to 0.5.0` and `Deferred & carried to 0.5.0` — but only because a *second*
declared verb happens to land adjacent to the preposition, not because the
punctuation was handled. That coincidence matters: an exit-status spot check on this
group would have looked partly green.

### 2c. A digit between verb and preposition defeats it

```
tracked in phase 2 for 0.5.0      carried over 3 rounds to 0.5.0
```
Both PASS. `[ a-z]*` excludes `0-9`.

### 2d. Markup on the version token defeats the deferral shape entirely

```
deferred to **0.5.0**    deferred to `0.5.0`    deferred to *0.5.0*
deferred to _0.5.0_      deferred to [0.5.0](CHANGELOG.md)
deferred to "0.5.0"      deferred to (0.5.0)    deferred to <0.5.0>
deferred **to** 0.5.0    deferred `to` 0.5.0
```

All PASS. The deferral branch matches a bare `v?0\.5\.0` with nothing tolerated on its
left. The asymmetry is the tell: the `is <version>` branch at `:240` *does* tolerate a
backtick (`` is +`?v?<v>`? ``), so the author already knew version tokens get quoted —
the tolerance was simply never carried to the other four branches.
`deferred to [0.5.0](CHANGELOG.md)` is the link-label case from the mandate, and it is
missed. So is ``deferred to `0.5.0` `` — which is how anyone writing markdown would
naturally set a version.

Trailing punctuation is fine: `0.5.0.`, `0.5.0,`, `v0.5.0` all FIRE (the match is
unanchored on the right). The defect is strictly on the **left** of the version token.

### 2e. Anything but one literal space before the version defeats it

```
deferred to  0.5.0    (two spaces)   PASS
deferred to<TAB>0.5.0 (tab)          PASS
deferred<TAB>to 0.5.0 (tab)          PASS
```

Cause: the literal `" v?"` in the pattern. A table cell, an aligned code comment, or a
reflow leaving a double space is enough. Note `deferred  to 0.5.0` (two spaces after
the *verb*) **does** fire, because `[ a-z]*` absorbs the extra space — so the two gaps
behave differently, which is itself evidence nobody drove this axis.

### 2f. The other four shapes carry the identical defect

```
will be **added** in 0.5.0     will be re-scoped in 0.5.0    will land phase 2 in 0.5.0
Will be `wired` in 0.5.0       needs a **receiver** in 0.5.0 needs a re-scoped fix in 0.5.0
needs 2 adapters in 0.5.0      new surface for **0.5.0**     new-surface for 0.5.0
is **0.5.0** work              is *0.5.0* work
```

All PASS. `will [a-z]+ [a-z ]*in` and `needs? [a-z ]*in` carry the same
lowercase-letters-and-spaces class, so a hyphen (`re-scoped`), a digit (`phase 2`,
`2 adapters`) or any markup breaks them. `is **0.5.0** work` and `is *0.5.0* work` miss
because the `` `? `` tolerance covers backticks only, not asterisks.

Sanity anchors inside the same run: `is 0.5.0 work.` FIRE, `is  0.5.0 work` FIRE.

### Provenance

Checked against `bf045ce` (`git show bf045ce:tests/lib/forward-promise-check.sh:208-217`):
the interior character classes are **byte-identical** to head. So block 2 is
**pre-existing, not introduced by `34976a2`**. It is nonetheless the seventh instance
of the named recurring class, it sits inside this commit's declared scope, and the
commit's own prose ("A position was never a shape"; "all five shapes match in any case
and anywhere in the line") reads as though the class were closed when it is not.

Files: `.../pgdrive/block2.txt` and its output.

**Label: REAL (pre-existing; mis-described as closed by this commit).**

---

## Attack block 3 — REAL, and this one IS fix-introduced: the fold widened the exemption

The mandate asked whether folding the *exemption* made the accept side over-broad.
It did, and it is measurable.

- `:240` at head:  `... && rec !~ /version|release \*?is\*?/`   (`rec` is `tolower($0)`)
- `:210` at base:  `... && $0 !~ /[Vv]ersion|release \*?is\*?/` (raw record)

Because the exemption is now tested against the folded record, `VERSION`, `VeRsIoN`
and `RELEASE IS` match it where `[Vv]ersion` and `release \*?is\*?` did not. Driven
through both drivers, same 10 inputs:

| line | base `bf045ce` | head `34976a2` |
|---|---|---|
| `# VERSION MATRIX: building the capability is 0.5.0` | FIRE | **PASS** |
| `# VERSION: making it branchable is 0.5.0 work` | FIRE | **PASS** |
| `# THE VERSION TABLE says building the receiver is 0.5.0` | FIRE | **PASS** |
| `# VeRsIoN note: building the capability is 0.5.0` | FIRE | **PASS** |
| `# RELEASE IS 0.5.0 and building the receiver is 0.5.0` | FIRE | **PASS** |
| `the version matrix says building the capability is 0.5.0` | PASS | PASS |
| `# Version note: building the capability is 0.5.0` | PASS | PASS |
| `// bump the version string; making it branchable is 0.5.0 work` | PASS | PASS |
| `# building the capability is 0.5.0` | FIRE | FIRE |
| `this release is 0.5.0` | PASS | PASS |

**Five genuine forward promises that the guard caught at `bf045ce` it no longer
catches at `34976a2`.** Each is the exact text of a declared refuse fixture
(`# building the capability is @V`, `// making it branchable is @V work`) with an
upper- or mixed-case "VERSION" or "RELEASE IS" elsewhere on the same line.

Two things to keep straight:

- The **general** over-breadth — the bare substring `version` anywhere on a line
  exempts that whole line from the `is <version>` shape (rows 6-8) — is pre-existing
  **and disclosed**, at `:117-120`: "A forward promise written into a line that also
  mentions a version would get past this." That disclosure is honest.
- The **widening** is new and is **not** disclosed. The commit and both documents
  describe the fold entirely as a refuse-side repair ("all five shapes match in any
  case"). Neither says the fold was also applied to the exemption, nor that doing so
  enlarges the disclosed hole by the set of upper- and mixed-case spellings. The
  fixture `ADAPTERVERSION IS @V AND IS NOT HELD EQUAL` (`tests/test_skill.sh:847`) is
  an **accept** case, so the suite now positively asserts the widened behaviour as
  intended — with no control anywhere distinguishing "ALL CAPS version *surface*" from
  "ALL CAPS version word sitting on a line that also makes a promise".

Net on the `is <version>` shape: the fold closed the upper/mixed-case refuse spellings
and opened the upper/mixed-case exemption spellings in the same stroke.

Fix that keeps the intended accept case and drops the widening: test the exemption
against the **raw** record (`$0 !~ /[Vv]ersion|.../`) rather than `rec`, and add the
one spelling the `ADAPTERVERSION` fixture actually needs, or better, exempt only when
the version-surface token is adjacent to the `is`. The former is one character of
change and restores the five rows above to FIRE while keeping `ADAPTERVERSION IS 0.5.0
AND IS NOT HELD EQUAL` accepted — because that line's shape match at head comes from
the fold and would need the `[Vv]ersion` alternative extended by `VERSION` only.
Whoever takes this should drive both fixture sides again, not reason about it.

Files: `.../pgdrive/block3.txt`.

**Label: REAL, fix-introduced, undisclosed.**

---

## Attack block 4 — Residual 3 (the heading probe): REAL, and its stated mitigation does not exist

This is where the fixer said it would look next. It was right to, and it understated it.

### 4a. Five ordinary ways to write the heading silently drop the whole section

Driven over the **real tracked tree**, mutating only `CHANGELOG.md:7` and restoring
after each run. Baseline (`## 0.5.0 — 2026-09-21`) gives
`skipped-in=CHANGELOG.md scope=247`; the whole file is 1,059 skippable lines.

| heading written as | rc | CHANGELOG scope-skips |
|---|---|---|
| `## 0.5.0 — 2026-09-21` (actual) | 0 | 247 |
| `## [0.5.0] — 2026-09-21` | 0 | **1059** |
| `## V0.5.0 — 2026-09-21` | 0 | **1059** |
| `##  0.5.0 — 2026-09-21` (two spaces) | 0 | **1059** |
| `## Release 0.5.0 — 2026-09-21` | 0 | **1059** |
| `### 0.5.0 — 2026-09-21` | 0 | **1059** |
| `## 0.5.0 - 2026-09-21` (ASCII dash) | 0 | 247 |

`scope=1059` means **the entire changelog left the walk** and the reader still exited 0.

`## [0.5.0]` is the Keep a Changelog convention — the most likely rewrite anyone would
make to this file. `## V0.5.0` is a capital V: the probe at `:270-271` does
`sub(/^## v?/, "", probe)` against the **raw** `$0`, before any fold, so the one
lowercase letter in the probe is itself a case-sensitivity bug — in the commit whose
entire subject is case-sensitivity. `sub(/[^0-9.].*$/, "", probe)` then truncates at
the first non-digit/non-dot, so `[0.5.0]`, ` 0.5.0`, `V0.5.0` and `Release 0.5.0` all
reduce `probe` to the empty string, which never equals `version`.

### 4b. A live forward promise inside the section, and the reader exits 0

End to end, on the real tree:

1. Planted at `CHANGELOG.md:10`, inside the 0.5.0 section, heading untouched:
   `- A receiver subcommand for the spool is deferred to 0.5.0 and will land later.`
   -> reader reports `CHANGELOG.md:10: [deferral verb] ...`, **rc=1**. Correct.
2. Same planted line, heading only rewritten to `## [0.5.0] — 2026-09-21`:
   -> **rc=0**, no finding, and the accounting is perfectly self-consistent:
   `files=162 lines=43852 examined=42674 skipped=1178` (42674 + 1178 = 43852).

So the failure mode is exactly "exit 0 with self-consistent accounting over a section
nobody read", which is the failure mode this entire file was written to make
impossible.

### 4c. The disclosed mitigation is not real — a false statement in the source

`forward-promise-check.sh:128-133` discloses the residual and then names its mitigation:

> "... which is why tests/test_skill.sh holds the **per-file skip counts** against this
> script's own HISTORY_DOCS rather than trusting the accounting to add up."

The header at `:44-47` makes the same claim.

**`tests/test_skill.sh` does not hold the per-file skip counts.** The only assertion
over `skipped-in=` is `tests/test_skill.sh:1070-1077`:

```sh
promise_scoped_reported() {
  { "$FORWARD_PROMISE_SCAN" "$skill_release_version" 2>/dev/null || true; } \
    | sed -n 's/^skipped-in=\([^ ]*\).*/\1/p' | sort -u
}
assert "the reader skipped records only in the files its own scope names, and in all of them" \
  test "$(promise_scoped_reported)" = "$(promise_scoped_in_script)"
```

The `sed` discards everything after the filename. It compares a **set of file names**,
nothing numeric. Verified by exhaustive grep: `scope=` and `exempt=` appear nowhere in
`tests/` outside `forward-promise-check.sh` itself, and `skipped-in` appears at exactly
one call site, the one above.

Consequence: under 4b the reported set is still
`{CHANGELOG.md, RELEASE_NOTES.md, profiler/profiler_test.go}` — unchanged — so the
assertion passes while 247 skips became 1,059. `:131-133` offers a non-existent check
as the reason the residual is acceptable. That sentence has to go or become true.

**Label: REAL (three parts): the residual is real and understated; the probe is
case-sensitive in a case-sensitivity fix; and the mitigation cited in the source and
relied on to justify shipping the residual does not exist.**

Cheapest closing move: assert `examined > 0` for each history doc, or assert the
CHANGELOG / RELEASE_NOTES scope-skip count equals `(file records - section records)`
derived independently. Either would have reddened 4b.

---

## Attack block 5 — Residuals 1 and 2, driven

### Residual 1 (`PROMISE_CASES_EXPECTED` / `promise_n` share a source): **REFUTED**

The claim is the both-sides-shrink shape. It is not that shape.
`tests/test_skill.sh:863` is `PROMISE_CASES_EXPECTED=35` — a **hardcoded literal**,
ten lines below the closing `PROMISE_CASES` delimiter at `:860`. `promise_n` is derived
from the heredoc; the literal is not. They do not share a source.

Driven: deleted the line `refuse|DeFeRrEd To @V.` from the heredoc (34 cases remain)
and ran `bash tests/test_skill.sh`.

```
124 passed, 1 failed
FAIL: every one of the 35 forward-promise cases was driven, not a prefix of them
```

**rc=1.** Shrinking the heredoc reddens the suite. The control works. Suite file
restored; tree clean. (Incidental confirmation of the commit's arithmetic: 126
assertions in this suite, minus the one fixture assert that no longer runs, gives 125.)

### Residual 2 (an apostrophe in any awk-body comment): **CONFIRMED, fails closed**

Driven: replaced the comment at `forward-promise-check.sh:300` with
`# the walk's unit is a line` — a single apostrophe inside the single-quoted awk
program — and ran the reader over the tree.

```
rc=2
./tests/lib/forward-promise-check.sh: line 325: syntax error near unexpected token `prev'
```

Confirmed on both counts, exactly as disclosed:
- **It fails closed.** rc=2, not 0. Every caller treats non-zero as failure
  (`assert ... quietly "$FORWARD_PROMISE_SCAN"` at `tests/test_skill.sh:837`), so the
  suite reddens. There is no silent-pass path.
- **The diagnostic points nowhere near the cause.** Reported at line 325; the
  apostrophe is at line 300. 25 lines of misdirection.

**Does any CI step catch it?** No. `.github/workflows/ci.yml` is the only workflow and
contains no `bash -n` and no `shellcheck` step; exhaustive grep for `bash -n`, `sh -n`
and `shellcheck` across `.github/`, `tests/` and `Makefile` returns only
`tests/test_f01.sh:860`, a fixture assertion about an unrelated guard. The reported
"no `bash -n` in CI" is accurate. (`bash -n` *does* detect it — it is simply not run.)

Net: a real ergonomic hazard, accurately described, cannot produce a false green.

---

## Attack block 6 — context, degenerate files, and the window

### 6a. Fenced blocks, inline code, tables, HTML comments: everything fires, and that is the stated design

All FIRE:

```
The guard prints `deferred to 0.5.0` as a deferral verb
| shape | example | | deferral verb | deferred to 0.5.0 |
<!-- deferred to 0.5.0 -->
[deferred to 0.5.0](#notes)
> deferred to 0.5.0
    deferred to 0.5.0
// TODO: deferred to 0.5.0
```

A real fenced block fires too: a 7-line file whose fence contains
`foo.md:3: [deferral verb] deferred to 0.5.0` reports at line 4, rc=1.

**Which SHOULD fire?** The repo has already decided, at `:80-84`: "Seeing it refuse its
own documentation is the check working, not a false positive." I agree with the
decision for this repository — it is documentation-heavy, a false negative ships a lie
to users, and a false positive costs one rewording. It is also internally consistent:
both release documents visibly work around it by describing the caught shape without
spelling it.

But name the coupling, because it lands on whoever fixes block 2: **the guard's
tolerance and the documents' vocabulary are one system.** Widening the matcher to
catch `deferred to `0.5.0`` will also make it fire on prose *about* the guard that is
currently safe. Any block-2 fix must be re-driven against the two release documents,
not just against fixtures. A fence-awareness rule is the wrong answer here — it would
reintroduce the enumerate-the-form defect at a third level.

One inconsistency worth recording: inline code around the *whole phrase*
(`` `deferred to 0.5.0` ``) fires, inline code around the *version only*
(``deferred to `0.5.0` ``) does not. So today an author can write a genuine forward
promise in perfectly idiomatic markdown and pass. That is block 2d restated as an
authoring hazard rather than a regex fact.

### 6b. Degenerate files

| input | rc | reader output |
|---|---|---|
| CRLF, `a receiver is deferred to 0.5.0` | 1 | fires |
| CRLF, `building the capability is 0.5.0` | **0** | **misses** |
| LF, same line (control) | 1 | fires |
| no trailing newline | 1 | `lines=1`, fires |
| zero-byte file | 0 | `files=0 lines=0 examined=0 skipped=0` |
| file that is only `## 0.5.0` | 0 | `files=1 lines=1 examined=1` |

**REAL, dormant: CRLF disables exactly one of the five shapes.** Driven across all
five: LF fires on all six representative lines; CRLF fires on five and misses the bare
`is <version>` shape. Cause is the right-hand alternation `( work|\.|,|$)` at `:240` —
`$` cannot match with `\r` occupying the end of the record. The other four shapes are
unanchored on the right and survive.

Exposure today is zero: there is no `.gitattributes`, and `git ls-files -z | xargs -0
grep -lI $'\r'` returns nothing. So this is latent, not live — it would arm the moment
one CRLF file is committed, and it would arm silently. Worth one character (`\r?` or
`[[:space:]]*$`), not worth blocking a tag.

The zero-byte and no-trailing-newline behaviours are both correct and both already
documented (`:33-37` on `grep -ac ''` vs `wc -l`; `RELEASE_NOTES.md:26` on
`profiler/testdata/otlp/empty.json` being why `files=162` and `git ls-files` is 163).

### 6c. The two-record window: no false positives found, and the residual is real

| input | rc |
|---|---|
| verb on line 1, version on line 2 | 1 — `[deferral verb across a line break]` |
| verb on line 2, version on line 3 (still adjacent) | 1 — reported at line 3 |
| verb line 1, filler line 2, version line 3 | **0** — unseen |
| verb line 1, two filler lines, version line 4 | **0** — unseen |
| verb, blank line, version | 0 — join reset, correct |
| verb line ending `.`, then version line | 0 — punctuation blocks, correct |
| two separate bullets `- ...deferred` / `- To 0.5.0 we...` | 0 — no false positive |
| whitespace-only separator line | 0 |
| two unpunctuated consecutive prose lines | 1 — correct, that *is* the wrap |

So the three-record spread is genuinely unseen, as published.

**But the published wording is loose in the reassuring direction.** Both documents say
the window is two records "so a promise spread over three lines **with neither verb nor
version beside the join** goes unseen". In my three-line case the verb *is* beside a
join (records 1-2) and the promise is still unseen — because no single join contains
*both*. A reader would take the published sentence to mean a three-line promise with
the verb next to a break is caught. It is not. The accurate statement is simply: **the
verb and the version must land on adjacent records.** Recommend that wording; it is
shorter and it is true.

### 6d. `--over` vs the tracked-tree path

They differ, exactly as `:200-201` states, and the difference is load-bearing:

- `--over CHANGELOG.md 0.5.0` -> rc=1, three findings, all in the **0.4.3 and 0.4.2
  sections** (`:875`, `:897`, `:954`), because `--over` applies no section scoping.
- `--over profiler/profiler_test.go 0.5.0` -> rc=1, two findings at `:325` and `:338`
  — the negative-control lines the tree path exempts.

Both are correct behaviour for a mode whose purpose is driving fixtures. The
consequence already noted in `tests/test_skill.sh:1085-1090` stands and is worth
restating: **all 35 fixture controls run in `--over` mode, so not one of them exercises
section scoping or the exemption.** That is structurally why block 4 (scoping) and
block 3 (exemption) both went unheld.

Incidental: `--over` still requires the *process's* cwd to be inside a git work tree
(`:176-181`), even though it reads a single absolute path. Running it from `/tmp`
returns rc=2 with "not inside a git work tree". Defensible, mildly surprising, not a
finding.

---

## Attack block 7 — the numerals. The question you said you would act on

### First, a correction to the premise

`101 ... across 91 lines` is published in **one** document, not two:
`CHANGELOG.md:380` ("finds the version **101** times across 91 lines") and `:381`
("every one of those 101 is correct"). `RELEASE_NOTES.md` does **not** carry it — the
only `101` in that file is inside `1014 subtests`.

### How many published figures in these two documents are self-referential?

Self-referential = the corpus counted includes the documents doing the publishing.
**Four figures, at six print sites:**

| figure | sites | corpus | share of corpus that is these two docs |
|---|---|---|---|
| `101` version occurrences | `CHANGELOG.md:380`, `:381` | whole tracked tree | **32 of 101 = 31.7%** (CHANGELOG 19, RELEASE_NOTES 13) |
| `91` version-bearing lines | `CHANGELOG.md:380` | whole tracked tree | same 31.7% |
| `162` files read | `CHANGELOG.md:441`, `RELEASE_NOTES.md:26` | whole tracked tree | 2 of 162 = 1.2% |
| `163` `git ls-files` | `CHANGELOG.md:441`, `RELEASE_NOTES.md:26` | whole tracked tree | 2 of 163 = 1.2% |

Reproduced at this head: `git grep -o '0\.5\.0' | wc -l` -> **101**;
`git grep -n '0\.5\.0' | wc -l` -> **91**. Per-file: CHANGELOG.md 19, test_skill.sh 13,
RELEASE_NOTES.md 13, profiler-spec.md 10, README.md 10, forward-promise-check.sh 7,
profiler_test.go 7, then a long tail. The 10-count gap between 101 and 91 is 8 lines
carrying more than one occurrence.

Only `101`/`91` are *densely* self-referential. `162`/`163` technically are, but they
move only when a tracked file is added or removed — not on an editorial pass.

### Exactly what breaks if someone fixes a typo?

Driven, not reasoned:

- **A typo fix that touches no version string and no line break: nothing breaks.**
  101/91 unchanged.
- **A pure reflow — not one word changed — breaks `91`.** `RELEASE_NOTES.md:26` is a
  single **3,499-character** line carrying two occurrences of the version. Folding just
  that one line to 90 columns and recounting:
  `occurrences=101 lines=92` against the published `101 across 91`.
  The number is now false. Reflowing a 3,499-character paragraph is the single most
  likely editorial act anyone will perform on that file — and this very commit added a
  wrapped-line window *because* reflow is expected here.
- **Nothing in the repository detects it.** Exhaustive grep: `101`, `91`, `829` and
  `11913` are asserted nowhere in `tests/` or `.github/` (the only `101` hit in `tests/`
  is an unrelated timing comment at `test_install.sh:1973`). They are prose only. The
  forward-promise check would still exit 0; CI stays green; the CHANGELOG just carries
  a false number.
- **The numeral carries none of the sentence's weight.** The claim that matters is
  "**every one of those 101 is correct**", and that is entailed by "the check reports no
  findings" — which CI re-derives on every run. The census adds nothing to the argument
  it sits inside. The guard's own header says precisely this at `:57-59`: *"No count is
  written here on purpose: the last one this comment carried was wrong by about forty
  per cent... The argument does not need it."*

**That is the finding.** `forward-promise-check.sh` refuses to carry this number, for
this reason, in this commit. `CHANGELOG.md:383-385` then boasts that both places that
carried a variant "now publish **no numeral at all**" — and `CHANGELOG.md:380`, four
lines earlier, publishes one for the same quantity. The release states a rule and
breaks it in the adjacent paragraph.

### Does the same objection apply to `sites=829` / `lines=11913`?

**No on self-reference; partly on staleness, and the difference is the whole point.**

- Its corpus is the `source`-closure of the seven shell suites — **nine files, all
  under `tests/`** (`CHANGELOG.md:506-509`). Neither release document is in it. No edit
  to CHANGELOG.md or RELEASE_NOTES.md can move `829` or `11913`. The self-reference
  objection simply does not apply.
- It is unasserted prose, so it does go stale on any test edit. But it is published
  **as a command's output, with the command beside it**, and the release says so
  explicitly at `CHANGELOG.md:510-512`: "which is exactly why the figure is given with
  its command rather than carried forward." That makes it a **receipt for a measurement
  at a named head**, not a standing claim about the tree. A stale receipt is untidy; a
  false standing claim is a defect.
- `101 across 91 lines` is written as a standing claim — "`git grep -o` **finds** the
  version 101 times" — in the present tense, about the tree, in service of a ratio
  argument that needs an order of magnitude and not a census.

Same test applied to the rest: `3137` assertions, the coverage percentages and the
frozen-section md5s are either asserted by a control or are receipts-with-command over
corpora that exclude these documents. None of them is in this class.

### Recommendation — do this

**Delete the three numerals at `CHANGELOG.md:380-381` (`101`, `91`, `101`). Keep
`sites=829 files=9 lines=11913`. Keep `162`/`163`. Change nothing else.**

Replacement for `:380-381`, preserving the argument and the reproducibility:

> ...and the reason is the ratio: `git grep -o '0\.5\.0' | wc -l` against
> `git grep -n '0\.5\.0' | wc -l` finds the version on far more lines than there are
> promises, the check reports no findings, so **every one of them is correct** and a
> check on the string alone would fire on all of them.

Why this and not the alternatives:

- **Why not keep it?** It is a self-referential standing claim at 31.7% density, in the
  repository's most-read file, that a formatting-only edit makes false, with no control
  to catch it. That is the exact object the release's own rule was adopted against, and
  the rule was enforced one level down in this same commit by *deleting* two stale
  numerals rather than correcting them (`forward-promise-check.sh` header,
  `tests/test_skill.sh`). Consistency is not cosmetic here: the release's credibility
  on numbers is the thing it spent four lanes buying.
- **Why not assert it?** A control over `101` would be a control over a figure that
  carries no information, and it would redden on every legitimate reflow. You would be
  paying maintenance forever to protect a decoration.
- **Why keep the command?** Because the tradeoff of deleting is real and should be
  named: you lose a concrete, checkable, currently-true measurement a reader could use
  to judge the ratio claim themselves — which is the value the release correctly
  argues numbers-with-commands provide. Publishing the *command* without its *output*
  keeps the reader's ability to verify and removes the thing that goes stale. That is
  the move already made for the two comments; make it here.

**Severity: prose defect, not a gate.** It cannot redden CI and cannot ship a wrong
artifact. Fold it into the same pass as block 3 (which needs a code change anyway);
do not spend a standalone round on it.

---

## Coverage line

**Denominator built.** The guard's own declared closed vocabulary, taken from its
header at `:61-68` and its matcher at `:237-249`: 7 deferral verbs x 4 prepositions,
plus the four non-deferral shapes (`is <version>` / `is <version> work`,
`will ... in`, `needs ... in`, `new surface for/in`). Crossed against: 4 case
spellings; 15 leading markers; interior markup, punctuation, digits and hyphens;
markup on the version token; 6 version spellings (`0.5.0`, `v0.5.0`, and trailing
`.` `,` `;` `)`); non-single-space separators; 7 syntactic contexts (inline code,
link label, table cell, fenced block, HTML comment, blockquote, indented); 6
degenerate file shapes (CRLF, no trailing newline, zero-byte, heading-only,
blank-separated, whitespace-separated); record spreads of 2, 3 and 4; and both
execution paths (`--over` and tracked-tree).

**Cases driven: 349 single-line cases through the guard, plus 11 multi-record fixture
files, 7 tracked-tree mutations, 2 suite-level mutations, and 1 full suite run.**
Breakdown: block 1 = 172; block 2 = 51; block 3 = 10 x 2 versions = 20; block 6a = 7 +
1 fenced file; block 6b = 6 degenerate files + 12 CRLF/LF shape pairs; block 6c = 9
multi-record files; block 8 = 76. Baseline replay = 35.

**Second sweep, different cases, proving dryness (block 8).** The case/position axis
was re-driven with constructions that share nothing with block 1: the 7 verbs embedded
in 10 real-prose templates (sentence-medial, Go comment, `####` heading, struct-field
comment, appositive clause, task-list checkbox, nested numbered list, blockquote NOTE,
table cell, HTML comment), rotating case and preposition per case, and rotating the
version through `0.5.0` / `v0.5.0` / `0.5.0.` / `0.5.0,` / `0.5.0;` / `0.5.0)`; plus
all four non-deferral shapes in ALL CAPS and Title Case behind markers.
**76 cases, 76 FIRE, 0 misses.** The case fold is genuinely closed. Block 1 and block 8
share no input string.

**What is dry:** case, ALL CAPS, mixed case, every leading marker, verb position in the
record, version trailing punctuation, the `v` prefix, all seven declared verbs, all
four declared prepositions, the window's false-positive surface, the blank-line and
sentence-punctuation resets, zero-byte and no-trailing-newline handling, and both
count-accounting sides.

**What is not:** everything between the verb and the version (block 2), the folded
exemption (block 3), the heading probe (block 4), CRLF on one shape (block 6b).

## Files

- Notes: this file.
- Drivers and fixtures: `/private/tmp/claude-501/-Users-matthewvandusen-Development-Auraprix-skill-architect/31edaf48-6b19-4833-82e7-6cb52dac691f/scratchpad/pgdrive/`
  (`drive.sh`, `drive-base.sh`, `baseline.txt`, `block1.txt`, `block2.txt`,
  `block3.txt`, `block6.txt`, `block8.txt`, `d_*` degenerate fixtures, `fence.md`).
- Worktree: `/private/tmp/claude-501/-Users-matthewvandusen-Development-Auraprix-skill-architect/31edaf48-6b19-4833-82e7-6cb52dac691f/scratchpad/pg7/head` — left clean at `34976a2`.
