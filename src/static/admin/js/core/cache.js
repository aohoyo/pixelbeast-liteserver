/**
 * Cache - 请求缓存模块
 *
 * 用于缓存 API 请求结果，减少重复请求
 */

class CacheManager {
    constructor() {
        this.cache = new Map();
        this.timers = new Map();
    }

    /**
     * 获取缓存
     * @param {string} key - 缓存键
     * @returns {*} 缓存值或 undefined
     */
    get(key) {
        return this.cache.get(key);
    }

    /**
     * 设置缓存
     * @param {string} key - 缓存键
     * @param {*} value - 缓存值
     * @param {number} ttl - 过期时间（毫秒），默认 30 秒
     */
    set(key, value, ttl = 30000) {
        // 清除旧的定时器
        if (this.timers.has(key)) {
            clearTimeout(this.timers.get(key));
        }

        this.cache.set(key, value);

        // 设置过期
        this.timers.set(key, setTimeout(() => {
            this.delete(key);
        }, ttl));
    }

    /**
     * 删除缓存
     * @param {string} key - 缓存键
     */
    delete(key) {
        this.cache.delete(key);
        if (this.timers.has(key)) {
            clearTimeout(this.timers.get(key));
            this.timers.delete(key);
        }
    }

    /**
     * 检查缓存是否存在
     * @param {string} key - 缓存键
     * @returns {boolean}
     */
    has(key) {
        return this.cache.has(key);
    }

    /**
     * 清空所有缓存
     */
    clear() {
        this.cache.clear();
        this.timers.forEach(timer => clearTimeout(timer));
        this.timers.clear();
    }

    /**
     * 获取或设置缓存（常用模式）
     * @param {string} key - 缓存键
     * @param {Function} fetcher - 获取数据的函数
     * @param {number} ttl - 过期时间（毫秒）
     * @returns {Promise<*>} 缓存值或新获取的值
     */
    async getOrSet(key, fetcher, ttl = 30000) {
        if (this.has(key)) {
            return this.get(key);
        }

        const value = await fetcher();
        this.set(key, value, ttl);
        return value;
    }

    /**
     * 批量删除匹配的缓存
     * @param {RegExp|string} pattern - 匹配模式
     */
    deletePattern(pattern) {
        const regex = pattern instanceof RegExp ? pattern : new RegExp(pattern);
        for (const key of this.cache.keys()) {
            if (regex.test(key)) {
                this.delete(key);
            }
        }
    }

    /**
     * 获取缓存统计
     * @returns {Object} 统计信息
     */
    stats() {
        return {
            size: this.cache.size,
            keys: Array.from(this.cache.keys())
        };
    }
}

// 全局缓存实例
export const cache = new CacheManager();

// API 缓存装饰器
export function cached(ttl = 30000) {
    return function (target, propertyKey, descriptor) {
        const originalMethod = descriptor.value;
        descriptor.value = async function (...args) {
            const key = `${propertyKey}:${JSON.stringify(args)}`;
            return cache.getOrSet(key, () => originalMethod.apply(this, args), ttl);
        };
        return descriptor;
    };
}

export default CacheManager;