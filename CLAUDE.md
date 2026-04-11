# 像素兽 (PixelBeast) - AI 开发指南

> 本文档为 AI 助手（Claude/GPT/Codex 等）提供项目上下文，帮助快速理解和修改代码。
> 当前版本: **v3.1.10** | Go: **1.25+** | 更新日期: 2026-04-10

---

## 项目概述

**像素兽** 是一个轻量级、高性能的多站点服务器，使用 Go 语言开发，单文件部署，适合个人开发者和小型团队。

### 核心特性

| 特性 | 说明 |
|------|------|
| 多站点管理 | 静态站点 + 反向代理，独立端口，域名绑定 |
| FTP 服务 | 多用户、容量限制、限速、有效期控制 |
| SSL 证书 | Let's Encrypt 自动证书、自定义证书、自动 HTTPS |
| 文件管理 | Web 文件浏览器、上传下载、压缩解压、文件分享 |
| 系统监控 | CPU/内存/磁盘/网络实时监控，进程统计 |
| Web 管理 | 现代化暗色主题 UI，标签页组件化架构 |
| 单文件部署 | 静态资源嵌入，无需外部依赖 |
| 配置加密 | AES-256-GCM 加密敏感信息 |

### 技术栈

```
后端: Go 1.25+ (标准库优先，仅 gopsutil 外部依赖)
前端: 原生 HTML/CSS/JS (ES Modules, 无框架)
加密: AES-256-GCM
主题: 橙色暗色 (CSS 变量系统)
```

---

## 快速开始

```bash
# 编译
go build -o pixelbeast

# 运行
./pixelbeast -config ./config

# 查看版本
./pixelbeast -version

# 访问管理面板
# http://nas.banayou.com:9527/admin
# 默认账号: admin / admin123
```

---

## 目录结构

