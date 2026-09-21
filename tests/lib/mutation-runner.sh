# The shared mutation runner: revert a mechanism, run the checks that hold it,
# and say — always in words — what happened.
#
# A mutation proves a check. The check exists to hold some mechanism; the
# mutation reverts the mechanism; the check must go red. That is the whole idea,
# and the reason it needs a runner rather than a habit is that **not going red
# has six different meanings**, five of which are indistinguishable from a pass
# and one of which is indistinguishable from a slow machine:
#
#   - the mutant did not build, so no check ran and a search of the output for a
#     failure line found nothing;
#   - the suite reached a tally with some of its checks **skipped**, so the
#     tally cannot say whether the killing one was among them — the form this
#     took was a safety barrier's own proof, skipping because it built a path
#     from a directory that was not there;
#   - the suite **never terminated**, and sat for ten minutes with nothing to
#     report;
#   - the suite ran and asserted nothing at all — a scan that named what it
#     refuses and then scanned its own directory, and a check that stayed green
#     against a pattern too wide to fail;
#   - the edit was a no-op, so the suite's green belonged to the unmutated tree.
#
# Each of those was found separately, by a different slice of one release,
# because every slice from the second on re-derived this runner by hand from the
# last one — five times over. That is the copy-by-hand pattern
# tests/lib/harness.sh exists to end, in its sharpest form: five copies of a
# runner are five chances for one of them to be the copy that stopped
# distinguishing, and the drift is invisible because each copy is only ever run
# once, by its author, on the mutations its author chose.
#
# So the machinery lives here once, and one invariant holds the file together:
#
#   a verdict is printed in every direction, and only KILLED is a success.
#
# **A silent pass is the defect.** There is no path through mutation_run that
# returns without naming what it decided, including the paths where it decided
# nothing could be decided. That is the same rule as
# tests/lib/out-of-scope-check.sh's — a bare grep status is not a verdict — one
# level up: a runner that left "the suite exited 0" standing for "the check
# works" is what four of the five meanings above got past.
#
# The seven verdicts:
#
#   KILLED         the mutant built, every check ran, and at least one reported
#                  a failure. The only success.
#   SURVIVED       the mutant built, every check ran, and none reported a
#                  failure. A real gap in the checks.
#   DID NOT BUILD  the suite reached no verdict at all. It could not load, could
#                  not compile, or died before its summary. It proves nothing —
#                  unless the build failure *is* the kill, because the defect is
#                  unrepresentable in the types, and then the caller says so in
#                  its own words rather than reading it out of this verdict.
#   DID NOT RUN    the suite reached a verdict that counts nothing: zero passed
#                  and zero failed. It started and examined nothing.
#   SKIPPED        the suite reached a verdict with more checks skipped than the
#                  caller declared. Outranks KILLED on purpose: a failure count
#                  taken from a run with unexplained skips is a count of the
#                  checks that happened to run.
#   DID NOT END    the suite was still running at the deadline, or reported its
#                  own timeout.
#   NOT APPLIED    the mutator left the file byte-identical. Nothing was tested.
#
# Use:
#
#   . "$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)/lib/harness.sh"
#   harness_init
#   . tests/lib/mutation-runner.sh
#
#   mutate_drop_the_guard() { sed -e 's/^  guard$//' "$1" > "$1.t" && mv "$1.t" "$1"; }
#   run_the_suite()         { bash tests/test_thing.sh; }
#
#   assert "dropping the guard is killed" \
#     mutation_run M1 skills/x/scripts/x.sh mutate_drop_the_guard run_the_suite
#
# mutation_run returns 0 only for KILLED, so it composes with `assert` and the
# desired outcome is the passing one. The other verdicts return distinct
# statuses (below) so a caller that wants to branch can, without parsing words.

