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
# What this file cannot enforce by itself is the first half of the invariant: at
# the moment `assert` is called, a verdict computed from the repository and a
# verdict written as a literal are the same bytes. That question is decidable
# only in the source text, so it is asked there, by tests/lib/audit-suites.sh
# over every suite and every library a suite loads, driven from
# tests/test_harness.sh. The two files are one mechanism.
#
# And what the text reader cannot enforce, this file does, because bash knows
# things no reader of text does:
#
#   - **whose copy of a name is live.** `declare -F` under `shopt -s extdebug`
#     names the file that defined a function. Every name this file loaded is
#     held against that, so a redefinition is caught whatever spelling it was
#     written in and whichever file it came from. That is not a theoretical
#     gap: one line appended to tests/lib/masked-path.sh redefined
#     `assert_value`, and tests/test_f01.sh reported its exact published 1886
#     passed, 0 failed over a deliberately broken tree.
#   - **whether the abort guard is still installed.** `trap -p EXIT` is asked,
#     rather than a reader guessing at the spellings of a sigspec — `trap x 0`
#     and `trap x exit` install the same handler `trap x EXIT` does.
#   - **which call sites actually ran.** Every assertion records the line it was
#     called from, and the set is held against the sites the audit found. A call
#     site the reader did not reach reddens the suite that ran it, which is the
#     other side the site count never had.
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
harness_lib_file="$harness_lib_dir/$(basename "${BASH_SOURCE[0]}")"
harness_audit="$harness_lib_dir/audit-suites.sh"

pass=0
fail=0
harness_summary_printed=false
harness_scratch=""
harness_suite_file=""
harness_site_log=""

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

# harness_definition_file <name> — the file bash says defined that function.
#
# This is the whole of the other side. A text reader can only ever answer "I
# found no redefinition in the spellings I know", and the spellings it knew
# were three of bash's many: `assert ( ) {` is legal on 3.2.57 and on 5.3.15,
# defines the function, and was not a redefinition as far as that reader was
# concerned. bash has no such gap, because bash is what decides.
#
# extdebug is turned on for the question and put back exactly as it was, since
# it also turns on error and function tracing and a suite did not ask for them.
harness_definition_file() {
  local was=off
  local line
  if shopt -q extdebug; then was=on; fi
  shopt -s extdebug
  line="$(declare -F "$1" 2>/dev/null)" || line=""
  if [ "$was" = off ]; then shopt -u extdebug; fi
  [ -n "$line" ] || return 1
  line="${line#* }"
  line="${line#* }"
  printf '%s\n' "$line"
}

# harness_ready
#
# Succeed when this file's contract is the one the suite is actually running
# under. Three claims, and each of them was silently false at some point in
# this repository's history:
#
#   1. every name this file exists to provide is defined. A file that stopped
#      part-way through defines some and not others, and a suite that checked
#      only the names it remembered would run on to `command not found` at the
#      point it needed the one that is gone.
#   2. every one of them is still *this file's* copy. A library or a suite that
#      redefines `assert_value` gets a private harness that no repair here
#      reaches, and every assertion after that line reports whatever it likes.
#   3. the EXIT trap is still the abort guard. `trap cleanup 0` displaces it and
#      is not the word EXIT, which is why this asks bash rather than a reader.
#
# The set of names is not a list. It is recorded at the bottom of this file from
# bash's own answer, so a primitive added here is covered the moment it lands
# and a file that stopped short has recorded nothing — which makes "stopped
# short" fail by the same route as "never loaded", as it always claimed to.
harness_ready() {
  local n
  local where
  if [ -z "${harness_provides:-}" ]; then
    echo "the harness recorded none of the names it provides, so it did not finish loading" >&2
    return 1
  fi
  for n in $harness_provides; do
    if ! where="$(harness_definition_file "$n")"; then
      echo "the harness name '$n' is not defined, so this file stopped short of providing it" >&2
      return 1
    fi
    if [ ! "$where" -ef "$harness_lib_file" ]; then
      echo "the live '$n' was defined by $where, not by $harness_lib_file: this suite is running a private copy of the harness" >&2
      return 1
    fi
  done
  where="$(trap -p EXIT 2>/dev/null)" || where=""
  case "$where" in
    *harness_exit*) ;;
    *)
      echo "the EXIT trap is no longer the harness abort guard but ${where:-nothing at all}, so a suite that dies says nothing" >&2
      return 1
      ;;
  esac
  return 0
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
#
# Readiness is asked here and again at the summary, because the two questions
# are different: here it is "did this file load", and there it is "is it still
# the one running" — a library sourced by the suite after this point is exactly
# where the answer changed.
harness_init() {
  trap harness_exit EXIT
  cd "$harness_repo_root" || exit 1
  harness_suite_file="${BASH_SOURCE[${#BASH_SOURCE[@]} - 1]}"
  if ! harness_ready; then
    echo "FAIL: the harness this suite loaded is not the one in $harness_lib_file"
    exit 1
  fi
  harness_scratch="$(mktemp -d)" || exit 1
  harness_site_log="$harness_scratch/assertion-call-sites"
  : > "$harness_site_log"
}

