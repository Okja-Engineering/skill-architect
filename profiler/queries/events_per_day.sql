-- How many spool lines per day, by the event name the envelope carries?
--
-- Envelope fields only: `ts` and `event` are written by the capture layer, so
-- this describes the spool and asks nothing of the payload.
SELECT
  (ts AT TIME ZONE 'UTC')::DATE AS day,
  event,
  count(*) AS lines
FROM read_json(
  coalesce(getenv('SPOOL_DIR'), getenv('HOME') || '/.skill-architect/spool') || '/*.jsonl',
  format = 'newline_delimited',
  union_by_name = true,
  columns = {ts: 'TIMESTAMPTZ', event: 'VARCHAR', schema_version: 'VARCHAR',
             cursor_version: 'VARCHAR', cwd: 'VARCHAR', conversation_id: 'VARCHAR',
             strict: 'BOOLEAN', raw: 'JSON'}
)
GROUP BY 1, 2
ORDER BY 1, 3 DESC;
