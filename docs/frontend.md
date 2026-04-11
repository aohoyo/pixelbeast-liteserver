# 前端开发指南

## 架构概述

前端采用**原生技术栈**，无框架依赖：

- **HTML5** - 语义化结构
- **CSS3** - CSS 变量系统，组件化样式
- **JavaScript ES Modules** - 模块化，BaseTab 组件模式

### 核心原则

1. **组件化** - UI 组件独立封装（toast、dialog、data-table...）
2. **模块化** - ES Modules 导入导出，单例模式
3. **事件驱动** - 模块间通过 events 系统通信
4. **依赖注入** - 避免硬编码依赖，通过构造函数注入

---

## 目录结构

```
src/static/admin/
├── index.html                  # 主页面 (模板变量: {{SERVER_NAME}}, {{VERSION}})
├── views/
│   └── login.html              # 登录页
│
├── css/
│   ├── base.css                # CSS 变量、重置
│   ├── layout.css              # 布局样式
│   ├── main.css                # 主页面样式
│   ├── login.css               # 登录页样式
│   └── components/             # 组件样式 (独立文件)
│       ├── button.css          # 按钮
│       ├── card.css            # 卡片
│       ├── data-table.css      # 数据表格
│       ├── dialog.css          # 对话框
│       ├── modal.css           # 模态框
│       ├── tooltip.css         # 提示框
│       ├── skeleton.css        # 骨架屏
│       ├── badge.css           # 徽章
│       ├── status.css          # 状态指示
│       ├── icon.css            # 图标
│       ├── file-manager.css    # 文件管理器
│       ├── file-browser.css    # 文件浏览器
│       ├── upload-manager.css  # 上传管理器
│       ├── ftp.css             # FTP 模块
│       ├── sites.css           # 站点模块
│       ├── settings.css        # 设置模块
│       ├── service-tab.css     # 服务管理
│       ├── form.css            # 表单
│       ├── toolbar.css         # 工具栏
│       └── logs.css            # 日志模块
│
├── js/
│   ├── app.js                  # 应用入口 (初始化核心模块、加载标签页)
│   ├── login.js                # 登录逻辑
│   │
│   ├── core/                   # 核心模块
│   │   ├── api.js              # API 封装 (GET/POST/PUT/DELETE、缓存、自动 token)
│   │   ├── state.js            # 状态管理 (get/set)
│   │   ├── events.js           # 事件系统 (emit/on/match)
│   │   ├── utils.js            # 工具函数 (formatSize, formatDate, escapeHtml...)
│   │   ├── cache.js            # 缓存管理 (TTL、LRU)
│   │   └── loader.js           # HTML 组件动态加载
│   │
│   ├── components/             # UI 组件
│   │   ├── toast.js            # 消息提示 (success/error/warning/info)
│   │   ├── dialog.js           # 对话框 (confirm/alert/show)
│   │   ├── message.js          # 消息封装
│   │   ├── data-table.js       # 数据表格 (排序、选择、批量操作)
│   │   ├── skeleton.js         # 骨架屏加载状态
│   │   ├── tooltip.js          # 提示框
│   │   ├── file-icons.js       # 文件类型图标映射
│   │   ├── file-manager.js     # 文件管理器 (浏览、上传、操作)
│   │   ├── upload-manager.js   # 分块上传管理
│   │   └── context-menu.js     # 右键上下文菜单
│   │
│   └── tabs/                   # 标签页模块
│       ├── BaseTab.js          # 基类 (生命周期管理) ⭐
│       ├── home.js             # 首页 (系统监控仪表盘)
│       ├── sites.js            # 站点管理
│       ├── ftp.js              # FTP 管理
│       ├── files.js            # 文件管理器
│       ├── cert.js             # 证书管理
│       ├── logs.js             # 日志查看
│       ├── settings.js         # 系统设置
│       ├── services.js         # 服务管理
│       └── settings-validator.js # 设置表单验证
│
└── components/                 # HTML 模板片段 (动态加载)
    └── *.html
```

---

## BaseTab 基类

所有标签页必须继承 `BaseTab`，遵循统一的生命周期管理。

### 生命周期

```javascript
import { BaseTab } from './BaseTab.js';

class XxxTab extends BaseTab {
    constructor(deps) {
        super(deps, 'xxx');  // deps: { api, state, toast, message, dialog, events }
    }

    onInit() { }           // 初始化（只执行一次）
    onLoad() { }           // 加载数据（首次激活）
    onRefresh() { }        // 刷新数据（再次激活）
    onDestroy() { }        // 销毁清理（可选）

    onError(error, context) {  // 错误处理（可选覆盖）
        console.error(`[${this.name}] ${context}:`, error);
        this.toast?.error(`${context}: ${error.message}`);
    }
}

// 导出单例
export default new XxxTab({ api, state, toast, message, dialog, events });
```

