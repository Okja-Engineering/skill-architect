# otlp-plan.md v2 — plan gate, round 2 (2026-09-13 16:55)

Round-1 roots A–I: closed, except D5 (half) and G3/R-F7 (closed with a false fact). New: 13 REAL, 13 LOW. Decisions inline. v3 closes these; the code gate re-verifies them; no third plan round.

## REAL-1 (high) — skill-name redaction claim is false
Docs and research §3.4: built-in, bundled, user-defined, and official-marketplace skill names appear verbatim on `skill.name`; only third-party plugin skills are replaced with "third-party"; `skill.name` is not gated by OTEL_LOG_TOOL_DETAILS ("custom_skill" is the Skill-tool span placeholder, not a token.usage value).
**Decision:** activation reason becomes "the claude_code adapter does not yet read skill.name (present on token.usage and api_request; third-party plugin skills appear as \"third-party\"); reading it is 0.5.0". Attribution reason: "Claude Code telemetry carries no output-to-skill mapping". Fixture `redacted_skill_name.json` → `skill_name_present.json`: token.usage carrying `skill.name: "my-skill"` verbatim; asserts activation `unknown` with the new reason and tokens `present`. README item 5, spec item 11, and §7 item 6 corrected; OTEL_LOG_TOOL_DETAILS stays in the env list described as "exposes tool parameters in tool_result events", not as a skill-name lever. R-F7 resolves the same way.

## REAL-2/3/4/5 (medium) — the reason/counter model has holes
- Add a counter and reason row for `tool_decision` with absent/unrecognised `decision`: not an entry, counted `undecided`, reason names it; the "all accepts" reason prints seen/accepts/undecided honestly.
- `tool_result` without a readable `success` is not assimilated (no entry), counted `unreadable_outcome`, and named in the reason if it leaves `calls == 0`. An unreadable outcome never serialises as `false`. R2's pin becomes a fixture: `tool_result` missing `success` → no entry, reason names it.
- Gauge row reason: "it carried no sum data points" with no asserted cause.
- Per-record degradation: a `present` result cannot carry a reason (profile/v1). State in the spec: "data points or records that cannot be read are skipped; the profile does not report how many (0.5.0)". Delete the false "surfaces in that signal's reason" sentence. Fixture: one unreadable point among readable ones → `present` with the readable total, and the plan says so.

## REAL-6/7 (low-med) — file-level reasons leak decoder text; the rule contradicts its rows
**Decision:** file-level classification: `error` = "the file could not be read as an OTLP/JSON export" (unreadable, empty, not a JSON object at top level, malformed JSON, or a value that does not fit the OTLP schema); `unknown` = "a well-formed OTLP/JSON export that carries no telemetry". Reasons are typed, never interpolating Go type names: top-level array/bool/number/string → "top-level JSON value is a <kind>; an OTLP export is an object"; syntax error → "malformed JSON at byte N in batch K: <decoder message with Go type names stripped>" plus the mid-write advice only for `.ndjson`-style multi-batch inputs after ≥1 good batch; schema type error → "value at <json path> is not the expected type". Strip a UTF-8 BOM before decoding (nit 8).

## REAL-8 — "accepts always produce a tool_result" is false
Reword: accepts are read from `tool_result`; a captured-mid-run accept has no result yet and is not listed; the reason for `calls == 0` with accepts > 0 says so. No dedup still holds by construction.

## REAL-9 — batch index base
1-based, stated at the loop and in the fixture pin ("batch 4" for 3 good + 1 partial).

## REAL-10/11/12 — §6/§7 mechanics
Fix the recount command (run from `profiler/`: `grep -c '^func Test' *_test.go cmd/*_test.go` prints per-file; sum them or use `go test -list`), fix cross-refs (item 15 for test counts; item 11 for new reasons), and restate counts as functions and subtests separately per commit (no "42 cases").

## REAL-13 — residual allocation
R-F2 → C4 (docs: "verified live" names the local route; remote route "works once on main"). R-F3/R-F4 → C2 (adapter refuses `CaptureOpts.ExportFile` for claude_code with the same loud error as the CLI, or the field is removed from the claude_code path; dead `ExportFile: *exportFile` removed; `cmd/main.go` added to C2's file list). R-F5 → C4 (see REAL-12). R-F6 → C4: AC10 "never present without a value read from the export; a read zero is a value; a single api_request yields total_ms 0 and is present". R-F7 → REAL-1.

## LOW nits to fold
1 multi-resource/multi-scope fixture; 2 second metric with a `type` attribute in the same file; 3 `asInt` as a JSON number; 4 series key uses raw attribute value bytes; 5 int64→int conversion stated (saturate or error, pick and pin); 6 ~~unreadable temporality → treat as cumulative~~ SUPERSEDED 17:05: refused and counted (see mandate.md item-19 decisions); 7 build the binary into the scratchpad or `$(mktemp -d)`, never `profiler/profiler`; 9 correct the "only these files reference them" sentence; 10 insert the README subsection after the skill sentences; 11 list CHANGELOG 0.4.1 "Fixed" bullets as "stay true as history" and add the one-line note; 12 RELEASE_NOTES cite is :24; 13 type-mismatch decode errors are a fifth file-level failure, listed.

## Sound, keep
All 22 RED expectations verified against 527ba4e; leaf coverage complete; accumulator expressive; collector config correct at source (shared `file` component is one writer); C1+C2 pushed together; line arithmetic; rebase command.
