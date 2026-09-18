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
control_doc="$work/control-SKILL.md"
printf 'Run it:\n\n```bash\nscripts/no-such-script.sh\n```\n' > "$control_doc"
assert "the documented-path check refuses a bare path that does not resolve" \
  test ! -e "$(resolve_doc_path scripts/no-such-script.sh)"
assert "the documented-path check reads a bash fence rather than skipping it" \
  test "$(doc_referenced_paths "$control_doc")" = "scripts/no-such-script.sh"

# --- the documented invocation ------------------------------------------------
#
# Rendered and run, not read. "The documented invocation works" is not a
# question about the text, and the one way to ask it is to execute what the
# document tells the reader to execute.
#
# The substitutions are the document's own `<...>` placeholder convention and
# nothing else: anything the block needs that the document does not offer a
# placeholder for is a thing the reader does not have either.
doc_block() {
  local heading="$1" target="$2" report="${3:-}"
  awk -v h="$heading" '
    !seen && $0 ~ h { seen = 1; next }
    seen && /^```/  { if (fence) exit; fence = 1; next }
    seen && fence   { print }
  ' "$SKILL" \
    | sed -e "s|<path-to-skill-architect>|$harness_repo_root|g" \
          -e "s|<target-skill-dir>|$target|g" \
          -e "s|<audit-report-path>|$report|g"
}

run_doc_block() {
  local script="$work/doc-block.sh"
  { echo 'set -eu'; doc_block "$@"; } > "$script"
  quietly bash "$script"
}

stage1_target="$(target_from tests/fixtures/f01/valid-full stage1)"
assert "the documented Stage 1 audit block runs as written, from the repository root" \
  run_doc_block '^### Stage 1' "$stage1_target"

stage2_target="$(target_from tests/fixtures/f01/valid-full stage2)"
stage2_report="$work/stage2-audit.md"
printf 'frontmatter OK\n' > "$stage2_report"
assert "the documented Stage 2 drafter block runs as written, from the repository root" \
  run_doc_block '^### Stage 2' "$stage2_target" "$stage2_report"
assert "the documented Stage 2 block wrote the draft it says it writes" \
  test -f "$stage2_target/REWRITE-DRAFT.md"

example_target="$(target_from tests/fixtures/f01/valid-full example)"
example_report="$work/example-audit.md"
printf 'frontmatter OK\n' > "$example_report"
assert "the worked example in Examples runs as written, from the repository root" \
  run_doc_block '^### Generate a rewrite draft' "$example_target" "$example_report"

# The control: the same machinery, on a block that names a script that is not
# there. A renderer that silently produced an empty script would report every
# block above as running.
control_skill="$work/control-block-SKILL.md"
printf '### Stage 2\n\n```bash\n"<path-to-skill-architect>/skills/skill-rewrite/scripts/absent.sh"\n```\n' \
  > "$control_skill"
control_block_runs() {
  local script="$work/control-block.sh"
  {
    echo 'set -eu'
    awk '!seen && /^### Stage 2/ { seen = 1; next }
         seen && /^```/ { if (fence) exit; fence = 1; next }
         seen && fence { print }' "$control_skill" \
      | sed -e "s|<path-to-skill-architect>|$harness_repo_root|g"
  } > "$script"
  quietly bash "$script"
}
block_runner_refuses_absent_script() {
  ! control_block_runs
}
assert "the block runner reports a block naming a script that is not there" \
  test ! -e "$harness_repo_root/skills/skill-rewrite/scripts/absent.sh"
assert "the block runner fails such a block rather than passing an empty render" \
  block_runner_refuses_absent_script

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
for flag in -t --target -a --audit; do
  draft_run "$flag"
  assert "$flag with no value exits 1 rather than aborting on an unbound variable" \
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

harness_summary
