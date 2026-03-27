/**
 * 日志管理模块 v3
 * 
 * 分类：系统日志 / HTTP日志 / FTP日志 / 面板日志
 */

import { BaseTab } from './BaseTab.js';
import { escapeHtml } from '../core/utils.js';

// 图标
const ICONS = {
    system: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`,
    http: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>`,
    ftp: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>`,
    panel: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="9" y1="21" x2="9" y2="9"/></svg>`,
    refresh: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>`,
    download: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>`,
    trash: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>`
};

// 分类配置
const CATEGORIES = [
    { id: 'system', label: '系统日志', icon: ICONS.system },
    { id: 'http', label: 'HTTP日志', icon: ICONS.http },
    { id: 'ftp', label: 'FTP日志', icon: ICONS.ftp },
    { id: 'panel', label: '面板日志', icon: ICONS.panel }
];

// 子类型配置
const SUB_TYPES = {
    system: [{ id: 'server', label: '服务日志' }],
    http: [],      // 动态加载站点列表
    ftp: [],       // 动态加载用户列表
    panel: [{ id: 'server', label: '面板日志' }]
};

class LogsTab extends BaseTab {
    constructor(deps) {
        super(deps, 'logs');
        this.category = 'system';
        this.type = 'server';
        this.date = '';
        this.logFiles = [];
        this.sites = [];
        this.ftpUsers = [];
    }

    onInit() {
        console.log('初始化日志管理面板...');
        this.initUI();
        this.bindEvents();
    }

    async onLoad() {
        await Promise.all([
            this.loadFiles(),
            this.loadSites(),
            this.loadFTPUsers()
        ]);
        await this.loadLogs();
        await this.loadStats();
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
                    <button id="log-refresh" class="btn btn-sm btn-secondary">${ICONS.refresh} 刷新</button>
                    <button id="log-download" class="btn btn-sm btn-secondary">${ICONS.download} 下载</button>
                    <button id="log-clear" class="btn btn-sm btn-danger">${ICONS.trash} 清空</button>
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
        `;

        this.updateSubTabs();
    }

    updateSubTabs() {
        const container = this.$('#logs-subtabs');
        if (!container) return;

        let types = SUB_TYPES[this.category] || [];
        
        // HTTP: 动态加载站点
        if (this.category === 'http' && this.sites.length > 0) {
            types = this.sites.map(s => ({ id: s.id, label: s.name }));
            if (types.length === 0) {
                types = [{ id: 'default', label: '默认站点' }];
            }
        }
        
        // FTP: 动态加载用户
        if (this.category === 'ftp' && this.ftpUsers.length > 0) {
            types = this.ftpUsers.map(u => ({ id: u.username, label: u.username }));
            if (types.length === 0) {
                types = [{ id: 'anonymous', label: '匿名用户' }];
            }
        }

        if (types.length <= 1) {
            container.innerHTML = '';
            this.type = types[0]?.id || 'all';
            return;
        }

        container.innerHTML = types.map(t => `
            <button class="logs-subtab ${t.id === this.type ? 'active' : ''}" data-type="${t.id}">${t.label}</button>
        `).join('');

        container.querySelectorAll('.logs-subtab').forEach(btn => {
            btn.addEventListener('click', () => {
                this.type = btn.dataset.type;
                container.querySelectorAll('.logs-subtab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                this.loadLogs();
                this.loadStats();
            });
        });

        // 默认选中第一个
        if (!types.find(t => t.id === this.type)) {
            this.type = types[0].id;
        }
    }

    bindEvents() {
        // 分类切换
        this.$$('.logs-tab').forEach(btn => {
            btn.addEventListener('click', () => {
                this.category = btn.dataset.category;
                this.$$('.logs-tab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                this.type = SUB_TYPES[this.category]?.[0]?.id || 'all';
                this.updateSubTabs();
                this.loadLogs();
                this.loadStats();
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

        // 按钮
        this.$('#log-refresh')?.addEventListener('click', () => {
            this.loadLogs();
            this.loadStats();
        });
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

    async loadSites() {
        try {
            this.sites = await this.api.getJSON('/api/sites') || [];
        } catch (error) {
            this.sites = [];
        }
    }

    async loadFTPUsers() {
        try {
            const data = await this.api.getJSON('/api/ftp/users') || [];
            this.ftpUsers = data;
        } catch (error) {
            this.ftpUsers = [];
        }
    }

    updateDateOptions() {
        const select = this.$('#log-date');
        if (!select) return;

        const dates = new Set(['']);
        const today = new Date().toISOString().split('T')[0];

        this.logFiles.forEach(f => {
            if (f.category === this.category || this.category === 'system') {
                if (!f.compressed) {
                    const match = f.name.match(/(\d{4}-\d{2}-\d{2})/);
                    if (match && match[1] !== today) dates.add(match[1]);
                }
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
            const params = new URLSearchParams({ 
                category: this.category, 
                type: this.type, 
                search, 
                level, 
                limit: '200' 
            });
            if (this.date) params.set('date', this.date);

            console.log('Loading logs:', this.category, this.type);  // 调试
            const data = await this.api.getJSON(`/api/logs/read?${params}`);
            console.log('Logs data:', data);  // 调试
            
            if (data?.entries) {
                this.renderLogs(data.entries);
                this.$('#log-count').textContent = `共 ${data.total} 条`;
            } else {
                viewer.innerHTML = '<div class="logs-empty">暂无日志</div>';
            }
        } catch (error) {
            console.error('Load logs error:', error);  // 调试
            viewer.innerHTML = `<div class="logs-error">加载失败: ${error.message}</div>`;
        }
    }

    async loadStats() {
        try {
            const params = new URLSearchParams({ category: this.category, type: this.type });
            const data = await this.api.getJSON(`/api/logs/stats?${params}`) || [];
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

        viewer.scrollTop = viewer.scrollHeight;
    }

    // ========== 操作 ==========

    download() {
        const params = new URLSearchParams({
            category: this.category,
            type: this.type,
            date: this.date
        });
        window.open(`/api/logs/download?${params}`, '_blank');
    }

    confirmClear() {
        const label = CATEGORIES.find(c => c.id === this.category)?.label || this.category;
        this.dialog.confirm(`确定要清空【${label}】吗？此操作不可恢复。`, async () => {
            try {
                await this.api.post(`/api/logs/clear?category=${this.category}&type=${this.type}`);
                this.message.success('日志已清空');
                await this.loadLogs();
            } catch (error) {
                this.message.error('清空失败: ' + error.message);
            }
        });
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