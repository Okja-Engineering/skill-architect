# "SonarQube for Agent Skills" — team synthesis

**Candidate: GAMMA · 2026-09-11.** Synthesizes `mandate.md`, `L1-static-report.md` (L1),
`L2-dynamic-report.md` (L2), `L3-prioritization-report.md` (L3),
`docs/research/otel-genai-alignment.md` (OTel), `spec.md`, `questions.md`.
Citations are section-level: `[L1 §2c]`, `[L2 §4.3]`, `[spec.md R-AT-07]`, etc. All
walkthrough numbers are **placeholders** chosen to demonstrate the pipeline, not findings.

---

## 1. The pitch in five lines

1. **What:** a static + dynamic audit tool for Agent Skills — rules with remediation-minutes, a paired-experiment engine that proves a refactor helped, and a portfolio ranker that names the next skill to fix.
2. **For whom:** AI-native dev teams whose skills silently fail — never trigger, call scripts that don't exist, or actively hurt (`PRINCIPLES.md`; ACES: **27.2% of measured skills have negative lift** `[L2 §1.1]`).
3. **Why now:** the crowded layers are already solved — spec conformance (agnix, 455 rules), security (SkillSpector, 71 patterns), duplication (jscpd) `[L1 §3]` — while the valuable layers are unclaimed: **remediation-cost economics, cross-skill reference graphs, new-code scoping, portfolio ranking** `[L1 §3 verdict]`.
4. **Why us:** we already own both halves — `skill-architect` (static checks + a Go profiler with a paired-experiment skeleton) and `cursor-profiler` (hook ingest → spool → profiles) — and both are specified for exactly this purpose `[spec.md §North star]`.
5. **The catch we will not hide:** on this team's environment every dollar figure is an **estimate** (no Admin API key, no Enterprise OTel) `[questions.md Q2]`, so honesty machinery — source-tier labels, per-side sources, CIs — is the product's core, not a footnote.

---

## 2. The workflow — one skill, end to end

Skill under audit: `deploy-runbook` (placeholder). Environment: Cursor hooks, no admin key → all dollars below are **estimated** (tier 2–4 `[L2 §3.3]`).

### Step 0 — Inventory + static audit (costs no model calls)

`skill-audit` engine scans the catalog; `deploy-runbook` returns `[L1 §4 rule table, effort values are uncalibrated placeholders]`:

| Finding | Rule | Quality / Severity | Effort |
|---|---|---|---|
| `scripts/bootstrap.sh` referenced in a fence, absent on disk | SK-R001 | Reliability / High | 10 min |
| `curl … \| sh` inside `scripts/install.sh` | SK-S001 | Security / Blocker **hotspot** | 30 min |
| Body = 5,800 `o200k` tokens (> 5,000 spec recommendation) | SK-M002 | Maintainability / Medium | 90 min |
| Reference chain depth 3 (> 1 per spec) | SK-M003 | Maintainability / Medium | 45 min |

**E = 175 min ≈ 2.9 h.** The hotspot also enters the **hotspot lane** — it is queue-jumped by blast-radius × permission scope regardless of the EV math below `[L3 §1.2]`.

### Step 1 — Baseline (dynamic, no experiment yet)

From the hook spool via `cursor-profiler analyze` (per-skill aggregates `[spec.md R-AT-10]`):

| Input | Value | Source | Label |
|---|---|---|---|
| `A` activations/user/week | 12 | spool `beforeReadFile` → SKILL.md heuristic | `inferred` — untested `[L3 §1.3, U-02]` |
| `U` users | 8 | spool `user_email` | observed |
| `C_a` cost per activation | $0.34 | `chars/4` sidecar | **estimated** |
| `F` failure fraction | 0.02 | `postToolUseFailure` share | inferred |
| `B` blast radius | 1.5 | static: one network-egress script | observed (static) |
| `C_f` cost per failure | $10 | **declared org assumption** | assumption |

→ **4,992 activations/yr** (`12 × 8 × 52`).

### Step 2 — Refactor

