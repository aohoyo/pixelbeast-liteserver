# 像素兽 (PixelBeast) - Claude AI 助手指南

## 项目概述

像素兽是一个轻量级、高性能的多站点服务器，用 Go 编写。支持 HTTP 静态文件服务、FTP 服务、反向代理和 SSL 证书管理，配备 Web 管理面板。

**核心特点**：
- 单文件部署（静态资源嵌入）
- 内存占用 < 15MB
- 配置文件加密存储敏感信息
- 原生 Web UI（无前端框架）

## 快速开始

```bash
# 编译
go build -o pixelbeast

# 运行
./pixelbeast -config ./config

# 访问管理面板
# http://nas.banayou.com:9527/admin
# 默认账号: admin / admin123
```

## 文档索引

| 文档 | 说明 |
|------|------|
| [docs/overview.md](docs/overview.md) | 项目概述 |
| [docs/architecture.md](docs/architecture.md) | 架构设计 |
| [docs/api.md](docs/api.md) | API 文档 |
| [docs/frontend.md](docs/frontend.md) | 前端开发指南 |
| [docs/coding-standards.md](docs/coding-standards.md) | 代码规范 |
| [docs/deployment.md](docs/deployment.md) | 部署指南 |

## 目录结构

```
pixelbeast-liteserver/
├── main.go              # 主程序入口
├── embed.go             # 静态资源嵌入
├── go.mod               # Go 模块依赖
├── config/              # 配置文件目录
│   ├── server.json      # 服务配置
│   ├── sites.json       # 站点配置
│   ├── ftp.json         # FTP 配置
│   └── secrets.key      # 加密密钥
├── src/
│   ├── admin/           # 管理面板后端
│   ├── handlers/        # HTTP/FTP 协议处理
│   ├── config/          # 配置管理
│   ├── crypto/          # 加密模块
│   └── static/admin/    # Web 管理界面
│       ├── css/         # 样式文件
│       │   ├── base.css     # CSS 变量、重置
│       │   ├── layout.css   # 布局样式
│       │   ├── main.css     # 页面样式
│       │   └── components/  # 组件样式
│       ├── js/          # JavaScript 模块
│       │   ├── core/        # 核心模块
│       │   ├── components/  # UI 组件
│       │   └── tabs/        # 标签页逻辑
│       └── components/  # HTML 组件
├── logs/                # 日志目录
└── docs/                # 文档目录
```

## 技术栈

### 后端
- **语言**: Go 1.21+
- **架构**: 两层架构（协议层 + 业务逻辑层）
- **加密**: AES-256-GCM（敏感配置加密）
- **依赖**: 最小化，优先标准库

### 前端
- **架构**: 组件化 + 模块化 JS（ES Modules）
- **样式**: CSS 变量系统，组件化 CSS
- **模式**: BaseTab 基类 + 单例模式
- **通信**: 事件驱动（globalEvents）

## 开发规范

### Go 代码
```go
// 公开函数：大驼峰
func GetServerManager() *ServerManager {}

// 私有函数：小驼峰
func validateConfig(cfg *config.Config) error {}

// 错误处理：始终检查
data, err := loadData()
if err != nil {
    return fmt.Errorf("加载失败: %w", err)
}
```

### JavaScript 代码
```javascript
// 使用 const/let，不用 var
const API_URL = '/api';
let currentPath = '/';

// 依赖注入模式
export function initTab({ state, api, toast }) {
    // 使用注入的依赖
}

// 事件驱动通信
globalEvents.emit('data:updated', newData);
globalEvents.on('data:updated', (data) => { });
```

### CSS 规范
```css
/* 使用 CSS 变量 */
.card {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
}

/* BEM 命名 */
.card__header { }
.card__body { }
.card--active { }
```

## 配置系统 v2

配置文件位于 `config/` 目录，敏感信息自动加密：

```json
// config/server.json
{
    "http_port": 8080,
    "admin_port": 9527,
    "admin_username": "admin",
    "admin_password": "加密后的密码",
    "admin_path": "/admin"
}
```

**加密字段**：
- `server.json` → `admin_password`
- `ftp.json` → `users[].password`

## 常用命令

```bash
# 运行测试
go test ./...

# 格式化代码
go fmt ./...

# 静态检查
go vet ./...

# 构建
go build -o pixelbeast

# 运行服务
./pixelbeast -config ./config
```

## 关键文件

| 文件 | 职责 |
|------|------|
| `main.go` | 程序入口，服务启动 |
| `src/config/config.go` | 配置管理器 |
| `src/crypto/crypto.go` | AES 加解密 |
| `src/admin/handler.go` | 管理面板路由 |
| `src/handlers/server.go` | 服务管理器 |
| `src/static/admin/js/app.js` | 前端入口 |
| `src/static/admin/js/core/utils.js` | 工具函数 |
| `src/static/admin/js/tabs/BaseTab.js` | 标签页基类 |

## 注意事项

1. **工作目录**: 在 `/home/wwlhlf/project/pixelbeast-liteserver` 下工作
2. **访问地址**: 使用 `nas.banayou.com` 域名（外网/内网通用）
3. **敏感操作**: 需要用户确认（合并 PR、发布版本等）
4. **代码提交**: 遵循 Conventional Commits 规范
5. **CSS 变量优先**: 禁止硬编码颜色值

## 故障排查

- **日志位置**: `./logs/` 目录
- **配置问题**: 检查 `config/` 目录权限
- **密码验证失败**: 检查 `secrets.key` 是否存在
- **端口冲突**: 检查 `server.json` 中的端口配置