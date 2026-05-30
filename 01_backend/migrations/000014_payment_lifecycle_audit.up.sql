CREATE TABLE IF NOT EXISTS payment_callback_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    payment_order_id UUID REFERENCES payment_orders(id),
    pay_no TEXT NOT NULL,
    channel TEXT NOT NULL,
    event_status TEXT NOT NULL,
    transaction_id TEXT,
    request_id TEXT,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    result_status TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payment_callback_events_tenant_time
ON payment_callback_events(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_payment_callback_events_pay_no
ON payment_callback_events(tenant_id, channel, pay_no, created_at DESC);
