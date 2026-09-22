# F4 — the forward-promise guard's case-sensitive matcher. Fix record.

**Received head** `bf045ce` · **Delivered head** `34976a2` · **Branch** `release/0.5.0`,
pushed · **Worktree** `…/scratchpad/pgfix` (own, created fresh at `bf045ce` on
`fix/0.5.0-promise-guard-case`) · **One commit** · **Author** `imagineux
<imagineux@gmail.com>`, no `Co-Authored-By`, no "Generated with" line.

```
$ git log -1 --format='%an <%ae>'
imagineux <imagineux@gmail.com>
$ git log -1 --format='%B' | grep -inE '^Co-Authored-By|Generated with \[Claude|noreply@anthropic'
(no output)
```

**Not merged. Not tagged.** PR #22 re-confirmed **MERGEABLE / CLEAN**, CI run
`35690760694` **completed success** on `34976a2`.

**Scope** — `git diff --name-only bf045ce..34976a2`:

```
CHANGELOG.md
RELEASE_NOTES.md
tests/lib/forward-promise-check.sh
tests/test_skill.sh
```

4 files, +371 / −57. `README.md` and `docs/profiler-spec.md` were **checked and not
touched** (see "The prose" below). The guards lane's F6 work in `tests/test_skill.sh` was
**not undone** — see "What I did not undo".

---

## The defect, and the root cause

Confirmed REAL at `bf045ce` before changing anything. The reader's five shape tests matched
`$0` — the **raw** record — and every alternative in them was written lowercase
(`(tracked|deferred|carried|…)`, `[ a-z]*`, `will [a-z]+`, `needs? [a-z ]*`). So the closed
verb set the script's own header declares was, in the matcher, a set of **lowercase
spellings**. A capitalised member of that set is *inside* the declared set and was missed.

The root cause is not a missing alternative. It is that **the matcher asked about spellings
where the vocabulary was about verbs.** That is the same root as the five prior rounds this
release has already paid for — the guard-primitive reader and `name ( ) {`, the shadowing
check and `assert ( ) {`, the trap check and `trap cleanup 0`, the suite audit and the
libraries the suites `source`. Every one was closed by deriving or handling the grammar
instead of listing the forms, and this one is closed the same way.

A second, independent mechanism was in the same finding's blast radius and is closed with
it: the reader's unit of analysis was a **line** while the unit of meaning is a
**sentence**. The prose lane recorded that the changelog's own copy of this finding's
evidence survived only because a line break fell between the verb and the version. That is
a latent CI failure keyed to paragraph reflow on a file nobody edits.

---

## The fix — two mechanisms, no spellings added

The vocabulary is **byte-for-byte unchanged**. Nothing was added to any list.

1. **`rec = tolower($0)`**, once, at the single point the record reaches the shape tests,
   and the tests are left **unanchored**. Case-folding collapses capitalised, ALL CAPS and
   mixed case into one operation; leaving the tests unanchored means a leading `-`, `*`,
   `**`, indentation, a colon or an em dash before the verb are *positions*, and a position
   was never a shape. Eleven of the twelve members of the spelling class below need no rule
   of their own as a result.

   The exemption inside the `is <version>` shape moved from `/[Vv]ersion|…/` to `/version|…/`
   **on the folded record**, which widens it consistently to `VERSION` — the same intent,
   now case-complete rather than two-thirds enumerated.

2. **A two-record window.** Each record is tested joined to the one before it, which is the
   join a reflow performs; a finding visible only that way reports `across a line break` in
   its shape. Reported only when `shape_of(rec) == ""`, `prev != ""` **and**
   `shape_of(prev) == ""` — so a promise already reported on its own line is never counted
   twice. `prev` resets at a file boundary, at a section heading, at every skip path and
   (as a free consequence of the empty-string test) at a blank line: none of those is a
   wrap, and joining across a section heading in `CHANGELOG.md` would manufacture a finding
   across the frozen boundary.