# --- Configuration ------------------------------------------------------------
#
# Read at each call rather than captured at source time, so a suite can change
# one between runs — which it must, because a suite that mutates both a shell
# script and a Go package needs both readers.
#
# `mutation_format` is a parameter rather than a grep inlined at the verdict
# because the reading is the part that differs between consumers and the six
# meanings are the part that does not. Collapsing them would put the format's
# details inside the classification, which is where the phrase-list defect below
# lived.
mutation_format="${mutation_format:-harness}"

# Seconds. The runner's own bound, so it holds for a suite with no timeout of
# its own — which every shell suite here is. Go's default is ten minutes, long
# enough to read as a hang, which is how DID NOT END was found.
mutation_deadline="${mutation_deadline:-120}"

# How many skipped checks the caller has accounted for. Default zero: an
# unexplained skip is a verdict. A suite with a deliberate skip states the
# number, which keeps the rule usable — without it the rule would refuse every
# run of such a suite and the pressure would be to drop the rule rather than to
# declare the skip.
#
# **The number for `go test ./...` over profiler/ is 1**, and it is worth
# writing down because the first run of any Go mutation reports SKIPPED
# otherwise: `TestHomeBarrierChild` in profiler/internal/homesafe is the body a
# parent test runs in a child process, and it skips in every ordinary run by
# design. Declaring it is the point — the previous runners printed the skip as a
# note beside a KILLED verdict, so nobody ever had to say which skip it was, and
# a second skip appearing would have read the same way.
mutation_allow_skips="${mutation_allow_skips:-0}"

# --- What the last run decided ------------------------------------------------
#
# Set on every path, including the ones that decide nothing can be decided, so a
# caller reading these after a call never reads the previous call's answer.
mutation_verdict=""
mutation_reason=""
mutation_passed=0
mutation_failed=0
mutation_skips=0
mutation_output=""

# Where the captured output and the backup live. The harness's scratch directory
# when there is one, because it is already owned and already removed by the
# harness's own EXIT guard; a directory of this file's own otherwise, so the
# runner is usable from a script that is not a suite.
mutation_scratch=""

# mutation_scratch_ready — establish the scratch directory, or fail.
#
# Not written to return the path on stdout, because the assignment has to escape
# to the caller's shell: a `$(...)` would set the global inside a subshell and
# the next call would make a second directory. That is the same mistake
# harness_init avoids by taking its own scratch in the shell that installs the
# trap.
mutation_scratch_ready() {
  if [ -n "$mutation_scratch" ] && [ -d "$mutation_scratch" ]; then
    return 0
  fi
  if [ -n "${harness_scratch:-}" ] && [ -d "${harness_scratch:-}" ]; then
    mutation_scratch="$harness_scratch/mutation-runner"
  else
    mutation_scratch="$(mktemp -d)" || return 1
  fi
  [ -n "$mutation_scratch" ] || return 1
  mkdir -p "$mutation_scratch" || return 1
  return 0
}

# --- Reading a suite's output -------------------------------------------------
#
# A reader sets `mutation_state` to one of `verdict`, `nobuild` or `timeout`,
# and on `verdict` sets the three counts. `nobuild` and `timeout` carry their
# own reason, because what they mean depends on the format and the classifier
# below is deliberately ignorant of it.
#
# The rule both readers are built on, and the one that closes the first of the
# six meanings: **the absence of a verdict is itself a verdict, and it is never
# a pass.** The runner this replaces searched the output for a failure line and
# read "no failure line" as "nothing failed", which is true of a suite that
# passed and equally true of a suite that never started.
mutation_state=""

