/**
 * Template - 安全 HTML 模板引擎
 *
 * 提供标签模板函数，自动对插值进行 XSS 转义，
 * 支持通过 raw() 标记原始 HTML（不转义）。
 */

import { escapeHtml } from './utils.js';

// 重新导出，方便其他模块直接从 template.js 导入
export { escapeHtml };

// 原始 HTML 标记符号
const RAW_SYMBOL = Symbol('raw');

/**
 * 标记字符串为原始 HTML（不转义）
 * @param {string} str - 原始 HTML 字符串
 * @returns {Object} 带标记的对象
 */
export function raw(str) {
    return { [RAW_SYMBOL]: true, value: String(str ?? '') };
}

/**
 * 检查值是否为原始 HTML 标记
 * @param {*} value - 要检查的值
 * @returns {boolean}
 */
function isRaw(value) {
    return value !== null && typeof value === 'object' && value[RAW_SYMBOL] === true;
}

/**
 * 安全 HTML 标签模板函数
 *
 * 自动对插值进行 XSS 转义，除非用 raw() 包装。
 * 数组插值会逐项处理并拼接。
 *
 * @example
 * html`<div>${userName}</div>`           // 自动转义
 * html`<div>${raw(htmlContent)}</div>`   // 不转义
 * html`<ul>${items.map(i => html`<li>${i}</li>`)}</ul>`  // 数组支持
 *
 * @param {string[]} strings - 模板字符串的静态部分
 * @param {...*} values - 插值
 * @returns {string} 安全的 HTML 字符串
 */
export function html(strings, ...values) {
    let result = strings[0];
    for (let i = 0; i < values.length; i++) {
        const value = values[i];
        if (isRaw(value)) {
            // 原始 HTML，不转义
            result += value.value;
        } else if (Array.isArray(value)) {
            // 数组元素逐项处理
            result += value.map(v =>
                isRaw(v) ? v.value : escapeHtml(v)
            ).join('');
        } else {
            result += escapeHtml(value);
        }
        result += strings[i + 1];
    }
    return result;
}

export default { html, escapeHtml, raw };
