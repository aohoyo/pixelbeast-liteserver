/**
 * API - API 请求封装
 *
 * 封装所有与后端的通信，处理：
 * - 认证和会话
 * - CSRF token 管理
 * - 错误处理
 * - 请求/响应拦截
 * - 请求缓存
 */

import StateManager from './state.js';
import { globalEvents } from './events.js';
import { cache } from './cache.js';

// 动态获取管理面板基础路径
const API_BASE = (function() {
    const path = window.location.pathname;
    return path.replace(/\/(index\.html)?$/, '') || '';
})();

// CSRF token
let csrfToken = null;

/**
 * API 类
 */
class API {
    constructor(state) {
        this.state = state;
        this.requestInterceptors = [];
        this.responseInterceptors = [];
        this.baseURL = API_BASE;
        this.cacheConfig = {
            '/api/status': 5000,           // 状态缓存 5 秒
            '/api/system/status': 5000,    // 系统状态 5 秒
            '/api/sites': 10000,           // 站点列表 10 秒
            '/api/ftp/users': 10000,       // FTP 用户 10 秒
            '/api/logs/stats': 30000,      // 日志统计 30 秒
        };
    }

    /**
     * 处理统一响应格式 {code, message, data}
     * @param {Object} data - 解析后的 JSON 数据
     * @returns {Object} 处理后的数据
     */
    _handleUnifiedResponse(data) {
        // 新格式 {code: 200, message: "success", data: {...}}
        if (data && typeof data === 'object' && 'code' in data) {
            if (data.code === 200) {
                // 成功响应，返回 data 字段内容（如果有）
                if (data.data !== undefined) {
                    return data.data;
                }
                // 没有 data 字段，返回整个对象（包含 message）
                return data;
            } else {
                // 错误响应
                throw new Error(data.message || '请求失败');
            }
        }
        // 旧格式或其他格式，直接返回
        return data;
    }

    /**
     * 设置 CSRF token
     * @param {string} token - CSRF token
     */
    setCSRFToken(token) {
        csrfToken = token;
    }

    /**
     * 获取 CSRF token
     * @returns {string} CSRF token
     */
    getCSRFToken() {
        return csrfToken;
    }

    /**
     * 添加请求拦截器
     * @param {Function} interceptor - (config) => config
     */
    addRequestInterceptor(interceptor) {
        this.requestInterceptors.push(interceptor);
    }

    /**
     * 添加响应拦截器
     * @param {Function} interceptor - (response) => response
     */
    addResponseInterceptor(interceptor) {
        this.responseInterceptors.push(interceptor);
    }

    /**
     * 发送 GET 请求（自动解析 JSON，支持缓存）
     * @param {string} endpoint - API 端点
     * @param {Object} options - 选项 { useCache: true, ttl: 30000 }
     * @returns {Promise<Object>} 解析后的数据
     */
    async getJSON(endpoint, options = {}) {
        const { useCache = true, ttl } = options;
        const cacheKey = `api:${endpoint}`;

        // 检查缓存配置
        const cacheTTL = ttl ?? this.cacheConfig[endpoint];

        // 使用缓存
        if (useCache && cacheTTL && cache.has(cacheKey)) {
            return cache.get(cacheKey);
        }

        const response = await this.get(endpoint);
        if (!response) return null;
        
        const data = await this.parseJSON(response);

        // 缓存结果
        if (useCache && cacheTTL && data) {
            cache.set(cacheKey, data, cacheTTL);
        }

        return data;
    }

    /**
     * 清除 API 缓存
     * @param {string} pattern - 匹配模式（可选）
     */
    clearCache(pattern) {
        if (pattern) {
            cache.deletePattern(`api:${pattern}`);
        } else {
            cache.deletePattern(/^api:/);
        }
    }

    /**
     * 发送 GET 请求
     * @param {string} endpoint - API 端点
     * @param {Object} options - fetch 选项
     * @returns {Promise<Response>}
     */
    async get(endpoint, options = {}) {
        return this.request(endpoint, { ...options, method: 'GET' });
    }

    /**
     * 发送 POST 请求
     * @param {string} endpoint - API 端点
     * @param {Object} data - 请求体数据
     * @param {Object} options - fetch 选项
     * @returns {Promise<Response>}
     */
    async post(endpoint, data, options = {}) {
        const config = {
            ...options,
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            }
        };

        if (data) {
            config.body = JSON.stringify(data);
        }

