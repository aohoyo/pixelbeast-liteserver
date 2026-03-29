# 像素兽 (PixelBeast) - AI 开发指南

> 本文档为 AI 助手（Claude/GPT/Codex 等）提供项目上下文，帮助快速理解和修改代码。

## 项目概述

**像素兽** 是一个轻量级、高性能的多站点服务器，使用 Go 语言开发。

### 核心特性

| 特性 | 说明 |
|------|------|
| 多站点管理 | 静态站点 + 反向代理，独立端口 |
| FTP 服务 | 多用户、容量限制、有效期控制 |
| SSL 证书 | 自动 HTTPS、Let's Encrypt 支持 |
| Web 管理 | 现代化暗色主题 UI |
| 单文件部署 | 静态资源嵌入，无需外部依赖 |
| 配置加密 | AES-256-GCM 加密敏感信息 |

### 技术栈

```
后端: Go 1.21+ (标准库优先)
前端: 原生 HTML/CSS/JS (ES Modules)
加密: AES-256-GCM
主题: 橙色暗色 (CSS 变量系统)
```

---

## 快速开始

```bash
# 进入项目目录
cd /home/wwlhlf/project/pixelbeast-liteserver

# 编译
go build -o pixelbeast

# 运行
./pixelbeast -config ./config

# 访问管理面板
# http://nas.banayou.com:9527/admin
# 默认账号: admin / admin123
```

---

## 目录结构

```
pixelbeast-liteserver/
├── main.go                 # 程序入口
├── embed.go                # 静态资源嵌入
├── go.mod / go.sum         # Go 模块
│
├── config/                 # 配置目录 (加密存储)
│   ├── server.json         # 服务配置 (端口、管理员)
│   ├── sites.json          # 站点配置
│   ├── ftp.json            # FTP 配置
│   └── secrets.key         # 加密密钥 (自动生成)
│
├── src/
│   ├── admin/              # 管理面板后端
│   │   ├── handler.go      # 路由入口
│   │   ├── api.go          # API 处理
│   │   ├── sites.go        # 站点管理
│   │   ├── ftp.go          # FTP 管理
│   │   ├── files.go        # 文件管理
│   │   └── static/admin/   # 前端资源
│   │
│   ├── handlers/           # HTTP/FTP 协议层
│   │   ├── server.go       # 服务管理器
│   │   ├── http.go         # HTTP 处理
│   │   ├── ftp.go          # FTP 服务
│   │   ├── vhost.go        # 虚拟主机路由
│   │   └── ssl.go          # SSL 管理
│   │
│   ├── config/             # 配置管理
│   │   └── config.go       # ConfigManager
│   │
│   └── crypto/             # 加密模块
│       └── crypto.go       # AES-256-GCM
│
├── docs/                   # 文档目录
│   ├── api.md              # API 文档
│   ├── architecture.md     # 架构设计
│   ├── frontend.md         # 前端开发指南
│   └── coding-standards.md # 代码规范
│
└── log/                   # 日志目录
```

---

## 架构设计

### 两层架构

```
┌─────────────────────────────────────────────┐
│              协议层 (handlers/)              │
│  HTTP Handler │ FTP Server │ VHost Router   │
├─────────────────────────────────────────────┤
│              业务层 (admin/)                │
│  站点管理 │ FTP 管理 │ 文件管理 │ 系统监控   │
├─────────────────────────────────────────────┤
│              基础层 (config/crypto)          │
│  ConfigManager │ AES-256-GCM │ Logger       │
└─────────────────────────────────────────────┘
```

### 依赖方向

```
main.go
    ↓
handlers/ (协议层)
    ↓
admin/ (业务层)
    ↓
config/ + crypto/ (基础层)
```

**规则**: 上层可依赖下层，下层不可依赖上层。

---

## 配置系统

### 配置文件

| 文件 | 内容 | 加密字段 |
|------|------|----------|
| `server.json` | 服务配置 | `admin_password` |
| `sites.json` | 站点列表 | 无 |
| `ftp.json` | FTP 配置 | `users[].password` |

### ConfigManager

```go
// 创建配置管理器
cm, err := config.NewConfigManager("./config")

// 访问配置
cm.Server.HTTPPort
cm.Sites.Sites
cm.FTP.Users

// 保存配置
cm.Save()

// 密码管理
cm.SetAdminPassword("newpass")
cm.ValidateAdmin("admin", "password")

// 站点管理
cm.AddSite(site)
cm.UpdateSite(id, site)
cm.DeleteSite(id)

// FTP 用户管理
cm.AddFTPUser(user, "password")
cm.ValidateFTPUser("username", "password")
```

---

## 前端架构

### 目录结构

