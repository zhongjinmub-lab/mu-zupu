# 智能体族谱SAAS API 接口清单 v1.0

统一前缀：`/api/v1`  
鉴权方式：`Authorization: Bearer <token>`

## 1. 通用响应

```json
{
  "code": 0,
  "message": "ok",
  "data": {},
  "request_id": "trace-id"
}
```

## 2. 已落地接口

| 方法 | 路径 | 说明 | 鉴权 |
|---|---|---|---|
| GET | /health | 健康检查，包含数据库连通性 | public |
| GET | /ready | 就绪检查 | public |
| GET | /settings/sensitive-fields | 敏感字段保护与响应脱敏摘要 | user |
| GET | /settings/rate-limit-audit | API 限流与审计闭环摘要 | user |
| POST | /auth/register | 注册并签发 JWT | public |
| POST | /auth/login | 登录并签发 JWT | public |
| GET | /auth/me | 当前用户 | user |
| GET | /tenants | 当前用户租户列表 | user |
| POST | /tenants | 创建租户 | user |
| GET | /tenant/role-permissions | 当前租户角色权限矩阵摘要 | tenant |
| GET | /audit-logs/export | 导出当前租户审计日志 CSV，支持筛选条件 | tenant |
| POST | /agents | 创建 Agent | tenant |
| GET | /agents | 当前租户 Agent 列表 | tenant |
| GET | /agents/{agent_id} | Agent 详情 | tenant |
| PUT | /agents/{agent_id} | 更新 Agent，已发布 Agent 更新后回到 draft | tenant |
| POST | /agents/{agent_id}/publish | 发布 Agent | tenant |
| POST | /agents/{agent_id}/rollback | 回滚 Agent 到 draft | tenant |
| POST | /agents/{agent_id}/test-chat | Agent 测试会话，基于绑定 KB 检索并生成回答 | tenant |
| POST | /agents/{agent_id}/chat | Agent 多轮会话，支持 conversation_id 续聊 | tenant |
| GET | /agents/{agent_id}/conversations | Agent 会话列表 | tenant |
| GET | /agents/{agent_id}/conversations/{conversation_id}/messages | 会话消息列表 | tenant |
| GET | /agents/tool-safety-policy | Agent 工具安全策略摘要 | tenant |
| GET | /agents/conversation-orchestration-policy | Agent 多轮会话编排策略摘要 | tenant |
| DELETE | /agents/{agent_id} | 归档 Agent | tenant |
| POST | /agents/{agent_id}/knowledge-bases | 绑定当前租户 KB | tenant |
| GET | /agents/{agent_id}/knowledge-bases | Agent 已绑定 KB 列表 | tenant |
| DELETE | /agents/{agent_id}/knowledge-bases/{kb_id} | 解绑 Agent KB | tenant |
| GET | /billing/plans | 可用套餐列表 | tenant |
| GET | /billing/subscription | 当前租户订阅 | tenant |
| GET | /billing/usage/summary | 当前租户用量汇总 | tenant |
| GET | /billing/quota/status | 当前租户套餐额度状态 | tenant |
| POST | /orders | 创建当前租户业务订单 | tenant |
| GET | /orders | 当前租户订单列表 | tenant |
| POST | /payment-orders | 创建 mock 支付单 | tenant |
| POST | /payments/{payment_id}/query | 查询当前租户支付单 | tenant |
| POST | /payment-callbacks/{channel} | mock 支付回调 | tenant |
| GET | /analytics/summary | 当前租户统计汇总，包含资源、经营、智能体族谱、用量趋势、最近操作和风险 | tenant |
| GET | /analytics/summary/export | 导出当前租户统计汇总 CSV，包含资源、经营、智能体族谱、用量趋势、风险和最近操作 | tenant |
| GET | /licenses | 当前租户 License 列表 | tenant |
| POST | /licenses | 创建当前租户 License | tenant |
| POST | /licenses/{license_id}/verify | 验证当前租户 License 离线签名和状态 | tenant |
| POST | /licenses/{license_id}/activate | 激活当前租户 License | tenant |
| POST | /licenses/{license_id}/revoke | 吊销当前租户 License | tenant |
| POST | /files/upload | 上传文件到 MinIO/S3 | tenant |
| GET | /files | 当前租户文件列表 | tenant |
| GET | /kbs | 当前租户知识库列表 | tenant |
| POST | /kbs | 创建知识库 | tenant |
| POST | /kbs/{kb_id}/documents | 创建文档 | tenant |
| POST | /kbs/{kb_id}/documents/from-file | 从已上传文本文件生成文档和切片 | tenant |
| POST | /kbs/{kb_id}/documents/{document_id}/rebuild | 重建 file-backed 文档切片 | tenant |
| POST | /kbs/{kb_id}/document-jobs | 创建文档解析/切片任务 | tenant |
| GET | /kbs/{kb_id}/document-jobs | 查询文档任务列表 | tenant |
| POST | /kbs/{kb_id}/document-jobs/run | 同步执行 pending/failed 文档任务 | tenant |
| POST | /kbs/{kb_id}/chunks | 创建 chunk 并写入 embedding | tenant |
| GET | /kbs/{kb_id}/chunks/pending | 查询待向量化 chunks | tenant |
| PUT | /kbs/{kb_id}/chunks/{chunk_id}/embedding | 写回 chunk embedding | tenant |
| POST | /kbs/{kb_id}/embedding/run | 使用当前 embedding provider 同步处理 pending chunks | tenant |
| POST | /kbs/{kb_id}/search | 当前租户知识库检索 | tenant |
| POST | /kbs/{kb_id}/ask | RAG 问答：问题向量化、混合检索、生成回答并返回引用 | tenant |
| POST | /kb/search/vector | 知识库向量检索 | tenant |
| POST | /kb/search/hybrid | 知识库混合检索 | tenant |

## 3. MVP 规划接口

### 3.1 认证与用户

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /auth/register | 注册 |
| POST | /auth/login | 登录 |
| POST | /auth/refresh | 刷新 Token |
| POST | /auth/logout | 退出 |
| GET | /auth/me | 当前用户 |
| GET | /users | 用户列表 |
| POST | /users | 创建用户 |

### 3.2 租户与权限

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /tenants | 租户列表 |
| POST | /tenants | 创建租户 |
| GET | /tenant/members | 当前租户成员列表 |
| POST | /tenant/members | 添加当前租户成员 |
| PUT | /tenant/members/{member_id}/role | 调整当前租户成员角色 |
| DELETE | /tenant/members/{member_id} | 移除当前租户成员 |
| GET | /tenant/role-permissions | 当前租户角色权限矩阵摘要 |

