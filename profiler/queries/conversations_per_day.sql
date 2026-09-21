-- How many distinct conversation ids appear per day?
--
-- A conversation id is an envelope field, promoted out of the payload by the
-- capture layer. Two lines carrying the same id were captured under the same
-- id; whether Cursor starts a new one per session is its business and not a
-- claim made here.
SELECT
  (ts AT TIME ZONE 'UTC')::DATE AS day,
  count(DISTINCT conversation_id) AS conversation_ids,
  count(*) FILTER (event = 'sessionStart') AS session_start_lines
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
