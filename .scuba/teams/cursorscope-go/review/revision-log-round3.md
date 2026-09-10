# Revision log — E-CS roadmap, round 3

**Artifact:** `roadmap.md` (335 lines) + **17** slice files · **Date:** 2026-09-10 · **Groomer:** cursorscope-go
**Input:** `hunter-verifiability-round3.md` (5 HIGH + 7 MED + 4 LOW = **V3-1…16**) + `hunter-conformance-round3.md` (12 REAL + 1 SUSPECTED = **C3-1…12 + S**) = **29 dispositions**
**Method:** both hunters named the same remaining root. Round 2's representability table proved every DoD assertion had a contract **field**; it never asked whether a hook event **supplies the value**. §1 below is the missing table: for every assertion claiming a value is `present` from hooks, the exact event and payload field that supplies it, cited to a ledger line — or **NONE**, with the consequence applied. §2 records the token-source decision. §3 is the per-finding disposition. §4 re-derives every citation both hunters challenged.

**Structural changes:**
1. **D11 added** — the four values the hook surface does not supply (timestamps, tool-call outcome, identifier kinds, the attribution join), each derived once under a stated caveat instead of four times implicitly. Two caveats are now doc-visible (spec doc + S8b) and one is now **test-asserted** (`TotalMs` vs `sessionEnd.duration_ms`).
2. **D12 added** — one `TokenResult` has one `Source`, so `preCompact`'s occupancy fields leave `TokenCounts` for a new **`context_window`** metric, and `tokens` carries the single best-honesty flow source chosen by a named total order. SC's diff to `TokenCounts` drops to **zero**.
3. **D13 added** — `skill-architect/comparison/v1` (`compare.go:11`) versions independently and stays `v1` under the same additive rule as D1, proven by a pinned golden report.
4. **`Status` is now always set on the hooks path** (S1b), and **`successes` is defined exactly once** (SC), which collapses the 0.75-vs-0.50 divergence at its root rather than in the comparator.
5. **SP1 widened from one question to four.** D11 makes three decisions from documentation alone; the spike that is already happening can settle all three for free. Slice count stays **17**; edges go **32 → 33** (S4 → S11).

**Line budget:** the roadmap is **335** lines against a ~330 target. Every added line is D11 (5), D12/D13 (2), the new §A root row (1), the `test_guardrails.sh` ownership row (1) and the S4→S11 edge (1). No line was added for prose.

---

## 1. Source-supply table — every "present from hooks" assertion → the event.field that supplies it (root A)

Citations are ledger line numbers (`.scuba/teams/research-profiling/ledger.md`). §A5.1:113 is the ten base fields; :117-137 is the 21-row per-event payload table. **SUPPLIED** = a documented field carries it. **DERIVED** = no field carries it; a named Design decision produces it, with a caveat. **NONE** = the assertion is removed or downgraded.

