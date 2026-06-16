#!/usr/bin/env bash
# 针对 P1 新增只读端点的冒烟测试：工具 / MCP / 插件 / 工作流 / 渠道。
# 需要环境变量 TOKEN 与 TENANT_ID（登录后获取）；未设置时仅提示并以 0 退出。
set -euo pipefail

API_BASE_URL="${API_BASE_URL:-${BASE_URL:-https://zupu.jiangxinnet.com/saas-api/api/v1}}"
TOKEN="${TOKEN:-}"
TENANT_ID="${TENANT_ID:-}"

if [[ -z "$TOKEN" || -z "$TENANT_ID" ]]; then
  echo "SKIP feature smoke: 需要设置 TOKEN 与 TENANT_ID 环境变量"
  exit 0
fi

check_get() {
  local name="$1"
  local path="$2"
  local meta
  meta="$(curl -kfsS -o /dev/null -w '%{http_code} %{content_type}' \
    -H "Authorization: Bearer $TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    "$API_BASE_URL$path")"
  local status="${meta%% *}"
  local content_type="${meta#* }"
  if [[ "$status" != "200" ]]; then
    echo "FAIL $name status=$status path=$path"
    exit 1
  fi
  if [[ "$content_type" != *application/json* ]]; then
    echo "FAIL $name content_type=$content_type path=$path"
    exit 1
  fi
  echo "OK $name status=$status"
}

# 插件工具（工具安全策略 / 工具目录 / 工具调用日志 / MCP 网关 / 插件市场）
check_get "tool_safety_policy" "/agents/tool-safety-policy"
check_get "tools" "/tools"
check_get "tool_call_logs" "/tool-call-logs?limit=5"
check_get "mcp_gateway_policy" "/mcp-gateway/policy"
check_get "mcp_servers" "/mcp-servers"
check_get "plugins" "/plugins"

# 工作流（编排策略 / 节点类型 / 列表 / 概览统计）
check_get "workflow_policy" "/workflows/orchestration-policy"
check_get "workflow_node_types" "/workflow-node-types"
check_get "workflows" "/workflows"
check_get "workflows_summary" "/workflows/summary"

# 渠道入口（类型目录 / 列表 / 概览统计）
check_get "channel_types" "/channel-types"
check_get "channels" "/channels"
check_get "channels_summary" "/channels/summary"

echo "feature smoke ok $(date -Is)"
