#!/usr/bin/env bash
# The suite for skill-rewrite.
#
# skill-rewrite is the skill PR #4's conventions never reached, and it had no
# suite of its own: the only assertions over draft-rewrite.sh were four in
# tests/test_walk.sh that ran it and read two words out of the draft it wrote.
# What that left unheld is everything a reader of the SKILL.md relies on — that
# the invocation it documents runs at all, that the draft it describes is the
# draft the script writes, and that the script has an exit contract.
#
# So the assertions here are written as comparisons between two sides rather
# than as restatements of either, in the shape tests/test_f01.sh uses for the
# rule-ID registry: the documented paths against the filesystem, the documented
# invocation against a run of it, the documented inventory against the emitted
# artifact. Either side growing without the other fails, which is the only
# arrangement under which the documentation stays true on its own.
#
# Every check that can only ever say "no violation found" carries a control
# that makes it say the opposite, because a check that cannot fail is the
# defect tests/test_harness.sh exists to refuse.
set -euo pipefail
. "$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)/lib/harness.sh"
harness_init

SKILL=skills/skill-rewrite/SKILL.md
DRAFTER=skills/skill-rewrite/scripts/draft-rewrite.sh

work="$harness_scratch/rewrite"
mkdir -p "$work"

# masked_path / run_on_path / run_masked / run_present live in the shared
# harness. A masked PATH is a symlink farm of the real PATH minus one binary:
# nothing is deleted, moved or uninstalled, and nothing is written into it.
mask_root="$harness_scratch/mask"
mkdir -p "$mask_root"
source tests/lib/masked-path.sh

# --- targets ------------------------------------------------------------------
#
# A target is copied out of tests/fixtures before it is drafted for, because
# the drafter writes REWRITE-DRAFT.md into the directory it is pointed at and a
# fixture in the repository is not a scratch directory.
#
# The copy keeps the fixture's own basename. A skill's directory name is part of
# what check-frontmatter.sh judges — `name` must match it — so a copy under a
# slot name of the suite's choosing would turn a clean fixture into a spec
# failure and the audit the drafter runs would be reporting the copy, not the
# fixture.
target_from() {
  local fixture="$1" slot="$2" dest="$work/$2/${1##*/}"
  rm -rf "$work/$slot"
  mkdir -p "$work/$slot"
  cp -R "$fixture" "$dest"
  echo "$dest"
}

# --- the documented paths -----------------------------------------------------
#
# A path in the documentation is read as a path, expanded the way the document
# tells the reader to expand it, and resolved. Reading it as prose is what let
# `scripts/draft-rewrite.sh` stand for two releases: it is text that looks
# right and resolves only when the reader's cwd is already the skill's own
# directory, which the document never said.
#
# ```text blocks are skipped. They hold illustrations of some other skill's
# layout, where a path that does not exist here is the point.
doc_referenced_paths() {
  local file="$1"
  awk '
    /^```bash$/ { fence = "bash"; next }
    /^```/      { if (fence != "") { fence = ""; next } fence = "other"; next }
    fence == "other" { next }
    { print }
  ' "$file" \
    | { grep -oE '(\$\{?[A-Za-z_][A-Za-z_0-9]*\}?/)?(scripts|references|assets)/[A-Za-z0-9_.-]+' || true; } \
    | sort -u
}

# resolve_doc_path <path> — the path with the variables the document defines
# expanded, so a reader following the document lands where the test looks.
resolve_doc_path() {
  printf '%s\n' "$1" \
    | sed -e "s|^\${*skill_root}*|skills/skill-rewrite|" \
          -e "s|^\${*audit_root}*|skills/skill-audit|"
}

doc_paths_all_resolve() {
  local p resolved bad=0
  for p in $(doc_referenced_paths "$1"); do
    resolved="$(resolve_doc_path "$p")"
    if [ ! -e "$resolved" ]; then
      echo "  documented path does not resolve: $p -> $resolved" >&2
      bad=$((bad + 1))
    fi
  done
  [ "$bad" -eq 0 ]
}

assert "every path skill-rewrite's SKILL.md documents resolves as the document says to expand it" \
  doc_paths_all_resolve "$SKILL"

# The control. A document whose path is bare and broken must be refused, or the
# assertion above is a report that the extractor found nothing.
#
# Three lines, because the control is three separate questions and only one of
# them is the refusal: the path is not there, the extractor reads it out of a
# bash fence, and the census refuses the document that carries it. The first
# two are premises and say so — naming them "the check refuses an unresolvable
# path" would put a claim in a CI log that the line underneath it never
# decided.
control_doc="$work/control-SKILL.md"
printf 'Run it:\n\n```bash\nscripts/no-such-script.sh\n```\n' > "$control_doc"
control_doc_is_refused() {
  ! doc_paths_all_resolve "$control_doc" 2>/dev/null
}
assert "the control's bare path is not there, so the census has something to refuse" \
  test ! -e "$(resolve_doc_path scripts/no-such-script.sh)"
assert "the documented-path census refuses a document carrying a path that does not resolve" \
  control_doc_is_refused
assert "the documented-path check reads a bash fence rather than skipping it" \
  test "$(doc_referenced_paths "$control_doc")" = "scripts/no-such-script.sh"

# --- the roots those paths hang from ------------------------------------------
#
# Stage 0 says every command below is anchored on `skill_root`, directly or
# through `audit_root`. Two things have to hold for that to be true, and the
# second one did not: every path in a fence carries one of the two roots, and
# every definition of `audit_root` derives it from `skill_root`. Stage 1 set
# `audit_root` to a literal path of its own, so an input Stage 0 had already
# defined had a second definition, and the first block under the claim was not
# anchored on the root the claim names.
unanchored_doc_paths_in() {
  doc_referenced_paths "$1" \
    | { grep -vE '^\$\{?(skill_root|audit_root)\}?/' || true; }
}
audit_root_definitions() {
  { grep -hoE '^audit_root=[^ ]*' "$SKILL" || true; } | sort -u
}
every_audit_root_definition_derives_from_skill_root() {
  local d found=0
  for d in $(audit_root_definitions); do
    found=1
    case "$d" in
      *'$skill_root'*) ;;
      *) echo "  the document defines $d, which is not derived from \$skill_root" >&2
         return 1 ;;
    esac
  done
  [ "$found" -eq 1 ]
}

assert "every path the SKILL.md writes in a bash fence is anchored on one of its two roots" \
  test -z "$(unanchored_doc_paths_in "$SKILL")"
assert "the SKILL.md derives audit_root from skill_root wherever it defines it" \
  every_audit_root_definition_derives_from_skill_root
# The control, on the same bare path the census above is held by: a document
# that anchors nothing must read as anchoring nothing.
assert "the anchoring check reads a bare path as anchored on neither root" \
  test "$(unanchored_doc_paths_in "$control_doc")" = "scripts/no-such-script.sh"

# --- the documented invocation ------------------------------------------------
#
# Rendered and run, not read. "The documented invocation works" is not a
# question about the text, and the one way to ask it is to execute what the
# document tells the reader to execute.
#
# The substitutions are the document's own `<...>` placeholder convention and
# nothing else: anything the block needs that the document does not offer a
# placeholder for is a thing the reader does not have either.
#
# One reader answers every question of the form "what is in the fenced block
# under this heading": the invocation blocks here, the section inventory
# further down, and the controls for both. It takes the file it reads, so a
# control can put the same question to a copy of this document that no longer
# carries the block — which is the only way to ask whether an assertion that
# the document's invocation runs is still able to fail. Three hand-written
# copies of one awk is also how this project's harness drifted into four
# versions of itself, and a repair to one copy is a repair the other two never
# see.
# Not finding the block is an answer, and it is not the empty string. A reader
# that returns nothing for "no such heading", "a heading with no fence under
# it" and "a fence with nothing in it" hands its caller a script of nothing,
# which bash runs to exit 0 — so the caller reports that the document's
# invocation ran, of a document that no longer carries it. So the three ways of
# finding nothing are three statuses, and the caller gets a refusal rather than
# an empty render.
#
# The search also stops at the next heading, and that half is not a nicety. An
# unbounded search does something worse than return nothing: with a section's
# own fence gone it runs on and returns the *next* section's block, so "the
# documented Stage 1 audit block runs as written" was reported of a document
# whose Stage 1 block was gone, on the strength of Stage 2's. A block the
# assertion names must come from the section the assertion names.
#
# `^#+ ` rather than an interval expression, because not every awk this project
# runs under has `{1,6}`. It is only consulted before a fence is open, so a
# `#` comment inside a bash block is never read as a heading.
#
# The diagnostic is printed here rather than from awk: `/dev/stderr` is not
# something every awk this project runs under provides, and a reader whose own
# diagnostic vanished would be the same defect one level down.
fenced_block_under() {
  local file="$1" heading="$2" block status=0
  block="$(awk -v h="$heading" '
    !seen && $0 ~ h          { seen = 1; next }
    seen && !fence && /^#+ / { exit }
    seen && /^```/           { if (fence) exit; fence = 1; next }
    seen && fence            { print; lines++ }
    END {
      if (!seen)  exit 2
      if (!fence) exit 3
      if (!lines) exit 4
    }
  ' "$file")" || status=$?
  case "$status" in
    0) printf '%s\n' "$block" ;;
    2) echo "  $file carries no heading matching $heading" >&2 ;;
    3) echo "  $file has a heading matching $heading with no fenced block under it" >&2 ;;
    4) echo "  $file has an empty fenced block under $heading" >&2 ;;
    *) echo "  reading $heading out of $file failed with status $status" >&2 ;;
  esac
  [ "$status" -eq 0 ]
}

doc_block() {
  local file="$1" heading="$2" target="$3" report="${4:-}" out="${5:-}" block
  block="$(fenced_block_under "$file" "$heading")" || return 1
  printf '%s\n' "$block" \
    | sed -e "s|<path-to-skill-architect>|$harness_repo_root|g" \
          -e "s|<target-skill-dir>|$target|g" \
          -e "s|<audit-report-path>|$report|g" \
          -e "s|<output-path>|$out|g"
}

run_doc_block() {
  local script="$work/doc-block.sh" block
  block="$(doc_block "$@")" || return 1
  { echo 'set -eu'; printf '%s\n' "$block"; } > "$script"
  quietly bash "$script"
}

stage1_target="$(target_from tests/fixtures/f01/valid-full stage1)"
assert "the documented Stage 1 audit block runs as written, from the repository root" \
  run_doc_block "$SKILL" '^### Stage 1' "$stage1_target"

stage2_target="$(target_from tests/fixtures/f01/valid-full stage2)"
stage2_report="$work/stage2-audit.md"
stage2_out="$work/stage2-elsewhere.md"
printf 'frontmatter OK\n' > "$stage2_report"
assert "the documented Stage 2 drafter block runs as written, from the repository root" \
  run_doc_block "$SKILL" '^### Stage 2' "$stage2_target" "$stage2_report" "$stage2_out"
assert "the documented Stage 2 block wrote the draft it says it writes" \
  test -f "$stage2_target/REWRITE-DRAFT.md"
# And the third invocation in that block, which is the one the flag exists for.
# Asserted by its own artifact rather than by the block's exit status: all three
# commands run under `set -eu`, so a `-o` line that silently wrote nowhere would
# leave the block green.
assert "the documented Stage 2 block's -o invocation wrote the draft where the document says" \
  test -f "$stage2_out"

example_target="$(target_from tests/fixtures/f01/valid-full example)"
example_report="$work/example-audit.md"
printf 'frontmatter OK\n' > "$example_report"
assert "the worked example in Examples runs as written, from the repository root" \
  run_doc_block "$SKILL" '^### Generate a rewrite draft' "$example_target" "$example_report"

# A control's expected failure is not news, so the reader's diagnostic is
# dropped here: the harness's rule is that a passing check adds no noise and a
# failing one says why, and what a control here reports is a render that ran
# when it should not have.
doc_block_refused() {
  ! run_doc_block "$@" 2>/dev/null
}

# The first control: a block that names a script that is not there. This one
# only ever covered *a block whose command fails*.
control_skill="$work/control-block-SKILL.md"
printf '### Stage 2\n\n```bash\n"<path-to-skill-architect>/skills/skill-rewrite/scripts/absent.sh"\n```\n' \
  > "$control_skill"
assert "the control block names a script that is not there, so the runner has something to fail on" \
  test ! -e "$harness_repo_root/skills/skill-rewrite/scripts/absent.sh"
assert "the block runner fails a block whose command is not there" \
  doc_block_refused "$control_skill" '^### Stage 2' "$work"

# And the controls for the other way the three assertions above stop meaning
# anything: the document no longer carrying the block at all.
#
# An assertion that a document's invocation runs must fail when the document no
# longer carries that invocation. A rendered-empty block is a missing document,
# not a passing one — so each way of arriving at an empty render is a control
# here, and each control is a copy of this document with one section mutated:
# the heading renamed, the heading kept and its fence removed, the section
# deleted whole, and the fence emptied.
#
# <mode> is one of rename-heading, drop-fence, drop-section, empty-fence.
mutated_skill() {
  local slot="$1" mode="$2" heading="$3" dest="$work/mutated-$1.md"
  if [ "$mode" = rename-heading ]; then
    sed -e "s|${heading}.*|### A heading this document no longer carries|" "$SKILL" > "$dest"
  else
    awk -v h="$heading" -v mode="$mode" '
      !seen && $0 ~ h { seen = 1; if (mode != "drop-section") print; next }
      seen && !done && /^```/ {
        if (mode == "empty-fence") print
        if (infence) { done = 1; infence = 0 } else { infence = 1 }
        next
      }
      seen && infence { next }
      { print }
    ' "$SKILL" > "$dest"
  fi
  echo "$dest"
}

