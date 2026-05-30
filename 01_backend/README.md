# 智能体族谱SAAS Backend

Go + Gin 后端工程，提供 SaaS 多租户、知识库检索和后续 Agent Runtime 扩展基础。

## 1. 技术栈

- Go go1.26.3
- Gin
- PostgreSQL 16 + pgvector
- Redis 7.4
- MinIO
- Docker Compose

## 2. 启动

```bash
make compose-up
make migrate-up
make run
```

## 3. 验证

```bash
go test ./...
```

迁移状态：

```bash
make migrate-status
```

构建 Linux amd64 发布包：

```powershell
.\scripts\build_release.ps1 -Version v0.1.0
```

或在 Linux/macOS：

```bash
VERSION=v0.1.0 make release
```

## 4. 已落地接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /api/v1/health | 健康检查，包含数据库连通性 |
| GET | /api/v1/ready | 就绪检查 |
| POST | /api/v1/auth/register | 用户注册并签发 JWT |
| POST | /api/v1/auth/login | 用户登录并签发 JWT |
| GET | /api/v1/auth/me | 当前用户信息 |
| GET | /api/v1/tenants | 当前用户租户列表 |
| POST | /api/v1/tenants | 创建租户并绑定 owner 成员 |
| GET | /api/v1/audit-logs/export | 导出当前租户审计日志 CSV，支持 action/resource_type/actor_user_id/from/to 过滤 |
| POST | /api/v1/agents | 创建 Agent |
| GET | /api/v1/agents | 当前租户 Agent 列表 |
| GET | /api/v1/agents/:agent_id | Agent 详情 |
| PUT | /api/v1/agents/:agent_id | 更新 Agent，已发布 Agent 更新后回到 draft |
| POST | /api/v1/agents/:agent_id/publish | 发布 Agent |
| POST | /api/v1/agents/:agent_id/rollback | 回滚 Agent 到 draft |
| POST | /api/v1/agents/:agent_id/test-chat | Agent 测试会话，基于绑定 KB 检索并生成回答 |
| POST | /api/v1/agents/:agent_id/chat | Agent 多轮会话，支持 conversation_id 续聊、历史消息编排和 RAG 引用 |
| GET | /api/v1/agents/:agent_id/conversations | 当前租户 Agent 会话列表 |
| GET | /api/v1/agents/:agent_id/conversations/:conversation_id/messages | 指定会话消息列表 |
| DELETE | /api/v1/agents/:agent_id | 归档 Agent |
| POST | /api/v1/agents/:agent_id/knowledge-bases | 绑定当前租户 KB |
| GET | /api/v1/agents/:agent_id/knowledge-bases | Agent 已绑定 KB 列表 |
| DELETE | /api/v1/agents/:agent_id/knowledge-bases/:kb_id | 解绑 Agent KB |
| GET | /api/v1/billing/plans | 可用套餐列表 |
| GET | /api/v1/billing/subscription | 当前租户订阅，未订阅时自动创建 free 订阅 |
| GET | /api/v1/billing/usage/summary | 当前租户用量汇总，支持 from/to 查询 |
| POST | /api/v1/orders | 创建当前租户业务订单 |
| GET | /api/v1/orders | 当前租户订单列表 |
| POST | /api/v1/payment-orders | 创建 mock 支付单 |
| POST | /api/v1/payments/:payment_id/query | 查询当前租户支付单 |
| POST | /api/v1/payment-callbacks/:channel | mock 支付回调，幂等更新支付单、订单和订阅 |
| GET | /api/v1/analytics/summary | 当前租户统计汇总，包含资源、经营、用量趋势、最近操作和风险 |
| GET | /api/v1/licenses | 当前租户 License 列表 |
| POST | /api/v1/licenses | 创建当前租户 License，支持 limits/subject 和过期时间 |
| POST | /api/v1/licenses/:license_id/verify | 验证当前租户 License 离线签名和状态 |
| POST | /api/v1/licenses/:license_id/activate | 激活当前租户 License |
| POST | /api/v1/licenses/:license_id/revoke | 吊销当前租户 License |
| POST | /api/v1/files/upload | 上传文件到 MinIO/S3 并写入文件记录 |
| GET | /api/v1/files | 当前租户文件列表 |
| GET | /api/v1/kbs | 当前租户知识库列表 |
| POST | /api/v1/kbs | 创建知识库 |
| POST | /api/v1/kbs/:kb_id/documents | 创建知识库文档 |
| POST | /api/v1/kbs/:kb_id/documents/from-file | 从已上传文本文件生成文档和 pending chunks |
| POST | /api/v1/kbs/:kb_id/documents/:document_id/rebuild | 重建 file-backed 文档切片 |
| POST | /api/v1/kbs/:kb_id/document-jobs | 创建文档解析/切片任务 |
| GET | /api/v1/kbs/:kb_id/document-jobs | 查询文档任务列表 |
| POST | /api/v1/kbs/:kb_id/document-jobs/run | 同步执行 pending/failed 文档任务 |
| POST | /api/v1/kbs/:kb_id/chunks | 创建 chunk 并写入 1536 维 embedding |
| GET | /api/v1/kbs/:kb_id/chunks/pending | 查询待向量化 chunks |
| PUT | /api/v1/kbs/:kb_id/chunks/:chunk_id/embedding | 写回单个 chunk 的 1536 维 embedding |
| POST | /api/v1/kbs/:kb_id/embedding/run | 同步处理 pending chunks 并写回 1536 维 embedding |
| POST | /api/v1/kbs/:kb_id/search | 当前租户知识库检索 |
| POST | /api/v1/kbs/:kb_id/ask | RAG 问答：问题向量化、混合检索、生成回答并返回引用 |
| POST | /api/v1/kb/search/vector | 知识库向量检索 |
| POST | /api/v1/kb/search/hybrid | 知识库混合检索 |

