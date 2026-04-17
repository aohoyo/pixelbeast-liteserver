# 配置结构重构计划

## 目标

简化配置结构：去掉 `http` 全局配置，新增统一的 `directories` 和 `backup` 配置节，首次启动自动创建默认站点。

## 新配置结构

```json
{
    "admin": {
        "port": 9527,
        "path": "/admin",
        "username": "admin",
        "password": "***",
        "bind_domain": "",
        "ssl_enabled": false
    },
    "directories": {
        "sites": "./data/sites",
        "ftp": "./ftp",
        "backup": "./backups"
    },
    "ftp": {
        "enabled": false,
        "port": 2121,
        "users": []
    },
    "log": {
        "retention_days": 30,
        "max_size_mb": 100,
        "compress_days": 7,
        "cleanup_hour": 3,
        "level": "info"
    },
    "backup": {
        "auto_enabled": true,
        "schedule": "daily",
        "retention": 7,
        "items": ["config", "sites", "ftp"]
    },
    "sites": [
        {
            "id": "default",
            "name": "默认站点",
            "port": 3080,
            "root": "./data/sites/default",
            ...
        }
    ]
}
```

### 变更对比

| 旧字段 | 新字段 | 说明 |
|--------|--------|------|
| `http.port` | 删除 | 改用默认站点的端口 |
| `http.root` | `directories.sites` | 统一到目录设置 |
| `ftp.root` | `directories.ftp` | 统一到目录设置 |
| `backup_dir` | `directories.backup` | 统一到目录设置 |
| (无) | `backup.items` | 选择备份内容 |
| (无) | `backup.schedule` | 备份策略 |
| (无) | `backup.retention` | 备份保留份数 |

### 共享端口逻辑变更

- **旧**: `http.port` 作为共享端口，`site.port=0` 的站点跑在上面
- **新**: 第一个站点（默认站点）的端口即为共享端口，`site.port=0` 仍使用该端口

---

## 实施步骤

### 第 1 步：Go 后端 — config.go 结构体重构

1. **新增结构体**:
   ```go
   type DirectoriesConfig struct {
       Sites  string `json:"sites"`
       FTP    string `json:"ftp"`
       Backup string `json:"backup"`
   }

   type BackupConfig struct {
       AutoEnabled bool     `json:"auto_enabled"`
       Schedule    string   `json:"schedule"`    // daily, weekly, monthly
       Retention   int      `json:"retention"`   // 保留份数
       Items       []string `json:"items"`       // config, sites, ftp
   }
   ```

2. **修改 ServerConfig**: 删除 `HTTPPort`、`HTTPDir`、`FTPDir`、`BackupDir`，新增 `Directories DirectoriesConfig` 和 `Backup BackupConfig`

3. **修改 FTPConfig**: 删除 `Root` 字段（移到 `Directories.FTP`）

4. **更新 `defaultServerConfig()`**: 使用新结构

5. **更新 `defaultSitesConfig()`**: 端口来自默认站点（3080），根目录来自 `Directories.Sites`

6. **添加配置迁移逻辑**: 在 `loadServer()` 中检测旧字段（`http_port`、`http_dir`）并自动迁移到新结构

### 第 2 步：Go 后端 — api.go API 适配

1. **`getConfig()`**: 返回 `directories`、`backup` 替代 `http`、`backup_dir`
2. **`saveConfig()`**: 解析 `directories`、`backup` 配置
3. **`resetConfig()`**: 使用新的默认配置结构
4. **`getStatus()`**: 移除 `ftp_root`、`ftp_dir`、`backup_dir`，改用 `directories`

### 第 3 步：Go 后端 — server.go 共享端口适配

1. **`NewServerManager()`**: 从第一个站点获取共享端口，不再用 `cm.Server.HTTPPort`
2. **`StartSitesServer()`**: 同上
3. **`ReloadSites()`**: 同上
4. **`ReloadConfig()`**: 更新 `BackupDir` 引用为 `Directories.Backup`

### 第 4 步：Go 后端 — vhost.go 适配

1. **共享端口**: 改为从站点列表推导，不再依赖全局配置

### 第 5 步：Go 后端 — ftp.go (admin) 适配

1. 所有 `h.ConfigManager.FTP.Root` 替换为 `h.ConfigManager.Server.Directories.FTP`

### 第 6 步：Go 后端 — handlers/ftp.go 适配

1. FTP 服务器初始化时从 `Directories.FTP` 获取根目录

### 第 7 步：前端 — settings-section.html 重构

1. **删除** "HTTP 服务" 标签页
2. **新增** "目录设置" 标签页（sites、ftp、backup 三个目录输入框）
3. **修改** "FTP 服务" 标签页（移除 FTP 根目录字段）
4. **修改** "备份设置" 标签页（增加备份内容勾选、定时策略、保留份数）

### 第 8 步：前端 — settings.js 适配

1. 删除 HTTP port/root 相关代码
2. 新增 directories 渲染和收集逻辑
3. 更新 backup 渲染和收集逻辑
4. 更新 FTP 渲染（移除 root 字段）

### 第 9 步：前端 — services.js / app.js 适配

1. 移除对 `http.port`、`http.root` 的引用
2. 更新目录显示来源为 `directories`

---

## 涉及文件清单

| 文件 | 改动类型 |
|------|----------|
| `src/config/config.go` | 结构体重构 + 迁移逻辑 |
| `src/admin/api.go` | API 适配 |
| `src/handlers/server.go` | 共享端口逻辑 |
| `src/handlers/vhost.go` | 共享端口推导 |
| `src/admin/ftp.go` | FTP 根目录引用 |
| `src/handlers/ftp.go` | FTP 服务器初始化 |
| `src/admin/sites.go` | 默认站点根目录 |
| `src/static/admin/components/settings-section.html` | UI 重构 |
| `src/static/admin/js/tabs/settings.js` | 前端逻辑 |
| `src/static/admin/js/tabs/services.js` | 服务状态显示 |
| `src/static/admin/js/app.js` | 状态数据适配 |

## 向后兼容

- `loadServer()` 检测旧 JSON 字段并自动迁移
- 首次加载时自动写入新格式，旧字段不再保留
