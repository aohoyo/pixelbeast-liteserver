# 前端优化计划 - 计划四

> 日期：2026-04-13
> 状态：待确认
> 预计总工时：15-20h

---

## 一、现状分析

### 1.1 技术栈
- 原生 JS ES Modules，无框架
- 组件化：BaseTab 基类 + 独立 HTML 组件文件
- CSS：CSS 变量 + 组件级 CSS 文件
- 无构建工具，浏览器直接加载

### 1.2 前端文件统计
| 目录 | 文件数 | 说明 |
|------|--------|------|
| js/core/ | 5 | 基础框架（BaseTab、loader、utils） |
| js/tabs/ | 7 | 功能标签页 |
| js/components/ | 5 | 组件（file-manager、file-browser、file-icons） |
| css/ | 15+ | 样式文件 |
| components/ | 10+ | HTML 组件模板 |

### 1.3 当前痛点
1. **无构建工具** — 每个 JS/CSS 文件都是独立 HTTP 请求，加载慢
2. **无模板引擎** — HTML 通过 innerHTML 拼接，可读性差、维护难
3. **状态管理原始** — 依赖 DOM 查询和全局变量
4. **无错误边界** — JS 报错会导致整个标签页白屏
5. **CSS 无作用域** — 全局 CSS 容易冲突

---

## 二、优化方案

### 2.1 引入轻量构建工具（esbuild）

**不引入 Vue/React**，保持原生 JS，但用 esbuild 打包。

#### 为什么不用 Vue/React
- 项目定位轻量级，管理面板页面有限（6-7个标签页）
- 引入框架增加复杂度（构建配置、状态管理、SSR 等）
- 宝塔面板也是原生 JS，证明不需要框架也能做好
- 当前 BaseTab 组件化已经能满足需求

#### esbuild 方案
```json
// package.json
{
  "scripts": {
    "build": "esbuild js/app.js --bundle --outfile=dist/app.js --minify",
    "watch": "esbuild js/app.js --bundle --outfile=dist/app.js --watch"
  }
}
```

**收益**：
- 合并几十个 JS 文件为 1-2 个，减少 HTTP 请求
- Tree-shaking 删除未使用代码
- 压缩减小体积
- 支持 TypeScript（可选）

**工时**：2h

### 2.2 简易模板引擎

用 tagged template literals 实现轻量模板：

```js
// 当前方式（innerHTML 拼接）
listEl.innerHTML = items.map(f => 
  `<div class="fb-item" data-path="${escapeHtml(f.path)}">
    <span>${getFileIcon(f.name)}</span>
    <span>${escapeHtml(f.name)}</span>
  </div>`
).join('');

// 优化后（模板标签）
const itemTemplate = (f) => html`
  <div class="fb-item" data-path="${f.path}">
    <span>${getFileIcon(f.name)}</span>
    <span>${f.name}</span>
  </div>
`;

// html 标签自动转义，防止 XSS
function html(strings, ...values) {
  return strings.reduce((result, str, i) => {
    const val = values[i] != null ? String(values[i]) : '';
    return result + str + val.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
  }, '');
}
```

**收益**：
- 自动 XSS 转义
- 可读性更好
- 无外部依赖

**工时**：3h

### 2.3 事件总线（状态管理）

替代散落各处的 DOM 事件和全局变量：

```js
// src/static/admin/js/core/event-bus.js
class EventBus {
  #listeners = new Map();
  
  on(event, callback) { ... }
  off(event, callback) { ... }
  emit(event, data) { ... }
  once(event, callback) { ... }
}

// 全局单例
window.appEvents = new EventBus();

// 使用：站点状态变更时通知其他组件
appEvents.emit('site:changed', { id: 'xxx', enabled: true });
appEvents.on('site:changed', (data) => { updateSiteList(data); });
```

**应用场景**：
- 站点启停后通知仪表盘刷新
- SSL 证书申请成功后通知站点列表
- FTP 用户变更后通知文件管理器
- 系统设置变更后通知所有标签页

