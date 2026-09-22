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
# # How far a shape reaches, stated so nobody has to find out
#
# A shape is the span from the verb to the version, and the markup a writer
# puts *around* that span is a position: a list marker, indentation, a colon or
# an em dash in front, emphasis that opens before the verb and closes after the
# version. Those need no rule, which is what leaving the tests unanchored buys.
#
# Inside the span it is the other way round, and this is the honest boundary.
# The interiors are still written as letters and spaces, with one literal space
# in front of the version and the version itself bare, so a phrase is missed
# when anything interrupts it between the verb and the version: emphasis closed
# around the verb alone rather than around the whole phrase — the near
# neighbour of the worked example above, and the lead-in style this repository
# actually writes; the version set in inline code, or written as a link label,
# or in quotes; a comma, a colon, a bracketed aside or a digit between the verb
# and its preposition; and a tab or a doubled space where one space is wanted.
# Trailing punctuation after the version is fine, and so is a `v` in front of
# it: the defect is on the left of the version, not the right.
#
# This is pre-existing rather than opened or closed by any recent round, and it
# is recorded as 0.6.0 surface rather than repaired here, because widening the
# interiors is a false-positive budget and not a regex. This tree is
# documentation-heavy and both release documents describe this reader, so a
# wider matcher starts reading prose about the check as the check — measured,
# a crude widening of the last three shapes lands on a sentence in the release
# notes that is describing this reader, while the same treatment of the
# deferral shape alone adds nothing on this tree. Which of those is worth
# buying is a decision with evidence behind it, not a tightening to slip into a
# release gate. Nothing here claims this class is closed.
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
# A version surface is exempt from the bare `is <version>` shape, because
# "AdapterVersion is" and "building the capability is" the same version are the
# same three words doing opposite jobs. What the exemption asks is a relation
# and not a word count: the subject standing beside *that* `is` — through any
# markup — has to be the version surface, and every occurrence on the record is
# asked separately, so a record can carry an exempt clause and an assignment at
# once and still be refused.
#
# It was a substring over the whole record until this round, which is two holes
# rather than one. A record naming a version surface anywhere was exempt from
# every occurrence of the shape on it, however far away; and the discrimination
# that remained was carried entirely by capitalisation, so the moment the
# record was folded for the shape tests the exemption folded with it and five
# assignments this reader had been refusing stopped being refused. A substring
# was standing in for a relation, and the fold only made that louder. The
# controls for both halves are in tests/test_skill.sh, where the literals
# belong.
#
# What still gets past: a subject that genuinely is a version surface, in a
# sentence that is nonetheless assigning work. Separating those needs the
# sentence parsed, which is not what this does.
#
# The window is two records, and the honest way to say what that buys is that
# the verb and the version have to land on records next to each other. An
# earlier wording here said a promise spread over three lines goes unseen "with
# neither the verb nor the version adjacent to the join", which reads as though
# a verb sitting next to a break were enough; driven, it is not — no single
# join carries both, and the promise is unseen.
#
# The section-heading probe reads `## v?<version>`, in any case. It used to
# read the raw record, so a capital V meant no section at all. That is fixed
# and it is not the interesting half: any heading this probe does not
# recognise leaves the file scoped to a section that does not exist, which is
# not a narrower scope but no scope, and the END rule below refuses exactly
# that. Tolerating a list of heading spellings instead would leave the next
# one; what is asserted is that the section was entered.
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
    # The bare `is <version>` shape, asked once per occurrence rather than once
    # per record.
    #
    # `AdapterVersion is <version>` and "building the capability is" the same
    # version are the same three words doing opposite jobs, so this shape needs
    # an exemption. What makes the first one a version surface is the *subject*
    # standing beside that `is` — a position, like the lead-in markers above —
    # and not the word appearing somewhere in the record. Asking the record
    # carried two faults at once: a record naming a version surface anywhere
    # was exempt from every occurrence of the shape it carried, however far
    # away; and folding the record for the shape tests folded the record the
    # exemption read too, so the hole grew by every spelling of the word in one
    # stroke and five assignments this reader used to refuse stopped being
    # refused. A substring was standing in for a relation.
    #
    # So the occurrences are walked, each is asked about its own subject, and a
    # record may hold one of each — a version surface beside one `is` no longer
    # speaks for the next `is` down the line. The subject test folds with the
    # record like everything else here, so it stays a set of subjects rather
    # than becoming a set of spellings.
    function is_assignment(rec,   tail, subject) {
      tail = rec
      while (match(tail, "is +`?v?" v "`?( work|\\.|,|$)")) {
        subject = substr(tail, 1, RSTART - 1)
        if (subject !~ /(version|release)[^a-z0-9]* $/)
          return 1
        tail = substr(tail, RSTART + RLENGTH)
      }
      return 0
    }

    function shape_of(rec) {
      if (rec ~ ("(tracked|deferred|carried|reserved|scheduled|planned|postponed)[ a-z]* (for|to|until|with it for) v?" v))
        return "deferral verb"
      if (is_assignment(rec))
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
      # Which files are being scoped by a heading, so the END rule can ask
      # whether each of them was ever actually entered.
      if (sectioned) sectioned_files[FILENAME] = 1
      # A file boundary is never a line wrap, so the window starts empty.
      prev = ""
      prev_raw = ""
    }

    # A heading is structure and not prose, so it is skipped — and the skip says
    # so. Every record below leaves this program through exactly one of three
    # counters, which is what makes "the whole of what was read was looked at"
    # an answerable question rather than a claim.
    sectioned && /^## / {
      # Folded like every other test here, because the probe read the raw
      # record and a capital V in the heading was enough to mean no section.
      probe = tolower($0)
      sub(/^## v?/, "", probe)
      sub(/[^0-9.].*$/, "", probe)
      in_section = (probe == version)
      if (in_section) saw_section[FILENAME] = 1
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
      # A scope that misses fails closed, which is the only reason the scoping
      # above is safe to have.
      #
      # in_section starts false in a scoped file and is turned on by a heading
      # that matches. So a heading written any other way is not a narrower
      # scope, it is *no* scope: every record in the file leaves through the
      # skip counter, the accounting still adds up perfectly, and the reader
      # exits 0 over a document it did not read a line of. Driven, four
      # ordinary spellings of the heading did exactly that — a bracketed
      # version, a leading word, a doubled space, a deeper heading level — and
      # a live assignment planted inside the section went unreported with the
      # counts all self-consistent, which is the one failure this whole file
      # exists to make impossible. (A fifth, a capital V, was the raw-record
      # read fixed above; it is a spelling of the heading and is now read.)
      #
      # Tolerating those four spellings would leave the fifth, and enumerating
      # forms is the defect this reader has now been repaired for twice. So
      # what is asserted instead is the thing that has to be true: a file
      # scoped by a heading has to have entered its section. Miss it and this
      # is a finding, loudly, rather than a silent narrowing to nothing.
      for (f in sectioned_files) {
        if (!(f in saw_section)) {
          printf "%s: no heading for the release being cut was found, so none of this file was read: it is scoped to a section that is not there\n", f
          found++
        }
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
