---
name: dynamic-paths
description: A skill that uses variable-based paths that cannot be statically resolved.
license: MIT
---

# dynamic-paths

## When to use

Use this skill when:

- You need to test dynamic path detection.

## Deterministic actions (60%)

```bash
"$skill_root/scripts/run.sh"
"$target_skill/scripts/check.sh"
```

## Orchestration (30%)

1. Resolve the variable paths at runtime.

## Examples

```bash
"$skill_root/scripts/run.sh"
```

## Constraints

- Variable paths are reported as unverified, not failures.
