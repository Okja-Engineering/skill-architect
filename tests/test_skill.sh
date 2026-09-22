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
assert data['version'] == '0.5.0', 'version mismatch'
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
assert plugins[0]['version'] == '0.5.0', 'plugin entry version mismatch'
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
assert ".claude-plugin/marketplace.json is valid and at 0.5.0" quietly marketplace_is_valid

# The profiler records its own version in every profile it writes, and it is the
# sixth version surface. It is asserted here beside the manifests so one place
# shows all of them, and in Go by TestAdapterVersionIsThisRelease.
#
# It is deliberately not required to equal the five above. The manifests version
# the plugin and move when the release is cut; this one tracks what a profile
# contains and moves as soon as that changes, which is inside the release rather
# than at the end of it. Holding them equal would mean either labelling every
# profile built during a release as the previous release, or bumping the plugin
# five times on the way there.
assert "profiler AdapterVersion is 0.5.0" \
  grep -q 'AdapterVersion = "0.5.0"' profiler/types.go

# The denominator itself, asserted before anything is walked over it. A glob
# that matched nothing would make every per-skill check below vacuously true,
# which is the shape tests/test_harness.sh refuses one level up.
#
# It was `-ge 2` against a real 2, which is floor == real: a number that says
# nothing today and has to be raised by hand in the commit that ships a third
# skill, by an author with no reason to come here. Its own sentence claims only
# that the set was "not matched as an empty set", which is what `-gt 0` says
# and what 2 never said. And a floor cannot see the denominator move the *other*
# way: measured, an untracked `skills/skill-untracked/SKILL.md` took the set to
# three, the floor passed, seven assertions joined this suite's published total,
# and the only failures were two incidental content checks that name nothing
# about where the extra skill came from — a well-formed one would have gone
# through green.
#
# So the set gets the other side it never had, and it is git's: the filesystem
# says which directories hold a SKILL.md, and the index says which of them this
# repository actually ships. Held for equality, with no number. The index side
# is read from the repository root rather than the process's working directory,
# so the two cannot narrow together — the shape this round found in three
# separate places.
shipped_skills_tracked() {
  git -C "$(git rev-parse --show-toplevel)" ls-files -- 'skills/*/SKILL.md' \
    | sed -e 's|^skills/||' -e 's|/SKILL\.md$||' | sort -u
}
echo "  shipped skills: $(shipped_skills | tr '\n' ' ')"
echo "  skills git tracks a SKILL.md for: $(shipped_skills_tracked | tr '\n' ' ')"
require "git tracks a SKILL.md at all, so the comparison below has a second side" \
  test -n "$(shipped_skills_tracked)"
assert "the shipped skills were read from the tree, not matched as an empty set" \
  test "${shipped_skill_count:-0}" -gt 0
assert "every skill directory in the tree is one this repository tracks a SKILL.md for, and none it does not" \
  test "$(shipped_skills | sort -u)" = "$(shipped_skills_tracked)"

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

# --- The release documents are a version surface too -------------------------
#
# CHANGELOG.md and RELEASE_NOTES.md carry the version in the only form a reader
# ever sees it, and nothing held them. Measured at the release gate: deleting
# the **entire** 0.5.0 section from both documents left every assertion in every
# suite green and byte-identical, and `grep -rn 'CHANGELOG\|RELEASE_NOTES'
# tests/*.sh tests/lib/*.sh` returned nothing at all. This file learned that
# lesson for `marketplace.json` — the fifth version surface, which carries its
# own copy of the version and would otherwise advertise the previous release —
# and did not apply it to the two documents that *are* the release.
#
# The version is read out of a manifest rather than written here, so there is no
# number in this block for the next release to make wrong. The manifests' own
# value is asserted against the literal above, which is where a release bump is
# supposed to be a deliberate edit; everything below is derived from it.
release_version() {
  python3 -c "import json; print(json.load(open('.claude-plugin/plugin.json'))['version'])"
}

# <file> — the release versions that file has a section for, `v` prefix or not.
documented_release_versions() {
  { grep -oE '^## v?[0-9]+\.[0-9]+\.[0-9]+' "$1" || true; } \
    | sed -e 's/^## v\{0,1\}//' | sort -u
}

# <file> <version> — the body under that version's heading, down to the next
# `## `. Asked for separately from the heading because a heading with nothing
# under it is the same defect as no heading: the section was deleted, one line
# later.
release_section_body() {
  awk -v want="$2" '
    /^## / {
      inside = 0
      probe = $0
      sub(/^## v?/, "", probe)
      sub(/ .*$/, "", probe)
      if (probe == want) inside = 1
      next
    }
    inside && NF { print }
  ' "$1"
}

skill_release_version="$(release_version)"
echo "  the release the manifests declare: $skill_release_version"
assert "the release version was read out of a manifest, not written down here" \
  test -n "$skill_release_version"

