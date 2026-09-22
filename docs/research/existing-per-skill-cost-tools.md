# Existing per-skill cost tools for Cursor — survey

**Date:** 2026-09-10
**Method:** Deep-research workflow — 5 search angles (Cursor-native analytics and roadmap signals; observability vendor integrations; independent GitHub projects parsing local Cursor state; VS Code / Cursor extensions for cost display; skills evaluation and benchmark tooling with Cursor adapters), 19 sources fetched, 92 claims extracted, 25 claims put to 3-vote adversarial verification (24 confirmed, 1 refuted), 101 agent calls.

> **Note on the brief:** the dispatch said "15 sources fetched." The workflow result reports `sourcesFetched: 19` and lists 19 source records. The higher, primary-source number is used here (External evidence).

## Question as asked

> What existing tools, products, or open-source projects (as of September 2026) can track the token usage or dollar cost of individual Agent Skill invocations (or Cursor rules / `.cursor/skills`) inside Cursor — attributing cost per skill run, not just per session or per team?
>
> Already known and excluded from "new" findings but kept as comparison baselines: `last9/cursorscope` (Node.js hook→OTel exporter with chars/4 estimates), Cursor Enterprise OpenTelemetry export (`cursor.skill.activated`, `cursor.api.request` token attrs), Cursor Admin API `filtered-usage-events` (per-request `tokenUsage` joined by `conversationId`), and Anthropic Claude Code's `/skill-doctor` (Claude Code only, not Cursor).
>
> For each candidate report: what it measures (billed tokens vs estimates), attribution granularity (skill / conversation / session / team), data source (hooks, OTel, SQLite, Admin API, proxy), language/runtime, license, maturity, and a URL.

## Answer

As of September 2026, **no tool, product, or open-source project beyond the already-known baselines attributes token usage or dollar cost to individual Agent Skill or Cursor rule invocations inside Cursor** (External evidence). Every candidate surveyed stops at a coarser boundary: Cursor's own Teams/Enterprise Usage Analytics dashboard reports per user/team/model/repo/product-surface, and the Cursor 3.9/3.11 Customize-page skills leaderboard shows only invocation counts, not tokens or dollars; hook-to-OTel exporters (Dash0 Agent Plugin, `o11y-dev/opentelemetry-hooks`) emit per-turn/session spans with no skill signal; local `state.vscdb` readers (CodeBurn, Token Use, cursor-chronicle, `Pryowin/token_tracker`) attribute per conversation/turn/repo using stored-but-unreliable or `chars/4`-estimated counts; and dashboard-API scrapers (`xiufengsun/TokenTracker`, ElisCarvalho Cursor Cost Tracker) yield Cursor-billed figures but only per request/model/project. The only public evidence of demand for per-skill analytics is an unanswered Feb 2026 Cursor forum feature request. The practical path to per-skill cost in Cursor therefore remains a **custom join of the Enterprise OTel `cursor.skill.activated` event with per-request token usage keyed on `conversationId`** — which no tool found packages.

## Comparison table

Baseline rows (already known, listed for comparison) are marked **[baseline]**.

