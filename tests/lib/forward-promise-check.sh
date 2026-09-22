#!/usr/bin/env bash
# Read the repository and refuse a line that assigns work to the release being
# cut.
#
# # The class
#
# A sentence saying a gap is "tracked for", "deferred to" or "reserved for"
# release N was true when it was written and is false the moment N is the
# release shipping. It has now been found and repaired in five rounds — in
# README.md, in docs/profiler-spec.md, in eight Go comments and in a test's own
# comment — and every round found it somewhere the previous round's reader did
# not look. Those readers were rooted at `.`, `../docs` and `../README.md` from
# inside `profiler/`, so the instances in `tests/` and `skills/` survived; two
# of the eight lived there.
#
# So the denominator is `git ls-files`: every tracked file, not a list of roots.
#
# # How much was read, and how much was looked at
#
# Those are two questions and this used to answer only the first, by printing a
# `lines` figure that nothing compared against anything. A scan limit inserted
# below the counting rule took the reader past a live forward promise in
# README.md with `files` fully green and the reader exiting 0 — the exact
# regression that motivated the two-sided count in tests/lib/audit-suites.sh,
# walked here because the second side was never wired up.
#
# Three figures now, and each has an other side:
#
#   files     — every file opened. Other side: `git ls-files`, from the
#               repository root rather than the process's working directory, so
#               running this from a subdirectory cannot narrow both sides
#               together.
#   lines     — every record the reader saw, counted before any rule below can
#               skip one. Other side: `grep -ac ''` over the same files, which
#               is a different program counting the same thing. Not `wc -l`: a
#               file with no final newline has one more record than `wc -l`
#               reports, and two tracked files are like that.
#   examined  — every record that reached the shape tests. Other side: `lines`,
#               through the two deliberate skips, each of which counts itself.
#
# `examined + skipped == lines` is checked **here**, not only by the caller, so
# a limit inserted anywhere in the walk is refused by the reader itself. And
# because the skips are legitimate in exactly three files, the per-file skip
# counts are printed: tests/test_skill.sh holds the set of files with a nonzero
# skip against this script's own HISTORY_DOCS and EXEMPT_FILE, so a skip path
# that started swallowing some other file is a finding rather than an
# accounting that still adds up.
#
# # Why it reads a vocabulary and not the version string
#
# The version appears about seventy times in this tree and roughly sixty of
# those are correct: `0.5.0 did not add one`, `Until 0.5.0`, `every key 0.5.0
# adds`, `this release is 0.5.0`, `AdapterVersion is 0.5.0`, a section heading,
# a manifest value. A check on the string alone fires on all of them, and a
# check that fires on forty correct lines is a check somebody turns off.
#
# What makes a line an assignment is the verb beside the version, and this
# repository has used the same handful every time: tracked, deferred, carried,
# reserved, scheduled, planned, postponed; "is <version> work" or a bare "is
# <version>"; "will … in <version>"; "needs … in <version>"; "new surface for
# <version>". That set is short and closed, which is the property a denylist
# needs to be worth having: it is not the list of phrasings somebody might use,
# it is the list this repository has used, so a phrasing outside it is a new
# finding rather than something this was meant to catch and missed.
#
# # Two scopes, because history is legitimate in exactly two files
#
# CHANGELOG.md and RELEASE_NOTES.md say what was true at each release, and
# 0.4.3's own section carries a deferral phrase naming its successor because
# that is what 0.4.3 meant. Those sections are byte-identical to the tag and
# must stay so, so in those two files only the section for the release being cut
# is in scope. Every other file has no section structure and no reader who can
# tell which release a sentence was written for, so every line of it is in
# scope.
#
# # The boundary, stated rather than implied
#
# A line naming a version surface is exempt from the bare `is <version>` shape,
# because "AdapterVersion is 0.5.0" and "building the capability is 0.5.0" are
# the same three words doing opposite jobs. A forward promise written into a
# line that also mentions a version would get past this. Closing that needs the
# sentence parsed, which is not what this does; what it does close is the
# vocabulary that was live five times.
#
# Usage:
#   forward-promise-check.sh <version>                the tracked tree
#   forward-promise-check.sh --over <file> <version>  one file, every line
#
# Exit 0 when nothing is found, 1 on any finding, 2 on a usage error. Findings
# print as `path:line: [shape] text`; a file with a deliberate skip prints
# `skipped-in=<path> scope=<n> exempt=<n>`; and the last line is always
# `files=<N> lines=<L> examined=<E> skipped=<S>` so "nothing is wrong" can be
# told from "nothing was read" and from "nothing was looked at".

set -euo pipefail

mode=tree
over=""
if [ "${1:-}" = "--over" ]; then
  mode=file
  over="${2:-}"
  shift 2 || true
  [ -n "$over" ] && [ -f "$over" ] || {
    echo "forward-promise-check.sh: not a file: $over" >&2
    exit 2
  }
  # Made absolute before the working directory moves below, so a relative
  # --over path still names the file the caller meant.
  over="$(CDPATH= cd -P -- "$(dirname -- "$over")" && pwd -P)/$(basename -- "$over")"
fi

version="${1:-}"
if [ -z "$version" ]; then
  echo "usage: forward-promise-check.sh [--over <file>] <version>" >&2
  exit 2
fi

# The repository root, and the reader stands in it.
#
# Not decoration: `git ls-files` is relative to the process's working directory,
# and so is the other side tests/test_skill.sh derives from it. Run from a
# subdirectory, both sides narrowed together and stayed equal over a fraction of
# the tree — the same "denominator derived from the thing it checks" shape this
# file's own `lines` figure was missing, one level out. Standing in the root
# makes the walk the same walk from anywhere.
repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || repo_root=""
if [ -z "$repo_root" ] || [ ! -d "$repo_root" ]; then
  echo "forward-promise-check.sh: not inside a git work tree, so the tracked tree cannot be the denominator" >&2
  exit 2
