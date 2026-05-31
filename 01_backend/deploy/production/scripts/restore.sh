#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/mu-agent-saas}"
DB_BACKUP="${1:-${DB_BACKUP:-}}"
TARGET_DB="${TARGET_DB:-mu_agent_saas}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-mu-agent-saas-postgres}"
POSTGRES_USER="${POSTGRES_USER:-mu}"
CONFIRM_RESTORE="${CONFIRM_RESTORE:-}"

if [[ -z "$DB_BACKUP" ]]; then
  echo "usage: CONFIRM_RESTORE=yes DB_BACKUP=/opt/mu-agent-saas/backups/mu_agent_saas_YYYYmmddHHMMSS.sql.gz $0"
  echo "   or: CONFIRM_RESTORE=yes $0 /opt/mu-agent-saas/backups/mu_agent_saas_YYYYmmddHHMMSS.sql.gz"
  exit 2
fi

if [[ ! -f "$DB_BACKUP" ]]; then
  echo "database backup not found: $DB_BACKUP"
  exit 2
fi

if [[ "$CONFIRM_RESTORE" != "yes" ]]; then
  echo "refuse to restore without CONFIRM_RESTORE=yes"
  echo "target database will be dropped and recreated: $TARGET_DB"
  exit 2
fi

echo "step 1/5 backup current database before restore"
"$APP_DIR/scripts/backup.sh"

echo "step 2/5 stop application services"
systemctl stop mu-agent-webhook-worker || true
systemctl stop mu-agent-document-worker || true
systemctl stop mu-agent-saas || true

echo "step 3/5 terminate active database sessions"
docker exec -i "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d postgres -v ON_ERROR_STOP=1 <<SQL
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = '$TARGET_DB'
  AND pid <> pg_backend_pid();
SQL

echo "step 4/5 drop and restore database"
docker exec -i "$POSTGRES_CONTAINER" dropdb -U "$POSTGRES_USER" --if-exists "$TARGET_DB"
docker exec -i "$POSTGRES_CONTAINER" createdb -U "$POSTGRES_USER" "$TARGET_DB"
gzip -dc "$DB_BACKUP" | docker exec -i "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d "$TARGET_DB" -v ON_ERROR_STOP=1 >/tmp/mu-agent-restore.log

echo "step 5/5 restart services and verify"
systemctl start mu-agent-saas
systemctl start mu-agent-document-worker
systemctl start mu-agent-webhook-worker
"$APP_DIR/scripts/smoke.sh"

echo "restore ok db=$TARGET_DB backup=$DB_BACKUP"