| Tool | What it measures | Attribution granularity | Data source | Runtime/lang | License | Maturity | Skill signal? | URL |
|---|---|---|---|---|---|---|---|---|
| **[baseline]** `last9/cursorscope` | Estimate (`chars/4`) | Invocation category (incl. a heuristic `skill` class from `/skills/` or `SKILL.md` file paths), session | Cursor hooks → OTLP | Node ≥20 | MIT | v0.3.7, active | Partial — heuristic, inferred from file reads, not a Cursor skill event | https://github.com/last9/cursorscope |
| **[baseline]** Cursor Enterprise OpenTelemetry export | Billed token attrs on `cursor.api.request` | Request + a separate `cursor.skill.activated` event; join is the caller's problem | Cursor-native OTel export | Vendor-side | Proprietary (Enterprise plan only) | GA | **Yes** — `cursor.skill.activated` | https://cursor.com/docs/enterprise/opentelemetry-export/wire |
| **[baseline]** Cursor Admin API `filtered-usage-events` | Billed tokens + cost (`tokenUsage`, `totalCents`) | Per request/usage event, keyed by `conversationId` | Admin API (documented) | HTTP/JSON | Proprietary (Teams/Enterprise admin) | GA | No | https://cursor.com/docs/account/teams/admin-api |
| **[baseline]** Claude Code `/skill-doctor` | Context/diagnostic audit, not billed cost | Per skill | Claude Code internal | Proprietary | Proprietary | GA | **Yes** — but Claude Code only, no Cursor path | https://code.claude.com/docs/en/skills |
| Cursor native Usage Analytics (Teams/Enterprise) | No tokens or dollars per skill at all; dashboard aggregates only | User, team, model, repository, file extension, git commit, AD group, date range, product surface | Cursor-native | Vendor-side | Proprietary | GA product | No — Customize-page skills leaderboard is 30-day **invocation counts** only | https://cursor.com/docs/account/teams/analytics |
| Dash0 Agent Plugin (`dash0hq/dash0-agent-plugin`) | Hook-supplied token attrs (`gen_ai.usage.input_tokens` / `output_tokens` / `cache_read.input_tokens`); billed-vs-estimated not stated | Per agent turn (one `chat default` span), per tool call (`execute_tool`), session, team | Cursor hooks (9 events) → OTLP | **Go binary + shell bootstrap** | Apache-2.0 | v0.1.28 (2026-09-02), active but early | No | https://github.com/dash0hq/dash0-agent-plugin |
| `o11y-dev/opentelemetry-hooks` | Passthrough `gen_ai.usage.*` only when the hook payload supplies them; **no** estimation, **no** dollar cost | Session → generation (turn) → hook event, keyed by `gen_ai.client.session_id` / `generation_id` / `conversation.id` | Hooks → any OTLP backend; 8 agents incl. Cursor IDE + Cursor CLI | Python 3.12+ | MIT | v0.14.0 (2026-07-22), ~34 stars | No — zero `skill` hits in README or code | https://github.com/o11y-dev/opentelemetry-hooks |
| `Pryowin/token_tracker` | Input from SQLite fields; output **estimated** via context-window delta (author rates it "Medium (estimated)"); optional WorkOS billing sync for real billed tokens | Per git repository and branch | `state.vscdb` / `ai-code-tracking.db` + hooks (`sessionStart`, `beforeSubmitPrompt`, `stop`, `sessionEnd`) | Python 3.10 stdlib, macOS-only v1 | MIT | 2 commits (2026-06-17), 0 stars — very immature | No | https://github.com/Pryowin/token_tracker |
| Token Use (`russmckendrick/tokenuse`) | Cursor's context meter (`promptTokenBreakdown.totalUsedTokens`) when present, else `ceil(chars/4)`; flags `token_quality` exact/estimated/mixed. Current builds write `{0,0}` per bubble, so effectively estimates | Per conversation (composer UUID) and per turn | Local SQLite read-only (`state.vscdb`, `store.db`, `ai-code-tracking.db`, transcript JSONL) | Rust | MIT | v1.2.5 (2026-09-07), ~32 stars, active | No — Coach docs **explicitly defer** skill/subagent/slash-command detection for lack of invocation signals | https://github.com/russmckendrick/tokenuse |
| CodeBurn (`getagentseal/codeburn`) | Self-labelled **estimates**: input from per-conversation context meter credited once, output a reply-text estimate, cache tokens unavailable locally, `CHARS_PER_TOKEN=4` fallback, Auto-mode priced at Sonnet rates; undercounts the admin console | Per conversation / model / project / task | Local `state.vscdb` (`cursorDiskKV` bubbles + `agentKv`) | TypeScript | MIT | v0.9.24 (2026-09-04), ~10.9k stars — most popular in the survey | No for Cursor — its per-skill cataloging applies only to `~/.claude/` (Claude Code) | https://github.com/getagentseal/codeburn |
| `mikhailsal/cursor-chronicle` | Stored per-bubble `tokenCount` verbatim; **no** estimation, **no** dollar cost. Cursor staff say the stored field is best-effort, often 0 — so "exact" is unreliable | Per conversation (`composerId`), per bubble (`bubbleId`), per tool call (`toolFormerData`); aggregates by project/day | `state.vscdb` read-only (`mode=ro`, `PRAGMA query_only=ON`) | Python 3.8+, stdlib-only | AGPL-3.0 (LICENSE) with an inconsistent MIT note in README | v1.8.1, last commit 2026-07-20, ~20 stars | No — a `cursorRules` bubble field is used only as a model-inference hint | https://github.com/mikhailsal/cursor-chronicle |
| TokenTracker (`xiufengsun/TokenTracker`) | **Billed** tokens and cost (Date, Model, Input w/ and w/o cache write, Cache Read, Output, Total, Cost, Kind, Max Mode) | Per request / model / provider / project-session / git commit | Undocumented dashboard endpoints (`export-usage-events-csv`, `usage-summary`, `api2.cursor.sh DashboardService`) + `state.vscdb` for the auth token | JavaScript / Node | MIT | ~1,563 stars, pushed 2026-09-10, active | No — its "Skills" feature is a browse/sync leaderboard of public skills, not a cost boundary; Cursor is not even a sync target | https://github.com/xiufengsun/TokenTracker |
| Cursor Cost Tracker (`ElisCarvalho.cursor-cost-tracker`) | **Billed** per usage event (`tokenUsage.*Tokens`, `totalCents`); README itself says "approximate cost per chat interaction" | Per usage event/interaction, day, billing cycle | Undocumented `get-filtered-usage-events` + `get-team-spend` via synthesized `WorkosCursorSessionToken`; `state.vscdb` for the auth token | TypeScript (VS Code / Open VSX extension) | MIT | v0.2.6 (2026-02-25), ~1,408 Open VSX downloads / 24 Marketplace installs; declared source repo 404s | No — compiled VSIX has zero `skill`, `cursorrules`, `.cursor/rules`, or `conversationId` references | https://marketplace.visualstudio.com/items?itemName=ElisCarvalho.cursor-cost-tracker |

