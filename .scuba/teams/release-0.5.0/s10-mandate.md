# S10 — the v0.5.0 release commit

**Branch:** `release/0.5.0`, cut in a worktree from `origin/main` @ `10326b3`.
**Base verified by CoS:** 0 open PRs; S1–S9 all squash-merged; history linear.
**Never draft the PR.** One writer; this branch is yours alone.

## Goal

Write the release surface for 0.5.0. This is the last slice of the release and the
only one that makes claims about the whole of it. **No behaviour changes.** No Go
file, no shell script under `skills/` or `tests/` may change, except the version
asserts named below. If you believe a code change is required, stop and report it.

## What lands

1. **The five plugin manifests, 0.4.3 → 0.5.0.** All five, verified by CoS:
   `.claude-plugin/plugin.json`, `.codex-plugin/plugin.json`,
   `.cursor-plugin/plugin.json`, `.devin-plugin/plugin.json`, and
   `.claude-plugin/marketplace.json` (line 13 — this fifth one is easy to miss; the
   plan's own prose says "four manifests" in places and it is wrong).
2. **The version asserts in `tests/test_skill.sh`** — `:44`, `:58`, `:96`. These are
   the only test edits permitted.
3. **A new `## 0.5.0` section in `CHANGELOG.md`** above 0.4.3, under the existing
   `## Unreleased` heading. A *new section* — do not edit the 0.4.3 or 0.4.0 entries.
4. **A new 0.5.0 section in `RELEASE_NOTES.md`.**
5. **README capability tables**, where 0.5.0 changed what the tool does.
6. **No tag.** The `v0.5.0` tag is the user's; they have pushed all four existing tags
   themselves. Do not create or push a tag. Say in your report that it is theirs.

## What the notes MUST state — each of these is owed

These are handoffs from the slices that shipped. A release note that omits one is
incomplete, and three of them exist because a slice deliberately declined to ship
something rather than ship a thing that does not work.

- **The removed `doctor` tiers.** `hooks`, `server_api` and `enterprise` are gone.
  This needs its own line, because a user upgrading from a preview that advertised
  them will notice, and **the reason is the point**. Source: S8 §8.2, restated S9 §7.1.
- **`experiment run` has no time limit.** It shells out in a loop with no cap. S6
  dropped the draft's `Budget` rather than ship a cap that does not cap. **State
  plainly that there is no cap.** You may note that this release proved a portable
  bash-3.2-safe deadline in `tests/lib/mutation-runner.sh` (`mutation_bounded`) and
  that the Go side has `exec.CommandContext` — but that is a test library and nothing
  in `experiment run` sources it, so it does not soften the sentence. Source: S6 §9.1, S9 §7.1.
- **The hook spool's verification is documentary only.** The payload shapes are
  `[DOCS]` — asserted from documentation, not measured against a live emitter.
- **`AppendSpool` has no locking.** Source: S7 §9.5. Still owed a line.
- **No Cursor, Codex or Devin adapter ships.** All three were deleted, deliberately,
  on a measured finding that the gap was not a porting gap. README still says
  "Planned" for all three and must keep saying it. The Cursor *spool* does ship and
  `analyze` summarises it — the README row already draws that distinction; preserve it.
- **`skill-rewrite`'s `draft-rewrite.sh` gains `-o|--output`, with its bound.** Name
  the four refusals, because a caller scripting `-o` will hit one: an existing
  directory, a symlink, any `SKILL.md`, and anything inside
  `$HOME/{.claude,.cursor,.codex,.devin,.config}`. The bound is decided **after**
  resolving the path. The last refusal exists because an agent reads
  `~/.claude/skills/` as skills, so a draft written there may be loaded as instructions.
- **What this release does not close.** The deferred ledger's open 0.5.0 items
  (`.scuba/teams/gap-audit-0.4.3/deferred-ledger.md`, the `L*` rows). Count them
  yourself against that file; do not quote a number from this mandate or from the plan.
- **R4, as an explicitly accepted risk.** **Verified open by CoS:** `ci.yml` runs
  `ubuntu-latest` only, so every "green on CI" claim in this release is a bash-5
  claim. The bash 3.2 half is a local macOS step. This matters specifically because
  C7's finding in 0.4.3 was that bash 3.2 exempts `[[ ]]` from `errexit` and bash 5
  does not. The plan (§R4) permits either a `macos-latest` job or a recorded risk;
  **S1 did not add the job, so record the risk.** Do not add the job — that is new CI
  surface and belongs to 0.6.0.

## The open user decision — state it, do not decide it

Skill `metadata.version` fields are **untouched and stay untouched**: `skill-audit`
declares `0.2.0`, `skill-rewrite` declares `0.1.0`, while the manifests go to 0.5.0.
This is the user's call and it is now more conspicuous, because `skill-rewrite`'s
documented interface gained a flag in S9 while still declaring `0.1.0`.

**Put one honest line in the release notes** saying the skill metadata versions are
independent of the plugin version and where they stand. Do not bump them. Do not
add an assert pinning them. The user decides before tagging; the notes should not
pre-empt the decision, and should not hide it either.

## Derive every number. Do not quote one.

**Five figures have gone stale in this release already.** Every count in the
changelog, the release notes and the README must be produced by a command you ran on
this branch, and your report must give the command beside the number. Specifically:
the shell assertion total on **both** bash 3.2.57 and 5.3.15, the Go test total with
its per-package breakdown, the suite count, and the deferred-item count. S9 measured
2649 and 275 (233 profiler / 34 cmd / 8 homesafe) — **treat those as unverified and
re-derive them.** If yours differ, yours are right and say so.

## Verification

- All seven shell suites on **both** `/bin/bash` (3.2.57) and bash 5.
- `cd profiler && go build ./... && go vet ./... && gofmt -l . && go test ./...`
- `tests/test_skill.sh` with the asserts at 0.5.0.
- The plan's definition of done. **CoS verified five of these already pass at
  `10326b3`** — no `skillgate`/`skill-gate`/`go.work`/`NOTICE` in the tree, no
  `cache_write`/`CacheWrite`, no `profiler/{cursor,codex,devin}.go`,
  `AdapterVersion == "0.5.0"`, `ProfileSchema == "skill-architect/profile/v1"`.
  Re-check them on your branch so the release commit is not the thing that breaks one.
- **The prose gate.** This repo's release ritual includes a claim-by-claim pass,
  because the equivalent gate on PR #5 found six false claims and the one on 0.4.3's
  release commit found three — one of which the repo's own suite asserts must be
  refused. Every sentence you write that asserts a fact about the code must be one
  you drove against the code. Dogfooding is a release gate (AGENTS.md).

## Authorship — hard constraint

`imagineux` is the only valid author. **Do not add `Co-Authored-By` trailers, and do
not add a "Generated with" line to the commit or the PR body.** No AI co-author, on
any commit or PR, for any reason. If tooling suggests one, drop it.

## Deliverables

- The branch pushed, PR opened (not draft) against `main`.
- `.scuba/teams/release-0.5.0/s10-record.md` — written with Write/Edit, **never a
  Bash heredoc** (a heredoc truncates silently on a broken shell and reports success).
  It must carry: every number with the command that produced it; each owed item above
  with the file and line where you stated it; and anything you found that the release
  notes should say and you could not verify.
- Report back: what you measured, what you could not, and the tag left undone.

Artifacts go to `/Users/matthewvandusen/Development/Auraprix/skill-architect/.scuba/teams/release-0.5.0/`
by absolute path — never inside your worktree. A permission hook denies tracked-code
writes outside the session scratchpad; work in your worktree and write control-plane
files by absolute path.
