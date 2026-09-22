#!/usr/bin/env bash
# The suite for the substrate the other suites are written on.
#
# This cluster exists because the verification substrate could not report a
# failure. Two shapes of that were live at once: 23 assertions across two suites
# whose verdict was the literal `true`, so no repository state could make them
# print FAIL; and three of four suites whose only EXIT handler was `rm -rf`, so
# a suite that died before its summary said nothing at all. Both had been
# "fixed" once, in one file, by hand — which is precisely why they were still
# open everywhere else.
#
# So the invariant is asserted here, mechanically, over every suite there is
# rather than over the ones a fixer remembered:
#
#   an assertion's verdict is a function of repository state, and a suite that
#   does not reach its summary says so.
#
# The suite list is a glob, so a suite added tomorrow is covered the day it
# lands — and every count this file compares against is read out of the CI
# workflow or the README rather than written down here, so a suite added
# tomorrow cannot leave a stale number behind that quietly stops guarding.
# Both halves are read out of the source text by
# tests/lib/audit-suites.sh: a verdict that cannot depend on the repository, and
# a private copy of the harness — including a private EXIT trap — that a repair
# to the shared one would never reach. Neither is answerable at runtime, and
# neither is answerable with a grep, because these suites embed heredocs that
# deliberately contain both.
#
# Every check that can only ever say "no violation found" carries a control that
# makes it say the opposite, because a check that cannot fail is the exact
# defect this file is about.
set -euo pipefail
. "$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)/lib/harness.sh"
harness_init

AUDIT=tests/lib/audit-suites.sh
HARNESS=tests/lib/harness.sh

# audit_accepts <file>... — the audit finds nothing to refuse.
audit_accepts() {
  quietly "$AUDIT" --harness "$HARNESS" "$@"
}

# audit_rejects <file>... — the audit finds one, and says where.
#
# Written as a captured run rather than `! audit`, because the expected output
# of a firing control is noise on a passing run, and because a control that
# silently stopped firing has to be able to say so.
audit_rejects() {
  local out
  if out="$("$AUDIT" --harness "$HARNESS" "$@" 2>&1)"; then
    printf 'expected a rejection, got acceptance:\n%s\n' "$out" >&2
    return 1
  fi
  return 0
}

# lacks <ere> <file> — the file does not match the pattern.
lacks() {
  local pattern="$1"
  local file="$2"
  if grep -qE "$pattern" "$file"; then
    return 1
  fi
  return 0
}

# lacks_dir <path> — nothing is at the path.
lacks_dir() {
  if [ -d "$1" ]; then
    return 1
  fi
  return 0
}

# audit_field <name> <file> — one of the audit's per-file figures for <file>.
#
# Read from the audit's own `read=<path> …` line and not from its total, for two
# reasons. The audit now reads the **closure under `source`** — four of the
# seven suites get their code from tests/lib/, and a definition there is as live
# as one in the suite — so a total is a sum over more files than the one being
# asked about. And per file is the only shape in which "one file went
# unexamined" is visible at all, which is the case that actually happened.
#
# The audit's nonzero exit means "it found something", which is not a failure to
# count, so it is read for its tally either way.
audit_field() {
  local field="$1"
  local file="$2"
  local out
  out="$("$AUDIT" --harness "$HARNESS" "$file" 2>/dev/null)" || true
  printf '%s\n' "$out" | awk -v f="$file" -v k="$field" '
    $1 == "read=" f {
      for (i = 2; i <= NF; i++) { split($i, kv, "="); if (kv[1] == k) print kv[2] }
    }'
}

# sites_counted <file> — how many assertion call sites the audit examined in it.
sites_counted() {
  audit_field sites "$1"
}

# sites_total <file>... — the audit's tally over a set of files together.
sites_total() {
  local out
  out="$("$AUDIT" --harness "$HARNESS" "$@" 2>/dev/null)" || true
  printf '%s\n' "$out" | sed -n 's/^sites=\([0-9]*\).*/\1/p'
}

# lines_read <file> — how many input lines the audit's reader saw in it.
lines_read() {
  audit_field lines "$1"
}

# lines_accounted <file> — how many the walk actually disposed of. Every
# record leaves the audit's main rule through one of four paths and each one
# counts itself, so this is lines_read unless something skipped a record without
# saying so.
lines_accounted() {
  audit_field accounted "$1"
}

# audited_closure <file>... — every file the audit reads, following `source`.
audited_closure() {
  "$AUDIT" --harness "$HARNESS" --closure "$@" 2>/dev/null || true
}

# audit_read_whole_file <file> — the audit read every line there is.
#
# This is the derived denominator, and it replaces a floor. The site count used
# to be asserted `-ge 200` against a real 740, which is the shape this file
# already refused sixty lines below and then left standing here: a hand-written
# bound cannot say how much of the surface was examined, only that it was not
# none. Measured — injecting `FNR > 400 { next }` into the audit dropped the
# examined sites from 727 to 234 with test_install.sh audited at *zero*, and all
# seven suites stayed green at exactly their published totals on both shells.
# 68% of the audited surface gone and nothing fired. This audit is the only
# thing standing behind non-vacuity for the assert_value sites
# tests/lib/harness.sh says rest on it.
#
# The other side of the count is a second program counting the same thing over
# the same file, compared for equality. It is asked per file rather than in
# total, because a total can be made up: one suite going unexamined is the case
# that happened, and it is invisible in a sum. There is no number in this file
# for the next assertion to make wrong.
#
# `grep -ac ''` and not `wc -l`, for the reason
# tests/lib/forward-promise-check.sh's own record count states: a file whose
# last line has no newline holds one more *record* than `wc -l` reports, and
# awk — which is what the audit reads with — sees that record. Two files in
# this repository are already written that way, so the condition is reachable
# here and not hypothetical; it is only the closure that happens to exclude
# them today. Measured: appending `: ;` with no trailing newline to
# tests/lib/masked-path.sh made this check report "the audit read 90 lines of
# tests/lib/masked-path.sh, which has 89" and redden both shells — the audit
# being right and its own denominator being wrong. A guard that refuses a legal
# file is a guard somebody turns off, and this one is the whole replacement for
# the `-ge 200` floor.
#
# Two comparisons and not one, because there are two ends to introduce a limit
# at. `lines` against the second count catches a reader that stopped.
# `accounted` against `lines` catches a walk that stopped while the reader went
# on — which is what the measured regression was: a rule above the walk, with
# awk still reading every record and reporting so.
audit_read_whole_file() {
  local suite="$1"
  local read_lines
  local walked_lines
  local file_lines
  read_lines="$(lines_read "$suite")"
  walked_lines="$(lines_accounted "$suite")"
  file_lines="$(grep -ac '' "$suite" | tr -d '[:space:]')"
  if [ "${read_lines:-0}" != "$file_lines" ]; then
    printf 'the audit read %s lines of %s, which has %s: a check over a file it did not finish reading says nothing about the rest of it\n' \
      "${read_lines:-0}" "$suite" "$file_lines" >&2
    return 1
  fi
  if [ "${walked_lines:-0}" != "$read_lines" ]; then
    printf 'the audit read %s lines of %s and accounted for %s: %s lines went past the walk without being examined or deliberately skipped\n' \
      "$read_lines" "$suite" "${walked_lines:-0}" "$((read_lines - ${walked_lines:-0}))" >&2
    return 1
  fi
  return 0
}

# <audit script> <file> — the line check's verdict over another audit. A
# subshell, so the override cannot leak into the checks below.
read_check_over() {
  ( AUDIT="$1"; shift; audit_read_whole_file "$@" )
}

# <audit script> <file> — that audit does not read the whole file, and the check
# says so. Written as the inverse rather than as `! read_check_over`, because
# the expected diagnostic of a firing control is noise on a passing run.
read_check_rejects() {
  if read_check_over "$@" 2>/dev/null; then
    printf 'the line check accepted an audit that stops reading at line 400, so it cannot refuse one\n' >&2
    return 1
  fi
  return 0
}

fixtures="$harness_scratch/fixtures"
mkdir -p "$fixtures"
# A copy of a real suite resolves its harness from its own directory, by the
# idiom every suite opens with. So the fixture directory is given the same
# `lib` the real one has, and a copied suite loads the real harness rather
# than being refused for a `source` that cannot be resolved — which would be a
# refusal about the fixture and not about what the case is testing.
ln -sfn "$PWD/tests/lib" "$fixtures/lib"

# --- The suites, enumerated rather than named ---------------------------------

suites=""
suite_count=0
for suite in tests/test_*.sh; do
  suites="$suites $suite"
  suite_count=$((suite_count + 1))
done

# The two other places the same list is written down, read out of those places
# rather than restated here. Every count below is derived from one of them, so
# there is no number in this file for the next suite to make wrong.
WORKFLOW=.github/workflows/ci.yml

# The suites the workflow runs, as a list that is allowed to be empty.
#
# `grep` exits 1 when it matches nothing, and a workflow that names no suites is
# a state this file *reports* rather than one it dies on. That status used to
# come out of the command substitution that derives the count, and `set -e`
# killed the suite there, one line above the precondition written to refuse
# exactly that — so the invariant failed closed through the abort guard and the
# diagnostic naming the cause was lost. Finding nothing is an empty list here.
# Whether an empty list is acceptable is the precondition's to say, and it is
# the only thing that says it.
workflow_suites() {
  grep -oE 'tests/test_[A-Za-z0-9_]+\.sh' "$WORKFLOW" | sort -u || :
}

