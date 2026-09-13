# PR-0 — docs/research-and-repairs

- **Stage:** ready to dispatch — blocked on prototype-mode lift
- **Owner:** bug-fixer
- **Branch:** `docs/research-and-repairs` (not created)
- **Base:** `origin/main` (independent — can open first)
- **Depends on:** nothing
- **Blocks:** nothing in the chain, but PR-1 wants `.gitignore` + README notes

## Goal
Commit the canonical research corpus and the accumulated doc repairs as the
release's independent lead-off PR. `docs/research/**` stays permanently
(user call 2026-09-13 — no post-release deletion).

## Files
- `.gitignore` (`.venv*/` + `.scuba/` — already edited in tree)
- `docs/research/**` (38 files) + one README banner noting recommendation
  §6/PR table and HANDOFF describe unbuilt subcommands, superseded by
  `docs/skillgate-spec.md`
- `.out-of-scope.md`
- `skills/skill-audit/SKILL.md` (degrade instead of exit 1 — already in tree)
- `README.md` — Devin row → placeholder; Development note that
  `go build ./...` at root fails by design (run per module)
- `CHANGELOG.md` — Unreleased bullets for profiler + skill-rewrite
- `docs/skillgate-intent.md` — reconcile fall-away inventory (:62-69) with
  the keep-research decision

## Gate
4 shell suites pass; `git ls-files docs/research | wc -l` = 38;
no `.venv*`/`.scuba` in status.

## Next
Dispatch on prototype-mode lift. Base `origin/main`; the `.gitignore` edit
is already in the working tree — the worker carries it onto the branch.
