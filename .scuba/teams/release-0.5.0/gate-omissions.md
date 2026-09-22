# Gate lens: OMISSIONS — PR #22 @ `9c8ba53`

> **Persisted by the chief of staff.** The hunter had no Write/Edit tool, said so
> explicitly, verified the absence by `ls`, and returned its report inline rather than
> falling back to a heredoc. This is that report.

**VERDICT: NOT CLEAN — 18 findings.**

## Coverage — two denominators, built independently, reconciled

**D1 — the code (59/59 diff files walked)** via `git diff v0.4.3..9c8ba53`, plus `ci.yml`
confirmed **unchanged**. Enumerated and walked: **9/9 subcommand paths**; **all 4 exit
statuses** across both binaries; **89 new/changed JSON keys** extracted mechanically from
`json:"…"` tags; every new value added to an existing vocabulary (`MetricState`,
`MetricSource`, `MetricName`); **0 deletions, 0 renames**; exported Go API diffed for
removals (`go doc -all` v0.4.3 vs head: **149 added, 0 removed**); 5/5 manifests; **8/8
DuckDB queries actually executed**. Both binaries built and driven against real inputs.

**D2 — the nine slice records (9/9)**, **51 explicitly bracketed flagged items**
(`[S10]`/`[decide]`/`[note]`/`[carry]`/`[open, not mine]`/`[gap]`) enumerated
mechanically, plus all 9 "Not done, and why" tables and S9 §7.

