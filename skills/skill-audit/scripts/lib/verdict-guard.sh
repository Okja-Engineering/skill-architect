# Shared guards for the skill-audit check scripts.
#
# A check script must never report a verdict it could not compute. Whenever a
# tool it needs is missing, or a child check exits with a status it cannot
# interpret, it says so and exits 3 (execution error) instead of letting the
# failure read as a clean pass.
#
# Source this from a check script:
#
#   script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
#   source "$script_dir/lib/verdict-guard.sh"

# cannot_compute <rule> <message> <emit_json:true|false> [detail]
#
# Report that no verdict could be reached, and exit 3. In JSON mode the payload
# keeps the documented {"findings": [...], "passed": bool} shape so a machine
# consumer sees a failing verdict rather than a missing one, and carries the
# reason as an extra "error" field. Any detail — a failed tool's own output —
# follows the reason on stderr.
cannot_compute() {
  local rule="$1"
  local message="$2"
  local emit_json="${3:-false}"
  local detail="${4:-}"

  echo "ERROR: $message" >&2
  [[ -n "$detail" ]] && echo "$detail" >&2
  if [[ "$emit_json" == "true" ]]; then
    printf '{"findings": [{"level": "fail", "rule": "%s", "message": "%s"}], "passed": false, "error": "%s"}\n' \
      "$rule" "$message" "$message"
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
