# Mandate draft — Cursor hook telemetry in Go (working name: `skill-scope`)

**Status:** draft for user grilling · **Date:** 2026-09-10 · **Drafter:** intake-drafter
**Raw ask:** "maybe we can create a road map for a https://github.com/last9/cursorscope but in golan bash semgrep ect targeting cursor. extract requirements and slices from this repo and create a new roadmap that considers code from cursor scope and execute more research on current state of profiling skill performance and token usage efficiency"

---

## 1. Goal

Give `skill-architect`'s profiler a **real Cursor runtime signal source** by porting the load-bearing parts of `last9/cursorscope` — Cursor lifecycle-hook ingestion, invocation attribution, and privacy redaction — into Go and bash, so that the Cursor adapter stops being an OTel-export-file reader that nobody can actually feed, and starts producing `Profile` documents from sessions the user actually runs. The roadmap should extract cursorscope's requirements as requirements (not copy its architecture), and it should be sequenced so the first slice answers the one question everything else rests on: *can a Go binary receive Cursor hook JSON and turn it into a `Profile` with metrics that are honestly `present`?*

### The central ambiguity, stated loudly

Three readings of the ask, and they build different things:

| Reading | What gets built | Who it serves |
|---|---|---|
| **(a) Standalone port** | A general Cursor→OTLP observability daemon in Go. Traces, metrics, logs, backends, dashboards. A competitor/alternative to cursorscope. | Teams who want to observe Cursor usage. Not `skill-architect`. |
| **(b) Ingestion source** | A hook-sourced telemetry path feeding `CursorAdapter`, so `skill_activation`, `tool_calls`, `timing`, and attribution stop being `unknown`. No daemon, no backend. | `skill-architect`'s F04 comparison engine. |
| **(c) Both, (b) motivating (a)** | (b) first as the core; OTLP export and a daemon added later as optional surfaces on the same spool. | Both, sequenced. |

**Recommended: (c), sequenced (b)-first, with the (a) half explicitly conditional and gated on demand.**

Why: (b) is the only part that pays `skill-architect` back, and it is achievable without a daemon, a port, or an OTel SDK dependency. (a) is a genuinely different product with a different customer and would contradict `.out-of-scope.md` if adopted as the primary goal. But (a) is cheap *after* (b) — once normalized hook events are on disk, an OTLP exporter is one more consumer of the same spool. So: build (b) in slices 1–6, hold (a) in slices 7–8 behind an explicit go/no-go.

### The finding that reframes the ask

The task brief describes reading (b) as getting "**real** hooks-sourced tokens". **Reading cursorscope's source says that is not achievable.** Repository fact, from the clone:

- `src/attribution.js:estimateTokens()` is `Math.ceil(text.length / 4)`. Every token number cursorscope emits from a hook is a **chars/4 estimate**, tagged `source: "estimated"`.
- The only non-estimated numbers are: `extractReportedTokenUsage()` (tokens an *MCP tool* self-reports in `result_json`), `handlePreCompact`'s `context_tokens` (Cursor-reported context size at compaction), and the **Admin API** poller (`/teams/{id}/daily-usage-data`) which is team-level daily aggregate — not attributable to a session or a skill.
- cursorscope's own `.env.example` says it plainly: `CURSOR_TRACK_ATTRIBUTED_TOKENS=true # Estimate context tokens ... (chars/4). Not Cursor LLM billing.`

So: **hooks do not carry billed token counts.** This collides head-on with `MetricResult`'s honesty contract (`present` means "value captured from telemetry"). Resolving that collision is Fork (vi) and User Question 4, and it is the single highest-risk item in this mandate.

**What hooks *do* deliver that `skill-architect` has none of today:** skill activation and attribution. `src/attribution.js:classifyInvocation()` already returns `category: "skill"` with a resolved skill name from `skills/<name>/` and `SKILL.md` paths. Every adapter in `profiler/` currently returns `skill_activation: unknown` and `attribution: unknown` — for all four harnesses. That is the metric F04 most wants and the one nothing in the repo can produce. **The real prize here is attribution, not tokens.** The roadmap below is sequenced accordingly.

---

## 2. Assumptions

Correct any of these and the draft changes shape. This is the section to attack.

**About intent**

- **A01.** The user wants working telemetry for `skill-architect`, not a published OSS product competing with cursorscope. (If wrong → reading (a), separate repo, different DoD.)
- **A02.** "in golan bash semgrep ect" is a *language/tooling constraint* restating `AGENTS.md` (Go for logic, bash for scripts, no Python), not a demand that semgrep be load-bearing. See §5 — semgrep is a marginal fit and is itself a Python package.
- **A03.** "targeting cursor" means Cursor only for this epic. Claude Code / Codex hooks are a later, cheap follow-on (the design-space note says 3/4 harnesses fire JSON-on-stdin hooks) but are not in scope now.
- **A04.** "extract requirements and slices from this repo" means extract from **cursorscope** and re-slice them for **skill-architect**, producing a roadmap document — not "start building this week".
- **A05.** The deliverable of *this* mandate is an approved roadmap/spec, not code. Code follows a separate dispatch.

**About the technical ground**

