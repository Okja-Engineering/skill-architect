#!/usr/bin/env bash
# F02 tests: unified audit report (audit-report.sh)
# Verifies the composer merges skill-validator, skillscore, and house-policy
# findings into a single machine-readable JSON document.
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
  assert "$label" "$([[ "$actual" == "$expected" ]] && echo true || echo false)"
}

# Helper: assert a jq path is greater than a number.
assert_jq_gt() {
  local label="$1"
  local path="$2"
  local threshold="$3"
  local actual
  actual=$(echo "$output" | jq -r "$path")
  assert "$label" "$([[ "$actual" -gt "$threshold" ]] && echo true || echo false)"
}

# Helper: assert a jq path exists (is not null).
assert_jq_exists() {
  local label="$1"
  local path="$2"
  assert "$label" "$(echo "$output" | jq -e "$path" >/dev/null 2>&1 && echo true || echo false)"
}

# Helper: assert a finding with a given rule exists in policy.findings.
assert_has_finding() {
  local label="$1"
  local rule="$2"
  assert "$label" "$(echo "$output" | jq -e --arg r "$rule" '.policy.findings[] | select(.rule == $r)' >/dev/null 2>&1 && echo true || echo false)"
}

# Helper: assert a finding with a given rule does NOT exist.
assert_no_finding() {
  local label="$1"
  local rule="$2"
  assert "$label" "$(echo "$output" | jq -e --arg r "$rule" '.policy.findings[] | select(.rule == $r)' >/dev/null 2>&1 && echo false || echo true)"
}

# --- Top-level JSON shape ---

run_report tests/fixtures/f01/valid-full
assert "valid-full: report exits 0" "$([[ $code -eq 0 ]] && echo true || echo false)"
assert "valid-full: output is valid JSON" "$(echo "$output" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
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
assert "valid-full: spec has results array" "$(echo "$output" | jq -e '.spec.results | length > 0' >/dev/null 2>&1 && echo true || echo false)"
assert_jq_exists "valid-full: spec has token_counts" '.spec.token_counts'
assert_jq_exists "valid-full: spec has content_analysis" '.spec.content_analysis'
assert_jq_exists "valid-full: spec has contamination_analysis" '.spec.contamination_analysis'

# --- Quality source (skillscore) is nested ---

assert_jq_exists "valid-full: quality has overallScore" '.quality.overallScore'
assert "valid-full: quality has 7 categories" "$(echo "$output" | jq -e '.quality.categories | length == 7' >/dev/null 2>&1 && echo true || echo false)"

# --- PL001: no-license fixture (license is house policy, not spec) ---

run_report tests/fixtures/f01/no-license
assert_jq_eq "no-license: summary.passed is false" '.summary.passed' 'false'
assert_jq_eq "no-license: summary.spec_passed is true" '.summary.spec_passed' 'true'
assert_has_finding "no-license: reports PL001 finding" 'PL001'
assert_jq_gt "no-license: policy_failures > 0" '.summary.policy_failures' 0

# --- PL002-PL005: missing headings, code blocks via bad skill ---

tmp_bad="$(mktemp -d)"
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
assert "skill-audit self-audit: report exits 0" "$([[ $code -eq 0 ]] && echo true || echo false)"
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
assert "check-paths.sh text: valid-full passes (exit 0)" "$([[ $pcode -eq 0 ]] && echo true || echo false)"

run_paths_text tests/fixtures/f01/missing-script-ref
assert "check-paths.sh text: missing-script-ref fails (exit 1)" "$([[ $pcode -eq 1 ]] && echo true || echo false)"
assert "check-paths.sh text: reports PT001" "$(echo "$poutput" | grep -q 'PT001' && echo true || echo false)"

# JSON mode works
run_paths_json tests/fixtures/f01/valid-full
assert "check-paths.sh --json: valid-full passes (exit 0)" "$([[ $pcode -eq 0 ]] && echo true || echo false)"
assert "check-paths.sh --json: output is valid JSON" "$(echo "$poutput" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
assert "check-paths.sh --json: has findings array" "$(echo "$poutput" | jq -e '.findings' >/dev/null 2>&1 && echo true || echo false)"
assert "check-paths.sh --json: passed is true" "$([[ "$(echo "$poutput" | jq -r '.passed')" == "true" ]] && echo true || echo false)"

run_paths_json tests/fixtures/f01/missing-script-ref
assert "check-paths.sh --json: missing-script-ref fails (exit 1)" "$([[ $pcode -eq 1 ]] && echo true || echo false)"
assert "check-paths.sh --json: passed is false" "$([[ "$(echo "$poutput" | jq -r '.passed')" == "false" ]] && echo true || echo false)"
assert "check-paths.sh --json: has PT001 finding" "$(echo "$poutput" | jq -e '.findings[] | select(.rule == "PT001")' >/dev/null 2>&1 && echo true || echo false)"

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
assert "check-structure.sh text: valid-full passes (exit 0)" "$([[ $scode -eq 0 ]] && echo true || echo false)"

# JSON mode works
run_struct_json tests/fixtures/f01/valid-full
assert "check-structure.sh --json: valid-full passes (exit 0)" "$([[ $scode -eq 0 ]] && echo true || echo false)"
assert "check-structure.sh --json: output is valid JSON" "$(echo "$soutput" | jq -e . >/dev/null 2>&1 && echo true || echo false)"
assert "check-structure.sh --json: passed is true" "$([[ "$(echo "$soutput" | jq -r '.passed')" == "true" ]] && echo true || echo false)"

run_struct_json tests/fixtures/f01/missing-script-ref
assert "check-structure.sh --json: missing-script-ref has findings" "$([[ "$(echo "$soutput" | jq -r '.findings | length')" -gt 0 ]] && echo true || echo false)"
assert "check-structure.sh --json: missing-script-ref includes PT001 from check-paths" "$(echo "$soutput" | jq -e '.findings[] | select(.rule == "PT001")' >/dev/null 2>&1 && echo true || echo false)"

echo
echo "$pass passed, $fail failed"
if [[ "$fail" -gt 0 ]]; then
  exit 1
fi
