# Mandate: "SonarQube for Agent Skills" — static + dynamic audit tool (research only)

**Dispatched:** 2026-09-11 · **Mode:** research only. No code. No edits to any repo file
except your own report under this directory. Explore outside this repo freely (web, GitHub,
the local skill-architect checkout), bring findings back here.

## The user's goal, verbatim intent
Build a best-in-class audit tool for AI-native software developers that does **static and
dynamic** analysis of Agent Skills (SKILL.md packages and their bundled scripts). The core
workflow: run it against one specific skill, collect a baseline, refactor the skill, run it
again, and **prove the refactor made a difference** — a difference expressible as a dollar
amount, and as risk, impact, and value — then **identify the next best skill to improve**.
SonarQube is the mental model (rules, quality gates, complexity, circular references,
technical-debt cost). The output of this research is a document the user can share with their team.

## What already exists (read first, do not re-research)
- /Users/matthewvandusen/Development/Auraprix/skill-architect — the in-house plugin:
  README.md, PRINCIPLES.md, skills/skill-audit/SKILL.md (static checks today),
  skills/skill-rewrite/SKILL.md, docs/profiler-spec.md, profiler/*.go (dynamic profiler,
  compare.go = before/after comparison, experiment.go = paired experiment design).
- /Users/matthewvandusen/Development/Auraprix/cursor-profiler/docs/research/
  skill-profiling-state-of-the-art.md, skill-evaluation-research.md,
  existing-per-skill-cost-tools.md, cursor-telemetry-surfaces.md.
- /Users/matthewvandusen/Development/Auraprix/cursor-profiler/.scuba/teams/otel-genai-alignment/report.md
  (OTel GenAI naming review, done today).

## Lenses (one researcher each)

### L1 — Static analysis: the SonarQube model applied to skills
Deliverable: `.scuba/teams/skill-audit-tool/L1-static-report.md`
1. SonarQube's model, precisely: rules, quality profiles, quality gates, "new code" focus,
   SQALE / technical-debt remediation-cost model (how issues become minutes then money),
   security hotspots, duplication, cognitive vs cyclomatic complexity. Cite SonarSource docs.
2. Map each concept to skill packages: what is "complexity" for an instruction file (candidates:
   token weight, instruction count, branching/conditional instructions, nesting of references,
   progressive-disclosure depth); "circular reference" (skill→skill, skill→script→skill,
   `[[links]]`, slash-command chains); dead references (scripts/files named but absent);
   duplication across skills; spec conformance; security hotspots in bundled scripts
   (shell, network, credentials); description-trigger quality.
3. Landscape: every existing static tool for skills/prompts you can verify — Anthropic
   `/skill-doctor` and `claude plugin eval`, skill-validator, skill-architect's skill-audit,
   promptfoo, agentlint-style linters, markdown/prompt linters, any SonarQube community plugin
   for prompts or markdown. For each: what it measures, how, license, maturity, URL.
4. Build options with tradeoffs: (a) a real SonarQube language plugin / custom-rules plugin
   (what the plugin API requires; would skills be a "language"?), (b) standalone rule engine
   emitting SARIF or Sonar generic-issue JSON so SonarQube ingests it, (c) stay a Claude Code /
   Cursor plugin. Recommend one.

### L2 — Dynamic analysis: proving a refactor made a difference
Deliverable: `.scuba/teams/skill-audit-tool/L2-dynamic-report.md`
1. Paired evaluation design for one skill: baseline vs refactored, same task set, N repetitions,
   what to hold constant (model, harness, seed where possible), what varies. What the field
   does (build on skill-profiling-state-of-the-art.md §1, do not repeat it).
2. Metrics that can move: tokens by type, cost, latency, tool-call count and success rate,
   activation precision/recall (did the skill trigger when it should, not when it shouldn't),
   task success / eval pass rate, context-window pressure, compaction events.
3. Statistics for "proved": variance of LLM runs, paired tests vs bootstrap CIs, minimum
   detectable effect vs N, how to report honestly when tokens are estimated not billed.
4. Converting a measured delta to dollars: price per token by model × delta × activation
   frequency × users × period; where each factor comes from (billing APIs, hook spool, OTel);
   what is estimate vs billed; a worked example with placeholder numbers.
5. Gap analysis against skill-architect's experiment.go / compare.go: what they already do,
   what's missing to run the full before/after loop, and the known defect that compareTokens
   launders estimate sources.

### L3 — Risk, impact, value, and "next best skill to improve"
Deliverable: `.scuba/teams/skill-audit-tool/L3-prioritization-report.md`
1. A scoring model: for each skill, expected value of improving it =
   activation frequency × cost per activation × achievable reduction, plus risk terms
   (failure rate, blast radius of bundled scripts, permission scope, silent-failure modes)
   and impact terms (tasks depending on it, users depending on it). Compare against
   SonarQube's technical-debt ratio / hotspot ranking, WSJF, RICE. Recommend one and show the
   formula and the inputs each term needs and where each input comes from.
2. Risk taxonomy for skills specifically (what can go wrong, how you'd detect it statically vs
   dynamically).
3. What "best in class" looks like end to end for an AI-native dev team: personas, the
   before/refactor/after workflow, a quality gate in CI, a portfolio view ranking skills,
   trend over time. Cite any product that does a piece of this well (SonarQube, Braintrust,
   LangSmith, Datadog LLM Observability, etc.) and what to borrow.
4. Anything in L1/L2 scope you notice that they might miss, as a short handoff note.

## Definition of done (all lenses)
- Every claim cites a URL or a repo path + line, marked **[O]** observed (you read it) or
  **[I]** inferred.
- Tables over prose. ≤ 4 pages each. A short summary at the top: the answer in five lines.
- End with "Decisions needed above my level" and "Handoff to synthesis" sections.
- No code written. No files modified except your report.

## Quality bar
Precision over breadth. A verified fact beats three plausible ones. Where sources disagree,
say so and cite both. Say "no evidence found" rather than guessing.
