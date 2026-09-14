# Status — skill-audit script guards (0.4.2)

**State: PR open, not merged.** Both defects fixed, the class behind them closed, and
four fix rounds applied at the root: the confirming pass's eight items, the correctness
pass's five (A1–A5), the final pass's eight (F1–F8), and the closing pass's three.
Shape and verdict are separate propositions with shape settled first, in **all three**
of the composer's sources rather than one; the encoder is settled by ordinal rather than
by a locale-dependent character class; every new arm is mutation-tested; all four shell
suites green at 850 assertions under both `LC_ALL=C` and a UTF-8 locale; both shipped
skills validate with zero warnings; `profiler/` untouched. The exit-set sweep at this
head finds nothing outside `{0,1,2,3}` anywhere — a claim that was made a round early
and is now true with a re-run sweep behind it.

- PR: https://github.com/Okja-Engineering/skill-architect/pull/4 — non-draft, base `main`.
- Branch: `fix/skill-audit-tool-guards`, worktree
  `/Users/matthewvandusen/Development/Auraprix/skill-architect-wt/scripts-0.4.2`
- Base: `origin/main` = `2ed34b8` (v0.4.1). Head: `85ad0ea1d15799609df6de674f2a20675d12a3dc`.
- Nine commits, author and committer `imagineux <imagineux@gmail.com>` on all nine, zero
  `Co-Authored-By` trailers and no AI attribution of any kind in any commit message. No
  `-c user.email=`, `--author=` or `GIT_*_EMAIL` was used.

## Defects — both fixed, reproduced before and after with a masked PATH

Masked PATHs are symlink farms of the real PATH minus one binary. Nothing was deleted.

### S1 — `check-structure.sh --json` reported a clean pass with `jq` absent

`check-structure.sh:71` (at `2ed34b8`) swallowed the missing binary with `2>/dev/null || true`,
so path findings vanished and the zero-findings branch at `:92-93` hard-coded `passed: true`.
`:68`/`:86` mapped only child exit 1 onto failure, so the child's 127 read as success.

    jq masked, before : {"findings": [], "passed": true}   exit 0
    jq masked, after  : DEP001 payload, passed false       exit 3   (+ "ERROR: required tool not found: jq" on stderr)
    jq present, after : two PT001 findings, passed false   exit 1   (unchanged)

`check-paths.sh` — same root, different symptom: died at `:94` with a bare
`jq: command not found`, exit 127, no payload. Now exit 3 with a `DEP001` payload.

### S2 — `check-frontmatter.sh` printed `frontmatter OK` with `skill-validator` absent

`:24` captured 127; `:26`/`:32` special-cased only 3 and 1, so it fell through to the
license gate.

    licensed skill,  validator masked, before : frontmatter OK                 exit 0
    licensed skill,  validator masked, after  : ERROR: ... skill-validator     exit 3
    malformed-yaml,  validator masked, before : POLICY FAIL [PL001]            exit 2
    malformed-yaml,  validator masked, after  : ERROR: ... skill-validator     exit 3
    validator present, after                  : frontmatter OK / SPEC FAIL     exit 0 / 1 (unchanged)

## What shipped

- New `skills/skill-audit/scripts/verdict-guard.sh` — `cannot_compute()` owns
  "no verdict could be reached" (stderr reason, `passed:false` JSON payload, exit 3);
  `require_tool()` is its missing-tool caller. Replaces the prior session's
  `lib/require-tool.sh`, which was adopted, extended to cover the child-exit case, and
  renamed to match what it now holds.
- `check-paths.sh` / `check-structure.sh`: `--json` requires `jq` unconditionally
  (manager decision 1 — a stated precondition, not a data-dependent one). Text mode
  is unaffected.
- `check-structure.sh`: enumerates `check-paths.sh`'s exit codes, anything unenumerated
  is an execution error; stops folding the child's stderr into the payload it parses;
  no longer swallows an unreadable payload.
- `check-frontmatter.sh`: `case` enumerating 0/1/2/3, `*` is an execution error.
- Both `--json` writers derive `passed` from the computed verdict in one place; the
  zero-findings shortcut that hard-coded `passed: true` is gone.
- `audit-report.sh`: sources the same guard and requires `jq`, which it composes its
  whole report with. Exit 3 naming the tool, no partial report, instead of exit 127
  from the shell at `:85`. `skill-validator` and `skillscore` stay soft there.
