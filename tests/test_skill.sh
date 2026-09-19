#!/usr/bin/env bash
set -euo pipefail
. "$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)/lib/harness.sh"
harness_init

# The scratch directory belongs to the shell that installed the trap, which is
# the harness. A helper that made its own could be running inside a command
# substitution, where the assignment never escapes and the trap has nothing to
# clean up.
tmp_skill="$harness_scratch/skill"
mkdir -p "$tmp_skill"

# masked_path / run_on_path / run_masked / run_present live in the shared
# harness. A masked PATH is a symlink farm of the real PATH minus one binary:
# nothing is deleted, moved or uninstalled, and nothing is written into it.
mask_root="$harness_scratch/mask"
mkdir -p "$mask_root"
source tests/lib/masked-path.sh

# The shipped skills, derived from the tree rather than written down.
#
# This is the denominator every claim below is walked over, and it is the reason
# the walk exists: a compatibility line and a claim-shape refusal were both
# built for one of the two shipped skills and pinned to that skill's own
# SKILL.md, so the identical sentence next door was un-held and went on being
# false. A written-down pair reproduces that the day a third skill lands.
shipped_skills() {
  local d
  for d in skills/*/; do
    d="${d%/}"
    [ -f "$d/SKILL.md" ] || continue
    printf '%s\n' "${d##*/}"
  done
}
shipped_skill_count="$({ shipped_skills | grep -c . || true; } | tr -d '[:space:]')"
skill_md_of() { printf 'skills/%s/SKILL.md' "$1"; }

manifest_is_valid() {
  python3 - "$1" <<'PY'
import json, sys
with open(sys.argv[1]) as f:
    data = json.load(f)
assert data['name'] == 'skill-architect', 'name mismatch'
assert data['version'] == '0.4.2', 'version mismatch'
assert 'skills' in data, 'missing skills'
PY
}

marketplace_is_valid() {
  python3 - <<'PY'
import json
with open('.claude-plugin/marketplace.json') as f:
    data = json.load(f)
assert data['name'] == 'skill-architect', 'marketplace name mismatch'
plugins = data['plugins']
assert len(plugins) == 1, 'expected exactly one plugin entry'
assert plugins[0]['name'] == 'skill-architect', 'plugin entry name mismatch'
assert plugins[0]['version'] == '0.4.2', 'plugin entry version mismatch'
assert plugins[0]['source'] == './', 'plugin entry source mismatch'
PY
}

frontmatter_name_matches_directory() {
  local skill="$1"
  local name_in_file
  name_in_file="$(awk '
    BEGIN { in_fm = 0 }
    /^---$/ {
      if (in_fm) { exit }
      in_fm = 1
      next
    }
    in_fm && /^name:/ { sub(/^name:[[:space:]]*/, ""); sub(/[[:space:]]*$/, ""); print; exit }
  ' "skills/$skill/SKILL.md")" || return 1
  [[ "$name_in_file" == "$skill" ]]
}

draft_rewrite_produces_draft() {
  local work="$tmp_skill/skill-rewrite-test"
  cp -R skills/skill-rewrite "$work" || return 1
  skills/skill-rewrite/scripts/draft-rewrite.sh -t "$work" >/dev/null || return 1
  [[ -f "$work/REWRITE-DRAFT.md" ]]
}

# Plugin manifests are valid JSON and version matches.
for manifest in .devin-plugin/plugin.json .claude-plugin/plugin.json .cursor-plugin/plugin.json .codex-plugin/plugin.json; do
  assert "$manifest exists" test -f "$manifest"
  assert "$manifest is valid plugin.json" quietly manifest_is_valid "$manifest"
done

# The marketplace manifest is the fifth version surface. `claude plugin
# marketplace add` reads it, and it carries its own copy of the version, so a
# release that bumps the four plugin.json files and forgets this one advertises
# the previous release to anyone installing by name.
assert ".claude-plugin/marketplace.json exists" test -f .claude-plugin/marketplace.json
assert ".claude-plugin/marketplace.json is valid and at 0.4.2" quietly marketplace_is_valid

