#!/usr/bin/env bash
# Draft a rewrite of a skill: run the skill-audit checks over it, and compose
# their output with the structural templates the audit says are missing.
# Writes REWRITE-DRAFT.md into the target skill directory and names it on
# stdout. The draft is a skeleton plus the audit output for a reader to work
# from; it does not rewrite the skill and does not touch its SKILL.md.
# Exit codes: 0=draft written, 1=usage or target error, 3=execution error.
#
# A usage or target error is a missing or unknown option, an option given
# without its value, a target directory or SKILL.md that is not there, or a
# report named with -a that is not there. Every one of them is reported on
# stderr, naming the option or the path, and leaves no draft behind.
#
# An execution error is one of three: a required tool is absent, which the rule
# registry calls DEP001; an audit source gave this script no answer it could use,
# DEP002; or a verdict-guard.sh that would not load, reported before either ID
# exists to name it. Those IDs classify the exits and are not printed — this
# script's stdout says where it wrote the draft and carries no findings array for
# an ID to arrive in.
#
# The audit's own verdict is not this script's exit status: a skill with
# failing house policy is the case the drafter exists for, so findings in the
# report are a successful run. A check that could not compute a verdict is a
# different thing, and it stops the draft.
#
# This script used to run both audit checks with `2>&1 || true`, which is two
# defects in one line. The status was discarded, so a check that reached no
# verdict at all was indistinguishable from one that reported a failing skill,
# and this script wrote a draft and exited 0 either way. And merging stderr into
# the capture put the checks' diagnostics into the draft's "Current state"
# section as if they were findings about the skill: with skill-validator absent,
# a clean skill's draft opened with "ERROR: skill-validator exited with
# unexpected status 127" presented as the current state of the skill.
#
# A draft is a document about an audit, so there has to have been an audit. A
# spec or policy failure is exactly the input this script wants and is accepted;
# a check that could not compute a verdict is not, and stops the draft.
set -euo pipefail

# `dirname` runs before the guard exists to announce it, so both calls carry the
# same explicit refusal the load check below uses. Unread, their status was this
# script's own: a broken dirname left `cd` with nothing to enter and errexit
# exited 1, which this contract spends on "usage or target error" — a broken
# tool reported as a mistake in the caller's command line.
script_dir="$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)" \
  || { echo "ERROR: cannot resolve this script's own directory: dirname or cd gave no answer; no draft was written" >&2; exit 3; }
skill_audit_root="$(dirname "$script_dir")/../skill-audit" \
  || { echo "ERROR: cannot resolve the skill-audit directory: dirname gave no answer; no draft was written" >&2; exit 3; }
verdict_guard="$skill_audit_root/scripts/verdict-guard.sh"
# The guard is the one dependency it cannot announce itself, so loading it is
# checked before and after — see its header for why an unchecked source would
# exit with a status that means a draft was written.
bash -n "$verdict_guard" 2>/dev/null \
  || { echo "ERROR: cannot load $verdict_guard: missing or malformed; no draft was written" >&2; exit 3; }
# shellcheck source=../../skill-audit/scripts/verdict-guard.sh
source "$verdict_guard"
{ declare -F verdict_guard_ready >/dev/null && verdict_guard_ready; } \
  || { echo "ERROR: $verdict_guard did not load its guards; no draft was written" >&2; exit 3; }

target_skill=""
audit_report=""
own_audit_report=""

# The temporary audit this script writes when it was not handed one. It is
# removed on every exit path rather than only the successful one: the checks
# below can now stop the draft, and a new exit path that leaked a file each time
# would be this change's own doing.
cleanup() {
  [[ -n "$own_audit_report" ]] && rm -f "$own_audit_report"
  return 0
}
trap cleanup EXIT

# Written with the shell's own `printf` rather than a `cat` heredoc, and that is
# the fix rather than a style preference. This function runs on the
# argument-parsing paths — `-h`, no argument, an unknown option — which are all
# before `require_tool cat` further down, and there is nowhere earlier to put
# that precondition: the arguments have to be read before this script knows
# whether it was asked to do any work at all. So a broken `cat` made `-h` exit 2
# where this contract says 0, and the other two exit 2 where it says 1, printing
# nothing on either channel.
#
# Reordering the precondition would not have fixed it and would have cost
# something: `-h` would then refuse to print help because a tool the help text
# does not need was broken. Telling the caller how to invoke this script is not
# a computation, so it is done with no tool to require and no status to read.
# `cat` stays a stated precondition below, where the draft is composed, which is
# the work that genuinely needs it.
usage() {
  printf '%s\n' \
    'Usage: draft-rewrite.sh -t <target-skill-dir> [-a <audit-report-path>]' \
    '' \
    'Options:' \
    '  -t, --target    Target skill directory to rewrite (required)' \
    '  -a, --audit     Path to an existing skill-audit report (optional)' \
    '  -h, --help      Show this help'
}

