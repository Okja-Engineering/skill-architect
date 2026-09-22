# Round-2 reconciliation — tree vs research (verified from git + code, 2026-09-12)

Verified from `git status`, `git diff`, `go test` per module, and a live gate run — never from status files. Two new decisions for the maintainer first, then the landed/contradicted/unapplied ledger, then the flagged defaults.

## New decisions (mine to make — flagged, defaults used)

- **D14 — `.scuba/` ignore state.** `HANDOFF.md` and `research-contradictions.md` both describe `.scuba/` as "gitignored, local"; it is **not** — `.gitignore` covers only `tmp/` and `.venv/`, so `.scuba/` showed as 5 untracked entries. The mechanism that existed is `.scuba/.gitignore` with `*` commented out ("restore `*` to re-hide"). **Default used:** added `.scuba/` to root `.gitignore` so the control plane is hidden deterministically for every clone, and restored `*` in `.scuba/.gitignore`. Nothing under `.scuba/` is committed.
- **D15 — PR topology.** The answers doc puts the `claude_code.go` comment, D-6 (`experiment.go` `stopping_rule`) and H3 (`ci.yml`) in PR-0. Two can't hold: `experiment.go` and `claude_code.go` were created/heavily rewritten by the **7 unpushed F03/F04 commits**, so repairs to them cannot sit on `origin/main` without dragging that work in; and `ci.yml`'s `go-version-file: go.work` fails without `go.work`, which belongs to PR-1. **Default used:** PR-0 = `docs/research/**` + repairs to files untouched by the 7 commits (`.out-of-scope.md`, README/CHANGELOG/RELEASE_NOTES, `skill-audit` degrade, `.gitignore`); PR-1 = `skillgate/` + `skills/skill-gate/` + `go.work` + `ci.yml` + `docs/skillgate-*.md`; PR-2 = all remaining profiler work (hook-spool pipeline `ingest`/`doctor`/`hooks`/`analyze` + `queries/`, compare/cursor/types deltas, **carrying** the claude_code.go comment and D-6 repairs and the 7 commits); PR-3 = `skills/skill-rewrite` self-contained vendoring. PR-2/PR-3 are based on local `main` and show the 7 commits until they merge — unavoidable, per `.scuba/roadmap.md`'s own slice map.

## Landed — verified

| Claim | Verified by | Result |
|---|---|---|
| `skillgate/`: G0 quarantine, G1 ledger, G2 SK-T001–T020, G3 SkillSpector shell-out, G7 verdict, baseline, SARIF, `report/v1` | `go test ./...` in `skillgate/` — green; `quarantine.go`, `ledger.go`, `tripwire*.go`, `skillspector.go`, `verdict.go`, `baseline.go`, `sarif.go`, `report.go` | ✅ all present; 28 live rule IDs (20 T + 2 G + 5 I + H002; `"SK-T"` is a prefix constant — the "27" was off by one) |
| Slice 2: agnix shell-out, Pack E per-item attribution | `agnix.go`, `skillvalidator.go`, `budget.go`; live run with all three binaries absent → `checks_skipped[]` names `skillspector`, `agnix` (+ standing `preactivation-bash-leg`) and verdict caps at CAUTION | ✅ degrades exactly as specified |
| Slice 3: SK-G001, SK-G002 Tarjan, SK-I001–I005, SK-H002 | `refgraph.go`, `icm.go`, `frontmatter.go`; `gate_test.go` clean-corpus test runs all three real skills | ✅ |
| `allowed-tools` on our skills; skill-rewrite self-contained | Gate on `skills/skill-rewrite`: **CAUTION, exactly 1 finding (SK-H002 info), coverage 7/7** (was REJECT with 7); no `../` in `skills/skill-rewrite/` | ✅ both answers applied |
| PR-0 repairs | `claude_code.go` comment corrected + `skill.name` capability probed; `experiment.go` rejects `stopping_rule: threshold` with the alpha-spending reason; `ci.yml` reads `go-version-file: go.work` and tests both modules; `.out-of-scope.md` = "we gate, we do not certify"; `skill-audit` degrades instead of `exit 1`; `F04.status.md` marked OBSOLETE | ✅ all in the diff — but `tmp/` is gitignored, so the F04 marker is disk-only, not committable |
| The eight pi items | F14: standing `preactivation-bash-leg` skip caps every verdict at CAUTION (`gate.go:107`); manifest-trap: `packaging_test.go` fails the build if `skills/skill-gate/SKILL.md` isn't discoverable via the plugin manifests; G0: quarantine unit is the package, `provenance.package_manifests` recorded; T017: `pi.extensions`/`extensions`/`bin`/npm-lifecycle legs + `.pi/extensions/` convention leg; T020: `.pi/`, `agent/trust.json`, `.cursor/` targets; Cursor `SKILL.md` conformance owned in `frontmatter.go`/`icm.go` (charset, name==dir, multi-root collision); Pack E: `tool_description` + `tool_input_schema` item classes char-exact in `budget.go`; cache-miss detector: absent by design (`report.go:135` names the refusal); D9: pi static-only, bench deferred past the <1-day bar | ✅ **all eight applied** — the dogfood loop went further than the handoff assumed |