租户级接口需要 `Authorization: Bearer <token>` 和 `X-Tenant-ID: <tenant_id>`。

## 5. 数据库迁移

迁移文件位于 `migrations/`：

```text
000001_init
000002_vector_optimization
000003_cleanup_and_hardening
000004_auth_tenant_kb_mvp
000005_files_upload
000006_documents_from_file
000007_rebuild_document_chunks
000008_document_jobs
000009_agent_kb_bindings
000010_usage_billing_mvp
000011_license_mvp
000012_ensure_free_plan
000013_orders_payments_mvp
```

可执行命令：

- `make migrate-up`：顺序执行未应用的 `*.up.sql`；
- `make migrate-down`：回滚最后一个已应用 migration；
- `make migrate-status`：查看已应用和待执行 migration。

迁移记录写入 `schema_migrations`，包含版本、名称、checksum 和应用时间。

## 6. 生产配置

- `APP_ENV=production` 时 Gin 使用 Release Mode；
- `JWT_SECRET` 必须通过环境变量注入，长度至少 32 字符；
- `JWT_ISSUER` 默认 `mu-agent-saas`；
- `JWT_TTL_HOURS` 默认 24；
- `STORAGE_ENDPOINT` MinIO / S3 endpoint，默认 `127.0.0.1:19000`；
- `STORAGE_ACCESS_KEY` 对象存储 access key；
- `STORAGE_SECRET_KEY` 对象存储 secret key；
- `STORAGE_BUCKET` 文件桶，默认 `mu-agent-files`；
- `STORAGE_USE_SSL` 对象存储是否使用 HTTPS；
- `STORAGE_PUBLIC_BASE` 文件公开访问前缀，默认空；
- `UPLOAD_MAX_BYTES` 上传大小限制，默认 50MB；
- `EMBEDDING_PROVIDER` embedding provider，默认 `local`，支持 `local`、`openai_compatible`、`http`；
- `EMBEDDING_MODEL` embedding model，默认 `local-hash-1536`；
- `EMBEDDING_BASE_URL` OpenAI-compatible endpoint 前缀，例如 `https://api.example.com/v1`，系统会请求 `{base_url}/embeddings`；
- `EMBEDDING_API_KEY` 外部 embedding 服务 API key，通过 `Authorization: Bearer` 发送，不能提交到代码仓库；
- `EMBEDDING_TIMEOUT_SECONDS` 外部 embedding 请求超时，默认 30；
- `GENERATION_PROVIDER` generation provider，默认 `local`，支持 `local`、`openai_compatible`、`http`；
- `GENERATION_MODEL` generation model，默认 `local-rag`；
- `GENERATION_BASE_URL` OpenAI-compatible endpoint 前缀，例如 `https://api.example.com/v1`，系统会请求 `{base_url}/chat/completions`；
- `GENERATION_API_KEY` 外部 generation 服务 API key，通过 `Authorization: Bearer` 发送，不能提交到代码仓库；
- `GENERATION_TIMEOUT_SECONDS` 外部 generation 请求超时，默认 60；
- `DOCUMENT_WORKER_INTERVAL_SECONDS` 后台文档 worker 轮询间隔，默认 10；
- `DOCUMENT_WORKER_BATCH_SIZE` 后台文档 worker 每批认领任务数，默认 5；
- `LICENSE_PUBLIC_KEYS` License 验签公钥映射，支持 JSON：`{"default":"base64-ed25519-public-key"}`，或逗号分隔：`default=base64,key2=base64`；
- 密钥、Token、数据库密码必须通过环境变量注入；
- 外部请求建议传递 `X-Request-ID`，未传递时系统自动生成；
- 检索接口必须校验 `tenant_id + knowledge_base_id`，生产接口从登录态和 `X-Tenant-ID` 派生租户上下文；
- embedding 维度固定为 1536，外部 provider 返回非 1536 维会拒绝写入。
- 套餐 quota 会在文件上传、embedding run、RAG 问答和 Agent 会话执行前拦截；超额返回 `402` 和 `quota exceeded` 错误。

