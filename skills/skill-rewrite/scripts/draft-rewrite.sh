#!/usr/bin/env bash
# Draft a rewrite of a skill: run the skill-audit checks over it, and compose
# their output with the structural templates the audit says are missing.
# Writes REWRITE-DRAFT.md into the target skill directory — or wherever -o names
# — and names it on stdout. The draft is a skeleton plus the audit output for a
# reader to work from; it does not rewrite the skill and does not touch its
# SKILL.md.
# Exit codes: 0=draft written, 1=usage or target error, 3=execution error.
#
# A usage or target error is a missing or unknown option, an option given
# without its value, a target directory or SKILL.md that is not there, a
# report named with -a that is not there, or a destination named with -o that
# this script will not write to. Every one of them is reported on stderr,
# naming the option or the path, and leaves no draft behind.
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
output_arg=""
output=""
output_refusal=""
draft_incomplete=false

# The temporary audit this script writes when it was not handed one. It is
# removed on every exit path rather than only the successful one: the checks
# below can now stop the draft, and a new exit path that leaked a file each time
# would be this change's own doing.
#
# The draft itself is the trap's too, from the moment it is opened until the
# moment it is finished. "No draft was written" is what this script says on
# every path that does not reach the end, and it was not true: composing the
# draft is eight writes, any one of which can fail, and `-a` pointing at a file
# that exists and cannot be read put a six-line stub in the caller's skill
# directory at exit 1 — a status this contract spends on the caller's own
# mistake. No amount of checking before the first byte fixes that, because the
# writes come after it. So the sentence is made true the only way it can be:
# an incomplete draft is not left behind.
cleanup() {
  [[ -n "$own_audit_report" ]] && rm -f "$own_audit_report"
  [[ "$draft_incomplete" == true ]] && rm -f "$output"
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
    'Usage: draft-rewrite.sh -t <target-skill-dir> [-a <audit-report-path>] [-o <output-path>]' \
    '' \
    'Options:' \
    '  -t, --target    Target skill directory to rewrite (required)' \
    '  -a, --audit     Path to an existing skill-audit report (optional)' \
    '  -o, --output    Where to write the draft (default: <target-skill-dir>/REWRITE-DRAFT.md)' \
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
    -o|--output)
      require_value "$1" "$#"; output_arg="$2"; shift 2 ;;
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

