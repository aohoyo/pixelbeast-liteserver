/**
 * Keyboard - 键盘快捷键管理器
 *
 * 提供全局快捷键注册与管理：
 * - on/off 注册与取消快捷键
 * - 输入框聚焦时自动屏蔽
 * - 内置快捷键（Escape 关闭弹窗）
 */

// 不拦截快捷键的元素标签
const IGNORE_TAGS = new Set(['INPUT', 'TEXTAREA', 'SELECT']);

// 快捷键注册表 { normalizedKey: Set<{callback, raw}> }
const registry = new Map();

/**
 * 将用户输入的快捷键描述标准化
 * 支持格式：ctrl+s, delete, f2, escape, ctrl+a 等
 * @param {string} key - 快捷键描述
 * @returns {string} 标准化后的键名
 */
function normalizeKey(key) {
    return key.trim().toLowerCase()
        .replace(/\s+/g, '')
        .replace(/^ctrl\+/, 'ctrl+')
        .replace(/^alt\+/, 'alt+')
        .replace(/^shift\+/, 'shift+')
        .replace(/^meta\+/, 'meta+');
}

/**
 * 根据 KeyboardEvent 构造标准化键名
 * @param {KeyboardEvent} e
 * @returns {string}
 */
function eventToKey(e) {
    const parts = [];
    if (e.ctrlKey) parts.push('ctrl');
    if (e.altKey) parts.push('alt');
    if (e.shiftKey) parts.push('shift');
    if (e.metaKey) parts.push('meta');

    // 忽略单独的修饰键按下
    const ignorable = new Set(['Control', 'Alt', 'Shift', 'Meta']);
    if (ignorable.has(e.key)) return '';
    // e.key 可能为 undefined（如部分浏览器合成事件）
    if (!e.key) return '';

    // 键名转小写，统一格式
    let key = e.key.toLowerCase();
    // 统一特殊键名
    if (key === 'esc') key = 'escape';
    parts.push(key);

    return parts.join('+');
}

/**
 * 判断事件目标是否在可输入元素中
 * @param {EventTarget} target
 * @returns {boolean}
 */
function isEditable(target) {
    if (!target || !(target instanceof HTMLElement)) return false;
    const tag = target.tagName;
    if (IGNORE_TAGS.has(tag)) return true;
    // contentEditable 元素也屏蔽
    if (target.isContentEditable) return true;
    return false;
}

/**
 * 注册快捷键
 * @param {string} key - 快捷键描述，如 "ctrl+s", "escape", "f2"
 * @param {Function} callback - 触发回调
 * @returns {Function} 取消注册函数
 */
export function on(key, callback) {
    const normalized = normalizeKey(key);
    if (!registry.has(normalized)) {
        registry.set(normalized, new Set());
    }
    const entry = { callback, raw: key };
    registry.get(normalized).add(entry);

    // 返回取消注册函数
    return () => off(key, callback);
}

/**
 * 取消注册快捷键
 * @param {string} key - 快捷键描述
 * @param {Function} callback - 要移除的回调（可选，省略则移除该键所有回调）
 */
export function off(key, callback) {
    const normalized = normalizeKey(key);
    const entries = registry.get(normalized);
    if (!entries) return;

    if (!callback) {
        registry.delete(normalized);
        return;
    }

    for (const entry of entries) {
        if (entry.callback === callback) {
            entries.delete(entry);
            break;
        }
    }
    if (entries.size === 0) {
        registry.delete(normalized);
    }
}

// ---- 内置快捷键：Escape 关闭所有打开的 modal ----
on('escape', () => {
    document.querySelectorAll('.modal.active').forEach(modal => {
        // 优先触发关闭按钮
        const closeBtn = modal.querySelector('[data-close], .modal-close');
        if (closeBtn) {
            closeBtn.click();
        } else {
            modal.classList.remove('active');
        }
    });
});

// ---- 全局键盘事件监听 ----
document.addEventListener('keydown', (e) => {
    const key = eventToKey(e);
    if (!key) return;

    // 可输入元素聚焦时不触发（Escape 除外）
    if (isEditable(e.target) && key !== 'escape') return;

    const entries = registry.get(key);
    if (!entries || entries.size === 0) return;

    // 执行回调，阻止浏览器默认行为
    e.preventDefault();
    e.stopPropagation();

    for (const { callback } of entries) {
        try {
            callback(e);
        } catch (err) {
            console.error(`[Keyboard] 快捷键 "${key}" 回调异常:`, err);
        }
    }
});

export default { on, off };
