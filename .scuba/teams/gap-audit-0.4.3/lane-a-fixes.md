# Lane A — C1, C3, C4 · fix record

**Branch** `fix/0.4.3-body-guard-exits`, cut from `origin/main` @ `71cc866`.
**Worktree** `/private/tmp/claude-501/-Users-matthewvandusen-Development-Auraprix-skill-architect/31edaf48-6b19-4833-82e7-6cb52dac691f/scratchpad/wt-lane-a`
(the primary tree was never touched; the worktree sits under the scratchpad because
the enforcement hook's whitelist is the temp anchor, not a sibling directory).

**Baseline before any change**, bash 3.2: harness 46 · skill 27 · walk 20 · f01 575 ·
f02 243 = **911 passed, 0 failed**.
**Final**, both bash 3.2 and bash 5: harness 46 · skill 35 · walk 20 · f01 941 ·
f02 272 = **1314 passed, 0 failed**.

## Commits

| SHA | Cluster | What |
|---|---|---|
| `6e99aed` | C1 | repro: fixtures + cases, RED against the tree |
| `0041596` | C1 | the shared body/frontmatter primitive, four private copies deleted |
| `1ba6c6d` | C3 | require_tool's contract extended; every tool routed; payload channels proven |
| `7f87a87` | C3 | check-frontmatter.sh reads the spec payload it asked for (G3-01) |
| `fc92ac7` | C3 | audit-report.sh names the source it could not read (G3-04) |
| `211f7aa` | C3 | check-quality.sh rewritten inside the guard (G3-05, G4-02) |
| `41ef533` | C3 | draft-rewrite.sh inside the guard (G3-09) |
| `d698798` | C4 | the documented exit contract derived from the scripts' headers |
| `0d0c2fb` | C3 | draft-rewrite's temp audit made observable, so its cleanup is too |

---

## C1 — no single body-extraction primitive · DONE

### Verified before the fix, on the tree itself, on a healthy toolchain

**G1-01** P1, `tests/fixtures/f01/frontmatter-bleed` — six required headings as
column-0 YAML comments, a code fence inside a block scalar, the closing `---` read
as a list item, body one prose sentence:

```
skill-validator validate structure -o json  -> passed true, errors 0
check-structure.sh --json                   -> {"findings": [], "passed": true}, exit 0
check-frontmatter.sh                        -> frontmatter OK, exit 0
audit-report.sh | jq -c .summary
  {"passed":true,...,"policy_failures":0,"path_failures":0,"total_findings":0}
```

**G1-02** P1, `rule-before-refs` / `no-rule-before-refs`, differing by one `---`:

```
rule-before-refs    -> {"findings": [], "passed": true}   exit 0
no-rule-before-refs -> PT002 + PT001, "passed": false     exit 1
```

### The repair

`skill_frontmatter` / `skill_body` (sharing `skill_section`) in `verdict-guard.sh`;
all four scripts route through them; **all four private copies deleted**, including
the two that were correct. One extraction site remains in the whole tree, and a
source-text case with a firing control refuses a second one.

Three decisions stated in the primitive, each a defect in one of the copies:
frontmatter opens on line 1 or not at all; closes at the next `---` and never
re-opens; frontmatter running to EOF leaves no body.

G1-03 falls out: PL003 counts the body, which is what `draft-rewrite.sh:78` always
said the limit was.

### Non-vacuity — RED / GREEN

Scripts reverted, tests and fixtures kept:

```
FAIL: structure --json, body rules satisfied only in frontmatter: reports all six headings missing
FAIL: structure --json, body rules satisfied only in frontmatter: reports PL004, the body has no fence
FAIL: structure --json, body rules satisfied only in frontmatter: reports PL005, the body has no list
FAIL: structure --json, body rules satisfied only in frontmatter: never reports passed true
FAIL: structure --json, body rules satisfied only in frontmatter: exits 2 (policy failure)
FAIL: audit-report, body rules satisfied only in frontmatter: summary.passed is false
FAIL: audit-report, body rules satisfied only in frontmatter: counts every body rule it broke
FAIL: paths --json, a horizontal rule before the broken references: finds them anyway
FAIL: paths --json, a horizontal rule before the broken references: same findings as without it
FAIL: paths --json, a horizontal rule before the broken references: same exit as without it
FAIL: structure --json, a 500-line body in a longer file: PL003 raised is false
FAIL: no script outside the shared primitive reads the frontmatter delimiter itself
586 passed, 12 failed
```

Fix restored → `608 passed, 0 failed`. The headline verdict, on the surface the
defect was reported on:

```
audit-report.sh frontmatter-bleed | jq -c .summary
  before: {"passed":true, ...,"policy_failures":0,"total_findings":0}
  after:  {"passed":false,...,"policy_failures":8,"total_findings":8}
           PL002 x6, PL004, PL005
check-paths.sh --json rule-before-refs
  before: 0 findings, exit 0
  after:  PT002 + PT001, exit 1 — identical to its twin
```

Fixture preconditions are asserted, not assumed, so the cases cannot be satisfied
by editing the fixture. The guard drop-set in `test_f01.sh` is now derived from
`verdict-guard.sh` with a count assertion refusing an empty reading.

---

## C3 — the guard's invariant applied to part of the surface · DONE

All nine entries reproduced against the head before any fix. `require_tool`'s
contract extension turned out to be **one root under six symptoms** (G3-02, G3-03a,
G3-06, G3-08, G4-02, G4-03), which is why they are one commit.

| ID | Reproduced before | After |
|---|---|---|
| G3-01 | validator exits 0 printing non-JSON → `frontmatter OK`, exit 0 | exit 3, DEP002 naming the source |
| G3-02 | jq silent → audit-report exit 0, empty stdout | exit 3, naming jq |
| G3-03 | jq silent → check-paths --json exit 0, empty stdout | exit 3 + DEP002 payload |
| G3-03b | `chmod 000 SKILL.md` → check-structure --json exit 1, empty channel | **already closed by C1** (exit 3 + DEP002) |
| G3-04 | spec source exits 0 silent → `spec_error: null` beside `spec_passed: false` | a sentence naming the source and its status |
| G3-05 | no guard, no SKILL.md check, skillscore's 9 and its silence passed through | exit 3 on all four, inside the guard |
| G3-06 | grep exits 2 → nine fabricated PL failures, `policy_error: null` | exit 3, naming grep |
| G3-07 | `chmod 000` → audit-report exit 2, 0 bytes | **already closed by C1** (exit 3) |
| G3-08 | healthy check-paths shim + a jq that answers only `-se` → blamed check-paths.sh | the diagnostic names jq |
| G3-09 | tools masked → a draft written, exit 0, "Current state" = the tooling's own errors | exit 3, no draft |
| G4-02 | skillscore exits 9 → check-quality exits 9 | exit 3 |
| G4-03 | jq exits 5 → check-paths exits 5 | exit 3 |

### The repair

- **`require_tool` = present AND answered.** `tool_answers` holds one
  known-answer question per compute tool (jq, awk, sed, tr, wc, grep, cat,
  mktemp). DEP001 for absent, DEP002 for present-and-not-answering.
- **Every tool declared** where the dependency is established, in all six scripts.
- **`text_matches`** replaces every bare `! grep -q`: a command under `!` is exempt
  from errexit by definition, so an errored grep read as "not there" and a tool
  fault became a policy finding.
- **`text_extract`** replaces both reference sweeps in `check-paths.sh`, which sat
  inside the process substitutions feeding their loops — a subshell, where
  `cannot_compute`'s exit 3 dies unheard. One said `|| true` out loud. It assigns
  rather than echoes so no caller can put it back in a subshell.
- **The code-block awk in `check-paths.sh`** had its status discarded and it is
  reachable: a body with bytes invalid in the current locale makes awk exit 2,
  which under errexit became check-paths' own exit 2 — outside its contract.
  Reproduced with an invalid-UTF-8 body.
- **The mirror of the invariant**: all three payload channels prove their own
  document before printing it.
- **`check-frontmatter.sh`** asks the spec source the same three questions
  `check-structure.sh` asks its child (shape; agreement with status), and stops
  merging stderr into the payload channel.
- **`audit-report.sh`**'s `*_error` fields are sentences it writes, naming the
  source and its status; both `|| true`s gone; the policy source's status read
  against its payload.
- **`check-quality.sh`** rewritten inside the guard.
- **`draft-rewrite.sh`** inside the guard, statuses enumerated, stderr unmerged,
  section detection routed through `skill_body` (it had the same C1 defect), and
  its temp audit removed on every exit path.
- **Subtractions**: `sed` and `tr` leave `check-paths.sh` and `basename` leaves
  `draft-rewrite.sh` — parameter expansion does the same two operations with no
  tool to require and no status to interpret.

### Two defects the test matrix found in my own fix

The present-and-broken cross product caught both, and both are fixed with the rule
they broke written into `tool_answers`:

1. **A probe wrote to the payload channel.** The grep probe used `-q` without
   redirecting stdout; a broken grep does not honour `-q`, so the stub's output
   landed on the payload channel twice, ahead of the DEP002 payload. The guard
   that caught the fault corrupted the report of it. Rule: a probe writes to
   neither channel.
2. **A probe depended on another probed tool.** Reading wc's answer through `tr`
   made a broken `tr` fail the wc probe, and the diagnostic named **wc** — the
   wrong component, which is the one thing a diagnostic must not do. Rule: each
   probe asks exactly one tool, and the shell reads the answer.

### Non-vacuity

**These integers are per-commit measurements, not head measurements, and they do
not reproduce at head.** Corrected after the PR #10 gate, which was right about
this. Each was taken at that commit's own point in the series, against whatever
suite existed then. They are not reproducible at head by construction: the
broken-tool cross product is *derived* from each script's required-tools list,
so reverting a script changes how many cases the suite generates — the
denominator moves with the revert. Direction and magnitude hold; the integers
are historical.

Each commit reverted in isolation, tests kept, **as measured at the time**:

- `require_tool`'s self-test clause alone removed → 727 passed, 99 failed.
- `check-frontmatter.sh` alone reverted → 842 passed, 25 failed.
- `audit-report.sh` alone reverted → 262 passed, 10 failed (f02).
- `check-quality.sh` alone reverted → 894 passed, 32 failed.
- `draft-rewrite.sh` alone reverted → 28 passed, 7 failed (skill).
- cleanup trap alone removed → the leak assertion reports two files left behind.

**Re-measured at `8a72f73` (the fix-round head), one denominator throughout:**
f01 + f02 + skill, which is **1549 passed, 0 failed** with nothing reverted.

| Reverted in isolation | At head |
|---|---|
| `require_tool`'s answered-clause removed | **1398 passed, 151 failed** |
| `check-frontmatter.sh` to base | **1455 passed, 30 failed** |
| `audit-report.sh` to base | **1459 passed, 58 failed** |
| `check-quality.sh` to base | **1494 passed, 38 failed** |
| `draft-rewrite.sh` to base | **1440 passed, 33 failed** |

The pass totals differ per row for the reason above — a reverted script declares
fewer tools, so fewer cases are generated and the denominator shrinks. Only the
failure counts are comparable, and only within a row.

Every case carries a control that forwards to the real tool, so a green run
cannot mean the stub mechanism broke everything.

### One prescription I did not follow, with the reason

The worklist's G3-05 criterion says "with `skillscore` absent it must emit
`DEP001`". `check-quality.sh` has no payload channel — stdout is the report
channel and carries one skillscore report or nothing — so there is no findings
array for a rule ID to arrive in. Printing DEP001 on stderr would be a convention
none of its five siblings follow, and the project has already settled the point
twice: `check-frontmatter.sh`'s header says so in as many words, and
`test_f01.sh`'s census deliberately refuses to credit DEP001 to a
`require_tool <tool> false` caller, noting that doing so "made this check demand
DEP001 in the header of a script that cannot emit it, and the header was duly
changed to say something untrue." So the IDs classify its exits in its header,
exactly as in `check-frontmatter.sh`, and nothing prints them. The invariant the
criterion is defending — the script never reports a verdict over a source it could
not read — holds, and is pinned by 39 cases.

### One deliberate widening, stated

`check-frontmatter.sh` now requires `jq`. Reading what `skill-validator` answers
needs the guard's payload predicate, and that predicate answers with jq. jq is
already a prerequisite of this skill: `audit-report.sh` requires it, both `--json`
modes require it, and `SKILL.md`'s own pipeline pipes into it. Stated in the
script's header rather than left implied.

---

## C4 — one sentence asserting an exit contract over scripts that do not share one · DONE

`audit-report.sh` **keeps exiting 0**, including over a failed audit, per the
user's decision. No exit status changed anywhere except where a script previously
leaked a tool's status outside its own documented set.

The doc now **derives**: a row per script, copied from that script's own
`# Exit codes:` header, with `tests/test_f01.sh` comparing every row against that
header — the same mechanism as the rule-ID census. Both directions are checked, so
neither a new script nor a deleted one can leave the table quietly wrong.

| Script | Exit codes |
|---|---|
| `check-frontmatter.sh` | 0=pass, 1=spec failure, 2=policy failure, 3=execution error. |
| `check-paths.sh` | 0=pass, 1=path failure, 3=execution error. |
| `check-structure.sh` | 0=pass, 1=path failure, 2=policy failure, 3=execution error. |
| `check-quality.sh` | 0=report produced, 3=execution error. |
| `audit-report.sh` | 0=report generated, 3=execution error. |

The doc states in those words that `audit-report.sh "$skill" && echo PASS` prints
PASS for a skill that failed every check, and directs a caller to
`jq -e '.summary.passed'`. Cases pin all three: the exit stays 0 over the
`frontmatter-bleed` fixture, `summary.passed` is false, and the doc says both.

The rule-ID registry moved to its own line under its own anchor (`^Rule IDs:`). It
shared one sentence with the exit-code claim, which is how a single line came to
carry two unrelated contracts.

### Non-vacuity

The old single-contract sentence restored → **932 passed, 9 failed**: five rows
reported `<no row>` against their script's header, the table read as an empty set,
the rule registry read as empty (proving the anchor split is load-bearing), and
the "&& echo PASS" warning was absent.

---

## Verification gate

```
bash 3.2 (3.2.57)         bash 5 (5.3.15)
test_harness.sh    46/0   test_harness.sh    46/0
test_skill.sh      35/0   test_skill.sh      35/0
test_walk.sh       20/0   test_walk.sh       20/0
test_f01.sh       941/0   test_f01.sh       941/0
test_f02.sh       272/0   test_f02.sh       272/0
                 1314/0                    1314/0

cd profiler && go build ./...   OK
                go vet ./...    OK
                gofmt -l .      (empty)
                go test -race ./...
  ok  .../profiler       1.296s
  ok  .../profiler/cmd   2.035s
```

`tests/test_harness.sh` (the meta-check that refuses an assertion which cannot
fail) is green on both, was not weakened, and rejected nothing in this diff after
the coverage denominators were added.

No version surface touched: `git diff 71cc866..HEAD` contains no `AdapterVersion`
and no version string, and no Go file.

Attribution grep over every commit message and author/committer field returns
zero.

## Coverage denominators reported by the suites

```
guard primitives examined: json_string cannot_compute tool_answers require_tool
  text_matches text_extract json_document_conforms payload_is_conforming
  quality_report_conforms skill_section skill_frontmatter skill_body
  verdict_guard_ready
tools the guard probes: awk cat grep jq mktemp sed tr wc
tool preconditions walked: 22
present-but-broken cases driven: 45
spec-source disagreement cases driven: 9
quality-source failure cases driven: 7
unreadable-source reason cases driven: 5
scripts whose exit contract was compared: 5
exit-table rows read from SKILL.md: 5
```

## Left for their own clusters (not touched here)

- **G8-04's provenance half** — the draft still states its temp audit's absolute
  path as its provenance. The leak is closed and the path is now legible
  (`draft-rewrite-audit.XXXXXX` under TMPDIR); what the draft should say instead is
  C8's.
- **G8-02, G8-03, G8-05 …** — `skill-rewrite/SKILL.md` documents no `verdict-guard.sh`
  dependency and its preflight still exits 1 for a missing tool. Bringing
  `draft-rewrite.sh` inside the guard makes those doc statements more wrong, not
  less; C8 owns the prose.
- **G6-06** — `skills/skill-audit/SKILL.md`'s `metadata.version` is a version
  surface and out of scope here, though this diff is the largest script change yet
  made to that skill.

---

# Fix round after the PR #10 gate · `0d0c2fb` → `8a72f73`

Eight findings, all closed. Red repro first wherever the defect is behavioural;
the repro commits are labelled `(repro)` and are **expected to be red** — they
are the RED half of the pair, not a CI failure.

**Suites at `8a72f73`, both shells, real counts:**

```
                    bash 3.2.57      bash 5.3.15
test_harness.sh       46 / 0           46 / 0
test_skill.sh         35 / 0           35 / 0
test_walk.sh          20 / 0           20 / 0
test_f01.sh         1164 / 0         1164 / 0
test_f02.sh          350 / 0          350 / 0
                    1615 / 0         1615 / 0
```

```
cd profiler && go build ./...   OK
                go vet ./...    OK
                gofmt -l .      (empty)
                go test -race ./...
  ok  .../profiler       1.467s
  ok  .../profiler/cmd   2.207s
```

1615 from 1314: +301 assertions, every one of them added by this round.

## The shared root, closed

**An external command invoked outside `require_tool`'s coverage leaks its own
status as the script's exit status under `set -e`, with no diagnostic and no
verdict.** Five findings shared it. The rule now holds everywhere: every external
is either inside `require_tool` with its status read where it is called, or it
carries an explicit refusal of its own. Where a tool could not be probed, the
reason is written down rather than the check being dropped.

| ID | Disposition | Commits |
|---|---|---|
| F1 | FIXED | `1f0d9c8` (repro) · `cd3ec8f` |
| F2 | FIXED | `9dff401` (repro) · `7b88156` |
| F3 | FIXED, scope extended | `2641722` (repro) · `c97ab67` |
| F4 · F5 · F6 | FIXED together | `f9cb52b` (repro) · `9a8654f` |
| F7 | FIXED | `d4471d7` (repro) · `8a72f73` |
| G4-04 (F8) | DELIVERED | `5e7a011` (repro) · `f59f5f4` |

### F2 — the mktemp probe compared against nothing

The only arm of `tool_answers` that asked merely whether there *was* an answer,
and the only one whose answer the caller does not discard: it is a path the
drafter opens, writes to, reads back, and publishes as the draft's provenance.
The answer is now compared to the template it was given — the prefix asked for,
then exactly six `[[:alnum:]]`, and not the template verbatim. Deliberately not
checked for absence of `X`: mktemp draws from the alphanumerics, so a valid
answer may contain one.

One comment corrected rather than kept: `-u` was said to create nothing.
Measured on BSD it creates and unlinks, which is why it exits 1 on an unwritable
directory. Right conclusion, wrong reason — and a wrong reason is how the next
reader talks themselves out of a check.

The repo's matrix missed this because its cross product drove a hand-written
list of four scripts. It is now derived from the tree, with an invocation per
script and coverage asserted both ways: **45 cases → 60**, bringing in
`check-quality.sh` and `draft-rewrite.sh`, and therefore `mktemp`.

### F1 — mktemp's status at the call site

Unread, so `mktemp`'s own exit 1 became the drafter's, and 1 is "usage or target
error" in its contract: a broken temp directory was reported as the caller's
mistake. The probe catches the common case a layer earlier and is not a
substitute — a precondition is checked once and the directory can stop being
writable between the check and the call. Driven with the only stub that can
reach the line: one that answers `-u` from the real tool and fails the real
request. The answer is checked beside the status, since a mktemp exiting 0
having printed nothing leaves the path the empty string.

### F3 — scope extended past what the gate named, with evidence

The gate named the spec source. The same root was one column over, and fixing
only the spec source would have been a bolt-on:

> `skillscore` exiting **7** with a conforming report gave
> `quality_score: 99, quality_error: null` — while `check-quality.sh`, reading
> that same payload, exits 3, because skillscore's contract says nonzero means
> it produced no report. One source, two readers, two answers.

All three sources now prove shape as deep as they are read and read their status
against their payload. The policy source always did; the other two were brought
up to it. Both stay soft and `audit-report.sh` still exits 0.

### F4 · F5 · F6, and the structural widening

`date` cannot be probed — a probe compares against an answer already known, and
the point of asking the time is not knowing it; nor is there a portable constant
(`-r 0` vs `-d @0`). So its status is read at the call site. `dirname` and `cat`
run before the guard and before argument parsing respectively, so `dirname` takes
the explicit refusal the load-check already uses (six call sites, plus the
drafter's second `dirname` which was also unread), and `usage()` drops the
heredoc entirely — telling a caller how to invoke a script is not a computation,
so it needs no tool. Reordering `require_tool cat` would have made `-h` refuse to
print help because a tool the help text does not need was broken.

**The gate asked whether widening the exit-table derivation belongs here. Split
answer, and the load-bearing half is here.**

- **Here (done):** the derivation's *behavioural* half — each script's own
  `# Exit codes:` header against the statuses it can be made to emit. Needs no
  doc, covers **both** skills and all six scripts, and is asserted across every
  adverse condition the suite drives. This is what closes the class; it is the
  half whose absence let F1 and F6 through.
- **Not here (C8 / 0.5.0):** the *doc-table* half for `skill-rewrite`. It needs a
  table in `skills/skill-rewrite/SKILL.md`, and that file's prose is C8's — it is
  already known wrong about the guard dependency and the preflight exit. Adding a
  derived table to a doc another cluster is rewriting would collide. The drafter's
  contract is no longer compared against nothing, which was the actual defect.

### F7 — the readme row no cluster owned

"Text mode needs no jq and is unaffected" named exactly the two scripts it was
wrong about: both newly jq-requiring scripts are text-mode-only and one has no
JSON mode at all. Sentence gone; the row now says all five `skill-audit` scripts
require jq, with the reason. The count is derived on a `^Required tools:` anchor
compared against every `require_tool` in both skills, both directions, plus the
prose count against the list's own length. Measured: the scripts require **ten**
tools, the readme claimed three.

### G4-04 — delivered, per the chief of staff's ruling

The entry the lane dropped without a line. Help and usage-error were one path, so
they shared one status. Separated rather than special-cased: a `help` arm beside
`version`, and `parseFlags` distinguishing `flag.ErrHelp` from flags that do not
parse — placed in `parseFlags` because both subcommands ask it the same question.

```
profiler -h          1 -> 0      no args               1 (unchanged)
profiler --help      1 -> 0      unknown command       1 (unchanged)
profiler probe -h    1 -> 0      unknown flag          1 (unchanged)
profiler capture -h  1 -> 0      capture read nothing  2 (unchanged)
```

Three further paths on the same root also move to 0: `probe --help`,
`capture --help`, and the bare word `help`.

## Constraints honoured

- No version surface touched: `git diff 0d0c2fb..HEAD` carries no version string,
  no `AdapterVersion`, no `metadata.version`. The README avoids naming a release
  number in prose for the same reason.
- ~~`skills/skill-audit/SKILL.md` **not touched** — C5 owns it.~~ **Wrong, and
  corrected in the round below — see finding G.** This branch edits that file:
  19 lines at `8a72f73`, 20 at the head of the fix round. The edit is confined
  to the exit-code section (the single-sentence contract replaced by a
  per-script table, plus the generator paragraph and one word). It is
  legitimate and was ruled so at the gate. **C5 must read this corrected line,
  not the struck one:** the exit-code section of that file is taken; line 5 and
  everything else in it are still C5's.
- Attribution grep over every commit message, author and committer field in
  `71cc866..HEAD`: **zero**. Author and committer `imagineux <imagineux@gmail.com>`
  throughout.
- Stubs are one directory holding one file, prepended to the real PATH. Nothing
  installed, mirrored, uninstalled, or written into a live config directory.
- bash 3.2 and 5 both green. No `local a=$1 b="$a"` introduced.

## Still left for their own clusters

Unchanged from above — G8-04's provenance half, G8-02/03/05, G6-06. Plus:

- **The exit-table doc half for `skill-rewrite`** — see the split above. C8 or
  0.5.0, with the rest of that skill's prose.

---

# Fix round — PR #10 confirming pass (A1–A4, B, C, D, E, F, G)

`fix/0.4.3-body-guard-exits`, `8a72f73` → `523b061`, twelve commits, red repro
first for every finding.

**All five suites green on bash 3.2.57 and 5.3.15: 2307 assertions, 0 failed.**
f01 1856, f02 350, harness 46, skill 35, walk 20. `go build ./...`, `go vet
./...`, `go test -race -count=1 ./...` and `gofmt -l .` all clean. f01 takes
2m15s, from 1m30s.

## The verdict I accepted

The rule was right and was applied to three instances instead of to the class.
`require_tool` answers "is this tool present and does it answer" at one moment;
a call site is a different moment. Every one of the twenty-four unread call
sites was a place where that difference mattered, and every existing break mode
failed the probe, so `require_tool` stopped each script before a single call
site ran. The suite was structurally blind, exactly as the gate said.

The gate counted 23 call sites. The real count is **24**: 16 jq + 1 sed + 7 cat.
The 23 is an arithmetic slip in the gate's own headline; its per-finding lists
enumerate 24, and all 24 are covered.

## The root repair

One rule, applied everywhere: **an external's status is read where it is called,
and the sentence names that external.** The guard already made this argument for
grep, in `text_matches`/`text_extract` — "asked here once rather than at seven
call sites, where it would be forgotten at one of them". It is now made for the
rest.

| What | Where | Covers |
|---|---|---|
| `jq_answer` — new guard primitive, reads jq's status, names jq, assigns rather than echoes | `verdict-guard.sh` | 16 jq sites: `check-frontmatter.sh:115`; `audit-report.sh:195,278,279,303,317`; `check-paths.sh:167,170`; `check-structure.sh:194,195,207,210,211,212,241,244` |
| `skill_read` — the section read, awk's status read once, awk named | `verdict-guard.sh` | 4 callers that each wrote the sentence privately and named the SKILL.md |
| `wc` read on its own, named, `tr` deleted | `check-structure.sh` | the count was read through a pipe into `tr -d`, so under pipefail either tool gave the same status and the sentence could name neither |
| `sed` deleted | `check-frontmatter.sh:128` | A2 |
| `draft_append` + the trap owning the draft | `draft-rewrite.sh` | 7 cat writes, plus the partial-draft invariant |
| `json_document_conforms` asks jq to *say* which of the three happened | `verdict-guard.sh:386` | A4 |
| `awk` named at the one non-section read | `check-paths.sh:130` | found by the refusal walk |
| the child's stderr no longer captured in text mode | `check-structure.sh:172` | found by the refusal walk |

### A1 — 16 jq sites

`jq_answer <emit_json> <what> <text> <jq-arg>…` sets `answered`. It assigns
rather than echoes for the reason `text_extract` already states: inside `$(…)`
its `exit 3` would kill only the subshell and the caller would read the empty
result as a computed answer.

At head, with a jq that forwards the probe to the real binary and exits 5
otherwise:

```
check-paths.sh --json    exit 5, zero bytes on stdout, zero on stderr
audit-report.sh          exit 5, no report, no diagnostic of its own
```

At `523b061`, the same jq:

```
$ check-paths.sh --json <skill>
exit=3
stdout: {"findings": [{"level": "fail", "rule": "DEP002", "message": "jq could not
close the findings payload (status 5); no verdict was computed"}], "passed": false,
"error": "jq could not close the findings payload (status 5); no verdict was computed"}
stderr: ERROR: jq could not close the findings payload (status 5); no verdict was computed

$ audit-report.sh <skill>
exit=3 stdout_bytes=0
stderr: ERROR: jq could not answer whether the source is one JSON document of the
shape this read needs; no verdict was computed
```

### A2 — sed, deleted rather than guarded

`sed 's/^/SPEC FAIL: /'` prefixed lines, which the shell does itself. Unread it
was fatal in the worst place: a spec failure this script had *already computed*
became exit 5 with nothing on either channel. Measured at head, with a sed that
answers the probe and fails the call:

```
A2: exit=5 stdout_bytes=0 stderr_bytes=0
```

Reading sed's status would have relabelled that loss as DEP002. Not needing sed
means there is nothing to lose. At `523b061` the same case:

```
exit=1  first line: SPEC FAIL: {   stderr: []
```

`tr` went the same way: its whole job was stripping the padding BSD `wc` puts
round its count, which is parameter expansion. Both leave the readme's
prerequisite list, which is derived.

Left deliberately: `tool_answers` keeps its `sed` and `tr` arms. That function
is a table of known answers, not this repo's requirement list — the requirement
list is the `require_tool` call sites, and the census derives from those.

### A3 — the partial draft, reproduced with real tools

At head:

```
$ draft-rewrite.sh -t <skill> -a <existing-but-unreadable-file>
cat: …/unreadable-audit: Permission denied
EXIT=1
6-line REWRITE-DRAFT.md left in the target
```

Two things were wrong and only one of them was the read. The audit report was
read three writes into the document, so it is read before the first byte now,
with cat's status read and cat named. But the claim at `:181-183` — everything
that can stop this script stops it before the first byte — **cannot be made true
by moving reads earlier**, because composing the draft is eight writes and they
come after. What can be made true is the sentence those words stand in for. The
EXIT trap owns the draft from the moment it is opened until the moment it is
finished and removes an incomplete one, so "no draft was written" is now a
property of the design rather than of remembering to check every writer.

No new external was introduced to do it: no `mv`, no temp file, no rename.

At `523b061`:

```
ERROR: cat could not read the audit report at …/unreadable-audit (status 1); no draft was written
EXIT=3
draft present: NO
```

The suite drives this case with real tools and no stub, with a control beside it
(the same file readable → draft written, exit 0, and the draft carries what the
audit said) and an assertion that the file is genuinely unreadable to the user
running the suite, so a run as root fails loudly rather than passing vacuously.

### A4 — the diagnostic names jq

Measured: `jq -se` answers **5 both for "the text is not JSON at all" and for
"jq could not run"**. The old body read any nonzero as "not that shape", so the
two were published as one:

```
check-quality.sh   ERROR: skillscore did not produce a readable quality report
                   + skillscore's entire, perfectly good report dumped to stderr
check-frontmatter  ERROR: skill-validator did not produce a readable spec payload
                   + skill-validator's entire, perfectly good payload
```

The word `jq` appears in neither. The repair is the one the mktemp probe needed
last round: compare the answer to the question. jq is asked to print a token;
`fromjson` inside a `try` is what makes "exactly one document" part of the
claim, so a stream, an empty input and a parse failure are all the same `no`
they were before; anything that is not one of the two tokens is jq failing to
answer.

`if . then` and `last` preserve `-e`'s reading exactly — false and null are
"no", everything else is "yes", and the final output decides. Which documents
conform is not in question here. Verified over the matrix: valid+true, valid+
false, invalid JSON, empty, two-document stream, `null`, a number, a raising
claim, a multi-output claim, and the real payload claims — every answer
identical to the old body's.

At `523b061`, check-quality.sh with a jq that answers only the probe:

```
exit=3 stdout_bytes=0
stderr: ERROR: jq could not answer whether the source is one JSON document of the
shape this read needs; no verdict was computed
```

## The new break mode, and why one was not enough

The gate asked for a mode where the tool answers the probe and fails a later
call. I built it, and it went red on 24 assertions. Then I checked what it could
not reach, and the answer was: everything behind the first unguarded call,
because the script stops at the first one.

So the refusal **walks the run**. One case per question a clean run puts to the
tool, driven as the question it refuses, from the probe's last question to one
short of the last of all. Both bounds are measured, not written down — the probe
count by running `require_tool` against a counting stub (grep asks two questions,
every other probe one), the total by running the script against the same stub.
A call site added to a script joins the walk the day it lands.

**119 refusal positions, plus the 3 probe-failing modes over 18 pairs: 173 cases,
up from 60.** A control sits at the far end of each walk: a stub that answers
every question a clean run asks must still reach the verdict, so no case can be
passing because the stub mechanism breaks the script.

What the walk caught that the single mode did not — eleven failures, two roots:

- `check-paths.sh:130`, the one awk call in this skill that is not a section
  read, so `skill_read` does not cover it. `ERROR: could not read the code
  blocks of …/SKILL.md` over a file that was perfectly readable, awk nowhere.
- `check-structure.sh:172`, text mode capturing the child with `2>&1`. On the
  path where the child reached no verdict the capture is never printed, so the
  child's sentence naming awk or grep was swallowed and the caller was told
  `check-paths.sh exited with unexpected status 3` — **a sibling script blamed
  for a tool**. Six of the eleven arrived through the drafter, which runs
  check-structure.sh in text mode: one capture, three scripts deep.

Both are G3-08's invariant recurring, which is what A4 was. The child's stdout is
still captured in both modes — in one it is a payload to parse, in the other
findings to relay — and its stderr is captured in neither.

The contract each case asserts is stated once, in `assert_refused`, because the
probe-failing modes and the walk are the same contract driven from two places.

## B — the census counts what a script refuses without

RED: `required by a script, absent from the readme: date dirname`.

The readme's criterion is behavioural. The census derived from `require_tool`
sites, which is a mechanism and not that criterion, so the two tools that meet
it through the other mechanism were invisible — and the F4/F5 fix that made them
hard preconditions is the same change that moved them out of sight of the list
that counts them.

The census is now `require_tool` **plus what the suite proved**, and proved is
literal: each name in `unguarded_proved` is recorded beside the assertions that
drove that tool broken and watched the script refuse it, name it, and exit
inside its own stated set. Nine assertions already stood over `date` and
eighteen over `dirname`; nothing carried their result to the list. The census
block moved down the file to sit below its evidence.

Ten tools again, but not the same ten: `sed` and `tr` left because nothing
shells out to them, `date` and `dirname` arrived because something does.

**Boundary, stated rather than implied:** this finds an unguarded tool something
drove, not one nobody thought of. Closing that needs the set of external command
words in each script, which is a shell parser.

## C — three, not four

RED: `SKILL.md says: four / the table says: three`.

Derived on the table's own criterion — a script that can exit with something
other than 0 or 3 is saying something about the skill in its exit status. Both
halves of the sentence pair are checked, so neither number can be right only
because the check read one of them. `english_count` replaces the second private
copy of a number-word list.

The SKILL.md edit is one word, inside the exit-code section.

## D — the header against the behaviour

The gate's claim, reproduced exactly on the pre-fix tree, with SKILL.md's row
edited to agree with the header each time so the existing derivation stayed
green:

```
add  `4=phantom` to check-paths.sh     1246 passed, 0 failed
drop `1=path failure` from it          1246 passed, 0 failed
```

With the witness, the same two mutations:

```
check-paths.sh stated [0 1 3 4] seen [0 1 3]
FAIL: every status a script's header states is one this suite made it emit
  stated but never emitted here: check-paths.sh:4

check-paths.sh stated [0 3] seen [0 1 3]
FAIL: every status this suite made a script emit is one its header states
  emitted but not stated: check-paths.sh:1
```

The witness records every run through the shared harness, so there is no call
site left to forget, and it lives in `harness.sh` — `audit-suites.sh` now
refuses a private copy of `witness_exit` as it does every other harness name.

It found a real hole on its first run: `check-frontmatter.sh` exit 2 is stated
by its header, driven three times over the no-license fixture through the
suite's own merged-channel runner, and witnessed nowhere. That runner records
now.

Final state, and all five rows were true before and after:

```
audit-report.sh      stated [0 3]     seen [0 3]
check-frontmatter.sh stated [0 1 2 3] seen [0 1 2 3]
check-paths.sh       stated [0 1 3]   seen [0 1 3]
check-quality.sh     stated [0 3]     seen [0 3]
check-structure.sh   stated [0 1 2 3] seen [0 1 2 3]
draft-rewrite.sh     stated [0 1 3]   seen [0 1 3]
```

## E — the list is read off the tree, as the comment said

Three lines of argument for deriving the list, followed by the list. The
behaviour was right; the comment is what the next reader trusts. Rather than
correcting the comment to describe the literal, the code now does what the
comment says. Measured with a seventh runnable script dropped into the tree:

```
literal list   present-but-broken cases driven: 72
               FAIL: every runnable script in both skills is driven against a broken tool
               runnable but never driven against a broken tool: check-extra.sh

derived list   runnable scripts read off the tree: 7
               present-but-broken cases driven: 76
```

The membership half of the coverage check goes with the literal: it can no
longer fail, and an assertion that cannot fail is what this suite exists to
refuse. The half that remains can still fail — a script with no invocation that
reaches a verdict — and it now skips rather than taking the suite down at the
assignment under errexit.

## F — a count, not a floor

`-ge 45` was written when the product was 60. Measured, with `require_tool grep`
deleted from one script — one pair, four cases, twenty-one assertions:

```
before   present-but-broken cases driven: 68
         PASS: the present-but-broken cross product was enumerated, not read as empty
         1232 passed, 0 failed

after    FAIL: the present-but-broken cross product ran every one of its 72 cases
         the cross product is 68 cases, and this file says 72
         1231 passed, 1 failed
```

The count is a literal on purpose: deriving it from the same lists the loop
walks would make it agree with any product those lists produce, which is the
property being removed. It is `173` at the head of this round.

The other floors in the file are left alone — each sits beside an exact
comparison that does the real work, so the floor is only refusing an empty read.
Nothing else counts these cases.

## G — the record line, corrected

Corrected in place above. `skills/skill-audit/SKILL.md` **is** touched by this
branch: 19 lines at `8a72f73`, 20 now, all inside the exit-code section. The
edit is legitimate and was ruled so. **C5 reads the corrected line.**

## Nothing weakened, accounted assertion by assertion

f01 was 1164 at `8a72f73` and is 1856 at `523b061`. Diffed label by label:
**37 labels gone, 729 new, 1164 − 37 + 729 = 1856.**

The 37, every one of them:

| n | What | Why |
|---|---|---|
| 16 | `check-frontmatter.sh, sed …` — 5 assertions × 3 modes, plus the control | `sed` is not used by any script any more |
| 19 | `check-structure.sh, tr …` — 6 assertions × 3 modes, plus the control | `tr` is not used by any script any more |
| 1 | `every runnable script in both skills is driven against a broken tool` | **relabelled**, not removed: it asked two questions, one of which became true by construction when the list became derived. The half that can still fail is now `every runnable script in both skills has an invocation that reaches a verdict` |
| 1 | `the present-but-broken cross product was enumerated, not read as empty` | **relabelled** into `…ran every one of its 173 cases`, which is strictly stronger — the old one passed a 25% shrink |

The 729 new are the walk (113 further cases at 5–7 assertions each), the six
guard-drop assertions `jq_answer` and `skill_read` earn from
`verdict_guard_ready`'s own list, the walk's per-pair bound and far-end control
(18 + 18), the drafter's unreadable-audit case with its two controls, and the
new derivations for B, C, D, E and F.

**No assertion was deleted, loosened, or made unable to fail.** Every removal is
either a case for a tool that no longer exists in these scripts, or a label
replaced by a strictly stronger one.

## Constraints honoured

- **No version surface touched.** `git diff 8a72f73..HEAD` carries no version
  string, no `AdapterVersion`, no `metadata.version`.
- `skills/skill-audit/SKILL.md`: one word, inside the exit-code section. Line 5
  and everything else untouched, for C5.
- Attribution grep over every commit message, author and committer field in
  `71cc866..HEAD`: **zero**. Author and committer `imagineux <imagineux@gmail.com>`
  throughout.
- Stubs are one directory holding one real file, prepended to the real PATH. No
  symlink mirror. Nothing installed, reconfigured, uninstalled, or written into
  a live config directory.
- Every harness run under `bash` explicitly; zsh's lack of word-splitting bit
  one measurement early and was caught by the control coming back 127.
- Commit and push per finding, red repro first, twelve commits.
- Nothing merged, nothing tagged.

## Left for their own clusters, unchanged

G8-04's provenance half, G8-02/03/05, G6-06, and the exit-table doc half for
`skill-rewrite`.
