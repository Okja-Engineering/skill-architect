#!/usr/bin/env bash
# Check frontmatter: spec validation via skill-validator, plus license policy gate.
# skill-validator handles: frontmatter format, name, description, YAML validity.
# We add: license requirement (house policy, not spec).
# Exit codes: 0=pass, 1=spec failure, 2=policy failure, 3=execution error.
# Spec validation requires skill-validator.
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/verdict-guard.sh
source "$script_dir/lib/verdict-guard.sh"

if [[ $# -ne 1 ]]; then
  echo "Usage: check-frontmatter.sh <skill-dir>" >&2
  exit 3
fi

skill_dir="$1"
skill_md="$skill_dir/SKILL.md"

if [[ ! -f "$skill_md" ]]; then
  echo "FAIL: SKILL.md not found in $skill_dir"
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
