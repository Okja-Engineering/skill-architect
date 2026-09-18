#!/usr/bin/env bash
set -euo pipefail
. "$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)/lib/harness.sh"
harness_init

# The suite for the documented install routes.
#
# This exists because the install instructions were written and never run. The
# manual standalone copy was the documented *update* path — README tells the
# reader they "pull updates when you choose" — and `cp -R src dst` copies
# *into* dst once dst exists, so the second run of the documented command
# nested a second SKILL.md inside the installed skill and any harness that
# walks the skills root recursively registered the skill twice.
#
# So the assertions below do not check that the README says a particular
# thing. They **extract the command the README documents and run it**, twice,
# and require the destination to end up identical to the source. That is the
# invariant: the documented install command is an install *and* an update, so
# running it again must converge on the source rather than accumulate, and no
# failure of any step may leave the destination worse than it started. Any
# command that satisfies that passes — the test is pinned to the behaviour, not
# to the particular command that happens to be documented today.
#
# Nothing here can touch a live config, and that is a property of how the block
# is run rather than an assertion about its text. See the containment section
# below: an earlier version of this suite rewrote the destination with `sed` and
# checked the rewrite with `grep`, which is a guard that holds only for the
# spellings someone thought of — and it reported its verdict through `assert`,
# which reports and returns, so the suite ran `rm -rf` against a live skills
# directory after its own guard had already said no.

install_scratch="$harness_scratch/install"
mkdir -p "$install_scratch"

# The bash the suite itself is running under. The block is a documented shell
# command and this project supports two shells, so a block that behaves
# differently on 3.2 has to be seen on 3.2 — `bash` off PATH would quietly run
# every extraction under whichever one is first there.
suite_bash="${BASH:-bash}"

# --- What the repository ships, as the denominator for everything below -------
#
# The skills are enumerated from the repository rather than named, so a third
# skill is covered the day it lands. Nothing below knows how many there are.

repository_skill_names() {
  local dir
  for dir in "$harness_repo_root"/skills/*/; do
    [ -f "$dir/SKILL.md" ] || continue
    dir="${dir%/}"
    printf '%s\n' "${dir##*/}"
  done
}

repository_ships_skills() {
  [ -n "$(repository_skill_names)" ]
}

# --- The block the README documents ------------------------------------------

# The commands the README documents under "Manual standalone copy", read out of
# the README itself so the suite cannot drift from the document it is proving.
#
# Heredoc bodies and fenced blocks elsewhere in the README are not candidates:
# the scan starts at the heading and takes the first bash block after it.
documented_manual_copy_block() {
  awk '
    /^### Manual standalone copy$/ { found = 1; next }
    found && /^```bash$/ { in_block = 1; next }
    in_block && /^```$/ { exit }
    in_block { print }
  ' README.md
}

# The block is a block, and it names every skill the repository ships. An
# extractor that silently returned nothing, or a block that named only one of
# the skills, would make "installs and converges" true of a script that
# installed nothing. The name is looked for on its own rather than as
# `skills/<name>`, because where the block says it — spelled into a path, or
# listed once for a loop to walk — is the block's business and not this check's.
documented_block_is_extractable() {
  local block name
  block="$(documented_manual_copy_block)" || return 1
  [ -n "$block" ] || return 1
  while read -r name; do
    [ -n "$name" ] || continue
    printf '%s\n' "$block" | grep -qF "$name" || return 1
  done <<BLOCK_SKILLS
$(repository_skill_names)
BLOCK_SKILLS
  return 0
}