# The block the README tells a reader to run. A reader who follows a list that
# is missing a suite never runs it, which is the same defect as a suite CI never
# runs, one audience over.
readme_test_block() {
  awk '
    /^Run the tests:$/ { found = 1; next }
    found && /^```bash$/ { in_block = 1; next }
    in_block && /^```$/ { exit }
    in_block { print }
  ' README.md
}

listed_in_the_readme() {
  readme_test_block | grep -qF -- "$1"
}

workflow_suite_count="$(workflow_suites | wc -l | tr -d '[:space:]')"

# The denominator, and it is a precondition: an empty or shrunken glob makes
# every per-suite check below vacuously true, which is the failure this file
# exists to refuse, and a vacuous check that only reports is not refused.
#
# It is derived rather than written down, and that is the whole repair. The
# number used to be a floor — `-ge 6`, raised by hand each time a suite landed.
# A branch adding the seventh suite did not touch this file, because nothing
# made it: the floor still passed, one suite went unexamined, and removing a
# suite altogether would have passed too. So the count this file expects is now
# the number of suites the workflow runs, compared for *equality* rather than as
# a bound. Equality in both directions, because each suite in the glob is also
# required to be a step in the workflow below: the two lists cannot differ in
# either direction without this failing.
require "the CI workflow names suites, so these checks have a denominator" \
  test "$workflow_suite_count" -gt 0
require "tests/ holds exactly the suites the CI workflow runs" \
  test "$suite_count" -eq "$workflow_suite_count"
require "the README's list of tests to run is extractable" \
  test -n "$(readme_test_block)"
echo "  suites examined:$suites"
echo "  suites the workflow runs: $(workflow_suites | tr '\n' ' ')"

# --- Control: the denominator the precondition above refuses -----------------
#
# A workflow naming no suites is the state that precondition exists to refuse,
# and it can only refuse a count it is handed. The derivation has to survive
# producing zero for that to happen: the same question as anywhere else in this
# file, which is whether the machinery can report the thing it is watching for.
#
# Both halves are asserted, because "it did not die" and "it said zero" are
# different claims and only the pair of them gets the count as far as the
# precondition.
empty_workflow="$fixtures/names-no-suites.yml"
cat > "$empty_workflow" <<'EOF'
name: ci
jobs:
  test:
    steps:
      - run: echo this workflow runs no suites
EOF

# <workflow file> — the count the precondition above reads, over another
# workflow. A subshell, so the override cannot leak into the checks below.
derived_count_over() {
  ( WORKFLOW="$1"; workflow_suites | wc -l | tr -d '[:space:]' )
}

a_workflow_naming_no_suites_derives_a_denominator_of_zero() {
  local count
  count="$(derived_count_over "$empty_workflow")" || return 1
  [ "$count" = 0 ]
}

assert "a workflow naming no suites derives a denominator of zero to refuse" \
  a_workflow_naming_no_suites_derives_a_denominator_of_zero

for suite in $suites; do
  assert "$suite is on the shared harness" grep -q 'lib/harness\.sh' "$suite"
  assert "$suite reaches its verdict through harness_summary" \
    grep -q 'harness_summary' "$suite"
  assert "$suite has no assertion that cannot fail, and no harness of its own" \
    audit_accepts "$suite"
  # A suite CI never runs reports nothing either, whatever it would have said.
  # Everything above holds only over the suites that actually execute, so the
  # glob and the workflow are held to the same list.
  assert "$suite is a step in the CI workflow" \
    grep -qF -- "$suite" "$WORKFLOW"
  assert "$suite is in the README's list of the tests to run" \
    listed_in_the_readme "$suite"
  assert "the audit found an assertion call site in $suite at all" \
    test "$(sites_counted "$suite")" -gt 0
done

# --- The audited set is the closure under `source`, not the list of suites ----
#
# The list of files used to be whatever a caller passed, and the caller passed
# the glob. Four of these seven suites source a library out of tests/lib/, and a
# definition in a library is as live as one in the suite — measured: one line
# appended to tests/lib/masked-path.sh redefined `assert_value`, and
# tests/test_f01.sh reported its exact published 1886 passed, 0 failed over a
# tree whose skills/skill-audit/SKILL.md had been replaced with the word BROKEN,
# with the audit green at `sites=767 files=7`.
#
# So the audit computes the closure itself, following every `source` it can
# resolve and refusing every one it cannot, and this file reads that closure
# rather than restating it. There is no list of libraries here, and a suite that
# starts sourcing a new one covers it the day it lands.
#
# Held two ways, because a derivation with one side is a list with extra steps:
#
#   - by a **second reader** below — a line-oriented grep for a literal
#     tests/lib path on a `.`/`source` line, which over-approximates because it
#     sees heredocs too, so every one of its hits must be in the closure; and
#   - by **bash**, in tests/lib/harness.sh itself, where `harness_closure_covers`
#     holds the files bash says defined the live functions against this same
#     closure, at every suite's summary. That side cannot be fooled by any
#     spelling, and it is why a library loaded by a route no reader can follow
#     fails rather than passing.
audit_closure="$(audited_closure $suites)"
echo "  files the audit reads: $(printf '%s' "$audit_closure" | tr '\n' ' ')"

require "the audit names the files it read, so this file has a set to hold" \
  test -n "$audit_closure"

for suite in $suites; do
  assert "$suite is in the set of files the audit reads" \
    quietly grep -qxF -- "$suite" <<CLOSURE_HAS
$audit_closure
CLOSURE_HAS
done

# The second reader, and only one direction of it is asserted: every library it
# sees sourced has to be in the closure, because a library the audit did not
# read is the hole.
#
# Not the other direction, and the reason is measured rather than assumed. It
# was written here that `grep` "finds at least as much as the audit's tokenizer
# does", since it cannot tell a `source` in a heredoc from a real one — and that
# is false: it also finds *less*. It matches only a literal path on the
# directive, so `. "$mutation_runner"` at the bottom of this file is invisible
# to it while the tokenizer resolves the variable and reads the library. Today
# grep sees one library and the closure holds two. The claim this side can carry
# is the containment, not the equality, which is what is asserted.
sourced_by_grep() {
  grep -hoE '(^|[^a-zA-Z])(\.|source)[[:space:]]+"?tests/lib/[A-Za-z0-9_-]+\.sh' $suites \
    | grep -oE 'tests/lib/[A-Za-z0-9_-]+\.sh' | sort -u || :
}
echo "  libraries a second reader finds sourced: $(sourced_by_grep | tr '\n' ' ')"
require "the second reader finds a library sourced at all, so its side is not empty" \
  test -n "$(sourced_by_grep)"
while IFS= read -r grep_lib; do
  [ -n "$grep_lib" ] || continue
  assert "$grep_lib, which a second reader sees sourced, is a file the audit reads" \
    quietly grep -qxF -- "$grep_lib" <<CLOSURE_HAS_LIB
$audit_closure
CLOSURE_HAS_LIB
done <<GREP_LIBS
$(sourced_by_grep)
GREP_LIBS

# And the denominator of the audit itself, per file in the closure.
# `audit_accepts` above can only ever say "no violation found", and it says
# exactly that over a file it stopped reading and over one it never opened.
for read_file in $audit_closure; do
  assert "the audit read every line of $read_file, not a prefix of it" \
    audit_read_whole_file "$read_file"
done

# And the tally over the suites together is the tally over each of them, so the
# figure this file prints — the one the release notes quote — is the sum of the
# per-suite figures the checks above hold, rather than a number produced by a
# run nothing else saw.
examined="$(sites_total $suites)"
per_file_total=0
for read_file in $audit_closure; do
  per_file_total=$((per_file_total + $(sites_counted "$read_file")))
