-- When during the day do I use it? (UTC hour — spool ts is UTC)
SELECT
  hour(ts AT TIME ZONE 'UTC') AS hour_utc,
  count(*) AS events,
  count(DISTINCT conversation_id) AS conversations
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
