/**
 * 像素兽 1.0 - 应用入口
 *
 * 模块化架构的主入口文件
 * 组件化模式 - 动态加载 HTML 组件
 */

import StateManager from './core/state.js';
import { createAPI } from './core/api.js';
import { globalEvents } from './core/events.js';
import toast from './components/toast.js';
import message from './components/message.js';
import dialog from './components/dialog.js';
import tooltip from './components/tooltip.js';
import { loadContentSections, loadModal } from './core/loader.js';

// 导入标签页模块
import { initHomeTab } from './tabs/home.js';
import { initSitesTab } from './tabs/sites.js';
import { initFtpTab } from './tabs/ftp.js';
import { initFilesTab } from './tabs/files.js';
import { initSettingsTab } from './tabs/settings.js';
import { initLogsTab } from './tabs/logs.js';
import { initCertTab } from './tabs/cert.js';

// 应用状态
const state = new StateManager();

// API 实例
const api = createAPI(state);

// 应用配置
const config = {
    refreshInterval: 30000,  // 30秒自动刷新状态
    autoRefresh: true
};

// 自动刷新定时器
let refreshTimer = null;

/**
 * 初始化应用
 */
async function init() {
    console.log('🪶 像素兽 1.0 管理面板初始化...');

    try {
        // 1. 加载组件
        await loadComponents();

        // 2. 初始化事件监听
        initEventListeners();

        // 3. 检查认证状态
        checkAuth();

        // 4. 初始化标签页
        initTabs();

        // 5. 初始化各标签页模块
        initTabModules();

        // 6. 激活默认标签页（触发数据加载）
        const defaultTab = state.get('currentTab') || 'home';
        switchTab(defaultTab);

        // 7. 加载初始数据
        loadInitialData();

        // 7. 设置自动刷新
        if (config.autoRefresh) {
            startAutoRefresh();
        }

        console.log('✅ 像素兽 1.0 初始化完成');
    } catch (error) {
        console.error('❌ 初始化失败:', error);
        message.error('初始化失败: ' + error.message);
    }
}

/**
 * 加载组件
 */
async function loadComponents() {
    console.log('📦 加载组件...');
    try {
        // 加载所有内容区域组件
        await loadContentSections();
        // 加载模态框
        await loadModal();
        // 初始化 tooltip
        tooltip.init();
        console.log('✅ 组件加载完成');
    } catch (error) {
        console.error('❌ 组件加载失败:', error);
        throw error;
    }
}

/**
 * 检查认证状态
 */
function checkAuth() {
    const currentPath = window.location.pathname;
    const loginPath = currentPath.replace(/\/$/, '') + '/login';

    // 尝试获取状态来验证认证
    api.get('/api/status')
        .then(response => {
            if (response && response.ok) {
                // 已认证，继续
                console.log('✅ 用户已认证');
            } else {
                // 未认证，跳转到登录页
                console.log('⚠️ 用户未认证，跳转到登录页');
                window.location.href = loginPath;
            }
        })
        .catch(() => {
            // 请求失败，可能需要登录
            window.location.href = loginPath;
        });
}

/**
 * 初始化事件监听
 */
function initEventListeners() {
    // 全局事件监听
    globalEvents.match('auth:*', (event, data) => {
        console.log(`[Auth Event] ${event}:`, data);
    });

    globalEvents.match('api:*', (event, data) => {
        console.log(`[API Event] ${event}:`, data);
    });

    // 键盘快捷键
    document.addEventListener('keydown', (e) => {
        // ESC 键关闭模态框
        if (e.key === 'Escape') {
            globalEvents.emit('ui:closeModal');
            document.querySelector('.modal.active')?.classList.remove('active');
        }
    });

    // 页面可见性变化
    document.addEventListener('visibilitychange', () => {
        if (document.hidden) {
            stopAutoRefresh();
        } else {
            if (config.autoRefresh) {
                startAutoRefresh();
            }
        }
    });

    // 页面卸载前清理
    window.addEventListener('beforeunload', () => {
        stopAutoRefresh();
    });

    // 登出按钮
    const logoutBtn = document.getElementById('logout-btn');
    logoutBtn?.addEventListener('click', logout);
}

/**
 * 初始化标签页
 */