- **A06.** Cursor's hook contract is as `.cursor/hooks.json` in the clone describes: 19 named events, JSON on stdin, non-fatal on failure. **Unverified against Cursor's current docs or a live install.** Slice 1 exists to verify it.
- **A07.** The `cursor.token.usage`, `cursor.skill.activated`, and `cursor.hook.execution_complete` names hardcoded in `profiler/cursor.go` are **unverified**. cursorscope — a real, shipping Cursor integration — never references them; it synthesizes its own spans from hooks instead. I am assuming these names may be invented or enterprise-only, and that the existing Cursor adapter may be parsing a format nothing emits. Slice 6 must confirm or retract them.
- **A08.** `profiler/`'s "OTel export file" shape (`{"metrics":[...],"logs":[...]}` in `cursor.go:cursorOtelExport`) is a **bespoke format invented by this repo**, not OTLP/JSON. Nothing in the wild emits it. It stays as a test/import path but is not a real ingestion route.
- **A09.** The user runs Cursor locally and is willing to install a hook into `~/.cursor/hooks.json` on their own machine. Without this, no real fixtures exist and the epic is unverifiable.
- **A10.** The user is **not** on Cursor Business/Enterprise, so the Admin API poller and enterprise OTel export are unavailable. Assumed `drop`; see User Question 2.
- **A11.** `profiler/go.mod` has **zero external dependencies** today. I treat that as a deliberate property worth preserving, not an accident.
- **A12.** A JSONL spool on local disk is an acceptable integration surface. The profiler already reads files for every adapter.

**About process and state**

- **A13.** This is a **new epic** running alongside/after F04, not a replacement. F01–F02 are done; F04 slices 1–3 are committed (compare, experiment plan, experiment run).
- **A14.** `tmp/teams/architect/F04.status.md` still says `Stage: parked` while three F04 slices are committed — the old control plane is stale. I am trusting git, not the status files. The new epic lives under `.scuba/`.
- **A15.** `.scuba/` is gitignored (`.scuba/.gitignore` = `*`), so this mandate is local-only and will not appear on a branch. Durable across a kill, invisible to a reviewer.
- **A16.** The concurrent research ledger at `.scuba/teams/research-profiling/ledger.md` will answer *what* to measure for token efficiency and *what claims are defensible*. This roadmap consumes it; it does not duplicate it. Slices S4 and S6 are gated on it.
- **A17.** Porting cursorscope code (MIT) requires attribution. I assume we prefer **reimplementing from extracted requirements**, using its tests as a behavioral oracle, over line-by-line translation.
- **A18.** CI currently pins `go-version: "1.23"` while `profiler/go.mod` declares `go 1.27.1`. Likely a latent CI break unrelated to this epic; flagged, not owned here.

---

## 3. Requirements extracted from cursorscope (R-CS)

Tagged **keep** (carry over as-is in spirit), **adapt** (carry the requirement, change the mechanism), **drop** (deliberately out).

### Ingestion and hook plumbing

| ID | Requirement | Tag | Rationale |
|---|---|---|---|
| **R-CS-01** | Cover Cursor's lifecycle hook set: `sessionStart`, `sessionEnd`, `beforeSubmitPrompt`, `preToolUse`, `postToolUse`, `postToolUseFailure`, `beforeShellExecution`, `afterShellExecution`, `beforeMCPExecution`, `afterMCPExecution`, `beforeReadFile`, `afterFileEdit`, `afterTabFileEdit`, `subagentStart`, `subagentStop`, `afterAgentResponse`, `afterAgentThought`, `stop`, `preCompact` | adapt | We need a subset for `Profile`: session/prompt boundaries (timing), tool pre/post (tool_calls), `beforeReadFile`+shell (skill attribution), `preCompact` (context tokens). Register the rest as pass-through so fixtures capture everything once. |
| **R-CS-02** | A hook entrypoint reads JSON on stdin, enriches with `conversation_id`, `generation_id`, `session_id`, `model`, `cursor_version`, `workspace_roots`, user, repo, timestamp | keep | This is the load-bearing contract. Go binary instead of Node forwarder. |
| **R-CS-03** | Hook failure must never break the user's Cursor session (`\|\| true`, non-zero exit tolerated, timeouts bounded) | keep | Non-negotiable. A profiler that wedges the editor is worse than no profiler. |
| **R-CS-04** | Forward to a local HTTP ingestor at `POST /cursor/hooks`, 202 on accept | adapt | Fork (i). Default is append-to-JSONL-spool; HTTP becomes an optional second sink. |
| **R-CS-05** | Correlate events into a hierarchy by `conversation_id` (session) → `generation_id` (prompt/turn) → `tool_use_id` (tool call), with subagents branching | keep | The correlation *keys* are what make a `Profile` coherent. We keep the keys and the nesting semantics; we do not need OTel spans to express them. |
| **R-CS-06** | TTL sweep for un-closed state: 5 min for tool/subagent, 30 min for interaction/session; close and mark stale rather than leak | adapt | cursorscope shipped this as a memory-leak fix in 0.3.7. In a stateless spool design this becomes a *session-close heuristic in the reader* — same requirement, no long-lived process, no leak class at all. Strong argument for Fork (i)'s default. |
| **R-CS-07** | Config via env vars / `.env`, with sane local defaults | adapt | Env vars yes; `.env` file loading is a Node-ism. Flags + env, no dotenv. |

