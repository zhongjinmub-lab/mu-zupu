# 智能体族谱SAAS 最终交付包

更新时间：2026-05-31

## 1. 目录说明

| 目录/文件 | 说明 |
|---|---|
| `01_backend` | Go + Gin 后端工程，含多租户、License、支付、RAG、Webhook、SSE、限流、审计、运维脚本 |
| `02_frontend` | 免构建静态管理台，覆盖总览、知识库、Agent、族谱、License、用量订单、Webhook、审计和设置 |
| `02_docs_v1_delivery` | API、数据库、Redis Key、增量验收和交付验收文档 |
| `03_智能体族谱SAAS_开发方案.md` | 总体开发方案 |
| `04_最终交付验收清单.md` | 顶层最终交付验收清单 |
| `manifest.json` | 文件清单与 SHA256 摘要 |

## 2. 技术基线

- Go + Gin
- PostgreSQL 16 + pgvector
- Redis 7.4
- MinIO / S3
- 静态 HTML/CSS/JavaScript 管理台
- Docker Compose
- systemd + Nginx 私有化部署模板

## 3. 已完成

- SaaS 多租户、登录鉴权、角色权限、成员邀请和租户隔离。
- 知识库文件上传、文档归档、切片、向量化、RAG 检索和问答生成闭环。
- Agent 创建、编辑、发布、回滚、归档、知识库绑定、多轮会话和 SSE 流式对话。
- 智能体族谱图谱、关系维护、筛选、结构诊断、统计分析和 CSV 导出。
- 套餐、订阅、用量统计、额度限制、订单生命周期和支付回调验签。
- License 在线/离线验证、签名脱敏和授权生命周期管理。
- Webhook 配置、测试发送、投递记录、重试 Worker、状态摘要和 CSV 导出。
- API 限流、通用审计、审计筛选分页和 CSV 导出。
- 管理台中文摘要展示，避免新增黑色 JSON 原文区域。
- 生产部署模板、Linux amd64 打包、Nginx/systemd、冒烟检查、备份、升级、回滚和恢复演练脚本。
- `02_docs_v1_delivery/06_交付验收清单.md` 已全部勾选。

## 4. 验证命令

```bash
cd 01_backend
go test ./...
```

```powershell
D:\Node.js\node.exe --check 02_frontend\assets\app.js
```

当前本地验证结果：通过。

## 5. 部署与运维

生产部署模板位于：

```text
01_backend/deploy/production/
```

关键脚本：

- `scripts/backup.sh`：数据库和配置备份。
- `scripts/restore-drill.sh`：恢复到临时库演练并校验。
- `scripts/restore.sh`：显式确认后的真实恢复。
- `scripts/upgrade.sh`：发布包升级、迁移、重启和冒烟检查。
- `scripts/rollback.sh`：运行文件回滚，可选迁移回滚。
- `scripts/smoke.sh`：健康检查和迁移状态检查。

线上入口记录：

- 管理台：[https://zupu.jiangxinnet.com/saas/](https://zupu.jiangxinnet.com/saas/)
- 后端 API：[https://zupu.jiangxinnet.com/saas-api/api/v1/ready](https://zupu.jiangxinnet.com/saas-api/api/v1/ready)

## 6. 后续增强建议

- Vue3 + TypeScript + Vite + Element Plus 工程化迁移。
- 真实第三方支付渠道接入。
- 多实例生产环境 Redis 限流压测和监控告警增强。
