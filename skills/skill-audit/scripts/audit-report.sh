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
# exists to prevent. Nor is "it is an object": the spec source's claim stopped
# at the top level while three fields were read one level in, so an object
# carrying none of them was read as a source that had answered, and a `passed`
# that was the string "yes" became a verdict of true. The shape claimed differs
# per source, because what is read out of each of them differs.
# And every source's status is read against its payload, all three of them
# alike. A status and a payload are one run's two statements, so a source whose
# two statements contradict each other is a source that has told us nothing and
# there is no half of it to pick. The policy source did this from the start; the
# other two captured a status and never compared it, which published a spec
# verdict from a source that exited 1 saying it had failed nothing, and a
# quality score from a source whose nonzero status meant it produced no report.
# Rule IDs reach the report by relay, from the policy source, with two
# exceptions this file raises itself: PL001 for a missing license, which it
# checks inline and adds to the findings array, and DEP002 on the one source it
# cannot hold soft — the SKILL.md it reads that license out of. A source it
# could not read is named in the report; a *target* it could not read leaves no
# report to name anything in, so that one exits 3.
# What relays is check-structure.sh's whole findings array, so DEP001, DEP002,
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

# `dirname` runs before the guard exists to announce it, so it carries the same
# explicit refusal the load check below uses. Unread, its status was this
# script's own: a broken dirname left `cd` with nothing to enter and errexit
# exited 1, which is a status this contract spends on a verdict.
script_dir="$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)" \
  || { echo "ERROR: cannot resolve this script's own directory: dirname or cd gave no answer; no verdict was computed" >&2; exit 3; }
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

# Every tool this script computes with, stated as a precondition here: jq
# composes the report, awk reads the frontmatter, grep answers the license rule.
# Each is required to be present *and* to answer a question with a known answer
# — a jq on PATH that runs and prints nothing satisfies a presence check and
# then emits an empty report at exit 0, which is the fault this closes.
#
# And each is read again wherever it is called. This line settles what the tool
# did when it was asked here; it settles nothing about the five jq calls below
# it, and unread they were fatal: a jq that answered this question and failed
# the merge left this script exiting 5 with no report and no diagnostic, which
# is the exact outcome the header above says it exists to prevent.
require_tool jq false
require_tool awk false
require_tool grep false

# `date` is not on that list and its status is read here instead, because the
# question the list asks cannot be asked of a clock: a probe compares an answer
# against one this file already knows, and the whole point of asking the time is
# that this file does not know it. There is no portable constant to ask for
# either — `-r 0` on BSD and `-d @0` on GNU are different commands.
#
# What was here was a comment concluding that date needed no guard at all,
# because a clock that failed leaves a visibly empty timestamp beside a report
# whose findings are all still true. That is correct for the ways date can fail
# at exit 0, and it does not cover a nonzero exit, which is the case that
# existed: under errexit a date exiting 2 made this script exit 2 — outside the
# {0, 3} its header states — with no report and no diagnostic at all. A report
# generator that emits nothing must say why, and 3 is how this file says it.
timestamp_status=0
timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")" || timestamp_status=$?
[[ $timestamp_status -eq 0 && -n "$timestamp" ]] \
  || cannot_compute DEP002 "date could not answer what time it is (status $timestamp_status); no report was generated" false

# source_failure <source> <status> <output>
#
# The sentence this report puts in a *_error field: what this script asked, what
# the source answered with, and what it said — in that order, and in this
# script's own words.
#
# The fields used to carry the source's own output verbatim, and that is why a
# source which exited 0 and printed nothing left `spec_error: null` beside
# `spec_passed: false`: an empty answer and a working source produced the same
# empty string, jq rendered it as null, and the report asserted a spec failure
# while naming no reason for it. "The source said nothing" is a fact about the
# source and has to be written down as one. The status is named for the same
# reason — it used to be dropped with `|| true`, so a source that died was
# indistinguishable from one that answered badly.
source_failure() {
  local source="$1"
  local status="$2"
  local said="$3"
  if [[ -z "$said" ]]; then
    printf '%s did not produce a readable report: it exited %s and said nothing at all on its output channel\n' \
      "$source" "$status"
  else
    printf '%s did not produce a readable report: it exited %s and its output channel carried: %s\n' \
      "$source" "$status" "$said"
  fi
}

