-- 智能体族谱SAAS - 回滚文档解析/切片任务队列

DROP INDEX IF EXISTS idx_document_jobs_pending;
DROP INDEX IF EXISTS idx_document_jobs_tenant_kb_status;
DROP TABLE IF EXISTS document_jobs;
