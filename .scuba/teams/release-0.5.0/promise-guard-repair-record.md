# Promise-guard repair — four items off `confirm-promise-guard.md`. Record.

**Received head** `34976a2` (`origin/release/0.5.0`) · **Branch**
`fix/0.5.0-promise-guard-repair`, pushed to `release/0.5.0` · **Worktree**
`…/scratchpad/pgrepair`, my own, created fresh at `34976a2` · **Author** `imagineux`,
no `Co-Authored-By`, no "Generated with" line.

**Not merged. Not tagged.** `v0.5.0` is the maintainer's.

Drivers: `…/scratchpad/pgdrive2/drive.sh` — one line at a time through
`<reader> --over <fixture> 0.5.0`, printing FIRE (rc=1) / PASS (rc=0).
Reader copies under test: `reader.base.sh` (`git show bf045ce:…`, md5
`2054197e8580e3098de583296ce0643f`), `reader.head.sh` (`34976a2`, md5
`bda9af76ee1d33f2e9afde67b7c4d08d`), `reader.fix.sh` (delivered).

The primary tree at `/Users/matthewvandusen/Development/Auraprix/skill-architect` was
never touched except this file, which is in the gitignored `.scuba/` control plane.
`scratchpad/pgfix` was never entered. No `checkout`/`reset`/`clean`/`stash` outside
my own worktree.

**On commit order.** The role default is a red repro commit with the fix on top. I did
not do that here and am naming it rather than leaving it implicit: this branch is one
push away from a tag and its CI run gates it, so a deliberately-red commit on it would
hand the maintainer a broken revision to tag from. Each commit is a logical unit that
is green on its own, and the RED evidence is recorded verbatim below instead — driven,
reproducible from the copies named above.

---

## Item 1 — the fix-introduced regression. REAL, confirmed, repaired.

### Confirmed against the head before anything changed

The report's 10 rows, driven through both readers, unmodified:

```
=== BASE bf045ce ===
FIRE | # VERSION MATRIX: building the capability is 0.5.0
FIRE | # VERSION: making it branchable is 0.5.0 work
FIRE | # THE VERSION TABLE says building the receiver is 0.5.0
FIRE | # VeRsIoN note: building the capability is 0.5.0
FIRE | # RELEASE IS 0.5.0 and building the receiver is 0.5.0
PASS | the version matrix says building the capability is 0.5.0
PASS | # Version note: building the capability is 0.5.0
PASS | // bump the version string; making it branchable is 0.5.0 work
FIRE | # building the capability is 0.5.0
PASS | this release is 0.5.0
driven=10 fire=6 pass=4

=== HEAD 34976a2 ===
PASS | # VERSION MATRIX: building the capability is 0.5.0
PASS | # VERSION: making it branchable is 0.5.0 work
PASS | # THE VERSION TABLE says building the receiver is 0.5.0
PASS | # VeRsIoN note: building the capability is 0.5.0
PASS | # RELEASE IS 0.5.0 and building the receiver is 0.5.0
PASS | the version matrix says building the capability is 0.5.0
PASS | # Version note: building the capability is 0.5.0
PASS | // bump the version string; making it branchable is 0.5.0 work
FIRE | # building the capability is 0.5.0
PASS | this release is 0.5.0
driven=10 fire=1 pass=9
```

Five FIRE → PASS. The finding is REAL exactly as described.

### One thing the report did not reach: the accept fixture is vacuous, not merely wrong

The report says the suite "positively asserts the widened behaviour" through
`accept|ADAPTERVERSION IS @V AND IS NOT HELD EQUAL`. Driven, it is worse than that —
that fixture asserts **nothing at all** about the exemption. Against a reader with the
exemption **deleted outright**:

```
--- the shipped accept fixture, against a reader with NO exemption at all ---
PASS | ADAPTERVERSION IS 0.5.0 AND IS NOT HELD EQUAL
driven=1 fire=0 pass=1
```

Mechanism: the shape is `is +`?v?<v>`?( work|\.|,|$)`. The words after the version put
` and` where the right-hand alternation requires end-of-record, ` work`, `.` or `,`, so
the record never matches the shape and never reaches the exemption. It cannot tell a
working exemption from a missing one. That is why the previous round could fold the
exemption and see the whole accept side stay green — the control for that boundary was
not a control.

