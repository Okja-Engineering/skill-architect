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
  local file="$1" heading="$2" target="$3" report="${4:-}" block
  block="$(fenced_block_under "$file" "$heading")" || return 1
  printf '%s\n' "$block" \
    | sed -e "s|<path-to-skill-architect>|$harness_repo_root|g" \
          -e "s|<target-skill-dir>|$target|g" \
          -e "s|<audit-report-path>|$report|g"
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
printf 'frontmatter OK\n' > "$stage2_report"
assert "the documented Stage 2 drafter block runs as written, from the repository root" \
  run_doc_block "$SKILL" '^### Stage 2' "$stage2_target" "$stage2_report"
assert "the documented Stage 2 block wrote the draft it says it writes" \
  test -f "$stage2_target/REWRITE-DRAFT.md"

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
statuses_emitted() {
  { grep -hoE '(^|[^[:alnum:]_])exit[[:space:]]+[0-9]+' "$DRAFTER" || true; } \
    | grep -oE '[0-9]+' | sort -u
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
for flag in -t --target -a --audit; do
  draft_run "$flag"
  assert "$flag with no value exits 1, the status the drafter's header registers for a usage error" \
    test "$code" -eq 1
  assert "$flag with no value names the option rather than a shell positional" \
    test -z "$(grep -F 'unbound variable' "$drafted/stderr" || true)"
  assert "$flag with no value says which option is missing a value" \
    grep -qF -- "$flag" "$drafted/stderr"
done

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
# The denominator is skill-audit's own `require_tool` calls: the set of tools
# its scripts state a precondition on, and so the whole set that can stop this
# skill for want of a tool. It is read rather than listed, so a tool added
# there cannot fall out of this census.
#
# verdict-guard.sh is skipped, for the reason tests/lib/audit-suites.sh is
# never run over tests/lib/harness.sh: it is where `require_tool` is defined
# and documented, not where a tool is asked for, and reading it yields its own
# parameter name and the names of its sibling guards as if they were tools.
candidate_tools() {
  local f
  for f in skills/skill-audit/scripts/*.sh; do
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

# The rest of that paragraph, about the path that does not refuse: the drafter
# drafts anyway, and the check's diagnostic becomes the draft's own `Current
# state` rather than reaching stderr — which is also why the inventory below
# describes `Current state` as a merged stream and not as the checks' output
# verbatim.
#
# Asserted in the present tense, so that when cluster C3 brings the drafter
# inside the guard these go red and the paragraph has to be rewritten, rather
# than being left describing a behaviour the skill no longer has.
run_without_validator draft-rewrite.sh
assert "without skill-validator the drafter writes a draft anyway, as the SKILL.md warns it does" \
  test -f "$no_validator_target/REWRITE-DRAFT.md"
assert "without skill-validator the check's diagnostic is the draft's Current state, as the SKILL.md warns" \
  test -n "$(current_state_of "$no_validator_target" | grep -F 'required tool not found: skill-validator' || true)"

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
sibling_checks_the_drafter_runs() {
  { grep -hoE '\$\{?skill_audit_root\}?/scripts/[A-Za-z0-9_.-]+\.sh' "$DRAFTER" || true; } \
    | sed -e 's|.*/||' | sort -u
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

# A claim about the rest of the repository is decidable against the rest of the
# repository, and this one was not decided. The document called
# verdict-guard.sh a file "which no other file references by name"; a recursive
# grep finds the name in twelve files, including four of skill-audit's own
# scripts, the readme, two suites and the drafter. The assertion covering the
# sentence only checked that the filename appeared *in the document*, so it
# passed over the falsehood — a string where there should have been a claim.
#
# Read as a claim and not as a phrase: a backticked filename with, in the same
# sentence, a denial that anything else names it.
unreferenced_claims_in() {
  tr '\n' ' ' < "$1" \
    | { grep -oE '`[A-Za-z0-9_.-]+\.(sh|md)`[^.]*(no other file|no other script|nothing else|referenced nowhere|unreferenced)[^.]*' || true; } \
    | { grep -oE '^`[^`]+`' || true; } | tr -d '`' | sort -u
}

# Every file in the repository that names <1>, other than <2>. `.git` is not
# part of the repository's text, and the leading `./` is dropped because not
# every grep prints it.
files_naming_other_than() {
  { grep -rlF "$1" --exclude-dir=.git . || true; } \
    | sed -e 's|^\./||' | { grep -vxF "$2" || true; } | sort -u
}

unreferenced_claims_hold_in() {
  local doc="$1" f others bad=0
  for f in $(unreferenced_claims_in "$doc"); do
    others="$(files_naming_other_than "$f" "$doc")"
    if [ -n "$others" ]; then
      echo "  $doc says nothing else names $f; these files do: $(printf '%s' "$others" | tr '\n' ' ')" >&2
      bad=$((bad + 1))
    fi
  done
  [ "$bad" -eq 0 ]
}

assert "no claim in the SKILL.md that a file is named nowhere else survives a grep of the repository" \
  unreferenced_claims_hold_in "$SKILL"

# The control, which is the sentence as it was written, in a document of its
# own. Once the claim is gone the assertion above has nothing to decide, and a
# reader that found no claim would look exactly the same — so the reader is
# held to finding this one, and to refusing it.
control_claim_doc="$work/control-unreferenced.md"
printf 'It borrows every check from `$audit_root/scripts/`, including `verdict-guard.sh`, which no other file references by name and which every one of those scripts refuses to compute a verdict without.\n' \
  > "$control_claim_doc"
assert "the claim reader finds the claim in the sentence this document carried" \
  test "$(unreferenced_claims_in "$control_claim_doc")" = "verdict-guard.sh"
control_claim_is_refused() {
  ! unreferenced_claims_hold_in "$control_claim_doc" 2>/dev/null
}
assert "a document claiming verdict-guard.sh is named nowhere else is refused" \
  control_claim_is_refused

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
# purpose: 0.4.3 corrects the description, and building the capability is
# 0.5.0. When it is built these three go red, which is the point — they are
# what will stop the document being left describing the old draft a second
# time.
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

# --- the compatibility the skill actually has ---------------------------------
#
# A `compatibility:` line is a promise to whoever is deciding whether to install
# the skill, and it is the one claim in the frontmatter nothing was checking.
# Three ways to break it, so three checks, each decidable everywhere rather than
# only on the machine the suite happens to run on.
compatibility_line() {
  { grep -m1 -E '^compatibility:' "$SKILL" || true; }
}

# "POSIX shell" is read as a claim of `sh`, because that is what it means to
# whoever is deciding whether to install: the word does not have to appear for
# the claim to have been made, and check (3) below is where it is answered.
claimed_interpreters_in() {
  {
    printf '%s\n' "$1" \
      | { grep -oE '(^|[^a-z-])(sh|bash|zsh|ksh|dash|fish)([^a-z-]|$)' || true; } \
      | { grep -oE '(sh|bash|zsh|ksh|dash|fish)' || true; }
    if printf '%s\n' "$1" | grep -qi 'POSIX'; then echo sh; fi
  } | sort -u
}
claimed_interpreters() { claimed_interpreters_in "$(compatibility_line)"; }

# The commands named on the line that are not interpreters: a compatibility
# line that names a tool is stating a dependency on it.
claimed_commands_in() {
  local w out=""
  for w in $(printf '%s\n' "$1" | tr -cs 'a-zA-Z0-9_-' ' '); do
    case " $(claimed_interpreters_in "$1" | tr '\n' ' ') " in *" $w "*) continue ;; esac
    if command -v "$w" >/dev/null 2>&1; then
      out="$out$w
"
    fi
  done
  printf '%s' "$out" | sort -u
}
claimed_commands() { claimed_commands_in "$(compatibility_line)"; }

# (1) Behavioural: every interpreter the line claims runs the drafter to a
# draft that carries the audit. Not merely "exits 0" — under zsh the drafter
# exits 0 having written a draft whose Current state is two file-not-found
# lines, and an exit-status check would call that compatibility.
runs_the_drafter() {
  local interp="$1" target
  command -v "$interp" >/dev/null 2>&1 || return 1
  target="$(target_from tests/fixtures/rewrite/all-sections "under-$interp")"
  "$interp" "$DRAFTER" -t "$target" >/dev/null 2>&1 || return 1
  [ -f "$target/REWRITE-DRAFT.md" ] || return 1
  grep -qF 'frontmatter OK' "$target/REWRITE-DRAFT.md"
}

assert "the compatibility line names an interpreter at all" \
  test -n "$(claimed_interpreters)"
for interp in $(claimed_interpreters); do
  assert "the skill works under $interp, which its compatibility line claims" \
    runs_the_drafter "$interp"
done

# (2) Every tool the line names is a tool the skill needs. `git` was on it and
# no script in either skill runs git.
for tool in $(claimed_commands); do
  assert "the compatibility line's $tool is a tool this skill actually needs" \
    tool_is_required "$tool"
done
# The control for the command extractor, which reads nothing once the line is
# corrected. Without it, "every tool named is needed" would be true of a line
# naming anything at all.
assert "the compatibility-line reader finds a tool named on such a line" \
  test "$(claimed_commands_in 'compatibility: POSIX shell (bash 3.2+ or zsh), git.')" = "git"

# (3) Declarative: the line claims the interpreter the bundled script's shebang
# names, and no other family. This is what makes "POSIX shell" answerable — a
# behavioural `sh` run cannot answer it, because /bin/sh is bash in sh mode on
# macOS and dash on Linux, so the same assertion passes here and fails there.
shebang_interpreter() {
  sed -n '1s|^#!.*[/ ]\([a-z]*sh\)[[:space:]]*$|\1|p' "$DRAFTER"
}
assert "the bundled script's shebang names an interpreter" \
  test -n "$(shebang_interpreter)"
assert "the compatibility line claims the shebang's interpreter and no other family" \
  test "$(claimed_interpreters)" = "$(shebang_interpreter)"
if [ "$(claimed_interpreters)" != "$(shebang_interpreter)" ]; then
  echo "  claimed: $(claimed_interpreters | tr '\n' ' ')"
  echo "  shebang: $(shebang_interpreter)"
fi

# The version half of the claim, pinned where it can be pinned. CI runs one
# Linux job with one bash, so "3.2+" is a claim no run verifies; what a run can
# verify is that the script uses nothing bash 3.2 lacks.
assert "the drafter uses no construct bash 3.2 does not have" \
  test -z "$(grep -nE 'declare -A|mapfile|readarray|local -n|wait -n|globstar|\$\{[A-Za-z_][A-Za-z_0-9]*,,\}|\$\{[A-Za-z_][A-Za-z_0-9]*\^\^\}' "$DRAFTER" || true)"

harness_summary
