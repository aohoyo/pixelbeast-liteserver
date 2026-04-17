/**
 * 文件管理模块
 *
 * 宝塔风格文件管理器
 * 直接管理服务器文件系统
 * 快捷目录通过 API 动态获取
 */

import { BaseTab } from './BaseTab.js';
import { FileManager } from '../components/file-manager.js';
import { escapeHtml } from '../core/utils.js';

// 图标 SVG
const ICONS = {
    server: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="2" width="20" height="8" rx="2" ry="2"/><rect x="2" y="14" width="20" height="8" rx="2" ry="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/></svg>`,
    home: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/></svg>`,
    database: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/></svg>`,
    settings: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`,
    trash: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>`,
    package: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="16.5" y1="9.4" x2="7.5" y2="4.21"/><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>`,
    folder: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>`,
    'file-text': `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>`,
    computer: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>`,
    'hard-drive': `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="22" y1="12" x2="2" y2="12"/><path d="M5.45 5.11L2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/><line x1="6" y1="16" x2="6.01" y2="16"/><line x1="10" y1="16" x2="10.01" y2="16"/></svg>`,
    users: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
    desktop: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>`,
    download: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>`,
    image: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>`,
    music: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>`,
    video: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/></svg>`,
    share: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/></svg>`,
    pin: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L12 22"/><path d="M17 7l-5-5-5 5"/><circle cx="12" cy="12" r="3"/></svg>`,
    edit: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>`,
    'close-x': `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>`,
    'restore': `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>`,
};

class FilesTab extends BaseTab {
    constructor(deps) {
        super(deps, 'files');
        this.fileManager = null;
        this.programDir = null;
        this._trashOverlay = null;
    }

    onInit() {
        // 监听外部导航事件（如 FTP 页面跳转）
        this.events.on('files:navigate', (path) => {
            if (this.fileManager && path) {
                this.fileManager.navigate(path);
            }
        });

        // 从 API 加载快捷目录
        this.loadQuickDirs();

        // 初始化文件管理器
        this.initFileManager();

        // 绑定分享管理按钮
        this.bindShareButton();

        // 绑定回收站按钮
        this.bindTrashButton();
    }

    /**
     * 从 API 加载快捷目录
     */
    async loadQuickDirs() {
        const container = this.$('#fm-quick-nav');
        if (!container) return;

        try {
            const result = await this.api.getJSON('/api/files/quick-dirs');
            if (result?.dirs) {
                if (result.program_dir) {
                    this.programDir = result.program_dir;
                }
                this.renderQuickNav(result.dirs);
            }
        } catch (e) {
            // 快捷目录加载失败，忽略
        }
    }

    /**
     * 渲染快捷目录（支持固定/编辑/删除）
     */
    renderQuickNav(dirs) {
        const container = this.$('#fm-quick-nav');
        if (!container) return;

        let html = '';
        for (const item of dirs) {
            if (item.section) {
                if (html) html += `<div class="fm-quick-divider"></div>`;
            } else {
                const icon = ICONS[item.icon] || ICONS.folder;
                const pinned = item.pinned ? ' data-pinned="1"' : '';

                html += `
                    <a class="fm-quick-item" data-path="${escapeHtml(item.path)}" data-name="${escapeHtml(item.name)}" data-icon="${escapeHtml(item.icon || 'folder')}"${pinned}>
                        ${icon}
                        <span class="fm-quick-name">${escapeHtml(item.name)}</span>
                    </a>
                `;
            }
        }

        container.innerHTML = html;
        this.bindQuickNavEvents();
    }

    bindQuickNavEvents() {
        this.$$('.fm-quick-item').forEach(item => {
            // 点击导航
            item.addEventListener('click', (e) => {
                // 如果点击的是操作按钮，不导航
                if (e.target.closest('.fm-quick-action-btn')) return;

                this.$$('.fm-quick-item').forEach(i => i.classList.remove('active'));
                item.classList.add('active');

                const path = item.dataset.path;
                if (this.fileManager && path) {
                    this.fileManager.navigate(path);
                }
            });

            // 右键菜单（编辑/移除通过右键操作）
            item.addEventListener('contextmenu', (e) => {
                e.preventDefault();
                this.showQuickDirMenu(item, e);
            });
        });
    }

    /**
     * 快捷目录右键菜单
     */
    showQuickDirMenu(item, e) {
        const path = item.dataset.path;
        const name = item.dataset.name;
        const isPinned = item.dataset.pinned === '1';
        const isDefault = path === '.' || path === './';

        document.querySelectorAll('.fm-context-menu').forEach(m => m.remove());

        const menu = document.createElement('div');
        menu.className = 'fm-context-menu';
        menu.style.left = e.clientX + 'px';
        menu.style.top = e.clientY + 'px';

        let items = '';
        if (isDefault) {
            items = '<div class="fm-menu-item disabled">项目目录（始终显示）</div>';
        } else if (isPinned) {
            items = '<div class="fm-menu-item" data-action="unpin">取消固定</div>';
        } else {
            items = '<div class="fm-menu-item" data-action="hide">隐藏此目录</div>';
        }
        menu.innerHTML = items;
        document.body.appendChild(menu);

        const rect = menu.getBoundingClientRect();
        if (rect.right > window.innerWidth) menu.style.left = (e.clientX - rect.width) + 'px';
        if (rect.bottom > window.innerHeight) menu.style.top = (e.clientY - rect.height) + 'px';

        menu.addEventListener('click', async (ev) => {
            const action = ev.target.dataset.action;
            menu.remove();
            if (!action || ev.target.classList.contains('disabled')) return;
            await this.removeQuickDir(item);
        });

        const closeMenu = () => { menu.remove(); document.removeEventListener('click', closeMenu); };
        setTimeout(() => document.addEventListener('click', closeMenu), 0);
    }

    /**
     * 移除快捷目录（取消固定或隐藏）
     */
    async removeQuickDir(item) {
        const path = item.dataset.path;
        const name = item.dataset.name;
        const isPinned = item.dataset.pinned === '1';

        const msg = isPinned
            ? `确定要取消固定"${name}"吗？`
            : `确定要隐藏"${name}"吗？可在设置中恢复。`;

        const confirmed = await this.fileManager.showConfirmDialog(
            isPinned ? '取消固定' : '隐藏目录',
            msg
        );
        if (!confirmed) return;

        try {
            await this.api.post('/api/files/quick-dirs/remove', {
                path,
                pinned: isPinned
            });
            this.toast?.success?.(isPinned ? '已取消固定' : '已隐藏');
            await this.loadQuickDirs();
        } catch {
            this.toast?.error?.('操作失败');
        }
    }

        /**
     * 固定目录到快速访问（由 file-manager 右键菜单调用）
     */
    async pinToQuickAccess(path) {
        const name = path.split('/').pop() || path;
        try {
            await this.api.post('/api/files/quick-dirs/add', {
                path,
                name,
                icon: 'folder'
            });
            this.toast?.success?.('已固定到快速访问');
            await this.loadQuickDirs();
        } catch (e) {
            const msg = e?.message || '固定失败';
            this.toast?.error?.(msg);
        }
    }

    initFileManager() {
        const container = this.$('#file-manager-container');
        if (!container) return;

        // 保存引用供右键菜单回调
        const filesTab = this;

        this.fileManager = new FileManager({
            container: container,
            apiPath: '/admin/api/files',
            root: '.',
            viewMode: 'grid',
            toast: this.toast,
            dialog: this.dialog,
            onOpen: (path) => {
                window.open(`/admin/api/files/download?path=${encodeURIComponent(path)}`, '_blank');
            },
            onPathChange: (_path, programDir) => {
                if (programDir && !this.programDir) {
                    this.programDir = programDir;
                }
            },
            onPinFolder: (path) => {
                filesTab.pinToQuickAccess(path);
            }
        });

        // 监听路径变化
        const originalNavigate = this.fileManager.navigate.bind(this.fileManager);
        this.fileManager.navigate = (path) => {
            originalNavigate(path);
            this.updateQuickNavActive(path);
        };

        // 延迟加载检查
        setTimeout(() => {
            if (this.$('#files')?.classList.contains('active')) {
                this.fileManager?.loadFilesForTab();
            }
        }, 500);
    }

    updateQuickNavActive(path) {
        this.$$('.fm-quick-item').forEach(item => {
            const itemPath = item.dataset.path;
            item.classList.toggle('active', itemPath === path);
        });
    }

    // ==================== 回收站 ====================

    bindTrashButton() {
        const btn = this.$('#fm-trash-btn');
        if (!btn) return;
        btn.addEventListener('click', () => this.openTrashDialog());
    }

    /**
     * 打开回收站弹窗
     */
    async openTrashDialog() {
        if (this._trashOverlay) {
            this._trashOverlay.remove();
            this._trashOverlay = null;
        }

        const overlay = document.createElement('div');
        overlay.className = 'fm-share-overlay';
        overlay.innerHTML = `
            <div class="fm-trash-dialog">
                <div class="fm-share-dialog-header">
                    <div class="fm-share-dialog-title">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:20px;height:20px;color:var(--primary)">
                            <polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                        </svg>
                        回收站
                    </div>
                    <button class="fm-share-dialog-close">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                    </button>
                </div>
                <div class="fm-trash-toolbar">
                    <button class="fm-trash-restore-all">
                        ${ICONS.restore}
                        <span>全部恢复</span>
                    </button>
                    <button class="fm-trash-clear fm-btn-danger-text">
                        ${ICONS.trash}
                        <span>清空回收站</span>
                    </button>
                </div>
                <div class="fm-trash-body">
                    <table class="fm-trash-table">
                        <colgroup>
                            <col class="col-name">
                            <col class="col-path">
                            <col class="col-size">
                            <col class="col-date">
                            <col class="col-actions">
                        </colgroup>
                        <thead>
                            <tr>
                                <th>文件名</th>
                                <th>原路径</th>
                                <th>大小</th>
                                <th>删除时间</th>
                                <th>操作</th>
                            </tr>
                        </thead>
                        <tbody id="fm-trash-list">
                            <tr><td colspan="5" class="fm-trash-empty">加载中...</td></tr>
                        </tbody>
                    </table>
                </div>
            </div>
        `;

        document.body.appendChild(overlay);
        this._trashOverlay = overlay;

        const close = () => { overlay.remove(); this._trashOverlay = null; };
        overlay.querySelector('.fm-share-dialog-close').addEventListener('click', close);
        overlay.addEventListener('click', (e) => { if (e.target === overlay) close(); });

        // 全部恢复
        overlay.querySelector('.fm-trash-restore-all').addEventListener('click', async () => {
            const tbody = overlay.querySelector('#fm-trash-list');
            const rows = tbody.querySelectorAll('.fm-trash-row');
            if (rows.length === 0) return;
            const confirmed = await this.fileManager.showConfirmDialog('全部恢复', '确定要恢复全部文件吗？');
            if (!confirmed) return;

            let success = 0;
            for (const row of rows) {
                const id = row.dataset.id;
                if (!id) continue;
                try {
                    await this.api.post('/api/files/trash/restore', { id });
                    success++;
                } catch { /* ignore */ }
            }
            this.toast?.success?.(`已恢复 ${success} 个项目`);
            this.loadTrashList(overlay);
            this.fileManager?.loadFilesForTab();
        });

        // 清空回收站
        overlay.querySelector('.fm-trash-clear').addEventListener('click', async () => {
            const confirmed = await this.fileManager.showConfirmDialog('清空回收站', '确定要永久清空回收站吗？此操作不可恢复。');
            if (!confirmed) return;
            try {
                await this.api.post('/api/files/trash/clear', {});
                this.toast?.success?.('回收站已清空');
                this.loadTrashList(overlay);
            } catch {
                this.toast?.error?.('清空失败');
            }
        });

        // 加载数据
        this.loadTrashList(overlay);
    }

    async loadTrashList(overlay) {
        const tbody = overlay.querySelector('#fm-trash-list');
        if (!tbody) return;

        try {
            const result = await this.api.getJSON('/api/files/trash/list');
            this.renderTrashList(result?.items || [], overlay);
        } catch {
            tbody.innerHTML = `<tr><td colspan="5" class="fm-trash-empty">加载失败</td></tr>`;
        }
    }

    formatSize(bytes) {
        if (!bytes) return '0 B';
        if (bytes < 1024) return bytes + ' B';
        if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
        if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
        return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB';
    }

    formatDate(dateStr) {
        if (!dateStr) return '';
        const d = new Date(dateStr);
        const pad = n => String(n).padStart(2, '0');
        return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
    }

    renderTrashList(items, overlay) {
        const tbody = overlay.querySelector('#fm-trash-list');
        if (!tbody) return;

        const toolbar = overlay.querySelector('.fm-trash-toolbar');

        if (!items.length) {
            tbody.innerHTML = `<tr><td colspan="5" class="fm-trash-empty">回收站为空</td></tr>`;
            if (toolbar) toolbar.style.display = 'none';
            return;
        }

        if (toolbar) toolbar.style.display = '';

        tbody.innerHTML = items.map(item => `
            <tr class="fm-trash-row" data-id="${escapeHtml(item.id)}">
                <td>
                    <div class="fm-trash-name-wrap">
                        <span class="fm-trash-file-icon">${item.is_dir ? '📁' : '📄'}</span>
                        <span class="fm-trash-file-name">${escapeHtml(item.original_name)}</span>
                    </div>
                </td>
                <td class="fm-trash-col-path" title="${escapeHtml(item.original_path)}">${escapeHtml(item.original_path)}</td>
                <td class="fm-trash-col-size">${this.formatSize(item.size)}</td>
                <td class="fm-trash-col-date">${this.formatDate(item.deleted_at)}</td>
                <td>
                    <div class="fm-trash-actions-wrap">
                        <button class="fm-trash-action restore" title="恢复">${ICONS.restore}</button>
                        <button class="fm-trash-action delete" title="永久删除">${ICONS.trash}</button>
                    </div>
                </td>
            </tr>
        `).join('');

        // 恢复
        tbody.querySelectorAll('.fm-trash-action.restore').forEach(btn => {
            btn.addEventListener('click', async () => {
                const id = btn.closest('.fm-trash-row')?.dataset.id;
                if (!id) return;
                try {
                    await this.api.post('/api/files/trash/restore', { id });
                    this.toast?.success?.('已恢复');
                    this.loadTrashList(overlay);
                    this.fileManager?.loadFilesForTab();
                } catch {
                    this.toast?.error?.('恢复失败');
                }
            });
        });

        // 永久删除
        tbody.querySelectorAll('.fm-trash-action.delete').forEach(btn => {
            btn.addEventListener('click', async () => {
                const id = btn.closest('.fm-trash-row')?.dataset.id;
                if (!id) return;
                const confirmed = await this.fileManager.showConfirmDialog('永久删除', '此操作不可恢复，确定要永久删除吗？');
                if (!confirmed) return;
                try {
                    await this.api.post('/api/files/trash/delete', { id });
                    this.toast?.success?.('已永久删除');
                    this.loadTrashList(overlay);
                } catch {
                    this.toast?.error?.('删除失败');
                }
            });
        });
    }

    // ==================== 分享管理（独立弹窗） ====================

    bindShareButton() {
        const btn = this.$('#fm-share-btn');
        if (!btn) return;
        btn.addEventListener('click', () => this.openShareDialog());
    }

    /**
     * 打开分享管理弹窗
     */
    async openShareDialog() {
        // 创建弹窗
        const overlay = document.createElement('div');
        overlay.className = 'fm-share-overlay';
        overlay.innerHTML = `
            <div class="fm-share-dialog">
                <div class="fm-share-dialog-header">
                    <div class="fm-share-dialog-title">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:20px;height:20px;color:var(--primary)">
                            <circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/>
                            <line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/>
                        </svg>
                        我的分享
                    </div>
                    <button class="fm-share-dialog-close" title="关闭">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                    </button>
                </div>
                <div class="fm-share-dialog-body">
                    <div class="fm-share-list" id="fm-share-list-dialog">
                        <div class="fm-share-empty">加载中...</div>
                    </div>
                </div>
            </div>
        `;

        document.body.appendChild(overlay);

        // 关闭
        const close = () => overlay.remove();
        overlay.querySelector('.fm-share-dialog-close').addEventListener('click', close);
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) close();
        });

        // 加载数据
        try {
            const result = await this.api.getJSON('/api/files/share/list');
            this.renderShareList(result?.links || [], overlay, close);
        } catch (e) {
            const listEl = overlay.querySelector('#fm-share-list-dialog');
            if (listEl) listEl.innerHTML = `<div class="fm-share-empty">加载失败</div>`;
        }
    }

    formatExpires(expiresAt) {
        if (!expiresAt) return '永久';
        const diff = new Date(expiresAt) - Date.now();
        if (diff <= 0) return '已过期';
        const days = Math.floor(diff / 86400000);
        const hours = Math.floor((diff % 86400000) / 3600000);
        if (days > 0) return `${days}天${hours}小时`;
        if (hours > 0) return `${hours}小时`;
        return `${Math.floor(diff / 60000)}分钟`;
    }

    renderShareList(links, overlay, close) {
        const listEl = overlay.querySelector('#fm-share-list-dialog');
        if (!listEl) return;

        if (!links.length) {
            listEl.innerHTML = `<div class="fm-share-empty">暂无分享文件</div>`;
            return;
        }

        const scheme = location.protocol === 'https:' ? 'https' : 'http';
        const host = location.host;

        listEl.innerHTML = links.map(link => `
            <div class="fm-share-item" data-token="${escapeHtml(link.token)}">
                <div class="fm-share-item-icon">📄</div>
                <div class="fm-share-item-info">
                    <div class="fm-share-item-name" title="${escapeHtml(link.fileName)}">${escapeHtml(link.fileName)}</div>
                    <div class="fm-share-item-meta">
                        <span>${this.formatSize(link.fileSize)}</span>
                        <span>剩余 ${this.formatExpires(link.expiresAt)}</span>
                        <span>${link.downloadCount} 次下载</span>
                    </div>
                </div>
                <div class="fm-share-item-actions">
                    <button class="fm-share-action copy" data-url="${scheme}://${host}/s/${escapeHtml(link.token)}" title="复制链接">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                    </button>
                    <button class="fm-share-action delete" data-token="${escapeHtml(link.token)}" title="删除">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                    </button>
                </div>
            </div>
        `).join('');

        // 复制链接
        listEl.querySelectorAll('.fm-share-action.copy').forEach(btn => {
            btn.addEventListener('click', async () => {
                try {
                    await navigator.clipboard.writeText(btn.dataset.url);
                    this.toast?.success?.('链接已复制');
                } catch {
                    this.toast?.error?.('复制失败');
                }
            });
        });

        // 删除分享
        listEl.querySelectorAll('.fm-share-action.delete').forEach(btn => {
            btn.addEventListener('click', async () => {
                const token = btn.dataset.token;
                try {
                    await this.api.post('/api/files/share/delete', { token });
                    this.toast?.success?.('已删除');
                    btn.closest('.fm-share-item')?.remove();
                    if (!listEl.querySelector('.fm-share-item')) {
                        listEl.innerHTML = `<div class="fm-share-empty">暂无分享文件</div>`;
                    }
                } catch {
                    this.toast?.error?.('删除失败');
                }
            });
        });
    }

    async onLoad() {
        if (this.fileManager) {
            this.fileManager.loadFilesForTab();
        }
    }
}

// 单例
let instance = null;

export function initFilesTab(deps) {
    if (!instance) {
        instance = new FilesTab(deps);
        instance.init();
    }
    return instance;
}
