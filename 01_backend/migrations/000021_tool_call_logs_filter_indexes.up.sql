CREATE INDEX IF NOT EXISTS idx_tool_logs_tenant_time_id
ON tool_call_logs(tenant_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_tool_logs_tenant_status_time
ON tool_call_logs(tenant_id, status, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_tool_logs_tenant_agent_time
ON tool_call_logs(tenant_id, agent_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_tool_logs_tenant_tool_time
ON tool_call_logs(tenant_id, tool_name, created_at DESC, id DESC);
