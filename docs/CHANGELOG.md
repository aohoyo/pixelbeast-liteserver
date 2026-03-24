# 更新日志

本文档记录项目的重大变更和新功能。

## [3.0.0] - 2024-03-24

### 重大变更

- **前端架构**: 采用组件化模式
  - 标签页组件存储为独立的 HTML 文件
  - 通过 `loader.js` 动态加载组件
  - 更易于维护和扩展

- **添加 `/components/` 路由**: 服务器端新增组件文件支持

### 新增

- 完整的文档系统 (`docs/` 目录)
- Air 热重载配置 (`.air.toml`)

### 修复

- 修复了 `ftp.js` 和 `cert.js` 中的依赖访问问题
- 修复了组件加载路径问题

### 文档

- 新增 `docs/overview.md` - 项目概述
- 新增 `docs/architecture.md` - 架构设计
- 新增 `docs/api.md` - API 文档
- 新增 `docs/frontend.md` - 前端开发指南
- 新增 `docs/coding-standards.md` - 代码规范
- 新增 `docs/deployment.md` - 部署指南
- 更新 `CLAUDE.md` - 改为文档索引

---

## 版本号规则

遵循语义化版本 (Semantic Versioning)：

- **MAJOR**: 不兼容的 API 变更
- **MINOR**: 向后兼容的功能新增
- **PATCH**: 向后兼容的问题修复

---

## 更新流程

当你进行重大更新时，请在此文件顶部添加新条目：

```markdown
## [x.x.x] - YYYY-MM-DD

### 新增
- 功能描述

### 修改
- 变更描述

### 修复
- 修复描述

### 文档
- 文档更新
```
