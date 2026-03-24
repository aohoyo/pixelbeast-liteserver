/**
 * 站点管理模块 v2
 * 
 * 使用 DataTable 组件实现站点列表、分页、搜索、批量操作
 */

import { globalEvents } from '../core/events.js';
import { DataTable } from '../components/data-table.js';

// 存储依赖
let deps = null;

// DataTable 实例
let dataTable = null;

// 站点数据
let sites = [];

// 当前编辑的站点
let editingSite = null;

// 列配置
const columns = [
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
                    <input type="checkbox" class="site-status-toggle" data-id="${escapeHtml(row.id)}"
                        ${value ? 'checked' : ''}>
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
                return `
                    <div class="site-root">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="icon-sm">
                            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                        </svg>
                        <code>${escapeHtml(value || '-')}</code>
                    </div>
                `;
            }
            return `
                <div class="site-proxy">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="icon-sm">
                        <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
                        <polyline points="15 3 21 3 21 9"/>
                        <line x1="10" y1="14" x2="21" y2="3"/>
                    </svg>
                    <code>${escapeHtml(row.proxy?.target || '-')}</code>
                </div>
            `;
        }
    },
    {
        title: '快速链接',
        dataIndex: 'id',
        className: 'col-link',
        render: (value, row) => {
            let url = '';
            if (row.port > 0) {
                url = `http://nas.banayou.com:${row.port}`;
            } else if (row.domain && row.domain.length > 0) {
                url = `http://${row.domain[0]}`;
            }
            if (!url) return '-';
            return `
                <div class="site-quick-link">
                    <span class="site-link-text" title="${escapeHtml(url)}">${escapeHtml(url)}</span>
                    <button class="site-link-copy" data-url="${escapeHtml(url)}" title="复制链接">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                        </svg>
                    </button>
                    <a href="${escapeHtml(url)}" target="_blank" class="site-link-open" title="打开">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
                            <polyline points="15 3 21 3 21 9"/>
                            <line x1="10" y1="14" x2="21" y2="3"/>
                        </svg>
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
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                    </svg>
                </button>
                <button class="site-action-btn delete" data-id="${escapeHtml(value)}" title="删除">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="3 6 5 6 21 6"/>
                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                    </svg>
                </button>
            </div>
        `
    }
];

/**
 * 初始化站点管理
 */
export function initSitesTab({ state, api, toast, message, dialog }) {
    console.log('初始化站点管理...');

    deps = { state, api, toast, message, dialog };

    // 初始化 DataTable
    initDataTable();

    // 绑定事件
    bindEvents();

    // 监听标签页切换
    globalEvents.match('tab:switch:sites', () => {
        loadSites();
    });
}

/**
 * 初始化 DataTable
 */
function initDataTable() {
    const container = document.getElementById('sites-table-container');
    if (!container) return;

    dataTable = new DataTable({
        container,
        columns,
        selectable: true,
        pageSize: 20,
        emptyText: '暂无站点',
        emptyHint: '点击上方"添加网站"按钮创建',
        loadingText: '加载中...',
        onSelectionChange: ({ selectedCount }) => {
            updateBatchActions(selectedCount);
        },
        onPageChange: () => {
            setTimeout(bindRowEvents, 0);
        }
    });
}

/**
 * 绑定事件
 */
function bindEvents() {
    if (!deps) return;

    // 添加站点
    document.getElementById('add-site-btn')?.addEventListener('click', () => {
        editingSite = null;
        showSiteEditor();
    });

    // 刷新
    document.getElementById('sites-refresh-btn')?.addEventListener('click', loadSites);

    // 搜索
    let searchTimer;
    document.getElementById('sites-search')?.addEventListener('input', () => {
        clearTimeout(searchTimer);
        searchTimer = setTimeout(filterSites, 300);
    });

    // 类型过滤
    document.getElementById('sites-type-filter')?.addEventListener('change', filterSites);

    // 状态过滤
    document.getElementById('sites-status-filter')?.addEventListener('change', filterSites);

    // 批量操作
    document.getElementById('sites-batch-enable')?.addEventListener('click', () => batchToggle(true));
    document.getElementById('sites-batch-disable')?.addEventListener('click', () => batchToggle(false));
    document.getElementById('sites-batch-delete')?.addEventListener('click', batchDelete);

    // 弹窗
    document.getElementById('site-modal-cancel')?.addEventListener('click', hideSiteEditor);
    document.getElementById('site-modal-close')?.addEventListener('click', hideSiteEditor);
    document.getElementById('site-modal-confirm')?.addEventListener('click', saveSite);
    document.getElementById('site-modal')?.querySelector('.modal-overlay')?.addEventListener('click', hideSiteEditor);

    // 类型切换
    document.getElementById('site-form-type')?.addEventListener('change', onTypeChange);
}

