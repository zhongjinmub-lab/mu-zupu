CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_action_time_id
ON audit_logs(tenant_id, action, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_resource_time
ON audit_logs(tenant_id, resource_type, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_actor_time
ON audit_logs(tenant_id, actor_user_id, created_at DESC, id DESC);
