CREATE TABLE IF NOT EXISTS agent_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    agent_id UUID NOT NULL REFERENCES agents(id),
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'enabled',
    channel_key TEXT NOT NULL DEFAULT ('ch_' || replace(gen_random_uuid()::text, '-', '')),
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (tenant_id, channel_key)
);

CREATE INDEX IF NOT EXISTS idx_agent_channels_tenant_status
ON agent_channels(tenant_id, status) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_agent_channels_agent
ON agent_channels(tenant_id, agent_id) WHERE deleted_at IS NULL;
