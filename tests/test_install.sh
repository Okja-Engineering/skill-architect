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
# against a redirected destination, and require the destination to end up
# identical to the source. That is the invariant: the documented install
# command is an install *and* an update, so running it again must converge on
# the source rather than accumulate. Any command that satisfies it passes —
# the test is pinned to the behaviour, not to the particular command that
# happens to be documented today.
#
# Nothing here touches a live config. The documented destination is rewritten
# to a scratch directory under the harness's own scratch root, and the rewrite
# is asserted (see destination_is_fully_redirected) rather than assumed: a
# README edit that changed the destination spelling must fail this suite, not
# quietly install into the developer's real ~/.claude/skills.

install_scratch="$harness_scratch/install"
mkdir -p "$install_scratch"

# The destination the README documents, as a literal. This is the one string
# the suite has to know, because it is the string it redirects.
documented_dest='~/.claude/skills'

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

# The block is a block. An extractor that silently returned nothing would make
# every assertion below pass over an empty script, which is the failure mode
# this suite is least able to notice from its own output.
documented_block_is_extractable() {
  local block
  block="$(documented_manual_copy_block)" || return 1
  [ -n "$block" ] || return 1
  # Both shipped skills have to be named, or "runs twice cleanly" could be true
  # of a script that installs neither of them.
  printf '%s\n' "$block" | grep -q 'skills/skill-audit' || return 1
  printf '%s\n' "$block" | grep -q 'skills/skill-rewrite' || return 1
  return 0
}

# Rewrite the documented destination to a scratch root and emit the result.
redirected_script() {
  local dest="$1"
  documented_manual_copy_block | sed "s|$documented_dest|$dest|g"
}

# Refuse to run anything still pointing at a real home directory.
#
# This is a safety assertion, not a style one. The suite executes the README's
# commands verbatim apart from one substitution; if that substitution stops
# matching, the commands run against the developer's actual skills directory
# and this suite becomes the thing that corrupts an install.
destination_is_fully_redirected() {
  local dest="$1"
  local script
  script="$(redirected_script "$dest")" || return 1
  printf '%s\n' "$script" | grep -q '~/' && return 1
  printf '%s\n' "$script" | grep -q '\$HOME' && return 1
  printf '%s\n' "$script" | grep -qF "$dest" || return 1
  return 0
}

# Run the documented commands once, from the repository root, with errexit on
# so a documented command that fails is a failure here.
run_documented_install() {
  local dest="$1"
  mkdir -p "$dest"
  redirected_script "$dest" | bash -euo pipefail
}

# --- The invariant ----------------------------------------------------------

# One SKILL.md per installed skill, however many times the command has run.
# This is the shape a recursive harness walk registers, so it is the shape
# asserted: the count of SKILL.md files under the destination root is the
# number of skills installed, not the number of runs.
skill_manifest_count() {
  find "$1" -name SKILL.md | wc -l | tr -d '[:space:]'
}

documented_install_is_idempotent() {
  local dest="$install_scratch/idempotent"
  rm -rf "$dest"
  run_documented_install "$dest" || return 1
  [ "$(skill_manifest_count "$dest")" = 2 ] || return 1
  # The second run is the documented update path.
  run_documented_install "$dest" || return 1
  [ "$(skill_manifest_count "$dest")" = 2 ] || return 1
  # And a third, because "converges" is the claim, not "survives one repeat".
  run_documented_install "$dest" || return 1
  [ "$(skill_manifest_count "$dest")" = 2 ] || return 1
  # The specific corruption, named so a failure reads as itself.
  [ ! -e "$dest/skill-audit/skill-audit" ] || return 1
  [ ! -e "$dest/skill-rewrite/skill-rewrite" ] || return 1
  return 0
}

# Convergence, stated in full: after the update path has run, what is installed
# is what is in the repository. `diff -r` catches the nested copy, a file the
# update failed to refresh, and a file deleted upstream that the update left
# behind — three symptoms of the one root, which is a command that merges into
# the destination instead of replacing it.
documented_install_converges_on_the_source() {
  local dest="$install_scratch/converges"
  rm -rf "$dest"
  run_documented_install "$dest" || return 1

  # A file the repository does not have, as an upstream deletion would leave.
  : > "$dest/skill-audit/scripts/withdrawn-upstream.sh"
  # A file the repository does have, with the wrong contents, as a stale copy
  # would leave.
  echo "stale" > "$dest/skill-rewrite/SKILL.md"

  run_documented_install "$dest" || return 1

  diff -r skills/skill-audit "$dest/skill-audit" >/dev/null || return 1
  diff -r skills/skill-rewrite "$dest/skill-rewrite" >/dev/null || return 1
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

plugin_manifest_dirs() {
  echo ".claude-plugin .codex-plugin .cursor-plugin .devin-plugin"
}

manifest_skills_value() {
  python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["skills"])' "$1"
}

skills_resolve_from_the_repository_root() {
  local dir value
  for dir in $(plugin_manifest_dirs); do
    [ -f "$dir/plugin.json" ] || return 1
    value="$(manifest_skills_value "$dir/plugin.json")" || return 1
    [ -n "$value" ] || return 1
    # Resolves from the repository root...
    [ -d "$value" ] || return 1
    [ -f "$value/skill-audit/SKILL.md" ] || return 1
    [ -f "$value/skill-rewrite/SKILL.md" ] || return 1
    # ...and not from the manifest's own directory, which is the misreading the
    # install instructions have to rule out.
    [ ! -d "$dir/$value" ] || return 1
  done
  return 0
}

# The manifest directory is not itself a distributable plugin: it holds the
# manifest and nothing else. If a skill ever lands inside one, "copy the plugin
# root" stops being unambiguous and this fails.
manifest_dir_holds_only_manifests() {
  local dir entry
  for dir in $(plugin_manifest_dirs); do
    for entry in "$dir"/*; do
      case "${entry##*/}" in
        plugin.json|marketplace.json) ;;
        *) return 1 ;;
      esac
    done
  done
  return 0
}

assert "every native route resolves its skills from the repository root" \
  skills_resolve_from_the_repository_root
assert "no manifest directory is mistakable for the plugin root" \
  manifest_dir_holds_only_manifests

assert "the README documents a runnable manual-copy block" \
  documented_block_is_extractable
assert "the documented destination is redirected away from any live config" \
  destination_is_fully_redirected "$install_scratch/redirect-check"
assert "the documented manual copy installs two skills and is safe to re-run" \
  documented_install_is_idempotent
assert "the documented manual copy converges on the repository contents" \
  documented_install_converges_on_the_source

harness_summary
