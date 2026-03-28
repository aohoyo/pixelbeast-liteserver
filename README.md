# 像素兽 (PixelBeast)

🪶 小而强悍，无所不能 - 轻量级多站点服务器

## 特性

- **多站点管理** - 支持静态站点和反向代理
- **FTP 服务** - 独立 FTP 服务器，支持多用户
- **SSL 证书** - 自动 HTTPS 证书管理
- **Web 管理面板** - 现代化暗色主题 UI
- **单文件部署** - 静态资源嵌入，无需外部依赖
- **配置加密** - 敏感信息 AES-256-GCM 加密存储

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

## 目录结构

```
pixelbeast-liteserver/
├── main.go              # 主程序入口
├── embed.go             # 静态资源嵌入
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
└── docs/                # 文档目录
```

## 配置

### 服务配置 (config/server.json)

```json
{
    "http_port": 8080,
    "admin_port": 9527,
    "admin_username": "admin",
    "admin_password": "加密后的密码",
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
            "auto_index": true
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
            "password": "加密后的密码",
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
# 运行测试
go test ./...

# 格式化代码
go fmt ./...

# 静态检查
go vet ./...
```

## 技术栈

- **后端**: Go 1.21+
- **前端**: 原生 HTML/CSS/JS (ES Modules)
- **加密**: AES-256-GCM
- **依赖**: 最小化，优先标准库

## License

MIT