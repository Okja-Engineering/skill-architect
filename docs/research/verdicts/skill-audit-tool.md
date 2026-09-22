# Arena verdict: skill-audit-tool synthesis

**Judge:** fresh hunter, wrote none of the candidates · **Date:** 2026-09-11
**Bar applied:** `arena.md` "Judging bar", B1–B7 only. B1/B2/B3 are must-pass.

**Coverage.** 3/3 candidates walked section-by-section (7/7 required sections each) × 7/7 criteria
= 21/21 scorecard cells. Every dollar, ratio, SDY, MDE, precision/recall and McNemar figure in all
three walkthroughs recomputed independently (46 arithmetic checks). Source spot-checks: L1 (rule
table, rating grid, landscape figures, §2b/§2d measurements), L2 (§1.1, §2.1, §3.1–3.4, §4.1–4.3,
§5.1–5.3 D-1…D-9), L3 (§1.2 formula, §1.3 input table, worked example, §2, §3.1–3.5, §4 handoff),
OTel report (§1, G1/G4, R1/R2, R16, R22/R23, S-1/S-2/S-5), `questions.md` Q2/Q3/Q4/Q5/Q8/Q9/Q10 and
environment facts, `spec.md:38,90,164`, and the live source at
`skill-architect/profiler/compare.go:106-125` and `claude_code.go:100-108`.

---

## 0. The disputed item, settled first

**All three candidates claim L3's own worked example does not reproduce from its own formula. All
three are right.** Recomputed from `L3-prioritization-report.md:69-73` inputs
(`A`=12/user/wk, `U`=8, `C_a`=$0.34, `r`=0.22, `F`=0.06, `B`=1, `C_f`=$40, `c`=0.5, `E`=3 h):

| Term | L3's formula (§1.2) | Correct value | L3 states |
|---|---|---:|---:|
| `Waste$` = A×U×C_a×r | 12 × 8 × 0.34 × 0.22 | **$7.18/wk → $373/yr** | $9.0/wk → ≈$470/yr |
| `Risk$` = A×U×F×B×C_f | 12 × 8 × 0.06 × 1 × 40 | **$230.40/wk → $11,981/yr** | ≈$920/yr |
| `SDY` = (W+R)×c ÷ E | (373 + 11,981) × 0.5 ÷ 3 | **≈$2,059/eng-hour** | ≈$232 |

L3's SDY *is* internally consistent with its own (wrong) $470/$920, so the slip is upstream in the
two dollar terms — `Waste$` is off by ~1.26×, `Risk$` by ~13×, i.e. `Risk$` looks like it was
annualised from a monthly figure. A (`A:279`) and B (`B:217`) state the annual pair correctly.
C (`C:303`) additionally shows the weekly step, which is the only version that *locates* the slip.
L3 itself says "do not take the worked example's numbers" (`L3:197`), so no candidate is penalised
for L3's error — only for how cleanly it disowned it. All three disowned it and re-derived.

---

## 1. Per-candidate scorecards

### Candidate A — `Skill Debt` / `skillgate` (4,299 words)