# The profiler records its own version in every profile it writes, and it is the
# sixth surface carrying this release's number. It is asserted here beside the
# manifests so one place shows all of them, and in Go by
# TestAdapterVersionIsThisRelease.
assert "profiler AdapterVersion is 0.4.2" \
  grep -q 'AdapterVersion = "0.4.2"' profiler/types.go

# The denominator itself, asserted before anything is walked over it. A glob
# that matched nothing would make every per-skill check below vacuously true,
# which is the shape tests/test_harness.sh refuses one level up.
echo "  shipped skills: $(shipped_skills | tr '\n' ' ')"
assert "the shipped skills were read from the tree, not matched as an empty set" \
  test "${shipped_skill_count:-0}" -ge 2

# Each skill has a valid SKILL.md with frontmatter and name matching directory.
for skill in $(shipped_skills); do
  assert "skills/$skill directory exists" test -d "skills/$skill"
  assert "skills/$skill/SKILL.md exists" test -f "skills/$skill/SKILL.md"
  assert "skills/$skill frontmatter name matches directory" \
    frontmatter_name_matches_directory "$skill"
done

# skill-audit carries its scripts and references.
assert "skill-audit check-frontmatter.sh is executable" \
  test -x skills/skill-audit/scripts/check-frontmatter.sh
assert "skill-audit check-structure.sh is executable" \
  test -x skills/skill-audit/scripts/check-structure.sh
assert "skill-audit best-practices reference exists" \
  test -f skills/skill-audit/references/best-practices.md
assert "skill-audit evaluation-matrix reference exists" \
  test -f skills/skill-audit/references/evaluation-matrix.md

# The skill-audit scripts can audit themselves.
assert "skill-audit passes its own frontmatter check" \
  quietly skills/skill-audit/scripts/check-frontmatter.sh skills/skill-audit
assert "skill-audit passes its own structure check" \
  quietly skills/skill-audit/scripts/check-structure.sh skills/skill-audit

# The shipped skills validate clean — warnings included.
#
# skill-audit is the repo's reference skill: it is what the audit skill points
# other skills at, and it invokes this very validator. A warning in our own
# artifact is a defect. Nothing checked for one before, which is how a
# `scripts/lib/` directory earned a "deep nesting detected" warning and carried
# it through every gate unnoticed.
#
# `skill-validator check` exits 0 only on a clean pass; 1 means errors and 2
# means warnings only. Asserting 0 pins both counts at zero.
for skill in $(shipped_skills); do
  assert "skills/$skill validates with zero errors and zero warnings" \
    quietly skill-validator check "skills/$skill"
done

# skill-rewrite carries its script and can draft a rewrite for itself.
assert "skill-rewrite draft-rewrite.sh is executable" \
  test -x skills/skill-rewrite/scripts/draft-rewrite.sh
assert "skill-rewrite draft-rewrite.sh produces REWRITE-DRAFT.md" \
  quietly draft_rewrite_produces_draft

# --- draft-rewrite.sh does not report success over an audit that did not run ---
#
# It ran both audit checks with `2>&1 || true`, which is two defects in one
# line. The status was discarded, so a check that reached no verdict was
# indistinguishable from one reporting a failing skill, and a draft was written
# and exited 0 either way. And merging stderr into the capture put the checks'
# own diagnostics into the draft's "Current state" section as findings about the
# skill: with skill-validator absent, a clean skill's draft opened with "ERROR:
# skill-validator exited with unexpected status 127" as its current state.
#
# A stub here is one directory holding one file, prepended to the real PATH.
# Nothing is mirrored, replaced or uninstalled.

DRAFT=skills/skill-rewrite/scripts/draft-rewrite.sh

# draft_target — a fresh copy of a real skill to draft over. Each case gets its
# own, so a draft one case wrote cannot satisfy the next.
draft_target() {
  local name="$1"
  local dir="$harness_scratch/draft-$name"
  rm -rf "$dir"
  cp -R skills/skill-rewrite "$dir"
  rm -f "$dir/REWRITE-DRAFT.md"
  printf '%s' "$dir"
}

