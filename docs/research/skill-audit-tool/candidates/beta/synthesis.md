# Skill Audit Tool — synthesis (candidate BETA)

**What to build, why, and how to prove it.** Sources: `mandate.md`, `L1-static-report.md` (L1),
`L2-dynamic-report.md` (L2), `L3-prioritization-report.md` (L3), `docs/research/otel-genai-alignment.md`
(OTel), `spec.md`, `questions.md`. All dollar figures below are placeholders and labeled by tier.

---

## 1. The pitch in five lines

1. **What:** "SonarQube for Agent Skills" — one Go binary that statically audits `SKILL.md` packages
   (rules → findings → remediation minutes → debt) and dynamically proves whether a refactor helped
   (paired three-arm runs → token deltas → labeled dollars).
2. **For whom:** AI-native dev teams who author and ship skills; the skill owner, the DevEx lead,
   the security reviewer, and the manager who asks what it cost (L3 §3.1).
3. **Why now:** the static white space is verified empty — spec-conformance (agnix), security
   (SkillSpector), and duplication (jscpd) are solved, but **remediation cost, cross-skill reference
   graphs/cycles, new-code scoping, and portfolio ranking have no tool** (L1 §3 verdict).
4. **Why us:** we already own both halves — skill-architect's audit/profiler and cursor-profiler's
   hook spool + DuckDB store — and the research found the dynamic side's defects before we built on
   them (L2 §5.2, D-1–D-9).
5. **The catch, stated up front:** this team's harness is Cursor with hook telemetry only, so every
   dollar today is **estimated** and per-run token deltas are **ordinal**. The tool's ranking still
   works; its honesty contract is what makes it trustworthy (§4).

---

## 2. The workflow — one skill, end to end

Skill under audit: `release-checklist` (fictional; every number is a placeholder).

### Step 0 — Static baseline (seconds, zero model calls)

The rule engine scans the skill + catalog. Findings carry remediation minutes (L1 §4 rule set):

| Finding | Rule | Quality / Severity | Effort |
|---|---|---|---|
| `scripts/verify.sh` named in a code fence, absent on disk | SK-R001 | Reliability / High | 10 min |
| `description` 12 chars — skill may never trigger (L3 R1) | SK-R004 | Reliability / Blocker | 15 min |
| Body = 6,200 tokens > 5,000 spec recommendation | SK-M002 | Maintainability / Medium | 90 min |
| Reference chain depth 2 > 1 | SK-M003 | Maintainability / Medium | 45 min |
| **Debt Σ** | | | **160 min → E = 2.7 h** |

`skill-validator check -o json` supplies the real `o200k_base` counts: listing 180 tok, body 6,200 tok
(L1 §2b, §2d). Debt ratio and A–E rating require an author-cost-per-line constant — uncalibrated,
flagged as an open decision (§7).

### Step 1 — Dynamic baseline (cursor-profiler spool → DuckDB)

| Input | Value | Source / label |
|---|---|---|
| `A` activations/user/wk | 12 | hook-spool heuristic — **inferred, untested (U-02)** |
| `U` users | 8 | hook `user_email` — observed |
| `C_a` cost/activation | $0.34 | chars/4 × price table — **estimated (tier 4→3 label)** |
| `F` failure fraction | 0.06 | `postToolUseFailure`, `stop.status` — inferred |

Gross attributable spend ≈ 4,992 act/yr × $0.34 ≈ **$1,697/yr estimated** — attribution, *not*
causation (L2 §2.1: `sum by skill` = cost-when-active, never cost-caused-by).

### Step 2 — Refactor

Fix dead path; rewrite `description`; move 3,800 tokens into `references/` (progressive disclosure,
PRINCIPLES L3). New static scan: debt 160 → 25 min.

### Step 3 — Prove (three-arm paired run)

Stratified task set (L2 §1.3): 15 should-fire, 10 should-not-fire, 5 neutral; arms **A** no-skill /
**B** v1 / **C** v2; model + harness + repo SHA pinned; arm order randomized (L2 §1.2).

| Result | Value | Statistical status |
|---|---|---|
| Context tokens C−B per activation | **−22% median**, BCa 95% CI [−34%, −8%] | significant — CI excludes 0; **relative estimate, not billed** (tier 4 on Cursor) |
| Activation 2×2 (should-fire × fired) | TP 13, FN 2, FP 1, TN 9 → precision 0.93, recall 0.87 | reported as raw 2×2 + both rates, never F1 (L2 §2.1) |
| Verifier pass rate B→C | 11/15 → 12/15 | McNemar n.s.; **achieved MDE printed**: powered only for ≥30-pt swings |
| Sanity vs arm A | lift(B)=+0.18, lift(C)=+0.21 | v2 beats nothing — rules out R12 negative lift (ACES: 27.2% of skills hurt) |

