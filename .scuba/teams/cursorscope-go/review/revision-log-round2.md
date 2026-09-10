# Revision log — E-CS roadmap, round 2

**Artifact:** `roadmap.md` (325 lines) + **17** slice files · **Date:** 2026-09-10 · **Groomer:** cursorscope-go
**Input:** `hunter-verifiability-round2.md` (9 REAL + 1 SUSPECTED + 6 LOW = **V2-1…19**) + `hunter-conformance-round2.md` (12 REAL = **C2-1…12**) = **31 dispositions**
**Method:** both hunters named the same meta-defect — *round 1 fixed the named instances, not the class*. Round 2 therefore fixes by class and **proves the sweep with tables**: §1 walks every DoD assertion in all 17 slices to the contract field that carries it; §2 cites every enum literal and every Specification / Repository-fact label to a source line; §3 recomputes every `Depends on` from DoD + Files rather than from memory. §4 is the per-finding disposition.

**Structural changes:**
1. **SC widened** from 4 channels to 8 — it now also adds tool-call identity/hierarchy (`ID`/`GenerationID`/`ParentID`), the three `preCompact` context fields, a `none` honesty class for `SourceNone`, a fail-closed `unrecognized` class for any unknown wire `Source`, **encoding-invariant** tool-call counting, and a tri-state-aware `successRate`. Every field it adds is `omitempty` and declared last, which is what makes its byte-identity DoD true instead of self-contradictory.
2. **S8 split into S8a (guardrails) + S8b (docs).** The single S8 depended on S3, so the *SP1 = no* fallback MVE did not close. S8a's deps are {SC, S0, S4}; both MVEs now close. Slice count 16 → 17; `S8.status.md` deleted.
3. **S1b creates an enrichment seam** — `enrich()` plus two no-op stubs in `attribution.go`/`tokens.go`, the files S3 and S5 then own. This *removes* the wave-3 `hooks.go` collision (S3 ‖ S5) rather than documenting it, and leaves exactly one same-wave collision in the whole epic (S0 ‖ S1b on `cursor.go`).
4. **D10 added** — the reuse-vs-build decision `AGENTS.md:8-9` makes mandatory, against Dash0 Agent Plugin and `o11y-dev/opentelemetry-hooks`.
5. §F's ownership map rebuilt from all 17 slices' `Files` sections; §C's graph rebuilt at **32 edges**.

---

## 1. Representability sweep — every DoD assertion → the field that carries it (root A)

Legend: **NEW** = a carrier SC did not have and now adds · *contract* = a field that already exists in `profiler/types.go` / `compare.go` · *non-profile* = the assertion is about a file, a process or an artifact other than a `Profile`, and its carrier is named.