当前角色模型为固定 SaaS 租户角色：`owner`、`admin`、`member`、`viewer`。`owner/admin` 可管理成员、账单授权和 Webhook；`member` 可维护知识库、文件和智能体；`viewer` 仅可查看。管理台“角色权限矩阵”会以中文摘要展示当前角色、可读/可写/可管理能力、权限范围和受限操作，不展示黑色 JSON 原文。

### 3.3 Agent

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /agents | 智能体列表 |
| POST | /agents | 创建智能体 |
| GET | /agents/{id} | 智能体详情 |
| PUT | /agents/{id} | 更新智能体 |
| POST | /agents/{id}/publish | 发布版本 |
| POST | /agents/{id}/rollback | 回滚版本 |
| POST | /agents/{id}/test-chat | 测试会话 |
| POST | /agents/{id}/chat | 多轮会话 |
| GET | /agents/{id}/conversations | 会话列表 |
| GET | /agents/{id}/conversations/{conversation_id}/messages | 消息列表 |

当前已落地 Agent 基础管理、Agent-KB 绑定和测试会话接口：

```json
{
  "name": "族谱问答助手",
  "code": "genealogy_qa",
  "description": "面向族谱知识库的问答 Agent",
  "system_prompt": "仅基于绑定知识库回答。",
  "model_config": {"model": "local-rag"},
  "tool_policy": {},
  "memory_policy": {}
}
```

绑定知识库：

```json
{
  "knowledge_base_id": "uuid",
  "metadata": {"priority": 1}
}
```

绑定前会校验当前 `X-Tenant-ID` 对 `knowledge_base_id` 有访问权限，不能跨租户绑定。

测试会话：

```json
{
  "message": "这个族谱知识库里有哪些关键信息？",
  "knowledge_base_id": "uuid",
  "top_k": 5,
  "candidate_k": 25,
  "min_score": 0.2,
  "max_tokens": 1024,
  "temperature": 0.2
}
```

`knowledge_base_id` 可省略，省略时使用 Agent 第一个 active 绑定 KB。接口会创建 `conversations` 和两条 `messages`，并返回回答、引用片段和模型信息。

多轮会话：

```json
{
  "conversation_id": "可选，传入时续聊，不传时自动创建",
  "message": "继续解释上一轮提到的审批规则",
  "knowledge_base_id": "可选，省略时使用第一个 active 绑定 KB",
  "history_limit": 20,
  "top_k": 5,
  "candidate_k": 25,
  "min_score": 0
}
```

`POST /agents/{id}/chat` 会校验当前租户对 Agent、会话和绑定 KB 的访问权限，读取最近历史消息参与生成，写入 user/assistant 两条消息，并返回 `conversation_id`、消息 ID、回答、引用片段与 `history_used`。会话和消息查询接口均按 `tenant_id + agent_id + conversation_id` 隔离。
`GET /agents/conversation-orchestration-policy` 返回当前多轮会话编排摘要：默认历史 20 条、最大历史 50 条、RAG 检索、SSE 流式事件、套餐额度指标和工具默认拒绝策略。管理台多轮会话区会用中文摘要展示编排流程，不展示黑色 JSON 原文。

### 3.4 知识库

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /files/upload | 上传文件 |
| GET | /files | 文件列表 |
| GET | /knowledge-bases | 知识库列表 |
| POST | /knowledge-bases | 创建知识库 |
| POST | /knowledge-bases/{id}/documents | 添加文档 |
| POST | /documents/{id}/rebuild | 重建索引 |
| POST | /knowledge-bases/{id}/search | 知识检索 |

### 3.5 工具插件

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /plugins | 插件列表 |
| POST | /plugins/{id}/enable | 启用插件 |
| POST | /plugins/{id}/disable | 禁用插件 |
| GET | /tools | 工具列表 |
| POST | /tools/{id}/test | 测试工具 |
| GET | /tool-call-logs | 工具调用日志 |

当前 Agent 工具执行首版采用安全默认策略：`GET /agents/tool-safety-policy` 返回工具安全摘要，默认 `enabled=false`、`default_action=deny`。知识库检索和文件资料查询仅作为计划中的只读工具展示；知识库写入、账单授权等危险工具默认阻断，后续启用前必须校验 `tenant writer/admin` 权限、要求人工确认，并写入 `agent.tool.call` 审计动作。管理台“Agent 工具安全策略”用中文卡片展示工具状态、危险确认、审计动作和后续执行要求，不展示黑色 JSON 原文。

### 3.6 订单与授权

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /orders | 创建业务订单 |
| GET | /orders | 订单列表 |
| POST | /payment-orders | 创建支付单 |
| POST | /payments/{channel}/callback | 支付回调 |
| POST | /payments/{id}/query | 主动查单 |
| GET | /licenses | License 列表 |
| POST | /licenses | 创建 License |
| POST | /licenses/{id}/activate | 激活 License |
| POST | /licenses/{id}/revoke | 吊销 License |

## 4. 检索请求要求

## 3.7 套餐与用量

当前已落地 MVP：

- `GET /billing/plans`：返回 active 套餐，默认包含 `free` 套餐；
- `GET /billing/subscription`：返回当前租户 active 订阅；若无订阅，自动创建 `free` 订阅；
- `GET /billing/usage/summary?from=2026-05-01&to=2026-06-01`：按 metric 汇总当前租户用量。
- `GET /billing/quota/status`：返回当前租户 active 订阅下各关键指标的额度状态，包含套餐编码、指标名称、上限、已用、剩余、是否限制和是否允许继续使用。

已自动记录的 metric：

| metric | 触发点 | unit |
|---|---|---|
| `file_upload_bytes` | 文件上传成功 | bytes |
| `embedding_chunks` | `/kbs/{kb_id}/embedding/run` 成功处理 chunk | chunks |
| `rag_requests` | `/kbs/{kb_id}/ask` 成功 | requests |
| `agent_messages` | `/agents/{id}/test-chat` 和 `/agents/{id}/chat` 成功 | messages |

用量记录按 `tenant_id` 隔离，并保留 `subject_type`、`subject_id`、`request_id` 和 metadata，便于后续接入套餐额度拦截、计费和审计。

