#!/usr/bin/env bash
# Check frontmatter: spec validation via skill-validator, plus license policy gate.
# skill-validator handles: frontmatter format, name, description, YAML validity.
# We add: license requirement (house policy, not spec).
# Exit codes: 0=pass, 1=spec failure, 2=policy failure, 3=execution error.
# The policy failure carries rule ID PL001, a missing license.
# An execution error is one of three: a required tool is absent, which the
# rule registry calls DEP001; a source gave this script no answer it could use,
# DEP002 — skill-validator returning a status or a payload it cannot interpret,
# the two disagreeing with each other, a tool that is present and does not work,
# or a SKILL.md it could not read the frontmatter out of; or a verdict-guard.sh
# that would not load, reported before either ID exists to name it.
# Those IDs classify the exits; they are not printed. This script has no --json
# mode, so an exit 3 carries the guard's message on stderr — "required tool not
# found: skill-validator" — and no rule ID reaches the caller at all. The IDs
# above say which class an exit belongs to, for a reader of the registry; a
# consumer that needs to read one out of a payload wants --json, which only the
# two scripts that have it can give.
# Spec validation requires skill-validator, and reading what skill-validator
# answers requires jq. That is a wider requirement than this script used to
# state, and it is deliberate: it asks for `-o json` and now reads the payload
# it asked for rather than inferring it from an exit status, and the predicate
# that proves a payload readable is the guard's, which answers with jq. jq is
# already a prerequisite of this skill — audit-report.sh requires it, both
# --json modes require it, and SKILL.md's own pipeline pipes into it.
set -euo pipefail

script_dir="$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
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
  echo "Usage: check-frontmatter.sh <skill-dir>" >&2
  exit 3
fi

skill_dir="$1"
skill_md="$skill_dir/SKILL.md"

if [[ ! -f "$skill_md" ]]; then
  echo "ERROR: SKILL.md not found in $skill_dir" >&2
  exit 3
fi

# Run skill-validator for spec validation (frontmatter, name, description).
#
# Every tool this script computes with, stated as a precondition here: awk reads
# the frontmatter, grep answers the license rule, sed formats a spec failure,
# and skill-validator is the spec source itself. Each is required to be present
# *and* to answer a question with a known answer, because a tool that is on PATH
# and does not work is the fault that reached a consumer.
require_tool awk false
require_tool grep false
require_tool sed false
require_tool jq false
require_tool skill-validator false

# The spec source's stdout is captured on its own. It used to arrive merged with
# stderr, which made the payload this script asks for unparseable the moment the
# source said anything at all on the other channel — and unparseable was never
# going to be noticed, because nothing parsed it.
code=0
output="$(skill-validator validate structure -o json "$skill_dir")" || code=$?

# skill-validator exit codes: 0=clean pass, 1=errors, 2=warnings only, 3=usage
# error. Anything outside that set is a status we cannot interpret, so it must
# not reach the license gate and be reported as a verdict.
case $code in
  0|1|2) ;;
  3)
    cannot_compute DEP002 "skill-validator failed to run" false "$output"
    ;;
  *)
    cannot_compute DEP002 "skill-validator exited with unexpected status $code" false "$output"
    ;;
esac

# This script asks for `-o json` and then derived its whole spec verdict from
# `$?`, reading the payload only to quote it back in a failure message. On a
# healthy toolchain the status is a faithful proxy for the payload, and that is
# exactly what made the gap invisible: a skill-validator that exits 0 while
# printing something that is not JSON — a build with a broken formatter, a
# wrapper on PATH under the same name — satisfied every check here and reported
# `frontmatter OK`. The predicate this needs is not a new one. The guard already
# owns json_document_conforms; the spec source was simply never put through it.
#
# So the same three questions check-structure.sh asks of its child are asked
# here of the spec source, in the same order and for the same reasons: is the
# payload the shape it claims, and does it agree with the status it arrived
# with. A disagreement is not a verdict to pick a side of — the payload and the
# status are the source's two statements about one run, and a source
# contradicting itself has told us nothing.
#
# The claim reaches `.errors` and no further, because that is the whole of what
# is read out of it below.
json_document_conforms "$output" '
  if type != "object" then false
  else (.errors | type) == "number"
  end' \
  || cannot_compute DEP002 "skill-validator did not produce a readable spec payload" false "$output"

spec_errors="$(jq -r '.errors' <<< "$output")"

# 0 = clean pass and 2 = warnings only (e.g. orphan files) both assert no spec
# error; 1 asserts at least one. Either statement failing to match the other is
# DEP002.
case $code in
  0|2)
    [[ "$spec_errors" -eq 0 ]] \
      || cannot_compute DEP002 "skill-validator exited $code but its payload reports $spec_errors spec errors" false "$output"
    ;;
  1)
    [[ "$spec_errors" -gt 0 ]] \
      || cannot_compute DEP002 "skill-validator exited 1 but its payload reports no spec error" false "$output"
    sed 's/^/SPEC FAIL: /' <<< "$output"
    exit 1
    ;;
esac

# Check license as a policy gate (house rule, not spec).
#
# The frontmatter comes from the shared primitive. This copy was one of the two
# correct answers to "where does the frontmatter end", and both are gone: two
# correct private copies still leave a reader of these scripts four answers to
# one question, and the two that were wrong were wrong in ways nothing caught.
frontmatter="$(skill_frontmatter "$skill_md")" \
  || cannot_compute DEP002 "could not read the frontmatter of $skill_md" false

if ! text_matches false false "^license:[[:space:]]" "$frontmatter"; then
  echo "POLICY FAIL [PL001]: missing license (house policy)"
  exit 2
fi

echo "frontmatter OK"
