# 生产部署模板

本目录固化当前私有化部署结构，所有示例文件只放占位配置，不写入真实密钥。

## 目录结构

```text
/opt/mu-agent-saas
├── bin/mu-agent-server
├── bin/mu-agent-migrate
├── bin/mu-agent-document-worker
├── bin/mu-agent-webhook-worker
├── docker-compose.yml
├── compose.env
├── mu-agent-saas.env
├── migrations/
├── frontend/
├── scripts/
├── rollback/
└── backups/
```

## 打包

在后端根目录执行：

```bash
VERSION=v0.1.0 make release
```

Windows 本地打包：

```powershell
.\scripts\build_release.ps1 -Version v0.1.0
```

生成产物：`dist/mu-agent-saas-<version>-linux-amd64.tar.gz`。

## 首次安装

将发布包上传到服务器并解压，然后安装文件：

```bash
install -m 0755 bin/mu-agent-server /opt/mu-agent-saas/bin/mu-agent-server
install -m 0755 bin/mu-agent-migrate /opt/mu-agent-saas/bin/mu-agent-migrate
install -m 0755 bin/mu-agent-document-worker /opt/mu-agent-saas/bin/mu-agent-document-worker
install -m 0755 bin/mu-agent-webhook-worker /opt/mu-agent-saas/bin/mu-agent-webhook-worker
cp -r migrations scripts nginx systemd /opt/mu-agent-saas/
cp -r frontend /opt/mu-agent-saas/frontend
cp docker-compose.yml /opt/mu-agent-saas/docker-compose.yml
cp compose.env.example /opt/mu-agent-saas/compose.env
cp mu-agent-saas.env.example /opt/mu-agent-saas/mu-agent-saas.env
cp systemd/*.service /etc/systemd/system/
cp systemd/*.timer /etc/systemd/system/
```

启动前必须替换 `mu-agent-saas.env` 和 `compose.env` 中的全部生产密钥。`MINIO_ROOT_USER/MINIO_ROOT_PASSWORD` 需与 `STORAGE_ACCESS_KEY/STORAGE_SECRET_KEY` 保持一致。

```bash
cd /opt/mu-agent-saas
docker compose --env-file compose.env up -d
set -a && . ./mu-agent-saas.env && set +a
./bin/mu-agent-migrate up
systemctl daemon-reload
systemctl enable --now mu-agent-saas
systemctl enable --now mu-agent-document-worker
systemctl enable --now mu-agent-webhook-worker
systemctl enable --now mu-agent-saas-backup.timer
```

## 升级

升级脚本会先执行备份，再解压新发布包、保存当前运行文件快照、安装新二进制、迁移文件和前端静态文件、执行 `mu-agent-migrate up`、重启服务并运行冒烟检查。

```bash
cd /opt/mu-agent-saas
RELEASE_ARCHIVE=/opt/mu-agent-saas/releases/mu-agent-saas-v0.2.0-linux-amd64.tar.gz ./scripts/upgrade.sh
```

成功后会输出 `rollback_snapshot=/opt/mu-agent-saas/rollback/current-...`，用于必要时回滚。

## 回滚

默认回滚到最近一次升级前保存的运行文件快照。默认不回滚数据库迁移，避免误删生产数据；确需回滚迁移时，明确设置 `MIGRATION_STEPS`。

```bash
cd /opt/mu-agent-saas
./scripts/rollback.sh
```

指定快照回滚：

```bash
ROLLBACK_DIR=/opt/mu-agent-saas/rollback/current-20260531120000 ./scripts/rollback.sh
```

回滚 1 个数据库迁移版本：

```bash
MIGRATION_STEPS=1 ./scripts/rollback.sh
```

## 验证

```bash
/opt/mu-agent-saas/scripts/smoke.sh
/opt/mu-agent-saas/scripts/backup.sh
```

`backup.sh` 会生成两类备份：

- `mu_agent_saas_<timestamp>.sql.gz`：PostgreSQL 数据库备份。
- `mu_agent_saas_config_<timestamp>.tar.gz`：运行配置归档，包含 `docker-compose.yml`、`mu-agent-saas.env`、`migrations/`、`scripts/`、`frontend/`、`nginx/` 和 `systemd/`。

`smoke.sh` 默认检查：

- `https://zupu.jiangxinnet.com/saas-api/api/v1/health`
- `https://zupu.jiangxinnet.com/saas-api/api/v1/ready`
- `https://zupu.jiangxinnet.com/saas/`
- `https://zupu.jiangxinnet.com/saas/assets/app.js`
- `https://zupu.jiangxinnet.com/saas/assets/app.css`
- `nginx -t`
- `mu-agent-saas`、`mu-agent-document-worker`、`mu-agent-webhook-worker` systemd 状态
- `mu-agent-migrate status`

可按环境覆盖：

```bash
API_BASE_URL=https://example.com/saas-api/api/v1 \
FRONTEND_BASE_URL=https://example.com/saas \
EXPECTED_MIGRATION='000004_auth_tenant_kb_mvp applied' \
/opt/mu-agent-saas/scripts/smoke.sh
```

## 备份恢复演练

日常演练优先使用 `restore-drill.sh`，它会把备份恢复到临时数据库 `mu_agent_saas_restore_drill`，校验 `schema_migrations` 后自动删除临时库，不影响生产库。

```bash
cd /opt/mu-agent-saas
./scripts/backup.sh
./scripts/restore-drill.sh
```

指定备份文件演练：

```bash
DB_BACKUP=/opt/mu-agent-saas/backups/mu_agent_saas_20260531120000.sql.gz ./scripts/restore-drill.sh
```

如需保留演练库用于人工检查：

```bash
KEEP_DRILL_DB=yes ./scripts/restore-drill.sh
```

真实恢复会删除并重建目标数据库，必须显式设置 `CONFIRM_RESTORE=yes`。执行真实恢复前，脚本会再次调用 `backup.sh` 生成当前现场备份。

```bash
CONFIRM_RESTORE=yes DB_BACKUP=/opt/mu-agent-saas/backups/mu_agent_saas_20260531120000.sql.gz ./scripts/restore.sh
```

配置归档恢复用于恢复 `frontend/`、`nginx/`、`systemd/`、`scripts/`、`migrations/`、`docker-compose.yml` 和 `mu-agent-saas.env` 等运行文件，必须显式设置 `CONFIRM_CONFIG_RESTORE=yes`。执行前会再次调用 `backup.sh` 生成当前现场备份。

```bash
CONFIRM_CONFIG_RESTORE=yes CONFIG_BACKUP=/opt/mu-agent-saas/backups/mu_agent_saas_config_20260531120000.tar.gz ./scripts/restore-config.sh
```

升级和回滚后都必须确认：

- `systemctl status mu-agent-saas`
- `systemctl status mu-agent-document-worker`
- `systemctl status mu-agent-webhook-worker`
- 公网 `/saas-api/api/v1/health`、`/saas-api/api/v1/ready` 返回正常
- 公网 `/saas/`、`/saas/assets/app.js`、`/saas/assets/app.css` 返回正常
- `./bin/mu-agent-migrate status` 输出符合预期
