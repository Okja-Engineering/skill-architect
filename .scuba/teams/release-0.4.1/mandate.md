# Mandate — v0.4.1 patch: make 0.4.0's promises true

_Chief of staff, 2026-09-13. Approved plan: ~/.claude/plans/determine-the-next-menaingful-glistening-horizon.md §0.4.1. Evidence: 52-promise ledger from 0.4.0 docs at origin/main._

## Goal
Ship v0.4.1 of skill-architect to Okja-Engineering/skill-architect as one PR from `origin/main` (30f374c). It contains only things 0.4.0 promised and did not deliver, or documented wrongly. Nothing new.

## Hard boundaries
- Base is `origin/main` at 30f374c. NOT local main (7 unpushed commits are 0.5.0 scope). Do not cherry-pick them.
- Touch nothing under `skillgate/`, `skills/skill-gate/`, `docs/skillgate-*`, `docs/research/`, `.scuba/`, `go.work`.
- Do NOT rename `cache_creation` → `cache_write` (breaking schema change, 0.5.0).
- Do not soften the 0.3.1 dependency guards (`exit 1` on missing skill-validator/skillscore stays). Document the deps instead.
- Never a draft PR. User merges; you never merge to main.
- Work in your own worktree: `git worktree add /Users/matthewvandusen/Development/Auraprix/skill-architect-wt/release-0.4.1 -b release/0.4.1 origin/main`. Do not touch the primary tree at /Users/matthewvandusen/Development/Auraprix/skill-architect except this .scuba/ directory.

## Definition of done (each item verified, cite file:line in status)

### Bug
1. `profiler/claude_code.go:87` — Capture gates the whole OTel path on `Capabilities[MetricTokens] == SourceNone`. A partial export with `tool_decision` + `api_request` but no `token.usage` returns "OTel export not configured" and discards tool-call and timing data that Probe advertised. Fix: gate on "no capability at all", return tokens `unknown` with reason while tool_calls/timing are `present`. Test first: fixture with tool calls + timing, no token metric → RED (tool_calls/timing unknown) → fix → GREEN → revert → RED → restore. Pin the test to the invariant (each signal Probe reports available is present in Capture), not to the patch.

### Install truth
2. `README.md` gains a Prerequisites block: `skill-validator` (brew, agent-ecosystem tap), `skillscore` (npm -g), `jq`; state what fails without each. Verify by running the documented Quick example in a shell with the tools masked from PATH (expect the documented loud failure naming them) and unmasked (expect success).
3. `README.md` manual-copy section: `skill-rewrite` requires a sibling `skill-audit` directory (draft-rewrite.sh resolves `../skill-audit`). Say so.
4. `profiler/go.mod` module path `github.com/auraprix/skill-architect/profiler` → `github.com/Okja-Engineering/skill-architect/profiler`; update `profiler/cmd/main.go` import. `go build ./... && go vet ./... && go test ./...` green.
5. `.github/workflows/ci.yml:34` `go-version: "1.23"` vs `profiler/go.mod` `go 1.27.1`. Set CI to `go-version-file: profiler/go.mod` (do NOT reference go.work; it is not in this release).

### Spec truth (`docs/profiler-spec.md`)
6. Replace the `MetricResult[T any]` generic block (~:24, :114-118) with the real shape in `profiler/types.go` (`RawMetricResult` + `TokenResult`/`ToolCallResult`/… wrappers). Verify: a throwaway `_scratch/adapter_test.go` that compiles against the documented types, then delete it.
7. AC2 (~:195) describes pre-30f374c probe semantics; align with the probe-logic section (~:173-178): each signal is reported available only when present in the export.
8. Fallback reason string (~:190) names env vars this adapter ignores; match `profiler/claude_code.go:88` wording.
9. `CaptureOpts.OtelEndpoint` (`profiler/types.go:59`, spec ~:92, :181) is never read and has no flag. Delete from struct and spec.
10. "Snapshot-pinned" (RELEASE_NOTES.md:9, spec) overstates: nothing hashes or validates `--skill-dir`. Reword to what it does (echoes the caller-supplied snapshot id). Do not add hashing.