## 7. 下一步

```text
文件上传 → MinIO → document worker → 切片 → embedding run/provider → pgvector → RAG 问答生成
```

## 8. 服务器部署记录

当前已部署到服务器：

```text
目录：/opt/mu-agent-saas
服务：mu-agent-saas.service
Worker：mu-agent-document-worker.service
本机监听：127.0.0.1:18082
HTTPS 入口：https://zupu.jiangxinnet.com/saas-api/api/v1
```

常用命令：

```bash
systemctl status mu-agent-saas
systemctl status mu-agent-document-worker
systemctl restart mu-agent-saas
cd /opt/mu-agent-saas && docker compose ps
/opt/mu-agent-saas/scripts/smoke.sh
/opt/mu-agent-saas/scripts/backup.sh
```

线上依赖容器：

```text
mu-agent-saas-postgres 127.0.0.1:15432
mu-agent-saas-redis    127.0.0.1:16379
mu-agent-saas-minio    127.0.0.1:19000 / 19001
```

备份：

- systemd timer：`mu-agent-saas-backup.timer`
- 每日时间：`03:20`
- 保留目录：`/opt/mu-agent-saas/backups`
- 保留策略：删除 14 天前的 `.gz` 备份

Nginx：

- 站点配置：`/www/server/panel/vhost/nginx/zupu.jiangxinnet.com.conf`
- 已添加前缀反代：`/saas-api/ -> http://127.0.0.1:18082/`
- 修改前会保留 `zupu.jiangxinnet.com.conf.bak.*` 备份。

## 9. 生产部署模板

生产部署模板已固化到 `deploy/production/`：

```text
deploy/production/docker-compose.yml
deploy/production/compose.env.example
deploy/production/mu-agent-saas.env.example
deploy/production/systemd/
deploy/production/scripts/
deploy/production/nginx/saas-api-location.conf
scripts/build_release.ps1
scripts/build_release.sh
```

