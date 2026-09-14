# Deferred ledger — what 0.4.1 and 0.4.2 chose not to fix, audited at `origin/main` `5c847e1`

_Hunter: deferred-ledger. 2026-09-14. Worktree cut from `5c847e1`; primary tree untouched._

**Coverage.** 33 numbered ledger items enumerated from every deferral source the mandate
names — `release-0.4.1/{mandate,status,worklist-round1,worklist-round1-residuals,worklist-round2}.md`,
`release-0.4.2/{mandate,status,release-commit-requirements,release-commit-draft,apply-record}.md`,
`release-0.4.2-scripts/{mandate,status,worklist-round2}.md`, `.out-of-scope.md`,
and the deferral text in `CHANGELOG.md` / `RELEASE_NOTES.md` — all 33 walked against the tree
at `5c847e1` by execution. 26 reproduced or confirmed by running code, 7 disposed by reading.
Four shell suites run from the repo root: 850 passed / 0 failed. Two mutation experiments run
in scratch copies (`git archive 5c847e1`), never in a shared tree.

**Headline.** 18 of the 33 are still open. 9 were closed — 6 incidentally by 0.4.1's own later
rounds and 3 by the 0.4.2 work — and are proved closed below. 4 are in flight on PR #5
(`13b3141`) and are not 0.4.3 work provided that PR merges. 2 are out of product scope.

**Two deferrals whose stated reason is no longer valid**: L14 (composite attribute identity —
recorded as "a bound on the fix, not a known bug", but it produces a *wrong, over-counted*
number and contradicts the OTel spec) and L17 (`cache_creation` → `cache_write` — deferred as
breaking, and on the evidence at head the rename is no longer desirable at all).

---

## The ledger

Legend: **OPEN** still present · **CLOSED** fixed since it was deferred, proved below ·
**IN-FLIGHT** carried by PR #5 · **OUT** outside the delivered product.

