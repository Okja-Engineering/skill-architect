-- Which subagent types run, and how much work do they do?
SELECT
  raw->>'$.subagent_type' AS subagent,
  count(*) FILTER (event = 'subagentStart')                    AS runs,
  round(avg((raw->>'$.duration_ms')::DOUBLE)
        FILTER (event = 'subagentStop'))                       AS avg_duration_ms,
  round(avg((raw->>'$.tool_call_count')::DOUBLE)
        FILTER (event = 'subagentStop'))                       AS avg_tool_calls
FROM read_json(
  coalesce(getenv('SPOOL_DIR'), getenv('HOME') || '/.cursor-profiler/spool') || '/*.jsonl',
  format = 'newline_delimited',
  union_by_name = true,
  columns = {ts: 'TIMESTAMPTZ', event: 'VARCHAR', schema_version: 'VARCHAR',
             cursor_version: 'VARCHAR', cwd: 'VARCHAR', conversation_id: 'VARCHAR',
             raw: 'JSON'}
)
WHERE event IN ('subagentStart', 'subagentStop')
GROUP BY 1
ORDER BY 2 DESC;
