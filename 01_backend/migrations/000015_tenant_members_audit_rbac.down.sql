DROP INDEX IF EXISTS idx_audit_logs_tenant_action_time;
DROP INDEX IF EXISTS idx_tenant_members_tenant_status;

ALTER TABLE tenant_members
    DROP CONSTRAINT IF EXISTS chk_tenant_members_status;

ALTER TABLE tenant_members
    DROP CONSTRAINT IF EXISTS chk_tenant_members_role_code;
