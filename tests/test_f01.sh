#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

pass=0
fail=0

assert() {
  local label="$1"
  local condition="$2"
  if [[ "$condition" == "true" ]]; then
    echo "PASS: $label"
    pass=$((pass + 1))
  else
    echo "FAIL: $label"
    fail=$((fail + 1))
  fi
}

# Locate tools.
SV="skill-validator"
CHECK_FM="skills/skill-audit/scripts/check-frontmatter.sh"
CHECK_STRUCT="skills/skill-audit/scripts/check-structure.sh"
CHECK_PATHS="skills/skill-audit/scripts/check-paths.sh"
CHECK_QUALITY="skills/skill-audit/scripts/check-quality.sh"

# Helper: run a command and capture output + exit code.
run() {
  local cmd="$1"
  shift
  code=0
  output=$("$cmd" "$@" 2>&1) || code=$?
}

# Helper: run skill-validator validate structure and capture output + exit code.
run_sv() {
  local dir="$1"
  code=0
  output=$("$SV" validate structure -o json "$dir" 2>&1) || code=$?
}

# --- Spec validation (skill-validator validate structure) ---

# valid-minimal: spec passes (name + description only)
run_sv tests/fixtures/f01/valid-minimal
assert "valid-minimal: spec passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"

# valid-full: spec passes
run_sv tests/fixtures/f01/valid-full
assert "valid-full: spec passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"

# malformed-yaml: spec fails
run_sv tests/fixtures/f01/malformed-yaml
assert "malformed-yaml: spec fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert "malformed-yaml: reports YAML error" "$(echo "$output" | grep -qi 'yaml\|parse' && echo true || echo false)"

# invalid-name-format: spec fails
run_sv tests/fixtures/f01/invalid-name-format
assert "invalid-name-format: spec fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert "invalid-name-format: reports name error" "$(echo "$output" | grep -qi 'name' && echo true || echo false)"

# name-mismatch: spec fails
run_sv tests/fixtures/f01/name-mismatch
assert "name-mismatch: spec fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert "name-mismatch: reports match error" "$(echo "$output" | grep -qi 'match\|directory' && echo true || echo false)"

# missing-description: spec fails
run_sv tests/fixtures/f01/missing-description
assert "missing-description: spec fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert "missing-description: reports description missing" "$(echo "$output" | grep -qi 'description' && echo true || echo false)"

# description-too-long: spec fails
run_sv tests/fixtures/f01/description-too-long
assert "description-too-long: spec fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert "description-too-long: reports exceeds limit" "$(echo "$output" | grep -qi 'exceeds\|1024' && echo true || echo false)"

# no-license: spec passes (license is optional per spec)
run_sv tests/fixtures/f01/no-license
assert "no-license: spec passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"

# --- Policy validation (check-frontmatter.sh for license, check-structure.sh for headings) ---

# valid-minimal: policy fails (no license)
run "$CHECK_FM" tests/fixtures/f01/valid-minimal
assert "valid-minimal: frontmatter fails (exit 2)" "$([[ $code -eq 2 ]] && echo true || echo false)"
assert "valid-minimal: reports PL001" "$(echo "$output" | grep -q 'PL001' && echo true || echo false)"

# valid-full: frontmatter passes
run "$CHECK_FM" tests/fixtures/f01/valid-full
assert "valid-full: frontmatter passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"

# no-license: spec passes, policy fails
run "$CHECK_FM" tests/fixtures/f01/no-license
assert "no-license: frontmatter fails (exit 2)" "$([[ $code -eq 2 ]] && echo true || echo false)"
assert "no-license: reports PL001" "$(echo "$output" | grep -q 'PL001' && echo true || echo false)"

# --- Structure/policy checks (check-structure.sh) ---

# valid-full: structure passes
run "$CHECK_STRUCT" tests/fixtures/f01/valid-full
assert "valid-full: structure passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"

# --- Path validation (check-paths.sh) ---

# missing-script-ref: path check fails
run "$CHECK_PATHS" tests/fixtures/f01/missing-script-ref
assert "missing-script-ref: paths fail (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert "missing-script-ref: reports PT001" "$(echo "$output" | grep -q 'PT001' && echo true || echo false)"

# missing-markdown-ref: path check fails
run "$CHECK_PATHS" tests/fixtures/f01/missing-markdown-ref
assert "missing-markdown-ref: paths fail (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert "missing-markdown-ref: reports PT002" "$(echo "$output" | grep -q 'PT002' && echo true || echo false)"

