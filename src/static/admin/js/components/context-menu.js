/**
 * ContextMenu - 右键菜单组件
 * 
 * 单例模式，全局共享
 * 支持图标 + 文字 + 快捷键 + 分隔线
 */

class ContextMenu {
    constructor() {
        this.menu = null;
        this.visible = false;
        this.currentTarget = null;
        this.onHide = null;
    }

    /**
     * 初始化菜单（首次调用时创建 DOM）
     */
    init() {
        if (this.menu) return;
        
        this.menu = document.createElement('div');
        this.menu.className = 'context-menu';
        this.menu.style.display = 'none';
        document.body.appendChild(this.menu);

        // 点击外部关闭
        document.addEventListener('click', (e) => {
            if (this.visible && !this.menu.contains(e.target)) {
                this.hide();
            }
        });

        // ESC 关闭
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.visible) {
                this.hide();
            }
        });
    }

    /**
     * 显示菜单
     * @param {number} x - 鼠标 X 坐标
     * @param {number} y - 鼠标 Y 坐标
     * @param {Array} items - 菜单项配置
     * @param {Object} target - 关联的目标数据
     */
    show(x, y, items, target = null) {
        this.init();
        this.currentTarget = target;

        // 构建菜单内容
        let html = '';
        for (const item of items) {
            if (item.divider) {
                html += '<div class="context-menu-divider"></div>';
            } else if (item.section) {
                html += `<div class="context-menu-section">${item.section}</div>`;
            } else {
                const disabled = item.disabled ? 'disabled' : '';
                const icon = item.icon || '';
                const shortcut = item.shortcut ? `<span class="context-menu-shortcut">${item.shortcut}</span>` : '';
                
                html += `
                    <div class="context-menu-item ${disabled}" data-action="${item.action}">
                        <span class="context-menu-icon">${icon}</span>
                        <span class="context-menu-label">${item.label}</span>
                        ${shortcut}
                    </div>
                `;
            }
        }

        this.menu.innerHTML = html;

        // 绑定点击事件
        this.menu.querySelectorAll('.context-menu-item:not(.disabled)').forEach(el => {
            el.addEventListener('click', () => {
                const action = el.dataset.action;
                const item = items.find(i => i.action === action);
                if (item && item.onClick) {
                    item.onClick(target);
                }
                this.hide();
            });
        });

        // 定位菜单（边界检测）
        this.menu.style.display = 'block';
        const rect = this.menu.getBoundingClientRect();
        const viewportW = window.innerWidth;
        const viewportH = window.innerHeight;

        let posX = x;
        let posY = y;

        // 右边界检测
        if (x + rect.width > viewportW) {
            posX = viewportW - rect.width - 5;
        }

        // 下边界检测
        if (y + rect.height > viewportH) {
            posY = viewportH - rect.height - 5;
        }

        this.menu.style.left = posX + 'px';
        this.menu.style.top = posY + 'px';
        this.visible = true;
    }

    /**
     * 隐藏菜单
     */
    hide() {
        if (this.menu) {
            this.menu.style.display = 'none';
        }
        this.visible = false;
        this.currentTarget = null;
        if (this.onHide) {
            this.onHide();
        }
    }

    /**
     * 获取当前目标
     */
    getTarget() {
        return this.currentTarget;
    }
}

// 单例导出
export const contextMenu = new ContextMenu();