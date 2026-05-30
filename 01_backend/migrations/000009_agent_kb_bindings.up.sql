-- 智能体族谱SAAS - Agent 与知识库绑定

CREATE TABLE IF NOT EXISTS agent_knowledge_bases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    knowledge_base_id UUID NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'active',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, agent_id, knowledge_base_id),
    CONSTRAINT chk_agent_kb_binding_status CHECK (status IN ('active', 'disabled'))
);

CREATE INDEX IF NOT EXISTS idx_agent_kb_bindings_agent
ON agent_knowledge_bases(tenant_id, agent_id, status)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_agent_kb_bindings_kb
ON agent_knowledge_bases(tenant_id, knowledge_base_id, status)
WHERE deleted_at IS NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_agents_status'
          AND conrelid = 'agents'::regclass
    ) THEN
        ALTER TABLE agents
            ADD CONSTRAINT chk_agents_status
            CHECK (status IN ('draft', 'published', 'archived'));
    END IF;
END $$;
