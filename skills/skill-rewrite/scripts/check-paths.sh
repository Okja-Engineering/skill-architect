#!/usr/bin/env bash
# Check local file references in a skill directory.
# Finds executable script references (./ or $ prefixed) in code blocks
# and markdown links, then resolves them against the filesystem.
# Exit codes: 0=pass, 1=path failure, 3=execution error.
# Use --json for machine-readable output: {"findings": [...], "passed": bool}
set -euo pipefail

json_output=false
skill_dir=""

for arg in "$@"; do
  case "$arg" in
    --json) json_output=true ;;
    -*) echo "Usage: check-paths.sh [--json] <skill-dir>" >&2; exit 3 ;;
    *) skill_dir="$arg" ;;
  esac
done

if [[ -z "$skill_dir" ]]; then
  echo "Usage: check-paths.sh [--json] <skill-dir>" >&2
  exit 3
fi

skill_md="$skill_dir/SKILL.md"

if [[ ! -f "$skill_md" ]]; then
  echo "ERROR: SKILL.md not found in $skill_dir" >&2
  exit 3
fi

fail=0
findings=()

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
    findings+=("unverified|PATH|dynamic/glob link: $path")
    continue
  fi
  if [[ ! -e "$skill_dir/$path" ]]; then
    findings+=("fail|PT002|markdown link target not found: $path")
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
    findings+=("unverified|PATH|dynamic/glob path: $path")
    continue
  fi
  if [[ ! -e "$skill_dir/$path" ]]; then
    findings+=("fail|PT001|script/reference path not found: $path")
    fail=1
  fi
done < <(echo "$code_body" | grep -oE '(\./|\$)\S*(scripts|references|assets)/\S+' 2>/dev/null || true)

# --- Output ---
if $json_output; then
  if [[ ${#findings[@]} -eq 0 ]]; then
    echo '{"findings": [], "passed": true}'
  else
    # Build JSON findings array from pipe-delimited entries.
    json_findings="[]"
    for f in "${findings[@]}"; do
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
    if [[ "$level" == "fail" ]]; then
      echo "PATH FAIL [$rule]: $message"
    else
      echo "UNVERIFIED [$rule]: $message"
    fi
  done
fi

exit $fail
