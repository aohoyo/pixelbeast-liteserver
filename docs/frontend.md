# 前端开发

## 设计模式

管理面板采用**组件化 + 模块化 JS** 架构：

- 标签页组件存储为独立的 HTML 文件
- 通过 `loader.js` 动态加载组件
- 更易于维护和扩展

## 目录结构

```
src/static/admin/
├── index.html         # 主面板（容器）
├── login.html         # 登录页
├── components/        # 标签页组件
│   ├── home-section.html
│   ├── sites-section.html
│   ├── ftp-section.html
│   ├── files-section.html
│   ├── logs-section.html
│   ├── cert-section.html
│   ├── settings-section.html
│   └── modal.html
├── css/
│   ├── base.css       # CSS 变量、重置样式
│   ├── main.css       # 主面板样式
│   └── login.css      # 登录页样式
└── js/
    ├── app.js         # 应用入口
    ├── login.js       # 登录逻辑
    ├── core/          # 核心模块
    │   ├── state.js   # 状态管理
    │   ├── api.js     # API 封装
    │   ├── events.js  # 事件系统
    │   └── loader.js  # 组件加载器
    ├── components/    # JS 组件
    │   ├── toast.js   # Toast 提示（右下角）
    │   ├── message.js # Message 消息（顶部居中）
    │   ├── dialog.js  # 对话框
    │   └── tooltip.js # 工具提示
    └── tabs/          # 标签页逻辑
        ├── home.js    # 首页（系统概览）
        ├── sites.js   # 网站管理
        ├── ftp.js     # FTP 文件管理
        ├── files.js   # HTTP 文件管理
        ├── logs.js    # 日志查看
        ├── cert.js    # 证书管理
        └── settings.js # 配置编辑
```

## 核心模块

### state.js - 状态管理器

```javascript
import StateManager from './core/state.js';

const state = new StateManager();

// 设置值
state.set('key', value);

// 获取值
const value = state.get('key');

// 订阅变化
state.subscribe('key', (newValue, oldValue) => {
    console.log(`${key}: ${oldValue} -> ${newValue}`);
});

// 批量更新
state.batch({
    'key1': value1,
    'key2': value2
});
```

### api.js - API 封装

```javascript
import { createAPI } from './core/api.js';

const api = createAPI(state);

// GET 请求
const response = await api.get('/api/status');
const data = await api.parseJSON(response);

// POST 请求
const response = await api.post('/api/config/save', config);

// 文件上传
await api.uploadFile('/api/files/upload', file, { path: '/' });

// 获取 CSRF Token
const token = api.getCSRFToken();
```

### events.js - 事件系统

```javascript
import { globalEvents } from './core/events.js';

// 监听事件
globalEvents.on('tab:switch:home', (data) => {
    console.log('切换到首页', data);
});

// 通配符匹配
globalEvents.match('tab:switch:*', (event, data) => {
    console.log(`切换标签: ${event}`, data);
});

// 触发事件
globalEvents.emit('status:loaded', { memory: 100 });
globalEvents.emit('tab:switch:home');
```

## 标签页开发

### 标签页模板

```javascript
// src/static/admin/js/tabs/mytab.js

import { globalEvents } from '../core/events.js';

/**
 * 初始化标签页
 * @param {Object} dependencies - 依赖注入 { state, api, toast, message, dialog, events }
 */
export function initMyTab({ state, api, toast, message, dialog, events }) {
    console.log('🔧 初始化我的标签页...');

    // 监听标签页切换
    events.match('tab:switch:mytab', () => {
        loadData();
    });

    // 初始加载
    loadData();

    /**
     * 加载数据
     */
    async function loadData() {
        try {
            const data = await api.getJSON('/api/mydata');
            updateUI(data);
        } catch (error) {
            toast.error('加载失败: ' + error.message);
        }
    }

    /**
     * 保存数据
     */
    async function saveData() {
        try {
            await api.post('/api/mydata', data);
            message.success('保存成功');
        } catch (error) {
            message.error('保存失败: ' + error.message);
        }
    }

    /**
     * 更新 UI
     */
    function updateUI(data) {
        // 更新 DOM
    }
}
```

### 添加新标签页步骤

1. **在 `index.html` 中添加 HTML 结构**：

```html
<section id="mytab" class="tab-content">
    <div class="card">
        <div class="card-header">
            <h3>我的标签页</h3>
        </div>
        <div class="card-body">
            <!-- 内容 -->
        </div>
    </div>
</section>
```

2. **在导航栏添加按钮**：

```html
<button class="nav-item" data-tab="mytab">
    <span class="nav-icon">🔧</span>
    <span class="nav-label">我的标签</span>
</button>
```

3. **创建标签页 JS 文件**：

```javascript
// src/static/admin/js/tabs/mytab.js
export function initMyTab({ state, api, toast }) {
    // 实现
}
```

