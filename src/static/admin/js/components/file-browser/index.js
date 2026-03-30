/**
 * 文件浏览器组件
 * 
 * 轻量级文件选择器，支持文件夹/文件选择
 * 参考 Windows 资源管理器设计
 */

import { getFileIcon, getIconColorClass, formatFileSize, formatDate } from '../file-icons.js';

// 当前实例
let currentInstance = null;

// SVG 图标
const ICONS = {
    back: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>`,
    forward: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>`,
    up: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="18 15 12 9 6 15"/></svg>`,
    refresh: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>`,
    mkdir: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/><line x1="12" y1="11" x2="12" y2="17"/><line x1="9" y1="14" x2="15" y2="14"/></svg>`,
    search: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>`
};

/**
 * 文件浏览器类
 */
class FileBrowser {
    constructor(options) {
        this.options = {
            title: '选择文件',
            root: '/',
            selectMode: 'both',
            multiple: false,
            showHidden: false,
            filter: null,
            actions: ['refresh', 'mkdir'],
            onSelect: null,
            api: null,
            ...options
        };
        
        this.currentPath = this.options.root;
        this.selectedItems = [];
        this.items = [];
        this.filteredItems = [];
        this.history = [];
        this.historyIndex = -1;
        this.modal = null;
        this.resolvePromise = null;
        this.searchKeyword = '';
        this.programDir = null; // 程序运行目录
    }
    
    open() {
        return new Promise((resolve) => {
            this.resolvePromise = resolve;
            this.createModal();
            this.loadDirectory(this.currentPath);
        });
    }
    
    createModal() {
        const modal = document.createElement('div');
        modal.className = 'file-browser-modal';
        modal.innerHTML = `
            <div class="file-browser-overlay"></div>
            <div class="file-browser-container">
                <div class="file-browser-header">
                    <h3 class="file-browser-title">${this.options.title}</h3>
                    <button class="file-browser-close" title="关闭 (Esc)">×</button>
                </div>
                
                <!-- 工具栏 - Windows 风格 -->
                <div class="fb-toolbar">
                    <div class="fb-toolbar-nav">
                        <button class="fb-tool-btn" id="fb-back" title="后退 (Alt+←)" disabled>
                            ${ICONS.back}
                        </button>
                        <button class="fb-tool-btn" id="fb-forward" title="前进 (Alt+→)" disabled>
                            ${ICONS.forward}
                        </button>
                        <button class="fb-tool-btn" id="fb-up" title="上级目录 (Alt+↑)">
                            ${ICONS.up}
                        </button>
                    </div>
                    
                    <div class="fb-toolbar-divider"></div>
                    
                    <div class="fb-toolbar-actions">
                        <button class="fb-tool-btn" id="fb-refresh" title="刷新 (F5)">
                            ${ICONS.refresh}
                        </button>
                        ${this.options.actions.includes('mkdir') ? `
                        <button class="fb-tool-btn" id="fb-mkdir" title="新建文件夹 (Ctrl+Shift+N)">
                            ${ICONS.mkdir}
                        </button>
                        ` : ''}
                    </div>
                    
                    <!-- 地址栏 -->
                    <div class="fb-address-bar">
                        <div class="fb-path-breadcrumb" id="fb-breadcrumb"></div>
                        <input type="text" class="fb-address-input" id="fb-path-input" style="display:none;" placeholder="输入路径">
                    </div>
                    
                    <!-- 搜索框 -->
                    <div class="fb-search-box">
                        ${ICONS.search}
                        <input type="text" class="fb-search-input" id="fb-search" placeholder="搜索当前目录">
                    </div>
                </div>
                
                <div class="file-browser-content">
                    <div class="file-browser-list" id="fb-list">
                        <div class="fb-loading">加载中...</div>
                    </div>
                </div>
                
                <div class="file-browser-footer">
                    <div class="fb-status" id="fb-status">就绪</div>
                    <div class="fb-actions">
                        <button class="fb-btn fb-btn-secondary fb-btn-cancel">取消</button>
                        <button class="fb-btn fb-btn-primary fb-btn-confirm">确认选择</button>
                    </div>
                </div>
            </div>
        `;
        
        document.body.appendChild(modal);
        this.modal = modal;
        
        this.bindEvents();
        
        requestAnimationFrame(() => modal.classList.add('active'));
    }
    
