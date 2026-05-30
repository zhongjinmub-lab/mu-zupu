# 智能体族谱SAAS 最终交付包

更新时间：2026-05-27

## 1. 目录说明

| 目录/文件 | 说明 |
|---|---|
| 01_backend | Go + Gin 后端工程，含迁移、认证、租户、文件上传、知识库写入与检索 |
| 02_frontend | 免构建静态管理台 MVP，覆盖登录、租户、知识库、Agent、License 和用量 |
| 02_docs_v1_delivery | MVP 开发和商业交付文档 |
| 03_智能体族谱SAAS_开发方案.md | 精简版总开发方案 |
| 04_最终交付验收清单.md | 最终交付验收清单 |
| manifest.json | 文件清单与 SHA256 摘要 |

## 2. 技术基线

- Go go1.26.3
- Gin
- PostgreSQL 16 + pgvector
- Redis 7.4
- MinIO / S3
- 静态 HTML/CSS/JavaScript 管理台 MVP
- Vue3 + TypeScript + Vite + Element Plus（后续工程化目标）
- Docker Compose

## 3. 已完成

- 后端工程骨架；
- Docker Compose 基础依赖；
- PostgreSQL/pgvector 迁移；
- 健康检查和就绪检查；
- 请求 ID 中间件；
- 用户注册、登录、JWT 签发与解析；
- JWT 中间件和当前用户接口；
- 租户创建、租户列表与 `X-Tenant-ID` 租户上下文；
- 知识库创建、列表、文档创建、chunk 创建；
- 1536 维 embedding 写入校验；
- 文件上传到 MinIO/S3 和租户文件列表；
- 已上传文本文件生成知识库文档与 pending chunks；
- file-backed 文档切片重建；
- 数据库 document_jobs 队列与同步 worker run；
- 后台常驻文档 worker 进程；
- pending chunks 查询、单 chunk embedding 写回；
- 本地 deterministic embedding provider 与同步 embedding run；
- OpenAI-compatible HTTP embedding provider 接入；
- OpenAI-compatible HTTP generation provider 接入；
- KB 向量检索接口；
- KB 混合检索接口；
- 当前租户 KB 检索接口；
- RAG 问答生成闭环接口；
- Agent 创建、列表、详情、更新、发布、回滚、归档接口；
- Agent 与知识库绑定/解绑接口，绑定前校验 tenant/kb 访问权限；
- Agent 测试会话接口，写入 conversation/messages 并返回回答与引用；
- Agent 多轮会话接口，支持历史消息编排；
- 套餐、订阅和用量统计 MVP；
- License 授权 MVP，支持租户内创建、列表、激活和吊销；
- 前端管理台 MVP，已部署到 `/saas/`；
- HNSW、全文、Trigram 索引；
- 检索 Profile 和检索日志；
- 生产部署模板、Linux amd64 打包脚本、服务器 systemd/Nginx/备份脚本；
- 文档精简、API 文档和验收清单更新。

## 4. 验证命令

```bash
cd 01_backend
go test ./...
```

当前验证结果：通过。

线上入口：

- 管理台：[https://zupu.jiangxinnet.com/saas/](https://zupu.jiangxinnet.com/saas/)
- 后端 API：[https://zupu.jiangxinnet.com/saas-api/api/v1/ready](https://zupu.jiangxinnet.com/saas-api/api/v1/ready)

## 5. 下一步

```text
套餐额度硬限制 → 支付接入 → License 离线签名验签闭环 → 前端工程化
```
