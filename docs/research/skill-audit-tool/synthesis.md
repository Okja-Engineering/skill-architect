# Skill Audit: what we should build, why, and how we prove it works

**Merged synthesis · 2026-09-11 · base candidate B, fold-ins from C and A (see Provenance).**

Citation keys: **[L1 §x]**, **[L2 §x]**, **[L3 §x]** = `.scuba/teams/skill-audit-tool/{L1-static,L2-dynamic,L3-prioritization}-report.md` · **[OT §x]** = `.scuba/teams/otel-genai-alignment/report.md` · **[Qn]** = `questions.md` · **[spec:n]** = `spec.md`. All are symlinks into `docs/research/`; all exist and were read on 2026-09-11.

---

## 1. The pitch in five lines

1. **What:** `skillaudit` — a single-binary static rule engine for Agent Skills that prices every finding in *remediation minutes*, plus honesty repairs to `skill-architect/profiler` so a before/after refactor yields a delta with a confidence interval instead of a vibe.
2. **For whom:** the skill author ("did my refactor help?"), the platform lead ("what do I fix next?"), the security reviewer ("what can this skill reach?") **[L3 §3.1]**.
3. **Why now:** spec conformance is solved (agnix, 455 rules), security is solved (SkillSpector, 71 patterns, 16,984★), duplication is solved (jscpd). **Remediation-cost economics, cross-skill reference graphs with cycle detection, and portfolio ranking are unclaimed** **[L1 §3]** — exactly what was asked for.
4. **Why us:** Claude Code emits `skill.name` on `claude_code.token.usage` **[L2 §2]**, so activation frequency and cost-when-active are free; our `profiler` already has a metric-state contract that refuses to invent data (`types.go:11-15`) **[L2 §5.1]**.
5. **The bet:** dollars come from minutes and from measured token deltas. What cannot trace to one of those, we refuse to print — and until the minutes are *calibrated*, we refuse to convert them to a letter grade at all.

---

## 2. Walkthrough: `pr-review`, baseline → refactor → prove → next best

`pr-review`, `deploy-helper`, `commit-msg`, `test-writer` are **fictional**. Every number is a **placeholder** marked *(assumed)*, *(measured)* or *(billed)*, internally consistent and recomputed end to end. No cell describes a real skill.

### 2a. Baseline — static scan (no model calls, no telemetry)

| Finding | Rule **[L1 §4]** | Quality / severity | Effort |
|---|---|---|---:|
| `SKILL.md` body 6,400 tokens (> 5,000 budget) | `SK-M002` | Maintainability / Medium | 90 min |
| `references/rubric.md` → `style.md` (depth 2) | `SK-M003` | Maintainability / Medium | 45 min |
| `scripts/gen.sh` referenced, absent on disk | `SK-R001` | Reliability / High | 10 min |
| 64% description overlap with `code-review` | `SK-R005` | Reliability / Medium | 30 min |
| Missing `## Constraints` heading (house policy) | `SK-M005` | Maintainability / Low | 5 min |
| **Total debt** | | | **180 min = 3.0 h** |

Token measures are exact `o200k_base` BPE from `skill-validator check -o json` **[L1 §2b]**: `always_loaded` 118 · `body` 6,400 · `deferred` 3,200.

**Debt ratio and A–E rating are withheld, deliberately.** Sonar's ratio is `Σ effort ÷ (author-cost-per-line × lines)` with a default 30 min/line **[L1 §1]**, which L1 calls "certainly wrong for skills" **[L1 §4]**. A letter from a guessed denominator is the most confident-looking wrong number this tool could emit. We ship **minutes**; the letter is earned via the calibration ledger (§3, refusal **F4**).

### 2b. Baseline — telemetry, and the pre-ranking