- `README.md`: jq row, the `skill-validator` row's now-false clause, the "no such
  guard for jq" sentence (manager decision 2), and a `DEP002` paragraph.
- `skills/skill-audit/SKILL.md`: the rule-ID enumeration now registers `DEP001`,
  `DEP002` and `PATH`, and states both levels. Its `exit 1` dependency guards are
  byte-identical; the whole file diff is **two lines** (`git diff --numstat` says `2 2`),
  the other being the `spec_error`/`policy_error` sentence. An earlier claim of "one
  line" was true at `21ee4b3` and went stale.
- `tests/fixtures/f01/path-fault-only/SKILL.md` — adopted from the prior session
  unchanged; verified it raises PT001 and no PL finding.
- `tests/lib/masked-path.sh` — the PATH-masking harness, lifted out of `test_f01.sh`
  because `test_f02.sh` needs the same masking.
- 86 new assertions across three suites at the time of that round; **194** at `fbd1fc8`
  after the A1–A5 round; **583** at head `7866db5` after the F1–F8 round (see Round 3).

## Fix round — the confirming pass's eight items

All eight verified against the tree before acting; all eight disposed of.

1. **`scripts/lib/` cost skill-audit a clean bill of health** (MEDIUM) — real.
   `warnings: 1` at `60c7f44`, `0` at `2ed34b8`, the warning being
   `deep nesting detected: scripts/lib/`. Flattened to `scripts/verdict-guard.sh`;
   sourcing updated in all three scripts (plus `audit-report.sh`, new); `lib/` removed.
   Now `warnings: 0` for both shipped skills, pinned by `tests/test_skill.sh`.
2. **Unregistered rule IDs** (MEDIUM) — real. `DEP001`/`DEP002` registered at
   `SKILL.md`'s rule-ID line, in all three emitting script headers, and in README prose.
   Five new assertions pin `DEP002` in the payload for both of its cases; `DEP001` was
   already pinned in both scripts' payloads.
3. **`audit-report.sh` unguarded** (LOW) — real, exit 127 at `:85` reproduced. Guarded
   through the same `require_tool`. README's jq row updated. Four new assertions,
   plus four controls pinning that the two soft dependencies stayed soft.
4. **Guard built JSON by interpolation** (LOW) — real; a message with a quote emitted
   invalid JSON, demonstrated live. Replaced with a shell JSON-string encoder
   (`json_string`), which the guard needs rather than `jq` because a missing `jq` is
   what calls it. Six new assertions.
5. **Headers under-described the payload** (LOW) — real. Both headers now name the
   `"error"` key and the rule IDs they emit.
6. **Exit 3 did not always carry a payload** (LOW) — partly real, and the real part was
   a channel bug, not a documentation gap: `check-structure.sh:34` and
   `check-frontmatter.sh:22` wrote the SKILL.md-not-found diagnostic to **stdout**,
   which in `--json` mode is the payload channel. Moved to stderr, matching
   `check-paths.sh` and `audit-report.sh`. The remaining exception is now one clause —
   a usage error and an unresolvable target carry no payload because there is no skill
   to render a verdict about — stated in both headers and pinned by thirteen assertions.
   Deliberately **not** done: minting new rule IDs for those two paths. See below.
7. **"35 new assertions" was wrong** (MEDIUM) — real. Recounted by running both trees.
8. **Undisclosed formatting change** (LOW) — real. Disclosed as deliberate decision 3
   in the PR body, with the compact form left removed and the reason given.

### Judgment call, flagged

On item 6 the finding offered "make them consistent or name them". Making the
usage-error and target-not-found paths emit payloads would have meant minting two more
consumer-observable rule IDs — enlarging exactly the surface item 2 exists to police —
to describe caller errors rather than verdicts about a skill. The channel bug inside
item 6 was fixed at the root; the remaining no-payload exits are named in both headers
and pinned by tests instead. `check-quality.sh` was left alone: it already guards
`skillscore` inline with an install hint README quotes, so it is not an instance of the
unguarded class.

## Verification as at `fbd1fc8` — superseded by "Round 3 verification" below

Kept as the record of that head. Every current figure is in Round 3 verification.

- New assertions vs `2ed34b8`, counted by running both trees. At head `fbd1fc8`:
  `test_f01` 32 → 198 (+166), `test_f02` 58 → 84 (+26), `test_skill` 25 → 27 (+2),
  `test_walk` 19 → 19 (+0). **194 new**, 328 in total. At `21ee4b3`, before the A1–A5
  round, this read 106 / 68 / 27 / 19 and **86 new** — which was correct then, and was
  itself a correction of an earlier 35 and 48.
