# Lane B — C2 profiler provenance + G9-02 · record

**Branch** `fix/0.4.3-profiler-provenance`, cut from `origin/main` @ `71cc866` in a fresh
worktree. PR against `main`. Not merged, not tagged.

**Worktree** `…/31edaf48…/scratchpad/wt-provenance`. The primary tree was never touched.

---

## 1. Entries, dispositions, commits

| Entry | Sev | Disposition | Commit |
|---|---|---|---|
| **G2-01** — `Capture` ignores `sessionID`; every number in a session-stamped profile is drawn from every session in the export | P1 | **REAL, fixed** | repro `e8a7cdc` · fix `5e670fb` · reason wording `3e00395` |
| **G2-02** — foreign log events attributed to Claude Code (no scope filter; `eventName` re-qualifies a bare body) | P2 | **REAL, fixed in the same root repair** | repro `e8a7cdc` · fix `5e670fb` |
| **G2-03** — the comment above `extractToolCalls` claims "the reason below says so" for mid-run accepts | P3 | **REAL, fixed** (comment + a test pinning the behaviour it misdescribed) | `2b2abd4` |
| **G9-02** — `kvlistValue` has order-sensitive identity, so equal maps read as two series and cumulative totals **add** | P2 | **REAL, fixed**; deferral reason corrected | repro `ece6208` · fix `4bdc213` |

All six commits are on `origin/fix/0.4.3-profiler-provenance`. Author and committer
`imagineux <imagineux@gmail.com>` throughout; zero AI attribution trailers.

**Explicitly not touched**, per mandate: reject-vs-failed indistinguishability (G9-04),
the `api_error` span exclusion (ah F6), the int64 ceiling marker (G9-06), the unreachable
constructors (ah F8), the output-to-skill mapping reason (G5-01), and every version
surface — `AdapterVersion` is still `"0.4.2"`.

---

## 2. The root, as re-derived

The worklist's root analysis holds and I confirmed it against `71cc866`:
`resolve()` took no identity, so no extractor below it could scope anything, while the
result was stamped with a caller-asserted identity the data never justified.

I did **not** take the prescribed shape ("apply it in the extractors"). Applying the test
in three extractors leaves a fourth signal free to reintroduce the defect, which is the
same accretion the finding is an instance of. Instead the export is **projected onto the
provenance once**, in `profiler/provenance.go`, and the extractors now take a
`scopedExport` rather than an `otlpExport` — so an extractor cannot be handed the whole
file by accident and scoping is not a step any of them has to remember. `otlp.go` supplies
the mechanism (it owns scopes, attributes and series identity); `claude_code.go` supplies
the names. A second OTLP-speaking adapter gets provenance by supplying its own.

### Two deliberate deviations from the worklist's phrasing

**(a) The scope test is an exclusion, not an allowlist.** The worklist states the invariant
as "a record is Claude Code's only if its scope says so". Read absolutely that refuses a
record whose scope names nothing — and an instrumentation scope is optional in OTLP, a
receiver or a collector in the path is free not to carry one through, and **35 of the
scope entries across the committed fixtures carry no name at all**. Refusing those trades
a wrong number for *no* number on every pipeline that drops the scope, and the capability
report would say `none` for a perfectly good export. So: a scope that names no library is
read; a scope that positively names another product is not. The project's own rule for
exactly this shape is already written at `otlp.go`'s `otlpResource` — "an entry that
carries no resource at all and one that carries a resource with no attributes are one
resource rather than two … neither is a guess worth making". Absent is not different.
The `session.id` test still stands over every unscoped record, so nothing gets in free.

**(b) No new vendor constant.** The match is on the harness's own namespace component,
taken from `a.Name()` — the same token its signal names are qualified with — so
`com.anthropic.claude_code`, its `.events` and `.subagent` siblings and any later tail all
read, while `some.other.product` does not. An allowlist of the exact observed string would
silently zero every real capture the day Anthropic changed the tail. There is no second
copy of the harness identity in the code to drift from the first.

