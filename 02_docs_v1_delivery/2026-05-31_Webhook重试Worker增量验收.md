# 2026-05-31 Webhook 重试 Worker 增量验收

## 本轮目标

在已上线的 Webhook 同步投递基础上，补齐失败投递的自动重试能力，降低外部系统短暂不可用时的通知丢失风险。

## 后端变更

- 新增 `000019_webhook_retries` 数据库迁移。
- `webhook_deliveries` 新增：
  - `next_retry_at`：下一次重试时间。
  - `last_attempt_at`：最近一次投递尝试时间。
- Webhook 同步投递失败后，会按默认策略写入下一次重试时间。
- 新增 `mu-agent-webhook-worker` 独立 worker 进程。
- Worker 周期性领取到期失败投递，并重新发送。
- Worker 使用 5 分钟领取租约，避免进程异常退出后投递记录永久卡住。
- 默认重试策略：
  - 最大重试次数：3 次。
  - 基础重试间隔：60 秒。
  - 退避方式：按重试次数指数递增。

## 配置项

- `WEBHOOK_WORKER_INTERVAL_SECONDS`
- `WEBHOOK_WORKER_BATCH_SIZE`
- `WEBHOOK_MAX_RETRIES`
- `WEBHOOK_RETRY_BASE_SECONDS`

## 部署模板

- 新增二进制：`bin/mu-agent-webhook-worker`
- 新增 systemd 服务：`mu-agent-webhook-worker.service`
- 发布脚本已包含 webhook worker 构建。
- 生产 README 已补充安装和启用说明。

## 本地验证

- `go test ./...` 通过。
- `D:\Node.js\node.exe --check 02_frontend\assets\app.js` 通过。
- Webhook service 单元测试覆盖：
  - 签名头写入。
  - 成功投递状态。
  - 失败投递重试时间计算。
  - 达到最大重试次数后停止调度。

## 服务器部署验证

- 应用 `000019_webhook_retries` 迁移。
- 安装并启动 `mu-agent-webhook-worker`。
- 确认 `mu-agent-saas`、`mu-agent-document-worker`、`mu-agent-webhook-worker` 均为 `active`。
- 检查 `/ready`、`/health` 正常。
- 检查 worker 日志无启动错误。

## 已完成部署结果

- 服务器：`192.168.6.66`
- 部署时间戳：`20260531020522`
- 数据库迁移：`000019_webhook_retries` 已应用。
- 服务状态：
  - `mu-agent-saas`：`active`
  - `mu-agent-document-worker`：`active`
  - `mu-agent-webhook-worker`：`active`
- 线上验证：
  - `/saas-api/api/v1/ready` 返回 `200`。
  - `/saas-api/api/v1/health` 返回 `200`。
  - `journalctl -u mu-agent-webhook-worker` 显示 worker 已按 `interval=15s batch_size=20 max_retries=3` 启动。
- 服务器备份目录：`/opt/mu-agent-saas/backups/20260531020522`
