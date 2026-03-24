/**
 * Message - 消息提示组件
 *
 * 参考 Element UI Message 组件设计
 * 特点：顶部居中显示，从上往下淡入，支持多个堆叠
 */

// Message 容器
let container = null;
let stylesInjected = false;
let messageId = 0;

// 存储所有活动的 message 定时器
const messageTimers = new WeakMap();

/**
 * 显示 Message 消息
 * @param {string|Object} options - 消息内容或配置对象
 * @param {string} options.message - 消息内容
 * @param {string} options.type - 类型 (success, warning, info, error)
 * @param {number} options.duration - 持续时间（毫秒），0 表示不自动关闭
 * @param {boolean} options.showClose - 是否显示关闭按钮
 * @param {Function} options.onClose - 关闭时的回调
 * @returns {Object} Message 控制对象
 */
export function showMessage(options) {
    // 支持字符串参数
    if (typeof options === 'string') {
        options = { message: options };
    }

    const {
        message = '',
        type = 'info',
        duration = 3000,
        showClose = false,
        onClose = null
    } = options;

    // 确保样式已注入
    if (!stylesInjected) {
        injectStyles();
        stylesInjected = true;
    }

    // 确保容器存在
    if (!container) {
        container = document.createElement('div');
        container.id = 'message-container';
        document.body.appendChild(container);
    }

    // 创建 Message 元素
    const msg = document.createElement('div');
    msg.className = `message message-${type}`;
    msg.dataset.messageId = ++messageId;

    // 图标映射
    const iconMap = {
        success: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/></svg>',
        warning: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/></svg>',
        info: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/></svg>',
        error: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>'
    };

    // 构建内容
    let content = '<div class="message-content">';
    content += `<span class="message-icon">${iconMap[type] || iconMap.info}</span>`;
    content += `<span class="message-text">${String(message).replace(/</g, '&lt;').replace(/>/g, '&gt;')}</span>`;
    if (showClose) {
        content += '<button class="message-close" type="button"><svg viewBox="0 0 24 24" fill="currentColor"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg></button>';
    }
    content += '</div>';
    msg.innerHTML = content;

    // 存储回调
    if (onClose) {
        msg._onClose = onClose;
    }

    // 添加到容器
    container.appendChild(msg);

    // 下一帧显示动画
    requestAnimationFrame(() => {
        requestAnimationFrame(() => {
            msg.classList.add('message-visible');
        });
    });

    // 自动关闭
    let timer = null;
    if (duration > 0) {
        timer = setTimeout(() => {
            closeMessage(msg);
        }, duration);
        messageTimers.set(msg, timer);
    }

    // 关闭按钮事件
    if (showClose) {
        const closeBtn = msg.querySelector('.message-close');
        if (closeBtn) {
            closeBtn.addEventListener('click', () => {
                closeMessage(msg);
            });
        }
    }

    // 鼠标悬停暂停
    let paused = false;
    let remainingTime = duration;

    msg.addEventListener('mouseenter', () => {
        if (timer) {
            remainingTime = duration - (Date.now() - msg._startTime);
            clearTimeout(timer);
            paused = true;
        }
    });

    msg.addEventListener('mouseleave', () => {
        if (paused && remainingTime > 0) {
            timer = setTimeout(() => {
                closeMessage(msg);
            }, remainingTime);
            messageTimers.set(msg, timer);
            paused = false;
        }
    });

    // 记录开始时间
    msg._startTime = Date.now();

    return {
        close: () => closeMessage(msg),
        element: msg
    };
}

/**
 * 关闭 Message
 */
function closeMessage(msg) {
    if (!msg || !msg.parentElement) return;

    const timer = messageTimers.get(msg);
    if (timer) {
        clearTimeout(timer);
        messageTimers.delete(msg);
    }

    msg.classList.remove('message-visible');
    msg.classList.add('message-closing');

    setTimeout(() => {
        if (msg.parentElement) {
            msg.parentElement.removeChild(msg);
        }
        // 触发回调
        if (msg._onClose) {
            msg._onClose(msg);
        }
    }, 300);
}

/**
 * 关闭所有 Message
 */
