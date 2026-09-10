#!/usr/bin/env bash
# Check local file references in a skill directory.
# Finds executable script references (./ or $ prefixed) in code blocks
# and markdown links, then resolves them against the filesystem.
# Exit codes: 0=pass, 1=path failure
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: check-paths.sh <skill-dir>" >&2
  exit 3
fi

skill_dir="$1"
skill_md="$skill_dir/SKILL.md"

if [[ ! -f "$skill_md" ]]; then
  echo "ERROR: SKILL.md not found in $skill_dir" >&2
  exit 3
fi

fail=0

# Extract body (after frontmatter)
body="$(awk '
  BEGIN { in_fm = 0 }
  /^---$/ {
    if (in_fm) { in_fm = 0; next }
    in_fm = 1; next
  }
  !in_fm { print }
' "$skill_md")"

# Check markdown links [text](path) — skip http/https/anchor/mailto
while IFS= read -r path; do
  [[ -z "$path" ]] && continue
  [[ "$path" =~ ^https?:// ]] && continue
  [[ "$path" =~ ^# ]] && continue
  [[ "$path" =~ ^mailto: ]] && continue
  if echo "$path" | grep -qE '[*?]|\$'; then
    echo "UNVERIFIED [PATH]: dynamic/glob link: $path"
    continue
  fi
  if [[ ! -e "$skill_dir/$path" ]]; then
    echo "PATH FAIL [PT002]: markdown link target not found: $path"
    fail=1
  fi
done < <(echo "$body" | grep -oE '\[[^]]*\]\([^)]+\)' | sed 's/\[[^]]*\](\([^)]*\))/\1/')

# Check executable script references in code blocks (./ or $ prefixed)
# Extract code block content, then find path references
code_body="$(echo "$body" | awk '
  /^```/ { in_code = !in_code; next }
  in_code { print }
')"

while IFS= read -r path; do
  [[ -z "$path" ]] && continue
  path="$(echo "$path" | tr -d "\"'")"
  if echo "$path" | grep -qE '[*?]|\$'; then
    echo "UNVERIFIED [PATH]: dynamic/glob path: $path"
    continue
  fi
  if [[ ! -e "$skill_dir/$path" ]]; then
    echo "PATH FAIL [PT001]: script/reference path not found: $path"
    fail=1
  fi
done < <(echo "$code_body" | grep -oE '(?:\./|\$)\S*(?:scripts|references|assets)/\S+' 2>/dev/null || true)

exit $fail
