# v0.5.0 internal consistency pass — 2026-09-13

> *Amendment (same day): the 0.5 boundary decision superseded parts of
> this pass — `difftest/` is carved into its own module (gate go.mod
> returns to zero requires); views/tables/artifact/semgrep/typed-Rule are
> the 0.6 epic; PR-1 additionally owns G003 reachability, `--list-checks`/
> `--only` + parallel checks, and the E2-verified raw-lane fixes. Source of
> truth for scope: `problem-brief.md` + `mandate.md`.*

Read of every shipping surface against the release plan. Verdict: the plan is
sound and the tree is closer than the docs suggest; the inconsistencies are
enumerated below with the PR that owns each fix.

## What v0.5.0 builds

- `skillgate` binary: `gate` + `version` only. `go install
  github.com/Okja-Engineering/skill-architect/skillgate/cmd/skillgate@v0.5.0`.
- `skills/skill-gate` — the third skill wrapping the binary.
- The profiler program already on `main`: 7 commits (F03 adapters, F04
  compare/experiment) + hook-spool subcommands (probe, capture, compare,
  ingest, doctor, hooks, analyze, experiment, version).
- Contract: every file gets a ledger outcome; no verdict above CAUTION with
  a skipped check; absent binaries are named skips, never "safe". CAUTION is
  the reachable ceiling (F14, live-verified today).

## What v0.5.0 does not build

- `skillgate env`, `skillgate scan`, `calibration.jsonl`, Pack D tri-state
  delta, slices 4–6. These appear only in `docs/research/**` (working notes)
  and in the `icm.go:110` comment — the shipped spec/SKILL/README surface is
  already clean of them (grep-verified).
- SK-H001 (per-harness-invalid frontmatter key) — deliberately ceded to
  agnix CUR-005 (`recommendation.md:156`). The catalog gap is a decision,
  not an omission.
- Pre-activation blocking, executing/sandboxing bundles, certifying safety —
  `skillgate-intent.md` "Deliberately not built" + `.out-of-scope.md`.

## Verified consistent

- Verdict engine `verdict.go` matches spec exit semantics (0/1/2); live run
  on `skills/skill-gate`: CAUTION, exit 0, `checks_skipped` =
  [skillspector, agnix, preactivation-bash-leg]; skill-validator ran
  (`o200k_base`, `basis: measured`).
- Rule catalog: spec table and code both carry exactly
  T001–T020 + G001/G002 + I001–I005 + H002 (28 IDs).
- All 4 manifests declare a skills root → `skills/`; `packaging_test.go`
  enforces skill-gate discoverability. No per-skill enumeration to update.
- `skill-rewrite` is self-contained in tree: vendored `scripts/` (5) +
  `references/`, `allowed-tools` on all three skills, zero `../` refs.
- `ci.yml` in tree already reads `go-version-file: go.work` and tests both
  modules (H3 resolved). Only the `test_gate.sh` step is missing.
- `.out-of-scope.md` already carries the gate's "never certify" language.
- `docs/research/README.md` correctly calls itself the canonical copy —
  PR-0 commits it permanently (user decision: keep).

## Gaps found this pass (new — not in the original plan)

1. **`profiler/types.go:210` `AdapterVersion = "0.1.0"`** — a sixth version
   surface. `profiler version` prints it; every profile JSON stamps it.
   Semantically it versions the adapter implementation, not the plugin.
   DECISION NEEDED: lockstep to 0.5.0 with everything else ("one version to
   explain") or keep independent + document. Recommend lockstep — add to PR-4.
2. **`docs/skillgate-intent.md` drift** — :5-6 and :62-69 still say
   `docs/research/**` "may fall away at release" (user kept it); :65 calls
   `.scuba/` "temporarily un-ignored" (now re-hidden); :74 says
   `rule-language.md` must "land" before rule refactors — the file is in
   `docs/research/` already. Owner: PR-0 reconcile.
3. **Plugin manifest `description` fields** — all four say "auditing and
   improving Agent Skills"; no mention of the gate. Marketplace-facing.
   Owner: PR-4 (ride the version bump).
4. **README profiler section is v0.4.0-scoped** — shows only probe/capture;
   v0.5.0 ships 9 commands. Also `README.md:28` "v0.4.0 adds" phrasing.
   Owner: PR-2 (spec) / PR-4 (README refresh if desired — minor).
5. **Spec "no external dependency" wording** — spec:45 describes the rule
   catalog; `difftest` now carries `golang.org/x/text` + `run_upstream.py`
   (dev-only harness, skips without `SKILLSPECTOR_PYTHON`, not imported by
   the gate binary — `go install` unaffected). One clarifying clause in
   spec or intent would preempt the question. Owner: PR-1.
6. **`skills/skill-gate` ships SKILL.md only** — the skill wraps a binary
   the plugin doesn't install. Covered by plan (install line + degrade
   path), but sharpen: after plugin install on a Go-less machine the skill
   must degrade to "install skillgate first", not fail. Owner: PR-1.

## Gaps already owned by the plan

- Versions disagree 5 ways → PR-4 lockstep (now +AdapterVersion pending call).
- README never mentions skillgate; Devin row ✅ vs placeholder → PR-0 + PR-1.
- `docs/profiler-spec.md` omits 5 commands → PR-2.
- `skillgate-spec.md:131-132` CLI omits `version` subcommand → PR-1.
- `icm.go:110` comment cites `skillgate env` (descoped) → PR-1.
- Untested rules I003/I004/I005/H002 + no committed malicious fixture → PR-1.
- `tests/test_skill.sh` skips skill-gate + pins 0.4.0 → PR-1 + PR-4.
- `.venv-skillspector/` unignored; `.scuba/.gitignore` commented → done in
  tree this session.
- `docs/research/**` describes unbuilt subcommands → PR-0 banner.

## Standing policy calls (not release-blocking)

- Binary-asset `binary_unparsed` → REJECT on untrusted — keep strict vs.
  `inspected-as-binary` outcome.
- T006 placeholder-vocabulary exclusion (trades recall).
- T019 `../sibling` monorepo scripts — baseline case.
- rule-language.md's raw-view fidelity gap (normalized views unported) —
  post-release; gate detection currently loses fullwidth/Cf-interleaved/
  entity-encoded spellings. Documented, deliberate.
