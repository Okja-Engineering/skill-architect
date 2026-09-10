# F03 dogfood decision memo

**Run:** 2026-09-10 · SHA 541af3e · 11 API requests, 16 tool calls, 382 s · `tmp/profiler-run-2026-09-10/`

**Context:** Slice 1 acceptance evidence was met, but the dogfood run showed the Claude Code adapter is fragile and the captured profile is session-level. These decisions must be made before F03 implementation resumes.

## 1. Attribution / session boundary (finding 1)

| Option | What it means | Downstream for F04 |
|---|---|---|
| A — F04 absorbs it | Keep the profile per-session. F04 compares a full "with-skill" session against a full "without-skill" session. | F04 documents that cost deltas are session-level; no skill-level token subtraction. |
| B — F03 builds attribution | Add a session-boundary/attribution mechanism for Claude Code (hooks `SessionStart`/`PreToolUse`/`Stop` carrying skill name, or transcript context injection) and a F03 slice to capture it. | F04 can do skill-level subtraction; `Attribution` in the profile becomes meaningful. |

**Recommendation:** A. Claude Code does not emit skill-level events, and adding reliable attribution for a subagent session is invasive. The with-skill/without-skill design already provides a clean comparison without per-token attribution. Only choose B if a future F04 story needs to isolate the marginal cost of a skill within a mixed session.

## 2. Claude Code telemetry source (finding 4)

| Option | What it means | Downstream for F04 |
|---|---|---|
| A — Add `session_data` (JSONL transcript) adapter | Read `~/.claude/projects/<project>/<session>.jsonl` directly. No collector. Handles last-record-per-`message.id` dedup and `tool_result.is_error`. | Profiles can be produced from a normal Claude Code session; the comparison barrier drops. `Source` becomes `session_data` for Claude Code. |
| B — Keep OTel-only | Require `CLAUDE_CODE_ENABLE_TELEMETRY=1` and a collector/file export. The user must build `claude-code-otel.json` by hand. | Profiles are only available when telemetry is configured; in practice the comparison may never run. |

**Recommendation:** A. The run was only possible because the transcript exists. The OTel file itself was built from the transcript. Making the adapter read the transcript removes an operational blocker.

## 3. OTel probe validation (finding 3)

| Option | What it means | Downstream for F04 |
|---|---|---|
| A — Probe parses and validates the file | `Probe` opens the file, checks for at least one recognized metric or log event, and returns `none` for any source it cannot actually read. | Probe is trustworthy; F04 will not receive a `present` profile that is actually empty. |
| B — Leave as is | `Probe` uses `os.Stat` and reports `otel` whenever the file exists. Capture may later fail. | F04 may see `present` token counts that turn into `error` on recompute, comparing invalid profiles. |

**Recommendation:** A. This is a small change that restores the honesty of the capability report.

## 4. Tool-call `Success` and `Duration` (findings 5, 6)

| Option | What it means | Downstream for F04 |
|---|---|---|
| A — Keep `Success` as "not denied"; compute or drop `Duration` | `Success` stays the permission decision. Either remove `Duration` (it is unpopulated) or compute it from adjacent log timestamps if we keep it. | Tool calls are a count, not a correctness signal; F04 cost comparisons ignore these fields. |
| B — Use `tool_result.is_error` for `Success`; add `Duration` | Add transcript source so `Success` reflects execution. Compute `Duration` from `tool_result` timestamps when available. | F04 can compare tool-call failure rates and per-call latency, changing the comparison contract. |
| C — Add both fields | Keep `Success` as permission, add `ExecutionSucceeded` and `Duration` for execution. | Richer but wider profile surface; F04 must decide which fields to compare. |

**Recommendation:** A, with `Duration` dropped from `ToolCallEntry` and the profile schema. The current source does not support reliable execution success or latency; inventing approximate values is worse than honest omission. F04 has no stated need for tool-call duration.

## 5. Cache cost weighting (finding 2)

| Option | What it means | Downstream for F04 |
|---|---|---|
| A — Defer to F04 | F03 keeps raw counts (input, output, cache_read, cache_creation, reasoning). F04 defines the cost function. | F03 stays telemetry-only; F04 owns the business rule. |
| B — Add a cost-weighted `effective_cost` metric in F03 | F03 pre-computes a weighted sum. | Couples the profiler to pricing assumptions. |

**Recommendation:** A. `TokenCounts` already exposes cache fields. F04 should decide how to weight them.

## 6. skill-rewrite patch (findings 10, 11, 13)

| Option | What it means | Downstream for F04 |
|---|---|---|
| Approve the draft | Turn `skill-rewrite-draft.md` into a story and apply the listed `draft-rewrite.sh` + SKILL.md fixes. | No F04 impact. Improves the skills used to audit/rewrite future skills. |
| Reject the draft | Do not apply the rewrite; document why. | No F04 impact. |

**Recommendation:** Approve the draft. It contains concrete, low-risk fixes to `draft-rewrite.sh` and `SKILL.md` that match the audit findings.

## Approval requested

Reply with one of:
- `approve all` — implement the recommendations above.
- `override <number> <option>` — change a specific decision.
- `need more time` — no action until next session.