for release_doc in CHANGELOG.md RELEASE_NOTES.md; do
  assert "$release_doc has a section for the release the manifests declare" \
    quietly grep -qE "^## v?$(printf '%s' "$skill_release_version" | sed 's/\./\\./g')( |\$)" "$release_doc"
  assert "$release_doc's section for that release says something" \
    test -n "$(release_section_body "$release_doc" "$skill_release_version")"
done

# And the two documents against each other, which is the half with no number in
# it and the half that holds every release rather than this one. A section
# deleted from either file — this release's or any earlier one's — leaves the
# lists unequal, whichever file it was deleted from.
assert "the release documents name a version at all, so the comparison has two sides" \
  test -n "$(documented_release_versions CHANGELOG.md)"
assert "the changelog and the release notes carry a section for the same set of releases" \
  test "$(documented_release_versions CHANGELOG.md)" = "$(documented_release_versions RELEASE_NOTES.md)"
if [ "$(documented_release_versions CHANGELOG.md)" != "$(documented_release_versions RELEASE_NOTES.md)" ]; then
  echo "  in the changelog    : $(documented_release_versions CHANGELOG.md | tr '\n' ' ')"
  echo "  in the release notes: $(documented_release_versions RELEASE_NOTES.md | tr '\n' ' ')"
fi

# The control. Both assertions above can only say "the section is there", and a
# reader that stopped reading would say exactly that — so it is driven over
# copies with the section taken out, which is the deletion that was measured.
release_doc_scratch="$harness_scratch/release-docs"
mkdir -p "$release_doc_scratch"
without_release_section() {
  awk -v want="$2" '
    /^## / {
      probe = $0
      sub(/^## v?/, "", probe)
      sub(/ .*$/, "", probe)
      dropping = (probe == want)
    }
    !dropping { print }
  ' "$1"
}
for release_doc in CHANGELOG.md RELEASE_NOTES.md; do
  without_release_section "$release_doc" "$skill_release_version" \
    > "$release_doc_scratch/$release_doc"
  assert "the section reader finds nothing in a $release_doc with that release cut out" \
    test -z "$(release_section_body "$release_doc_scratch/$release_doc" "$skill_release_version")"
  assert "a $release_doc with that release cut out no longer names it, so the check above can fail" \
    test -z "$(documented_release_versions "$release_doc_scratch/$release_doc" | grep -Fx "$skill_release_version" || true)"
done

# --- No file in the tree assigns work to the release being cut ----------------
#
# The fifth recurrence of one class. A comment or a sentence that says a gap is
# "tracked for", "deferred to" or "reserved for" the release now being shipped
# was true when it was written and is false the moment that release is this one.
# It has been found and repaired in four rounds now — in README, in the spec, in
# eight Go comments and in a test's own comment — and each round found it in a
# place the previous round's check did not look.
#
# So the denominator is `git ls-files`: every tracked file, not a list of roots.
# The rounds before this one were checked by readers rooted at `.`, `../docs`
# and `../README.md` from inside `profiler/`, which is why the instances in
# `tests/` and `skills/` survived — two of the eight lived there.
#
# What it looks for is a **deferral vocabulary**, not the version string. The
# version occurs far more often in this tree than there are promises, and
# essentially all of those occurrences are correct: `0.5.0 did not add one`,
# `Until 0.5.0`, `every key 0.5.0 adds`, `this release is 0.5.0`,
# `AdapterVersion is 0.5.0`, a heading, a manifest value. A check on the string
# alone fires on all of them and gets turned off. No count is written here on
# purpose — the numerals this comment used to carry ("69 times … about sixty of
# those") were both wrong, and a figure in a comment that nothing re-derives is
# stale by the next release. The argument does not rest on it.
#
# The words that make a line an *assignment* are few and this repository has
# used the same ones every time: tracked, deferred, carried, reserved,
# scheduled, planned, postponed, "is <version> work", "will … in <version>",
# "needs … in <version>", "new surface for <version>". That is a set of
# **verbs**, and the distinction from a set of *spellings* is what this round
# repaired: the reader matched the raw record, so a capitalised member of its
# own closed vocabulary went past it. It case-folds the record now.
#
# Two scopes, because the two release documents are the one place where a
# forward promise is legitimately *history*. `CHANGELOG.md` and
# `RELEASE_NOTES.md` say what was true at each release, and 0.4.3's own section
# carries a deferral phrase naming its successor because that is what 0.4.3
# meant; those sections are byte-identical to the tag and must stay so. So in
# those two files only the section for the release being cut is in scope.
# Everywhere else — code, tests, skills, README, the spec — there is no section
# structure and no reader who knows which release a sentence was written for, so
# every line is in scope.
#
# The boundary, stated rather than implied: a line that names a version surface
# is exempt from the bare "is <version>" shape, because "AdapterVersion is
# 0.5.0" and "this release is 0.5.0" are the same words doing the opposite job.
# A forward promise written into a line that also mentions a version would get
# past this. Closing that needs the sentence parsed, which is not what a grep
# does; what it does close is the vocabulary that was live five times.
FORWARD_PROMISE_SCAN=tests/lib/forward-promise-check.sh

