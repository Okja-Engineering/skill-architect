# Revision log — round 5 (E-CS `skill-scope`)

**Input:** `review/hunter-verifiability-round5.md` — NOT-CLEAN, 2 HIGH · 3 MED · 9 LOW · 1 SUSPECTED · 0 INVALID, one shared root behind findings 1–4, plus a DEFERRED recommendation (split SC) that both round-5 hunters made.

**Scope of this round:** the fourteen findings, the suspected item, and the SC split. Nothing else was reopened.

---

## 0. The root, and the invariant that closes it

The hunter's root: *D12's single-writer invariant was specified at repo width while its justification and the epic's file scope are Cursor-only — and rule 5's mechanism was never run against `main`.* The same widening left the assignment site homeless (#3) and stopped at `tokens` when the identical unscopable-aggregate problem exists on `tool_calls` (#4).

**The invariant adopted, and now stated in `S8a.status.md`, in D12 (`roadmap.md`) and in `SC-a.status.md` item 8:**

> The guardrail's width, the contract's admissible-source set, and the set of files the epic owns are **one set** — and no rule may claim red→green until its command has been executed against `main` and its result recorded in the slice file.

Three consequences, all mechanical:

1. **Width.** Rules 5(a)/5(b) are scoped to a **named Cursor-path file set** — `cursor.go`, `cursor_wirekeys.go`, `cursor_admin_api.go`, `hooks.go`, `hooks_read.go`, `enrich.go`, `tokens.go`, `tokens_select.go` — enumerated once, in the two files that run the rules, with a stated maintenance rule. `claude_code.go`, `codex.go`, `devin.go` are never scanned.
2. **Admissible set.** `SelectTokenSource` became an **allowlist** over the five flow labels rather than a denylist over five bad ones. On the Cursor-path set the complement's only constructible member is `hooks`, so the guardrail's denylist and the contract's allowlist coincide — which is what "one set" means in practice. `none`/`""`/typos are caught at runtime by `ValidateProfile` rule (b), on **every** adapter, so the source-level walk does not duplicate them.
3. **Execution.** Every S8a rule now carries an **executed baseline table** (run 2026-09-10 against `main`), and the DoD requires the baselines be re-run and pasted into the PR.

### Executed baselines (the step round 4 skipped)

| Rule | Command | Result on `main` |
|---|---|---|
| 1 | `grep -rn 'python3' skills/*/scripts/` | **0** |
| 2 | `grep -rn 'func [a-zA-Z_]*[Ee]stimate' profiler/` | **0** (nothing for the walk to fire on) |
| 4 | `grep -rn 'cursor\.' profiler/ --include='*.go'` minus allowances | **12**, all `cursor.go` |
| 5(a) repo-wide (round-4 form) | `grep -rn '\.Tokens[[:space:]]*=' profiler/ --include='*.go'` | **24** — `claude_code.go` 8, `codex.go` 5, `cursor.go` 6, `devin.go` 5 |
| 5(a) Cursor-path (round-5 form) | same, restricted to the set, minus `tokens_select.go`/`*_test.go` | **6**, all `cursor.go:95,114,124,134,141,150` → **0** once S0 merges |
| 5(b) package-wide (round-4 form) | `PresentTokenResult`/`TokenCandidate` label walk | **1** — `claude_code.go:465`, `SourceSessionData`, **legitimate** |
| 5(b) Cursor-path (round-5 form) | same, restricted to the set, banned set `hooks` | **0** (the set's only call is `cursor.go:218`, `SourceOtel`) |

---

## 1. Dispositions

| # | Sev | Disposition | Where |
|---|---|---|---|
| **1** | HIGH | **FIXED at the root.** Rule 5(a) scoped to the Cursor-path file set; the repo-wide grep's 24 hits and the scoped grep's 6 are both stated as executed baselines, with "green when S0 merges" named. The 18 hits in the three legacy adapters are recorded as new residue **H8** with a recommended chore thread — not silently dropped, not silently enforced. | `S8a` baselines table + rule 5(a); `roadmap.md` §C S8a row, §H8; D12 (d) |
| **2** | HIGH | **FIXED at the root.** Rule 5(b) scoped to the same file set; its banned set is **`hooks` only**, with the justification written out: SC-a item 8's admissible set is the five flow labels, `hooks` is the only complement member a Cursor-path slice can construct (D12/Q1:300), `session_data`/`sqlite` label other adapters' legitimate surfaces (`claude_code.go:465` maps to `harness_reported`), and `none`/`""` are caught at runtime by `ValidateProfile` rule (b). The selector itself became an allowlist so contract and guardrail agree by construction. | `S8a` rule 5(b); `SC-a` item 8 (allowlist bullet); D12 (d) |
| **3** | MED | **FIXED at the root.** `SelectTokenSource` stays the pure rank; **`ApplyTokenSelection(p *Profile, cands []TokenCandidate, sessionRequested bool)`** in `tokens_select.go` is the one function containing `p.Tokens = …`, and `tokens_select.go` is the one file the guardrail allowlists. The **accumulator got an owner too**: `TokenCandidates` + `Add`/`List`, also SC-a. **Call sites named:** S0 creates the OTel branch's in `cursor.go`; S1b creates the hook branch's as the last statement of `enrich()` (deliberately after `enrichTokens`, so S5 and S9 register from inside the seam without a call site of their own); **S7 collapses the two to one** (new S7 DoD 6). All five producers' DoDs now read *registers candidates; calls nothing that assigns*, and S0/S1b/S5/S6/S9 tests gained the **positive** half — "and calls `ApplyTokenSelection` exactly once" / "adds no second call site" — because the negative half alone is satisfied by an inert file. | `SC-a` item 8; `S0` DoD 3a + tests; `S1b` DoD 10 + tests; `S5` Part 2 + tests; `S6` DoD 4 + tests; `S9` DoD 3 + tests; `S7` DoD 6; D12 (c) |
| **4** | MED | **FIXED at the root, as one rule for both metrics.** SC-a exports `SessionEligible(sessionRequested, sourceIsSessionScoped, exportIsSingleConversation)` and `UnscopedAggregateReason(wireKey, session)`. `cursor.tool.calls` carries no conversation id (§A5.2:159), so `tool_calls` is bound by the identical predicate: under `--session` it is `unknown` with the reason **unless the export is provably single-conversation**. **Detection stated where the export is read** (new S0 DoD 3): the distinct `cursor.conversation.id` set over the export's *log* records equals `{requested session}`; an export with **no** log records leaves the set empty and proves nothing → `unknown`. A metric admitted through the escape carries a **third caveat constant, `CaveatExportScopedAggregate`**. S7 DoD 5 reworded: zero delta is guaranteed only **over the same call set**, and the out-of-scope case is now a test, not an assumption. | `SC-a` items 8, 9; `S0` DoD 3, 3a, 4 + tests; `S6` DoD 2 + tests; `S7` DoD 5 + tests; `SC-b` item 14; D12 (e) |
| **5** | MED | **FIXED.** Validator rule (c) now reads: *a `Status` that is **non-empty and** outside `success\|failure\|aborted`.* `""` is legal — it is exactly the encoding item 14 declares legitimate and *gates* on, and the round-4 rule would have flooded `Notes` on every legacy comparison the moment SC-b wired the call. One rule, one reaction: the **gate** reacts to `Status: ""`, the **validator** does not. Pinned by two new tests — `Status:""` passes, and a whole legacy profile returns zero errors; plus, in SC-b, two legacy profiles compared produce an empty `Notes`. | `SC-a` item 12 (c) + tests; `SC-b` item 12w, item 14, tests |
| **6** | LOW | **FIXED.** Rule 3's heading is now the mechanism — *"No caller of the raw spool appender outside `hooks.go` and `privacy.go`"* — and the property it buys ("nothing unredacted reaches disk") is explicitly attributed to S4's own spool-file grep, not to this rule. | `S8a` rule 3; `roadmap.md` §C S8a row |
| **7** | LOW | **FIXED.** Rule 5(b)'s test is **`TestNoHooksTokenLabel`** (rule 2's remains `TestNoMislabelledEstimate`), named in the rule, in the test approach and in the roadmap row. | `S8a` rule 5(b), Test approach |
| **8–10** | LOW | **FIXED, all three, re-derived from the file.** `ComparisonReport.Notes` field is **`compare.go:40`** (appends at `:81,84,87`) — was `:83-88`. The five comparators are **`:90-94`** — was `:88-93`. `MetricComparison` is **`:23-30`** — was `:22-29`. Corrected in SC-b items 10/12w/14, S7's test bullet and `roadmap.md` H4. Two nearby cites were re-checked and are **correct as written**: `compareTokens` `:106-126` and `compare.go:115`. One more drift found in passing and fixed: `S5` cited "S7 DoD 25"; S7 has five DoD items — now **DoD 5**. | `SC-b` 10, 12w, 14; `S7` tests; `S5` tests; `roadmap.md` H4 |
| **11** | LOW | **FIXED.** Item 17's scope is no longer "items 1–9" but **every field either half of the contract adds**, item 11's `BaselineSource`/`CandidateSource` included; SC-b item 11 restates the obligation locally and its `comparison-v1-pre11.json` golden is the proof. | `SC-a` item 17; `SC-b` item 11 |
| **12** | LOW | **FIXED.** `compareContextWindow` takes `*ContextResult` on both sides. Nil on either → `comparable:false` with `"context_window not captured in <baseline\|candidate>"`; nil on both → both named. Never a zero-valued `ContextData`, never a nil deref. Three test cases added. The nil case's producers are named: every pre-v1.1 profile, every non-compacting hooks capture, all four shipped adapters. | `SC-b` item 10 + tests; `S5` DoD 1 |
| **13** | LOW | **FIXED both ways the hunter allowed.** The constant is now `P95_CEILING_MS`, pinned **with the machine and date beside it — `ubuntu-latest`, the runner `ci.yml:11` names, measured in the PR** — *and* enforcement is gated on `GITHUB_ACTIONS=true && RUNNER_OS=Linux`; anywhere else the script records p50/p95 and exits 0, advisory. A flake in the epic's only `ci.yml` writer is the worst place to put one. | `S1a` Test approach |
| **14** | LOW | **FIXED both directions.** R-CS-05's register row now names **SC-a (carriers) · S1b (producer)**, matching SC-a's `## Requirements`. R-SA-16 is resolved the other way: it is an **epic-wide constraint carried in §F and deliberately claimed by no slice's `## Requirements`** — a slice claiming it would be claiming what the other sixteen also satisfy — and the register row and §F now say so in the same words. S0's exemption is stated in both places. | `roadmap.md` :46, :92, §F bullet; `S0` Test approach |
| **15** (DEFERRED rec.) | — | **DONE — SC split into SC-a and SC-b.** See §2. §H2 closed. | `SC-a.status.md`, `SC-b.status.md`, `SC.status.md` (pointer), `roadmap.md` §C/§F/§H |
| **SUSPECTED** | LOW | **ACCEPTED as suspected; H3 downgraded, not closed.** With `GOTOOLCHAIN` at its default `auto`, a Go 1.23 toolchain reading `go 1.27.1` downloads and re-execs 1.27.1, so the "latent CI break" very likely never fires and the real defect is a workflow that lies about which Go it runs. H3's wording, severity and instruction are rewritten: **confirm from an actual CI run** (read the toolchain line in the `go test` step's log) before acting; if `GOTOOLCHAIN=local` is set anywhere the original break is real. Still a one-line PR to `main`, at whatever urgency the log warrants. Not folded into S1a either way — it is pre-existing on `main`. | `roadmap.md` H3 |

