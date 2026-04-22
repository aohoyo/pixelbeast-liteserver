/**
 * DirPicker - 目录选择器工具
 *
 * 从 core/utils.js 中分离，避免对 UI 组件的循环依赖
 */

import { openFileBrowser } from '../components/file-browser/index.js';

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