| Term **[L3 §1.3]** | Value | Source | Tier **[L2 §3.3]** |
|---|---:|---|---|
| `A` activations/user/week | 12 *(measured)* | `count(claude_code.token.usage) group by skill.name, user.id` **[L2 §4.1]** | measured |
| `U` users | 25 *(measured)* | distinct `user.id` | known |
| activations/year | **15,600** | 12 × 25 × 52 | — |
| `C_a` $/activation | $0.34 *(estimated)* | measured tokens × pinned price table | **tier 2** |
| attributed spend/yr | **$5,304** | 15,600 × 0.34 | tier 2 |
| `r̂` modeled reduction | 0.10 *(assumed)* | static headroom 1,400 tok. **Not** SkillReducer's 39/48% **[L3 §1.3]** | modeled |
| `Waste$` | **$530/yr** | 5,304 × 0.10 = 530.40 | — |
| `F` failure rate | 0.004 *(assumed)* | spool `postToolUseFailure` **[L3 §1.3]** | inferred |
| `B` blast radius | 1.25 *(assumed)* | one bundled script, no egress **[L3 §2 R8/R9]** | observed (static) |
| `C_f` cost per failure | $40 *(declared)* | engineer-hour × rework minutes | declared assumption |
| `Risk$` | **$3,120/yr** | 15,600 × 0.004 × 1.25 × 40 | — |
| `c` confidence | 0.5 | rule below | derived |
| `E` remediation | 3.0 h | §2a, 180 min | modeled |
| **SDY** = (Waste$+Risk$)×c ÷ E | **$608/eng-hour** | 3,650.40 × 0.5 ÷ 3.0 = 608.40 | — |

**The one rule that sets `c`**, governing every table here:

| `c` | Condition |
|---:|---|
| 1.0 | tier-1 **billed** dollars — never today, no Admin API key **[Q2]**. We cap at 0.8 |
| 0.8 | tier-2 estimated dollars **and** `r` from a completed paired experiment whose CI excludes 0 |
| 0.5 | tier-2 estimated dollars, `r̂` modeled |
| — | tier 3 or 4 on either side: no dollar is rendered, so no `c` exists |

This overrides **[L3 §1.3]**'s `otel` = 1.0. OTel gives *measured tokens*; the dollars remain our own price table, so 1.0 would claim billing accuracy we do not have.

**Do not reuse L3's worked example.** Its stated Waste$ ≈ $470/yr and Risk$ ≈ $920/yr do not reproduce from its own inputs (`A`=12, `U`=8, `C_a`=$0.34, `r`=0.22, `F`=0.06, `B`=1, `C_f`=$40, `c`=0.5, `E`=3 h). Correct: Waste$ = 12×8×0.34×0.22 = **$7.18/wk = $373/yr**; Risk$ = 12×8×0.06×1×40 = **$230.40/wk = $11,981/yr**; SDY = (373+11,981)×0.5÷3 = **≈$2,059/eng-hour**, not $232. L3 itself says "do not take the worked example's numbers" **[L3 handoff]**. Shape adopted; numbers re-derived.

`pr-review` ranks **#1 in the EV lane** (§2e). We work it.

### 2c. Refactor, then prove

Split the body behind one reference level (which also flattens the depth-2 chain) and delete the dead script ref. Body 6,400 → 4,950 tokens (**static delta −1,450, exact and free** **[L1 handoff]**); listing −28 tokens.

Paired experiment **[L2 §1]**: three arms (A no-skill / B v1 / C v2 — 27.2% of skills have negative lift **[L2 §1.1]**), 35 tasks = 20 should-fire + 10 should-not-fire + 5 neutral **[L2 §1.3]**; model pinned to a dated snapshot; 10-rep pilot first for σ **[L2 §3.4]**.

| Class | Δ tokens/activation *(measured)* | $/MTok **[L2 §4.3]** | Δ$ |
|---|---:|---:|---:|
| base input (1,450 body + 550 removed ref read) | −2,000 | 5.00 | −0.0100 |
| cache read (1,450 carried × ~11 later turns **[L2 §2.1]**) | −16,000 | 0.50 | −0.0080 |
| cache write | −500 | 6.25 | −0.0031 |
| output | −400 | 25.00 | −0.0100 |
| **per activation** | | | **−$0.0311** *(estimated $, measured tokens)* |

**The naive number, correctly labelled.** L2's "$0.1025 naive" is **not** "all 18,500 input tokens at base rate": 18,500 × $5/MTok = **$0.0925**; the extra $0.0100 is the 400 *output* tokens at $25/MTok. Pricing every class naively gives $0.1025, a **3.3×** overstatement; pricing only the input-side tokens at base gives $0.0925, a **3.0×** one. Same mechanism either way — cache reads are **0.1× base** **[L2 §4.2]**. L2's own label is wrong **[L2 §4.3]**; corrected here.

