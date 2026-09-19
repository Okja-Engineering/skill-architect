#!/usr/bin/env bash
# F02 tests: unified audit report (audit-report.sh)
# Verifies the composer merges skill-validator, skillscore, and house-policy
# findings into a single machine-readable JSON document.
set -euo pipefail
. "$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)/lib/harness.sh"
harness_init

# This suite's assertions compute their verdict inside a command substitution
# and hand the harness the result, so they call `assert_value` rather than
# `assert`. Both live in tests/lib/harness.sh and both count into the same
# summary; what they do not have is a copy here to drift from it. See
# tests/lib/audit-suites.sh for what holds the value shape non-vacuous.

# Helper: run audit-report.sh and capture JSON output.
run_report() {
  local dir="$1"
  code=0
  output=$(skills/skill-audit/scripts/audit-report.sh "$dir" 2>&1) || code=$?
}

# Helper: assert a jq path equals an expected string value.
assert_jq_eq() {
  local label="$1"
  local path="$2"
  local expected="$3"
  local actual
  actual=$(echo "$output" | jq -r "$path")
  assert_value "$label" "$([[ "$actual" == "$expected" ]] && echo true || echo false)"
}

# Helper: assert a jq path is greater than a number.
assert_jq_gt() {
  local label="$1"
  local path="$2"
  local threshold="$3"
  local actual
  actual=$(echo "$output" | jq -r "$path")
  assert_value "$label" "$([[ "$actual" -gt "$threshold" ]] && echo true || echo false)"
}

# Helper: assert a jq path exists (is not null).
assert_jq_exists() {
  local label="$1"
  local path="$2"
  assert_value "$label" "$(echo "$output" | jq -e "$path" >/dev/null 2>&1 && echo true || echo false)"
}

# Helper: assert a finding with a given rule exists in policy.findings.
assert_has_finding() {
  local label="$1"
  local rule="$2"
  assert_value "$label" "$(echo "$output" | jq -e --arg r "$rule" '.policy.findings[] | select(.rule == $r)' >/dev/null 2>&1 && echo true || echo false)"
}

# Helper: assert a finding with a given rule does NOT exist.
assert_no_finding() {
  local label="$1"
  local rule="$2"
  assert_value "$label" "$(echo "$output" | jq -e --arg r "$rule" '.policy.findings[] | select(.rule == $r)' >/dev/null 2>&1 && echo false || echo true)"
}

# --- Top-level JSON shape ---

run_report tests/fixtures/f01/valid-full
assert_value "valid-full: report exits 0" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert_value "valid-full: output is valid JSON" "$(echo "$output" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
assert_jq_exists "valid-full: has skill field" '.skill'
assert_jq_exists "valid-full: has timestamp field" '.timestamp'
assert_jq_exists "valid-full: has summary object" '.summary'
assert_jq_exists "valid-full: has spec object" '.spec'
assert_jq_exists "valid-full: has quality object" '.quality'
assert_jq_exists "valid-full: has policy object" '.policy'
assert_jq_exists "valid-full: has policy.findings array" '.policy.findings'

# --- Summary fields ---

assert_jq_eq "valid-full: summary.passed is true" '.summary.passed' 'true'
assert_jq_eq "valid-full: summary.spec_passed is true" '.summary.spec_passed' 'true'
assert_jq_eq "valid-full: summary.spec_errors is 0" '.summary.spec_errors' '0'
assert_jq_eq "valid-full: summary.quality_score is 80" '.summary.quality_score' '80'
assert_jq_eq "valid-full: summary.quality_grade is B-" '.summary.quality_grade' 'B-'
assert_jq_eq "valid-full: summary.policy_failures is 0" '.summary.policy_failures' '0'
assert_jq_eq "valid-full: summary.path_failures is 0" '.summary.path_failures' '0'
assert_jq_eq "valid-full: summary.total_findings is 0" '.summary.total_findings' '0'

# --- Spec source (skill-validator) is nested ---

assert_jq_eq "valid-full: spec.passed is true" '.spec.passed' 'true'
assert_value "valid-full: spec has results array" "$(echo "$output" | jq -e '.spec.results | length > 0' >/dev/null 2>&1 && echo true || echo false)"
assert_jq_exists "valid-full: spec has token_counts" '.spec.token_counts'
assert_jq_exists "valid-full: spec has content_analysis" '.spec.content_analysis'
assert_jq_exists "valid-full: spec has contamination_analysis" '.spec.contamination_analysis'

# --- Quality source (skillscore) is nested ---

assert_jq_exists "valid-full: quality has overallScore" '.quality.overallScore'
assert_value "valid-full: quality has 7 categories" "$(echo "$output" | jq -e '.quality.categories | length == 7' >/dev/null 2>&1 && echo true || echo false)"

# --- PL001: no-license fixture (license is house policy, not spec) ---

run_report tests/fixtures/f01/no-license
assert_jq_eq "no-license: summary.passed is false" '.summary.passed' 'false'
assert_jq_eq "no-license: summary.spec_passed is true" '.summary.spec_passed' 'true'
assert_has_finding "no-license: reports PL001 finding" 'PL001'
assert_jq_gt "no-license: policy_failures > 0" '.summary.policy_failures' 0

# --- PL002-PL005: missing headings, code blocks via bad skill ---

tmp_bad="$harness_scratch/bad"
mkdir -p "$tmp_bad/bad-skill"
cat > "$tmp_bad/bad-skill/SKILL.md" <<'SKILL_EOF'
---
name: bad-skill
description: A bad skill with no sections.
license: MIT
---

# bad-skill

No sections here.
SKILL_EOF

run_report "$tmp_bad/bad-skill"
assert_jq_eq "bad-skill: summary.passed is false" '.summary.passed' 'false'
assert_has_finding "bad-skill: reports PL002 finding" 'PL002'
assert_has_finding "bad-skill: reports PL004 finding" 'PL004'
assert_no_finding "bad-skill: does NOT report PL001 (has license)" 'PL001'

rm -rf "$tmp_bad"

# --- PT001: missing script reference ---

