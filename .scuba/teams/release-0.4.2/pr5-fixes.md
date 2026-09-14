# PR #5 — gate fixes

Branch `release/0.4.2-version`, worktree
`/Users/matthewvandusen/Development/Auraprix/skill-architect-wt/release-0.4.2-version`.
Gate report: `pr5-gate.md`. Base for the prose diff: `2ed34b8`. Release commit
under review: `13b3141`. Head after fixes: `2488ff0`.

Ten commits, one per finding, each pushed as it landed.

## Standing invariants, re-checked before every push

`scratchpad/guard.sh`, green before all ten pushes and at head:

1. `CHANGELOG.md` / `RELEASE_NOTES.md` vs `2ed34b8` are **pure insertions** — 0
   deleted or context-changed lines. 41 and 16 insertions, unchanged throughout.
2. Everything from the `0.4.1` heading down is **byte-identical** to `2ed34b8`
   (`7bb2bf71…` / `4e0e1033…`, unchanged throughout). 0 edits touch a 0.4.1
   heading.
3. **No AI attribution** in any new commit's metadata or added lines — 0 hits.
4. Version surfaces untouched: five manifests `0.4.2`, `AdapterVersion "0.4.2"`,
   `profiler version` prints `profiler 0.4.2`.
5. Author and committer `imagineux <imagineux@gmail.com>` on all ten.

Total diff of this round: `CHANGELOG.md` 22, `RELEASE_NOTES.md` 4,
`docs/profiler-spec.md` 8, `profiler/otlp.go` 17.

## Dispositions

| # | Verdict | Commit |
|---|---------|--------|
| F1 | REAL | `1f8a4eb` |
| F2 | REAL | `e51cba4` |
| F3 | REAL | `57e1e86` |
| F4 | REAL, **+1 the gate passed** | `4b4941c` |
| F5 | REAL | `6b8845b` |
| F6 | REAL, **gate's own diagnosis half wrong** | `bb956eb` |
| F7 | REAL | `691e9c0` |
| F8 | REAL | `f926d27` |
| F9 | REAL — **proven**, not removed | `4507aed` |
| F10–F12 | REAL (consistency) | `2488ff0` |

Nothing was classified INVALID. One gate item (the adjacent, out-of-diff one) was
measured and found **true**; see below.

### F1 — `counterAccumulator` "clamped to zero" — `1f8a4eb`

Measured by driving `add`:

    cumulative unplaced, value -500  -> total = 0
    cumulative run,      value -500  -> total = 0
    delta,               value -500  -> total = -500
    delta, 100 then -500             -> total = -400

Delta calls `addSaturating` with no clamp. Comment now describes every branch.
**No behaviour change.** Its CHANGELOG repeat corrected in the same commit.

**No test added, deliberately.** The prose asserts "66 profiler test functions and
230 subtests", verified true. A new test function or subtest would falsify a
verified claim in the prose being corrected. Evidence is the out-of-tree probe.

### F2 — "eight fixtures" — `e51cba4`

`HEAD 55 ; 2ed34b8 39 ; delta 16`. `bd32b26` added all 16; `5c847e1` and
`13b3141` added 0; none removed. All 16 named in the testdata README, so only the
number was wrong.

### F3 — "exactly one deviation" — `57e1e86`

Reference `protojson` v1.36.12 / OTLP proto v1.11.0:

    {"aggregationTemporality":2}    err=<nil>
    {"aggregationTemporality":"2"}  err=invalid value for enum field
    {"aggregationTemporality":"1"}  err=invalid value for enum field

This adapter: `"2"` → value=2 declared. `cumulative.ndjson` ships the quoted form.
Note now enumerates both, names each direction, and says the count is an
enumeration of what was measured.

**Scope call:** `docs/profiler-spec.md` was not in PR #5's diff (the note came in
with `bd32b26`). Pulled in because the CHANGELOG entry points at it.

### F4 — "all three carry a `passed: false` payload" — `4b4941c`

Tools genuinely absent (PATH rebuilt as a symlink mirror minus one tool):

    check-structure.sh --json,   jq masked  exit=3 stdout=<payload> stderr=ERROR: …
    check-paths.sh --json,       jq masked  exit=3 stdout=<payload> stderr=ERROR: …
    check-frontmatter.sh,        sv masked  exit=3 stdout=<0 chars> stderr=ERROR: …
    check-frontmatter.sh --json, sv masked  exit=3 stdout=<0 chars> stderr=Usage: …

