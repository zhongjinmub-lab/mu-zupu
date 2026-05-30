ALTER TABLE vector_index_profiles
    DROP CONSTRAINT IF EXISTS chk_vector_profile_hnsw,
    DROP CONSTRAINT IF EXISTS chk_vector_profile_score,
    DROP CONSTRAINT IF EXISTS chk_vector_profile_index_type,
    DROP CONSTRAINT IF EXISTS chk_vector_profile_distance;

ALTER TABLE document_chunks
    DROP CONSTRAINT IF EXISTS chk_chunks_tokens_non_negative,
    DROP CONSTRAINT IF EXISTS chk_chunks_content_not_blank,
    DROP CONSTRAINT IF EXISTS chk_chunks_embedding_status;

ALTER TABLE documents
    DROP CONSTRAINT IF EXISTS chk_documents_embedding_status,
    DROP CONSTRAINT IF EXISTS chk_documents_chunk_status,
    DROP CONSTRAINT IF EXISTS chk_documents_parse_status;

DROP INDEX IF EXISTS idx_documents_parse_pending;
DROP INDEX IF EXISTS idx_chunks_embedding_pending;
DROP INDEX IF EXISTS idx_chunks_doc_sha_active;
DROP INDEX IF EXISTS idx_documents_kb_sha_active;
