# 部署指南

## 系统要求

- **操作系统**: Linux / macOS / Windows
- **Go 版本**: 1.21+
- **内存**: 最低 64MB
- **磁盘**: 最低 100MB

---

## 编译

### 本地编译

```bash
# 进入项目目录
cd /home/wwlhlf/project/pixelbeast-liteserver

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

### 配置目录

```bash
mkdir -p config
```

### 初始化配置

首次运行会自动创建默认配置：

```bash
./pixelbeast -config ./config
```

生成的文件：
- `config/server.json` - 服务配置
- `config/sites.json` - 站点配置
- `config/ftp.json` - FTP 配置
- `config/secrets.key` - 加密密钥

### 配置文件说明

#### server.json

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
        "compress_days": 7,
        "cleanup_hour": 3,
        "level": "info"
    },
    "ftp_dir": "./ftp",
    "backup_dir": "./backups"
}
```

#### sites.json

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

#### ftp.json

```json
{
    "enabled": false,
    "port": 2121,
    "root": "./ftp",
    "users": []
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

# screen
screen -S pixelbeast
./pixelbeast -config ./config
# Ctrl+A+D 分离

# tmux
tmux new -s pixelbeast
./pixelbeast -config ./config
# Ctrl+B+D 分离
```

### Systemd 服务

创建服务文件 `/etc/systemd/system/pixelbeast.service`：

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

启动服务：

```bash
# 重载配置
systemctl daemon-reload

# 启动
systemctl start pixelbeast

# 开机自启
systemctl enable pixelbeast

# 查看状态
systemctl status pixelbeast
```

---

## 访问

### 管理面板

```
http://nas.banayou.com:9527/admin
```

默认账号：
- 用户名: `admin`
- 密码: `admin123`

### 网站服务

```
http://nas.banayou.com:8080
```

### FTP 服务

```
ftp://nas.banayou.com:2121
```

---

## 安全配置

### 修改管理员密码

1. 登录管理面板
2. 进入「系统设置」
3. 修改管理员密码

### 修改安全入口

```json
// config/server.json
{
    "admin_path": "/my-admin-portal"
}
```

访问地址变为：`http://nas.banayou.com:9527/my-admin-portal`

### 防火墙配置

```bash
# 开放端口
ufw allow 9527/tcp  # 管理面板
ufw allow 8080/tcp  # HTTP 服务
ufw allow 2121/tcp  # FTP 服务（如启用）
```

### SSL 配置

```json
// config/sites.json
{
    "sites": [{
        "ssl": {
            "enabled": true,
            "auto_https": true,
            "email": "admin@example.com"
        }
    }]
}
```

---

## 备份与恢复

### 备份

```bash
# 备份配置
tar -czvf pixelbeast-config.tar.gz config/

# 备份数据
tar -czvf pixelbeast-data.tar.gz www/ ftp/ logs/
```

### 恢复

```bash
# 解压配置
tar -xzvf pixelbeast-config.tar.gz

# 解压数据
tar -xzvf pixelbeast-data.tar.gz
```

---

## 监控与日志

### 日志位置

```
logs/
├── server.log      # 服务日志
├── access.log      # 访问日志
└── error.log       # 错误日志
```

### 日志配置

```json
{
    "log": {
        "retention_days": 30,
        "max_size_mb": 100,
        "compress_days": 7,
        "level": "info"
    }
}
```

### 监控

管理面板首页提供：
- CPU 使用率
- 内存使用量
- 协程数量
- 运行时间
- 服务状态

---

## 故障排查

### 端口被占用

```bash
# 查看端口占用
netstat -tlnp | grep 9527

# 修改端口
# config/server.json
{
    "admin_port": 9528
}
```

### 配置加载失败

```bash
# 检查配置文件
cat config/server.json

# 检查权限
ls -la config/

# 检查密钥
ls -la config/secrets.key
```

### 密码验证失败

```bash
# 删除密钥（会重置所有密码）
rm config/secrets.key

# 重新启动，会生成新密钥和默认密码
./pixelbeast -config ./config
```

### 内存占用过高

```bash
# 查看内存
ps aux | grep pixelbeast

# 调整日志级别
# config/server.json
{
    "log": {
        "level": "error"
    }
}
```

---

## 更新

### 更新步骤

```bash
# 1. 备份配置
cp -r config config.bak

# 2. 停止服务
systemctl stop pixelbeast

# 3. 替换二进制
cp pixelbeast-new /opt/pixelbeast/pixelbeast

# 4. 启动服务
systemctl start pixelbeast

# 5. 检查状态
systemctl status pixelbeast
```