4. **在 `app.js` 中注册**：

```javascript
import { initMyTab } from './tabs/mytab.js';

// 在 initTabModules() 中调用
initMyTab({ state, api, toast, message, dialog, events: globalEvents });
```

## UI 组件

### Toast 提示

```javascript
import toast from './ui/toast.js';

// 成功提示
toast.success('操作成功');

// 错误提示
toast.error('操作失败');

// 信息提示
toast.info('提示信息');

// 警告提示
toast.warning('警告信息');
```

### Message 消息提示

参考 Element UI Message 组件设计，顶部居中显示的消息提示。

```javascript
import message from './components/message.js';

// 基础用法
message.success('操作成功');
message.error('操作失败');
message.warning('警告信息');
message.info('提示信息');

// 配置对象
message({
    message: '这是一条消息',
    type: 'success',        // success | warning | info | error
    duration: 3000,         // 持续时间，0 表示不自动关闭
    showClose: true,        // 显示关闭按钮
    onClose: () => {}       // 关闭回调
});

// 关闭所有消息
message.closeAll();
```

### Message vs Toast 使用规范

根据场景选择合适的组件：

| 场景 | 组件 | 原因 |
|------|------|------|
| **创建/编辑/删除** | Message | 重要操作，需要用户确认结果 |
| **保存设置成功/失败** | Message | 重要操作反馈，醒目 |
| **文件上传完成** | Message | 用户等待的操作结果 |
| **表单验证失败** | Message | 错误提示需要醒目 |
| **批量操作结果** | Message | 重要操作，需要明确反馈 |
| **复制成功** | Message | 用户需要确认操作成功 |
| **状态开关切换** | Toast | 快速反馈，用户已预期结果 |
| **开始下载** | Toast | 后台操作提示 |
| **加载失败** | Toast | 网络错误，不需要太醒目 |
| **功能开发中** | Toast | 临时提示 |

**注意**：随机密码生成不需要通知，用户可以从输入框直接看到结果。

**位置差异**：
- **Message**: 顶部居中，从上往下淡入，更醒目
- **Toast**: 右下角显示，不打断用户操作

## CSS 变量

在 `base.css` 中定义的全局变量：

```css
:root {
    /* 颜色 */
    --color-primary: #4a90d9;
    --color-success: #52c41a;
    --color-warning: #faad14;
    --color-danger: #f5222d;
    --color-text: #333;
    --color-text-secondary: #666;
    --color-bg: #f5f5f5;
    --color-border: #e8e8e8;

    /* 间距 */
    --spacing-xs: 4px;
    --spacing-sm: 8px;
    --spacing-md: 16px;
    --spacing-lg: 24px;
    --spacing-xl: 32px;

    /* 圆角 */
    --radius-sm: 4px;
    --radius-md: 8px;
    --radius-lg: 12px;

    /* 阴影 */
    --shadow-sm: 0 1px 2px rgba(0,0,0,0.05);
    --shadow-md: 0 4px 12px rgba(0,0,0,0.1);
    --shadow-lg: 0 8px 24px rgba(0,0,0,0.15);
}
```

## 开发规范

### 命名约定

- **文件名**: 小写，连字符分隔 (`my-tab.js`)
- **函数名**: 驼峰命名 (`initMyTab`)
- **变量名**: 驼峰命名 (`currentPath`)
- **常量名**: 大写下划线 (`MAX_SIZE`)

### 依赖注入

所有标签页初始化函数接收统一的依赖对象：

```javascript
export function initXxxTab({ state, api, toast, message, dialog, events }) {
    // 使用注入的依赖，不要全局导入
}
```

### 事件驱动

使用事件系统进行模块间通信：

```javascript
// 发布事件
globalEvents.emit('data:updated', newData);

// 订阅事件
globalEvents.on('data:updated', (data) => {
    // 处理更新
});
```

### 错误处理

统一使用 try-catch，根据场景选择 toast 或 message：

```javascript
// 加载数据（使用 toast，不阻断用户）
async function loadData() {
    try {
        const data = await api.getJSON('/api/data');
        // 处理数据
    } catch (error) {
        console.error('加载失败:', error);
        toast.error('加载失败: ' + error.message);
    }
}

// 保存数据（使用 message，重要操作）
async function saveData() {
    try {
        await api.post('/api/data', data);
        message.success('保存成功');
    } catch (error) {
        console.error('保存失败:', error);
        message.error('保存失败: ' + error.message);
    }
}
}
```

## 调试技巧

### 查看状态

```javascript
// 在浏览器控制台
console.log(window.state.get('currentTab'));
```

### 查看事件

```javascript
// 监听所有事件
globalEvents.match('*', (event, data) => {
    console.log(`[Event] ${event}:`, data);
});
```

### 查看 CSRF Token

```javascript
console.log(window.csrfToken);
```
