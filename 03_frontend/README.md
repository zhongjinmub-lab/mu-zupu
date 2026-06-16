# 智能体族谱 SAAS 管理台（Vue3 工程化版本）

基于 Vue3 + TypeScript + Vite + Element Plus 的前端工程化管理台。

## 技术栈

- Vue 3.4+（Composition API）
- TypeScript 5.5+
- Vite 5
- Element Plus 2.7+
- Pinia（状态管理）
- Vue Router 4（路由管理）
- Axios（HTTP 请求）

## 开发

```bash
# 安装依赖
npm install

# 启动开发服务器（默认 3000 端口）
npm run dev

# 类型检查
npm run type-check

# 构建生产包
npm run build
```

## 目录结构

```
src/
├── api/          # API 请求封装
├── assets/       # 静态资源
├── components/   # 公共组件
├── layouts/      # 布局组件
├── router/       # 路由配置
├── stores/       # Pinia 状态
└── views/        # 页面视图
```

## 已实现视图

- 登录页
- 总览仪表盘
- 智能体列表 / 详情 / 版本管理
- 族谱图
- 渠道接入（类型目录、CRUD、启用/禁用、复制、接入代码生成）
- 知识库（列表/创建、文档管理、RAG 问答测试）
- License（创建/激活/吊销/验签）
- 数据分析（资源统计、用量趋势、风险项、经营分布）
- Webhook（端点 CRUD、测试发送、投递记录、重试）
- 设置（限流/运行配置/监控/向量检索/敏感字段/告警）
