/**
 * 设置模块
 *
 * 负责系统设置的加载、编辑、保存
 */

import { globalEvents } from '../core/events.js';
import { settingsValidator } from './settings-validator.js';

/**
 * 默认配置模板
 */
const DEFAULT_CONFIG = {
    http: {
        enabled: true,
        port: 8080,
        root: './web',
        domain: ''
    },
    ftp: {
        enabled: true,
        port: 2121,
        root: './ftp',
        users: [
            { username: 'flash', password: 'flash' }
        ]
    },
    admin: {
        enabled: true,
        path: '/admin',
        username: 'admin',
        password: 'admin123',
        port: 9527,
        domain: '',
        ssl: false
    },
    security: {
        allowIps: [],
        sessionTimeout: 86400,
        enable2FA: false,
        logOperations: true,
        confirmDangerous: true
    },
    directories: {
        defaultWebRoot: './web',
        defaultBackupDir: './backups'
    }
};

// 存储依赖以便在闭包函数中使用
let deps = null;

// 存储当前配置
let currentConfig = null;

// 存储原始表单值快照（用于取消时恢复）
let originalFormValues = {};

// 跟踪是否有未保存的修改
let hasUnsavedChanges = false;
// 当前活跃的 Tab
let currentTab = 'panel';

/**
 * 生成随机密码
 * @param {number} length - 密码长度
 * @returns {string} 随机密码
 */
function generateRandomPassword(length = 16) {
    const charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
    let password = '';

    // 确保包含各种字符类型
    password += 'abcdefghijklmnopqrstuvwxyz'[Math.floor(Math.random() * 26)]; // 小写
    password += 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'[Math.floor(Math.random() * 26)]; // 大写
    password += '0123456789'[Math.floor(Math.random() * 10)]; // 数字
    password += '!@#$%^&*'[Math.floor(Math.random() * 8)]; // 特殊字符

    // 填充剩余长度
    for (let i = password.length; i < length; i++) {
        password += charset[Math.floor(Math.random() * charset.length)];
    }

    // 打乱顺序
    return password.split('').sort(() => Math.random() - 0.5).join('');
}

/**
 * 初始化配置面板
 * @param {Object} dependencies - 依赖注入 { state, api, toast, dialog }
 */
