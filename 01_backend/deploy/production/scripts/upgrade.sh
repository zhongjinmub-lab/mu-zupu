#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/mu-agent-saas}"
RELEASE_ARCHIVE="${1:-${RELEASE_ARCHIVE:-}}"

if [[ -z "$RELEASE_ARCHIVE" ]]; then
  echo "usage: RELEASE_ARCHIVE=/path/to/package.tar.gz $0"
  echo "   or: $0 /path/to/package.tar.gz"
  exit 2
fi

if [[ ! -f "$RELEASE_ARCHIVE" ]]; then
  echo "release archive not found: $RELEASE_ARCHIVE"
  exit 2
fi

mkdir -p "$APP_DIR/releases" "$APP_DIR/rollback" "$APP_DIR/backups"
TS="$(date +%Y%m%d%H%M%S)"
STAGE_DIR="$APP_DIR/releases/stage-$TS"
CURRENT_DIR="$APP_DIR/rollback/current-$TS"

echo "step 1/7 backup current database and config"
"$APP_DIR/scripts/backup.sh"

echo "step 2/7 extract release archive"
mkdir -p "$STAGE_DIR"
tar -xzf "$RELEASE_ARCHIVE" -C "$STAGE_DIR" --strip-components=1

echo "step 3/7 snapshot current runtime files"
mkdir -p "$CURRENT_DIR"
for item in bin migrations scripts nginx systemd frontend docker-compose.yml compose.env.example mu-agent-saas.env.example README.md; do
  if [[ -e "$APP_DIR/$item" ]]; then
    cp -a "$APP_DIR/$item" "$CURRENT_DIR/"
  fi
done

echo "step 4/7 install release files"
install -m 0755 "$STAGE_DIR/bin/mu-agent-server" "$APP_DIR/bin/mu-agent-server"
install -m 0755 "$STAGE_DIR/bin/mu-agent-migrate" "$APP_DIR/bin/mu-agent-migrate"
install -m 0755 "$STAGE_DIR/bin/mu-agent-document-worker" "$APP_DIR/bin/mu-agent-document-worker"
install -m 0755 "$STAGE_DIR/bin/mu-agent-webhook-worker" "$APP_DIR/bin/mu-agent-webhook-worker"
cp -a "$STAGE_DIR/migrations" "$APP_DIR/migrations"
cp -a "$STAGE_DIR/scripts" "$APP_DIR/scripts"
cp -a "$STAGE_DIR/nginx" "$APP_DIR/nginx"
cp -a "$STAGE_DIR/systemd" "$APP_DIR/systemd"
if [[ -d "$STAGE_DIR/frontend" ]]; then
  rm -rf "$APP_DIR/frontend"
  cp -a "$STAGE_DIR/frontend" "$APP_DIR/frontend"
fi
cp -f "$STAGE_DIR/docker-compose.yml" "$APP_DIR/docker-compose.yml"
cp -f "$STAGE_DIR/compose.env.example" "$APP_DIR/compose.env.example"
cp -f "$STAGE_DIR/mu-agent-saas.env.example" "$APP_DIR/mu-agent-saas.env.example"
cp -f "$STAGE_DIR/README.md" "$APP_DIR/README.md"
echo "$CURRENT_DIR" > "$APP_DIR/rollback/last_release_path"

echo "step 5/7 apply database migrations"
cd "$APP_DIR"
set -a
. ./mu-agent-saas.env
set +a
./bin/mu-agent-migrate up

echo "step 6/7 restart services"
systemctl daemon-reload
systemctl restart mu-agent-saas
systemctl restart mu-agent-document-worker
systemctl restart mu-agent-webhook-worker

echo "step 7/7 run smoke check"
"$APP_DIR/scripts/smoke.sh"

echo "upgrade ok $TS rollback_snapshot=$CURRENT_DIR"
