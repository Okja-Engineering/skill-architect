# Research ledger — Skill performance profiling & token-usage efficiency

**Date:** 2026-09-10 · **Researcher:** research-profiling · **Status:** done

Evidence labels per `AGENTS.md`: **Specification** / **External evidence** / **Repository fact** / **Design decision** / **Local hypothesis**.

## Question being answered

Two unknowns, scoped to decisions the architect must make now:

- **(A)** As of Sept 2026, how do people actually measure whether a skill helps an agent, and what telemetry does each harness (especially Cursor) expose to support that?
- **(B)** What is the evidence on skill token-usage efficiency, and how can tokens be attributed to a skill?

"Answered" means: the four decision questions at the end are specific enough to design against without re-researching, and every non-obvious claim carries a URL or a `Local hypothesis` label.

---

## Part A — Profiling Agent Skill performance

### A1. The field converged on paired (with-skill / without-skill) evaluation

**External evidence.** SkillsBench (arXiv:2602.12670) is the anchor benchmark. Abstract figures: 87 tasks across 8 domains, 18 model-harness configurations, deterministic verifiers. Pass rate **33.9% without Skills → 50.5% with curated Skills (+16.6 pp, 25.5% normalized gain)**; configuration-level gains **+4.1 to +25.7 pp**. Self-generated Skills give **no benefit on average**. "Focused Skills with at most three modules outperform larger or exhaustive bundles, and smaller models with Skills can match larger models without them."
<https://arxiv.org/abs/2602.12670>

> **Conflicting figures — flagged.** Secondary summaries report "86 tasks across 11 domains", "7 agent-model configurations over 7,308 trajectories", and "+16.2 pp". The arXiv abstract (primary) says 87 tasks / 8 domains / 18 configurations / +16.6 pp. **Use the arXiv abstract numbers; treat trajectory counts as UNKNOWN** until the PDF body is read.
> Secondary: <https://www.alphaxiv.org/overview/2602.12670>, <https://evaluatingevals.substack.com/p/skillsbench-review>

**External evidence.** NVIDIA **SkillEvaluator** (open source) operationalizes this as a 3-tier framework, and its Tier 3 is the reference implementation of the paired protocol:

- Tier 1 — safety & structure: schema, prompt-injection/exfiltration scan, secret/PII detection, license, script lint.
- Tier 2 — distinctiveness: embedding similarity, intra-skill duplication and cross-catalog overlap.
- Tier 3 — live evaluation: two isolated sandbox runs per case, identical prompt/model/inputs/grading; **the only variable is skill presence**.
- **Skill Lift** = with-skill score minus without-skill score, in points. Five scoring dimensions: Correctness, Discoverability (activation), Effectiveness, Efficiency (steps), Security.
- Supported harnesses in the blog: **Claude Code and OpenAI Codex** (not Cursor).
- Repetitions: 85% of skills single attempt, 15% two attempts.
- Benchmark, 2026-08-12: average Skill Lift **+31 points** across dimensions (**+39** excluding Security); Correctness **46 → 87 (+41)**.

<https://developer.nvidia.com/blog/evaluating-ai-agent-skill-performance-with-nvidia-skillevaluator/> · <https://github.com/NVIDIA/SkillEvaluator> · <https://docs.nvidia.com/skills/skillevaluator>

**External evidence.** The methodology paper behind it — **ACES** ("Evaluating Skills, Not Just Agents: Agentic Continuous Evaluation of Skills", arXiv:2608.20614): 145 real skills; **947 scored paired cases from 58 production skills across four primary harnesses**; **mean composite Skill Lift 0.2134, 95% CI [0.1967, 0.2301]**; mean outcome-only lift 0.1799; **positive lift rate 72.8%**. Six default runtime metrics grading process signals (skill execution, behavior checks, skill efficiency) "that structural scans cannot measure."
<https://arxiv.org/abs/2608.20614>

**Implication for this repo (Design decision candidate).** `profiler/experiment.go` + `profiler/compare.go` should adopt **Skill Lift** as the headline statistic and report a CI, not a point estimate. ACES's 95% CI over ~950 paired cases sets the bar: a single paired run is not evidence. The 27.2% of cases with *negative* lift (and SkillsBench's 16/84 negative-delta tasks) means the comparison engine must be able to report "this skill hurt."

### A2. Anthropic shipped in-harness skill profiling: `/skill-doctor`

