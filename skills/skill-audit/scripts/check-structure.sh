#!/usr/bin/env bash
# Check structure: house-policy checks (headings, line limit, code blocks, lists)
# via grep, and local reference path validation via check-paths.sh.
# skill-validator handles spec compliance and link resolution separately;
# this script covers our repository-specific policy rules (PL002-PL005, PT001-PT002).
# Exit codes: 0=pass, 1=path failure, 2=policy failure, 3=execution error.
# Use --json for machine-readable output: {"findings": [...], "passed": bool}
set -euo pipefail

json_output=false
skill_dir=""

for arg in "$@"; do
  case "$arg" in
    --json) json_output=true ;;
    -*) echo "Usage: check-structure.sh [--json] <skill-dir>" >&2; exit 3 ;;
    *) skill_dir="$arg" ;;
  esac
done

if [[ -z "$skill_dir" ]]; then
  echo "Usage: check-structure.sh [--json] <skill-dir>" >&2
  exit 3
fi

skill_md="$skill_dir/SKILL.md"

if [[ ! -f "$skill_md" ]]; then
  echo "FAIL: SKILL.md not found in $skill_dir"
  exit 3
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fail=0
findings=()

# --- Policy checks (grep-based absence detection) ---

# Required headings (case-insensitive) — match the original check-structure.sh patterns
for heading_spec in "When to use" ".*Process" "Examples?" "Deterministic" "Orchestration" "Constraints"; do
  if ! grep -qiE "^#{2,6}[[:space:]]+${heading_spec}" "$skill_md"; then
    findings+=("fail|PL002|missing heading matching '${heading_spec}'")
    fail=2
  fi
done

# Code blocks
if ! grep -qE '^[[:space:]]*```' "$skill_md"; then
  findings+=("fail|PL004|no code blocks found")
  fail=2
fi

# List items
if ! grep -qE '^[[:space:]]*[-*]' "$skill_md"; then
  findings+=("fail|PL005|no list items found")
  fail=2
fi

# Line count
line_count="$(wc -l < "$skill_md" | tr -d '[:space:]')"
if [[ "$line_count" -gt 500 ]]; then
  findings+=("fail|PL003|SKILL.md is $line_count lines (max 500)")
  fail=2
fi

# --- Path checks ---
if $json_output; then
  path_json="$("$script_dir/check-paths.sh" --json "$skill_dir" 2>&1)" || path_code=$? || path_code=0
  path_code=${path_code:-0}
  # Merge path findings into our findings array.
  path_findings=$(echo "$path_json" | jq -c '.findings[]' 2>/dev/null || true)
  if [[ -n "$path_findings" ]]; then
    while IFS= read -r pf; do
      level=$(echo "$pf" | jq -r '.level')
      rule=$(echo "$pf" | jq -r '.rule')
      message=$(echo "$pf" | jq -r '.message')
      findings+=("${level}|${rule}|${message}")
    done <<< "$path_findings"
  fi
else
  path_output="$("$script_dir/check-paths.sh" "$skill_dir" 2>&1)" || path_code=$? || path_code=0
  path_code=${path_code:-0}
  echo "$path_output"
fi

if [[ $path_code -eq 1 && $fail -eq 0 ]]; then
  fail=1
fi

# --- Output ---
if $json_output; then
  if [[ ${#findings[@]} -eq 0 ]]; then
    echo '{"findings": [], "passed": true}'
  else
    json_findings="[]"
    for f in ${findings[@]+"${findings[@]}"}; do
      level="${f%%|*}"
      rest="${f#*|}"
      rule="${rest%%|*}"
      message="${rest#*|}"
      json_findings=$(echo "$json_findings" | jq --arg level "$level" --arg rule "$rule" --arg msg "$message" \
        '. + [{"level": $level, "rule": $rule, "message": $msg}]')
    done
    echo "$json_findings" | jq --argjson passed $([[ $fail -eq 0 ]] && echo true || echo false) \
      '{findings: ., passed: $passed}'
  fi
else
  for f in ${findings[@]+"${findings[@]}"}; do
    level="${f%%|*}"
    rest="${f#*|}"
    rule="${rest%%|*}"
    message="${rest#*|}"
    echo "POLICY FAIL [$rule]: $message"
  done
fi

exit $fail