```
pixelbeast-liteserver/
├── main.go                     # 程序入口 (版本号定义、服务启动)
├── embed.go                    # 静态资源嵌入 (开发/生产模式切换)
├── go.mod / go.sum             # Go 模块
│
├── config/                     # 配置目录 (加密存储)
│   ├── server.json             # 服务配置 (管理员、日志、备份)
│   ├── sites.json              # 站点列表
│   ├── ftp.json                # FTP 配置
│   ├── shares.json             # 文件分享数据 (运行时)
│   └── secrets.key             # AES-256-GCM 密钥 (自动生成)
│
├── src/
│   ├── admin/                  # 管理面板后端 (业务层)
│   │   ├── handler.go          # 路由入口、认证、会话管理
│   │   ├── api.go              # 通用 API (系统状态、配置、备份、系统操作)
│   │   ├── sites.go            # 站点管理 API (CRUD、启停、批量操作)
│   │   ├── ftp.go              # FTP 管理 API (用户 CRUD、文件管理、服务控制)
│   │   ├── files.go            # 文件管理 API (浏览、上传下载、重命名、权限)
│   │   ├── logs.go             # 日志管理 API (查看、统计、清理、导出)
│   │   ├── share.go            # 文件分享 API (创建、下载、管理)
│   │   ├── compress.go         # 文件压缩/解压 API
│   │   ├── response.go         # HTTP 响应工具函数
│   │   └── static.go           # 静态资源嵌入声明
│   │
│   ├── handlers/               # 协议层 (HTTP/FTP/SSL)
│   │   ├── server.go           # ServerManager (统一管理所有服务)
│   │   ├── http.go             # HTTP 服务器
│   │   ├── ftp.go              # FTP 服务器
│   │   ├── vhost.go            # 虚拟主机路由 (多站点域名匹配)
│   │   ├── proxy.go            # 反向代理
│   │   ├── ssl.go              # SSL 证书管理器
│   │   ├── logger.go           # 日志系统 (三分类: Server/Auth/API)
│   │   ├── system.go           # 系统监控 (CPU/内存/磁盘/网络)
│   │   ├── filemanager.go      # 文件管理器 (书签、快捷目录)
│   │   ├── utils.go            # 工具函数
│   │   ├── mem_release_*.go    # 平台相关内存释放
│   │   └── mem_windows.go      # Windows 内存查询
│   │
│   ├── config/                 # 配置管理 (基础层)
│   │   └── config.go           # ConfigManager (CRUD、加密密码)
│   │
│   └── crypto/                 # 加密模块 (基础层)
│       └── crypto.go           # AES-256-GCM 加解密
│
├── src/static/admin/           # 前端资源 (嵌入到二进制)
│   ├── index.html              # 主页面入口
│   ├── views/
│   │   └── login.html          # 登录页
│   ├── css/
│   │   ├── base.css            # CSS 变量、重置
│   │   ├── layout.css          # 布局样式
│   │   ├── main.css            # 主页面样式
│   │   ├── login.css           # 登录页样式
│   │   └── components/         # 组件样式 (20+ 独立文件)
│   │       ├── button.css, card.css, data-table.css
│   │       ├── dialog.css, modal.css, tooltip.css
│   │       ├── file-manager.css, file-browser.css
│   │       ├── ftp.css, sites.css, service-tab.css
│   │       ├── settings.css, form.css, logs.css
│   │       └── badge.css, status.css, skeleton.css, ...
│   ├── js/
│   │   ├── app.js              # 应用入口
│   │   ├── login.js            # 登录逻辑
│   │   ├── core/               # 核心模块
│   │   │   ├── api.js          # API 封装 (缓存、自动重试)
│   │   │   ├── state.js        # 状态管理
│   │   │   ├── events.js       # 事件系统
│   │   │   ├── utils.js        # 工具函数
│   │   │   ├── cache.js        # 缓存管理
│   │   │   └── loader.js       # 组件动态加载
│   │   ├── components/         # UI 组件
│   │   │   ├── toast.js        # 消息提示
│   │   │   ├── dialog.js       # 对话框
│   │   │   ├── message.js      # 消息封装
│   │   │   ├── data-table.js   # 数据表格
│   │   │   ├── skeleton.js     # 骨架屏
│   │   │   ├── tooltip.js      # 提示框
│   │   │   ├── file-icons.js   # 文件图标映射
│   │   │   ├── file-manager.js # 文件管理器
│   │   │   ├── upload-manager.js # 上传管理器
│   │   │   └── context-menu.js # 右键菜单
│   │   └── tabs/               # 标签页模块
│   │       ├── BaseTab.js      # 基类 (生命周期管理)
│   │       ├── home.js         # 首页 (系统监控仪表盘)
│   │       ├── sites.js        # 站点管理
│   │       ├── ftp.js          # FTP 管理
│   │       ├── files.js        # 文件管理
│   │       ├── cert.js         # 证书管理
│   │       ├── logs.js         # 日志查看
│   │       ├── settings.js     # 系统设置
│   │       ├── services.js     # 服务管理
│   │       └── settings-validator.js # 设置表单验证
│   └── components/             # HTML 模板片段
│
├── docs/                       # 文档目录
│   ├── overview.md             # 项目概述
│   ├── architecture.md         # 架构设计
│   ├── api.md                  # API 文档
│   ├── frontend.md             # 前端开发指南
│   ├── coding-standards.md     # 代码规范
│   ├── deployment.md           # 部署指南
│   └── CHANGELOG.md            # 更新日志
│
├── log/                        # 运行时日志目录
│   ├── server.log              # 服务日志
│   ├── auth.log                # 认证日志
│   ├── api.log                 # API 日志
│   ├── http/                   # HTTP 访问/错误日志
│   └── ftp/                    # FTP 访问/错误日志
│
├── ssl/                        # SSL 证书目录 (运行时)
├── sites/                      # 站点文件根目录 (运行时)
├── ftp/                        # FTP 文件根目录 (运行时)
└── backups/                    # 备份目录 (运行时)
```

---

## 架构设计

### 三层架构

