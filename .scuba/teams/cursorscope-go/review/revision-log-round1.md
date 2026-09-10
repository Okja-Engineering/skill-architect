# Revision log — E-CS roadmap, round 1

**Artifact:** `roadmap.md` + slice files · **Date:** 2026-09-10 · **Groomer:** cursorscope-go
**Input:** `hunter-verifiability.md` (18 findings, V-n) + `hunter-conformance.md` (15 REAL C-n, 4 low C-L-n) = **37 dispositions**
**Method:** every finding re-verified against `profiler/types.go`, `profiler/compare.go`, `profiler/cursor.go`, `profiler/profiler_test.go`, `docs/profiler-spec.md`, `AGENTS.md` and the ledger before acting. Repairs were made at five roots, not finding-by-finding.

**Structural changes:** new **SC** (contract slice, ships first) · new **SP1** (gating spike, moved ahead of S3) · **S1 split into S1a/S1b** (`S1.status.md` deleted) · every slice file gains a **Test approach** section and its own `_test.go` file · **B.4** added: nine Design decisions (D1–D9) closing register conflicts.

---

## Root R1 — contract ownership (verifiability findings 1–5)

| # | Verdict | Disposition |
|---|---|---|
| V-1 | **REAL** — reproduced: `CapabilityReport.Capabilities` is `map[MetricName]MetricSource` (`types.go:50-55`); no reason field exists | SC adds `Notes map[MetricName]string`; S0's DoD now asserts `Notes["tokens"]` explicitly (`S0.status.md` §DoD 5) |
| V-2 | **REAL** — reproduced: `types.go:54`, second assignment overwrites | SC adds `Available map[MetricName][]MetricSource`; S7's DoD 1 rewritten against it and SC added to S7's deps |
| V-3 | **REAL** — no aggregate member in the enum (`types.go:41-47`) | SC adds `SourceOtelAggregate`; S6's DoD now asserts the literal `otel_aggregate` label (`S6.status.md` §DoD 2) |
| V-4 | **REAL** — `profiler-spec.md:135-139` is normative and S0 excluded it | `docs/profiler-spec.md` added to **both** SC's and S0's Files; SC owns the schema, S0 owns the adapter capability table |
| V-5 | **REAL** — reproduced: `compare.go:107,115` gates on state only and reports `candidate.Source` | SC adds `BaselineSource`/`CandidateSource` to every comparator **and** D3's honesty classes, so a cross-class comparison is refused, not silently differenced. `present`'s definition rewritten (D2), removing the `SourceEstimated` contradiction |

## Root A — thesis vs the adapter's real states (conformance 1–2)

| # | Verdict | Disposition |
|---|---|---|
| C-1 | **REAL** — reproduced: `cursor.go:259-273` returns `PresentActivationResult`; `profiler_test.go:644-652` passes on wrong keys | §A thesis rewritten from the adapter's actual states: three adapters return `unknown` unconditionally (`claude_code.go:104`, `codex.go:74`, `devin.go:68`); Cursor returns **`present` with an empty skill name** — worse than unknown |
| C-2 | **REAL** — same root | S3's goal reframed (`S3.status.md` §Goal): "the first activation that names a real skill, and the first from a non-Enterprise surface", with the old claim quoted and refuted in place |

## Root B — evidence labels assigned by topic (conformance 3–10)

