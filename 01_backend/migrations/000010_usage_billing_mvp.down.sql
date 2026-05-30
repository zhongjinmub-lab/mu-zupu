DROP INDEX IF EXISTS idx_usage_records_subject_time;
DROP INDEX IF EXISTS idx_usage_records_tenant_metric_time;
DROP INDEX IF EXISTS idx_subscriptions_tenant_active;

DROP TABLE IF EXISTS usage_records;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS plans;
