# 2026-05-31 Webhook 投递记录 CSV 导出增量验收

## 本轮目标

在已有 Webhook 投递记录筛选、摘要和重试能力基础上，补齐投递记录导出能力，便于排查失败投递、交付验收和离线留档。

本轮只做本地主线开发，不执行服务器部署验证，不读取服务器信息文件。

## 后端变更

- 新增接口：`GET /api/v1/webhook-deliveries/export`。
- 接口固定按当前租户上下文导出，不接受客户端覆盖 `tenant_id`。
- 支持复用投递记录筛选条件：
  - `endpoint_id`
  - `event_type`
  - `status`
  - `limit`
- 列表接口仍保持最多 200 条；导出接口最多 1000 条。
- CSV 字段包括：
  - 投递 ID、租户 ID、Webhook ID、事件类型、目标地址、状态。
  - HTTP 状态、耗时、重试次数。
  - 下次重试时间、最近尝试时间。
  - 错误信息、响应摘要、请求体、创建时间。

## 前端变更

- Webhook 投递记录区新增“导出 CSV”按钮。
- 导出时使用当前筛选表单中的 Webhook、事件、状态和数量条件。
- 导出失败时显示中文错误摘要。

## 本地验证

- 后端：`go test ./...`
- 前端：`D:\Node.js\node.exe --check 02_frontend\assets\app.js`
- 代码检查：`git diff --check`

## 未执行项

- 未部署服务器。
- 未读取 `C:\Users\zhongjinmu\Desktop\192.168.6.66服务器信息.txt`。
- 未使用线上投递数据执行真实导出。