# --- where the draft is written -----------------------------------------------
#
# With no -o the destination is what it has always been: beside the target's own
# SKILL.md. `-t` is the bound on that — the caller named a directory this script
# has already checked exists and holds a SKILL.md — so the worst it can
# overwrite is a previous draft of its own.
#
# `-o` removes that bound, which is the point of it, so the bound is restated
# rather than dropped. **It is stated narrowly: -o may write anywhere the caller
# can write, except three destinations where a Markdown draft is not a new file
# but a destruction or an impersonation.** A rule that confined -o to the target
# directory would be -o not existing; the answer to "the caller owns the
# destination" is that they mostly do, and here is the short list of where they
# do not.
#
#   1. A directory, or a symbolic link. The first would fail the redirect and
#      report "could not open" for a caller whose actual mistake was naming a
#      directory. The second is a destination whose name and whose landing place
#      are two different things: `: >` follows the link and truncates whatever
#      is on the other end, so a link is refused and the caller is told to name
#      the file it points at.
#   2. A SKILL.md. This skill's own Constraints say "do not overwrite the
#      original SKILL.md without explicit approval", and that had no mechanism
#      for as long as the destination was hardcoded — there was no way to reach
#      it. -o is what makes it reachable, so -o is where it becomes enforceable.
#      A rewrite draft is not a skill: written over a SKILL.md it destroys the
#      document it was built by reading, and leaves a draft where an agent will
#      next read a skill.
#   3. Anywhere inside a live agent configuration directory. This is where this
#      repository has already done damage — a documented install block that
#      `rm -rf`'d a live skills directory — and the harm is worse than a lost
#      file: an agent reads `~/.claude/skills/` as skills, so a draft dropped in
#      it is a document that may be loaded as instructions.
#
# **The third of those is decided by identity and never by any name**, for the
# reasons profiler/internal/homesafe sets out for the same decision on the Go
# side. Three spellings got past it before it was, each defeating the repair
# before:
#
#   - A symlink into a protected directory is outside by every comparison of
#     *unresolved* names and inside in fact. `$HOME/drafts/x.md` is a path whose
#     spelling says nothing and whose resolution says everything — and a sibling
#     sharing a name prefix is the reverse, `$HOME/.claude-notes` having
#     `$HOME/.claude` as a string prefix while being nowhere inside it.
#   - A `..` folded textually is folded against the *link's own name* rather
#     than against its target, so resolution has to happen during the walk and
#     not before or after it. `$HOME/aside/../x.md`, with `aside` a link into
#     `$HOME/.claude/skills`, folded to `$HOME/x.md` — outside every protected
#     root — while the redirect went through the kernel, which resolves `aside`
#     first, and wrote the draft into `$HOME/.claude`.
#   - And a comparison of *resolved* names is wrong too, which is the half that
#     is easy to miss because the fix for the first two looks complete. This
#     volume is case-insensitive and APFS is normalisation-insensitive, so
#     `.CLAUDE` and an NFD spelling of an NFC home are other names for the same
#     directory; bash's builtin `pwd -P` returns the caller's spelling rather
#     than the kernel's stored name, so a resolved path is only as canonical as
#     the way it was asked for; and a resolved path carried back through `$( )`
#     loses its trailing newlines, so the destination that was *decided* and the
#     destination that was *written* were two different entries. All six
#     protected roots were writable by spelling their names in capitals.
#
# So the decision is `path_target_is_inside`, which compares device and inode
# and never a path. See the barrier block below.
#
# And the other property from the same place: **a path whose landing place
# cannot be determined counts as protected.** "I could not tell where this would
# land" must not read as "go ahead", so it is not exit 1 — the caller's command
# line is not what was wrong — but cannot_compute and exit 3. What counts as
# undetermined is narrow, and it used to be far too wide: while resolution ended
# in one `cd -P` over the deepest existing ancestor, every destination that
# already existed as a *regular file* came back unresolvable, because you cannot
# `cd` into a file. That was an undocumented fifth refusal, it made `-o` unable
# to re-run over its own output, and it stood in front of the leaf-symlink rule
# so that rule never ran at all.
#
# The decision lives here rather than in verdict-guard.sh because the guard's
# primitives are about computing a verdict over a skill, and this is about where
# this script writes. This is the only script in either skill with a destination
# to decide: the sibling checks read a directory and print to stdout. If a
# second one ever takes a destination, this moves to the guard rather than being
# copied.
#
# It is, however, the same text as the copy in tests/test_install.sh, and that
# suite asserts that the two are byte-identical. The previous round said in prose
# that the copies were one walk and they were not: they disagreed about a leaf
# symlink, about an empty path, and about a root that resolves to `/`. Whether
# the shared walk should move into verdict-guard.sh or into tests/lib/ — neither
# of which both callers can reach today — is recorded for 0.6.0.

# --- BEGIN THE PATH CONTAINMENT BARRIER --------------------------------------
#
# Byte-identical in skills/skill-rewrite/scripts/draft-rewrite.sh and in
# tests/test_install.sh, between these two markers, and tests/test_install.sh
# asserts that it is byte-identical rather than trusting that it stayed so. The
# previous round's claim that "all three are now the same walk" was false in
# three places — the leaf symlink, the empty path, and a root that resolves to
# `/` — and every one of them was a divergence between two copies nobody could
# diff. Prose cannot hold two copies together; a diff can.

# __path_link_target <path> — the bytes a symlink holds, exactly, in
# __path_target.
#
# Two separate things would lose those bytes, and both are the defect this whole
# block replaces, one level down: a target decided on one spelling while the
# kernel follows another.
#
# Command substitution strips every trailing newline, so `$(readlink …)` on a
# target ending in one hands back a name that is a different entry. The
# `printf X` carries them through: the last character of the captured output is
# an X, so there is nothing trailing for the substitution to remove, and `%X`
# takes the guard off again.
#
# And `readlink`'s own terminator is not portable. GNU readlink writes the
# target followed by a newline; the BSD readlink on macOS writes the target and
# nothing else — measured, not assumed. So stripping "one trailing character"
# is right on one platform and eats a byte of the target on the other, which is
# how this function was first written and what the generated bound caught. `-n`
# is the flag both of them have for "no terminator", so with it there is nothing
# to strip and no platform to be right about.
__path_link_target() {
  local raw
  raw="$(readlink -n -- "$1"; printf X)" || return 1
  __path_target="${raw%X}"
  return 0
}