套餐 quota 已接入硬限制 MVP：文件上传、`/kbs/{kb_id}/embedding/run`、`/kbs/{kb_id}/ask`、`/agents/{id}/test-chat` 和 `/agents/{id}/chat` 会在执行前检查当前 active 订阅的 quota；超过配额时返回 `402`，响应 message 以 `quota exceeded` 开头。
管理台“套餐额度状态”会调用 `/billing/quota/status`，用中文卡片展示 RAG 问答、Agent 消息、文件上传容量和知识切片向量化的已用、上限、剩余和使用率，不展示黑色 JSON 原文。

订单/支付 MVP 已接入 mock 通道：

- `POST /orders`：按 `plan_code` 创建业务订单；
- `GET /orders`：列出当前租户订单；
- `POST /payment-orders`：为 pending 业务订单创建 mock 支付单；
- `POST /payments/{payment_id}/query`：查询当前租户支付单；
- `POST /payment-callbacks/mock`：按 `pay_no` 幂等更新支付单，支付成功后将业务订单置为 `paid` 并创建订阅。
- 支付回调支持服务端验签：未配置 `PAYMENT_CALLBACK_SECRET` 时保留 mock 开发体验；配置后必须对原始 JSON 请求体使用 HMAC-SHA256 计算签名，并在 `X-Payment-Signature` 传入 `sha256=<hex>`，签名失败返回中文错误且不更新支付单。

## 3.8 License 授权

当前已落地 License 生命周期 MVP：

- `GET /licenses`：返回当前租户 License 列表；
- `POST /licenses`：创建 License，`license_no` 可省略，服务端自动生成；
- `POST /licenses/{license_id}/verify`：验证 License 的 Ed25519 离线签名、过期和吊销状态；
- `POST /licenses/{license_id}/activate`：仅激活当前租户未吊销、未过期的 License；
- `POST /licenses/{license_id}/revoke`：仅吊销当前租户 License。

创建请求示例：

```json
{
  "license_type": "tenant",
  "subject": {"tenant_name": "示例租户"},
  "limits": {"agent_messages": 10000, "rag_requests": 10000},
  "expired_at": "2026-12-31T23:59:59+08:00"
}
```

License 查询和状态变更均按 `tenant_id + license_id` 隔离；`public_key_id` 和服务端保存的签名使用 `LICENSE_PUBLIC_KEYS` 中配置的 Ed25519 公钥验签，激活带签名 License 前会强制验签。接口响应不返回 `signature` 原文，只返回 `has_signature`。`POST /licenses/{license_id}/verify` 会返回 `mode`、`status`、`has_signature` 和中文可展示摘要所需字段：普通 `tenant/trial` License 走在线状态校验，`offline` 或带签名的 License 走离线验签。

## 4. 检索请求要求

租户级接口必须包含：

- `Authorization: Bearer <token>`
- `X-Tenant-ID: <tenant_id>`

向量检索和混合检索必须包含 `embedding`，维度必须为 1536。生产环境必须从登录态或服务端上下文获取 `tenant_id`，避免客户端伪造；`/kbs/{kb_id}/search` 从路径获取 `knowledge_base_id` 并校验当前租户权限。

## 5. Embedding Provider 配置

`/kbs/{kb_id}/embedding/run` 使用服务端环境变量配置的 provider：

- `EMBEDDING_PROVIDER=local`：本地 deterministic provider，默认 `local-hash-1536`，用于 MVP 离线闭环；
- `EMBEDDING_PROVIDER=openai_compatible` 或 `http`：请求 `EMBEDDING_BASE_URL + /embeddings`；
- `EMBEDDING_MODEL`：外部请求 body 的 `model`；
- `EMBEDDING_API_KEY`：通过 `Authorization: Bearer` 发送；
- `EMBEDDING_TIMEOUT_SECONDS`：请求超时，默认 30 秒。

所有 provider 返回的 embedding 维度都必须是 1536，否则拒绝写入 chunk。

## 6. RAG 问答生成

`POST /kbs/{kb_id}/ask` 使用当前租户上下文和路径中的 `kb_id` 做权限校验，然后执行：

1. 使用服务端 `EMBEDDING_*` provider 将 `question` 转为 1536 维向量；
2. 在当前 tenant/kb 内执行混合检索；
3. 使用服务端 `GENERATION_*` provider 生成回答；
4. 返回 `answer`、`references`、检索参数和模型信息。

请求示例：

```json
{
  "question": "这份知识库讲了什么？",
  "top_k": 5,
  "candidate_k": 25,
  "min_score": 0.2,
  "max_tokens": 1024,
  "temperature": 0.2
}
```

响应中的 `references` 包含命中的 `chunk_id`、`document_id`、`title`、`content` 和 `score`，便于前端展示引用来源。

Generation provider 配置：

- `GENERATION_PROVIDER=local`：本地回答草稿 provider，用于离线闭环验证；
- `GENERATION_PROVIDER=openai_compatible` 或 `http`：请求 `GENERATION_BASE_URL + /chat/completions`；
- `GENERATION_MODEL`：外部请求 body 的 `model`；
- `GENERATION_API_KEY`：通过 `Authorization: Bearer` 发送；
- `GENERATION_TIMEOUT_SECONDS`：请求超时，默认 60 秒。

## 7. 文档切片重建

`POST /kbs/{kb_id}/documents/{document_id}/rebuild` 仅支持由文件生成的文档：

- 必须校验当前登录用户、`X-Tenant-ID`、`kb_id`、`document_id` 同属一个租户；
- 读取原文件对象重新切片；
- 旧 chunks 软删除，新 chunks 从 `chunk_no=1` 重新生成；
- 新 chunks 的 `embedding_status=pending`，后续通过 `/kbs/{kb_id}/embedding/run` 写入 embedding。

## 8. 文档任务队列

`document_jobs` 是数据库任务队列，当前同时提供 API 同步 worker run 和后台常驻 `mu-agent-document-worker`：

- `POST /kbs/{kb_id}/document-jobs`：创建任务；
- `GET /kbs/{kb_id}/document-jobs?limit=50`：查询任务；
- `POST /kbs/{kb_id}/document-jobs/run`：认领并执行当前 KB 下 pending/failed 任务。

创建任务请求示例：

```json
{
  "file_id": "uuid",
  "job_type": "parse_chunk",
  "title": "文档标题",
  "max_chars": 1200,
  "overlap_chars": 120
}
```