The five shape tests also moved out of a growing `} else if` chain into one
`function shape_of(rec)` with early returns — required, since the window has to evaluate
the tests three times, and it removes the accretion smell the chain had become.

### Does the matcher handle the grammar, or still list forms?

**It handles the grammar, for case and for position.** Case is one `tolower`; position is
the absence of an anchor. Neither is a list, and adding a new capitalisation or a new
leading marker requires no change to the reader.

**It still lists verbs — deliberately, and that is not this defect.** The script's header
argues the verb set is worth having *because* it is the closed set this repository has
actually used, so a verb outside it is a new finding rather than a miss. I did not widen
it; widening it is a different question with a different false-positive budget, and nothing
in the evidence justified it.

**One residual is genuinely a list-shaped limit, now published rather than left to be
found:** the window is **two** records, so a promise spread over three lines with neither
the verb nor the version adjacent to the join is unseen. Extending to N records trades a
rarer catch against a rising false-positive rate across paragraph boundaries; two records
is what a reflow actually produces. Stated in the script header and in `CHANGELOG.md`.

---

## The spelling class, enumerated, with each verdict before and after

Driven through `--over` one-file mode by
`/private/tmp/…/scratchpad/class/drive.sh <worktree> 0.5.0` against (a) the reader exactly
as at `bf045ce` and (b) the delivered reader. `rc=1` is a refusal, `rc=0` an acceptance.

| # | spelling / position | literal driven | `bf045ce` | `34976a2` |
|---|---|---|---|---|
| L01 | **lowercase control** | `the caveat channel is deferred to 0.5.0.` | **rc=1 refuse** | **rc=1 refuse** |
| L02 | capitalised, sentence start | `Deferred to 0.5.0.` | rc=0 **MISS** | rc=1 refuse |
| L03 | ALL CAPS | `DEFERRED TO 0.5.0.` | rc=0 **MISS** | rc=1 refuse |
| L04 | mixed case | `DeFeRrEd To 0.5.0.` | rc=0 **MISS** | rc=1 refuse |
| L05 | leading bold marker | `**Deferred to 0.5.0**: a receiver subcommand.` | rc=0 **MISS** | rc=1 refuse |
| L06 | leading emphasis marker | `*Deferred to 0.5.0.*` | rc=0 **MISS** | rc=1 refuse |
| L07 | list marker + capitalised | `- Deferred to 0.5.0.` | rc=0 **MISS** | rc=1 refuse |
| L08 | list marker + bold lead-in | `- **Deferred to 0.5.0**: a receiver subcommand.` | rc=0 **MISS** | rc=1 refuse |
| L09 | leading whitespace (tab+spaces) | `\t  Deferred to 0.5.0.` | rc=0 **MISS** | rc=1 refuse |
| L10 | mid-sentence capitalised | `the receiver is Deferred to 0.5.0 for now.` | rc=0 **MISS** | rc=1 refuse |
| L11 | immediately after a colon | `Known limits: Deferred to 0.5.0.` | rc=0 **MISS** | rc=1 refuse |
| L12 | immediately after an em dash | `Known limits — Deferred to 0.5.0.` | rc=0 **MISS** | rc=1 refuse |
| L13 | capital `V` version prefix | `Deferred to V0.5.0.` | rc=0 **MISS** | rc=1 refuse |
| L14 | verb `Tracked` | `Tracked for 0.5.0.` | rc=0 **MISS** | rc=1 refuse |
| L15 | verb `Reserved` | `Reserved for 0.5.0.` | rc=0 **MISS** | rc=1 refuse |
| L16 | verb `Carried`, in a bold lead-in | `**Known limits, Carried to 0.5.0**` | rc=0 **MISS** | rc=1 refuse |
| L17 | verb `Scheduled`, ALL CAPS | `SCHEDULED FOR 0.5.0.` | rc=0 **MISS** | rc=1 refuse |
| L18 | verb `Planned`, list marker | `- Planned for 0.5.0, once the receiver lands.` | rc=0 **MISS** | rc=1 refuse |
| L19 | verb `Postponed` | `Postponed to 0.5.0.` | rc=0 **MISS** | rc=1 refuse |
| L20 | shape `is <version>`, capitalised | `Making it branchable Is 0.5.0 work` | rc=0 **MISS** | rc=1 refuse |
| L21 | shape `will … in <version>` | `Names it as the source it Will read in 0.5.0` | rc=0 **MISS** | rc=1 refuse |
| L22 | shape `needs … in <version>` | `Needs a receiver subcommand in 0.5.0` | rc=0 **MISS** | rc=1 refuse |
| L23 | shape `new surface for <version>` | `Giving probe an exit contract is New surface for 0.5.0` | rc=0 **MISS** | rc=1 refuse |

