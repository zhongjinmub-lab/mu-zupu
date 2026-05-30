-- 智能体族谱SAAS - 认证、租户上下文与 KB 写入 MVP 补强
-- 说明：不修改历史迁移，仅补齐认证链路和租户成员查询所需字段、索引与约束。

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

ALTER TABLE tenant_members
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_users_email_active
ON users(email)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tenant_members_user_active
ON tenant_members(user_id, tenant_id)
WHERE deleted_at IS NULL AND status = 'active';

CREATE INDEX IF NOT EXISTS idx_kb_tenant_id_active
ON knowledge_bases(tenant_id, id)
WHERE deleted_at IS NULL AND status = 'active';

CREATE INDEX IF NOT EXISTS idx_documents_tenant_kb_id_active
ON documents(tenant_id, knowledge_base_id, id)
WHERE deleted_at IS NULL;

ALTER TABLE knowledge_bases
    ADD CONSTRAINT chk_kb_embedding_dim_1536 CHECK (embedding_dim = 1536);
