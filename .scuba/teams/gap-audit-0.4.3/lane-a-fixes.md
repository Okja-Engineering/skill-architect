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

Each commit reverted in isolation, tests kept:

- `require_tool`'s self-test clause alone removed → **727 passed, 99 failed**.
- `check-frontmatter.sh` alone reverted → **842 passed, 25 failed**.
- `audit-report.sh` alone reverted → **262 passed, 10 failed** (f02).
- `check-quality.sh` alone reverted → **894 passed, 32 failed**.
- `draft-rewrite.sh` alone reverted → **28 passed, 7 failed** (skill).
- cleanup trap alone removed → the leak assertion reports two files left behind.

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