```
┌──────────────────────────────────────────────────────┐
│                  协议层 (handlers/)                    │
│  HTTP Server │ FTP Server │ VHost Router │ Proxy     │
│  SSL Manager │ Logger │ System Monitor │ FileManager │
├──────────────────────────────────────────────────────┤
│                  业务层 (admin/)                       │
│  Sites │ FTP │ Files │ Logs │ Share │ Compress │ API │
├──────────────────────────────────────────────────────┤
│                  基础层 (config/crypto)                │
│  ConfigManager │ AES-256-GCM │ Secrets Key           │
└──────────────────────────────────────────────────────┘
```

### 依赖方向

```
main.go → handlers/ → admin/ → config/ + crypto/
```

**规则**: 上层可依赖下层，下层不可依赖上层。

---

## 配置系统

### 配置文件

| 文件 | 内容 | 加密字段 |
|------|------|----------|
| `server.json` | 服务、管理面板、日志、备份配置 | `admin.password` |
| `sites.json` | 站点列表 | 无 |
| `ftp.json` | FTP 服务配置 | `users[].password` |
| `secrets.key` | AES-256-GCM 加密密钥 | - |

### 配置结构 (Go 类型)

```go
type ServerConfig struct {
    Name        string            // 服务器名称
    Timezone    string            // 时区 (默认 Asia/Shanghai)
    Admin       AdminConfig       // 管理面板配置
    Directories DirectoriesConfig // 目录配置
    Backup      BackupConfig      // 备份配置
    Log         LogConfig         // 日志配置
}

type AdminConfig struct {
    Port       int    `json:"port"`        // 管理面板端口 (9527)
    Username   string `json:"username"`    // 管理员用户名
    Password   string `json:"password"`    // AES 加密密码
    Path       string `json:"path"`        // 安全入口 (/admin)
    Domain     string `json:"domain"`      // 绑定域名 (空=全部)
    SSLEnabled bool   `json:"ssl_enabled"` // HTTPS
}

type DirectoriesConfig struct {
    Sites  string `json:"sites"`  // ./sites
    FTP    string `json:"ftp"`    // ./ftp
    Backup string `json:"backup"` // ./backups
}

type SiteConfig struct {
    ID, Name    string
    Enabled     bool
    Type        string       // "static" | "proxy"
    Port        int
    Domain      []string
    Root        string       // 自定义根目录
    IndexFiles  []string
    AutoIndex   bool
    Proxy       *ProxyConfig // 反向代理配置
    SSL         *SSLConfig   // SSL 配置
}

type FTPUser struct {
    Username, Password, RootPath, Status string
    Quota, UsedSpace                     int64  // 容量 (MB)
    SpeedLimit, Bandwidth                int64  // 下载/上传限速 KB/s
    MaxConnections, MaxFiles             int
    MaxFileSize                          int64  // 单文件限制 MB
    ExpiryDays                           int
    ExpiryDate, Remark                   string
}
```

### ConfigManager 常用方法

```go
cm, _ := config.NewConfigManager("./config")

// 配置访问
cm.Server.Admin.Port
cm.Sites.Sites
cm.FTP.Users

// 目录路径 (带默认值)
cm.GetSitesDir()      // → "./sites"
cm.GetFTPRoot()       // → "./ftp"
cm.GetBackupDir()     // → "./backups"
cm.GetSiteRoot(site)  // → site.Root || SitesDir/siteID
cm.GetSharedPort()    // 从第一个启用站点推导

// 保存
cm.Save()             // 保存所有
cm.ResetToDefaults()  // 重置服务配置 (保留站点和FTP数据)

// 密码
cm.SetAdminPassword("pass")
cm.ValidateAdmin("admin", "pass")
cm.SetFTPUserPassword("user", "pass")
cm.ValidateFTPUser("user", "pass")
cm.EncryptPassword("pass")

// 站点 CRUD
cm.AddSite(site)
cm.UpdateSite(id, site)
cm.DeleteSite(id)
cm.GetSiteByID(id)

// FTP 用户 CRUD
cm.AddFTPUser(user, "password")
cm.UpdateFTPUser(username, user)
cm.DeleteFTPUser(username)
cm.GetFTPUser(username)
cm.GetFTPUserConfig(username) // 带锁，返回副本
```

