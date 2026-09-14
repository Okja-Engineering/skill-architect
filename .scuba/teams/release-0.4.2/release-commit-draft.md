# v0.4.2 release commit — pre-drafted

_Implementer, 2026-09-14. Written against `release-commit-requirements.md`, the two team
status files, and the two branches' diffs. **Nothing was written outside `.scuba/`, no repo
edit, commit, branch, push or tag was made.** The one command run outside a read was
`go test -v -count=1 ./...` inside the existing `release-0.4.2` worktree, to measure the
post-merge profiler test counts; it modified no file and that worktree is still clean at
`bad47c5`._

## What "post-merge main" means here, and what I resolved line numbers against

| | |
|---|---|
| Base for both PRs | `origin/main` = `2ed34b8` (the v0.4.1 release commit) |
| PR #3 | `release/0.4.2`, head `bad47c5`, 8 commits, +1841 −180 over 23 files |
| PR #4 | `fix/skill-audit-tool-guards`, head `85ad0ea`, 9 commits, +1731 −70 over 12 files |
| Post-merge tree | `2ed34b8` + PR #3's diff + PR #4's diff |

The two diffs are file-disjoint except for `README.md`, and even there the hunks do not
overlap (PR #3 rewrites original lines 94–106; PR #4 rewrites original lines 242–254), so
both squashes apply cleanly and post-merge content for a file equals that PR's head content.
Every line number below was resolved with `git show <rev>:<path>`, not from any list.

---

## 0. BLOCKING and OPEN — read before applying anything

### OPEN-1 (BLOCKING) — the local checkout's `main` is *not* `origin/main`

```
git rev-parse main        -> 0a836156f87f707923ee29c4c1ebb1fc200d375a
git rev-parse origin/main -> 2ed34b835b95a1fb092f093a0b92e4c8f35a3bbe
git merge-base main origin/main -> 30f374c      # they have DIVERGED
```

Local `main` carries seven commits `origin/main` does not (`fd2cfe5`, `b73326d`, `1e4e845`,
`f13173e`, `e0e2a52`, `236152c`, `0a83615` — the F03/F04 adapter and experiment work) and is
**missing the v0.4.1 release commit `2ed34b8` entirely**. The working tree on top of it is
also dirty: 21 modified tracked files and ~25 untracked paths (`skillgate/`, `go.work`,
`profiler/analyze.go`, `docs/skillgate-*.md`, `skills/skill-gate/`, …).

Consequence: **the release commit must not be made on the local `main` as it stands.** Every
line number in this document is against `origin/main`/the PR heads, and several of them do
not resolve in the local tree. The apply checklist in §8 opens by resetting to the real
post-merge `origin/main`, and the F03/F04 work must be preserved on a branch first.

**Question for the manager:** is the local `main` divergence known and intentional (F04 work
parked on an unpushed local `main`), and where should it go before `main` is reset to
`origin/main`? I did not touch it.

### OPEN-2 — `skillgate` is not part of this release

The task's verification suite names "`profiler` and `skillgate` build/vet/test". `skillgate/`
does not exist on `origin/main`, is in neither PR's diff, and is not in `.github/workflows/ci.yml`.
It is untracked local work in the primary tree (see OPEN-1). §8 therefore verifies `profiler`
plus the four shell suites, which is exactly what CI runs, and omits `skillgate`.
**Question:** should `skillgate` ship in 0.4.2 at all? If yes, it is a separate slice and its
own version surface, not a line edit in this commit.

### OPEN-3 — subtest/assertion counts are measured, one is not re-measurable until apply

Measured at `bad47c5` (post-merge profiler is byte-identical to it — PR #4 touches no
`profiler/` file): **66 test functions** (61 in the profiler package, 5 in the CLI),
**230 subtests** (200 direct, 30 nested deeper), **55 OTLP fixtures**. Shell assertions
**850** (`test_f01` 561, `test_f02` 243, `test_skill` 27, `test_walk` 19), taken from the
scripts team's round-4 re-measurement at `85ad0ea`; `test_skill` stays 27 after the version
bump because the bump changes literals, not assertion count. Re-confirm all of these in §8
step 7 before committing; the commands are given there.

### OPEN-4 — the skills' own `metadata.version` fields are deliberately *not* bumped

`skills/skill-audit/SKILL.md` frontmatter carries `metadata.version: "0.2.0"` and
`skills/skill-rewrite/SKILL.md` carries `"0.1.0"`. These are a **separate axis** from the
plugin release: history shows them moving only when the skill's own content changes
(`7607210` took skill-audit 0.1.0 → 0.2.0; neither v0.4.0 nor v0.4.1 touched them), and the
0.3.1 changelog records such a bump as its own bullet. PR #4 changed skill-audit's scripts
substantially and added two consumer-visible rule IDs, which by that convention is a bump.
The requirements document does not list it. **Question:** bump `skill-audit` to `0.2.1` (or
`0.3.0`) in this commit, or leave it? I have left it out of §1 and flagged it rather than
deciding.

### OPEN-5 — a convention question on the correction heading (see §3)

### OPEN-6 — two `v0.4.0` references in README are stale but out of the stated scope

`README.md:28` ("v0.4.0 adds a harness-agnostic profiler…") and `README.md:365`
(post-merge 423, "the profiler (v0.4.0 preview)"), plus `.out-of-scope.md:8`, still describe
the profiler as a v0.4.0 preview. Not version surfaces for this release and not in the
requirements; named so they are not rediscovered.

---

## 1. Version surfaces

**16 lines across 8 files.** Every one verified by `git show` against the revision that will
be post-merge `main` for that file. Nothing here was taken from a list.

### 1a. The five manifests — untouched by both PRs, line numbers stable

| # | path:line (post-merge) | current literal | 0.4.2 replacement | shift risk |
|---|---|---|---|---|
| 1 | `.claude-plugin/plugin.json:4` | `  "version": "0.4.1",` | `  "version": "0.4.2",` | none — file in neither PR diff |
| 2 | `.codex-plugin/plugin.json:3` | `  "version": "0.4.1",` | `  "version": "0.4.2",` | none |
| 3 | `.cursor-plugin/plugin.json:3` | `  "version": "0.4.1",` | `  "version": "0.4.2",` | none |
| 4 | `.devin-plugin/plugin.json:3` | `  "version": "0.4.1",` | `  "version": "0.4.2",` | none |
| 5 | `.claude-plugin/marketplace.json:13` | `      "version": "0.4.1"` | `      "version": "0.4.2"` | none |

### 1b. The profiler — `profiler/types.go` in neither PR diff; `profiler_test.go` in PR #3 but unshifted

| # | path:line (post-merge) | current literal | 0.4.2 replacement | shift risk |
|---|---|---|---|---|
| 6 | `profiler/types.go:234` | `const AdapterVersion = "0.4.1"` | `const AdapterVersion = "0.4.2"` | none — `types.go` in neither PR diff |
| 7 | `profiler/types.go:233` | `// changes what a profile contains for the same input — as 0.4.1 did.` | `// changes what a profile contains for the same input — as 0.4.1 and 0.4.2 both did.` | none |
| 8 | `profiler/profiler_test.go:410` | `	const want = "0.4.1"` | `	const want = "0.4.2"` | **PR #3 rewrites this file**, but its edits are all below line 1000 — verified still 410 at `bad47c5` |
| 9 | `profiler/profiler_test.go:408` | `// quietly ship a 0.4.1 profile labelled as something else.` | `// quietly ship a 0.4.2 profile labelled as something else.` | same — verified still 408 at `bad47c5` |

`AdapterVersion` is not optional, per the requirements: profiles at both PR heads still stamp
`0.4.1` while reporting different totals than 0.4.1 did for the same export. The invariant —
*no two profiles carrying the same `adapter_version` may report different totals for the same
export* — is violated the moment PR #3 merges and stays violated until #6 lands.

### 1c. `tests/test_skill.sh` — touched by PR #4, asserts unshifted

PR #4's only hunk is `@@ -95,6 +95,25 @@` (a `skill-validator` zero-warning loop inserted
after line 97). Every version assert is **above** it, so all five line numbers hold.

