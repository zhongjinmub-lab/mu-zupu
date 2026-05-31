-- 回滚：删除 agent_versions 表及其索引
DROP INDEX IF EXISTS idx_agent_versions_channel;
DROP INDEX IF EXISTS idx_agent_versions_status;
DROP INDEX IF EXISTS idx_agent_versions_tenant_agent;
DROP TABLE IF EXISTS agent_versions;
