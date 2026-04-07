/**
 * 站点管理模块 v2
 * 
 * 使用 DataTable 组件实现站点列表、分页、搜索、批量操作
 */

import { BaseTab } from './BaseTab.js';
import { DataTable } from '../components/data-table.js';
import { openFileBrowser } from '../components/file-browser/index.js';
import { escapeHtml, copyToClipboard } from '../core/utils.js';

class SitesTab extends BaseTab {
    constructor(deps) {
        super(deps, 'sites');
        this.dataTable = null;
        this.sites = [];
        this.editingSite = null;
        this.deleteTargetId = null;
    }

    onInit() {
        console.log('初始化站点管理...');
        this.initDataTable();
        this.bindEvents();
        this.checkServiceStatus();
    }

    // ========== DataTable ==========

    initDataTable() {
        const container = this.$('#sites-table-container');
        if (!container) return;

        this.dataTable = new DataTable({
            container,
            columns: this.getColumns(),
            selectable: true,
            pageSize: 20,
            emptyText: '暂无站点',
            emptyHint: '点击上方"添加网站"按钮创建',
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
                title: '站点名称',
                dataIndex: 'name',
                className: 'col-name',
                render: (value, row) => {
                    const isStatic = row.type === 'static';
                    const icon = isStatic ?
                        `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="site-icon"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>` :
                        `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="site-icon"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>`;
                    return `
                        <div class="site-name-cell">
                            ${icon}
                            <div class="site-name-info">
                                <span class="site-name">${escapeHtml(value)}</span>
                                <span class="site-type-badge ${row.type}">${row.type === 'static' ? '静态' : '代理'}</span>
                            </div>
                        </div>
                    `;
                }
            },
            {
                title: '状态',
                dataIndex: 'enabled',
                className: 'col-status',
                render: (value, row) => `
                    <button class="status-btn ${value ? 'active' : ''}" data-id="${escapeHtml(row.id)}">${value ? '运行中' : '已停止'}</button>
                `
            },
            {
                title: '端口',
                dataIndex: 'port',
                className: 'col-port',
                render: (value) => `<span class="site-port">${value > 0 ? value : '共享'}</span>`
            },
            {
                title: '域名',
                dataIndex: 'domain',
                className: 'col-domain',
                render: (value) => {
                    if (!value || value.length === 0) return '<span class="site-domain empty">-</span>';
                    return `<span class="site-domain">${value.map(d => escapeHtml(d)).join(', ')}</span>`;
                }
            },
                        {
                title: '快速链接',
                dataIndex: 'id',
                className: 'col-link',
                render: (value, row) => {
                    let url = row.port > 0 ? `http://${window.location.hostname}:${row.port}` :
                              (row.domain?.length > 0 ? `http://${row.domain[0]}` : '');
                    if (!url) return '-';
                    return `
                        <div class="quick-link">
                            <a class="quick-link-text" href="${escapeHtml(url)}" target="_blank" title="点击打开">${escapeHtml(url)}</a>
                            <button class="quick-link-copy" data-link="${escapeHtml(url)}" title="复制链接">
                                <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                            </button>
                        </div>
                    `;
                }
            },
            {
                title: '根目录',
                dataIndex: 'root',
                className: 'col-root',
                render: (value, row) => {
                    if (row.type === 'static') {
                        return `<div class="root-path"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="icon"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg><a class="root-link" href="#" data-browse-path="${escapeHtml(value || '.')}" title="在文件管理中打开">${escapeHtml(value || '-')}</a></div>`;
                    }
                    return `<div class="root-path"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="icon"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg><code>${escapeHtml(row.proxy?.target || '-')}</code></div>`;
                }
            },
            {
                title: '操作',
                dataIndex: 'id',
                className: 'col-actions',
                render: (value) => `<div class="actions"><button class="action-text edit" data-id="${escapeHtml(value)}">编辑</button><button class="action-text delete" data-id="${escapeHtml(value)}">删除</button></div>`
            }
        ];
    }

    // ========== 事件绑定 ==========

