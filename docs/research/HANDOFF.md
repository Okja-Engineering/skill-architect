# Devin — cold start: build the skillgate plugin

## Who you are and what this repo is

You are Devin, the agent who builds this repo. You wrote its earlier program spec and built `profiler/`; you start today with no memory of the research below.

`skill-architect` is my MIT plugin of Agent Skills for auditing Agent Skills — `skills/skill-audit`, `skills/skill-rewrite`, manifests at v0.4.0, a Go `profiler/` module, an uncommitted `skillgate/` module.

I'm the maintainer: I make the calls at the end of this page, and I alone merge to `main`.

## The durable unit

The durable unit is **this plugin**, targeting **Claude Code and Cursor** (`.claude-plugin/plugin.json`, `.cursor-plugin/plugin.json`). The research justifying the plan lives in `docs/research/`, **untracked but not gitignored** — committing `docs/research/**` is part of your first PR, so the repo carries its own rationale.

## My intent

"We need a kick-ass, state-of-the-art skill audit / skill doctor / skill inspector — safety, all the things you'd be mindful of in an enterprise situation. Especially if I'm bringing in skills of my own, or skills I might grab off the internet, I want to run something against them and make sure that as we move faster I don't miss something." Earlier goals stand: understanding my usage and spend; before/after proof a refactor paid in dollars and risk; the next best skill to improve.

Four emphases, non-negotiable:

1. **Safety for third-party skills is first.** Nothing below displaces the gate.
2. **Static analysis must be excellent, not adequate** — per-harness frontmatter validity (valid / invalid / silently ignored), and **project-level always-on token cost** with per-item attribution, distinct from per-activation cost.
3. **ICM / progressive disclosure is the house methodology** (`PRINCIPLES.md`): a first-class static dimension, and the tool built that way.
4. **Dynamic rides only on surfaces the targets expose** — Claude Code OTel `skill.name`, Cursor hooks and Enterprise OTel. Never design against a surface they lack.

## Read in this order

`docs/research/recommendation.md` first and in full, then `docs/research/pi-integration.md`, then the rest by need. `docs/research/README.md` indexes the directory and how each file was produced. Shareable rendered pages (private until shared): the recommendation at https://claude.ai/code/artifact/b7f94b51-c3b9-4e3e-aa49-e8c16876e36c and the audit-engine design at https://claude.ai/code/artifact/26764cb5-d04f-47e8-80d6-086b64a1bcd7.

| File (under `docs/research/`) | What it is |
|---|---|
| `recommendation.md` | The plan: threat model, gate stages G0–G7, architecture, refusals, six slices, D1–D9. Fifteen **[pi]** edits. |
| `pi-integration.md` · `pi.md` | The pi delta log · the pi finding itself. |
| `skill-audit-tool/synthesis.md` · `skillspector.md` | Measurement contract, refusals F1–F11, scoring · SkillSpector re-enumerated: 113 rules, 4 dead, no Cursor coverage. |
| `warp-skill-doctor.md` · `otel-genai-alignment.md` · `open-unknowns.md` | Warp taken apart · `gen_ai.*` naming and the never-emit list · unknowns U-01…20. |
| `skill-profiling-state-of-the-art.md`, `existing-per-skill-cost-tools.md`, `cursor-telemetry-surfaces.md` | Harness constants · no Cursor per-skill cost tool exists · the 21-event Cursor surface. |
| `skill-audit-tool/L1-static-report.md`, `skill-audit-tool/L2-dynamic-report.md`, `skill-audit-tool/L3-prioritization-report.md` · `verdicts/` | Single-lens detail · the verdicts that chose each base. |

## The recommendation in ten bullets

1. One Go binary plus a `skill-gate` skill; one pass: is it safe, what does it cost every session, did the refactor pay (§1).
2. **Integrate, don't reimplement** — shell out to unmodified SkillSpector and agnix, never vendor: Apache-2.0 into MIT means NOTICE files plus a permanent Python sync burden (§4).
3. **The install tax is named:** no PyPI, git-URL install, Python ≥3.12,<3.15, native `yara-python` — opt-in, off, never load-bearing for a REJECT.
4. So **Pack B: 20 Go tripwires, capped at 20**, blocking all five dangerous classes with no Python and no Rust present (§3).
5. **F12 / F13:** never "safe" or "clean" — APPROVE / CAUTION / REJECT beside the ledger; a skipped check caps the verdict at CAUTION, named in `checks_skipped[]` (§5).
6. **[pi] F14:** a gate audits *before* load and never contains what loaded. Claims scope to the **read path**; the `bash` leg is open on every harness and ships in `checks_skipped[]`.
7. **[pi] The manifest trap:** a manifest short-circuits the convention-directory walk, so an extensions-only manifest ships the binary and **silently drops the skill — the interface**. Slice-1 test: the skill is discoverable after install (§6).
8. **[pi] Cursor ships a stock skills system** — eight roots including `~/.claude/skills`, recursive, precedence undocumented; agnix's `CUR-*` cover `.mdc`, not `SKILL.md`, so Cursor `SKILL.md` conformance and collisions are ours.
9. **[pi] pi is the always-on acceptance bench, not a target until D9** — slice 2 checks `skillgate env`'s per-item arithmetic in characters against the system prompt pi hands an extension; Pack E gains injected tool text and schemas, 3.9× the prose.
10. **Not built:** SkillSpector reimplemented, a Cursor conformance linter, a Claude Code pre-activation hook (no surface), a SonarQube plugin, a daemon/OTLP, `gen_ai.*` literals, early letter grades, a cache-miss cost signal (§7). **Slice 1 is gated on nothing.**