done
echo "  assertion call sites examined: $examined"
echo "  per-file: $(for read_file in $audit_closure; do printf '%s=%s ' "${read_file##*/}" "$(sites_counted "$read_file")"; done)"
assert "the audit's tally over the closure together is the sum of its tallies over each file in it" \
  test "${examined:-0}" -eq "$per_file_total"

# --- Control: the denominator above can say a file was not finished -----------
#
# The check that matters most here can only ever say "it read all of it", and a
# check that cannot say the opposite is the defect this whole file is about. So
# it is handed the regression that was used to prove the floor it replaced:
# `FNR > 400 { next }` inside the audit's own awk program, which is what a
# plausible limit or an off-by-one in a bound looks like from the outside.
#
# The audit is copied and the rule inserted rather than the real one edited, so
# nothing about this control can reach the checks above — and it is driven over
# a real suite rather than a fixture, because a four-line fixture is under 400
# lines and a control that cannot reach the limit it is testing is the shape
# being refused.
truncating_audit="$fixtures/audit-that-stops-reading.sh"
awk '/^FNR == 1 \{$/ { print "FNR > 400 { next }" } { print }' "$AUDIT" \
  > "$truncating_audit"
chmod +x "$truncating_audit"

assert "the injected control really is a different program from the audit" \
  quietly grep -qF 'FNR > 400' "$truncating_audit"
assert "an audit that stops reading at line 400 is refused by the line count" \
  read_check_rejects "$truncating_audit" tests/test_install.sh
# And the other direction, over the same copied program with the rule taken back
# out, so the refusal above is the injected rule's doing and not the copy's.
untruncated_audit="$fixtures/audit-that-reads-it-all.sh"
grep -v '^FNR > 400 { next }$' "$truncating_audit" > "$untruncated_audit"
chmod +x "$untruncated_audit"
assert "the same audit without the rule reads the whole file, so the copy is not what was refused" \
  read_check_over "$untruncated_audit" tests/test_install.sh

# --- Controls: the audit rejects each shape of a verdict that cannot fail -----

cat > "$fixtures/literal-true.sh" <<'EOF'
assert "the moon is made of green cheese" true
EOF
assert "the audit rejects a verdict that is the literal true" \
  audit_rejects "$fixtures/literal-true.sh"

cat > "$fixtures/literal-colon.sh" <<'EOF'
assert "every user is authenticated" :
EOF
assert "the audit rejects a verdict that is the null command" \
  audit_rejects "$fixtures/literal-colon.sh"

cat > "$fixtures/literal-echo.sh" <<'EOF'
assert "the database is encrypted at rest" echo checking...
EOF
assert "the audit rejects a verdict that is an echo" \
  audit_rejects "$fixtures/literal-echo.sh"

cat > "$fixtures/wrapped-true.sh" <<'EOF'
assert "CI is green" quietly true
EOF
assert "the audit looks through a transparent wrapper to the command underneath" \
  audit_rejects "$fixtures/wrapped-true.sh"

cat > "$fixtures/no-verdict.sh" <<'EOF'
assert "nothing was asserted at all"
EOF
assert "the audit rejects an assertion with no verdict to run" \
  audit_rejects "$fixtures/no-verdict.sh"

cat > "$fixtures/value-literal.sh" <<'EOF'
assert_value "the release is signed" true
EOF
assert "the audit rejects a value-shaped verdict written as a literal" \
  audit_rejects "$fixtures/value-literal.sh"

# A precondition is an assertion that also refuses, so it carries an assertion's
# vacuity question unchanged. A precondition that cannot fail is worse than a
# vacuous assertion: it is a guard that can never refuse.
cat > "$fixtures/require-literal.sh" <<'EOF'
require "the destination is inside the scratch root" true
EOF
assert "the audit rejects a precondition that cannot fail" \
  audit_rejects "$fixtures/require-literal.sh"

cat > "$fixtures/trailing-true.sh" <<'EOF'
mkdir -p tests && assert "a vacuous assertion after a separator" true
EOF
assert "the audit reads an assertion that is not the first word on its line" \
  audit_rejects "$fixtures/trailing-true.sh"

# --- Controls: the audit accepts a verdict the repository decides -------------

cat > "$fixtures/real-command.sh" <<'EOF'
assert "tests/ exists" test -d tests
assert "the harness is readable" quietly cat tests/lib/harness.sh
require "the repository is where the suite thinks it is" test -d skills
EOF
assert "the audit accepts a verdict that runs a command" \
  audit_accepts "$fixtures/real-command.sh"

cat > "$fixtures/real-value.sh" <<'EOF'
assert_value "tests/ exists" "$(test -d tests && echo true || echo false)"
assert_value "a verdict read from a variable" "$verdict"
EOF
assert "the audit accepts a value-shaped verdict with something to expand" \
  audit_accepts "$fixtures/real-value.sh"

cat > "$fixtures/heredoc-body.sh" <<'SH'
python3 - <<'PY'
assert 1 == 1, 'a heredoc body is not shell'
assert True
PY
assert "the shell assertion on this line is real" test -d tests
SH
assert "the audit reads shell and not heredoc bodies" \
  audit_accepts "$fixtures/heredoc-body.sh"

# --- Controls: the audit rejects a suite carrying its own harness ------------

cat > "$fixtures/own-assert.sh" <<'EOF'
assert() {
  echo "a private copy of the harness, free to drift from the shared one"
}
EOF
assert "the audit rejects a suite that defines its own assert" \
  audit_rejects "$fixtures/own-assert.sh"

cat > "$fixtures/own-require.sh" <<'EOF'
require() {
  echo "a private copy of the precondition, free to stop refusing"
}
EOF
assert "the audit rejects a suite that defines its own require" \
  audit_rejects "$fixtures/own-require.sh"

cat > "$fixtures/own-assert-spaced.sh" <<'EOF'
harness_summary ()
{
  echo "the same thing, spelled the other way"
}
EOF
assert "the audit rejects a redefinition spelled with a space" \
  audit_rejects "$fixtures/own-assert-spaced.sh"

cat > "$fixtures/own-assert-keyword.sh" <<'EOF'
function quietly {
  echo "and the third way bash spells it"
}
EOF
assert "the audit rejects a redefinition spelled with the function keyword" \
  audit_rejects "$fixtures/own-assert-keyword.sh"

cat > "$fixtures/own-trap.sh" <<'EOF'
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
EOF
assert "the audit rejects a suite that installs its own EXIT trap" \
  audit_rejects "$fixtures/own-trap.sh"

cat > "$fixtures/other-trap.sh" <<'EOF'
trap 'echo interrupted' INT
EOF
assert "the audit leaves a trap that is not the exit handler alone" \
  audit_accepts "$fixtures/other-trap.sh"

# --- bash is the other side, because a list of spellings is not a denominator -
#
# Every fixture above is a spelling somebody thought of, and that is exactly the
# defect: four of the audit's sides were hand-enumerated literals — the harness
# names, the definition spellings, the EXIT-trap spellings, and the set of files
# read — and not one of them was held against an independent derivation. Each of
# the four was a live fail-open. `assert ( ) {` defines the function on 3.2.57
# and on 5.3.15 and the reader accepted it; a suite shadowed that way reported
# 20 passed, 0 failed where the real harness reported 18 passed, 2 failed.
#
# So the sides are derived and held against the one authority that cannot have
# a short list: **bash**. Each sweep below generates a product and asks bash what
# it did, and the reader has to agree. No case names an expected verdict, and
# the refusal the audit must return is computed from the oracle rather than
# written beside the case — which is what keeps a wrong repair from passing
# against its own patch.
#
# What is still enumerated, stated plainly: the **operators**. A new spelling
# mechanism is a line somebody has to write. That is a much smaller and much
# more reviewable question than "did the author remember every spelling of every
# name", and it is not zero.

# The names the harness provides, from the two sides that must agree: the text
# reader that decides what a suite may not shadow, and bash's own record of what
# this file defined. Held for **equality**, with no number in it — the shape
# tests/test_f01.sh already holds verdict-guard.sh's primitives to, and the one
# that was not applied to the list the same audit reads.
harness_names_derived="$("$AUDIT" --names "$HARNESS" 2>/dev/null | sort -u | tr '\n' ' ')"
harness_names_loaded="$(printf '%s\n' $harness_provides | grep -E '^[A-Za-z_]' | sort -u | tr '\n' ' ')"
echo "  harness names the reader derives: $harness_names_derived"
echo "  harness names bash says it loaded: $harness_names_loaded"
require "the reader found names in the harness at all, so the comparison has a side" \
  test -n "$harness_names_derived"
require "bash recorded the names the harness loaded, so the comparison has a second side" \
  test -n "$harness_names_loaded"
assert "every name the reader says the harness defines is one bash loaded from it, and none it did not" \
  test "$harness_names_derived" = "$harness_names_loaded"

# --- Control: the derivation refuses to come back empty ----------------------
#
# The precondition above says this file will not compare an empty set, and that
# is this file's own protection. The audit is run by tests/lib/harness.sh at
# every summary of every suite as well, and there the derivation had no floor
# under it at all: handed a harness that defines no function, it read a suite
# whose first line was `assert() {` and reported nothing about it, because an
# empty name set makes the shadowing half of the reader unconditionally silent.
# A derived side that can be emptied from outside and says nothing is a short
# list with the list taken out, which is not the repair.
#
# So the script refuses it, and this is the control that makes it say so. Both
# directions, because "it exited nonzero" is also what a found violation looks
# like: the refusal has to be the usage status and it has to name the harness.
empty_harness="$fixtures/a-harness-that-defines-nothing.sh"
printf '# A harness with no function definition in it.\npass=0\nfail=0\n' \
  > "$empty_harness"
shadowing_suite="$fixtures/shadows-the-shared-assert.sh"
printf 'assert() {\n  echo "PASS: $1"\n}\n' > "$shadowing_suite"

audit_refuses_as_unusable() {
  local out
  local code=0
  out="$("$AUDIT" --harness "$1" "$2" 2>&1)" || code=$?
  if [ "$code" -ne 2 ]; then
    printf 'expected the usage status 2, got %s:\n%s\n' "$code" "$out" >&2
    return 1
  fi
  printf '%s\n' "$out" | grep -q 'no function definition was found in the harness'
}

assert "an audit whose harness defines nothing refuses rather than reading a suite against an empty set" \
  audit_refuses_as_unusable "$empty_harness" "$shadowing_suite"
assert "the same suite against the real harness is refused for the shadow itself, so the case is about the harness" \
  audit_rejects "$shadowing_suite"

# What bash itself says a file defines. The oracle.
name_oracle="$fixtures/what-bash-defines.sh"
cat > "$name_oracle" <<'EOF'
#!/usr/bin/env bash
# Source the file and let bash answer. `declare -F` is not a reading of the
# text; it is the shell's own record of which names it bound.
. "$1" >/dev/null 2>&1 || true
declare -F | awk '{ print $NF }' | sort -u
EOF
chmod +x "$name_oracle"

bash_defines_in() {
  "$BASH" "$name_oracle" "$1" 2>/dev/null | tr '\n' ' '
}

reader_agrees_with_bash_about_definitions() {
  local fix="$1"
  local from_bash
  local from_reader
  from_bash="$(bash_defines_in "$fix")"
  from_reader="$("$AUDIT" --names "$fix" 2>/dev/null | sort -u | tr '\n' ' ')"
  if [ "$from_bash" = "$from_reader" ]; then
    return 0
  fi
  printf 'bash defines [%s]\nthe reader says [%s]\n' "$from_bash" "$from_reader" >&2
  return 1
}

# The verdict the audit must return, computed from the oracle rather than
# written down: a file that defines a harness name is refused, and one that does
# not is accepted. A repair that made everything a redefinition would pass the
# first half and fail this.
audit_verdict_follows_bash() {
  local fix="$1"
  local defined
  local nm
  defined=" $(bash_defines_in "$fix")"
  for nm in $harness_provides; do
    case "$defined" in
      *" $nm "*) audit_rejects "$fix"; return $? ;;
    esac
  done
  audit_accepts "$fix"
}