### Attribution and metrics (the valuable part)

| ID | Requirement | Tag | Rationale |
|---|---|---|---|
| **R-CS-08** | Classify every invocation into `mcp` / `cli` / `skill` / `subagent` / `tool` with a resolved name and detail (`classifyInvocation`) | **keep — highest value** | This is what produces skill activation and attribution, which no `skill-architect` adapter can produce today. |
| **R-CS-09** | Resolve a skill name from a path: `…/skills/<name>/…` or `…/<name>/SKILL.md` (`skillNameFromPath`) | **keep** | Directly maps to `ActivationEntry.SkillName` and `Attribution.SkillName`. Note it detects *skill files being read*, which is a proxy for activation, not an activation event. Must be labeled as inference. |
| **R-CS-10** | Resolve MCP server/tool identity from `mcp_server`, URL host, command, or `MCP: server/tool` name patterns | adapt | Useful for attribution completeness; secondary to skill attribution. |
| **R-CS-11** | Estimate context tokens as `chars/4` from tool input/output, shell command/output, MCP payloads, with a `source` label (`estimated` / `mcp_reported`) | **adapt — with an honesty gate** | The estimate is a legitimate *relative* signal for A/B comparison; it is not a token count. Must not serialize as a bare `present` `TokenCounts`. See R-SA-03 and Fork (vi). |
| **R-CS-12** | Extract genuinely-reported token usage from MCP results (`input_tokens`/`prompt_tokens`/`output_tokens`/`completion_tokens`/`total_tokens`) | keep | Real numbers when present. Small surface, high honesty value. |
| **R-CS-13** | Capture `context_tokens` and `context_usage_percent` from `preCompact` | **keep** | Cursor-reported, not estimated. Probably the most defensible token-adjacent number available from hooks. |
| **R-CS-14** | Capture line-edit stats from `afterFileEdit` payloads (added/removed, extension, basename) | adapt | Genuine effort signal for a with/without-skill comparison, but `Profile` has no field for it. Needs a schema addition or drop. Defer to a `v2` profile discussion. |
| **R-CS-15** | Capture shell `exit_code` for failure-type discrimination | keep | Cheap, feeds `ToolCallEntry.Success` honestly instead of guessing. |
| **R-CS-16** | Named metrics: `cursor_hook_events_total`, `cursor_session_total`, `cursor_prompt_total`, `cursor_tool_executions_total`, `cursor_lines_of_code_total`, `cursor_mcp_invocations_total`, `cursor_attribution_invocations_total`, `cursor_attributed_context_tokens_total`, `cursor_api_metric_value` | adapt | Only meaningful if we export OTLP (S7). If we do, reuse these names verbatim rather than inventing parallel ones — interop beats novelty. |
| **R-CS-17** | GenAI semconv attributes: `gen_ai.provider.name`, `gen_ai.conversation.id`, `gen_ai.request.model`, `gen_ai.response.id`, `gen_ai.operation.name`, `gen_ai.agent.name/id`, `gen_ai.tool.name/type/call.id/arguments/result`, `gen_ai.usage.input_tokens`/`output_tokens`/`reasoning_output_tokens`; metrics `gen_ai.client.operation.duration`, `gen_ai.client.token.usage` | adapt (conditional on S7) | Correct standard to follow *if* we export. Caution: putting chars/4 estimates into `gen_ai.usage.*_tokens` is exactly the dishonesty this repo's `MetricResult` exists to prevent. cursorscope does it; we should not, or must qualify it. |

### Privacy, install, ops

