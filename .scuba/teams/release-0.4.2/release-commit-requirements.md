# 0.4.2 release commit — what it must carry

_Chief of staff, 2026-09-14. Accumulated from the gates on PR #3 and PR #4. The release commit lands after both merge; neither PR was allowed to touch a version surface, so this is the one place they change._

## Version surfaces, all in lockstep
Four plugin manifests, `.claude-plugin/marketplace.json`, `profiler/types.go` `AdapterVersion`, and the `tests/test_skill.sh` asserts, all to **0.4.2**.

**`AdapterVersion` is not optional.** Profiles at both PR heads still stamp `0.4.1` while reporting different totals than 0.4.1 did for the same export. The invariant from the review: *no two profiles carrying the same `adapter_version` may report different totals for the same export.* Leaving it would make the field that exists to tell F04 which adapter produced an artifact stop distinguishing them.

## Supersede three false claims in the released 0.4.1 entries
All three are history and were correctly left untouched by both PRs. The 0.4.2 entry must explicitly supersede them:

1. `CHANGELOG.md:15` — "Every 64-bit integer is accepted as a JSON number or a decimal string, as the OTLP spec requires." False: the proto3 JSON mapping accepts exponent notation for 64-bit integer fields and this adapter deliberately refuses it. 0.4.2 records the deviation with its reason.
2. `CHANGELOG.md:47` — "`docs/profiler-spec.md` and `README.md` state two known limits of the token series key…" False at head: PR #3 implemented both and deleted that text.
3. `RELEASE_NOTES.md:20` — "Both are described in the README and the spec." Same, and its heading/body already disagreed with each other at 0.4.1 (heading said 0.4.2, body said 0.5.0).

## Two stale comments to correct while here
Both are comment-only, zero risk, and both teach a model that no longer exists. Folded here rather than spending another PR round.

- `profiler/profiler_test.go:1235-1237` — "Merged, the later point supersedes the earlier and the total is 200." The number is right; the mechanism is the deleted time-ordering model, and the sentence is false as stated: with the values reversed (200 then 100) head reports 200 while "later supersedes" would give 100.
- `profiler/otlp.go` `counterAccumulator` doc comment — it advertises the type as a reusable format layer ("a second OTLP-speaking adapter counting something else would use it unchanged"), but negative point values are clamped to 0 inside it rather than refused; the guarantee lives in the caller (`claude_code.go:291` refuses negatives before they arrive). Either state that the caller owns it, or refuse in the type.

## Changelog content
Describe both fixes as one release. The profiler half: a token series is identified by resource, scope, metric and attributes; a cumulative reset is read from `startTimeUnixNano`; a point that cannot identify its run sets a floor rather than adding mass; a mixed-temporality series is refused and counted. The scripts half: no skill-audit script emits a verdict a missing tool could not compute, `--json` requires `jq` unconditionally, and the composer no longer reads an unparseable child payload as "no findings".

Disclose the behaviour change plainly: totals for the same cumulative export differ from 0.4.1, in the direction of being correct.

## Carried to 0.5.0, record in the changelog's known-limits or the spec
- `isMonotonic` is unread. A `claude_code.token.usage` sum declaring `isMonotonic: false` with a genuinely decreasing run reports the greatest value where v0.4.1 reported the latest — 1000 against 200 in the reproduced case. Judged safe for this metric: greatest-wins cannot over-report a monotonic cumulative sum, the OTel data model says a reader should expect non-decreasing values, every fixture in the repo declares `true`, and `claude_code.token.usage` is a Counter. It is the same class as the `flags` item and belongs beside it.
- `flags` is unread, so a stale non-zero `NO_RECORDED_VALUE` point still reads as a running total.
- A `--json` exit 3 does not always carry a payload: the usage branches, the target-not-found paths, and the guard-load failure. Documented and pinned; named here so it is not rediscovered.
- Schema v1 has no channel to report that a malformed series was excluded from a `present` total. The caveat channel is the 0.5.0 item the spec already reserves.