- RED first: with the tests as they stand at head over `2ed34b8`'s `skills/` in a
  scratch copy, **110** fail (100 in `test_f01`, 10 in `test_f02`). GREEN after.
  Recounted at head `fbd1fc8`; the figure was 44 when this section was written, before
  the A1–A5 round added its assertions. The `skill-audit` validator-cleanliness
  assertion is RED against `60c7f44` instead, because `scripts/lib/` is this PR's own
  regression; `skill-rewrite`'s is green throughout (the earlier claim of "two"
  assertions was one).
- All four suites green from the worktree root: `test_skill` 27/27, `test_walk` 19/19,
  `test_f01` 198/198, `test_f02` 84/84 — 328 total.
- `skill-validator check -o json` on both skills: `errors 0, warnings 0` (was
  `warnings 1` for `skill-audit` at `60c7f44`).
- Masked-PATH matrix re-run and matching README: `audit-report.sh` exit 3 naming jq with
  empty stdout; `check-paths.sh`/`check-structure.sh --json` exit 3 with a `DEP001`
  payload; both text modes unaffected; `check-frontmatter.sh` exit 3 naming
  skill-validator; `audit-report.sh` exit 0 with `spec: null` + `spec_error` and
  `summary.passed: false` under a masked validator, and `quality: null` + `quality_error`
  with `quality_score`/`quality_grade` null under a masked skillscore.
- `cd profiler && go test -count=1 ./...` passes; `git status profiler/` is empty.
- No version surface touched: no manifest, no `marketplace.json`, no `profiler/types.go`,
  no `test_skill.sh` version asserts, no CHANGELOG, no RELEASE_NOTES.

## Round 2 — the correctness pass's five items (A1–A5)

Worklist: `worklist-round2.md`. Its line numbers were against `60c7f44`, so every site was
re-located and every finding re-verified at `21ee4b3` before being touched. Two commits:
`c8607b6` (tests, RED) then `fbd1fc8` (fix).

**A1 — did not reproduce at `21ee4b3`; skipped as reported, but its invariant was still
violated.** The previous round's `stub_paths_garbage_tree` case already kills A1's mutant
(3 assertions). The worklist's pointer to a child "that prints nothing" was stale. Walking
the accepted edge properly, however, showed the invariant still broken by a route the
garbage case did not cover: a child exiting 0 having printed **nothing** left
`jq -c '.findings[]'` succeeding with no output, so `--json` reported `passed: true`.
Fixed with A2, same root.

**A2, A3 — reproduced; both mutants survived all 174 f01+f02 assertions at `21ee4b3`.**
Root confirmed as stated: one witness of each rejected set, never the accepted set's edge.

**Root fix.** Every `case` and `||` the branch introduced is walked across its whole
boundary. `check-frontmatter.sh` at 0, 1, 2, 3, 4, 42. `check-structure.sh` over a
nine-row table of child results — each documented status with a conforming payload, plus
unreadable, absent, misshapen, and payloads contradicting their status — asserting exit,
`passed`, and rule ID, with one assertion carried across all nine rows and pinned to the
invariant rather than any arm: a payload may never claim it passed while carrying a
`level: "fail"` finding. `json_string`'s `case` gets one character from every arm.

In code, `check-structure.sh` makes one validating pass over the child's payload that
answers readability and verdict together, treats a verdict disagreeing with its status as
`DEP002`, and derives the child's pass/fail once for every use below. The read-back's own
guard was **deleted**, not kept: that pass makes it unreachable, and dead code kept "to be
safe" is what this branch exists to remove.

**Corrected in Round 3.** "Answers readability and verdict together" was the defect, not
the fix: the two halves were joined by an `and` that short-circuits, so readability went
unasked on the branch the verdict took, and the read-back's guard was not in fact
unreachable — a misshapen element reached it and exited 5. The claim that the nine-row
invariant assertion "survives any rework of the arms that currently enforce it" was
false: it never fired, because no row paired `passed: true` with a `level: "fail"`
finding. See Round 3.

**A4 — reproduced exactly; fixed.** `audit-report.sh` captured its only guarded child with
`2>&1`, so a `passed:false` payload never parsed and findings stayed empty.

    check-paths.sh removed, all tools present
    before : check-structure.sh --json -> DEP002 payload, exit 3
             audit-report.sh          -> {"passed": true,  "total_findings": 0}, exit 0
    after  : audit-report.sh          -> {"passed": false, "total_findings": 1, DEP002}, exit 0

