/**
 * 日志管理模块
 *
 * 支持日志查看、搜索、统计、下载
 */

import { globalEvents } from '../core/events.js';

/**
 * SVG 图标
 */
const ICONS = {
    http: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>`,
    folder: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>`,
    settings: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`,
    refresh: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>`,
    download: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>`,
    trash: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>`
};

/**
 * 日志分类配置
 */
const LOG_CATEGORIES = [
    { id: 'http', label: 'HTTP', icon: ICONS.http },
    { id: 'ftp', label: 'FTP', icon: ICONS.folder },
    { id: 'panel', label: '面板', icon: ICONS.settings }
];

const LOG_TYPES = {
    http: [
        { id: 'access', label: '访问日志' },
        { id: 'error', label: '错误日志' }
    ],
    ftp: [
        { id: 'access', label: '访问日志' },
        { id: 'error', label: '错误日志' }
    ],
    panel: [
        { id: 'access', label: '访问日志' },
        { id: 'api', label: 'API 日志' },
        { id: 'auth', label: '认证日志' }
    ]
};

/**
 * 初始化日志面板
 */
export function initLogsTab({ api, toast, message, dialog }) {
    console.log('初始化日志管理面板...');

    // 状态
    let currentCategory = 'http';
    let currentType = 'access';
    let currentDate = '';
    let autoRefresh = true;
    let refreshTimer = null;
    let logFiles = [];

    // 初始化
    initUI();
    bindEvents();
    loadLogFiles();

    // 监听标签页切换
    globalEvents.match('tab:switch:logs', () => {
        loadLogFiles();
        loadLogs();
        loadStats();
        if (autoRefresh) startAutoRefresh();
    });

    /**
     * 初始化 UI
     */
    function initUI() {
        const container = document.getElementById('logs-container');
        if (!container) return;

        // 创建日志管理界面
        container.innerHTML = `
            <!-- 分类 Tab -->
            <div class="logs-tabs">
                ${LOG_CATEGORIES.map(c => `
                    <button class="logs-tab ${c.id === 'http' ? 'active' : ''}" data-category="${c.id}">
                        <span class="logs-tab-icon">${c.icon}</span>
                        <span class="logs-tab-label">${c.label}</span>
                    </button>
                `).join('')}
            </div>

            <!-- 类型 Tab -->
            <div class="logs-subtabs" id="logs-subtabs"></div>

            <!-- 工具栏 -->
            <div class="logs-toolbar">
                <div class="logs-toolbar-left">
                    <select id="log-date" class="form-select-sm">
                        <option value="">今天</option>
                    </select>
                    <select id="log-level" class="form-select-sm">
                        <option value="">全部级别</option>
                        <option value="error">错误</option>
                        <option value="warn">警告</option>
                        <option value="info">信息</option>
                    </select>
                    <input type="text" id="log-search" class="form-input-sm" placeholder="搜索关键词...">
                </div>
                <div class="logs-toolbar-right">
                    <label class="checkbox-label">
                        <input type="checkbox" id="log-auto-refresh" checked>
                        自动刷新
                    </label>
                    <button id="log-refresh" class="btn btn-sm btn-secondary"><span class="btn-icon">${ICONS.refresh}</span> 刷新</button>
                    <button id="log-download" class="btn btn-sm btn-secondary"><span class="btn-icon">${ICONS.download}</span> 下载</button>
                    <button id="log-clear" class="btn btn-sm btn-danger"><span class="btn-icon">${ICONS.trash}</span> 清空</button>
                </div>
            </div>

            <!-- 统计卡片 -->
            <div class="logs-stats" id="logs-stats">
                <div class="stat-card">
                    <div class="stat-value" id="stat-count">-</div>
                    <div class="stat-label">今日请求</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value stat-error" id="stat-errors">-</div>
                    <div class="stat-label">错误</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value stat-warning" id="stat-warnings">-</div>
                    <div class="stat-label">警告</div>
                </div>
            </div>

            <!-- 日志查看器 -->
            <div class="logs-viewer-container">
                <div class="logs-viewer" id="log-viewer">
                    <div class="logs-loading">加载中...</div>
                </div>
                <label class="checkbox-label auto-scroll-label">
                    <input type="checkbox" id="log-auto-scroll" checked>
                    自动滚动
                </label>
            </div>

            <!-- 分页信息 -->
            <div class="logs-pagination" id="logs-pagination">
                <span id="log-count">共 0 条</span>
            </div>
        `;

        // 初始化子标签
        updateSubTabs();
    }

    /**
     * 更新子标签
     */
    function updateSubTabs() {
        const container = document.getElementById('logs-subtabs');
        if (!container) return;

        const types = LOG_TYPES[currentCategory] || [];
        container.innerHTML = types.map(t => `
            <button class="logs-subtab ${t.id === currentType ? 'active' : ''}" data-type="${t.id}">
                ${t.label}
            </button>
        `).join('');

        // 绑定事件
        container.querySelectorAll('.logs-subtab').forEach(btn => {
            btn.addEventListener('click', () => {
                currentType = btn.dataset.type;
                container.querySelectorAll('.logs-subtab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                loadLogs();
                loadStats();
            });
        });
    }

    /**
     * 绑定事件
     */
    function bindEvents() {
        // 分类 Tab 切换
        document.querySelectorAll('.logs-tab').forEach(btn => {
            btn.addEventListener('click', () => {
                currentCategory = btn.dataset.category;
                document.querySelectorAll('.logs-tab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                updateSubTabs();
                currentType = LOG_TYPES[currentCategory]?.[0]?.id || 'access';
                loadLogs();
                loadStats();
            });
        });

        // 日期切换
        document.getElementById('log-date')?.addEventListener('change', (e) => {
            currentDate = e.target.value;
            loadLogs();
        });

        // 搜索
        let searchTimer;
        document.getElementById('log-search')?.addEventListener('input', () => {
            clearTimeout(searchTimer);
            searchTimer = setTimeout(() => loadLogs(), 300);
        });

        // 级别过滤
        document.getElementById('log-level')?.addEventListener('change', () => loadLogs());

        // 自动刷新
        document.getElementById('log-auto-refresh')?.addEventListener('change', (e) => {
            autoRefresh = e.target.checked;
            if (autoRefresh) {
                startAutoRefresh();
            } else {
                stopAutoRefresh();
            }
        });

        // 刷新按钮
        document.getElementById('log-refresh')?.addEventListener('click', () => loadLogs());

        // 下载按钮
        document.getElementById('log-download')?.addEventListener('click', () => downloadLog());

        // 清空按钮
        document.getElementById('log-clear')?.addEventListener('click', () => confirmClearLog());

        // 分页
        document.getElementById('log-prev')?.addEventListener('click', () => loadLogs(-100));
        document.getElementById('log-next')?.addEventListener('click', () => loadLogs(100));
    }

    /**
     * 加载日志文件列表
     */
    async function loadLogFiles() {
        try {
            const data = await api.getJSON('/api/logs');
            if (data && Array.isArray(data)) {
                logFiles = data;
                updateDateOptions(data);
            }
        } catch (error) {
            console.error('加载日志文件列表失败:', error);
        }
    }

    /**
     * 更新日期选项
     */
    function updateDateOptions(files) {
        const select = document.getElementById('log-date');
        if (!select) return;

        const dates = new Set();
        const today = new Date().toISOString().split('T')[0];
        dates.add('');

        files.forEach(f => {
            if (f.category === currentCategory && !f.compressed) {
                const dateMatch = f.name.match(/(\d{4}-\d{2}-\d{2})/);
                if (dateMatch && dateMatch[1] !== today) {
                    dates.add(dateMatch[1]);
                }
            }
        });

        const sortedDates = Array.from(dates).sort().reverse();
        select.innerHTML = sortedDates.map(d => {
            if (d === '') {
                return '<option value="">今天</option>';
            }
            return `<option value="${d}">${d}</option>`;
        }).join('');
    }

    /**
     * 加载日志
     */
    async function loadLogs() {
        const viewer = document.getElementById('log-viewer');
        if (!viewer) return;

        const search = document.getElementById('log-search')?.value || '';
        const level = document.getElementById('log-level')?.value || '';

        viewer.innerHTML = '<div class="logs-loading">加载中...</div>';

        try {
            const params = new URLSearchParams({
                category: currentCategory,
                type: currentType,
                search: search,
                level: level,
                limit: '100'
            });

            if (currentDate) {
                params.set('date', currentDate);
            }

            const data = await api.getJSON(`/api/logs/read?${params}`);

            if (data && data.entries) {
                renderLogs(data.entries);
                updatePagination(data.total);
            } else {
                viewer.innerHTML = '<div class="logs-empty">暂无日志</div>';
            }
        } catch (error) {
            console.error('加载日志失败:', error);
            viewer.innerHTML = `<div class="logs-error">加载失败: ${error.message}</div>`;
        }
    }

    /**
     * 渲染日志
     */
    function renderLogs(entries) {
        const viewer = document.getElementById('log-viewer');
        if (!viewer) return;

        if (!entries || entries.length === 0) {
            viewer.innerHTML = '<div class="logs-empty">暂无日志</div>';
            return;
        }

        const html = entries.map(entry => {
            const levelClass = entry.level ? `log-${entry.level}` : '';
            return `<div class="log-entry ${levelClass}">
                <span class="log-time">${entry.timestamp || ''}</span>
                <span class="log-level">${entry.level ? `[${entry.level.toUpperCase()}]` : ''}</span>
                <span class="log-message">${escapeHtml(entry.message || entry.raw)}</span>
            </div>`;
        }).join('');

        viewer.innerHTML = html;

        if (document.getElementById('log-auto-scroll')?.checked) {
            viewer.scrollTop = viewer.scrollHeight;
        }
    }

    /**
     * 加载统计
     */
    async function loadStats() {
        try {
            const data = await api.getJSON(`/api/logs/stats?category=${currentCategory}`);
            if (data && Array.isArray(data)) {
                let total = 0, errors = 0, warnings = 0;
                data.forEach(stat => {
                    total += stat.count;
                    errors += stat.errors;
                    warnings += stat.warnings;
                });

                document.getElementById('stat-count').textContent = total;
                document.getElementById('stat-errors').textContent = errors;
                document.getElementById('stat-warnings').textContent = warnings;
            }
        } catch (error) {
            console.error('加载统计失败:', error);
        }
    }

    /**
     * 更新分页
     */
    function updatePagination(total) {
        const countEl = document.getElementById('log-count');
        if (countEl) {
            countEl.textContent = `共 ${total} 条`;
        }
    }

    /**
     * 下载日志
     */
    function downloadLog() {
        const params = new URLSearchParams({
            category: currentCategory,
            file: currentDate ? `${currentType}.${currentDate}.log` : `${currentType}.log`
        });
        window.open(`/api/logs/download?${params}`, '_blank');
    }

    /**
     * 确认清空日志
     */
    function confirmClearLog() {
        const label = `${currentCategory.toUpperCase()} - ${currentType}`;
        dialog.confirm(`确定要清空【${label}】吗？此操作不可恢复。`, async () => {
            try {
                await api.getJSON(`/api/logs/clear?category=${currentCategory}&type=${currentType}`, { method: 'POST' });
                message.success('日志已清空');
                loadLogs();
                loadStats();
            } catch (error) {
                message.error('清空失败: ' + error.message);
            }
        });
    }

    /**
     * 开始自动刷新
     */
    function startAutoRefresh() {
        stopAutoRefresh();
        refreshTimer = setInterval(() => {
            if (document.visibilityState === 'visible') {
                loadLogs();
            }
        }, 5000);
    }

    /**
     * 停止自动刷新
     */
    function stopAutoRefresh() {
        if (refreshTimer) {
            clearInterval(refreshTimer);
            refreshTimer = null;
        }
    }

    /**
     * HTML 转义
     */
    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // 清理
    window.addEventListener('beforeunload', stopAutoRefresh);
}