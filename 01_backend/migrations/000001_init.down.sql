DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS tool_call_logs;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS document_chunks;
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS knowledge_bases;
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS tenant_members;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;

-- 扩展一般不建议在业务回滚中 DROP，避免影响同库其他业务。
-- DROP EXTENSION IF EXISTS vector;
-- DROP EXTENSION IF EXISTS pgcrypto;
-- DROP EXTENSION IF EXISTS citext;
-- DROP EXTENSION IF EXISTS pg_trgm;