stdout captured alone per the child's contract (payload on stdout, diagnostics on stderr),
which is what makes the `DEP002` payload reachable by the consumer it was registered for.
A source yielding nothing readable is named in a new `policy_error` with `summary.passed`
false, mirroring `spec_error`. `quality_error` deliberately unchanged — a score, not a
verdict.

**A5 — reproduced, and worse than reported.** A *malformed* guard exits **2**, not 1 — a
status the contract reserves for "policy failure". Reproduced on all four scripts,
`audit-report.sh` included, where exit 1 is not even an enumerated status. Each script now
checks the load on both sides (`bash -n` before, `declare -F` after) and exits 3 with no
payload. Kept small as directed. Adds a third payload-less exit-3, stated in both script
headers, the guard's header and `README.md`, and pinned in `test_f01.sh`.

**Disagreements recorded.** A5's "and `audit-report.sh` then reports passed" is not
reachable as described — all four scripts share the one guard file, so if the check
scripts cannot load it neither can the composer; the real harm is `audit-report.sh`
exiting 1 with no report. Separately, `tests/test_skill.sh` has a latent cwd dependency
(no `cd "$(dirname "$0")/.."`, unlike the other three suites) and fails when invoked by
absolute path from elsewhere — pre-existing, out of scope, left alone and flagged.

### Round 2 verification

- Mutation-tested, since the point of Root A is that the previous tests could not fail.
  Eight mutants on `fbd1fc8`, all eight killed: A1's read-back **15**, A2's `0|1|3)` **3**,
  A3's fall-through **3**, readability arm deleted **11**, consistency check deleted **6**,
  guarded `source` reverted **7**, `audit-report.sh`'s `2>&1` restored with the
  else-branch dropped **5**, `json_string`'s control-char arm deleted **4**.
  A1's mutant is applied where that guard now lives; applied literally to the read-back
  line alone it is an *equivalent* mutant, because that line can no longer fail.
- RED against the previous head `21ee4b3`: **35** assertions fail (28 `test_f01`,
  7 `test_f02`). GREEN at `fbd1fc8`.
- Masked-PATH matrix re-run against every claim `README.md` makes about it: **30 claims,
  30 matches**, including the nine new guard-load claims.
- `skill-validator check -o json`: `passed true, errors 0, warnings 0` on both skills.
- `cd profiler && go test -count=1 ./...` passes; `git status profiler/` empty.
- No version surface touched. Identity: one author/committer across all five commits,
  zero trailers.
- **Round 3 found this eight-mutant set incomplete, not wrong.** Each of the eight was
  genuinely killed; five further non-equivalent mutants survived it, all inside the single
  validating pass this round introduced. See Round 3.

## Round 3 — the final pass's eight items (F1–F8)

Every item re-verified at `fbd1fc8` before being touched, and the five mutants the pass
reported as survivors re-run against the fixed tree. Two commits: `8e4df2a` (tests, RED)
then `7866db5` (fix).

**Root A (F1–F5) — one root, closed once, and it was this PR's own root one level down.**
`check-structure.sh:125-129` asked "is this payload well-formed?" and "what verdict does
it carry?" in one jq expression joined by an `and` that short-circuits, so the
well-formedness half ran only on the branch the verdict took.

    child exits 1 with {"passed": false, "findings": [42]}
    before : jq: Cannot index number with string "level"   exit 5, stdout empty
    after  : DEP002 payload, passed false                  exit 3

Same for `["x"]` and `[[1]]`. `audit-report.sh:96` had the same split one level up —
`.findings` proved to be an array, then `.level` selected and `startswith` called on
`.rule` of every element — giving exit 5, and over a stream of documents a further
**exit 2** (`jq: invalid JSON text passed to --argjson`) that the pass had not listed.

The repair is one shared predicate, `payload_is_conforming` in `verdict-guard.sh`: the
documented payload shape, proven all the way down to each element because that is how far
the consumers read, and proven unconditionally — before any verdict is read and whatever
the verdict says. Both consumers use it, so there is one definition of "conforming
payload" rather than two partial ones. `check-structure.sh` then asks the two remaining
questions separately: does the payload agree with itself, and does it agree with the
status it arrived with.

