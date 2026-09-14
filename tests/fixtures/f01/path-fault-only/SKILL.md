---
name: path-fault-only
description: "A skill that satisfies every house policy rule but references a script path that does not exist."
license: MIT
metadata:
  version: "0.1.0"
---

# path-fault-only

Every house heading is present and the body has code blocks and list items, so
`check-structure.sh` raises no PL002-PL005 policy finding. The only fault is the
dangling script reference below, which `check-paths.sh` reports as PT001.

That isolation is the point of this fixture: a policy finding never round-trips
through `jq`, so a skill with one dies loudly when `jq` is absent. A skill whose
faults are exclusively path findings is the case that must not be silently passed.

## When to use

Use this skill when:

- You need a fixture whose only fault is a path fault.

## Deterministic actions (60%)

```bash
./scripts/does-not-exist.sh
```

## Orchestration (30%)

### Process

1. Run the script that is not there.

## Examples

```bash
./scripts/does-not-exist.sh
```

## Constraints

- Do not use in production.