spellings="$fixtures/spellings"
mkdir -p "$spellings"
spelling_n=0
spelling_shadowing=0
spelling_innocent=0
while IFS= read -r spelling_template; do
  [ -n "$spelling_template" ] || continue
  spelling_n=$((spelling_n + 1))
  spelling_fix="$spellings/case-$spelling_n.sh"
  : > "$spelling_fix"
  for spelling_name in $harness_provides definitely_not_a_harness_name; do
    printf '%s\n' "$spelling_template" | sed "s/@N/$spelling_name/g" >> "$spelling_fix"
  done
  case " $(bash_defines_in "$spelling_fix")" in
    *" assert "*) spelling_shadowing=$((spelling_shadowing + 1)) ;;
    *)            spelling_innocent=$((spelling_innocent + 1)) ;;
  esac
  assert "the reader and bash agree about what \`$spelling_template\` defines" \
    reader_agrees_with_bash_about_definitions "$spelling_fix"
  assert "the audit's verdict on \`$spelling_template\` follows what bash defined" \
    audit_verdict_follows_bash "$spelling_fix"
done <<'DEFINITION_SPELLINGS'
@N() { :; }
@N () { :; }
@N  ()  { :; }
@N ( ) { :; }
@N (  ) { :; }
@N(){ :; }
function @N { :; }
function @N() { :; }
function @N () { :; }
function @N ( ) { :; }
function @N(){ :; }
@N() { :; }  # with a trailing comment
echo "@N ( ) {"
: @N
DEFINITION_SPELLINGS

echo "  definition spellings driven: $spelling_n ($spelling_shadowing defining, $spelling_innocent not)"
# Both directions have to have happened, or the sweep passed by never reaching
# the condition it tests — which is the shape this file refuses elsewhere and
# the shape the release gate found in tests/test_skill.sh's own controls.
assert "the spelling sweep drove a spelling bash treats as a definition" \
  test "$spelling_shadowing" -gt 0
assert "the spelling sweep drove one bash does not, so the refusal is not unconditional" \
  test "$spelling_innocent" -gt 0

# --- The EXIT trap, held against whether the guard is still there -------------
#
# The reader used to look for the word EXIT, and `trap cleanup 0` and `trap
# cleanup exit` install exactly the handler `trap cleanup EXIT` installs —
# verified by firing all three on both shells. `trap - EXIT` and `trap '' EXIT`
# remove it while naming no handler at all, so "does a handler fire" is the
# wrong oracle too. The invariant is whether the **harness's own** guard is
# still the EXIT trap afterwards, and that is what is asked.
trap_oracle="$fixtures/does-the-abort-guard-survive.sh"
cat > "$trap_oracle" <<'EOF'
#!/usr/bin/env bash
# Install a guard the way harness_init does, run the line under test, and let
# the guard say whether it is still installed. $1 is the marker, $2 the fixture.
guard() { printf 'SURVIVED\n' >> "$MARKER"; }
MARKER="$1"
export MARKER
trap guard EXIT
. "$2" >/dev/null 2>&1 || true
exit 0
EOF
chmod +x "$trap_oracle"

abort_guard_survives() {
  local marker="$1"
  local fix="$2"
  : > "$marker"
  "$BASH" "$trap_oracle" "$marker" "$fix" >/dev/null 2>&1 || true
  [ -s "$marker" ]
}

audit_verdict_follows_the_guard() {
  local fix="$1"
  local marker="$2"
  if abort_guard_survives "$marker" "$fix"; then
    audit_accepts "$fix"
  else
    audit_rejects "$fix"
  fi
}

traps="$fixtures/traps"
mkdir -p "$traps"
trap_n=0
trap_displacing=0
trap_harmless=0
while IFS= read -r trap_template; do
  [ -n "$trap_template" ] || continue
  trap_n=$((trap_n + 1))
  trap_fix="$traps/case-$trap_n.sh"
  printf 'cleanup() { :; }\n%s\n' "$trap_template" > "$trap_fix"
  trap_marker="$traps/marker-$trap_n"
  if abort_guard_survives "$trap_marker" "$trap_fix"; then
    trap_harmless=$((trap_harmless + 1))
  else
    trap_displacing=$((trap_displacing + 1))
  fi
  assert "the audit's verdict on \`$trap_template\` follows whether the abort guard survived it" \
    audit_verdict_follows_the_guard "$trap_fix" "$trap_marker"
done <<'TRAP_SPELLINGS'
trap cleanup EXIT
trap 'cleanup' EXIT
trap "cleanup" EXIT
trap cleanup 0
trap cleanup exit
trap cleanup Exit
trap -- cleanup EXIT
trap cleanup INT EXIT
trap - EXIT
trap '' EXIT
trap cleanup INT
trap 'echo interrupted' INT
trap -p EXIT
TRAP_SPELLINGS

echo "  EXIT-trap spellings driven: $trap_n ($trap_displacing displacing the guard, $trap_harmless leaving it)"
assert "the trap sweep drove a spelling that displaces the abort guard" \
  test "$trap_displacing" -gt 0
assert "the trap sweep drove one that leaves it, so the refusal is not unconditional" \
  test "$trap_harmless" -gt 0

# --- The library a suite sources is part of the suite ------------------------
#
# The set of files was the one side with no derivation at all, and it is the one
# that cost the most: four of the seven suites source a library out of
# tests/lib/, the audit read only what a caller passed, and one line appended to
# tests/lib/masked-path.sh redefined `assert_value` while the audit reported
# `sites=767 files=7` and exit 0. tests/test_f01.sh printed its exact published
# 1886 passed, 0 failed over a tree with skills/skill-audit/SKILL.md replaced by
# the word BROKEN.
shadow_lib="$fixtures/library-with-a-private-assert.sh"
cat > "$shadow_lib" <<'EOF'
# A library, not a suite. Every suite that sources it gets this instead of the
# shared harness, and no repair to the shared one reaches it.
assert_value() { echo "PASS: $1"; }
EOF
clean_lib="$fixtures/library-with-nothing-private.sh"
cat > "$clean_lib" <<'EOF'
a_helper_that_shadows_nothing() { echo hello; }
EOF

printf 'source %s\nassert "a check the repository decides" test -d tests\n' \
  "$shadow_lib" > "$fixtures/sources-a-shadow.sh"
printf 'source %s\nassert "a check the repository decides" test -d tests\n' \
  "$clean_lib" > "$fixtures/sources-a-clean-library.sh"

assert "the audit follows a source into a library and finds the private copy there" \
  audit_rejects "$fixtures/sources-a-shadow.sh"
assert "the audit leaves a suite whose library shadows nothing, so the refusal is the shadow's doing" \
  audit_accepts "$fixtures/sources-a-clean-library.sh"
assert "the file a suite sources is in the set the audit reads" \
  quietly grep -qxF -- "$clean_lib" <<CLEAN_LIB_CLOSURE
$(audited_closure "$fixtures/sources-a-clean-library.sh")
CLEAN_LIB_CLOSURE

# And the direction that makes the derivation total rather than smaller: a
# `source` this cannot resolve is a file whose text would be part of the suite
# and would not be read, so it is refused rather than shrugged at. A derivation
# that quietly returns a smaller world is the list it replaced.
printf '. "$A_PATH_NOTHING_IN_THIS_TREE_CAN_RESOLVE"\nassert "a check" test -d tests\n' \
  > "$fixtures/sources-something-unresolvable.sh"
assert "the audit refuses a source directive it cannot resolve to a file" \
  audit_rejects "$fixtures/sources-something-unresolvable.sh"

# --- The runaway heredoc, in the suite it was walked through -----------------
#
# The reproduction, kept: one legal, non-vacuous assertion whose **quoted**
# argument contains `<<INSTALL_ROUTE`. The old reader ran a greedy `sub` over
# the raw line, did not know it was inside quotes, and read the rest of the file
# as heredoc body — `sites=1 files=1 lines=2846 accounted=2846` with `wc -l` at
# 2846, so both sides of the two-sided count agreed and the audit exited 0 with
# two vacuous assertions in the swallowed region.
#
# Driven over a copy of a real suite and not over a short fixture, because the
# defect is about swallowing the rest of a file and a four-line fixture has no
# rest to swallow — a control that cannot reach the condition it tests is the
# shape this file refuses.
quoted_opener_line='  quietly grep -qv '"'"'<<INSTALL_ROUTE'"'"' tests/test_install.sh'
walked_suite="$fixtures/suite-with-a-quoted-heredoc-opener.sh"
{
  head -n 20 tests/test_install.sh
  printf 'assert "the install route writes no heredoc terminator of its own" \\\n%s\n' "$quoted_opener_line"
  tail -n +21 tests/test_install.sh
  printf 'assert "the destination is contained" true\n'
  printf 'assert_value "the copy landed" true\n'
} > "$walked_suite"

# The same file with the two vacuous assertions left off, so the refusal above
# is theirs and not the quoted opener's — the quoted word is legal and must not
# be a finding of its own.
innocent_opener_suite="$fixtures/suite-with-only-the-quoted-opener.sh"
{
  head -n 20 tests/test_install.sh
  printf 'assert "the install route writes no heredoc terminator of its own" \\\n%s\n' "$quoted_opener_line"
  tail -n +21 tests/test_install.sh
} > "$innocent_opener_suite"