**Refactor verdict: improved (relative estimate).** No dollar figure appears on this line — tier-4
inputs (§4). On Claude Code, the same run would carry measured tokens (tier 2) and an *estimated*
dollar delta.

### Step 4 — Dollars and ranking (portfolio layer, all cells labeled)

`Waste$ = A·U·C_a·r` and `Risk$ = A·U·F·B·C_f`, `SDY = (Waste$+Risk$)·c ÷ E` (L3 §1.2):

| Term | Value | Note |
|---|---|---|
| `r` achievable reduction | 0.22, CI [0.08, 0.34] | from step 3 — labeled estimated; **never** seeded from SkillReducer's static compression figures (L3 §4) |
| `Waste$` | 4,992 × $0.34 × 0.22 = **$373/yr** est. | CI-propagated: **[$136, $577]/yr** |
| `Risk$` | 4,992 × 0.06 × B=1 × $40 = **$11,980/yr** est. | Risk dominates — a flaky high-frequency skill; that is the model working |
| `c` confidence | 0.5 | mechanical: hooks-estimated source, single run (L3 §1.3) |
| `E` | 2.7 h | from step 0 debt minutes |
| **SDY** | (373 + 11,980) × 0.5 ÷ 2.7 ≈ **$2,290 / engineer-hour / yr** | **ranking is meaningful; the absolute is not, until `C_a` is billed** (L3 §1.3) |

*Reconciliation note:* L3 §1.3's printed example ($9.0/wk Waste$, $920/yr Risk$) does not reconcile
with its own formula applied to its stated inputs; all figures above are recomputed from the formula.

### Step 5 — Next best skill

Portfolio table, two lanes (L3 §1.2, §3.4): an **EV lane** sorted by SDY, and a **hotspot lane**
(`B` × permission scope) that jumps the queue regardless of EV — a skill bundling `curl | sh` with
broad `allowed-tools` but rare use scores ~0 SDY and is still fixed first.

| Skill | SDY | est. $/wk | debt | lane |
|---|---:|---:|---:|---|
| `deploy-helper` | $3,100/h | $240 | 4 h | hotspot ↑ (R9 script scan) |
| `release-checklist` | $2,290/h | $237 | 2.7 h | EV |
| `sql-migrate` | $410/h | $41 | 6 h | EV |

**Output: fix `deploy-helper` next** — the hotspot lane ordered it ahead of the higher-SDY row.

---

## 3. Architecture

```
SKILL.md packages                Cursor hooks ──stdin──► cursor-profiler ──► JSONL spool (truth)
      │                                                                    │  (questions.md Q9:
      ▼                                                                    ▼   DuckDB = derived cache)
┌─ STATIC ENGINE (new Go binary) ──────────┐   ┌─ DYNAMIC ENGINE ───────────────────────────┐
│ rule registry (SK-*, effort minutes)     │   │ cursor-profiler capture → profile/v1        │
│ reference graph + Tarjan SCC (NEW)       │   │ skill-architect/profiler:                   │
│ cross-skill dup + trigger-collision(NEW) │   │   compare.go + experiment.go (FIX D-1..D-9) │
│ new-code scoping via git diff (NEW)      │   │ 3-arm runner + bootstrap CI + verifier(NEW) │
│ emit: native JSON | generic-issue | SARIF│   │ price table, date-pinned (NEW)              │
└──────────────┬───────────────────────────┘   └───────────────┬────────────────────────────┘
               └───────────► DuckDB results tables ◄───────────┘
                              │  per-skill aggregates (R-AT-10): A, U, C_a, F
                              ▼
                    SCORING (new): SDY + two-lane queue + portfolio/trend view
                              ▼
                    CI gate on changed skill (L3 §3.3); optional SonarQube ingest
```

| Reused component | Role | Evidence |
|---|---|---|
| **skill-architect** `skills/skill-audit`, `skill-rewrite` | Author-facing UX; its bash checks (PL/PT) migrate into the rule engine | L1 §2d |
| **skill-architect/profiler** `types.go`, `compare.go`, `experiment.go` | `MetricResult` honesty contract and experiment skeleton — sound at type level, broken at compare level (D-1–D-9) | L2 §5.1–5.2 |
| **cursor-profiler** | Dynamic data plane: hook ingest → spool → `profile/v1` + per-skill aggregates | spec.md north star; Q9 |
| **DuckDB** | Query engine over JSONL spool; rebuildable derived cache | questions.md Q9 |
| **OTel** | Claude Code: `skill.name` on `claude_code.token.usage`/`cost.usage` gives A and cost-when-active free. `gen_ai.*` semconv is **export-alias only** — unreleased, `development` stability, `gen_ai.skill.*` in unmerged PR #498 | L2 §2, summary; OTel §1–4 |
| **skill-validator / SkillSpector / jscpd / agnix** | BPE counts & spec validation / 71 security patterns / Markdown-aware dup / 455-rule conformance — **do not rebuild** | L1 §3 |
| **SonarQube** (if present) | Ingests generic-issue JSON; its Shell+Secrets analyzers already cover `scripts/*.sh` | L1 §1, §4 |

