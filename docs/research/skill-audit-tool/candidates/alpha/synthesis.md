# Synthesis — "SonarQube for Agent Skills" (candidate: alpha)

**For:** an engineering team that has not seen the research. **Answers:** what to build, why, how we prove it works.
**Traceability key:** `L1`/`L2`/`L3` = the three lens reports in this directory · `OTel` = `docs/research/otel-genai-alignment.md` · `spec`/`Q#` = `cursor-profiler/spec.md` and `questions.md`. All dollar figures in the walkthrough are **placeholders** and labelled with their measurement tier.

## 1. The pitch in five lines

1. **What:** one audit loop for Agent Skills — a static rule engine (issues with remediation-minutes, reference graph, debt per skill) plus a paired-run dynamic engine (baseline vs refactored, with confidence intervals) feeding a portfolio ranker that answers "which skill do we fix next."
2. **For whom:** AI-native dev teams whose agents load `SKILL.md` packages — skill authors ("did my refactor help?"), DevEx leads ("what next?"), security reviewers ("what can it reach?"), managers ("what did it cost?") (L3 §3.1).
3. **Why it matters:** skills fail *silently* — never triggered, dead script paths, over-broad triggers (skill-architect `PRINCIPLES.md:9-15`) — and **27.2% of measured skills have negative lift** (ACES, via L2 §1.1). Unaudited skills can cost more than they save.
4. **Why now:** Claude Code already attributes per-request tokens and cost to `skill.name` (L2 summary 2), so activation frequency and cost-when-active are free telemetry; and the static landscape is crowded at spec-conformance but **empty at remediation-cost economics, cross-skill cycles, and proven before/after deltas** (L1 §3 verdict).
5. **The shape:** a standalone Go engine that works without SonarQube, emits Sonar generic-issue JSON for teams that have one, and reuses the security/duplication/spec layers that are already solved (L1 §4).

## 2. The workflow, end to end on one skill

Skill under audit: `deploy-helper` (placeholder). Harness: Claude Code, model pinned to a dated Opus 5 snapshot. Numbers below are illustrative; the *structure* is the contract.

### Step 0 — Static audit (seconds, free, exactly reproducible)

| Output | Value | Source of rule |
|---|---|---|
| `activation_tokens` (real `o200k_base` BPE, via skill-validator) | 6,800 — over the 5,000-token spec recommendation | L1 §2b |
| `always_loaded_tokens`, `deferred_tokens`, `reference_depth`, `branch_count` | 312 / 4,100 / 3 / 14 | L1 §2b |
| Findings with effort | `SK-R001` dead `scripts/rollback.sh` ref (10 min, Reliability/High) · `SK-M002` body >5k tok (90 min) · `SK-M003` ref depth 3 (45 min) · `SK-R004` weak description (15 min, Blocker) · `SK-S001` `curl\|sh` in `pull.sh` (30 min, **hotspot → review queue**) | L1 §4 min rule set |
| **Debt E = Σ effort** | 190 min = **3.2 h** | SQALE model, L1 §1 |
| Blast radius `B` | 1.5 (network egress + broad `allowed-tools`) | L3 §1.3 |

### Step 1 — Baseline: what it costs today (telemetry, no experiment)

| Term | Value | Source | Tier |
|---|---|---|---|
| `A` activations/user/week | 10 | `count(claude_code.token.usage) group by skill.name, user.id` | measured (L2 §4.1) |
| `U` users | 8 | distinct `user.id` | measured |
| `C_a` cost/activation | $0.50 | measured tokens × pinned price table | **estimated $** (L2 §3.3 tier 2) |
| `F` failure fraction | 0.01 | `tool_result` events with `error_type` | measured (L2 §2) |

Current run-rate: 10 × 8 × $0.50 = **$40/week ≈ $2,080/yr, estimated** — attribution, not yet a saving claim.

### Step 2 — Paired run (the counterfactual)

Design per L2 §1–§3: pilot of 10 arm-B reps to measure σ, then **N=20 should-fire + 12 should-not-fire + 6 neutral** tasks; arms **A** (no skill, run once and cached), **B** (v1 SHA), **C** (v2 SHA); model, effort, harness version, permissions, prompt bytes all pinned; arms interleaved.

