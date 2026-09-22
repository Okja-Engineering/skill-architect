# cursorscope (last9) source analysis

Cloned from: https://github.com/last9/cursorscope.git (MIT, v0.3.7, Node ≥20)
Location: `tmp/research/cursorscope/`

## Architecture

- A Node/Express daemon listens on `PORT` (default 4327).
- Cursor hooks call `cursorscope-forward.sh`, which:
  1. Auto-starts the daemon via `ensure-cursorscope.sh` (pidfile, logfile, health check, version match, `npm install` on first run).
  2. Runs `cursor-hook-forwarder.js` to read stdin and POST JSON to `http://localhost:4327/cursor/hooks`.
- `install-global-hooks.sh` merges a `cursorscope-forward.sh` entry into `~/.cursor/hooks.json` for 19 events, with Python `json` merge and backup.
- The server (`src/server.js`) exposes `healthz`, `debug/otel-config`, `debug/emit-and-flush`, `debug/otlp-probe`, and `POST /cursor/hooks`.
- `src/telemetry.js` builds OTel spans/metrics/logs in memory and exports to OTLP endpoints.

## Hook event coverage

Cursor documents **21** lifecycle hook events. cursorscope's `.cursor/hooks.json` registers **19** of them, omitting `beforeTabFileRead` and `workspaceOpen`.

### Cursor's 21 hook events

- **Agent lifecycle:** `sessionStart`, `sessionEnd`
- **Prompt / tool execution:** `beforeSubmitPrompt`, `preToolUse`, `postToolUse`, `postToolUseFailure`
- **Shell / MCP:** `beforeShellExecution`, `afterShellExecution`, `beforeMCPExecution`, `afterMCPExecution`
- **File access / edits:** `beforeReadFile`, `afterFileEdit`, `beforeTabFileRead`, `afterTabFileEdit`
- **Subagents:** `subagentStart`, `subagentStop`
- **Agent internals:** `preCompact`, `stop`, `afterAgentResponse`, `afterAgentThought`
- **App lifecycle:** `workspaceOpen`

### cursorscope's 19

`sessionStart`, `sessionEnd`, `beforeSubmitPrompt`, `preToolUse`, `postToolUse`, `postToolUseFailure`, `beforeShellExecution`, `afterShellExecution`, `beforeMCPExecution`, `afterMCPExecution`, `beforeReadFile`, `afterFileEdit`, `afterTabFileEdit`, `subagentStart`, `subagentStop`, `afterAgentResponse`, `afterAgentThought`, `stop`, `preCompact`.

## Attribution (`src/attribution.js`)

- `classifyInvocation(hookName, hookData)` returns `{ category, name, detail, operationName }`:
  - `mcp` — from `beforeMCPExecution`/`afterMCPExecution` or `mcp:` prefix in `tool_name`.
  - `cli` — from `beforeShellExecution`/`afterShellExecution`.
  - `skill` — from a `file_path` matching `/skills/` or ending in `SKILL.md`; `name` is the skill directory.
  - `subagent` — from `subagent_type` or `Task` tool.
  - `tool` — fallback.
- `skillNameFromPath()` resolves a skill name from `.../skills/<name>/...` or `.../<dir>/SKILL.md`.
- `resolveMcpServer()` / `resolveMcpTool()` parse MCP server/tool identity from `mcp_server`, `url` hostname, `command`, `mcp_tool_name`, or `MCP: server/tool` patterns.

## Token accounting — the key finding

- `estimateTokens(value)` = `Math.ceil(text.length / 4)`. It is a **character-ratio estimate**, not a billed token count.
- `.env.example` line 32: `CURSOR_TRACK_ATTRIBUTED_TOKENS=true # Estimate context tokens for MCP, shell/CLI, skills, and tools (chars/4). Not Cursor LLM billing.`
- `extractReportedTokenUsage(resultJson)` looks for real token fields on MCP tool results: `input_tokens`/`prompt_tokens`/`input`, `output_tokens`/`completion_tokens`/`output`, `total_tokens`/`total`.
- `preCompact` payloads carry `context_tokens`, `context_window_size`, and `context_usage_percent`, which are Cursor-reported (the denominator is needed to make occupancy meaningful).
- Implication: hooks alone cannot produce billed token usage. The best honest signals are:
  1. MCP-reported usage (real, when present).
  2. `preCompact` context size (real, Cursor-reported).
  3. Chars/4 estimates (relative, must be labeled as estimated).

