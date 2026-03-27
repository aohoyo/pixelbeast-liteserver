/**
 * DataTable 组件
 * 通用的数据表格组件，支持分页、选择、批量操作
 */

import { escapeHtml } from '../core/utils.js';

// 批量操作条样式（单例）
let batchBarStylesInjected = false;
let batchBarContainer = null;

function injectBatchBarStyles() {
    if (batchBarStylesInjected) return;
    
    const style = document.createElement('style');
    style.id = 'dt-batch-bar-styles';
    style.textContent = `
        #dt-batch-bar-container {
            position: fixed !important;
            bottom: 0 !important;
            left: 0 !important;
            right: 0 !important;
            z-index: 9998 !important;
            display: flex !important;
            align-items: flex-end !important;
            justify-content: center !important;
            padding: var(--space-md, 16px) 20px !important;
            pointer-events: none !important;
        }
        
        #dt-batch-bar-container.active {
            top: 0 !important;
            pointer-events: auto !important;
        }
        
        .dt-batch-bar {
            display: flex !important;
            align-items: center !important;
            gap: var(--space-md, 16px) !important;
            padding: 12px 20px !important;
            background: var(--card-bg) !important;
            border: 1px solid var(--border) !important;
            border-radius: var(--radius-xl, 16px) !important;
            box-shadow: var(--shadow-lg) !important;
            pointer-events: auto !important;
            transform: translateY(120%) !important;
            opacity: 0 !important;
            transition: transform 0.35s cubic-bezier(0.34, 1.56, 0.64, 1), opacity 0.25s ease !important;
            max-width: 600px !important;
        }
        
        .dt-batch-bar.show {
            transform: translateY(0) !important;
            opacity: 1 !important;
        }
        
        .dt-batch-bar-count {
            display: flex !important;
            align-items: center !important;
            gap: var(--space-sm, 8px) !important;
            color: var(--text-secondary) !important;
            font-size: 14px !important;
            font-weight: 500 !important;
        }
        
        .dt-batch-bar-count svg {
            color: var(--primary) !important;
        }
        
        .dt-batch-bar-count strong {
            font-size: 20px !important;
            font-weight: 700 !important;
            color: var(--primary) !important;
        }
        
        .dt-batch-bar-divider {
            width: 1px !important;
            height: 28px !important;
            background: var(--border) !important;
        }
        
        .dt-batch-bar-actions {
            display: flex !important;
            align-items: center !important;
            gap: var(--space-sm, 8px) !important;
        }
        
        .dt-batch-bar-btn {
            display: flex !important;
            align-items: center !important;
            gap: 6px !important;
            padding: var(--space-sm, 8px) var(--space-md, 16px) !important;
            background: var(--bg-hover) !important;
            border: 1px solid var(--border) !important;
            border-radius: var(--radius-md, 10px) !important;
            color: var(--text-secondary) !important;
            font-size: 13px !important;
            font-weight: 500 !important;
            cursor: pointer !important;
            transition: all var(--transition-fast, 0.15s ease) !important;
        }
        
        .dt-batch-bar-btn:hover {
            background: var(--card-bg-hover) !important;
            border-color: var(--border-light) !important;
            transform: translateY(-1px) !important;
        }
        
        .dt-batch-bar-btn:active {
            transform: translateY(0) scale(0.98) !important;
        }
        
        .dt-batch-bar-btn svg {
            width: 16px !important;
            height: 16px !important;
            stroke: currentColor !important;
            stroke-width: 2 !important;
            fill: none !important;
        }
        
        .dt-batch-bar-btn.success {
            background: linear-gradient(135deg, var(--success), #16a34a) !important;
            border-color: transparent !important;
            color: #fff !important;
        }
        
        .dt-batch-bar-btn.danger {
            background: linear-gradient(135deg, var(--danger), #dc2626) !important;
            border-color: transparent !important;
            color: #fff !important;
        }
        
        .dt-batch-bar-close {
            display: flex !important;
            align-items: center !important;
            justify-content: center !important;
            width: 36px !important;
            height: 36px !important;
            background: var(--bg-hover) !important;
            border: 1px solid var(--border) !important;
            border-radius: var(--radius-md, 10px) !important;
            color: var(--text-muted) !important;
            cursor: pointer !important;
            transition: all var(--transition-fast, 0.15s ease) !important;
            font-size: 20px !important;
        }
        
        .dt-batch-bar-close:hover {
            background: var(--danger-alpha) !important;
            border-color: var(--danger) !important;
            color: var(--danger) !important;
        }
        
        @media (max-width: 480px) {
            .dt-batch-bar {
                width: calc(100% - 20px) !important;
                max-width: none !important;
                padding: 10px var(--space-md, 16px) !important;
                gap: 12px !important;
            }
            
            .dt-batch-bar-btn {
                padding: 6px 10px !important;
                font-size: 12px !important;
            }
            
            .dt-batch-bar-btn span {
                display: none !important;
            }
        }
    `;
    
    document.head.appendChild(style);
    batchBarStylesInjected = true;
}

