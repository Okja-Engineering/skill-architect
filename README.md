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
| `tokens` | a `claude_code.token.usage` sum that declares an `aggregationTemporality` of 1 or 2 — as a bare number, as a quoted digit, or as the protobuf JSON enum name (`AGGREGATION_TEMPORALITY_DELTA` / `AGGREGATION_TEMPORALITY_CUMULATIVE`) — with a data point carrying a `type` attribute of `input`, `output`, `cacheRead` or `cacheCreation` and a value that is a token count — a whole number from 0 to 2⁶³−1, from `asDouble` or `asInt` |
| `tool_calls` | `claude_code.tool_result` events carrying a tool name and a readable `success` value, **or** `claude_code.tool_decision` events recording a reject with a tool name. Either alone is enough |
| `timing` | `claude_code.api_request` events carrying a parseable timestamp |

Anything else is `none`, and `capture` delivers exactly what `probe` advertised, because
both read the export through the same extractor. A partial export carrying tool calls and
timing but no token metric yields those two `present` and `tokens` `unknown` with a
reason, rather than discarding the run.

A file you supplied is never reported as "unconfigured". If it cannot be read as an
OTLP/JSON export — unreadable, empty, not a JSON object at the top level, malformed, or
carrying a value that does not fit the OTLP schema — all three OTel signals come back
`error`, naming the failure. Two of those five sit inside a batch and name it (1-based):
malformed JSON, and a value that does not fit the OTLP schema, which also names the field
path when the decoder reports one — a batch whose own top level is the wrong shape (an
array, say) has no path inside it to name, and that reason names the batch alone. Only
malformed JSON has a single byte to point at, so only it carries an offset: the 0-based
offset of the first byte the decoder could not accept, or the file's length when the file
ended mid-object. If it parses but carries no telemetry, the signals come back
`unknown` instead. What makes a file an OTLP export is a *value* under `resourceMetrics`
or `resourceLogs`, not the key: an empty list is a value, so `{"resourceMetrics":[]}` is
an export of a session that emitted nothing and each signal gets its own "none of mine is
in here" reason. A JSON `null` is not a value — ProtoJSON reads `null` as the field's
default, so `{"resourceMetrics":null}` said nothing about metrics — and JSON carrying a
value for neither field is reported as not being an OTLP export rather than as a broken
one.

A token count the export said nothing about is **absent** from the profile rather than
reported as `0`: a cache-only export gives you `cache_read` and no `input` or `output`
key, because "nothing was said about output tokens" is not "no output tokens". A count
read as zero keeps its key, with `0` in it. No key names changed; the profile schema is
still `skill-architect/profile/v1`.

Counts are merged per **time series**, and a series is what OTel says it is: the resource
the points came from, the instrumentation scope that recorded them, the metric's name, and
the data point's whole attribute set. Two that differ anywhere are two series and never
merge, so a capture aggregating several resources or scopes adds them up rather than
letting one replace another — two resources reporting cumulative 100 and 200 give 300.
Under cumulative temporality `startTimeUnixNano` says which **run** of a counter a point
belongs to: points sharing a start are running totals of each other, so the run holds the
greatest of them, while a point carrying a different start is a counter that restarted,
and both runs count — 100 followed by a restart reaching 20 gives 120. `timeUnixNano` is
not read for this: a running total only goes up, so the values carry their own order, and
a flush reporting less than an earlier one is a capture contradicting itself rather than
tokens given back. A point whose start time is absent, zero or unreadable cannot say which
run it is from, but it came from one — so it is neither a run of its own nor free. It
joins the run it is cheapest to have come from, the largest one, and raises the series
only by what it reports over and above that run: runs of 100 and 20 beside an unplaceable
500 give **520**, since the 500 is a running total of the run that reached 100 and the
other run's 20 is still beside it. A capture carrying no start times at all has no runs to
place its points against, so it holds the greatest running total they reported — which is
what v0.4.1 reported too wherever an exporter wrote its flushes in time order, since
v0.4.1 took the latest by `timeUnixNano`. A session whose last flush happens to omit
`startTimeUnixNano` is reported at its size rather than at twice it. A start time of `0`
is a start time that is absent: OTLP uses the protobuf JSON mapping, in which an explicit
zero and an omitted field are the same message. If one series somehow carries *both* delta
and cumulative points, no total it could contribute is in the export, so that series is
refused rather than resolved one way — the other series in the file still count. When no
series survives, `tokens` is `unknown` and the reason names the refusal; when a healthy
series survives beside it, `tokens` is `present` with a total the refused series is missing
from, and **schema v1 has no field that can say so** — a `present` result carries no
reason. Surfacing that, with the count of skipped data points it belongs beside, is
tracked for 0.5.0. Delta temporality — Claude Code's default — is unaffected throughout:
increments add up whatever series they are on.

