#!/usr/bin/env bash
# Check structure: house-policy checks (headings, line limit, code blocks, lists)
# via grep, and local reference path validation via check-paths.sh.
# skill-validator handles spec compliance and link resolution separately;
# this script covers our repository-specific policy rules (PL002-PL005, PT001-PT002).
# Exit codes: 0=pass, 1=path failure, 2=policy failure, 3=execution error.
# Every rule here is a rule about the body, and is computed over the body: the
# frontmatter is metadata, not the content these rules judge.
# Findings carry rule IDs: PL002 a missing heading, PL003 a SKILL.md body over
# the line limit, PL004 no code blocks, PL005 no list items; DEP001 a required
# tool is absent; DEP002 a source returned a result this script cannot read —
# check-paths.sh answering with an unenumerated exit status, a payload that is
# not the documented shape, or a payload that contradicts itself or the status
# it arrived with; or a SKILL.md it could not read the body out of. In --json mode
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

# `dirname` runs before the guard exists to announce it, so it carries the same
# explicit refusal the load check below uses. Unread, its status was this
# script's own: a broken dirname left `cd` with nothing to enter and errexit
# exited 1, which is a status this contract spends on a verdict.
script_dir="$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)" \
  || { echo "ERROR: cannot resolve this script's own directory: dirname or cd gave no answer; no verdict was computed" >&2; exit 3; }
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

# Every tool this script computes with, stated as a precondition here rather
# than discovered at the call site that needed it. awk reads the body, grep
# answers every policy rule, and wc and tr count the lines; jq is asked for only
# in --json mode, where it is what builds the payload.
#
# They are preconditions and not just presence checks: a tool that is on PATH
# and does not work is the fault that reached a consumer, not a tool that is
# missing. require_tool asks each of them a question it knows the answer to.
require_tool awk "$json_output"
require_tool grep "$json_output"
require_tool wc "$json_output"
require_tool tr "$json_output"
if $json_output; then
  require_tool jq true
fi

fail=0
findings=()

# --- Policy checks (grep-based absence detection) ---
#
# Every rule below is a rule about the body, so every rule below reads the body
# and nothing else. Reading the whole file instead made the entire policy
# verdict satisfiable out of frontmatter: `## When to use` at column 0 inside
# the YAML is a comment to a parser and a heading to this grep, a code fence
# indented in a block scalar matches PL004's `^[[:space:]]*```, and the closing
# `---` matches PL005's `^[[:space:]]*[-*]`. A skill whose body was one prose
# sentence passed all four, and audit-report.sh reported it clean.
#
# The body is read once, here, rather than per rule: four reads of one file are
# four chances for the rules to disagree about what they are judging, and the
# tool that does the reading has to be answered for once either way.
body="$(skill_body "$skill_md")" \
  || cannot_compute DEP002 "could not read the body of $skill_md" "$json_output"

# Every rule below is an absence check, and an absence check is exactly where a
# tool fault becomes a finding: `! grep -q` reads an errored grep as "not
# there", and a command under `!` is exempt from errexit by definition, so
# nothing else catches it either. text_matches asks grep's status by name and
# reaches no verdict at all on anything outside {matched, did not match}.

# Required headings (case-insensitive) — match the original check-structure.sh patterns
for heading_spec in "When to use" ".*Process" "Examples?" "Deterministic" "Orchestration" "Constraints"; do
  if ! text_matches "$json_output" true "^#{2,6}[[:space:]]+${heading_spec}" "$body"; then
    findings+=("fail|PL002|missing heading matching '${heading_spec}'")
    fail=2
  fi
done

# Code blocks
if ! text_matches "$json_output" false '^[[:space:]]*```' "$body"; then
  findings+=("fail|PL004|no code blocks found")
  fail=2
fi

# List items
if ! text_matches "$json_output" false '^[[:space:]]*[-*]' "$body"; then
  findings+=("fail|PL005|no list items found")
  fail=2
fi

# Line count. PL003 is a limit on the body, as draft-rewrite.sh's own statement
# of the rule says ("`SKILL.md` body is under ~500 lines"): frontmatter is
# metadata a reader does not pay for, and counting it made the limit depend on
# how much of it a skill declared. An empty body is zero lines rather than the
# one a newline-terminated read of nothing would report.
#
# A count that is not a number is not a count. An empty or non-numeric result
# reads as 0 inside `[[ -gt ]]`, so a wc or tr that failed would silently retire
# PL003 rather than raise it, which is a verdict drawn from a source that said
# nothing.
line_count=0
if [[ -n "$body" ]]; then
  line_count="$(wc -l <<< "$body" | tr -d '[:space:]')" \
    || cannot_compute DEP002 "could not count the lines of $skill_md; no verdict was computed" "$json_output"
  [[ "$line_count" =~ ^[0-9]+$ ]] \
    || cannot_compute DEP002 "the line count of $skill_md came back as '$line_count', which is not a count" "$json_output"
fi
if [[ "$line_count" -gt 500 ]]; then
  findings+=("fail|PL003|SKILL.md body is $line_count lines (max 500)")
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
#
# The mirror of the rule this whole file is built on. A script must not report a
# verdict from a source it could not read; it must equally not *emit* a verdict
# it could not build. Both halves were open: this stdout is the payload channel
# and it carried nothing at exit 0 whenever the encoder that fills it answered
# with nothing, which reads to a consumer as a skill with no findings. So the
# payload is proven to be a payload by the same predicate its own consumers use,
# before it is printed, and a payload that will not build is the same DEP002 as
# a source that will not read.
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
  payload="$(echo "$json_findings" | jq --argjson passed "$([[ $fail -eq 0 ]] && echo true || echo false)" \
    '{findings: ., passed: $passed}')"
  payload_is_conforming "$payload" \
    || cannot_compute DEP002 "the findings payload could not be built; no verdict was computed" true
  printf '%s\n' "$payload"
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