(It also means the previous record's claim that an intermediate half-folded build made
this case fire is not reproducible: with no exemption whatsoever it still passes.)

### Root cause

Not "the fold was applied one place too many". The exemption asked the wrong question,
and the fold only made the wrong question louder:

- It was a **substring test over the whole record** (`rec !~ /version|…/`) standing in
  for a **relation** — whether the subject beside *this* `is` names a version surface.
- Because it was per-record, one exempt clause exempted every other occurrence of the
  shape on the same line. `# RELEASE IS 0.5.0 and building the receiver is 0.5.0`
  carries one of each, and no per-record exemption can refuse it while accepting
  `this release is 0.5.0`. The fifth regression row is therefore unrestorable by any
  amount of respelling the exemption — it forces the per-occurrence repair.
- Because the discrimination it did have was carried entirely by **case**
  (`[Vv]ersion` against a raw record), folding the record dissolved it. Restoring
  `[Vv]ersion` and adding `VERSION` would put the release back on the
  enumerate-the-spellings defect it spent this commit removing, one level down.

### The repair

`tests/lib/forward-promise-check.sh` — the bare `is <version>` shape is decided once
per **occurrence** by `is_assignment(rec)`, walking the record with `match()` and asking
each occurrence about its own subject:

```awk
    function is_assignment(rec,   tail, subject) {
      tail = rec
      while (match(tail, "is +`?v?" v "`?( work|\\.|,|$)")) {
        subject = substr(tail, 1, RSTART - 1)
        if (subject !~ /(version|release)[^a-z0-9]* $/)
          return 1
        tail = substr(tail, RSTART + RLENGTH)
      }
      return 0
    }
```

The subject test is a **position** — the word touching the `is`, through markup — in the
same way the lead-in markers are positions, so it folds with the record and stays a set
of subjects rather than becoming a set of spellings. Nothing was added to any
vocabulary; the shape pattern itself is byte-identical to head.

### After the fix

```
=== block3, FIXED ===
FIRE | # VERSION MATRIX: building the capability is 0.5.0
FIRE | # VERSION: making it branchable is 0.5.0 work
FIRE | # THE VERSION TABLE says building the receiver is 0.5.0
FIRE | # VeRsIoN note: building the capability is 0.5.0
FIRE | # RELEASE IS 0.5.0 and building the receiver is 0.5.0
FIRE | the version matrix says building the capability is 0.5.0
FIRE | # Version note: building the capability is 0.5.0
FIRE | // bump the version string; making it branchable is 0.5.0 work
FIRE | # building the capability is 0.5.0
PASS | this release is 0.5.0
driven=10 fire=9 pass=1
```

All five regression rows fire again. Rows 6-8 — the **pre-existing, disclosed**
general over-breadth, PASS at both `bf045ce` and `34976a2` — now fire too: they are
genuine assignments and the adjacency question is the one that separates them. Row 10
is the true exemption and still passes.

The accept side, 15 cases including every accept fixture in the suite, all PASS after
the fix (`driven=15 fire=0 pass=15`), against `fire=5` for the same 15 with the
exemption deleted — so the accepts are now load-bearing rather than incidental.

### The fixtures, re-derived

`tests/test_skill.sh`, `PROMISE_CASES`: 35 → **42** (24 → 30 refuse, 11 → 12 accept).

- `accept|ADAPTERVERSION IS @V AND IS NOT HELD EQUAL` → `accept|ADAPTERVERSION IS @V`.
  Cut back to the clause that actually meets the shape, so it fails against a reader
  with no exemption. This is the accept case the mandate named.
- `accept|this release is @V` added — the second subject the exemption recognises, also
  non-vacuous (it fires with the exemption removed).
- Six refuses added: the five regression rows, plus the same construction in **lower
  case** (`the version matrix says building the capability is @V`) so the control pins
  *adjacency* and not case — a reader that merely un-folded the exemption would pass the
  five and fail this one.

### Non-vacuous: RED → GREEN, both shells

RED — the new controls in place, reader restored byte-identically to `34976a2`
(`git diff --stat` empty for that path):