**F2 — confirmed, including the false sentence.** The mutant survived all 282 assertions
at `fbd1fc8`. The assertion claiming to pin the invariant across all nine rows never
fired: no row paired `passed: true` with a `level: "fail"` finding. That row now exists at
*both* statuses, and the payload is refused at both — previously it was accepted whenever
the status happened to match the half being read. The PR body sentence claiming the
assertion "survives a rework of the arms currently enforcing it" is corrected, not
repeated.

**F5 — reproduced, and worse in one direction than reported.** Against a guard defining
only `cannot_compute`, `check-frontmatter.sh` and `audit-report.sh` exit 127 as reported,
but `check-paths.sh` runs to a **clean exit 0** with no guard behind the verdict. The
post-source check now asks `verdict_guard_ready` — one question the guard file answers
about itself, with the list of guards living once beside the definitions it covers. That
is also what kept this round's new predicate from becoming a third unpinned operand in
four `&&` chains. The fixture grew a `half` mode and a parametrised `drop-<name>` mode
that cuts one definition out of the real file, so the list itself is pinned.

**Root B (F6) — census taken; the sentence is made true and kept true.** The real set is
**ten**: PL001–PL005, PT001, PT002, DEP001, DEP002, and `PATH` at level `unverified`
(`check-paths.sh:77` and `:97`), which propagates into `audit-report.sh`'s
`policy.findings` and appeared in no document, no script header and no test. The registry
line names all ten and states both levels; each script's header registers what it emits;
and `test_f01.sh` extracts every rule literal the scripts can emit and compares it against
the registry line, so neither side can grow without the other. `PL003`, also untested, is
pinned by a generated fixture over the line limit.

**F7 — fixed and pinned on real multi-byte text.**

    json_string "café 😀 naïve"
    before : "caf￿ffffffffffc3￿ffffffffffa9 …"   parses; decodes to mojibake
    after  : "café 😀 naïve"                                round-trips exactly

`printf -v ord '%d' "'$c"` yields −61 for `0xC3`, so every byte of a UTF-8 character took
the `< 32` branch. It now escapes on `[[:cntrl:]]` and passes every other byte through.
The five named-escape arms are left alone: the pass is right that they are equivalent to
the `\u00xx` fallback.

**F8 and the nits.** `FAIL:` → `ERROR:` is now disclosed beside the channel move it
travelled with, and PR body decision 3 no longer claims one changed output. The
fabricated `{"level": "null", "rule": "null", "message": "null"}` finding is gone at the
root — a non-conforming element makes the whole payload non-conforming, so there is
nothing to fabricate from. `audit-report.sh` reads the policy source's own `.passed`
instead of re-deriving it from the findings; it was reporting `summary.passed: true` over
a source that had said `"passed": false`.

### Round 3 verification

- **Mutation-tested: 17 mutants, 15 killed, 2 shown equivalent.** All seven survivors the
  pass reported are dead: cross-check deleted (2 assertions), `.passed`-is-boolean dropped
  (9), `.findings`-is-array dropped (1), shape check deleted (100), composer back to
  `jq -e .` (43), guard check narrowed to one name (7), encoder back to ordinals (3). This
  round's own arms: element shape (59), one-document (10), `.level` (3), `.rule` (10),
  `.message` (1), the completeness list losing a guard (2), the composer re-deriving the
  verdict (1), payload-vs-status (11). The two survivors are the `type != "object"` gates:
  without them jq raises, and a raise already reads as "not the documented shape", so no
  test can distinguish them. Kept, labelled as equivalent in the code, and reported as
  equivalent rather than as kills.
- **Exit-set sweep, 1078 invocations: nothing outside `{0,1,2,3}` anywhere.**
  **Corrected in Round 4: this was false at `7866db5`.** A re-run sweep found nineteen
  `audit-report.sh` invocations outside its documented set there — fifteen at exit 5 and
  four at exit 2 — all of them the two soft sources this round had not reached. The
  sentence is true at `85ad0ea`. See Round 4. Every
  fixture × every script × every mode × each tool masked; every child payload shape ×
  every child status including 42 and 127; every argument error and unresolvable target;
  every broken-guard mode; `skill-validator` at each status; a SKILL.md over the line limit
  and an empty one. The identical sweep over `fbd1fc8` found **47** outside a documented
  set. The two hits remaining are `check-quality.sh` forwarding `skillscore`'s exit 1 —
  see found-not-fixed.
- Assertion counts, by running the trees: `test_f01` 32 → 518, `test_f02` 58 → 153,
  `test_skill` 25 → 27, `test_walk` 19 → 19. **583 new**, **717 total**, all green.