| Result (placeholders) | v1 → v2 |
|---|---|
| Per-activation token delta | input −4,000 · cache-read −60,000 · cache-write −1,600 · output −2,000 |
| → per-activation $ delta | −(4000·5 + 60000·0.5 + 1600·6.25 + 2000·25)/10⁶ = **−$0.11/activation, estimated** (Opus 5 prices, L2 §4.3) |
| 95% BCa CI on per-task Δ$ | **[−$0.16, −$0.06]** — excludes 0 → the cost claim is *proved* (see §4) |
| Activation recall / precision | 0.80 → 0.90 / 0.84 → 0.90 — the saving is **not** a recall collapse (L2 §2.1) |
| Verifier pass rate | 15/20 → 17/20, McNemar n.s. — no quality regression inside tolerance |
| Sanity floor vs arm A | lift(B) = +0.18, lift(C) = +0.24 — both beat no-skill; the 27.2%-negative check passes |

### Step 3 — Dollars and the score

```
Δ$/activation −0.11 × A 10 × U 8 × 52wk  →  Waste$  = $458/yr  (CI [$250, $666])   [estimated]
A·U 80/wk × F 0.01 × B 1.5 × C_f $20     →  Risk$   = $1,248/yr                    [C_f declared assumption]
SDY = (Waste$ + Risk$) × c ÷ E = 1,706 × 1.0 ÷ 3.2 h →  $533 per engineer-hour per year
```

Honesty footnotes the tool prints automatically: naive "all input at $5/MTok" would report $0.33/activation — a **3.0× overstatement** (cache reads price at 0.1×, L2 §4.2); and trimming the listing description saves ~$2/yr (0.5% of the activation path) — listing size is a context-pressure play, not a cost play (L2 §4.3).

### Step 4 — Next best skill

Re-score the catalog (L3 §3.4 portfolio row):

| Skill | A·U /wk | C_a (est) | Debt (h) | Lane | Last measured lift ± CI | Action |
|---|---|---|---|---|---|---|
| `deploy-helper` | 80 | $0.50 | 0.4 left | — | +0.24 [0.19, 0.29] | merged; CI gate now watches it |
| `query-runner` | 140 | $0.30 | 1.5 | EV — r unmeasured | — | **top of queue: fund its baseline run** |
| `old-jira-sync` | 4 | $0.20 | 6.0 | **hotspot** (curl\|sh, broad tools) | — | security review first regardless of EV |

Two lanes is deliberate: SDY orders by dollars-per-hour, but a rarely-triggered skill bundling `curl|sh` still jumps the queue — SonarQube's own debt-vs-hotspot split (L3 §1.2, L1 §2e).

> **Same walkthrough on Cursor today:** everything through step 2 works on the hook spool, but tokens are tier-4 (`chars/4`), so the report renders "**relative estimate, not billed**," produces rankings and ordinals, and renders **no dollar figure** until a real token source exists (measurement contract, spec §; Q2).

## 3. Architecture

```
skill catalog ─► skaudit (static engine) ─► findings + effort-min + measures ─┐
                                                                             ▼
hooks/OTel ─► cursor-profiler / skill-architect profiler ─► spool ─► DuckDB ─► SDY scorer ─► portfolio (EV lane + hotspot lane)
                                                                             ▲
task set ─► experiment runner (arms A/B/C, verifier) ─► comparison + CI + MDE ─┘
                                                        └► lift + Δ$ ─► re-score ─► CI quality gate
```