`job_type=rebuild` 时必须传 `document_id`，且该文档必须是同租户同 KB 下由同一个 `file_id` 生成的文档。任务执行成功后会生成或重建 chunks，embedding 仍通过 `/kbs/{kb_id}/embedding/run` 写入。后台 worker 通过 `DOCUMENT_WORKER_INTERVAL_SECONDS` 和 `DOCUMENT_WORKER_BATCH_SIZE` 控制轮询频率与批大小。
## 2026-05-28 增量：审计日志筛选分页

`GET /audit-logs` 已支持在当前租户上下文内筛选与 cursor 分页。

| 参数 | 说明 |
|---|---|
| `action` | 审计动作，例如 `http.post` |
| `resource_type` | 资源类型，例如 `http_request` |
| `actor_user_id` | 操作者用户 UUID |
| `from` | 起始时间，RFC3339/RFC3339Nano 或 `YYYY-MM-DD` |
| `to` | 结束时间，RFC3339/RFC3339Nano 或 `YYYY-MM-DD` |
| `limit` | 每页数量，默认 50，最大 100 |
| `cursor` | 下一页游标，使用上一页响应的 `next_cursor` |

响应 `data`：

```json
{
  "items": [],
  "next_cursor": ""
}
```

该接口必须携带 `Authorization: Bearer <token>` 和 `X-Tenant-ID: <tenant_id>`；服务端固定按当前租户隔离查询，不接受客户端覆盖 `tenant_id`。

## 2026-05-28 增量：KB 文档详情与 Chunk 列表

| 方法 | 路径 | 说明 | 权限 |
|---|---|---|---|
| GET | /kbs/{kb_id}/documents/{document_id} | 查询当前租户当前 KB 下的单个文档详情，返回 `document` 与 `file_id` | tenant |
| GET | /kbs/{kb_id}/documents/{document_id}/chunks | 查询当前租户当前 KB 下单个文档的 chunks，按 `chunk_no` 升序返回 | tenant |

两个接口都复用 `pkg/response`，并固定按 `tenant_id + kb_id + document_id` 校验隔离；跨租户、跨 KB 或不存在的文档返回 404。

## 2026-05-29 增量：文件下载

| 方法 | 路径 | 说明 | 权限 |
|---|---|---|---|
| GET | /files/{file_id}/download | 下载当前租户下已上传文件对象 | tenant |

接口按 `tenant_id + file_id` 查询文件记录，并从对象存储读取 `object_key` 返回二进制流；跨租户或不存在的文件返回 404。

## 2026-05-29 增量：KB 文档归档

| 方法 | 路径 | 说明 | 权限 |
|---|---|---|---|
| DELETE | /kbs/{kb_id}/documents/{document_id} | 归档当前租户当前 KB 下的文档 | tenant writer |

接口按 `tenant_id + kb_id + document_id` 软删除文档；同事务软删除该文档 chunks 和关联 document_jobs。原始上传文件记录与对象存储文件不删除。

## 2026-05-29 增量：Agent 多轮会话管理台

管理台已接入已落地的 Agent 多轮会话接口：

| 方法 | 路径 | 说明 | 权限 |
|---|---|---|---|
| POST | /agents/{agent_id}/chat | 发送多轮会话消息，支持 `conversation_id` 续聊 | tenant writer |
| GET | /agents/{agent_id}/conversations | 查询当前租户当前 Agent 会话列表 | tenant |
| GET | /agents/{agent_id}/conversations/{conversation_id}/messages | 查询当前租户当前 Agent 当前会话消息 | tenant |

前端发送成功后保留返回的 `conversation_id` 用于续聊，并刷新会话列表、消息列表和用量汇总。

## 2026-05-29 增量：Agent 知识库绑定管理台

管理台已接入已落地的 Agent-KB 绑定接口：

| 方法 | 路径 | 说明 | 权限 |
|---|---|---|---|
| POST | /agents/{agent_id}/knowledge-bases | 绑定当前租户 KB 到当前 Agent | tenant writer |
| GET | /agents/{agent_id}/knowledge-bases | 查询当前 Agent 已绑定 KB 列表 | tenant |
| DELETE | /agents/{agent_id}/knowledge-bases/{kb_id} | 解绑当前 Agent KB | tenant writer |

绑定前后端会校验当前租户对 KB 的访问权限；解绑按 `tenant_id + agent_id + kb_id` 隔离。

## 2026-05-29 增量：Agent 归档管理台

管理台已接入已落地的 Agent 归档接口：

| 方法 | 路径 | 说明 | 权限 |
|---|---|---|---|
| DELETE | /agents/{agent_id} | 归档当前租户 Agent | tenant writer |

前端归档前会要求确认；归档成功后刷新 Agent 列表，并清空/刷新当前绑定关系、会话列表和消息列表。

## 2026-05-29 增量：Agent 编辑管理台

管理台已接入已落地的 Agent 更新接口：

| 方法 | 路径 | 说明 | 权限 |
|---|---|---|---|
| PUT | /agents/{agent_id} | 更新当前租户 Agent 名称、描述、系统提示词和策略配置 | tenant writer |

前端点击“编辑”后回填 Agent 表单；编码只读，提交保存后刷新 Agent 列表。已发布 Agent 更新后由后端回到 `draft`。

## 2026-05-31 增量：可选 Redis API 限流

后端限流中间件已支持内存与 Redis 两种后端，默认保持内存限流，便于本地开发和单实例部署；生产多实例可切换到 Redis。

| 配置项 | 默认值 | 说明 |
|---|---:|---|
| `RATE_LIMIT_BACKEND` | `memory` | 可选 `memory` 或 `redis` |
| `RATE_LIMIT_WINDOW_SECONDS` | `60` | 固定窗口长度，单位秒 |
| `RATE_LIMIT_TENANT_PER_MINUTE` | `120` | 同一租户窗口内最大请求数 |
| `RATE_LIMIT_USER_PER_MINUTE` | `60` | 同一用户窗口内最大请求数 |
| `RATE_LIMIT_AUTH_IP_PER_MINUTE` | `20` | 登录/注册同一 IP 窗口内最大请求数 |

启用 `RATE_LIMIT_BACKEND=redis` 后使用已有 `REDIS_ADDR`、`REDIS_PASS`、`REDIS_DB` 连接 Redis。Redis 计数异常时会自动回退到内存限流，避免 Redis 短时不可用导致业务接口整体不可用。

触发限流时返回 HTTP `429`，响应仍使用统一结构，中文错误信息为“请求过于频繁，请稍后再试”。