| # | Item | Source | State | R/S | P | Home |
|---|---|---|---|---|---|---|
| L1 | `tests/test_skill.sh` has no `cd` to the repo root | scripts st. R3/R4 | OPEN | REAL | P2 | 0.4.3 |
| L2 | `test_skill.sh`'s 11 `[[ ]]` guards cannot report FAIL (bash 3.2 errexit skew) | bash skew | OPEN | REAL | P1 | 0.4.3 |
| L3 | `check-quality.sh` forwards `skillscore`'s exit status | scripts st. R3/R4 | OPEN | REAL | P2 | 0.4.3 |
| L4 | `test_f01.sh`'s rule-ID census is syntax-shaped | scripts st. R4 | OPEN | REAL | P3 | 0.4.3 (test-only) |
| L5 | Per-finding `jq` re-invocation in both `--json` writers | scripts st. R3/R4 | OPEN | REAL | P3 | 0.5.0 |
| L6 | `isMonotonic` is unread | 0.4.2 req. | OPEN | REAL | P3 | 0.5.0 (doc IN-FLIGHT) |
| L7 | `flags` is unread (`NO_RECORDED_VALUE`) | 0.4.2 req. | OPEN | REAL | P2 | 0.5.0 (doc IN-FLIGHT) |
| L8 | A `--json` exit 3 does not always carry a payload | 0.4.2 req. | OPEN, deliberate | REAL | P3 | 0.5.0 — no action |
| L9 | Schema v1 has no channel for a refused series | 0.4.2 req. | OPEN | REAL | P2 | 0.5.0 |
| L10 | Two stale `v0.4.0` refs in README + one in `.out-of-scope.md` | draft OPEN-6 | OPEN | REAL | P3 | 0.4.3 |
| L11 | Skill `metadata.version` not bumped (`skill-audit` 0.2.0) | draft OPEN-4 / apply-record | OPEN | REAL | P3 | 0.4.3 |
| L12 | `skill-rewrite` `metadata.version` 0.1.0 "despite substantial change" | draft OPEN-4 | **INVALID at 5c847e1** | REAL | — | no action |
| L13 | Whole-export slurp (~4.5× file size in RSS) | 0.4.1 X12 | OPEN | REAL | P3 | 0.5.0 |
| L14 | Composite attribute values compacted, not canonicalised | 0.4.1 R2 fnf | OPEN, **reason stale** | REAL | P2 | 0.4.3 or early 0.5.0 |
| L15 | A truncated final line costs the whole capture | item-19 fnf | OPEN | REAL | P2 | 0.5.0 |
| L16 | `ToolCallEntry.Success`: rejected vs failed indistinguishable | W4 / 0.4.2 mandate | OPEN (moved, not removed) | REAL | P2 | 0.5.0 |
| L17 | `cache_creation` → `cache_write` rename | 0.4.1 mandate | OPEN, **reason stale** | REAL | P3 | drop or 1.0 |
| L18 | `profiler -h` / `probe -h` / `capture -h` exit 1 | 0.4.1 R3 #7 | OPEN | REAL | P3 | 0.4.3 |
| L19 | not-a-count reason's upper bound is a hair generous | 0.4.1 R3 fnf | OPEN | REAL | P3 | 0.5.0 |
| L20 | `npm install -g skillscore` unpinned in CI | W18 / R2 fnf | OPEN | REAL | P2 | 0.4.3 |
| L21 | `profiler/cmd` coverage thin (12.6%) | 0.4.1 R2 fnf | OPEN | REAL | P3 | 0.5.0 |
| L22 | Devin / Codex / Cursor install rows unverified | 0.4.1 R1 fnf #3 | OPEN, disclosed | REAL | P3 | 0.5.0 |
| L23 | Cumulative temporality never seen on real output | item-19 fnf | OPEN, honest | REAL | P3 | 0.5.0 |
| L24 | `tool_result`/`tool_decision` attrs `[DOCS]` not `[OBSERVED]` | item-19 fnf | OPEN, honest | REAL | P3 | 0.5.0 |
| L25 | `CaptureOpts.APIKey` never read | 0.4.1 fnf #5 | OPEN, documented | REAL | P3 | 0.5.0 — no action |
| L26 | `draft-rewrite.sh:49` comment is imprecise | 0.4.1 R1 fnf #5 | OPEN | REAL | P3 | 0.4.3 |
| L27 | `CHANGELOG.md:15`, `:47`, `RELEASE_NOTES.md:20` false claims | 0.4.2 req. | IN-FLIGHT (PR #5) | REAL | P2 | PR #5 |
| L28 | Two stale code comments (`profiler_test.go:1236`, `otlp.go` `counterAccumulator`) | 0.4.2 req. | IN-FLIGHT (PR #5) | REAL | P3 | PR #5 |
| L29 | `AdapterVersion` stamps 0.4.1 over changed totals | 0.4.2 req. | IN-FLIGHT (PR #5) | REAL | P1 | PR #5 |
| L30 | PR #5's "Known limits, carried to 0.5.0" block records 4 of ~18 | draft §6b (unanswered) | OPEN | REAL | P2 | PR #5 or 0.4.3 |
| L31 | The two 0.4.1 series-key limits (multi-resource, counter reset) | 0.4.1 R6 fnf | **CLOSED** by 0.4.2 | REAL | — | proved below |
| L32 | Six 0.4.1-era deferrals closed by later 0.4.1 rounds | various | **CLOSED** | REAL | — | proved below |
| L33 | `scuba-guard` hook misclassifies `-wt/` worktrees (3 agents) | 0.4.2 st. | OUT (tooling) | REAL | P2 | not a release item |

---

## Detail — the 0.4.3 patch set

### L2 (P1, REAL, reproduced) — `test_skill.sh`'s structural guards cannot report a failure

`tests/test_skill.sh` has no `assert` that evaluates anything. 16 of its 25 `assert` calls are
`assert "<label>" true` — a **literal**, so the function prints PASS unconditionally. The real
check is a bare `[[ ]]` on the preceding line, relying on `set -euo pipefail` (`:2`) to abort.
Under the local bash (3.2.57, the only bash on this machine) **`set -e` does not fire on a
failing `[[ ]]` at all**:

    $ bash -c 'set -euo pipefail; [[ -f .nope ]]; echo REACHED'
    REACHED          exit 0

Eleven guards depend on it: `tests/test_skill.sh:21, 38, 63, 65, 78, 83, 85, 87, 89, 118, 123`.
Mutation-tested in a scratch copy of `5c847e1` — `skills/skill-audit/references/best-practices.md`
removed, which `:87`/`:88` exist to catch:

    PASS: skill-audit best-practices reference exists      <- the mutant survives its own assertion

The run later aborts silently (21 PASS lines, no FAIL line, no summary, exit 1) at a different
check, so the suite reports *nothing about the file that is missing*.

Invariant that must hold: every assertion in the suite must be able to print FAIL and be counted,
on every bash the project supports. Today the condition and the reported verdict are two
independent things.

**On the bash-5 half (SUSPECTED, not reproduced — no bash 5 or container runtime on this host).**
CI is `ubuntu-latest`, where GitHub Actions runs `run:` steps under bash 5.x, and POSIX requires
errexit to fire on a failing `[[ ]]` outside an exempt context. If so CI *does* catch these, but
as a bare abort with no FAIL line and no summary. Either way the product-level statement holds
and does not depend on the bash version: **11 of 27 assertions cannot report FAIL.**

Scoped precisely: `test_walk.sh`'s 7 literal-`true` asserts are each preceded by a *real command*
(a script, a python heredoc), which does trip errexit on 3.2, so they are sound. `test_f01.sh`
and `test_f02.sh` compute every condition (`0` literal-true asserts). The defect is confined to
`test_skill.sh`. A sweep for bash-4+ constructs (`declare -A`, `mapfile`, `${v^^}`, `local -n`,
`|&`) across `tests/*.sh`, `tests/lib/*.sh` and `skills/*/scripts/*.sh` is clean — this is the
only version-skew site in the tree.

### L1 (P2, REAL, reproduced) — `test_skill.sh` has no `cd "$(dirname "$0")/.."`

`tests/test_walk.sh:4`, `tests/test_f01.sh:4` and `tests/test_f02.sh:7` all carry it;
`tests/test_skill.sh` does not. Run by absolute path from `/tmp`:

    PASS: .devin-plugin/plugin.json exists            <- L2's vacuous assert, again
    FileNotFoundError: '.devin-plugin/plugin.json'    exit 1

It does not bite CI (`.github/workflows/ci.yml:25` runs it from the checkout root) and it did
not bite the release apply (`apply-record.md` notes the suites were run from the root "because
`tests/test_skill.sh` has no `cd` of its own"). That work-around has now been carried in three
separate status files, which is the signal the deferral has outlived itself.

**Shared root with L2.** `test_skill.sh` is the one suite never brought up to the other three's
conventions: no `cd`, and an `assert` that takes a literal rather than a condition. One repair
to that file closes both, and closing L2 alone while leaving the literal-`true` pattern would
re-open as the next round's finding.

### L3 (P2, REAL, reproduced) — `check-quality.sh` forwards `skillscore`'s exit status

`skills/skill-audit/scripts/check-quality.sh:5` documents `Exit codes: 0=success, 3=execution
error`. `:20` is a bare `skillscore "$skill_dir" --json` at the end of a `set -e` script, so
every caller error reaches the caller as `skillscore`'s own status:

    check-quality.sh /no/such/dir   -> exit 1     (documented set is {0,3})
    check-quality.sh README.md      -> exit 1
    check-quality.sh skills/skill-audit -> exit 0 (control)

It is a class, not a count: a bad flag, a non-directory target and a nonexistent path all
forward. It is the one script in the skill PR #4 never touched, which is why it is the last
member of the "no script reports a status it did not compute" class still open. The invariant
0.4.2 established for the other four — a script's exit status is drawn from its own documented
set, and an unenumerated child status is an execution error — does not hold here.

### L20 (P2, REAL) — `npm install -g skillscore` is unpinned in CI

`.github/workflows/ci.yml:23`. `skill-validator` is pinned at `v1.6.1` on `:19`; `skillscore`
takes whatever npm serves that day, and `tests/test_f02.sh` (243 assertions) composes reports
over its output. A skillscore release changing its JSON shape turns a green suite red with no
repo change — and, worse, could turn a red one green. The 0.4.1 deferral ("record it") has been
carried through W18, round 2 and the 0.4.2 out-of-scope list without being taken. It is a
one-line change with no schema and no behaviour impact: patch material.

### L14 (P2, REAL, reproduced — the deferral's reason is no longer valid)

Recorded at 0.4.1 round 2 as "Composite attribute values are compacted, not canonicalised …
No producer varies its own serialisation within a file, so this is a bound on the fix, not a
known bug." That framing was taken from `arrayValue` whitespace. It under-scopes the class.

`profiler/otlp.go:715` gives a `kvlistValue` its identity as `"k" + compactJSON(...)`, and the
comment at `:695` states the rule: "the order of a composite's members is part of its value."
That is right for `arrayValue` and **wrong for `kvlistValue`**. The OTel common specification
says of map values: *"Maps are equal when they contain the same key-value pairs, irrespective of
the order in which those elements appear"* (https://opentelemetry.io/docs/specs/otel/common/).

Reproduced through the built CLI on two cumulative data points carrying the same two-entry
`kvlistValue` attribute, members in opposite order, each reporting 100:

    members in the same order (control)  -> {"input":100}     correct
    members reversed                     -> {"input":200}     wrong

This is not an undercount like the two limits 0.4.2 fixed — it **over-counts**, which is the one
direction the profiler's own claim ("unavailable metrics are `unknown` with a reason, never
invented") forbids. Cumulative only; delta is unaffected because both points add either way.
No documented capture route produces a kvlist token attribute today, which is why it is P2 and
not P1 — but the recorded reason for deferring it ("not a known bug") is now false.

Invariant: two data points whose attribute sets are equal under the OTel data model are one
series. Members of a map value are unordered; members of an array value are not.

### L10 (P3, REAL) — stale `v0.4.0` references

`README.md:28` ("v0.4.0 adds a harness-agnostic profiler…"), `README.md:423` ("the profiler
(v0.4.0 preview)") and `.out-of-scope.md:8` ("The profiler (v0.4.0 preview)"). PR #5 changes
`README.md:55` and `:327` only and does not touch these three. Note the mandate's framing of
"two stale *preview* references in the readme" is slightly off: only `:423` says "preview";
`:28` is a bare version claim, and the third site is in `.out-of-scope.md`, not the README.

The *substance* of both `.out-of-scope.md:8` and `README.md:423` is still true at `5c847e1` —
`profiler/compare.go` and `profiler/experiment.go` do not exist there, so the comparison engine
genuinely is not built. Only the version label is stale.

### L11 (P3, REAL) — `skill-audit` `metadata.version` has not moved

`skills/skill-audit/SKILL.md:7` reads `version: "0.2.0"`, set by `7607210`. Since then, on
`origin/main`:

    git diff --stat 7607210 5c847e1 -- skills/skill-audit/
      7 files changed, 543 insertions(+), 168 deletions(-)
      (incl. a new scripts/verdict-guard.sh, +216)

with two new consumer-observable rule IDs (`DEP001`, `DEP002`) and a third registered (`PATH`).
The repo's own convention makes this a bump: `CHANGELOG.md:79` files "Bumped `skill-audit`
metadata version to 0.2.0" as a release bullet. `apply-record.md` records it as knowingly left
unresolved, so it is a live open question, not an oversight. No user-visible effect today —
nothing in `tests/` or the scripts reads `metadata.version` — hence P3.

### L12 — INVALID at `5c847e1`, and this matters

The brief names `skill-rewrite` 0.1.0 "not moving despite substantial change". On the **delivered**
tree that is false:

    git diff --stat 0a03af1 5c847e1 -- skills/skill-rewrite/
      1 file changed, 1 insertion(+), 1 deletion(-)

The substantial `skill-rewrite` change is `b73326d`, which `git merge-base --is-ancestor
b73326d 5c847e1` shows is **not** an ancestor of `origin/main` — it is unlanded local work and
explicitly out of this audit's scope. `skill-rewrite`'s version is correct for what is delivered.
L11 stands alone; do not bundle the two.

### L18 (P3, REAL, reproduced) and L26 (P3, REAL)

    profiler -h        -> exit 1        profiler probe -h    -> exit 1
    profiler --help    -> exit 1        profiler capture -h  -> exit 1
    profiler version   -> exit 0 (control)

An explicitly requested help conventionally exits 0. Two mechanisms, as round 5 recorded:
`main`'s switch has no `-h` case and falls to `default: usage(); os.Exit(1)`, while the
subcommands go through `flag`'s `ErrHelp` in `parseFlags` (`profiler/cmd/main.go:42`). The
deferral's reason ("it does not touch the 2-means-nothing-was-read contract") is still sound —
1 already means usage error, so moving help to 0 is safe and does not disturb 2.

L26: `skills/skill-rewrite/scripts/draft-rewrite.sh:49` still says "Resolve skill-audit scripts
relative to this script" where `:50-51` resolve two levels above it. Comment-only.

### L4 (P3, REAL, mutation-proved) — the rule-ID census is syntax-shaped

`tests/test_f01.sh:825-832`'s `rules_emitted_by` recognises four literal forms. Verified against
the tree, and then mutation-tested in a scratch copy of `5c847e1`: adding
`cannot_compute "XX003" "mutant" true` to `check-paths.sh` — the rule ID *quoted* — leaves

    PASS: SKILL.md registers every rule ID the scripts emit, and registers no ID none of them emits

green, with `XX003` reachable by a consumer and registered nowhere. The census also attributes
`DEP001` only to `verdict-guard.sh:136`, never to `check-paths.sh:57`, `check-structure.sh:60`,
`check-frontmatter.sh:41` or `audit-report.sh:60`, which emit it indirectly through
`require_tool` — so the per-script header assertion at `:851-855` passes vacuously for all four.
And `check-structure.sh:158`'s `findings+=("${level}|${rule}|${message}")` relay is invisible to
it entirely. Set equality holds today only because every call site happens to use a recognised
form. Test-only, zero product risk; cheap enough to fold into 0.4.3.

---

## Detail — honest 0.5.0 deferrals (reason re-checked and still valid)

### L9 (P2, REAL, reproduced) — schema v1 has no channel for a refused series

    testdata/otlp/mixed_temporality.ndjson       -> {"state":"present","value":{"input":640}}
    testdata/otlp/mixed_temporality_only.ndjson  -> {"state":"unknown","reason":"… 1 time series
                                                     carried both delta (1) and cumulative (2) …"}

The first is the gap: a series was refused, its tokens are absent from the total, the result is
`present`, and **nothing in the profile says so**. A `present` result carries no reason in v1.
The deferral reason ("adding one inside v1 is a schema change wearing a bug fix's clothes") is
defensible and I am not overriding it — but it is worth re-examining once: an additive
`omitempty` caveat field breaks no v1 consumer, and this is the highest-harm item on the whole
ledger, because it is the one place the product silently reports a number it knows is short.
The same channel is what L7 and L16 need, so one design closes three.

### L7 (P2, REAL) — `flags` is unread

`grep -rn '"flags"' profiler/*.go` returns nothing at `5c847e1`; the field is not on
`otlpDataPoint` at all. A `NO_RECORDED_VALUE` point (bit 1) carrying 0 still reads as a running
total. Reason for deferring (a new leaf, a new refusal class, its own reason clause, fixtures
and a spec paragraph — the size of a feature, and no documented capture route emits one) is
still correct. Its documentation lands on PR #5.

### L6 (P3, REAL) — `isMonotonic` is unread

Same grep, no occurrence anywhere in `profiler/`. The judgement ("greatest-wins cannot
over-report a monotonic cumulative sum; every fixture declares `true`; `claude_code.token.usage`
is a Counter") still holds. Documentation lands on PR #5.

### L15 (P2, REAL, reproduced) — a truncated final line costs the whole capture

Two complete batches worth 300 input tokens, plus a truncated third line:

    two full batches (control) -> present {"input":300}   exit 0
    + truncated tail           -> error "malformed JSON at byte 979 in batch 3: unexpected end
                                  of JSON input. No data from earlier batches was used; if the
                                  collector was stopped mid-write, delete the final partial
                                  line and retry."       exit 2

Realistic for README route (b), but the reason string names the exact remedy, so there is a
first-class workaround. Deferral ("if it bites, the repair is an explicit `--allow-truncated`,
not a softer classifier") is still the right call — a softer classifier would re-introduce the
round-6 defect where a partial read is reported as a complete measurement.

### L16 (P2, REAL, reproduced) — the `Success` conflation moved rather than closed

0.4.1 item 19 changed what `ToolCallEntry.Success` means (spec `docs/profiler-spec.md:244`:
"the tool ran and succeeded"), which closed W4 as originally written. The residual is that
rejected calls are listed with `success: false` too, so the two are indistinguishable:

    tool_failure.json (ran and failed) -> [{"name":"Write","timestamp":"…","success":false}]
    full_export.ndjson (rejected)      -> [{"name":"Bash", "timestamp":"…","success":false}, …]

`profiler/types.go:126-128` has no third field. Documented at spec `:244`, so it is a stated
limit rather than a false claim. Needs a schema field — same channel as L9. Not a patch.

### L13 (P3, REAL, measured) — whole-export slurp

`profiler/otlp.go:246` (`export = append(export, b)`) accumulates every batch.
Measured at head, 9,500,000-byte NDJSON:

    maximum resident set size  43,171,840    (~4.5x the file)

Consistent with the recorded ~6x. No documented capture route produces a file where this bites.
Streaming changes `otlpExport` and every walk over it — larger than any bug it prevents. Stands.

### L5 (P3, REAL, measured) — per-finding `jq`

`check-paths.sh:118` and `check-structure.sh:155-157,:177` spawn a `jq` per finding, over a
`json_findings` accumulator that is re-piped each iteration. Measured on synthetic skills whose
only faults are broken script references:

    200 findings -> 1.94s      400 -> 5.09s      800 -> 9.92s

Extrapolates to the recorded ~95s at 4,000. A real skill raises single-digit findings, so this
is ~0.1s in practice. Performance, not correctness; wants its own change. Stands.

### L8 (P3, REAL, deliberate) — payload-less exit 3

    check-structure.sh --json                 -> exit 3, stdout empty
    check-structure.sh --json /no/such/skill  -> exit 3, stdout empty

Stated in the script header at `check-structure.sh:17-21` ("Three exits carry no payload …"),
and pinned by tests. This is a documented boundary, not an open gap. **No 0.4.3 action.**

### L19, L21, L22, L23, L24, L25 (all P3, all confirmed OPEN and honestly held)

- **L19** — `profiler/claude_code.go:380`. `asDouble: 9223372036854775807` (which float64 rounds
  to 2^63) is refused with `"a count is a whole number from 0 to 9223372036854775807"`, naming a
  bound the `asDouble` path cannot in fact reach. The spec pins the string verbatim, so a reword
  moves the string, its fixtures and its tests together. Stands.
- **L21** — `go test -cover ./...` at head: `profiler` 96.0%, `profiler/cmd` **12.6%**.
- **L20 is separate from L22**: `README.md:329-337` marks the Devin, Codex and Cursor install
  rows "not verified in this release" and says why. That is an honest disclosure, not a false
  claim; no CLI for any of the three is available here either. Stands.
- **L23/L24** — the fixture README's provenance markers (`testdata/otlp/README.md:18` `[OBSERVED]`,
  `:26` `[DOCS]`) are intact, and no shipped document claims cumulative temporality or the
  tool-event attributes were observed live. Honest; stands.
- **L25** — `profiler/types.go:64,72`: `APIKey` documented as reserved for the Devin and Cursor
  adapters. Deliberate; no action.

### L17 (P3, REAL — deferral correct, but the item itself should probably be dropped)

`cache_creation` is still the profile key (`profiler/types.go:110`, spec `:164`); `cache_write`
appears nowhere in the repo. The 0.4.1 mandate forbade the rename as a breaking schema change,
and that reason still holds — it is a `profile/v1` output key, so it is definitionally not patch
material. But the case *for* the rename has weakened: `profiler/claude_code.go:32` reads the
OTLP attribute value `cacheCreation`, so `cache_creation` is a faithful snake_case of the wire
name. Recommendation: take it off the 0.5.0 list as a deliberate decision rather than carry it
as debt, or pair it with the only schema bump the project actually needs (L9/L16).

---

## Closed since they were deferred — proved, not assumed

### L31 — the two 0.4.1 series-key limits, closed by 0.4.2

Both were shipped as known limits in `docs/profiler-spec.md`, `README.md`, `CHANGELOG.md`,
`RELEASE_NOTES.md` and PR #2's body. Re-run through a binary built from `5c847e1`:

    multi_resource_cumulative.json  -> {"input":300}    (v0.4.1 gave 200)
    counter_reset.ndjson            -> {"input":120}    (v0.4.1 gave 20)

`grep -n "Two known limits\|known limits of the token series key" README.md
docs/profiler-spec.md` is now empty — which is exactly why L27's changelog claims are false.

### L32 — six 0.4.1-era deferrals closed by 0.4.1's own later rounds or by 0.4.2

| deferral | closed by | proof at `5c847e1` |
|---|---|---|
| `check-structure.sh --json` silent pass with `jq` absent | 0.4.2 PR #4 | `require_tool jq true` at `check-structure.sh:60`; 850 suite assertions |
| `check-frontmatter.sh` prints `frontmatter OK` with validator absent | 0.4.2 PR #4 | guard + enumerated `case` |
| `capture --export-file` inert for the only adapter | 0.4.1 W7 | `--export-file is not read by the claude_code adapter; supply an OTel export with --otel-file`, exit 1 |
| `Capture` error-path asymmetry (tokens `error`, others `unknown`) | 0.4.1 W6 | malformed export -> `{"t":"error","tc":"error","ti":"error"}` |
| CHANGELOG 0.4.0 "pinned to a snapshot hash" | 0.4.1 R1 W12 | `grep -n "snapshot hash" CHANGELOG.md RELEASE_NOTES.md` empty |
| `go vet` / `-race` / `gofmt` absent from CI | 0.4.1 R2 T7 | `.github/workflows/ci.yml:41-53` runs all three |

Two more, superseded rather than fixed, and worth naming so they are not re-raised:

- *"A cumulative run whose value decreases under an unchanged start keeps the later, smaller
  value"* (0.4.2 round 1) — superseded by round 3's greatest-wins. Reproduced at head: one run,
  900 at t1001 then 500 at t1002 -> `{"input":900}`.
- *"Within a run, a timed point beats an untimed one unconditionally"* — closed by deletion;
  `timeUnixNano` is not read by the merge at all (spec `:218`).
- The *"README `jq` row is now long … if 0.5.0 adds the guard, the row collapses"* note — the
  guard landed in 0.4.2 and the row did collapse: `README.md:271` now describes the guard, not
  the three-branch silent drop.
- The exponent-form refusal is no longer an open conformance gap: it is recorded as a stated
  deviation at `docs/profiler-spec.md:214` and `:223`.
- The 0.3.1 `exit 1` dependency guards (`skills/skill-audit/SKILL.md:45-46`) were not softened.

---

## In flight on PR #5 — verify, do not re-derive

PR #5 is `release/0.4.2-version` at `13b3141`, base `5c847e1`, per
`.scuba/teams/release-0.4.2/apply-record.md`. Confirmed on that branch by `git show`:

- **L29** — `profiler/types.go` now `const AdapterVersion = "0.4.2"`. At `5c847e1` a profile from
  `multi_resource_cumulative.json` stamps `"adapter_version": "0.4.1"` while reporting `300`
  where real 0.4.1 reported `200`. That violates the invariant the release requirements state —
  *no two profiles carrying the same `adapter_version` may report different totals for the same
  export* — and it is violated **on delivered `main` right now**, not hypothetically. It is P1
  and it is PR #5's to close; if that PR stalls, it becomes 0.4.3's most urgent item.
- **L27** — `CHANGELOG.md:15` still carries "as the OTLP spec requires"; `CHANGELOG.md:47` still
  says the spec and README "state two known limits" (they no longer do — see L31);
  `RELEASE_NOTES.md:20` still says both "both 0.4.2" and "fixed together in 0.5.0" in one bullet.
  All three superseded, not rewritten, in PR #5's 0.4.2 entry.
- **L28** — `profiler/profiler_test.go:1236` ("the later point supersedes the earlier") and
  `profiler/otlp.go`'s `counterAccumulator` doc comment ("would use it unchanged") both still
  present at `5c847e1`; both corrected on PR #5.
- **L6 / L7 / L8 / L9** — all four recorded under a new `**Known limits, carried to 0.5.0**`
  heading at `CHANGELOG.md:52` on that branch.

### L30 (P2, REAL) — that known-limits block is an under-count

PR #5's block names four items. This audit finds **18 open** at `5c847e1`. The gap is not an
accident: `release-commit-draft.md` §6b asks the manager in writing whether the scripts team's
four found-and-not-fixed items belong in the changelog's known-limits block, and the question
was never answered, so the implementer correctly declined to extend the list on its own
judgement. The result is that 0.4.2 ships a "Known limits, carried to 0.5.0" heading under which
L5, L13, L14, L15, L16, L19, L21 and the rest do not appear.

Invariant: a release's known-limits section should be the *whole* set of limits its authors know
about, or it should say which subset it is. Two directions, both advisory: extend the block from
this ledger, or retitle it to what it actually is (for example, the limits of the token merge
and the verdict guard) so a reader is not told the list is complete.

---

## Out of the delivered product

**L33** — the `scuba-guard` hook anchors containment on `<project>/.claude/worktrees/agent-*`,
so it classified three separate workers in `skill-architect-wt/…` as the top-level lead and
denied `Write`/`Edit` on every tracked file. Three agents, three different work-arounds
(scratch-mirror + `cp`, `python3` substitution scripts). Recorded in
`release-0.4.2/status.md` twice. Real, and worth fixing, but it is tooling — it does not belong
in a 0.4.3 release and no user of the plugin is affected.

Also out of scope by the mandate and untouched here: local `main` `0a83615` and its 7 unpushed
commits, the 21 modified tracked files in the primary tree, `skillgate/` and `skills/skill-gate/`,
and PR #5's version surfaces themselves.

---

## Verification run for this audit

- Own worktree at `5c847e1`, detached; `git status --porcelain` empty throughout. All mutation
  experiments in scratch copies made with `git archive 5c847e1`. No `checkout`, `reset`, `clean`
  or `stash` anywhere.
- `tests/test_skill.sh` 27 · `tests/test_walk.sh` 19 · `tests/test_f01.sh` 561 ·
  `tests/test_f02.sh` 243 — **850 passed, 0 failed**, run from the worktree root.
- `cd profiler && go test -cover -count=1 ./...` green: `profiler` 96.0%, `cmd` 12.6%.
- CLI built into the session scratchpad only; never installed.
- Local bash is 3.2.57 and it is the only bash on this host; no container runtime is reachable
  (`docker` binary present, daemon down). Every bash-5 statement above is labelled SUSPECTED.
