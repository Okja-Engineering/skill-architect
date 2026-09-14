# v0.4.2 release commit — apply record

_Implementer, 2026-09-14. Executed against `release-commit-draft.md` and
`release-commit-requirements.md`. Applied, verified, pushed, PR opened. **Not merged, not
tagged, no tag pushed** — the user merges to `main` and owns the tag._

## Result

| | |
|---|---|
| Worktree | `/Users/matthewvandusen/Development/Auraprix/skill-architect-wt/release-0.4.2-version` |
| Branch | `release/0.4.2-version` |
| Base | `origin/main` = `5c847e1` (confirmed parent of the commit) |
| Commit | `13b31410d7f071861a07b00d700d150f7051b13c` |
| PR | https://github.com/Okja-Engineering/skill-architect/pull/5 — OPEN, not draft, `release/0.4.2-version` → `main`, MERGEABLE |
| CI | `test` **pass**, 1m13s |
| Author / committer | `imagineux <imagineux@gmail.com>` on both, no trailers |
| Tags | none created, none pushed; `git tag -l 'v0.4.2*'` is empty locally and on origin |
| Primary tree | untouched — still `main` at `0a83615` with 21 modified tracked files |

## Version surfaces — 16 lines, 9 files, all `0.4.1` → `0.4.2`

`.claude-plugin/plugin.json:4` · `.codex-plugin/plugin.json:3` ·
`.cursor-plugin/plugin.json:3` · `.devin-plugin/plugin.json:3` ·
`.claude-plugin/marketplace.json:13` · `profiler/types.go:233,234` ·
`profiler/profiler_test.go:408,410` · `tests/test_skill.sh:28,48,51,57,58` ·
`README.md:55,327`

`profiler/profiler_test.go:410` is the second independent pin (`const want` in
`TestAdapterVersionIsThisRelease`); it was moved, and CI would have failed without it.

Left byte-for-byte as history: `CHANGELOG.md:7,22`; `README.md:115,116`;
`RELEASE_NOTES.md:3,7,26`; `docs/profiler-spec.md:217,218`;
`profiler/testdata/otlp/README.md:61,74,99`. The released `## 0.4.1` / `## v0.4.1` entries
and everything below them diff clean against `2ed34b8`.

## Where the draft was wrong

1. **`skills/skill-audit/README.md` does not exist.** The changelog's Documentation section
   claimed that file gains a `jq` row and a `DEP002` paragraph. The repo has two READMEs only.
   That content is in the root `README.md` (269–291). Corrected the bullet; then swept all 16
   backticked path tokens in the new prose — the other 15 resolve.
2. **"16 lines across 8 files"** in the draft's §1 header is wrong; its own table lists 9, and
   9 is correct.
3. The brief named `profiler/testdata/otlp/README.md` lines **58 and 83** as the historical
   prose to leave alone. The real historical hits are **61, 74 and 99** — three, not two. All
   three left untouched.
4. The draft's predicted shift `README.md:269 → :327` **held exactly**; verified, not assumed.
   The three false-claim line numbers (`CHANGELOG.md:15`, `:47`, `RELEASE_NOTES.md:20`) also
   still resolve, since neither PR touched those files. Located by text regardless.

## Counts, all re-measured on this tree

66 test functions (61 profiler + 5 CLI) · 230 subtests (200 direct + 30 nested one deeper) ·
55 OTLP fixtures · 850 shell assertions (`test_f01` 561, `test_f02` 243, `test_skill` 27,
`test_walk` 19). The 0.4.1 baselines the prose compares against — 58 / 179 / 134 / 39 — match
the released v0.4.1 entries and the tree at `2ed34b8`.

## Verification, from the worktree root

`gofmt -l .` clean · `go build ./...` OK · `go vet ./...` OK ·
`go test -count=1 ./...` ok (0.535s / 1.348s) · `go test -race -count=1 ./...` ok
(1.575s / 2.253s) · four shell suites 850 passed / 0 failed, run from the repo root because
`tests/test_skill.sh` has no `cd` of its own · all four re-run under `LC_ALL=C` and
`LC_ALL=en_US.UTF-8`, 850/0 under each · `skill-validator check` `passed: true`, 0 errors,
0 warnings for both shipped skills.

## Open for the user

- **Skill metadata versions were NOT bumped and this is unresolved.**
  `skills/skill-audit/SKILL.md` `metadata.version: "0.2.0"` and
  `skills/skill-rewrite/SKILL.md` `"0.1.0"`. PR #4 changed skill-audit's scripts substantially
  and added two consumer-visible rule IDs, which by repo convention (`7607210`; the 0.3.1
  changelog) would be a bump. The requirements document does not call for it. Out of scope as
  directed, recorded as deferred in the PR body.
- **The tag is the user's.** `git tag -a v0.4.2 -m 'v0.4.2 — correct cumulative token counts,
  guard every verdict'` mirrors `v0.4.1`'s annotated form, to be run after merge by the user.
- `skillgate/` excluded throughout: untracked, absent from `main` and CI, not part of this
  release or its verification suite.
