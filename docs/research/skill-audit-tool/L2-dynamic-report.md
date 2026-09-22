# L2 — Dynamic analysis: proving a refactor made a difference

**Researcher lens L2** · 2026-09-11 · Evidence marks: **[O]** = observed by me (read the page/code/ran it), **[I]** = inferred/derived.

## Summary (the answer in five lines)

1. The field's paired design answers "is this skill worth having"; the user's question ("did my refactor help") needs a **three-arm** design — no-skill / v1 / v2 — because 27.2% of real skills have *negative* lift, so v2-beats-v1 does not mean v2 beats nothing **[O ACES]**.
2. **Claude Code already emits per-skill token and cost attribution natively**: `skill.name` is an attribute on `claude_code.token.usage` and `claude_code.cost.usage` **[O]**. This changes the plan — activation frequency and cost-when-active come for free; paired A/B is now needed only for the *counterfactual* (lift), not for attribution.
3. Tokens are **measured** (API `usage` block); dollars from any harness counter are **estimated** (local price table). Only `cost_report` is billed, and it has no skill or session dimension **[O]**.
4. Derived from ACES's published CI, per-case σ of Skill Lift ≈ **1.23× the mean effect** → ~**12 paired cases** to detect an average-sized lift at 80% power, ~**216** to detect a small one **[I, arithmetic shown]**.
5. `compare.go` cannot support any of this today: it launders estimate sources (verified by running it), never compares the estimate field at all, panics on a round-tripped profile, and compares activation *counts* only — so precision/recall is unreachable.

---

## 1. Paired evaluation design for one skill

Building on `skill-profiling-state-of-the-art.md` §1 (SkillsBench / ACES / SkillEvaluator Tier 3 — not repeated). The gap in that prior art: **all three compare skill-present vs skill-absent.** None compares skill v1 vs skill v2. Two additions are needed.

### 1.1 Three arms, not two

| Arm | Condition | Why it must exist |
|---|---|---|
| **A** | no skill installed | ACES: mean lift 0.2134 but **positive in only 72.8%** — 27.2% of skills hurt **[O]**. Without A you can ship a v2 that beats v1 and still loses to nothing. |
| **B** | skill v1 (baseline commit) | the thing being refactored |
| **C** | skill v2 (refactored commit) | the claim under test |

Primary endpoint is **C−B** (did the refactor help). **A** is the sanity floor, reported as `lift(C) = C−A` and `lift(B) = B−A`. A is amortizable: run it once per (task set × model) and cache it, re-run only when the task set or the pinned model changes **[I]**.

### 1.2 Hold constant / allow to vary

| Hold constant | Mechanism | Note |
|---|---|---|
| Model | dated snapshot id, never an alias | Claude Code OTel exposes `model` per request for post-hoc verification **[O]** |
| Effort / speed | pin; both are billed dimensions | `effort` and `speed` are attributes on `claude_code.cost.usage`; fast mode is priced $10/$50 vs $5/$25 on Opus 5 **[O]** |
| Harness + version | pin the CLI version | `mcp_server.name` semantics changed at v2.1.222 **[O]** — harness versions silently change telemetry |
| Repo state | fresh clone / worktree per run at a pinned SHA | SkillEvaluator Tier 3 uses two isolated sandboxes per case **[O]** |
| Task set & prompt text | byte-identical | |
| Tool permission set | identical allow-list | |
| Arm ordering | randomize *within* task, interleave arms in time | guards against endpoint drift during a long run **[I]** |

**Varies:** only the skill directory contents. **Cannot be held constant:** RNG. No `seed` parameter is documented on the Messages API, and even at temperature 0 an agentic run is non-deterministic because tool results feed back into context **[I]**. Determinism is not available; **replication is the only variance control.**

### 1.3 Task set composition — this is where activation precision/recall is bought

