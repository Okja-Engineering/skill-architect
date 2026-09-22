# Gate lens: stale and false version markers — PR #22 @ `9c8ba53`

> **Persisted by the chief of staff.** The hunter had no Write tool, said so
> explicitly, verified the absence by `ls`, and returned its report inline without
> falling back to a heredoc. This is that report.

**VERDICT: NOT CLEAN — 18 findings beyond the seeded seven.**

## Coverage

**162/162 tracked files** at `9c8ba53`, walked in its own worktree
(`scratchpad/stale/head`, detached; primary tree at `0a83615` untouched).

- **Partition, not sampling.** A 24-term class pattern run against every `git ls-files`
  entry: **95 hit / 67 clean = 162**. All 95 read.
- **18 named marker-grammar greps** (`-F`, exact): `for 0.5.0`, `in 0.5.0`, `is 0.5.0`,
  `to 0.5.0`, `0.5.0 work`, `0.5.0 item`, `0.5.0 adds`, `0.5.0 did`, `0.5.0 does`,
  `0.5.0.`, `0.5.0,`, `0.5.0;`, `belongs to 0.5.0`, `0.5.0 will`, `0.5.0 content`,
  plus `0.6.0|0.7.0|0.8.0|1.0`.
- **Reference-tracing, not grep, for identifiers**: all 7 `MetricSource` and both
  `EnvironmentTier` constants traced to every occurrence in the module.
- 5/5 plugin manifests read in full; six declared version surfaces cross-checked
  against their asserts. Ladder read from `.scuba/roadmap.md:16-22,44`. 8 spec commits
  confirmed via `git log`.
- **Second sweep used different patterns** (maturity, scheduling, slice-level, deleted-adapter
  identifier vocabularies) and produced F10–F17. A third pass added nothing.

**Empirical proofs run, not reasoned:** built `./cmd`; `probe` on unreadable export
(exit 0 — F1); `capture --harness cursor|codex|devin` (all "unknown harness" — F5/F6);
`tests/test_rewrite.sh` 136/136 with the three absence-asserts green (F2); `go test ./...`
all 3 packages ok. **The tree is green — every finding is a false marker in a passing tree.**

## Shared root — two, and they differ

1. **Re-point commit `9c8ba53` touched only `README.md` and `docs/profiler-spec.md`.**
   It repaired 7 lines across 4 marker topics in prose and left **every other surface
   carrying the same sentence.** That produces the seeded seven *and* F1, F2, F3.
2. **The class has no tripwire, though this repo built the machinery twice and applied
   it elsewhere.** `scanFor` (`profiler/profiler_test.go:2075`) is a generic tree-scan
   already holding one refused string out; `declaredTiers`
   (`profiler/adapter_contract_test.go:1080-1124`) reads constants out of `doctor.go`
   by AST and asserts exactly `{TierExport, TierNone}`. Neither is pointed at version
   markers or at `MetricSource`. **`scanFor`'s roots are `.`, `../docs`, `../README.md`
   — a guard built on it as-is would miss `tests/`, `skills/`, the manifests and
   `.gitignore`, i.e. would miss F2 and F10. Any guard's denominator must be `git ls-files`.**

---

## A. False-forward 0.5.0 markers the re-point never touched

**F1 · HIGH · REAL (reproduced) · `profiler/cmd/main.go:118`**
`// failure audible is the repair; making it branchable is 0.5.0.`
The probe-exit-contract marker — same marker whose prose copies were repaired at
`README.md:98` and `docs/profiler-spec.md:837`. The README's new text directly
contradicts it: *"v0.5.0 did not add one, and no release has committed to adding one."*
Proven: `probe --harness claude_code --otel-file /nonexistent/nope.json` → report on
stdout, error on stderr, **EXIT=0**.

**F2 · HIGH · REAL (reproduced) · `tests/test_rewrite.sh:1170-1176`**
Comment says the capability "is 0.5.0" and that the three absence-asserts "go red when
it is built". Proven: 136 passed / 0 failed, all three green. Not built in 0.5.0.
Worse, contradicted by the **shipped** `skills/skill-rewrite/SKILL.md:117`, which frames
the gap as a permanent design boundary (*"Stage 1 and Stage 3 work, done by the reader"*).
One of the two is wrong and neither cites the other.