# --- Source 1: skill-validator (spec + structure + content + contamination) ---
# Read as `.passed`, `.errors` and `.warnings` of one document, so those three
# fields are what has to hold before any of it is read — not merely that the
# document is an object.
#
# The claim used to stop at `type == "object"` while the read went a level
# further in, which is the defect this file's header warns about, one level
# down. An object carrying none of the three passed it, and the merge then read
# `.passed` out of it and got null: `{"foo": 1}` produced a failing verdict with
# `spec_error: null` — the G3-04 defect arriving again through a source that
# parsed. `{"passed": "yes"}` was worse, because jq reads a non-empty string as
# truthy, so `summary.passed` came out true: a verdict read off a string, and
# the final conformance check below still held, since `summary.passed` was
# indeed a boolean.
#
# `.errors` is required to be a number rather than allowed to be absent because
# the status agreement below compares it, and a comparison against null decides
# nothing. `.warnings` is only reported, never compared, so it may be absent and
# null is what the read yields — the claim reaches as far as the read and no
# further, per source. check-frontmatter.sh reads the same tool and claims only
# `.errors`, because `.errors` is the whole of what it reads; the two claims
# differ because the reads differ, and that is the rule rather than a drift.
#
# Its stdout is captured on its own. Merged with stderr, the payload this script
# asks for became unreadable the moment the source said anything at all on the
# other channel, and the report then named a source that had answered correctly.
# Diagnostics belong on our stderr, where they pass straight through.
spec_json="null"
spec_error=""
if command -v skill-validator &>/dev/null; then
  spec_status=0
  spec_raw="$(skill-validator check -o json "$skill_dir")" || spec_status=$?
  if ! json_document_conforms "$spec_raw" '
        if type != "object" then false
        elif (.passed | type) != "boolean" then false
        elif (.errors | type) != "number" then false
        else ((.warnings | type) == "number" or (.warnings | type) == "null")
        end'; then
    spec_error="$(source_failure skill-validator "$spec_status" "$spec_raw")"
  else
    # And the status against the payload. They are the source's two statements
    # about one run, and a source contradicting itself has told us nothing —
    # so there is no half to pick, exactly as for the policy source below.
    # Captured all along and never read: a conforming payload at exit 1 was
    # published as `passed: true` with the contradiction unmentioned.
    #
    # skill-validator's documented statuses are the set check-frontmatter.sh
    # enumerates over the same tool: 0 clean, 1 errors, 2 warnings only, 3
    # usage error. 0 and 2 assert no spec error, 1 asserts at least one, and 3
    # or anything outside the set is not a verdict at all.
    jq_answer false "read the spec error count out of skill-validator's payload" \
      "$spec_raw" -r '.errors'
    spec_errors_said="$answered"
    case $spec_status in
      0|2)
        [[ "$spec_errors_said" -eq 0 ]] \
          || spec_error="skill-validator exited $spec_status but its payload reports $spec_errors_said spec errors; the two contradict each other" ;;
      1)
        [[ "$spec_errors_said" -gt 0 ]] \
          || spec_error="skill-validator exited 1 but its payload reports no spec error; the two contradict each other" ;;
      *)
        spec_error="$(source_failure skill-validator "$spec_status" "$spec_raw")" ;;
    esac
    [[ -n "$spec_error" ]] || spec_json="$spec_raw"
  fi
else
  spec_error="skill-validator not found. Install with: brew install agent-ecosystem/tap/skill-validator"
fi

