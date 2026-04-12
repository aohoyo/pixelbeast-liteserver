/**
 * FTP 用户管理模块
 *
 * 使用 DataTable 组件实现用户列表、分页、搜索、批量操作
 */

import { BaseTab } from './BaseTab.js';
import { DataTable } from '../components/data-table.js';
import { escapeHtml, formatDate, copyToClipboard, initNumberInputs, openDirPicker } from '../core/utils.js';

class FtpTab extends BaseTab {
    constructor(deps) {
        super(deps, 'ftp');
        this.dataTable = null;
        this.users = [];
        this.editingUser = null;
        this.ftpPort = 2121;
    }

    onInit() {
        this._svc = this.createServiceControls({
            apiPrefix: '/api/service/ftp',
            statusId: 'ftp-service-status',
            toggleId: 'ftp-service-toggle',
            label: 'FTP 服务',
        });
        this.initDataTable();
        initNumberInputs(this.container || document);
        this.bindEvents();
        this.checkServiceStatus();
    }

    // ========== DataTable ==========

    initDataTable() {
        const container = this.$('#ftp-table-container');
        if (!container) return;

        this.dataTable = new DataTable({
            container,
            columns: this.getColumns(),
            selectable: true,
            pageSize: 20,
            emptyText: '暂无 FTP 用户',
            emptyHint: '点击上方"添加用户"按钮创建',
            batchActions: [
                {
                    key: 'enable',
                    label: '启用',
                    icon: '<svg viewBox="0 0 24 24"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>',
                    type: 'success',
                    handler: () => this.batchToggle(true)
                },
                {
                    key: 'disable',
                    label: '禁用',
                    icon: '<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10"></circle><line x1="4.93" y1="4.93" x2="19.07" y2="19.07"></line></svg>',
                    handler: () => this.batchToggle(false)
                },
                {
                    key: 'delete',
                    label: '删除',
                    icon: '<svg viewBox="0 0 24 24"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>',
                    type: 'danger',
                    handler: () => this.batchDelete()
                }
            ],
            onPageChange: () => setTimeout(() => this.bindRowEvents(), 0)
        });
    }

    getColumns() {
        return [
            {
                title: '用户名',
                dataIndex: 'username',
                className: 'col-username',
                render: (value) => `<div class="ftp-user-info"><div class="ftp-user-avatar">${value?.charAt(0).toUpperCase() || '?'}</div><span class="ftp-user-name">${escapeHtml(value)}</span></div>`
            },
            {
                title: '密码',
                dataIndex: 'password',
                className: 'col-password',
                render: (value) => `<div class="ftp-password"><span class="ftp-password-text">••••••••</span><button class="copy-btn" data-password="${escapeHtml(value)}" title="复制密码"><svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg></button></div>`
            },
            {
                title: '状态',
                dataIndex: 'status',
                className: 'col-status',
                render: (value, row) => `<button class="status-btn ${value === 'enabled' ? 'active' : ''}" data-username="${escapeHtml(row.username)}">${value === 'enabled' ? '运行中' : '已停止'}</button>`
            },
            {
                title: '快速链接',
                dataIndex: 'username',
                className: 'col-link',
                render: (value, row) => {
                    const password = row.password || '';
                    const port = this.ftpPort || 2121;
                    const link = `ftp://${value}:${password}@${window.location.hostname}:${port}`;
                    return `<div class="quick-link"><a class="quick-link-text" href="${escapeHtml(link)}" target="_blank" title="点击打开">${escapeHtml(link)}</a><button class="quick-link-copy" data-link="${escapeHtml(link)}" title="复制链接"><svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg></button></div>`;
                }
            },
            {
                title: '根目录',
                dataIndex: 'rootPath',
                className: 'col-root',
                render: (value) => `<div class="root-path"><svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path></svg><a class="root-link" href="#" data-browse-path="${escapeHtml(value || '/')}" title="在文件管理中打开">${escapeHtml(value || '/')}</a></div>`
            },
            {
                title: '操作',
                dataIndex: 'username',
                className: 'col-actions',
                render: (value) => `<div class="actions"><button class="action-text config" data-username="${escapeHtml(value)}">配置</button><button class="action-text edit" data-username="${escapeHtml(value)}">编辑</button><button class="action-text delete" data-username="${escapeHtml(value)}">删除</button></div>`
            }
        ];
    }

