# PR #5 confirming pass — `release/0.4.2-version` @ `2488ff0`

**Verdict: NOT CLEAN. One REAL finding blocks.**

The six original findings are all correctly fixed, both contested corrections are right and
independently reproduced, and the parser comment is accurate on all three branches. The
single blocker is a **new** factual claim the fixer introduced in `57e1e86` while fixing F3.

> Persisted by the chief of staff: the hunter's session had Write disabled.

## Coverage

13/13 diff files walked. 52 added `CHANGELOG.md` lines, 19 added `RELEASE_NOTES.md` lines,
8 changed `docs/profiler-spec.md` lines, 4 code/test hunks — each walked with a verdict.
**13/13 sentences the fixer changed** verified by measurement. 3/3 branches of `add` driven
directly. **55/55 OTLP fixtures** through v0.4.1 and head side by side. **Four independently
built binaries** across 12 purpose-built fixtures. **840-invocation exit sweep**
(15 targets x 7 script/modes x 4 tool masks x 2 locales). Go suite under `-race` at head and
at `2ed34b8`; 4 shell suites x 2 locales at both. Guard-load reconstructed under 3 shapes x
2 bash versions. Reference `protojson` v1.36.12 / OTLP proto v1.11.0 run directly. Two full
sweeps; the second added nothing.

---

## BLOCKING — F13, introduced by the fixer

**`CHANGELOG.md:39` and `docs/profiler-spec.md:227` — REAL — "Collectors emit the quoted
form" is asserted as fact, unsupported anywhere, and overstates the code comment it
documents.**

The code being documented, `profiler/otlp.go:811`, hedges: "a collector **may** quote the
digit." The prose upgraded **may** to **do**.

Three measurements:

1. **The repo contains no evidence.** `cumulative.ndjson` carries `"aggregationTemporality":"2"`,
   but it is a repo-authored fixture written to pin the acceptance, and the spec passage says
   so itself. No observation is recorded anywhere. Contrast `CHANGELOG.md:17`, careful to say
   delta was "observed live", and the adjacent clause in this same bullet, careful to say the
   exponent form is "a form `protojson` never emits and nothing observed produces".
2. **OTLP says the opposite is required.** OTLP 1.11.0, JSON Protobuf Encoding: enum field
   values MUST be encoded as integers, and enum name strings MUST NOT be used. A collector
   emitting `"2"` is non-conformant twice: a string where an integer is mandated, and not even
   a valid enum name.
3. **Measured against reference `protojson` v1.36.12** on OTLP `Sum`:
   `{"aggregationTemporality":"2"}` gives `invalid value for enum field aggregationTemporality: "2"`.

Both sentences arrived in `57e1e86`; neither existed at `13b3141`.

This is the gate's own shared root reappearing: **a sub-clause inside a claim whose headline
is true, generalised rather than measured.** The headline — the adapter accepts three
spellings, wider than the mapping — is TRUE and fully reproduced. The justification clause is
not.

It cannot be proven false, because no one can enumerate every collector. **That is the point.**
It is stated as settled fact in the permanent release record with nothing behind it, in a note
whose own closing sentence sets the opposite standard.

**Invariant**: every factual assertion in this release's prose is either measured and
reproducible, or marked as unverified. Prose must not assert more strongly than the code it
documents.

**Advisory fix direction**: either cite a measured producer, or bring the prose down to the
code comment's hedge. The design rationale survives either way, because the load-bearing half
— refusing a temporality costs an entire series' merge — is independent of whether any
collector has been seen doing it.

---

## The two contested corrections — both CONFIRMED RIGHT

### (a) F6, the worked example

The gate rebuilt the candidate design from the prose's description **alone** and reproduced
the fixer's four-fixture table **exactly, all 30 cells**:

```
FIXTURE                      v041     HEAD  candA  candB
1_all_timed                  1000     1000   1000   1000
1_all_timed_NOSTART          1000     1000   1000   1000
2_changelog                   500     1000    500    500
2_changelog_NOSTART           500     1000   1000    500
3_two_point                   500      900    500    500
3_two_point_NOSTART           500      900    900    500
4_1000_last                  1000     1000   1000   1000
4_1000_last_NOSTART          1000     1000   1000   1000
5_1000_first                  500     1000    500    500
5_1000_first_NOSTART          500     1000   1000    500
```

**The first gate was wrong.** Its blessed shape gives v0.4.1 500 both ways, so "erasing the
start raised it" cannot be a v0.4.1 result.

Worth knowing, not a finding: `candB` is a second reading of "the obvious repair" that also
time-orders the unplaced bucket, under which the narrated erase-all case gives 500/500. The
passage's general conclusion survives anyway — the partial-erase case inverts under **both**
variants, 500 to 900. The argument is robust; only the narrated numbers are specific to the
candidate as literally described, which is what the prose describes and says it measured.

### (b) F9, the guard shapes

Reconstructed the pre-fix state and measured all six invocations in both modes:

- **Unparseable guard gives exit 2 from every script**, confirmed from three unrelated syntax
  errors.
- **Absent guard gives exit 1 from every script**, all modes.
- Mutually exclusive, exactly as the corrected sentence now says.
- At head, absent and unparseable both give **exit 3, stdout 0 bytes**, from all six.

---

## Parser comment: accurate on all three branches, no behaviour change

```
cumulative unplaced, -500        total=0    unplaced=0
cumulative run, -500             total=0    runs=map[]      (no run key created)
cumulative run, 100 then -500    total=100  runs=map[7:100]
delta, -500                      total=-500
delta, 100 then -500             total=-400
```

Every clause verified. The original defect, true of two branches and false of the third, is
closed. **The `otlp.go` diff is comment-only.**

## `docs/profiler-spec.md` was necessary to pull in

At `13b3141` the note read "One deviation" while the changelog described it as enumerating
two. Leaving it out would have shipped a changelog entry pointing at a document that
contradicts it. True apart from F13.

## Nothing the first gate passed went stale

All counts re-measured, not carried: head 66 functions and 230 subtests; v0.4.1 58 and 179;
shell 850 under both locales; v0.4.1 134; fixtures 39 to 55. Five manifests at 0.4.2,
`AdapterVersion` 0.4.2, captured profile reads 0.4.2, every `json:` tag byte-identical to
`2ed34b8`. Exit sweep: **840 invocations, zero outside `{0,1,2,3}`**.

Every narrated number reproduced. The encoder claim reproduces **under bash 3.2**, the
environment it was observed in, with the C1 controls collapsing identically as claimed. Under
bash 5.3 the pre-fix defect does not fire, which is why a naive re-run looks like a
non-repro; the prose narrates a past observation, so this is not a finding.

---

## Incident — a tool on the host was damaged and has been repaired

The gate built a PATH mirror of symlinks and wrote a stub over one, which wrote **through**
the symlink into the real install, replacing `skillscore` v2.0.2 with a 49-byte bash stub
returning a fake score of 80.

**Repaired by the chief of staff**: the pristine file was restored from the staged tarball and
verified — `skillscore --version` reports 2.0.2 and a real scoring run produces real output.

**Every measurement in this report was taken before the clobber**, so none is contaminated.
Two sub-clauses are carried on the first gate's evidence plus the repo's passing tests rather
than the gate's own re-run: the two-document composer case (pinned at `tests/test_f02.sh:493,541`)
and the pre-fix jq-crash exit codes. Both corroborated; neither independently reproduced here.
