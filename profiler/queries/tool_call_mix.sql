-- Which tools run, how often, and how do they fail?
SELECT
  raw->>'$.tool_name' AS tool,
  count(*) FILTER (event = 'preToolUse')          AS calls,
  count(*) FILTER (event = 'postToolUseFailure')  AS failures,
  round(avg((raw->>'$.duration')::DOUBLE)
        FILTER (event = 'postToolUse'))           AS avg_duration_ms
FROM read_json(
  coalesce(getenv('SPOOL_DIR'), getenv('HOME') || '/.cursor-profiler/spool') || '/*.jsonl',
  format = 'newline_delimited',
  union_by_name = true,
  columns = {ts: 'TIMESTAMPTZ', event: 'VARCHAR', schema_version: 'VARCHAR',
             cursor_version: 'VARCHAR', cwd: 'VARCHAR', conversation_id: 'VARCHAR',
             raw: 'JSON'}
)
WHERE event IN ('preToolUse', 'postToolUse', 'postToolUseFailure')
GROUP BY 1
ORDER BY 2 DESC;