`capture` exits **2** when nothing was read and at least one signal came back `error` — a
supplied export that could not be used — and **0** otherwise, including a profile that is
entirely `unknown` because no telemetry was configured. The profile is written to stdout either
way, so a non-zero status still gives you the reasons. Scripts that store capture output
should check the status: exiting 0 after reading nothing is how a broken `--otel-file`
path becomes a row of zeros in a comparison. Every usage error — an unknown command or
harness, a missing required flag, an unrecognised flag, `--export-file` — exits **1**, so
2 means the capture and nothing else.

`skill_activation` and `attribution` are always `none`, for two different reasons. Claude
Code *does* emit skill telemetry: `claude_code.skill_activated` is logged whenever a skill
is invoked, whether Claude calls it through the Skill tool or you run it as a `/` command,
and it carries `skill.name`, `invocation_trigger`, `skill.source` and `skill.kind`. A
`skill.name` attribute rides along on request-scoped signals too — `claude_code.token.usage`,
`claude_code.cost.usage`, `claude_code.api_request`, `claude_code.api_error` and
`claude_code.api_refusal` — marking the skill active for that request; on those, built-in,
bundled, user-defined and official-marketplace skill names appear verbatim and only
third-party plugin skills are replaced with `"third-party"`, while on `skill_activated`
itself user-defined *and* third-party plugin skills read as `"custom_skill"` unless
`OTEL_LOG_TOOL_DETAILS=1`. This adapter reads none of it yet, which is why activation is
`none`: the gap is in the reader, not in the harness. Reading
`claude_code.skill_activated` is 0.5.0. Attribution is the other case — it has no source
at all, because nothing in the telemetry maps an output back to the skill that produced it.

### Capturing an OTel export

The adapter reads OTLP/JSON: one `ExportMetricsServiceRequest` or
`ExportLogsServiceRequest` JSON object per batch, either one per line (NDJSON) or
concatenated. Claude Code has **no file exporter**, so the file has to be written by
something downstream of it. Two routes produce exactly that format.

**Route (a) — a local OTLP receiver.** Nothing to install, and it is the route the
envelopes this adapter's test fixtures are modelled on were observed through:

```bash
export CLAUDE_CODE_ENABLE_TELEMETRY=1
export OTEL_METRICS_EXPORTER=otlp
export OTEL_LOGS_EXPORTER=otlp
export OTEL_EXPORTER_OTLP_PROTOCOL=http/json
export OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4318
export OTEL_METRIC_EXPORT_INTERVAL=2000   # default 60000 — too slow for a short run
export OTEL_LOGS_EXPORT_INTERVAL=1000     # default 5000
claude -p "…"
```

Claude Code POSTs one complete JSON object to `/v1/metrics` and `/v1/logs` per export
interval. Appending each body as a line gives you the file:

```python
# otlp-capture.py — run before `claude -p`; Ctrl-C when the session ends
import http.server

PATH = "otel-export.ndjson"

class Handler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        body = self.rfile.read(int(self.headers.get("content-length", 0)))
        with open(PATH, "ab") as f:
            f.write(body.strip() + b"\n")
        self.send_response(200)
        self.send_header("content-type", "application/json")
        self.end_headers()
        self.wfile.write(b"{}")

    def log_message(self, *args):
        pass

http.server.HTTPServer(("127.0.0.1", 4318), Handler).serve_forever()
```

**Route (b) — an OpenTelemetry collector.** Point Claude Code at the collector with the
same env block, and give both pipelines one shared `file` exporter, so there is one file
and nothing to concatenate:

```yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: 127.0.0.1:4318
exporters:
  file:
    path: ./otel-export.ndjson
    format: json    # one JSON object per line
    append: true    # defaults to false, which truncates your capture on every start
                    # leave compression unset: compressed output is length-prefixed
                    # binary framing, not line JSON
service:
  pipelines:
    metrics: { receivers: [otlp], exporters: [file] }
    logs: { receivers: [otlp], exporters: [file] }
```

This config is derived from the collector `fileexporter` README and source, not from a
run of it here; route (a) is the one that was exercised.

**Not the `console` exporter.** Anthropic's docs suggest it for debugging, and its output
is `console.dir` object inspection — unquoted keys, bare `undefined`, trailing commas.
That is not JSON at all, and it is not the OTLP envelope either. Nothing can parse it.

Two things to know about the resulting file. If the collector or receiver is stopped
mid-write, the last line is a partial object: the profiler reports `error` naming that
batch and offset and uses nothing from the earlier ones, so restart cleanly or delete the
partial last line and retry.

**A capture identifies you.** Every metric data point and every log record carries
`user.email`, `user.id`, `user.account_id`, `user.account_uuid`, `organization.id` and
`session.id`, and that list is what was observed on one version — treat it as the floor,
not the whole of it, and read the file before you share it. `prompt` and `response` are
`<REDACTED>` by default, and the recipe above does nothing to change that.