run_report tests/fixtures/f01/missing-script-ref
assert_jq_gt "missing-script-ref: summary.path_failures > 0" '.summary.path_failures' 0
assert_has_finding "missing-script-ref: reports PT001 finding" 'PT001'

# --- PT002: missing markdown reference ---

run_report tests/fixtures/f01/missing-markdown-ref
assert_has_finding "missing-markdown-ref: reports PT002 finding" 'PT002'

# --- Spec failure: malformed YAML ---

run_report tests/fixtures/f01/malformed-yaml
assert_jq_eq "malformed-yaml: summary.spec_passed is false" '.summary.spec_passed' 'false'
assert_jq_eq "malformed-yaml: summary.spec_errors is 1" '.summary.spec_errors' '1'
assert_jq_eq "malformed-yaml: summary.passed is false" '.summary.passed' 'false'
assert_jq_eq "malformed-yaml: spec_error is null (valid JSON with errors)" '.spec_error' 'null'

# --- Spec failure: invalid name format ---

run_report tests/fixtures/f01/invalid-name-format
assert_jq_eq "invalid-name-format: summary.spec_passed is false" '.summary.spec_passed' 'false'

# --- Self-audit: skill-audit can audit itself ---

run_report skills/skill-audit
assert_value "skill-audit self-audit: report exits 0" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert_jq_eq "skill-audit self-audit: summary.passed is true" '.summary.passed' 'true'

# --- check-paths.sh --json backward compat ---

run_paths_text() {
  local dir="$1"
  pcode=0
  poutput=$(skills/skill-audit/scripts/check-paths.sh "$dir" 2>&1) || pcode=$?
}

run_paths_json() {
  local dir="$1"
  pcode=0
  poutput=$(skills/skill-audit/scripts/check-paths.sh --json "$dir" 2>&1) || pcode=$?
}

# Text mode still works (no --json)
run_paths_text tests/fixtures/f01/valid-full
assert_value "check-paths.sh text: valid-full passes (exit 0)" "$([[ $pcode -eq 0 ]] && echo true || echo false)"

run_paths_text tests/fixtures/f01/missing-script-ref
assert_value "check-paths.sh text: missing-script-ref fails (exit 1)" "$([[ $pcode -eq 1 ]] && echo true || echo false)"
assert_value "check-paths.sh text: reports PT001" "$(echo "$poutput" | grep -q 'PT001' && echo true || echo false)"

# JSON mode works
run_paths_json tests/fixtures/f01/valid-full
assert_value "check-paths.sh --json: valid-full passes (exit 0)" "$([[ $pcode -eq 0 ]] && echo true || echo false)"
assert_value "check-paths.sh --json: output is valid JSON" "$(echo "$poutput" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
assert_value "check-paths.sh --json: has findings array" "$(echo "$poutput" | jq -e '.findings' >/dev/null 2>&1 && echo true || echo false)"
assert_value "check-paths.sh --json: passed is true" "$([[ "$(echo "$poutput" | jq -r '.passed')" == "true" ]] && echo true || echo false)"

run_paths_json tests/fixtures/f01/missing-script-ref
assert_value "check-paths.sh --json: missing-script-ref fails (exit 1)" "$([[ $pcode -eq 1 ]] && echo true || echo false)"
assert_value "check-paths.sh --json: passed is false" "$([[ "$(echo "$poutput" | jq -r '.passed')" == "false" ]] && echo true || echo false)"
assert_value "check-paths.sh --json: has PT001 finding" "$(echo "$poutput" | jq -e '.findings[] | select(.rule == "PT001")' >/dev/null 2>&1 && echo true || echo false)"

# --- check-structure.sh --json backward compat ---

run_struct_text() {
  local dir="$1"
  scode=0
  soutput=$(skills/skill-audit/scripts/check-structure.sh "$dir" 2>&1) || scode=$?
}

run_struct_json() {
  local dir="$1"
  scode=0
  soutput=$(skills/skill-audit/scripts/check-structure.sh --json "$dir" 2>&1) || scode=$?
}

# Text mode still works
run_struct_text tests/fixtures/f01/valid-full
assert_value "check-structure.sh text: valid-full passes (exit 0)" "$([[ $scode -eq 0 ]] && echo true || echo false)"

# JSON mode works
run_struct_json tests/fixtures/f01/valid-full
assert_value "check-structure.sh --json: valid-full passes (exit 0)" "$([[ $scode -eq 0 ]] && echo true || echo false)"
assert_value "check-structure.sh --json: output is valid JSON" "$(echo "$soutput" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
assert_value "check-structure.sh --json: passed is true" "$([[ "$(echo "$soutput" | jq -r '.passed')" == "true" ]] && echo true || echo false)"

run_struct_json tests/fixtures/f01/missing-script-ref
assert_value "check-structure.sh --json: missing-script-ref has findings" "$([[ "$(echo "$soutput" | jq -r '.findings | length')" -gt 0 ]] && echo true || echo false)"
assert_value "check-structure.sh --json: missing-script-ref includes PT001 from check-paths" "$(echo "$soutput" | jq -e '.findings[] | select(.rule == "PT001")' >/dev/null 2>&1 && echo true || echo false)"

# --- audit-report.sh states its jq dependency instead of tripping over it ---
#
# Every other script in the skill names a missing tool and exits 3. This one
# composed its whole report through jq and, without it, died mid-run at the
# final merge with the shell's own `jq: command not found` and exit 127 — the
# very shape the rest of the skill exists to prevent. The invariant is the same
# one: a script that cannot reach its result says which tool is missing and
# exits 3, rather than failing in a way a caller has to decode.
#
# Masked PATHs are symlink farms of the real PATH minus one binary. Nothing is
# deleted, moved or uninstalled.

mask_root="$harness_scratch/mask"
mkdir -p "$mask_root"
source tests/lib/masked-path.sh

REPORT=skills/skill-audit/scripts/audit-report.sh

