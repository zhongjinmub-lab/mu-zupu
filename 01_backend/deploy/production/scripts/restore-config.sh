#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/mu-agent-saas}"
CONFIG_BACKUP="${1:-${CONFIG_BACKUP:-}}"
CONFIRM_CONFIG_RESTORE="${CONFIRM_CONFIG_RESTORE:-}"

if [[ -z "$CONFIG_BACKUP" ]]; then
  CONFIG_BACKUP="$(ls -1t "$APP_DIR"/backups/mu_agent_saas_config_*.tar.gz 2>/dev/null | head -n 1 || true)"
fi

if [[ -z "$CONFIG_BACKUP" || ! -f "$CONFIG_BACKUP" ]]; then
  echo "usage: CONFIRM_CONFIG_RESTORE=yes CONFIG_BACKUP=/opt/mu-agent-saas/backups/mu_agent_saas_config_YYYYmmddHHMMSS.tar.gz $0"
  echo "   or: CONFIRM_CONFIG_RESTORE=yes $0 /opt/mu-agent-saas/backups/mu_agent_saas_config_YYYYmmddHHMMSS.tar.gz"
  exit 2
fi

if [[ "$CONFIRM_CONFIG_RESTORE" != "yes" ]]; then
  echo "refuse to restore config without CONFIRM_CONFIG_RESTORE=yes"
  echo "runtime files will be replaced from config backup: $CONFIG_BACKUP"
  exit 2
fi

echo "step 1/5 backup current database and config before config restore"
"$APP_DIR/scripts/backup.sh"

echo "step 2/5 extract config backup"
TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT
tar -xzf "$CONFIG_BACKUP" -C "$TMP_DIR"

RESTORE_ROOT="$TMP_DIR/mu-agent-saas"
if [[ ! -d "$RESTORE_ROOT" ]]; then
  echo "invalid config backup: missing mu-agent-saas root"
  exit 2
fi

echo "step 3/5 stop services"
systemctl stop mu-agent-webhook-worker || true
systemctl stop mu-agent-document-worker || true
systemctl stop mu-agent-saas || true

echo "step 4/5 restore runtime config files"
for item in docker-compose.yml mu-agent-saas.env migrations scripts frontend nginx systemd; do
  if [[ -e "$RESTORE_ROOT/$item" ]]; then
    rm -rf "$APP_DIR/$item"
    cp -a "$RESTORE_ROOT/$item" "$APP_DIR/$item"
  fi
done
chmod +x "$APP_DIR/scripts/"*.sh

echo "step 5/5 reload services and verify"
systemctl daemon-reload
systemctl start mu-agent-saas
systemctl start mu-agent-document-worker
systemctl start mu-agent-webhook-worker
"$APP_DIR/scripts/smoke.sh"

echo "config restore ok backup=$CONFIG_BACKUP"
