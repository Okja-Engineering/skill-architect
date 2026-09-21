-- What values does the payload field `tool_name` carry, on the events Cursor's
-- documentation says are the tool-use hooks?
--
-- Read this as "lines whose payload said tool_name was X", not as "X ran N
-- times". The field name and the three event names are documentary: nothing in
-- this repository has seen a payload from a running Cursor. If the field is
-- called something else, this returns nothing — and `payload_keys.sql` is the
-- query that would show it.
--
-- `duration` is reported as the field's own average with no unit attached. The
-- payload's documentation does not say what unit it is in, so calling the
-- column milliseconds would be a claim about a number nobody has read.
--
-- The payload fields are projected in a CTE, behind a `json_type` guard, rather
-- than extracted in the outer WHERE. A spool holds whatever was piped to
-- `ingest`, including input that was not JSON at all — kept deliberately, as a
-- JSON string — and a JSON path test against that in a WHERE clause is pushed
-- into the scan and fails the whole query. Measured on DuckDB 1.5.5, against a
-- spool containing one such line.
WITH payload AS (
  SELECT
    event,
    raw->>'$.tool_name' AS tool_name_value,
    raw->>'$.duration'  AS duration_value
  FROM read_json(
    coalesce(getenv('SPOOL_DIR'), getenv('HOME') || '/.skill-architect/spool') || '/*.jsonl',
    format = 'newline_delimited',
    union_by_name = true,
    columns = {ts: 'TIMESTAMPTZ', event: 'VARCHAR', schema_version: 'VARCHAR',
               cursor_version: 'VARCHAR', cwd: 'VARCHAR', conversation_id: 'VARCHAR',
               strict: 'BOOLEAN', raw: 'JSON'}
  )
  WHERE json_type(raw) = 'OBJECT'
)
SELECT
  tool_name_value,
  count(*) FILTER (event = 'preToolUse')         AS pre_tool_use_lines,
  count(*) FILTER (event = 'postToolUseFailure') AS post_tool_use_failure_lines,
  round(avg(duration_value::DOUBLE)
        FILTER (event = 'postToolUse'))          AS avg_duration_field
FROM payload
WHERE event IN ('preToolUse', 'postToolUse', 'postToolUseFailure')
  AND tool_name_value IS NOT NULL
GROUP BY 1
ORDER BY 2 DESC;