run_masked jq "$REPORT" tests/fixtures/f01/valid-full
assert_value "audit-report, jq masked: exits 3 (execution error)" "$([[ $code -eq 3 ]] && echo true || echo false)"
assert_value "audit-report, jq masked: names the missing tool" "$(echo "$errout" | grep -q 'required tool not found: jq' && echo true || echo false)"
assert_value "audit-report, jq masked: does not die with a bare command-not-found" "$(echo "$errout" | grep -q 'command not found' && echo false || echo true)"
assert_value "audit-report, jq masked: emits no partial report on stdout" "$([[ -z "$output" ]] && echo true || echo false)"

# Control: with jq present the report is produced exactly as before.
run_present "$REPORT" tests/fixtures/f01/valid-full
assert_value "audit-report, jq present: exits 0" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert_value "audit-report, jq present: stdout is a valid report" "$(echo "$output" | jq -e '.summary.passed != null' >/dev/null 2>&1 && echo true || echo false)"

# The other two tools stay soft dependencies: the report still generates and
# says in-band which source it could not read. Guarding jq must not have turned
# these into hard failures.
run_masked skill-validator "$REPORT" tests/fixtures/f01/valid-full
assert_value "audit-report, validator masked: still exits 0" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert_value "audit-report, validator masked: spec is null with a spec_error" "$(echo "$output" | jq -e '.spec == null and (.spec_error | test("skill-validator"))' >/dev/null 2>&1 && echo true || echo false)"

run_masked skillscore "$REPORT" tests/fixtures/f01/valid-full
assert_value "audit-report, skillscore masked: still exits 0" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert_value "audit-report, skillscore masked: quality is null with a quality_error" "$(echo "$output" | jq -e '.quality == null and (.quality_error | test("skillscore"))' >/dev/null 2>&1 && echo true || echo false)"

# --- A composer must not read an unreadable child payload as "no findings" ---
#
# This is where the guard's promise has to land. check-structure.sh emits a
# passed:false payload naming why it reached no verdict, and exits 3 — and the
# one machine consumer of that payload read it through `2>&1`, which mixed the
# diagnostic into the payload channel and left nothing that parses as JSON. With
# no else-branch, policy findings then stayed empty and the report said the
# skill passed with zero findings: a missing verdict read as a clean one, which
# is the exact failure the guard exists to prevent, surviving at the composition
# boundary.
#
# The two halves of the invariant: a readable failing payload must reach the
# report as findings, and an unreadable one must never read as no findings.

# A scripts copy whose check-structure.sh cannot reach a verdict, because the
# child it depends on is gone. It still emits a DEP002 payload and exits 3.
broken_child_tree="$mask_root/no-check-paths"
cp -R skills/skill-audit/scripts "$broken_child_tree"
rm -f "$broken_child_tree/check-paths.sh"

