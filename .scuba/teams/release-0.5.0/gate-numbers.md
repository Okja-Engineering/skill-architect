# Gate lens: EVERY NUMBER — PR #22 @ `9c8ba53`

> **Persisted by the chief of staff.** The hunter had no Write tool, said so, verified by
> `ls`, and did not fall back to a heredoc. This is that report.

**VERDICT: NOT CLEAN — 10 REAL, 2 SUSPECTED.** The headline figures are all correct; the
defects are in **totals that don't equal their parts** and in **counts guarded by floors
that cannot fire**.

## Coverage

88/88 added-or-modified lines carrying a numeral across the three prose files (CHANGELOG
62, RELEASE_NOTES 16, README 10); **61/61 distinct numeric tokens**, each re-derived with
its own command; 7/7 suites × 2 bashes in full; 3/3 Go packages counted statically **and**
at runtime; **15/15** deferred-ledger 0.5.0 rows re-walked individually; 4/4 baselines
re-measured from the `v0.4.3` tag; **9/9 numeric floor asserts in the repo enumerated**
and the two largest driven to a mutation. Every figure got a *second* derivation where one
existed. 7 worktrees, all left clean; primary tree untouched at `0a83615`.

## The big figures — all correct

Re-derived independently, two ways each. S10's commands were not read.

| Claim | Derived | Second derivation |
|---|---|---|
| 2649 / 7 suites / 0 failed | **2649** | suite summaries **and** `grep -c '^PASS: '`; per-suite 1868/350/174/136/53/48/20 exact |
| identical on both bashes | **identical** | both shells with a PATH shim so nested `bash` resolved to the same build |
| 275 = 233+34+8 | **275 = 233+34+8** | `grep -c '^func Test'` **and** `=== RUN` with no `/` |
| 669 = 636+33 | **669 = 636+33** | slash-depth 1 and 2; depth-3+ = 0 |
| 1 by-design skip | **1**, `TestHomeBarrierChild` | only `--- SKIP` in the log |
| 97.9 / 94.5 / 93.5 % | **exact** | `go test -cover` **and** `go tool cover -func` |
| 64 fixtures (vs 59) | **64 / 59** | file count at head and at `v0.4.3` |
| 8 DuckDB queries | **8** | *single derivation* |
| 6 new subcommands | **6** | diffed `profiler -h` at `v0.4.3` vs head |
| 15 ledger rows, 13 open | **15 / 13** | `awk` over Home+State; the 13-item list maps 1:1 to L5,6,7,8,9,13,15,16,19,22,23,24,25 |
| baselines 2499 / 93 / 452 / 59 | **all four exact** | measured from the tag, not quoted |

**Ledger re-walk, all 15 individually:** the 13 open are genuinely open at head — L5
(`check-structure.sh:247-251`), L6/L7 (`isMonotonic`, `flags`: zero occurrences), L8
(exactly **3** payload-less exit-3 caller-error paths per `--json` writer, derived), L9,
L13 (`readOTLP` materialises the whole export), L15, L16 (`claude_code.go:729` sets only
Name/Timestamp/Success), L19, L22, L23/L24 (`testdata/otlp/README.md:26` `[DOCS]`), L25
(`APIKey` declared at `types.go:85`, read nowhere). Both claimed closures check out: **L14**
— the canonicalisation test and `kvlist_attribute_series.ndjson` already present at
`v0.4.3`; **L21** — 12.6% correct at the ledger's pin (`v0.4.1` = 12.6% exactly), 94.5% now.

**The three version asserts are non-vacuous — proven RED independently.** Reverting the
five manifests to `10326b3` gives **exactly 5 `FAIL:` lines** on **both** bashes, exit 1,
`48 passed, 5 failed` (= 53). Each names a real surface. Separately mutating
`AdapterVersion` → 1 independent FAIL, confirming `CHANGELOG:118`'s "asserts both and
asserts them separately". Also driven live: all four `-o` refusals exit 1, unresolvable
path exits 3, `$HOME/.claude-notes/x.md` accepted, `$HOME/drafts → .claude/skills`
refused, and **exit 7 reproduced on both shells**.

---

## F1 · HIGH · REAL · REPRODUCED · `tests/test_harness.sh:211`

```sh
assert "the audit examined every suite's assertions, not an empty reading" \
  test "${examined:-0}" -ge 200
```

**Written: `200`. Derived: `727`** (`292+101+108+48+120+38+20`, per-suite via
`tests/lib/audit-suites.sh`). A floor **3.6× below** the real count.

Proven, not reasoned. Injecting a plausible limit regression into the audit
(`FNR > 400 { next }`) dropped it from **727 to 234** examined call sites —
`test_install.sh` audited at **zero** — and then:

