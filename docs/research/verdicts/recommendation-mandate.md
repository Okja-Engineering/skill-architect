# Mandate: the final recommendation — a state-of-the-art skill audit / doctor / inspector (arena)

**Dispatched:** 2026-09-11 · **Mode:** synthesis and recommendation. No new research beyond
verifying a cited line. No code. Write exactly one file.

## The user's intent (recorded 2026-09-11, their words paraphrased closely)
"We just really need to make a kick-ass, state-of-the-art skill audit / skill doctor / skill
inspector — safety, all the things you'd be mindful of in an enterprise situation. Especially
if I'm bringing in skills of my own, or skills I might grab off the Internet, I want to run
something against them and make sure that as we move faster I don't miss something. Safety.
Third-party skills." Plugin targets Claude Code and Cursor primarily. The earlier goals still
stand: quantitative self-understanding of AI usage and spend; before/after proof that a
refactor paid off in dollars, risk, impact, value; identify the next best skill to improve.

## Inputs (all under /Users/matthewvandusen/Development/Auraprix/cursor-profiler/)
- docs/research/skill-audit-tool/synthesis.md — the arena-merged "SonarQube for skills" design
  (and its lens reports L1/L2/L3 in the same directory for detail)
- docs/research/warp-skill-doctor.md — arena-merged finding on Warp's skill-doctor
- docs/research/skillspector.md — arena-merged finding on NVIDIA SkillSpector + evaluation of
  skill-architect's spec/intent/docs (the security lens; read this most carefully)
- docs/research/otel-genai-alignment.md — naming/export alignment
- docs/research/skill-profiling-state-of-the-art.md, existing-per-skill-cost-tools.md
- intent.md, spec.md, questions.md (cursor-profiler), and
  /Users/matthewvandusen/Development/Auraprix/skill-architect/{README,PRINCIPLES}.md,
  skills/*/SKILL.md, docs/profiler-spec.md, .scuba/roadmap.md

## Deliverable: `candidates/<X>/recommendation.md` — a team-shareable document
1. **The pitch** (five lines): what we build, for whom, why it's state of the art, why now.
2. **The threat and quality model for a skill entering an enterprise**: what can go wrong
   when a first-party or third-party skill is installed and runs (prompt injection, data
   exfiltration, supply chain, over-broad tools/permissions, silent failure, negative lift,
   context bloat, cost). One table: threat/defect → how detected (static rule / LLM-assisted /
   dynamic-runtime) → which existing tool covers it today (SkillSpector, skill-audit,
   agnix, Warp, Anthropic skill-doctor) → what we add.
3. **The gate**: the "run this against a skill before I trust it" workflow, end to end,
   for (a) a skill I wrote, (b) a skill I pulled from the Internet. Inputs, stages, outputs,
   what blocks vs warns, how fast it must be, where it runs (pre-install, pre-commit, CI,
   pre-activation in the harness). Claude Code and Cursor specifics called out separately.
4. **Architecture**: static engine, security scanning (integrate SkillSpector or reimplement —
   decide, with licensing), LLM-assisted checks and how they stay honest, dynamic profiling
   (cursor-profiler / Claude Code OTel), data store, scoring (debt, risk, yield), reporting
   (native report, SARIF/Sonar), packaging as a Claude Code + Cursor plugin. Every reused
   component named; every new one justified.
5. **Measurement contract**: billed vs estimated, what "proved" means, what the tool refuses
   to claim. Reuse synthesis.md's contract; do not re-derive.
6. **Roadmap**: sequenced, independently-shippable slices; the first three PRs; what is
   safety-critical and ships first; what waits on a Cursor admin key or Enterprise OTel.
7. **What we deliberately don't build**, and why.
8. **Risks and decisions for the user.**
Every claim traces to one of the inputs (path + section) or a URL. Where inputs disagree,
say so and pick. ≤ 7 pages, tables over prose.

## Addendum (user, 2026-09-12): non-negotiable emphases
1. **Progressive disclosure is the house methodology.** The project's Interpretable Context
   Methodology (ICM) is a repackaging of progressive disclosure: skills are built as files and
   folders that load incrementally, never one long SKILL.md. skill-architect's PRINCIPLES.md
   already codifies ICM criteria; the recommendation must treat ICM conformance as a
   first-class static dimension, and the tool itself must be built that way.
2. **Static analysis must be excellent, not adequate.** Beyond spec conformance and security,
   the static engine must know the *environment*: which frontmatter fields are valid, invalid,
   or silently ignored in Claude Code vs Cursor (per harness version), and report per-harness
   validity. It must compute **project-level always-on token cost** — everything a harness
   loads into every session (CLAUDE.md / AGENTS.md / rules, every skill's description line,
   hooks, MCP tool manifests) — as a budget with per-item attribution, distinct from
   per-activation cost. Enumerate the full static catalogue the tool should ship, drawing on
   L1, agnix, SkillSpector, Warp, and Anthropic's skill-doctor, and mark which are new.
3. **Dynamic analysis rides on OTel.** Claude Code and Cursor are the targets precisely
   because they expose telemetry (Claude Code OTel with `skill.name`; Cursor hooks + Enterprise
   OTel / Admin API). The dynamic engine must consume those surfaces; do not design around
   surfaces the targets lack.
4. **Safety for third-party skills remains first.** Item 1–3 do not displace the gate.

## Final deliverable (user, 2026-09-12)
After the merged recommendation is published, produce a **paste-able handoff prompt for Devin**
(the agent that authored intent.md / spec.md / questions.md and builds in skill-architect):
new research has landed (list paths), these are the recommendations (link the page + the merged
docs), this is what changed in the plan, evaluate the current state of cursor-profiler and
skill-architect against it, advise where to go next, then start building again.

## Closing sequence (user, 2026-09-12, after the pi workflow lands)
The durable unit is the skill-architect plugin. Devin cold-starts there with no memory.
1. Integration path: fold docs/research/pi.md into recommendation.md only where it changes a decision;
   record the delta in a short "pi integration" note rather than rewriting.
2. Copy docs/research/** (including pi.md and the arena verdicts that matter) into
   /Users/matthewvandusen/Development/Auraprix/skill-architect/docs/research/ as untracked files,
   overwriting nothing, so the plugin repo carries its own rationale.
3. Rewrite the Devin handoff as a cold-start prompt rooted in skill-architect: self-contained,
   every path inside skill-architect, provenance explained, plan deltas, user decisions with
   defaults, then evaluate → advise → build slice one (the safety gate) as small PRs, never merging main.