# mutation_read_harness <output-file> — this repository's shell harness.
#
# The verdict is harness_summary's line, which is the one thing every suite here
# prints and prints last. A suite that died on the way to it prints the abort
# guard's line instead, and that is not a tally: it is the guard saying the
# suite reported nothing, which is exactly `nobuild`.
#
# A skipped check in a shell suite is `require` refusing — the harness prints
# REFUSED and stops, so every check after it never ran and the tally counts only
# what came before. It is read as a skip for that reason and not as a failure,
# even though `require` also increments the failure count: the failure is real
# and the *completeness* of the run is not, and the second of those is what a
# mutation's verdict rests on.
mutation_read_harness() {
  local out
  local summary
  out="$1"
  summary="$(grep -E '^[0-9]+ passed, [0-9]+ failed$' "$out" 2>/dev/null | tail -1)" \
    || summary=""
  if [ -z "$summary" ]; then
    mutation_state=nobuild
    if grep -q '^FAIL: suite aborted before reaching its summary' "$out" 2>/dev/null; then
      mutation_reason="the suite aborted before reaching its summary, so it reported no verdict"
    else
      mutation_reason="the suite printed no summary line, so it reached no verdict"
    fi
    return 0
  fi
  mutation_passed="${summary%% passed,*}"
  mutation_failed="${summary##*, }"
  mutation_failed="${mutation_failed%% failed}"
  mutation_skips="$(grep -c '^REFUSED:' "$out" 2>/dev/null)" || mutation_skips=0
  mutation_state=verdict
  return 0
}

# mutation_read_go <output-file> — `go test`.
#
# Written for `-v`, because the per-test lines are what the counts are read
# from; without it a passing package prints one `ok` line and this reader would
# have nothing to count, which lands as DID NOT RUN rather than as a false
# SURVIVED. Failing closed was the choice: the alternative is to credit a run
# whose contents were never printed.
#
# The non-building mutant is recognised from `go test`'s own marker and not from
# a list of compiler-error phrases. The five scratch runners searched for a
# dozen of them — `undefined:`, `declared and not used`, `has no field or
# method` — a list that has to grow with every error Go learns to emit, and
# whose failure mode is to fall through to SURVIVED. `go test` already says it
# once, in a line of its own, for every reason a package might not build.
#
# The timeout is read here too rather than left to the deadline, because
# `go test -timeout` normally fires first: it panics, prints this, and exits, so
# the runner's watchdog never sees a process to stop. Two independent detectors,
# one verdict.
mutation_read_go() {
  local out
  out="$1"
  if grep -q 'panic: test timed out' "$out" 2>/dev/null; then
    mutation_state=timeout
    mutation_reason="go test reported its own timeout, so the suite did not terminate"
    return 0
  fi
  if grep -qE '\[(build|setup) failed\]' "$out" 2>/dev/null; then
    mutation_state=nobuild
    mutation_reason="go test marked the package [build failed], so no test in it ran"
    return 0
  fi
  mutation_passed="$(grep -cE '^ *--- PASS' "$out" 2>/dev/null)" || mutation_passed=0
  mutation_failed="$(grep -cE '^ *--- FAIL' "$out" 2>/dev/null)" || mutation_failed=0
  mutation_skips="$(grep -cE '^ *--- SKIP' "$out" 2>/dev/null)" || mutation_skips=0
  if ! grep -qE '^(ok|FAIL|\?)[	 ]' "$out" 2>/dev/null \
     && ! grep -qE '^ *--- (PASS|FAIL|SKIP)' "$out" 2>/dev/null; then
    mutation_state=nobuild
    mutation_reason="go test printed neither a package result nor a test result, so nothing ran"
    return 0
  fi
  mutation_state=verdict
  return 0
}

# --- Running the suite under a deadline ---------------------------------------

