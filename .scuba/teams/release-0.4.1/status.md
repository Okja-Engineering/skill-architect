# Status — v0.4.1 patch release — build COMPLETE, round-1 repair COMPLETE, item 19 COMPLETE

**Branch:** `release/0.4.1` (based on `origin/main` @ 30f374c)
**Worktree:** `/Users/matthewvandusen/Development/Auraprix/skill-architect-wt/release-0.4.1`
**Head SHA:** `82ae271d6c21fdc001f3ecad540c00fe46f0af26` (item 19, OTLP parser; round-1 repair ended at `527ba4e`; build at `7cb32e2`)
**PR:** https://github.com/Okja-Engineering/skill-architect/pull/2 — OPEN, **not** a draft, base `main`

All 18 mandate items done at `7cb32e2`; the round-1 worklist (W1–W18) is
repaired at `527ba4e`; mandate item 19 (parse real OTLP/JSON) is built at
`82ae271` — see **Item 19 — OTLP parser** at the bottom of this file, which is
the current record, and **Round 1 — root repair** above it. The build sections below are kept as written
and are accurate as of `7cb32e2`; where round 1 superseded them, the round-1
section says so.

Primary tree untouched (still on `main` @ 0a83615 with its own uncommitted
prototype); the only thing written there is this `.scuba/` file.

## Verification (at build head 7cb32e2)

```
cd profiler && go build ./... && go vet ./... && go test ./...
ok  	github.com/Okja-Engineering/skill-architect/profiler	0.297s
?   	github.com/Okja-Engineering/skill-architect/profiler/cmd	[no test files]
17 test functions pass (15 at base + 2 added); gofmt -l clean; go vet clean

tests/test_skill.sh  → 22 passed, 0 failed
tests/test_walk.sh   → 19 passed, 0 failed
tests/test_f01.sh    → 32 passed, 0 failed
tests/test_f02.sh    → 58 passed, 0 failed    (131 total, unchanged from base)
```

Baseline at 30f374c was identical (131 shell, 15 Go) — no regressions.

## Commits (13, oldest first)

```
ad62ced test(profiler): pin the Probe/Capture contract for partial OTel exports   [RED]
a1e7d04 fix(profiler): capture every signal the probe advertises, not just tokens [GREEN]
f41c6e3 chore(profiler): correct the module path to the repo's actual owner
3429e1f ci: take the Go version from profiler/go.mod
ccdd41c refactor(profiler): drop CaptureOpts.OtelEndpoint, which nothing reads
e39cae1 docs(spec): describe the profiler types and probe semantics that exist
cab7026 docs: stop claiming the profile is snapshot-pinned
8293c6e docs(readme): state the install prerequisites and the profiler surface
74f53ac docs(changelog): correct 0.4.0's claims and add the 0.4.1 section
6c17c1c chore(release): bump plugin manifests to 0.4.1
d1cb0b0 docs(profiler): align the adapter's own comments with the softened claim
a8ad895 docs(readme): correct what a missing skill-validator actually does
7cb32e2 docs(spec): tighten three acceptance criteria left imprecise
```

Every message ends `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>`.

## Per-item table

| # | Item | State | Evidence (file:line **at the build commit `7cb32e2`**, not at the current head — round 1, item 19 and round 2 have moved these lines; see the later sections for the head cites) |
|---|---|---|---|
| 1 | Capture gated the whole OTel path on token capability | done `a1e7d04` | `profiler/claude_code.go:99` `if !cap.AnySource()`; `profiler/types.go:57-69`. Tests `profiler/profiler_test.go:420` (contract, 6 shapes) and `:483` (values). RED→GREEN→RED→GREEN below. |
| 2 | README Prerequisites block | done `8293c6e`, `a8ad895` | `README.md:81-96`. Masked/unmasked PATH evidence below. |
| 3 | README sibling `skill-audit` requirement | done `8293c6e` | `README.md:135-140`. Claim verified empirically — see below. |
| 4 | module path → Okja-Engineering | done `f41c6e3` | `profiler/go.mod:1`, `profiler/cmd/main.go:9`. Build/vet/test green on the new path; grep for `auraprix` clean. |
| 5 | CI `go-version-file` | done `3429e1f` | `.github/workflows/ci.yml:32-34`. YAML parses. |
| 6 | spec: real `RawMetricResult` shape | done `e39cae1`, `7cb32e2` | `docs/profiler-spec.md:11-69`, `:151-155`. Verified by compiling + running a throwaway `profiler/scratch_spec_conformance_test.go` written only against the documented types, then deleting it. |
| 7 | spec AC2 probe semantics | done `e39cae1`, `7cb32e2` | `docs/profiler-spec.md:232`; AC3 `:233`; capture-logic `:217-225`; new AC9 `:239`. |
| 8 | spec fallback reason string | done `e39cae1` | `docs/profiler-spec.md:227` matches `profiler/claude_code.go:100` verbatim. |
| 9 | delete `CaptureOpts.OtelEndpoint` | done `ccdd41c` | Removed from `profiler/types.go` CaptureOpts and `docs/profiler-spec.md:129-134`. Nothing read it (grep clean). |
| 10 | "snapshot-pinned" reword | done `cab7026` | `README.md:28`, `RELEASE_NOTES.md:22`, `docs/profiler-spec.md:7,139,147,199`, `profiler/types.go:166-168`. No hashing added. |
| 11 | test count | done `74f53ac` | 14 → 15 in the 0.4.0 entries (`CHANGELOG.md`, `RELEASE_NOTES.md`); 0.4.1 sections state **17**. |
| 12 | CHANGELOG gitignored path | done `74f53ac` | `tmp/teams/architect/profiler-design-space.md` reference removed; sentence kept. |
| 13 | CHANGELOG 0.3.1 `references/` note | done `74f53ac` | Note under 0.4.1 "Known past changes"; 0.3.1 entries left as written. 3e2efd8 confirmed as the move. |
| 14 | `ToolCallEntry.Duration` removal note | done `74f53ac` | Note under 0.4.1. Confirmed against 30f374c (`-Duration int64 json:"duration_ms,omitempty"`). Schema string unchanged at `v1`. |
| 15 | "no skill-level events" softening | done `74f53ac`, `d1cb0b0` | `CHANGELOG.md`, `RELEASE_NOTES.md`, plus `profiler/claude_code.go` adapter doc comment and `docs/profiler-spec.md:215` which asserted the same. `skill.name` on token/cost metrics corroborated against Claude Code OTel docs. |
| 16 | version → 0.4.1 | done `6c17c1c` | 4 manifests + `tests/test_skill.sh:28`. Assertion bumped first → RED `AssertionError: version mismatch` → manifests bumped → GREEN. |
| 17 | 0.4.1 sections | done `74f53ac` | `CHANGELOG.md:7` (`## Unreleased` empty above it); `RELEASE_NOTES.md:3`. |
| 18 | document version/--export-file/per-signal probe | done `8293c6e` | `README.md:54-75`. All three verified against the built CLI. |

## Item 1 — RED → GREEN → RED → GREEN

RED at `ad62ced` (test only, no fix):

```
--- FAIL: TestCaptureDeliversEverySignalProbeAdvertises/tool_calls_only
    tool_calls: probe advertised "otel", capture state = "unknown"
    (reason "OTel export not configured. Provide an OTel export file via
    --otel-file or OtelExportFile."), want "present"
--- FAIL: TestCaptureDeliversEverySignalProbeAdvertises/timing_only
--- FAIL: TestCaptureDeliversEverySignalProbeAdvertises/tool_calls_and_timing,_no_token_metric
--- FAIL: TestCapture_ClaudeCode_PartialExport_KeepsToolCallsAndTiming
    tool_calls count = 0, want 1 (state "unknown", ...)
```

GREEN at `a1e7d04`. Reverting `!cap.AnySource()` back to
`cap.Capabilities[MetricTokens] == SourceNone` reproduces exactly those four
failures; restoring goes green again.

Confirmed on the CLI surface, not only in unit tests:

```
$ profiler capture --harness claude_code --session s1 --snapshot sha1 \
    --skill-dir ./skills/skill-audit --otel-file partial.json
tokens      unknown | no claude_code.token.usage metric found in OTel export
tool_calls  present | ['Bash']
timing      present | {'start_time':'…22:00:00Z','end_time':'…22:00:15Z','total_ms':15000}
```

The reported symptom was one export shape; the class was three — tool-calls-only
and timing-only exports were broken identically and are covered by the contract test.

## Item 2 — masked / unmasked PATH evidence

```
skill-validator masked → "skill-validator not found. Install with: brew install
                          agent-ecosystem/tap/skill-validator"          exit 1
skillscore masked      → "ERROR: skillscore not found. Install with:
                          npm install -g skillscore"                    exit 3
jq masked              → audit-report.sh                                exit 127
                       → check-paths.sh --json      exit 127 (a finding to serialise)
                       → check-structure.sh --json  exit 0, {"findings": [], "passed": true}
  [round 2] The two --json lines above were observed on this repo's own skills,
  which happen to raise path findings only. Re-derived in round 2 against a
  fixture per branch — the surface has three, not one:
    a finding to serialise      → 127  (check-structure.sh:102, check-paths.sh:94)
    nothing to report           → exit 0, {"findings": [], "passed": true}
                                  (check-structure.sh:93, check-paths.sh:84;
                                   that branch never calls jq)
    path faults only, -structure→ jq present: passed:false exit 1
                                  jq masked : passed:true  exit 0   ← the drop
sv+ss masked, jq ok    → audit-report.sh exit 0, spec:null / quality:null,
                          spec_error + quality_error set, summary.passed:false
unmasked               → audit-report.sh exit 0, summary.passed:true,
                          quality_score 89.5, quality_grade "B+"
```

## Item 3 — sibling requirement, verified

`skill-rewrite` copied alone to a temp dir, `draft-rewrite.sh -t <skill>`: exits 0
and writes `REWRITE-DRAFT.md`, whose "Current state" section contains
`…/skill-rewrite/../skill-audit/scripts/check-frontmatter.sh: No such file or
directory` instead of an audit. The README says that, not "it fails".

## Found, not fixed

1. **`check-structure.sh --json` silently drops *path* findings when `jq` is absent.**
   `skills/skill-audit/scripts/check-structure.sh:71` —
   `path_findings=$(echo "$path_json" | jq -c '.findings[]' 2>/dev/null || true)`
   swallows the missing-binary error, so it exits 0 with
   `{"findings": [], "passed": true}` where jq-present returns two findings.
   **[round 2 — confirmed, and scoped.]** Re-derived against a fixture whose only
   fault is a broken script path: jq present → `{"findings":[{PT001…}], "passed":
   false}` exit 1; jq masked → `{"findings": [], "passed": true}` exit 0. A failing
   skill reported as a clean pass, which is the defect as written. Two refinements,
   both of which narrow it rather than dissolve it:
   - It is **path** findings only. Policy findings (PL002–PL005) are built in this
     script and never round-trip through jq before the output stage, so a skill with
     one of those dies at `:102` with `jq: command not found` and exit **127** instead.
   - The child's 127 is also lost: `:68` captures it into `path_code`, and `:86` only
     maps `path_code -eq 1` onto `fail`, so a child that died is indistinguishable
     from a child that passed.
   The fix is still a code guard (`command -v jq` at the top of both scripts, or
   checking `path_code` for anything non-zero), not a doc sentence. Still 0.5.0.
   A round-2 worklist item proposed recording this as "an inconsistent 127-vs-0
   surface, not a silent drop". That is wrong — see `a4fb247`; the drop reproduces.
2. **`check-frontmatter.sh` prints `frontmatter OK` when `skill-validator` is missing.**
   `skills/skill-audit/scripts/check-frontmatter.sh:24` — the missing binary makes the
   command substitution exit **127**, and the script special-cases only 1 and 3, so 127
   falls through the spec-failure branches to the license gate and exits 0 having
   validated nothing. The SKILL.md Stage 2 guard is what protects the documented
   workflow; running the script directly bypasses it.
   Both (1) and (2) are the silent-failure class this plugin exists to catch, and both
   need a code guard, not a doc sentence. Recommend opening them for 0.5.0. Documented
   in the README prerequisites in the meantime.
3. **`capture --export-file` is inert for the only shipped adapter.** `profiler/cmd/main.go:63`
   registers the flag and `profiler/claude_code.go` falls back to `opts.ExportFile`, but
   that branch is unreachable: `Probe` reports a capability only when `OtelExportFile` is
   set, so `--export-file` alone always yields an all-`unknown` profile. Documented as
   reserved rather than wired up — wiring it is adapter work.
4. **Error-path asymmetry in `Capture`.** When the export file exists but cannot be read
   or parsed, tokens become `error` while tool_calls and timing become `unknown` with the
   same reason (`profiler/claude_code.go:118-137`). By the spec's own definition
   ("error — source existed but failed") all three should be `error`. Same
   tokens-as-primary-signal family as item 1, but changing the states is a
   profile-output change beyond the mandate.
5. **`CaptureOpts.APIKey` is never read** (`profiler/types.go:73`), exactly like the
   `OtelEndpoint` item 9 removed. Left because item 9 named only `OtelEndpoint` and
   `APIKey` is documented as the Devin adapter's future auth surface.
6. **`CHANGELOG.md` 0.4.0 still says "`MetricResult` … pinned to a snapshot hash"**, both
   of which items 6 and 10 correct elsewhere. Left as dated history per the item-13
   rule; the 0.4.1 section carries the correction.

## Additions beyond the literal item text (flagged in the PR)

Each taken to avoid shipping a contradiction the item itself would have introduced:

