# PR #7 gate — C2 profiler provenance + G9-02

Gated `fix/0.4.3-profiler-provenance` @ `3e00395` against base `71cc866`.

**Verdict: NOT CLEAN.** One blocker (F1), one close behind (F2), ten non-blocking.

**Every headline claim re-run reproduces exactly.** What blocks is a third unpinned surface
the record's non-vacuity table never reaches, and a claim of a test that does not exist.

> Persisted by the chief of staff: the hunter's session had Write disabled.
> **Verified by CoS** notes are independent confirmation.

## Coverage

37/37 diff files and every hunk · 24/24 edited pre-existing fixtures mechanically diffed
against base · **55/55 pre-existing fixtures re-compared base-vs-head three ways by
execution** · 3/3 new fixtures recomputed · 58/58 fixtures scanned for records lacking
`session.id` and for unnamed scope entries · **5/5 claimed neutering mutations re-run plus 5
more of the gate's own** · 14 constructed scope names probed · 3/3 C2 entries plus G9-02
checked against invariant and stated proof · 2/2 RED-first repro commits re-run.

## Claims re-run — all reproduce

- **The fixture edits are clean.** Stripping every `session.id` entry from both sides makes
  **24/24 byte-identical to base**. All 53 added entries carry the single id the fixtures were
  already a capture of. The edit adds nothing else.
- **`identical=55 changed=0` reproduced three ways**: base-binary over base-fixtures versus
  head over head (stdout, stderr **and** exit code); the base binary over old versus new
  fixtures; probe base versus head.
- **All five neutering mutations reproduce verbatim, including the numbers.** The gate also
  ran the event-name re-qualification alone and found it independently RED.
- **The `{"values":null}` self-correction is right and grounded** in the proto3 JSON mapping,
  where `null` or `[]` for a repeated field behaves as absent. OTLP deviates from that only
  for trace and span ids.
- **RED-first confirmed** on both repro commits.
- **Deviation (a), the scope exclusion, is sound.** Mutating it to a strict allowlist fails
  **53 tests and subtests across 16 top-level tests**. Far stronger than the record claimed.

---

## F1 — BLOCKER — `profiler/provenance.go:185-209`, rule at `:200`

**No fixture in the 58 holds two different `session.id`s inside one metric's `dataPoints`
array** — the shape a collector produces when it flushes two sessions in one batch, which is
the primary real-world multi-session **metric** layout.

`two_sessions.ndjson` puts each session's metrics in its own NDJSON batch, so session A's
headline 48000/9100 is proven only by dropping a whole batch's metric, never by filtering
**within** a `dataPoints` array. The log side **is** covered, because one `logRecords` array
spans both sessions. So the gap is specific to `tokens` — the P1's own headline number.

**Two mutations leave `go test ./...` fully GREEN:**

1. drop the metric if *any* point is foreign — on a real mixed-session metric this turns
   session A's tokens into `unknown`.
2. never drop an emptied metric — makes the reason say "it carried no sum data points" for an
   export carrying several, the exact false statement the code's own comment says the rule
   exists to prevent.

Head behaviour on a hand-built mixed-metric fixture is **correct**, so this is a test gap
rather than a live defect. But the record presents its non-vacuity proof as complete and it
does not reach this surface.

**Invariant**: a single `dataPoints` array holding several sessions contributes exactly the
asserted session's points; a metric that had points and kept none must not produce the
"carried no sum data points" reason.

> **Verified by CoS.** The fixture's three batches are: metrics for one session, metrics for
> the other, then logs spanning both. The metric side is never multi-session within a batch.
> Confirmed REAL.

## F2 — `provenance_test.go` — the claimed pin does not exist

The record says the probe/capture divergence is "pinned in `provenance_test.go`". That file
contains **zero `Probe()` calls**, and `anySession` has no test anywhere. The behaviour is now
documented out loud in the readme and the spec with no assertion behind it.

**On the divergence itself: honest and documented, not a defect.** A profile's capability
block comes from the scoped resolve, verified `capability.tokens = none` for an absent
session. What blocks is the missing pin plus the false claim.

