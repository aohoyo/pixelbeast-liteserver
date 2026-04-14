/**
 * EventBus - 事件总线系统
 *
 * 提供发布-订阅模式的事件系统，用于模块间通信
 * 支持命名空间、一次性监听、异步事件等特性
 */

export default class EventBus {
    constructor() {
        this.events = new Map();
        this.onceEvents = new Map();
        this.wildcardListeners = [];
    }

    /**
     * 注册事件监听器
     * @param {string} event - 事件名称，支持命名空间如 'file:upload'
     * @param {Function} callback - 回调函数 (data) => void
     * @returns {Function} 取消订阅函数
     */
    on(event, callback) {
        if (!this.events.has(event)) {
            this.events.set(event, []);
        }
        this.events.get(event).push(callback);

        // 返回取消订阅函数
        return () => this.off(event, callback);
    }

    /**
     * 注册一次性事件监听器
     * @param {string} event - 事件名称
     * @param {Function} callback - 回调函数
     */
    once(event, callback) {
        if (!this.onceEvents.has(event)) {
            this.onceEvents.set(event, []);
        }
        this.onceEvents.get(event).push(callback);
    }

    /**
     * 移除事件监听器
     * @param {string} event - 事件名称
     * @param {Function} callback - 要移除的回调函数（可选）
     */
    off(event, callback) {
        if (!callback) {
            // 移除该事件的所有监听器
            this.events.delete(event);
            this.onceEvents.delete(event);
            return;
        }

        // 移除特定回调
        const callbacks = this.events.get(event);
        if (callbacks) {
            const index = callbacks.indexOf(callback);
            if (index > -1) {
                callbacks.splice(index, 1);
            }
        }
    }

    /**
     * 触发事件
     * @param {string} event - 事件名称
     * @param {*} data - 事件数据
     * @param {Object} options - 选项 { async: boolean, timeout: number }
     * @returns {Promise|void} 如果是异步模式返回 Promise
     */
    emit(event, data, options = {}) {
        const { async = false } = options;

        // 处理通配符监听器
        this._emitWildcard(event, data);

        // 处理命名空间事件 (如 'file:upload' 匹配 'file:*')
        const namespace = event.split(':')[0];
        if (namespace) {
            this._emitWildcard(`${namespace}:*`, { ...data, __originalEvent: event });
        }

        // 处理普通监听器
        const callbacks = this.events.get(event);
        if (callbacks && callbacks.length > 0) {
            if (async) {
                return this._emitAsync(callbacks, data);
            } else {
                this._emitSync(callbacks, data);
            }
        }

        // 处理一次性监听器
        const onceCallbacks = this.onceEvents.get(event);
        if (onceCallbacks && onceCallbacks.length > 0) {
            this.onceEvents.delete(event);
            this._emitSync(onceCallbacks, data);
        }
    }

    /**
     * 注册通配符监听器
     * @param {string} pattern - 匹配模式，如 'file:*' 匹配 'file:upload', 'file:delete' 等
     * @param {Function} callback - 回调函数，接收 (eventName, data)
     */
    match(pattern, callback) {
        this.wildcardListeners.push({ pattern, callback });
    }

    /**
     * 同步触发监听器
     * @private
     */
    _emitSync(callbacks, data) {
        for (const callback of callbacks) {
            try {
                callback(data);
            } catch (e) {
                console.error(`Event listener error:`, e);
                this.emit('error', { event: e, context: data });
            }
        }
    }

    /**
     * 异步触发监听器
     * @private
     */
    async _emitAsync(callbacks, data) {
        const results = [];
        for (const callback of callbacks) {
            try {
                const result = await callback(data);
                results.push(result);
            } catch (e) {
                console.error(`Async event listener error:`, e);
                this.emit('error', { event: e, context: data });
                results.push({ error: e });
            }
        }
        return results;
    }

    /**
     * 触发通配符监听器
     * @private
     */
    _emitWildcard(event, data) {
        for (const { pattern, callback } of this.wildcardListeners) {
            if (this._matchPattern(pattern, event)) {
                try {
                    callback(event, data);
                } catch (e) {
                    console.error(`Wildcard listener error for "${pattern}":`, e);
                }
            }
        }
    }

    /**
     * 匹配通配符模式
     * @private
     */
    _matchPattern(pattern, event) {
        if (pattern === '*') return true;
        if (pattern.endsWith('*')) {
            const prefix = pattern.slice(0, -2); // 移除 ':*'
            return event.startsWith(prefix + ':') || event.startsWith(prefix + '.');
        }
        return pattern === event;
    }

    /**
     * 等待事件触发
     * @param {string} event - 要等待的事件名称
     * @param {Object} options - 选项 { timeout: number }
     * @returns {Promise} 事件数据的 Promise
     */
    waitFor(event, options = {}) {
        return new Promise((resolve, reject) => {
            const timeout = options.timeout || 30000;

            const timer = setTimeout(() => {
                this.off(event, handler);
                reject(new Error(`Event "${event}" timeout after ${timeout}ms`));
            }, timeout);

            const handler = (data) => {
                clearTimeout(timer);
                resolve(data);
            };

            this.once(event, handler);
        });
    }

    /**
     * 清除所有监听器或特定事件的监听器
     * @param {string} event - 可选，指定要清除的事件
     */
    clear(event) {
        if (event) {
            this.events.delete(event);
            this.onceEvents.delete(event);
        } else {
            this.events.clear();
            this.onceEvents.clear();
            this.wildcardListeners = [];
        }
    }

    /**
     * 获取事件监听器数量
     * @param {string} event - 可选，指定事件
     * @returns {number|Object} 监听器数量
     */
    listenerCount(event) {
        if (event) {
            const callbacks = this.events.get(event);
            const once = this.onceEvents.get(event);
            return (callbacks?.length || 0) + (once?.length || 0);
        }
        let total = this.wildcardListeners.length;
        for (const [, callbacks] of this.events) {
            total += callbacks.length;
        }
        for (const [, callbacks] of this.onceEvents) {
            total += callbacks.length;
        }
        return total;
    }

    /**
     * 获取所有已注册的事件名称
     * @returns {Array<string>} 事件名称数组
     */
    eventNames() {
        const names = new Set();
        for (const event of this.events.keys()) {
            names.add(event);
        }
        for (const event of this.onceEvents.keys()) {
            names.add(event);
        }
        return Array.from(names);
    }
}

// 核心事件常量
export const Events = Object.freeze({
    /** 站点配置变更 */
    SITE_CHANGED: 'site:changed',
    /** SSL 证书变更 */
    SSL_CHANGED: 'ssl:changed',
    /** FTP 配置变更 */
    FTP_CHANGED: 'ftp:changed',
    /** 系统配置变更 */
    CONFIG_CHANGED: 'config:changed',
    /** 用户登录 */
    USER_LOGIN: 'user:login',
    /** 用户登出 */
    USER_LOGOUT: 'user:logout',
});

// 创建全局默认实例
export const globalEvents = new EventBus();

// 便捷方法：直接使用全局实例
export const on = (event, callback) => globalEvents.on(event, callback);
export const once = (event, callback) => globalEvents.once(event, callback);
export const off = (event, callback) => globalEvents.off(event, callback);
export const emit = (event, data) => globalEvents.emit(event, data);