**13 rows: 4 baselines + 9 newly surveyed Cursor-capable tools.**

## Per-finding detail

### F-01 — Cursor's native analytics has no per-skill cost dimension

- **Claim:** Cursor's Usage Analytics dashboard (Teams/Enterprise) has no per-skill or per-rule dimension and presents no token counts or dollar cost per skill, rule, or conversation. Its granularity is per user, team, model, repository, file extension, git commit, AD group, date range, and (since the 2026-05-04 changelog) product surface. The Cursor 3.9/3.11 Customize page adds a skills leaderboard with 30-day invocation counts and agent- vs human-initiated share, but no token or dollar figures. A Feb 2026 forum feature request for a skills adoption analytics endpoint has no staff reply and no shipped feature.
- **Confidence:** high — **Vote:** 2-1 and 3-0 (merged)
- **Evidence (External evidence):** Direct fetch of the analytics docs (2026-09-10) shows 32 headings, none mentioning tokens, spend, cost, skills, rules, plugins, or MCP; grep for skill/rule hits only the sidebar nav. The only monetary reference is a fee note for Conversation Insights inference. The 2026-05-04 changelog adds only user and product-surface filters. The skills leaderboard is counts-only and lives on Customize, not Analytics.
- **Sources:** `cursor.com/docs/account/teams/analytics`, `cursor.com/changelog/05-04-26`, `forum.cursor.com/t/.../151688`

### F-02 — Dash0 Agent Plugin: 9 Cursor hooks → OTLP in Go, but per-turn only

- **Claim:** `dash0hq/dash0-agent-plugin` (Apache-2.0, Go binary with shell bootstrap, v0.1.28 released 2026-09-02) registers nine Cursor hook events (`sessionStart`, `sessionEnd`, `beforeSubmitPrompt`, `afterAgentResponse`, `preToolUse`, `postToolUse`, `postToolUseFailure`, `subagentStart`, `subagentStop`) and emits OpenTelemetry traces to Dash0, but attributes tokens only per turn: one `chat default` span per agent turn carrying `gen_ai.usage.input_tokens` / `output_tokens` / `cache_read.input_tokens`, plus one `execute_tool` span per tool call. There is no skill-activation span and no per-skill/per-rule cost attribute.
- **Confidence:** high — **Vote:** 3-0, 3-0 (merged)
- **Evidence (External evidence):** `cursor/hooks.json` contains exactly the nine keys, all routed to `./cursor/cursor-on-event.sh`; `.cursor-plugin/README.md` documents the per-turn chat span and `execute_tool` child spans; the only skill mentioned is the plugin's own `/dash0-configure` setup skill. Cursor subagent work produces no span (documented QA finding). The README does not state whether token counts are billed or estimated.
- **Sources:** `github.com/dash0hq/dash0-agent-plugin`, `dash0.com/hub/integrations/int_cursor_plugin/overview`

### F-03 — `o11y-dev/opentelemetry-hooks`: multi-agent hook exporter, passthrough tokens, no skill signal

