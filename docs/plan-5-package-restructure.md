# 计划五：包结构重组 — 去层级 + 站点/面板拆分 + 文件合并

## 背景与问题

### 当前结构

```
pixelbeast/
├── main.go                      ← 入口在根目录
├── embed.go
├── src/
│   ├── admin/                   ← 管理面板（22 文件，6000+ 行，偏扁平）
│   ├── config/
│   ├── crypto/
│   ├── core/
│   │   ├── server/              ← God Object：同时管面板 + 站点 + FTP + SSL
│   │   ├── logger/
│   │   ├── monitor/
│   │   ├── file/
│   │   ├── backup/
│   │   ├── ssl/
│   │   └── ftp/
│   └── static/
```

### 核心问题

1. **core/ 层级多余** — 多一层无意义嵌套，import 路径变长
2. **ServerManager 是 God Object**（`core/server/server.go` 781 行）— 同时管理面板启动、站点启停、FTP 控制、SSL 生命周期，职责不清
3. **站点和面板耦合** — 面板 API 通过 ServerManager 直接控制站点服务，没有接口隔离
4. **admin 命名模糊** — 不如 panel 直观
5. **panel 文件偏碎** — 22 个文件中有多个同域小文件可合并

## 目标结构

```
pixelbeast/
├── src/
│   ├── cmd/                     ← 入口
│   │   ├── main.go              ← 只做组装和启动
│   │   └── embed.go             ← 静态资源嵌入
│   │
│   ├── site/                    ← 站点服务（原 core/server）
│   │   ├── manager.go           ← SiteManager（只管站点启停、虚拟主机）
│   │   ├── vhost.go             ← 虚拟主机路由
│   │   ├── proxy.go             ← 反向代理
│   │   └── server.go            ← 静态文件/目录列表
│   │
│   ├── panel/                   ← 管理面板（原 admin/，22 文件 → 14 文件）
│   │   ├── handler.go           ← 会话、认证、静态资源服务
│   │   ├── router.go            ← 路由表
│   │   ├── middleware.go        ← 中间件
│   │   ├── response.go          ← API 响应工具
│   │   ├── static.go            ← 静态文件系统
│   │   ├── api_system.go        ← 系统监控、清理、自启（合并 service.go + uptime/priv）
│   │   ├── api_site.go          ← 站点管理
│   │   ├── api_ssl.go           ← 证书申请/续签/导入/DNS（三合一）
│   │   ├── api_ftp.go           ← FTP 服务 + 用户 + 文件管理
│   │   ├── api_file.go          ← 文件 + 压缩 + 分享（三合一）
│   │   ├── api_config.go        ← 配置管理
│   │   ├── api_backup.go        ← 备份管理
│   │   ├── api_log.go           ← 日志管理
│   │   └── api_service.go       ← 自启动服务管理
│   │
│   ├── ssl/                     ← SSL 证书核心（原 core/ssl）
│   │   ├── manager.go
│   │   ├── lego.go
│   │   └── provider.go
│   │
│   ├── ftp/                     ← FTP 服务器（原 core/ftp）
│   │   └── server.go
│   │
│   ├── config/                  ← 配置管理（不变）
│   │   └── config.go
│   │
│   ├── logger/                  ← 日志系统（原 core/logger）
│   │   └── logger.go
│   │
│   ├── monitor/                 ← 系统监控（原 core/monitor）
│   │   ├── system.go
│   │   ├── utils.go
│   │   └── mem_release_*.go
│   │
│   ├── file/                    ← 文件操作（原 core/file）
│   │   ├── operations.go
│   │   └── compress.go
│   │
│   ├── backup/                  ← 备份管理（原 core/backup）
│   │   └── manager.go
│   │
│   ├── crypto/                  ← 加密工具（不变）
│   │   └── crypto.go
│   │
│   └── static/                  ← 前端静态资源（不变）
│       └── admin/
```

## Panel 文件合并策略

22 个文件 → 14 个，靠统一 `api_` 前缀 + 同域合并：

| 原文件 | 行数 | → 目标文件 | 合并说明 |
|--------|------|-----------|---------|
| handler.go | 491 | handler.go | 提纯：只保留会话、认证、静态资源 |
| router.go | 172 | router.go | 不变 |
| middleware.go | 144 | middleware.go | 不变 |
| response.go | 134 | response.go | 不变 |
| static.go | 7 | static.go | 不变 |
| sites.go | 520 | api_site.go | 改名 |
| ftp.go | 844 | api_ftp.go | 改名 |
| logs.go | 602 | api_log.go | 改名 |
| api_config.go | 197 | api_config.go | 不变 |
| api_backup.go | 144 | api_backup.go | 不变 |
| api_system.go | 1925 | api_system.go | 吸收 uptime/priv 相关函数 |
| service.go | 355 | api_service.go | 改名 |
| ssl.go + ssl_cert.go + ssl_dns.go | 1100 | api_ssl.go | 三合一 |
| files.go + compress.go + share.go | 1580 | api_file.go | 三合一 |
| priv_*.go + uptime_*.go | 125 | → api_system.go 内 | 平台函数内联到 api_system.go |

## 关键拆分：ServerManager → SiteManager + main 组装

### 之前

```go
// core/server/server.go — 781 行 God Object
type ServerManager struct {
    ConfigManager *config.ConfigManager
    AdminServer   *http.Server    // 面板
    AdminHandler  http.Handler    // 面板
    SitesServer   *http.Server    // 站点
    SSLManager    *ssl.SSLManager
    FTPServer     *ftp.FTPServer
    FileManager   *file.FileManager
}
```

### 之后

```go
// src/cmd/main.go — 只做组装
func main() {
    cm := config.NewConfigManager(...)

    // 日志
    logger.InitLoggerWithConfig(...)

    // SSL（独立）
    sslMgr := ssl.NewSSLManager(cm, ...)
    sslMgr.Start()

    // 站点服务（独立，不关心面板）
    siteMgr := site.NewManager(cm, sslMgr, fileMgr)
    siteMgr.Start()

    // FTP（独立）
    ftpSrv := ftp.NewFTPServer(...)
    ftpSrv.Start()

    // 面板（引用其他服务，不反向依赖）
    panelHandler := panel.New(cm, siteMgr, sslMgr, ftpSrv)
    panelSrv := &http.Server{Handler: panelHandler}
    panelSrv.ListenAndServe()
}
```

## 依赖方向

```
src/cmd → 所有包（只做组装）

panel → config, logger, file, monitor, backup, site, ssl, ftp
site  → config, logger, file, ssl
ssl   → config, logger
ftp   → config, logger
file  → config
logger → config
config → crypto
crypto → 无依赖
```

## 实施步骤

| 步骤 | 内容 | 风险 |
|------|------|------|
| 1 | `main.go` + `embed.go` → `src/cmd/` | 低 |
| 2 | `core/server/` → `site/`（改名 + 包名） | 中 |
| 3 | `core/*` → `src/` 平级（去掉 core/ 层级，全局替换 import） | 低（机械） |
| 4 | `admin/` → `panel/`（全局替换 import） | 低（机械） |
| 5 | ServerManager 拆为 SiteManager + 面板独立启动 | 高 |
| 6 | Panel 文件合并（ssl 三合一、file 三合一、平台函数内联） | 中 |

每步完成后 `go build ./...` && `go vet ./...` 验证。

## 与已完成工作的关系

- 计划三已完成：handlers/ 转发层消除、core/ 包体系、中间件链、路由表
- 本次在计划三基础上继续：去 core/ 层级、God Object 拆分、panel 文件整理
- 不影响前端代码（static/ 不变）