forward_promises_in_tree() {
  "$FORWARD_PROMISE_SCAN" "$1"
}

assert "the forward-promise reader is on the tree" test -x "$FORWARD_PROMISE_SCAN"
assert "no tracked file assigns work to the release being cut" \
  quietly "$FORWARD_PROMISE_SCAN" "$skill_release_version"
if ! "$FORWARD_PROMISE_SCAN" "$skill_release_version" >/dev/null 2>&1; then
  # `|| true` before the pipe, and it is not decoration: the reader exits 1 when
  # it finds something, `pipefail` passes that out of the pipeline, and `set -e`
  # then killed the suite here — one line after the assertion written to report
  # exactly this, so the invariant failed closed through the abort guard and the
  # diagnostic naming the cause was lost. tests/test_harness.sh has the same
  # note against `workflow_suites` for the same reason.
  { "$FORWARD_PROMISE_SCAN" "$skill_release_version" 2>&1 || true; } | sed 's/^/  /'
fi

# The controls. The check can only ever say "none found", which is what it says
# over a tree it never read — the failure this repository has now found six ways
# — so it is shown finding each shape and shown not firing on the correct ones.
#
# The case class, and why every member of it is written out here rather than in
# the reader. The reader matched the raw record, so its closed verb set was in
# practice a set of *lowercase spellings*, and a capitalised member of its own
# vocabulary passed silently — while a capitalised verb at a sentence start, a
# bullet start or inside a bolded lead-in is the dominant prose shape in this
# repository. All 18 cases here were lowercase, so nothing held that half.
#
# The repair case-folds the record once at the point it reaches the shape tests
# and leaves the tests unanchored, so capitalisation, ALL CAPS, mixed case and
# every leading marker stop being separate forms to enumerate. The enumeration
# belongs here instead, where a fixture is the point: capitalised, ALL CAPS,
# mixed, leading `**`, leading `*`, a list marker, leading whitespace,
# mid-sentence, after a colon, after an em dash, a capital `V` before the
# version, and each of the four non-deferral shapes capitalised. Accepts hold
# the other side of the fold — a version surface stays exempt in any case, and
# sentence-ending punctuation between the verb and the version still blocks the
# match — so the fold cannot have been bought with false positives.
#
# **An accept case earns its place only if something can make it fail.** The
# ALL CAPS version surface here used to read `ADAPTERVERSION IS @V AND IS NOT
# HELD EQUAL`, and the words after the version put an `and` where the shape
# requires end-of-record, ` work`, `.` or `,`. So the record never reached the
# exemption at all: driven against a reader with the exemption **deleted
# outright**, that line was still accepted. It could not tell a working
# exemption from a missing one, let alone a widened one — and while it sat
# there looking like the control for this boundary, the exemption was folded
# along with the record and five assignments this reader had been refusing
# stopped being refused. The case is now cut back to the clause that actually
# meets the shape, and it fails against a reader with no exemption.
#
# The refuses below it are the other half of that boundary: the version word is
# on the line, in each of the spellings the fold made equal, but it is nowhere
# near the `is` — so what is being asked is whether the subject beside that
# `is` names a version surface, not whether the word occurs. The last of them
# carries both on one record, an exempt clause and an assignment, because a
# per-record exemption cannot tell them apart and a per-occurrence one must.
promise_scan_rejects() {
  local fixture="$1"
  if "$FORWARD_PROMISE_SCAN" --over "$fixture" "$skill_release_version" >/dev/null 2>&1; then
    printf 'the forward-promise reader accepted a line it must refuse:\n  %s\n' "$(cat "$fixture")" >&2
    return 1
  fi
  return 0
}
promise_scan_accepts() {
  "$FORWARD_PROMISE_SCAN" --over "$1" "$skill_release_version" >/dev/null 2>&1
}

promise_fixtures="$harness_scratch/forward-promise"
mkdir -p "$promise_fixtures"
promise_n=0
while IFS='|' read -r verdict line; do
  [ -n "$verdict" ] || continue
  promise_n=$((promise_n + 1))
  printf '%s\n' "$(printf '%s' "$line" | sed "s/@V/$skill_release_version/g")" \
    > "$promise_fixtures/case-$promise_n"
  case "$verdict" in
    refuse)
      assert "the forward-promise reader refuses: $line" \
        promise_scan_rejects "$promise_fixtures/case-$promise_n" ;;
    accept)
      assert "the forward-promise reader accepts: $line" \
        promise_scan_accepts "$promise_fixtures/case-$promise_n" ;;
  esac
