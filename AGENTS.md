# Skill Architect — conventions

## Language choice
- **Bash** for scripts and simple automation.
- **Go** when logic is more complex (parsing, validation, multiple modes, structured output).
- Do not introduce Python for skill scripts. It adds runtime dependencies and breaks the self-contained principle.

## Tool selection — prefer existing tools over building
Before building a custom solution, check whether an existing, maintained tool already solves the problem:
- **`skill-validator`** (`agent-ecosystem/skill-validator`, Go) — spec validation, link resolution, token counts, content analysis, contamination detection. Install via Homebrew.
- **`skillscore`** (npm) — 7-dimension Anthropic-aligned quality scoring. Install via npm.
- **`grep`** — absence checks (missing headings, missing code blocks). Semgrep is for pattern finding, not absence detection.
- Custom bash only for filesystem path resolution of script references in code blocks, which no standard tool covers.

## Git
- Do not add Co-Authored-By or "Generated with" trailers to commits.

## Documents
- **Lead with a TL;DR.** State the conclusion first, in a sentence or two. A reader who stops there should still have the answer.
- **Unslop.** Cut throat-clearing, restatement, and stacked hedges. Say an uncertain thing plainly once rather than qualifying it three times.
- **Prefer a table or a short list to a paragraph** whenever the content is structured.
- **Say what a document does not cover**, near the top, when its scope is narrower than its title suggests.
- These rules apply to runbooks, specs, reports, and PR bodies — not just user-facing docs.

## Evidence labels
- Specification — from the Agent Skills spec or official docs
- External evidence — from third-party tools or benchmarks
- Repository fact — from this repository's files or history
- Design decision — a choice made by the project maintainers
- Local hypothesis — an unverified claim that needs testing

## Operating principles
- **Human-in-the-loop is the product contract.** Tools produce evidence and recommendations; a human approves every change to an audited artifact. A verdict is never a certification.
- **60/30/10 applies to our own skills.** Deterministic scripts carry mechanical work; `SKILL.md` carries orchestration; judgment stays explicit and small. Skills that drift get rewritten, not excused.
- **Dogfooding is a release gate.** Run skill-architect's own tools on its own skills before each release and record the numbers.
- **Honest measurement.** Every number carries a basis — measured, estimated, inferred, or billed. Estimates are never rendered as dollars.