**22 of 23 members of the class passed silently at `bf045ce`; the lowercase control was the
only one caught. All 23 are refused now, and the lowercase control still fires.**

### The acceptances — the other side of the fold

Case-folding is only a fix if it was not bought with false positives. Nine acceptances held
identically before and after (`rc=0` in both columns), including the two that test the fold
itself:

| # | literal | `bf045ce` | `34976a2` |
|---|---|---|---|
| A01 | `// making it branchable is new surface, which 0.5.0 did not add` | rc=0 | rc=0 |
| A02 | `// Until 0.5.0 the only list of them was a switch in the CLI` | rc=0 | rc=0 |
| A03 | `- Every key 0.5.0 adds is optional, so a round trip is not vacuous` | rc=0 | rc=0 |
| A04 | `because this release *is* 0.5.0 and closes none of them` | rc=0 | rc=0 |
| A05 | `` - **`AdapterVersion` is `0.5.0`, and it is deliberately not held equal `` | rc=0 | rc=0 |
| A06 | `## 0.5.0 — 2026-09-21` | rc=0 | rc=0 |
| **A07** | `ADAPTERVERSION IS 0.5.0 AND IS NOT HELD EQUAL` — **the exemption, ALL CAPS** | rc=0 | rc=0 |
| A08 | `// 0.5.0 ships none of those adapters, so these fields are reserved` | rc=0 | rc=0 |
| A09 | `a stored 0.4.x profile against a fresh 0.5.0 one differs by four keys` | rc=0 | rc=0 |

A07 is the one that matters: it is why the exemption had to fold with the record rather than
stay `[Vv]ersion`. An intermediate build that folded the *shape tests* but left the
exemption half-enumerated was driven, and A07 went **rc=1 — a false positive** — so the
accept side caught the inconsistent half-fold. That intermediate was discarded.

---

## The wrapped-line case, specifically

This is the one the mandate flagged as mattering, and it did.

| # | records (verb / version split) | want | `bf045ce` | `34976a2` |
|---|---|---|---|---|
| W01 | `…skipped records is deferred` / `to 0.5.0, once the receiver lands.` | refuse | rc=0 **MISS** | rc=1 refuse |
| W02 | `**Known limits, Deferred` / `to 0.5.0**: a receiver subcommand.` | refuse | rc=0 **MISS** | rc=1 refuse |
| W03 | `…Cursor adapters Needs` / `a receiver subcommand in 0.5.0` | refuse | rc=0 **MISS** | rc=1 refuse |
| W04 | `two adapters were deferred.` / `To 0.5.0 we added a receiver instead.` | **accept** | rc=0 | rc=0 |
| W05 | `…was a switch in the CLI -` / `Until 0.5.0, that is.` | **accept** | rc=0 | rc=0 |

W04 is the control that the window did not over-fire: a sentence that *ends* in a deferral
verb followed by a sentence that *begins* with the version is not a promise, and
sentence-ending punctuation blocks the join on its own — no alternative in the vocabulary
admits `.` between the verb and its preposition. That falls out of the existing `[ a-z]*`
rather than needing a rule, which is the same "handle the grammar" property.