```
FAIL: the forward-promise reader refuses: # VERSION MATRIX: building the capability is @V
FAIL: the forward-promise reader refuses: # VERSION: making it branchable is @V work
FAIL: the forward-promise reader refuses: # THE VERSION TABLE says building the receiver is @V
FAIL: the forward-promise reader refuses: # VeRsIoN note: building the capability is @V
FAIL: the forward-promise reader refuses: the version matrix says building the capability is @V
FAIL: the forward-promise reader refuses: # RELEASE IS @V and building the receiver is @V
127 passed, 6 failed
```

Exactly the six new refuses, no accept among them.

GREEN — the fix restored:

```
=== GREEN: /bin/bash ===
133 passed, 0 failed
=== GREEN: bash 5 ===
133 passed, 0 failed
```

And over the real tracked tree, both shells:

```
$ bash tests/lib/forward-promise-check.sh 0.5.0
skipped-in=CHANGELOG.md scope=247 exempt=0
skipped-in=RELEASE_NOTES.md scope=115 exempt=0
skipped-in=profiler/profiler_test.go scope=0 exempt=3
files=162 lines=43882 examined=43517 skipped=365
rc=0
```

Commit `48d44c4`.

---

## Item 4 — the heading probe. REAL, worse than published, repaired.

Taken before item 2 because it is the same root as the first false claim: the
justification cited a control for this residual, and writing the control is what makes
the sentence true.

### Confirmed on the real tree, mutating only `CHANGELOG.md:7`

```
--- heading spellings, reader over the real tracked tree --- (34976a2 reader)
## 0.5.0 — 2026-09-21            rc=0  skipped-in=CHANGELOG.md scope=247 exempt=0
## [0.5.0] — 2026-09-21          rc=0  skipped-in=CHANGELOG.md scope=1078 exempt=0
## V0.5.0 — 2026-09-21           rc=0  skipped-in=CHANGELOG.md scope=1078 exempt=0
##  0.5.0 — 2026-09-21           rc=0  skipped-in=CHANGELOG.md scope=1078 exempt=0
## Release 0.5.0 — 2026-09-21    rc=0  skipped-in=CHANGELOG.md scope=1078 exempt=0
### 0.5.0 — 2026-09-21           rc=0  skipped-in=CHANGELOG.md scope=1078 exempt=0
## 0.5.0 - 2026-09-21            rc=0  skipped-in=CHANGELOG.md scope=247 exempt=0
```

`scope=1078` is the whole file leaving the walk, with rc=0. End to end, an assignment
planted at `CHANGELOG.md:10` inside the section:

```
=== BEFORE (34976a2 reader) ===
planted promise, heading untouched:
  rc=1
  CHANGELOG.md:10: [deferral verb] - A receiver subcommand for the spool is deferred to 0.5.0 and will land later.
  files=162 lines=43978 examined=43613 skipped=365
same promise, heading as `## [0.5.0] — 2026-09-21`:
  rc=0
  files=162 lines=43978 examined=42781 skipped=1197
```

rc=0, no finding, accounting perfectly self-consistent (42781 + 1197 = 43978).

### The repair, and why it is not a list of heading spellings

Two moves, and only the first is tolerance:

1. The probe folds case, like every other test in the file. It read the raw record, so
   `## V0.5.0` meant no section — a case-sensitivity bug inside the case-sensitivity fix.
2. **A scoped file that never entered its section is a finding.** `in_section` starts
   false in a scoped file, so a heading written any other way is not a narrower scope but
   *no* scope. Tolerating five spellings leaves the sixth; what is asserted instead is the
   thing that has to be true. New END rule:

```awk
      for (f in sectioned_files) {
        if (!(f in saw_section)) {
          printf "%s: no heading for the release being cut was found, so none of this file was read: it is scoped to a section that is not there\n", f
          found++
        }
      }
```

### After

