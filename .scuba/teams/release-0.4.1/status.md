# Status — v0.4.1 patch release

**Branch:** `release/0.4.1` (based on `origin/main` @ 30f374c)
**Worktree:** `/Users/matthewvandusen/Development/Auraprix/skill-architect-wt/release-0.4.1`
**Last SHA:** _(none yet — baseline verified)_
**PR URL:** _(not yet opened)_

## Baseline verification (at 30f374c, before any change)

```
cd profiler && go build ./... && go vet ./... && go test ./...
ok  	github.com/auraprix/skill-architect/profiler	0.325s
?   	github.com/auraprix/skill-architect/profiler/cmd	[no test files]

tests/test_skill.sh  → 22 passed, 0 failed
tests/test_walk.sh   → 19 passed, 0 failed
tests/test_f01.sh    → 32 passed, 0 failed
tests/test_f02.sh    → 58 passed, 0 failed   (131 total)
```

Go test count at base: 15 test functions in `profiler/profiler_test.go` (confirms item 11 — docs say 14).

## Per-item progress

| # | Item | State | Evidence |
|---|---|---|---|
| 1 | Capture gates whole OTel path on token capability | in progress | — |
| 2 | README Prerequisites block | todo | — |
| 3 | README sibling `skill-audit` requirement | todo | — |
| 4 | module path → Okja-Engineering | todo | — |
| 5 | CI `go-version-file` | todo | — |
| 6 | spec: real `RawMetricResult` shape | todo | — |
| 7 | spec AC2 probe semantics | todo | — |
| 8 | spec fallback reason string | todo | — |
| 9 | delete `CaptureOpts.OtelEndpoint` | todo | — |
| 10 | "snapshot-pinned" reword | todo | — |
| 11 | test count 14 → real number | todo | — |
| 12 | CHANGELOG gitignored path | todo | — |
| 13 | CHANGELOG 0.3.1 references/ note | todo | — |
| 14 | `ToolCallEntry.Duration` removal note | todo | — |
| 15 | "no skill-level events" softening | todo | — |
| 16 | version → 0.4.1 (4 manifests + test) | todo | — |
| 17 | CHANGELOG/RELEASE_NOTES 0.4.1 sections | todo | — |
| 18 | document version/--export-file/per-signal probe | todo | — |

## Found, not fixed

_(nothing yet)_
