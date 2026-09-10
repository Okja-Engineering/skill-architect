#!/usr/bin/env bash
set -euo pipefail

pass=0
fail=0

assert() {
  local label="$1"
  local condition="$2"
  if [[ "$condition" == "true" ]]; then
    echo "PASS: $label"
    pass=$((pass + 1))
  else
    echo "FAIL: $label"
    fail=$((fail + 1))
  fi
}

# Plugin manifests are valid JSON and version matches.
for manifest in .devin-plugin/plugin.json .claude-plugin/plugin.json .cursor-plugin/plugin.json .codex-plugin/plugin.json; do
  [[ -f "$manifest" ]]
  assert "$manifest exists" true
  python3 - <<PY
import json, sys
with open('$manifest') as f:
    data = json.load(f)
assert data['name'] == 'skill-architect', 'name mismatch'
assert data['version'] == '0.3.1', 'version mismatch'
assert 'skills' in data, 'missing skills'
PY
  assert "$manifest is valid plugin.json" true
done

# Each skill has a valid SKILL.md with frontmatter and name matching directory.
for skill in skill-audit skill-rewrite; do
  dir="skills/$skill"
  [[ -d "$dir" ]]
  assert "skills/$skill directory exists" true
  [[ -f "$dir/SKILL.md" ]]
  assert "skills/$skill/SKILL.md exists" true

  # Name in frontmatter matches directory.
  name_in_file="$(awk '
    BEGIN { in_fm = 0 }
    /^---$/ {
      if (in_fm) { exit }
      in_fm = 1
      next
    }
    in_fm && /^name:/ { sub(/^name:[[:space:]]*/, ""); sub(/[[:space:]]*$/, ""); print; exit }
  ' "$dir/SKILL.md")"
  [[ "$name_in_file" == "$skill" ]]
  assert "skills/$skill frontmatter name matches directory" true
done

# skill-audit carries its scripts and references.
[[ -x skills/skill-audit/scripts/check-frontmatter.sh ]]
assert "skill-audit check-frontmatter.sh is executable" true
[[ -x skills/skill-audit/scripts/check-structure.sh ]]
assert "skill-audit check-structure.sh is executable" true
[[ -f skills/skill-audit/references/best-practices.md ]]
assert "skill-audit best-practices reference exists" true
[[ -f skills/skill-audit/references/evaluation-matrix.md ]]
assert "skill-audit evaluation-matrix reference exists" true

# The skill-audit scripts can audit themselves.
skills/skill-audit/scripts/check-frontmatter.sh skills/skill-audit >/dev/null
assert "skill-audit passes its own frontmatter check" true
skills/skill-audit/scripts/check-structure.sh skills/skill-audit >/dev/null
assert "skill-audit passes its own structure check" true

# skill-rewrite carries its script and can draft a rewrite for itself.
[[ -x skills/skill-rewrite/scripts/draft-rewrite.sh ]]
assert "skill-rewrite draft-rewrite.sh is executable" true
tmp_skill="$(mktemp -d)"
cp -R skills/skill-rewrite "$tmp_skill/skill-rewrite-test"
skills/skill-rewrite/scripts/draft-rewrite.sh -t "$tmp_skill/skill-rewrite-test" >/dev/null
[[ -f "$tmp_skill/skill-rewrite-test/REWRITE-DRAFT.md" ]]
assert "skill-rewrite draft-rewrite.sh produces REWRITE-DRAFT.md" true
rm -rf "$tmp_skill"

echo
echo "$pass passed, $fail failed"
if [[ "$fail" -gt 0 ]]; then
  exit 1
fi
