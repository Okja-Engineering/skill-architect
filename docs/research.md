# Skill evaluation research synthesis

This document summarizes the public state of the art for evaluating and improving Agent Skills, as of 2026-09-07. It informed the design of the `skill-audit` skill in this plugin.

## Core sources

- Anthropic — Agent Skills announcement and best practices
- Agent Skills open specification (agentskills.io)
- Microsoft — Agent Skills on Azure / Agent Framework
- NVIDIA SkillEvaluator
- SkillsBench benchmark paper (arXiv:2602.12670)
- SoK: Agentic Skills — Beyond Tool Use in LLM Agents (arXiv:2602.20867)
- Interpretable Context Methodology (arXiv:2603.16021)
- Progressive disclosure study (arXiv:2607.17598)
- Spotify — AiKA, Xirp, Backstage Skill Exchange

## Convergent principles

### 1. A skill is a directory with progressive disclosure layers

Every major source agrees on the same shape:

```text
skill-name/
├── SKILL.md          # frontmatter + body
├── scripts/          # optional executable helpers
├── references/       # optional deep docs
└── assets/           # optional templates/resources
```

The frontmatter (`name`, `description`) is the only part always loaded. The body loads when the agent decides the skill applies. Bundled files load only when referenced.

### 2. The description is the trigger

Anthropic, Microsoft, and the spec all emphasize that `description` is the primary triggering mechanism. A weak description means the rest of the skill is invisible.

Strong descriptions include:

- What the skill does.
- When to use it.
- Specific keywords and alternative phrasings.
- Negative scope when the edge is unclear.
- A slightly "pushy" tone, because skills tend to undertrigger.

### 3. Mechanical work belongs in scripts

Scripts provide determinism that LLM reasoning cannot. They should have error handling, avoid hardcoded absolute paths, and be referenced exactly from the skill instructions.

### 4. Evaluation is tiered

NVIDIA's SkillEvaluator is the clearest public articulation:

| Tier | Question | Cost |
|---|---|---|
| Tier 1 — Validation | Is it safe and well-formed? | Cheap |
| Tier 2 — Deduplication | Does it overlap with existing skills? | Medium |
| Tier 3 — Live evaluation | Does it actually help the agent? | Expensive |

SkillsBench confirms that paired evaluation (with-skill vs without-skill) is the right foundation for measuring live efficacy, and that focused, smaller skill sets outperform exhaustive bundles.

### 5. Context management is a first-class concern

ICM and the progressive-disclosure research both argue that folder structure and load order are architecture. Lessons:

- Keep always-loaded catalogs small.
- Do not inline large reference payloads into the main contract.
- One deeper routing level helps with large corpora; more than one hurts.
- Separate stable reference material from per-run working artifacts.

## Implications for this plugin

The `skill-audit` skill focuses on Tier 1 plus lightweight Tier 2 guidance. It deliberately does not attempt Tier 3 live evaluation, which requires a sandbox, a task set, and significant compute. The audit report is a human-in-the-loop artifact: it reports, scores, and recommends, but does not silently rewrite.

The 10-dimension scoring matrix used by `skill-audit` maps onto these industry findings:

- **Spec compliance, Trigger, Body structure, Determinism, Self-contained, Token discipline** are Tier 1 checks.
- **Scope, Context management, Validation, Examples** bridge Tier 1 and Tier 2: some have deterministic signals, others require judgment.
- **Live efficacy** is Tier 3 and is out of scope for v0.1.0.

## Open questions for future versions

- Should the plugin grow a Tier 2 overlap check that compares a new skill against an existing skill catalog?
- Should it support a lightweight Tier 3 harness for skills with objectively verifiable outputs?
- Should it integrate with the Repository Learning Protocol's capture/triage flow to improve its own rules over time?