**New components:** the rule engine + reference-graph analyzer; the SDY scorer and two-lane
portfolio view; the three-arm experiment runner with stats package and verifier hooks; the emitter
layer. Everything else is borrowed.

---

## 4. The measurement contract

Tier hierarchy (L2 §3.3 = spec.md Measurement Contract — non-negotiable):

| Tier | Signal | May claim |
|---|---|---|
| 1 Billed | Admin `cost_report`/`usage_report` — **unavailable to this team** (Q2) | "cost", "spend" |
| 2 Measured tokens, estimated $ | `claude_code.token.usage`, `api_request` events | real token deltas; "estimated cost" |
| 3 Vendor-estimated $ | `claude_code.cost.usage`, Analytics API | carry vendor's "estimated" verbatim |
| 4 Self-estimated | chars/4 over hook payloads — **Cursor's only token signal** | **ordinal only**: "less than" — never absolute, never dollars |

**Rules:** never compare across tiers; every delta stamps the source of *both* sides; any tier-4
operand → line reads "relative estimate, not billed" and renders no `$`; intervals, never point
estimates; `sum(tokens) by skill` is attribution, not causation (L2 §2.1).

**"Proved" means (L2 §3):** paired bootstrap BCa CI on per-task deltas **excludes 0** on the primary
metric, **and** verifier pass-rate does not regress (McNemar), **and** lift vs arm A stays positive,
**and** the achieved MDE is printed beside every non-significant result so "no difference detected"
is never misread as "no difference." Power: ~12 paired cases detects an average-sized lift, ~216 a
small one — a 10-rep arm-B pilot sizes N before committing (L2 §3.2, §3.4).

**The tool refuses to claim:** billed spend without a tier-1 source; a dollar figure off a tier-4
delta; significance below the funded N (`stopping_rule: threshold` is removed or alpha-guarded —
D-6); a causal per-skill cost from telemetry alone; a quality improvement with no verifier (a token
saving is indistinguishable from a capability loss — D-9).

*Where reports disagree:* L2 §3.3 forbids dollar rendering on tier-4 deltas; L3 §1.3 publishes
estimated portfolio dollars. **Resolution:** per-run comparison reports are strict (ordinal); the
portfolio may carry labeled `estimated` dollars because the confidence term `c` discounts them
mechanically and every cell keeps its source label (L3 §3.5). Estimates may feed *ranking*; they may
never be presented as *spend*.

---

## 5. Build options

| Option | Gets you | Costs you | Verdict |
|---|---|---|---|
| **(a) SonarQube Java plugin** (skill = new "language") | Native rules page, profiles, debt, hotspot lifecycle, trends | Java codebase + mandatory server + a Markdown grammar; nobody without SonarQube can run it (L1 §4a) | Reject |
| **(b) Standalone Go engine → generic-issue JSON (+ SARIF)** | Runs everywhere; `effortMinutes` feeds Sonar debt; issues count toward quality gates; matches Go stack and self-contained rule | Imported rules invisible on Sonar's Rules page/profiles — **we own the profile config and hotspot "reviewed" baseline** | **RECOMMENDED** (L1 §4) |
| **(b′) SARIF only** | GitHub Code Scanning free | **Disqualifying:** Sonar's SARIF importer forces `SECURITY` quality + `CONVENTIONAL` attribute and drops effort — destroys the debt model (L1 §4b′) | Reject as primary |
| **(c) Harness plugin only** (status quo) | Zero infra; lives where authors work | No CI gate, no machine-readable interop, structurally single-skill (L1 §4c) | Keep as thin front end over (b) |

**One caveat to verify before building the economics on Sonar ingest:** that imported generic issues
with `softwareQuality: MAINTAINABILITY` + `effortMinutes` roll into `sqale_index` is docs-assistant
asserted, not primary-page verified — test on a throwaway instance (L1 unresolved; questions.md).

---

## 6. First three slices — each independently shippable