```
--- heading spellings, reader over the real tracked tree --- (repaired)
## 0.5.0 — 2026-09-21            rc=0  skipped-in=CHANGELOG.md scope=247 exempt=0
## [0.5.0] — 2026-09-21          rc=1  CHANGELOG.md: no heading for the release being cut was found…
## V0.5.0 — 2026-09-21           rc=0  skipped-in=CHANGELOG.md scope=247 exempt=0
##  0.5.0 — 2026-09-21           rc=1  CHANGELOG.md: no heading for the release being cut was found…
## Release 0.5.0 — 2026-09-21    rc=1  CHANGELOG.md: no heading for the release being cut was found…
### 0.5.0 — 2026-09-21           rc=1  CHANGELOG.md: no heading for the release being cut was found…
## 0.5.0 - 2026-09-21            rc=0  skipped-in=CHANGELOG.md scope=247 exempt=0
```

`## V0.5.0` is now **read** (scope=247, the sections above it) rather than refused — it is
a spelling of the heading, not a different heading. The other four fail closed by name.
End to end, the same planted assignment under `## [0.5.0]`:

```
=== AFTER (repaired reader) ===
same promise, heading as `## [0.5.0] — 2026-09-21`:
  rc=1
  CHANGELOG.md: no heading for the release being cut was found, so none of this file was read: it is scoped to a section that is not there
  files=162 lines=43978 examined=42781 skipped=1197
```

### The controls, which did not exist in any form

Every fixture case runs `--over`, which applies neither scoping nor the exemption — which
is structurally why *both* of the reader halves that decide what it reads at all shipped
wrong. The new block builds a scratch git repository with the two history documents, a
0.4.3 section carrying legitimate history, and the file the exemption names, and drives
the reader from inside it. Eight assertions, two of them positive so the refusals cannot
be a reader that refuses everything.

RED — the probe repair reverted, the item-1 repair kept:

```
FAIL: a bracketed heading does not silently take the document out of scope
FAIL: a heading with a word in front of the version does not silently take the document out of scope
FAIL: a deeper heading level does not silently take the document out of scope
FAIL: a doubled space after the marker does not silently take the document out of scope
FAIL: an unreadable heading is refused even when the section holds nothing to find
FAIL: a capital V in the heading is read as the section, not refused as a missing one
135 passed, 6 failed
```

The two positive controls stay green in both directions, which is what proves the fixture
repository is not simply always-red.

GREEN:

```
PASS: with the heading it knows, the scoped reader reads the section and names the assignment in it
PASS: with the heading it knows and nothing to find, the scoped reader is quiet
PASS: a bracketed heading does not silently take the document out of scope
PASS: a heading with a word in front of the version does not silently take the document out of scope
PASS: a deeper heading level does not silently take the document out of scope
PASS: a doubled space after the marker does not silently take the document out of scope
PASS: an unreadable heading is refused even when the section holds nothing to find
PASS: a capital V in the heading is read as the section, not refused as a missing one