done <<'PROMISE_CASES'
refuse|// that gap is tracked for @V with the skipped data points
refuse|// surfacing a count of skipped records is deferred to @V
refuse|**Known limits, carried to @V**
refuse|	APIKey string // server API auth; reserved for @V
refuse|# building the capability is @V
refuse|// making it branchable is @V work
refuse|and names it as the source it will read in @V
refuse|// the shape the Devin and Cursor adapters need in @V
refuse|// giving probe an exit contract is new surface for @V
refuse|**Deferred to @V**: a receiver subcommand for the spool
refuse|DEFERRED TO @V.
refuse|DeFeRrEd To @V.
refuse|- Tracked for @V, once the receiver lands
refuse|	  Reserved for @V.
refuse|the receiver is Postponed to @V for now
refuse|Known limits: Carried to @V
refuse|Known limits — Scheduled for @V
refuse|- **Planned for @V**
refuse|*Deferred to @V.*
refuse|Deferred to V@V.
refuse|Making it branchable Is @V work
refuse|Names it as the source it Will read in @V
refuse|Needs a receiver subcommand in @V
refuse|Giving probe an exit contract is New surface for @V
refuse|# VERSION MATRIX: building the capability is @V
refuse|# VERSION: making it branchable is @V work
refuse|# THE VERSION TABLE says building the receiver is @V
refuse|# VeRsIoN note: building the capability is @V
refuse|the version matrix says building the capability is @V
refuse|# RELEASE IS @V and building the receiver is @V
accept|ADAPTERVERSION IS @V
accept|this release is @V
accept|Two adapters were deferred. To @V we added a receiver instead
accept|// making it branchable is new surface, which @V did not add
accept|// Until @V the only list of them was a switch in the CLI
accept|- Every key @V adds is optional, so a round trip is not vacuous
accept|because this release *is* @V and closes none of them
accept|- **`AdapterVersion` is `@V`, and it is deliberately not held equal
accept|## @V — 2026-09-21
accept|  grep -q 'AdapterVersion = "@V"' profiler/types.go
accept|// @V ships none of those adapters, so these fields are reserved
accept|a stored 0.4.x profile against a fresh @V one differs by four keys
PROMISE_CASES

echo "  forward-promise cases driven: $promise_n"
PROMISE_CASES_EXPECTED=42
assert "every one of the $PROMISE_CASES_EXPECTED forward-promise cases was driven, not a prefix of them" \
  test "$promise_n" -eq "$PROMISE_CASES_EXPECTED"

# --- The same class over a real multi-line document --------------------------
#
# Every case above is one line in a file of one line, which is the shape this
# block has twice been caught having: a control that cannot reach the condition
# it tests. None of them can say anything about a promise whose verb and
# version fall either side of a line wrap, because a one-line fixture has no
# second line — and that case is not hypothetical. The changelog copy of the
# sentence describing this very finding escaped the reader only because a break
# happened to fall between the verb and the version, so a reflow moving that
# break by one word would have reddened CI on a file nobody edited.
#
# So one document, with the texture the reader actually meets: headings, a
# bulleted list with bolded lead-ins, wrapped paragraphs, an indented line, a
# fenced block, and blank lines between them. Five promises are planted in it
# and eight innocent lines that name the version are planted beside them.
#
# The assertion is on the **set of line numbers the reader names**, not on the
# exit status, and the expected set is written out here as literals. That is
# what makes it two-sided: a reader blind to any planted shape reports a
# smaller set, and a reader firing on any innocent line reports a larger one,
# and either way the sets differ. An exit-status check would have passed on
# four of the five.
#
# The document length is asserted too, so the literals below cannot silently
# come to mean different lines than the ones the promises were written on.
promise_doc="$promise_fixtures/release-notes-draft.md"
{
  echo '# Release notes draft'                                                      # 1
  echo ''                                                                           # 2
  echo "This release closes the capability gap and this release is $skill_release_version." # 3
  echo ''                                                                           # 4
  echo '## Known limits'                                                            # 5
  echo ''                                                                           # 6
  echo "- **Deferred to $skill_release_version**: a receiver subcommand for the spool." # 7  PROMISE
  echo "- Every key $skill_release_version adds is optional, so a round trip is not vacuous." # 8
  echo "- TRACKED FOR $skill_release_version — the second half of the redaction table." # 9  PROMISE
  echo ''                                                                           # 10
  echo "Until $skill_release_version the only list of them was a switch in the CLI, and that" # 11
  echo 'list is gone now.'                                                          # 12
  echo ''                                                                           # 13
  echo 'The receiver that reads the spool back out over the wire is Deferred'       # 14
  echo "to $skill_release_version, once the envelope shape settles."                # 15 PROMISE (wrapped)
  echo ''                                                                           # 16
  echo '```'                                                                        # 17
  echo "AdapterVersion is $skill_release_version"                                   # 18
  echo '```'                                                                        # 19
  echo ''                                                                           # 20
  printf '\t  Reserved for %s.\n' "$skill_release_version"                          # 21 PROMISE
  echo ''                                                                           # 22
  echo "Known limits: Postponed to $skill_release_version."                         # 23 PROMISE
  echo ''                                                                           # 24
  echo "Two adapters were deferred. To $skill_release_version we added a receiver instead." # 25
  echo ''                                                                           # 26
  echo "a stored 0.4.x profile against a fresh $skill_release_version one differs by four keys" # 27
} > "$promise_doc"

