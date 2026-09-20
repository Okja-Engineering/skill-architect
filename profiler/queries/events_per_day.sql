-- How many hook events per day?
SELECT
  (ts AT TIME ZONE 'UTC')::DATE AS day,
  event,
  count(*) AS n
FROM read_json(
  coalesce(getenv('SPOOL_DIR'), getenv('HOME') || '/.cursor-profiler/spool') || '/*.jsonl',
  format = 'newline_delimited',
  union_by_name = true,
  columns = {ts: 'TIMESTAMPTZ', event: 'VARCHAR', schema_version: 'VARCHAR',
             cursor_version: 'VARCHAR', cwd: 'VARCHAR', conversation_id: 'VARCHAR',
             raw: 'JSON'}
)
GROUP BY 1, 2
ORDER BY 1, 3 DESC;
