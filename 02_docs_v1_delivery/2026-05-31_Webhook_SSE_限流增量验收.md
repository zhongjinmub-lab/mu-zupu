# 2026-05-31 Webhook、SSE 流式对话、API 限流增量验收

## 本轮目标

本轮继续沿用现有 Go/Gin 后端和静态管理台，不切换 Vue3 工程化，优先补齐平台能力：

- Webhook 配置管理、测试发送、投递记录。
- Agent 多轮会话 SSE 流式响应。
- 租户级、用户级、登录注册 IP 级 API 限流。
- 静态管理台新增 Webhook 管理区、流式发送入口、限流策略说明。

## 后端变更

- 新增 `webhook_endpoints`、`webhook_deliveries` 两张表，全部带 `tenant_id` 隔离。
- 新增 `/api/v1/webhooks`、`/api/v1/webhooks/:webhook_id/test`、`/api/v1/webhook-deliveries`。
- 新增同步投递服务，记录投递状态、HTTP 状态码、错误信息、耗时、重试次数。
- 首批事件已接入：
  - `webhook.test`
  - `order.paid`
  - `license.activated`
  - `license.revoked`
  - `agent.chat.finished`
- 新增 `/api/v1/agents/:agent_id/chat/stream`，事件类型为 `start`、`reference`、`delta`、`done`、`error`。
- 当前模型未提供真实流式能力时，后端会先生成完整回答，再按段输出 `delta`，保证前端流式体验和协议稳定。
- 新增内存限流中间件：
  - 同一租户每分钟 120 次请求。
  - 同一用户每分钟 60 次请求。
  - 登录/注册同一 IP 每分钟 20 次请求。
  - 触发限流时返回 `429` 和中文错误信息。

## 前端变更

- 导航新增 `Webhook` 管理入口。
- Webhook 页面支持：
  - 创建配置。
  - 查看配置列表。
  - 启用/停用。
  - 删除。
  - 测试发送。
  - 查看投递记录中文摘要。
- Agent 多轮会话新增“流式发送”按钮，可实时追加回答内容。
- 设置页新增 API 限流策略只读说明。
- 新增展示均使用中文摘要，没有新增黑色 JSON 原文区域。

## 本地验证

- 后端执行 `go test ./...` 通过。
- 前端执行 `D:\Node.js\node.exe --check 02_frontend\assets\app.js` 通过。

## 服务器部署

- 服务器：`192.168.6.66`
- 后端目录：`/opt/mu-agent-saas`
- 前端目录：`/www/wwwroot/zupu.jiangxinnet.com/saas`
- 数据库迁移：`000018_webhooks` 已应用。
- 后端服务：`mu-agent-saas` 已重启并处于 `active`。
- 文档 worker：`mu-agent-document-worker` 已重启并处于 `active`。
- Nginx 配置检查通过。
- 服务器备份目录：`/opt/mu-agent-saas/backups/20260531011911`

## 线上验证

- `/saas-api/api/v1/ready` 返回 ready。
- `/saas-api/api/v1/health` 返回 ok。
- `/saas/` 可访问，页面包含 Webhook 新入口。
- `/saas/assets/app.js` 可访问，包含 `/webhooks` 和 `chat/stream` 调用。
- `/saas/assets/app.css` 可访问，包含限流说明样式。
- `journalctl -u mu-agent-saas` 显示本次启动正常，未发现启动错误。

## 未执行项

- 未在线上批量压测限流阈值，避免对生产登录接口造成不必要影响。
- 未使用真实第三方 Webhook 接收器做公网回调验收，当前已验证接口、迁移、部署和管理台入口可用。
- 未使用真实业务账号执行完整 SSE 对话，因为本轮部署验证不读取或新增生产登录凭据。

## 下一步建议

- 增加 Webhook 异步队列和失败重试 worker。
- 将内存限流升级为 Redis 限流，支持多实例部署。
- 给 Webhook 和 SSE 增加专门的 handler 单元测试。
- 在管理台增加 Webhook 投递记录按事件类型和状态筛选。
