# 智能体族谱SAAS 开发方案

版本：v1.2-final
更新时间：2026-05-31

## 1. 项目定位

智能体族谱SAAS 是面向企业、服务商和私有化客户的 AI Agent 平台，核心能力包括：

- 多租户 SaaS 管理；
- 智能体创建、派生、版本、发布与回滚；
- RAG 知识库与向量检索；
- 工具、插件、MCP 网关；
- 套餐、计费、用量、License 授权；
- 开发商后台、服务商后台、客户租户后台。

## 2. 技术基线

| 层级 | 技术 |
|---|---|
| 后端 | Go go1.26.3、Gin |
| 数据库 | PostgreSQL 16 + pgvector |
| 缓存/队列 | Redis 7.4 |
| 文件存储 | MinIO / S3 兼容存储 |
| 前端 | 免构建静态管理台已交付，Vue3 + TypeScript + Vite + Element Plus 作为后续工程化方向 |
| 移动端 | UniApp / H5 / 小程序预留 |
| 部署 | Docker Compose 起步，支持 Kubernetes |
| AI | Agent Runtime、RAG、Tool Schema、MCP、模型网关 |

## 3. 架构原则

1. **模块化单体优先**：MVP 阶段降低复杂度，后续可拆分微服务。
2. **租户强隔离**：核心业务表必须包含 `tenant_id`，检索必须同时校验 `tenant_id + knowledge_base_id`。
3. **商业可交付**：源码、SQL、API、部署、验收、授权文档完整保留。
4. **可审计**：登录、支付、授权、工具调用、Agent 执行都必须留痕。
5. **可扩展**：模型、插件、存储、支付、授权均采用接口化设计。
6. **生产安全**：生产关闭 debug，密钥只走环境变量或密钥管理。

## 4. 核心模块

| 模块 | 说明 | 优先级 |
|---|---|---|
| 基础工程 | 配置、日志、响应、错误码、健康检查、迁移 | P0 |
| 租户权限 | 租户、成员、角色、权限、审计 | P0 |
| Agent Runtime | 智能体、版本、会话、执行步骤、Memory | P0 |
| RAG 知识库 | 文件、文档、切片、embedding、向量/混合检索 | P0 |
| 套餐计费 | 套餐、订阅、额度、用量、订单 | P0 |
| License | 在线/离线授权、机器绑定、吊销、验签 | P0 |
| 插件工具 | Tool Schema、插件市场、MCP、调用审计 | P1 |
| 工作流 | 节点编排、条件、人工确认、执行日志 | P1 |
| 渠道入口 | Web/H5/小程序/公众号/企微等 | P1 |
| 监控运维 | 指标、日志、告警、备份、恢复 | P0 |

## 5. 后端目录规划

```text
01_backend/
├── cmd/server              # 服务入口
├── internal/bootstrap      # 应用装配
├── internal/config         # 配置
├── internal/middleware     # 中间件
├── internal/module         # 业务模块
├── pkg/database            # 数据库连接
├── pkg/response            # 统一响应
├── migrations              # SQL 迁移
└── deploy                  # 部署文件
```

## 6. 数据库设计要点

- PostgreSQL 16 + pgvector；
- 向量字段统一使用 `vector(1536)`；
- RAG chunk 使用 HNSW cosine 索引；
- 文档和 chunk 保留 SHA 去重字段；
- 高频查询增加组合索引；
- 关键表保留 `created_at / updated_at / deleted_at`；
- 生产迁移必须配套 down 回滚脚本。

核心表方向：

| 领域 | 核心表 |
|---|---|
| 租户权限 | tenants、users、tenant_members、roles、permissions |
| Agent | agents、agent_versions、conversations、messages、agent_runs |
| RAG | files、knowledge_bases、documents、document_chunks、embedding_jobs |
| 工具插件 | plugins、tools、tool_call_logs |
| 计费授权 | plans、subscriptions、usage_records、orders、licenses |
| 审计监控 | audit_logs、security_events、vector_search_logs |

## 7. RAG 检索方案

MVP 使用 PostgreSQL + pgvector 完成向量检索与混合检索。

检索链路：

1. 校验租户、知识库、向量维度；
2. 读取检索 Profile；
3. 设置 `hnsw.ef_search`；
4. 按 `tenant_id + knowledge_base_id` 过滤；
5. 执行向量召回或混合召回；
6. 按 `min_score` 过滤；
7. 返回 `top_k`；
8. 写入检索日志。

默认评分：

```text
score = vector_score * 0.7 + text_score * 0.3
```

## 8. Redis 设计要点

Key 规范：

```text
mu:{env}:{module}:{tenant_id}:{biz_key}
```

重点用途：

- Token 状态；
- 权限缓存；
- 套餐额度缓存；
- 会话短期上下文；
- RAG 检索缓存；
- 支付回调幂等；
- 分布式锁；
- embedding / usage 队列。

## 9. API 规范

统一前缀：

```text
/api/v1
```

