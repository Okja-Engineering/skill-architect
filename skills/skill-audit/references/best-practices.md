# Skill evaluation best practices

A synthesis of the open Agent Skills spec, Anthropic's skill-building guidance, NVIDIA's SkillEvaluator framework, the SkillsBench and SoK research literature, and the Interpretable Context Methodology (ICM).

## Agent Skills spec compliance

- One `SKILL.md` per skill directory.
- Frontmatter with `name`, `description`, optional `license`, `compatibility`, `metadata`.
- `name` matches directory name; lowercase, hyphens, 1-64 chars.
- `description` ≤1024 chars, describes what + when to use.
- Optional `scripts/`, `references/`, `assets/` directories.

## Trigger quality

The description is the primary triggering mechanism.

- Imperative phrasing: "Use when..."
- User intent, not implementation.
- Specific contexts and trigger phrases.
- Synonym coverage.
- Negative scope boundaries ("NOT for...").
- Third person (no "you" or "I").
- Be slightly "pushy": skills tend to undertrigger, so include adjacent contexts where the skill applies.

## Scope

- One skill does one job.
- It should fit cleanly into one category of work.
- A skill that straddles multiple categories usually confuses the agent.

## Determinism and scripts

- Mechanical work should be a script; judgment stays in markdown.
- Scripts should have error handling and avoid hardcoded paths.
- File references should be exact relative paths or clearly scoped globs.
- A skill should be self-contained or explicitly source its dependencies.

## Progressive disclosure

- Frontmatter metadata loads first (tiny, always in context).
- `SKILL.md` body loads only when the skill is relevant.
- `references/` and `assets/` load only when explicitly needed.
- Do not instruct the agent to load everything up front.

## ICM context management

- Folder structure carries architecture.
- One stage, one job.
- Plain text (markdown/JSON) is the interface.
- Load only what the current step needs.
- Every output is an edit surface for a human.
- Configure the factory (L3), not the product (L4).

### Five-layer hierarchy

| Layer | File | Question | Role |
|---|---|---|---|
| L0 | `CLAUDE.md` / `AGENTS.md` | Where am I? | routing |
| L1 | root `CONTEXT.md` | Where do I go? | routing |
| L2 | stage `CONTEXT.md` | What do I do? | control point |
| L3 | `references/`, `_shared/`, `_config/` | What rules apply? | factory |
| L4 | `output/`, run artifacts | What am I working with? | product |

For multi-step skills, a stage contract should include Inputs, Process, Outputs, and optionally Checkpoints and Audit.

## Token discipline

- Keep stage context around 2,000–8,000 tokens.
- Do not inline large reference payloads into the main contract.
- Use L3 files for stable rules and point to them from L2.
- Catalog files (L0–L2) should stay small: L0 ~300–800 tokens, L1–L2 ~200–500 tokens each.

## Validation

- A strong skill has a way to verify it works: evals, tests, a checklist, or pass/fail criteria.
- Evals should be specific, realistic, and compare with-skill vs without-skill performance.
- Checklists are better than vague "review this" instructions.

## Examples

- Concrete input/output examples beat abstract descriptions.
- Examples should show the actual file paths, commands, and formats the skill produces.

## Evaluation tiers (NVIDIA SkillEvaluator model)

| Tier | Question | Examples |
|---|---|---|
| Tier 1 — Validation | Is it safe and well-formed? | schema, license, security scan, script lint, PII |
| Tier 2 — Deduplication | Does it overlap with existing skills? | embedding similarity, repeated guidance |
| Tier 3 — Live evaluation | Does it actually help the agent? | with-skill vs without-skill on realistic tasks |

For most local/organizational use, Tier 1 plus a lightweight Tier 2 is the highest-leverage starting point.

## Common anti-patterns

- Loading everything instead of scoping context.
- Burying workflow logic in scripts instead of markdown contracts.
- Missing pass/fail criteria or audit step.
- Putting intermediate state in chat instead of files.
- Overly broad descriptions that trigger too often.
- Descriptions too short or too vague to trigger reliably.
- Hardcoded absolute paths.
- Scripts without error handling.

## References

- Agent Skills spec: https://agentskills.io/specification
- Anthropic — Equipping agents for the real world with Agent Skills: https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills
- Microsoft — Agent Skills: https://learn.microsoft.com/en-us/agent-framework/agents/skills
- NVIDIA — Evaluate Agent Skills Before Publication: https://docs.nvidia.com/skills/evaluating-agent-skills
- SkillsBench: arXiv:2602.12670
- SoK: Agentic Skills — Beyond Tool Use in LLM Agents: arXiv:2602.20867
- ICM paper: arXiv:2603.16021