# Every status this harness has watched a script exit with, as
# `<basename>:<code>` words. A suite can then ask whether a script's own
# `# Exit codes:` header is true of the script rather than only of the document
# that copies it: the header and the doc are compared to each other, and both
# could move together — a status added that the script can never emit, or one
# dropped that it emits on every failure — with nothing anywhere noticing.
# Recorded here rather than per suite because every run goes through this
# function, so there is no call site left to forget.
exit_witness=""

# witness_exit <cmd> <code> — record a status, if <cmd> was a script whose
# contract that status is a statement about. A run of `bash -c` is a statement
# about the fragment it was handed, and skill-validator's status is
# skill-validator's contract, not this repository's.
witness_exit() {
  case "$1" in
    *.sh) exit_witness="$exit_witness ${1##*/}:$2" ;;
  esac
}

# harness_note_site — record where the assertion above this frame was written.
#
# The census, and it is the other side the site count never had. The audit's
# `sites=` figure could only ever be compared against itself: a floor of 200
# against a real 767 accepted a 71% collapse, and the two-sided line count that
# replaced the floor agrees with itself over a file whose assertions were
# swallowed by a runaway heredoc, because it counts *disposal* and the heredoc
# path disposes of whatever it is handed. What no reader of text can fake is
# which lines bash actually ran an assertion on. Held against the audit's spans
# at the summary, this makes "the reader reached every call site" an answerable
# question with no number in it.
#
# The frame two out is the caller of `assert`; when that is this file it is
# `require` calling `assert`, and `require` has already recorded the suite's own
# line. Appended to a file rather than a variable because a suite may compute a
# verdict inside `$( )`, where an assignment never escapes.
#
# It cannot fail the suite: errexit is live in every caller, so every path out
# of here is a success.
harness_note_site() {
  local src="${BASH_SOURCE[2]:-}"
  local ln="${BASH_LINENO[1]:-}"
  if [ -n "$harness_site_log" ] && [ -n "$src" ] && [ -n "$ln" ] \
     && [ "$src" != "$harness_lib_file" ]; then
    printf '%s\t%s\n' "$src" "$ln" >> "$harness_site_log" 2>/dev/null || true
  fi
  return 0
}