/**
 * 加载站点列表
 */
async function loadSites() {
    if (!deps || !dataTable) return;
    const { api, toast } = deps;

    dataTable.setLoading(true);

    try {
        const data = await api.getJSON('/api/sites');
        sites = data || [];
        filterSites();
    } catch (error) {
        console.error('加载站点失败:', error);
        toast.error('加载失败: ' + error.message);
        dataTable.updateData([]);
    } finally {
        dataTable.setLoading(false);
    }
}

/**
 * 筛选站点
 */
function filterSites() {
    if (!dataTable) return;

    const search = document.getElementById('sites-search')?.value?.toLowerCase() || '';
    const typeFilter = document.getElementById('sites-type-filter')?.value || '';
    const statusFilter = document.getElementById('sites-status-filter')?.value || '';

    const filteredSites = sites.filter(site => {
        // 搜索过滤
        if (search && !site.name?.toLowerCase().includes(search) &&
            !site.domain?.some(d => d.toLowerCase().includes(search))) {
            return false;
        }
        // 类型过滤
        if (typeFilter && site.type !== typeFilter) {
            return false;
        }
        // 状态过滤
        if (statusFilter === 'enabled' && !site.enabled) {
            return false;
        }
        if (statusFilter === 'disabled' && site.enabled) {
            return false;
        }
        return true;
    });

    dataTable.updateData(filteredSites);
    setTimeout(bindRowEvents, 0);
}

/**
 * 绑定行内事件
 */
function bindRowEvents() {
    if (!deps) return;
    const { message } = deps;

    // 状态开关
    document.querySelectorAll('.site-status-toggle').forEach(toggle => {
        toggle.addEventListener('change', (e) => {
            const id = e.target.dataset.id;
            const enabled = e.target.checked;
            toggleSiteStatus(id, enabled);
        });
    });

    // 复制链接
    document.querySelectorAll('.site-link-copy').forEach(btn => {
        btn.addEventListener('click', () => {
            const url = btn.dataset.url;
            copyToClipboard(url);
            message.success('链接已复制');
        });
    });

    // 编辑
    document.querySelectorAll('.site-action-btn.edit').forEach(btn => {
        btn.addEventListener('click', () => {
            const id = btn.dataset.id;
            editingSite = sites.find(s => s.id === id);
            showSiteEditor();
        });
    });

    // 删除
    document.querySelectorAll('.site-action-btn.delete').forEach(btn => {
        btn.addEventListener('click', () => {
            const id = btn.dataset.id;
            const site = sites.find(s => s.id === id);
            if (confirm(`确定要删除站点 "${site?.name}" 吗？`)) {
                deleteSite(id);
            }
        });
    });
}

/**
 * 更新批量操作显示
 */