# --- Containment, proven rather than asserted about a substitution -----------
#
# The block runs with HOME and the working directory both inside the harness
# scratch root, with an environment that carries nothing else, and under
# `set -u`, which turns any other expansion into an abort *before* the command
# on that line runs. That leaves exactly two spellings that could still name
# something outside:
#
#   an absolute path — `/Users/you/.claude/skills` ignores HOME entirely;
#   `~name` — a tilde with a user attached expands from the password database
#             and not from HOME, so `~root/.claude` escapes a redirected home.
#
# Those two are refused here, over every word of the block. This is the whole
# set of escapes from the containment above, which is what makes it a proof
# rather than a list of the spellings that happened to be wrong once. The
# refusal is a `require`, not an `assert`: a failed precondition has to stop the
# suite, because a reported failure is not a refusal.

text_names_nothing_outside_a_redirected_home() {
  awk '
    {
      n = split($0, w, /[[:space:]]+/)
      for (i = 1; i <= n; i++) {
        t = w[i]
        if (t == "") continue
        if (substr(t, 1, 1) == "#") break
        gsub(/["'"'"']/, "", t)
        sub(/^[A-Za-z_][A-Za-z0-9_]*=/, "", t)
        if (t ~ /^\//) {
          printf "%d: absolute path, which ignores a redirected HOME: %s\n", FNR, w[i]
          bad++
        } else if (t ~ /^~[^\/]/) {
          printf "%d: tilde with a user name, which expands from the password database and not from HOME: %s\n", FNR, w[i]
          bad++
        }
      }
    }
    END { exit (bad > 0) }
  '
}

block_names_nothing_outside_a_redirected_home() {
  documented_manual_copy_block | text_names_nothing_outside_a_redirected_home
}

# The control for the guard. A guard that cannot refuse is not a proof of
# anything, so each escape it exists to catch is handed to it and the refusal is
# required — before anything is run, because these two checks are what the
# containment argument rests on.
guard_refuses() {
  local why
  if why="$(printf '%s\n' "$1" | text_names_nothing_outside_a_redirected_home 2>&1)"; then
    printf 'the containment guard accepted a block it must refuse:\n%s\n' "$1" >&2
    return 1
  fi
  return 0
}

guard_refuses_an_absolute_destination() {
  guard_refuses 'rm -rf /Users/you/.claude/skills/skill-audit && cp -R skills/skill-audit /Users/you/.claude/skills/skill-audit'
}

guard_refuses_a_tilde_with_a_user_name() {
  guard_refuses 'cp -R skills/skill-audit ~root/.claude/skills/skill-audit'
}

guard_refuses_a_block_whose_second_line_escapes() {
  guard_refuses 'mkdir -p ~/.claude/skills
cp -R skills/skill-audit /etc/codex/skills/skill-audit'
}

