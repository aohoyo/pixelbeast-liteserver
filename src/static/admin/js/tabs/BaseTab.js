/**
 * BaseTab - 标签页基类
 *
 * 提供标签页的通用功能：
 * - 依赖注入
 * - 生命周期管理
 * - 事件注册
 * - 数据加载状态
 * - 模态框操作
 * - 通用行事件
 */

import { copyToClipboard } from '../core/utils.js';

export class BaseTab {
    /**
     * @param {Object} deps - 依赖注入
     * @param {string} name - 标签页名称
     */
    constructor(deps, name) {
        this.state = deps.state;
        this.api = deps.api;
        this.toast = deps.toast;
        this.message = deps.message;
        this.dialog = deps.dialog;
        this.events = deps.events;
        this.name = name;
        
        // 数据加载状态
        this._loaded = false;
        this._loading = false;
    }

    /**
     * 初始化标签页
     * 自动注册切换事件
     */
    init() {
        // 注册切换事件
        this.events.match(`tab:switch:${this.name}`, () => {
            this.onActivate();
        });

        // 注册全局刷新事件
        this.events.on(`refresh:${this.name}`, () => {
            this.refresh();
        });

        // 子类初始化
        this.onInit();
    }

    /**
     * 激活标签页
     */
    onActivate() {
        if (!this._loaded) {
            this.load();
        } else {
            this.refresh();
        }
    }

    /**
     * 加载数据（首次）
     */
    async load() {
        if (this._loading) return;
        this._loading = true;

        try {
            await this.onLoad();
            this._loaded = true;
        } catch (error) {
            this.onError(error, '加载失败');
        } finally {
            this._loading = false;
        }
    }

    /**
     * 刷新数据
     */
    async refresh() {
        if (this._loading) {
            this._pendingRefresh = true;
            return;
        }
        this._loading = true;

        try {
            await this.onRefresh();
        } catch (error) {
            this.onError(error, '刷新失败');
        } finally {
            this._loading = false;
            if (this._pendingRefresh) {
                this._pendingRefresh = false;
                this.refresh();
            }
        }
    }

    /**
     * 销毁标签页
     */
    destroy() {
        this._loaded = false;
        this._loading = false;
        this.onDestroy();
    }

    // ========== 子类实现 ==========

    /**
     * 初始化时调用（子类可选实现）
     */
    onInit() {}

    /**
     * 加载数据（子类必须实现）
     */
    async onLoad() {
        // 子类必须实现
    }

    /**
     * 刷新数据（子类可选实现，默认调用 onLoad）
     */
    async onRefresh() {
        await this.onLoad();
    }

    /**
     * 销毁时调用（子类可选实现）
     */
    onDestroy() {}

    /**
     * 错误处理（子类可覆盖）
     */
    onError(error, context) {
        console.error(`[${this.name}] ${context}:`, error);
        this.toast?.error(`${context}: ${error.message}`);
    }

    // ========== 工具方法 ==========

    /**
     * 获取元素
     */
    $(selector) {
        return document.querySelector(selector);
    }

    /**
     * 获取所有元素
     */
    $$(selector) {
        return document.querySelectorAll(selector);
    }

    /**
     * 创建服务控制方法集合（提取公共的启停/重启/重载逻辑）
     * @param {Object} config
     * @param {string} config.apiPrefix - API 前缀, e.g. '/api/service/sites'
     * @param {string} config.statusId - 状态显示元素 ID
     * @param {string} config.toggleId - 切换按钮元素 ID
     * @param {string} config.label - 服务名称 (用于提示, e.g. '站点服务')
     * @returns {{ toggleService, restartService, reloadConfig, updateServiceStatus }}
     */
    createServiceControls({ apiPrefix, statusId, toggleId, label }) {
        const msg = this.message || this.toast;

        return {
            toggleService: async () => {
                const btn = this.$(`#${toggleId}`);
                const isRunning = btn?.classList.contains('running');
                try {
                    if (isRunning) {
                        await this.api.post(`${apiPrefix}/stop`);
                        msg.success(`${label}已停止`);
                    } else {
                        await this.api.post(`${apiPrefix}/start`);
                        msg.success(`${label}已启动`);
                    }
                } catch (error) {
                    msg.error('操作失败: ' + error.message);
                }
            },

            restartService: async () => {
                try {
                    await this.api.post(`${apiPrefix}/restart`);
                    msg.success(`${label}已重启`);
                } catch (error) {
                    msg.error('重启失败: ' + error.message);
                }
            },

            reloadConfig: async () => {
                try {
                    await this.api.post(`${apiPrefix}/reload`);
                    msg.success(`${label}配置已重载`);
                } catch (error) {
                    msg.error('重载失败: ' + error.message);
                }
            },

            updateServiceStatus: (running) => {
                const statusEl = this.$(`#${statusId}`);
                const toggleBtn = this.$(`#${toggleId}`);

                if (statusEl) {
                    statusEl.textContent = running ? '运行中' : '已停止';
                    statusEl.classList.toggle('running', running);
                    statusEl.classList.toggle('stopped', !running);
                }

                if (toggleBtn) {
                    toggleBtn.classList.toggle('running', running);
                    toggleBtn.classList.toggle('stopped', running);
                    const textSpan = toggleBtn.querySelector('.btn-text');
                    if (textSpan) {
                        textSpan.textContent = running ? '停止' : '启动';
                    }
                }
            }
        };
    }

