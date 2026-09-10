# Skill Architect

A plugin of Agent Skills for auditing and improving Agent Skills. It codifies the spec checks, best practices, and context-management principles that keep skills from silently failing.

Read the research and rationale at [`PRINCIPLES.md`](PRINCIPLES.md).

## Why

Skills are instructions read by an LLM. That runtime is unreliable. A broken skill usually fails quietly:

- The description is too vague, so the skill never triggers.
- The description mentions a script name that does not exist.
- `SKILL.md` is missing a required heading, so the agent skips a step.
- A bundled script lacks error handling and silently produces the wrong file.
- Everything is inlined into one long markdown file instead of loaded progressively.

Skill Architect catches these problems with deterministic checks first, then reports on the parts that need human judgment.

## The skills

| Skill | Invoke | What it does |
|---|---|---|
| [`skill-audit`](skills/skill-audit/SKILL.md) | `/skill-architect:skill-audit` | Evaluate a skill directory against the Agent Skills spec, Anthropic best practices, and ICM context-management criteria. Reports pass/fail per dimension with concrete fixes. |
| [`skill-rewrite`](skills/skill-rewrite/SKILL.md) | `/skill-architect:skill-rewrite` | Draft a rewritten `SKILL.md` from a `skill-audit` report. Produces a `REWRITE-DRAFT.md` with templates for missing sections; does not apply changes without approval. |

## Install

### Native plugin (recommended)

Install `skill-architect` as a plugin in your agent. The plugin namespace keeps the skills grouped and avoids collisions with other skills you may have installed.

| Agent | Command |
|---|---|
| Devin | `devin plugins install Okja-Engineering/skill-architect` |
| Claude Code | `claude plugins install Okja-Engineering/skill-architect` |
| Codex | Install from the local plugin directory or marketplace entry (see [Codex plugin docs](https://www.codex-docs.com/en/docs/build-plugins)) |
| Cursor | Copy or symlink the plugin directory to your Cursor plugins folder (see [Cursor plugin docs](https://cursor.com/docs/plugins)) |

All native plugins use the same namespace:

```text
/skill-architect:skill-audit
/skill-architect:skill-rewrite
```

### Local checkout

```bash
# Devin
devin plugins install .

# Claude Code (from inside the repo)
claude plugins install .
```

### Manual standalone copy

Copy only the skills you want into your agent's skill directory. You own the files and pull updates when you choose.

```bash
cp -R skills/skill-audit ~/.claude/skills/skill-audit
cp -R skills/skill-rewrite ~/.claude/skills/skill-rewrite
```

The exact path depends on the agent (`~/.claude/skills/`, `.cursor/skills/`, `.codex/skills/`, `.devin/skills/`, etc.).

## Quick example

```text
/skill-architect:skill-audit ~/.claude/skills/my-skill
```

The skill inspects the directory, runs deterministic checks, scores the skill on 10 dimensions, and returns an ordered list of concrete fixes.

To draft a rewrite:

```text
/skill-architect:skill-rewrite ~/.claude/skills/my-skill
```

This produces `~/.claude/skills/my-skill/REWRITE-DRAFT.md` for review before any changes are applied.

## How it works

`skill-audit` runs a staged evaluation:

1. **Orient** — identify the target skill directory.
2. **Inspect** — read `SKILL.md`, `scripts/`, `references/`, `assets/`.
3. **Validate** — run `check-frontmatter.sh`, `check-structure.sh`, and `check-quality.sh`, or produce a unified machine-readable report with `audit-report.sh`.
4. **Score** — evaluate 10 dimensions on a 0–2 scale.
5. **Report** — produce a pass/fail report with ordered fixes.

`skill-rewrite` consumes the audit report and drafts a `REWRITE-DRAFT.md` with section templates and action items. The maintainer reviews and approves before changes are applied.

## Development

Run the tests:

```bash
tests/test_skill.sh
tests/test_walk.sh
tests/test_f01.sh
tests/test_f02.sh
```

## What this plugin does not do

- It does not automatically rewrite the audited skill.
- It does not run live agent evaluations (with-skill vs without-skill) in v0.3.0.
- It does not judge subjective writing quality or correctness of domain advice.

See [`.out-of-scope.md`](.out-of-scope.md) for deliberate boundaries.

## License

MIT.
