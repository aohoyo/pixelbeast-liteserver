/**
 * 站点管理模块 v3
 *
 * 使用 DataTable 组件实现站点列表、分页、搜索、批量操作
 * 集成 SSL 证书管理：证书库选择、粘贴证书
 */

import { BaseTab } from './BaseTab.js';
import { DataTable } from '../components/data-table.js';
import { escapeHtml } from '../core/utils.js';
import { openDirPicker } from '../core/dir-picker.js';

class SitesTab extends BaseTab {
    constructor(deps) {
        super(deps, 'sites');
        this.dataTable = null;
        this.sites = [];
        this.availableCerts = []; // 证书库中可用的证书列表
        this.editingSite = null;
        this.deleteTargetId = null;
        this._sslTab = 'off'; // 当前 SSL 子选项卡: off | pool | paste
        this._pasteCertInfo = null; // 粘贴证书解析后的信息
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
                title: 'SSL',
                dataIndex: 'ssl',
                className: 'col-ssl',
                render: (value) => {
                    if (!value || !value.enabled) {
                        return '<span class="ssl-badge off">关</span>';
                    }
                    const certDomain = value._cert_domain || '';
                    const daysLeft = value._days_left;
                    if (daysLeft !== undefined && daysLeft !== null) {
                        let cls = 'valid';
                        if (daysLeft <= 0) cls = 'expired';
                        else if (daysLeft <= 30) cls = 'expiring';
                        return `<span class="ssl-badge ${cls}" title="${certDomain ? '证书: ' + escapeHtml(certDomain) : ''}">${daysLeft > 0 ? daysLeft + '天' : '已过期'}</span>`;
                    }
                    return '<span class="ssl-badge on">开</span>';
                }
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
                    let url = '';
                    const useHttps = row.ssl?.enabled;
                    const scheme = useHttps ? 'https' : 'http';
                    if (row.domain?.length > 0) {
                        const port = row.port > 0 && row.port !== 80 && row.port !== 443 ? `:${row.port}` : '';
                        url = `${scheme}://${row.domain[0]}${port}`;
                    } else if (row.port > 0) {
                        url = `${scheme}://${window.location.hostname}:${row.port}`;
                    }
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
                    const target = row.proxy?.target || '';
                    if (target) {
                        return `<div class="root-path"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="icon"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg><a class="root-link" href="${escapeHtml(target)}" target="_blank" title="打开代理目标">${escapeHtml(target)}</a></div>`;
                    }
                    return `<div class="root-path"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="icon"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg><span class="text-muted">-</span></div>`;
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

        // 弹窗
        this.bindModalClose('site-modal', () => this.hideEditor(), {
            confirmId: 'site-modal-confirm',
            onConfirm: () => this.saveSite()
        });
        this.$('#site-form-type')?.addEventListener('change', () => this.onTypeChange());

        // SSL 子选项卡
        this.$$('#site-ssl-tabs .ssl-tab-btn').forEach(btn => {
            btn.addEventListener('click', () => this.switchSSLTab(btn.dataset.sslTab));
        });

        // SSL 证书选择
        this.$('#site-form-ssl-cert-select')?.addEventListener('change', () => this.onCertSelect());

        // SSL 粘贴证书验证
        this.$('#site-form-ssl-paste-verify')?.addEventListener('click', () => this.verifyAndPasteCert());

        // 目录浏览按钮
        this.$$('.directory-picker-btn').forEach(btn => {
            btn.addEventListener('click', () => openDirPicker(btn.dataset.dir, this.api));
        });
        // 删除确认弹窗
        this.bindModalClose('site-delete-modal', () => this.hideDeleteConfirm(), {
            confirmId: 'site-delete-confirm',
            onConfirm: () => this.confirmDelete()
        });
    }