assert "a quoted <<WORD does not swallow the file, so the vacuous assertions after it are still found" \
  audit_rejects "$walked_suite"
assert "the quoted <<WORD is not itself a finding, so the refusal above is the vacuous assertions'" \
  audit_accepts "$innocent_opener_suite"

# A heredoc that never closes is the whole class, whatever opened it, and it is
# refused with no counter: a runaway heredoc is by definition one that reaches
# the end of the file without its terminator.
printf 'cat > /dev/null <<A_TERMINATOR_NEVER_WRITTEN\nsome body\nassert "swallowed and unexamined" true\n' \
  > "$fixtures/runaway-heredoc.sh"
printf 'cat > /dev/null <<A_TERMINATOR\nsome body\nA_TERMINATOR\nassert "a real check" test -d tests\n' \
  > "$fixtures/closed-heredoc.sh"
assert "the audit refuses a file whose heredoc is never terminated" \
  audit_rejects "$fixtures/runaway-heredoc.sh"
assert "the audit accepts a heredoc that closes, so the refusal is the runaway's doing" \
  audit_accepts "$fixtures/closed-heredoc.sh"

# --- The harness's own runtime guards, made to fire --------------------------
#
# tests/lib/harness.sh asks bash three questions at every summary, and all three
# can only ever say nothing. So each is handed the thing it watches for.
#
# Every fixture below is run with `"$BASH"` — the interpreter this suite is
# itself running under — and not with a bare `bash`, which is whichever one is
# first on PATH. That is the same reason `bash_defines_in` and
# `abort_guard_survives` already do it, applied to the rest of the file: with a
# bare `bash` these controls ran under 5.3.15 whichever shell drove the suite,
# so "the abort guard fires on both shells" was a claim about one of them and a
# 3.2-only fail-open in any of these three guards had no control that could
# reach it. Driven under 3.2.57 for the first time, all nine pass — including
# `declare -F` under `extdebug` naming the file that defined a function, which
# every one of the shadowing checks rests on.
shadow_after_load="$fixtures/shadows-after-loading.sh"
cat > "$shadow_after_load" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
. "$HARNESS_LIB"
harness_init
assert "control: a check the repository decides" test -d tests
assert_value() { echo "PASS: $1"; }
harness_summary
EOF
shadow_out="$harness_scratch/shadows-after-loading.out"
shadow_code=0
HARNESS_LIB="$PWD/$HARNESS" "$BASH" "$shadow_after_load" > "$shadow_out" 2>&1 \
  || shadow_code=$?
assert "a suite that redefines a harness name after loading is told whose copy is live" \
  grep -q 'is running a private copy of the harness' "$shadow_out"
assert "a suite that redefines a harness name after loading fails" \
  test "$shadow_code" -ne 0

# The census: an assertion that ran at a line the reader never examined. The
# audit is copied and crippled rather than the real one edited, and the fixture
# points the harness at the copy the same way read_check_over points this file's
# helpers at one.
crippled_audit="$fixtures/audit-that-stops-at-line-5.sh"
awk '/^FNR == 1 \{$/ { print "FNR > 5 { next }" } { print }' "$AUDIT" > "$crippled_audit"
chmod +x "$crippled_audit"
assert "the crippled audit really is a different program" \
  quietly grep -qF 'FNR > 5' "$crippled_audit"

census_fixture="$fixtures/runs-a-site-the-reader-missed.sh"
cat > "$census_fixture" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
. "$HARNESS_LIB"
harness_init
assert "control: a check the reader can see" test -d tests
assert "control: a check past where the reader stopped" test -d skills
harness_audit="$CRIPPLED_AUDIT"
harness_summary
EOF
census_out="$harness_scratch/census.out"
census_code=0
HARNESS_LIB="$PWD/$HARNESS" CRIPPLED_AUDIT="$crippled_audit" \
  "$BASH" "$census_fixture" > "$census_out" 2>&1 || census_code=$?
assert "a suite that ran an assertion the reader never examined says which line" \
  grep -q 'ran an assertion at a line the audit never examined' "$census_out"
assert "a suite that ran an assertion the reader never examined fails" \
  test "$census_code" -ne 0
# The other direction, over the same fixture pointed at the real audit, so the
# refusal above is the crippling and not the fixture.
census_ok_out="$harness_scratch/census-ok.out"
census_ok_code=0
HARNESS_LIB="$PWD/$HARNESS" CRIPPLED_AUDIT="$PWD/$AUDIT" \
  "$BASH" "$census_fixture" > "$census_ok_out" 2>&1 || census_ok_code=$?
assert "the same fixture against the real audit passes, so the census refuses the crippling" \
  test "$census_ok_code" -eq 0

# The closure: code loaded by a route no reader can follow. `eval` is that
# route, and it is the honest limit of any text reader — which is why the check
# is bash's and not the reader's.
smuggled_lib="$fixtures/loaded-without-a-readable-directive.sh"
cat > "$smuggled_lib" <<'EOF'
a_function_no_reader_saw_arrive() { echo hello; }
EOF
smuggle_fixture="$fixtures/loads-code-the-reader-cannot-see.sh"
cat > "$smuggle_fixture" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
. "$HARNESS_LIB"
harness_init
eval ". \"$SMUGGLED_LIB\""
assert "control: a check the repository decides" test -d tests
harness_summary
EOF
smuggle_out="$harness_scratch/smuggled.out"
smuggle_code=0
HARNESS_LIB="$PWD/$HARNESS" SMUGGLED_LIB="$smuggled_lib" \
  "$BASH" "$smuggle_fixture" > "$smuggle_out" 2>&1 || smuggle_code=$?
assert "a suite running code the audit never read is told which file" \
  grep -q 'running code from a file the assertion audit never read' "$smuggle_out"
assert "a suite running code the audit never read fails" \
  test "$smuggle_code" -ne 0

# --- A path named in prose is a path that exists -----------------------------
#
# Four sentences in tests/lib/harness.sh named a sibling of this cluster,
# `audit-assertions.sh`, under the lib directory — including the one asserting
# that the non-vacuity of every `assert_value` "rests on" it. No file of that
# name has ever existed; the file is `audit-suites.sh`. One of those sentences
# is load-bearing documentation of the mechanism this whole cluster is, and a
# reader who went looking found nothing. Nothing read it, which is why it
# survived four releases — so something reads it now. This check is why the
# sentence above spells that name without a directory in front of it.
lib_paths_named_in() {
  grep -hoE 'tests/lib/[A-Za-z0-9_-]+\.sh' "$@" | sort -u || :
}
echo "  tests/lib paths named in the audited files: $(lib_paths_named_in $audit_closure "$HARNESS" "$AUDIT" | tr '\n' ' ')"
require "a tests/lib path is named somewhere in these files, so this has a set to walk" \
  test -n "$(lib_paths_named_in $audit_closure "$HARNESS" "$AUDIT")"
while IFS= read -r named_path; do
  [ -n "$named_path" ] || continue
  assert "$named_path, named in the harness cluster, is a file that exists" \
    test -f "$named_path"
done <<NAMED_LIB_PATHS
$(lib_paths_named_in $audit_closure "$HARNESS" "$AUDIT")
NAMED_LIB_PATHS

# --- Controls: the harness reports, and says so when it cannot ---------------

# A failing check prints its own label, reaches the summary, and exits nonzero.
# All three, because the defect was never "it exited 0" — it was that the label
# was never mentioned and no verdict was printed.
cat > "$fixtures/one-failure.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
. "$HARNESS_LIB"
harness_init
assert "control: a check the repository fails" test -e tests/there-is-no-such-file
harness_summary
EOF

failing_out="$harness_scratch/one-failure.out"
failing_code=0
HARNESS_LIB="$PWD/$HARNESS" "$BASH" "$fixtures/one-failure.sh" > "$failing_out" 2>&1 \
  || failing_code=$?

assert "a failing check prints FAIL against its own label" \
  grep -q '^FAIL: control: a check the repository fails$' "$failing_out"
assert "a failing check still reaches the summary" \
  grep -q '^0 passed, 1 failed$' "$failing_out"
assert "a failing check exits nonzero" test "$failing_code" -ne 0

# A suite that dies before its summary says so. This is the half that was
# missing from three of four suites: the exit status was already nonzero and
# nobody was told why, or that anything had been skipped.
cat > "$fixtures/aborts.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
. "$HARNESS_LIB"
harness_init
assert "control: a check that passes before the abort" test -d tests
false
harness_summary
EOF

abort_out="$harness_scratch/aborts.out"
abort_code=0
HARNESS_LIB="$PWD/$HARNESS" "$BASH" "$fixtures/aborts.sh" > "$abort_out" 2>&1 \
  || abort_code=$?

assert "a suite that dies before its summary says it aborted" \
  grep -q '^FAIL: suite aborted before reaching its summary (exit 1)$' "$abort_out"
assert "a suite that dies before its summary prints no summary" \
  lacks '^[0-9]+ passed' "$abort_out"
assert "a suite that dies before its summary exits nonzero" test "$abort_code" -ne 0

