/**
 * Dialog 对话框组件
 * 纯 JS 实现，支持淡入淡出动画
 *
 * 支持两种调用方式：
 * - show('dialog-id') — 显示 HTML 中已有的 .dialog 元素
 * - show({ title, message, ... }) — 动态创建对话框
 */

import { escapeHtml } from '../core/utils.js';

class Dialog {
    constructor() {
        this.currentDialog = null;

        // 全局事件委托：处理 data-close-dialog 按钮
        document.addEventListener('click', (e) => {
            if (e.target.closest('[data-close-dialog]')) {
                this.close();
            }
        });

        // ESC 键关闭
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.currentDialog) {
                this.close();
            }
        });
    }

    /**
     * 显示对话框
     * @param {string|Object} options - 字符串 ID 或配置选项
     * @param {string} options.title - 标题
     * @param {string} options.message - 内容
     * @param {string} options.type - 类型: 'info' | 'warning' | 'danger'
     * @param {string} options.confirmText - 确认按钮文字
     * @param {string} options.cancelText - 取消按钮文字
     * @param {Function} options.onConfirm - 确认回调
     * @param {Function} options.onCancel - 取消回调
     */
    show(options = {}) {
        // 支持传入字符串 ID，显示已有的 HTML 弹窗
        if (typeof options === 'string') {
            const el = document.getElementById(options);
            if (el) {
                // 关闭之前的弹窗
                if (this.currentDialog) {
                    this.close();
                }
                this.currentDialog = el;
                el.offsetHeight; // 触发重排
                el.classList.add('active');
            }
            return;
        }

        const {
            title = '提示',
            message = '',
            type = 'info',
            confirmText = '确定',
            cancelText = '取消',
            onConfirm = null,
            onCancel = null
        } = options;

        // 如果已有对话框，先关闭
        if (this.currentDialog) {
            this.close();
        }

        // 根据类型选择图标
        const iconMap = {
            warning: 'icon-tip',
            danger: 'icon-trash',
            info: 'icon-tip'
        };
        const iconClass = iconMap[type] || 'icon-tip';

        // 创建对话框 DOM
        const dialog = document.createElement('div');
        dialog.className = `dialog dialog--dynamic dialog--${type}`;
        dialog.innerHTML = `
            <div class="dialog-content">
                <div class="dialog-header">
                    <i class="icon ${iconClass}"></i>
                    <h3>${escapeHtml(title)}</h3>
                </div>
                <div class="dialog-body">${message}</div>
                <div class="dialog-footer">
                    <button class="btn btn-secondary dialog-cancel">${escapeHtml(cancelText)}</button>
                    <button class="btn ${type === 'danger' ? 'btn-danger' : 'btn-primary'} dialog-confirm">${escapeHtml(confirmText)}</button>
                </div>
            </div>
        `;

        // 添加到页面
        document.body.appendChild(dialog);
        this.currentDialog = dialog;

        // 绑定事件
        const confirmBtn = dialog.querySelector('.dialog-confirm');
        const cancelBtn = dialog.querySelector('.dialog-cancel');

        confirmBtn.addEventListener('click', () => {
            if (onConfirm) onConfirm();
            this.close();
        });

        cancelBtn.addEventListener('click', () => {
            if (onCancel) onCancel();
            this.close();
        });

        // 点击遮罩层关闭
        dialog.addEventListener('click', (e) => {
            if (e.target === dialog) {
                if (onCancel) onCancel();
                this.close();
            }
        });

        // 触发重排以启动动画
        dialog.offsetHeight;
        dialog.classList.add('active');
    }

    /**
     * 关闭对话框
     */
    close() {
        if (!this.currentDialog) return;

        const dialog = this.currentDialog;
        dialog.classList.remove('active');

        if (dialog.classList.contains('dialog--dynamic')) {
            // 动态创建的弹窗：动画结束后移除 DOM
            setTimeout(() => {
                if (dialog.parentNode) {
                    dialog.parentNode.removeChild(dialog);
                }
            }, 200);
        }

        this.currentDialog = null;
    }

    /**
     * 快捷方法：确认对话框（message 会被自动转义）
     */
    confirm(message, onConfirm, onCancel) {
        this.show({
            title: '确认操作',
            message: escapeHtml(message),
            type: 'warning',
            confirmText: '确定',
            cancelText: '取消',
            onConfirm,
            onCancel
        });
    }

    /**
     * 快捷方法：警告对话框（message 会被自动转义）
     */
    alert(message, titleOrOnConfirm, onConfirm) {
        // 支持两种调用方式：alert(message, onConfirm) 或 alert(message, title, onConfirm)
        const hasTitle = typeof titleOrOnConfirm === 'string';
        this.show({
            title: hasTitle ? titleOrOnConfirm : '提示',
            message: escapeHtml(message),
            type: 'info',
            confirmText: '知道了',
            cancelText: '',
            onConfirm: hasTitle ? onConfirm : titleOrOnConfirm
        });
    }

    /**
     * 快捷方法：危险操作确认（message 会被自动转义）
     */
    danger(message, onConfirm, onCancel) {
        this.show({
            title: '危险操作',
            message: escapeHtml(message),
            type: 'danger',
            confirmText: '确认删除',
            cancelText: '取消',
            onConfirm,
            onCancel
        });
    }
}

// 创建单例
const dialog = new Dialog();

export default dialog;
