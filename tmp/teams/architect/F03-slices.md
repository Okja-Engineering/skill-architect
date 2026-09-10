# F03 — verifiable slice plan

**End-to-end outcome:** A harness-agnostic profiler that captures runtime signals (tokens, tool calls, timing, attribution) from any of the four target harnesses (Cursor, Claude Code, Codex, Devin), degrades gracefully to `unknown` for unavailable metrics, and produces a serialized profile pinned to a snapshot hash that F04 can read and recompute.

**Major risks:**
1. OTel export reliability in practice (Codex `codex exec` has a known metric bug — tokens only in span/log attrs, not metrics). — *Local hypothesis*
2. Devin ATIF export schema is unpublished — token counts may not be present. — *Local hypothesis*
3. Skill activation events are Cursor-only — F04 cannot depend on skill-level activation for 3/4 harnesses. — *External evidence*
4. Over-engineering the adapter interface before F04's actual needs are known. — *Design risk*

---

## Slice 1: Adapter interface + Claude Code adapter + profile serialization

- **Boundary:** Define `MetricResult<T>` (present/unknown/error), `CapabilityReport`, and `ProfilerAdapter` interface. Implement the Claude Code adapter using OTel export. Define and implement the serialized profile format (JSON, pinned to snapshot hash, carries CapabilityReport). Exclude: other harness adapters, F04 comparison engine.
- **Depends on:** User confirmation of Option C.
- **Change:** The system can capture a real Claude Code session's runtime signals and produce a serialized, reparseable profile with honest capability reporting.
- **Intermediate state:** One working adapter proves the interface is viable. The profile format is exercisable end-to-end. No other harness is supported yet, but the system is useful for single-harness analysis.
- **Acceptance evidence:** Run a Claude Code session with `CLAUDE_CODE_ENABLE_TELEMETRY=1` and OTel export configured. The adapter produces a profile where: token counts are `present`, tool calls are `present`, skill activation is `unknown` (Claude Code has no skill-level events), timing is `present`. The profile serializes to JSON and reparses with the CapabilityReport intact. A fixture with no OTel data produces a profile where all metrics are `unknown` with reason "no telemetry source detected."
- **Integration points:** OTel collector or file-based OTel export → adapter → profile JSON. The profile JSON is the integration point for F04.
- **Rollback:** Adapter is additive — no existing code changes. Remove the adapter and profile format if the interface proves wrong.

**Why Claude Code first:** Best-documented OTel support, local (no enterprise plan needed), simplest to test in CI. If OTel doesn't work in practice, we learn immediately on the easiest harness.

---

## Slice 2: Cursor adapter (OTel + SQLite fallback)

- **Boundary:** Implement the Cursor adapter. Uses OTel enterprise export when available (token counts, `skill.activated` events). Falls back to SQLite `state.vscdb` session data (composers/bubbles with token estimates) when OTel is not configured. Exclude: Codex, Devin adapters.
- **Depends on:** Slice 1 (adapter interface must exist).
- **Change:** The system can capture Cursor session signals, including skill activation events (Cursor-only capability). Tests the interface against a different telemetry surface with a fallback path.
- **Intermediate state:** Two adapters working. The interface is proven against two different surfaces (OTel-only vs OTel+SQLite). Skill activation is now `present` for Cursor, `unknown` for Claude Code — the capability difference is visible in profiles.
- **Acceptance evidence:** Run a Cursor session with OTel export configured. Adapter produces a profile where: tokens `present`, tool calls `present`, skill activation `present` (Cursor emits `skill.activated`), timing `present`. Run without OTel — adapter falls back to SQLite, tokens `present` (estimated), skill activation `unknown` (SQLite has no activation events). Capability report differs between the two modes.
- **Integration points:** OTel export → adapter; SQLite `state.vscdb` → adapter. Same profile JSON output as Slice 1.
- **Rollback:** Adapter is additive. Remove if Cursor's OTel or SQLite surface proves unusable.

---

## Slice 3: Codex adapter (OTel logs + hooks)

