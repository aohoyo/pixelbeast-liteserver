/**
 * Utils - 通用工具函数
 *
 * 提供项目中常用的工具函数，避免重复代码
 */

import { openFileBrowser } from '../components/file-browser/index.js';

/**
 * HTML 转义，防止 XSS
 * @param {string} text - 要转义的文本
 * @returns {string} 转义后的文本
 */
export function escapeHtml(text) {
    if (text === null || text === undefined) return '';
    const str = String(text);
    const escapeMap = {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#039;'
    };
    return str.replace(/[&<>"']/g, char => escapeMap[char]);
}

/**
 * 格式化文件大小
 * @param {number} bytes - 字节数
 * @param {number} decimals - 小数位数
 * @returns {string} 格式化后的字符串
 */
export function formatSize(bytes, decimals = 2) {
    if (bytes === 0) return '0 B';
    if (bytes === null || bytes === undefined) return '--';
    
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(decimals)) + ' ' + sizes[i];
}

/**
 * 格式化存储大小（GB 为单位）
 * @param {number} gb - GB 数
 * @returns {string} 格式化后的字符串
 */
export function formatStorage(gb) {
    if (gb === null || gb === undefined) return '--';
    if (gb >= 1000) return `${(gb / 1000).toFixed(1)}TB`;
    if (gb < 1) return `${(gb * 1024).toFixed(0)}MB`;
    return `${gb.toFixed(1)}GB`;
}

/**
 * 格式化运行时间
 * @param {number} ms - 毫秒数
 * @returns {string} 格式化后的字符串
 */
export function formatUptime(ms) {
    if (typeof ms !== 'number' || isNaN(ms)) return '--';
    
    const seconds = Math.floor(ms / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);
    
    if (days > 0) return `${days}天${hours % 24}时${minutes % 60}分`;
    if (hours > 0) return `${hours}时${minutes % 60}分`;
    if (minutes > 0) return `${minutes}分${seconds % 60}秒`;
    return `${seconds}秒`;
}

/**
 * 格式化日期时间
 * @param {Date|string|number} date - 日期
 * @param {string} format - 格式 'full' | 'date' | 'time' | 'datetime'
 * @returns {string} 格式化后的字符串
 */
export function formatDate(date, format = 'datetime') {
    if (!date) return '--';
    
    const d = date instanceof Date ? date : new Date(date);
    if (isNaN(d.getTime())) return '--';
    
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    const hours = String(d.getHours()).padStart(2, '0');
    const minutes = String(d.getMinutes()).padStart(2, '0');
    const seconds = String(d.getSeconds()).padStart(2, '0');
    
    switch (format) {
        case 'date':
            return `${year}-${month}-${day}`;
        case 'time':
            return `${hours}:${minutes}:${seconds}`;
        case 'full':
            return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
        default:
            return `${year}-${month}-${day} ${hours}:${minutes}`;
    }
}

/**
 * 防抖函数
 * @param {Function} fn - 要防抖的函数
 * @param {number} delay - 延迟时间（毫秒）
 * @returns {Function} 防抖后的函数
 */
export function debounce(fn, delay = 300) {
    let timer = null;
    return function (...args) {
        if (timer) clearTimeout(timer);
        timer = setTimeout(() => fn.apply(this, args), delay);
    };
}

/**
 * 节流函数
 * @param {Function} fn - 要节流的函数
 * @param {number} interval - 间隔时间（毫秒）
 * @returns {Function} 节流后的函数
 */
export function throttle(fn, interval = 100) {
    let lastTime = 0;
    return function (...args) {
        const now = Date.now();
        if (now - lastTime >= interval) {
            lastTime = now;
            fn.apply(this, args);
        }
    };
}

/**
 * 数字动画
 * @param {HTMLElement} element - 目标元素
 * @param {number} target - 目标值
 * @param {string} suffix - 后缀
 * @param {number} duration - 动画时长（毫秒）
 */
export function animateValue(element, target, suffix = '', duration = 600) {
    if (!element) return;
    
    const start = parseFloat(element.textContent) || 0;
    const startTime = performance.now();
    
    function update(currentTime) {
        const elapsed = currentTime - startTime;
        const progress = Math.min(elapsed / duration, 1);
        const easeProgress = 1 - Math.pow(1 - progress, 3);
        const current = start + (target - start) * easeProgress;
        
        element.textContent = (target % 1 === 0 ? Math.floor(current) : current.toFixed(1)) + suffix;
        
        if (progress < 1) {
            requestAnimationFrame(update);
        } else {
            element.textContent = (target % 1 === 0 ? target : target.toFixed(1)) + suffix;
        }
    }
    
    requestAnimationFrame(update);
}

/**
 * 复制文本到剪贴板
 * @param {string} text - 要复制的文本
 * @returns {Promise<boolean>} 是否成功
 */
export async function copyToClipboard(text) {
    try {
        await navigator.clipboard.writeText(text);
        return true;
    } catch (err) {
        // 降级方案
        const textarea = document.createElement('textarea');
        textarea.value = text;
        textarea.style.position = 'fixed';
        textarea.style.opacity = '0';
        document.body.appendChild(textarea);
        textarea.select();
        try {
            document.execCommand('copy');
            return true;
        } catch (e) {
            return false;
        } finally {
            document.body.removeChild(textarea);
        }
    }
}

/**
 * 生成唯一 ID
 * @param {string} prefix - 前缀
 * @returns {string} 唯一 ID
 */
export function generateId(prefix = 'id') {
    return `${prefix}_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
}

/**
 * 深拷贝
 * @param {*} obj - 要拷贝的对象
 * @returns {*} 拷贝后的对象
 */
export function deepClone(obj) {
    if (obj === null || typeof obj !== 'object') return obj;
    if (obj instanceof Date) return new Date(obj);
    if (obj instanceof Array) return obj.map(item => deepClone(item));
    if (obj instanceof Object) {
        const copy = {};
        for (const key in obj) {
            if (obj.hasOwnProperty(key)) {
                copy[key] = deepClone(obj[key]);
            }
        }
        return copy;
    }
    return obj;
}

/**
 * 判断是否为空
 * @param {*} value - 要判断的值
 * @returns {boolean} 是否为空
 */
export function isEmpty(value) {
    if (value === null || value === undefined) return true;
    if (typeof value === 'string') return value.trim() === '';
    if (Array.isArray(value)) return value.length === 0;
    if (typeof value === 'object') return Object.keys(value).length === 0;
    return false;
}

/**
 * 安全获取嵌套对象属性
 * @param {Object} obj - 对象
 * @param {string} path - 路径（点分隔）
 * @param {*} defaultValue - 默认值
 * @returns {*} 属性值
 */
export function get(obj, path, defaultValue = undefined) {
    const keys = path.split('.');
    let result = obj;
    for (const key of keys) {
        result = result?.[key];
        if (result === undefined) return defaultValue;
    }
    return result;
}

/**
 * 目录选择器（调用文件浏览器选择文件夹）
 * @param {string} inputId - 目标 input 元素 ID
 * @param {Object} api - API 实例
 */
export async function openDirPicker(inputId, api) {
    const input = document.querySelector(`#${inputId}`);
    if (!input) return;
    try {
        const selected = await openFileBrowser({
            title: '选择目录',
            selectMode: 'folder',
            root: input.value || '.',
            api,
        });
        if (selected) {
            input.value = selected;
            input.dispatchEvent(new Event('change', { bubbles: true }));
        }
    } catch (e) {
        // 用户取消
    }
}

/**
 * 初始化数字输入组件（上下按钮和键盘快捷键）
 * @param {HTMLElement} root - 容器元素
 */
export function initNumberInputs(root = document) {
    root.querySelectorAll('.number-input-wrapper').forEach(wrapper => {
        const input = wrapper.querySelector('input[type="number"]');
        const upBtn = wrapper.querySelector('[data-action="up"]');
        const downBtn = wrapper.querySelector('[data-action="down"]');

        if (!input) return;

        const step = parseInt(input.dataset.step) || 1;
        const min = input.min !== '' ? parseInt(input.min) : null;
        const max = input.max !== '' ? parseInt(input.max) : null;

        const updateValue = (delta) => {
            let value = parseInt(input.value) || 0;
            value += delta;
            if (min !== null && value < min) value = min;
            if (max !== null && value > max) value = max;
            input.value = value;
            input.dispatchEvent(new Event('input', { bubbles: true }));
        };

        upBtn?.addEventListener('click', () => updateValue(step));
        downBtn?.addEventListener('click', () => updateValue(-step));

        input.addEventListener('keydown', (e) => {
            if (e.key === 'ArrowUp') {
                e.preventDefault();
                updateValue(step);
            } else if (e.key === 'ArrowDown') {
                e.preventDefault();
                updateValue(-step);
            }
        });
    });
}

// 默认导出所有工具
export default {
    escapeHtml,
    formatSize,
    formatStorage,
    formatUptime,
    formatDate,
    debounce,
    throttle,
    animateValue,
    copyToClipboard,
    generateId,
    deepClone,
    isEmpty,
    get,
    initNumberInputs,
    openDirPicker
};