fi
cd "$repo_root" || exit 2

# The two files where a forward promise may be history.
HISTORY_DOCS="CHANGELOG.md RELEASE_NOTES.md"

# The one exemption, and it is self-checking: if it stops matching anything the
# scan reports that as a finding of its own, so a stale exemption cannot sit
# here quietly widening the hole.
#
# profiler/profiler_test.go carries the refused sentence as a *negative
# control* — `retiredActivationDeferral` is the string a test asserts is gone
# from the tree, and the comment above it names which sentence that was. A
# check that fired on those two lines would refuse the guard standing against
# the thing it guards.
EXEMPT_FILE=profiler/profiler_test.go
EXEMPT_PATTERN='retiredActivationDeferral|telling its reader that reading the activation'

# scan <one-file-mode> <file>... — the reader itself.
#
# In one-file mode there is no section scoping and no exemption: that is how the
# controls drive each shape over a fixture of their own and see it refused.
scan() {
  local single="$1"
  shift
  awk -v version="$version" -v history_docs=" $HISTORY_DOCS " \
      -v exempt_file="$EXEMPT_FILE" -v exempt_pattern="$EXEMPT_PATTERN" \
      -v single="$single" '
    BEGIN {
      # The version as a regex: its dots are literal.
      v = version
      gsub(/\./, "\\.", v)
      files = 0
      lines = 0
      examined = 0
      skipped = 0
      found = 0
      exempt_seen = 0
    }

    # Every record, counted before any rule below can skip one, for the reason
    # tests/lib/audit-suites.sh counts its own: a reader that stopped is
    # indistinguishable from a clean tree unless the amount read is reported.
    { lines++ }

    FNR == 1 {
      files++
      sectioned = (single == "" && index(history_docs, " " FILENAME " ") > 0)
      in_section = !sectioned
    }

    # A heading is structure and not prose, so it is skipped — and the skip says
    # so. Every record below leaves this program through exactly one of three
    # counters, which is what makes "the whole of what was read was looked at"
    # an answerable question rather than a claim.
    sectioned && /^## / {
      probe = $0
      sub(/^## v?/, "", probe)
      sub(/[^0-9.].*$/, "", probe)
      in_section = (probe == version)
      skipped++
      scope_skips[FILENAME]++
      next
    }

    !in_section { skipped++; scope_skips[FILENAME]++; next }

    {
      if (single == "" && FILENAME == exempt_file && $0 ~ exempt_pattern) {
        exempt_seen++
        skipped++
        exempt_skips[FILENAME]++
        next
      }

      examined++

      shape = ""
      if ($0 ~ ("(tracked|deferred|carried|reserved|scheduled|planned|postponed)[ a-z]* (for|to|until|with it for) v?" v)) {
        shape = "deferral verb"
      } else if ($0 ~ ("is +`?v?" v "`?( work|\\.|,|$)") && $0 !~ /[Vv]ersion|release \*?is\*?/) {
        shape = "is <version>"
      } else if ($0 ~ ("will [a-z]+ [a-z ]*in v?" v)) {
        shape = "will ... in <version>"
      } else if ($0 ~ ("needs? [a-z ]*in v?" v)) {
        shape = "needs ... in <version>"
      } else if ($0 ~ ("new surface (for|in) v?" v)) {
        shape = "new surface for <version>"
      }

      if (shape != "") {
        printf "%s:%d: [%s] %s\n", FILENAME, FNR, shape, substr($0, 1, 160)
        found++
      }
    }

    END {
      if (single == "" && exempt_seen == 0) {
        printf "%s: the exemption for the negative control matched nothing, so it is stale or the control is gone\n", exempt_file
        found++
      }
      # Refused here and not only by the caller, so a limit inserted anywhere
      # in the walk above is refused by the reader on its own terms. A scan that
      # read a record and neither looked at it nor said it was skipping it has
      # reported "none found" over a surface nobody knows the size of.
      if (examined + skipped != lines) {
        printf "the reader saw %d records, looked at %d and deliberately skipped %d: %d went past the walk unaccounted for\n", \
          lines, examined, skipped, lines - examined - skipped
        found++
      }
      for (f in scope_skips) skipped_files[f] = 1
      for (f in exempt_skips) skipped_files[f] = 1
      for (f in skipped_files) {
        printf "skipped-in=%s scope=%d exempt=%d\n", f, scope_skips[f] + 0, exempt_skips[f] + 0
      }
      printf "files=%d lines=%d examined=%d skipped=%d\n", files, lines, examined, skipped
      if (found > 0) exit 1
      exit 0
    }
  ' "$@"
}

if [ "$mode" = file ]; then
  status=0
  scan "$over" "$over" || status=$?
  exit "$status"
fi

# The tracked tree, read from git rather than from a glob so nothing tracked is
# outside the walk. Collected into an array rather than piped through `xargs`,
# because `xargs` may split a long list across several awk processes and every
# count would restart.
tracked=()
while IFS= read -r -d '' path; do
  tracked+=("$path")
done < <(git ls-files -z)

if [ "${#tracked[@]}" -eq 0 ]; then
  echo "forward-promise-check.sh: git ls-files named no file, so nothing was read" >&2
  exit 2
fi

status=0
scan "" "${tracked[@]}" || status=$?
exit "$status"