| Slice | DoD assertion | Carrier after SC | Status |
|---|---|---|---|
| **SC** | all 14 items | this slice **is** the carrier | — |
| **SP1** | `beforeReadFile` with a `SKILL.md` path; per-event base fields | *non-profile*: `tests/fixtures/cursor-hooks/session-01.jsonl`; answer → `.scuba/…/spike-r-rl-17.md` | ok |
| **S0** 1 | every `cursor.*` literal in one file | *non-profile*: `profiler/cursor_wirekeys.go` | ok |
| **S0** 2 | `skill_activation: present`, non-empty name, source, first-party trigger | `ActivationResult.State`, `ActivationEntry.SkillName`, `.Source` **NEW**, `.Trigger` *contract* | ok |
| **S0** 3 | `tokens` present from `cursor.token.type` | `TokenCounts.Input/.Output/.CacheRead/.CacheCreation` *contract* | ok |
| **S0** 4 | one row per (name, status) with count, tri-state status, empty timestamp, aggregate source | `ToolCallEntry.Name` *contract*, `.Status` **NEW**, `.Count` **NEW**, `.Timestamp` *contract*, `RawMetricResult.Source` + `SourceOtelAggregate` **NEW** | ok |
| **S0** 4′ | *implicit:* these `Count` rows must compare correctly against per-call rows | `countToolCalls` sums `Count` **NEW (V2-#1)** — previously `len(calls)`, `compare.go:148-149` | **fixed** |
| **S0** 5 | `tokens: none` **with a reason** | `CapabilityReport.Capabilities` *contract* + `.Notes` **NEW** | ok |
| **S0** 6 | adapter capability table updated | *non-profile*: `docs/profiler-spec.md:135-139` | ok |
| **S1a** 1–5 | 21 events → one spool line each; per-event base fields; exit 0; bounded read; no `require` | *non-profile*: the spool line format in `profiler/hooks.go`, process exit code, `go.mod` | ok |
| **S1b** 1 | `tool_calls`+`timing` present, `Source: hooks` | `ToolCallResult`, `TimingResult`, `RawMetricResult.Source` *contract* | ok |
| **S1b** 2 | `conversation_id` → `generation_id` → `tool_use_id`, subagents branch | `Profile.SessionID` *contract* + `ToolCallEntry.GenerationID`/`.ID`/`.ParentID` **NEW (V2-#2)** | **fixed** |
| **S1b** 3 | reader-side session close | `TimingData.EndTime` *contract* | ok |
| **S1b** 4 | `snapshot_hash` survives round-trip | `Profile.SnapshotHash` *contract* | ok |
| **S1b** 5 | `--session` selects one conversation | `Profile.SessionID` *contract* | ok |
| **S1b** 6 | all-`unknown`, exit 0 | `MetricUnknown` + `RawMetricResult.Reason` *contract* | ok |
| **S1b** 7 | enrichment seam | *non-profile*: `profiler/enrich.go` + two stubs | ok |
| **S2** 1–4 | `hooks.json` byte-identity, backup, idempotence, `doctor` | *non-profile*: filesystem + stdout | ok |
| **S3** 1 | `classifyInvocation` → 5 categories with name **and detail** | *non-profile*: the Go function's return value, asserted in `attribution_test.go`. **`detail` is not serialized** — `Attribution` carries only `Target` + `SkillName`. Scope stated explicitly so no reader assumes a profile field | **clarified** |
| **S3** 2 | activation present, non-empty name, `hooks`, `inferred_file_read` | `ActivationEntry.SkillName`/`.Trigger` *contract*, `RawMetricResult.Source` *contract* | ok |
| **S3** 3 | `attribution: present`, ≥1 `tool_use_id` mapped to a skill | `AttributionData.Attributions[].Target` *contract* — **but it dangled**; now `ToolCallEntry.ID` **NEW** gives it a referent, plus SC's referential-integrity test **(V2-#2)** | **fixed** |
| **S3** 4 | inferred-only trigger vocabulary | `ActivationEntry.Trigger` *contract* + D6 | ok |
| **S4** | secrets absent, prompt length only, 8192 boundary | *non-profile*: the written spool file | ok |
| **S5** r1 | `chars/4` → `estimated` | `RawMetricResult.Source` + `SourceEstimated` **NEW** | ok |
| **S5** r2 | MCP self-report → `mcp_reported` | `SourceMCPReported` **NEW** | ok |
| **S5** r3 | `preCompact.context_tokens` **+ `context_window_size` + `context_usage_percent`** | `TokenCounts.ContextTokens`/`.ContextWindowSize`/`.ContextUsagePercent` **NEW (V2-#4)** — `TokenCounts` (`types.go:80-86`) had no field for any of the three and S5 does not own `types.go` | **fixed** |
| **S5** r4 | billed → `server_api`/`otel` | `RawMetricResult.Source` *contract* | ok |
| **S5** | honesty class per row; `compare` refuses cross-class | SC's class function + `MetricComparison.Comparable`/`.Reason`/`.BaselineSource`/`.CandidateSource` **NEW** | ok |
| **S6** 1 | scoped tokens differ from unscoped | `TokenResult.Value` *contract* | ok |
| **S6** 2 | aggregate fallback labelled | `SourceOtelAggregate` **NEW** | ok |
| **S6** 3 | new wire keys | *non-profile*: `profiler/cursor_wirekeys.go` — **added to S6's Files (C2-#9)** | **fixed** |
| **S7** 1 | two sources for one metric; `Capabilities` names one | `CapabilityReport.Available` **NEW** + `.Capabilities` *contract* | ok |
| **S7** 2 | `HookSpool` on `Condition`/`ExperimentStep` | `profiler/experiment.go`, a file **S7 owns** | ok |
| **S7** 3 | `comparable:true` with both sources | `MetricComparison.BaselineSource`/`.CandidateSource` **NEW** | ok |
| **S8a** 1–4 | four rules fail CI on a seeded violation | *non-profile*: CI exit codes; rule 2's mechanism now named as a `go/ast` walk **(V2-#12)** | **fixed** |
| **S8b** 1–5 | labelled doc claims; scope amendment | *non-profile*: `docs/cursor-hooks-spec.md`, `.out-of-scope.md`, `README.md` | ok |
| **S9** | `server_api` + four-way splits; missing key degrades | `RawMetricResult.Source` *contract*, `TokenCounts.*` *contract* with the `cacheWriteTokens→CacheCreation` mapping stated, `MetricUnknown`+`Reason` *contract* | ok |
| **S10** 1 | correctly nested conversation → generation → tool spans | `Profile.SessionID` *contract* + `ToolCallEntry.GenerationID`/`.ID`/`.ParentID` **NEW** — the nesting was unrepresentable before | **fixed** |
| **S10** 2–4 | no `cursor.*` in the exporter; zero requires; no estimate in `gen_ai.usage.*` | *non-profile*: exporter source, `go.mod`; the estimate check binds to SC's honesty class | ok |
| **S11** 1–3 | byte-equivalent spool; 4xx on malformed; no per-session state | *non-profile*: two spool files diffed | ok |
| **T1** | `summary.token_cost` keys; Context budget dimension | *non-profile*: `audit.json` (skill-audit's artifact, not a `Profile`) | ok |

**Assertions struck for lack of a carrier:** one — R-CS-14's line-edit stats, already dropped by **D8** (no `Profile` field, and SC stays additive-only for what this epic needs). Everything else was given a carrier rather than removed.

**Two invariants added to SC as a direct result of the sweep:**
- **Encoding invariance (V2-#1):** two profiles of one session yield **zero** delta on `count`/`successes`/`success_rate` regardless of row encoding. Without it S0's aggregate rows vs S1b's per-call rows for the same 42 calls yield `comparable:true` with a count delta of 40 — S7's DoD 3 satisfied while wrong.
- **Fail-closed source classing (V2-#5):** `RawMetricResult.Source` is a free `string` (`types.go:23`), so an unknown or legacy value maps to class `unrecognized` and `compare` refuses it. Fail-open would have reinstated round 1's V-5.

---

## 2. Citation table — every enum literal and every Specification / Repository-fact label (root B)

Wire Reference = <https://cursor.com/docs/enterprise/opentelemetry-export/wire>, transcribed into the ledger at the lines cited. cursorscope clone = `cursor-profiler/tmp/research/cursorscope/` (a **third-party** tree → **External evidence** per `AGENTS.md:20`, *not* a Repository fact per `:21`).

### 2.1 Enum value sets

| Enum | Values, verbatim | Citation | Round-1 state |
|---|---|---|---|
| `cursor.token.type` | `input \| output \| cache_read \| cache_creation` | ledger §A5.2:158 | correct |
| `cursor.tool.kind` | `builtin \| mcp` | ledger §A5.2:159 | correct |
| **`cursor.tool.status`** | **`success \| failure \| aborted`** | ledger §A5.2:159 | **`error` was invented (C2-#1) — fixed in `SC.status.md` DoD 2** |
| `cursor.api.status` | `success \| errored \| aborted` | ledger §A5.2:158 | correct (unused by us) |
| `cursor.skill.trigger` | `agent_read \| manually_attached \| skill_name_in_prompt` | ledger §A5.2:169 | correct |
| `cursor.skill.source` | `unspecified \| workspace \| user \| builtin \| plugin \| claude` | ledger §A5.2:169 | correct |
| `cursor.hook.type` | `pre_tool_use \| post_tool_use \| post_tool_use_failure \| before_submit_prompt \| after_agent_response \| after_agent_thought \| stop \| subagent_start \| subagent_stop` | ledger §A5.2:170 | correct (not consumed — S0 stops treating this event as a tool call) |
| `cursor.hook.outcome` | `success \| blocked \| failed \| timeout` | ledger §A5.2:170 | correct |
| Hook event set (21) | `sessionStart, sessionEnd, beforeSubmitPrompt, preToolUse, postToolUse, postToolUseFailure, beforeShellExecution, afterShellExecution, beforeMCPExecution, afterMCPExecution, beforeReadFile, afterFileEdit, beforeTabFileRead, afterTabFileEdit, subagentStart, subagentStop, afterAgentResponse, afterAgentThought, stop, preCompact, workspaceOpen` | ledger §A5.1:117-137 (21 table rows) | correct |
| Base fields (**10**, not 8) | `conversation_id, generation_id, model, model_id (opt), model_params (opt), hook_event_name, cursor_version, workspace_roots, user_email, transcript_path` | ledger §A5.1:113 | **wrong (C2-#3): roadmap presented 8 as "the base set" and asserted all 21 events carry them. `workspaceOpen` supplies only `hook_event_name, cursor_version, workspace_roots, user_email` (§A5.1:137). Fixed in R-CS-02, the S1a row, `S1a.status.md` DoD 1 and `SP1.status.md`** |
| `classifyInvocation` categories | `mcp \| cli \| skill \| subagent \| tool` | cursorscope `src/attribution.js:174,183,193,226,234,242` (the six literal `category:` assignments, five distinct values) | correct; citation added |
| `SKILL_PATH_RE` | `/(?:^\|[/\\])skills[/\\]\|SKILL\.md$/i` | cursorscope `src/attribution.js:1` | correct; citation added |
| Admin API `tokenUsage` | `inputTokens, outputTokens, cacheWriteTokens, cacheReadTokens` (+ `totalCents`, `discountPercentOff`) | ledger §A5.3:180 | correct; the `cacheWriteTokens→CacheCreation` mapping is now stated in S9 |
| `gen_ai.usage.*` | `input_tokens, output_tokens, cache_read.input_tokens, cache_write.input_tokens, reasoning.output_tokens` | ledger §A6:210 | correct |
| `MetricState` | `present \| unknown \| error` | `profiler/types.go:11-15` | correct |
| `MetricSource` (existing 6) | `otel \| hooks \| session_data \| server_api \| sqlite \| none` | `profiler/types.go:40-47` | correct — but `none` was unmapped by D3 (C2-#2); now class `none` |
| `MetricSource` (new 3) | `estimated \| otel_aggregate \| mcp_reported` | **Design decision** — ledger §A5.1:145 and §A4:300 *recommend* `SourceEstimated`; the other two are ours (D3) | label now stated as Design decision |
| Honesty classes | `measured_billed \| harness_reported \| self_reported \| estimated \| none \| unrecognized` | **Design decision** D3 | totalised + fail-closed (C2-#2, V2-#5) |
| Inferred triggers | `inferred_file_read \| inferred_shell_command \| inferred_subagent` | **Design decision** D6 | correct; label restated in S3 DoD 4 |

**0 invented values remain.** One was invented in round 1 (`error`), by C-13's own fix — the exact regression both hunters flagged.

### 2.2 Evidence labels re-derived

| Where | Round-1 label | Round-2 label + citation | Finding |
|---|---|---|---|
| R-CS-02 | Specification (8 fields) | **Specification** — ledger §A5.1:113 (ten fields), §A5.1:137 (the `workspaceOpen` exception) | C2-#3 |
| R-CS-08 | External evidence | **External evidence** — cursorscope `src/attribution.js:174,183,193,226,234,242` (line cite added) | — |
| R-CS-11 | Repository fact (ledger §A4 calibration) | **split:** mechanism = **External evidence** (cursorscope `src/telemetry.js:recordAttributedContextTokens`, ledger §A5.1:145); calibration = **Repository fact** (ledger §A4:83, measured) | C2-#6 |
| R-CS-13 | Specification | **Specification** — ledger §A5.1:136 (line cite added) | — |
| R-CS-20 | External evidence | **External evidence** — cursorscope `src/privacy.js` (source named) | — |
| R-CS-30 §G ref | "§G6" | **§G7** — §G6 is the Admin API item; the latency item is §G7 (numbering shifted when §G3 was inserted). Fixed at R-CS-30 and in the S1a table row | C2-#5 |
| R-RL-05 | Specification | **split:** the six-value enum = **Specification** (ledger §A5.2:169); *adding the field additively with no schema bump* = **Design decision** (D1) | C2-#6 |
| R-RL-08 | Specification + **Repository fact** (`src/cursor-api-poller.js:44,47`) | **split:** documented endpoint shape = **Specification** (ledger §A5.3:179,186); the cursorscope defect = **External evidence** — that file lives in the cursorscope clone, a third-party tree (`AGENTS.md:20` vs `:21`). Verified in place: `src/cursor-api-poller.js:44` is `` `/teams/${teamId}/daily-usage-data` `` and `:47` is `` Authorization: `Bearer ${adminApiKey}` `` | C2-#7 |
| R-RL-13 | **Specification** (semconv 1.44.0) | **split:** *semconv defines no skill/rule convention; `gen_ai.usage.*` names are as listed* = **Specification** (ledger §A6:210,212); *namespace ours `skill_architect.skill.*`* = **Design decision** — ledger §A6:214 calls it a "Design decision recommendation" and nothing specifies it | C2-#6 |
| R-RL-15 | Design decision over a Local hypothesis | unchanged; citation added (ledger §C3:308 — "semgrep is not installed on this machine") | — |
| R-RL-11 | mixed | unchanged; citations added (ledger §B1:228 for 1,536 chars, :230 for 500 lines) | — |
| S5's source→slice map | "billed → `server_api`/`otel` (S6/S9)" | **inverted → (S9/S6)**: S9 is the Admin API (`server_api`), S6 is Enterprise OTel (`otel`). `S5.status.md` had it right; the roadmap row did not | C2-#10 |
| SP1 sizing | "a ~40-line PR" | **removed** — its DoD commits a redacted session capture, a test, and a control-plane answer document. Re-described as a fixture-bearing PR | C2-#12 |
| S3 goal | "the first activation that names a real skill, and the first from a non-Enterprise surface" | **"the first from a non-Enterprise surface"** only — S0 merges in wave 2 and emits `present` with a non-empty `cursor.skill.name` before S3 exists | C2-#8 |
| revision-log-round1:71 | "26 edges" | **28** — corrected in place with a round-2 marker | C2-#11 / V2-#18 |
| D10 (new) | absent | **External evidence** — `cursor-profiler/docs/research/existing-per-skill-cost-tools.md:53,55,119-127,129-131` | C2-#4 |

---

## 3. Recomputed dependency table (roots B, C)

Each row derived from that slice's **DoD + Files**, not from the previous graph. "Why" names the specific artifact the dependency supplies.

| Slice | Depends on | Why (artifact-level) | Change |
|---|---|---|---|
| SC | none | — | — |
| SP1 | none | needs no code from us | — |
| S0 | SC | `ActivationEntry.Source`, `ToolCallEntry.Status`/`Count`, `SourceOtelAggregate`, `CapabilityReport.Notes` | — |
| S1a | none | — | — |
| S1b | SC, S1a | SC: `ToolCallEntry.ID`/`GenerationID`/`ParentID` (DoD 2). S1a: the spool the reader reads | — |
| S2 | S1a | the hook binary's path + flags, to write into `hooks.json` | — |
| S3 | SC, S1b · **gate SP1=yes** | SC: `ToolCallEntry.ID` for `Attribution.Target`. S1b: the profile + the `attribution.go` stub | — |
| S4 | S1a | the spool write path it inserts redaction into | — |
| S5 | SC, S1b | SC: the three `TokenCounts` context fields + `SourceEstimated`/`SourceMCPReported`. S1b: the `tokens.go` stub | — |
| S6 | SC, S0 | SC: `SourceOtelAggregate`. S0: `cursor_wirekeys.go`, which S6 extends | — |
| S7 | SC, S0, S1b, S3, S4, S5 | all six produce something S7 reconciles | — |
| **S8a** | **SC, S0, S4** | SC: the `estimated` constant rule 2 binds to. **S0: `cursor_wirekeys.go` — `cursor.go:175,182,184,186,197,224,242,262` holds 8 live `cursor.` literals, so rule 4 is red until S0 merges.** S4: the redaction path rule 3 points at | **+S0 (V2-#6); S3 dropped (V2-#7)** |
| **S8b** | **SC, S1b, S4 · +S3 iff S3 ships** | docs describe what merged: the contract, the capture path, the privacy behaviour, and attribution *if it exists* | **new split (V2-#7)** |
| S9 | S1b, S5 | S1b: the hook-sourced profile to join onto. S5: the token plumbing + honesty classing | — |
| S10 | S1b, S2, S5 | S1b: `ToolCallEntry.ID`/`GenerationID`/`ParentID` for span nesting. S2: `profiler/cmd/skill-scope/`. S5: the estimate-rejection rule | — |
| S11 | S1a, S2 | S1a: the 21 fixtures **and** the spool writer it calls. S2: `profiler/cmd/skill-scope/` | **DoD rebound from SP1's fixtures to S1a's (V2-#9)** |
| T1 | none | — | — |

**Graph:** 32 edges, acyclic; in-degrees sum to 32 and match the table in both directions.
**Primary MVE** {SC, S0, S1a, S1b, S2, S3, S4, S8a, S8b} — closes.
**Fallback MVE (SP1 = no)** {SC, S0, S1a, S1b, S2, S4, S8a, S8b} — **closes**: S8a needs {SC, S0, S4} ⊂ set; S8b needs {SC, S1b, S4} ⊂ set. Round 1's did not (V2-#7).
**Fork (vi) option-1 flip** no longer deletes a constant an MVE slice needs (V2-#8): S5 under option (1) *stops producing* estimates; `SourceEstimated` is retained because S8a's guardrail 2 and D3's total class mapping both bind to it.

**Ownership map (§F), rebuilt from all 17 `Files` sections.** Every same-wave shared file: **`profiler/cursor.go`, S0 ‖ S1b in wave 2** — and that is the only one. Round 1 claimed `hooks.go`'s writers "land in different waves except S1b ‖ S4"; in fact wave 3 ran S3 ‖ S5, both listing it. Fixed structurally by S1b's seam, not by a note. Also newly listed: `docs/profiler-spec.md` (SC w1, S0 w2, S5 w3 — three sections, three waves), `profiler/cmd/main.go` (S1b w2, S7 w5), `.github/workflows/ci.yml` (S1a w1, S8a w3), and `profiler/cursor_wirekeys.go` (S0 creates, S6 extends). `profiler/hooks.go` is removed from S3's, S5's and S11's `Files`.

---

## 4. Per-finding disposition

### Verifiability lens (V2-1…19)

| # | Verdict | Disposition |
|---|---|---|
| V2-1 | **REAL, reproduced** | SC DoD 9: `countToolCalls` sums `Count` (absent/zero = 1) and successes sum `Count` over `Status=="success"`. New **encoding-invariance** test: 42 per-call rows vs aggregate rows → zero delta. S7's DoD 3 now exercises the mixed-encoding case too, so it can no longer be satisfied while wrong |
| V2-2 | **REAL** | SC DoD 3 adds `ToolCallEntry.ID`/`GenerationID`/`ParentID`; S1b DoD 2 asserts the chain and the subagent branch against them; S3 DoD 3 asserts `Attribution.Target` **resolves to** a `ToolCallEntry.ID`, with a referential-integrity test in SC |
| V2-3 | **REAL** | Resolved in favour of **`omitempty` for every new field**, declared after existing fields (SC DoD 12). Byte-identity is kept and is now *derivable* rather than asserted; D1 states the reasoning |
| V2-4 | **REAL** | SC DoD 4 adds `TokenCounts.ContextTokens`/`.ContextWindowSize`/`.ContextUsagePercent`; S5's source table names them as the carrier row by row |
| V2-5 | **SUSPECTED → accepted as REAL** | D3 is now **fail-closed**: any `Source` string outside the nine constants maps to class `unrecognized` and `compare` refuses it, naming the string. SC's test includes `compareTokens("wat", hooks)` |
| V2-6 | **REAL** | **S0 added to S8a's deps**, with the eight live `cursor.` literal sites cited in the slice file |
| V2-7 | **REAL** | **S8 split into S8a + S8b.** S8a (guardrails) drops S3; S8b (docs) keeps it, conditionally. Both MVEs close; the fallback MVE's closure is now stated arithmetically in the roadmap |
| V2-8 | **REAL** | Fork (vi)'s "what flips it" and `S5.status.md` §Next rewritten: option (1) **retains** `SourceEstimated` and removes only the producer. SC's §Next says the same |
| V2-9 | **REAL — PARTIAL DISAGREEMENT, argued** | The invariant (deps ⊇ what the DoD requires) is real. I closed it the other way: S11's DoD now binds to **S1a's 21 fixtures**, which it already depends on. Evidence: byte-equivalence is a property of the two write paths, not of a fixture's provenance — the same diff proves it with any input. Adding SP1 would put a conditional, recommend-never-ships slice behind a hardware-gated spike it does not need. SP1's real capture is noted as an *upgrade*, not a dependency |
| V2-10 | **REAL** | §F ownership map rebuilt from all 17 `Files` sections (see §3). The S3 ‖ S5 collision is **removed** by S1b's enrichment seam, not documented; `docs/profiler-spec.md`, `cmd/main.go`, `ci.yml` and `cursor_wirekeys.go` are now listed; `hooks.go` removed from S3/S5/S11 |
| V2-11 | **REAL** | S9, S10, S11 and T1 all gained `## Files`; T1 gained `## Requirements` (R-RL-09, R-RL-10 fallback half, R-RL-11, R-SA-10, R-SA-02) — closing the last matrix gap |
| V2-12 | **SUSPECTED → accepted** | S8a rule 2's mechanism named: a **`go/ast` walk** in `profiler/guardrails_test.go` over every `FuncDecl` matching `(?i)estimate`, flagging a `MetricPresent` result whose `Source` ≠ `SourceEstimated`, with a seeded `estimateBogus` fixture. A sanctioned fallback (drop to a documented manual check in S8b) is stated; an unnamed mechanism is not acceptable |
| V2-13 | **REAL** | S11's test file named: `profiler/cmd/skill-scope/serve_test.go` |
| V2-14 | **REAL** | S1a DoD 3 added: stdin read through a bounded reader; a 10 MB payload exits 0 and writes one capped line. R-CS-30 now reads "both halves" in S1a's Requirements |
| V2-15 | **REAL** | S11's TTL sweep **deleted**. The server holds no per-session state — which is *why* byte-equivalence holds. R-CS-06's adapted form stays S1b's; the original stays deleted by Fork (i). DoD and Requirements now agree |
| V2-16 | **REAL** | SC DoD 10: with `Status` set, the report carries `successes`/`failures`/`aborted` and `success_rate = successes/(successes+failures)`; with `Status` empty, today's behaviour is preserved exactly, so the 34 tests do not move |
| V2-17 | **REAL** | SP1's `Files` split into *branch* (fixture + test) and *control plane* (`.scuba/…/spike-r-rl-17.md`, **not** committed to `epic/skill-scope`). §F states the rule |
| V2-18 | **REAL** | `revision-log-round1.md:71` corrected in place: 28 edges, not 26 |
| V2-19 | **REAL** | R-SA-14's Slice column is now `S0, S8a (D5), S10`; S10's DoD 2 states it is S10's half of the requirement |

### Conformance lens (C2-1…12)

| # | Verdict | Disposition |
|---|---|---|
| C2-1 | **REAL, HIGH — my regression** | `SC.status.md` DoD 2 now reads `success \| failure \| aborted`, cited to ledger §A5.2:159, with the rule stated in the file: *a pass-through field's vocabulary is the source's vocabulary, verbatim.* S0's DoD 4 repeats the three values with the same citation. `error` appears nowhere in the 17 files |
| C2-2 | **REAL** | D3 totalised: `SourceNone` → class `none`, and a `present` result carrying it is a contract violation SC's test catches. All **nine** constants are now mapped, so SC's own day-one test goes green |
| C2-3 | **REAL** | Base fields corrected to **10** with the two `(opt)` members marked, and `workspaceOpen`'s four-field shape called out as its own case. Fixed in R-CS-02, the S1a roadmap row, `S1a.status.md` DoD 1 and its test table, and `SP1.status.md` |
| C2-4 | **REAL — Design-decision gap** | **D10 added.** Decision: **build; treat Dash0 as a reference implementation, not a fork base.** Reasons cited to `existing-per-skill-cost-tools.md:53,55,126,127`: 9 of 21 hook events, per-turn `chat default` span with **no skill span**, no span for subagent work, billed-vs-estimated unstated, Dash0-backend-oriented, 0.1.x. `o11y-dev/opentelemetry-hooks` excluded as Python 3.12+ per `AGENTS.md:6` (:129-131), retained as a design reference whose pass-through-only token posture D3 mirrors. **What flips it** is stated. R-SA-10's row now names D10 as its discharge; §G16 records that the plugin was read, not run. **Not** deferred to §E — it is an engineering call, not the user's; the only user-shaped edge (Q5 answering "Dash0") is noted inside D10 and in Q5 |
| C2-5 | **REAL** | `§G6` → `§G7` at R-CS-30 and in the S1a row. §G6 (Admin API) references in `S9.status.md` were already correct and are unchanged |
| C2-6 | **REAL, three instances** | R-RL-13, R-CS-11 and R-RL-05 all re-labelled as splits with line-level citations (see §2.2). The general rule is now written into the log's method: a label is derived from the cited line at the moment of writing, never from the row's topic |
| C2-7 | **REAL** | R-RL-08 split: endpoint shape **Specification**; the cursorscope poller defect **External evidence**. I re-opened `src/cursor-api-poller.js` in the clone and confirmed both lines before relabelling |
| C2-8 | **REAL** | S3's goal and the roadmap row now claim only **"first from a non-Enterprise surface"**, with the reason (S0 ships first-party `present` in wave 2) written into the slice file |
| C2-9 | **REAL, both halves** | `profiler/cursor_wirekeys.go` added to S6's `Files` and to its DoD as item 3. **D5 widened to name `*_test.go` explicitly**, so S8a's guardrail-4 allowance and D5 now say the same thing — round 1 widened the guardrail without widening the decision |
| C2-10 | **REAL** | Roadmap's S5 row corrected to `server_api` (**S9**) / `otel` (**S6**); `S5.status.md`'s table carries the correction and says which way round 1 had it |
| C2-11 | **REAL** | Corrected in `revision-log-round1.md:71` in place, marked as a round-2 correction |
| C2-12 | **REAL** | "~40-line PR" removed; SP1 re-described as a fixture-bearing PR in both the roadmap MVE line and `SP1.status.md` §Next |
| C2 — INVALID | **agreed** | `gen-ai-semconv.test.js` as an S3 oracle stands; the hunter withdrew it themselves (`:92` does assert CLI/skill invocation labeling). No change |
| C2 — out-of-lens | **REAL, adopted** | Same as V2-#7; fixed by the S8a/S8b split |

---

## 5. Rejected / partial disagreement

**Rejected outright: none.**

**One partial disagreement, argued: V2-#9 (S11 ← SP1).** The finding is real — S11's DoD required an artifact its dependency list did not declare. I closed it by rebinding the DoD to S1a's 21 fixtures instead of adding SP1 as a dependency, because byte-equivalence between the HTTP path and the stdin path is a property of the two writers and is proven by the same `diff` with any input; SP1's provenance adds nothing to that proof, and making a recommend-never-ships conditional slice wait on a hardware-gated spike is invented sequencing of exactly the kind round 1 was told to remove. SP1's real capture is recorded as an upgrade case. If the hunter still prefers the dependency, adding it costs nothing structurally — the slice is already outside both MVEs.

**One judgment call worth flagging: S1b's enrichment seam.** It lands two no-op stub functions that do nothing until S3 and S5 fill them — a small amount of inert code, which the slicing discipline normally resists. Taken because it converts a real same-wave, same-file collision between two wave-3 slices into zero contention, for six lines. Named here so a reviewer can overrule it.

**Both hunters' "clean" lists are preserved intact:** all 22 `cursor.*` names, the 21-event hook surface and its per-event payload fields, `afterShellExecution` without `exit_code`, `preCompact`'s field set, `SKILL_PATH_RE`, the Admin API verb/path/auth/`conversationId`/`tokenUsage` shape and the 60-per-minute figure with its re-verify instruction, every Fork (v) measurement, the ACES figures and calibration range, `AGENTS.md`'s conventions and line numbers, the 131/34 test counts, the 65-ID requirement matrix (0 orphans, 0 phantoms), and an acyclic graph.