- **Claim:** `o11y-dev/opentelemetry-hooks` (Python 3.12+, MIT, v0.14.0 released 2026-07-22, ~34 stars) is a hook-based OTel exporter supporting Cursor IDE and Cursor CLI among eight agents (Antigravity, Claude Code, Codex, Cursor, Gemini CLI, GitHub Copilot, OpenCode, Windsurf). It passes through `gen_ai.usage.*` token attributes only when the hook payload supplies them, performs no independent estimation, and computes no dollar cost. Attribution is session → generation (turn) → hook event, keyed by `gen_ai.client.session_id` / `generation_id` / `conversation.id`. Neither README nor code mentions skills or rules.
- **Confidence:** high — **Vote:** 3-0, 3-0, 2-1 (merged)
- **Evidence (External evidence):** `pyproject.toml` confirms name, MIT, `requires-python >=3.12`, `otel-hook` entry point; `otel_hook.py` ~4589-4616 is the only `gen_ai.usage.*` code and reads input/output/cache tokens from the payload via `_set_if_present`; repo-wide grep for `cost` / `estimat` / `chars/4` / `tiktoken` / `price` returns nothing relevant; `grep -i skill` returns zero hits in README and code. Cursor setup writes `~/.cursor/hooks.json` or `.cursor/hooks.json`; supported Cursor events are `preToolUse` / `postToolUse` / `postToolUseFailure`. Default mode emits only length + SHA-256 of prompts.
- **Sources:** `github.com/o11y-dev/opentelemetry-hooks`, `pypi.org/project/opentelemetry-hooks/`

### F-04 — `Pryowin/token_tracker`: repo/branch granularity, mostly estimated

- **Claim:** `Pryowin/token_tracker` (Python 3.10 stdlib, MIT, macOS-only v1, 2 commits dated 2026-06-17, 0 stars) tracks Cursor token usage per git repository and branch only. Input tokens are read from Cursor's local SQLite (`contextWindowStatusAtCreation.tokensUsed`, then `tokenCount`, then `percentageRemaining` fallback); output tokens are estimated via a context-window delta, which the author rates "Medium (estimated)" while pointing to the Cursor dashboard as the source of truth. An optional WorkOS session-token billing sync can pull real billed tokens. No per-skill or per-rule attribution exists.
- **Confidence:** high — **Vote:** 3-0, 3-0 (merged)
- **Evidence (External evidence):** README line 1 states repo/branch tracking; `scripts/report.py` exposes only `--last` / `--since` / `--repo` / `--branch` / `--format`; grep for skill/rules/`.mdc` finds only a bundled `skills/token-report/SKILL.md` that explains reports. Uses `~/.cursor/hooks.json` (`sessionStart`, `beforeSubmitPrompt`, `stop`, `sessionEnd`) for branch snapshots plus `ai-code-tracking.db` and workspace storage for repo. Raw data is `GROUP BY conversationId` internally before roll-up.
- **Sources:** `github.com/Pryowin/token_tracker`

### F-05 — Token Use (Rust): conversation/turn granularity and an explicit deferral of skill detection

- **Claim:** Token Use (`russmckendrick/tokenuse`, Rust, MIT, v1.2.5 released 2026-09-07, ~32 stars) reads Cursor's local SQLite read-only (`state.vscdb` globalStorage, `~/.cursor/chats/<workspace>/<conversation>/store.db`, `~/.cursor/ai-tracking/ai-code-tracking.db`, plus agent-transcript JSONL) with `mode=ro` — never via hooks, OTel, or the Admin API. It attributes per conversation (composer UUID) and per turn, uses Cursor's context meter (`composerData.promptTokenBreakdown.totalUsedTokens`) when present and otherwise `ceil(chars/4)`, marking `token_quality` exact/estimated/mixed; current Cursor builds write `{0,0}` per-bubble counts, so figures are effectively estimates. Its Coach engine docs explicitly defer skill/subagent/slash-command detection because invocation signals are not stored.
- **Confidence:** high — **Vote:** 3-0, 3-0, 3-0 (merged)
- **Evidence (External evidence):** `src/tools/cursor/parser.rs:92-101` opens `file:{path}?mode=ro` with `SQLITE_OPEN_READ_ONLY`; `config.rs` defines `STATE_DB`, `AGENT_TRACKING_DB`, `STORE_DB`; grep finds no HTTP client or `cursorAuth` / `api2` references in the Cursor module. Docs state the composer UUID is the canonical session id, one `ParsedCall` per user request, and "A missing input or output side alone is estimated as `ceil(chars / 4)`". Coach docs, verbatim: "Ghost agents/skills/slash-commands detection needs invocation signals ... the archive does not yet store".
- **Sources:** `tokenuse.app/docs/development/tools/cursor/`, `tokenuse.app/docs/development/coach/`, `github.com/russmckendrick/tokenuse`

### F-06 — CodeBurn: the most popular tool in the survey, and its Cursor path is estimates-only