| Stratum | Size (recommended) | Purpose |
|---|---|---|
| **Should-fire** tasks | ≥ 15 | recall; and the lift estimate |
| **Should-not-fire** tasks (near-miss, adjacent-skill) | ≥ 10 | **precision** — the only way to catch a description refactor that made the trigger greedy |
| Neutral/unrelated tasks | ≥ 5 | detects listing-cost regressions and cross-skill interference |

A task set with no should-not-fire stratum cannot detect the most common refactor regression: broadening `description`/`when_to_use` to improve recall and silently wrecking precision **[I]**.

---

## 2. Metrics that can move

| Metric | Definition | Source | Billed or estimated |
|---|---|---|---|
| Tokens by type | input / output / cacheRead / cacheCreation | `claude_code.token.usage{type}`, or `claude_code.api_request` event fields **[O]** | **measured** (API usage block) |
| Cost | USD | `claude_code.cost.usage` — docs say **"estimated, not billed"** **[O]** | estimated |
| Billed tokens | same four classes | `/v1/organizations/usage_report/messages`, buckets `1m`/`1h`/`1d` **[O]** | **billed** |
| Billed dollars | USD cents | `/v1/organizations/cost_report`, **`1d` only**, group by workspace/description **[O]** | **billed** |
| Latency | wall clock per task | harness timing / `duration_ms` on tool_result **[O]** | measured |
| Tool-call count & success rate | count, `success`, `error_type`, `duration_ms` | `claude_code.tool_result` event **[O]** | measured |
| **Activation** | did `skill.name` appear on any request in the run | `skill.name` on `claude_code.{token,cost}.usage` **[O]** | measured |
| Task success | verifier pass/fail | your own deterministic verifier (SkillsBench pattern) **[O]** | measured |
| Context pressure | context occupancy | Cursor `preCompact.context_tokens` — the only real Cursor-reported token number **[O, prior research]** | Cursor-reported |
| Compaction events | count of compactions per task | `preCompact` firing count **[O, prior research]** | measured |
| Tool-acceptance rate | accepted/(accepted+rejected) per edit tool | Claude Code Analytics API **[O]** | measured, daily/per-user only |

### 2.1 Activation precision and recall — definitions and the cost of each error

Per skill, over the task set, with activation = `skill.name == <skill>` observed on ≥1 request in that run:

| | skill fired | skill did not fire |
|---|---|---|
| **should-fire task** | TP | FN |
| **should-not-fire task** | FP | TN |

`precision = TP/(TP+FP)` · `recall = TP/(TP+FN)` · report both, plus the raw 2×2, never a single F1 — the two errors have different dollar costs:

| Error | What it costs |
|---|---|
| **False positive** | the full `SKILL.md` body enters context, and Claude Code docs state the rendered body "**stays there across later turns**" **[O]** — so one bad activation is paid on every subsequent turn of that session, at cache-read price. Cost ≈ body_tokens × remaining_turns × cache-read rate. |
| **False negative** | the run silently falls back to arm A behaviour; you lose the entire lift on that task. Invisible in cost data — it looks like a *cheap* run. |

**This is why token delta must never be reported alone.** A refactor that destroys recall shows up as a large, attractive token saving.

**Crucial distinction — attribution ≠ counterfactual.** `skill.name` is documented as "**Skill active for the request**" **[O]**, so *all* tokens of that request are attributed to the active skill. That gives cost-**when**-active. It does **not** give cost-**caused**-by. Only the paired arms give the causal delta. Anything that reports `sum(token.usage) group by skill.name` as "what this skill costs you" is making a counterfactual claim the data does not support **[I]**.

---

## 3. Statistics for claiming a proved difference

### 3.1 Test choice

