# 给 Codex 的开发启动指令

> 项目：智能体族谱SAAS  
> 指派人：木哥  
> 执行对象：Codex 开发助手  
> 当前状态：最终交付包已整理完成，进入后端 MVP 正式开发阶段。  
> 工作目录：`智能体族谱SAAS_最终交付包/01_backend`

## 1. 已启用规则清单

- 商业全栈交付大师
- 智能体 / Agent / RAG 框架规则
- PostgreSQL 16 + pgvector 规则
- Redis 缓存 / 限流 / 幂等规则
- SaaS 多租户 / 授权 / 审计规则
- 私有化交付 / 部署 / 安全规则
- Go / Gin 后端商业交付规则

## 2. 技术栈与默认假设

- 后端：Go + Gin
- 数据库：PostgreSQL 16 + pgvector
- 缓存：Redis 7
- 文件存储：MinIO / S3 兼容
- 向量维度：1536
- 向量检索：pgvector HNSW cosine
- 混合检索：向量检索 + 全文检索 + Trigram
- SaaS 隔离：必须保留并强化 `tenant_id` / `kb_id` 隔离
- 当前阶段：优先完成后端 MVP，不扩展前端

## 3. Codex 必读文件

请 Codex 开始编码前，按顺序阅读：

1. `README.md`
2. `03_智能体族谱SAAS_开发方案.md`
3. `04_最终交付验收清单.md`
4. `02_docs_v1_delivery/01_开发任务清单.md`
5. `02_docs_v1_delivery/03_数据库表结构_v1.sql`
6. `02_docs_v1_delivery/04_API接口清单.md`
7. `02_docs_v1_delivery/05_Redis_Key设计.md`
8. `01_backend/README.md`
9. `01_backend/向量存储与索引优化说明.md`
10. `给_聊天窗口_族谱开发_启动指令.md`

## 4. 当前代码基线

后端目录：

```text
01_backend/
```

已具备：

- Gin 服务骨架；
- `/healthz` 健康检查；
- `/readyz` 就绪检查；
- Request-ID 中间件；
- 生产 Release Mode；
- PostgreSQL 连接；
- KB 向量 / 混合检索基础代码；
- PostgreSQL HNSW cosine 索引迁移；
- 全文检索 / Trigram 检索迁移；
- tenant_id / kb_id 隔离字段；
- profile 字段；
- 1536 维 embedding 校验；
- 迁移脚本 up/down；
- 当前 `go test ./...` 已通过。

## 5. Codex 第一阶段开发任务

### P0-1：认证与租户基础

目标：完成 SaaS 后端最小可用鉴权链路。

需要实现：

- `internal/module/auth`
  - 用户注册；
  - 用户登录；
  - JWT 签发；
  - JWT 解析；
  - JWT 中间件；
  - 当前用户信息接口；
- `internal/module/tenant`
  - 租户创建；
  - 当前用户租户列表；
  - 当前租户上下文识别；
- 密码必须使用 bcrypt 或 argon2；
- JWT Secret 必须从环境变量读取，不允许硬编码；
- API 响应复用 `pkg/response`；
- 需要补齐必要 migration，包含 up/down。

建议接口：

```text
POST /api/v1/auth/register
POST /api/v1/auth/login
GET  /api/v1/auth/me
GET  /api/v1/tenants
POST /api/v1/tenants
```

### P0-2：KB 文档写入链路

目标：让知识库具备可写、可检索的 MVP 闭环。

需要实现：

- 创建 KB；
- KB 列表；
- 创建文档；
- 创建 chunk；
- 写入 embedding；
- 检索接口接入 tenant / kb 权限校验；
- embedding 维度必须校验为 1536；
- 所有写入接口必须绑定当前租户上下文。

建议接口：

```text
POST /api/v1/kbs
GET  /api/v1/kbs
POST /api/v1/kbs/:kb_id/documents
POST /api/v1/kbs/:kb_id/chunks
POST /api/v1/kbs/:kb_id/search
```

### P0-3：基础测试

需要覆盖：

- 配置加载；
- JWT 签发与解析；
- JWT 中间件基础行为；
- Request-ID；
- KB search 参数校验；
- embedding 维度校验；
- repository SQL 构建边界；
- tenant / kb 隔离关键路径。

验收命令：

```bash
cd 01_backend
go test ./...
```

## 6. 数据库与迁移要求

- 所有结构变更必须新增 migration 文件；
- migration 必须同时提供 `.up.sql` 和 `.down.sql`；
- 不允许直接修改历史 migration，除非确认为未发布基线且同步说明原因；
- 必须保留 PostgreSQL 16、pgvector、HNSW、全文检索、Trigram 相关能力；
- 新表必须考虑：
  - `id`；
  - `tenant_id`；
  - `created_at`；
  - `updated_at`；
  - 必要索引；
  - 软删除字段按需要设计。

## 7. 安全与交付约束

- 不允许把密钥、Token、数据库密码写死进代码；
- 不允许降低 `tenant_id` / `kb_id` 隔离；
- 不允许删除已有 migrations；
- 不允许引入不必要的大框架；
- 不允许破坏现有健康检查、就绪检查、Request-ID；
- 不允许跳过测试；
- 每完成一个 P0 任务，必须更新相关 README / API 文档 / 验收清单；
- 生产默认 Gin Release Mode，不打开 debug。

## 8. Codex 回传格式

Codex 完成后，请按以下格式回传：

```text
## 代码变更清单
- ...

## 新增 / 修改 migration
- ...

## 新增 / 修改 API
- ...

## 新增环境变量
- ...

## 测试结果
- 命令：go test ./...
- 结果：...

## 风险与注意事项
- ...

## 下一步建议
- ...
```

## 9. 给 Codex 的一键执行提示词

可以直接把下面这段发给 Codex：

```text
你是 Codex 开发助手。请进入 `智能体族谱SAAS_最终交付包/01_backend`，先阅读上级目录中的 `给_Codex_开发启动指令.md`、`README.md`、`03_智能体族谱SAAS_开发方案.md`、`02_docs_v1_delivery/04_API接口清单.md`，再基于现有 Go/Gin/PostgreSQL16/pgvector 项目开始开发。

优先级：
1. 先完成 P0-1 认证与租户基础：注册、登录、JWT、JWT 中间件、当前用户、租户创建、租户列表、当前租户上下文。
2. 再完成 P0-2 KB 文档写入链路：创建 KB、KB 列表、创建文档、创建 chunk、写入 1536 维 embedding、检索接口接入 tenant/kb 权限校验。
3. 最后完成 P0-3 基础测试：配置加载、JWT、Request-ID、KB 参数校验、embedding 维度校验、tenant/kb 隔离。

要求：所有数据库变更必须新增 migration up/down；不允许硬编码密钥；不允许降低 tenant_id/kb_id 隔离；所有响应复用 pkg/response；完成后更新 README/API/验收清单，并执行 `go test ./...`，按启动指令中的回传格式输出结果。
```
