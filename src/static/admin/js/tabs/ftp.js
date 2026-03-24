/**
 * FTP 用户管理模块
 *
 * 使用 DataTable 组件实现用户列表、分页、搜索、批量操作
 */

import { globalEvents } from '../core/events.js';
import { DataTable } from '../components/data-table.js';

// 存储依赖以便在闭包函数中使用
let deps = null;

// DataTable 实例
let dataTable = null;

// 状态
let users = [];
let editingUser = null;

// 列配置
const columns = [
    {
        title: '用户名',
        dataIndex: 'username',
        className: 'col-username',
        render: (value) => `
            <div class="ftp-user-info">
                <div class="ftp-user-avatar">${value?.charAt(0).toUpperCase() || '?'}</div>
                <span class="ftp-user-name">${escapeHtml(value)}</span>
            </div>
        `
    },
    {
        title: '密码',
        dataIndex: 'password',
        className: 'col-password',
        render: (value, row) => `
            <div class="ftp-password">
                <span class="ftp-password-text">••••••••</span>
                <button class="ftp-password-copy" data-password="${escapeHtml(value)}" title="复制密码">
                    <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                        <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                    </svg>
                </button>
            </div>
        `
    },
    {
        title: '状态',
        dataIndex: 'status',
        className: 'col-status',
        render: (value, row) => `
            <div class="ftp-status">
                <label class="switch">
                    <input type="checkbox" class="ftp-status-toggle" data-username="${escapeHtml(row.username)}"
                        ${value === 'enabled' ? 'checked' : ''}>
                    <span class="slider"></span>
                </label>
                <span class="ftp-status-text ${value}">${value === 'enabled' ? '已启用' : '已禁用'}</span>
            </div>
        `
    },
    {
        title: '快速链接',
        dataIndex: 'username',
        className: 'col-link',
        render: (value) => {
            const link = `ftp://${value}@${window.location.hostname}`;
            return `
                <div class="ftp-quick-link">
                    <span class="ftp-link-text" title="${escapeHtml(link)}">${escapeHtml(link)}</span>
                    <button class="ftp-link-copy" data-link="${escapeHtml(link)}" title="复制链接">
                        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                        </svg>
                    </button>
                </div>
            `;
        }
    },
    {
        title: '根目录',
        dataIndex: 'rootPath',
        className: 'col-root',
        render: (value) => `
            <div class="ftp-root-path">
                <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
                </svg>
                <span>${escapeHtml(value || '/')}</span>
            </div>
        `
    },
    {
        title: '密码有效期',
        dataIndex: 'expiryDate',
        className: 'col-expiry',
        render: (value) => renderExpiry(value)
    },
    {
        title: '容量',
        dataIndex: 'usedSpace',
        className: 'col-quota',
        render: (value, row) => renderQuota(value, row.quota)
    },
    {
        title: '备注',
        dataIndex: 'remark',
        className: 'col-remark',
        render: (value) => `<span class="ftp-remark" title="${escapeHtml(value || '')}">${escapeHtml(value || '-')}</span>`
    },
    {
        title: '操作',
        dataIndex: 'username',
        className: 'col-actions',
        render: (value) => `
            <div class="ftp-actions">
                <button class="ftp-action-btn edit" data-username="${escapeHtml(value)}" title="编辑">
                    <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                    </svg>
                </button>
                <button class="ftp-action-btn delete" data-username="${escapeHtml(value)}" title="删除">
                    <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="3 6 5 6 21 6"></polyline>
                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                    </svg>
                </button>
            </div>
        `
    }
];

/**
 * 初始化 FTP 管理面板
 * @param {Object} dependencies - 依赖注入 { state, api, toast, message }
 */
export function initFtpTab({ state, api, toast, message }) {
    console.log('初始化 FTP 用户管理面板...');

    deps = { state, api, toast, message };

    // 初始化 DataTable
    initDataTable();

    // 绑定事件
    bindEvents();

    // 监听标签页切换
    globalEvents.match('tab:switch:ftp', () => {
        loadUsers();
    });
}

/**
 * 初始化 DataTable
 */