**Did they agree? No — and the disagreement is where the findings are.** D2 surfaced 4
items the notes drop (O4, O7, O11, O12's RN half) that D1 alone would not flag as *owed*.
D1 surfaced 6 the records never anticipated (O1, O2, O3, O5, O6, O8) because they are
consequences of **combining** slices. Union taken, every item verified against a running
binary. Second full sweep added O10, O13–O18 and nothing further.

**Structural honesty — all three CLEAN.**
- `CHANGELOG.md` 345→EOF **byte-identical** to `v0.4.3` 7→EOF; header 1–6 identical.
  `RELEASE_NOTES.md` 30→EOF byte-identical to `v0.4.3` 3→EOF. **No historical entry
  rewritten.**
- `## Unreleased` exists at `CHANGELOG.md:5` and is **correctly empty**.
- **Voice/candor not regressed** — 0.5.0 carries three dedicated limit sections plus
  "Known past changes, recorded late", *more* candid than 0.4.3's single section.

## Shared root

**The project's recurring defect has reproduced inside the release notes themselves: a
closed-looking enumeration that is short.** Every finding is an enumeration presented as
complete (*"Three profile keys…"*, *"except four destinations"*, *"all three drafts read
them"*, *"the three refusals"*, *"Four things were deliberately not shipped"*) that a
mechanical walk shows is a strict subset.

A second, independent root affects O1/O3: **no slice owned the release's cross-slice
consequences.** S3 bumped `AdapterVersion`, S5 built the refusal that reads it, S4
changed a state vocabulary — and the release commit reported each slice's own list rather
than deriving the upgrade consequence from the pair.

---

## O1 — HIGH · REAL · reproduced — every 0.4.x profile is refused by this release's headline command. Absent from both documents.

`profiler/types.go:373` moves `AdapterVersion` to `"0.5.0"`; `profiler/compare.go:187`
(`adapterVersionRefusal`) refuses any pair whose versions differ.

```
$ prof-0.5.0 compare --baseline old043.json --candidate new050.json ; echo $?
"refusal": "adapter version mismatch: baseline \"0.4.3\" vs candidate \"0.5.0\" …"
2
```

`CHANGELOG.md:115-118` states the bump; `:43-49` / `RELEASE_NOTES.md:7` state the
refusal. **Neither joins them.** A user's entire stored profile library cannot be
compared against anything captured with this release, and that appears nowhere. 0.4.3
made exactly this disclosure for its own bump (`RELEASE_NOTES.md:34`: *"A stored 0.4.2
profile beside a fresh 0.4.3 one is not a regression; it is the fix"*) — so this is a
measurable regression in candor against the section this release is modelled on.
`RELEASE_NOTES.md` 0.5.0 never states `AdapterVersion` moved at all.

**Invariant:** when a release bumps `AdapterVersion` *and* ships the command that
refuses on it, the notes must state what happens to profiles captured before the bump.

## O2 — HIGH · REAL · reproduced with S10's own procedure — "all three drafts read them" is false for `devin.go`.

`CHANGELOG.md:19-21` / `RELEASE_NOTES.md:15`: *"Measured … they do not compile.
`otelMetric`, `otelLog`, `toInt` and `parseTime` were removed by v0.4.1's OTLP rewrite
and **all three drafts read them**."*

| draft | `otelMetric` | `otelLog` | `toInt` | `parseTime` |
|---|---|---|---|---|
| `cursor.go` | 1 | 1 | 1 | 1 |
| `codex.go` | 1 | 1 | 5 | 1 |
| **`devin.go`** | **0** | **0** | **0** | **0** |

Reproducing S10's exact recipe from its own record (`s10-record.md:265-268`) —
`git archive 9c8ba53 profiler`, `git show 1e4e845:profiler/devin.go` copied in,
`go build ./...` — **exits 0. `devin.go` compiles cleanly against release head.** The
compile failure is `codex.go` and `cursor.go` only. The measurement was sound; the
sentence generalised it to a third file the compiler never named. This sits in the
paragraph justifying the release's largest deletion, in the release whose subject is
claims that assert more than the code delivers.

Everything else in that paragraph **verified true**: `cursor.go:60`/`:150` and
`devin.go:37`/`:114` correct line-for-line against `0a83615`; `codex.go` does not
over-advertise so "two of them" is right; `git ls-tree v0.4.3` confirms "none ever has";
README still says "Planned" ×3.

**Secondary:** those `file:line` citations point into `0a83615`/`1e4e845`, commits **not
reachable from the release** and never named in the notes. A reader of the shipped
changelog cannot open `cursor.go:60`.

## O3 — MEDIUM-HIGH · REAL · reproduced — `skill_activation` gained the `error` state, undisclosed.

`profiler/types.go:442-445` adds `ErrorActivationResult`; `claude_code.go:210` calls it
from `erroredSignals`.

```
0.4.3: {… skill_activation: unknown …}
0.5.0: {… skill_activation: error   …}
```

At v0.4.3 `{present, unknown}` was the complete and *documented* domain for that key —
v0.4.3's `types.go` said so explicitly (*"has no Error constructor: no export can fail
it"*). A stored-profile consumer that switched on those two values is now wrong.
`CHANGELOG.md:119-125` sets out the schema rule verbatim — *"v1 stays while a v1 reader
is merely ignorant of a new key, and v2 is required when a v1 reader would be wrong"* —
then omits the only change in the release touching an existing key's **value domain**.
**The rule's own test case is the case that went unmentioned.**

## O4 — MEDIUM · REAL — `compare` treats a differing `harness` and `skill_dir` as notes; the notes disclose one of three.

`profiler/compare.go:249-263` (`pairNotes`) emits a note, never a refusal, for
**`harness`**, **`snapshot_hash`** and **`skill_dir`**. Both documents disclose only
`snapshot_hash` (`CHANGELOG.md:47-49`, `RELEASE_NOTES.md:7`) and present the refusal
machinery as *"the reason it can be trusted."* A reader concludes the tool protects them
from subtracting two different meters. It does not — that is a note.

D2 makes it squarely owed: `s5-record.md:305` *"Cross-harness comparison is a note, and
that is a decision with a shelf life"*; `s6-record.md:466` *"the hand-assembled case is
still the second adapter's call."* Both slices left it open for downstream. It reaches
neither document.

## O5 — MEDIUM · REAL — "Three profile keys this release adds are written by nothing that ships" — the real number is eight, and the document contains both figures.

`CHANGELOG.md:268` / `RELEASE_NOTES.md:22` say three (`error_type`, `count`, `id`).
`CHANGELOG.md:121-122` enumerates the added keys as *"`error_type`, `count`, `id`, the
four attribution fields and `estimated_context_tokens`"* — eight, and **all eight** are
written by nothing. `category`, `detail`, `operation_name`, `confidence`
(`types.go:225-230`) are **undisclosed as unwritten**: `PresentAttributionResult` has
zero non-test call sites and no shipped code constructs an `AttributionData`. The same
bullet says *"`attribution` is `unknown` on every profile this release can produce"* —
the proof, one sentence from the count of three. A reader takes the other five as live.

## O6 — MEDIUM · REAL · reproduced — "Every key this release adds … is optional and absent when it was not read" is false in the capability report.

Holds for the profile body (`types.go:342`, pointer + `omitempty`). Does **not** hold for
`capability.capabilities`, where `claude_code.go:228` hard-sets
`MetricEstimatedContextTokens: SourceNone` unconditionally — 5 capability keys at 0.4.3,
6 at 0.5.0, always present, in `capture`'s stored output too. Neither document says the
capability report gained a signal. `capability.capabilities` is read by `compare` and
walked by the contract table, so this is the schema-relevant half.

## O7 — MEDIUM · REAL · reproduced — `hooks install` never registers `--strict`; the metadata-only mode is unreachable through the documented install path.

`CHANGELOG.md:64` / `RELEASE_NOTES.md:10` present `--strict` as a shipped privacy
control. `cmd/main.go:550` (`resolveHookCommand`) returns `self + " ingest || true"` — no
`--strict` — and there is **no environment variable** (`s8-record.md:538`: *"S7 dropped
`<SIBLING>_STRICT` … the flag still has to be baked into the registered command by
hand"*). Reproduced: registered command `'… ingest || true'`, `any --strict: False`,
21 events. A user on a shared machine who runs the documented `hooks install` captures
full prompts and tool I/O through the redaction defaults only. Reaching strict mode
requires hand-writing `--command`, which no document mentions. The README's only
`--strict` example (`:471`) is a hand-piped `ingest`, not how a registered hook is
invoked.

> **Corroborated independently** by the dogfood hooks lane, which reached the same
> conclusion from the live installer rather than from the source.

## O8 — MEDIUM · REAL · reproduced — `analyze` emits a census of payload key **values**; both documents say only "a census of payload keys", and give it no publish warning while `experiment` gets one.

`analyze.go:121` ships `payload_key_values map[string]map[string]int`, quoting the value
of any non-content, non-numeric string key up to 64 chars, **unbounded in cardinality**:

```json
"payload_key_values": {
  "conversation_id": {"conv-SECRET-42": 1},
  "cwd":             {"/secret/proj": 1},
  "custom_field":    {"my-private-value": 1}
}
```

`cwd` (every absolute working directory captured), conversation ids, and any Cursor field
not on the redaction list. `analyze.go:112-121`'s own comment says *"a summary is a
document a user pastes into an issue"* — the authors knew and bounded it. `README.md:~540`
and `docs/profiler-spec.md:683` **do** disclose it; the release notes do not, and the
asymmetry is sharp: `RELEASE_NOTES.md:24` gives the experiment result a dedicated *"Read
a result before you publish it"* while `:14` actively invites pasting a `doctor`/`analyze`
report.

## O9 — MEDIUM · REAL — `experiment` refuses a cross-harness design; a shipped refusal absent from both documents.

`experiment.go:316-319` in `designRefusal`: *"the two conditions must name the same
harness … two harnesses measure with different meters."* Deliberately ordered **before**
the registration check so it survives a second adapter (`s6-record.md:167-190`). Both
documents enumerate the design refusals explicitly (`CHANGELOG.md:51-53` /
`RELEASE_NOTES.md:8`) and this one is not in the list — the refusal that resolves the
open question S5 raised and S6 answered, invisible. Related: `validatePlan`
(`experiment.go:208-232`) refuses a no-run plan, a step with no command, a step with no
profile path, and two steps writing the same path — a **fourth** refusal time, against
*"Three refusals at three times."*

## O10 — MEDIUM-LOW · REAL — the activation honesty sentence cites the provenance of the attribute the adapter deliberately does **not** read.

`CHANGELOG.md:259-264` / `RELEASE_NOTES.md:23` cite `skill.name` as `[DOCS]`-labelled —
but `skill.name` is precisely the source both documents say was **rejected**. What
shipped keys on the event name `claude_code.skill_activated`.
`profiler/testdata/otlp/README.md` carries labels on exactly two lines (18 `[OBSERVED]`,
26 `[DOCS]`), **both byte-identical to v0.4.3**, and `claude_code.skill_activated`,
`invocation_trigger`, `skill.source`, `skill.kind` appear in **neither**. Five new
activation fixtures landed and the provenance ledger was not extended for the identifier
the whole new signal turns on: the disclosure points at the wrong record, and the right
record does not exist.

## O11 — LOW-MEDIUM · REAL — `experiment run`'s freshness stamp accepts a re-touched stale profile. Flagged by S6, absent from both.

`experiment.go:503-517` stamps `(exists, size, modTime)` and refuses `after == before` —
fails closed for *"writes nothing"*, so the notes' sentence is literally true. The
residual hole, named by the slice that built it (`s6-record.md:448`): a step that touches
or rewrites an identical earlier-run profile passes the stamp, passes
`profileMatchesStep`, and **an earlier run's numbers are reported as this run's.** The
notes disclose limits at *finer* granularity than this (`CHANGELOG.md:315-317` discloses
a backup collision needing three runs inside one second), so the omission is inconsistent
with the document's own threshold.

## O12 — LOW-MEDIUM · REAL — the `hooks.json` same-second backup collision is the one stated limit the two documents disagree on.

`CHANGELOG.md:315-317` states it; grepping `RELEASE_NOTES.md:3-29` for
`backup`/`collide`/`same second` gives **zero hits**. All 19 changelog limits
cross-checked; every other one has an RN counterpart. This is the single divergence, and
`RELEASE_NOTES.md` is the user-facing document.

## O13 — LOW · REAL · reproduced — `compare` and `experiment run` exit **2**; neither status appears in either document.

`compareExitCode`/`experimentExitCode` → `exitForComparable` → 2 when not comparable.
Reproduced on an all-unknown pair and on a version-mismatch refusal. Fully documented in
`README.md:370,451` and the spec, but the changelog is demonstrably exit-status-aware —
it spends a sentence on *"No new exit status"* for `draft-rewrite.sh` and a bullet on
*"`doctor` exits 0 whatever it finds"* — so two new scriptable statuses on two new
commands are a gap in the same enumeration.

## O14 — LOW · REAL · reproduced — `duckdb` is a new external tool, absent from Prerequisites, and its eight queries are exercised by no test and no CI step.

`CHANGELOG.md:75-77` ships *"eight independently runnable DuckDB queries."* All eight run
against a seeded spool on DuckDB v1.5.5: **8/8 exit 0 and return rows — the claim is
true.** But no Go test, shell suite, or CI step ever invokes `duckdb`;
`analyze_test.go`/`hooks_test.go` read the `.sql` files **as text only**; `ci.yml` is
byte-unchanged and installs no DuckDB; and `duckdb` does not appear in README
Prerequisites — the table 0.4.3 made a point of holding in both directions. The single
verification is a manual one-off in `profiler/queries/README.md` that no released note
references. In a release whose thesis is *"a capability is what a probe read,"*
"independently runnable" is an unautomated claim presented beside automated ones.

## O15 — LOW · REAL — "Four things were deliberately not shipped … a Cursor adapter" undercounts its own paragraph.

`CHANGELOG.md:13-15` / `RELEASE_NOTES.md:5`. Twelve lines later the same section says
*"No Cursor, Codex or Devin adapter ships."* Three adapters were declined on a measured
finding, not one.

## O16 — LOW · SUSPECTED · reproduced — `-o ""` is silently accepted and writes to the default destination.

`draft-rewrite.sh:394` refuses an empty `-o`, but the call site at `:434` is guarded by
`-n "$output_arg"`, so the branch is unreachable. Reproduced: `-o ""` → *"Rewrite draft
written to: …/sk/REWRITE-DRAFT.md"*, exit 0. "Four destinations" is therefore *correct*
for reachable refusals; the undisclosed behaviour is the fall-through — a script running
`-o "$DEST"` with `$DEST` unset silently writes into the target skill's own directory,
the outcome `-o` exists to avoid.

## O17 — LOW · REAL — new vocabulary shipped unnamed in both documents.

`hooks_estimated` (`types.go:60`) is a new legal value of the `source` key, written by
nothing that ships (`s8-record.md:573`: *"declared and used by nothing"*). The four
attribution field names appear in neither document. Same root as O5.

## O18 — LOW · REAL — "Every slice in this release proved its own checks by mutation" against a known-vacuous assertion half left in place.

`CHANGELOG.md:216-218`. `s9-record.md:516` §8.3 records that one assertion's label
*"promises slightly more than its command decides… Named, not changed."* S9 argues
correctly that mutation A4 kills through the load-bearing half, so this is a labelling
defect rather than a hole — but the notes state the mutation claim unqualified while the
slice that made it recorded a residual. (S7 §4.5's two by-construction survivals and S5
§7.6's unreachable `successRate` guard are the same family; all three argued sound
in-record.)

---

## Walked and CLEAN

5/5 plugin manifests · `.gitignore` (6 entries + widening, disclosed with two-direction
check) · `.out-of-scope.md` · `AGENTS.md` · README's 16 new headings · `spool_read.go`
rename (nothing shipped the old name — no migration note owed) · `doctor` two tiers +
`getenv` removal + exit 0 · `hooks install` idempotency (rerun → `events_registered: 0`,
no backup) · 21/21 event names · redaction defaults (`nested.api_key` → `[REDACTED]`
verified) · `TestTheContractTableCanFail` exists · 3/3 contract obligations · 7/7
mutation verdicts with 7 distinct statuses · `mutation_bounded` present ·
`skill_name_present.json` exists · `--harness` accepted set unchanged · `--export-file`
still rejected · `probe`/`capture` exit codes unchanged · `test_skill.sh` version asserts
· `docs/profiler-spec.md:72` correctly true past tense · exported Go API **0 removals** ·
**0 deletions, 0 renames** · all three structural-honesty checks.

The seven seeded Go-file forward-promises were **not re-litigated**; for the record they
reproduce at head.

## Fix direction — ADVISORY, re-derive at the root

**The invariant: an enumeration in the notes must be derivable from the code, or must not
claim closure.** The release already owns this technique — `SignalStates` is
reflection-derived, the query columns are derived from `SpoolEvent`, the suite list from
`ci.yml`. The counted prose lists are where it was not applied. Candidate direction:
derive the "written by nothing that ships" set and the "note, not refusal" set the way
`SignalStates` is derived, and drop the bare numerals from sentences that cannot be held
mechanically.

**O1 and O3 share a different root and want a different repair:** a release-commit step
that diffs `AdapterVersion`, `ProfileSchema` and each signal's reachable state set against
the previous tag, so an upgrade consequence cannot fall between two slices that each did
their part correctly.

**O2** wants the sentence narrowed to what the compiler printed (`codex.go` and
`cursor.go`), and the commit its `file:line` citations refer to named in the notes so a
reader can reach it.
