# OTel GenAI semconv alignment — cursor-profiler schema review

**Researcher report · 2026-09-11 · research only, no code or spec changes made.**

**Question answered:** where should cursor-profiler's field names (spool line, hook fields,
`skill-architect/profile/v1`, attribution entry) adopt OTel `gen_ai.*` names, and where should they
deliberately not?

**Answered means:** a per-field verdict with a concrete name, a stability rating, and a named
tradeoff — precise enough to write into `spec.md` §Data formats without further lookup.

Evidence labels: **[O] Observed** (I read the file/page/diff) · **[I] Inferred** (reasoned, not
directly stated by a source).

---

## Summary — the answer in five lines

1. **The GenAI semconv moved out of the main `semantic-conventions` repo and has never been
   released.** The authoritative source is now `open-telemetry/semantic-conventions-genai`, which
   has **zero tags, zero releases, and `Schema URL: TODO`**. Every `gen_ai.*` attribute is
   `development`. **[O]**
2. **Published docs are stale and actively wrong in one place we care about.**
   `opentelemetry.io/.../registry/attributes/gen-ai/` still shows
   `gen_ai.usage.cache_creation.input_tokens`; the working repo renamed it to
   `gen_ai.usage.cache_write.input_tokens` as a **breaking** change. **[O]**
3. **Skills are an open PR, not a convention.** `gen_ai.skill.name`, `gen_ai.skill.description`,
   `gen_ai.skill.source.uri`, `gen_ai.skill.resource.name` are proposed in **open PR #498**
   (updated 2026-09-11, unmerged). Nothing named "skill" exists in `main`. **[O]**
4. **Four things we need have no convention at all:** skill *activation trigger*, *attribution*
   (output → skill), *estimated-vs-billed token provenance*, and *hook lifecycle events*. Each has
   an open issue but none has a merged shape. **[O]**
5. **Recommendation: do not rename `profile/v1` fields to `gen_ai.*`.** Keep the internal schema as
   the F04 consumer contract; add semconv names only as an **export-time alias layer** in the
   conditional S7 OTLP slice. Take exactly **two** internal renames now (`cache_creation` →
   `cache_write`; `ToolCallEntry.Success` → error-typed) because both are cheap today and both
   align with Cursor's *own* Admin API spelling as well as semconv.

---

## 1. Semconv snapshot

| Fact | Value | Evidence |
|---|---|---|
| Authoritative repo | `open-telemetry/semantic-conventions-genai`, created 2026-05-05, default branch `main` | [O] `gh api repos/open-telemetry/semantic-conventions-genai` |
| Commit reviewed | `0c875949` (`0c875949751956…`), 2026-09-10T04:43:13Z | [O] `git log -1` on shallow clone |
| Tags / releases | **none** — `gh api .../tags` and `.../releases` both empty | [O] |
| Schema URL | `TODO` | [O] `README.md:16-18` of the genai repo |
| CHANGELOG | only an `## Unreleased` section; all changes sit as towncrier fragments in `changelog.d/` | [O] `CHANGELOG.md:1-10` |
| Stability of every `gen_ai.*` attribute, span, metric, event | **`development`** — no `stable` or `release_candidate` entry exists anywhere in `model/gen-ai/` | [O] `grep -c 'stability: stable' model/gen-ai/*.yaml` → 0 |
| Move happened between | semconv **v1.41.0** (`model/gen-ai/` still full) and **v1.42.0** (only `deprecated/`), i.e. 2026-04-28 → 2026-06-12 | [O] `gh api .../contents/model/gen-ai?ref=v1.4x.0` |
| Latest main semconv release | **v1.44.0**, 2026-08-04 — carries only `model/gen-ai/deprecated/` | [O] |
| Published page status | `opentelemetry.io/docs/specs/semconv/gen-ai/` is a redirect stub: *"GenAI semantic conventions have moved… This page has moved and is no longer maintained in this repository."* | [O] fetched page + `docs/gen-ai/README.md` in semconv `main` |

### Per-group stability

| Group | Where | Stability | Notes |
|---|---|---|---|
| `gen_ai.*` registry (73 keys) | `model/gen-ai/registry.yaml` | development | every key, every enum member [O] |
| GenAI spans (11 span types) | `model/gen-ai/spans.yaml:171-700` | development | `inference.client`, `embeddings.client`, `retrieval.client`, `fetch_response.client`, `memory.client`, `create_agent.client`, `invoke_agent.client`, `invoke_agent.internal`, `execute_tool.internal`, `invoke_workflow.internal`, `plan.internal` [O] |
| GenAI metrics (11) | `model/gen-ai/metrics.yaml` | development | all histograms; **no counters exist yet** [O] |
| GenAI events (3) | `model/gen-ai/events.yaml` | development | `gen_ai.client.inference.operation.details` (opt-in), `gen_ai.evaluation.result`, `gen_ai.client.operation.exception` [O] |
| `mcp.*` registry (4 keys) | `model/mcp/registry.yaml` | development | `mcp.method.name`, `mcp.session.id`, `mcp.resource.uri`, `mcp.protocol.version` [O] |
| Provider-specific (`openai.*`, `aws.bedrock.*`, `anthropic`, `azure.ai.*`) | `model/openai/`, `model/aws-bedrock/` | development | no Cursor-relevant content [O] |
| `gen_ai.*` in main semconv v1.44.0 | `model/gen-ai/deprecated/` | **deprecated** | whole `registry.gen_ai` group carries `deprecated: note: "Moved to the … GenAI semantic conventions repository"` [O] |