# __path_max_links bounds a symlink chain so that a cycle terminates. It is not
# the kernel's limit and does not claim to be — macOS refuses a chain at 32 and
# Linux at 40 — and it does not need to be: a chain the kernel refuses produces
# no write at all, so such a path has no object for a verdict to be about, and
# refusing is the safe answer for it. What the counter is actually for is a
# cycle, where `readlink` succeeds for ever.
__path_max_links=32

# __path_max_climb bounds the climb from a directory to the filesystem root.
# The climb already stops by identity, because `..` at the root is the root;
# this is the second stop, for a filesystem on which it is not.
__path_max_climb=1024

# __path_walk <spelled> — chdir to the deepest directory the kernel would reach
# while resolving <spelled>, and set __path_tail to the components after it,
# each preceded by `/`, or to the empty string.
#
# # Why it moves instead of building a string
#
# Each of the three copies of this barrier used to build a *name* for the
# destination and compare it against a name built for the root. A name is not an
# identity, and on this platform three separate mechanisms make it not one. The
# volume is case-insensitive, so `.CLAUDE` and `.claude` are one directory with
# two names. APFS is normalisation-insensitive, so an NFC and an NFD spelling
# are one directory with two names. And bash's *builtin* `pwd -P` hands back the
# caller's own spelling rather than the kernel's stored name, so even a fully
# resolved path is only as canonical as the way it was asked for. All three were
# live escapes: every one of the six protected roots could be written into by
# spelling its name in capitals, and the exact spelling was refused while the
# capitalised one was not.
#
# So nothing here is decided by comparing paths. The walk performs the kernel's
# own resolution as a sequence of chdirs, which cannot be spelled around because
# it is not reading a spelling — it is moving. What comes out is the process's
# working directory, and the comparison in path_target_is_inside is `-ef`:
# device and inode, the one handle no spelling can change.
#
# # The rule, once, for every component
#
# A component that **is a directory** is entered, which follows a link, a chain
# of links, a relative target and the platform's own aliasing in a single step —
# on macOS a scratch directory arrives through /var and lives at /private/var,
# so a comparison of unresolved names answers "outside" for a path that is in
# fact inside.
#
# A component that **is a symlink** and is not a directory has its target walked
# in its place, exactly as the kernel splices it in, **including at the leaf**.
# `: > "$dest"` follows a leaf symlink, so a leaf link into a protected
# directory is a write into that directory; appending the leaf as named is how
# both shell copies of this came to disagree with the Go one, and it was a hole
# standing behind two accidents rather than behind a rule.
#
# A component that **does not exist** begins the tail. From there on there is
# nothing with an identity, which is why the tail is the only thing this barrier
# ever compares as text — and it compares it byte for byte or not at all, for
# the reason path_target_is_inside sets out.
#
# A component that exists, is not a directory and is not the last has **no
# answer**: the kernel answers ENOTDIR, so no write happens, and no answer is a
# refusal rather than a guess. So is a directory that cannot be entered, and so
# is a chain that does not end.
#
# A `..` inside the tail folds against the tail and, past its start, climbs the
# resolved prefix — which is correct precisely because that prefix holds no link
# and no `..` any more. The kernel answers ENOENT for a `..` that follows a
# component which does not exist, so no write happens on such a path either way;
# folding is the conservative reading of it, and it keeps `a/b/../c` meaning
# `a/c` when `a` exists and `b` does not.
#
# An empty path is not a path and has no answer. It used to return the working
# directory at status 0 in both shell copies while the Go one refused, which is
# the sort of divergence two copies held together by prose produce.
__path_walk() {
  local spelled remaining name links
  spelled="$1"
  __path_tail=""
  links=0
  [ -n "$spelled" ] || return 1
  case "$spelled" in
    /*) cd -P -- / 2>/dev/null || return 1 ;;
  esac
  remaining="$spelled"
  while [ -n "$remaining" ]; do
    case "$remaining" in
      */*) name="${remaining%%/*}"; remaining="${remaining#*/}" ;;
      *)   name="$remaining"; remaining="" ;;
    esac
    if [ -z "$name" ] || [ "$name" = "." ]; then
      continue
    fi
    if [ -n "$__path_tail" ]; then
      if [ "$name" = ".." ]; then
        __path_tail="${__path_tail%/*}"
      else
        __path_tail="$__path_tail/$name"
      fi
      continue
    fi
    if [ "$name" = ".." ]; then
      cd -P -- .. 2>/dev/null || return 1
      continue
    fi
    if [ -d "$name" ]; then
      cd -P -- "$name" 2>/dev/null || return 1
      continue
    fi
    if [ -L "$name" ]; then
      links=$((links + 1))
      [ "$links" -le "$__path_max_links" ] || return 1
      __path_link_target "$name" || return 1
      [ -n "$__path_target" ] || return 1
      case "$__path_target" in
        /*) cd -P -- / 2>/dev/null || return 1 ;;
      esac
      if [ -n "$remaining" ]; then
        remaining="$__path_target/$remaining"
      else
        remaining="$__path_target"
      fi
      continue
    fi
    if [ -e "$name" ] && [ -n "${remaining//\//}" ]; then
      return 1
    fi
    __path_tail="/$name"
  done
  return 0
}

# path_target_is_resolvable <spelled> — there is an answer to where a write on
# <spelled> would land. 0 there is, 1 there is not.
path_target_is_resolvable() {
  ( CDPATH= ; __path_walk "$1" ) >/dev/null 2>&1
}

# path_target_is_inside <spelled> <root> <mode> — does a write on <spelled>
# create-or-truncate an object that is <root>, or one reachable from <root>
# without leaving it?
#
# `or-equal` counts <root> itself as inside; `strictly` does not.
#
# 0 inside, 1 outside, 3 there is no answer — which every caller turns into a
# refusal and never into a pass.
#
# The third status is 3 and not 2 because 3 is the status the drafter's own
# header registers for "no verdict was reached", and this is the same statement
# one layer down: the two numbers would have meant one thing, and a second
# number for it is a second thing to keep in agreement. It also keeps the static
# check in tests/test_rewrite.sh honest — it reads this file for `exit N` and
# cannot see that these are a subshell's exits, so every number spelled here has
# to be a number the script is allowed to exit with, which is a good discipline
# for the block rather than a concession to the checker.
#
# The verdict is decided by identity for every component that exists, and by
# name only for components that do not exist yet. That is the whole invariant,
# and it reduces to a single question: the object a write creates or truncates
# lives in the deepest directory the kernel reaches while resolving the
# spelling, so the verdict is whether *that directory* is the root or is under
# it — asked with `-ef`, which compares device and inode and therefore gives
# the same answer however either side is spelled, and asked by climbing with
# `cd -P -- ..`, which is the kernel answering "reachable without leaving it"
# rather than this file computing it. The tail is compared as text only when the
# root itself does not exist yet, where there is no identity on either side and
# the answer may be that there is no answer.
#
# The whole of it happens in a subshell, so the chdirs are the subshell's and
# the caller's working directory is untouched; and no path is ever carried back
# out through a command substitution, which is the mechanism that made the
# previous round's barrier decide on one path and write on another.
path_target_is_inside() {
  (
    CDPATH=
    local spelled root mode here root_dir root_tail dest_tail climb dest_head dest_rest root_rest
    spelled="$1"
    root="$2"
    mode="$3"
    here="$PWD"
    __path_walk "$root" 2>/dev/null || exit 3
    root_dir="$PWD"
    root_tail="$__path_tail"
    cd -P -- "$here" 2>/dev/null || exit 3
    __path_walk "$spelled" 2>/dev/null || exit 3
    dest_tail="$__path_tail"

    if [ -n "$root_tail" ]; then
      # The root does not exist yet. Nothing can exist below a directory that
      # does not exist, so the destination's own deepest directory has to be
      # the same object as the root's, and what is left over is the name
      # comparison the invariant allows for names that are not yet anything.
      [ . -ef "$root_dir" ] || exit 1
      # Three answers and not two, which is the whole of what makes this branch
      # honest.
      #
      # **The same bytes** are the same name on every filesystem, so a byte
      # match is an answer anywhere: inside.
      #
      # **Names that could not be one name however the volume compares them**
      # are an answer too: outside. This is the common case by far and it has to
      # stay cheap and definite — on a machine where `~/.devin` does not exist,
      # every `-o` the drafter is ever given reaches this line against it, and
      # answering "I cannot tell" there would turn the whole flag into exit 3.
      # Two names are definitely different when both are ASCII and they differ
      # by more than case, because ASCII case is the only folding a filesystem
      # applies to an ASCII name.
      #
      # **Anything left is undecidable, and it says so.** Two ASCII names that
      # differ only in case are one directory on this volume and two on a
      # case-sensitive one; two names either of which carries a byte at or above
      # 0x80 may be an NFC and an NFD spelling of one name, and normalising
      # Unicode needs tables this file has no business carrying. The temptation
      # is to guess, and the guess was written twice before this comment was. It
      # cannot be right: "this name may be the root's name" is a *refusal* for a
      # fence that protects the root and a *pass* for a fence that keeps writes
      # inside it, and this one walk serves one of each — the drafter refuses a
      # destination inside a protected directory, while tests/test_install.sh
      # runs a documented block only if it stays inside a scratch root. No guess
      # is fail-closed for both. `exit 3` is, because no answer is a refusal in
      # both.
      #
      # `LC_ALL=C` is what makes the byte range below a range of bytes, and it
      # is also what keeps `nocasematch` to ASCII case, which is the only case
      # this comparison claims to know about.
      if [ "$dest_tail" = "$root_tail" ]; then
        [ "$mode" = or-equal ] || exit 1
        exit 0
      fi
      # The destination's tail down to the root's own depth, taken a component
      # at a time so that the comparison below is the same comparison the
      # kernel would make component by component.
      dest_head=""
      dest_rest="$dest_tail"
      root_rest="$root_tail"
      while [ -n "$root_rest" ]; do
        [ -n "$dest_rest" ] || exit 1
        root_rest="${root_rest#/}"
        case "$root_rest" in
          */*) root_rest="/${root_rest#*/}" ;;
          *)   root_rest="" ;;
        esac
        dest_rest="${dest_rest#/}"
        case "$dest_rest" in
          */*) dest_head="$dest_head/${dest_rest%%/*}"; dest_rest="/${dest_rest#*/}" ;;
          *)   dest_head="$dest_head/$dest_rest"; dest_rest="" ;;
        esac
      done
      if [ "$dest_head" = "$root_tail" ]; then
        exit 0
      fi
      (
        LC_ALL=C
        case "$dest_head$root_tail" in
          *[$'\200'-$'\377']*) exit 3 ;;
        esac
        shopt -s nocasematch
        if [[ "$dest_head" == "$root_tail" ]]; then
          exit 3
        fi
        exit 1
      )
      exit $?
    fi

    # The root exists, so the question is pure identity: climb from the
    # destination's own directory towards the filesystem root, and the
    # destination is inside iff the root is one of the directories passed on
    # the way. A root that resolves to `/` therefore contains everything, which
    # is the true answer for it and not the fail-open half of the two opposite
    # answers the two copies of this used to give.
    climb=0
    while :; do
      if [ . -ef "$root_dir" ]; then
        if [ "$climb" -eq 0 ] && [ -z "$dest_tail" ] && [ "$mode" != or-equal ]; then
          exit 1
        fi
        exit 0
      fi
      [ . -ef / ] && exit 1
      cd -P -- .. 2>/dev/null || exit 1
      climb=$((climb + 1))
      [ "$climb" -le "$__path_max_climb" ] || exit 3
    done
  )
}
# --- END THE PATH CONTAINMENT BARRIER ----------------------------------------