# --- Controls: a precondition reports *and refuses* --------------------------
#
# `assert` is a reporter. It prints FAIL, counts it, and returns, so the line
# after it runs — which is right for a check on repository state and wrong for a
# check that guards a destructive step. A suite that computes "the destination is
# not redirected away from the developer's live config", prints FAIL, and then
# runs the install anyway has *reported* a failure, not refused one; and a
# reported failure is not a refusal. That shape was live in
# tests/test_install.sh, where it ran `rm -rf` against a real skills directory
# after its own guard had already said no.
#
# So the harness provides the second shape too, and the refusal is what is
# asserted here: nothing after a failed precondition runs. The suite still
# reports — a precondition that aborted silently would trade this defect for the
# one the abort guard exists to catch.
cat > "$fixtures/refuses.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
. "$HARNESS_LIB"
harness_init
assert "control: a check that passes before the precondition" test -d tests
require "control: a precondition the repository fails" test -e tests/no-such-file
: > "$SIDE_EFFECT"
assert "control: an assertion after the failed precondition" test -d tests
harness_summary
EOF

refuse_out="$harness_scratch/refuses.out"
refuse_code=0
side_effect="$harness_scratch/the-step-the-precondition-guards"
HARNESS_LIB="$PWD/$HARNESS" SIDE_EFFECT="$side_effect" \
  "$BASH" "$fixtures/refuses.sh" > "$refuse_out" 2>&1 || refuse_code=$?

assert "a failed precondition prints FAIL against its own label" \
  grep -q '^FAIL: control: a precondition the repository fails$' "$refuse_out"
assert "a failed precondition stops the step it guards from running" \
  test ! -e "$side_effect"
assert "a failed precondition stops the assertions after it from running" \
  lacks '^(PASS|FAIL): control: an assertion after the failed precondition$' \
  "$refuse_out"
assert "a refusing suite still reports the verdict it reached" \
  grep -q '^1 passed, 1 failed$' "$refuse_out"
assert "a refusing suite is a report, not a silent abort" \
  lacks 'aborted before reaching its summary' "$refuse_out"
assert "a refusing suite exits nonzero" test "$refuse_code" -ne 0

# And the other half: a precondition the repository meets is not a stop. A
# `require` that always halted would pass every check above.
cat > "$fixtures/requires-met.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
. "$HARNESS_LIB"
harness_init
require "control: a precondition the repository meets" test -d tests
assert "control: an assertion after the met precondition" test -d skills
harness_summary
EOF

met_out="$harness_scratch/requires-met.out"
met_code=0
HARNESS_LIB="$PWD/$HARNESS" "$BASH" "$fixtures/requires-met.sh" > "$met_out" 2>&1 \
  || met_code=$?

assert "a met precondition lets the suite run on" \
  grep -q '^PASS: control: an assertion after the met precondition$' "$met_out"
assert "a met precondition reaches the summary and passes" \
  grep -q '^2 passed, 0 failed$' "$met_out"
assert "a met precondition exits zero" test "$met_code" -eq 0

# The guard's own diagnostic does not depend on the guard's cleanup succeeding.
# errexit is live in a suite, and an EXIT handler that lost it would take its
# report down with a failing `rm` — the guard silently losing its report is the
# same defect one level in.
cat > "$fixtures/cleanup-fails.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
. "$HARNESS_LIB"
harness_init
echo "SCRATCH=$harness_scratch"
mkdir -p "$harness_scratch/locked/inner"
: > "$harness_scratch/locked/inner/file"
chmod 500 "$harness_scratch/locked/inner"
chmod 500 "$harness_scratch/locked"
false
harness_summary
EOF

stuck_out="$harness_scratch/cleanup-fails.out"
stuck_code=0
HARNESS_LIB="$PWD/$HARNESS" "$BASH" "$fixtures/cleanup-fails.sh" > "$stuck_out" 2>&1 \
  || stuck_code=$?

assert "the abort diagnostic survives a cleanup step that fails" \
  grep -q '^FAIL: suite aborted before reaching its summary (exit 1)$' "$stuck_out"
assert "a suite whose cleanup fails still exits nonzero" test "$stuck_code" -ne 0

# The control left a directory it could not remove. Take it back, or the run
# leaks it — which is a thing this cluster has already had to fix once.
stuck_scratch="$(sed -n 's/^SCRATCH=//p' "$stuck_out")"
if [ -n "$stuck_scratch" ] && [ -d "$stuck_scratch" ]; then
  chmod -R u+rwx "$stuck_scratch" 2>/dev/null || true
  rm -rf "$stuck_scratch"
fi
assert "the control's own scratch directory is not left behind" \
  lacks_dir "$stuck_scratch"

# --- The repository boundary, and whether anything holds it -------------------
#
# This file is about enforcement that cannot report. `.gitignore` is where the
# repository states, in a form `git add -A` obeys, which paths are not a
# release's to stage. Through 0.5.0 that list carried the skillgate work, which
# sat in the working tree beside the release; 0.6.0 lands it, so those four
# entries are gone and what remains is local scratch and the control plane.
# They were landed with nothing asserting them, so a later change could delete
# one and every suite would still print the same verdict — the boundary would
# be gone and the only thing that would notice is a reviewer reading a diff.
#
# The second direction below is not a formality, and after a release that moves
# paths across the boundary it is the one that matters: an entry a release
# lifts has to be shown *lifted*, or a pattern left behind goes on hiding files
# nobody is looking for — and the ignored half would stay just as green.
#
# That is the shape this file exists to refuse, one level up: a guard whose
# removal is invisible. So the entries are asserted here.
#
# The list is written down rather than derived, and that is deliberate. Every
# other count in this file is read out of a second place that already holds it;
# this one has no second place, because the boundary's only statement of itself
# *is* `.gitignore`, and deriving the list from the file under test would make
# deleting an entry delete the check for it. Writing it down is what makes the
# deletion visible.
#
# Asked of `git check-ignore` rather than of the file's text, so each assertion
# is about the effect an entry has and not about how it is spelled: reordering,
# recommenting or rewriting `.venv/` as `.venv*/` leaves these alone, and
# removing the enforcement does not.
#
# `--no-index` is load-bearing and was not obvious. Without it check-ignore
# consults the index first and reports every *tracked* path as not ignored,
# whatever the patterns say — so the second direction below, asked of files that
# are all tracked, answered "still stageable" for an over-broad pattern that
# would have hidden every file added under it from then on. The question worth
# asking is about the patterns, not about what happens to be tracked today: a
# path already in the index is safe either way, and the file that is about to be
# written beside it is the one that disappears.

# ignored_in <dir> <path> — the tree's boundary matches this path.
ignored_in() {
  local dir="$1"
  local path="$2"
  git -C "$dir" check-ignore --no-index -q -- "$path"
}

# stageable_in <dir> <path> — the tree's boundary does not match this path.
#
# Written as its own function rather than as `! ignored_in`, because
# check-ignore answers "not ignored" and "I could not answer" with 1 and 128,
# and a negation reads both as the reassuring one.
stageable_in() {
  local dir="$1"
  local path="$2"
  local status=0
  git -C "$dir" check-ignore --no-index -q -- "$path" || status=$?
  [ "$status" -eq 1 ]
}

# Local scratch and the control plane: the paths no release stages.
# `.venv-skillspector/` is named as well as `.venv/`: the widening from
# `.venv/` to `.venv*/` is what keeps a 15,000-file virtualenv out, and an
# entry narrowed back would pass a check that only asked about `.venv/`.
boundary_ignored="tmp/ .venv/ .venv-skillspector/ .scuba/"

# The other direction: an over-broad pattern is the more expensive mistake. It
# blocks a later slice silently, and the release has several left. Every path
# here is one a remaining slice has to be able to stage.
#
# The skillgate paths are here because 0.6.0 landed them, and they are named at
# a file rather than at their directory on purpose: `skillgate/` asked of
# `check-ignore` is a question about the directory entry, while
# `skillgate/gate.go` is the question a slice actually has — can I write a file
# under here — and it is the one a re-added directory pattern would answer no.
boundary_stageable="README.md AGENTS.md .gitignore .out-of-scope.md docs/profiler-spec.md .github/workflows/ci.yml profiler/types.go profiler/claude_code.go profiler/cmd/main.go skills/skill-audit/SKILL.md skills/skill-rewrite/SKILL.md tests/test_harness.sh go.work go.work.sum NOTICE docs/skillgate-spec.md docs/skillgate-intent.md skillgate/gate.go skills/skill-gate/SKILL.md"

for path in $boundary_ignored; do
  assert "the boundary hides $path" ignored_in . "$path"
done

for path in $boundary_stageable; do
  assert "the boundary leaves $path stageable" stageable_in . "$path"
done

# --- Controls: both directions of the boundary check can report ---------------
#
# Each half above can only ever say the reassuring thing — "still ignored" and
# "still stageable" are also what a check that had stopped consulting anything
# would say. So both predicates are run against trees carrying this
# repository's boundary with one deliberate defect in it.
#
# Real git repositories rather than a grep over text, because `check-ignore` is
# the thing under control and a control that exercises something else proves
# nothing about it.

# boundary_tree <dir> — an empty repository for a boundary to be written into.
boundary_tree() {
  local dir="$1"
  mkdir -p "$dir"
  quietly git -C "$dir" init
}

# A boundary missing one of its entries: the change a later slice could make
# without anything noticing, which is why these assertions exist.
#
# The entry it deletes is `.scuba/`, and which entry that is carries a lesson.
# It was `skills/skill-gate/` until 0.6.0 lifted that line, and the control
# then built a `.gitignore` identical to the real one — `grep -vFx` removing
# nothing — so the "entry that was removed" assertion passed on a tree where
# nothing had been removed, and the line-count `require` beside it was the only
# thing that could say so. Both halves are load-bearing: the count proves the
# control did something, and the pair of assertions proves what it did. An
# entry chosen here must be one no release is going to lift.
missing_entry="$fixtures/boundary-without-scuba"
require "the tree for the missing-entry control is a repository" \
  boundary_tree "$missing_entry"
