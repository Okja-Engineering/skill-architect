#!/usr/bin/env bash
# Check frontmatter: spec validation via skill-validator, plus license policy gate.
# skill-validator handles: frontmatter format, name, description, YAML validity.
# We add: license requirement (house policy, not spec).
# Exit codes: 0=pass, 1=spec failure, 2=policy failure, 3=execution error.
set -euo pipefail

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
# skill-validator exit codes: 0=clean pass, 1=errors, 2=warnings only, 3=usage error.
code=0
output="$(skill-validator validate structure -o json "$skill_dir" 2>&1)" || code=$?

if [[ $code -eq 3 ]]; then
  echo "ERROR: skill-validator failed to run" >&2
  echo "$output" >&2
  exit 3
fi

if [[ $code -eq 1 ]]; then
  echo "$output" | sed 's/^/SPEC FAIL: /'
  exit 1
fi

# Exit code 2 = warnings only (e.g. orphan files), not a spec failure.

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