# The live agent configuration directories, relative to the caller's own home.
#
# `$HOME`-relative is what makes this decision askable by a test without
# pointing one at the developer's own `~/.claude` — tests/test_rewrite.sh
# redirects HOME into the harness scratch directory — and it is also the only
# correct anchor: these are per-user directories.
#
# An unset or empty HOME means there is no home and therefore no live
# configuration directory for a destination to be inside. That is a different
# statement from "a path that could not be resolved", which is why it accepts
# rather than refusing: there is nothing here that could not be determined.
#
# One physical line per root, and the list is compared against two documents:
# the one skills/skill-rewrite/SKILL.md gives a reader, and every `$HOME`-relative
# skills path the README spells. Neither side can grow without the others.
#
# `.agents` is not any one harness's configuration directory — it is the shared
# skills directory Codex and Cursor both read, which the README documents at
# `~/.agents/skills/` for each of them. A list assembled from harness names
# missed it for that reason, and the miss was the whole list failing its own
# stated rationale: the refusal is about what an agent reads as instructions,
# not about whose dot-directory it is. Reproduced before it was added — rc=0,
# draft written into `$HOME/.agents/skills/`.
protected_home_dirs=".claude .cursor .codex .devin .config .agents"

# output_is_permitted <spelled> — decide the destination named with -o.
#
# Sets output_refusal to the reason when it is refused, and returns 1. It hands
# the reason back in a variable rather than on stdout because a destination it
# *cannot decide* is cannot_compute's to report, and an `exit 3` inside a command
# substitution exits the subshell: the script would carry on and write the
# draft. The same reason skill_body answers in `$section`.
#
# Every question below is asked about `$spelled` — the caller's own bytes, the
# ones the redirect at the bottom of this script will be performed on. Not about
# a resolved copy of them, and that is the whole of the repair rather than a
# detail of it: the previous version resolved the destination into a string,
# carried that string back through `$( )`, and asked its questions about *that*.
# Command substitution strips trailing newlines, so for a destination whose name
# ends in one the questions were answered about a different entry from the one
# the write went to — a leaf link named `notes.md\n` pointing into
# `~/.claude/skills` was accepted at exit 0 and the draft landed there. There is
# no resolved path in this function now, so there is nothing for a substitution
# to truncate. `-L` and `-d` on the spelling are lstat and stat on exactly those
# bytes; the containment question is answered by identity inside
# path_target_is_inside and the answer it hands back is a status, not a path.
#
# The leaf name is taken from the spelling for the same reason, and it is the
# spelling's own last component because a leaf symlink is already refused above
# it: with no link at the leaf, the name the kernel writes under is the name the
# caller wrote. A spelling that ends in `/`, `.` or `..` names a directory, which
# the rule above this one refuses when it exists and the redirect refuses when it
# does not.
output_is_permitted() {
  local spelled
  local leaf
  local root
  local verdict
  spelled="$1"
  output_refusal=""

  if [[ -z "$spelled" ]]; then
    output_refusal="the destination given with -o is empty"
    return 1
  fi

  path_target_is_resolvable "$spelled" \
    || cannot_compute DEP002 "the destination $spelled could not be resolved, so where the draft would be written is unknown; no draft was written" false

  if [[ -L "$spelled" ]]; then
    output_refusal="$spelled is a symbolic link, so where the draft would be written is not where it is named; give the path it points at"
    return 1
  fi
  if [[ -d "$spelled" ]]; then
    output_refusal="$spelled is a directory, and the draft is a file; name the file to write"
    return 1
  fi
  leaf="$spelled"
  while [[ "$leaf" == */ ]]; do
    leaf="${leaf%/}"
  done
  leaf="${leaf##*/}"
  if [[ "$leaf" == SKILL.md ]]; then
    output_refusal="$spelled names a SKILL.md: a rewrite draft is not a skill, and this script does not overwrite one"
    return 1
  fi

  if [[ -n "${HOME:-}" ]]; then
    for root in $protected_home_dirs; do
      # `|| verdict=$?` and not a bare call followed by `case $?`: errexit fires
      # on a simple command whose status is not being tested, so a bare call
      # here made the *script* exit 2 for every destination the barrier could
      # not decide — a status its own header does not register, reported as a
      # crash rather than as the refusal it is. Caught by the registry
      # assertion in tests/test_rewrite.sh.
      verdict=0
      path_target_is_inside "$spelled" "$HOME/$root" or-equal || verdict=$?
      case "$verdict" in
        0)
          output_refusal="a write on $spelled lands in the live configuration directory $HOME/$root, whatever it is spelled; the draft would be read as configuration"
          return 1
          ;;
        1) ;;
        *)
          cannot_compute DEP002 "the destination $spelled cannot be shown to be outside the live configuration directory $HOME/$root, so where the draft would be written is unknown; no draft was written" false
          ;;
      esac
    done
  fi

  return 0
}