Fix the dead ref, pin the installer, move `references/` one level up, trim the body to 4,200 tokens. Static re-run: 0 findings, debt cleared. This static delta is free, exactly reproducible, and reported as the floor of every before/after claim `[L1 handoff]`.

### Step 3 — Prove (three-arm paired experiment `[L2 §1, R-AT-05..09]`)

- **Arms:** A = no skill (sanity floor, cached per task-set × pinned model), B = v1, C = v2.
- **Task set:** 15 should-fire + 10 near-miss + 5 neutral `[L2 §1.3]` — 30 paired tasks > the N ≈ 12 needed to detect an average-sized lift at 80% power `[L2 §3.2]`.
- **Held constant:** dated model snapshot, effort/speed, harness version, repo SHA, task text, permissions; arms interleaved in time. RNG cannot be held — replication is the only variance control `[L2 §1.2]`.

Measured per-activation delta, C−B, per token class `[L2 §4.3]`:

| Class | Δtokens | Rate ($/MTok, Opus 5, date-pinned) | Δ$ |
|---|---:|---:|---:|
| base input | −2,000 | 5.00 | −0.0100 |
| cache read | −16,000 | 0.50 | −0.0080 |
| cache write | −500 | 6.25 | −0.0031 |
| output | −400 | 25.00 | −0.0100 |
| **per activation** | | | **−$0.0311**, BCa 95% CI **[−$0.045, −$0.017]** |

Guard metrics: verifier pass rate 11/12 → 11/12 (McNemar, n.s. — no capability regression); recall 14/15 → 15/15; precision held at 9/10 near-misses `[L2 §2.1]`. A token saving with a recall collapse would be reported as a **loss**, not a win.

### Step 4 — Dollars

`annual_$ = Σ(Δtokens × price) × A × U × 52` `[L2 §4.1]`:
4,992 × $0.0311 = **$155/yr, estimated, 95% CI [$85, $225]**.
(Naive "all input at $5/MTok" would print $512/yr — a 3.3× overstatement; per-class pricing is mandatory `[L2 §4.2–4.3]`.)
Footnote: the −28-token listing trim saves ~$2/yr at cache-read prices — context-pressure play, not a cost play `[L2 §4.3]`.

### Step 5 — Risk/impact score → next-best-skill

`SDY = (Waste$ + Risk$) × c ÷ E` `[L3 §1.2]`:
`Waste$ = $155`; `Risk$ = 4,992 × 0.02 × 1.5 × $10 ≈ $1,500`; `c = 0.8` (hooks + paired reps `[L3 §1.3]`); `E = 2.9 h` → **SDY ≈ $456 recovered per engineer-hour per year**, all estimated.
The portfolio table ranks every skill by SDY; `deploy-runbook`'s hotspot (`curl|sh`) is reported in the **hotspot lane** whether or not its SDY leads `[L3 §3.4 portfolio view]`. "Next best" = the top of the re-scored list.

---

## 3. Architecture

