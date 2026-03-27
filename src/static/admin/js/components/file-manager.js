/**
 * FileManager - 文件管理器组件
 * 
 * 支持多标签页、仿 Windows 资源管理器
 */

import { getFileIcon, getIconColorClass, formatFileSize, formatDate } from './file-icons.js';
import { contextMenu } from './context-menu.js';

// SVG 图标
const ICONS = {
    back: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>`,
    forward: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>`,
    up: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="18 15 12 9 6 15"/></svg>`,
    refresh: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>`,
    mkdir: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/><line x1="12" y1="11" x2="12" y2="17"/><line x1="9" y1="14" x2="15" y2="14"/></svg>`,
    file: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>`,
    delete: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>`,
    copy: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>`,
    cut: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="6" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><line x1="20" y1="4" x2="8.12" y2="15.88"/><line x1="14.47" y1="14.48" x2="20" y2="20"/><line x1="8.12" y1="8.12" x2="12" y2="12"/></svg>`,
    paste: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/><rect x="8" y="2" width="8" height="4" rx="1" ry="1"/></svg>`,
    search: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>`,
    grid: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>`,
    list: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>`,
    folder: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>`,
    sortAsc: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 5h10"/><path d="M11 9h7"/><path d="M11 13h4"/><path d="M3 17l3 3 3-3"/><path d="M6 18V4"/></svg>`,
    sortDesc: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 5h10"/><path d="M11 9h7"/><path d="M11 13h4"/><path d="M3 7l3-3 3 3"/><path d="M6 6v14"/></svg>`,
    close: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>`,
    plus: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>`,
    trash: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>`,
    download: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>`,
    selectAll: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M8 12l3 3 5-6"/></svg>`,
    // 右键菜单新增图标
    edit: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>`,
    rename: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"/></svg>`,
    share: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/></svg>`,
    compress: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 8v13H3V8"/><path d="M1 3h22v5H1z"/><path d="M10 12h4"/></svg>`,
    extract: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 8v13H3V8"/><path d="M1 3h22v5H1z"/><polyline points="8 12 12 16 16 12"/></svg>`,
    chmod: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>`,
    copyPath: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>`,
    info: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>`
};

export class FileManager {
    constructor(options) {
        this.options = {
            container: null,
            apiPath: '/admin/api/files',
            root: '.',
            viewMode: 'grid',
            onSelect: null,
            onOpen: null,
            ...options
        };

        // 多标签支持
        this.tabs = [];
        this.activeTabId = null;
        this.tabIdCounter = 0;
        
        // 项目目录完整路径
        this.programDir = null;
        
        // 地址栏模式：breadcrumb / input
        this.addressMode = 'breadcrumb';

        this.container = typeof this.options.container === 'string'
            ? document.querySelector(this.options.container)
            : this.options.container;

        if (this.container) {
            this.init();
        }
    }

    init() {
        this.render();
        this.bindEvents();
        // 创建第一个标签
        this.createTab(this.options.root);
    }

    render() {
        this.container.innerHTML = `
            <div class="file-manager" style="position:absolute; top:0; left:0; right:0; bottom:0; display:flex; flex-direction:column;">
                <!-- 标签栏 -->
                <div class="fm-tabs-bar">
                    <div class="fm-tabs" id="fm-tabs"></div>
                    <button class="fm-tab-new" id="fm-new-tab" title="新建标签">${ICONS.plus}</button>
                </div>
                
                <!-- 工具栏第一行 -->
                <div class="fm-toolbar fm-toolbar-row1">
                    <div class="fm-toolbar-group">
                        <button class="fm-btn" id="fm-back" title="后退 (Alt+←)" disabled>${ICONS.back}</button>
                        <button class="fm-btn" id="fm-forward" title="前进 (Alt+→)" disabled>${ICONS.forward}</button>
                        <button class="fm-btn" id="fm-up" title="向上 (Alt+↑)">${ICONS.up}</button>
                        <button class="fm-btn" id="fm-refresh" title="刷新 (F5)">${ICONS.refresh}</button>
                    </div>
                    
                    <div class="fm-toolbar-divider"></div>
                    
                    <!-- 地址栏 - 面包屑导航 -->
                    <div class="fm-address-bar" id="fm-breadcrumb">
                        <span class="fm-breadcrumb-root" data-path="/">${ICONS.folder}</span>
                        <span class="fm-breadcrumb-sep">/</span>
                        <span class="fm-breadcrumb-item" data-path="/home">home</span>
                        <!-- 由 JS 动态生成 -->
                    </div>
                    
                    <!-- 隐藏的输入框，用于手动输入路径 -->
                    <input type="text" class="fm-address-input" id="fm-path" placeholder="路径" style="display:none;">
                    
                    <div class="fm-toolbar-divider"></div>
                    
                    <!-- 视图切换 -->
                    <div class="fm-view-btns" id="fm-view-btns">
                        <button class="fm-view-btn" data-view="list" title="列表视图">${ICONS.list}</button>
                        <button class="fm-view-btn active" data-view="grid" title="图标视图">${ICONS.grid}</button>
                    </div>
                    
                    <div class="fm-toolbar-divider"></div>
                    
                    <!-- 搜索 -->
                    <div class="fm-search">
                        ${ICONS.search}
                        <input type="text" id="fm-search-input" placeholder="搜索当前目录..." class="fm-search-input">
                    </div>
                </div>
                
                <!-- 工具栏第二行 -->
                <div class="fm-toolbar fm-toolbar-row2">
                    <div class="fm-toolbar-group">
                        <div class="fm-dropdown">
                            <button class="fm-btn fm-btn-text fm-dropdown-toggle" id="fm-new-btn">
                                ${ICONS.plus}
                                <span>新建</span>
                                <svg class="fm-dropdown-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
                            </button>
                            <div class="fm-dropdown-menu">
                                <div class="fm-dropdown-item" id="fm-mkdir">
                                    ${ICONS.mkdir}
                                    <span>新建文件夹</span>
                                </div>
                                <div class="fm-dropdown-item" id="fm-newfile">
                                    ${ICONS.file}
                                    <span>新建文件</span>
                                </div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="fm-toolbar-divider"></div>
                    
                    <div class="fm-toolbar-group">
                        <button class="fm-btn fm-btn-text" id="fm-cut" title="剪切 (Ctrl+X)" disabled>
                            ${ICONS.cut}
                            <span>剪切</span>
                        </button>
                        <button class="fm-btn fm-btn-text" id="fm-copy" title="复制 (Ctrl+C)" disabled>
                            ${ICONS.copy}
                            <span>复制</span>
                        </button>
                        <button class="fm-btn fm-btn-text" id="fm-paste" title="粘贴 (Ctrl+V)" disabled>
                            ${ICONS.paste}
                            <span>粘贴</span>
                        </button>
                        <button class="fm-btn fm-btn-text fm-btn-danger" id="fm-delete" title="删除 (Delete)" disabled>
                            ${ICONS.delete}
                            <span>删除</span>
                        </button>
                    </div>
                    
                    <div class="fm-toolbar-divider"></div>
                    
                    <div class="fm-toolbar-group">
                        <button class="fm-btn fm-btn-text" id="fm-select-all" title="全选 (Ctrl+A)">
                            ${ICONS.selectAll}
                            <span>全选</span>
                        </button>
                    </div>
                    
                    <div class="fm-toolbar-divider"></div>
                    
                    <div class="fm-toolbar-group fm-toolbar-sort">
                        <label class="fm-sort-label">排序：</label>
                        <select id="fm-sort" class="fm-select">
                            <option value="name">名称</option>
                            <option value="size">大小</option>
                            <option value="date">日期</option>
                            <option value="type">类型</option>
                        </select>
                        <button class="fm-btn" id="fm-sort-order" title="排序顺序">${ICONS.sortAsc}</button>
                    </div>
                </div>
                
                <!-- 主内容区 -->
                <div class="fm-main-area">
                    <!-- 文件列表 -->
                    <div class="fm-content" id="fm-content"></div>
                </div>
                
                <!-- 拖拽上传遮罩 -->
                <div class="fm-drop-overlay" id="fm-drop-overlay">
                    <div class="fm-drop-overlay-text">拖放文件到此处上传</div>
                </div>
            </div>
            
            <input type="file" id="fm-file-input" multiple hidden>
        `;
        
        // 缓存元素
        this.els = {
            tabsContainer: this.container.querySelector('#fm-tabs'),
            newTabBtn: this.container.querySelector('#fm-new-tab'),
            back: this.container.querySelector('#fm-back'),
            forward: this.container.querySelector('#fm-forward'),
            up: this.container.querySelector('#fm-up'),
            refresh: this.container.querySelector('#fm-refresh'),
            breadcrumb: this.container.querySelector('#fm-breadcrumb'),
            path: this.container.querySelector('#fm-path'),
            search: this.container.querySelector('#fm-search-input'),
            content: this.container.querySelector('#fm-content'),
            dropOverlay: this.container.querySelector('#fm-drop-overlay'),
            fileInput: this.container.querySelector('#fm-file-input'),
            viewBtns: this.container.querySelector('#fm-view-btns'),
            newBtn: this.container.querySelector('#fm-new-btn'),
            mkdir: this.container.querySelector('#fm-mkdir'),
            newfile: this.container.querySelector('#fm-newfile'),
            cut: this.container.querySelector('#fm-cut'),
            copy: this.container.querySelector('#fm-copy'),
            paste: this.container.querySelector('#fm-paste'),
            delete: this.container.querySelector('#fm-delete'),
            selectAll: this.container.querySelector('#fm-select-all'),
            sort: this.container.querySelector('#fm-sort'),
            sortOrder: this.container.querySelector('#fm-sort-order')
        };
        
        // 剪贴板
        this.clipboard = { type: null, items: [], sourcePath: null };
    }

    // ========== 标签管理 ==========

    createTab(path = '.') {
        const tabId = ++this.tabIdCounter;
        const tab = {
            id: tabId,
            path: path,
            history: [],
            historyIndex: -1,
            files: [],
            selectedItems: new Set(),
            viewMode: 'grid',
            sortBy: 'name',
            sortOrder: 'asc'
        };
        
        this.tabs.push(tab);
        this.renderTabs();
        this.switchToTab(tabId);
        this.loadFilesForTab(tabId);
        
        return tabId;
    }

    switchToTab(tabId) {
        this.activeTabId = tabId;
        const tab = this.getActiveTab();
        if (!tab) return;
        
        // 更新 UI
        this.renderTabs();
        this.els.path.value = tab.path;
        this.updateNavButtons();
        this.updateToolbarButtons();
        this.renderFilesForTab();
    }

    closeTab(tabId) {
        if (this.tabs.length <= 1) return; // 至少保留一个标签
        
        const index = this.tabs.findIndex(t => t.id === tabId);
        if (index === -1) return;
        
        this.tabs.splice(index, 1);
        
        if (this.activeTabId === tabId) {
            // 切换到相邻标签
            const newActiveTab = this.tabs[Math.min(index, this.tabs.length - 1)];
            this.switchToTab(newActiveTab.id);
        } else {
            this.renderTabs();
        }
    }

    getActiveTab() {
        return this.tabs.find(t => t.id === this.activeTabId);
    }

    renderTabs() {
        this.els.tabsContainer.innerHTML = this.tabs.map(tab => {
            const isActive = tab.id === this.activeTabId;
            const name = this.getTabName(tab.path);
            return `
                <div class="fm-tab ${isActive ? 'active' : ''}" data-tab-id="${tab.id}">
                    <span class="fm-tab-name">${name}</span>
                    ${this.tabs.length > 1 ? `<button class="fm-tab-close" data-tab-id="${tab.id}">${ICONS.close}</button>` : ''}
                </div>
            `;
        }).join('');
        
        // 绑定标签点击
        this.els.tabsContainer.querySelectorAll('.fm-tab').forEach(el => {
            el.addEventListener('click', (e) => {
                if (!e.target.classList.contains('fm-tab-close')) {
                    this.switchToTab(parseInt(el.dataset.tabId));
                }
            });
        });
        
        // 绑定关闭按钮
        this.els.tabsContainer.querySelectorAll('.fm-tab-close').forEach(el => {
            el.addEventListener('click', (e) => {
                e.stopPropagation();
                this.closeTab(parseInt(el.dataset.tabId));
            });
        });
    }

    getTabName(path) {
        if (path === '.' || path === './') return '项目目录';
        if (path === '/') return '根目录';
        if (path === '此电脑') return '此电脑';
        
        const parts = path.split('/');
        return parts[parts.length - 1] || path;
    }

    // ========== 事件绑定 ==========

    bindEvents() {
        // 新建标签
        this.els.newTabBtn?.addEventListener('click', () => this.createTab());
        
        // 导航
        this.els.back?.addEventListener('click', () => this.goBack());
        this.els.forward?.addEventListener('click', () => this.goForward());
        this.els.up?.addEventListener('click', () => this.goUp());
        this.els.refresh?.addEventListener('click', () => this.loadFilesForTab());
        
        // 地址栏 - 点击面包屑切换到输入框
        this.els.breadcrumb?.addEventListener('click', (e) => {
            // 如果点击的不是面包屑项（点击空白区域），切换到输入框
            if (!e.target.closest('.fm-breadcrumb-item, .fm-breadcrumb-root')) {
                this.switchToInputMode();
            }
        });
        
        // 地址栏输入框事件
        this.els.path?.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                this.navigate(this.els.path.value);
                this.switchToBreadcrumbMode();
            } else if (e.key === 'Escape') {
                e.preventDefault();
                this.switchToBreadcrumbMode();
            }
        });
        
        // 输入框失焦切回面包屑
        this.els.path?.addEventListener('blur', () => {
            // 延迟执行，避免点击面包屑项时立即切换
            setTimeout(() => this.switchToBreadcrumbMode(), 150);
        });
        
        // 搜索
        let searchTimer;
        this.els.search?.addEventListener('input', (e) => {
            clearTimeout(searchTimer);
            searchTimer = setTimeout(() => this.handleSearch(e.target.value), 300);
        });
        
        // ESC 清除搜索
        this.els.search?.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.els.search.value = '';
                this.clearSearch();
            }
        });
        
        // 视图切换
        this.els.viewBtns?.addEventListener('click', (e) => {
            const btn = e.target.closest('.fm-view-btn');
            if (!btn) return;
            
            const tab = this.getActiveTab();
            if (tab) {
                tab.viewMode = btn.dataset.view;
                this.updateViewButtons();
                this.renderFilesForTab();
            }
        });
        
        // 新建下拉菜单
        this.els.newBtn?.addEventListener('click', (e) => {
            e.stopPropagation();
            const dropdown = this.els.newBtn.parentElement;
            dropdown.classList.toggle('open');
        });
        
        document.addEventListener('click', () => {
            document.querySelectorAll('.fm-dropdown.open').forEach(el => el.classList.remove('open'));
        });
        
        this.els.mkdir?.addEventListener('click', () => this.showMkdirInline());
        this.els.newfile?.addEventListener('click', () => this.showNewFileInline());
        
        // 剪切、复制、粘贴、删除
        this.els.cut?.addEventListener('click', () => this.cutSelected());
        this.els.copy?.addEventListener('click', () => this.copySelected());
        this.els.paste?.addEventListener('click', () => this.paste());
        this.els.delete?.addEventListener('click', () => this.deleteSelected());
        this.els.selectAll?.addEventListener('click', () => this.selectAll());
        
        // 排序
        this.els.sort?.addEventListener('change', (e) => {
            const tab = this.getActiveTab();
            if (tab) {
                tab.sortBy = e.target.value;
                this.renderFilesForTab();
            }
        });
        
        this.els.sortOrder?.addEventListener('click', () => {
            const tab = this.getActiveTab();
            if (tab) {
                tab.sortOrder = tab.sortOrder === 'asc' ? 'desc' : 'asc';
                this.els.sortOrder.innerHTML = tab.sortOrder === 'asc' ? ICONS.sortAsc : ICONS.sortDesc;
                this.renderFilesForTab();
            }
        });
        
        // 拖拽上传
        this.els.content?.addEventListener('dragover', (e) => {
            e.preventDefault();
            this.els.dropOverlay?.classList.add('active');
        });
        
        this.els.content?.addEventListener('dragleave', () => {
            this.els.dropOverlay?.classList.remove('active');
        });
        
        this.els.content?.addEventListener('drop', (e) => {
            e.preventDefault();
            this.els.dropOverlay?.classList.remove('active');
            if (e.dataTransfer?.files?.length) {
                this.uploadFiles(e.dataTransfer.files);
            }
        });
        
        // 右键菜单
        this.els.content?.addEventListener('contextmenu', (e) => {
            e.preventDefault();
            this.showContextMenu(e);
        });
        
        // 键盘快捷键
        this.container.addEventListener('keydown', (e) => {
            if (e.ctrlKey || e.metaKey) {
                if (e.key === 'x') { e.preventDefault(); this.cutSelected(); }
                else if (e.key === 'c') { e.preventDefault(); this.copySelected(); }
                else if (e.key === 'v') { e.preventDefault(); this.paste(); }
                else if (e.key === 'a') { e.preventDefault(); this.selectAll(); }
            } else if (e.key === 'Delete') {
                this.deleteSelected();
            }
        });
    }

    // ... 其他方法保持不变，只需将 this.xxx 改为 tab.xxx ...
    
    // ========== 导航 ==========
    
    async loadFilesForTab(tabId = this.activeTabId) {
        const tab = this.tabs.find(t => t.id === tabId);
        if (!tab) return;
        
        this.els.content.innerHTML = '<div class="fm-empty"><div class="skeleton skeleton-animated" style="height: 200px;"></div></div>';
        
        try {
            const response = await fetch(`${this.options.apiPath}?path=${encodeURIComponent(tab.path)}`);
            const result = await response.json();
            
            if (result?.code === 200 && result.data?.files) {
                tab.files = result.data.files;
                if (result.data.path) tab.path = result.data.path;
                
                // 保存项目目录
                if (result.data.program_dir && !this.programDir) {
                    this.programDir = result.data.program_dir;
                }
                
                this.els.path.value = tab.path;
                this.updateNavButtons();
                this.renderFilesForTab();
                
                // 回调路径变化
                if (this.options.onPathChange) {
                    this.options.onPathChange(tab.path, result.data.program_dir);
                }
            } else {
                this.showError(result?.message || '加载失败');
            }
        } catch (error) {
            this.showError(error.message);
        }
    }
    
    navigate(path) {
        const tab = this.getActiveTab();
        if (!tab) return;
        
        tab.history = tab.history.slice(0, tab.historyIndex + 1);
        tab.history.push(tab.path);
        tab.historyIndex = tab.history.length - 1;
        
        tab.path = path;
        tab.selectedItems.clear();
        this.els.path.value = path;
        this.updateNavButtons();
        this.loadFilesForTab();
    }
    
    goBack() {
        const tab = this.getActiveTab();
        if (!tab || tab.historyIndex <= 0) return;
        
        tab.historyIndex--;
        tab.path = tab.history[tab.historyIndex];
        this.els.path.value = tab.path;
        this.updateNavButtons();
        this.loadFilesForTab();
    }
    
    goForward() {
        const tab = this.getActiveTab();
        if (!tab || tab.historyIndex >= tab.history.length - 1) return;
        
        tab.historyIndex++;
        tab.path = tab.history[tab.historyIndex];
        this.els.path.value = tab.path;
        this.updateNavButtons();
        this.loadFilesForTab();
    }
    
    goUp() {
        const tab = this.getActiveTab();
        if (!tab) return;
        
        // 获取父目录
        const currentPath = tab.path;
        const parentPath = currentPath.split('/').slice(0, -1).join('/') || '/';
        
        if (parentPath !== currentPath) {
            this.navigate(parentPath);
        }
    }
    
    updateNavButtons() {
        const tab = this.getActiveTab();
        if (this.els.back) this.els.back.disabled = !tab || tab.historyIndex <= 0;
        if (this.els.forward) this.els.forward.disabled = !tab || tab.historyIndex >= tab.history.length - 1;
        
        // 向上按钮：根目录时禁用
        if (this.els.up) {
            const currentPath = tab?.path || '';
            this.els.up.disabled = currentPath === '/' || currentPath === '';
        }
        
        // 更新面包屑
        this.updateBreadcrumb(tab?.path || '/');
    }
    
    updateBreadcrumb(path) {
        if (!this.els.breadcrumb) return;
        
        // 标准化路径
        path = path.replace(/\/+/g, '/').replace(/\/$/, '') || '/';
        
        // 如果是相对路径，转换为完整路径显示
        let displayPath = path;
        if ((path === '.' || path === './') && this.programDir) {
            displayPath = this.programDir;
        } else if (path.startsWith('./') && this.programDir) {
            displayPath = this.programDir + path.substring(1);
        }
        
        // 标准化显示路径
        displayPath = displayPath.replace(/\/+/g, '/').replace(/\/$/, '') || '/';
        
        // 分割路径
        const parts = displayPath.split('/').filter(p => p);
        
        // 构建面包屑 HTML
        let html = `<span class="fm-breadcrumb-root" data-path="/">${ICONS.folder}</span>`;
        
        let currentPath = '';
        for (const part of parts) {
            currentPath += '/' + part;
            const normalizedPath = currentPath.replace(/\/+/g, '/');
            html += `<span class="fm-breadcrumb-sep">/</span>`;
            html += `<span class="fm-breadcrumb-item" data-path="${normalizedPath}">${part}</span>`;
        }
        
        this.els.breadcrumb.innerHTML = html;
        
        // 绑定点击事件
        this.els.breadcrumb.querySelectorAll('.fm-breadcrumb-item, .fm-breadcrumb-root').forEach(item => {
            item.addEventListener('click', () => {
                const targetPath = item.dataset.path;
                if (targetPath) {
                    this.navigate(targetPath);
                }
            });
        });
        
        // 绑定双击事件：切换到输入框模式
        this.els.breadcrumb.addEventListener('dblclick', (e) => {
            e.preventDefault();
            this.switchToInputMode();
        });
    }
    
    /**
     * 切换到输入框模式
     */
    switchToInputMode() {
        if (this.addressMode === 'input') return;
        
        this.addressMode = 'input';
        this.els.breadcrumb.style.display = 'none';
        this.els.path.style.display = 'block';
        
        // 显示完整路径
        const tab = this.getActiveTab();
        let fullPath = tab?.path || '.';
        
        // 转换为完整路径
        if ((fullPath === '.' || fullPath === './') && this.programDir) {
            fullPath = this.programDir;
        } else if (fullPath.startsWith('./') && this.programDir) {
            fullPath = this.programDir + fullPath.substring(1);
        }
        
        this.els.path.value = fullPath;
        this.els.path.focus();
        this.els.path.select();
    }
    
    /**
     * 切换到面包屑模式
     */
    switchToBreadcrumbMode() {
        if (this.addressMode === 'breadcrumb') return;
        
        this.addressMode = 'breadcrumb';
        this.els.path.style.display = 'none';
        this.els.breadcrumb.style.display = 'flex';
    }
    
    updateViewButtons() {
        const tab = this.getActiveTab();
        if (!tab || !this.els.viewBtns) return;
        
        this.els.viewBtns.querySelectorAll('.fm-view-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.view === tab.viewMode);
        });
    }
    
    updateToolbarButtons() {
        const tab = this.getActiveTab();
        const hasSelection = tab && tab.selectedItems.size > 0;
        const hasClipboard = this.clipboard.items.length > 0;
        
        if (this.els.cut) this.els.cut.disabled = !hasSelection;
        if (this.els.copy) this.els.copy.disabled = !hasSelection;
        if (this.els.paste) this.els.paste.disabled = !hasClipboard;
        if (this.els.delete) this.els.delete.disabled = !hasSelection;
    }
    
    // ========== 渲染 ==========
    
    renderFilesForTab() {
        const tab = this.getActiveTab();
        if (!tab) return;
        
        // 更新视图按钮状态
        this.updateViewButtons();
        
        // 更新排序选择
        if (this.els.sort) this.els.sort.value = tab.sortBy;
        if (this.els.sortOrder) {
            this.els.sortOrder.innerHTML = tab.sortOrder === 'asc' ? ICONS.sortAsc : ICONS.sortDesc;
        }
        
        if (tab.files.length === 0) {
            this.showEmpty();
            return;
        }
        
        // 排序
        const files = [...tab.files];
        files.sort((a, b) => {
            if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1;
            let cmp = 0;
            switch (tab.sortBy) {
                case 'name': cmp = a.name.localeCompare(b.name); break;
                case 'size': cmp = (a.size || 0) - (b.size || 0); break;
                case 'date': cmp = new Date(a.modified) - new Date(b.modified); break;
            }
            return tab.sortOrder === 'asc' ? cmp : -cmp;
        });
        
        if (tab.viewMode === 'list') {
            this.renderListView(files);
        } else {
            this.renderGridView(files);
        }
    }
    
    renderListView(files) {
        const tab = this.getActiveTab();
        this.els.content.innerHTML = `
            <div class="fm-list-view">
                <div class="fm-list-header">
                    <span>名称</span>
                    <span>大小</span>
                    <span>修改日期</span>
                    <span>类型</span>
                </div>
                <div class="fm-list-body">
                    ${files.map(f => `
                        <div class="fm-list-item ${tab?.selectedItems.has(f.name) ? 'selected' : ''}" data-name="${f.name}" data-is-dir="${f.is_dir}">
                            <div class="fm-list-name">
                                <div class="fm-list-icon ${getIconColorClass(f.name, f.is_dir)}">${getFileIcon(f.name, f.is_dir)}</div>
                                <span class="fm-list-filename">${f.name}</span>
                            </div>
                            <span class="fm-list-size">${f.is_dir ? '-' : formatFileSize(f.size)}</span>
                            <span class="fm-list-date">${formatDate(f.modified)}</span>
                            <span class="fm-list-type">${f.is_dir ? '文件夹' : '文件'}</span>
                        </div>
                    `).join('')}
                </div>
            </div>
        `;
        this.bindFileEvents();
    }
    
    renderGridView(files) {
        const tab = this.getActiveTab();
        this.els.content.innerHTML = `
            <div class="fm-grid-view">
                ${files.map(f => `
                    <div class="fm-grid-item ${tab?.selectedItems.has(f.name) ? 'selected' : ''}" data-name="${f.name}" data-is-dir="${f.is_dir}">
                        <div class="fm-grid-icon ${getIconColorClass(f.name, f.is_dir)}">${getFileIcon(f.name, f.is_dir)}</div>
                        <span class="fm-grid-name">${f.name}</span>
                    </div>
                `).join('')}
            </div>
        `;
        this.bindFileEvents();
    }
    
    bindFileEvents() {
        const tab = this.getActiveTab();
        if (!tab) return;
        
        this.els.content.querySelectorAll('[data-name]').forEach(item => {
            const name = item.dataset.name;
            const isDir = item.dataset.isDir === 'true';
            
            // 单击选择
            item.addEventListener('click', (e) => {
                e.stopPropagation(); // 阻止冒泡
                if (e.ctrlKey || e.metaKey) {
                    if (tab.selectedItems.has(name)) {
                        tab.selectedItems.delete(name);
                        item.classList.remove('selected');
                    } else {
                        tab.selectedItems.add(name);
                        item.classList.add('selected');
                    }
                } else {
                    tab.selectedItems.clear();
                    this.els.content.querySelectorAll('[data-name]').forEach(i => i.classList.remove('selected'));
                    tab.selectedItems.add(name);
                    item.classList.add('selected');
                }
                this.updateToolbarButtons();
            });
            
            // 双击打开
            item.addEventListener('dblclick', (e) => {
                e.stopPropagation();
                if (isDir) {
                    this.navigate(this.joinPath(tab.path, name));
                } else {
                    // 文件：打开编辑器
                    this.editFile(name);
                }
            });
        });
        
        // 点击空白处取消选中
        this.els.content.addEventListener('click', (e) => {
            // 只有点击 content 本身才取消选中
            if (e.target === this.els.content || e.target.classList.contains('fm-list-body') || e.target.classList.contains('fm-list-view') || e.target.classList.contains('fm-grid-view')) {
                tab.selectedItems.clear();
                this.els.content.querySelectorAll('[data-name]').forEach(i => i.classList.remove('selected'));
                this.updateToolbarButtons();
            }
        }, { once: true }); // once: true 防止重复绑定
    }
    
    // ========== 搜索功能 ==========

    handleSearch(keyword) {
        const tab = this.getActiveTab();
        if (!tab) return;
        
        const recursive = this.els.searchRecursive?.checked;
        
        if (!keyword || keyword.trim() === '') {
            this.clearSearch();
            return;
        }
        
        if (recursive) {
            this.searchRecursive(keyword);
        } else {
            this.searchLocal(keyword);
        }
    }
    
    searchLocal(keyword) {
        const tab = this.getActiveTab();
        if (!tab) return;
        
        const lower = keyword.toLowerCase();
        const results = tab.files.filter(f => 
            f.name.toLowerCase().includes(lower)
        );
        
        if (results.length === 0) {
            this.showEmpty(`未找到 "${keyword}"`);
        } else {
            this.renderSearchResults(results, keyword);
            this.options.toast?.info?.(`找到 ${results.length} 个结果`);
        }
    }
    
    async searchRecursive(keyword) {
        const tab = this.getActiveTab();
        if (!tab) return;
        
        this.els.content.innerHTML = `
            <div class="fm-empty">
                <div class="skeleton skeleton-animated" style="height: 200px;"></div>
                <p style="margin-top: 16px; color: var(--text-secondary);">正在搜索 "${keyword}"...</p>
            </div>
        `;
        
        try {
            const response = await fetch(
                `${this.options.apiPath}/search?` + new URLSearchParams({
                    path: tab.path,
                    keyword: keyword,
                    recursive: 'true'
                })
            );
            
            const result = await response.json();
            
            if (result?.code === 200 && result.data?.results) {
                const results = result.data.results;
                
                if (results.length === 0) {
                    this.showEmpty(`未找到 "${keyword}"`);
                } else {
                    this.renderSearchResults(results, keyword, true);
                    this.options.toast?.success?.(`找到 ${results.length} 个结果`);
                }
            } else {
                this.options.toast?.warning?.('后端暂不支持递归搜索');
                this.searchLocal(keyword);
            }
        } catch (error) {
            this.options.toast?.warning?.('搜索失败，仅搜索当前目录');
            this.searchLocal(keyword);
        }
    }
    
    renderSearchResults(results, keyword, isRecursive = false) {
        const tab = this.getActiveTab();
        if (!tab) return;
        
        results.sort((a, b) => {
            if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1;
            return a.name.localeCompare(b.name);
        });
        
        if (tab.viewMode === 'list') {
            this.els.content.innerHTML = `
                <div class="fm-list-view">
                    <div class="fm-list-header">
                        <span>名称</span>
                        <span>大小</span>
                        <span>修改日期</span>
                        <span>类型</span>
                    </div>
                    ${results.map(f => `
                        <div class="fm-list-item" data-name="${f.name}" data-is-dir="${f.is_dir}">
                            <div class="fm-list-name">
                                <div class="fm-list-icon ${getIconColorClass(f.name, f.is_dir)}">${getFileIcon(f.name, f.is_dir)}</div>
                                <span class="fm-list-filename">${this.highlightMatch(f.name, keyword)}</span>
                            </div>
                            <span class="fm-list-size">${f.is_dir ? '-' : formatFileSize(f.size)}</span>
                            <span class="fm-list-date">${formatDate(f.modified)}</span>
                            <span class="fm-list-type">${f.is_dir ? '文件夹' : '文件'}</span>
                        </div>
                    `).join('')}
                </div>
            `;
        } else {
            this.els.content.innerHTML = `
                <div class="fm-grid-view">
                    ${results.map(f => `
                        <div class="fm-grid-item" data-name="${f.name}" data-is-dir="${f.is_dir}">
                            <div class="fm-grid-icon ${getIconColorClass(f.name, f.is_dir)}">${getFileIcon(f.name, f.is_dir)}</div>
                            <span class="fm-grid-name">${this.highlightMatch(f.name, keyword)}</span>
                        </div>
                    `).join('')}
                </div>
            `;
        }
        
        this.bindFileEvents();
    }
    
    highlightMatch(text, keyword) {
        if (!keyword) return text;
        const regex = new RegExp(`(${keyword.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
        return text.replace(regex, '<mark class="fm-search-highlight">$1</mark>');
    }
    
    clearSearch() {
        this.renderFilesForTab();
    }
    
    filterFiles(keyword) {
        this.handleSearch(keyword);
    }

    // ========== 操作 ==========
    
    selectAll() {
        const tab = this.getActiveTab();
        if (!tab) return;
        
        // 如果已经全选，则取消全选
        if (tab.selectedItems.size === tab.files.length) {
            tab.selectedItems.clear();
            this.options.toast?.info?.('已取消全选');
        } else {
            // 全选
            tab.files.forEach(f => tab.selectedItems.add(f.name));
            this.options.toast?.success?.(`已全选 ${tab.files.length} 个项目`);
        }
        
        this.els.content.querySelectorAll('[data-name]').forEach(item => {
            item.classList.toggle('selected', tab.selectedItems.has(item.dataset.name));
        });
        this.updateToolbarButtons();
    }
    
    showMkdirInline() {
        const tab = this.getActiveTab();
        if (!tab) return;
        
        let defaultName = '新建文件夹';
        let counter = 1;
        while (tab.files.some(f => f.name === defaultName)) {
            defaultName = `新建文件夹 (${counter++})`;
        }
        
        tab.files.unshift({ name: defaultName, is_dir: true, size: 0, modified: new Date().toISOString(), _isNew: true });
        this.renderFilesForTab();
        
        setTimeout(() => {
            const item = this.els.content.querySelector(`[data-name="${defaultName}"]`);
            if (item) this.enterEditMode(item, defaultName, false);
        }, 0);
    }
    
    showNewFileInline() {
        const tab = this.getActiveTab();
        if (!tab) return;
        
        let defaultName = '新建文件.txt';
        let counter = 1;
        while (tab.files.some(f => f.name === defaultName)) {
            defaultName = `新建文件 (${counter++}).txt`;
        }
        
        tab.files.unshift({ name: defaultName, is_dir: false, size: 0, modified: new Date().toISOString(), _isNew: true });
        this.renderFilesForTab();
        
        setTimeout(() => {
            const item = this.els.content.querySelector(`[data-name="${defaultName}"]`);
            if (item) this.enterEditMode(item, defaultName, true);
        }, 0);
    }
    
    /**
     * 验证文件/文件夹名称
     * @returns { valid: boolean, error: string|null }
     */
    validateFileName(name, isFile) {
        if (!name || name.trim() === '') {
            return { valid: false, error: '名称不能为空' };
        }
        
        // 非法字符
        const illegalChars = /[\/\\:*?"<>|]/;
        if (illegalChars.test(name)) {
            return { valid: false, error: '名称不能包含 / \\ : * ? " < > | 这些字符' };
        }
        
        // 不能以点或空格开头/结尾
        if (name.startsWith('.') || name.startsWith(' ')) {
            return { valid: false, error: '名称不能以点或空格开头' };
        }
        if (name.endsWith('.') || name.endsWith(' ')) {
            return { valid: false, error: '名称不能以点或空格结尾' };
        }
        
        // Windows 保留名称
        const reservedNames = ['CON', 'PRN', 'AUX', 'NUL', 
            'COM1', 'COM2', 'COM3', 'COM4', 'COM5', 'COM6', 'COM7', 'COM8', 'COM9',
            'LPT1', 'LPT2', 'LPT3', 'LPT4', 'LPT5', 'LPT6', 'LPT7', 'LPT8', 'LPT9'];
        const upperName = name.toUpperCase().split('.')[0]; // 去掉扩展名检查
        if (reservedNames.includes(upperName)) {
            return { valid: false, error: `"${upperName}" 是系统保留名称，不可使用` };
        }
        
        // 名称长度限制
        if (name.length > 255) {
            return { valid: false, error: '名称长度不能超过 255 个字符' };
        }
        
        return { valid: true, error: null };
    }
    
    async createEmptyFile(fileName) {
        const tab = this.getActiveTab();
        if (!tab) return;

        try {
            // 直接调用后端的 touchFile API
            const response = await fetch(`${this.options.apiPath}/touch`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    path: tab.path,
                    name: fileName
                })
            });

            const result = await response.json();
            if (result?.code === 200) {
                this.options.toast?.success?.('文件创建成功');
                this.loadFilesForTab();
            } else {
                throw new Error(result?.message || '创建失败');
            }
        } catch (error) {
            this.options.toast?.error?.('创建文件失败: ' + error.message);
        }
    }
    
    enterEditMode(item, originalName, isFile) {
        const nameEl = item.querySelector('.fm-list-filename, .fm-grid-name');
        if (!nameEl) return;
        
        const input = document.createElement('input');
        input.type = 'text';
        input.className = 'fm-edit-input';
        input.value = originalName;
        
        nameEl.style.display = 'none';
        nameEl.parentElement.insertBefore(input, nameEl);
        input.focus();
        input.select();
        
        const confirm = async () => {
            const newName = input.value.trim() || originalName;
            const tab = this.getActiveTab();
            
            // 验证名称
            const validation = this.validateFileName(newName, isFile);
            if (!validation.valid) {
                this.options.toast?.error?.(validation.error);
                input.focus();
                input.select();
                return;
            }
            
            // 检查是否重名
            if (tab.files.some(f => f.name === newName && f.name !== originalName)) {
                this.options.toast?.error?.('已存在同名文件或文件夹');
                input.focus();
                input.select();
                return;
            }
            
            try {
                let response;
                if (isFile) {
                    // 创建文件使用 /touch
                    response = await fetch(`${this.options.apiPath}/touch`, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ path: tab.path, name: newName })
                    });
                } else {
                    // 创建文件夹使用 /mkdir
                    response = await fetch(`${this.options.apiPath}/mkdir`, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ path: this.joinPath(tab.path, newName) })
                    });
                }
                
                const result = await response.json();
                if (result?.code !== 200) throw new Error(result?.message || '创建失败');
                
                const file = tab.files.find(f => f.name === originalName);
                if (file) { file.name = newName; file._isNew = false; }
                
                nameEl.textContent = newName;
                nameEl.style.display = '';
                item.dataset.name = newName;
                input.remove();
                
                this.options.toast?.success?.('创建成功');
            } catch (error) {
                this.options.toast?.error?.(error.message);
                tab.files = tab.files.filter(f => f.name !== originalName);
                item.remove();
            }
        };
        
        input.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') { e.preventDefault(); confirm(); }
            else if (e.key === 'Escape') { 
                e.preventDefault();
                const tab = this.getActiveTab();
                tab.files = tab.files.filter(f => f.name !== originalName);
                item.remove();
            }
        });
        
        input.addEventListener('blur', () => setTimeout(() => document.body.contains(input) && confirm(), 100));
    }
    
    cutSelected() {
        const tab = this.getActiveTab();
        if (!tab || tab.selectedItems.size === 0) return;
        
        this.clipboard = { type: 'cut', items: Array.from(tab.selectedItems), sourcePath: tab.path };
        this.options.toast?.success?.(`已剪切 ${tab.selectedItems.size} 个项目`);
        this.updateToolbarButtons();
    }
    
    copySelected() {
        const tab = this.getActiveTab();
        if (!tab || tab.selectedItems.size === 0) return;
        
        this.clipboard = { type: 'copy', items: Array.from(tab.selectedItems), sourcePath: tab.path };
        this.options.toast?.success?.(`已复制 ${tab.selectedItems.size} 个项目`);
        this.updateToolbarButtons();
    }
    
    async paste() {
        if (this.clipboard.items.length === 0) return;
        
        const tab = this.getActiveTab();
        if (!tab) return;

        const isCut = this.clipboard.type === 'cut';
        const isSameDir = this.clipboard.sourcePath === tab.path;
        let success = 0;
        let failed = 0;
        
        for (const name of this.clipboard.items) {
            try {
                let dstName = name;
                
                // 同目录复制：生成新名称
                if (!isCut && isSameDir) {
                    dstName = this.generateCopyName(name, tab.files);
                }
                
                if (isCut) {
                    // 剪切 -> 移动
                    const response = await fetch(`${this.options.apiPath}/move`, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            srcPath: this.clipboard.sourcePath,
                            srcName: name,
                            dstPath: tab.path
                        })
                    });
                    const result = await response.json();
                    if (result?.code === 200) {
                        success++;
                    } else {
                        failed++;
                    }
                } else {
                    // 复制
                    const response = await fetch(`${this.options.apiPath}/copy`, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            srcPath: this.clipboard.sourcePath,
                            srcName: name,
                            dstPath: tab.path,
                            dstName: dstName !== name ? dstName : undefined
                        })
                    });
                    const result = await response.json();
                    if (result?.code === 200) {
                        success++;
                    } else {
                        failed++;
                    }
                }
            } catch (error) {
                failed++;
            }
        }
        
        // 剪切操作完成后清空剪贴板
        if (isCut) {
            this.clipboard = { type: null, items: [], sourcePath: null };
            this.updateToolbarButtons();
        }
        
        if (success > 0) {
            this.options.toast?.success?.(`${isCut ? '移动' : '复制'}完成: ${success} 个文件`);
        }
        if (failed > 0) {
            this.options.toast?.error?.(`${failed} 个文件失败`);
        }
        
        this.loadFilesForTab();
    }
    
    /**
     * 生成复制文件的新名称
     */
    generateCopyName(name, files) {
        const ext = name.includes('.') ? '.' + name.split('.').pop() : '';
        const baseName = ext ? name.slice(0, -ext.length) : name;
        
        let counter = 1;
        let newName = `${baseName} (${counter})${ext}`;
        
        while (files.some(f => f.name === newName)) {
            counter++;
            newName = `${baseName} (${counter})${ext}`;
        }
        
        return newName;
    }
    
    deleteSelected() {
        const tab = this.getActiveTab();
        if (!tab || tab.selectedItems.size === 0) return;
        
        const count = tab.selectedItems.size;
        const items = Array.from(tab.selectedItems);
        
        // 使用自定义确认对话框
        this.showConfirmDialog('删除确认', `确定要删除选中的 ${count} 个项目吗？此操作不可恢复。`)
            .then(async (confirmed) => {
                if (!confirmed) return;
                
                // 防止重复操作
                if (this._deleting) return;
                this._deleting = true;
                
                let success = 0;
                let failed = 0;
                
                try {
                    // 批量删除（限制并发）
                    const batchSize = 5;
                    for (let i = 0; i < items.length; i += batchSize) {
                        const batch = items.slice(i, i + batchSize);
                        const results = await Promise.all(batch.map(name => 
                            fetch(`${this.options.apiPath}/delete`, {
                                method: 'POST',
                                headers: { 'Content-Type': 'application/json' },
                                body: JSON.stringify({ path: tab.path, name: name })
                            }).then(r => r.ok).catch(() => false)
                        ));
                        
                        results.forEach(ok => ok ? success++ : failed++);
                    }
                    
                    tab.selectedItems.clear();
                    this.updateToolbarButtons();
                    
                    if (failed === 0) {
                        this.options.toast?.success?.(`成功删除 ${success} 个项目`);
                    } else {
                        this.options.toast?.warning?.(`成功 ${success} 个，失败 ${failed} 个`);
                    }
                    
                    // 刷新文件列表
                    await this.loadFilesForTab();
                } catch (error) {
                    this.options.toast?.error?.('删除失败: ' + error.message);
                } finally {
                    this._deleting = false;
                }
            });
    }
    
    uploadFiles(files) {
        const tab = this.getActiveTab();
        if (!tab || !files?.length) return;
        
        const formData = new FormData();
        formData.append('path', tab.path);
        for (const file of files) formData.append('files', file);
        
        fetch(`${this.options.apiPath}/upload`, { method: 'POST', body: formData })
            .then(() => { this.options.toast?.success?.('上传成功'); this.loadFilesForTab(); })
            .catch((e) => this.options.toast?.error?.('上传失败: ' + e.message));
        
        if (this.els.fileInput) this.els.fileInput.value = '';
    }
    
    showContextMenu(e) {
        const tab = this.getActiveTab();
        if (!tab) return;

        // 判断点击的目标（支持列表和网格视图）
        const fileItem = e.target.closest('.fm-list-item, .fm-grid-item');
        const isOnFile = !!fileItem;
        const selectedItems = Array.from(tab.selectedItems);

        // 如果右键的文件不在选中列表中，清空选中并只选中当前
        if (isOnFile && fileItem) {
            const fileName = fileItem.dataset.name;
            if (!selectedItems.includes(fileName)) {
                tab.selectedItems.clear();
                tab.selectedItems.add(fileName);
                this.renderFilesForTab();
                selectedItems.length = 0;
                selectedItems.push(fileName);
            }
        }

        // 构建菜单项
        let menuItems = [];

        if (isOnFile && selectedItems.length > 0) {
            // 文件/文件夹上右键
            const firstItem = tab.files.find(f => f.name === selectedItems[0]);
            const isSingleFile = selectedItems.length === 1;
            const isDir = firstItem?.is_dir;

            menuItems = [
                {
                    label: '打开',
                    icon: isDir ? ICONS.folder : ICONS.file,
                    action: 'open',
                    onClick: () => {
                        if (isDir) {
                            this.navigate(this.joinPath(tab.path, selectedItems[0]));
                        } else {
                            this.openFile(selectedItems[0]);
                        }
                    }
                },
                { divider: true }
            ];

            // 单个文件才显示编辑
            if (isSingleFile && !isDir && this.isEditableFile(selectedItems[0])) {
                menuItems.push({
                    label: '编辑',
                    icon: ICONS.edit,
                    action: 'edit',
                    onClick: () => this.editFile(selectedItems[0])
                });
            }

            menuItems.push({
                label: '重命名',
                icon: ICONS.rename,
                action: 'rename',
                disabled: !isSingleFile,
                onClick: () => this.renameFile(selectedItems[0])
            });

            menuItems.push({ divider: true });

            menuItems.push(
                {
                    label: '复制',
                    icon: ICONS.copy,
                    action: 'copy',
                    onClick: () => this.copySelected()
                },
                {
                    label: '剪切',
                    icon: ICONS.cut,
                    action: 'cut',
                    onClick: () => this.cutSelected()
                },
                {
                    label: '删除',
                    icon: ICONS.delete,
                    action: 'delete',
                    onClick: () => this.deleteSelected()
                }
            );

            // 权限（仅 Linux）
            const isWindows = navigator.userAgent.includes('Windows');
            if (!isWindows) {
                menuItems.push({ divider: true });
                menuItems.push({
                    label: '权限',
                    icon: ICONS.chmod,
                    action: 'chmod',
                    onClick: () => this.showChmodDialog(selectedItems)
                });
            }

            // 压缩
            menuItems.push({ divider: true });
            menuItems.push({
                label: '压缩',
                icon: ICONS.compress,
                action: 'compress',
                onClick: () => this.compressFiles(selectedItems)
            });

            // 解压（单个压缩文件才显示）
            if (isSingleFile && !isDir && this.isArchiveFile(selectedItems[0])) {
                menuItems.push({
                    label: '解压',
                    icon: ICONS.extract,
                    action: 'extract',
                    onClick: () => this.extractFile(selectedItems[0])
                });
            }

            menuItems.push({ divider: true });

            // 下载
            if (!isDir || isSingleFile) {
                menuItems.push({
                    label: '下载',
                    icon: ICONS.download,
                    action: 'download',
                    onClick: () => this.downloadFile(selectedItems[0])
                });
            }

            // 外链分享
            menuItems.push({
                label: '外链分享',
                icon: ICONS.share,
                action: 'share',
                onClick: () => this.shareFile(selectedItems[0])
            });

            menuItems.push({ divider: true });

            // 复制路径（单个文件）
            if (isSingleFile) {
                menuItems.push(
                    {
                        label: '复制文件名',
                        icon: ICONS.copyPath,
                        action: 'copyName',
                        onClick: () => {
                            this.copyToClipboard(selectedItems[0], '文件名');
                        }
                    },
                    {
                        label: '复制路径',
                        icon: ICONS.copyPath,
                        action: 'copyPath',
                        onClick: () => {
                            const fullPath = this.joinPath(tab.path, selectedItems[0]);
                            this.copyToClipboard(fullPath, '路径');
                        }
                    }
                );
            }

        } else {
            // 空白区域右键
            menuItems = [
                {
                    label: '刷新',
                    icon: ICONS.refresh,
                    action: 'refresh',
                    onClick: () => this.loadFilesForTab()
                },
                { divider: true },
                {
                    label: '粘贴',
                    icon: ICONS.paste,
                    action: 'paste',
                    disabled: this.clipboard.items.length === 0,
                    onClick: () => this.paste()
                },
                { divider: true },
                {
                    label: '新建文件夹',
                    icon: ICONS.mkdir,
                    action: 'mkdir',
                    onClick: () => this.showMkdirInline()
                },
                {
                    label: '新建文件',
                    icon: ICONS.file,
                    action: 'newfile',
                    onClick: () => this.showNewFileInline()
                },
                { divider: true },
                {
                    label: '全选',
                    icon: ICONS.selectAll,
                    action: 'selectAll',
                    onClick: () => this.selectAll()
                }
            ];
        }

        // 显示菜单
        contextMenu.show(e.clientX, e.clientY, menuItems, { tab, selectedItems });
    }

    /**
     * 判断是否为可编辑文件
     */
    isEditableFile(filename) {
        const ext = filename.split('.').pop().toLowerCase();
        const editableExts = ['txt', 'json', 'md', 'yml', 'yaml', 'xml', 'html', 'css', 'js', 'ts', 'go', 'py', 'sh', 'conf', 'cfg', 'ini', 'log', 'env'];
        return editableExts.includes(ext);
    }

    /**
     * 判断是否为压缩文件
     */
    isArchiveFile(filename) {
        const ext = filename.split('.').pop().toLowerCase();
        return ['zip', 'tar', 'gz', 'tgz', 'tar.gz', 'rar', '7z'].includes(ext);
    }

    /**
     * 编辑文件
     */
    async editFile(filename) {
        const tab = this.getActiveTab();
        if (!tab) return;

        try {
            // 读取文件内容
            const response = await fetch(`${this.options.apiPath}/read?path=${encodeURIComponent(tab.path)}&name=${encodeURIComponent(filename)}`);
            const result = await response.json();
            
            if (result?.code !== 200) {
                this.options.toast?.error?.(result?.message || '读取文件失败');
                return;
            }

            // 显示编辑器
            this.showEditor(filename, result.data);
        } catch (error) {
            this.options.toast?.error?.('读取文件失败: ' + error.message);
        }
    }

    /**
     * 显示编辑器弹窗
     */
    showEditor(filename, data) {
        const tab = this.getActiveTab();
        
        // 检查数据
        if (!data) {
            this.options.toast?.error?.('无法读取文件数据');
            return;
        }
        
        const content = data.content || '';
        const fileSize = data.size || 0;
        const fileModified = data.modified || new Date().toISOString();
        const fileType = data.type || 'text';
        const lines = content.split('\n');
        
        // 判断是否需要代码高亮风格
        const codeTypes = ['json', 'javascript', 'typescript', 'go', 'python', 'css', 'html', 'xml', 'yaml', 'shell', 'sql', 'java', 'c', 'cpp', 'rust', 'php', 'ruby', 'lua', 'conf', 'ini', 'env', 'sh', 'bash'];
        const isCodeFile = codeTypes.includes(fileType);
        
        // 创建编辑器弹窗
        const overlay = document.createElement('div');
        overlay.className = 'editor-overlay';
        overlay.innerHTML = `
            <div class="editor-dialog ${isCodeFile ? 'code-editor' : ''}">
                <div class="editor-header">
                    <div class="editor-drag-area">
                        <div class="editor-title">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18">
                                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                            </svg>
                            <span class="editor-filename">${this.escapeHtml(filename)}</span>
                            <span class="editor-type-badge">${fileType}</span>
                        </div>
                    </div>
                    <div class="editor-toolbar">
                        <button class="editor-tool-btn" id="editor-format" title="格式化 (Ctrl+Shift+F)">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                                <line x1="21" y1="10" x2="3" y2="10"/>
                                <line x1="21" y1="6" x2="3" y2="6"/>
                                <line x1="21" y1="14" x2="3" y2="14"/>
                                <line x1="21" y1="18" x2="3" y2="18"/>
                            </svg>
                        </button>
                        <button class="editor-tool-btn primary" id="editor-save" title="保存 (Ctrl+S)">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                                <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/>
                                <polyline points="17 21 17 13 7 13 7 21"/>
                                <polyline points="7 3 7 8 15 8"/>
                            </svg>
                        </button>
                        <div class="editor-divider"></div>
                        <button class="editor-tool-btn" id="editor-maximize" title="最大化">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                                <polyline points="15 3 21 3 21 9"/>
                                <polyline points="9 21 3 21 3 15"/>
                                <line x1="21" y1="3" x2="14" y2="10"/>
                                <line x1="3" y1="21" x2="10" y2="14"/>
                            </svg>
                        </button>
                        <button class="editor-tool-btn close" id="editor-close" title="关闭">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                                <line x1="18" y1="6" x2="6" y2="18"/>
                                <line x1="6" y1="6" x2="18" y2="18"/>
                            </svg>
                        </button>
                    </div>
                </div>
                <div class="editor-body">
                    <div class="editor-container">
                        <div class="editor-gutter" id="editor-gutter"></div>
                        <textarea class="editor-textarea" id="editor-content" spellcheck="false" wrap="off"></textarea>
                    </div>
                </div>
                <div class="editor-footer">
                    <span class="editor-info">大小: ${this.formatSize(fileSize)}</span>
                    <span class="editor-info">修改: ${new Date(fileModified).toLocaleString()}</span>
                    <span class="editor-info">行数: ${lines.length}</span>
                    <span class="editor-info cursor-pos" id="editor-cursor">行 1, 列 1</span>
                </div>
            </div>
        `;

        document.body.appendChild(overlay);

        // 缓存元素
        const dialog = overlay.querySelector('.editor-dialog');
        const textarea = overlay.querySelector('#editor-content');
        const gutter = overlay.querySelector('#editor-gutter');
        const cursorPos = overlay.querySelector('#editor-cursor');
        const saveBtn = overlay.querySelector('#editor-save');
        const closeBtn = overlay.querySelector('#editor-close');
        const maximizeBtn = overlay.querySelector('#editor-maximize');
        const formatBtn = overlay.querySelector('#editor-format');

        // 使用 value 设置内容
        textarea.value = content;

        // 更新行号
        const updateLineNumbers = () => {
            const lineCount = textarea.value.split('\n').length;
            let html = '';
            for (let i = 1; i <= lineCount; i++) {
                html += `<div class="editor-line-num">${i}</div>`;
            }
            gutter.innerHTML = html;
        };
        
        // 同步滚动
        const syncScroll = () => {
            gutter.scrollTop = textarea.scrollTop;
        };
        
        // 更新光标位置
        const updateCursorPosition = () => {
            const value = textarea.value;
            const selectionStart = textarea.selectionStart;
            const lines = value.substring(0, selectionStart).split('\n');
            const line = lines.length;
            const col = lines[lines.length - 1].length + 1;
            cursorPos.textContent = `行 ${line}, 列 ${col}`;
        };

        // 初始化行号
        updateLineNumbers();
        
        // 事件绑定
        textarea.addEventListener('input', updateLineNumbers);
        textarea.addEventListener('scroll', syncScroll);
        textarea.addEventListener('click', updateCursorPosition);
        textarea.addEventListener('keyup', updateCursorPosition);
        
        // Tab 键支持
        textarea.addEventListener('keydown', (e) => {
            if (e.key === 'Tab') {
                e.preventDefault();
                const start = textarea.selectionStart;
                const end = textarea.selectionEnd;
                textarea.value = textarea.value.substring(0, start) + '    ' + textarea.value.substring(end);
                textarea.selectionStart = textarea.selectionEnd = start + 4;
                updateLineNumbers();
            }
        });

        // 保存快捷键
        const handleKeydown = (e) => {
            if ((e.ctrlKey || e.metaKey) && e.key === 's') {
                e.preventDefault();
                this.saveEditorContent(filename, textarea.value, overlay, tab);
            }
            if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'F') {
                e.preventDefault();
                this.formatContent(textarea, fileType);
            }
            if (e.key === 'Escape') {
                e.preventDefault();
                this.confirmCloseEditor(overlay, textarea, content);
            }
        };
        
        textarea.addEventListener('keydown', handleKeydown);
        overlay.addEventListener('keydown', handleKeydown);

        // 保存按钮
        saveBtn.addEventListener('click', () => {
            this.saveEditorContent(filename, textarea.value, overlay, tab);
        });

        // 关闭按钮
        closeBtn.addEventListener('click', () => {
            this.confirmCloseEditor(overlay, textarea, content);
        });

        // 最大化按钮
        let isMaximized = false;
        maximizeBtn.addEventListener('click', () => {
            isMaximized = !isMaximized;
            if (isMaximized) {
                dialog.classList.add('maximized');
                maximizeBtn.innerHTML = `
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                        <polyline points="4 14 10 14 10 20"/>
                        <polyline points="20 10 14 10 14 4"/>
                        <line x1="14" y1="10" x2="21" y2="3"/>
                        <line x1="3" y1="21" x2="10" y2="14"/>
                    </svg>
                `;
            } else {
                dialog.classList.remove('maximized');
                maximizeBtn.innerHTML = `
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                        <polyline points="15 3 21 3 21 9"/>
                        <polyline points="9 21 3 21 3 15"/>
                        <line x1="21" y1="3" x2="14" y2="10"/>
                        <line x1="3" y1="21" x2="10" y2="14"/>
                    </svg>
                `;
            }
        });

        // 格式化按钮
        formatBtn.addEventListener('click', () => {
            this.formatContent(textarea, fileType);
            updateLineNumbers();
        });

        // 拖动功能（居中显示）
        this.makeDraggable(overlay, dialog, overlay.querySelector('.editor-drag-area'));

        // 聚焦到编辑器
        textarea.focus();
        
        // 初始化光标位置
        updateCursorPosition();
    }

    /**
     * 格式化内容
     */
    formatContent(textarea, fileType) {
        const content = textarea.value;
        let formatted = content;
        
        try {
            if (fileType === 'json') {
                formatted = JSON.stringify(JSON.parse(content), null, 2);
            } else if (fileType === 'html' || fileType === 'xml') {
                // 简单的 HTML/XML 格式化
                formatted = content
                    .replace(/></g, '>\n<')
                    .replace(/^\s+|\s+$/gm, '');
            }
            textarea.value = formatted;
            this.options.toast?.success?.('格式化完成');
        } catch (error) {
            this.options.toast?.error?.('格式化失败: ' + error.message);
        }
    }

    /**
     * 使元素可拖动
     */
    makeDraggable(overlay, dialog, dragArea) {
        let isDragging = false;
        let startX, startY, initialLeft, initialTop;
        
        // 计算初始居中位置（先让 dialog 渲染，获取尺寸）
        requestAnimationFrame(() => {
            const rect = dialog.getBoundingClientRect();
            initialLeft = (window.innerWidth - rect.width) / 2;
            initialTop = (window.innerHeight - rect.height) / 2;
            dialog.style.left = initialLeft + 'px';
            dialog.style.top = initialTop + 'px';
        });

        dragArea.addEventListener('mousedown', (e) => {
            isDragging = true;
            startX = e.clientX;
            startY = e.clientY;
            
            const rect = dialog.getBoundingClientRect();
            initialLeft = rect.left;
            initialTop = rect.top;
            
            dragArea.style.cursor = 'grabbing';
            e.preventDefault();
        });

        document.addEventListener('mousemove', (e) => {
            if (!isDragging) return;
            
            const deltaX = e.clientX - startX;
            const deltaY = e.clientY - startY;
            
            const newLeft = initialLeft + deltaX;
            const newTop = initialTop + deltaY;
            
            // 边界限制
            const maxLeft = window.innerWidth - 100;
            const maxTop = window.innerHeight - 50;
            
            dialog.style.left = Math.max(-dialog.offsetWidth + 100, Math.min(maxLeft, newLeft)) + 'px';
            dialog.style.top = Math.max(0, Math.min(maxTop, newTop)) + 'px';
        });

        document.addEventListener('mouseup', () => {
            if (isDragging) {
                isDragging = false;
                dragArea.style.cursor = 'move';
            }
        });
    }

    /**
     * 确认关闭编辑器（检查未保存更改）
     */
    async confirmCloseEditor(overlay, textarea, originalContent) {
        // 如果已保存，直接关闭
        if (overlay.dataset.saved === 'true') {
            this.closeEditor(overlay);
            return;
        }
        
        const currentContent = textarea.value;
        
        // 检查是否有未保存的更改
        if (currentContent !== originalContent) {
            const confirmed = await this.showConfirmDialog('关闭编辑器', '有未保存的更改，确定要关闭吗？');
            if (!confirmed) return;
        }
        
        this.closeEditor(overlay);
    }

    /**
     * 保存编辑器内容
     */
    async saveEditorContent(filename, content, overlay, tab) {
        const saveBtn = overlay.querySelector('#editor-save');
        saveBtn.disabled = true;
        saveBtn.innerHTML = '<span class="editor-loading"></span> 保存中...';

        try {
            const response = await fetch(`${this.options.apiPath}/save`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    path: tab.path,
                    name: filename,
                    content: content
                })
            });

            const result = await response.json();
            if (result?.code === 200) {
                this.options.toast?.success?.('保存成功');
                // 标记已保存
                overlay.dataset.saved = 'true';
                // 更新页脚信息
                const footer = overlay.querySelector('.editor-footer');
                const lines = content.split('\n').length;
                const size = new Blob([content]).size;
                footer.innerHTML = `
                    <span class="editor-info">大小: ${this.formatSize(size)}</span>
                    <span class="editor-info">已保存</span>
                    <span class="editor-info">行数: ${lines}</span>
                `;
            } else {
                this.options.toast?.error?.(result?.message || '保存失败');
            }
        } catch (error) {
            this.options.toast?.error?.('保存失败: ' + error.message);
        } finally {
            saveBtn.disabled = false;
            saveBtn.innerHTML = `
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                    <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/>
                    <polyline points="17 21 17 13 7 13 7 21"/>
                    <polyline points="7 3 7 8 15 8"/>
                </svg>
                保存
            `;
        }
    }

    /**
     * 关闭编辑器
     */
    closeEditor(overlay) {
        overlay.remove();
    }

    /**
     * HTML 转义
     */
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    /**
     * 格式化文件大小
     */
    formatSize(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    /**
     * 重命名文件 - 使用内联编辑
     */
    async renameFile(oldName) {
        const tab = this.getActiveTab();
        if (!tab) return;
        
        // 找到文件项
        const item = this.els.content.querySelector(`[data-name="${oldName}"]`);
        if (!item) return;
        
        // 判断是文件还是文件夹
        const fileItem = tab.files.find(f => f.name === oldName);
        const isFile = fileItem && !fileItem.is_dir;
        
        // 进入内联编辑模式
        const nameEl = item.querySelector('.fm-list-filename, .fm-grid-name');
        if (!nameEl) return;
        
        const input = document.createElement('input');
        input.type = 'text';
        input.className = 'fm-edit-input';
        input.value = oldName;
        
        nameEl.style.display = 'none';
        nameEl.parentElement.insertBefore(input, nameEl);
        input.focus();
        input.select();
        
        const confirm = async () => {
            const newName = input.value.trim() || oldName;
            
            // 名称未改变
            if (newName === oldName) {
                nameEl.style.display = '';
                input.remove();
                return;
            }
            
            // 验证名称
            const validation = this.validateFileName(newName, isFile);
            if (!validation.valid) {
                this.options.toast?.error?.(validation.error);
                input.focus();
                input.select();
                return;
            }
            
            // 检查是否重名
            if (tab.files.some(f => f.name === newName && f.name !== oldName)) {
                this.options.toast?.error?.('已存在同名文件或文件夹');
                input.focus();
                input.select();
                return;
            }

            try {
                const response = await fetch(`${this.options.apiPath}/rename`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        path: tab.path,
                        oldName: oldName,
                        newName: newName
                    })
                });
                const result = await response.json();
                if (result?.code === 200) {
                    // 更新本地数据
                    const file = tab.files.find(f => f.name === oldName);
                    if (file) file.name = newName;
                    
                    nameEl.textContent = newName;
                    nameEl.style.display = '';
                    item.dataset.name = newName;
                    input.remove();
                    
                    this.options.toast?.success?.('重命名成功');
                } else {
                    this.options.toast?.error?.(result?.message || '重命名失败');
                    input.focus();
                    input.select();
                }
            } catch (error) {
                this.options.toast?.error?.('重命名失败: ' + error.message);
                input.focus();
                input.select();
            }
        };
        
        input.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') { e.preventDefault(); confirm(); }
            else if (e.key === 'Escape') { 
                e.preventDefault();
                nameEl.style.display = '';
                input.remove();
            }
        });
        
        input.addEventListener('blur', () => setTimeout(() => {
            if (document.body.contains(input)) confirm();
        }, 100));
    }

    /**
     * 显示权限对话框
     */
    async showChmodDialog(filenames) {
        // TODO: 实现权限对话框
        this.options.toast?.info('权限功能开发中...');
    }

    /**
     * 压缩文件 - 使用自定义对话框
     */
    async compressFiles(filenames) {
        const tab = this.getActiveTab();
        if (!tab || filenames.length === 0) return;

        // 默认压缩包名称
        const defaultName = filenames.length === 1 ? filenames[0] : 'archive';
        
        // 使用自定义输入对话框
        const targetName = await this.showInputDialog('压缩文件', '请输入压缩包名称:', defaultName);
        if (!targetName) return;

        try {
            const response = await fetch(`${this.options.apiPath}/compress`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    path: tab.path,
                    files: filenames,
                    target: targetName,
                    format: 'zip'
                })
            });
            const result = await response.json();
            if (result?.code === 200) {
                this.options.toast?.success?.(`压缩成功: ${result.data?.file}`);
                this.loadFilesForTab();
            } else {
                this.options.toast?.error?.(result?.message || '压缩失败');
            }
        } catch (error) {
            this.options.toast?.error?.('压缩失败: ' + error.message);
        }
    }
    
    /**
     * 显示输入对话框
     */
    showInputDialog(title, message, defaultValue = '') {
        return new Promise((resolve) => {
            const overlay = document.createElement('div');
            overlay.className = 'dialog-overlay';
            overlay.innerHTML = `
                <div class="dialog-box">
                    <div class="dialog-header">
                        <span class="dialog-title">${title}</span>
                    </div>
                    <div class="dialog-body">
                        <p class="dialog-message">${message}</p>
                        <input type="text" class="dialog-input" id="dialog-input" value="${defaultValue}" autofocus>
                    </div>
                    <div class="dialog-footer">
                        <button class="dialog-btn dialog-btn-cancel">取消</button>
                        <button class="dialog-btn dialog-btn-confirm">确定</button>
                    </div>
                </div>
            `;

            document.body.appendChild(overlay);

            const input = overlay.querySelector('#dialog-input');
            const cancelBtn = overlay.querySelector('.dialog-btn-cancel');
            const confirmBtn = overlay.querySelector('.dialog-btn-confirm');

            input.focus();
            input.select();

            const close = (value) => {
                overlay.remove();
                resolve(value);
            };

            cancelBtn.addEventListener('click', () => close(null));
            confirmBtn.addEventListener('click', () => close(input.value.trim()));
            
            input.addEventListener('keydown', (e) => {
                if (e.key === 'Enter') close(input.value.trim());
                else if (e.key === 'Escape') close(null);
            });
            
            overlay.addEventListener('click', (e) => {
                if (e.target === overlay) close(null);
            });
        });
    }

    /**
     * 解压文件
     */
    async extractFile(filename) {
        const tab = this.getActiveTab();
        if (!tab) return;

        // 使用自定义确认对话框
        const confirmed = await this.showConfirmDialog('解压文件', `确定要解压 ${filename} 吗？`);
        if (!confirmed) return;

        try {
            const response = await fetch(`${this.options.apiPath}/extract`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    path: tab.path,
                    name: filename
                })
            });
            const result = await response.json();
            if (result?.code === 200) {
                this.options.toast?.success?.('解压成功');
                this.loadFilesForTab();
            } else {
                this.options.toast?.error?.(result?.message || '解压失败');
            }
        } catch (error) {
            this.options.toast?.error?.('解压失败: ' + error.message);
        }
    }
    
    /**
     * 显示确认对话框
     */
    showConfirmDialog(title, message) {
        return new Promise((resolve) => {
            const overlay = document.createElement('div');
            overlay.className = 'dialog-overlay';
            overlay.innerHTML = `
                <div class="dialog-box">
                    <div class="dialog-header">
                        <span class="dialog-title">${title}</span>
                    </div>
                    <div class="dialog-body">
                        <p class="dialog-message">${message}</p>
                    </div>
                    <div class="dialog-footer">
                        <button class="dialog-btn dialog-btn-cancel">取消</button>
                        <button class="dialog-btn dialog-btn-confirm">确定</button>
                    </div>
                </div>
            `;

            document.body.appendChild(overlay);

            const cancelBtn = overlay.querySelector('.dialog-btn-cancel');
            const confirmBtn = overlay.querySelector('.dialog-btn-confirm');

            const close = (value) => {
                overlay.remove();
                resolve(value);
            };

            cancelBtn.addEventListener('click', () => close(false));
            confirmBtn.addEventListener('click', () => close(true));
            
            overlay.addEventListener('keydown', (e) => {
                if (e.key === 'Enter') close(true);
                else if (e.key === 'Escape') close(false);
            });
            
            overlay.addEventListener('click', (e) => {
                if (e.target === overlay) close(false);
            });
            
            confirmBtn.focus();
        });
    }

    /**
     * 下载文件
     */
    downloadFile(filename) {
        const tab = this.getActiveTab();
        const url = `${this.options.apiPath}/download?path=${encodeURIComponent(tab.path)}&name=${encodeURIComponent(filename)}`;
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        a.click();
    }

    /**
     * 打开文件
     */
    openFile(filename) {
        // 默认行为：下载
        this.downloadFile(filename);
    }

    /**
     * 分享文件 - 使用自定义对话框
     */
    async shareFile(filename) {
        const tab = this.getActiveTab();
        if (!tab) return;

        // 生成随机提取码
        const generateCode = () => {
            const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
            let code = '';
            for (let i = 0; i < 4; i++) {
                code += chars.charAt(Math.floor(Math.random() * chars.length));
            }
            return code;
        };

        const randomCode = generateCode();

        // 显示分享设置对话框
        const overlay = document.createElement('div');
        overlay.className = 'dialog-overlay';
        overlay.innerHTML = `
            <div class="dialog-box share-dialog">
                <div class="dialog-header">
                    <span class="dialog-title">📤 外链分享</span>
                </div>
                <div class="dialog-body">
                    <div class="share-file-info">
                        <span class="share-file-icon">📄</span>
                        <span class="share-file-name">${filename}</span>
                    </div>
                    <div class="share-form-row">
                        <label>有效期</label>
                        <div class="share-duration-btns">
                            <button class="share-duration-btn active" data-hours="24">1天</button>
                            <button class="share-duration-btn" data-hours="168">7天</button>
                            <button class="share-duration-btn" data-hours="720">30天</button>
                            <button class="share-duration-btn" data-hours="8760">1年</button>
                            <button class="share-duration-btn" data-hours="0">永久</button>
                        </div>
                    </div>
                    <div class="share-form-row">
                        <label>提取码</label>
                        <div class="share-password-wrap">
                            <input type="text" id="share-password" value="${randomCode}" placeholder="提取码" maxlength="8">
                            <button class="share-gen-btn" id="share-gen-code" title="随机生成">🎲</button>
                        </div>
                        <label class="share-checkbox">
                            <input type="checkbox" id="share-auto-fill" checked>
                            <span>分享链接自动填充提取码</span>
                        </label>
                    </div>
                </div>
                <div class="dialog-footer">
                    <button class="dialog-btn dialog-btn-cancel" id="share-cancel-btn">取消</button>
                    <button class="dialog-btn dialog-btn-confirm" id="share-create-btn">创建分享</button>
                </div>
            </div>
        `;

        document.body.appendChild(overlay);

        const cancelBtn = overlay.querySelector('#share-cancel-btn');
        const createBtn = overlay.querySelector('#share-create-btn');
        const durationBtns = overlay.querySelectorAll('.share-duration-btn');
        const passwordInput = overlay.querySelector('#share-password');
        const genCodeBtn = overlay.querySelector('#share-gen-code');
        const autoFillCheck = overlay.querySelector('#share-auto-fill');
        
        let selectedDuration = 24;

        // 有效期按钮切换
        durationBtns.forEach(btn => {
            btn.addEventListener('click', () => {
                durationBtns.forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                selectedDuration = parseInt(btn.dataset.hours) || 0;
            });
        });

        // 随机生成提取码
        genCodeBtn.addEventListener('click', () => {
            passwordInput.value = generateCode();
        });

        const close = () => overlay.remove();

        const createShare = async () => {
            const password = passwordInput.value.trim();
            const autoFill = autoFillCheck.checked;

            createBtn.disabled = true;
            createBtn.textContent = '创建中...';

            try {
                const response = await fetch(`${this.options.apiPath}/share`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        path: tab.path,
                        name: filename,
                        duration: selectedDuration,
                        password: password || undefined
                    })
                });
                const result = await response.json();
                if (result?.code === 200) {
                    close();
                    // 显示分享结果
                    this.showShareDialog(result.data, password, autoFill);
                } else {
                    this.options.toast?.error?.(result?.message || '分享失败');
                    createBtn.disabled = false;
                    createBtn.textContent = '创建分享';
                }
            } catch (error) {
                this.options.toast?.error?.('分享失败: ' + error.message);
                createBtn.disabled = false;
                createBtn.textContent = '创建分享';
            }
        };

        cancelBtn.addEventListener('click', close);
        createBtn.addEventListener('click', createShare);

        overlay.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') close();
            if (e.key === 'Enter') createShare();
        });

        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) close();
        });

        // 聚焦密码输入框
        setTimeout(() => passwordInput.focus(), 100);
    }

    /**
     * 显示分享对话框
     */
    showShareDialog(data, password = '', autoFill = false) {
        // 检查数据
        if (!data || !data.url) {
            this.options.toast?.error?.('分享链接生成失败');
            return;
        }
        
        // 如果有密码且需要自动填充，拼接提取码
        let shareUrl = data.url;
        if (password && autoFill) {
            const separator = shareUrl.includes('?') ? '&' : '?';
            shareUrl = `${shareUrl}${separator}pwd=${encodeURIComponent(password)}`;
        }
        
        const overlay = document.createElement('div');
        overlay.className = 'dialog-overlay';
        overlay.innerHTML = `
            <div class="dialog-box share-result-dialog">
                <div class="dialog-header">
                    <span class="dialog-title">✅ 分享链接已创建</span>
                </div>
                <div class="dialog-body">
                    <div class="share-result-row">
                        <div class="share-result-info">
                            <label>文件名:</label>
                            <span>${data.fileName || '未知'}</span>
                        </div>
                        <div class="share-result-info">
                            <label>文件大小:</label>
                            <span>${this.formatSize(data.fileSize || 0)}</span>
                        </div>
                        <div class="share-result-info">
                            <label>过期时间:</label>
                            <span>${data.expiresAt ? new Date(data.expiresAt).toLocaleString() : '永久有效'}</span>
                        </div>
                        ${password ? `<div class="share-result-info"><label>提取码:</label><span class="share-pwd">${password}</span></div>` : ''}
                    </div>
                    <div class="share-link-box">
                        <input type="text" class="share-link-input" id="share-url" value="${shareUrl}" readonly>
                        <button class="dialog-btn dialog-btn-confirm" id="share-copy-btn">复制链接</button>
                    </div>
                </div>
                <div class="dialog-footer">
                    <button class="dialog-btn dialog-btn-cancel" id="share-close-btn">关闭</button>
                </div>
            </div>
        `;

        document.body.appendChild(overlay);

        const urlInput = overlay.querySelector('#share-url');
        const copyBtn = overlay.querySelector('#share-copy-btn');
        const closeBtn = overlay.querySelector('#share-close-btn');

        // 自动选中链接
        setTimeout(() => {
            urlInput.focus();
            urlInput.select();
        }, 100);

        const close = () => overlay.remove();

        // 复制功能
        const copyToClipboard = async () => {
            try {
                // 先选中文本
                urlInput.select();
                urlInput.setSelectionRange(0, shareUrl.length);
                
                // 尝试使用现代 API
                if (navigator.clipboard && navigator.clipboard.writeText) {
                    await navigator.clipboard.writeText(shareUrl);
                    this.options.toast?.success?.('已复制到剪贴板');
                } else {
                    // 备用方案：使用 execCommand
                    const success = document.execCommand('copy');
                    if (success) {
                        this.options.toast?.success?.('已复制到剪贴板');
                    } else {
                        this.options.toast?.error?.('复制失败，请手动复制');
                    }
                }
            } catch (err) {
                console.error('复制失败:', err);
                this.options.toast?.error?.('复制失败，请手动复制');
            }
        };

        copyBtn.addEventListener('click', copyToClipboard);
        closeBtn.addEventListener('click', close);

        overlay.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') close();
        });

        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) close();
        });
    }
    
    filterFiles(keyword) {
        const tab = this.getActiveTab();
        if (!tab) return;
        // 搜索实现...
    }
    
    /**
     * 复制文本到剪贴板（支持非 HTTPS 环境的备用方案）
     */
    async copyToClipboard(text, label = '内容') {
        try {
            // 优先使用现代 API
            if (navigator.clipboard && navigator.clipboard.writeText) {
                await navigator.clipboard.writeText(text);
                this.options.toast?.success?.(`已复制${label}`);
                return true;
            }
            
            // 备用方案：使用 textarea + execCommand
            const textarea = document.createElement('textarea');
            textarea.value = text;
            textarea.style.position = 'fixed';
            textarea.style.left = '-9999px';
            textarea.style.top = '0';
            document.body.appendChild(textarea);
            textarea.focus();
            textarea.select();
            
            const success = document.execCommand('copy');
            document.body.removeChild(textarea);
            
            if (success) {
                this.options.toast?.success?.(`已复制${label}`);
                return true;
            } else {
                this.options.toast?.error?.('复制失败');
                return false;
            }
        } catch (err) {
            console.error('复制失败:', err);
            this.options.toast?.error?.('复制失败: ' + err.message);
            return false;
        }
    }

    joinPath(...parts) { return parts.join('/').replace(/\/+/g, '/'); }
    
    showEmpty() { this.els.content.innerHTML = '<div class="fm-empty"><p>此文件夹为空</p></div>'; }
    showError(msg) { this.els.content.innerHTML = `<div class="fm-empty"><p style="color:var(--danger)">${msg}</p></div>`; }
}

export default FileManager;