-- 智能体族谱SAAS - PostgreSQL 初始化迁移
-- 重点：多租户、RAG、pgvector、混合检索、审计与可扩展字段

CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'active',
    plan_code TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email CITEXT UNIQUE,
    mobile TEXT UNIQUE,
    password_hash TEXT,
    nickname TEXT,
    avatar_url TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS tenant_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    user_id UUID NOT NULL REFERENCES users(id),
    role_code TEXT NOT NULL DEFAULT 'member',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, user_id)
);

CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    description TEXT,
    system_prompt TEXT,
    model_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    tool_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
    memory_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'draft',
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, code)
);

CREATE TABLE IF NOT EXISTS knowledge_bases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    embedding_provider TEXT NOT NULL DEFAULT 'openai_compatible',
    embedding_model TEXT NOT NULL DEFAULT 'text-embedding-3-small',
    embedding_dim INT NOT NULL DEFAULT 1536,
    chunk_config JSONB NOT NULL DEFAULT '{"max_chars":1200,"overlap_chars":120}'::jsonb,
    retrieval_config JSONB NOT NULL DEFAULT '{"top_k":10,"min_score":0.2,"hybrid":true,"vector_weight":0.7,"text_weight":0.3}'::jsonb,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, code),
    CONSTRAINT chk_kb_embedding_dim CHECK (embedding_dim IN (384, 512, 768, 1024, 1536, 3072))
);

CREATE TABLE IF NOT EXISTS documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    knowledge_base_id UUID NOT NULL REFERENCES knowledge_bases(id),
    title TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT 'upload',
    source_uri TEXT,
    mime_type TEXT,
    content_sha256 TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    parse_status TEXT NOT NULL DEFAULT 'pending',
    chunk_status TEXT NOT NULL DEFAULT 'pending',
    embedding_status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- 说明：当前 MVP 默认 1536 维，适配 text-embedding-3-small / bge-m3 映射策略。
-- 若模型维度不同，建议按维度拆分 chunk 表，如 document_chunks_768 / document_chunks_3072，避免单表 vector 维度不一致。
CREATE TABLE IF NOT EXISTS document_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    knowledge_base_id UUID NOT NULL REFERENCES knowledge_bases(id),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    chunk_no INT NOT NULL,
    content TEXT NOT NULL,
    content_tokens INT NOT NULL DEFAULT 0,
    content_sha256 TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    embedding vector(1536),
    embedding_model TEXT,
    embedding_status TEXT NOT NULL DEFAULT 'pending',
    search_vector tsvector GENERATED ALWAYS AS (to_tsvector('simple', coalesce(content, ''))) STORED,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(document_id, chunk_no)
);

CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    agent_id UUID REFERENCES agents(id),
    user_id UUID REFERENCES users(id),
    title TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    token_usage JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tool_call_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    agent_id UUID REFERENCES agents(id),
    conversation_id UUID REFERENCES conversations(id),
    tool_name TEXT NOT NULL,
    input JSONB NOT NULL DEFAULT '{}'::jsonb,
    output JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL,
    cost_ms INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),
    actor_user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    ip TEXT,
    user_agent TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_agents_tenant_status ON agents(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_kb_tenant_status ON knowledge_bases(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_documents_kb_status ON documents(tenant_id, knowledge_base_id, embedding_status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_chunks_tenant_kb_doc ON document_chunks(tenant_id, knowledge_base_id, document_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_chunks_metadata_gin ON document_chunks USING gin(metadata);
CREATE INDEX IF NOT EXISTS idx_chunks_search_vector_gin ON document_chunks USING gin(search_vector);
CREATE INDEX IF NOT EXISTS idx_chunks_content_trgm ON document_chunks USING gin(content gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_conversations_tenant_agent ON conversations(tenant_id, agent_id, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_messages_conversation_time ON messages(conversation_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_tool_logs_tenant_time ON tool_call_logs(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_time ON audit_logs(tenant_id, created_at DESC);

-- pgvector 索引策略：
-- 1. HNSW：适合在线高召回低延迟检索，构建较慢、占用内存更高。
-- 2. IVFFLAT：适合百万级以上批量数据，需 ANALYZE，lists/probes 需要按数据量调优。
-- MVP 默认启用 HNSW cosine；若写入量特别大，可先不建索引，批量导入完成后再 CONCURRENTLY 建索引。
CREATE INDEX IF NOT EXISTS idx_chunks_embedding_hnsw_cosine
ON document_chunks USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64)
WHERE embedding IS NOT NULL AND deleted_at IS NULL;

-- 可选 IVFFLAT，数据量超过 100万 chunk 且批量导入场景可启用：
-- CREATE INDEX CONCURRENTLY idx_chunks_embedding_ivfflat_cosine
-- ON document_chunks USING ivfflat (embedding vector_cosine_ops)
-- WITH (lists = 1000)
-- WHERE embedding IS NOT NULL AND deleted_at IS NULL;
