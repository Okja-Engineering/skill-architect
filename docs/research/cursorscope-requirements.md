# Requirements register — extracted from cursorscope, plus research additions

Collected 2026-09-10 from `skill-architect/.scuba/teams/cursorscope-go/mandate-draft.md` §3–§4 and
`roadmap.md` §B. Evidence labels per `AGENTS.md`.

**Companion file:** `cursorscope-source-analysis.md` holds the architecture, file-by-file source
findings, and the port/drop implications. This file is the **requirement register only** — what we
carry, how, and why. It does not repeat the source analysis.

Tags: **keep** (carry as-is in spirit) · **adapt** (carry the requirement, change the mechanism) ·
**drop** (deliberately out) · **conditional** (gated on a user answer).

Deltas the research ledger introduced against the original extraction are marked ⚠.

---

## 1. R-CS — extracted from cursorscope

### 1.1 Ingestion and hook plumbing

| ID | Requirement | Tag | Rationale / delta |
|---|---|---|---|
| **R-CS-01** | Cover Cursor's lifecycle hook set | adapt | ⚠ **21 documented events, not 19.** cursorscope registers 19, omitting `beforeTabFileRead` and `workspaceOpen` (**Repository fact**, verified in the clone: `scripts/install-global-hooks.sh:12-18` and the `all` list at `:39`). We need a subset for a `Profile`: session/prompt boundaries (timing), tool pre/post (tool_calls), `beforeReadFile`+shell (skill attribution), `preCompact` (context tokens). Register the rest as pass-through so fixtures capture everything once. |
| **R-CS-02** | Hook entrypoint reads JSON on stdin, enriched with correlation and environment fields | keep | ⚠ Ledger confirms the exact base-field set: `conversation_id`, `generation_id`, `model`, `hook_event_name`, `cursor_version`, `workspace_roots`, `user_email`, `transcript_path` (**Specification**). Go binary instead of a Node forwarder. |
| **R-CS-03** | Hook failure must never break the user's Cursor session (non-zero exit tolerated, timeouts bounded) | keep | Non-negotiable. A profiler that wedges the editor is worse than no profiler. |
| **R-CS-04** | Forward to a local HTTP ingestor at `POST /cursor/hooks`, 202 on accept | adapt | Default becomes append-to-JSONL-spool; HTTP is a conditional later slice. |
| **R-CS-05** | Correlate by `conversation_id` (session) → `generation_id` (turn) → `tool_use_id` (tool call), subagents branching | keep | The correlation *keys* make a `Profile` coherent. Keep the keys and nesting semantics; OTel spans are not required to express them. |
| **R-CS-06** | TTL sweep for un-closed state (tool/subagent 5 min, interaction/session 30 min) | adapt | cursorscope shipped this as a memory-leak fix in 0.3.7. In a stateless spool design it becomes a **session-close heuristic in the reader** — same requirement, no long-lived process, no leak class at all. |
| **R-CS-07** | Config via env vars / `.env`, sane local defaults | adapt | Env vars yes; `.env` loading is a Node-ism. Flags + env, no dotenv. |

### 1.2 Attribution and metrics — the valuable part

