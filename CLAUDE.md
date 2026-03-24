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
├── admin/       # 管理面板 handlers
├── static/admin/ # Web 管理界面（嵌入）
├── services/    # 业务逻辑层
└── config/      # 配置管理
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