# draft_stub_path <name> <tool> <exit> — a PATH whose <tool> is useless.
draft_stub_path() {
  local dir="$harness_scratch/draft-stub-$1-$3"
  if [ ! -d "$dir" ]; then
    mkdir -p "$dir"
    printf '#!/usr/bin/env bash\nexit %s\n' "$3" > "$dir/$2"
    chmod +x "$dir/$2"
  fi
  printf '%s:%s' "$dir" "$PATH"
}

# draft_refused <name> <tool> <exit> — the audit cannot answer, so no draft is
# written and the status says so. Run as a captured command: the expected
# diagnostics are noise on a passing run.
draft_refused() {
  local name="$1" tool="$2" tool_exit="$3"
  local target status=0 out
  target="$(draft_target "$name")"
  out="$(PATH="$(draft_stub_path "$name" "$tool" "$tool_exit")" "$DRAFT" -t "$target" 2>&1)" || status=$?
  if [ "$status" -ne 3 ]; then
    printf 'expected exit 3, got %s:\n%s\n' "$status" "$out" >&2
    return 1
  fi
  if [ -f "$target/REWRITE-DRAFT.md" ]; then
    printf 'a draft was written over an audit that reached no verdict:\n' >&2
    cat "$target/REWRITE-DRAFT.md" >&2
    return 1
  fi
  return 0
}

assert "draft-rewrite, the spec source cannot run: exits 3 and writes no draft" \
  draft_refused spec-127 skill-validator 127
assert "draft-rewrite, the spec source answers with a status nobody can read: exits 3 and writes no draft" \
  draft_refused spec-42 skill-validator 42
assert "draft-rewrite, jq is present and useless: exits 3 and writes no draft" \
  draft_refused jq-silent jq 0
assert "draft-rewrite, grep is present and useless: exits 3 and writes no draft" \
  draft_refused grep-err grep 2

# draft_state_is_findings_only — the control side of the same invariant. With a
# healthy toolchain a draft is written, and nothing on its "Current state" is a
# diagnostic about the tooling. The checks' own stderr must not be in there.
draft_state_is_findings_only() {
  local target
  target="$(draft_target healthy)"
  "$DRAFT" -t "$target" >/dev/null 2>&1 || return 1
  [ -f "$target/REWRITE-DRAFT.md" ] || return 1
  if grep -q 'ERROR:' "$target/REWRITE-DRAFT.md"; then
    printf 'the draft carries a tooling diagnostic as the skill state:\n' >&2
    grep -n 'ERROR:' "$target/REWRITE-DRAFT.md" >&2
    return 1
  fi
  return 0
}

assert "draft-rewrite, a healthy toolchain: the draft states findings, not diagnostics" \
  quietly draft_state_is_findings_only

# The guard is a dependency it cannot announce itself, so its absence must read
# as "no draft", like every other unmet precondition.
draft_without_guard() {
  # Not named draft-<something>: draft_target owns that prefix, and a tree
  # sharing a name with a target had the target's `rm -rf` delete it.
  local tree="$harness_scratch/skills-without-guard"
  local target status=0
  rm -rf "$tree"
  cp -R skills "$tree"
  rm -f "$tree/skill-audit/scripts/verdict-guard.sh"
  target="$(draft_target no-guard)"
  "$tree/skill-rewrite/scripts/draft-rewrite.sh" -t "$target" >/dev/null 2>&1 || status=$?
  [ "$status" -eq 3 ] || return 1
  [ ! -f "$target/REWRITE-DRAFT.md" ]
}

assert "draft-rewrite, verdict-guard.sh absent: exits 3 and writes no draft" \
  draft_without_guard

# It leaves no temporary audit behind, on the path that writes a draft or on the
# path that refuses to. The refusing path is new, and a new exit path that
# leaked a file each time would be this change's own doing.
# The temporary directory is a private one, not the shared /tmp: mktemp honours
# TMPDIR, and counting entries in a directory other processes also write to is
# a test that fails when something unrelated happens to run beside it.
draft_leaves_no_temp() {
  local target left
  local tmphome="$harness_scratch/draft-tmphome"
  rm -rf "$tmphome"
  mkdir -p "$tmphome"

  target="$(draft_target leak-ok)"
  TMPDIR="$tmphome" "$DRAFT" -t "$target" >/dev/null 2>&1 || return 1

  target="$(draft_target leak-refused)"
  TMPDIR="$tmphome" PATH="$(draft_stub_path leak-refused skill-validator 127)" \
    "$DRAFT" -t "$target" >/dev/null 2>&1 || true

  left="$(find "$tmphome" -type f | wc -l | tr -d '[:space:]')"
  if [ "$left" != 0 ]; then
    printf 'the draft left %s temporary file(s) behind:\n' "$left" >&2
    find "$tmphome" -type f >&2
    return 1
  fi
  return 0
}

