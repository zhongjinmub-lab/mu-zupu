CREATE TABLE IF NOT EXISTS plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    price_cents BIGINT NOT NULL DEFAULT 0,
    billing_cycle TEXT NOT NULL DEFAULT 'month',
    quota JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    plan_id UUID NOT NULL REFERENCES plans(id),
    status TEXT NOT NULL DEFAULT 'active',
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expired_at TIMESTAMPTZ,
    auto_renew BOOLEAN NOT NULL DEFAULT false,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    subject_type TEXT NOT NULL,
    subject_id UUID,
    metric TEXT NOT NULL,
    quantity NUMERIC(20,6) NOT NULL DEFAULT 0,
    unit TEXT NOT NULL DEFAULT 'count',
    request_id TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant_active
ON subscriptions(tenant_id, status, started_at DESC)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_usage_records_tenant_metric_time
ON usage_records(tenant_id, metric, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_usage_records_subject_time
ON usage_records(tenant_id, subject_type, subject_id, occurred_at DESC);

INSERT INTO plans(code, name, quota)
VALUES (
    'free',
    'Free',
    '{"rag_requests":1000,"agent_messages":1000,"file_upload_bytes":104857600,"embedding_chunks":5000}'::jsonb
)
ON CONFLICT (code) DO NOTHING;
