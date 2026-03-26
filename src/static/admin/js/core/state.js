/**
 * StateManager - 统一状态管理
 *
 * 提供响应式的状态管理，支持：
 * - 状态订阅和通知
 * - 路径式状态访问
 * - 不可变更新
 */

export default class StateManager {
    constructor() {
        this.state = {
            // 当前激活的标签页
            currentTab: 'status',

            // 用户信息
            user: null,

            // 服务器配置
            config: null,

            // 系统信息（初始化时设置）
            system: {
                os: 'linux',        // 'linux' | 'windows' | 'darwin'
                arch: 'amd64',
                hostname: '',
                isWindows: false,
                isLinux: true,
                isMac: false
            },

            // 服务状态
            services: {
                http: { running: true, port: 1880 },
                ftp: { running: true, port: 2121 }
            },

            // 文件路径
            paths: {
                http: '/',
                ftp: '/'
            },

            // 加载状态
            loading: {},

            // 错误状态
            errors: {}
        };

        this.listeners = new Map();
        this.history = [];
    }

    /**
     * 初始化系统信息（应用启动时调用一次）
     * @param {Object} sysInfo - 系统信息
     */
    initSystem(sysInfo) {
        const os = (sysInfo.os || 'linux').toLowerCase();
        this.state.system = {
            os: os,
            arch: sysInfo.arch || 'amd64',
            hostname: sysInfo.hostname || '',
            isWindows: os === 'windows',
            isLinux: os === 'linux',
            isMac: os === 'darwin'
        };
    }

    /**
     * 获取状态值
     * @param {string} path - 点分隔的路径，如 'services.http.running'
     * @returns {*} 状态值
     */
    get(path) {
        return path.split('.').reduce((obj, key) => obj?.[key], this.state);
    }

    /**
     * 设置状态值
     * @param {string} path - 点分隔的路径
     * @param {*} value - 新值
     * @param {boolean} notify - 是否通知订阅者（默认 true）
     */
    set(path, value, notify = true) {
        const keys = path.split('.');
        const lastKey = keys.pop();

        // 找到父对象
        let parent = this.state;
        for (const key of keys) {
            if (!(key in parent)) {
                parent[key] = {};
            }
            parent = parent[key];
        }

        // 保存旧值用于比较
        const oldValue = parent[lastKey];

        // 设置新值
        parent[lastKey] = value;

        // 保存历史（用于撤销，可选）
        this.history.push({ path, oldValue, newValue: value, timestamp: Date.now() });
        if (this.history.length > 50) {
            this.history.shift();
        }

        // 通知订阅者
        if (notify && oldValue !== value) {
            this._notify(path, value, oldValue);
        }
    }

    /**
     * 批量更新状态
     * @param {Object} updates - 更新对象 { path: value }
     */
    batch(updates) {
        for (const [path, value] of Object.entries(updates)) {
            this.set(path, value, false);
        }
        // 统一通知
        this._notify('*', this.state, null);
    }

    /**
     * 订阅状态变化
     * @param {string} path - 要监听的路径
     * @param {Function} callback - 回调函数 (newValue, oldValue) => void
     * @returns {Function} 取消订阅函数
     */
    subscribe(path, callback) {
        if (!this.listeners.has(path)) {
            this.listeners.set(path, []);
        }
        this.listeners.get(path).push(callback);

        // 返回取消订阅函数
        return () => {
            const callbacks = this.listeners.get(path);
            if (callbacks) {
                const index = callbacks.indexOf(callback);
                if (index > -1) {
                    callbacks.splice(index, 1);
                }
            }
        };
    }

    /**
     * 订阅一次后自动取消
     * @param {string} path - 要监听的路径
     * @param {Function} callback - 回调函数
     */
    once(path, callback) {
        const unsubscribe = this.subscribe(path, (newValue, oldValue) => {
            callback(newValue, oldValue);
            unsubscribe();
        });
    }

    /**
     * 通知订阅者
     * @private
     */
    _notify(path, newValue, oldValue) {
        // 通知精确路径的订阅者
        const exactCallbacks = this.listeners.get(path);
        if (exactCallbacks) {
            for (const cb of exactCallbacks) {
                try {
                    cb(newValue, oldValue);
                } catch (e) {
                    console.error(`State listener error for "${path}":`, e);
                }
            }
        }

        // 通知通配符订阅者
        const wildcardCallbacks = this.listeners.get('*');
        if (wildcardCallbacks) {
            for (const cb of wildcardCallbacks) {
                try {
                    cb({ path, newValue, oldValue });
                } catch (e) {
                    console.error('Wildcard state listener error:', e);
                }
            }
        }

        // 通知父路径的订阅者
        const parts = path.split('.');
        for (let i = parts.length - 1; i > 0; i--) {
            const parentPath = parts.slice(0, i).join('.');
            const parentCallbacks = this.listeners.get(parentPath);
            if (parentCallbacks) {
                for (const cb of parentCallbacks) {
                    try {
                        cb(this.get(parentPath), oldValue);
                    } catch (e) {
                        console.error(`Parent state listener error for "${parentPath}":`, e);
                    }
                }
            }
        }
    }

    /**
     * 重置状态
     * @param {string} path - 要重置的路径（可选，不传则重置全部）
     */
    reset(path) {
        if (path) {
            const keys = path.split('.');
            const lastKey = keys.pop();
            let parent = this.state;
            for (const key of keys) {
                parent = parent[key];
            }
            delete parent[lastKey];
            this._notify(path, undefined, this.get(path));
        } else {
            const oldState = { ...this.state };
            this.state = {
                currentTab: 'status',
                user: null,
                config: null,
                services: { http: { running: true, port: 1880 }, ftp: { running: true, port: 2121 } },
                paths: { http: '/', ftp: '/' },
                loading: {},
                errors: {}
            };
            this._notify('*', this.state, oldState);
        }
    }

    /**
     * 获取整个状态副本
     * @returns {Object} 状态的深拷贝
     */
    toJSON() {
        return JSON.parse(JSON.stringify(this.state));
    }

    /**
     * 从 JSON 恢复状态
     * @param {Object} data - JSON 对象
     */
    fromJSON(data) {
        if (data && typeof data === 'object') {
            const oldState = { ...this.state };
            this.state = { ...this.state, ...data };
            this._notify('*', this.state, oldState);
        }
    }

    /**
     * 获取状态历史
     * @param {number} limit - 返回的记录数
     * @returns {Array} 历史记录
     */
    getHistory(limit = 10) {
        return this.history.slice(-limit);
    }
}
