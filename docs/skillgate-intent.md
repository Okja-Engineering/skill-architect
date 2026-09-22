# skillgate — intent

What this program is and why, in one file. The normative contract is
`docs/skillgate-spec.md`; the research that produced this is
`docs/research/recommendation.md` + `pi-integration.md` (working notes — may
fall away at release). This file and the spec are kept current as we deliver;
if they drift from the code, that's a defect.

## What

One Go binary (`skillgate`) plus a `skill-gate` Agent Skill wrapping it —
a safety-first audit/doctor/inspector for skill packages targeting Claude
Code and Cursor. Three questions, one pass:

1. **Is this safe to install** — deterministic tripwires, coverage ledger,
   quarantine + provenance for remote bundles.
2. **What does it cost every session** — always-on token budget with
   per-item attribution.
3. **Did my refactor pay** — paired comparison (profiler, slices 4–5).

## Safety position

We gate, we do not certify (`.out-of-scope.md`). Verdicts are
`APPROVE`/`CAUTION`/`REJECT` beside a full coverage ledger — never "safe",
never "clean". Today `CAUTION` is the reachable ceiling: the bash read leg
is open on every examined harness and is a standing named skip, never
claimed closed (F14). The gate audits the **package** — a skill dir or a
plugin/package root — never just the `SKILL.md` in isolation.

## Who for

- The engineer pulling a skill off the internet → the gate.
- The author refactoring their own → debt findings + proof.
- The platform lead ranking the estate → economics, slices 4–6.

## Decisions (D1–D9, recommendation §8)

| # | Decision | State |
|---|---|---|
| D1 | Take a safety position: "we gate, we do not certify" | **In force** — `.out-of-scope.md` updated |
| D2 | Opt-in external Python binary (SkillSpector) | **In force** — advisory-only, never load-bearing |
| D3 | Adopt agnix shell-out; Cursor *conformance* via agnix `.mdc` rules, Cursor `SKILL.md` conformance is ours | **In force** — SK-I002/H002 legs |
| D4 | Block merges or warn | **In force** — block on tripwire + high; baselines need `reason:` |
| D5 | Own `C_f` + engineer-hour rate | Deferred to slice 6 (SDY) |
| D6 | Time 20 real fixes/rule | Minutes only, no letter grade (F4) |
| D7 | Cursor access | Static vs fixtures now; dynamic warn-only |
| D8 | `PL*`/`PT*` → `SK-*` rename + catalogue scope | **Done** — `SK-*` shipped in the tree |
| D9 | pi: static compatibility yes (extensions/lifecycle → T017, state roots → T020); dynamic deferred behind slice 4; no pi package without a thin-shell decision | **In force** — bench fixture, not a target |

## Deliberately not built

- A reimplementation or vendored copy of SkillSpector (113 IDs — shell out).
- A Cursor `.mdc`/`mcp.json`/`AGENTS.md` conformance linter (agnix owns it).
- A pre-activation blocking hook — no such surface exists; the bash leg
  stays a named skip.
- A cache-miss / prefix-invalidation detector as per-skill cost signal.
- A postinstall binary download or a TypeScript implementation in the core.
- Session-transcript harvesting, a daemon/OTLP pipeline, a letter grade
  before calibration (F4).
- Executing or sandboxing the target bundle. Ever.

## What falls away before release

Working notes and control plane, not product: `docs/research/**`,
`tmp/teams/**` (gitignored), `.scuba/**` (temporarily un-ignored for gate
spec work — re-hide or archive at release), `docs/research/verdicts/**`.
What stays: this file, `docs/skillgate-spec.md`, `PRINCIPLES.md`,
`.out-of-scope.md`, `docs/profiler-spec.md`, README/CHANGELOG, the skills,
and both Go modules.

## Where it lives

Slices and status: `.scuba/roadmap.md` (resume anchor). Pending research
gate: `docs/research/rule-language.md` lands before any rule-representation
refactor.
