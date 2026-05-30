ALTER TABLE tenant_members
    ADD CONSTRAINT chk_tenant_members_role_code
    CHECK (role_code IN ('owner', 'admin', 'member', 'viewer'));

ALTER TABLE tenant_members
    ADD CONSTRAINT chk_tenant_members_status
    CHECK (status IN ('active', 'inactive'));

CREATE INDEX IF NOT EXISTS idx_tenant_members_tenant_status
ON tenant_members(tenant_id, status, role_code)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_action_time
ON audit_logs(tenant_id, action, created_at DESC);
