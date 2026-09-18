#!/usr/bin/env bash
# Produce a unified machine-readable audit report for a skill directory.
# Composes three JSON sources into one document:
#   1. skill-validator check -o json  (spec, structure, content, contamination)
#   2. skillscore --json              (7-dimension quality scoring)
#   3. House-policy checks            (PL001-PL005, PT001-PT002)
# Composes them with jq and requires it: without jq there is no report to
# generate, so it says on stderr which tool is missing and exits 3 rather than
# dying part-way through with the shell's own "command not found". That exit
# carries nothing on stdout and no rule ID: stdout here is the report channel,
# and the thing that builds a report is the thing that is missing.
# Every source stays soft: the report still generates and names in-band, as
# spec_error, quality_error or policy_error, the source it could not read. The
# two sources that carry a verdict are never read as sources with nothing to
# say — an unread spec or policy source leaves summary.passed false. Quality is
# a score rather than a verdict, so quality_error leaves the score null and the
# verdict alone.
# Every source is proven to carry the shape it is about to be read as — one
# document, of the type being indexed, as far down as the read goes — before any
# of it is read, so "could not read it" covers every way a source can be
# misshapen rather than the one way this file happened to check for. "It parsed"
# is not that proof: a number parses, and then indexing it raises inside the
# merge below and there is no report at all, which is the one outcome this file
# exists to prevent. The shape claimed differs per source, because what is read
# out of each of them differs.
# Rule IDs reach the report only by relay, from the policy source, except PL001
# for a missing license, which this file checks inline and adds itself. What
# relays is check-structure.sh's whole findings array, so DEP001, DEP002,
# PL002, PL003, PL004, PL005, PT001, PT002 and PATH all reach a consumer
# through here without this file naming any of them in a finding it built —
# which is exactly why they are enumerated rather than left implied.
# DEP001 is in that list and nowhere else in this file. Its own missing-jq exit
# above builds no finding, so nothing a reader of this report sees ever carries
# DEP001 unless check-structure.sh put it there; and since the only tool either
# script requires is the same jq, the exit above gets there first. Enumerated
# because the array can carry it, not because this file has been seen to
# deliver it.
# Exit codes: 0=report generated, 3=execution error.
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
verdict_guard="$script_dir/verdict-guard.sh"
# The guard is the one dependency it cannot announce itself, so loading it is
# checked before and after — see its header for why an unchecked source would
# exit with a status that means a verdict was computed.
bash -n "$verdict_guard" 2>/dev/null \
  || { echo "ERROR: cannot load $verdict_guard: missing or malformed; no verdict was computed" >&2; exit 3; }
# shellcheck source=verdict-guard.sh
source "$verdict_guard"
{ declare -F verdict_guard_ready >/dev/null && verdict_guard_ready; } \
  || { echo "ERROR: $verdict_guard did not load its guards; no verdict was computed" >&2; exit 3; }

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

require_tool jq false

timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

# --- Source 1: skill-validator (spec + structure + content + contamination) ---
# Read as `.passed`, `.errors` and `.warnings` of one document, so one document
# that is an object is the whole of what has to hold before any of it is read.
spec_json="null"
spec_error=""
if command -v skill-validator &>/dev/null; then
  spec_raw="$(skill-validator check -o json "$skill_dir" 2>&1)" || true
  if json_document_conforms "$spec_raw" 'type == "object"'; then
    spec_json="$spec_raw"
  else
    spec_error="$spec_raw"
  fi
else
  spec_error="skill-validator not found. Install with: brew install agent-ecosystem/tap/skill-validator"
fi

# --- Source 2: skillscore (7-dimension quality scoring) ---
# Read two levels in, at `.overallScore.percentage` and `.overallScore.
# letterGrade`, so the claim reaches two levels in: proving the top level and
# then indexing `.overallScore` would be the same defect one level down. It
# reaches no further than the read does — a source carrying no `overallScore` at
# all is read, and leaves the score null, because null is what the read yields.
quality_json="null"
quality_error=""
if command -v skillscore &>/dev/null; then
  quality_raw="$(skillscore "$skill_dir" --json 2>&1)" || true
  if json_document_conforms "$quality_raw" '
       if type != "object" then false
       elif (.overallScore | type) == "null" then true
       else (.overallScore | type) == "object"
       end'; then
    quality_json="$quality_raw"
  else
    quality_error="$quality_raw"
  fi
else
  quality_error="skillscore not found. Install with: npm install -g skillscore"
fi

# --- Source 3: House-policy checks (PL002-PL005, PT001-PT002) via check-structure.sh ---
# Unlike the two sources above, this one has a stdout contract: it carries a
# payload and nothing else, with every diagnostic on stderr. So its stdout is
# captured alone — merging the two channels turned the passed:false payload it
# emits when it cannot reach a verdict into unparseable text — and its
# diagnostics pass through to our stderr, where they belong.
#
# And an unreadable policy source is not a skill with no policy findings. It is
# a source this report could not read: it is named, and the report does not
# claim a pass over it.
#
# The payload is proven to be the documented shape before anything is read out
# of it, by the same predicate its producer's other consumer uses. Proving one
# thing about it — that .findings is an array — and then reading a great deal
# more is how the merge below came to select on .level and call startswith on
# .rule of elements nothing had checked, and die inside jq with no report at all.
# Everything after this line is a total read.
#
# The verdict is the source's own, not one re-derived from the findings it came
# with: a source saying it failed for a reason it did not enumerate as a
# level: "fail" finding is still a source saying it failed.
policy_findings="[]"
policy_passed=false
policy_error=""
if [[ -x "$script_dir/check-structure.sh" ]]; then
  struct_raw="$("$script_dir/check-structure.sh" --json "$skill_dir")" || true
  if payload_is_conforming "$struct_raw"; then
    policy_findings=$(echo "$struct_raw" | jq -c '.findings')
    policy_passed=$(echo "$struct_raw" | jq -r '.passed')
  else
    policy_error="check-structure.sh --json did not produce a readable payload"
  fi
else
  policy_error="check-structure.sh is not present or not executable at $script_dir"
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
  --argjson policy_passed "$policy_passed" \
  --arg spec_error "$spec_error" \
  --arg quality_error "$quality_error" \
  --arg policy_error "$policy_error" \
  '{
    skill: $skill,
    timestamp: $timestamp,
    summary: {
      passed: (
        ($spec | if . == null then false else .passed end)
        and $policy_error == ""
        and $policy_passed
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
    },
    policy_error: (if $policy_error == "" then null else $policy_error end)
  }'
