/**
 * Toast - 通知组件
 *
 * 简化可靠版，确保自动关闭功能正常工作
 */

// Toast 容器
let container = null;
let stylesInjected = false;

// 存储所有活动的 toast 定时器
const toastTimers = new WeakMap();

/**
 * 显示 Toast 通知
 * @param {string} message - 消息内容
 * @param {string} type - 类型 (success, error, warning, info)
 * @param {number} duration - 持续时间（毫秒），0 表示不自动关闭
 * @returns {Object} Toast 控制对象
 */
export function showToast(message, type = 'info', duration = 3000) {
    // 确保样式已注入
    if (!stylesInjected) {
        injectStyles();
        stylesInjected = true;
    }

    // 确保容器存在
    if (!container) {
        container = document.createElement('div');
        container.id = 'toast-container';
        document.body.appendChild(container);
    }

    // 创建 Toast 元素
    const toast = document.createElement('div');
    toast.className = 'toast';

    // 图标和颜色映射
    const config = {
        success: { icon: '✓', color: '#22c55e' },
        error: { icon: '✕', color: '#ef4444' },
        warning: { icon: '⚠', color: '#fbbf24' },
        info: { icon: 'ℹ', color: '#f97316' }
    }[type] || config.info;

    toast.innerHTML = `
        <span class="toast-icon" style="color: ${config.color}">${config.icon}</span>
        <span class="toast-message">${String(message).replace(/</g, '&lt;').replace(/>/g, '&gt;')}</span>
        <button class="toast-close" type="button">×</button>
    `;

    toast.dataset.toastType = type;
    toast.style.borderLeftColor = config.color;

    // 添加到容器
    container.appendChild(toast);

    // 下一帧显示
    requestAnimationFrame(() => {
        requestAnimationFrame(() => {
            toast.classList.add('show');
        });
    });

    // 自动关闭
    let timer = null;
    if (duration > 0) {
        timer = setTimeout(() => {
            closeToast(toast);
        }, duration);
        toastTimers.set(toast, timer);
    }

    // 关闭按钮
    const closeBtn = toast.querySelector('.toast-close');
    if (closeBtn) {
        closeBtn.addEventListener('click', () => {
            closeToast(toast);
        });
    }

    // 鼠标悬停暂停
    let paused = false;
    toast.addEventListener('mouseenter', () => {
        if (timer) {
            clearTimeout(timer);
            paused = true;
        }
    });

    toast.addEventListener('mouseleave', () => {
        if (paused && duration > 0) {
            timer = setTimeout(() => {
                closeToast(toast);
            }, 500);
            toastTimers.set(toast, timer);
            paused = false;
        }
    });

    return {
        close: () => closeToast(toast),
        element: toast
    };
}

/**
 * 关闭 Toast
 */
function closeToast(toast) {
    const timer = toastTimers.get(toast);
    if (timer) {
        clearTimeout(timer);
        toastTimers.delete(toast);
    }

    toast.classList.remove('show');
    toast.classList.add('hide');

    setTimeout(() => {
        if (toast.parentElement) {
            toast.parentElement.removeChild(toast);
        }
    }, 300);
}

/**
 * 注入样式
 */
function injectStyles() {
    const style = document.createElement('style');
    style.textContent = `
        #toast-container {
            position: fixed !important;
            z-index: 99999 !important;
            display: flex !important;
            flex-direction: column !important;
            gap: 10px !important;
            pointer-events: none !important;
            bottom: 20px !important;
            right: 20px !important;
            max-width: 380px !important;
        }

        .toast {
            display: flex !important;
            align-items: center !important;
            gap: 12px !important;
            min-width: 280px !important;
            padding: 14px 16px !important;
            background: rgba(30, 30, 30, 0.96) !important;
            border-radius: 10px !important;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5) !important;
            pointer-events: auto !important;
            opacity: 0 !important;
            transform: translateX(100%) !important;
            transition: opacity 0.3s ease, transform 0.3s ease !important;
            border-left: 4px solid !important;
            font: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif !important;
            font-size: 14px !important;
            color: #fafafa !important;
        }

        .toast.show {
            opacity: 1 !important;
            transform: translateX(0) !important;
        }

        .toast.hide {
            opacity: 0 !important;
            transform: translateX(100%) !important;
        }

        .toast-icon {
            flex-shrink: 0 !important;
            font-size: 18px !important;
            width: 24px !important;
            height: 24px !important;
            display: flex !important;
            align-items: center !important;
            justify-content: center !important;
            border-radius: 50% !important;
            background: rgba(255, 255, 255, 0.08) !important;
        }

        .toast-message {
            flex: 1 !important;
            line-height: 1.4 !important;
            word-break: break-word !important;
        }

        .toast-close {
            flex-shrink: 0 !important;
            background: none !important;
            border: none !important;
            color: #78716c !important;
            font-size: 20px !important;
            line-height: 1 !important;
            cursor: pointer !important;
            padding: 0 !important;
            width: 24px !important;
            height: 24px !important;
            display: flex !important;
            align-items: center !important;
            justify-content: center !important;
            opacity: 0.6 !important;
            transition: opacity 0.2s !important;
            border-radius: 4px !important;
        }

        .toast-close:hover {
            opacity: 1 !important;
            background: rgba(255, 255, 255, 0.12) !important;
        }

        @media (max-width: 480px) {
            #toast-container {
                left: 10px !important;
                right: 10px !important;
                bottom: 10px !important;
            }
            .toast {
                min-width: auto !important;
            }
        }
    `;

    document.head.appendChild(style);
}

// 便捷函数
export function showSuccess(message, duration) {
    return showToast(message, 'success', duration);
}

export function showError(message, duration) {
    return showToast(message, 'error', duration);
}

export function showWarning(message, duration) {
    return showToast(message, 'warning', duration);
}

// 导出
export const toast = {
    show: showToast,
    success: showSuccess,
    error: showError,
    warning: showWarning,
    info: showToast
};

export default toast;