# Cursor telemetry surfaces — what each source actually carries

Collected 2026-09-10 from `skill-architect/.scuba/teams/research-profiling/ledger.md` §A5 and §B3.
Evidence labels per `AGENTS.md`: **Specification** / **External evidence** / **Repository fact** /
**Design decision** / **Local hypothesis**.

Cursor exposes four distinct signal surfaces. They differ in plan gating, granularity, and — most
importantly — in whether the token numbers they carry are billed, Cursor-reported, or estimated.

| Surface | Plan | Grain | Skill activation? | Real tokens? |
|---|---|---|---|---|
| Lifecycle hooks | Free/local | per event | inferred only | **no** (one exception) |
| Enterprise OTel export | Enterprise (beta) | per request | **yes, first-party** | **yes** |
| Admin API `filtered-usage-events` | Team/Enterprise | per request | no | **yes** |
| `state.vscdb` SQLite | Free/local | per conversation | no | **no** |

---

## 1. Lifecycle hooks (free, local, no enterprise plan)

**Specification** — <https://cursor.com/docs/agent/hooks> (also <https://cursor.com/docs/hooks>).

**Common base fields on all agent hooks:** `conversation_id`, `generation_id`, `model`, `model_id`
(opt), `model_params` (opt), `hook_event_name`, `cursor_version`, `workspace_roots`, `user_email`,
`transcript_path`.

**The documented surface is 21 events.** cursorscope registers only 19 of them — see
§5 "Discrepancies" below and `cursorscope-source-analysis.md`.

| Hook event | Payload fields beyond base | Token-bearing? |
|---|---|---|
| `sessionStart` | `session_id`, `is_background_agent`, `composer_mode` (opt) | no |
| `sessionEnd` | `session_id`, `reason`, `duration_ms`, `is_background_agent`, `final_status`, `error_message` (opt) | no |
| `beforeSubmitPrompt` | `prompt`, `attachments` | proxy (prompt text) |
| `preToolUse` | `tool_name`, `tool_input`, `tool_use_id`, `cwd`, `model`, `model_id`, `model_params`, `agent_message` | proxy (`tool_input`) |
| `postToolUse` | `tool_name`, `tool_input`, `tool_output`, `tool_use_id`, `cwd`, `duration`, `model`, … | proxy (`tool_output`) |
| `postToolUseFailure` | `tool_name`, `tool_input`, `tool_use_id`, `cwd`, `error_message`, `failure_type`, `duration`, `is_interrupt` | no |
| `beforeShellExecution` | `command`, `cwd`, `sandbox` | proxy |
| `afterShellExecution` | `command`, `output`, `duration`, `sandbox` | proxy (`output`) |
| `beforeMCPExecution` | `tool_name`, `tool_input`, `mcp_server_name`, `url`/`mcp_server_url` or `command` | proxy |
| `afterMCPExecution` | `tool_name`, `tool_input`, `mcp_server_name`, `mcp_server_url` (opt), `result_json`, `duration` | **sometimes real** (MCP self-report) |
| `beforeReadFile` | `file_path`, `content`, `attachments` | proxy (`content`) — **also the skill-activation proxy** |
| `afterFileEdit` | `file_path`, `edits[]{old_string,new_string}` | no |
| `beforeTabFileRead` | `file_path`, `content` | proxy |
| `afterTabFileEdit` | `file_path`, `edits[]{old_string,new_string,range,old_line,new_line}` | no |
| `subagentStart` | `subagent_id`, `subagent_type`, `task`, `parent_conversation_id`, `tool_call_id`, `subagent_model`, `is_parallel_worker`, `git_branch` (opt) | no |
| `subagentStop` | `subagent_type`, `status`, `task`, `description`, `summary`, `duration_ms`, `message_count`, `tool_call_count`, `loop_count`, `modified_files`, `agent_transcript_path` | no |
| `afterAgentResponse` | `text` | proxy (output text) |
| `afterAgentThought` | `text`, `duration_ms` (opt) | proxy (reasoning) |
| `stop` | `status`, `loop_count` | no |
| **`preCompact`** | `trigger`, **`context_usage_percent`**, **`context_tokens`**, **`context_window_size`**, `message_count`, `messages_to_compact`, `is_first_compaction` | **REAL — the only one** |
| `workspaceOpen` | `hook_event_name`, `cursor_version`, `workspace_roots`, `user_email` (**no** `conversation_id`/`generation_id`) | no |

### Three findings that drive the design

1. **`preCompact.context_tokens` is the only real, Cursor-reported token number in the entire hook
   surface** (**Specification**). It is Cursor's own count of current context-window occupancy,
   alongside `context_window_size`. **It fires only on compaction** — short sessions emit nothing.
   cursorscope records it into `gen_ai.client.token.usage` with `gen_ai.token.type=input`
   (**External evidence**, `src/telemetry.js:handlePreCompact`).