| ID | Requirement | Tag | Rationale / delta |
|---|---|---|---|
| **R-CS-08** | Classify every invocation into `mcp`/`cli`/`skill`/`subagent`/`tool` with a resolved name and detail | **keep — highest value** | This produces skill activation and attribution, which no `skill-architect` adapter can produce today (**Repository fact**: all four adapters return `unknown` for both). |
| **R-CS-09** | Resolve a skill name from `…/skills/<name>/…` or `…/<name>/SKILL.md` | keep | ⚠ Ledger: **there is no skill activation hook in Cursor's free surface.** This is *inference from a file read*, not an activation event, and the profile must label it as such. Maps to `ActivationEntry.SkillName` / `Attribution.SkillName`. |
| **R-CS-10** | Resolve MCP server/tool identity from `mcp_server`, URL host, command, or `MCP: server/tool` patterns | adapt | Useful for attribution completeness; secondary to skill attribution. |
| **R-CS-11** | Estimate context tokens as `chars/4` with a `source` label (`estimated` / `mcp_reported`) | **adapt — with an honesty gate** | ⚠ Resolved via a new `estimated` source constant. Legitimate *relative* signal; must not serialize as a bare `present` `TokenCounts`. Calibration: −7%…+15% vs `o200k_base` on English markdown. |
| **R-CS-12** | Extract genuinely-reported token usage from MCP `result_json` | keep | Real numbers when present. Small surface, high honesty value. |
| **R-CS-13** | Capture `context_tokens` + `context_usage_percent` from `preCompact` | keep | ⚠ Ledger adds `context_window_size` and confirms this is the **only** Cursor-reported token number in the whole hook surface — and it fires **only on compaction**. |
| **R-CS-14** | Capture line-edit stats from `afterFileEdit` (added/removed, extension, basename) | adapt → **deferred** | Genuine effort signal for a paired comparison, but `Profile` v1 has no field for it (**Repository fact**, `profiler/types.go`). Needs a v2 schema discussion. **Orphan: no slice owns it.** |
| **R-CS-15** | Capture shell `exit_code` for failure discrimination | keep | Cheap; feeds `ToolCallEntry.Success` honestly instead of guessing. |
| **R-CS-16** | Reuse cursorscope's `cursor_*_total` metric names if we export | adapt (conditional) | Only meaningful if OTLP export ships. If it does, reuse the names verbatim — interop beats novelty. |
| **R-CS-17** | GenAI semconv attributes and metrics | adapt (conditional) | ⚠ Ledger: semconv 1.44.0 is **Development status**, and there is **no skill/rule convention at all**. Putting chars/4 estimates into `gen_ai.usage.*_tokens` is exactly the dishonesty a `MetricResult` contract exists to prevent. cursorscope does it; we must not. See R-RL-13. |

### 1.3 Privacy, install, ops

| ID | Requirement | Tag | Rationale / delta |
|---|---|---|---|
| **R-CS-18** | Redact user prompts by default (length only); opt-in flag to include text | **keep** | Non-negotiable. Prompts hit **disk** in a spool design, which is *more* sensitive than cursorscope's in-memory→OTLP path. |
| **R-CS-19** | Scrub bearer tokens, `sk-`/`key-` keys, emails; mask `api_key`/`token`/`password`/`secret`/`authorization` | **keep** | Must run **before the spool write**, not before export. |
| **R-CS-20** | Optional email masking; bounded tool-payload size (8192) | keep | Cheap; prevents unbounded spool growth. |
| **R-CS-21** | Non-clobbering `~/.cursor/hooks.json` merge; timestamped backup; clean uninstall of only our entries | **keep — reimplement** | ⚠ cursorscope's merge shells out to a `python3` heredoc (**Repository fact**, verified in the clone). Reimplement with `jq` or Go. Merge owner (jq vs Go) is still **ambiguous in the plan** — decide it. |
| **R-CS-22** | Auto-start the ingestor; version-match and restart; pidfile + logfile + `nohup` | **drop under the default fork** | ~104 lines of bash existing only because there is a daemon. The stdin+spool default **deletes the requirement** rather than porting it. |
| **R-CS-23** | Debug HTTP endpoints (`/healthz`, `/debug/*`) | adapt | Replaced by a one-shot `doctor` command reporting hook install state, spool path/size/recency, and last parse errors. Same diagnostic need, no server. |
| **R-CS-24** | OTLP endpoint resolution semantics (per-signal env → `OTEL_EXPORTER_OTLP_ENDPOINT` + `/v1/{signal}` → `localhost:4318`; `OTEL_EXPORTER_OTLP_HEADERS` as comma-separated `k=v`) | keep (conditional) | Well-specified, tested upstream, conventional. Copy semantics exactly if export ships. |
| **R-CS-25** | Setup CLI verbs | adapt | Keep `hooks install/uninstall`, `doctor`/`status`, `--dry-run`, `--yes`. Drop `start`/`stop` (no daemon) and the interactive backend wizard. |
| **R-CS-26** | Cursor Admin API polling | ⚠ **drop → conditional keep, retargeted** | The original extraction evaluated the **wrong endpoint**. `daily-usage-data` is team-daily and unattributable; `filtered-usage-events` is per-request with `tokenUsage` **and** `conversationId`. See R-RL-07/08. |
| **R-CS-27** | Bundled OTel Collector configs + `docker-compose.yml` | **drop** | Infrastructure for an observability product. Not our problem. |
| **R-CS-28** | Node ≥20 runtime, `npm install` on first run | **drop → replaced** | A single static Go binary with no runtime and no `node_modules`. **This is the port's actual justification** and should be stated as such. |
| **R-CS-29** | Unit tests per module + a fake OTLP collector | **keep** | `attribution.test.js` / `privacy.test.js` are a ready-made behavioral oracle; `test/fake-collector.mjs` maps to an `httptest.Server`. ⚠ **Not verified:** those tests were listed, not executed. |
| **R-CS-30** | Bounded memory; hook latency invisible in the editor | keep | With a spool design memory is trivially bounded; the live constraint becomes hook process startup + write latency. **Needs a measured budget — currently unmeasured.** |

