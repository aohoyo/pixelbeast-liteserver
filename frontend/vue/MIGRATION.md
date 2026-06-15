# Vue 前端迁移说明

> 像素兽管理面板从原生 JS（ES Modules）迁移至 Vue 3 + TypeScript + Element Plus。

## 技术栈

- **Vue 3** + **Vite 6** — 构建工具 + HMR 热更新
- **TypeScript** — 类型安全
- **Pinia** — 状态管理
- **Element Plus** — UI 组件库（暗色橙色主题）
- **Vue Router** — 客户端路由
- **axios** — HTTP 客户端
- **@xterm/xterm** — Web 终端

## 目录结构

```
frontend/
├── embed.go                 # Go embed：优先 vue/dist，兜底 admin/
├── admin/                   # 原生版（保留兜底，可删）
└── vue/                     # Vue 版
    ├── package.json
    ├── vite.config.ts       # base: /admin/，dev proxy → 9527
    ├── tsconfig*.json
    ├── index.html
    └── src/
        ├── main.ts          # 入口
        ├── App.vue
        ├── router/          # 路由（base /admin/）
        ├── stores/          # Pinia（auth）
        ├── api/             # axios 封装 + 各模块 API
        │   ├── client.ts    # 统一响应/CSRF/认证
        │   └── modules/     # sites/ftp/files/certs/logs/system/backup
        ├── types/           # TS 类型定义
        ├── views/           # 9 个页面视图
        ├── components/      # 可复用组件（对话框/编辑器）
        └── styles/          # 暗色主题
```

## 开发

```bash
cd frontend/vue
npm install          # 首次（注意 .npmrc 已配置，无需管 NODE_ENV）
npm run dev          # Vite dev server → http://localhost:5173
```

开发时 Vite 代理 `/admin/api` → `http://localhost:9527`（Go 后端）。需同时运行后端：

```bash
# 项目根目录
go run ./backend/cmd
```

## 构建

```bash
cd frontend/vue
npm run build        # 输出到 dist/，vue-tsc 类型检查 + Vite 打包
```

构建产物 `dist/` 会被 `frontend/embed.go` 自动嵌入 Go 二进制（生产模式）。

## 关键架构决策

### 1. 后端零改动
所有后端 API、CSRF、session 逻辑完全不动。Vue `api/client.ts` 精确复刻后端约定：
- 统一响应 `{code, message, data}` → 自动解包
- CSRF：登录后 `/api/system/status` 返回 `data.csrf_token`，自动注入 `X-CSRF-Token`
- 认证：Cookie session（`credentials: 'include'`）

### 2. 安全入口路径
后端安全入口要求所有请求带 `adminPath` 前缀（默认 `/admin`）：
- 页面：`/admin/login`、`/admin/sites`（Vue Router base = `/admin/`）
- API：`/admin/api/*`（axios baseURL 从 URL 推导）
- 静态资源：`/admin/assets/*`（Vite base = `/admin/`）
- WS：`/admin/api/terminal/ws`

### 3. embed 优先级
`frontend/embed.go` 按优先级返回 FS：
1. `frontend/vue/dist`（Vue 构建产物，开发+生产磁盘模式）
2. `frontend/admin`（原生版，磁盘）
3. 嵌入的原生版（二进制兜底）

## 模块迁移状态

| 模块 | 视图 | 状态 |
|------|------|------|
| 登录 | LoginView.vue | ✅ |
| 首页/监控 | HomeView.vue | ✅ 实时 CPU/内存/磁盘/网络 |
| 站点管理 | SitesView.vue | ✅ CRUD + 启停 + 批量 |
| FTP | FtpView.vue | ✅ 用户 CRUD + 服务状态 |
| 文件管理 | FilesView.vue | ✅ 浏览/上传/编辑/压缩/分享/回收站 |
| 终端 | TerminalView.vue | ✅ xterm.js + WS |
| SSL 证书 | CertView.vue | ✅ 申请/续期/粘贴/进度 |
| 日志 | LogsView.vue | ✅ 分类/过滤/统计 |
| 设置 | SettingsView.vue | ✅ 常规/目录/日志/备份 |

## 体积

| 项 | 原生版 | Vue 版 |
|----|--------|--------|
| JS gzip | codemirror+xterm >1MB | ~365KB（含 EP 全量） |
| 路由懒加载 | — | ✅ 每模块独立 chunk |

后续用 Element Plus 按需引入可降至 ~150KB。

## 已知简化项（后续优化）

- 文件编辑器用 textarea（原生版用 CodeMirror 6，可后续集成 `@codemirror/*`）
- DNS 服务商管理 UI 未做完整表单（API 已就绪）
- 系统清理项 UI 未做（API 已就绪）
- 备份创建进度未做轮询
