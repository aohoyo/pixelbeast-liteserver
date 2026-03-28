/**
 * 文件管理模块
 * 
 * 宝塔风格文件管理器
 * 直接管理服务器文件系统
 * 根据操作系统类型显示不同的快捷目录
 */

import { BaseTab } from './BaseTab.js';
import { FileManager } from '../components/file-manager.js';

// 快捷目录配置
const QUICK_DIRS = {
    // Linux 系统目录
    linux: [
        { path: '.', name: '项目目录', icon: 'folder', isDefault: true, showFullPath: true },
        { section: '系统目录' },
        { path: '/', name: '根目录', icon: 'server' },
        { path: '/home', name: 'home', icon: 'home' },
        { path: '/var', name: 'var', icon: 'database' },
        { path: '/etc', name: 'etc', icon: 'settings' },
        { path: '/tmp', name: 'tmp', icon: 'trash' },
        { path: '/usr', name: 'usr', icon: 'package' }
    ],
    
    // Windows 系统目录
    windows: [
        { path: '.', name: '项目目录', icon: 'folder', isDefault: true, showFullPath: true },
        { section: '系统目录' },
        { path: '此电脑', name: '此电脑', icon: 'computer' },
        { path: 'C:/', name: 'C 盘', icon: 'hard-drive' },
        { path: 'D:/', name: 'D 盘', icon: 'hard-drive' },
        { path: 'E:/', name: 'E 盘', icon: 'hard-drive' }
    ],
    
    // macOS 系统目录
    darwin: [
        { path: '.', name: '项目目录', icon: 'folder', isDefault: true, showFullPath: true },
        { section: '系统目录' },
        { path: '/', name: '根目录', icon: 'server' },
        { path: '/Users', name: 'Users', icon: 'users' },
        { path: '/Applications', name: 'Applications', icon: 'package' },
        { path: '/Library', name: 'Library', icon: 'folder' },
        { path: '/tmp', name: 'tmp', icon: 'trash' }
    ]
};

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
    users: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`
};

class FilesTab extends BaseTab {
    constructor(deps) {
        super(deps, 'files');
        this.fileManager = null;
        this.programDir = null;
        this.systemInfo = null;
    }

    onInit() {
        console.log('初始化文件管理...');
        
        // 获取系统信息，确保正确获取
        this.systemInfo = this.state?.get?.('system');
        if (!this.systemInfo) {
            // 如果还没有系统信息，使用默认值
            this.systemInfo = { os: 'linux', isWindows: false, isLinux: true, isMac: false };
            console.warn('[Files] 系统信息未初始化，使用默认值:', this.systemInfo);
        } else {
            console.log('[Files] 系统信息:', this.systemInfo);
        }

        // 渲染快捷目录
        this.renderQuickNav();
        
        // 初始化文件管理器
        this.initFileManager();
    }

    /**
     * 根据系统类型渲染快捷目录
     */
    renderQuickNav() {
        const container = this.$('#fm-quick-nav');
        if (!container) return;

        const os = this.systemInfo.os || 'linux';
        const dirs = QUICK_DIRS[os] || QUICK_DIRS.linux;

        let html = '';
        for (const item of dirs) {
            if (item.section) {
                // 分隔标题
                html += `<div class="fm-quick-divider"></div>`;
                html += `<div class="fm-quick-section-title">${item.section}</div>`;
            } else {
                // 快捷项
                const icon = ICONS[item.icon] || ICONS.folder;
                // 显示名称，hover 显示完整路径
                let displayName = item.name;
                let titlePath = item.path;
                
                // 项目目录特殊处理：始终显示"项目目录"，hover 显示完整路径
                if (item.showFullPath) {
                    titlePath = this.programDir || item.path;
                }
                
                html += `
                    <a class="fm-quick-item" data-path="${item.path}" title="${titlePath}">
                        ${icon}
                        <span class="fm-quick-name">${displayName}</span>
                    </a>
                `;
            }
        }

        container.innerHTML = html;

        // 绑定点击事件
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
            root: '.',  // 默认程序运行目录（Windows/Linux 通用）
            viewMode: 'grid',  // 默认图标模式
            toast: this.toast,
            dialog: this.dialog,  // 传递 dialog 组件
            onOpen: (path) => {
                window.open(`/admin/api/files/download?path=${encodeURIComponent(path)}`, '_blank');
            },
            onPathChange: (path, programDir) => {
                // 更新项目目录显示
                if (programDir && !this.programDir) {
                    this.programDir = programDir;
                    this.renderQuickNav();
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