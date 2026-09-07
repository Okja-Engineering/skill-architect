# Evaluation matrix

This matrix combines the Agent Skills spec, Anthropic best practices, ICM context-management criteria, and the NVIDIA three-tier evaluation model. Score each dimension 0–2:

- 0 = missing or broken
- 1 = present but weak
- 2 = strong

## Dimensions

| Dimension | Strong signal |
|---|---|
| **Spec compliance** | Directory name, frontmatter, file layout all correct. |
| **Trigger** | Imperative, intent-focused, specific contexts, synonyms, negative scope, ≤1024 chars. |
| **Scope** | One job; does not straddle categories. |
| **Body structure** | Imperative, <500 lines, purpose, when-to-use, workflow, examples, constraints. |
| **Determinism** | Scripts for mechanical work, exact paths, error handling, checklists. |
| **Context mgmt** | Progressive disclosure, scoped loading, stage contracts, L3/L4 separation. |
| **Token discipline** | L0–L2 small, no inlined L3 payloads, stage context ~2k–8k tokens. |
| **Validation** | Checklist, evals, tests, or pass/fail criteria. |
| **Self-contained** | All resources in directory or explicitly sourced. |
| **Examples** | Concrete input/output examples, not only abstract descriptions. |

## Evaluation tiers

| Tier | Focus | Cost | When to run |
|---|---|---|---|
| **Tier 1 — Validation** | Spec compliance, structure, determinism, scripts, self-containedness, token discipline | Cheap | Every edit, pre-commit, before release |
| **Tier 2 — Quality** | Trigger quality, scope, context management, validation, examples; plus overlap check against other skills | Medium | Before release, when adding to a catalog |
| **Tier 3 — Live efficacy** | With-skill vs without-skill performance on representative tasks | Expensive | Before publishing, when behavior is uncertain |

## Suggested output format

```markdown
# Skill audit: <skill-name>

| Dimension | Score | Notes |
|---|---|---|
| Spec compliance | 0/1/2 | ... |
| Trigger | 0/1/2 | ... |
| Scope | 0/1/2 | ... |
| Body structure | 0/1/2 | ... |
| Determinism | 0/1/2 | ... |
| Context mgmt | 0/1/2 | ... |
| Token discipline | 0/1/2 | ... |
| Validation | 0/1/2 | ... |
| Self-contained | 0/1/2 | ... |
| Examples | 0/1/2 | ... |

## Ratio

- Deterministic: X%
- Orchestration: Y%
- AI judgment: Z%

## Fixes

1. ...
```

Use the ratio 60/30/10 as a default when a skill explicitly labels sections as Deterministic actions (60%), Orchestration (30%), and AI judgment (10%).
