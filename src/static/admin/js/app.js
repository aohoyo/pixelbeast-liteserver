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
import { escapeHtml } from './core/utils.js';

// 标签页模块动态加载注册表
const APP_VERSION = document.querySelector('meta[name="app-version"]')?.content || 'dev';
const vSuffix = APP_VERSION !== 'dev' ? `?v=${APP_VERSION}` : '';

const TAB_MODULES = {
    home:     () => import(`./tabs/home.js${vSuffix}`),
    sites:    () => import(`./tabs/sites.js${vSuffix}`),
    ftp:      () => import(`./tabs/ftp.js${vSuffix}`),
    files:    () => import(`./tabs/files.js${vSuffix}`),
    terminal: () => import(`./tabs/terminal.js${vSuffix}`),
    settings: () => import(`./tabs/settings.js${vSuffix}`),
    logs:     () => import(`./tabs/logs.js${vSuffix}`),
    cert:     () => import(`./tabs/cert.js${vSuffix}`),
};

// 已加载的标签页模块缓存
const loadedTabs = new Map();

async function loadTabModule(tabName) {
    if (loadedTabs.has(tabName)) return loadedTabs.get(tabName);
    const loader = TAB_MODULES[tabName];
    if (!loader) return null;
    const mod = await loader();
    loadedTabs.set(tabName, mod);
    return mod;
}

// 标签页初始化函数名映射
const TAB_INIT_FNS = {
    home:     'initHomeTab',
    sites:    'initSitesTab',
    ftp:      'initFtpTab',
    files:    'initFilesTab',
    terminal: 'initTerminalTab',
    settings: 'initSettingsTab',
    logs:     'initLogsTab',
    cert:     'initCertTab',
};

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

/**
 * 初始化应用
 */
async function init() {
    try {
        // 1. 加载组件
        await loadComponents();

        // 2. 初始化事件监听
        initEventListeners();

        // 3. 从 meta 标签初始化 CSRF token
        const csrfMeta = document.querySelector('meta[name="csrf-token"]');
        if (csrfMeta && csrfMeta.content) {
            api.setCSRFToken(csrfMeta.content);
        }

        // 4. 检查认证状态
        checkAuth();

        // 5. 初始化标签页
        initTabs();

        // 6. 初始化各标签页模块
        initTabModules();

        // 7. 激活默认标签页（触发数据加载）
        const defaultTab = state.get('currentTab') || 'home';
        switchTab(defaultTab);

        // 7. 加载初始数据
        loadInitialData();
    } catch (error) {
        message.error('初始化失败: ' + error.message);
    }
}

/**
 * 加载组件
 */