    bindEvents() {
        const { modal } = this;
        
        // 关闭
        modal.querySelector('.file-browser-close').onclick = () => this.close(null);
        modal.querySelector('.file-browser-overlay').onclick = () => this.close(null);
        modal.querySelector('.fb-btn-cancel').onclick = () => this.close(null);
        
        // 工具栏
        modal.querySelector('#fb-back').onclick = () => this.goBack();
        modal.querySelector('#fb-forward').onclick = () => this.goForward();
        modal.querySelector('#fb-up').onclick = () => this.goUp();
        modal.querySelector('#fb-refresh').onclick = () => this.refresh();
        modal.querySelector('#fb-mkdir')?.addEventListener('click', () => this.createFolder());
        modal.querySelector('.fb-btn-confirm').onclick = () => this.confirmSelection();
        
        // 搜索
        let searchTimer;
        modal.querySelector('#fb-search').oninput = (e) => {
            clearTimeout(searchTimer);
            searchTimer = setTimeout(() => {
                this.searchKeyword = e.target.value.toLowerCase();
                this.filterAndRender();
            }, 200);
        };

        // 地址栏输入框
        const pathInput = modal.querySelector('#fb-path-input');
        pathInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                const path = pathInput.value.trim();
                if (path) this.loadDirectory(path);
                this.switchToBreadcrumbMode();
            } else if (e.key === 'Escape') {
                e.preventDefault();
                this.switchToBreadcrumbMode();
            }
        });
        pathInput.addEventListener('blur', () => {
            setTimeout(() => this.switchToBreadcrumbMode(), 150);
        });
        
        // 键盘快捷键
        this.keyHandler = (e) => {
            if (e.key === 'Escape') this.close(null);
            else if (e.key === 'Enter' && !e.target.matches('input')) this.confirmSelection();
            else if (e.key === 'F5') { e.preventDefault(); this.refresh(); }
            else if (e.key === 'Backspace' && !e.target.matches('input')) { e.preventDefault(); this.goUp(); }
            else if (e.altKey && e.key === 'ArrowLeft') { e.preventDefault(); this.goBack(); }
            else if (e.altKey && e.key === 'ArrowRight') { e.preventDefault(); this.goForward(); }
            else if (e.altKey && e.key === 'ArrowUp') { e.preventDefault(); this.goUp(); }
        };
        document.addEventListener('keydown', this.keyHandler);
        
        // 列表点击
        modal.querySelector('#fb-list').onclick = (e) => {
            const item = e.target.closest('.fb-item');
            if (item) this.handleItemClick(item, e);
        };
        
        // 列表双击
        modal.querySelector('#fb-list').ondblclick = (e) => {
            const item = e.target.closest('.fb-item');
            if (item) this.handleItemDblClick(item);
        };
    }
    
    async loadDirectory(path) {
        const listEl = this.modal.querySelector('#fb-list');
        const statusEl = this.modal.querySelector('#fb-status');
        
        listEl.innerHTML = '<div class="fb-loading">加载中...</div>';
        statusEl.textContent = '正在加载...';
        
        try {
            // 构建请求 URL
            const url = `/api/files?path=${encodeURIComponent(path)}`;
            console.log('[FileBrowser] Loading:', path);
            
            const response = await this.options.api.get(url);
            const data = await this.options.api.parseJSON(response);
            console.log('[FileBrowser] Response:', data);
            
            if (data && Array.isArray(data.files)) {
                this.items = data.files.map(f => ({
                    ...f,
                    is_dir: f.is_dir,
                    modified: f.modified
                }));
                
                // 使用 API 返回的完整路径
                if (data.path) {
                    this.currentPath = data.path;
                } else {
                    this.currentPath = path;
                }
                
                // 保存程序目录
                if (data.program_dir) {
                    this.programDir = data.program_dir;
                }
                
                // 更新历史
                if (this.history[this.historyIndex] !== this.currentPath) {
                    this.history = this.history.slice(0, this.historyIndex + 1);
                    this.history.push(this.currentPath);
                    this.historyIndex = this.history.length - 1;
                }
                
                this.updateNavButtons();
                this.updateBreadcrumb();
                this.filterAndRender();
                
                // 更新状态栏
                this.updateSelectionUI();
            } else {
                listEl.innerHTML = '<div class="fb-empty">无法加载目录</div>';
                statusEl.textContent = '加载失败';
            }
        } catch (error) {
            console.error('加载目录失败:', error);
            listEl.innerHTML = `<div class="fb-error">加载失败: ${error.message}</div>`;
            statusEl.textContent = '加载失败';
        }
    }
    
    filterAndRender() {
        const listEl = this.modal.querySelector('#fb-list');
        const { selectMode, showHidden, filter } = this.options;
        
        // 过滤
        let items = this.items;
        
        if (this.searchKeyword) {
            items = items.filter(item => 
                item.name.toLowerCase().includes(this.searchKeyword)
            );
        }
        
        if (!showHidden) {
            items = items.filter(item => !item.name.startsWith('.'));
        }
        
        if (filter) {
            items = items.filter(filter);
        }
        
        // 排序：文件夹优先
        items.sort((a, b) => {
            if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1;
            return a.name.localeCompare(b.name);
        });
        
        this.filteredItems = items;
        
        if (items.length === 0) {
            listEl.innerHTML = this.searchKeyword 
                ? `<div class="fb-empty">未找到匹配 "${this.searchKeyword}" 的文件</div>`
                : '<div class="fb-empty">空文件夹</div>';
            return;
        }
        
        // 判断是否在"此电脑"视图
        const isThisPC = this.currentPath === '此电脑';
        
        listEl.innerHTML = items.map(item => {
            const iconClass = getIconColorClass(item.name, item.is_dir);
            const iconHtml = getFileIcon(item.name, item.is_dir);
            const selectable = this.isSelectable(item);

            // 构建路径
            let itemPath;
            if (isThisPC) {
                itemPath = item.name + '/';
            } else if (this.currentPath === '/') {
                itemPath = '/' + item.name;
            } else {
                const basePath = this.currentPath.endsWith('/') ? this.currentPath : this.currentPath + '/';
                itemPath = basePath + item.name;
            }

            const selected = this.selectedItems.some(s => s.path === itemPath);

            return `
                <div class="fb-item ${selected ? 'selected' : ''} ${!selectable ? 'disabled' : ''}"
                     data-path="${itemPath}"
                     data-name="${item.name}"
                     data-is-dir="${item.is_dir}">
                    <span class="fb-item-icon ${iconClass}">${iconHtml}</span>
                    <span class="fb-item-name">${this.escapeHtml(item.name)}${isThisPC && item.is_dir ? '/' : ''}</span>
                    ${!item.is_dir ? `<span class="fb-item-size">${formatFileSize(item.size)}</span>` : ''}
                    ${!isThisPC && item.modified ? `<span class="fb-item-date">${formatDate(item.modified)}</span>` : ''}
                </div>
            `;
        }).join('');
    }
    
    updateNavButtons() {
        const backBtn = this.modal.querySelector('#fb-back');
        const forwardBtn = this.modal.querySelector('#fb-forward');
        const upBtn = this.modal.querySelector('#fb-up');
        
        backBtn.disabled = this.historyIndex <= 0;
        forwardBtn.disabled = this.historyIndex >= this.history.length - 1;
        
        // 只有在最顶层时才禁用向上按钮
        // Linux: currentPath === '/'
        // Windows: currentPath === '此电脑'
        const isAtTop = this.currentPath === '/' || this.currentPath === '此电脑';
        upBtn.disabled = isAtTop;
    }
    
    updateBreadcrumb() {
        const breadcrumb = this.modal.querySelector('#fb-breadcrumb');
        const parts = this.currentPath.split('/').filter(p => p);
        
        // 检测是否是 Windows 路径
        const isWindowsPath = parts.length > 0 && parts[0].endsWith(':');
        
        let html = '';
        let current = '';
        
        if (isWindowsPath) {
            // Windows 路径：第一级是"此电脑"，第二级是驱动器
            html = `<span class="fb-crumb ${this.currentPath === '此电脑' ? 'active' : ''}" data-path="此电脑">此电脑</span>`;
            html += `<span class="fb-crumb-sep">›</span>`;
            html += `<span class="fb-crumb ${this.currentPath === parts[0] + '/' ? 'active' : ''}" data-path="${parts[0]}/">${parts[0]}/</span>`;
            current = parts[0] + '/';
            
            for (let i = 1; i < parts.length; i++) {
                current += parts[i];
                const isActive = current === this.currentPath;
                html += `<span class="fb-crumb-sep">›</span>`;
                html += `<span class="fb-crumb ${isActive ? 'active' : ''}" data-path="${current}" title="${this.escapeHtml(parts[i])}">${this.escapeHtml(parts[i])}</span>`;
                if (i < parts.length - 1) current += '/';
            }
        } else {
            // Unix 路径：第一级始终是 `/`（系统根目录）
            html = `<span class="fb-crumb ${this.currentPath === '/' ? 'active' : ''}" data-path="/">/</span>`;
            
            parts.forEach((part, index) => {
                current += '/' + part;
                const isActive = current === this.currentPath;
                html += `<span class="fb-crumb-sep">›</span>`;
                html += `<span class="fb-crumb ${isActive ? 'active' : ''}" data-path="${current}" title="${this.escapeHtml(part)}">${this.escapeHtml(part)}</span>`;
            });
        }
        
        breadcrumb.innerHTML = html;
        
        // 绑定点击
        breadcrumb.querySelectorAll('.fb-crumb').forEach(el => {
            el.onclick = () => {
                if (el.dataset.path && el.dataset.path !== this.currentPath) {
                    this.loadDirectory(el.dataset.path);
                }
            };
        });

        // 双击面包屑切换到输入模式
        breadcrumb.ondblclick = (e) => {
            e.preventDefault();
            this.switchToInputMode();
        };

        // 点击空白区域也切换到输入模式
        breadcrumb.onclick = (e) => {
            if (!e.target.closest('.fb-crumb')) {
                this.switchToInputMode();
            }
        };
    }

    switchToInputMode() {
        const breadcrumb = this.modal.querySelector('#fb-breadcrumb');
        const input = this.modal.querySelector('#fb-path-input');
        if (!breadcrumb || !input) return;

        breadcrumb.style.display = 'none';
        input.style.display = 'block';

        // 显示完整路径
        let displayPath = this.currentPath;
        if (this.programDir && (displayPath === '.' || displayPath === './')) {
            displayPath = this.programDir;
        } else if (this.programDir && displayPath.startsWith('./')) {
            displayPath = this.programDir + displayPath.substring(1);
        }

        input.value = displayPath;
        input.focus();
        input.select();
    }

    switchToBreadcrumbMode() {
        const breadcrumb = this.modal.querySelector('#fb-breadcrumb');
        const input = this.modal.querySelector('#fb-path-input');
        if (!breadcrumb || !input) return;

        input.style.display = 'none';
        breadcrumb.style.display = 'flex';
    }
    
    goBack() {
        if (this.historyIndex > 0) {
            this.historyIndex--;
            this.loadDirectory(this.history[this.historyIndex]);
        }
    }
    
    goForward() {
        if (this.historyIndex < this.history.length - 1) {
            this.historyIndex++;
            this.loadDirectory(this.history[this.historyIndex]);
        }
    }
    
    goUp() {
        if (this.currentPath === '/' || this.currentPath === '此电脑') return;
        
        // Windows 驱动器根目录（如 C:/）向上跳转到"此电脑"
        if (/^[A-Za-z]:\/?$/.test(this.currentPath)) {
            this.loadDirectory('此电脑');
            return;
        }
        
        // 发送带 ".." 的路径，让后端处理跨平台兼容
        const parentPath = this.currentPath + '/..';
        this.loadDirectory(parentPath);
    }
    
    refresh() {
        this.loadDirectory(this.currentPath);
    }
    
    isSelectable(item) {
        const { selectMode } = this.options;
        if (selectMode === 'folder') return true; // 文件夹模式下所有项目都可交互
        if (selectMode === 'file') return !item.is_dir;
        return true;
    }
    
    handleItemClick(itemEl, e) {
        const path = itemEl.dataset.path;
        const isDir = itemEl.dataset.isDir === 'true';
        
        // 文件夹：单击进入
        if (isDir) {
            this.loadDirectory(path);
            return;
        }
        
        // 文件夹模式下，点击文件不做选择
        if (this.options.selectMode === 'folder') {
            return;
        }
        
        // 文件：选中
        const item = this.filteredItems.find(i => {
            const basePath = this.currentPath.endsWith('/') ? this.currentPath : this.currentPath + '/';
            const itemPath = basePath + i.name;
            return itemPath === path;
        });
        
        if (!item || !this.isSelectable(item)) return;
        
        // Ctrl 多选
        if (this.options.multiple && (e.ctrlKey || e.metaKey)) {
            const index = this.selectedItems.findIndex(s => s.path === path);
            if (index >= 0) {
                this.selectedItems.splice(index, 1);
            } else {
                this.selectedItems.push({ ...item, path });
            }
        } else {
            this.selectedItems = [{ ...item, path }];
        }
        
        this.updateSelectionUI();
    }
    
    handleItemDblClick(itemEl) {
        const path = itemEl.dataset.path;
        const isDir = itemEl.dataset.isDir === 'true';
        
        if (isDir) {
            this.loadDirectory(path);
        } else {
            const item = this.filteredItems.find(i => {
                const basePath = this.currentPath.endsWith('/') ? this.currentPath : this.currentPath + '/';
                const itemPath = basePath + i.name;
                return itemPath === path;
            });
            
            if (item && this.isSelectable(item)) {
                this.selectedItems = [{ ...item, path }];
                this.confirmSelection();
            }
        }
    }
    
    updateSelectionUI() {
        this.modal.querySelectorAll('.fb-item').forEach(el => {
            const path = el.dataset.path;
            const selected = this.selectedItems.some(s => s.path === path);
            el.classList.toggle('selected', selected);
        });
        
        const statusEl = this.modal.querySelector('#fb-status');
        const { selectMode } = this.options;
        
        if (selectMode === 'folder') {
            // 文件夹模式：显示当前文件夹路径
            const displayPath = this.currentPath === '/' ? '根目录' : this.currentPath;
            statusEl.textContent = `当前文件夹：${displayPath}`;
        } else if (this.selectedItems.length > 0) {
            statusEl.textContent = `已选择 ${this.selectedItems.length} 项`;
        } else {
            statusEl.textContent = `${this.filteredItems.length} 项`;
        }
    }
    
    createFolder() {
        const listEl = this.modal.querySelector('#fb-list');
        
        // 创建新建文件夹行
        const newItem = document.createElement('div');
        newItem.className = 'fb-item new-folder';
        newItem.innerHTML = `
            <span class="fb-item-icon file-icon-folder">${getFileIcon('', true)}</span>
            <input type="text" class="fb-folder-name-input" placeholder="输入文件夹名称" autofocus>
            <div class="fb-folder-actions">
                <button class="fb-folder-btn confirm">确定</button>
                <button class="fb-folder-btn cancel">取消</button>
            </div>
        `;
        
        // 插入到列表最前面
        listEl.insertBefore(newItem, listEl.firstChild);
        
        const input = newItem.querySelector('.fb-folder-name-input');
        const confirmBtn = newItem.querySelector('.fb-folder-btn.confirm');
        const cancelBtn = newItem.querySelector('.fb-folder-btn.cancel');
        
        // 聚焦输入框
        setTimeout(() => input.focus(), 100);
        
        // 取消
        const cancel = () => newItem.remove();
        cancelBtn.onclick = cancel;
        
        // 创建
        const doCreate = async () => {
            const name = input.value.trim();
            if (!name) {
                input.focus();
                return;
            }
            
            try {
                const basePath = this.currentPath.endsWith('/') ? this.currentPath : this.currentPath + '/';
                const newPath = basePath + name;
                
                await this.options.api.post('/api/files/mkdir', { path: newPath });
                newItem.remove();
                this.refresh();
            } catch (error) {
                input.style.borderColor = 'var(--danger)';
                input.style.animation = 'shake 0.3s';
            }
        };
        
        confirmBtn.onclick = doCreate;
        
        // 回车确认
        input.onkeydown = (e) => {
            if (e.key === 'Enter') doCreate();
            else if (e.key === 'Escape') cancel();
        };
    }
    
    confirmSelection() {
        const { selectMode } = this.options;
        
        // 文件夹模式：直接选择当前浏览的文件夹
        if (selectMode === 'folder') {
            // 返回当前路径，后端会智能转换为相对/绝对路径
            this.close(this.currentPath);
            return;
        }
        
        // 文件/混合模式：需要有选中项
        if (this.selectedItems.length === 0) {
            const status = this.modal.querySelector('#fb-status');
            status.textContent = '请选择文件或文件夹';
            status.style.color = 'var(--warning)';
            setTimeout(() => {
                status.style.color = '';
                status.textContent = `${this.filteredItems.length} 项`;
            }, 1500);
            return;
        }
        
        const result = this.options.multiple 
            ? this.selectedItems.map(i => i.path)
            : this.selectedItems[0].path;
        
        this.close(result);
    }
    
    close(result) {
        if (this.modal) {
            this.modal.classList.remove('active');
            setTimeout(() => this.modal.remove(), 200);
        }
        
        if (this.keyHandler) {
            document.removeEventListener('keydown', this.keyHandler);
        }
        
        if (this.resolvePromise) {
            this.resolvePromise(result);
        }
        
        currentInstance = null;
    }
    
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

export function openFileBrowser(options) {
    if (currentInstance) currentInstance.close(null);
    
    const instance = new FileBrowser(options);
    currentInstance = instance;
    return instance.open();
}

export function createFileBrowser(api) {
    return {
        open: (options) => openFileBrowser({ ...options, api })
    };
}

export default { open: openFileBrowser, create: createFileBrowser };