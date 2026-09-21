# S5 record — the comparison, and the refusal that is the reason it can be trusted

**Implementer** senior-implementer · **Date** 2026-09-20 · **Base** `origin/main` = `a7de555`
**Branch** `feat(profiler)/0.5.0-compare` · **Commit** `83f5c5e` · **PR** [#17](https://github.com/Okja-Engineering/skill-architect/pull/17) (open, not draft, base `main`)
**Worktree** `<scratch>/wt-s5-compare` (session scratchpad)
**Status** delivered, not merged. The user merges. Nothing tagged.

---

## 1. What landed

7 files, 1786 insertions, 16 deletions. Two of the files are new.

| File | What |
|---|---|
| `profiler/compare.go` | **new** — the comparison, the two refusals, `LoadProfile` |
| `profiler/compare_test.go` | **new** — 26 tests over the comparison contract |
| `profiler/cmd/main.go` | the `compare` subcommand, its exit contract, the help entry |
| `profiler/cmd/main_test.go` | the CLI half of `compare`; the help/dispatcher derivation; S4's flagged inaccuracy fixed |
| `docs/profiler-spec.md` | `## Comparison contract`, and one false claim corrected |
| `README.md` | "Comparing two profiles"; the two false claims corrected |
| `.out-of-scope.md` | the bullet S1 deferred here, now true |

`git add` was those seven, named. No `git add -A`, no `commit -a`.

---

## 2. How the adapter-version refusal is implemented

### 2.1 The rule is one function, and it owns the whole of it

```go
func adapterVersionRefusal(baseline, candidate string) string
```

Two profiles may be compared only when they name the **same, non-empty**
`capability.adapter_version`. Different versions is one refusal naming both;
two empty versions is a second, because `"" == ""` is agreement between two
unknowns and neither side can show it was read the way the other was. Both
reasons come out of one function, so there is one place the rule lives.

### 2.2 The refusal is upstream of every subtraction, not a flag set afterwards

This is the whole of the design decision, and it is what §5's M2 proves.

The obvious shape is: compare everything, then set `report.Comparable = false`
when the versions differ. That leaves each `MetricComparison` saying
`comparable: true` **with the delta still in it**, so anything reading one
metric out of the report — which is what a dashboard or a script does — gets
the fabricated number anyway. The top-level flag is not where the number is.

So the pair verdict is computed first and handed to a `comparator` that every
signal passes through:

```go
c := comparator{refusal: report.Refusal, baseline: profileSides(baseline), candidate: profileSides(candidate)}
```

and `c.pair(metric)` returns *not comparable* before it looks at anything else.
Under a refusal no delta is computed at all, every metric carries the same
reason, and `Baseline`/`Candidate`/`Delta` are absent together.

**`report.Comparable` is then derived, not asserted** — true when some signal
was compared. A refusal makes every signal incomparable by construction, so the
top-level flag needs no clause of its own, which is what stops the two ever
disagreeing. M25 (declare it true without deriving) kills 8 tests.

### 2.3 Both versions reach the report, not only the refusal string

`ProfileRef` gained `adapter_version`. A stored comparison has to stay
interpretable later, and "which reader produced this side" is the question the
refusal is about.

### 2.4 Mutation proof — 25 mutations, every one reverted and watched

Restores were done by `cp` from a scratch backup. **No `checkout`, `reset`,
`clean`, `stash` or `rebase` was run anywhere**, in the worktree or out of it.

| # | Mutation | Result |
|---|---|---|
| M1 | the version refusal removed entirely | KILLED — unit tests **and** the CLI test: `exit status = 0, want 2`, `timing: comparable=true delta=map[total_ms:-4000]` |
| **M2** | **the bolt-on shape: compare everything, then flag the report incomparable** | **KILLED — every metric "reported comparable under a refused pair" and carried its delta.** The design, not just the check |
| M3 | the refusal names only the baseline's version | KILLED — unit + CLI |
| M4 | the empty-version clause removed | KILLED |
| M5 | the source-mismatch refusal removed | KILLED — unit + the CLI exit-status case |
| M6 | an unread signal's source stays `""` instead of `none` | KILLED — 4 |
| M6b | the sources gain `omitempty`, so an unread side has no key | KILLED — 6, including the CLI wire test |
| M7 | an unread token count treated as zero (the fabrication) | KILLED — 2 |
| M8 | a tool-call entry's aggregate `Count` ignored | KILLED |
| M9 | `round2` back to truncate-after-adding-a-half | KILLED — the negative success-rate delta |
| M10 | the tokens "no shared count" refusal removed | KILLED (see §5 — this one first read as green) |
| M11 | a result marked `present` with no value is trusted | KILLED — 4 |
| M12 | `compare` stops covering one of the profile's signals | KILLED — the derived coverage test |
| M13 | **control:** the signal derivation finds nothing | KILLED by the `> 0` guard |
| M14 | `compare` exits 0 even when nothing is comparable | KILLED — 5 |
| M15 | the refusal is no longer said on stderr | KILLED |
| M16 | `LoadProfile` stops validating values | KILLED |
| M17 | `LoadProfile` stops checking the schema | KILLED — 6, including two usage-error cases |
| M18 | `compare` dispatched and left out of the help | KILLED |
| M19 | `compare` in the help and not dispatched | KILLED — 11 |
| M20 | **control:** the dispatcher derivation reads nothing | KILLED by the `> 0` guard |
| M21 | **control:** the help derivation reads nothing | KILLED by the `> 0` guard |
| M22 | the estimate loses its ordinal note | KILLED |
| M23 | activation compared by count alone | KILLED |
| M24 | a snapshot/harness difference becomes a refusal | KILLED |
| M25 | the report declares itself comparable without deriving it | KILLED — 8 |

Run live against the built binary, a stored 0.4.1 profile of 1000 input tokens
against a fresh 0.5.0 one of 500: `comparable: false`, the refusal on stdout and
stderr, **`-500` appears nowhere in the output**, exit 2.

---

## 3. The CLI tests, and what coverage they actually reach

### 3.1 What was written

The comparison's own unit tests cannot reach the dispatcher, the exit status,
the stderr refusal or the JSON a consumer parses. So the command is tested by
building the binary and running it, as the existing `capture` tests do:

- `TestCompare_RefusesAProfileStoredByAnotherAdapterVersion` — the headline,
  end to end, **with its control**: the same two profiles at one adapter
  version must compare and must produce the delta, or the refusal proves
  nothing.
- `TestCompare_ExitStatusSaysWhetherAnythingWasCompared` — 4 cases: comparable,
  two versions, a profile that read nothing, two sources. Each also asserts the
  status agrees with the report's own `comparable`.
- `TestCompare_EveryMetricInTheReportNamesBothSources` — over the wire.
- 8 new cases on `TestUsageErrorsExitOneSoThatTwoMeansOneThing`: neither
  profile, one profile, an unknown flag, a path that is not there (×2), a
  document that is not a profile (×2).
- 2 new cases on `TestRequestedHelpExitsZero`.
- `TestTheHelpListsEveryCommandTheDispatcherAccepts` — §3.3.

### 3.2 The coverage number in the plan is an artifact. The real one is 89.5%.

The plan and deferred-ledger L21 say `profiler/cmd` sits at 12.6%; at `a7de555`
`go test -cover` reports **11.6%**, and **after this slice it reports 8.5%**.

**That number does not measure what the CLI tests exercise.** They build the
binary with `go build` and run it as a subprocess, and `go test -cover` counts
only statements executed *in the test process*. Adding `cmdCompare` therefore
*lowers* the percentage while raising the coverage.

Measured properly — built with `go build -cover -coverpkg=…/profiler/cmd`, run
with `GOCOVERDIR`, read back with `go tool covdata percent`:

| | base `a7de555` | head |
|---|---|---|
| `profiler/cmd`, subprocess counted | **89.5%** | **91.5%** |
| `profiler/cmd`, `go test -cover` | 11.6% | 8.5% |

The measurement was done in a scratch copy; **no test file was changed to make
it**. See §7.2 — the deferred ledger's L21 figure should be corrected, and the
`-cover` build is worth adopting.

### 3.3 The help and the dispatcher now derive from each other

Every remaining slice adds a subcommand. A command the dispatcher accepts and
the help does not list is one nobody can find; a command the help lists and the
dispatcher rejects does not exist. Both sets are now derived — one read out of
`main.go`'s own `switch` with `go/ast` (the technique S3 introduced), one out of
the help text the binary prints — and asserted equal, with a `> 0` guard on each
derivation. The two flag spellings of `help` are excluded as aliases, and the
help is separately asserted to still mention them, so the exclusion cannot hide
them.

M18, M19, M20 and M21 prove all four directions.

---

## 4. Every false documentation claim found, including one beyond the brief's three

The brief named three and said to sweep rather than trust the list. The sweep
(`grep -rn -i 'comparison engine|paired compar|F04|not yet built|compare'` over
every `.md`, `.sh` and `.json` in the tree, plus a read of the spec's
serialization rules) found a fourth.

| # | Claim | Where | Correction |
|---|---|---|---|
| 1 | "the comparison engine is not yet built" | `README.md:629` | now says what *is* still out of scope: scheduling and executing the paired runs |
| 2 | "the integration point for **future** paired comparisons (F04)" | `README.md:320` | replaced by a "Comparing two profiles" section documenting the command, the two refusals, the notes, the delta rule and the exit codes |
| 3 | `.out-of-scope.md`'s F04 bullet, held back by S1 as premature | `.out-of-scope.md:8` | landed verbatim from the snapshot. **Only that bullet** — `main`'s "Guarantee security" is untouched and the snapshot's `skillgate` bullet was not taken |
| **4** | **"F04 is expected to compare only profiles carrying the same `snapshot_hash`"** | `docs/profiler-spec.md:309` | **found by sweeping.** It is false and must stay false: comparing a skill before and after a change is the paired comparison's entire purpose, so two snapshot ids are a `note`, never a refusal. The line now says that, and points at the comparison contract for what *is* refused |

**Claims checked and left alone, with the reason:**

- `README.md:35–37` — the three harnesses still say "Planned". True; no adapter
  ships (D6).
- `README.md:315`, `docs/profiler-spec.md:394/398/412` — "tracked for 0.5.0"
  deferrals (a bundled receiver, the skipped-record count, `active_time`). This
  slice closes none of them, so they remain true. **S10 must re-check them**,
  because the release commit is where "0.5.0" stops being the future.
- `profiler/types.go`'s package comment — "the comparison engine (F04) reads
  profiles without knowing which adapter produced them". Still true: `compare`
  compares two version *strings* and is indifferent to which adapter wrote
  them. `types.go` is outside this slice's `git add` list and was not touched.
- `CHANGELOG.md` / `RELEASE_NOTES.md` — their 0.4.x entries are historical
  records of those releases and remain true of them. S10's.
- `README.md:28`'s "(tokens, tool calls, timing)" summary omits
  `skill_activation`, which S4 added. Incomplete rather than false, and not
  falsified by this commit. **Flagged for S10** (§7.4).

No shell suite reads any of these files' prose (verified: `tests/test_walk.sh`
walks the skills; `test_harness.sh` names `README.md` and `.out-of-scope.md`
only in its stageability list), which is why the suite counts did not move.