## 2026-05-31 增量：Webhook 投递记录筛选

`GET /webhook-deliveries` 已支持在当前租户上下文内筛选投递记录。

| 参数 | 说明 |
|---|---|
| `endpoint_id` | Webhook 配置 ID，可选，必须是合法 UUID |
| `event_type` | 事件类型，可选，例如 `order.paid`、`license.activated` |
| `status` | 投递状态，可选，支持 `success`、`failed` |
| `from` | 创建时间起点，可选，支持 RFC3339 或 `YYYY-MM-DD` |
| `to` | 创建时间终点，可选，支持 RFC3339 或 `YYYY-MM-DD` |
| `limit` | 返回数量，默认 50，最大 200 |

管理台 Webhook 页面已增加投递记录筛选栏，可按 Webhook、事件类型、状态和数量查看中文摘要结果。

## 2026-05-31 增量：Webhook 单条投递手动重试

`POST /webhook-deliveries/{delivery_id}/retry` 已支持对当前租户下的失败投递记录立即重试。

| 项目 | 说明 |
|---|---|
| 权限 | tenant admin |
| `delivery_id` | 投递记录 ID，必须是合法 UUID |
| 可重试状态 | 仅 `failed` 投递可重试 |
| 结果写入 | 重试后更新原投递记录的状态、HTTP 状态、响应摘要、错误信息、耗时、重试次数和下一次重试时间 |

管理台 Webhook 投递记录中，失败记录会显示“立即重试”按钮；重试后按当前筛选条件刷新列表，并显示中文结果提示。

## 2026-05-31 增量：Agent SSE 流式体验加固

`POST /agents/{agent_id}/chat/stream` 继续使用 `text/event-stream`，事件类型保持 `start`、`reference`、`delta`、`done`、`error`。

本轮补强点：

- 后端流式分段按 rune 边界切分，避免中文内容被截断成乱码。
- 空回答仍会输出一个空 `delta`，保持前端协议处理稳定。
- 管理台遇到 SSE `error` 事件时，会保留已输出的流式内容，并在结果区显示中文错误摘要。
- 流式失败时不会清空输入框，也不会误刷新会话和用量。

## 2026-05-31 增量：限流策略只读接口

`GET /settings/rate-limit` 用于返回当前后端生效的 API 限流策略摘要，供管理台设置页只读展示。

| 项目 | 说明 |
|---|---|
| 权限 | user，需携带 `Authorization: Bearer <token>` |
| 租户头 | 不强制 `X-Tenant-ID`，该接口返回全局运行策略摘要 |
| 敏感信息 | 不返回 `REDIS_ADDR`、`REDIS_PASS`、数据库连接串或任何密钥 |

响应 `data` 示例：

```json
{
  "backend": "memory",
  "window_seconds": 60,
  "tenant_per_window": 120,
  "user_per_window": 60,
  "auth_ip_per_window": 20,
  "redis_enabled": false,
  "redis_fallback_label": "当前使用内存限流，适合单实例或本地环境",
  "scoped_policies": [
    {
      "scope": "auth_ip",
      "name": "登录注册",
      "route_pattern": "POST /auth/login, POST /auth/register",
      "key_strategy": "ip",
      "ip_per_window": 20
    },
    {
      "scope": "webhook_test",
      "name": "Webhook 测试发送",
      "route_pattern": "POST /webhooks/{webhook_id}/test",
      "key_strategy": "tenant_id + user_id + scope",
      "tenant_per_window": 60,
      "user_per_window": 30
    },
    {
      "scope": "agent_stream",
      "name": "Agent SSE 流式对话",
      "route_pattern": "POST /agents/{agent_id}/chat/stream",
      "key_strategy": "tenant_id + user_id + scope",
      "tenant_per_window": 40,
      "user_per_window": 20
    }
  ]
}
```

`scoped_policies` 表示按接口分组的差异化限流策略。当前已实际挂载独立 scope 的接口包括：

- `POST /webhooks/{webhook_id}/test`：Webhook 测试发送，按 `webhook_test` scope 计数。
- `POST /agents/{agent_id}/chat/stream`：Agent SSE 流式对话，按 `agent_stream` scope 计数。

管理台设置页已从静态说明改为读取该接口动态展示，展示内容为中文摘要，包含默认策略和分组策略，不展示黑色 JSON 原文区域。

## 2026-05-31 增量：Webhook 投递状态摘要

`GET /webhook-deliveries/summary` 已支持查询当前租户下 Webhook 投递聚合状态。

| 项目 | 说明 |
|---|---|
| 权限 | tenant |
| 租户隔离 | 固定使用当前 `X-Tenant-ID`，不接受客户端覆盖 `tenant_id` |
| 返回内容 | 总投递、成功、失败、已排队重试、已到期重试、需要人工处理、最近尝试时间 |

支持查询参数：

| 参数 | 说明 |
|---|---|
| `endpoint_id` | Webhook 配置 ID，可选，必须是合法 UUID |
| `event_type` | 事件类型，可选，例如 `webhook.test`、`order.paid` |
| `status` | 投递状态，可选，支持 `success`、`failed` |
| `from` | 创建时间起点，可选，支持 RFC3339 或 `YYYY-MM-DD` |
| `to` | 创建时间终点，可选，支持 RFC3339 或 `YYYY-MM-DD` |

响应 `data` 示例：

```json
{
  "total": 12,
  "success": 9,
  "failed": 3,
  "retry_scheduled": 2,
  "retry_due": 1,
  "manual_review": 1,
  "last_attempt_at": "2026-05-31T10:20:30Z"
}
```

管理台 Webhook 页面“投递状态摘要”会复用当前筛选条件，和投递记录列表、CSV 导出保持同一统计口径。

## 2026-05-31 增量：运行配置只读摘要

`GET /settings/runtime` 用于返回当前后端运行配置摘要，供管理台设置页只读展示。

| 项目 | 说明 |
|---|---|
| 权限 | user，需携带 `Authorization: Bearer <token>` |
| 租户头 | 不强制 `X-Tenant-ID`，该接口返回全局运行配置摘要 |
| 敏感信息 | 不返回数据库 DSN、Redis 地址、MinIO 密钥、模型 API Key、JWT 密钥或 License 公钥原文 |

响应 `data` 示例：