stage1_renamed="$(mutated_skill stage1-renamed rename-heading '^### Stage 1')"
example_renamed="$(mutated_skill example-renamed rename-heading '^### Generate a rewrite draft')"
stage1_unfenced="$(mutated_skill stage1-unfenced drop-fence '^### Stage 1')"
stage1_deleted="$(mutated_skill stage1-deleted drop-section '^### Stage 1')"
stage1_emptied="$(mutated_skill stage1-emptied empty-fence '^### Stage 1')"

assert "the block runner refuses a document whose Stage 1 heading was renamed" \
  doc_block_refused "$stage1_renamed" '^### Stage 1' "$stage1_target"
assert "the block runner refuses a document whose worked-example heading was renamed" \
  doc_block_refused "$example_renamed" '^### Generate a rewrite draft' "$example_target" "$example_report"
assert "the block runner refuses a document that kept the Stage 1 heading and lost its fence" \
  doc_block_refused "$stage1_unfenced" '^### Stage 1' "$stage1_target"
assert "the block runner refuses a document the whole Stage 1 section is gone from" \
  doc_block_refused "$stage1_deleted" '^### Stage 1' "$stage1_target"
assert "the block runner refuses a Stage 1 fence with nothing in it" \
  doc_block_refused "$stage1_emptied" '^### Stage 1' "$stage1_target"

# The premise those five rest on: each mutated copy is this document minus the
# one section its mutation names, so a control that passes is reporting the
# reader's refusal and not an empty or unreadable file.
mutations_kept_the_rest_of_the_document() {
  local m
  for m in "$stage1_renamed" "$example_renamed" "$stage1_unfenced" \
           "$stage1_deleted" "$stage1_emptied"; do
    run_doc_block "$m" '^### Stage 2' "$stage2_target" "$stage2_report" || return 1
  done
}
assert "each mutated copy still carries the Stage 2 block its mutation did not name" \
  mutations_kept_the_rest_of_the_document

# --- the report the caller named ----------------------------------------------
#
# `-a <path>` is the caller stating what the draft is to be built from. A path
# that is not there is a typo, and the two answers available to the drafter are
# to use the report it was given and to say it could not. Quietly auditing
# afresh instead is neither: it substitutes a different basis for the one the
# caller named, writes a draft over it, and then says "No audit report
# provided" — which is not what happened, and leaves the caller reading a draft
# built from something they did not ask for.
drafted="$work/run"
mkdir -p "$drafted"
draft_run() {
  code=0
  output="$("$DRAFTER" "$@" 2>"$drafted/stderr")" || code=$?
  errout="$(cat "$drafted/stderr")"
}

absent_report_target="$(target_from tests/fixtures/f01/valid-full absent-report)"
draft_run -t "$absent_report_target" -a "$work/no-such-report.md"
assert "a -a report that is not there is refused rather than replaced" \
  test "$code" -ne 0
assert "a -a report that is not there is named in the diagnostic" \
  grep -q "no-such-report.md" "$drafted/stderr"
assert "a -a report that is not there leaves no draft behind" \
  test ! -f "$absent_report_target/REWRITE-DRAFT.md"
assert "a refused -a report does not claim that no report was provided" \
  test -z "$(grep -F 'No audit report provided' "$drafted/stderr" || true)"

# The control: the same call with a report that is there must succeed and must
# build the draft from that report, or every assertion above would pass on a
# drafter that refuses every `-a`.
present_report_target="$(target_from tests/fixtures/f01/valid-full present-report)"
present_report="$work/present-audit.md"
printf 'POLICY FAIL [PL999]: a finding only this report carries\n' > "$present_report"
draft_run -t "$present_report_target" -a "$present_report"
assert "a -a report that is there is accepted" test "$code" -eq 0
assert "a -a report that is there is what the draft is built from" \
  grep -q "PL999" "$present_report_target/REWRITE-DRAFT.md"
assert "a -a report that is there is not shadowed by a fresh audit" \
  test ! -s "$drafted/stderr"

# --- the exit contract --------------------------------------------------------
#
# Every status the script can exit with, against every status its own header
# registers. This is the shape tests/test_f01.sh uses for the rule-ID registry,
# and for the same reason: a header that restates the code is a header free to
# drift from it, and the four skill-audit scripts each carry an `Exit codes:`
# line a reader reaches for before reaching for the skill's documentation.
# draft-rewrite.sh carried none, so there was nothing for a caller to wire to
# and nothing to hold the script to.
#
# Read from the source rather than from a run, because the point is the whole
# set: a behavioural sweep can only report the statuses the sweep happened to
# provoke.
#
# Code lines only. A comment is where the script explains the statuses it used
# to exit with and no longer does — `a broken cat made -h exit 2 where this
# contract says 0` is a sentence about a repair, not a status the script can
# reach — and counting those would demand the header register a status the code
# cannot emit, which is the false header this assertion exists to refuse. It is
# the distinction the registry above draws with backticks, made the other way
# round: a paragraph may discuss what it does not declare. The narrowing gives
# up nothing, because no `exit N` the script can take is on a line that begins
# with `#`.
statuses_emitted() {
  { grep -vE '^[[:space:]]*#' "$DRAFTER" || true; } \
    | { grep -hoE '(^|[^[:alnum:]_])exit[[:space:]]+[0-9]+' || true; } \
    | { grep -oE '[0-9]+' || true; } | sort -u
}

statuses_registered() {
  { grep -m1 -E '^# Exit codes:' "$DRAFTER" || true; } \
    | { grep -oE '[0-9]+=' || true; } | tr -d '=' | sort -u
}

assert "the drafter's header registers a status at all" \
  test -n "$(statuses_registered)"
assert "the drafter's header registers every status it can exit with, and no status it cannot" \
  test "$(statuses_emitted)" = "$(statuses_registered)"
if [ "$(statuses_emitted)" != "$(statuses_registered)" ]; then
  echo "  emitted   : $(statuses_emitted | tr '\n' ' ')"
  echo "  registered: $(statuses_registered | tr '\n' ' ')"
fi

# And the statuses a caller actually meets, so the registry above is a registry
# of something the script does rather than of something it says.
draft_run -h
assert "--help exits 0" test "$code" -eq 0
draft_run
assert "no target exits 1" test "$code" -eq 1
draft_run --nonsense
assert "an unknown option exits 1" test "$code" -eq 1
draft_run -t "$work/no-such-directory"
assert "a target directory that is not there exits 1" test "$code" -eq 1

