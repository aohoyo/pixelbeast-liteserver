# 部署指南

## 系统要求

- **操作系统**: Linux / macOS / Windows
- **Go 版本**: 1.25+
- **内存**: 最低 64MB
- **磁盘**: 最低 100MB

---

## 编译

### 本地编译

```bash
# 编译
go build -o pixelbeast

# 查看版本
./pixelbeast -version
```

### 交叉编译

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o pixelbeast-linux

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o pixelbeast-arm64

# Windows
GOOS=windows GOARCH=amd64 go build -o pixelbeast.exe

# macOS
GOOS=darwin GOARCH=amd64 go build -o pixelbeast-mac
```

---

## 配置

### 初始化

首次运行自动创建默认配置：

```bash
./pixelbeast -config ./config
```

生成的文件：
- `config/server.json` - 服务配置
- `config/sites.json` - 默认站点 (端口 3380)
- `config/ftp.json` - FTP 配置 (默认关闭)
- `config/secrets.key` - AES-256-GCM 加密密钥

### server.json 结构

```json
{
    "name": "PixelBeast Server",
    "timezone": "Asia/Shanghai",
    "admin": {
        "port": 9527,
        "username": "admin",
        "password": "加密后的密码",
        "path": "/admin",
        "domain": "",
        "ssl_enabled": false
    },
    "directories": {
        "sites": "./sites",
        "ftp": "./ftp",
        "backup": "./backups"
    },
    "backup": {
        "auto_enabled": true,
        "schedule": "daily",
        "retention": 3,
        "items": ["config", "sites", "ftp"]
    },
    "log": {
        "retention_days": 30,
        "max_size_mb": 100,
        "compress_days": 7,
        "cleanup_hour": 3,
        "level": "info"
    }
}
```

---

## 运行

### 直接运行

```bash
./pixelbeast -config ./config
```

### 后台运行

```bash
# nohup
nohup ./pixelbeast -config ./config > pixelbeast.log 2>&1 &

# tmux
tmux new -s pixelbeast
./pixelbeast -config ./config
# Ctrl+B+D 分离
```

### Systemd 服务

创建 `/etc/systemd/system/pixelbeast.service`：

```ini
[Unit]
Description=PixelBeast Server
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/pixelbeast
ExecStart=/opt/pixelbeast/pixelbeast -config /opt/pixelbeast/config
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload
systemctl start pixelbeast
systemctl enable pixelbeast
systemctl status pixelbeast
```

---

## 访问

| 服务 | 地址 |
|------|------|
| 管理面板 | `http://nas.banayou.com:9527/admin` |
| 默认站点 | `http://nas.banayou.com:3380` |
| FTP | `ftp://nas.banayou.com:2121` |

默认账号: `admin` / `admin123`

---

## 安全配置

### 修改管理员密码

管理面板 → 系统设置 → 修改管理员密码

### 修改安全入口

```json
{ "admin": { "path": "/my-admin-portal" } }
```

访问地址变为: `http://nas.banayou.com:9527/my-admin-portal`

### 域名绑定

```json
{ "admin": { "domain": "nas.banayou.com" } }
```

仅允许该域名访问管理面板。

### 防火墙

```bash
ufw allow 9527/tcp  # 管理面板
ufw allow 3380/tcp  # 默认站点
ufw allow 2121/tcp  # FTP
```

---

## 日志

### 日志位置

```
log/
├── server.log      # 面板服务日志
├── auth.log        # 认证日志 (登录/登出)
├── api.log         # API 操作日志
├── http/
│   ├── access.log  # HTTP 访问日志
│   └── error.log   # HTTP 错误日志
└── ftp/
    ├── access.log  # FTP 访问日志
    └── error.log   # FTP 错误日志
```

---

## 故障排查

### 端口被占用

```bash
netstat -tlnp | grep 9527
# 修改 config/server.json 中的 admin.port
```

### 配置加载失败

```bash
ls -la config/
# 确认目录权限 755，文件权限 644
```

### 密码验证失败

```bash
rm config/secrets.key
# 重新启动，会生成新密钥和默认密码 admin/admin123
```

### 前端资源 404

```bash
go build -o pixelbeast  # 重新编译嵌入前端资源
```

---

## 更新

```bash
cp -r config config.bak          # 1. 备份配置
systemctl stop pixelbeast         # 2. 停止服务
cp pixelbeast-new /opt/pixelbeast/pixelbeast  # 3. 替换二进制
systemctl start pixelbeast        # 4. 启动服务
systemctl status pixelbeast       # 5. 检查状态
```
