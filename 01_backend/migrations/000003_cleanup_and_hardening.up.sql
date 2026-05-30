-- 智能体族谱SAAS - 清理冗余与检索硬化
-- 目标：减少无效/重复数据，补齐状态约束、去重索引、检索日志清理策略。

CREATE UNIQUE INDEX IF NOT EXISTS idx_documents_kb_sha_active
ON documents(tenant_id, knowledge_base_id, content_sha256)
WHERE deleted_at IS NULL AND content_sha256 IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_chunks_doc_sha_active
ON document_chunks(document_id, content_sha256)
WHERE deleted_at IS NULL AND content_sha256 IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_chunks_embedding_pending
ON document_chunks(tenant_id, knowledge_base_id, created_at)
WHERE deleted_at IS NULL AND embedding_status IN ('pending', 'failed');

CREATE INDEX IF NOT EXISTS idx_documents_parse_pending
ON documents(tenant_id, knowledge_base_id, created_at)
WHERE deleted_at IS NULL AND (parse_status IN ('pending', 'failed') OR chunk_status IN ('pending', 'failed') OR embedding_status IN ('pending', 'failed'));

ALTER TABLE documents
    ADD CONSTRAINT chk_documents_parse_status CHECK (parse_status IN ('pending', 'processing', 'success', 'failed', 'skipped')),
    ADD CONSTRAINT chk_documents_chunk_status CHECK (chunk_status IN ('pending', 'processing', 'success', 'failed', 'skipped')),
    ADD CONSTRAINT chk_documents_embedding_status CHECK (embedding_status IN ('pending', 'processing', 'success', 'failed', 'skipped'));

ALTER TABLE document_chunks
    ADD CONSTRAINT chk_chunks_embedding_status CHECK (embedding_status IN ('pending', 'processing', 'success', 'failed', 'skipped')),
    ADD CONSTRAINT chk_chunks_content_not_blank CHECK (length(trim(content)) > 0),
    ADD CONSTRAINT chk_chunks_tokens_non_negative CHECK (content_tokens >= 0);

ALTER TABLE vector_index_profiles
    ADD CONSTRAINT chk_vector_profile_distance CHECK (distance_metric IN ('cosine', 'l2', 'ip')),
    ADD CONSTRAINT chk_vector_profile_index_type CHECK (index_type IN ('hnsw', 'ivfflat')),
    ADD CONSTRAINT chk_vector_profile_score CHECK (min_score >= 0 AND min_score <= 1),
    ADD CONSTRAINT chk_vector_profile_hnsw CHECK (hnsw_m BETWEEN 4 AND 64 AND hnsw_ef_construction BETWEEN 16 AND 512 AND hnsw_ef_search BETWEEN 10 AND 500);

-- 可选：生产环境建议通过定时任务删除 180 天前的检索明细，保留聚合报表。
-- DELETE FROM vector_search_logs WHERE created_at < now() - interval '180 days';