---

## 5. A vacuous check caught — in the mutation harness itself

**M10 first read as green.** The mutation was `if shared == 0 {` → `if false {`,
which makes `shared` an unused variable: the package did not build, so the tests
never ran, and my runner's `grep -E '^--- FAIL'` matched nothing and printed
nothing. **No output read as "the check survived".**

That is the S3 lesson — "a quiet grep reads as clean" — reproduced by a
different mechanism one slice later, and it was caught only because the brief
required proving every check by reverting it and I then asked why M10 alone was
silent.

The runner now reports an **explicit verdict in every direction**: `MUTATION DID
NOT BUILD — it was never executed`, `SURVIVED — the suites are GREEN`, or
`KILLED — n failing tests`. It was proved in both directions before being
trusted (a no-op mutation → SURVIVED; a non-building one → the build verdict).
M10 was then re-run with a mutation that compiles (`shared < 0`) and **KILLED**.

M3 was also caught by the hardened runner on its re-run — a `fmt.Sprintf` arg
mismatch that `go vet` refuses — and re-run with a well-formed message.
**M3 through M9 and M11 had all printed `--- FAIL` lines, which a non-building
package cannot do, so those results stood.**

**The lesson for the remaining slices:** the rule "a derived denominator needs a
`> 0` guard" (S4 §9.4) is one instance of a wider one. *Any* check whose passing
signal is the absence of output can be satisfied by not running. The verdict has
to be printed in both directions. Both vacuity controls this slice wrote (M13,
M20, M21) followed S4's rule and fired correctly; the one that failed was the
harness, which nobody had written a rule for.