| ID | Requirement | Tag | Rationale |
|---|---|---|---|
| **R-CS-18** | Redact user prompts by default (record length only); opt-in via flag to include text | **keep** | Non-negotiable. Prompts hit disk in our design, which is *more* sensitive than cursorscope's in-memory→OTLP path. |
| **R-CS-19** | Scrub bearer tokens, `sk-`/`key-` API keys, and emails from all payloads regardless of settings; mask sensitive keys (`api_key`, `token`, `password`, `secret`, `authorization`) | **keep** | Same. Must run *before* the spool write, not before export. |
| **R-CS-20** | Optional email masking; bounded tool-payload size (`CURSOR_TOOL_PAYLOAD_MAX_LEN`, 8192) | keep | Cheap, prevents unbounded spool growth. |
| **R-CS-21** | Merge into `~/.cursor/hooks.json` **without clobbering** existing user hooks; timestamped backup first; clean up own legacy entries; uninstall removes only our entries | **keep — reimplement** | The requirement is right and carefully done in cursorscope. But its implementation shells out to `python3` for the JSON merge, which `AGENTS.md` forbids. Reimplement in Go (or `jq`, already present at `/usr/bin/jq`). |
| **R-CS-22** | Auto-start the ingestor from the hook if not healthy; version-match a running instance and restart on mismatch; pidfile + logfile + `nohup` | **drop under default fork** | This entire class of complexity (~104 lines of bash in `ensure-cursorscope.sh`) exists only because there is a daemon. Fork (i)'s default deletes the requirement instead of porting it. Returns only if S8 is approved. |
| **R-CS-23** | Debug endpoints: `GET /healthz`, `GET /debug/otel-config`, `POST /debug/emit-and-flush`, `GET /debug/otlp-probe` | adapt | Replaced under the default fork by `skill-scope doctor` — a one-shot command that reports hook install state, spool path/size/recency, and last parse errors. Same diagnostic need, no server. |
| **R-CS-24** | OTLP endpoint resolution: per-signal env override → `OTEL_EXPORTER_OTLP_ENDPOINT` + `/v1/{signal}` → `localhost:4318` default; `OTEL_EXPORTER_OTLP_HEADERS` as comma-separated `k=v` | keep (conditional on S7) | Well-specified, tested in cursorscope, and the conventional behavior. Copy the semantics exactly. |
| **R-CS-25** | Setup CLI: `setup` / `start` / `stop` / `status` / `hooks install` / `hooks uninstall`, with `--dry-run`, `--yes`, `--home`, non-interactive flags, and `.env` patching that preserves unrelated user config | adapt | Keep `hooks install/uninstall`, `status`/`doctor`, `--dry-run`, `--yes`. Drop `start`/`stop` (no daemon) and the interactive backend wizard. |
| **R-CS-26** | Cursor Admin API polling for team daily usage → gauges | **drop** | Team-level daily aggregates cannot be attributed to a session or a skill snapshot, so they cannot serve a paired comparison. Also requires Business tier (A10). Revisit only if User Question 2 says otherwise. |
| **R-CS-27** | Bundled OTel Collector configs + `docker-compose.yml` for local fan-out | **drop** | Infrastructure for an observability product. Not our problem. |
| **R-CS-28** | Node ≥20 runtime, `npm install` on first run, no build step | **drop → replaced** | Replacing this is the actual justification for the Go port: a single static binary with no runtime, no `node_modules`, no first-run install. State this as the port's *reason* in the roadmap. |
| **R-CS-29** | Unit tests per module + a fake OTLP collector for end-to-end assertions (`test/fake-collector.mjs`) | **keep** | The fake-collector pattern is directly reusable as an `httptest.Server` in Go for S7. cursorscope's `attribution.test.js` and `gen-ai-semconv.test.js` cases are a ready-made behavioral oracle for our port of R-CS-08/09/10. |
| **R-CS-30** | Non-functional: bounded memory, no unbounded state growth, hook latency small enough to be invisible in the editor | keep | With a spool design, memory is trivially bounded; the live constraint becomes hook process startup + write latency. Needs a measured budget. |

---

## 4. Requirements from `skill-architect` itself (R-SA)

These come from this repo's existing contracts and conventions, and they constrain the port more than cursorscope does.

| ID | Requirement | Source |
|---|---|---|
| **R-SA-01** | Output must be a valid `Profile` at schema `skill-architect/profile/v1`, and must round-trip `Marshal → Unmarshal → Marshal` identically. | `docs/profiler-spec.md`, `profiler/types.go` |
| **R-SA-02** | Every metric is a `MetricResult`: `present` (value + `Source`, no `Reason`), `unknown` (no value, `Reason` required), `error` (no value, `Reason` required). No metric may be invented or silently zero-filled. | `docs/profiler-spec.md` serialization rules |
| **R-SA-03** | **Estimated values are not `present` values.** A chars/4 token estimate must not serialize as a `TokenCounts` with `state: "present"` without an explicit estimation marker in the schema. This requires either a new `Source` constant (e.g. `hooks_estimated`), a `MetricResult.Estimated bool`, or reporting tokens as `unknown` and putting the estimate in a separate field. **Unresolved — Fork (vi).** | `PRINCIPLES.md` (honest degradation), R-CS-11 |
| **R-SA-04** | The profile is pinned to a `snapshot_hash` for the skill under test, and F04 revalidates against it before comparing. | `docs/profiler-spec.md`, `profiler/compare.go` |
| **R-SA-05** | The adapter declares a `CapabilityReport` before capture. `SourceHooks` is already defined in `types.go` and currently **used by no adapter** — this epic is its first consumer. | `profiler/types.go:SourceHooks` |
| **R-SA-06** | Must satisfy the F04 comparison contract: `CompareProfiles` compares only metrics `present` on *both* sides, emits deltas, and notes harness / snapshot / skill_dir mismatches. Hook-sourced profiles must be comparable against each other. | `profiler/compare.go` |
| **R-SA-07** | `skill_activation` and `attribution` are `unknown` in all four current adapters. Making at least one of them `present` for Cursor is the epic's primary success condition. | `profiler/{cursor,claude_code,codex,devin}.go` |
| **R-SA-08** | `profiler capture --session <id>` takes a harness-specific session ID. Cursor's `conversation_id` / `generation_id` / `session_id` triple must map onto it unambiguously and be documented. | `profiler/cmd/main.go`, cursorscope forwarder payload |
| **R-SA-09** | **Bash** for scripts, **Go** for complex logic, **no Python**. This directly blocks porting `install-global-hooks.sh` as-is (it embeds a `python3` heredoc). | `AGENTS.md` |
| **R-SA-10** | Prefer existing tools over building. `jq` is available; `skill-validator` and `skillscore` cover static analysis. Do not hand-roll what they do. | `AGENTS.md` |
| **R-SA-11** | Docs must carry evidence labels: Specification / External evidence / Repository fact / Design decision / Local hypothesis. Every claim about Cursor's hook payloads starts as **Local hypothesis** until a real payload is captured. | `AGENTS.md` |
| **R-SA-12** | `profiler/` has zero external Go dependencies. Any dependency added (notably the OTel SDK) is a deliberate, argued decision — Fork (v). | `profiler/go.mod` |
| **R-SA-13** | Degrade gracefully with no Cursor present: `probe` reports `none`, `capture` returns an all-`unknown` profile with reasons. Never fail hard. | `docs/profiler-spec.md` acceptance criteria 3 & 5 |
| **R-SA-14** | **No invented telemetry names.** Includes auditing the *existing* `cursor.token.usage` / `cursor.skill.activated` / `cursor.hook.execution_complete` strings already in `cursor.go` (A07). If unverifiable, they get relabeled or removed, not left to imply capability. | `AGENTS.md` evidence labels, `PRINCIPLES.md` |
| **R-SA-15** | Must slot into `experiment plan` / `experiment run`: each `ExperimentStep` carries `ExportFile` / `OtelFile` and a `ProfilePath`. Hook-sourced capture needs an equivalent input path on `Condition`. | `profiler/experiment.go` |
| **R-SA-16** | No regressions: 131 bash tests + 34 Go tests must stay green. | `CHANGELOG.md` 0.4.0, `tests/`, `profiler/profiler_test.go` |
| **R-SA-17** | If a background hook-capture surface is adopted, `.out-of-scope.md` and `README.md` must be amended to say so — the current text says the plugin does not run live comparisons and bounds itself to reporting. | `.out-of-scope.md`, `README.md` |

