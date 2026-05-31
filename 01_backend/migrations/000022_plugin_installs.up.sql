CREATE TABLE IF NOT EXISTS plugin_installs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    plugin_code TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'enabled',
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, plugin_code)
);

CREATE INDEX IF NOT EXISTS idx_plugin_installs_tenant
ON plugin_installs(tenant_id, plugin_code);
