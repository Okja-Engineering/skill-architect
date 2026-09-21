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

## The profiler

A harness-agnostic profiler that captures runtime signals from agent sessions and writes them to a profile JSON labelled with the snapshot id you pass in. It arrived in v0.4.0; v0.5.0 added six subcommands to it — `compare`, `experiment`, `hooks`, `ingest`, `analyze` and `doctor` — so it is no longer the preview earlier editions of this file called it. The Claude Code adapter reads four signals — tokens, tool calls, timing, and, since v0.5.0, skill activation. It degrades gracefully — unavailable metrics are `unknown` with a reason, never invented.

The profiler uses an adapter-per-harness architecture:

| Adapter | Status | Telemetry surface |
|---|---|---|
| Claude Code | ✅ Ships | OTel export (tokens, tool calls, timing, skill activation) |
| Cursor | Planned | Lifecycle hooks. The [spool that captures them](#capturing-cursor-hooks-to-a-spool) ships now and [`analyze`](#reading-a-spool-back) summarises it; the adapter that turns one into a profile does not |
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

# Compare two captured profiles
./profiler compare --baseline ./before.json --candidate ./after.json

# Ask what this machine can and cannot measure
./profiler doctor

# Summarise what a captured hook spool contains
./profiler analyze
```

`version` prints the adapter version that every profile records in
`capability.adapter_version` — `profiler 0.5.0` at this release. It tracks what
the adapter captures, so profiles produced by different adapter versions stay
distinguishable — and `compare` is what acts on that: it refuses a comparison
between two profiles whose adapter versions differ, because the difference
between them may be the reader rather than the skill. It is not the plugin
version: it moves as soon as the adapter changes what a profile contains for
the same input, which is usually before the release that carries it is tagged,
so the two numbers can differ.

Probe detection is per signal, not all-or-nothing, and a signal is reported available
only when the export yields a value the adapter can actually read:

| Capability | Reported `otel` when the export carries |
|---|---|
| `tokens` | a `claude_code.token.usage` sum that declares an `aggregationTemporality` of 1 or 2 — as a bare number, as a quoted digit, or as the protobuf JSON enum name (`AGGREGATION_TEMPORALITY_DELTA` / `AGGREGATION_TEMPORALITY_CUMULATIVE`) — with a data point carrying a `type` attribute of `input`, `output`, `cacheRead` or `cacheCreation` and a value that is a token count — a whole number from 0 to 2⁶³−1, from `asDouble` or `asInt` |
| `tool_calls` | `claude_code.tool_result` events carrying a tool name and a readable `success` value, **or** `claude_code.tool_decision` events recording a reject with a tool name. Either alone is enough |
| `skill_activation` | `claude_code.skill_activated` events carrying a `skill.name`. The **event**, not the `skill.name` attribute that rides along on request-scoped signals — see below |
| `timing` | `claude_code.api_request` events carrying a parseable timestamp |

Anything else is `none`.

`none` is all a capability can say, so it covers two different situations: no telemetry was
configured, and the export you named could not be read. `capture` tells them apart — the
second is an `error` state with a reason, and exit 2 — and `probe` now says so too, on
stderr, naming the same reason `capture` would put in the profile:

```text
$ ./profiler probe --harness claude_code --otel-file ./typo.json
{ "harness": "claude_code", ..., "capabilities": { "tokens": "none", ... } }
probe: failed to read OTel export file: open ./typo.json: no such file or directory
```

stdout is unchanged — the report is the same JSON a caller already parses — and the exit
status is still 0. `probe` has no documented exit contract to extend, so giving it one is
new surface rather than a repair; **v0.5.0 did not add one**, and no release has committed
to adding one. Until one does, a script that has to branch on a bad export should use
`capture`, which does have an exit contract. A `probe` that read a
file and found nothing in it stays silent, because that is an answer about the session
rather than a fault of the run. A partial export carrying tool calls and
timing but no token metric yields those two `present` and `tokens` `unknown` with a
reason, rather than discarding the run.

**`--session` selects what is read, not just what the profile is labelled with.** Both
capture routes documented below append to one file by design, and route (a) listens on the
standard OTLP port, so one export legitimately holds several sessions and whatever else on
the machine was exporting. A record contributes to a profile only when it carries that
`session.id` **and** was not recorded by another product's instrumentation scope. A
session id the export does not contain gives `unknown`, with a reason naming how many
records were passed over and why — never a confident number belonging to somebody
else's run. A record
carrying no `session.id` at all contributes to nothing: a record that does not say which
run it is from cannot be attributed to one. `capture` refuses an empty `--session` for the
same reason — with no identity asserted there is nothing to scope by — and the library's
`Capture` refuses it too, so a Go caller is told rather than handed a profile of the whole
file.

The scope half is an exclusion, not an allowlist. A scope that names **no** library is
read, because an instrumentation scope is optional in OTLP and a receiver or collector in
the path may not carry one through; refusing those would trade a wrong number for no
number on every pipeline that drops it, and the `session.id` test still stands over them.
A scope that positively names another product is not read.

`probe` takes no `--session`, because it is asked what an export can yield without being
told which session: it answers for the export as a whole. So an export holding two
sessions can probe `tokens: otel` while `capture --session <the one that is not in it>`
reports `unknown`. A profile never disagrees with itself, though — the `capability` block
it carries is derived from that capture's own session-scoped read, through the same
extractor that produced its values.

A file you supplied is never reported as "unconfigured". If it cannot be read as an
OTLP/JSON export — unreadable, empty, not a JSON object at the top level, malformed, or
carrying a value that does not fit the OTLP schema — all four OTel signals come back
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
reason. Surfacing that, with the count of skipped data points it belongs beside, needs a
caveat channel schema v1 has not got; **v0.5.0 did not add one** and the schema is still
v1. Delta temporality — Claude Code's default — is unaffected throughout:
increments add up whatever series they are on.

`capture` exits **2** when nothing was read and at least one signal came back `error` — a
supplied export that could not be used — and **0** otherwise, including a profile that is
entirely `unknown` because no telemetry was configured. The profile is written to stdout either
way, so a non-zero status still gives you the reasons. Scripts that store capture output
should check the status: exiting 0 after reading nothing is how a broken `--otel-file`
path becomes a row of zeros in a comparison. Every usage error — an unknown command or
harness, a missing required flag, an unrecognised flag, `--export-file` — exits **1**, so
2 means the capture and nothing else.

`skill_activation` is read from `claude_code.skill_activated`, which Claude Code logs
whenever a skill is invoked — whether Claude calls it through the Skill tool or you run it
as a `/` command — and only then. One event is one activation, so you get one entry per
event carrying a `skill.name`, in timestamp order, with the `invocation_trigger` the event
named when it named one. An event that carries no `skill.name` is not listed: an
activation of no named skill is one you can do nothing with.

**The event is the source, and the attribute is not.** A `skill.name` attribute also rides
along on request-scoped signals — `claude_code.token.usage`, `claude_code.cost.usage`,
`claude_code.api_request`, `claude_code.api_error` and `claude_code.api_refusal` — where it
marks the skill active *for that request*. One skill used across five requests carries it
five times, and those records are stamped with flush and request times rather than with
the time the skill was invoked. Reading them as activations would report a count and a set
of times the harness never recorded, so an export carrying the attribute and no
`skill_activated` event reports `skill_activation: "none"` in the **capability report** and
`skill_activation: {"state": "unknown", "reason": …}` in the **profile**, with the reason
naming the event that was looked for. Those are the two different vocabularies this file
uses throughout: `none`/`otel` is what a capability can say, `unknown`/`present`/`error` is
what a signal can say, and a value supplied but unreadable is an `error` in the second
vocabulary and a `none` in the first.

Redaction differs between the two, which is the other reason not to pool them: on the
request-scoped signals, built-in, bundled, user-defined and official-marketplace skill
names appear verbatim and only third-party plugin skills are replaced with
`"third-party"`, while on `skill_activated` itself user-defined *and* third-party plugin
skills read as `"custom_skill"` unless `OTEL_LOG_TOOL_DETAILS=1`. Names are reported
exactly as the export spelled them — un-redacting one would be inventing a name nobody
recorded.

`attribution` is the signal whose capability is still always `none` and whose profile entry
is therefore always `unknown` with a reason, and for a different reason entirely from the
above: it has no source at all, because nothing in the telemetry maps an output back to
the skill that produced it. Checked against every distinct attribute key on a live export:
nothing does.

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

# Uncomment to have skill_activation carry the skill's real name instead of
# "custom_skill". It also widens the export — see the tradeoff below.
# export OTEL_LOG_TOOL_DETAILS=1

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

**`OTEL_LOG_TOOL_DETAILS` decides whether the profile can name your skill, so this is a
real tradeoff rather than a flag to leave alone.** Since v0.5.0 this adapter reads
`skill.name` off `claude_code.skill_activated`, and Claude Code logs that attribute as the
placeholder `custom_skill` for a user-defined or third-party plugin skill unless the flag is
set. Measured on 2.1.221, two captures identical but for that one variable:

| `OTEL_LOG_TOOL_DETAILS` | wire `skill.name` | profile `skill_activation.value[].skill_name` |
|---|---|---|
| unset | `custom_skill` | `custom_skill` |
| `=1` | the skill's real name | the skill's real name |

The activation count and the invocation times are the same either way; only the identity
differs. **If you are profiling a skill you wrote — which is the case this tool exists
for — set it**, because otherwise every user-authored skill reports as `custom_skill` and
two such profiles cannot be told apart. Names are reported exactly as the export spelled
them; un-redacting one here would invent a name nobody recorded.

What setting it widens, from Anthropic's docs: tool parameters and input, Bash commands,
MCP server and tool names, skill names, and user-authored workflow names, plus the custom,
plugin and MCP command names on `user_prompt` events that are otherwise collapsed. None of
that is read by this adapter — `skill.name` is the one attribute the flag gates that this
adapter does read — so if you would rather not have the rest of it in a file you may end up
pasting somewhere, leave the flag off and accept `custom_skill`. That is the tradeoff, and
it is yours. (Earlier releases of this file said the flag "is not needed" because "this
adapter reads nothing it adds." That was true before v0.5.0 and is false now.)

There is no bundled receiver subcommand — **v0.5.0 did not add one** — so `--otel-file` is
still the only input.

`capture` also declares `--export-file`, for adapters that read a non-OTel session export
such as Devin's ATIF. No shipped adapter reads it, so passing it fails rather than
silently ignoring the path you gave:

```text
$ ./profiler capture --harness claude_code … --export-file session.json
--export-file is not read by the claude_code adapter; supply an OTel export with --otel-file
```

### Comparing two profiles

`compare` reads two stored profiles and reports what changed between them:

```bash
./profiler compare --baseline ./before.json --candidate ./after.json
```

The report is JSON on stdout, schema `skill-architect/comparison/v1`, with one
entry per signal the profile carries. Each entry says whether it was comparable
and carries **both sides' sources** — always, including the signals neither side
read, which say `none`. A delta is never laundered into a single source.

**A comparison is refused rather than approximated.** Two refusals, at two
levels:

- **The pair**, when the two profiles name different `capability.adapter_version`
  values, or when neither names one. The report carries `refusal`, no signal is
  comparable, and no delta is computed at all. This is what stops a profile you
  stored under 0.4.x being subtracted from a fresh one: the reader changed four
  times between them, and those changes would read as the skill's regression.
  The refusal names both versions.
- **One signal**, when the two sides read it from different sources — an `otel`
  count against a `sqlite` one is not a delta. The other signals still compare.

Differences that do *not* stop a comparison are reported as `notes`: a different
`harness`, `snapshot_hash` or `skill_dir`. Comparing two snapshots of a skill is
what the tool is for, so a different snapshot id is said and not refused.

Deltas are only ever over what both sides read. A token count one profile never
read has no key in the delta either, because the difference between a number and
an absence is not a number — the same rule that gives an unread count no key in
the profile. Both profiles' values are carried in full beside the delta, so it
is visible which side was missing it.

`compare` exits **0** when at least one signal was compared and **2** when the
report is complete and nothing in it is comparable — a refused pair, or two
profiles sharing no signal they both read. The report is written to stdout
either way, because when nothing compared the reasons are the whole point.
Every usage error — a missing `--baseline` or `--candidate`, an unrecognised
flag, a path that is not there, a document that is not a profile of this schema
— exits **1**, so 2 means the comparison and nothing else.

`estimated_context_tokens` is compared with a note that rides on the answer
itself: it is a chars/4 estimate over payload bytes, an ordinal signal only, and
never a cost. `skill_activation` is compared as a set of skill names —
`only_in_baseline` and `only_in_candidate` — because a bare count cannot say
which skill stopped firing.

### Running a paired experiment

`compare` is handed two profiles. `experiment` is what produces them: you
declare the two conditions once, and it expands them into runs, executes them,
and compares each pair.

```bash
./profiler experiment design --file ./design.json   # what the design means
./profiler experiment plan   --file ./design.json   # what would be run
./profiler experiment run    --design ./design.json > ./result.json
```

A design is JSON you write by hand:

```json
{
  "schema": "skill-architect/experiment/v1",
  "name": "skill-rewrite-efficiency",
  "task_families": ["refactor-audit", "refactor-rewrite"],
  "repetitions": 3,
  "ordering": "blocked",
  "analysis_method": "difference",
  "stopping_rule": "fixed",
  "output_dir": "./results",
  "baseline": {
    "name": "no-skill",
    "harness": "claude_code",
    "snapshot_hash": "sha-before",
    "skill_dir": "./skills/skill-audit",
    "command": "./capture-baseline.sh $TASK $REP \"$PROFILE\""
  },
  "candidate": {
    "name": "with-skill",
    "harness": "claude_code",
    "snapshot_hash": "sha-after",
    "skill_dir": "./skills/skill-audit",
    "command": "./capture-candidate.sh $TASK $REP \"$PROFILE\""
  }
}
```

Each command is yours: it runs the agent and writes a profile to the path it is
given as `$PROFILE` — `profiler capture … > "$PROFILE"` is the usual body.
`$TASK` and `$REP` are substituted too. **Quote `"$PROFILE"`** if your paths can
contain spaces; the placeholder is replaced before the shell sees it.

**A design may declare only what the runner does.** `ordering` accepts only
`blocked`, `analysis_method` only `difference`, and `stopping_rule` only
`fixed` — early stopping needs alpha-spending guards that do not exist here, and
without them it is peeking. A design asking for anything else is **refused with
the reason**, not accepted and quietly run as something else. The two conditions
must also name the same registered `harness`: two harnesses measure with
different meters, so a difference between them is not a difference in the skill.

`plan` executes nothing — it is the document you read before you spend. `run`
takes either a design or a materialized plan, never both.

What `run` refuses, after the money is spent: a capture command that exits 0 and
writes no profile (the file left at that path would be an earlier run's, and it
would be compared and reported as this run's numbers), and a profile whose
`harness`, `snapshot_hash` or `skill_dir` is not what the step declared.

What it cannot refuse in advance is the pair itself — your two commands may be
two different builds of the profiler. That pair reaches `CompareProfiles` and is
**refused there**, so a run that used a stale baseline reports a refusal and no
delta rather than the reader's changes as the skill's.

`run` exits **0** when every run compared something, **2** when it ran and some
run compared nothing, and **1** for a usage error or a failed step. The result
document is on stdout; the refusals are also on stderr, one line per run.

### Capturing Cursor hooks to a spool

Cursor can run a command on each of its lifecycle events. `profiler ingest` is
that command: it reads one event payload on stdin and appends one line to a
daily file under `~/.skill-architect/spool`. `profiler hooks install` registers
it for every documented event.

```bash
# Register this binary's ingest in ~/.cursor/hooks.json
./profiler hooks install

# …or somewhere else entirely, which is how you try it out first
./profiler hooks install --home /tmp/try-it

# Metadata only: prompts and tool I/O become their sizes
CURSOR_HOOK_PAYLOAD='{"hook_event_name":"sessionStart"}'
echo "$CURSOR_HOOK_PAYLOAD" | ./profiler ingest --strict

# Take it back out. Only our entries; yours are left alone
./profiler hooks uninstall
```

**This is capture only, and no profile is produced from a spool.** There is no
Cursor adapter: `profiler capture --harness cursor` does not exist. What can be
read back out is what the files contain, not a measurement of the session —
[`analyze`](#reading-a-spool-back) counts the lines, the envelope fields and the
payload keys. The sentence saying a spool yields no measurement is in
[`doctor`](#what-can-this-machine-measure)'s output, under
`observed.spool.yields`, and not in `analyze`'s — `analyze` has sixteen fields and
none of them is that sentence, so if you are pasting a summary somewhere the
caveat does not travel with it. What the spool does is put on disk what it was
handed, minus a credential, so a reader written later works from files that
already exist.

**None of it has been checked against a running Cursor.** No Cursor install was
reachable here, so **we cannot tell you it works.** What has been checked is
Cursor's own published hooks reference, field for field, which rules out the
documentation having drifted since this code was written but not Cursor
behaving differently from its docs:

- **Confirmed.** `~/.cursor/hooks.json` is the User-scope location — there are
  Enterprise, Team and Project scopes above it, and this tool writes none of
  them. The 21 event names this registers are exactly the documented set, with
  no twenty-second. And the top-level `"version"` the reference marks required
  is written on install.
- **Corrected.** `cwd` is **not** a field every payload carries: the reference
  puts it on `preToolUse`, `postToolUse` and `beforeShellExecution`, and the
  field carried on all of them for workspace location is `workspace_roots`,
  which this build does not promote. So `cwd` is promoted **when present** and
  is blank otherwise, which is the right behaviour and not a sign anything went
  wrong. `raw` keeps whatever arrived either way.
- **Still documentary.** That a hook is invoked with one JSON document on stdin,
  and every payload shape for the tool-call events: `preToolUse` and
  `postToolUse` were registered on a live hook emitter here and never fired.

If the documentary half is wrong, `hooks install` writes a file Cursor ignores
and the spool stays empty, or lines arrive with blank envelope fields. What we
can tell you is what it does with what it is given, and that is tested — plus
one live exercise of the whole path, `ingest` → spool → `analyze` → `doctor`,
driven by a real hook emitter that was not Cursor:

- **One line per invocation**, appended, never replacing. Directory `0700` and
  file `0600`, because a line can carry prompt text and paths.
- **Nothing is dropped.** An event name this build has never heard of is kept
  verbatim, a field nobody reads is carried through, and a payload that is not
  JSON at all is stored as a string holding the bytes as received. Numbers keep
  their exact value, including integers too large for a float. What is *not*
  preserved is the order of an object's keys and a duplicate key — neither
  changes any field's value.
- **Credentials are removed**, and that is the one deliberate loss. Fields named
  like a credential go wholesale; a bearer token or an `sk-` key inside a string
  is replaced in place, so the command around it survives. Measurement names
  like `context_tokens` are not credentials and stay.
- **`hooks install` is idempotent and additive.** Running it twice leaves one
  entry and rewrites nothing. Hooks you or another tool registered survive, so
  do events we do not register and top-level fields we have never heard of. The
  file is backed up before any write — install *and* uninstall — and one this
  build cannot parse is refused with its path named rather than replaced. The
  backup suffix is second-resolution, so an install and an uninstall inside one
  second leave a single backup holding the post-install state rather than the
  file you started with.
- **`uninstall` does not leave a virgin machine as it found it.** On a home with
  no `.cursor` directory, install-then-uninstall leaves the directory, a
  `hooks.json` containing `{"hooks": {}, "version": 1}` — schema-valid, so it
  will not break hooks you add by hand later — and the timestamped backup, which
  still carries our absolute binary path in all 21 entries. Delete all three if
  you want the machine back.
- **`ingest` prints nothing on success**, because a capture tool that echoes the
  payload back is one you can see in the thing it is capturing. It exits 1 and
  says why if the spool cannot be written.

The registered command is this binary's absolute path plus
`ingest --spool-dir <home>/.skill-architect/spool || true`: absolute because
`PATH` inside a hook's environment is not ours to assume, `--spool-dir` because
`$HOME` inside a hook's environment is not ours to assume either — that is what
makes `--home /tmp/try-it` above relocate the *capture* and not only the
registration, so the `analyze --spool-dir /tmp/try-it/…` below reads the lines
back — and `|| true` so a failure of ours cannot take your Cursor session down.

**`--strict` is not registered, and there is no environment variable for it.**
If you want the metadata-only mode on a registered hook rather than on a
hand-piped `ingest`, write the command yourself:

```bash
./profiler hooks install --home /tmp/try-it \
  --command '/absolute/path/to/profiler ingest --strict --spool-dir /tmp/try-it/.skill-architect/spool || true'
```

And know what `--strict` does not do: it replaces content fields with their
sizes, and it does not remove identity. Cursor documents `user_email` on every
hook payload; it is not a content field and not a credential, so it reaches disk
in both modes.

### Reading a spool back

```bash
# What is in the spool? Counts, envelope fields, and a census of payload keys
./profiler analyze

# …or somewhere else
./profiler analyze --spool-dir /tmp/try-it/.skill-architect/spool
```

**It tells you what is in the files. It does not tell you what Cursor did.**
Nothing here has seen a payload from a running Cursor, so a summary that turned
`tool_name` into a tool call would be asserting Cursor's documentation is right
— and if the field is called something else, you would be told a session made no
tool calls rather than that we could not find the field. So you get: how many
files and lines, how many lines we could not read, the envelope's own fields by
day and by event, payload bytes, and **a census of the top-level keys the
payloads actually carry**, with the values of the keys we are allowed to quote.

That census is the useful part. It is what an adapter should be written against
once somebody has run Cursor, and it is right whatever the field names turn out
to be.

Two things it will not quote. **Content fields** — prompts, tool I/O, the set
`--strict` replaces with sizes — are counted and never printed, because a
summary is something you paste into an issue. **Numbers** are counted and never
tallied, because a histogram of every number in a spool is a table of
measurements nobody made. `payload_bytes` is bytes; it is not tokens.

A spool directory that is not there exits **1** and says so, rather than
printing an empty summary: "nothing was captured" and "there is nothing to read"
are different answers.

For bigger questions there is [`profiler/queries/`](profiler/queries/) — DuckDB
SQL over the same files, no ETL. Start with `payload_keys.sql`. These need
`duckdb` on your `PATH`, which is **not** one of the ten tools under
[Prerequisites](#prerequisites) — those are the skills' preconditions, and
nothing in the plugin shells out to `duckdb`. No test and no CI step runs these
queries either; all eight were run by hand against a real spool on DuckDB
v1.5.5 and all eight returned rows.

### What can this machine measure?

```bash
# What is set up here, and what it yields
./profiler doctor

# …and whether an export we can read produces anything
./profiler doctor --harness claude_code --otel-file ./otel-export.json
```

`doctor` is the place this tool is most likely to overstate, because making
claims about capability is its whole job. So its report has two halves that
cannot be confused for each other:

- **`measurement`** — what a harness *read*. The tier is `none` unless you give
  `doctor` an export and a harness to read it with, and it reports the probe's
  own signals so you can see what the tier was derived from. A file existing, a
  hook being registered or an environment variable being set never raises it,
  because none of those is a measurement.
- **`observed`** — what is on the machine: your `hooks.json` and which events
  carry our entry, and the spool with its file, line and byte counts.

**And the thing it will not tell you: that this machine can capture anything
from Cursor.** No adapter in this release reads a spool, so there is no hook
tier — and every spool count ships with the sentence saying so, in the document
rather than in this README, because a report gets pasted somewhere else and a
caveat does not travel with it. A spool that exists and is filling up is worth
knowing about, and it is a capture waiting for a reader, not a measurement.

`doctor` exits **0** whatever it finds: "nothing here measures anything" is the
answer for most machines today, and a status that called that a failure would
make the command useless in the one situation it exists for. It writes nothing,
anywhere.

See [`docs/profiler-spec.md`](docs/profiler-spec.md) for the adapter interface
contract, the comparison contract, the experiment contract, the spool contract
and the tier rules.

## Install

### Prerequisites

The skills shell out to ten external tools, and every one of them is a hard
precondition rather than a nice-to-have: a script that cannot get an answer from
one of them exits 3 and says which, instead of reporting a verdict it did not
compute.

Required tools: `awk`, `cat`, `date`, `dirname`, `grep`, `jq`, `mktemp`, `skill-validator`, `skillscore`, `wc`.

That list is compared against the scripts themselves by `tests/test_f01.sh`, in
both directions, so it cannot fall behind what they actually require. It is
compared against two things, because the scripts say "required" in two ways:
most tools through `require_tool`, and `date` and `dirname` through a refusal
written where they are used. The suite drives each of those two broken and
checks the script refuses, names it, and exits inside its own stated set — the
sentence above is the criterion, and it is measured rather than asserted.

Three of them you install. Seven are POSIX utilities you already have, and they
are named here anyway, because "you already have it" is not what these scripts
require of them — see below.

| Tool | Install | Without it |
|---|---|---|
| [`skill-validator`](https://github.com/agent-ecosystem/skill-validator) | `brew install agent-ecosystem/tap/skill-validator` | `skill-audit` stops at its structural-checks stage with `skill-validator not found` and exit 1 — that guard is what protects you. `audit-report.sh` also detects it: `spec` is `null`, `spec_error` names the tool, and `summary.passed` is `false`. `check-frontmatter.sh` run on its own detects it too: it exits 3 with `required tool not found: skill-validator` and never reaches the license gate, so a missing validator can neither pass a skill nor be mistaken for a policy failure. |
| [`skillscore`](https://www.npmjs.com/package/skillscore) | `npm install -g skillscore` | `skill-audit` stops at the same stage with `skillscore not found` and exit 1. `check-quality.sh` exits 3. `audit-report.sh` emits `quality: null` with `quality_error`, and `quality_score` / `quality_grade` are `null`. |
| `jq` | `brew install jq` (macOS) · `apt-get install jq` (Debian/Ubuntu) | **All five `skill-audit` scripts require jq**, not only the two `--json` modes. `audit-report.sh` composes its whole report with it, so without it there is no report to generate: it exits 3 with `required tool not found: jq` and writes nothing to stdout, rather than dying part-way through. `check-paths.sh --json` and `check-structure.sh --json` exit 3 with the same message and a `passed: false` payload carrying a `DEP001` finding, whatever the skill contains. `check-frontmatter.sh` and `check-quality.sh` need it too, and both are text-mode-only — `check-frontmatter.sh` has no `--json` mode at all. They read what `skill-validator` and `skillscore` answer, and proving a payload readable is done with jq, so a missing jq stops them at exit 3 on stderr with no rule ID. |

`awk`, `cat`, `grep`, `mktemp` and `wc` are required in the same sense and by
the same mechanism. Being on `PATH` is not enough: each is asked a question with
a known answer before it is computed with, **and its answer is read again at
every call**, so a tool that is present and broken — a `jq` that runs and prints
nothing, a `grep` that answers with an error status, a `wc` that exits 127, a
`jq` that answers the first question and fails the next — is reported as an
execution error rather than read as a verdict about your skill. A script that
cannot get a straight answer from one of them exits 3 and names it, and the
tool it names is the one that failed.

`date` and `dirname` are required in the same sense and by a different
mechanism, and the difference is worth a sentence because it is the reason they
were missing from this list for a while. `dirname` runs in the first line of
every script, before the guard that announces a missing tool has been loaded,
so it cannot be announced by it; `date` has no constant answer to be asked for,
since the whole point of asking the time is that the script does not know it,
and `-r 0` on BSD is not `-d @0` on GNU. Each carries its own refusal at the
point it is used instead: a `dirname` that gives no answer stops every script
with `cannot resolve this script's own directory` and exit 3, and a `date` that
exits nonzero stops `audit-report.sh` with `date could not answer what time it
is` and exit 3 rather than a report with a blank timestamp.

`sed` and `tr` were on this list and are not, because nothing shells out to
them any more. Each did one thing the shell does itself — prefixing lines, and
stripping the spaces `wc` pads its count with — and each was a tool whose status
nothing read. A dependency that buys nothing is cheaper to delete than to
guard.

`check-frontmatter.sh` and `check-quality.sh` now have hard tool requirements
they did not have before, `jq` among them. That is a deliberate
widening, stated in each script's header: reading what a source answered needs
the predicate that proves a payload readable, and that predicate answers with
jq. The alternative was inferring a verdict from an exit status alone, which is
what let a `skill-validator` exiting 0 while printing non-JSON report
`frontmatter OK`.

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

`claude plugin list` then shows `skill-architect@skill-architect` at version 0.5.0.

Every row below carries the status of the route as **executed in 0.4.3**, the release that
ran them, not as described by a vendor's documentation. 0.5.0 did not re-run them; it changed
no install or update route. Routes marked verified were run against a clean install on that
release's tree. Where a route says "not verified" the row says why — nobody
here could run it, or running it would have changed something that is not ours to change —
and it is a pointer to the vendor's own docs rather than a claim of ours; if it turns out
wrong, the manual copy under "Manual standalone copy" works everywhere.

| Agent | Command | Status |
|---|---|---|
| Devin | `devin plugins install Okja-Engineering/skill-architect` | not verified — **deliberately**. This form syncs to Devin Cloud, so running it would rewrite the plugin list of whichever account is logged in on the machine doing the verifying. That is not ours to change, so it was not run. The local-path form under "Local checkout" *was* run and is verified |
| Codex | `codex plugin marketplace add Okja-Engineering/skill-architect` then `codex plugin add skill-architect@skill-architect` (see [Codex plugin docs](https://learn.chatgpt.com/docs/plugins)) | verified — both steps run against `codex-cli 0.153.4` with `CODEX_HOME` pointed at a scratch directory. `codex plugin list` then reports the plugin installed and enabled. Codex reads this repo's `.claude-plugin/marketplace.json` |
| Cursor | Copy or symlink **the repository root** — the directory holding `.cursor-plugin/plugin.json`, not `.cursor-plugin/` itself — into your Cursor plugins folder (see [Cursor plugin docs](https://cursor.com/docs/plugins)) | not verified — no Cursor install was reachable here, so the destination folder is unconfirmed and comes from Cursor's docs, not from us. Which directory to copy *is* confirmed, from this repository's own layout |

There is no `codex plugin install`. `codex plugin --help` lists `add`, `list`, `marketplace`
and `remove`, and installing takes the same two steps as Claude Code: register a
marketplace, then add the plugin from it.

All native plugins use the same namespace:

```text
/skill-architect:skill-audit
/skill-architect:skill-rewrite
```

### Local checkout

```bash
# Devin (from inside the repo) — verified live against Devin 3000.6.14
devin plugins install --local .

# Claude Code (from inside the repo) — verified live against a clean install
claude plugin marketplace add ./
claude plugin install skill-architect@skill-architect

# Codex (from inside the repo) — verified live against codex-cli 0.153.4
codex plugin marketplace add .
codex plugin add skill-architect@skill-architect
```

`--local` is not optional for a local path. Without it Devin refuses the install outright,
because a local path is not a source it can sync to Devin Cloud:

```text
$ devin plugins install .
Error: local path sources can't sync to Devin Cloud; run `devin plugins install --local .` to install on this machine only.
```

Devin rejects `.` and `./` identically, naming the same flag for each, so the
trailing-slash trap below is specific to Claude Code and not a Devin concern.

The trailing slash matters for Claude Code: `claude plugin marketplace add .` is rejected as
an invalid source format, `./` is accepted. Codex accepts both spellings and resolves them
to the same root, so the trap is Claude Code's alone — checked, not assumed, for all three
CLIs that take a local path.

### Manual standalone copy

Copy the skills into your agent's skill directory. You own the files and pull updates when
you choose: this is also the update command, so it replaces the installed skill rather than
copying into it.

Run it from the root of a checkout of this repository — `skills/` is resolved from the
working directory. Set `skills_dir` to your agent's skills directory, taken from the list
below, and change nothing else:

```bash
skills_dir=~/.claude/skills

mkdir -p "$skills_dir"
for skill in skill-audit skill-rewrite; do
  rm -rf "$skills_dir/.$skill.new" &&
    cp -R "skills/$skill" "$skills_dir/.$skill.new" &&
    rm -rf "$skills_dir/$skill" &&
    mv "$skills_dir/.$skill.new" "$skills_dir/$skill" ||
    { echo "$skill was not updated; your installed copy is untouched" >&2; break; }
done
```

Four things about its shape, because a shorter version of it was wrong twice:

- **It replaces rather than copies into.** `cp -R src dst` copies *into* `dst` once `dst`
  exists, so a bare `cp -R` on the second run leaves a second copy of the skill nested
  inside the first, and any agent that walks the skills directory recursively registers
  the skill twice. Replacing the directory also drops files that were removed upstream,
  which a copy over the top leaves behind forever.
- **It does not remove your installed skill until the replacement is ready.** The copy is
  staged beside the destination and moved into place with `mv`, a rename within one
  directory. The obvious spelling, `rm -rf dst && cp -R src dst`, guards the copy against
  a failed remove but leaves nothing guarding the install against a failed copy: run it
  from the wrong directory over a working install and you are left with no skill at all.
- **It clears its own staging directory first**, so an interrupted run leaves nothing for
  the next one to copy *into*. If a run fails, fix what it reported and run it again; it
  converges.
- **`skills_dir` is the only thing to change**, and it holds a skills *directory* — the
  same thing the list below gives you. Every step appends the skill name itself, so no
  substitution you make can turn this into `rm -rf` on the directory holding your other
  skills. Those are left alone: each pass names exactly the one skill directory it
  replaces.

The loop copies both, and both is not optional even if you only want `skill-rewrite`. Its
`draft-rewrite.sh` resolves the audit scripts at `../skill-audit` relative to the
`skill-rewrite` directory it lives in,
so `skill-audit` must sit beside it in the same skills directory. Without the sibling it
does not fail loudly: it still writes a `REWRITE-DRAFT.md`, but the "Current state"
section contains `No such file or directory` for `check-frontmatter.sh` and
`check-structure.sh` instead of an audit.

The exact path depends on the agent. Each row says where the path came from, because a
path read off a vendor's website and a path a CLI printed here are not the same kind of
claim:

- **Claude Code** — `~/.claude/skills/`, the value `skills_dir` takes in the command above.
  Verified.
- **Devin** — `.devin/skills/` for a project, `~/.config/devin/skills/` globally. Verified by
  running `devin skills paths`, which prints both. Note it is *not* `~/.devin/skills/`.
- **Codex** — `~/.codex/skills/<skill-name>`, or `$CODEX_HOME/skills/<skill-name>` if you
  have moved your Codex home. That is where OpenAI's own `skill-installer` skill — shipped
  *inside* the Codex CLI, at `~/.codex/skills/.system/skill-installer/` — says it installs
  to, and where the skills bundled with Codex sit. Codex also discovers skills from
  `.agents/skills/` in each directory from your working directory up to the repository
  root, from `~/.agents/skills/`, and from `/etc/codex/skills/`, per its
  [skills docs](https://learn.chatgpt.com/docs/build-skills). Both are real; they are
  different locations, not competing accounts of one. Read off those two vendor artifacts,
  not from a placement run here: for Codex the route that was actually executed, in 0.4.3,
  is the plugin one above.
- **Cursor** — `.cursor/skills/` or `.agents/skills/` in a repository, `~/.cursor/skills/`
  or `~/.agents/skills/` globally, per Cursor's [skills docs](https://cursor.com/docs/skills)
  — which is a different page from the plugins docs cited above. **Not verified here**: no
  Cursor install was reachable, so this is Cursor's claim rather than ours.

A skill placed somewhere the agent does not read fails silently — it simply never appears —
so where the path is the vendor's claim and not ours, confirm it with the agent.

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
# First: the suite for the substrate the others are written on
tests/test_harness.sh

tests/test_skill.sh
tests/test_install.sh
tests/test_walk.sh
tests/test_rewrite.sh
tests/test_f01.sh
tests/test_f02.sh

# Profiler tests (Go)
cd profiler && go test ./...
```

`tests/test_harness.sh` holds this list to the suites CI runs, so a suite missing from
here fails it rather than going quietly unrun.

## What this plugin does not do

- It does not automatically rewrite the audited skill.
- It does not drive the agent itself. `profiler experiment` schedules a paired run (with-skill vs without-skill), executes it and compares each pair, but the command that runs the agent and writes each profile is yours to supply — the plugin never launches a model or spends tokens of its own.
- It does not judge subjective writing quality or correctness of domain advice.

See [`.out-of-scope.md`](.out-of-scope.md) for deliberate boundaries.

## License

MIT.