# mutation_bounded <output-file> <command> [args...] — run the command with its
# output captured, and stop it at `mutation_deadline` seconds.
#
# Returns 124 when the deadline fired, otherwise the command's own status.
#
# A background job polled with `kill -0` rather than `wait -n` with a sleeping
# sibling, because `wait -n` is bash 4.3 and this repository runs its suites on
# bash 3.2 as well. TERM then KILL, because a suite that ignores TERM is still a
# suite that has to be stopped — a deadline that can be declined is not one.
mutation_bounded() {
  local out
  local pid
  local waited
  local status
  out="$1"
  shift
  ( "$@" ) > "$out" 2>&1 &
  pid=$!
  waited=0
  while kill -0 "$pid" 2>/dev/null; do
    if [ "$waited" -ge "$mutation_deadline" ]; then
      kill -TERM "$pid" 2>/dev/null || true
      sleep 1
      kill -KILL "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
      return 124
    fi
    sleep 1
    waited=$((waited + 1))
  done
  status=0
  wait "$pid" 2>/dev/null || status=$?
  return "$status"
}

# --- The runner ---------------------------------------------------------------

# mutation_run <id> <target-file> <mutator> <suite>
#
# <mutator> is invoked as `<mutator> <target-file>` and edits the file in place.
# <suite> is invoked with no arguments and its two output channels are captured
# together.
#
# Returns 0 for KILLED, 1 SURVIVED, 2 NOT APPLIED, 3 DID NOT BUILD, 4 DID NOT
# RUN, 5 DID NOT END, 6 SKIPPED, and 7 when the runner itself could not run or
# could not put the file back.
#
# The restore is unconditional and it is *verified*, on every path including the
# ones that end in a refusal. A runner that left a mutated file behind would
# hand the next mutation a tree that is already wrong, and every verdict after
# it would be about a mutant nobody chose — so a restore that did not restore is
# the one thing here louder than a verdict. `cp` from a backup, never a
# `checkout`, `reset`, `clean`, `stash` or `rebase`: the file may be one a
# release is holding uncommitted, and a git operation would take the rest of the
# tree with it.
mutation_run() {
  local id
  local target
  local mutator
  local suite
  local backup
  local out
  local status
  local deadline_fired

  mutation_verdict=""
  mutation_reason=""
  mutation_passed=0
  mutation_failed=0
  mutation_skips=0
  mutation_output=""
  mutation_state=""

  if [ "$#" -ne 4 ]; then
    mutation_verdict="CANNOT RUN"
    mutation_reason="mutation_run takes <id> <target-file> <mutator> <suite>; it was given $# arguments"
    mutation_report
    return 7
  fi

  id="$1"
  target="$2"
  mutator="$3"
  suite="$4"

  if [ ! -f "$target" ]; then
    mutation_verdict="CANNOT RUN"
    mutation_reason="there is no file at $target to mutate"
    mutation_report "$id"
    return 7
  fi
  if ! mutation_scratch_ready; then
    mutation_verdict="CANNOT RUN"
    mutation_reason="no scratch directory could be made, so no backup of $target could be taken"
    mutation_report "$id"
    return 7
  fi

  backup="$mutation_scratch/backup"
  out="$mutation_scratch/output"
  mutation_output="$out"

  if ! cp -- "$target" "$backup" || ! cmp -s -- "$target" "$backup"; then
    mutation_verdict="CANNOT RUN"
    mutation_reason="no verified backup of $target could be taken, so it must not be mutated"
    mutation_report "$id"
    return 7
  fi

  # The mutator's own status is read. An unread one was how a `sed` that had
  # stopped matching became a green run: the edit failed, the file was
  # unchanged, and the suite was asked about the tree it already held.
  status=0
  "$mutator" "$target" || status=$?
  if [ "$status" -ne 0 ]; then
    mutation_restore "$id" "$target" "$backup" || return 7
    mutation_verdict="CANNOT RUN"
    mutation_reason="the mutator $mutator exited $status, so no mutation was made"
    mutation_report "$id"
    return 7
  fi

  if cmp -s -- "$target" "$backup"; then
    mutation_restore "$id" "$target" "$backup" || return 7
    mutation_verdict="NOT APPLIED"
    mutation_reason="$mutator left $target byte-identical, so nothing was tested"
    mutation_report "$id"
    return 2
  fi

  deadline_fired=false
  status=0
  mutation_bounded "$out" "$suite" || status=$?
  if [ "$status" -eq 124 ]; then
    deadline_fired=true
  fi

  mutation_restore "$id" "$target" "$backup" || return 7

  case "$mutation_format" in
    harness) mutation_read_harness "$out" ;;
    go) mutation_read_go "$out" ;;
    *)
      mutation_verdict="CANNOT RUN"
      mutation_reason="mutation_format is [$mutation_format], which is not one of harness or go"
      mutation_report "$id"
      return 7 ;;
  esac

  if [ "$deadline_fired" = true ]; then
    mutation_verdict="DID NOT END"
    mutation_reason="the suite was still running after $mutation_deadline seconds and was stopped, so it reported nothing"
    mutation_report "$id"
    return 5
  fi
  if [ "$mutation_state" = timeout ]; then
    mutation_verdict="DID NOT END"
    mutation_report "$id"
    return 5
  fi
  if [ "$mutation_state" = nobuild ]; then
    mutation_verdict="DID NOT BUILD"
    mutation_report "$id"
    return 3
  fi
  if [ "$mutation_skips" -gt "$mutation_allow_skips" ]; then
    mutation_verdict="SKIPPED"
    mutation_reason="$mutation_skips of the suite's checks did not run and $mutation_allow_skips were accounted for, so its tally of $mutation_failed failed counts only the checks that ran"
    mutation_report "$id"
    return 6
  fi
  if [ "$mutation_failed" -gt 0 ]; then
    mutation_verdict="KILLED"
    mutation_reason="$mutation_failed of $((mutation_passed + mutation_failed)) checks went red"
    mutation_report "$id"
    return 0
  fi
  if [ "$mutation_passed" -eq 0 ]; then
    mutation_verdict="DID NOT RUN"
    mutation_reason="the suite reached its verdict having counted nothing at all, so no check examined the mutation"
    mutation_report "$id"
    return 4
  fi
  mutation_verdict="SURVIVED"
  mutation_reason="the mutant ran and all $mutation_passed checks passed, so nothing holds the mechanism it reverted"
  mutation_report "$id"
  return 1
}