---

## 2. R-SA — constraints from `skill-architect` itself

These constrain the port more than cursorscope does.

| ID | Requirement | Source | Delta |
|---|---|---|---|
| **R-SA-01** | Valid `skill-architect/profile/v1`; `Marshal→Unmarshal→Marshal` identical | `docs/profiler-spec.md`, `profiler/types.go` | — |
| **R-SA-02** | Every metric a `MetricResult`: `present` (value + `Source`, no `Reason`) / `unknown` (reason required) / `error` (reason required). Nothing invented or zero-filled. | `docs/profiler-spec.md` | — |
| **R-SA-03** | **Estimated values are not `present` values** | `PRINCIPLES.md`, R-CS-11 | ⚠ **Resolved** by R-RL-06 (`SourceEstimated`), but see the contract limits in `go-dependency-and-tooling-decisions.md` §4 — the resolution is incomplete downstream. |
| **R-SA-04** | Profile pinned to `snapshot_hash`; comparison revalidates before comparing | `docs/profiler-spec.md`, `profiler/compare.go` | ⚠ **Orphan risk:** assigned to a slice whose DoD never mentions `snapshot_hash`. |
| **R-SA-05** | Adapter declares a `CapabilityReport` before capture. `SourceHooks` exists in `types.go` and is used by **no adapter** — this work is its first consumer. | `profiler/types.go` | — |
| **R-SA-06** | Satisfy the comparison contract: `CompareProfiles` compares only metrics `present` on **both** sides, emits deltas, notes harness/snapshot/skill_dir mismatches | `profiler/compare.go` | ⚠ **Not verified** against a hook-sourced profile. |
| **R-SA-07** | Making `skill_activation` / `attribution` `present` for Cursor is the primary success condition | all four adapters | The prize. |
| **R-SA-08** | `conversation_id` / `generation_id` / `session_id` must map onto `capture --session` unambiguously | `profiler/cmd/main.go` | ⚠ Ledger: **`conversation_id` is the join key to both the Admin API and Enterprise OTel** — it is the right `SessionID`. |
| **R-SA-09** | Bash for scripts, Go for logic, **no Python** | `AGENTS.md` | Blocks porting `install-global-hooks.sh` as-is. |
| **R-SA-10** | Prefer existing tools (`jq`, `skill-validator`, `skillscore`) over building | `AGENTS.md` | — |
| **R-SA-11** | Evidence labels on every doc claim; hook-payload claims start as **Local hypothesis** until a real payload is captured | `AGENTS.md` | — |
| **R-SA-12** | `profiler/` has zero external Go dependencies | `profiler/go.mod` | ⚠ Fork closed, not merely defaulted — see R-RL-14. |
| **R-SA-13** | Degrade gracefully with no Cursor present: `probe` reports `none`, `capture` returns all-`unknown` with reasons. Never fail hard. | `docs/profiler-spec.md` | — |
| **R-SA-14** | **No invented telemetry names**, including auditing the existing `cursor.*` strings | `AGENTS.md`, `PRINCIPLES.md` | ⚠ Ledger **verified all four Cursor names are real**. The defect is one level down: every attribute key is wrong. |
| **R-SA-15** | Must slot into `experiment plan` / `experiment run`; `Condition`/`ExperimentStep` need a hook-spool input path | `profiler/experiment.go` | — |
| **R-SA-16** | No regressions: 131 bash tests + 34 Go tests stay green | `CHANGELOG.md` 0.4.0, `tests/` | — |
| **R-SA-17** | Amend `.out-of-scope.md` and `README.md` if a live capture surface lands — they currently promise the plugin does *not* run live comparisons | those files | — |