I also repaired the half of G2-02 the worklist's file:line did not name: `eventName`
stripped the `claude_code.` prefix and **unconditionally re-added it**, so a body naming a
bare `tool_result` was promoted into a `claude_code.tool_result`. That manufactures an
identity. The body carries the qualified name (`[OBSERVED]`, per the fixture README) and
is now taken as it stands; only the `event.name` attribute is qualified, because the short
form is its documented spelling. Without this half, a foreign record needs only to omit
its scope to get back in.

---

## 3. `--session`: the decision

**It stays required, and it becomes genuinely load-bearing.** "Required and ignored" does
not survive the fix in either direction:

- `Capture` now **refuses an empty `sessionID`** (`SessionIDRequiredError`), parallel to the
  existing `ExportFileUnsupportedError`. An empty id is not a session that matched nothing;
  it is no assertion, and a profile stamped with it could only report every session the
  export happens to carry. So a Go caller is told too — the CLI's requirement now *derives*
  from the adapter's contract instead of restating it.
- `probe` keeps **no** `--session`, and is documented as unscoped: it is asked what an
  export can yield without being told a session, so it answers for the export as a whole.
  The instrumentation-scope half still applies to it. The one place the session test is
  dropped is a named, greppable `anySession` field with a single call site
  (`probeProvenance`), unreachable from `Capture` — not an empty string quietly becoming a
  wildcard. An export holding two sessions can therefore probe `otel` while
  `capture --session <one it does not hold>` reports `unknown`; a profile never disagrees
  with *itself*, because its `capability` block comes from its own scoped read. Documented
  in `README.md` and `docs/profiler-spec.md`, and pinned in `provenance_test.go`.

---

## 4. Multi-session: before and after

`profiler/testdata/otlp/two_sessions.ndjson` — two sessions in one export, which is what
both documented capture routes produce (they append to one file by design, and route (a)
listens on the standard OTLP port). Session A spent 48000/9100 over 12 s; session B sits
beside it with 1200/340, 27.8 hours later.

Measured with a `71cc866` build against the `HEAD` build, same file, same flags:

```
--- two_sessions.ndjson  --session 22222222-2222-4222-8222-222222222222 (A)
 BEFORE  tokens     : present {"input": 49200, "output": 9440}
         tool_calls : present ['Read', 'Bash', 'GrepInAnotherSession']
         timing     : present 99999000 ms  (27.8 hours)
 AFTER   tokens     : present {"input": 48000, "output": 9100}
         tool_calls : present ['Read', 'Bash']
         timing     : present 12000 ms

--- two_sessions.ndjson  --session 33333333-3333-4333-8333-333333333333 (B)
 BEFORE  tokens     : present {"input": 49200, "output": 9440}      <- identical to A's
         tool_calls : present ['Read', 'Bash', 'GrepInAnotherSession']
         timing     : present 99999000 ms
 AFTER   tokens     : present {"input": 1200, "output": 340}
         tool_calls : present ['GrepInAnotherSession']
         timing     : present 0 ms

--- two_sessions.ndjson  --session 44444444-4444-4444-8444-444444444444 (absent)
 BEFORE  tokens     : present {"input": 49200, "output": 9440}      <- confident, and nobody's
         tool_calls : present ['Read', 'Bash', 'GrepInAnotherSession']
         timing     : present 99999000 ms
 AFTER   tokens     : unknown  no claude_code.token.usage metric found in OTel export;
                      the export also carries 4 data points not carrying session.id 4444...
         tool_calls : unknown  no claude_code.tool_result or claude_code.tool_decision log
                      events found in OTel export; the export also carries 6 log records
                      not carrying session.id 4444...
         timing     : unknown  no claude_code.api_request log events found in OTel export;
                      the export also carries 6 log records not carrying session.id 4444...
         capability.tokens = none
```

The `49200 / 9440` against a real `48000 / 9100`, and the `99999000 ms` span, reproduce the
audit's figures exactly.

`profiler/testdata/otlp/foreign_scope.json` — one session id on **every** record, so only
scope and event naming can tell three of them apart:

```
 BEFORE  tokens     : present {"input": 8000}
         tool_calls : present ['Read', 'ViaEventName', 'BareBodyTool', 'SomeOtherProductTool']
         timing     : present 99999000 ms
 AFTER   tokens     : present {"input": 1000}
         tool_calls : present ['Read', 'ViaEventName']
         timing     : present 0 ms
```