| Situation | Test | Why |
|---|---|---|
| Continuous metric (tokens, cost, latency), paired by task | **paired bootstrap BCa CI over per-task mean deltas** as primary; paired *t* only as a cross-check | token deltas are heavy-tailed and bimodal — published per-skill effects run **−76.9%** to **+120.3%** **[O]**; normality is not safe |
| Same, distribution-free backstop | Wilcoxon signed-rank / sign test | reports direction robustly when the CI is wide |
| Binary metric (pass/fail, activation) | **McNemar** on the paired 2×2 | SkillsBench-style verifier outcomes are paired by task |
| Reporting | CI, never a point estimate | ACES reports mean **0.2134, 95% CI [0.1967, 0.2301]** over 947 paired cases — the prior art already sets the bar at intervals **[O]** |

Pair **by task**, and report the per-task delta table alongside the grand mean. An aggregate mean over unequal-cost tasks can invert the per-task sign (Simpson) **[I]**.

### 3.2 Minimum detectable effect vs N

No primary source was found for run-to-run token variance in agentic coding harnesses — **no evidence found**; measure it in a pilot (§3.4). But σ for *Skill Lift* is recoverable from ACES's published interval:

> half-width 0.0167 → SE = 0.0167/1.96 = 0.00852 → **σ = 0.00852 × √947 = 0.262** → **σ/mean = 1.23** **[I, derived from [O] figures]**

With `MDE = (z₀.₉₇₅ + z₀.₈₀)·σ/√N = 2.8016·σ/√N`:

| N (paired tasks) | MDE (lift units) | as % of the average skill's lift (0.2134) |
|---:|---:|---:|
| 5 | 0.329 | 154% |
| 10 | 0.232 | 109% |
| **12** | 0.212 | **99%** |
| 20 | 0.164 | 77% |
| 30 | 0.134 | 63% |
| 50 | 0.104 | 49% |
| 100 | 0.074 | 34% |
| 200 | 0.052 | 24% |

Inverted: detecting an **average-sized** lift needs **N ≈ 12**; a *moderate* one (0.10) needs **54**; a *small* one (0.05) needs **216**; 0.02 needs **1,350** **[I]**.

**The operational consequence.** A refactor's C−B delta is by construction *smaller* than a whole-skill A/B lift. A team running 5 tasks × 3 reps is powered only to detect a change roughly the size of adding the skill in the first place. **The tool must print the achieved MDE next to every non-significant result**, so "no difference detected" is never misread as "no difference" **[I]**.

### 3.3 Honest reporting when tokens are estimated

The hierarchy, and the rule for each:

| Tier | Signal | Rule |
|---|---|---|
| 1. Billed | `cost_report` (USD, 1d) / `usage_report/messages` (tokens, 1m) **[O]** | may be called cost. No skill or session dimension exists — you must isolate arms by **API key** or by **disjoint 1-minute windows** to use it **[O, derived use]** |
| 2. Measured tokens, estimated dollars | `claude_code.token.usage` / `api_request` events **[O]** | token deltas are real; the dollar figure is Claude Code's local price table — label **"estimated cost"**, never "spend" |
| 3. Vendor-estimated dollars | `claude_code.cost.usage`, Analytics API `estimated_cost` **[O]** | carry the vendor's own "estimated" label through to the UI verbatim |
| 4. Self-estimated | `chars/4` over hook payloads **[O, prior research]** | **ordinal only.** Valid for "C used less than B on the same task"; invalid for any absolute or dollar claim |

Three non-negotiable rules **[I]**:
- **Never compare across tiers.** A tier-2 baseline minus a tier-4 candidate is not a delta. (`compare.go` does exactly this today — §5.)
- **Stamp every reported delta with the source of *both* sides**, not one.
- If any side is tier 4, the report must say **"relative estimate, not billed"** on the same line as the number, and must not render a dollar figure at all.

### 3.4 The pilot you must run before committing N

Run arm B alone, same task, **10 repetitions**. Compute per-task σ of the token delta empirically, then size N from §3.2's formula with the measured σ. Cost of the pilot is 10 runs; the cost of skipping it is an underpowered experiment whose null result is uninterpretable **[I]**.

---

## 4. Converting a measured delta to dollars

### 4.1 The formula and the provenance of every factor