---

## 前端架构

### BaseTab 基类

所有标签页继承 `BaseTab`，使用依赖注入和单例模式：

```javascript
import { BaseTab } from './BaseTab.js';

class XxxTab extends BaseTab {
    constructor(deps) {
        super(deps, 'xxx');  // { api, state, toast, message, dialog, events }
    }

    onInit() { }           // 只执行一次
    onLoad() { }           // 首次激活
    onRefresh() { }        // 再次激活
    onDestroy() { }        // 销毁

    // 工具方法
    this.$('#el')           // querySelector
    this.$$('.els')         // querySelectorAll
    this.setText('#el', v)  // 设置文本
    this.showLoading(c)     // 显示加载
}

export default new XxxTab({ api, state, toast, message, dialog, events });
```

### API 调用

```javascript
const data = await api.getJSON('/api/xxx');
await api.post('/api/xxx', { key: 'value' });
const data = await api.getJSON('/api/xxx', { cache: true });
api.clearCache('/api/xxx');
```

### 事件通信

```javascript
events.emit('data:updated', data);
events.on('data:updated', (data) => { });
events.match('tab:switch:xxx', () => { });
```

---

## CSS 规范

```css
/* 使用 CSS 变量，禁止硬编码颜色 */
--primary: #f97316;    --success: #22c55e;
--danger: #ef4444;     --warning: #fbbf24;

--bg: #0c0a09;         --bg-elevated: #1c1917;
--card-bg: #292524;    --border: #44403c;

--text: #fafaf9;       --text-secondary: #d6d3d1;
--text-muted: #78716c;

--space-xs: 4px;       --space-sm: 8px;
--space-md: 16px;      --space-lg: 24px;
--radius-sm: 4px;      --radius: 8px;    --radius-lg: 12px;
```

---

## 开发规范

### Go 代码

```go
// 公开函数：大驼峰 | 私有函数：小驼峰
func GetServerManager() *ServerManager {}
func validateConfig(cfg *config.Config) error {}

// 错误处理：始终检查
data, err := loadData()
if err != nil {
    return fmt.Errorf("加载失败: %w", err)
}
```

### JavaScript 代码

```javascript
// const/let，不用 var
// 依赖注入模式
export function initTab({ state, api, toast }) { }
// 事件驱动通信
events.emit('data:updated', newData);
```

### Git 提交

```
<type>(<scope>): <subject>
feat(ftp): 添加用户配额限制
fix(admin): 修复密码验证问题
chore: 发布 v3.1.10，日志系统优化
```

---

## 常用命令

```bash
go build -o pixelbeast              # 编译
./pixelbeast -config ./config       # 运行
./pixelbeast -version               # 版本
go test ./...                       # 测试
go fmt ./...                        # 格式化
go vet ./...                        # 静态检查
```

---

## 故障排查

| 问题 | 解决方案 |
|------|----------|
| 端口冲突 | 检查 `config/server.json` 的 `admin.port` |
| 密码验证失败 | 检查 `secrets.key` 是否存在且匹配 |
| 配置加载失败 | 检查 `config/` 目录权限 (755) |
| 前端资源 404 | 重新编译 (`go build`) |
| 日志目录不存在 | 自动创建 `./log/` |

---

## 文档索引

| 文档 | 说明 |
|------|------|
| [docs/overview.md](docs/overview.md) | 项目概述 |
| [docs/architecture.md](docs/architecture.md) | 架构设计 |
| [docs/api.md](docs/api.md) | API 文档 |
| [docs/frontend.md](docs/frontend.md) | 前端开发指南 |
| [docs/coding-standards.md](docs/coding-standards.md) | 代码规范 |
| [docs/deployment.md](docs/deployment.md) | 部署指南 |
| [docs/CHANGELOG.md](docs/CHANGELOG.md) | 更新日志 |

---

## 联系方式

- **项目地址**: https://github.com/aohoyo/litefeather
- **开发者**: 王伟
- **访问域名**: nas.banayou.com
