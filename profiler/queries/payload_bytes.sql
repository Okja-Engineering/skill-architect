-- How many payload bytes were captured, per conversation id and per day?
--
-- Bytes, deliberately: this file used to divide them by four and call the
-- result estimated context tokens. Nothing in this release measures a token
-- from a spool — there is no adapter that reads one — and a column named for a
-- token count travels without the sentence that said it was an estimate. What
-- is true of these files is how many bytes of payload are in them, and a
-- conversation whose payloads are ten times larger than another's is a real
-- signal about the corpus without pretending to be a billed number.
--
-- Note `strict`: on a line captured in metadata-only mode the content fields
-- were replaced by their sizes before the line was written, so its byte count
-- is not comparable with an unstripped one. The two are separated rather than
-- summed.
WITH spool AS (
  SELECT * FROM read_json(
    coalesce(getenv('SPOOL_DIR'), getenv('HOME') || '/.skill-architect/spool') || '/*.jsonl',
    format = 'newline_delimited',
    union_by_name = true,
    columns = {ts: 'TIMESTAMPTZ', event: 'VARCHAR', schema_version: 'VARCHAR',
               cursor_version: 'VARCHAR', cwd: 'VARCHAR', conversation_id: 'VARCHAR',
               strict: 'BOOLEAN', raw: 'JSON'}
  )
)
SELECT
  'per_conversation_id' AS scope,
  conversation_id AS label,
  count(*) AS lines,
  count(*) FILTER (strict) AS stripped_lines,
  sum(length(raw::VARCHAR)) FILTER (strict IS NOT TRUE) AS payload_bytes_unstripped,
  sum(length(raw::VARCHAR)) FILTER (strict) AS payload_bytes_stripped
FROM spool
WHERE conversation_id IS NOT NULL
GROUP BY 1, 2
UNION ALL
SELECT
  'per_day',
  ((ts AT TIME ZONE 'UTC')::DATE)::VARCHAR,
  count(*),
  count(*) FILTER (strict),
  sum(length(raw::VARCHAR)) FILTER (strict IS NOT TRUE),
  sum(length(raw::VARCHAR)) FILTER (strict)
FROM spool
GROUP BY 1, 2
ORDER BY 1, 5 DESC;
