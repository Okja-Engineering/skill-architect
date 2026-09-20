# queries/ — the evolving list of questions

DuckDB SQL over the JSONL spool. No ETL: each file reads
`$SPOOL_DIR/*.jsonl` (default `~/.cursor-profiler/spool`) directly with
`read_json_auto`. A `.duckdb` file, if one ever exists, is a derived cache —
the spool is the source of truth.

## Running

```bash
brew install duckdb   # one-time
SPOOL_DIR=~/.cursor-profiler/spool duckdb -c ".read queries/events_per_day.sql"
# or ad-hoc:
SPOOL_DIR=~/.cursor-profiler/spool duckdb
```

## Conventions

- Envelope columns: `ts`, `event`, `schema_version`, `cursor_version`, `cwd`,
  `conversation_id`, `raw` (the full hook payload, post-secret-redaction).
- Payload fields live under `raw` — e.g. `raw->>'$.tool_name'`,
  `raw->>'$.model'`, `raw->>'$.context_tokens'`.
- `estimated_tokens` anywhere in these queries is `length(raw)/4` — a relative
  signal, never billed usage. The only Cursor-reported token number is
  `raw.context_tokens` on `preCompact` events.
- Skill activation is inferred from `beforeReadFile`/`beforeTabFileRead` on a
  `skills/<name>/SKILL.md` path — `confidence: inferred`, see U-02 in
  `docs/research/open-unknowns.md` (cursor-profiler repo).

## Seeded questions

| File | Question |
|---|---|
| `events_per_day.sql` | How many hook events per day? |
| `conversations_per_day.sql` | How many distinct conversations per day? |
| `tool_call_mix.sql` | Which tools run, how often, how do they fail? |
| `model_mix.sql` | Which models do I actually use? |
| `subagent_usage.sql` | Which subagent types run, and how many tool calls do they make? |
| `estimated_tokens.sql` | Estimated context tokens per conversation and per inferred skill |
| `time_of_day.sql` | When during the day do I use it? |
