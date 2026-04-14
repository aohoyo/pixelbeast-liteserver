/**
 * 全局错误边界 - 捕获未处理的 JS 错误和 Promise rejection
 *
 * 尽早注册 window.onerror / unhandledrejection，
 * 错误发生时尝试用 toast 提示用户，不可用时降级为 alert。
 */

/**
 * 向用户显示错误提示
 * 优先使用 toast 组件（通过 window.app.toast），不可用时降级为 alert
 */
function notifyError(message) {
    try {
        const t = window.app?.toast;
        if (t && typeof t.error === 'function') {
            t.error(message, 5000);
            return;
        }
    } catch (_) {
        // toast 不可用，降级
    }
    alert('[错误] ' + message);
}

/**
 * 格式化错误信息，截断过长内容
 */
function formatError(message, source, lineno, colno) {
    let result = String(message || '未知错误');
    // 附加位置信息
    const loc = [];
    if (source) loc.push(source.replace(/^.*\/([^/]+)$/, '$1'));
    if (lineno) loc.push(lineno);
    if (colno) loc.push(colno);
    if (loc.length) result += ` (${loc.join(':')})`;
    // 截断
    if (result.length > 200) result = result.slice(0, 200) + '…';
    return result;
}

/**
 * 格式化 Promise rejection 原因
 */
function formatReason(reason) {
    if (reason instanceof Error) {
        let msg = reason.message || reason.toString();
        if (msg.length > 200) msg = msg.slice(0, 200) + '…';
        return msg;
    }
    const s = String(reason || '未知 Promise 异常');
    return s.length > 200 ? s.slice(0, 200) + '…' : s;
}

// 注册 window.onerror — 捕获同步运行时错误
window.onerror = function (message, source, lineno, colno, error) {
    const formatted = formatError(message, source, lineno, colno);
    console.error('[全局错误]', error || message, source, lineno, colno);
    notifyError(formatted);
};

// 注册 unhandledrejection — 捕获未处理的 Promise rejection
window.addEventListener('unhandledrejection', function (event) {
    const reason = event.reason;
    const formatted = formatReason(reason);
    console.error('[未处理 Promise]', reason);
    notifyError(formatted);
});