# require_value <option> <count> — refuse an option written without its value.
#
# It yields nothing, which is why it is not called `value_of`: each caller reads
# its own "$2", and this decides only whether there is one to read. Reading
# "$2" with nothing there is what made a mistyped option report `$2: unbound
# variable`: a line of bash internals naming a position in the parser, for a
# caller who needs to be told which option they left empty. The refusal lives
# here once so that a third option cannot be added without it. The count is
# passed in because `$#` inside a function is the function's own.
require_value() {
  if [[ "$2" -lt 2 ]]; then
    echo "Missing value for $1" >&2
    usage >&2
    exit 1
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -t|--target)
      require_value "$1" "$#"; target_skill="$2"; shift 2 ;;
    -a|--audit)
      require_value "$1" "$#"; audit_report="$2"; shift 2 ;;
    -h|--help)
      usage; exit 0 ;;
    *)
      echo "Unknown option: $1" >&2; usage >&2; exit 1 ;;
  esac
done

if [[ -z "$target_skill" ]]; then
  usage >&2
  exit 1
fi

if [[ ! -d "$target_skill" ]]; then
  echo "Target skill directory not found: $target_skill" >&2
  exit 1
fi

if [[ ! -f "$target_skill/SKILL.md" ]]; then
  echo "SKILL.md not found in $target_skill" >&2
  exit 1
fi

# A report named with -a is the caller saying what the draft is to be built
# from, so it is validated here with the other inputs rather than tested again
# at the point of use. Folding the two questions into one condition there —
# "was a report given, and is it readable" — made a typo indistinguishable from
# no report at all: the drafter audited afresh, drafted over that instead, and
# reported that no report had been provided.
if [[ -n "$audit_report" && ! -f "$audit_report" ]]; then
  echo "Audit report not found: $audit_report" >&2
  echo "Omit -a to have the structural checks run instead." >&2
  exit 1
fi

# Every tool this script computes with: awk reads the target's body, grep
# answers which sections it is missing, cat writes the draft, and mktemp holds
# the audit when it was not handed one. basename used to be on this list and is
# not, because it is no longer used: the skill's name is the last path component
# and parameter expansion is the same operation with nothing to require.
require_tool awk false
require_tool grep false
require_tool cat false
require_tool mktemp false

skill_name="${target_skill%/}"
skill_name="${skill_name##*/}"
output="$target_skill/REWRITE-DRAFT.md"

# skill-audit is skill-rewrite's sibling, so it is resolved by going up from
# this script to the skill directory that holds it and then across. That
# happens at the top of this file now, beside the guard load check that cannot
# be written without it. Resolved from this script's own location and not from
# the caller's working directory, which is not part of the answer: `CDPATH=`
# and `--` are what keep it that way, for the reason verdict-guard.sh sets out
# at length.

# What the draft says it was built from.
#
# A machine-local mktemp path was the one thing it could not be. It names a
# file that is gone by the time anyone reads the draft — or worse, one that is
# still there, since nothing removed it — and it never meant anything outside
# the process that wrote it. So the two cases state what they actually are: a
# report the caller named is named, and checks the drafter ran itself are
# described as checks the drafter ran itself.
if [[ -z "$audit_report" ]]; then
  provenance="structural checks run by this script: skill-audit's check-frontmatter.sh and check-structure.sh"
  echo "No audit report provided; running structural checks..." >&2
  # An explicit template, under the caller's temp directory and named after the
  # tool that made it. `mktemp` with no template ignores TMPDIR on BSD and
  # honours it on GNU, so a bare call puts this file somewhere the caller did
  # not choose on one of the two platforms this skill supports — and leaves a
  # leak nobody can look for, because its location differs by platform and its
  # name says nothing about where it came from.
  #
  # Its status is read, like every other external this script computes with.
  # `require_tool mktemp` above proves mktemp can make a file under TMPDIR, and
  # that is a different statement from this call having made one: the check
  # happens once, and the directory can stop being writable between the check
  # and the call. Unread, mktemp's own exit 1 became this script's, and 1 is
  # "usage or target error" here — so a temp directory that does not exist was
  # reported as a mistake in the caller's command line, with nothing naming
  # mktemp and no "no draft was written".
  #
  # The answer is checked as well as the status. A mktemp exiting 0 having
  # printed nothing leaves this the empty string, and every use of it below is a
  # redirection: the draft would be composed over a file named "".
  own_audit_status=0
  own_audit_report="$(mktemp "${TMPDIR:-/tmp}"/draft-rewrite-audit.XXXXXX)" \
    || own_audit_status=$?
  [[ $own_audit_status -eq 0 && -n "$own_audit_report" ]] \
    || cannot_compute DEP002 "mktemp could not make a temporary audit under ${TMPDIR:-/tmp} (status $own_audit_status); no draft was written" false
  audit_report="$own_audit_report"
  # Each check's status is read, and each check's stdout is captured alone.
  # check-frontmatter.sh and check-structure.sh both exit 0 pass, 1 a spec or
  # path failure, 2 a policy failure, 3 no verdict reached. The first three are
  # findings about the skill, which is what a rewrite draft is written from. The
  # fourth is not a finding about anything: it means the check could not answer,
  # and a draft composed over it would present the check's own failure as the
  # state of the skill.
  #
  # The two checks are named once, in full, at this one declaration site rather
  # than spelled as a bare basename joined to the root inside the loop. A
  # reader asking which of the sibling's checks this script runs gets the
  # answer by reading it, and so does tests/test_rewrite.sh, which decides that
  # question between this file and the SKILL.md and can only do so if the
  # names are here to be read.
  for audit_check in \
    "$skill_audit_root/scripts/check-frontmatter.sh" \
    "$skill_audit_root/scripts/check-structure.sh"; do
    audit_status=0
    "$audit_check" "$target_skill" >> "$audit_report" || audit_status=$?
    case $audit_status in
      0|1|2) ;;
      *) cannot_compute DEP002 "${audit_check##*/} reached no verdict over $target_skill (status $audit_status); no draft was written" false ;;
    esac
  done