        return this.request(endpoint, config);
    }

    /**
     * 发送 POST 请求（自动解析 JSON）
     * @param {string} endpoint - API 端点
     * @param {Object} data - 请求体数据
     * @returns {Promise<Object>} 解析后的数据
     */
    async postJSON(endpoint, data) {
        const response = await this.post(endpoint, data);
        if (!response) return null;
        return this.parseJSON(response);
    }

    /**
     * 发送 PUT 请求
     * @param {string} endpoint - API 端点
     * @param {Object} data - 请求体数据
     * @param {Object} options - fetch 选项
     * @returns {Promise<Response>}
     */
    async put(endpoint, data, options = {}) {
        const config = {
            ...options,
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            }
        };

        if (data) {
            config.body = JSON.stringify(data);
        }

        return this.request(endpoint, config);
    }

    /**
     * 发送 DELETE 请求
     * @param {string} endpoint - API 端点
     * @param {Object} options - fetch 选项
     * @returns {Promise<Response>}
     */
    async delete(endpoint, options = {}) {
        return this.request(endpoint, { ...options, method: 'DELETE' });
    }

    /**
     * 上传文件（FormData）
     * @param {string} endpoint - API 端点
     * @param {FormData} formData - 表单数据
     * @param {Function} onProgress - 进度回调
     * @returns {Promise} 上传结果
     */
    upload(endpoint, formData, onProgress) {
        return new Promise((resolve, reject) => {
            const xhr = new XMLHttpRequest();

            xhr.upload.addEventListener('progress', (e) => {
                if (e.lengthComputable && onProgress) {
                    onProgress(Math.round((e.loaded / e.total) * 100));
                }
            });

            xhr.addEventListener('load', () => {
                if (xhr.status >= 200 && xhr.status < 300) {
                    try {
                        const data = JSON.parse(xhr.responseText);
                        // 更新 CSRF token（新格式：data.data.csrf_token）
                        if (data.code === 200 && data.data && data.data.csrf_token) {
                            csrfToken = data.data.csrf_token;
                        }
                        // 统一格式：{code: 200, message: "...", data: {...}}
                        if (data.code === 200) {
                            resolve(data);
                        } else {
                            reject(new Error(data.message || '上传失败'));
                        }
                    } catch (e) {
                        reject(new Error('响应解析失败'));
                    }
                } else if (xhr.status === 401) {
                    this._handleAuthError();
                    reject(new Error('认证失败'));
                } else {
                    reject(new Error(`HTTP ${xhr.status}`));
                }
            });

            xhr.addEventListener('error', () => reject(new Error('网络错误')));
            xhr.addEventListener('abort', () => reject(new Error('上传取消')));

            xhr.open('POST', this.baseURL + endpoint);
            xhr.withCredentials = true;
            // 获取最新的 CSRF token
            const token = this.getCSRFToken();
            if (token) {
                xhr.setRequestHeader('X-CSRF-Token', token);
            }
            xhr.send(formData);
        });
    }

    /**
     * 发送请求（核心方法）
     * @param {string} endpoint - API 端点
     * @param {Object} options - fetch 选项
     * @returns {Promise<Response>}
     */
    async request(endpoint, options = {}) {
        const config = {
            ...options,
            credentials: 'include',
            headers: {
                ...options.headers
            }
        };

        // 添加 CSRF token（非 GET 请求）
        if (config.method && config.method !== 'GET' && config.method !== 'HEAD' && csrfToken) {
            config.headers['X-CSRF-Token'] = csrfToken;
        }

        // 请求拦截器
        for (const interceptor of this.requestInterceptors) {
            try {
                const result = interceptor(config);
                if (result) {
                    config = result;
                }
            } catch (e) {
                console.error('Request interceptor error:', e);
            }
        }

        const url = this.baseURL + endpoint;

        try {
            const response = await fetch(url, config);

            // 自动提取 CSRF token
            if (response.ok) {
                const contentType = response.headers.get('content-type');
                if (contentType && contentType.includes('application/json')) {
                    // 克隆响应以便读取
                    const clonedResponse = response.clone();
                    try {
                        const data = await clonedResponse.json();
                        // 新格式：csrf_token 在 data 字段中
                        if (data.code === 200 && data.data && data.data.csrf_token) {
                            csrfToken = data.data.csrf_token;
                            console.log('CSRF token updated');
                        }
                    } catch (e) {
                        // JSON 解析失败，忽略
                    }
                }
            }

            // 响应拦截器
            for (const interceptor of this.responseInterceptors) {
                try {
                    const result = await interceptor(response);
                    if (result) {
                        response = result;
                    }
                } catch (e) {
                    console.error('Response interceptor error:', e);
                }
            }

            // 处理认证错误
            if (response.status === 401) {
                this._handleAuthError();
                return null;
            }

            // 处理 CSRF 错误
            if (response.status === 403) {
                try {
                    const data = await response.json();
                    if (data.error && data.error.includes('CSRF')) {
                        globalEvents.emit('auth:csrf_expired');
                        return null;
                    }
                } catch (e) {
                    // JSON 解析失败，继续处理
                }
            }

            return response;
        } catch (error) {
            console.error(`API request failed: ${endpoint}`, error);
            globalEvents.emit('api:error', { endpoint, error });
            throw error;
        }
    }

    /**
     * 处理认证错误
     * @private
     */
    _handleAuthError() {
        // 清除状态
        this.state.set('user', null);

        // 跳转到登录页
        window.location.href = 'login';
    }

    /**
     * 解析 JSON 响应（处理统一格式）
     * @param {Response} response - fetch 响应
     * @returns {Promise<Object>} 解析后的对象
     */
    async parseJSON(response) {
        if (!response) return null;
        const text = await response.text();
        if (!text) return null;
        try {
            const data = JSON.parse(text);
            return this._handleUnifiedResponse(data);
        } catch (e) {
            // JSON 解析失败，可能是 404 等错误响应
            // 返回错误信息而不是抛出异常
            console.warn('JSON parse error:', text.substring(0, 100));
            return { error: true, message: text, code: response.status || 500 };
        }
    }

    /**
     * 解析 JSON 响应（原始版本，不处理统一格式）
     * @param {Response} response - fetch 响应
     * @returns {Promise<Object>} 解析后的对象
     */
    async parseJSONRaw(response) {
        if (!response) return null;
        const text = await response.text();
        if (!text) return null;
        try {
            return JSON.parse(text);
        } catch (e) {
            console.error('JSON parse error:', text);
            return null;
        }
    }

    /**
     * 显示错误消息
     * @param {Error} error - 错误对象
     * @param {string} context - 错误上下文
     */
    showError(error, context = '') {
        const message = error?.message || String(error);
        console.error(`API Error${context ? ` (${context})` : ''}:`, message);
        globalEvents.emit('ui:toast', {
            type: 'error',
            message: message,
            duration: 3000
        });
    }
}

/**
 * 创建 API 实例
 * @param {StateManager} state - 状态管理器实例
 * @returns {API} API 实例
 */
export function createAPI(state) {
    return new API(state);
}

export { API_BASE, csrfToken };
