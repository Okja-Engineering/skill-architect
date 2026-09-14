# PR #5 gate — every factual claim the new release prose makes

Gated `release/0.4.2-version` @ `13b3141` against base `5c847e1`.

**Verdict: DO NOT MERGE AS IS. Six REAL false claims, four of them in prose that ships
as the permanent record of this release.**

> Persisted by the chief of staff: the hunter's session had Write disabled.
> **Verified by CoS** notes are independent confirmation against the tree.

Nothing found is a code defect. The code does what every claim's headline says. What is
wrong is the detail inside the claims.

## Coverage

12/12 diff files walked. **160 checkable assertions enumerated and walked, each with a
verdict.** 91/91 backticked tokens enumerated; 27 path-like ones resolved at `13b3141`.
All 55 OTLP fixtures run through head and v0.4.1 binaries side by side, plus 23
purpose-built fixtures constructed to induce the exact cases the prose narrates. 4/4
shell suites under two locales at head and at `2ed34b8`. Go suite under `-race` at both.
**968 script invocations** swept for out-of-range exits. Two full sweeps.

## Shared root

Every false item is a **sub-clause inside a claim whose headline is true**, and every one
is a detail that could only be settled by measuring rather than recalling. The prose was
written from the development branch's narrative and from the author's model of the code.
Where those agree with the tree it is right, including every hard number re-measured.
Where a sub-clause was generalised, rounded, or carried over from an intermediate branch
state, it is wrong.

**Correcting six sentences without re-measuring the rest will leave the rest.**

---

## F1 — BLOCKING — `profiler/otlp.go:846-849`, repeated at `CHANGELOG.md:41`

**"A negative value is clamped to zero rather than refused"** is false on the delta
branch of `add`.

The two cumulative accumulators do clamp, each against a zero start. The delta branch
does not:

```go
s.increments = addSaturating(s.increments, p.value)
```

and `addSaturating` has an explicit `b < 0` arm saturating to `math.MinInt64`. A negative
delta value is **added**, and `total()` returns it.

```
cumulative unplaced, value -500  -> total = 0
cumulative run,      value -500  -> total = 0
delta,               value -500  -> total = -500
delta, 100 then -500             -> total = -400
```

The comment's audience is explicitly "a second OTLP-speaking adapter" — the one reader
for whom the claim matters, and for whom it is false **in the unsafe direction**. A
reader who trusts "clamped to zero" concludes the type fails safe. On a delta series it
emits a negative total.

Verified true in the same comment: that `add` validates nothing, and that
`claude_code.go` drops a non-count point before `add` is reached.

**Invariant**: the comment must describe what `add` does on **every** branch it has, or
scope its claim to the branches it describes.

> **Verified by CoS.** `otlp.go` delta branch calls `addSaturating` with no clamp;
> `addSaturating` has an explicit `case b < 0`. The two cumulative branches clamp.
> Confirmed REAL.

## F2 — BLOCKING — `CHANGELOG.md:42`

**"documents the eight fixtures this release adds. The fixture set goes from 39 to 55."**
Self-contradicting; 39 to 55 is **sixteen**, all added by `bd32b26`, none removed.
`RELEASE_NOTES.md:19` states it correctly.

## F3 — BLOCKING — `CHANGELOG.md:39` and `docs/profiler-spec.md:223`

**"Exactly one deviation"** from the proto3 JSON mapping. There are at least two.

The documented one is real, proven both ways. The undocumented second: the adapter
**accepts an enum written as a quoted decimal**, which the mapping rejects. `otlp.go:809-813`
says so outright, and `cumulative.ndjson` carries a batch declaring `"aggregationTemporality": "2"`.
Against real protojson:

```
{"aggregationTemporality":2}              err=<nil>
{"aggregationTemporality":"2"}            err=invalid value for enum field
{"aggregationTemporality":"T_CUMULATIVE"} err=<nil>
```

The note's stated purpose is that it "not be mistaken for a conformance claim". A
non-exhaustive exhaustive count is the exact failure it was written to close.

## F4 — BLOCKING — `RELEASE_NOTES.md:13`

