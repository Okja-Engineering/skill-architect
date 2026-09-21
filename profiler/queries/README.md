# queries/ — asking the spool bigger questions than a summary answers

DuckDB SQL over the JSONL spool. No ETL: each file reads `$SPOOL_DIR/*.jsonl`
(default `~/.skill-architect/spool`) directly with `read_json`, so there is one
copy of the data and it is the one the capture layer wrote. `profiler analyze`
answers the same shape of question in pure Go with no dependency; these are for
the questions that outgrow one summary document.

## What these describe, and what they do not

**They describe the files.** A row here is a count of spool lines, of envelope
field values, or of payload keys. None of it is a measurement of what the
harness did: no adapter in this release reads a spool, so nothing here produces
a `Profile`, a token count or a skill activation, and no column is named as if
it did. `payload_bytes.sql` reports bytes because bytes are what is there — it
used to divide them by four and call them estimated context tokens.

**The payload field names are documentary.** No payload in this repository was
produced by a running Cursor. `tool_name`, `model`, `subagent_type` and the rest
come from Cursor's published hooks documentation, so a query over one of them
returning no rows may mean the field is called something else rather than that
nothing happened. **`payload_keys.sql` is the query that settles it**: it lists
the keys the payloads actually carry, and a reader written later should be
written against its output rather than against these guesses.

## Running

```bash
brew install duckdb   # one-time
SPOOL_DIR=~/.skill-architect/spool duckdb -c ".read queries/events_per_day.sql"
# or ad-hoc:
SPOOL_DIR=~/.skill-architect/spool duckdb
```

## Conventions

- **Envelope columns**, written by the capture layer and declared by every
  query: `ts`, `event`, `schema_version`, `cursor_version`, `cwd`,
  `conversation_id`, `strict`, `raw`. The set is not maintained by hand —
  `TestSpoolQueries_ReadTheEnvelopeThatIsWritten` derives it from the
  `SpoolEvent` struct and fails any query that does not name all of it, which is
  how `strict` was found missing from all seven.
- **`strict`** marks a line whose content fields were replaced by their sizes
  before it was written. Its byte counts are not comparable with an unstripped
  line's, so `payload_bytes.sql` separates them rather than summing them.
- **Payload fields live under `raw`** — `raw->>'$.tool_name'`, and
  `json_keys(raw)` for the census.
- The default spool directory is not spelled from memory either:
  `TestSpoolQueries_DefaultToTheSpoolThisProductWrites` derives it from
  `DefaultSpoolDir`, the function that decides where lines are written.
- **A spool with no files is an error, not an empty answer.** DuckDB reports
  `IO Error: No files found that match the pattern …` and names the path. That
  is the wanted behaviour and the same distinction `AnalyzeSpool` makes: zero
  rows for a capture that never ran reads as "nothing happened".
- Payload fields are projected in a CTE behind `WHERE json_type(raw) = 'OBJECT'`
  rather than extracted in the outer `WHERE`. A spool holds whatever was piped
  to `ingest`, including input that was not JSON at all, kept deliberately as a
  JSON string — and a JSON path test against that in an outer `WHERE` is pushed
  into the scan and fails the whole query. Measured on DuckDB 1.5.5.

All eight files were run against a seeded spool containing a non-object payload
and a `--strict` line; all eight return rows.

## The questions

| File | Question |
|---|---|
| `payload_keys.sql` | **Which keys do the payloads carry, and on which events?** The one to run first |
| `events_per_day.sql` | How many spool lines per day, by event name? |
| `conversations_per_day.sql` | How many distinct conversation ids per day? |
| `time_of_day.sql` | At what hour of the UTC day were lines captured? |
| `tool_call_mix.sql` | What values does the payload field `tool_name` carry? |
| `model_mix.sql` | What values does the payload field `model` carry? |
| `subagent_usage.sql` | What values does the payload field `subagent_type` carry? |
| `payload_bytes.sql` | How many payload bytes per conversation id and per day? |
