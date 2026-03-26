/**
 * 日志管理模块
 *
 * 支持日志查看、搜索、统计、下载
 */

import { BaseTab } from './BaseTab.js';
import { escapeHtml } from '../core/utils.js';

// 图标
const ICONS = {
    http: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>`,
    folder: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>`,
    settings: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`,
    refresh: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>`,
    download: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>`,
    trash: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>`
};

const CATEGORIES = [
    { id: 'http', label: 'HTTP', icon: ICONS.http },
    { id: 'ftp', label: 'FTP', icon: ICONS.folder },
    { id: 'panel', label: '面板', icon: ICONS.settings }
];

const TYPES = {
    http: [{ id: 'access', label: '访问日志' }, { id: 'error', label: '错误日志' }],
    ftp: [{ id: 'access', label: '访问日志' }, { id: 'error', label: '错误日志' }],
    panel: [{ id: 'access', label: '访问日志' }, { id: 'api', label: 'API 日志' }, { id: 'auth', label: '认证日志' }]
};

class LogsTab extends BaseTab {
    constructor(deps) {
        super(deps, 'logs');
        this.category = 'http';
        this.type = 'access';
        this.date = '';
        this.autoRefresh = true;
        this.refreshTimer = null;
        this.logFiles = [];
    }

    onInit() {
        console.log('初始化日志管理面板...');
        this.initUI();
        this.bindEvents();
    }

    async onLoad() {
        await this.loadFiles();
        await this.loadLogs();
        await this.loadStats();
        if (this.autoRefresh) this.startAutoRefresh();
    }