function initDataTable() {
    const container = document.getElementById('ftp-table-container');
    if (!container) return;

    dataTable = new DataTable({
        container,
        columns,
        selectable: true,
        pageSize: 20,
        emptyText: '暂无 FTP 用户',
        emptyHint: '点击上方"添加用户"按钮创建',
        loadingText: '加载中...',
        onSelectionChange: ({ selectedCount }) => {
            updateBatchActions(selectedCount);
        },
        onPageChange: (page) => {
            // 页面变化后重新绑定行内事件
            setTimeout(bindRowEvents, 0);
        }
    });
}

/**
 * 绑定事件
 */
function bindEvents() {
    if (!deps) return;

    // 添加用户
    document.getElementById('ftp-add-user-btn')?.addEventListener('click', () => {
        editingUser = null;
        showUserModal();
    });

    // 端口设置
    document.getElementById('ftp-port-btn')?.addEventListener('click', showPortModal);

    // 刷新
    document.getElementById('ftp-refresh-btn')?.addEventListener('click', loadUsers);

    // 搜索
    let searchTimer;
    document.getElementById('ftp-search')?.addEventListener('input', () => {
        clearTimeout(searchTimer);
        searchTimer = setTimeout(() => {
            filterUsers();
        }, 300);
    });

    // 状态筛选
    document.getElementById('ftp-status-filter')?.addEventListener('change', filterUsers);

    // 批量操作
    document.getElementById('ftp-batch-enable')?.addEventListener('click', () => batchEnable(true));
    document.getElementById('ftp-batch-disable')?.addEventListener('click', () => batchEnable(false));
    document.getElementById('ftp-batch-delete')?.addEventListener('click', batchDelete);

    // 用户弹窗
    document.getElementById('ftp-modal-cancel')?.addEventListener('click', hideUserModal);
    document.getElementById('ftp-modal-close')?.addEventListener('click', hideUserModal);
    document.getElementById('ftp-modal-confirm')?.addEventListener('click', saveUser);
    document.getElementById('ftp-generate-password')?.addEventListener('click', generateRandomPassword);
    document.getElementById('ftp-user-modal')?.querySelector('.modal-overlay')?.addEventListener('click', hideUserModal);

    // 端口弹窗
    document.getElementById('ftp-port-cancel')?.addEventListener('click', hidePortModal);
    document.getElementById('ftp-port-modal-close')?.addEventListener('click', hidePortModal);
    document.getElementById('ftp-port-confirm')?.addEventListener('click', savePort);
    document.getElementById('ftp-port-modal')?.querySelector('.modal-overlay')?.addEventListener('click', hidePortModal);

    // 数字输入控件
    initNumberInputs();
}

/**
 * 加载用户列表
 */
async function loadUsers() {
    if (!deps || !dataTable) return;
    const { api, toast } = deps;

    dataTable.setLoading(true);

    try {
        const data = await api.getJSON('/api/ftp/users');
        users = data || [];
        filterUsers();
    } catch (error) {
        console.error('加载 FTP 用户失败:', error);
        toast.error('加载失败: ' + error.message);
        dataTable.updateData([]);
    } finally {
        dataTable.setLoading(false);
    }
}

/**
 * 筛选用户
 */
function filterUsers() {
    if (!dataTable) return;

    const search = document.getElementById('ftp-search')?.value?.toLowerCase() || '';
    const statusFilter = document.getElementById('ftp-status-filter')?.value || '';

    const filteredUsers = users.filter(user => {
        // 搜索过滤
        if (search && !user.username?.toLowerCase().includes(search) &&
            !user.remark?.toLowerCase().includes(search)) {
            return false;
        }
        // 状态过滤
        if (statusFilter && user.status !== statusFilter) {
            return false;
        }
        return true;
    });

    dataTable.updateData(filteredUsers);

    // 数据更新后重新绑定行内事件
    setTimeout(bindRowEvents, 0);
}

/**
 * 绑定行内事件
 */
