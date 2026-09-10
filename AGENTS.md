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

## Evidence labels
- Specification — from the Agent Skills spec or official docs
- External evidence — from third-party tools or benchmarks
- Repository fact — from this repository's files or history
- Design decision — a choice made by the project maintainers
- Local hypothesis — an unverified claim that needs testing
