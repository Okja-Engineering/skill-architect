#!/usr/bin/env bash
# Check local file references in a skill directory.
# Finds executable script references (./ or $ prefixed) in code blocks
# and markdown links, then resolves them against the filesystem.
# Exit codes: 0=pass, 1=path failure, 3=execution error.
# Findings carry rule IDs: at level fail, PT001 missing script/reference and
# PT002 missing markdown link, plus DEP001 when a required tool is absent and
# DEP002 when a source this script has to read gave it no answer it could use —
# a SKILL.md it could not read the body or the code blocks out of, a tool that
# is present and does not work, or a payload it could not build; at level unverified, PATH for a
# reference built from a glob or a variable, which cannot be resolved and so is
# reported without being judged — an unverified finding is not a failure and
# does not change the exit status.
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

# Every tool this script computes with, stated as a precondition rather than
# discovered at the call site that needed it: awk reads the body and the code
# blocks, grep finds and classifies the references. jq is asked for only in
# --json mode, where it is what builds the payload — text mode reaches its
# verdict without it and is unaffected by its absence.
#
# sed and tr used to be on this list. They are not here because they are no
# longer used: a markdown link's target and a quoted path are now taken apart
# with parameter expansion, which is the same two operations with no tool to
# require, no status to interpret and nothing to go wrong.
require_tool awk "$json_output"
require_tool grep "$json_output"
if $json_output; then
  require_tool jq true
fi

fail=0
findings=()

# The body, read through the shared primitive. This was a private toggle that
# re-entered frontmatter on a third `---`, so one markdown horizontal rule ended
# every check below it and a skill with broken references reported none.
body="$(skill_body "$skill_md")" \
  || cannot_compute DEP002 "could not read the body of $skill_md" "$json_output"

# Both reference sweeps read their input through text_extract, and the reason is
# the shape they used to have: the extraction sat inside the process
# substitution feeding the loop. A subshell is where a guard cannot reach —
# whatever grep answered there, the loop read the result and iterated over it,
# so a grep that could not read the source produced zero references and a clean
# pass. One of the two even said `|| true` out loud. text_extract runs in this
# shell, distinguishes "nothing matched" from "could not answer", and exits 3
# on the second.

# Check markdown links [text](path) — skip http/https/anchor/mailto
#
# The target is taken out of `[text](target)` by parameter expansion rather than
# by sed. grep's pattern already refuses a `]` inside the text, so the first
# `](` is the only one there is, and cutting at it needs no tool.
text_extract "$json_output" '\[[^]]*\]\([^)]+\)' "$body"
while IFS= read -r link; do
  [[ -z "$link" ]] && continue
  path="${link#*](}"
  path="${path%)}"
  [[ -z "$path" ]] && continue
  [[ "$path" =~ ^https?:// ]] && continue
  [[ "$path" =~ ^# ]] && continue
  [[ "$path" =~ ^mailto: ]] && continue
  if text_matches "$json_output" false '[*?]|\$' "$path"; then
    findings+=("unverified|PATH|dynamic/glob link: $path")
    continue
  fi
  if [[ ! -e "$skill_dir/$path" ]]; then
    findings+=("fail|PT002|markdown link target not found: $path")
    fail=1
  fi
done <<< "$extracted"

# Check executable script references in code blocks (./ or $ prefixed)
# Extract code block content, then find path references
#
# This awk's status was discarded, and it is reachable: a body carrying bytes
# that are not valid in the current locale makes awk exit 2, which under errexit
# became this script's own exit status — a 2 its contract does not enumerate,
# with nothing on the payload channel, read by check-structure.sh as a child
# that reached no verdict.
code_body="$(awk '
  /^```/ { in_code = !in_code; next }
  in_code { print }
' <<< "$body")" \
  || cannot_compute DEP002 "could not read the code blocks of $skill_md; no verdict was computed" "$json_output"

text_extract "$json_output" '(\./|\$)\S*(scripts|references|assets)/\S+' "$code_body"
while IFS= read -r path; do
  [[ -z "$path" ]] && continue
  # Strip the quotes a path picks up from the command line it was written on.
  path="${path//[\"\']/}"
  if text_matches "$json_output" false '[*?]|\$' "$path"; then
    findings+=("unverified|PATH|dynamic/glob path: $path")
    continue
  fi
  if [[ ! -e "$skill_dir/$path" ]]; then
    findings+=("fail|PT001|script/reference path not found: $path")
    fail=1
  fi
done <<< "$extracted"

# --- Output ---
#
# The mirror of the rule this script is built on. It must not report a verdict
# from a source it could not read, and it must not emit a verdict it could not
# build: stdout here is the payload channel, and it carried nothing at exit 0
# whenever the encoder that fills it answered with nothing — which a consumer
# reads as a skill with no findings. So the payload is proven to be a payload,
# by the predicate its own consumer uses, before it is printed.
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
    if [[ "$level" == "fail" ]]; then
      echo "PATH FAIL [$rule]: $message"
    else
      echo "UNVERIFIED [$rule]: $message"
    fi
  done
fi

exit $fail
