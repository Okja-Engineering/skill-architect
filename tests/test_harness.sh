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
  quietly "$AUDIT" "$@"
}

# audit_rejects <file>... — the audit finds one, and says where.
#
# Written as a captured run rather than `! audit`, because the expected output
# of a firing control is noise on a passing run, and because a control that
# silently stopped firing has to be able to say so.
audit_rejects() {
  local out
  if out="$("$AUDIT" "$@" 2>&1)"; then
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

# sites_counted <file>... — how many assertion call sites the audit examined.
#
# The audit's nonzero exit means "it found something", which is not a failure to
# count, so it is read for its tally either way.
sites_counted() {
  local out
  out="$("$AUDIT" "$@" 2>/dev/null)" || true
  printf '%s\n' "$out" | sed -n 's/^sites=\([0-9]*\).*/\1/p'
}

fixtures="$harness_scratch/fixtures"
mkdir -p "$fixtures"

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

workflow_suites() {
  grep -oE 'tests/test_[A-Za-z0-9_]+\.sh' "$WORKFLOW" | sort -u
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
done

# The audit ran over something. A tokenizer that matched nothing would report no
# violations, and every per-suite check above would pass on an empty reading.
examined="$(sites_counted $suites)"
echo "  assertion call sites examined: $examined"
assert "the audit examined every suite's assertions, not an empty reading" \
  test "${examined:-0}" -ge 200

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
HARNESS_LIB="$PWD/$HARNESS" bash "$fixtures/one-failure.sh" > "$failing_out" 2>&1 \
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
HARNESS_LIB="$PWD/$HARNESS" bash "$fixtures/aborts.sh" > "$abort_out" 2>&1 \
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
  bash "$fixtures/refuses.sh" > "$refuse_out" 2>&1 || refuse_code=$?

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
HARNESS_LIB="$PWD/$HARNESS" bash "$fixtures/requires-met.sh" > "$met_out" 2>&1 \
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
HARNESS_LIB="$PWD/$HARNESS" bash "$fixtures/cleanup-fails.sh" > "$stuck_out" 2>&1 \
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

harness_summary