export function initSettingsTab({ state, api, toast, dialog }) {
    console.log('⚙️ 初始化设置面板...');

    // 保存依赖
    deps = { state, api, toast, dialog };

    // 绑定事件
    bindEvents();

    // 初始化实时验证
    settingsValidator.initRealtimeValidation();

    // 监听配置加载事件
    globalEvents.on('config:loaded', (data) => {
        loadConfigToForm(data);
    });

    // 监听标签页切换
    globalEvents.on('tab:switch:settings', () => {
        loadConfig();
    });

    /**
     * 绑定事件监听器
     */
    function bindEvents() {
        // Tab 切换
        const tabs = document.querySelectorAll('.settings-tab');
        tabs.forEach(tab => {
            tab.addEventListener('click', () => {
                const tabName = tab.dataset.tab;
                switchSettingsTab(tabName);
            });
        });

        // 保存设置
        const saveBtn = document.getElementById('save-settings');
        saveBtn?.addEventListener('click', saveSettings);

        // 恢复默认
        const resetBtn = document.getElementById('reset-settings');
        resetBtn?.addEventListener('click', () => {
            deps.dialog.confirm('确定要恢复默认设置吗？<br><small>此操作将所有设置重置为默认值</small>', () => {
                resetToDefault();
            });
        });

        // 密码显示/隐藏切换
        const togglePasswordBtn = document.getElementById('toggle-password');
        const passwordInput = document.getElementById('admin-password');
        let passwordVisible = false;

        // 更新密码显示状态
        function updatePasswordVisibility(visible) {
            passwordVisible = visible;
            passwordInput.type = visible ? 'text' : 'password';
            // 更新图标：显示时用睁眼，隐藏时用闭眼
            togglePasswordBtn.innerHTML = visible
                ? '<i class="icon icon-eye"></i>'
                : '<i class="icon icon-eye-off"></i>';
            togglePasswordBtn.title = visible ? '隐藏密码' : '显示密码';
        }

        togglePasswordBtn?.addEventListener('click', () => {
            updatePasswordVisibility(!passwordVisible);
        });

        // 随机密码生成
        const generatePasswordBtn = document.getElementById('generate-password');
        generatePasswordBtn?.addEventListener('click', () => {
            const password = generateRandomPassword(16);
            passwordInput.value = password;
            passwordInput.type = 'text'; // 显示生成的密码
            updatePasswordVisibility(true);
            hasUnsavedChanges = true;
            deps.toast.success('已生成随机密码');
        });

        // 监听表单变化
        const formInputs = document.querySelectorAll('.settings-content input, .settings-content select, .settings-content textarea');
        formInputs.forEach(input => {
            input.addEventListener('input', () => {
                hasUnsavedChanges = true;
            });
            input.addEventListener('change', () => {
                hasUnsavedChanges = true;
            });
        });

        // 服务器时间同步按钮
        const syncTimeBtn = document.getElementById('sync-time-btn');
        syncTimeBtn?.addEventListener('click', syncServerTime);

        // 初始化服务器时间显示
        updateServerTimeDisplay();
        // 每秒更新时间显示
        setInterval(updateServerTimeDisplay, 1000);
    }

    /**
     * 更新服务器时间显示
     */
    function updateServerTimeDisplay() {
        const timeElement = document.getElementById('server-time');
        if (!timeElement) return;

        try {
            const frontendSettings = JSON.parse(localStorage.getItem('pixelbeast_frontend_settings') || '{}');
            const timezone = frontendSettings.serverTimezone || 'Asia/Shanghai';

            // 使用 Intl.DateTimeFormat 格式化时间
            const now = new Date();
            const formatter = new Intl.DateTimeFormat('zh-CN', {
                timeZone: timezone,
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit',
                hour12: false
            });

            timeElement.textContent = formatter.format(now);
        } catch (e) {
            // 如果时区无效，使用本地时间
            timeElement.textContent = now.toLocaleString('zh-CN');
        }
    }

    /**
     * 同步服务器时间（功能暂未启用，等待后端实现）
     */
    async function syncServerTime() {
        if (!deps) return;
        const { toast } = deps;

        // 按钮已禁用，此函数不会被调用
        // 等待后端实现 NTP 时间同步后再启用
        toast.info('时间同步功能需要后端支持，暂未启用');
    }

    /**
     * 切换设置 Tab
     */
    function switchSettingsTab(tabName) {
        // 如果有未保存的修改，提示用户
        if (hasUnsavedChanges && tabName !== currentTab) {
            // 使用 dialog 确认
            deps.dialog.show({
                title: '未保存的修改',
                message: '您有未保存的修改，是否放弃更改？',
                type: 'warning',
                confirmText: '继续切换',
                cancelText: '留在当前页',
                onConfirm: () => {
                    // 用户确认切换，恢复表单值并切换
                    restoreFormValues();
                    hasUnsavedChanges = false;
                    performTabSwitch(tabName);
                },
                onCancel: () => {
                    // 用户取消，留在当前页，保留修改
                    // 不做任何操作
                }
            });
            return;
        }

        // 没有未保存修改，直接切换
        performTabSwitch(tabName);
    }

    /**
     * 执行标签页切换
     */
    function performTabSwitch(tabName) {

        // 更新当前 Tab
        currentTab = tabName;

        // 更新按钮状态
        document.querySelectorAll('.settings-tab').forEach(tab => {
            if (tab.dataset.tab === tabName) {
                tab.classList.add('active');
            } else {
                tab.classList.remove('active');
            }
        });

        // 更新内容显示
        document.querySelectorAll('.settings-content').forEach(content => {
            content.classList.remove('active');
            if (content.id === `settings-${tabName}`) {
                content.classList.add('active');
            }
        });
    }

    /**
     * 加载配置
     */
    async function loadConfig() {
        if (!deps) return;
        const { api } = deps;

        try {
            const response = await api.get('/api/config');
            if (response && response.ok) {
                const data = await api.parseJSON(response);
                currentConfig = data;
                loadConfigToForm(data);
            }
        } catch (error) {
            console.error('加载配置失败:', error);
        }
    }

    /**
     * 加载配置到表单
     */
    function loadConfigToForm(config) {
        if (!config) return;

        // 面板设置 - 从 global.admin_port 获取端口
        setFieldValue('admin-username', config.admin?.username || 'admin');
        setFieldValue('admin-port', config.global?.admin_port || config.admin?.port || 9527);
        setFieldValue('admin-path', config.admin?.path || '/admin');
        setFieldValue('admin-domain', config.admin?.domain || '');
        setCheckboxValue('admin-ssl', config.admin?.ssl || false);
        setFieldValue('server-timezone', config.server?.timezone || 'Asia/Shanghai');

        // 目录设置
        setFieldValue('ftp-root-dir', config.ftp?.root || './ftp');

        // 从 localStorage 读取前端设置
        try {
            const frontendSettings = JSON.parse(localStorage.getItem('pixelbeast_frontend_settings') || '{}');
            setFieldValue('web-root-dir', frontendSettings.webRootDir || config.http?.root || './web');
            setFieldValue('backup-dir', frontendSettings.backupDir || './backups');

            // 安全设置
            const securitySettings = JSON.parse(localStorage.getItem('pixelbeast_security_settings') || '{}');
            // IP 白名单
            const allowIpsTextarea = document.getElementById('admin-allow-ips');
            if (allowIpsTextarea && securitySettings.allowIps) {
                allowIpsTextarea.value = securitySettings.allowIps.join('\n');
            }
            // 会话超时
            setFieldValue('session-timeout', securitySettings.sessionTimeout || 86400);
            // 2FA
            setCheckboxValue('admin-2fa', securitySettings.enable2FA || false);
            // 记录操作日志
            setCheckboxValue('admin-log-operations', securitySettings.logOperations !== false);
            // 敏感操作确认
            setCheckboxValue('admin-confirm-dangerous', securitySettings.confirmDangerous !== false);
        } catch (e) {
            console.log('无法读取前端设置:', e);
        }

        // 重置未保存标记
        hasUnsavedChanges = false;

        // 保存表单值快照（用于取消时恢复）
        saveFormSnapshot();
    }

    /**
     * 保存当前表单值的快照
     */
    function saveFormSnapshot() {
        originalFormValues = {};

        // 保存所有输入框的值
        const inputs = document.querySelectorAll('.settings-content input, .settings-content select, .settings-content textarea');
        inputs.forEach(input => {
            if (input.id) {
                if (input.type === 'checkbox') {
                    originalFormValues[input.id] = input.checked;
                } else {
                    originalFormValues[input.id] = input.value;
                }
            }
        });
    }

    /**
     * 恢复表单值到快照状态
     */
    function restoreFormValues() {
        if (!originalFormValues || Object.keys(originalFormValues).length === 0) {
            return;
        }

        // 恢复所有输入框的值
        Object.keys(originalFormValues).forEach(id => {
            const el = document.getElementById(id);
            if (el) {
                if (el.type === 'checkbox') {
                    el.checked = originalFormValues[id];
                } else {
                    el.value = originalFormValues[id];
                }
            }
        });
    }

    /**
     * 保存设置
     */
    async function saveSettings() {
        if (!deps) return;
        const { api, toast } = deps;

        // 验证表单
        const validation = settingsValidator.validateAll();
        if (!validation.valid) {
            toast.error(validation.message);
            return;
        }

        try {
            const config = getFormConfig();

            const response = await api.post('/api/config/save', config);
            if (response && response.ok) {
                const data = await api.parseJSON(response);
                toast.success(data.message || '设置保存成功，请重启程序生效');
                // 保存成功后重置未保存标记
                hasUnsavedChanges = false;
            } else {
                toast.error('设置保存失败');
            }
        } catch (error) {
            console.error('保存设置失败:', error);
            toast.error('保存设置失败: ' + error.message);
        }
    }

    /**
     * 从表单获取配置
     */
    function getFormConfig() {
        // 获取当前完整配置，优先使用模块级变量，然后从 state 获取
        const storedConfig = currentConfig || deps?.state?.get('config') || {};

        // 面板设置
        const adminUsername = getFieldValue('admin-username') || 'admin';
        const adminPassword = getFieldValue('admin-password');
        const adminPort = parseInt(getFieldValue('admin-port')) || 9527;
        const adminPath = getFieldValue('admin-path') || '/admin';
        const adminDomain = getFieldValue('admin-domain') || '';
        const adminSsl = getCheckboxValue('admin-ssl');
        const serverTimezone = getFieldValue('server-timezone') || 'Asia/Shanghai';

        // 目录设置
        const webRootDir = getFieldValue('web-root-dir') || './web';
        const ftpRootDir = getFieldValue('ftp-root-dir') || './ftp';
        const backupDir = getFieldValue('backup-dir') || './backups';

        // 安全设置（暂存到 localStorage，不发送到后端）
        const allowIpsText = getFieldValue('admin-allow-ips') || '';
        const allowIps = allowIpsText.split('\n')
            .map(ip => ip.trim())
            .filter(ip => ip.length > 0);

        const sessionTimeout = parseInt(getFieldValue('session-timeout')) || 86400;
        const enable2FA = getCheckboxValue('admin-2fa');
        const logOperations = getCheckboxValue('admin-log-operations');
        const confirmDangerous = getCheckboxValue('admin-confirm-dangerous');

        // 保存前端设置到 localStorage
        localStorage.setItem('pixelbeast_frontend_settings', JSON.stringify({
            webRootDir,
            backupDir,
            serverTimezone
        }));
        localStorage.setItem('pixelbeast_security_settings', JSON.stringify({
            allowIps,
            sessionTimeout,
            enable2FA,
            logOperations,
            confirmDangerous
        }));

        // 构建配置对象，只包含后端支持的字段
        const newConfig = {
            // 保留现有站点配置
            sites: storedConfig.sites || [],
            // 保留现有 HTTP 配置（向后兼容）
            http: storedConfig.http || { enabled: true, port: 8080, root: './web', domain: '' },
            // FTP 配置
            ftp: {
                enabled: storedConfig.ftp?.enabled ?? true,
                port: storedConfig.ftp?.port || 2121,
                root: ftpRootDir,
                users: storedConfig.ftp?.users || [{ username: 'flash', password: 'flash' }]
            },
            // Admin 配置
            admin: {
                enabled: true,
                username: adminUsername,
                password: adminPassword || storedConfig.admin?.password || 'admin123',
                path: adminPath,
                domain: adminDomain,
                ssl: adminSsl
            },
            // Global 配置
            global: {
                admin_port: adminPort,
                data_dir: storedConfig.global?.data_dir || './data'
            },
            // Server 配置
            server: {
                timezone: serverTimezone
            }
        };

        return newConfig;
    }

    /**
     * 恢复默认设置
     */
    function resetToDefault() {
        if (!deps) return;
        const { toast } = deps;

        loadConfigToForm(DEFAULT_CONFIG);
        toast.info('已恢复默认设置，请点击保存按钮保存');
    }

    /**
     * 辅助函数：设置输入框值
     */
    function setFieldValue(id, value) {
        const el = document.getElementById(id);
        if (el) el.value = value;
    }

    /**
     * 辅助函数：获取输入框值
     */
    function getFieldValue(id) {
        const el = document.getElementById(id);
        return el ? el.value : '';
    }

    /**
     * 辅助函数：设置复选框值
     */
    function setCheckboxValue(id, checked) {
        const el = document.getElementById(id);
        if (el) el.checked = checked;
    }

    /**
     * 辅助函数：获取复选框值
     */
    function getCheckboxValue(id) {
        const el = document.getElementById(id);
        return el ? el.checked : false;
    }

    // 初始加载
    loadConfig();
}
