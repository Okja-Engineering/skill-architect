#!/usr/bin/env bash
# The suite for the release pointer: what users install, as distinct from what
# lands on trunk.
#
# Two invariants, and neither had anything holding it before this file.
#
#   1. **Every manifest in this repository agrees on the version, and a pinned
#      marketplace source agrees with it too.** Measured: `claude plugin
#      validate --strict` passes a marketplace entry that says `9.9.9` while
#      `plugin.json` says `0.5.0`, installs `0.5.0`, and advertises `9.9.9` to
#      anyone browsing. Only `claude plugin tag --dry-run` refuses it, and that
#      runs at tag time on a machine with Claude Code installed — which CI is
#      not. So the comparison is made here, against the files, with no CLI.
#
#   2. **The release branch never carries a version that was never released.**
#      The repository's default branch is what an unpinned clone lands on, and
#      every documented install route for all four agents clones without naming
#      a ref. So whatever the default branch says is what the world installs.
#      The rule that makes that safe is that the release branch's HEAD is a
#      commit some tag `v<version>` names, where `<version>` is the version its
#      own manifests declare.
#
# The second invariant is a statement about a branch, and most runs of this
# suite are not on that branch. A check that quietly does nothing on six runs
# out of seven reports nothing on the seventh either, because nobody would
# notice it had stopped working. So the predicate is exercised on every run,
# over throwaway repositories built here, in both directions — it accepts a
# tagged head, and it refuses an untagged one and a tag that names some other
# commit. The live tree is then held to whichever half of the rule its branch
# falls under, and the suite prints which half that was.
set -euo pipefail
. "$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)/lib/harness.sh"
harness_init

# refuses <command> [args...] — the command under test says no.
#
# Written as a captured run rather than `! cmd`, for the reason
# tests/test_harness.sh gives for the same shape: the expected output of a
# firing control is noise on a passing run, and a control that silently stopped
# firing has to be able to say so.
refuses() {
  local out
  if out="$("$@" 2>&1)"; then
    printf 'expected a refusal, got acceptance:\n%s\n' "$out" >&2
    return 1
  fi
  return 0
}

tmp="$harness_scratch/release"
mkdir -p "$tmp"

# The branch that carries what users install. One spelling, here, so the
# workflow, the runbook and this suite cannot drift apart silently: everything
# else reads it from the environment or from this default.
RELEASE_BRANCH="${SKILL_ARCHITECT_RELEASE_BRANCH:-release}"

# --- Reading the manifests ----------------------------------------------------
#
# The manifest directories are a glob rather than the four that exist today, for
# the reason tests/test_install.sh gives: a fifth agent's manifest is covered
# the day it lands, and hardcoding the four blinds every check below to exactly
# the manifest most likely to be wrong.

manifest_dirs() {
  local dir
  for dir in "$1"/.*-plugin; do
    [ -d "$dir" ] || continue
    printf '%s\n' "$dir"
  done
}

declared_version() {
  python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["version"])' "$1"
}

# manifests_agree_on_the_version <repo> — every plugin.json says the same thing.
#
# Not a style point. The four manifests are maintained by hand and nothing
# compared their versions; test_install.sh compares their `skills` paths and
# their `author` blocks and stops there. A release that bumps three of four
# ships an agent installing a plugin that reports the previous version.
manifests_agree_on_the_version() {
  local dir value first=""
  while read -r dir; do
    [ -n "$dir" ] || continue
    [ -f "$dir/plugin.json" ] || return 1
    value="$(declared_version "$dir/plugin.json")" || return 1
    [ -n "$value" ] || return 1
    if [ -z "$first" ]; then
      first="$value"
    elif [ "$value" != "$first" ]; then
      echo "$dir/plugin.json says $value, an earlier manifest says $first" >&2
      return 1
    fi
  done <<MANIFEST_DIRS
$(manifest_dirs "$1")
MANIFEST_DIRS
  [ -n "$first" ]
}

