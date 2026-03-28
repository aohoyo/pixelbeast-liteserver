# 前端开发指南

## 架构概述

前端采用**原生技术栈**，无框架依赖：

- **HTML5** - 语义化结构
- **CSS3** - CSS 变量系统
- **JavaScript ES Modules** - 模块化

### 核心原则

1. **组件化** - UI 组件独立封装
2. **模块化** - ES Modules 导入导出
3. **事件驱动** - 模块间通过事件通信
4. **依赖注入** - 避免硬编码依赖

---

## 目录结构

```
src/static/admin/
├── index.html              # 主页面
├── login.html              # 登录页
│
├── css/
│   ├── base.css            # CSS 变量、重置
│   ├── layout.css          # 布局样式
│   ├── main.css            # 页面样式
│   └── components/         # 组件样式
│       ├── button.css
│       ├── card.css
│       ├── data-table.css
│       └── ...
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
│   │   ├── cache.js        # 缓存管理
│   │   └── loader.js       # 组件加载
│   │
│   ├── components/         # UI 组件
│   │   ├── toast.js        # 消息提示
│   │   ├── dialog.js       # 对话框
│   │   ├── data-table.js   # 数据表格
│   │   ├── skeleton.js     # 骨架屏
│   │   ├── tooltip.js      # 提示框
│   │   ├── file-manager.js # 文件管理器
│   │   └── ...
│   │
│   └── tabs/               # 标签页模块
│       ├── BaseTab.js      # 基类 ⭐
│       ├── home.js         # 首页
│       ├── sites.js        # 站点管理
│       ├── ftp.js          # FTP 管理
│       ├── files.js        # 文件管理
│       ├── logs.js         # 日志查看
│       ├── settings.js     # 系统设置
│       └── cert.js         # 证书管理
│
└── components/             # HTML 模板
    ├── home-section.html
    ├── sites-section.html
    └── ...
```

---

## BaseTab 基类

所有标签页必须继承 `BaseTab`：

### 生命周期

```javascript
import { BaseTab } from './BaseTab.js';

class XxxTab extends BaseTab {
    constructor(deps) {
        super(deps, 'xxx');  // 传入依赖和标签页名称
    }

    // 初始化（只执行一次）
    onInit() {
        this.bindEvents();
        this.initComponents();
    }

    // 加载数据（首次激活）
    async onLoad() {
        const data = await this.api.getJSON('/api/xxx');
        this.render(data);
    }

    // 刷新数据（再次激活）
    async onRefresh() {
        await this.onLoad();
    }

    // 销毁（可选）
    onDestroy() {
        // 清理资源
    }

    // 错误处理（可选覆盖）
    onError(error, context) {
        console.error(`[${this.name}] ${context}:`, error);
        this.toast?.error(`${context}: ${error.message}`);
    }
}

// 导出单例
export default new XxxTab({
    api,
    state,
    toast,
    message,
    dialog,
    events
});
```

### 工具方法

```javascript
// DOM 操作
this.$('#element')           // querySelector
this.$$('.elements')         // querySelectorAll

// 内容设置
this.setText('#el', value)   // 设置文本
this.setHTML('#el', html)    // 设置 HTML

// 加载状态
this.showLoading(container)  // 显示加载
this.showEmpty(container)    // 显示空状态
```

### 依赖注入

```javascript
const deps = {
    api,        // API 实例
    state,      // 状态管理
    toast,      // 消息提示
    message,    // 消息封装
    dialog,     // 对话框
    events      // 事件系统
};
```

---

## 状态管理

### 访问状态

```javascript
// 获取配置
const config = state.get('config');

// 获取系统状态
const status = state.get('status');

// 获取其他数据
const data = state.get('xxx');
```

### 配置结构

```javascript
state.get('config') = {
    admin: { username, port, path },
    http: { port },
    ftp: { enabled, port, root, users },
    log: { retention_days, level },
    backup_dir: './backups'
}
```

### 状态结构

```javascript
state.get('status') = {
    os: 'linux',
    arch: 'amd64',
    memory_mb: 50,
    goroutines: 15,
    uptime: 3600000,
    server_start_time: '2026-03-29T00:00:00Z',
    services: {
        http: { running: true, port: 8080 },
        ftp: { running: false, port: 2121 }
    }
}
```

---

## API 封装

### 基本用法

```javascript
// GET 请求
const data = await api.getJSON('/api/xxx');

// POST 请求
await api.post('/api/xxx', { key: 'value' });

// PUT 请求
await api.put('/api/xxx/id', { key: 'value' });

// DELETE 请求
await api.delete('/api/xxx/id');
```

