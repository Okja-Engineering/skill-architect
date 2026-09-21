# PR #10 rebase onto `a06af3f` — steward closeout

Written 2026-09-19. Mandate: rebase PR #10 `fix/0.4.3-body-guard-exits` onto the moved
`origin/main` and re-verify live. **No merge and no tag were performed. The user merges.**

## Pinned state

| Fact | Value |
| --- | --- |
| PR | #10 `fix/0.4.3-body-guard-exits` -> `main` |
| Head before | `523b0616ec9c6a0692484bf03b60214a007862d7` (33 commits, base `71cc866`) |
| **Head after** | **`16552367338c20c47e3b3bb2496d5e7370eaf711`** (34 commits) |
| Base | `origin/main` = `a06af3fb0e14a5a08e1ced88f5861a413e1075eb` (PR #9's merge) |
| Old merge-base | `71cc8667ba2aac1550ebb0dce8418b81a1bd7319` |
| `mergeable` / `mergeStateStatus` (per-PR, live) | **`MERGEABLE`** / **`CLEAN`** — was `CONFLICTING`/`DIRTY` |
| CI at this head | **success** — run `35455746703`, all 19 steps green, `headSha` == this head, read live |
| Review threads (live, paginated) | `totalCount` 0, nodes 0, `hasNextPage` false — a real zero |
| Reviews / issue comments / `reviewDecision` | 0 / 2 / null |
| Merges in range | 0 — linear, as the ruleset requires |
| Suites at this head | **2405 passed, 0 failed** on bash 3.2.57 *and* bash 5.3.15 |

The conflict is gone and the ruleset is satisfied: read per-PR and live, not off a list
endpoint, at `16552367` — `mergeable` `MERGEABLE`, `mergeStateStatus` `CLEAN`.

## The union, and what was recovered from each side

One file conflicted: `skills/skill-rewrite/scripts/draft-rewrite.sh`. The rebase stopped
exactly once, at commit 7 of 33 (`41ef533 C3: bring draft-rewrite.sh inside the guard`),
in three hunks. The other 26 commits replayed clean, including the four later commits that
also touch the drafter.

**Neither side was taken whole.** Confirmed against the brief's measurement rather than
assumed: a side-pick of PR #10 leaves PR #9's repairs on the floor.

### Recovered from PR #9 (now on `main`) — all four, verbatim in substance

| Recovered | Where it is now |
| --- | --- |
| `require_value` — the value-reading helper, and both call sites that use it | auto-merged; kept |
| The `-a` refusal: `Audit report not found:` + `Omit -a to have the structural checks run instead.` | hunk 2, restored ahead of PR #10's `require_tool` block |
| The `provenance` variable and `Generated from: $provenance` in the draft header | hunk 3 + auto-merge; PR #10's `Generated from audit report: $audit_report` dropped |
| The folded condition split: `if [[ -z "$audit_report" ]]`, not base's `[[ -z … \|\| ! -f … ]]` | hunk 3; this is what makes the `-a` refusal reachable |
| PR #9's header prose: what the script writes, the usage-error enumeration, "the audit's own verdict is not this script's exit status" | hunk 1, unioned into PR #10's header |
| PR #9's `CDPATH=`/`--` sibling-resolution rationale | hunk 2, rewritten to point at the top of the file, where PR #10 moved the resolution |

### Kept from PR #10 — every mechanism, none weakened

`verdict-guard.sh` load check (before and after `source`) and the `exit 3` / DEP001 / DEP002
contract · the `cleanup` EXIT trap owning both the temp audit and the partial draft ·
`printf`-based `usage` · `require_tool awk grep cat mktemp` with `basename` dropped for
`${target_skill%/}` / `${…##*/}` · `mktemp` with an explicit `${TMPDIR:-/tmp}` template, its
status *and* its answer read · the status-checked loop over the two sibling checks ·
`skill_body` + `text_matches` for the three heading probes · `audit_text` read through
`cat --` with its status read · `draft_append` reading `cat`'s status at one site ·
`draft_incomplete`.

### Two supersessions, deliberate

- **PR #9's `trap 'rm -f "$audit_report"' EXIT` is not in the result.** PR #10's `cleanup`
  trap, installed at the top of the file, holds the same invariant on strictly more paths
  (it also removes a partial draft). Recovery by supersession, not loss.
- **PR #9's clause "or worse, one that is still there, since nothing removed it"** was
  dropped from the `provenance` comment. Under the union the trap always removes that file,
  so the clause describes something now unreachable. A false clause the union itself would
  have introduced is not one to ship.

### One shape change inside PR #10's own repair

PR #10 replaced the two literal check invocations with `"$skill_audit_root/scripts/$audit_check"`
inside a loop. The loop is kept — its status reading is the repair — but the two names are
now declared in full at the one loop head:

```bash
for audit_check in \
  "$skill_audit_root/scripts/check-frontmatter.sh" \
  "$skill_audit_root/scripts/check-structure.sh"; do
```

That keeps the code/document census (item 5 below) decidable by reading the file, instead of
pinning the suite's extractor to one loop's spelling.

## The coupled assertions: the brief's five, and a sixth

The brief gave five as the denominator and warned that PR #9's own hand-off list named only
three of them. **Measured at the rebased head, there were six.** All six are now green,
verified by name in the suite's output.

| # | Assertion | Why it was red | How it was cleared |
| --- | --- | --- | --- |
| 1 | `without skill-validator draft-rewrite.sh exits 0, …` | PR #10 makes it exit **3** | `SKILL.md:42` now reads `draft-rewrite.sh` exits `3`. The label is generated from the document, so it now reads "exits 3" |
| 2 | `… the drafter writes a draft anyway, as the SKILL.md warns it does` | it no longer does | turned around: `… writes no draft, as the SKILL.md says it does not` |
| 3 | `… the check's diagnostic is the draft's Current state, as the SKILL.md warns` | the diagnostic now reaches stderr | turned around: `… the missing tool is named on stderr, as the SKILL.md says` |
| 4 | `the SKILL.md declares every tool this skill needs, and none it does not` | measured required set is seven, declared was one | `Required tools:` line rewritten; census denominator widened (below) |
| 5 | `the sibling checks the drafter runs are the ones the SKILL.md shows it running, and no others` | drafter yielded only `verdict-guard.sh`; document yields the two checks | loop head declares both checks in full; census skips the guard it *sources* |
| **6** | **`the drafter's header registers every status it can exit with, and no status it cannot`** | **not in the brief's five.** emitted `0 1 2 3` vs registered `0 1 3` | the census reads code lines only |

### Item 6, in full, because it was not on the list

`statuses_emitted()` greps the drafter for `exit <N>` **without skipping comments**. PR #10's
new header explains, in prose, the statuses the script used to exit with:

> `So a broken \`cat\` made \`-h\` exit 2 where this contract says 0, and the other two exit 2 where it says 1`

Three such comment lines. The census read those as emitted statuses, so the assertion
demanded the header register `2` — a status the code cannot reach. Registering it would have
produced exactly the false header the assertion exists to refuse, which is the defect class
this release closes. Repaired at the root rather than by rewording the comment: the census
now reads code lines only, and the suite records why. It is the same distinction the tool
registry already draws with backticks — a paragraph may discuss what it does not declare.

### Item 4's denominator was wrong, not just its answer

The brief framed item 4 as "PR #10 adds `require_tool` for awk, grep, cat and mktemp, and
drops basename". That is what the drafter does, but it is not what the assertion measures.
`candidate_tools()` read **only skill-audit's** `require_tool` sites, on the stated premise
that they are "the whole set that can stop this skill for want of a tool". PR #10 falsifies
that premise: the drafter now states preconditions of its own, and `cat` and `mktemp` are
asked for nowhere else. So the denominator is widened to the union of skill-audit's sites and
the drafter's, and the required set was **measured by masking**, not read off the source:

```
candidates: asks awk cat grep jq mktemp skill-validator skillscore wc
required  : awk cat grep jq mktemp skill-validator wc
declared  : awk cat grep jq mktemp skill-validator wc
```

`jq` and `wc` are required through the sibling checks (`check-frontmatter.sh` has
`require_tool jq false` — unconditional, not `--json`-gated; `check-structure.sh` has
`require_tool wc "$json_output"`, and `$json_output` is false on this path). PR #9's sentence
saying nothing on this path needs `jq` was therefore **false after the rebase** and is gone.
`skillscore` is still not needed; measured, not assumed.

### Item 5: a library it sources is not a check it runs

The drafter now names `verdict-guard.sh` under the same `$skill_audit_root/scripts/` prefix
as the two checks, because it sources the guard so it can refuse in the guard's own words.
Counting it as a check it *runs* would require the `SKILL.md` to show a reader invoking the
guard, which is not a command anybody invokes. It is skipped for the same reason
`candidate_tools()` already skipped it. The dependency itself is unchanged and still held
where it was, by `guard_is_load_bearing` — removing the guard from a copy of both skills and
watching the drafter stop.

## The rewritten prerequisites paragraph

`SKILL.md:38`:

> Required tools: `awk`, `cat`, `grep`, `jq`, `mktemp`, `skill-validator`, `wc`.

`SKILL.md:40`:

> Stage 1 and `draft-rewrite.sh` both run `skill-audit`'s `check-frontmatter.sh`, which
> validates the spec with `skill-validator`. Install it with
> `brew install agent-ecosystem/tap/skill-validator`, because without it neither path will
> draft for you:

`SKILL.md:42`:

> Without `skill-validator`: `check-frontmatter.sh` exits `3`; `draft-rewrite.sh` exits `3`.

`SKILL.md:44` — the paragraph the rebase is about:

> Stage 1 runs the check directly, so it exits 3 with `required tool not found:
> skill-validator` on stderr and reports no verdict rather than one it could not compute.
> `draft-rewrite.sh` is inside that same guard now: it reads each check's exit status instead
> of discarding it, captures each check's stdout on its own instead of merging the check's
> stderr into it, and a check that reached no verdict stops the draft rather than becoming
> its content. So it exits 3 too, with `required tool not found: skill-validator` on stderr,
> and it writes no `REWRITE-DRAFT.md` — the partial one it had opened is removed on the way
> out, so there is no half-draft in your skill directory either. **What earlier releases
> warned about here — a drafter that exited 0 and presented `required tool not found:
> skill-validator` as the draft's own `Current state` — is closed, not something to work
> around.** A draft you are handed is a draft built from an audit that ran.

`SKILL.md:46` — replaces the sentence that said `jq` was `--json`-only:

> The other six required tools refuse the same way, so this is the skill's behaviour and not
> one tool's special case. `awk`, `grep` and `cat` are what this skill computes with, and
> `mktemp` holds the audit when you do not hand it one with `-a`; `jq` and `wc` are
> preconditions of the two sibling checks. Mask any one of the seven and `draft-rewrite.sh`
> exits 3 naming that tool and leaves no draft. Nothing on this skill's path needs
> `skillscore`, which scores quality this skill does not run; add it if you extend the stages
> to use it.

`SKILL.md:48` — gained one clause, so the census exclusion above is readable:

> … `draft-rewrite.sh` loads that guard too, which is how it refuses in the guard's own words
> rather than in the shell's; it sources the guard rather than running it, which is why the
> guard is not one of the checks named above. A pruner who removes the guard as an unused
> file, or who installs this skill alone, gets **no draft at all**. …

One more claim was corrected because the same behaviour change falsified it — `SKILL.md:93`,
the `Current state` inventory bullet, said the section is the checks' "stdout and stderr
merged into one stream" and that "the merge is why a missing tool shows up here as the
skill's own state". PR #10 captures stdout alone. It now reads:

> `Current state` — the Stage 1 checks' stdout, captured one check at a time, or the contents
> of the report given with `-a`. Their stderr is not folded in: a check that could not compute
> a verdict stops the draft instead of appearing here as a finding about the skill, so nothing
> in this section is a diagnostic about the check itself; see Prerequisites.

Everything above landed in **one commit on top of the 33** (`1655236`), not folded into a
replayed commit: `SKILL.md` did not conflict, so its update is the rebase's consequence
rather than part of any original commit. The 33 are otherwise unsquashed and unreworded;
only `41ef533` changed, and only by its conflict resolution.

## Three guards were touched. All three were proved still to discriminate.

Each mutation was applied at the rebased head, run, and reverted.

| Mutation | Result |
| --- | --- |
| A real `exit 2` in code (`Unknown option:` path), header unchanged | **FAIL**: `emitted : 0 1 2 3` / `registered: 0 1 3` — the narrowed census still catches a real status |
| `check-structure.sh` dropped from the drafter's loop head | **FAIL**: `the sibling checks the drafter runs are the ones the SKILL.md shows it running, and no others` |
| `Required tools:` reverted to the pre-widening five | **FAIL**: `required: awk cat grep jq mktemp skill-validator wc` vs `declared: awk grep jq skill-validator wc` — the widening is load-bearing, not cosmetic |

## Live verification at `1655236`

### Suites — six, not seven, and the total is 2405, not 2307

| Suite | bash 3.2.57 | bash 5.3.15 | PR #10 @ `523b061` |
| --- | --- | --- | --- |
| `test_harness` | 50 / 0 | 50 / 0 | 46 |
| `test_skill` | 35 / 0 | 35 / 0 | 35 |
| `test_walk` | 20 / 0 | 20 / 0 | 20 |
| `test_rewrite` | **94 / 0** | **94 / 0** | *(not in tree)* |
| `test_f01` | 1856 / 0 | 1856 / 0 | 1856 |
| `test_f02` | 350 / 0 | 350 / 0 | 350 |
| **Total** | **2405 / 0** | **2405 / 0** | 2307 |

All `rc=0`. **4810 assertions across both shells, zero failures.**

**There are six suites, not seven.** `tests/test_*.sh` and `.github/workflows/ci.yml` agree
on six. The seventh is PR #8's `tests/test_install.sh`, which is not in yet — as the brief
itself notes. The "seven" in the brief is a count carried over from `pr9-closeout.md`, which
measured the *combined* #8+#9 tree.

**The +98 is fully accounted for, and one of the brief's numbers was stale.**

- **+94** — `tests/test_rewrite.sh`, which arrived on `main` with PR #9.
- **+4** — `test_harness` 46 -> 50. Its suite list is a glob and it asserts four things per
  suite; the glob gained a sixth suite. Measured at both ends, not inferred.
- `test_skill`, `test_walk`, `test_f01`, `test_f02` are **unchanged** at 35 / 20 / 1856 / 350.
  PR #10's counts survive the rebase exactly.

**`test_rewrite` is 94, not the 70 in `pr9-closeout.md`.** That record pinned PR #9 at
`483903a`; PR #9 went on to `3df3207` before merging, adding 474 lines to that suite
(`git diff --stat 483903a a06af3f`). Confirmed by running the suite at `origin/main` in a
throwaway worktree: **94 passed, 0 failed**. So 94 is `main`'s number, not something this
rebase created — and my coupling commit changed no assertion count: the suite was 94 before
it (88 passed, 6 failed) and 94 after (94 passed, 0 failed).

### Go gate

`go build ./...` OK · `go vet ./...` OK · `go test -race -count=1 ./...` OK
(`profiler` 1.546s, `profiler/cmd` 2.249s) · `gofmt -l .` empty.

### The meta-check still has teeth — red, then green

A deliberately vacuous assertion was injected into `tests/test_walk.sh` before its
`harness_summary`:

```bash
assert "a deliberately vacuous assertion injected to prove the meta-check" true
```

`tests/test_harness.sh` reddened, **naming the file and the line**:

```
tests/test_walk.sh:182: verdict is the constant command 'true', so this assertion cannot fail: assert "a deliberately vacuous assertion injected to prove the meta-check" true
FAIL: tests/test_walk.sh has no assertion that cannot fail, and no harness of its own
49 passed, 1 failed        rc=1
```

Injection removed; `test_harness` back to **50 passed, 0 failed on both shells**, `test_walk`
back to 20/0, working tree clean.

### The coupling's subject, re-proved directly

`skill-validator` masked through the repo's own `tests/lib/masked-path.sh`, used read-only —
the file is unmodified, nothing was installed, moved or written into the farm, and the farm
is the harness's own symlink mirror.

| Shell | masked | exit | draft | stderr |
| --- | --- | --- | --- | --- |
| bash 3.2.57 | `skill-validator` | **3** | **absent** | `ERROR: required tool not found: skill-validator` |
| bash 3.2.57 | *(control, nothing masked)* | 0 | PRESENT | — |
| bash 5.3.15 | `skill-validator` | **3** | **absent** | `ERROR: required tool not found: skill-validator` |
| bash 5.3.15 | *(control, nothing masked)* | 0 | PRESENT | — |

The drafter **refuses**, and `SKILL.md:42` says `draft-rewrite.sh` exits `3` while
`SKILL.md:44` says it writes no draft. The same sweep over all nine candidates returned
`awk cat grep jq mktemp skill-validator wc` refusing at exit 3 and `skillscore`/`asks`
drafting fine — the measurement behind the `Required tools:` line.

### CI at the new head

Run `35455746703`, **conclusion `success`**, `headSha` `16552367338c20c47e3b3bb2496d5e7370eaf711`
— the same SHA this report is pinned to, not a predecessor. **All 19 steps green**, including
each of the six suite steps and the three profiler steps:

```
Run verification-harness tests                       success
Run skill layout tests                               success
Run end-to-end walk tests                            success
Run skill-rewrite drafter and documentation tests    success
Run F01 format/policy separation tests               success
Run F02 unified audit report tests                   success
Check profiler formatting / Build and vet / tests    success
```

With CI reported, `mergeStateStatus` moved `BLOCKED` -> **`CLEAN`**.

## Disposition ledger

| Item | Disposition |
| --- | --- |
| `draft-rewrite.sh` conflict, PR #9 x PR #10 | **RESOLVED** — union, three hunks, one stop in 33 commits |
| Coupled assertions 1-3 (the guard's status, the draft, the diagnostic) | **RESOLVED** — `SKILL.md` rewritten, two assertions turned around |
| Coupled assertion 4 (tool declaration) | **RESOLVED** — line rewritten to the measured seven; census denominator widened, and the widening proved load-bearing |
| Coupled assertion 5 (sibling-check census) | **RESOLVED** — loop head declares both checks; sourced guard excluded with the reason stated |
| **Coupled assertion 6 (exit-status census) — not on the brief's list** | **RESOLVED** — census reads code, not prose; proved still to redden on a real `exit 2` |
| `SKILL.md:93` "merged stream" claim | **RESOLVED** — falsified by the same change; corrected |
| PR #9's `jq`-is-`--json`-only sentence | **RESOLVED** — false after the rebase; removed |
| PR #9 clause "still there, since nothing removed it" | **RESOLVED** — unreachable under the union's trap; dropped |
| Review threads on PR #10 | **NONE** — 0 live-verified, `totalCount` == nodes, `hasNextPage` false |
| P3 A5 — `rm`'s status unread in the drafter's EXIT trap | **DEFERRED, untouched by design** — knowingly shipping per `final-8-10.md`; `cleanup()` is byte-identical to PR #10's |
| P3 — false durability clause in test prose | **DEFERRED** — PR #8's, not on this branch |
| P3 — the item that arrived in the round closing its own class | **DEFERRED** — recorded in `final-8-10.md`; left |
| Adapter version bump to 0.4.3, release note for `jq` | **NOT MINE** — no version surface touched; `git diff --name-only origin/main HEAD` matches nothing under CHANGELOG / RELEASE_NOTES / version |
| Merge of #10 | **NOT DONE — the user merges.** Brief withheld merge and tag authority. PR is `MERGEABLE`/`CLEAN` and ready |

## For the next rebase: PR #8 will conflict here, and the answer is the union

Recorded so it is not rediscovered. After PR #10 lands, **PR #8 conflicts with it in
`tests/lib/harness.sh` and `tests/lib/audit-suites.sh`** — PR #10's exit-witness repair added
a name to the same two lists PR #8 added its refusal verb to. Both names, measured at both
heads:

| List | PR #8 adds | PR #10 adds |
| --- | --- | --- |
| `tests/lib/audit-suites.sh`, the awk `BEGIN` harness-name list (the private-copy detector) | `require` | `witness_exit` |
| `tests/lib/harness.sh`, the `harness_ready` readiness loop | `require` | `witness_exit` |

**Resolution is the union — both names in both lists.** Verbatim:

```awk
n = split("assert assert_value require quietly witness_exit harness_init harness_exit harness_summary harness_ready", list, " ")
```

```bash
for n in harness_init harness_exit assert assert_value require quietly witness_exit \
         harness_summary; do
```

**Picking a side silently disables a guard.** Out of the readiness loop, a suite stops
proving the harness loaded that primitive. Out of the auditor, a suite's private copy of that
name stops being detected. Neither produces a failure at resolution time, which is why it has
to be written down.

Still open from `pr9-closeout.md` and unaffected by this rebase: after both #8 and #10 land
there will be **seven** suites while `test_harness.sh`'s floor reads `-ge 5` on this branch
and `-ge 6` on PR #8's. **Whoever rebases second should bump it to 7.** No conflict will
prompt for it. This rebase did not touch the floor — it is PR #8's line.

## Constraint compliance

- All work in the existing worktree
  `…/scratchpad/wt-lane-a`, on `fix/0.4.3-body-guard-exits`. **The primary working tree was
  never touched** — no checkout, reset, clean, stash or rebase there; only reads, `gh`, and
  this report. Its uncommitted 0.5.0 work is intact.
- One throwaway worktree at `origin/main` (`…/scratchpad/rb-main`) to measure `main`'s own
  `test_rewrite` count rather than trust a prior report. Removed.
- Pushed once: `523b061..1655236`, **`--force-with-lease=fix/0.4.3-body-guard-exits:523b061`**,
  never bare `--force`. Lease verified to match the remote ref before pushing.
- `imagineux <imagineux@gmail.com>` author **and** committer on all 34 commits, verified by
  `git log --format='%an <%ae>|%cn <%ce>' | sort -u` returning exactly one line. Grep for
  `co-authored-by|generated with|claude|anthropic|🤖|ai-assisted` across all 34 commit bodies
  and identities returned **0** before the push.
- 0 merge commits in `origin/main..HEAD` — linear, as the ruleset requires.
- Not squashed, not reworded: 33 original commits plus one new one on top.
- No version surface touched.
- Both bash 3.2.57 and 5.3.15 exercised for every suite, the meta-check, and the masked-tool
  re-proof. No `local a=$1 b="$a"` was written.
- **No PATH mirror was built and no stub was written into one.** The only masking used is the
  repo's own `tests/lib/masked-path.sh`, unmodified, which symlinks the real PATH minus one
  name and writes nothing. Nothing was installed or reconfigured; no live config directory was
  written. `mktemp -d` was always given an explicit `${TMPDIR:-/tmp}` template.
- This report written with the `Write` tool and read back to confirm it exists.
