#!/usr/bin/env bash
#
# The out-of-scope barrier: refuse a commit that stages work belonging to a
# later release.
#
# A body of 0.6.0 work — skillgate/, skills/skill-gate/, its docs and research,
# the go workspace files and NOTICE — lives in the working tree beside the
# 0.5.0 release. `.gitignore` keeps most of it unstageable and tests/test_harness.sh
# asserts that it does, but the index is the last place to ask: a path added
# with `git add -f`, or one whose ignore entry was narrowed, reaches a commit
# with nothing between it and `main`.
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

# The paths that belong to a later release. `NOTICE$` is anchored on both ends
# deliberately: the root file is out of scope and a NOTICE inside a vendored
# directory somebody adds later is a different question.
out_of_scope_pattern='^(skillgate/|skills/skill-gate/|docs/skillgate|go\.work|NOTICE$|docs/research/|\.venv)'

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
