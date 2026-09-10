# Revision log — E-CS roadmap, round 4 (final planned revision)

**Artifact:** `roadmap.md` (354 lines, was 335) + **17** slice files · **Date:** 2026-09-10 · **Groomer:** cursorscope-go
**Input:** `hunter-verifiability-round4.md` (1 HIGH + 8 MED + 7 LOW = **V4-1…16**) + `hunter-conformance-round4.md` (3 MED + 6 LOW = **C4-1…9**) = **25 dispositions**
**Method:** both hunters named the same shape of remaining defect and both named it as a *root*, not a list. Round 3 proved every "present from hooks" assertion had a **supplier**; round 4 asks the question that follows — **does any code enforce the rule the DoD states, and did the fix that closed one instance close its class?** §1 sweeps the token single-writer class across all five producers. §2 lists every SC rule against the mechanism that now enforces it. §3 lists every slice that adds a bash test against how CI runs it. §4 is the per-finding disposition. §5 re-derives every citation both hunters challenged. §6 is what is deliberately left open (mirrored into `roadmap.md` §H).

**Structural changes:**
1. **`SelectTokenSource` + `TokenCandidate` move from S5 to SC** (`profiler/tokens_select.go`), becoming the only assigner of `Profile.Tokens` in the repo. The rank order becomes **total over five labels**, and a **session-eligibility** rule makes "an unscoped number never answers a `--session` query" mechanical rather than each slice's local promise.
2. **SC gains the five mechanisms its rules lacked:** a `context_window` comparator, `ValidateProfile` (called from `CompareProfiles`), a closed-set `RawMetricResult.Caveat`, a comparability gate for `Status`-less profiles, and the additive rule extended from two wire strings to **four**.
3. **S8a gains guardrail 5** (single writer + no `hooks` flow label) and **rule 1 is scoped to `skills/*/scripts/`** so it ships green. Its `ci.yml` edit is dropped.
4. **S1a gains DoD 7:** CI runs `tests/test_*.sh` by glob. `ci.yml` now has exactly one writer in the whole epic and four slices' bash tests are wired by inheritance.
5. **Two edges added** (SC → S9, SC → S10): 33 → **35**. The S6 → S5 edge the round-4 brief scoped is **not** added — see V4-#2 below.
6. **`roadmap.md` §H "Known residue"** added: seven items (H1–H7), each with its finding id.