```
┌─ STATIC ENGINE (new: Go binary, extends skills/skill-audit) ──────────┐
│ SKILL.md+scripts → SK-* rules (effortMinutes, severity×quality)       │
│   reuses: skill-validator check -o json (real o200k BPE) [L1 §2b/2d] │
│           jscpd (duplication), SkillSpector (security)*, skills-ref   │
│   new: reference graph + cycle detection (Tarjan), profile config,    │
│        hotspot baseline file, emitters: native JSON → Sonar           │
│        generic-issue JSON (primary) + SARIF (secondary) [L1 §4]       │
└──────┬───────────────────────────────────────────────────────────────┘
       │ findings, E (remediation h), B/permission scope, dead-ref/cycle flags
┌──────┴───────────────────────────────────────────────────────────────┐
│ DATA STORE — JSONL spool = source of truth; DuckDB queries over       │
│ read_json_auto('spool/*.jsonl'); .duckdb is a rebuildable cache       │
│ [questions.md Q9]. JSON artifacts: profile/v1, audit reports,         │
│ experiment reports (pinned by snapshot_hash)                          │
└──┬───────────────────────────────┬───────────────────────────────────┘
   │ spool rows                    │ profiles
┌──┴─ DYNAMIC ENGINE ──────────────┴──────────────────────────────────┐
│ cursor-profiler: Cursor hooks → spool → capture → profile/v1;       │
│   analyze → per-skill A, F, est. C_a [spec.md R-AT-10]              │
│ skill-architect/profiler: adapters per harness; experiment.go →     │
│   3-arm runs; compare.go → paired deltas (after D-1..D-9 fixes      │
│   [L2 §5]); aggregation: bootstrap BCa CI, McNemar, achieved-MDE    │
│ OTel (read-only where available): Claude Code skill.name on         │
│   claude_code.{token,cost}.usage gives A + cost-when-active free    │
│   [L2 summary]; cursor.skill.activated = Enterprise-only [L3 §1.3]  │
└──┬──────────────────────────────────────────────────────────────────┘
   │ A, U, F, C_a, Δtokens±CI, activation precision/recall
┌──┴─ SCORING (new) ──────────────────────────────────────────────────┐
│ SDY = (Waste$ + Risk$) × c ÷ E per skill → portfolio table;         │
│ hotspot lane ordered by B × permission scope, independent of SDY    │
│ [L3 §1.2]; CI quality gate on the *changed* skill (new-code focus)  │
│ [L3 §3.3]                                                           │
└─────────────────────────────────────────────────────────────────────┘
```

\* SkillSpector/jscpd integration mode (vendor / shell-out / reimplement-subset) is an open user decision `[L1 decision 3; questions.md parked]` — shelling out violates the self-contained rule, vendoring means owning 71 patterns.

