# 像素兽 (PixelBeast)

🪶 小而强悍，无所不能 - 轻量级多站点服务器

## 特性

- **多站点管理** - 支持静态站点和反向代理，域名绑定，独立端口
- **SSL 证书** - Let's Encrypt 自动申请/续签（HTTP-01/DNS-01），自定义证书导入
- **FTP 服务** - 独立 FTP 服务器，支持多用户，密码加密存储
- **文件管理** - 在线编辑、上传下载、压缩解压、分享链接
- **Web 管理面板** - 暗色主题 UI，现代化响应式设计
- **系统监控** - CPU、内存、磁盘实时监控
- **单文件部署** - 静态资源嵌入，无需外部依赖
- **配置加密** - 敏感信息 AES-256-GCM 加密存储

## 快速开始

### 环境要求

- [Go](https://go.dev/dl/) 1.21+

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
默认账号: admin / admin123（首次登录后请修改）
```

## 目录结构

```
pixelbeast-liteserver/
├── pixelbeast               # 编译后的二进制文件
├── config/                  # 配置文件目录（运行时生成）
│   ├── server.json          # 服务配置（端口、日志、管理账号）
│   ├── sites.json           # 站点配置
│   ├── ftp.json             # FTP 配置
│   └── secrets.key          # AES 加密密钥（自动生成）
├── src/
│   ├── cmd/main.go          # 程序入口
│   ├── panel/               # 管理面板（API、路由、中间件）
│   ├── site/                # 站点服务（虚拟主机、反向代理）
│   ├── ssl/                 # SSL 证书管理（ACME、Lego）
│   ├── config/              # 配置管理
│   ├── logger/              # 日志系统
│   ├── monitor/             # 系统监控
│   ├── file/                # 文件操作
│   ├── ftp/                 # FTP 服务器
│   ├── backup/              # 备份管理
│   ├── crypto/              # 加密工具
│   └── static/admin/        # 前端界面
└── docs/                    # 文档
```

## 配置

### 服务配置 (config/server.json)

```json
{
    "http_port": 8080,
    "admin_port": 9527,
    "admin_username": "admin",
    "admin_password": "AES-256-GCM 加密后的密码",
    "admin_path": "/admin"
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
            "port": 8080,
            "domain": ["example.com"],
            "root": "./www",
            "index_files": ["index.html"],
            "auto_index": true,
            "ssl": {
                "enabled": false,
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
    "root": "./ftp",
    "users": [
        {
            "username": "user1",
            "password": "AES-256-GCM 加密后的密码",
            "root_path": "/user1",
            "status": "enabled"
        }
    ]
}
```

## API 文档

详见 [docs/api.md](docs/api.md)

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
```

## 技术栈

- **后端**: Go 1.25+（标准库优先）
- **前端**: 原生 HTML/CSS/JS (ES Modules)
- **加密**: AES-256-GCM
- **SSL**: Lego ACME 客户端
- **监控**: gopsutil

## 安全特性

- 路径遍历防护（resolvePath 限制根目录）
- CSRF 中间件（所有 POST/PUT/DELETE 需 Token）
- 登录速率限制（5 次失败锁定 10 分钟）
- 敏感信息加密存储（AES-256-GCM）
- 配置文件权限 0600（仅所有者可读写）
- XSS 防护（HTML 转义 + 模板引擎 escapeHtml）

## License

MIT
