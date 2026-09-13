# v1.0 pilot — charter & plan

**Status:** charter ratified 2026-09-13 (user session) · stage: 🟡 spec → 🔵 plan
**Control plane:** `.scuba/teams/v1-pilot/` (gitignored, local-only)
**Depends on:** v0.5.0 release chain (staged, awaiting prototype-mode lift)

## What v1 is

A **single-team pilot**. One AI-native team uses skill-architect as a tool
in their toolbox to audit — statically and dynamically — the skills they
have, in their current state. Not all their questions; three of them:

1. **What is the quality of this skill** — from an opinionated perspective
   (spec, Anthropic best practices, ICM, 60/30/10).
2. **What does it cost to run** — measured tokens where telemetry exists;
   labelled estimates where it doesn't; never a judgment number dressed up
   as a measurement.
3. **How often is it run** — activation counts from profiler capture;
   `unknown` where no telemetry surface exists.

We are not the oracle — we produce **evidence**. A 1,400-line skill is not
automatically bad; length in context matters (bundled scripts, references,
per-run cost). The consumer decides what the evidence means for them.

## The lifecycle — this is the product

```
audit → evidence → digest/evaluate → recommend → consumer decides
```

- **Audit** — static (skillgate + skill-audit checks) and dynamic
  (profiler capture where a telemetry surface exists).
- **Evidence** — checkable artifacts: verdict + coverage ledger, dimension
  scores, token/cost basis labels, activation counts. Every claim
  traceable to a file, rule ID, or measurement.
- **Digest/evaluate** — the human reads the evidence; we help interpret
  (what each dimension means, what would make a 1 a 2).
- **Recommend** — ordered, concrete fixes *with the why* ("saves time",
  "more deterministic", "never triggers today"). Recommendations are
  proposals, not verdicts.
- **Consumer decides** — the fix is theirs. `skill-rewrite` drafts on
  request; nothing is applied without approval. Exploring the fix skill
  is in scope; owning remediation is not.

## Operating commitments (codified)

1. **Human holds the judgment.** Every mutation of an audited artifact
   needs explicit approval. Verdicts are evidence for review, never
   certification. Recommendations always say why.
2. **60/30/10 on ourselves.** Our own skills must demonstrate the ratio
   we preach — deterministic scripts carry mechanical work, SKILL.md
   carries orchestration, judgment stays explicit and small. Skills that
   drift get rewritten, not excused.
3. **Honest measurement.** `present`/`unknown`/`error`; basis labels
   `measured|estimated|inferred|billed`; estimates never render as dollars.
4. **Dogfooding is a release gate.** Every release runs skill-architect's
   own tools on its own skills and records the numbers.

## Gap assessment (2026-09-13 session)

What the pilot needs that the tree doesn't have:

- **Estate mode — the central ask.** "Our skills" is plural. Nothing joins
  per-skill audit + activation counts into a ranked plan. Research exists
  (L3 SDY prioritization, hotspot queue); slices 4–6 parked.
- **Surfaced usage.** `profiler analyze` already counts skill activations
  from the spool / `skill.name`; no skill or report exposes it.
- **Closed loop.** rewrite → re-audit exists; rewrite → re-profile →
  compare is manual. The before/after receipt isn't a single path.
- **Regulated-adoption pack.** Own install docs trip our own SK-T008
  (unpinned `npm install -g`, `brew`, `pip git+https`). Segmented
  environments need pinned/vendored deps, SBOM, offline mode, and a
  privacy statement for the hook spool (redaction exists — undocumented).
- **Dedupe / unify.** `skill-rewrite/scripts/` vendors copies of four
  skill-audit scripts (drift risk). Three frontends — bash skills,
  `profiler`, `skillgate` — overlap verbs on the same artifact type.

## ~1,000h allocation

| # | Workstream | ~h | Note |
|---|---|---|---|
| 1 | Land staged work — prototype lift, PR-0→4, v0.5.0, publish dogfood numbers | 100 | Blocks everything |
| 2 | **Estate mode** — discover SKILL.md set, per-skill gate+audit, join activations, ranked plan | 180 | The missing product |
| 3 | Close the loop — rewrite→re-capture→compare as one documented path | 100 | The receipt |
| 4 | Honest cost model (D5) — per-activation cost, engineer-hour roll-up | 80 | "Worth an engineer" slide |
| 5 | Regulated adoption pack — pinned/vendored installer, SBOM, offline doc, signed release | 120 | Segmented-env unblock |
| 6 | Dedupe + unify entry points; plugin-vs-toolchain positioning call | 80 | |
| 7 | Trigger precision/recall (SP1 spike + should-not-fire stratum) | 150 | Makes "too much automation?" answerable |
| 8 | Buffer — calibration, cross-platform CI, docs polish | 90 | |

## Non-goals for v1

- Owning remediation — the consumer decides; we draft on request.
- Answering what telemetry can't support — `unknown` stays `unknown`.
- Dynamic slices 4–5 beyond the SP1 spike; rule-language refactor;
  catalogue duplication (jscpd).
- Certifying skills safe — we gate, we do not certify.

## Open calls

- Plugin-of-skills vs toolchain positioning (README currently says plugin;
  the tree is two Go binaries + skills).
- Offline-pack scope: vendored binaries vs port-the-checks (skill-validator
  spec legs are re-expressible, same trick as difftest/SkillSpector).
- Whether estate mode lands as a `skillgate` subcommand, a `profiler`
  subcommand, or the thin-skill wrapper decision.
