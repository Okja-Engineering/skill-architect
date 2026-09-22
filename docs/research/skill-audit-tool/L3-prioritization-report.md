# L3 — Risk, impact, value, and "next best skill to improve"

**Researcher lens L3 · 2026-09-11 · research only.** Evidence labels: **[O]** = observed (I read the
source), **[I]** = inferred. Sibling lenses L1 (static) and L2 (dynamic) own their ground; §4 is the
handoff.

## The answer in five lines

1. Rank skills by **Skill Debt Yield (SDY) = (Waste$ + Risk$) × Confidence ÷ Remediation-hours** —
   a dollarized RICE numerator, a WSJF ratio shape, and a SonarQube remediation-cost denominator.
2. Run **two lanes, not one**: an EV lane ordered by SDY, and a **hotspot lane** (blast radius ×
   permission scope) that jumps the queue regardless of EV — exactly SonarQube's split between
   technical debt and security hotspots.
3. SonarQube's debt ratio alone fails here: its denominator (lines of code) has no skill analogue and
   it is blind to usage frequency, so a never-triggered skill ranks equal to a hot one.
4. On this team's current environment every dollar is an **estimate** (no Admin API key, no
   Enterprise OTel) [O], so `Confidence` is not decorative — it is the term that keeps the ranking honest.
5. Best-in-class end to end = SonarQube's new-code gate + Braintrust/LangSmith's pinned-baseline
   regression view + Datadog's PR Gates and trend dashboards + `/skill-doctor`'s per-skill context cost.

---

## 1. Scoring model

### 1.1 Candidate models compared

