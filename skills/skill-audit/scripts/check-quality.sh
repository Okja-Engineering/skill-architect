#!/usr/bin/env bash
# Check quality: run skillscore for Anthropic-aligned quality scoring.
# Produces a 7-dimension quality report (identity, conciseness, clarity,
# routing, robustness, safety, portability) as JSON.
# Exit codes: 0=report produced, 3=execution error.
#
# This was the one script in the skill that sat outside the shared guard. It
# sourced nothing, so a verdict-guard.sh that was missing or malformed changed
# nothing about how it behaved while its four siblings all refused to answer; it
# never checked that the directory it was handed held a SKILL.md, so it reported
# skillscore's own exit 1 for "there is no skill here", a status its contract
# does not enumerate; and it passed skillscore's status straight through, so
# skillscore exiting 9 made this script exit 9, and skillscore exiting 0 having
# printed nothing made it exit 0 having printed nothing — a report generator
# reporting success over no report at all. It emitted no rule ID for any of it,
# which is also why no test reached it: the rule-ID census found no rule literal
# here, contributed the empty set for this file, and passed vacuously over it.
#
# An execution error is one of four: a required tool is absent, which the rule
# registry calls DEP001; a source gave this script no answer it could use,
# DEP002 — skillscore exiting outside its contract, or answering with something
# that is not the report this skill reads; a target with no SKILL.md in it,
# where there is no skill to score; or a verdict-guard.sh that would not load,
# reported before either ID exists to name it.
#
# Those IDs classify the exits; they are not printed, exactly as in
# check-frontmatter.sh. stdout here is the report channel and it carries one
# skillscore report or nothing at all, so there is no findings array for a rule
# ID to arrive in, and inventing a second stdout shape for the failure path
# would make every consumer distinguish two shapes where the exit status already
# says which happened. The IDs above say which class an exit belongs to, for a
# reader of the registry.
#
# Reading what skillscore answers requires jq, through the guard's
# quality_report_conforms — the same claim audit-report.sh proves about the same
# report, named once in the guard rather than kept privately in both.
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

if [[ $# -ne 1 ]]; then
  echo "Usage: check-quality.sh <skill-dir>" >&2
  exit 3
fi

skill_dir="$1"
skill_md="$skill_dir/SKILL.md"

# A directory with no SKILL.md in it is not a skill this script scored badly; it
# is a target it cannot score at all. Its four siblings all say so with exit 3,
# and it used to relay skillscore's exit 1 instead — a status its own contract
# does not enumerate, and one a caller reads as a computed verdict.
if [[ ! -f "$skill_md" ]]; then
  echo "ERROR: SKILL.md not found in $skill_dir" >&2
  exit 3
fi

require_tool jq false
require_tool skillscore false

# skillscore's stdout is captured on its own: it is the report channel, and
# merging its diagnostics into it would make the report unreadable exactly when
# there is something to say about it.
quality_status=0
quality_raw="$(skillscore "$skill_dir" --json)" || quality_status=$?

# skillscore exits 0 when it produced a report and nonzero when it did not.
# Anything nonzero is therefore a status this script cannot build a report on,
# and it is reported as one rather than passed through as if it were this
# script's own verdict.
[[ $quality_status -eq 0 ]] \
  || cannot_compute DEP002 "skillscore exited $quality_status and produced no report" false "$quality_raw"

# And a status is not a report. skillscore exiting 0 having printed nothing left
# this script exiting 0 having printed nothing, which a consumer reads as a
# skill with no quality to speak of rather than as a source it could not read.
quality_report_conforms "$quality_raw" \
  || cannot_compute DEP002 "skillscore did not produce a readable quality report" false "$quality_raw"

printf '%s\n' "$quality_raw"