---

## 5. Where semgrep fits — and honestly, it mostly does not

**Assessment: marginal fit for this epic. Recommend it be a small optional guardrail slice (S9) or dropped entirely.**

Where it genuinely fits:

- **Repo guardrail rules in CI.** A tiny ruleset that fails the build on this epic's specific footguns: a `python3` invocation in a shell script (violates `AGENTS.md`), a `MetricResult` constructed as `present` from a function whose name contains `estimate` (violates R-SA-03), an unredacted prompt/payload written to disk (violates R-CS-18/19), or a hardcoded `cursor.*` telemetry string outside the one file allowed to declare them (R-SA-14). These are real pattern-finding tasks and semgrep is the right shape of tool for them.
- **Auditing *target* skills' bundled scripts** for risky patterns as a future `skill-audit` dimension. Real, but a different feature — not this epic.

Where it does not fit:

- **Not in the ingest path.** Parsing hook JSON, correlating events, building profiles — that is Go with `encoding/json`. Semgrep has no role.
- **Not for absence checks.** `AGENTS.md` already rules on this: "Semgrep is for pattern finding, not absence detection." Use `grep`.
- **Not for telemetry validation.** Whether `cursor.token.usage` exists is answered by a captured payload, not a static rule.

Two honest cautions:

1. **semgrep is not installed** in this environment (`command -v semgrep` → not found). Adopting it adds a CI install step.
2. **semgrep is distributed as a Python package.** `AGENTS.md` bans Python *for skill scripts* to preserve self-containment. Introducing a Python-based CI dependency to enforce a "no Python" rule is at minimum ironic and worth an explicit decision rather than a silent one. `grep`/`go vet` plus a focused Go test may cover the same four rules with zero new dependencies.

**Working default:** carry the four guardrail rules as *requirements*, implement them with `grep` + `go vet` + tests in S9, and only reach for semgrep if the rules prove too structural for grep. If the user specifically wants semgrep in the stack, S9 becomes the semgrep slice.

---

## 6. Proposed epic and slices

**Epic:** `E-CS` — Cursor hook telemetry for the profiler.
Each slice is independently shippable and independently verifiable. Sequencing is deliberate: **S1 fails fast if the whole premise is wrong.**