- **Claim:** CodeBurn (`getagentseal/codeburn`, TypeScript, MIT, ~10.9k stars, v0.9.24 released 2026-09-04, pushed 2026-09-10) reads Cursor usage from the local `state.vscdb` SQLite under globalStorage (bubbles and `agentKv` rows in `cursorDiskKV`) — not hooks, OTel, the Admin API, or a proxy. Input tokens come from Cursor's per-conversation context meter (`composerData.promptTokenBreakdown`) credited once per conversation; output is a reply-text estimate; cache tokens are unavailable locally; Cursor v3 reports zero per-bubble token counts so it falls back to `CHARS_PER_TOKEN=4`; Auto-mode costs are priced at Sonnet rates. Figures are self-labelled estimated and undercount the Cursor admin console. Attribution is per conversation/model/project/task; its per-skill cataloging applies only to `~/.claude/` (Claude Code), and hook/guard features are Claude Code only.
- **Confidence:** high — **Vote:** 3-0 ×4 (merged)
- **Evidence (External evidence):** README "How it reads your data" table and `docs/providers/cursor.md` list platform paths for `state.vscdb` and a cache at `~/.cache/codeburn/cursor-results.v<n>.json`; README verbatim: "Output is a reply-text estimate and cache tokens are server-side only, so figures are marked estimated and undercount the Cursor admin console for long conversations"; grep for hook/OTel/Admin API finds only Claude Code guard hooks, an Antigravity status-line hook, and a Copilot `agent-traces.db`. Release notes through v0.9.24 never mention skills, rules, `.cursor/rules`, or `.cursor/skills`. Issue #114 (macOS zero usage) closed via PR #145.
- **Sources:** `github.com/getagentseal/codeburn`, `github.com/getagentseal/codeburn/blob/main/docs/providers/cursor.md`

### F-07 — `cursor-chronicle`: per-bubble counts read verbatim, but Cursor staff say the field is unreliable

- **Claim:** `mikhailsal/cursor-chronicle` (Python 3.8+ stdlib-only, AGPL-3.0 LICENSE with an inconsistent MIT note in README, v1.8.1, last commit 2026-07-20, ~20 stars) is a CLI that reads Cursor's `state.vscdb` read-only (`mode=ro`, `PRAGMA query_only=ON`) for chat search, token usage stats, and tool-call analysis. Token counts are read verbatim from stored per-bubble `tokenCount {inputTokens, outputTokens}` with no estimation and no dollar cost; models are inferred heuristically. Attribution is per conversation (`composerId`), per message bubble (`bubbleId`), and per tool call (`toolFormerData`), with aggregates by project and day. A `cursorRules` bubble field is used only as a model-inference hint; there is no per-skill or per-rule attribution.
- **Confidence:** high — **Vote:** 3-0, 3-0, 2-1 (merged)
- **Evidence (External evidence):** `pyproject.toml` has `dependencies=[]` and `requires-python >=3.8`; `utils.py:32-34` builds the `mode=ro` URI; `messages.py:91` reads `bubble_data.get('tokenCount')`; grep of source for `estimat`, `/4`, `cost`, `price`, `usd` returns zero; grep for `.cursor/skills`, `per-skill`, `skill invocation` returns zero. **Important qualifier:** Cursor staff (Dean Rie, 2026-03-26, Cursor 2.6.21) state the stored `tokenCount` field is best-effort, unreliable, often 0, and not the source of truth — so this tool's "exact" counts are unreliable in practice.
- **Sources:** `github.com/mikhailsal/cursor-chronicle`, `forum.cursor.com/t/cursordiskkv-table-records-always-show-0-for-tokencount/155984`

### F-08 — `xiufengsun/TokenTracker`: real billed numbers, per request, via undocumented endpoints

- **Claim:** TokenTracker (`xiufengsun/TokenTracker`, JavaScript/Node, MIT, ~1,563 stars, created 2026-04-05, pushed 2026-09-10) supports Cursor by auto-detection: it reads `cursorAuth/accessToken` from `state.vscdb` and calls Cursor's undocumented dashboard endpoints (`cursor.com/api/dashboard/export-usage-events-csv?strategy=tokens`, `cursor.com/api/usage-summary`, and `api2.cursor.sh DashboardService/GetSandUsageStatus`) — not hooks, OTel, or a proxy. This yields Cursor-billed per-request token and cost rows (Date, Model, Input w/ and w/o cache write, Cache Read, Output, Total, Cost, Kind, Max Mode) attributed per model/provider, project/session, and git commit. Its "Skills" feature is a browse/sync leaderboard of 250+ public skills (Cursor is not even a sync target), not a cost-attribution boundary; no `conversationId`, skill, or rule fields are parsed.
- **Confidence:** high — **Vote:** 3-0, 2-1 (merged)
- **Evidence (External evidence):** README row: "Cursor | ✅ Auto | API + SQLite auth token" and "Cursor uses API instead of hooks"; `src/lib/cursor-config.js:46` runs `SELECT value FROM ItemTable WHERE key = 'cursorAuth/accessToken'`, lines 149-150 define the CSV and usage-summary URLs, lines 151-410 implement the protobuf RPC; the CSV parser (lines 439-530) resolves only the listed columns; grep for `hook|otel|proxy` returns 0 in `cursor-config.js` and `cursor-store.js`. Code comments note column order has changed multiple times.
- **Sources:** `github.com/xiufengsun/TokenTracker`