**收益**：
- 组件间解耦
- 避免直接 DOM 操作监听变化
- 支持一对多通知

**工时**：3h

### 2.4 错误边界和全局错误处理

```js
// 全局错误捕获，防止白屏
window.onerror = (msg, url, line, col, error) => {
  appEvents.emit('error', { msg, url, line, error });
  toast.error('操作失败，请刷新页面重试');
  return true; // 阻止默认错误处理
};

window.addEventListener('unhandledrejection', (e) => {
  toast.error('网络请求失败');
  e.preventDefault();
});
```

**工时**：1h

### 2.5 请求层封装（统一 API 调用）

当前 API 调用散落在各处，没有统一处理：
- 无自动重试
- 无统一错误处理
- 无请求取消
- 无 loading 状态管理

```js
// src/static/admin/js/core/api.js
class ApiClient {
  constructor(basePath) {
    this.basePath = basePath;
    this.token = null;
  }

  async request(method, path, options = {}) {
    const { body, timeout = 30000, retry = 1 } = options;
    // 统一：CSRF token、错误处理、超时、重试
  }

  get(path, opts) { return this.request('GET', path, opts); }
  post(path, body, opts) { return this.request('POST', path, { body, ...opts }); }
  delete(path, opts) { return this.request('DELETE', path, opts); }
}
```

**收益**：
- 统一错误处理（401 自动跳转登录、429 提示、500 通用错误）
- 自动 CSRF token
- 请求取消（切换标签页时取消未完成请求）

**工时**：3h

### 2.6 CSS 优化

#### 2.6.1 CSS 变量完善
当前已有 CSS 变量系统，但覆盖不完整。补充：
- 间距变量（`--spacing-xs/sm/md/lg/xl`）
- 圆角变量（`--radius-sm/md/lg`）
- 阴影变量（`--shadow-sm/md/lg`）
- 动画变量（`--transition-fast/normal/slow`）

#### 2.6.2 CSS 文件合并
将组件级 CSS 合并为逻辑分组：
```
css/
├── base.css          # 重置 + CSS 变量 + 通用样式
├── layout.css        # 布局（sidebar、header、tabs）
├── components.css    # 通用组件（modal、button、input、table、toast）
├── tabs.css          # 标签页特定样式
└── themes.css        # 主题（暗色/亮色）
```

**工时**：3h

### 2.7 拖拽排序（文件管理器）

文件管理器支持拖拽：
- 文件/文件夹拖拽移动
- 拖拽排序（列表视图）
- 拖拽上传（拖文件到列表区域）

**工时**：3h

### 2.8 键盘快捷键

| 快捷键 | 功能 |
|--------|------|
| Ctrl+S | 保存（编辑器中） |
| Delete | 删除选中项 |
| F2 | 重命名 |
| Ctrl+A | 全选 |
| Ctrl+C/V/X | 复制/粘贴/剪切 |
| Ctrl+F | 搜索 |
| Esc | 关闭弹窗/取消选择 |

**工时**：2h

---

## 三、分阶段执行

| 阶段 | 内容 | 工时 | 优先级 |
|------|------|------|--------|
| 一 | 错误边界 + 全局错误处理 | 1h | P0 |
| 二 | 请求层封装（ApiClient） | 3h | P0 |
| 三 | CSS 变量完善 + 文件合并 | 3h | P1 |
| 四 | 事件总线（EventBus） | 3h | P1 |
| 五 | 简易模板引擎（html标签） | 3h | P1 |
| 六 | esbuild 构建工具 | 2h | P2 |
| 七 | 键盘快捷键 | 2h | P2 |
| 八 | 拖拽排序 | 3h | P2 |

---

## 四、不做的事

- ❌ 不引入 Vue/React/Angular
- ❌ 不引入 TypeScript（可选，暂不做）
- ❌ 不做 SSR/SSG
- ❌ 不做国际化（P3 长期）
- ❌ 不做 PWA
- ❌ 不做单元测试（前端）
