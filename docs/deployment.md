# 部署指南

## 开发环境

### 前置要求

- Go 1.21+
- Node.js 14+ (用于 npm 脚本)
- Git

### 启动开发模式

```bash
# 克隆仓库
git clone <repository-url>
cd pixelbeast-liteserver

# 启动开发模式（热重载）
npm run dev

# 访问管理面板
open http://localhost:9527/admin
```

### 开发脚本

```bash
npm run dev          # 开发模式（热重载）
npm run dev:go       # 仅 Go 热重载
npm run build        # 构建当前平台
npm run build:linux  # 构建 Linux x64
npm run build:windows # 构建 Windows x64
npm run build:arm    # 构建 ARM64
npm run clean        # 清理构建文件
```

## 生产构建

### 编译

```bash
# 当前平台
go build -o pixelbeast

# 指定平台
GOOS=linux GOARCH=amd64 go build -o pixelbeast
GOOS=windows GOARCH=amd64 go build -o pixelbeast.exe
GOOS=linux GOARCH=arm64 go build -o pixelbeast
```

### 验证构建

```bash
# 检查版本
./pixelbeast -version

# 测试运行
./pixelbeast -config pixelbeast.json
```

## 生产部署

### Linux 部署

```bash
# 1. 上传文件
scp pixelbeast user@server:/opt/pixelbeast/
scp pixelbeast.json user@server:/opt/pixelbeast/

# 2. 创建必要目录
ssh user@server
cd /opt/pixelbeast
mkdir -p logs web ftp

# 3. 设置权限
chmod +x pixelbeast

# 4. 创建 systemd 服务
sudo tee /etc/systemd/system/pixelbeast.service << EOF
[Unit]
Description=PixelBeast LiteServer
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/pixelbeast
ExecStart=/opt/pixelbeast/pixelbeast -config /opt/pixelbeast/pixelbeast.json
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# 5. 启动服务
sudo systemctl enable pixelbeast
sudo systemctl start pixelbeast

# 6. 查看状态
sudo systemctl status pixelbeast
```

### Windows 部署

```batch
REM 1. 创建目录
mkdir C:\pixelbeast
cd C:\pixelbeast

REM 2. 复制文件
copy pixelbeast.exe C:\pixelbeast\
copy pixelbeast.json C:\pixelbeast\

REM 3. 创建必要目录
mkdir logs web ftp

REM 4. 创建 Windows 服务（使用 NSSM）
nssm install PixelBeast C:\pixelbeast\pixelbeast.exe -config C:\pixelbeast\pixelbeast.json
nssm start PixelBeast
```

### Docker 部署

```dockerfile
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY pixelbeast .
COPY pixelbeast.json .

RUN mkdir -p logs web ftp

EXPOSE 8080 2121 9527

CMD ["./pixelbeast", "-config", "pixelbeast.json"]
```

构建和运行：

```bash
# 构建镜像
docker build -t pixelbeast:latest .

# 运行容器
docker run -d \
  --name pixelbeast \
  -p 8080:8080 \
  -p 2121:2121 \
  -p 9527:9527 \
  -v $(pwd)/web:/app/web \
  -v $(pwd)/ftp:/app/ftp \
  -v $(pwd)/logs:/app/logs \
  pixelbeast:latest
```

## 配置管理

### 配置文件位置

- **开发**: `./pixelbeast.json`
- **生产**: `/opt/pixelbeast/pixelbeast.json`

### 环境变量

```bash
# Go 代理（国内）
export GOPROXY=https://goproxy.cn,direct

# 配置文件路径
./pixelbeast -config /path/to/config.json
```

## 日志管理

### 日志位置

```
logs/
├── http/
│   ├── access.log
│   └── error.log
├── ftp/
│   ├── access.log
│   └── error.log
└── panel/
    ├── access.log
    ├── api.log
    └── auth.log
```

### 日志轮转

使用 logrotate：

```bash
# /etc/logrotate.d/pixelbeast
/opt/pixelbeast/logs/*/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0640 www-data www-data
    sharedscripts
    postrotate
        systemctl reload pixelbeast >/dev/null 2>&1 || true
    endscript
}
```

## 性能优化

### Go 编译优化

```bash
# 减小二进制大小
go build -ldflags="-s -w" -o pixelbeast

# 使用 upx 压缩（可选）
upx --best --lzma pixelbeast
```

### 运行时优化

```json
{
  "http": {
    "readTimeout": 30,
    "writeTimeout": 30,
    "maxConnections": 1000
  },
  "ftp": {
    "maxConnections": 50,
    "timeout": 300
  }
}
```

## 监控

### 健康检查

```bash
# 检查服务状态
curl http://localhost:9527/admin/api/status

# 检查进程
ps aux | grep pixelbeast
```

### 日志监控

```bash
# 实时查看日志
tail -f logs/panel/api.log

# 查看错误日志
tail -f logs/http/error.log
```

## 故障排除

### 常见问题

**端口被占用**
```bash
# 查看端口占用
sudo lsof -i :8080

# 修改配置文件中的端口
```

**权限问题**
```bash
# 确保运行用户有权限访问目录
chown -R www-data:www-data /opt/pixelbeast
```

**服务启动失败**
```bash
# 查看详细日志
sudo journalctl -u pixelbeast -n 50
```

## 备份与恢复

### 备份

```bash
# 备份配置和数据
tar -czf pixelbeast-backup-$(date +%Y%m%d).tar.gz \
  pixelbeast.json \
  web/ \
  ftp/ \
  logs/
```

### 恢复

```bash
# 解压备份
tar -xzf pixelbeast-backup-20240324.tar.gz

# 重启服务
sudo systemctl restart pixelbeast
```
