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

## The profiler (preview)

v0.4.0 adds a harness-agnostic profiler that captures runtime signals (tokens, tool calls, timing) from agent sessions and writes them to a profile JSON labelled with the snapshot id you pass in. It degrades gracefully — unavailable metrics are `unknown` with a reason, never invented.

The profiler uses an adapter-per-harness architecture:

| Adapter | Status | Telemetry surface |
|---|---|---|
| Claude Code | ✅ Slice 1 | OTel export (tokens, tool calls, timing) |
| Cursor | Planned | OTel + SQLite fallback |
| Codex | Planned | OTel logs + hooks |
| Devin | Planned | ATIF export + server API |

```bash
# Build the profiler CLI
cd profiler && go build -o profiler ./cmd/

# Print the adapter version
./profiler version

# Probe what the adapter can capture
./profiler probe --harness claude_code --otel-file ./otel-export.json

# Capture a session profile
./profiler capture --harness claude_code --session abc123 --snapshot sha123 \
  --skill-dir ./skills/my-skill --otel-file ./otel-export.json
```

`version` prints the adapter version that every profile records in
`capability.adapter_version` — `profiler 0.4.1` at this release. It tracks what
the adapter captures, so profiles produced by different adapter versions stay
distinguishable. It is not the plugin version, though the two match here.

Probe detection is per signal, not all-or-nothing, and a signal is reported available
only when the export yields a value the adapter can actually read:

| Capability | Reported `otel` when the export carries |
|---|---|
| `tokens` | a `claude_code.token.usage` metric with a recognised token type and a numeric value |
| `tool_calls` | `claude_code.tool_decision` events carrying a tool name |
| `timing` | `claude_code.api_request` events carrying a parseable timestamp |

Anything else is `none`, and `capture` delivers exactly what `probe` advertised, because
both read the export through the same extractor. A partial export carrying tool calls and
timing but no token metric yields those two `present` and `tokens` `unknown` with a
reason, rather than discarding the run. An export that is missing, unreadable, or
malformed is not "unconfigured": all three OTel signals come back `error` naming the
failure, so a file you supplied but the profiler cannot use says so.

`skill_activation` and `attribution` are always `none`: Claude Code emits no skill
activation event, and this adapter does not read the skill-level attributes it does emit.

`capture` also declares `--export-file`, for adapters that read a non-OTel session export
such as Devin's ATIF. No shipped adapter reads it, so passing it fails rather than
silently ignoring the path you gave:

```text
$ ./profiler capture --harness claude_code … --export-file session.json
--export-file is not read by the claude_code adapter; supply an OTel export with --otel-file
```

The profile JSON is the integration point for future paired comparisons (F04). See [`docs/profiler-spec.md`](docs/profiler-spec.md) for the adapter interface contract.

## Install

### Prerequisites

The skills shell out to three external tools. Install them before running an audit:

| Tool | Install | Without it |
|---|---|---|
| [`skill-validator`](https://github.com/agent-ecosystem/skill-validator) | `brew install agent-ecosystem/tap/skill-validator` | `skill-audit` stops at its structural-checks stage with `skill-validator not found` and exit 1 — that guard is what protects you. `audit-report.sh` also detects it: `spec` is `null`, `spec_error` names the tool, and `summary.passed` is `false`. `check-frontmatter.sh` run on its own does **not** detect it: the missing binary exits 127, which the script does not handle, so it skips spec validation silently. Only the license policy gate still runs — a license-less skill exits 2, a licensed one prints `frontmatter OK` and exits 0 having checked nothing else. Do not bypass the guard. |
| [`skillscore`](https://www.npmjs.com/package/skillscore) | `npm install -g skillscore` | `skill-audit` stops at the same stage with `skillscore not found` and exit 1. `check-quality.sh` exits 3. `audit-report.sh` emits `quality: null` with `quality_error`, and `quality_score` / `quality_grade` are `null`. |
| `jq` | `brew install jq` (macOS) · `apt-get install jq` (Debian/Ubuntu) | `audit-report.sh` and `check-paths.sh --json` die with `jq: command not found` (exit 127). `check-structure.sh --json` is quieter and worse: it exits 0 and returns `{"findings": [], "passed": true}`, dropping every finding. |

The `skill-validator` and `skillscore` checks are deliberately loud — a missing tool
fails the audit instead of quietly scoring an unchecked skill as a pass. There is no
such guard for `jq`, so confirm it is installed yourself.

Building the profiler additionally needs Go, at the version declared in
[`profiler/go.mod`](profiler/go.mod).

### Native plugin (recommended)

Install `skill-architect` as a plugin in your agent. The plugin namespace keeps the skills grouped and avoids collisions with other skills you may have installed.

`claude plugin install` resolves a plugin *name* against the marketplaces you have
configured — it does not take a repo path — so Claude Code installs in two steps. This
repo is its own marketplace (`.claude-plugin/marketplace.json`):

```bash
claude plugin marketplace add Okja-Engineering/skill-architect
claude plugin install skill-architect@skill-architect
```

`claude plugin list` then shows `skill-architect@skill-architect` at version 0.4.1.

| Agent | Command |
|---|---|
| Devin | `devin plugins install Okja-Engineering/skill-architect` |
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
claude plugin marketplace add ./
claude plugin install skill-architect@skill-architect
```

The trailing slash matters: `claude plugin marketplace add .` is rejected as an invalid
source format, `./` is accepted.

### Manual standalone copy

Copy only the skills you want into your agent's skill directory. You own the files and pull updates when you choose.

```bash
cp -R skills/skill-audit ~/.claude/skills/skill-audit
cp -R skills/skill-rewrite ~/.claude/skills/skill-rewrite
```

Copy both, even if you only want `skill-rewrite`. Its `draft-rewrite.sh` resolves the
audit scripts at `../skill-audit` relative to the `skill-rewrite` directory it lives in,
so `skill-audit` must sit beside it in the same skills directory. Without the sibling it
does not fail loudly: it still writes a `REWRITE-DRAFT.md`, but the "Current state"
section contains `No such file or directory` for `check-frontmatter.sh` and
`check-structure.sh` instead of an audit.

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

# Profiler tests (Go)
cd profiler && go test ./...
```

## What this plugin does not do

- It does not automatically rewrite the audited skill.
- It does not run live paired comparisons (with-skill vs without-skill) — the profiler (v0.4.0 preview) captures runtime signals, but the comparison engine is not yet built.
- It does not judge subjective writing quality or correctness of domain advice.

See [`.out-of-scope.md`](.out-of-scope.md) for deliberate boundaries.

## License

MIT.
