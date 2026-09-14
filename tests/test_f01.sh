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
# A copy of the scripts directory whose shared verdict-guard.sh is unusable:
# absent, syntactically broken, or present but defining none of the guards.
# Echoes the path to the copied directory. Cached per mode.
guard_broken_tree() {
  local mode="$1"
  local dir="$mask_root/guard-$mode"
  if [[ ! -d "$dir" ]]; then
    cp -R skills/skill-audit/scripts "$dir"
    case "$mode" in
      missing)   rm -f "$dir/verdict-guard.sh" ;;
      malformed) printf 'if then fi (\n' > "$dir/verdict-guard.sh" ;;
      empty)     printf '# a guard that defines no guards\n' > "$dir/verdict-guard.sh" ;;
    esac
  fi
  echo "$dir"
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
  assert "frontmatter, validator exits $vcode: exits $vwant" \
    "$([[ $code -eq $vwant ]] && echo true || echo false)"
  if [[ "$vverdict" == "ok" ]]; then
    assert "frontmatter, validator exits $vcode: prints frontmatter OK" \
      "$(echo "$output" | grep -q 'frontmatter OK' && echo true || echo false)"
  else
    assert "frontmatter, validator exits $vcode: never prints frontmatter OK" \
      "$(echo "$output" | grep -q 'frontmatter OK' && echo false || echo true)"
  fi
done

# The unenumerated arm still names what it could not interpret, and the
# enumerated exit-3 arm names the tool that failed to run.
run_on_path "$(stub_tool_path skill-validator 42)" "$CHECK_FM" tests/fixtures/f01/valid-full
assert "frontmatter, validator exits 42: names the tool and the status" \
  "$(echo "$errout" | grep -q 'skill-validator' && echo "$errout" | grep -q '42' && echo true || echo false)"

run_on_path "$(stub_tool_path skill-validator 3)" "$CHECK_FM" tests/fixtures/f01/valid-full
assert "frontmatter, validator exits 3: names the tool that failed to run" \
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
  assert "structure --json, child gives $why: exits ${want_exit[$i]}" \
    "$([[ $code -eq ${want_exit[$i]} ]] && echo true || echo false)"
  assert "structure --json, child gives $why: stdout is a payload" \
    "$(echo "$output" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
  assert "structure --json, child gives $why: payload passed is ${want_passed[$i]}" \
    "$([[ "$(echo "$output" | jq -r '.passed' 2>/dev/null)" == "${want_passed[$i]}" ]] && echo true || echo false)"
  if [[ -n "${want_rule[$i]}" ]]; then
    assert "structure --json, child gives $why: payload carries ${want_rule[$i]}" \
      "$(echo "$output" | jq -e --arg r "${want_rule[$i]}" '.findings[] | select(.rule == $r)' >/dev/null 2>&1 && echo true || echo false)"
  fi
  # The invariant every arm above exists to hold, pinned over the whole boundary
  # so it survives any rework of the arms that currently enforce it: a payload
  # may never claim it passed while carrying a finding that says it failed.
  assert "structure --json, child gives $why: never claims passed beside a fail finding" \
    "$(echo "$output" | jq -e '.passed == true and ([.findings[] | select(.level == "fail")] | length > 0)' >/dev/null 2>&1 && echo false || echo true)"
done

# The DEP002 message names the status it could not interpret, so a consumer can
# tell which child result it is looking at.
run_present "$(stub_paths_tree u42 42 "")" --json tests/fixtures/f01/valid-full
assert "structure --json, child exits 42: DEP002 names the status" \
  "$(echo "$output" | jq -e '.findings[] | select(.rule == "DEP002") | select(.message | test("42"))' >/dev/null 2>&1 && echo true || echo false)"

# Text mode reads no payload, so only the exit-status enumeration applies — and
# it applies identically: a child that reached no verdict leaves us with none.
run_present "$(stub_paths_tree pass0 0 "$PAYLOAD_PASS")" tests/fixtures/f01/valid-full
assert "structure text, child gives a conforming pass: exits 0" "$([[ $code -eq 0 ]] && echo true || echo false)"

run_present "$(stub_paths_tree dep3 3 "$PAYLOAD_DEP")" tests/fixtures/f01/valid-full
assert "structure text, child exits 3 (no verdict): exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"

run_present "$(stub_paths_tree u42 42 "")" tests/fixtures/f01/valid-full
assert "structure text, child exits 42: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"

# DEP001's own registration is pinned by the jq-masked cases above, which assert
# the rule ID in both check-structure.sh's and check-paths.sh's payloads.

# --- The shared guard is itself a dependency, and require_tool cannot cover it ---
#
# Every other dependency is announced by the guard. The guard's own absence is
# announced by nothing: `source` fails, `set -e` aborts, and the script exits 1
# or 2 — statuses this contract reserves for verdicts the script actually
# computed. A missing guard must read as "no verdict reached", like every other
# unmet precondition, in the exit status and on the payload channel alike.

