#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: check-frontmatter.sh <skill-dir>" >&2
  exit 1
fi

skill_dir="$1"
name="$(basename "$skill_dir")"
skill_md="$skill_dir/SKILL.md"

if [[ ! -f "$skill_md" ]]; then
  echo "FAIL: SKILL.md not found in $skill_dir"
  exit 1
fi

delimiter_count="$(awk '/^---$/ { count++ } END { print count + 0 }' "$skill_md")"
if [[ "$delimiter_count" -lt 2 ]]; then
  echo "FAIL: frontmatter must have opening and closing --- delimiters"
  exit 1
fi

frontmatter="$(awk '
  BEGIN { in_fm = 0 }
  /^---$/ {
    if (in_fm) { exit }
    in_fm = 1
    next
  }
  in_fm { print }
' "$skill_md")"

if [[ -z "$frontmatter" ]]; then
  echo "FAIL: no frontmatter"
  exit 1
fi

fail=0

# name matches directory
if [[ ! "$name" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ || ${#name} -gt 64 ]]; then
  echo "FAIL: directory name must be 1-64 lowercase letters, numbers, and hyphen-separated words"
  fail=1
fi
actual_name="$(echo "$frontmatter" | awk '/^name:/ { sub(/^name:[[:space:]]*/, ""); sub(/[[:space:]]*$/, ""); print; exit }')"
if [[ "$actual_name" == "$name" ]]; then
  echo "OK: name matches directory ($name)"
else
  echo "FAIL: name mismatch: ${actual_name:-<missing>} != $name"
  fail=1
fi

# description present and <= 1024 chars
if echo "$frontmatter" | grep -qE "^description:[[:space:]]+"; then
  description="$(echo "$frontmatter" | awk '/^description:/ { sub(/^description:[[:space:]]*/, ""); print }')"
  desc_len="${#description}"
  if [[ "$desc_len" -le 1024 ]]; then
    echo "OK: description present ($desc_len chars)"
  else
    echo "FAIL: description too long ($desc_len > 1024)"
    fail=1
  fi
else
  echo "FAIL: missing description"
  fail=1
fi

# license present
if echo "$frontmatter" | grep -qE "^license:[[:space:]]+"; then
  echo "OK: license present"
else
  echo "FAIL: missing license"
  fail=1
fi

if [[ "$fail" -eq 1 ]]; then
  exit 1
fi

echo "frontmatter OK"
