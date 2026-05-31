CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    description TEXT,
    definition JSONB NOT NULL DEFAULT '{"nodes":[],"edges":[]}'::jsonb,
    status TEXT NOT NULL DEFAULT 'draft',
    version INT NOT NULL DEFAULT 1,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS idx_workflows_tenant_status
ON workflows(tenant_id, status) WHERE deleted_at IS NULL;