### F-09 — Cursor Cost Tracker extension: billed per usage event, status-bar only

- **Claim:** Cursor Cost Tracker (`ElisCarvalho.cursor-cost-tracker`, VS Marketplace / Open VSX, MIT, v0.2.6 published 2026-02-25, ~1,408 Open VSX downloads, 24 Marketplace installs; declared repo `github.com/ecarvalho/cursor-cost-tracker` returns 404) is a Cursor status-bar extension showing dollar spend today, this billing cycle, and for the last interaction, with a tooltip of the last 5 interactions (model, tokens, cost). It reads `cursorAuth/accessToken` from `state.vscdb` (`sqlite3` CLI, `sql.js` fallback), synthesizes a `WorkosCursorSessionToken` cookie, and paginates `cursor.com/api/dashboard/get-filtered-usage-events` (pageSize 500, 20 pages = 10k events per cycle) plus `get-team-spend`, summing `tokenUsage.*Tokens` and `totalCents`. Figures are Cursor-billed per usage event; the compiled VSIX contains zero references to skill, cursorrules, `.cursor/rules`, or `conversationId`.
- **Confidence:** high — **Vote:** 3-0 ×3 (merged)
- **Evidence (External evidence):** README verbatim: "Extension for Cursor IDE that shows real-time spend in $ on LLMs in the status bar" and "Detailed Tooltip: View the last 5 interactions with model, tokens, and cost". VSIX v0.2.6 inspected: `out/api.js` `getCursorDbPath` targets `globalStorage/state.vscdb`, `readTokenFromDb` queries `cursorAuth/accessToken`, `apiRequest` sets `Cookie: WorkosCursorSessionToken`, `fetchUsageData` breaks at `page>20`; `parseCostCents` uses `tokenUsage.totalCents`. README itself says "approximate cost per chat interaction".
- **Sources:** `marketplace.visualstudio.com/items?itemName=ElisCarvalho.cursor-cost-tracker`

### F-10 — Cross-cutting: two structural blockers, not nine independent gaps

- **Claim:** None of the nine Cursor-capable tools surveyed provides per-skill or per-rule cost attribution. The finest granularity anywhere is per request/usage event (billed, via dashboard-API scrapers) or per conversation/turn/bubble (estimated or unreliable stored counts, via SQLite readers and hook exporters). No tool parses `cursor.skill.activated`, `.cursor/skills`, or `.cursor/rules` as an attribution key, and no skills-benchmark or skill-evaluation tool with a Cursor adapter surfaced among verified claims.
- **Confidence:** high — **Vote:** synthesized from all 24 confirmed claims
- **Evidence (External evidence):** Each tool was verified against its primary repo, docs, or shipped binary; every verifier independently recorded the absence of skill/rule attribution. Two structural blockers recur:
  1. **Cursor's user-visible hook events** (`sessionStart`, `beforeSubmitPrompt`, `afterAgentResponse`, `pre`/`postToolUse`, `subagent*`) **carry no skill-activation signal**, so hook-based exporters cannot build a skill span.
  2. **Local `state.vscdb` per-bubble `tokenCount` is officially best-effort and often zero** on current builds, so SQLite readers fall back to `chars/4`.
  Billed data only reaches third parties through the Admin API / undocumented dashboard endpoints at per-request granularity — which is why the Enterprise-only OTel `skill.activated` + Admin API `conversationId` join (an already-known baseline) remains the only viable route.
- **Sources:** all nine primary tool sources listed above.

## Implications for cursor-profiler

### (a) Dash0 Agent Plugin is the closest prior art in Go — a reuse-vs-build candidate

`dash0hq/dash0-agent-plugin` (Apache-2.0, **Go + shell**, 9 Cursor hooks → OTLP) is the nearest thing to `cursor-profiler`'s intended shape that exists today (External evidence). AGENTS.md's **"prefer existing tools over building"** rule makes this a live reuse-vs-build decision rather than a footnote: it is the right language (Go, not Node like cursorscope, not Python like `opentelemetry-hooks`), the right transport (hooks → OTLP), and a permissive license.

