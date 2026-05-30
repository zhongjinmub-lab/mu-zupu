#!/usr/bin/env bash
set -euo pipefail

BASE="${BASE_URL:-https://zupu.jiangxinnet.com/saas-api/api/v1}"

curl -kfsS "$BASE/health" >/tmp/mu-agent-health.json
curl -kfsS "$BASE/ready" >/tmp/mu-agent-ready.json
systemctl is-active --quiet mu-agent-saas

cd /opt/mu-agent-saas
set -a
. ./mu-agent-saas.env
set +a
./bin/mu-agent-migrate status | grep -q '000004_auth_tenant_kb_mvp applied'

echo "smoke ok $(date -Is)"