function bindRowEvents() {
    if (!deps) return;
    const { message } = deps;

    // 状态开关
    document.querySelectorAll('.ftp-status-toggle').forEach(toggle => {
        toggle.addEventListener('change', (e) => {
            const username = e.target.dataset.username;
            const enabled = e.target.checked;
            updateUserStatus(username, enabled);
        });
    });

    // 复制密码
    document.querySelectorAll('.ftp-password-copy').forEach(btn => {
        btn.addEventListener('click', () => {
            const password = btn.dataset.password;
            copyToClipboard(password);
            message.success('密码已复制');
        });
    });

    // 复制链接
    document.querySelectorAll('.ftp-link-copy').forEach(btn => {
        btn.addEventListener('click', () => {
            const link = btn.dataset.link;
            copyToClipboard(link);
            message.success('链接已复制');
        });
    });

    // 编辑
    document.querySelectorAll('.ftp-action-btn.edit').forEach(btn => {
        btn.addEventListener('click', () => {
            const username = btn.dataset.username;
            editingUser = users.find(u => u.username === username);
            showUserModal();
        });
    });

    // 删除
    document.querySelectorAll('.ftp-action-btn.delete').forEach(btn => {
        btn.addEventListener('click', () => {
            const username = btn.dataset.username;
            showConfirmDialog(`确定要删除用户 "${username}" 吗？`, () => {
                deleteUser(username);
            });
        });
    });
}

/**
 * 更新批量操作显示
 */
function updateBatchActions(selectedCount) {
    const batchEl = document.getElementById('ftp-batch-actions');
    const countEl = document.getElementById('ftp-selected-count');

    if (batchEl && countEl) {
        if (selectedCount > 0) {
            batchEl.style.display = 'flex';
            countEl.textContent = selectedCount;
        } else {
            batchEl.style.display = 'none';
        }
    }
}

/**
 * 批量启用/禁用
 */
async function batchEnable(enabled) {
    if (!deps || !dataTable) return;
    const { api, message } = deps;

    const selectedKeys = dataTable.getSelectedKeys();
    if (selectedKeys.length === 0) return;

    try {
        await api.post('/api/ftp/users/batch', {
            action: enabled ? 'enable' : 'disable',
            usernames: selectedKeys
        });
        message.success(`已${enabled ? '启用' : '禁用'} ${selectedKeys.length} 个用户`);
        dataTable.clearSelection();
        loadUsers();
    } catch (error) {
        message.error('操作失败: ' + error.message);
    }
}

/**
 * 批量删除
 */
async function batchDelete() {
    if (!deps || !dataTable) return;
    const { api, message } = deps;

    const selectedKeys = dataTable.getSelectedKeys();
    if (selectedKeys.length === 0) return;

    showConfirmDialog(`确定要删除选中的 ${selectedKeys.length} 个用户吗？`, async () => {
        try {
            await api.post('/api/ftp/users/batch', {
                action: 'delete',
                usernames: selectedKeys
            });
            message.success(`已删除 ${selectedKeys.length} 个用户`);
            dataTable.clearSelection();
            loadUsers();
        } catch (error) {
            message.error('删除失败: ' + error.message);
        }
    });
}

/**
 * 更新用户状态
 */
async function updateUserStatus(username, enabled) {
    if (!deps) return;
    const { api, toast } = deps;

    try {
        await api.post(`/api/ftp/users/${username}/status`, { enabled });
        toast.success(`用户已${enabled ? '启用' : '禁用'}`);
        const user = users.find(u => u.username === username);
        if (user) {
            user.status = enabled ? 'enabled' : 'disabled';
            filterUsers();
        }
    } catch (error) {
        toast.error('操作失败: ' + error.message);
        filterUsers();
    }
}

/**
 * 删除用户
 */
async function deleteUser(username) {
    if (!deps) return;
    const { api, message } = deps;

    try {
        await api.post(`/api/ftp/users/${username}/delete`);
        message.success('用户已删除');
        loadUsers();
    } catch (error) {
        message.error('删除失败: ' + error.message);
    }
}

/**
 * 渲染有效期
 */
function renderExpiry(expiryDate) {
    if (!expiryDate) {
        return '<div class="ftp-expiry"><span class="ftp-expiry-date">-</span></div>';
    }

    const expiry = new Date(expiryDate);
    const now = new Date();
    const days = Math.ceil((expiry - now) / (1000 * 60 * 60 * 24));

    let remainingClass = '';
    let remainingText = '';

    if (days < 0) {
        remainingClass = 'danger';
        remainingText = '已过期';
    } else if (days <= 7) {
        remainingClass = 'danger';
        remainingText = `剩余 ${days} 天`;
    } else if (days <= 30) {
        remainingClass = 'warning';
        remainingText = `剩余 ${days} 天`;
    } else {
        remainingText = `剩余 ${days} 天`;
    }

    return `
        <div class="ftp-expiry">
            <span class="ftp-expiry-date">${expiry.toLocaleDateString()}</span>
            <span class="ftp-expiry-remaining ${remainingClass}">${remainingText}</span>
        </div>
    `;
}