---

## 6. Porting from the snapshot: what changed, and why

Both files were hash-checked against `snapshot-0.5.0.manifest` before being read:

- `profiler/compare.go` — `b344940f…1daa0`, matches.
- `profiler/compare_test.go` — `557a11bf…10dc4`, matches.
- `profiler/cmd/main.go` — `b87626c4…b4789`, matches.
- `docs/profiler-spec.md` — `0cfb63bb…c5c96`, matches.
- `README.md` — `33980fef…7420f`, matches.
- `.out-of-scope.md` — `ec51f66d…c483c`, matches.

**Nothing was copied except the one `.out-of-scope.md` bullet.** The snapshot's
`compare.go` was read for intent and ported by hand, and the warning held for
the fourth time: its assumptions about the profile shape are three releases out
of date.

| What the snapshot does | Why it could not be ported | What landed |
|---|---|---|
| `TokenCounts{Input: c.Input - b.Input, …}` over **value** ints | v0.4.2 made every count a `*int`. Subtracting through a nil is the proto3-zero fabrication those releases removed | `tokenDelta` subtracts only the counts **both** sides read; a count one side never read has **no key in the delta**, and both values are carried beside it. When the two share no count, `tokens` is not comparable and says so — an all-absent delta object reads as "no change" |
| `CacheWrite` | the rename is dropped by user decision (D5) | `CacheCreation`. `grep -rn 'cache_write\|CacheWrite'` over `profiler/ docs/ README.md`: **no matches** |
| `EstimatedContextTokens` held **by value** | S3 made it `*EstimatedTokensResult`; a value is the D5/R5 trap | resolved through `main`'s own `orAbsent()`, so an absent estimate is `unknown`/`none` and never a nil dereference |
| `len(calls)` as the tool-call count | S3 added `ToolCallEntry.Count`, the aggregate for a delta-aggregated source. Counting entries under-reports such a source by however much it aggregated | `countToolCalls` counts **calls**: `max(1, entry.Count)` |
| no `skill_activation` in the profile's own vocabulary | S4 added the signal | the comparison covers it, and a test derives the metric set from `Profile.SignalStates()` so a seventh signal fails rather than passing unnoticed |
| `round2(f) = float64(int(f*100+0.5))/100` | truncates towards zero, so it rounds **negatives** wrong — and a success-rate delta is negative exactly when the candidate did worse | `math.Round(f*100)/100` |
| no adapter-version check at all | the gap the plan exists to close | §2 |
| `guardPair(metric, bState, cState, bMissing, cMissing, bSrc, cSrc)` — seven arguments | `main`'s `RawMetricResult` already carries state and source together | one `profileSides(p)` derivation feeding `c.pair(metric)`; the same derivation is what `LoadProfile` validates against, so the comparison and the file check cannot disagree about what a signal is |
| `cmdCompare` with `flag.ExitOnError` | exits **2** on a flag typo, and 2 is "nothing was comparable" | `flag.ContinueOnError` through `main`'s own `parseFlags`, so help exits 0 and a typo exits 1 |
| `cmdCompare --output <path>`, printing "comparison written to: …" | extra surface that a shell redirect already does, and it returned 0 regardless of whether anything compared | **dropped.** stdout only |
| `cmdCompare` always exits 0 | a wrapper cannot tell a refused comparison from a real one | `compareExitCode`: 0 compared, 2 nothing comparable |
| the spec's comparison contract, 4 bullets | true as far as it went | ported and extended to 7 rules plus the exit contract |

