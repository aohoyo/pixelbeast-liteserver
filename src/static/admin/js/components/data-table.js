/**
 * DataTable 组件
 * 通用的数据表格组件，支持分页、选择、自定义渲染
 */

import { escapeHtml } from '../core/utils.js';

export class DataTable {
    constructor(options) {
        this.options = {
            container: null,           // 表格容器选择器或元素
            columns: [],              // 列配置
            data: [],                 // 数据
            pageSize: 20,             // 默认每页条数
            pageSizeOptions: [10, 20, 50, 100],
            selectable: false,        // 是否支持选择
            emptyText: '暂无数据',     // 空状态文本
            emptyHint: '',            // 空状态提示
            loadingText: '加载中...',  // 加载文本
            onPageChange: null,       // 页码变化回调
            onSelectionChange: null,  // 选择变化回调
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

    /**
     * 渲染表格结构
     */
    render() {
        const { columns, selectable, pageSizeOptions } = this.options;

        this.container.innerHTML = `
            <div class="dt-container">
                <div class="dt-table-wrapper">
                    <table class="dt-table">
                        <thead>
                            <tr>
                                ${selectable ? `
                                    <th class="dt-checkbox-col">
                                        <input type="checkbox" class="dt-checkbox dt-select-all">
                                    </th>
                                ` : ''}
                                ${columns.map(col => `
                                    <th class="${col.className || ''}" style="${col.width ? `width:${col.width}` : ''}">
                                        ${col.title}
                                    </th>
                                `).join('')}
                            </tr>
                        </thead>
                        <tbody class="dt-body">
                            <tr>
                                <td colspan="${selectable ? columns.length + 1 : columns.length}" class="dt-loading-cell">
                                    <div class="dt-loading-content">
                                        <div class="dt-loading-spinner"></div>
                                        <span>${this.options.loadingText}</span>
                                    </div>
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
                <div class="dt-pagination">
                    <div class="dt-pagination-left">
                        <div class="dt-info">
                            共 <strong class="dt-total">0</strong> 条记录
                        </div>
                        <div class="dt-size-selector">
                            <span>每页</span>
                            <select class="dt-page-size">
                                ${pageSizeOptions.map(size => `
                                    <option value="${size}" ${size === this.state.pageSize ? 'selected' : ''}>${size}</option>
                                `).join('')}
                            </select>
                            <span>条</span>
                        </div>
                    </div>
                    <div class="dt-pagination-right">
                        <nav class="dt-nav">
                            <button class="dt-btn dt-prev" disabled>
                                <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <polyline points="15 18 9 12 15 6"></polyline>
                                </svg>
                                <span>上一页</span>
                            </button>
                            <div class="dt-pages"></div>
                            <button class="dt-btn dt-next" disabled>
                                <span>下一页</span>
                                <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <polyline points="9 18 15 12 9 6"></polyline>
                                </svg>
                            </button>
                        </nav>
                        <div class="dt-jump">
                            <span>跳至</span>
                            <input type="number" class="dt-jump-input" min="1">
                            <span>页</span>
                        </div>
                    </div>
                </div>
            </div>
        `;

        // 缓存元素引用
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

    /**
     * 绑定事件
     */
    bindEvents() {
        // 全选
        if (this.els.selectAll) {
            this.els.selectAll.addEventListener('change', (e) => {
                const checked = e.target.checked;
                if (checked) {
                    this.getCurrentPageData().forEach(row => {
                        this.state.selectedKeys.add(this.getRowKey(row));
                    });
                } else {
                    this.getCurrentPageData().forEach(row => {
                        this.state.selectedKeys.delete(this.getRowKey(row));
                    });
                }
                this.renderBody();
                this.updateSelectAllState();
                this.emitSelectionChange();
            });
        }

        // 分页大小
        this.els.pageSize?.addEventListener('change', (e) => {
            this.state.pageSize = parseInt(e.target.value);
            this.state.currentPage = 1;
            this.renderBody();
            this.renderPagination();
        });

        // 上一页
        this.els.prevBtn?.addEventListener('click', () => {
            if (this.state.currentPage > 1) {
                this.state.currentPage--;
                this.renderBody();
                this.renderPagination();
                this.options.onPageChange?.(this.state.currentPage);
            }
        });

        // 下一页
        this.els.nextBtn?.addEventListener('click', () => {
            const totalPages = this.getTotalPages();
            if (this.state.currentPage < totalPages) {
                this.state.currentPage++;
                this.renderBody();
                this.renderPagination();
                this.options.onPageChange?.(this.state.currentPage);
            }
        });

        // 快速跳转
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

    /**
     * 获取行唯一键
     */
    getRowKey(row) {
        return row.id || row.key || row.username || row.name || JSON.stringify(row);
    }

    /**
     * 更新数据
     */
    updateData(data) {
        this.state.filteredData = data || [];
        this.state.currentPage = 1;
        this.state.selectedKeys.clear();
        this.updateSelectAllState();
        this.renderBody();
        this.renderPagination();
    }

    /**
     * 设置加载状态
     */
    setLoading(loading) {
        this.state.loading = loading;
        if (loading) {
            this.els.tbody.innerHTML = `
                <tr>
                    <td colspan="${this.getColSpan()}" class="dt-loading-cell">
                        <div class="dt-loading-content">
                            <div class="dt-loading-spinner"></div>
                            <span>${this.options.loadingText}</span>
                        </div>
                    </td>
                </tr>
            `;
        } else {
            // 加载完成后重新渲染
            this.renderBody();
        }
    }

    /**
     * 获取当前页数据
     */
    getCurrentPageData() {
        const start = (this.state.currentPage - 1) * this.state.pageSize;
        const end = start + this.state.pageSize;
        return this.state.filteredData.slice(start, end);
    }

    /**
     * 获取总页数
     */
    getTotalPages() {
        return Math.ceil(this.state.filteredData.length / this.state.pageSize) || 1;
    }

    /**
     * 获取列跨度
     */
    getColSpan() {
        return this.options.selectable
            ? this.options.columns.length + 1
            : this.options.columns.length;
    }

    /**
     * 渲染表格主体
     */
    renderBody() {
        const { columns, selectable, emptyText, emptyHint } = this.options;
        const data = this.getCurrentPageData();

        // 空状态
        if (this.state.filteredData.length === 0) {
            this.els.tbody.innerHTML = `
                <tr>
                    <td colspan="${this.getColSpan()}" class="dt-empty-cell">
                        <div class="dt-empty">
                            <div class="dt-empty-text">${emptyText}</div>
                            ${emptyHint ? `<div class="dt-empty-hint">${emptyHint}</div>` : ''}
                        </div>
                    </td>
                </tr>
            `;
            return;
        }

        // 渲染数据行
        this.els.tbody.innerHTML = data.map(row => {
            const rowKey = this.getRowKey(row);
            const isSelected = this.state.selectedKeys.has(rowKey);

            return `
                <tr data-key="${escapeHtml(rowKey)}">
                    ${selectable ? `
                        <td class="dt-checkbox-col">
                            <input type="checkbox" class="dt-checkbox dt-row-checkbox"
                                data-key="${escapeHtml(rowKey)}"
                                ${isSelected ? 'checked' : ''}>
                        </td>
                    ` : ''}
                    ${columns.map(col => `
                        <td class="${col.className || ''}">
                            ${col.render
                                ? col.render(row[col.dataIndex], row, this)
                                : escapeHtml(row[col.dataIndex] ?? '-')
                            }
                        </td>
                    `).join('')}
                </tr>
            `;
        }).join('');

        // 绑定行事件
        this.bindRowEvents();
    }

    /**
     * 绑定行内事件
     */
    bindRowEvents() {
        const { selectable } = this.options;

        // 行选择
        if (selectable) {
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

    /**
     * 更新全选框状态
     */
    updateSelectAllState() {
        if (!this.els.selectAll) return;

        const currentPageData = this.getCurrentPageData();
        if (currentPageData.length === 0) {
            this.els.selectAll.checked = false;
            this.els.selectAll.indeterminate = false;
            return;
        }

        const selectedCount = currentPageData.filter(row =>
            this.state.selectedKeys.has(this.getRowKey(row))
        ).length;

        if (selectedCount === 0) {
            this.els.selectAll.checked = false;
            this.els.selectAll.indeterminate = false;
        } else if (selectedCount === currentPageData.length) {
            this.els.selectAll.checked = true;
            this.els.selectAll.indeterminate = false;
        } else {
            this.els.selectAll.checked = false;
            this.els.selectAll.indeterminate = true;
        }
    }

    /**
     * 触发选择变化事件
     */
    emitSelectionChange() {
        this.options.onSelectionChange?.({
            selectedKeys: Array.from(this.state.selectedKeys),
            selectedCount: this.state.selectedKeys.size
        });
    }

    /**
     * 渲染分页
     */
    renderPagination() {
        const totalPages = this.getTotalPages();
        const { currentPage } = this.state;

        // 更新总数
        if (this.els.total) {
            this.els.total.textContent = this.state.filteredData.length;
        }

        // 更新按钮状态
        if (this.els.prevBtn) this.els.prevBtn.disabled = currentPage <= 1;
        if (this.els.nextBtn) this.els.nextBtn.disabled = currentPage >= totalPages;

        // 生成页码
        if (!this.els.pages) return;

        let html = '';
        const maxVisible = 5;

        if (totalPages <= maxVisible) {
            for (let i = 1; i <= totalPages; i++) {
                html += this.renderPageButton(i, i === currentPage);
            }
        } else {
            if (currentPage <= 3) {
                for (let i = 1; i <= 4; i++) {
                    html += this.renderPageButton(i, i === currentPage);
                }
                html += '<span class="dt-ellipsis">...</span>';
                html += this.renderPageButton(totalPages, false);
            } else if (currentPage >= totalPages - 2) {
                html += this.renderPageButton(1, false);
                html += '<span class="dt-ellipsis">...</span>';
                for (let i = totalPages - 3; i <= totalPages; i++) {
                    html += this.renderPageButton(i, i === currentPage);
                }
            } else {
                html += this.renderPageButton(1, false);
                html += '<span class="dt-ellipsis">...</span>';
                for (let i = currentPage - 1; i <= currentPage + 1; i++) {
                    html += this.renderPageButton(i, i === currentPage);
                }
                html += '<span class="dt-ellipsis">...</span>';
                html += this.renderPageButton(totalPages, false);
            }
        }

        this.els.pages.innerHTML = html;

        // 绑定页码点击
        this.els.pages.querySelectorAll('.dt-page-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                this.state.currentPage = parseInt(btn.dataset.page);
                this.renderBody();
                this.renderPagination();
                this.options.onPageChange?.(this.state.currentPage);
            });
        });
    }

    /**
     * 渲染页码按钮
     */
    renderPageButton(page, active) {
        return `<button class="dt-page-btn ${active ? 'active' : ''}" data-page="${page}">${page}</button>`;
    }

    /**
     * 获取选中的数据
     */
    getSelectedData() {
        return this.state.filteredData.filter(row =>
            this.state.selectedKeys.has(this.getRowKey(row))
        );
    }

    /**
     * 获取选中的 keys
     */
    getSelectedKeys() {
        return Array.from(this.state.selectedKeys);
    }

    /**
     * 清空选择
     */
    clearSelection() {
        this.state.selectedKeys.clear();
        this.updateSelectAllState();
        this.renderBody();
        this.emitSelectionChange();
    }

    /**
     * 销毁组件
     */
    destroy() {
        if (this.container) {
            this.container.innerHTML = '';
        }
    }
}
