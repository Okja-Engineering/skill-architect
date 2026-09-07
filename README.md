# Skill Architect

A plugin of Agent Skills for auditing and improving Agent Skills. It codifies the spec checks, best practices, and context-management principles that keep skills from silently failing.

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

### Devin

```bash
devin plugins install <org>/skill-architect
```

### Claude Code / Cursor / Codex

Copy or symlink the plugin directory into the agent's plugin folder, or install from the marketplace when available.

### Manual standalone copy

```bash
cp -R skills/skill-audit ~/.claude/skills/skill-audit
```

The exact path depends on the agent (`~/.claude/skills/`, `.cursor/skills/`, `.codex/skills/`, `.devin/skills/`, etc.).

## Invocation

All native plugins use the same namespace:

```text
/skill-architect:skill-audit
```

## How it works

`skill-audit` runs a staged evaluation:

1. **Orient** — identify the target skill directory.
2. **Inspect** — read `SKILL.md`, `scripts/`, `references/`, `assets/`.
3. **Validate** — run `check-frontmatter.sh` and `check-structure.sh`.
4. **Score** — evaluate 10 dimensions on a 0–2 scale.
5. **Report** — produce a pass/fail report with ordered fixes.

The deterministic layer catches the common mechanical failures; the AI-judgment layer handles trigger quality, scope, and context-management tradeoffs.

## What this plugin does not do

- It does not automatically rewrite the audited skill.
- It does not run live agent evaluations (with-skill vs without-skill) in v0.1.0.
- It does not judge subjective writing quality or correctness of domain advice.

See [`.out-of-scope.md`](.out-of-scope.md) for deliberate boundaries.

## License

MIT.
