# CLAUDE.md

> 本文件为 Claude Code (claude.ai/code) 提供项目文档索引。

## 📚 文档目录

| 文档 | 说明 |
|------|------|
| [docs/overview.md](docs/overview.md) | 项目概述、特性、技术栈 |
| [docs/architecture.md](docs/architecture.md) | 架构设计、模块说明、数据流 |
| [docs/api.md](docs/api.md) | API 端点、响应格式、调用示例 |
| [docs/frontend.md](docs/frontend.md) | 前端开发、模块说明、开发规范 |
| [docs/coding-standards.md](docs/coding-standards.md) | 代码规范、提交规范、文档更新规则 |
| [docs/deployment.md](docs/deployment.md) | 开发环境、构建、部署指南 |
| [docs/CHANGELOG.md](docs/CHANGELOG.md) | 更新日志、版本历史 |

## 🚀 快速开始

### 开发模式
```bash
npm run dev
# 访问: http://localhost:9527/admin
# 默认凭据: admin / admin123
```

### 构建
```bash
npm run build
```

## 📂 关键目录

```
src/
├── handlers/    # 协议层 (HTTP/FTP)
├── admin/       # 管理面板业务逻辑 + API（面板专用）
├── static/admin/ # Web 管理界面（嵌入）
└── config/      # 配置管理
```

## 🏗️ 架构设计原则

### 为什么没有 services/ 层？

本项目采用**轻量级分层**，`admin/` 目录同时承担业务逻辑和 API 职责：

| 场景 | 架构选择 |
|------|----------|
| 业务逻辑被多处复用 | 抽到独立 `services/` 层 |
| 业务逻辑只在面板用 | 放在 `admin/` 下 ✅（本项目） |

**原因**：
- 站点管理、文件管理、FTP 管理都是**面板独占**功能
- 不需要在 handlers 层复用这些业务逻辑
- 避免过度设计，保持代码简洁

**何时需要拆分 services/**：
- 单个文件超过 800 行
- 业务逻辑需要在多处复用
- 需要单独测试业务逻辑层
- 多人协作职责边界模糊

### 分层职责

```
handlers/     ← 对外协议层（HTTP 站点服务、FTP 服务）
admin/        ← 管理面板（业务逻辑 + API，面板专用）
config/       ← 配置管理
static/       ← 前端资源（嵌入二进制）
```

## 🔑 重要提醒

> **每次更新或重大变更时，必须同步更新相关文档！**

| 变更类型 | 更新文档 |
|---------|---------|
| 新增 API | `docs/api.md` |
| 修改架构 | `docs/architecture.md` |
| 前端改动 | `docs/frontend.md` |
| 配置变更 | `docs/overview.md` |
| 规范变更 | `docs/coding-standards.md` |
| 任何变更 | `docs/CHANGELOG.md` |

## 📝 Claude 使用建议

1. **查看架构**: 先读 `docs/architecture.md` 理解整体设计
2. **API 开发**: 参考 `docs/api.md` 了解响应格式规范
3. **前端开发**: 参考 `docs/frontend.md` 了解模块结构
4. **代码提交**: 遵循 `docs/coding-standards.md` 规范

## 🎯 当前版本

- **版本**: v3.0.0
- **模式**: 静态 HTML + 模块化 JS
- **热重载**: Air (`npm run dev`)