**One rule taken from `main`'s types rather than from the snapshot.** A
list-valued result marked `present` with no entries is refused: `types.go` says
"a result with no entries is never present", and `claude_code.go` routes
`len(calls) == 0` to `unknown`. `null` and `[]` are the same absence once a file
has been through a decoder, so both are refused (M11 proves both spellings).

---

## 7. Findings — what should change before slice six

### 7.1 A check whose pass signal is silence can pass by not running. **[carry — this is the one]**

§5. The vacuous-derived-denominator rule has now been joined by its parent
class. S6 writes CLI tests and a mutation harness of its own: **print the
verdict in both directions, and make "did not run" a distinct, loud outcome from
"survived".**

### 7.2 `profiler/cmd`'s coverage figure is an artifact, and the ledger records it as fact. **[decide]**

§3.2. L21's 12.6% is `go test -cover` failing to see a subprocess; the real
figure is **89.5% at base, 91.5% at head**. Two consequences:

1. **The ledger entry should be corrected**, or the next slice will "close a
   gap" that is not there — and will see the number *fall* when it adds a
   subcommand, exactly as it did here.
2. **`buildProfiler` could build with `-cover` when `GOCOVERDIR` is set**, four
   lines, making the honest number available on demand and in CI. I did not do
   it: it is test infrastructure beyond this slice, and the existing `capture`
   tests have had the same blind spot for three releases. **Worth doing in S6**,
   which adds `experiment` and will face the same question.

