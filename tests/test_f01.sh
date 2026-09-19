#!/usr/bin/env bash
set -euo pipefail
. "$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)/lib/harness.sh"
harness_init

# This suite's assertions compute their verdict inside a command substitution
# and hand the harness the result, so they call `assert_value` rather than
# `assert`. Both live in tests/lib/harness.sh and both count into the same
# summary; what they do not have is a copy here to drift from it.
#
# The value shape is the weaker of the two, because once the call is made a
# computed "true" and a written one are the same bytes. What holds it is
# tests/lib/audit-suites.sh, which reads the call sites and refuses a verdict
# with nothing in it to expand.

# Locate tools.
SV="skill-validator"
CHECK_FM="skills/skill-audit/scripts/check-frontmatter.sh"
CHECK_STRUCT="skills/skill-audit/scripts/check-structure.sh"
CHECK_PATHS="skills/skill-audit/scripts/check-paths.sh"
CHECK_QUALITY="skills/skill-audit/scripts/check-quality.sh"

# Helper: run a command and capture output + exit code.
# The suite's own runner for the cases that want both channels merged. It is
# not run_on_path with a different PATH — that one keeps stdout and stderr
# apart so a payload can be parsed — but the status it sees is the same kind of
# fact, so it is recorded the same way. Without this, every case driven through
# here was invisible to the exit-status witness at the end of this file, and
# `check-frontmatter.sh`'s exit 2 was stated by its header, driven three times
# in this suite, and witnessed nowhere.
run() {
  local cmd="$1"
  shift
  code=0
  output=$("$cmd" "$@" 2>&1) || code=$?
  witness_exit "$cmd" "$code"
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
assert_value "valid-minimal: spec passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"

# valid-full: spec passes
run_sv tests/fixtures/f01/valid-full
assert_value "valid-full: spec passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"

# malformed-yaml: spec fails
run_sv tests/fixtures/f01/malformed-yaml
assert_value "malformed-yaml: spec fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert_value "malformed-yaml: reports YAML error" "$(echo "$output" | grep -qi 'yaml\|parse' && echo true || echo false)"

# invalid-name-format: spec fails
run_sv tests/fixtures/f01/invalid-name-format
assert_value "invalid-name-format: spec fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert_value "invalid-name-format: reports name error" "$(echo "$output" | grep -qi 'name' && echo true || echo false)"

# name-mismatch: spec fails
run_sv tests/fixtures/f01/name-mismatch
assert_value "name-mismatch: spec fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert_value "name-mismatch: reports match error" "$(echo "$output" | grep -qi 'match\|directory' && echo true || echo false)"

# missing-description: spec fails
run_sv tests/fixtures/f01/missing-description
assert_value "missing-description: spec fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert_value "missing-description: reports description missing" "$(echo "$output" | grep -qi 'description' && echo true || echo false)"

# description-too-long: spec fails
run_sv tests/fixtures/f01/description-too-long
assert_value "description-too-long: spec fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert_value "description-too-long: reports exceeds limit" "$(echo "$output" | grep -qi 'exceeds\|1024' && echo true || echo false)"

# no-license: spec passes (license is optional per spec)
run_sv tests/fixtures/f01/no-license
assert_value "no-license: spec passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"

# --- Policy validation (check-frontmatter.sh for license, check-structure.sh for headings) ---

# valid-minimal: policy fails (no license)
run "$CHECK_FM" tests/fixtures/f01/valid-minimal
assert_value "valid-minimal: frontmatter fails (exit 2)" "$([[ $code -eq 2 ]] && echo true || echo false)"
assert_value "valid-minimal: reports PL001" "$(echo "$output" | grep -q 'PL001' && echo true || echo false)"

# valid-full: frontmatter passes
run "$CHECK_FM" tests/fixtures/f01/valid-full
assert_value "valid-full: frontmatter passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"

# no-license: spec passes, policy fails
run "$CHECK_FM" tests/fixtures/f01/no-license
assert_value "no-license: frontmatter fails (exit 2)" "$([[ $code -eq 2 ]] && echo true || echo false)"
assert_value "no-license: reports PL001" "$(echo "$output" | grep -q 'PL001' && echo true || echo false)"

# --- Structure/policy checks (check-structure.sh) ---

# valid-full: structure passes
run "$CHECK_STRUCT" tests/fixtures/f01/valid-full
assert_value "valid-full: structure passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"

# --- Path validation (check-paths.sh) ---

# missing-script-ref: path check fails
run "$CHECK_PATHS" tests/fixtures/f01/missing-script-ref
assert_value "missing-script-ref: paths fail (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert_value "missing-script-ref: reports PT001" "$(echo "$output" | grep -q 'PT001' && echo true || echo false)"

# missing-markdown-ref: path check fails
run "$CHECK_PATHS" tests/fixtures/f01/missing-markdown-ref
assert_value "missing-markdown-ref: paths fail (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert_value "missing-markdown-ref: reports PT002" "$(echo "$output" | grep -q 'PT002' && echo true || echo false)"

# dynamic-paths: path check reports unverified, not failure
run "$CHECK_PATHS" tests/fixtures/f01/dynamic-paths
assert_value "dynamic-paths: paths pass (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert_value "dynamic-paths: reports UNVERIFIED" "$(echo "$output" | grep -qi 'unverified' && echo true || echo false)"

# glob-paths: path check reports unverified, not failure
run "$CHECK_PATHS" tests/fixtures/f01/glob-paths
assert_value "glob-paths: paths pass (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert_value "glob-paths: reports UNVERIFIED" "$(echo "$output" | grep -qi 'unverified' && echo true || echo false)"

# --- Quality scoring (check-quality.sh / skillscore) ---

# valid-full: quality scoring produces JSON with categories
run "$CHECK_QUALITY" tests/fixtures/f01/valid-full
assert_value "valid-full: quality scoring succeeds (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert_value "valid-full: quality JSON has categories" "$(echo "$output" | grep -q 'categories' && echo true || echo false)"

# --- Bash entrypoint backward compatibility ---

# check-frontmatter.sh on valid-full should pass
run "$CHECK_FM" tests/fixtures/f01/valid-full
assert_value "check-frontmatter.sh valid-full passes" "$([[ $code -eq 0 ]] && echo true || echo false)"

# check-frontmatter.sh on no-license should fail (policy)
run "$CHECK_FM" tests/fixtures/f01/no-license
assert_value "check-frontmatter.sh no-license fails" "$([[ $code -ne 0 ]] && echo true || echo false)"

# check-structure.sh on valid-full should pass
run "$CHECK_STRUCT" tests/fixtures/f01/valid-full
assert_value "check-structure.sh valid-full passes" "$([[ $code -eq 0 ]] && echo true || echo false)"

# --- Dependency guards: never report a verdict a missing tool could not compute ---
#
# Invariant under test: when a tool a script needs to reach its verdict is absent,
# the script exits non-zero, says which tool is missing, and never emits a passing
# verdict — in the exit status and in the --json payload alike. Unenumerated child
# exit codes are execution errors, not passes.
#
# Masked PATHs are symlink farms of the real PATH minus one binary. Nothing is
# deleted, moved or uninstalled.

mask_root="$harness_scratch/mask"
mkdir -p "$mask_root"

# masked_path / run_on_path / run_masked / run_present live in the shared
# harness, because test_f02.sh needs the same masking for audit-report.sh.
source tests/lib/masked-path.sh

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

# stub_paths_tree <name> <exit-code> [stdout-payload]
#
# A copy of the scripts directory whose check-paths.sh writes <stdout-payload>
# and exits <exit-code>, so check-structure.sh can be driven against any child
# result — any status, paired with any payload or with none at all. Echoes the
# path to the copied check-structure.sh. Cached per name.
stub_paths_tree() {
  local name="$1"
  local exit_code="$2"
  local payload="${3-}"
  local dir="$mask_root/tree-$name"
  if [[ ! -d "$dir" ]]; then
    cp -R skills/skill-audit/scripts "$dir"
    printf '%s' "$payload" > "$dir/stub-payload"
    printf '#!/usr/bin/env bash\ncat "$(dirname "$0")/stub-payload"\nexit %s\n' "$exit_code" \
      > "$dir/check-paths.sh"
    chmod +x "$dir/check-paths.sh"
  fi
  echo "$dir/check-structure.sh"
}

# The payloads a conforming check-paths.sh --json produces, and the one its own
# guard produces when it cannot reach a verdict. Named here so a case can pair
# any of them with any exit status — including the pairings no conforming child
# would ever produce.
PAYLOAD_PASS='{"findings": [], "passed": true}'
PAYLOAD_FAIL='{"findings": [{"level": "fail", "rule": "PT001", "message": "script/reference path not found: ./scripts/nope.sh"}], "passed": false}'
PAYLOAD_DEP='{"findings": [{"level": "fail", "rule": "DEP001", "message": "required tool not found: jq"}], "passed": false, "error": "required tool not found: jq"}'

# guard_broken_tree <mode>
#
# A copy of the scripts directory whose shared verdict-guard.sh is unusable.
# Echoes the path to the copied directory. Cached per mode.
#
#   missing      no guard file at all
#   malformed    a file that will not parse
#   empty        a file that parses and defines nothing
#   half         a file defining one guard and not the others
#   drop-<name>  the real guard with <name>'s definition cut out
#
# The last two matter because "the guard loaded" is not one proposition. A file
# that stopped short — truncated, or edited down — defines some of its guards
# and not others, and a caller that only confirmed the ones it happened to name
# runs on until `command not found` at the point it needed the one that is gone.
# A mode defining a proper subset is the only witness that tells a complete
# check apart from a partial one.
guard_broken_tree() {
  local mode="$1"
  local dir="$mask_root/guard-$mode"
  if [[ ! -d "$dir" ]]; then
    cp -R skills/skill-audit/scripts "$dir"
    case "$mode" in
      missing)   rm -f "$dir/verdict-guard.sh" ;;
      malformed) printf 'if then fi (\n' > "$dir/verdict-guard.sh" ;;
      empty)     printf '# a guard that defines no guards\n' > "$dir/verdict-guard.sh" ;;
      half)      printf 'cannot_compute() { echo "ERROR: $2" >&2; exit 3; }\n' > "$dir/verdict-guard.sh" ;;
      drop-*)    sed "/^${mode#drop-}() {\$/,/^}\$/d" \
                   skills/skill-audit/scripts/verdict-guard.sh > "$dir/verdict-guard.sh" ;;
    esac
  fi
  echo "$dir"
}

PATH_FAULT=tests/fixtures/f01/path-fault-only

# --- The path-fault-only fixture: preconditions the S1 cases depend on ---
# If this fixture ever grew a policy fault, the S1 cases below would stop testing
# S1 (a policy finding never round-trips through jq), and they would still pass.

run_present "$CHECK_STRUCT" --json "$PATH_FAULT"
assert_value "path-fault-only: structure --json fails with jq present (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert_value "path-fault-only: structure --json reports passed false with jq present" "$([[ "$(echo "$output" | jq -r '.passed')" == "false" ]] && echo true || echo false)"
assert_value "path-fault-only: has a PT001 finding" "$(echo "$output" | jq -e '.findings[] | select(.rule == "PT001")' >/dev/null 2>&1 && echo true || echo false)"
assert_value "path-fault-only: has no PL policy finding (keeps S1 under test)" "$([[ "$(echo "$output" | jq -r '[.findings[] | select(.rule | startswith("PL"))] | length')" -eq 0 ]] && echo true || echo false)"

# --- S1: check-structure.sh --json must not pass a failing skill when jq is gone ---

run_masked jq "$CHECK_STRUCT" --json "$PATH_FAULT"
assert_value "S1 structure --json, jq masked: exits non-zero" "$([[ $code -ne 0 ]] && echo true || echo false)"
assert_value "S1 structure --json, jq masked: exits 3 (execution error)" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "S1 structure --json, jq masked: names the missing tool" "$(echo "$errout" | grep -q 'jq' && echo true || echo false)"
assert_value "S1 structure --json, jq masked: never reports passed true" "$(echo "$output" | grep -qE '"passed":[[:space:]]*true' && echo false || echo true)"
assert_value "S1 structure --json, jq masked: payload is valid JSON" "$(echo "$output" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
assert_value "S1 structure --json, jq masked: payload passed is false" "$([[ "$(echo "$output" | jq -r '.passed')" == "false" ]] && echo true || echo false)"
assert_value "S1 structure --json, jq masked: payload carries DEP001 naming jq" "$(echo "$output" | jq -e '.findings[] | select(.rule == "DEP001") | select(.message | test("jq"))' >/dev/null 2>&1 && echo true || echo false)"

# --- --json requires jq unconditionally, including on a clean skill ---
# The requirement is a stated precondition, not a function of what the skill
# happens to contain; a data-dependent dependency is what let S1 hide.

run_masked jq "$CHECK_STRUCT" --json tests/fixtures/f01/valid-full
assert_value "structure --json, jq masked, clean skill: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "structure --json, jq masked, clean skill: names the missing tool" "$(echo "$errout" | grep -q 'jq' && echo true || echo false)"
assert_value "structure --json, jq masked, clean skill: never reports passed true" "$(echo "$output" | grep -qE '"passed":[[:space:]]*true' && echo false || echo true)"

run_masked jq "$CHECK_PATHS" --json tests/fixtures/f01/valid-full
assert_value "paths --json, jq masked, clean skill: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "paths --json, jq masked, clean skill: names the missing tool" "$(echo "$errout" | grep -q 'jq' && echo true || echo false)"
assert_value "paths --json, jq masked, clean skill: never reports passed true" "$(echo "$output" | grep -qE '"passed":[[:space:]]*true' && echo false || echo true)"

# --- check-paths.sh --json under the same missing tool ---

run_masked jq "$CHECK_PATHS" --json "$PATH_FAULT"
assert_value "paths --json, jq masked: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "paths --json, jq masked: names the missing tool" "$(echo "$errout" | grep -q 'jq' && echo true || echo false)"
assert_value "paths --json, jq masked: payload passed is false" "$([[ "$(echo "$output" | jq -r '.passed')" == "false" ]] && echo true || echo false)"
assert_value "paths --json, jq masked: payload carries DEP001 naming jq" "$(echo "$output" | jq -e '.findings[] | select(.rule == "DEP001") | select(.message | test("jq"))' >/dev/null 2>&1 && echo true || echo false)"

# --- Text mode needs no jq and is unaffected by its absence ---

run_masked jq "$CHECK_PATHS" "$PATH_FAULT"
assert_value "paths text, jq masked: still fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert_value "paths text, jq masked: still reports PT001" "$(echo "$output" | grep -q 'PT001' && echo true || echo false)"

run_masked jq "$CHECK_STRUCT" "$PATH_FAULT"
assert_value "structure text, jq masked: still fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert_value "structure text, jq masked: still reports PT001" "$(echo "$output" | grep -q 'PT001' && echo true || echo false)"

run_masked jq "$CHECK_PATHS" tests/fixtures/f01/valid-full
assert_value "paths text, jq masked, clean skill: still passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"

run_masked jq "$CHECK_STRUCT" tests/fixtures/f01/valid-full
assert_value "structure text, jq masked, clean skill: still passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"

# --- S2: check-frontmatter.sh must not pass when skill-validator is gone ---

run_masked skill-validator "$CHECK_FM" tests/fixtures/f01/valid-full
assert_value "S2 frontmatter, validator masked: exits non-zero" "$([[ $code -ne 0 ]] && echo true || echo false)"
assert_value "S2 frontmatter, validator masked: exits 3 (execution error)" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "S2 frontmatter, validator masked: names the missing tool" "$(echo "$errout" | grep -q 'skill-validator' && echo true || echo false)"
assert_value "S2 frontmatter, validator masked: never prints frontmatter OK" "$(echo "$output" | grep -q 'frontmatter OK' && echo false || echo true)"

# A missing validator must not reclassify a spec failure as a policy failure.
run_masked skill-validator "$CHECK_FM" tests/fixtures/f01/malformed-yaml
assert_value "S2 frontmatter, validator masked, broken spec: exits 3 not 2" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "S2 frontmatter, validator masked, broken spec: does not report PL001" "$(echo "$output" | grep -q 'PL001' && echo false || echo true)"

# Controls: with the validator present, behaviour is unchanged.
run_present "$CHECK_FM" tests/fixtures/f01/valid-full
assert_value "frontmatter, validator present: valid-full passes (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert_value "frontmatter, validator present: valid-full prints frontmatter OK" "$(echo "$output" | grep -q 'frontmatter OK' && echo true || echo false)"

run_present "$CHECK_FM" tests/fixtures/f01/malformed-yaml
assert_value "frontmatter, validator present: malformed-yaml fails (exit 1)" "$([[ $code -eq 1 ]] && echo true || echo false)"
assert_value "frontmatter, validator present: malformed-yaml reports SPEC FAIL" "$(echo "$output" | grep -q 'SPEC FAIL' && echo true || echo false)"

# --- The spec source's payload is read, not inferred from its exit status ---
#
# check-frontmatter.sh asks for `-o json` and derived its whole spec verdict
# from `$?`, reading the payload only to quote it back in a failure message. On
# a healthy toolchain the status is a faithful proxy for the payload, which is
# what kept this invisible: a skill-validator that exits 0 while printing
# something that is not JSON satisfied every check and printed `frontmatter OK`.
# Masking the tool does not reach it — absence was already handled. The tool has
# to be *present and lying*.
#
# Both directions are driven, because a check on one of them is a check on the
# status again: a payload that cannot be read, and a payload that can be read
# and disagrees with the status it arrived with, each way round.

