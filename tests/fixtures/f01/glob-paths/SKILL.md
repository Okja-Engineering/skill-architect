---
name: glob-paths
description: A skill that uses glob patterns that cannot be statically resolved.
license: MIT
---

# glob-paths

## When to use

Use this skill when:

- You need to test glob path detection.

## Deterministic actions (60%)

Read these in order:

1. `./scripts/*`
2. `./references/*`

## Orchestration (30%)

1. Expand the globs at runtime.

## Examples

```bash
ls ./scripts/*
```

## Constraints

- Glob paths are reported as unverified, not failures.
