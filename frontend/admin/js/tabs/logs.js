/**
 * 日志管理模块 v5
 *
 * 面板日志：操作日志 / 登录日志 / 运行日志（子标签切换）
 * HTTP/FTP：统一 access.log + error.log，下拉框按站点/用户筛选
 */

import { BaseTab } from './BaseTab.js';
import { escapeHtml } from '../core/utils.js';

function localDateStr(offsetDays = 0) {
    const d = new Date();
    if (offsetDays) d.setDate(d.getDate() + offsetDays);
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

const ICONS = {
    panel: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="9" y1="21" x2="9" y2="9"/></svg>`,
    http: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>`,
    ftp: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>`,
    refresh: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>`,
    download: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>`,
    trash: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>`,
    settings: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`
};

const CATEGORIES = [
    { id: 'panel', label: '面板日志', icon: ICONS.panel },
    { id: 'http', label: 'HTTP日志', icon: ICONS.http },
    { id: 'ftp', label: 'FTP日志', icon: ICONS.ftp }
];

const PANEL_SUB_TYPES = [
    { id: 'operation', label: '操作日志' },
    { id: 'login', label: '登录日志' },
    { id: 'runtime', label: '运行日志' }
];

const OP_TYPE_OPTIONS = [
    { value: '', label: '全部操作' },
    { value: '[站点]', label: '站点操作' },
    { value: '[FTP]', label: 'FTP操作' },
    { value: '[文件]', label: '文件操作' },
    { value: '[配置]', label: '配置操作' },
    { value: '[服务]', label: '服务控制' }
];

const NET_SUB_TYPES = [
    { id: 'access', label: '访问日志' },
    { id: 'error', label: '错误日志' }
];

class LogsTab extends BaseTab {
    constructor(deps) {
        super(deps, 'logs');
        this.category = 'panel';
        this.type = 'operation';
        this.date = localDateStr();
        this.logFiles = [];
        this.sites = [];
        this.ftpUsers = [];
        this.showSettings = false;
        this.logConfig = null;
    }

    onInit() {
        this.initUI();
        this.bindEvents();
    }

    async onLoad() {
        await Promise.all([
            this.loadFiles(),
            this.loadSites(),
            this.loadFTPUsers(),
            this.loadLogConfig()
        ]);
        this.updateSubTabs();
        this.updateFilterVisibility();
        await this.refresh();
    }

    // ========== UI ==========

    initUI() {
        const container = this.$('#logs-container');
        if (!container) return;

        container.innerHTML = `
            <div class="logs-tabs">
                ${CATEGORIES.map(c => `
                    <button class="logs-tab ${c.id === this.category ? 'active' : ''}" data-category="${c.id}">
                        <span class="logs-tab-icon">${c.icon}</span>
                        <span class="logs-tab-label">${c.label}</span>
                    </button>
                `).join('')}
            </div>
            <div class="logs-subtabs" id="logs-subtabs"></div>
            <div class="logs-toolbar">
                <div class="logs-toolbar-left">
                    <select id="log-date" class="form-select-sm"></select>
                    <select id="log-filter" class="form-select-sm" style="display:none"></select>
                    <input type="text" id="log-search" class="form-input-sm" placeholder="搜索关键词...">
                </div>
                <div class="logs-toolbar-right">
                    <button id="log-refresh" class="btn btn-sm btn-secondary">${ICONS.refresh} 刷新</button>
                    <button id="log-download" class="btn btn-sm btn-secondary">${ICONS.download} 导出</button>
                    <button id="log-clear" class="btn btn-sm btn-danger">${ICONS.trash} 清空</button>
                    <button id="log-settings" class="btn btn-sm btn-secondary">${ICONS.settings}</button>
                </div>
            </div>
            <div class="logs-stats" id="logs-stats">
                <div class="stat-card"><div class="stat-value" id="stat-count">-</div><div class="stat-label">今日记录</div></div>
                <div class="stat-card"><div class="stat-value stat-error" id="stat-errors">-</div><div class="stat-label">错误</div></div>
                <div class="stat-card"><div class="stat-value stat-warning" id="stat-warnings">-</div><div class="stat-label">警告</div></div>
            </div>
            <div class="logs-viewer-container">
                <div class="logs-viewer" id="log-viewer"><div class="logs-loading">加载中...</div></div>
            </div>
            <div class="logs-pagination"><span id="log-count">共 0 条</span></div>
            <div class="logs-settings" id="logs-settings-panel" style="display:none">
                <div class="logs-settings-title">日志轮转设置</div>
                <div class="logs-settings-grid">
                    <div class="logs-settings-field">
                        <label>保留天数</label>
                        <input type="number" id="log-cfg-retention" class="form-input-sm" min="1" max="365" value="30">
                    </div>
                    <div class="logs-settings-field">
                        <label>单文件上限 (MB)</label>
                        <input type="number" id="log-cfg-max-size" class="form-input-sm" min="1" max="1024" value="100">
                    </div>
                    <div class="logs-settings-field">
                        <label>压缩天数</label>
                        <input type="number" id="log-cfg-compress-days" class="form-input-sm" min="1" max="90" value="7">
                    </div>
                </div>
                <div class="logs-settings-actions">
                    <button id="log-cfg-save" class="btn btn-sm btn-primary">保存设置</button>
                </div>
            </div>
        `;

        this.updateSubTabs();
    }

    updateSubTabs() {
        const container = this.$('#logs-subtabs');
        if (!container) return;

        let types;
        if (this.category === 'panel') {
            types = PANEL_SUB_TYPES;
        } else {
            // HTTP / FTP: access + error
            types = NET_SUB_TYPES;
        }

        if (!types.find(t => t.id === this.type)) {
            this.type = types[0].id;
        }

        container.innerHTML = types.map(t => `
            <button class="logs-subtab ${t.id === this.type ? 'active' : ''}" data-type="${t.id}">${t.label}</button>
        `).join('');

        container.querySelectorAll('.logs-subtab').forEach(btn => {
            btn.addEventListener('click', () => {
                this.type = btn.dataset.type;
                container.querySelectorAll('.logs-subtab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                const filterEl = this.$('#log-filter');
                if (filterEl) filterEl.value = '';
                this.updateFilterVisibility();
                this.refresh();
            });
        });
    }

    updateFilterVisibility() {
        const filterEl = this.$('#log-filter');
        if (!filterEl) return;

        if (this.category === 'panel') {
            // 面板：仅操作日志显示操作类型筛选
            if (this.type === 'operation') {
                filterEl.style.display = '';
                filterEl.innerHTML = OP_TYPE_OPTIONS.map(o => `<option value="${o.value}">${o.label}</option>`).join('');
            } else {
                filterEl.style.display = 'none';
                filterEl.value = '';
            }
            if (!PANEL_SUB_TYPES.find(t => t.id === this.type)) {
                this.type = 'operation';
            }
        } else if (this.category === 'http') {
            // HTTP：按站点筛选（仅 access.log 有站点标记）
            const options = [{ value: '', label: '全部站点' }];
            this.sites.forEach(s => options.push({ value: `[${s.id}]`, label: s.name }));
            filterEl.style.display = this.type === 'access' && options.length > 1 ? '' : 'none';
            filterEl.innerHTML = options.map(o => `<option value="${o.value}">${o.label}</option>`).join('');
            if (!NET_SUB_TYPES.find(t => t.id === this.type)) {
                this.type = 'access';
            }
        } else if (this.category === 'ftp') {
            // FTP：按用户筛选（仅 access.log 有用户标记）
            const options = [{ value: '', label: '全部用户' }];
            this.ftpUsers.forEach(u => options.push({ value: `[${u.username}]`, label: u.username }));
            filterEl.style.display = this.type === 'access' && options.length > 1 ? '' : 'none';
            filterEl.innerHTML = options.map(o => `<option value="${o.value}">${o.label}</option>`).join('');
            if (!NET_SUB_TYPES.find(t => t.id === this.type)) {
                this.type = 'access';
            }
        }
    }

    bindEvents() {
        // 分类切换
        this.$$('.logs-tab').forEach(btn => {
            btn.addEventListener('click', () => {
                this.category = btn.dataset.category;
                this.$$('.logs-tab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                this.date = localDateStr();
                this.updateSubTabs();
                this.updateFilterVisibility();
                this.updateDateOptions();
                this.refresh();
            });
        });

        // 过滤
        this.$('#log-date')?.addEventListener('change', (e) => { this.date = e.target.value; this.refresh(); });
        this.$('#log-filter')?.addEventListener('change', () => this.refresh());

        let searchTimer;
        this.$('#log-search')?.addEventListener('input', () => {
            clearTimeout(searchTimer);
            searchTimer = setTimeout(() => this.refresh(), 300);
        });

        // 按钮
        this.$('#log-refresh')?.addEventListener('click', () => this.refresh());
        this.$('#log-download')?.addEventListener('click', () => this.exportLogs());
        this.$('#log-clear')?.addEventListener('click', () => this.confirmClear());
        this.$('#log-settings')?.addEventListener('click', () => this.toggleSettings());
        this.$('#log-cfg-save')?.addEventListener('click', () => this.saveLogConfig());
    }

    toggleSettings() {
        const panel = this.$('#logs-settings-panel');
        if (!panel) return;
        this.showSettings = !this.showSettings;
        panel.style.display = this.showSettings ? '' : 'none';
    }

    // ========== 数据加载 ==========

    async refresh() {
        await Promise.all([this.loadLogs(), this.loadStats()]);
    }

    buildParams() {
        const search = this.$('#log-search')?.value || '';
        const filterVal = this.$('#log-filter')?.value || '';

        let finalSearch = search;
        if (filterVal && search) {
            finalSearch = filterVal + ' ' + search;
        } else if (filterVal) {
            finalSearch = filterVal;
        }

        return new URLSearchParams({
            category: this.category,
            type: this.type,
            date: this.date,
            search: finalSearch
        });
    }

    async loadFiles() {
        try {
            this.logFiles = await this.api.getJSON('/api/logs') || [];
            this.updateDateOptions();
        } catch (_) {}
    }

    async loadSites() {
        try { this.sites = await this.api.getJSON('/api/sites') || []; }
        catch (_) { this.sites = []; }
    }

    async loadFTPUsers() {
        try {
            const data = await this.api.getJSON('/api/ftp/users') || {};
            this.ftpUsers = data.users || data || [];
        } catch (_) { this.ftpUsers = []; }
    }

    async loadLogConfig() {
        try {
            this.logConfig = await this.api.getJSON('/api/logs/config');
            if (this.logConfig) {
                const el = (id) => this.$(id);
                if (el('#log-cfg-retention')) el('#log-cfg-retention').value = this.logConfig.retention_days || 30;
                if (el('#log-cfg-max-size')) el('#log-cfg-max-size').value = this.logConfig.max_size_mb || 100;
                if (el('#log-cfg-compress-days')) el('#log-cfg-compress-days').value = this.logConfig.compress_days || 7;
            }
        } catch (_) {}
    }

    updateDateOptions() {
        const select = this.$('#log-date');
        if (!select) return;

        const today = localDateStr();
        const dates = new Set([today]);

        this.logFiles.forEach(f => {
            if (f.category === this.category && !f.compressed) {
                const match = f.name.match(/(\d{4}-\d{2}-\d{2})/);
                if (match && match[1] !== today) dates.add(match[1]);
            }
        });

        if (dates.size <= 1) {
            for (let i = 1; i <= 7; i++) {
                dates.add(localDateStr(-i));
            }
        }

        select.innerHTML = `<option value="${today}">今天</option>` +
            Array.from(dates).filter(d => d !== today).sort().reverse()
                .map(d => `<option value="${d}">${d}</option>`).join('');
    }

    async loadLogs() {
        const viewer = this.$('#log-viewer');
        if (!viewer) return;

        viewer.innerHTML = '<div class="logs-loading">加载中...</div>';

        try {
            const params = this.buildParams();
            params.set('limit', '200');
            const data = await this.api.getJSON(`/api/logs/read?${params}`);

            if (data?.entries?.length) {
                this.renderLogs(data.entries);
                this.$('#log-count').textContent = `共 ${data.total} 条`;
            } else {
                viewer.innerHTML = '<div class="logs-empty">暂无日志</div>';
                this.$('#log-count').textContent = '共 0 条';
            }
        } catch (error) {
            viewer.innerHTML = `<div class="logs-error">加载失败: ${error.message}</div>`;
        }
    }

    async loadStats() {
        try {
            const data = await this.api.getJSON(`/api/logs/stats?${this.buildParams()}`) || [];
            let total = 0, errors = 0, warnings = 0;
            data.forEach(s => { total += s.count; errors += s.errors; warnings += s.warnings; });
            this.setText('#stat-count', total);
            this.setText('#stat-errors', errors);
            this.setText('#stat-warnings', warnings);
        } catch (_) {}
    }

    renderLogs(entries) {
        const viewer = this.$('#log-viewer');
        if (!viewer) return;

        viewer.innerHTML = entries.map(e => {
            const cls = e.level ? `log-${e.level}` : '';
            return `<div class="log-entry ${cls}">
                <span class="log-time">${e.timestamp || ''}</span>
                <span class="log-level">${e.level ? `[${e.level.toUpperCase()}]` : ''}</span>
                <span class="log-message">${escapeHtml(e.message || e.raw)}</span>
            </div>`;
        }).join('');

        viewer.scrollTop = viewer.scrollHeight;
    }

    // ========== 操作 ==========

    exportLogs() {
        const base = window.location.pathname.replace(/\/(index\.html)?$/, '') || '';
        window.open(`${base}/api/logs/download?${this.buildParams()}`, '_blank');
    }

    confirmClear() {
        const label = CATEGORIES.find(c => c.id === this.category)?.label || this.category;
        this.dialog.confirm(`确定要清空【${label}】当前日志吗？此操作不可恢复。`, async () => {
            try {
                await this.api.post(`/api/logs/clear?category=${this.category}&type=${this.type}`);
                this.message.success('日志已清空');
                await this.refresh();
            } catch (error) {
                this.message.error('清空失败: ' + error.message);
            }
        });
    }

    async saveLogConfig() {
        try {
            const config = {
                retention_days: parseInt(this.$('#log-cfg-retention')?.value) || 30,
                max_size_mb: parseInt(this.$('#log-cfg-max-size')?.value) || 100,
                compress_days: parseInt(this.$('#log-cfg-compress-days')?.value) || 7
            };
            await this.api.post('/api/logs/config', config);
            this.message.success('日志设置已保存');
        } catch (error) {
            this.message.error('保存失败: ' + error.message);
        }
    }
}

let instance = null;

export function initLogsTab(deps) {
    if (!instance) {
        instance = new LogsTab(deps);
        instance.init();
    }
    return instance;
}
