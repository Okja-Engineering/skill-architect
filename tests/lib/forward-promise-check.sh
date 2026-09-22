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
# The version string occurs far more often in this tree than there are promises
# — every occurrence of it is correct at the moment of writing, and the
# overwhelming majority stay correct: `0.5.0 did not add one`, `Until 0.5.0`,
# `every key 0.5.0 adds`, `this release is 0.5.0`, `AdapterVersion is 0.5.0`, a
# section heading, a manifest value. A check on the string alone fires on all
# of them, and a check that fires on dozens of correct lines is a check
# somebody turns off. No count is written here on purpose: the last one this
# comment carried was wrong by about forty per cent, and a numeral in a comment
# nothing re-derives goes stale by next release. The argument does not need it.
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
# # A vocabulary is a set of verbs, not a set of spellings
#
# The distinction is the whole reason this reader was wrong once already. The
# verb set above was closed and correct, and the matcher still walked past any
# one of those verbs capitalised and standing beside the version — because it
# tested the raw record, so every alternative in the set was implicitly a
# lowercase *spelling* rather than a verb. A capitalised member of a closed set
# is inside the set; missing it is not a gap in the vocabulary, it is the
# matcher asking the wrong question.
#
# That shape is described here and deliberately not written out, because the
# reader below reads this file too and would refuse the example. Seeing it
# refuse its own documentation is the check working, not a false positive; the
# controls in tests/test_skill.sh hold the literal spellings instead, over
# fixtures, where they belong.
#
# So the record is case-folded once, at the single point it reaches the shape
# tests, and the tests are unanchored. That makes three things fall out
# together instead of needing a rule each: capitalisation, ALL CAPS and mixed
# case are one `tolower`; and a leading `-`, `*`, `**`, indentation, a colon or
# an em dash before the verb are positions the tests never cared about. The
# alternative — adding `Deferred|DEFERRED|…` to the list — is the move that has
# now cost this repository five rounds, in the guard-primitive reader
# (`name ( ) {`), the shadowing check (`assert ( ) {`), the trap check
# (`trap cleanup 0`), the suite audit (the libraries the suites `source`), and
# here.
#
# And the record is read joined to the record before it, because the unit of
# meaning is a sentence while the unit of the walk is a line. A promise whose
# verb and version fall either side of a wrap is still a promise; this
# release's own changelog copy of that sentence escaped only because the break
# landed between them, which made a prose reflow a latent CI failure on a file
# nobody had edited. Such a finding prints with `across a line break` in its
# shape so the reader can see why the named line looks innocent on its own.
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
# vocabulary that was live five times — now in any case, in any position in the
# record, and with the verb and the version either side of a line wrap.
#
# Two things this still does not do, stated so the next round does not have to
# find them. The window is two records, so a promise spread over three lines
# with neither the verb nor the version adjacent to the join is not seen. And
# the section-heading probe reads a literal `## v?<version>`, so the two
# history documents are scoped by an exact heading match; a heading written
# some other way would take that file's whole section out of scope rather than
# report anything, which is why tests/test_skill.sh holds the per-file skip
# counts against this script's own HISTORY_DOCS rather than trusting the
# accounting to add up.
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

    # The shape tests, in one place, over a record the caller has already
    # case-folded.
    #
    # Case is normalised away rather than enumerated. `Deferred`, `DEFERRED`
    # and `DeFeRrEd` are the same closed verb set arriving in a different
    # spelling, not members the set was missing, so the repair is `tolower` at
    # the one place the record reaches the tests — not seven more alternatives
    # in the vocabulary. Listing spellings is what left this reader blind to
    # the capitalised half, and listing spellings is what left the guard-
    # primitive reader blind to `name ( ) {`, the shadowing check blind to
    # `assert ( ) {`, and the trap check blind to `trap cleanup 0`.
    #
    # Nothing here is anchored, and that is load-bearing rather than
    # incidental: a leading list marker, a bold or emphasis lead-in,
    # indentation, a colon or an em dash in front of the verb are *positions*,
    # and a position is not a shape. `- **Deferred to <version>**:` needs no
    # rule of its own once the verb is found wherever it sits in the record.
    function shape_of(rec) {
      if (rec ~ ("(tracked|deferred|carried|reserved|scheduled|planned|postponed)[ a-z]* (for|to|until|with it for) v?" v))
        return "deferral verb"
      if (rec ~ ("is +`?v?" v "`?( work|\\.|,|$)") && rec !~ /version|release \*?is\*?/)
        return "is <version>"
      if (rec ~ ("will [a-z]+ [a-z ]*in v?" v))
        return "will ... in <version>"
      if (rec ~ ("needs? [a-z ]*in v?" v))
        return "needs ... in <version>"
      if (rec ~ ("new surface (for|in) v?" v))
        return "new surface for <version>"
      return ""
    }

    # Every record, counted before any rule below can skip one, for the reason
    # tests/lib/audit-suites.sh counts its own: a reader that stopped is
    # indistinguishable from a clean tree unless the amount read is reported.
    { lines++ }

    FNR == 1 {
      files++
      sectioned = (single == "" && index(history_docs, " " FILENAME " ") > 0)
      in_section = !sectioned
      # A file boundary is never a line wrap, so the window starts empty.
      prev = ""
      prev_raw = ""
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
      prev = ""
      prev_raw = ""
      next
    }

    !in_section { skipped++; scope_skips[FILENAME]++; prev = ""; prev_raw = ""; next }

    {
      if (single == "" && FILENAME == exempt_file && $0 ~ exempt_pattern) {
        exempt_seen++
        skipped++
        exempt_skips[FILENAME]++
        prev = ""
        prev_raw = ""
        next
      }

      examined++

      rec = tolower($0)
      shape = shape_of(rec)
      text = $0

      # Two records, because the unit of meaning is a sentence and the unit of
      # the walk is a line.
      #
      # This is not hypothetical. The changelog copy of the sentence describing
      # this very finding survived only because a line break happened to fall
      # between the verb and the version; a reflow moving that break by one
      # word would have reddened CI on a file nobody edited. A promise is a
      # promise whichever side of the wrap the version lands on, so the reader
      # looks at the record joined to the one before it — the same join a
      # reflow would perform.
      #
      # No apostrophe appears in any comment in this awk program, and that is
      # deliberate rather than styleless: the program is a single-quoted shell
      # word, so one apostrophe here ends it and bash reports a syntax error
      # somewhere below. It fails loudly and closed, but it fails.
      #
      # `prev` is the previous record *this file examined*, reset at a file
      # boundary and at every skip, so the window never spans a section heading
      # or the exempt control: those two lines were never one sentence.
      #
      # Reported only when the match genuinely crosses the boundary. If `prev`
      # matched on its own it was already reported at its own line, so
      # requiring `shape_of(prev) == ""` is what keeps one promise from being
      # counted twice. Sentence-ending punctuation blocks the join on its own,
      # because no alternative in the vocabulary admits `.` between the verb
      # and its preposition.
      if (shape == "" && prev != "" && shape_of(prev) == "") {
        shape = shape_of(prev " " rec)
        if (shape != "") {
          shape = shape " across a line break"
          text = prev_raw " " $0
        }
      }

      if (shape != "") {
        printf "%s:%d: [%s] %s\n", FILENAME, FNR, shape, substr(text, 1, 160)
        found++
      }

      prev = rec
      prev_raw = $0
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