async function loadComponents() {
    try {
        // 加载所有内容区域组件
        await loadContentSections();
        // 加载模态框
        await loadModal();
        // 初始化 tooltip
        tooltip.init();
    } catch (error) {
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
    api.get('/api/system/status')
        .then(response => {
            if (response && response.ok) {
                // 已认证，继续
            } else {
                // 未认证，跳转到登录页
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
    });

    globalEvents.match('api:*', (event, data) => {
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
        if (!document.hidden) {
            // 页面重新可见时刷新当前 tab
            const currentTab = state.get('currentTab');
            if (currentTab) {
                globalEvents.emit(`refresh:${currentTab}`);
            }
        }
    });

    // 登出按钮
    const logoutBtn = document.getElementById('logout-btn');
    logoutBtn?.addEventListener('click', logout);

    // 头部按钮：更新检查 & 重启
    const updateBtn = document.getElementById('header-update-btn');
    const restartBtn = document.getElementById('header-restart-btn');
    updateBtn?.addEventListener('click', checkForUpdate);
    restartBtn?.addEventListener('click', restartServer);
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
 * 初始化标签页模块（仅加载首页）
 */
async function initTabModules() {
    const dependencies = { state, api, toast, message, dialog, events: globalEvents };

    // 只初始化首页标签（默认激活页），其余标签首次切换时加载
    const homeMod = await loadTabModule('home');
    if (homeMod) homeMod.initHomeTab(dependencies);
}

// 标签页标题映射
const tabTitles = {
    'home': '首页',
    'sites': '网站管理',
    'ftp': 'FTP 服务',
    'files': '文件管理',
    'terminal': '终端',
    'logs': '系统日志',
    'cert': 'SSL 证书',
    'settings': '系统设置'
};

/**
 * 切换标签页
 * @param {string} tabName - 标签页名称
 */
async function switchTab(tabName) {
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

    // 按需加载标签页模块
    const mod = await loadTabModule(tabName);
    if (mod) {
        const initFn = TAB_INIT_FNS[tabName];
        if (initFn && mod[initFn]) {
            const dependencies = { state, api, toast, message, dialog, events: globalEvents };
            mod[initFn](dependencies);
        }
    }

    // 触发标签页切换事件
    globalEvents.emit(`tab:switch:${tabName}`, { tabName });
    globalEvents.emit('tab:switch:*', { tabName });
}

/**
 * 更新顶部栏系统信息
 */
// 操作系统图标映射（官方品牌 Logo）
function getOSIcon(osName) {
    const lower = (osName || '').toLowerCase();
    if (lower.includes('debian')) return 'debian';
    if (lower.includes('ubuntu')) return 'ubuntu';
    if (lower.includes('centos')) return 'centos';
    if (lower.includes('fedora')) return 'fedora';
    if (lower.includes('arch')) return 'arch';
    if (lower.includes('opensuse') || lower.includes('suse')) return 'opensuse';
    if (lower.includes('alpine')) return 'alpine';
    if (lower.includes('macos') || lower.includes('darwin')) return 'macos';
    if (lower.includes('windows')) return 'windows';
    if (lower.includes('linux')) return 'linux';
    return 'linux';
}

function updateHeaderOS(data) {
    const osNameShort = data.os_name_short || data.os_name || data.os || '--';
    const osNameFull = data.os_name || data.os || '--';
    const nameEl = document.getElementById('header-os-name');
    if (nameEl) nameEl.textContent = osNameShort;

    // 启动双计时器：服务 + 系统
    startUptimeTickers(data.server_uptime_ms, data.system_uptime_ms);

    const iconEl = document.getElementById('header-os-icon');
    if (iconEl) {
        const iconKey = getOSIcon(osNameFull);
        const base = new URL('.', window.location.href).pathname;
        iconEl.src = `${base}images/os/${iconKey}.svg`;
        iconEl.alt = iconKey;
    }

    const setText = (id, val) => {
        const el = document.getElementById(id);
        if (el) el.textContent = val || '--';
    };
    setText('os-hostname', data.hostname);
    setText('os-name-full', osNameFull);
    setText('os-arch', data.arch);
    setText('os-kernel', data.kernel);
}

// 运行时间计时器（服务 + 系统）
let _tickTimer = null;
let _tickStart = 0;
let _tickServerMs = 0;
let _tickSystemMs = 0;

function formatDuration(ms) {
    const s = Math.floor(ms / 1000);
    const d = Math.floor(s / 86400);
    const h = Math.floor((s % 86400) / 3600);
    const m = Math.floor((s % 3600) / 60);
    const sec = s % 60;
    if (d > 0) return `${d}天 ${h}时 ${m}分 ${sec}秒`;
    if (h > 0) return `${h}时 ${m}分 ${sec}秒`;
    if (m > 0) return `${m}分 ${sec}秒`;
    return `${sec}秒`;
}

function startUptimeTickers(serverMs, systemMs) {
    _tickServerMs = serverMs || 0;
    _tickSystemMs = systemMs || 0;
    _tickStart = Date.now();
    if (_tickTimer) clearInterval(_tickTimer);
    const tick = () => {
        const elapsed = Date.now() - _tickStart;
        const serverEl = document.getElementById('os-uptime');
        const systemEl = document.getElementById('os-system-uptime');
        if (serverEl && _tickServerMs > 0) serverEl.textContent = formatDuration(_tickServerMs + elapsed);
        if (systemEl && _tickSystemMs > 0) systemEl.textContent = formatDuration(_tickSystemMs + elapsed);
    };
    tick();
    _tickTimer = setInterval(tick, 1000);
}

/**
 * 加载初始数据
 */
async function loadInitialData() {
    try {
        // 1. 加载系统状态
        const statusResponse = await api.get('/api/system/status');
        if (statusResponse && statusResponse.ok) {
            const data = await api.parseJSON(statusResponse);
            if (data) {
                // 初始化系统信息（只执行一次）
                state.initSystem({
                    os: data.os,
                    arch: data.arch,
                    hostname: data.hostname,
                    osName: data.os_name,
                    osNameShort: data.os_name_short,
                    kernel: data.kernel
                });

                // 更新顶部栏系统信息
                updateHeaderOS(data);

                globalEvents.emit('status:loaded', data);

                // 检查是否需要修改密码
                if (data.require_password_change) {
                    showPasswordChangeDialog();
                }
            }
        }

        // 2. 加载全局配置（统一来源）
        const configResponse = await api.get('/api/config');
        if (configResponse && configResponse.ok) {
            const config = await api.parseJSON(configResponse);
            if (config) {
                state.set('config', config);
                globalEvents.emit('config:loaded', config);
            }
        }
    } catch (error) {
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

async function checkForUpdate() {
    const btn = document.getElementById('header-update-btn');
    if (!btn) return;

    btn.classList.add('spinning');

    try {
        const resp = await api.getJSON('/api/system/check-update');
        if (!resp) return;

        if (resp.has_update) {
            const ver = resp.latest_version || '';
            const dlUrl = resp.download_url || '';
            const changelog = resp.changelog || '暂无更新日志';

            dialog.alert(`
                <div style="text-align:left;margin-bottom:12px">
                    <p style="font-size:0.9rem;color:var(--text-secondary)">发现新版本</p>
                    <p style="font-size:1.25rem;font-weight:700;color:var(--primary)">${escapeHtml(ver)}</p>
                    <p style="font-size:0.8rem;color:var(--text-secondary);white-space:pre-line;max-height:200px;overflow-y:auto">${escapeHtml(changelog)}</p>
                    ${dlUrl ? `<a href="${escapeHtml(dlUrl)}" target="_blank" class="btn" style="margin-top:8px;display:inline-block">下载更新</a>` : ''}
                </div>
            `, '更新');
        } else {
            toast.success(resp.message || '已是最新版本');
        }
    } catch (error) {
        toast.error('更新失败: ' + error.message);
    } finally {
        btn.classList.remove('spinning');
    }
}

async function restartServer() {
    dialog.confirm('确定要重启服务吗？重启期间会短暂断开连接。', async () => {
        try {
            await api.post('/api/system/restart');
            toast.success('服务正在重启...');
            setTimeout(() => {
                window.location.reload();
            }, 3000);
        } catch (error) {
            toast.error('重启失败: ' + error.message);
        }
    });
}

/**
 * 首次登录强制修改密码弹窗
 */
function showPasswordChangeDialog() {
    const overlay = document.createElement('div');
    overlay.style.cssText = 'position:fixed;top:0;left:0;width:100%;height:100%;background:rgba(0,0,0,0.5);z-index:10000;display:flex;align-items:center;justify-content:center;';

    const dialog = document.createElement('div');
    dialog.style.cssText = 'background:var(--bg-primary,#fff);border-radius:12px;padding:32px;width:400px;max-width:90vw;box-shadow:0 20px 60px rgba(0,0,0,0.3);';
    dialog.innerHTML = `
        <div style="text-align:center;margin-bottom:20px">
            <div style="font-size:48px">🔐</div>
            <h3 style="margin:12px 0 8px;color:var(--text-primary,#333)">为了您的账户安全</h3>
            <p style="color:var(--text-secondary,#666);font-size:14px">检测到您正在使用初始密码，请立即修改。</p>
        </div>
        <form id="force-change-pwd-form">
            <div style="margin-bottom:12px">
                <input type="password" id="force-old-pwd" placeholder="原密码" required
                    style="width:100%;padding:10px 12px;border:1px solid var(--border-color,#ddd);border-radius:8px;box-sizing:border-box;font-size:14px">
            </div>
            <div style="margin-bottom:12px">
                <input type="password" id="force-new-pwd" placeholder="新密码（至少8位，含大小写字母和数字）" required minlength="8"
                    style="width:100%;padding:10px 12px;border:1px solid var(--border-color,#ddd);border-radius:8px;box-sizing:border-box;font-size:14px">
            </div>
            <div style="margin-bottom:16px">
                <input type="password" id="force-confirm-pwd" placeholder="确认新密码" required
                    style="width:100%;padding:10px 12px;border:1px solid var(--border-color,#ddd);border-radius:8px;box-sizing:border-box;font-size:14px">
            </div>
            <div id="force-pwd-error" style="color:#e74c3c;font-size:13px;margin-bottom:8px;display:none"></div>
            <button type="submit" style="width:100%;padding:12px;background:var(--primary,#4a90d9);color:#fff;border:none;border-radius:8px;font-size:15px;cursor:pointer">
                修改密码
            </button>
        </form>
    `;

    overlay.appendChild(dialog);
    document.body.appendChild(overlay);

    const errorEl = dialog.querySelector('#force-pwd-error');

    dialog.querySelector('#force-change-pwd-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const oldPwd = dialog.querySelector('#force-old-pwd').value;
        const newPwd = dialog.querySelector('#force-new-pwd').value;
        const confirmPwd = dialog.querySelector('#force-confirm-pwd').value;

        errorEl.style.display = 'none';

        if (newPwd !== confirmPwd) {
            errorEl.textContent = '两次输入的密码不一致';
            errorEl.style.display = 'block';
            return;
        }
        if (newPwd.length < 8) {
            errorEl.textContent = '密码长度至少 8 位';
            errorEl.style.display = 'block';
            return;
        }
        if (!/[a-z]/.test(newPwd) || !/[A-Z]/.test(newPwd) || !/[0-9]/.test(newPwd)) {
            errorEl.textContent = '密码必须包含大小写字母和数字';
            errorEl.style.display = 'block';
            return;
        }

        try {
            const resp = await api.post('/api/auth/change-password', {
                old_password: oldPwd,
                new_password: newPwd
            });
            if (resp && resp.ok) {
                overlay.remove();
                toast.success('密码修改成功！');
            } else {
                const data = await api.parseJSON(resp).catch(() => null);
                errorEl.textContent = data?.message || '修改失败';
                errorEl.style.display = 'block';
            }
        } catch (err) {
            errorEl.textContent = err.message || '网络错误';
            errorEl.style.display = 'block';
        }
    });
}
