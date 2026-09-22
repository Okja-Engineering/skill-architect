# `docs/research/` — index and provenance

The research behind the `skillgate` program: what to build, why, and what the tool may never claim. Every file here was copied into this repo on **2026-09-12** from the `cursor-profiler` scratch repo, where the research was run. **This is now the canonical durable copy** — that repo has no commits and is not the unit we ship.

The directory is untracked but **not** gitignored; committing it is part of the next PR. Start with `HANDOFF.md`, then `recommendation.md`.

## The plan

| File | What it is | How produced | Date |
|---|---|---|---|
| `HANDOFF.md` | Paste-able cold-start prompt for the agent building the plugin: intent, reading order, plan deltas, decisions, first check. | Written by hand from the mandate, the merged recommendation, and a live read of the working tree. | 2026-09-12 |
| `recommendation.md` | **The plan.** `skillgate`: threat/quality model, gate stages G0–G7, architecture, measurement contract and refusals F12–F14, six slices, decisions D1–D9. | Arena (below), base **C**, fold-ins from A and B; then 15 in-line **[pi]** edits from `pi.md`. | 2026-09-12 |
| `pi-integration.md` | Delta log for those 15 edits: where each landed, what changed, and what was deliberately *not* integrated. | Single integration pass over `pi.md` → `recommendation.md`; four load-bearing cites re-verified from primary source. | 2026-09-12 |
| `pi.md` | The pi (earendil-works) finding: manifest trap, Cursor's eight stock skill roots, the Pack E ground-truth bench, the cache-miss trap, decisions P1–P7. | pi workflow (below), base **C**, fold-ins from A and B, plus a gap-fill pass. | 2026-09-12 |

## Merged findings

| File | What it is | How produced | Date |
|---|---|---|---|
| `skillspector.md` | NVIDIA SkillSpector re-enumerated from source: 113 rule IDs, 27 analyzer nodes, 4 dead rules, no Cursor coverage — plus an audit of this repo's own docs. | Arena, base **C**, fold-ins from B and A; the judge re-enumerated the source independently. | 2026-09-12 |
| `warp-skill-doctor.md` | Warp's `skill-doctor` taken apart: what to borrow, and its scoring-curve and `skill_coverage` defects. | Arena, base **B**, fold-ins from A and C. | 2026-09-11 |
| `skill-audit-tool/synthesis.md` | The "SonarQube for Agent Skills" design: measurement contract, refusals F1–F11, scoring. `recommendation.md` §5 reuses it verbatim. | Arena, base **B** per its own footer, fold-ins from C and A. See the conflict note below. | 2026-09-11 |

## Single-lens and single-researcher reports

| File | What it is | How produced | Date |
|---|---|---|---|
| `skill-audit-tool/L1-static-report.md` | Static rules, rule catalogues, tool comparison. | One researcher, one lens. | 2026-09-11 |
| `skill-audit-tool/L2-dynamic-report.md` | Dynamic statistics, activation precision/recall, the verified `compare.go` defects D-1–D-9. | One researcher, one lens. | 2026-09-11 |
| `skill-audit-tool/L3-prioritization-report.md` | SDY prioritization, blast radius, the hotspot queue. | One researcher, one lens. | 2026-09-11 |
| `otel-genai-alignment.md` | Field-by-field `gen_ai.*` semconv review and the never-emit list. | One researcher; not an arena. | 2026-09-11 |
| `existing-per-skill-cost-tools.md` | 13 tools surveyed for per-skill token/cost attribution in Cursor: none exists. | Deep research, 3-vote verified. | 2026-09-10 |
| `cursor-telemetry-surfaces.md` | 21-event hook payload schema, Enterprise OTel wire reference, Admin API, `state.vscdb`, token-source honesty ranking. | From the skill-architect research ledger and spec gate. | 2026-09-10 |
| `skill-profiling-state-of-the-art.md` | SkillsBench, ACES, NVIDIA SkillEvaluator, `/skill-doctor`, skill-validator BPE counts, harness constants. | Same ledger. | 2026-09-10 |
| `cursorscope-requirements.md` | The R-CS / R-SA / R-RL requirement register with keep / adapt / drop tags. | Same ledger. | 2026-09-10 |
| `cursorscope-source-analysis.md` | `last9/cursorscope` (MIT): architecture, hook contract, attribution, what to port vs drop. | Source read, reconciled against the 21-event surface. | 2026-09-10 |
| `go-dependency-and-tooling-decisions.md` | Measured Go OTel SDK cost vs hand-rolled OTLP, semgrep assessment, profiler contract limits. | Same ledger; measurements run locally. | 2026-09-10 |
| `open-unknowns.md` | Unknowns U-01…20 and the experiment that would settle each. | Consolidated from the ledger. | 2026-09-10 |
| `research-contradictions.md` | Contradictions found across the research, plus two profiler-contract findings that need a decision. **Open:** contradiction #6 (Cursor's stock skills system) is still unfiled — see `pi-integration.md`. | Spec-gate round. | 2026-09-10 |
| `skill-evaluation-research.md` | ACES, SkillReducer, SkillEval and `agent-profiler` on evaluation and paired comparison. | Source survey, reconciled. | 2026-09-10 |

