# Mandate — v0.5.0 "skill-gate" release

_Intake output, 2026-09-13. Source plan: `plan.md` · consistency findings:
`consistency-pass.md` · per-PR detail: `PR-*.status.md`._

## Goal

Ship v0.5.0 of `skill-architect` to `Okja-Engineering/skill-architect`:
the `skillgate` safety gate as a third skill plus an installable Go binary,
and the profiler program already committed locally. Done means a clean
machine can `go install …/skillgate/cmd/skillgate@v0.5.0`, run
`skillgate gate`, and get a report/v1 verdict with a complete ledger —
and the GitHub release `v0.5.0` exists.

## Constraints (real, not convenience)

- **Prototype mode is in force** — no commits, branches, pushes, or PRs
  until the user lifts it (`session-prompt-dogfood.md:7`; user: "not yet",
  2026-09-13). Everything else is staged in the uncommitted tree.
- User merges every PR; agents never push to `main` directly. Never draft
  PRs. Each PR passes `ship-gate` before merge.
- Bash for glue, Go for logic; **no Python for skill scripts** (AGENTS.md).
- Verdicts are `APPROVE`/`CAUTION`/`REJECT` — never "safe"/"clean" (F12).
- No postinstall binary downloads (`skillgate-intent.md:57`) — `go install`
  is the distribution mechanism; goreleaser deferred until a Go-less user
  asks.
- Rule representation frozen pending the rule-language work — no
  rule-representation refactor in this release. Views are the front half
  of that representation (a rule must declare which views it reads), so
  the view layer is 0.6, not 0.5.
- **Zero required deps** — `go.mod` zero requires, self-contained binary,
  zero required external tools (user 2026-09-13). Valuable concepts are
  rewritten in-repo (Go/bash), not depended on; external engines are
  advisory-only forever.
- `NOTICE` must ship with `skillgate/difftest/` (Apache-2.0 attribution).

## Definition of done

1. `GOPROXY=direct go install …/skillgate/cmd/skillgate@v0.5.0` succeeds on a
   clean machine; `skillgate version` prints `0.5.0`. Repeated via default
   proxy.
2. Version surfaces lockstep at 0.5.0: four manifests, `gate.go`,
   three SKILL.md `metadata.version`, `test_skill.sh` pin (+assert gate.go
   matches manifests). `profiler AdapterVersion` — see open questions.
3. Gate behaviour verified: own skill → CAUTION exit 0; malicious fixture →
   REJECT exit 1; bare `PATH` → `checks_skipped[]` names all four expected
   checks; `--format sarif` parses.
4. All five PRs merged in chain order (PR-0; PR-2→PR-3→PR-1→PR-4), each
   green at open, each through ship-gate.
5. `v0.5.0` + `skillgate/v0.5.0` tags on the PR-4 merge commit;
   `gh release create v0.5.0` with the RELEASE_NOTES section.

## Scope

Five PRs per `PR-*.status.md`: PR-0 docs/research + repairs (independent);
PR-2 profiler program (carries 7 local commits); PR-3 skill-rewrite
self-containment (on PR-2); PR-1 skillgate + skill + CI + tests (on merged
PR-3); PR-4 version lockstep + release notes (on merged PR-1).
Consistency-pass amendments already folded into PR scopes: `NOTICE`→PR-1,
manifest descriptions→PR-4, `intent.md` reconcile→PR-0, spec dep-clause→PR-1,
README refresh→PR-2/PR-4. Boundary amendments (2026-09-13) folded into
PR-1: G003 + `reachability` block + seed table; `--list-checks`/`--only` +
parallel checks; difftest module carve; E2 raw-lane fixes; ceded-lane
spec sentence.

## Non-goals

`skillgate env`, `skillgate scan`, `calibration.jsonl`, Pack D tri-state,
slices 4–6, goreleaser cross-builds, ground-truth bench, Cursor dynamic
telemetry, deleting `docs/research/**` (kept permanently — user call).

**0.6 epic** (one epic, `docs/research/rule-language.md` is its spec;
design cards in `problem-brief.md`): all four derived text views,
generated Unicode tables, authored semgrep ruleset + advisory leg, the
eval-artifact model (`BuildModel → *Model`, manifest consolidation into
`Artifact.Manifests` as first slice), the typed-`Rule` migration
(lane-by-lane, difftest fidelity per rule).

## Assumptions

- Remote is and stays `github.com/Okja-Engineering/skill-architect`.
- Go toolchain is an acceptable install prerequisite for the gate's early
  audience (it is the repo's only build dep; CI already installs it).
- `difftest/` ships **carved into its own module** joined via `go.work`
  (user 2026-09-13) — its `x/text` require must not live in the gate
  module's go.mod, or the zero-requires claim is false. Self-skips
  without `SKILLSPECTOR_PYTHON`; not imported by the binary.
- Workers build PR-1/PR-3 branches in parallel; PRs open strictly in chain
  order.

## Decisions ratified

- Module rename → `Okja-Engineering`, both modules (done in tree).
- Version lockstep at 0.5.0 ("one version to explain") — includes
  `profiler/types.go:210` `AdapterVersion` (ratified 2026-09-13; PR-4 bumps
  it and `test_skill.sh` asserts it).
- `docs/research/**` committed in PR-0 and kept permanently.
- `skillgate/difftest/` ships in PR-1 — provenance/evidence trail for the
  SK-P ports; NOTICE ships with it (ratified 2026-09-13). Ships as its own
  module (carve), per the zero-dep constraint.
- **0.5 boundary (ratified 2026-09-13)**: contract + zero-risk wins only.
  In 0.5: G003 reachability + `reachability` report block + shared
  "convention loads" seed table; `--list-checks`/`--only` + parallel
  native checks; difftest module carve; raw-lane miss fixes E2 shows
  (BH1 port misses, TP1 escaped-comment port miss, T017 http-hook leg,
  data-URI base64 leg — all verified live); spec sentence naming ceded
  lanes. In 0.6: views, tables, artifact, semgrep, typed-Rule.

## Open questions

| # | Fork | Recommendation | Owner |
|---|------|----------------|-------|
| Q3 | Lift prototype mode — when? | On your word; all pre-flight done | user |
| Q4 | Binary-asset `binary_unparsed` → REJECT on untrusted vs. `inspected-as-binary` outcome | Keep strict for v0.5.0; revisit if a real skill trips it | user (later) |
| Q5 | T006 placeholder-vocabulary exclusion | Keep firing; baselines absorb FPs | user (later) |

## Readiness

**Ready.** Downstream workers can act on every PR without inventing product
intent: file lists, gates, ordering, and all release-scope decisions are
explicit. Q3 alone blocks execution. Q4–Q5 are standing policy calls, out
of scope for this release.