if [[ -n "$output_arg" ]]; then
  # The refusal refuses. It is decided here, with the other inputs and before
  # the audit runs, so there is nothing to undo — and it is `exit 1` rather than
  # a diagnostic followed by the write, which is the defect tests/test_install.sh
  # carried: a containment check that computed "no", printed FAIL, and ran the
  # install on the next line. A reported failure is not a refusal.
  output_is_permitted "$output_arg" \
    || { echo "Refusing to write the draft: $output_refusal" >&2; exit 1; }
  output="$output_arg"
else
  output="$target_skill/REWRITE-DRAFT.md"
fi

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
# file the trap above has already removed by the time anyone reads the draft,
# and it never meant anything outside the process that wrote it. So the two cases state what they actually are: a
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
# Read before the draft file is opened, with everything else this script reads.
# That ordering is worth keeping and it was never the whole of the rule, which
# is where the claim went wrong: composing the draft is itself eight writes, so
# "everything that can stop this script stops it before the first byte" cannot
# be made true by moving reads earlier. What can be made true, and is, is the
# sentence those words were standing in for — "no draft was written" — and the
# trap above is what makes it true, on every path including the ones that fail
# in the middle of writing.
skill_body "$target_skill/SKILL.md" false
target_body="$section"

# And the audit itself, which is the other thing the draft is composed from.
# It used to be read with `cat "$audit_report" >> "$output"`, three writes into
# the document — so `-a` naming a file that exists and cannot be read passed
# the `-f` test above, opened the draft, wrote its header, and died inside cat
# with errexit spending exit 1 and the caller left holding six lines. Read here
# it is one more thing that stops this script before it starts writing, and
# cat's status is read where cat is called, with the sentence naming cat.
audit_status=0
audit_text="$(cat -- "$audit_report")" || audit_status=$?
[[ $audit_status -eq 0 ]] \
  || cannot_compute DEP002 "cat could not read the audit report at $audit_report (status $audit_status); no draft was written" false