else
  provenance="audit report $audit_report"
fi

# Which sections the target is missing is a question about its body, and the
# body is read through the shared primitive for the reason check-structure.sh
# now does: `## When to use` written at column 0 inside the YAML frontmatter is
# a comment to a parser and a heading to a grep over the whole file, so a skill
# whose body has no sections at all was handed a draft with no templates in it.
#
# Read before the draft file is opened. Everything that can stop this script
# has to stop it before the first byte is written, or "no draft was written"
# stops being true and the caller is left with half a document.
target_body="$(skill_body "$target_skill/SKILL.md")" \
  || cannot_compute DEP002 "could not read the body of $target_skill/SKILL.md; no draft was written" false

cat > "$output" <<EOF
# Rewrite draft: $skill_name

Generated from: $provenance

## Current state

EOF

cat "$audit_report" >> "$output"

cat >> "$output" <<'EOF'

## Proposed structure

Follow the Agent Skills spec and ICM context-management principles:

- Frontmatter stays small and includes `name`, `description`, `license`, and optional `compatibility`/`metadata`.
- `SKILL.md` body is under ~500 lines.
- Required sections:
  - `## When to use`
  - `## Deterministic actions (60%)`
  - `## Orchestration (30%)`
  - `## AI judgment (10%)`
  - `## Examples`
  - `## Constraints`
- Mechanical work lives in `scripts/`; deep reference material lives in `references/`.

## Missing section templates

EOF

if ! text_matches false true "^#{2,6}[[:space:]]+When to use[[:space:]]*$" "$target_body"; then
  cat >> "$output" <<'EOF'
### When to use

Use this skill when:

- <specific scenario 1>
- <specific scenario 2>
- <specific scenario 3>

Do not use this skill when <negative scope>.

EOF
fi

if ! text_matches false true "^#{2,6}[[:space:]]+Examples?[[:space:]]*$" "$target_body"; then
  cat >> "$output" <<'EOF'
### Examples

#### Example 1: <scenario>

Input:

```text
<concrete user request>
```

Command:

```bash
<exact command the skill runs>
```

Expected output:

```text
<concrete output>
```

EOF
fi

if ! text_matches false true "^#{2,6}[[:space:]]+Validation" "$target_body"; then
  cat >> "$output" <<'EOF'
### Validation checklist

- [ ] <observable pass/fail criterion 1>
- [ ] <observable pass/fail criterion 2>
- [ ] <observable pass/fail criterion 3>

EOF
fi

cat >> "$output" <<'EOF'

## Action items

- [ ] Review the proposed structure against the skill's original intent.
- [ ] Fill in the missing section templates above with domain-specific content.
- [ ] Move mechanical steps to `scripts/` if they are currently described in prose.
- [ ] Move deep reference content to `references/` or `assets/`.
- [ ] Re-run the audit after applying changes.

## Notes

- Default ratio for a spec-compliant skill: Deterministic 60%, Orchestration 30%, AI judgment 10%.
- Preserve the skill's original domain knowledge; only restructure how it is disclosed.
- If the skill is doing more than one job, consider splitting it instead of expanding the rewrite.
EOF

echo "Rewrite draft written to: $output"
