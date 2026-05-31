CREATE TABLE IF NOT EXISTS workflow_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    workflow_id UUID NOT NULL REFERENCES workflows(id),
    status TEXT NOT NULL,
    mode TEXT NOT NULL DEFAULT 'dry_run',
    input JSONB NOT NULL DEFAULT '{}'::jsonb,
    steps JSONB NOT NULL DEFAULT '[]'::jsonb,
    cost_ms INT NOT NULL DEFAULT 0,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_workflow_runs_tenant_time
ON workflow_runs(tenant_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_workflow_runs_workflow
ON workflow_runs(tenant_id, workflow_id, created_at DESC);