# A flag whose value is missing is a usage error like any other. Reading `$2`
# when there is no `$2` makes it an unbound-variable abort instead: the caller
# gets a line of bash internals naming a position in the parser rather than the
# option they mistyped.
#
# Which of the two happened is not a question the exit status answers — an
# unbound-variable abort under `set -u` exits 1 as well — so the status line
# below claims only the status, and the line that tells the two states apart is
# the one that reads stderr for the abort's own words. A label that named a
# distinction its own command cannot make would be read in CI as a check that
# had made it.
for flag in -t --target -a --audit -o --output; do
  draft_run "$flag"
  assert "$flag with no value exits 1, the status the drafter's header registers for a usage error" \
    test "$code" -eq 1
  assert "$flag with no value names the option rather than a shell positional" \
    test -z "$(grep -F 'unbound variable' "$drafted/stderr" || true)"
  assert "$flag with no value says which option is missing a value" \
    grep -qF -- "$flag" "$drafted/stderr"
done

# --- the options, against the usage line that names them ----------------------
#
# The usage line is the only description of this script a caller ever reads
# without opening it, and it was a hand-kept list: `-o` could be parsed and go
# unnamed, or named and not parsed, and nothing would notice either way. So the
# two sides are compared rather than restated — the options the parser accepts,
# read out of its own case arms, against the options `-h` prints.
#
# Read from the source for the parser's side, because the point is the whole
# set: a behavioural sweep can only report the options the sweep happened to
# try, which is exactly how an undocumented one survives.
parsed_options() {
  { grep -oE '^ *-[a-z]\|--[a-z]+\)' "$DRAFTER" || true; } \
    | tr -d ' )' | tr '|' '\n' | sort -u
}
usage_options() {
  "$DRAFTER" -h | { grep -oE '^  -[a-z], --[a-z]+' || true; } \
    | tr -d ',' | tr ' ' '\n' | { grep '^-' || true; } | sort -u
}

assert "the drafter's parser accepts an option at all, so the comparison has two sides" \
  test -n "$(parsed_options)"
assert "the usage line names an option at all" test -n "$(usage_options)"
assert "the usage line names every option the parser accepts, and none it does not" \
  test "$(parsed_options)" = "$(usage_options)"
if [ "$(parsed_options)" != "$(usage_options)" ]; then
  echo "  parsed: $(parsed_options | tr '\n' ' ')"
  echo "  usage : $(usage_options | tr '\n' ' ')"
fi

# --- where the draft is written -----------------------------------------------
#
# `-o` makes the destination the caller's to name, and that is the whole point
# of it: a draft you can put beside your notes instead of inside the skill you
# are auditing. It also removes the only bound the drafter had. Until now the
# destination was `$target_skill/REWRITE-DRAFT.md`, and the `-t` checks were the
# containment: the caller had named a directory that exists and holds a
# `SKILL.md`, so the worst the drafter could overwrite was a previous draft of
# its own.
#
# This repository has twice shipped the failure that follows from not restating
# such a bound — a documented command that `rm -rf`'d a live skills directory,
# and a suite whose containment guard printed FAIL and then ran the install
# anyway. So the bound is restated, in the narrow form the drafter can justify:
# `-o` may write anywhere the caller can write, **except** three destinations
# where a Markdown draft is not a new file but a destruction or an
# impersonation. Those three are asserted here, and so is the far larger set it
# does not refuse — because a bound that refused the destinations the flag
# exists for would be the flag not existing, and an assertion that something is
# refused means nothing beside an assertion that something else is not.
#
# Every destination below is under a redirected `HOME` inside the harness's
# scratch directory. Nothing here reads or writes a live configuration
# directory, and nothing installs or reconfigures anything: the protected roots
# the drafter computes are `$HOME`-relative, which is what makes the decision
# askable at all without pointing a test at the developer's own `~/.claude`.
out_home="$work/out-home"
rm -rf "$out_home"
mkdir -p "$out_home/.claude/skills" "$out_home/.claude-notes" "$out_home/plain"

# The destination with nothing wrong in its spelling that lands inside a
# protected directory anyway, because something on the way is a symlink. This is
# the case a string test cannot reach and the reason the decision is taken after
# resolution: `~/drafts/x.md` is a path whose shape says nothing and whose
# resolution says everything.
ln -s "$out_home/.claude/skills" "$out_home/drafts"

draft_run_home() {
  local home="$1"
  shift
  code=0
  output="$(HOME="$home" "$DRAFTER" "$@" 2>"$drafted/stderr")" || code=$?
  errout="$(cat "$drafted/stderr")"
}

# What the flag is for, asserted first: the draft goes where the caller said,
# is named on stdout, and is *not* also left in the target directory. A flag
# that wrote both places would satisfy every refusal below and still not be the
# flag anybody asked for.
named_out_target="$(target_from tests/fixtures/f01/valid-full named-out)"
named_out="$out_home/plain/named.md"
draft_run_home "$out_home" -t "$named_out_target" -o "$named_out"
assert "a draft written with -o exits 0" test "$code" -eq 0
assert "a draft written with -o is at the path the caller named" test -f "$named_out"
assert "a draft written with -o carries the audit it was built from" \
  grep -q 'frontmatter OK' "$named_out"
assert "a draft written with -o names its own path on stdout" \
  test -n "$(printf '%s\n' "$output" | grep -F "$named_out" || true)"
assert "a draft written with -o is not also left in the target directory" \
  test ! -f "$named_out_target/REWRITE-DRAFT.md"

# And the long option, because the parser has two arms and only one of them was
# just exercised.
long_out_target="$(target_from tests/fixtures/f01/valid-full long-out)"
long_out="$out_home/plain/long.md"
draft_run_home "$out_home" -t "$long_out_target" --output "$long_out"
assert "--output is the same flag as -o" test "$code" -eq 0
assert "--output wrote the draft where it was told to" test -f "$long_out"

# The sibling that shares a name prefix with a protected directory, and is not
# inside it. `$HOME/.claude-notes` has `$HOME/.claude` as a string prefix, so a
# `HasPrefix` containment test refuses this — a legitimate destination lost to a
# decision taken on the shape of the string. Accepting it is half of what makes
# the decision resolution-based rather than textual, and it is the half a
# refusal-only suite would never notice was broken.
prefix_target="$(target_from tests/fixtures/f01/valid-full prefix-sibling)"
prefix_out="$out_home/.claude-notes/draft.md"
draft_run_home "$out_home" -t "$prefix_target" -o "$prefix_out"
assert "a destination that merely begins like a protected directory is accepted" \
  test "$code" -eq 0
assert "the prefix-sibling destination was written" test -f "$prefix_out"

# refused_out <home> <target> <destination> — the drafter refuses this
# destination, and refuses it rather than reporting it.
#
# Three questions, not one, and that is the lesson of the guard that printed
# FAIL and installed anyway: the status is a refusal, nothing was written at the
# destination, and nothing was left in the target directory either. A drafter
# that said no and wrote anyway would pass the first.
#
# It reads the status and not merely `!= 0`, and the difference is a whole
# defect. While this asked only for nonzero, every refusal below was satisfied
# by exit 3 — *"the destination could not be resolved, so where the draft would
# be written is unknown"* — which is a different answer from the one being
# asserted: it says no verdict was reached, not that the caller named a
# destination the drafter will not write. The resolution step was emitting it for
# any destination that merely already existed as a regular file, and two of the
# refusals here were passing on it while the rules they name never ran. The
# status registry the header publishes makes 1 a usage error, so a refusal of
# the caller's destination is 1 and nothing else.
refused_out() {
  local home="$1"
  local target="$2"
  local dest="$3"
  draft_run_home "$home" -t "$target" -o "$dest"
  if [ "$code" -eq 0 ]; then
    printf 'the drafter accepted a destination it must refuse: %s\n' "$dest" >&2
    return 1
  fi
  if [ "$code" -ne 1 ]; then
    printf 'the drafter refused %s with exit %s; a refusal of the destination is exit 1, and %s means no verdict was reached:\n%s\n' \
      "$dest" "$code" "$code" "$errout" >&2
    return 1
  fi
  return 0
}

# (i) Inside a live agent configuration directory, reached through a symlink.
# The destination's own spelling is `$HOME/drafts/draft.md`: outside by every
# string test, and inside in fact.
symlink_target="$(target_from tests/fixtures/f01/valid-full symlinked-out)"
assert "the symlinked destination is outside the protected directory by its spelling" \
  test -z "$(printf '%s\n' "$out_home/drafts/draft.md" | grep -F "$out_home/.claude" || true)"
assert "a destination that resolves into a live config directory is refused" \
  refused_out "$out_home" "$symlink_target" "$out_home/drafts/draft.md"
assert "the destination that resolved into a live config directory was not written" \
  test ! -f "$out_home/.claude/skills/draft.md"
assert "a refused destination leaves no draft in the target directory either" \
  test ! -f "$symlink_target/REWRITE-DRAFT.md"
assert "the refusal names the destination the caller gave" \
  test -n "$(printf '%s\n' "$errout" | grep -F "$out_home/drafts/draft.md" || true)"
assert "the refusal names the protected directory it resolved into" \
  test -n "$(printf '%s\n' "$errout" | grep -F '.claude' || true)"

# (ii) The same directory named directly, which is the accident rather than the
# aliased case: a `$HOME`-relative path assembled by a caller that meant
# somewhere else.
direct_target="$(target_from tests/fixtures/f01/valid-full direct-config-out)"
assert "a destination named directly inside a live config directory is refused" \
  refused_out "$out_home" "$direct_target" "$out_home/.claude/skills/direct.md"
assert "the directly named config destination was not written" \
  test ! -f "$out_home/.claude/skills/direct.md"

# The same rule at the root the list did not have. `$HOME/.agents/skills/` is
# not a harness's own configuration directory — it is the shared one Codex and
# Cursor both read — which is exactly why the list built from harness names
# missed it, and why the rationale is about what an agent reads rather than
# about whose directory it is. Driven behaviourally as well as compared as a
# list, because the list comparison would pass over a root the loop never uses.
mkdir -p "$out_home/.agents/skills"
agents_target="$(target_from tests/fixtures/f01/valid-full agents-config-out)"
assert "a destination inside the shared agent skills directory is refused" \
  refused_out "$out_home" "$agents_target" "$out_home/.agents/skills/shared.md"
assert "the shared agent skills destination was not written" \
  test ! -f "$out_home/.agents/skills/shared.md"