| # | path:line (post-merge) | current literal | 0.4.2 replacement |
|---|---|---|---|
| 10 | `tests/test_skill.sh:28` | `assert data['version'] == '0.4.1', 'version mismatch'` | `assert data['version'] == '0.4.2', 'version mismatch'` |
| 11 | `tests/test_skill.sh:48` | `assert plugins[0]['version'] == '0.4.1', 'plugin entry version mismatch'` | `assert plugins[0]['version'] == '0.4.2', 'plugin entry version mismatch'` |
| 12 | `tests/test_skill.sh:51` | `assert ".claude-plugin/marketplace.json is valid and at 0.4.1" true` | `assert ".claude-plugin/marketplace.json is valid and at 0.4.2" true` |
| 13 | `tests/test_skill.sh:57` | `grep -q 'AdapterVersion = "0.4.1"' profiler/types.go` | `grep -q 'AdapterVersion = "0.4.2"' profiler/types.go` |
| 14 | `tests/test_skill.sh:58` | `assert "profiler AdapterVersion is 0.4.1" true` | `assert "profiler AdapterVersion is 0.4.2" true` |

### 1d. `README.md` — touched by BOTH PRs; one line shifts

| # | path:line | current literal | 0.4.2 replacement | shift |
|---|---|---|---|---|
| 15 | `README.md:55` (unchanged post-merge) | ``` `capability.adapter_version` — `profiler 0.4.1` at this release. It tracks what ``` | ``` `capability.adapter_version` — `profiler 0.4.2` at this release. It tracks what ``` | **none** — both PR hunks are below it |
| 16 | `README.md:269` → **post-merge `README.md:327`** | ``` `claude plugin list` then shows `skill-architect@skill-architect` at version 0.4.1. ``` | ``` `claude plugin list` then shows `skill-architect@skill-architect` at version 0.4.2. ``` | **+58** — PR #3 is net +24, PR #4 net +34, both above this line |

Arithmetic checked against the heads: the line sits at 293 in `bad47c5` (269 + 24) and at 303
in `85ad0ea` (269 + 34); combined, 327. README goes 372 → 430 lines. **Do not trust 327
blindly** — §8 step 3 re-locates every surface by `grep -n` before editing.

### Lines carrying `0.4.1` that are correct as history and must NOT change

`README.md:115`, `README.md:116` (what v0.4.1 reported, post-merge coordinates);
`profiler/testdata/otlp/README.md:61`, `:74`, `:99`; `docs/profiler-spec.md:217`, `:218`;
`profiler/claude_code.go:573` (`0.4.0`); every `0.4.1` inside the released `## 0.4.1`
CHANGELOG and `## v0.4.1` RELEASE_NOTES entries.

---

## 2. The 0.4.2 CHANGELOG section — final prose

Insert between `CHANGELOG.md:5` (`## Unreleased`) and `CHANGELOG.md:7` (`## 0.4.1 — 2026-09-13`),
leaving `## Unreleased` in place with no body, which is how the file already carries it.
Set the date to the day the commit is actually made; `2026-09-14` is today.

~~~~markdown
## 0.4.2 — 2026-09-14

Two fixes, one release. The profiler's cumulative token merge was wrong in three ways that
each read as "fewer tokens than the session spent", and no `skill-audit` script could tell
"this passed" from "I could not check". No new features, and no profile schema keys changed.

**Totals for the same cumulative export differ from 0.4.1**, in the direction of being
correct: a capture spanning several resources or scopes now adds its series instead of letting
one replace another, a counter that restarted inside the capture keeps the run before the
restart, and a flush that reported less than an earlier one on the same run no longer takes
that run down with it. Delta temporality — Claude Code's default, `aggregationTemporality: 1`,
observed live — is unaffected throughout. `capability.adapter_version` moves to 0.4.2, which
is what tells a profile written by this adapter apart from one written by 0.4.1.

**Fixed**

