# PR-2 — profiler/hook-spool-and-adapters

- **Stage:** ready to dispatch — blocked on prototype-mode lift
- **Owner:** senior-implementer
- **Branch:** `profiler/hook-spool-and-adapters` (not created)
- **Base:** `origin/main`; carries commits `fd2cfe5..0a83615` (7 unpushed:
  F03 adapters + F04 compare/experiment + two skill-rewrite commits)
- **Depends on:** nothing (parallel with PR-0)
- **Blocks:** PR-3 (needs b73326d/f13173e), therefore PR-1

## Goal
Land the entire profiler program: the 7 local commits plus the uncommitted
hook-spool/adapter work.

## Files
- The 7 commits fd2cfe5..0a83615
- Untracked: `profiler/{analyze,doctor,hooks,hooks_install,spool_profile}.go`
  + tests, `strict_test.go`, `compare_test.go`, `queries/`
- Modified profiler sources (see `git status`)
- `profiler/go.mod` + `cmd/main.go:10` module rename — **done in tree**
- `docs/profiler-spec.md` — sections for ingest, doctor, hooks (states that
  `hooks install` writes `~/.cursor/hooks.json`), analyze, experiment

## Gate
`cd profiler && go build ./... && go vet ./... && go test ./...`;
usage lists 9 commands; 4 shell suites.

## Next
Dispatch on lift, parallel with PR-0. Branch carries the 7 commits, so
`origin/main..HEAD` must equal the chain plus PR-2's work.