**16/16 addressed** (14 findings + suspected + the deferred recommendation).

---

## 2. The split — SC → SC-a + SC-b

| | SC-a | SC-b |
|---|---|---|
| **Owns** | `types.go`, `tokens_select.go`, `validate.go`, `docs/profiler-spec.md` (channel sections), `contract_test.go`, `profile-v1-pre11.json` | `compare.go`, `docs/profiler-spec.md` (comparison sections), `compare_contract_test.go`, the `comparison`/`experiment`/`experiment-plan` goldens |
| **Items** | 1–9, 12 (defines), 15, 16 (states the rule, pins the profile golden), 17, 18, 19 | 10, 11, 12w (wires), 13, 14, 16b, 19b |
| **Reviewer's question** | "does the type carry it?" | "does the comparison tell the truth about it?" |
| **Depends on** | none — **ships first** | SC-a |

Item numbering is **preserved** across the two files, so every existing citation ("SC item 8", "SC item 14") resolves without ambiguity; `SC.status.md` is a one-line pointer recording the mapping.

### Recomputed edges

| Change | Δ |
|---|---|
| removed `SC → {S0, S1b, S3, S5, S6, S7, S8a, S8b, S9, S10}` | −10 |
| added `SC-a → {SC-b, S0, S1b, S3, S5, S6, S7, S8a, S8b, S9, S10}` | +11 |
| added `SC-b → {S0, S5, S7}` | +3 |
| **total** | **35 → 39** |

