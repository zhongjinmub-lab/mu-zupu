# Production Deploy Templates

This directory records the production layout used for the current server deployment.

## Layout

```text
/opt/mu-agent-saas
├── bin/mu-agent-server
├── bin/mu-agent-migrate
├── bin/mu-agent-document-worker
├── docker-compose.yml
├── mu-agent-saas.env
├── migrations/
├── scripts/
└── backups/
```

## Install Outline

Build a Linux amd64 package from the backend root:

```bash
VERSION=v0.1.0 make release
```

On Windows:

```powershell
.\scripts\build_release.ps1 -Version v0.1.0
```

Then upload and extract the generated `dist/mu-agent-saas-*-linux-amd64.tar.gz`.

```bash
install -m 0755 bin/mu-agent-server /opt/mu-agent-saas/bin/mu-agent-server
install -m 0755 bin/mu-agent-migrate /opt/mu-agent-saas/bin/mu-agent-migrate
install -m 0755 bin/mu-agent-document-worker /opt/mu-agent-saas/bin/mu-agent-document-worker
cp -r migrations /opt/mu-agent-saas/
cp deploy/production/docker-compose.yml /opt/mu-agent-saas/docker-compose.yml
cp deploy/production/mu-agent-saas.env.example /opt/mu-agent-saas/mu-agent-saas.env
cp deploy/production/systemd/*.service /etc/systemd/system/
cp deploy/production/systemd/*.timer /etc/systemd/system/
```

Replace all secrets in `mu-agent-saas.env` and Compose `.env` before starting.
Keep `MINIO_ROOT_USER/MINIO_ROOT_PASSWORD` aligned with `STORAGE_ACCESS_KEY/STORAGE_SECRET_KEY`.

```bash
cd /opt/mu-agent-saas
docker compose --env-file compose.env up -d
set -a && . ./mu-agent-saas.env && set +a
./bin/mu-agent-migrate up
systemctl daemon-reload
systemctl enable --now mu-agent-saas
systemctl enable --now mu-agent-document-worker
systemctl enable --now mu-agent-saas-backup.timer
```

## Verification

```bash
/opt/mu-agent-saas/scripts/smoke.sh
/opt/mu-agent-saas/scripts/backup.sh
```