**Proved?** Paired bootstrap BCa 95% CI on per-task mean deltas: **[−$0.045, −$0.017]** — lower bound excludes 0 → **regression-free improvement, proved at 95%** **[L2 §3.1]**. Achieved MDE at N=20 = 0.164 lift units = 77% of the average skill's whole-skill lift **[L2 §3.2]**, printed beside the result. Activation recall 20/20, precision 20/22 — **reported alongside the token saving, never without it** **[L2 §2.1]**.

Annualized: 15,600 × $0.0311 = **$485/yr, CI [$265, $702]** (15,600 × 0.017 and × 0.045). From the unrounded −$0.031125 it is $486 — a $1 swing, which is why the interval is the published figure. The listing −28 tokens is worth **$7.28/yr** (400 turns/user/wk × 25 × 52 × 28 tok × $0.50/MTok) — 1.5% of the activation-path saving **[L2 §4.3]**. Listing size is a context-pressure play, not a cost play.

Measured `r` = 0.0311 ÷ 0.34 = **0.092** vs modeled `r̂` = 0.10. Loop closed; `c` moves 0.5 → **0.8**.

### 2d. Re-score

| | Pre | Post |
|---|---:|---:|
| Debt (minutes; letter withheld) | 180 min | **35 min** — `SK-R005` 30 + `SK-M005` 5 remain; M002/M003/R001 fixed |
| `C_a` | $0.34 | **$0.3089** (0.34 − 0.0311) |
| `r̂` residual | 0.10 | 0.02 *(assumed)* |
| `Waste$` | $530 | **$96** (15,600 × 0.3089 × 0.02 = 96.38) |
| `Risk$` (dead ref fixed → `F` 0.001, `B` 1.0) | $3,120 | **$624** (15,600 × 0.001 × 1.0 × 40) |
| Recoverable total | $3,650 | **$720** |
| `c` / `E` | 0.5 / 3.0 h | 0.8 / **0.583 h** (35 ÷ 60) |
| SDY | **$608**/h | **$988**/h ← **the ratio pathology** |

Every post cell uses the **post** `C_a` and the **unrounded** `E`: 720.38 × 0.8 ÷ 0.58333 = **988.0**.

SDY *rises* after a successful fix because `E` shrank, and a skill can game it by splitting one finding into slivers. **Our fix, a departure from L3 as written:** an **absolute floor** — drop any skill whose recoverable total is under $1,000/yr *(org-set)* from the EV lane regardless of SDY. $720 < $1,000, so `pr-review` leaves the queue.

### 2e. Next best skill

SDY computed from unrounded terms. Last two columns are the `C_f` sensitivity: SDY if the org constant halves to $20 or doubles to $80.

| Lane | Skill | acts/yr | `C_a` | `r̂` | Waste$ | `F`·`B` | Risk$ | `c` | `E` | **SDY** | @½`C_f` | @2×`C_f` | Verdict |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| **Hotspot** | `deploy-helper` | 650 | $0.21 | 0.10 | $13.65 | 0.01·3.0 | $780 | 0.5 | 3.0 h | **$132** | $67 | $262 | **First, regardless of SDY** — `curl \| sh` + broad `allowed-tools`, `B` = 3.0 **[L3 §1.2, §2 R8/R9]** |
| EV | `commit-msg` | 26,000 | $0.09 | 0.08 | $187.20 | 0.001·1.0 | $1,040 | 0.5 | 1.5 h | **$409** | $236 | $756 | next |
| EV | `test-writer` | 7,800 | $0.52 | 0.15 | $608.40 | 0.002·1.0 | $624 | 0.5 | 2.0 h | **$308** | $230 | $464 | then |
| — | `pr-review` (post) | 15,600 | $0.3089 | 0.02 | $96.38 | 0.001·1.0 | $624 | 0.8 | 0.583 h | **$988** | $560 | $1,844 | recoverable $720 < $1,000 floor → **out** |

`deploy-helper` is **last on EV and goes first anyway**: the hotspot lane is ordered by `B` × permission scope, not by SDY **[L3 §1.2]**. That two-lane split is Sonar's debt-vs-hotspot split, borrowed intact **[L1 §1, L3 §3.4]**.