export function closeAllMessages() {
    if (!container) return;
    const messages = container.querySelectorAll('.message');
    messages.forEach(msg => closeMessage(msg));
}

/**
 * 注入样式
 */
function injectStyles() {
    const style = document.createElement('style');
    style.textContent = `
        #message-container {
            position: fixed !important;
            top: 20px !important;
            left: 0 !important;
            right: 0 !important;
            z-index: 100000 !important;
            display: flex !important;
            flex-direction: column !important;
            align-items: center !important;
            pointer-events: none !important;
        }

        .message {
            display: flex !important;
            align-items: center !important;
            min-width: 300px !important;
            max-width: 480px !important;
            margin-bottom: 12px !important;
            padding: 12px 16px !important;
            background: var(--bg-elevated, #1c1917) !important;
            border-radius: var(--radius, 8px) !important;
            box-shadow: var(--shadow-lg, 0 25px 50px -12px rgba(0, 0, 0, 0.6)) !important;
            pointer-events: auto !important;
            opacity: 0 !important;
            transform: translateY(-100%) !important;
            transition: opacity 0.3s ease, transform 0.3s ease !important;
            font-family: var(--font-sans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif) !important;
            font-size: 14px !important;
            color: var(--text, #fafaf9) !important;
            border: 1px solid var(--border, #44403c) !important;
        }

        .message-visible {
            opacity: 1 !important;
            transform: translateY(0) !important;
        }

        .message-closing {
            opacity: 0 !important;
            transform: translateY(-20px) !important;
        }

        .message-content {
            display: flex !important;
            align-items: center !important;
            width: 100% !important;
            gap: 10px !important;
        }

        .message-icon {
            flex-shrink: 0 !important;
            width: 18px !important;
            height: 18px !important;
            display: flex !important;
            align-items: center !important;
            justify-content: center !important;
        }

        .message-icon svg {
            width: 100% !important;
            height: 100% !important;
        }

        /* 类型样式 */
        .message-success .message-icon {
            color: var(--success, #22c55e) !important;
        }

        .message-warning .message-icon {
            color: var(--warning, #fbbf24) !important;
        }

        .message-info .message-icon {
            color: var(--info, #3b82f6) !important;
        }

        .message-error .message-icon {
            color: var(--danger, #ef4444) !important;
        }

        .message-text {
            flex: 1 !important;
            line-height: 1.5 !important;
            word-break: break-word !important;
        }

        .message-close {
            flex-shrink: 0 !important;
            background: none !important;
            border: none !important;
            color: var(--text-muted, #78716c) !important;
            cursor: pointer !important;
            padding: 4px !important;
            margin: -4px -4px -4px 8px !important;
            width: 24px !important;
            height: 24px !important;
            display: flex !important;
            align-items: center !important;
            justify-content: center !important;
            opacity: 0.6 !important;
            transition: opacity 0.2s, color 0.2s !important;
            border-radius: var(--radius-sm, 4px) !important;
        }

        .message-close svg {
            width: 16px !important;
            height: 16px !important;
        }

        .message-close:hover {
            opacity: 1 !important;
            color: var(--text, #fafaf9) !important;
            background: var(--bg-hover, #272524) !important;
        }

        /* 响应式 */
        @media (max-width: 480px) {
            .message {
                min-width: auto !important;
                max-width: calc(100vw - 40px) !important;
                margin-left: 20px !important;
                margin-right: 20px !important;
            }
        }
    `;

    document.head.appendChild(style);
}

// 便捷方法
export function showSuccessMessage(message, duration) {
    return showMessage({ message, type: 'success', duration });
}

export function showWarningMessage(message, duration) {
    return showMessage({ message, type: 'warning', duration });
}

export function showInfoMessage(message, duration) {
    return showMessage({ message, type: 'info', duration });
}

export function showErrorMessage(message, duration) {
    return showMessage({ message, type: 'error', duration });
}

// 导出
export const message = {
    show: showMessage,
    success: showSuccessMessage,
    warning: showWarningMessage,
    info: showInfoMessage,
    error: showErrorMessage,
    closeAll: closeAllMessages
};

export default message;