function ensureBatchBarContainer() {
    if (!batchBarContainer) {
        batchBarContainer = document.createElement('div');
        batchBarContainer.id = 'dt-batch-bar-container';
        batchBarContainer.innerHTML = '<div class="dt-batch-bar" id="dt-batch-bar"></div>';
        document.body.appendChild(batchBarContainer);
    }
}

function showBatchBar(count, actions, onCancel) {
    injectBatchBarStyles();
    ensureBatchBarContainer();
    
    const bar = document.getElementById('dt-batch-bar');
    if (!bar) return;
    
    let html = `
        <div class="dt-batch-bar-count">
            <svg viewBox="0 0 24 24" width="18" height="18" stroke="currentColor" stroke-width="2" fill="none">
                <polyline points="9 11 12 14 22 4"></polyline>
                <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"></path>
            </svg>
            已选择 <strong>${count}</strong> 项
        </div>
        <div class="dt-batch-bar-divider"></div>
        <div class="dt-batch-bar-actions">
    `;
    
    actions.forEach(action => {
        const typeClass = action.type || '';
        html += `
            <button class="dt-batch-bar-btn ${typeClass}" data-action="${action.key}">
                ${action.icon || ''}
                <span>${action.label}</span>
            </button>
        `;
    });
    
    html += `
        </div>
        <button class="dt-batch-bar-close" id="dt-batch-bar-cancel" title="取消选择">×</button>
    `;
    
    bar.innerHTML = html;
    
    requestAnimationFrame(() => {
        bar.classList.add('show');
        batchBarContainer?.classList.add('active');
    });
    
    // 绑定按钮事件
    bar.querySelectorAll('.dt-batch-bar-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const key = btn.dataset.action;
            const handler = actions.find(a => a.key === key)?.handler;
            if (handler) handler();
        });
    });
    
    // 取消按钮
    bar.querySelector('#dt-batch-bar-cancel')?.addEventListener('click', () => {
        hideBatchBar();
        if (onCancel) onCancel();
    });
    
    // 点击外部区域关闭
    const container = document.getElementById('dt-batch-bar-container');
    if (container) {
        container.onclick = (e) => {
            if (e.target === container) {
                hideBatchBar();
                if (onCancel) onCancel();
            }
        };
    }
    
    // 保存取消回调供外部调用
    batchBarContainer._onCancel = onCancel;
}

function hideBatchBar() {
    const bar = document.getElementById('dt-batch-bar');
    const container = document.getElementById('dt-batch-bar-container');
    if (bar) {
        bar.classList.remove('show');
    }
    if (container) {
        container.classList.remove('active');
    }
}

export class DataTable {
    constructor(options) {
        this.options = {
            container: null,
            columns: [],
            data: [],
            pageSize: 20,
            pageSizeOptions: [10, 20, 50, 100],
            selectable: false,
            batchActions: null,
            emptyText: '暂无数据',
            emptyHint: '',
            loadingText: '加载中...',
            onPageChange: null,
            onSelectionChange: null,
            ...options
        };

        this.state = {
            currentPage: 1,
            pageSize: this.options.pageSize,
            selectedKeys: new Set(),
            loading: false,
            filteredData: []
        };

        this.container = typeof this.options.container === 'string'
            ? document.querySelector(this.options.container)
            : this.options.container;

        if (!this.container) {
            console.error('DataTable: container not found');
            return;
        }

        this.init();
    }

    init() {
        this.render();
        this.bindEvents();
        this.updateData(this.options.data);
    }