141 passed, 0 failed
```

Commit `fc9412b`.

---

## Item 2 — the two false claims

### 2a. `forward-promise-check.sh:131-133` — the control that did not exist

Verified before acting. `tests/test_skill.sh:1070-1077` (now `:1095-1104`) pipes the
reader through `sed -n 's/^skipped-in=\([^ ]*\).*/\1/p' | sort -u` and compares a set of
**file names**. Exhaustive grep at the received head: `scope=` and `exempt=` appear
nowhere under `tests/` outside the reader itself. The claim was false, and under the 4b
mutation the reported name set is unchanged while the skip count goes from 247 to 1078 —
so the cited control passes through the failure it was cited as catching.

**Resolution: the control is written, so the justification becomes true** — but not the
one that was claimed. What justifies the residual now is that there is no longer a
silent residual: the sentence says the section has to have been entered and that a miss
is a finding, and eight controls drive it. The false sentence is gone from the header.

(The header at `:44-47` was checked and **left**: it claims the suite holds the *set of
files* with a nonzero skip against `HISTORY_DOCS` and `EXEMPT_FILE`, which is exactly
what that assertion does. The report labelled it as making "the same claim"; driven, it
does not — only `:131-133` claimed counts.)

### 2b. `CHANGELOG.md:383-385` — "now publish no numeral at all"

True of the two places it names, false four lines from a third that published one for the
same quantity. Both halves are dealt with under item 3: the numeral at `:380-381` is gone,
and the parenthetical now says so instead of boasting past it.

---

## Item 3 — the self-referential numerals

Re-derived at the received head and then at mine:

```
$ git grep -o '0\.5\.0' -- . | wc -l     # 34976a2: 101      mine: 98
$ git grep -n '0\.5\.0' -- . | wc -l     # 34976a2:  91      mine: 89
$ git grep -o '0\.5\.0' -- CHANGELOG.md RELEASE_NOTES.md | wc -l   # 31 of them
```

**The published `101 across 91` was already false by the time I reached it.** Three
occurrences and two lines left the tree through this round's prose edits alone — rewriting
a sentence that happened to contain the version string. That is the report's reflow
argument, demonstrated live rather than constructed: no control anywhere re-derives it,
CI stays green, and the changelog carries a false number.

Deleted: the three numerals in that pair of sentences. **Kept:** both commands, so a
reader can take the measurement; `162`/`163`; `sites=`/`lines=` (corpus is the nine-file
`source` closure under `tests/` — neither document is in it, so no prose edit can move
them); `3137`→`3152`; coverage; the md5s. The parenthetical that boasted about the two
comments now records that this sentence carried one too, and why it went.

Commit `ac0a16c`.

---

## Finding 1 — the scope call. I agree, and here is the evidence I took it on.

**I did not broaden the matcher. I corrected the claim, in all three places, and recorded
the gap as 0.6.0 surface beside the `macos-latest` job.** Commit `fec81b1`.

Confirmed at my own head first, rather than taken from the report:

```
FIRE | - **Deferred to 0.5.0**: a receiver subcommand for the spool
PASS | - **Deferred** to 0.5.0: a receiver subcommand for the spool
PASS | `deferred` to 0.5.0                     PASS | deferred to `0.5.0`
PASS | deferred to [0.5.0](CHANGELOG.md)       PASS | deferred to "0.5.0"
PASS | deferred to  0.5.0   (two spaces)       PASS | deferred to<TAB>0.5.0
PASS | Deferred, to 0.5.0                      PASS | tracked in phase 2 for 0.5.0
PASS | will be re-scoped in 0.5.0              PASS | needs 2 adapters in 0.5.0
PASS | new surface for **0.5.0**               PASS | is **0.5.0** work
FIRE | deferred to 0.5.0.   FIRE | deferred to v0.5.0   FIRE | deferred  to 0.5.0
driven=23 fire=4 pass=19
```

Markup *around* the span is handled; markup *inside* it is not; the defect is strictly on
the left of the version. Pre-existing, byte-identical interiors at `bf045ce`.

**Why I agree with correcting the claim rather than broadening now**, in order of weight:

1. **It is pre-existing and the release neither opened nor closed it.** What was actually
   wrong — and what a reader is harmed by — is prose that reads as though the class were
   closed. That is repairable at a release gate; a matcher widening is not.
2. **The coupling is real, and I measured it rather than repeating it.** A crude widening
   of the last three shapes (`[^0-9]*` for the interior spans) run over the tracked tree
   produces two findings, one of them `RELEASE_NOTES.md:26` — the sentence describing this
   very reader — and one in `README.md` across a line break. A widening is therefore a
   false-positive budget to be spent deliberately, with both documents re-driven, not a
   regex to slip in.
3. **A tightening at a tag gate has no room to be wrong.** The last two rounds each shipped
   a defect *inside* the repair; this round found one of them.

**The nuance the maintainer should have, because it cuts the other way.** The same
experiment run on the **deferral shape alone** — the one that covers the four
constructions a writer will actually hit — catches all of them and produces **zero** new
findings on this tree:

```
--- the broadened deferral matcher over the tracked tree ---
0 findings
--- and the constructions it now catches ---
FIRE | - **Deferred** to 0.5.0: …   FIRE | deferred to `0.5.0`
FIRE | deferred to [0.5.0](CHANGELOG.md)   FIRE | deferred to<TAB>0.5.0
```

So the stated reason for deferring — "it would start firing on both documents' own
descriptions of the guard" — **did not reproduce for the deferral shape at this head**. It
reproduced for the other three. That is a narrower and more tractable 0.6.0 change than
the finding implies, and it is the first thing whoever picks this up should re-drive. I am
reporting that rather than acting on it: it is a scope decision, and it was made.

One claim inside the new prose, checked rather than asserted — that emphasis closing
mid-sentence is this repository's actual lead-in style, which is what makes the gap a
live authoring hazard rather than a curiosity:

```
$ grep -cE '^- \*\*[^*]+\*\* ' RELEASE_NOTES.md     72
$ grep -cE '^- \*\*[^*]+\*\*[ .]' CHANGELOG.md      35
```

---

## Every number, with its command

| figure | command | value |
|---|---|---|
| shell assertions, **bash 3.2.57** | `for s in tests/test_*.sh; do /bin/bash "$s" \| tail -1; done` | **3152**, 0 failed |
| shell assertions, **bash 5.3.15** | same with `/opt/homebrew/bin/bash` | **3152**, 0 failed, identical per suite |
| — per suite, both shells | as above | `test_f01` 1912 · `test_f02` 350 · `test_harness` 276 · `test_install` 52 · `test_rewrite` 401 · **`test_skill` 141** · `test_walk` 20 |
| Go tests | `go test -list '.*'` per package, `grep -cE '^(Test\|Example)'` | **286** = 239 / 35 / 12 — unchanged |
| Go subtests | `go test -v` per package, `grep -cE '^\s+--- (PASS\|SKIP\|FAIL): '` | **1014** = 579 / 87 / 348 — unchanged |
| coverage | `go test -cover ./...` | **97.9 / 94.7 / 95.9** — unchanged |
| build / vet / gofmt | `go build ./... && go vet ./... && gofmt -l .` | clean, `gofmt -l` empty |
| assertion audit | `bash tests/lib/audit-suites.sh tests/test_*.sh \| tail -1` | `sites=837 files=9 lines=12059 accounted=12059` |
| promise reader, tree | `bash tests/lib/forward-promise-check.sh 0.5.0` | `files=162 lines=44184 examined=43819 skipped=365`, **rc=0** |
| fixture table | `sed -n "/^done <<'PROMISE_CASES'$/,/^PROMISE_CASES$/p" … \| grep -cE '^(refuse\|accept)\|'` | **42** (30 refuse / 12 accept) |
| version occurrences | `git grep -o '0\.5\.0' \| wc -l` / `git grep -n …` | **not published any more** (was 101/91, false before I touched it) |
| `skill-validator` | `skill-validator check skills/skill-audit` and `… skill-rewrite` | both `Result: passed`, rc=0, **zero errors, zero warnings** |
| `skillscore` | `skillscore --version` | **2.0.2**, global install intact, no PATH mirror built |

### Deltas, and why each is what it should be

- **3137 → 3152 (+15)**, all in `test_skill` (**126 → 141**): 7 new rows in `PROMISE_CASES`
  driven through the loop's two existing `assert` sites, **plus** 8 new `assert` sites in
  the section-scope block.
- **`sites` 829 → 837 (+8)** — the same 8 sites, counted by a different program reading
  the source rather than by the suite at runtime. The two sides agreeing on 8 is the
  cross-check, not a restatement.
- **`lines` 11913 → 12059 (+146)** — what `tests/test_skill.sh` grew by.
- **Go figures unchanged**, which is the evidence no Go file was touched.

### Frozen sections — byte-identical to `v0.4.3`, verified after the last edit

```
$ sed -n '/^## 0\.4\.3/,$p' CHANGELOG.md | wc -l ; … | md5 -q
240   ef2fe48af306b4ce06ace48391eddcb8
$ git show v0.4.3:CHANGELOG.md | …
240   ef2fe48af306b4ce06ace48391eddcb8
$ sed -n '/^## v0\.4\.3/,$p' RELEASE_NOTES.md | wc -l ; … | md5 -q
112   46d66b1e331726dbe56a830d449c19aa
$ git show v0.4.3:RELEASE_NOTES.md | …
112   46d66b1e331726dbe56a830d449c19aa
```

---

## Commits, all pushed to `release/0.5.0`, all authored `imagineux`

| commit | item | CI |
|---|---|---|
| `48d44c4` | item 1 — the exemption asks about one `is`, not the whole line | success |
| `fc9412b` | item 4 + false claim 2a — a section it cannot find is a finding | success |
| `ac0a16c` | item 3 + false claim 2b — the census of its own version string goes | success |
| `fec81b1` | finding 1 — the published boundary is the one the reader has | success |
| `f9c8a17` | the counts, re-derived | success |
| `9342964` | the scratch repository is guarded before it is removed | see below |
| `906a1b4` | four spellings fail closed, the fifth is read (comment accuracy) | see below |

```
$ git log 34976a2..HEAD --format='%h %an <%ae>' ; git log 34976a2..HEAD --format='%B' \
    | grep -icE '^Co-Authored-By|Generated with \[Claude|noreply@anthropic'
