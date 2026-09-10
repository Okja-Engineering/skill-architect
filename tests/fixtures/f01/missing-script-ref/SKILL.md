---
name: missing-script-ref
description: A skill that references a script path that does not exist.
license: MIT
---

# missing-script-ref

## When to use

Use this skill when:

- You need to test missing script detection.

## Deterministic actions (60%)

```bash
./scripts/does-not-exist.sh
```

## Orchestration (30%)

1. Run the script.

## Examples

```bash
./scripts/does-not-exist.sh
```

## Constraints

- Do not use in production.
