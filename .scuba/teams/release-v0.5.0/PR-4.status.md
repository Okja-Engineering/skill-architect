# PR-4 — release/v0.5.0

- **Stage:** queued — blocked on PR-1 merge (and lift)
- **Owner:** steward
- **Branch:** `release/v0.5.0` (not created)
- **Base:** `main` after PR-1 merges
- **Depends on:** PR-1
- **Blocks:** tag + `gh release`

## Goal
Version lockstep to 0.5.0 across every surface, then tag twice and release.

## Files / edits
- Four plugin manifests 0.4.0 → 0.5.0
- `tests/test_skill.sh:27` → '0.5.0' + new assert that `skillgate/gate.go`
  Version matches manifests
- `skillgate/gate.go:9` → `const Version = "0.5.0"`
- `profiler/types.go:210` → `const AdapterVersion = "0.5.0"` (lockstep —
  ratified 2026-09-13); `test_skill.sh` asserts it alongside `gate.go`
- Four manifests' `description` fields — mention the safety gate
  (marketplace surface; consistency-pass finding)
- Three SKILL.md `metadata.version` → 0.5.0
- `CHANGELOG.md` Unreleased → `## 0.5.0 — <date>` + fresh empty Unreleased
- `RELEASE_NOTES.md` v0.5.0 section: gate contract, install command,
  descope list, profiler commands, Devin placeholder

## Gate
Full per-PR verification suite (see plan.md), then release verification:
`GOPROXY=direct go install ...@v0.5.0` + clone-at-tag test run.

## Next
After PR-1 merge. Tag merge commit `v0.5.0` + `skillgate/v0.5.0`, push both,
clean-machine install check, `gh release create v0.5.0`. If install fails:
delete both tags before proxy caches, fix, re-tag.
