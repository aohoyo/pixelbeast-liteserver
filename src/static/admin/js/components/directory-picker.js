/**
 * DirectoryPicker - 目录选择器组件
 *
 * 用于选择服务器上的目录路径
 */

export class DirectoryPicker {
    /**
     * @param {Object} options
     * @param {HTMLElement} options.container - 容器元素
     * @param {Object} options.api - API 实例（带 CSRF token）
     * @param {string} options.apiPath - API 路径前缀
     * @param {string} options.rootPath - 根目录路径（ftp 或 web）
     * @param {string} options.value - 初始值
     * @param {Function} options.onChange - 值变化回调
     * @param {string} options.placeholder - 占位符
     */
    constructor(options) {
        this.container = options.container;
        this.api = options.api;
        this.apiPath = options.apiPath || '/api/files';
        this.rootPath = options.rootPath || '';
        this.value = options.value || '';
        this.onChange = options.onChange || (() => {});
        this.placeholder = options.placeholder || '请选择目录';
        this.currentPath = '/';
        this.directories = [];
        
        // 生成唯一 ID（避免多个实例 id 冲突）
        this.uid = 'dp-' + Math.random().toString(36).substr(2, 9);
        
        this.render();
        this.bindEvents();
    }

    render() {
        const displayValue = this.value || '';
        const displayPlaceholder = displayValue ? '' : this.placeholder;
        
        this.container.innerHTML = `
            <div class="directory-picker">
                <div class="directory-picker-input">
                    <input type="text" class="form-input-inline" id="${this.uid}-value" value="${this.escapeHtml(displayValue)}" placeholder="${displayPlaceholder}" />
                    <button type="button" class="btn btn-secondary directory-picker-btn" id="${this.uid}-browse-btn" title="浏览目录">
                        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
                        </svg>
                    </button>
                </div>
            </div>
            
            <div class="modal directory-picker-modal" id="${this.uid}-modal">
                <div class="modal-overlay"></div>
                <div class="modal-container" style="width: 500px; max-width: 90vw;">
                    <div class="modal-header">
                        <h3>选择目录</h3>
                        <button class="modal-close" id="${this.uid}-close">&times;</button>
                    </div>
                    <div class="modal-body">
                        <div class="directory-picker-path">
                            <button type="button" class="btn btn-sm" id="${this.uid}-root" title="根目录">
                                <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path>
                                    <polyline points="9 22 9 12 15 12 15 22"></polyline>
                                </svg>
                            </button>
                            <div class="directory-picker-breadcrumb" id="${this.uid}-breadcrumb"></div>
                        </div>
                        <div class="directory-picker-list" id="${this.uid}-list">
                            <div class="directory-picker-loading">加载中...</div>
                        </div>
                        <div class="directory-picker-selected">
                            <span class="directory-picker-selected-label">已选择：</span>
                            <span class="directory-picker-selected-path" id="${this.uid}-selected-path">/</span>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button class="btn" id="${this.uid}-cancel">取消</button>
                        <button class="btn btn-primary" id="${this.uid}-confirm">确定</button>
                    </div>
                </div>
            </div>
        `;

        // 缓存元素
        this.modal = this.container.querySelector(`#${this.uid}-modal`);
        this.listEl = this.container.querySelector(`#${this.uid}-list`);
        this.breadcrumbEl = this.container.querySelector(`#${this.uid}-breadcrumb`);
        this.selectedPathEl = this.container.querySelector(`#${this.uid}-selected-path`);
        this.valueInput = this.container.querySelector(`#${this.uid}-value`);
    }

    bindEvents() {
        // 打开选择器（只通过按钮打开）
        this.container.querySelector(`#${this.uid}-browse-btn`).addEventListener('click', () => {
            this.open();
        });

        // 关闭
        this.container.querySelector(`#${this.uid}-close`).addEventListener('click', () => this.close());
        this.container.querySelector(`#${this.uid}-cancel`).addEventListener('click', () => this.close());
        this.modal.querySelector('.modal-overlay').addEventListener('click', () => this.close());

        // 确认
        this.container.querySelector(`#${this.uid}-confirm`).addEventListener('click', () => this.confirm());

        // 根目录
        this.container.querySelector(`#${this.uid}-root`).addEventListener('click', () => {
            this.currentPath = '.';
            this.loadDirectories();
        });
    }

    open() {
        // 从当前值解析路径
        if (this.value) {
            this.currentPath = this.value;
        } else {
            // 默认使用当前目录（程序运行目录）
            this.currentPath = '.';
        }
        
        this.modal.classList.add('active');
        this.loadDirectories();
    }