```json
{
  "env": "dev",
  "upload_max_mb": 50,
  "storage_mode": "s3/minio http",
  "storage_public_enabled": false,
  "embedding_provider": "local",
  "embedding_model": "local-hash-1536",
  "embedding_external_configured": false,
  "generation_provider": "local",
  "generation_model": "local-rag",
  "generation_external_configured": false,
  "document_worker_interval_seconds": 10,
  "document_worker_batch_size": 5,
  "webhook_worker_interval_seconds": 15,
  "webhook_worker_batch_size": 20,
  "webhook_max_retries": 3,
  "webhook_retry_base_seconds": 60
}
```

管理台设置页已新增“运行配置摘要”，用中文卡片展示环境、上传上限、对象存储模式、模型 Provider、文档 Worker、Webhook Worker 和重试策略，不展示黑色 JSON 原文区域。

## 2026-05-31 增量：运行监控只读摘要

`GET /settings/monitoring` 用于返回当前后端进程的运行监控快照，供管理台设置页只读展示。

| 项目 | 说明 |
|---|---|
| 权限 | user，需携带 `Authorization: Bearer <token>` |
| 租户头 | 不强制 `X-Tenant-ID`，该接口返回当前服务进程指标 |
| 敏感信息 | 不返回数据库 DSN、Redis 地址、MinIO 密钥、模型 API Key、JWT 密钥或 License 公钥原文 |

响应 `data` 字段：

| 字段 | 说明 |
|---|---|
| `status` | 进程状态，当前为 `ok` |
| `checked_at` | 指标采集时间 |
| `uptime_seconds` | 进程启动至今秒数 |
| `goroutines` | 当前 goroutine 数量 |
| `heap_alloc_mb` | 当前堆内存已分配 MB |
| `heap_sys_mb` | 堆内存向系统申请 MB |
| `heap_objects` | 当前堆对象数量 |
| `gc_count` | 累计 GC 次数 |
| `last_gc_ago_seconds` | 距离最近一次 GC 的秒数；无 GC 记录时为 `-1` |

管理台设置页已新增“运行监控摘要”，用中文卡片展示进程状态、启动时长、goroutine、堆内存、堆对象和 GC 状态，不展示黑色 JSON 原文区域。

## 2026-05-31 增量：敏感字段保护摘要

`GET /settings/sensitive-fields` 用于返回当前后端敏感字段保护和响应脱敏策略摘要，供管理台设置页只读展示。

| 项目 | 说明 |
|---|---|
| 权限 | user，需携带 `Authorization: Bearer <token>` |
| 租户头 | 不强制 `X-Tenant-ID`，该接口返回全局安全策略摘要 |
| 敏感信息 | 不返回真实密钥、Token、数据库 DSN、Redis 密码、对象存储密钥、模型 API Key、Webhook secret 或 License signature |

响应 `data` 字段：

| 字段 | 说明 |
|---|---|
| `environment_secrets` | 环境变量注入类密钥摘要，例如 JWT、数据库连接串、对象存储、模型 API Key 和支付回调验签密钥 |
| `stored_secrets` | 数据库存储类敏感字段摘要，例如密码哈希、邀请 token_hash、Webhook secret 和 License signature |
| `response_redactions` | 已脱敏响应摘要，例如运行配置、限流策略和运行监控接口 |
| `operational_notes` | 运维注意事项，说明生产密钥注入、布尔摘要展示和后续 KMS 加密扩展边界 |

管理台设置页已新增“敏感字段保护摘要”，用中文卡片展示环境变量隔离、哈希/内部存储和响应脱敏状态，不展示黑色 JSON 原文区域。租户邀请 Token 的创建结果也改为中文一次性提示，不再使用黑色代码块展示。

## 2026-05-31 增量：API 限流与审计闭环摘要

`GET /settings/rate-limit-audit` 用于返回 API 限流和审计覆盖摘要，供管理台设置页只读展示。

| 项目 | 说明 |
|---|---|
| 权限 | user，需携带 `Authorization: Bearer <token>` |
| 租户头 | 不强制 `X-Tenant-ID`，该接口返回全局限流和审计策略摘要 |
| 敏感信息 | 不返回 Redis 地址、Redis 密码、数据库编号或任何连接密钥 |

响应 `data` 字段：

| 字段 | 说明 |
|---|---|
| `rate_limit` | 当前限流后端、窗口、租户/用户/IP 阈值和 Redis 回退说明 |
| `audit.scope` | 审计链路覆盖范围 |
| `audit.automatic_actions` | 通用写操作审计动作，覆盖 POST、PUT、PATCH、DELETE |
| `audit.business_actions` | 业务关键操作审计动作，覆盖成员、邀请等关键租户操作 |
| `audit.query_capabilities` | 审计日志查询能力，包含 action、resource_type、actor_user_id、时间范围、cursor 分页和 CSV 导出 |
| `audit.metadata_fields` | 通用 HTTP 审计元数据字段 |
| `notes` | 限流与审计闭环说明 |

管理台设置页已新增“限流与审计闭环摘要”，用中文卡片展示租户/用户/IP 限流、自动审计动作、业务审计动作、筛选分页和 CSV 导出能力，不展示黑色 JSON 原文区域。

## 2026-05-31 增量：向量检索健康摘要

`GET /settings/vector-search` 用于返回当前 RAG 向量检索能力、隔离校验和运维检查摘要，供管理台设置页只读展示。

| 项目 | 说明 |
|---|---|
| 权限 | user，需携带 `Authorization: Bearer <token>` |
| 租户头 | 不强制 `X-Tenant-ID`，该接口返回全局检索能力摘要 |
| 敏感信息 | 不返回数据库连接串、模型 API Key、原始向量内容或检索请求原文 |

响应 `data` 字段：

| 字段 | 说明 |
|---|---|
| `status` | 向量检索能力状态，当前为 `ready` |
| `embedding_provider` / `embedding_model` | 当前 Embedding Provider 和模型名称摘要 |
| `embedding_dimension` | 当前固定向量维度，首版为 `1536` |
| `index_profile` | pgvector、HNSW、ef_search、向量/全文权重、TopK 和最低分配置摘要 |
| `isolation_checks` | 租户隔离、知识库隔离、删除态过滤等检查项 |
| `retrieval_checks` | 向量维度、混合检索、检索日志等链路检查项 |
| `operations_checks` | 索引迁移、批量导入维护、慢查询观察等运维检查项 |
| `operational_notes` | 运维说明和后续调优边界 |

