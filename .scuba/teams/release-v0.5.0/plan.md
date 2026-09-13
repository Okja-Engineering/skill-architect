# v0.5.0 "skill-gate" — release plan

**Status:** staged · awaiting prototype-mode lift (user call 2026-09-13: "not yet")
**Integration:** `main` via user merges · one writer per branch · never draft PRs
**Control plane:** `.scuba/teams/release-v0.5.0/` (gitignored, local-only)

## What ships

The `skillgate` safety gate as the third skill, an installable binary
(`go install github.com/Okja-Engineering/skill-architect/skillgate/cmd/skillgate@v0.5.0`),
plus the profiler work already committed locally (F03 adapters, F04
compare/experiment, hook spool). Promises: every file gets a ledger outcome;
no verdict above CAUTION with a skipped check; binary absent ⇒ named skip,
never "safe".

On top of the staged chain (boundary ratified 2026-09-13 — contract +
zero-risk wins; see `problem-brief.md`): G003 reachability +
`reachability` report block + shared convention-loads seed table;
`--list-checks`/`--only` + parallel native checks; `difftest/` module
carve; E2-verified raw-lane miss fixes (BH1/TP1 ports, T017 http-hook
leg, data-URI leg); spec sentence naming ceded lanes.

Explicitly not in v0.5.0: `skillgate env`, `skillgate scan`,
`calibration.jsonl`, Pack D tri-state delta, slices 4–6. **0.6 epic**
(one epic, `docs/research/rule-language.md` spec): derived text views,
generated Unicode tables, semgrep ruleset + leg, eval-artifact model +
manifest consolidation, typed-`Rule` migration.

## Decisions (ratified 2026-09-13 unless noted)

1. **Prototype mode** — user: **not yet**. All git ops blocked; everything
   else is staged in the uncommitted tree.
2. **Module rename** — `github.com/Okja-Engineering/skill-architect/{skillgate,profiler}`.
   Done in tree (both `go.mod` + one import each).
3. **Version lockstep** — `skillgate/gate.go` Version tracks the plugin
   version: 0.5.0. Extended 2026-09-13: `profiler/types.go:210`
   `AdapterVersion` locksteps too (PR-4 + test assert).
4. **docs/research/\*\*** — commit in PR-0 and **keep permanently** (user:
   "it really is for you"). The post-release deletion chore is cancelled;
   `docs/skillgate-intent.md:62-69` fall-away inventory needs a reconciling
   line in PR-0 or the release PR.
5. **difftest ships in PR-1, carved into its own module** — fixed and
   green this session; NOTICE rides with it. The carve is required: its
   `x/text` require in the gate module's go.mod would falsify the
   zero-requires constraint (ratified 2026-09-13).

## Pre-flight — DONE in tree (uncommitted)

- `.gitignore`: `.venv*/` + `.scuba/` added; `.scuba/.gitignore` restored to `*`.
  Check passes: `git status --short | grep -E '^\?\? (\.venv|\.scuba)'` → empty.
- Module rename applied; `go mod tidy` pulled `golang.org/x/text v0.42.0`
  (difftest was missing its require — new `skillgate/go.sum`). **Interim
  state only**: PR-1 carves `difftest/` into its own module; the gate
  module returns to zero requires.
- `skillgate/difftest/ports.go:467` — upstream `{0,8192}` repeat exceeds RE2's
  1000 cap → ported to unbounded `*`. **Fidelity divergence:** extras vs
  upstream possible on >8192-char windows; the differ will surface them.
- `cd skillgate && go build/vet/test ./...` green; `cd profiler && ...` green.

## PR chain — D15 topology + release flip

Order: **PR-0 independent; PR-2 → PR-3 → PR-1 → PR-4 stacked.** Workers may
build PR-1 and PR-3 in parallel; PRs open in chain order so each is green at
open. Each PR passes `ship-gate` (hunter swarm + steward) before user merges.

| PR | Branch | Base | Role |
|----|--------|------|------|
| PR-0 | `docs/research-and-repairs` | `origin/main` | bug-fixer |
| PR-2 | `profiler/hook-spool-and-adapters` | `origin/main` + carries fd2cfe5..0a83615 | senior-implementer |
| PR-3 | `skill-rewrite/self-contained` | PR-2 branch | bug-fixer |
| PR-1 | `skillgate/gate-v1` | `main` after PR-3 merge | senior-implementer |
| PR-4 | `release/v0.5.0` | `main` after PR-1 merge | steward |

