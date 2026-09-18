---
name: no-rule-before-refs
description: A skill whose broken references all sit after a markdown horizontal rule, so a frontmatter scan that re-enters on a third delimiter would stop reading before reaching any of them.
license: MIT
---

## When to use

Use this fixture to prove that frontmatter ends once and a horizontal rule is body.

## Process

Its twin, `rule-before-refs`, is byte-identical but for the rule below.


- A list item, so PL005 is satisfied.

See [the missing checklist](references/checklists.md) for the criteria.

```bash
./scripts/does-not-exist.sh "$target_skill"
```

## Examples

## Deterministic actions (60%)

## Orchestration (30%)

## Constraints

None.
