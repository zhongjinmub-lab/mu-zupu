#!/usr/bin/env bash
set -euo pipefail

BACKUP_DIR=/opt/mu-agent-saas/backups
mkdir -p "$BACKUP_DIR"

TS=$(date +%Y%m%d%H%M%S)
DB_FILE="$BACKUP_DIR/mu_agent_saas_${TS}.sql.gz"
CFG_FILE="$BACKUP_DIR/mu_agent_saas_config_${TS}.tar.gz"

docker exec mu-agent-saas-postgres pg_dump -U mu -d mu_agent_saas | gzip > "$DB_FILE"
tar -C /opt -czf "$CFG_FILE" \
  mu-agent-saas/docker-compose.yml \
  mu-agent-saas/mu-agent-saas.env \
  mu-agent-saas/migrations \
  mu-agent-saas/scripts

find "$BACKUP_DIR" -type f -name '*.gz' -mtime +14 -delete

echo "backup ok $DB_FILE $CFG_FILE"