- Named the three OTel signal strings once in `profiler/claude_code.go` — they were
  duplicated between the probe's detector and the extractors, which is the mechanism by
  which probe and capture drift apart (item 1's root cause).
- Spec AC9 and the capture-logic alignment — item 1 changed the behaviour that section
  describes.
- Adapter doc comment and spec probe-note (`d1cb0b0`) still asserted the claim item 15
  softens.
- One clause in the Prerequisites block noting the profiler needs Go at the `profiler/go.mod`
  version.

## Boundaries held

Nothing touched under `skillgate/`, `skills/skill-gate/`, `docs/skillgate-*`,
`docs/research/`, `.scuba/` (in the worktree), or `go.work`. No `cache_creation` →
`cache_write` rename. No commits cherry-picked from local `main`. The 0.3.1 `exit 1`
dependency guards in `skills/skill-audit/SKILL.md:46-47` are intact and now documented.
Never merged; PR left open for the user.

---

# Round 1 — root repair (bug-fixer, 2026-09-13)

**Head after repair:** `527ba4ea3e13f1cda87f2a843c5659c1c3dbd19e`, pushed to
`release/0.4.1`. PR #2 updated in place; no new PR, not merged. Worklist:
`worklist-round1.md`. Verified each finding against `7cb32e2` before fixing.

Repaired as **one integration pass**, not eighteen patches. Roots 1 and 2 are
one defect: `Probe` scanned the export for *structure* while `Capture` extracted
*values*, so there were two predicates per signal over two reads of one file.
The repair removes the second predicate rather than aligning it — `resolve()`
reads and parses once, each extractor is its signal's only predicate, and the
capability report is *derived* from what those extractors returned. Probe and
capture now cannot disagree, because they are the same code. `hasOtelSignals`
(which swallowed the parse error) is gone; `CapabilityReport.AnySource` (added
on this branch for the old shape) has no caller and is deleted too.

## Commits (11, oldest first)

```
a6f3b4d test(profiler): pin "present means a value was read" ...        [RED]
c192f6f fix(profiler): resolve the export once, so probe and capture
        cannot disagree                                                 [GREEN]
7cff73f fix(profiler): refuse --export-file rather than accept a path nothing reads
23f82df chore(profiler): version the adapter at the release that changed its output
d8b9a6b docs(readme): make the install route real and the profiler section true
6205fdc docs(spec): describe the adapter that exists, and drop the helper nothing reads
99e2cde docs(changelog): record what this release actually fixes, with true numbers
6514a0f ci: stop setup-go caching a module with nothing to cache
3aa4e51 docs(readme): say "all three OTel signals", not "every signal"
411ed0a docs: unknown now also means "the source was there and unreadable"
527ba4e docs(changelog): two entries overstated their own claims
```

Each message ends `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>`.

## Per-W disposition

| W | Verdict at head | Fix | Where (at 527ba4e) |
|---|---|---|---|
| W1 | REAL, reproduced 3 ways | `extractTokenCounts` requires a recognised `token_type` **and** a numeric value; `toInt` returns `(int, bool)` so "zero" and "not a number" stop being the same thing. Same rule applied to the other two signals — `tool_decision` needs a `tool_name`, `api_request` needs a parseable timestamp — because the class, not the instance, was the bug. | `profiler/claude_code.go:189` `extractTokenCounts`, `:226` `extractToolCalls`, `:257` `extractTiming`, `:297` `toInt`. `c192f6f` |
| W2 | REAL, both halves | Timing has no separate found-sentinel: it is one of the three extractors, present only when it read a parseable timestamp. The span is earliest→latest, not first→last in file order, so `end_time >= start_time` and `total_ms >= 0` hold on any ordering. Reproduced before the fix as `total_ms = -8000`. | `profiler/claude_code.go:257` `extractTiming`. `c192f6f` |
| W3 | REAL | Contract test now walks `report.Capabilities` as its denominator (asserts all 5 signals are covered), asserts `present ⇒ value carried` and `not present ⇒ reason, no value`, over 17 export shapes including malformed, missing-file, unknown-type, non-numeric, no-timestamp, no-tool-name, out-of-order, and mixed readable/unreadable. Plus 3 dedicated tests. | `profiler/profiler_test.go:411` (invariants + `capturedSignals` helper), `:461` `TestCaptureDeliversEverySignalProbeAdvertises`, `:597` `TestCapture_UnusableExport_IsDiagnosable`, `:646` `TestTimingIsASpanNotFileOrder`, `:684` `TestTokens_OnlyReadableMetricsAreCounted`. `a6f3b4d` |
| W4 | DEFERRED as routed | Documented in both places rather than changed: spec capture-step 3 and the code where the value is set. | `docs/profiler-spec.md:216`; `profiler/claude_code.go:241-245`. `c192f6f`, `6205fdc` |
| W5 | REAL | Root, not symptom: the parse error was swallowed in `hasOtelSignals` because probe re-read the file. One read, one parse, one classification — a supplied-but-unusable export is now `error` naming the read or parse failure for all three OTel signals. | `profiler/claude_code.go:63` `resolve()`. `c192f6f` |
| W6 | REAL | Picked **error** (the spec's Rules), and made code, spec and test agree. The three unreachable branches are gone. `ErrorToolCallResult`/`ErrorTimingResult` added — their absence was *why* the asymmetry existed. `MetricError` is now emitted and covered. | `profiler/types.go:224` `ErrorToolCallResult`, `:259` `ErrorTimingResult`; `docs/profiler-spec.md:224` (Fallback), `:238` (AC11); `profiler/profiler_test.go:597`. `c192f6f` |
| W7 | REAL | Decided the fail-loudly arm. Adapter-side fallback deleted with the refactor; the CLI refuses the flag, naming the adapter and the flag that works. Flag kept declared because `CaptureOpts.ExportFile` is the documented surface for a future session-export adapter, and a named refusal beats `flag provided but not defined`. | `profiler/cmd/main.go:110` `captureFlagError`, called at `:77`; test `profiler/cmd/main_test.go`. `7cff73f` |
| W8 | REAL | All reason strings build from `otelTokenUsageMetric` / `otelToolDecisionLog` / `otelAPIRequestLog`. `grep '"claude_code\.' profiler/claude_code.go` returns only the three const definitions. | `profiler/claude_code.go:13-17` and the extractors. `c192f6f` |
| W9 | REAL | `AdapterVersion = "0.4.1"`. Nothing else added. README's `version` sentence corrected with it (it claimed `0.1.0`; the CLI prints `profiler 0.1.0` → now `profiler 0.4.1`). | `profiler/types.go:179`; `README.md:54-57`. `23f82df` |
| W10 | REAL, both | `1.23` → `1.27.1` is **four** minors (CHANGELOG). Only `30f374c` sits between v0.4.0 and base, so AC2 was **one** commit out of date (RELEASE_NOTES). Both verified from `git log 541af3e..30f374c`. | `CHANGELOG.md:23`; `RELEASE_NOTES.md:14`. `99e2cde` |
| W11 | REAL | Interface comment no longer offers an "OTel endpoint" the branch deleted from `CaptureOpts`. | `docs/profiler-spec.md:120-121`. `6205fdc` |
| W12 | REAL | Corrected in place, per the worklist's reasoning (this branch already edits sibling 0.4.0 bullets in place). `types.go` comments, `CHANGELOG.md` 0.4.0 bullet, `RELEASE_NOTES.md` 0.4.0 bullet. The 0.4.1 bullet that said it "corrects 0.4.0's … below" was updated to match. | `profiler/types.go:17`, `:121`, `:128`, `:135`, `:141`, `:148`; `CHANGELOG.md:49`; `RELEASE_NOTES.md:23`. `6205fdc`, `99e2cde` |
| W13 | REAL — confirmed against the published reference | Anthropic's monitoring docs put `skill.name` on `claude_code.token.usage`, `claude_code.cost.usage`, `claude_code.api_request` and `claude_code.api_error`, and `skill_name` inside `tool_parameters` on `claude_code.tool_result`. `api_request` is a log this adapter reads, so "no output-to-skill mapping" was false. Every site now states the adapter-side fact instead of a harness-side absolute, and none enumerates which events carry the attribute — that is item 19's territory. "Claude Code has no skill-level activation *events*" is kept: the published event list has no activation event, so it is true. | `profiler/claude_code.go:19-28` (doc comment), `:115-118` (capability comment), `:158-161` (activation kept, attribution reason now "this adapter does not read Claude Code's skill attribution attributes"); `docs/profiler-spec.md:211`, `:220`; `README.md:75-76`; `CHANGELOG.md:38`. `c192f6f`, `6205fdc`, `d8b9a6b`, `99e2cde` |
| W14 | REAL | Fallback and AC5 rewritten: the not-configured reason belongs to the three OTel signals only; `skill_activation`/`attribution` always carry their own; a supplied-but-unreadable file is `error`, not that reason. | `docs/profiler-spec.md:224` (Fallback), `:232` (AC5). `6205fdc` |
| W15 | REAL — verified live, see below | Added `.claude-plugin/marketplace.json` (the standard mechanism) and documented `marketplace add` + `install name@marketplace`. | `.claude-plugin/marketplace.json`; `README.md:112-121`, `:142-148`. `d8b9a6b` |
| W16 | REAL, all six | version string (W9 row); "without validating anything" → the license gate still runs, verified empirically; `../skill-audit` resolves from the **skill** dir, one level above the scripts dir; spec header date → 2026-09-13; test counts stated per commit (14 at v0.4.0, 15 after 30f374c, 22 functions / 42 cases now — counted at each commit with `git show <sha>:profiler/profiler_test.go \| grep -c '^func Test'`). | `README.md:54`, `:97` (prerequisites cell), `:158-162`; `docs/profiler-spec.md:3`; `CHANGELOG.md:36`, `:53`; `RELEASE_NOTES.md:17`, `:27`. `23f82df`, `d8b9a6b`, `6205fdc`, `99e2cde` |
| W17 | REAL | This section is the record; the omitted additions are listed under "Found, not fixed — round 1" below. | this file |
| W18 | REAL (`cache: false` only) | Added with the reason in a comment. Workflow still parses; the step resolves to `{go-version-file: profiler/go.mod, cache: False}`. The `go vet`/`-race`/`gofmt` and pinned-install items stay deferred and are recorded below. | `.github/workflows/ci.yml:32-37`. `6514a0f` |

## RED → GREEN → RED → GREEN

**Roots 1 and 2.** At `a6f3b4d` (test only, no fix), 11 subtests fail:

```
--- FAIL: TestCaptureDeliversEverySignalProbeAdvertises/token_metric_with_an_unrecognised_token_type
    tokens: nothing readable in the export, probe advertised "otel"
    tokens: state = present with no readable value in the export
    tokens: state "present" carries a value; unknown and error results must carry none
    tokens: state = "present", want "unknown"
--- FAIL: .../token_metric_with_a_non-numeric_value          (same four)
--- FAIL: .../token_metric_with_no_attributes                (same four)
--- FAIL: .../tool_decision_with_no_tool_name                (same four, tool_calls)
--- FAIL: .../api_request_with_no_timestamp                  (same four, timing)
--- FAIL: .../malformed_JSON                 tokens/tool_calls/timing: state = "unknown", want "error"
--- FAIL: .../truncated_after_a_valid_prefix (same)
--- FAIL: .../export_file_path_set_but_file_missing (same)
--- FAIL: TestCapture_UnusableExport_IsDiagnosable/malformed_JSON
    tokens: state = "unknown" (reason "OTel export not configured. Provide an OTel
    export file via --otel-file or OtelExportFile."), want "error" — the export
    existed and failed
    tokens: reason … — the export was configured; this sends the user to fix the
    wrong thing
--- FAIL: TestCapture_UnusableExport_IsDiagnosable/not_JSON_at_all      (same)
--- FAIL: TestCapture_UnusableExport_IsDiagnosable/file_does_not_exist  (same)
--- FAIL: TestTimingIsASpanNotFileOrder
    start_time = "2026-09-10T22:00:15Z", want the earliest event 2026-09-10T22:00:00Z
    end_time   = "2026-09-10T22:00:07Z", want the latest event 2026-09-10T22:00:15Z
    total_ms = -8000, a session cannot take negative time
```

GREEN at `c192f6f`. Reverting `profiler/claude_code.go` to its `7cb32e2` content
(copied in, no `checkout`/`stash`) reproduces exactly those 11; copying the fixed
file back is green again.

**Confirmed on the CLI surface, not only in unit tests.** An export using Claude
Code's *real* attribute name (`type` rather than `token_type`) — the shape that
produced `present` with all-zero counts at `7cb32e2`:

```
$ profiler probe --harness claude_code --otel-file unknown-type.json
  "capabilities": { …, "tokens": "none", … }
$ profiler capture … --otel-file unknown-type.json     # tokens
  { "state": "unknown",
    "reason": "no readable claude_code.token.usage metric in OTel export: none
               carried both a recognised token_type and a numeric value" }

$ profiler capture … --otel-file malformed.json        # tokens/tool_calls/timing
  { "state": "error",
    "reason": "failed to parse OTel export JSON: unexpected end of JSON input" }
```

**W7.** RED with the `7cb32e2` binary: `capture --harness claude_code … --export-file <path>`
exits **0** and prints a full all-unknown profile, the supplied path silently dropped.
GREEN: `--export-file is not read by the claude_code adapter; supply an OTel export with --otel-file`,
exit **1**. Neutering the guard to `if false` puts
`TestCaptureFlagError_RefusesAnExportFileNoAdapterReads` back to RED with
"--export-file accepted by an adapter that cannot read it".

**W16 "without validating anything".** With `skill-validator` masked from PATH:

```
$ check-frontmatter.sh skills/skill-audit          → frontmatter OK            exit 0
$ check-frontmatter.sh <same skill, license line removed>
                                                   → POLICY FAIL [PL001]: missing
                                                     license (house policy)   exit 2
```

The license gate does run, so "without validating anything" was overstated; the
README now says spec validation is skipped and only the license gate remains.

## W15 — verified live with `claude plugin install` (CLI 2.1.221)

RED, both commands exactly as the README had them:

```
$ claude plugins install Okja-Engineering/skill-architect
Installing plugin "Okja-Engineering/skill-architect"...✘ Failed to install plugin
"Okja-Engineering/skill-architect": Plugin "Okja-Engineering/skill-architect" not
found in any configured marketplace

$ claude plugins install .
Installing plugin "."...✘ Failed to install plugin ".": Plugin "." not found in any
configured marketplace
```

(`plugins` is a valid alias for `plugin`; the argument form was the problem —
`install` takes `name[@marketplace]` resolved against configured marketplaces.)

GREEN after adding `.claude-plugin/marketplace.json`, run from the worktree:

```
$ claude plugin validate .
Validating marketplace manifest: …/.claude-plugin/marketplace.json
⚠ Found 1 warning:
  ❯ plugins[0] plugin.json → author: No author information provided…
✔ Validation passed with warnings

$ claude plugin marketplace add ./
Adding marketplace…✔ Successfully added marketplace: skill-architect (declared in user settings)

$ claude plugin install skill-architect@skill-architect
Installing plugin "skill-architect@skill-architect"...✔ Successfully installed plugin:
skill-architect@skill-architect (scope: user)

$ claude plugin list
  ❯ skill-architect@skill-architect
    Version: 0.4.1   Scope: user   Status: ✔ enabled
```

Then uninstalled and the marketplace removed again, so the local machine is back
to where it started (`claude plugin uninstall skill-architect@skill-architect`,
`claude plugin marketplace remove skill-architect` — both reported success).

Two honest caveats, both recorded rather than papered over:

1. `claude plugin marketplace add Okja-Engineering/skill-architect` (the remote
   route the README documents first) **fails today** — `Marketplace file not
   found at …/Okja-Engineering-skill-architect/.claude-plugin/marketplace.json` —
   because `main` does not have the manifest yet. It starts working the moment
   this PR merges. The local-checkout route is verified end to end above.
2. `claude plugin marketplace add .` is rejected (`Invalid marketplace source
   format`); `./` is accepted. The README says so.

## Verification at 527ba4e

```
cd profiler && go build ./... && go vet ./... && gofmt -l . && go test -race ./...
ok  	github.com/Okja-Engineering/skill-architect/profiler      1.506s
ok  	github.com/Okja-Engineering/skill-architect/profiler/cmd  1.309s
(go clean -testcache first; gofmt -l and go vet both silent)

tests/test_skill.sh  → 22 passed, 0 failed
tests/test_walk.sh   → 19 passed, 0 failed
tests/test_f01.sh    → 32 passed, 0 failed
tests/test_f02.sh    → 58 passed, 0 failed      (131 total, unchanged from base)
```

22 Go test functions (20 profiler + 2 cmd), 42 cases counting subtests, up from
17/15 at `7cb32e2`. Working tree clean; nothing untracked left behind.

## Found, not fixed — round 1

The build's "Found, not fixed" list above still stands except items 3 and 4,
which W7 and W6 fixed. New and carried-over:

1. **W17's omissions, now recorded.** `7cb32e2` also edited AC4, AC5 and the spec
   preamble, and the build's found-not-fixed list omitted `profiler/claude_code.go:94`
   (the attribution reason string), `docs/profiler-spec.md:224`, `RELEASE_NOTES.md:20`
   and the `profiler/types.go` `MetricResult` comments. All four are fixed in this
   round (W12, W13).
2. **Deferred to 0.5.0, as routed:** W4 (`ToolCallEntry.Success` conflates the
   permission decision with the execution outcome — a schema-meaning change; now
   documented in the spec and in the code where the value is set); `go vet`,
   `-race` and `gofmt` are still not run in CI; the `curl` and `npm` installs in
   CI are still unpinned (`skill-validator` is pinned to v1.6.1, `skillscore` is
   not); and the two silent-failure scripts the build recorded
   (`check-structure.sh --json` and `check-frontmatter.sh` swallowing exit 127).
3. **The Devin install rows are unverified.** `devin plugins install
   Okja-Engineering/skill-architect` and `devin plugins install .` are the same
   shorthand shape that turned out to be wrong for Claude Code, and no Devin CLI
   is available here to test them. Left as written; worth checking before the
   next release claims them.
4. **`CaptureOpts.APIKey` is still never read** (`profiler/types.go:62`), like
   the `OtelEndpoint` that was deleted. Kept because it is the documented auth
   surface for the future Devin adapter, and because `ExportFile` beside it is
   now protected by a loud refusal rather than a silent no-op.
5. **`skills/skill-rewrite/scripts/draft-rewrite.sh:49`** still comments
   "Resolve skill-audit scripts relative to this script" when it resolves
   relative to the script's parent. Untouched by this branch, so out of scope;
   the README now states the correct level.

## ROOT A / mandate item 19 — deliberately untouched

Per the dispatch and the mandate's item 19, nothing here changes attribute
names, the bespoke `{"metrics":[…],"logs":[…]}` envelope or its documentation,
or the OTel file-exporter claims. Specifically left as they are, pending the
senior-implementer's OTLP work:

- `docs/profiler-spec.md:203` — "**Telemetry surface:** OTel export via
  `CLAUDE_CODE_ENABLE_TELEMETRY=1` + `OTEL_*` env vars", the setup that cannot
  by itself produce an `--otel-file`.
- `RELEASE_NOTES.md:24` — "token counts (including reasoning tokens)", and the
  `reasoning` / `reasoning_output` cases in `extractTokenCounts`, kept so
  `TestCapture_ClaudeCode_WithOtelData` (which asserts 350 reasoning tokens)
  stays green.
- `profiler/claude_code.go:165-182` — the envelope structs, and the `token_type`
  / `cache_read` / `cache_creation` / `decision` attribute names in the
  extractors.

Two things from this round that help item 19 rather than collide with it: the
adapter now has exactly one predicate per signal and one place that owns file
resolution and failure classification (item 19's DoD asks for both), and a
realistic export using the real attribute name `type` now honestly reports
`unknown` with a reason instead of `present` with zeros — so the gap item 19
closes is visible rather than hidden.

## Disagreements with the worklist

None on the verdicts: W1–W18 were all REAL at `7cb32e2` and all reproduced
before being fixed. Three notes on direction, where I took something other than
the literal prescription:

1. **W1/W2 were fixed by deletion, not alignment.** The worklist asks that the
   probe predicate and the capture predicate be "the same predicate". Making two
   predicates equal leaves two predicates to drift apart again; the repair leaves
   one, and derives the capability report from it. This is why the diff to
   `claude_code.go` is larger than the findings suggested (374 lines touched) —
   named here rather than done quietly.
2. **Three extra defects of the same class were fixed with W1**, though the
   worklist named only the token case: `tool_decision` events with no
   `tool_name`, and `api_request` events with no timestamp, both produced
   `present` with an empty value the same way. Fixing the named instance and
   leaving its two siblings would have been the bolt-on.
3. **`CapabilityReport.AnySource` was deleted**, which no W item asked for. It
   was added on this branch for the gating shape the repair removes; its doc
   comment described a decision Capture no longer makes. Leaving it would leave
   a documented helper whose documentation is false — the same defect class as
   Root 3.

---

# Item 19 — OTLP parser (senior implementer, 2026-09-13)

**Head after this pass:** `82ae271d6c21fdc001f3ecad540c00fe46f0af26`, pushed to
`release/0.4.1` (PR #2 updated; no new PR, no merge). Based on `527ba4e`, which
was still the branch head — no rebase was needed.

| Commit | Subject |
|---|---|
| `ade5324` | `test(profiler): pin the adapter against real OTLP/JSON exports` **[RED]** |
| `63a3d3b` | `fix(profiler): parse OTLP/JSON, not an envelope nothing emits` **[GREEN]** |
| `a7f3210` | `docs: describe the OTel capture route and the format the adapter reads` |
| `82ae271` | `docs(changelog): record the OTLP parser and correct three stale claims` |

C1 and C2 were committed separately and pushed together, as the plan requires.

## What was built

`profiler/otlp.go` (new, 590 lines) is the format layer: envelope structs with
`json.RawMessage` at every leaf the OTLP spec lets vary, the BOM/prelude strip
and top-level classifier, the streaming batch reader, typed failure reasons, the
tolerant leaf readers, the two walks, and the series-keyed token accumulator.
`profiler/claude_code.go` keeps the adapter and the three extractors and speaks
only `otlpMetric` / `otlpLogRecord`. `claudeCodeOtelExport`, `otelMetric`,
`otelLog`, `toInt`, `getString` and `parseTime` are deleted outright — no shim,
no second accepted input format. `resolve()` / `otelSignals` / `capabilityReport()`
/ `sourceOf()` are untouched in shape: this swapped what `resolve` decodes and
what the extractors walk, exactly as §1 scoped it.

28 fixtures under `profiler/testdata/otlp/` with a README recording, per file,
what it pins and which fields are [OBSERVED] vs constructed.

## RED (C1 against the bespoke parser, at `ade5324`)

`go test ./...` — 66 failing tests/subtests, per fixture, for the reasons the
plan predicted. Representative lines from the recorded run:

```
--- FAIL: TestCaptureDeliversEverySignalProbeAdvertises/every_signal
    tokens: capture state = "error" (reason "failed to parse OTel export JSON:
    invalid character '{' after top-level value"), want "present"      [full_export.ndjson]
--- FAIL: .../tokens_only
    tokens: state "unknown" (reason "no claude_code.token.usage metric found in OTel export")
--- FAIL: .../the_envelope_the_adapter_used_to_invent
    tokens: nothing readable in the export, probe advertised "otel"
    tokens: state = "present", want "unknown"                          [bespoke_envelope.json]
--- FAIL: TestCapture_ReasonNamesWhatTheExportActuallyCarried/empty.json/tokens
    reason = "failed to parse OTel export JSON: unexpected end of JSON input"
    want   = "OTel export file is empty"
--- FAIL: .../truncated_final_line.ndjson/tokens
    reason = "failed to parse OTel export JSON: invalid character '{' after top-level value"
    want it to name "in batch 4", "malformed JSON at byte", the mid-write advice
--- FAIL: .../top_level_array.json/tool_calls
    reason = "... json: cannot unmarshal array into Go value of type profiler.claudeCodeOtelExport"
    must not contain "profiler."
--- FAIL: .../unreadable_tool_events.json/tool_calls
    reason = "no claude_code.tool_decision log events found in OTel export"
    want   = "no tool call outcomes in OTel export: 1 ... carried no tool_name; 1 ...
              carried no readable success value; 1 ... carried no recognised decision"
--- FAIL: TestCapture_ClaudeCode_RefusesAnExportFileItCannotRead
    --export-file accepted by an adapter that cannot read it
```

`bespoke_envelope.json` was **`present` with values** at C1 — the inverted truth
this release exists to fix — and is `unknown` naming the expected format at C2.

**Two of the plan's RED predictions were wrong**, both harmlessly:
`type_mismatch.json` was `unknown` / "no …metric found", not an `error` leaking
`profiler.claudeCodeOtelExport` (the bespoke struct simply ignores an unknown
`resourceMetrics` key), and a bare top-level `null` was `unknown` for the same
reason. Every other per-fixture prediction held.

## GREEN, and RED again on revert

At `63a3d3b` the whole suite is green. Reverting C2's code in a **scratch copy**
(`git show ade5324:` for `claude_code.go`, `types.go`, `cmd/main.go`; `otlp.go`
removed; `otlp_test.go` back to its C1 content) and re-running gives 66 failures
again. The worktree was not touched. Separately, replacing `addSaturating` with
`a + b` in a second scratch copy makes the saturation unit fail with
`input total = -9223372036854774809`, so that test is not vacuous.

## Verification at the pushed head

```
cd profiler && go build ./... && go vet ./... && gofmt -l . && go test -race -count=1 ./...
  (build, vet clean; gofmt lists nothing)
  ok  github.com/Okja-Engineering/skill-architect/profiler        1.365s
  ok  github.com/Okja-Engineering/skill-architect/profiler/cmd    1.516s

tests/test_skill.sh  22 passed, 0 failed
tests/test_walk.sh   19 passed, 0 failed
tests/test_f01.sh    32 passed, 0 failed
tests/test_f02.sh    58 passed, 0 failed        (131 total, unchanged)

BIN=$(mktemp -d)/profiler && go build -o "$BIN" ./cmd
"$BIN" capture --harness claude_code --session s1 --snapshot sha1 \
  --skill-dir ../skills/skill-audit --otel-file testdata/otlp/full_export.ndjson
```

```json
"capabilities": { "timing": "otel", "tokens": "otel", "tool_calls": "otel",
                  "attribution": "none", "skill_activation": "none" },
"tokens":    { "state": "present", "source": "otel",
               "value": { "input": 1523, "output": 412,
                          "cache_read": 20480, "cache_creation": 3072 } },
"tool_calls":{ "state": "present", "source": "otel", "value": [
               { "name": "Bash", "timestamp": "2026-09-13T20:49:55.3Z",  "success": false },
               { "name": "Read", "timestamp": "2026-09-13T20:49:55.46Z", "success": true } ] },
"timing":    { "state": "present", "source": "otel",
               "value": { "start_time": "2026-09-13T20:49:55.1Z",
                          "end_time": "2026-09-13T20:49:56.272Z", "total_ms": 1172 } }
```

No `reasoning` key, no schema-key changes. **Recount at the C4 head** (§6): test
**functions** 33 — 31 in the profiler package (`otlp_test.go` 4 +
`profiler_test.go` 27), 2 in the CLI; cross-checked with
`go test ./... -list '.*'` → 33. **Subtests** 65
(`go test ./... -v | grep -c '^=== RUN .*/'`). Both stated separately and
labelled in `CHANGELOG.md` and `RELEASE_NOTES.md`; no "N cases" figure remains.

## Documentation — §7, item by item

| § | Site | Done in | Note |
|---|---|---|---|
| 1 | README tool_calls row | `a7f3210` | now `tool_result` + rejected `tool_decision` |
| 2 | README tokens row | `a7f3210` | names `type` and its four camelCase values, `asDouble`/`asInt` |
| 3 | README timing row | — | **no change**, as planned |
| 4 | README error paragraph | `a7f3210` | `error` = could not be read as OTLP/JSON; `unknown` = parsed, no telemetry |
| 5 | README skill sentences | `a7f3210` | says the adapter does not read `skill.name`; **no redaction claim** (REAL-1) |
| 6 | README "Capturing an OTel export" | `a7f3210` | routes (a) and (b), console ruled out, truncation, PII, 0.5.0 note |
| 7 | spec "Input format" | `a7f3210` | envelope, temporality, event identification, skipped-record rule, timing-span caveat |
| 8 | spec probe logic | `a7f3210` | all three bullets + the skill sentence in step 2 |
| 9 | spec capture logic 2–3 | `a7f3210` | D5 semantics + rule 2; the R-F3 sentence is folded into **step 1**, not a new step — the refusal happens before resolution. Step 4 unchanged |
| 10 | spec steps 6/7 reasons | `a7f3210` | verbatim new reasons; steps renumbered 6/7 after the step-5 removal |
| 11 | `claude_code.go` comment + two reasons | **`63a3d3b`** | had to land with the code: C1's tests pin the reason strings |
| 12 | spec Fallback | `a7f3210` | `error` set extended, no-envelope case added; AC2 unchanged |
| 13 | spec AC3 | `a7f3210` | rewritten in full, incl. gauge and unreadable temporality |
| 14 | spec AC10/AC11/AC12 | `a7f3210` | AC10 amended (R-F6), AC11 extended, AC12 added |
| 15 | CHANGELOG + RELEASE_NOTES counts | `82ae271` | 33 functions / 65 subtests, two labelled numbers |
| 16 | RELEASE_NOTES "(including reasoning tokens)" | `82ae271` | deleted |
| 17 | CHANGELOG 0.4.1 Fixed + RELEASE_NOTES bullet | `82ae271` | new bullet + the history note (nit 11); `CHANGELOG:38` and `RELEASE_NOTES:8` unchanged |
| 18 | `types.go` `Reasoning` comment | **`63a3d3b`** | on the `Reasoning` field, where D4 puts it |
| 19 | spec `State == "unknown"` bullet | `a7f3210` | second quoted reason updated with item 11 |
| 20 | "verified live" (R-F2) | `82ae271` | names the local route; remote route "works once this release is on `main`" |

## Round-1 residuals

- **R-F2** — closed in C4 (item 20 above).
- **R-F3** — closed in C2. `ClaudeCodeAdapter.Capture` returns the refusal when
  `opts.ExportFile != ""`, per the chief's override. To keep one owner for the
  sentence, `ExportFileUnsupportedError(harness)` lives in `types.go` beside
  `CaptureOpts`, and the CLI's `captureFlagError` now calls it — the CLI guard
  stays as the earlier, cheaper copy. Pinned by
  `TestCapture_ClaudeCode_RefusesAnExportFileItCannotRead`; the two CLI tests
  still pass unchanged.
- **R-F4** — closed in C2: the dead `ExportFile: *exportFile` is gone.
- **R-F5** — closed in C4 (recount above).
- **R-F6** — closed in C3 (AC10) and pinned by `timing_only.json` +
  `TestTiming_SingleRequestIsAZeroLengthSpan`: one `api_request` → `total_ms: 0`,
  `present`.
- **R-F7** — closed in C2 with REAL-1's wording: attribution is "Claude Code
  telemetry carries no output-to-skill mapping".
- **R-obs** — not addressed; it was recorded as an observation, not allocated by
  the plan.

## Deviations from plan v3, and why

1. **Override (b) from the chief replaces §2.4/nit 6.** A data point whose
   `aggregationTemporality` reads as neither 1 nor 2 (absent, `0`, or
   unparseable) is refused, counted, and named in the tokens reason if the
   signal ends `unknown` — not treated as cumulative. Pinned by a new fixture
   row, `unreadable_temporality.json`, and asserted verbatim:
   `"no readable claude_code.token.usage metric in OTel export: 2 data points
   declared an aggregationTemporality that is neither 1 (delta) nor 2 (cumulative)"`.
2. **D8 gained a fifth counter and clause.** A `tool_decision` recording a
   reject with no `tool_name` matched none of the plan's four counters, so a
   file containing only that record would have produced
   `"no tool call outcomes in OTel export: "` with an empty clause list — the
   exact class of hole D8 exists to close. Added `unnamedRejects` with
   `"%d claude_code.tool_decision events recorded a reject with no tool_name"`,
   appended last so the four planned clause strings and their order are
   unchanged. Pinned by `unnamed_reject.json`.
3. **Two fixtures beyond the plan's list.** `partial_no_tokens.json`, because
   round 1's partial-export regression test (mandate item 1) needed an OTLP
   input to keep its meaning; `untimed_api_request.json`, because AC3 names "an
   `api_request` with no parseable `timeUnixNano`" and nothing pinned the
   timing-seen-but-unreadable reason. 28 fixtures total.
4. **The schema-mismatch reason uses `te.Field` only, not `te.Struct`.** §4 says
   "path from `te.Struct`+`te.Field`", but `te.Struct` is the Go struct name
   (`otlpBatch`), which nit 13 forbids printing. `te.Field` is the JSON path
   built from the struct tags, which is what the sentence promises:
   `"value at resourceMetrics in batch 1 is not the expected type"`.
5. **Top-level kind strings carry their article** ("an array", "a string",
   "a boolean", "null", "a number"). §4's `"...is a %s"` with a bare kind would
   have printed "a array"; the fixture row's expected text ("is an array") wins.
6. **The clamp unit landed in C2, not C1.** A direct unit on
   `tokenAccumulator.reduce` cannot compile before the type exists, and a
   compile failure in C1 would have destroyed the per-fixture RED evidence C1
   exists to produce. The C1 half of `otlp_test.go` (BOM, concatenation,
   top-level kinds) does compile at `ade5324` and is RED there. Non-vacuity of
   the clamp unit was verified by reverting the saturation in a scratch copy.
7. **`eventName` lives in `claude_code.go`, not `otlp.go`.** The
   `claude_code.` prefix is a harness fact, and `otlp.go`'s contract is that it
   knows none. Its format half, `bodyName` (object / bare string / absent), is
   in `otlp.go`.
8. **`malformedJSON` puts a full stop before the mid-write advice.** The plan's
   literal concatenation read "…unexpected end of JSON input No data from…".
9. The token reason is now **composed from clauses** the same way D8's is,
   because deviation 1 gave tokens a second independent defect to name. The
   single-defect wording is byte-identical to the plan's sentence.

## Not done, and not claimed

- **Route (a) was executed live in round-2 review** (D6, chief of staff's
  worklist, 2026-09-13). The docs hunter ran the README's receiver script and
  env block against Claude Code v2.1.221 and pointed the adapter at the
  resulting NDJSON; it read correctly. README:115-116 ("the route the envelopes
  this adapter's test fixtures are modelled on were observed through") and
  README:178 ("route (a) is the one that was exercised") are therefore true of
  head (`a4fb247`; they were :114-115 and :177 at `9b39eb0`, and the round-2
  docs pass added a line above them), and the implementer's original "no live
  capture" statement is
  superseded for route (a). Item 19's DoD is met by that run plus the fixtures
  modelled on the researcher's observed envelopes.
- **Still no live end-to-end capture from an authenticated session by this
  worker.** The round-2 repair was verified against fixtures, the CLI, and the
  live NDJSON the docs hunter produced — not against a session it captured
  itself.
- **No end-to-end collector run.** `otelcol-contrib` is not on `PATH` and is not
  installed; installing it would have been a `brew install`, which the rule for
  route (b) does not authorise. The README's collector config is marked in the
  README itself as derived from the `fileexporter` README and source, with route
  (a) named as the route that was exercised.
- Nothing from §9's out-of-scope list was built: no bundled receiver or shipped
  script, no `skill.name` promotion, no `duration_ms` / `active_time.total`
  surfacing, no count of skipped records, no key renames.

## Found, not fixed — item 19

- **Cumulative temporality is still untested against real output** (plan R1).
  The branch exists on spec reading plus two fixtures; delta is the documented
  default and the only thing ever observed.
- **`tool_result` / `tool_decision` attributes remain [DOCS], not [OBSERVED]**
  (plan R2). `success` as a string is documented, not seen on the wire; the
  adapter reads both string and boolean forms, and a rename would fail loudly as
  `unknown` with a counted reason rather than silently as `false`.
- **`ToolCallEntry.Success` changes meaning between a 0.4.0 and a 0.4.1 profile**
  with no key change to signal it (plan R3, decided in the mandate). Carried by
  prose and by `capability.adapter_version`.
- **A truncated final line still costs the whole capture** (plan R4/I1, and the
  architect's own §10 disagreement). Planned as decided; if it bites, the repair
  is an explicit `--allow-truncated`, not a softer classifier. 0.5.0.

# Round 2 — root repair (bug-fixer, 2026-09-13)

**Worklist:** `worklist-round2.md` (four hunters on `82ae271`, verdict NOT CLEAN).
**Head after repair:** `9b39eb0a518356811be695cbf0e76ae065aed46a` — 12 commits on top of `82ae271`.
**Scope taken:** X1, X2, X4–X11, T1–T8, D1–D7. X12 and the other tracked items recorded only.
**Merge hold stands.** Nothing was merged; the branch is pushed to `release/0.4.1` and PR #2 updated.

Every item in the worklist was verified against the code at `82ae271` before it
was fixed. One turned out to be a finding about a line outside the diff whose
original wording was true when written (D4's skill-audit score) and one was
about a decision the fix itself created (the exit-code collision) — both are in
**Disagreements and additions** below.

## Commits (12, oldest first)

| SHA | Commit |
|---|---|
| `a69947d` | test(profiler): pin what may reach a profile as a token count **[RED]** |
| `e6e26bd` | fix(profiler): a value reaches a profile only if it can be a count **[GREEN]** |
| `ce7b9f5` | test(profiler): pin every reason to the input that produced it **[RED]** |
| `57d0e18` | fix(profiler): say what the export did, not what it did not **[GREEN]** |
| `e6b3013` | test(cmd): a capture that read nothing must not exit 0 **[RED]** |
| `39b0f0d` | fix(cmd): exit 2 when a capture read nothing and something failed **[GREEN]** |
| `98eb0f6` | refactor(profiler): keep harness vocabulary out of the format layer |
| `fa1aab5` | test: assert the behaviour, and count the shapes nobody asserted |
| `e9fda9b` | ci: run the checks the release actually claims were run |
| `77732eb` | test(profiler): assert the series key is injective, not that nine pairs differ |
| `e7ab209` | fix(cmd): a usage error exits 1, so exit 2 means the capture **[GREEN]** |
| `9b39eb0` | docs: re-derive every sentence beside the code from the code |

## Per-item disposition

All cites are `file:line` at head `9b39eb0`.

| Item | Disposition | Commit | Cite at head | Evidence |
|---|---|---|---|---|
| **X1** | REAL, fixed | `a69947d`→`e6e26bd` | `profiler/otlp.go:610-625` (`count`), `:403-413` (`jsonFloat64`), `:591` (`maxCountPlusOne`) | RED: `tokens value = {"input":9223372036854775807,…}` on arm64, `{"input":-9223372036854775808,…}` on amd64, same file. GREEN: both refuse, byte-identical profiles. |
| **X2** | REAL, fixed | `a69947d`→`e6e26bd`, strengthened `77732eb` | `profiler/otlp.go:507-514` (`seriesKey`), `:518-521` (`lengthPrefixed`), `:535-562` (`identity`), `:564-570` (`compactJSON`), `:97-107` (composite kinds decoded) | RED: 11 collisions/splits. GREEN: injective over a 31-set corpus. |
| **X4** | REAL, fixed | `a69947d`→`e6e26bd` | `profiler/types.go:106-117` (`TokenCounts`), `:122` (`Count`), `profiler/claude_code.go:358-374` (`tokenCounts`) | RED: `{"input":0,"output":0,"cache_read":20480}`. GREEN: `{"cache_read":20480}`. Read zero: `{"input":0,"cache_creation":0}`. |
| **X5** | REAL, fixed | `ce7b9f5`→`57d0e18` | `profiler/otlp.go:52-55` (pointer envelope), `:75-80` (`entries`), `:143-150` (`hasEnvelope`) | RED: `{"resourceMetrics":[]}` → "not OTLP/JSON". GREEN: `no claude_code.token.usage metric found in OTel export` and the two matching per-signal reasons. |
| **X6** | REAL, fixed | `ce7b9f5`→`57d0e18` | `profiler/otlp.go:637-647` (`temporalityState`), `:650-670` (`temporality`), `profiler/claude_code.go:255-275` (the three counters) | RED: absent → "declared an aggregationTemporality that is neither 1 nor 2". GREEN: "1 data point carried no aggregationTemporality". New fixture `absent_temporality.json`. |
| **X7** | REAL, fixed | `ce7b9f5`→`57d0e18` | `profiler/otlp.go:197-205` (`countingReader`), `:267-295` (`decodeFailure`) | RED: `malformed.json` (87 bytes) → byte 0; syntax errors named the byte *after* the offending one. GREEN: byte 87; offsets index the offending byte through a BOM/whitespace prelude. |
| **X8** | REAL, fixed | `ce7b9f5`→`57d0e18` | `profiler/claude_code.go:345-350` (`quantity`), `:463-479` (`toolCallCounters.reason`), `:315-341` (`tokenPointCounters` and its reason) | RED: "1 claude_code.tool_result events". GREEN: "1 claude_code.tool_result event". Six clauses, all through `quantity`. |
| **X9** | REAL, fixed | `e6b3013`→`39b0f0d` | `profiler/cmd/main.go:129-143` (`captureExitCode`), `profiler/types.go:218-226` (`SignalStates`) | RED: exit 0 on `malformed.json` and on a missing file. GREEN: exit 2; exit 0 for unconfigured, for `no_envelope.json`, and for any export that yielded a reading. |
| **X10** | REAL, fixed | `98eb0f6` | `profiler/types.go:62-75` (`CaptureOpts` doc and both fields) | `APIKey` and `ExportFile` both documented as the 0.5.0 adapters' input contract; neither deleted. |
| **X11** | REAL, fixed | `98eb0f6` | `profiler/otlp.go:672-712` (`counterSeries`/`counterAccumulator`/`label`), `:302-304` (`fileReadError`) | Three copies of "failed to read OTel export file" → one owner, called from `skipPrelude` (`:220`), `decodeFailure` (`:294`) and `ClaudeCodeAdapter.resolve` (`claude_code.go:93`). |
| **X12** | Tracked, not fixed | — | — | Whole-export slurp (~6× RSS). Recorded under **Found, not fixed** below for 0.5.0. |
| **T1** | REAL, fixed | `ce7b9f5` | `profiler/testdata/otlp/unrecognised_token_type.json`, `profiler_test.go:788-790` (verbatim reason), `:593` (contract case) | Verbatim: `no readable claude_code.token.usage metric in OTel export: 2 data points carried no recognised type attribute (input/output/cacheRead/cacheCreation)`. Mutation 26 (`isTokenType` → `s != ""`) now fails. |
| **T2** | REAL, fixed | `fa1aab5`, `e6e26bd` | `otlp_test.go:430-505` (cumulative tie-breaks), `:507-530` (mixed temporality), `profiler_test.go:1225-1250` (log record shapes), `:1259-1262` (asDouble before asInt), `:837-843` (enum names) | Ten shapes added: unrecognised type, uncountable values, absent temporality, enum-name temporality, asDouble-beside-asInt, body-vs-`event.name`, multi-resource/scope logs, two untimed calls in file order, four cumulative tie-breaks, mixed temporality. Five new fixtures. |
| **T3** | REAL, fixed | `fa1aab5` | `profiler_test.go:383-407` (`TestProfileSchemaField`), `:409-421` (`TestAdapterVersionIsThisRelease`), `tests/test_skill.sh:34-58` | `TestProfileSchemaField` holds the literal; `TestAdapterVersionIsThisRelease` holds `0.4.1` in Go and in `capability.adapter_version`; `test_skill.sh` asserts `.claude-plugin/marketplace.json` (X3, the fifth surface) and `AdapterVersion` (the sixth). |
| **T4** | REAL, fixed | `fa1aab5` | `profiler_test.go:286-293` (the two spec-quoted constants), `:255-266` (fallback asserted), `:297-321` (activation asserted over five inputs) | Both reasons asserted whole, not by substring. Mutations 35 and 36 (reword either) fail. |
| **T5** | REAL, fixed | `ce7b9f5` | `otlp_test.go:352-366` (`reasonOffset`), `:369-402`, `:404-421` | Offsets asserted numerically, against the file: `content[at] == ']'` for four prelude variants, and `at == len(file)` for the two truncated fixtures. |
| **T6** | REAL, fixed | `98eb0f6` | `profiler/otlp.go:771-787` (`clampToInt`/`clampToRange`), `otlp_test.go:139-172` | Bound made injectable. Deleting the clamp: old unit green, new unit fails on four inputs. |
| **T7** | REAL, fixed | `e9fda9b` | `.github/workflows/ci.yml:38-53` | Three steps: `gofmt -l` failing on output, `go build && go vet`, `go test -race -count=1`. Same four commands as the local gate. |
| **T8** | REAL, fixed | `fa1aab5` | `profiler_test.go:133-151` | Renamed `TestCapabilityReport_ClaudeCode_NoOtlpEnvelope`; walks all five signals and asserts the report covers five. Coverage denominator added at `:607-633` (`TestEveryFixtureIsAContractCase`). |
| **D1** | REAL, fixed | — (PR metadata) | PR #2 body | Rewritten from head via `gh pr edit 2 --body-file`. |
| **D2** | REAL, fixed | `9b39eb0` | `README.md:64`, `:65`, `:73-83`; `docs/profiler-spec.md:61-66`, `:68`, `:108`, `:155-165`, `:208-210`, `:216-220`, `:240`; `profiler/types.go:164-167`, `:174-178`, `:187-190`; `profiler/otlp.go:439-446`; `profiler/testdata/otlp/README.md:4-12`, `:57`, `:70` | Thirteen sentences re-derived; see the commit message for the list. |
| **D3** | REAL, fixed | `9b39eb0` | `README.md:109-118` (recipe, flag removed), `:188-199` (privacy note) | `OTEL_LOG_TOOL_DETAILS=1` out of the recipe, named once as a thing to leave unset. Scrub list opened from three attributes to six, stated as a floor. |
| **D4** | REAL, fixed | `9b39eb0` | `CHANGELOG.md:25`, `:42`, `:44`, `:50`, `:53`; `RELEASE_NOTES.md:14`, `:25-26` | `AnySource` clause dropped; per-signal probe detection re-attributed; counts recounted (50/128/134); 0.3.1 score recorded under 0.4.1 rather than rewritten. |
| **D5** | REAL, fixed | `9b39eb0` | `README.md:246-252` (table and preamble), `:265-269` (local checkout) | Three install rows marked "not verified in this release", with the Claude Code route marked verified beside them. |
| **D6** | REAL, fixed | — (this file) | **Not done, and not claimed**, above | Route (a)'s live run recorded; the no-collector-run statement kept. |
| **D7** | REAL, fixed | — (this file) | `status.md:56` | Per-item table header now says the cites are at the build commit `7cb32e2`, not at head. |

## RED → GREEN → revert-RED, per fix

The revert-RED step is the mutation sweep below: every decided behaviour was
inverted in a scratch copy under
`/private/tmp/claude-501/…/scratchpad/mutruns` and the suite re-run.

**X1 — the finding, reproduced on two architectures at `82ae271`:**

```
=== native (arm64) ===
    profiler_test.go:1037: tokens value = {"input":9223372036854775807,"output":-500,
      "cache_read":9223372036854775807,"cache_creation":-1}, want none
=== amd64 (cross-compiled) ===
    profiler_test.go:1037: tokens value = {"input":-9223372036854775808,"output":-500,
      "cache_read":-9223372036854775808,"cache_creation":-1}, want none
```

At head, the same file under binaries built for both architectures:

```
=== arm64: value_not_a_count.json ===
{"state":"unknown","reason":"no readable claude_code.token.usage metric in OTel export:
 4 data points carried a value that is not a token count: a count is a whole number
 from 0 to 9223372036854775807"}
=== amd64: value_not_a_count.json ===
{"state":"unknown","reason":"no readable claude_code.token.usage metric in OTel export:
 4 data points carried a value that is not a token count: a count is a whole number
 from 0 to 9223372036854775807"}
```

and on the happy path, `{"input":1523,"output":412,"cache_read":20480,"cache_creation":3072}`
from both.

**X2 — RED at `82ae271`** (11 of 12 corpus cases; excerpt):

```
--- FAIL: TestOTLP_SeriesKeyIsTheAttributeSetNotItsText
    [{"key":"m","value":{}}] and [{"key":"m","value":{"stringValue":""}}] share the
      series key "m\x01"
    [… arrayValue "Read"] and [… arrayValue "Bash"] share the series key "tool.names\x01"
    [{"key":"m","value":{"stringValue":"5"}}] and [{"key":"m","value":{"intValue":5}}]
      share the series key "m\x015"
    [{"key":"m","value":{"intValue":5}}] keys as "m\x015" and [… "intValue":"5"] as
      "m\x01\"5\"" — one series would split in two
```

**X4 — RED at `82ae271`:**

```
--- FAIL: TestTokens_AnUnreadCountIsAbsentAndAReadZeroIsZero
    a_cache-only_export…: tokens JSON = {"input":0,"output":0,"cache_read":20480}
      want {"cache_read":20480}
    a_count_read_as_zero…: tokens JSON = {"input":0,"output":0}
      want {"input":0,"cache_creation":0}
    an_unreadable_point…: tokens JSON = {"input":1523,"output":0}  want {"input":1523}
```

**X5/X6/X7/X8 — RED at `ce7b9f5`** (excerpt):

```
--- FAIL: TestOTLP_AnOffsetNamesTheOffendingByteInTheFile
    no_prelude: reason names byte 20, which is "}"; the byte the decoder could not
      accept is the "]" at 19
    a_byte-order_mark_and_leading_whitespace: reason names byte 26, which is "}"; … at 25
--- FAIL: TestOTLP_AnUnexpectedEndOfFileNamesTheFileLength
    malformed.json: reason names byte 0 for a 87-byte file that ends mid-object
    truncated_final_line.ndjson: reason names byte 642 for a 714-byte file
--- FAIL: TestCapture_ReasonNamesWhatTheExportActuallyCarried
    empty_envelope.json/tokens: reason = "OTel export is not OTLP/JSON: no resourceMetrics
      or resourceLogs found. …"  want "no claude_code.token.usage metric found in OTel export"
    absent_temporality.json/tokens: reason = "… 1 data point declared an
      aggregationTemporality that is neither 1 (delta) nor 2 (cumulative)"
      want "… 1 data point carried no aggregationTemporality"
    unreadable_tool_events.json/tool_calls: "1 claude_code.tool_result events carried no
      tool_name; …"  want "1 claude_code.tool_result event carried no tool_name; …"
```

**X9 — RED at `e6b3013`:**

```
--- FAIL: TestCapture_ExitStatusSaysWhetherAnythingWasRead
    an_export_that_cannot_be_read_at_all: exit status = 0, want 2
    an_export_file_that_is_not_there:     exit status = 0, want 2
```

**The exit-code collision — RED at `e7ab209`'s parent:**

```
--- FAIL: TestUsageErrorsExitOneSoThatTwoMeansOneThing
    an_unknown_flag_on_capture: exit status = 2, want 1 — a usage error is not a
      capture that read nothing
```

**GREEN at head, through the CLI:**

```
cache_only      exit=0 :: {"cache_read":20480}
zero_counts     exit=0 :: {"input":0,"cache_creation":0}
malformed       exit=2 :: "malformed JSON at byte 87 in batch 1: unexpected end of JSON input"
missing file    exit=2 :: "failed to read OTel export file: open /nope/nope.json: no such file…"
unconfigured    exit=0 :: "OTel export not configured. Provide an OTel export file via --otel-file…"
empty_envelope  exit=0 :: "no claude_code.token.usage metric found in OTel export"
```

## Mutation sweep — Root 3's own proof

41 decided behaviours inverted one at a time in a scratch copy, suite re-run
each time. **40 caught, 1 survivor.** Driver:
`/private/tmp/claude-501/…/31edaf48…/mutate.py`.

Caught, with the test that caught each (abridged): out-of-range float
assimilated (`TestCaptureDeliversEverySignalProbeAdvertises`); negative `asInt`
accepted (same); truncation instead of rounding
(`TestTokens_OnlyReadableMetricsAreCounted`); `asInt` read before `asDouble`
(`TestTokens_AsDoubleIsReadBeforeAsInt`); NaN/Inf accepted
(`TestOTLP_FloatLeafReadsOnlyRealNumbers`); fractional `jsonInt64`
(`TestOTLP_IntegerLeafRefusesWhatIsNotADecimalInteger`); kind tag dropped,
composite kinds collapsed, length prefixes dropped, sorting dropped (all
`TestOTLP_SeriesKeyIsTheAttributeSetNotItsText`); `hasEnvelope` requiring
entries and absent temporality reported as unreadable (both
`TestCapture_ReasonNamesWhatTheExportActuallyCarried`); the offset `-1` lost,
EOF naming byte 0, the prelude uncounted (the two offset tests); delta and
cumulative swapped (`TestOTLP_ConcatenatedObjectsNeedNoNewline`); three
cumulative tie-breaks and mixed temporality (the two accumulator tests); the
metric and log walks indexing `[0]` (`TestTokens_EveryResourceAndScopeIsWalked`,
`TestToolCalls_TheBodyNamesTheEventAndEveryScopeIsWalked`); clamp removed
(`TestClampToRange_SaturatesAtTheBoundsItIsGiven`); `addSaturating` wrapping
(`TestCounterAccumulator_TotalsSaturateRatherThanWrap`); BOM not stripped
(`TestOTLP_LeadingByteOrderMarkIsStripped`); `isTokenType` weakened to `s != ""`
(the contract test); all counts always set, and `input`/`output` losing
`omitempty` (both caught); `quantity` always plural; the absent-temporality
clause reworded; `event.name` overriding the body; an unreadable outcome
recorded as `false`; an untimed tool call dropped
(`TestToolCalls_EntriesRecordExecutionOutcomes`); timing following file order
(`TestTimingIsASpanNotFileOrder`); the capability report not derived from what
was read (`TestCapabilityReport_ClaudeCode_WithoutOtel`); either spec-quoted
reason reworded; `ProfileSchema` typo; `AdapterVersion` left at `0.4.0`;
`capture` always exiting 0, and exiting 2 too eagerly (both the exit-status
test).

**Survivor (1), and why it is an equivalent mutant rather than a gap:**

- *`identity()` reports an absent attribute value as `""` instead of `"-"`.*
  Every present kind's identity begins with a one-character tag, so the empty
  string is reachable only from an absent value, and with the length prefix an
  absent value keys as `0:` while `{"stringValue":""}` keys as `1:s`.
  Injectivity — the invariant the test asserts — is preserved by the mutation,
  so no test *should* fail. `"-"` is kept because it reads better in a debugger,
  not because anything depends on it. **No test was added for it**: a test that
  pinned `"-"` would pin the encoding rather than the invariant, which is the
  defect the first sweep found in the previous version of this test.

The first sweep had **three** survivors, all in `seriesKey`; two were real gaps
(a string spelling another kind's tag, and a value carrying a separator followed
by a tag) and closed by replacing the pair-by-pair table with the injectivity
corpus in `77732eb`. That replacement is itself the evidence that mutation
testing was not a formality here.

## Verification at head `9b39eb0`

```
$ cd profiler && go build ./... && go vet ./... && gofmt -l . && go test -race -count=1 ./...
=== go build ===   OK
=== go vet ===     OK
=== gofmt -l ===   (no output)
=== go test -race -count=1 ./... ===
ok  	github.com/Okja-Engineering/skill-architect/profiler	1.402s
ok  	github.com/Okja-Engineering/skill-architect/profiler/cmd	2.398s
```

```
$ tests/test_skill.sh && tests/test_walk.sh && tests/test_f01.sh && tests/test_f02.sh
test_skill  25 passed, 0 failed
test_walk   19 passed, 0 failed
test_f01    32 passed, 0 failed
test_f02    58 passed, 0 failed      (134 total, up from 131: three new version-surface assertions)
```

Binaries built into `$(mktemp -d)`, never into the worktree: native, `GOARCH=amd64`
and `GOARCH=arm64`, all three Mach-O, both cross-arch binaries executed.

Counts, recounted at head and written into the changelog and release notes:
~~**50** Go test functions (47 profiler, 3 CLI), **128** subtests~~ — **stale.**
That count was taken before `e7ab209` added `TestUsageErrorsExitOneSoThatTwoMeansOneThing`
and its seven subtests, so it went into the changelog, the release notes and the
PR body one function and seven subtests short. Recounted in round 2 at `a4fb247`
(see "Round 2 — docs residuals" below): **51** Go test functions (47 profiler,
**4** CLI), **135** subtests (108 direct, 27 nested deeper). **134** shell
assertions and **38** OTLP fixtures are unchanged and still correct — and
`TestEveryFixtureIsAContractCase` now makes the fixture directory the coverage
denominator, so a fixture added without a contract case fails the suite.

T7's CI change runs exactly the four commands above, so the claim in this
section is one CI can contradict.

## Disagreements and additions

1. **D4's skill-audit score — the finding is right about the number and wrong
   about the defect.** The worklist reads as though `CHANGELOG:69` /
   `RELEASE_NOTES:39` state something false. Measured with `skillscore --json`
   against three trees: **92.5 (A-) at `9897acc`**, the 0.3.1 release commit;
   **94 (A) at `3e2efd8`**; **94 (A) at head**. So 0.3.1's entry was true when
   written, and `skill-rewrite`'s 89.5 (B+) is still exactly right. Rewriting
   0.3.1 would also have crossed the mandate's item-13 boundary ("do not rewrite
   0.3.1 history"). Recorded instead as a correction under 0.4.1's **Known past
   changes, recorded late**, which is where the mandate already puts this class.
   Neither line is in this PR's diff, so the Root-4 invariant did not require a
   change at all; the note is the honest version of what D4 asked for.

2. **Root 1's fix forced the token reason into Root 2's shape, one commit
   early.** X1 creates a third independent defect a `token.usage` data point can
   have, and the existing reason fused two of them into one sentence ("no data
   point carried both a recognised type attribute … and a numeric asDouble or
   asInt value") that is true of no input carrying only one. Splitting it into
   counter-composed clauses — the shape plan §4's D8 already chose for tool
   calls — landed in `e6e26bd` rather than `57d0e18`. It changes an asserted
   string the plan quotes. Recorded as a deviation from plan §4; the alternative
   was a reason that alleges a defect the export does not have, which is the
   exact class Root 2 exists to close.

3. **The X9 fix created an exit-code collision, found by re-walking the prose
   against the binary.** `flag.ExitOnError` exits **2** on an unrecognised flag.
   Giving "the capture read nothing" the same status would have made 2
   unbranchable — and the capture case is the one a wrapper must act on. Fixed
   in `e7ab209` by moving both flag sets to `ContinueOnError` and having the CLI
   choose: 1 for every usage error, 2 for the capture. This was not in the
   worklist; it is a defect this round's own fix introduced, and it is why the
   Root-4 re-walk is done against the built binary rather than against the
   source.

4. **The worklist's X2 wording ("`intValue` 5 vs \"5\" are two keys") lists a
   *splitting* defect beside the merging ones.** Both are real and both are
   fixed, but they pull in opposite directions, so the test asserts the single
   property that covers both — injectivity over the attribute set — rather than
   a list of pairs. The first mutation sweep proved the distinction matters: the
   pair list passed under two weakened encodings.

5. **Nothing was done for X3 beyond T3's assertion.** X3 is named in the
   worklist only inside T3 ("a fifth version surface with no assert"), and
   `marketplace.json` was already at `0.4.1` at `82ae271`. The gap was the
   missing assertion, which `tests/test_skill.sh` now carries, along with a
   sixth for `AdapterVersion`.

## Found, not fixed — round 2

Carried forward for 0.5.0; none is in this release's scope.

- **X12 — the whole export is slurped into memory** (~6× the file's size in RSS).
  Streaming the decode per batch is the repair; it changes the shape of
  `otlpExport` and every walk over it, which is larger than any bug this round
  fixed.
- **`profiler/cmd` coverage is thin.** The new exit-status tests exercise the
  process end to end, but `cmdProbe` and the marshal paths are still only
  incidentally covered.
- **CI installs are unpinned.** `npm install -g skillscore` takes whatever is
  latest; `skill-validator` is pinned at `v1.6.1`.
- **W4 rename** (`cache_creation` → `cache_write`) remains a breaking schema
  change, deferred by the mandate.
- **Composite attribute values are compacted, not canonicalised**, for series
  identity. Two spellings of the same `arrayValue` that differ by more than
  whitespace would split one series in two. No producer varies its own
  serialisation within a file, so this is a bound on the fix, not a known bug.
- **`tool_result` / `tool_decision` attributes are still [DOCS], not [OBSERVED]**
  — unchanged from item 19's record.

## Round 2 — docs residuals

Docs-only pass over the eight residual claims left after `9b39eb0`. No Go or
shell code changed; `profiler/` and `tests/` are byte-identical to `9b39eb0`.

`9b39eb0` → `2d987c5` (counts and four narrowed claims) → `a4fb247` (the `jq`
row). Pushed to `release/0.4.1`; PR #2 body updated to match.

### Recount at `a4fb247`

```
$ cd profiler && go test -count=1 -v ./... | grep '^=== RUN' | wc -l
     186
$ ... | grep -c '^=== RUN   [^/]*$'          # top-level functions
51
$ ... | grep -c '/'                          # subtests, all depths
135
$ ... | awk -F'/' 'NF==2' | wc -l            # direct subtests
     108
$ ... | awk -F'/' 'NF>=3' | wc -l            # nested deeper
      27
$ go test -count=1 -v . | grep -c '^=== RUN   [^/]*$'       # profiler pkg
47
$ go test -count=1 -v ./cmd | grep -c '^=== RUN   [^/]*$'   # cmd pkg
4
```

**51** functions (47 profiler + 4 CLI) and **135** subtests (108 direct, 27
nested deeper). The previously recorded 50/128 was taken before `e7ab209` added
`TestUsageErrorsExitOneSoThatTwoMeansOneThing` and its seven subtests. Shell
suites unchanged at **134** (25 + 19 + 32 + 58), Go suite green:

```
$ cd profiler && go test -count=1 ./...
ok  	github.com/Okja-Engineering/skill-architect/profiler	0.391s
ok  	github.com/Okja-Engineering/skill-architect/profiler/cmd	1.030s
```

### Disposition

| # | Item | Verdict | Where |
|---|---|---|---|
| 1 | Test counts stale (50/128) | REAL, fixed | `CHANGELOG.md:45`, `RELEASE_NOTES.md:24`, PR body, this file |
| 2 | README `jq` row | REAL, fixed — **but not as prescribed** (below) | `README.md:223` |
| 3 | PR body size line | REAL, fixed | PR body ¶3: base..head `58 files, +4,809/−409`, round pair labelled |
| 4 | Fixture README scrub list | REAL, fixed | `profiler/testdata/otlp/README.md:36-41` — six, floor wording, pointer to the README paragraph |
| 5a | "malformed JSON is the one … to point at" | REAL, fixed | `README.md:76-78`, `docs/profiler-spec.md:221`. The schema-mismatch reason names a batch too (`otlp.go:281-284`) and a field path; only the *byte* is unique |
| 5b | Temporality spellings omitted | REAL, fixed | `README.md:64`, `docs/profiler-spec.md:231`. Quoted digits via `numericText`/`jsonInt64` (`otlp.go:416-425`, `:386-393`); enum names at `otlp.go:661-669` |
| 5c | PR body "speaks only `otlpMetric`" | REAL, fixed | `claude_code.go` names `otlpExport` (×3) and `otlpLogRecord` (×2); never `otlpMetric`. Reaches the rest via `export.metrics()` / `export.logRecords()` |
| 5d | `no_envelope.json` listed as unreadable | REAL, fixed | `profiler/testdata/otlp/README.md:9-14`. It is `unknown`, not `error` — its own row at `:81` and `profiler_test.go:589` already said so |
| 5e | CHANGELOG "both commits" | REAL, fixed | `CHANGELOG.md:53` → "all three commits" |
| 5f | PR "Known limits" omissions | REAL, fixed | PR body: `cmd` coverage and unpinned CI installs added. Both were already in this file's "Found, not fixed" — only the PR body was short |
| 5g | PR row label `zero_counts` | REAL, fixed | The fixture is `zero_token_count.json`. Whole block realigned; every row re-run and reproduced exactly |
| 5h | D6 line citations | REAL, fixed | Were `:96-97`/`:161`; correct at `9b39eb0` was `:114-115`/`:177`; at `a4fb247` it is `:115-116`/`:178` (this pass added a line above them) |

### Disagreement — item 2's prescribed direction was wrong

The worklist said the README's "drops every finding" claim is **false**, on the
evidence that with findings present both scripts exit 127 and with none they emit
the empty literal and exit 0, and asked that this file's "found, not fixed" #1 be
rewritten as "an inconsistent 127-vs-0 surface, not a silent drop".

That observation is real but was taken on this repo's own skills, which raise
**path** findings only. `check-structure.sh` does not put its child's path
findings in its own `findings` array until it has parsed them back through jq at
`:71`, and that line ends `2>/dev/null || true`. So the zero-findings branch at
`:92` is reached *because* jq is missing, not despite it.

The repo's own `missing-script-ref` fixture cannot demonstrate the drop: it
raises `PL002` beside its two `PT001`s, so there is a finding to serialise and it
exits 127 like any other — which is why this took a purpose-built fixture.

Built a fixture whose only fault is a broken script path:

```
$ bash check-structure.sh --json fx/pathfail            # jq present
{"findings":[{"level":"fail","rule":"PT001","message":"script/reference path
 not found: ./scripts/does-not-exist.sh"}],"passed":false}          exit=1

$ PATH=<no jq> bash check-structure.sh --json fx/pathfail
{"findings": [], "passed": true}                                    exit=0
```

A failing skill reported as a clean pass. The silent drop is real; it is **path
findings** that are dropped. A fixture with a policy finding (`PL002`) does die at
`:102` with 127, which is where the worklist's observation comes from. Both are
true, of different inputs.

So: the README row was rewritten to name all three branches with line numbers
rather than to retract the claim, and "found, not fixed" #1 was **kept and
scoped**, not replaced. Installing the prescribed wording would have replaced a
true statement with a false one and retired a real defect from the 0.5.0 list.

Full matrix, `jq` masked from `PATH` (symlink farm of `/bin` + `/usr/bin` less
`jq`; both scripts are `set -euo pipefail`, so the failing `echo | jq` pipeline
aborts them at 127 rather than reaching their `exit $fail`):

```
audit-report.sh, any input            127   (audit-report.sh:85, the merge every
                                            run reaches; :81 comes first, but only
                                            for a skill with no license line)
a finding to serialise                127   (check-structure.sh:102, check-paths.sh:94)
nothing to report                     0, {"findings": [], "passed": true}
                                            (check-structure.sh:93, check-paths.sh:84)
path faults only, check-structure     0, passed:true   (jq present: 1, passed:false)
```

### Not done

- The code guard both silent-failure scripts need (`check-structure.sh:71`,
  `check-frontmatter.sh:24`) is still 0.5.0. This pass documented the behaviour;
  it did not change it. No Go or shell code was touched, per the pass's mandate.
- `README.md:223`'s row is now long. It earns its length — it is the only place a
  reader learns that a green `check-structure.sh --json` may mean jq is missing —
  but if 0.5.0 adds the guard, the row collapses to one sentence.

## Round 3 — final residuals

Head `cc56fa2`, on `a4fb247`. Two commits: `e736b1c` (docs) and `cc56fa2`
(tests + the two code comments they correct + the recount). No behaviour
changed in this round — every production line touched is a comment.

| # | Finding | Disposition | Evidence |
|---|---|---|---|
| 1 | The 2^63 boundary is correct but unpinned (`otlp.go:591`, `:613`) | REAL, fixed | `otlp_test.go:341-433` — `TestOTLP_CountIsExactlyTheValuesAnInt64Holds`, 20 subtests, over a decoded wire data point. RED/GREEN below |
| 2 | The schema-mismatch reason does not always name a field path | REAL, fixed | `README.md:76-81`, `docs/profiler-spec.md:221` now describe both arms; `otlp_test.go:145-197` pins both |
| 3 | PR body ¶3 labelled a range by position | REAL, fixed | ¶3 now names the round-2 code repair, the docs-residuals pass and the final-residuals pass, each with its own range and stat |
| 4 | `otlp.go:604` "a count is never negative" (now `:608`) | REAL, fixed | The comment states the rule the code applies: the test is on the *rounded* value, so `-0.4` is the zero it was and `-0.5` is not a count. Same correction in `docs/profiler-spec.md:215` |
| 5 | `captureExitCode` had no test that can fail | REAL, fixed | `cmd/main_test.go:106-161`. The mixed profile (one signal read, one errored) is unreachable through the CLI and is the case the rule turns on |
| 6 | `otlp.go:138-140` hasEnvelope comment | REAL, fixed | It said "named the key"; `{"resourceMetrics":null}` names the key and is *not* an envelope, because `*[]T` takes nil from a JSON null. The classification is right per ProtoJSON (null reads as the field default); the comment now says that, and `otlp_test.go:106-143` pins it |
| 7 | `profiler -h`, `probe -h` and `capture -h` exit 1 | REAL, **not fixed** — recorded for 0.5.0 | An explicitly requested help is conventionally 0; `flag` returns `ErrHelp` and `parseFlags` (`cmd/main.go:41-45`) treats it as a usage error. Left alone: it does not touch the 2-means-nothing-was-read contract a script branches on |
| 8 | PR body / `README.md:195-198` "full error text" under `OTEL_LOG_TOOL_DETAILS` | REAL, fixed | Anthropic's docs list tool parameters and input, Bash commands, MCP server and tool names, skill names, user-authored workflow names, and the `user_prompt` command names otherwise collapsed. Nothing about untruncated commands or error text. Corrected in the README, the PR body and `CHANGELOG.md:39` |
| 9 | status.md cite nits (`:1168`, `:1123`, the Disagreement section) | REAL, fixed | All three re-derived by running the scripts with `jq` masked; see the corrected matrix above |

### RED/GREEN for #1

The predicate was right and untested. Relaxing `>=` to `>` in a scratch copy
leaves the *entire* suite green — no fixture can reach the boundary, because
`value_not_a_count.json`'s `9.3e+18` is refused either way:

```
$ # mutant: if r < 0 || r > maxCountPlusOne
$ go test -count=1 ./...          # with the new test removed
ok  github.com/Okja-Engineering/skill-architect/profiler       0.425s
ok  github.com/Okja-Engineering/skill-architect/profiler/cmd   0.975s
```

And the mutant resurrects X1 — on arm64, for a `token.usage` point carrying
`{"asDouble": 9223372036854775807}`:

```
$ prof-mutant capture --harness claude_code … --otel-file maxint.json
  "tokens": { "state": "present", "source": "otel",
              "value": { "input": 9223372036854775807 } }

$ prof-head   capture --harness claude_code … --otel-file maxint.json
  "tokens": { "state": "unknown",
              "reason": "no readable claude_code.token.usage metric in OTel
               export: 1 data point carried a value that is not a token count…" }
```

With the new table, the mutant goes RED on four subtests:

```
--- FAIL: TestOTLP_CountIsExactlyTheValuesAnInt64Holds/MaxInt64_as_a_bare_double
    otlp_test.go:426: count({"asDouble":9223372036854775807}) read as valueRead, want valueNotACount
--- FAIL: …/MaxInt64_as_a_quoted_double
--- FAIL: …/2^63_itself
--- FAIL: …/a_double_that_rounds_up_to_2^63
```

GREEN at head: all 20 subtests pass. The table walks the boundary itself —
2^63 bare and quoted, the largest representable `float64` below it
(2^63−1024, accepted exactly), and 9223372036854775296, which is under
MaxInt64 as written and ties-to-even *up* to 2^63 — plus the accepted window
(`-0.4` → 0, `-0.5` refused, 1522.7 → 1523), the non-finite cases, and the
`asInt` path that carries MaxInt64 exactly because it never goes through a
float. It is pinned to the invariant, not to the comparison: **a value becomes
a count iff it is finite and 0 ≤ round(v) < 2^63, and the count is the same
number on every architecture.**

### RED/GREEN for #2, #5 and #6

Each new test was mutated the same way, in a scratch copy, and each goes RED:

```
#2  collapse the fieldless arm into the path arm
    reason = "…: value at  in batch 2 is not the expected type"   ← the hole
    want it to contain "a value in batch 2"                        RED

#5  captureExitCode: drop `&& read == 0`
    main_test.go:157: captureExitCode = 2, want 0 — the capture produced
    something, so a wrapper must not treat it as a dead run        RED
    (every pre-existing cmd test stays green: the mutant is equivalent
     through the CLI, which is the whole reason the unit exists)

#6  hasEnvelope: `return len(e) > 0`
    reason = "no claude_code.token.usage metric found in OTel export"
    want it to contain "OTel export is not OTLP/JSON"              RED
```

### Recount at `cc56fa2`

**55** Go test functions (50 in the profiler package, 5 in the CLI), **165**
subtests (138 direct, 27 nested deeper) — up 4 and 30. Shell suites unchanged
at **134** (25 + 19 + 32 + 58). `CHANGELOG.md:45`, `RELEASE_NOTES.md:24` and
the PR body carry the new numbers; the changelog count moved in the same
commit as the tests that changed it.

```
$ cd profiler && go build ./... && go vet ./... && gofmt -l . && go test -race -count=1 ./...
ok  github.com/Okja-Engineering/skill-architect/profiler       1.365s
ok  github.com/Okja-Engineering/skill-architect/profiler/cmd   2.306s
$ tests/test_skill.sh; tests/test_walk.sh; tests/test_f01.sh; tests/test_f02.sh
25 passed, 0 failed   19 passed, 0 failed   32 passed, 0 failed   58 passed, 0 failed
```

### Found, not fixed (added this round)

- `profiler -h`, `probe -h` and `capture -h` exit 1 (#7 above). 0.5.0.
- The not-a-count reason says "a count is a whole number from 0 to
  9223372036854775807" (`claude_code.go:343`). For an `asDouble` that is a
  hair generous: the literal 9223372036854775807 is *inside* the range as
  written and is still refused, because no `float64` is MaxInt64. The domain
  statement is right and only `asInt` can reach the top of it; rewording a
  user-facing reason (and the fixtures and tests that assert it) was more
  churn than this pass earns. 0.5.0.

---

## Round 4 — final truth pass

Head `a98b8c4`, on `cc56fa2`. Two commits: `d53ae70` (the skill-activation
falsehood, code + test + five docs sites) and `a98b8c4` (the value-bucket and
envelope rules, plus two test-comment nits and one new table row). One
user-visible string changed — the `skill_activation` reason printed in every
Claude Code profile. No other behaviour changed; every other production line
touched is a comment.

| # | Finding | Disposition | Evidence |
|---|---|---|---|
| 1 | "Claude Code emits no skill activation event" is false (HIGH, ships to every user) | REAL, fixed | Verified against the vendor doc directly, not via a summariser — see "Vendor verification" below. Fixed in all nine places: `claude_code.go:46` and `:197`, `profiler_test.go:290`, `README.md:106-119`, `RELEASE_NOTES.md:22`, `docs/profiler-spec.md:70`, `:235`, `:243`, plus `CHANGELOG.md:47` whose round-3 correction did not go far enough. RED/GREEN below |
| 2 | `docs/profiler-spec.md:215` misclassifies NaN/Infinity | REAL, fixed | Reproduced at head before fixing (runtime table below). The spec now names the two buckets `tokenPointCounters` actually keeps. `otlp_test.go:426` adds `{"asDouble":"NaN","asInt":"9"}` so the spec's new claim is executable; mutation-tested |
| 3 | PR body `cmd` coverage bullet is stale and inverted | REAL, fixed | Five functions, not four; `go test -cover ./...` measured at `a98b8c4` gives **12.6%** for `cmd` and **95.7%** for `profiler`, not 3.4%/95.4%. The bullet now also says *why* coverage is non-zero: the fifth test calls `captureExitCode` in-process rather than through a subprocess |
| 4 | The envelope key-vs-value correction never reached the prose | REAL, fixed | Reproduced at head (runtime table below). One statement of the predicate, in terms of value presence: `README.md:83`, `docs/profiler-spec.md:225`, `:266` (AC13) |
| 5 | `otlp_test.go:358` "within 512" is off by one at the low end | REAL, fixed | MaxInt64−512 is 9223372036854775295, one *below* the midpoint 9223372036854775296, and it rounds down. The comment now names the midpoint, which is the value the table already used |
| 6 | `otlp_test.go:110-112` contradicts `testdata/otlp/empty_envelope.json` | REAL, fixed | The empty-list case *is* a reviewed fixture. The comment now says so and explains why only the `null` spelling is pinned inline |
| 7 | PR body Known limits missing the round-3 not-a-count item | REAL, fixed | Added, with the reason it is deferred: the two leaves have genuinely different upper bounds and one clause cannot state both |
| 8 | `README.md:109-113` / `spec:235` list `skill.name` on three events | REAL, fixed — **but the finding's list was wrong** | See "Disagreement" below |

### Vendor verification for #1

`WebFetch` of `code.claude.com/docs/en/monitoring-usage` returned "**there is
no section or event documentation for a 'Skill activated' event**". That is
wrong — the markdown conversion drops it. Fetching the raw page and grepping
found four occurrences, and the section reads, verbatim:

```
Skill activated event
Logged when a skill is invoked, whether Claude calls it through the Skill tool
or you run it as a / command.
Event Name : claude_code.skill_activated
Attributes :
  All standard attributes
  event.name : "skill_activated"
  event.timestamp : ISO 8601 timestamp
  event.sequence : monotonically increasing counter for ordering events within a session
  skill.name : Name of the skill. For user-defined and third-party plugin skills
    the value is the placeholder "custom_skill" unless OTEL_LOG_TOOL_DETAILS=1
  invocation_trigger : How the skill was triggered ("user-slash",
    "claude-proactive", or "nested-skill")
  skill.source : Where the skill was loaded from (for example, "bundled",
    "userSettings", "projectSettings", "plugin")
  skill.kind : "workflow" when the skill is a workflow skill. Absent otherwise
```

No version gate. Neighbouring attributes in the same document carry explicit
"Requires Claude Code v2.1.x or later" notes; this event and its four
attributes carry none.

### RED/GREEN for #1

The test pins the spec's reason string verbatim, so it is the contract between
the spec and the code. Changed the test constant first, with the code
untouched:

```
$ go test -count=1 -run TestCapture_TheHarnessLevelReasonsAreTheSpecsWordForWord ./...
--- FAIL: TestCapture_TheHarnessLevelReasonsAreTheSpecsWordForWord (0.00s)
    --- FAIL: .../full_export.ndjson (0.00s)
        profiler_test.go:305: skill_activation reason =
              "Claude Code emits no skill activation event; this adapter does not yet read skill.name, ..."
            want
              "This adapter does not yet read Claude Code's skill telemetry: the claude_code.skill_activated event, ..."
    --- FAIL: .../skill_name_present.json (0.00s)
    --- FAIL: .../malformed.json (0.00s)
    --- FAIL: .../no_envelope.json (0.00s)
```

RED across all four fixtures. Then the code:

```
$ go test -count=1 -run TestCapture_TheHarnessLevelReasonsAreTheSpecsWordForWord ./...
ok  	github.com/Okja-Engineering/skill-architect/profiler	0.231s
```

And through the built CLI, which is the surface the claim ships on:

```
$ profiler capture --harness claude_code --session s1 --skill-dir /s --snapshot x \
    --otel-file profiler/testdata/otlp/skill_name_present.json | jq -r .skill_activation.reason
This adapter does not yet read Claude Code's skill telemetry: the
claude_code.skill_activated event, logged when a skill is invoked through the
Skill tool or a / command, carries skill.name, invocation_trigger, skill.source
and skill.kind. Reading it is 0.5.0.
```

One deliberate narrowing. The first draft of the reason carried the
`"custom_skill"` placeholder caveat, and that made
`profiler_test.go:820`'s `wantOut: {"redact", "OTEL_LOG_TOOL_DETAILS",
"custom_skill"}` fail. That assertion guards a real invariant — a profile
must not tell a user their verbatim skill name was redacted, and
`skill_name_present.json` carries it verbatim — so the caveat came out of the
reason string and stayed in the README prose, where the two surfaces can be
distinguished. The invariant test is untouched.

### Reproduction for #2 and #4, at head, before any edit

Driven through `ClaudeCodeAdapter.Capture` on crafted exports:

```
asDouble NaN                       -> unknown "1 data point carried no asDouble or asInt value that reads as a number"
asDouble Infinity                  -> unknown "1 data point carried no asDouble or asInt value that reads as a number"
asDouble NaN + asInt 9             -> present {"input":9}
asDouble 1e400 (overflow)          -> unknown "1 data point carried no asDouble or asInt value that reads as a number"
asDouble -5                        -> unknown "1 data point carried a value that is not a token count: ..."
asDouble 9223372036854775807       -> unknown "1 data point carried a value that is not a token count: ..."
asInt "9223372036854775807"        -> present {"input":9223372036854775807}

{"resourceMetrics":null}           -> unknown "OTel export is not OTLP/JSON: no resourceMetrics or resourceLogs found..."
{"resourceLogs":null}              -> unknown "OTel export is not OTLP/JSON: ..."
{"resourceMetrics":null,"resourceLogs":null} -> unknown "OTel export is not OTLP/JSON: ..."
{"resourceMetrics":[]}             -> unknown "no claude_code.token.usage metric found in OTel export"
{"resourceLogs":[]}                -> unknown "no claude_code.token.usage metric found in OTel export"
{}                                 -> unknown "OTel export is not OTLP/JSON: ..."
```

So NaN and Infinity are *unreadable*, not not-a-count; they do not shadow
`asInt` the way an out-of-range finite double does; and the envelope test is
value presence, not key presence. Both spec sentences were false. The code was
right in both cases and already pinned (`otlp_test.go:409-410`,
`:115`) — this round is prose catching up to code, which is the same
recurrence twice and is why the standing grep rule below exists.

### Mutation test for the one new test row

```
$ # mutant: NaN returns valueNotACount -- the classification the spec claimed
$ go test -count=1 -run '.../asInt_is_read_when_asDouble_is_NaN' -v .
    otlp_test.go:435: count({"asDouble":"NaN","asInt":"9"}) read as valueNotACount, want valueRead
--- FAIL: TestOTLP_CountIsExactlyTheValuesAnInt64Holds/asInt_is_read_when_asDouble_is_NaN
```

Caught. Restored, `git diff --quiet profiler/otlp.go` clean. Tally for the PR
is now 46 mutations, 45 caught.

### Standing rule: greps run for each rule touched

Required before push: for each rule corrected, a grep that *would* surface a
contradicting restatement.

**Rule 1 — a reason may say what this adapter does not read; it may not say
what the harness does not emit unless true.**

```
$ grep -rni "emits no skill\|no skill activation event\|no skill_activated\|does not emit.*skill\|emits no.*activation" \
    --include="*.go" --include="*.md" --include="*.sh" --include="*.json" .
(no matches)
```

Dry. A second, wider sweep over every remaining `skill activation|skill.name|
skill_activated` mention confirmed the survivors are all true statements
(`types.go:131` and `:171` name the *result type*; `testdata/otlp/README.md:68`
already said "this adapter does not read the attribute yet").

**Rule 2 — NaN/Infinity are unreadable, not not-a-count.**

```
$ grep -rni "NaN\|infinit" --include="*.go" --include="*.md" --include="*.sh" .
```

This one was **not** dry, and surfaced three restatements the report did not
list — all fixed in `a98b8c4`:

- `docs/profiler-spec.md:256` — "a value that is not a count (negative, not a
  number, or past `int64`)". A direct contradiction of the corrected `:215`,
  in the same document.
- `RELEASE_NOTES.md:15` — "Negative values, `NaN` and infinity were all
  assimilated too. Each is refused and counted now".
- `CHANGELOG.md:26` — "Each of those is now refused at the leaf".

Re-run after the fix leaves only true statements.

**Rule 3 — the envelope test is a value, not a key.**

```
$ grep -rni "resourceMetrics\|resourceLogs" --include="*.md" . | grep -i "key\|names either\|neither\|nothing under"
$ grep -rni "envelope" --include="*.go" --include="*.md" --include="*.sh" . | grep -i "key\|null\|empty\|neither"
```

Dry of contradictions. The survivors are consistent: `CHANGELOG.md:29` already
described `{"resourceMetrics": []}` reaching the extractors, `otlp.go:46`'s
"envelope keys as pointers" is about the Go field representation (which is
exactly the mechanism that makes the test a value test), and
`spec:250`/`:256`'s "carries no OTLP envelope" is the predicate stated
abstractly.

### Disagreement — the `skill.name` event list in the LOW nits

The nit said the vendor doc also carries `skill.name` on "API error, API
refusal, and **request/response body events**". The first two are right; the
last is not. Attributing every `skill.name` occurrence in the rendered page to
its own section gives exactly six carriers:

| Carrier | `skill.name`? |
|---|---|
| `claude_code.cost.usage` (Cost counter) | yes |
| `claude_code.token.usage` (Token counter) | yes |
| `claude_code.api_request` | yes |
| `claude_code.api_error` | yes |
| `claude_code.api_refusal` | yes |
| `claude_code.skill_activated` | yes |
| `claude_code.api_request_body` | **no** |
| `claude_code.api_response_body` | **no** |

The body events carry `body`, `body_ref`, `body_length`, `body_truncated` and
`model`, and no attribution attributes at all. The README and spec therefore
list the five request-scoped carriers by name rather than saying "including" —
the list is closed and short enough to state, and "including" would have hidden
that the body events are not on it.

### Other judgement calls

- **The not-a-count bound (`claude_code.go:343`) is left unfixed, as routed**,
  and is now in the PR body's Known limits. I think it is genuinely fixable
  and worth 0.5.0: the honest message names the constraint without implying
  the `asDouble` bound, because `asInt` and `asDouble` have different upper
  bounds and one clause cannot state both. Not done here — it changes a
  user-facing string the spec pins verbatim, and that is not a change to make
  in a final truth pass on a release PR.
- **No code fix was warranted for #2 or #4.** Both were prose drift away from
  code that was already correct and already pinned. Shipping a code change
  here would have been a guess with no runtime evidence behind it.

### Recount and coverage at `a98b8c4`

```
$ cd profiler && go build ./... && go vet ./... && gofmt -l . && go test -race -count=1 ./...
ok  	github.com/Okja-Engineering/skill-architect/profiler	1.356s
ok  	github.com/Okja-Engineering/skill-architect/profiler/cmd	2.313s

$ go test -cover -count=1 ./...
ok  	github.com/Okja-Engineering/skill-architect/profiler	0.245s	coverage: 95.7% of statements
ok  	github.com/Okja-Engineering/skill-architect/profiler/cmd	1.221s	coverage: 12.6% of statements

$ tests/test_skill.sh; tests/test_walk.sh; tests/test_f01.sh; tests/test_f02.sh
25 passed, 0 failed   19 passed, 0 failed   32 passed, 0 failed   58 passed, 0 failed
```

Functions **55** (unchanged). Subtests **165 → 166** (139 direct, 27 nested
deeper) — the one new `count()` table row. Shell assertions **134**
(unchanged, 25+19+32+58). OTLP fixtures **38** (unchanged). The count change
is carried in the same commit into `CHANGELOG.md:45`, `RELEASE_NOTES.md:25`
and the PR body.

Behaviour through the built CLI, re-verified at `a98b8c4` (binary in
`$(mktemp -d)`), unchanged from round 3:

```
cache_only         exit=0  {"cache_read": 20480}
zero_token_count   exit=0  {"input": 0, "cache_creation": 0}
malformed          exit=2  "malformed JSON at byte 87 in batch 1: unexpected end of JSON input"
empty_envelope     exit=0  "no claude_code.token.usage metric found in OTel export"
missing file       exit=2
unconfigured       exit=0
```

### Threads

PR #2 has **no review threads** — `reviewThreads` returns empty, and every
prior round was reported as an issue comment. Nothing to resolve; round 4 is
reported the same way.

---

## Round 5 — closing residuals

Head `f7311f8`, on `a98b8c4`. One commit, `f7311f8`: 4 files, +13/-10.
Comments and prose only. No production statement changed, no test changed, no
user-visible string changed. Four routed items plus five LOW nits, all REAL,
all fixed.

| # | Finding | Disposition | Evidence |
|---|---|---|---|
| 1 | The value-bucket rule is restated by magnitude in three shipped docs | REAL, fixed | Reproduced at head before editing (runtime table below). `CHANGELOG.md:26`, `RELEASE_NOTES.md:15` and `docs/profiler-spec.md:256` now state the predicate the way `docs/profiler-spec.md:215` already did: the bucket turns on whether the leaf *decoded as a finite number*, and only then on magnitude. Each of the three now names `{"asInt":"9223372036854775808"}` as the case that separates them |
| 2 | `profiler/otlp.go:46-52` frames the envelope rule as key presence | REAL, fixed | Same root as #1 — a predicate restated as the thing it correlates with rather than the thing the code tests — and the same reproduction. The comment now states the value test and agrees with `hasEnvelope`'s own comment at `:139-144`. The pointer representation was already right; only the justifying sentence changed |
| 3 | PR body coverage bullet describes the wrong mechanism | REAL, fixed | `go tool cover -func` below. **Two** tests build the binary as a subprocess (`cmd/main_test.go:59` and `:168`, both via `buildProfiler`), **three** run in-process, and `captureFlagError` is 100% covered alongside `captureExitCode` — the two of them are the whole of the 12.6%. The bullet now names what the cover profile attributes, function by function |
| 4 | `claude_code.go:338` cite is stale at head | REAL, fixed | `a98b8c4` added five lines to the doc comment above the clause; the not-a-count clause is now `:343`, and `:338` is the `otherTemporality` clause. Corrected in the PR body and in both places here (round 3 "Found, not fixed", round 4 "Other judgement calls") |
| L1 | Round-4 table cites `README.md:102`/`:103` for prose now moved | REAL, fixed | The skill-activation prose is `README.md:106-119`; the `skill.name` rider list inside it is `:109-113`. Both cites updated |
| L2 | "all seven places" lists eight and omits `RELEASE_NOTES.md:22` | REAL, fixed | Now "all nine places", with `RELEASE_NOTES.md:22` added. The other eight cites were re-read at head and are all still accurate |
| L3 | PR body recount command returns 221, not 166 | REAL, fixed | 55 + 166 = 221; the single `grep '^=== RUN'` counts functions and subtests together. The body now gives the two commands that yield the two numbers separately |
| L4 | Known-limits bullet names only `probe -h`/`capture -h` | REAL, fixed | `profiler -h` exits 1 too, by a *different* route: `main`'s switch has no `-h` case, so it takes `default:` -> `usage(); os.Exit(1)`, the unknown-command path — not `parseFlags`/`ErrHelp`. The bullet now names all three and distinguishes the two mechanisms. Also corrected at the two statements of the rule here |
| L5 | `CHANGELOG.md:45` and `RELEASE_NOTES.md` word the three new shell assertions differently | REAL, fixed | The sentence is at `RELEASE_NOTES.md:25`, not `:29` (`:29` is blank). It said "three new version-surface assertions"; it now uses CHANGELOG's specific naming — two marketplace-manifest assertions and the adapter-version one — which is what the diff actually added (`git diff 30f374c..a98b8c4 -- tests/`) |

### Reproduction for #1 and #2, at head `a98b8c4`, before any edit

Binary built into `$(mktemp -d)`; fixtures generated into a scratch dir. Run as
`capture --harness claude_code --session s --snapshot h --skill-dir /d
--otel-file <f>`, reading `tokens.reason` out of the profile on stdout:

| input | `tokens` reason | bucket |
|---|---|---|
| `{"asInt":"9223372036854775808"}` | `1 data point carried no asDouble or asInt value that reads as a number` | **unreadable** |
| `{"asDouble":9223372036854775808}` | `1 data point carried a value that is not a token count: a count is a whole number from 0 to 9223372036854775807` | **not a count** |
| `{"resourceMetrics":null}` | `OTel export is not OTLP/JSON: no resourceMetrics or resourceLogs found.` | **not an export** |
| `{"resourceMetrics":[]}` | `no claude_code.token.usage metric found in OTel export` | **an export** |

Rows 1 and 2 carry the same magnitude — exactly 2^63, present, finite and
numeric — and land in *different* buckets. So magnitude is not the predicate:
`jsonInt64` (`otlp.go:388-395`) returns false on `strconv.ParseInt`'s range
error, so that leaf never decoded at all and `count` (`otlp.go:615-629`) falls
through to `valueUnreadable`. Pinned at `otlp_test.go:416`. The three docs said
"a finite number that rounds negative or to 2^63 or beyond *is not a count*",
which is false for row 1.

Rows 3 and 4 both carry the `resourceMetrics` *key* and land in different
buckets, so key presence is not the predicate either — value presence is,
exactly as `hasEnvelope` and `spec:266` already said.

No code change was warranted for either item. Both were prose that had drifted
from code that is already correct and already pinned; changing code to match the
prose would have been a guess with no runtime evidence behind it.

### Standing rule: greps run for each rule touched

Value-bucket rule — every sentence in the repo that states it:

```
$ grep -rn "2^63\|9223372036854775808\|maxCountPlusOne\|not a count\|not-a-count\|rounds negative" \
    --include="*.md" --include="*.go" --include="*.sh" . | grep -v "^\./\.git"

CHANGELOG.md:26              magnitude-only            -> FIXED
RELEASE_NOTES.md:15          magnitude-only            -> FIXED
docs/profiler-spec.md:256    magnitude-only            -> FIXED
docs/profiler-spec.md:215    authoritative, correct    -> left
README.md:64                 states only what a count *is* (a whole number 0..2^63-1,
                             from asDouble or asInt). True, and it states no refusal
                             split, so magnitude cannot mislead here  -> left
profiler/otlp.go:590,593,609,618      code and its own comment, correct  -> left
profiler/otlp_test.go:380,392,416     table pins both buckets, correct   -> left
```

Envelope rule — every sentence in the repo that states it:

```
$ grep -rn "resourceMetrics\|resourceLogs\|hasEnvelope\|envelope" \
    --include="*.md" --include="*.go" --include="*.sh" . | grep -v "^\./\.git" \
  | grep -iv "testdata/" | grep -iE "key|value|null|empty|not an OTLP|not OTLP|carr"

profiler/otlp.go:46-52       key-presence framing      -> FIXED
README.md:83-87              value-not-key, names the null case, correct  -> left
docs/profiler-spec.md:225    correct                                      -> left
docs/profiler-spec.md:266    AC13, correct                                -> left
profiler/otlp.go:74          "only hasEnvelope cares which one arrived"    -> left
profiler/otlp.go:138-144     hasEnvelope's own comment, correct           -> left
profiler/otlp_test.go:106-114  correct                                    -> left
profiler/profiler_test.go:748  "neither envelope key is there to be empty" —
                             accurate for no_envelope.json, which carries neither
                             key at all, so key and value coincide         -> left
```

Help-exit rule and the recount command are stated **only** in the PR body and
this file — no shipped doc states either, so there was nothing to sweep:

```
$ grep -rn "probe -h\|capture -h\|ErrHelp" --include="*.md" --include="*.go" --include="*.sh" . \
    | grep -v "^\./\.git"
(no hits)
$ grep -rn "=== RUN" --include="*.md" . | grep -v "^\./\.git"
(no hits)
```

The sweep surfaced two sites beyond the routed ones — `README.md:64` and
`profiler_test.go:748` — both read in full and both correct as written. Unlike
round 4, this round's greps ended dry: no extra defects. That is the signal the
rule is meant to produce when a rule really is stated consistently.

### Verification at `f7311f8`

```
$ cd profiler && go build ./... && go vet ./... && gofmt -l . && go test -race -count=1 ./...
ok  	github.com/Okja-Engineering/skill-architect/profiler	1.329s
ok  	github.com/Okja-Engineering/skill-architect/profiler/cmd	2.075s

$ go test -cover -count=1 ./...
ok  	github.com/Okja-Engineering/skill-architect/profiler	0.201s	coverage: 95.7% of statements
ok  	github.com/Okja-Engineering/skill-architect/profiler/cmd	0.956s	coverage: 12.6% of statements

$ go tool cover -func=cmd.cover          # the evidence for #3
cmd/main.go:16   main               0.0%
cmd/main.go:42   parseFlags         0.0%
cmd/main.go:48   cmdProbe           0.0%
cmd/main.go:69   cmdCapture         0.0%
cmd/main.go:129  captureExitCode  100.0%
cmd/main.go:157  captureFlagError 100.0%
cmd/main.go:164  getAdapter         0.0%
cmd/main.go:172  usage              0.0%
total:                             12.6%

$ tests/test_skill.sh; tests/test_walk.sh; tests/test_f01.sh; tests/test_f02.sh
25 passed, 0 failed   19 passed, 0 failed   32 passed, 0 failed   58 passed, 0 failed
```

Counts, every one unchanged from `a98b8c4`:

```
$ cd profiler
$ go test -count=1 -v ./... | grep -c '^=== RUN   [^/]*$'      # functions
55
$ go test -count=1 -v ./... | grep '^=== RUN' | grep -c '/'    # subtests, all depths
166
$ go test -count=1 -v . | grep -c '^=== RUN   [^/]*$'          # profiler pkg
50
$ go test -count=1 -v ./cmd | grep -c '^=== RUN   [^/]*$'      # cmd pkg
5
$ ls testdata/otlp/*.json testdata/otlp/*.ndjson | wc -l       # fixtures
38
```

Shell assertions **134** (25+19+32+58), unchanged. `CHANGELOG.md:45`,
`RELEASE_NOTES.md:25`, the PR body and this file all state 55 / 166 / 134 / 38
and agree with each other and with the commands above.

### Threads

PR #2 still has **no review threads** — the `reviewThreads` GraphQL connection
returns 0 nodes, and every round including this one arrived as an issue comment.
Nothing to resolve or reply to. The PR body is updated to head `f7311f8`, with
the diff totals refreshed (58 files, +5,092/-409).

## History cleaned before merge (2026-09-13 ~21:40)

All 47 commits on `release/0.4.1` carried a `Co-Authored-By: Claude` trailer. The user does not want AI co-authorship on this work. Stripped with `git filter-repo` in a fresh clone (the primary repo and both worktrees untouched), force-pushed with a lease.

Head `f7311f8` -> `847b7dc`. Proof nothing but messages changed: `git diff f7311f8 847b7dc` is empty and the tree hash is `dcabe9ec` on both. 47 commits, subjects byte-identical, authorship entirely `imagineux`, zero trailers anywhere, CI success on the new head. `origin/main` never carried a trailer, so no published history was rewritten.

Every SHA cited in the round 1-5 sections above refers to the pre-rewrite commits. They are accurate as a record of what happened; their post-rewrite equivalents differ only in the removed trailer line.

Standing rule from here: no `Co-Authored-By` trailer for Claude or any AI agent, in any commit.

## Round 6 — malformed tail

Head `82ecdbb` → `f3c51a4`. Three commits, 9 files, +181/−8. One routed defect,
fixed at the root; two further defects verified, reproduced, and recorded below
for 0.5.0 with the accumulator untouched, as routed.

| # | Finding | Disposition | Evidence |
|---|---|---|---|
| 1 | `readOTLP` accepts a malformed tail and silently drops real data (`profiler/otlp.go:189`, `dec.More()` as the loop condition) | REAL, fixed | Reproduced at `82ecdbb` through the built CLI (table below), then fixed in `1196254`. `dec.More()` returns false on a stray `}` or `]` without an error, so the walk exited cleanly and everything after the stray byte was ignored — including whole batches |
| 2 | Resource and instrumentation-scope identity are not part of the series key | REAL, deferred to 0.5.0 | Reproduced at `82ecdbb` (table below). Shares one root with #3; documented as a known limit in `docs/profiler-spec.md`, `README.md`, `CHANGELOG.md`, `RELEASE_NOTES.md` and the PR body |
| 3 | `startTimeUnixNano` is not read, so a cumulative counter reset inside one capture discards the earlier run | REAL, deferred to 0.5.0 | Reproduced at `82ecdbb` (table below). Same root as #2; same documentation |

### Commits (3, oldest first)

```
47b557d  test(profiler): pin that a capture is read to its end or reported as a failure
1196254  fix(profiler): end the batch walk at the end of the file, not at the first byte that cannot start one
f3c51a4  docs: record the malformed-tail fix and the two series-key limits deferred to 0.5.0
```

The red repro is recorded first and the fix sits on top of it: `47b557d` is RED
on its own, `1196254` turns it GREEN, and no test changed after the fix landed.

### Verifying #1 before fixing it, at `82ecdbb`

Binary built from `82ecdbb` into `$(mktemp -d)`; fixtures generated into a
scratch dir from `testdata/otlp/multi_batch_delta.ndjson`'s first line (delta,
100 input tokens) and a copy carrying 500. Run as `capture --harness claude_code
--session s1 --snapshot abc --skill-dir /skills/x --otel-file <f>`:

| input | exit | `tokens` |
|---|---|---|
| valid batch + `\n}` | 0 | `present {"input":100}` |
| valid batch + `\n]` | 0 | `present {"input":100}` |
| valid batch + `\n}\n` + a real batch carrying 500 | 0 | **`present {"input":100}`** — the 500 vanished |
| two valid batches (control) | 0 | `present {"input":600}` |
| valid batch + `\nnot json` (control) | 2 | `error`, `malformed JSON at byte 476 in batch 2: ...` |

Row 3 is the defect at its worst: a complete measurement reported over a file
two thirds of which was never read. It contradicts `docs/profiler-spec.md:221`
("A parse failure is never swallowed: nothing from earlier batches is used") and
breaks the invariant that a `present` result is a complete measurement — the one
failure a profile in schema v1 has no field to disclose.

### Root cause

`encoding/json`'s `Decoder.More` reports whether another *element* follows, and
its answer is "no" for `]`, `}` **and** the end of the stream alike — three
different facts behind one boolean. Driving the batch walk on it made "this byte
cannot begin a value" indistinguishable from "the file is over", so the walk
stopped at the first such byte, returned the batches it had, and left the rest of
the file unread and unmentioned. The failure is structural, not a missing case:
no additional check inside the loop body could have told the two apart, because
the loop was never entered for the stray byte.

Decoding is what tells them apart, so decoding now ends the walk
(`profiler/otlp.go:189-204`): `io.EOF` is the file ending *between* batches —
`Decoder.Decode` returns it only when nothing but whitespace remains, which is
exactly a clean end of a text file — and every other error goes through
`decodeFailure` with the batch index and offset unchanged. One consequence in
`decodeFailure`: `io.EOF` can no longer mean a truncated object, so that arm now
owns `io.ErrUnexpectedEOF` alone instead of claiming both. Nothing else changed;
the accumulator, the series key, metric iteration and `otlpDataPoint` were not
touched.

The invariant the tests pin, rather than the loop: **every byte of the capture is
either consumed as a batch or reported as a failure; nothing between the last
decoded batch and end-of-file is ignored.**

### RED → GREEN → revert-RED

RED, at `82ecdbb` with the new tests applied and `otlp.go` untouched:

```
--- FAIL: TestOTLP_ATailThatIsNotABatchFailsTheWholeFile
    --- FAIL: .../a_stray_closing_brace
        otlp_test.go:107: a file ending in "}" read as 1 batches and no failure; its tail was neither read nor reported
    --- FAIL: .../a_stray_closing_brace_on_its_own_line
    --- FAIL: .../a_stray_closing_bracket
    --- FAIL: .../a_stray_closing_bracket_on_its_own_line
--- FAIL: TestOTLP_ABatchAfterAnUnreadableTailIsNotReadPast
    otlp_test.go:151: read 1 batches and no failure across a stray `}`; the batch behind it was dropped in silence
--- FAIL: TestCaptureDeliversEverySignalProbeAdvertises/a_stray_closing_brace_between_two_batches
    profiler_test.go:714: tokens: state = "present", want "error"
```

The other three subtests of the first function (a stray comma, text that is not
JSON, a batch that ends mid-object) and all five of
`TestOTLP_AFileEndsAtItsLastBatchOrTheWhitespaceAfterIt` passed at `82ecdbb` and
still pass: they are the controls, and a fix that tightened the end-of-file rule
into refusing trailing whitespace would have failed them.

GREEN after `1196254`: all 8 + 1 + 5 + 1 pass, and `go test -race -count=1 ./...`
is clean across both packages.

Revert-RED: `profiler/` copied to a scratch dir, the loop alone reverted to
`for batchIdx := 1; dec.More(); batchIdx++` with the tests left exactly as
committed — the four stray-delimiter subtests, the dropped-batch test and the
fixture case all go RED again with the same messages. The fix is not vacuous.

### The five reproductions re-run after the fix

Binary built from `f3c51a4` into `$(mktemp -d)`, same fixtures, same command:

| input | exit | `tokens` |
|---|---|---|
| valid batch + `\n}` | 2 | `error`, `malformed JSON at byte 475 in batch 2: invalid character '}' looking for beginning of value. No data from earlier batches was used; ...` |
| valid batch + `\n]` | 2 | `error`, `malformed JSON at byte 475 in batch 2: invalid character ']' looking for beginning of value. ...` |
| valid batch + `\n}\n` + a real batch carrying 500 | 2 | `error`, byte 475, batch 2 — and the 500 appears nowhere in the profile |
| two valid batches (control) | 0 | `present {"input":600}` |
| valid batch + `\nnot json` (control) | 2 | `error`, `malformed JSON at byte 476 in batch 2: invalid character 'o' in literal null (expecting 'u'). ...` |

Byte 475 is the stray byte itself: the first batch is 475 bytes including its
newline, so the file's offsets 0–474 are the batch and 475 is the `}`. The batch
index is 2 — the position the tail occupies — and the existing "No data from
earlier batches was used; if the collector was stopped mid-write, delete the
final partial line and retry" advice is true of it. The offset convention and
the reason wording are unchanged.

### Recount at `f3c51a4`

```
$ cd profiler
$ go test -count=1 -v ./... | grep -c '^=== RUN   [^/]*$'      # functions
58
$ go test -count=1 -v ./... | grep '^=== RUN' | grep -c '/'    # subtests, all depths
179
$ go test -count=1 -v . | grep -c '^=== RUN   [^/]*$'          # profiler pkg
53
$ go test -count=1 -v ./cmd | grep -c '^=== RUN   [^/]*$'      # cmd pkg
5
$ ls testdata/otlp/*.json testdata/otlp/*.ndjson | wc -l       # fixtures
39
```

Round 6 added 3 test functions, 13 subtests (7 + 5 + the new fixture's contract
case) and 1 fixture. Nested-deeper subtests are unchanged at 27, so the split is
152 direct + 27 nested. The baseline was re-counted from a clean `git archive` of
`82ecdbb` before the change and came back 55 / 166, matching what the docs
claimed.

```
$ cd profiler && go build ./... && go vet ./... && gofmt -l . && go test -race -count=1 ./...
ok  	github.com/Okja-Engineering/skill-architect/profiler	1.386s
ok  	github.com/Okja-Engineering/skill-architect/profiler/cmd	2.341s

$ tests/test_skill.sh; tests/test_walk.sh; tests/test_f01.sh; tests/test_f02.sh
25 passed, 0 failed   19 passed, 0 failed   32 passed, 0 failed   58 passed, 0 failed
```

Shell assertions **134**, unchanged. `CHANGELOG.md:45`, `RELEASE_NOTES.md:25`,
the PR body and this file all now state 58 / 179 / 134 / 39.

### Found, not fixed — round 6 (both 0.5.0, and one change)

Both are REAL, both were reproduced at `82ecdbb` before being written down, and
both are deliberately deferred: they share one root, and repairing that root
means changing the accumulator, the series key, metric iteration and
`otlpDataPoint`'s fields — all of which this round was scoped out of, and none of
which belongs in a patch release.

The shared root: **the accumulator models a time series as the data-point
attribute set alone**, where the OTel data model identifies a series by resource
+ instrumentation scope + metric + attributes, and a *cumulative* series also
needs `startTimeUnixNano` to mark its reset boundaries. Both defects need
cumulative temporality to bite, which is off Claude Code's default — delta,
`aggregationTemporality: 1`, observed live — so neither can be reached by a
default-configured Claude Code capture today.

1. **Resource and scope identity are not part of the series key, so a capture
   aggregating several resources under cumulative temporality undercounts.**
   Reproduced at `82ecdbb`: two `resourceMetrics` entries with distinct
   `service.instance.id`, identical data-point attributes, cumulative, carrying
   100 and 200 → `present {"input":200}`; the capture holds 300. The two
   resources merge into one series, and cumulative means the latest point
   supersedes rather than adds. The same pair under delta correctly gives 300,
   which is the control that isolates the cause to the series key rather than to
   the walk.

2. **`startTimeUnixNano` is not read, so a cumulative counter reset inside one
   capture discards the earlier run.** Reproduced at `82ecdbb`: one resource,
   cumulative, `startTimeUnixNano` 1→10 carrying 100, then 11→20 carrying 20 →
   `present {"input":20}`; the capture holds 120. A new start time is a new
   series by the OTel data model, and the adapter cannot see it.

Shipped as known limits, in the reader's words rather than the model's:
`docs/profiler-spec.md` (in the series-key bullet, beside the sentences that
already scope what the adapter does), `README.md` (one line each, beside the
token-count paragraph), `CHANGELOG.md`, `RELEASE_NOTES.md`, and the PR body's
known-limits list. Each says plainly that the capture undercounts, and that both
are fixed together in 0.5.0.

### Judgement calls, and one thing not done

- **No bespoke reason for a stray delimiter.** The routed direction allowed one
  if the raw syntax error read poorly. It does not: `malformed JSON at byte 475
  in batch 2: invalid character '}' looking for beginning of value` names the
  shape, the batch and the exact byte, quotes no Go type, and is the same wording
  class the existing `not json` control already produces. Writing a
  stray-delimiter-specific reason would need the offending byte plumbed out of
  the decoder — `json.SyntaxError` carries an offset, not the byte — which is
  more machinery than the evidence asks for. Declined, deliberately.
- **`decodeFailure`'s `io.EOF` arm removed rather than left as a defensive
  duplicate.** With the walk ending on `io.EOF`, that arm was unreachable, and
  leaving two sites disagreeing about what `io.EOF` means is the seam this bug
  lived in. `io.ErrUnexpectedEOF` — the truncated-object case, which
  `malformed.json` and `truncated_final_line.ndjson` both pin — is untouched and
  still names the file's length.
- **One reviewable fixture, not seven.** `stray_close_then_batch.ndjson` is the
  dangerous shape (a real batch behind a stray byte), so it earns a file a
  reviewer can read and a row in `testdata/otlp/README.md`. The one-byte tail
  variants are pinned inline in `otlp_test.go`, per the rule the fixture README
  already states: a file differing from another by one byte is one a reviewer
  reads straight past.
- **The accumulator was not touched**, nor the series key, metric iteration, or
  `otlpDataPoint`'s fields.

### Threads

PR #2 still has **no review threads** — the `reviewThreads` GraphQL connection
returns 0 nodes, and this round arrived as an issue comment like the five before
it. Nothing to resolve. A round-6 comment citing `1196254` and `f3c51a4` is
posted on the PR, and the PR body is updated to head `f3c51a4` with the diff
totals, the recount, the new fix section and the two new known limits.

**The merge hold stands. Nothing was merged.**
