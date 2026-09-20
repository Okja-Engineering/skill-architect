# Skill Architect principles

This document explains the ideas behind the `skill-architect` plugin. Read it before adding new skills, changing the evaluation matrix, or interpreting an audit report.

## The problem

Agent Skills are instructions read by an LLM. That makes them fragile in ways normal code is not:

- A weak `description` means the skill never loads, even if the body is perfect.
- A missing heading means the agent skips a step it was supposed to follow.
- A script name mismatch means the agent runs a command that does not exist.
- An inlined wall of text wastes context and drowns the signal.

These failures are usually silent. The agent does not report "I could not find `scripts/foo.sh`"; it just does something else.

Skill Architect makes these failures visible and fixable.

## Core beliefs

1. **Deterministic checks first.** Frontmatter, file layout, headings, scripts, and paths can be checked mechanically. These checks are cheap and catch most real failures.
2. **The description is the trigger.** It is the only part of a skill that is always loaded. If it is vague, the rest of the skill does not exist.
3. **Mechanical work belongs in scripts.** Scripts are reliable; prose instructions are not. A skill should tell the agent *when* to run a script, not recreate the script's logic in markdown.
4. **Progressive disclosure.** Load only what the current step needs. Keep `SKILL.md` small; move detail to `references/` and `assets/`.
5. **Human-in-the-loop for changes.** The plugin reports and recommends. It does not silently rewrite a skill.
6. **Focused skills beat large bundles.** One skill should do one job. If it does not fit cleanly into one category, split it.
7. **Live evaluation is Tier 3.** Measuring whether a skill actually helps an agent is valuable but expensive. Static checks come first.

## The 60/30/10 ratio

A strong skill splits its instructions into three layers:

- **Deterministic actions (60%)** — exact commands, scripts, file paths, formats.
- **Orchestration (30%)** — process, stages, when to do what.
- **AI judgment (10%)** — the small set of decisions that genuinely require interpretation.

This ratio is a heuristic, not a law. A skill that is entirely mechanical may be 90/10/0. A skill that is mostly judgment may be 30/40/30. The point is to be explicit about which layer each instruction belongs to.

## The 10 evaluation dimensions

| Dimension | Tier | Why it matters |
|---|---|---|
| Spec compliance | 1 | Prevents silent load failures. |
| Trigger | 2 | Decides whether the skill ever runs. |
| Scope | 2 | Keeps the skill focused on one job. |
| Body structure | 1 | Makes the skill readable and navigable. |
| Determinism | 1 | Scripts are the reliable part. |
| Context management | 2 | Keeps context windows usable. |
| Token discipline | 1 | Avoids inlined bloat. |
| Validation | 2 | Provides a way to verify the skill works. |
| Self-contained | 1 | No hidden external dependencies. |
| Examples | 2 | Concrete beats abstract. |

## Progressive disclosure levels

| Level | Content | When loaded |
|---|---|---|
| L0 | Plugin metadata | At agent startup |
| L1 | Skill frontmatter (`name`, `description`) | At agent startup |
| L2 | `SKILL.md` body | When the skill triggers |
| L3 | `references/`, `assets/` | When explicitly referenced |
| L4 | Working artifacts / run output | Per task |

## How to read an audit report

A failing dimension is not a moral judgment. It is a signal about where the skill will likely fail in practice:

- **Spec compliance / Body structure failures** → the skill may not load or the agent may skip steps.
- **Trigger failures** → the skill will sit unused.
- **Determinism / Self-contained failures** → the skill will break when run.
- **Context management / Token discipline failures** → the skill will degrade as it grows.
- **Validation / Examples failures** → the skill is hard to verify or reuse.

Fix Tier 1 failures before Tier 2. When in doubt, cut scope rather than add content.

## References

- Agent Skills spec: https://agentskills.io/specification
- Anthropic — Equipping agents for the real world with Agent Skills: https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills
- NVIDIA SkillEvaluator: https://docs.nvidia.com/skills/evaluating-agent-skills
- SkillsBench: arXiv:2602.12670
- SoK: Agentic Skills: arXiv:2602.20867
- ICM paper: arXiv:2603.16021