| # | Asserted by | Value | Hook event | Payload field | Cite | Verdict |
|---|---|---|---|---|---|---|
| 1 | S1b·1 | `Profile.SessionID` | all agent hooks | `conversation_id` | :113 | **SUPPLIED** |
| 2 | S1b·1 | `ToolCallEntry.GenerationID` | all agent hooks | `generation_id` | :113 | **SUPPLIED** |
| 3 | S1b·1 | `ToolCallEntry.Name` | `preToolUse`/`postToolUse`/`postToolUseFailure` | `tool_name` | :120,121,122 | **SUPPLIED** |
| 4 | S1b·1, SC·3 | `ToolCallEntry.ID` | `preToolUse`/`postToolUse`/`postToolUseFailure` | `tool_use_id` | :120,121,122 | **SUPPLIED** |
| 5 | S1b·3, SC·3 | `ToolCallEntry.ParentID` | `subagentStart` | **`tool_call_id`** — *not* `tool_use_id` | :131 | **SUPPLIED**; spelling pinned per event (D11.3) |
| 6 | S1b·3 | subagent → parent stitch | `subagentStart` | `parent_conversation_id` | :131 | **SUPPLIED as a reader-side join key only.** It is a *conversation* id and is **barred from `ParentID`**, which holds one kind |
| 7 | S1b·1 | `ToolCallEntry.Timestamp` | — | — | :113 (no timestamp among the ten); :117-137 (none per-event) | **NONE → DERIVED.** Hook-stamped `received_at` (D11.1) |
| 8 | S1b·4 | `TimingData.StartTime`/`.EndTime`/`.TotalMs` | — | — | same | **NONE → DERIVED.** Hook-stamped; `Source: hooks` **with the caveat stated in the DoD and the spec doc** |
| 9 | S1b·4 | cross-check on `TotalMs` | `sessionEnd` | `duration_ms` | :118 | **SUPPLIED — used as a test-time cross-check only**, never as a second provenance inside one `TimingResult` |
| 10 | S1b·2 | `Status = "success"` | `postToolUse` | the event's occurrence for a `tool_use_id` | :121 | **SUPPLIED** |
| 11 | S1b·2 | `Status = "aborted"` | `postToolUseFailure` | **`is_interrupt` == true** | :122 | **SUPPLIED** |
| 12 | S1b·2 | `Status = "failure"` | `postToolUseFailure` | `is_interrupt` false/absent (with `error_message`, `failure_type`) | :122 | **SUPPLIED** |
| 13 | S1b·2 | `Status` for an orphan `preToolUse` | — | — | — | **NONE → DERIVED `aborted`** (D11.2). Rate-neutral: `success_rate` excludes aborted from its denominator |
| 14 | S1b (round 2) | `Status`/`Success` from `afterShellExecution` | `afterShellExecution` | `command`, `output`, `duration`, `sandbox` — **no outcome among them** | :124 | **NONE → ASSERTION REMOVED** (V3-#4) |
| 15 | R-CS-15 (round 2) | `Success` from `exit_code` | — | undocumented | :124 | **NONE → demoted to a cross-check** SP1 may reinstate |
| 16 | S1b·1 | `tokens: present, Source: hooks` | — | — | Q1:300 *"No hook reports input/output/cache-read/cache-creation for a model call"* | **NONE → `unknown`-with-reason.** `hooks` is not a token-flow label (D12) |
| 17 | S3·2 | `ActivationEntry.SkillName` | `beforeReadFile` | `file_path` → `skillNameFromPath` | :127, :143 | **SUPPLIED *if* SP1 = yes** — the mechanism itself is a Local hypothesis (:143) |
| 18 | S3·2 | `ActivationEntry.Timestamp` | — | — | :113 | **NONE → DERIVED** (D11.1) |
| 19 | S3·3 | `Attribution.Target` → a `ToolCallEntry.ID` | `beforeReadFile` | **no `tool_use_id`** (`file_path`, `content`, `attachments` only) | :127 | **NONE from the inferring event → join specified (D11.4).** Form A: a same-generation `preToolUse`/`postToolUse` whose `tool_input.file_path` matches → its `tool_use_id`. Form B: `file://<path>` output identifier, generation-scoped (`generation_id` is base-supplied, :113). **SP1 decides which** |
| 20 | S3·1 | `classifyInvocation` inputs | `before`/`afterMCPExecution`; `before`/`afterShellExecution`; `subagentStart`; `preToolUse` | `mcp_server_name`, `tool_name`; `command`; `subagent_type`; `tool_input` | :125,126,123,124,131,120 | **SUPPLIED** |
| 21 | S5 | `context_window`: three fields | `preCompact` | `context_tokens`, `context_window_size`, `context_usage_percent` | :136 | **SUPPLIED** — moved out of `TokenCounts` into its own metric (D12) |
| 22 | S5 | `mcp_reported` tokens | `afterMCPExecution` | `result_json` → `usage`/`token_usage`/`tokens` | :126 (field), :142 (the probe) | **SUPPLIED — External evidence**, cursorscope's probe; the docs do not specify a `usage` block |
| 23 | S5 | `estimated` tokens (`chars/4`) | `beforeSubmitPrompt`, `postToolUse`, `afterShellExecution`, `beforeReadFile`, `afterMCPExecution`, `afterAgentResponse`, `afterAgentThought` | `prompt`, `tool_output`, `output`, `content`, `result_json`, `text`, `text` | Q1:300 (the exact list) | **SUPPLIED** |
| 24 | S1a·1 | `workspaceOpen`'s four fields | `workspaceOpen` | `hook_event_name`, `cursor_version`, `workspace_roots`, `user_email` | :137 | **SUPPLIED** |
| 25 | S1b·5 | reader-side session close | `sessionEnd` / `stop` | occurrence, `final_status` / `status` | :118, :135 | **SUPPLIED** for *when* to close; the `EndTime` value itself is row 8's derived stamp |
| 26 | S6·2 | aggregate tokens scoped by session | `cursor.token.usage` metric | attributes are `cursor.token.type`, `.model.name`, `.api.status`, `.api.billable` — **no conversation id** | §A5.2:158 | **NONE → with `--session`, `unknown`-with-reason**; without it, export-scoped `otel_aggregate` (V3-#8) |
| 27 | S6·1 | per-request tokens scoped by session | `cursor.api.request` **log** event | `cursor.conversation.id` is a documented log join key | §A5.2:152,166 | **SUPPLIED** |
| 28 | S9·1 | `server_api` join | Admin API ↔ hooks | `filtered-usage-events.conversationId` (**optional**) ↔ `conversation_id` | §A5.3:180,184; :113 | **SUPPLIED (Specification, unobserved §G6).** A row without the optional id contributes nothing |
| 29 | S0·5 | `cursor.hook.*` corrected keys | — | nothing reads them after S0 DoD 4 | §A5.2:170 | **NONE → NOT CONSUMED.** R-RL-01's hook half resolves to deletion; no constant is added (C3-#11) |

**Result: 4 assertions removed or downgraded (rows 14, 15, 16, 26), 5 derived under D11 (rows 7, 8, 13, 18, 19), 1 requirement half marked not-consumed (row 29).** Everything else is supplied and cited.

---

## 2. The token-source decision (V3-#2)

**The defect.** `RawMetricResult.Source` is a single `string` (`types.go:23`) and `profiler-spec.md:36` requires it populated whenever `State == present`. S5's round-2 DoD table gave four labels to one `TokenCounts`; S9's DoD assigned `Profile.Tokens` directly, as did S6's — three writers, last-write-wins, and the honesty label of whichever ran first silently vanished.

**The menu offered, and why two options were rejected.**

| Option | Verdict |
|---|---|
| (a) `TokenCounts` gains `Sources map[string]MetricSource` | **Rejected.** It pushes provenance below the result, so `compareTokens` — which gates on **one** honesty class (D3) — would need per-field class gating, widening both SC and the comparison contract. It also puts a provenance map inside a value struct whose whole shape is flat counts. |
| (b) A new `TokenEstimates` side field carrying the non-primary results | **Rejected.** `CompareProfiles` would not read it, so it ships as inert data — a horizontal layer that pays off only if a later slice consumes it, which none does. D8's precedent (R-CS-14 dropped for want of a field) cuts the other way here: don't add a field this epic won't read. |
| (c) S5 emits only the single best-honesty source per profile | **Adopted, with one correction.** |

**The correction, and it is a disagreement with the menu as posed.** Option (c) alone is wrong, because `preCompact`'s `context_tokens` is not a *competing measurement of the same quantity*. It is context-window **occupancy at compaction**; `Input`/`Output` are token **flow**. Ranking them against each other means either dropping the occupancy number (it loses to nothing, since `hooks` outranks `estimated`) or emitting a `present` `TokenCounts` whose `Input`/`Output` are zero because the winning source does not supply them — a zero-filled value, which R-SA-02 forbids. **The conflation was created in round 2**, when SC bolted the three `preCompact` fields onto `TokenCounts` to give R-CS-13/R-RL-18 a carrier.

**Decision — D12, two parts:**

1. **Separate the quantities.** `preCompact`'s three fields leave `TokenCounts` for a new `context_window` metric (`ContextData`/`ContextResult`, `Profile.ContextWindow *ContextResult` declared last, `omitempty` — a **pointer**, because `omitempty` is a no-op on a value struct and would have broken SC's byte-identity DoD). Its source is `hooks`, always, and it is the *only* hook-supplied token-ish number. **`TokenCounts` reverts to its shipped shape — SC's diff to it is now zero.**
2. **Rank the flow sources.** `tokens` carries one source, chosen by `selectTokenSource` — defined in S5, the **only** writer of `Profile.Tokens` — over the total order **`server_api` > `otel` > `mcp_reported` > `estimated`** (D3's honesty classes, best first). S6 and S9 *register candidates*; they do not assign. Lower-ranked candidates are **discarded, not merged**, so no field of `TokenCounts` ever comes from a source other than the one named in `Source`. A `present` result requires its source to supply both `Input` and `Output`, else `unknown`-with-reason.

**Consequences applied:** S5 retitled and rewritten; S6 DoD 4 and S9 DoD 3 register rather than assign; SC item 5 adds the carrier and leaves `TokenCounts` alone; S7 asserts the two metrics compare separately; Fork (vi) updated; §A's "four honesty levels" framing survives intact, because ranking preserves the four labels — it just stops them sharing one field. **Also note the honesty gain for Fork (vi) option 1:** if the user rejects estimates, a hooks-only profile is no longer token-silent, because `context_window` still ships `present`.

---

## 3. Per-finding disposition

### Verifiability lens (V3-1…16)

| # | Sev | Disposition |
|---|---|---|
| 1 | HIGH | **Accepted — fixed at the root, not in the comparator.** Two changes: S1b DoD 2 **always sets `Status`** (D11.2), and SC DoD 10 defines `successes` **once** by a row-outcome rule (`Status` when set, else `Success`). The legacy number now *falls out* of the single rule instead of being a second definition — a pre-v1.1 profile has no `aborted` outcome, so `successes + failures == count` and today's rate is reproduced arithmetically. SC's and S1b's fixtures are now **required to be heterogeneous** (3s/1f/2a in both encodings); the hunter's 0.75-vs-0.50 case is the fixture. S7 DoD 4 asserts it cross-adapter. |
| 2 | HIGH | **Accepted.** See §2. D12. |
| 3 | HIGH | **Accepted; the recommended option taken.** No hook event carries a timestamp (`:113`, `:117-137`). S1a's binary stamps `received_at` into the spool envelope, outside `hook`. `TimingResult.Source` stays `hooks` **with the caveat** — *wall-clock at hook receipt, not model time* — stated in S1b DoD 4, in D11.1, in the spec doc and in S8b's doc. Recorded as a **Design decision**. Given teeth by a test: `TotalMs` must be within tolerance of `sessionEnd.duration_ms` when both exist. Per-call `duration` has no carrier and goes to D8's "not in v1.1" list. |
| 4 | HIGH | **Accepted.** `afterShellExecution` is removed as a `Status` source. Carriers are `postToolUse` → success, `postToolUseFailure` + `is_interrupt` → aborted, else failure (`:121,122`). `is_interrupt` is the find that makes the tri-state derivable at all without inventing a fourth value. |
| 5 | HIGH | **Accepted.** `beforeReadFile` has no `tool_use_id` (`:127`). D11.4 specifies Form A (`tool_use_id` from a same-generation `preToolUse`) and Form B (`file://` output identifier, generation-scoped). **SC does not narrow `Attribution.Target`** — see C3-#3. The question is added to **SP1 DoD 2**, which decides which form S3 builds. |
| 6 | MED | **Accepted.** S1a DoD 6 pins the spool envelope: two members (`hook`, `received_at`) and an explicit list of ~25 non-base fields downstream slices may read, each cited to its ledger line, each asserted in the fixture table. "No later slice may rely on a field outside this list without amending this DoD." |
| 7 | MED | **Accepted.** No "adapter capability table" exists; `profiler-spec.md:135-139` is `ActivationEntry` (verified). S0 DoD 7 now targets **a new "Cursor adapter" subsection under §"Adapter implementations" (`:166`)**, mirroring the Claude Code adapter at `:168-190`. Fixed in S0, the §C table and the §F ownership map. |
| 8 | MED | **Accepted.** `cursor.token.usage` has four attributes and none is a conversation id (§A5.2:158). S6 DoD 2 splits the cases: no `--session` → export-scoped `otel_aggregate`; with `--session` → **`unknown` with the reason naming the missing id**. `cursor.model.name` is the only further scoping available and is not used. |
| 9 | MED | **Accepted — the rule narrowed to the mechanism, not the mechanism widened.** Rule 2 now reads "in the body of". The residual gap (a mislabel built in a helper whose name does not match) is stated in the rule, documented in S8b DoD 3, and mitigated by naming convention. A guardrail whose text overreaches its grep reads as proof and is not. |
| 10 | MED | **Accepted.** S11's DoD asserts redaction, so **S4 is added to its `Depends on`**; graph 32 → 33 edges; §F's conditional-wave row updated. |
| 11 | MED | **Accepted, fixed better than by exception.** Rather than allowlisting `serve.go`, S1a pins two names — `WriteHookEvent` (exported, redacting) and `appendSpoolLine` (private, raw) — and guardrail 3 greps for callers of **the raw one**. `serve.go` legitimately calls the redacting entrypoint, so the rule never fires on it, and a future sink that bypasses redaction still trips. An exception list would have had to name a file that may never exist. |
| 12 | MED | **Accepted.** D11.3: `ID` and `ParentID` are both tool-call ids and only that; `subagentStart.tool_call_id` feeds `ParentID`; `parent_conversation_id` is **barred** from it and is a reader-side join key. S1b's test asserts the negative (`parent_conversation_id` appears in no `ToolCallEntry` field). |
| 13 | LOW | **Accepted, all three parts.** `tests/test_guardrails.sh` added to S8b's `## Files` and to §F's ownership map; the fallback is **named** (`tests/test_doc_labels.sh`, a separate file, never both); "claim line" is **defined** in S8b DoD 1 so the grep is writable. |
| 14 | LOW | **Accepted.** `ComparisonSchema` (`compare.go:11`) is real and D1 covered only profiles. **D13** states the rule (same additive discipline, string stays `v1`, `Source` retained beside the new `BaselineSource`/`CandidateSource`) and SC proves it with a pinned pre-change golden report. |
| 15 | LOW | **Accepted.** SC's test approach pins golden provenance: generated once from the **merge-base** `types.go` via `git show <sha>:…`, committed verbatim, with the command and SHA in a header comment. A hand-indented golden proves formatting, not additivity. |
| 16 | LOW | **Accepted.** R-CS-04's Slice column gains **S11** (the original HTTP form, per D9); R-SA-13's gains **S9** (and S6 and S7, which also assert graceful degradation). |

### Conformance lens (C3-1…12 + SUSPECTED)

| # | Sev | Disposition |
|---|---|---|
| 1 | MED | **Accepted — count re-derived by running the rule.** `grep -n 'cursor\.' profiler/cursor.go` → **12** at `:175,182,184,186,197,216,224,234,242,250,262,271`; four are inside `Unknown*Result` reason strings. S0 DoD 1 now covers reason strings explicitly (compose them from the constants), and **runs guardrail 4's own grep as a test** so the 12→0 claim is executed. Fixed in S0, S8a rule 4, S8a's `Depends on`, D5 and the §C table. |
| 2 | MED | **Accepted.** Same root as V3-#1, fixed the same way; S7 DoD 4 rewritten to *assert* the cross-encoding equality rather than claim it holds. |
| 3 | MED | **Accepted, and it is load-bearing.** `types.go:114` and `profiler-spec.md:154` both say "tool call ID **or output identifier**". SC's round-2 narrowing was not merely a doc slip — it made V3-#5 **unsatisfiable**, because the inferring event supplies no tool-call id. SC item 4 keeps both forms under a **disjunctive** referential rule (resolving `ID` ✓, declared-scheme output identifier ✓, bare unresolvable string ✗), which is still mechanically checkable. |
| 4 | MED | **Accepted.** See V3-#12 / D11.3. Both halves fixed: the per-event spelling (`tool_use_id` vs `tool_call_id`) and the one-kind-per-field rule. |
| 5 | LOW-MED | **Accepted, and re-derived from the clone.** Verified: `src/gen-ai-semconv.js:34` — `const TOOL_PAYLOAD_MAX_LEN = Number(process.env.CURSOR_TOOL_PAYLOAD_MAX_LEN || 8192)` — env-overridable; `:223,226` do `slice(0, MAX)` **plus** a `…` marker. `src/privacy.js` has no truncation, and `privacy.test.js`'s six cases cover masking/redaction/recursion only — **no truncation case**. So S4's 8192 assertions are **ours**, not oracle-derived, and S4 now says so; the DoD is restated in **bytes** ("at most 8192 bytes *including* the marker, never splitting a rune") rather than repeating a JS character slice. Fixed in S4, R-CS-20, R-CS-29, §C and §G13. |
| 6 | LOW-MED | **Accepted — and the correction goes further than the finding.** §A5.3:177 states *both* "20–250 req/min by endpoint" *and* "30-day max date range" as properties of **the Admin API as a whole**; :182 adds "20 req/min per team". The ledger pins the 30-day cap to nothing, and **never mentions an `audit-logs` endpoint at all** — round 2 invented both the attribution and the endpoint. R-RL-07 and S9 now say the ledger cannot settle it and that every figure is Specification-unverified pending a live read (§G6). |
| 7 | LOW | **Accepted.** `o11y-dev/opentelemetry-hooks` is Python 3.12+, as D10 itself states. §A's row now reads "**Two hook-based prior-art tools exist — one Go (Dash0), one Python (o11y-dev)**". |
| 8 | LOW | **Accepted.** Ledger `:195` sits under §A5.4's **External evidence** header (`:190`); `:200` is explicitly a **Repository fact** ("Cursor is not installed on this machine"). R-RL-03's label is split accordingly, and S0 DoD 6 states which line supports which claim — the absence of token fields (External) versus why the schema is unverified (Repository fact). |
| 9 | LOW | **Accepted.** Line 300 is under "Decision questions" Q1, not §A4. Corrected **in place** at `revision-log-round2.md:96` (`§A4:300` → `Q1:300`) rather than only here, because §2 of that log is the citation register round 3 builds on; leaving a known-wrong line there guarantees the next hunter re-finds it. |
| 10 | LOW | **Accepted, re-derived from the table.** The measurement table at `:85-90` gives +5.5%, +5.2%, +15.0%, **−6.6%** — so the measured span is **−6.6%…+15.0%**. The rounded phrase "−7%…+15%" is the ledger's own finding sentence at `:92`, repeated at Q1:300. `:83` is the section's label line and measures nothing. §A's thesis now quotes the **measured** span; R-RL-10 and S5 cite `:85-90` for the measurement and `:92` for the rounding; §G8 updated. |
| 11 | LOW | **Accepted.** R-RL-01's `cursor.hook.*` half is marked **not consumed** in the register, with S0 DoD 5 as its carrier: the event leaves the read path entirely, so "correct the keys" resolves to **deletion**, no `cursor.hook.*` constant is added (a constant nothing reads is dead code guardrail 4 would then have to allow), and hook-health telemetry joins D8's "not in v1.1" list. |
| 12 | LOW | **Accepted.** S2 gains **DoD 5** — `--dry-run` prints the merged file and writes nothing; `--yes` skips the prompt — plus a test proving the dry run from both sides (printed body equals what a real install writes; directory byte-identical after). R-CS-25's register row notes it. |
| **S** | SUSPECTED | **Accepted — the hunter is right and the constructors prove it.** `UnknownTokenResult` (`types.go:196`) and its five siblings (`:216,231,246,261` and the error variants) construct `RawMetricResult` without `Source`, so `""` is a zero value the class function genuinely receives. D3 and SC items 8/test-approach now map **`""` → `none`**, deliberately **not** `unrecognized`: an absent provenance is not a typo'd one, and `""` on a *present* result is the same contract violation `none` is. The test table covers `SourceNone`, `""` and `"wat"`. |

### Out-of-lens items both hunters raised

- **`ledger.md:111`** claims cursorscope's registered event list "matches exactly" Cursor's — it registers **19**, missing `beforeTabFileRead` and `workspaceOpen`. R-CS-01 was already correct; the *ledger* is wrong. Recorded as **unowned note (c)** in §F for the manager, not folded into a slice — it is another team's artifact.
- **`S8b:22` edits a file absent from its `## Files`** — same as V3-#13, fixed there.

---

## 4. Citations re-derived (the shared root both hunters named)

Every count below was produced by running the rule that consumes it; every label is the cited line's own label.

| Claim | Round-2 citation | Re-derived | Fixed in |
|---|---|---|---|
| live `cursor.` literals | "8", `cursor.go:175…262` | **12**, `:175,182,184,186,197,216,224,234,242,250,262,271` — by guardrail 4's own grep | S0, S8a, D5, §C |
| 8192 payload cap | `src/privacy.js` | `src/gen-ai-semconv.js:34` (env-overridable); truncation at `:223,226` | S4, R-CS-20 |
| `privacy.test.js` coverage | "our behavioral oracle" for truncation | six cases, **no truncation case** | S4, R-CS-29, §G13 |
| char/4 calibration | "−7%…+15%", §A4:83 | measured **−6.6%…+15.0%** at `:85-90`; the rounded phrase is the ledger's finding line at `:92`, repeated Q1:300 | §A, R-RL-10, S5, §G8 |
| Admin API 30-day cap | "pinned to `daily-usage-data`/`audit-logs`" | §A5.3:177 states it **API-wide**; no `audit-logs` endpoint exists in the ledger | R-RL-07, S9, §G6 |
| R-RL-03 evidence | External evidence, §A5.4:195,200 | `:195` External (§A5.4 header at `:190`); **`:200` Repository fact** | R-RL-03, S0 |
| prior art | "Two Go/hook tools" | one **Go** (Dash0), one **Python** (o11y-dev) | §A |
| `SourceEstimated` recommendation | "§A5.1:145 and §A4:300" | `:145` and **Q1:300** — line 300 is a Decision question, not §A4 | `revision-log-round2.md:96` (corrected in place) |
| `skillscore` note | "ledger §A4 note" | `§A4:97` | §F note (b) |
| `skillListingBudgetFraction` | "ledger §B1" | `§B1:236` | R-RL-11 |
| OTel SDK measurement | "ledger Q2" | `Q2:304` | R-RL-14 |
| semgrep reasoning | "ledger §C3:308" | `Q3:308` — §C3 does not exist; line 308 is Decision question Q3 | R-RL-15, S8a |
| `o200k_base` is real BPE | "ledger §A4" | `§A4:80` | R-RL-09 |
| adapter capability table | `profiler-spec.md:135-139` | **does not exist**; `:135-139` is `ActivationEntry`. Real target: §"Adapter implementations" `:166` | S0, §C, §F |
| comparison schema | (uncited) | `compare.go:11`, `skill-architect/comparison/v1` | D13, SC |

---

## 5. What is still unproven, and deliberately so

D11 makes four decisions from **documentation, not observation**, because Cursor is absent from this machine (§G1). That is not a defect this round can close — it is the epic's premise, and it is User Question 1. What round 3 changed is that each decision now (a) names the ledger line that shows the gap, (b) states its caveat where a reader will see it, and (c) has a **SP1 DoD item that validates or refutes it** the first time anyone runs Cursor. §G3 records this as a single named unknown rather than four scattered assumptions.

The one that would hurt most if wrong: **D11.2 assumes shell work produces `postToolUse`/`postToolUseFailure` pairs.** If only the `*ShellExecution` pair fires, every shell row lands as `aborted` under the orphan rule — rate-neutral, but visibly lossy. **SP1 DoD 3(a) asks exactly that**, and S1b's `## Next` instructs the implementer to flag it to the manager rather than invent a success.