# <path> <root> — path resolves to somewhere strictly inside root. Both sides are
# resolved, because on this platform the scratch root arrives through /var and
# lives at /private/var, and a string compare on the unresolved pair says "no"
# for a path that is in fact inside.
path_is_inside() {
  local resolved root
  resolved="$(CDPATH= cd -P -- "$1" 2>/dev/null && pwd -P)" || return 1
  root="$(CDPATH= cd -P -- "$2" 2>/dev/null && pwd -P)" || return 1
  case "$resolved" in
    "$root"/*) return 0 ;;
  esac
  return 1
}

everything_the_block_can_reach_is_inside_the_scratch_root() {
  path_is_inside "$install_scratch" "$harness_scratch"
}

# --- Running the block, contained --------------------------------------------

# A home the block installs into, and a working directory it runs from. The
# working directory holds a copy of the repository's skills tree rather than the
# repository itself, so a block that wrote to its source would write to the copy
# — and so that a run from a directory *without* the tree can be staged just by
# not making it.
staged_run_dir() {
  local name="$1"
  local dir="$install_scratch/$name"
  rm -rf "$dir"
  mkdir -p "$dir/home" "$dir/cwd"
  printf '%s\n' "$dir"
}

stage_the_repository_skills() {
  cp -R "$harness_repo_root/skills" "$1/cwd/skills"
}

run_documented_block() {
  local dir="$1"
  local script="$dir/documented-block.sh"
  documented_manual_copy_block > "$script" || return 1
  (
    cd "$dir/cwd" || exit 1
    env -i HOME="$dir/home" PATH="$PATH" "$suite_bash" -euo pipefail "$script"
  )
}

# --- The invariant ------------------------------------------------------------

# Where the block put things, discovered rather than assumed. The suite does not
# need to know the path the README documents: it needs the installed tree to be
# the repository's tree, wherever the README says to put it.
installed_skill_dirs() {
  find "$1" -name SKILL.md | sed 's|/SKILL\.md$||' | sort
}

# The destination root, read back from where the block actually installed. Named
# from the repository's own skill list, so it is discovered and not spelled.
installed_destination_root() {
  local home="$1" first
  first="$(repository_skill_names | head -1)"
  [ -n "$first" ] || return 1
  find "$home" -type d -name "$first" | head -1 | sed "s|/$first\$||"
}

# What is installed is what is in the repository: every shipped skill present,
# one manifest each however many times the command has run, and byte-identical
# contents. `diff -r` catches a file the update failed to refresh and a file
# deleted upstream that the update left behind; the manifest count catches the
# nested copy, which adds a SKILL.md inside a skill that is otherwise right.
every_repository_skill_is_installed_intact() {
  local dest="$1" name
  [ -n "$dest" ] || return 1
  while read -r name; do
    [ -n "$name" ] || continue
    [ -d "$dest/$name" ] || return 1
    [ "$(find "$dest/$name" -name SKILL.md | wc -l | tr -d '[:space:]')" = 1 ] || return 1
    diff -r "$harness_repo_root/skills/$name" "$dest/$name" >/dev/null || return 1
  done <<SKILL_NAMES
$(repository_skill_names)
SKILL_NAMES
  return 0
}

installed_tree_is_the_repository_tree() {
  local home="$1" dest
  dest="$(installed_destination_root "$home")" || return 1
  every_repository_skill_is_installed_intact "$dest" || return 1
  # And nothing else anywhere under the home: a nested copy, a staging directory
  # an interrupted run left behind, and a second install somewhere else all show
  # up as a manifest the repository does not account for.
  [ "$(find "$home" -name SKILL.md | wc -l | tr -d '[:space:]')" \
    = "$(repository_skill_names | wc -l | tr -d '[:space:]')" ] || return 1
  return 0
}

# --- Never worse than it started ----------------------------------------------
#
# The second invariant, and the one the chosen command broke. Converging is not
# enough: no failure of any step may leave the destination worse than it was,
# because the documented command is the *update* path and it is run over a
# working install.

# The failure a reader actually hits: the block run from a directory that is not
# a checkout, so its source is not there. `rm -rf dst && cp -R src dst` guards
# the copy against a failed remove and leaves the install *empty* when the copy
# is what fails.
documented_update_survives_a_failing_step() {
  local dir dest before after
  dir="$(staged_run_dir failing-step)" || return 1
  stage_the_repository_skills "$dir" || return 1
  run_documented_block "$dir" || return 1
  installed_tree_is_the_repository_tree "$dir/home" || return 1

  dest="$(installed_destination_root "$dir/home")" || return 1
  [ -n "$dest" ] || return 1
  before="$(cd "$dest" && find . | sort)" || return 1

  # The same block, from a directory with no skills tree in it. Its exit status
  # is not what is asserted — the destination is.
  rm -rf "$dir/cwd/skills"
  run_documented_block "$dir" >/dev/null 2>&1 || true

  after="$(cd "$dest" && find . | sort)" || return 1
  [ "$before" = "$after" ] || return 1
  every_repository_skill_is_installed_intact "$dest" || return 1
  return 0
}

# A run that fails part-way must also leave the *next* run able to converge. A
# staged form that does not clear its own staging directory passes the check
# above and then nests the next update inside what it left behind — which is the
# nesting defect this cluster exists to fix, arriving by another route.
#
# The interruption is a read-only installed skill directory rather than a signal
# or a stubbed tool: it stops whichever step replaces the install, on every
# command form, with no timing to get right and nothing installed or faked.
#
# What is asserted after the interruption is recovery, not survival, and the
# distinction is deliberate. A `rm -rf` on a directory it cannot finish removing
# deletes what it can reach and leaves the rest, so *no* remove-then-replace
# form — and that is every dependency-free form — keeps an install intact
# through this one. Measured: the installed skill loses the contents of every
# writable subdirectory. Demanding survival here would be demanding a
# rename-swap with an `.old` directory and six steps in a README block, to
# protect against a permissions state the reader would have had to create.
# Recovery is the property that is both reachable and the one that matters: the
# documented command, run again, converges.
an_interrupted_update_leaves_the_next_one_able_to_converge() {
  local dir dest first
  dir="$(staged_run_dir interrupted)" || return 1
  stage_the_repository_skills "$dir" || return 1
  run_documented_block "$dir" || return 1
  dest="$(installed_destination_root "$dir/home")" || return 1
  first="$(repository_skill_names | head -1)"
  [ -n "$first" ] || return 1

  chmod 500 "$dest/$first" || return 1
  run_documented_block "$dir" >/dev/null 2>&1 || true
  chmod -R 700 "$dest/$first" || return 1

  # The documented command, run again, converges on the repository instead of
  # merging into whatever the interrupted run left behind.
  run_documented_block "$dir" || return 1
  installed_tree_is_the_repository_tree "$dir/home" || return 1
  return 0
}

# The README claims it in prose — "nothing else under the skills directory is
# touched" — and the reader whose other skills live there is the one who pays if
# it is wrong. This is also what catches the adaptation hazard: a destination
# substituted one level too high turns the update into `rm -rf` on the directory
# holding every skill the reader has.
documented_update_leaves_the_other_skills_alone() {
  local dir dest
  dir="$(staged_run_dir siblings)" || return 1
  stage_the_repository_skills "$dir" || return 1
  run_documented_block "$dir" || return 1
  dest="$(installed_destination_root "$dir/home")" || return 1
  [ -n "$dest" ] || return 1

  mkdir -p "$dest/someone-elses-skill" || return 1
  echo "not ours" > "$dest/someone-elses-skill/SKILL.md"
  run_documented_block "$dir" || return 1
  run_documented_block "$dir" || return 1

  [ "$(cat "$dest/someone-elses-skill/SKILL.md" 2>/dev/null)" = "not ours" ] || return 1
  every_repository_skill_is_installed_intact "$dest" || return 1
  return 0
}

# --- One destination, spelled once, in the words the README's own list uses ----
#
# The other half of the same blocker. The command embedded the skills directory
# *and* the skill name at every destructive step, while the list of paths a
# reader takes the destination from gives skills directories. A reader adapting
# one into the other produces `rm -rf` on the directory holding all their
# skills. Neither the form of the command nor a warning fixes that: the two
# places have to agree, and there has to be exactly one of them.

# The one thing the block asks the reader to change: its first assignment. Read
# as "the first assignment" rather than by variable name, so the block stays
# free to call it something else.
documented_destination() {
  documented_manual_copy_block \
    | sed -n 's/^[A-Za-z_][A-Za-z0-9_]*=\([^ 	#]*\).*$/\1/p' \
    | head -1
}

# The same path as the README's own list of where each agent reads skills from
# gives it. Claude Code is the row the command is written against.
readme_listed_destination() {
  sed -n 's/^- \*\*Claude Code\*\* — `\([^`]*\)`.*$/\1/p' README.md | head -1
}

the_list_and_the_block_agree_on_the_destination() {
  local from_block from_list
  from_block="$(documented_destination)"
  from_list="$(readme_listed_destination)"
  [ -n "$from_block" ] || return 1
  [ -n "$from_list" ] || return 1
  [ "${from_block%/}" = "${from_list%/}" ]
}

the_destination_has_exactly_one_substitution_point() {
  local dest count
  dest="$(documented_destination)"
  [ -n "$dest" ] || return 1
  count="$(documented_manual_copy_block | grep -oF "$dest" | wc -l | tr -d '[:space:]')"
  [ "$count" = 1 ]
}

documented_install_is_an_update_that_converges() {
  local dir
  dir="$(staged_run_dir idempotent)" || return 1
  stage_the_repository_skills "$dir" || return 1

  run_documented_block "$dir" || return 1
  installed_tree_is_the_repository_tree "$dir/home" || return 1

  # The second run is the documented update path. Before it, the two states a
  # merge-into-the-destination command leaves behind for good: a file the
  # repository no longer has, and a file whose contents are stale.
  local installed
  installed="$(installed_skill_dirs "$dir/home" | head -1)"
  [ -n "$installed" ] || return 1
  : > "$installed/withdrawn-upstream.md"
  echo "stale" > "$installed/SKILL.md"

  run_documented_block "$dir" || return 1
  installed_tree_is_the_repository_tree "$dir/home" || return 1

  # And a third, because "converges" is the claim, not "survives one repeat".
  run_documented_block "$dir" || return 1
  installed_tree_is_the_repository_tree "$dir/home" || return 1
  return 0
}

# --- What "the plugin directory" means, for every native route --------------
#
# Each agent reads its manifest from its own subdirectory of the repository
# root, and the manifest's `skills` value resolves from the *root*, not from the
# directory the manifest sits in. That makes the repository root the plugin root
# and the manifest directory merely where the manifest lives — a distinction the
# install instructions have to get right, because `.cursor-plugin/` on its own
# holds one JSON file and no skills, so a reader who copies that installs a
# plugin with nothing in it and gets no error telling them so.
#
# Asserted here rather than stated in prose, because prose is what was wrong.
#
# The manifest directories are a glob for the same reason the skills are: a
# fifth agent's manifest is covered the day it lands. Hardcoding the four made
# every check below blind to exactly the manifest most likely to be wrong.

plugin_manifest_dirs() {
  local dir
  for dir in .*-plugin; do
    [ -d "$dir" ] || continue
    printf '%s\n' "$dir"
  done
}

repository_has_plugin_manifests() {
  [ -n "$(plugin_manifest_dirs)" ]
}

manifest_skills_value() {
  python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["skills"])' "$1"
}

skills_resolve_from_the_repository_root() {
  local dir value name
  while read -r dir; do
    [ -n "$dir" ] || continue
    [ -f "$dir/plugin.json" ] || return 1
    value="$(manifest_skills_value "$dir/plugin.json")" || return 1
    [ -n "$value" ] || return 1
    # Resolves from the repository root...
    [ -d "$value" ] || return 1
    while read -r name; do
      [ -n "$name" ] || continue
      [ -f "$value/$name/SKILL.md" ] || return 1
    done <<MANIFEST_SKILLS
$(repository_skill_names)
MANIFEST_SKILLS
    # ...and not from the manifest's own directory, which is the misreading the
    # install instructions have to rule out.
    [ ! -d "$dir/$value" ] || return 1
  done <<MANIFEST_DIRS
$(plugin_manifest_dirs)
MANIFEST_DIRS
  return 0
}

# The manifest directory is not itself a distributable plugin: it holds the
# manifest and nothing else. If a skill ever lands inside one, "copy the plugin
# root" stops being unambiguous and this fails.
manifest_dir_holds_only_manifests() {
  local dir entry
  while read -r dir; do
    [ -n "$dir" ] || continue
    for entry in "$dir"/* "$dir"/.*; do
      case "${entry##*/}" in
        '.'|'..'|'*'|'.*') ;;
        plugin.json|marketplace.json) ;;
        *) return 1 ;;
      esac
    done
  done <<MANIFEST_DIRS