# --- Source 2: skillscore (7-dimension quality scoring) ---
# The claim is the guard's quality_report_conforms, which is also what
# check-quality.sh proves before it emits the same report. Two private copies of
# one shape is one shape proven two ways, and the first of them to drift is the
# one nobody is reading when it does.
# Its status is read against its payload for the same reason, and the contract
# is the one check-quality.sh states over the same tool: skillscore exits 0 when
# it produced a report and nonzero when it did not. So a nonzero status is a
# source with no report to read, whatever arrived on its stdout — and this was
# the sharper of the two unread statuses, because check-quality.sh already
# refused what this file published. A conforming report at exit 7 gave
# `quality_score: 99` here and exit 3 there: one source, two readers, two
# answers, which is the drift the guard's shared predicates exist to prevent,
# arriving through the status rather than through the shape.
quality_json="null"
quality_error=""
if command -v skillscore &>/dev/null; then
  quality_status=0
  quality_raw="$(skillscore "$skill_dir" --json)" || quality_status=$?
  if ! quality_report_conforms "$quality_raw" false; then
    quality_error="$(source_failure skillscore "$quality_status" "$quality_raw")"
  elif [[ $quality_status -ne 0 ]]; then
    quality_error="$(source_failure skillscore "$quality_status" "$quality_raw")"
  else
    quality_json="$quality_raw"
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
  struct_status=0
  struct_raw="$("$script_dir/check-structure.sh" --json "$skill_dir")" || struct_status=$?
  if ! payload_is_conforming "$struct_raw" false; then
    policy_error="$(source_failure check-structure.sh "$struct_status" "$struct_raw")"
  else
    # check-structure.sh exits 0 pass, 1 path failure, 2 policy failure, 3 no
    # verdict. A status outside that set means it did not reach a verdict
    # whatever its payload looked like, and a status inside it that disagrees
    # with the payload's own verdict means the two halves of one answer
    # contradict each other. Either way there is nothing here to report as a
    # policy verdict, and the report says so rather than picking a half.
    jq_answer false "read check-structure.sh's findings out of its payload" \
      "$struct_raw" -c '.findings'
    policy_findings="$answered"
    jq_answer false "read check-structure.sh's own verdict out of its payload" \
      "$struct_raw" -r '.passed'
    policy_passed="$answered"
    case $struct_status in
      0) [[ "$policy_passed" == "true" ]] \
           || policy_error="check-structure.sh exited 0 beside a payload that reports it did not pass" ;;
      1|2|3) [[ "$policy_passed" == "false" ]] \
           || policy_error="check-structure.sh exited $struct_status beside a payload that reports it passed" ;;
      *) policy_error="check-structure.sh exited with unexpected status $struct_status; it reached no verdict this report can read" ;;
    esac
    if [[ -n "$policy_error" ]]; then
      policy_findings="[]"
      policy_passed=false
    fi
  fi
else
  policy_error="check-structure.sh is not present or not executable at $script_dir"
fi

# --- PL001: license check (inline — avoids double-running skill-validator) ---
# The frontmatter comes from the shared primitive, which is where the question
# "where does the frontmatter end" is now asked, once, for all of these scripts.
skill_frontmatter "$skill_md" false
frontmatter="$section"

if ! text_matches false false "^license:[[:space:]]" "$frontmatter"; then
  jq_answer false "add the PL001 finding to the policy findings" "$policy_findings" \
    -c '. + [{"level": "fail", "rule": "PL001", "message": "missing license (house policy)"}]'
  policy_findings="$answered"
fi

# --- Merge into unified report ---
#
# "0 = report generated" is this script's whole exit contract, so it must not
# reach 0 without one. It did: stdout here is the report channel, and it carried
# nothing at exit 0 whenever the tool that composes the report answered with
# nothing — a caller reading `.summary.passed` out of that gets null, which is
# neither a pass nor a failure and is indistinguishable from a skill it never
# asked about. So the document is proven to be the report before it is printed,
# as far down as a consumer reads the summary, and a report that would not
# compose is the same DEP002 exit 3 as a source that would not read.
jq_answer false "compose the report" "" \
  -n \
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
report="$answered"

json_document_conforms "$report" '
  if type != "object" then false
  elif (.summary | type) != "object" then false
  elif (.summary.passed | type) != "boolean" then false
  elif (.policy | type) != "object" then false
  else (.policy.findings | type) == "array"
  end' \
  || cannot_compute DEP002 "the report could not be composed; no report was generated" false

printf '%s\n' "$report"
