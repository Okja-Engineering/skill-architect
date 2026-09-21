-- At what hour of the UTC day were lines captured?
--
-- `ts` is the capture time the spool writer stamped, not a time Cursor
-- reported, so this is when the hook ran on this machine.
SELECT
  hour(ts AT TIME ZONE 'UTC') AS hour_utc,
  count(*) AS lines,
  count(DISTINCT conversation_id) AS conversation_ids
FROM read_json(
  coalesce(getenv('SPOOL_DIR'), getenv('HOME') || '/.skill-architect/spool') || '/*.jsonl',
  format = 'newline_delimited',
  union_by_name = true,
  columns = {ts: 'TIMESTAMPTZ', event: 'VARCHAR', schema_version: 'VARCHAR',
             cursor_version: 'VARCHAR', cwd: 'VARCHAR', conversation_id: 'VARCHAR',
             strict: 'BOOLEAN', raw: 'JSON'}
)
GROUP BY 1
ORDER BY 1;