assert "draft-rewrite leaves no temporary audit behind, on either path" \
  quietly draft_leaves_no_temp

# The templates it offers are decided by what the *body* holds. A skill whose
# section headings sit at column 0 inside its YAML frontmatter has none of them
# in its body, and used to be handed a draft with no templates at all.
draft_reads_the_body() {
  local target="$harness_scratch/draft-bleed"
  rm -rf "$target"
  mkdir -p "$target"
  cp tests/fixtures/f01/frontmatter-bleed/SKILL.md "$target/SKILL.md"
  "$DRAFT" -t "$target" >/dev/null 2>&1 || return 1
  grep -q '^### When to use' "$target/REWRITE-DRAFT.md" \
    && grep -q '^### Examples' "$target/REWRITE-DRAFT.md"
}

assert "draft-rewrite, headings only in frontmatter: offers the templates the body lacks" \
  quietly draft_reads_the_body

# --- the compatibility each shipped skill actually has -------------------------
#
# A `compatibility:` line is a promise to whoever is deciding whether to install
# the skill, and it is the one claim in the frontmatter nothing was checking.
# Three ways to break it, so three checks, each decidable everywhere rather than
# only on the machine the suite happens to run on.
#
# This lives here, walked over `shipped_skills`, rather than in one skill's own
# suite pinned to one SKILL.md. That is not tidying: the machinery was built for
# skill-rewrite and read a single `$SKILL`, skill-rewrite's line was corrected,
# and skill-audit's identical line went on declaring POSIX shell with zsh and
# `git` with nothing able to see it. Two siblings with identically bash-only
# scripts declaring contradictory compatibility is what a mechanism hard-wired
# to one of them produces.
compatibility_line() {
  { grep -m1 -E '^compatibility:' "$(skill_md_of "$1")" || true; }
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
claimed_interpreters() { claimed_interpreters_in "$(compatibility_line "$1")"; }

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
claimed_commands() { claimed_commands_in "$(compatibility_line "$1")"; }

# A target the witness below can be run against: a copy of a fixture, keeping
# the fixture's own basename, because a skill's directory name is part of what
# check-frontmatter.sh judges and a copy under a slot name of the suite's
# choosing would turn a clean fixture into a spec failure.
witness_fixture_of() {
  case "$1" in
    skill-audit)   printf 'tests/fixtures/f01/valid-minimal' ;;
    skill-rewrite) printf 'tests/fixtures/rewrite/all-sections' ;;
    *) return 1 ;;
  esac
}
witness_target() {
  local skill="$1" slot="$2" fixture slotdir
  fixture="$(witness_fixture_of "$skill")" || return 1
  slotdir="$harness_scratch/witness/$skill/$slot"
  rm -rf "$slotdir"
  mkdir -p "$slotdir"
  cp -R "$fixture" "$slotdir/${fixture##*/}" || return 1
  printf '%s' "$slotdir/${fixture##*/}"
}

# skill_witness <skill> <interpreter> <slot>
#
# Run one of the skill's own bundled scripts under <interpreter> and succeed
# only when it reached a verdict about the skill it was pointed at. Not merely
# "exits 0": under zsh the drafter exits 0 having written a draft whose Current
# state is two file-not-found lines, and an exit-status check would call that
# compatibility. So each arm names the verdict it expects to see -- the audit
# side the PL001 its fixture earns, the rewrite side the audit carried into the
# draft.
#
# A skill with no arm here fails, which is what makes `shipped_skills` a real
# denominator: a third skill cannot join the tree and be walked over vacuously.
skill_witness() {
  local skill="$1" interp="$2" slot="$3" target out=""
  command -v "$interp" >/dev/null 2>&1 || return 1
  target="$(witness_target "$skill" "$slot")" || return 1
  case "$skill" in
    skill-audit)
      out="$("$interp" skills/skill-audit/scripts/check-frontmatter.sh "$target" 2>/dev/null)"
      printf '%s' "$out" | grep -qF 'PL001'
      ;;
    skill-rewrite)
      "$interp" skills/skill-rewrite/scripts/draft-rewrite.sh -t "$target" >/dev/null 2>&1 || return 1
      [ -f "$target/REWRITE-DRAFT.md" ] || return 1
      grep -qF 'frontmatter OK' "$target/REWRITE-DRAFT.md"
      ;;
    *) return 1 ;;
  esac
}