**Sensitivity, structurally.** `Risk$` dominates `Waste$` in every row, and `Risk$` is two *declared assumptions* (`F`, `C_f`) times one static input (`B`). So the report **always** prints Waste$ and Risk$ separately plus the sensitivity pair, and **refuses to sum them into a headline dollar** (**F8**): SDY is a ranking key, never a claim about money. At ½`C_f` the gap between `commit-msg` and `test-writer` collapses from $101 to **$6** — a ranking that close is not a ranking, and the tool says so.

---

## 3. Architecture

```
skills/*/SKILL.md ──► skillaudit scan ──► report/v1.json  (findings + effortMinutes + quality/severity + graph)
                          │                    ├─► calibration.jsonl  (append: observed fix minutes)
                          │                    ├─► sonar generic-issue JSON  (flag)
                          │                    └─► SARIF 2.1.0               (flag)
Cursor hooks ─► cursor-profiler ingest ─► JSONL spool ─┐
Claude Code OTel (skill.name) ────────────────────────┤
                                                      ├─► DuckDB read_json_auto ─► A, U, C_a per skill
profiler capture ─► profile/v1 ──► profiler compare ──┤
                                   (N-arm experiment)  └─► skillaudit rank ─► portfolio + hotspot lanes
```

| Component | Reuse / new | Role | Boundary it must not cross |
|---|---|---|---|
| `skill-audit` skill + `check-*.sh` | **reuse as policy, supersede in code** | PL001–PT002 become `SK-*`; `SK-R001` resolves the `$var` paths `check-paths.sh:50,71` skips **[L1 §2d]** | Cannot see across skills (`SKILL.md:33`) — the new engine must **[L1 §2d]** |
| `skill-validator check -o json` | reuse (external) | real `o200k_base` counts; `analyze content` does not give them **[L2 handoff]** | **Never invent a token count when absent** |
| **`skillaudit` (new, Go, zero-dep)** | **new** | rule engine w/ `effortMinutes`, reference graph + Tarjan cycles, catalog inventory, emitters | **Never prices anything.** Minutes and tokens only |
| **`calibration.jsonl` (new)** | **new** | per fixed finding: `(rule_id, finding_id, first_seen_commit_ts, fixed_commit_ts, observed_minutes)` from the skill dir's git history | **Records observed minutes; never estimates them.** Append-only |
| `cursor-profiler` ingest/spool/capture | reuse **[spec:90, spec:164]** | lossless JSONL capture, `profile/v1` emission | Redacts secrets before disk **[Q10]**; `profile/v1` stays additive-only **[spec:38]** |
| DuckDB over `spool/*.jsonl` | reuse **[Q9]** | query engine for `A`, `U`, `C_a`, `F`; JSONL is truth, `.duckdb` a rebuildable cache | **No ETL, no server, no daemon** **[Q9]** |
| `skill-architect/profiler` | **reuse after repair** | paired runs, N arms, pooled CI; 9 defects catalogued **[L2 §5.2]**. Carries **R1** (`CacheCreation`→`CacheWrite`, JSON `cache_creation`→`cache_write`) and **R2** (`ToolCallEntry.ErrorType`) **[OT R1, R2]** | **Refuses to compare across metric sources** |
| **`scoring` (new)** | **new** | SDY, two lanes, floor, `c` per §2b | **Never reads a file.** Takes report/v1 + query output only |
| **price table (new)** | **new** | pinned, dated `$/MTok` per class **[L2 §4.1]** | Dated and owned; staleness is a bug **[L2 decisions §5]** |
| OTel `gen_ai.*` | **export-time alias only, if ever** | `gen_ai.skill.name` is unmerged PR #498; nothing named "skill" in `main`; registry is `development`, zero releases **[OT §1, G1, S-2]** | **No `gen_ai.*` literal in `types.go`** **[OT S-1]**; no OTLP export today **[Q4]** |
| SkillSpector / jscpd / skillscore | **shell out, opt-in, off by default** | security patterns, duplication, 7-dimension score **[L1 §3]** | Degrade to `unknown` when absent, never abort. Conflicts with **[Q8]** — see §7 |