# assert <label> <command> [args...]
#
# Run the command and report the label. Taking a command rather than a value is
# what keeps a caller from writing the real check as a bare `[[ ]]` on the
# preceding line and handing the assertion the literal `true` — the shape that
# made 23 assertions across two suites unable to print FAIL on any bash this
# project supports. It is not sufficient on its own: `assert "label" true` still
# passes, which is why audit-suites.sh reads the call sites.
assert() {
  harness_note_site
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

# require <label> <command> [args...]
#
# The same assertion, plus the refusal. `assert` reports and returns, so the
# line after it runs; that is right for a check on repository state and wrong
# for a check that guards a destructive step. A suite that computes "the
# destination is not redirected away from the developer's live config", prints
# FAIL, and then runs the install anyway has *reported* a failure, not refused
# one — and a reported failure is not a refusal. That was live in
# tests/test_install.sh, which ran `rm -rf` against a real skills directory
# after its own guard had said no.
#
# It lives here rather than in the suite that needed it because "a precondition
# that halts" is the same machinery for every suite, and a per-suite copy of it
# is the drift this file exists to end. It is `assert` plus three lines: the
# report, the count and the exit status all stay in one place, and the stop goes
# through harness_summary so a refusing suite still prints the verdict it
# reached. A precondition that aborted silently would trade this defect for the
# one the abort guard exists to catch.
require() {
  harness_note_site
  local failures_before="$fail"
  assert "$@"
  if [ "$fail" -ne "$failures_before" ]; then
    echo "REFUSED: the precondition above failed, so nothing that depended on it ran"
    harness_summary
  fi
}

# assert_value <label> <verdict>
#
# Report the label on a verdict already computed as the string "true".
#
# This is the older of the two shapes, kept because the two large suites hold
# most of their assertions in it: the verdict is computed inside a command
# substitution and the result is passed — `assert_value "label" "$(cmd && echo
# true || echo false)"`. Those are not vacuous: the substitution runs. What
# makes the shape weaker is that a *literal* verdict is indistinguishable from
# a computed one once the call is made, so the non-vacuity of every one of them
# rests on audit-suites.sh, which reads the call site and refuses a verdict with
# nothing to expand in it.
#
# There is deliberately no figure in this sentence. The one that stood here for
# four releases — "819 assertions" — was false under every derivation anybody
# could construct (396 or 398 call sites in those two suites, 402 across all
# seven, 2236 assertions executed, 217 call sites on the day it was written),
# nothing read it, and a number nobody can reproduce is worse than no number.
# The reproducible ones are printed by tests/test_harness.sh, which derives
# them, and by each suite's own summary.
#
# Anything other than exactly "true" is a failure, so a substitution that
# produced something unexpected fails rather than passing.
assert_value() {
  harness_note_site
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
# It is transparent: the verdict is the command's. audit-suites.sh knows
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

# Reading a producer's output, without asking a short-circuiting reader to read
# a pipe.
#
# `producer | grep -q PATTERN` and `producer | head -1` under `set -o pipefail`
# cannot tell a found answer from a dead producer. Both readers exit the moment
# they have their answer and close the pipe; a producer with anything still to
# write is then killed by SIGPIPE, the pipeline's status is 141 — or bash's own
# `printf` gets EPIPE, reports `write error: Broken pipe` and returns 1 — and
# `pipefail` hands that status to the assertion, which reports FAIL over an
# answer it actually got.
#
# Whether the producer had finished writing when the reader stopped listening
# is a race, so the shape does not fail, it fails *sometimes*. It reddened one
# of seven heading assertions in tests/test_skill.sh at random, and which
# assertion it took varied by run — so the report named a different check each
# time and none of them was what was wrong.
#
# The exposure is not the pipe by itself. It is a producer that does work
# *after* the reader is already blocked in `read()`, so its output lands on the
# far side of the reader's early exit: a function that builds a fixture and
# runs a script, a `find` that is still walking, a `sed` still reading a file.
# An `echo "$already_captured"` writes before the reader can exec and measured
# 0 in 400 on both shells at 400 bytes and at 40kB. The two primitives below
# are for the first kind, where the caller captures the producer's output
# first and there is no pipe left to race over.
#
# They live here rather than in the suite that needed them for the reason the
# rest of this file does: a per-suite copy is the drift this file exists to
# end, and `harness_provides` then covers them the moment they land, so a suite
# cannot quietly redefine one and get a private matcher no repair reaches.
#
# The controls are in tests/test_harness.sh, driven at a size larger than a
# pipe buffer so the shape they replace fails there every time rather than
# sometimes.

# holds_line <text> [grep-option]... <pattern>
#
# True when <text> holds a line grep matches.
#
# The options and the pattern are handed to grep untouched, so a call site
# keeps the dialect and the anchoring it already had — `-F`, `-x`, `-E`, `-i`
# and a leading `^` all mean here exactly what they meant in the pipeline this
# replaces. That is deliberate: a matcher that imposed its own dialect would
# widen or narrow every call site it was migrated to, and a widened match is a
# check that stops being able to fail, which is the defect this harness is
# about rather than a repair for it.
#
# The text reaches grep as a here-string. bash writes a here-string into its
# temporary file or its pipe *before* grep is started, so there is no producer
# left running for grep's early exit to kill — verified from 100 bytes to 2MB
# on 3.2.57 and on 5.3.15. `grep -q` is kept because the question really is
# "is it there", and with nothing writing behind it, stopping early is free.
holds_line() {
  local text="$1"
  shift
  grep -q "$@" <<<"$text"
}

# first_line <text>
#
# The first line of <text>, and nothing if <text> is empty — which is what
# `producer | head -1` reported, so a caller's comparison does not change.
#
# Parameter expansion rather than `head`, so there is no second process and no
# pipe at all. The caller captures the producer with `$( )`, which is where the
# producer's own exit status stays visible to it.
first_line() {
  printf '%s\n' "${1%%$'\n'*}"
}

# harness_census — every call site this suite ran is one the audit reached.
#
# The direction that matters is this one, and only this one is asserted: a line
# bash ran an assertion on which the text reader did not find is a hole in the
# reader, and it is the hole that let a runaway heredoc hide two vacuous
# assertions with every counter in the audit agreeing. The other direction is
# deliberately not asserted, because a call site inside a branch this run did
# not take is a site that legitimately never executes.
#
# The span and not the line, because the two shells disagree: for a call
# continued over three physical lines starting at 4, bash 3.2.57 reports line 5
# and bash 5.3.15 reports line 4. Both are inside the logical line, which is
# what the audit reports the extent of, so containment is the comparison that is
# true on both rather than a convention picked from one.
harness_census() {
  local spans
  local unreached
  local f
  if [ ! -x "$harness_audit" ]; then
    echo "the assertion audit is not executable at $harness_audit, so the call sites this suite ran stand behind nothing" >&2
    return 1
  fi
  if [ ! -s "$harness_site_log" ]; then
    echo "this suite recorded no assertion call site at all, so the census has nothing to hold and cannot report" >&2
    return 1
  fi
  set --
  while IFS= read -r f; do
    [ -n "$f" ] || continue
    [ -f "$f" ] || continue
    set -- "$@" "$f"
  done <<CENSUS_FILES
$(cut -f1 "$harness_site_log" | sort -u)
CENSUS_FILES
  if [ "$#" -eq 0 ]; then
    echo "every file this suite ran an assertion from has gone missing, so the census cannot be taken" >&2
    return 1
  fi
  spans="$harness_scratch/assertion-call-site-spans"
  "$harness_audit" --harness "$harness_lib_file" --sites "$@" > "$spans" 2>/dev/null || true
  if [ ! -s "$spans" ]; then
    echo "the audit found no assertion call site in $*, while this suite ran $(wc -l < "$harness_site_log" | tr -d '[:space:]') of them" >&2
    return 1
  fi
  unreached="$(
    sort -u "$harness_site_log" | awk -F'\t' '
      NR == FNR {
        for (i = $2 + 0; i <= $3 + 0 && i < $2 + 500; i++) seen[$1 ":" i] = 1
        next
      }
      !(($1 ":" $2) in seen) { print "  " $1 ":" $2 }
    ' "$spans" -
  )"
  if [ -n "$unreached" ]; then
    echo "this suite ran an assertion at a line the audit never examined, so the audit's verdict says nothing about it:" >&2
    printf '%s\n' "$unreached" >&2
    return 1
  fi
  return 0
}

# harness_loaded_files — every file that defined a function live in this shell.
#
# bash's answer to "whose text is running", which is the other side of the
# audit's closure under `source`. A library that a suite loads is as much a part
# of that suite as the suite's own text, and the audit used to read only what a
# caller passed it.
harness_loaded_files() {
  local n
  local where
  for n in $(declare -F | awk '{ print $NF }'); do
    where="$(harness_definition_file "$n")" || continue
    printf '%s\n' "$where"
  done | sort -u
}

# harness_closure_covers — every file whose code is running was audited.
#
# Both sides derived: bash names the files that defined the live functions, and
# the audit names the files it reads, following `source` from this suite. A
# library that is loaded and not read is the F4 shape; a library that is read
# and not loaded is harmless and is not asserted.
harness_closure_covers() {
  local closure
  local where
  local covered
  local missing=""
  closure="$("$harness_audit" --harness "$harness_lib_file" --closure "$harness_suite_file" 2>/dev/null)" || closure=""
  if [ -z "$closure" ]; then
    echo "the audit reports no file for $harness_suite_file, so nothing says which text was read" >&2
    return 1
  fi
  while IFS= read -r where; do
    [ -n "$where" ] || continue
    [ -f "$where" ] || continue
    if [ "$where" -ef "$harness_lib_file" ]; then continue; fi
    covered=no
    while IFS= read -r c; do
      [ -n "$c" ] || continue
      if [ -f "$c" ] && [ "$where" -ef "$c" ]; then covered=yes; break; fi
    done <<CLOSURE
$closure
CLOSURE
    if [ "$covered" = no ]; then missing="$missing  $where
"; fi
  done <<LOADED
$(harness_loaded_files)
LOADED
  if [ -n "$missing" ]; then
    echo "this suite is running code from a file the assertion audit never read:" >&2
    printf '%s' "$missing" >&2
    return 1
  fi
  return 0
}

# harness_summary — print the verdict and exit nonzero if anything failed.
#
# Reaching this is what the abort guard is watching for, so the flag is set
# here and nowhere else.
#
# The three integrity checks run before the verdict is printed, so a failure of
# one of them is counted in it. Each of them is silent when it holds: they are
# not assertions about the repository and adding them to the pass count would
# make every suite's published total a function of this file rather than of what
# the suite checks.
harness_summary() {
  if ! harness_ready; then
    echo "FAIL: the harness this suite is running is not the one in $harness_lib_file"
    fail=$((fail + 1))
  fi
  if ! harness_census; then
    echo "FAIL: an assertion ran at a line the assertion audit never examined"
    fail=$((fail + 1))
  fi
  if ! harness_closure_covers; then
    echo "FAIL: this suite loads code the assertion audit does not read"
    fail=$((fail + 1))
  fi
  echo
  echo "$pass passed, $fail failed"
  harness_summary_printed=true
  if [ "$fail" -gt 0 ]; then
    exit 1
  fi
}

# The names this file provides, as bash reports them, recorded last on purpose.
#
# Derived and not written down, which is the repair: the list that used to live
# in `harness_ready` and the one in tests/lib/audit-suites.sh were two
# hand-kept copies of this set, and a primitive added here was shadowable until
# somebody remembered to go and edit both. `require` and `witness_exit` both
# arrived that way. Nothing reaches this line if the file stopped short, so a
# truncated harness records nothing and `harness_ready` refuses — which is what
# "defined last on purpose" was always for, and it now has a caller.
#
# tests/test_harness.sh holds this set equal to what audit-suites.sh derives
# from this file's text, so the reader and the shell have to agree about what
# this file defines.
harness_provides="$(
  for harness_n in $(declare -F | awk '{ print $NF }'); do
    if [ "$(harness_definition_file "$harness_n")" -ef "$harness_lib_file" ] 2>/dev/null; then
      printf '%s\n' "$harness_n"
    fi
  done | sort -u | tr '\n' ' '
)"
unset harness_n