**Deprecated names to never emit:** `gen_ai.usage.prompt_tokens` (→ `input_tokens`),
`gen_ai.usage.completion_tokens` (→ `output_tokens`), `gen_ai.prompt`, `gen_ai.completion` (removed,
no replacement), `gen_ai.system` (→ `gen_ai.provider.name`). [O] `registry-deprecated.yaml:19-65`

---

## 2. Field-by-field mapping

Sources abbreviated: `types.go` / `hooks.go` / `cursor.go` = `/Users/matthewvandusen/Development/Auraprix/skill-architect/profiler/*`;
`reg` = `model/gen-ai/registry.yaml` @ `0c875949`; `surfaces` = `docs/research/cursor-telemetry-surfaces.md`.

### 2a. Spool line envelope (`SpoolEvent`, `hooks.go:25-33`)

| Our field | Type | Closest semconv | Semconv type / enum | Stability | Verdict | Evidence |
|---|---|---|---|---|---|---|
| `ts` | string RFC3339 | none (span start/end timestamps are envelope, not attributes) | — | — | no equivalent; keep | [O] `hooks.go:27` |
| `event` | string (hook name) | **no equivalent** — hook lifecycle unmodelled | — | — | gap, see §3 | [O] `hooks.go:28`; grep "hook" in `model/` → 0 hits |
| `schema_version` | `"cursor-profiler/spool/v1"` | none | — | — | keep | [O] `hooks.go:20` |
| `cursor_version` | string | no `gen_ai.harness.version`; closest is core `service.version` on the Resource | string, stable (core) | stable (core) | export as resource attr | [O] `hooks.go:29`; [I] mapping |
| `cwd` | string | none | — | — | keep | [O] `hooks.go:30` |
| `conversation_id` | string | **`gen_ai.conversation.id`** | string | development | **direct match** | [O] `hooks.go:31`; `reg:412` |
| `raw` | JSON blob | none | — | — | keep | [O] `hooks.go:32` |

> **Divergence to note:** `spec.md:36` (R-CS-02) promises the envelope also carries `generation_id`,
> `session_id`, `model`, `workspace_roots`, user, repo, timestamp. The live `SpoolEvent` promotes
> only `cursor_version`, `cwd`, `conversation_id`; the rest survive inside `raw`. Any mapping table
> written into `spec.md` must say which layer it describes. [O] `spec.md:36` vs `hooks.go:25-33`

### 2b. Cursor hook payload fields → semconv