grep -vFx '.scuba/' .gitignore > "$missing_entry/.gitignore"

require "the missing-entry control is this repository's boundary minus one line" \
  test "$(wc -l < "$missing_entry/.gitignore")" \
    -eq "$(( $(wc -l < .gitignore) - 1 ))"

assert "the boundary check reports an entry that was removed" \
  stageable_in "$missing_entry" .scuba/
assert "the missing-entry control keeps the rest of the boundary" \
  ignored_in "$missing_entry" tmp/

# A boundary with one pattern too wide: the other mistake, and the one that
# fails silently — it hides every file added under the path from then on.
too_wide="$fixtures/boundary-too-wide"
require "the tree for the over-broad control is a repository" \
  boundary_tree "$too_wide"
cat .gitignore > "$too_wide/.gitignore"
echo 'profiler/' >> "$too_wide/.gitignore"

assert "the boundary check reports a pattern that hides a path a slice needs" \
  ignored_in "$too_wide" profiler/types.go
assert "the over-broad control leaves the paths it did not widen alone" \
  stageable_in "$too_wide" README.md

# --- The index barrier, and both directions of it -----------------------------
#
# `.gitignore` decides what `git add` will pick up; the index is where the
# question is finally settled, because `git add -f` and a narrowed entry both
# get past the first check. tests/lib/out-of-scope-check.sh is that last look,
# and it is asserted here for the same reason the boundary above is: it was a
# sentence in a ruling that each slice retyped, and a check nobody can see run
# is a check nobody notices the loss of.
#
# Both directions, over real repositories, because the thing under test is what
# the script concludes from an index. The clean direction matters more than it
# looks: the form this replaces exited 1 on a clean index — `grep` reports 1
# when it finds nothing — so it announced a block precisely when there was
# nothing to block, and a green run of it proved nothing at all.

out_of_scope_check="$harness_repo_root/tests/lib/out-of-scope-check.sh"

require "the out-of-scope check is an executable script" \
  test -x "$out_of_scope_check"

# staged_tree <dir> <path>... — a repository with exactly these paths staged.
#
# `git add -f`, deliberately: what this check is for is the path that got past
# `.gitignore`, and a fixture that could not stage one would assert nothing.
staged_tree() {
  local dir
  local path
  dir="$1"
  shift
  mkdir -p "$dir"
  quietly git -C "$dir" init || return 1
  for path in "$@"; do
    mkdir -p "$dir/$(dirname "$path")"
    echo x > "$dir/$path"
    quietly git -C "$dir" add -f -- "$path" || return 1
  done
}

# check_passes <dir> <outfile> — the barrier lets this index through, and its
# words are kept for the assertion that reads them.
check_passes() {
  "$out_of_scope_check" "$1" > "$2" 2>&1
}

# check_refuses <dir> <outfile> — the barrier blocks this index.
#
# Exactly 1, not merely nonzero. The script exits 2 when it cannot run at all,
# and a control that accepted any failure would go on passing against a check
# that had stopped working — which is the shape this whole file refuses.
check_refuses() {
  local status
  status=0
  "$out_of_scope_check" "$1" > "$2" 2>&1 || status=$?
  [ "$status" -eq 1 ]
}

# An index carrying only paths this release owns. This is the direction the
# inverted form got wrong, so it is asserted first.
in_scope="$fixtures/index-in-scope"
in_scope_out="$harness_scratch/in-scope.out"
require "the tree for the in-scope control is a repository with a staged index" \
  staged_tree "$in_scope" profiler/types.go docs/profiler-spec.md README.md

assert "the barrier passes an index carrying only in-scope paths" \
  check_passes "$in_scope" "$in_scope_out"
assert "the barrier says it is clean, rather than leaving a bare grep status as the verdict" \
  grep -q '^out-of-scope check: clean$' "$in_scope_out"

# An empty index is the same answer and is asked separately: it is the state a
# `grep` that matched nothing reports as a failure, and the first one a
# pre-commit hook written from this ever meets.
empty_index="$fixtures/index-empty"
empty_index_out="$harness_scratch/empty-index.out"
require "the tree for the empty-index control is a repository" \
  staged_tree "$empty_index"

assert "the barrier passes an index with nothing staged at all" \
  check_passes "$empty_index" "$empty_index_out"

# The other direction, one repository per pattern: a single fixture staging all
# of them would still pass with only one alternative left working.
#
# Five of these were skillgate paths until 0.6.0 landed them, and a control
# that goes on naming a path the check no longer refuses does not fail — it
# stops being a control, quietly, which is the one failure mode this whole file
# exists to refuse. So the list is the alternatives the pattern *has*, and it
# shrinks when the pattern does. `.venv-skillspector/` is here beside `.venv/`
# for the reason the boundary names it too: the arm is `\.venv` rather than
# `\.venv/`, and a control that only ever staged `.venv/…` would pass against
# an arm narrowed back to the directory.
for out_of_scope_path in \
  docs/research/notes.md \
  .venv/pyvenv.cfg \
  .venv-skillspector/lib/python3.12/site.py
do
  blocked_tree="$fixtures/index-blocked-$(printf '%s' "$out_of_scope_path" | tr '/.' '--')"
  blocked_out="$harness_scratch/blocked-$(printf '%s' "$out_of_scope_path" | tr '/.' '--').out"
  require "the tree for the $out_of_scope_path control is a repository with it staged" \
    staged_tree "$blocked_tree" "$out_of_scope_path"

  assert "the barrier refuses an index staging $out_of_scope_path" \
    check_refuses "$blocked_tree" "$blocked_out"
  assert "the refusal of $out_of_scope_path names the path it found" \
    grep -qF -- "$out_of_scope_path" "$blocked_out"
done

# An in-scope path that begins like a refused one. Without this the pattern
# could be widened to bare substrings — dropping the `^`, or dropping the
# trailing `/` off `docs/research/` — and every assertion above would still
# pass while a later slice found its own files refused.
#
# Each member here defends one character the pattern cannot lose.
# `docs/research.md` is a *tracked* file of this repository sitting one
# character away from the `docs/research/` arm, so dropping that `/` would
# refuse a file `main` already carries. `vendor/.venv-cache/x` carries `.venv`
# somewhere other than the front, which is what the `^` is for. The member this
# replaces was `vendor/NOTICE`, and it guarded the anchors on a `NOTICE$` arm
# that 0.6.0 removed: a near-miss for a pattern that is gone misses nothing.
near_miss="$fixtures/index-near-miss"
near_miss_out="$harness_scratch/near-miss.out"
require "the tree for the near-miss control is a repository with its paths staged" \
  staged_tree "$near_miss" skills/skill-audit/SKILL.md docs/profiler-spec.md docs/research.md vendor/.venv-cache/x

assert "the barrier passes an index staging paths that merely read like the refused ones" \
  check_passes "$near_miss" "$near_miss_out"

# --- The mutation runner, and a verdict in every direction ---------------------
#
# A mutation proves a check by reverting the mechanism the check exists to hold
# and requiring the check to go red. What makes that a proof rather than a
# ritual is that *not going red* has more than one meaning, and this release met
# five of them:
#
#   - a mutant that did not compile, so the tests never ran and a search for a
#     failure line found nothing;
#   - a safety barrier's own proof that **skipped**, because it built a path
#     from a directory that was not there;
#   - a mutant that **never terminated**;
#   - a check that stayed green against a pattern too wide to fail, because
#     `git` consults the index before the working tree;
#   - a scan that refused itself, having named what it refuses and then scanned
#     its own directory.
#
# Four of those five are indistinguishable from a pass in a pass/fail tally and
# the fifth is indistinguishable from a slow machine. Every one of them was
# found separately, by a different slice, because every slice from S2 on
# re-derived the runner by hand from the last one — five times by S8's count.
# That is the copy-by-hand pattern tests/lib/harness.sh exists to end: five
# copies of a runner are five chances for one of them to be the copy that
# stopped distinguishing, and the drift is invisible because each copy is only
# ever run once.
#
# So the runner lives in tests/lib/ beside the harness, and it is held here, in
# the suite for the substrate the other suites are written on, by this file's
# own rule: every direction it can report is made to happen and the report is
# read. The contract is one sentence — **a verdict is printed in every
# direction, and only KILLED is a success** — so silence is the defect and a
# direction with no assertion over it is the runner half-held.

mutation_runner=tests/lib/mutation-runner.sh

require "the mutation runner is there to be loaded" test -f "$mutation_runner"
. "$mutation_runner"
require "the mutation runner defined every name it exists to provide" \
  mutation_runner_ready

mut="$harness_scratch/mutation"
mkdir -p "$mut"
mut_target="$mut/target"
mut_out="$mut/run.out"

# The mutators. One changes the file and one does not, because "the edit was a
# no-op" is itself a direction: nothing was tested, and a runner that reported
# the suite's own green as the answer would be crediting a mutation it never
# made.
mutant_edit() { printf 'mutated\n' > "$1"; }
mutant_noop() { return 0; }

# The suites, one per direction, each printing what a real suite prints when it
# reaches that direction. They are fakes deliberately: the question here is what
# the runner concludes from a suite's output and status, and no real suite can
# be made to reach seven different directions on demand — which is exactly why
# five of the seven were only ever met one slice at a time.
suite_kills() { printf '\n3 passed, 1 failed\n'; return 1; }
suite_survives() { printf '\n4 passed, 0 failed\n'; return 0; }
suite_unloadable() {
  echo 'tests/x.sh: line 9: syntax error near unexpected token `fi'"'"
  return 2
}
suite_asserts_nothing() { printf '\n0 passed, 0 failed\n'; return 0; }
suite_refuses() {
  printf 'PASS: the first check\n'
  printf 'REFUSED: the precondition above failed, so nothing that depended on it ran\n'
  printf '\n1 passed, 1 failed\n'
  return 1
}
suite_never_ends() { while :; do sleep 1; done; }