| Component | New or reused | Role |
|---|---|---|
| `skill-architect/profiler` | **reused, needs repair** | `profile/v1` schema, capture adapters, `experiment.go`/`compare.go` — defects D-1–D-9 verified by execution (L2 §5.2) |
| `skill-audit` shell checks (PL/PT rules) | reused as rule inputs | existing static checks; folded into `SK-*` rules with effort minutes (L1 §2d) |
| `skill-validator` | reused | real `o200k_base` BPE counts, spec validation, content analysis (L1 §2d) |
| `SkillSpector` / `jscpd` | reused — vendor vs shell-out is an open decision (§7) | security hotspot patterns; cross-skill duplication (L1 §3) |
| `cursor-profiler` + DuckDB | reused (in flight per spec) | Cursor hook spool → JSONL source of truth → per-skill aggregates (A, U, F, C_a) via `read_json_auto` (Q9, R-AT-10) |
| OTel | reused two ways | *input:* Claude Code `skill.name` attribution; *output:* `gen_ai.*` names only behind a gated export layer — the semconv repo is unreleased, all `development` stability (OTel §1, §4b) |
| SonarQube | optional consumer | ingests generic-issue JSON; only if a server exists (§7 decision) |
| **`skaudit` static engine** | **new** — Go binary | rules + effort minutes, reference graph + cycle detection (Tarjan), new-code scoping; native JSON + generic-issue JSON + SARIF emitters (L1 §4) |
| **quality-profile config** | **new** — checked-in file | which rules/severities/efforts are on; we must own this because imported rules can't be managed inside SonarQube (L1 §4 named tradeoff) |
| **hotspot baseline file** | **new** | reviewed-state lifecycle for hotspots (SkillSpector's pattern, L1 §3) |
| **experiment runner v2** | **new** — extends `experiment.go` | three arms, stratified task sets, verifier hook, pooled bootstrap CI + achieved-MDE line (L2 §5.3 fixes 5–7) |
| **SDY scorer + portfolio report** | **new** | the two-lane ranker of §2 step 4 (L3 §1.2) |

## 4. The measurement contract

Non-negotiable (spec §Measurement contract; L2 §3.3):

| Tier | Signal | Rule |
|---|---|---|
| 1 Billed | Admin `cost_report`/`usage_report` — **unavailable to this team today** (Q2) | may be called "cost/spend" |
| 2 Measured tokens, estimated $ | harness token counters (`claude_code.token.usage`) | token deltas real; dollars labelled "estimated cost" |
| 3 Vendor-estimated $ | `claude_code.cost.usage` | carry the vendor's "estimated" label verbatim |
| 4 Self-estimated | `chars/4` over hook payloads | **ordinal only** — valid for "less than," never a dollar figure |

**"Proved" means:** the 95% paired-bootstrap BCa CI on per-task deltas excludes zero **and** precision/recall/verifier stay inside tolerance **and** lift vs the no-skill arm is ≥ 0 — reported as an interval plus the achieved MDE, never a point estimate (L2 §3.1–3.2, R-AT-07).

**The tool refuses to claim:** cross-tier deltas (refuses, not warns — R-AT-01) · dollar figures when either side is tier 4 · a token delta without the activation 2×2 · "no difference" when it means "no difference detected" (prints MDE instead) · `sum(tokens) by skill` as caused-cost (attribution ≠ counterfactual, L2 §2.1) · SkillReducer static-compression numbers as runtime savings (L3 §4) · significance from unguarded threshold-stopping (D-6, R-AT-08).

## 5. Build options considered

| Option | Gets you | Costs you | Verdict |
|---|---|---|---|
| (a) SonarQube Java language plugin | native rules page, profiles, debt, hotspot lifecycle | Java codebase in a Go shop, server dependency, a Markdown grammar, narrowest reach | rejected (L1 §4) |
| **(b) Standalone Go engine → generic-issue JSON primary, SARIF secondary** | `effortMinutes` survives into debt; runs with or without a Sonar server; CI-gateable exit codes | we own the profile config + hotspot-reviewed state; two emitters | **recommended** (L1 §4) |
| (b′) SARIF only | GitHub Code Scanning free | **disqualifying:** Sonar's SARIF importer forces `SECURITY` quality on every issue and has no effort field — destroys the debt model (L1 §4) | rejected |
| (c) Harness plugin only (today's shape) | zero infra, lives where authors work | no CI gate, no interop, single-skill contract can't see cycles/duplication | keep as thin UX over the binary |

Named tradeoff to accept now: imported external rules are invisible on Sonar's Rules page and can't be toggled in a quality profile — profile management stays in our config file by design (L1 §4).

## 6. The first three slices — independent, each proves something

Any order works; none blocks another. (Sequencing rationale: SDY is inert until static effort-minutes and per-activation cost exist — L3 §4.)

| # | PR | Contents | Proves |
|---|---|---|---|
| 1 | **Comparator honesty** (`skill-architect/profiler`) | Refuse cross-tier deltas and carry both sides' `MetricSource`; nil-guard `.Value`; compare `EstimatedContextTokens` tagged "relative estimate, not billed"; read `skill.name`, compare activation per task by name (D-1–D-5, R-AT-01–04). Already decided first in Q12/order amendment. | No report can launder an estimate again; a hook-only profile pair yields a labelled ordinal delta instead of `comparable:false` everywhere. |
| 2 | **`skaudit` v0** (new repo/binary) | Catalog-scope scan; `SK-*` rules with effort minutes; reference graph + cycle detection; native JSON + exit code. Zero runtime deps. | The unclaimed static layer works: debt-minutes on day one, plus a dead-ref and a cycle found in a fixture catalog that no existing tool detects (L1 §2c). |
| 3 | **`cursor-profiler analyze`** (DuckDB over spool) | `.sql` queries emitting per-skill aggregates: activation frequency, failure rate, estimated C_a (R-AT-10). | The ranking inputs are real: a portfolio shortlist (A×U×C_a) from captured data before any experiment machinery exists. |

Next after these (not part of the three): experiment runner v2 (arms/CI/verifier), SDY scorer + portfolio report, generic-issue emitter — each gated on its §7 decisions.

## 7. Risks and open decisions for the user

**Decisions needed** (from the lens reports' "above my level" sections; several already parked in questions.md):

| # | Decision | Source | Stakes |
|---|---|---|---|
| 1 | **Harness order:** Claude-Code-first gets tier-2 measured tokens and free `skill.name` attribution; Cursor-first stays tier-4 ordinal-only until an admin key or Enterprise OTel materializes | L2 dec. 1; Q2 | whether §2's dollar line is renderable at all today |
| 2 | **Buy billed data?** one Admin API key moves the product from "ranking tool" to "cost tool" (c 0.5→1.0) | L3 dec. 1; Q2 | every dollar figure's label |
| 3 | **Fund arm A and N:** arm A ≈ +50% experiment cost but makes "improved" falsifiable vs "delete it"; N≈12 detects an average lift, 216 a small one — at N=5 the tool should refuse significance claims | L2 dec. 3–4 | whether "proved" means anything |
| 4 | **Calibrate effort minutes + author-cost/line** — someone must time real fixes | L1 dec. 2 | uncalibrated SQALE = confident wrong dollars |
| 5 | **SkillSpector/jscpd: vendor, shell out, or reimplement a subset** | L1 dec. 3 | runtime deps vs owning 71 security patterns |
| 6 | **Single-skill → catalog scope** breaks `skill-audit`'s shipped contract and the `PL*/PT*` rule namespace | L1 dec. 4–5 | where the unclaimed value lives vs a user-visible break |
| 7 | **Hotspot lane: block merges or warn?** | L3 dec. 3 | day-one false positives vs unenforced security |
| 8 | **Own the org constants** (`C_f`, engineer-hour rate) and the **price table** staleness policy | L3 dec. 2; L2 dec. 5 | the Risk$/Waste$ weight; prices moved 2026-09-01 |
| 9 | **Cross-user aggregation** — Q10 keeps prompts/paths on disk; pooling them is a consent decision | L3 dec. 4 | portfolio-across-people vs privacy |
| 10 | **Show negative `r`?** 27.2% of skills hurt — a UI that only shows improvement hides deletion candidates | L3 dec. 5 | the most valuable finding is "delete this skill" |
| 11 | **S7 OTLP export** — every `gen_ai.*` alias is dead weight if unbuilt; only the two internal renames (`cache_write`, `error_type`) are worth doing regardless | OTel §4, dec. 1 | don't build the alias layer speculatively |
| 12 | **SonarQube: target or metaphor?** if nobody runs a server, defer the generic-issue emitter behind native JSON + SARIF | L1 dec. 1 | speculative surface area |

**Thin evidence flagged by the reports (do not design on it):** the hook-based activation heuristic is untested and is the sole source of `A` on Cursor (U-02) · whether `skill.name` fires for listed-but-not-invoked skills is unresolved — decides if false negatives are telemetry-detectable (L2 §handoff) · `effortMinutes`→`sqale_index` roll-up for imported issues is docs-assistant-asserted, not primary-page verified — check on a throwaway SonarQube instance first (L1 §handoff) · ClawHavoc malicious-skill counts conflict (1,184 vs 341) (L3 §2 R11) · `session.id` vs `gen_ai.conversation.id` hierarchy unresolved upstream (OTel S-8).

**Judgment calls this synthesis makes where inputs pull apart:**
- *Cursor vs Claude Code for dollars:* build harness-agnostic, but the prove-with-dollars path is Claude-Code-first; the Cursor path ships honest ordinals until a real token source exists (resolves L2 dec. 1 with Q2).
- *Single skill vs catalog:* the engine is catalog-capable from day one (cycles, duplication, trigger collision need the whole estate — L1 §2c, L3 §4); the *invocation* UX stays per-skill. The mandate's "run it against one specific skill" is a workflow, not a scope limit.
- *OTel path:* arena/mandate cite `.scuba/teams/otel-genai-alignment/report.md`, which does not exist; the real file is `docs/research/otel-genai-alignment.md` (L3 §4).
