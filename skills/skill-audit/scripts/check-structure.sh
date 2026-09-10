#!/usr/bin/env bash
# Check structure: house-policy checks (headings, line limit, code blocks, lists)
# via grep, and local reference path validation via check-paths.sh.
# skill-validator handles spec compliance and link resolution separately;
# this script covers our repository-specific policy rules (PL002-PL005, PT001-PT002).
# Exit codes: 0=pass, 1=path failure, 2=policy failure, 3=execution error.
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: check-structure.sh <skill-dir>" >&2
  exit 3
fi

skill_dir="$1"
skill_md="$skill_dir/SKILL.md"

if [[ ! -f "$skill_md" ]]; then
  echo "FAIL: SKILL.md not found in $skill_dir"
  exit 3
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fail=0

# --- Policy checks (grep-based absence detection) ---

# Required headings (case-insensitive) — match the original check-structure.sh patterns
for heading_spec in "When to use" ".*Process" "Examples?" "Deterministic" "Orchestration" "Constraints"; do
  if ! grep -qiE "^#{2,6}[[:space:]]+${heading_spec}" "$skill_md"; then
    echo "POLICY FAIL [PL002]: missing heading matching '${heading_spec}'"
    fail=2
  fi
done

# Code blocks
if ! grep -qE '^[[:space:]]*```' "$skill_md"; then
  echo "POLICY FAIL [PL004]: no code blocks found"
  fail=2
fi

# List items
if ! grep -qE '^[[:space:]]*[-*]' "$skill_md"; then
  echo "POLICY FAIL [PL005]: no list items found"
  fail=2
fi

# Line count
line_count="$(wc -l < "$skill_md" | tr -d '[:space:]')"
if [[ "$line_count" -gt 500 ]]; then
  echo "POLICY FAIL [PL003]: SKILL.md is $line_count lines (max 500)"
  fail=2
fi

# --- Path checks ---
path_output="$("$script_dir/check-paths.sh" "$skill_dir" 2>&1)" || path_code=$? || path_code=0
path_code=${path_code:-0}
echo "$path_output"

if [[ $path_code -eq 1 && $fail -eq 0 ]]; then
  fail=1
fi

exit $fail