2. **`afterMCPExecution.result_json` can carry real usage — but it is the MCP server's self-report,
   not Cursor's** (**External evidence**). `src/attribution.js:extractReportedTokenUsage` probes
   `usage` / `token_usage` / `tokens` for `input_tokens|prompt_tokens|input`,
   `output_tokens|completion_tokens|output`, `total_tokens|total`, and tags the result
   `source: "mcp_reported"` vs `"estimated"`. Right pattern, narrow applicability.
3. **There is no skill or rule activation hook.** Nothing in the hook surface names a skill
   (**Specification**, by absence). cursorscope infers it from `beforeReadFile.file_path` / shell
   `command` matching `/(?:^|[/\\])skills[/\\]|SKILL\.md$/i` plus `subagent_type` matching `/skill/i`.
   **Local hypothesis (load-bearing, untested):** Cursor loads `.cursor/skills/*/SKILL.md` through a
   path that fires `beforeReadFile`. If it loads skills internally without a file-read tool call,
   the heuristic yields nothing. See `open-unknowns.md` U-02.

### What hooks cannot do — the structural gap

**Local hypothesis, well-founded, untested here.** Billed input tokens per turn are approximately
the *cumulative* conversation re-sent each turn, not the per-event payload size. Summing per-event
payload estimates therefore **systematically undercounts billed input** — the true curve is roughly
quadratic in turn count, the sum-of-payloads is linear — and it completely misses the system prompt,
the always-loaded skill listing, rules, and the cache-read vs cache-creation split.
`preCompact.context_tokens` is the only hook datum reflecting the true cumulative figure.

**Verdict (ledger Q1): hooks alone cannot give per-session token counts.** The hook-derived number
is an **ordinal signal only** — valid for ranking two skills in a paired A/B where turn counts are
comparable, invalid for any absolute cost or dollar claim.

---

## 2. Enterprise OpenTelemetry export — verified wire reference

**Specification.** <https://cursor.com/docs/enterprise/opentelemetry-export> and the Wire Reference
at <https://cursor.com/docs/enterprise/opentelemetry-export/wire>. **Enterprise plan only, beta**,
configured by admins in Team Settings → OpenTelemetry Export, to **one** team-managed destination.

Resource attributes: `service.name=cursor`, `cursor.team.id`, `cursor.user.id` (opt).
Join keys on logs: `cursor.event.id` (dedup), `cursor.conversation.id`, `cursor.usage_event.id`
(request grain), `cursor.request.id`, `cursor.grok_bot.turn.id`.

### Metrics (delta temporality)

| Metric | Unit | Attributes |
|---|---|---|
| `cursor.token.usage` | `{token}` | **`cursor.token.type`**: `input` \| `output` \| `cache_read` \| `cache_creation`; `cursor.model.name` (opt); `cursor.api.status`: `success`\|`errored`\|`aborted` (opt); `cursor.api.billable` bool (opt) |
| `cursor.tool.calls` | `{call}` | `cursor.tool.kind`: `builtin`\|`mcp`; `cursor.tool.name`; `cursor.tool.status`: `success`\|`failure`\|`aborted`; `cursor.mcp.server.name` (MCP only) |
| `cursor.cost.usage` | USD | `cursor.model.name` (opt) — best-effort estimates, not invoices |

### Log events

| Event | Severity | Attributes |
|---|---|---|
| `cursor.api.request` | INFO | **`cursor.api.request.input_tokens`**, **`.output_tokens`**, **`.cache_read_tokens`**, **`.cache_creation_tokens`** (all int); `cursor.model.name` (opt); `cursor.api.billable` (opt) |
| `cursor.api.error` | — | error events without raw messages |
| `cursor.api.correction` | — | billing finalization |
| **`cursor.skill.activated`** | INFO | **`cursor.skill.name`**; **`cursor.skill.trigger`**: `agent_read` \| `manually_attached` \| `skill_name_in_prompt`; **`cursor.skill.source`**: `unspecified`\|`workspace`\|`user`\|`builtin`\|`plugin`\|`claude`; `cursor.plugin.name` (opt) |
| `cursor.hook.execution_complete` | INFO/ERROR | **`cursor.hook.name`**; **`cursor.hook.type`**: `pre_tool_use`\|`post_tool_use`\|`post_tool_use_failure`\|`before_submit_prompt`\|`after_agent_response`\|`after_agent_thought`\|`stop`\|`subagent_start`\|`subagent_stop`; **`cursor.hook.outcome`**: `success`\|`blocked`\|`failed`\|`timeout`; **`cursor.hook.duration_ms`** int; `cursor.plugin.name` (opt) |
| `cursor.plugin.installed`, `cursor.cloud_agent.*`, `cursor.grok_bot.*` | — | lifecycle |