## The research contradicts

- Root `go test ./...` fails — no root module under `go.work` (workspace has `./profiler` + `./skillgate` only). Per-module commands are the contract; CI already runs them per module. **Action:** documented in PR-1's README edit.
- `HANDOFF.md`'s "`.scuba/` (gitignored, local)" — see D14.
- `F04.status.md` cannot be committed (gitignored `tmp/`); the OBSOLETE marker is real on disk but invisible to the repo.
- `rule-language.md` §8's map-order findings are **real at head**: `refgraph.go:263` seeds Tarjan from `for v := range graph` and emits `scc[0]` + a `strings.Join(scc)` evidence in that order; `tripwire_b.go:278–285` ranges `mcpServers` and `env` maps for T015; findings are never sorted before fingerprinting (`gate.go:116–121`). → **E1.**
- SkillSpector go/no-go: **reproduced this session** — `pip install git+https://github.com/NVIDIA/skillspector.git` into a brew `python3.12` venv, `scan --no-llm --format json` on all three skills, exit 0 each. `skill-audit` 43/MEDIUM→CAUTION (7 issues, all AE1), `skill-rewrite` 43/MEDIUM→CAUTION (9 issues, all AE1), `skill-gate` 0/LOW→CAUTION (0 issues). The 16 findings are 100% AE1 reference-resolution noise, matching the recorded claim. JSON committed as `skillgate/testdata/skillspector/*.json`. **Verdict: integrate is a go** — shell-out viable, output parseable, deterministic under `--no-llm`; the FP class confirms baseline-with-`reason:` is load-bearing.

## Flagged defaults in force (I did not decide)

`HANDOFF.md` D1–D9 and `rule-language.md` §9 D10–D13, defaults applied:

- **D1** gate-not-certify (`.out-of-scope.md` language shipped) · **D2** opt-in Python shell-out, never load-bearing (live-verified) · **D3** agnix adopted, Cursor *security* is the delta · **D4** block on tripwire+CRITICAL/HIGH, baselines need `reason:` · **D5** no `C_f`/hour-rate yet · **D6** minutes only, no letter grade · **D7** Cursor static-only · **D8** both breaks in one release · **D9** pi = fixture/bench only, thin-shell package later.
- **D10** (fallback duplication when the engine *is* present): no stated default — **proposed: run-and-dedup** (fail-closed favors coverage); flagging, not deciding. Irrelevant to E2's corpus runs, which measure patterns not the engine.
- **D11** per-activation ceiling: default = **shipping constant** 8,000 tok / 500 lines (`icm.go:21-22`), per "thresholds come from the shipping constant, never the plan doc".
- **D12** page budget: no yardstick named — **proposed: leave the body as-is**; the annex carries the overflow. Flagging.
- **D13** ported-pattern namespace: default = **Option 1** — `SK-P*` in a new Pack P, `Origin` carries the SkillSpector id + Apache-2.0/NVIDIA/`8421a2e`, generating the NOTICE entry. Used in E2.

`rule-language.md` §9 D1–D9 (canonical form, `Weight`, P5 tag, AS3 self-reference, licensing mechanism, in-tree `x/text`, report/v1 fields, ceded-vs-cap, policy profile) all concern rule migration — **deferred by mandate** ("do not migrate the rule representation until E1–E3 are in"); defaults assumed, flagged here so they are on record.