# marketplace_entries_agree <repo> — a catalogue entry cannot advertise one
# version and install another, and a pinned source cannot name a ref that
# disagrees with the version beside it.
#
# `version` is optional in a marketplace entry, and an absent one is correct:
# the fetched plugin.json wins at install time either way. What is refused is a
# *present* one that disagrees, because that is the field a browsing user reads.
#
# The `ref` half is here rather than in prose because the two forms of the
# `source` field are not interchangeable and the difference is invisible at
# install time: a string source resolves inside the marketplace's own checkout,
# an object source with a `ref` goes to the host and fetches that ref. Nothing
# in the vendor's validator compares that ref to the version. This does.
marketplace_entries_agree() {
  local dir
  while read -r dir; do
    [ -n "$dir" ] || continue
    [ -f "$dir/marketplace.json" ] || continue
    python3 - "$dir/marketplace.json" "$dir/plugin.json" <<'PY' || return 1
import json, sys

catalogue = json.load(open(sys.argv[1]))
declared = json.load(open(sys.argv[2]))["version"]

for entry in catalogue["plugins"]:
    advertised = entry.get("version")
    if advertised is not None and advertised != declared:
        raise SystemExit(
            "%s advertises %r for a plugin that installs as %r"
            % (sys.argv[1], advertised, declared))
    source = entry.get("source")
    if isinstance(source, dict):
        ref = source.get("ref")
        if ref is not None and ref != "v" + declared:
            raise SystemExit(
                "%s pins ref %r beside version %r; a pin that disagrees with "
                "the version it ships installs neither what it advertises nor "
                "what the branch holds" % (sys.argv[1], ref, declared))
PY
  done <<MANIFEST_DIRS
$(manifest_dirs "$1")
MANIFEST_DIRS
  return 0
}

# --- The release-branch rule --------------------------------------------------

# head_is_a_released_version <repo> — the predicate, over any repository.
#
# True when some tag named `v<version>` resolves to this commit, where
# <version> is what the repository's own manifests declare. Peeled with `^{}`
# so an annotated tag answers with the commit it names rather than the tag
# object, which is the shape every tag in this repository's history has.
head_is_a_released_version() {
  local repo="$1"
  local version head tagged
  version="$(declared_version "$repo/.claude-plugin/plugin.json")" || return 1
  [ -n "$version" ] || return 1
  head="$(git -C "$repo" rev-parse HEAD)" || return 1
  tagged="$(git -C "$repo" rev-parse -q --verify "refs/tags/v$version^{commit}" 2>/dev/null)" || {
    echo "no tag v$version exists, so this commit is not a released version" >&2
    return 1
  }
  if [ "$tagged" != "$head" ]; then
    echo "tag v$version names $tagged, but HEAD is $head" >&2
    return 1
  fi
  return 0
}

# --- Throwaway repositories the predicate is exercised over -------------------
#
# Built rather than fixtured, because the question is about tags and a tag is
# not a file a fixture directory can carry.

a_repo_declaring() {
  local dir="$1"
  local version="$2"
  mkdir -p "$dir/.claude-plugin"
  printf '{ "name": "probe", "version": "%s", "skills": "./skills/" }\n' \
    "$version" > "$dir/.claude-plugin/plugin.json"
  git -C "$dir" init -q
  git -C "$dir" config user.email probe@example.invalid
  git -C "$dir" config user.name probe
  git -C "$dir" add -A
  git -C "$dir" -c commit.gpgsign=false commit -qm "probe"
}

tagged_repo="$tmp/tagged"
a_repo_declaring "$tagged_repo" 1.2.3
git -C "$tagged_repo" tag -a v1.2.3 -m "v1.2.3"

untagged_repo="$tmp/untagged"
a_repo_declaring "$untagged_repo" 1.2.3

misplaced_repo="$tmp/misplaced"
a_repo_declaring "$misplaced_repo" 1.2.3
git -C "$misplaced_repo" tag -a v1.2.3 -m "v1.2.3"
printf 'a later commit\n' > "$misplaced_repo/LATER.md"
git -C "$misplaced_repo" add -A
git -C "$misplaced_repo" -c commit.gpgsign=false commit -qm "after the tag"

# A tree whose manifests disagree, and one whose catalogue does, so the two
# manifest checks are held in the direction that can actually go wrong.
drifted="$tmp/drifted"
mkdir -p "$drifted"
cp -R "$PWD"/.*-plugin "$drifted/"
python3 - "$drifted/.codex-plugin/plugin.json" <<'PY'
import json, sys
path = sys.argv[1]
manifest = json.load(open(path))
manifest["version"] = "9.9.9"
json.dump(manifest, open(path, "w"), indent=2)
PY