**Specification** (Claude Code docs, <https://code.claude.com/docs/en/skills>):

- `/skill-doctor` reports, per skill in the session: **context cost** and **how often it was invoked**; flags listing entries never invoked and says where to turn them off.
- Scope: covers session skills **other than bundled and enterprise skills**.
- Requires **Claude Code v2.1.252 or later**. Not available in sessions that skip feature-flag fetching. Not available over Remote Control (`Skill usage reports are not available on this connection.`).
- Output: interactive → `/plugin` manager **Stats** tab; non-interactive (`-p`) → **plain text**.

> **Conflict — flagged.** Third-party reporting says it shipped in **2.1.261 on 2026-09-04** (<https://www.implicator.ai/anthropic-claude-code-skill-doctor-context-audit/>); official docs say **v2.1.252 or later**. Both can be true (feature-flagged rollout). **UNKNOWN:** the exact first version.

- **Repository fact / environment.** The local machine runs `claude 2.1.221` (`/Users/matthewvandusen/.local/share/claude/versions/2.1.221`). **`/skill-doctor` is not available here.** Any design that consumes it must gate on version.
- **UNKNOWN:** whether `/skill-doctor -p` output is stable/parseable or JSON-emitting. Docs say "prints as text". **This is the single highest-value spike** for a Claude Code token-attribution path — it is the only first-party per-skill context-cost number in the ecosystem.
- **External evidence (limitation).** It records *whether* a skill ran, not whether it *improved* results — a rarely-used-but-valuable skill is indistinguishable from dead weight. It is a **cost** instrument, not an **efficacy** instrument. (implicator.ai, above.) That is precisely the gap SkillsBench/ACES paired evaluation fills, and precisely the gap F04 targets.

### A3. `claude plugin eval` and the skill-creator eval loop

**External evidence.** Anthropic's `skill-creator` plugin ships an eval loop: test cases in `evals/evals.json`, each test case run as an isolated spawned subagent, assertion-based grading, and an HTML review viewer. A `claude plugin eval init --bare regression` command scaffolds a skills-directory plugin.
<https://github.com/anthropics/claude-plugins-official/blob/main/plugins/skill-creator/skills/skill-creator/SKILL.md> · <https://code.claude.com/docs/en/skills>

**UNKNOWN / thin evidence.** I could not reach a canonical CLI reference page for `claude plugin eval` (subcommands, flags, output schema). Everything above is from the skill-creator SKILL.md and secondary write-ups. **Do not design a hard integration against `claude plugin eval` without first running `claude plugin eval --help` on a ≥2.1.252 build.** Note this is *assertion* grading, not *paired* grading — it answers "did the skill produce the right output", not "did the skill help versus not having it".

### A4. Static tooling: what already exists (verified locally)

**Repository fact — verified by running the tools on this repo:**

`skill-validator` (Go, `/opt/homebrew/bin/skill-validator`) does emit token counts, but **only under `check`, not under `analyze content`**:

```
skill-validator check skills/skill-audit -o json  → .token_counts.files[] {file, tokens}, .token_counts.total
skill-validator analyze content skills/skill-audit -o json  → NO token_counts key
```

Measured on `skills/skill-audit`: `SKILL.md body` 1814, `references/best-practices.md` 1135, `references/checklists.md` 432, `references/evaluation-matrix.md` 649, **total 4030**.

**External evidence.** skill-validator counts with **`o200k_base`** encoding (real BPE, not an estimate). Purely static — no runtime/profiling capability.
<https://github.com/agent-ecosystem/skill-validator>

**Repository fact — char/4 calibration spike (run 2026-09-10).** Comparing cursorscope's `Math.ceil(len/4)` heuristic against skill-validator's `o200k_base` on this repo's own markdown:

| File | chars | char/4 est | o200k_base | error |
|---|---:|---:|---:|---:|
| `skills/skill-audit/SKILL.md` (body only) | 7,658 | 1,914 | 1,814 | **+5.5%** |
| `references/best-practices.md` | 4,776 | 1,194 | 1,135 | **+5.2%** |
| `references/checklists.md` | 1,991 | 497 | 432 | **+15.0%** |
| `references/evaluation-matrix.md` | 2,424 | 606 | 649 | **−6.6%** |

**Finding: char/4 lands within roughly −7% to +15% of a real BPE count on prose/markdown.** Good enough for ranking and for relative deltas; not good enough for absolute cost claims. Error is worst on dense list-heavy markdown (checklists, +15%). **Local hypothesis:** error will be materially worse on code, JSON, and non-English text, where BPE efficiency diverges most from 4 chars/token — untested here.

**Other tooling (External evidence, less verified):**

- `skillscore` — offline **Dart** CLI (also on npm and pub.dev), lints/scores SKILL.md against Claude/Codex/Antigravity authoring guides across discoverability, conciseness, structure, instruction design, hygiene, safety, with a cited source per rule. <https://github.com/sayed3li97/skillscore> · <https://pub.dev/packages/skillscore>
  - **Note for AGENTS.md accuracy:** AGENTS.md calls skillscore an "npm, 7-dimension Anthropic-aligned" tool. The upstream repo describes **six** dimensions and a Dart implementation. **Repository fact + External evidence conflict — worth a doc fix, out of scope here.**
- `@crafter/skillkit` (npm) — claims usage analytics, conflict detection, cost analysis, context-budget monitoring; **requires Bun**. <https://www.npmjs.com/package/@crafter/skillkit> — **UNVERIFIED**: npmjs.com returned HTTP 403 to automated fetch; all detail is from search snippets. Bun runtime dependency conflicts with this repo's AGENTS.md "no extra runtime dependencies" rule.
- `moutons/skills-validator`, `getskillcheck.com`, `vercel-labs/skills` (`npx skills`) — spec validation, no runtime profiling.
- **PluginEval** — third-party three-layer framework (static analysis + LLM judge + Monte Carlo) producing calibrated scores with CIs. <https://github.com/wshobson/agents/blob/main/docs/plugin-eval.md> — **UNVERIFIED**, not run.

**Newer benchmarks worth tracking (External evidence, abstracts only):**
- SkillGenBench — benchmarking skill *generation* pipelines. <https://arxiv.org/abs/2605.18693>
- SkillResolve-Bench — same-capability ambiguity in skill *retrieval*. <https://arxiv.org/pdf/2606.10388>
- MalSkills — neuro-symbolic static analyzer for malicious skills. <https://github.com/security-pride/MalSkills>

### A5. Cursor telemetry surface — the full picture

#### A5.1 Hooks (free, local, no enterprise plan)

**Specification** — <https://cursor.com/docs/agent/hooks> (also <https://cursor.com/docs/hooks>). Verified against cursorscope's registered event list (`scripts/install-global-hooks.sh:40`), which matches exactly.

**Common base fields on all agent hooks:** `conversation_id`, `generation_id`, `model`, `model_id` (opt), `model_params` (opt), `hook_event_name`, `cursor_version`, `workspace_roots`, `user_email`, `transcript_path`.

| Hook event | Payload fields beyond base | Token-bearing? |
|---|---|---|
| `sessionStart` | `session_id`, `is_background_agent`, `composer_mode` (opt) | no |
| `sessionEnd` | `session_id`, `reason`, `duration_ms`, `is_background_agent`, `final_status`, `error_message` (opt) | no |
| `beforeSubmitPrompt` | `prompt`, `attachments` | **proxy** (prompt text) |
| `preToolUse` | `tool_name`, `tool_input`, `tool_use_id`, `cwd`, `model`, `model_id`, `model_params`, `agent_message` | **proxy** (tool_input) |
| `postToolUse` | `tool_name`, `tool_input`, `tool_output`, `tool_use_id`, `cwd`, `duration`, `model`, … | **proxy** (tool_output) |
| `postToolUseFailure` | `tool_name`, `tool_input`, `tool_use_id`, `cwd`, `error_message`, `failure_type`, `duration`, `is_interrupt` | no |
| `beforeShellExecution` | `command`, `cwd`, `sandbox` | **proxy** |
| `afterShellExecution` | `command`, `output`, `duration`, `sandbox` | **proxy** (output) |
| `beforeMCPExecution` | `tool_name`, `tool_input`, `mcp_server_name`, `url`/`mcp_server_url` or `command` | **proxy** |
| `afterMCPExecution` | `tool_name`, `tool_input`, `mcp_server_name`, `mcp_server_url` (opt), `result_json`, `duration` | **sometimes real** — see below |
| `beforeReadFile` | `file_path`, `content`, `attachments` | **proxy** (content) — **also the skill-activation proxy** |
| `afterFileEdit` | `file_path`, `edits[]{old_string,new_string}` | no |
| `beforeTabFileRead` | `file_path`, `content` | proxy |
| `afterTabFileEdit` | `file_path`, `edits[]{old_string,new_string,range,old_line,new_line}` | no |
| `subagentStart` | `subagent_id`, `subagent_type`, `task`, `parent_conversation_id`, `tool_call_id`, `subagent_model`, `is_parallel_worker`, `git_branch` (opt) | no |
| `subagentStop` | `subagent_type`, `status`, `task`, `description`, `summary`, `duration_ms`, `message_count`, `tool_call_count`, `loop_count`, `modified_files`, `agent_transcript_path` | no |
| `afterAgentResponse` | `text` | **proxy** (output text) |
| `afterAgentThought` | `text`, `duration_ms` (opt) | **proxy** (reasoning) |
| `stop` | `status`, `loop_count` | no |
| **`preCompact`** | `trigger`, **`context_usage_percent`**, **`context_tokens`**, **`context_window_size`**, `message_count`, `messages_to_compact`, `is_first_compaction` | **REAL — the only one** |
| `workspaceOpen` | `hook_event_name`, `cursor_version`, `workspace_roots`, `user_email` (**no** conversation_id/generation_id) | no |

**Three findings that drive the design:**

1. **`preCompact.context_tokens` is the only real, Cursor-reported token number in the entire hook surface.** It is Cursor's own count of current context-window occupancy, alongside `context_window_size`. cursorscope uses it as the one non-estimated input: `src/telemetry.js:handlePreCompact` records it into `gen_ai.client.token.usage` with `gen_ai.token.type=input`. **But it only fires on compaction** — short sessions never emit it.
2. **`afterMCPExecution.result_json` can carry real usage** — but that is the *MCP server's* self-reported usage, not Cursor's. `src/attribution.js:extractReportedTokenUsage` probes `usage` / `token_usage` / `tokens` for `input_tokens|prompt_tokens|input`, `output_tokens|completion_tokens|output`, `total_tokens|total`, and tags the result `source: "mcp_reported"` vs `"estimated"`. Good pattern; narrow applicability.
3. **There is no skill or rule activation hook.** Nothing in the hook surface names a skill. cursorscope infers it from `beforeReadFile.file_path` / shell `command` matching `/(?:^|[/\\])skills[/\\]|SKILL\.md$/i` (`src/attribution.js:SKILL_PATH_RE`, `skillNameFromPath`), plus `subagent_type` matching `/skill/i`. **Local hypothesis (needs a spike):** Cursor loads `.cursor/skills/*/SKILL.md` through a path that fires `beforeReadFile`. If it loads skills internally without a file-read tool call, this heuristic yields nothing. Spike: register a `beforeReadFile` hook, put a skill in `.cursor/skills/`, prompt to trigger it, and check whether a `beforeReadFile` with the SKILL.md path arrives.

**Design decision worth copying from cursorscope:** it never launders an estimate as a measurement. Every attributed token carries `token_source` / `cursor.attribution.token_source` = `"estimated"` or `"mcp_reported"` (`src/telemetry.js:recordAttributedContextTokens`). This maps cleanly onto this repo's `MetricResult.Source` field in `docs/profiler-spec.md`. **Recommend adding a `SourceEstimated` MetricSource** so an estimate is never reported as `SourceOtel`/`SourceHooks`.

#### A5.2 Cursor enterprise OTel export — verified wire reference

**Specification.** <https://cursor.com/docs/enterprise/opentelemetry-export> and the Wire Reference at <https://cursor.com/docs/enterprise/opentelemetry-export/wire>. **Enterprise plan only, beta**, configured by admins in Team Settings → OpenTelemetry Export, to **one** team-managed destination.

Resource attributes: `service.name=cursor`, `cursor.team.id`, `cursor.user.id` (optional).
Join keys on logs: `cursor.event.id` (dedup), `cursor.conversation.id`, `cursor.usage_event.id` (request grain), `cursor.request.id`, `cursor.grok_bot.turn.id`.

**Metrics (delta temporality):**

| Metric | Unit | Attributes |
|---|---|---|
| `cursor.token.usage` | `{token}` | **`cursor.token.type`**: `input` \| `output` \| `cache_read` \| `cache_creation`; `cursor.model.name` (opt); `cursor.api.status`: `success`\|`errored`\|`aborted` (opt); `cursor.api.billable` bool (opt) |
| `cursor.tool.calls` | `{call}` | `cursor.tool.kind`: `builtin`\|`mcp`; `cursor.tool.name`; `cursor.tool.status`: `success`\|`failure`\|`aborted`; `cursor.mcp.server.name` (MCP only) |
| `cursor.cost.usage` | USD | `cursor.model.name` (opt) — best-effort estimates, not invoices |

**Log events:**

| Event | Severity | Attributes |
|---|---|---|
| `cursor.api.request` | INFO | **`cursor.api.request.input_tokens`**, **`.output_tokens`**, **`.cache_read_tokens`**, **`.cache_creation_tokens`** (all int); `cursor.model.name` (opt); `cursor.api.billable` (opt) |
| `cursor.api.error` | — | error events without raw messages |
| `cursor.api.correction` | — | billing finalization |
| **`cursor.skill.activated`** | INFO | **`cursor.skill.name`**; **`cursor.skill.trigger`**: `agent_read` \| `manually_attached` \| `skill_name_in_prompt`; **`cursor.skill.source`**: `unspecified`\|`workspace`\|`user`\|`builtin`\|`plugin`\|`claude`; `cursor.plugin.name` (opt) |
| `cursor.hook.execution_complete` | INFO/ERROR | **`cursor.hook.name`**; **`cursor.hook.type`**: `pre_tool_use`\|`post_tool_use`\|`post_tool_use_failure`\|`before_submit_prompt`\|`after_agent_response`\|`after_agent_thought`\|`stop`\|`subagent_start`\|`subagent_stop`; **`cursor.hook.outcome`**: `success`\|`blocked`\|`failed`\|`timeout`; **`cursor.hook.duration_ms`** int; `cursor.plugin.name` (opt) |
| `cursor.plugin.installed`, `cursor.cloud_agent.*`, `cursor.grok_bot.*` | — | lifecycle |

**This is the single most valuable finding in the ledger.** `cursor.skill.activated` with `cursor.skill.trigger` is the *only* first-party skill-activation event in any harness surveyed, and `cursor.api.request` gives **per-request four-way token splits joinable to `cursor.conversation.id`**. It makes Cursor the *best*-instrumented harness for skill profiling — but only on Enterprise.

#### A5.3 Cursor Admin API

**Specification.** <https://cursor.com/docs/account/teams/admin-api>. Basic auth (API key as username). Rate limits 20–250 req/min by endpoint. 30-day max date range. Team/Enterprise plans.

- `POST /teams/daily-usage-data` — **per-user-per-day**, no tokens. Fields: `userId`, `day`, `date`, `email`, `isActive`, `totalLinesAdded/Deleted`, `acceptedLinesAdded/Deleted`, `totalApplies/Accepts/Rejects`, `totalTabsShown/Accepted`, `composerRequests`, `chatRequests`, `agentRequests`, `cmdkUsages`, `subscriptionIncludedReqs`, `apiKeyReqs`, `usageBasedReqs`, `bugbotUsages`, `mostUsedModel`, `applyMostUsedExtension`, `tabMostUsedExtension`, `clientVersion`.
- `POST /teams/filtered-usage-events` — **per-request granularity, and it has tokens AND a conversation id.** Fields: `timestamp`, `userEmail`, `serviceAccountId`/`Name` (opt), `cloudAgentId` (opt), `automationId` (opt), **`conversationId` (opt)**, `model`, `kind`, `maxMode`, `requestsCosts`, `isTokenBasedCall`, `isChargeable`, `isHeadless`, **`tokenUsage { inputTokens, outputTokens, cacheWriteTokens, cacheReadTokens, totalCents, discountPercentOff }`**, `chargedCents`, `cursorTokenFee` (opt).
- `POST /teams/spend` — per-user `spendCents`, `overallSpendCents`, limits.
- Data is aggregated hourly; poll at most once per hour; 20 req/min per team.

**Finding:** `filtered-usage-events.conversationId` joins to hook `conversation_id`. **This is the cheapest real-token path for Cursor that does not require Enterprise OTel** — it needs a Team plan admin API key, not an Enterprise OTel pipeline.

**Repository fact.** cursorscope's poller (`src/cursor-api-poller.js`) uses `GET /teams/{teamId}/daily-usage-data` with `Authorization: Bearer`. Per the docs the endpoint is **`POST /teams/daily-usage-data`** with **Basic** auth and no team path segment, and `daily-usage-data` **has no token fields anyway**. The file even comments "This endpoint may evolve. Keep this helper easy to customize." **A Go rewrite should not port this; it should target `POST /teams/filtered-usage-events` instead.**

#### A5.4 Cursor local `state.vscdb`

**External evidence.** Two tables, both `key TEXT UNIQUE, value BLOB` holding UTF-8 JSON: `ItemTable` and `cursorDiskKV`.

- Global `globalStorage/state.vscdb` keys: `composerData:{composerId}`, `bubbleId:{composerId}:{bubbleId}`, `checkpointId:{composerId}:{checkpointId}`, `messageRequestContext:{composerId}:{messageId}`, `composer.content.{hash}`.
- Workspace `workspaceStorage/{id}/state.vscdb`: `composer.composerData` (Cursor ≤2.6) → `composer.composerHeaders` (Cursor 3.0+).
- Bubble fields: `text`, `richText`, `type`, `context`, `codeBlocks`, `suggestedCodeBlocks`, `toolResults`, `checkpointId`, `allThinkingBlocks`, `createdAt` (~60+ fields).
- **No token counts or usage data documented in bubble or composer records.**
- Verified Feb 2026 (Cursor ~2.6), updated Apr 2026 (Cursor 3.0). A **breaking, one-way migration** occurred between versions; no stability guarantee.

<https://github.com/Callum-Ward/cursaves/blob/main/docs/how-cursor-stores-chats.md> · <https://github.com/S2thend/cursor-history> · <https://vibe-replay.com/blog/cursor-local-storage/>

**Repository fact.** **Cursor is not installed on this machine** (`~/Library/Application Support/Cursor/` absent, `~/.cursor/` absent). None of the above could be empirically verified locally. Treat the schema as External evidence, **not** verified, and as **version-fragile**.

**Implication:** `state.vscdb` gives conversation reconstruction (prompts, responses, tool results, thinking blocks) → good for **estimated** tokens and for reconstructing which files were read. It does **not** give billed tokens.

### A6. OTel GenAI semantic conventions — status

**Specification.** <https://github.com/open-telemetry/semantic-conventions-genai/blob/main/docs/gen-ai/gen-ai-spans.md> (semconv **1.44.0**, OTel spec v1.56.0).

- **Status: Development.** All GenAI attributes and span types carry development badges. **Not stable — do not treat as a frozen contract.**
- Agent operations: `create_agent`, `invoke_agent`, `execute_tool`.
- Token attributes: `gen_ai.usage.input_tokens`, `gen_ai.usage.output_tokens`, `gen_ai.usage.cache_read.input_tokens`, `gen_ai.usage.cache_write.input_tokens`, `gen_ai.usage.reasoning.output_tokens`.
- Related: `gen_ai.tool.definitions`, `gen_ai.system_instructions`, `gen_ai.prompt.variable`.
- **There is NO convention for skills, rules, or prompt-fragment attribution.** No fragment-level prompt attribution exists in the standard.

**Consequence.** Any skill-level attribute is necessarily vendor-specific. Cursor invented `cursor.skill.*`; cursorscope invented `cursor.attribution.category` / `.name` / `.detail` / `.token_source`. **Design decision recommendation:** namespace this project's own skill attributes (`skill_architect.skill.name`, `.token_source`, …) rather than squatting on the `gen_ai.*` namespace, and map to `gen_ai.*` only where the convention actually defines the field.

**Repository fact.** cursorscope pins `@opentelemetry/semantic-conventions@^1.38.0` and imports agent attributes from the `/incubating` entrypoint (`src/gen-ai-semconv.js`) — i.e. upstream itself treats these as unstable.

---

## Part B — Token-usage efficiency of skills

### B1. What is always loaded, and exactly what it costs (Claude Code)

**Specification** — <https://code.claude.com/docs/en/skills>. This is the most precise public accounting of skill token cost anywhere:

- Three-level progressive disclosure: **(1) skill listing — always loaded, every turn**: only `description` + `when_to_use`. **(2) full `SKILL.md`** on invocation. **(3) supporting files** only when Claude opens them.
- > "Every skill in the skill listing adds to your context on every turn, whether or not Claude ever uses it."
- > "the combined `description` and `when_to_use` text is truncated at **1,536 characters** in the skill listing to reduce context usage."
- > "Unlike CLAUDE.md content, a skill's body loads only when it's used, so long reference material costs almost nothing until you need it."
- Recommendation: > "Keep `SKILL.md` under **500 lines**. Move detailed reference material to separate files."
- **Stickiness:** > "the rendered `SKILL.md` content enters the conversation as a single message and **stays there across later turns**." A skill invoked once keeps paying on every subsequent turn.
- **Compaction budget:** > "Claude Code re-attaches the most recent invocation of each skill after the summary, keeping the **first 5,000 tokens** of each. Re-attached skills share a combined budget of **25,000 tokens**. Claude Code fills this budget starting from the most recently invoked skill, so older skills can be dropped entirely after compaction if you have invoked many in one session."
- Mitigations: `disable-model-invocation: true`, `skillOverrides`, `/skill-doctor`.

**External evidence** — the listing budget knobs:
- **`skillListingBudgetFraction`**, default **0.01** (1% of context window), shipped in Claude Code **v2.1.129**.
- **`skillListingMaxDescChars`**, default **1536** — truncates individual descriptions before the overall budget applies.
- `SLASH_COMMAND_TOOL_CHAR_BUDGET` env var for a fixed character budget.
- When the listing overflows, Claude Code **drops descriptions starting with the least-invoked skills**.
- **Known bug:** the fraction is computed against a fixed ~200K-token reference rather than the model's real context window, so on a 1M-context model the effective budget is **~5× smaller than documented**. <https://github.com/anthropics/claude-code/issues/57941> · <https://github.com/anthropics/claude-code/issues/56966> · <https://claudefa.st/blog/guide/mechanics/skill-listing-budget>

**Repository fact — this repo's own numbers (measured 2026-09-10):**

| Skill | SKILL.md bytes | lines | char/4 est | frontmatter `description` chars | est. always-loaded tokens |
|---|---:|---:|---:|---:|---:|
| `skills/skill-audit` | 8,069 | 195 | 2,017 | 262 | ~65 |
| `skills/skill-rewrite` | 7,249 | 185 | 1,812 | 441 | ~110 |

Both are well under the 1,536-char listing cap and the 500-line body guidance. Total on-invocation cost for `skill-audit` including references is **4,030 o200k_base tokens** (skill-validator), of which **1,814 is the body** and **2,216 is references** — i.e. **55% of this skill's token mass is behind progressive disclosure** and only paid on demand.

### B2. Published numbers on size, compression, and progressive disclosure

**External evidence — SkillReducer** (arXiv:2603.29919, submitted 2026-03-31, revised 2026-06-24):
- Analyzed **55,315 public skills**. Systemic problems found: **26.4% missing routing descriptions**, **>60% excessive non-actionable content**, bloated reference files.
- Achieved **48% description compression** and **39% body compression** while **improving functional quality by 2.8%**, validated on **600 skills** + SkillsBench, model retention mean **0.965**, **5 models from 4 families**.
- > "less-is-more effect where removing non-essential content reduces distraction in the context window."
- **The abstract does NOT report absolute runtime token or cost reduction — only compression ratios.** Do not cite it for runtime savings.
<https://arxiv.org/abs/2603.29919>

**External evidence — SkillJuror** (arXiv:2606.11543, 2026-06-11, 82-task SkillsBench study, <https://github.com/zhiyuchen-ai/skill-juror>):
- From the arXiv abstract (primary): distinct Skill resources touched per trajectory rise **1.18 → 3.85**; effective uptake events rise **1.33 → 3.92**; **+17 verifier-passing trials out of 410 matched trials (+4.1%)**.
- From secondary summarization (**lower confidence — could not confirm against the PDF body**): Progressive Disclosure **46.1% pass** vs **29.0% No Skill**; yield-normalized wall-clock **20.1 → 17.8 min per strict pass**; **cost/pass nearly tied, $1.31 vs $1.28** — higher pass yield offsets higher per-attempt cost.
- **The important, well-supported conclusion:** progressive disclosure **raises resource touches ~3×**, which means it converts always-loaded cost into on-demand cost but **does not reduce total cost per successful task**. It buys *accuracy*, not *cheapness*.

**External evidence — NVIDIA SkillEvaluator Tier 3 token tracking** (2026-08-12 benchmark). Concrete published per-skill token deltas:
- `jetson-optimize-memory`: **tokens reduced 76.9%**
- `cuopt-install`: **tokens increased 120.3%**

**This is the strongest published evidence that a skill's token effect is bimodal and must be measured per-skill, not assumed.** A skill that shortcuts exploration saves massively; a skill that adds procedure can more than double token use. Both can be *good* skills.
<https://developer.nvidia.com/blog/evaluating-ai-agent-skill-performance-with-nvidia-skillevaluator/>

**Low-confidence / UNVERIFIED numbers — do NOT use.** Blog claims of "60–92% token savings from progressive disclosure" (dev.to, chudi.dev) and "85× token savings" (matthewkruczek.ai) are **reported without accuracy control** — they measure tokens not loaded, with no check that task success held constant. One secondary source cites a measured band of **26.8%–43.2%** but I could not trace it to a primary paper. **Treat all of these as marketing.** The only accuracy-controlled runtime numbers in this ledger are SkillEvaluator's and SkillJuror's.

### B3. Attributing tokens to a skill — the techniques that exist

**Repository fact — cursorscope's approach** (`/private/tmp/.../scratchpad/cursorscope`, `@last9/cursorscope@0.3.7`):

1. **Estimate:** `estimateTokens(value)` = `Math.ceil((typeof value === "string" ? value : JSON.stringify(value)).length / 4)` (`src/attribution.js`). Applied to `tool_input`/`tool_output` (`estimateToolContextTokens`), `command`/`output` (`estimateShellContextTokens`), `hookData.content` on `beforeReadFile`, and `text` on `afterAgentThought` (as `gen_ai.usage.reasoning.output_tokens`).
2. **Prefer real when available:** `estimateMcpContextTokens` first calls `extractReportedTokenUsage(hookData.result_json)` and only falls back to the estimate; tags `source: "mcp_reported"` vs `"estimated"`.
3. **Attribute:** `classifyInvocation(hookName, hookData, toolNameOverride)` buckets every invocation into `mcp` / `cli` / `skill` / `subagent` / `tool` with a `name` and `detail`, using MCP hook names + `mcp:server/tool` regexes, shell hook names, `skillNameFromPath(file_path)`, `subagent_type` matching `/skill/i`, and `command` matching the skill path regex.
4. **Emit:** counter `cursor_attributed_context_tokens_total` with `cursor.attribution.category` / `.name` / `.token_source`, plus the semconv histogram `gen_ai.client.token.usage`.
5. **Correlate:** spans nest by `conversation_id` → `generation_id` → `tool_use_id`, so tool-call spans sit under their parent prompt span.

**Honest read:** this is a **context-bytes-admitted** meter, not a token-billing meter. It counts what tool payloads *carried*, and labels it as estimated. It is the right shape; the number is a proxy.

**The gap it cannot close (Local hypothesis, well-founded, untested here):** billed input tokens per turn are approximately the *cumulative* conversation re-sent each turn, not the per-event payload size. Summing per-event payload estimates therefore **systematically undercounts billed input tokens** — the true curve is roughly quadratic in turn count while the sum-of-payloads is linear — and it completely misses the system prompt, the always-loaded skill listing, rules, and cache-read vs cache-creation splits. `preCompact.context_tokens` is the only hook datum that reflects the true cumulative figure, and it fires only at compaction. **This gap is the reason Q1 below answers "no."**

**Other attribution techniques surveyed:**
- **First-party per-skill accounting** — Claude Code `/skill-doctor` (per-skill context cost + invocation count). Best-in-class, Claude-Code-only, ≥2.1.252.
- **First-party activation events** — Cursor `cursor.skill.activated` (`cursor.skill.name`, `.trigger`, `.source`). Enterprise-only.
- **Static BPE accounting** — `skill-validator check -o json` `.token_counts` with `o200k_base`. Exact for "what this skill would cost if fully loaded", says nothing about what was actually loaded.
- **Paired A/B differencing** — SkillsBench / ACES / SkillEvaluator Tier 3. **This is the only method that attributes tokens to a skill without needing per-fragment instrumentation**: run the same task twice, difference the totals. It requires no vendor cooperation and works on every harness that reports session totals. It is also the method this repo has already chosen (`profiler/experiment.go`, `profiler/compare.go`).

---

## Decision questions

### Q1. Can Cursor hook payloads alone (no enterprise OTel) give per-session token counts? If not, what is the best available proxy and how good is it?

**No.** Verified against the official hook reference: of the 21 hook events, exactly one carries a Cursor-reported token number — `preCompact.context_tokens` (with `context_window_size`, `context_usage_percent`) — and it only fires when the session compacts, so short sessions emit nothing. `afterMCPExecution.result_json` sometimes carries a `usage` block, but that is the MCP server's self-report, not Cursor's. No hook reports input/output/cache-read/cache-creation for a model call. **The best available proxy is cursorscope's approach: sum `Math.ceil(len/4)` over `beforeSubmitPrompt.prompt`, `postToolUse.tool_output`, `afterShellExecution.output`, `beforeReadFile.content`, `afterMCPExecution.result_json`, `afterAgentResponse.text`, and `afterAgentThought.text`, keyed by `conversation_id`/`generation_id`, and tag it `token_source=estimated`.** Its quality: the char/4 heuristic itself is accurate to **−7%…+15%** against `o200k_base` on this repo's markdown (measured, §A4), but that is the *small* error. The *large* error is structural: the sum measures context bytes admitted per event, whereas billed input tokens are the cumulative conversation re-sent every turn, so the proxy undercounts billed input by a growing margin over a session and misses the system prompt, the always-loaded skill listing, and the cache-read/cache-creation split entirely. **Recommendation: treat the hook-derived number as an ordinal signal only — valid for ranking two skills in a paired A/B where turn counts are comparable, invalid for any absolute cost or dollar claim — and never surface it as `SourceOtel`. Add a `SourceEstimated` value to `MetricSource` in `docs/profiler-spec.md` and require it here.** When a real number is needed without Enterprise OTel, the answer is not hooks: it is `POST /teams/filtered-usage-events` on the Admin API (Team plan admin key), which returns per-request `tokenUsage{inputTokens, outputTokens, cacheWriteTokens, cacheReadTokens}` joined by `conversationId` — the same id the hooks emit. That join is the recommended architecture: hooks for structure and attribution, Admin API for the true totals.

### Q2. Is there a Go OTel SDK path that keeps a Go rewrite of cursorscope dependency-light? Name the packages and versions.

**There is a supported path, but it is not light, and I recommend against it for the export side.** Measured on 2026-09-10 with Go 1.27.1 via `proxy.golang.org` and two throwaway builds. Current versions: `go.opentelemetry.io/otel` **v1.46.0** (2026-08-25), `go.opentelemetry.io/otel/sdk` **v1.46.0**, `.../exporters/otlp/otlptrace/otlptracehttp` **v1.46.0**, `.../exporters/otlp/otlpmetric/otlpmetrichttp` **v1.46.0**, `.../exporters/otlp/otlplog/otlploghttp` **v0.22.0** (logs still pre-1.0), `go.opentelemetry.io/otel/log` **v0.22.0**, `go.opentelemetry.io/proto/otlp` **v1.11.0**. A minimal `otlptracehttp` + `otlploghttp` program requires **2 direct + 22 indirect modules** and — despite being HTTP-only — compiles **66 `google.golang.org/grpc` packages**, plus `genproto`, `grpc-gateway/v2`, and `protobuf`, for **408 packages** and a **17.7 MB** arm64 binary. The equivalent stdlib-only program that POSTs OTLP/JSON to `:4318/v1/traces` has **zero** module requirements and a **6.4 MB** binary. Context: `profiler/go.mod` today declares **no dependencies at all**, and AGENTS.md's self-contained principle exists precisely to protect that. **Recommendation: do not adopt the OTel Go SDK for export. Hand-roll OTLP/HTTP JSON (`Content-Type: application/json` to `{endpoint}/v1/traces|/v1/logs|/v1/metrics`), which is a first-class, spec-mandated OTLP encoding that every collector accepts, using `encoding/json` + `net/http` only.** The tradeoff is real and should be named: you give up batching/retry/backoff, the semconv constant packages, context propagation, and automatic resource detection, and you take on schema drift against a moving OTLP JSON schema. That is an acceptable trade here because the profiler emits a bounded, hand-authored set of ~6 span shapes and ~5 metrics, not arbitrary third-party instrumentation. If a future requirement forces the SDK (e.g. gRPC transport, or consuming third-party instrumentation), isolate it: put the exporter behind a `ProfileExporter` interface in a **separate Go module** under `profiler/exporters/otlp/` so the core `profiler` module stays dependency-free. Note separately that for *reading* Cursor's OTel export — what `profiler/cursor.go` actually does today — no OTel dependency is needed at all; `encoding/json` over a collector-written file is correct and already implemented.

### Q3. What can semgrep realistically contribute to skill profiling or token efficiency, versus what needs runtime signals?

**Semgrep's realistic contribution is narrow and mostly already covered by tools this repo has.** First, an environment fact: **semgrep is not installed on this machine**, so nothing below was empirically verified — it is reasoning from semgrep's documented capabilities, and should be labeled Local hypothesis until a spike runs. Semgrep is a syntactic pattern matcher over parsed source; markdown prose is not a language it parses semantically, so `SKILL.md` bodies would have to go through `generic` mode (whitespace-tolerant token matching), which gives you roughly what `grep` already gives you. It **cannot** do the thing that matters most — token-cost estimation — because that is a character/BPE counting problem, and `skill-validator check -o json` already answers it exactly with `o200k_base` (§A4). What semgrep *can* do that grep cannot: (a) parse the **YAML frontmatter** structurally to enforce always-loaded budget rules — `description` + `when_to_use` combined length ≤1,536 chars, presence of `disable-model-invocation`, malformed `allowed-tools`; (b) lint the **`scripts/` bash and Go** in a skill bundle for the determinism/error-handling/no-absolute-paths rules in PRINCIPLES.md, which is squarely semgrep's home turf and is the one place it beats every alternative; (c) detect **always-loaded bloat patterns** like a large fenced block or inlined reference table sitting in `SKILL.md` rather than behind a `references/` link — but that is a structural heuristic that a small Go pass over the markdown AST does more accurately and without a Python runtime dependency, which AGENTS.md forbids introducing. Everything about *whether the skill was loaded, when, how often, and at what real cost* is unreachable by static analysis and needs runtime signals: `/skill-doctor` (per-skill context cost, Claude Code ≥2.1.252), `cursor.skill.activated` (Cursor Enterprise OTel), hook-derived estimates (Cursor, ordinal only), or paired A/B differencing (any harness). **Recommendation: do not add semgrep for skill profiling or token estimation — it duplicates `skill-validator` and adds a Python runtime that AGENTS.md rules out. Consider it only as an optional, opt-in linter for the `scripts/` directory of a skill bundle, invoked as an external tool with graceful degradation when absent, and never on the critical path of an audit.** The tradeoff named: you forgo a mature rule ecosystem for bundled scripts, in exchange for keeping the plugin self-contained and single-runtime.

### Q4. What does `profiler/cursor.go` assume about Cursor OTel event names, and are those names real?

**All four names are real and verified — but every attribute key the file reads is wrong, and one event is semantically misused.** Verified against the Cursor OpenTelemetry Export Wire Reference (<https://cursor.com/docs/enterprise/opentelemetry-export/wire>): `cursor.token.usage` (metric), `cursor.api.request`, `cursor.skill.activated`, and `cursor.hook.execution_complete` (log events) all exist exactly as spelled. **Nothing is invented.** The prior design note (`tmp/teams/architect/profiler-design-space.md`) that predicted them was right. The defects are one level down. (1) `extractCursorTokenCounts` reads `m.Attributes["token_type"]`; the real key is **`cursor.token.type`**, so every token count silently sums to zero against a real export. Its `reasoning`/`reasoning_output` cases are dead — the enum is only `input|output|cache_read|cache_creation`. (2) `extractCursorSkillActivation` reads `skill_name` and `trigger`; the real keys are **`cursor.skill.name`** and **`cursor.skill.trigger`** (`agent_read|manually_attached|skill_name_in_prompt`), and it ignores **`cursor.skill.source`**, which is the field that tells you whether the skill came from workspace, user, builtin, plugin, or `claude`. (3) The serious one: `extractCursorToolCalls` treats `cursor.hook.execution_complete` as a **tool-call** event and reads `tool_name` / `is_error` from it. That event is about **hook** executions, not tool calls — its attributes are `cursor.hook.name`, `cursor.hook.type`, `cursor.hook.outcome`, `cursor.hook.duration_ms`. Tool calls live in the **`cursor.tool.calls` metric** (`cursor.tool.kind`, `cursor.tool.name`, `cursor.tool.status`), which the adapter never reads. So `MetricToolCalls` is wired to the wrong source and would report nothing on a real export — or, worse, report hook invocations as tool calls. (4) `extractCursorTiming` uses only `cursor.api.request` timestamps and throws away the four token fields that event carries (`cursor.api.request.input_tokens|.output_tokens|.cache_read_tokens|.cache_creation_tokens`) — those are per-request and joinable via `cursor.conversation.id`, strictly better than the aggregate metric for session-scoped profiling. (5) `Probe()` sets `caps[MetricTokens] = SourceSQLite` when a `state.vscdb` exists, but the documented schema contains **no token data**, and `Capture()` then returns `unknown` — a capability report that promises what it cannot deliver, violating the spec's own "capability vs result" contract. **Recommendation: file a correction story before F04 relies on Cursor at all. Fix the four attribute-key sets to their `cursor.*` namespaced forms; rewire `MetricToolCalls` to the `cursor.tool.calls` metric and add a separate hook-execution signal if hook health is wanted; source tokens primarily from `cursor.api.request` per-request fields (falling back to the `cursor.token.usage` metric) so sessions can be scoped by `cursor.conversation.id`; add `cursor.skill.source` to `ActivationEntry`; and downgrade the SQLite token capability to `SourceNone` with the reason "Cursor state.vscdb contains no token data".** The tradeoff: these are all corrections to unreleased code with no external consumers, so the cost is one story; the cost of not doing it is an adapter that returns confidently-shaped zeros against real Enterprise data, which is worse than returning `unknown`.

---

## What is UNKNOWN or unverifiable

1. **`/skill-doctor` machine-readable output.** Docs say `-p` "prints as text". Whether that text is stable or parseable is unknown. Local Claude Code is 2.1.221 — below the ≥2.1.252 requirement — so this could not be tested. **Highest-value spike.**
2. **Exact `/skill-doctor` first version.** Docs say ≥2.1.252; press says 2.1.261 on 2026-09-04.
3. **`claude plugin eval` CLI surface.** No canonical reference page reached. Subcommands, flags, and output schema unverified.
4. **SkillsBench task/domain/trajectory counts.** arXiv abstract (87 tasks / 8 domains / 18 configs) conflicts with secondary summaries (86 / 11 / 7 configs / 7,308 trajectories). PDF body not read.
5. **SkillJuror per-arm pass rates and cost/pass.** The 46.1% / 29.0% / $1.31 vs $1.28 figures come from a search summary, not the PDF body (PDF text streams did not extract). The arXiv abstract figures (1.18→3.85 resources touched, +4.1% on 410 matched trials) are confirmed.
6. **Cursor `state.vscdb` schema.** Cursor is not installed here. External evidence only, verified Feb/Apr 2026 against Cursor 2.6/3.0, with a documented one-way breaking migration. Version-fragile.
7. **Whether Cursor's skill loading fires `beforeReadFile`.** cursorscope's entire local skill-attribution heuristic depends on it. Untested. Spike defined in §A5.1.
8. **char/4 accuracy on code/JSON/non-English.** Calibrated here only on this repo's English markdown.
9. **`@crafter/skillkit` capabilities.** npmjs.com returned 403 to automated fetch; all detail is from search snippets.
10. **Whether Cursor's Enterprise OTel `cursor.skill.activated` fires for `.cursor/skills/` workspace skills specifically**, versus only marketplace/plugin skills. The `cursor.skill.source` enum includes `workspace`, which strongly suggests yes, but this is inference, not confirmation.
11. **`gen_ai.*` semconv stability.** Status is Development at semconv 1.44.0. Attribute names may change.
12. **Semgrep behaviour on SKILL.md.** Not installed locally; §Q3 is reasoning from documented capability, not measurement.

## Sources

- SkillsBench — <https://arxiv.org/abs/2602.12670>
- ACES / Skill Lift — <https://arxiv.org/abs/2608.20614>
- NVIDIA SkillEvaluator — <https://developer.nvidia.com/blog/evaluating-ai-agent-skill-performance-with-nvidia-skillevaluator/> · <https://github.com/NVIDIA/SkillEvaluator> · <https://docs.nvidia.com/skills/skillevaluator>
- SkillReducer — <https://arxiv.org/abs/2603.29919>
- SkillJuror — <https://arxiv.org/abs/2606.11543> · <https://github.com/zhiyuchen-ai/skill-juror>
- SkillGenBench — <https://arxiv.org/abs/2605.18693>
- SkillResolve-Bench — <https://arxiv.org/pdf/2606.10388>
- Claude Code skills (token budgets, `/skill-doctor`) — <https://code.claude.com/docs/en/skills>
- `skillListingBudgetFraction` bug — <https://github.com/anthropics/claude-code/issues/57941> · <https://github.com/anthropics/claude-code/issues/56966>
- `/skill-doctor` press — <https://www.implicator.ai/anthropic-claude-code-skill-doctor-context-audit/>
- Cursor hooks — <https://cursor.com/docs/agent/hooks> · <https://cursor.com/docs/hooks>
- Cursor OTel export — <https://cursor.com/docs/enterprise/opentelemetry-export>
- Cursor OTel **Wire Reference** — <https://cursor.com/docs/enterprise/opentelemetry-export/wire>
- Cursor Admin API — <https://cursor.com/docs/account/teams/admin-api> · <https://cursor.com/docs/account/teams/analytics-api>
- Cursor `state.vscdb` — <https://github.com/Callum-Ward/cursaves/blob/main/docs/how-cursor-stores-chats.md> · <https://github.com/S2thend/cursor-history>
- OTel GenAI semconv (spans) — <https://github.com/open-telemetry/semantic-conventions-genai/blob/main/docs/gen-ai/gen-ai-spans.md>
- OTel Go — <https://github.com/open-telemetry/opentelemetry-go> · <https://pkg.go.dev/go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp>
- cursorscope — <https://github.com/last9/cursorscope> (`@last9/cursorscope@0.3.7`; local clone at `/private/tmp/claude-501/-Users-matthewvandusen-Development-Auraprix-skill-architect/f2fa2952-c83d-4122-afe9-0e05a74dddd7/scratchpad/cursorscope`)
- skill-validator — <https://github.com/agent-ecosystem/skill-validator>
- skillscore — <https://github.com/sayed3li97/skillscore> · <https://pub.dev/packages/skillscore>
- skillkit — <https://www.npmjs.com/package/@crafter/skillkit> (403 to automated fetch)
- MalSkills — <https://github.com/security-pride/MalSkills>