## Privacy (`src/privacy.js`)

- `redactForLogs(value, { includeToolDetails })`:
  - Replaces strings with `[redacted string len=N]` unless `includeToolDetails`.
  - Redacts emails, bearer tokens, `sk-...` / `key-...` keys via regex.
  - Masks keys matching `api[_-]?key|token|password|secret|authorization`.
- This runs before the event is written to the spool/export. For a JSONL file sink, the same rules must apply before disk.

## Metrics and span names (`src/telemetry.js` and `src/gen-ai-semconv.js`)

Metrics created:
- `cursor_hook_events_total`
- `cursor_session_total`
- `cursor_prompt_total`
- `cursor_tool_executions_total`
- `cursor_lines_of_code_total`
- `cursor_mcp_invocations_total`
- `cursor_attribution_invocations_total`
- `cursor_attributed_context_tokens_total` (description explicitly says "not Cursor LLM billing")
- `gen_ai.client.operation.duration`
- `gen_ai.client.token.usage`
- `cursor_api_metric_value` (Cursor Admin API polling)

Span/attribute conventions:
- GenAI semantic conventions: `gen_ai.provider.name`, `gen_ai.conversation.id`, `gen_ai.request.model`, `gen_ai.response.id`, `gen_ai.operation.name`, `gen_ai.agent.name`, `gen_ai.agent.id`, `gen_ai.tool.name`, `gen_ai.tool.type`, `gen_ai.tool.call.id`, `gen_ai.tool.call.arguments`, `gen_ai.tool.call.result`, `gen_ai.usage.input_tokens`, `gen_ai.usage.output_tokens`, `gen_ai.usage.reasoning_output_tokens`.

## State and TTL (`src/telemetry.js`)

- In-memory maps keyed by `toolUseId`, `generationId`/`conversationId`, `sessionId`.
- TTL sweep: tool/subagent 5 min, interaction/session 30 min.
- On `stop` or `sessionEnd`, the active interaction/session span is ended.

## CLI / install (`package.json` and `scripts/`)

- `bin/cursorscope.js` (entrypoint).
- `setup` command prompts for backend and writes `.env`.
- `install:global-hooks` / `uninstall:global-hooks`.
- `start`/`stop`/`restart` daemon.
- `ensure-cursorscope.sh` is ~100 lines of bash plus Node snippets for version parsing.
- `install-global-hooks.sh` uses an embedded `python3` heredoc for JSON merge.

## Dependencies

- Node ≥20.
- 10 `@opentelemetry/*` packages, `express`, `dotenv`.
- ~3,400 lines across `src/`.

## Implications for cursor-profiler

1. **Do not replicate the daemon.** The Go port should default to a stdin → JSONL spool design to avoid the state/TTL/auto-start complexity.
2. **Do not replicate Node/.env/Express.** A single Go binary with flags/env and a file sink is the smallest honest replacement.
3. **Preserve attribution classification logic** in Go — it is the highest-value feature.
4. **Preserve privacy redaction rules** before any disk write.
5. **Label estimates honestly.** Chars/4 must not appear as a `present` `TokenCounts`; use a separate `estimated_context_tokens` field or `source: "hooks_estimated"`.
6. **Use GenAI semconv names** for any OTLP export slice, so we are not inventing a parallel taxonomy.
7. **The 21 Cursor hook events are the contract; cursorscope covers 19 of them.** Write a tolerant parser that accepts `additionalProperties` because Cursor may add fields.
8. **The install script must be Go/bash, no Python.** `AGENTS.md` forbids Python in skill scripts; cursorscope's `python3` JSON merge can be replaced by `jq` or a small Go helper.