- **Boundary:** Implement the Codex adapter. Uses OTel log events for token counts (extracted from `codex.sse_event` span/log attributes, not metrics — working around the `codex exec` metric bug). Uses lifecycle hooks for tool-call attribution. Exclude: Devin adapter.
- **Depends on:** Slice 1 (adapter interface). Can run in parallel with Slice 2.
- **Change:** The system can capture Codex session signals. Tests the interface against a telemetry surface where the primary metric instrument is broken and data must be extracted from log/span attributes instead.
- **Intermediate state:** Three adapters working. The interface is proven against three different surfaces, including one that requires log-attribute parsing rather than metric reading. This validates the graceful-degradation contract against a real-world broken-metric scenario.
- **Acceptance evidence:** Run `codex exec` with OTel configured. Adapter produces a profile where: tokens `present` (from `input_token_count`/`output_token_count` log attributes on `codex.sse_event`), tool calls `present` (from `codex.tool_decision` events), skill activation `unknown` (Codex has no skill-level events), timing `present`. If the metric bug is fixed upstream, adapter reads the metric directly; if not, it falls back to log attrs. Capability report notes the source.
- **Integration points:** OTel logs/spans → adapter. Hooks (optional supplement) → adapter. Same profile JSON output.
- **Rollback:** Adapter is additive. Remove if Codex OTel proves unusable in practice.

---

## Slice 4: Devin adapter (ATIF export + server API)

- **Boundary:** Implement the Devin adapter. Parses ATIF export files (`--export` flag) for per-step telemetry and timing. Falls back to enterprise server API (`/v3/sessions/{id}/insights`) when available. Exclude: nothing — this completes the adapter set.
- **Depends on:** Slice 1 (adapter interface). Can run in parallel with Slices 2 and 3. **Risk gate:** requires a real ATIF export to inspect (open question 2). If ATIF lacks token counts, adapter reports tokens as `unknown` with reason "ATIF export does not include token data."
- **Change:** The system can capture Devin session signals from a completely different surface (file export + server API, no OTel, no hooks). Tests the interface against the outlier harness.
- **Intermediate state:** All four adapters working. The interface is proven across the full ecosystem range: OTel-only, OTel+SQLite, OTel-logs+hooks, ATIF+API. Graceful degradation is exercised in every direction.
- **Acceptance evidence:** Parse a real Devin ATIF export. Adapter produces a profile where available metrics are `present` and unavailable metrics are `unknown` with reason. If enterprise API is available, adapter enriches with server-side insights (ACUs, message counts). Capability report honestly reflects what ATIF vs API provides.
- **Integration points:** ATIF file → adapter. Server API (optional) → adapter. Same profile JSON output.
- **Rollback:** Adapter is additive. If ATIF proves too sparse, adapter reports all metrics `unknown` — honest degradation, not a failure.

---

## Dependency graph

```
Slice 1 (interface + Claude Code + serialization)
  ├── Slice 2 (Cursor: OTel + SQLite)
  ├── Slice 3 (Codex: OTel logs + hooks)
  └── Slice 4 (Devin: ATIF + server API)
```

Slices 2, 3, 4 are independent of each other and can proceed in parallel once Slice 1 lands. Slice 1 is the critical path — it proves the interface, the serialization, and the first real telemetry capture.

## What F04 needs from this plan

F04 (paired comparisons) can start after **Slice 1 only** — a single-harness comparison (same harness, with-skill vs without-skill) needs just one adapter and the serialized profile. Additional adapters expand the comparison surface but are not prerequisites for F04 to begin.

## Quality test results

- Each slice lands without unfinished hidden dependencies: ✅ (each adapter is self-contained)
- Evidence can run before later slices exist: ✅ (each adapter is independently testable)
- Acceptance criteria describe behavior, not task completion: ✅ (produce a profile with specific metrics present/unknown)
- Boundary is small enough for one coherent review: ✅ (one adapter per slice)
- Parallel execution cannot produce conflicting ownership: ✅ (each adapter owns its harness; profile format is frozen in Slice 1)

---

## Slice 1.1: Claude Code OTel probe validation and tool-call semantics