# dynamic-paths: path check reports unverified, not failure
run "$CHECK_PATHS" tests/fixtures/f01/dynamic-paths
assert "dynamic-paths: paths pass (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert "dynamic-paths: reports UNVERIFIED" "$(echo "$output" | grep -qi 'unverified' && echo true || echo false)"

# glob-paths: path check reports unverified, not failure
run "$CHECK_PATHS" tests/fixtures/f01/glob-paths
assert "glob-paths: paths pass (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert "glob-paths: reports UNVERIFIED" "$(echo "$output" | grep -qi 'unverified' && echo true || echo false)"

# --- Quality scoring (check-quality.sh / skillscore) ---

# valid-full: quality scoring produces JSON with categories
run "$CHECK_QUALITY" tests/fixtures/f01/valid-full
assert "valid-full: quality scoring succeeds (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert "valid-full: quality JSON has categories" "$(echo "$output" | grep -q 'categories' && echo true || echo false)"

# --- Bash entrypoint backward compatibility ---

# check-frontmatter.sh on valid-full should pass
run "$CHECK_FM" tests/fixtures/f01/valid-full
assert "check-frontmatter.sh valid-full passes" "$([[ $code -eq 0 ]] && echo true || echo false)"

# check-frontmatter.sh on no-license should fail (policy)
run "$CHECK_FM" tests/fixtures/f01/no-license
assert "check-frontmatter.sh no-license fails" "$([[ $code -ne 0 ]] && echo true || echo false)"

# check-structure.sh on valid-full should pass
run "$CHECK_STRUCT" tests/fixtures/f01/valid-full
assert "check-structure.sh valid-full passes" "$([[ $code -eq 0 ]] && echo true || echo false)"

# --- Dependency guards: never report a verdict a missing tool could not compute ---
#
# Invariant under test: when a tool a script needs to reach its verdict is absent,
# the script exits non-zero, says which tool is missing, and never emits a passing
# verdict — in the exit status and in the --json payload alike. Unenumerated child
# exit codes are execution errors, not passes.
#
# Masked PATHs are symlink farms of the real PATH minus one binary. Nothing is
# deleted, moved or uninstalled.

mask_root="$(mktemp -d)"
trap 'rm -rf "$mask_root"' EXIT

# A PATH holding every binary the real PATH offers except one. Cached per tool.
masked_path() {
  local hide="$1"
  local farm="$mask_root/without-$hide"
  if [[ -d "$farm" ]]; then
    echo "$farm"
    return 0
  fi
  mkdir -p "$farm"
  local dirs d f b
  IFS=: read -ra dirs <<< "$PATH"
  for d in "${dirs[@]}"; do
    [[ -d "$d" ]] || continue
    for f in "$d"/*; do
      b="${f##*/}"
      [[ "$b" == "$hide" ]] && continue
      [[ -e "$farm/$b" ]] && continue
      ln -s "$f" "$farm/$b" 2>/dev/null || true
    done
  done
  echo "$farm"
}

# A PATH whose first entry provides a stub tool exiting with a chosen code.
stub_tool_path() {
  local tool="$1"
  local exit_code="$2"
  local dir="$mask_root/stub-$tool-$exit_code"
  if [[ ! -d "$dir" ]]; then
    mkdir -p "$dir"
    printf '#!/usr/bin/env bash\necho "stub %s output"\nexit %s\n' "$tool" "$exit_code" > "$dir/$tool"
    chmod +x "$dir/$tool"
  fi
  echo "$dir:$PATH"
}

# A copy of the scripts directory whose check-paths.sh exits with a chosen code.
# Echoes the path to the copied check-structure.sh.
stub_paths_tree() {
  local exit_code="$1"
  local dir="$mask_root/tree-$exit_code"
  if [[ ! -d "$dir" ]]; then
    cp -R skills/skill-audit/scripts "$dir"
    printf '#!/usr/bin/env bash\nexit %s\n' "$exit_code" > "$dir/check-paths.sh"
    chmod +x "$dir/check-paths.sh"
  fi
  echo "$dir/check-structure.sh"
}

# Run a command under a given PATH, capturing stdout and stderr separately so a
# --json payload can be parsed without the diagnostics mixed into it.
run_on_path() {
  local use_path="$1"
  shift
  local errfile="$mask_root/stderr"
  code=0
  output=$(PATH="$use_path" "$@" 2>"$errfile") || code=$?
  errout="$(cat "$errfile")"
}