# mutation_restore <id> <target> <backup> — put the file back, and prove it.
#
# The one refusal in this file that is louder than a verdict, and the reason it
# is a function rather than two lines at each of the five exits above: five
# copies of a restore are five chances for one of them to be the copy that
# stopped restoring, which is this file's own subject.
mutation_restore() {
  local id
  local target
  local backup
  id="$1"
  target="$2"
  backup="$3"
  if cp -- "$backup" "$target" && cmp -s -- "$backup" "$target"; then
    return 0
  fi
  mutation_verdict="NOT RESTORED"
  mutation_reason="$target could not be put back from $backup — the tree is still mutated and no further mutation may be run against it"
  mutation_report "$id"
  return 1
}

# mutation_report [id] — print the verdict. The one exit from this file.
#
# Printed rather than returned, because a status nobody printed is what the
# runners this replaces left a reader of a log to infer. The reason is on the
# same line as the verdict: a verdict with its reason a scroll away is read as
# the word alone.
mutation_report() {
  local id
  id="${1:-?}"
  printf '%s  %s — %s\n' "$id" "$mutation_verdict" "$mutation_reason"
  return 0
}

# mutation_runner_ready
#
# Succeed when every name this file exists to provide is defined. A file that
# stopped part-way through defines some of them and not others, and a caller
# that checked only the names it remembered would run on to `command not found`
# at the point it needed the one that is gone. Defined last on purpose, so
# "stopped short" fails by the same route as "never loaded" — the shape
# harness_ready and verdict_guard_ready both use.
mutation_runner_ready() {
  local n
  for n in mutation_scratch_ready mutation_read_harness mutation_read_go \
           mutation_bounded mutation_run mutation_restore mutation_report; do
    declare -F "$n" >/dev/null 2>&1 || return 1
  done
  return 0
}
