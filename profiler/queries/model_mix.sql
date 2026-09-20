-- Which models do I actually use?
SELECT
  raw->>'$.model' AS model,
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
WHERE raw->>'$.model' IS NOT NULL
GROUP BY 1
ORDER BY 2 DESC;