    onDestroy() {
        this.stopAutoRefresh();
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
                    <select id="log-date" class="form-select-sm"><option value="">今天</option></select>
                    <select id="log-level" class="form-select-sm">
                        <option value="">全部级别</option>
                        <option value="error">错误</option>
                        <option value="warn">警告</option>
                        <option value="info">信息</option>
                    </select>
                    <input type="text" id="log-search" class="form-input-sm" placeholder="搜索关键词...">
                </div>
                <div class="logs-toolbar-right">
                    <label class="checkbox-label"><input type="checkbox" id="log-auto-refresh" checked>自动刷新</label>
                    <button id="log-refresh" class="btn btn-sm btn-secondary">${ICONS.refresh} 刷新</button>
                    <button id="log-download" class="btn btn-sm btn-secondary">${ICONS.download} 下载</button>
                    <button id="log-clear" class="btn btn-sm btn-danger">${ICONS.trash} 清空</button>
                </div>
            </div>
            <div class="logs-stats" id="logs-stats">
                <div class="stat-card"><div class="stat-value" id="stat-count">-</div><div class="stat-label">今日请求</div></div>
                <div class="stat-card"><div class="stat-value stat-error" id="stat-errors">-</div><div class="stat-label">错误</div></div>
                <div class="stat-card"><div class="stat-value stat-warning" id="stat-warnings">-</div><div class="stat-label">警告</div></div>
            </div>
            <div class="logs-viewer-container">
                <div class="logs-viewer" id="log-viewer"><div class="logs-loading">加载中...</div></div>
                <label class="checkbox-label auto-scroll-label"><input type="checkbox" id="log-auto-scroll" checked>自动滚动</label>
            </div>
            <div class="logs-pagination"><span id="log-count">共 0 条</span></div>
        `;

        this.updateSubTabs();
    }

    updateSubTabs() {
        const container = this.$('#logs-subtabs');
        if (!container) return;

        const types = TYPES[this.category] || [];
        container.innerHTML = types.map(t => `
            <button class="logs-subtab ${t.id === this.type ? 'active' : ''}" data-type="${t.id}">${t.label}</button>
        `).join('');

        container.querySelectorAll('.logs-subtab').forEach(btn => {
            btn.addEventListener('click', () => {
                this.type = btn.dataset.type;
                container.querySelectorAll('.logs-subtab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                this.refresh();
            });
        });
    }

    bindEvents() {
        // 分类切换
        this.$$('.logs-tab').forEach(btn => {
            btn.addEventListener('click', () => {
                this.category = btn.dataset.category;
                this.$$('.logs-tab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                this.type = TYPES[this.category]?.[0]?.id || 'access';
                this.updateSubTabs();
                this.refresh();
            });
        });

        // 过滤
        this.$('#log-date')?.addEventListener('change', (e) => { this.date = e.target.value; this.loadLogs(); });
        this.$('#log-level')?.addEventListener('change', () => this.loadLogs());
        
        let searchTimer;
        this.$('#log-search')?.addEventListener('input', () => {
            clearTimeout(searchTimer);
            searchTimer = setTimeout(() => this.loadLogs(), 300);
        });

        // 自动刷新
        this.$('#log-auto-refresh')?.addEventListener('change', (e) => {
            this.autoRefresh = e.target.checked;
            this.autoRefresh ? this.startAutoRefresh() : this.stopAutoRefresh();
        });

        // 按钮
        this.$('#log-refresh')?.addEventListener('click', () => this.loadLogs());
        this.$('#log-download')?.addEventListener('click', () => this.download());
        this.$('#log-clear')?.addEventListener('click', () => this.confirmClear());
    }

    // ========== 数据加载 ==========

    async loadFiles() {
        try {
            this.logFiles = await this.api.getJSON('/api/logs') || [];
            this.updateDateOptions();
        } catch (error) {
            console.error('加载日志文件列表失败:', error);
        }
    }

    updateDateOptions() {
        const select = this.$('#log-date');
        if (!select) return;

        const dates = new Set(['']);
        const today = new Date().toISOString().split('T')[0];

        this.logFiles.forEach(f => {
            if (f.category === this.category && !f.compressed) {
                const match = f.name.match(/(\d{4}-\d{2}-\d{2})/);
                if (match && match[1] !== today) dates.add(match[1]);
            }
        });

        select.innerHTML = Array.from(dates).sort().reverse()
            .map(d => `<option value="${d}">${d || '今天'}</option>`).join('');
    }

    async loadLogs() {
        const viewer = this.$('#log-viewer');
        if (!viewer) return;

        const search = this.$('#log-search')?.value || '';
        const level = this.$('#log-level')?.value || '';

        viewer.innerHTML = '<div class="logs-loading">加载中...</div>';

        try {
            const params = new URLSearchParams({ category: this.category, type: this.type, search, level, limit: '100' });
            if (this.date) params.set('date', this.date);

            const data = await this.api.getJSON(`/api/logs/read?${params}`);
            if (data?.entries) {
                this.renderLogs(data.entries);
                this.$('#log-count').textContent = `共 ${data.total} 条`;
            } else {
                viewer.innerHTML = '<div class="logs-empty">暂无日志</div>';
            }
        } catch (error) {
            viewer.innerHTML = `<div class="logs-error">加载失败: ${error.message}</div>`;
        }
    }

    async loadStats() {
        try {
            const data = await this.api.getJSON(`/api/logs/stats?category=${this.category}`) || [];
            let total = 0, errors = 0, warnings = 0;
            data.forEach(s => { total += s.count; errors += s.errors; warnings += s.warnings; });
            this.setText('#stat-count', total);
            this.setText('#stat-errors', errors);
            this.setText('#stat-warnings', warnings);
        } catch (error) {
            console.error('加载统计失败:', error);
        }
    }

    renderLogs(entries) {
        const viewer = this.$('#log-viewer');
        if (!viewer || !entries?.length) {
            viewer.innerHTML = '<div class="logs-empty">暂无日志</div>';
            return;
        }

        viewer.innerHTML = entries.map(e => {
            const cls = e.level ? `log-${e.level}` : '';
            return `<div class="log-entry ${cls}">
                <span class="log-time">${e.timestamp || ''}</span>
                <span class="log-level">${e.level ? `[${e.level.toUpperCase()}]` : ''}</span>
                <span class="log-message">${escapeHtml(e.message || e.raw)}</span>
            </div>`;
        }).join('');

        if (this.$('#log-auto-scroll')?.checked) {
            viewer.scrollTop = viewer.scrollHeight;
        }
    }

    // ========== 操作 ==========

    download() {
        const params = new URLSearchParams({
            category: this.category,
            file: this.date ? `${this.type}.${this.date}.log` : `${this.type}.log`
        });
        window.open(`/api/logs/download?${params}`, '_blank');
    }

    confirmClear() {
        const label = `${this.category.toUpperCase()} - ${this.type}`;
        this.dialog.confirm(`确定要清空【${label}】吗？此操作不可恢复。`, async () => {
            try {
                await this.api.post(`/api/logs/clear?category=${this.category}&type=${this.type}`);
                this.message.success('日志已清空');
                await this.refresh();
            } catch (error) {
                this.message.error('清空失败: ' + error.message);
            }
        });
    }

    // ========== 自动刷新 ==========

    startAutoRefresh() {
        this.stopAutoRefresh();
        this.refreshTimer = setInterval(() => {
            if (document.visibilityState === 'visible') this.loadLogs();
        }, 5000);
    }

    stopAutoRefresh() {
        if (this.refreshTimer) {
            clearInterval(this.refreshTimer);
            this.refreshTimer = null;
        }
    }
}

// 单例
let instance = null;

export function initLogsTab(deps) {
    if (!instance) {
        instance = new LogsTab(deps);
        instance.init();
    }
    return instance;
}