- A token time series is identified the way OTel identifies one: the resource it was exported from, the instrumentation scope that recorded it, the metric's name, and the data point's whole attribute set. This adapter keyed on the data-point attributes alone, so a capture aggregating several resources or scopes under cumulative temporality merged their series and let one running total replace another — two resources reporting 100 and 200 gave 200, not 300, where the same pair under delta correctly gave 300. `schemaUrl` is deliberately not part of that identity: it declares which version of the semantic conventions the attributes follow, not a different origin, and including it would split one series in two and double-count it. An absent `resource` and a resource carrying no attributes are one resource, for the same reason: a resource is its attribute set.
- A cumulative series is a set of runs, and `startTimeUnixNano` says which run a point belongs to. It was not read at all, so a counter that restarted inside one capture discarded everything before the restart — 100 followed by a restart reaching 20 gave 20, not 120. The points sharing a start are one run; a point carrying a different start is a counter that restarted, and its run sits beside the earlier one rather than replacing it, so the series contributes the sum of its runs. Runs are keyed by their start rather than by file order, so batches written out of order still add up. A `startTimeUnixNano` of `0` is a `startTimeUnixNano` that is absent: it is a proto3 `fixed64` and OTLP mandates the proto3 JSON mapping, in which an explicit zero and an omitted field are two encodings of one message — `protojson` writes `"0"` with `EmitUnpopulated` on and omits the field with it off. The same rule now holds for `timeUnixNano`, which read them apart in the same reader. A delta point's `startTimeUnixNano` is the start of that point's own interval rather than of a run, and is not read.
- What a run holds is the **greatest** running total its points reported, and `timeUnixNano` is not read by the merge at all. These are monotonic counters: a running total only goes up, so the values carry their own order, every point of a run is a prefix of the greatest, and a flush reporting less than an earlier one on that run is a capture contradicting itself rather than tokens given back. Ordering by `timeUnixNano` let such a flush take its run down with it, which is how supplying the field that says which run a point is from came to make the profile report *fewer* tokens: one run at start 1000 carrying an untimed 900, a 500 at t2000 and an untimed 1000 reported 500, and erasing the start time raised it to 1000 — adding information lowered the number. The time-ordering apparatus is deleted rather than corrected. Monotonicity is now stated as the assumption it always was; `isMonotonic` is still unread, and that is named under known limits below.
- A cumulative point whose `startTimeUnixNano` is absent, zero or unreadable cannot name its run, but it came from one — either a run that named itself, or a run nothing else in the capture observed — so it is neither a run of its own nor free. The series now contributes the **least total its runs can account for**: the runs added, plus whatever such a point reported over and above the largest of them. Both ends of that are wrong in a direction already shipped. Counting the point as a run of its own adds mass no point ever reported, so one export flush that omitted one field doubled a session; taking the series to be the greatest running total observed anywhere on it throws away the runs the point did not join, which are tokens the session really spent. Runs of 100 and 20 beside an unplaceable 500 give **520** — 500 drops a run that named itself, and 620 invents a third — and runs of 100, 100 and 100 beside an unplaceable 150 give **350**, the case where the unplaceable total is above the largest run but below their sum, so a rule comparing it only with the sum never fires and 50 reported tokens go missing with no case to notice. The property behind all of it, and the one to check a change here against: taking information away — erasing a start time — may lower the total and may never raise it.
- A series carrying both delta and cumulative points is refused whole and counted, rather than resolved one way. The two are opposite instructions — add, or supersede — so every resolution invents a number the export does not contain: adding the increments to a running total counts them twice, dropping them counts them not at all. 0.4.1 treated such a series as cumulative and let the point with the latest `timeUnixNano` win, so which number it produced turned on the timestamps rather than on write order — a cumulative 50 at t9000 written before a delta 100 at t1000 gave 50. The refusal is counted per *series* rather than per data point, because the points are individually fine and it is their company that is malformed, and the `tokens` reason names it. Refusing one series is not refusing the export: the well-formed series beside it still count, and `tokens` is `unknown` only when no series survives.
- No `skill-audit` script reports a verdict a missing tool could not compute. `check-structure.sh --json` with `jq` absent swallowed the missing binary with `2>/dev/null || true`, lost every path finding, and took the zero-findings branch that hard-coded `passed: true` — a clean bill of health at exit 0 from a check that never ran. `check-paths.sh --json` died instead at a bare `jq: command not found`, exit 127 with no payload at all. `check-frontmatter.sh` with `skill-validator` absent captured its 127, special-cased only exits 3 and 1, fell through to the license gate and printed `frontmatter OK` at exit 0 — and printed `POLICY FAIL [PL001]` at exit 2 for a malformed-YAML skill it had not validated. A new `skills/skill-audit/scripts/verdict-guard.sh` owns the one sentence behind all three: `cannot_compute` writes the reason to stderr, a `passed: false` payload to stdout and exits 3, and `require_tool` is its missing-tool caller. `--json` now requires `jq` unconditionally in both writers and in `audit-report.sh` — a stated precondition, not a data-dependent one. Text mode is unaffected.
- A child result a parent does not recognise is an execution error, not a pass. `check-structure.sh` mapped only child exit 1 onto failure, so `check-paths.sh`'s 127 read as success; it now enumerates the child's documented statuses and treats anything unenumerated as an execution error, stops folding the child's stderr into the payload it parses, and no longer reads an unparseable or absent payload as "no findings". A child exiting 0 having printed nothing did the same thing by a different route and is closed by the same pass. `check-frontmatter.sh` enumerates 0, 1, 2 and 3 with `*` as an execution error. Both `--json` writers derive `passed` from the computed verdict in one place, and the zero-findings shortcut is gone.
- `audit-report.sh` no longer reads an unreadable source as a clean one. It captured its only guarded child with `2>&1`, so a `passed: false` payload never parsed and findings stayed empty: with `check-paths.sh` removed, `check-structure.sh --json` returned a `DEP002` payload at exit 3 and the composed report was `{"passed": true, "total_findings": 0}` at exit 0. stdout is now captured alone, per the child's own contract that the payload is stdout and diagnostics are stderr, and a source yielding nothing readable is named in a `policy_error` with `summary.passed` false, mirroring the existing `spec_error`. The composer also proves each source's shape before it reads a verdict out of it: `skill-validator` emitting `0` or `[1,2]`, `skillscore` emitting `true`, and either emitting a two-document stream all took `jq` down — exit 5 or exit 2, stdout empty, no report at all. One `json_document_conforms` in the guard owns the half that is the same for every source, that there is exactly one document, and each source states only the shape it is actually read as, so the quality source's claim reaches inside `overallScore` because the read does, while a `skillscore` payload with no `overallScore` at all is still read and simply leaves the score null.
- The guard's JSON string encoder is settled by ordinal, so its output no longer depends on the locale. Built by interpolation at first, it emitted invalid JSON for a message containing a quote; the replacement escaped through `[[:cntrl:]]`, which under a UTF-8 locale matches bytes bash reports as *negative* ordinals — `printf 'a\x80z'` came out as a sixteen-hex-digit escape under `en_US.UTF-8` and passed through untouched under `C`, and the valid C1 controls U+0080–U+009F, which JSON asks no one to escape, were corrupted identically. The decision is now the ordinal alone, asked only about the range `\u00xx` can spell, so no escape is wider than four hex digits and the same bytes come out under every locale.
- A guard file that will not load is an execution error rather than a policy failure. A malformed `verdict-guard.sh` exited **2** from every script — the status the contract reserves for "policy failure" — and left `audit-report.sh` exiting 1, which is not one of its documented statuses at all. Each script now checks the load on both sides, `bash -n` before and `declare -F` after, and exits 3 with no payload. `check-structure.sh` and `check-frontmatter.sh` wrote their SKILL.md-not-found diagnostic to stdout, which in `--json` mode is the payload channel; both write it to stderr now, matching the other two scripts. `audit-report.sh` exits 3 naming a missing `jq` instead of exiting 127 from the shell mid-report, with no partial report; `skill-validator` and `skillscore` stay soft dependencies there. Swept across every fixture, script, mode and masked tool, no invocation exits outside `{0, 1, 2, 3}`.
- The rule IDs the scripts emit are registered where the skill enumerates them. `DEP001`, `DEP002` and `PATH` are now in `skills/skill-audit/SKILL.md`'s rule-ID enumeration with both levels stated, in the header of every script that emits them, and in the README. `scripts/lib/` was flattened to `scripts/verdict-guard.sh`, which removes the `deep nesting detected` warning it had earned the skill from `skill-validator`, and `tests/test_skill.sh` now asserts that both shipped skills validate with zero errors **and** zero warnings — nothing checked for a warning before, which is how that one rode through every gate unnoticed.
- `AdapterVersion` moved from `0.4.1` to `0.4.2`. It is recorded in every profile, and this release changed what a profile contains for the same export. The five manifests, the `tests/test_skill.sh` asserts and the README's two version references move with it.

**Documentation**

