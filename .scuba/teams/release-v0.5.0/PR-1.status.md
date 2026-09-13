# PR-1 — skillgate/gate-v1

- **Stage:** ready to dispatch once PR-3's branch exists — blocked on lift
- **Owner:** senior-implementer
- **Branch:** `skillgate/gate-v1` (not created)
- **Base:** `main` after PR-3 merges
- **Depends on:** PR-3 (clean-corpus test), PR-2 transitively
- **Blocks:** PR-4

## Goal
Ship the safety gate: module, skill wrapper, CI, docs, and the new test
surface. The release's core deliverable.

## Files
- `skillgate/**` (module rename done in tree)
- **`skillgate/difftest/` carved into its own module** — own `go.mod`,
  joined via `go.work`; gate module's `go.mod` returns to zero requires
  (zero-dep constraint, user 2026-09-13). `NOTICE` ships with it.
- **G003 reachability** — `skillgate/refgraph.go` (+ maybe `refgraph_reach.go`):
  BFS from convention-load seeds over resolved refs on **all** inspected
  files (scripts/configs emit edges too — wider than G001/G002's prose
  scope, stated divergence); three-way classification
  (reachable/unreachable/dynamic_only); scope = files under a SKILL.md
  root; emits additive `reachability` block in report/v1
  (`seeds`/`reached`/`unreached`/`dynamic_only`, sorted).
- **Shared "convention loads" seed table** — one list naming SKILL.md +
  harness-convention configs + convention-loaded text, read by G003 and
  referenced by T015–T018's paths. Extraction only; per-rule parsing
  stays put.
- **`--list-checks` / `--only`** flags + parallel native checks
  (errgroup over the immutable ledger; ICM keeps its ordering dependency
  on the token budget).
- **Raw-lane miss fixes** (E2-verified, no new machinery):
  - difftest ports: BH1 document-level misses on hooks.json/settings.json;
    TP1 escaped-comment miss (`<\!--`).
  - live rules: T017 must cover `"type": "http"` hooks (verified miss —
    `exfil.example` webhook → zero findings); a `data:…;base64,` payload
    decoding to injection text gets zero findings — decide: new leg or
    documented cede. If fixed, it's a tripwire leg, not a view.
- `docs/skillgate-spec.md` — the ceded-lane sentence (raw-view limits
  gone for what we port; remaining ceded lanes named).
- **`NOTICE`** — NVIDIA SkillSpector attribution for `difftest/` (added to
  scope 2026-09-13; was in no file list)
- `go.work`; `.github/workflows/ci.yml` (`go-version-file: go.work`, both
  modules, `tests/test_gate.sh` step — in-tree diff)
- `skills/skill-gate/SKILL.md` — install line, keep `go run` fallback +
  degrade path
- `docs/skillgate-intent.md` — "Not in v0.5.0" list; `.scuba/**` line →
  "gitignored, never committed"
- `docs/skillgate-spec.md:131-132` — CLI surface is `gate` + `version`
- `skillgate/icm.go:110` comment fix
- `README.md` — "The safety gate" section + skills-table row + install
  command + `cd skillgate && go test` in Development
- `tests/test_skill.sh:33` — loop adds `skill-gate`

## New tests
- `skillgate/icm_test.go`: fires + no-fire rows for SK-I003/I004/I005/H002
- `skillgate/testdata/malicious-bundle/` fixture (T017 + T020) + gate_test.go
  case asserting REJECT + exit 1
- **G003 fixtures**: unreachable executable → finding; unreachable doc/asset
  → Info; convention-seeded `hooks/hooks.json` not flagged; dir-iterated /
  `$var`-only file → `dynamic_only`, recorded not flagged; file outside any
  skill root → exempt or Info; deterministic sorted `reachability` block.
- **`--list-checks`/`--only`**: `--only tripwire-exfil` runs that check
  alone; `--list-checks` names every registered check; parallel run output
  is byte-identical to sequential (determinism test).
- **Raw-lane regressions**: hooks.json with `"type":"http"` → T017 (or
  named new leg) fires; data-URI base64 fixture → fires or documented cede;
  difftest BH1/TP1 port fixes verified by the differential.
- `tests/test_gate.sh`: build binary; gate own skill → exit 0 CAUTION;
  gate fixture → exit 1 REJECT; `--format sarif` parses; `version` matches
  `gate.go`; bare `PATH` → `checks_skipped[]` names skillspector, agnix,
  skill-validator, preactivation-bash-leg

## Gate
`cd skillgate && go build/vet/test ./...`; `tests/test_gate.sh`; 4 shell
suites; packaging test; one foreground dogfood run on
`https://github.com/anthropics/skills` recorded in PR body (gates only on
"not exit 2").

## Next
Dispatch when PR-3's branch exists. brief-specialist renders the
architecture brief before build starts.