**"All three now exit 3, name the missing tool on stderr, and carry a `passed: false`
payload."** `check-frontmatter.sh` has no `--json` mode; its own header says so, and it
calls `require_tool` with the emit-json flag false. Measured with the tool masked: exit 3,
**stdout empty**. `CHANGELOG.md:28` gets it right, so the two files contradict each other.

## F5 — BLOCKING — `CHANGELOG.md:43`

**"`README.md` gains a `jq` row"** — it did not. The row existed at `2ed34b8:README.md:247`;
this release rewrote it. The bullet's other three sub-claims are true.

## F6 — BLOCKING — `RELEASE_NOTES.md:10`

**"a run reporting 900, then 500, then 1000 came back as 500, and erasing its start time
raised it to 1000."** False on its plain reading. With all three points timed, v0.4.1
gives **1000** and erasing the start changes nothing, so the "adding information lowered
the number" point collapses. It holds only for the shape the CHANGELOG states, an
**untimed** 900, a 500 at t2000 and an **untimed** 1000, which does reproduce: v0.4.1
**500**, start erased **1000**. The summarisation dropped the one detail that makes the
number true.

---

## Non-blocking

- **F7** `CHANGELOG.md:56` — "documented in both script headers and in the README". The
  headers enumerate all three no-payload paths and tests pin them, but the README
  documents only the guard-load path.
- **F8** `CHANGELOG.md:40` — "and the Go test reasons". No Go test reason states anything
  about v0.4.1. The other three named files do, correctly.
- **F9 SUSPECTED** `CHANGELOG.md:32` — "left `audit-report.sh` exiting 1" is not
  reproducible against either pre-fix state. No guard shape gives 2 from every script and
  1 from audit-report; they are mutually exclusive. The first half does reproduce.
- **F10/F11/F12 informational** — three entries describe states that existed only on the
  unmerged branch without saying so. `CHANGELOG.md:31` handles the same situation
  honestly. Consistency is the ask, not correction.

---

## Ledger — 160 assertions walked, all OK by execution except the above

- **Header 7/7**: no schema keys changed, every `json:` tag byte-identical to `2ed34b8`;
  `adapter_version` reads `"0.4.2"` in a captured profile.
- **Merge semantics 25/26**: series identity, `schemaUrl` excluded, absent resource
  equals empty-attribute resource, zero start equals absent start, runs keyed by start
  not file order, `isMonotonic` and `flags` unread, the unplaceable rule.
- **Protobuf facts 5/6**: `fixed64 start_time_unix_nano = 2;` confirmed upstream;
  protojson default-value behavior exactly as claimed; float64 ulp at 1.7893e18 is 256ns.
- **Every narrated number reproduced**: 200/300, 300/300, 20/120, 700/900, 520, 350,
  refusal counted per series, survivors still counted, 500/900, 200/100, 1000/200.
- **All four audit-script "before" strings reproduced verbatim** at `bd32b26`.
- **All "after" claims verified**, including payload derivation and the 127-to-3 change.
- **jq-crash claims exact**: five induced shapes, stdout 0 bytes in all five.
- **Encoder**: the pre-fix guard emits sixteen hex digits under a UTF-8 locale exactly as
  claimed; head swept all 256 byte values under both locales, escape width always 4.
- **Exit sweep**: 484 invocations x 2 locales, **0 outside `{0,1,2,3}`**.
- **Every count re-measured, none carried**: head 66 functions and 230 subtests;
  `2ed34b8` 58 and 179; shell 850 and 134; fixtures 39 to 55.
- **The "not changed" claims hold.** Both changelog and release-notes diffs are **pure
  insertions** with zero context-line changes. The 0.4.1 entries are byte-for-byte intact.
- **All three corrections are true**, each quoted verbatim from the entry it supersedes,
  with v0.4.1 numbers re-measured against the v0.4.1 binary.
- **Versions**: all five manifests at 0.4.2, `profiler version` prints `profiler 0.4.2`,
  no stale `0.4.1` version reference anywhere.

## Adjacent, out of diff

`profiler/testdata/otlp/README.md:56` describes a counterfactual as "collapse them into
one run holding 200"; measured, v0.4.1 reports **60**. Defensible as a counterfactual
about the current rule, but it sits one row below a genuine past-tense v0.4.1 statement.
Worth disambiguating if that file is reopened.
