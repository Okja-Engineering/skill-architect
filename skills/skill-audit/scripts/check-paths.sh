#!/usr/bin/env bash
# Check local file references in a skill directory.
# Finds executable script references (./ or $ prefixed) in code blocks
# and markdown links, then resolves them against the filesystem.
# Exit codes: 0=pass, 1=path failure, 3=execution error.
# Findings carry rule IDs: at level fail, PT001 missing script/reference and
# PT002 missing markdown link, plus DEP001 when a required tool is absent; at
# level unverified, PATH for a reference built from a glob or a variable, which
# cannot be resolved and so is reported without being judged — an unverified
# finding is not a failure and does not change the exit status.
# Use --json for machine-readable output: {"findings": [...], "passed": bool},
# plus an "error" key on the exit-3 payload naming why no verdict was reached.
# --json builds its verdict with jq and requires it.
# In --json mode stdout is the payload channel: it carries a payload or it
# carries nothing, and every diagnostic goes to stderr. Three exits carry no
# payload: a usage error and an unresolvable target, where there is no skill to
# render a verdict about, and a verdict-guard.sh that would not load, where
# there is nothing left to build a payload with.
set -euo pipefail

script_dir="$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
verdict_guard="$script_dir/verdict-guard.sh"
# The guard is the one dependency it cannot announce itself, so loading it is
# checked before and after — see its header for why an unchecked source would
# exit with a status that means a verdict was computed.
bash -n "$verdict_guard" 2>/dev/null \
  || { echo "ERROR: cannot load $verdict_guard: missing or malformed; no verdict was computed" >&2; exit 3; }
# shellcheck source=verdict-guard.sh
source "$verdict_guard"
{ declare -F verdict_guard_ready >/dev/null && verdict_guard_ready; } \
  || { echo "ERROR: $verdict_guard did not load its guards; no verdict was computed" >&2; exit 3; }

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

if $json_output; then
  require_tool jq true
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
  # Build JSON findings array from pipe-delimited entries.
  json_findings="[]"
  for f in ${findings[@]+"${findings[@]}"}; do
    level="${f%%|*}"
    rest="${f#*|}"
    rule="${rest%%|*}"
    message="${rest#*|}"
    json_findings=$(echo "$json_findings" | jq --arg level "$level" --arg rule "$rule" --arg msg "$message" \
      '. + [{"level": $level, "rule": $rule, "message": $msg}]')
  done
  echo "$json_findings" | jq --argjson passed "$([[ $fail -eq 0 ]] && echo true || echo false)" \
    '{findings: ., passed: $passed}'
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
