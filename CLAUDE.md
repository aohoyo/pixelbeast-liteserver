# CLAUDE.md - PixelBeast LiteServer 项目指南

## 项目概述
轻量级 Web 服务器管理面板（类似宝塔面板精简版），Go 后端 + 原生 HTML/CSS/JS 前端。

## 技术栈
- Go 1.25, 标准库 + lego(ACME) + gopsutil(监控)
- 前端：原生 JS ES Modules, 无框架
- 配置：JSON + AES-256-GCM 加密

## 当前进度（2026-04-12）

### 已完成
- [x] 站点管理（多站点、域名绑定、独立端口、反向代理）
- [x] SSL 证书管理（Let's Encrypt HTTP-01/DNS-01, 自动续签, Lego 集成）
- [x] 站点 SSL 配置集成（从证书库选择、HSTS）
- [x] FTP 服务（用户管理、加密密码）
- [x] 文件管理（在线编辑、上传下载、压缩解压）
- [x] 分享链接、系统监控、日志管理
- [x] 管理面板（登录认证、CSRF 防护）

### BUG 修复（2026-04-12）
- [x] P0: FTP 密码明文返回 → 改为掩码
- [x] P0: 路径遍历 → URL 解码后校验
- [x] P0: XSS → 目录列表 HTML 转义
- [x] P0: 硬编码默认密码 → 已删除
- [x] P1: FTP OOM → 流式传输
- [x] P1: defer-in-loop → 手动 close
- [x] P1: escapeHtml 未导入 → 添加 import

### 代码规范清理 - 高优先级（2026-04-12）
- [x] 1.1 config.go save 方法提取 saveJSON
- [x] 1.2 config.go load 方法提取 loadOrCreate
- [x] 1.5 handler.go 静态资源服务函数合并
- [x] 1.7 sites.go 服务控制方法去重
- [x] 1.8 ftp.go 服务控制方法去重
- [x] 静默忽略 error 修复
- [x] 1.10 openDirPicker() 三文件统一到 utils.js
- [x] 1.11 initNumberInputs() 两文件统一
- [x] 1.12 escapeHtml() 三处统一
- [x] 1.13 站点/FTP 服务控制提取到 BaseTab.js（~150行）
- [x] console.log/warn 清理（9处）

### 待完成 - 代码规范清理
- [ ] 1.3 config.go delete 方法提取（中）
- [ ] 1.4 crypto.go GCM 初始化提取（中）
- [ ] 1.6 FTP/HTTP 文件操作 API 去重（中）
- [ ] 1.9 os.UserHomeDir() 重复调用（中）
- [ ] 1.14 bindRowEvents 重复（低）
- [ ] 1.15 HTML Modal 结构重复（低）
- [ ] 1.16 服务控制工具栏重复（低）
- [ ] 2.x 命名规范统一（中）
- [ ] 3.x 长函数拆分（中）
- [ ] 4.x CSS 重复定义清理（中）

### 待完成 - 架构优化（计划二）
- 详见 docs/plan-2-architecture.md
- 后端拆分（api.go、SSL 模块、路由重构）
- 中间件体系
- 可测试性改造

## 文档索引
- `docs/ssl-redesign-plan.md` — SSL 重设计规划
- `docs/code-audit-report.md` — 代码审计报告
- `docs/project-report.md` — 项目综合报告
- `docs/plan-1-code-cleanup.md` — 代码规范清理计划
- `docs/plan-2-architecture.md` — 架构优化计划

## 开发规范
- 中文注释
- gofmt 格式化
- 错误处理显式，不忽略 error
- 保持代码风格一致
- 改完编译验证 go build ./... && go vet ./...
