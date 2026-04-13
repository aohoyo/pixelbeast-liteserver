# CLAUDE.md - PixelBeast LiteServer 项目指南

## 项目概述
轻量级 Web 服务器管理面板（类似宝塔面板精简版），Go 后端 + 原生 HTML/CSS/JS 前端。

## 技术栈
- Go 1.25, 标准库 + lego(ACME) + gopsutil(监控)
- 前端：原生 JS ES Modules, 无框架
- 配置：JSON + AES-256-GCM 加密

## 项目结构
```
src/cmd/main.go           — 入口（只做组装和启动）
src/embed.go              — 静态资源嵌入
src/panel/                — 管理面板（HTTP API、路由、中间件）
  ├── handler.go          — 会话、认证、静态资源、FTP 服务管理
  ├── router.go           — 路由表
  ├── middleware.go        — 中间件链
  ├── response.go         — API 响应工具
  ├── api_system.go       — 系统监控、清理、自启
  ├── api_site.go         — 站点管理
  ├── api_ssl.go          — 证书申请/续签/导入/DNS
  ├── api_ftp.go          — FTP 服务 + 用户 + 文件管理
  ├── api_file.go         — 文件 + 压缩 + 分享
  ├── api_config.go       — 配置管理
  ├── api_backup.go       — 备份管理
  ├── api_log.go          — 日志管理
  └── api_service.go      — 自启动服务管理
src/site/                 — 站点服务（虚拟主机、反向代理、静态文件）
src/ssl/                  — SSL 证书核心（ACME、自动续签）
src/ftp/                  — FTP 服务器
src/config/               — 配置管理（JSON + AES 加密）
src/crypto/               — 加密工具
src/logger/               — 日志系统（多分类、轮转、压缩）
src/monitor/              — 系统监控（内存、CPU、磁盘）
src/file/                 — 文件操作（管理、压缩、安全检查）
src/backup/               — 备份管理
src/static/               — 前端静态资源
```

## 依赖方向
```
src/cmd → 所有包（只做组装）

panel → config, logger, file, monitor, backup, site, ssl, ftp
site  → config, logger, file, ssl
ssl   → config, logger
ftp   → config, logger
file  → config
logger → config
config → crypto
crypto → 无依赖
```
**禁止**：site/ssl/ftp/file/logger 不引用 panel/。无循环依赖。

## 当前进度（2026-04-14）

### 已完成
- [x] 站点管理（多站点、域名绑定、独立端口、反向代理）
- [x] SSL 证书管理（Let's Encrypt HTTP-01/DNS-01, 自动续签, Lego 集成）
- [x] 站点 SSL 配置集成（从证书库选择、HSTS）
- [x] FTP 服务（用户管理、加密密码）
- [x] 文件管理（在线编辑、上传下载、压缩解压）
- [x] 分享链接、系统监控、日志管理
- [x] 管理面板（登录认证、CSRF 防护）
- [x] 架构重构（handlers/ 转发层消除、core/ 包体系、中间件链、路由表）
- [x] 包结构重组（去 core/ 层级、admin→panel、God Object 拆分、文件合并）

### BUG 修复（2026-04-12）
- [x] P0: FTP 密码明文返回 → 改为掩码
- [x] P0: 路径遍历 → URL 解码后校验
- [x] P0: XSS → 目录列表 HTML 转义
- [x] P0: 硬编码默认密码 → 已删除
- [x] P1: FTP OOM → 流式传输
- [x] P1: defer-in-loop → 手动 close
- [x] P1: escapeHtml 未导入 → 添加 import

### 待完成 - 代码规范清理
- [ ] 1.3 config.go delete 方法提取（中）
- [ ] 1.4 crypto.go GCM 初始化提取（中）
- [ ] 1.6 FTP/HTTP 文件操作 API 去重（中）
- [ ] 1.9 os.UserHomeDir() 重复调用（中）
- [ ] 2.x 命名规范统一（中）
- [ ] 3.x 长函数拆分（中）

### 待完成 - 后续优化
- CSRF 中间件集成（已实现 CSRPMiddleware，待接入路由）
- RateLimit 中间件
- 可测试性改造（接口抽象、mock）

## 文档索引
- `docs/ssl-redesign-plan.md` — SSL 重设计规划
- `docs/code-audit-report.md` — 代码审计报告
- `docs/project-report.md` — 项目综合报告
- `docs/plan-1-code-cleanup.md` — 代码规范清理计划
- `docs/plan-2-architecture.md` — 架构优化计划
- `docs/plan-5-package-restructure.md` — 包结构重组计划

## 开发规范
- 中文注释
- gofmt 格式化
- 错误处理显式，不忽略 error
- 保持代码风格一致
- 改完编译验证 go build ./... && go vet ./...
- panel/ 中使用 `fileop` 作为 `file` 包的别名（避免与 multipart.File 变量冲突）
