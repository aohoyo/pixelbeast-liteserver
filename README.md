# 像素兽 (PixelBeast)

🪶 小而强悍，无所不能 - 轻量级多站点服务器管理面板

一个用 Go 编写的轻量级 Web 服务器与运维管理面板（类似宝塔面板精简版）。单文件部署、标准库优先，集站点管理、SSL 证书、FTP、文件管理、Web 终端、系统监控于一体。

## 特性

- **多站点管理** - 静态站点 + 反向代理，域名绑定、独立端口、共享端口域名路由
- **SSL 证书** - Let's Encrypt 自动申请/续签（HTTP-01 自动/文件/DNS-01），支持阿里云/腾讯云/宝塔 DNS，自定义证书导入
- **Web 终端** - 浏览器内 PTY 终端（基于 xterm.js + creack/pty），支持脚本执行、窗口大小调整
- **文件管理** - 在线编辑（CodeMirror 6）、上传下载、分块上传、压缩解压（zip/tar/7z）、权限管理、分享链接、回收站
- **FTP 服务** - 独立 FTP 服务器，多用户、限速/配额/连接数限制、密码加密存储
- **Web 管理面板** - 暗色主题 UI，组件化 + ES Modules，响应式设计
- **系统监控** - CPU/内存/Swap/磁盘/网络/磁盘IO 实时监控，内存释放、系统清理
- **告警通知** - 健康检查、SSL 证书到期告警，多渠道通知（飞书/邮件/Browser）
- **日志系统** - 多分类（Server/Auth/API）、轮转、压缩、在线查看/导出
- **备份管理** - 配置/站点/FTP 多目录打包备份、下载、恢复
- **开机自启** - systemd / Windows 服务注册与检测
- **单文件部署** - 静态资源嵌入二进制，无外部依赖
- **配置加密** - 敏感信息 AES-256-GCM 加密存储，管理员密码 bcrypt 哈希

## 快速开始

### 环境要求

