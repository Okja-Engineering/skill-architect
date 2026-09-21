-- What values does the payload field `model` carry?
--
-- Read this as "lines whose payload said model was X", not as "X served N
-- requests": one Cursor turn may fire several hooks, and whether every hook
-- payload carries the field at all is documentary. `payload_keys.sql` says
-- which keys are really there.
--
-- The field is projected in a CTE behind a `json_type` guard; see the note in
-- `tool_call_mix.sql` for the non-object payload that makes the outer form
-- fail.
WITH payload AS (
  SELECT
    conversation_id,
    raw->>'$.model' AS model_value
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
SELECT
  model_value,
  count(*) AS lines,
  count(DISTINCT conversation_id) AS conversation_ids
FROM payload
WHERE model_value IS NOT NULL
GROUP BY 1
ORDER BY 2 DESC;