Double-reporting was driven explicitly over a 5-line document holding one same-line promise
and one wrapped promise:

```
dbl.md:1: [deferral verb] a one-line promise: Deferred to 0.5.0.
dbl.md:4: [deferral verb across a line break] the receiver is deferred to 0.5.0, once it lands.
files=1 lines=5 examined=5 skipped=0
rc=1
```

Two findings for two promises — the same-line one is not re-reported at line 2 as part of
the next window, and the wrapped one is reported once, at the line carrying the version.

---

## The controls

**Why the old ones could not reach it.** All 18 ran `--over` a file of **one line**, and all
18 were lowercase. Neither half of the class was held, and no one-line fixture can exhibit
a wrap at all. This is the "control that cannot reach the condition it tests" shape the
previous lane found twice — `tests/test_skill.sh` names it explicitly a hundred lines
further down, about a different counter.

**35 single-line cases** now (was 18): **24 refusals**, covering every row of the class
table above plus the four non-deferral shapes capitalised; **11 acceptances**, including
A07 and the same-line punctuation-blocking case. Derived, not asserted:

```
$ sed -n "/^done <<'PROMISE_CASES'$/,/^PROMISE_CASES$/p" tests/test_skill.sh | grep -cE '^(refuse|accept)\|'
35
$ … | grep -oE '^(refuse|accept)' | sort | uniq -c
  11 accept
  24 refuse
```

**And the class driven over a real multi-line document**, which is the mandate's
requirement. `$harness_scratch/forward-promise/release-notes-draft.md`, 27 lines, with the
texture the reader actually meets: an H1 and an H2, a bulleted list with a bolded lead-in,
two wrapped paragraphs, a tab-indented line, a fenced code block, blank lines throughout.
**Five promises planted** (lines 7, 9, 15, 21, 23 — bold lead-in, ALL CAPS, **wrapped**,
leading whitespace, after a colon) and **eight innocent lines naming the version** planted
beside them (3, 8, 11, 12, 18, 25, 27 and the fence).

The assertion is on the **set of line numbers the reader names**, not on exit status:

```
assert "over a real multi-line document the reader names exactly the planted promises, and no innocent line" \
  test "$promise_doc_found" = "$PROMISE_DOC_PLANTED"      # PROMISE_DOC_PLANTED='7 9 15 21 23'
```

Two-sided by construction: a reader blind to a planted shape returns a **smaller** set, a
reader firing on an innocent line returns a **larger** one, and either differs.
**An exit-status check would have passed on four of the five**, because lines 7/9/21/23
would still have reddened it at `bf045ce`— which is exactly the vacuity to avoid.

The denominator is **not derived from the reader**: `PROMISE_DOC_PLANTED` is a literal, and
`PROMISE_DOC_LINES=27` is `require`d against `grep -ac ''` on the document, so the literals
cannot come to mean different lines if the document is edited.

The wrapped case is asserted **by the shape the reader reports**, so a reader that reached
line 15 some other way would not satisfy it:

```
  the shape reported at line 15: deferral verb across a line break
PASS: the promise split across a line wrap is reported as having been found across the wrap
```

and driven in **both** inverse directions over the same document:

- wrap closed up (lines 14+15 joined by `awk`, length `require`d to be 26): still refused,
  and now reported as a plain `deferral verb` at line 14 — not attributed to a break.
- promise removed, wrap left in place: the reader names `7 9 21 23` — those two lines go
  quiet and the other four still fire. So what was refused was the promise, not the wrap.

---

## Non-vacuous: RED → GREEN → RED → GREEN

**RED, authoritative** — the reader restored byte-identically to `bf045ce`
(`git diff --stat` empty for that path) with the new controls in place:

```
105 passed, 21 failed
```

All 21 failures are the new controls, and **no acceptance is among them**:

```
FAIL: the forward-promise reader refuses: **Deferred to @V**: a receiver subcommand for the spool
FAIL: the forward-promise reader refuses: DEFERRED TO @V.
FAIL: the forward-promise reader refuses: DeFeRrEd To @V.
FAIL: the forward-promise reader refuses: - Tracked for @V, once the receiver lands
FAIL: the forward-promise reader refuses: 	  Reserved for @V.
FAIL: the forward-promise reader refuses: the receiver is Postponed to @V for now
FAIL: the forward-promise reader refuses: Known limits: Carried to @V
FAIL: the forward-promise reader refuses: Known limits — Scheduled for @V
FAIL: the forward-promise reader refuses: - **Planned for @V**
FAIL: the forward-promise reader refuses: *Deferred to @V.*
FAIL: the forward-promise reader refuses: Deferred to V@V.
FAIL: the forward-promise reader refuses: Making it branchable Is @V work
FAIL: the forward-promise reader refuses: Names it as the source it Will read in @V
FAIL: the forward-promise reader refuses: Needs a receiver subcommand in @V
FAIL: the forward-promise reader refuses: Giving probe an exit contract is New surface for @V
FAIL: over a real multi-line document the reader names exactly the planted promises, and no innocent line
FAIL: the multi-line document is refused outright, not merely annotated
FAIL: the promise split across a line wrap is reported as having been found across the wrap
FAIL: the same promise with the wrap closed up is still refused
FAIL: with the wrap closed up the finding is no longer attributed to a line break
FAIL: taking the wrapped promise out quiets those two lines and leaves the other four firing
```

**RED, per mechanism** — the case-fold kept and *only* the seven-line window removed:

```
FAIL: over a real multi-line document the reader names exactly the planted promises, and no innocent line
FAIL: the promise split across a line wrap is reported as having been found across the wrap
124 passed, 2 failed
```

Exactly the two wrapped-line assertions, and the entire case class stays green. **Each
mechanism has its own control, and neither is carrying the other.**

**GREEN, restored** (`md5 bda9af76ee1d33f2e9afde67b7c4d08d`): `126 passed, 0 failed`, and
the standalone driver reports `whole class: 37 as-specified, 0 mismatched`.

---

## The prose

### Changed

- **`CHANGELOG.md`** — the bullet that published `The match is **case-sensitive**` and its
  surrounding paragraph. Replaced with four labelled paragraphs saying what the reader
  **does**: the verbs-not-spellings distinction and the one `tolower`; the wrapped-line
  window with the `across a line break` marker and why it was live; the rebuilt controls
  including the multi-line document and the line-number-set assertion; and the residual
  two-record limit. The "it is not a check on the class" claim is gone — it is a check on
  the class for case and position now.
- **`RELEASE_NOTES.md:26`** — the sentence `the verb match is case-sensitive, so a lowercase
  … is caught and the same phrase capitalised … is not`, and `it is not a check on the
  class`. Rewritten to the same substance in that document's single-paragraph register.
- Both keep **describing the caught shape without spelling it**, and both now say that the
  guard fired on its author. It did: writing the fix's own header comment tripped it —

  ```
  tests/lib/forward-promise-check.sh:74: [deferral verb] # `Deferred to 0.5.0` — because it tested the raw record, so every alternative
  rc=1
  ```

  The header was reworded to describe the shape and point at the fixtures, which are the
  right home for the literals. **The guard catching its own author is the guard working**,
  and it is now the third round in which it has done so.
- **`tests/lib/forward-promise-check.sh`** header — a new section, *"A vocabulary is a set
  of verbs, not a set of spellings"*, plus the wrapped-line rationale and the two stated
  residuals. The stale `about seventy … roughly sixty` ratio (F10, left as a code-side
  residual by the prose lane) is **dropped, not corrected** — the release's own rule for a
  numeral that nothing re-derives.
