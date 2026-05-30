INSERT INTO plans(code, name, quota, status)
VALUES (
    'free',
    'Free',
    '{"rag_requests":1000,"agent_messages":1000,"file_upload_bytes":104857600,"embedding_chunks":5000}'::jsonb,
    'active'
)
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    quota = EXCLUDED.quota,
    status = 'active',
    deleted_at = NULL,
    updated_at = now();
