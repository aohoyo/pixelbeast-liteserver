# 代码规范

## Go 代码规范

### 文件命名

- **文件名**: 小写，下划线分隔 (`file_manager.go`)
- **测试文件**: 添加 `_test.go` 后缀
- **包名**: 小写单词，简洁明了

### 包组织

```go
// 包注释，描述包的功能
package handlers

// 导入分组
import (
    // 标准库
    "fmt"
    "log"
    "net/http"

    // 项目内部包
    "pixelbeast/src/config"
    "pixelbeast/src/admin"

    // 第三方库（如有）
)
```

### 函数命名

```go
// 公开函数：大驼峰
func GetServerManager() *ServerManager {
    // ...
}

// 私有函数：小驼峰
func validateConfig(cfg *config.Config) error {
    // ...
}
```

### 错误处理

```go
// 总是检查错误
data, err := loadData()
if err != nil {
    log.Printf("加载失败: %v", err)
    return err
}

// 错误信息要清晰
return fmt.Errorf("无法加载配置 %s: %w", path, err)
```

### 注释规范

```go
// ServerManager 管理所有服务器实例
type ServerManager struct {
    // httpServer 是 HTTP 服务器实例
    httpServer *http.Server
}

// Start 启动所有服务器
// 返回 error 表示启动失败
func (sm *ServerManager) Start() error {
    // 实现代码
}
```

## JavaScript 代码规范

### 文件命名

- **文件名**: 小写，连字符分隔 (`my-tab.js`)
- **类文件**: 大驼峰 (`EventManager.js`)

### 代码格式

```javascript
// 使用 const/let，不用 var
const API_URL = '/api';
let currentPath = '/';

// 函数声明：函数名与参数左括号之间有空格
function initTab({ state, api }) {
    // ...
}

// 箭头函数：单参数省略括号，多参数使用括号
data => data.value
(a, b) => a + b
```

### 注释规范

```javascript
/**
 * 初始化标签页
 * @param {Object} dependencies - 依赖注入 { state, api, toast }
 */
export function initTab({ state, api, toast }) {
    // 单行注释：解释代码意图
    const data = loadData();
}
```

### 导入顺序

```javascript
// 1. 核心模块
import StateManager from '../core/state.js';

// 2. UI 组件
import toast from '../ui/toast.js';

// 3. 功能模块
import { loadData } from '../api/data.js';
```

## HTML/CSS 规范

### HTML 结构

```html
<!-- 使用语义化标签 -->
<section class="tab-content" id="home">
    <div class="card">
        <div class="card-header">
            <h3>标题</h3>
        </div>
        <div class="card-body">
            <!-- 内容 -->
        </div>
    </div>
</section>
```

### CSS 命名

```css
/* BEM 风格命名 */
.card { }
.card__header { }
.card__body { }
.card--active { }

/* 状态类 */
.is-active { }
.is-hidden { }
.is-disabled { }
```

## 项目规范

### 目录结构

```
src/
├── handlers/       # 协议层
├── admin/          # 管理面板
├── services/       # 业务逻辑
├── static/         # 静态资源
└── config/         # 配置管理
```

### 依赖原则

1. **最小依赖**: 优先使用标准库
2. **单向依赖**: 上层可以依赖下层，下层不依赖上层
3. **接口隔离**: 定义清晰的接口

### 代码审查要点

- [ ] 代码格式符合规范
- [ ] 错误处理完整
- [ ] 注释清晰准确
- [ ] 没有硬编码的配置值
- [ ] 资源正确释放（defer）
- [ ] 并发安全（如有 goroutine）

## 提交规范

### Commit Message 格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type 类型

- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式（不影响功能）
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建/工具相关

### 示例

```
feat(admin): 添加 FTP 用户管理功能

- 实现用户列表展示
- 支持添加/删除用户
- 添加用户权限验证

Closes #123
```

## 文档更新规范

### 何时更新文档

- **必须更新**: 新增 API、修改架构、变更配置格式
- **建议更新**: 新增功能、修复重要 bug、优化性能
- **可选更新**: 代码重构、小改动

### 更新哪些文档

| 变更类型 | 需要更新的文档 |
|---------|---------------|
| 新增 API | `docs/api.md` |
| 修改架构 | `docs/architecture.md` |
| 前端改动 | `docs/frontend.md` |
| 配置变更 | `docs/overview.md` |
| 代码规范 | `docs/coding-standards.md` |

### 更新流程

1. 修改代码
2. 同步更新相关文档
3. 在 Commit 中说明文档变更