# spec_source_path <name> <exit> <stdout> — a skill-validator that says exactly
# this. One directory, one file, prepended to the real PATH.
spec_source_path() {
  local name="$1"
  local exit_code="$2"
  local payload="$3"
  local dir="$mask_root/spec-$name"
  if [[ ! -d "$dir" ]]; then
    mkdir -p "$dir"
    printf '%s' "$payload" > "$dir/spec-payload"
    printf '#!/usr/bin/env bash\ncat "$(dirname "$0")/spec-payload"\nexit %s\n' "$exit_code" \
      > "$dir/skill-validator"
    chmod +x "$dir/skill-validator"
  fi
  echo "$dir:$PATH"
}

# name | exit | payload | why it is no verdict
spec_liars="
notjson|0|checked 1 skill, all good|exit 0 with a payload that is not JSON
notjson-fail|1|1 error found|exit 1 with a payload that is not JSON
empty|0||exit 0 with nothing on the payload channel
number|0|7|exit 0 with a JSON document that is not an object
noerrors|0|{\"passed\": true}|exit 0 with an object carrying no errors count
stream|0|{\"errors\": 0}{\"errors\": 0}|exit 0 with two documents
clean-at-1|1|{\"passed\": true, \"errors\": 0}|exit 1 beside a payload reporting no error
dirty-at-0|0|{\"passed\": false, \"errors\": 3}|exit 0 beside a payload reporting three
dirty-at-2|2|{\"passed\": false, \"errors\": 2}|exit 2 beside a payload reporting two
"

spec_liar_cases=0
while IFS='|' read -r sname sexit spayload swhy; do
  [[ -z "$sname" ]] && continue
  spec_liar_cases=$((spec_liar_cases + 1))
  run_on_path "$(spec_source_path "$sname" "$sexit" "$spayload")" "$CHECK_FM" tests/fixtures/f01/valid-full
  assert_value "frontmatter, spec source $swhy: exits 3, not a status meaning a verdict" \
    "$([[ $code -eq 3 ]] && echo true || echo false)"
  assert_value "frontmatter, spec source $swhy: never prints frontmatter OK" \
    "$(echo "$output" | grep -q 'frontmatter OK' && echo false || echo true)"
  assert_value "frontmatter, spec source $swhy: never reports a policy verdict either" \
    "$(echo "$output" | grep -q 'PL001' && echo false || echo true)"
  assert_value "frontmatter, spec source $swhy: says on stderr that it could not read the source" \
    "$(echo "$errout" | grep -qi 'skill-validator' && echo true || echo false)"
done <<< "$spec_liars"

echo "  spec-source disagreement cases driven: $spec_liar_cases"
assert_value "the spec-source cases were enumerated, not read as empty" \
  "$([[ "$spec_liar_cases" -eq 9 ]] && echo true || echo false)"

# The controls, both statuses that carry a verdict. Without these the cases
# above would pass against a script that refused every payload there is.
run_on_path "$(spec_source_path "agrees-clean" 0 '{"passed": true, "errors": 0, "warnings": 0}')" \
  "$CHECK_FM" tests/fixtures/f01/valid-full
assert_value "frontmatter, spec source agreeing at exit 0: reaches its verdict (exit 0)" \
  "$([[ $code -eq 0 ]] && echo true || echo false)"

run_on_path "$(spec_source_path "agrees-warn" 2 '{"passed": true, "errors": 0, "warnings": 1}')" \
  "$CHECK_FM" tests/fixtures/f01/valid-full
assert_value "frontmatter, spec source agreeing at exit 2: warnings only is not a spec failure" \
  "$([[ $code -eq 0 ]] && echo true || echo false)"

run_on_path "$(spec_source_path "agrees-fail" 1 '{"passed": false, "errors": 1, "warnings": 0}')" \
  "$CHECK_FM" tests/fixtures/f01/valid-full
assert_value "frontmatter, spec source agreeing at exit 1: reports the spec failure (exit 1)" \
  "$([[ $code -eq 1 ]] && echo true || echo false)"
assert_value "frontmatter, spec source agreeing at exit 1: relays the payload it read" \
  "$(echo "$output" | grep -q 'SPEC FAIL' && echo true || echo false)"

# --- Every enumeration the guard introduced, walked at both edges ---
#
# A `case` arm or an `||` splits results into two sets: the ones a script will
# build a verdict on, and the ones it refuses. Pinning a single witness of the
# refused set leaves the accepted set's edge free to move — widen an enumeration
# by one status, or let one arm fall through, and the suite still passes while
# the script reports a verdict nothing computed. That is S1 and S2's shape
# exactly, so each enumeration below is walked across its whole boundary: every
# status the accepted set contains, the statuses immediately outside it, and the
# unenumerated remainder.

# skill-validator's statuses: 0 clean, 1 errors, 2 warnings only, 3 its own
# usage error. check-frontmatter.sh accepts {0, 2} as a verdict it may build on
# and reports 1 as a spec failure; everything else, 3 included, is a status it
# cannot interpret. An arm that reached the license gate anyway would print
# "frontmatter OK" over a validator that never ran — S2's exact symptom.

#
# Each stub here carries a payload that *agrees* with the status it exits with,
# so these cases ask about the status enumeration and nothing else. They used to
# carry a line of plain text, which made every one of them also a case about an
# unreadable payload — and passed, because the payload was not read at all. The
# payload question is its own section above, walked in both directions; keeping
# the two apart is what lets either boundary move without the other's cases
# going quiet.
SPEC_AGREES_CLEAN='{"passed": true, "errors": 0, "warnings": 0}'
SPEC_AGREES_FAIL='{"passed": false, "errors": 1, "warnings": 0}'

for spec in "0:0:ok:clean" "1:1:no:fail" "2:0:ok:clean" "3:3:no:clean" "4:3:no:clean" "42:3:no:clean"; do
  vcode="${spec%%:*}"
  vrest="${spec#*:}"
  vwant="${vrest%%:*}"
  vrest="${vrest#*:}"
  vverdict="${vrest%%:*}"
  vshape="${vrest#*:}"
  vpayload="$SPEC_AGREES_CLEAN"
  [[ "$vshape" == "fail" ]] && vpayload="$SPEC_AGREES_FAIL"
  run_on_path "$(spec_source_path "status-$vcode" "$vcode" "$vpayload")" "$CHECK_FM" tests/fixtures/f01/valid-full
  assert_value "frontmatter, validator exits $vcode: exits $vwant" \
    "$([[ $code -eq $vwant ]] && echo true || echo false)"
  if [[ "$vverdict" == "ok" ]]; then
    assert_value "frontmatter, validator exits $vcode: prints frontmatter OK" \
      "$(echo "$output" | grep -q 'frontmatter OK' && echo true || echo false)"
  else
    assert_value "frontmatter, validator exits $vcode: never prints frontmatter OK" \
      "$(echo "$output" | grep -q 'frontmatter OK' && echo false || echo true)"
  fi
done

# The unenumerated arm still names what it could not interpret, and the
# enumerated exit-3 arm names the tool that failed to run.
run_on_path "$(spec_source_path "status-42" 42 "$SPEC_AGREES_CLEAN")" "$CHECK_FM" tests/fixtures/f01/valid-full
assert_value "frontmatter, validator exits 42: names the tool and the status" \
  "$(echo "$errout" | grep -q 'skill-validator' && echo "$errout" | grep -q '42' && echo true || echo false)"

run_on_path "$(spec_source_path "status-3" 3 "$SPEC_AGREES_CLEAN")" "$CHECK_FM" tests/fixtures/f01/valid-full
assert_value "frontmatter, validator exits 3: names the tool that failed to run" \
  "$(echo "$errout" | grep -q 'skill-validator' && echo true || echo false)"

# --- check-structure.sh over every child result it can be handed ---
#
# Two enumerations meet here: the child's exit status, and the read-back of the
# child's payload. check-paths.sh's contract is 0 = pass, 1 = path failure, and
# 3 = its own guard reporting that it reached no verdict; in --json mode it also
# promises a {"findings": [...], "passed": bool} payload on stdout whenever it
# exits 0 or 1. Every column below is a way that contract can be broken, and the
# rows immediately inside it are the accepted set's edge.
#
# DEP001 (a required tool is absent) and DEP002 (a child produced a result this
# script cannot interpret) are both consumer-observable: they reach a --json
# consumer in the findings array. A consumer has to be able to tell them apart,
# so both IDs are documented in skills/skill-audit/SKILL.md and both are pinned
# here. An emitted ID that appears in no document and no test is how the two of
# them shipped unregistered.

child_names=(pass0 fail1 dep3 u42 garbage0 empty0 shape0 liar0 liar1)
child_codes=(0 1 3 42 0 0 0 0 1)
child_payloads=("$PAYLOAD_PASS" "$PAYLOAD_FAIL" "$PAYLOAD_DEP" "" "not json at all" "" '{"ok": true}' "$PAYLOAD_FAIL" "$PAYLOAD_PASS")
child_why=(
  "a conforming pass"
  "a conforming path failure"
  "the child's own no-verdict (exit 3)"
  "an unenumerated status"
  "an unreadable payload"
  "no payload at all"
  "a payload of the wrong shape"
  "a payload contradicting its exit 0"
  "a payload contradicting its exit 1"
)
want_exit=(0 1 3 3 3 3 3 3 3)
want_passed=(true false false false false false false false false)
want_rule=("" PT001 DEP002 DEP002 DEP002 DEP002 DEP002 DEP002 DEP002)

for i in "${!child_names[@]}"; do
  why="${child_why[$i]}"
  run_present "$(stub_paths_tree "${child_names[$i]}" "${child_codes[$i]}" "${child_payloads[$i]}")" \
    --json tests/fixtures/f01/valid-full
  assert_value "structure --json, child gives $why: exits ${want_exit[$i]}" \
    "$([[ $code -eq ${want_exit[$i]} ]] && echo true || echo false)"
  assert_value "structure --json, child gives $why: stdout is a payload" \
    "$(echo "$output" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
  assert_value "structure --json, child gives $why: payload passed is ${want_passed[$i]}" \
    "$([[ "$(echo "$output" | jq -r '.passed' 2>/dev/null)" == "${want_passed[$i]}" ]] && echo true || echo false)"
  if [[ -n "${want_rule[$i]}" ]]; then
    assert_value "structure --json, child gives $why: payload carries ${want_rule[$i]}" \
      "$(echo "$output" | jq -e --arg r "${want_rule[$i]}" '.findings[] | select(.rule == $r)' >/dev/null 2>&1 && echo true || echo false)"
  fi
  # The invariant every arm above exists to hold, pinned over the whole boundary
  # so it survives any rework of the arms that currently enforce it: a payload
  # may never claim it passed while carrying a finding that says it failed.
  assert_value "structure --json, child gives $why: never claims passed beside a fail finding" \
    "$(echo "$output" | jq -e '.passed == true and ([.findings[] | select(.level == "fail")] | length > 0)' >/dev/null 2>&1 && echo false || echo true)"
done

# The DEP002 message names the status it could not interpret, so a consumer can
# tell which child result it is looking at.
run_present "$(stub_paths_tree u42 42 "")" --json tests/fixtures/f01/valid-full
assert_value "structure --json, child exits 42: DEP002 names the status" \
  "$(echo "$output" | jq -e '.findings[] | select(.rule == "DEP002") | select(.message | test("42"))' >/dev/null 2>&1 && echo true || echo false)"

# --- check-structure.sh over shape-space, walked independently of status-space ---
#
# The table above varies which canned payload pairs with which exit status. It
# never varies a payload's internal type-shape, and the predicate it was written
# against answered two questions in one expression — "is this payload
# well-formed?" and "what verdict does it carry?" — joined by an `and` that
# short-circuits. So the well-formedness half ran only on the branch the verdict
# took: a payload saying `"passed": false` had its findings elements checked by
# nothing, a misshapen element reached the read-back below, and the script died
# with jq's own exit 5 — outside the {0,1,2,3} this contract documents, with
# nothing at all on the payload channel.
#
# Shape is therefore driven here as an axis of its own, crossed with status
# rather than folded into it: every way the documented
# {"findings": [{level, rule, message}, ...], "passed": bool} shape can break,
# at both of `.passed`'s truth values wherever the break is below the top level,
# each against both statuses a conforming child may exit with. Nothing below is
# pinned to the arm that enforces the contract, only to the contract itself — a
# child payload is the documented shape, agrees with itself, and agrees with the
# status it arrived with, or it is a DEP002 no-verdict — and, over every row, no
# child result of any kind may push this script outside its documented exit set.

F_FAIL='{"level": "fail", "rule": "PT001", "message": "script/reference path not found: ./scripts/nope.sh"}'
F_UNVER='{"level": "unverified", "rule": "PATH", "message": "dynamic/glob path: ./scripts/*"}'

shape_names=()
shape_payloads=()
shape_at0=()
shape_at1=()

# shape_case <name> <payload> <outcome when the child exits 0> <outcome when it exits 1>
#
# pass = exit 0 and a passing payload; fail = exit 1 and a failing payload;
# dep  = exit 3 and a DEP002 payload, this script's one way of saying it read no
# verdict there. Each outcome is written out per row rather than derived, so the
# table states the contract instead of restating the implementation.
shape_case() {
  shape_names+=("$1")
  shape_payloads+=("$2")
  shape_at0+=("$3")
  shape_at1+=("$4")
}

# Conforming payloads. The shape holds, so the verdict is read — and then judged
# against itself and against the status it arrived with.
shape_case conforming-pass         "{\"passed\": true, \"findings\": []}"        pass dep
shape_case conforming-fail         "{\"passed\": false, \"findings\": [$F_FAIL]}" dep  fail
shape_case conforming-unverified   "{\"passed\": true, \"findings\": [$F_UNVER]}" pass dep
shape_case conforming-false-empty  "{\"passed\": false, \"findings\": []}"       dep  fail
# A payload that claims it passed while carrying a finding saying it failed is
# self-contradictory whatever status it arrives with. Neither half can be
# believed over the other, so no verdict is read from it at either status. This
# is the row the previous table never carried: the assertion that claimed to pin
# this invariant could not fire, because no row paired `passed: true` with a
# `level: "fail"` finding.
shape_case self-contradictory      "{\"passed\": true, \"findings\": [$F_FAIL]}"  dep  dep

# The top level is not the documented object.
shape_case top-number              '42'                                          dep dep
shape_case top-string              '"a payload"'                                 dep dep
shape_case top-array               '[{"passed": true, "findings": []}]'          dep dep

# `.passed` is not a boolean. A truthy string here is what read as a computed
# pass when only the findings half of the shape was proven.
shape_case passed-string           '{"passed": "yes", "findings": []}'           dep dep
shape_case passed-number           '{"passed": 1, "findings": []}'               dep dep
shape_case passed-null             '{"passed": null, "findings": []}'            dep dep
shape_case passed-absent           '{"findings": []}'                            dep dep

# `.findings` is not an array.
shape_case findings-string         '{"passed": true, "findings": "none"}'        dep dep
shape_case findings-object         '{"passed": true, "findings": {}}'            dep dep
shape_case findings-null           '{"passed": true, "findings": null}'          dep dep
shape_case findings-absent         '{"passed": true}'                            dep dep

# An element is not an object — at both truth values of `.passed`, because it
# was `.passed` deciding whether these were looked at that let them through.
shape_case element-number-true     '{"passed": true, "findings": [42]}'          dep dep
shape_case element-number-false    '{"passed": false, "findings": [42]}'         dep dep
shape_case element-string-true     '{"passed": true, "findings": ["x"]}'         dep dep
shape_case element-string-false    '{"passed": false, "findings": ["x"]}'        dep dep
shape_case element-array-true      '{"passed": true, "findings": [[1]]}'         dep dep
shape_case element-array-false     '{"passed": false, "findings": [[1]]}'        dep dep
shape_case element-null-true       '{"passed": true, "findings": [null]}'        dep dep
shape_case element-null-false      '{"passed": false, "findings": [null]}'       dep dep

# An element is an object, but not the documented one. Relaying these verbatim
# is how a parent fabricates a finding: jq stringifies an absent field as the
# four characters `null`, and the merged payload then carries
# {"level": "null", "rule": "null", "message": "null"} as though the child had
# said it.
shape_case element-no-level-true   '{"passed": true, "findings": [{"rule": "PT001", "message": "m"}]}'                 dep dep
shape_case element-no-level-false  '{"passed": false, "findings": [{"rule": "PT001", "message": "m"}]}'                dep dep
shape_case element-level-number    '{"passed": false, "findings": [{"level": 1, "rule": "PT001", "message": "m"}]}'    dep dep
shape_case element-no-rule         '{"passed": false, "findings": [{"level": "fail", "message": "m"}]}'                dep dep
shape_case element-rule-number     '{"passed": false, "findings": [{"level": "fail", "rule": 123, "message": "m"}]}'   dep dep
shape_case element-no-message      '{"passed": false, "findings": [{"level": "fail", "rule": "PT001"}]}'               dep dep

# Every element, not merely some element. Each row above breaks the shape in an
# array where *no* element conforms, so a predicate asking "does some element
# conform?" answers the same as one asking "do all of them?", and no row can
# tell the two apart. A conforming element beside a non-conforming one in the
# same array separates them: the weaker question reads the payload as the
# documented shape and hands it to a reader that then indexes `.level` of a
# number — the exact defect the predicate exists to close, reopened.
shape_case element-mixed-false     "{\"passed\": false, \"findings\": [$F_FAIL, 42]}"   dep dep
shape_case element-mixed-true      "{\"passed\": true, \"findings\": [$F_UNVER, 42]}"   dep dep

# One document is the contract. A stream of them is not, whichever position the
# conforming one holds in it.
shape_case trailing-garbage        '{"passed": true, "findings": []} {"x": 1}'                             dep dep
shape_case leading-garbage         '{"x": 1} {"passed": true, "findings": []}'                             dep dep
shape_case two-payloads            '{"passed": true, "findings": []} {"passed": false, "findings": []}'    dep dep

for i in "${!shape_names[@]}"; do
  sname="${shape_names[$i]}"
  for sstatus in 0 1; do
    if [[ $sstatus -eq 0 ]]; then swant="${shape_at0[$i]}"; else swant="${shape_at1[$i]}"; fi
    run_present "$(stub_paths_tree "shape-$sname-$sstatus" "$sstatus" "${shape_payloads[$i]}")" \
      --json tests/fixtures/f01/valid-full

    assert_value "structure --json, child payload $sname at exit $sstatus: exits inside the documented set" \
      "$([[ $code -eq 0 || $code -eq 1 || $code -eq 2 || $code -eq 3 ]] && echo true || echo false)"
    assert_value "structure --json, child payload $sname at exit $sstatus: stdout is a payload" \
      "$(echo "$output" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
    assert_value "structure --json, child payload $sname at exit $sstatus: never claims passed beside a fail finding" \
      "$(echo "$output" | jq -e '.passed == true and ([.findings[] | select(.level == "fail")] | length > 0)' >/dev/null 2>&1 && echo false || echo true)"

    sgot=false
    case "$swant" in
      pass)
        if [[ $code -eq 0 && "$(echo "$output" | jq -r '.passed' 2>/dev/null)" == "true" ]]; then sgot=true; fi
        assert_value "structure --json, child payload $sname at exit $sstatus: is a clean pass" "$sgot" ;;
      fail)
        if [[ $code -eq 1 && "$(echo "$output" | jq -r '.passed' 2>/dev/null)" == "false" ]]; then sgot=true; fi
        assert_value "structure --json, child payload $sname at exit $sstatus: is a path failure" "$sgot" ;;
      dep)
        if [[ $code -eq 3 && "$(echo "$output" | jq -r '.passed' 2>/dev/null)" == "false" ]] \
           && echo "$output" | jq -e '.findings[] | select(.rule == "DEP002")' >/dev/null 2>&1; then sgot=true; fi
        assert_value "structure --json, child payload $sname at exit $sstatus: is a DEP002 no-verdict" "$sgot" ;;
    esac
  done
done

# A non-conforming element is refused, so nothing is fabricated out of it: the
# merged findings never contain the string "null" standing in for a field the
# child did not send.
run_present "$(stub_paths_tree shape-element-null-false 1 '{"passed": false, "findings": [null]}')" \
  --json tests/fixtures/f01/valid-full
assert_value "structure --json, child sends a null finding: no finding is fabricated from it" \
  "$(echo "$output" | jq -e '[.findings[] | select(.level == "null" or .rule == "null" or .message == "null")] | length == 0' >/dev/null 2>&1 && echo true || echo false)"

# Text mode reads no payload, so only the exit-status enumeration applies — and
# it applies identically: a child that reached no verdict leaves us with none.
run_present "$(stub_paths_tree pass0 0 "$PAYLOAD_PASS")" tests/fixtures/f01/valid-full
assert_value "structure text, child gives a conforming pass: exits 0" "$([[ $code -eq 0 ]] && echo true || echo false)"

run_present "$(stub_paths_tree dep3 3 "$PAYLOAD_DEP")" tests/fixtures/f01/valid-full
assert_value "structure text, child exits 3 (no verdict): exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"

run_present "$(stub_paths_tree u42 42 "")" tests/fixtures/f01/valid-full
assert_value "structure text, child exits 42: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"

# DEP001's own registration is pinned by the jq-masked cases above, which assert
# the rule ID in both check-structure.sh's and check-paths.sh's payloads.

# --- The shared guard is itself a dependency, and require_tool cannot cover it ---
#
# Every other dependency is announced by the guard. The guard's own absence is
# announced by nothing: `source` fails, `set -e` aborts, and the script exits 1
# or 2 — statuses this contract reserves for verdicts the script actually
# computed. A missing guard must read as "no verdict reached", like every other
# unmet precondition, in the exit status and on the payload channel alike.

for gmode in missing malformed empty half; do
  gdir="$(guard_broken_tree "$gmode")"
  for gscript in check-structure.sh check-paths.sh check-frontmatter.sh audit-report.sh; do
    run_present "$gdir/$gscript" tests/fixtures/f01/valid-full
    assert_value "$gscript, guard $gmode: exits 3, not a status meaning a verdict" \
      "$([[ $code -eq 3 ]] && echo true || echo false)"
    assert_value "$gscript, guard $gmode: stdout carries no verdict" \
      "$([[ -z "$output" ]] && echo true || echo false)"
    assert_value "$gscript, guard $gmode: says on stderr that it could not load the guard" \
      "$(echo "$errout" | grep -q 'verdict-guard.sh' && echo true || echo false)"
  done
  run_present "$gdir/check-structure.sh" --json tests/fixtures/f01/valid-full
  assert_value "check-structure.sh --json, guard $gmode: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
  assert_value "check-structure.sh --json, guard $gmode: never reports passed true" \
    "$(echo "$output" | grep -qE '"passed":[[:space:]]*true' && echo false || echo true)"
done

# "The guard loaded" is not one proposition, and a caller that checks it by
# naming the guards it happens to remember is checking a proper subset of it.
# The `half` mode above defines one guard and not the rest; a caller confirming
# only that one runs on to `command not found`, or — worse, and reproduced —
# straight through to a clean exit 0 with no guard behind the verdict at all.
# These modes take the real guard and cut out one definition each, so the check
# has to cover the whole set rather than whichever names a caller listed.
#
# The set is read out of the guard rather than written here, so a primitive
# added to the shared file tomorrow is covered the day it lands. A hand-kept
# list is the same defect one level up: it covers the names a fixer remembered,
# and the one it did not is exactly the one whose absence nothing catches. The
# count is asserted first, because an expression that matched no definition
# would make every case below vacuously true.
guard_definitions="$(sed -n 's/^\([a-z_][a-z_0-9]*\)() {$/\1/p' skills/skill-audit/scripts/verdict-guard.sh)"
guard_definition_count="$(printf '%s\n' "$guard_definitions" | grep -c '[a-z]' || true)"
assert_value "the guard's definitions were enumerated, not read as an empty set" \
  "$([[ "${guard_definition_count:-0}" -ge 6 ]] && echo true || echo false)"
echo "  guard primitives examined: $(printf '%s' "$guard_definitions" | tr '\n' ' ')"

for gdrop in $guard_definitions; do
  gdir="$(guard_broken_tree "drop-$gdrop")"
  run_present "$gdir/check-structure.sh" --json tests/fixtures/f01/valid-full
  assert_value "check-structure.sh --json, guard missing $gdrop: exits 3, not a status meaning a verdict" \
    "$([[ $code -eq 3 ]] && echo true || echo false)"
  assert_value "check-structure.sh --json, guard missing $gdrop: stdout carries no verdict" \
    "$([[ -z "$output" ]] && echo true || echo false)"
  assert_value "check-structure.sh --json, guard missing $gdrop: says on stderr that it could not load the guard" \
    "$(echo "$errout" | grep -q 'verdict-guard.sh' && echo true || echo false)"
done

# --- The guard emits JSON, whatever the message holds ---
#
# cannot_compute promises the documented payload shape. Building that shape by
# interpolating an unescaped message makes the promise conditional on every
# caller passing a string with no quote, backslash or control character — a
# convention no caller is checked against. Pin the promise, not the convention.
#
# The encoder's own `case` is an enumeration like any other, so the message
# below carries one character from every arm of it — the six named escapes, the
# quote and backslash, and two characters that fall to the \u00xx arm at the
# edges of the control range. A message that exercised only the arms someone
# thought of is how an encoder ships escaping most of what it is handed.

GUARD=skills/skill-audit/scripts/verdict-guard.sh
assert_value "verdict-guard.sh sits beside the scripts that source it" "$([[ -f "$GUARD" ]] && echo true || echo false)"

guard_nasty=$'he said "boom" \\ then a tab\there, a newline\na return\ra formfeed\fa backspace\bthen \x01 and \x1f'
guard_out="$(bash -c 'source "$1"; cannot_compute DEP002 "$2" true' _ "$GUARD" "$guard_nasty" 2>/dev/null || true)"
assert_value "guard: payload is valid JSON when the message holds every character the encoder escapes" "$(echo "$guard_out" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
assert_value "guard: error field round-trips the message exactly" "$([[ "$(echo "$guard_out" | jq -r '.error' 2>/dev/null)" == "$guard_nasty" ]] && echo true || echo false)"
assert_value "guard: finding message round-trips the message exactly" "$([[ "$(echo "$guard_out" | jq -r '.findings[0].message' 2>/dev/null)" == "$guard_nasty" ]] && echo true || echo false)"
assert_value "guard: finding carries the rule it was given" "$([[ "$(echo "$guard_out" | jq -r '.findings[0].rule' 2>/dev/null)" == "DEP002" ]] && echo true || echo false)"
assert_value "guard: payload never reports passed true" "$(echo "$guard_out" | grep -qE '"passed":[[:space:]]*true' && echo false || echo true)"

# A message is bytes, not ASCII, and the encoder exists because a future caller
# relaying a tool's own output would break the interpolated form it replaced.
# Deciding what needs escaping by reading a character's ordinal is what broke
# that: the shell yields a *negative* ordinal for any byte at or above 0x80, so
# every byte of a multi-byte character took the control-character branch and
# came back as ￿ffffffffffc3. The payload still parsed — the shape promise
# held — while the message it carried no longer said what it was given, which is
# the encoder's whole job. So the round-trip is asserted over text that is
# actually multi-byte, not only over an ASCII nasty string.

guard_utf8=$'caf\xc3\xa9 \xf0\x9f\x98\x80 na\xc3\xafve \xe2\x80\x94 \xc2\xa0 \xe6\x97\xa5\xe6\x9c\xac\xe8\xaa\x9e'
guard_utf8_out="$(bash -c 'source "$1"; cannot_compute DEP002 "$2" true' _ "$GUARD" "$guard_utf8" 2>/dev/null || true)"
assert_value "guard: payload is valid JSON when the message is multi-byte UTF-8" \
  "$(echo "$guard_utf8_out" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
assert_value "guard: error field round-trips multi-byte UTF-8 exactly" \
  "$([[ "$(echo "$guard_utf8_out" | jq -r '.error' 2>/dev/null)" == "$guard_utf8" ]] && echo true || echo false)"
assert_value "guard: finding message round-trips multi-byte UTF-8 exactly" \
  "$([[ "$(echo "$guard_utf8_out" | jq -r '.findings[0].message' 2>/dev/null)" == "$guard_utf8" ]] && echo true || echo false)"

# Control characters and multi-byte characters in one message, straight through
# the encoder: neither may be read as the other.
guard_mixed=$'"\\ caf\xc3\xa9\ttab\nnewline\x01 \xf0\x9f\x98\x80 \x1f end'
guard_mixed_out="$(bash -c 'source "$1"; json_string "$2"' _ "$GUARD" "$guard_mixed" 2>/dev/null || true)"
assert_value "guard: json_string emits valid JSON for control characters beside multi-byte ones" \
  "$(echo "$guard_mixed_out" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
assert_value "guard: json_string round-trips control characters beside multi-byte ones exactly" \
  "$([[ "$(echo "$guard_mixed_out" | jq -r . 2>/dev/null)" == "$guard_mixed" ]] && echo true || echo false)"

# The decision and the format have to be the same question. `\u00xx` can spell
# an ordinal below 0x80 and nothing else, so the arm that emits it may never be
# reached with an ordinal it cannot spell — and deciding by character class
# rather than by ordinal is what let the two disagree. Under a UTF-8 locale
# `[[:cntrl:]]` matches a byte at or above 0x80, both an invalid UTF-8 byte and
# the valid C1 controls U+0080-U+009F, while bash reports that byte as a
# *negative* ordinal: the format then emitted sixteen hex digits where it
# promised four, and the message stopped saying what it was given. That is the
# same corruption as the ordinal test this replaced, one input class short of
# closed, and it is invisible to a check that only asks whether the payload
# parses — `￿` is a legal escape and the rest of the digits are literal
# text.
#
# So the invariant is pinned directly — no escape this encoder emits is wider
# than four hex digits — and beside it the consequence that makes it matter:
# the same bytes in, the same bytes out, whatever locale the encoder ran under.

utf8_locale=""
for lcand in en_US.UTF-8 en_US.utf8 C.UTF-8 C.utf8; do
  if [[ "$(LC_ALL="$lcand" locale charmap 2>/dev/null)" == "UTF-8" ]]; then
    utf8_locale="$lcand"
    break
  fi
done
assert_value "guard: a UTF-8 locale is available to drive the encoder under" \
  "$([[ -n "$utf8_locale" ]] && echo true || echo false)"

enc_names=(invalid-byte-80 invalid-byte-ff c1-control-nel c1-control-9f ascii-control del multi-byte)
# Each witness ends in a character that is not a hex digit. A JSON escape
# carries no terminator, so `\u0001b` is five hex digits after the `\u` by
# inspection and four by construction, and a width check over it would be
# reading the text after the escape as part of it.
enc_inputs=($'a\x80z' $'a\xffz' $'a\xc2\x85z' $'a\xc2\x9fz' $'a\x01z' $'a\x7fz' $'caf\xc3\xa9 \xf0\x9f\x98\x80')

for ei in "${!enc_names[@]}"; do
  ename="${enc_names[$ei]}"
  enc_c="$(LC_ALL=C bash -c 'source "$1"; json_string "$2"' _ "$GUARD" "${enc_inputs[$ei]}" 2>/dev/null || true)"
  enc_u="$(LC_ALL="$utf8_locale" bash -c 'source "$1"; json_string "$2"' _ "$GUARD" "${enc_inputs[$ei]}" 2>/dev/null || true)"

  assert_value "guard: json_string on $ename emits no escape wider than four hex digits, under C" \
    "$(printf '%s' "$enc_c" | grep -qE '\\u[0-9a-fA-F]{5}' && echo false || echo true)"
  assert_value "guard: json_string on $ename emits no escape wider than four hex digits, under $utf8_locale" \
    "$(printf '%s' "$enc_u" | grep -qE '\\u[0-9a-fA-F]{5}' && echo false || echo true)"
  assert_value "guard: json_string on $ename encodes identically under C and $utf8_locale" \
    "$([[ "$enc_c" == "$enc_u" ]] && echo true || echo false)"
done

# The two ends of the class, stated as behaviour rather than as byte counts.
# U+0085 is valid UTF-8 and a control character, and JSON asks for neither of
# those to be escaped, so it has to survive the encoder intact. A byte that is
# not valid UTF-8 at all is not the encoder's to interpret either: the header
# promises everything outside the three escaped classes is passed through
# exactly as it arrived.

enc_c1_msg=$'before \xc2\x85 after'
enc_c1_out="$(LC_ALL="$utf8_locale" bash -c 'source "$1"; json_string "$2"' _ "$GUARD" "$enc_c1_msg" 2>/dev/null || true)"
assert_value "guard: json_string round-trips the C1 control U+0085 exactly under a UTF-8 locale" \
  "$([[ "$(printf '%s' "$enc_c1_out" | jq -r . 2>/dev/null)" == "$enc_c1_msg" ]] && echo true || echo false)"

enc_raw_msg=$'a\x80z'
enc_raw_out="$(LC_ALL="$utf8_locale" bash -c 'source "$1"; json_string "$2"' _ "$GUARD" "$enc_raw_msg" 2>/dev/null || true)"
assert_value "guard: json_string passes an invalid UTF-8 byte through unchanged under a UTF-8 locale" \
  "$([[ "$enc_raw_out" == "\"$enc_raw_msg\"" ]] && echo true || echo false)"