`ViaEventName` carries no body and is named by its `event.name` attribute in the short
form — the documented spelling — and is still read. That is the guard against a repair
that over-filters.

`profiler/testdata/otlp/kvlist_attribute_series.ndjson` (G9-02) — two cumulative points
whose two-member `kvlistValue` arrives in opposite orders, plus the same pair with that map
nested inside an `arrayValue`:

```
 BEFORE  tokens : present {"input": 300}     <- four series; two running totals added twice
 AFTER   tokens : present {"input": 150}     <- two series, holding 100 and 50
```

---

## 5. Fixtures: what changed, and why no result did

**No pre-existing fixture's result changed. All 55 verified byte-identical**, comparing a
`71cc866` binary against the `HEAD` binary over every field of the profile except
`profiled_at`, `capability.probed_at` and the caller-supplied `session_id`, plus
`probe`'s capability map over all 55:

```
pre-existing fixtures: identical=55 changed=0
probe capability changes: 0
```

Fixture **content** did change, in one mechanical way, to keep those results identical:

| Change | Files | Records |
|---|---|---|
| `session.id` added as the first attribute of each log record | 14 | 31 |
| `session.id` added as the first attribute of each metric data point | 12 | 22 |

All use the one session id those fixtures were already a capture of,
`00000000-0000-4000-8000-000000000001`. The justification is the fixtures' own premise:
`README.md:234-236` and the fixture README both state that a real capture carries
`session.id` on **every** metric data point and every log record, so a fixture without it
modelled an export Claude Code does not produce and pinned nothing about a real one. The
tests now read them through a `fixtureSession` constant.

Files whose log records were backfilled: `accept_then_result.json`,
`accepts_no_results.json`, `log_record_shapes.json`, `no_timestamp_tool_call.json`,
`number_string_variants.json`, `out_of_order.ndjson`, `partial_no_tokens.json`,
`skill_name_present.json`, `tool_calls_only.json`, `tool_failure.json`,
`unknown_events.json`, `unnamed_reject.json`, `unreadable_tool_events.json`,
`untimed_api_request.json`.

Files whose metric data points were backfilled: `absent_temporality.json`,
`array_attribute_series.ndjson`, `as_double_rounding.json`,
`as_double_wins_over_as_int.json`, `cache_only.json`, `number_string_variants.json`,
`skill_name_present.json`, `temporality_enum_names.json`, `unreadable_temporality.json`,
`unrecognised_token_type.json`, `value_not_a_count.json`, `zero_token_count.json`.

**Left alone:** `malformed.json`, `top_level_array.json`, `empty.json`,
`type_mismatch.json`, `stray_close_then_batch.ndjson` and the truncated tail of
`truncated_final_line.ndjson` — all `error` results, nothing in them is reached, and the
first two cannot be parsed to edit. `gauge_not_sum.json`, `no_envelope.json`,
`bespoke_envelope.json`, `empty_envelope.json` have no data point or log record to carry
one.

Also `metricsBatch` in `otlp_test.go`, the inline export three format-layer tests build,
now carries `fixtureSession` for the same reason.

**Three fixtures added:** `two_sessions.ndjson`, `foreign_scope.json`,
`kvlist_attribute_series.ndjson`. All three registered in `captureCases`, which
`TestEveryFixtureIsAContractCase` enforces as the coverage denominator.

---

## 6. Non-vacuity

Every fix went RED first, and each half was then neutered independently to show it is
load-bearing and that neither half covers for the other — which is the audit's own claim
that "patching one signal leaves the other".

| Neutered | RED | Still GREEN |
|---|---|---|
| `ownsAttrs` → `return true` (session half) | the three session tests, with the 49200/9440 and 99999000 numbers back | the foreign-scope test |
| `ownsScope` → `return true` **and** `eventName` re-qualifying (scope half) | the foreign-scope test, with 8000 tokens and four tool calls back | the session tests |
| `identity` → `"k"/"a" + compactJSON` (G9-02) | the four map-identity tests and the 300-vs-150 total | — |
| the `accepts` counter suppressed (G2-03 pin) | the unknown-path half | — |
| the counter reason attached to a `present` result (G2-03 pin) | the present-path half | — |

