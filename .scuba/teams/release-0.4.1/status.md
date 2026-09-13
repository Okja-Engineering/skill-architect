# Status — v0.4.1 patch release — build COMPLETE, round-1 repair COMPLETE

**Branch:** `release/0.4.1` (based on `origin/main` @ 30f374c)
**Worktree:** `/Users/matthewvandusen/Development/Auraprix/skill-architect-wt/release-0.4.1`
**Head SHA:** `527ba4ea3e13f1cda87f2a843c5659c1c3dbd19e` (round-1 repair; build ended at `7cb32e2`)
**PR:** https://github.com/Okja-Engineering/skill-architect/pull/2 — OPEN, **not** a draft, base `main`, 17 files, +880 / −252

All 18 mandate items done at `7cb32e2`; the round-1 worklist (W1–W18) is
repaired at `527ba4e` — see **Round 1 — root repair** at the bottom of this
file, which is the current record. The build sections below are kept as written
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

| # | Item | State | Evidence (file:line at PR head) |
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
jq masked              → audit-report.sh / check-paths.sh --json        exit 127
                       → check-structure.sh --json  exit 0, {"findings": [], "passed": true}
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

1. **`check-structure.sh --json` silently drops all findings when `jq` is absent.**
   `skills/skill-audit/scripts/check-structure.sh:71` —
   `path_findings=$(echo "$path_json" | jq -c '.findings[]' 2>/dev/null || true)`
   swallows the missing-binary error, so it exits 0 with
   `{"findings": [], "passed": true}` where jq-present returns two findings.
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