    close() {
        this.modal.classList.remove('active');
    }

    confirm() {
        this.value = this.currentPath;
        this.valueInput.value = this.value;
        this.onChange(this.value);
        this.close();
    }

    async loadDirectories() {
        this.listEl.innerHTML = '<div class="directory-picker-loading">加载中...</div>';
        this.selectedPathEl.textContent = this.currentPath;

        try {
            const url = `${this.apiPath}?path=${encodeURIComponent(this.currentPath)}&dirsOnly=true`;
            let result;
            
            if (this.api && typeof this.api.getJSON === 'function') {
                result = await this.api.getJSON(url);
            } else {
                const response = await fetch(url);
                result = await response.json();
            }

            if (result && result.files) {
                this.directories = result.files.filter(f => f.is_dir);
                this.renderList();
                // 更新当前路径为完整路径（标准化处理）
                if (result.path) {
                    // 移除多余的斜杠
                    this.currentPath = result.path.replace(/\/+/g, '/');
                    this.selectedPathEl.textContent = this.currentPath;
                }
                this.renderBreadcrumb();
            } else if (result && result.data?.files) {
                this.directories = result.data.files.filter(f => f.is_dir);
                this.renderList();
                // 更新当前路径为完整路径（标准化处理）
                if (result.data.path) {
                    // 移除多余的斜杠
                    this.currentPath = result.data.path.replace(/\/+/g, '/');
                    this.selectedPathEl.textContent = this.currentPath;
                }
                this.renderBreadcrumb();
            } else {
                this.listEl.innerHTML = '<div class="directory-picker-empty">无法加载目录</div>';
            }
        } catch (error) {
            console.error('DirectoryPicker loadDirectories error:', error);
            this.listEl.innerHTML = `<div class="directory-picker-error">加载失败: ${error.message}</div>`;
        }
    }

    renderList() {
        if (this.directories.length === 0) {
            this.listEl.innerHTML = '<div class="directory-picker-empty">当前目录为空</div>';
            return;
        }

        this.listEl.innerHTML = this.directories.map(dir => `
            <div class="directory-picker-item" data-name="${this.escapeHtml(dir.name)}">
                <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
                </svg>
                <span class="directory-picker-item-name">${this.escapeHtml(dir.name)}</span>
                <svg class="icon directory-picker-item-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="9 18 15 12 9 6"></polyline>
                </svg>
            </div>
        `).join('');

        // 绑定点击事件
        this.listEl.querySelectorAll('.directory-picker-item').forEach(item => {
            item.addEventListener('click', () => {
                const name = item.dataset.name;
                this.currentPath = this.currentPath === '/' ? `/${name}` : `${this.currentPath}/${name}`;
                this.loadDirectories();
            });

            // 双击选择
            item.addEventListener('dblclick', () => {
                this.confirm();
            });
        });
    }

    renderBreadcrumb() {
        // 标准化路径：移除多余的 / 和 .
        let normalizedPath = this.currentPath;
        
        // 如果是根目录
        if (normalizedPath === '/' || normalizedPath === '') {
            this.breadcrumbEl.innerHTML = '<span class="directory-picker-breadcrumb-item active">根目录</span>';
            return;
        }
        
        // 过滤掉空字符串和 .
        const parts = normalizedPath.split('/').filter(p => p && p !== '.');
        
        if (parts.length === 0) {
            this.breadcrumbEl.innerHTML = '<span class="directory-picker-breadcrumb-item active">根目录</span>';
            return;
        }

        let pathAccum = '';

        const items = parts.map((part, index) => {
            pathAccum += '/' + part;
            const path = pathAccum;
            const isLast = index === parts.length - 1;
            return `
                <span class="directory-picker-breadcrumb-separator">/</span>
                <span class="directory-picker-breadcrumb-item ${isLast ? 'active' : ''}" data-path="${path}">${this.escapeHtml(part)}</span>
            `;
        });

        // 不再显示开头的 / 按钮（已有根目录按钮）
        this.breadcrumbEl.innerHTML = items.join('');

        // 绑定点击
        this.breadcrumbEl.querySelectorAll('.directory-picker-breadcrumb-item').forEach(item => {
            item.addEventListener('click', () => {
                if (item.dataset.path) {
                    this.currentPath = item.dataset.path;
                    this.loadDirectories();
                }
            });
        });
    }

    setValue(value) {
        this.value = value || '';
        this.valueInput.value = this.value;
    }

    getValue() {
        return this.value;
    }

    escapeHtml(text) {
        if (!text) return '';
        const map = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#039;' };
        return String(text).replace(/[&<>"']/g, m => map[m]);
    }
}

export default DirectoryPicker;