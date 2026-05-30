ALTER TABLE webhook_deliveries
    ADD COLUMN IF NOT EXISTS next_retry_at timestamptz,
    ADD COLUMN IF NOT EXISTS last_attempt_at timestamptz NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_retry_due
    ON webhook_deliveries(next_retry_at, created_at)
    WHERE status = 'failed' AND next_retry_at IS NOT NULL;
