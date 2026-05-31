CREATE TABLE IF NOT EXISTS agent_genealogy (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    parent_agent_id UUID REFERENCES agents(id),
    child_agent_id UUID NOT NULL REFERENCES agents(id),
    relation_type TEXT NOT NULL DEFAULT 'fork',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_agent_genealogy_relation_type CHECK (relation_type IN ('fork', 'inherit', 'compose', 'route')),
    CONSTRAINT chk_agent_genealogy_not_self CHECK (parent_agent_id IS NULL OR parent_agent_id <> child_agent_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_agent_genealogy_edge
ON agent_genealogy(tenant_id, parent_agent_id, child_agent_id, relation_type);

CREATE INDEX IF NOT EXISTS idx_agent_genealogy_tenant_parent
ON agent_genealogy(tenant_id, parent_agent_id);

CREATE INDEX IF NOT EXISTS idx_agent_genealogy_tenant_child
ON agent_genealogy(tenant_id, child_agent_id);