- `README.md` and `docs/profiler-spec.md` describe the merge the code now performs: a series is resource + scope + metric + attributes, a cumulative run is identified by `startTimeUnixNano` and holds the greatest running total its points reported, an unplaceable point joins the run it is cheapest to have come from, a `0` start time is an absent start time, and a mixed-temporality series is refused whole. The paragraph in both files naming the two known limits as deferred to a later release is deleted — this release is where they were fixed.
- `docs/profiler-spec.md` gains a **Deliberate deviations from the proto3 JSON mapping** note. Exactly one deviation, stated so it is not mistaken for a conformance claim: a 64-bit integer field written in exponent form (`1.7893e18`) is refused, though the mapping accepts it. Reading it means going through a `float64`, whose spacing at 1.789e18 is 256ns, and `startTimeUnixNano` is a run identity — two points of one run that round apart would become two runs. Exact exponent parsing was considered and not taken: it needs decimal big-integer arithmetic in a hot leaf for a form `protojson` never emits and nothing observed produces.
- What v0.4.1 actually did is now stated from measurement rather than from memory, in `README.md`, `docs/profiler-spec.md`, `profiler/testdata/otlp/README.md` and the Go test reasons. Two earlier claims were checked against the v0.4.1 binary and disproved: that a capture carrying no start times "merges exactly as it did before runs existed" (900 at t1001 then 500 at t1002 gave 500 at v0.4.1 and gives 900 here), and that "the last value written won" on a mixed-temporality series (it turned cumulative and the latest by `timeUnixNano` won).
- Two code comments that taught a model the code no longer has are corrected: the `array_attribute_series` reason in `profiler/profiler_test.go`, which explained a 200 by "the later point supersedes the earlier" — the right number from a deleted mechanism, and false as stated, since with the values reversed head reports 200 where "later supersedes" gives 100 — and `counterAccumulator`'s doc comment in `profiler/otlp.go`, which offered the type to a second OTLP-speaking adapter "unchanged" while silently clamping a negative point value to zero instead of refusing it. The guarantee lives in the caller, which refuses a value that is not a count before it arrives, and the comment now says so.
- `profiler/testdata/otlp/README.md` rewrites the rows whose explanations described the superseded model, and documents the eight fixtures this release adds. The fixture set goes from 39 to 55.
- `skills/skill-audit/README.md` gains a `jq` row that says what it is required for and when, corrects the `skill-validator` row's now-false clause, removes the sentence stating there is no guard for `jq`, and gains a `DEP002` paragraph.
- 66 profiler test functions pass — 61 in the profiler package, 5 in the CLI — and 230 subtests, 200 direct and 30 nested deeper, under `-race`. The shell suites are at **850**: `test_f01` 561, `test_f02` 243, `test_skill` 27, `test_walk` 19, against 0.4.1's 134. Both shipped skills validate with zero errors and zero warnings. All four suites were run under `LC_ALL=C` and under a UTF-8 locale, because one defect this release fixes was visible only under one of them.

**Known past changes, recorded late**

- 0.4.1's first Fixed entry says the adapter "accepts every 64-bit integer as a JSON number or a decimal string as the OTLP spec requires". The conformance clause is false and was false when written. The proto3 JSON mapping, which OTLP mandates, accepts exponent notation for 64-bit integer fields; this adapter refuses it. The behaviour is deliberate — reading an exponent form means going through a `float64`, which cannot represent every nanosecond instant, and two points of one run that round apart would become two runs — and it is recorded in this release under "Deliberate deviations from the proto3 JSON mapping" in `docs/profiler-spec.md`. What the adapter does is accept a 64-bit integer as a JSON number or a decimal string; what it does not do is conform to the mapping. The 0.4.1 entry is left as written; this is the correction.
- 0.4.1's Documentation section says `docs/profiler-spec.md` and `README.md` "state two known limits of the token series key". They no longer do: both limits are fixed in this release and the paragraph describing them is deleted from both files. The numbers in that entry are still the right numbers for v0.4.1 — two resources reporting 100 and 200 gave 200, and 100 followed by a restart at 20 gave 20 — and they are what this release changes. The 0.4.1 entry is left as written; this is the correction.
- `RELEASE_NOTES.md`'s v0.4.1 entry carries the same two claims and disagrees with itself about when they land: its heading says the limits are "both 0.4.2" and its body says they are "fixed together in 0.5.0". 0.4.2 is the right number and they are fixed here. "Both are described in the README and the spec" was true at v0.4.1 and is no longer, for the reason above. The v0.4.1 release note is left as written; the correction is in the v0.4.2 note.

**Known limits, carried to 0.5.0**

- **`isMonotonic` is unread.** A `claude_code.token.usage` sum declaring `isMonotonic: false` with a genuinely decreasing run reports the greatest value where v0.4.1 reported the latest — 1000 against 200 in the reproduced case. Judged safe for this metric and left deliberately: greatest-wins cannot over-report a monotonic cumulative sum, the OTel data model says a reader should expect non-decreasing values, every fixture in the repo declares `true`, and `claude_code.token.usage` is a Counter. It is the same class as `flags` and belongs beside it.
- **`flags` is unread**, so a stale non-zero `NO_RECORDED_VALUE` data point still reads as a running total. Fixing it is a new leaf on the data point, a new refusal class, its own reason clause, fixtures and a spec paragraph, and no capture route the README documents produces one today. It is the next one to take.
- **A `--json` exit 3 does not always carry a payload.** Three paths deliberately carry none, because there is no skill to render a verdict about: a usage error, a target that cannot be resolved, and a guard file that will not load. Documented in both script headers and in the README, and pinned by tests. Minting rule IDs for them would enlarge the consumer-observable surface to describe caller errors rather than verdicts, so it was not done.
- **Schema v1 has no channel to say that a malformed series was excluded from a `present` total.** A `present` result carries no reason, so a series refused beside a healthy one reduces a total by a defect the profile has no field to name — the same gap as a skipped data point, and tracked with it. Inventing a channel inside v1 would mean a reason on a `present` result, which is a schema change wearing a bug fix's clothes. The caveat channel is the 0.5.0 item the spec already reserves.
~~~~

**Note on the two new headings.** `**Fixed**` and `**Documentation**` are the 0.4.1 entry's
own headings, used unchanged. `**Known past changes, recorded late**` is also 0.4.1's, reused
deliberately — see §3. `**Known limits, carried to 0.5.0**` is **new**: the file has no
existing home for deferred items (0.4.1 put its deferrals in a `**Documentation**` bullet,
which is exactly the bullet §3 has to supersede). Flagged as a structural addition for the
manager's call; the alternative is a `**Documentation**` bullet in 0.4.1's style.

---

## 3. The three false-claim supersessions

### Convention: the file already has one, and it is followed

`CHANGELOG.md` corrects released entries by leaving them byte-for-byte and recording the
correction in the current release's entry, under the heading
**`Known past changes, recorded late`**, with the closing formula *"The 0.3.1 entries are left
as written; this is the correction."* (`CHANGELOG.md:53`) / *"The 0.3.1 entry is left as
written; this is the update."* (`CHANGELOG.md:55`). The 0.4.1 entry's own first bullet does
the same thing inline: *"The bullets below stay as written — they are true as history…"*
(`CHANGELOG.md:15`). `RELEASE_NOTES.md:28` follows the same pattern without a heading.