`annual_$ = Σ_class ( Δtokens_class × price_class ) × activations_per_user_per_week × users × 52`

| Factor | Where it comes from | Tier |
|---|---|---|
| `Δtokens_class` | paired experiment, §1 — per token class, with a CI | measured |
| `price_class` | published price table, pinned by date **[O]** | billed rates |
| `activations_per_user_per_week` | `count(claude_code.token.usage) group by skill.name, user.id` **[O]** — *this is the factor `skill.name` makes free* | measured |
| `users` | seat count, or distinct `user.id` in OTel **[O]** | known |
| period | policy choice | assumption |

### 4.2 The error that dominates everything: pricing all input at base rate

In an agent harness most input tokens are **cache reads**, priced at **0.1× base** (0.025× on Fable/Mythos 5.1) **[O]**. Pricing a mixed input delta at the base rate overstates savings by up to 10×.

### 4.3 Worked example (placeholder deltas, real 2026-09-11 Claude Opus 5 prices [O])

Prices: input **$5/MTok**, 5m cache write **$6.25**, cache read **$0.50**, output **$25/MTok** **[O]**.
Placeholder measured per-activation delta (illustrative; would come from §1):

| Class | Δtokens | rate ($/MTok) | Δ$ |
|---|---:|---:|---:|
| base input | −2,000 | 5.00 | −0.0100 |
| cache read | −16,000 | 0.50 | −0.0080 |
| cache write | −500 | 6.25 | −0.0031 |
| output | −400 | 25.00 | −0.0100 |
| **per activation** | | | **−$0.0311** |

Naive "all 18,500 input tokens at $5/MTok" gives −$0.1025 — a **3.3× overstatement** **[I, arithmetic]**.

Scale-up, placeholders named as assumptions: 12 activations/user/week × 25 users × 52 weeks = 15,600 activations/yr → **$485/yr**.
Propagating a placeholder 95% CI of [−$0.045, −$0.017] per activation → **[$265, $702]/yr**. **Report the interval.**

**Counterintuitive result worth telling the team:** the same refactor also trims the always-loaded listing description by ~28 tokens. At 400 turns/user/week × 25 users × 52 weeks × cache-read price that is **$7.28/yr** — 1.5% of the activation-path saving **[I]**. Listing-size optimization is a context-pressure play, not a cost play.

**Latency → dollars:** convertible only with a *blocking factor* (what fraction of agent wall-clock actually blocks the engineer). That factor is an assumption, not a measurement; state it explicitly or omit the conversion **[I]**.

---

## 5. Gap analysis vs `skill-architect/profiler/experiment.go` and `compare.go`

All line references observed in the working tree on 2026-09-11.

### 5.1 What already exists and is sound

| Capability | Location |
|---|---|
| Declared-before-run design: task families, repetitions, ordering, budget, quality tolerance, stopping rule | `experiment.go:41-55` **[O]** |
| Validation with required snapshot hash + skill dir on both arms | `experiment.go:121-132` **[O]** |
| Plan materialization, one baseline + one candidate step per (task, rep) | `experiment.go:177-191` **[O]** |
| Execution + per-run comparison + report write | `experiment.go:247-288` **[O]** |
| Explicit metric-state contract so missing data is never silently zero | `types.go:11-15`, `compare.go:107-112` **[O]** |
| Mismatch notes for harness / snapshot / skill_dir | `compare.go:80-88` **[O]** |

The honesty contract is genuinely well-designed at the *type* level (`types.go:43-45, 171-183` keep `chars/4` out of `TokenCounts` on purpose) **[O]**. The comparison layer then breaks it.

### 5.2 Defects — verified by running the code

I built a throwaway module against the package and ran three cases. Results:

**D-1 (the mandate's known defect) — `compareTokens` launders estimate sources.** `compare.go:115` sets `Source: candidate.Source` and **discards `baseline.Source` entirely**. There is no source-mismatch note, though the mechanism exists for harness/snapshot/skill_dir at `compare.go:80-88`. Observed output for a `session_data` baseline vs an `otel` candidate:

```
{"comparable": true, "source": "otel",
 "delta": {"input": -30000, "output": -200}}   // notes: []
```

A −30,000-token delta computed across two different measurement methods, stamped with one of them, with no warning **[O, executed]**. Identical bug at `compare.go:141, 180, 193, 210` for tool calls, timing, activation and attribution. Under §3.3 this is a tier-crossing comparison presented as a single-tier one.

**D-2 — the honest estimate field is never compared at all.** `CompareProfiles` compares exactly five metrics (`compare.go:90-94`); `EstimatedContextTokens` (`types.go:203`) is not among them. `spool_profile.go:113` is the **only** producer of that field. So on a hook-only harness — the common free case — a paired run where both sides carry a clean `hooks_estimated` figure returns `comparable: false` on every metric and discards the delta **[O, executed]**. The tool cannot compare the one number it can actually get.

**D-3 — nil-pointer panic on a round-tripped profile.** `LoadProfile` (`compare.go:44-57`) validates only the schema string. A JSON profile with `{"state":"present"}` and no `"value"` key loads cleanly with `Value == nil`, then `compare.go:119` dereferences it: `runtime error: invalid memory address or nil pointer dereference` **[O, executed]**. Same shape at `compare.go:181`. Any hand-written or third-party profile crashes the comparator.

**D-4 — a hardcoded false negative on the highest-value signal.** `claude_code.go:104` sets `UnknownActivationResult("Claude Code has no skill-level activation events")`. That is **now factually wrong**: `skill.name` is a documented attribute on `claude_code.token.usage` and `claude_code.cost.usage` **[O]**. `grep` confirms the adapter reads neither `skill.name` nor any cost metric **[O]**. The profiler is blind to the one first-party per-skill attribution surface that exists.

**D-5 — activation is compared as a bare count.** `compare.go:185-199` compares `len(Value)` only — not skill names, not per-task. Precision/recall (§2.1) is structurally unreachable from this schema **[O]**.

**D-6 — unguarded optional stopping.** `experiment.go:156` accepts `stopping_rule: "threshold"` with no alpha-spending or minimum-N guard. Repeatedly peeking and stopping when a threshold is crossed inflates the false-positive rate; as written it is a significance-manufacturing machine **[I]**.

**D-7 — no aggregation across repetitions.** `RunPlan` emits one independent `ComparisonReport` per (task, rep) (`experiment.go:249-269`) **[O]**. There is no pooling, no CI, no test — `Repetitions` (`experiment.go:112-114`) buys N but nothing consumes it. Everything in §3 is missing.

**D-8 — two arms only.** `Steps = []ExperimentStep{baseStep, candStep}` (`experiment.go:188`) hardcodes two conditions **[O]**. Arm A (§1.1) cannot be expressed.

**D-9 — no verifier, no dollars.** No pass/fail grading hook and no price table anywhere in the package **[O]**. Without §1.3's verifier a token saving cannot be distinguished from a capability loss; without a price table §4 cannot run.

### 5.3 Fix order (cheapest first, highest risk first)

| # | Fix | Defect |
|---|---|---|
| 1 | Refuse to compare when `baseline.Source != candidate.Source`; carry both sources in the report | D-1 |
| 2 | Nil-guard every `.Value` deref, or validate value-presence in `LoadProfile` | D-3 |
| 3 | Compare `EstimatedContextTokens`, output tagged `relative estimate, not billed` | D-2 |
| 4 | Read `skill.name` from Claude Code OTel; delete the hardcoded claim | D-4, D-5 |
| 5 | N-arm plan + per-task pairing + pooled bootstrap CI + achieved-MDE line | D-7, D-8 |
| 6 | Verifier hook and pinned price table | D-9 |
| 7 | Guard or delete `stopping_rule: threshold` | D-6 |

---

## Decisions needed above my level

1. **Does the product target Claude Code first?** `skill.name` attribution exists in Claude Code and, per prior research, nothing equivalent exists for Cursor outside Enterprise. Claude-Code-first makes items 2/4 cheap; Cursor-first keeps the tool on tier-4 estimates. This is a scope call, not a research finding.
2. **Billed or estimated dollars?** Billed requires an Admin API key (unavailable to individual accounts **[O]**) plus per-arm API-key or 1-minute-window isolation. Estimated ships today but can never be called "spend".
3. **Is the team willing to pay for arm A?** It roughly increases experiment cost by 50% (amortizable). Dropping it makes "we improved the skill" unfalsifiable against "delete the skill".
4. **What N is the team willing to fund?** §3.2 says ~12 paired tasks for a large effect, ~216 for a small one. If the answer is "5", the tool should refuse to print a significance claim.
5. **Who owns the price table** and its staleness policy? Prices moved this year (Sonnet 5 introductory pricing became standard on 2026-09-01 **[O]**).

## Handoff to synthesis

- **To L1 (static):** the always-loaded listing cost is fully computable statically and needs no experiment — but §4.3 shows it is ~1.5% of the activation-path saving. Don't let a static "token weight" rule become the headline metric. Also: `skill-validator check -o json` gives exact o200k_base counts; `analyze content` does not **[O, prior research]** — the static side should call the right subcommand.
- **To L3 (prioritization):** `count(claude_code.token.usage) group by skill.name, user.id` gives *activation frequency* and *cost-when-active* per skill directly from OTel, with no experiment **[O]**. That is two of L3's three scoring inputs for free. The third — achievable reduction — is the only one that needs L2's paired machinery. Recommend L3's portfolio view run off OTel and reserve experiments for the top-ranked few. Also carry §2.1's warning: `sum by skill.name` is attribution, not causation.
- **Unresolved for synthesis:** whether `skill.name` is emitted for skills that were *listed but not invoked* (it is documented as "absent when no skill is active" **[O]**, implying listing-only cost is invisible) — that determines whether false-*negative* activations can be detected from telemetry alone, or only via the task set.

## Sources

- Claude Code monitoring / OTel attributes (`skill.name`, `claude_code.token.usage`, `claude_code.cost.usage`, `api_request`, `tool_result`, cost is "estimated, not billed") — <https://code.claude.com/docs/en/monitoring-usage> **[O, fetched twice, attribute rows quoted]**
- Pricing (Opus 5 $5/$6.25/$0.50/$25 per MTok; cache multipliers 1.25×/2×/0.1×; Sonnet 5 pricing note) — <https://platform.claude.com/docs/en/about-claude/pricing> **[O]**
- Usage & Cost Admin API (bucket widths `1m`/`1h`/`1d`; cost `1d` only; group-by dimensions; unavailable to individual accounts) — <https://platform.claude.com/docs/en/manage-claude/usage-cost-api> **[O]**
- Claude Code Analytics API (daily, per-user, `estimated_cost`, no skill dimension) — <https://platform.claude.com/docs/en/manage-claude/claude-code-analytics-api> **[O]**
- Claude Code skills (body "stays there across later turns", listing budget, 1,536-char cap) — <https://code.claude.com/docs/en/skills> **[O via prior research]**
- ACES (lift 0.2134, 95% CI [0.1967, 0.2301], N=947, 72.8% positive) — <https://arxiv.org/abs/2608.20614> **[O via prior research]**
- SkillsBench — <https://arxiv.org/abs/2602.12670> · NVIDIA SkillEvaluator (Tier 3 two isolated sandboxes; −76.9% / +120.3% token effects) — <https://developer.nvidia.com/blog/evaluating-ai-agent-skill-performance-with-nvidia-skillevaluator/> **[O via prior research]**
- Code: `/Users/matthewvandusen/Development/Auraprix/skill-architect/profiler/{compare.go,experiment.go,types.go,claude_code.go,spool_profile.go}` **[O, read + executed]**