### Changelog / release-notes truth
11. `CHANGELOG.md:15` / `RELEASE_NOTES.md:11` "14 profiler tests" → 15 (30f374c added one). Recount after item 1 and state the new number.
12. `CHANGELOG.md:17` cites `tmp/teams/architect/profiler-design-space.md` (gitignored). Remove the path reference.
13. CHANGELOG 0.3.1 (~:24-25) says checklists and scoring table are inline in skill-audit SKILL.md; 3e2efd8 moved them to `references/`. Add a note under 0.4.1 (do not rewrite 0.3.1 history).
14. `ToolCallEntry.Duration` was removed from schema v1 in 30f374c with no entry. Add one under 0.4.1 as a known past change.
15. `CHANGELOG.md:12` / `RELEASE_NOTES.md:8` "Claude Code has no skill-level events" — soften to "the Claude Code adapter does not yet read skill-level attributes" (skill.name exists on token/cost metrics; reading it is 0.5.0).

### Release surfaces
16. Four plugin manifests `version` → `0.4.1`; `tests/test_skill.sh:28` assert → `0.4.1`.
17. `CHANGELOG.md`: new `## 0.4.1 — <date>` section listing items 1–15; `## Unreleased` stays empty above it. `RELEASE_NOTES.md`: `## v0.4.1` section.
18. Undocumented-but-delivered, document in README profiler section: `profiler version` subcommand, `capture --export-file` flag, per-signal probe detection.


### Item 19 — added by user decision 2026-09-13: parse real OTLP/JSON
The Claude Code adapter must read OTLP/JSON as Claude Code actually emits it (`resourceMetrics[].scopeMetrics[].metrics[]` and `resourceLogs[].scopeLogs[].logRecords[]`, real attribute keys such as `type` ∈ input/output/cacheRead/cacheCreation, `decision` on `tool_decision`), not the bespoke `{"metrics":[...],"logs":[...]}` envelope. Ground truth: `otlp-research.md` in this directory (researcher output). Build: a senior-implementer, after round-1 repair lands, against a plan derived from that document. Definition of done: a realistic OTLP/JSON export from the documented route yields tokens, tool_calls, timing `present` with real values; the bespoke envelope is dropped unless the research gives a reason to keep it; the "reasoning tokens" claim is removed unless the type exists; README/spec/RELEASE_NOTES describe the documented capture route and the real format; contract test covers real-shape fixtures; probe and capture share one predicate per signal. Sequencing: do not start until round-1 repair is pushed (same file, one writer).

**Item 19 decisions (chief of staff, 2026-09-13 16:12, defaults the user may override):** plan is `otlp-plan.md`. W4 resolved by item 19: `ToolCallEntry.Success` means "the tool ran and succeeded" (from `tool_result`); rejected calls are listed from `tool_decision` with success=false; spec, RELEASE_NOTES 0.4.1 and the AdapterVersion bump (W9) carry the semantic change; no JSON key changes, profile/v1 stays. skill_activation and attribution stay `unknown` with accurate reasons. Bundled receiver subcommand deferred to 0.5.0; README documents the collector file-exporter route. No live end-to-end run is possible without an authenticated session; the implementer states that in status.md rather than claiming one.

## Verification (all must pass in the worktree before the PR opens)
```
cd profiler && go build ./... && go vet ./... && go test ./...
tests/test_skill.sh && tests/test_walk.sh && tests/test_f01.sh && tests/test_f02.sh
```
Plus the item-2 masked/unmasked PATH run and the item-6 throwaway compile.

## Deliverable
- Commits on `release/0.4.1` in your worktree, conventional messages, each ending with `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>`.
- Push the branch; open a NON-draft PR against `main` titled `release: v0.4.1 — make 0.4.0's promises true`, body listing items 1–18 with evidence, ending with `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.
- Write `/Users/matthewvandusen/Development/Auraprix/skill-architect/.scuba/teams/release-0.4.1/status.md`: branch, worktree path, last SHA, PR URL, per-item done/evidence, anything skipped and why. Update it as you go, not only at the end.

## Quality bar
Every doc sentence you change must be true of the code at the PR head. Every code change has a test that fails without it. No scope beyond the 18 items; if you find another 0.4.0 gap, record it in status.md under "found, not fixed" and stop.