# The detail — a failed tool's own output — is the one thing the guard relays
# verbatim, and it is exactly the thing that would corrupt the payload if it
# went to stdout. It belongs on stderr, beside the reason, whatever it holds.

run_present bash -c 'source "$1"; cannot_compute DEP002 "the reason" true "DETAIL-MARKER: {not json"' _ "$GUARD"
assert_value "guard: detail reaches stderr" "$(echo "$errout" | grep -q 'DETAIL-MARKER' && echo true || echo false)"
assert_value "guard: detail never reaches the payload channel" "$(echo "$output" | grep -q 'DETAIL-MARKER' && echo false || echo true)"
assert_value "guard: stdout is still exactly the payload when a detail is relayed" "$(echo "$output" | jq -e '.findings[0].rule == "DEP002"' >/dev/null 2>&1 && echo true || echo false)"
assert_value "guard: the reason reaches stderr too" "$(echo "$errout" | grep -q 'the reason' && echo true || echo false)"

# The guard needs no jq to build its payload: it is what runs when jq is the
# tool that went missing.
run_masked jq bash -c 'source "$1"; cannot_compute DEP001 "required tool not found: jq" true' _ "$GUARD"
assert_value "guard: emits its payload with jq itself absent" "$(echo "$output" | jq -e '.findings[0].rule == "DEP001"' >/dev/null 2>&1 && echo true || echo false)"

# --- stdout is the payload channel, on every exit path ---
#
# In --json mode a machine reads stdout. So stdout carries a payload or it
# carries nothing; a diagnostic belongs on stderr. Three exits carry no payload:
# an argument error and an unresolvable target, where the script has no target to
# render a verdict about, and a guard that would not load, where the thing that
# builds payloads is the thing that is missing. That is the whole of the
# exception, stated in both script headers and pinned here — and, for the guard,
# in the guard-load cases above — so it cannot quietly grow.

run_present "$CHECK_PATHS" --json --bogus
assert_value "paths --json, bad flag: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "paths --json, bad flag: stdout is empty" "$([[ -z "$output" ]] && echo true || echo false)"
assert_value "paths --json, bad flag: diagnostic is on stderr" "$(echo "$errout" | grep -q 'Usage' && echo true || echo false)"