/**
 * 渲染容量
 */
function renderQuota(used, quota) {
    const usedMB = (used || 0) / (1024 * 1024);
    const quotaMB = quota || 0;

    if (!quotaMB) {
        return `
            <div class="ftp-quota">
                <div class="ftp-quota-bar">
                    <div class="ftp-quota-fill normal" style="width: 0%"></div>
                </div>
                <span class="ftp-quota-text">${formatSize(used)} / 无限制</span>
            </div>
        `;
    }

    const percent = Math.min((usedMB / quotaMB) * 100, 100);
    let fillClass = 'normal';
    if (percent >= 90) fillClass = 'danger';
    else if (percent >= 70) fillClass = 'warning';

    return `
        <div class="ftp-quota">
            <div class="ftp-quota-bar">
                <div class="ftp-quota-fill ${fillClass}" style="width: ${percent}%"></div>
            </div>
            <span class="ftp-quota-text">${formatSize(used)} / ${formatSize(quota * 1024 * 1024)}</span>
        </div>
    `;
}

/**
 * 格式化文件大小
 */
function formatSize(bytes) {
    if (!bytes) return '0 MB';
    const mb = bytes / (1024 * 1024);
    if (mb < 1024) return `${mb.toFixed(1)} MB`;
    return `${(mb / 1024).toFixed(2)} GB`;
}

// ========== 弹窗相关 ==========

/**
 * 显示用户弹窗
 */
function showUserModal() {
    const modal = document.getElementById('ftp-user-modal');
    const title = document.getElementById('ftp-modal-title');
    const form = document.getElementById('ftp-user-form');

    if (!modal) return;

    if (editingUser) {
        title.textContent = '编辑 FTP 用户';
        document.getElementById('ftp-form-username').value = editingUser.username;
        document.getElementById('ftp-form-username').disabled = true;
        document.getElementById('ftp-form-password').value = '';
        document.getElementById('ftp-form-password').placeholder = '留空则不修改';
        document.getElementById('ftp-form-root').value = editingUser.rootPath || '/';
        document.getElementById('ftp-form-quota').value = editingUser.quota || '';
        document.getElementById('ftp-form-expiry').value = editingUser.expiryDays || '';
        document.getElementById('ftp-form-status').value = editingUser.status || 'enabled';
        document.getElementById('ftp-form-remark').value = editingUser.remark || '';
    } else {
        title.textContent = '添加 FTP 用户';
        form?.reset();
        const usernameInput = document.getElementById('ftp-form-username');
        if (usernameInput) usernameInput.disabled = false;
        const passwordInput = document.getElementById('ftp-form-password');
        if (passwordInput) passwordInput.placeholder = '请输入密码';
    }

    modal.classList.add('active');

    // 初始化数字输入控件
    setTimeout(initNumberInputs, 0);
}

/**
 * 隐藏用户弹窗
 */
function hideUserModal() {
    const modal = document.getElementById('ftp-user-modal');
    if (modal) modal.classList.remove('active');
    editingUser = null;
}

/**
 * 保存用户
 */
async function saveUser() {
    if (!deps) return;
    const { api, toast, message } = deps;

    const username = document.getElementById('ftp-form-username')?.value.trim();
    const password = document.getElementById('ftp-form-password')?.value;
    const rootPath = document.getElementById('ftp-form-root')?.value.trim();
    const quota = document.getElementById('ftp-form-quota')?.value;
    const expiryDays = document.getElementById('ftp-form-expiry')?.value;
    const status = document.getElementById('ftp-form-status')?.value;
    const remark = document.getElementById('ftp-form-remark')?.value.trim();

    if (!username) {
        message.error('请输入用户名');
        return;
    }
    if (!editingUser && !password) {
        message.error('请输入密码');
        return;
    }
    if (!rootPath) {
        message.error('请输入根目录');
        return;
    }

    const data = {
        username,
        rootPath,
        status,
        remark,
        quota: quota ? parseInt(quota) : 0,
        expiryDays: expiryDays ? parseInt(expiryDays) : 0
    };

    if (password) {
        data.password = password;
    }

    try {
        if (editingUser) {
            await api.post(`/api/ftp/users/${username}`, data);
            message.success('用户已更新');
        } else {
            await api.post('/api/ftp/users', data);
            message.success('用户已创建');
        }
        hideUserModal();
        loadUsers();
    } catch (error) {
        message.error('保存失败: ' + error.message);
    }
}