Verified: 39 edges in the Mermaid block, 39 in the §C `Depends on` column, 39 across the seventeen slice files; acyclic; SC-a is the single source.

**Three slices need both halves.** S5 and S7 as the hunter predicted. **S0 is the deviation from the round-5 instruction**, which recomputed S0 → SC-a: S0 is the slice that first emits `Count`-bearing aggregate rows, and until SC-b lands `countToolCalls` is `len(calls)` (`compare.go:148-155`), so a six-call session compares as `count` 3 against its own per-call encoding — the HIGH V2-#1 that SC-b item 13 exists to fix, shipped inside the MVE for a merge window. S0's own DoD 4 asserts the consequence, so the dependency is genuine. Declared, with the reasoning, in `SC-b` § Next, `S0` § Next and the MVE note.

**Two edges considered and NOT added** (an unjustified edge is the same defect as a missing one):
- **S1b → SC-b** — dissolved by removing S1b's test clause that asserted the `success_rate` its session yields *under the aggregate encoding*. That arithmetic is SC-b's own heterogeneous fixture and S7's end-to-end test; S1b owns the row-level derivation.
- **S9 → SC-b** — dissolved by removing S9's duplicate cross-class-refusal assertion. That fact is proven once, in SC-b.
- (**S6 → S5** remains not added, unchanged from round 4.)

