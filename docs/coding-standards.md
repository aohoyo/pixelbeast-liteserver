# 代码规范

## Go 代码规范

### 文件命名

- **文件名**: 小写，下划线分隔
  - `file_manager.go` ✅
  - `FileManager.go` ❌

- **测试文件**: 添加 `_test.go` 后缀
  - `file_manager_test.go`

### 包组织

```go
// 包注释
// Package panel 提供管理面板 HTTP API
package panel

import (
    // 标准库
    "fmt"
    "net/http"
    
    // 项目内部包
    "pixelbeast/src/config"
    
    // 第三方库（如有）
)
```

### 命名规范

```go
// 公开函数：大驼峰
func GetServerManager() *ServerManager {}

// 私有函数：小驼峰
func validateConfig(cfg *config.Config) error {}

// 常量：大驼峰或全大写
const MaxRetries = 3
const DEFAULT_PORT = 8080

// 接口：动词或名词 + er
type Reader interface {}
type Writer interface {}

// 结构体：大驼峰
type ServerManager struct {
    httpServer *http.Server  // 私有字段：小驼峰
    Port       int           // 公开字段：大驼峰
}
```

### 错误处理

```go
// 总是检查错误
data, err := loadData()
if err != nil {
    return fmt.Errorf("加载失败: %w", err)
}

// 错误信息要清晰
return fmt.Errorf("无法加载配置 %s: %w", path, err)

// 使用 errors.Is 和 errors.As
if errors.Is(err, os.ErrNotExist) {
    // 文件不存在
}
```

### 注释规范

```go
// ServerManager 管理所有服务器实例
// 线程安全，支持并发访问
type ServerManager struct {
    // httpServer 是 HTTP 服务器实例
    httpServer *http.Server
}

// Start 启动所有服务器
// 返回 error 表示启动失败
func (sm *ServerManager) Start() error {
    // 实现
}
```

### 并发安全

```go
type ServerManager struct {
    mu sync.RWMutex
    
    // 受 mu 保护
    adminRunning bool
}

func (sm *ServerManager) IsRunning() bool {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    return sm.adminRunning
}

func (sm *ServerManager) SetRunning(running bool) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    sm.adminRunning = running
}
```

---

## JavaScript 代码规范

### 文件命名

- **文件名**: 小写，连字符分隔
  - `my-tab.js` ✅
  - `MyTab.js` ❌

- **类文件**: 大驼峰
  - `BaseTab.js`
  - `DataTable.js`

### 变量声明

```javascript
// 使用 const/let，不用 var
const API_URL = '/api';
let currentPath = '/';

// 常量大写
const MAX_ITEMS = 100;

// 布尔值前缀 is/has
const isLoading = true;
const hasError = false;
```

### 函数声明

```javascript
// 函数名与参数左括号之间有空格
function initTab({ state, api }) {
    // ...
}

// 箭头函数
const getValue = (item) => item.value;
const sum = (a, b) => a + b;

// 异步函数
async function loadData() {
    const data = await api.getJSON('/api/data');
    return data;
}
```

### 模块导出

```javascript
// 命名导出
export function formatDate(date) { }
export const API_URL = '/api';

// 默认导出（单例）
export default new XxxTab({ api, state });
```

### 注释规范

```javascript
/**
 * 初始化标签页
 * @param {Object} deps - 依赖注入
 * @param {Object} deps.state - 状态管理
 * @param {Object} deps.api - API 实例
 */
export function initTab({ state, api }) {
    // 单行注释：解释代码意图
    const config = state.get('config');
}
```

### 异步处理

```javascript
// 使用 async/await
async function saveData() {
    try {
        await api.post('/api/data', data);
        toast.success('保存成功');
    } catch (error) {
        console.error('保存失败:', error);
        toast.error('保存失败: ' + error.message);
    }
}

// 并行请求
const [users, sites] = await Promise.all([
    api.getJSON('/api/users'),
    api.getJSON('/api/sites')
]);
```

---

## CSS 规范

### 文件组织

```
css/
├── base.css        # CSS 变量、重置
├── layout.css      # 布局样式
├── main.css        # 页面样式
└── components/     # 组件样式
    ├── button.css
    ├── card.css
    └── ...
```

### 命名规范

```css
/* BEM 风格 */
.card { }
.card__header { }
.card__body { }
.card--active { }

/* 状态类 */
.is-active { }
.is-hidden { }
.is-disabled { }

/* 工具类 */
.text-center { }
.mt-lg { }
```

### CSS 变量

```css
/* 定义在 :root */
:root {
    --primary: #f97316;
    --space-md: 16px;
}

/* 使用 var() */
.card {
    background: var(--card-bg);
    padding: var(--space-md);
}
```

### 选择器

```css
/* 避免过深嵌套 */
/* ✅ */
.card .card-header .title { }

/* ❌ */
.container .wrapper .content .card .card-header .title { }

/* 使用类选择器，避免标签选择器 */
/* ✅ */
.card-header { }

/* ❌ */
div > div > div { }
```

---

## Git 提交规范

### Commit Message 格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type 类型

| Type | 说明 |
|------|------|
| feat | 新功能 |
| fix | 修复 bug |
| docs | 文档更新 |
| style | 代码格式 |
| refactor | 重构 |
| test | 测试相关 |
| chore | 构建/工具 |

### 示例

```
feat(ftp): 添加用户配额限制

- 支持设置用户容量上限
- 显示已用空间和剩余空间
- 超出限制时拒绝上传

Closes #123
```

```
fix(admin): 修复密码验证失败问题

密码解密时未正确处理 base64 编码，
导致验证总是失败。

Fixes #456
```

---

## 代码审查清单

### Go 代码

- [ ] 代码格式符合 `gofmt`
- [ ] 错误处理完整
- [ ] 注释清晰准确
- [ ] 没有硬编码的配置值
- [ ] 资源正确释放（defer）
- [ ] 并发安全（如有 goroutine）
- [ ] 单元测试覆盖关键路径

### JavaScript 代码

- [ ] 使用 const/let，不用 var
- [ ] 异步操作使用 async/await
- [ ] 错误处理完整
- [ ] 避免内存泄漏（事件监听器）
- [ ] 依赖注入而非硬编码

### CSS 代码

- [ ] 使用 CSS 变量
- [ ] 没有硬编码颜色值
- [ ] 命名语义化
- [ ] 没有冗余样式