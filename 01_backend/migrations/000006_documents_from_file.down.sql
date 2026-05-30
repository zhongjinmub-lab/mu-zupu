DROP INDEX IF EXISTS idx_documents_file_active;

ALTER TABLE documents
    DROP COLUMN IF EXISTS file_id;