### MVEs — both close

| | Round 4 | Round 5 | Closes? |
|---|---|---|---|
| **Primary** (SP1 = yes) | SC + S0 + S1a + S1b + S2 + S3 + S4 + S8a + S8b — 9 PRs + SP1 | **SC-a + SC-b + S0 + S1a + S1b + S2 + S3 + S4 + S8a + S8b — 10 PRs + SP1** | **Yes.** SC-b:{SC-a} · S0:{SC-a,SC-b} · S1b:{SC-a,S1a} · S3:{SC-a,S1b}+SP1 · S8a:{SC-a,S0,S4} · S8b:{SC-a,S1b,S4,S3} — every dep inside the set |
| **Fallback** (SP1 = no) | the same minus S3 — 8 PRs | **the same minus S3 — 9 PRs** | **Yes.** S8b's S3 edge is the conditional one (`-.if S3 ships.->`), so it reduces to {SC-a, S1b, S4}; nothing else in the set depends on S3 |

The +1 is SC-b, and it is there because S0 is — not because the split cost a PR. The split itself is 1-for-1.

### Waves — the plan is unchanged at five (strict depth is four, as it was in round 4: S7 and S8b sit one wave low deliberately)

| Wave | Round 4 | Round 5 |
|---|---|---|
| 1 | SC ‖ S1a ‖ SP1 ‖ T1 | **SC-a** ‖ S1a ‖ SP1 ‖ T1 |
| 2 | S0 ‖ S1b ‖ S2 ‖ S4 | **SC-b** ‖ S1b ‖ S2 ‖ S4 |
| 3 | S3 ‖ S5 ‖ S6 ‖ S8a | **S0** ‖ S3 ‖ **S5** |
| 4 | S8b | **S6** ‖ **S8a** |
| 5 | S7 | S7 ‖ **S8b** |

Movements and their consequences, all recorded in the ownership map:
- **The wave-2 `cursor.go` collision (S0 ‖ S1b) is gone.** S1b lands first; S0 rebases onto it. Round 4 called this "the one real collision".
- **A new, textual one appears:** `docs/profiler-spec.md` is written by S0 and S5 in wave 3, in different sections. One rebase, no semantic conflict.
- **S8b moved 4 → 5** so it never shares a wave with S8a — both write `tests/test_guardrails.sh`, and S8b's named fallback exists precisely because they are one wave apart.
- **`profiler/enrich.go` gains a second writer:** S1b creates it (wave 2, with the call site), S7 deletes that call site (wave 5) when the two branches collapse to one. Three waves apart.

---

## 3. What round 5 deliberately did not do

- **Did not widen rules 5(a)/5(b) to the package.** Migrating `claude_code.go`, `codex.go` and `devin.go` onto `ApplyTokenSelection` is a change to three shipped adapters with their own tests, owned by no E-CS slice. Residue **H8**, recommended as a chore thread like H1.
- **Did not fix H3.** Suspected-inert; needs a CI log, not a roadmap edit.
- **Did not reopen** §G, the forks, the user questions, or any slice outside the fourteen findings and the split.
- **Did not add a per-figure `comparable` flag** (H4 stands, with its line range corrected).