# (iii) A `SKILL.md`. This is the one refusal that is not about the caller's
# machine but about this skill's own stated constraint — "Do not overwrite the
# original `SKILL.md` without explicit approval" — which had no mechanism for as
# long as the destination was hardcoded. `-o` is what makes it reachable, so
# `-o` is where it becomes enforceable. A rewrite draft is not a skill, and a
# draft written over a `SKILL.md` destroys the document the draft was built by
# reading.
skillmd_target="$(target_from tests/fixtures/f01/valid-full skillmd-out)"
skillmd_before="$work/skillmd-before"
cp "$skillmd_target/SKILL.md" "$skillmd_before"
assert "a destination that is the target's own SKILL.md is refused" \
  refused_out "$out_home" "$skillmd_target" "$skillmd_target/SKILL.md"
assert "the refused SKILL.md is byte-identical to what it was" \
  quietly cmp -s "$skillmd_before" "$skillmd_target/SKILL.md"

# And the alias, which is how a SKILL.md is reached without being named. A
# destination whose own last component says nothing points at one, and `: >`
# follows the link and truncates what is on the other end — so the SKILL.md rule
# alone does not close this, and the rule that does is the refusal of a leaf
# symlink. The assertion is written over the outcome rather than over which of
# the two rules fired, because the outcome is what the skill's Constraints
# promise: the SKILL.md is still there, byte for byte.
alias_target="$(target_from tests/fixtures/f01/valid-full aliased-skillmd-out)"
alias_before="$work/alias-before"
cp "$alias_target/SKILL.md" "$alias_before"
ln -s "$alias_target/SKILL.md" "$out_home/plain/notes.md"
assert "the aliased destination's own name is not SKILL.md" \
  test "$(basename "$out_home/plain/notes.md")" != SKILL.md
assert "a destination that resolves onto a SKILL.md is refused" \
  refused_out "$out_home" "$alias_target" "$out_home/plain/notes.md"
assert "the aliased SKILL.md is byte-identical to what it was" \
  quietly cmp -s "$alias_before" "$alias_target/SKILL.md"

# (iv) A directory. The redirect would fail of its own accord, and the caller
# would be told the draft could not be opened when the truth is that they named
# a directory. A wrong diagnostic is a wrong answer.
dir_target="$(target_from tests/fixtures/f01/valid-full dir-out)"
assert "a destination that is an existing directory is refused" \
  refused_out "$out_home" "$dir_target" "$out_home/plain"
assert "the refusal of a directory says it is a directory" \
  test -n "$(printf '%s\n' "$errout" | grep -Fi 'directory' || true)"

# The list of protected directories, against the list the SKILL.md gives a
# reader. Asserted as a comparison rather than as five literals, in the shape
# this suite reads every other registry in: a reader deciding whether their
# destination is refused reads the document, so the document has to be the
# script's list and not a remembered copy of it. A root added to either side
# leaves them unequal, whichever side is edited first.
protected_roots_in_script() {
  { grep -m1 -E '^protected_home_dirs=' "$DRAFTER" || true; } \
    | sed -e 's/^[^=]*=//' -e 's/"//g' | tr ' ' '\n' \
    | { grep '^\.' || true; } | sort -u
}
documented_protected_roots() {
  { grep -m1 -E '^Protected destinations:' "$SKILL" || true; } \
    | { grep -oE '`\$HOME/\.[a-z]+`' || true; } \
    | sed -e 's|.*/||' -e 's|`$||' | sort -u
}

assert "the drafter protects a live configuration directory at all" \
  test -n "$(protected_roots_in_script)"
assert "the SKILL.md names a protected destination at all" \
  test -n "$(documented_protected_roots)"
assert "the SKILL.md names every live configuration directory the drafter protects, and none it does not" \
  test "$(protected_roots_in_script)" = "$(documented_protected_roots)"
if [ "$(protected_roots_in_script)" != "$(documented_protected_roots)" ]; then
  echo "  in the script: $(protected_roots_in_script | tr '\n' ' ')"
  echo "  documented   : $(documented_protected_roots | tr '\n' ' ')"
fi

# And the side that was missing, which is the one the rationale actually rests
# on. The refusal exists because "an agent reads its skills directory as
# skills", so the list has to be the set of directories *this repository tells a
# reader an agent reads skills from* — and the document that tells them that is
# the README, not the SKILL.md. Held against the SKILL.md alone, the two lists
# agreed with each other and both disagreed with the README: `$HOME/.agents/`
# is documented there as a global skills directory for Codex and for Cursor, and
# it was not protected. Reproduced — rc=0, draft written into
# `$HOME/.agents/skills/`.
#
# Read off the README as a path shape rather than out of a table, because the
# table is prose and the paths are spread through it: every `$HOME`-relative
# path the document spells with a `skills` component in it, reduced to its first
# component. That is deliberately over-inclusive — `~/.devin/skills/` appears
# there only to say it is *not* the place, and this counts it anyway — and
# over-inclusive is the safe direction for a refusal list: protecting a
# directory nobody reads costs a destination, and not protecting one costs a
# document loaded as instructions.
readme_skills_roots() {
  { grep -oE '~/\.[a-z][a-z-]*(/[a-z][a-z-]*)*/skills' README.md || true; } \
    | sed -e 's|^~/||' -e 's|/.*$||' | sort -u
}

assert "the README names a \$HOME-relative skills directory at all, so the comparison has two sides" \
  test -n "$(readme_skills_roots)"
assert "the drafter protects every directory the README documents an agent reading skills from, and none it does not" \
  test "$(protected_roots_in_script)" = "$(readme_skills_roots)"
if [ "$(protected_roots_in_script)" != "$(readme_skills_roots)" ]; then
  echo "  in the script  : $(protected_roots_in_script | tr '\n' ' ')"
  echo "  in the README  : $(readme_skills_roots | tr '\n' ' ')"
fi

# And the premise that comparison rests on: the roots are `$HOME`-relative, so
# the refusals above were decided against the redirected home this suite made
# and not against the developer's own. A drafter that had resolved them from a
# literal `/Users/...` would pass every refusal assertion above on the machine
# that wrote it and none on anybody else's — and would be reading a live
# configuration directory to do it.
assert "the protected directories are anchored on \$HOME, not on a literal home path" \
  test -n "$(grep -F 'path_target_is_inside "$spelled" "$HOME/$root" or-equal' "$DRAFTER" || true)"

# Every refusal above is a usage error — the caller named a destination the
# drafter will not write — so it takes the status this script's header registers
# for one. Asserted as the status and not merely as nonzero, because the two
# statuses mean different things to a caller and the registry assertion further
# up would pass either way.
assert "a refused destination exits 1, the status the header registers for a usage error" \
  test "$code" -eq 1

# --- the bound, against where a write on that spelling actually lands ---------
#
# Everything above names a destination somebody thought of, and that is how the
# bound came to be defeatable purely by spelling — twice. The first round of it
# was a textual `..` fold standing in front of symlink resolution. The second
# round was the repair for the first: it compared the *bytes of a resolved name*
# against the bytes of a name built for the root, and a name is not an identity.
# `$HOME/.CLAUDE/skills/x` and an NFD spelling of an NFC home both resolved to a
# string that did not match and landed in the live directory anyway, for all six
# roots; and the resolved name came back through a command substitution, which
# strips trailing newlines, so a destination spelled with one was *decided* on
# one path and *written* on another.
#
# So no case below names an expected verdict — and, the repair to the test this
# time, no case below is written out by hand either. The previous round's table
# was twelve spellings, which is twelve of the spellings its author thought of:
# two whole classes of escape sat outside it, and inserting one row into it
# turned it red. A table cannot be the completeness argument, for the same
# reason a denylist cannot: it is complete only about what somebody enumerated.
#
# What is enumerated here instead is *the protected roots the script itself
# declares* and *a set of mutation operators*, and the cases are their product.
# The roots come from `protected_roots_in_script`, which reads the drafter's own
# `protected_home_dirs` line, so a seventh root is covered the moment it is
# added and without anybody remembering that this file exists. The operators are
# the mechanisms by which a name can name something other than what it appears
# to — case, Unicode normalisation, a trailing byte, a link at the leaf, a link
# mid-path, a dangling link, a `..` crossing any of them — so a newly understood
# mechanism is one line here and is then applied to every root.
#
# Each generated spelling is performed twice over two trees built by one
# function. Once as a plain redirect, so the filesystem says where a write on
# that exact spelling lands; once through the drafter. The oracle is the kernel,
# and it is an *identity* oracle rather than a name one: the object the write
# created is compared with the root by `-ef` and searched for under the root by
# content, so neither a link hanging out of the tree nor a spelling of the root
# can fool it.
#
# What is asserted is the safety direction without qualification — a spelling
# whose bytes land inside a protected directory is refused — and the other
# direction up to the three refusals this script publishes: a destination that
# lands outside is accepted unless it is refused as a symbolic link, as a
# directory or as a SKILL.md, and a refusal that names none of those is a
# failure. That is what stops the repair from being "refuse everything" while
# still allowing the rules `-o` documents. The leaf-symlink rule refuses every
# leaf link before containment is ever asked, so the proof that the walk
# *follows* a leaf link lives where there is no such rule in front of it: in
# tests/test_install.sh, over the copy of the walk this file's own copy is
# asserted to be byte-identical to.
#
# A spelling the kernel refuses to write on at all is recorded and not asserted:
# no object was created or truncated, so there is no landing place for a verdict
# to agree or disagree with. The count is asserted instead, because a generator
# whose cases had all quietly become unperformable would otherwise look exactly
# like a generator that passed.
containment_tree() {
  local h="$1"
  local r="$2"
  rm -rf "$h"
  mkdir -p "$h/$r/skills" "$h/${r}-notes" "$h/${r}X" "$h/plain" "$h/elsewhere"
  # Beside the protected directory, pointing into it — and a chain of two, so
  # one hop of resolution is not mistaken for all of it.
  ln -s "$h/$r/skills" "$h/aside"
  ln -s "$h/aside" "$h/chain"
  # Inside the protected directory, pointing out of it. Once with an absolute
  # target and once with a relative one, because a relative target is resolved
  # against the link's own directory and that is a second thing to get wrong.
  ln -s "$h/elsewhere" "$h/$r/away"
  ln -s "../elsewhere" "$h/$r/rel"
  # At the leaf. `: > "$dest"` follows a leaf symlink, so each of these is a
  # write into the protected directory whatever the destination is called.
  : > "$h/$r/skills/leaf.md"
  ln -s "$h/$r/skills/leaf.md" "$h/plain/leaflink"
  ln -s "$h/$r/skills/dangles.md" "$h/plain/dangling"
  # The same link with a trailing newline in *its own* name, which is the
  # spelling the previous round's barrier decided on one path and wrote on
  # another: the resolved name came back through `$( )`, which strips the
  # newline, so the verdict was reached for `nl-link` and the redirect was
  # performed on `nl-link` followed by a newline — a different entry, and this
  # one is a link into the protected directory.
  ln -s "$h/$r/skills/nl-leaf.md" "$h/plain/nl-link"$'\n'
  # And the mirror of them, pointing out, so the repair cannot be "follow the
  # leaf and refuse".
  : > "$h/elsewhere/leaf.md"
  ln -s "$h/elsewhere/leaf.md" "$h/$r/skills/outlink"
  # A directory link in front of the leaf links, to reach the same two through
  # one more hop.
  ln -s "$h/plain" "$h/plainlink"
  # A link whose *target* ends with a newline, which is a different defect from
  # a link whose *name* does. `twin` is a plain file outside the protected
  # directory; `twin` followed by a newline is a link to a file inside it. A
  # link whose target is the second of those, read back through a command
  # substitution, comes back naming the first — so the decision would be taken
  # about a file outside while the redirect truncates one inside.
  : > "$h/plain/twin"
  ln -s "$h/$r/skills/leaf.md" "$h/plain/twin"$'\n'
  ln -s "$h/plain/twin"$'\n' "$h/plain/via-nl-target"
}

