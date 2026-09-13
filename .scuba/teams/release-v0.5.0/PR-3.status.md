# PR-3 — skill-rewrite/self-contained

- **Stage:** ready to dispatch once PR-2's branch exists — blocked on lift
- **Owner:** bug-fixer
- **Branch:** `skill-rewrite/self-contained` (not created)
- **Base:** PR-2 branch (needs skill-rewrite commits b73326d/f13173e)
- **Depends on:** PR-2
- **Blocks:** PR-1 (`gate_test.go:168` clean-corpus test requires
  skill-rewrite self-contained / zero SK-T findings)

## Goal
Make `skills/skill-rewrite` self-contained: vendored scripts/references,
`allowed-tools` declared, no `../` escapes.

## Files
- `skills/skill-rewrite/SKILL.md` (allowed-tools, no `../`)
- `scripts/{draft-rewrite,audit-report,check-frontmatter,check-paths,check-structure}.sh`
- `references/**`

## Gate
`grep -rn '\.\./' skills/skill-rewrite` empty; 4 shell suites; from a PR-1
worktree `skillgate gate ../skills/skill-rewrite` → CAUTION, zero SK-T.

## Next
Dispatch when PR-2's branch exists. Most content already in tree
(uncommitted) — worker moves it onto the branch.