- **`tests/test_skill.sh`** — the same stale ratio in a `69 times … about sixty` variant
  (also F10's residual): dropped. The control-block comment explaining the case class. And
  the guards lane's downstream comment that said *"The 18 fixture controls … all 18 run in
  `--over` single-file mode over one-line fixtures"*, which my change made false: corrected
  to 35, with the multi-line document noted as reaching a wrap rather than a scan limit.

### Checked and deliberately not changed

- **`README.md`** — carries **no** claim about this guard.
  `grep -inE 'case-sensitiv|lowercase|capitalis|spelling'` returns two hits, `:860` and
  `:896`, both about Codex source-format spellings and a `cp -R` idiom. Not touched.
- **`docs/profiler-spec.md`** — likewise none. Its `:860` and `:890` hits are about the
  documented spelling of OTLP `event.name`. Not touched.
- `RELEASE_NOTES.md:22` and `CHANGELOG.md:681-686` discuss case-sensitivity of
  `ubuntu-latest` and APFS for the **containment barrier** — a different subsystem, correct
  as written, not touched.

### Frozen sections — byte-identical to `v0.4.3`, verified after every edit

```
$ sed -n '/^## 0\.4\.3/,$p' CHANGELOG.md | wc -l ; … | md5 -q
240   ef2fe48af306b4ce06ace48391eddcb8
$ git show v0.4.3:CHANGELOG.md | sed -n '/^## 0\.4\.3/,$p' | wc -l ; … | md5 -q
240   ef2fe48af306b4ce06ace48391eddcb8

$ sed -n '/^## v0\.4\.3/,$p' RELEASE_NOTES.md | wc -l ; … | md5 -q
112   46d66b1e331726dbe56a830d449c19aa
$ git show v0.4.3:RELEASE_NOTES.md | sed -n '/^## v0\.4\.3/,$p' | wc -l ; … | md5 -q
112   46d66b1e331726dbe56a830d449c19aa
```

---

## Every number, with its command

| figure | command | value |
|---|---|---|
| shell assertions, **bash 3.2.57** | `for s in tests/test_*.sh; do /bin/bash "$s" \| tail -1; done` | **3137**, 0 failed |
| shell assertions, **bash 5.3.15** | same with `/opt/homebrew/bin/bash` | **3137**, 0 failed, identical per-suite |
| — per suite, both shells | as above | `test_f01` 1912 · `test_f02` 350 · `test_harness` 276 · `test_install` 52 · `test_rewrite` 401 · **`test_skill` 126** · `test_walk` 20 |
| Go tests | `go test -list '.*'` per package, `grep -cE '^(Test\|Example)'` | **286** = 239 / 35 / 12 — **unchanged** |
| Go subtests | `go test -v` per package, `grep -cE '^\s+--- (PASS\|SKIP\|FAIL): '` | **1014** = 579 / 87 / 348 — **unchanged** |
| coverage | `go test -cover ./...` | **97.9 / 94.7 / 95.9** — unchanged |
| build / vet / gofmt | `go build ./... && go vet ./... && gofmt -l .` | clean, `gofmt -l` empty |
| assertion audit | `bash tests/lib/audit-suites.sh tests/test_*.sh \| tail -1` | `sites=829 files=9 lines=11913 accounted=11913` |
| promise reader, tree | `bash tests/lib/forward-promise-check.sh 0.5.0` | `files=162 lines=43851 examined=43486 skipped=365`, **rc=0** |
| version occurrences | `git grep -o '0\.5\.0' -- . \| wc -l` | **101** across **91** lines (`git grep -c`), check clean ⇒ all 101 correct |
| promise case table | `sed -n "/^done <<'PROMISE_CASES'$/,/^PROMISE_CASES$/p" … \| grep -cE '^(refuse\|accept)\|'` | **35** (24 refuse / 11 accept) |
| `skill-validator` | `skill-validator check skills/{skill-audit,skill-rewrite}` | both `Result: passed`, rc=0, **zero errors, zero warnings** |
| `skillscore` | `skillscore --version` | **2.0.2**, global install intact, no PATH mirror built |

### Deltas, and why each is exactly what it should be

- **3112 → 3137 (+25)**, all in `test_skill` (**101 → 126**). Accounted for exactly: **17**
  new rows in the `PROMISE_CASES` table (driven through the loop's two pre-existing `assert`
  call sites) **+ 8** new `assert`/`require` call sites in the multi-line-document block.
- **`sites` 821 → 829 (+8)** — the same 8 new call sites, counted by a different program
  (`audit-suites.sh` reads call sites from source; the suite total counts assertions at
  runtime). The two sides agreeing on 8 is a real cross-check, not a restatement.
- **`lines` 11749 → 11913 (+164)** — `tests/test_skill.sh` grew by that many records.
- Go figures **unchanged**, which is the expected result and is the evidence that no Go file
  was touched. They match the mandate's stated `bf045ce` values exactly.

### Published counts updated

- `CHANGELOG.md:473-475` — `3112` → `3137`, `test_skill` `101` → `126`.
- `RELEASE_NOTES.md:21` — `3112` → `3137`.
- `CHANGELOG.md:508` — `sites=821 … lines=11749 accounted=11749` → `sites=829 …
  lines=11913 accounted=11913`.
- `CHANGELOG.md:380-381` — `99` times across `88` lines → **`101` across `91`**.

Historical mentions of `18` cases, `761/9397` and `sites=767` were **left alone**: they are
explicitly narrated as what an *earlier draft or an earlier commit* measured, and are
correct as history.

---

## What I did not undo

The guards lane's F6 change — the promise reader's record count having an independently
derived other side — is **intact and still driven**. `lines` is still counted by
`{ lines++ }` before any rule can skip a record; the reader still refuses
`examined + skipped != lines` on its own terms; `tests/test_skill.sh` still holds `lines`
against `grep -ac ''` and the skip set against the script's own `HISTORY_DOCS` and
`EXEMPT_FILE`; and their two limit-injected reader copies still run over the real tracked
tree. All of it passes:

```
PASS: the forward-promise reader read every record in those files, not a prefix of each
PASS: every record the reader read, it either looked at or said it was skipping
PASS: the reader skipped records only in the files its own scope names, and in all of them
PASS: a reader that stops reading at line 400 is refused by the record count
PASS: a walk that stops at line 400 while the reader goes on is refused by the accounting
PASS: a walk that stops is refused by the reader on its own terms, without a caller
PASS: the same copy without the rule reads the whole tree, so the copy is not what was refused
```

My window additions were placed so as not to disturb that accounting: `prev`/`prev_raw` are
assigned on the skip paths *alongside* the existing counters, never instead of them, and
the window is evaluated **after** `examined++`. `examined + skipped == lines` holds
throughout, on both shells.

The only edit inside their text is the stale `18` in a comment their change did not concern
— corrected, not reverted.

---

## The two shapes I was told to watch for

**1. A control that cannot reach the condition it tests.** Checked, and the check is the
per-mechanism revert above: removing only the window reddens exactly the two wrapped
assertions and nothing else, and restoring `bf045ce`'s reader reddens all 21. A control
that could not reach its condition would have stayed green through those. The multi-line
document's `require` on its own length is the guard against its literals drifting off the
lines they were written for.

**2. A denominator derived from the thing it checks.** One found, pre-existing, and I am
reporting it rather than claiming I closed it — see below. I introduced none: the
multi-line document's expected set is a literal, and the `sites`/suite-total agreement on
+8 comes from two different programs.

---

## For the maintainer, before tagging

1. **`tests/test_skill.sh:821` (`PROMISE_CASES_EXPECTED`) is still a mild both-sides-shrink.**
   `promise_n` and the literal `35` both describe the same heredoc table, so deleting rows
   from the table and editing the literal moves both together. It does catch the thing it
   was written for — a `read` loop dying partway — but it is not a second reading of the
   artifact. Closing it properly means counting the table from the file text
   (`grep -cE '^(refuse|accept)\|'` on the heredoc span) as an independent side, the same
   treatment `sites` got. **I did not change it: it is F5/F6's class, not F4's, it was not
   routed to me, and the honest fix touches a line the guards lane owns.** Same shape, same
   file, noted at `:815` in the mandate — still there.

2. **`CHANGELOG.md`'s `101 across 91 lines` is self-referential.** It counts occurrences of
   the version string in a tree that includes the document publishing the count, so editing
   that paragraph moves the number. It is true at `34976a2` and I verified it is stable
   (correcting a numeral adds no occurrence of the version string), but **any further prose
   edit to either release document invalidates it**, and it will need re-deriving if you
   touch them before tagging. The same is true of `sites=829 … lines=11913`. Both are the
   F5/F10 class structurally; I kept them because the changelog's argument leans on the
   ratio, but dropping them the way the two code comments were dropped would be defensible.

3. **An apostrophe in any comment inside the reader's awk program breaks the script.** The
   awk body is a single-quoted shell word, so one `'` ends it and bash reports a syntax
   error tens of lines below. I hit this while writing the fix — `rc=2` on every invocation,
   `syntax error near unexpected token 'prev'`. It fails **loudly and closed** (rc=2, and
   `tests/test_skill.sh` runs the reader), so it cannot pass silently, but the diagnostic
   points nowhere near the cause. I added a comment in the awk body recording the
   constraint. A `bash -n` step in CI would name it directly; there is none today.

4. **The section-heading probe is an exact-match scope, and a mismatch fails open-ish.**
   `sub(/^## v?/, …)` reads a literal heading. If `CHANGELOG.md`'s or `RELEASE_NOTES.md`'s
   heading for the release being cut were ever written differently, `in_section` would never
   become true, that file's whole section would leave scope, and the reader would exit 0
   with its accounting perfectly self-consistent. What stands behind it today is the skip-set
   comparison against `HISTORY_DOCS`, which proves the reader skipped *in* those files, not
   that it examined the current section. **There is no control asserting that the release
   being cut's own section was examined at all.** It is correct at this head (`scope=247` and
   `scope=115` are the sections *above* the current one). I stated it in the script header
   as a known limit rather than silently adding a control outside F4's mandate, but it is the
   nearest remaining thing to this release's recurring defect class, and it is where I would
   look next.