/**
 * 生成随机密码
 */
function generateRandomPassword() {
    const length = 12;
    const charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
    let password = '';
    for (let i = 0; i < length; i++) {
        password += charset.charAt(Math.floor(Math.random() * charset.length));
    }
    const passwordInput = document.getElementById('ftp-form-password');
    if (passwordInput) {
        passwordInput.value = password;
        passwordInput.type = 'text';
        setTimeout(() => {
            passwordInput.type = 'password';
        }, 2000);
    }
}

/**
 * 显示端口弹窗
 */
async function showPortModal() {
    if (!deps) return;
    const { api, toast } = deps;

    try {
        const config = await api.getJSON('/api/ftp/config');
        const portInput = document.getElementById('ftp-port-input');
        if (portInput) portInput.value = config.port || 21;
        const modal = document.getElementById('ftp-port-modal');
        if (modal) {
            modal.classList.add('active');
            // 初始化数字输入控件
            setTimeout(initNumberInputs, 0);
        }
    } catch (error) {
        toast.error('获取配置失败: ' + error.message);
    }
}

/**
 * 隐藏端口弹窗
 */
function hidePortModal() {
    const modal = document.getElementById('ftp-port-modal');
    if (modal) modal.classList.remove('active');
}

/**
 * 保存端口设置
 */
async function savePort() {
    if (!deps) return;
    const { api, toast, message } = deps;

    const portInput = document.getElementById('ftp-port-input');
    const port = parseInt(portInput?.value);

    if (!port || port < 1 || port > 65535) {
        message.error('请输入有效的端口号 (1-65535)');
        return;
    }

    try {
        await api.post('/api/ftp/config', { port });
        message.success('端口设置已保存，重启服务后生效');
        hidePortModal();
    } catch (error) {
        message.error('保存失败: ' + error.message);
    }
}

/**
 * 显示确认对话框
 */
function showConfirmDialog(message, onConfirm) {
    if (confirm(message)) {
        onConfirm();
    }
}

/**
 * 复制到剪贴板
 */
function copyToClipboard(text) {
    if (navigator.clipboard) {
        navigator.clipboard.writeText(text);
    } else {
        const input = document.createElement('input');
        input.value = text;
        document.body.appendChild(input);
        input.select();
        document.execCommand('copy');
        document.body.removeChild(input);
    }
}

/**
 * HTML 转义
 */
function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

/**
 * 初始化数字输入控件
 */
function initNumberInputs() {
    document.querySelectorAll('.number-input-wrapper').forEach(wrapper => {
        const input = wrapper.querySelector('input[type="number"]');
        const upBtn = wrapper.querySelector('[data-action="up"]');
        const downBtn = wrapper.querySelector('[data-action="down"]');

        if (!input) return;

        const step = parseInt(input.dataset.step) || 1;
        const min = input.min !== '' ? parseInt(input.min) : null;
        const max = input.max !== '' ? parseInt(input.max) : null;

        const updateValue = (delta) => {
            let value = parseInt(input.value) || 0;
            value += delta;

            if (min !== null && value < min) value = min;
            if (max !== null && value > max) value = max;

            input.value = value;
            input.dispatchEvent(new Event('input', { bubbles: true }));
        };

        upBtn?.addEventListener('click', () => updateValue(step));
        downBtn?.addEventListener('click', () => updateValue(-step));

        // 键盘支持
        input.addEventListener('keydown', (e) => {
            if (e.key === 'ArrowUp') {
                e.preventDefault();
                updateValue(step);
            } else if (e.key === 'ArrowDown') {
                e.preventDefault();
                updateValue(-step);
            }
        });
    });
}