# ran <mutator> <suite> — one run, with the runner's own line captured so the
# assertions can read what it *said* as well as what it decided. A verdict the
# runner reaches and does not print is the defect, so both are asserted every
# time.
ran() {
  printf 'original\n' > "$mut_target"
  mut_status=0
  mutation_run MX "$mut_target" "$1" "$2" > "$mut_out" 2>&1 || mut_status=$?
  return 0
}

mutation_format=harness
mutation_deadline=1
mutation_allow_skips=0

ran mutant_edit suite_kills
assert "a mutant the suite fails on is KILLED" \
  test "$mutation_verdict" = KILLED
assert "the KILLED verdict is printed and not only returned" \
  grep -q 'KILLED' "$mut_out"
assert "KILLED is the one direction that succeeds" test "$mut_status" -eq 0
assert "a killed run restored the file it mutated" \
  test "$(cat "$mut_target")" = original

ran mutant_edit suite_survives
assert "a mutant every check passes is SURVIVED" \
  test "$mutation_verdict" = SURVIVED
assert "the SURVIVED verdict is printed" grep -q 'SURVIVED' "$mut_out"
assert "SURVIVED is not a success" test "$mut_status" -ne 0
assert "a surviving run restored the file it mutated" \
  test "$(cat "$mut_target")" = original

# The first of the five. A suite that never loaded prints no verdict, and the
# runner that searched its output for a failure line found none and read that as
# a pass. So the rule is inverted: the *absence* of a verdict is a verdict, and
# it is never a pass.
ran mutant_edit suite_unloadable
assert "a mutant whose suite prints no verdict at all is DID NOT BUILD" \
  test "$mutation_verdict" = "DID NOT BUILD"
assert "the DID NOT BUILD verdict is printed" grep -q 'DID NOT BUILD' "$mut_out"
assert "DID NOT BUILD is not a success" test "$mut_status" -ne 0
assert "a run that never built restored the file it mutated" \
  test "$(cat "$mut_target")" = original

# The last of the five, and the one that is a passing tally: a scan that named
# what it refuses, scanned its own directory, and so examined nothing. It
# reaches its summary and reports zero of both.
ran mutant_edit suite_asserts_nothing
assert "a suite that reached its summary having asserted nothing is DID NOT RUN" \
  test "$mutation_verdict" = "DID NOT RUN"
assert "the DID NOT RUN verdict is printed" grep -q 'DID NOT RUN' "$mut_out"
assert "DID NOT RUN is not a success" test "$mut_status" -ne 0

# The second of the five: the barrier's own proof that skipped. In a shell suite
# that is `require` refusing — the suite reports a tally, and the checks after
# the refusal never ran, so the tally cannot say whether the killing one was
# among them. It reads as KILLED on the failure count alone, which is why the
# skip outranks it.
ran mutant_edit suite_refuses
assert "a tally reached with checks that never ran is SKIPPED, not KILLED" \
  test "$mutation_verdict" = SKIPPED
assert "the SKIPPED verdict is printed" grep -q 'SKIPPED' "$mut_out"
assert "SKIPPED is not a success" test "$mut_status" -ne 0
assert "the SKIPPED report says how many checks did not run" \
  test "$mutation_skips" -eq 1

# And the knob that keeps the strictness usable: a suite with a skip by design
# declares how many, and only a skip beyond the declared count is a verdict.
# Without this the rule above would refuse every run of a suite that carries a
# deliberate skip, and the pressure would be to drop the rule.
mutation_allow_skips=1
ran mutant_edit suite_refuses
assert "a skip the caller declared is not a SKIPPED verdict" \
  test "$mutation_verdict" = KILLED
assert "the declared skip is still counted and reported" \
  test "$mutation_skips" -eq 1
mutation_allow_skips=0

# The third of the five, and the only one that does not look like a pass — it
# looks like a slow machine, which is why it sat for ten minutes. The deadline
# is the runner's own, so it holds for a suite with no timeout flag of its own.
mut_started=$SECONDS
ran mutant_edit suite_never_ends
mut_elapsed=$((SECONDS - mut_started))
assert "a mutant whose suite never terminates is DID NOT END" \
  test "$mutation_verdict" = "DID NOT END"
assert "the DID NOT END verdict is printed" grep -q 'DID NOT END' "$mut_out"
assert "DID NOT END is not a success" test "$mut_status" -ne 0
assert "the runner stopped the suite at its deadline rather than waiting on it" \
  test "$mut_elapsed" -lt 15
assert "a run that never ended restored the file it mutated" \
  test "$(cat "$mut_target")" = original

# The edit that was not an edit. Carried from the runners the slices wrote,
# where it caught a `sed` whose pattern had stopped matching the line it was
# written for — so the mutation was never applied and the suite's green was the
# unmutated tree's.
ran mutant_noop suite_survives
assert "a mutator that changed nothing is NOT APPLIED" \
  test "$mutation_verdict" = "NOT APPLIED"
assert "the NOT APPLIED verdict is printed" grep -q 'NOT APPLIED' "$mut_out"
assert "NOT APPLIED is not a success" test "$mut_status" -ne 0
assert "a no-op mutation does not report the suite's own result" \
  test "$mutation_verdict" != SURVIVED

# The control on all of the above: the verdict is a function of the suite's
# output, so no two directions may share one. A runner that had collapsed two of
# them would still satisfy every assertion that reads a single verdict.
mutation_verdicts_are_distinct() {
  local pair
  local seen
  seen=""
  for pair in \
    "mutant_edit suite_kills" \
    "mutant_edit suite_survives" \
    "mutant_edit suite_unloadable" \
    "mutant_edit suite_asserts_nothing" \
    "mutant_edit suite_refuses" \
    "mutant_edit suite_never_ends" \
    "mutant_noop suite_survives"
  do
    ran $pair
    case "$seen" in
      *"[$mutation_verdict]"*)
        echo "  two directions share the verdict $mutation_verdict" >&2
        return 1 ;;
    esac
    seen="$seen[$mutation_verdict]"
  done
  return 0
}
assert "the seven directions reach seven different verdicts" \
  mutation_verdicts_are_distinct

# --- and the same seven read out of `go test` ---------------------------------
#
# The format the five re-derived runners were written against, because the
# slices that needed them were Go slices. It is the same seven directions and a
# different way of reading them, which is the whole reason the reading is a
# parameter and not a grep inlined at the verdict.
#
# One correction came with the move. The scratch runners recognised a
# non-building mutant by searching for a dozen compiler-error phrases — a list
# that grows with every error Go learns to emit and fails closed to SURVIVED
# when one is missing. `go test` marks it once, in a line of its own, and that
# is what is read.
mutation_format=go

suite_go_kills() {
  printf '=== RUN   TestX\n--- FAIL: TestX (0.00s)\nFAIL\nFAIL\tgithub.com/x/y\t0.301s\nFAIL\n'
  return 1
}
suite_go_unbuildable() {
  printf '# github.com/x/y [github.com/x/y.test]\n'
  printf './y_test.go:12:2: declared and not used: got\n'
  printf 'FAIL\tgithub.com/x/y [build failed]\n'
  return 1
}
suite_go_skips() {
  printf '=== RUN   TestBarrier\n--- SKIP: TestBarrier (0.00s)\nPASS\nok  \tgithub.com/x/y\t0.201s\n'
  return 0
}
suite_go_times_out() {
  printf 'panic: test timed out after 30s\n\tgoroutine 1 [running]:\n'
  return 2
}

ran mutant_edit suite_go_kills
assert "a Go mutant a test fails on is KILLED" test "$mutation_verdict" = KILLED
assert "the Go KILLED verdict is printed" grep -q 'KILLED' "$mut_out"

# `FAIL <pkg> [build failed]` is a nonzero exit with no `--- FAIL` in it, which
# is the exact shape the phrase-list runners read as SURVIVED.
ran mutant_edit suite_go_unbuildable
assert "a Go mutant that does not compile is DID NOT BUILD" \
  test "$mutation_verdict" = "DID NOT BUILD"
assert "the Go DID NOT BUILD verdict is printed" grep -q 'DID NOT BUILD' "$mut_out"
assert "a Go mutant that does not compile is not read as a kill" \
  test "$mutation_verdict" != KILLED

# The barrier's own proof, in the form it actually took: `go test` exits 0 with
# a SKIP in it, so the suite is green and the case that would have killed the
# mutant never ran.
ran mutant_edit suite_go_skips
assert "a Go run with a skipped test is SKIPPED, not SURVIVED" \
  test "$mutation_verdict" = SKIPPED
assert "the Go SKIPPED verdict is printed" grep -q 'SKIPPED' "$mut_out"
assert "the Go SKIPPED report says how many tests did not run" \
  test "$mutation_skips" -eq 1

# Read from the output as well as from the deadline, because `go test -timeout`
# usually fires first and prints this rather than hanging until the runner's own
# deadline. Two detectors, one verdict.
ran mutant_edit suite_go_times_out
assert "a Go run that reports its own timeout is DID NOT END" \
  test "$mutation_verdict" = "DID NOT END"
assert "the Go DID NOT END verdict is printed" grep -q 'DID NOT END' "$mut_out"

mutation_format=harness

harness_summary
