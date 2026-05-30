-- 智能体族谱SAAS - 回滚文档切片重建支持

DROP INDEX IF EXISTS idx_documents_rebuild_active;
DROP INDEX IF EXISTS idx_document_chunks_active_chunk_no;

DELETE FROM document_chunks
WHERE deleted_at IS NOT NULL;

ALTER TABLE document_chunks
    ADD CONSTRAINT document_chunks_document_id_chunk_no_key UNIQUE(document_id, chunk_no);
