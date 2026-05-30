-- 智能体族谱SAAS - 文档解析/切片任务队列

CREATE TABLE IF NOT EXISTS document_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    knowledge_base_id UUID NOT NULL REFERENCES knowledge_bases(id),
    document_id UUID REFERENCES documents(id),
    file_id UUID NOT NULL REFERENCES files(id),
    job_type TEXT NOT NULL DEFAULT 'parse_chunk',
    status TEXT NOT NULL DEFAULT 'pending',
    max_chars INT NOT NULL DEFAULT 1200,
    overlap_chars INT NOT NULL DEFAULT 120,
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT chk_document_jobs_type CHECK (job_type IN ('parse_chunk', 'rebuild')),
    CONSTRAINT chk_document_jobs_status CHECK (status IN ('pending', 'processing', 'success', 'failed')),
    CONSTRAINT chk_document_jobs_chunk_params CHECK (max_chars > 0 AND max_chars <= 8000 AND overlap_chars >= 0 AND overlap_chars < max_chars)
);

CREATE INDEX IF NOT EXISTS idx_document_jobs_tenant_kb_status
ON document_jobs(tenant_id, knowledge_base_id, status, created_at)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_document_jobs_pending
ON document_jobs(status, created_at)
WHERE deleted_at IS NULL AND status IN ('pending', 'failed');