**New components:** the Go rule engine, the reference-graph builder, the Sonar-generic-issue + SARIF emitters, the N-arm experiment runner + stats layer, the date-pinned price table, the SDY scorer + portfolio renderer. **Reused:** everything else above. **Explicitly not built:** a SonarQube Java plugin, a daemon, an OTLP exporter (parked; `gen_ai.*` is entirely `development`-stability and `gen_ai.skill.*` exists only in unmerged PR #498 → any future export uses our own `cursor_profiler.*` namespace, gated `[OTel §4b, §5]`).

---

## 4. The measurement contract

### 4a. Billed vs estimated — the four tiers `[L2 §3.3; spec.md §Measurement contract]`

| Tier | Signal | May be called | On this team today |
|---|---|---|---|
| 1 Billed | Admin `cost_report` / `usage_report` | "cost"/"spend" | unavailable — no admin key `[Q2]` |
| 2 Measured tokens, estimated $ | harness `token.usage` | "estimated cost" | where present |
| 3 Vendor-estimated $ | `cost.usage` | vendor's "estimated" label, verbatim | where present |
| 4 Self-estimated | `chars/4` over hook payloads | **ordinal only** — "C < B", never a dollar | our default |

Non-negotiables: **never compare across tiers** (a tier-2 baseline minus a tier-4 candidate is not a delta — `compare.go` does exactly this today, defect D-1 `[L2 §5.2]`); every delta carries **both** sides' source; tier-4 output says *"relative estimate, not billed"* on the same line and renders no `$`; per-class pricing only.

### 4b. What "proved" means `[L2 §3]`

- **Design:** three arms (A amortized), stratified task set (≥15/≥10/≥5), pinned model+harness+SHA, interleaved arms.
- **Test:** paired bootstrap BCa CI over per-task deltas (token deltas are heavy-tailed: −76.9% to +120.3% published effects → normality unsafe); McNemar for binary; per-task delta table beside the grand mean (Simpson's-paradox guard).
- **Power:** pilot = 10 arm-B reps to measure σ before committing N. Reference points: N≈12 detects an average lift; N≈216 a small one `[L2 §3.2/3.4]`.
- **Honesty floor:** the achieved MDE is printed next to every non-significant result — "no difference detected" is never reported as "no difference" `[R-AT-07]`. `stopping_rule: threshold` is guarded or removed (unguarded peeking manufactures significance, D-6).

### 4c. What the tool refuses to claim

- A causal cost from `sum(tokens) group by skill.name` — that is **cost-when-active**, attribution ≠ counterfactual; only paired arms give the causal delta `[L2 §2.1]`.
- Any dollar figure when either side is tier 4; any "spend" claim without tier-1 data.
- A significance claim when the funded N can't reach a meaningful MDE — it prints the MDE instead `[L2 decision 4]`.
- That "v2 beat v1" implies "v2 beats nothing" — arm A exists because 27.2% of skills have negative lift `[L2 §1.1]`.
- A token delta alone — always paired with verifier pass rate and activation precision/recall `[L2 §2.1]`.

---

## 5. Build options considered → recommendation

| Option | Gets | Costs | Verdict |
|---|---|---|---|
| (a) SonarQube Java plugin ("Agent Skill" as a language) | Native rules page, profiles, hotspot lifecycle, trend — full model free | Java codebase + server dependency + Markdown grammar; useless to anyone without SonarQube. Highest cost, narrowest reach `[L1 §4a]` | **Reject** |
| (b) Standalone Go engine → **Sonar generic-issue JSON** (primary) + SARIF (secondary) | `effortMinutes` survives → debt/gate math; runs with or without a Sonar server `[L1 §4b]` | Imported rules invisible on Sonar's Rules page → we own the profile config + hotspot baseline file | **Recommended** |
| (b′) Same, SARIF-only | GitHub Code Scanning free | **Disqualifying:** Sonar's SARIF importer forces `SECURITY`/`CONVENTIONAL` on every issue and drops effort → destroys the debt model `[L1 §4b′]` | Reject as primary |
| (c) Harness plugin only (status quo) | Zero infra; lives where authors work | No CI gate, no interop, no trend store; current impl. is single-skill so cycles/duplication unreachable `[L1 §4c]` | Keep as thin UX over (b) |

**Named tradeoff to accept:** (b) over (a) permanently gives up Sonar-side profile management and the native hotspot lifecycle — both become our config files `[L1 §4 rec.]`. **Gate:** if nobody on the team runs SonarQube, the generic-issue emitter is deferred and SARIF + native JSON ship alone `[L1 decision 1; questions.md parked]`.

---

## 6. First three slices — independent, each proves something

| # | PR | Contents | Proves | Depends on |
|---|---|---|---|---|
| 1 | **Honest comparison** (`skill-architect/profiler`) | Refuse cross-tier deltas; carry both `MetricSource`s; compare `EstimatedContextTokens` tagged *"relative estimate, not billed"*; nil-guard `LoadProfile`; per-task-by-name activation comparison; read `skill.name` on Claude Code OTel + delete the false "no activation events" claim — defects D-1..D-5 `[L2 §5.2–5.3; spec.md R-AT-01..04; questions.md Q12]` | The delta contract holds: no estimate is ever laundered as a measurement again. Small, all code+tests. | — |
| 2 | **Static engine v0** | Go binary: SK-* minimum rule set with `effortMinutes`, reference graph + cycle detection, per-rule severity×quality, native JSON report `[L1 §4, min rule set]` | Debt-minutes and cycles are computable on a real catalog with **zero model calls** — the "audit" half stands alone. | — |
| 3 | **Per-skill aggregates** (`cursor-profiler analyze`) | DuckDB queries over the spool: activations/user/week, failure rate, est. cost/activation per `skill_name` `[spec.md R-AT-10; questions.md Q9]` | The portfolio's three dynamic inputs exist from real usage — first SDY-able row without any experiment. Also first real test of the untested U-02 activation heuristic `[L3 §1.3]`. | — |

**Deliberately not in the first three:** the N-arm experiment runner + stats (D-6..D-9 machinery) builds on slice 1's honest compare; SDY scoring + portfolio UI is inert until slices 2 (E) and 3 (A, F, C_a) emit their inputs `[L3 handoff: "sequence those two before any ranking UI"]`; the CI quality gate waits for a skill repo that wants gating `[questions.md parked]`.

---

## 7. Risks and open decisions for the user

### Decisions needed (real, sourced)

| # | Decision | What's at stake | Source |
|---|---|---|---|
| 1 | **SonarQube: target or metaphor?** | If no one runs a server, the generic-issue emitter is speculative surface — defer it | `[L1 dec.1; questions.md]` |
| 2 | **Who calibrates `effortMinutes` and author-cost-per-line?** | Uncalibrated SQALE produces confident, wrong dollar figures; it's a measurement commitment, not a coding task | `[L1 dec.2]` |
| 3 | **Buy billed tokens or stay estimated?** | One admin key moves the product from "ranking tool" to "cost tool" and lifts `c` above its 0.5–0.8 cap; today every `$` is estimated | `[L2 dec.2; L3 dec.1; Q2]` |
| 4 | **Harness scope: Claude Code first or Cursor first?** | `skill.name` attribution is free on Claude Code; Cursor has nothing equivalent outside Enterprise → Cursor-first keeps us at tier 4 | `[L2 dec.1]` |
| 5 | **SkillSpector/jscpd: vendor, shell out, or reimplement?** | Self-contained rule vs owning 71 security patterns | `[L1 dec.3]` |
| 6 | **Portfolio vs single-skill static scope** | Cross-skill rules are where the unclaimed value is, but widening `skill-audit`'s contract is a breaking change to a shipped plugin | `[L1 dec.4; L3 §4]` |
| 7 | **Fund arm A and fund N** | Arm A ≈ +50% experiment cost but is amortizable; without it "we improved" is unfalsifiable vs "delete it". If funded N is ~5, the tool should refuse significance claims | `[L2 dec.3–4]` |
| 8 | **Hotspot lane: block merges or warn?** | Sonar/Datadog gates block; a heuristic script scan will false-positive on day one | `[L3 dec.3]` |
| 9 | **Own the org constants** `C_f` + engineer-hour rate | They set the Risk$ vs Waste$ weight — visible in §2's walkthrough, where Risk$ is ~10× Waste$ | `[L3 dec.2]` |
| 10 | **Cross-user aggregation consent** | Portfolio ranking needs per-user data; Q10 keeps prompt text/paths on disk by default — privacy call | `[L3 dec.4; Q10]` |
| 11 | **Show negative lift / deletion candidates?** | 27.2% of skills hurt; a UI that only shows improvement hides the most valuable finding | `[L3 dec.5]` |
| 12 | **Rule-ID namespace** PL*/PT* → SK-* | Published exit-code contract; a user-visible break | `[L1 dec.5]` |

### Risks / thin evidence flagged by the reports

| Risk | Severity | Source |
|---|---|---|
| **U-02 hook-based activation heuristic is untested** — and it is this team's *only* source of `A` | high | `[L3 §1.3, handoff; questions.md]` |
| `effortMinutes` on imported generic issues rolling into `sqale_index` is docs-assistant asserted, not primary-page verified — verify on a throwaway SonarQube before building economics on it | medium | `[L1 unresolved]` |
| `gen_ai.skill.*` is unmerged PR #498 and every `gen_ai.*` name is `development` stability (7 breaking changes queued) — export layer only, own namespace, revisit at first tag | high if S7 ever ships | `[OTel §4–5; questions.md parked]` |
| Whether `skill.name` is emitted for listed-but-not-invoked skills is unresolved → false negatives may be detectable only via the task set, not telemetry | medium | `[L2 unresolved]` |
| `compare.go` today panics on round-tripped profiles and launders estimate sources (verified by execution) — the comparison path is unsafe until slice 1 lands | high, already fixed-scoped | `[L2 §5.2 D-1/D-3]` |
| ClawHavoc malicious-skill counts conflict across reports (1,184 vs 341) | low — cite range | `[L3 §2 R11]` |
| Unguarded `stopping_rule: threshold` is a significance-manufacturing machine | medium | `[L2 §5.2 D-6]` |

---

*Bottom line for the team:* build the standalone Go engine (option b), fix the comparison layer's honesty defects first, then sequence static engine and per-skill aggregates in parallel — each of the first three PRs ships alone and proves its layer works. Every dollar the tool ever prints carries its tier label or it doesn't print.