Leave `OTEL_LOG_TOOL_DETAILS` unset. It is not needed — this adapter reads nothing it
adds — and setting it widens the export to what Anthropic's docs list for it: tool
parameters and input, Bash commands, MCP server and tool names, skill names, and
user-authored workflow names, plus the custom, plugin and MCP command names on
`user_prompt` events that are otherwise collapsed. Those docs suggest it for other
purposes; for capturing a profile it only puts more of your session in a file you may
end up pasting somewhere.

A bundled receiver subcommand is 0.5.0; `--otel-file` is the only input today.

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
| [`skill-validator`](https://github.com/agent-ecosystem/skill-validator) | `brew install agent-ecosystem/tap/skill-validator` | `skill-audit` stops at its structural-checks stage with `skill-validator not found` and exit 1 — that guard is what protects you. `audit-report.sh` also detects it: `spec` is `null`, `spec_error` names the tool, and `summary.passed` is `false`. `check-frontmatter.sh` run on its own detects it too: it exits 3 with `required tool not found: skill-validator` and never reaches the license gate, so a missing validator can neither pass a skill nor be mistaken for a policy failure. |
| [`skillscore`](https://www.npmjs.com/package/skillscore) | `npm install -g skillscore` | `skill-audit` stops at the same stage with `skillscore not found` and exit 1. `check-quality.sh` exits 3. `audit-report.sh` emits `quality: null` with `quality_error`, and `quality_score` / `quality_grade` are `null`. |
| `jq` | `brew install jq` (macOS) · `apt-get install jq` (Debian/Ubuntu) | `audit-report.sh` composes its whole report with jq, so without it there is no report to generate: it exits 3 with `required tool not found: jq` and writes nothing to stdout, rather than dying part-way through. `check-paths.sh --json` and `check-structure.sh --json` require jq unconditionally too: they exit 3 with the same message and a `passed: false` payload carrying a `DEP001` finding, whatever the skill contains. Text mode needs no jq and is unaffected. |

These checks are deliberately loud — a missing tool fails the audit instead of
quietly scoring an unchecked skill as a pass. That holds for `jq` as well: a script
that cannot reach a verdict says which tool is missing and exits 3, rather than
reporting a result it did not compute.

The same rule covers the case where every tool is present but one of them answers
with something the caller cannot interpret. `check-structure.sh` exits 3 with a
`DEP002` finding when `check-paths.sh` returns an unenumerated status, a payload it
cannot read, or a payload that contradicts the status it arrived with; and
`check-frontmatter.sh` does the same on stderr for `skill-validator`. `DEP001` and
`DEP002` are the only rule IDs an exit 3 emits, and
[`skills/skill-audit/SKILL.md`](skills/skill-audit/SKILL.md) registers them beside
the `PL`, `PT` and `PATH` findings — the whole set these scripts emit, compared
against the scripts themselves by `tests/test_f01.sh` so the list cannot fall
behind what they produce.

That rule has to survive composition, so `audit-report.sh` applies it to its own
sources. It reads `check-structure.sh` on stdout alone, where the payload is, so a
`DEP002` payload reaches the report as a finding instead of being lost among the
diagnostics; and when a source produces nothing it can read, it names the source in
`spec_error` or `policy_error` and leaves `summary.passed` false rather than
reporting a skill with nothing wrong with it. `quality_error` is the exception, by
design: quality is a score, not a verdict, so an unread `skillscore` leaves
`quality_score` null and the verdict alone.

The scripts share `skills/skill-audit/scripts/verdict-guard.sh`, which is the one
dependency they cannot announce through the guard itself. Each checks that it loads
before relying on it — and that it loaded *completely*, since a file that stopped
short defines some guards and not others — so a missing, damaged or partial copy
exits 3 with `verdict-guard.sh` named and no payload, rather than aborting with the
1 or 2 that mean a spec, path or policy verdict was actually computed.

The same guard holds the one definition of the payload shape those scripts pass
between themselves, `{"findings": [{level, rule, message}, ...], "passed": bool}`.
A consumer proves the whole shape, elements included, before reading any of it,
and a payload that is not that shape is a source it could not read — never a
source with nothing to report.

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

The three below are **not verified in this release** — nobody ran them against a clean
install, unlike the Claude Code route above. They are what each agent's documentation
describes; if one is wrong, the manual copy under "Local checkout" works everywhere.

| Agent | Command | Status |
|---|---|---|
| Devin | `devin plugins install Okja-Engineering/skill-architect` | not verified; check the exact source form against Devin's own docs, since this release found that `.` and `./` are not interchangeable for Claude Code |
| Codex | Install from the local plugin directory or marketplace entry (see [Codex plugin docs](https://www.codex-docs.com/en/docs/build-plugins)) | not verified |
| Cursor | Copy or symlink the plugin directory to your Cursor plugins folder (see [Cursor plugin docs](https://cursor.com/docs/plugins)) | not verified |

All native plugins use the same namespace:

```text
/skill-architect:skill-audit
/skill-architect:skill-rewrite
```

### Local checkout

```bash
# Devin — not verified in this release
devin plugins install .

# Claude Code (from inside the repo) — verified live against a clean install
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
