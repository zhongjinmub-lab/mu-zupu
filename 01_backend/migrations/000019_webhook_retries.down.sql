DROP INDEX IF EXISTS idx_webhook_deliveries_retry_due;

ALTER TABLE webhook_deliveries
    DROP COLUMN IF EXISTS last_attempt_at,
    DROP COLUMN IF EXISTS next_retry_at;