$(plugin_manifest_dirs)
MANIFEST_DIRS
  return 0
}

# The manifests are maintained by hand and nothing compared them, so they
# drifted: three spelled the skills path `./skills/` and one spelled it
# `skills`. Both worked, which is why it survived — the cost of that kind of
# drift is not a broken install, it is a later "fix them all" that misses one.
# Comparing them is the only thing that stops it recurring.
manifests_agree_on_the_skills_path() {
  local dir value first=""
  while read -r dir; do
    [ -n "$dir" ] || continue
    value="$(manifest_skills_value "$dir/plugin.json")" || return 1
    if [ -z "$first" ]; then
      first="$value"
    elif [ "$value" != "$first" ]; then
      return 1
    fi
  done <<MANIFEST_DIRS
$(plugin_manifest_dirs)
MANIFEST_DIRS
  [ -n "$first" ]
}

# `claude plugin validate --strict` fails on missing metadata, and author was
# missing from all four. Claude Code is not installed in CI, so the requirement
# is asserted against the manifests directly — a check that skipped itself when
# the CLI was absent would report nothing in the one place it has to report.
manifests_carry_author_attribution() {
  local dir
  while read -r dir; do
    [ -n "$dir" ] || continue
    python3 - "$dir/plugin.json" <<'PY' || return 1
import json, sys
a = json.load(open(sys.argv[1])).get("author")
assert isinstance(a, dict), "author must be an object"
assert a.get("name"), "author.name must be non-empty"
assert a.get("url"), "author.url must be non-empty"
PY
  done <<MANIFEST_DIRS
$(plugin_manifest_dirs)
MANIFEST_DIRS
  return 0
}