5. **No PR threads exist.**
   `gh api graphql … pullRequest(number:22){reviewThreads}` → `totalCount: 0`. There was
   nothing to reply to and nothing to resolve, so I resolved none. The external reviewer has
   posted no inline threads on #22.

6. **`.out-of-scope.md` / `deferred-ledger.md` not re-audited.** F4's fix widens what the
   guard catches, and the guard is clean over the tracked tree — but the ledger is
   untracked, so a capitalised marker in it was never in scope before or now. Unchanged by
   this work; flagging only so it is not assumed covered.

---

## Environment and isolation

- All work in `…/scratchpad/pgfix`, my own worktree, created fresh at `bf045ce`. `pwd`
  confirmed inside it before every write to the tree.
- **The primary tree at `/Users/matthewvandusen/Development/Auraprix/skill-architect` was not
  touched** except this record file, which is in `.scuba/` — `git ls-files | grep -c
  '^\.scuba'` → `0`, `.gitignore:3` is `.scuba/`. It is still at `0a83615` with its ~2,900
  uncommitted lines.
- No `checkout`, `reset`, `clean` or `stash` in any tree but my own. The stale worktrees at
  `…/gfin`, `…/lane2b`, `…/guards-lane` and the rest were not touched.
- Every reader injection was made from a copy taken first (`reader.fixed`, md5
  `bda9af76ee1d33f2e9afde67b7c4d08d`) and restored by md5; the tree was confirmed clean
  after each.
- This file was written with the **Write tool**, by absolute path, not a heredoc.
- Not merged. Not tagged. `v0.5.0` is the maintainer's to push.