# <home> <root> <destination> — inside, outside, or nowrite: where a plain
# redirect on this exact spelling put its bytes, answered by identity and by
# content and never by comparing the destination's name with the root's.
containment_landed() {
  local h="$1" r="$2" dest="$3" marker
  marker="where-did-this-land-$$-${RANDOM}"
  if ! printf '%s\n' "$marker" > "$dest" 2>/dev/null; then
    printf 'nowrite\n'
    return 0
  fi
  if [ -e "$h/$r" ] && [ "$dest" -ef "$h/$r" ]; then
    printf 'inside\n'
    return 0
  fi
  # `find` does not follow the links this tree hangs out of the protected
  # directory, so a file it reports under the root really is under it.
  if [ -d "$h/$r" ] &&
    [ -n "$(find "$h/$r" -type f -exec grep -lF -- "$marker" {} + 2>/dev/null || true)" ]; then
    printf 'inside\n'
  else
    printf 'outside\n'
  fi
}

# <root> <destination template> — the drafter's verdict for this spelling is the
# answer the filesystem gives for it. `@H` stands for the home the tree is built
# under, so one template can be instantiated in two trees; `@N` for that same
# home spelled NFD, which is the same directory under a different name; and `@R`
# for the protected root being generated over.
containment_agrees() {
  local r="$1"
  local template="${2//@R/$1}"
  local oracle_home="$work/bound-oracle/$bound_home_nfc"
  local drafter_home="$work/bound-drafter/$bound_home_nfc"
  local oracle_dest drafter_dest landed verdict
  containment_tree "$oracle_home" "$r"
  containment_tree "$drafter_home" "$r"
  oracle_dest="${template//@H/$oracle_home}"
  oracle_dest="${oracle_dest//@N/$work/bound-oracle/$bound_home_nfd}"
  drafter_dest="${template//@H/$drafter_home}"
  drafter_dest="${drafter_dest//@N/$work/bound-drafter/$bound_home_nfd}"

  bound_cases=$((bound_cases + 1))
  landed="$(containment_landed "$oracle_home" "$r" "$oracle_dest")"
  if [ "$landed" = nowrite ]; then
    bound_unperformable=$((bound_unperformable + 1))
    return 0
  fi

  draft_run_home "$drafter_home" -t "$bound_target" -o "$drafter_dest"
  if [ "$code" -eq 0 ]; then verdict=accepted; else verdict=refused; fi

  if [ "$landed" = inside ]; then
    if [ "$verdict" = refused ]; then
      bound_refused_inside=$((bound_refused_inside + 1))
      return 0
    fi
    printf 'the destination %s: a write on that spelling lands inside the protected directory, and the drafter accepted it (exit %s)\n%s\n' \
      "$template" "$code" "$errout" >&2
    return 1
  fi
  if [ "$verdict" = accepted ]; then
    bound_accepted_outside=$((bound_accepted_outside + 1))
    return 0
  fi
  case "$errout" in
    *'is a symbolic link'* | *'is a directory, and the draft is a file'* | *'which is a SKILL.md'*)
      bound_refused_by_rule=$((bound_refused_by_rule + 1))
      return 0
      ;;
  esac
  printf 'the destination %s: a write on that spelling lands outside every protected directory, and the drafter refused it (exit %s) for a reason that is not one of the three refusals it publishes\n%s\n' \
    "$template" "$code" "$errout" >&2
  return 1
}

# The operators. One line per mechanism by which a spelling can name an object
# other than the one it appears to name, each applied to every root. `@R` is the
# root, so the case set grows with the root list and not with this list.
#
# Deliberately in both directions: the ones that resolve *into* the protected
# directory while their spelling says otherwise are the hole, and the ones that
# resolve *out* of it while their spelling says inside are the false refusal,
# which costs a caller a legitimate destination. A repair that closed only the
# first would pass half of this.
bound_operators=(
  '@H/@R/skills/exact.md'
  '@H/@R'
  '@H/@R/skills/UPPER-LEAF.md'
  '@H/plain/plain.md'
  '@H/@RX/sibling.md'
  '@H/@R-notes/sibling.md'
  '@H/aside/mid-link.md'
  '@H/chain/mid-chain.md'
  '@H/aside/../climb-out-of-link.md'
  '@H/chain/../climb-out-of-chain.md'
  '@H/aside/../../@R/skills/climb-back-in.md'
  '@H/@R/away/out-link.md'
  '@H/@R/away/../climb-out.md'
  '@H/@R/rel/../climb-out-rel.md'
  '@H/plain/leaflink'
  '@H/plain/dangling'
  '@H/plainlink/leaflink'
  '@H/@R/skills/outlink'
  '@H/plain/../@R/skills/through-dotdot.md'
  '@H/@R/skills/../../plain/out-through-dotdot.md'
  '@H/@R/nope/../tail-fold.md'
  '@H/@R/nope/../../plain/tail-climb.md'
  '@H//@R///skills//doubled.md'
  '@H/./@R/./skills/./dots.md'
  '@N/@R/skills/nfd-home.md'
  '@N/plain/nfd-home-outside.md'
  '@N/aside/nfd-home-mid-link.md'
  '@H/@R/skills/leaf.md/through-a-file.md'
  '@H/elsewhere/leaf.md/through-a-file.md'
  '@H/plain/via-nl-target'
)

# The mutations that are about the *bytes* of the spelling rather than its
# shape. These are the ones the previous round's table had no row for and could
# not have had one for: the escape is not a path shape, it is the difference
# between the name a barrier compares and the name the kernel writes on. Written
# with `$'…'` so the byte is in the array and not in a format string — a
# trailing newline put through `$(printf …)` would be stripped by the very
# mechanism being tested.
bound_byte_operators=(
  '@H/@R/skills/trailing-nl.md'$'\n'
  '@H/plain/trailing-nl.md'$'\n'
  '@H/@R/skills/trailing-tab.md'$'\t'
  '@H/plain/trailing-tab.md'$'\t'
  '@H/@R/skills/trailing-space.md '
  '@H/plain/trailing-space.md '
  '@H/@R/skills/two-newlines.md'$'\n\n'
  '@H/plain/nl-link'$'\n'
  '@H/plainlink/nl-link'$'\n'
)

bound_cases=0
bound_unperformable=0
bound_refused_inside=0
bound_accepted_outside=0
bound_refused_by_rule=0
bound_home_nfc=$'caf\xc3\xa9/h'
bound_home_nfd=$'cafe\xcc\x81/h'
bound_target="$(target_from tests/fixtures/f01/valid-full bound-target)"

