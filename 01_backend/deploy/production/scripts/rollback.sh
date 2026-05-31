#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/mu-agent-saas}"
ROLLBACK_DIR="${1:-${ROLLBACK_DIR:-}}"
MIGRATION_STEPS="${MIGRATION_STEPS:-0}"

if [[ -z "$ROLLBACK_DIR" && -f "$APP_DIR/rollback/last_release_path" ]]; then
  ROLLBACK_DIR="$(cat "$APP_DIR/rollback/last_release_path")"
fi

if [[ -z "$ROLLBACK_DIR" ]]; then
  echo "usage: ROLLBACK_DIR=/opt/mu-agent-saas/rollback/current-YYYYmmddHHMMSS $0"
  echo "   or: $0 /opt/mu-agent-saas/rollback/current-YYYYmmddHHMMSS"
  exit 2
fi

if [[ ! -d "$ROLLBACK_DIR" ]]; then
  echo "rollback snapshot not found: $ROLLBACK_DIR"
  exit 2
fi

echo "step 1/5 backup current database and config before rollback"
"$APP_DIR/scripts/backup.sh"

echo "step 2/5 stop services"
systemctl stop mu-agent-webhook-worker || true
systemctl stop mu-agent-document-worker || true
systemctl stop mu-agent-saas || true

echo "step 3/5 restore runtime files from snapshot"
for item in bin migrations scripts nginx systemd frontend docker-compose.yml compose.env.example mu-agent-saas.env.example README.md; do
  if [[ -e "$ROLLBACK_DIR/$item" ]]; then
    rm -rf "$APP_DIR/$item"
    cp -a "$ROLLBACK_DIR/$item" "$APP_DIR/$item"
  fi
done

echo "step 4/5 optionally roll back database migrations"
cd "$APP_DIR"
set -a
. ./mu-agent-saas.env
set +a
if [[ "$MIGRATION_STEPS" =~ ^[0-9]+$ ]] && [[ "$MIGRATION_STEPS" -gt 0 ]]; then
  for ((i = 1; i <= MIGRATION_STEPS; i++)); do
    ./bin/mu-agent-migrate down
  done
else
  echo "skip migration rollback because MIGRATION_STEPS=0"
fi

echo "step 5/5 restart services and smoke check"
systemctl daemon-reload
systemctl start mu-agent-saas
systemctl start mu-agent-document-worker
systemctl start mu-agent-webhook-worker
"$APP_DIR/scripts/smoke.sh"

echo "rollback ok snapshot=$ROLLBACK_DIR migration_steps=$MIGRATION_STEPS"