run_masked() {
  local hide="$1"
  shift
  run_on_path "$(masked_path "$hide")" "$@"
}

run_present() {
  run_on_path "$PATH" "$@"
}

PATH_FAULT=tests/fixtures/f01/path-fault-only

# --- The path-fault-only fixture: preconditions the S1 cases depend on ---
# If this fixture ever grew a policy fault, the S1 cases below would stop testing
# S1 (a policy finding never round-trips through jq), and they would still pass.

run_present "$CHECK_STRUCT" --json "$PATH_FAULT"
assert "path-fault-only: structure --json fails with jq present (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert "path-fault-only: structure --json reports passed false with jq present" "$([[ "$(echo "$output" | jq -r '.passed')" == "false" ]] && echo true || echo false)"
assert "path-fault-only: has a PT001 finding" "$(echo "$output" | jq -e '.findings[] | select(.rule == "PT001")' >/dev/null 2>&1 && echo true || echo false)"
assert "path-fault-only: has no PL policy finding (keeps S1 under test)" "$([[ "$(echo "$output" | jq -r '[.findings[] | select(.rule | startswith("PL"))] | length')" -eq 0 ]] && echo true || echo false)"

# --- S1: check-structure.sh --json must not pass a failing skill when jq is gone ---

run_masked jq "$CHECK_STRUCT" --json "$PATH_FAULT"
assert "S1 structure --json, jq masked: exits non-zero" "$([[ $code -ne 0 ]] && echo true || echo false)"
assert "S1 structure --json, jq masked: exits 3 (execution error)" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert "S1 structure --json, jq masked: names the missing tool" "$(echo "$errout" | grep -q 'jq' && echo true || echo false)"
assert "S1 structure --json, jq masked: never reports passed true" "$(echo "$output" | grep -qE '"passed":[[:space:]]*true' && echo false || echo true)"
assert "S1 structure --json, jq masked: payload is valid JSON" "$(echo "$output" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
assert "S1 structure --json, jq masked: payload passed is false" "$([[ "$(echo "$output" | jq -r '.passed')" == "false" ]] && echo true || echo false)"
assert "S1 structure --json, jq masked: payload carries DEP001 naming jq" "$(echo "$output" | jq -e '.findings[] | select(.rule == "DEP001") | select(.message | test("jq"))' >/dev/null 2>&1 && echo true || echo false)"

# --- --json requires jq unconditionally, including on a clean skill ---
# The requirement is a stated precondition, not a function of what the skill
# happens to contain; a data-dependent dependency is what let S1 hide.

run_masked jq "$CHECK_STRUCT" --json tests/fixtures/f01/valid-full
assert "structure --json, jq masked, clean skill: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert "structure --json, jq masked, clean skill: names the missing tool" "$(echo "$errout" | grep -q 'jq' && echo true || echo false)"
assert "structure --json, jq masked, clean skill: never reports passed true" "$(echo "$output" | grep -qE '"passed":[[:space:]]*true' && echo false || echo true)"

run_masked jq "$CHECK_PATHS" --json tests/fixtures/f01/valid-full
assert "paths --json, jq masked, clean skill: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert "paths --json, jq masked, clean skill: names the missing tool" "$(echo "$errout" | grep -q 'jq' && echo true || echo false)"
assert "paths --json, jq masked, clean skill: never reports passed true" "$(echo "$output" | grep -qE '"passed":[[:space:]]*true' && echo false || echo true)"

# --- check-paths.sh --json under the same missing tool ---

run_masked jq "$CHECK_PATHS" --json "$PATH_FAULT"
assert "paths --json, jq masked: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert "paths --json, jq masked: names the missing tool" "$(echo "$errout" | grep -q 'jq' && echo true || echo false)"
assert "paths --json, jq masked: payload passed is false" "$([[ "$(echo "$output" | jq -r '.passed')" == "false" ]] && echo true || echo false)"
assert "paths --json, jq masked: payload carries DEP001 naming jq" "$(echo "$output" | jq -e '.findings[] | select(.rule == "DEP001") | select(.message | test("jq"))' >/dev/null 2>&1 && echo true || echo false)"

# --- Text mode needs no jq and is unaffected by its absence ---

