# Mandate — v0.4.2 patch: the accumulator's time-series model

_Chief of staff, 2026-09-13. Placement decided by the user: these are bug fixes that produce wrong numbers with no schema change, so they are a patch, not 0.5.0 feature work. Cut from `origin/main` at `2ed34b8` (v0.4.1). NOT from local `main`, which carries 7 unpushed commits._

## Goal

Make the token accumulator model an OTel time series correctly, so a capture that aggregates several resources, or that spans a counter reset, reports the numbers the capture actually contains. One PR, both findings, because they are one root.

## Commit rules (override any default guidance)

- **No `Co-Authored-By` trailer** for Claude or any AI agent, on any commit.
- **Never** pass `-c user.email=`, `--author=`, `GIT_AUTHOR_EMAIL` or `GIT_COMMITTER_EMAIL`. Let git use the configured identity, `imagineux <imagineux@gmail.com>`.
- Before opening the PR: `git log origin/main..HEAD --format='%an <%ae>' | sort | uniq -c` must show exactly one identity, and `git log origin/main..HEAD --format=%B | grep -c 'Co-Authored-By'` must be 0.

## The root

`profiler/otlp.go` `metrics()` walks `resourceMetrics[].scopeMetrics[].metrics[]` and yields only the metric, discarding resource and scope. `profiler/claude_code.go` keys the accumulator on `dp.Attributes.seriesKey()` alone. `otlpDataPoint` has no `startTimeUnixNano` field. So the adapter's notion of a series is "the data-point attribute set", while the OTel data model identifies a series by resource + scope + metric + attributes, and a cumulative series additionally needs the start time to detect resets.

Both findings below were reproduced by the chief of staff against a binary built from 82ecdbb. Both require cumulative temporality, which is off Claude Code's default (delta, `aggregationTemporality: 1`, observed live), so neither blocked 0.4.1.

### F2 — independent resources overwrite each other under cumulative
Two `resourceMetrics` entries with distinct `service.instance.id`, identical data-point attributes, cumulative values 100 and 200 → `present {"input":200}`. Correct is 300. The same pair under **delta** correctly gives 300, so the defect is specific to the cumulative last-value-wins merge over a key that cannot tell the two series apart.

### F3 — a cumulative counter reset discards the earlier run
One resource, cumulative, `startTimeUnixNano` 1 with `timeUnixNano` 10 carrying 100, then `startTimeUnixNano` 11 with `timeUnixNano` 20 carrying 20 → `present {"input":20}`. The capture totals 120. A no-reset control (same start, 100 then 120) correctly gives 120.

## Invariants the fix must hold

1. A series is identified by resource identity, instrumentation-scope identity, metric name, and the full data-point attribute set. Two series that differ in any of those do not merge.
2. Under cumulative temporality, a data point whose `startTimeUnixNano` differs from the series' current start is a new run: the earlier run's contribution is retained, not replaced.
3. Delta behaviour is unchanged. Every existing fixture keeps its current result.
4. `profile/v1` output keys are unchanged. Only values become correct; no schema bump.
5. Resource and scope identity are derived the same way the series key already treats attributes: canonical, injective, and not collapsing distinct `AnyValue` kinds.
6. A data point whose `startTimeUnixNano` is absent or unreadable is handled by a stated rule, not by accident, and the rule is in the spec.

## Scope

- `profiler/otlp.go`: thread resource and scope identity through `metrics()`; add `startTimeUnixNano` to `otlpDataPoint` as a tolerant leaf like the others; extend the series key; implement the reset boundary in the cumulative merge.
- `profiler/claude_code.go`: pass the new identity into `acc.add`.
- Fixtures for each: multi-resource cumulative, multi-resource delta (control), reset, no-reset (control), absent start time, unreadable start time, multi-scope, and resource identity differing only by a non-string attribute value.
- Docs: replace the 0.4.1 known-limit sentences (which name these as deferred) with the behaviour now implemented, and update the spec's series-key and cumulative paragraphs. Leave CHANGELOG and RELEASE_NOTES alone entirely; the release commit writes their 0.4.2 sections.
- **Do NOT touch any version surface.** No plugin manifest, no `.claude-plugin/marketplace.json`, no `AdapterVersion`, no `tests/test_skill.sh` version asserts, no CHANGELOG/RELEASE_NOTES version heading. A separate release commit bumps all of them once, after this and the skill-audit script PR have both merged, so the two parallel PRs cannot conflict. Describe the fix in the PR body; the changelog entry is written at release time.

## Out of scope

The whole-export slurp (memory ceiling), thin `profiler/cmd` coverage, unpinned CI installs, `-h` exit status, the not-a-count bound wording, the tool-call `success` rename, and the unverified Devin/Codex/Cursor install rows. All stay on the 0.5.0 list.

**Now a sibling PR, running in parallel:** the two skill-audit script defects, where a failing skill is reported as passing when `jq` or `skill-validator` is absent (`check-structure.sh:71` swallows a missing jq so path findings vanish; `check-frontmatter.sh:24` treats exit 127 as success). Same silent-wrong-answer family, different subsystem. They ship in the same 0.4.2 release from their own branch; see `teams/release-0.4.2-scripts/mandate.md`.

## Definition of done

Both reproductions above return the correct totals; all four controls unchanged; every existing fixture's result unchanged; `go build`/`go vet`/`gofmt -l`/`go test -race` clean; four shell suites green; ship-gate CLEAN before the merge, which is the user's.

## Method

Test-first per finding, RED proven against the current code before the fix, then GREEN, then revert in a scratch copy and confirm RED again. One integration pass, not two bolt-ons: the identity threading and the reset rule land together because they edit the same call sites.