| Model | Formula (verbatim) | What it gets right for skills | Why it fails alone |
|---|---|---|---|
| SonarQube technical-debt ratio | `sqale_debt_ratio = technical debt / (cost to develop one line of code * number of lines of code)`; debt = "sum of the maintainability issue remediation costs… effort (in minutes)"; default cost/line 30 min; rating A "≤ 5% to 0%" … E "≥ 50%" **[O]** ([metrics-definition](https://docs.sonarsource.com/sonarqube-server/user-guide/code-metrics/metrics-definition.md)) | The remediation-minutes → debt → ratio chain is the cleanest "issues become money" model in shipped software. Letter ratings give a portfolio view non-engineers read. | Denominator is LOC; a SKILL.md's line count is not its cost. **Blind to activation frequency** — the metric that dominates skill economics **[I]**. |
| SonarQube security-hotspot ranking | "Review priority is determined by the security category"; high = "categories ranked high on the OWASP Top 10 and CWE Top 25" **[O]** ([security-hotspots](https://docs.sonarsource.com/sonarqube-server/user-guide/security-hotspots)) | Correct shape for the risk lane: an ordered queue by category severity, separate from debt. | Not value-ranked; cannot answer "which skill is worth my Tuesday". |
| WSJF | "WSJF is estimated as the relative cost of delay divided by the relative job duration"; CoD = user/business value + time criticality + risk reduction/opportunity enablement **[O]** ([SAFe](https://framework.scaledagile.com/wsjf)) | Ratio-of-value-to-size sequencing is the right economic shape, and CoD explicitly carries a risk term. | Fibonacci relative scores, no dollars. The mandate requires a dollar amount **[O]** (mandate.md:11-12). |
| RICE | `RICE = (Reach × Impact × Confidence) / Effort`; Impact 3x/2x/1x/0.5x/0.25x; Confidence 100%/80%/50% **[O]** ([Intercom](https://www.intercom.com/blog/rice-simple-prioritization-for-product-managers/)) | **Confidence as an explicit multiplier** is the single most important borrow, given estimated-token inputs. Reach maps directly onto activations × users. | Impact is a judgment scale, not measured; no risk term; Effort in person-months is too coarse for a skill edit. |

### 1.2 Recommended: Skill Debt Yield (SDY)

```
SDY  =  ( Waste$  +  Risk$ ) × c  ÷  E          [dollars recovered per engineer-hour, per period]

Waste$ =  A × U × C_a × r                       recoverable spend per period
Risk$  =  A × U × F × B × C_f                   expected cost of failures avoided
c      =  confidence multiplier (1.0 / 0.8 / 0.5), derived mechanically from MetricSource
E      =  remediation effort in hours = Σ (per-rule remediation minutes from the static audit) / 60
```

Three design choices, stated so they can be argued with:

- **Impact terms are inside `A` and `U`, not added.** "Tasks depending on it" is activation frequency;
  "users depending on it" is `U`. Adding them separately double-counts **[I]**.
- **Risk is dollarized before it is added**, so the `+` is legal. `B` (blast radius) and permission
  scope are multipliers on the cost of a failure, not points on an unrelated scale **[I]**.
- **SDY never governs the safety queue.** A skill bundling `curl … | sh` with broad `allowed-tools` but
  two activations a month scores near zero on SDY and must still be fixed first. That is the
  hotspot lane (§2), ordered by `B` × permission scope, not by SDY **[I]**.

### 1.3 Every input term and where it comes from

| Term | Meaning | Preferred source | Fallback (this team, today) | Honesty label |
|---|---|---|---|---|
| `A` | activations per user per period | Enterprise OTel log `cursor.skill.activated` with `cursor.skill.name`/`.trigger` — "the *only* first-party skill-activation event in any harness surveyed" **[O]** (`docs/research/cursor-telemetry-surfaces.md` §2) | Hook spool: `beforeReadFile.file_path` matching `SKILL.md` + `subagent_type` — cursorscope's heuristic, **load-bearing and untested** **[O]** (ibid. §1, U-02). `/skill-doctor` gives invocation frequency per skill in Claude Code ≥ v2.1.252 **[O]** ([docs](https://code.claude.com/docs/en/skills)) | `inferred` |
| `U` | distinct users with ≥1 activation | Admin API `filtered-usage-events.userEmail`; OTel `cursor.user.id` **[O]** (ibid. §2–3) | hook base field `user_email` **[O]** (ibid. §1) | observed |
| `C_a` | mean $ attributable to one activation | Admin API `filtered-usage-events.tokenUsage{…}` + `totalCents`/`chargedCents`, joined on `conversationId` ↔ hook `conversation_id` — "**billed**, per request, four-way split" **[O]** (ibid. §3, §5 rank 1) | `chars/4` over hook payloads, "±−7%…+15%" on markdown but structurally undercounting because "billed input tokens per turn are approximately the *cumulative* conversation re-sent each turn" **[O]** (ibid. §1, §5) | **`estimated` only today** — Q2 resolves Admin API and OTel as unavailable **[O]** (`questions.md` Q2) |
| `r` | achievable fractional reduction | L2's paired experiment posterior with a CI **[I]** | Prior from SkillReducer: 48% description / 39% body compression, quality +2.8% **[O]** (`docs/research/skill-evaluation-research.md`) — **static compression, not a runtime saving; must not be used as `r` unmodified** **[O]** (ibid., explicit warning) | modeled |
| `F` | fraction of activations ending in failure | OTel `cursor.tool.calls` `status=failure`; `cursor.api.error` **[O]** (telemetry §2) | spool `postToolUseFailure`, `stop.status`, `subagentStop.status`/`loop_count` **[O]** (ibid. §1) | `inferred` |
| `B` | blast-radius multiplier | Static audit of bundled scripts (shell, network, credential reads) + `allowed-tools` breadth. `allowed-tools` "grants permission… does **not** restrict which tools are available" and "can grant broad access" **[O]** ([Claude Code skills](https://code.claude.com/docs/en/skills)) | same (static is the primary source here) | observed (static) |
| `C_f` | cost of one failed activation | Org constant: engineer-hour rate × mean rework minutes | — | **declared assumption** |
| `c` | confidence | Mechanical from `MetricSource`: `server_api`/`otel` = 1.0; `hooks` + N≥ paired reps = 0.8; `hooks_estimated` single run = 0.5 **[I]**, modelled on RICE's 100/80/50 **[O]** | — | derived |
| `E` | remediation hours | L1 rule table: per-rule remediation minutes, SQALE-style **[I]** — L1 must emit these (§4) | expert estimate per finding | modeled |

**Worked shape (placeholders, not claims):** a skill activating 12×/user/week across 8 users at an
estimated $0.34/activation with a measured `r` = 0.22 yields `Waste$` ≈ $9.0/week ≈ $470/yr; with
`F`=0.06, `B`=1 (no scripts), `C_f`=$40 → `Risk$` ≈ $920/yr; `c`=0.5 (estimated tokens, single run);
`E`=3 h → **SDY ≈ $232 per engineer-hour per year**. Ranking is meaningful; the absolute figure is not,
until `C_a` is billed **[I]**.

---

## 2. Risk taxonomy for skills, with detectability

`Static` = a file scan can decide it. `Dynamic` = only runtime can. Feeds = which SDY term it moves.

| # | Risk | Concrete failure | Static detectability | Dynamic detectability | Feeds |
|---|---|---|---|---|---|
| R1 | **Never triggers** | "A weak `description` means the skill never loads, even if the body is perfect" **[O]** (`skill-architect/PRINCIPLES.md`) | Partial — description heuristics only | **Decisive** — activation count 0 against eligible tasks; `/skill-doctor` flags "skills that have never been used" **[O]** | `A` |
| R2 | **Over-triggers** | fires on unrelated work, evicting better context | Partial — cross-catalog embedding overlap (SkillEvaluator Tier 2) **[O]** (`skill-profiling-state-of-the-art.md` §1) | **Decisive** — activation precision needs negative tasks | `A`, `Waste$` |
| R3 | **Silent step-skip** | "A missing heading means the agent skips a step… These failures are usually silent" **[O]** (PRINCIPLES.md) | Partial | **Decisive** — trajectory/behaviour check | `F` |
| R4 | **Dead reference** | "A script name mismatch means the agent runs a command that does not exist" **[O]** (ibid.) | **Decisive** — `check-paths.sh` **[O]** | `postToolUseFailure` | `F` |
| R5 | **Circular / runaway disclosure** | skill→script→skill loop | **Decisive** — reference graph | `loop_count`, `preCompact` | `C_a` |
| R6 | **Context bloat** | Claude Code re-attaches skills after auto-compaction at "first 5,000 tokens per skill, 25,000 token combined budget" **[O]** ([docs](https://code.claude.com/docs/en/skills)); Anthropic budgets ~100 tok metadata / <5k body **[O]** ([platform docs](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview)) | **Decisive** — token weight vs those constants | `preCompact.context_tokens` / `context_usage_percent` — "the only real, Cursor-reported token number" **[O]** | `C_a` |
| R7 | **Sticky context cost** | skill content "stays in context across later turns" and is re-read only on re-invocation **[O]** (ibid.) — so `C_a` is a carried, not one-shot, cost | No | **Decisive** | `C_a` |
| R8 | **Excessive agency** | `allowed-tools` grants and does not restrict **[O]** | **Decisive** — frontmatter parse | observed vs declared tool set | `B` |
| R9 | **Destructive bundled script** | `rm -rf`, `curl \| sh` | **Decisive** — pattern scan | `beforeShellExecution` | `B` (hotspot lane) |
| R10 | **Prompt injection in payload / fetched content** | "Skills that fetch data from external URLs pose particular risk, as fetched content may contain malicious instructions" **[O]** (platform docs) | Partial — static URL fetches yes, dynamic content no | Partial — behavioural diff | `B` (hotspot lane) |
| R11 | **Supply chain / trojanized skill** | ClawHavoc: "nearly 1,200 malicious skills infiltrated a major agent marketplace, exfiltrating API keys, cryptocurrency wallets, and browser credentials" **[O]** (arXiv:2602.20867 abstract). *Counts conflict: 1,184 vs 341 across reports — flagged, unresolved.* | **Decisive for provenance** — hash pinning, egress patterns | egress observation | `B` (hotspot lane) |
| R12 | **Negative lift — the skill actively hurts** | ACES: mean composite Skill Lift 0.2134, 95% CI [0.1967, 0.2301], **positive in only 72.8% of cases → 27.2% of skills hurt** **[O]** (`skill-profiling-state-of-the-art.md` §1) | **Undetectable statically** | **Decisive** — paired eval | `r` (can be negative) |
| R13 | **Model/harness drift** | tuned to one model | No | **Decisive** — cross-model repeat; SkillReducer retention 0.965 across 5 models **[O]** | `r` |
| R14 | **Catalog duplication** | two skills claim the same job | **Decisive** — needs the whole catalog, not one skill | — | `A` |

**The load-bearing asymmetry:** R12 and R7 — the two risks with the largest dollar consequence — are
invisible to any static audit. R8–R11 — the two with the largest *tail* consequence — are largely
decidable statically and cheaply. That asymmetry is the argument for the two-lane queue **[I]**.

---

## 3. What best in class looks like end to end

### 3.1 Personas and what each one opens

| Persona | Question | Artifact |
|---|---|---|
| Skill owner (author) | "Did my refactor help?" | Before/after comparison with pinned baseline, red/green per task |
| Platform / DevEx lead | "What do I fix next?" | Portfolio table ranked by SDY, plus the hotspot lane |
| Security reviewer | "What can this skill reach?" | Hotspot queue: bundled-script capabilities, `allowed-tools` breadth, provenance |
| Eng manager / finance | "What did this cost and what did we recover?" | $/skill/week trend, with every figure labelled billed vs estimated |

### 3.2 The loop

`Inventory → static audit (cheap) → SDY rank → pick top skill → baseline paired run → refactor →
re-run same tasks → Skill Lift with CI → re-score → gate in CI → trend`.
Ordering cheap-before-expensive is NVIDIA SkillEvaluator's tier model and skill-architect's own
Tier 1/2/3 **[O]** (`skill-audit/references/evaluation-matrix.md`; `skill-profiling-state-of-the-art.md` §1).

### 3.3 The CI quality gate ("clean as you edit")

Borrow Sonar's **new-code** focus verbatim: gate the *changed* skill, not the catalog. Sonar way's four
conditions are "No new issues are introduced / All new Security Hotspots are reviewed / New code test
coverage ≥ 80.0% / Duplication in the new code ≤ 3.0%", and it "focuses on keeping high quality
standards for new code, rather than spending a lot of effort remediating old code" **[O]**
([quality gates](https://docs.sonarsource.com/sonarqube-server/2026.1/quality-standards-administration/managing-quality-gates/introduction-to-quality-gates)).

| Proposed condition on the changed skill | Rationale |
|---|---|
| No new Blocker/High static issues | Sonar way condition 1 **[O]** |
| 100% of new bundled-script hotspots reviewed | Sonar way condition 2; Sonar's Security Review rating is A at "≥ 80%" reviewed **[O]** |
| Dead references = 0, reference cycles = 0 | R4/R5 are decisive statically and silent at runtime **[O]** |
| SKILL.md body ≤ 5,000 tokens; catalog metadata within the 25,000-token combined budget | Claude Code's own re-attachment constants **[O]** |
| For skills above an SDY threshold: paired-eval Skill Lift **CI lower bound ≥ 0** | ACES's 27.2% negative-lift rate makes "no regression" the right assertion, and a point estimate is not evidence **[O]** |

### 3.4 What to borrow, from whom

| Product | Borrow | Evidence |
|---|---|---|
| **SonarQube** | remediation-minutes → debt → ratio → letter rating; new-code gate; **hotspot lane kept separate from debt**, ranked by category priority | **[O]** metrics-definition, quality-gates, security-hotspots docs |
| **Braintrust** | persistent/project-default **baseline experiment**; "Each row is color-coded: green for improvements, red for regressions"; click a score header → "X regressions"; summary grades **Improvement / Regression / Tradeoff / Tie** | **[O]** ([compare-experiments](https://www.braintrust.dev/docs/evaluate/compare-experiments)) |
| **LangSmith** | "**Set as source experiment**"; red/green vs source with per-column better/worse counts; **pairwise evaluators**; **online evaluations** on production traffic "detecting issues, anomalies, and quality degradation in real-time"; **annotation queues** with prescribed rubrics | **[O]** ([compare](https://docs.langchain.com/langsmith/compare-experiment-results), [concepts](https://docs.langchain.com/langsmith/evaluation-concepts)) |
| **Datadog LLM Observability** | OOTB cost/latency/usage **trend** dashboards; **Insights** anomaly detection; **PR Gates** — "configuring rules to block pull requests with substandard code from being merged", enforced as required GitHub checks | **[O]** ([LLM Obs](https://docs.datadoghq.com/llm_observability/), [PR Gates](https://docs.datadoghq.com/pr_gates/)) |
| **Claude Code `/skill-doctor`** | per-skill **context cost + invocation frequency + never-used flag**, and "where to turn off unused skills" — free `A` and part of `C_a` in that harness | **[O]** ([docs](https://code.claude.com/docs/en/skills)) |
| **NVIDIA SkillEvaluator / ACES** | tiered gating; **Skill Lift** as the headline statistic, reported with a CI | **[O]** (research docs) |

**Portfolio view** = one row per skill: `SDY | A | C_a | est. $/week | debt (hours) | risk letter | last measured lift ± CI | source label`.
Sort by SDY; filter to the hotspot lane. **Trend** = per-skill $/week and debt-hours sparklines plus a
catalog-level "% of skills with a measured lift in the last 90 days" — the skill analogue of coverage.

### 3.5 The non-negotiable presentation rule

Every dollar carries its source label. cursorscope's discipline — "it never launders an estimate as a
measurement" **[O]** (`cursor-telemetry-surfaces.md` §5) — must survive into the portfolio UI, or the
ranking becomes a confident lie. Today, that means most cells read `estimated`.

---

## 4. Handoff note — what L1 and L2 might miss

| To | Gap | Why it matters to L3 |
|---|---|---|
| L1 | Emit **per-rule remediation minutes**, not just pass/fail. | `E` is the SDY denominator; without it there is no ranking, only a list. Sonar's whole money model rests on this constant **[O]**. |
| L1 | Emit a **machine-readable severity × software-quality** pair per issue (Sonar MQR: security/reliability/maintainability, Blocker/High/Medium/Low/Info) **[O]** ([MQR](https://docs.sonarsource.com/sonarqube-server/instance-administration/analysis-functions/instance-mode/mqr-mode)). | The hotspot lane needs an ordering key that is not the EV score. |
| L1 | Scope is **catalog-level, not skill-level**, for R14 duplication and R2 overlap. The mandate's workflow is "run it against one specific skill" **[O]** (mandate.md:10) — that shape cannot see overlap. | Portfolio ranking needs an inventory pass. |
| L1 | Treat `allowed-tools`, `disable-model-invocation`, and `context: fork` as first-class measured fields. | `allowed-tools` is the permission-scope input to `B`; `context: fork` materially changes `C_a` **[O]**. |
| L2 | Emit **cost per activation**, not only per-session delta. Skill content "stays in context across later turns" **[O]**, so per-session deltas under-attribute long sessions. | `C_a` is the largest term in `Waste$`. |
| L2 | Activation **precision** requires negative tasks (where the skill should *not* fire). An all-positive task set cannot measure R2. | Over-triggering is pure waste and is otherwise invisible. |
| L2 | The comparison report must carry **`MetricSource` per side**, not just per report — the known `compareTokens` laundering defect **[O]** (mandate.md:64; `profiler/compare.go:106-120` returns a single `Source: candidate.Source`). | `c` is computed mechanically from that field. If sources differ across sides, the delta is not comparable at all. |
| L2 | Do not seed `r` from SkillReducer's 39%/48% compression figures — those are static compression ratios, explicitly "not a runtime token-savings claim" **[O]** (`skill-evaluation-research.md`). | Would inflate every SDY in the portfolio simultaneously. |
| Both | `.scuba/teams/otel-genai-alignment/report.md` named in the mandate **does not exist**; only `mandate.md` is present **[O]**. The OTel content I used came from `docs/research/otel-genai-alignment.md` and `cursor-telemetry-surfaces.md`. | Anyone citing that path will cite a missing file. |

---

## Decisions needed above my level

1. **Buy billed tokens or stay on estimates?** `questions.md` Q2 resolves Admin API and Enterprise OTel
   as unavailable **[O]**. Without a Cursor Team-plan admin key, every `C_a` and therefore every dollar
   in the portfolio is `estimated`, and `c` caps at 0.5. A single admin key moves the whole product
   from "ranking tool" to "cost tool". Cost/policy call, not mine.
2. **The org constants.** `C_f` (cost of one failed activation) and the engineer-hour rate are declared
   assumptions that set the relative weight of `Risk$` vs `Waste$`. Someone must own those numbers.
3. **Does the hotspot lane block merges or only warn?** Datadog's PR Gates and Sonar's gate both block
   **[O]**. Blocking on a heuristic script scan will produce false positives on day one.
4. **Cross-user portfolio scope.** Ranking across the team needs per-user activation data; Q10 keeps
   prompt text and file paths on disk by default **[O]**. Aggregating that across people is a privacy
   and consent decision.
5. **Is `r` allowed to be negative in the portfolio?** ACES says 27.2% of skills hurt **[O]**. A UI that
   can only show improvement will hide the most valuable finding — deletion candidates.

## Handoff to synthesis

- **Take:** SDY formula (§1.2), the input/source table (§1.3), the two-lane queue, the 14-row risk
  taxonomy (§2), the CI gate conditions (§3.3), the borrow table (§3.4).
- **Do not take:** the worked example's numbers — they are placeholders.
- **Thin evidence, flagged:** the hook-based activation heuristic is untested (U-02) and is the sole
  source of `A` for this team **[O]**; ClawHavoc skill counts conflict across reports (1,184 vs 341)
  **[O]**; SAFe's own WSJF page is partly behind login, so the Fibonacci scale detail is secondary-source
  only **[O]**.
- **Cross-lens dependency:** SDY is inert until L1 emits remediation minutes and L2 emits per-activation
  cost with per-side sources. Sequence those two before any ranking UI.