    // ========== 事件绑定 ==========

    bindEvents() {
        // 添加、刷新、搜索
        // 服务控制
        this.$('#ftp-service-toggle')?.addEventListener('click', () => this.toggleService());
        this.$('#ftp-service-restart')?.addEventListener('click', () => this.restartService());
        this.$('#ftp-service-reload')?.addEventListener('click', () => this.reloadConfig());
        
        // 添加 FTP 用户
        this.$('#ftp-add-user-btn')?.addEventListener('click', () => { this.editingUser = null; this.showEditor(); });
        
        // 端口设置
        this.$('#ftp-port-btn')?.addEventListener('click', () => this.showPortModal());
        this.$('#ftp-refresh-btn')?.addEventListener('click', () => this.refresh());
        
        let searchTimer;
        this.$('#ftp-search')?.addEventListener('input', () => {
            clearTimeout(searchTimer);
            searchTimer = setTimeout(() => this.filterUsers(), 300);
        });

        // 批量操作（已改用 batch-bar 组件）
        
        // 弹窗
        this.$('#ftp-modal-cancel')?.addEventListener('click', () => this.hideEditor());
        this.$('#ftp-modal-close')?.addEventListener('click', () => this.hideEditor());
        this.$('#ftp-modal-confirm')?.addEventListener('click', () => this.saveUser());
        this.$('#ftp-user-modal')?.querySelector('.modal-overlay')?.addEventListener('click', () => this.hideEditor());

        // 目录浏览按钮
        this.$$('.directory-picker-btn').forEach(btn => {
            btn.addEventListener('click', () => openDirPicker(btn.dataset.dir, this.api));
        });

        // 生成随机密码
        this.$('#ftp-generate-password')?.addEventListener('click', () => {
            this.$('#ftp-form-password').value = this.generatePassword();
        });

        // 显示/隐藏密码
        this.$('#ftp-toggle-password')?.addEventListener('click', () => {
            const input = this.$('#ftp-form-password');
            const btn = this.$('#ftp-toggle-password');
            if (input && btn) {
                const isPassword = input.type === 'password';
                input.type = isPassword ? 'text' : 'password';
                // 切换图标
                btn.innerHTML = isPassword 
                    ? '<svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"></path><line x1="1" y1="1" x2="23" y2="23"></line></svg>'
                    : '<svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg>';
            }
        });

        // 端口设置弹窗
        this.$('#ftp-port-modal-close')?.addEventListener('click', () => this.hidePortModal());
        this.$('#ftp-port-cancel')?.addEventListener('click', () => this.hidePortModal());
        this.$('#ftp-port-confirm')?.addEventListener('click', () => this.savePort());
        this.$('#ftp-port-modal')?.querySelector('.modal-overlay')?.addEventListener('click', () => this.hidePortModal());

        // 配置弹窗
        this.$('#ftp-config-modal-close')?.addEventListener('click', () => this.hideConfigModal());
        this.$('#ftp-config-cancel')?.addEventListener('click', () => this.hideConfigModal());
        this.$('#ftp-config-confirm')?.addEventListener('click', () => this.saveConfig());
        this.$('#ftp-config-modal')?.querySelector('.modal-overlay')?.addEventListener('click', () => this.hideConfigModal());

        // 删除确认弹窗
        this.$('#ftp-delete-modal-close')?.addEventListener('click', () => this.hideDeleteConfirm());
        this.$('#ftp-delete-cancel')?.addEventListener('click', () => this.hideDeleteConfirm());
        this.$('#ftp-delete-confirm')?.addEventListener('click', () => this.confirmDelete());
        this.$('#ftp-delete-modal')?.querySelector('.modal-overlay')?.addEventListener('click', () => this.hideDeleteConfirm());
    }

