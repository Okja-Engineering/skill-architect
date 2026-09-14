#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

pass=0
fail=0
summary_printed=false
tmp_skill=""

# Every check in this suite runs through `assert`, which takes a *command* and
# runs it. An earlier shape took a value, so a caller wrote the real check as a
# bare `[[ ]]` on the preceding line and passed the literal `true` — and bash
# 3.2 does not apply `errexit` to `[[ ]]`, so on the bash this project supports
# a failing check printed PASS. Taking a command removes that shape: there is no
# value to hand in, and a check that fails is counted and reported.
assert() {
  local label="$1"
  shift
  if "$@"; then
    echo "PASS: $label"
    pass=$((pass + 1))
  else
    echo "FAIL: $label"
    fail=$((fail + 1))
  fi
}

# Run a command silently while it succeeds, and surface everything it said when
# it fails. A passing check should add no noise; a failing one must say why.
quietly() {
  local output
  local status=0
  output="$("$@" 2>&1)" || status=$?
  if [[ $status -ne 0 ]]; then
    printf '%s\n' "$output" >&2
  fi
  return $status
}

# A suite that dies before its summary has reported nothing, whatever its exit
# status says. Both failure modes were real here: bash 3.2 ran past a failed
# check and bash 5 aborted on it, and neither printed a verdict. This makes an
# abort loud and never silently successful.
cleanup() {
  local code=$?
  if [[ -n "$tmp_skill" ]]; then
    rm -rf "$tmp_skill"
  fi
  if [[ "$summary_printed" != true ]]; then
    echo
    echo "FAIL: suite aborted before reaching its summary (exit $code)"
    if [[ $code -eq 0 ]]; then
      code=1
    fi
  fi
  exit "$code"
}
trap cleanup EXIT

# The scratch directory belongs to the shell that installed the trap. A helper
# that made its own could be running inside a command substitution, where the
# assignment never escapes and the trap has nothing to clean up.
tmp_skill="$(mktemp -d)"

manifest_is_valid() {
  python3 - "$1" <<'PY'
import json, sys
with open(sys.argv[1]) as f:
    data = json.load(f)
assert data['name'] == 'skill-architect', 'name mismatch'
assert data['version'] == '0.4.2', 'version mismatch'
assert 'skills' in data, 'missing skills'
PY
}

marketplace_is_valid() {
  python3 - <<'PY'
import json
with open('.claude-plugin/marketplace.json') as f:
    data = json.load(f)
assert data['name'] == 'skill-architect', 'marketplace name mismatch'
plugins = data['plugins']
assert len(plugins) == 1, 'expected exactly one plugin entry'
assert plugins[0]['name'] == 'skill-architect', 'plugin entry name mismatch'
assert plugins[0]['version'] == '0.4.2', 'plugin entry version mismatch'
assert plugins[0]['source'] == './', 'plugin entry source mismatch'
PY
}

frontmatter_name_matches_directory() {
  local skill="$1"
  local name_in_file
  name_in_file="$(awk '
    BEGIN { in_fm = 0 }
    /^---$/ {
      if (in_fm) { exit }
      in_fm = 1
      next
    }
    in_fm && /^name:/ { sub(/^name:[[:space:]]*/, ""); sub(/[[:space:]]*$/, ""); print; exit }
  ' "skills/$skill/SKILL.md")" || return 1
  [[ "$name_in_file" == "$skill" ]]
}

draft_rewrite_produces_draft() {
  local work="$tmp_skill/skill-rewrite-test"
  cp -R skills/skill-rewrite "$work" || return 1
  skills/skill-rewrite/scripts/draft-rewrite.sh -t "$work" >/dev/null || return 1
  [[ -f "$work/REWRITE-DRAFT.md" ]]
}

# Plugin manifests are valid JSON and version matches.
for manifest in .devin-plugin/plugin.json .claude-plugin/plugin.json .cursor-plugin/plugin.json .codex-plugin/plugin.json; do
  assert "$manifest exists" test -f "$manifest"
  assert "$manifest is valid plugin.json" quietly manifest_is_valid "$manifest"
done

# The marketplace manifest is the fifth version surface. `claude plugin
# marketplace add` reads it, and it carries its own copy of the version, so a
# release that bumps the four plugin.json files and forgets this one advertises
# the previous release to anyone installing by name.
assert ".claude-plugin/marketplace.json exists" test -f .claude-plugin/marketplace.json
assert ".claude-plugin/marketplace.json is valid and at 0.4.2" quietly marketplace_is_valid

# The profiler records its own version in every profile it writes, and it is the
# sixth surface carrying this release's number. It is asserted here beside the
# manifests so one place shows all of them, and in Go by
# TestAdapterVersionIsThisRelease.
assert "profiler AdapterVersion is 0.4.2" \
  grep -q 'AdapterVersion = "0.4.2"' profiler/types.go

# Each skill has a valid SKILL.md with frontmatter and name matching directory.
for skill in skill-audit skill-rewrite; do
  assert "skills/$skill directory exists" test -d "skills/$skill"
  assert "skills/$skill/SKILL.md exists" test -f "skills/$skill/SKILL.md"
  assert "skills/$skill frontmatter name matches directory" \
    frontmatter_name_matches_directory "$skill"
done

# skill-audit carries its scripts and references.
assert "skill-audit check-frontmatter.sh is executable" \
  test -x skills/skill-audit/scripts/check-frontmatter.sh
assert "skill-audit check-structure.sh is executable" \
  test -x skills/skill-audit/scripts/check-structure.sh
assert "skill-audit best-practices reference exists" \
  test -f skills/skill-audit/references/best-practices.md
assert "skill-audit evaluation-matrix reference exists" \
  test -f skills/skill-audit/references/evaluation-matrix.md

# The skill-audit scripts can audit themselves.
assert "skill-audit passes its own frontmatter check" \
  quietly skills/skill-audit/scripts/check-frontmatter.sh skills/skill-audit
assert "skill-audit passes its own structure check" \
  quietly skills/skill-audit/scripts/check-structure.sh skills/skill-audit

# The shipped skills validate clean — warnings included.
#
# skill-audit is the repo's reference skill: it is what the audit skill points
# other skills at, and it invokes this very validator. A warning in our own
# artifact is a defect. Nothing checked for one before, which is how a
# `scripts/lib/` directory earned a "deep nesting detected" warning and carried
# it through every gate unnoticed.
#
# `skill-validator check` exits 0 only on a clean pass; 1 means errors and 2
# means warnings only. Asserting 0 pins both counts at zero.
for skill in skills/skill-audit skills/skill-rewrite; do
  assert "$skill validates with zero errors and zero warnings" \
    quietly skill-validator check "$skill"
done

# skill-rewrite carries its script and can draft a rewrite for itself.
assert "skill-rewrite draft-rewrite.sh is executable" \
  test -x skills/skill-rewrite/scripts/draft-rewrite.sh
assert "skill-rewrite draft-rewrite.sh produces REWRITE-DRAFT.md" \
  quietly draft_rewrite_produces_draft

echo
echo "$pass passed, $fail failed"
summary_printed=true
if [[ "$fail" -gt 0 ]]; then
  exit 1
fi
