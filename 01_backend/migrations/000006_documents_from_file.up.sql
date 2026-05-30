-- 智能体族谱SAAS - 从文件生成文档与切片

ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS file_id UUID REFERENCES files(id);

CREATE INDEX IF NOT EXISTS idx_documents_file_active
ON documents(tenant_id, file_id)
WHERE deleted_at IS NULL AND file_id IS NOT NULL;