run_present "$CHECK_STRUCT" --json --bogus
assert_value "structure --json, bad flag: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "structure --json, bad flag: stdout is empty" "$([[ -z "$output" ]] && echo true || echo false)"
assert_value "structure --json, bad flag: diagnostic is on stderr" "$(echo "$errout" | grep -q 'Usage' && echo true || echo false)"

run_present "$CHECK_PATHS" --json "$mask_root/no-such-skill"
assert_value "paths --json, no SKILL.md: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "paths --json, no SKILL.md: stdout is empty" "$([[ -z "$output" ]] && echo true || echo false)"
assert_value "paths --json, no SKILL.md: diagnostic is on stderr" "$(echo "$errout" | grep -q 'SKILL.md not found' && echo true || echo false)"

run_present "$CHECK_STRUCT" --json "$mask_root/no-such-skill"
assert_value "structure --json, no SKILL.md: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "structure --json, no SKILL.md: stdout is empty" "$([[ -z "$output" ]] && echo true || echo false)"
assert_value "structure --json, no SKILL.md: diagnostic is on stderr" "$(echo "$errout" | grep -q 'SKILL.md not found' && echo true || echo false)"

run_present "$CHECK_FM" "$mask_root/no-such-skill"
assert_value "frontmatter, no SKILL.md: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "frontmatter, no SKILL.md: diagnostic is on stderr" "$(echo "$errout" | grep -q 'SKILL.md not found' && echo true || echo false)"

# --- Every rule ID these scripts can emit is registered, and no other ---
#
# A rule ID is consumer-visible: it reaches a `--json` consumer in the findings
# array, and a consumer that meets one it has never been told about cannot tell
# a policy failure from an execution error. SKILL.md's rule-ID line therefore
# claims to be the whole set. That claim was false — `PATH`, the level-
# `unverified` finding check-paths.sh raises for a reference built from a glob
# or a variable, appeared in no document, no script header and no test, and
# propagated through check-structure.sh --json into audit-report.sh's
# policy.findings all the same.
#
# A sentence claiming to be exhaustive needs something that keeps it exhaustive,
# so the two sides are compared here rather than restated: every rule literal
# the scripts can emit, against every rule ID the registry line names. Either
# side growing without the other fails.
#
# Both sides read what is written rather than what is run, so both have to read
# every form a rule ID reaches a consumer through, not only the forms that
# happen to be literals. Three of them are not literals in the emitting script
# at all: an ID handed to the guard is quoted at some call sites and bare at
# others, `require_tool` raises DEP001 on behalf of every script that states a
# tool precondition, and a script that runs a sibling with `--json` carries that
# sibling's whole findings array out to its own consumer. A census blind to
# those credits a script with none of the rules it actually emits, and then
# passes while an unregistered ID reaches a `--json` consumer — which is the one
# thing it exists to prevent.

SCRIPTS_DIR=skills/skill-audit/scripts

# rules_emitted_directly_by <file...> — the rule IDs written in these files: the
# pipe-delimited entries they build, the IDs they hand the guard quoted or bare,
# the objects they compose directly, the IDs they print in text mode, and
# DEP001 where a tool precondition is stated in a form that carries the ID to a
# consumer. Each form tolerates no match, so a script that emits nothing yields
# nothing instead of killing the suite.
#
# `require_tool <tool> <emit_json>` only reaches a consumer with the rule ID on
# it when <emit_json> is `true`. With `false` the guard writes "required tool
# not found: <tool>" to stderr and exits 3 with nothing on stdout, so no reader
# of that script ever sees DEP001 from it. Crediting the ID anyway is not a
# harmless over-count: it made this check demand DEP001 in the header of a
# script that cannot emit it, and the header was duly changed to say something
# untrue. Attribution that is only ever too generous still forces a lie.
#
# The argument is read, and the line start is not anchored, because a
# precondition stated after a `&&` is the same precondition.
rules_emitted_directly_by() {
  {
    grep -hoE 'findings\+=\("[a-z]+\|[A-Z][A-Z0-9]*\|' "$@" | sed -E 's/.*\|([A-Z][A-Z0-9]*)\|/\1/' || true
    grep -hoE 'cannot_compute[[:space:]]+"?[A-Z][A-Z0-9]*' "$@" | tr -d '"' | awk '{print $NF}' || true
    # The spacing inside a composed object is the author's, not the contract's,
    # so it is not read as if it were: an object built by a `printf` that omits
    # the space after the colon emits the same rule to the same consumer.
    grep -hoE '"rule"[[:space:]]*:[[:space:]]*"[A-Z][A-Z0-9]*"' "$@" | sed -E 's/.*"([A-Z][A-Z0-9]*)"/\1/' || true
    grep -hoE '\[[A-Z][A-Z0-9]*\]' "$@" | tr -d '[]' || true
    if grep -qE 'require_tool[[:space:]]+[^[:space:]]+[[:space:]]+true([[:space:]]|$)' "$@"; then
      echo DEP001
    fi
  } | sort -u
}

# rules_of <file> <chain> — what this file can emit, directly and through every
# sibling it runs with `--json`. check-structure.sh rebuilds each of
# check-paths.sh's findings from variables and audit-report.sh splices
# check-structure.sh's array in whole, so neither leaves a rule literal of the
# child's anywhere in the parent. Relaying is transitive, so this follows the
# chain; <chain> carries the files already on the path so a cycle cannot
# recurse forever.
#
# A relay is the sibling and `--json` on one line, in either order, rather than
# `--json` immediately after the name. Where the flag sits among the arguments
# is not what makes the child's findings arrive, and reading it as if it were
# made the check blind to the same relay written `check-paths.sh "$dir" --json`.
rules_of() {
  local file="$1"
  local chain="$2"
  local child
  case " $chain " in
    *" $file "*) return 0 ;;
  esac
  rules_emitted_directly_by "$file"
  for child in "$SCRIPTS_DIR"/*.sh; do
    if [[ "$child" == "$file" ]]; then
      continue
    fi
    if grep -hF -- "$(basename "$child")" "$file" | grep -qF -- '--json'; then
      rules_of "$child" "$chain $file"
    fi
  done
}

rules_emitted_by() {
  local file
  for file in "$@"; do
    rules_of "$file" ""
  done | sort -u
}

emitted_rules="$(rules_emitted_by "$SCRIPTS_DIR"/*.sh)"
# Both reads tolerate finding nothing, and both are meant to. "The registry
# line is gone" and "the registry line names no rule" are two of the things the
# assertions below exist to report, and a `grep` that exits 1 under
# `errexit`/`pipefail` would kill the suite at the assignment instead — the
# check unable to report the very state it was written for, which is the defect
# this whole cluster is about.
#
# The anchor is `^Rule IDs:`. It used to be `^Exit codes:`, because the rule
# registry and an exit-code claim shared one sentence — and that sentence
# asserted a single exit contract over five scripts that do not share one. The
# exit contracts are a table now, derived per script and checked below; the rule
# registry keeps its own line and its own anchor, because they were never one
# claim.
registry_line="$(grep -m1 '^Rule IDs:' skills/skill-audit/SKILL.md || true)"
# Any rule-shaped token in backticks, not a fixed list of the prefixes in use. A
# whitelist here would make the registry side unable to grow either: a new rule
# announced under a new prefix would be invisible to the very line that claims
# to name the whole set.
#
# The backticks are what make a registration a registration rather than a word
# that happens to be capitalised. A bare rule-shaped token counted prose:
# writing "the JSON payload" on this line registered a rule called JSON, the
# comparison below went red, and the only way to quiet it was to avoid a capital
# in a sentence of documentation. Rule IDs on that line are set in code because
# they are code, which is a thing the line can carry and a paragraph cannot.
registered_rules="$(printf '%s\n' "$registry_line" \
  | { grep -oE '`[A-Z][A-Z0-9]{2,}`' || true; } | tr -d '`' | sort -u)"

assert_value "SKILL.md's rule-ID line names a rule the scripts emit" "$([[ -n "$registered_rules" ]] && echo true || echo false)"
assert_value "SKILL.md registers every rule ID the scripts emit, and registers no ID none of them emits" \
  "$([[ "$registered_rules" == "$emitted_rules" ]] && echo true || echo false)"
if [[ "$registered_rules" != "$emitted_rules" ]]; then
  echo "  emitted   : $(echo "$emitted_rules" | tr '\n' ' ')"
  echo "  registered: $(echo "$registered_rules" | tr '\n' ' ')"
fi

# Each script's own header registers what that script emits, because a reader
# reaches for the header of the script in front of them before the skill's docs.
for rscript in "$SCRIPTS_DIR"/*.sh; do
  rheader="$(awk 'NR > 1 && !/^#/ && NF { exit } { print }' "$rscript")"
  for rrule in $(rules_emitted_by "$rscript"); do
    assert_value "$(basename "$rscript") header registers $rrule, which it emits" \
      "$(printf '%s\n' "$rheader" | grep -q "$rrule" && echo true || echo false)"
  done
done

# The two IDs no test reached before, pinned where they are produced rather than
# where they are written down.

# PATH — level `unverified`, and not a failure: an unresolvable reference the
# scripts decline to judge is not the same as one they judged and rejected.
run_present "$CHECK_PATHS" --json tests/fixtures/f01/glob-paths
assert_value "paths --json, a glob reference: emits rule PATH at level unverified" \
  "$(echo "$output" | jq -e '.findings[] | select(.rule == "PATH" and .level == "unverified")' >/dev/null 2>&1 && echo true || echo false)"
assert_value "paths --json, a glob reference: an unverified finding is not a failure (exit 0)" \
  "$([[ $code -eq 0 && "$(echo "$output" | jq -r '.passed')" == "true" ]] && echo true || echo false)"

run_present "$CHECK_PATHS" --json tests/fixtures/f01/dynamic-paths
assert_value "paths --json, a variable reference: emits rule PATH at level unverified" \
  "$(echo "$output" | jq -e '.findings[] | select(.rule == "PATH" and .level == "unverified")' >/dev/null 2>&1 && echo true || echo false)"

run_present "$CHECK_STRUCT" --json tests/fixtures/f01/glob-paths
assert_value "structure --json: relays the child's PATH finding at its own level" \
  "$(echo "$output" | jq -e '.findings[] | select(.rule == "PATH" and .level == "unverified")' >/dev/null 2>&1 && echo true || echo false)"

# --- A tool is a precondition only when it answered -----------------------------
#
# `command -v` proves a name resolves. Every fault this cluster closes was of
# the other kind: a jq on PATH that ran and printed nothing left check-paths.sh
# and audit-report.sh exiting 0 with an empty payload channel; a grep that
# answered with an error status turned a clean skill into nine fabricated PL
# failures beside `policy_error: null`; a wc that exited 127 became
# check-structure.sh's own exit status. None of those is absence, and a masked
# PATH cannot reach any of them.
#
# So the cases below drive a tool that is *present and broken*, over the whole
# cross product of the scripts and the tools each one declares. The tool list is
# read out of the script and the probe list out of the guard, so a dependency
# added tomorrow is covered the day it lands rather than the day someone
# remembers to add a case.
#
# A stub here is one directory holding one file, prepended to the real PATH.
# Nothing is mirrored, replaced or uninstalled.

# counting_tool_path <tool> — a PATH whose <tool> forwards to the real one and
# writes down how many times it was asked anything. Used only to find out how
# many questions require_tool's probe puts to a tool, below.
counting_tool_path() {
  local tool="$1"
  local dir="$mask_root/counting-$tool"
  mkdir -p "$dir"
  printf '#!/usr/bin/env bash\nprintf x >> "%s/.asked"\nexec %s "$@"\n' \
    "$dir" "$(command -v "$tool")" > "$dir/$tool"
  chmod +x "$dir/$tool"
  : > "$dir/.asked"
  echo "$dir:$PATH"
}

# probe_calls_of <tool> — how many times require_tool's probe invokes <tool>.
#
# Measured against the guard rather than written down here, because it is the
# guard's number: `grep` is asked twice (a pattern that matches and one that
# must not), every other probe once, and a probe rewritten tomorrow changes it
# without changing this file. The probe-only stub below needs it to know which
# invocation is the first one require_tool did *not* make.
probe_calls_of() {
  local tool="$1"
  local dir="$mask_root/counting-$tool"
  local cpath
  cpath="$(counting_tool_path "$tool")"
  PATH="$cpath" /usr/bin/env bash -c \
    'source skills/skill-audit/scripts/verdict-guard.sh; require_tool "$1" false' \
    _ "$tool" >/dev/null 2>&1 || true
  local asked
  asked="$(wc -c < "$dir/.asked")"
  printf '%s' "${asked//[[:space:]]/}"
}

# broken_tool_path <tool> <mode> — a PATH whose <tool> is present and useless.
#   silent     exit 0, no output           (the jq fault, verbatim)
#   erroring   exit 2                      (the grep fault, verbatim)
#   wrong      exit 0, a confident lie
#
# All three fail the probe, and that is the whole of what was wrong with them:
# `require_tool` stopped every script before a single call site ran, so the
# cross product proved the *precondition* twenty times over and never once
# reached the code the precondition is a precondition for. Twenty-four calls
# read a tool's output without reading its status, and no case here could fail.
# What is missing is a tool that answers and then stops, which is not a
# contrived shape: a wrapper that handles the flags it knows, a build with one
# codec missing, a binary that works until a resource runs out. It is inside
# the threat model this project already states, which is "present and broken"
# rather than "absent". answering_tool_path below is that tool.
#
# Nothing is mirrored, replaced or uninstalled: one directory, one real file,
# prepended to the real PATH.
broken_tool_path() {
  local tool="$1"
  local mode="$2"
  local dir="$mask_root/broken-$tool-$mode"
  if [[ ! -d "$dir" ]]; then
    mkdir -p "$dir"
    case "$mode" in
      silent)   printf '#!/usr/bin/env bash\nexit 0\n' > "$dir/$tool" ;;
      erroring) printf '#!/usr/bin/env bash\nexit 2\n' > "$dir/$tool" ;;
      wrong)    printf '#!/usr/bin/env bash\nprintf %s\nexit 0\n' "'not-an-answer\\n'" > "$dir/$tool" ;;
    esac
    chmod +x "$dir/$tool"
  fi
  echo "$dir:$PATH"
}

# answering_tool_path <tool> <n> — a PATH whose <tool> answers its first <n>
# questions and refuses every one after them.
#
# One of these per call a clean run makes is what turns "the first call site is
# guarded" into "every call site is guarded". A stub that refuses everything
# after the probe only ever reaches the *first* unguarded call, because the
# script stops there — so it proves one site per script and tool and says
# nothing about the ones behind it. Measured on this branch: it proved the
# guard on check-paths.sh's body read and left the code-block read three calls
# later unexamined, and that read named the SKILL.md over a file that was
# perfectly readable with awk nowhere in the sentence.
#
# So the refusal walks the run. `n` runs from the number of questions the probe
# asks up to one short of the number a clean run asks in total, which is a
# derived denominator: every call a clean run makes is driven as the call that
# fails, and adding a call site to a script adds a case here the day it lands.
#
# Every read in the stub is a shell builtin. `cat` is one of the tools this
# stands in for, and a stub that shelled out to `cat` to read its own counter
# would call itself.
answering_tool_path() {
  local tool="$1"
  local answers="$2"
  local dir="$mask_root/answers-$tool-$answers"
  if [[ ! -d "$dir" ]]; then
    mkdir -p "$dir"
    printf '%s\n' \
      '#!/usr/bin/env bash' \
      "asked=\"$dir/.asked\"" \
      'n=0' \
      '[[ -f "$asked" ]] && read -r n < "$asked"' \
      'n=$((n + 1))' \
      'printf %s "$n" > "$asked"' \
      "if (( n <= $answers )); then exec $(command -v "$tool") \"\$@\"; fi" \
      'exit 5' > "$dir/$tool"
    chmod +x "$dir/$tool"
  fi
  # The counter is reset on every call, so each case starts with the tool able
  # to answer again.
  rm -f "$dir/.asked"
  echo "$dir:$PATH"
}

# calls_in_a_clean_run <script> <args> <tool> — how many questions a run that
# reaches its verdict puts to <tool>. The denominator of the walk above.
calls_in_a_clean_run() {
  local script="$1"
  local args="$2"
  local tool="$3"
  local dir="$mask_root/counting-$tool"
  local cpath ctarget asked
  cpath="$(counting_tool_path "$tool")"
  ctarget="$(adverse_target)"
  PATH="$cpath" "$script" ${args//@target/$ctarget} >/dev/null 2>&1 || true
  asked="$(wc -c < "$dir/.asked")"
  printf '%s' "${asked//[[:space:]]/}"
}

# working_tool_path <tool> — the control. A stub that forwards to the real tool
# must be accepted, or every case above would pass because the stub mechanism
# itself breaks the script rather than because the guard caught anything.
working_tool_path() {
  local tool="$1"
  local dir="$mask_root/working-$tool"
  if [[ ! -d "$dir" ]]; then
    mkdir -p "$dir"
    printf '#!/usr/bin/env bash\nexec %s "$@"\n' "$(command -v "$tool")" > "$dir/$tool"
    chmod +x "$dir/$tool"
  fi
  echo "$dir:$PATH"
}

# The probes the guard actually holds, read from its own case arms. A tool the
# guard has no probe for is not asserted here: skill-validator and skillscore
# are sources whose answers are proven where they are read, not by a probe.
guard_probed_tools="$(sed -n '/^tool_answers() {/,/^}$/p' skills/skill-audit/scripts/verdict-guard.sh \
  | sed -n 's/^[[:space:]]*\([a-z][a-z]*\))[[:space:]].*/\1/p' | sort -u)"
echo "  tools the guard probes: $(printf '%s' "$guard_probed_tools" | tr '\n' ' ')"

