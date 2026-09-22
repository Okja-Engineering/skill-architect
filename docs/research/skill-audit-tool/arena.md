# Arena: synthesis of the skill-audit-tool research

**Play:** three independent candidates, same mandate, no visibility into each other. A fresh
hunter judges all three against the bar below. Pick a base by soundest shape, fold the single
strongest idea from each runner-up into it. Then publish.

## Inputs (identical for every candidate)
- mandate.md (the user's goal)
- L1-static-report.md, L2-dynamic-report.md, L3-prioritization-report.md
- ../otel-genai-alignment/report.md
- /Users/matthewvandusen/Development/Auraprix/skill-architect (README, PRINCIPLES, profiler/)

## Candidate mandate
Produce ONE markdown document at `candidates/<name>/synthesis.md` that a team lead can share
with an engineering team, answering: what should we build, why, and how do we prove it works.
Required sections, in this order:
1. The pitch in five lines (what, for whom, why now).
2. The workflow: baseline → refactor → prove → next-best-skill, as one concrete walkthrough
   of a single skill with placeholder numbers that flow end to end into a dollar figure and a
   risk/impact score.
3. Architecture: static engine, dynamic engine, data store, scoring, and how they connect.
   Name every reused component (skill-architect, cursor-profiler, DuckDB, OTel) and every new one.
4. The measurement contract: what is billed vs estimated, what "proved" means statistically,
   what the tool refuses to claim.
5. Build options considered and the one recommended, with tradeoffs (SonarQube plugin vs
   SARIF/generic-issue emitter vs harness plugin).
6. Sequenced slices: the first three independently shippable PRs.
7. Risks and open decisions for the user.
Every factual claim traces to a lens report section or a cited URL. No new research: synthesize
what the reports found; where the reports disagree, say so and pick.

## Judging bar (hunter applies this, nothing else)
| # | Criterion | Weight |
|---|---|---|
| B1 | Traceability: every claim traces to a report section or URL; no invented facts | must-pass |
| B2 | The walkthrough's numbers flow end to end without a gap (tokens → $ → risk → ranking) | must-pass |
| B3 | Honesty: estimate vs billed never blurred; "proved" has a stated statistical meaning | must-pass |
| B4 | Soundness of shape: components have clean boundaries; reuses what exists; no gold-plating | high |
| B5 | Readability for a team lead who hasn't seen the research: ≤ 6 pages, tables over prose | high |
| B6 | Sequencing: first three slices are genuinely independent and each proves something | medium |
| B7 | Candor: risks and user decisions are real, not padding | medium |

Hunter output: per-candidate scorecard on B1–B7 with evidence, a ranking, the recommended base,
and for each runner-up the single strongest idea worth folding in.
