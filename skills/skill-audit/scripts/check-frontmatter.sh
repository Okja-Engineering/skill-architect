#!/usr/bin/env bash
# Check frontmatter: spec validation via skill-validator, plus license policy gate.
# skill-validator handles: frontmatter format, name, description, YAML validity.
# We add: license requirement (house policy, not spec).
# Exit codes: 0=pass, 1=spec failure, 2=policy failure, 3=execution error.
# The policy failure carries rule ID PL001, a missing license.
# An execution error is one of three: a required tool is absent, which the
# rule registry calls DEP001; skill-validator returned a status this script
# cannot interpret, DEP002; or a verdict-guard.sh that would not load, reported
# before either ID exists to name it.
# Those IDs classify the exits; they are not printed. This script has no --json
# mode, so an exit 3 carries the guard's message on stderr — "required tool not
# found: skill-validator" — and no rule ID reaches the caller at all. The IDs
# above say which class an exit belongs to, for a reader of the registry; a
# consumer that needs to read one out of a payload wants --json, which only the
# two scripts that have it can give.
# Spec validation requires skill-validator.
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

if [[ $# -ne 1 ]]; then
  echo "Usage: check-frontmatter.sh <skill-dir>" >&2
  exit 3
fi

skill_dir="$1"
skill_md="$skill_dir/SKILL.md"

if [[ ! -f "$skill_md" ]]; then
  echo "ERROR: SKILL.md not found in $skill_dir" >&2
  exit 3
fi

# Run skill-validator for spec validation (frontmatter, name, description).
require_tool skill-validator false

code=0
output="$(skill-validator validate structure -o json "$skill_dir" 2>&1)" || code=$?

# skill-validator exit codes: 0=clean pass, 1=errors, 2=warnings only, 3=usage
# error. Anything outside that set is a status we cannot interpret, so it must
# not reach the license gate and be reported as a verdict.
case $code in
  0|2)
    # 0 = clean pass, 2 = warnings only (e.g. orphan files), not a spec failure.
    ;;
  1)
    echo "$output" | sed 's/^/SPEC FAIL: /'
    exit 1
    ;;
  3)
    cannot_compute DEP002 "skill-validator failed to run" false "$output"
    ;;
  *)
    cannot_compute DEP002 "skill-validator exited with unexpected status $code" false "$output"
    ;;
esac

# Check license as a policy gate (house rule, not spec).
frontmatter="$(awk '
  BEGIN { in_fm = 0 }
  /^---$/ {
    if (in_fm) { exit }
    in_fm = 1; next
  }
  in_fm { print }
' "$skill_md")"

if ! echo "$frontmatter" | grep -qE "^license:[[:space:]]"; then
  echo "POLICY FAIL [PL001]: missing license (house policy)"
  exit 2
fi

echo "frontmatter OK"
