# CLAUDE.md - PixelBeast LiteServer 项目指南

## 项目概述
轻量级 Web 服务器管理面板（类似宝塔面板精简版），Go 后端 + 原生 HTML/CSS/JS 前端。

## 技术栈
- Go 1.25, 标准库 + lego(ACME) + gopsutil(监控)
- 前端：原生 JS ES Modules, 无框架
- 配置：JSON + AES-256-GCM 加密

## 项目结构
```
backend/cmd/main.go        — 入口（只做组装和启动）
backend/internal/           — 业务包（Go internal/，仅本模块可导入）
  ├── panel/                — 管理面板（HTTP API、路由、中间件）
  │   ├── handler.go          — 会话、认证、静态资源、FTP 服务管理
  │   ├── router.go           — 路由表
  │   ├── middleware.go        — 中间件链（Auth/Recovery/CSRF/Logging）
  │   ├── response.go         — API 响应工具
  │   ├── api_system.go       — 系统监控、清理、自启
  │   ├── api_site.go         — 站点管理
  │   ├── api_ssl.go          — 证书申请/续签/导入/DNS
  │   ├── api_ftp.go          — FTP 服务 + 用户 + 文件管理
  │   ├── api_file.go         — 文件 + 压缩 + 分享
  │   ├── api_config.go       — 配置管理
  │   ├── api_backup.go       — 备份管理
  │   ├── api_log.go          — 日志管理
  │   └── api_service.go      — 自启动服务管理
  ├── site/                  — 站点服务（虚拟主机、反向代理、静态文件）
  ├── ssl/                   — SSL 证书核心（ACME、Lego、自动续签）
  ├── ftp/                   — FTP 服务器
  ├── config/                — 配置管理（JSON + AES 加密）
  ├── crypto/                — 加密工具
  ├── logger/                — 日志系统（多分类、轮转、压缩）
  ├── monitor/               — 系统监控（内存、CPU、磁盘）
  ├── file/                  — 文件操作（管理、压缩、安全检查）
  └── backup/                — 备份管理
frontend/embed.go           — 前端 //go:embed admin（前端自己打包资源）
frontend/admin/             — 前端静态资源
  ├── css/                  — 样式（base.css 变量 + components/ + tabs/）
  ├── js/                   — 脚本
  │   ├── core/             — 核心模块（error-handler、api、events、template、keyboard）
  │   ├── utils.js          — 工具函数（escapeHtml 等）
  │   └── app.js            — 应用入口
  └── views/                — HTML 页面
```

## 依赖方向
```
backend/cmd → 所有包 + frontend（只做组装）

panel → config, logger, file, monitor, backup, site, ssl, ftp
site  → config, logger, file, ssl
ssl   → config, logger
ftp   → config, logger
file  → config
logger → config
config → crypto
crypto → 无依赖
frontend → 无（仅 embed 自身资源）

backend/internal/* 互不反向引用 panel/，无循环依赖。
```
**禁止**：site/ssl/ftp/file/logger 不引用 panel/。无循环依赖。

## 编译与运行

```bash
# 编译（NAS 内存有限，必须限制）
cd backend/cmd && GOPROXY=https://goproxy.cn,direct GOMEMLIMIT=300MiB go build -o ../../pixelbeast .

# 运行
./pixelbeast

# 测试
go test ./backend/...

# 静态检查
go vet ./backend/...
```

## .gitignore 注意
- `/ssl/` — 根目录证书存储目录（已忽略）
- `backend/internal/ssl/` — SSL 源码包（**不忽略**，已用 git add -f 追踪）

## 当前进度（v3.2.0-dev）

### 已完成
- [x] 站点管理（多站点、域名绑定、独立端口、反向代理）
- [x] SSL 证书管理（Let's Encrypt HTTP-01/DNS-01, 自动续签, Lego 集成）
- [x] 站点 SSL 配置集成（从证书库选择、HSTS）
- [x] FTP 服务（用户管理、加密密码）
- [x] 文件管理（在线编辑、上传下载、压缩解压）
- [x] 分享链接、系统监控、日志管理
- [x] 管理面板（登录认证、CSRF 防护、速率限制）
- [x] 架构重构（admin→panel、handlers→site、main.go→cmd）
- [x] 代码质量全面提升（37+ 项修复，评分 6.0→9.0）
- [x] 前端优化（6 阶段：错误边界、API 封装、CSS 变量、EventBus、模板引擎、快捷键）
- [x] 单元测试（config/panel/file 3 包 29 用例）

### 代码审查修复清单
- 安全：路径遍历防护、CSRF 中间件接入、配置权限 0600、XSS 转义
- 并发：sync.Once、mutex、goroutine context 退出机制
- 错误处理：HTTP 状态码修正、panic→error、日志统一
- 资源管理：defer-in-loop 修复、Handler.Close()
- 前端：CSS 变量迁移（16 文件 85+ 处）、JS 去重、核心模块

## 开发规范
- 中文注释
- gofmt 格式化
- 错误处理显式，不忽略 error
- 日志使用 logger 包：`logger.LogPanelRuntime(logger.LogLevelInfo, "msg")`
- 前端 CSS 使用变量：`var(--primary)` 而非硬编码色值
- 前端 JS 使用 ES Modules，从 utils.js import escapeHtml
- panel/ 中使用 `fileop` 作为 `file` 包的别名（避免与 multipart.File 冲突）
- 改完编译验证：`cd backend/cmd && GOPROXY=https://goproxy.cn,direct go build -o /dev/null . && go vet ./...`
