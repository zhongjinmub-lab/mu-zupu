#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:-dev}"
OUT_DIR="${OUT_DIR:-dist}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST="$ROOT/$OUT_DIR"
PACKAGE_NAME="mu-agent-saas-${VERSION}-linux-amd64"
PACKAGE_DIR="$DIST/$PACKAGE_NAME"

rm -rf "$PACKAGE_DIR"
mkdir -p "$PACKAGE_DIR/bin"

cd "$ROOT"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$PACKAGE_DIR/bin/mu-agent-server" ./cmd/server
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$PACKAGE_DIR/bin/mu-agent-migrate" ./cmd/migrate
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$PACKAGE_DIR/bin/mu-agent-document-worker" ./cmd/document-worker
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$PACKAGE_DIR/bin/mu-agent-webhook-worker" ./cmd/webhook-worker

cp -R migrations "$PACKAGE_DIR/migrations"
cp deploy/production/docker-compose.yml "$PACKAGE_DIR/docker-compose.yml"
cp deploy/production/compose.env.example "$PACKAGE_DIR/compose.env.example"
cp deploy/production/mu-agent-saas.env.example "$PACKAGE_DIR/mu-agent-saas.env.example"
cp -R deploy/production/systemd "$PACKAGE_DIR/systemd"
cp -R deploy/production/scripts "$PACKAGE_DIR/scripts"
cp -R deploy/production/nginx "$PACKAGE_DIR/nginx"
cp deploy/production/README.md "$PACKAGE_DIR/README.md"

tar -C "$DIST" -czf "$DIST/$PACKAGE_NAME.tar.gz" "$PACKAGE_NAME"
echo "release package: $DIST/$PACKAGE_NAME.tar.gz"