### 缓存控制

```javascript
// 带缓存（10秒）
const data = await api.getJSON('/api/xxx', { cache: true });

// 清除缓存
api.clearCache('/api/xxx');

// 清除所有缓存
api.clearAllCache();
```

---

## 事件系统

### 触发事件

```javascript
// 触发事件
events.emit('event:name', data);

// 触发标签页切换
events.emit('tab:switch', 'home');
events.emit('tab:switch:home');
```

### 监听事件

```javascript
// 监听事件
events.on('event:name', (data) => {
    console.log(data);
});

// 匹配模式
events.match('tab:switch:xxx', () => {
    // 只匹配 'tab:switch:xxx'
});
```

### 常用事件

```javascript
// 配置加载完成
events.on('config:loaded', (config) => { });

// 状态更新
events.on('status:updated', (status) => { });

// 标签页切换
events.on('tab:switch', (tabName) => { });
events.match('tab:switch:home', () => { });

// 数据刷新
events.emit('refresh:home');
```

---

## 工具函数

```javascript
import { 
    escapeHtml, 
    formatSize, 
    formatUptime, 
    formatDate,
    debounce,
    throttle,
    copyToClipboard,
    generateId,
    deepClone,
    isEmpty,
    get
} from '../core/utils.js';

// HTML 转义
escapeHtml('<script>')  // '&lt;script&gt;'

// 格式化文件大小
formatSize(1024)        // '1 KB'
formatSize(1048576)     // '1 MB'

// 格式化运行时间
formatUptime(3600000)   // '1时0分'

// 格式化日期
formatDate(new Date(), 'datetime')  // '2026-03-29 12:00'

// 防抖
const debouncedFn = debounce(fn, 300);

// 节流
const throttledFn = throttle(fn, 100);

// 复制到剪贴板
await copyToClipboard('text');

// 生成唯一 ID
generateId('item')  // 'item_1712345678_abc123'

// 安全获取嵌套属性
get(obj, 'a.b.c', defaultValue)
```

---

## CSS 规范

### CSS 变量系统

```css
/* 颜色 */
--primary: #f97316;
--primary-light: #fb923c;
--primary-alpha: rgba(249, 115, 22, 0.15);

--success: #22c55e;
--danger: #ef4444;
--warning: #fbbf24;
--info: #3b82f6;

/* 背景 */
--bg: #0c0a09;
--bg-elevated: #1c1917;
--bg-hover: #272524;
--card-bg: #292524;

/* 文字 */
--text: #fafaf9;
--text-secondary: #d6d3d1;
--text-muted: #78716c;

/* 边框 */
--border: #44403c;
--border-light: #57534e;

/* 间距 */
--space-xs: 4px;
--space-sm: 8px;
--space-md: 16px;
--space-lg: 24px;
--space-xl: 32px;

/* 圆角 */
--radius-sm: 4px;
--radius: 8px;
--radius-lg: 12px;
--radius-full: 9999px;

/* 过渡 */
--transition: 250ms ease;
```

### 使用规范

```css
/* ✅ 正确：使用 CSS 变量 */
.card {
    background: var(--card-bg);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    padding: var(--space-lg);
}

/* ❌ 错误：硬编码 */
.card {
    background: #292524;
    border: 1px solid #44403c;
}
```

---

## 组件示例

### Toast 消息

```javascript
// 成功消息
toast.success('操作成功');

// 错误消息
toast.error('操作失败');

// 警告消息
toast.warning('请注意');

// 信息消息
toast.info('提示信息');
```

### Dialog 对话框

```javascript
// 确认对话框
dialog.confirm('确定删除吗？', () => {
    // 确认回调
});

// 提示对话框
dialog.alert('提示', '内容');

// 自定义对话框
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
        { title: '操作', dataIndex: 'id', render: (id) => `<button onclick="edit('${id}')">编辑</button>` }
    ],
    selectable: true,
    batchActions: [
        { key: 'enable', label: '启用', type: 'success', handler: (ids) => {} },
        { key: 'delete', label: '删除', type: 'danger', handler: (ids) => {} }
    ]
});

// 设置数据
table.setData(items);
```

---

## 开发调试

### 浏览器控制台

```javascript
// 查看状态
state.get('config')
state.get('status')

// 查看 API 缓存
api._cache

// 触发事件
events.emit('refresh:home')

// 调用 Tab 方法
window.homeTab.refresh()
```

### 常见问题

1. **组件加载失败** - 检查路径是否正确
2. **事件未触发** - 检查事件名称拼写
3. **状态未更新** - 检查 state.set() 是否调用
4. **API 返回 401** - 检查登录状态