```
src/static/admin/
├── index.html              # 入口页面
├── login.html              # 登录页
│
├── css/
│   ├── base.css            # CSS 变量、重置
│   ├── layout.css          # 布局样式
│   ├── main.css            # 页面样式
│   └── components/         # 组件样式
│
├── js/
│   ├── app.js              # 应用入口
│   ├── login.js            # 登录逻辑
│   │
│   ├── core/               # 核心模块
│   │   ├── api.js          # API 封装
│   │   ├── state.js        # 状态管理
│   │   ├── events.js       # 事件系统
│   │   ├── utils.js        # 工具函数
│   │   └── loader.js       # 组件加载
│   │
│   ├── components/         # UI 组件
│   │   ├── toast.js        # 消息提示
│   │   ├── dialog.js       # 对话框
│   │   ├── data-table.js   # 数据表格
│   │   └── ...
│   │
│   └── tabs/               # 标签页模块
│       ├── BaseTab.js      # 基类 ⭐
│       ├── home.js         # 首页
│       ├── sites.js        # 站点管理
│       ├── ftp.js          # FTP 管理
│       └── ...
│
└── components/             # HTML 模板
    ├── home-section.html
    ├── sites-section.html
    └── ...
```

### BaseTab 基类

所有标签页必须继承 `BaseTab`：

```javascript
import { BaseTab } from './BaseTab.js';

class XxxTab extends BaseTab {
    constructor(deps) {
        super(deps, 'xxx');  // 传入依赖和名称
    }

    // 初始化（只执行一次）
    onInit() {
        this.bindEvents();
    }

    // 加载数据（每次激活执行）
    async onLoad() {
        const data = await this.api.getJSON('/api/xxx');
        this.render(data);
    }

    // 刷新数据
    async onRefresh() {
        await this.onLoad();
    }
}

// 导出单例
export default new XxxTab({
    api,
    state,
    toast,
    dialog,
    events
});
```

### 状态管理

```javascript
// 获取配置
const config = state.get('config');

// 获取状态
const status = state.get('status');

// 设置状态
state.set('xxx', data);

// 监听变化
events.on('config:loaded', (config) => { });
```

### API 调用

```javascript
// GET
const data = await api.getJSON('/api/xxx');

// POST
await api.post('/api/xxx', { key: 'value' });

// 带缓存
const data = await api.getJSON('/api/xxx', { cache: true });

// 清除缓存
api.clearCache('/api/xxx');
```

### 事件通信

```javascript
// 触发事件
events.emit('data:updated', newData);

// 监听事件
events.on('data:updated', (data) => { });

// 匹配模式
events.match('tab:switch:xxx', () => { });
```

---

## CSS 规范

### CSS 变量系统

```css
/* 颜色 */
--primary: #f97316;
--success: #22c55e;
--danger: #ef4444;
--warning: #fbbf24;

/* 背景 */
--bg: #0c0a09;
--bg-elevated: #1c1917;
--card-bg: #292524;

/* 文字 */
--text: #fafaf9;
--text-secondary: #d6d3d1;
--text-muted: #78716c;

/* 边框 */
--border: #44403c;

/* 间距 */
--space-xs: 4px;
--space-sm: 8px;
--space-md: 16px;
--space-lg: 24px;

/* 圆角 */
--radius-sm: 4px;
--radius: 8px;
--radius-lg: 12px;
```

### 使用规范

```css
/* ✅ 正确：使用 CSS 变量 */
.card {
    background: var(--card-bg);
    border: 1px solid var(--border);
    color: var(--text);
}

/* ❌ 错误：硬编码颜色 */
.card {
    background: #292524;
    border: 1px solid #44403c;
}
```

---

## 开发规范

### Go 代码

```go
// 公开函数：大驼峰
func GetServerManager() *ServerManager {}

// 私有函数：小驼峰
func validateConfig(cfg *config.Config) error {}

// 错误处理：始终检查
data, err := loadData()
if err != nil {
    return fmt.Errorf("加载失败: %w", err)
}
```

### JavaScript 代码

```javascript
// 使用 const/let，不用 var
const API_URL = '/api';
let currentPath = '/';

// 依赖注入模式
export function initTab({ state, api, toast }) { }

// 事件驱动通信
events.emit('data:updated', newData);
events.on('data:updated', (data) => { });
```

### Git 提交

```
<type>(<scope>): <subject>

feat(ftp): 添加用户配额限制
fix(admin): 修复密码验证问题
docs(api): 更新 API 文档
refactor(css): 精简重复样式
```

---

## 常用命令

```bash
# 编译
go build -o pixelbeast

# 运行
./pixelbeast -config ./config

# 测试
go test ./...

# 格式化
go fmt ./...

# 静态检查
go vet ./...
```

---

## 故障排查

| 问题 | 解决方案 |
|------|----------|
| 端口冲突 | 检查 `config/server.json` |
| 密码验证失败 | 检查 `secrets.key` 是否存在 |
| 配置加载失败 | 检查 `config/` 目录权限 |
| 前端资源 404 | 重新编译（`go build`） |

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