## What changed against the existing plan

Verified from git and files on 2026-09-12, not from memory.

| Item | Now | State |
|---|---|---|
| Old program: profiler is the point; S7 OTLP; S8 daemon | The gate **is** the program, profiling is slices 4–5; OTLP and daemon not built | Superseded |
| "No Python, no Semgrep" (`AGENTS.md:6`) | Opt-in Python binary, off, never load-bearing | Changed — my call (D2) |
| `.out-of-scope.md:10` "certify skills safe"; `skills/skill-audit/SKILL.md` hard `exit 1` | Both repaired in the tree: "we gate, we do not certify", and the engine degrades instead of exiting | D1's default, landed |
| D-1/D-2/D-3, R1/R2, D-6, the stale `profiler/claude_code.go` comment, the CI Go pin | Closed in the tree: `profiler/experiment.go` rejects non-`fixed` stopping rules; comment corrected; `.github/workflows/ci.yml:34` reads `go-version-file: go.work` | Done — re-verify by `git diff` |
| `docs/profiler-spec.md` | +35/−11 uncommitted: `cache_write`, `error_type`/`count`/`id`, attribution `category`/`confidence`, a comparison contract requiring `baseline_source`/`candidate_source` on every delta | **You own it** |
| `.scuba/roadmap.md` (gitignored, local) | Has the program and D1–D8 but predates pi — no D9, no F14, no manifest-trap test; its slice-1 link points outside the repo (repoint to `docs/research/recommendation.md`). Its open T013 tension — our skills ship `scripts/` with no `allowed-tools`, so the gate REJECTs them — is real | **You own it** |
| Slice 1 | Built, uncommitted: `skillgate/` (G0–G3+G7, 20 tripwires, agnix and skill-validator shell-outs, SARIF), `skills/skill-gate/SKILL.md`, `go.work`; both modules build and `go test ./...` passes today. On `main`: 16 modified, 17 untracked entries, 7 commits ahead of `origin/main`, nothing pushed | Land it first |
| `tmp/teams/architect/F04.status.md` says "parked" | F04 slices 1–3 are committed (`0a83615`, `236152c`, `e0e2a52`) | Stale — trust git |

## What to do, in order

1. **Read** the documents above, `recommendation.md` first and in full.
2. **Evaluate both the plan and the code against it**, verifying from git and files rather than memory or status files. Report what exists, what is stale, what contradicts.
3. **Update what you own** — `.scuba/roadmap.md` and `docs/profiler-spec.md`. Where you disagree, **flag it to me; don't silently change it**.
4. **Advise me where to go next**: a recommended first PR and its definition of done. I expect it to commit `docs/research/**` plus the landed repairs; argue a better split if you see one.
5. **Then build slice one, the safety gate**, as small independently shippable PRs on a branch. **Never merge to `main`** — I merge. The roadmap's older "all work on `main`, uncommitted" line is superseded; flag it if that bites.

## Decisions that are mine

Don't decide these. Use the default, flag it, move on.

| # | Decision | Default |
|---|---|---|
| D1 | A safety position at all, against `.out-of-scope.md:10`? | "We gate, we do not certify"; ship slice 1 |
| D2 | Opt-in external Python binary, against `AGENTS.md:6`? | Yes — opt-in, off, never load-bearing |
| D3 | agnix as a shipped shell-out, losing Cursor *conformance* as differentiation? | Adopt; re-aim slice 2 at Pack E and the Cursor security port |
| D4 | Block merges or only warn? | Block on tripwire + CRITICAL/HIGH; baselines need a written `reason:` |
| D5 | Who owns `C_f` and the engineer-hour rate? | Slices 1–3 ship without them; SDY waits for slice 6 |
| D6 | Time 20 real fixes per rule? | Minutes only — no letter grade |
| D7 | Cursor access (work machine only) | Static against fixtures now; dynamic Cursor warn-only |
| D8 | `PL*`/`PT*` → `SK-*` plus catalogue-first scope | Both breaks in one release or neither |
| D9 | **[pi]** pi: third target, or only a bench fixture? | Static yes (slices 2–3); package yes but a thin shell — logic in Go, binary from `SKILLGATE_BIN`/PATH, absent ⇒ `checks_skipped[]` + CAUTION, never a `postinstall` download; pi dynamic behind slice 4, except the slice-2 bench |

## One cheap check first

Run `skillspector scan --no-llm` in a Python 3.12 venv against `skills/skill-audit` and `skills/skill-rewrite` — the go/no-go on the integrate call (§8, "Thin evidence"). `.scuba/roadmap.md` records it as passed on 2026-09-12 (exit 0, 43/CAUTION each, 16 findings, all AE1 noise), but nothing here reproduces that: no venv survives and neither `skillspector` nor `agnix` is on PATH (system Python is 3.9.6; `python3.12` is in Homebrew). Re-run it, commit the JSON as a fixture, and tell me which claim you verified.