for bound_root in $(protected_roots_in_script); do
  bound_upper="$(printf '%s' "$bound_root" | tr 'a-z' 'A-Z')"
  for bound_case in "${bound_operators[@]}" "${bound_byte_operators[@]}"; do
    assert "the drafter's verdict for $(printf '%q' "${bound_case//@R/$bound_root}") is where a write on it lands" \
      containment_agrees "$bound_root" "$bound_case"
  done
  # The root component spelled in a case the volume folds. Live for all six
  # roots at the head this replaces: the exact spelling was refused and the
  # capitalised one was accepted, and the draft landed in the live directory.
  for bound_case in '@H/@U/skills/case-folded.md' '@H/@U/skills/../@U/skills/case-folded-twice.md' '@N/@U/skills/nfd-and-case-folded.md'; do
    assert "the drafter's verdict for ${bound_case//@U/$bound_upper} is where a write on it lands" \
      containment_agrees "$bound_root" "${bound_case//@U/$bound_upper}"
  done
done

# A generated case set that had quietly become unperformable would look exactly
# like one that passed — every case would return 0 having asserted nothing — so
# the sweep reports what it actually did and both directions are asserted to
# have happened.
#
# Both directions and not a proportion of unperformable cases, deliberately: the
# proportion is different on a case-sensitive filesystem, where a folded
# spelling of a protected root names a directory that is not there and this
# script, which never creates a parent, cannot write to it. A bound that has to
# be retuned per platform is a bound nobody trusts. What has to be true
# everywhere is that the sweep really saw a write land inside a protected
# directory and really saw the drafter refuse it, and really saw one land outside
# and really saw the drafter accept it.
echo "  the generated bound drove $bound_cases spellings: $bound_unperformable unwritable, $bound_refused_inside refused for landing inside, $bound_accepted_outside accepted for landing outside, $bound_refused_by_rule refused by a published destination rule"
assert "the generated bound generated a case set at all" \
  test "$bound_cases" -gt 100
assert "the generated bound saw a write land inside a protected directory and the drafter refuse it" \
  test "$bound_refused_inside" -gt 0
assert "the generated bound saw a write land outside every protected directory and the drafter accept it" \
  test "$bound_accepted_outside" -gt 0

# `-o` re-run over its own output. The flag exists so a draft can be kept
# somewhere of the caller's choosing, and a caller who audits the same skill
# twice writes to the same file twice — which the default destination does
# happily, overwriting the previous `REWRITE-DRAFT.md`. While resolution
# ended in `cd -P`, every destination that already existed as a regular file
# came back unresolvable, so the second run exited 3 saying it could not tell
# where the draft would go: an undocumented fifth refusal, and the one
# destination the flag is most likely to be pointed at twice.
idem_target="$(target_from tests/fixtures/f01/valid-full idempotent-out)"
idem_out="$out_home/plain/idempotent.md"
rm -f "$idem_out"
draft_run_home "$out_home" -t "$idem_target" -o "$idem_out"
assert "the first run to a fresh destination exits 0" test "$code" -eq 0
idem_first="$work/idempotent-first"
cp "$idem_out" "$idem_first"
draft_run_home "$out_home" -t "$idem_target" -o "$idem_out"
assert "-o re-run over its own output exits 0, as the default destination does" \
  test "$code" -eq 0
assert "-o re-run over its own output rewrote the draft rather than leaving the first one" \
  test -f "$idem_out"
assert "the re-run did not report that it could not tell where the draft would go" \
  test -z "$(printf '%s\n' "$errout" | grep -F 'could not be resolved' || true)"

# And the same property one step out: a destination that is somebody else's
# existing file is a destination, not a path with no answer. The drafter
# overwrites it, which is what `-o` promises — "anywhere else you can write, it
# will write" — and the refusals are a short list rather than a sandbox.
existing_target="$(target_from tests/fixtures/f01/valid-full existing-file-out)"
existing_out="$out_home/plain/notes-of-mine.md"
printf 'my own notes\n' > "$existing_out"
draft_run_home "$out_home" -t "$existing_target" -o "$existing_out"
assert "a destination that already exists as a regular file is written, not refused" \
  test "$code" -eq 0
assert "the draft replaced the file that was there" \
  grep -q 'Rewrite draft' "$existing_out"

# And the other half of the resolution rule, which is the one that cannot be
# got at by choosing a better destination: **a path that cannot be resolved
# counts as protected.** "I could not tell where this would land" must not read
# as "go ahead", so it is not a refusal of the caller's command line but a
# statement that no verdict was reached — status 3, the one this header
# registers for an execution error.
# Two assertions here are about *which* refusal happened, and they are not
# decoration. Removing the resolution refusal entirely still yields exit 3 for
# this destination — the decision lets the path through, the audit runs, and
# then `: >` fails to open it and the drafter reports an execution error from
# there. The status alone therefore cannot tell "I could not tell where this
# would land" from "I could not write there", and they are different answers:
# the first is a destination the caller should respell, the second is a
# permission to fix. Found by mutating the refusal away and watching the status
# assertion pass on its own, which is what the rest of this suite calls a check
# passing by not running.
#
# So the mechanism is pinned twice over, once on each channel available: the
# diagnostic says the destination could not be *resolved*, and the audit has not
# run — which it has, on the other path, because the write is the last thing
# this script does and the decision is one of the first.
unresolvable_target="$(target_from tests/fixtures/f01/valid-full unresolvable-out)"
locked="$out_home/locked"
mkdir -p "$locked/inner"
chmod 000 "$locked"
draft_run_home "$out_home" -t "$unresolvable_target" -o "$locked/inner/draft.md"
unresolvable_code="$code"
unresolvable_err="$errout"
chmod 755 "$locked"
assert "a destination whose path cannot be resolved is not accepted" \
  test "$unresolvable_code" -ne 0
assert "a destination that cannot be resolved exits 3, not 1: no verdict was reached" \
  test "$unresolvable_code" -eq 3
assert "the unresolvable destination is named in the diagnostic" \
  test -n "$(printf '%s\n' "$unresolvable_err" | grep -F "$locked/inner/draft.md" || true)"
assert "the diagnostic says the destination could not be resolved, not that it could not be written" \
  test -n "$(printf '%s\n' "$unresolvable_err" | grep -F 'could not be resolved' || true)"
assert "an unresolvable destination is refused before the audit runs" \
  test -z "$(printf '%s\n' "$unresolvable_err" | grep -F 'running structural checks' || true)"
assert "an unresolvable destination leaves no draft in the target directory" \
  test ! -f "$unresolvable_target/REWRITE-DRAFT.md"
# The premise that control rests on: the locked directory really was unreadable,
# or the assertions above are about an ordinary path.
assert "the locked directory was traversable again afterwards, so the control was about permissions" \
  test -d "$locked/inner"

# And the same rule reached by a different cause, so that it is the rule being
# held and not one instance of it. A symlink cycle in a directory component is
# ELOOP rather than EACCES: nothing about the path is forbidden, it simply has
# no answer.
cycle_target="$(target_from tests/fixtures/f01/valid-full cycle-out)"
ln -s "$out_home/loop-b" "$out_home/loop-a"
ln -s "$out_home/loop-a" "$out_home/loop-b"
draft_run_home "$out_home" -t "$cycle_target" -o "$out_home/loop-a/draft.md"
assert "a destination whose directory component is a symlink cycle exits 3" \
  test "$code" -eq 3
assert "the symlink cycle is reported as a destination that could not be resolved" \
  test -n "$(printf '%s\n' "$errout" | grep -F 'could not be resolved' || true)"
assert "a cyclic destination leaves no draft in the target directory" \
  test ! -f "$cycle_target/REWRITE-DRAFT.md"

# The scope of the decision, stated as an assertion because it is the part a
# reader is most likely to get wrong. The three refusals are about the
# destination `-o` names, not about every destination: with no `-o` the draft
# still goes beside the target's `SKILL.md`, and a target that itself lives
# inside a live config directory still drafts. The caller who named `-t` named a
# directory the drafter then verified holds a `SKILL.md`; the caller who names
# `-o` has asserted nothing the drafter can check. The asymmetry is in the
# evidence, not in the spelling.
in_config_target="$out_home/.claude/skills/valid-full"
mkdir -p "$out_home/.claude/skills"
cp -R tests/fixtures/f01/valid-full "$in_config_target"
draft_run_home "$out_home" -t "$in_config_target"
assert "a target inside a live config directory still drafts by default" \
  test "$code" -eq 0
assert "the default destination is still beside the target's SKILL.md" \
  test -f "$in_config_target/REWRITE-DRAFT.md"

# --- where the drafter finds skill-audit --------------------------------------
#
# The drafter borrows skill-audit's check scripts, and it resolves them from its
# own location so that the caller's working directory is not part of the answer.
# Asserted from a directory that is not the repository, by absolute path,
# because a resolution that silently collapses to the cwd passes every test run
# from the repository root.
elsewhere_target="$(target_from tests/fixtures/f01/valid-full elsewhere)"
drafts_from_any_cwd() {
  ( cd / && quietly "$harness_repo_root/$DRAFTER" -t "$elsewhere_target" )
}
assert "the drafter resolves skill-audit from its own location, not from the caller's cwd" \
  drafts_from_any_cwd
assert "a draft run from an unrelated cwd carries the audit, not a diagnostic about it" \
  grep -q 'frontmatter OK' "$elsewhere_target/REWRITE-DRAFT.md"

# --- the report the drafter makes for itself ----------------------------------
#
# With no `-a` the drafter runs the checks into a temp file. Two things are then
# true of that file and neither should be: it outlives the run, and its path —
# a machine-local name under whatever TMPDIR happened to be set — is written
# into the draft as the draft's stated provenance. A reader who follows that
# line finds nothing, and a reader who keeps the draft has a provenance that
# never meant anything outside the process that wrote it.
#
# The run is watched through a recording `mktemp` rather than through a
# directory, because the question is "did the drafter remove the file it made"
# and the answer has to hold wherever mktemp puts it. TMPDIR does not answer
# it: BSD mktemp ignores TMPDIR for a template-less call, so a TMPDIR-pointed
# check is an assertion that cannot fail on macOS — the one shape
# tests/test_harness.sh exists to refuse.
#
# The stub is a plain file in a directory of its own, prepended to PATH. It is
# not written into a masked-path symlink farm: a write through a symlink named
# for a real binary goes to the real binary.
mktemp_log="$work/mktemp-handed-out"
mktemp_stub_path() {
  local dir="$work/stub-mktemp"
  if [ ! -d "$dir" ]; then
    mkdir -p "$dir"
    {
      echo '#!/usr/bin/env bash'
      printf 'p="$(%s "$@")"\n' "$(command -v mktemp)"
      printf 'printf "%%s\\n" "$p" >> %s\n' "$mktemp_log"
      echo 'printf "%s\n" "$p"'
    } > "$dir/mktemp"
    chmod +x "$dir/mktemp"
  fi
  echo "$dir:$PATH"
}

draft_run_watching_mktemp() {
  : > "$mktemp_log"
  code=0
  output="$(PATH="$(mktemp_stub_path)" "$DRAFTER" "$@" 2>"$drafted/stderr")" || code=$?
}

# The paths mktemp handed out during the run that are still there afterwards.
left_behind() {
  local p
  while read -r p; do
    [ -n "$p" ] || continue
    [ -e "$p" ] && echo "$p"
  done < "$mktemp_log"
  return 0
}

# The control for the watcher itself: a stub that recorded nothing would make
# "nothing was left behind" true of every run, including a leaking one.
assert "the mktemp watcher records the paths a run was handed" \
  test -n "$(draft_run_watching_mktemp -t "$(target_from tests/fixtures/f01/valid-full watcher)" >/dev/null 2>&1; cat "$mktemp_log")"

provenance_of() {
  { grep -m1 -E '^Generated from' "$1/REWRITE-DRAFT.md" || true; }
}

own_report_target="$(target_from tests/fixtures/f01/valid-full own-report)"
draft_run_watching_mktemp -t "$own_report_target"
assert "a run with no -a still writes its draft" test "$code" -eq 0
assert "a run with no -a leaves no temp file behind" test -z "$(left_behind)"
assert "a run with no -a states a provenance" test -n "$(provenance_of "$own_report_target")"
assert "a run with no -a does not name a temp path as the draft's provenance" \
  test -z "$(provenance_of "$own_report_target" | grep -F -f "$mktemp_log" || true)"
assert "a run with no -a names the checks the draft was built from" \
  test -n "$(provenance_of "$own_report_target" | grep -F 'check-structure.sh' || true)"

# The control: when the caller did name a report, that path is the provenance
# and saying so is right. The assertion above must not be satisfiable by
# dropping the provenance line altogether.
named_report_target="$(target_from tests/fixtures/f01/valid-full named-report)"
named_report="$work/named-audit.md"
printf 'frontmatter OK\n' > "$named_report"
draft_run_watching_mktemp -t "$named_report_target" -a "$named_report"
assert "a run with -a names the report the caller gave as the draft's provenance" \
  test -n "$(provenance_of "$named_report_target" | grep -F "$named_report" || true)"
assert "a run with -a leaves no temp file behind either" test -z "$(left_behind)"

# --- the tools this skill needs -----------------------------------------------
#
# Discovered by masking each candidate in turn and watching what the drafter
# can still produce, not read off the source. Reading the source over-credits:
# `require_tool jq true` in check-structure.sh only fires in `--json` mode, and
# these stages use none, so a source census would have this skill declare a
# prerequisite it does not have — and a declaration that is only ever too
# generous still forces the documentation to say something untrue.
#
# The denominator is the `require_tool` calls in skill-audit's scripts *and* in
# the drafter: the set of tools either states a precondition on, and so the
# whole set that can stop this skill for want of a tool. It is read rather than
# listed, so a tool added in either place cannot fall out of this census.
#
# The drafter is on that list because it now states preconditions of its own.
# While it stated none, skill-audit's scripts were the whole set and reading
# only them was the same answer; `awk`, `grep`, `cat` and `mktemp` are tools the
# drafter computes with and two of them no sibling check asks for, so the
# narrower denominator would now leave this skill free to require a tool its
# documentation never names.
#
# verdict-guard.sh is skipped, for the reason tests/lib/audit-suites.sh is
# never run over tests/lib/harness.sh: it is where `require_tool` is defined
# and documented, not where a tool is asked for, and reading it yields its own
# parameter name and the names of its sibling guards as if they were tools.
candidate_tools() {
  local f
  for f in skills/skill-audit/scripts/*.sh "$DRAFTER"; do
    case "${f##*/}" in verdict-guard.sh) continue ;; esac
    { grep -hoE 'require_tool[[:space:]]+[a-z][a-z0-9-]*' "$f" || true; }
  done | awk '{print $NF}' | sort -u
}

