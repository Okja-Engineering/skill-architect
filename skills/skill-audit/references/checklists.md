# Evaluation checklists

Use these checklists for criteria that the scripts do not cover mechanically. Consult [best practices](best-practices.md) for criteria requiring interpretation.

## Agent Skills spec checklist

- Directory name matches `name` in frontmatter.
- `name` is lowercase, hyphenated, 1-64 chars.
- `description` ≤1024 chars, describes what + when to use.
- `license` present.
- `SKILL.md` is at the skill root.

## Trigger-quality checklist

- Imperative phrasing: "Use when..."
- User intent, not implementation details.
- Specific contexts and trigger phrases.
- Synonym coverage for key terms.
- Negative scope boundaries if the edge is unclear.
- Third person; no "you" or "I".

## Body checklist

- Imperative voice throughout.
- Under ~500 lines.
- Clear purpose / what this does.
- "When to use" with specific scenarios.
- Structured process or workflow.
- At least one concrete example with real paths or commands.
- Constraints or guardrails.

## Determinism checklist

- Mechanical work is in scripts, not prose.
- Scripts have error handling (`set -euo pipefail` or equivalent).
- No hardcoded absolute paths.
- Exact commands and file references.
- Checklists instead of vague "review this" instructions.

## ICM context-management checklist

- Progressive disclosure: frontmatter → body → references/assets.
- No "load everything" instructions.
- Multi-step skills use stage contracts: Inputs, Process, Outputs.
- Working artifacts (per-run) separated from stable reference material.
- L0–L2 catalog files stay small; L3 reference payloads are pointed at, not inlined.

## Validation checklist

- Skill includes a checklist, evals, tests, or pass/fail criteria.
- Criteria are concrete enough to verify.
- Evals are specific and realistic if present.

## Self-contained checklist

- All needed resources are in the skill directory or explicitly sourced.
- Templates are blank and reusable, not filled-in deployments.
- No hidden external dependencies.
