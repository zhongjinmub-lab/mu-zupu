ALTER TABLE knowledge_bases
    DROP CONSTRAINT IF EXISTS chk_kb_embedding_dim_1536;

DROP INDEX IF EXISTS idx_documents_tenant_kb_id_active;
DROP INDEX IF EXISTS idx_kb_tenant_id_active;
DROP INDEX IF EXISTS idx_tenant_members_user_active;
DROP INDEX IF EXISTS idx_users_email_active;

ALTER TABLE tenant_members
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS updated_at;

ALTER TABLE users
    DROP COLUMN IF EXISTS last_login_at;
