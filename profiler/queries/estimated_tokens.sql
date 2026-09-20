-- Estimated context tokens per conversation, and per inferred skill.
-- ESTIMATE ONLY: length(raw)/4 — a relative signal, never billed usage.
-- The only Cursor-reported number is raw.context_tokens on preCompact.
WITH spool AS (
  SELECT * FROM read_json(
    coalesce(getenv('SPOOL_DIR'), getenv('HOME') || '/.cursor-profiler/spool') || '/*.jsonl',
    format = 'newline_delimited',
    union_by_name = true,
    columns = {ts: 'TIMESTAMPTZ', event: 'VARCHAR', schema_version: 'VARCHAR',
               cursor_version: 'VARCHAR', cwd: 'VARCHAR', conversation_id: 'VARCHAR',
               raw: 'JSON'}
  )
),
per_conversation AS (
  SELECT
    conversation_id AS label,
    count(*) AS events,
    sum(length(raw::VARCHAR)) / 4 AS est_tokens,
    max((raw->>'$.context_tokens')::BIGINT) AS cursor_reported_context_tokens
  FROM spool
  WHERE conversation_id IS NOT NULL
  GROUP BY 1
),
per_skill AS (
  SELECT
    regexp_extract(raw->>'$.file_path', 'skills[/\\]([^/\\]+)[/\\]SKILL\.md$', 1) AS label,
    count(*) AS events,
    sum(length(raw::VARCHAR)) / 4 AS est_tokens,
    NULL::BIGINT AS cursor_reported_context_tokens
  FROM spool
  WHERE event IN ('beforeReadFile', 'beforeTabFileRead')
    AND regexp_matches(raw->>'$.file_path', 'skills[/\\][^/\\]+[/\\]SKILL\.md$')
  GROUP BY 1
)
SELECT 'per_conversation' AS scope, * FROM per_conversation
UNION ALL
SELECT 'per_skill', * FROM per_skill
ORDER BY 1, 3 DESC;