# draft_append — append this function's stdin to the draft, reading cat's status
# where cat is called.
#
# `require_tool cat` above says cat answered one question at one moment. It says
# nothing about these seven writes, and unread their status was this script's:
# under errexit a cat that failed on the third one exited this script with cat's
# own status, having already written the first two. The status is read here once
# rather than at seven call sites, where it would be forgotten at one of them.
draft_append() {
  local status=0
  cat >> "$output" || status=$?
  [[ $status -eq 0 ]] \
    || cannot_compute DEP002 "cat could not write the draft to $output (status $status); no draft was written" false
  return 0
}

# From here on there is a partial document on disk, and the trap owns it.
: > "$output" \
  || cannot_compute DEP002 "could not open $output for writing; no draft was written" false
draft_incomplete=true

draft_append <<EOF
# Rewrite draft: $skill_name

Generated from: $provenance

## Current state

EOF

[[ -z "$audit_text" ]] || draft_append <<< "$audit_text"

draft_append <<'EOF'

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
  draft_append <<'EOF'
### When to use

Use this skill when:

- <specific scenario 1>
- <specific scenario 2>
- <specific scenario 3>

Do not use this skill when <negative scope>.

EOF
fi

if ! text_matches false true "^#{2,6}[[:space:]]+Examples?[[:space:]]*$" "$target_body"; then
  draft_append <<'EOF'
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
  draft_append <<'EOF'
### Validation checklist

- [ ] <observable pass/fail criterion 1>
- [ ] <observable pass/fail criterion 2>
- [ ] <observable pass/fail criterion 3>

EOF
fi

draft_append <<'EOF'

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

draft_incomplete=false
echo "Rewrite draft written to: $output"