**The calibration ledger** answers the one question L1 escalated above its own level: who calibrates `effortMinutes` and the author-cost denominator **[L1 decisions §2]**. Make it a measurement, not a guess. At **n ≥ 20** observed fixes for a rule, that rule's `effortMinutes` switches from the shipped prior to the team's observed **median**; the same ledger yields the denominator. Until then the letter and the ratio are suppressed (**F4**). Cost: one JSONL writer.

**The two OTel renames** pay off regardless of whether `gen_ai.skill.*` ever lands — the OTel report's own bottom line **[OT §6]**. R1: upstream renamed it in breaking change #440 *and* Cursor's Admin API already spells it `cacheWriteTokens` **[OT R1]** — two of three surfaces agree with semconv. R2: semconv has no success boolean; failure is the *presence* of `error.type`, and Cursor hands us `failure_type`/`error_message` on `postToolUseFailure` that `success: false` discards **[OT R2]**. **Caveat A did not make:** R2 is additive and in scope under **[spec:38]**; R1 is a *rename*, free only because the field is `omitempty` and no source populates it today **[OT R1]**. Take it now or never.

**Data store:** there isn't a new one. JSONL spool + DuckDB + `report/v1.json` per scan + `calibration.jsonl`. Trend is the spool's history; no database server **[Q9]**.

---

## 4. The measurement contract

### 4a. The four tiers **[L2 §3.3]**

| Tier | Signal | May be called | Rule |
|---|---|---|---|
| 1 | Admin API `cost_report` (1d) / `usage_report/messages` (1m) | **billed spend** | No skill or session dimension — arms isolated by API key or disjoint 1-min windows. **Unavailable today** **[Q2]** |
| 2 | `claude_code.token.usage` / `api_request` | **measured tokens, estimated dollars** | The dollar is a local price table. Never the word "spend" |
| 3 | `claude_code.cost.usage`, Analytics `estimated_cost` | vendor-estimated | Carry the vendor's label verbatim |
| 4 | `chars/4` over hook payloads **[Q5]** | **relative estimate, not billed** | **Ordinal only. No dollar figure at all** |

Three non-negotiables **[L2 §3.3]**: never compare across tiers; stamp **both** sides of a delta with their source; a tier-4 side forbids a dollar. Said plainly: **on today's environment the word "billed" never appears in the product.** `compareTokens` violated all three as L2 observed it — `Source: candidate.Source`, discarding the baseline's, at `compare.go:115`, no mismatch note though the mechanism existed at `compare.go:80-88` **[L2 §5.2 D-1, executed]**. See §6: partly repaired already.

### 4b. What "proved" means

Proved only when **all four** hold; otherwise the verdict is "no difference detected at MDE = X".

| # | Condition | Test / threshold |
|---|---|---|
| 1 | Cost/token delta is real | **paired bootstrap BCa 95% CI** on per-task means excludes 0 (Wilcoxon backstop). Deltas run −76.9% to +120.3%, so normality is unsafe **[L2 §3.1]** |
| 2 | Task success did not regress | **McNemar** on the paired 2×2, p < 0.05 **[L2 §3.1]** |
| 3 | Activation **precision does not fall** | measured on the should-not-fire stratum — the only way to catch recall bought by wrecking precision **[L2 §2.1]** |
| 4 | **Arm-A floor**: `lift(C) > 0` | v2 must beat *no skill*, not just v1 — 27.2% of skills have negative lift **[L2 §1.1]** |

Nulls print their **achieved MDE** **[L2 §3.2]**: "no difference detected" ≠ "no difference". Size N from a 10-rep pilot's σ **[L2 §3.4]**; from ACES's σ/mean = 1.23 **[L2 §3.2]**, an average effect needs **N≈12**, moderate 54, small 216. A refactor delta (C−B) is smaller than a whole-skill lift by construction, so below N=12 the tool prints the MDE and refuses any significance claim.

### 4c. What the tool refuses to claim

Emitted as a machine-readable `refusals[]` array in every report, not merely as policy.

