# 智能体族谱SAAS v1.0 交付文档

本目录保留 MVP 开发和商业交付必需文档，已清理冗余说明。

## 文件清单

| 文件 | 说明 |
|---|---|
| 01_开发任务清单.md | P0/P1 开发任务与下一步优先级 |
| 02_MVP一期开发排期.md | 8 周 MVP 排期和主链路 |
| 03_数据库表结构_v1.sql | PostgreSQL 核心表结构与索引 |
| 04_API接口清单.md | 已落地接口与 MVP 规划接口 |
| 05_Redis_Key设计.md | Redis Key、TTL、锁、幂等、队列规范 |
| 06_交付验收清单.md | 功能、安全、运维、MVP 验收清单 |

## 技术基线

- Go go1.26.3 + Gin
- PostgreSQL 16 + pgvector
- Redis 7.4
- MinIO / S3
- Vue3 + TypeScript + Vite + Element Plus
- Docker Compose

## 使用建议

1. 先按 `01_开发任务清单.md` 拆任务；
2. 再按 `02_MVP一期开发排期.md` 推进迭代；
3. 开发中同步维护 SQL、API 和 Redis 文档；
4. 每阶段按 `06_交付验收清单.md` 验收。