One case my own guard test caught and reddened *my* fix: I had written
`{"kvlistValue":{"values":null}}` vs `{"values":[]}` as "an empty map against no map at
all" and expected them distinct. They are not — OTLP mandates the proto3 JSON mapping,
which reads a `null` as the field's default, so those are two encodings of one empty map,
and splitting a series over the difference is the same formatting defect as the member
order. The test was wrong, not the fix; it is now restated over whole `AnyValue`s, with the
real distinction (an empty map against an `AnyValue` of no kind, and a map against the
array of its own members) pinned separately, and `null`-vs-`[]` pinned as one series in
both the map and array forms.

---

## 7. Final gate — real counts

```
$ cd profiler && gofmt -l .                  (no output)
$ go build ./...                             OK
$ go vet ./...                               OK
$ go test -race -count=1 ./...
ok  github.com/Okja-Engineering/skill-architect/profiler        1.282s
ok  github.com/Okja-Engineering/skill-architect/profiler/cmd    1.928s
```

77 top-level Go tests, 323 subtests, 0 failures.

All five shell suites, on **both** bashes:

| Suite | bash 3.2 | bash 5.3 |
|---|---|---|
| `tests/test_harness.sh` | 46 passed, 0 failed | 46 passed, 0 failed |
| `tests/test_skill.sh` | 27 passed, 0 failed | 27 passed, 0 failed |
| `tests/test_walk.sh` | 20 passed, 0 failed | 20 passed, 0 failed |
| `tests/test_f01.sh` | 575 passed, 0 failed | 575 passed, 0 failed |
| `tests/test_f02.sh` | 243 passed, 0 failed | 243 passed, 0 failed |
| **total** | **911, 0 failed** | **911, 0 failed** |

`test_harness.sh`'s meta-check — the PR #6 repair that refuses an assertion which cannot
fail — passes unmodified. No assertion was weakened; the suites were not edited at all.

---

## 8. `AdapterVersion` — confirmed still needed

`profiler/types.go:234` is untouched at `"0.4.2"`, and no version surface was edited
(`git diff 71cc866..HEAD -- CHANGELOG.md RELEASE_NOTES.md .out-of-scope.md` is empty;
`types.go` gained only the 12-line `SessionIDRequiredError`).

**The bump is still required and this lane is the reason.** Both halves change what a
profile contains for the same input, provably:

- C2 — `two_sessions.ndjson` @ session A: `{"input":49200,"output":9440}` → `{"input":48000,"output":9100}`, and `total_ms` `99999000` → `12000`.
- G9-02 — `kvlist_attribute_series.ndjson`: `{"input":300}` → `{"input":150}`.

A stored 0.4.2 profile beside a fresh 0.4.3 one is otherwise an unexplained regression in
the numbers, which is exactly what PR #5 exists to prevent one release earlier. One bump
covers both. It belongs in the release commit, per the mandate, and it is in the 0.4.3
definition of done.

---

## 9. Files

- `profiler/provenance.go` — **new.** The provenance test, the projection, the exclusion
  counts, the not-read clause. 143 non-comment lines.
- `profiler/claude_code.go` — `resolve(prov)`, `provenanceFor`, `probeProvenance`,
  `Capture` refuses an empty id, `eventName` no longer re-qualifies a body, extractors
  take a `scopedExport`, unknown reasons carry the clause, G2-03 comment.
- `profiler/otlp.go` — `otlpScopeLogs.Scope` decoded; `arrayIdentity` / `kvlistIdentity`
  replace `compactJSON` for the two composite kinds.
- `profiler/types.go` — `SessionIDRequiredError`.
- `profiler/provenance_test.go` — **new.** Session scoping, the absent session, the empty
  id, foreign scope, and the absent-scope guard.
- `profiler/otlp_test.go` — the four series-identity tests; `metricsBatch` carries a session.
- `profiler/profiler_test.go` — `fixtureSession`; `captureCase.session`; the two new
  contract cases; the G2-03 behaviour pin.
- `README.md`, `docs/profiler-spec.md`, `profiler/testdata/otlp/README.md` — the
  provenance contract, the `--session` decision, the probe-is-unscoped note, the composite
  identity rule, the three new fixtures.

---

# PR #7 gate round — dispositions, and corrections to this record