Per-PR file lists, gates, and recovery fields: `PR-*.status.md` in this dir.

## Tag and release

Tag the PR-4 merge commit twice: `v0.5.0` (plugin) **and** `skillgate/v0.5.0`
(Go resolves a nested module only from a subdirectory-prefixed tag). Push both,
verify clean-machine install with `GOPROXY=direct` first then default proxy,
then `gh release create v0.5.0` with the RELEASE_NOTES section. If install
fails: delete both tags before the proxy caches, fix, re-tag.

## Verification

Per PR:

```bash
(cd profiler  && go build ./... && go vet ./... && go test ./...)
(cd skillgate && go build ./... && go vet ./... && go test ./...)
(cd skillgate/difftest && go build ./... && go vet ./... && go test ./...)   # after PR-1's module carve
tests/test_skill.sh && tests/test_walk.sh && tests/test_f01.sh && tests/test_f02.sh
tests/test_gate.sh   # from PR-1 on
```

Gate behaviour (PR-1 on):

```bash
(cd skillgate && go run ./cmd/skillgate gate ../skills/skill-gate; echo exit=$?)        # CAUTION, exit=0
(cd skillgate && go run ./cmd/skillgate gate ../skills/skill-rewrite; echo exit=$?)     # CAUTION, 0 SK-T, exit=0
(cd skillgate && go run ./cmd/skillgate gate testdata/malicious-bundle; echo exit=$?)   # REJECT, exit=1
(cd skillgate && PATH=/usr/bin:/bin go run ./cmd/skillgate gate ../skills/skill-gate | jq -r '.checks_skipped[].check')
```

Release (after tags, before `gh release create`):

```bash
cd "$(mktemp -d)" && GOPROXY=direct go install github.com/Okja-Engineering/skill-architect/skillgate/cmd/skillgate@v0.5.0 && "$(go env GOPATH)/bin/skillgate" version
git clone --depth 1 -b v0.5.0 https://github.com/Okja-Engineering/skill-architect.git rel && cd rel && tests/test_skill.sh && (cd skillgate && go test ./...)
```

## Execution shape (scuba stack)

- Chief of staff wears `team-manager`: epic = v0.5.0, integration on `main`
  via user merges, one writer per branch.
- Dispatch PR-0 and PR-2 in parallel (background); PR-3 when PR-2's branch
  exists; PR-1 when PR-3's branch exists. `ship-gate` per PR; `steward`
  closeout; user merges.
- `process-health-monitor` tick ~10 min; `scribe` keeps roadmap current.
- `brief-specialist`: architecture brief before PR-1 build; executive brief
  at the v0.5.0 tag → `.scuba/briefs/release-v0.5.0.html`.

## Findings added this session (2026-09-13)

- **`NOTICE`** (untracked, NVIDIA SkillSpector attribution for
  `skillgate/difftest/`) was in no PR file list → assigned to **PR-1**.
- **`skillgate/difftest` was broken on arrival**: missing `x/text` require
  (fixed via `go mod tidy` → new `go.sum`) and an init-time `MustCompile`
  panic on `{0,8192}` (fixed, see pre-flight). Its differential test
  self-skips without `SKILLSPECTOR_PYTHON`; the panic pre-empted the skip.
- PR ordering constraint stands: `gate_test.go:168`
  `TestCleanCorpusHasNoTripwireFindings` gates `skills/skill-rewrite` on
  self-containment → PR-3 must land before PR-1 opens; PR-3 needs commits
  b73326d/f13173e carried by PR-2.
- **Live-gate verification of E2 misses (2026-09-13):** the difftest port
  misses have live-gate siblings — `hooks/hooks.json` with a
  `"type":"http"` hook to `exfil.example` gates **CAUTION, zero findings**
  (T017 only reads `"command"` values); a `data:text/plain;base64,` blob
  decoding to injection text → zero findings; `mcp_poisoned_tool` REJECTs
  via T002/T003/T013/I002 (the TP1 classes are covered by other rules).
  `bundle-benign`'s `command: echo` fires T017 — deliberate (bare command
  names can PATH-resolve into the bundle), not an FP. All land in PR-1's
  raw-lane fix scope.