**Why this matters most.** `cursor.skill.activated` with `cursor.skill.trigger` is the *only*
first-party skill-activation event in any harness surveyed, and `cursor.api.request` gives
per-request four-way token splits joinable to `cursor.conversation.id`. On Enterprise, Cursor is the
**best**-instrumented harness for skill profiling. Off Enterprise, none of it exists.

**Note — `cursor.hook.execution_complete` is about hook executions, not tool calls.** Tool calls live
in the `cursor.tool.calls` metric. Conflating the two is the defect ledger Q4 found in
`skill-architect`'s `profiler/cursor.go`; see `go-dependency-and-tooling-decisions.md` §4.

---

## 3. Admin API — `POST /teams/filtered-usage-events`

**Specification.** <https://cursor.com/docs/account/teams/admin-api>. Basic auth (API key as
username). Rate limits 20–250 req/min by endpoint. 30-day max date range. Team/Enterprise plans.
Data aggregated hourly; poll at most once per hour; 20 req/min per team.

| Endpoint | Grain | Tokens? | Join key |
|---|---|---|---|
| `POST /teams/daily-usage-data` | per-user-per-day | **no token fields at all** | none usable |
| **`POST /teams/filtered-usage-events`** | **per request** | **yes** | **`conversationId`** |
| `POST /teams/spend` | per user | no | none |

`filtered-usage-events` fields: `timestamp`, `userEmail`, `serviceAccountId`/`Name` (opt),
`cloudAgentId` (opt), `automationId` (opt), **`conversationId` (opt)**, `model`, `kind`, `maxMode`,
`requestsCosts`, `isTokenBasedCall`, `isChargeable`, `isHeadless`,
**`tokenUsage { inputTokens, outputTokens, cacheWriteTokens, cacheReadTokens, totalCents,
discountPercentOff }`**, `chargedCents`, `cursorTokenFee` (opt).

**The recommended architecture (Design decision, ledger Q1):** `filtered-usage-events.conversationId`
joins to hook `conversation_id`. **Hooks for structure and attribution; Admin API for the true
totals.** This is the cheapest real-token path for Cursor that does *not* require Enterprise OTel —
it needs a Team-plan admin API key.

**Repository fact.** cursorscope's poller (`src/cursor-api-poller.js`) uses
`GET /teams/{teamId}/daily-usage-data` with `Authorization: Bearer`. Per the docs the endpoint is
`POST /teams/daily-usage-data` with **Basic** auth and no team path segment — and `daily-usage-data`
has no token fields anyway. **Do not port that poller.** Target `filtered-usage-events` instead.

---

## 4. Local `state.vscdb` SQLite

**External evidence** (not verified locally). Two tables, both `key TEXT UNIQUE, value BLOB` holding
UTF-8 JSON: `ItemTable` and `cursorDiskKV`.

- Global `globalStorage/state.vscdb` keys: `composerData:{composerId}`,
  `bubbleId:{composerId}:{bubbleId}`, `checkpointId:{composerId}:{checkpointId}`,
  `messageRequestContext:{composerId}:{messageId}`, `composer.content.{hash}`.
- Workspace `workspaceStorage/{id}/state.vscdb`: `composer.composerData` (Cursor ≤2.6) →
  `composer.composerHeaders` (Cursor 3.0+).
- Bubble fields: `text`, `richText`, `type`, `context`, `codeBlocks`, `suggestedCodeBlocks`,
  `toolResults`, `checkpointId`, `allThinkingBlocks`, `createdAt` (~60+ fields).
- **No token counts or usage data documented in bubble or composer records.**
- Verified Feb 2026 (Cursor ~2.6), updated Apr 2026 (Cursor 3.0). A **breaking, one-way migration**
  occurred between versions; no stability guarantee.

<https://github.com/Callum-Ward/cursaves/blob/main/docs/how-cursor-stores-chats.md> ·
<https://github.com/S2thend/cursor-history> · <https://vibe-replay.com/blog/cursor-local-storage/>

**Repository fact.** Cursor is not installed on the research machine
(`~/Library/Application Support/Cursor/` absent, `~/.cursor/` absent). None of the above was
empirically verified. Treat the schema as **version-fragile External evidence**.

**Implication.** `state.vscdb` gives conversation reconstruction (prompts, responses, tool results,
thinking blocks) → good for *estimated* tokens and for reconstructing which files were read. It does
**not** give billed tokens. Any capability report claiming `tokens` from SQLite is claiming what it
cannot deliver.

