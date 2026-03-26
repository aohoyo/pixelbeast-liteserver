/**
 * FTP 用户管理模块
 *
 * 使用 DataTable 组件实现用户列表、分页、搜索、批量操作
 */

import { BaseTab } from './BaseTab.js';
import { DataTable } from '../components/data-table.js';
import { escapeHtml, formatDate, copyToClipboard } from '../core/utils.js';

class FtpTab extends BaseTab {
    constructor(deps) {
        super(deps, 'ftp');
        this.dataTable = null;
        this.users = [];
        this.editingUser = null;
    }

    onInit() {
        console.log('初始化 FTP 用户管理面板...');
        this.initDataTable();
        this.bindEvents();
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
            onSelectionChange: ({ selectedCount }) => this.updateBatchActions(selectedCount),
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
                render: (value) => `<div class="ftp-password"><span class="ftp-password-text">••••••••</span><button class="ftp-password-copy" data-password="${escapeHtml(value)}" title="复制密码"><svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg></button></div>`
            },
            {
                title: '状态',
                dataIndex: 'status',
                className: 'col-status',
                render: (value, row) => `<div class="ftp-status"><label class="switch"><input type="checkbox" class="ftp-status-toggle" data-username="${escapeHtml(row.username)}" ${value === 'enabled' ? 'checked' : ''}><span class="slider"></span></label><span class="ftp-status-text ${value}">${value === 'enabled' ? '已启用' : '已禁用'}</span></div>`
            },
            {
                title: '快速链接',
                dataIndex: 'username',
                className: 'col-link',
                render: (value) => {
                    const link = `ftp://${value}@${window.location.hostname}`;
                    return `<div class="ftp-quick-link"><span class="ftp-link-text" title="${escapeHtml(link)}">${escapeHtml(link)}</span><button class="ftp-link-copy" data-link="${escapeHtml(link)}" title="复制链接"><svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg></button></div>`;
                }
            },
            {
                title: '根目录',
                dataIndex: 'rootPath',
                className: 'col-root',
                render: (value) => `<div class="ftp-root-path"><svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path></svg><span>${escapeHtml(value || '/')}</span></div>`
            },
            {
                title: '操作',
                dataIndex: 'username',
                className: 'col-actions',
                render: (value) => `<div class="ftp-actions"><button class="ftp-action-btn edit" data-username="${escapeHtml(value)}" title="编辑"><svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg></button><button class="ftp-action-btn delete" data-username="${escapeHtml(value)}" title="删除"><svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg></button></div>`
            }
        ];
    }

    // ========== 事件绑定 ==========

    bindEvents() {
        // 添加、刷新、搜索
        this.$('#add-ftp-user-btn')?.addEventListener('click', () => { this.editingUser = null; this.showEditor(); });
        this.$('#ftp-refresh-btn')?.addEventListener('click', () => this.refresh());
        
        let searchTimer;
        this.$('#ftp-search')?.addEventListener('input', () => {
            clearTimeout(searchTimer);
            searchTimer = setTimeout(() => this.filterUsers(), 300);
        });

        // 批量操作
        this.$('#ftp-batch-enable')?.addEventListener('click', () => this.batchToggle(true));
        this.$('#ftp-batch-disable')?.addEventListener('click', () => this.batchToggle(false));
        this.$('#ftp-batch-delete')?.addEventListener('click', () => this.batchDelete());

        // 弹窗
        this.$('#ftp-user-modal-cancel')?.addEventListener('click', () => this.hideEditor());
        this.$('#ftp-user-modal-close')?.addEventListener('click', () => this.hideEditor());
        this.$('#ftp-user-modal-confirm')?.addEventListener('click', () => this.saveUser());
        this.$('#ftp-user-modal')?.querySelector('.modal-overlay')?.addEventListener('click', () => this.hideEditor());
    }

    bindRowEvents() {
        // 状态开关
        this.$$('.ftp-status-toggle').forEach(toggle => {
            toggle.addEventListener('change', (e) => {
                this.toggleStatus(e.target.dataset.username, e.target.checked);
            });
        });

        // 复制密码
        this.$$('.ftp-password-copy').forEach(btn => {
            btn.addEventListener('click', () => {
                copyToClipboard(btn.dataset.password);
                this.message.success('密码已复制');
            });
        });

        // 复制链接
        this.$$('.ftp-link-copy').forEach(btn => {
            btn.addEventListener('click', () => {
                copyToClipboard(btn.dataset.link);
                this.message.success('链接已复制');
            });
        });

        // 编辑
        this.$$('.ftp-action-btn.edit').forEach(btn => {
            btn.addEventListener('click', () => {
                this.editingUser = this.users.find(u => u.username === btn.dataset.username);
                this.showEditor();
            });
        });

        // 删除
        this.$$('.ftp-action-btn.delete').forEach(btn => {
            btn.addEventListener('click', () => {
                if (confirm(`确定要删除用户 "${btn.dataset.username}" 吗？`)) {
                    this.deleteUser(btn.dataset.username);
                }
            });
        });
    }

    // ========== 数据操作 ==========

    async onLoad() {
        if (!this.dataTable) return;
        this.dataTable.setLoading(true);

        try {
            this.users = await this.api.getJSON('/api/ftp/users') || [];
            this.filterUsers();
        } finally {
            this.dataTable.setLoading(false);
        }
    }

    filterUsers() {
        if (!this.dataTable) return;

        const search = this.$('#ftp-search')?.value?.toLowerCase() || '';
        const filtered = this.users.filter(u => !search || u.username?.toLowerCase().includes(search));

        this.dataTable.updateData(filtered);
        setTimeout(() => this.bindRowEvents(), 0);
    }

    async toggleStatus(username, enabled) {
        try {
            await this.api.post('/api/ftp/users/toggle', { username, status: enabled ? 'enabled' : 'disabled' });
            this.toast.success(`用户已${enabled ? '启用' : '禁用'}`);
            const user = this.users.find(u => u.username === username);
            if (user) user.status = enabled ? 'enabled' : 'disabled';
            this.filterUsers();
        } catch (error) {
            this.toast.error('操作失败: ' + error.message);
        }
    }

    async deleteUser(username) {
        try {
            await this.api.delete(`/api/ftp/users/${username}`);
            this.message.success('用户已删除');
            await this.refresh();
        } catch (error) {
            this.message.error('删除失败: ' + error.message);
        }
    }

    // ========== 批量操作 ==========

    updateBatchActions(count) {
        const batchEl = this.$('#ftp-batch-actions');
        const countEl = this.$('#ftp-selected-count');
        if (batchEl && countEl) {
            batchEl.style.display = count > 0 ? 'flex' : 'none';
            countEl.textContent = count;
        }
    }

    async batchToggle(enabled) {
        const keys = this.dataTable?.getSelectedKeys() || [];
        if (keys.length === 0) return;

        try {
            await this.api.post('/api/ftp/users/batch', { action: enabled ? 'enable' : 'disable', usernames: keys });
            this.message.success(`已${enabled ? '启用' : '禁用'} ${keys.length} 个用户`);
            this.dataTable.clearSelection();
            await this.refresh();
        } catch (error) {
            this.message.error('操作失败: ' + error.message);
        }
    }

    async batchDelete() {
        const keys = this.dataTable?.getSelectedKeys() || [];
        if (keys.length === 0 || !confirm(`确定要删除选中的 ${keys.length} 个用户吗？`)) return;

        try {
            await this.api.post('/api/ftp/users/batch', { action: 'delete', usernames: keys });
            this.message.success(`已删除 ${keys.length} 个用户`);
            this.dataTable.clearSelection();
            await this.refresh();
        } catch (error) {
            this.message.error('删除失败: ' + error.message);
        }
    }

    // ========== 编辑器 ==========

    showEditor() {
        const modal = this.$('#ftp-user-modal');
        const title = this.$('#ftp-user-modal-title');
        if (!modal) return;

        if (this.editingUser) {
            title.textContent = '编辑用户';
            this.$('#ftp-form-username').value = this.editingUser.username || '';
            this.$('#ftp-form-password').value = this.editingUser.password || '';
            this.$('#ftp-form-root').value = this.editingUser.rootPath || '';
        } else {
            title.textContent = '添加用户';
            this.$('#ftp-user-form')?.reset();
        }

        modal.classList.add('active');
    }

    hideEditor() {
        this.$('#ftp-user-modal')?.classList.remove('active');
        this.editingUser = null;
    }

    async saveUser() {
        const username = this.$('#ftp-form-username')?.value.trim();
        const password = this.$('#ftp-form-password')?.value.trim();
        const rootPath = this.$('#ftp-form-root')?.value.trim() || '/';

        if (!username) { this.message.error('请输入用户名'); return; }
        if (!password) { this.message.error('请输入密码'); return; }

        const data = { username, password, rootPath, status: 'enabled' };

        try {
            if (this.editingUser) {
                await this.api.put(`/api/ftp/users/${this.editingUser.username}`, data);
                this.message.success('用户已更新');
            } else {
                await this.api.post('/api/ftp/users', data);
                this.message.success('用户已创建');
            }
            this.hideEditor();
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