- **Boundary:** Harden the existing Claude Code OTel adapter in `profiler/claude_code.go` and `types.go`. `Probe` must parse the export file and confirm at least one recognizable event (`claude_code.token.usage` metric or `claude_code.tool_decision`/`claude_code.api_request` log) before declaring `otel` for tokens, tool_calls, or timing. Decide and document the meaning of `ToolCallEntry.Success` (permission vs execution) and either populate `Duration` from adjacent timestamps or remove `Duration` from `ToolCallEntry` and the profile schema. No new adapter sources, no new harnesses, no F04 changes.
- **Depends on:** Slice 1 (types and adapter exist) and the [F03-dogfood-decisions.md](F03-dogfood-decisions.md) memo.
- **Change:** The Claude Code adapter's `Probe` is honest about what it can actually read; `Capture` no longer reports `present` on empty or malformed files.
- **Intermediate state:** `Probe` returns `none` for an empty or malformed OTel file; the profile either carries `duration_ms` for every tool call or no `duration_ms` field at all.
- **Acceptance evidence:** An empty `claude-code-otel.json` yields all metrics `unknown` or `error` (not `present`). The real 2026-09-10 OTel fixture still produces `present` tokens, tool_calls, and timing. If `Duration` is kept, every `ToolCallEntry` has a non-negative value; if dropped, the round-trip JSON no longer contains `duration_ms`.
- **Integration points:** Same as Slice 1 — OTel file → adapter → `Profile` JSON.
- **Rollback:** Revert `claude_code.go` to the Slice 1 implementation; add back `Duration` if it was removed.

## Slice 1.2: Claude Code session_data (JSONL transcript) adapter

- **Boundary:** Add a `session_data` source for Claude Code that reads the local JSONL session transcript (`~/.claude/projects/<project>/<session>.jsonl`). The adapter must sum usage from the last record per `message.id` to preserve reasoning tokens, extract `tool_result.is_error` for execution success, and compute timing from request/result timestamps. It produces the same `Profile` JSON as the OTel adapter. Exclude: other harnesses, F04 changes.
- **Depends on:** Slice 1.1 (probe/tool semantics resolved) and the F03 telemetry-source decision.
- **Change:** Claude Code sessions can be profiled without a collector or telemetry env vars.
- **Intermediate state:** Claude Code has two source modes (`otel` and `session_data`) and the same serialized profile output.
- **Acceptance evidence:** Run `profiler` with `--export-file <session>.jsonl` from the 2026-09-10 run. The profile reports `tokens`, `tool_calls`, `timing` as `present` with `source: "session_data"`. `skill_activation` and `attribution` are `unknown` with documented reasons. The 2026-09-10 OTel fixture still passes.
- **Integration points:** JSONL transcript → adapter → `Profile` JSON.
- **Rollback:** Remove the transcript source and leave only the OTel adapter.

## Slice 1.3: Claude Code skill attribution / session boundary (proposed, conditional)

- **Boundary:** If the human chooses to isolate per-skill cost in Claude Code, add an attribution/session-boundary mechanism. Options: hooks (`SessionStart`/`PreToolUse`/`Stop`) carrying the active skill name, or transcript context injection. The mechanism would emit `SkillActivation` and `Attribution` results for the named skill. This slice is not started if the F04 with-skill/without-skill design is chosen.
- **Depends on:** F03-dogfood-decision 1; Slice 1.2 (transcript source likely needed).
- **Change:** F03 can attribute token and tool-call usage to a named skill in Claude Code.
- **Intermediate state:** A Claude Code session with the skill active produces a profile with non-empty `Attribution` or `SkillActivation`.
- **Acceptance evidence:** A "with-skill" session has `Attribution` mapping tool calls to the skill, or `SkillActivation` `present` with a start/stop pair. A "without-skill" control session differs materially.
- **Integration points:** Hooks or transcript context → adapter → `Profile` JSON.
- **Rollback:** Remove the attribution source; all Claude Code profiles return to session-level.

## Quality test results for new slices
- Each new slice lands without unfinished hidden dependencies: ✅
- Evidence can run before later slices exist: ✅
- Acceptance criteria describe behavior, not task completion: ✅
- Boundary is small enough for one coherent review: ✅
- Parallel execution cannot produce conflicting ownership: ✅