# tool_is_required <skill> <tool> — the skill cannot reach its verdict without
# it. Asked by masking the tool, not by reading the scripts, because the
# question the compatibility line raises is what a reader without that tool
# gets.
tool_is_required() {
  local skill="$1" tool="$2" farm
  farm="$(masked_path "$tool")"
  ! ( PATH="$farm"; skill_witness "$skill" bash "without-$tool" )
}

# The interpreters the skill's own executable scripts name. Only executables:
# `verdict-guard.sh` is sourced rather than run and carries no shebang, so a
# census over every file would read it as a script naming no interpreter. Which
# is itself a claim, so it is asserted rather than assumed.
bundled_executables() {
  local f out=""
  for f in "skills/$1/scripts/"*; do
    [ -f "$f" ] && [ -x "$f" ] || continue
    out="$out$f
"
  done
  printf '%s' "$out"
}
shebang_interpreters() {
  local f
  for f in $(bundled_executables "$1"); do
    sed -n '1s|^#!.*[/ ]\([a-z]*sh\)[[:space:]]*$|\1|p' "$f"
  done | sort -u
}
every_executable_names_an_interpreter() {
  local f
  for f in $(bundled_executables "$1"); do
    [ -n "$(sed -n '1s|^#!.*[/ ]\([a-z]*sh\)[[:space:]]*$|\1|p' "$f")" ] || return 1
  done
  [ -n "$(bundled_executables "$1")" ]
}

# The version half of the claim, pinned where it can be pinned. CI runs one
# Linux job with one bash, so "3.2+" is a claim no run verifies; what a run can
# verify is that the scripts use nothing bash 3.2 lacks. Over every bundled
# file, sourced ones included: a construct in the shared guard reaches bash 3.2
# through the five scripts that load it.
uses_only_bash_32() {
  local f
  for f in "skills/$1/scripts/"*; do
    [ -f "$f" ] || continue
    if grep -qE 'declare -A|mapfile|readarray|local -n|wait -n|globstar|\$\{[A-Za-z_][A-Za-z_0-9]*,,\}|\$\{[A-Za-z_][A-Za-z_0-9]*\^\^\}' "$f"; then
      grep -nE 'declare -A|mapfile|readarray|local -n|wait -n|globstar|\$\{[A-Za-z_][A-Za-z_0-9]*,,\}|\$\{[A-Za-z_][A-Za-z_0-9]*\^\^\}' "$f" >&2
      return 1
    fi
  done
  return 0
}

for skill in $(shipped_skills); do
  echo "  $skill compatibility: $(compatibility_line "$skill")"
  echo "    claims interpreters: $(claimed_interpreters "$skill" | tr '\n' ' ')"
  echo "    claims tools       : $(claimed_commands "$skill" | tr '\n' ' ')"
  echo "    shebangs name      : $(shebang_interpreters "$skill" | tr '\n' ' ')"

  assert "$skill's compatibility line names an interpreter at all" \
    test -n "$(claimed_interpreters "$skill")"

  # (1) Behavioural: every interpreter the line claims runs one of this skill's
  # own scripts to a verdict.
  for interp in $(claimed_interpreters "$skill"); do
    assert "$skill works under $interp, which its compatibility line claims" \
      skill_witness "$skill" "$interp" "under-$interp"
  done

  # (2) Every tool the line names is a tool the skill needs. `git` was on
  # skill-audit's line and no script in either skill runs git.
  for tool in $(claimed_commands "$skill"); do
    assert "$skill's compatibility line names $tool, a tool it actually needs" \
      tool_is_required "$skill" "$tool"
  done

  # (3) Declarative: the line claims the interpreter the bundled scripts'
  # shebangs name, and no other family. This is what makes "POSIX shell"
  # answerable -- a behavioural `sh` run cannot answer it, because /bin/sh is
  # bash in sh mode on macOS and dash on Linux, so the same assertion passes
  # here and fails there.
  assert "$skill's executable scripts each name an interpreter in their shebang" \
    every_executable_names_an_interpreter "$skill"
  assert "$skill's compatibility line claims its scripts' interpreter and no other family" \
    test "$(claimed_interpreters "$skill")" = "$(shebang_interpreters "$skill")"
  if [ "$(claimed_interpreters "$skill")" != "$(shebang_interpreters "$skill")" ]; then
    echo "  claimed: $(claimed_interpreters "$skill" | tr '\n' ' ')"
    echo "  shebang: $(shebang_interpreters "$skill" | tr '\n' ' ')"
  fi

  assert "$skill's bundled scripts use no construct bash 3.2 does not have" \
    quietly uses_only_bash_32 "$skill"
