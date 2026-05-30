CREATE TABLE IF NOT EXISTS webhook_endpoints (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name text NOT NULL,
    url text NOT NULL,
    secret text,
    events jsonb NOT NULL DEFAULT '[]'::jsonb,
    status text NOT NULL DEFAULT 'active',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_tenant_status
    ON webhook_endpoints(tenant_id, status)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    endpoint_id uuid REFERENCES webhook_endpoints(id) ON DELETE SET NULL,
    event_type text NOT NULL,
    target_url text NOT NULL,
    status text NOT NULL,
    http_status int,
    request_body jsonb NOT NULL DEFAULT '{}'::jsonb,
    response_body text,
    error_message text,
    duration_ms int NOT NULL DEFAULT 0,
    retry_count int NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_tenant_created
    ON webhook_deliveries(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_endpoint_created
    ON webhook_deliveries(endpoint_id, created_at DESC);
