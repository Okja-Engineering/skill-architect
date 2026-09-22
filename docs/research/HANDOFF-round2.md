# Devin — cold start, round 2: pick up, reconcile the diffs, run the next experiment

You start with no memory. This repo is `skill-architect`, my MIT plugin for auditing Agent Skills, targeting Claude Code and Cursor. I'm the maintainer; I alone merge to `main`. Everything you need is under `docs/research/`, which is untracked and must be committed in your first PR.

## Read, in this order (about 40 minutes)
1. `docs/research/HANDOFF.md` — the original cold start: intent, the plan in ten bullets, decisions D1–D9.
2. `docs/research/HANDOFF-round2-answers.md` — my answers to the last round's two decisions, the branch rule, the eight pi findings to fold in, and the rule-language summary.
3. `docs/research/rule-language.md` + `rule-language-samples.md` — how rules should be expressed and migrated; the 15-rule sample; §8 migration, §9 decisions D10–D13.
4. `docs/research/recommendation.md` — the plan itself, when a bullet above needs its source.

## Step 1 — reconcile the working tree against the research (report before building)
The tree is on `main`, uncommitted, 7 commits ahead of `origin/main`, nothing pushed. Verify from `git status`, `git diff --stat`, and the code, never from a status file:

| Claimed landed | Verify by |
|---|---|
| `skillgate/`: G0 quarantine, G1 ledger, G2 tripwires SK-T001–T020, G3 SkillSpector shell-out, G7 verdict, baseline, SARIF, `report/v1` | `go test ./...` **inside** `skillgate/` (root `go test ./...` fails: no root module under `go.work` — document the per-module command) |
| Slice 2: agnix shell-out, Pack E budget via skill-validator with per-file attribution | `skillgate/agnix.go`, `skillvalidator.go`; neither binary is on this machine — the shell-outs must degrade to `checks_skipped[]` + CAUTION, prove it |
| Slice 3: SK-G001 dangling refs, SK-G002 Tarjan cycles, SK-I001–I005 ICM, SK-H002 | `refgraph.go`, `icm.go`, `frontmatter.go`; 28 rule IDs, not 27 |
| My two answers applied: `allowed-tools` on our skills; skill-rewrite self-contained | Run the gate on `skills/skill-rewrite`: expect CAUTION with 1 finding, coverage 7/7 (was REJECT with 7) |
| PR-0 repairs | `profiler/claude_code.go` comment, `profiler/experiment.go` rejects `stopping_rule: threshold`, `.github/workflows/ci.yml` uses `go-version-file`, `.out-of-scope.md`, `skills/skill-audit/SKILL.md` degrades, `tmp/teams/architect/F04.status.md` obsolete |

Then produce a short reconciliation: what is landed, what the research contradicts, and which of the eight pi items in the answers file are still unapplied (F14 read-path scoping, manifest-trap acceptance test, G0 package unit, T017/T020 extension, Cursor `SKILL.md` conformance, Pack E's two new item classes, the cache-miss don't-build, D9). **Cut a branch and open PR-0 (docs/research + repairs) and PR-1 (skillgate + skill-gate + go.work) before Step 2.** Small, each green alone. I merge.

## Step 2 — the experiments, in this order, each its own small PR

The research is only satisfied when three things are demonstrated, not argued. Do them in order; each has a definition of done.

**E1 — Determinism (prerequisite, hours).** `rule-language.md` §8 and the judges found map-order iteration in `skillgate/refgraph.go` (Tarjan, ~line 259) and `skillgate/tripwire_b.go` (~lines 207–209). Fix both; add a permanent repeated-run probe test that runs the gate N=50 times over `testdata/` and asserts byte-identical `report/v1` output. Done when the probe is in CI and green.

**E2 — Differential fidelity (satisfies rule-language's failed bar).** No candidate cleared 12/15 on the sample. Build the differential harness the document asks for: for each of the 15 sample rules, run SkillSpector's original Python `re` pattern and our Go RE2 re-expression over the same corpus (SkillSpector's own test fixtures plus our three skills plus a hostile corpus you write) and diff match spans and evidence bytes. Start with OH1 and P5, which were just corrected. Done when the annex's fit column is replaced by measured agreement per rule (matches, misses, extras, evidence-byte drift), and every "whole" is either confirmed or demoted. Use the `SK-P*` namespace for ported patterns per D13's default, `Origin` recording the SkillSpector id, and Apache-2.0 attribution in NOTICE.

**E3 — The before/after proof on our own refactor (satisfies the core goal).** skill-rewrite just went from REJECT (7 findings) to CAUTION (1) by becoming self-contained. That is the experiment the whole program exists for. Produce the four-number impact picture for baseline vs refactored — Risk, Debt, Load, Lift, each stamped billed / measured / estimated, never blended (`rule-language.md` §5):
- Risk and Debt: from the gate's two reports (before = `git stash` or the pre-fix commit; after = HEAD). Debt in minutes only, no letter grade (D6).
- Load: Pack E always-on tokens per item, before and after; characters are fine, label them.
- Lift: the only dynamic number. Use `profiler/experiment.go` with a **three-arm** paired design (`L2-dynamic-report.md` §1: no-skill / old skill / new skill), a fixed task set of ≥12 tasks, fixed model, on Claude Code, with OTel export to a local file exporter so `skill.name` attribution is real (`skill-profiling-state-of-the-art.md` §2–3). First verify whether telemetry export is enabled on this machine; if not, enable it locally per those docs and say so. Report the achieved minimum detectable effect beside any null result. If the runs cannot happen here, produce the design and the exact command lines, mark Lift `unknown` with the reason, and stop — never estimate Lift.
- Then dogfood the audit contract: for each finding that changed, write **problem / background context / proposed solution / expected outcome**, where expected outcome is the rule-and-path re-run predicate, and show the predicate passing after.
Done when a single `docs/research/experiments/E3-skill-rewrite.md` carries all four numbers with their basis, the arm design, the MDE, and the per-finding contract, and can be re-generated by a script in the repo.

**Do not** migrate the rule representation or start the pi bench until E1–E3 are in.

## Decisions that are mine
D1–D9 (`HANDOFF.md`), D10–D13 (`rule-language.md` §9). Use each default, flag it, don't decide. Two new ones from you are welcome; put them at the top of your first report.

## One cheap check first
`skillspector scan --no-llm` in a Python 3.12 venv against `skills/skill-audit`, `skills/skill-rewrite`, `skills/skill-gate`. It has never been reproduced here (no venv, nothing on PATH). Commit the JSON as fixtures; it is the go/no-go for the integrate call and the corpus E2 needs.