### 工具方法

```javascript
this.$('#element')           // querySelector
this.$$('.elements')         // querySelectorAll
this.setText('#el', value)   // 设置文本
this.setHTML('#el', html)    // 设置 HTML
this.showLoading(container)  // 显示加载骨架
this.showEmpty(container)    // 显示空状态
```

### 依赖注入

```javascript
const deps = {
    api,        // API 实例 (getJSON, post, put, delete)
    state,      // 状态管理 (get, set)
    toast,      // 消息提示 (success, error, warning, info)
    message,    // 消息封装
    dialog,     // 对话框 (confirm, alert, show)
    events      // 事件系统 (emit, on, match)
};
```

---

## 状态管理

```javascript
state.get('config')     // 服务配置
state.get('status')     // 运行状态
state.get('xxx')        // 自定义数据
state.set('xxx', data)  // 设置状态
```

---

## API 封装

```javascript
// 基本请求
const data = await api.getJSON('/api/xxx');
await api.post('/api/xxx', { key: 'value' });
await api.put('/api/xxx/id', { key: 'value' });
await api.delete('/api/xxx/id');

// 缓存控制
const data = await api.getJSON('/api/xxx', { cache: true });
api.clearCache('/api/xxx');
api.clearAllCache();
```

---

## 事件系统

```javascript
// 触发
events.emit('data:updated', newData);
events.emit('tab:switch', 'home');
events.emit('tab:switch:home');

// 监听
events.on('data:updated', (data) => { });
events.match('tab:switch:xxx', () => { });

// 常用事件
'config:loaded'    // 配置加载完成
'status:updated'   // 状态更新
'tab:switch'       // 标签页切换
'tab:switch:xxx'   // 特定标签页切换
'refresh:home'     // 刷新首页
```

---

## 工具函数

```javascript
import { escapeHtml, formatSize, formatUptime, formatDate,
         debounce, throttle, copyToClipboard, generateId, deepClone,
         isEmpty, get } from '../core/utils.js';

escapeHtml('<script>')          // '&lt;script&gt;'
formatSize(1048576)             // '1 MB'
formatDate(new Date(), 'datetime')  // '2026-04-10 12:00'
const debouncedFn = debounce(fn, 300);
const throttledFn = throttle(fn, 100);
await copyToClipboard('text');
generateId('item')              // 'item_1712345678_abc123'
get(obj, 'a.b.c', default)
```

---

## 组件示例

### Toast 消息

```javascript
toast.success('操作成功');
toast.error('操作失败');
toast.warning('请注意');
toast.info('提示信息');
```

### Dialog 对话框

```javascript
dialog.confirm('确定删除吗？', () => { /* 确认 */ });
dialog.alert('提示', '内容');
dialog.show({
    title: '标题',
    content: '<p>内容</p>',
    buttons: [
        { text: '取消', action: () => {} },
        { text: '确定', type: 'primary', action: () => {} }
    ]
});
```

### DataTable 数据表格

```javascript
const table = new DataTable({
    container: '#table-container',
    columns: [
        { title: '名称', dataIndex: 'name' },
        { title: '状态', dataIndex: 'status', render: (val) => val ? '✅' : '❌' },
    ],
    selectable: true,
    batchActions: [
        { key: 'delete', label: '删除', type: 'danger', handler: (ids) => {} }
    ]
});
table.setData(items);
```

---

## CSS 规范

### CSS 变量系统

```css
/* 颜色 */
--primary: #f97316;            --primary-light: #fb923c;
--success: #22c55e;            --danger: #ef4444;
--warning: #fbbf24;            --info: #3b82f6;

/* 背景 */
--bg: #0c0a09;                 --bg-elevated: #1c1917;
--bg-hover: #272524;           --card-bg: #292524;

/* 文字 */
--text: #fafaf9;               --text-secondary: #d6d3d1;
--text-muted: #78716c;

/* 边框 */
--border: #44403c;             --border-light: #57534e;

/* 间距 */
--space-xs: 4px;               --space-sm: 8px;
--space-md: 16px;              --space-lg: 24px;   --space-xl: 32px;

/* 圆角 */
--radius-sm: 4px;              --radius: 8px;
--radius-lg: 12px;             --radius-full: 9999px;

/* 过渡 */
--transition: 250ms ease;
```

### 规则

```css
/* ✅ 使用 CSS 变量 */
.card { background: var(--card-bg); border: 1px solid var(--border); }

/* ❌ 禁止硬编码 */
.card { background: #292524; border: 1px solid #44403c; }
```

---

## 开发调试

```javascript
// 浏览器控制台
state.get('config')          // 查看配置
state.get('status')          // 查看状态
api._cache                   // 查看 API 缓存
events.emit('refresh:home')  // 触发刷新
```