# --- Preconditions ------------------------------------------------------------
#
# Everything below runs the README's commands, so these come first and each one
# stops the suite rather than reporting on its way past.

require "the containment guard refuses an absolute destination" \
  guard_refuses_an_absolute_destination
require "the containment guard refuses a tilde with a user name" \
  guard_refuses_a_tilde_with_a_user_name
require "the containment guard reads the whole block, not just its first line" \
  guard_refuses_a_block_whose_second_line_escapes
require "the destination the block can reach is inside the harness scratch root" \
  everything_the_block_can_reach_is_inside_the_scratch_root
require "the README's manual-copy block names nothing outside a redirected home" \
  quietly block_names_nothing_outside_a_redirected_home
require "the repository ships skills for the checks below to be about" \
  repository_ships_skills
require "the repository holds plugin manifests for the checks below to be about" \
  repository_has_plugin_manifests
require "the README documents a manual-copy block naming every shipped skill" \
  documented_block_is_extractable

echo "  skills examined: $(repository_skill_names | tr '\n' ' ')"
echo "  plugin manifests examined: $(plugin_manifest_dirs | tr '\n' ' ')"

# --- Assertions ---------------------------------------------------------------

assert "every native route resolves its skills from the repository root" \
  skills_resolve_from_the_repository_root
assert "no manifest directory is mistakable for the plugin root" \
  manifest_dir_holds_only_manifests
assert "every plugin manifest agrees on how the skills path is spelled" \
  manifests_agree_on_the_skills_path
assert "every plugin manifest carries author attribution" \
  quietly manifests_carry_author_attribution

assert "the documented manual copy installs every shipped skill and converges on re-run" \
  documented_install_is_an_update_that_converges
assert "a failing step leaves the previous install exactly as it was" \
  documented_update_survives_a_failing_step
assert "an interrupted update leaves the next one able to converge" \
  an_interrupted_update_leaves_the_next_one_able_to_converge
assert "the documented update leaves the reader's other skills alone" \
  documented_update_leaves_the_other_skills_alone
assert "the destination is spelled in exactly one place in the block" \
  the_destination_has_exactly_one_substitution_point
assert "the block's destination is the path the README's own list gives" \
  the_list_and_the_block_agree_on_the_destination

harness_summary
