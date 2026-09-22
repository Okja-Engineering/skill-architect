# Skill performance and token-efficiency research

Sources:
- ACES — "Evaluating Skills, Not Just Agents: Agentic Continuous Evaluation of Skills" (arXiv 2608.20614)
- SkillReducer — "SkillReducer: Optimizing LLM Agent Skills for Token Efficiency" (arXiv 2603.29919)
- SkillEval — "SkillEval: Decomposing Agent Skill Quality into Interpretable Signals" (arXiv 2608.06891v1) — fetched but not deeply reviewed; treated as **Local hypothesis**.
- agent-profiler — https://github.com/cleverb/agent-profiler — fetched via web search; not cloned; treated as **Local hypothesis**.

## ACES

- A repository-native framework for evaluating skills as executable artifacts.
- Runs paired live trials **with and without** a target skill.
- Normalizes trajectories into ATIF (Agent Trajectory Interchange Format).
- Grades six default runtime metrics and reports **Skill Lift**: the target skill's added value.
- Key result on 947 scored paired cases: mean composite Skill Lift = 0.2134; positive in 72.8% of cases.
- Largest gains are in **skill execution**, **behavior check**, and **skill efficiency** — signals document scans cannot observe.
- These are **ACES process-metric labels**, not a general taxonomy. Do not conflate them with SkillEvaluator's five published dimensions.
- Implication: static audits are Tier 1; runtime paired comparison is the only way to measure skill value and token efficiency.

## SkillEvaluator (NVIDIA) dimensions

- Published five dimensions: **Correctness, Discoverability, Effectiveness, Efficiency, Security**.
- **Efficiency** in that framework is a runtime/economic dimension; it is not the same as ACES's "skill efficiency" process metric.
- Implication: if the audit or report uses "efficiency" as a labeled dimension, say which framework it comes from.

## SkillReducer

- A two-stage optimization framework for skills.
- Stage 1: compress verbose descriptions and generate missing ones via adversarial delta debugging.
- Stage 2: restructure skill bodies through taxonomy-driven classification and progressive disclosure.
- Results on 600 skills:
  - 48% description compression.
  - 39% body compression.
  - Functional quality **improved** 2.8% (less-is-more: removing non-essential content reduces distraction).
  - Benefits transfer across five models from four families (retention 0.965).
- Implication: **static compression** comes from compressing/removing skill content; this is not a runtime token-savings claim. The paper's abstract reports compression ratios, not runtime token or cost reduction. A baseline comparison can only measure runtime token usage with and without the skill active.

## SkillEval

- Document-level skill evaluation with interpretable signals.
- Learns directions in hidden representation space from controlled positive/negative skill pairs.
- Scores correlate with downstream task performance.
- Implication: there are measurable, interpretable quality dimensions beyond pass/fail; our audit should produce dimensioned scores, not just a pass/fail.

## agent-profiler

- Local-first tool that records Cursor/Codex/Claude/OpenCode telemetry into SQLite.
- Provides:
  - Estimated input/output/tool/shell tokens with bar proportions.
  - Efficiency score heuristic.
  - Session timeline.
  - Tool result-size histograms.
  - Context audit for always-on instruction paths.
- No remote telemetry.
- Implication: a file-based, privacy-preserving local store is a proven pattern; SQLite or JSONL both work. The value is in the *analysis*, not the transport.

## Implications for cursor-profiler

- The core deliverable is **paired runtime comparison** (with-skill vs without-skill), not a single-session profile.
- Token efficiency must be measured as the delta when the skill is active vs inactive.
- Chars/4 estimates can be a valid **relative** signal for A/B comparison if honestly labeled, but they are not billing numbers.
- `context_tokens` from `preCompact` is a real, Cursor-reported proxy for memory pressure.
- Static compression (SkillReducer) is a separate optimization axis; runtime profiling validates whether compression actually helps.
- The evaluation report should include Skill-Lift-style **process metrics** (skill execution, behavior check, skill efficiency) as **signals**, not a dimension taxonomy. Use SkillEvaluator's **Efficiency** dimension for a labeled, interpretable score.
