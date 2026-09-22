# Research contradictions and spec-gate findings

**Date:** 2026-09-10  
**Source:** Hunter report (telemetry-name and factual conformance lens, round 1) against the `skill-architect/.scuba/teams/cursorscope-go/` roadmap and slice files, plus local reconciliation with `cursor-profiler/docs/research/`.

## Notes on provenance

- The `skill-architect/.scuba/` control-plane files are **gitignored** and not readable or writable by this agent (Repository fact).
- The Hunter report was pasted into the conversation by the user; this file is a fold-in of its findings against the tracked `cursor-profiler` research.
- Evidence labels: **Repository fact** = from files in this workspace; **External evidence** = from the Hunter report or fetched sources; **Local hypothesis** = not yet verified by a primary source.

## Contradictions between this repo and the skill-architect ledger

| # | This repo | skill-architect ledger / verified fact | Resolution |
|---|---|---|---|
| 1 | `cursorscope-source-analysis.md` says cursorscope handles **19** hook events. | Cursor documents **21** hook events (Specification). cursorscope's `.cursor/hooks.json` registers 19, omitting `beforeTabFileRead` and `workspaceOpen` (Repository fact, verified in clone). | **Fixed in `cursorscope-source-analysis.md`:** now lists 21 Cursor events and notes cursorscope's 19-event subset. |
| 2 | `skill-evaluation-research.md` says SkillReducer shows runtime token savings. | The SkillReducer abstract reports **compression ratios only**; no runtime token or cost reduction is reported (External evidence). | **Fixed in `skill-evaluation-research.md`:** now says SkillReducer reports static compression, not runtime token savings. |
| 3 | `skill-evaluation-research.md` lists Skill-Lift dimensions as skill execution, behavior check, skill efficiency, token delta. | SkillEvaluator's published five dimensions are **Correctness, Discoverability, Effectiveness, Efficiency, Security** (External evidence). ACES process-metric labels are a separate vocabulary. | **Fixed in `skill-evaluation-research.md`:** now separates ACES process metrics from SkillEvaluator dimensions. |
| 4 | `skill-evaluation-research.md` cites SkillEval and `agent-profiler` as sources. | Neither appears in the skill-architect ledger; not verified this session (Local hypothesis). | **Noted in `skill-evaluation-research.md`:** both are now flagged as **Local hypothesis** until read in full. |
| 5 | `cursorscope-source-analysis.md` lists `preCompact` payload fields without `context_window_size`. | Cursor's `preCompact` carries `context_tokens` **and** `context_window_size` (Specification). | **Fixed in `cursorscope-source-analysis.md`:** added `context_window_size` and noted the denominator is needed for occupancy. |
| 6 | `recommendation.md` §3 originally described the Cursor surface as `.mdc` rules + `AGENTS.md`; the docs' silence on skills was read as absence. | **Cursor ships a stock skills system** — four native roots (`.agents/skills/`, `.cursor/skills/`, `~/.agents/skills/`, `~/.cursor/skills/`) plus four compatibility roots (`.claude/skills/`, `.codex/skills/`, `~/.claude/skills/`, `~/.codex/skills/`), walked recursively, `name` required to match the parent folder, root precedence undocumented (Specification — re-verified against `cursor.com/docs/skills.md`, 2026-09-12, via `pi.md §4a`). | **Fixed in `recommendation.md` §3** ([pi] edit 6): `~/.claude/skills` is live in Cursor with zero config so portability findings hit Cursor first; agnix `CUR-*` covers `.mdc` not `SKILL.md`, so Cursor `SKILL.md` conformance and root-collision detection are ours (skillgate `SK-I002` legs); multi-root collision is a Cursor finding class needing a fixture. |

## Contract findings that need a decision in `skill-architect`

These are not fixable by this repo alone; they affect `skill-architect/profiler`.

### C1. `compareTokens` drops the source label on comparison

- **Where:** `profiler/compare.go:107,115`
- **Finding:** `compareTokens` gates only on `State == present` and reports the **candidate's** `Source`.
- **Risk:** An estimated baseline compared against a billed candidate yields `comparable: true` with the estimate's honesty label dropped.
- **What would settle it:** Add a `SourceEstimated` constant and/or require matching `Source` values before comparing token pairs. Needs a design decision and schema update in `skill-architect`.

### C2. `CapabilityReport` has one `MetricSource` per metric, no reason channel

- **Where:** `profiler/types.go:50-55`
- **Finding:** `CapabilityReport` is `map[MetricName]MetricSource` with no `Reason` field.
- **Risk:** An adapter cannot report both `hooks` and `otel` as independent sources for the same metric, nor explain why a capability is `none`.
- **What would settle it:** Add a `MetricCapability` struct (source + reason) or split into `sources []MetricSource` per metric. Needs a `skill-architect` contract slice.

## Other high-value Hunter findings to fold into the next spec pass

| # | Finding | Where tracked | Implication |
|---|---|---|---|
| A1 | `cursor.go:259-273` already returns `PresentActivationResult`; `attribution: unknown` is correct, but `skill_activation` is **not** the first `present` for all adapters. | `profiler/cursor.go`, `profiler/profiler_test.go:644-652` | Do not write the spec as if no adapter ever produced `present` skill activation; write that no adapter produced **honest, attributable** activation with confidence labels. |
| A2 | `exit_code` in `afterShellExecution` is not in Cursor's documented payload; it traces only to cursorscope `CHANGELOG 0.3.7` and `src/telemetry.js:530`. | R-CS-15 | Treat `exit_code` as **Local hypothesis** unless a real Cursor payload or updated docs prove it. Do not label it Specification. |
| A3 | Existing `cursor.go` tests pin `"trigger":"keyword"` and `Reasoning==150`; these should be **replaced**, not extended. | `profiler_test.go:560,564,565,605,626` | S1's golden fixtures need to replace the unverified `cursor.*` telemetry names, not add to them. |
| A4 | `cursor.token.usage`, `cursor.skill.activated`, and `cursor.hook.execution_complete` in existing `cursor.go` are **unverified** telemetry names. | `profiler/cursor.go` | Re-verify or remove; R-SA-14 applies. |

## What is intentionally not resolved here

- `cursor-telemetry-surfaces.md` with the full 21-event payload schema and Admin API details (needs a dedicated doc; not created this pass).
- `cursorscope-requirements.md` with the full R-CS / R-SA / R-RL register (exists in `cursor-profiler/spec.md` but not broken out as a research file).
- `go-dependency-and-tooling-decisions.md` with measured OTel SDK cost and semgrep assessment (exists in `cursor-profiler/spec.md` forks).
- `skill-profiling-state-of-the-art.md` with the full SkillsBench, ACES CI, and Claude Code `/skill-doctor` numbers.
- `open-unknowns.md` with U-01…20 — the open questions live in `cursor-profiler/intent.md` and `cursor-profiler/spec.md` for now.
