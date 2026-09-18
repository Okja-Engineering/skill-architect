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

for spec in "0:0:ok" "1:1:no" "2:0:ok" "3:3:no" "4:3:no" "42:3:no"; do
  vcode="${spec%%:*}"
  vrest="${spec#*:}"
  vwant="${vrest%%:*}"
  vverdict="${vrest#*:}"
  run_on_path "$(stub_tool_path skill-validator "$vcode")" "$CHECK_FM" tests/fixtures/f01/valid-full
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
run_on_path "$(stub_tool_path skill-validator 42)" "$CHECK_FM" tests/fixtures/f01/valid-full
assert_value "frontmatter, validator exits 42: names the tool and the status" \
  "$(echo "$errout" | grep -q 'skill-validator' && echo "$errout" | grep -q '42' && echo true || echo false)"

run_on_path "$(stub_tool_path skill-validator 3)" "$CHECK_FM" tests/fixtures/f01/valid-full
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
# has to cover the whole set rather than whichever names a caller listed. Adding
# a guard to the shared file without adding it to that set fails here.

for gdrop in json_string cannot_compute require_tool json_document_conforms payload_is_conforming verdict_guard_ready; do
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
# DEP001 for stating a tool precondition. Each form tolerates no match, so a
# script that emits nothing yields nothing instead of killing the suite.
rules_emitted_directly_by() {
  {
    grep -hoE 'findings\+=\("[a-z]+\|[A-Z][A-Z0-9]*\|' "$@" | sed -E 's/.*\|([A-Z][A-Z0-9]*)\|/\1/' || true
    grep -hoE 'cannot_compute[[:space:]]+"?[A-Z][A-Z0-9]*' "$@" | tr -d '"' | awk '{print $NF}' || true
    grep -hoE '"rule": "[A-Z][A-Z0-9]*"' "$@" | sed -E 's/.*"([A-Z][A-Z0-9]*)"/\1/' || true
    grep -hoE '\[[A-Z][A-Z0-9]*\]' "$@" | tr -d '[]' || true
    if grep -qE '^[[:space:]]*require_tool[[:space:]]' "$@"; then
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
    if grep -qE "$(basename "$child")\"? --json" "$file"; then
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
registry_line="$(grep -m1 '^Exit codes:' skills/skill-audit/SKILL.md)"
# Any rule-shaped token, not a fixed list of the prefixes in use. A whitelist
# here would make the registry side unable to grow either: a new rule announced
# under a new prefix would be invisible to the very line that claims to name the
# whole set.
registered_rules="$(printf '%s\n' "$registry_line" \
  | grep -oE '\b[A-Z][A-Z0-9]{2,}\b' | sort -u)"

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

harness_summary