| # | Verdict | Disposition |
|---|---|---|
| C-3 | **REAL, with a caveat of my own** | R-RL-07 and `S9.status.md` corrected to **60 req/min** for `filtered-usage-events`. I could not fetch the live docs; the ledger says only "20–250 req/min **by endpoint**" (§A5.3), which does not support the roadmap's flat "20/min". I recorded the reviewer's figure **plus an explicit re-verify-before-building instruction** rather than asserting it as settled |
| C-4 | **REAL** | "30-day max range" removed from R-RL-07/S9 and attributed to `daily-usage-data`/`audit-logs`; §G6 now names the rate limit as unverified |
| C-5 | **REAL** — confirmed: ledger §A5.1:124 lists `afterShellExecution` as `command`/`output`/`duration`/`sandbox` only | R-CS-15 relabelled **Local hypothesis**, retagged *adapt*; **S1b's DoD no longer depends on `exit_code`** — `Status` falls back to `unknown` and SP1's real capture settles it. Added as §G3 |
| C-6 | **REAL** | R-RL-11's label split: 1,536/500 stay **Specification**; `skillListingBudgetFraction` 0.01 becomes **External evidence** with the "fraction of context window, not an absolute" caveat restored (roadmap R-RL-11, `T1.status.md` §Grounding) |
| C-7 | **REAL** — ledger §A5.1 finding 3 says *"Local hypothesis (needs a spike)"* in its own words; the roadmap cited it as Specification | R-CS-09 relabelled **Local hypothesis**; noted in `S3.status.md` §Requirements |
| C-8 | **REAL** | R-CS-12 relabelled **External evidence** (cursorscope's probe); noted in `S5.status.md` §Requirements |
| C-9 | **REAL** | R-RL-09 label split: the tool emitting `.token_counts` is **Repository fact** (measured); `o200k_base` being real BPE is **External evidence**. `T1.status.md` §Grounding corrected |
| C-10 | **REAL** — `AGENTS.md` is 23 lines and has no env/dotenv rule | R-CS-07 relabelled **Design decision**; `S1a.status.md` DoD 3 says so explicitly |

## Root C — S0's DoD vs the wire shape (conformance 11–14)

| # | Verdict | Disposition |
|---|---|---|
| C-11 | **REAL** | `docs/profiler-spec.md` added to S0's Files; the schema change itself moved to SC so the doc has one owner |
| C-12 | **REAL** — same as V-1 | S0's "tokens: none with a reason" replaced by an assertion on SC's `Notes` channel |
| C-13 | **REAL** — confirmed: a delta counter has no timestamp, no `cursor.conversation.id`, and `cursor.tool.status` is tri-state against `Success bool` | S0's DoD 4 rewritten to what the wire supports: **one row per (`cursor.tool.name`, `cursor.tool.status`) with `Count`, `Status`, empty `Timestamp`, `Source: otel_aggregate`** — not per-call rows. SC adds `ToolCallEntry.Status`/`Count` to make `aborted` representable. R-RL-02 amended |
| C-14 | **REAL** — reproduced: `profiler_test.go:560,564,565,605,626,644-652` pin the defect | S0's Test approach now says **replace, not extend**, and lists the five pinned lines. R-SA-16 retagged *adapt* and **exempts S0**; §F states the Go test count changes there by design |

## Root D — register conflicts (conformance 15)

| # | Verdict | Disposition |
|---|---|---|
| C-15 | **REAL**, both halves | **D4:** R-RL-13 wins, **R-CS-16 dropped** — emitting `cursor_*` from a non-Cursor tool is the R-SA-14 failure. **D5:** the one file allowed to hold `cursor.*` literals is `profiler/cursor_wirekeys.go`, created by S0; since S10 now emits no `cursor.*` string at all, the S8-guardrail-4 vs S10-DoD conflict dissolves rather than being arbitrated |

## Verifiability / dependency findings

| # | Verdict | Disposition |
|---|---|---|
| V-6 | **REAL on the rule conflict; PARTIAL on the dep — I disagree in part** | Guardrail 2 rewritten to forbid `present` with a **non-`estimated` source** from an `estimate*` function, i.e. the mislabel, not the state (`S8.status.md` rule 2). But S8's new dep is **SC, not S5**: a seeded-violation test needs only the `estimated` constant to exist, so requiring S5 would gate the MVE on User Q3 while proving nothing extra. This is what keeps the MVE closed |
| V-7 | **REAL; cut in two, not three — partial disagreement** | S1 split into **S1a** (stdin → spool: 21 events, base fields, never-break-session, config, latency) and **S1b** (spool → Profile: correlation, round-trip, `snapshot_hash`, session select, degradation). The third body of work the hunter counted is attribution, which is already S3. Each half's DoD now proves each requirement it claims; R-CS-01 is proven by a 21-fixture table rather than asserted |
| V-8 | **REAL, both halves** | S5's dep on S3 **removed** (phantom — token extraction needs no attribution). MCP self-reported tokens given a label: SC adds `SourceMCPReported`; S5's DoD is now a four-row source→label→honesty-class table |
| V-9 | **REAL** | §F corrected: `profiler/cursor.go` is written by **S0, S1b, S6, S7, S9**, with an ownership order and a named single collision (S0 ‖ S1b). Test-file collisions removed at the root: **every slice owns its own `_test.go`**; only S0 edits `profiler_test.go` |
| V-10 | **REAL** | S10 and S11 both gain their transitive dep on **S2** (which creates `profiler/cmd/skill-scope/`); noted in both slice files and in the graph |
| V-11 | **REAL — the most load-bearing sequencing finding** | The R-RL-17 spike is extracted into **SP1**, placed **before S3** as a hard gate, depending on no code from us (the user hand-installs a five-line hook). S2 is thereby unblocked from User Q1 and is now buildable today against a fake `$HOME`. The MVE states what happens if SP1 answers **no** |
| V-12 | **REAL, systemic** | Every slice file (16 of 16) now has a **Test approach** section naming the test file and the assertions |
| V-13 | **REAL** | **D6** defines two disjoint `Trigger` vocabularies — first-party `agent_read\|manually_attached\|skill_name_in_prompt` (S0 pass-through) and inferred `inferred_file_read\|inferred_shell_command\|inferred_subagent` (S3). SC documents both; S3 carries a negative test enforcing the split |
| V-14 | **REAL — orphan** | R-SA-04 assigned to **S1b** and now appears in its DoD 4 and its test list (`snapshot_hash` survives round-trip) |
| V-15 | **REAL — three orphans** | **R-CS-14** → explicitly **dropped from E-CS** (D8), recorded in the spec doc's "considered, not in v1.1" list. **R-CS-04** → satisfied in adapted form by **S1a**'s spool (D9); S11 carries only the original HTTP form. **R-RL-16** → retagged **defer → F04**; S8 documents the constraint and hands it off, since no E-CS slice implements a CI |
| V-16 | **REAL — four phantom claims** | Removed: R-CS-06 and R-CS-23 from S11 (with a line saying why each is not re-claimed); R-RL-08 from S9 (it is a *drop*, carried as a do-not-port note); R-SA-11 from S3 (it writes no doc file — R-SA-11 belongs to S8) |
| V-17 | **REAL** | Graph rebuilt; the S1→S7 edge exists as `S1b --> S7`. All 26 edges cross-checked against the table's `Depends on` column in both directions; graph is acyclic |
| V-18 | **REAL** | **D7:** the `hooks.json` merge is **Go** (atomic, non-clobbering, backed up); `jq` appears only in the bash test's byte-comparison. The roadmap's "`jq` does the merge" line is gone |

## Low findings (conformance)

| # | Verdict | Disposition |
|---|---|---|
| C-L-1 | **REAL** | The "<50 ms" gate is removed. `S1a` **measures first**: `tests/test_hook_latency.sh` records p50/p95 as a Repository fact, then pins the CI ceiling at 2× measured p95. §G7 rewritten so no slice gates on an invented threshold |
| C-L-2 | **REAL** | S4's DoD proves the 8192 boundary from **both** sides: a 16 KB payload truncated to exactly 8192, a 4 KB payload written untruncated |
| C-L-3 | **REAL** | The generic "AGENTS.md forbids Python" overread is removed from §A and R-RL-15; remaining `AGENTS.md:6` citations are scoped to shell **scripts** (S2's merge, S8 guardrail 1), which is what the rule actually covers. R-RL-15 relabelled a **Design decision over a Local hypothesis** — semgrep is not installed here (§G12) |
| C-L-4 | **REAL** | Carried: §F "Notes, unowned" now lists the `AGENTS.md:11` skillscore error (npm/7-dimension → **Dart/six**) alongside the CI `go-version` note; repeated in `T1.status.md` §Next as the nearest neighbour |

## Deferred (honestly flagged by the hunter, not defects)

S5 gated on User Q3 · S8 partly gated on Q8 · S2/S6/S9/S10/S11 conditionals — all retained as gates, with **two changes**: S2 is no longer gated on Q1 (V-11), and SC is explicitly **not** gated on Q3, because Q3 decides S5's producer, not whether the constant exists.

## Rejected

**None outright.** Two partial disagreements are recorded in place (V-6: the dep is on SC, not S5; V-7: two slices, not three) and one caveat on evidence I could not independently confirm (C-3: the 60 req/min figure is carried with a re-verify instruction, because neither this repo nor the ledger can settle it). Both hunters' "clean" lists are preserved intact: all 21 `cursor.*` names, the 21-event hook surface and its payload fields, the Admin API verb/path/auth/`conversationId`/`tokenUsage` shape, every Fork (v) measurement, `AGENTS.md` conventions, and the acyclic graph.