# A tool is required on this skill's path when masking it stops the drafter
# from producing the audit. Both channels are read: the guard's message reaches
# the draft today, through the `2>&1` that folds the child's stderr into the
# report, and reaches stderr once cluster C3 brings this script inside the
# guard. The question is the same either way.
tool_is_required() {
  local tool="$1"
  local target
  target="$(target_from tests/fixtures/f01/valid-full "need-$tool")"
  run_masked "$tool" "$DRAFTER" -t "$target"
  if [ "$code" -ne 0 ]; then
    return 0
  fi
  if [ -f "$target/REWRITE-DRAFT.md" ] \
     && grep -qF "not found: $tool" "$target/REWRITE-DRAFT.md"; then
    return 0
  fi
  printf '%s\n' "$errout" | grep -qF "not found: $tool"
}

required_tools() {
  local t
  for t in $(candidate_tools); do
    if tool_is_required "$t"; then
      echo "$t"
    fi
  done
}

# The registry line. One physical line, read for the tools set in code on it,
# in the shape tests/test_f01.sh reads skill-audit's `Exit codes:` line: the
# backticks are what make a name a declaration rather than a word in a
# sentence, so the paragraph under it can discuss a tool without declaring it.
declared_tools() {
  { grep -m1 -E '^Required tools:' "$SKILL" || true; } \
    | { grep -oE '`[a-z][a-z0-9-]+`' || true; } | tr -d '`' | sort -u
}

assert "the SKILL.md declares a required tool at all" test -n "$(declared_tools)"
assert "the candidate set is read from skill-audit rather than being empty" \
  test -n "$(candidate_tools)"
assert "the SKILL.md declares every tool this skill needs, and none it does not" \
  test "$(required_tools)" = "$(declared_tools)"
if [ "$(required_tools)" != "$(declared_tools)" ]; then
  echo "  candidates: $(candidate_tools | tr '\n' ' ')"
  echo "  required  : $(required_tools | tr '\n' ' ')"
  echo "  declared  : $(declared_tools | tr '\n' ' ')"
fi

# --- and what happens when the tool is not there ------------------------------
#
# Naming the tool is half of a prerequisite. The other half is what the reader
# gets when it is missing, and this skill's two paths do not answer that the
# same way: `check-frontmatter.sh` run directly refuses to report a verdict it
# could not compute, and the same check run through `draft-rewrite.sh` — the
# command the paragraph's readers actually run — does not. Nothing here held
# that paragraph, so it was free to describe the guarded path and leave the
# reader to meet the other one.
#
# So the claim is written on one physical line in code form, the shape
# tests/test_f01.sh reads skill-audit's `Exit codes:` line in, and each half is
# compared with a run. The document decides which commands are on the line and
# what status each exits with; the invocation shape is the suite's to know,
# since the drafter takes `-t <dir>` and the check takes the directory.
masked_status_claims() {
  { grep -m1 -E '^Without `skill-validator`:' "$SKILL" || true; } \
    | { grep -oE '`[a-z][a-z0-9-]*\.sh` exits `[0-9]+`' || true; } \
    | tr -d '`' | sed -e 's/ exits / /'
}
claimed_status_of() {
  masked_status_claims | awk -v c="$1" '$1 == c { print $2 }'
}

# Sets `code`, `output`, `errout` and `no_validator_target`.
run_without_validator() {
  local script="$1"
  no_validator_target="$(target_from tests/fixtures/f01/valid-full "no-validator-${script%.sh}")"
  case "$script" in
    draft-rewrite.sh)
      run_masked skill-validator "$DRAFTER" -t "$no_validator_target" ;;
    *)
      run_masked skill-validator "skills/skill-audit/scripts/$script" "$no_validator_target" ;;
  esac
}

# Read by two sections: here, for the diagnostic the drafter folds into the
# draft, and below, for what the audit said about two different targets.
current_state_of() {
  sed -n '/^## Current state$/,/^## Proposed structure$/p' "$1/REWRITE-DRAFT.md"
}

assert "the SKILL.md states what each of its commands does without skill-validator" \
  test -n "$(masked_status_claims)"
for script in $(masked_status_claims | awk '{print $1}'); do
  run_without_validator "$script"
  assert "without skill-validator $script exits $(claimed_status_of "$script"), the status the SKILL.md gives it" \
    test "$code" -eq "$(claimed_status_of "$script")"
done

# The rest of that paragraph, which used to be about the path that did not
# refuse: the drafter drafted anyway, and the check's diagnostic became the
# draft's own `Current state` rather than reaching stderr. Both halves were
# written in the present tense so that they would go red when cluster C3
# brought the drafter inside the guard, rather than leave the paragraph
# describing a behaviour the skill no longer has. C3 has landed and they did,
# so they are asserted the other way round here and the paragraph now says the
# drafter refuses — which is also why the inventory below no longer describes
# `Current state` as a merged stream.
#
# Still two assertions, because "it refuses" is two claims: that the reader is
# told which tool is missing, and that nothing was left in their skill
# directory. A drafter that printed the refusal and kept the partial draft
# would satisfy one of them, and that partial draft is what the trap in the
# drafter exists to remove.
run_without_validator draft-rewrite.sh
assert "without skill-validator the drafter writes no draft, as the SKILL.md says it does not" \
  test ! -f "$no_validator_target/REWRITE-DRAFT.md"
assert "without skill-validator the missing tool is named on stderr, as the SKILL.md says" \
  test -n "$(printf '%s\n' "$errout" | grep -F 'required tool not found: skill-validator' || true)"

# --- the sibling skill this one is not without --------------------------------
#
# skill-rewrite bundles one script and borrows every check it runs from the
# sibling skill-audit, verdict-guard.sh included: every one of those scripts
# refuses to compute a verdict without it. A reader who installs this skill
# alone, or who prunes what looks like an unreferenced file out of the sibling,
# gets a draft built from nothing — so the dependency is a thing the SKILL.md
# has to name.
#
# Proved on a copy of both skills with the guard removed, because the drafter
# resolves the sibling from its own location: the copy is the only way to ask
# the question without touching the repository's own tree.
pruned="$work/pruned"
rm -rf "$pruned"
mkdir -p "$pruned"
cp -R skills "$pruned/skills"
rm -f "$pruned/skills/skill-audit/scripts/verdict-guard.sh"
pruned_target="$(target_from tests/fixtures/f01/valid-full pruned-target)"
run_present "$pruned/skills/skill-rewrite/scripts/draft-rewrite.sh" -t "$pruned_target"
guard_is_load_bearing() {
  [ ! -f "$pruned_target/REWRITE-DRAFT.md" ] \
    || ! grep -qF 'frontmatter OK' "$pruned_target/REWRITE-DRAFT.md"
}
assert "without the sibling's verdict-guard.sh the drafter cannot produce the audit" \
  guard_is_load_bearing
# What the dependency paragraph claims, each half decided rather than greped
# for. It says this skill bundles one script and runs every check from the
# sibling's directory, and names which two: three comparisons between the
# document and the tree, so neither side can move without the other.
#
# The document's side is read by the same extractor the documented-path census
# uses, which reads bash fences and skips ```text — a command is written in a
# fence, and a script merely discussed in a sentence is not a claim that this
# skill runs it.
bundled_scripts() {
  { ls skills/skill-rewrite/scripts/ || true; } | sort -u
}
documented_bundled_scripts() {
  doc_referenced_paths "$SKILL" \
    | { grep -E '^\$\{?skill_root\}?/scripts/' || true; } \
    | sed -e 's|.*/||' | sort -u
}
# verdict-guard.sh is skipped here for the reason candidate_tools above skips
# it. The drafter names it under the same `$skill_audit_root/scripts/` prefix as
# the two checks, but it sources the guard rather than running it — that is how
# it refuses in the guard's own words — and a library it loads is not a check it
# runs. Counted as one, this census would require the SKILL.md to show a reader
# invoking the guard, which is not a command anybody invokes. The dependency
# itself is held two assertions down, by removing the guard from a copy of both
# skills and watching the drafter stop.
sibling_checks_the_drafter_runs() {
  { grep -hoE '\$\{?skill_audit_root\}?/scripts/[A-Za-z0-9_.-]+\.sh' "$DRAFTER" || true; } \
    | sed -e 's|.*/||' | { grep -vxF 'verdict-guard.sh' || true; } | sort -u
}
documented_sibling_checks() {
  doc_referenced_paths "$SKILL" \
    | { grep -E '^\$\{?audit_root\}?/scripts/' || true; } \
    | sed -e 's|.*/||' | sort -u
}
every_check_it_runs_loads_the_guard() {
  local s
  for s in $(sibling_checks_the_drafter_runs); do
    grep -qF 'verdict-guard.sh' "skills/skill-audit/scripts/$s" || return 1
  done
  [ -n "$(sibling_checks_the_drafter_runs)" ]
}