---

## 3. R-RL — new requirements from the research ledger

| ID | Requirement | Tag | Evidence |
|---|---|---|---|
| **R-RL-01** | Correct every Cursor OTel attribute key to its namespaced form: `cursor.token.type` (`input\|output\|cache_read\|cache_creation`), `cursor.skill.name`, `cursor.skill.trigger`, `cursor.tool.kind\|.name\|.status`, `cursor.hook.name\|.type\|.outcome\|.duration_ms`. **Delete the dead `reasoning`/`reasoning_output` token cases — the enum has no such member.** | keep | **Specification** (Wire Reference) |
| **R-RL-02** | `MetricToolCalls` must read the **`cursor.tool.calls` metric**, not `cursor.hook.execution_complete`. The latter describes *hook* executions and may only feed a separate hook-health signal, never `ToolCallEntry`. | keep | **Specification** |
| **R-RL-03** | `Probe()` must not set `MetricTokens = SourceSQLite` on a `state.vscdb`. The documented schema contains **no token data**; downgrade to `SourceNone` with the reason stated. | keep | **External evidence** |
| **R-RL-04** | Source tokens primarily from `cursor.api.request.*_tokens`, scoped by `cursor.conversation.id`; fall back to the aggregate `cursor.token.usage` metric. | keep | **Specification** |
| **R-RL-05** | `ActivationEntry` gains an additive `source` field carrying `cursor.skill.source` (`unspecified\|workspace\|user\|builtin\|plugin\|claude`) — it is what distinguishes a workspace skill from a builtin. Additive; no schema bump. | keep | **Specification** |
| **R-RL-06** | Add `SourceEstimated` to `MetricSource`. An estimate must never serialize with `Source: "otel"` or `"hooks"`. Resolves R-SA-03. | keep | **Design decision** |
| **R-RL-07** | Real Cursor tokens without Enterprise: `POST /teams/filtered-usage-events` (Basic auth, API key as username) → per-request `tokenUsage{inputTokens, outputTokens, cacheWriteTokens, cacheReadTokens}` with `conversationId`, **the same id the hooks emit**. Poll ≤1×/hour; 20 req/min/team; 30-day max range. | conditional | **Specification** |
| **R-RL-08** | Do **not** port cursorscope's `GET /teams/{teamId}/daily-usage-data` poller: wrong verb (`POST`), wrong auth (Basic, not Bearer), wrong path (no team segment), and the endpoint **has no token fields at all**. | drop | **Specification** + **Repository fact** |
| **R-RL-09** | Static per-skill token cost from `skill-validator check -o json` → `.token_counts.files[]` / `.token_counts.total`, real `o200k_base` BPE. `analyze content` does **not** emit it. Already flows into `audit.json` under `.spec.token_counts` but is not surfaced. | keep | **Repository fact** (measured) |
| **R-RL-10** | `chars/4` is calibrated to −7%…+15% vs `o200k_base` on English markdown (worst on dense lists). Valid for ranking and relative deltas; invalid for absolute cost. Accuracy on code/JSON/non-English **untested**. | adapt | **Repository fact** |
| **R-RL-11** | Always-loaded budget rules to check statically: `description` + `when_to_use` ≤ **1,536 chars**; `SKILL.md` ≤ **500 lines**; listing budget `skillListingBudgetFraction` default **0.01**. | keep | **Specification** |
| **R-RL-12** | `/skill-doctor` (per-skill context cost + invocation count, Claude Code ≥2.1.252) is the target shape for a first-party per-skill cost number. Local Claude Code is **2.1.221** — unavailable. `-p` parseability **UNKNOWN**. | defer | **Specification** + **Repository fact** |
| **R-RL-13** | If we export: namespace our own skill attributes as `skill_architect.skill.*`; map to `gen_ai.*` **only** where the convention defines the field. There is **no** skill/rule convention, and `gen_ai.*` is Development status. Never put a `chars/4` estimate in `gen_ai.usage.*_tokens`. | keep | **Specification** (semconv 1.44.0) |
| **R-RL-14** | Hand-roll OTLP/HTTP JSON with `encoding/json` + `net/http`. If the SDK ever becomes mandatory, isolate it in a **separate module** at `profiler/exporters/otlp/`. | keep | **Repository fact** (measured — see `go-dependency-and-tooling-decisions.md` §1) |
| **R-RL-15** | Do not adopt semgrep. Guardrails ship as `grep` + `go vet` + Go tests. | keep | **Design decision** (see §2 of the tooling file) |
| **R-RL-16** | The comparison report must express Skill Lift with a **confidence interval, not a point estimate**. ACES: mean 0.2134, 95% CI [0.1967, 0.2301] over 947 paired cases; positive lift rate only **72.8%** — the engine must be able to report "this skill hurt." | keep | **External evidence** (arXiv:2608.20614) |
| **R-RL-17** | **Spike, load-bearing:** does Cursor's skill loading actually fire `beforeReadFile` on a `SKILL.md` path? The entire local skill-attribution heuristic depends on it and it is **untested**. | keep | **Local hypothesis** |
| **R-RL-18** | `preCompact.context_tokens` (with `context_window_size`, `context_usage_percent`) is the **only** Cursor-reported token number in the 21-event hook surface, and it fires **only on compaction**. | keep | **Specification** |

