# 智能体族谱SAAS Redis Key 设计 v1.0

Redis 用于缓存、限流、锁、幂等、短期上下文和异步队列。

## 1. Key 命名

```text
mu:{env}:{module}:{tenant_id}:{biz_key}
```

全局 Key 使用 `global` 替代 `tenant_id`。

## 2. Key 清单

| 模块 | Key | 类型 | TTL | 说明 |
|---|---|---|---|---|
| 认证 | mu:{env}:auth:global:access:{jti} | string | 2h | Access Token 状态 |
| 认证 | mu:{env}:auth:global:refresh:{jti} | string | 30d | Refresh Token 状态 |
| 限流 | mu:{env}:rate:global:{scope}:{key} | string | 1m | API 限流，当前 scope 包含 tenant、user、ip |
| 锁 | mu:{env}:lock:{tenant}:{resource}:{id} | string | 10s-5m | 分布式锁 |
| 幂等 | mu:{env}:idem:{tenant}:{key} | string | 24h | 请求幂等 |
| 租户 | mu:{env}:tenant:global:{tenant_id} | json | 10m | 租户信息缓存 |
| 权限 | mu:{env}:perm:{tenant}:{user_id} | json/set | 10m | 用户权限缓存 |
| 套餐 | mu:{env}:quota:{tenant}:current | json | 5m | 当前额度 |
| 用量 | mu:{env}:usage:{tenant}:{date} | hash | 3d | 当日用量 |
| Agent | mu:{env}:agent:{tenant}:{agent_id}:config | json | 5m | Agent 配置 |
| 会话 | mu:{env}:conv:{tenant}:{conversation_id}:ctx | list/json | 2h | 短期上下文 |
| RAG | mu:{env}:rag:{tenant}:{kb_id}:search:{hash} | json | 10m | 检索缓存 |
| 支付 | mu:{env}:pay:global:callback:{pay_no} | string | 24h | 回调幂等 |
| License | mu:{env}:license:global:revoked | set | 永久 | 吊销列表 |
| 队列 | mu:{env}:queue:embedding | stream/list | - | embedding 任务 |
| 队列 | mu:{env}:queue:usage | stream/list | - | 用量汇总任务 |

## 3. 使用规范

- 写库成功后删除缓存，不直接更新复杂缓存；
- 锁必须写入随机 token，释放时校验 token；
- 支付、订单、License、危险工具调用必须幂等；
- 关键权限与额度最终以数据库为准；
- 生产环境需监控 Redis 内存、命中率、慢命令和 Key 数量。

## 4. 当前接入状态

- `RATE_LIMIT_BACKEND=memory`：默认策略，使用进程内固定窗口限流，适合单实例或本地开发。
- `RATE_LIMIT_BACKEND=redis`：使用 Redis `INCR` + `EXPIRE` 实现固定窗口限流，适合多实例部署。
- Redis 限流异常时会自动回退到内存限流，避免 Redis 短时不可用导致业务 API 整体不可用。
- 当前限流窗口默认 60 秒，可通过 `RATE_LIMIT_WINDOW_SECONDS` 调整。