run_present "$broken_child_tree/audit-report.sh" tests/fixtures/f01/valid-full
assert_value "audit-report, policy source reached no verdict: report is still produced (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert_value "audit-report, policy source reached no verdict: output is valid JSON" "$(echo "$output" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
assert_value "audit-report, policy source reached no verdict: summary.passed is false" "$([[ "$(echo "$output" | jq -r '.summary.passed')" == "false" ]] && echo true || echo false)"
assert_value "audit-report, policy source reached no verdict: the DEP002 finding reaches the report" "$(echo "$output" | jq -e '.policy.findings[] | select(.rule == "DEP002")' >/dev/null 2>&1 && echo true || echo false)"
assert_value "audit-report, policy source reached no verdict: total_findings is not zero" "$([[ "$(echo "$output" | jq -r '.summary.total_findings')" -gt 0 ]] && echo true || echo false)"

# A scripts copy whose check-structure.sh produces nothing a consumer can parse.
# There is no payload to relay, so the report must say so rather than infer an
# empty findings list from it.
unreadable_tree="$mask_root/unreadable-policy"
cp -R skills/skill-audit/scripts "$unreadable_tree"
printf '#!/usr/bin/env bash\necho "not json at all"\nexit 3\n' > "$unreadable_tree/check-structure.sh"
chmod +x "$unreadable_tree/check-structure.sh"

run_present "$unreadable_tree/audit-report.sh" tests/fixtures/f01/valid-full
assert_value "audit-report, policy payload unreadable: report is still produced (exit 0)" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert_value "audit-report, policy payload unreadable: output is valid JSON" "$(echo "$output" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
assert_value "audit-report, policy payload unreadable: summary.passed is false" "$([[ "$(echo "$output" | jq -r '.summary.passed')" == "false" ]] && echo true || echo false)"
assert_value "audit-report, policy payload unreadable: names the source it could not read" "$(echo "$output" | jq -e '.policy_error | test("check-structure")' >/dev/null 2>&1 && echo true || echo false)"

# The same when the policy source is not there to run at all.
absent_tree="$mask_root/absent-policy"
cp -R skills/skill-audit/scripts "$absent_tree"
rm -f "$absent_tree/check-structure.sh"

run_present "$absent_tree/audit-report.sh" tests/fixtures/f01/valid-full
assert_value "audit-report, policy source absent: summary.passed is false" "$([[ "$(echo "$output" | jq -r '.summary.passed')" == "false" ]] && echo true || echo false)"
assert_value "audit-report, policy source absent: names the source it could not run" "$(echo "$output" | jq -e '.policy_error | test("check-structure")' >/dev/null 2>&1 && echo true || echo false)"

# Control: with the policy source intact the report is unchanged, and says so.
run_present "$REPORT" tests/fixtures/f01/valid-full
assert_value "audit-report, policy source intact: summary.passed is true" "$([[ "$(echo "$output" | jq -r '.summary.passed')" == "true" ]] && echo true || echo false)"
assert_value "audit-report, policy source intact: policy_error is null" "$([[ "$(echo "$output" | jq -r '.policy_error')" == "null" ]] && echo true || echo false)"
assert_value "audit-report, policy source intact: stdout stays a single clean document" "$(echo "$output" | jq -e '.summary.total_findings == 0' >/dev/null 2>&1 && echo true || echo false)"

# A failing policy payload still reaches the report as findings, unchanged: the
# fix must not have turned "readable and failing" into "unreadable".
run_present "$REPORT" tests/fixtures/f01/missing-script-ref
assert_value "audit-report, policy source reports a path failure: PT001 reaches the report" "$(echo "$output" | jq -e '.policy.findings[] | select(.rule == "PT001")' >/dev/null 2>&1 && echo true || echo false)"
assert_value "audit-report, policy source reports a path failure: policy_error is null" "$([[ "$(echo "$output" | jq -r '.policy_error')" == "null" ]] && echo true || echo false)"

# --- The composer over the same shape-space as its child ---
#
# `audit-report.sh` is the other consumer of the documented
# {"findings": [{level, rule, message}, ...], "passed": bool} payload, and it
# had the same split as `check-structure.sh` did: it proved one thing about the
# payload — that `.findings` is an array — and then read a great deal more,
# selecting on `.level` and calling `startswith` on `.rule` of every element.
# A payload that satisfied the partial check and broke the full assumption took
# jq down inside the final merge, and the composer exited 5 with no report at
# all: outside the {0, 3} its header documents, and the opposite of the
# "the report still generates, and names the source it could not read" promise
# the whole file is built around.
#
# So the composer is driven over the same shape-space, and the contract is the
# same one in both places: the payload is the documented shape or it is a source
# this report could not read.

# policy_stub_tree <name> <payload> [exit-code]
#
# The exit code is a parameter because a policy source's status and its payload
# are two statements about one run, and this suite has cases about the payload
# and cases about their agreement. Wiring every stub to exit 0 made the
# payload-shape rows above also assert that a failing payload at exit 0 is fine,
# which it is not: the real check-structure.sh exits 2 when it reports a policy
# failure. Each row states the status its payload belongs with, so the rows about
# shape stay rows about shape.
policy_stub_tree() {
  local name="$1"
  local payload="$2"
  local exit_code="${3:-0}"
  local dir="$mask_root/policy-$name"
  if [[ ! -d "$dir" ]]; then
    cp -R skills/skill-audit/scripts "$dir"
    printf '%s' "$payload" > "$dir/stub-policy-payload"
    printf '#!/usr/bin/env bash\ncat "$(dirname "$0")/stub-policy-payload"\nexit %s\n' "$exit_code" \
      > "$dir/check-structure.sh"
    chmod +x "$dir/check-structure.sh"
  fi
  echo "$dir/audit-report.sh"
}

P_FAIL='{"level": "fail", "rule": "PT001", "message": "script/reference path not found: ./scripts/nope.sh"}'
P_UNVER='{"level": "unverified", "rule": "PATH", "message": "dynamic/glob path: ./scripts/*"}'

# name | payload | readable: yes = relayed as findings, no = named in policy_error
policy_names=()
policy_payloads=()
policy_readable=()
policy_status=()
policy_case() {
  policy_names+=("$1")
  policy_payloads+=("$2")
  policy_readable+=("$3")
  policy_status+=("${4:-0}")
}

policy_case conforming-pass      "{\"passed\": true, \"findings\": []}"                   yes
policy_case conforming-fail      "{\"passed\": false, \"findings\": [$P_FAIL]}"            yes 2
policy_case conforming-unverified "{\"passed\": true, \"findings\": [$P_UNVER]}"           yes
policy_case top-number           '123'                                                     no
policy_case top-error-object     '{"error": "x"}'                                          no
policy_case passed-absent        '{"findings": []}'                                        no
policy_case passed-string        '{"passed": "yes", "findings": []}'                       no
policy_case findings-absent      '{"passed": true}'                                        no
policy_case findings-string      '{"passed": true, "findings": "oops"}'                    no
policy_case findings-null        '{"passed": true, "findings": null}'                      no
policy_case element-number       '{"passed": true, "findings": [42]}'                      no
policy_case element-null         '{"passed": true, "findings": [null]}'                    no
# A conforming element beside a non-conforming one: the one arrangement
# every row above leaves indistinguishable from "some element conforms".
policy_case element-mixed        "{\"passed\": false, \"findings\": [$P_FAIL, 42]}"       no
policy_case element-no-rule      '{"passed": false, "findings": [{"level": "fail", "message": "m"}]}'             no
policy_case element-rule-number  '{"passed": false, "findings": [{"level": "fail", "rule": 123, "message": "m"}]}' no
policy_case not-json             'not json at all'                                         no
policy_case empty                ''                                                        no
policy_case trailing-garbage     '{"passed": true, "findings": []} {"x": 1}'               no

for i in "${!policy_names[@]}"; do
  pname="${policy_names[$i]}"
  run_present "$(policy_stub_tree "$pname" "${policy_payloads[$i]}" "${policy_status[$i]}")" tests/fixtures/f01/valid-full

  assert_value "audit-report, policy payload $pname: exits inside the documented set" \
    "$([[ $code -eq 0 || $code -eq 3 ]] && echo true || echo false)"
  assert_value "audit-report, policy payload $pname: a report is still produced" \
    "$([[ $code -eq 0 ]] && echo "$output" | jq -e '.summary.passed != null' >/dev/null 2>&1 && echo true || echo false)"

  if [[ "${policy_readable[$i]}" == "yes" ]]; then
    assert_value "audit-report, policy payload $pname: is read, so no policy_error is raised" \
      "$([[ "$(echo "$output" | jq -r '.policy_error' 2>/dev/null)" == "null" ]] && echo true || echo false)"
  else
    assert_value "audit-report, policy payload $pname: names the source it could not read" \
      "$(echo "$output" | jq -e '.policy_error | test("check-structure")' >/dev/null 2>&1 && echo true || echo false)"
    assert_value "audit-report, policy payload $pname: does not claim the skill passed over a source it could not read" \
      "$([[ "$(echo "$output" | jq -r '.summary.passed' 2>/dev/null)" == "false" ]] && echo true || echo false)"
  fi
done

# The policy source carries its own verdict, and the report's job is to compose
# the verdicts it was given, not to re-derive them from the findings that came
# with them. A source saying it failed, for a reason it did not enumerate as a
# `level: "fail"` finding, is still a source saying it failed.
run_present "$(policy_stub_tree says-failed '{"passed": false, "findings": []}' 2)" tests/fixtures/f01/valid-full
assert_value "audit-report, policy source says it failed: summary.passed is false" \
  "$([[ "$(echo "$output" | jq -r '.summary.passed')" == "false" ]] && echo true || echo false)"
assert_value "audit-report, policy source says it failed: the source is read, not errored" \
  "$([[ "$(echo "$output" | jq -r '.policy_error')" == "null" ]] && echo true || echo false)"

# And a source saying it passed, over a skill whose license the report itself
# checks, still fails on that: the composer's own PL001 is not the child's to
# overrule.
run_present "$(policy_stub_tree says-passed '{"passed": true, "findings": []}')" tests/fixtures/f01/no-license
assert_value "audit-report, policy source passes but the license is missing: summary.passed is false" \
  "$([[ "$(echo "$output" | jq -r '.summary.passed')" == "false" ]] && echo true || echo false)"
assert_value "audit-report, policy source passes but the license is missing: PL001 is in the findings" \
  "$(echo "$output" | jq -e '.policy.findings[] | select(.rule == "PL001")' >/dev/null 2>&1 && echo true || echo false)"

# --- Every source is read only as far as its shape has been proven ---
#
# `jq -e .` proves a source parsed and nothing more. The merge then indexes
# `.passed`, `.errors` and `.warnings` out of the spec source and
# `.overallScore.percentage` out of the quality source — fields of a shape
# nothing had checked. A source that parses without carrying them takes jq down
# inside the final merge and the composer exits 5 with no report at all, or,
# over a stream of documents, exits 2 from `--argjson` with jq's usage dump on
# stderr. Both are outside the {0, 3} this file's header documents, and both are
# the opposite of the promise the file is built around: the report still
# generates, and names the source it could not read.
#
# This is the same split the policy table above walks, at the two sources one
# level up, and it is closed the same way. The contract is one sentence in all
# three places: a source is proven to carry the shape about to be read out of it
# — one document, of the type being indexed — or it is a source this report
# could not read. What that shape is differs per source, because what is read
# out of them differs.
#
# A source stub is the symlink farm above minus the tool, with a directory in
# front of it holding a script that emits a canned payload. Nothing is deleted,
# moved or uninstalled.

source_stub_path() {
  local tool="$1"
  local name="$2"
  local payload="$3"
  local dir="$mask_root/stub-$tool-$name"
  if [[ ! -d "$dir" ]]; then
    mkdir -p "$dir"
    printf '%s' "$payload" > "$dir/payload"
    printf '#!/usr/bin/env bash\ncat "$(dirname "$0")/payload"\nexit 0\n' > "$dir/$tool"
    chmod +x "$dir/$tool"
  fi
  echo "$dir:$(masked_path "$tool")"
}

# The spec source is read as `.passed`, `.errors` and `.warnings` of one
# document, so those three fields are what has to hold — not merely that the
# document is an object.
#
# `type == "object"` was the whole claim, and the cases below drove it with
# non-objects only. That is the blind spot, and it is the same defect one level
# in that the guard's own comment warns about: an object carrying none of the
# three fields passed the claim, and the merge then read `.passed` out of it and
# got null. `{"foo": 1}` produced a failing verdict with `spec_error: null` —
# the defect G3-04 was raised for, reappearing through a source that parsed.
# `{"passed": "yes"}` was worse: jq reads a non-empty string as truthy, so
# `summary.passed` came out **true**, a verdict read off a string.
#
# For contrast, `check-frontmatter.sh` reads the same source and gets this
# right; its claim reaches `.errors` because `.errors` is the whole of what it
# reads. The two claims differ because the reads differ, which is the rule, not
# an inconsistency: the claim reaches as far as the caller reads and no further.
spec_names=()
spec_payloads=()
spec_readable=()
spec_case() {
  spec_names+=("$1")
  spec_payloads+=("$2")
  spec_readable+=("$3")
}

spec_case conforming    '{"passed": true, "errors": 0, "warnings": 0}'   yes
spec_case no-warnings   '{"passed": true, "errors": 0}'                  yes
spec_case top-number    '0'                                             no
spec_case top-array     '[1, 2]'                                        no
spec_case top-string    '"a verdict"'                                   no
spec_case top-boolean   'true'                                          no
spec_case top-null      'null'                                          no
spec_case two-documents '{"passed": true} {"passed": false}'            no
spec_case not-json      'not json at all'                               no
# An object, one level in — the half the table never drove.
spec_case no-fields     '{"foo": 1}'                                    no
spec_case passed-string '{"passed": "yes", "errors": 0}'                no
spec_case passed-number '{"passed": 1, "errors": 0}'                    no
spec_case passed-null   '{"passed": null, "errors": 0}'                  no
spec_case errors-string '{"passed": true, "errors": "none"}'            no
spec_case errors-absent '{"passed": true, "warnings": 0}'               no
spec_case warnings-str  '{"passed": true, "errors": 0, "warnings": "x"}' no

for i in "${!spec_names[@]}"; do
  vname="${spec_names[$i]}"
  run_on_path "$(source_stub_path skill-validator "$vname" "${spec_payloads[$i]}")" \
    "$REPORT" tests/fixtures/f01/valid-full

  assert_value "audit-report, spec source $vname: exits inside the documented set" \
    "$([[ $code -eq 0 || $code -eq 3 ]] && echo true || echo false)"
  assert_value "audit-report, spec source $vname: a report is still produced" \
    "$([[ $code -eq 0 ]] && echo "$output" | jq -e '.summary.passed != null' >/dev/null 2>&1 && echo true || echo false)"

  if [[ "${spec_readable[$i]}" == "yes" ]]; then
    assert_value "audit-report, spec source $vname: is read, so the report carries the source's verdict" \
      "$(echo "$output" | jq -e '.spec.passed == true and .summary.spec_passed == true and .spec_error == null' >/dev/null 2>&1 && echo true || echo false)"
  else
    assert_value "audit-report, spec source $vname: the source is not read" \
      "$(echo "$output" | jq -e '.spec == null' >/dev/null 2>&1 && echo true || echo false)"
    assert_value "audit-report, spec source $vname: no field is read out of a source it could not read" \
      "$(echo "$output" | jq -e '.summary.spec_errors == null and .summary.spec_warnings == null' >/dev/null 2>&1 && echo true || echo false)"
    assert_value "audit-report, spec source $vname: does not claim the skill passed over a source it could not read" \
      "$([[ "$(echo "$output" | jq -r '.summary.spec_passed' 2>/dev/null)" == "false" && "$(echo "$output" | jq -r '.summary.passed' 2>/dev/null)" == "false" ]] && echo true || echo false)"
  fi
done

# The quality source is read two levels in — `.overallScore.percentage` and
# `.overallScore.letterGrade` — so proving the top level and then indexing
# `.overallScore` would be the same defect one level down. The claim reaches as
# far as the read reaches, and no further: a source with no `overallScore` at
# all is read, and leaves the score null, because that is what the read yields.
quality_names=()
quality_payloads=()
quality_readable=()
quality_case() {
  quality_names+=("$1")
  quality_payloads+=("$2")
  quality_readable+=("$3")
}

quality_case conforming    '{"overallScore": {"percentage": 80, "letterGrade": "B-"}}' yes
quality_case score-absent  '{"categories": []}'                                        yes
quality_case score-number  '{"overallScore": 80}'                                      no
quality_case score-string  '{"overallScore": "B-"}'                                    no
quality_case score-array   '{"overallScore": [80]}'                                    no
quality_case top-boolean   'true'                                                      no
quality_case top-number    '7'                                                         no
quality_case top-array     '[]'                                                        no
quality_case two-documents '{"overallScore": {}} {"overallScore": {}}'                 no
quality_case not-json      'not json at all'                                           no

for i in "${!quality_names[@]}"; do
  qname="${quality_names[$i]}"
  run_on_path "$(source_stub_path skillscore "$qname" "${quality_payloads[$i]}")" \
    "$REPORT" tests/fixtures/f01/valid-full

  assert_value "audit-report, quality source $qname: exits inside the documented set" \
    "$([[ $code -eq 0 || $code -eq 3 ]] && echo true || echo false)"
  assert_value "audit-report, quality source $qname: a report is still produced" \
    "$([[ $code -eq 0 ]] && echo "$output" | jq -e '.summary.passed != null' >/dev/null 2>&1 && echo true || echo false)"
  # Quality is a score, not a verdict. However this source turns out, the
  # verdict over a skill that passes is still a pass.
  assert_value "audit-report, quality source $qname: the verdict is the quality source's to inform, not to decide" \
    "$([[ "$(echo "$output" | jq -r '.summary.passed' 2>/dev/null)" == "true" ]] && echo true || echo false)"

  if [[ "${quality_readable[$i]}" == "yes" ]]; then
    assert_value "audit-report, quality source $qname: is read, so no quality_error is raised" \
      "$(echo "$output" | jq -e '.quality != null and .quality_error == null' >/dev/null 2>&1 && echo true || echo false)"
  else
    assert_value "audit-report, quality source $qname: the source is not read, and is named" \
      "$(echo "$output" | jq -e '.quality == null and .quality_error != null' >/dev/null 2>&1 && echo true || echo false)"
    assert_value "audit-report, quality source $qname: no score is read out of a source it could not read" \
      "$(echo "$output" | jq -e '.summary.quality_score == null and .summary.quality_grade == null' >/dev/null 2>&1 && echo true || echo false)"
  fi
done

#
# The *_error fields carried the source's own output verbatim, and a source that
# exited 0 printing nothing therefore left `spec_error: null` beside
# `spec_passed: false`: the empty answer and a working source produced the same
# empty string, jq rendered it as null, and the report asserted a spec failure
# while naming no reason for it. A reader cannot tell that from a skill whose
# spec genuinely failed with no message.
#
# "The source said nothing" is a fact about the source, so the report states it.
# The source's status is stated too — it used to be discarded with `|| true`, so
# a source that died was indistinguishable from one that answered badly.
#
# Each case drives a source that is present and useless, in the three shapes a
# source can be useless in, and each is asserted on the same three things: the
# verdict does not claim a pass, the error field is not null, and the error
# field names the source rather than being whatever the source happened to say.

# silent_source_path <name> <tool> <exit> <stdout>
silent_source_path() {
  local name="$1"
  local tool="$2"
  local exit_code="$3"
  local said="$4"
  local dir="$mask_root/quiet-$name"
  if [[ ! -d "$dir" ]]; then
    mkdir -p "$dir"
    printf '%s' "$said" > "$dir/said"
    printf '#!/usr/bin/env bash\ncat "$(dirname "$0")/said"\nexit %s\n' "$exit_code" > "$dir/$tool"
    chmod +x "$dir/$tool"
  fi
  echo "$dir:$PATH"
}

reason_cases=0
for rcase in "spec|skill-validator|0||exited 0 and said nothing" \
             "spec|skill-validator|4|boom|exited 4 with text that is not a report" \
             "spec|skill-validator|0|{\"a\":1}{\"b\":2}|answered with two documents" \
             "quality|skillscore|0||exited 0 and said nothing" \
             "quality|skillscore|9|boom|exited 9 with text that is not a report"; do
  rfield="${rcase%%|*}"; rrest="${rcase#*|}"
  rtool="${rrest%%|*}";  rrest="${rrest#*|}"
  rexit="${rrest%%|*}";  rrest="${rrest#*|}"
  rsaid="${rrest%%|*}";  rwhy="${rrest#*|}"
  reason_cases=$((reason_cases + 1))
  run_on_path "$(silent_source_path "$rfield-$rexit-$reason_cases" "$rtool" "$rexit" "$rsaid")" \
    "$REPORT" tests/fixtures/f01/valid-full
  assert_value "audit-report, $rfield source $rwhy: a report is still produced (exit 0)" \
    "$([[ $code -eq 0 ]] && echo true || echo false)"
  assert_value "audit-report, $rfield source $rwhy: the source is not read" \
    "$(echo "$output" | jq -e --arg f "$rfield" '.[$f] == null' >/dev/null 2>&1 && echo true || echo false)"
  assert_value "audit-report, $rfield source $rwhy: the error field is not null" \
    "$(echo "$output" | jq -e --arg f "${rfield}_error" '.[$f] != null' >/dev/null 2>&1 && echo true || echo false)"
  assert_value "audit-report, $rfield source $rwhy: the error names the source and the status" \
    "$(echo "$output" | jq -e --arg f "${rfield}_error" --arg t "$rtool" --arg c "$rexit" \
        '.[$f] | test($t) and test("exited " + $c)' >/dev/null 2>&1 && echo true || echo false)"
done

echo "  unreadable-source reason cases driven: $reason_cases"
assert_value "the unreadable-source reason cases were enumerated, not read as empty" \
  "$([[ "$reason_cases" -eq 5 ]] && echo true || echo false)"

# --- A source's status is read against its payload -----------------------------
#
# The shape half above is only half. Both soft sources have their status
# captured and then never compared to what they said, so a source that
# contradicted itself was believed:
#
#   skill-validator, a conforming payload at exit 1  -> `passed: true`, unread
#   skillscore, a conforming report at exit 7        -> `quality_score: 99`
#
# The second is the sharper one, because the same source read by
# `check-quality.sh` is refused: skillscore's contract is "0 when it produced a
# report, nonzero when it did not", so exit 7 says there is no report, and
# `check-quality.sh` exits 3 over the very payload this file published a grade
# from. One source, two readers, two answers — which is precisely the drift the
# guard's shared predicates exist to prevent, arriving through the status
# instead of through the shape.
#
# The policy source, one level down in this same file, has read its status
# against its payload all along. These two are brought up to it rather than
# given a mechanism of their own.
#
# A status and a payload are the source's two statements about one run. Where
# they contradict each other the source has told us nothing, and this file does
# what it does with every other unreadable source: names it, leaves the field
# null, and still produces a report.

# skill-validator's documented statuses are 0 clean, 1 errors, 2 warnings only,
# 3 usage error — the same set check-frontmatter.sh enumerates over the same
# tool. 0 and 2 assert no spec error; 1 asserts at least one.
spec_status_cases=0
for scase in "0|{\"passed\": true, \"errors\": 3, \"warnings\": 0}|exit 0 beside a payload reporting three errors" \
             "2|{\"passed\": true, \"errors\": 2, \"warnings\": 1}|exit 2 beside a payload reporting two errors" \
             "1|{\"passed\": true, \"errors\": 0, \"warnings\": 0}|exit 1 beside a payload reporting none" \
             "3|{\"passed\": true, \"errors\": 0, \"warnings\": 0}|a usage error beside a conforming payload" \
             "9|{\"passed\": true, \"errors\": 0, \"warnings\": 0}|a status outside its documented set"; do
  sexit="${scase%%|*}"; srest="${scase#*|}"
  spayload="${srest%%|*}"; swhy="${srest#*|}"
  spec_status_cases=$((spec_status_cases + 1))
  run_on_path "$(silent_source_path "spec-status-$spec_status_cases" skill-validator "$sexit" "$spayload")" \
    "$REPORT" tests/fixtures/f01/valid-full

  assert_value "audit-report, spec source $swhy: exits inside the documented set" \
    "$([[ $code -eq 0 || $code -eq 3 ]] && echo true || echo false)"
  assert_value "audit-report, spec source $swhy: the source is not read" \
    "$(echo "$output" | jq -e '.spec == null' >/dev/null 2>&1 && echo true || echo false)"
  assert_value "audit-report, spec source $swhy: names the source and its status" \
    "$(echo "$output" | jq -e '.spec_error != null and (.spec_error | test("skill-validator"))' >/dev/null 2>&1 && echo true || echo false)"
  assert_value "audit-report, spec source $swhy: claims no pass over a source it could not read" \
    "$([[ "$(echo "$output" | jq -r '.summary.spec_passed' 2>/dev/null)" == "false" && "$(echo "$output" | jq -r '.summary.passed' 2>/dev/null)" == "false" ]] && echo true || echo false)"
done
echo "  spec-source status cases driven: $spec_status_cases"
assert_value "the spec-source status cases were enumerated, not read as empty" \
  "$([[ "$spec_status_cases" -eq 5 ]] && echo true || echo false)"

# The controls, one per status that carries a verdict. Without them the cases
# above would pass against a file that refused every status there is.
spec_agree_cases=0
for scase in "0|{\"passed\": true, \"errors\": 0, \"warnings\": 0}|true|a clean pass at exit 0" \
             "2|{\"passed\": true, \"errors\": 0, \"warnings\": 1}|true|warnings only at exit 2" \
             "1|{\"passed\": false, \"errors\": 1, \"warnings\": 0}|false|a spec failure at exit 1"; do
  sexit="${scase%%|*}"; srest="${scase#*|}"
  spayload="${srest%%|*}"; srest="${srest#*|}"
  swant="${srest%%|*}"; swhy="${srest#*|}"
  spec_agree_cases=$((spec_agree_cases + 1))
  run_on_path "$(silent_source_path "spec-agree-$spec_agree_cases" skill-validator "$sexit" "$spayload")" \
    "$REPORT" tests/fixtures/f01/valid-full
  assert_value "audit-report, spec source $swhy: is read, with no spec_error" \
    "$(echo "$output" | jq -e '.spec != null and .spec_error == null' >/dev/null 2>&1 && echo true || echo false)"
  assert_value "audit-report, spec source $swhy: the report carries the source's own verdict" \
    "$([[ "$(echo "$output" | jq -r '.summary.spec_passed' 2>/dev/null)" == "$swant" ]] && echo true || echo false)"
done
echo "  spec-source agreement controls driven: $spec_agree_cases"
assert_value "the spec-source agreement controls were enumerated, not read as empty" \
  "$([[ "$spec_agree_cases" -eq 3 ]] && echo true || echo false)"

# skillscore says it produced a report by exiting 0, so any nonzero status is a
# source with no report to read, whatever arrived on its stdout.
quality_status_cases=0
for qcase in "7|{\"overallScore\": {\"percentage\": 99, \"letterGrade\": \"A\"}}|a conforming report at exit 7" \
             "1|{\"overallScore\": {\"percentage\": 50, \"letterGrade\": \"F\"}}|a conforming report at exit 1"; do
  qexit="${qcase%%|*}"; qrest="${qcase#*|}"
  qpayload="${qrest%%|*}"; qwhy="${qrest#*|}"
  quality_status_cases=$((quality_status_cases + 1))
  run_on_path "$(silent_source_path "quality-status-$quality_status_cases" skillscore "$qexit" "$qpayload")" \
    "$REPORT" tests/fixtures/f01/valid-full

  assert_value "audit-report, quality source $qwhy: exits inside the documented set" \
    "$([[ $code -eq 0 || $code -eq 3 ]] && echo true || echo false)"
  assert_value "audit-report, quality source $qwhy: the source is not read, and is named" \
    "$(echo "$output" | jq -e '.quality == null and .quality_error != null and (.quality_error | test("skillscore"))' >/dev/null 2>&1 && echo true || echo false)"
  assert_value "audit-report, quality source $qwhy: no score is read out of a source that reported none" \
    "$(echo "$output" | jq -e '.summary.quality_score == null and .summary.quality_grade == null' >/dev/null 2>&1 && echo true || echo false)"
  # Quality informs, it does not decide. A refused score leaves the verdict be.
  assert_value "audit-report, quality source $qwhy: the verdict over a passing skill is still a pass" \
    "$([[ "$(echo "$output" | jq -r '.summary.passed' 2>/dev/null)" == "true" ]] && echo true || echo false)"
  # The same payload, put to the script whose whole job is that one source.
  # Two readers of one source must not disagree about whether it answered.
  run_on_path "$(silent_source_path "quality-status-$quality_status_cases" skillscore "$qexit" "$qpayload")" \
    skills/skill-audit/scripts/check-quality.sh tests/fixtures/f01/valid-full
  assert_value "check-quality, $qwhy: refuses it too, so the two readers agree" \
    "$([[ $code -eq 3 ]] && echo true || echo false)"
done
echo "  quality-source status cases driven: $quality_status_cases"
assert_value "the quality-source status cases were enumerated, not read as empty" \
  "$([[ "$quality_status_cases" -eq 2 ]] && echo true || echo false)"

# The control: exit 0 with the same report is read.
run_on_path "$(silent_source_path "quality-agree" skillscore 0 '{"overallScore": {"percentage": 99, "letterGrade": "A"}}')" \
  "$REPORT" tests/fixtures/f01/valid-full
assert_value "audit-report, quality source conforming at exit 0: is read, with the score" \
  "$(echo "$output" | jq -e '.quality_error == null and .summary.quality_score == 99' >/dev/null 2>&1 && echo true || echo false)"


# A spec source this report cannot read leaves no spec verdict to claim.
run_on_path "$(silent_source_path "spec-verdict" skill-validator 0 "")" "$REPORT" tests/fixtures/f01/valid-full
assert_value "audit-report, spec source unreadable: summary.passed is false" \
  "$([[ "$(echo "$output" | jq -r '.summary.passed')" == "false" ]] && echo true || echo false)"
assert_value "audit-report, spec source unreadable: no error count is read out of it" \
  "$(echo "$output" | jq -e '.summary.spec_errors == null' >/dev/null 2>&1 && echo true || echo false)"

# The control. Without it every case above would pass against a report that
# named a reason for every source, readable or not.
run_present "$REPORT" tests/fixtures/f01/valid-full
assert_value "audit-report, every source readable: no reason is named for any of them" \
  "$(echo "$output" | jq -e '.spec_error == null and .quality_error == null and .policy_error == null' >/dev/null 2>&1 && echo true || echo false)"

# A policy source whose status and payload disagree is two statements about one
# run that contradict each other, and there is no half of it to report.
disagreeing_tree="$mask_root/disagreeing-policy"
cp -R skills/skill-audit/scripts "$disagreeing_tree"
printf '#!/usr/bin/env bash\nprintf %s\nexit 2\n' \
  "'{\"findings\": [], \"passed\": true}\\n'" > "$disagreeing_tree/check-structure.sh"
chmod +x "$disagreeing_tree/check-structure.sh"

run_present "$disagreeing_tree/audit-report.sh" tests/fixtures/f01/valid-full
assert_value "audit-report, policy status and payload disagree: summary.passed is false" \
  "$([[ "$(echo "$output" | jq -r '.summary.passed')" == "false" ]] && echo true || echo false)"
assert_value "audit-report, policy status and payload disagree: the report names the contradiction" \
  "$(echo "$output" | jq -e '.policy_error | test("check-structure") and test("2")' >/dev/null 2>&1 && echo true || echo false)"
assert_value "audit-report, policy status and payload disagree: it claims no findings from either half" \
  "$(echo "$output" | jq -e '.summary.total_findings == 0 and (.policy.findings | length) == 0' >/dev/null 2>&1 && echo true || echo false)"

# A policy source exiting outside its own contract reached no verdict at all.
outside_tree="$mask_root/outside-contract-policy"
cp -R skills/skill-audit/scripts "$outside_tree"
printf '#!/usr/bin/env bash\nprintf %s\nexit 42\n' \
  "'{\"findings\": [], \"passed\": false}\\n'" > "$outside_tree/check-structure.sh"
chmod +x "$outside_tree/check-structure.sh"

run_present "$outside_tree/audit-report.sh" tests/fixtures/f01/valid-full
assert_value "audit-report, policy source exits outside its contract: summary.passed is false" \
  "$([[ "$(echo "$output" | jq -r '.summary.passed')" == "false" ]] && echo true || echo false)"
assert_value "audit-report, policy source exits outside its contract: the report names the status" \
  "$(echo "$output" | jq -e '.policy_error | test("42")' >/dev/null 2>&1 && echo true || echo false)"

harness_summary
