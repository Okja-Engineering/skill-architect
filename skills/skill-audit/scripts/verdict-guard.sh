# Shared guards for the skill-audit check scripts.
#
# A check script must never report a verdict it could not compute. Whenever a
# tool it needs is missing, or a child check exits with a status it cannot
# interpret, it says so and exits 3 (execution error) instead of letting the
# failure read as a clean pass.
#
# The rule ID this file emits directly is DEP001, a required tool is absent.
# Its callers pass it DEP002 for a child result they could not interpret.
#
# Source this from a check script, checking the load on both sides:
#
#   script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
#   verdict_guard="$script_dir/verdict-guard.sh"
#   bash -n "$verdict_guard" 2>/dev/null \
#     || { echo "ERROR: cannot load $verdict_guard: ..." >&2; exit 3; }
#   source "$verdict_guard"
#   { declare -F verdict_guard_ready >/dev/null && verdict_guard_ready; } \
#     || { echo "ERROR: $verdict_guard did not load its guards: ..." >&2; exit 3; }
#
# What the caller confirms afterwards is verdict_guard_ready, not a list of the
# guards it happens to use. "The guard loaded" is not one proposition: a file
# that stopped short defines some of them and not others, and a caller checking
# only the names it remembered runs on until `command not found` at the point it
# needed the one that is gone. The list lives here, once, beside the definitions
# it covers.
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
# JSON asks for three things to be escaped: the quote, the backslash, and the
# control characters. Everything else is passed through exactly as it arrived —
# every byte of a multi-byte character included, which is why the remaining arm
# matches a character class instead of reading the character's ordinal. bash
# yields a *negative* ordinal for any byte at or above 0x80, so an ordinal test
# sent every byte of a UTF-8 character down the control-character path and
# emitted ￿ffffffffffc3 for it. The payload still parsed, so the shape
# promise held while the message it carried no longer said what it was given —
# and carrying the message is the reason this encoder exists at all.
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
      [[:cntrl:]])
        printf -v ord '%d' "'$c"
        printf -v c '\\u%04x' "$ord"
        out+="$c"
        ;;
      *) out+="$c" ;;
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

# payload_is_conforming <text>
#
# Succeed when <text> is one findings payload in the shape these scripts
# document: an object with a boolean `passed` and an array `findings` whose
# every element is an object carrying a string `level`, `rule` and `message`.
#
# Shape is a question about the payload alone, so it is asked about the payload
# alone — before any verdict is read out of it, and whatever that verdict turns
# out to be. Asking it in the same expression that reads the verdict is what
# broke: `and` short-circuits, so the shape half ran only on the branch the
# verdict took, and a payload taking the other branch reached a reader assuming
# a shape nobody had checked. Every read a caller makes after this returns 0 is
# total; there is nothing left for it to trip over.
#
# The shape is proven all the way down to the elements, because that is how far
# the callers read. Proving the top level and then indexing the elements is the
# same defect one level in.
#
# `-s` is what makes "one payload" part of the claim: a stream of documents
# slurps to an array longer than one and is refused, whichever position a
# conforming document holds in it, and no input at all slurps to an empty array.
# The expression itself is total rather than short-circuiting — each `if`
# settles a type before anything indexes through it — so a wrong type answers
# false where it would otherwise raise an error. The two `type != "object"`
# gates are not observable from outside: without them jq raises instead, and a
# raise is already read here as "not the documented shape", so no test can tell
# the two apart. They stay because the point of this predicate is to decide the
# question rather than to survive being wrong about it — an expression whose
# answer depends on which branch happens to be evaluated is the exact defect it
# was written to close.
#
# It answers with jq, so it is callable only where jq is already a proven
# precondition. Both callers require jq before reading any payload.
payload_is_conforming() {
  printf '%s' "$1" | jq -se '
    length == 1 and (.[0] |
      if type != "object" then false
      elif (.passed | type) != "boolean" then false
      elif (.findings | type) != "array" then false
      else [.findings[] |
        if type != "object" then false
        else (.level | type) == "string"
             and (.rule | type) == "string"
             and (.message | type) == "string"
        end] | all
      end)' >/dev/null 2>&1
}

# verdict_guard_ready
#
# Succeed when every guard this file exists to provide is defined. This is the
# one question a caller asks after sourcing, so the set lives here rather than
# being re-listed at every call site — where it would be re-listed incompletely,
# and each omission would be a guard whose absence nothing catches.
#
# Defined last on purpose: a file that did not reach the end does not define
# this either, so "stopped short" fails by the same route as "never had it".
verdict_guard_ready() {
  local g
  for g in json_string cannot_compute require_tool payload_is_conforming; do
    declare -F "$g" >/dev/null 2>&1 || return 1
  done
  return 0
}