- [Go](https://go.dev/dl/) 1.25+

> 国内用户建议设置代理加速（只需一次）：
> ```bash
> go env -w GOPROXY=https://goproxy.cn,direct
> ```

### 开发运行

```bash
# 直接运行（无需编译，适合开发调试）
go run ./src/cmd
```

### 编译

```bash
# 普通编译
go build -o pixelbeast ./src/cmd

# 压缩编译（去掉调试信息，体积减小约 30%）
go build -ldflags "-s -w" -o pixelbeast ./src/cmd

# 运行
./pixelbeast
```

### 热重载开发（Air）

修改代码后自动重新编译运行，不用手动重启。

```bash
# 1. 安装 air（只需一次）
go install github.com/air-verse/air@latest

# 2. 在项目根目录运行
air
```

首次运行会自动生成 `.air.toml` 配置文件。之后每次修改 `.go` 文件保存后会自动重新编译。

### 交叉编译

```bash
# Windows
cd src/cmd && GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o ../../pixelbeast.exe .

# Linux ARM（如树莓派、NAS）
cd src/cmd && GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o ../../pixelbeast .

# macOS Intel
cd src/cmd && GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o ../../pixelbeast .

# macOS Apple Silicon
cd src/cmd && GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o ../../pixelbeast .
```

### 访问管理面板

```
http://localhost:9527/admin
```

首次启动时**初始密码随机生成**，仅在终端输出一次（不写入磁盘），首次登录后强制修改：

```
🦖 管理面板: http://localhost:9527/admin
  账号: admin
  密码: <随机生成的初始密码>
```

## 目录结构

```
pixelbeast/
├── pixelbeast               # 编译后的二进制文件
├── config/                  # 配置文件目录（运行时生成）
│   ├── server.json          # 服务配置（端口、日志、管理账号、目录）
│   ├── sites.json           # 站点配置
│   ├── ftp.json             # FTP 配置
│   ├── dns_providers.json   # DNS 服务商凭证（加密存储）
│   └── secrets.key          # AES 加密密钥（自动生成，权限 0600）
├── ssl/                     # SSL 证书存储目录（.gitignore 忽略）
├── src/
│   ├── cmd/main.go          # 程序入口（只做组装和启动）
│   ├── embed.go             # 静态资源嵌入
│   ├── panel/               # 管理面板（API、路由、中间件）
│   │   ├── handler.go       # 会话、认证、静态资源、FTP 服务管理
│   │   ├── router.go        # 路由表
│   │   ├── middleware.go     # 中间件链（Auth/Recovery/CSRF/Logging/SecurityHeaders）
│   │   ├── api_system.go    # 系统监控、清理、自启
│   │   ├── api_site.go      # 站点管理
│   │   ├── api_ssl.go       # 证书申请/续签/导入/DNS
│   │   ├── api_ftp.go       # FTP 服务 + 用户 + 文件管理
│   │   ├── api_file.go      # 文件 + 压缩 + 分享 + 脚本执行
│   │   ├── api_trash.go     # 回收站（删除/恢复/清空）
│   │   ├── api_terminal.go  # Web 终端（WebSocket + PTY）
│   │   ├── api_config.go    # 配置管理
│   │   ├── api_backup.go    # 备份管理
│   │   ├── api_log.go       # 日志管理
│   │   └── api_service.go   # 自启动服务管理
│   ├── site/                # 站点服务（虚拟主机、反向代理、静态文件）
│   ├── ssl/                 # SSL 证书核心（ACME、Lego、自动续签）
│   ├── ftp/                 # FTP 服务器
│   ├── notify/              # 告警通知（健康检查、SSL 到期、多渠道）
│   ├── config/              # 配置管理（JSON + AES/bcrypt 加密）
│   ├── crypto/              # 加密工具
│   ├── logger/              # 日志系统（多分类、轮转、压缩）
│   ├── monitor/             # 系统监控（内存、CPU、磁盘、网络）
│   ├── file/                # 文件操作（管理、压缩、安全检查）
│   ├── backup/              # 备份管理
│   └── static/admin/        # 前端界面（CSS / JS / views）
└── docs/                    # 文档（API、CHANGELOG、规范、路线图）
```

## 配置

### 服务配置 (config/server.json)

```json
{
    "name": "PixelBeast Server",
    "timezone": "Asia/Shanghai",
    "admin": {
        "port": 9527,
        "username": "admin",
        "password": "bcrypt 哈希后的密码",
        "path": "/admin",
        "domain": "",
        "ssl_enabled": false,
        "require_password_change": false
    },
    "directories": {
        "sites": "./www",
        "ftp": "./ftp",
        "backup": "./backups"
    },
    "log": {
        "retention_days": 30,
        "max_size_mb": 100,
        "compress_days": 7,
        "cleanup_hour": 3,
        "level": "info"
    },
    "auto_start": { "enabled": true }
}
```

### 站点配置 (config/sites.json)

```json
{
    "sites": [
        {
            "id": "site-1",
            "name": "我的网站",
            "enabled": true,
            "type": "static",
            "port": 3380,
            "domain": ["example.com"],
            "root": "./www/site-1",
            "index_files": ["index.html"],
            "auto_index": true,
            "ssl": {
                "enabled": false,
                "auto_https": true,
                "force_https": true,
                "hsts": true
            }
        }
    ]
}
```

### FTP 配置 (config/ftp.json)

```json
{
    "enabled": true,
    "port": 2121,
    "users": [
        {
            "username": "user1",
            "password": "bcrypt 哈希后的密码",
            "root_path": "/user1",
            "status": "enabled",
            "quota": 1024,
            "speed_limit": 0,
            "max_connections": 5
        }
    ]
}
```

## API 文档

详见 [docs/api.md](docs/api.md)，更新日志见 [docs/CHANGELOG.md](docs/CHANGELOG.md)。

## 开发

```bash
# 编译（NAS 环境需限制内存）
cd src/cmd && GOPROXY=https://goproxy.cn,direct GOMEMLIMIT=300MiB go build -o ../../pixelbeast .

# 运行测试
cd src/cmd && go test ./...

# 静态检查
cd src/cmd && go vet ./...

# 格式化
go fmt ./...

# 编译验证（不产出文件）
cd src/cmd && GOPROXY=https://goproxy.cn,direct go build -o /dev/null . && go vet ./...
```

开发规范参见 [docs/coding-standards.md](docs/coding-standards.md)。

## 技术栈

- **后端**: Go 1.25+（标准库优先）
- **ACME**: [lego](https://github.com/go-acme/lego) v4
- **监控**: [gopsutil](https://github.com/shirou/gopsutil) v4
- **终端**: [creack/pty](https://github.com/creack/pty) + [gorilla/websocket](https://github.com/gorilla/websocket)
- **加密**: AES-256-GCM + bcrypt
- **前端**: 原生 HTML/CSS/JS (ES Modules)，[CodeMirror 6](https://codemirror.net/) 编辑器

## 安全特性

- **密码存储**: 管理员/FTP 密码使用 bcrypt 哈希，敏感凭证 AES-256-GCM 加密
- **初始密码**: 首次启动随机生成，仅终端输出一次，首次登录强制修改
- **路径遍历防护**: resolvePath 使用 filepath.Rel 校验，限制根目录
- **CSRF 中间件**: 所有 POST/PUT/DELETE 需 X-CSRF-Token（支持多标签页）
- **WebSocket Origin 校验**: 终端 WebSocket 严格校验同源
- **XSS 防护**: HTML 转义 + 模板引擎 escapeHtml + 安全响应头
- **登录速率限制**: 5 次失败锁定 10 分钟
- **配置文件权限 0600**: 仅所有者可读写
- **安全响应头**: X-Frame-Options、X-Content-Type-Options 等
- **API 错误脱敏**: 不向前端暴露内部错误细节

## License

MIT