    render() {
        const { columns, selectable, pageSizeOptions } = this.options;

        this.container.innerHTML = `
            <div class="dt-container">
                <div class="dt-table-wrapper">
                    <table class="dt-table">
                        <thead>
                            <tr>
                                ${selectable ? '<th class="dt-checkbox-col"><input type="checkbox" class="dt-checkbox dt-select-all"></th>' : ''}
                                ${columns.map(col => `<th class="${col.className || ''}" style="${col.width ? 'width:' + col.width : ''}">${col.title}</th>`).join('')}
                            </tr>
                        </thead>
                        <tbody class="dt-body">
                            <tr><td colspan="${selectable ? columns.length + 1 : columns.length}" class="dt-loading-cell"><div class="dt-loading-content"><div class="dt-loading-spinner"></div><span>${this.options.loadingText}</span></div></td></tr>
                        </tbody>
                    </table>
                </div>
                <div class="dt-pagination">
                    <div class="dt-pagination-left">
                        <div class="dt-info">共 <strong class="dt-total">0</strong> 条记录</div>
                        <div class="dt-size-selector"><span>每页</span><select class="dt-page-size">${pageSizeOptions.map(size => `<option value="${size}" ${size === this.state.pageSize ? 'selected' : ''}>${size}</option>`).join('')}</select><span>条</span></div>
                    </div>
                    <div class="dt-pagination-right">
                        <nav class="dt-nav">
                            <button class="dt-btn dt-prev" disabled><svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"></polyline></svg><span>上一页</span></button>
                            <div class="dt-pages"></div>
                            <button class="dt-btn dt-next" disabled><span>下一页</span><svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"></polyline></svg></button>
                        </nav>
                        <div class="dt-jump"><span>跳至</span><input type="number" class="dt-jump-input" min="1"><span>页</span></div>
                    </div>
                </div>
            </div>
        `;

        this.els = {
            tbody: this.container.querySelector('.dt-body'),
            selectAll: this.container.querySelector('.dt-select-all'),
            total: this.container.querySelector('.dt-total'),
            pageSize: this.container.querySelector('.dt-page-size'),
            prevBtn: this.container.querySelector('.dt-prev'),
            nextBtn: this.container.querySelector('.dt-next'),
            pages: this.container.querySelector('.dt-pages'),
            jumpInput: this.container.querySelector('.dt-jump-input')
        };
    }

