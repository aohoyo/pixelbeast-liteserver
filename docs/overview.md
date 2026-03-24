# 项目概述

**像素兽 轻量服务器** (PixelBeast LiteServer) 是一个用 Go 编写的超轻量级跨平台 Web+FTP 服务器。设计理念是"小而强悍，无所不能"。

## 核心特性

- **内存占用**：8-15MB RAM
- **单文件部署**：依赖极少，下载即用
- **跨平台**：Windows、Linux、ARM64 从单一代码库构建
- **功能集成**：HTTP + FTP + Web 管理界面

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
| FTP | `flash` | `flash` |

### 配置文件示例

```json
{
  "global": {
    "adminPort": 9527
  },
  "admin": {
    "enabled": true,
    "path": "/admin",
    "username": "admin",
    "password": "admin123"
  },
  "ftp": {
    "enabled": true,
    "port": 2121,
    "root": "./ftp",
    "users": [
      {"username": "flash", "password": "flash"}
    ]
  },
  "sites": []
}
```

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端语言 | Go |
| HTTP 服务 | 内置 `net/http` |
| FTP 服务 | 自实现服务器 |
| 静态资源 | Go `embed` 包 |
| 配置管理 | JSON 文件 |
| 前端 | 原生 HTML/CSS/JavaScript |

## 项目结构

```
pixelbeast-liteserver/
├── main.go                 # 程序入口
├── pixelbeast.json         # 运行时配置
├── package.json            # npm 脚本
├── dev.sh                  # 开发模式启动脚本
├── .air.toml               # Air 热重载配置
├── CLAUDE.md               # Claude Code 文档索引
├── docs/                   # 项目文档
├── src/                    # 源码目录
│   ├── handlers/           # 协议层 (HTTP/FTP)
│   ├── admin/              # 管理面板 handlers
│   ├── static/admin/       # Web 管理界面（嵌入）
│   ├── services/           # 业务逻辑层
│   └── config/             # 配置管理
├── web/                    # HTTP 运行时根目录
├── ftp/                    # FTP 运行时根目录
└── logs/                   # 日志文件目录
```
