/**
 * 文件管理模块
 *
 * 宝塔风格文件管理器
 * 直接管理服务器文件系统
 * 快捷目录通过 API 动态获取
 */

import { BaseTab } from './BaseTab.js';
import { FileManager } from '../components/file-manager.js';

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
};

class FilesTab extends BaseTab {
    constructor(deps) {
        super(deps, 'files');
        this.fileManager = null;
        this.programDir = null;
    }

    onInit() {
        // 从 API 加载快捷目录
        this.loadQuickDirs();

        // 初始化文件管理器
        this.initFileManager();

        // 绑定分享管理按钮
        this.bindShareButton();
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
            console.warn('[Files] 加载快捷目录失败:', e);
        }
    }

    /**
     * 渲染快捷目录
     */
    renderQuickNav(dirs) {
        const container = this.$('#fm-quick-nav');
        if (!container) return;

        let html = '';
        for (const item of dirs) {
            if (item.section) {
                html += `<div class="fm-quick-divider"></div>`;
                html += `<div class="fm-quick-section-title">${item.section}</div>`;
            } else {
                const icon = ICONS[item.icon] || ICONS.folder;
                let titlePath = item.path;
                if (item.isDefault) {
                    titlePath = this.programDir || item.path;
                }

                html += `
                    <a class="fm-quick-item" data-path="${item.path}" title="${titlePath}">
                        ${icon}
                        <span class="fm-quick-name">${item.name}</span>
                    </a>
                `;
            }
        }

        container.innerHTML = html;
        this.bindQuickNavEvents();
    }

    bindQuickNavEvents() {
        this.$$('.fm-quick-item').forEach(item => {
            item.addEventListener('click', () => {
                this.$$('.fm-quick-item').forEach(i => i.classList.remove('active'));
                item.classList.add('active');

                const path = item.dataset.path;
                if (this.fileManager && path) {
                    this.fileManager.navigate(path);
                }
            });
        });
    }

    initFileManager() {
        const container = this.$('#file-manager-container');
        if (!container) return;

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

    formatSize(bytes) {
        if (!bytes) return '0 B';
        if (bytes < 1024) return bytes + ' B';
        if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
        if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
        return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB';
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
            <div class="fm-share-item" data-token="${link.token}">
                <div class="fm-share-item-icon">📄</div>
                <div class="fm-share-item-info">
                    <div class="fm-share-item-name" title="${link.fileName}">${link.fileName}</div>
                    <div class="fm-share-item-meta">
                        <span>${this.formatSize(link.fileSize)}</span>
                        <span>剩余 ${this.formatExpires(link.expiresAt)}</span>
                        <span>${link.downloadCount} 次下载</span>
                    </div>
                </div>
                <div class="fm-share-item-actions">
                    <button class="fm-share-action copy" data-url="${scheme}://${host}/s/${link.token}" title="复制链接">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                    </button>
                    <button class="fm-share-action delete" data-token="${link.token}" title="删除">
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