PROMISE_DOC_LINES=27
PROMISE_DOC_PLANTED='7 9 15 21 23'

promise_doc_named() {
  { "$FORWARD_PROMISE_SCAN" --over "$promise_doc" "$skill_release_version" 2>/dev/null || true; } \
    | sed -n "s|^.*/$(basename "$promise_doc"):\([0-9]*\):.*|\1|p" | sort -n | tr '\n' ' ' \
    | sed 's/ $//'
}

promise_doc_found="$(promise_doc_named)"
echo "  multi-line document: the reader named lines [$promise_doc_found]; planted on [$PROMISE_DOC_PLANTED]"

require "the multi-line document is the length its expected line numbers were written against" \
  test "$(grep -ac '' "$promise_doc")" -eq "$PROMISE_DOC_LINES"
assert "over a real multi-line document the reader names exactly the planted promises, and no innocent line" \
  test "$promise_doc_found" = "$PROMISE_DOC_PLANTED"
assert "the multi-line document is refused outright, not merely annotated" \
  promise_scan_rejects "$promise_doc"

# The wrapped promise specifically, named as the shape it is. The reader prints
# `across a line break` in the shape for a finding it could only have seen by
# joining two records, so this asserts the mechanism and not merely the line
# number: a reader that arrived at line 15 some other way would not print it.
PROMISE_DOC_WRAP_AT=15
promise_doc_report() {
  { "$FORWARD_PROMISE_SCAN" --over "$1" "$skill_release_version" 2>/dev/null || true; }
}
promise_doc_shape_at() {
  promise_doc_report "$1" | sed -n "s|^.*:$2: \[\([^]]*\)\].*|\1|p"
}
echo "  the shape reported at line $PROMISE_DOC_WRAP_AT: $(promise_doc_shape_at "$promise_doc" "$PROMISE_DOC_WRAP_AT")"
assert "the promise split across a line wrap is reported as having been found across the wrap" \
  test "$(promise_doc_shape_at "$promise_doc" "$PROMISE_DOC_WRAP_AT")" \
     = "deferral verb across a line break"

# And both inverses over the same document, so "it found line 15" says why.
#
# Wrap closed up, verb and version in one record: still refused, and now as a
# plain deferral verb rather than a wrapped one — the promise is the same
# promise and the reader reaches it by the ordinary path.
promise_doc_joined="$promise_fixtures/wrap-closed.md"
awk 'NR==14 { held = $0; next } NR==15 { print held " " $0; next } { print }' \
  "$promise_doc" > "$promise_doc_joined"
require "closing the wrap produced a document one line shorter, so the join happened" \
  test "$(grep -ac '' "$promise_doc_joined")" -eq "$((PROMISE_DOC_LINES - 1))"
assert "the same promise with the wrap closed up is still refused" \
  promise_scan_rejects "$promise_doc_joined"
assert "with the wrap closed up the finding is no longer attributed to a line break" \
  test "$(promise_doc_shape_at "$promise_doc_joined" 14)" = "deferral verb"

# Promise taken out, wrap left in place: those two lines go quiet and the other
# four planted promises still fire, so what was refused above was the promise
# and not the presence of a wrap.
promise_doc_clean="$promise_fixtures/wrap-innocent.md"
awk 'NR==14 { print "The receiver that reads the spool back out over the wire is there"; next }
     NR==15 { print "in this release, so nothing is owed."; next }
     { print }' "$promise_doc" > "$promise_doc_clean"
promise_doc_clean_found="$(promise_doc_report "$promise_doc_clean" \
  | sed -n "s|^.*/wrap-innocent.md:\([0-9]*\):.*|\1|p" | sort -n | tr '\n' ' ' | sed 's/ $//')"
echo "  with the wrapped promise taken out, the reader named lines [$promise_doc_clean_found]"
assert "taking the wrapped promise out quiets those two lines and leaves the other four firing" \
  test "$promise_doc_clean_found" = "7 9 21 23"