- RED first: **56** assertions fail against `fbd1fc8` (37 `test_f01`, 19 `test_f02`);
  **289** against base `2ed34b8` (232 and 57). GREEN at `7866db5`.
- Masked-PATH matrix against every claim `README.md` makes: **45 claims, 45 matches**.
- `skill-validator check -o json`: `passed true, errors 0, warnings 0` on both skills.
- `cd profiler && go test -count=1 ./...` passes; `git status profiler/` empty.
- No version surface touched: `git diff --name-only origin/main...HEAD` matches no
  manifest, no `marketplace.json`, no `profiler/`, no CHANGELOG, no RELEASE_NOTES, and no
  version-assert line in `tests/test_skill.sh`.
- Identity: one author/committer across all seven commits, zero trailers.

### Found, not fixed

- **The per-finding `jq` re-invocation.** Both `--json` writers spawn a `jq` per finding;
  a 4,000-finding payload takes about 95 seconds. Pre-existing, and now on a hotter path
  because the composer's shape check runs before it. A performance change, not a
  correctness one; it wants its own change.
- **`tests/test_skill.sh` has no `cd "$(dirname "$0")/.."`,** unlike the other three
  suites, so it fails when invoked by absolute path from elsewhere. Pre-existing.
- **`check-quality.sh` forwards `skillscore`'s exit status,** so a bad argument leaves it
  exiting 1 where its header documents `{0, 3}`. It is the one script in the skill this PR
  does not touch. A header that is wrong about an untouched script is not worth widening
  the final round's diff for. **Corrected in Round 4: "its two sweep hits" understates
  it.** Every caller error forwards, so it is a class; how many hits a sweep records is a
  fact about the sweep's inputs (3 in Round 4's sweep, 9 in the closing pass's), not about
  the script. Still deferred.

### Disagreement recorded

The pass asked for "no reachable input may exit outside `{0,1,2,3}`" and that holds
everywhere. The two sweep hits above are outside `check-quality.sh`'s *own* narrower
documented set of `{0, 3}`, not outside `{0,1,2,3}` — a different claim, and one about a
file outside this PR's diff.

## Round 4 — the closing pass's three items

Scope was three items and was held to three. Every figure below is this round's
own re-measurement, not a figure carried forward. Two commits: `2336942`
(tests, RED) then `85ad0ea` (fix).

**Item 1 (blocking) — the closed class was still open at two of three sources.**
Confirmed at `7866db5` before touching anything, on a stubbed `skill-validator`
and `skillscore`:

    skill-validator emits 0             -> jq: Cannot index number with "passed"          exit 5
    skill-validator emits [1,2]         -> jq: Cannot index array with "passed"           exit 5
    skillscore emits true               -> jq: Cannot index boolean with "overallScore"   exit 5
    either emits a two-document stream  -> jq: invalid JSON text passed to --argjson      exit 2
    in every case: stdout empty, no report at all

`audit-report.sh:64` and `:78` at `7866db5` asked `jq -e .`, which proves only
that a document parsed; `:142`/`:143` then passed it to `--argjson` and `:154`
and `:159-163` indexed `.passed`, `.errors`, `.warnings`,
`.overallScore.percentage` and `.overallScore.letterGrade`. Pre-existing and
unchanged from base; unreachable with the installed tools, which emit objects.

**The repair hoists the shared half instead of copying the predicate.** New
`json_document_conforms <text> <claim>` in `verdict-guard.sh:155-190` owns the
sentence that is the same for every source — exactly one document — and each
source states only the shape it is read as:

| source | claim | depth of the read |
| --- | --- | --- |
| `skill-validator` (`audit-report.sh:66`) | one document, an object | `.passed`, `.errors`, `.warnings` |
| `skillscore` (`audit-report.sh:84-88`) | one document, an object whose `overallScore` is an object or absent | `.overallScore.percentage`, two levels |
| `check-structure.sh` (`audit-report.sh:124`) | `payload_is_conforming`, meaning unchanged | down to each finding's `level`/`rule`/`message` |

The claim reaches exactly as far as the read. Stopping the quality source's claim
at its top level is the same defect one level down — `{"overallScore": 80}`
parses, is an object, and still takes jq down — and is mutation-tested as such.
Reaching further would make a source unreadable over a field nobody wanted: a
`skillscore` payload with no `overallScore` at all is read, and leaves the score
null. `payload_is_conforming` is now expressed through the same guard
(`verdict-guard.sh:192-211`), so nothing else in the tree defines "one document".