function initTabs() {
    // 支持 .menu-item
    const tabButtons = document.querySelectorAll('.menu-item');

    tabButtons.forEach((button) => {
        const tabName = button.dataset.tab;
        if (tabName) {
            button.addEventListener('click', () => {
                switchTab(tabName);
            });
        }
    });

    // 默认激活第一个标签页
    const defaultTab = tabButtons[0]?.dataset.tab || 'home';
    state.set('currentTab', defaultTab, false);
}

/**
 * 初始化标签页模块
 */
function initTabModules() {
    const dependencies = { state, api, toast, message, dialog, events: globalEvents };

    // 初始化各标签页
    initHomeTab(dependencies);
    initSitesTab(dependencies);
    initFtpTab(dependencies);
    initFilesTab(dependencies);
    initSettingsTab(dependencies);
    initLogsTab(dependencies);
    initCertTab(dependencies);
}

// 标签页标题映射
const tabTitles = {
    'home': '首页',
    'sites': '网站管理',
    'ftp': 'FTP 服务',
    'files': '文件管理',
    'logs': '系统日志',
    'cert': 'SSL 证书',
    'settings': '系统设置'
};

/**
 * 切换标签页
 * @param {string} tabName - 标签页名称
 */
function switchTab(tabName) {
    // 关闭批量操作条
    const batchBar = document.getElementById('dt-batch-bar');
    const batchContainer = document.getElementById('dt-batch-bar-container');
    if (batchBar) batchBar.classList.remove('show');
    if (batchContainer) batchContainer.classList.remove('active');
    
    // 更新按钮状态
    const buttons = document.querySelectorAll('.menu-item');
    buttons.forEach(btn => {
        if (btn.dataset.tab === tabName) {
            btn.classList.add('active');
        } else {
            btn.classList.remove('active');
        }
    });

    // 更新内容显示
    const contents = document.querySelectorAll('.tab-content');
    contents.forEach(content => {
        content.classList.remove('active');
        if (content.id === tabName) {
            content.classList.add('active');
        }
    });

    // 更新页面标题
    const pageTitle = document.getElementById('page-title');
    if (pageTitle && tabTitles[tabName]) {
        pageTitle.textContent = tabTitles[tabName];
    }

    // 更新状态
    state.set('currentTab', tabName);

    // 触发标签页切换事件
    globalEvents.emit(`tab:switch:${tabName}`, { tabName });
    globalEvents.emit('tab:switch:*', { tabName });
}

/**
 * 加载初始数据
 */
function loadInitialData() {
    // 初始加载系统状态数据
    api.get('/api/system/status')
        .then(async response => {
            if (response && response.ok) {
                const data = await api.parseJSON(response);
                if (data) {
                    // 初始化系统信息（只执行一次）
                    state.initSystem({
                        os: data.os,
                        arch: data.arch,
                        hostname: data.hostname
                    });

                    state.batch({
                        'services': {
                            http: { running: true, port: data.http_port },
                            ftp: { running: data.ftp_running, port: data.ftp_port }
                        }
                    });

                    globalEvents.emit('status:loaded', data);
                }
            }
        })
        .catch(error => {
            console.error('加载初始数据失败:', error);
        });
}

/**
 * 启动自动刷新
 */
function startAutoRefresh() {
    if (refreshTimer) return;

    refreshTimer = setInterval(() => {
        const currentTab = state.get('currentTab');
        if (currentTab === 'status') {
            api.get('/api/status')
                .then(async response => {
                    if (response && response.ok) {
                        const data = await api.parseJSON(response);
                        if (data) {
                            globalEvents.emit('status:loaded', data);
                        }
                    }
                })
                .catch(() => {});
        }
    }, config.refreshInterval);
}

/**
 * 停止自动刷新
 */
function stopAutoRefresh() {
    if (refreshTimer) {
        clearInterval(refreshTimer);
        refreshTimer = null;
    }
}

/**
 * 登出
 */
function logout() {
    dialog.confirm('确定要退出登录吗？', () => {
        api.post('/api/logout')
            .then(() => {
                window.location.href = 'login';
            })
            .catch(() => {
                window.location.href = 'login';
            });
    });
}

// 导出到全局，供 HTML 中使用
window.app = {
    state,
    api,
    events: globalEvents,
    toast,
    message,
    dialog,
    switchTab,
    logout
};

// 导出便捷访问
window.state = state;
window.events = globalEvents;

// 导出 CSRF token 访问器（使用 getter 实现实时访问）
Object.defineProperty(window, 'csrfToken', {
    get: function() { return api.getCSRFToken(); },
    enumerable: true
});

// 页面加载完成后初始化
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
