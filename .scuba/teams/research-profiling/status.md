# research-profiling — status

**stage:** done
**date:** 2026-09-10
**ledger:** `/Users/matthewvandusen/Development/Auraprix/skill-architect/.scuba/teams/research-profiling/ledger.md`

## Summary

- **Cursor's Enterprise OTel wire reference is public and verifies all four event names in `profiler/cursor.go`** (`cursor.token.usage`, `cursor.api.request`, `cursor.skill.activated`, `cursor.hook.execution_complete`) — nothing invented. But **every attribute key the adapter reads is wrong** (`token_type` → `cursor.token.type`, `skill_name` → `cursor.skill.name`, etc.), and `cursor.hook.execution_complete` is misused as a tool-call event when tool calls live in the `cursor.tool.calls` metric. Needs a correction story before F04 depends on Cursor.
- **Cursor hooks alone cannot give per-session token counts.** Of 21 hook events, only `preCompact` carries a real Cursor token number (`context_tokens`), and only on compaction. Best proxy is char/4 over hook payloads (accurate to −7%…+15% vs `o200k_base`, measured on this repo) but structurally undercounts billed input. Real fallback without Enterprise OTel is `POST /teams/filtered-usage-events` (per-request `tokenUsage`, joins on `conversationId`).
- **Do not adopt the OTel Go SDK for export.** Measured: `otlptracehttp` + `otlploghttp` at v1.46.0/v0.22.0 pulls 24 modules, compiles 66 gRPC packages, 408 packages, 17.7 MB binary — versus 0 deps / 6.4 MB for hand-rolled OTLP/JSON over stdlib. `profiler/go.mod` currently has zero dependencies.
- **Paired A/B differencing is the settled method** (SkillsBench 33.9%→50.5%; NVIDIA ACES "Skill Lift" 0.2134, 95% CI [0.1967, 0.2301], 947 paired cases). Token effects are **bimodal** — SkillEvaluator published one skill at −76.9% tokens and another at +120.3%. Report Skill Lift with a CI, and be able to say "this skill hurt."
- **Anthropic shipped `/skill-doctor`** (Claude Code ≥2.1.252) giving per-skill context cost and invocation counts — the only first-party per-skill token accounting anywhere. **Local Claude Code is 2.1.221, so it could not be tested; whether `-p` output is parseable is the highest-value open spike.**

## Escalations

- **Needs a decision above researcher level:** whether F04's Cursor path targets Enterprise OTel (best signal, gated plan), Admin API `filtered-usage-events` (real tokens, Team plan), or hook estimates only (free, ordinal-only). These are three different products.
- `AGENTS.md` describes `skillscore` as "npm, 7-dimension"; upstream is a Dart CLI with six dimensions. Also claims `skill-validator` gives token counts — true, but only under `check`, not `analyze content`. Doc fix, out of scope here.
