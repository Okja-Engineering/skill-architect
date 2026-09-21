-- Which top-level keys do the captured payloads carry, and on which events?
--
-- This is the question the spool exists to answer. No payload shape in this
-- repository has been observed against a running Cursor: the field names in the
-- queries beside this one are taken from Cursor's documentation, so a query
-- returning no rows may mean the field is called something else rather than
-- that nothing happened. This one asks the files instead of assuming, and its
-- answer is what a later reader should be written against.
--
-- Only keys are listed, never values: a key census cannot leak a prompt.
SELECT
  key,
  count(*) AS lines,
  count(DISTINCT event) AS distinct_events,
  string_agg(DISTINCT event, ', ' ORDER BY event) AS events
FROM (
  SELECT event, unnest(json_keys(raw)) AS key
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
GROUP BY 1
ORDER BY 2 DESC, 1;
