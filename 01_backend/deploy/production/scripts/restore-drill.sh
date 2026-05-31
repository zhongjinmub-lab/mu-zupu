#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/mu-agent-saas}"
DB_BACKUP="${1:-${DB_BACKUP:-}}"
DRILL_DB="${DRILL_DB:-mu_agent_saas_restore_drill}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-mu-agent-saas-postgres}"
POSTGRES_USER="${POSTGRES_USER:-mu}"
KEEP_DRILL_DB="${KEEP_DRILL_DB:-no}"

if [[ -z "$DB_BACKUP" ]]; then
  DB_BACKUP="$(ls -1t "$APP_DIR"/backups/mu_agent_saas_*.sql.gz 2>/dev/null | head -n 1 || true)"
fi

if [[ -z "$DB_BACKUP" || ! -f "$DB_BACKUP" ]]; then
  echo "usage: DB_BACKUP=/opt/mu-agent-saas/backups/mu_agent_saas_YYYYmmddHHMMSS.sql.gz $0"
  echo "   or: $0 /opt/mu-agent-saas/backups/mu_agent_saas_YYYYmmddHHMMSS.sql.gz"
  exit 2
fi

cleanup() {
  if [[ "$KEEP_DRILL_DB" != "yes" ]]; then
    docker exec -i "$POSTGRES_CONTAINER" dropdb -U "$POSTGRES_USER" --if-exists "$DRILL_DB" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

echo "step 1/4 prepare drill database $DRILL_DB"
docker exec -i "$POSTGRES_CONTAINER" dropdb -U "$POSTGRES_USER" --if-exists "$DRILL_DB" >/dev/null
docker exec -i "$POSTGRES_CONTAINER" createdb -U "$POSTGRES_USER" "$DRILL_DB"

echo "step 2/4 restore backup into drill database"
gzip -dc "$DB_BACKUP" | docker exec -i "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d "$DRILL_DB" -v ON_ERROR_STOP=1 >/tmp/mu-agent-restore-drill.log

echo "step 3/4 verify restored schema"
docker exec -i "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d "$DRILL_DB" -v ON_ERROR_STOP=1 -tAc "SELECT COUNT(*) FROM schema_migrations;" >/tmp/mu-agent-restore-drill-schema-count.txt
SCHEMA_COUNT="$(tr -d '[:space:]' </tmp/mu-agent-restore-drill-schema-count.txt)"
if [[ -z "$SCHEMA_COUNT" || "$SCHEMA_COUNT" -lt 1 ]]; then
  echo "schema_migrations is empty after restore drill"
  exit 1
fi
docker exec -i "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d "$DRILL_DB" -v ON_ERROR_STOP=1 -c "SELECT version, name, applied_at FROM schema_migrations ORDER BY version DESC LIMIT 5;"

echo "step 4/4 cleanup drill database"
cleanup
trap - EXIT

echo "restore drill ok db=$DRILL_DB backup=$DB_BACKUP schema_migrations=$SCHEMA_COUNT"