| Hook field | Closest semconv | Type / enum | Stability | Verdict | Evidence |
|---|---|---|---|---|---|
| `conversation_id` | `gen_ai.conversation.id` | string | dev | match | [O] `reg:412` |
| `generation_id` | `gen_ai.response.id` (loose) | string | dev | **weak** — `response.id` identifies one completion; a Cursor generation may span several | [I] |
| `session_id` | **no `gen_ai.*` equivalent.** Core semconv has `session.id` | string | dev | gap; issues #51, #303 open | [O] `grep session.id model/` → only `mcp.session.id` |
| `model`, `model_id` | `gen_ai.request.model` | string | dev | match | [O] `reg:98` |
| `model_params` | `gen_ai.request.temperature` / `.top_p` / `.top_k` / `.max_tokens` / `.seed` / `.stop_sequences` | double/int/string[] | dev | decompose, don't blob | [O] `reg:103-166` |
| `hook_event_name` | none | — | — | gap §3 | [O] |
| `workspace_roots` | none | — | — | no equivalent | [O] |
| `user_email` | core `user.email` — **but R-CS-18 says redact** | string | dev | drop | [O] `spec.md:51` |
| `tool_name` | **`gen_ai.tool.name`** | string | dev | match | [O] `reg:473` |
| `tool_use_id` | **`gen_ai.tool.call.id`** | string | dev | match, but see open #489 (should it be provider-supplied only?) | [O] `reg:478`; issue #489 |
| `tool_input` | `gen_ai.tool.call.arguments` | `any`, **opt_in** | dev | opt-in only; we redact | [O] `reg:502`; `spans.yaml:611` |
| `tool_output` / `result_json` | `gen_ai.tool.call.result` | `any`, **opt_in** | dev | opt-in only | [O] `reg:522` |
| `duration` / `duration_ms` | metric `gen_ai.execute_tool.duration`, **unit `s` (double)** | histogram | dev | **unit conflict**: ours is ms | [O] `metrics.yaml:232-247` |
| `error_message` | `error.type` (core) — low cardinality **only** | string | stable (core) | **do not** put the message in `error.type`; message → span status description | [O] `spans.yaml:3-12` |
| `failure_type`, `is_interrupt` | `error.type` value space | string | stable (core) | map to low-card codes | [I] |
| `status` (stop, subagentStop) | `gen_ai.response.status` = `queued\|in_progress\|completed\|incomplete\|failed\|cancelled` | enum | dev | **but it is referenced only on `fetch_response` / `openai.fetch_response` spans** — reusing it on a session span is off-label | [O] `reg:230-268`; `grep` → `spans.yaml:389,768` only |
| `mcp_server_name`, `mcp_server_url` | **no `mcp.server.name`** exists; semconv carries server identity via `server.address` / `server.port` | string | dev | gap; issue #437 open | [O] `model/mcp/registry.yaml` has only 4 keys |
| `file_path`, `edits[]` | none | — | — | no equivalent | [O] |
| `subagent_id` | **NOT `gen_ai.agent.id`** — that key is for provider-assigned *stable resource* ids and says "NOT RECOMMENDED to record in-memory agent instance ids" | string | dev | do not map | [O] `reg:446-456` |
| `subagent_type` | `gen_ai.agent.name` | string | dev | match (human-readable name) | [O] `reg:458` |
| `parent_conversation_id` | none; parenthood is span parentage | — | — | structural, not an attribute | [I] |
| `context_tokens` (preCompact) | **not** `gen_ai.usage.input_tokens` — that key is per-request usage; `context_tokens` is context-window occupancy | int | dev | see §3 G7/G8 | [O] `reg:278-299`; cursorscope does map it this way (`surfaces:59-60`) — **we should not copy that** |
| `context_window_size`, `context_usage_percent` | no equivalent | — | — | gap | [O] |
| `is_first_compaction` / `trigger` (preCompact) | **`gen_ai.conversation.compacted`** (boolean, positive-indicator-only: set only when true, never `false`) | boolean | dev | **real match, recently landed** | [O] `reg:436-445` |
| `prompt`, `text`, `content` | `gen_ai.input.messages` / `gen_ai.output.messages` / `gen_ai.system_instructions` (opt_in) | complex | dev | we redact; do not emit | [O] `reg:830,872,926` |
| shell `exit_code` (R-CS-15) | none in semconv; `gen_ai.skill.script.exit_code` appeared in an **earlier revision of PR #498 and is not in the current diff** | — | — | gap | [O] PR #498 body mentions it; current `registry.yaml` diff adds only 4 keys, none an exit code |

### 2c. `skill-architect/profile/v1` (`types.go`) → semconv