管理台设置页已新增“向量检索健康摘要”，用中文卡片展示 pgvector/HNSW、1536 维 embedding、租户隔离、知识库隔离、检索日志和 ANALYZE 运维建议，不展示黑色 JSON 原文区域。

## 2026-05-31 增量：Webhook 投递记录 CSV 导出

`GET /webhook-deliveries/export` 已支持按当前租户导出 Webhook 投递记录 CSV。

| 参数 | 说明 |
|---|---|
| `endpoint_id` | Webhook 配置 ID，可选，必须是合法 UUID |
| `event_type` | 事件类型，可选，例如 `webhook.test`、`order.paid` |
| `status` | 投递状态，可选，支持 `success`、`failed` |
| `from` | 创建时间起点，可选，支持 RFC3339 或 `YYYY-MM-DD` |
| `to` | 创建时间终点，可选，支持 RFC3339 或 `YYYY-MM-DD` |
| `limit` | 导出数量，默认 1000，最大 1000 |

导出字段包括：投递 ID、租户 ID、Webhook ID、事件类型、目标地址、状态、HTTP 状态、耗时、重试次数、下次重试时间、最近尝试时间、错误信息、响应摘要、请求体和创建时间。

管理台 Webhook 投递记录区已新增“导出 CSV”按钮，导出时使用当前筛选条件，不展示黑色 JSON 原文区域。

## 2026-05-31 增量：Webhook 投递记录时间范围筛选

`GET /webhook-deliveries` 和 `GET /webhook-deliveries/export` 已支持按创建时间范围筛选。

| 参数 | 说明 |
|---|---|
| `from` | 创建时间起点，可选，支持 RFC3339 或 `YYYY-MM-DD` |
| `to` | 创建时间终点，可选，支持 RFC3339 或 `YYYY-MM-DD` |

当同时传入 `from` 和 `to` 时，服务端会校验起点不能晚于终点。管理台 Webhook 投递记录筛选栏已增加“开始日期”“结束日期”，列表刷新和 CSV 导出均复用该日期范围。

## 2026-05-31 增量：Agent SSE 管理台体验增强

`POST /agents/{agent_id}/chat/stream` 接口协议保持不变，管理台多轮会话区已补齐“流式发送”入口和结果摘要展示。

本轮前端增强点：

- 多轮会话表单新增“流式发送”按钮，修复已有 JS 绑定但页面缺少按钮导致的事件绑定异常。
- 流式发送期间按钮会临时禁用并显示“流式发送中”，避免重复点击造成并发请求。
- 流式结果区展示中文状态：连接中、接收第几段回答、失败原因、完成元信息。
- `done` 事件后展示会话 ID、知识库 ID、使用历史条数、生成模型、生成来源、引用数量和耗时。
- 对最后未以空行结束的 SSE 缓冲区做兜底解析，降低边界情况下最后一段内容丢失的风险。
- 错误事件继续保留已输出内容，并用中文摘要展示失败原因，不展示黑色 JSON 原文区域。

## 2026-05-31 增量：Webhook 配置编辑与密钥保留

`PUT /webhooks/{webhook_id}` 用于编辑当前租户下的 Webhook 配置，继续保持租户隔离，客户端不能跨租户更新配置。

| 字段 | 说明 |
|---|---|
| `name` | Webhook 名称，可选；不传时保留原名称 |
| `url` | 接收地址，可选；不传时保留原地址，传入时必须是合法 HTTP/HTTPS URL |
| `secret` | 签名密钥，可选；不传或传空字符串时保留原密钥，填写新值时替换原密钥 |
| `events` | 订阅事件列表，可选；不传或空数组时保留原事件列表 |
| `status` | 状态，可选，支持 `active`、`disabled`；用于启停 Webhook |

管理台 Webhook 列表已新增“编辑”按钮。点击后会把名称、接收地址、订阅事件和状态载入表单；签名密钥输入框不展示已有密钥原文，留空保存时保留原密钥，填写新密钥时才会替换。启用/停用 Webhook 只提交状态字段，不会清空签名密钥。

## 2026-05-31 增量：Webhook 签名密钥响应脱敏

Webhook 配置相关响应不再返回 `secret` 原文，避免接口响应、浏览器状态或日志中泄漏签名密钥。

| 字段 | 说明 |
|---|---|
| `has_secret` | 是否已配置签名密钥；`true` 表示当前 Webhook 已有密钥，`false` 表示未配置 |

适用接口：

- `GET /webhooks`
- `POST /webhooks`
- `PUT /webhooks/{webhook_id}`

内部投递、测试发送和重试 Worker 仍会从数据库读取真实密钥用于 HMAC 签名，但不会通过 JSON 响应返回给前端。管理台编辑 Webhook 时根据 `has_secret` 展示中文提示，密钥输入框始终为空，留空保存会保留原密钥。

## 2026-05-31 增量：智能体族谱图谱只读接口

`GET /agent-genealogy/graph` 用于返回当前租户下的智能体族谱图谱，首版只读展示，不提供关系编辑能力。

| 项目 | 说明 |
|---|---|
| 权限 | tenant |
| 租户隔离 | 固定使用当前 `X-Tenant-ID`，只返回当前租户智能体和关系 |
| 节点来源 | 当前租户未归档的 `agents` |
| 关系来源 | `agent_genealogy` |

响应 `data` 字段：

| 字段 | 说明 |
|---|---|
| `nodes` | 智能体节点列表，包含 ID、名称、编码、描述、状态和创建时间 |
| `edges` | 族谱关系列表，包含父节点、子节点、关系类型和创建时间 |
| `summary` | 族谱结构诊断摘要，包含节点数、关系数、根节点数、孤立节点数和关系类型分布 |

关系类型首版支持：

- `fork`：派生
- `inherit`：继承
- `compose`：组合
- `route`：路由

管理台智能体页已新增“智能体族谱图谱”区域，用中文摘要展示节点数量、关系数量和根节点数量，并用节点卡片与关系列表展示图谱，不展示黑色 JSON 原文区域。

## 2026-05-31 增量：智能体族谱关系维护

在只读图谱基础上，新增当前租户下的族谱关系创建和删除能力。

### 创建族谱关系

`POST /agent-genealogy/edges`

| 字段 | 说明 |
|---|---|
| `parent_agent_id` | 父智能体 ID，可选；为空时表示根节点关系 |
| `child_agent_id` | 子智能体 ID，必填 |
| `relation_type` | 关系类型，可选，默认 `fork`；支持 `fork`、`inherit`、`compose`、`route` |

校验规则：