assert "the scripts this skill bundles are the ones the SKILL.md shows it running, and no others" \
  test "$(bundled_scripts)" = "$(documented_bundled_scripts)"
assert "the sibling checks the drafter runs are the ones the SKILL.md shows it running, and no others" \
  test "$(sibling_checks_the_drafter_runs)" = "$(documented_sibling_checks)"
assert "every check this skill runs loads verdict-guard.sh, as the SKILL.md says both of them do" \
  every_check_it_runs_loads_the_guard

# The compatibility line this skill declares, and the refusal of any claim that
# a file is named nowhere else, both used to live here reading this SKILL.md
# alone. Both are claims every shipped skill makes, so both moved to
# tests/test_skill.sh and are walked over `skills/*/`: held here, skill-audit's
# identical compatibility line was un-held and went on being false.

# --- the draft the SKILL.md describes -----------------------------------------
#
# The inventory the document gives, against the headings a run actually writes.
# This is the comparison the cluster turns on: the document promised preserved
# frontmatter, five section templates and a checklist mapping each failed audit
# dimension to a fix, and the script writes none of the first, three of the
# second — one of which the document does not list — and a fixed five-item
# boilerplate for the third.
#
# Two targets, because the drafter's only variation is three probes for
# headings the target may already have. tests/fixtures/rewrite/all-sections
# holds all three, so no template fires; tests/fixtures/f01/valid-minimal holds
# none, so every template fires. Each probe is therefore observed in both
# directions, and their union is the whole set of headings the script can
# write — which is what makes the comparison below a comparison over the set
# rather than over one path through it.
headings_of() {
  { grep -E '^#{2,4} ' "$1/REWRITE-DRAFT.md" || true; } | sort -u
}

no_templates_target="$(target_from tests/fixtures/rewrite/all-sections no-templates)"
draft_run -t "$no_templates_target"
assert "a target holding every probed heading still gets a draft" test "$code" -eq 0

all_templates_target="$(target_from tests/fixtures/f01/valid-minimal all-templates)"
draft_run -t "$all_templates_target"
assert "a target holding none of them gets a draft too" test "$code" -eq 0

# The control on the fixture pair. If both targets drew the same templates, the
# union below would be one path's headings and the inventory could agree with it
# while the document stayed wrong about the conditional ones.
assert "the two targets draw different templates, so both directions of each probe are observed" \
  test "$(headings_of "$no_templates_target")" != "$(headings_of "$all_templates_target")"

emitted_sections() {
  { headings_of "$no_templates_target"; headings_of "$all_templates_target"; } | sort -u
}

# The inventory: the ```text block under the heading that introduces it. Read as
# a block of headings rather than as prose, so the document cannot list a
# section in a sentence the comparison does not see.
documented_sections() {
  fenced_block_under "$SKILL" '^#### The sections the drafter writes' \
    | { grep -E '^#{2,4} ' || true; } | sort -u
}

assert "the SKILL.md carries an inventory of the draft's sections" \
  test -n "$(documented_sections)"
assert "the drafter writes a section at all, so the comparison has two sides" \
  test -n "$(emitted_sections)"
assert "the SKILL.md inventories every section the drafter writes, and none it does not" \
  test "$(emitted_sections)" = "$(documented_sections)"
if [ "$(emitted_sections)" != "$(documented_sections)" ]; then
  echo "  written    : $(emitted_sections | tr '\n' '|')"
  echo "  inventoried: $(documented_sections | tr '\n' '|')"
fi

# The three claims the document made that the script does not keep, each asked
# of the artifact rather than of the prose. They are asserted as absences on
# purpose: 0.4.3 corrected the description, and if the capability is ever built
# these three go red, which is the point — they are what stops the document
# being left describing the old draft a second time.
#
# This comment used to schedule the capability into the release now being cut,
# which became a false statement about a shipped release: 0.5.0 did not build
# it. It also contradicted the document it is written to hold.
# `skills/skill-rewrite/SKILL.md`
# states the same gap as a boundary rather than a schedule — "**What the drafter
# does not do**, and what Stage 3 is therefore for … Scoring the dimensions and
# turning them into fixes is Stage 1 and Stage 3 work, done by the reader" — and
# the shipped document is the one that defines what this skill promises. Nothing
# in the repository commits to moving that boundary: there is no open ledger row
# for it, and the two places that scheduled it are 0.4.3's own text, which is
# history. So this comment names no release. It says what is true of the
# artifact and cites the sentence it is holding, which is what neither side did.
assert "the draft does not carry the target's frontmatter, so the document must not promise it" \
  test -z "$(grep -m1 -F 'name: all-sections' "$no_templates_target/REWRITE-DRAFT.md" || true)"

# The list of what the reader is left to write themselves, against the draft
# itself. Three named sections were asserted one by one and the document listed
# the same three — and the answer is four: the draft's own `Proposed structure`
# requires six sections, the drafter templates two of them, and `AI judgment`
# fell out of the sentence that tells the reader which ones are theirs.
#
# So it is a comparison over the set rather than three assertions over three
# literals, and the set is computed: a section the drafter learns to write
# leaves the two sides unequal, whichever side is edited first. The target that
# holds none of the probed headings is the one read, because it draws every
# template there is.
#
# The percentages are dropped from the required names: `Deterministic actions
# (60%)` is a heading the spec writes with its ratio and a sentence names
# without it, and the ratio is not what either side is claiming here.
required_sections_of() {
  { grep -oE '^ +- `#{2,4} [^`]+`' "$1/REWRITE-DRAFT.md" || true; } \
    | sed -e 's|.*`#* ||' -e 's|`$||' -e 's| ([0-9]*%)$||' | sort -u
}
templated_sections_of() {
  headings_of "$1" | sed -e 's|^#* ||' | sort -u
}
untemplated_required_sections() {
  comm -23 \
    <(required_sections_of "$all_templates_target") \
    <(templated_sections_of "$all_templates_target")
}
sections_the_document_leaves_to_the_reader() {
  tr '\n' ' ' < "$SKILL" \
    | { grep -oE 'writes no template for [^.]*' || true; } \
    | { grep -oE '`[^`]+`' || true; } | tr -d '`' | sort -u
}

assert "the draft requires sections the drafter writes no template for, so the comparison has two sides" \
  test -n "$(untemplated_required_sections)"
assert "the SKILL.md names the sections the reader is left to write" \
  test -n "$(sections_the_document_leaves_to_the_reader)"
assert "the SKILL.md names every required section the drafter writes no template for, and none it does write" \
  test "$(untemplated_required_sections)" = "$(sections_the_document_leaves_to_the_reader)"
if [ "$(untemplated_required_sections)" != "$(sections_the_document_leaves_to_the_reader)" ]; then
  echo "  untemplated: $(untemplated_required_sections | tr '\n' ' ')"
  echo "  documented : $(sections_the_document_leaves_to_the_reader | tr '\n' ' ')"
fi

# And the checklist. Byte-identical for a target that audits clean and one that
# fails, which is the whole of the claim that it maps each failed dimension to
# a fix.
checklist_of() {
  sed -n '/^## Action items$/,/^## Notes$/p' "$1/REWRITE-DRAFT.md"
}
assert "the audit reported different things about the two targets" \
  test "$(current_state_of "$no_templates_target")" != "$(current_state_of "$all_templates_target")"
assert "the Action items checklist is the same text whatever the audit found" \
  test "$(checklist_of "$no_templates_target")" = "$(checklist_of "$all_templates_target")"
assert "the checklist read is not empty, so its sameness means something" \
  test -n "$(checklist_of "$no_templates_target")"
assert "the drafter never reads skill-audit's evaluation matrix" \
  test -z "$(grep -F 'evaluation-matrix' "$DRAFTER" || true)"

# The draft's title names the skill it is for. Level 1, so the inventory above
# does not cover it, and target-dependent, so it cannot be a listed literal.
assert "the draft's title names the target skill" \
  grep -qF '# Rewrite draft: all-sections' "$no_templates_target/REWRITE-DRAFT.md"

# --- how much of a heading a probe has to see ---------------------------------
#
# The probes are not all the same shape, and the document used to describe them
# as though they were. `Validation` is matched as a prefix, so a target's own
# `### Validation` — which is what the all-sections fixture carries, and how it
# suppresses the third template — is enough; `When to use` and
# `Example`/`Examples` have to be the whole heading, so a heading that merely
# begins with one of them suppresses nothing.
#
# Both directions are asked of a target rather than read off the probe, because
# the difference is only visible in what a run draws. The second target is
# built here rather than added to tests/fixtures: it is this fixture plus one
# heading, and the heading is the whole of the question.
assert "the all-sections fixture's Validation heading is not the template's own name" \
  grep -qE '^### Validation$' tests/fixtures/rewrite/all-sections/SKILL.md
assert "a heading that only begins with Validation suppresses the Validation checklist template" \
  test -z "$(grep -E '^#{3,6} Validation checklist' "$no_templates_target/REWRITE-DRAFT.md" || true)"

examples_prefix_target="$(target_from tests/fixtures/f01/valid-minimal examples-prefix)"
printf '\n## Examples of what not to do\n\nNothing here.\n' >> "$examples_prefix_target/SKILL.md"
draft_run -t "$examples_prefix_target"
assert "the built target's heading only begins with Examples, so the probe has the case to decide" \
  grep -qE '^## Examples of what not to do$' "$examples_prefix_target/SKILL.md"
assert "a heading that only begins with Examples does not suppress the Examples template" \
  grep -qE '^#{3,6} Examples$' "$examples_prefix_target/REWRITE-DRAFT.md"

harness_summary