    bindEvents() {
        if (this.els.selectAll) {
            this.els.selectAll.addEventListener('change', (e) => {
                const checked = e.target.checked;
                if (checked) {
                    this.getCurrentPageData().forEach(row => this.state.selectedKeys.add(this.getRowKey(row)));
                } else {
                    this.getCurrentPageData().forEach(row => this.state.selectedKeys.delete(this.getRowKey(row)));
                }
                this.renderBody();
                this.updateSelectAllState();
                this.emitSelectionChange();
            });
        }

        this.els.pageSize?.addEventListener('change', (e) => {
            this.state.pageSize = parseInt(e.target.value);
            this.state.currentPage = 1;
            this.renderBody();
            this.renderPagination();
        });

        this.els.prevBtn?.addEventListener('click', () => {
            if (this.state.currentPage > 1) {
                this.state.currentPage--;
                this.renderBody();
                this.renderPagination();
                this.options.onPageChange?.(this.state.currentPage);
            }
        });

        this.els.nextBtn?.addEventListener('click', () => {
            const totalPages = this.getTotalPages();
            if (this.state.currentPage < totalPages) {
                this.state.currentPage++;
                this.renderBody();
                this.renderPagination();
                this.options.onPageChange?.(this.state.currentPage);
            }
        });

        this.els.jumpInput?.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                const page = parseInt(e.target.value);
                const totalPages = this.getTotalPages();
                if (page >= 1 && page <= totalPages) {
                    this.state.currentPage = page;
                    this.renderBody();
                    this.renderPagination();
                    this.options.onPageChange?.(this.state.currentPage);
                }
            }
        });
    }

    getRowKey(row) {
        return row.id || row.key || row.username || row.name || JSON.stringify(row);
    }

    updateData(data) {
        this.state.filteredData = data || [];
        this.state.currentPage = 1;
        this.state.selectedKeys.clear();
        this.updateSelectAllState();
        this.renderBody();
        this.renderPagination();
        this.updateBatchBar();
    }

    setLoading(loading) {
        this.state.loading = loading;
        if (loading) {
            this.els.tbody.innerHTML = `<tr><td colspan="${this.getColSpan()}" class="dt-loading-cell"><div class="dt-loading-content"><div class="dt-loading-spinner"></div><span>${this.options.loadingText}</span></div></td></tr>`;
        } else {
            this.renderBody();
        }
    }

    getCurrentPageData() {
        const start = (this.state.currentPage - 1) * this.state.pageSize;
        return this.state.filteredData.slice(start, start + this.state.pageSize);
    }

    getTotalPages() {
        return Math.ceil(this.state.filteredData.length / this.state.pageSize) || 1;
    }

    getColSpan() {
        return this.options.selectable ? this.options.columns.length + 1 : this.options.columns.length;
    }

    renderBody() {
        const { columns, selectable, emptyText, emptyHint } = this.options;
        const data = this.getCurrentPageData();

        if (this.state.filteredData.length === 0) {
            this.els.tbody.innerHTML = `<tr><td colspan="${this.getColSpan()}" class="dt-empty-cell"><div class="dt-empty"><div class="dt-empty-text">${emptyText}</div>${emptyHint ? '<div class="dt-empty-hint">' + emptyHint + '</div>' : ''}</div></td></tr>`;
            return;
        }

        this.els.tbody.innerHTML = data.map(row => {
            const rowKey = this.getRowKey(row);
            const isSelected = this.state.selectedKeys.has(rowKey);
            return `<tr data-key="${escapeHtml(rowKey)}">
                ${selectable ? `<td class="dt-checkbox-col"><input type="checkbox" class="dt-checkbox dt-row-checkbox" data-key="${escapeHtml(rowKey)}" ${isSelected ? 'checked' : ''}></td>` : ''}
                ${columns.map(col => `<td class="${col.className || ''}">${col.render ? col.render(row[col.dataIndex], row, this) : escapeHtml(row[col.dataIndex] ?? '-')}</td>`).join('')}
            </tr>`;
        }).join('');

        this.bindRowEvents();
    }

    bindRowEvents() {
        if (this.options.selectable) {
            this.container.querySelectorAll('.dt-row-checkbox').forEach(cb => {
                cb.addEventListener('change', (e) => {
                    const key = e.target.dataset.key;
                    if (e.target.checked) {
                        this.state.selectedKeys.add(key);
                    } else {
                        this.state.selectedKeys.delete(key);
                    }
                    this.updateSelectAllState();
                    this.emitSelectionChange();
                });
            });
        }
    }

    updateSelectAllState() {
        if (!this.els.selectAll) return;
        const currentPageData = this.getCurrentPageData();
        if (currentPageData.length === 0) {
            this.els.selectAll.checked = false;
            this.els.selectAll.indeterminate = false;
            return;
        }
        const selectedCount = currentPageData.filter(row => this.state.selectedKeys.has(this.getRowKey(row))).length;
        this.els.selectAll.checked = selectedCount === currentPageData.length;
        this.els.selectAll.indeterminate = selectedCount > 0 && selectedCount < currentPageData.length;
    }

    emitSelectionChange() {
        this.options.onSelectionChange?.({
            selectedKeys: Array.from(this.state.selectedKeys),
            selectedCount: this.state.selectedKeys.size
        });
        this.updateBatchBar();
    }

    updateBatchBar() {
        const { batchActions } = this.options;
        const count = this.state.selectedKeys.size;
        if (batchActions && batchActions.length > 0) {
            if (count > 0) {
                showBatchBar(count, batchActions, () => this.clearSelection());
            } else {
                hideBatchBar();
            }
        }
    }

    renderPagination() {
        const totalPages = this.getTotalPages();
        const { currentPage } = this.state;

        if (this.els.total) this.els.total.textContent = this.state.filteredData.length;
        if (this.els.prevBtn) this.els.prevBtn.disabled = currentPage <= 1;
        if (this.els.nextBtn) this.els.nextBtn.disabled = currentPage >= totalPages;

        if (!this.els.pages) return;

        let html = '';
        const maxVisible = 5;

        if (totalPages <= maxVisible) {
            for (let i = 1; i <= totalPages; i++) html += this.renderPageButton(i, i === currentPage);
        } else {
            if (currentPage <= 3) {
                for (let i = 1; i <= 4; i++) html += this.renderPageButton(i, i === currentPage);
                html += '<span class="dt-ellipsis">...</span>';
                html += this.renderPageButton(totalPages, false);
            } else if (currentPage >= totalPages - 2) {
                html += this.renderPageButton(1, false);
                html += '<span class="dt-ellipsis">...</span>';
                for (let i = totalPages - 3; i <= totalPages; i++) html += this.renderPageButton(i, i === currentPage);
            } else {
                html += this.renderPageButton(1, false);
                html += '<span class="dt-ellipsis">...</span>';
                for (let i = currentPage - 1; i <= currentPage + 1; i++) html += this.renderPageButton(i, i === currentPage);
                html += '<span class="dt-ellipsis">...</span>';
                html += this.renderPageButton(totalPages, false);
            }
        }

        this.els.pages.innerHTML = html;
        this.els.pages.querySelectorAll('.dt-page-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                this.state.currentPage = parseInt(btn.dataset.page);
                this.renderBody();
                this.renderPagination();
                this.options.onPageChange?.(this.state.currentPage);
            });
        });
    }

    renderPageButton(page, active) {
        return `<button class="dt-page-btn ${active ? 'active' : ''}" data-page="${page}">${page}</button>`;
    }

    getSelectedData() {
        return this.state.filteredData.filter(row => this.state.selectedKeys.has(this.getRowKey(row)));
    }

    getSelectedKeys() {
        return Array.from(this.state.selectedKeys);
    }

    clearSelection() {
        this.state.selectedKeys.clear();
        this.updateSelectAllState();
        this.renderBody();
        this.emitSelectionChange();
    }

    destroy() {
        hideBatchBar();
        if (this.container) this.container.innerHTML = '';
    }
}