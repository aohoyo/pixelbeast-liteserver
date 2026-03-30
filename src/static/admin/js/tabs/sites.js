/**
 * 站点管理模块 v2
 * 
 * 使用 DataTable 组件实现站点列表、分页、搜索、批量操作
 */

import { BaseTab } from './BaseTab.js';
import { DataTable } from '../components/data-table.js';
import { escapeHtml, copyToClipboard } from '../core/utils.js';

class SitesTab extends BaseTab {
    constructor(deps) {
        super(deps, 'sites');
        this.dataTable = null;
        this.sites = [];
        this.editingSite = null;
    }

    onInit() {
        console.log('初始化站点管理...');
        this.initDataTable();
        this.bindEvents();
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
                    <div class="site-status">
                        <label class="switch">
                            <input type="checkbox" class="site-status-toggle" data-id="${escapeHtml(row.id)}" ${value ? 'checked' : ''}>
                            <span class="slider"></span>
                        </label>
                        <span class="site-status-text ${value ? 'running' : 'stopped'}">${value ? '运行中' : '已停止'}</span>
                    </div>
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
                title: '根目录/代理目标',
                dataIndex: 'root',
                className: 'col-root',
                render: (value, row) => {
                    if (row.type === 'static') {
                        return `<div class="site-root"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="icon-sm"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg><code>${escapeHtml(value || '-')}</code></div>`;
                    }
                    return `<div class="site-proxy"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="icon-sm"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg><code>${escapeHtml(row.proxy?.target || '-')}</code></div>`;
                }
            },
            {
                title: '快速链接',
                dataIndex: 'id',
                className: 'col-link',
                render: (value, row) => {
                    let url = row.port > 0 ? `http://nas.banayou.com:${row.port}` :
                              (row.domain?.length > 0 ? `http://${row.domain[0]}` : '');
                    if (!url) return '-';
                    return `
                        <div class="site-quick-link">
                            <span class="site-link-text" title="${escapeHtml(url)}">${escapeHtml(url)}</span>
                            <button class="site-link-copy" data-url="${escapeHtml(url)}" title="复制链接">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                            </button>
                            <a href="${escapeHtml(url)}" target="_blank" class="site-link-open" title="打开">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
                            </a>
                        </div>
                    `;
                }
            },
            {
                title: '操作',
                dataIndex: 'id',
                className: 'col-actions',
                render: (value) => `
                    <div class="site-actions">
                        <button class="site-action-btn edit" data-id="${escapeHtml(value)}" title="编辑">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                        </button>
                        <button class="site-action-btn delete" data-id="${escapeHtml(value)}" title="删除">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                        </button>
                    </div>
                `
            }
        ];
    }

    // ========== 事件绑定 ==========

    bindEvents() {
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
    }

    bindRowEvents() {
        // 状态开关
        this.$$('.site-status-toggle').forEach(toggle => {
            toggle.addEventListener('change', (e) => {
                this.toggleStatus(e.target.dataset.id, e.target.checked);
            });
        });

        // 复制链接
        this.$$('.site-link-copy').forEach(btn => {
            btn.addEventListener('click', () => {
                copyToClipboard(btn.dataset.url);
                this.message.success('链接已复制');
            });
        });

        // 编辑
        this.$$('.site-action-btn.edit').forEach(btn => {
            btn.addEventListener('click', () => {
                this.editingSite = this.sites.find(s => s.id === btn.dataset.id);
                this.showEditor();
            });
        });

        // 删除
        this.$$('.site-action-btn.delete').forEach(btn => {
            btn.addEventListener('click', () => {
                const site = this.sites.find(s => s.id === btn.dataset.id);
                if (confirm(`确定要删除站点 "${site?.name}" 吗？`)) {
                    this.deleteSite(btn.dataset.id);
                }
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
            await this.api.post('/api/sites/toggle', { id, enabled });
            this.toast.success(`站点已${enabled ? '启用' : '禁用'}`);
            const site = this.sites.find(s => s.id === id);
            if (site) site.enabled = enabled;
            this.filterSites();
        } catch (error) {
            this.toast.error('操作失败: ' + error.message);
        }
    }

    async deleteSite(id) {
        try {
            await this.api.delete(`/api/sites/${id}`);
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
        if (keys.length === 0 || !confirm(`确定要删除选中的 ${keys.length} 个站点吗？`)) return;

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
            this.$('#site-form-root').value = this.editingSite.root || '';
            this.$('#site-form-proxy').value = this.editingSite.proxy?.target || '';
        } else {
            title.textContent = '添加网站';
            this.$('#site-form')?.reset();
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
            data.index_files = ['index.html', 'index.htm'];
            data.auto_index = true;
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