done

# The control for the command extractor, which reads nothing once both lines
# are corrected. Without it, "every tool named is needed" would be true of a
# line naming anything at all. The line is skill-audit's, as it stood.
assert "the compatibility-line reader finds a tool named on such a line" \
  test "$(claimed_commands_in 'compatibility: POSIX shell (bash 3.2+ or zsh), git.')" = "git"
# And the interpreter reader, over the same line: three families, one of them
# claimed by the words "POSIX shell" rather than named.
assert "the compatibility-line reader finds every interpreter family such a line claims" \
  test "$(claimed_interpreters_in 'compatibility: POSIX shell (bash 3.2+ or zsh), git.')" = "$(printf 'bash\nsh\nzsh')"
# The control for the witness, which every check above rests on: a skill run
# under an interpreter that cannot reach its verdict must fail, or "it works
# under what it claims" is true of anything. zsh is the case in hand -- the
# audit scripts resolve their own directory from `BASH_SOURCE[0]`, which zsh
# leaves unset, so `script_dir` collapses and the shared guard is not found.
witness_refuses_zsh() { ! skill_witness skill-audit zsh control-zsh; }
if command -v zsh >/dev/null 2>&1; then
  assert "the witness refuses skill-audit under zsh, the interpreter its line used to claim" \
    witness_refuses_zsh
fi

# --- a claim about the rest of the repository, in either document --------------
#
# A claim about the rest of the repository is decidable against the rest of the
# repository, and this one was not decided. skill-rewrite's document called
# verdict-guard.sh a file "which no other file references by name"; a recursive
# grep finds the name in twelve files, including four of skill-audit's own
# scripts, the readme, two suites and the drafter. The assertion covering the
# sentence only checked that the filename appeared *in the document*, so it
# passed over the falsehood — a string where there should have been a claim.
#
# Walked over both documents for the same reason the compatibility line is:
# skill-audit's document gained a sentence about verdict-guard.sh in this same
# cluster, and a refusal that reads one file cannot see the other.
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

for skill in $(shipped_skills); do
  assert "no claim in $skill's SKILL.md that a file is named nowhere else survives a grep of the repository" \
    unreferenced_claims_hold_in "$(skill_md_of "$skill")"
done

# The control, which is the sentence as it was written, in a document of its
# own. Once the claim is gone the assertion above has nothing to decide, and a
# reader that found no claim would look exactly the same — so the reader is
# held to finding this one, and to refusing it.
control_claim_doc="$harness_scratch/control-unreferenced.md"
printf 'It borrows every check from `$audit_root/scripts/`, including `verdict-guard.sh`, which no other file references by name and which every one of those scripts refuses to compute a verdict without.\n' \
  > "$control_claim_doc"
assert "the claim reader finds the claim in the sentence this document carried" \
  test "$(unreferenced_claims_in "$control_claim_doc")" = "verdict-guard.sh"
control_claim_is_refused() {
  ! unreferenced_claims_hold_in "$control_claim_doc" 2>/dev/null
}
assert "a document claiming verdict-guard.sh is named nowhere else is refused" \
  control_claim_is_refused

harness_summary