| # | Refusal | Why |
|---|---|---|
| F1 | No delta when `baseline.Source != candidate.Source` | **[L2 §5.2 D-1]**; PR2 |
| F2 | `sum(tokens) group by skill.name` is never "what this skill costs you" | Attribution, not causation: `skill.name` means "active for the request" — all its tokens, caused or not **[L2 §2.1]** |
| F3 | No dollar when either side is tier 4 | §4a |
| **F4** | **No letter rating and no debt ratio before n ≥ 20 observed fixes per rule in play** | Denominator uncalibrated; Sonar's 30 min/line is certainly wrong for skills **[L1 §4]**. Minutes ship meanwhile |
| F5 | `r` is never seeded from SkillReducer's 39%/48% | Static compression, explicitly not runtime saving **[L3 §1.3]** |
| F6 | No token saving without activation precision/recall beside it | A refactor that destroys recall looks like an attractive saving **[L2 §2.1]** |
| F7 | No `chars/4` emitted as `gen_ai.usage.input_tokens`, nor `context_tokens` as usage | Reserved for provider-reported, billing-aligned counts **[OT R22, R23]** |
| F8 | No blended SDY as a headline dollar; Waste$/Risk$ always separate, with the `C_f` sensitivity pair | A blended SDY is an opinion about `C_f` wearing a dollar sign (§2e) |
| F9 | No absolute SDY presented as money until `C_a` is billed | Ranking is meaningful; the absolute is not **[L3 §1.2]** |
| F10 | Negative results are shown, including "delete this skill" | 27.2% negative lift **[L2 §1.1]**; a UI showing only improvement hides the best finding |
| F11 | No significance claim below the funded N | **[L2 decisions §4]** |

---

## 5. Build options and the recommendation

| # | Option | Gets you | Costs you |
|---|---|---|---|
| a | **SonarQube Java language plugin** | Native rules page, profiles, debt/rating, hotspot lifecycle, trend — free | Java in a Go shop; a Markdown grammar; a Sonar server as a hard dependency **[L1 §4]** |
| b | **Standalone engine, Sonar generic-issue JSON primary** | `effortMinutes` survives; external issues count toward the quality gate **[L1 §4]** | Imported rules invisible on the Rules page and in every profile — we own profile management anyway **[L1 §4]** |
| b′ | SARIF only | GitHub Code Scanning free | **Disqualifying:** Sonar's SARIF importer forces `SECURITY`/`CONVENTIONAL` and has **no effort field** **[L1 §4]** |
| c | Harness plugin only | Zero infra, lives where the author works | No CI gate for non-Claude users, no interop, no trend; today's implementation is structurally single-skill **[L1 §2d]** |

### Recommended: **b″ — native `report/v1` JSON is the contract; Sonar generic-issue and SARIF are thin emitters behind flags; the harness plugin (c) is the front end.**

A deliberate amendment to L1, which named generic-issue JSON *primary* **[L1 §4]**. L1's own first open decision asks whether anyone here runs SonarQube and found no evidence either way **[L1 decisions §1]**; L1 also flags as thin evidence the exact behaviour the economics would rest on — that imported `effortMinutes` rolls into `sqale_index` **[L1 handoff]**. Making an unverified integration the primary contract inverts the dependency. Our schema carries `effortMinutes` + `{softwareQuality, severity}` natively; both emitters are ~150 lines of projection. L1's evidence survives intact: **SARIF never the sole output**, **no Java plugin**.

**The named tradeoff:** we give up Sonar's native quality-profile management and hotspot review lifecycle permanently, so we must own (i) a checked-in profile file — which rules, what severity, what effort minutes — and (ii) a hotspot-review baseline file, the pattern SkillSpector already uses **[L1 §4]**. Real work; it belongs in the spec, not in a later surprise.

---

## 6. The first three PRs

