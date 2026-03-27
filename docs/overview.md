# 项目概述

**像素兽 轻量服务器** (PixelBeast LiteServer) 是一个用 Go 编写的超轻量级跨平台 Web+FTP 服务器。设计理念是"小而强悍，无所不能"。

## 核心特性

- **内存占用**：8-15MB RAM
- **单文件部署**：依赖极少，下载即用
- **跨平台**：Windows、Linux、ARM64 从单一代码库构建
- **功能集成**：HTTP + FTP + Web 管理界面
- **配置加密**：敏感信息 AES-256-GCM 加密存储

## 版本信息

- 当前版本：v3.0.0
- 许可证：MIT

## 默认配置

### 端口

| 服务 | 默认端口 |
|------|----------|
| 管理面板 | 9527 |
| HTTP | 8080 |
| FTP | 2121 |

### 默认凭据

| 服务 | 用户名 | 密码 |
|------|--------|------|
| 管理面板 | `admin` | `admin123` |

> **注意**：密码在配置文件中加密存储

### 配置文件结构

```
config/
├── server.json      # 服务配置（端口、日志、admin账号）
├── sites.json       # 站点配置
├── ftp.json         # FTP配置（用户密码加密）
└── secrets.key      # 加密密钥（自动生成）
```

**server.json 示例**：
```json
{
  "http_port": 8080,
  "admin_port": 9527,
  "admin_username": "admin",
  "admin_password": "加密后的密码",
  "admin_path": "/admin",
  "log": {
    "retention_days": 30,
    "max_size_mb": 100,
    "level": "info"
  }
}
```

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端语言 | Go |
| HTTP 服务 | 内置 `net/http` |
| FTP 服务 | 自实现服务器 |
| 静态资源 | Go `embed` 包 |
| 配置管理 | JSON 文件 + AES 加密 |
| 前端 | 原生 HTML/CSS/JavaScript |

## 项目结构

```
pixelbeast-liteserver/
├── main.go                 # 程序入口
├── embed.go                # 静态资源嵌入
├── go.mod                  # Go 模块依赖
├── CLAUDE.md               # Claude AI 助手指南
├── config/                 # 配置文件目录
│   ├── server.json         # 服务配置
│   ├── sites.json          # 站点配置
│   ├── ftp.json            # FTP 配置
│   └── secrets.key         # 加密密钥
├── docs/                   # 项目文档
├── src/                    # 源码目录
│   ├── handlers/           # 协议层 (HTTP/FTP)
│   ├── admin/              # 管理面板后端
│   ├── config/             # 配置管理
│   ├── crypto/             # 加密模块
│   └── static/admin/       # Web 管理界面（嵌入）
├── web/                    # HTTP 运行时根目录
├── ftp/                    # FTP 运行时根目录
└── logs/                   # 日志文件目录
```

## 快速开始

```bash
# 编译
go build -o pixelbeast

# 运行
./pixelbeast -config ./config

# 访问管理面板
# http://localhost:9527/admin
# 默认账号: admin / admin123
```

## 文档索引

| 文档 | 说明 |
|------|------|
| [architecture.md](architecture.md) | 架构设计 |
| [api.md](api.md) | API 文档 |
| [frontend.md](frontend.md) | 前端开发指南 |
| [coding-standards.md](coding-standards.md) | 代码规范 |
| [deployment.md](deployment.md) | 部署指南 |
| [CHANGELOG.md](CHANGELOG.md) | 更新日志 |