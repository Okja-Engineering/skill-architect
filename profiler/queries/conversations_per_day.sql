-- How many distinct conversations per day? Plus sessions via sessionStart.
SELECT
  (ts AT TIME ZONE 'UTC')::DATE AS day,
  count(DISTINCT conversation_id) AS conversations,
  count(*) FILTER (event = 'sessionStart') AS sessions
FROM read_json(
  coalesce(getenv('SPOOL_DIR'), getenv('HOME') || '/.cursor-profiler/spool') || '/*.jsonl',
  format = 'newline_delimited',
  union_by_name = true,
  columns = {ts: 'TIMESTAMPTZ', event: 'VARCHAR', schema_version: 'VARCHAR',
             cursor_version: 'VARCHAR', cwd: 'VARCHAR', conversation_id: 'VARCHAR',
             raw: 'JSON'}
)
GROUP BY 1
ORDER BY 1;