| # | PR | Proves | Independent because | Test |
|---|---|---|---|---|
| **1** | **`skillaudit scan`** — rules over one skill *and* a catalog: `SK-R001` (dead ref, resolving `$var`), `SK-R002` (cycle, Tarjan), `SK-R005` (trigger collision ≥60%), `SK-M001–M005`, each with `effortMinutes` + `{quality, severity}`. Emits `report/v1.json` + exit code and **appends `calibration.jsonl`** | Debt **minutes** for a real skill on day one — what no existing tool does **[L1 §3]** — while the ledger accrues the data that later earns the letter (**F4**) | Pure filesystem. No telemetry, no model calls, no profiler. The ledger is a file append, not a dependency | Golden fixture: a skill dir with each defect → expected report JSON; a fixture git history → expected ledger rows |
| **2** | **Profiler honesty repairs + the two OTel renames** — D-1 refuse cross-`MetricSource` comparison, carry both sides; D-3 nil-guard every `.Value`; D-2 compare `EstimatedContextTokens` tagged *relative estimate, not billed*; **D-6** guard or delete `stopping_rule: "threshold"`; delete the now-false claim at `claude_code.go:83`; **R1** and **R2** **[L2 §5.2; OT R1, R2]** | The comparator stops laundering estimates and stops manufacturing significance; the schema stops disagreeing with semconv while renames are still free | Touches only `skill-architect/profiler`. Needs nothing from PR 1. Bundling R1/R2 avoids a second breaking touch of `types.go` | Re-run L2's three executed repros: the `session_data`-vs-`otel` delta must refuse; `{"state":"present"}` must not panic |
| **3** | **`skillaudit rank`** — DuckDB over the spool for `A`, `U`, `C_a`, `F`; join PR 1's minutes; emit both lanes with per-cell source labels, the sensitivity pair and the absolute floor | The full ranking, including "next best skill", **without running a single experiment** — OTel gives two of three SDY inputs free **[L2 handoff]** | **Degrades cleanly:** with no `report/v1.json` it ranks on Waste$/Risk$ and prints `E: unknown` | Fixture spool → expected ranked table; assert every dollar cell carries a tier label and no letter appears |

> **Scope flag, found at merge time** (uncommitted working tree of `skill-architect`, branch `main`, 2026-09-11): most of PR2 appears **already done** by a parallel worker. `compare.go` now has a `guardPair` helper that refuses on source mismatch and records both sources; `validateProfile` rejects present-without-value; `estimated_context_tokens` is compared at `compare.go:148`; `types.go` already carries `CacheWrite` (R1, normalizing `cache_creation` on input) and `ToolCallEntry.ErrorType` (R2). **Still open:** D-6 (`experiment.go:156` accepts `stopping_rule: "threshold"` with no alpha-spending guard) and the stale comment at `claude_code.go:83`. **Re-scope PR2 to land, test and commit what exists plus D-6 — not to rebuild it.** A working-tree observation, not a lens finding; confirm ownership first.

Deferred to PR 4+: N-arm experiment plan, pooled bootstrap CI, achieved-MDE line, verifier hook, pinned price table **[L2 §5.3]**. Those turn `r̂` into `r`; nothing above depends on them.

---

## 7. Where the lenses disagree, the risks, and what needs the user

### 7a. Disagreements resolved

| Conflict | Pick |
|---|---|
| Is static token weight the headline? L1 makes it High-severity `SK-M001` **[L1 §4]**; L2 measures the listing saving at 1.5% of the activation-path saving **[L2 §4.3]** | **L2.** Demote `SK-M001` to Medium, a *context-pressure* rule, not a cost rule |
| **`SK-M001` compares tokens to characters** — L1 states it as "`always_loaded_tokens` over the **1,536-char** listing cap" **[L1 §4]** | **Characters.** The primary source is a *truncation* cap: Claude Code truncates combined `description`+`when_to_use` at 1,536 chars **[L1 §2b]**. Rule, threshold and message all in characters; text past the cap is silently dropped. `always_loaded_tokens` ships as a **measure**, not an issue — Sonar's own measure-vs-issue distinction **[L1 §1]** |
| Primary output format — L1: Sonar generic-issue JSON **[L1 §4]** | **Amended** — native `report/v1`, both emitters behind flags (§5) |
| Are dollars ever better than estimates? L3: every dollar is an estimate, `c` caps at 0.5; L2: Claude Code OTel gives measured tokens **[L2 §2]** | **Both, scoped.** L3 is right for **Cursor** (no admin key, no Enterprise OTel **[Q2]**); Claude Code reaches tier 2. `c` keys off the **dollar** tier and a closed experiment (§2b), capped at 0.8 |
| Single-skill vs catalog — `skill-audit`'s contract is one skill (`SKILL.md:33`) **[L1 §2d]**; L3 needs a catalog pass **[L3 §4]** | **Catalog.** PR 1 takes a directory *or* a root — a user-visible interface change to a shipped plugin |
| L3's worked example does not reproduce from its own formula | Disowned, with corrected figures, in §2b |
| Shelling out to SkillSpector/jscpd — L1 recommends it **[L1 §4]** | **Conflicts with [Q8]** ("no Semgrep, no Python") and `PRINCIPLES.md:24` self-containment. Opt-in, off by default, degrade to `unknown`; see 7c #2 |