**F3 · MEDIUM · REAL · `profiler/profiler_test.go:2015`**
`// The unlanded 0.5.0 work renamed TokenCounts.CacheCreation, and its`
At the tag, 0.5.0 is not "unlanded". Per the standing ruling (`.scuba/roadmap.md:44`)
the rename **stays dropped outright**, not deferred. Test passes; only the comment is stale.

## B. The release's own disclosure of the re-point is incomplete

**F4 · MEDIUM · REAL · `RELEASE_NOTES.md:21`, `CHANGELOG.md:189-193`** — a **fifth**
0.5.0 promise exists, disclosed nowhere. `CHANGELOG.md:400` (frozen 0.4.3 section) says
*"Building that capability is 0.5.0 work"* about the drafter. The project's convention
puts the correction in the current section (`RELEASE_NOTES.md:26`, `CHANGELOG.md:227`:
*"The released notes are left as written; this is the record"*). It is not there —
confirmed by grepping the whole 0.5.0 CHANGELOG section (7-344) and RELEASE_NOTES
section (3-29). Doc-side twin of F2.

**F5 · MEDIUM · REAL · same lines** — "**Four** deferral markers", **three** enumerated.
The fourth, visible in `9c8ba53`'s own diff, is the timing gap at
`docs/profiler-spec.md:806`. Re-pointed in the spec, named in neither release document.
*A count that does not match its own list is the defect this release is about.*
(Note: the code copy at `profiler/claude_code.go:837-838` carries **no** version and is
correct — do not "fix" it into one.)

## C. Deleted-adapter references false at head

Baseline proven: `capture --harness cursor|codex|devin` → `unknown harness`.
`adapterRegistry` (`profiler/adapters.go:38-43`) holds exactly one entry.

**F6 · MEDIUM · REAL · `docs/profiler-spec.md:7`** — Purpose says the profiler captures
from *"any target harness (Cursor, Claude Code, Codex, Devin)"*. Present tense, all four,
first sentence a reader meets. `README.md:35-37` handles the same fact honestly.

**F7 · MEDIUM · REAL · three copies of one comment** — `profiler/types.go:69`,
`docs/profiler-spec.md:100`, `docs/profiler-spec.md:130`: the `Harness` field's stated
vocabulary is `"cursor" | "claude_code" | "codex" | "devin"`. Three of four are names
`NewAdapter` refuses. No qualifier.

**F8 · HIGH · REAL · five exported `MetricSource` constants no shipped capture can produce**

| constant | decl | refs in tree | production refs outside decl |
|---|---|---|---|
| `SourceHooks` (`"hooks"`) | `types.go:54` | **1** | 0 |
| `SourceSessionData` (`"session_data"`) | `types.go:61` | **1** | 0 |
| `SourceServerAPI` (`"server_api"`) | `types.go:62` | **1** | 0 |
| `SourceSQLite` (`"sqlite"`) | `types.go:63` | 7 | 0 (tests only) |
| `SourceHooksEstimated` (`"hooks_estimated"`) | `types.go:60` | 8 | 0 (tests only) |

`SourceSessionData` was the Devin draft's source, `SourceSQLite` the Cursor draft's
(both named in `RELEASE_NOTES.md:15`). `SourceServerAPI`'s string is one of the three
**removed doctor tiers**. Unlike `CaptureOpts.ExportFile`/`APIKey`, **none carries any
comment** — the claim is made silently by the declaration. Compounding:
- `types.go:28` and `docs/profiler-spec.md:30` declare the whole set as the `source`
  field's vocabulary — a published schema vocabulary in which one value is producible.
- `profiler/cmd/main_test.go:1704-1707` asserts the doctor report may **not** contain
  `"hooks"`, `"server_api"`, `"enterprise"` — *"which no capture in this release
  delivers"* — while two of those three are declared valid profile sources 1,600 lines away.
- The release built the exhaustiveness guard for the **other** vocabulary and not this
  one. `git grep declaredSources` → empty; `SourceServerAPI` and `SourceSessionData`
  appear in **no** test.