# And the denominator, in three parts, because "the reader ran", "the reader
# read the whole of what it opened" and "the reader looked at what it read" are
# three claims and only the first had anything behind it.
#
# The reader printed a `lines` figure that nothing in this file compared against
# anything. Measured: a forward promise appended to README.md, and
# `FNR > 400 { next }` inserted below the reader's counting rule — the same
# regression that motivated the two-sided count in tests/lib/audit-suites.sh —
# and the reader reported `files=162` and exit 0 over a tree with a live
# violation in it. `files` was green because `files` was the only side wired up.
#
#   files    — against `git ls-files`, compared for equality.
#   lines    — against `grep -ac ''` over the same files, which is a different
#              program counting the same thing. Not `wc -l`: a file with no
#              final newline has one more record than `wc -l` reports, and two
#              tracked files are like that, so `wc -l` would be off by two and
#              somebody would "fix" it with a fudge.
#   skipped  — against the reader's own HISTORY_DOCS and EXEMPT_FILE, read out
#              of the script's text. The reader holds `examined + skipped ==
#              lines` itself, so an accounting that still adds up is not enough:
#              what this side refuses is a skip path that started swallowing a
#              file nobody scoped it to.
#
# Non-empty files, because awk's per-file counter cannot fire for a file with no
# records and `profiler/testdata/otlp/empty.json` is deliberately one.
#
# Every side here is derived from the repository root and not from the process's
# working directory. That is not hygiene. The previous version asked
# `git ls-files` twice from the cwd, once inside the reader and once here, so
# run from a subdirectory both sides narrowed together and stayed equal over a
# fraction of the tree — a denominator derived from the thing it is checking,
# which is the shape this whole round is about, in its third location. It also
# built its side with `xargs -0 -I{} sh -c '[ -s "{}" ]'`, interpolating each
# tracked filename into a shell string; no filename in this tree has a
# metacharacter in it today and the loop below does not care whether one ever
# does.
promise_repo_root="$(git rev-parse --show-toplevel)"
promise_scan_report() {
  { "${1:-$FORWARD_PROMISE_SCAN}" "$skill_release_version" 2>/dev/null || true; } \
    | sed -n '/^files=/p'
}
promise_figure() {
  printf '%s\n' "$2" | sed -n "s/^.*$1=\\([0-9]*\\).*/\\1/p"
}

promise_report="$(promise_scan_report)"
promise_files_read="$(promise_figure files "$promise_report")"
promise_lines_read="$(promise_figure lines "$promise_report")"
promise_examined="$(promise_figure examined "$promise_report")"
promise_skipped="$(promise_figure skipped "$promise_report")"

promise_files_tracked=0
promise_lines_tracked=0
while IFS= read -r -d '' promise_path; do
  promise_n="$(grep -ac '' "$promise_repo_root/$promise_path" 2>/dev/null || true)"
  [ -n "$promise_n" ] || promise_n=0
  [ "$promise_n" -gt 0 ] || continue
  promise_files_tracked=$((promise_files_tracked + 1))
  promise_lines_tracked=$((promise_lines_tracked + promise_n))
done < <(git -C "$promise_repo_root" ls-files -z)

echo "  files the forward-promise reader read: $promise_files_read of $promise_files_tracked non-empty tracked files"
echo "  records it read: $promise_lines_read of $promise_lines_tracked; it looked at $promise_examined and skipped $promise_skipped"

require "a second reader counted the tracked tree, so these comparisons have a side" \
  test "$promise_lines_tracked" -gt 0
assert "the forward-promise reader read every non-empty tracked file, not a prefix of the tree" \
  test "${promise_files_read:-0}" -eq "$promise_files_tracked"
assert "the forward-promise reader read every record in those files, not a prefix of each" \
  test "${promise_lines_read:-0}" -eq "$promise_lines_tracked"
assert "every record the reader read, it either looked at or said it was skipping" \
  test "$((${promise_examined:-0} + ${promise_skipped:-0}))" -eq "${promise_lines_read:-0}"
assert "the reader looked at records at all, so 'none found' is not 'none read'" \
  test "${promise_examined:-0}" -gt 0

# The skips, held against the scope the script states for itself. Two readings
# of one artifact: the files the reader reports skipping in, and the two
# variables that are the only reason it may skip anything.
promise_scoped_in_script() {
  { sed -n 's/^HISTORY_DOCS="\(.*\)"$/\1/p' "$FORWARD_PROMISE_SCAN" | tr ' ' '\n'
    sed -n 's/^EXEMPT_FILE=\(.*\)$/\1/p' "$FORWARD_PROMISE_SCAN"
  } | grep -v '^$' | sort -u
}
promise_scoped_reported() {
  { "$FORWARD_PROMISE_SCAN" "$skill_release_version" 2>/dev/null || true; } \
    | sed -n 's/^skipped-in=\([^ ]*\).*/\1/p' | sort -u
}
echo "  the reader skipped records in: $(promise_scoped_reported | tr '\n' ' ')"
echo "  the script scopes skipping to: $(promise_scoped_in_script | tr '\n' ' ')"
require "the script states the files it may skip in, so this comparison has two sides" \
  test -n "$(promise_scoped_in_script)"
