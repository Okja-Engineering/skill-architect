# The shared harness for every shell suite under tests/.
#
# One invariant holds this file together:
#
#   an assertion's verdict is a function of repository state, and a suite that
#   does not reach its summary says so.
#
# Both halves were broken, and both were broken the same way: the counters, the
# assertion and the exit handling were copied into each suite by hand, so each
# suite carried its own version of them and the versions drifted. One suite's
# `assert` took a value while another's took a command; one suite installed an
# abort guard while the other three installed a bare `rm -rf`. A repair applied
# to one copy left the others exactly as they were. So the machinery lives here
# once, and there is no longer a per-suite copy to forget or to update.
#
# What this file cannot enforce is the first half of the invariant: at the
# moment `assert` is called, a verdict computed from the repository and a
# verdict written as a literal are the same bytes. That question is decidable
# only in the source text, so it is asked there, by
# tests/lib/audit-assertions.sh over every suite, driven from
# tests/test_harness.sh. The two files are one mechanism.
#
# Use:
#
#   #!/usr/bin/env bash
#   set -euo pipefail
#   . "$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)/lib/harness.sh"
#   harness_init
#   ...
#   harness_summary

# Where the repository is, answered from this file's own location rather than
# from `$0` or the caller's cwd, so a suite resolves its fixtures the same way
# whatever directory it was started from. `CDPATH=` and `--` are not
# decoration: with CDPATH exported, `cd tests` can land in a `tests` somewhere
# else entirely and echo the path it chose, which puts a second line into the
# value and sends every relative path in the suite somewhere unintended.
harness_lib_dir="$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
harness_repo_root="$(CDPATH= cd -P -- "$harness_lib_dir/../.." && pwd -P)"

pass=0
fail=0
harness_summary_printed=false
harness_scratch=""

# The abort guard. A suite that dies before its summary has reported nothing,
# whatever its exit status says, and "nothing" is what a reader of a CI log
# takes for "nothing wrong". Both live failure modes produced it: bash 3.2 ran
# past a failed check and bash 5 aborted on it, and neither printed a verdict.
#
# errexit and nounset are dropped on the way in. This is the one handler that
# has to reach its last line whatever failed on the way to it, and under
# errexit a failing `rm` or an unset variable here would take the diagnostic
# down with it — the guard silently losing its own report is the defect this
# guard exists to prevent, one level up. For the same reason the diagnostic is
# printed before the scratch directory is removed: the report never waits on
# the cleanup.
harness_exit() {
  local code=$?
  set +e
  set +u
  if [ "$harness_summary_printed" != true ]; then
    echo
    echo "FAIL: suite aborted before reaching its summary (exit $code)"
    if [ "$code" -eq 0 ]; then
      code=1
    fi
  fi
  if [ -n "$harness_scratch" ]; then
    rm -rf "$harness_scratch"
  fi
  exit "$code"
}

# Enter the repository, install the guard, and hand the suite a scratch
# directory it does not own.
#
# The trap goes on before the directory exists, so there is no window in which
# a suite can die with the guard not yet installed. The directory belongs to
# the shell that installed the trap: a helper that made its own could be running
# inside a command substitution, where the assignment never escapes and the
# trap has nothing to clean up. Suites take subdirectories of it rather than
# calling `mktemp -d` again.
harness_init() {
  trap harness_exit EXIT
  cd "$harness_repo_root" || exit 1
  harness_scratch="$(mktemp -d)" || exit 1
}

# assert <label> <command> [args...]
#
# Run the command and report the label. Taking a command rather than a value is
# what keeps a caller from writing the real check as a bare `[[ ]]` on the
# preceding line and handing the assertion the literal `true` — the shape that
# made 23 assertions across two suites unable to print FAIL on any bash this
# project supports. It is not sufficient on its own: `assert "label" true` still
# passes, which is why audit-assertions.sh reads the call sites.
assert() {
  local label="$1"
  shift
  if "$@"; then
    echo "PASS: $label"
    pass=$((pass + 1))
  else
    echo "FAIL: $label"
    fail=$((fail + 1))
  fi
}

# assert_value <label> <verdict>
#
# Report the label on a verdict already computed as the string "true".
#
# This is the older of the two shapes, kept because the two large suites hold
# 819 assertions that compute their verdict inside a command substitution and
# pass the result — `assert_value "label" "$(cmd && echo true || echo false)"`.
# Those are not vacuous: the substitution runs. What makes the shape weaker is
# that a *literal* verdict is indistinguishable from a computed one once the
# call is made, so the non-vacuity of every one of them rests on
# audit-assertions.sh, which reads the call site and refuses a verdict with
# nothing to expand in it.
#
# Anything other than exactly "true" is a failure, so a substitution that
# produced something unexpected fails rather than passing.
assert_value() {
  local label="$1"
  local verdict="$2"
  if [ "$verdict" = true ]; then
    echo "PASS: $label"
    pass=$((pass + 1))
  else
    echo "FAIL: $label"
    fail=$((fail + 1))
  fi
}

# quietly <command> [args...]
#
# Run a command silently while it succeeds, and surface everything it said when
# it fails. A passing check should add no noise; a failing one must say why.
#
# It is transparent: the verdict is the command's. audit-assertions.sh knows
# that, and looks through it to the command underneath — `assert "label"
# quietly true` is as vacuous as `assert "label" true`.
quietly() {
  local output
  local status=0
  output="$("$@" 2>&1)" || status=$?
  if [ "$status" -ne 0 ]; then
    printf '%s\n' "$output" >&2
  fi
  return "$status"
}

# harness_summary — print the verdict and exit nonzero if anything failed.
#
# Reaching this is what the abort guard is watching for, so the flag is set
# here and nowhere else.
harness_summary() {
  echo
  echo "$pass passed, $fail failed"
  harness_summary_printed=true
  if [ "$fail" -gt 0 ]; then
    exit 1
  fi
}

# harness_ready
#
# Succeed when every name this file exists to provide is defined. A file that
# stopped part-way through defines some of them and not others, and a suite that
# checked only the names it remembered would run on to `command not found` at
# the point it needed the one that is gone. Defined last on purpose, so
# "stopped short" fails by the same route as "never loaded".
harness_ready() {
  local n
  for n in harness_init harness_exit assert assert_value quietly harness_summary; do
    declare -F "$n" >/dev/null 2>&1 || return 1
  done
  return 0
}