    bindEvents() {
        // 服务控制
        this.$('#site-service-toggle')?.addEventListener('click', () => this.toggleService());
        this.$('#site-service-restart')?.addEventListener('click', () => this.restartService());
        this.$('#site-service-reload')?.addEventListener('click', () => this.reloadConfig());

        // 添加、刷新、搜索、过滤
        this.$('#add-site-btn')?.addEventListener('click', () => { this.editingSite = null; this.showEditor(); });
        this.$('#sites-refresh-btn')?.addEventListener('click', () => this.refresh());
        
        let searchTimer;
        this.$('#sites-search')?.addEventListener('input', () => {
            clearTimeout(searchTimer);
            searchTimer = setTimeout(() => this.filterSites(), 300);
        });
        this.$('#sites-type-filter')?.addEventListener('change', () => this.filterSites());
        this.$('#sites-status-filter')?.addEventListener('change', () => this.filterSites());

        // 批量操作按钮事件已移至 batch-bar 组件内部处理

        // 弹窗
        this.$('#site-modal-cancel')?.addEventListener('click', () => this.hideEditor());
        this.$('#site-modal-close')?.addEventListener('click', () => this.hideEditor());
        this.$('#site-modal-confirm')?.addEventListener('click', () => this.saveSite());
        this.$('#site-modal')?.querySelector('.modal-overlay')?.addEventListener('click', () => this.hideEditor());
        this.$('#site-form-type')?.addEventListener('change', () => this.onTypeChange());

        // 目录浏览按钮
        this.$$('.directory-picker-btn').forEach(btn => {
            btn.addEventListener('click', () => this.openDirPicker(btn.dataset.dir));
        });
        // 删除确认弹窗
        this.$('#site-delete-modal-close')?.addEventListener('click', () => this.hideDeleteConfirm());
        this.$('#site-delete-cancel')?.addEventListener('click', () => this.hideDeleteConfirm());
        this.$('#site-delete-confirm')?.addEventListener('click', () => this.confirmDelete());
        this.$('#site-delete-modal')?.querySelector('.modal-overlay')?.addEventListener('click', () => this.hideDeleteConfirm());
    }

    async openDirPicker(inputId) {
        const input = this.$(`#${inputId}`);
        if (!input) return;
        try {
            const selected = await openFileBrowser({
                title: '选择目录',
                selectMode: 'folder',
                root: input.value || '.',
                api: this.api,
            });
            if (selected) {
                input.value = selected;
                input.dispatchEvent(new Event('change', { bubbles: true }));
            }
        } catch (e) {
            // 用户取消
        }
    }

