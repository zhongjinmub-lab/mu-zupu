#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/mu-agent-saas}"
API_BASE_URL="${API_BASE_URL:-${BASE_URL:-https://zupu.jiangxinnet.com/saas-api/api/v1}}"
FRONTEND_BASE_URL="${FRONTEND_BASE_URL:-https://zupu.jiangxinnet.com/saas}"
EXPECTED_MIGRATION="${EXPECTED_MIGRATION:-000004_auth_tenant_kb_mvp applied}"
CHECK_NGINX="${CHECK_NGINX:-yes}"

check_url() {
  local name="$1"
  local url="$2"
  local expected_type="${3:-}"
  local meta
  meta="$(curl -kfsS -o /dev/null -w '%{http_code} %{content_type}' "$url")"
  local status="${meta%% *}"
  local content_type="${meta#* }"
  if [[ "$status" != "200" ]]; then
    echo "FAIL $name status=$status url=$url"
    exit 1
  fi
  if [[ -n "$expected_type" && "$content_type" != *"$expected_type"* ]]; then
    echo "FAIL $name content_type=$content_type expected=$expected_type url=$url"
    exit 1
  fi
  echo "OK $name status=$status content_type=$content_type"
}

check_service() {
  local service="$1"
  systemctl is-active --quiet "$service"
  echo "OK service $service active"
}

check_url "api_health" "$API_BASE_URL/health" "application/json"
check_url "api_ready" "$API_BASE_URL/ready" "application/json"
check_url "frontend_html" "$FRONTEND_BASE_URL/" "text/html"

frontend_html="$(curl -kfsS "$FRONTEND_BASE_URL/")"
frontend_js_path="$(printf '%s' "$frontend_html" | grep -oE 'src="[^"]+\.js"' | head -n 1 | sed -E 's/^src="|"$//g')"
frontend_css_path="$(printf '%s' "$frontend_html" | grep -oE 'href="[^"]+\.css"' | head -n 1 | sed -E 's/^href="|"$//g')"
if [[ -z "$frontend_js_path" || -z "$frontend_css_path" ]]; then
  echo "FAIL frontend assets not found"
  exit 1
fi
frontend_origin="$(printf '%s' "$FRONTEND_BASE_URL" | sed -E 's#^(https?://[^/]+).*#\1#')"
if [[ "$frontend_js_path" == http* ]]; then
  frontend_js_url="$frontend_js_path"
elif [[ "$frontend_js_path" == /* ]]; then
  frontend_js_url="$frontend_origin$frontend_js_path"
else
  frontend_js_url="${FRONTEND_BASE_URL%/}/$frontend_js_path"
fi
if [[ "$frontend_css_path" == http* ]]; then
  frontend_css_url="$frontend_css_path"
elif [[ "$frontend_css_path" == /* ]]; then
  frontend_css_url="$frontend_origin$frontend_css_path"
else
  frontend_css_url="${FRONTEND_BASE_URL%/}/$frontend_css_path"
fi
check_url "frontend_js" "$frontend_js_url" "javascript"
check_url "frontend_css" "$frontend_css_url" "text/css"

if [[ "$CHECK_NGINX" == "yes" ]] && command -v nginx >/dev/null 2>&1; then
  nginx -t >/dev/null
  echo "OK nginx config"
fi

check_service "mu-agent-saas"
check_service "mu-agent-document-worker"
check_service "mu-agent-webhook-worker"

cd "$APP_DIR"
set -a
. ./mu-agent-saas.env
set +a
./bin/mu-agent-migrate status | grep -q "$EXPECTED_MIGRATION"
echo "OK migration $EXPECTED_MIGRATION"

echo "smoke ok $(date -Is)"