---

## 4. What the research changed about the original extraction

| Original assumption | Ledger verdict | Consequence |
|---|---|---|
| `cursor.token.usage` / `.skill.activated` / `.hook.execution_complete` may be invented | **All four names are real**, but **every attribute key read is wrong**, `cursor.hook.execution_complete` is misused as a tool-call event, and `Probe()` claims a SQLite token capability the schema lacks | A standalone bug-fix work item in shipped code, independent of everything else |
| R-CS-26 Admin API → **drop** (team-daily aggregates, unattributable) | Wrong endpoint was evaluated. `filtered-usage-events` is per-request, has `tokenUsage`, and has `conversationId` | Conditional keep, gated on plan tier |
| Tokens stay `unknown` | Three concrete sources of differing honesty now exist | Default flips to a `SourceEstimated` constant + per-source representation |
| "Prefer" zero Go dependencies | Measured: OTel Go SDK = 2 direct + 22 indirect modules, 408 packages, **17.7 MB**; stdlib OTLP/JSON = **0 modules, 6.4 MB** | Fork closed, not merely defaulted |
| semgrep is a marginal fit | Confirmed: cannot count tokens; adds a Python runtime | Guardrails as `grep` + `go vet` + tests |
| The user runs Cursor locally | **Repository fact:** `~/.cursor/` and `~/Library/Application Support/Cursor/` are **absent on the research machine** | The premise is unverified at its root — the gating user question |

---

## 5. Requirement coverage warnings

Carried forward from the spec-gate hunter review; these are **register defects, not design defects**,
and should be closed before any of this is dispatched as work.

| Warning | Detail |
|---|---|
| **Orphans (requirement → no owner)** | **R-CS-14** (tagged *adapt*, but "deferred" with no slice). **R-CS-04** (tagged *adapt*, assigned only to a conditional slice recommended never to ship). **R-RL-16** (tagged *keep*, assigned to a slice that explicitly declines to implement it). **R-SA-04** assigned to a slice whose DoD never mentions `snapshot_hash`. |
| **Phantom claims (owner → requirement it does not satisfy)** | A conditional daemon slice claims R-CS-06 (owned elsewhere, and in a TTL form the chosen fork deleted) and R-CS-23 (owned elsewhere). The Admin-API slice claims **R-RL-08, which is tagged `drop`.** An attribution slice claims R-SA-11 with no doc file in scope. |
| **Direct design conflict** | A proposed guardrail rule — "no `present` `MetricResult` from a function named `estimate*`" — **forbids exactly what R-RL-06 mandates** (`SourceEstimated` with `state: present`). One of the two has to move. |
| **Vocabulary collision** | R-CS-09 requires the trigger be "marked as inferred", but R-RL-01 imports Cursor's `cursor.skill.trigger` enum (`agent_read|manually_attached|skill_name_in_prompt`), which has no "inferred" member. **No requirement reconciles them.** |
| **Ownership ambiguity** | R-CS-21's hooks.json merge is assigned to `jq` in one place and to a Go command in another. |
