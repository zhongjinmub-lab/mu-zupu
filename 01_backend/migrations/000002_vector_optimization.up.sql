-- 智能体族谱SAAS - 向量存储与索引增强迁移
-- 用途：补齐向量检索参数表、检索日志、按租户/知识库的调优能力。

CREATE TABLE IF NOT EXISTS vector_index_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),
    knowledge_base_id UUID REFERENCES knowledge_bases(id),
    profile_name TEXT NOT NULL DEFAULT 'default',
    distance_metric TEXT NOT NULL DEFAULT 'cosine',
    index_type TEXT NOT NULL DEFAULT 'hnsw',
    hnsw_m INT NOT NULL DEFAULT 16,
    hnsw_ef_construction INT NOT NULL DEFAULT 64,
    hnsw_ef_search INT NOT NULL DEFAULT 80,
    ivfflat_lists INT NOT NULL DEFAULT 1000,
    ivfflat_probes INT NOT NULL DEFAULT 10,
    vector_weight NUMERIC(5,4) NOT NULL DEFAULT 0.7000,
    text_weight NUMERIC(5,4) NOT NULL DEFAULT 0.3000,
    rerank_weight NUMERIC(5,4) NOT NULL DEFAULT 0.0000,
    top_k INT NOT NULL DEFAULT 10,
    candidate_k INT NOT NULL DEFAULT 50,
    min_score NUMERIC(8,6) NOT NULL DEFAULT 0.200000,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, knowledge_base_id, profile_name),
    CONSTRAINT chk_vector_profile_weights CHECK (vector_weight >= 0 AND text_weight >= 0 AND rerank_weight >= 0),
    CONSTRAINT chk_vector_profile_topk CHECK (top_k > 0 AND top_k <= 100 AND candidate_k >= top_k)
);

CREATE TABLE IF NOT EXISTS vector_search_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    knowledge_base_id UUID REFERENCES knowledge_bases(id),
    agent_id UUID REFERENCES agents(id),
    query TEXT,
    embedding_model TEXT,
    top_k INT NOT NULL DEFAULT 10,
    candidate_k INT NOT NULL DEFAULT 50,
    min_score NUMERIC(8,6),
    result_count INT NOT NULL DEFAULT 0,
    latency_ms INT NOT NULL DEFAULT 0,
    profile JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_vector_profiles_tenant_kb ON vector_index_profiles(tenant_id, knowledge_base_id) WHERE enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_vector_search_logs_tenant_time ON vector_search_logs(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_vector_search_logs_kb_time ON vector_search_logs(knowledge_base_id, created_at DESC);

-- 向量检索常用会话参数示例：
-- SET LOCAL hnsw.ef_search = 80;
-- SET LOCAL ivfflat.probes = 10;

-- 推荐按规模调整：
-- < 10万 chunks: HNSW(m=16, ef_construction=64, ef_search=40~80)
-- 10万~300万 chunks: HNSW(m=16~32, ef_search=80~160) 或 IVFFLAT(lists=sqrt(N)~N/1000)
-- > 300万 chunks: 分区 + IVFFLAT/HNSW + 冷热数据拆分 + 重排模型