**Beyond the gate:** it called `CHANGELOG.md:28` correct. It is not — it described
`cannot_compute` as writing a payload with no condition, and `cannot_compute`
emits one only when `emit_json` is true, which `check-frontmatter.sh` passes as
false. Same generalisation, one line above the bullet that got it right. Fixed in
the same commit.

### F5 — README "gains a `jq` row" — `6b8845b`

Row present at `2ed34b8:README.md:247`, rewritten at `HEAD:README.md:271`. Other
three sub-claims re-measured and true (skill-validator clause; removal of "There
is no such guard for `jq`", which wraps across lines; `DEP002` 0 → 3).

### F6 — the worked example — `bb956eb`

**The gate's diagnosis was itself half wrong.** It blessed the CHANGELOG shape as
reproducing "v0.4.1 500, start erased 1000". Measured: 500 both ways.

Structural reason: v0.4.1 does not read `startTimeUnixNano`, so "erasing the
start time raised it" cannot be a v0.4.1 result under any input.

    FIXTURE                  v0.4.1   head   candidate
    1_all_timed              1000     1000   1000
    1_all_timed_NOSTART      1000     1000   1000
    2_changelog               500     1000    500
    2_changelog_NOSTART       500     1000   1000
    3_two_point               500      900    500
    3_two_point_NOSTART       500      900    900
    4_1000_last              1000     1000   1000
    4_1000_last_NOSTART      1000     1000   1000
    5_1000_first              500     1000    500
    5_1000_first_NOSTART      500     1000   1000

v0.4.1 is identical across every `_NOSTART` pair. Column 3 is a reconstructed
build of the design the numbers *do* describe: runs keyed by start (as head) with
each run holding its latest point by `timeUnixNano` (as v0.4.1) — the obvious way
to add runs without deleting the ordering. Under it `2_changelog` reports 500 and
erasing the start raises it to 1000, exactly as narrated.

So the passage is not a v0.4.1 bug report; it is the argument for *deleting* the
time-ordering rather than repairing it, because the repair inverts the property
the merge is built on. All three files now attribute it to the candidate and say
it was measured against a build of one. `docs/profiler-spec.md` carried the same
misattribution in softer form and is corrected with them.

### F7 — "documented in both script headers and in the README" — `691e9c0`

Headers of both `--json` writers enumerate all three no-payload paths
(`check-structure.sh:17-21`, `check-paths.sh:14-18`); `tests/test_f01.sh:773`
onward pins them. README documents the guard-load path only. Two smaller
corrections in the same sentence: the reason clause was applied to all three
paths but is the reason for two, and "both script headers" became "each `--json`
writer" since `check-frontmatter.sh` has no payload channel.

### F8 — "and the Go test reasons" — `f926d27`

`grep -rn '0\.4\.1' profiler/*_test.go profiler/cmd/*_test.go` → no matches, and
no version-free equivalent. The three named files do carry such statements (2, 2,
1). The one Go test reason this release did change removes a stale model rather
than recording v0.4.1, and the next bullet already credits it.

### F9 — SUSPECTED → **proven** — `4507aed`

Both halves reproduce, under *different* guard shapes; the sentence asserted them
of one. "Exited 2 from every script" and "left `audit-report.sh` exiting 1" cannot
both hold, because "every script" includes `audit-report.sh`.

`verdict-guard.sh` is absent at `2ed34b8` and `bd32b26`, so the pre-fix state is
the intermediate one where the guard existed but its load was unchecked.
Reconstructed by stripping the `bash -n` and `declare -F` checks:

    INVOCATION                 guard-syntax  guard-partial  guard-absent
    check-structure.sh         exit=2        exit=0 +out    exit=1
    check-structure.sh --json  exit=2        exit=127       exit=1
    check-paths.sh             exit=2        exit=0 +out    exit=1
    check-paths.sh --json      exit=2        exit=127       exit=1
    check-frontmatter.sh       exit=2        exit=127       exit=1
    audit-report.sh            exit=2        exit=127       exit=1

Unparseable → 2 everywhere; absent → 1 everywhere. No shape gives 2 from the
checkers and 1 from `audit-report.sh`. `audit-report.sh` documents only 0 and 3,
so both are undocumented — the bullet said that of 1 only. Corrected rather than
removed: **under-measured, not invented.** The "now" half verified: all three
shapes give exit 3, empty stdout, from all six invocations at head.

### F10–F12 — branch-only states — `2488ff0`

Each verified, not assumed:

- **audit-report bullet** — needs `check-structure.sh` to emit `DEP002` at exit 3.
  `DEP002` occurs 0× at `2ed34b8`, 6× at HEAD. Checker fixed, composer not: branch.
- **guard-load bullet** — `verdict-guard.sh` absent at `2ed34b8` and `bd32b26`.
- **`scripts/lib/` flattening** — 0 entries at `2ed34b8`, `bd32b26` and HEAD;
  exists only on dev commit `60c7f44`, which `git merge-base --is-ancestor`
  reports is **not** an ancestor of HEAD. No release carried the nested layout.

Phrasing now matches the encoder bullet, which already said "at first".

## Re-measurement sweep — beyond the six

Denominator: the gate's 160-assertion ledger. Everything below re-measured at head.

**Counts, all exact as claimed:**

| Claim | Measured |
|---|---|
| 66 test functions (61 profiler + 5 CLI) | 66 (61 + 5) |
| 230 subtests, 200 direct, 30 nested | 230, 200, 30 |
| v0.4.1: 58 functions, 179 subtests | 58, 179 |
| shell 850 = 561/243/27/19 | 850 = 561/243/27/19 |
| v0.4.1 shell 134 | 134 |
| fixtures 39 → 55 | 39 → 55 |
| both skills: 0 errors, 0 warnings | 0, 0 |

**Every narrated number reproduced** (v0.4.1 vs head):

    two resources cumulative 100+200   200  -> 300
    same pair under delta              300  -> 300
    100 then restart at 20              20  -> 120
    cumulative 50@t9000 + delta 100@t1000   50 -> unknown (refused)
    isMonotonic:false decreasing       200  -> 1000
    800000/1200000/unplaced 1200050  1200050 -> 1200050 (not 2400050)
    runs 100,20 + unplaced 500         500  -> 520
    runs 100,100,100 + unplaced 150    150  -> 350

**Protobuf facts, all confirmed:** ulp(1.7893e18) = 256 ns; the literal
`9223372036854775807` parses to exactly 2⁶³; `EmitUnpopulated` on writes `"0"`,
off omits; explicit zero and omitted field unmarshal alike;
`fixed64,2,name=start_time_unix_nano` confirmed in OTLP proto v1.11.0.

**Schema keys:** every `json:` tag in `profiler/types.go` byte-identical to
`2ed34b8`. No schema key changed.

**Exit sweep:** 168 invocations (7 scripts/modes × 4 targets × 3 tool masks × 2
locales) — histogram `{0: 68, 1: 12, 3: 88}`, **0 outside `{0,1,2,3}`**.

**Encoder:** all 255 byte values through `json_string` under `C` and
`en_US.UTF-8` — 0 escapes wider than 4 hex digits, and the two locales produce a
**byte-identical digest** (`e8ae756e…`), which proves the "same bytes under every
locale" claim directly.

**Corrections quote verbatim:** all five quoted strings occur exactly twice — once
in the 0.4.1 entry, once in the 0.4.2 correction.

**Adjacent item (gate's, out of diff) — measured TRUE, left alone.**
`profiler/testdata/otlp/README.md:57`, "a series key that ignores the scope
collapses them into one run holding 200, instead of … = 400". All four points
share one `startTimeUnixNano`, so under the *current* run rule one series holds
the greatest reported = **200**. Correct as a counterfactual about the current
rule. head reports 400, v0.4.1 reports 60 — the gate's 60 is the v0.4.1 number,
which the sentence does not claim. No change: the evidence says the claim is true,
and the file is not otherwise touched by this PR.

## Final verification at `2488ff0`

    go build ./...        OK
    go vet ./...          OK
    go test -race ./...   ok profiler 1.376s ; ok profiler/cmd 2.371s  (testcache cleared)
                          66 functions, 230 subtests

    LC_ALL=C          test_f01 561 / test_f02 243 / test_skill 27 / test_walk 19 = 850, 0 failed
    LC_ALL=en_US.UTF-8 test_f01 561 / test_f02 243 / test_skill 27 / test_walk 19 = 850, 0 failed

    tests/test_skill.sh from repo root: 27 passed, 0 failed, exit 0

**`tests/test_skill.sh`'s green is not treated as proof** — 16 of its assertions
compare a literal against itself and cannot fail. That defect is a 0.4.3 item and
was not touched here.

## Not done, by instruction

- No merge, no tag.
- No version surface touched (all 16 correct).
- `add`'s behaviour unchanged — whether the delta branch should clamp or refuse a
  negative is a behaviour change for 0.4.3.
- `tests/test_skill.sh`'s vacuous assertions left for 0.4.3.
