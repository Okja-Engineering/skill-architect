# Arena scorecard — skill-audit-tool synthesis

**Hunter:** fresh judge, no authorship stake. **Bar applied:** B1–B7 from `arena.md` only.
**Published result:** `synthesis.md` (base: **alpha**, one fold-in each from beta and gamma — see §5).

## 0. Arithmetic audit performed first

L3 §1.3's printed worked example does not reconcile with its own inputs — inputs A=12, U=8,
C_a=$0.34, r=0.22 imply `Waste$` = 4,992 × 0.34 × 0.22 = **$373/yr** (L3 prints "$9.0/wk ≈
$470/yr"), and F=0.06, B=1, C_f=$40 imply `Risk$` = 4,992 × 0.06 × 40 = **$11,981/yr** (L3
prints "$920/yr"). Per the hunter note, candidates that recomputed with consistent placeholders
are not penalized; candidates that copied broken arithmetic are.

| Walkthrough | Tokens → $ | Annualization | SDY | Verdict |
|---|---|---|---|---|
| alpha | −(4000·5+60000·0.5+1600·6.25+2000·25)/10⁶ = **−$0.11** ✓ | 0.11×10×8×52 = **$458**, CI [$250,$666] scales by CI bounds ✓ | (458+1248)×1.0÷3.2 = **$533/h** ✓ | fully verified, internally consistent |
| beta | −22% median (ordinal on Cursor — no $ rendered, correct) | 4,992×0.34×0.22 = **$373**, CI [$136,$577] propagated from r's CI ✓ | (373+11,980)×0.5÷2.7 = **$2,290/h** ✓ | verified; explicitly flags L3's non-reconciliation |
| gamma | L2 §4.3's table verbatim: **−$0.0311** ✓ arithmetic | 4,992×0.0311 = **$155** ✓; CI [$85,$225] ✓; naive $512 = 3.3× ✓ | (155+1,500)×0.8÷2.9 = **$456** ✓ arithmetic | arithmetic fine; **provenance broken** (see B2/B3) |

## 1. Alpha — scores

| Bar | Score | Evidence |
|---|---|---|
| B1 traceability | **PASS** | Every section carries section-level cites (`L1 §2b`, `L2 §3.3 tier 2`, `Q12`, `OTel §4`). Two minor stretches: pitch line 4 attributes "proven before/after deltas" emptiness to "L1 §3 verdict" — that verdict lists remediation cost/cycles/new-code/portfolio; the v1-vs-v2 gap is L2 §1. Slice-2's "a dead-ref and a cycle ... that no existing tool detects" over-bundles — PT001 detects dead refs today (though it silently skips `$var` paths, L1 §2d). Both fixed in the published doc. No invented facts found. |
| B2 numbers flow | **PASS** | All arithmetic verified (table above). End-to-end chain unbroken: static debt 190 min → E=3.2h; telemetry A·U·C_a → $2,080/yr run-rate; paired Δ → −$0.11/act with CI; → Waste$ $458 + Risk$ $1,248 → SDY $533/h → two-lane ranking. Bonus: a Cursor-variant callout shows the same pipeline degrading honestly to ordinals. |
| B3 honesty | **PASS** | Runs the dollar path on Claude Code (tier 2 → "estimated" is legal); the Cursor note renders "no dollar figure." Tiers table matches spec verbatim. "Proved" is a stated conjunction: BCa CI excludes 0 AND precision/recall/verifier in tolerance AND lift vs arm A ≥ 0, "reported as an interval plus the achieved MDE." Refusals list includes "a token delta without the activation 2×2" and unguarded stopping. |
| B4 shape | **Strong** | Component table marks every item new/reused/repair; profile-config and hotspot-baseline files are owned explicitly because "imported rules can't be managed inside SonarQube (L1 §4 named tradeoff)." OTel is "reused two ways" — input attribution vs gated export. No gold-plating. |
| B5 readability | **Strong** | ~165 lines; every section is tables-first; a team lead can read §2 alone and get the whole loop. Tightest of the three. |
| B6 sequencing | **Strong** | Three PRs, disjoint codepaths (profiler fixes / new binary / SQL over spool), each with a "proves" column; correctly notes Q12 already ordered comparator fixes first. |
| B7 candor | **Strong** | 12 sourced decisions with stakes; a "thin evidence — do not design on it" paragraph (U-02, `skill.name` listing gap, `sqale_index` roll-up, ClawHavoc 1,184-vs-341, S-8); and an explicit "judgment calls where inputs pull apart" section, including catching the arena's stale `../otel-genai-alignment/report.md` path. |

## 2. Beta — scores