0
```

All seven runs **completed success** on `ubuntu-latest` — which matters more than usual
here, because the awk on that runner is not the awk this was written against, and the
repair adds `match()`/`RSTART`/`RLENGTH` and two associative arrays to the program. PR #22
re-queried after the last push: **MERGEABLE / CLEAN**, check `test` SUCCESS.

---

## For the maintainer, before tagging

1. **One behaviour change to know about, and it is the point of item 4.** The reader now
   exits 1 if a history document has no heading it can read for the version it was given.
   If you ever run it for a version whose section is not written yet, that is a refusal
   rather than a quiet pass. It is consistent with the invariant the suite already holds
   elsewhere — each document must carry a section for the release being cut — and it is
   what makes the scoping safe to have at all.

2. **The exemption is stricter than it was even at `bf045ce`, deliberately.** Three lines
   that passed at both previous heads now fire, because the version word was on the record
   but nowhere near the `is` (`the version matrix says building the capability is …`). That
   is the disclosed general over-breadth being narrowed rather than restored, and it is
   driven clean over the whole tracked tree on both shells. Only four lines in the tree
   depend on the exemption at all: two in `tests/`, one in `CHANGELOG.md:188`, one in the
   reader header — all of them a version surface adjacent to the `is`.

3. **Residual 1 stays refuted, and I only moved its literal.** `PROMISE_CASES_EXPECTED`
   went 35 → 42. The report drove it and showed the control works (shrinking the heredoc
   reddens the suite), so this is a literal beside a heredoc rather than a both-sides
   shrink. Unchanged in shape by me.

4. **Two residuals I did not touch, both out of the four items.** CRLF disables the bare
   `is <version>` shape on its right-hand alternation (`$` cannot match past `\r`), dormant
   at zero CRLF files and no `.gitattributes`; and the apostrophe hazard in the awk body,
   which fails loudly (rc=2). I checked my own additions against the second: the awk
   program contains exactly two apostrophes, the shell quotes that delimit it, and
   `bash -n` is clean.

5. **The 0.6.0 shelf gained one row**, next to the `macos-latest` job: the verb-to-version
   span. See the finding-1 section above for the measurement that should shape it — the
   deferral shape alone looks free on this tree, the other three do not.

6. **No PR threads exist, so none were replied to and none resolved.**

```
$ gh api graphql … pullRequest(number:22){ reviewThreads, reviews, comments }
threads: 0
reviews: []
issue comments: []
```

There is no inline thread from the external reviewer on #22 and no review of any kind, so
there was nothing to cite a fixing commit into. If one appears before you tag, the four
commits above map one-to-one onto the four items.

7. **Not merged. Not tagged.** `v0.5.0` is yours.

---

## Environment and isolation

- All work in `…/scratchpad/pgrepair`, my own worktree, created fresh at `34976a2` on
  `fix/0.5.0-promise-guard-repair`, pushed to `release/0.5.0`. `pwd` confirmed inside it
  before every write to the tree.
- **The primary tree was not touched**: still at `0a83615`, still 47 entries in
  `git status --porcelain`, unchanged from when I started. The only file I wrote outside
  my worktree is this record, in the gitignored `.scuba/` control plane, by absolute path.
- `scratchpad/pgfix` was never entered. No `checkout`, `reset`, `clean` or `stash` in any
  tree but my own. Every mutation driven over the real tree (heading spellings, the planted
  assignment) was made from a copy taken first and restored by `cp`, with
  `git status --porcelain` confirmed clean after each run.
- This file was written with the **Write/Edit tools**, by absolute path, never a heredoc.
- `skillscore` still reports **2.0.2** from its global install; no PATH mirror was built.

