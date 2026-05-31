-- 智能体版本管理表
-- 支持 Agent 的版本快照、发布与回滚

CREATE TABLE IF NOT EXISTS agent_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    version_no VARCHAR(32) NOT NULL,
    prompt TEXT,
    model_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    tool_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    knowledge_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    channel VARCHAR(64) NOT NULL DEFAULT 'web',
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    publish_note TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ,
    CONSTRAINT uq_agent_version UNIQUE(agent_id, version_no),
    CONSTRAINT chk_agent_version_status CHECK (status IN ('draft', 'published', 'archived', 'rollback'))
);

COMMENT ON TABLE agent_versions IS '智能体版本快照，记录每次发布时的配置状态';
COMMENT ON COLUMN agent_versions.version_no IS '版本号，格式如 v1.0.0';
COMMENT ON COLUMN agent_versions.channel IS '发布渠道：web, wechat, api, miniapp 等';
COMMENT ON COLUMN agent_versions.status IS '版本状态：draft 草稿、published 已发布、archived 已归档、rollback 已回滚';
COMMENT ON COLUMN agent_versions.publish_note IS '发布说明/变更备注';

-- 按租户和 Agent 查询版本列表
CREATE INDEX IF NOT EXISTS idx_agent_versions_tenant_agent
ON agent_versions(tenant_id, agent_id, created_at DESC);

-- 按状态筛选已发布版本
CREATE INDEX IF NOT EXISTS idx_agent_versions_status
ON agent_versions(agent_id, status) WHERE status = 'published';

-- 按渠道查询
CREATE INDEX IF NOT EXISTS idx_agent_versions_channel
ON agent_versions(agent_id, channel);