advertised_drift="$tmp/advertised-drift"
mkdir -p "$advertised_drift"
cp -R "$PWD"/.*-plugin "$advertised_drift/"
python3 - "$advertised_drift/.claude-plugin/marketplace.json" <<'PY'
import json, sys
path = sys.argv[1]
catalogue = json.load(open(path))
catalogue["plugins"][0]["version"] = "9.9.9"
json.dump(catalogue, open(path, "w"), indent=2)
PY

pin_drift="$tmp/pin-drift"
mkdir -p "$pin_drift"
cp -R "$PWD"/.*-plugin "$pin_drift/"
python3 - "$pin_drift/.claude-plugin/marketplace.json" <<'PY'
import json, sys
path = sys.argv[1]
catalogue = json.load(open(path))
catalogue["plugins"][0]["source"] = {
    "source": "github",
    "repo": "Okja-Engineering/skill-architect",
    "ref": "v9.9.9",
}
json.dump(catalogue, open(path, "w"), indent=2)
PY

# --- Which half of the rule this run falls under ------------------------------
#
# Read from the event rather than from the checkout, because on a pull request
# the checked-out ref is a merge commit on a detached head and the branch the
# result is destined for is `GITHUB_BASE_REF`. A push carries `GITHUB_REF_NAME`.
# Locally there is neither, and the branch is the branch.
#
# Printed, not assumed. A suite that silently decides it is not on the release
# branch and reports nothing is the failure mode this whole file is written
# against.
branch_under_test() {
  if [ -n "${GITHUB_BASE_REF:-}" ]; then
    printf '%s\n' "$GITHUB_BASE_REF"
  elif [ -n "${GITHUB_REF_NAME:-}" ]; then
    printf '%s\n' "$GITHUB_REF_NAME"
  else
    git rev-parse --abbrev-ref HEAD
  fi
}

branch="$(branch_under_test)"
echo "  branch under test: $branch"
echo "  release branch:    $RELEASE_BRANCH"

# --- Preconditions ------------------------------------------------------------

require "the repository holds at least one plugin manifest to compare" \
  test -n "$(manifest_dirs "$PWD")"
require "the release-branch predicate has a repository to read a version from" \
  test -f .claude-plugin/plugin.json

# --- The predicate, in both directions, on every run --------------------------

assert "a head the version's tag names is a released version" \
  quietly head_is_a_released_version "$tagged_repo"
assert "a head with no tag for its version is refused" \
  refuses head_is_a_released_version "$untagged_repo"
assert "a head the version's tag does not name is refused" \
  refuses head_is_a_released_version "$misplaced_repo"

# --- The manifest checks, in both directions ----------------------------------

assert "every plugin manifest in this repository declares the same version" \
  quietly manifests_agree_on_the_version "$PWD"
assert "a tree whose manifests disagree on the version is refused" \
  refuses manifests_agree_on_the_version "$drifted"

assert "no marketplace entry advertises a version the plugin does not install" \
  quietly marketplace_entries_agree "$PWD"
assert "an entry advertising a version the plugin does not install is refused" \
  refuses marketplace_entries_agree "$advertised_drift"
assert "an entry pinning a ref that disagrees with its version is refused" \
  refuses marketplace_entries_agree "$pin_drift"

# --- The live tree, under whichever half of the rule its branch falls ---------

if [ "$branch" = "$RELEASE_BRANCH" ]; then
  # Without tags the predicate cannot distinguish "never released" from "not
  # fetched", and it would answer the first when the truth is the second. That
  # is a verdict this suite must not reach, so it halts instead: the workflow
  # checks out with `fetch-depth: 0` precisely so this holds.
  require "tags are present, so an absent tag means unreleased and not unfetched" \
    test -n "$(git tag --list 'v*')"
  assert "$RELEASE_BRANCH carries a version some tag names" \
    head_is_a_released_version "$PWD"
else
  echo "  the live release-branch check did not apply on this branch"
  assert "this run is not on the release branch, so the rule above is the release branch's" \
    test "$branch" != "$RELEASE_BRANCH"
fi

harness_summary