assert "the reader skipped records only in the files its own scope names, and in all of them" \
  test "$(promise_scoped_reported)" = "$(promise_scoped_in_script)"

# --- Controls: the reader can say it did not read the whole tree --------------
#
# Every comparison above can only ever say "it read all of it". The 35
# single-line fixture controls at the top of this block cannot say the opposite
# about any of them: all 35 run in `--over` mode over a file of one line, so
# not one reaches the line count it would take to exercise a limit. A control
# that cannot reach the condition it tests is the shape tests/test_harness.sh
# explicitly refuses, and this block had it for the whole of the release. (The
# multi-line document control above reaches a *wrap*, which is what it is for;
# at 27 lines it comes nowhere near a scan limit either.)
#
# So the reader is copied twice, with a limit inserted at each of the two ends a
# limit can be introduced at, and each copy is driven over the **real tracked
# tree**, which holds files of 2,800 lines — so the limit is reached.
#
#   above the counting rule — `lines` itself drops, and the comparison against
#   `grep -ac ''` refuses it. This is the end the audit's own control uses.
#   below the counting rule — `lines` is untouched and the *walk* stops, which
#   is the regression that was actually walked through here. The accounting
#   comparison refuses it, and so does the reader on its own terms.
promise_scanners="$harness_scratch/promise-scanners"
mkdir -p "$promise_scanners"
truncated_reader="$promise_scanners/stops-reading-at-400.sh"
awk '/^    \{ lines\+\+ \}$/ { print "    FNR > 400 { next }" } { print }' \
  "$FORWARD_PROMISE_SCAN" > "$truncated_reader"
halted_walk_reader="$promise_scanners/stops-walking-at-400.sh"
awk '/^    FNR == 1 \{$/ { print "    FNR > 400 { next }" } { print }' \
  "$FORWARD_PROMISE_SCAN" > "$halted_walk_reader"
chmod +x "$truncated_reader" "$halted_walk_reader"

assert "the truncated copy really is a different program from the reader" \
  quietly grep -qF 'FNR > 400' "$truncated_reader"
assert "the halted-walk copy really is a different program from the reader" \
  quietly grep -qF 'FNR > 400' "$halted_walk_reader"

reader_read_the_whole_tree() {
  local rep
  rep="$(promise_scan_report "$1")"
  test "$(promise_figure lines "$rep")" = "$promise_lines_tracked"
}
reader_accounted_for_what_it_read() {
  local rep
  rep="$(promise_scan_report "$1")"
  test "$(( $(promise_figure examined "$rep") + $(promise_figure skipped "$rep") ))" \
    -eq "$(promise_figure lines "$rep")"
}
# Written as the inverse rather than as `!`, because the expected diagnostic of
# a firing control is noise on a passing run.
refuses_a_truncated_reader() {
  if reader_read_the_whole_tree "$1" 2>/dev/null; then
    printf 'the record count accepted a reader that stops at line 400, so it cannot refuse one\n' >&2
    return 1
  fi
  return 0
}
refuses_an_unaccounted_walk() {
  if reader_accounted_for_what_it_read "$1" 2>/dev/null; then
    printf 'the accounting accepted a walk that stops at line 400, so it cannot refuse one\n' >&2
    return 1
  fi
  return 0
}

assert "a reader that stops reading at line 400 is refused by the record count" \
  refuses_a_truncated_reader "$truncated_reader"
assert "a walk that stops at line 400 while the reader goes on is refused by the accounting" \
  refuses_an_unaccounted_walk "$halted_walk_reader"
assert "a walk that stops is refused by the reader on its own terms, without a caller" \
  quietly test "$("$halted_walk_reader" "$skill_release_version" >/dev/null 2>&1; echo $?)" -ne 0
# And the other direction, over the same two copies with the rule taken back
# out, so the refusals above are the injected rule's doing and not the copy's.
untruncated_reader="$promise_scanners/reads-it-all.sh"
grep -v '^    FNR > 400 { next }$' "$truncated_reader" > "$untruncated_reader"
chmod +x "$untruncated_reader"
assert "the same copy without the rule reads the whole tree, so the copy is not what was refused" \
  reader_read_the_whole_tree "$untruncated_reader"
assert "the same copy without the rule accounts for what it read" \
  reader_accounted_for_what_it_read "$untruncated_reader"