**F9 · LOW-MEDIUM · REAL · `docs/profiler-spec.md:159-160`** — the spec's copy of the two
reserved fields is **silent** rather than false: no "reserved for 0.5.0", but also no
"no shipped adapter reads either", where `profiler/cmd/main.go:145` and `:675` say it
plainly. Repairing only the Go comments leaves the spec as the surviving undisclosed claim.

**F10 · LOW · SUSPECTED · `profiler/otlp.go:25`** — *"A second OTLP-speaking adapter
(Cursor, Codex) would read this layer as it stands"*. Hypothetical, so not a promise,
but the release's own measurement refutes the premise: `RELEASE_NOTES.md:15` records the
Cursor draft read tokens from **sqlite**, and `README.md:35` gives Cursor's surface as
lifecycle hooks.

## D. Beyond-0.5.0 markers against the approved ladder

**F11 · MEDIUM · REAL · `tests/lib/out-of-scope-check.sh:6-7`** — two mis-attributions:
`docs/research/` is called 0.6.0 work, but `docs/research/rule-language.md` is the
**0.7.0** spec and `docs/research/**` is under a standing ruling to be kept permanently;
and `NOTICE` exists solely to attribute `skillgate/difftest/`, which 0.6.0 has **carved
out**. The barrier's *behaviour* is right — none of these may ship with 0.5.0 — so this
is not a functional bug, it is the same defect one release out. `.gitignore:11` is
**correct** (names only `skillgate/`, `skills/skill-gate/`), as are
`tests/test_harness.sh:499,550`.

## E. Stale in-flight markers — the release is now the release

- **F12 · LOW-MEDIUM · REAL · `tests/test_harness.sh:556-559`** — justifies the
  `boundary_stageable` list by *"the release has several [slices] left"*. None remain.
- **F13 · LOW · REAL · `profiler/cmd/main_test.go:1479`** — *"Every slice from here to
  the release adds a subcommand."* At head, this **is** the release.
- **F14 · LOW · REAL · `README.md:34`** — Status column reads `✅ Slice 1` beside three
  "Planned" rows, though the adapter took work in 0.4.1/0.4.2/0.4.3 and S3/S4 here. The
  row's third column was updated for 0.5.0; the status was not.
- **F15 · LOW-MEDIUM · REAL · `docs/profiler-spec.md:776,778,844`** — headings labelled
  "(Slice 1 scope)" / "(Slice 1)" over content spanning four releases. AC2 (`:847`) and
  AC4 (`:849`) are the **0.5.0 S4** skill-activation criteria; AC3 (`:848`) is 0.4.3's
  mixed-temporality refusal. A false scope label on the release's normative ACs.
- **F16 · MEDIUM · REAL (reproduced) · `docs/profiler-spec.md:3`** — header reads
  `**Status:** spec · **Date:** 2026-09-13`. `git log` shows the file modified in
  **eight** commits of this release, last dated **2026-09-21**, gaining five new contract
  sections. Eight days and five slices behind. `Status: spec` compounds it — the document
  is now the reference for shipped behaviour and says so itself at `:518`.
- **F17 · LOW-MEDIUM · REAL · `README.md:26,28`** — `## The profiler (preview)` and
  *"v0.4.0 adds a harness-agnostic profiler"*, while 0.5.0's notes lead with *"Six new
  profiler subcommands"*. Both release documents confine "preview" to the v0.4.0 entry.
  `9c8ba53` edited that exact paragraph and left both markers.
- **F18 · LOW · SUSPECTED · `docs/research.md:78,80-83`** — a v0.1.0-era document
  carrying version-scoped scope claims and open questions with **no frozen/historical
  marker**, where CHANGELOG and RELEASE_NOTES both state their old entries are left as
  written. 0.5.0 shipped `experiment design|plan|run`, partly answering both.

---

## Walked and CLEAN — stated so a fixer does not "repair" a correct surface

- **All six declared version surfaces.** Five manifests at `0.5.0`
  (`.claude-plugin/plugin.json:4`, `marketplace.json:13`, `.codex-plugin:3`,
  `.cursor-plugin:3`, `.devin-plugin:3`), all five asserted (`tests/test_skill.sh:85-96`).
  `AdapterVersion = "0.5.0"` asserted twice, both PASS; `./profiler version` prints
  `profiler 0.5.0`. The deliberate non-equality of manifests and `AdapterVersion` is
  documented identically in three places.