function updateBatchActions(selectedCount) {
    const batchEl = document.getElementById('sites-batch-actions');
    const countEl = document.getElementById('sites-selected-count');

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
async function batchToggle(enabled) {
    if (!deps || !dataTable) return;
    const { api, message } = deps;

    const selectedKeys = dataTable.getSelectedKeys();
    if (selectedKeys.length === 0) return;

    try {
        await api.post('/api/sites/batch', {
            action: enabled ? 'enable' : 'disable',
            ids: selectedKeys
        });
        message.success(`已${enabled ? '启用' : '禁用'} ${selectedKeys.length} 个站点`);
        dataTable.clearSelection();
        loadSites();
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

    if (!confirm(`确定要删除选中的 ${selectedKeys.length} 个站点吗？`)) {
        return;
    }

    try {
        await api.post('/api/sites/batch', {
            action: 'delete',
            ids: selectedKeys
        });
        message.success(`已删除 ${selectedKeys.length} 个站点`);
        dataTable.clearSelection();
        loadSites();
    } catch (error) {
        message.error('删除失败: ' + error.message);
    }
}

/**
 * 切换站点状态
 */
async function toggleSiteStatus(id, enabled) {
    if (!deps) return;
    const { api, toast } = deps;

    try {
        await api.post('/api/sites/toggle', { id, enabled });
        toast.success(`站点已${enabled ? '启用' : '禁用'}`);
        const site = sites.find(s => s.id === id);
        if (site) {
            site.enabled = enabled;
            filterSites();
        }
    } catch (error) {
        toast.error('操作失败: ' + error.message);
        filterSites();
    }
}

/**
 * 删除站点
 */
async function deleteSite(id) {
    if (!deps) return;
    const { api, message } = deps;

    try {
        await api.delete(`/api/sites/${id}`);
        message.success('站点已删除');
        loadSites();
    } catch (error) {
        message.error('删除失败: ' + error.message);
    }
}

/**
 * 显示站点编辑器
 */
function showSiteEditor() {
    const modal = document.getElementById('site-modal');
    const title = document.getElementById('site-modal-title');
    const form = document.getElementById('site-form');

    if (!modal) return;

    if (editingSite) {
        title.textContent = '编辑网站';
        document.getElementById('site-form-name').value = editingSite.name || '';
        document.getElementById('site-form-type').value = editingSite.type || 'static';
        document.getElementById('site-form-port').value = editingSite.port || '';
        document.getElementById('site-form-domain').value = (editingSite.domain || []).join(', ');
        document.getElementById('site-form-root').value = editingSite.root || '';
        document.getElementById('site-form-proxy').value = editingSite.proxy?.target || '';
    } else {
        title.textContent = '添加网站';
        form?.reset();
    }

    onTypeChange();
    modal.classList.add('active');
}

/**
 * 隐藏站点编辑器
 */
function hideSiteEditor() {
    const modal = document.getElementById('site-modal');
    if (modal) modal.classList.remove('active');
    editingSite = null;
}

/**
 * 类型切换
 */
function onTypeChange() {
    const type = document.getElementById('site-form-type')?.value || 'static';
    const staticFields = document.getElementById('site-static-fields');
    const proxyFields = document.getElementById('site-proxy-fields');

    if (staticFields) {
        staticFields.style.display = type === 'static' ? 'block' : 'none';
    }
    if (proxyFields) {
        proxyFields.style.display = type === 'proxy' ? 'block' : 'none';
    }
}

/**
 * 保存站点
 */
async function saveSite() {
    if (!deps) return;
    const { api, message } = deps;

    const name = document.getElementById('site-form-name')?.value.trim();
    const type = document.getElementById('site-form-type')?.value;
    const port = parseInt(document.getElementById('site-form-port')?.value) || 0;
    const domainStr = document.getElementById('site-form-domain')?.value.trim();
    const root = document.getElementById('site-form-root')?.value.trim();
    const proxyTarget = document.getElementById('site-form-proxy')?.value.trim();

    if (!name) {
        message.error('请输入站点名称');
        return;
    }

    const data = {
        name,
        type,
        port,
        domain: domainStr ? domainStr.split(',').map(d => d.trim()).filter(Boolean) : [],
        enabled: true
    };

    if (type === 'static') {
        data.root = root || './data/sites/default';
        data.index_files = ['index.html', 'index.htm'];
        data.auto_index = true;
    } else {
        if (!proxyTarget) {
            message.error('请输入代理目标');
            return;
        }
        data.proxy = { target: proxyTarget };
    }

    try {
        if (editingSite) {
            await api.put(`/api/sites/${editingSite.id}`, data);
            message.success('站点已更新');
        } else {
            await api.post('/api/sites', data);
            message.success('站点已创建');
        }
        hideSiteEditor();
        loadSites();
    } catch (error) {
        message.error('保存失败: ' + error.message);
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