**Item 2 — the encoder corrupted one byte class, and the class is wider than
reported.** `verdict-guard.sh:74-76` at `7866db5`:

    LC_ALL=en_US.UTF-8  json_string "$(printf 'a\x80z')"  ->  "a￿ffffffffff80z"
    LC_ALL=C            same input                        ->  "a\x80z"  (passed through)

**Disagreement with the finding, recorded: "valid UTF-8 is unaffected in all
three locales" is false.** The valid C1 controls U+0080-U+009F are multi-byte
UTF-8 characters that JSON asks no one to escape, and they were corrupted
identically — `json_string` on U+0085 emitted `￿ffffffffff85` under a UTF-8
locale and passed it through untouched under `C`. So the encoder's output
depended on `LC_ALL`, which for a JSON encoder is the defect itself, and the
class is "every byte at or above 0x80 that the locale's `cntrl` class matches",
not only malformed input. (Also a recount: the escape is sixteen hex digits, not
eighteen.)

Root: the decision and the format were two questions that could disagree.
`\u00xx` can spell an ordinal below 0x80 and nothing else; `[[:cntrl:]]` matches
bytes bash reports as *negative* ordinals. The decision is now the ordinal
alone, asked only about the range the format can spell, so a negative ordinal has
no path to the format and the encoding is identical under every locale. Pinned as
the invariant — no escape wider than four hex digits, and the same bytes out
under `C` and under a UTF-8 locale — not as the instance.

Each witness ends in a character that is **not** a hex digit. A JSON escape
carries no terminator, so `\u0001b` is five hex digits by inspection and four by
construction; the first draft of the width assertion read the text after the
escape as part of it and produced two false REDs, which is recorded here because
it is exactly the kind of test that would have shipped as a true-looking pass.

**Item 3 — one surviving non-equivalent mutant, closed by fixture rows.**
Weakening `verdict-guard.sh`'s `end] | all` to `(length == 0 or any)` survived
the whole tree, because every existing row breaks the shape in an array where no
element conforms. A conforming element beside a non-conforming one separates
them. Added at both `.passed` values in `test_f01.sh`'s shape table and once in
`test_f02.sh`'s policy table, so the composer's own table is not blind to the
arrangement its child's table now covers. Production code was already correct;
this is coverage. The mutant is now dead: 13 assertions.

### Round 4 verification

- **Mutation: 8 mutants over the arms this round touches, 8 killed.** Encoder
  back to the character class (8 assertions), the negative-ordinal guard alone
  dropped (18), `all` weakened to `(length == 0 or any)` (13), the spec source
  back to `jq -e .` (25), the quality source back to `jq -e .` (35), the quality
  claim stopped one level short of its read (15), the shared one-document half
  dropped (10), the new guard left out of `verdict_guard_ready`'s list (1). The
  previous round's seventeen stand: fifteen killed, two equivalent.
- **RED first, GREEN after.** With this round's tests over each earlier tree's
  `skills/` (scratch copies from `git archive`): **370** fail against base
  `2ed34b8` (249 `test_f01`, 121 `test_f02`), **141** against `fbd1fc8` (58, 83),
  **71** against `7866db5` (11, 60). GREEN at head. `test_walk` needs repo files
  outside `skills/` and `tests/`, so it takes no part in that harness; it is
  19/19 in the worktree and untouched by this PR.
- **All four suites green under both `LC_ALL=C` and `LC_ALL=en_US.UTF-8`:**
  `test_skill` 27, `test_walk` 19, `test_f01` 561, `test_f02` 243 — **850**.
  Both locales, because item 2 was visible only under one of them. (Local bash is
  3.2, so the suites and every helper added stay 3.2-compatible — no associative
  arrays.)
- **Assertion recount, by running every tree.** 2ed34b8 → head: `test_f01`
  32 → 561, `test_f02` 58 → 243, `test_skill` 25 → 27, `test_walk` 19 → 19.
  **716 new**, **850** total. `7866db5` carried 717, so this round adds 133. The
  717 and 328 figures recorded in earlier rounds re-measure correctly.