---

## Non-blocking, cheap, and belong in the same pass

Several are **false statements in the lane record and the fixture readme**, in a cluster whose
subject is adapter honesty.

- **F3** the projection sees only `Sum`, so a gauge point is never session-filtered. No live
  consequence, but the fixture readme now asserts every readable data point carries
  `session.id` while `gauge_not_sum.json` is a readable export whose gauge point does not.
- **F4** `{"kvlistValue":null}` identifies as an empty map when under proto3 JSON it has no
  kind at all. Direction is an undercount. Contradicts the adjacent test that pins exactly
  that distinction.
- **F5** a comment claims a duplicate key is left as two members rather than collapsed, but
  the identity sorts, so reordered duplicates merge — the guess the comment disclaims.
  Newly introduced for kvlists.
- **F6** the record's "35 unnamed scope entries" is **38**, across 26 files. Substance is
  verified far more strongly than claimed; only the figure is wrong.
- **F7** the claim that the CLI requirement now "derives" from the adapter contract is false.
  `cmd/main.go` is not in the diff and still restates it independently. The contracts agree
  and the CLI check is earlier, so no behavioural hole, but two copies is the drift the record
  says it removed.
- **F8** the readme and spec both state the scope half as an **allowlist** in the bolded
  sentence, then correct it one or two sentences later. A record under no scope is read. The
  bolded sentence is the one a reader quotes.
- **F9** the spec's log-record bullet still describes an `event.name` **fallback** that no
  longer exists; a present-but-unqualified body is now refused outright. The spec says two
  things.

## Recorded judgments, not requested changes

- **F10** deviation (b)'s boundary, measured over 14 scope names. **Any vendor's scope
  carrying `claude_code` as a dot-component is read**, including a third party's, while a
  plausible Anthropic sibling is refused. Not exploitable without the run's session UUID. The
  gate judges the deviation acceptable and correctly documented; the boundary is recorded so
  the decision rests on measurement.
- **F11** the answer to "what happens to a record legitimately lacking `session.id`".
  **All** records lacking it gives every signal `unknown` with an honest reason — good.
  **Some** records lacking it leaves the signal **`present` with a silently reduced value and
  no reason at all**. This is not the original defect in a new form, and it is the schema-v1
  gap already tracked for 0.5.0, but it is a **new member of that deferred class** and is not
  listed beside the skipped-point and refused-series cases.
- **F12** exit 0 for a profile entirely `unknown` because the named session is absent from a
  readable export. Pre-existing rule, new case, undiscussed.

## D1 — drift needing the decision-maker's explicit acceptance

The worklist states G2-02's invariant as "**a record is Claude Code's only if its scope says
so**". The code implements "**unless its scope says otherwise**". The fixer flagged, argued and
documented the deviation, and the gate confirmed the mechanism independently: 38 unnamed scope
entries, and a strict allowlist fails 53 tests.

**The gate judges the deviation correct on the merits.** But the approved invariant's wording
no longer matches the code, and that row is what a later reader checks against.

> **Chief of staff ruling: deviation ACCEPTED.** An allowlist would refuse records from any
> pipeline that drops the instrumentation scope, trading a wrong number for no number. The
> worklist row is to be restated as the exclusion rule so the release gate does not read this
> as an unnoticed miss.

## What is clean

One projection site, executed once before any extractor; all three extractors take a
`scopedExport`; the batch rebuild loses no envelope field and preserves pointer nil-ness,
which is why the two envelope fixtures are unchanged. `anySession` is set at exactly one place,
reachable only from `Probe`, and an empty provenance fails **closed** so bypassing the guard
yields no wildcard. The fixture-denominator test is real, globbing the directory: 58 fixtures,
60 contract subtests. **No assertion in the diff was weakened** — every test edit is a session
constant rename. Six commits, single identity, zero attribution trailers.

## Blocking set

**F1** (one fixture plus two assertions closes both halves) and **F2** (a pin for the
documented divergence, plus correct the record). **F3 through F9** are cheap and belong in the
same pass. **D1** is decided above. **F10 through F12** are recorded, not requested.
