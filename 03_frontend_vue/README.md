# 智能体族谱 SaaS 管理台（当前生产版）

本目录是当前线上 `/saas/` 使用的生产管理台，技术栈为 Vue3 + Vite + TypeScript + Element Plus。

## 与其它前端目录的关系

- `03_frontend_vue`：当前生产发布入口，发布包默认打包 `dist/`。
- `02_frontend`：历史免构建静态管理台，保留作回退和旧版功能参考。
- `03_frontend`：工程化探索版，暂不作为生产默认发布入口。

## 本地开发

```powershell
npm install --legacy-peer-deps
npm run dev
```

开发模式默认把 `/api` 代理到 `http://localhost:8080`。

## 生产构建

```powershell
$env:VITE_API_BASE="/saas-api/api/v1"
npm run build
```

构建产物位于 `dist/`。Vite `base` 固定为 `/saas/`，适配线上 Nginx 子路径部署。

## 发布说明

后端发布脚本默认读取本目录的 `dist/`：

```powershell
cd ..\01_backend
.\scripts\build_release.ps1 -Version v0.1.0
```

如果需要临时打包其它前端目录，可传入 `-FrontendDir`。