| Our field | Path | Closest semconv | Type / enum | Stability | Verdict | Evidence |
|---|---|---|---|---|---|---|
| `harness: "cursor"` | `types.go:157` | **none.** `gen_ai.harness.name` is proposed only (issue #301, which names Cursor explicitly as an example) | string | n/a | gap §3. **Do not** use `gen_ai.provider.name` — `cursor` is not a member of its closed enum | [O] `reg:3-73` lists 16 members, no `cursor` |
| `session_id` | `types.go:158` | core `session.id`; no `gen_ai.*` | string | dev | gap §3 | [O] |
| `tokens.value.input` | `types.go:81` | **`gen_ai.usage.input_tokens`** | int | dev | match | [O] `reg:278` |
| `tokens.value.output` | `types.go:82` | **`gen_ai.usage.output_tokens`** | int | dev | match | [O] `reg:336` |
| `tokens.value.cache_read` | `types.go:83` | **`gen_ai.usage.cache_read.input_tokens`** | int | dev | match | [O] `reg:301` |
| `tokens.value.cache_creation` | `types.go:84` | **`gen_ai.usage.cache_write.input_tokens`** | int | dev | **renamed upstream — breaking change #440**; published otel.io registry still shows `cache_creation` | [O] `changelog.d/440.breaking.md`; `reg:308`; otel.io registry page |
| `tokens.value.reasoning` | `types.go:85` | **`gen_ai.usage.reasoning.output_tokens`** | int | dev | match | [O] `reg:346` |
| (absent) modality splits | — | `gen_ai.usage.{text,image,audio}.{input,output}_tokens`, `gen_ai.usage.{text,image,audio}.cache_read.input_tokens` | int | dev | no Cursor source; do not add | [O] `reg:315-396` |
| `tokens.source` enum | `types.go:40-47` (`otel\|hooks\|session_data\|server_api\|sqlite\|none`) | **no equivalent.** Nearest precedent: `gen_ai.usage.cost.source` = `provider\|local` in **open PR #443** | enum | n/a (unmerged) | gap §3 G7 | [O] PR #443 diff |
| `tool_calls[].name` | `types.go:90` | **`gen_ai.tool.name`** | string | dev | match | [O] `reg:473` |
| `tool_calls[].timestamp` | `types.go:91` | span start time | — | — | structural | [I] |
| `tool_calls[].success` | `types.go:92` | **no boolean.** semconv models failure as presence of `error.type`; success = attribute absent | string | stable (core) | polarity mismatch | [O] `spans.yaml:3-12` |
| (absent) tool kind | — | `gen_ai.tool.type`, free string, examples `function\|extension\|datastore` | string | dev | **different axis** from our `mcp\|cli\|skill\|subagent\|tool` | [O] `reg:491-501` |
| `skill_activation[].skill_name` | `types.go:97` | **`gen_ai.skill.name`** — **open PR #498 only** | string | unmerged | see §4 | [O] PR #498 registry diff |
| `skill_activation[].trigger` | `types.go:99` | **no equivalent, proposed or landed** | — | — | gap §3 G2 | [O] grep skill in `model/` → 0 |
| `skill_activation[].timestamp` | `types.go:98` | span start time | — | — | structural | [I] |
| `timing.start_time` / `.end_time` | `types.go:104-105` | span start/end | — | — | structural | [I] |
| `timing.total_ms` | `types.go:106` | `gen_ai.client.operation.duration` / `gen_ai.invoke_agent.duration`, **unit `s`, double** | histogram | dev | **unit conflict** | [O] `metrics.yaml:46-60,152-182` |
| `attribution[].target` | `types.go:116` | `gen_ai.tool.call.id` when the target is a tool call | string | dev | partial | [O] `reg:478` |
| `attribution[].skill_name` | `types.go:117` | `gen_ai.skill.name` (unmerged) | string | unmerged | partial | [O] PR #498 |
| `attribution[].category` (spec) | `spec.md:148` | none (`gen_ai.tool.type` is a different axis) | — | — | gap | [O] |
| `attribution[].detail` (spec) | `spec.md:149` | nearest `gen_ai.skill.resource.name` (unmerged) for the file case | string | unmerged | partial | [O] PR #498 registry diff |
| `attribution[].operation_name` (spec) | `spec.md:150` | **`gen_ai.operation.name`**; our value `execute_tool` is a valid member | enum (20 members incl. `execute_tool`, `invoke_agent`, `invoke_workflow`, `plan`, `create_agent`, `chat`, `embeddings`, `retrieval`, memory ops) | dev | **match — keep the name and the value** | [O] `reg:600-681` |
| `attribution[].confidence` (spec) | `spec.md:151` | **no equivalent.** Nearest open proposal `gen_ai.evidence.origin = self_reported\|externally_observed` (issue #386) is a *vantage* axis, not a *certainty* axis | — | n/a | gap §3 G6 | [O] issue #386 |
| `estimated_context_tokens` | `spec.md:131` | **no equivalent** | — | — | gap §3 G7 | [O] `grep -i estimat model/` → 0 hits |
| `capability.capabilities` | `types.go:54` | none | — | — | keep | [O] |
| `snapshot_hash`, `skill_dir` | `types.go:159-160` | none | — | — | keep | [O] |

### 2d. Cursor's own `cursor.*` wire attributes vs `gen_ai.*` — conflicts

| Cursor wire name | Values | `gen_ai.*` counterpart | Conflict? | Evidence |
|---|---|---|---|---|
| `cursor.conversation.id` | string | `gen_ai.conversation.id` | same concept, different key — pure rename at export | [O] `surfaces:95`; `reg:412` |
| `cursor.token.type` | `input\|output\|cache_read\|cache_creation` | `gen_ai.token.type` = **`input\|output` only** | **hard conflict.** `cache_read` / `cache_creation` are **not** legal `gen_ai.token.type` values; they are separate `gen_ai.usage.*` attributes | [O] `surfaces:102`; `reg:398-410` |
| `cursor.api.request.cache_creation_tokens` | int | `gen_ai.usage.cache_write.input_tokens` | naming conflict after upstream rename #440 | [O] `surfaces:110`; `changelog.d/440.breaking.md` |
| `cursor.tool.kind` | `builtin\|mcp` | `gen_ai.tool.type` ex. `function\|extension\|datastore` | **value-space conflict** — do not coerce | [O] `surfaces:103`; `reg:491` |
| `cursor.tool.status` | `success\|failure\|aborted` | `error.type` presence/absence | polarity conflict | [O] `surfaces:103` |
| `cursor.skill.name` | string | `gen_ai.skill.name` (unmerged) | would be a straight alias **if #498 merges** | [O] `surfaces:113`; PR #498 |
| `cursor.skill.trigger` | `agent_read\|manually_attached\|skill_name_in_prompt` | none | no counterpart at all | [O] `surfaces:113` |
| `cursor.skill.source` | `unspecified\|workspace\|user\|builtin\|plugin\|claude` | `gen_ai.skill.source.uri` (unmerged) is a **URI**, not a category | **shape conflict** — enum vs URI | [O] `surfaces:113`; PR #498 |
| `cursor.hook.*` (name/type/outcome/duration_ms) | — | none | no counterpart; issue #320 open | [O] `surfaces:114` |
| `cursor.cost.usage` (USD) | double | `gen_ai.usage.cost.amount` + `.currency` + `.source` in **open PR #443** | would align if #443 merges | [O] `surfaces:104`; PR #443 diff |
| `cursor.model.name` | string | `gen_ai.request.model` / `gen_ai.response.model` | rename at export | [O] `surfaces:102`; `reg:98,205` |

> **Correctness note carried forward, not a semconv issue:** `cursor.go:182-188` treats
> `cursor.hook.execution_complete` as the tool-call source. Per `surfaces:122-124` that event is
> about **hook executions**, not tool calls; tool calls live in the `cursor.tool.calls` metric.
> Also, none of the `cursor.*` names in `cursor.go` were ever verified against a live export
> (`spec.md:78`, R-SA-14). Any semconv alias layer inherits that defect unless S6 retires it. [O]

---

## 3. Gaps in the standard

| # | What we need | State of the standard | Open issue / PR | Evidence |
|---|---|---|---|---|
| G1 | **Skill identity** (`skill_name`) | Nothing merged. PR **#498** (open, non-draft, updated 2026-09-11) adds `gen_ai.skill.name`, `gen_ai.skill.description`, `gen_ai.skill.source.uri`, `gen_ai.skill.resource.name` as **refinements on the `execute_tool` span**, not a new span type. Umbrella issue **#501**; span-type alternative in **#86**. | #498 open, #501 open, #86 open | [O] PR #498 diff of `model/gen-ai/registry.yaml` |
| G2 | **Skill *activation trigger*** (how the skill got loaded) | **No proposal exists.** #498 records *that* a skill loaded and from where, never *why*. Cursor's `cursor.skill.trigger` is first-party and unmatched. | none found | [O] grep `trigger` in `model/` and in PR #498 diff |
| G3 | **Skill metrics** | PR **#499** (open, **draft**) adds the registry's first counters: `gen_ai.skill.loads` (`{load}`), `gen_ai.skill.script.executions` (`{execution}`), plus histograms `gen_ai.invoke_agent.skill.loads` and `gen_ai.invoke_workflow.skill.loads` (`{skill}`), and attribute `gen_ai.skill.script.exited_with_error`. Naming is an explicitly open question in #501. | #499 open/draft | [O] PR #499 diff |
| G4 | **Skills-in-context list** | PR **#500** (`gen_ai.skills` opt-in array with per-skill `compacted`) is **CLOSED** (2026-09-10). | #500 closed | [O] `gh pr view 500` → `state=CLOSED` |
| G5 | **Attribution** (output → skill) | Nothing. Nearest: issue **#402** proposes `gen_ai.agent.run_id`, `gen_ai.agent.step_id`, `gen_ai.tool.side_effect_class`, `gen_ai.tool.args_hash`, `gen_ai.tool.result_hash` — correlation and traceability, not attribution. Issue **#243** asks how to represent multiple agents on the same telemetry. | #402, #243 open | [O] issue bodies |
| G6 | **Confidence / inferred-vs-observed** | Nothing merged. Issue **#386** proposes `gen_ai.evidence.origin = self_reported \| externally_observed` — *who observed it*, not *how sure we are*. It explicitly adopts the positive-indicator-only pattern (unset ≠ false). | #386 open | [O] issue #386 |
| G7 | **Estimated vs billed tokens** | **No token-provenance attribute exists, proposed or merged.** `gen_ai.usage.input_tokens`'s own note says instrumentations SHOULD report the *billed* count when a provider exposes both — i.e. the key is reserved for measured/billed values, which is exactly why a `chars/4` estimate must not go there. The only provenance precedent anywhere is `gen_ai.usage.cost.source = provider \| local` in **open PR #443**; per-class follow-up in **#484**; billing-detail wishlist in **#503**; generic cost proposal **#287**. | #443, #484, #503, #287 open | [O] `reg:278-299`; PR #443 diff |
| G8 | **Context-window occupancy** (`context_tokens`, `context_window_size`, `context_usage_percent`) | No attribute. Only the boolean **`gen_ai.conversation.compacted`** exists (landed, `reg:436`), plus issue **#508** proposing it as a duration-metric dimension. | #508 open | [O] `reg:436`; issue #508 |
| G9 | **Hook lifecycle events** | Nothing. Issue **#320** ("Agent Harness Hook Semantic Conventions") asks for both context propagation into hooks and span conventions for hook invocation; it tabulates Claude Code / Antigravity / OpenCode hooks but **not Cursor**. No PR. | #320 open | [O] issue #320 |
| G10 | **Harness identity** (`harness: "cursor"`) | Nothing. Issue **#301** proposes `gen_ai.harness.name` and lists `"Cursor"` as a worked example; naming still open (`harness` vs `orchestrator` vs `system`). | #301 open | [O] issue #301 |
| G11 | **Session id above conversation** | Nothing in `gen_ai.*`. Issues **#51** and **#303** both argue for reusing core `session.id`, with the hierarchy `session.id > gen_ai.conversation.id`. Unresolved since 2025-10-07. | #51, #303 open | [O] issue #51 |
| G12 | **MCP server identity by name** | `mcp.*` has only 4 attributes; no server name. Issue **#437** asks for peer server implementation metadata and warns against confusing a self-declared name with verified identity. | #437 open | [O] `model/mcp/registry.yaml`; issue #437 |

---

## 4. Recommendations

Two distinct actions, kept separate throughout:
**IR** = *internal rename* (changes `profile/v1` / spool JSON, a consumer contract for F04).
**XA** = *export-time alias* (a `gen_ai.*` key emitted only by the conditional S7 OTLP exporter; the
internal field name is untouched).

### 4a. Internal renames — take exactly two

| # | Field | Rating | Concrete change | Why now | Tradeoff |
|---|---|---|---|---|---|
| R1 | `TokenCounts.CacheCreation` → `CacheWrite`, JSON `cache_creation` → `cache_write` | **rename (IR)** | `types.go:84` | Upstream renamed it (breaking #440 [O]); Cursor's **own** Admin API already spells it `cacheWriteTokens` (`surfaces:143` [O]). Two of three Cursor surfaces agree with semconv. Field is `omitempty` and no Cursor source populates it today, so the rename costs one struct tag and a test. | Diverges from Cursor's Enterprise-OTel spelling `cursor.token.type=cache_creation`; the OTel-export reader (`cursor.go:207-209`) must map. |
| R2 | `ToolCallEntry.Success bool` → keep it, **add** `ErrorType string \`json:"error_type,omitempty"\`` | **add (IR)** | `types.go:89-93` | semconv has no success boolean; failure is the presence of `error.type` [O] `spans.yaml:3-12`. Cursor gives us `failure_type` and `error_message` on `postToolUseFailure` (`surfaces:37` [O]), which `success: false` throws away. | One more field in v1; `spec.md:27` allows additive fields only — this qualifies. |

**Do not rename anything else internally.** `profile/v1` is F04's input contract (R-SA-01,
R-SA-06); a `gen_ai.`-prefixed JSON key buys nothing for a file-based consumer and imports
`development`-stability naming risk into a schema we promised not to version-bump (`spec.md:27`).

### 4b. Export-time aliases (S7 only, conditional)

| # | Our field | Emit as | Rating | Confidence to emit | Evidence |
|---|---|---|---|---|---|
| R3 | `conversation_id` | `gen_ai.conversation.id` | **add (XA)** | high | [O] `reg:412` |
| R4 | `tokens.value.input/output/cache_read/reasoning` | `gen_ai.usage.input_tokens` / `.output_tokens` / `.cache_read.input_tokens` / `.reasoning.output_tokens` | **add (XA)** | high — **only when `source ∈ {server_api, otel}`** | [O] `reg:278,336,301,346` |
| R5 | `tool_calls[].name` | `gen_ai.tool.name` | **add (XA)** | high | [O] `reg:473` |
| R6 | `tool_use_id` | `gen_ai.tool.call.id` | **add (XA)** | medium — #489 may restrict it to provider-supplied ids | [O] `reg:478`; issue #489 |
| R7 | attribution `operation_name` | `gen_ai.operation.name` (value `execute_tool`) | **keep (already correct)** | high | [O] `reg:600,637` |
| R8 | hook `model` | `gen_ai.request.model` | **add (XA)** | high | [O] `reg:98` |
| R9 | `subagent_type` | `gen_ai.agent.name` on a `gen_ai.invoke_agent.internal` span | **add (XA)** | medium | [O] `reg:458`; `spans.yaml:549` |
| R10 | `subagent_id` | **nothing** — do **not** emit `gen_ai.agent.id` | **drop** | high | [O] `reg:446-456` ("NOT RECOMMENDED to record in-memory agent instance ids") |
| R11 | `preCompact` occurred | `gen_ai.conversation.compacted = true` (never `false`) | **add (XA)** | high — a real, landed match | [O] `reg:436-445` |
| R12 | tool-call span | name `execute_tool {gen_ai.tool.name}`; span kind INTERNAL | **add (XA)** | high | [O] `spans.yaml:573-580` |
| R13 | durations | `gen_ai.execute_tool.duration`, `gen_ai.client.operation.duration` — **in seconds, double** | **add (XA)** | high; convert from ms at the boundary | [O] `metrics.yaml:46,232` |
| R14 | `harness: "cursor"` | `cursor_profiler.harness.name` (our namespace) — **not** `gen_ai.provider.name` | **add (XA), own namespace** | high on the prohibition; switch to `gen_ai.harness.name` only if #301 merges | [O] `reg:3-73` (no `cursor` member); issue #301 |
| R15 | `session_id` | core `session.id` | **add (XA)** | medium — #51/#303 unresolved for 11 months | [O] issue #51 |
| R16 | `skill_activation[].skill_name` | `gen_ai.skill.name` — **behind a feature flag, off by default until #498 merges** | **add (XA), gated** | **low** — unmerged PR | [O] PR #498 open |
| R17 | `skill_activation[].trigger` | `cursor_profiler.skill.trigger`, reusing Cursor's own values (`agent_read\|manually_attached\|skill_name_in_prompt`) | **add (XA), own namespace** | high — nothing to collide with | [O] G2; `surfaces:113` |
| R18 | `estimated_context_tokens` | `cursor_profiler.context.estimated_tokens` + `cursor_profiler.token.source = estimated\|mcp_reported\|cursor_reported\|billed` | **add (XA), own namespace** | high on the prohibition; the shape mirrors `gen_ai.usage.cost.source` so a later migration is mechanical | [O] G7; PR #443 |
| R19 | `attribution[]` | `cursor_profiler.attribution.*` (`target`, `skill_name`, `category`, `detail`, `confidence`) | **add (XA), own namespace** | high | [O] G5 |
| R20 | `mcp_server_name` | `cursor_profiler.mcp.server.name` **and** core `server.address` when a URL is present | **add (XA), own namespace + core** | medium | [O] G12; `model/mcp/common.yaml` |
| R21 | `event` (hook name) | `cursor_profiler.hook.name` / `.outcome` | **add (XA), own namespace** | high | [O] G9 |

### 4c. Explicit prohibitions

| # | Do not | Rating | Why |
|---|---|---|---|
| R22 | Emit a `chars/4` estimate as `gen_ai.usage.input_tokens` | **drop** | The attribute's own note reserves it for provider-reported, billing-aligned counts [O] `reg:281-291`. Doing so launders an estimate — the exact failure `spec.md:195` (fork vi) exists to prevent. |
| R23 | Emit `preCompact.context_tokens` as `gen_ai.client.token.usage` with `gen_ai.token.type=input` (cursorscope's mapping, `surfaces:59-60`) | **drop** | Context-window occupancy is cumulative state, not per-request usage. Copying it would overstate billed input by roughly the conversation length. [I], grounded in `reg:278` and `surfaces:73-84` |
| R24 | Emit `gen_ai.token.type = cache_read` or `cache_creation` | **drop** | The enum has exactly two members, `input` and `output` [O] `reg:398-410`. Cursor's `cursor.token.type` has four; they are not interchangeable. |
| R25 | Map our `category` (`mcp\|cli\|skill\|subagent\|tool`) onto `gen_ai.tool.type` | **drop** | Different axis: `gen_ai.tool.type` classifies *where the tool executes* (`extension` = agent-side, `function` = client-side, `datastore`) [O] `reg:491-501`. |
| R26 | Add the OTel Go SDK for semconv constants | **keep existing decision** | `go-dependency-and-tooling-decisions.md:42-54` already decided hand-rolled OTLP/JSON [O]. The semconv-constant argument is **weaker now**, not stronger: the GenAI registry is unreleased, so no released Go semconv package contains `gen_ai.skill.*` or `gen_ai.usage.cache_write.input_tokens` anyway. |
| R27 | Pin to a `gen_ai` semconv *version* string | **cannot** | There is no version to pin to. Pin to the **commit** (`0c875949`, 2026-09-10) in a single declaring Go file, consistent with guardrail rule 4 in `go-dependency-and-tooling-decisions.md:85` [O]. |

---

## 5. Stability risks

| # | Risk | Severity | Evidence | Mitigation |
|---|---|---|---|---|
| S-1 | **Nothing is stable.** Every `gen_ai.*` name we might adopt is `development`; the repo has never cut a release and its Schema URL is `TODO`. | high | [O] §1 | Keep all `gen_ai.*` names in the export layer only; one declaring file; no `gen_ai.*` literal in `types.go`. |
| S-2 | **`gen_ai.skill.*` may never land, or land differently.** #498 is open and unmerged; #501 leaves "`execute_tool` refinement vs dedicated skill span" explicitly open, and #86 argues the other way. #500 was already closed. | **highest for us** — skills are the whole value proposition | [O] #498, #501, #86, #500 | R16: gated, off by default. Never make `skill_activation: present` depend on a `gen_ai.*` name. |
| S-3 | **Token attribute names have already broken once.** `gen_ai.usage.cache_creation.input_tokens` → `cache_write` (#440). Cache attributes were then **removed** from the internal `invoke_agent` span (#469) as "misleading". | high | [O] `changelog.d/440.breaking.md`, `469.breaking.md` | R1 now; treat the cache split as inference-span-only on export. |
| S-4 | **Published docs contradict the working repo.** otel.io's registry page still lists `cache_creation`; the working repo lists `cache_write`. Anyone reading the website gets the stale name. | medium | [O] both fetched | Cite the repo commit, never the website, in code comments. |
| S-5 | **Seven breaking fragments are queued for the first release** — type change (`top_k` double→int), attribute removals from spans (`gen_ai.agent.id`, `gen_ai.agent.version`, cache tokens), a scoping change on `gen_ai.provider.name`, a `system_instructions` restriction, plus one deprecation (`gen_ai.output.messages[].finish_reason`). The first tag will be a large step. | high | [O] `changelog.d/*.breaking.md` ×7, `363.deprecation.md` | Re-run this review at the first tagged release before S7 ships. |
| S-6 | **Counters are unprecedented here.** #499 would add the registry's first counter instruments; every existing `gen_ai.*` and `mcp.*` metric is a histogram. If the SIG rejects the shape, `gen_ai.skill.loads` disappears. | medium | [O] #499 body; #501 open question 2 | Do not design our metric surface around #499. |
| S-7 | **`gen_ai.harness.name` naming is unsettled** (`harness` vs `orchestrator` vs `system`), and it is the one attribute that would carry `"cursor"`. | medium | [O] #301 open questions | R14: own namespace until merged. |
| S-8 | **`session.id` vs `gen_ai.conversation.id` hierarchy is unresolved since 2025-10-07.** Our `Profile.SessionID` is populated from `conversation_id`/`generation_id`/`session_id` interchangeably (R-SA-08, `spec.md:72`), which will be wrong under whichever hierarchy lands. | medium | [O] #51, #303; `spec.md:72` | Decide *our* hierarchy explicitly in `spec.md` rather than inheriting the ambiguity. |
| S-9 | **`gen_ai.tool.call.id` semantics may narrow** (#489: provider-supplied identifiers only). Cursor's `tool_use_id` is harness-supplied, not provider-supplied. | low–medium | [O] #489 | R6 is medium-confidence; keep `target` internal. |
| S-10 | **`gen_ai.response.status` is off-label for session status.** It is referenced only on `fetch_response` spans. Using it for `sessionEnd.final_status` would be an invented usage — the exact class of thing R-SA-14 forbids. | medium | [O] `grep` → `spans.yaml:389,768` only | Use our own namespace for session status. |

---

## Decisions needed above my level

1. **Is S7 (OTLP export) going to happen?** Every `gen_ai.*` recommendation here is XA-scoped and
   therefore dead weight if S7 stays unbuilt. Only R1 and R2 (internal) are worth doing regardless.
2. **`profile/v1` additive fields R1/R2 touch `docs/profiler-spec.md`.** Per
   `go-dependency-and-tooling-decisions.md:194-198` (L-4), `types.go` and `docs/profiler-spec.md`
   must move as one owned, versioned artifact; no slice currently owns that pairing.

## Sources

- `open-telemetry/semantic-conventions-genai` @ `0c875949`, 2026-09-10 — cloned and read locally:
  `model/gen-ai/registry.yaml`, `spans.yaml`, `metrics.yaml`, `events.yaml`, `model/mcp/*.yaml`,
  `CHANGELOG.md`, `changelog.d/*`, `versions.env`, `README.md`
- PRs: [#498](https://github.com/open-telemetry/semantic-conventions-genai/pull/498),
  [#499](https://github.com/open-telemetry/semantic-conventions-genai/pull/499),
  [#500](https://github.com/open-telemetry/semantic-conventions-genai/pull/500) (closed),
  [#443](https://github.com/open-telemetry/semantic-conventions-genai/pull/443)
- Issues: [#501](https://github.com/open-telemetry/semantic-conventions-genai/issues/501),
  [#86](https://github.com/open-telemetry/semantic-conventions-genai/issues/86),
  [#320](https://github.com/open-telemetry/semantic-conventions-genai/issues/320),
  [#301](https://github.com/open-telemetry/semantic-conventions-genai/issues/301),
  [#51](https://github.com/open-telemetry/semantic-conventions-genai/issues/51),
  [#303](https://github.com/open-telemetry/semantic-conventions-genai/issues/303),
  [#386](https://github.com/open-telemetry/semantic-conventions-genai/issues/386),
  [#402](https://github.com/open-telemetry/semantic-conventions-genai/issues/402),
  [#437](https://github.com/open-telemetry/semantic-conventions-genai/issues/437),
  [#484](https://github.com/open-telemetry/semantic-conventions-genai/issues/484),
  [#489](https://github.com/open-telemetry/semantic-conventions-genai/issues/489),
  [#503](https://github.com/open-telemetry/semantic-conventions-genai/issues/503),
  [#508](https://github.com/open-telemetry/semantic-conventions-genai/issues/508),
  [#287](https://github.com/open-telemetry/semantic-conventions-genai/issues/287),
  [#243](https://github.com/open-telemetry/semantic-conventions-genai/issues/243)
- `open-telemetry/semantic-conventions` v1.44.0 — `model/gen-ai/deprecated/registry-deprecated.yaml`,
  `docs/gen-ai/README.md`
- <https://opentelemetry.io/docs/specs/semconv/gen-ai/> (redirect stub) ·
  <https://opentelemetry.io/docs/specs/semconv/registry/attributes/gen-ai/> (stale, deprecated)
- Local: `/Users/matthewvandusen/Development/Auraprix/skill-architect/profiler/{types.go,hooks.go,cursor.go}` ·
  `/Users/matthewvandusen/Development/Auraprix/cursor-profiler/spec.md` ·
  `/Users/matthewvandusen/Development/Auraprix/cursor-profiler/docs/research/{cursor-telemetry-surfaces.md,go-dependency-and-tooling-decisions.md}`