# --- Controls: the section scope, which nothing above can reach ---------------
#
# Every fixture control in this file runs the reader in `--over` mode, and
# `--over` applies no section scoping and no exemption. So the two halves of
# the reader that decide *what it reads at all* had no control anywhere — and
# both of them were wrong. The scoping was: a heading written in any of five
# ordinary ways — a bracketed version, a leading word, a doubled space, a
# deeper level — took the whole document out of scope, the accounting stayed
# perfectly self-consistent, and the reader exited 0 over a file it had not
# read a line of. Driven end to end, an assignment planted inside the section
# went unreported that way.
#
# The script header used to justify shipping that residual by pointing here,
# at "the per-file skip counts". Nothing here held a skip *count*: the only
# assertion over `skipped-in=` compares a set of file **names**, and the set is
# unchanged when a section silently becomes the whole file. The justification
# named a control that did not exist. This is that control.
#
# It needs a tracked tree of its own, because scoping only applies in tree
# mode: a scratch repository with the two history documents and the one file
# the exemption names, driven from inside it. The mutation is the heading and
# nothing else.
promise_scope_repo="$harness_scratch/promise-scope"
promise_scan_abs="$promise_repo_root/$FORWARD_PROMISE_SCAN"

promise_scope_build() {
  local heading="$1" planted="$2" dir="$promise_scope_repo"
  rm -rf "$dir"
  mkdir -p "$dir/profiler" || return 1
  {
    echo '# Changelog'
    echo ''
    echo '## Unreleased'
    echo ''
    printf '%s\n' "$heading"
    echo ''
    if [ "$planted" = planted ]; then
      printf -- '- a receiver subcommand for the spool is deferred to %s, once it lands\n' \
        "$skill_release_version"
    else
      echo '- nothing in this section assigns any work'
    fi
    echo ''
    echo '## 0.4.3 — earlier'
    echo ''
    # History, and the whole reason the scoping exists: an older section saying
    # what it had put off until its successor is true as written.
    printf -- '- surfacing a count of skipped records is deferred to %s\n' \
      "$skill_release_version"
  } > "$dir/CHANGELOG.md" || return 1
  {
    echo '# Release notes'
    echo ''
    printf '## v%s\n' "$skill_release_version"
    echo ''
    echo '- nothing in this section assigns any work'
  } > "$dir/RELEASE_NOTES.md" || return 1
  # The reader refuses an exemption that matched nothing, so the negative
  # control it exempts has to be here too or every run below is red for an
  # unrelated reason.
  printf '// retiredActivationDeferral: the string a test asserts is gone from the tree\n' \
    > "$dir/profiler/profiler_test.go" || return 1
  quietly git -C "$dir" init || return 1
  quietly git -C "$dir" add -A || return 1
}

# Output and status together, because what has to be asserted is *which*
# finding came back and not merely that one did.
promise_scope_run() {
  local heading="$1" planted="$2" out status=0
  promise_scope_build "$heading" "$planted" || return 1
  out="$( cd "$promise_scope_repo" && "$promise_scan_abs" "$skill_release_version" 2>&1 )" \
    || status=$?
  printf '%s\nrc=%d\n' "$out" "$status"
}

promise_scope_names_the_promise() {
  promise_scope_run "$1" planted | grep -q '^CHANGELOG\.md:7: \[deferral verb\]'
}
promise_scope_refuses_the_scope() {
  promise_scope_run "$1" "${2:-planted}" | grep -q '^CHANGELOG\.md: no heading for the release being cut was found'
}
promise_scope_is_quiet() {
  promise_scope_run "$1" clean | grep -q '^rc=0$'
}

promise_scope_heading="## $skill_release_version — 2026-09-21"

# The positive side first: with the heading it knows, the reader reads the
# section, names the planted assignment at its own line, and leaves the older
# section alone. Without this one the refusals below would pass over a reader
# that simply refuses everything.
assert "with the heading it knows, the scoped reader reads the section and names the assignment in it" \
  promise_scope_names_the_promise "$promise_scope_heading"
assert "with the heading it knows and nothing to find, the scoped reader is quiet" \
  promise_scope_is_quiet "$promise_scope_heading"

# The heading spellings. Each of these used to exit 0 over an unread document.
assert "a bracketed heading does not silently take the document out of scope" \
  promise_scope_refuses_the_scope "## [$skill_release_version] — 2026-09-21"
assert "a heading with a word in front of the version does not silently take the document out of scope" \
  promise_scope_refuses_the_scope "## Release $skill_release_version — 2026-09-21"
assert "a deeper heading level does not silently take the document out of scope" \
  promise_scope_refuses_the_scope "### $skill_release_version — 2026-09-21"
assert "a doubled space after the marker does not silently take the document out of scope" \
  promise_scope_refuses_the_scope "##  $skill_release_version — 2026-09-21"
# And with nothing planted, so what is refused is the scope itself rather than
# the assignment inside it.
assert "an unreadable heading is refused even when the section holds nothing to find" \
  promise_scope_refuses_the_scope "## [$skill_release_version] — 2026-09-21" clean

# Case, which is the other half and needs the opposite verdict: a capital V is
# a spelling of the heading and not a different heading, so this one must be
# *read*, not refused. A reader that only failed closed would fail this.
assert "a capital V in the heading is read as the section, not refused as a missing one" \
  promise_scope_names_the_promise "## V$skill_release_version — 2026-09-21"

harness_summary
