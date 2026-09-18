---
name: all-sections
description: A skill that already carries every heading draft-rewrite.sh looks for. Use when a test needs a target for which no section template fires.
license: MIT
metadata:
  version: "0.1.0"
---

# all-sections

A compliant skill that holds every heading the rewrite drafter probes for, so a
draft generated for it carries no section template at all. Its counterpart is
tests/fixtures/f01/valid-minimal, which holds none of them and so draws every
template. Between the two, each probe is observed both firing and not firing —
which is what makes the section inventory in skill-rewrite's SKILL.md a
comparison over the whole set rather than over one path through it.

## When to use

Use this skill when:

- A test needs a target that draws no section template.

## Deterministic actions (60%)

```bash
./scripts/run.sh
```

### Validation

Confirm the script ran:

```bash
./scripts/run.sh
```

## Orchestration (30%)

### Process

1. Run the script.

## AI judgment (10%)

- Decide nothing; this is a fixture.

## Examples

```bash
./scripts/run.sh
```

## Constraints

- This is a test fixture and has no use outside the suite.