- **Schemas** — `ProfileSchema`, `ComparisonSchema`, `ExperimentSchema`,
  `ExperimentPlanSchema`, `ExperimentResultSchema`, `SpoolSchemaVersion` all `/v1`,
  consistent with `RELEASE_NOTES.md:5`.
- **Removed doctor tiers — exemplary, do not touch.** Survive only in the honest "Three
  things it does not report" rationale (`doctor.go:34-49`, spec `:762-771`).
  Structurally guarded twice: AST-read constant set and a string-absence assert on real
  CLI output. `adapter_contract_test.go:982-990` even refuses a fixture for a tier
  `doctor.go` no longer declares.
- **`spool_profile.go` → `spool_read.go`** — exactly two references, both honest and
  disclosed. Nothing in test names, fixtures, or `queries/`.
- **Skill `metadata.version` disclosure** — consistent between `CHANGELOG.md:327-334`
  and `RELEASE_NOTES.md:23`; both name `0.2.0`/`0.1.0` and the manifests' `0.5.0`, both
  call it the maintainer's open decision, both state nothing asserts a relationship —
  and nothing does. Both even name the aggravating fact that 0.5.0 changed
  `skill-rewrite`'s interface. **No disclosure inconsistency found.**
- **The freeze convention holds** for every 0.4.x-section "0.5.0" marker in
  `CHANGELOG.md` (422, 424, 477, 479, 484) and `RELEASE_NOTES.md` (59, 67, 86, 90) —
  each is inside a frozen section with its correction in a later section. **The one
  exception is F4.**
- `.github/workflows/ci.yml` — no markers beyond pinned tool versions; no comment
  promising the `macos-latest` job R4 defers, and the deferral is disclosed at
  `RELEASE_NOTES.md:18`.
- `profiler/queries/` (9 files) — every version-adjacent sentence is "in this release"
  phrasing that stays true at 0.5.0.
- `profiler/profiler_test.go:322-359` — `retiredActivationDeferral = "Reading it is
  0.5.0."` is the one place this class **is** enforced. **This is the shape a root fix
  should generalise.**
- Honest deleted-adapter surfaces: `README.md:35-37,327`, `cmd/main.go:145,675`,
  `adapters.go:37`, and all of `hooks.go`/`hooks_install.go`/`spool_read.go`/`analyze.go`.
- `.out-of-scope.md`, `PRINCIPLES.md`, `AGENTS.md`, `.gitignore` — no stale markers.
- `TODO`/`FIXME`/`XXX`/`HACK`/`WIP` — zero real instances tree-wide.

## Fix direction — ADVISORY ONLY, re-derive at the root

**The invariant:** at the tag, no marker anywhere in the tree may assign work to the
version being tagged; every marker naming a later version must name one the approved
ladder actually assigns it to; and every declared vocabulary must be the set a shipped
capture can deliver, or each unproducible member must say so.

1. **F1/F2/F3 plus the seeded seven are one edit-set, not ten edits.** Group by *marker
   topic*, not by file: probe-exit (`main.go:118`, `main_test.go:369` + two repaired
   prose copies); caveat-channel (`claude_code.go:475`, `:611`, `otlp.go:1130` + three
   repaired prose copies); reserved-fields (`types.go:81`, `:84`, `:85` + silent spec
   copy F9); drafter-capability (`test_rewrite.sh:1173` + F4's missing disclosure).
2. **A guard is what stops the fifth recurrence**, and the repo has two working models.
   But the denominator must be `git ls-files`, and the check must distinguish
   *assignment* ("is 0.5.0", "for 0.5.0", "reserved for") from *history* ("0.5.0 did not
   add", "gained … in 0.5.0"), or it fires on ~40 correct lines. `declaredTiers`' AST
   approach is the better model for F8.
3. **F5's count mismatch and F4's missing item are in the notes, not the code.**
   Repairing the code without them leaves the release claiming a completeness it does
   not have — which is the release's own subject matter.

**Live worktree left for the fixer (do not `checkout`/`reset` in it):**
`scratchpad/stale/head` @ `9c8ba53`, built binary at `scratchpad/stale/profiler`.