The tradeoff, stated plainly:

- **Reuse/fork favors:** Go, Apache-2.0, hook wiring and OTLP emission already solved, active releases (v0.1.28, 2026-09-02).
- **Build favors:** it is **Dash0-backend-oriented**, covers only **9 of Cursor's 21** hook events (cursorscope covers 19), attributes tokens **per turn with no skill span**, produces **no span at all for Cursor subagent work**, does not state whether its token numbers are billed or estimated, and is early (0.1.x). The per-skill attribution — the actual product — would still have to be built on top.
- **Recommendation:** treat it as a **reference implementation and a source of hook/OTLP patterns to port**, not as a base to fork. It solves the part `cursor-profiler` was least worried about (transport) and none of the part that is hard (skill attribution). A fork would inherit a Dash0-shaped span model plus a 9-event ceiling for a component we can write once. This is a **Design decision** that should be recorded explicitly, because AGENTS.md's default is the other way.

### (b) `o11y-dev/opentelemetry-hooks` is the multi-agent hook exporter — and is excluded by AGENTS.md

It is the broadest hook→OTel exporter found (8 agents including Cursor IDE and Cursor CLI, MIT, active), and its session → generation → hook-event model with `conversation.id` keying is a useful design reference. But it is **Python 3.12+**, and AGENTS.md states: "Do not introduce Python for skill scripts. It adds runtime dependencies and breaks the self-contained principle." **Excluded as a dependency; retained as a design reference** (Design decision). Note also that it *only* passes through tokens the payload supplies and does no estimation — a stricter honesty posture than cursorscope's `chars/4`, worth mirroring in labeling.

### (c) Per-skill cost still requires a custom join — nobody packages it

The only path to per-skill cost in Cursor remains **joining a skill-activation signal with per-request token usage keyed on `conversationId`** (External evidence). No surveyed tool does this. The two ends of the join exist only in Enterprise/Teams surfaces:

- **Skill signal:** Cursor Enterprise OTel `cursor.skill.activated` (Enterprise plan only), or — for non-Enterprise — a **heuristic** skill signal of the cursorscope kind (file-path match on `/skills/` or `SKILL.md`), which is an inference, not a Cursor-reported activation (**Local hypothesis** that this heuristic is accurate enough to bill against).
- **Token signal:** Admin API `filtered-usage-events` (billed, per request, Teams/Enterprise admin) — the only trustworthy token source, since local `state.vscdb` counts are officially best-effort and often zero.