for gmode in missing malformed empty; do
  gdir="$(guard_broken_tree "$gmode")"
  for gscript in check-structure.sh check-paths.sh check-frontmatter.sh audit-report.sh; do
    run_present "$gdir/$gscript" tests/fixtures/f01/valid-full
    assert "$gscript, guard $gmode: exits 3, not a status meaning a verdict" \
      "$([[ $code -eq 3 ]] && echo true || echo false)"
    assert "$gscript, guard $gmode: stdout carries no verdict" \
      "$([[ -z "$output" ]] && echo true || echo false)"
    assert "$gscript, guard $gmode: says on stderr that it could not load the guard" \
      "$(echo "$errout" | grep -q 'verdict-guard.sh' && echo true || echo false)"
  done
  run_present "$gdir/check-structure.sh" --json tests/fixtures/f01/valid-full
  assert "check-structure.sh --json, guard $gmode: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
  assert "check-structure.sh --json, guard $gmode: never reports passed true" \
    "$(echo "$output" | grep -qE '"passed":[[:space:]]*true' && echo false || echo true)"
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
assert "verdict-guard.sh sits beside the scripts that source it" "$([[ -f "$GUARD" ]] && echo true || echo false)"

guard_nasty=$'he said "boom" \\ then a tab\there, a newline\na return\ra formfeed\fa backspace\bthen \x01 and \x1f'
guard_out="$(bash -c 'source "$1"; cannot_compute DEP002 "$2" true' _ "$GUARD" "$guard_nasty" 2>/dev/null || true)"
assert "guard: payload is valid JSON when the message holds every character the encoder escapes" "$(echo "$guard_out" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
assert "guard: error field round-trips the message exactly" "$([[ "$(echo "$guard_out" | jq -r '.error' 2>/dev/null)" == "$guard_nasty" ]] && echo true || echo false)"
assert "guard: finding message round-trips the message exactly" "$([[ "$(echo "$guard_out" | jq -r '.findings[0].message' 2>/dev/null)" == "$guard_nasty" ]] && echo true || echo false)"
assert "guard: finding carries the rule it was given" "$([[ "$(echo "$guard_out" | jq -r '.findings[0].rule' 2>/dev/null)" == "DEP002" ]] && echo true || echo false)"
assert "guard: payload never reports passed true" "$(echo "$guard_out" | grep -qE '"passed":[[:space:]]*true' && echo false || echo true)"

# The detail — a failed tool's own output — is the one thing the guard relays
# verbatim, and it is exactly the thing that would corrupt the payload if it
# went to stdout. It belongs on stderr, beside the reason, whatever it holds.

run_present bash -c 'source "$1"; cannot_compute DEP002 "the reason" true "DETAIL-MARKER: {not json"' _ "$GUARD"
assert "guard: detail reaches stderr" "$(echo "$errout" | grep -q 'DETAIL-MARKER' && echo true || echo false)"
assert "guard: detail never reaches the payload channel" "$(echo "$output" | grep -q 'DETAIL-MARKER' && echo false || echo true)"
assert "guard: stdout is still exactly the payload when a detail is relayed" "$(echo "$output" | jq -e '.findings[0].rule == "DEP002"' >/dev/null 2>&1 && echo true || echo false)"
assert "guard: the reason reaches stderr too" "$(echo "$errout" | grep -q 'the reason' && echo true || echo false)"

# The guard needs no jq to build its payload: it is what runs when jq is the
# tool that went missing.
run_masked jq bash -c 'source "$1"; cannot_compute DEP001 "required tool not found: jq" true' _ "$GUARD"
assert "guard: emits its payload with jq itself absent" "$(echo "$output" | jq -e '.findings[0].rule == "DEP001"' >/dev/null 2>&1 && echo true || echo false)"

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
assert "paths --json, bad flag: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert "paths --json, bad flag: stdout is empty" "$([[ -z "$output" ]] && echo true || echo false)"
assert "paths --json, bad flag: diagnostic is on stderr" "$(echo "$errout" | grep -q 'Usage' && echo true || echo false)"

run_present "$CHECK_STRUCT" --json --bogus
assert "structure --json, bad flag: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert "structure --json, bad flag: stdout is empty" "$([[ -z "$output" ]] && echo true || echo false)"
assert "structure --json, bad flag: diagnostic is on stderr" "$(echo "$errout" | grep -q 'Usage' && echo true || echo false)"

run_present "$CHECK_PATHS" --json "$mask_root/no-such-skill"
assert "paths --json, no SKILL.md: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert "paths --json, no SKILL.md: stdout is empty" "$([[ -z "$output" ]] && echo true || echo false)"
assert "paths --json, no SKILL.md: diagnostic is on stderr" "$(echo "$errout" | grep -q 'SKILL.md not found' && echo true || echo false)"

run_present "$CHECK_STRUCT" --json "$mask_root/no-such-skill"
assert "structure --json, no SKILL.md: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert "structure --json, no SKILL.md: stdout is empty" "$([[ -z "$output" ]] && echo true || echo false)"
assert "structure --json, no SKILL.md: diagnostic is on stderr" "$(echo "$errout" | grep -q 'SKILL.md not found' && echo true || echo false)"

run_present "$CHECK_FM" "$mask_root/no-such-skill"
assert "frontmatter, no SKILL.md: exits 3" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert "frontmatter, no SKILL.md: diagnostic is on stderr" "$(echo "$errout" | grep -q 'SKILL.md not found' && echo true || echo false)"

echo
echo "$pass passed, $fail failed"
if [[ "$fail" -gt 0 ]]; then
  exit 1
fi