    // ========== 模态框操作 ==========

    /**
     * 显示模态框
     * @param {string} id - 模态框 ID
     */
    showModal(id) {
        this.$(`#${id}`)?.classList.add('active');
    }

    /**
     * 隐藏模态框
     * @param {string} id - 模态框 ID
     */
    hideModal(id) {
        this.$(`#${id}`)?.classList.remove('active');
    }

    /**
     * 绑定模态框关闭事件（关闭按钮、取消按钮、遮罩层点击）
     * @param {string} modalId - 模态框 ID
     * @param {Function} onClose - 关闭回调
     * @param {Object} [opts] - 可选配置
     * @param {string} [opts.cancelId] - 取消按钮 ID（默认 {modalId}-cancel）
     * @param {string} [opts.confirmId] - 确认按钮 ID
     * @param {Function} [opts.onConfirm] - 确认回调
     */
    bindModalClose(modalId, onClose, opts = {}) {
        const modal = this.$(`#${modalId}`);
        if (!modal) return;

        // 关闭按钮
        modal.querySelector('.modal-close')?.addEventListener('click', onClose);
        // 遮罩层点击关闭
        modal.querySelector('.modal-overlay')?.addEventListener('click', onClose);
        // 取消按钮
        const cancelId = opts.cancelId || `${modalId}-cancel`;
        this.$(`#${cancelId}`)?.addEventListener('click', onClose);
        // 确认按钮
        if (opts.confirmId && opts.onConfirm) {
            this.$(`#${opts.confirmId}`)?.addEventListener('click', opts.onConfirm);
        }
    }

    // ========== 通用行事件 ==========

    /**
     * 绑定根目录链接（跳转文件管理）
     */
    bindBrowseLinks() {
        this.$$('.root-link[data-browse-path]').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const path = link.dataset.browsePath || '.';
                if (path && window.app?.switchTab) {
                    window.app.switchTab('files');
                    setTimeout(() => {
                        this.events.emit('files:navigate', path);
                    }, 150);
                }
            });
        });
    }

    /**
     * 绑定快速链接复制按钮
     */
    bindCopyLinks() {
        this.$$('.quick-link-copy').forEach(btn => {
            btn.addEventListener('click', () => {
                copyToClipboard(btn.dataset.link);
                this.message.success('链接已复制');
            });
        });
    }

    // ========== UI 状态 ==========

    /**
     * 显示加载状态
     */
    showLoading(container) {
        if (typeof container === 'string') {
            container = this.$(container);
        }
        if (container) {
            container.innerHTML = `<p class="loading">加载中...</p>`;
        }
    }

    /**
     * 显示空状态
     */
    showEmpty(container, text = '暂无数据') {
        if (typeof container === 'string') {
            container = this.$(container);
        }
        if (container) {
            container.innerHTML = `<p class="loading">${text}</p>`;
        }
    }

    /**
     * 设置元素文本内容
     * @param {string} selector - 选择器
     * @param {*} value - 要设置的值
     */
    setText(selector, value) {
        const el = typeof selector === 'string' ? this.$(selector) : selector;
        if (el && value !== undefined && value !== null) {
            el.textContent = value;
        }
    }

    /**
     * 设置元素 HTML 内容
     * @param {string} selector - 选择器
     * @param {string} html - HTML 内容
     */
    setHTML(selector, html) {
        const el = typeof selector === 'string' ? this.$(selector) : selector;
        if (el) {
            el.innerHTML = html;
        }
    }
}

export default BaseTab;