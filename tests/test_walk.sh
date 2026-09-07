#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

pass=0
fail=0

assert() {
  local label="$1"
  shift
  if "$@"; then
    pass=$((pass + 1))
  else
    echo "FAIL: $label"
    fail=$((fail + 1))
  fi
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# The distributable unit is a plugin; skills live under skills/.
test_plugin_layout() {
  python3 - <<'PY'
import json
with open('.devin-plugin/plugin.json') as f:
    manifest = json.load(f)
assert manifest['name'] == 'skill-architect'
assert manifest['skills'] == 'skills'
PY
  assert "plugin points to canonical skill directory" true
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
  assert "bad skill missing When to use" grep -q "FAIL: missing heading matching '.*When to use" "$out"
  assert "bad skill missing Examples" grep -q "FAIL: missing heading matching '.*Examples" "$out"
  assert "bad skill missing Deterministic" grep -q "FAIL: missing heading matching '.*Deterministic'" "$out"
  assert "bad skill missing Orchestration" grep -q "FAIL: missing heading matching '.*Orchestration'" "$out"
  assert "bad skill missing Constraints" grep -q "FAIL: missing heading matching '.*Constraints'" "$out"
  assert "bad skill has no code blocks" grep -q "FAIL: no code blocks found" "$out"
}

# Rewrite drafts a plan for the broken skill.
test_rewrite_drafts_plan() {
  local bad="$tmp/bad-skill"
  skills/skill-rewrite/scripts/draft-rewrite.sh -t "$bad" >/dev/null
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

  skills/skill-audit/scripts/check-frontmatter.sh "$fixed" >/dev/null
  assert "fixed skill frontmatter passes" true
  skills/skill-audit/scripts/check-structure.sh "$fixed" >/dev/null
  assert "fixed skill structure passes" true
}

# The skill-audit skill can audit the skill-rewrite skill and vice versa.
test_skills_audit_each_other() {
  skills/skill-audit/scripts/check-frontmatter.sh skills/skill-rewrite >/dev/null
  assert "skill-audit can audit skill-rewrite frontmatter" true
  skills/skill-audit/scripts/check-structure.sh skills/skill-rewrite >/dev/null
  assert "skill-audit can audit skill-rewrite structure" true
  skills/skill-audit/scripts/check-frontmatter.sh skills/skill-audit >/dev/null
  assert "skill-audit can audit itself frontmatter" true
  skills/skill-audit/scripts/check-structure.sh skills/skill-audit >/dev/null
  assert "skill-audit can audit itself structure" true
}

test_plugin_layout
test_audit_catches_bad_skill
test_rewrite_drafts_plan
test_fixes_improve_audit
test_skills_audit_each_other

echo
echo "$pass passed, $fail failed"
if [[ "$fail" -gt 0 ]]; then
  exit 1
fi