    generatePassword() {
        const chars = 'ABCDEFGHJKMNPQRSTWXYZabcdefhijkmnprstwxyz2345678';
        let password = '';
        for (let i = 0; i < 12; i++) {
            password += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        return password;
    }

    bindRowEvents() {
        // 状态按钮
        this.$$('.status-btn[data-username]').forEach(btn => {
            btn.addEventListener('click', () => {
                const username = btn.dataset.username;
                const isActive = btn.classList.contains('active');
                this.toggleStatus(username, !isActive);
            });
        });

        // 复制密码
        this.$$('.copy-btn[data-password]').forEach(btn => {
            btn.addEventListener('click', () => {
                copyToClipboard(btn.dataset.password);
                this.message.success('密码已复制');
            });
        });

        // 复制链接
        this.$$('.quick-link-copy').forEach(btn => {
            btn.addEventListener('click', () => {
                copyToClipboard(btn.dataset.link);
                this.message.success('链接已复制');
            });
        });

        // 编辑
        this.$$('.action-text.edit').forEach(btn => {
            btn.addEventListener('click', () => {
                this.editingUser = this.users.find(u => u.username === btn.dataset.username);
                this.showEditor();
            });
        });

        // 配置
        this.$$('.action-text.config').forEach(btn => {
            btn.addEventListener('click', () => {
                this.editingUser = this.users.find(u => u.username === btn.dataset.username);
                this.showConfigModal();
            });
        });

        // 删除按钮
        this.$$('.action-text.delete').forEach(btn => {
            btn.addEventListener('click', () => {
                this.showDeleteConfirm(btn.dataset.username);
            });
        });

        // 根目录跳转文件管理
        this.$$('.root-link[data-browse-path]').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const path = link.dataset.browsePath;
                if (path && window.app?.switchTab) {
                    window.app.switchTab('files');
                    setTimeout(() => {
                        this.events.emit('files:navigate', path);
                    }, 150);
                }
            });
        });
    }

    // ========== 数据操作 ==========

    async onLoad() {
        this.dataTable?.setLoading(true);

        try {
            const result = await this.api.getJSON('/api/ftp/users');
            // 兼容两种格式：直接数组 或 {users: [...]}
            this.users = Array.isArray(result) ? result : (result?.users || []);
            this.filterUsers();
        } finally {
            this.dataTable?.setLoading(false);
        }
    }

    async onRefresh() {
        await this.onLoad();
    }

    filterUsers() {
        if (!this.dataTable) return;

        const search = this.$('#ftp-search')?.value?.toLowerCase() || '';
        const filtered = this.users.filter(u => !search || u.username?.toLowerCase().includes(search));

        this.dataTable.updateData(filtered);
        setTimeout(() => this.bindRowEvents(), 0);
    }

    // ========== 服务控制 ==========

    async checkServiceStatus() {
        try {
            const response = await this.api.get('/api/ftp/status');
            if (response?.ok) {
                const data = await this.api.parseJSON(response);
                if (data) {
                    this._svc.updateServiceStatus(data.running);
                    this.ftpPort = data.port || 2121;
                    const portInput = this.$('#ftp-port-input');
                    if (portInput) {
                        portInput.value = data.port;
                    }
                }
            }
        } catch (error) {
        }
    }

    toggleService() { return this._svc.toggleService(); }
    restartService() { return this._svc.restartService(); }
    reloadConfig() { return this._svc.reloadConfig(); }

    async toggleStatus(username, enabled) {
        try {
            await this.api.post('/api/ftp/users/toggle', { username, enabled });
            this.message.success(`用户已${enabled ? '启用' : '禁用'}`);
            const user = this.users.find(u => u.username === username);
            if (user) user.status = enabled ? 'enabled' : 'disabled';
            this.filterUsers();
        } catch (error) {
            this.message.error('操作失败: ' + error.message);
        }
    }

    showDeleteConfirm(username) {
        this.deleteTargetUsername = username;
        this.$('#ftp-delete-username').textContent = username;
        this.$('#ftp-delete-modal')?.classList.add('active');
    }

    hideDeleteConfirm() {
        this.$('#ftp-delete-modal')?.classList.remove('active');
        this.deleteTargetUsername = null;
    }

    async confirmDelete() {
        if (!this.deleteTargetUsername) return;
        
        const username = this.deleteTargetUsername;
        const deleteFiles = this.$('#ftp-delete-files')?.checked ?? true;
        this.hideDeleteConfirm();
        
        try {
            await this.api.post('/api/ftp/users/delete', { username, deleteFiles });
            this.message.success('用户已删除');
            // 清除缓存后刷新
            this.api.clearCache('/api/ftp/users');
            await this.refresh();
        } catch (error) {
            this.message.error('删除失败: ' + error.message);
        }
    }

    async deleteUser(username) {
        try {
            await this.api.post('/api/ftp/users/delete', { username });
            this.message.success('用户已删除');
            // 清除缓存后刷新
            this.api.clearCache('/api/ftp/users');
            await this.refresh();
        } catch (error) {
            this.message.error('删除失败: ' + error.message);
        }
    }

    // ========== 批量操作 ==========

    async batchToggle(enabled) {
        const keys = this.dataTable?.getSelectedKeys() || [];
        if (keys.length === 0) return;

        try {
            await this.api.post('/api/ftp/users/batch', { action: enabled ? 'enable' : 'disable', usernames: keys });
            this.message.success(`已${enabled ? '启用' : '禁用'} ${keys.length} 个用户`);
            this.dataTable.clearSelection();
            // 清除缓存后刷新
            this.api.clearCache('/api/ftp/users');
            await this.refresh();
        } catch (error) {
            this.message.error('操作失败: ' + error.message);
        }
    }

    async batchDelete() {
        const keys = this.dataTable?.getSelectedKeys() || [];
        if (keys.length === 0) return;
        
        // 使用自定义确认弹窗
        this.deleteTargetUsername = keys.join(',');
        this.$('#ftp-delete-username').textContent = `${keys.length} 个用户`;
        this.$('#ftp-delete-modal')?.classList.add('active');
        
        // 修改确认按钮行为
        const confirmBtn = this.$('#ftp-delete-confirm');
        confirmBtn.onclick = async () => {
            this.hideDeleteConfirm();
            try {
                await this.api.post('/api/ftp/users/batch', { action: 'delete', usernames: keys });
                this.message.success(`已删除 ${keys.length} 个用户`);
                this.dataTable.clearSelection();
                // 清除缓存后刷新
                this.api.clearCache('/api/ftp/users');
                await this.refresh();
            } catch (error) {
                this.message.error('删除失败: ' + error.message);
            }
        };
    }

    // ========== 编辑器 ==========

    showEditor() {
        const modal = this.$('#ftp-user-modal');
        const title = this.$('#ftp-modal-title');
        if (!modal) return;

        if (this.editingUser) {
            title.textContent = '编辑FTP';
            this.$('#ftp-form-username').value = this.editingUser.username || '';
            this.$('#ftp-form-username').disabled = true;  // 编辑时禁用用户名
            this.$('#ftp-form-password').value = this.editingUser.password || '';
            this.$('#ftp-form-password').type = 'text';  // 编辑时显示密码
            this.$('#ftp-form-root').value = this.editingUser.rootPath || '/';
            this.$('#ftp-form-quota').value = this.editingUser.quota || '';
            this.$('#ftp-form-expiry').value = this.editingUser.expiryDays || '';
            this.$('#ftp-form-status').value = this.editingUser.status || 'enabled';
            this.$('#ftp-form-remark').value = this.editingUser.remark || '';
        } else {
            title.textContent = '添加FTP';
            this.$('#ftp-user-form')?.reset();
            this.$('#ftp-form-username').disabled = false;  // 添加时启用用户名
            this.$('#ftp-form-password').type = 'password';  // 添加时密码隐藏
            // 默认根目录 = FTP根目录
            const defaultFtpRoot = this.state?.get?.('config')?.ftp?.root || './ftp';
            this.$('#ftp-form-root').value = defaultFtpRoot;
        }

        modal.classList.add('active');
        
        // 监听用户名输入，自动更新根目录
        const usernameInput = this.$('#ftp-form-username');
        if (usernameInput && !this.editingUser) {
            usernameInput.addEventListener('input', () => {
                const username = usernameInput.value.trim();
                if (username) {
                    const defaultFtpRoot = this.state?.get?.('config')?.ftp?.root || './ftp';
                    const userPath = defaultFtpRoot.endsWith('/') 
                        ? defaultFtpRoot + username 
                        : defaultFtpRoot + '/' + username;
                    this.$('#ftp-form-root').value = userPath;
                }
            });
        }
    }

    hideEditor() {
        this.$('#ftp-user-modal')?.classList.remove('active');
        this.editingUser = null;
        // 重置表单状态
        this.$('#ftp-form-username').disabled = false;
        this.$('#ftp-form-password').type = 'password';
    }

    showPortModal() {
        this.$('#ftp-port-modal')?.classList.add('active');
    }

    hidePortModal() {
        this.$('#ftp-port-modal')?.classList.remove('active');
    }

    showConfigModal() {
        if (!this.editingUser) return;
        
        // 填充配置数据
        this.$('#ftp-config-username').textContent = this.editingUser.username || '';
        this.$('#ftp-config-rootpath').textContent = this.editingUser.rootPath || '/';
        
        // 默认值
        this.$('#ftp-config-speed-limit').value = this.editingUser.speedLimit || 0;
        this.$('#ftp-config-max-connections').value = this.editingUser.maxConnections || 0;
        this.$('#ftp-config-bandwidth').value = this.editingUser.bandwidth || 0;
        this.$('#ftp-config-max-files').value = this.editingUser.maxFiles || 0;
        this.$('#ftp-config-max-file-size').value = this.editingUser.maxFileSize || 0;
        
        this.$('#ftp-config-modal')?.classList.add('active');
    }

    hideConfigModal() {
        this.$('#ftp-config-modal')?.classList.remove('active');
    }

    async saveConfig() {
        if (!this.editingUser) return;
        
        const config = {
            username: this.editingUser.username,
            speedLimit: parseInt(this.$('#ftp-config-speed-limit')?.value) || 0,
            maxConnections: parseInt(this.$('#ftp-config-max-connections')?.value) || 0,
            bandwidth: parseInt(this.$('#ftp-config-bandwidth')?.value) || 0,
            maxFiles: parseInt(this.$('#ftp-config-max-files')?.value) || 0,
            maxFileSize: parseInt(this.$('#ftp-config-max-file-size')?.value) || 0,
        };
        
        try {
            await this.api.post(`/api/ftp/users/${this.editingUser.username}/config`, config);
            this.message.success('配置已保存');
            this.hideConfigModal();
            this.api.clearCache('/api/ftp/users');
            await this.refresh();
        } catch (error) {
            this.message.error('保存失败: ' + error.message);
        }
    }

    async savePort() {
        const port = parseInt(this.$('#ftp-port-input')?.value) || 2121;

        try {
            await this.api.post('/api/ftp/port', { port });
            this.message.success('FTP 端口已更新，重启服务生效');
            this.hidePortModal();
        } catch (error) {
            this.message.error('保存失败: ' + error.message);
        }
    }

    async saveUser() {
        const username = this.$('#ftp-form-username')?.value.trim();
        const password = this.$('#ftp-form-password')?.value.trim();
        const rootPath = this.$('#ftp-form-root')?.value.trim() || '/';
        const quota = parseInt(this.$('#ftp-form-quota')?.value) || 0;
        const expiryDays = parseInt(this.$('#ftp-form-expiry')?.value) || 0;
        const status = this.$('#ftp-form-status')?.value || 'enabled';
        const remark = this.$('#ftp-form-remark')?.value.trim() || '';

        if (!username) { this.message.error('请输入用户名'); return; }
        if (!this.editingUser && !password) { this.message.error('请输入密码'); return; }

        const data = { 
            username, 
            password,
            rootPath, 
            quota, 
            expiryDays, 
            status, 
            remark 
        };

        try {
            if (this.editingUser) {
                await this.api.put(`/api/ftp/users/${this.editingUser.username}`, data);
                this.message.success('用户已更新');
            } else {
                await this.api.post('/api/ftp/users/add', data);
                this.message.success('用户已创建');
            }
            this.hideEditor();
            // 清除缓存后刷新
            this.api.clearCache('/api/ftp/users');
            await this.refresh();
        } catch (error) {
            this.message.error('保存失败: ' + error.message);
        }
    }
}

// 单例
let instance = null;

export function initFtpTab(deps) {
    if (!instance) {
        instance = new FtpTab(deps);
        instance.init();
    }
    return instance;
}