- **Exit-set sweep, re-run at four revisions, 1079 invocations each.** Every
  fixture × every script × every mode × each tool masked; every payload shape ×
  every child status including 5, 42 and 127; the same shapes stubbed into each
  of `audit-report.sh`'s three sources; every argument error and unresolvable
  target; every broken-guard mode including one per named guard;
  `skill-validator` at each status; a SKILL.md over the line limit and an empty
  one. Counting invocations outside **each script's own** documented set:

  | revision | outside | of which `audit-report.sh` |
  | --- | --- | --- |
  | `2ed34b8` (base) | **89** (1051 invocations — one fixture this PR adds does not exist there) | 53 |
  | `fbd1fc8` | **25** | 21 |
  | `7866db5` | **22** | 19 |
  | `85ad0ea` (head) | **3** | 0 |

  All nineteen `audit-report.sh` hits at `7866db5` were the two unguarded sources
  (fifteen at exit 5, four at exit 2); all nineteen are gone. The three that
  remain are `check-quality.sh`, and they are exit **1** — inside `{0,1,2,3}`.
  So "nothing outside `{0,1,2,3}` anywhere" is **true at this head and was false
  at `7866db5`**; the PR body and this file said it too early and now say it with
  the sweep behind it. These are this round's own numbers from its own sweep;
  the closing pass's independent sweep reported 110/20/17 over a different input
  set, and neither set of totals is wrong — the classes agree.
- **Tools-present regression, 105 cases** (every script × every fixture and both
  shipped skills × every mode, nothing masked): this head is **byte-identical to
  `7866db5` in all 105**, which is the expected result for a defect unreachable
  with the installed tools. Against base `2ed34b8`, exit statuses match in all
  105, and of the 45 JSON-carrying cases **0** differ once the two disclosed
  differences are normalised away — the `--json` writers' pretty-printing and
  `audit-report.sh`'s added `policy_error` key.
- `skill-validator check -o json`: `passed true, errors 0, warnings 0` on both
  shipped skills.
- `cd profiler && go test -count=1 ./...` passes; `git status profiler/` empty.
  PR #3 owns `profiler/` and it was not touched.
- No version surface: `git diff --name-only origin/main...HEAD` matches no
  manifest, no `marketplace.json`, no `profiler/`, no CHANGELOG, no
  RELEASE_NOTES, and no version-assert line in `tests/test_skill.sh`.
- Identity: author and committer `imagineux <imagineux@gmail.com>` on all nine
  commits; **zero** `Co-Authored-By` trailers and no AI attribution of any kind
  in any commit message. No `-c user.email=`, `--author=`, `GIT_AUTHOR_EMAIL` or
  `GIT_COMMITTER_EMAIL` was used at any point.
- `README.md` was **not** changed. Its sentence "when a source produces nothing
  it can read, it names the source in `spec_error` or `policy_error` and leaves
  `summary.passed` false" was true of a narrower class before and is true of the
  whole class now; the fix made an existing claim hold further rather than
  needing a new one.

### Found, not fixed — round 4

- **`test_f01.sh`'s rule-ID census is syntax-shaped, so the guard is weaker than
  the sentence it defends.** `rules_emitted_by` recognises four literal forms.
  `cannot_compute "XX003" ...` with the rule quoted extracts to nothing, and so
  does every `findings+=(...)` built from variables — including
  `check-structure.sh:158`'s own relay, `findings+=("${level}|${rule}|${message}")`.
  And `DEP001` is attributed only to `verdict-guard.sh:136`, never to
  `check-paths.sh:57`, `check-structure.sh:60`, `check-frontmatter.sh:41` or
  `audit-report.sh:60`, which emit it indirectly through `require_tool`; those
  four headers do register it, but the per-script assertion never asks, so it
  passes vacuously for them. Verified against the tree, not taken on report. Set
  equality holds today because every call site uses a recognised form. Recorded
  as directed; not fixed.
- **`check-quality.sh` — named as a class, not a count.** It forwards
  `skillscore`'s own exit status rather than mapping it, so *every caller error*
  reaches the caller as exit 1 where its header documents `{0, 3}`: a bad flag, a
  target that is not a directory, a path that does not exist. How many hits a
  sweep records is a fact about the sweep's inputs (this round's sweep: 3; the
  closing pass's: 9), not about the script. The earlier record of "two hits" is
  corrected to the class. Still the one script in the skill this PR does not
  touch, and still deferred.
- The two entries carried from round 3 — the per-finding `jq` re-invocation and
  `test_skill.sh`'s missing `cd` — stand unchanged.

### Not widened

Everything else the closing verification raised was left alone deliberately, as
directed. `README.md` untouched, `check-quality.sh` untouched, `profiler/`
untouched, no version surface, no new rule IDs, and no third place that knows
what a conforming document is.

Do not merge — the user merges to `main`.