    bindRowEvents() {
        // 根目录链接（跳转文件管理）
        this.$$('.root-link[data-browse-path]').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const path = link.dataset.browsePath || '.';
                if (path && window.app?.switchTab) {
                    window.app.switchTab('files');
                    setTimeout(() => {
                        this.events.emit('files:navigate', path);
                    }, 150);
                }
            });
        });

        // 状态按钮
        this.$$('.status-btn[data-id]').forEach(btn => {
            btn.addEventListener('click', () => {
                const id = btn.dataset.id;
                const isActive = btn.classList.contains('active');
                this.toggleStatus(id, !isActive);
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
                this.editingSite = this.sites.find(s => s.id === btn.dataset.id);
                this.showEditor();
            });
        });

        // 删除
        this.$$('.action-text.delete').forEach(btn => {
            btn.addEventListener('click', () => {
                this.showDeleteConfirm(btn.dataset.id);
            });
        });
    }

    // ========== 数据操作 ==========

    async onLoad() {
        if (!this.dataTable) return;
        this.dataTable.setLoading(true);

        try {
            this.sites = await this.api.getJSON('/api/sites') || [];
            this.filterSites();
        } finally {
            this.dataTable.setLoading(false);
        }
    }

    filterSites() {
        if (!this.dataTable) return;

        const search = this.$('#sites-search')?.value?.toLowerCase() || '';
        const type = this.$('#sites-type-filter')?.value || '';
        const status = this.$('#sites-status-filter')?.value || '';

        const filtered = this.sites.filter(site => {
            if (search && !site.name?.toLowerCase().includes(search) &&
                !site.domain?.some(d => d.toLowerCase().includes(search))) return false;
            if (type && site.type !== type) return false;
            if (status === 'enabled' && !site.enabled) return false;
            if (status === 'disabled' && site.enabled) return false;
            return true;
        });

        this.dataTable.updateData(filtered);
        setTimeout(() => this.bindRowEvents(), 0);
    }

    async toggleStatus(id, enabled) {
        try {
            if (enabled) {
                await this.api.post('/api/sites/start', { id });
                this.toast.success('站点已启动');
            } else {
                await this.api.post('/api/sites/stop', { id });
                this.toast.success('站点已停止');
            }
            const site = this.sites.find(s => s.id === id);
            if (site) site.enabled = enabled;
            this.filterSites();
        } catch (error) {
            this.toast.error('操作失败: ' + error.message);
        }
    }

    // ========== 服务控制 ==========

    async checkServiceStatus() {
        try {
            const data = await this.api.getJSON('/api/sites/status');
            if (data) {
                this.updateServiceStatus(data.running);
            }
        } catch (error) {
            console.error('获取站点服务状态失败:', error);
        }
    }

    async toggleService() {
        const btn = this.$('#site-service-toggle');
        const isRunning = btn?.classList.contains('running');

        try {
            if (isRunning) {
                await this.api.post('/api/service/sites/stop');
                this.toast.success('站点服务已停止');
                this.updateServiceStatus(false);
            } else {
                await this.api.post('/api/service/sites/start');
                this.toast.success('站点服务已启动');
                this.updateServiceStatus(true);
            }
        } catch (error) {
            this.toast.error('操作失败: ' + error.message);
        }
    }

    async restartService() {
        try {
            await this.api.post('/api/service/sites/restart');
            this.toast.success('站点服务已重启');
            this.updateServiceStatus(true);
        } catch (error) {
            this.toast.error('重启失败: ' + error.message);
        }
    }

    async reloadConfig() {
        try {
            await this.api.post('/api/service/sites/reload');
            this.toast.success('站点配置已重载');
            await this.refresh();
        } catch (error) {
            this.toast.error('重载失败: ' + error.message);
        }
    }

    updateServiceStatus(running) {
        const statusEl = this.$('#site-service-status');
        const toggleBtn = this.$('#site-service-toggle');

        if (statusEl) {
            statusEl.textContent = running ? '运行中' : '已停止';
            statusEl.classList.toggle('running', running);
            statusEl.classList.toggle('stopped', !running);
        }

        if (toggleBtn) {
            toggleBtn.classList.toggle('running', running);
            toggleBtn.classList.toggle('stopped', !running);
            const textSpan = toggleBtn.querySelector('.btn-text');
            if (textSpan) {
                textSpan.textContent = running ? '停止' : '启动';
            }
        }
    }

    showDeleteConfirm(id) {
        const site = this.sites.find(s => s.id === id);
        if (!site) return;
        this.deleteTargetId = id;
        this.$('#site-delete-name').textContent = site.name;
        this.$('#site-delete-modal')?.classList.add('active');
    }

    hideDeleteConfirm() {
        this.$('#site-delete-modal')?.classList.remove('active');
        this.deleteTargetId = null;
    }

    async confirmDelete() {
        if (!this.deleteTargetId) return;

        if (this.deleteTargetId === 'batch') {
            // 批量删除
            const keys = this._batchDeleteKeys || [];
            this.hideDeleteConfirm();
            if (keys.length === 0) return;

            try {
                const response = await this.api.post('/api/sites/batch', { action: 'delete', ids: keys });
                if (!response || !response.ok) {
                    throw new Error(response ? `HTTP ${response.status}` : '请求失败');
                }
                this.api.clearCache('/api/sites');
                this.message.success(`已删除 ${keys.length} 个站点`);
                this.dataTable.clearSelection();
                await this.refresh();
            } catch (error) {
                this.message.error('删除失败: ' + error.message);
            }
            this._batchDeleteKeys = null;
        } else {
            // 单个删除
            const id = this.deleteTargetId;
            this.hideDeleteConfirm();
            await this.deleteSite(id);
        }
    }

    async deleteSite(id) {
        try {
            const response = await this.api.delete(`/api/sites/${id}`);
            if (!response || !response.ok) {
                throw new Error(response ? `HTTP ${response.status}` : '请求失败');
            }
            this.api.clearCache('/api/sites');
            this.message.success('站点已删除');
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
            await this.api.post('/api/sites/batch', { action: enabled ? 'enable' : 'disable', ids: keys });
            this.message.success(`已${enabled ? '启用' : '禁用'} ${keys.length} 个站点`);
            this.dataTable.clearSelection();
            await this.refresh();
        } catch (error) {
            this.message.error('操作失败: ' + error.message);
        }
    }

    async batchDelete() {
        const keys = this.dataTable?.getSelectedKeys() || [];
        if (keys.length === 0) return;

        // 显示批量删除确认弹窗
        this.deleteTargetId = 'batch';
        this.$('#site-delete-name').textContent = `${keys.length} 个站点`;
        this.$('#site-delete-modal')?.classList.add('active');
        this._batchDeleteKeys = keys;
    }

    async confirmBatchDelete() {
        const keys = this._batchDeleteKeys;
        if (!keys?.length) return;

        try {
            await this.api.post('/api/sites/batch', { action: 'delete', ids: keys });
            this.message.success(`已删除 ${keys.length} 个站点`);
            this.dataTable.clearSelection();
            await this.refresh();
        } catch (error) {
            this.message.error('删除失败: ' + error.message);
        }
    }

    // ========== 编辑器 ==========

    showEditor() {
        const modal = this.$('#site-modal');
        const title = this.$('#site-modal-title');
        if (!modal) return;

        if (this.editingSite) {
            title.textContent = '编辑网站';
            this.$('#site-form-name').value = this.editingSite.name || '';
            this.$('#site-form-type').value = this.editingSite.type || 'static';
            this.$('#site-form-port').value = this.editingSite.port || '';
            this.$('#site-form-domain').value = (this.editingSite.domain || []).join(', ');
            this.$('#site-form-proxy').value = this.editingSite.proxy?.target || '';
            this.$('#site-form-root').value = this.editingSite.root || './sites/default';
            // 首页文件 & 目录浏览
            const indexFiles = this.editingSite.index_files || ['index.html', 'index.htm'];
            this.$('#site-form-index-files').value = indexFiles.join(', ');
            const autoIndex = this.$('#site-form-auto-index');
            if (autoIndex) autoIndex.checked = this.editingSite.auto_index !== false;
        } else {
            title.textContent = '添加网站';
            this.$('#site-form')?.reset();
            this.$('#site-form-root').value = './sites/default';
            // 新建默认值
            this.$('#site-form-index-files').value = 'index.html, index.htm';
            const autoIndex = this.$('#site-form-auto-index');
            if (autoIndex) autoIndex.checked = true;
        }

        this.onTypeChange();
        modal.classList.add('active');
    }

    hideEditor() {
        this.$('#site-modal')?.classList.remove('active');
        this.editingSite = null;
    }

    onTypeChange() {
        const type = this.$('#site-form-type')?.value || 'static';
        const staticFields = this.$('#site-static-fields');
        const proxyFields = this.$('#site-proxy-fields');
        if (staticFields) staticFields.style.display = type === 'static' ? 'block' : 'none';
        if (proxyFields) proxyFields.style.display = type === 'proxy' ? 'block' : 'none';
    }

    async saveSite() {
        const name = this.$('#site-form-name')?.value.trim();
        const type = this.$('#site-form-type')?.value;
        const port = parseInt(this.$('#site-form-port')?.value) || 0;
        const domainStr = this.$('#site-form-domain')?.value.trim();
        const root = this.$('#site-form-root')?.value.trim();
        const proxyTarget = this.$('#site-form-proxy')?.value.trim();

        if (!name) { this.message.error('请输入站点名称'); return; }

        const data = {
            name, type, port,
            domain: domainStr ? domainStr.split(',').map(d => d.trim()).filter(Boolean) : [],
            enabled: true
        };

        if (type === 'static') {
            data.root = root || './sites/default';
            const indexFilesStr = this.$('#site-form-index-files')?.value.trim();
            data.index_files = indexFilesStr
                ? indexFilesStr.split(',').map(f => f.trim()).filter(Boolean)
                : ['index.html', 'index.htm'];
            data.auto_index = this.$('#site-form-auto-index')?.checked ?? true;
        } else {
            if (!proxyTarget) { this.message.error('请输入代理目标'); return; }
            data.proxy = { target: proxyTarget };
        }

        try {
            if (this.editingSite) {
                await this.api.put(`/api/sites/${this.editingSite.id}`, data);
                this.message.success('站点已更新');
            } else {
                await this.api.post('/api/sites', data);
                this.message.success('站点已创建');
            }
            this.hideEditor();
            this.api.clearCache('/api/sites');
            await this.refresh();
        } catch (error) {
            this.message.error('保存失败: ' + error.message);
        }
    }
}

// 单例
let instance = null;

export function initSitesTab(deps) {
    if (!instance) {
        instance = new SitesTab(deps);
        instance.init();
    }
    return instance;
}