-- What values does the payload field `subagent_type` carry, and what do the
-- numeric fields beside it average out to?
--
-- Read this as "lines whose payload said subagent_type was X", not as "X ran N
-- times and made N tool calls". Both field names and both event names are
-- documentary; the averaged fields are reported under their own names, with no
-- unit attached to `duration_ms` beyond the one the field itself claims.
--
-- The fields are projected in a CTE behind a `json_type` guard; see the note in
-- `tool_call_mix.sql` for the non-object payload that makes the outer form
-- fail.
WITH payload AS (
  SELECT
    event,
    raw->>'$.subagent_type'   AS subagent_type_value,
    raw->>'$.duration_ms'     AS duration_ms_value,
    raw->>'$.tool_call_count' AS tool_call_count_value
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
  subagent_type_value,
  count(*) FILTER (event = 'subagentStart')      AS subagent_start_lines,
  round(avg(duration_ms_value::DOUBLE)
        FILTER (event = 'subagentStop'))         AS avg_duration_ms_field,
  round(avg(tool_call_count_value::DOUBLE)
        FILTER (event = 'subagentStop'))         AS avg_tool_call_count_field
FROM payload
WHERE event IN ('subagentStart', 'subagentStop')
  AND subagent_type_value IS NOT NULL
GROUP BY 1
ORDER BY 2 DESC;