# tools_required_by <script> — the tools the script states as preconditions.
#
# The line start is not anchored, because a precondition stated after a `&&` or
# inside an `if` is the same precondition. Comment lines are excluded, because
# these files discuss require_tool in prose and `require_tool asks each of them`
# is not a dependency on a tool called `asks`. `[^#]*` cannot cross a `#`, so a
# line whose first non-space character is one contributes nothing.
tools_required_by() {
  { grep -hE '^[[:space:]]*[^#]*require_tool[[:space:]]+[a-z][a-z-]*' "$1" || true; } \
    | sed -E 's/.*require_tool[[:space:]]+([a-z][a-z-]*).*/\1/' | sort -u
}

# Every tool any script requires is either probed here or is one of the two
# audit sources, whose answers are proven where they are read — by
# json_document_conforms in check-frontmatter.sh, and by
# quality_report_conforms in check-quality.sh and audit-report.sh. There is no
# cheap question with a known answer to put to either, and the read is a
# stronger check than a probe would be.
#
# Written as coverage rather than as a list of names to match: a fixed list is
# the defect one level up, and would have to be edited by whoever adds a tool —
# which is the person least likely to notice it needs editing. This way, adding
# `require_tool foo` without a probe or a stated exemption fails here.
GUARD_PROBE_EXEMPT="skill-validator skillscore"
unprobed_tools=""
required_tool_count=0
for tscript in skills/skill-audit/scripts/*.sh skills/skill-rewrite/scripts/*.sh; do
  # verdict-guard.sh defines require_tool; it does not call it. Its
  # verdict_guard_ready name list mentions it beside the next primitive, which
  # reads as a dependency on a tool with that name. The question here is which
  # tools a *caller* requires.
  case "$tscript" in *verdict-guard.sh) continue ;; esac
  for ttool in $(tools_required_by "$tscript"); do
    required_tool_count=$((required_tool_count + 1))
    case "
$guard_probed_tools
" in *"
$ttool
"*) continue ;; esac
    case " $GUARD_PROBE_EXEMPT " in *" $ttool "*) continue ;; esac
    unprobed_tools="$unprobed_tools $ttool($(basename "$tscript"))"
  done
done
echo "  tool preconditions walked: $required_tool_count"
assert_value "the tool preconditions were enumerated, not read as an empty set" \
  "$([[ "$required_tool_count" -ge 15 ]] && echo true || echo false)"
assert_value "every tool a script requires is probed, or is a source proven where it is read" \
  "$([[ -z "$unprobed_tools" ]] && echo true || echo false)"
if [[ -n "$unprobed_tools" ]]; then
  echo "  required with neither a probe nor a stated exemption:$unprobed_tools"
fi

# The scripts the cross product drives are read off the tree, not written down
# here. The hand-written list was four, and the two it left out were the two
# that needed it: `mktemp` is required by `draft-rewrite.sh` alone, and
# `check-quality.sh` is in neither `--json` mode, so the one mechanism built to
# catch a present-and-broken tool never reached either of them. The mktemp probe
# that accepted any answer at all was reachable only through the drafter, and
# nothing drove the drafter. A list of names has to be edited by whoever adds a
# script, who is the person least likely to notice it needs editing — the same
# reason the tool list above is read off the scripts rather than restated.
#
# That paragraph was written above a list of six names. It was three lines of
# argument for deriving the list, followed by the list. The coverage assertion
# below held the literal to the tree, so the set was right; what was wrong was
# that the next reader would believe the comment instead of reading the three
# lines under it, and would add a script expecting it to be picked up.
#
# It is read off the tree now. A script arriving in either skill enters the
# cross product the day it lands, and the half of the coverage check that
# compared the literal against the tree is gone with the literal, because an
# assertion that cannot fail is the thing this suite exists to refuse. The half
# that remains is the one still capable of failing: a script driven here with
# no invocation that reaches a verdict.
#
# Runnable is the criterion, as it was for the coverage check: verdict-guard.sh
# is sourced rather than run, and a file without the executable bit is not
# something a caller invokes.
ADVERSE_SCRIPTS=""
for ascript in "$SCRIPTS_DIR"/*.sh skills/skill-rewrite/scripts/*.sh; do
  case "$ascript" in *verdict-guard.sh) continue ;; esac
  [[ -x "$ascript" ]] || continue
  ADVERSE_SCRIPTS="$ADVERSE_SCRIPTS $ascript"
done

# --- What a script can be made to exit with, against what it says ---------------
#
# The exit-table derivation below this compares each script's header against
# SKILL.md. That is one half of the question, and it is the half that needs a
# doc: it loops the audit scripts only, because skill-audit's SKILL.md is the
# only doc with a table. So `draft-rewrite.sh` states an exit contract that
# nothing anywhere compares against — which is why F1 and F6 went unseen. Both
# are statuses outside its stated set, and no case could have failed.
#
# The other half needs no doc at all: a script's own `# Exit codes:` header
# against the statuses it can actually be made to emit. That is what closes the
# class rather than the two instances, it covers both skills, and it is asserted
# here over every adverse condition this suite drives.
#
# exit_line_of <script>     — the script's own statement of its exit contract.
# stated_exit_codes <script> — the numbers in it.
exit_line_of() {
  sed -n 's/^# Exit codes:[[:space:]]*//p' "$1" | head -1
}

stated_exit_codes() {
  exit_line_of "$1" | grep -oE '(^|[^0-9])[0-9]+=' | grep -oE '[0-9]+' | sort -u
}

# english_count <n> — the number written the way these docs write it. Defined
# once because two derivations below compare a word in prose against a count,
# and two private copies of a number-word list is one list that can drift.
english_count() {
  printf '%s\n' zero one two three four five six seven eight nine ten eleven twelve \
    | sed -n "$(($1 + 1))p"
}

# states_exit <script> <code> — true when <code> is in that script's set.
states_exit() {
  local wanted="$2"
  local c
  for c in $(stated_exit_codes "$1"); do
    [[ "$c" == "$wanted" ]] && return 0
  done
  return 1
}

# The helper is read against a script whose contract is known, so a parse that
# silently produced the empty set would not pass as "every status conformed".
assert_value "the exit contract of audit-report.sh parses to the set its header states" \
  "$([[ "$(stated_exit_codes "$SCRIPTS_DIR/audit-report.sh" | tr '\n' ' ')" == "0 3 " ]] && echo true || echo false)"
assert_value "the exit contract of draft-rewrite.sh parses to the set its header states" \
  "$([[ "$(stated_exit_codes skills/skill-rewrite/scripts/draft-rewrite.sh | tr '\n' ' ')" == "0 1 3 " ]] && echo true || echo false)"
assert_value "a status outside a stated set is refused, so the check can fail" \
  "$(states_exit "$SCRIPTS_DIR/audit-report.sh" 2 && echo false || echo true)"
assert_value "a status inside a stated set is accepted, so the check is not refusing everything" \
  "$(states_exit "$SCRIPTS_DIR/audit-report.sh" 3 && echo true || echo false)"


# adverse_args_of <basename> — the invocation that reaches a verdict, with
# `@target` standing for the skill directory. A runnable script with no entry
# fails the coverage assertion below instead of being quietly skipped.
adverse_args_of() {
  case "$1" in
    check-structure.sh|check-paths.sh) echo "--json @target" ;;
    check-frontmatter.sh|check-quality.sh|audit-report.sh|check-extra.sh) echo "@target" ;;
    draft-rewrite.sh) echo "-t @target" ;;
    *) return 1 ;;
  esac
}

# Every script the cross product will drive has an invocation that reaches a
# verdict. Membership is no longer a question — the list is the tree — so what
# is asked here is the half that can still be answered no: a script with no
# entry in adverse_args_of would be driven with no arguments, reach its usage
# path, and prove nothing about a broken tool.
adverse_uncovered=""
adverse_scripts_seen=0
for ascript in $ADVERSE_SCRIPTS; do
  adverse_scripts_seen=$((adverse_scripts_seen + 1))
  adverse_args_of "$(basename "$ascript")" >/dev/null \
    || adverse_uncovered="$adverse_uncovered $(basename "$ascript")(no-invocation)"
done
echo "  runnable scripts read off the tree: $adverse_scripts_seen"
assert_value "the runnable scripts were read off the tree, not matched as an empty set" \
  "$([[ "$adverse_scripts_seen" -eq 6 ]] && echo true || echo false)"
assert_value "every runnable script in both skills has an invocation that reaches a verdict" \
  "$([[ -z "$adverse_uncovered" ]] && echo true || echo false)"
if [[ -n "$adverse_uncovered" ]]; then
  echo "  driven against a broken tool with no invocation to drive:$adverse_uncovered"
fi

# A fresh target per case. `draft-rewrite.sh` writes its draft into the target
# it was handed, so a shared copy would carry the previous case's draft into the
# next one. The copy keeps the fixture's own directory name because
# skill-validator checks `name` against it, and a renamed copy would fail the
# control for a reason that has nothing to do with the tool under test.
adverse_target() {
  local root="$mask_root/adverse"
  rm -rf "$root"
  mkdir -p "$root"
  cp -R tests/fixtures/f01/valid-full "$root/valid-full"
  echo "$root/valid-full"
}

# The break modes, named once so the count below and the loop cannot disagree
# about how many there are.
# assert_refused <script> <name> <tool> <why> <json> <target>
#
# What a script owes its caller when a tool it computes with will not answer,
# whichever way it will not answer and at whichever call. One statement of it,
# because it is one statement: the three fixed modes and the walk across every
# call position are the same contract driven from different places, and two
# copies of it is one copy that drifts.
assert_refused() {
  local rscript="$1"
  local rname="$2"
  local rtool="$3"
  local rwhy="$4"
  local rjson="$5"
  local rtarget="$6"

  assert_value "$rname, $rtool $rwhy: exits 3, not a status meaning a verdict" \
    "$([[ $code -eq 3 ]] && echo true || echo false)"
  assert_value "$rname, $rtool $rwhy: exits inside the set its own header states" \
    "$(states_exit "$rscript" "$code" && echo true || echo false)"
  assert_value "$rname, $rtool $rwhy: never reports passed true" \
    "$(echo "$output" | grep -qE '"passed":[[:space:]]*true' && echo false || echo true)"
  assert_value "$rname, $rtool $rwhy: the diagnostic names $rtool, not another component" \
    "$(echo "$errout" | grep -q -- "$rtool" && echo true || echo false)"
  # A script that produces a document must not have produced one. The drafter
  # is the case that matters: a tool it could not use left it writing a draft
  # anyway, at exit 0, and the draft is what a reader then treats as the
  # audit's findings. Driven at every call position, this is also what says the
  # draft is not left half-written when the tool stops in the middle of it.
  if [[ "$rname" == draft-rewrite.sh ]]; then
    assert_value "$rname, $rtool $rwhy: no draft was written" \
      "$([[ ! -e "$rtarget/REWRITE-DRAFT.md" ]] && echo true || echo false)"
  fi
  if [[ -n "$rjson" ]]; then
    assert_value "$rname, $rtool $rwhy: the payload carries DEP002" \
      "$(echo "$output" | jq -e '.findings[] | select(.rule == "DEP002")' >/dev/null 2>&1 && echo true || echo false)"
    # stdout is the payload channel, so it carries the payload and nothing
    # else. A broken tool does not honour `-q`: a grep stub that printed a word
    # and exited 0 put that word on this channel ahead of the payload, so the
    # guard that caught the fault corrupted the report of it.
    assert_value "$rname, $rtool $rwhy: stdout carries exactly the payload and nothing else" \
      "$(printf '%s' "$output" | jq -se 'length == 1' >/dev/null 2>&1 && echo true || echo false)"
  else
    assert_value "$rname, $rtool $rwhy: stdout carries no verdict at all" \
      "$([[ -z "$output" ]] && echo true || echo false)"
  fi
}

# The three modes that fail the probe, named once so the count below and the
# loop cannot disagree about how many there are.
BROKEN_MODES="silent erroring wrong"
broken_modes_n=0
for bmode in $BROKEN_MODES; do broken_modes_n=$((broken_modes_n + 1)); done

broken_cases=0
answering_cases=0
for bscript in $ADVERSE_SCRIPTS; do
  bname="$(basename "$bscript")"
  # A script with no invocation is reported by the coverage assertion above; it
  # is skipped rather than driven with no arguments, and rather than taking the
  # suite down at this assignment under errexit before it can report anything.
  bargs=""
  bargs="$(adverse_args_of "$bname")" || bargs=""
  [[ -n "$bargs" ]] || continue
  for btool in $(tools_required_by "$bscript"); do
    # Only the tools the guard has a probe for; the rest are proven at their read.
    case "
$guard_probed_tools
" in
      *"