### 7.3 Cross-harness comparison is a note, and that is a decision with a shelf life. **[carry to the slice that lands a second adapter]**

`harness` differing is a `note`, per the snapshot's contract and the plan. It
costs nothing today — `claude_code` is the only registered adapter, so a
cross-harness pair cannot arise from this release's own tools. It stops being
free the moment a second adapter ships: two harnesses both reporting `otel`
would pass the source check while measuring with different meters. **The slice
that registers a second adapter must decide whether `harness` joins the pair
refusal.** The shape is ready for it — one more clause in
`adapterVersionRefusal`'s place, and every metric inherits it.

### 7.4 `README.md:28`'s signal list is one release stale. **[S10]**

"captures runtime signals (tokens, tool calls, timing)" omits
`skill_activation`, which S4 landed, and the harness table row says the same.
Incomplete rather than false, and not falsified by this commit, so it was left
alone — but **S10's prose gate should close it**, together with the three
"tracked for 0.5.0" deferrals in §4 that the release commit turns into claims
about the past.

### 7.5 The refusal reason is repeated on all six metrics. **[note]**

Deliberate: a consumer reading one metric out of the report must not be able to
miss it. It makes a refused report verbose — six copies of one sentence. If that
becomes a complaint, the fix is a per-metric `reason_ref` pointing at the
top-level refusal, which costs a reader an indirection. Not worth it at six.

### 7.6 Smaller notes

- `successRate`'s zero-denominator guard is unreachable through
  `CompareProfiles` (a present tool-call result must carry entries). It is kept
  because the alternative is a `NaN` in a document somebody reads, and it is
  **asserted directly** rather than left as an unfalsifiable branch.
- `ComparisonReport.Refusal` is a single string, not a slice. There is exactly
  one pair-level rule today, and a one-element slice is speculative generality.
  The slice that adds a second (§7.3) should widen it then.
- `compare` reads `Profile` through `json.Unmarshal`, which has no custom
  `UnmarshalJSON` — the schema check in `LoadProfile` is what catches a
  document that is not a profile. M17 proves it load-bearing.
- The comparison report round-trips **modulo key ordering**, the rule the
  profile format already states. Ordering is not preserved because a comparison
  value is an `any`: it leaves as a struct in field order and returns as a map,
  which Go marshals sorted. The test compares the two documents, not the two
  byte strings.

---

## 8. Verification — measured, not carried

Every base figure was re-derived at `a7de555` in its own worktree before any
edit, and reproduced S4's record exactly (2555 / 114 top-level / 607 PASS).

### Shell suites, both shells

| suite | base | head, bash 3.2.57 | head, bash 5.3.15 |
|---|---|---|---|
| test_f01 | 1868 | 1868 | 1868 |
| test_f02 | 350 | 350 | 350 |
| test_harness | 131 | 131 | 131 |
| test_install | 48 | 48 | 48 |
| test_rewrite | 85 | 85 | 85 |
| test_skill | 53 | 53 | 53 |
| test_walk | 20 | 20 | 20 |
| **total** | **2555** | **2555 passed, 0 failed** | **2555 passed, 0 failed** |