Appended after the `pr7-gate.md` verdict (**NOT CLEAN**: F1 blocking, F2 close behind,
F3–F9 in the same pass, D1 decided, F10–F12 recorded). Nine commits on top of `3e00395`,
one per finding, each pushed to `fix/0.4.3-profiler-provenance` as it landed.

**Four statements in sections 1–9 above were false. They are corrected here rather than
edited out**, because the cluster this lane belongs to is adapter honesty and a record that
quietly repairs itself is the same defect one level up. Section 3's "pinned in
`provenance_test.go`" (F2), section 2's "35 of the scope entries" (F6), section 3's the CLI
requirement now "derives" from the adapter's contract (F7), and the fixture readme's "every
readable data point" (F3) were each wrong. The substance behind three of the four holds; one
was a test that did not exist.

## Dispositions

| Finding | Disposition | Commit |
|---|---|---|
| **F1** — no fixture holds two sessions inside one metric's `dataPoints`; both mutations of the rule at `provenance.go:200` leave the suite green | **REAL, fixed** — new fixture `two_sessions_one_metric.json` plus two assertions, one per half of the rule | `93b66ac` |
| **F2** — the probe/capture divergence is documented with no assertion behind it; the record claimed a pin that does not exist | **REAL, fixed** — three pins (probe-vs-capture over one export, probe still refuses a foreign scope, `anySession` is probe's alone); claim corrected below | `7ecc628` |
| **F3** — the fixture readme asserted every readable data point carries `session.id`; `gauge_not_sum.json` is readable and its gauge point does not | **REAL, fixed** — the claim is now about `sum` data points, and the projection states why only a sum's points are filtered and counted | `37c489d` |
| **F4** — `{"kvlistValue":null}` identified as an empty map; under proto3 JSON a null leaves the field unset, so it has no kind | **REAL, fixed** — one predicate (`wasSet`) for all five raw-JSON kinds, with the seven-kind table pinning null == `{}` and null ≠ the kind's zero | `918f8dc` |
| **F5** — the comment claimed a duplicate key is left as two members rather than collapsed, while the sort merges reordered duplicates | **REAL, fixed** — behaviour kept and pinned, comment rewritten to say what it does and why | `eb2ebb5` |
| **F6** — "35 unnamed scope entries" is wrong | **REAL, corrected** — the measured figure is **36 across 25 files**; see the pushback below | this record + `worklist.md` |
| **F7** — the claim that the CLI requirement now "derives" from the adapter contract is false | **REAL, corrected** — the claim was wrong, the code is right; the CLI check stays and now names `SessionIDRequiredError`, and an empty `--session` is pinned at the CLI surface | `c81e19b` |
| **F8** — readme and spec state the scope half as an allowlist in the bolded sentence | **REAL, fixed** — both now say "was not recorded by another product's instrumentation scope" first | `c3c82bc` |
| **F9** — the spec's log-record bullet still describes an `event.name` fallback that no longer exists | **REAL, fixed** — the bullet describes the identity the code has | `88e7a9c` |
| **F11** — a signal left `present` with a silently reduced value and no reason when *some* records lack `session.id` | **DEFERRED, documented** — added to the schema-v1 gap's list beside the skipped point and the refused series. Not fixed here: the fix is a field on a `present` result, which is 0.5.0 | `d787977` |
| **D1** — the worklist's G2-02 invariant wording | **Restated per the ruling** as the exclusion rule, with the measurement beside it. Only worklist edit made | `worklist.md` |
| **F10, F12** | Recorded judgments, not requested changes. Untouched | — |

## F1 — the proof that was missing, and the proof it is now there

`two_sessions.ndjson` gives each session its own NDJSON batch, so scoping it drops a whole
batch's metric and never filters **within** a `dataPoints` array — which is the layout a
collector writes when it flushes two sessions together, and the one the headline
48000/9100 is read from. The log side was already covered by a `logRecords` array spanning
both sessions.

`two_sessions_one_metric.json` is one metric whose four points interleave both sessions:
5000/700 for `2222…`, 90000/4300 for `3333…`, totalling 95000/5000, which is neither
session's. Two assertions over it, one per half of the rule.

Measured, in the pre-change tree and then at head:

| Mutation of `provenance.go:200` | Before (`3e00395`) | After |
|---|---|---|
| drop the metric if *any* point is foreign (`len(points) < len(m.Sum.DataPoints)`) | **`go test ./...` fully GREEN** | **RED** — `TestProvenance_OneMetricCarryingTwoSessionsContributesOnlyTheProfiledSession/session_A` and `/session_B`, both `tokens state = "unknown"`, plus the contract case |
| never drop an emptied metric (rule deleted) | **`go test ./...` fully GREEN** | **RED** — `TestProvenance_AMetricEmptiedByTheFilterIsNotReportedAsCarryingNoDataPoints`: reason came back "no readable claude_code.token.usage metric in OTel export: it carried no sum data points; …" for an export carrying four |

Each half is caught by its own assertion and by nothing else. Head behaviour was already
correct on this shape, so nothing about the numbers changed — this was a test gap, and the
gate was right that the non-vacuity table in section 6 never reached the surface.

## F2 — the claim, corrected, and the pins that now exist

Section 3 above says the probe/capture divergence is "Documented in `README.md` and
`docs/profiler-spec.md`, and pinned in `provenance_test.go`". **The last clause was false**:
that file contained zero `Probe()` calls and `anySession` had no test anywhere. The
divergence itself is real and correctly described; what did not exist was the assertion.

Three pins now stand, each independently load-bearing:

| Neutered | RED |
|---|---|
| `probeProvenance` no longer sets `anySession` | `ProbeAnswersForTheExportAndACaptureForItsSession`, `OnlyProbeDropsTheSessionTest` (and 40 pre-existing subtests) |
| `provenanceFor` also sets `anySession` | `OnlyProbeDropsTheSessionTest` — plus every session test, which is the defect this lane exists for |
| `ownsScope` returns true for every scope | `ProbeStillRefusesAForeignScope` (new — this was the unpinned half) and `ForeignScopeAndUnqualifiedEventsAreNotAttributed` |

## F6 — pushback: the figure is 36 across 25 files, not 38 across 26

The record's 35 is wrong and the gate's 38 across 26 files does not reproduce here under
any denominator I can construct. Measured at head, over the fixture directory, counting
`scopeMetrics` and `scopeLogs` entries and calling an entry unnamed when it carries no
`scope` object or a `scope` with no `name`:

```
unnamed scope entries: 36, across 25 files
named scope entries:   56
total scope entries:   92        (59 fixtures)
```

Same 36 across the same 25 files at base `71cc866` (where the totals are 46 named, 82
entries over 55 fixtures) — the fixtures this PR adds all carry named scopes, so the
figure did not move. Cross-checked by a second method: `grep -o '"scope"'` counts 55 scope
objects at `3e00395`, and every one of them carries a name, which is the same
named-entry count the parse gives.

The denominator matters, so it is stated: that count is over content the reader can
**parse**. Four scope entries sit in content that cannot be — `malformed.json`,
`top_level_array.json`, `type_mismatch.json` and the truncated final line of
`truncated_final_line.ndjson` each carry one, and nothing in them is ever reached — so the
textual count is 40 across 28 files. Neither is 38/26.

**The substance is stronger than any of these figures, which is the gate's own point.**
Re-measured independently: mutating the exclusion to a strict allowlist (`ownsScope`
returning false for an unnamed scope) fails **53 tests and subtests across 16 top-level
tests** — 16 top-level FAIL lines, 37 subtest FAIL lines. That reproduces the gate's
number exactly.

## F7 — the CLI requirement does not "derive", and stays where it is

Section 3's "the CLI's requirement now *derives* from the adapter's contract" is false.
`cmd/main.go` is not in the diff and still states the requirement itself, in one check with
`--snapshot` and `--skill-dir`, and that check runs first.

It stays there deliberately: the flag-layer check runs before a harness is resolved (for an
unknown `--harness` there is no adapter to ask) and it names all three missing flags in one
message. Deriving would trade a usage message for a "capture error" and reorder the two
failures for no behavioural gain — the contracts agree, and `claude_code.go`'s own comment
already said "the CLI that restates the requirement", which was the honest wording all
along.

What was missing is that a reader of either site could not see the other. The check now
names `profiler.SessionIDRequiredError` and says which of the two is the contract, in the
same idiom `captureFlagError` already uses for `--export-file`. And the invariant is pinned
at the CLI surface rather than left to a comment: an empty `--session` is a usage error and
never a profile, whichever layer refuses it. Removing **both** refusals reddens it — the
CLI prints a profile of the whole export and exits 0.

## F3 — what the readme claimed, and what is true

The fixture readme's bolded sentence asserted that *every readable data point and log
record* carries `session.id`. `gauge_not_sum.json` parses, carries an envelope and reports
`unknown` rather than `error` — it is readable — and its gauge point carries none, so the
sentence was false of the directory it introduces. Section 5 above repeats the error in its
"Left alone" list, which said that fixture has "no data point … to carry one"; it has one,
and it is a gauge.

The true rule: only a **sum's** data points are session-filtered, because a sum is the only
shape a signal reads a value out of. A gauge or histogram metric passes the projection
whole and the extractor refuses it for want of sum data points, whoever's it is. So a gauge
point is never tested against the session and never counted in the "not read" clause —
which is what keeps that clause a count of points a signal could have read. No behaviour
changed; the scope test is the level above and still covers every metric kind.

## Re-verification at the new head

**The 55 pre-existing fixtures are still result-identical.** Re-run at `d787977`, three
comparisons by execution, masking only `profiled_at`, `capability.probed_at` and the
caller-supplied `session_id`, and comparing stdout, stderr **and** exit code:

```
pre-existing fixtures compared: 55
(1) base-bin/base-fx == head-bin/head-fx : identical=55 changed=0
(2) base-bin/base-fx == base-bin/head-fx : identical=55 changed=0
(3) probe base/base  == probe head/head  : identical=55 changed=0
```

Base binary built from `71cc866` via `git archive` into the scratchpad; head binary from
this worktree. The F4 repair cannot have moved any of them for a second reason as well: no
fixture in the directory contains a JSON `null` anywhere.

**Final gate — real counts, at `d787977`:**

```
$ cd profiler && gofmt -l .                  (no output)
$ go build ./...                             OK
$ go vet ./...                               OK
$ go test -race -count=1 ./...
ok  github.com/Okja-Engineering/skill-architect/profiler        1.313s
ok  github.com/Okja-Engineering/skill-architect/profiler/cmd    1.939s
```

341 Go tests run — **84 top-level, 257 subtests, 0 failures**, no deeper nesting. (Section
7's "77 top-level, 323 subtests" counted differently; these are the numbers `-v` prints at
this head: `=== RUN` 341, `--- PASS` 84 at column 0 and 257 indented, `FAIL` 0.)

Shell suites — **there are five, not six**, and `tests/` holds no sixth on this branch
(`find . -name 'test_*.sh'` gives `test_f01.sh`, `test_f02.sh`, `test_harness.sh`,
`test_skill.sh`, `test_walk.sh`). All five, on both bashes, unmodified by this round:

| Suite | bash 3.2.57 | bash 5.3.15 |
|---|---|---|
| `tests/test_harness.sh` | 46 passed, 0 failed | 46 passed, 0 failed |
| `tests/test_skill.sh` | 27 passed, 0 failed | 27 passed, 0 failed |
| `tests/test_walk.sh` | 20 passed, 0 failed | 20 passed, 0 failed |
| `tests/test_f01.sh` | 575 passed, 0 failed | 575 passed, 0 failed |
| `tests/test_f02.sh` | 243 passed, 0 failed | 243 passed, 0 failed |
| **total** | **911, 0 failed** | **911, 0 failed** |

**No version surface touched.** `git diff 71cc866..HEAD -- CHANGELOG.md RELEASE_NOTES.md
.out-of-scope.md` is empty and `AdapterVersion` is still `"0.4.2"` at `types.go:246`. The
bump still belongs to the release commit.

**Fixture denominator moved by one**: 59 fixtures, all 59 named in `captureCases`, which
`TestEveryFixtureIsAContractCase` enforces by globbing the directory.

Fifteen commits on the branch, all `imagineux <imagineux@gmail.com>` as author and
committer, zero attribution trailers (`git log 71cc866..HEAD --format=%B | grep -ci
"co-authored-by\|generated with\|noreply@anthropic"` → 0).
