# API 文档

## 概述

所有 API 返回统一格式：

```json
{
    "code": 200,
    "message": "success",
    "data": { }
}
```

错误响应：

```json
{
    "code": 400,
    "message": "错误信息",
    "data": null
}
```

---

## 认证

### 登录

```
POST /api/login
```

请求：
```json
{
    "username": "admin",
    "password": "admin123"
}
```

响应：
```json
{
    "code": 200,
    "message": "登录成功",
    "data": {
        "token": "xxx"
    }
}
```

### 登出

```
POST /api/logout
```

---

## 系统状态

### 获取系统状态

```
GET /api/system/status
```

响应：
```json
{
    "code": 200,
    "data": {
        "os": "linux",
        "arch": "amd64",
        "memory_mb": 50,
        "goroutines": 15,
        "uptime": 3600000,
        "server_start_time": "2026-03-29T00:00:00Z",
        "services": {
            "http": { "running": true, "port": 8080 },
            "ftp": { "running": false, "port": 2121 }
        }
    }
}
```

### 获取配置

```
GET /api/config
```

响应：
```json
{
    "code": 200,
    "data": {
        "admin": { "username": "admin", "port": 9527, "path": "/admin" },
        "http": { "port": 8080 },
        "ftp": { "enabled": false, "port": 2121, "root": "./ftp" },
        "log": { "retention_days": 30, "level": "info" }
    }
}
```

### 保存配置

```
POST /api/config
```

请求：
```json
{
    "admin": { "username": "admin", "port": 9527 },
    "http": { "port": 8080 },
    "ftp": { "enabled": true, "port": 2121 }
}
```

---

## 站点管理

### 获取站点列表

```
GET /api/sites
```

响应：
```json
{
    "code": 200,
    "data": {
        "sites": [
            {
                "id": "site-1",
                "name": "我的网站",
                "enabled": true,
                "type": "static",
                "port": 8080,
                "domain": ["example.com"],
                "root": "./www"
            }
        ]
    }
}
```

### 创建站点

```
POST /api/sites
```

请求：
```json
{
    "name": "新站点",
    "type": "static",
    "port": 8081,
    "domain": ["new.example.com"],
    "root": "./new-site"
}
```

### 更新站点

```
PUT /api/sites/{id}
```

### 删除站点

```
DELETE /api/sites/{id}
```

---

## FTP 管理

### 获取 FTP 配置

```
GET /api/ftp
```

响应：
```json
{
    "code": 200,
    "data": {
        "enabled": true,
        "port": 2121,
        "root": "./ftp",
        "users": [
            {
                "username": "user1",
                "root_path": "/user1",
                "status": "enabled",
                "quota": 1024,
                "used_space": 512,
                "expiry_date": "2026-12-31"
            }
        ]
    }
}
```

### 保存 FTP 配置

```
POST /api/ftp
```

### 启动/停止 FTP 服务

```
POST /api/ftp/start
POST /api/ftp/stop
```

### 添加 FTP 用户

```
POST /api/ftp/users
```

请求：
```json
{
    "username": "newuser",
    "password": "password123",
    "root_path": "/newuser",
    "status": "enabled",
    "quota": 512
}
```

### 更新 FTP 用户

```
PUT /api/ftp/users/{username}
```

### 删除 FTP 用户

```
DELETE /api/ftp/users/{username}
```

---

## 文件管理

### 获取目录列表

```
GET /api/files/list?path=/
```

响应：
```json
{
    "code": 200,
    "data": {
        "path": "/",
        "items": [
            {
                "name": "folder",
                "type": "dir",
                "size": 0,
                "mod_time": "2026-03-29T00:00:00Z"
            },
            {
                "name": "file.txt",
                "type": "file",
                "size": 1024,
                "mod_time": "2026-03-29T00:00:00Z"
            }
        ]
    }
}
```

### 创建目录

```
POST /api/files/mkdir
```

请求：
```json
{
    "path": "/new-folder"
}
```

### 删除文件/目录

```
DELETE /api/files?path=/file.txt
```

### 重命名

```
POST /api/files/rename
```

请求：
```json
{
    "old_path": "/old-name",
    "new_path": "/new-name"
}
```

### 上传文件

```
POST /api/files/upload
Content-Type: multipart/form-data
```

### 下载文件

```
GET /api/files/download?path=/file.txt
```

---

## 日志

### 获取日志列表

```
GET /api/logs
```

响应：
```json
{
    "code": 200,
    "data": {
        "files": [
            {
                "name": "server.log",
                "size": 10240,
                "mod_time": "2026-03-29T00:00:00Z"
            }
        ]
    }
}
```

### 读取日志内容

```
GET /api/logs/{filename}?lines=100
```

### 清理日志

```
POST /api/logs/cleanup
```

---

## 分享

### 创建分享链接

```
POST /api/share
```

请求：
```json
{
    "path": "/shared-file.txt",
    "expire_hours": 24,
    "password": "optional-password"
}
```

响应：
```json
{
    "code": 200,
    "data": {
        "token": "abc123",
        "url": "/s/abc123"
    }
}
```

### 获取分享列表

```
GET /api/share
```

### 删除分享

```
DELETE /api/share/{token}
```

---

## 错误码

| Code | 说明 |
|------|------|
| 200 | 成功 |
| 400 | 请求错误 |
| 401 | 未授权 |
| 403 | 禁止访问 |
| 404 | 资源不存在 |
| 500 | 服务器错误 |