```
test_f01 exit=0 :: 1868 passed, 0 failed      test_skill   exit=0 :: 53 passed, 0 failed
test_f02 exit=0 :: 350 passed, 0 failed       test_install exit=0 :: 48 passed, 0 failed
test_harness exit=0 :: 174 passed, 0 failed   test_walk    exit=0 :: 20 passed, 0 failed
test_rewrite exit=0 :: 136 passed, 0 failed   TOTAL = 2649  (claimed 2649)
```

**Green on both bashes with 68% of the audited surface gone.** This audit is the *only*
thing standing behind non-vacuity for the 819 `assert_value` sites
`tests/lib/harness.sh:~150` says "rests on" it.

**Invariant:** the number of assertion call sites the audit examined must **equal** the
number present in the suites — a denominator compared for equality, not a bound.

**This file already contains the repair, 60 lines above, applied to a different floor.**
`test_harness.sh:140-148`: *"The number used to be a floor — `-ge 6` … the floor still
passed, one suite went unexamined."* The fix was applied to the suite count and not to the
site count in the same file.

## F2 · HIGH · REAL · REPRODUCED · `tests/test_f01.sh:736`

```sh
"$([[ "${guard_definition_count:-0}" -ge 6 ]] && echo true || echo false)"
```

**Written: `6`. Derived: `16`.**

Re-spelling 8 of the 16 guard primitives in `verdict-guard.sh` as `name ()  {` — a style
the derivation's `sed` misses, while `verdict_guard_ready` still loads fine — dropped the
derived set 16 → 8:

```
guard primitives examined: json_string cannot_compute mktemp_answers tool_answers
                           require_tool skill_frontmatter skill_body verdict_guard_ready
1844 passed, 0 failed        exit=0
```

Half the guard primitives lost their drop-one mutation coverage and **24 assertions
silently vanished from the published 1868 with zero failures reported.** No suite asserts
its own total, so the seven per-suite figures in `CHANGELOG:205` have nothing holding them.

This also **falsifies the documented exemption** at `test_f01.sh:1573-1577` — *"The other
floors in this file are not this shape … each sits beside an exact comparison that does
the real work"* — which is false for this one. There is no exact comparison beside it.

**Shared root of F1+F2:** a count asserted as a hand-written floor instead of derived and
compared for equality. It is the precedent the mandate names (`-ge 45` when the product
was 60), recurring at **200-vs-727** and **6-vs-16**. Both also contradict
`CHANGELOG:216-218`: *"Every slice in this release proved its own checks by mutation."*
**These two do not redden.**

## F3 · MEDIUM · REAL · `CHANGELOG.md:18-21`, `RELEASE_NOTES.md`

> "copying all three into `profiler/` at head and running `go build ./...`: they do not
> compile … and all three drafts read them"

**Written: all three. Derived: two of three.** `codex.go` and `cursor.go` fail on exactly
those four identifiers; **`devin.go` reads none of the four and `go build ./...` exits 0.**

The four identifiers *are* defined at `541af3e` and absent at `v0.4.1`, so "removed by
v0.4.1's OTLP rewrite" is correct. Devin's *other* stated defect is true and verified:
`devin.go:37` → `caps[MetricTokens] = SourceSessionData`, `devin.go:114` →
`UnknownTokenResult(reason)`. **The decision not to land it stands; the compile
measurement behind it does not.**

## F4 · MEDIUM · REAL · `CHANGELOG.md:189`, `RELEASE_NOTES.md`, and the commit subject

> "Four deferral markers reading '0.5.0' are re-pointed"

**Written: `4`. Derived: `7`** — 3 in `README.md` (`:98`, `:186`, `:323`) and 4 in
`docs/profiler-spec.md` (`:788`, `:792`, `:806`, `:837`), each a `-`/`+` pair in this
commit's diff. Two numbers wearing one name: `4` is right only counting *subjects*, and
under that reading the enumeration below names just **three**, omitting
`docs/profiler-spec.md:806`. **No consistent reading makes the sentence and its own list
agree.**

Good news: the sweep found **no forward-looking 0.5.0 promise left in any prose file.**

## F5 · MEDIUM · REAL · `profiler/cmd/main_test.go:63`

> "`go test -cover ./cmd` reports what the suite actually reaches: 8.5% becomes 91.5%,
> measured at `e26b2a5`."

**91.5% is the figure at neither commit.** At `e26b2a5` the `GOCOVERDIR` propagation
**does not exist yet** (`grep GOCOVERDIR cmd/main_test.go` empty there) and `-cover` =
**8.5%**, so 8.5% → 8.5%. The mechanism *and* this comment both landed at `d77d90e`
(`git log -S'91.5%'`), where the measurement is **94.5%**. At head: **94.5%**. Outside the
three prose files, but a number, in the coverage story `CHANGELOG:157-164` tells.

## F6 · LOW · REAL · `CHANGELOG.md:121-123`, `docs/profiler-spec.md:308`

