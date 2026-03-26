/**
 * BaseTab - 标签页基类
 *
 * 提供标签页的通用功能：
 * - 依赖注入
 * - 生命周期管理
 * - 事件注册
 * - 数据加载状态
 */

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
        if (this._loading) return;
        this._loading = true;
        
        try {
            await this.onRefresh();
        } catch (error) {
            this.onError(error, '刷新失败');
        } finally {
            this._loading = false;
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
        console.warn(`BaseTab: ${this.name} 未实现 onLoad`);
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