So: **no new convention is proposed.** The 0.4.2 section reuses that heading and that closing
formula verbatim, which is why all three supersessions appear under
`**Known past changes, recorded late**` in §2.

**OPEN-5:** the heading's words fit "a change we recorded late" better than "a claim that was
never true", and two of these three are the latter. Reusing it keeps the file's structure
unchanged and matches what 0.4.1 actually filed under it; a better-fitting
`**Corrections to released entries**` would be a new heading. I chose the existing one.
Manager's call if you disagree.

### Supersession 1 — `CHANGELOG.md:15`

**Claim as it currently reads** (the clause, inside the long first `**Fixed**` bullet of the
0.4.1 entry):

> accepts every 64-bit integer as a JSON number or a decimal string as the OTLP spec requires,

**Why it is false.** OTLP mandates the proto3 JSON mapping. That mapping permits a 64-bit
integer field to be written as a JSON number in **exponent notation** (`1.7893e18`), and this
adapter refuses that form — deliberately, and it refused it when the sentence was written. So
the adapter does not accept *every* 64-bit integer the spec allows, and the clause "as the
OTLP spec requires" asserts a conformance the code does not have. The refusal itself is
correct and is kept: reading an exponent form means going through a `float64`, whose spacing at
1.789e18 is 256ns, and `startTimeUnixNano` is a run identity, so two points of one run that
round apart would become two runs. PR #3 already withdrew the same claim from
`docs/profiler-spec.md:214` and recorded the deviation at `docs/profiler-spec.md:223`; the
changelog copy of it was left untouched because the entry is history and the version surface
was out of that PR's mandate.

**`CHANGELOG.md:15` is not edited.** Exact replacement text — a new bullet under
`**Known past changes, recorded late**` in the 0.4.2 section:

~~~~markdown
- 0.4.1's first Fixed entry says the adapter "accepts every 64-bit integer as a JSON number or a decimal string as the OTLP spec requires". The conformance clause is false and was false when written. The proto3 JSON mapping, which OTLP mandates, accepts exponent notation for 64-bit integer fields; this adapter refuses it. The behaviour is deliberate — reading an exponent form means going through a `float64`, which cannot represent every nanosecond instant, and two points of one run that round apart would become two runs — and it is recorded in this release under "Deliberate deviations from the proto3 JSON mapping" in `docs/profiler-spec.md`. What the adapter does is accept a 64-bit integer as a JSON number or a decimal string; what it does not do is conform to the mapping. The 0.4.1 entry is left as written; this is the correction.
~~~~

### Supersession 2 — `CHANGELOG.md:47`

**Claim as it currently reads** (opening of the bullet; the bullet is one line, 47, under
`**Documentation**`):

> - `docs/profiler-spec.md` and `README.md` state two known limits of the token series key, both fixed together in 0.4.2 and both confined to cumulative temporality — Claude Code's default is delta (`aggregationTemporality: 1`, observed live), where neither arises. […] Both are real and reproduced; they share one cause and are one change, which is why they are named here rather than half-fixed.

**Why it is false.** Its present tense is a claim about what those two files contain, and PR #3
made it untrue: the paragraph is deleted from `README.md` (the `@@ -94,13 +94,37 @@` hunk
removes the seven-line "Two known limits of how those counts are merged…" paragraph) and
rewritten out of `docs/profiler-spec.md`. Both limits were fixed on that branch, so the files
now describe the merge instead of the limits. Note the rest of the bullet is still accurate as
history: the numbers it gives — 100 and 200 giving 200, 100 then a restart at 20 giving 20 —
are what v0.4.1 did, and are exactly what this release changes.

**`CHANGELOG.md:47` is not edited.** Exact replacement text — bullet 2 under
`**Known past changes, recorded late**`:

~~~~markdown
- 0.4.1's Documentation section says `docs/profiler-spec.md` and `README.md` "state two known limits of the token series key". They no longer do: both limits are fixed in this release and the paragraph describing them is deleted from both files. The numbers in that entry are still the right numbers for v0.4.1 — two resources reporting 100 and 200 gave 200, and 100 followed by a restart at 20 gave 20 — and they are what this release changes. The 0.4.1 entry is left as written; this is the correction.
~~~~

### Supersession 3 — `RELEASE_NOTES.md:20`

**Claim as it currently reads** (the whole bullet, one line):

> - **Two known limits, both 0.4.2.** Under cumulative temporality — not Claude Code's default, which is delta — token counts can undercount two ways: a capture that aggregates several resources merges their series, because resource and scope identity are not part of the series key, and a counter that resets inside one capture discards the run before the reset, because `startTimeUnixNano` is not read. Both are described in the README and the spec, and both are fixed together in 0.5.0.

**Why it is false, twice over.** First, "Both are described in the README and the spec" is
false at both PR heads for the same reason as supersession 2 — the description is deleted.
Second, the bullet **contradicts itself and did so at v0.4.1**: its bold lead says the limits
are "both 0.4.2" and its closing clause says they are "fixed together in 0.5.0". One of the two
numbers was always wrong. 0.4.2 is the right one, and this is the release that makes it true.