**Line budget:** 335 → **354**. +8 is §H (the brief's deliverable), +2 the two Mermaid edges, +1 the ownership test-file row, +8 net inside replaced lines. Slice count stays **17**.

---

## 1. Class sweep — every slice that produces token data (V4-#1 HIGH, #2, #3, #9; C4-#2)

**The root.** D12 says one `TokenResult` has one `Source`, so `Profile.Tokens` must have one writer. Round 3 swept S5/S6/S9 and named S5's `selectTokenSource` as that writer. It missed **S0**, which reads `cursor.token.usage` and assigns `Profile.Tokens` directly in **wave 2, inside the MVE, gated on nothing** — while S5 is wave 3 and gated on User Question 3. A single-writer function cannot live in a slice that its writers do not depend on, and making S0 depend on S5 would park an MVE slice behind a user answer. **The carrier therefore moves to SC**, which all five producers already depend on (S9 and S10 now declare it — V4-#16).

| Slice | Wave | Token data it produces | Supplier | Round-3 behaviour | Round-4 behaviour | Label | `SessionScoped` | Assigns `Profile.Tokens`? |
|---|---|---|---|---|---|---|---|---|
| **SC** | 1 | *none* — it owns the carrier | — | selector lived in S5 | defines `TokenCandidate` + **`SelectTokenSource`**; ranks, filters, refuses, assigns | — | — | **YES — the only one** |
| **S0** | 2 | `cursor.token.usage` metric, export-wide | ledger §A5.2:158 (no conversation id) | **assigned `Tokens` directly, labelled `otel`** — the fourth writer, unswept | registers a candidate | **`otel_aggregate`** (was `otel`; it is the same number S6's fallback reads, so it must wear the same label) | **false** | no |
| **S1b** | 2 | *none* | ledger Q1:300 — no hook reports flow | left `tokens` `unknown`-with-reason (correct, but by hand) | registers **zero** candidates and asserts the empty-list result | — | — | no |
| **S5** | 3 | `chars/4` estimate; MCP self-report | spool payloads; `afterMCPExecution.result_json` | owned the selector **and** the producers | producers only; carries `CaveatEstimatedFlow` | `estimated`, `mcp_reported` | **true** | no |
| **S6** | 3 | `cursor.api.request.*_tokens`; `cursor.token.usage` fallback | §A5.2:152, :158, :166 | DoD 2 said `otel_aggregate`, DoD 4 said "registers an `otel` candidate" — **one number, two names, neither ranked** (C4-#2) | registers **two** candidates, one per metric | `otel` (per-request) **and** `otel_aggregate` (fallback) | true / **false** | no |
| **S9** | cond | Admin API `tokenUsage` | §A5.3:180 | registered with S5's selector | registers with **SC's** selector | `server_api` | **true** | no |

**The rank, now total (V4-#3, C4-#2):** `server_api` > `otel` > **`otel_aggregate`** > `mcp_reported` > `estimated`. Class order first (D3: `measured_billed` > `harness_reported` > `self_reported` > `estimated`); the single intra-class tie — `otel` vs `otel_aggregate`, both `harness_reported` — breaks on **scope precision**, not honesty: `cursor.api.request` is per-request and conversation-scoped (§A5.2:152), `cursor.token.usage` is an export-wide sum with no conversation id (§A5.2:158). Both are the harness reporting on itself, so both outrank an MCP server's self-report about itself. Round 3's four-label order left `otel_aggregate` unranked while **three** slices could emit it.

**Session eligibility (new).** A candidate carries `SessionScoped bool`; under `--session`, unscoped candidates are **ineligible** and dropped with a named reason. This turns S6 DoD 2's second case and S0's export-scoped aggregate from two local promises into one contract rule.

**The `hooks` invariant, now enforced twice (V4-#9).** D12's "`hooks` is never a token-flow label" had **no test and no guardrail in any slice**. Now: (a) `SelectTokenSource` **refuses** a candidate whose `Source` is `hooks`, `session_data`, `sqlite`, `none` or `""`, naming the string — asserted in **`profiler/contract_test.go`** (SC); (b) **S8a guardrail 5(b)**, a `go/ast` walk in **`profiler/guardrails_test.go`**, fails any `TokenCandidate` literal or `PresentTokenResult` call carrying one of those labels, anywhere in non-test source. **Single-writer** is enforced by **guardrail 5(a)** in **`tests/test_guardrails.sh`**: `grep -rn '\.Tokens[[:space:]]*=' profiler/ --include='*.go'` minus `tokens_select.go` and `*_test.go` must return zero, seeded red→green. **Owner: S8a**, which already depends on SC — no new edge.

**V4-#2, the S6 → S5 edge.** Real against round 3's plan: S6 DoD 4 called `selectTokenSource`, defined in `tokens.go`, created by S1b and filled by S5 in the **same wave** — S6 would not have compiled. The repair chosen is the one that *also* fixes S0 (which no edge could have fixed without parking the MVE): the selector moved to SC. With the carrier contract-owned, **S6 → S5 would be an invented dependency** — the two slices append to a shared list and read none of each other's code — so per `sequence-verifiable-units` ("name genuine dependencies, invent none") it is **not** added. The reasoning is recorded in `S6.status.md` § Next and in `roadmap.md:180`, so a reviewer who prefers the selector in S5 can see exactly what the edge would cost: S6 → S5 **and** S0 → S5, the second of which parks an MVE slice behind User Q3.

---

## 2. Class sweep — every SC rule against the mechanism that enforces it (V4-#4, #5, #6, #8; C4-#3)

**The root.** SC's DoD added *channels* (fields, constants, classes) and *rules* (prose in the DoD and the spec doc). Five rules had no code anywhere in the epic — and three of them were asserted against by **other** slices, which is how the gap survived three rounds: the assertion looked covered because someone tested it, but nothing implemented it.

| # | Rule as round 3 stated it | Enforcement in round 3 | Enforcement in round 4 | Where | Finding |
|---|---|---|---|---|---|
| 1 | `tokens` and `context_window` are compared as separate metrics | **none** — `CompareProfiles` (`compare.go:88-93`) has five comparators and no `context_window`; only SC may edit `compare.go` and no SC DoD added one; S7 asserted it anyway | **`compareContextWindow`** + `report.Metrics["context_window"]`, both sides' sources, cross-class refusal (adding a map key is what D13 permits, so `ComparisonSchema` stays `v1`) | SC item 10; tested in `contract_test.go`; exercised end-to-end in S7's `unify_test.go` | V4-#4 |
| 2 | Every `Attribution.Target` resolves to a `ToolCallEntry.ID` or bears a declared scheme prefix; a bare string is "a contract violation the test catches" | **none** — `grep -n 'func.*Validate' profiler/*.go` returns nothing and no DoD created one | **`func ValidateProfile(p Profile) []error`** in `profiler/validate.go`, covering five violation kinds, **called from `CompareProfiles`**, which appends each to the existing `ComparisonReport.Notes` — so it is live, not decorative. Safe against the 34 green tests: all four shipped adapters return `attribution: unknown`, so rule (a) has nothing to fire on until S3 ships | SC item 12; `TestValidateProfile`; S3 and S1b assert **through** it rather than restating it | V4-#5, C4-#3 |
| 3 | New fields are `omitempty` and declared last, so a pre-change golden re-marshals byte-identically | covered `ProfileSchema` (D1) and `ComparisonSchema` (D13) — **two of four** wire strings | rule extended to **`ExperimentSchema` and `ExperimentPlanSchema`** (`experiment.go:14,17`, both strictly validated at `:98-99,237-238`), with two more pinned goldens. SC edits **no** line of `experiment.go`; **S7 inherits the check** — declare `HookSpool` wrongly and SC's test goes red in a file S7 does not own | SC item 16 + two goldens; S7 DoD 3 | V4-#6 |
| 4 | "Two profiles of one session yield zero delta on `count`, `successes` and `success_rate` **regardless of row encoding**" | **falsified by SC's own next bullet** — a `Status`-less encoding of the same session yields 0.50 where a `Status`-bearing one yields 0.75 | **A comparability gate.** `count` compares always; the four outcome figures are emitted **only when both sides carry non-empty `Status` on every row**, else the keys are absent and `Reason` names the offending side. **Wording fixed to what the gate guarantees:** *"zero delta on `count` in every encoding, and on the outcome figures whenever both sides carry `Status`."* Both v1.1 producers (S1b per-call, S0 aggregate) sit inside the gate; a pre-v1.1 profile does not, and now says so | SC items 13–14; the third-encoding test in `contract_test.go`; S7 asserts both real adapters land **inside** the gate | V4-#8 |
| 5 | "The profile … states the [timing] caveat verbatim" | **no carrier existed.** `RawMetricResult` is `{State, Reason, Source}` and `Reason` must be empty when `present` (`profiler-spec.md:33`); `Notes` is on `CapabilityReport`, which describes a capability, not a value | **Option A taken: `RawMetricResult.Caveat`** — additive, `omitempty`, declared last, permitted only when `State == present`, and restricted to a **closed set** of two declared constants (`CaveatHookReceiptClock`, `CaveatEstimatedFlow`) so it cannot become free prose. Chosen over "doc-only" because §A's thesis is that *the profile* must say which honesty level it used; a caveat that lives only in a doc is not in the artifact F04 reads. S1b's DoD now sets a field instead of claiming prose | SC item 9; S1b DoD 4 + round-trip assertion; S5 DoD 2; S3 DoD 2 | C4-#3 |

---

## 3. Class sweep — CI wiring per slice (V4-#7, C4-#1; and V4-#11, C4-#9 on ownership)

**The root.** `.github/workflows/ci.yml:24-31` runs four bash scripts **by name**. Four slices add a `tests/test_*.sh`; three of them (`S2`, `SP1`, `S8b`) did not list `ci.yml` in `## Files` at all, and `roadmap.md` §F claimed only S1a and S8a write it. Those tests would have been committed, green locally, and **never executed**.

**Decision — the brief's option (b): S1a switches CI to a glob once and everyone inherits.** Recommended over "every slice lists `ci.yml` and wires its own script" because (i) it makes `ci.yml` a **single-writer** file, removing the S1a‖S8a co-ownership the ownership map had to explain; (ii) a per-slice wiring rule is a convention that the next slice forgets, which is the failure mode this table exists to catch; (iii) it costs one step in one PR instead of four edits in four PRs to a shared file.

| Slice | Bash test it adds | Round-3 wiring | Round-4 wiring | Lists `ci.yml`? |
|---|---|---|---|---|
| **S1a** | `tests/test_hook_latency.sh` | named step, plus the latency ceiling in the workflow | **DoD 7: replaces the four named steps with `for f in tests/test_*.sh`**, placed **after `actions/setup-go`** so scripts may build Go binaries. Ceiling moves **inside the script**. Proven with a throwaway failing canary observed red and removed in the same PR | **yes — the epic's only writer** |
| **S2** | `tests/test_hooks_install.sh` | **none — would never have run** | discovered by the glob | no (by design) |
| **SP1** | `tests/test_spike_activation.sh` | **none — would never have run** | discovered by the glob | no |
| **S8a** | `tests/test_guardrails.sh` (+ `guardrails_test.go`) | listed `ci.yml`, wired its own rules | discovered by the glob; its Go test already runs under `go test ./...` | **no longer** |
| **S8b** | `tests/test_doc_labels.sh` (fallback only) | **none — would never have run** | discovered by the glob, fallback included | no |
| T1 | extends `tests/test_f02.sh` | already named | already covered; unchanged by the glob | no |

**Two rules ride with the glob**, stated in S1a DoD 7 because that is where they become load-bearing: every `tests/test_*.sh` is **self-contained** (builds or skips what it needs) and exits non-zero on failure; the glob runs in **lexical order** and no script may depend on another having run.

**Ownership map (V4-#11, C4-#9).** The claim "every file any slice touches appears here" omitted `guardrails_test.go`, `cursor_admin_api.go`, `otlp.go`, ~12 `_test.go` files and SP1's fixture. Fixed by **listing them, not by narrowing the claim**: the header now reads "every file any slice **writes** … including `_test.go` files and fixtures", the single-owner source row gained `tokens_select.go`, `validate.go`, `cursor_admin_api.go` and `otlp.go`, and a **new row enumerates all 15 single-owner test files and SP1's fixture with their owning slice and wave**. `profiler_test.go` remains the one shared test file, written only by S0.

---

## 4. Per-finding disposition

### Verifiability lens (V4-1…16)

| # | Sev | Disposition | Where |
|---|---|---|---|
| **1** | **HIGH** | **FIXED AT ROOT, CLASS SWEPT.** S0 was the unswept fourth writer. Rather than patch S0, the single-writer carrier moved to SC (wave 1), which all five producers depend on. All five slices swept: §1's table is the proof, and every row now says what it registers and that it assigns nothing | SC item 8; S0 DoD 3; S1b DoD 10; S5 Part 2; S6 DoD 4; S9 DoD 3; roadmap D12, S0/S1b/S5/S6/S9 rows |
| **2** | MED | **FIXED — by dissolution, not by adding the edge.** The compile break was real; the repair (selector → SC) removes the call across a wave boundary entirely. Adding S6 → S5 afterwards would be an invented dependency, and S0 would still need S5 — parking an MVE slice behind User Q3. Recorded in three places so the choice is auditable | `S6.status.md` § Next; roadmap:180; §1 above |
| **3** | MED | **FIXED.** `otel_aggregate` ranked, order now total over five labels, with the tie-break reason stated (class order first, then scope precision). Added to S5's Part-2 pointer and to D12 | SC item 8; roadmap D12; S5 |
| **4** | MED | **FIXED.** `compareContextWindow` added to SC; S7's assertion now has code to bind to | SC item 10; S7 test bullet 6 |
| **5** | MED | **FIXED.** `ValidateProfile` named, specified (five violation kinds), tested (`TestValidateProfile`) and **called** from `CompareProfiles` | SC item 12; S3, S1b assert through it |
| **6** | MED | **FIXED.** Additive rule extended to the two experiment wire strings, with two more pinned goldens; S7 inherits the check without editing SC's file | SC item 16; S7 DoD 3 |
| **7** | MED | **FIXED AT ROOT.** CI glob in S1a DoD 7; three slices' orphaned scripts wired by inheritance; `ci.yml` becomes single-writer; S8a drops it from Files | S1a DoD 7 + Files; S2/SP1/S8b § CI wiring notes; S8a; roadmap ownership row |
| **8** | MED | **FIXED.** Comparability gate specified; the "regardless of row encoding" wording replaced with what the gate guarantees | SC item 14; S7 DoD 5 |
| **9** | MED | **FIXED.** Guardrail 5 added to S8a (grep + AST, both seeded red→green) and a refusal test in SC | S8a rule 5; SC item 8 + test bullet 3 |
| **10** | LOW | **FIXED.** Both join forms come from `beforeReadFile` and therefore share `Trigger: inferred_file_read`; the **scheme prefix is the only distinction**, which is exactly what `ValidateProfile`'s disjunction reads. S3 DoD 3 rewritten and a test added asserting both forms carry the same trigger | S3 DoD 3–4 + test bullet 4; roadmap D11.4 |
| **11** | LOW | **FIXED by listing, not narrowing.** See §3 | roadmap:285, :301, and the new test-file row |
| **12** | LOW | **FIXED.** `ParseDir` given a filter excluding `_test.go`; the seed lives in a **scratch non-test file** written by the harness and deleted; walked types gain `ContextResult` (plus `TimingResult`, `AttributionResult`) | S8a rule 2 + test approach |
| **13** | LOW | **FIXED.** `[+]` is 3 bytes; the rule is now stated in bytes throughout — truncate to the largest rune boundary ≤ **8189** bytes, append the 3-byte marker, total ≤ 8192 — and the test asserts a rune straddling the boundary is not split | S4 DoD 2 + test bullets |
| **14** | LOW | **FIXED both directions.** R-SA-13 added to S6's `## Requirements` (with DoD 5 as its carrier); R-RL-18's and R-CS-13's register rows now name **SC (carrier) · S5 (producer)**, matching SC's claim | S6 DoD 5 + Requirements; roadmap:54, :116; SC Requirements |
| **15** | LOW | **FIXED.** `status` (`stop`, ledger:135) added to S1a's permitted non-base field list, with S1b DoD 5 named as its reader | S1a DoD 6; S1b DoD 5 |
| **16** | LOW | **FIXED.** S9 and S10 declare **SC**, with the reason in the dependency line; two edges added (33 → 35), Mermaid and the wave table updated | S9, S10 headers; roadmap:180, Mermaid, :315 |

### Conformance lens (C4-1…9)

| # | Sev | Disposition | Where |
|---|---|---|---|
| **1** | MED | **FIXED — rule scoped to its citation.** Executed at this gate: `scripts/` does not exist; `tests/test_skill.sh:23` and `tests/test_walk.sh:25` invoke `python3`; `skills/*/scripts/` (six scripts) is clean. Rule 1 is now `grep -rn 'python3' skills/*/scripts/` — the width `AGENTS.md:6` ("skill scripts") actually states — so it **ships green**. Widening it to `tests/` requires removing those two invocations; **no slice owns that**, and it is recorded as residue **H1** rather than silently assumed | S8a rule 1; roadmap §H H1 |
| **2** | MED | **FIXED.** One number, one label: S6 registers `otel_aggregate` for the fallback and `otel` for per-request, and `otel_aggregate` is ranked. See §1 | S6 DoD 4; SC item 8 |
| **3** | MED | **FIXED.** `RawMetricResult.Caveat`, closed-set. S1b stops claiming the profile states a caveat it could not carry; it sets the field | SC item 9; S1b DoD 4 |
| **4** | LOW | **FIXED.** Verified by reading `types.go`: **five** `Unknown*Result` constructors (`:196,216,231,246,261`) plus **`ErrorTokenResult` (`:201`)** as the sixth `Source == ""` constructor, now named | roadmap D3:122; SC item 15 |
| **5** | LOW | **FIXED — one reading adopted.** S9 now builds against **60 req/min for `filtered-usage-events`** and **the 30-day maximum date range applying to it**, labelled *Specification, live docs re-read 2026-09-10*, with the ledger's API-wide figures (§A5.3:177, :182) noted as non-endpoint-specific and stale where they disagree. Re-verify before building (§G6, residue **H6**) | S9 § Next; roadmap S9 row |
| **6** | LOW | **FIXED.** Counted: header at :115, separator at :116, **21 data rows at :117-137** | roadmap:42; S1a DoD 1; S1a row |
| **7** | LOW | **FIXED.** See V4-#13 — same line, same fix | S4 DoD 2 |
| **8** | LOW | **FIXED.** Read verbatim: `types.go:114` says *"a target (tool call ID **or output**) to a skill"*; the word "identifier" is `profiler-spec.md:154`'s. Both are now cited for what each says | roadmap D11.4:134; SC item 4; S3 DoD 3 |
| **9** | LOW | **FIXED.** See V4-#11 / §3 | roadmap:285 + new row |

---

## 5. Citations re-derived at this gate

Every count and quote below was produced by running the command or reading the line, not inferred from the block's topic — the shared root the conformance hunter named across five findings.

| Claim | Round-3 citation | Re-derived at round 4 | Fixed in |
|---|---|---|---|
| `Unknown*Result` constructors leaving `Source == ""` | "six `Unknown*Result`", `types.go:196,216,231,246,261` | **five** `Unknown*Result` (Token, ToolCall, Activation, Timing, Attribution) + **`ErrorTokenResult` at `:201`** = six constructors, five of that name | D3, SC item 15 |
| §A5.1 hook-event table | ":115-137, 21 table rows" | header `:115`, separator `:116`, **21 rows at `:117-137`** (`sed -n '117,137p' \| grep -c '^\|'` → 21) | R-CS-01, S1a |
| `Attribution.Target` wording | "tool call ID **or output identifier**" cited to `types.go:114` | `types.go:114` = *"links a target (tool call ID or output) to a skill"*; `profiler-spec.md:154` = *"// tool call ID or output identifier"* | D11.4, SC item 4, S3 |
| truncation marker size | "1-byte marker `[+]`" | `[+]` is **3 bytes**; rule restated in bytes with an 8189 + 3 split | S4, R-CS-20 |
| `python3` in shell scripts | rule over `tests/ scripts/` | `scripts/` absent (exit 2); `tests/test_skill.sh:23`, `tests/test_walk.sh:25` present; `skills/*/scripts/` (6 scripts) clean | S8a rule 1, H1 |
| `CompareProfiles` comparators | "compared as separate metrics" | **five** comparators at `compare.go:88-93`; no `context_window` | SC item 10, S7 |
| profile validators | "a contract violation the test catches" | `grep -n 'func.*Validate' profiler/*.go` → **no match** | SC item 12 |
| strictly-validated wire strings | two (`profile/v1`, `comparison/v1`) | **four** — `experiment.go:14,17`, enforced at `:98-99,237-238` | SC item 16, S7 |
| live `cursor.` literals | 12 at `:175,182,184,186,197,216,224,234,242,250,262,271` | **re-confirmed 12**, same lines (`grep -c 'cursor\.' cursor.go` → 12) | unchanged |
| `MetricSource` constants | nine after SC | **six shipped** (`types.go:40-46`) + three added = **nine** | unchanged |
| Admin API limits | ledger figures, adopted by neither | **60 req/min** for `filtered-usage-events`; **30-day range applies**; hourly aggregation (§A5.3:182) | S9, H6 |
| `stop` payload | absent from S1a's permitted list | `stop` carries `status`, `loop_count` at ledger **:135**; `status` is read by S1b DoD 5 | S1a DoD 6 |
| Mermaid edge count | 33 | **35** after SC → S9, SC → S10 (`grep -c '^  [A-Za-z0-9]* -[-.]'` → 35) | roadmap:180 |

---

## 6. Deliberately left open (mirrored to `roadmap.md` §H)

| id | Residue | Finding |
|---|---|---|
| **H1** | `tests/test_skill.sh:23` / `test_walk.sh:25` still invoke `python3`; guardrail 1 is scoped to `skills/*/scripts/` and does not cover `tests/`. No slice owns the cleanup — recommend a one-PR chore thread | C4-#1 |
| **H2** | SC is the largest PR and round 4 enlarged it (19 items, 7 files). A clean `types`/`compare` seam exists; **not** cut in round 4 because re-cutting on the last revision ships an unreviewed shape | V4-#1, #4, #5, #8 |
| **H3** | `ci.yml` pins Go 1.23 while `go.mod` declares 1.27.1 — pre-existing, not this epic's; recommend a one-line PR to `main` before S1a | §F note (a) |
| **H4** | The comparability gate omits keys rather than exposing a per-figure `comparable` flag, because `MetricComparison` has one flag per metric | V4-#8 |
| **H5** | D11's four derivations stay documentation-derived until SP1 runs — the epic's premise, User Q1 | §G3 |
| **H6** | Every Admin API figure in S9 is unexercised Specification; re-verify on the day S9 is built | C4-#5 |
| **H7** | cursorscope's suites remain a transcribed, unexecuted oracle for S3/S4 | §G13 |
