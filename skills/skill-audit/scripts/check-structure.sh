#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: check-structure.sh <skill-dir>" >&2
  exit 1
fi

skill_dir="$1"
skill_md="$skill_dir/SKILL.md"

if [[ ! -f "$skill_md" ]]; then
  echo "FAIL: SKILL.md not found in $skill_dir"
  exit 1
fi

# Count lines
fail=0
line_count="$(wc -l < "$skill_md" | tr -d '[:space:]')"
if [[ "$line_count" -le 500 ]]; then
  echo "OK: SKILL.md is $line_count lines (≤500)"
else
  echo "FAIL: SKILL.md is $line_count lines (>500)"
  fail=1
fi

# Check for key headings (case-insensitive)
for pattern in "^# " "^#{2,6}[[:space:]]+When to use[[:space:]]*$" "^#{2,6}[[:space:]]+.*[Pp]rocess[[:space:]]*$" "^#{2,6}[[:space:]]+Examples?[[:space:]]*$" "^#{2,6}[[:space:]]+Deterministic" "^#{2,6}[[:space:]]+Orchestration" "^#{2,6}[[:space:]]+Constraints"; do
  if grep -qiE "$pattern" "$skill_md"; then
    echo "OK: found heading matching '$pattern'"
  else
    echo "FAIL: missing heading matching '$pattern'"
    fail=1
  fi
done

# Check for code blocks (commands / examples)
if grep -qE "^[[:space:]]*\`\`\`" "$skill_md"; then
  echo "OK: contains code blocks"
else
  echo "FAIL: no code blocks found"
  fail=1
fi

# Check for list items
if grep -qE "^[[:space:]]*[-*]" "$skill_md"; then
  echo "OK: contains list items"
else
  echo "FAIL: no list items found"
  fail=1
fi

# Scripts directory
if [[ -d "$skill_dir/scripts" ]]; then
  script_count="$(find "$skill_dir/scripts" -type f | wc -l | tr -d '[:space:]')"
  echo "OK: scripts/ directory exists ($script_count files)"
else
  echo "INFO: no scripts/ directory"
fi

# References directory
if [[ -d "$skill_dir/references" ]]; then
  ref_count="$(find "$skill_dir/references" -type f | wc -l | tr -d '[:space:]')"
  echo "OK: references/ directory exists ($ref_count files)"
else
  echo "INFO: no references/ directory"
fi

# Assets directory
if [[ -d "$skill_dir/assets" ]]; then
  asset_count="$(find "$skill_dir/assets" -type f | wc -l | tr -d '[:space:]')"
  echo "OK: assets/ directory exists ($asset_count files)"
else
  echo "INFO: no assets/ directory"
fi

exit "$fail"