| # | Slice | Definition of done (one line) | Test approach | Depends on ledger? |
|---|---|---|---|---|
| **S1** | **Hook JSON → `Profile` (the proof)** | A Go binary reads one Cursor hook payload on stdin, appends a normalized event to a JSONL spool, and `profiler capture --harness cursor --hook-spool <f>` emits a schema-valid `Profile` where `tool_calls` and `timing` are `present` with `Source: "hooks"`. | Golden fixture: hand-authored spool JSONL → expected `Profile` JSON, asserted in `profiler_test.go`; round-trip test per R-SA-01. | no |
| **S2** | **Real install + fixture harvest** | `hooks install` merges our entry into a temp `~/.cursor/hooks.json` without dropping pre-existing hooks, writes a timestamped backup, and `hooks uninstall` restores exactly; one real Cursor session on the user's machine yields redacted raw payloads committed as fixtures. | Bash test against a fake `$HOME` seeded with a hooks.json containing a foreign hook; assert foreign hook survives install *and* uninstall. Manual capture step for the real session. | no |
| **S3** | **Skill activation + attribution** | A `Profile` from a real session reports `skill_activation: present` and `attribution: present` naming the skill that was exercised — the first non-`unknown` activation any adapter has ever produced. | Table-driven Go tests ported from cursorscope's `attribution.test.js` / `gen-ai-semconv.test.js` cases, plus an end-to-end assertion on the S2 fixture. | no |
| **S4** | **Token honesty** | The profile represents chars/4 estimates, MCP-reported usage, and `preCompact` `context_tokens` in a way a reader cannot mistake for billed tokens; `docs/profiler-spec.md` documents the representation. | Assertion that an estimate never serializes as bare `state:"present"` `TokenCounts`; fixture tests for each of the three token sources. | **yes** — the ledger decides what token accounting supports a defensible efficiency claim. |
| **S5** | **Privacy before disk** | Given a payload containing an email, a bearer token, an `sk-` key, and a long prompt, the spool file on disk contains none of them and records prompt length only. | Go table tests ported from `privacy.test.js`; plus a spool-file grep assertion (absence check → `grep`, per `AGENTS.md`). | no |
| **S6** | **Cursor adapter integration + F04 end-to-end** | `CursorAdapter.Probe/Capture` treat hooks as a first-class source alongside the export-file path; `profiler compare` on two hook-sourced profiles yields a report with `tool_calls` and `skill_activation` comparable; the unverified `cursor.*` OTel names are confirmed or retracted (R-SA-14). | Existing Cursor adapter tests extended; a new compare test on two hook-sourced fixtures; explicit test that a retracted name yields `unknown` with a reason rather than a silent zero. | **yes** — the ledger decides which metrics carry the efficiency claim in the comparison report. |
| **S7** | **OTLP export (conditional — Fork ii/v)** | `skill-scope export --spool <f>` posts OTLP/HTTP JSON traces+metrics to a configured endpoint using cursorscope's metric names and GenAI semconv attributes, with estimates clearly qualified. | Go `httptest` fake collector modeled on `test/fake-collector.mjs`; assert span nesting by conversation→generation→tool and correct attribute keys. | no |
| **S8** | **Daemon + HTTP ingest (conditional — Fork i)** | A long-running `skill-scope serve` accepts `POST /cursor/hooks`, exposes `/healthz` + config/probe diagnostics, auto-starts from the hook, and bounds state with the TTL sweep. | Integration test: start server, POST fixtures, assert spool/export equivalence with the stdin path; TTL test with backdated entries (cursorscope's `_testHooks` pattern). | no |
| **S9** | **Docs + guardrails** | `docs/cursor-hooks-spec.md` documents the hook payload contract with evidence labels; README's adapter table updates Cursor to a real status; CI runs the new tests and the four guardrail rules. | CI green; docs reviewed for evidence labels; guardrail rules demonstrated failing on a seeded violation. | partial — cites the ledger. |

**Minimum viable epic:** S1–S3 + S5 + S9. That delivers the prize (attribution) with privacy and docs, and skips tokens, OTLP, and the daemon entirely. If the user wants the smallest thing that pays back, propose that.

---

## 7. Forks (each with a chosen default)

**(i) Daemon + HTTP ingestor vs. stdin-only binary + JSONL spool**
→ **Default: stdin-only binary appending to a local JSONL spool. No daemon.**
Rationale: kills an entire complexity class — no port, no pidfile, no auto-start bash, no version-mismatch restart, no long-lived maps to leak (cursorscope shipped a memory-leak fix in 0.3.7 for exactly this). It matches how every existing adapter works: read a file. A daemon can be added later (S8) consuming the same spool, so this is reversible. Cost: no real-time steering, and correlation/session-close becomes reader-side logic instead of in-process state.

**(ii) OTLP export as a first-class output vs. local spool first, OTLP later**
→ **Default: local spool first; OTLP deferred to S7 and gated on demand.**
Rationale: `skill-architect`'s consumer is a `Profile` JSON file, not an observability backend. OTLP serves reading (a), which is deferred. Building it first would mean designing for a customer we do not have yet. Cost: if the user actually wants dashboards today, this sequencing is wrong — see User Question 3.

**(iii) Code location: extend `profiler/` vs. new top-level `cursorscope/` Go module vs. separate repo**
→ **Default: extend the existing `profiler/` module with a new `profiler/cmd/cursor-hook/` binary; hook/attribution logic as new files in the `profiler` package.**
Rationale: the `Profile` types already live there; a second module would either duplicate them or force a public API contract between two modules in one repo. A separate repo splits CI and orphans the epic from the thing it serves. Cost: `profiler/` grows a second binary and a bigger surface. Revisit only if Forks (i) and (ii) *both* go the daemon+OTLP way — at that point it genuinely is a separate product and deserves its own repo.

**(iv) Naming**
→ **Default: epic and binary named `skill-scope`; the stdin entrypoint is `cursor-hook`. Avoid the name "cursorscope".**
Rationale: `cursorscope` is Last9's published npm package and brand (`@last9/cursorscope`). A Go program with the same name invites user confusion and an avoidable attribution/trademark question. Alternatives considered: `cursorscope-go` (still leans on their brand), `scope` (too generic), folding it in as `profiler hook` (accurate, least discoverable). Related: if we port code rather than reimplement, MIT attribution is required — A17 assumes reimplementation from requirements, using their tests as an oracle.

**(v) Go OTel SDK dependency policy**
→ **Default: keep `profiler/` at zero external dependencies. If S7 lands, hand-roll OTLP/HTTP JSON rather than importing the OTel SDK.**
Rationale: OTLP/HTTP+JSON is a POST of a well-specified JSON body — cursorscope's own 71-line `fake-collector.mjs` shows how small the surface is. The alternative pulls ~10 `@opentelemetry`-equivalent Go modules into a module that currently has none, for one exporter. Cost: we own the wire format and must track spec drift; the SDK gives batching, retry, and compression for free. If the user wants production-grade export to a real backend, this default flips.

**(vi) How estimated tokens are represented — the honesty fork** *(added; the brief did not anticipate it, and it is the riskiest decision here)*
→ **Default: tokens stay `unknown` with reason "Cursor hooks do not expose billed token counts", and the chars/4 figure is carried in a separate, clearly-named field (e.g. `attributed_context_tokens`) that F04 may compare but never labels as token usage.**
Options: (1) as defaulted; (2) new `Source` constant `hooks_estimated` with `state: "present"`; (3) `MetricResult.Estimated bool` flag; (4) do what cursorscope does and populate `gen_ai.usage.*_tokens` with estimates. Rationale for the default: `PRINCIPLES.md` and the `MetricResult` contract exist precisely to stop a plausible number from passing as a measured one; option (4) is the failure mode this repo was built to prevent. Cost: the profile's token story for Cursor becomes "we cannot measure this", which may make the user's original token-efficiency goal unreachable via hooks. That is honest, and it should be surfaced now rather than discovered in S4.

---

## 8. Scope and non-goals

**In scope**

- Cursor lifecycle-hook ingestion in Go, invoked via bash from `~/.cursor/hooks.json`.
- Normalization, correlation, privacy redaction, and attribution of hook events.
- Producing `skill-architect/profile/v1` documents with `Source: "hooks"`.
- Making `skill_activation` and `attribution` `present` for Cursor for the first time.
- Hook install/uninstall with backup and non-clobbering merge.
- Documentation with evidence labels; CI coverage; guardrail rules.
- Conditional, gated: OTLP export (S7) and a daemon (S8).

**Explicit non-goals**

- **Not** a general-purpose Cursor observability product, dashboard, or Last9 competitor.
- **Not** a re-implementation of cursorscope feature-for-feature; requirements are extracted, architecture is not inherited.
- **No** Cursor Admin API / team-usage polling (R-CS-26) unless User Question 2 flips it.
- **No** bundled OTel Collector configs or `docker-compose` (R-CS-27).
- **No** Claude Code / Codex / Devin hook adapters in this epic (A03) — Cursor only.
- **No** Python anywhere, including the hooks.json merge (R-SA-09) and, by default, including semgrep (§5).
- **No** changes to the `Profile` schema version — additive fields only; a `v2` bump is a separate decision (touches R-CS-14, R-SA-03).
- **No** modification of the F04 comparison semantics; hook-sourced profiles must fit the existing contract.
- **No** duplication of the concurrent profiling/token-efficiency research; this epic consumes `.scuba/teams/research-profiling/ledger.md`.
- **No** claim that hook-derived token numbers are billed tokens (Fork vi).
- **No** agent merges to `main`.

---

## 9. Definition of done and quality bar

**Epic is done when:**

1. A Cursor session run by the user on their own machine, with the hook installed, produces a schema-valid `Profile` with at least `tool_calls`, `timing`, and `skill_activation` in state `present` with `Source: "hooks"`.
2. `profiler compare` over two such profiles (with-skill vs without-skill, same snapshot hash) produces a `ComparisonReport` where at least one metric is `comparable: true` and the report honestly marks everything else.
3. Every metric the hooks cannot supply is `unknown` with a specific, actionable reason — and no number in the profile is an estimate presented as a measurement (Fork vi resolved and implemented).
4. Installing and uninstalling the hook leaves a pre-existing `~/.cursor/hooks.json` byte-identical to its backup apart from our own entries.
5. No prompt text, email, bearer token, or API key appears in the spool under default settings.
6. `.out-of-scope.md` and `README.md` reflect what the plugin now actually does (R-SA-17).

**Quality bar, every slice:**

- Tests land **with** the slice, not after. Go tests in `profiler/profiler_test.go` (or a new `_test.go`); bash tests in `tests/`, wired into CI.
- `go vet ./...` clean; `go build` clean; `gofmt` clean.
- All 131 bash tests + existing 34 Go tests stay green (R-SA-16).
- **No invented telemetry names.** Every `cursor.*` / `gen_ai.*` string is traceable to a captured payload, Cursor's docs, or the OTel semconv — including an audit of the three already in `cursor.go`.
- Every doc claim carries an `AGENTS.md` evidence label; claims about hook payloads stay **Local hypothesis** until a real payload proves them, then become **Repository fact** with the fixture as the citation.
- No Python. No new Go dependencies without an argued decision recorded against Fork (v).
- Hook execution adds no user-perceptible latency, with a measured number in S1's report.
- No commit trailers (`AGENTS.md` git rule).

---

## 10. Open questions

### The spec must resolve these (investigation, not user input)

- **Q-S1.** What fields does each Cursor hook event *actually* carry? Only real captured payloads settle this. Everything in §3 that references a field name is a hypothesis until S2.
- **Q-S2.** Does Cursor fire any event on skill invocation, or must activation be *inferred* from `beforeReadFile` on a `SKILL.md` path? If inferred, the profile must label it inference, and `ActivationEntry.Trigger` should say so.
- **Q-S3.** Are `cursor.token.usage`, `cursor.skill.activated`, `cursor.hook.execution_complete` real Cursor telemetry names, or invented in this repo? (A07 — affects existing shipped code, not just new work.)
- **Q-S4.** How do `conversation_id`, `generation_id`, and `session_id` relate, and which one is `Profile.SessionID`? (R-SA-08.)
- **Q-S5.** Where does the spool live, and what rotates/truncates it? Per-project `.scuba/`? `~/.cursor/`? A `--spool` flag with no default?
- **Q-S6.** What is the hook process's latency budget, and does a slow hook block the editor?
- **Q-S7.** How does a hook-sourced capture express itself as an `ExperimentStep` field so `experiment run` can drive it? (R-SA-15.)
- **Q-S8.** Does `Profile` need additive fields for line-edit stats and context tokens, and can that be done without a schema version bump?

### Only the user can answer these

1. **Reading (a), (b), or (c)?** Is this for `skill-architect`'s benefit, or a standalone thing you intend to publish? Everything downstream — repo layout, naming, whether the daemon exists — hangs on this. *My recommendation: (c), sequenced (b)-first.*
2. **The token question, and it is the big one.** Cursor hooks cannot give billed token counts — only chars/4 estimates, MCP self-reports, and compaction context size. Is an estimate acceptable as a *relative* efficiency signal in an A/B comparison, or does your honesty bar mean Cursor token efficiency is simply `unknown`? If the latter, the token half of your original goal is unreachable through hooks and the epic's value is attribution alone. (Fork vi.)
3. **Do you have Cursor Business/Enterprise?** An Admin API key + team ID, or access to Cursor's enterprise OTel export? Flips R-CS-26 and R-CS-24 from drop to keep, and would give real team-level token data that hooks cannot.
4. **Do you actually want an OTLP backend** (Last9, Grafana, Honeycomb, a local collector) in your workflow, or is a local JSON spool the whole ask? Decides Fork (ii) and whether S7/S8 ever run.
5. **Naming and provenance.** Comfortable with `skill-scope`? And do you want us to *reimplement* from extracted requirements (my assumption) or *port* cursorscope's MIT-licensed code with attribution?
6. **Semgrep: did you mean it literally?** It is a marginal fit here and is itself a Python package, which sits awkwardly against `AGENTS.md`'s no-Python rule. I defaulted to `grep` + `go vet` + tests for the same guardrails. Say the word and S9 becomes a real semgrep slice.
7. **Will you install a global hook into your own `~/.cursor/hooks.json`?** There is no other way to harvest real payloads, and without them S1's premise stays unverified. (We back up and can cleanly uninstall.)
8. **Epic placement.** Is this a new epic alongside F04–F06, does it precede them, or does it displace F05/F06? Relatedly: are you willing to amend `.out-of-scope.md` and `README.md`, which currently promise the plugin does *not* do live runtime capture? (R-SA-17.)

---

## Appendix — grounding facts used

- **Repository fact.** `git log`: F04 slices 1–3 committed (`0a83615`, `236152c`, `e0e2a52`); branch `main` clean; `f03-dogfood-intake` branch exists.
- **Repository fact.** `tmp/teams/architect/F04.status.md` says `Stage: parked` — stale relative to git (A14).
- **Repository fact.** `profiler/` = 3,396 lines across 9 Go files; `go.mod` declares `go 1.27.1` with zero external requires.
- **Repository fact.** `profiler/types.go` defines `SourceHooks` — referenced by no adapter.
- **Repository fact.** All four adapters return `unknown` for `skill_activation` and `attribution`.
- **Repository fact.** `.github/workflows/ci.yml` pins Go 1.23 vs `go.mod`'s 1.27.1 (A18).
- **Repository fact.** `semgrep` not installed; `jq` at `/usr/bin/jq`; `go1.27.1` at `/opt/homebrew/bin/go`.
- **External evidence.** cursorscope v0.3.7 (MIT, `@last9/cursorscope`), 19 hook events, Node ≥20, deps include 10 `@opentelemetry/*` packages + express + dotenv.
- **External evidence.** `src/attribution.js:estimateTokens()` = `Math.ceil(text.length / 4)`; `.env.example` states "Not Cursor LLM billing".
- **External evidence.** `scripts/install-global-hooks.sh` performs its hooks.json merge via an embedded `python3` heredoc.
- **Local hypothesis.** Everything in §3 asserting a specific Cursor hook field name, pending S2 fixtures.