Unchanged, and that is the expected result rather than a null one: this slice
touches no shell surface. No suite reads the prose that changed (§4).

### Go, in `profiler/`

| gate | base | head |
|---|---|---|
| `gofmt -l .` | empty | empty |
| `go build ./...` | clean | clean |
| `go vet ./...` | clean | clean |
| `go test -race -count=1 ./...` | ok | **ok**, both packages |
| top-level tests | 114 | **144** (+30) |
| PASS lines incl. subtests | 607 | **662** (+55) |
| failures | 0 | **0** |
| `profiler` coverage | 95.5% | **96.7%** |
| `profiler/cmd` coverage, subprocess counted | 89.5% | **91.5%** |

The 30 new top-level tests: **26** in `compare_test.go` (the two refusals, the
source rules, the delta rules, the notes, the report as an artifact and
`LoadProfile`) and **4** in `cmd/main_test.go` (the three `compare` command
tests and the help/dispatcher derivation). The other new cases are rows added
to tables that already existed — 8 usage errors, 2 help paths, and the capture
exit-status case §4 corrected — which is why the subtest count rose by 55.

### CI

`gh pr checks 17` → **pass, 2m55s** (ubuntu-latest, bash 5, Linux git).

### Hygiene

- `git add` was **seven named paths**. No `git add -A`, no `commit -a`.
- `tests/lib/out-of-scope-check.sh` run over the staged index: `out-of-scope
  check: clean`, exit 0 — the committed script, not a re-derivation (S3 §10.4).
  **Proved non-vacuous against this index**: staging `docs/skillgate-probe.md`
  produced `BLOCKED: out-of-scope path staged`, exit 1, naming the path; it was
  unstaged and deleted and the check returned clean.
- Author and committer `imagineux <imagineux@gmail.com>`. AI-attribution grep
  over the commit message, the whole commit object, the PR title and the PR
  body: **0 matches**, before and after push.
- **No version surface touched.** `AdapterVersion` stays at S3's `0.5.0`,
  `ProfileSchema` stays `skill-architect/profile/v1`, the five plugin manifests
  stay `0.4.3` for S10. `git diff a7de555..HEAD --name-only` names no manifest.
- `grep -rn 'cache_write\|CacheWrite\|cacheWrite' profiler/ docs/ README.md`:
  **no matches**.
- No `checkout`, `reset`, `clean`, `stash` or `rebase` anywhere, including
  during 25 mutation runs — restores were `cp` from a scratch backup, and the
  worktree was diffed against that backup afterwards to prove every mutation was
  undone.
- Nothing installed, nothing reconfigured, no PATH mirror built, no live config
  directory written. Every test file lives in the suite's own `t.TempDir()`.
- **Primary working tree verified untouched after the slice:** still `0a83615`,
  47 `git status --porcelain` lines — identical to the figure S3 and S4 recorded.

---

## 9. Not done, and why

| Not done | Why |
|---|---|
| Merging or tagging | The user merges. Nothing tagged. |
| Touching any version surface | §8. S3 set `AdapterVersion`; the manifests are S10's. |
| `--output <path>` on `compare` | §6 — a shell redirect does it, and it defeated the exit contract. Named, not silently dropped. |
| Refusing a cross-`harness` pair | §7.3 — not decidable while one adapter ships, and it costs nothing today. |
| Building the CLI with `-cover` in the committed tests | §7.2 — test infrastructure beyond this slice. Measured in a scratch copy instead; recommended for S6. |
| `profiler/experiment.go` and the `experiment` subcommand | S6's. This slice lands the contract `run` must agree with. |
| `CHANGELOG.md` / `RELEASE_NOTES.md` | S10's. Their 0.4.x entries are historical and remain true of those releases. |
| Fixing `README.md:28`'s stale signal list | §7.4 — not falsified by this commit; flagged for S10's prose gate. |
| Rebuilding the snapshot | Frozen. Read-only, hash-checked, not written to. |
