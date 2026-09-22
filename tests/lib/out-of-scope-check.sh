#!/usr/bin/env bash
#
# The out-of-scope barrier: refuse a commit that stages work belonging to a
# later release.
#
# Work for a later release lives in the working tree beside the current one.
# `.gitignore` keeps some of it unstageable and tests/test_harness.sh asserts
# that it does, but the index is the last place to ask: a path added with
# `git add -f`, or one whose ignore entry was narrowed, reaches a commit with
# nothing between it and `main`.
#
# Through 0.5.0 the list below carried the whole skillgate program. 0.6.0 ships
# it, so skillgate/, skills/skill-gate/, docs/skillgate*, the go workspace files
# and NOTICE are gone from here. They are named in this sentence rather than
# deleted silently, because the comment that stood here attributed every entry
# to 0.6.0 and two of them were never 0.6.0's — see below.
#
# It is a script because it used to be a sentence. Every slice of the release
# re-derived the same pipeline from a ruling, by hand, which is the copy-by-hand
# pattern tests/lib/harness.sh exists to end: the copies drift, and the drift is
# invisible because each copy is only ever run once.
#
# One correction came with it. The form in the plan was
#
#     git diff --cached --name-only | grep -E '<pattern>' && { echo "..."; exit 1; }
#
# and as a script's last line that is inverted. `grep` exits 1 when it matches
# nothing, so a clean index leaves the script exiting 1 with no message, and a
# caller — or a pre-commit hook — reads a refusal exactly when there was nothing
# to refuse. Whoever then "fixes" the hook by dropping it has removed the
# barrier. So this reports an explicit verdict in both directions and never
# leaves a bare grep status as the answer.
#
# Usage: tests/lib/out-of-scope-check.sh [repo-dir]    (default: the cwd)
# Exit:  0 — nothing out of scope is staged
#        1 — an out-of-scope path is staged, and it is named
#        2 — the check could not be run, which is not a clean verdict

set -euo pipefail

# The paths that do not belong in a commit, and what each one is:
#
#   docs/research/  — 0.7.0. The prior comment filed it under 0.6.0, and it is
#                     not: the roadmap ladder places it at 0.7.0, and a standing
#                     user ruling keeps it permanently rather than letting it
#                     fall away at release, so it will be *added* one day, not
#                     dropped from this line as 0.6.0's entries were.
#   \.venv          — never. A virtualenv is a local build artifact of any
#                     release. Deliberately not `\.venv/`: the tree carries
#                     `.venv-skillspector/`, 15k files, and the narrower spelling
#                     would let it through.
#
# `NOTICE` used to be a third arm, filed under 0.6.0 and anchored `NOTICE$` so a
# vendored copy stayed a separate question. It belonged to no release: it exists
# to carry the licence of the code ported into skillgate/difftest/, so it landed
# with that code and the arm is gone. Nothing replaced the anchor — there is no
# arm left for a vendored NOTICE to be confused with.
#
# The `^` and the trailing `/` on `docs/research/` are both load-bearing;
# tests/test_harness.sh's near-miss control holds each of them with a path that
# would be refused if it were lost.
out_of_scope_pattern='^(docs/research/|\.venv)'

repo="${1:-.}"

if ! git -C "$repo" rev-parse --git-dir >/dev/null 2>&1; then
  echo "out-of-scope check: cannot run — $repo is not a git repository" >&2
  exit 2
fi

# The staged list is captured before it is searched, rather than piped into
# grep. `grep -q` exits on its first match and the writer upstream takes a
# SIGPIPE for it, so under `set -o pipefail` the pipeline reports 141 — and a
# condition testing that pipeline reads the block as a clean run. Two variables
# cost nothing and the question stays the one being asked.
staged="$(git -C "$repo" diff --cached --name-only)"
offenders="$(printf '%s\n' "$staged" | grep -E "$out_of_scope_pattern" || true)"

if [ -n "$offenders" ]; then
  echo "BLOCKED: out-of-scope path staged"
  printf '%s\n' "$offenders" | sed 's/^/  /'
  exit 1
fi

echo "out-of-scope check: clean"
exit 0