- 子智能体必须属于当前租户。
- 父智能体不为空时，必须属于当前租户。
- 父智能体和子智能体不能相同。
- 同一租户下相同父节点、子节点和关系类型不能重复创建。
- 新增关系不能让族谱形成循环，例如已有 `A → B` 时，不允许再创建 `B → A`。

### 删除族谱关系

`DELETE /agent-genealogy/edges/{edge_id}`

仅删除当前租户下匹配的关系；不存在或不属于当前租户时返回未找到。

管理台智能体页“智能体族谱图谱”区域已新增关系维护表单，支持选择父智能体、子智能体和关系类型新增关系；关系列表新增“删除关系”按钮。新增或删除后会自动刷新图谱，所有结果使用中文摘要展示。

## 2026-05-31 增量：智能体族谱环路校验

`POST /agent-genealogy/edges` 已增加环路校验。创建新关系前，服务端会沿子智能体的下游关系递归检查是否已经能回到父智能体；如果会形成循环，返回 `400` 和中文错误信息。

示例：

- 已存在 `A → B`，允许继续创建 `B → C`。
- 已存在 `A → B`，不允许创建 `B → A`。
- 已存在 `A → B → C`，不允许创建 `C → A`。

该校验只在存在父智能体时执行；根节点关系不会形成环路。

## 2026-05-31 增量：智能体族谱 CSV 导出

`GET /agent-genealogy/export` 用于导出当前租户下的智能体族谱节点和关系，返回 `text/csv` 文件。

| 项目 | 说明 |
|---|---|
| 权限 | tenant |
| 租户隔离 | 固定使用当前 `X-Tenant-ID`，只导出当前租户数据 |
| 导出范围 | 当前租户未归档智能体节点，以及当前租户下的族谱关系 |

CSV 字段：

| 字段 | 说明 |
|---|---|
| `section` | 行类型，`node` 表示节点，`edge` 表示关系 |
| `id` | 节点 ID 或关系 ID |
| `name`、`code`、`status`、`description` | 节点信息，仅 `node` 行有值 |
| `parent_agent_id`、`parent_name` | 父智能体信息，仅 `edge` 行有值；根节点关系可为空 |
| `child_agent_id`、`child_name` | 子智能体信息，仅 `edge` 行有值 |
| `relation_type` | 关系类型，仅 `edge` 行有值 |
| `created_at` | 创建时间 |

管理台智能体页“智能体族谱图谱”区域已新增“导出 CSV”按钮，导出时不展示黑色 JSON 原文区域，直接下载 CSV 文件。

## 2026-05-31 增量：智能体族谱筛选

`GET /agent-genealogy/graph` 和 `GET /agent-genealogy/export` 已支持按关键词和关系类型筛选。

查询参数：

| 参数 | 说明 |
|---|---|
| `q` | 可选，按智能体名称、编码或描述模糊匹配，最长 80 个字符 |
| `relation_type` | 可选，按关系类型筛选，支持 `fork`、`inherit`、`compose`、`route` |

筛选口径：

- `q` 会匹配节点自身，也会匹配关系两端的父/子智能体。
- `relation_type` 会限制返回的族谱关系，并只展示相关节点。
- 筛选仍固定使用当前 `X-Tenant-ID`，不允许跨租户查询。
- 已归档智能体不会进入筛选结果；父节点已归档的关系不会进入结果。

管理台“智能体族谱图谱”区域已新增筛选栏，支持输入关键词、选择关系类型、重置筛选。图谱刷新和 CSV 导出都会复用当前筛选条件，所有结果使用中文摘要展示。

## 2026-05-31 增量：智能体族谱结构诊断摘要

`GET /agent-genealogy/graph` 已新增 `summary` 字段，由服务端按当前图谱结果统一计算结构诊断指标。

`summary` 字段：

| 字段 | 说明 |
|---|---|
| `nodes` | 当前图谱结果中的智能体节点数量 |
| `edges` | 当前图谱结果中的族谱关系数量 |
| `roots` | 没有有效父节点关系的节点数量 |
| `isolated` | 既没有有效入边，也没有有效出边的孤立节点数量 |
| `relation_types` | 按关系类型聚合的数量，字段为 `relation_type` 和 `count` |

诊断摘要会复用当前 `q` 和 `relation_type` 筛选结果，管理台“智能体族谱图谱”摘要区已展示根节点、孤立节点和关系类型分布，继续使用中文摘要，不展示黑色 JSON 原文区域。

## 2026-05-31 增量：智能体族谱统计分析

`GET /analytics/summary` 已新增 `genealogy` 字段，用于在数据分析面板展示当前租户智能体族谱结构。

统计口径：

| 字段 | 说明 |
|---|---|
| `nodes` | 当前租户未归档智能体数量 |
| `edges` | 当前租户有效族谱关系数量；子智能体必须未归档，父智能体为空或未归档 |
| `roots` | 没有有效父节点关系的未归档智能体数量 |
| `isolated` | 既没有有效父节点关系，也没有有效子节点关系的未归档智能体数量 |
| `relation_types` | 按 `fork`、`inherit`、`compose`、`route` 等关系类型聚合的数量 |

管理台“数据分析面板”已新增“智能体族谱结构”区域，展示节点、关系、根节点、孤立节点和关系类型分布。所有结果使用中文摘要展示，不新增黑色 JSON 原文区域。

## 2026-05-31 增量：数据分析摘要 CSV 导出

`GET /analytics/summary/export` 用于导出当前租户的数据分析摘要，返回 `text/csv` 文件。

| 项目 | 说明 |
|---|---|
| 权限 | tenant |
| 租户隔离 | 固定使用当前 `X-Tenant-ID`，只导出当前租户统计 |
| 导出范围 | 资源结构、经营指标、智能体族谱结构、用量趋势、风险和最近操作 |

CSV 字段：

| 字段 | 说明 |
|---|---|
| `section` | 统计分区，例如 `resource`、`business`、`genealogy`、`usage_trend`、`risk`、`recent_action` |
| `name` | 中文指标名称或操作名称 |
| `status` | 状态、风险等级或补充信息 |
| `quantity` | 数量或金额 |
| `unit` | 单位 |
| `occurred_at` | 用量趋势日期或最近操作发生时间，可为空 |
| `generated_at` | 本次统计生成时间 |

管理台“数据分析面板”已新增“导出 CSV”按钮，点击后直接下载分析摘要文件，不展示黑色 JSON 原文区域。
