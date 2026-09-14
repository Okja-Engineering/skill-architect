#!/usr/bin/env bash
# Check structure: house-policy checks (headings, line limit, code blocks, lists)
# via grep, and local reference path validation via check-paths.sh.
# skill-validator handles spec compliance and link resolution separately;
# this script covers our repository-specific policy rules (PL002-PL005, PT001-PT002).
# Exit codes: 0=pass, 1=path failure, 2=policy failure, 3=execution error.
# Findings carry rule IDs: PL002 a missing heading, PL003 a SKILL.md over the
# line limit, PL004 no code blocks, PL005 no list items; DEP001 a required tool
# is absent; DEP002 check-paths.sh returned a result this script cannot read —
# an unenumerated exit status, a payload that is not the documented shape, or a
# payload that contradicts itself or the status it arrived with. In --json mode
# it also relays check-paths.sh's own findings unchanged, so PT001, PT002 and
# PATH reach a consumer through here too.
# Use --json for machine-readable output: {"findings": [...], "passed": bool},
# plus an "error" key on the exit-3 payload naming why no verdict was reached.
# --json builds its verdict with jq and requires it.
# In --json mode stdout is the payload channel: it carries a payload or it
# carries nothing, and every diagnostic goes to stderr. Three exits carry no
# payload: a usage error and an unresolvable target, where there is no skill to
# render a verdict about, and a verdict-guard.sh that would not load, where
# there is nothing left to build a payload with.
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
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
  echo "ERROR: SKILL.md not found in $skill_dir" >&2
  exit 3
fi

if $json_output; then
  require_tool jq true
fi

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
# Keep the child's stdout clean of its diagnostics: in JSON mode stdout is a
# payload we parse.
path_code=0
if $json_output; then
  path_json="$("$script_dir/check-paths.sh" --json "$skill_dir")" || path_code=$?
else
  path_output="$("$script_dir/check-paths.sh" "$skill_dir" 2>&1)" || path_code=$?
fi

# check-paths.sh exits 0=pass, 1=path failure. Any other status means it did not
# reach a verdict, so neither did we.
case $path_code in
  0|1) ;;
  *) cannot_compute DEP002 "check-paths.sh exited with unexpected status $path_code" "$json_output" ;;
esac

# The child's verdict, derived once. Everything below reads this rather than
# testing $path_code again, so our own exit status and the findings we merged
# cannot drift into a payload that reports a pass beside a failing finding.
path_passed=true
[[ $path_code -eq 0 ]] || path_passed=false

if $json_output; then
  # Three questions about the child's result, each answered on its own and in
  # this order: is the payload the documented shape, does it agree with itself,
  # and does it agree with the status it arrived with. Any "no" is the same
  # answer — there was no verdict there to read, DEP002 — but they are not the
  # same question, and folding the first into the others is what let a misshapen
  # payload through. Shape was asked in the expression that read the verdict,
  # joined by an `and` that short-circuits, so it went unasked on the branch the
  # verdict took; the read-back below then indexed elements nothing had checked
  # and died with jq's own status.
  #
  # Silence is not a payload and neither is a contradiction: reading either as
  # zero findings is the same silent pass as reading garbage as zero findings.

  # 1. Shape, proven before anything is read and whatever the verdict says.
  payload_is_conforming "$path_json" \
    || cannot_compute DEP002 "check-paths.sh --json did not produce a readable payload" true

  # 2. The child's own verdict, now safe to read, against its own findings. A
  #    payload claiming it passed while carrying a finding that says it failed
  #    contradicts itself, and neither half can be believed over the other.
  child_passed=$(echo "$path_json" | jq -r 'if .passed then "true" else "false" end')
  child_has_fail=$(echo "$path_json" | jq -r 'if any(.findings[]; .level == "fail") then "true" else "false" end')
  if [[ "$child_passed" == "true" && "$child_has_fail" == "true" ]]; then
    cannot_compute DEP002 "check-paths.sh --json payload claims it passed beside a finding that says it failed" true
  fi

  # 3. And against the status it arrived with.
  [[ "$child_passed" == "$path_passed" ]] \
    || cannot_compute DEP002 "check-paths.sh --json payload contradicts its exit status $path_code" true

  # Merge path findings into our findings array. The read-back needs no guard of
  # its own: every element has been shown to carry a string level, rule and
  # message, so there is nothing here left to trip over.
  path_findings=$(echo "$path_json" | jq -c '.findings[]')
  if [[ -n "$path_findings" ]]; then
    while IFS= read -r pf; do
      level=$(echo "$pf" | jq -r '.level')
      rule=$(echo "$pf" | jq -r '.rule')
      message=$(echo "$pf" | jq -r '.message')
      findings+=("${level}|${rule}|${message}")
    done <<< "$path_findings"
  fi
else
  echo "$path_output"
fi

if [[ "$path_passed" == "false" && $fail -eq 0 ]]; then
  fail=1
fi

# --- Output ---
if $json_output; then
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
    echo "POLICY FAIL [$rule]: $message"
  done
fi

exit $fail