统一响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {},
  "request_id": "trace-id"
}
```

核心已落地接口范围：

```text
GET  /api/v1/health
GET  /api/v1/ready
POST /api/v1/auth/register
POST /api/v1/auth/login
GET  /api/v1/tenants
GET  /api/v1/settings/rate-limit
GET  /api/v1/settings/runtime
GET  /api/v1/settings/monitoring
GET  /api/v1/settings/sensitive-fields
GET  /api/v1/settings/rate-limit-audit
POST /api/v1/kb/search/vector
POST /api/v1/kb/search/hybrid
POST /api/v1/kbs/{kb_id}/ask
POST /api/v1/agents/{agent_id}/chat
POST /api/v1/agents/{agent_id}/chat/stream
GET  /api/v1/agent-genealogy/graph
GET  /api/v1/analytics/summary
GET  /api/v1/webhooks
GET  /api/v1/webhook-deliveries
GET  /api/v1/licenses
GET  /api/v1/orders
GET  /api/v1/audit-logs
```

## 10. MVP 开发排期

| 周期 | 主题 | 验收结果 |
|---|---|---|
| 第 1 周 | 工程底座 | 本地启动、健康检查、迁移可用 |
| 第 2 周 | 租户权限 | 多租户隔离、RBAC 可用 |
| 第 3 周 | 套餐计费 | 套餐、额度、用量生效 |
| 第 4 周 | Agent Runtime | 可创建 Agent 并测试会话 |
| 第 5 周 | RAG 知识库 | 上传、切片、向量化、引用溯源跑通 |
| 第 6 周 | 插件工具 | Agent 可按权限调用工具 |
| 第 7 周 | 支付与 License | 支付/模拟支付、授权跑通 |
| 第 8 周 | 部署验收 | 演示环境、备份、监控、验收完成 |

## 11. 当前已交付状态

- SaaS 多租户、认证、角色权限、成员邀请和租户隔离已完成；
- 文件上传、知识库文档、切片、向量化、RAG 检索和问答闭环已完成；
- Agent 创建、编辑、发布、回滚、归档、知识库绑定、多轮会话和 SSE 流式对话已完成；
- Agent 工具安全策略、工具目录、Tool Schema 与参数校验、dry-run 测试、工具调用日志筛选分页与 CSV 导出，MCP 网关安全策略、服务目录和连通性 dry-run，以及插件市场启用/禁用（P1 插件工具）已完成；
- 工作流编排第一增量（节点类型目录、编排安全策略、图结构校验与拓扑执行顺序，P1 工作流）已完成；
- 工作流定义持久化（草稿/发布/归档 CRUD、写入图校验、版本号与租户隔离，P1 工作流）已完成；
- 工作流 dry-run 执行引擎与执行日志（拓扑模拟、human_approval 暂停、workflow_runs 记录，P1 工作流）已完成；
- Agent 渠道接入第一增量（渠道类型目录、Web/H5/API 接入点持久化与启用/禁用/归档，P1 渠道入口）已完成；
- 渠道接入代码生成（Web 嵌入脚本 / H5 链接 / API 调用示例，按渠道类型与请求 baseURL 生成，P1 渠道入口）已完成；
- 渠道公开连接端点（外部接入方凭 channel_key 拉取接入配置，公开 + IP 限流，P1 渠道入口）已完成；
- 智能体族谱图谱、关系维护、筛选、结构诊断、统计分析和 CSV 导出已完成；
- 套餐订阅、用量统计、额度限制、订单生命周期和支付回调验签已完成；
- License 在线/离线验证、签名脱敏和授权生命周期管理已完成；
- Webhook 配置、测试发送、投递记录、重试 Worker、状态摘要和 CSV 导出已完成；
- API 限流、审计筛选分页、审计 CSV 导出和安全摘要接口已完成；
- 生产部署、systemd/Nginx、备份、升级、回滚和恢复演练脚本已完成；
- 管理台已覆盖总览、知识库、Agent、族谱、License、用量订单、Webhook、审计和设置；
- `go test ./...`、`node --check 02_frontend/assets/app.js` 和 `git diff --check` 已通过；
- `02_docs_v1_delivery/06_交付验收清单.md` 已全部勾选。

## 12. 已跑通主链路

```text
文件上传 → MinIO → 文档解析 → 清洗切片 → embedding worker → pgvector 入库 → RAG 问答 → 引用溯源
```

```text
Agent 会话 → 历史编排 → KB 检索 → 生成模型 → SSE delta 输出 → 用量计费 → 审计记录
```

```text
订单支付 → 回调验签 → 订阅生效 → License 校验 → Webhook 投递 → 投递重试
```

```text
发布包 → 备份 → 升级 → 冒烟检查 → 回滚快照 → 恢复演练
```

后续增强建议：

1. 前端 Vue3/Vite 工程化迁移；
2. 真实第三方支付渠道接入；
3. 多实例生产环境 Redis 限流压测和监控告警增强；
4. 企业级自定义角色和插件市场扩展；
5. 工作流编排与人工确认节点。

## 13. 验收门禁

- [x] 生产环境关闭 debug；
- [x] 密钥、Token、License 私钥不入库；
- [x] 所有租户数据强制隔离；
- [x] 支付、授权、工具调用具备幂等；
- [x] RAG 检索记录日志并可追踪；
- [x] 数据库迁移有回滚脚本；
- [x] Docker Compose 可一键启动；
- [x] 备份恢复方案可执行；
- [x] 核心接口测试通过。

## 14. 交付物清单

| 文件/目录 | 说明 |
|---|---|
| 01_backend | 后端源码与迁移 |
| 02_docs_v1_delivery | v1.0 商业交付文档 |
| 03_智能体族谱SAAS_开发方案.md | 当前精简开发方案 |
| 04_最终交付验收清单.md | 最终交付验收清单 |
| README.md | 交付包说明 |
| manifest.json | 文件清单与摘要 |