| # | PR | Ships | Proves | Depends on |
|---|---|---|---|---|
| **1** | **Static engine core** | Go binary, SK-* minimum rule set with effort minutes (L1 §4), reference graph + cycle detection, catalog mode, native JSON + SARIF emitters, exit codes | Debt-in-minutes and cycle/dead-ref findings on a real skill catalog — zero model calls. Run it on skill-architect's own skills | none |
| **2** | **Honest comparator** | `skill-architect/profiler` fixes D-1–D-5: cross-tier refusal, `EstimatedContextTokens` compared + labeled, nil-guard, `skill.name` read, per-task activation (R-AT-01–04; questions.md: lands first anyway) | Two fixture profiles with mismatched sources refuse to delta; two hook-sourced profiles produce a labeled ordinal delta | none — disjoint codepath from PR1 |
| **3** | **Aggregates → SDY ranking** | cursor-profiler `analyze` per-skill aggregates over the spool (A, U, C_a, F — R-AT-10) + SDY scorer + two-lane table, on captured/fixture corpora | End-to-end ranking on real usage data with every cell source-labeled — the "next best skill" answer exists before any experiment runs | none — needs only the spool |

**Deliberately later (not in the first three):** the three-arm experiment runner with stats (D-6–D-9
machinery — needs PR2's comparator and funded N); generic-issue emitter (needs a SonarQube user, §7);
CI gate (needs a skill repo that wants gating; block-vs-warn undecided). This ordering is
cheap-before-expensive — the field's own tier model (L3 §3.2).

---

## 7. Risks and open decisions for the user

### Decisions needed (from the reports' own lists)

| # | Decision | Why it matters | Source |
|---|---|---|---|
| 1 | **Claude Code first or Cursor first?** `skill.name` attribution exists only in Claude Code; Cursor needs Enterprise OTel (absent) — Cursor-first keeps us on tier-4 ordinal data and the untested U-02 activation heuristic | scopes everything | L2 dec. 1; Q2 |
| 2 | **Billed or estimated dollars?** One Admin API key moves the product from "ranking tool" to "cost tool" and lifts `c` off its 0.5 cap | cost/policy call | L2 dec. 2; L3 dec. 1 |
| 3 | **Who calibrates `effortMinutes` and author-cost/line?** Uncalibrated SQALE = confident wrong numbers; measurement commitment, not coding | blocks debt ratio + all `$` claims | L1 dec. 2 |
| 4 | **Vendor vs shell-out vs reimplement** for SkillSpector (Py 3.12) / jscpd (Node) | trades the repo's self-contained rule | L1 dec. 3 |
| 5 | **Single-skill or catalog scope?** Cross-skill rules (dup, collision, cycles) are the unclaimed value but changing skill-audit's contract is a breaking change to a shipped plugin; `PL*`/`PT*` → `SK-*` is a user-visible break | interface decision | L1 dec. 4–5; L3 §4 |
| 6 | **Fund arm A and what N?** Arm A ≈ +50% experiment cost but makes "improved" falsifiable vs "delete it"; N=5 cannot support a significance claim — tool should refuse to print one | experiment budget | L2 dec. 3–4 |
| 7 | **Org constants `C_f` + engineer-hour rate** — set Risk$ vs Waste$ weight in SDY | owned assumption | L3 dec. 2 |
| 8 | **Hotspot lane: block merges or warn?** A heuristic script scan will false-positive on day one | CI policy | L3 dec. 3 |
| 9 | **Cross-user aggregation consent** — Q10 keeps prompt text + file paths on disk; pooling that across people is a privacy decision | legal/social | L3 dec. 4 |
| 10 | **Is SonarQube a target or a metaphor?** If nobody runs a server, the generic-issue emitter is speculative surface — defer behind SARIF + native JSON | scope | L1 dec. 1 |
| 11 | **May `r` render negative in the portfolio?** 27.2% of skills hurt — a UI that only shows improvement hides deletion candidates, the most valuable finding | product call | L3 dec. 5; L2 §1.1 |

### Risks

| Risk | Mitigation in design |
|---|---|
| Sole Cursor token signal is chars/4 (tier 4) → headline dollar claims are impossible on this harness today | contract §4 enforced in code (R-AT-01/02/11); admin-key path kept open (`server_api` enum, Q2) |
| U-02 hook activation heuristic is the only source of `A` and is untested | corpus validation is step 3 of the decided order of work; label `inferred` until then (questions.md) |
| Underpowered experiments read as "no difference" | achieved-MDE line mandatory (R-AT-07); unguarded `stopping_rule` removed (D-6) |
| `gen_ai.skill.*` may never land or land differently (PR #498 open; #500 already closed) | internal schema carries no `gen_ai.*` names; export alias gated and off by default (OTel R16, S-2) |
| Cursor hook format undocumented; `cursor.*` OTel names unverified (R-SA-14) | tolerant parser, fixture corpus, no invented names (spec.md risks) |
| Token savings can hide capability loss; a recall-destroying refactor looks like a big win | verifier gate + should-not-fire stratum + precision/recall reported with every delta (L2 §2.1) |

---

*Word to the team: the honest sentence this tool exists to make sayable is — "this refactor saved an
estimated $X/yr, CI [a,b], measured at tier N, and here is the next skill worth your Tuesday."*