| # | Criterion | Score | Evidence |
|---|---|---|---|
| **B1** | Traceability | **PASS** | Densest citation apparatus of the three; source key at `A:5-10`. Spot-checks all hold: agnix 455 rules / SkillSpector 71 patterns, 16,984★ / jscpd (`L1:9,126-129`); effort minutes 20/90/45/10/30 = exactly `L1:171-184`; `compare.go:115` **correct** (verified in source); `claude_code.go:104` correct; OTel R1/R2/R16/R22/R23, S-2, #498 open, #500 closed, "zero releases", "Schema URL: TODO" all verbatim-correct against `otel-genai-alignment.md:21,28,212-213,236,247-248,261`; `[Q]` claims (Q2, Q9, Q10, "Cursor is not even installed on the dev Mac") match `questions.md:15,25,32,33`. Only stale item is the shared OTel-path claim (see §5 defect 1). **A is the only candidate that surfaces OTel R1/R2 — the two renames that report says are worth doing regardless** (`otel:276`). |
| **B2** | Number flow | **FAIL (must-pass)** | `A:90` — `\| — \| deploy-helper \| 120 \| 8,900 \| 0.5 \| 1.5 \| **210** \| HOTSPOT \|`. **(120 + 8,900) × 0.5 ÷ 1.5 = 3,007, not 210.** This is not a rounding slip: at 3,007 `deploy-helper` ranks **first** on EV, which inverts A's own stated conclusion at `A:92-93` ("It ranks last on EV and is fixed **first**") and destroys the demonstration the two-lane split exists to make. Three further breaks: (i) `A:126` `15,600 × $0.0311 = $486` — it is **$485.16**, and the "cross-check ✓" printed beside it, `15,600 × $0.34 × 0.092`, yields **$488**, so the tick-marked reconciliation reconciles to a third number; (ii) `A:45` "`always_loaded_tokens` 1,430 (over the 1,536-char listing cap)" — tokens compared to a character cap, and 1,430 < 1,536 regardless, so the finding that opens the walkthrough is false on its own face; (iii) `A:143` post-refactor `Waste$` = $265 is computed from the **pre**-refactor `C_a` $0.34; the table's own post-refactor `C_a` ($0.3089) gives $241. Everything else in A recomputes exactly (SDY 2,323 / 2,025 / 605 / 1,888; McNemar exact p = 2×0.5⁶ = 0.031 ✓; 6/8 = 0.75 ✓; Risk$ 42,120 ✓; 87× ✓; $7.28/yr ✓; precision/recall from TP13 FN2 FP1 TN9 ✓). |
| **B3** | Honesty | **PASS — strongest of three** | Four evidence tiers (`A:199-210`) with the added, decisive line "**Today, on this team's environment, `C_a` is tier 4**". `A:133-136` "Two figures, never one number… **refuses to sum them**" — only candidate to make non-summation a product rule. Nine-row refusal table (`A:221-233`), including refusing the A–E letter outright until the denominator is calibrated (`A:229`, the correct reading of `L1:186`). "proved" table (`A:213-219`) pins BCa CI + exact McNemar + achieved-MDE-on-every-null + σ from a 10-rep pilot. `c` frozen at 0.5 and justified. |
| **B4** | Shape | **HIGH** | Explicit boundaries (`A:184-188`): "the static engine never needs telemetry; the scorer never parses Markdown"; `profile/v1` additive-only. Component table carries a *contract* column. Catalog-first inversion with the breaking change named and costed. De-risks the Sonar bet properly: emitter is "one file behind the native report", plus "verify on a throwaway Sonar instance **before** building economics on it" (`A:250-253`, correctly sourced to `L1:202`'s thin-evidence flag). Adds only one guard of its own (the `E` ≥ 1.0 h floor), and flags it as its own. |
| **B5** | Readability | **MEDIUM** | Longest of the three (4,299 words, ~8–9 pages vs the ≤6-page bar) and the heaviest reader load: a five-key source system (`[L1][L2][L3][OTel][Q][S]`) plus inline `§`-refs on nearly every row, and a parenthetical path-doesn't-exist argument inside the source key itself. Tables-over-prose is honoured. |
| **B6** | Sequencing | **MEDIUM** | PR1 static core / PR2 comparator honesty / PR3 portfolio, each with a "proves" and a "verified by" column — good. But PR3 "consumes PR1's file" (`A:263`), so it is **not** independent of PR1 and A offers no degrade path (contrast B, which does). 2 of 3 genuinely independent. |
| **B7** | Candor | **HIGH — richest** | 4 disagreements each with an explicit pick, 8 user decisions, 5 residual risks. Real, not padding: "conflicting source counts we did not resolve (ClawHavoc 1,184 vs 341; skillscore 6 vs 7 dimensions) → **do not cite either figure in product copy**" (`A:304`, traces to `L3:93,201` and `L1:135`). D2 makes calibration a named owner's problem ("time 10 real fixes before publishing any ratio"). |

### Candidate B — `skillaudit` (3,858 words)

| # | Criterion | Score | Evidence |
|---|---|---|---|
| **B1** | Traceability | **PASS** | Every spot-check holds. `compare.go:115` **correct**; `compare.go:80-88` mismatch-mechanism reference correct (`L2:206`); `claude_code.go:104` correct; rating grid A ≤5% / B 5–<10% correct against `L1:27`; effort minutes 90/45/10/30/5 exactly `L1:171-184`; σ/mean = 1.23 and N ≈ 12/54/216 exactly `L2:110,125`; MDE 0.164 = 77% at N=20 exactly `L2:119`; 27.2% negative lift, −76.9%/+120.3%, ACES CI all correct; OTel "seven breaking fragments" = `otel:264` S-5 ✓, "no `gen_ai.*` literal in `types.go`" is `otel:260` S-1's own mitigation verbatim ✓; `[Q2][Q4][Q5][Q8][Q9][Q10]` all match `questions.md`; `[spec:90][spec:164]` resolve. No invented facts found. |
| **B2** | Number flow | **PASS** | End-to-end chain recomputes: debt 180 min → 3.0 h ✓; ratio 180 ÷ (3 × 620) = 9.68% → rating **B** ✓ (grid-correct); 12 × 25 × 52 = 15,600 ✓; $5,304 ✓; Waste$ 530 ✓; Risk$ 15,600 × 0.004 × 1.25 × 40 = 3,120 ✓; SDY (530+3,120) × 0.5 ÷ 3 = **608** ✓; Δ$ −0.0311 ✓; annual **$485** ✓ (the only candidate that gets this exactly right); measured `r` 0.092 ✓; post: Waste$ 96 ✓ (15,600 × 0.3089 × 0.02), Risk$ 624 ✓, SDY (96+624) × 0.8 ÷ 0.58 = **994** ✓; all four portfolio rows recompute exactly (deploy-helper 397, commit-msg 409, test-writer 308, pr-review 994). **Two minor gaps, neither breaking the chain:** (i) `B:86` post-refactor ratio "2.2%" — 35 ÷ (3 × 620) = **1.9%**; 2.2% needs a line count of ~530, i.e. an unstated line-count change after the split; (ii) `B:99,104` calls `deploy-helper` "near-bottom on EV" when at SDY 397 it sits 2nd of 3 EV-lane rows (above test-writer's 308) — rhetoric outruns its own table. |
| **B3** | Honesty | **PASS** | Four tiers with a "may be called" column (`B:144-149`) and the three non-negotiables quoted from `L2:140-143` ✓. Tier 4 → "**no dollar figure may be rendered at all**" ✓. Per-cell `(assumed)/(measured)/(billed)` marks throughout §2 — the most granular labelling of the three. "proved" table with BCa primary + Wilcoxon backstop + McNemar + MDE-on-nulls + 10-rep pilot ✓. Refusal table includes "No absolute SDY presented as money until `C_a` is billed" ✓. **Two blemishes:** (i) `B:37,86` ships an A–E **rating** off an explicitly *(assumed)* 3 min/line denominator, and the risk-row mitigation at `B:224` — "Ship §2a as `rating` only until calibrated" — is backwards: the rating is precisely the part `L1:186` says is unfounded until calibration; minutes are the defensible part; (ii) `c`-derivation contradicts itself — `B:215` sets the rule "tier 2 = 0.8, tier 4 = 0.5" while `B:46,54` label `C_a` tier 2 and then assign `c` = 0.5. `c` is the honesty dial; it must obey one stated rule. |
| **B4** | Shape | **HIGH — soundest** | The boundary column is the strictest in the arena and is written as prohibitions: `skillaudit` "**Never prices anything.** Minutes and tokens only"; `scoring` "**Never reads a file.** Takes report/v1 + query output only"; `skill-validator` "Never invent a token count when absent" (`B:123-134`). Data store: "there isn't a new one" (`B:136`) — the correct subtractive answer, and it matches `Q9`. **`b″` (`B:187-191`) is the single soundest architectural judgment in the arena:** it makes the native `report/v1` the contract and demotes both Sonar generic-issue JSON and SARIF to flagged ~150-line emitters, and it argues this from L1's *own* first open decision ("is SonarQube a target or a metaphor? — no evidence either way", `L1:192`), i.e. it refuses to make an unverified external integration the primary dependency. It then keeps every piece of L1's evidence intact (no Java plugin; SARIF never sole output). Named tradeoff (own the profile file + own the hotspot-review baseline) carried forward correctly from `L1:167`. Least new surface of the three. |
| **B5** | Readability | **HIGH — best** | Shortest (3,858 words), flattest structure, one citation key, tables throughout, and the `§2` walkthrough is one continuous story with no side-quests. Still over a strict 6-page read, but it is the closest. |
| **B6** | Sequencing | **HIGH — best** | Only candidate that solves the PR3-depends-on-PR1 problem instead of ignoring it: "**Degrades cleanly:** with no `report/v1.json` it ranks on Waste$/Risk$ and prints `E: unknown`" (`B:201`). PR2 is fully isolated (`skill-architect/profiler` only) and its test is "re-run L2's three executed repros". PR1 is pure filesystem. Three genuinely independent slices, each proving a distinct thing, with the expensive N-arm work correctly deferred. |
| **B7** | Candor | **HIGH** | 7 resolved disagreements with a named pick each; 6 risks; 7 user decisions. Sharpest single item: `B:218,234` names the **live contradiction** between L1's "shell out to SkillSpector/jscpd" recommendation and the already-decided `Q8` ("no Semgrep, no Python") plus `PRINCIPLES.md:24` self-containment — a conflict with the repo's own settled constraints, not a hypothetical. `B:229` "Cursor isn't on the dev machine; all development is against fixtures" turns an environment fact into a design consequence. |

### Candidate C — `skillsonar` (4,189 words)

| # | Criterion | Score | Evidence |
|---|---|---|---|
| **B1** | Traceability | **PASS, with two named breaches** | Most spot-checks hold and the `[O]/[I]/[P]` mark system is used consistently outside §2: rating grid D 20–50% ✓ (`L1:27`); `skill-audit` 1,814 body / 4,030 total / 55% deferred is a **real** measurement, correctly marked `[O]` and correctly attributed to `L1:65` ✓; effort minutes ✓; prices, MDE 0.134 = 63%, 27.2%, 3.3× all ✓; OTel §1 / R16 / R22 / R23 / S-2 / S-5 ✓; `Q2/Q4/Q8/Q9/Q10` ✓. **Breach 1 (`C:247`, `C:286`): `compare.go:113` asserted as "Verified at that line today **[O]**".** It is wrong — verified in the working tree, `Source: candidate.Source` is at **:115**; :113 is the `return MetricComparison{` line. `L2:214` says :115. An `[O]` that claims independent re-verification and is wrong is worse than an uncited number. **Breach 2 (`C:56-59`): the walkthrough runs on the *real* `skill-audit` skill and mixes two genuine `[O]` measurements with four findings carrying no mark at all** — "2 script paths behind `$skill_root` **never resolve**" (`L1:114` says only that `$`-paths *silently escape validation*, which is not the same claim), "`references/` chain is 2 deep", "listing text over the 1,536-char cap", and "**61% keyword overlap with `skill-rewrite`**" (no source anywhere; `L1:86` gives a ≥60% *threshold*, and a 24% mean overlap across 77 published skills). A reader will take these as measured facts about two named, real, in-house skills. |
| **B2** | Number flow | **PASS — most exact** | The only candidate whose *every* SDY recomputes to the printed digit because it uses unrounded `E`: pre (786 + 37,440) × 0.5 ÷ (115/60) = **9,972.0** ✓; post (238 + 6,240) × 0.8 ÷ (140/60) = **2,221.0** ✓. Also the only one to show **both arms' absolute token and dollar columns** and reconcile them: B $0.3361 and C $0.3050 each sum from their four class rows, and their difference is exactly the −$0.0311 delta (`C:107-113`) — that closes the gap A and B leave by quoting only a delta. Ratios: 115/520 = 22.1% → D ✓; 140/1,080 = 13.0% → C ✓; debt 115 − 65 + 90 = 140 ✓; realized `r` 0.0311/0.3361 = 9.3% ✓; CI propagation 0.017/0.045 × 15,600 = [265, 702] ✓; "Risk$ is 48× Waste$" ✓. **One slip:** `C:155` re-scored `skill-audit` at SDY 2,221 is labelled "**#4 of 5**"; against its own portfolio (`C:90-92`: 3,410 / 2,880 / 1,220 / 210) it is **#3 of 5**. The terminal conclusion ("next best = `skill-rewrite` at 3,410") is unaffected and correct. |
| **B3** | Honesty | **PASS — best apparatus, one self-contradiction** | The crispest "proved" definition in the arena (`C:231-238`): **four** conditions that must *all* hold — BCa CI excludes zero, McNemar shows no task-success regression, activation **precision does not fall** (measured on the should-not-fire stratum), and the **arm-A floor** `lift(C) > 0` — the last being the only place any candidate operationalises `L2:23`'s 27.2%-negative-lift warning as a gate rather than a caveat. `refusals[]` made **machine-readable in every report** (`C:243`) — the only candidate to make honesty an artifact rather than a policy. Best single sentence for a team lead: "**on today's environment the word 'billed' never appears in the product**" (`C:227`). And `C:157-158` voluntarily surfaces an artifact that flatters it ("the rating improved D→C while absolute debt *rose* 115→140, because the denominator grew. Both numbers ship; the letter never ships alone"). **The contradiction:** `C:250` F4 refuses "No letter rating / debt ratio before calibration (n ≥ 20)", yet `C:64,145` print ratings **D** and **C** from a `[P]` 2 min/line denominator with no ledger data in existence. The document violates its own refusal in the section a team lead reads first. |
| **B4** | Shape | **HIGH** | Best one-sentence seam in the arena (`C:197-198`): "**static supplies `E` and `B`; dynamic supplies `r`, `C_a`, `c`; the ranker owns the arithmetic and owns nothing else**", backed by an explicit producer→artifact→consumer→contract table (`C:190-196`). `C:200-209` **the calibration ledger** is the strongest single new idea in the arena: it converts L1's unowned "who calibrates `effortMinutes`?" (`L1:193`, escalated as above-my-level) into a measurement — record `(rule_id, finding_id, first_seen_commit_ts, fixed_commit_ts, observed_minutes)` from the skill dir's own git history, switch each rule's effort from the shipped prior to the team's observed median at n ≥ 20, and derive the author-cost-per-line denominator from the same ledger; until then suppress the letter and the ratio. Costs "one JSONL writer". **Mild gold-plating:** three new binaries (`skillsonar-rules`, `-exp`, `-rank`) plus three append-only ledgers is more surface than A's or B's single binary, and option (d) (`claude plugin eval` as arm runner) adds a fifth build option the mandate didn't ask for — though correctly dispositioned. |
| **B5** | Readability | **MEDIUM-HIGH** | 4,189 words (~8 pages), but the cleanest numbering (`2.1`…`2.5`, `7.1`…`7.3`) and the most self-explaining tables — the step-by-step arithmetic column in `C:72-85` means a lead can audit the model without the lens reports. Loses ground to B on length and to nobody else. |
| **B6** | Sequencing | **HIGH** | Three genuinely independent slices with the **best ordering**: PR1 is the honest comparator (the defect that is already shipping wrong answers), PR2 the rule engine ("new binary, no dependency on PR 1"), PR3 activation+cost inventory ("reads OTel only; PR 2's `E` slots in later"). And `C:288` makes PR3 state its own limit in the product: "says plainly '**this ranks spend, not value; SDY needs `E` from PR 2**'" — honesty pushed into the slice boundary itself. Ties B; B edges it only on the explicit degrade path. |
| **B7** | Candor | **HIGH** | 4 conflicts with picks, 6 risks, 8 decisions — and the decisions table carries a "**Default if you say nothing**" column (`C:318-327`), the most usable candor device in the arena: it makes silence safe, pre-commits the tool's behaviour, and converts a decision list into a reviewable proposal. `C:301` also makes a *harder* call than L3: caps `c` at 0.8 without billed data, explicitly overriding `L3:66`'s `otel = 1.0`, on the correct ground that OTel tokens are measured but the dollars remain a local price table. |

### Summary matrix

| | B1 trace | B2 flow | B3 honesty | B4 shape | B5 read | B6 seq | B7 candor |
|---|---|---|---|---|---|---|---|
| **A** | PASS | **FAIL** | PASS (best) | HIGH | MEDIUM | MEDIUM | HIGH (richest) |
| **B** | PASS | PASS | PASS | **HIGH (best)** | **HIGH (best)** | **HIGH (best)** | HIGH |
| **C** | PASS (2 breaches) | PASS (most exact) | PASS (best apparatus) | HIGH | MED-HIGH | HIGH | HIGH |

---

## 2. Ranking

1. **B** — the only candidate clean on all three must-pass criteria with no structural defect. Soundest
   boundaries, soundest build recommendation (`b″`), best slice independence, most readable, shortest.
2. **C** — the most exact arithmetic and the best honesty *apparatus* in the arena, and the owner of the
   single best new idea (calibration ledger). Ranked second on shape, not on quality: it carries a false
   `[O]` citation, unmarked invented findings about two real in-house skills, a §2-violates-§4.3
   self-contradiction, and more new surface than the job needs.
3. **A** — richest candor and the only candidate to catch the two OTel renames worth taking regardless,
   but it **fails must-pass B2**: the `deploy-helper` SDY cell is wrong by 14× and its wrongness inverts
   the conclusion the row exists to demonstrate. Three further numeric breaks compound it.

---

## 3. Recommended base: **Candidate B**

Chosen by **shape**, per the arena's instruction, not by completeness — C is arguably more complete and
A is more exhaustive, and neither is the right spine.

B's shape is soundest for three reasons that survive independent of its content:

1. **Its boundaries are written as prohibitions, not responsibilities** ("never prices anything",
   "never reads a file", "never invent a token count"). A boundary stated as a prohibition is testable;
   a boundary stated as a role is not. This is what keeps the static engine, the query layer and the
   scorer from growing into each other as the tool accretes rules.
2. **`b″` inverts the one dependency that was pointing the wrong way.** L1 recommended Sonar
   generic-issue JSON as the *primary* format while its own first open decision recorded no evidence
   that anyone here runs SonarQube. B makes the native `report/v1` the contract and both emitters
   ~150-line flagged projections — so the economics cannot be held hostage by an unverified external
   integration (the very integration `L1:202` flags as asserted-by-docs-assistant, not primary-sourced).
   A reaches a similar place but keeps the Sonar emitter framed as primary-with-a-hedge; C reaches it
   too but via a longer argument. B states it as the design.
3. **Its slices are independent in fact, not in claim** — PR3 carries a degrade path
   (`E: unknown`) rather than a dependency it doesn't mention. A's PR3 consumes PR1's file with no
   such path.

B is also the shortest by ~340–440 words, which matters against a ≤6-page bar that all three overshoot.

---

## 4. Fold-ins from the runners-up (one each)

### From **C**: the calibration ledger (`C:200-209`, §3.3) plus its refusal F4

**What.** Record `(rule_id, finding_id, first_seen_commit_ts, fixed_commit_ts, observed_minutes)` for
every finding the team actually fixes, from the skill directory's own git history. At n ≥ 20 fixes per
rule, that rule's `effortMinutes` switches from the shipped prior to the team's observed median; the
same ledger yields the author-cost-per-line denominator. Until n ≥ 20, print debt in **minutes** and
suppress the letter rating and the debt ratio entirely.

**Why this one.** It is the only mechanism any candidate offers for the single decision L1 escalated as
above its level (`L1:193`) and which every dollar in the document depends on. It also repairs B's
weakest spot directly: B currently prints "rating **B**" off an *(assumed)* 3 min/line denominator that
L1 says is certainly wrong, and its own mitigation ("ship as `rating` only") points the wrong way.
Costs one JSONL writer.

**Where in B.** (i) `B:123-134` architecture table — add `calibration.jsonl` as a third artifact
emitted by `skillaudit scan`, with the boundary "records observed minutes; never estimates them";
(ii) `B:164-175` §4c refusal table — add "No letter rating or debt ratio before n ≥ 20 observed fixes
per rule in play"; (iii) `B:37,86` — demote the walkthrough's `9.7% → rating B` / `2.2% → rating A`
cells to **minutes-only** with the ratio shown as pending calibration; (iv) `B:197-203` — the ledger
writer rides along in PR1 (it is a file append, not a new dependency), which strengthens PR1's
"proves" claim rather than adding a PR.

### From **A**: the two internal OTel renames, R1 and R2 (`A:186-188`)

**What.** `TokenCounts.CacheCreation` → `CacheWrite` (JSON `cache_creation` → `cache_write`), and add
`ToolCallEntry.ErrorType string` beside the existing `Success` boolean — because semantic conventions
model failure as the *presence* of `error.type`, not as a boolean, and Cursor already hands us
`failure_type` and `error_message` on `postToolUseFailure` that `success: false` throws away.

**Why this one.** A is the only candidate to surface the OTel report's own bottom line — "**Only R1 and
R2 (internal) are worth doing regardless**" (`otel:276`) — and both are independently grounded:
upstream breaking change #440 renamed it, **and Cursor's own Admin API already spells it
`cacheWriteTokens`** (`otel:212`), so two of three surfaces agree with semconv. B's OTel row currently
says only "no `gen_ai.*` literal in `types.go`", which is correct but drops the two changes that pay
off whether or not `gen_ai.skill.*` ever lands — and both are additive under `spec.md:38`'s
additive-fields-only rule, so they are nearly free today and expensive after `profile/v1` has users.

**Where in B.** (i) `B:123-134` architecture table — extend the `skill-architect/profiler` and
`cursor-profiler` rows with the two renames and the note that `profile/v1` stays additive-only;
(ii) `B:197-203` PR2 ("profiler honesty repairs") — fold R1 and R2 in with D-1/D-2/D-3, since PR2
already opens `types.go`/`compare.go` and this avoids a second breaking touch of the same file;
(iii) `B:226` risk row — keep B's gated-export stance and add "R1/R2 are internal and do not wait on
#498".

*(Runner-up ideas not taken, recorded so they aren't lost: C's "Default if you say nothing" column on
the decisions table (`C:318-327`) — strictly better than B's bare decision list and nearly free to
adopt; A's `E` ≥ 1.0 h floor against sliver-gaming of a ratio metric (`A:94`) — B already has a
stronger sibling in its absolute $1,000 recoverable floor, but the two guard different attacks and
both are cheap.)*

---

## 5. Defects the merged document must fix regardless of base

1. **The OTel path claim is now stale and must be deleted.**
   `.scuba/teams/otel-genai-alignment/report.md` **exists** — verified 2026-09-11, a symlink to
   `../../../docs/research/otel-genai-alignment.md`, alongside `L1/L2/L3` and `arena.md` which are
   symlinked the same way. All three candidates (`A:8-9`, `B:6`, `C:12-14`) and `L3:173` assert it does
   not. It was true when the lens reports were written and is false now. Cite the arena's own path.
2. **The "$0.1025 naive" figure is mislabelled in all three, inherited verbatim from `L2:182`.**
   18,500 input tokens at $5/MTok = **$0.0925**, not $0.1025; the extra $0.0100 is the 400 output tokens
   at $25/MTok. The 3.3× ratio is only correct for "price *every* class at base/naive", not for "price
   all 18,500 *input* tokens at base rate" as `A:119`, `B:74` and `C:115` all word it. Re-label, or
   re-state as $0.0925 and a 3.0× overstatement. This is the arena's headline pedagogical number.
3. **`SK-M001` compares tokens to characters.** Inherited from `L1:180` ("`always_loaded_tokens` over
   the **1,536-char** listing cap"). A materialises it into an impossible finding (`A:45`: 1,430 tokens
   "over" a 1,536-char cap — wrong unit *and* below the threshold); C restates it unitless (`C:58`).
   Pick one unit for the rule, its threshold and its finding message.
4. **Decide once whether a letter rating / debt ratio may ship before calibration, and obey it
   everywhere.** A refuses it outright (`A:229`); B prints B and A ratings off an assumed 3 min/line
   (`B:37,86`); C forbids it in F4 (`C:250`) and prints D and C anyway (`C:64,145`). Recommended
   resolution: A's refusal as the rule, C's ledger as the path to earning it, minutes shipped meanwhile.
5. **`compare.go` line number: the laundering is at `:115`, not `:113`.** Verified in the working tree —
   `115: Source: candidate.Source,`; `:113` is the enclosing `return MetricComparison{`. `C:247,286` is
   wrong and, worse, claims independent verification at that line. A and B are correct.
6. **State the ratio pathology and its guard once, in the walkthrough.** SDY *rises* after a successful
   fix because `E` shrinks — only B names it (`B:91-93`) and supplies an absolute recoverable-dollar
   floor; C's walkthrough has `E` *growing* (1.92 h → 2.33 h) so the pathology never surfaces; A guards
   only the sliver-splitting variant. The merged doc needs the pathology named and at least one floor.
7. **Fix the `c` rule so one derivation governs every table.** `L3:66` says `otel` = 1.0; C caps at 0.8
   without billed data (`C:301`, well argued); A freezes at 0.5; **B contradicts itself** — `B:215`
   states "tier 2 = 0.8, tier 4 = 0.5" while `B:46,54` mark `C_a` tier 2 and assign `c` = 0.5. `c` is
   the honesty multiplier; an unstated or self-contradicting rule there discredits the whole ranking.
8. **Placeholder discipline when the walkthrough names a real skill.** C runs on the real
   `skill-architect/skills/skill-audit` and mixes two genuine `[O]` measurements (1,814 / 4,030 / 55%,
   `L1:65`) with four unmarked findings — notably "61% keyword overlap with `skill-rewrite`" (no source)
   and "2 script paths … **never resolve**" (`L1:114` says only that `$`-paths escape validation).
   Either use a fictional skill (A and B do) or mark every cell. A team lead will forward this document.
9. **Risk$ / Waste$ separation must be structural.** All three find Risk$ dominating (A: 87×, C: 48×)
   off two *declared* constants (`F`, `C_f`). Carry A's hard rule — the tool **refuses to sum them into
   a headline** (`A:133-136`) — plus C's sensitivity column (`C:310`), not just a cautionary sentence.
10. **Propagate the post-refactor `C_a` consistently.** `A:143` computes post-refactor Waste$ ($265)
    from the pre-refactor `C_a` $0.34; its own post `C_a` $0.3089 gives $241. `B:86`'s post ratio 2.2%
    requires an unstated drop in line count (35 ÷ (3 × 620) = 1.9%). State the post-refactor line count
    and reuse the post-refactor `C_a` in every downstream cell.
11. **`C:155`'s rank label is off by one:** SDY 2,221 against its own portfolio (3,410 / 2,880 / 1,220 /
    210) is **#3 of 5**, not #4. The "next best = `skill-rewrite`" conclusion is unaffected.
12. **`A:90`'s `deploy-helper` row must be recomputed if any of A's §2c is carried over:**
    (120 + 8,900) × 0.5 ÷ 1.5 = **3,007**, not 210 — and at 3,007 it ranks *first* on EV, so A's
    accompanying claim that it "ranks last on EV and is fixed first" must be rewritten, not just
    re-numbered. The two-lane demonstration needs a row whose EV genuinely is low.
13. **Keep the explicit disavowal of L3's worked example** (§0 above). All three candidates do this and
    it should survive into the merged document at the point where SDY is introduced, with the corrected
    figures ($373/yr, $11,981/yr, SDY ≈ $2,059) shown so no reader re-derives the old ones.
14. **Length.** All three overshoot the ≤6-page bar (A 4,299 / C 4,189 / B 3,858 words). The merged
    document inherits B's spine plus two fold-ins; budget for cuts, not additions — the walkthrough and
    the measurement contract earn their space, the disagreement tables can compress.