$btool
"*) ;;
      *) continue ;;
    esac
    bjson=""
    case "$bargs" in *--json*) bjson="--json" ;; esac
    for bmode in $BROKEN_MODES; do
      broken_cases=$((broken_cases + 1))
      btarget="$(adverse_target)"
      run_on_path "$(broken_tool_path "$btool" "$bmode")" "$bscript" ${bargs//@target/$btarget}
      assert_refused "$bscript" "$bname" "$btool" "present but $bmode" "$bjson" "$btarget"
    done
    # And the walk: every question a clean run puts to this tool, driven as the
    # question it refuses. The first of them is require_tool's probe answered
    # and the first real call refused; the last is every call answered but the
    # final one. A call site added to this script joins the walk the day it
    # lands, because the bound is measured rather than written down.
    bprobe="$(probe_calls_of "$btool")"
    bcalls="$(calls_in_a_clean_run "$bscript" "$bargs" "$btool")"
    assert_value "$bname asks $btool a countable number of questions, and more than the probe does" \
      "$([[ "$bcalls" -gt "$bprobe" ]] && echo true || echo false)"
    bn="$bprobe"
    while [[ "$bn" -lt "$bcalls" ]]; do
      broken_cases=$((broken_cases + 1))
      answering_cases=$((answering_cases + 1))
      btarget="$(adverse_target)"
      run_on_path "$(answering_tool_path "$btool" "$bn")" "$bscript" ${bargs//@target/$btarget}
      assert_refused "$bscript" "$bname" "$btool" "answering only its first $bn of $bcalls questions" "$bjson" "$btarget"
      bn=$((bn + 1))
    done
    # The control, per tool: forwarded to the real thing, the verdict is reached.
    btarget="$(adverse_target)"
    run_on_path "$(working_tool_path "$btool")" "$bscript" ${bargs//@target/$btarget}
    assert_value "$bname, $btool forwarded to the real tool: reaches its verdict (exit 0)" \
      "$([[ $code -eq 0 ]] && echo true || echo false)"
    # And the control on the walk's own bound: a stub that answers every
    # question a clean run asks must reach the verdict too, or the cases above
    # would be passing because the stub mechanism breaks the script.
    btarget="$(adverse_target)"
    run_on_path "$(answering_tool_path "$btool" "$bcalls")" "$bscript" ${bargs//@target/$btarget}
    assert_value "$bname, $btool answering all $bcalls of its questions: reaches its verdict (exit 0)" \
      "$([[ $code -eq 0 ]] && echo true || echo false)"
  done
done

echo "  present-but-broken cases driven: $broken_cases ($adverse_scripts_seen runnable scripts, $broken_modes_n probe-failing modes, $answering_cases refusal positions)"
echo "  probe calls per tool: $(for pt in $guard_probed_tools; do printf '%s=%s ' "$pt" "$(probe_calls_of "$pt")"; done)"
# The probe-only stub is built around this number, so a measurement that came
# back as zero would build a stub that refuses its own probe — every case would
# still exit 3, for the wrong reason, and the mode would be testing nothing.
probe_call_floor=1
for pt in $guard_probed_tools; do
  [[ "$(probe_calls_of "$pt")" -ge 1 ]] || probe_call_floor=0
done
assert_value "every probed tool is asked at least one question by require_tool, so the probe-only stub has a probe to answer" \
  "$([[ "$probe_call_floor" -eq 1 ]] && echo true || echo false)"
# The denominator, and it is an equality on purpose. `-ge 45` was written when
# the product was 60, and a floor is not a denominator: the whole reason for
# counting is that the product can shrink without anything else here noticing,
# and 45 let it lose a quarter of itself and still say it had been enumerated.
# Measured — `require_tool grep` deleted from one script, which is one pair and
# four cases:
#
#   present-but-broken cases driven: 68
#   PASS: the present-but-broken cross product was enumerated, not read as empty
#   1232 passed, 0 failed
#
# Twenty-one assertions gone and the suite fully green. An exact count is a
# number someone has to change deliberately, in the commit that changed the
# product, which is the only moment anyone can say whether the change was
# meant.
#
# The other floors in this file are not this shape and are left as they are:
# each sits beside an exact comparison that does the real work — the readme's
# list against the scripts' in both directions, the probed tools against the
# required ones, the doc's rows against the headers — so the floor there is
# only refusing an empty read. Nothing else counts these cases.
BROKEN_CASES_EXPECTED=173
assert_value "the present-but-broken cross product ran every one of its $BROKEN_CASES_EXPECTED cases" \
  "$([[ "$broken_cases" -eq "$BROKEN_CASES_EXPECTED" ]] && echo true || echo false)"
if [[ "$broken_cases" -ne "$BROKEN_CASES_EXPECTED" ]]; then
  echo "  the cross product is $broken_cases cases, and this file says $BROKEN_CASES_EXPECTED"
fi

# --- What a probe that accepts any answer actually costs ------------------------
#
# The three assertions above say exit 3, no pass, and the right tool named. They
# do not say where the answer went. `mktemp` was the one arm of tool_answers
# that compared against nothing — it asked only that there *was* an answer —
# and the answer is not a diagnostic: it is a path, which the drafter opens,
# writes an audit into, reads back, and states in the draft as that draft's
# provenance. So a mktemp answering `not-an-answer` at exit 0 passed the probe
# and the drafter created a file called `not-an-answer` in whatever directory
# the caller happened to be standing in, then published a draft at exit 0
# naming it as the audit it was composed from.
#
# The case is driven from a directory of its own, because the defect is defined
# by where the file lands. Running it from the repository root would put the
# stray file in the repository.

# run_in_dir <dir> <path> <cmd> [args...] — run_on_path, from a given cwd.
run_in_dir() {
  local dir="$1"
  shift
  local use_path="$1"
  shift
  local errfile="$mask_root/stderr"
  code=0
  output=$(cd "$dir" && PATH="$use_path" "$@" 2>"$errfile") || code=$?
  errout="$(cat "$errfile")"
}

mktemp_answer="not-an-answer"
mktemp_cwd="$mask_root/mktemp-caller-cwd"
rm -rf "$mktemp_cwd"
mkdir -p "$mktemp_cwd"
mktemp_target="$(adverse_target)"
DRAFT_ABS="$(CDPATH= cd -P -- skills/skill-rewrite/scripts && pwd -P)/draft-rewrite.sh"
run_in_dir "$mktemp_cwd" "$(broken_tool_path mktemp wrong)" "$DRAFT_ABS" -t "$mktemp_target"

assert_value "drafter, mktemp answers with a name it did not make: exits 3, not 0" \
  "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "drafter, mktemp answers with a name it did not make: names mktemp on stderr" \
  "$(echo "$errout" | grep -q 'mktemp' && echo true || echo false)"
assert_value "drafter, mktemp answers with a name it did not make: no draft was written" \
  "$([[ ! -f "$mktemp_target/REWRITE-DRAFT.md" ]] && echo true || echo false)"
assert_value "drafter, mktemp answers with a name it did not make: nothing is created in the caller's directory" \
  "$([[ -z "$(ls -A "$mktemp_cwd")" ]] && echo true || echo false)"
assert_value "drafter, mktemp answers with a name it did not make: no draft claims that answer as its provenance" \
  "$(grep -rq "$mktemp_answer" "$mktemp_target" 2>/dev/null && echo false || echo true)"

# The probe itself, asked directly, over every way a tool can answer wrongly.
# Each of these satisfies "there is an answer" and none of them is one.
mktemp_probe_cases=0
for mcase in "wrong|an answer that is not a name it made" \
             "silent|no answer at all" \
             "erroring|a status instead of an answer"; do
  mmode="${mcase%%|*}"
  mwhy="${mcase#*|}"
  mktemp_probe_cases=$((mktemp_probe_cases + 1))
  run_on_path "$(broken_tool_path mktemp "$mmode")" \
    /usr/bin/env bash -c 'source skills/skill-audit/scripts/verdict-guard.sh; tool_answers mktemp'
  assert_value "the mktemp probe refuses $mwhy" \
    "$([[ $code -ne 0 ]] && echo true || echo false)"
done
echo "  mktemp probe cases driven: $mktemp_probe_cases"
assert_value "the mktemp probe cases were enumerated, not read as empty" \
  "$([[ "$mktemp_probe_cases" -eq 3 ]] && echo true || echo false)"

# The control: the real mktemp answers its own probe, so the cases above fail
# because the probe now reads the answer and not because it refuses everything.
run_on_path "$(working_tool_path mktemp)" \
  /usr/bin/env bash -c 'source skills/skill-audit/scripts/verdict-guard.sh; tool_answers mktemp'
assert_value "the mktemp probe accepts the real mktemp" \
  "$([[ $code -eq 0 ]] && echo true || echo false)"

# --- A temp directory the real mktemp cannot write in ---------------------------
#
# No stub is involved. The drafter asks mktemp for a file under `$TMPDIR`, and
# `mktemp` exits 1 when it cannot make one there — a TMPDIR that does not exist,
# or one it may not write to. Exit 1 is "usage or target error" in this script's
# own contract, so a broken temp directory arrived as a statement about the
# caller's command line, with no sentence naming mktemp and no "no draft was
# written". The status was never read.
#
# Driven both ways round, because they fail at different depths: a directory
# that is not there, and one that is there and refuses the write.
mktemp_env_cases=0
for tcase in "$mask_root/tmpdir-that-does-not-exist|a TMPDIR that does not exist" \
             "$mask_root/tmpdir-unwritable|a TMPDIR that cannot be written to"; do
  ttmp="${tcase%%|*}"
  twhy="${tcase#*|}"
  mktemp_env_cases=$((mktemp_env_cases + 1))
  rm -rf "$ttmp"
  case "$twhy" in
    *"cannot be written"*) mkdir -p "$ttmp"; chmod 500 "$ttmp" ;;
  esac
  ttarget="$(adverse_target)"
  code=0
  errout="$(TMPDIR="$ttmp" "$DRAFT_ABS" -t "$ttarget" 2>&1 >/dev/null)" || code=$?
  assert_value "drafter, $twhy: exits 3, not 1 — a broken temp directory is not the caller's usage" \
    "$([[ $code -eq 3 ]] && echo true || echo false)"
  assert_value "drafter, $twhy: names mktemp as the thing that could not answer" \
    "$(echo "$errout" | grep -q 'mktemp' && echo true || echo false)"
  # The sentence here is the guard's, not this script's: a tool precondition is
  # refused by require_tool, which says "no verdict was computed" in the same
  # words for all six callers and does not know that this one's product is a
  # draft. Making that message caller-specific would put six spellings of one
  # refusal back where the guard exists to hold one. What this path owes the
  # caller is a refusal that names the tool and a target with no draft in it;
  # the "no draft was written" sentence is asserted below, on the path that
  # belongs to this script.
  assert_value "drafter, $twhy: stated a refusal rather than dying silently" \
    "$(echo "$errout" | grep -q 'ERROR:' && echo true || echo false)"
  assert_value "drafter, $twhy: no draft was written" \
    "$([[ ! -f "$ttarget/REWRITE-DRAFT.md" ]] && echo true || echo false)"
  [[ -d "$ttmp" ]] && chmod 700 "$ttmp"
done
echo "  mktemp temp-directory cases driven: $mktemp_env_cases"
assert_value "the mktemp temp-directory cases were enumerated, not read as empty" \
  "$([[ "$mktemp_env_cases" -eq 2 ]] && echo true || echo false)"

# --- "No draft was written" has to be true of the draft, not of the reads -------
#
# The drafter says "no draft was written" on every path that refuses to finish,
# and the sentence was not true. Composing a draft is eight writes, and the
# checks that could stop it all ran before the first of them — which is a fine
# ordering and is not the same statement. `-a` naming a file that exists and
# cannot be read passes the `-f` test, so no temp audit is made; the header is
# written; `cat` then fails on the audit; errexit spends exit 1, which this
# contract reserves for the caller's own mistake; and a six-line REWRITE-DRAFT.md
# is left in the caller's skill directory with a raw `Permission denied` as the
# only thing said about it.
#
# This is the whole shape of it with real tools and no stub at all, which is why
# it is driven here as well as through the cross product above.
draft_unreadable_audit="$mask_root/unreadable-audit.md"
printf '%s\n' 'an audit nobody can read' > "$draft_unreadable_audit"
chmod 000 "$draft_unreadable_audit"
# Under a user that bypasses file permissions there is nothing here to test, and
# a case that quietly tests nothing is what this suite exists to refuse. So the
# premise is asserted rather than assumed.
assert_value "the unreadable audit report is genuinely unreadable to this user" \
  "$([[ ! -r "$draft_unreadable_audit" ]] && echo true || echo false)"

unreadable_target="$(adverse_target)"
code=0
errout="$("$DRAFT_ABS" -t "$unreadable_target" -a "$draft_unreadable_audit" 2>&1 >/dev/null)" || code=$?
assert_value "drafter, an audit report it cannot read: exits 3, not 1 — an unreadable source is not the caller's usage" \
  "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "drafter, an audit report it cannot read: exits inside the set its own header states" \
  "$(states_exit skills/skill-rewrite/scripts/draft-rewrite.sh "$code" && echo true || echo false)"
assert_value "drafter, an audit report it cannot read: names cat as the thing that could not answer" \
  "$(echo "$errout" | grep -q 'cat' && echo true || echo false)"
assert_value "drafter, an audit report it cannot read: says no draft was written" \
  "$(echo "$errout" | grep -q 'no draft was written' && echo true || echo false)"
assert_value "drafter, an audit report it cannot read: leaves no draft behind, not even a partial one" \
  "$([[ ! -e "$unreadable_target/REWRITE-DRAFT.md" ]] && echo true || echo false)"
chmod 600 "$draft_unreadable_audit"

# The control, and it is what makes the case above a case: the same invocation
# with the same audit report readable writes the draft and exits 0. Without it,
# a drafter that refused every `-a` would pass every assertion above.
readable_target="$(adverse_target)"
code=0
output="$("$DRAFT_ABS" -t "$readable_target" -a "$draft_unreadable_audit" 2>/dev/null)" || code=$?
assert_value "drafter, the same audit report readable: writes the draft and exits 0" \
  "$([[ $code -eq 0 && -f "$readable_target/REWRITE-DRAFT.md" ]] && echo true || echo false)"
assert_value "drafter, the same audit report readable: the draft carries what the audit said" \
  "$(grep -q 'an audit nobody can read' "$readable_target/REWRITE-DRAFT.md" && echo true || echo false)"

# --- The call site, which the probe cannot stand in for -------------------------
#
# The probe now refuses a temp directory mktemp cannot write in, so the two
# cases above are caught before the drafter asks for its file. That is one
# layer, and it is the wrong one to rely on: a precondition is checked once, and
# the directory can stop being writable between the check and the call. A
# precondition proved earlier is not a status read later, and the status is what
# the script's own exit is made of under errexit.
#
# So the call site is driven on its own, with the only stub that can reach it: a
# mktemp that answers the probe correctly — it forwards `-u` to the real tool —
# and fails the call that actually makes the file. Nothing else gets past
# require_tool to the line under test.
mktemp_late_dir="$mask_root/broken-mktemp-late"
if [[ ! -d "$mktemp_late_dir" ]]; then
  mkdir -p "$mktemp_late_dir"
  {
    printf '#!/usr/bin/env bash\n'
    printf '# Answers the probe, refuses the real request.\n'
    printf 'if [[ "${1:-}" == -u ]]; then exec %s "$@"; fi\n' "$(command -v mktemp)"
    printf 'echo "mktemp: cannot create a file there" >&2\n'
    printf 'exit 1\n'
  } > "$mktemp_late_dir/mktemp"
  chmod +x "$mktemp_late_dir/mktemp"
fi

# The stub is the control for itself: the probe must accept it, or the case
# below would be testing require_tool over again rather than the call site.
run_on_path "$mktemp_late_dir:$PATH" \
  /usr/bin/env bash -c 'source skills/skill-audit/scripts/verdict-guard.sh; tool_answers mktemp'
assert_value "the late-failing mktemp stub does answer the probe, so the case below reaches the call site" \
  "$([[ $code -eq 0 ]] && echo true || echo false)"

late_target="$(adverse_target)"
run_on_path "$mktemp_late_dir:$PATH" "$DRAFT_ABS" -t "$late_target"
assert_value "drafter, mktemp fails the call it passed the probe for: exits 3, not 1" \
  "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "drafter, mktemp fails the call it passed the probe for: names mktemp" \
  "$(echo "$errout" | grep -q 'mktemp' && echo true || echo false)"
assert_value "drafter, mktemp fails the call it passed the probe for: says no draft was written" \
  "$(echo "$errout" | grep -q 'no draft was written' && echo true || echo false)"
assert_value "drafter, mktemp fails the call it passed the probe for: and none was" \
  "$([[ ! -f "$late_target/REWRITE-DRAFT.md" ]] && echo true || echo false)"
assert_value "drafter, mktemp fails the call it passed the probe for: stdout carries no verdict" \
  "$([[ -z "$output" ]] && echo true || echo false)"

# And the other half of the same status: a mktemp that exits 0 saying nothing.
# An empty answer is a path the shell would open as the empty string, so the
# status alone is not the whole question the call site has to ask.
mktemp_empty_dir="$mask_root/broken-mktemp-empty"
if [[ ! -d "$mktemp_empty_dir" ]]; then
  mkdir -p "$mktemp_empty_dir"
  {
    printf '#!/usr/bin/env bash\n'
    printf 'if [[ "${1:-}" == -u ]]; then exec %s "$@"; fi\n' "$(command -v mktemp)"
    printf 'exit 0\n'
  } > "$mktemp_empty_dir/mktemp"
  chmod +x "$mktemp_empty_dir/mktemp"
fi
empty_target="$(adverse_target)"
run_on_path "$mktemp_empty_dir:$PATH" "$DRAFT_ABS" -t "$empty_target"
assert_value "drafter, mktemp exits 0 naming no file: exits 3" \
  "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "drafter, mktemp exits 0 naming no file: names mktemp" \
  "$(echo "$errout" | grep -q 'mktemp' && echo true || echo false)"
assert_value "drafter, mktemp exits 0 naming no file: no draft was written" \
  "$([[ ! -f "$empty_target/REWRITE-DRAFT.md" ]] && echo true || echo false)"

# The control: a writable TMPDIR of its own, and the drafter reaches its verdict.
mktemp_ok_tmp="$mask_root/tmpdir-writable"
mkdir -p "$mktemp_ok_tmp"
mktemp_ok_target="$(adverse_target)"
code=0
TMPDIR="$mktemp_ok_tmp" "$DRAFT_ABS" -t "$mktemp_ok_target" >/dev/null 2>&1 || code=$?
assert_value "drafter, a writable TMPDIR: reaches its verdict (exit 0)" \
  "$([[ $code -eq 0 ]] && echo true || echo false)"
assert_value "drafter, a writable TMPDIR: wrote the draft" \
  "$([[ -f "$mktemp_ok_target/REWRITE-DRAFT.md" ]] && echo true || echo false)"
assert_value "drafter, a writable TMPDIR: left no temporary audit behind in it" \
  "$([[ -z "$(ls -A "$mktemp_ok_tmp")" ]] && echo true || echo false)"

# --- The externals no require_tool covers -------------------------------------
#
# The same root as F1 and F2, in the three places the guard structurally cannot
# reach. An external invoked outside require_tool's coverage leaks its own
# status as the script's exit status under errexit, with no diagnostic and no
# verdict — because it runs before the precondition is stated, or because it was
# reasoned off the list on a premise that covers only its exit-0 failures.
#
# `date` was reasoned off: the comment says a clock that failed leaves a visibly
# empty timestamp beside findings that are all still true. That is correct for
# the exit-0 rows and says nothing about a nonzero exit, which is the row that
# exists. `dirname` and `cat` cannot be routed through the guard at all —
# `dirname` runs before the guard is loaded and `cat` before the arguments are
# parsed — so each needs the explicit refusal the guard load-check three lines
# below the first one already uses.

# What the sweeps below prove, recorded as they prove it. A tool is a hard
# precondition of these scripts when a script that cannot get an answer from it
# refuses to run, names it, and exits inside its own stated set — which is the
# readme's own criterion for listing one, and is exactly what each sweep here
# drives. The readme's census reads this variable, so the two halves of
# "required" — stated through require_tool, and stated by an explicit refusal
# where require_tool cannot reach — are one list rather than one list and a
# blind spot.
unguarded_proved=""

# One directory, one file, prepended to the real PATH. Nothing is mirrored,
# replaced or uninstalled.
unguarded_stub_path() {
  local tool="$1"
  local status="$2"
  local dir="$mask_root/unguarded-$tool-$status"
  if [[ ! -d "$dir" ]]; then
    mkdir -p "$dir"
    printf '#!/usr/bin/env bash\nexit %s\n' "$status" > "$dir/$tool"
    chmod +x "$dir/$tool"
  fi
  echo "$dir:$PATH"
}

# F4 — `date` in audit-report.sh. Its status was unread, so a nonzero clock
# became this script's own exit: 2, which is outside the {0, 3} its header
# states, with zero bytes of report and not one word on stderr.
date_cases=0
for dstatus in 1 2 127; do
  date_cases=$((date_cases + 1))
  run_on_path "$(unguarded_stub_path date "$dstatus")" \
    "$SCRIPTS_DIR/audit-report.sh" tests/fixtures/f01/valid-full
  assert_value "audit-report, date exits $dstatus: exits inside the set its own header states" \
    "$(states_exit "$SCRIPTS_DIR/audit-report.sh" "$code" && echo true || echo false)"
  assert_value "audit-report, date exits $dstatus: says which tool could not answer" \
    "$(echo "$errout" | grep -q 'date' && echo true || echo false)"
  assert_value "audit-report, date exits $dstatus: does not exit 0 without a report" \
    "$([[ $code -eq 0 && -z "$output" ]] && echo false || echo true)"
done
echo "  unguarded date cases driven: $date_cases"
assert_value "the unguarded date cases were enumerated, not read as empty" \
  "$([[ "$date_cases" -eq 3 ]] && echo true || echo false)"
# Recorded beside the assertions that are its proof, not written down again
# somewhere the proof cannot be seen.
unguarded_proved="$unguarded_proved date"

# The control: the real clock, and the report carries a timestamp.
run_on_path "$(working_tool_path date)" "$SCRIPTS_DIR/audit-report.sh" tests/fixtures/f01/valid-full
assert_value "audit-report, the real date: reaches its report (exit 0)" \
  "$([[ $code -eq 0 ]] && echo true || echo false)"
assert_value "audit-report, the real date: the report carries the timestamp it read" \
  "$(echo "$output" | jq -e '.timestamp | test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T")' >/dev/null 2>&1 && echo true || echo false)"

# F5 — `dirname` in the `script_dir=` block of all six scripts. It runs before
# the guard is loaded, so a broken one left every script exiting 1 with bash's
# own `cd:` message and nothing else. For check-paths.sh that 1 is "path
# failure" and for check-frontmatter.sh "spec failure" — a fabricated verdict;
# for audit-report.sh and check-quality.sh it is outside the stated set
# entirely, and check-quality.sh's block is new in this diff.
dirname_cases=0
for dscript in $ADVERSE_SCRIPTS; do
  dname="$(basename "$dscript")"
  dargs="$(adverse_args_of "$dname")"
  dtarget="$(adverse_target)"
  dirname_cases=$((dirname_cases + 1))
  run_on_path "$(unguarded_stub_path dirname 1)" "$dscript" ${dargs//@target/$dtarget}
  assert_value "$dname, dirname exits 1: exits inside the set its own header states" \
    "$(states_exit "$dscript" "$code" && echo true || echo false)"
  assert_value "$dname, dirname exits 1: says it could not resolve its own directory" \
    "$(echo "$errout" | grep -q 'ERROR:' && echo true || echo false)"
  assert_value "$dname, dirname exits 1: reports no verdict on stdout" \
    "$(echo "$output" | grep -qE '"passed":[[:space:]]*true|frontmatter OK|Rewrite draft written' && echo false || echo true)"
done
echo "  unguarded dirname cases driven: $dirname_cases"
assert_value "the unguarded dirname cases were enumerated, not read as empty" \
  "$([[ "$dirname_cases" -eq 6 ]] && echo true || echo false)"
unguarded_proved="$unguarded_proved dirname"

# --- The readme's prerequisite list is the scripts' own ------------------------
#
# Placed here, below every sweep above, because that is where its evidence is.
# The readme's criterion for listing a tool is behavioural — "a script that
# cannot get an answer from one of them exits 3 and says which" — and this
# derived the list from `require_tool` alone, which is a mechanism rather than
# a criterion. Two tools meet the criterion outside that mechanism and were
# therefore absent: `dirname`, which runs before the guard exists to announce
# it, and `date`, which has no constant answer for a probe to compare against.
# Both became hard preconditions in this branch, by the fix that gave each an
# explicit refusal — so the very change that made them required is the change
# that moved them out of sight of the list. The count was short by two.
#
# So the list is `require_tool` plus what the sweeps above proved, and "proved"
# is meant literally: each name in `unguarded_proved` is recorded beside the
# assertions that drove that tool broken and watched the script refuse, name it
# and exit inside its own stated set. A name cannot be added to that variable
# and stay true without those assertions holding.
#
# The boundary, stated rather than implied: this finds an unguarded tool that
# something drove, not one nobody thought of. Closing that needs the set of
# external command words in each script, which is a shell parser, and is not
# what this suite does.
#
# The readme said "The skills shell out to three external tools" and listed
# skill-validator, skillscore and jq. That was true when the only hard
# preconditions were those three. This diff made seven more of them hard: every
# tool a script now routes through require_tool exits 3 when it is absent or
# present and not answering, which is the whole point of the change.
#
# The trade is honest in the code and stated in each script's header. The
# user-facing table is where it went wrong, and it went wrong in the direction
# that matters: a reader was told a tool was optional when a script now refuses
# to run without it. So the claim is derived here rather than restated there —
# the same mechanism as the rule-ID census and the exit table, for the same
# reason, because a prose count is exactly the thing that drifts.
#
# The anchor is `^Required tools:` on its own line, holding every tool in
# backticks. Both directions are checked: a tool a script requires and the
# readme omits, and a tool the readme claims and nothing requires.
# `|| true` on the extraction, not on the comparison: an anchor that is missing
# entirely must reach the assertions as an empty set and fail them, rather than
# taking the suite down under pipefail before it can report anything. The
# "was read, not matched as an empty set" assertion below is what refuses the
# empty reading.
readme_anchor="$(sed -n 's/^Required tools:[[:space:]]*//p' README.md | head -1)"
readme_required="$({ printf '%s' "$readme_anchor" | grep -oE '`[a-z][a-z-]*`' || true; } \
  | tr -d '`' | sort -u)"

scripts_required="$unguarded_proved"
for pscript in "$SCRIPTS_DIR"/*.sh skills/skill-rewrite/scripts/*.sh; do
  case "$pscript" in *verdict-guard.sh) continue ;; esac
  scripts_required="$scripts_required $(tools_required_by "$pscript" | tr '\n' ' ')"
done
scripts_required="$({ printf '%s' "$scripts_required" | tr ' ' '\n' | grep -v '^$' || true; } | sort -u)"
readme_required_n="$({ printf '%s' "$readme_required" | grep -c . || true; } | tr -d ' ')"

readme_missing=""
for ptool in $scripts_required; do
  case "
$readme_required
" in *"
$ptool
"*) ;; *) readme_missing="$readme_missing $ptool" ;; esac
done

readme_stale=""
for ptool in $readme_required; do
  case "
$scripts_required
" in *"
$ptool
"*) ;; *) readme_stale="$readme_stale $ptool" ;; esac
done

echo "  tools the readme lists as prerequisites: $(printf '%s' "$readme_required" | tr '\n' ' ')"
echo "  tools the scripts require: $(printf '%s' "$scripts_required" | tr '\n' ' ')"
echo "  of those, required outside require_tool and proved so above:$unguarded_proved"
# A census that silently lost its second half would read exactly like one that
# never had it, and every assertion below would pass. The half is asserted.
assert_value "the tools required outside require_tool were proved and carried into the census, not read as an empty set" \
  "$([[ -n "$unguarded_proved" ]] && echo true || echo false)"
assert_value "the readme's prerequisite list was read, not matched as an empty set" \
  "$([[ "$readme_required_n" -ge 8 ]] && echo true || echo false)"
assert_value "every tool a script requires is listed as a prerequisite in the readme" \
  "$([[ -z "$readme_missing" ]] && echo true || echo false)"
if [[ -n "$readme_missing" ]]; then
  echo "  required by a script, absent from the readme:$readme_missing"
fi
assert_value "every prerequisite the readme lists is required by a script" \
  "$([[ -z "$readme_stale" ]] && echo true || echo false)"
if [[ -n "$readme_stale" ]]; then
  echo "  claimed by the readme, required by nothing:$readme_stale"
fi

# The count written in the prose is the list's own length, so the sentence and
# the list cannot disagree. This is the half that was wrong: the sentence said
# three while the scripts required ten.
readme_claimed_count="$(sed -n 's/^The skills shell out to \([a-z]*\) external tools.*/\1/p' README.md | head -1)"
readme_count_expected="$(english_count "$readme_required_n")"
assert_value "the readme's tool count is the number of tools it lists ($readme_required_n)" \
  "$([[ -n "$readme_claimed_count" && "$readme_claimed_count" == "$readme_count_expected" ]] && echo true || echo false)"

# The readme must not tell a reader that a script which now requires jq is
# unaffected by its absence. check-frontmatter.sh has no --json mode at all and
# requires jq; check-quality.sh required nothing at base and requires it now.
assert_value "the readme does not claim text mode is unaffected by a missing jq" \
  "$(grep -q 'Text mode needs no jq and is unaffected' README.md && echo false || echo true)"
jq_requirers="$(grep -lE '^[[:space:]]*[^#]*require_tool[[:space:]]+jq' "$SCRIPTS_DIR"/*.sh | wc -l | tr -d ' ')"
assert_value "the readme names every script that requires jq, and there are $jq_requirers of them" \
  "$([[ "$jq_requirers" -eq 5 ]] && grep -qi 'all five .*scripts require jq' README.md && echo true || echo false)"

# F6 — `usage()` in draft-rewrite.sh is a heredoc, and it runs on the
# argument-parsing paths, before `require_tool cat`. A broken `cat` made `-h`
# exit 2 and the no-argument and unknown-option paths exit 2 as well, where the
# contract says 0 and 1. Nothing was printed on either channel.
usage_cases=0
for ucase in "0|-h|an explicitly requested help" \
             "1||no argument at all" \
             "1|--no-such-option|an unknown option"; do
  uwant="${ucase%%|*}"; urest="${ucase#*|}"
  uarg="${urest%%|*}"; uwhy="${urest#*|}"
  usage_cases=$((usage_cases + 1))
  if [[ -n "$uarg" ]]; then
    run_on_path "$(unguarded_stub_path cat 2)" "$DRAFT_ABS" "$uarg"
  else
    run_on_path "$(unguarded_stub_path cat 2)" "$DRAFT_ABS"
  fi
  assert_value "drafter with a broken cat, $uwhy: exits $uwant, the status its contract states" \
    "$([[ $code -eq "$uwant" ]] && echo true || echo false)"
  assert_value "drafter with a broken cat, $uwhy: still prints the usage it was asked for" \
    "$([[ -n "$output$errout" ]] && echo true || echo false)"
done
echo "  usage-path cases driven: $usage_cases"
assert_value "the usage-path cases were enumerated, not read as empty" \
  "$([[ "$usage_cases" -eq 3 ]] && echo true || echo false)"

# The control: the same three paths on a healthy toolchain behave the same way,
# so the cases above pin the usage paths rather than the broken cat.
for ucase in "0|-h" "1|" "1|--no-such-option"; do
  uwant="${ucase%%|*}"; uarg="${ucase#*|}"
  if [[ -n "$uarg" ]]; then
    run_present "$DRAFT_ABS" "$uarg"
  else
    run_present "$DRAFT_ABS"
  fi
  assert_value "drafter with a real cat, '${uarg:-no argument}': exits $uwant as well" \
    "$([[ $code -eq "$uwant" ]] && echo true || echo false)"
done

# --- check-quality.sh, inside the guard with its four siblings -----------------
#
# It was the one script in the skill that sat outside the guard, and that is why
# nothing here reached it. It sourced nothing, so a verdict-guard.sh that was
# missing or malformed changed nothing about how it behaved while its four
# siblings all refused to answer. It never checked that the directory it was
# handed held a SKILL.md, so it relayed skillscore's exit 1 for "there is no
# skill here" — a status its own contract does not enumerate. And it passed
# skillscore's status straight through, so skillscore exiting 9 made it exit 9,
# and skillscore exiting 0 having printed nothing made it exit 0 having printed
# nothing: a report generator reporting success over no report.
#
# It also emitted no rule literal, so the rule-ID census contributed the empty
# set for this file and passed vacuously over it — the reason a script with no
# guard could sit here through two releases with every suite green.

QUALITY_GUARD_MODES="missing malformed empty half"
for qmode in $QUALITY_GUARD_MODES; do
  qdir="$(guard_broken_tree "$qmode")"
  run_present "$qdir/check-quality.sh" tests/fixtures/f01/valid-full
  assert_value "quality, guard $qmode: exits 3, like its four siblings" \
    "$([[ $code -eq 3 ]] && echo true || echo false)"
  assert_value "quality, guard $qmode: stdout carries no report" \
    "$([[ -z "$output" ]] && echo true || echo false)"
  assert_value "quality, guard $qmode: says on stderr that it could not load the guard" \
    "$(echo "$errout" | grep -q 'verdict-guard.sh' && echo true || echo false)"
done

# Absence: the one case the old script did handle, kept so the rewrite cannot
# lose it.
run_masked skillscore "$CHECK_QUALITY" tests/fixtures/f01/valid-full
assert_value "quality, skillscore masked: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "quality, skillscore masked: names the missing tool" \
  "$(echo "$errout" | grep -q 'skillscore' && echo true || echo false)"
assert_value "quality, skillscore masked: stdout carries no report" \
  "$([[ -z "$output" ]] && echo true || echo false)"

# A target with no SKILL.md is not a skill this script scored badly.
quality_empty_dir="$mask_root/quality-no-skill"
mkdir -p "$quality_empty_dir"
run_present "$CHECK_QUALITY" "$quality_empty_dir"
assert_value "quality, a directory with no SKILL.md: exits 3, not the scorer's own status" \
  "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "quality, a directory with no SKILL.md: says what it could not find" \
  "$(echo "$errout" | grep -q 'SKILL.md' && echo true || echo false)"

# Usage.
run_present "$CHECK_QUALITY"
assert_value "quality, no argument: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"

# Every way the scorer can fail to answer. Each of these used to become this
# script's own exit status or its own empty report.
quality_liar_cases=0
for qcase in "9|boom|the scorer exits 9" \
             "1||the scorer exits 1 saying nothing" \
             "0||the scorer exits 0 saying nothing" \
             "0|not a report at all|the scorer exits 0 with text that is not JSON" \
             "0|7|the scorer exits 0 with a document that is not an object" \
             "0|{\"overallScore\": 80}|the scorer exits 0 with a score that is not an object" \
             "0|{\"overallScore\": {}}{\"overallScore\": {}}|the scorer answers with two documents"; do
  qexit="${qcase%%|*}"; qrest="${qcase#*|}"
  qsaid="${qrest%%|*}"; qwhy="${qrest#*|}"
  quality_liar_cases=$((quality_liar_cases + 1))
  qstub="$mask_root/quality-stub-$quality_liar_cases"
  if [[ ! -d "$qstub" ]]; then
    mkdir -p "$qstub"
    printf '%s' "$qsaid" > "$qstub/said"
    printf '#!/usr/bin/env bash\ncat "$(dirname "$0")/said"\nexit %s\n' "$qexit" > "$qstub/skillscore"
    chmod +x "$qstub/skillscore"
  fi
  run_on_path "$qstub:$PATH" "$CHECK_QUALITY" tests/fixtures/f01/valid-full
  assert_value "quality, $qwhy: exits 3, not the scorer's own status" \
    "$([[ $code -eq 3 ]] && echo true || echo false)"
  assert_value "quality, $qwhy: stdout carries no report" \
    "$([[ -z "$output" ]] && echo true || echo false)"
  assert_value "quality, $qwhy: names the scorer on stderr" \
    "$(echo "$errout" | grep -q 'skillscore' && echo true || echo false)"
done

echo "  quality-source failure cases driven: $quality_liar_cases"
assert_value "the quality-source failure cases were enumerated, not read as empty" \
  "$([[ "$quality_liar_cases" -eq 7 ]] && echo true || echo false)"

# The controls. A readable report is still relayed unchanged, and the shape the
# guard proves is the shape audit-report.sh reads out of the same source — so a
# report with no overallScore at all is read rather than refused.
run_present "$CHECK_QUALITY" tests/fixtures/f01/valid-full
assert_value "quality, a real scorer: exits 0" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert_value "quality, a real scorer: stdout is one readable report" \
  "$(printf '%s' "$output" | jq -se 'length == 1 and (.[0].overallScore | type) == "object"' >/dev/null 2>&1 && echo true || echo false)"

qstub_ok="$mask_root/quality-stub-noscore"
mkdir -p "$qstub_ok"
printf '#!/usr/bin/env bash\nprintf %s\n' "'{\"skillName\": \"x\"}\\n'" > "$qstub_ok/skillscore"
chmod +x "$qstub_ok/skillscore"
run_on_path "$qstub_ok:$PATH" "$CHECK_QUALITY" tests/fixtures/f01/valid-full
assert_value "quality, a report carrying no overallScore: read, not refused (exit 0)" \
  "$([[ $code -eq 0 ]] && echo true || echo false)"

# PL003 — the line-count rule. No fixture was ever long enough to raise it.
big_skill="$mask_root/over-the-line-limit"
mkdir -p "$big_skill"
{
  printf -- '---\nname: over-the-line-limit\ndescription: A skill whose SKILL.md is longer than house policy allows.\nlicense: MIT\n---\n\n'
  printf '## When to use\n\n- a list item\n\n```bash\necho hi\n```\n\n'
  i=1
  while [[ $i -le 520 ]]; do printf 'filler line %s\n' "$i"; i=$((i + 1)); done
} > "$big_skill/SKILL.md"

run_present "$CHECK_STRUCT" --json "$big_skill"
assert_value "structure --json, SKILL.md over the line limit: emits PL003" \
  "$(echo "$output" | jq -e '.findings[] | select(.rule == "PL003" and .level == "fail")' >/dev/null 2>&1 && echo true || echo false)"
assert_value "structure --json, SKILL.md over the line limit: exits 2 (policy failure)" \
  "$([[ $code -eq 2 ]] && echo true || echo false)"
assert_value "structure --json, SKILL.md over the line limit: the message names the count and the limit" \
  "$(echo "$output" | jq -e '.findings[] | select(.rule == "PL003") | select(.message | test("500"))' >/dev/null 2>&1 && echo true || echo false)"

# --- The documented exit contract is the scripts' own ---------------------------
#
# SKILL.md asserted `0=pass, 1=spec/path failure, 2=policy failure, 3=execution
# error` over five scripts that do not share one contract. audit-report.sh's own
# header says `{0, 3}` and only that one is implemented; check-quality.sh said a
# third thing and implemented none of it; check-paths.sh has no 2. The doc
# restated rather than derived, which is why it drifted, and a caller following
# it wrote `audit-report.sh "$skill" && echo PASS` and got PASS for a skill that
# failed every check.
#
# So the doc carries a row per script, copied from that script's own header, and
# the two are compared here — the same shape as the rule-ID census, for the same
# reason. Both directions: every script has a row, and every row names a script.

# exit_line_of and stated_exit_codes are defined above, beside the adverse
# sweep that compares observed statuses against them. One definition serves both
# halves of the derivation: what the doc says a script exits with, and what the
# script can be made to exit with.

# doc_exit_row_of <basename> — what SKILL.md says that script exits with.
doc_exit_row_of() {
  sed -n "s/^|[[:space:]]*\`$1\`[[:space:]]*|[[:space:]]*\(.*[^[:space:]]\)[[:space:]]*|[[:space:]]*\$/\1/p" \
    skills/skill-audit/SKILL.md | head -1
}

exit_documented=0
exit_undocumented=""
for escript in "$SCRIPTS_DIR"/*.sh; do
  ename="$(basename "$escript")"
  eline="$(exit_line_of "$escript")"
  # verdict-guard.sh is sourced, never run, so it states no exit contract and
  # needs no row. A runnable script with no `# Exit codes:` header is the thing
  # this loop is watching for, and it is caught by the executable check below.
  if [[ -z "$eline" ]]; then
    if [[ -x "$escript" ]]; then
      exit_undocumented="$exit_undocumented $ename"
    fi
    continue
  fi
  exit_documented=$((exit_documented + 1))
  erow="$(doc_exit_row_of "$ename")"
  assert_value "SKILL.md's exit table says for $ename exactly what $ename says" \
    "$([[ -n "$erow" && "$erow" == "$eline" ]] && echo true || echo false)"
  if [[ "$erow" != "$eline" ]]; then
    echo "  $ename header: $eline"
    echo "  $ename in doc : ${erow:-<no row>}"
  fi
done

echo "  scripts whose exit contract was compared: $exit_documented"
assert_value "the exit contracts were enumerated, not read as an empty set" \
  "$([[ "$exit_documented" -ge 5 ]] && echo true || echo false)"
assert_value "every runnable script states its own exit contract in its header" \
  "$([[ -z "$exit_undocumented" ]] && echo true || echo false)"
if [[ -n "$exit_undocumented" ]]; then
  echo "  runnable with no exit contract stated:$exit_undocumented"
fi

# The other direction. A row for a script that no longer exists is a claim about
# nothing, and it would sit there passing every case above.
exit_rows_stale=""
exit_rows_seen=0
while IFS= read -r erow_name; do
  [[ -z "$erow_name" ]] && continue
  exit_rows_seen=$((exit_rows_seen + 1))
  [[ -f "$SCRIPTS_DIR/$erow_name" ]] || exit_rows_stale="$exit_rows_stale $erow_name"
done < <(sed -n 's/^|[[:space:]]*`\([a-z-]*\.sh\)`[[:space:]]*|.*|[[:space:]]*$/\1/p' skills/skill-audit/SKILL.md)

echo "  exit-table rows read from SKILL.md: $exit_rows_seen"
assert_value "SKILL.md's exit table was read, not matched as an empty set" \
  "$([[ "$exit_rows_seen" -eq "$exit_documented" ]] && echo true || echo false)"
assert_value "every row in SKILL.md's exit table names a script that exists" \
  "$([[ -z "$exit_rows_stale" ]] && echo true || echo false)"
if [[ -n "$exit_rows_stale" ]]; then
  echo "  rows naming no script:$exit_rows_stale"
fi

# The sentence beside the table counts the table's own rows, so the two cannot
# disagree. It said four house-policy checks carry their verdict in the exit
# status; there are three, and the fourth and fifth rows are the two generators
# the sentence above it names. An off-by-one prose count in the section whose
# whole subject is a doc that drifted from the scripts — which is what a count
# written as a word rather than derived does, every time.
#
# The criterion is the table's: a script that can exit with something other than
# 0 or 3 is saying something about the skill in its exit status. 0 and 3 alone
# say only "a document was produced" and "it was not".
exit_verdict_carriers=0
exit_generators=0
for vscript in "$SCRIPTS_DIR"/*.sh; do
  [[ -n "$(exit_line_of "$vscript")" ]] || continue
  vcarries=false
  for vcode in $(stated_exit_codes "$vscript"); do
    case "$vcode" in
      0|3) ;;
      *) vcarries=true ;;
    esac
  done
  if $vcarries; then
    exit_verdict_carriers=$((exit_verdict_carriers + 1))
  else
    exit_generators=$((exit_generators + 1))
  fi
done
echo "  scripts carrying a verdict in their exit status: $exit_verdict_carriers"
echo "  scripts that are generators: $exit_generators"
assert_value "the scripts were sorted into verdict-carriers and generators, not read as an empty set" \
  "$([[ $((exit_verdict_carriers + exit_generators)) -eq "$exit_documented" && "$exit_verdict_carriers" -gt 0 ]] && echo true || echo false)"

doc_carrier_word="$(sed -n 's/^The \([a-z][a-z]*\) house-policy checks do carry their verdict in the exit status.*/\1/p' \
  skills/skill-audit/SKILL.md | head -1)"
assert_value "SKILL.md's count of the checks that carry a verdict in the exit status is the table's own ($exit_verdict_carriers)" \
  "$([[ -n "$doc_carrier_word" && "$doc_carrier_word" == "$(english_count "$exit_verdict_carriers")" ]] && echo true || echo false)"
if [[ "$doc_carrier_word" != "$(english_count "$exit_verdict_carriers")" ]]; then
  echo "  SKILL.md says: ${doc_carrier_word:-<no such sentence>}"
  echo "  the table says: $(english_count "$exit_verdict_carriers")"
fi

# And the other half of the same sentence pair, so neither number can be right
# only because the check reads one of them.
doc_generators_named=0
for gscript in "$SCRIPTS_DIR"/*.sh; do
  [[ -n "$(exit_line_of "$gscript")" ]] || continue
  gcarries=false
  for gcode in $(stated_exit_codes "$gscript"); do
    case "$gcode" in 0|3) ;; *) gcarries=true ;; esac
  done
  $gcarries && continue
  grep -q "report a verdict in their output, not in their exit status" skills/skill-audit/SKILL.md \
    && sed -n 's/^\*\*\(.*\)report a verdict in their output.*/\1/p' skills/skill-audit/SKILL.md \
       | grep -qF -- "$(basename "$gscript")" \
    && doc_generators_named=$((doc_generators_named + 1))
done
assert_value "SKILL.md names every generator as one, and there are $exit_generators of them" \
  "$([[ "$doc_generators_named" -eq "$exit_generators" ]] && echo true || echo false)"

# And the behaviour the table now tells the truth about. audit-report.sh exits 0
# over a failing skill — deliberately, because 0 means a report was generated —
# so the verdict has to be read out of the report, and the doc has to say so.
run_present skills/skill-audit/scripts/audit-report.sh tests/fixtures/f01/frontmatter-bleed
assert_value "audit-report over a failing skill: still exits 0, the contract its header states" \
  "$([[ $code -eq 0 ]] && echo true || echo false)"
assert_value "audit-report over a failing skill: the verdict is in summary.passed, and it is false" \
  "$([[ "$(echo "$output" | jq -r '.summary.passed')" == "false" ]] && echo true || echo false)"
assert_value "SKILL.md tells a caller to read summary.passed rather than the exit status" \
  "$(grep -q 'summary.passed' skills/skill-audit/SKILL.md && echo true || echo false)"
assert_value "SKILL.md warns that the '&& echo PASS' shape prints PASS for a failing skill" \
  "$(grep -q 'echo PASS' skills/skill-audit/SKILL.md && echo true || echo false)"

# --- Where the frontmatter ends: one question, one answer ----------------------
#
# Four scripts used to answer it privately and they disagreed. Two exited at the
# second `---` and were right. check-paths.sh ran a toggle, so a third `---` put
# it back into frontmatter and one markdown horizontal rule ended every path
# check after it. check-structure.sh never asked, so its whole house-policy
# verdict was satisfiable out of frontmatter.
#
# These cases are pinned to the invariants, not to the primitive that now holds
# them: a rule about the body is computed over the body, and frontmatter ends
# once. A later repair is free to move the reading anywhere as long as both
# still hold.

BLEED=tests/fixtures/f01/frontmatter-bleed
RULE_REFS=tests/fixtures/f01/rule-before-refs
NO_RULE_REFS=tests/fixtures/f01/no-rule-before-refs

# The bleed fixture's preconditions, asserted rather than assumed. Each policy
# rule below has to be satisfiable from this file and unsatisfied by its body,
# or the case that follows would pass over a fixture that no longer reproduces
# anything — which is how a regression test quietly stops being one. Six
# heading-shaped lines, two fence-shaped lines and two list-shaped lines are in
# the file; none of them is in the body.
assert_value "bleed fixture: its frontmatter still holds six heading-shaped lines" \
  "$([[ "$(grep -cE '^#{2,6}[[:space:]]+' "$BLEED/SKILL.md" | tr -d '[:space:]')" -eq 6 ]] && echo true || echo false)"
assert_value "bleed fixture: its frontmatter still holds a code fence" \
  "$([[ "$(grep -cE '^[[:space:]]*```' "$BLEED/SKILL.md" | tr -d '[:space:]')" -ge 1 ]] && echo true || echo false)"
assert_value "bleed fixture: its delimiters still read as list items" \
  "$([[ "$(grep -cE '^[[:space:]]*[-*]' "$BLEED/SKILL.md" | tr -d '[:space:]')" -ge 1 ]] && echo true || echo false)"
assert_value "bleed fixture: skill-validator passes it, so the toolchain is healthy" \
  "$("$SV" validate structure -o json "$BLEED" >/dev/null 2>&1 && echo true || echo false)"

# G1-01. Every rule about the body, reported missing from the body, while the
# file that contains all of them sits right there.
run_present "$CHECK_STRUCT" --json "$BLEED"
assert_value "structure --json, body rules satisfied only in frontmatter: reports all six headings missing" \
  "$([[ "$(echo "$output" | jq '[.findings[] | select(.rule == "PL002")] | length')" -eq 6 ]] && echo true || echo false)"
assert_value "structure --json, body rules satisfied only in frontmatter: reports PL004, the body has no fence" \
  "$(echo "$output" | jq -e '.findings[] | select(.rule == "PL004")' >/dev/null 2>&1 && echo true || echo false)"
assert_value "structure --json, body rules satisfied only in frontmatter: reports PL005, the body has no list" \
  "$(echo "$output" | jq -e '.findings[] | select(.rule == "PL005")' >/dev/null 2>&1 && echo true || echo false)"
assert_value "structure --json, body rules satisfied only in frontmatter: never reports passed true" \
  "$([[ "$(echo "$output" | jq -r '.passed')" == "false" ]] && echo true || echo false)"
assert_value "structure --json, body rules satisfied only in frontmatter: exits 2 (policy failure)" \
  "$([[ $code -eq 2 ]] && echo true || echo false)"

# The headline command is the surface the defect was reported on, so it is the
# surface the repair is confirmed on. A green unit case over the child proves a
# branch runs; it does not prove audit-report.sh stopped issuing a clean bill of
# health.
run_present skills/skill-audit/scripts/audit-report.sh "$BLEED"
assert_value "audit-report, body rules satisfied only in frontmatter: summary.passed is false" \
  "$([[ "$(echo "$output" | jq -r '.summary.passed')" == "false" ]] && echo true || echo false)"
assert_value "audit-report, body rules satisfied only in frontmatter: counts every body rule it broke" \
  "$([[ "$(echo "$output" | jq -r '.summary.policy_failures')" -ge 8 ]] && echo true || echo false)"
assert_value "audit-report, body rules satisfied only in frontmatter: still exits 0, the report generated" \
  "$([[ $code -eq 0 ]] && echo true || echo false)"

# G1-02. The pair differs by one `---`. A frontmatter reading that re-enters
# reads everything after that line as frontmatter and checks none of it.
assert_value "rule fixture pair: they still differ by exactly one delimiter line" \
  "$([[ "$(grep -c '^---$' "$RULE_REFS/SKILL.md" | tr -d '[:space:]')" -eq 3 && "$(grep -c '^---$' "$NO_RULE_REFS/SKILL.md" | tr -d '[:space:]')" -eq 2 ]] && echo true || echo false)"

run_present "$CHECK_PATHS" --json "$RULE_REFS"
rule_findings="$(echo "$output" | jq -S -c '.findings')"
rule_code=$code
run_present "$CHECK_PATHS" --json "$NO_RULE_REFS"
no_rule_findings="$(echo "$output" | jq -S -c '.findings')"
no_rule_code=$code

# Parity on its own is not the invariant: two clean passes are also identical.
# The pair has to agree *and* both have to be the failure the references are.
assert_value "paths --json, a horizontal rule before the broken references: finds them anyway" \
  "$([[ "$rule_code" -eq 1 ]] && echo true || echo false)"
assert_value "paths --json, a horizontal rule before the broken references: same findings as without it" \
  "$([[ "$rule_findings" == "$no_rule_findings" ]] && echo true || echo false)"
assert_value "paths --json, a horizontal rule before the broken references: same exit as without it" \
  "$([[ "$rule_code" -eq "$no_rule_code" ]] && echo true || echo false)"
if [[ "$rule_findings" != "$no_rule_findings" ]]; then
  echo "  with a rule   : $rule_findings"
  echo "  without a rule: $no_rule_findings"
fi

# G1-03. PL003 is a limit on the body. Both skills here have a body under the
# limit or over it by one line, and a *file* over it either way — so a count
# taken over the whole file raises PL003 on the conforming one.
for body_lines in 500 501; do
  boundary_skill="$mask_root/pl003-body-$body_lines"
  mkdir -p "$boundary_skill"
  {
    printf -- '---\nname: pl003-body-%s\ndescription: A skill whose body is %s lines long, in a file that is longer, so PL003 can be seen to count the body.\nlicense: MIT\n---\n' \
      "$body_lines" "$body_lines"
    i=1
    while [[ $i -le $body_lines ]]; do printf 'body line %s\n' "$i"; i=$((i + 1)); done
  } > "$boundary_skill/SKILL.md"
  run_present "$CHECK_STRUCT" --json "$boundary_skill"
  raised="$(echo "$output" | jq -e '.findings[] | select(.rule == "PL003")' >/dev/null 2>&1 && echo true || echo false)"
  expected=false
  [[ $body_lines -gt 500 ]] && expected=true
  assert_value "structure --json, a $body_lines-line body in a longer file: PL003 raised is $expected" \
    "$([[ "$raised" == "$expected" ]] && echo true || echo false)"
  assert_value "structure --json, a $body_lines-line body: the file itself is over the limit, so the count is the body's" \
    "$([[ "$(wc -l < "$boundary_skill/SKILL.md" | tr -d '[:space:]')" -gt 500 ]] && echo true || echo false)"
done

# And the class, not just the two instances. A script that scans for the
# frontmatter delimiter itself holds a private answer to a question that has
# one, and the next reader is back to choosing between copies.
private_scanners="$(grep -lE '\^---\$|== "---"' skills/skill-audit/scripts/*.sh skills/skill-rewrite/scripts/*.sh \
  | grep -v 'verdict-guard\.sh' || true)"
assert_value "no script outside the shared primitive reads the frontmatter delimiter itself" \
  "$([[ -z "$private_scanners" ]] && echo true || echo false)"
if [[ -n "$private_scanners" ]]; then
  echo "  private frontmatter scans: $(printf '%s' "$private_scanners" | tr '\n' ' ')"
fi

# The control. A check that can only ever say "nothing found" is the shape this
# whole repository spent a cluster removing, so it is given something to find.
scanner_control="$mask_root/private-scanner.sh"
printf '%s\n' '#!/usr/bin/env bash' 'awk '\''$0 == "---" { next }'\'' "$1"' > "$scanner_control"
assert_value "the private-frontmatter-scan check fires on a script that has one" \
  "$([[ -n "$(grep -lE '\^---\$|== "---"' "$scanner_control" || true)" ]] && echo true || echo false)"

# --- What a script's header says, against what the script was seen to do -------
#
# The derivation above compares SKILL.md's table against each script's own
# `# Exit codes:` header, and the adverse sweep compares every status it drove
# against that header. Neither asks the question the other side of: whether
# every status the header states is one the script can actually be made to
# emit. Both sides moving together is uncaught — add a status a script can
# never emit, or drop one it emits on every failure, and the suite stays fully
# green. G4-01's original symptom was a stated code untrue of the script, so
# the mechanism installed to close it does not detect its own shape.
#
# The witness is every run this suite made through the shared harness, which is
# every run of a script it made at all. Both directions: a stated status nothing
# produced is a claim about nothing, and a produced status nothing stated is a
# script outside its own contract.
exit_witness_unwitnessed=""
exit_witness_unstated=""
exit_witness_scripts=0
for wscript in "$SCRIPTS_DIR"/*.sh skills/skill-rewrite/scripts/*.sh; do
  wname="$(basename "$wscript")"
  [[ -n "$(exit_line_of "$wscript")" ]] || continue
  exit_witness_scripts=$((exit_witness_scripts + 1))
  wseen="$(printf '%s\n' $exit_witness | sed -n "s/^$wname://p" | sort -u | tr '\n' ' ')"
  echo "  $wname stated [$(stated_exit_codes "$wscript" | tr '\n' ' ')] seen [$wseen]"
  for wcode in $(stated_exit_codes "$wscript"); do
    case " $wseen " in
      *" $wcode "*) ;;
      *) exit_witness_unwitnessed="$exit_witness_unwitnessed $wname:$wcode" ;;
    esac
  done
  for wcode in $wseen; do
    states_exit "$wscript" "$wcode" || exit_witness_unstated="$exit_witness_unstated $wname:$wcode"
  done
done
assert_value "the exit-status witness was collected, not read as an empty set" \
  "$([[ "$exit_witness_scripts" -ge 6 && -n "$exit_witness" ]] && echo true || echo false)"
assert_value "every status a script's header states is one this suite made it emit" \
  "$([[ -z "$exit_witness_unwitnessed" ]] && echo true || echo false)"
if [[ -n "$exit_witness_unwitnessed" ]]; then
  echo "  stated but never emitted here:$exit_witness_unwitnessed"
fi
assert_value "every status this suite made a script emit is one its header states" \
  "$([[ -z "$exit_witness_unstated" ]] && echo true || echo false)"
if [[ -n "$exit_witness_unstated" ]]; then
  echo "  emitted but not stated:$exit_witness_unstated"
fi

harness_summary
