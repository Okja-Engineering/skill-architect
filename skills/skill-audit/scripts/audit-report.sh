#!/usr/bin/env bash
# Produce a unified machine-readable audit report for a skill directory.
# Composes three JSON sources into one document:
#   1. skill-validator check -o json  (spec, structure, content, contamination)
#   2. skillscore --json              (7-dimension quality scoring)
#   3. House-policy checks            (PL001-PL005, PT001-PT002)
# Exit codes: 0=report generated, 3=execution error.
set -euo pipefail

skill_dir=""
for arg in "$@"; do
  case "$arg" in
    -*) echo "Usage: audit-report.sh <skill-dir>" >&2; exit 3 ;;
    *) skill_dir="$arg" ;;
  esac
done

if [[ -z "$skill_dir" ]]; then
  echo "Usage: audit-report.sh <skill-dir>" >&2
  exit 3
fi

skill_md="$skill_dir/SKILL.md"
if [[ ! -f "$skill_md" ]]; then
  echo "ERROR: SKILL.md not found in $skill_dir" >&2
  exit 3
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

# --- Source 1: skill-validator (spec + structure + content + contamination) ---
spec_json="null"
spec_error=""
if command -v skill-validator &>/dev/null; then
  spec_raw="$(skill-validator check -o json "$skill_dir" 2>&1)" || true
  if echo "$spec_raw" | jq -e . >/dev/null 2>&1; then
    spec_json="$spec_raw"
  else
    spec_error="$spec_raw"
  fi
else
  spec_error="skill-validator not found. Install with: brew install agent-ecosystem/tap/skill-validator"
fi

# --- Source 2: skillscore (7-dimension quality scoring) ---
quality_json="null"
quality_error=""
if command -v skillscore &>/dev/null; then
  quality_raw="$(skillscore "$skill_dir" --json 2>&1)" || true
  if echo "$quality_raw" | jq -e . >/dev/null 2>&1; then
    quality_json="$quality_raw"
  else
    quality_error="$quality_raw"
  fi
else
  quality_error="skillscore not found. Install with: npm install -g skillscore"
fi

# --- Source 3: House-policy checks (PL002-PL005, PT001-PT002) via check-structure.sh ---
policy_findings="[]"
if [[ -x "$script_dir/check-structure.sh" ]]; then
  struct_raw="$("$script_dir/check-structure.sh" --json "$skill_dir" 2>&1)" || true
  if echo "$struct_raw" | jq -e . >/dev/null 2>&1; then
    policy_findings=$(echo "$struct_raw" | jq -c '.findings')
  fi
fi

# --- PL001: license check (inline — avoids double-running skill-validator) ---
frontmatter="$(awk '
  BEGIN { in_fm = 0 }
  /^---$/ {
    if (in_fm) { exit }
    in_fm = 1; next
  }
  in_fm { print }
' "$skill_md")"

if ! echo "$frontmatter" | grep -qE "^license:[[:space:]]"; then
  policy_findings=$(echo "$policy_findings" | jq -c \
    '. + [{"level": "fail", "rule": "PL001", "message": "missing license (house policy)"}]')
fi

# --- Merge into unified report ---
jq -n \
  --arg skill "$skill_dir" \
  --arg timestamp "$timestamp" \
  --argjson spec "$spec_json" \
  --argjson quality "$quality_json" \
  --argjson policy_findings "$policy_findings" \
  --arg spec_error "$spec_error" \
  --arg quality_error "$quality_error" \
  '{
    skill: $skill,
    timestamp: $timestamp,
    summary: {
      passed: (
        ($spec | if . == null then false else .passed end)
        and ([($policy_findings[] | select(.level == "fail"))] | length == 0)
      ),
      spec_passed: ($spec | if . == null then false else .passed end),
      spec_errors: ($spec | if . == null then null else .errors end),
      spec_warnings: ($spec | if . == null then null else .warnings end),
      quality_score: ($quality | if . == null then null else .overallScore.percentage end),
      quality_grade: ($quality | if . == null then null else .overallScore.letterGrade end),
      policy_failures: [($policy_findings[] | select(.level == "fail" and (.rule | startswith("PL"))))] | length,
      path_failures: [($policy_findings[] | select(.level == "fail" and (.rule | startswith("PT"))))] | length,
      total_findings: ($policy_findings | length)
    },
    spec: $spec,
    spec_error: (if $spec_error == "" then null else $spec_error end),
    quality: $quality,
    quality_error: (if $quality_error == "" then null else $quality_error end),
    policy: {
      findings: $policy_findings
    }
  }'