## Arena inputs and judgments

| File | What it is | How produced | Date |
|---|---|---|---|
| `skill-audit-tool/mandate.md` · `skill-audit-tool/arena.md` | The lens mandate · the arena play and the B1–B7 judging bar. | Written before any candidate ran. | 2026-09-11 |
| `skill-audit-tool/candidates/{alpha,beta,gamma}/synthesis.md` | Three unmerged candidate syntheses. | Independent, no cross-visibility. | 2026-09-11 |
| `skill-audit-tool/arena-scorecard.md` | Hunter scorecard over that alpha/beta/gamma set, opening with an arithmetic audit; published result recorded as base **alpha**. | Fresh judge, no authorship stake. | 2026-09-11 |
| `verdicts/skill-audit-tool.md`, `verdicts/skillspector.md`, `verdicts/warp-skill-doctor.md`, `verdicts/recommendation.md`, `verdicts/pi.md` | The verdicts of record: scorecards, defect lists, chosen base, merge instructions. Byte-identical copies of the control-plane originals. | Fresh judges who authored no candidate. | 2026-09-11 / 12 |
| `verdicts/recommendation-mandate.md` | The mandate the final recommendation was written to, carrying the user's intent and the four emphases. | Written by the user's session. | 2026-09-11, addendum 09-12 |

**Conflict, unresolved.** `arena-scorecard.md` records the published `synthesis.md` as base **alpha**, while `synthesis.md`'s own footer records base **B** with fold-ins from C and A, citing a verdict over candidates A/B/C (`verdicts/skill-audit-tool.md`). The two candidate sets have different word counts, so these look like two judging rounds rather than two labels for one. Treat `synthesis.md`'s footer as authoritative for that document, and `arena-scorecard.md` as the earlier round.

## How this research was made

**The arena method.** For each question: one mandate, **three independent candidates** written with no visibility into each other; a **fresh hunter that authored none of them** judged all three against a bar stated in advance — must-pass criteria for traceability, honesty and internal arithmetic; weighted criteria for shape, readability, sequencing and candour. The judge picked a **base** (the soundest shape, not the most complete), folded in the single strongest idea from each runner-up, and listed the defects the merged document had to fix. Every published file above names its base and its fold-ins. Verdicts live in `verdicts/`; nothing was published that the verdict's defect list did not close.

**The workflow method used for pi.** `pi.md` came from a scripted multi-agent workflow rather than a hand-run arena: **27 agents, 0 errors** — six map lenses over the pinned pi source (`71dca87`), three independent candidates, three lensed verifiers per candidate (source accuracy, relevance, feasibility), then a **two-judge panel plus a critic**. Both judges ranked C > B > A with no must-pass failures; base **C**, fold-ins from A and B. A later gap-fill pass added Cursor's stock skills system, the tool-schema token class and the install-path analysis, re-fetching every load-bearing cite from primary source. Verifier output was not taken on faith: one relevance verifier's refutation was overruled on the record, and a candour row marked "inferred" was re-verified as observed before the merge.

**Integration discipline.** `pi.md` was folded into `recommendation.md` **only where it changed a decision, a threat row, a gate stage, a refusal, a roadmap slice or a don't-build item** — 15 edits, each tagged **[pi]** with a section cite and logged in `pi-integration.md`. The recommendation was not rewritten.