**`RELEASE_NOTES.md:20` is not edited.** Exact replacement text — a bullet in the new `## v0.4.2`
entry (it is the last bullet of §5's entry):

~~~~markdown
- **The two known limits named in v0.4.1 are fixed here, and that note needs two corrections.** Its heading said "both 0.4.2" and its body said "fixed together in 0.5.0" — one of those was always wrong, and 0.4.2 is the right one. And "Both are described in the README and the spec" is no longer true: fixing them deleted the paragraph that described them. The v0.4.1 note is left as written; this is the correction.
~~~~

---

## 4. The two stale code comments

Both are comment-only and both teach a model the code no longer has. Line numbers are
**post-merge coordinates**, which for `profiler/` files equals `bad47c5` — PR #4 touches no
`profiler/` file, so nothing shifts them further. Each is anchored by its own text so it can
be relocated by search if a number moves.

### 4a. `profiler/profiler_test.go:1235-1237`

Inside `TestTokens_SeriesDifferingOnlyByANonStringAttributeDoNotMerge`.

**Current:**
~~~~go
	// Two cumulative series, 100 and 200, distinguished only by an arrayValue
	// attribute. Merged, the later point supersedes the earlier and the total
	// is 200; kept apart, they are two running totals and add.
~~~~

**Why it is wrong.** The number 200 is right; the mechanism is the deleted time-ordering
model. And the sentence is false as stated at head: with the values reversed — 200 written
first, then 100 — head reports **200**, because a run holds the greatest total its points
reported, where "the later point supersedes the earlier" would give 100. A reader repairing
this test from the comment would derive the wrong expectation.

**Corrected:**
~~~~go
	// Two cumulative series, 100 and 200, distinguished only by an arrayValue
	// attribute. Merged, they are one series and hold the greatest running
	// total reported on it, which is 200 whichever order the points arrive in;
	// kept apart, they are two running totals and add.
~~~~

### 4b. `profiler/otlp.go:843-846` — the `counterAccumulator` doc comment

The type is declared at `profiler/otlp.go:847`; the comment runs 838–846 and the stale half is
843–846.

**Current:**
~~~~go
// The label is whatever the caller totals by — the token type, for this file's
// only caller today. Nothing here knows what a token is: this is the format
// layer, and a second OTLP-speaking adapter counting something else would use
// it unchanged.
~~~~

**Why it is wrong.** It advertises the type as a reusable format layer, but the type does not
hold the invariant that reuse would need. A negative point value is **clamped to zero inside
it, not refused**: `observe` keeps a value only `if value > s.runs[start]`, `add` raises
`unplaced` only `if p.value > s.unplaced`, and `total` initialises `largest` to zero — so a
negative never lowers anything and never surfaces. `counterSeries`'s own field comment
(`otlp.go:925-927`) even asserts "A count is never negative" as a fact about its input. It is a
fact about *this file's caller*: `claude_code.go:285-292` reads `dp.count()` and drops a
`valueNotACount` point — which is where negatives are refused — before `acc.add` is ever
called. A second adapter counting something that can legitimately decrease would be silently
wrong, not "use it unchanged".

**Corrected** (the requirements offer "state that the caller owns it, or refuse in the type";
this takes the first, keeping the release commit comment-only and zero-risk — refusing in the
type is a behaviour change and belongs in its own PR):
~~~~go
// The label is whatever the caller totals by — the token type, for this file's
// only caller today. Nothing here knows what a token is: this is the format
// layer. It is not, however, self-contained: it assumes every value handed to
// it is a count, and enforces that nowhere. A negative value is clamped to
// zero rather than refused, because a run keeps the greatest total reported
// and an unplaced point keeps the greatest it has seen, both of which start at
// zero. The caller owns the refusal — claude_code.go drops a point whose value
// is not a count before add is reached — so a second OTLP-speaking adapter
// counting something that can legitimately decrease must either refuse such a
// value itself or change this type, not reuse it as is.
~~~~

---

## 5. The RELEASE_NOTES v0.4.2 entry — final prose

Insert directly after `RELEASE_NOTES.md:2` (the blank line under `# Release notes`) and before
`## v0.4.1`. The file uses `## vX.Y.Z` with no date, a bold one-line summary, then bullets that
lead with a bold sentence.

~~~~markdown
## v0.4.2

**Cumulative token counts are right, and no audit script reports a verdict it could not compute.** Two fixes, one release. No new features, and no profile schema keys changed.

- **A cumulative export's totals change, in the direction of being correct.** Three defects in one merge each read as "fewer tokens than the session spent". A capture spanning several resources or scopes merged their series and let one running total replace another; a counter that restarted inside the capture discarded everything before the restart; and a flush reporting less than an earlier one on the same run took that run down with it. A profile written by 0.4.2 and one written by 0.4.1 from the same cumulative export will disagree. Delta temporality — Claude Code's default — is unaffected throughout, and `capability.adapter_version` moves to 0.4.2, which is what tells the two profiles apart.
- **A time series is what OTel says it is.** The resource it came from, the instrumentation scope that recorded it, the metric's name, and the data point's whole attribute set. The adapter keyed on the attributes alone, so two resources reporting cumulative 100 and 200 gave 200 rather than 300 — the same pair under delta gave 300 all along, which is how it hid.
- **`startTimeUnixNano` says which run of a counter a point is from, and it is now read.** The points sharing a start are one run and the run holds the greatest running total any of them reported; a point carrying a different start is a counter that restarted, and its run sits beside the earlier one instead of replacing it, so 100 followed by a restart reaching 20 is 120 and not 20. A start time of `0` is a start time that is absent — OTLP uses the protobuf JSON mapping, in which an explicit zero and an omitted field are the same message.
- **`timeUnixNano` is no longer read by the merge at all.** These are counters, so their values carry their own order and a flush reporting less than an earlier one on the same run is a capture contradicting itself, not tokens given back. Ordering by the timestamp meant that supplying the field which says what run a point is from could make the profile report *fewer* tokens: a run reporting 900, then 500, then 1000 came back as 500, and erasing its start time raised it to 1000.
- **A point that cannot say which run it is from joins the run it is cheapest to have come from.** It came from some run — one that named itself, or one nothing else in the capture observed — so it is neither a run of its own nor free. The series holds the least total its runs can account for: the runs added, plus whatever that point reported over and above the largest of them. Runs of 100 and 20 beside an unplaceable 500 give 520, not the 500 that drops a run which named itself and not the 620 that invents a third. One export flush that omits one field no longer reports a session at twice its size.
- **A series carrying both delta and cumulative points is refused rather than resolved.** They are opposite instructions, so every way of resolving the mix invents a number the export does not contain. The refusal is counted per series and named in the reason; the well-formed series beside it still count. What a `present` result cannot yet say is that a series was refused — schema v1 has no field for it, and that is tracked for 0.5.0.
- **No `skill-audit` script tells you a skill passed when it could not check.** With `jq` absent, `check-structure.sh --json` returned `{"findings": [], "passed": true}` at exit 0 and `check-paths.sh --json` died at exit 127 with no payload at all. With `skill-validator` absent, `check-frontmatter.sh` printed `frontmatter OK`. All three now exit 3, name the missing tool on stderr, and carry a `passed: false` payload. `--json` requires `jq` unconditionally; text mode is unchanged.
- **A child result a parent does not recognise is an execution error, not a pass.** `check-structure.sh` read `check-paths.sh`'s 127 as success, and read an unparseable payload as "no findings". Both parents now enumerate their child's documented statuses and treat anything else as an execution error, and `passed` is derived from the computed verdict in one place instead of being hard-coded on the zero-findings branch.
- **The report composer proves a source's shape before reading a verdict out of it.** `audit-report.sh` captured its guarded child with `2>&1`, so a failing payload never parsed and the composed report said `{"passed": true, "total_findings": 0}`. And a source emitting `0`, `[1,2]`, `true` or a two-document stream took `jq` down with no report at all. One guard now owns "exactly one document", each source claims only the shape it is actually read as, and a source yielding nothing readable is named in `policy_error` with `summary.passed` false.
- **A broken guard file is an execution error, not a policy failure.** A malformed `verdict-guard.sh` exited 2 — the status reserved for "policy failure" — from every script. Each script now checks the load on both sides and exits 3. Across every fixture, script, mode and masked tool, nothing exits outside `{0, 1, 2, 3}`.
- **The JSON encoder no longer depends on your locale.** It escaped through a character class that under a UTF-8 locale matched bytes with negative ordinals, corrupting them into a sixteen-hex-digit escape — including the valid C1 controls, which JSON asks no one to escape. The decision is the ordinal now, so the same bytes come out under `C` and under UTF-8.
- **`DEP001`, `DEP002` and `PATH` are registered rule IDs.** They are in the skill's rule-ID enumeration, in every emitting script's header, and in the README. `scripts/lib/` was flattened, which clears the `deep nesting detected` warning the skill had been carrying, and the layout suite now asserts both shipped skills validate with zero errors *and* zero warnings.
- 66 profiler test functions and 230 subtests pass under `-race`, against 0.4.1's 58 and 179. The shell suites are at 850, against 0.4.1's 134, and were run under both `LC_ALL=C` and a UTF-8 locale. 55 OTLP fixtures, against 39.
- **The two known limits named in v0.4.1 are fixed here, and that note needs two corrections.** Its heading said "both 0.4.2" and its body said "fixed together in 0.5.0" — one of those was always wrong, and 0.4.2 is the right one. And "Both are described in the README and the spec" is no longer true: fixing them deleted the paragraph that described them. The v0.4.1 note is left as written; this is the correction.
~~~~

---

## 6. Deferred items, carried forward

### 6a. The four the requirements mandate — they belong in the CHANGELOG's known-limits block (§2)

| item | reason it is deferred |
|---|---|
| **`isMonotonic` is unread.** A `claude_code.token.usage` sum declaring `isMonotonic: false` with a genuinely decreasing run reports the greatest value where v0.4.1 reported the latest — 1000 against 200 in the reproduced case. | Judged safe for this metric: greatest-wins cannot over-report a monotonic cumulative sum, the OTel data model says a reader should expect non-decreasing values, every fixture in the repo declares `true`, and `claude_code.token.usage` is a Counter. Same class as `flags`, and belongs beside it. |
| **`flags` is unread**, so a stale non-zero `NO_RECORDED_VALUE` point still reads as a running total. | Fixing it is a new leaf on the data point, a new refusal class, its own reason clause, fixtures and a spec paragraph — the size of a feature. No capture route the README documents produces one today (Claude Code's SDK does not emit it for sums). Named as the next one to take. |
| **A `--json` exit 3 does not always carry a payload**: the usage branches, the target-not-found paths, and the guard-load failure. | Deliberate, not a gap. There is no skill to render a verdict about on those paths, and minting rule IDs for them would enlarge the consumer-observable surface to describe caller errors rather than verdicts. Documented in both script headers and the README, and pinned by tests. Named here so it is not rediscovered as a defect. |
| **Schema v1 has no channel to report that a malformed series was excluded from a `present` total.** | A `present` result carries no reason in v1. Adding one inside v1 is a schema change wearing a bug fix's clothes. The caveat channel is the 0.5.0 item the spec already reserves, and this belongs beside the skipped-data-point count. |

### 6b. Found-and-not-fixed by the scripts team — **not** in the requirements' list

Recorded in `.scuba/teams/release-0.4.2-scripts/status.md` rounds 3 and 4. I have **not** put
these in the changelog, because the requirements document does not carry them forward and I am
not extending its list on my own judgement. **Question for the manager:** do these belong in
the 0.4.2 known-limits block, or do they stay in the team status?

| item | reason it is deferred |
|---|---|
| `tests/test_f01.sh`'s rule-ID census is syntax-shaped, so the guard is weaker than the sentence it defends. `rules_emitted_by` recognises four literal forms; a quoted rule or one built from variables extracts to nothing, and `DEP001` is attributed only to `verdict-guard.sh:136`, never to the four scripts that emit it indirectly through `require_tool`. | Set equality holds today because every call site uses a recognised form. Verified against the tree, not taken on report. Recorded as directed, not fixed. |
| `check-quality.sh` forwards `skillscore`'s own exit status rather than mapping it, so *every caller error* — a bad flag, a non-directory target, a path that does not exist — reaches the caller as exit 1 where its header documents `{0, 3}`. | It is the one script in the skill PR #4 does not touch; it already guards `skillscore` inline with an install hint the README quotes, so it is not an instance of the class that PR closed. A header wrong about an untouched script was not worth widening the final round's diff for. Still inside `{0,1,2,3}`. |
| Both `--json` writers spawn a `jq` per finding; a 4,000-finding payload takes about 95 seconds. Pre-existing, and now on a hotter path. | A performance change, not a correctness one. It wants its own change. |
| `tests/test_skill.sh` has no `cd "$(dirname "$0")/.."`, unlike the other three suites, so it fails when invoked by absolute path from elsewhere. Pre-existing. | Out of PR #4's scope and left alone. **This one has an operational consequence for §8: the apply checklist must run `tests/test_skill.sh` from the repo root.** |

---

## 7. The commit message — final text

**Identity.** Local `git config user.name` is `imagineux` and `user.email` is
`imagineux@gmail.com`, so a plain `git commit` produces author *and* committer
`imagineux <imagineux@gmail.com>` with no override flag. Do **not** pass `--author`,
`-c user.email=`, `GIT_AUTHOR_*` or `GIT_COMMITTER_*`. **No `Co-Authored-By` trailer and no
AI attribution of any kind.** (For the record: `2ed34b8`'s author name is
`Matthew Van Dusen <imagineux@gmail.com>` and its committer is `GitHub <noreply@github.com>`,
because it was squash-merged through the GitHub UI. This commit is made locally, so both
fields will read `imagineux <imagineux@gmail.com>` as required.)

Style matches `2ed34b8`: `release: vX.Y.Z — <summary>` subject, prose body wrapped at 72,
no backticks, `-` in place of em-dashes inside the body, closing counts line.

~~~~text
release: v0.4.2 — correct cumulative token counts, guard every verdict

Two fixes, one release. No new features, and no profile schema keys
changed.

The profiler's cumulative token merge was wrong in three ways that each
read as fewer tokens than the session spent. A time series was keyed on
the data point's attributes alone rather than on resource, scope, metric
and attributes, so a capture spanning several resources merged their
series and let one running total replace another: 100 and 200 gave 200.
startTimeUnixNano, which says which run of a counter a point is from,
was not read at all, so a counter that restarted inside one capture
discarded everything before the restart: 100 then a restart reaching 20
gave 20. And the merge answered "which point wins" with timeUnixNano,
so a flush reporting less than an earlier one on the same run took that
run down with it - which is how supplying the field that identifies a
run came to lower the total.

The model underneath is now stated and enforced. A run holds the
greatest running total its points reported, because these are monotonic
counters and the values carry their own order. A series holds the least
total its runs can account for: the runs added, plus whatever a point
that cannot name its run reported over and above the largest of them.
That point came from some run - one that named itself, or one nothing
else observed - so counting it as a run of its own invents mass, and
dropping the series to the greatest total seen anywhere on it throws
away runs the point did not join. Runs of 100 and 20 beside an
unplaceable 500 give 520. A start time of 0 is a start time that is
absent, per the protobuf JSON mapping. A series carrying both delta and
cumulative points is refused whole and counted, because every way of
resolving opposite instructions invents a number the export does not
contain.

Totals for the same cumulative export therefore differ from 0.4.1, in
the direction of being correct. Delta temporality, which is Claude
Code's default, is unaffected throughout.

The skill-audit scripts could not tell "this passed" from "I could not
check". With jq absent, check-structure.sh --json returned an empty
findings list and passed true at exit 0, and check-paths.sh --json died
at exit 127 with no payload. With skill-validator absent,
check-frontmatter.sh printed frontmatter OK. A new verdict-guard.sh owns
the one sentence behind all three: a reason on stderr, a passed-false
payload on stdout, exit 3. --json requires jq unconditionally and text
mode is unchanged. A child result a parent does not recognise is now an
execution error rather than a pass, passed is derived from the computed
verdict in one place, and the report composer proves each source's shape
before reading a verdict out of it - a source emitting 0, [1,2], true or
a two-document stream used to take jq down with no report at all. A
broken guard file is an execution error rather than a policy failure,
and the JSON encoder is settled by ordinal, so its output no longer
depends on the locale.

Three claims in the released 0.4.1 entries are superseded rather than
rewritten: that the adapter accepts every 64-bit integer as the OTLP
spec requires, which is false because the proto3 JSON mapping accepts
exponent notation and this adapter deliberately refuses it; and, twice,
that the README and the spec describe two known limits, which they no
longer do because this release fixed them. The v0.4.1 release note also
said 0.4.2 in its heading and 0.5.0 in its body; 0.4.2 is right.

Carried to 0.5.0 and recorded: isMonotonic and flags are unread, a
--json exit 3 deliberately carries no payload on three caller-error
paths, and schema v1 has no field to say a malformed series was left out
of a present total.

The version moves on all six surfaces: four plugin manifests, the
marketplace manifest, and the profiler's AdapterVersion, with the
layout suite's asserts and the README's two references.

66 profiler test functions, 230 subtests, 850 shell assertions, 55 OTLP
fixtures.
~~~~

---

## 8. Apply checklist

**Tagging and pushing are the user's call, not the agent's.** Steps 9 and 10 are written out
so they are ready, and they are the user's to run. No agent merges to `main` and no agent
pushes a tag.

### Step 0 — preconditions

1. Both PRs are squash-merged to `main` on GitHub by the user.
2. **Resolve OPEN-1 first.** The local `main` has diverged and its working tree is dirty; the
   F03/F04 work must be preserved on a branch before `main` is reset to `origin/main`.
3. `jq`, `skill-validator` and `skillscore` are installed and on `PATH` (the shell suites
   need them; CI installs `skill-validator` v1.6.1 and `skillscore` via npm).
4. Go toolchain matching `profiler/go.mod`.

### Step 1 — get to the real post-merge main

```
cd /Users/matthewvandusen/Development/Auraprix/skill-architect
git status --porcelain            # expect empty once OPEN-1 is resolved
git fetch origin --prune
git checkout main
git reset --hard origin/main      # only after OPEN-1's work is safely on a branch
git log --oneline -4              # expect the two squashes on top of 2ed34b8
```

### Step 2 — verify the base is what this draft assumed

```
git rev-parse HEAD^^              # should be 2ed34b8 if exactly two squashes landed
grep -rn '0\.4\.1' --include='*.json' --include='*.go' --include='*.sh' --include='README.md' \
  . | grep -v testdata
```
Expect exactly the 16 lines of §1 plus the history lines listed at the end of §1. **If the
count differs, stop and report — do not edit past a surprise.**

### Step 3 — re-locate every surface before editing

```
grep -n '"version": "0.4.1"' .claude-plugin/plugin.json .codex-plugin/plugin.json \
  .cursor-plugin/plugin.json .devin-plugin/plugin.json .claude-plugin/marketplace.json
grep -n '0\.4\.1' profiler/types.go profiler/profiler_test.go tests/test_skill.sh
grep -n '0\.4\.1' README.md
```
Confirm §1's numbers; §1d's `README.md:327` in particular is a computed prediction.

### Step 4 — apply the 16 version-surface edits (§1)

Five manifests, `profiler/types.go:233-234`, `profiler/profiler_test.go:408,410`,
`tests/test_skill.sh:28,48,51,57,58`, `README.md:55` and `README.md:327`.

### Step 5 — apply the two comment corrections (§4)

`profiler/profiler_test.go:1235-1237` and `profiler/otlp.go:843-846`.

### Step 6 — apply the prose (§2, §3, §5)

CHANGELOG 0.4.2 section between `## Unreleased` and `## 0.4.1 — 2026-09-13`, including the
three supersessions under `**Known past changes, recorded late**` and the known-limits block.
RELEASE_NOTES `## v0.4.2` entry above `## v0.4.1`. Set the CHANGELOG date to the commit date.

### Step 7 — verification suite (run all of it, before committing)

```
# Go: the four commands CI runs, in CI's order.
cd profiler
unformatted="$(gofmt -l .)"; [ -z "$unformatted" ] || { echo "$unformatted"; exit 1; }
go build ./...
go vet ./...
go test -race -count=1 ./...
cd ..

# Counts, to confirm the figures in the CHANGELOG, RELEASE_NOTES and commit message.
cd profiler && go test -v -count=1 ./... 2>&1 | grep -cE '^--- PASS:'        # expect 66
cd profiler && go test -v -count=1 ./... 2>&1 | grep -cE '^ +--- PASS:'      # expect 230
ls profiler/testdata/otlp | grep -v README | wc -l                           # expect 55

# Shell: all four suites, FROM THE REPO ROOT (test_skill.sh has no cd of its own).
tests/test_skill.sh
tests/test_walk.sh
tests/test_f01.sh
tests/test_f02.sh
```
Expect `test_skill` 27, `test_walk` 19, `test_f01` 561, `test_f02` 243 — **850** assertions,
0 failures. `test_skill.sh` must now pass its version asserts against 0.4.2; if it still reads
0.4.1 anywhere, step 4 missed a line.

Re-run the four shell suites under the other locale, since one of the defects this release
fixes was visible only under one of them:
```
LC_ALL=C tests/test_f01.sh && LC_ALL=C tests/test_f02.sh
LC_ALL=en_US.UTF-8 tests/test_f01.sh && LC_ALL=en_US.UTF-8 tests/test_f02.sh
```

Skill validation, which `test_skill.sh` now asserts but is worth seeing directly:
```
skill-validator check skills/skill-audit -o json    # passed true, errors 0, warnings 0
skill-validator check skills/skill-rewrite -o json  # passed true, errors 0, warnings 0
```

**`skillgate` is deliberately absent from this suite — see OPEN-2.**

### Step 8 — commit

```
git add -A
git status                      # review: expect exactly the files §1-§6 name, nothing else
git commit -F <message-file>    # the §7 text, written with an editor, not a heredoc
git log -1 --format='%an <%ae> | %cn <%ce>'   # must be: imagineux <imagineux@gmail.com> | imagineux <imagineux@gmail.com>
git log -1 --format='%B' | grep -ci 'co-authored-by\|generated with\|claude\|anthropic'  # must be 0
```
No `--author`, no `-c user.email=`, no `GIT_AUTHOR_*`/`GIT_COMMITTER_*`.

### Step 9 — tag (USER'S CALL)

`v0.4.1` is an **annotated** tag with tagger `imagineux`, subject
`v0.4.1 — parse real OTLP/JSON, and make 0.4.0's claims true`. Mirroring it:
```
git tag -a v0.4.2 -m 'v0.4.2 — correct cumulative token counts, guard every verdict'
git show v0.4.2 --stat | head -20
```

### Step 10 — push (USER'S CALL)

```
git push origin main
git push origin v0.4.2
```

Only the user pushes to `main` and only the user pushes the tag. An agent running this
checklist stops after step 8 and reports.