---

## 5. Honesty ranking of token sources

**Design decision.** Every token number in a profile must name which of these it is. Ranked most to
least trustworthy:

| Rank | Source | What it is | Availability | Proposed `MetricSource` |
|---|---|---|---|---|
| 1 | Admin API `filtered-usage-events.tokenUsage` | **Billed**, per request, four-way split | Team+ plan admin key | `server_api` |
| 1= | Enterprise OTel `cursor.api.request.*_tokens` | **Billed**, per request, four-way split | Enterprise beta | `otel` |
| 2 | Enterprise OTel `cursor.token.usage` metric | Billed, aggregate (not conversation-scoped) | Enterprise beta | `otel` |
| 3 | `preCompact.context_tokens` | **Cursor-reported** context occupancy at compaction | free, but only when a session compacts | `hooks` |
| 4 | `afterMCPExecution.result_json` usage block | Real, but **the MCP server's** self-report | free, when the server reports it | `hooks` (qualified) |
| 5 | `chars/4` over hook payloads | **Estimate** of context bytes admitted | always | **`estimated`** (new constant) |

**Repository fact — chars/4 calibration spike (2026-09-10).** cursorscope's `Math.ceil(len/4)` vs
`skill-validator`'s `o200k_base` BPE on `skill-architect`'s own markdown:

| File | chars | char/4 est | o200k_base | error |
|---|---:|---:|---:|---:|
| `skills/skill-audit/SKILL.md` (body only) | 7,658 | 1,914 | 1,814 | **+5.5%** |
| `references/best-practices.md` | 4,776 | 1,194 | 1,135 | **+5.2%** |
| `references/checklists.md` | 1,991 | 497 | 432 | **+15.0%** |
| `references/evaluation-matrix.md` | 2,424 | 606 | 649 | **−6.6%** |

**char/4 lands within roughly −7% to +15% of a real BPE count on English prose/markdown.** Worst on
dense list-heavy markdown. Good enough for ranking and relative deltas; not good enough for absolute
cost claims. **Local hypothesis:** error will be materially worse on code, JSON, and non-English text
— untested (see `open-unknowns.md` U-07).

**The small error is the heuristic; the large error is structural** (§1). Do not let a ±15%
calibration number imply the summed estimate is ±15% of billed tokens. It is not.

**Design decision worth copying from cursorscope:** it never launders an estimate as a measurement.
Every attributed token carries `token_source` / `cursor.attribution.token_source` = `"estimated"` or
`"mcp_reported"` (`src/telemetry.js:recordAttributedContextTokens`). Add a `SourceEstimated`
`MetricSource` so an estimate is never reported as `otel` or `hooks`.

---

## 6. Discrepancies to be aware of

| # | Claim A | Claim B | Status |
|---|---|---|---|
| D-1 | `cursorscope-source-analysis.md`: "19 hook events" | Ledger §A5.1: **21** documented events | Both true at different scopes: Cursor documents 21; cursorscope registers 19 (**Repository fact**, verified in the clone at `scripts/install-global-hooks.sh:12-18` and the `all` list at `:39`). cursorscope omits `beforeTabFileRead` and `workspaceOpen`. |
| D-2 | Ledger §A5.1: cursorscope's registered list "matches exactly" the docs | Its own table lists 21 rows | Ledger wording is imprecise. The clone registers 19 of 21. Prefer the 21-event table as the Cursor contract. |
| D-3 | `cursorscope-source-analysis.md`: `preCompact` carries `context_tokens`, `context_usage_percent` | Ledger adds `context_window_size`, `trigger`, `message_count`, `messages_to_compact`, `is_first_compaction` | Not a conflict — the existing note is incomplete. `context_window_size` matters: it is what turns occupancy into a percentage. |

---

## Sources

- Cursor hooks — <https://cursor.com/docs/agent/hooks> · <https://cursor.com/docs/hooks>
- Cursor OTel export — <https://cursor.com/docs/enterprise/opentelemetry-export>
- Cursor OTel **Wire Reference** — <https://cursor.com/docs/enterprise/opentelemetry-export/wire>
- Cursor Admin API — <https://cursor.com/docs/account/teams/admin-api> · <https://cursor.com/docs/account/teams/analytics-api>
- Cursor `state.vscdb` — <https://github.com/Callum-Ward/cursaves/blob/main/docs/how-cursor-stores-chats.md> · <https://github.com/S2thend/cursor-history> · <https://vibe-replay.com/blog/cursor-local-storage/>
- cursorscope — <https://github.com/last9/cursorscope> (`@last9/cursorscope@0.3.7`, MIT)
- skill-validator — <https://github.com/agent-ecosystem/skill-validator>