> "Every key this release adds — `error_type`, `count`, `id`, the four attribution fields
> and `estimated_context_tokens` — is optional and absent when it was not read."

**Written: 8 keys. Derived: 9.** Set-diffing `json:"…"` tags `v0.4.3` → head adds
**`total`** (`types.go:284`), unnamed — and it is the one new key *without* `omitempty`,
i.e. precisely the apparent counterexample to the sentence's predicate. The predicate
survives transitively (parent `*EstimatedTokens` is `omitempty`, and
`TestAProfileThatEstimatedNothingHasNoEstimateKey` passes), so this is a completeness gap
in an enumeration that says "every". ("The four attribution fields" is **correct**:
`v0.4.3` had 2, head has 6.)

## F7 · LOW · REAL · `profiler/cmd/main.go:118` — an eighth false forward promise

`// Making the failure audible is the repair; making it branchable is 0.5.0.`

An **extension** of the confirmed class, same root: S10 re-pointed both *prose* copies of
this exact sentence (`README.md:98`, `docs/profiler-spec.md:837`, both now "0.5.0 did not
add one") and left the code's copy. `CHANGELOG:311-313` confirms it is still open. No
ninth exists: `profiler/profiler_test.go:325`
(`retiredActivationDeferral = "Reading it is 0.5.0."`) is a legitimate negative control
asserting the sentence is *gone* — **do not flag it**.

## F8 · LOW · REAL · `CHANGELOG.md:62`, `RELEASE_NOTES.md`

> "a nanosecond epoch is about 1.7×10¹⁸"

Derived **1.790×10¹⁸**; fixtures carry `1789332594304000000`. To one decimal that is 1.8,
so 1.7 is a truncation rather than a rounding. Hedged by "about"; the load-bearing point
(2⁵³ ≈ 9.007×10¹⁵) is untouched. Cosmetic.

## F9 · LOW · REAL · `CHANGELOG.md:208`, `RELEASE_NOTES.md`

> "275 top-level Go test functions and 669 subtests **pass** under `-race`"

275 = **274 PASS + 1 SKIP**; all 669 subtests pass. The skip is named in the very next
sentence, so the qualification is present — but the total is stated as "pass" while its
parts are not.

## F10 · LOW · REAL · `CHANGELOG.md:22-26`

All four citations resolve **correctly** (`cursor.go:60` → `caps[MetricTokens] =
SourceSQLite`; `:150` → the exact quoted string; `devin.go:37`, `:114` ✓) — but they
resolve at `1e4e845`/`main`, and **no such file exists at PR head.** A reader of the 0.5.0
changelog cannot resolve `cursor.go:60`. The prose names no ref.

---

## SUSPECTED

**F11 · LOW — the remaining floors (same root as F1/F2).** Complete enumeration, floor vs
derived: `test_f01.sh:1290` **15 / 20**; `:2010` **8 / 10**; `:2188` **6 / 8**; `:2385`
**5 / 5**; `:2592` **8 / 8**; `:2696` **6 / 6**; `test_skill.sh:116` **2 / 2**. The
documented exemption genuinely *holds* for `:2010` (the `readme_missing` set comparison
catches a shrinking read) and `:2696` (bidirectional witness checks). It **does not hold**
for `:736` (F2, proven). **`:1290` is the untested one** — its neighbour `unprobed_tools`
is one-directional and would not catch a shrinking walk. The rest are floor == real today
and will drift silently.

**F12 · LOW · `CHANGELOG.md:104`** — "instead of rebuilt by hand in **five** consecutive
slices". **No tree derivation exists**: `tests/lib/mutation-runner.sh` landed only in S9,
and S1–S8 added *zero* mutation machinery to `tests/`. The claim is about uncommitted
worker state. S9's own record says "the two carried from S5–S8", so S5–S9 = five
consecutive slices is a consistent reading. Flagged only because it is the one figure in
the added lines that could not be driven against the tree.

## Passing note (not a number)

`tests/lib/harness.sh:20` directs the reader to `tests/lib/audit-assertions.sh`; the file
is `tests/lib/audit-suites.sh`. Outside this lens, found while deriving F1.

## Where the lens went dry

Second full pass over all 88 lines and 61 tokens added nothing. Every sum relation was
tested: `1868+350+174+136+53+48+20 = 2649` ✓, `233+34+8 = 275` ✓, `636+33 = 669` ✓,
`15 − 2 = 13` with a 13-item list ✓, and 0.4.3's own `86+7 = 93` / `422+30 = 452` match
the tag ✓. **The three relations that fail are F3, F4 and F6 — and all three fail the same
way: a total quantified over parts that were never counted** ("all three", "four markers",
"every key"). That, plus the floor root behind F1/F2/F11, is the shared root one fix
should close: **derive the count from the artifact and compare it for equality, in prose
and in asserts alike.** All fix directions advisory — the `bug-fixer` owns re-deriving.
