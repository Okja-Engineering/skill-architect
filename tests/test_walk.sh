#!/usr/bin/env bash
set -euo pipefail
. "$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)/lib/harness.sh"
harness_init

tmp="$harness_scratch/walk"
mkdir -p "$tmp"

# The distributable unit is a plugin; skills live under skills/.
#
# The manifest read is a function rather than a bare heredoc followed by
# `assert … true`, because a check on one line and the assertion that reports it
# on the next are not the same statement: errexit kills the suite at the check
# and the label is never printed, or bash 3.2 runs on and prints PASS. The
# assertion has to *be* the check.
plugin_points_to_canonical_skill_dir() {
  python3 - <<'PY'
import json
with open('.devin-plugin/plugin.json') as f:
    manifest = json.load(f)
assert manifest['name'] == 'skill-architect'
assert manifest['skills'] == 'skills'
PY
}

test_plugin_layout() {
  assert "plugin points to canonical skill directory" \
    quietly plugin_points_to_canonical_skill_dir
  assert "no skill contains a plugin manifest" test ! -e skills/skill-audit/.devin-plugin/plugin.json
  assert "no skill contains a plugin manifest" test ! -e skills/skill-rewrite/.devin-plugin/plugin.json
}

# Audit a deliberately broken skill and confirm failures are reported.
test_audit_catches_bad_skill() {
  local bad="$tmp/bad-skill"
  mkdir -p "$bad"
  cat > "$bad/SKILL.md" <<'EOF'
---
name: bad-skill
description: A bad skill.
license: MIT
---

# bad-skill

This skill has no sections.
EOF

  local out="$tmp/audit-out.txt"
  skills/skill-audit/scripts/check-frontmatter.sh "$bad" > "$out" 2>&1 || true
  skills/skill-audit/scripts/check-structure.sh "$bad" >> "$out" 2>&1 || true

  assert "bad skill frontmatter passes" grep -q "frontmatter OK" "$out"
  assert "bad skill missing When to use" grep -q "POLICY FAIL.*When to use" "$out"
  assert "bad skill missing Examples" grep -q "POLICY FAIL.*Examples" "$out"
  assert "bad skill missing Deterministic" grep -q "POLICY FAIL.*Deterministic" "$out"
  assert "bad skill missing Orchestration" grep -q "POLICY FAIL.*Orchestration" "$out"
  assert "bad skill missing Constraints" grep -q "POLICY FAIL.*Constraints" "$out"
  assert "bad skill has no code blocks" grep -q "POLICY FAIL.*no code blocks" "$out"
}

# Rewrite drafts a plan for the broken skill.
test_rewrite_drafts_plan() {
  local bad="$tmp/bad-skill"
  assert "rewrite drafts a plan for a broken skill" \
    quietly skills/skill-rewrite/scripts/draft-rewrite.sh -t "$bad"
  assert "rewrite draft created" test -f "$bad/REWRITE-DRAFT.md"
  assert "rewrite draft mentions When to use" grep -q "When to use" "$bad/REWRITE-DRAFT.md"
  assert "rewrite draft mentions Examples" grep -q "Examples" "$bad/REWRITE-DRAFT.md"
}

# Apply the structural fixes from the draft and confirm the skill passes audit.
test_fixes_improve_audit() {
  local bad="$tmp/bad-skill"
  local fixed="$tmp/fixed-skill"
  cp -R "$bad" "$fixed"

  # Reconstruct a minimal spec-compliant skill based on the draft's templates.
  cat > "$fixed/SKILL.md" <<'EOF'
---
name: fixed-skill
description: A fixed skill. Use when you need a minimal example that passes the Agent Skills spec.
license: MIT
---

# fixed-skill

A minimal spec-compliant skill.

## When to use

Use this skill when:

- You need a placeholder skill for testing.
- You want to verify a harness.

Do not use this skill in production.

## Deterministic actions (60%)

### Step one

Run the example:

```bash
echo "example"
```

### Validation

Check the output:

```bash
./scripts/example.sh
```

Validation checklist:

- [ ] Output is "example".

## Orchestration (30%)

### Process

1. Run the example.
2. Validate the output.

## AI judgment (10%)

- Decide whether the placeholder is sufficient for the task.

## Examples

```bash
./scripts/example.sh
```

Expected output:

```text
example
```

## Constraints

- This is a placeholder skill.
- Do not use in production.
EOF

  mkdir -p "$fixed/scripts"
  cat > "$fixed/scripts/example.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "example"
EOF
  chmod +x "$fixed/scripts/example.sh"

  assert "fixed skill frontmatter passes" \
    quietly skills/skill-audit/scripts/check-frontmatter.sh "$fixed"
  assert "fixed skill structure passes" \
    quietly skills/skill-audit/scripts/check-structure.sh "$fixed"
}

# The skill-audit skill can audit the skill-rewrite skill and vice versa.
test_skills_audit_each_other() {
  assert "skill-audit can audit skill-rewrite frontmatter" \
    quietly skills/skill-audit/scripts/check-frontmatter.sh skills/skill-rewrite
  assert "skill-audit can audit skill-rewrite structure" \
    quietly skills/skill-audit/scripts/check-structure.sh skills/skill-rewrite
  assert "skill-audit can audit itself frontmatter" \
    quietly skills/skill-audit/scripts/check-frontmatter.sh skills/skill-audit
  assert "skill-audit can audit itself structure" \
    quietly skills/skill-audit/scripts/check-structure.sh skills/skill-audit
}

test_plugin_layout
test_audit_catches_bad_skill
test_rewrite_drafts_plan
test_fixes_improve_audit
test_skills_audit_each_other

harness_summary