## 10. 2026-05-28 支付闭环增强

新增 migration：

```text
000014_payment_lifecycle_audit
```

新增接口：

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /api/v1/orders/:order_id/cancel | 取消当前租户 pending 订单，并关闭关联 pending 支付单 |
| POST | /api/v1/orders/:order_id/close | 关闭当前租户 pending 订单，并关闭关联 pending 支付单 |
| GET | /api/v1/payment-orders | 当前租户支付单列表，支持 business_order_id 过滤 |
| POST | /api/v1/payments/:payment_id/close | 关闭当前租户 pending/failed 支付单 |
| GET | /api/v1/payment-callback-events | 当前租户支付回调审计事件，支持 pay_no 过滤 |

约束：
- 已支付订单不允许取消或关闭。
- 已支付支付单不允许关闭。
- 支付回调写入 `payment_callback_events` 审计表；找不到支付单的失败回调也会记录事件。
- 所有新增查询和更新仍按当前 `tenant_id` 隔离。

注意：

- `*.example` 只放占位值；
- 真实 `JWT_SECRET`、数据库密码、MinIO 密码只放服务器环境文件；
- 不要把服务器真实 `.env`、私钥、Token 提交进交付包；
- 生产 Go 服务建议只监听 `127.0.0.1`，由 Nginx/宝塔对外提供 HTTPS。
## 2026-05-28 增量：审计日志筛选分页

`GET /api/v1/audit-logs` 已支持按 `action`、`resource_type`、`actor_user_id`、`from`、`to` 筛选，并通过 `cursor`/`next_cursor` 做稳定分页。接口继续复用统一 `pkg/response` 响应结构，并固定按当前 `X-Tenant-ID` 租户上下文隔离查询。

新增 migration：

```text
000017_audit_logs_filter_indexes
```

分页排序使用 `created_at DESC, id DESC`；`cursor` 由服务端生成，不包含密钥、token 或请求 body。

## 2026-05-28 增量：KB 文档详情与 Chunk 列表

新增接口：

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /api/v1/kbs/:kb_id/documents/:document_id | 查询当前租户当前 KB 下的单个文档详情，返回 `document` 与 `file_id` |
| GET | /api/v1/kbs/:kb_id/documents/:document_id/chunks | 查询当前租户当前 KB 下单个文档的 chunks，按 `chunk_no` 升序返回 |

约束：
- 两个接口都必须携带 `Authorization: Bearer <token>` 与 `X-Tenant-ID: <tenant_id>`；
- 服务端固定按 `tenant_id + kb_id + document_id` 校验，不接受客户端覆盖隔离字段；
- 本次只增加只读接口，无数据库结构变更，无新增 migration。

## 2026-05-29 增量：文件下载

新增接口：

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /api/v1/files/:file_id/download | 下载当前租户下已上传文件对象 |

约束：
- 必须携带 `Authorization: Bearer <token>` 与 `X-Tenant-ID: <tenant_id>`；
- 服务端固定按 `tenant_id + file_id` 查询文件记录，再读取对应对象存储 key；
- 跨租户或不存在的文件返回 404；
- 本次只增加只读下载接口，无数据库结构变更，无新增 migration。

## 2026-05-29 增量：KB 文档归档

新增接口：

| 方法 | 路径 | 说明 |
|---|---|---|
| DELETE | /api/v1/kbs/:kb_id/documents/:document_id | 归档当前租户当前 KB 下的文档 |

约束：
- 必须携带 `Authorization: Bearer <token>` 与 `X-Tenant-ID: <tenant_id>`；
- 服务端固定按 `tenant_id + kb_id + document_id` 软删除文档；
- 同事务软删除该文档 chunks 和关联 document_jobs；
- 原始上传文件记录和对象存储文件不删除，可继续用于其他文档任务；
- 本次无数据库结构变更，无新增 migration。
