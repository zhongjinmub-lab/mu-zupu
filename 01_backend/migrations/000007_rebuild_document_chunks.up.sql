-- 智能体族谱SAAS - 支持文档切片重建

ALTER TABLE document_chunks
    DROP CONSTRAINT IF EXISTS document_chunks_document_id_chunk_no_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_document_chunks_active_chunk_no
ON document_chunks(document_id, chunk_no)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_documents_rebuild_active
ON documents(tenant_id, knowledge_base_id, id, file_id)
WHERE deleted_at IS NULL;
