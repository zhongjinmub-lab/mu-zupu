CREATE TABLE IF NOT EXISTS business_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    order_no TEXT NOT NULL UNIQUE,
    order_type TEXT NOT NULL DEFAULT 'subscription',
    plan_id UUID REFERENCES plans(id),
    amount_cents BIGINT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'CNY',
    status TEXT NOT NULL DEFAULT 'pending',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT chk_business_orders_status CHECK (status IN ('pending', 'paid', 'closed', 'cancelled')),
    CONSTRAINT chk_business_orders_amount CHECK (amount_cents >= 0)
);

CREATE TABLE IF NOT EXISTS payment_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    business_order_id UUID NOT NULL REFERENCES business_orders(id),
    pay_no TEXT NOT NULL UNIQUE,
    channel TEXT NOT NULL DEFAULT 'mock',
    amount_cents BIGINT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'CNY',
    status TEXT NOT NULL DEFAULT 'pending',
    transaction_id TEXT,
    paid_at TIMESTAMPTZ,
    callback_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT chk_payment_orders_status CHECK (status IN ('pending', 'paid', 'failed', 'closed')),
    CONSTRAINT chk_payment_orders_amount CHECK (amount_cents >= 0)
);

CREATE INDEX IF NOT EXISTS idx_business_orders_tenant_status
ON business_orders(tenant_id, status, created_at DESC)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_payment_orders_tenant_status
ON payment_orders(tenant_id, status, created_at DESC)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_payment_orders_business_order
ON payment_orders(tenant_id, business_order_id)
WHERE deleted_at IS NULL;