    bindRowEvents() {
        // 根目录链接（跳转文件管理）
        this.bindBrowseLinks();

        // 状态按钮
        this.$$('.status-btn[data-id]').forEach(btn => {
            btn.addEventListener('click', () => {
                const id = btn.dataset.id;
                const isActive = btn.classList.contains('active');
                this.toggleStatus(id, !isActive);
            });
        });

        // 复制链接
        this.bindCopyLinks();

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
            this.enrichSitesWithCertInfo();
            this.filterSites();
        } finally {
            this.dataTable.setLoading(false);
        }
    }

    // 用证书库信息丰富站点的 SSL 状态
    enrichSitesWithCertInfo() {
        for (const site of this.sites) {
            if (!site.ssl?.enabled || !site.domain?.length) continue;
            // 查找证书库中匹配的证书
            for (const cert of this.availableCerts) {
                if (site.domain.includes(cert.domain)) {
                    site.ssl._cert_domain = cert.domain;
                    site.ssl._days_left = cert.days_left;
                    site.ssl._has_cert = cert.has_cert;
                    site.ssl._issuer = cert.issuer;
                    site.ssl._not_after = cert.not_after;
                    break;
                }
            }
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

    onInit() {
        this._svc = this.createServiceControls({
            apiPrefix: '/api/service/sites',
            statusId: 'site-service-status',
            toggleId: 'site-service-toggle',
            label: '站点服务',
        });
        this.initDataTable();
        this.bindEvents();
        this.checkServiceStatus();
    }

    async checkServiceStatus() {
        try {
            const data = await this.api.getJSON('/api/sites/status');
            if (data) {
                this._svc.updateServiceStatus(data.running);
            }
        } catch (error) {
            console.error('[Sites] 状态检查失败:', error);
        }
    }

    toggleService() { return this._svc.toggleService(); }
    restartService() { return this._svc.restartService(); }
    async reloadConfig() { await this._svc.reloadConfig(); await this.refresh(); }

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

    // ========== SSL 证书管理 ==========

    // 加载可用证书列表
    async loadAvailableCerts() {
        try {
            this.availableCerts = await this.api.getJSON('/api/certs', { cache: false }) || [];
        } catch (error) {
            this.availableCerts = [];
        }
    }

    // 填充证书选择下拉框
    populateCertSelect() {
        const select = this.$('#site-form-ssl-cert-select');
        if (!select) return;

        // 保留第一个默认选项
        select.innerHTML = '<option value="">-- 请选择证书 --</option>';

        for (const cert of this.availableCerts) {
            const opt = document.createElement('option');
            opt.value = cert.domain;
            let label = cert.domain;
            if (cert.days_left !== undefined && cert.days_left !== null) {
                if (cert.days_left <= 0) {
                    label += ' (已过期)';
                } else if (cert.days_left <= 30) {
                    label += ` (剩余${cert.days_left}天)`;
                } else {
                    label += ` (${cert.days_left}天)`;
                }
            }
            if (cert.issuer) {
                label += ` - ${cert.issuer}`;
            }
            opt.textContent = label;
            select.appendChild(opt);
        }

        // 显示/隐藏空提示
        const emptyHint = this.$('#site-ssl-cert-empty');
        if (emptyHint) {
            emptyHint.style.display = this.availableCerts.length === 0 ? 'flex' : 'none';
        }
    }

    // 证书选择变更 - 显示证书预览
    onCertSelect() {
        const domain = this.$('#site-form-ssl-cert-select')?.value;
        const preview = this.$('#site-ssl-cert-preview');
        if (!domain || !preview) {
            if (preview) preview.style.display = 'none';
            return;
        }

        const cert = this.availableCerts.find(c => c.domain === domain);
        if (!cert) {
            preview.style.display = 'none';
            return;
        }

        preview.style.display = 'block';
        this.setText('#cert-preview-domain', cert.domain || '-');
        this.setText('#cert-preview-issuer', cert.issuer || '-');

        if (cert.not_before && cert.not_after) {
            const from = new Date(cert.not_before).toLocaleDateString();
            const to = new Date(cert.not_after).toLocaleDateString();
            this.setText('#cert-preview-validity', `${from} ~ ${to}`);
        } else {
            this.setText('#cert-preview-validity', '-');
        }

        if (cert.days_left !== undefined && cert.days_left !== null) {
            const daysEl = this.$('#cert-preview-days');
            if (daysEl) {
                daysEl.textContent = cert.days_left > 0 ? `${cert.days_left} 天` : '已过期';
                daysEl.className = 'cert-preview-value ' + (cert.days_left <= 0 ? 'text-danger' : cert.days_left <= 30 ? 'text-warning' : 'text-success');
            }
        } else {
            this.setText('#cert-preview-days', '-');
        }
    }

    // ========== SSL 子选项卡 ==========

    // 切换 SSL 子选项卡
    switchSSLTab(tab) {
        this._sslTab = tab;

        // 更新按钮状态
        this.$$('#site-ssl-tabs .ssl-tab-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.sslTab === tab);
        });

        // 更新面板显示
        this.$$('.ssl-tab-panel').forEach(panel => {
            panel.classList.toggle('active', panel.dataset.sslPanel === tab);
        });

        // 显示/隐藏通用选项
        const commonOpts = this.$('#site-ssl-common-options');
        if (commonOpts) {
            commonOpts.style.display = tab !== 'off' ? 'block' : 'none';
        }
    }

    // 验证并保存粘贴的证书
    async verifyAndPasteCert() {
        const certPEM = this.$('#site-form-ssl-paste-cert')?.value?.trim();
        const keyPEM = this.$('#site-form-ssl-paste-key')?.value?.trim();
        const statusEl = this.$('#site-ssl-paste-status');
        const previewEl = this.$('#site-ssl-paste-preview');

        if (!certPEM || !keyPEM) {
            if (statusEl) statusEl.textContent = '请填写证书和私钥';
            return;
        }

        // 检查 PEM 格式
        if (!certPEM.includes('-----BEGIN')) {
            if (statusEl) statusEl.textContent = '证书格式错误，需要 PEM 格式';
            return;
        }
        if (!keyPEM.includes('-----BEGIN')) {
            if (statusEl) statusEl.textContent = '私钥格式错误，需要 PEM 格式';
            return;
        }

        // 从证书提取域名
        const domain = this.extractDomainFromPEM(certPEM);
        if (!domain) {
            if (statusEl) statusEl.textContent = '无法解析证书域名';
            return;
        }

        if (statusEl) statusEl.textContent = '验证中...';

        try {
            const result = await this.api.postJSON('/api/certs/paste', {
                domain,
                cert_pem: certPEM,
                key_pem: keyPEM,
            });

            this._pasteCertInfo = result;
            if (statusEl) {
                statusEl.innerHTML = '<span style="color:var(--success)">验证通过</span>';
            }

            // 显示证书预览
            if (previewEl && result) {
                previewEl.style.display = 'block';
                this.setText('#paste-preview-domain', result.domain || domain);
                this.setText('#paste-preview-issuer', result.issuer || '-');

                if (result.not_before && result.not_after) {
                    const from = new Date(result.not_before).toLocaleDateString();
                    const to = new Date(result.not_after).toLocaleDateString();
                    this.setText('#paste-preview-validity', `${from} ~ ${to}`);
                } else {
                    this.setText('#paste-preview-validity', '-');
                }

                const daysEl = this.$('#paste-preview-days');
                if (daysEl && result.days_left !== undefined) {
                    daysEl.textContent = result.days_left > 0 ? `${result.days_left} 天` : '已过期';
                    daysEl.className = 'cert-preview-value ' + (result.days_left <= 0 ? 'text-danger' : result.days_left <= 30 ? 'text-warning' : 'text-success');
                }
            }

            // 刷新证书库列表
            await this.loadAvailableCerts();
            this.populateCertSelect();
        } catch (error) {
            if (statusEl) {
                statusEl.innerHTML = `<span style="color:var(--danger)">验证失败: ${escapeHtml(error.message)}</span>`;
            }
            if (previewEl) previewEl.style.display = 'none';
            this._pasteCertInfo = null;
        }
    }

    // 从 PEM 证书中提取域名（简单解析 Common Name / SAN）
    extractDomainFromPEM(pem) {
        // 尝试从站点域名匹配（优先使用表单中的域名）
        const domainStr = this.$('#site-form-domain')?.value?.trim();
        if (domainStr) {
            const domains = domainStr.split(',').map(d => d.trim()).filter(Boolean);
            if (domains.length > 0) return domains[0];
        }
        // 回退：从 PEM 中提取 CN
        const cnMatch = pem.match(/CN\s*=\s*([^\s,]+)/);
        if (cnMatch) return cnMatch[1].replace(/^\*\./, ''); // 去掉通配符前缀
        return '';
    }

    // ========== 编辑器 ==========

    async showEditor() {
        const modal = this.$('#site-modal');
        const title = this.$('#site-modal-title');
        if (!modal) return;

        // 清理状态
        this._pasteCertInfo = null;

        // 加载可用证书
        await this.loadAvailableCerts();
        this.populateCertSelect();

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

            // SSL 字段
            const ssl = this.editingSite.ssl || {};

            // 判断 SSL 状态，选择对应的子选项卡
            let sslTab = 'off';
            if (ssl.enabled) {
                if (ssl.cert_file && ssl.key_file) {
                    // 有证书路径 — 检查是否在证书库中
                    const matchedCert = this.availableCerts.find(c =>
                        this.editingSite.domain?.includes(c.domain)
                    );
                    sslTab = matchedCert ? 'pool' : 'paste';
                } else {
                    sslTab = 'pool';
                }
            }

            this.switchSSLTab(sslTab);

            // 如果启用 SSL，自动选中匹配的证书
            if (sslTab !== 'off') {
                const certSelect = this.$('#site-form-ssl-cert-select');
                if (certSelect) {
                    const matchedCert = this.availableCerts.find(c =>
                        this.editingSite.domain?.includes(c.domain)
                    );
                    if (matchedCert) {
                        certSelect.value = matchedCert.domain;
                        this.onCertSelect();
                    }
                }
            }

            const sslForce = this.$('#site-form-ssl-force');
            if (sslForce) sslForce.checked = ssl.force_https || false;
            const sslHsts = this.$('#site-form-ssl-hsts');
            if (sslHsts) sslHsts.checked = ssl.hsts || false;
        } else {
            title.textContent = '添加网站';
            this.$('#site-form')?.reset();
            this.$('#site-form-root').value = './sites/default';
            // 新建默认值
            this.$('#site-form-index-files').value = 'index.html, index.htm';
            const autoIndex = this.$('#site-form-auto-index');
            if (autoIndex) autoIndex.checked = true;
            // 默认关闭 SSL
            this.switchSSLTab('off');
        }

        // 清空粘贴区域
        const pasteCert = this.$('#site-form-ssl-paste-cert');
        const pasteKey = this.$('#site-form-ssl-paste-key');
        if (pasteCert) pasteCert.value = '';
        if (pasteKey) pasteKey.value = '';
        const pastePreview = this.$('#site-ssl-paste-preview');
        if (pastePreview) pastePreview.style.display = 'none';
        const pasteStatus = this.$('#site-ssl-paste-status');
        if (pasteStatus) pasteStatus.textContent = '';

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

    // ========== 保存站点 ==========

    // 根据 SSL 子选项卡构建 SSL 配置
    buildSSLConfig() {
        const sslTab = this._sslTab;
        const forceHTTPS = this.$('#site-form-ssl-force')?.checked || false;
        const hsts = this.$('#site-form-ssl-hsts')?.checked || false;

        if (sslTab === 'off') {
            return { enabled: false };
        }

        if (sslTab === 'pool') {
            // 从证书库选择
            const selectedCertDomain = this.$('#site-form-ssl-cert-select')?.value;
            if (!selectedCertDomain) {
                // 未选择证书，视为关闭
                return { enabled: false };
            }
            const cert = this.availableCerts.find(c => c.domain === selectedCertDomain);
            return {
                enabled: true,
                auto_https: false,
                cert_file: cert?.cert_file || `./ssl/${selectedCertDomain}.crt`,
                key_file: cert?.key_file || `./ssl/${selectedCertDomain}.key`,
                force_https: forceHTTPS,
                hsts,
            };
        }

        if (sslTab === 'paste') {
            // 粘贴证书 — 证书已通过 verifyAndPasteCert 保存
            if (!this._pasteCertInfo) {
                // 未验证粘贴证书，视为关闭
                return { enabled: false };
            }
            return {
                enabled: true,
                auto_https: false,
                cert_file: `./ssl/${this._pasteCertInfo.domain}.crt`,
                key_file: `./ssl/${this._pasteCertInfo.domain}.key`,
                force_https: forceHTTPS,
                hsts,
            };
        }

        if (sslTab === 'auto') {
            // 自动申请（已移除，降级为证书库模式）
            return { enabled: false };
        }

        return { enabled: false };
    }

    // 保存站点（点击"确定"按钮）
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

        // SSL 配置
        const sslTab = this._sslTab;
        if (sslTab === 'pool') {
            const selectedCertDomain = this.$('#site-form-ssl-cert-select')?.value;
            if (!selectedCertDomain) {
                this.message.error('请选择一个证书');
                return;
            }
        } else if (sslTab === 'paste' && !this._pasteCertInfo) {
            this.message.error('请先验证并保存粘贴的证书');
            return;
        }

        data.ssl = this.buildSSLConfig();

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