Consequence for the spec: a **non-Enterprise, non-admin `cursor-profiler` cannot produce billed per-skill dollars at all**. It can produce honestly-labeled *relative estimates* for paired A/B comparison (consistent with `skill-evaluation-research.md`'s conclusion that `chars/4` is a valid relative signal but not a billing number). Whether the product targets the Enterprise join, the estimate-only local path, or both is a **decision above this note's level**.

### (d) The Feb 2026 Cursor forum request is the only demand evidence

The single piece of public evidence that anyone besides this team wants per-skill analytics is a **Feb 2026 Cursor forum feature request for a skills adoption analytics endpoint** — with **no staff reply and no shipped feature** (External evidence): https://forum.cursor.com/t/feature-request-skills-adoption-analytics-endpoint/151688

Read two ways, and both should be on the table: it is a genuine unserved need (no competitor, open field), *or* the market is thin enough that nobody has built it. One unanswered forum post is weak demand evidence. It also means **Cursor itself may ship this**, which would obsolete a large part of the tool — the Customize-page skills leaderboard (invocation counts, no tokens) shows they are already partway there.

## Caveats and coverage gaps

- **Time-sensitivity:** Cursor ships analytics and skills changes frequently (skills leaderboard in 3.9/3.11; analytics filter update 2026-05-04), and the undocumented `cursor.com/api/dashboard/*` and `api2.cursor.sh` endpoints used by TokenTracker and Cursor Cost Tracker change without notice — those tools can break silently.
- **Negative findings:** several conclusions are absences (no skill attribution) supported by grep and doc inspection rather than a vendor statement.
- **Split votes:** three merged sub-claims carried 2-1 votes (Cursor dashboard granularity wording; `opentelemetry-hooks` `.cursor` mention; `cursor-chronicle` "no mention of rules"; TokenTracker skills leaderboard). Dissent was limited to wording precision, not substance.
- **Unreliable local counts:** per Cursor staff, `tokenCount` is best-effort and often 0, so "exact" figures from `cursor-chronicle` or older-build Token Use rows must not be treated as billed.
- **Not covered:** the confirmed set contains **no** findings on Langfuse, Helicone, Datadog, Grafana, Honeycomb, PostHog, Braintrust, SigNoz, or `LangGuard-AI/cursor-otel-hook` Cursor integrations (SigNoz Cursor docs and `cursor-otel-hook` were mentioned only in passing as parallel hook→OTel projects and were not verified), nor on any skills-benchmark tool with a Cursor adapter. Their absence here is **"not found among verified claims," not proof they lack per-skill cost** (Local hypothesis).
- **Fetched but not confirmed:** `nguyenvinhtieng/cursor-token-tracker`, `Dwtexe/cursor-stats`, `unblocked/cursor-harness`, and `Anyesh/skillprobe` were fetched as sources but produced no claims that survived verification budget; they are not in the table.
- **Immaturity:** `Pryowin/token_tracker` (2 commits, 0 stars); Cursor Cost Tracker (source repo 404s); `cursor-chronicle` (~20 stars, license inconsistency).
- **One refuted claim (0-3):** a characterization of the 2026-05-04 changelog as the authority for dashboard attribution granularity was dropped; its substance is covered by the verified analytics-docs finding (F-01).

## Open questions

1. Does Cursor's Enterprise OTel export emit `cursor.skill.activated` with a `conversationId` or request id that can be **deterministically** joined to Admin API `filtered-usage-events` `tokenUsage`, and has anyone published a working join (Grafana/Datadog/Honeycomb dashboard or query pack) for per-skill cost?
2. Do any major observability vendors (Langfuse, Helicone, Datadog, Grafana, Honeycomb, PostHog, Braintrust, SigNoz) ship a Cursor-specific integration that consumes the Enterprise OTel `skill.activated` event, or are their Cursor integrations limited to generic hook→OTel spans like Dash0's?
3. Does Cursor's local state (`state.vscdb`, `agentKv`, `store.db`, or agent-transcript JSONL) record which skill or rule was loaded into a given request's context — which would let SQLite readers add a per-skill dimension without hooks or the Admin API?
4. Has Cursor responded to the skills-adoption-analytics feature request, or announced per-skill/per-rule cost reporting (beyond invocation counts on the Customize page) in any changelog, roadmap, or forum post after 2026-05-04?

## Sources

Primary sources (repo code, official docs, shipped binary inspection), fetched 2026-09-10:

1. https://cursor.com/docs/account/teams/analytics
2. https://cursor.com/changelog/05-04-26
3. https://forum.cursor.com/t/feature-request-skills-adoption-analytics-endpoint/151688
4. https://github.com/dash0hq/dash0-agent-plugin
5. https://dash0.com/hub/integrations/int_cursor_plugin/overview
6. https://github.com/o11y-dev/opentelemetry-hooks
7. https://pypi.org/project/opentelemetry-hooks/
8. https://github.com/Pryowin/token_tracker
9. https://tokenuse.app/docs/development/tools/cursor/
10. https://tokenuse.app/docs/development/coach/
11. https://github.com/russmckendrick/tokenuse
12. https://github.com/getagentseal/codeburn
13. https://github.com/getagentseal/codeburn/blob/main/docs/providers/cursor.md
14. https://github.com/mikhailsal/cursor-chronicle
15. https://forum.cursor.com/t/cursordiskkv-table-records-always-show-0-for-tokencount/155984
16. https://github.com/xiufengsun/TokenTracker
17. https://marketplace.visualstudio.com/items?itemName=ElisCarvalho.cursor-cost-tracker

Fetched as candidates, no verified claims retained:

18. https://github.com/nguyenvinhtieng/cursor-token-tracker
19. https://github.com/Dwtexe/cursor-stats
20. https://github.com/unblocked/cursor-harness
21. https://github.com/Anyesh/skillprobe
22. https://forum.cursor.com/t/understanding-tokens-spent/155246
23. https://forum.cursor.com/t/excessive-token-usage-cursor-auto-loads-too-many-skills-from-claude-skills-at-conversation-start/160677
24. https://forum.cursor.com/t/token-usage-and-costs-report-per-request-and-per-session/138980
25. https://www.datachamp.fr/en/posts/cursor-tokens-audit/

Baseline references (already known, not re-verified this round):

26. https://github.com/last9/cursorscope
27. https://cursor.com/docs/enterprise/opentelemetry-export/wire
28. https://cursor.com/docs/account/teams/admin-api
29. https://code.claude.com/docs/en/skills
