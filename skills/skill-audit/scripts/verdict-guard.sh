# Shared guards for the skill-audit check scripts.
#
# A check script must never report a verdict it could not compute. Whenever a
# tool it needs is missing, or a child check exits with a status it cannot
# interpret, it says so and exits 3 (execution error) instead of letting the
# failure read as a clean pass.
#
# Source this from a check script, checking the load on both sides:
#
#   script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
#   verdict_guard="$script_dir/verdict-guard.sh"
#   bash -n "$verdict_guard" 2>/dev/null \
#     || { echo "ERROR: cannot load $verdict_guard: ..." >&2; exit 3; }
#   source "$verdict_guard"
#   declare -F cannot_compute >/dev/null && declare -F require_tool >/dev/null \
#     || { echo "ERROR: $verdict_guard defines no guards: ..." >&2; exit 3; }
#
# This file is the one dependency require_tool cannot announce: if it is not
# there, nothing is there to do the announcing. An unchecked `source` of a
# missing file aborts the calling script with status 1, and of a malformed one
# with status 2 — statuses every caller's contract reserves for a spec, path or
# policy verdict it actually computed. A file that loads but defines nothing is
# worse still: the caller runs to completion and only discovers the guard is
# gone on the path where it needed it. So the caller syntax-checks the file
# before sourcing it and confirms the guards exist afterwards, and reports
# either failure the way it reports every other unmet precondition — exit 3, no
# verdict. That exit carries no payload: the arguments have not been parsed yet,
# so the caller does not know whether a payload was even asked for, and the
# thing that builds payloads is the thing that is missing.

# json_string <text>
#
# Echo <text> as a JSON string literal, surrounding quotes included.
#
# The guard encodes in the shell rather than shelling out to jq, because a
# missing jq is one of the things it is called to report: the payload has to be
# buildable with the encoder absent. Interpolating the text instead would make
# the documented shape conditional on every caller passing a string with no
# quote, backslash or control character — a convention no caller is checked
# against, and one a future caller relaying a tool's own output would break.
json_string() {
  local s="$1"
  local out='"'
  local i c ord
  for (( i = 0; i < ${#s}; i++ )); do
    c="${s:i:1}"
    case "$c" in
      '"')   out+='\"' ;;
      '\')   out+='\\' ;;
      $'\n') out+='\n' ;;
      $'\r') out+='\r' ;;
      $'\t') out+='\t' ;;
      $'\b') out+='\b' ;;
      $'\f') out+='\f' ;;
      *)
        printf -v ord '%d' "'$c"
        if (( ord < 32 )); then
          printf -v c '\\u%04x' "$ord"
        fi
        out+="$c"
        ;;
    esac
  done
  printf '%s"\n' "$out"
}

# cannot_compute <rule> <message> <emit_json:true|false> [detail]
#
# Report that no verdict could be reached, and exit 3. In JSON mode the payload
# keeps the documented {"findings": [...], "passed": bool} shape so a machine
# consumer sees a failing verdict rather than a missing one, and carries the
# reason as an extra "error" field. Any detail — a failed tool's own output —
# follows the reason on stderr, never on stdout, which in JSON mode is the
# payload channel.
cannot_compute() {
  local rule="$1"
  local message="$2"
  local emit_json="${3:-false}"
  local detail="${4:-}"

  echo "ERROR: $message" >&2
  [[ -n "$detail" ]] && echo "$detail" >&2
  if [[ "$emit_json" == "true" ]]; then
    printf '{"findings": [{"level": "fail", "rule": %s, "message": %s}], "passed": false, "error": %s}\n' \
      "$(json_string "$rule")" "$(json_string "$message")" "$(json_string "$message")"
  fi
  exit 3
}

# require_tool <tool> <emit_json:true|false>
#
# A stated precondition, checked once where the dependency is established rather
# than at each call site — so the requirement never depends on what the audited
# skill happens to contain.
require_tool() {
  local tool="$1"
  local emit_json="${2:-false}"

  command -v "$tool" >/dev/null 2>&1 && return 0

  cannot_compute DEP001 "required tool not found: $tool" "$emit_json"
}
