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
# The suite list is a glob, so a fifth suite added tomorrow is covered the day
# it lands. Both halves are read out of the source text by
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

# The denominator. An empty or shrunken glob would make every per-suite check
# below vacuously true, which is the failure this file exists to refuse.
assert "tests/ still holds every suite this check is written against" \
  test "$suite_count" -ge 5
echo "  suites examined:$suites"

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
    grep -qF -- "$suite" .github/workflows/ci.yml
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

cat > "$fixtures/trailing-true.sh" <<'EOF'
mkdir -p tests && assert "a vacuous assertion after a separator" true
EOF
assert "the audit reads an assertion that is not the first word on its line" \
  audit_rejects "$fixtures/trailing-true.sh"

# --- Controls: the audit accepts a verdict the repository decides -------------

cat > "$fixtures/real-command.sh" <<'EOF'
assert "tests/ exists" test -d tests
assert "the harness is readable" quietly cat tests/lib/harness.sh
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
