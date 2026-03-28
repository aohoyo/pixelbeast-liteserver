# 架构设计

## 整体架构

像素兽采用**两层架构**设计，清晰的职责分离：

```
┌─────────────────────────────────────────────────────────┐
│                    协议层 (handlers/)                    │
│                                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │ HTTP Server │  │ FTP Server  │  │ VHost Router│     │
│  │  (http.go)  │  │  (ftp.go)   │  │ (vhost.go)  │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
│                                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │ SSL Manager │  │  Logger     │  │ FileManager │     │
│  │  (ssl.go)   │  │ (logger.go) │  │(filemanager)│     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│                    业务层 (admin/)                       │
│                                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │ Sites API   │  │  FTP API    │  │ Files API   │     │
│  │ (sites.go)  │  │  (ftp.go)   │  │ (files.go)  │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
│                                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │ System API  │  │  Logs API   │  │ Share API   │     │
│  │ (system.go) │  │ (logs.go)   │  │ (share.go)  │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│                    基础层 (config/crypto)                │
│                                                         │
│  ┌───────────────────┐  ┌───────────────────┐          │
│  │  ConfigManager    │  │   AES-256-GCM     │          │
│  │   (config.go)     │  │   (crypto.go)     │          │
│  └───────────────────┘  └───────────────────┘          │
└─────────────────────────────────────────────────────────┘
```

## 模块说明

### 协议层 (handlers/)

负责网络协议处理，是系统的入口。

| 文件 | 职责 |
|------|------|
| `server.go` | 服务管理器，统一管理所有服务 |
| `http.go` | HTTP 服务器，静态文件服务 |
| `ftp.go` | FTP 服务器，文件传输 |
| `vhost.go` | 虚拟主机路由，多站点支持 |
| `ssl.go` | SSL 证书管理 |
| `logger.go` | 日志系统 |
| `filemanager.go` | 文件管理器 |

### 业务层 (admin/)

负责业务逻辑处理，提供 API 接口。

| 文件 | 职责 |
|------|------|
| `handler.go` | 路由入口，中间件 |
| `api.go` | 通用 API |
| `sites.go` | 站点管理 API |
| `ftp.go` | FTP 管理 API |
| `files.go` | 文件管理 API |
| `logs.go` | 日志查看 API |
| `share.go` | 分享功能 API |
| `compress.go` | 压缩功能 |

### 基础层 (config/crypto/)

提供底层支持，无业务逻辑。

| 模块 | 职责 |
|------|------|
| `config/` | 配置管理，读写 JSON 文件 |
| `crypto/` | 加密解密，AES-256-GCM |

## 数据流

### 请求处理流程

```
HTTP 请求
    ↓
ServerManager (handlers/server.go)
    ↓
VirtualHostRouter (handlers/vhost.go)
    ↓
匹配站点配置
    ↓
┌─────────────┬─────────────┐
│ 静态站点    │ 反向代理    │
│ (http.go)   │ (proxy.go)  │
└─────────────┴─────────────┘
    ↓
返回响应
```

### 管理面板请求流程

```
HTTP 请求 :9527
    ↓
AdminHandler (admin/handler.go)
    ↓
路由匹配
    ↓
┌─────────┬─────────┬─────────┐
│ /api/*  │ /admin/*│ 其他    │
│ API处理 │ 静态资源│ 404     │
└─────────┴─────────┴─────────┘
    ↓
返回响应
```

### 配置保存流程

```
前端提交
    ↓
API Handler (admin/*.go)
    ↓
ConfigManager (config/config.go)
    ↓
加密敏感字段 (crypto/crypto.go)
    ↓
写入 JSON 文件
```

## 配置系统

### 配置文件结构

```
config/
├── server.json      # 服务配置
├── sites.json       # 站点配置
├── ftp.json         # FTP 配置
└── secrets.key      # 加密密钥（自动生成）
```

### ConfigManager 核心方法

```go
// 创建
cm, err := config.NewConfigManager("./config")

// 访问
cm.Server.HTTPPort
cm.Sites.Sites[0]
cm.FTP.Users

// 保存
cm.Save()

// 密码管理
cm.SetAdminPassword("password")
cm.ValidateAdmin("admin", "password")
cm.SetFTPUserPassword("user", "password")
cm.ValidateFTPUser("user", "password")

// 站点管理
cm.AddSite(site)
cm.UpdateSite(id, site)
cm.DeleteSite(id)

// FTP 用户管理
cm.AddFTPUser(user, "password")
cm.UpdateFTPUser("user", user)
cm.DeleteFTPUser("user")
```

## 前端架构

### 模块化设计

```
前端采用 ES Modules + 组件化设计：

app.js (入口)
    ↓
core/ (核心模块)
    ├── api.js      - API 封装
    ├── state.js    - 状态管理
    ├── events.js   - 事件系统
    └── utils.js    - 工具函数
    ↓
components/ (UI 组件)
    ├── toast.js    - 消息提示
    ├── dialog.js   - 对话框
    └── data-table.js - 数据表格
    ↓
tabs/ (标签页)
    ├── BaseTab.js  - 基类
    ├── home.js     - 首页
    └── ...         - 其他标签页
```

### BaseTab 模式

```javascript
class XxxTab extends BaseTab {
    constructor(deps) {
        super(deps, 'xxx');
    }

    onInit() { }      // 初始化
    onLoad() { }      // 加载数据
    onRefresh() { }   // 刷新数据
    onDestroy() { }   // 销毁
}
```

### 数据流

```
用户操作
    ↓
Tab 模块 (tabs/*.js)
    ↓
API 调用 (core/api.js)
    ↓
后端 API (admin/*.go)
    ↓
ConfigManager (config/config.go)
    ↓
响应返回
    ↓
更新 State (core/state.js)
    ↓
UI 更新
```

## 安全设计

### 敏感信息加密

- 使用 AES-256-GCM 加密
- 加密字段：`admin_password`、`ftp.users[].password`
- 密钥文件：`secrets.key`（权限 600）

### 安全入口

- 默认 `/admin`，可自定义
- 隐藏后台存在
- 防止暴力破解

### 权限控制

- 配置文件权限 600
- 密钥文件不在 web 目录
- API 需要登录验证