run_masked jq "$CHECK_PATHS" "$PATH_FAULT"
assert "paths text, jq masked: still fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert "paths text, jq masked: still reports PT001" "$(echo "$output" | grep -q 'PT001' && echo true || echo false)"

run_masked jq "$CHECK_STRUCT" "$PATH_FAULT"
assert "structure text, jq masked: still fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert "structure text, jq masked: still reports PT001" "$(echo "$output" | grep -q 'PT001' && echo true || echo false)"

run_masked jq "$CHECK_PATHS" tests/fixtures/f01/valid-full
assert "paths text, jq masked, clean skill: still passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"

run_masked jq "$CHECK_STRUCT" tests/fixtures/f01/valid-full
assert "structure text, jq masked, clean skill: still passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"

# --- S2: check-frontmatter.sh must not pass when skill-validator is gone ---

run_masked skill-validator "$CHECK_FM" tests/fixtures/f01/valid-full
assert "S2 frontmatter, validator masked: exits non-zero" "$([[ $code -ne 0 ]] && echo true || echo false)"
assert "S2 frontmatter, validator masked: exits 3 (execution error)" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert "S2 frontmatter, validator masked: names the missing tool" "$(echo "$errout" | grep -q 'skill-validator' && echo true || echo false)"
assert "S2 frontmatter, validator masked: never prints frontmatter OK" "$(echo "$output" | grep -q 'frontmatter OK' && echo false || echo true)"

# A missing validator must not reclassify a spec failure as a policy failure.
run_masked skill-validator "$CHECK_FM" tests/fixtures/f01/malformed-yaml
assert "S2 frontmatter, validator masked, broken spec: exits 3 not 2" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert "S2 frontmatter, validator masked, broken spec: does not report PL001" "$(echo "$output" | grep -q 'PL001' && echo false || echo true)"

# Controls: with the validator present, behaviour is unchanged.
run_present "$CHECK_FM" tests/fixtures/f01/valid-full
assert "frontmatter, validator present: valid-full passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert "frontmatter, validator present: valid-full prints frontmatter OK" "$(echo "$output" | grep -q 'frontmatter OK' && echo true || echo false)"

run_present "$CHECK_FM" tests/fixtures/f01/malformed-yaml
assert "frontmatter, validator present: malformed-yaml fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert "frontmatter, validator present: malformed-yaml reports SPEC FAIL" "$(echo "$output" | grep -q 'SPEC FAIL' && echo true || echo false)"

# --- Unenumerated child exit codes are execution errors, not passes ---

run_on_path "$(stub_tool_path skill-validator 42)" "$CHECK_FM" tests/fixtures/f01/valid-full
assert "frontmatter, validator exits 42: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert "frontmatter, validator exits 42: never prints frontmatter OK" "$(echo "$output" | grep -q 'frontmatter OK' && echo false || echo true)"
assert "frontmatter, validator exits 42: names the tool and the status" "$(echo "$errout" | grep -q 'skill-validator' && echo "$errout" | grep -q '42' && echo true || echo false)"

# Control: exit 2 is enumerated (warnings only) and still a pass.
run_on_path "$(stub_tool_path skill-validator 2)" "$CHECK_FM" tests/fixtures/f01/valid-full
assert "frontmatter, validator exits 2 (warnings): still passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert "frontmatter, validator exits 2 (warnings): prints frontmatter OK" "$(echo "$output" | grep -q 'frontmatter OK' && echo true || echo false)"

run_present "$(stub_paths_tree 42)" --json tests/fixtures/f01/valid-full
assert "structure --json, check-paths exits 42: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert "structure --json, check-paths exits 42: never reports passed true" "$(echo "$output" | grep -qE '"passed":[[:space:]]*true' && echo false || echo true)"

run_present "$(stub_paths_tree 42)" tests/fixtures/f01/valid-full
assert "structure text, check-paths exits 42: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"

# Control: an enumerated child exit still behaves as before.
run_present "$(stub_paths_tree 0)" --json tests/fixtures/f01/valid-full
assert "structure --json, check-paths exits 0: still passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert "structure --json, check-paths exits 0: reports passed true" "$([[ "$(echo "$output" | jq -r '.passed')" == "true" ]] && echo true || echo false)"

run_present "$(stub_paths_tree 1)" --json tests/fixtures/f01/valid-full
assert "structure --json, check-paths exits 1: fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"

echo
echo "$pass passed, $fail failed"
if [[ "$fail" -gt 0 ]]; then
  exit 1
fi