### 7b. Risks

| Risk | Evidence | Mitigation |
|---|---|---|
| **Uncalibrated effort minutes produce confident, wrong dollars** — every figure in §2a is illustrative | **[L1 §4, explicit]** | The ledger (§3) + **F4**. Ship minutes; the letter is earned at n ≥ 20, not asserted |
| **`A` for Cursor rests on an untested hook heuristic (U-02)** and is this team's sole source of `A` | **[L3 §1.3, handoff]** | Validate against the first real capture; until then label `A` as `inferred` |
| **`gen_ai.skill.*` may never land** — PR #498 open, #500 closed, seven breaking changes queued for the first release | **[OT S-2, S-5, G4]** | Keep all `gen_ai.*` in a gated export layer, off by default. **R1/R2 are internal and do not wait on #498** |
| Optional stopping inflates false positives — `experiment.go:156` takes `stopping_rule: "threshold"` with no alpha spending | **[L2 §5.2 D-6]** | Guard or delete before any experiment ships. Still open in the working tree (§6) |
| R12/R7 (negative lift, sticky context cost) are **invisible to any static audit** and carry the largest dollar consequence | **[L3 §2]** | The argument for funding PR 4+; the static engine cannot answer "did it help" |
| Cursor isn't on the dev machine; all development is against fixtures | **[Q environment facts]** | PR 1 and PR 2 are fixture-only by construction |

### 7c. Decisions only the user can make

1. **Is SonarQube a target or a metaphor?** If nobody runs a server, the generic-issue emitter is speculative surface **[L1 decisions §1]**. Either way, verify `effortMinutes` → `sqale_index` on a throwaway instance before the economics depend on it **[L1 handoff]**.
2. **Vendor, shell out to, or reimplement SkillSpector (Apache-2.0, Python 3.12) and jscpd (Node)?** Shelling out re-imports the runtimes **[Q8]** rejects; vendoring means owning 71 security patterns **[L1 decisions §3]**.
3. **Own `C_f` and the engineer-hour rate.** §2e shows `Risk$` dominating every row and the EV gap collapsing to $6 at ½`C_f`; these two numbers set the ranking **[L3 decisions §2]**.
4. **Who owns the calibration ledger, and will the team time 20 fixes per rule?** Without it, **F4** means minutes ship and a letter never does. Honest, but a product decision **[L1 decisions §2]**.
5. **What N will we fund?** ~12 paired tasks for a large effect, ~216 for a small one **[L2 §3.2]**. If the answer is 5, the tool refuses to print significance **[L2 decisions §4]**.
6. **Does the hotspot lane block merges or only warn?** Datadog PR Gates and Sonar both block **[L3 §3.4]**; a heuristic script scan will false-positive on day one **[L3 decisions §3]**.
7. **Rule-ID rename `PL*`/`PT*` → `SK-*` plus catalog-first scope** are two user-visible breaks to a shipped plugin's exit-code contract **[L1 decisions §4, §5]**. Take both in one release or neither.
8. **Cross-user portfolio scope** needs per-user activation data, and **[Q10]** keeps prompt text and file paths on disk by default. Aggregating that across people is a consent decision **[L3 decisions §4]**.

---

## Provenance

Three independent candidates were written to the same mandate with no visibility into each other, then judged by a fresh hunter against the stated bar in `.scuba/teams/skill-audit-tool/arena.md` (B1–B7; B1/B2/B3 must-pass). Scorecard and defect list: `.scuba/teams/skill-audit-tool/arena-verdict.md`. **Base: candidate B** — soundest boundaries, soundest build recommendation, best slice independence, shortest. **Fold-in from C:** the calibration ledger and refusal **F4**. **Fold-in from A:** the OTel changes **R1** (`cache_write` rename) and **R2** (`error_type`). Every defect in the verdict's list was applied, and every number in §2 recomputed from the stated formula and inputs at merge time. **2026-09-11.**