| Bar | Score | Evidence |
|---|---|---|
| B1 traceability | **PASS** | Section-level cites throughout; adds an "Evidence" column to its reuse table. "chars/4 × price table — **estimated (tier 4→3 label)**" is idiosyncratic phrasing, not a factual invention. |
| B2 numbers flow | **PASS** | All arithmetic verified, and it is the only candidate to name the trap in-line: *"L3 §1.3's printed example ($9.0/wk Waste$, $920/yr Risk$) does not reconcile with its own formula applied to its stated inputs; all figures above are recomputed from the formula."* Two nits: the portfolio column "est. $/wk" is ambiguous (it is (Waste$+Risk$)/52, not gross spend); and *"the hotspot lane ordered it ahead of the higher-SDY row"* is muddled — `deploy-helper` at $3,100/h is itself the highest SDY. |
| B3 honesty | **PASS — best of the three** | Only candidate to explicitly resolve the real L2-vs-L3 contradiction: *"L2 §3.3 forbids dollar rendering on tier-4 deltas; L3 §1.3 publishes estimated portfolio dollars. Resolution: per-run comparison reports are strict (ordinal); the portfolio may carry labeled `estimated` dollars because the confidence term `c` discounts them mechanically... Estimates may feed ranking; they may never be presented as spend."* Its walkthrough demonstrates the contract working: on Cursor the per-run line is "−22% median, BCa 95% CI [−34%, −8%] ... **relative estimate, not billed** (tier 4)" with "No dollar figure appears on this line." |
| B4 shape | **Strong** | Clean two-engine diagram over a JSONL-spool/DuckDB store; reuse table lists `skill-validator / SkillSpector / jscpd / agnix` as "do not rebuild"; "Everything else is borrowed." |
| B5 readability | **Good** | ~245 lines — longest of the three but still ≤ 6 pages, tables over prose, and the pitch states "the catch" up front (line 5). |
| B6 sequencing | **Strong** | Adds an explicit "Depends on: none" column per slice; "Deliberately later" paragraph justifies exclusions via cheap-before-expensive (L3 §3.2). |
| B7 candor | **Strong** | 11 decisions + 6 named risks; leads the pitch with the team's tier-4 reality rather than burying it. |

## 3. Gamma — scores

| Bar | Score | Evidence |
|---|---|---|
| B1 traceability | **PASS (flagged)** | Dense bracket cites throughout. But the step-3 per-class Δtoken table (base input / cache read / cache write / output) is presented under *"Environment: Cursor hooks"* — chars/4 over hook payloads cannot decompose into token classes (spec tier 4; L2 §3.3). The numbers are placeholders so it isn't an invented *fact*, but it implies a capability no cited source supports in that environment. Also *"precision held at 9/10 near-misses"* mislabels the TN rate as precision (L2 §2.1 defines precision = TP/(TP+FP)). |
| B2 numbers flow | **Marginal PASS** | Arithmetic verified end to end (table above). The gap is provenance, not math: on the stated harness the token-delta row cannot exist as measured data, so tokens→$ has an unbridgeable step the other two candidates handle explicitly. |
| B3 honesty | **FAIL** | Its own contract refuses "any dollar figure when either side is tier 4" (§4c) — yet the walkthrough, declared "Cursor hooks, no admin key" (tier 4 only), prints "**$155/yr, estimated, 95% CI [$85, $225]**" and "SDY ≈ $456." "(tier 2–4)" in the step header lumps tiers that have different rules. Alpha avoids this by running dollars on Claude Code; beta avoids it by the explicit ordinal-vs-portfolio resolution. Gamma does neither and contradicts itself. Must-pass bar — this is disqualifying for base selection. |
| B4 shape | **Strong** | Best single structural idea of the three: an "**Explicitly not built**" list — "a SonarQube Java plugin, a daemon, an OTLP exporter (parked... any future export uses our own `cursor_profiler.*` namespace, gated `[OTel §4b, §5]`)." Verified against OTel R14–R21. |
| B5 readability | **Strong** | ~220 lines, tables over prose, clear step headings; the L2 worked example is reused cleanly (just in the wrong harness). |
| B6 sequencing | **Strong** | Same sensible three slices; slice 3 adds that it is "the first real test of the untested U-02 activation heuristic" — a nice honesty hook. |
| B7 candor | **Strong** | 12 decisions incl. the PL*/PT*→SK-* namespace break (L1 dec. 5, which alpha folds into decision 6); 7-row thin-evidence/risk table with severities. |

## 4. Ranking

1. **Alpha** — all must-passes green with margin; tightest document; the only walkthrough that is
   legal-by-construction on both harnesses (dollars on Claude Code, labelled ordinals on Cursor).
2. **Beta** — all must-passes green; the single most sophisticated honesty move (the explicit
   L2-vs-L3 resolution) and the cleanest handling of L3's broken example; loses to alpha on
   concision and on two small internal ambiguities.
3. **Gamma** — strong B4–B7, but B3 fails: its walkthrough prints dollar figures on a tier-4-only
   harness in direct violation of the measurement contract it itself writes (§4c), and its
   per-class token table is unobtainable from the stated Cursor-hooks source.

## 5. Base and fold-ins

**Base: alpha** — soundest shape (B4/B5), all must-passes clean, and the only candidate whose
walkthrough covers both harnesses without breaking its own contract.

Folded in:

- **From beta:** the explicit *resolution of the L2 §3.3 vs L3 §1.3 disagreement* — per-run
  comparisons strict/ordinal on tier 4, portfolio may carry labelled `estimated` dollars
  discounted mechanically by `c`, never presented as spend. Published as the "Where the reports
  disagree — resolved" paragraph in §4, and the Cursor callout in §2 was reworded to match.
  Supporting artefact also adopted: the in-line **reconciliation note** that L3 §1.3's printed
  numbers don't reconcile and that all figures are recomputed (§2 step 3 footnote).
- **From gamma:** the "**Explicitly not built**" list — no SonarQube Java plugin, no daemon, no
  OTLP exporter, own-`cursor_profiler.*`-namespace gated export. Added verbatim in spirit to §3.
  (Also adopted gamma's slice-3 framing that `analyze` is "the first real test of the untested
  U-02 heuristic.")

Editorial fixes applied to the base while publishing (alpha's two minor citation stretches):
pitch line 4 now cites `L1 §3 verdict; L2 §1` separately for the static gap vs the missing
v1-vs-v2 proof; slice 2's "proves" now distinguishes cycles (detected by no existing tool,
L1 §2c) from dead refs today's checks silently skip (L1 §2d).
