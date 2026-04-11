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

### 认证方式

所有 `/api/*` 接口（除 login/logout）需要 Cookie 中的 `session_id`。

---

## 认证

### 登录

```
POST /api/login
Content-Type: application/json

{ "username": "admin", "password": "admin123" }
```

### 登出

```
POST /api/logout
```

---

## 系统状态

### 获取系统状态

```
GET /api/status
```

返回服务运行状态（HTTP/FTP 端口、是否运行）。

### 获取系统监控

```
GET /api/system/status
```

返回详细系统信息：CPU 每核使用率、内存/Swap、多磁盘分区、网络速率/流量、磁盘 I/O、进程数、运行时长。

### 释放内存

```
POST /api/system/free-memory
```

### 扫描可清理项

```
POST /api/system/cleanup-scan
```

### 执行清理

```
POST /api/system/cleanup
```

### 同步系统时间

```
POST /api/system/time/sync
```

### 重启服务器

```
POST /api/system/restart
```

### 检查更新

```
GET /api/system/check-update
```

---

## 配置管理

### 获取配置

```
GET /api/config
```

### 保存配置

```
POST /api/config/save
Content-Type: application/json

{
    "name": "PixelBeast Server",
    "timezone": "Asia/Shanghai",
    "admin": { "port": 9527, "username": "admin", "path": "/admin" },
    "directories": { "sites": "./sites", "ftp": "./ftp", "backup": "./backups" },
    "backup": { "auto_enabled": true, "schedule": "daily", "retention": 3 },
    "log": { "retention_days": 30, "level": "info" }
}
```

### 重置配置

```
POST /api/config/reset
```

重置服务配置为默认值，保留站点和 FTP 用户数据。

---

## 备份管理

### 列出备份

```
GET /api/backups
```

### 创建备份

```
POST /api/backups/create
```

### 删除备份

```
POST /api/backups/delete
```

### 下载备份

```
GET /api/backups/download
```

### 恢复备份

```
POST /api/backups/restore
```

---

## 站点管理

### 获取站点列表

```
GET /api/sites
```

### 创建/更新站点

```
POST /api/sites                    # 创建
PUT  /api/sites/{id}               # 更新
```

站点类型：`static`（静态文件）、`proxy`（反向代理）。

```json
{
    "name": "我的网站",
    "type": "static",
    "port": 8080,
    "domain": ["example.com"],
    "root": "./www",
    "index_files": ["index.html"],
    "auto_index": true,
    "proxy": { "target": "", "websocket": false },
    "ssl": { "enabled": false }
}
```

### 删除站点

```
DELETE /api/sites/{id}
```

### 站点启停

```
POST /api/sites/toggle             # 切换启用/禁用
POST /api/sites/start              # 启动站点
POST /api/sites/stop               # 停止站点
POST /api/sites/restart            # 重启站点
```

### 批量操作

```
POST /api/sites/batch
{ "action": "enable|disable|delete", "ids": ["id1", "id2"] }
```

### 站点服务控制

```
POST /api/service/sites/toggle     # 切换站点服务
POST /api/service/sites/start      # 启动
POST /api/service/sites/stop       # 停止
POST /api/service/sites/restart    # 重启
POST /api/service/sites/reload     # 重载配置
GET   /api/sites/status            # 获取状态
```

---

## FTP 管理

### 获取 FTP 状态

```
GET /api/ftp/status
```

### FTP 服务控制

```
POST /api/service/ftp/toggle       # 切换
POST /api/service/ftp/start        # 启动
POST /api/service/ftp/stop         # 停止
POST /api/service/ftp/restart      # 重启
POST /api/service/ftp/reload       # 重载配置
```

### FTP 用户管理

```
GET    /api/ftp/users              # 列表
POST   /api/ftp/users/add          # 添加
PUT    /api/ftp/users/{username}   # 更新
DELETE /api/ftp/users/{username}   # 删除
POST   /api/ftp/users/toggle       # 切换状态
POST   /api/ftp/users/batch        # 批量操作
```

### FTP 端口

```
POST /api/ftp/port
{ "port": 2121 }
```

### FTP 文件管理

```
GET    /api/ftp/files              # 目录列表
POST   /api/ftp/files/upload       # 上传
DELETE /api/ftp/files/delete       # 删除
POST   /api/ftp/files/mkdir        # 创建目录
GET    /api/ftp/files/download     # 下载
POST   /api/ftp/files/rename       # 重命名
POST   /api/ftp/files/copy         # 复制
```

---

## 文件管理

### 目录列表

```
GET /api/files?path=/
```

### 快捷目录

```
GET /api/files/quick-dirs
```

### 上传

```
POST /api/files/upload/chunk       # 分块上传
POST /api/files/upload/merge       # 合并分块
GET  /api/files/upload/status      # 上传状态
POST /api/files/upload/path        # 指定路径上传
```

### 文件操作

```
DELETE /api/files/delete           # 删除
POST   /api/files/mkdir            # 创建目录
POST   /api/files/rename           # 重命名
GET    /api/files/download         # 下载
POST   /api/files/copy             # 复制
POST   /api/files/move             # 移动
POST   /api/files/touch            # 创建空文件
POST   /api/files/chmod            # 修改权限
GET    /api/files/permissions      # 获取权限
POST   /api/files/read             # 读取文件内容
POST   /api/files/save             # 保存文件内容
```

### 压缩/解压

```
POST /api/files/compress           # 压缩 (zip/tar.gz)
POST /api/files/extract            # 解压
```

### 分享

```
POST   /api/files/share            # 创建分享
GET    /api/files/share/list       # 分享列表
DELETE /api/files/share/delete     # 删除分享
```

### 公开分享下载（无需认证）

```
GET /s/{token}
GET /share/{token}
```

---

## 日志管理

```
GET    /api/logs                   # 日志文件列表
GET    /api/logs/read              # 读取日志内容 (?file=server&date=2026-04-10&lines=100)
GET    /api/logs/stats             # 日志统计
GET    /api/logs/download          # 下载日志
POST   /api/logs/clear             # 清除日志
POST   /api/logs/bulk-clear        # 批量清除
POST   /api/logs/bulk-export       # 批量导出
GET    /api/logs/config            # 日志配置
POST   /api/logs/config            # 更新日志配置
```

日志文件分类：
- 面板日志: `server.log`, `auth.log`, `api.log`
- HTTP 日志: `http/access.log`, `http/error.log`
- FTP 日志: `ftp/access.log`, `ftp/error.log`

---

## 错误码

| Code | 说明 |
|------|------|
| 200 | 成功 |
| 400 | 请求错误 |
| 401 | 未授权 |
| 403 | 禁止访问 |
| 404 | 资源不存在 |
| 429 | 请求过多（登录锁定） |
| 500 | 服务器错误 |
