/**
 * Tooltip 工具提示组件
 * Element UI 风格 - Hover 触发、自定义内容、智能定位
 */

class Tooltip {
    constructor() {
        this.currentTooltip = null;
        this.showDelay = 300;
        this.hideDelay = 100;
        this.showTimer = null;
        this.hideTimer = null;
        this.visible = false;
    }

    /**
     * 初始化页面中的 tooltip
     */
    init() {
        document.addEventListener('mouseover', this.handleMouseOver.bind(this));
        document.addEventListener('mouseout', this.handleMouseOut.bind(this));
    }

    /**
     * 处理鼠标悬停
     */
    handleMouseOver(e) {
        const target = e.target.closest('[data-tooltip]');
        if (!target) return;

        const content = target.dataset.tooltip;
        if (!content) return;

        // 清除隐藏计时器
        if (this.hideTimer) {
            clearTimeout(this.hideTimer);
            this.hideTimer = null;
        }

        // 延迟显示
        this.showTimer = setTimeout(() => {
            this.show(target, content);
        }, this.showDelay);
    }

    /**
     * 处理鼠标移出
     */
    handleMouseOut(e) {
        // 清除显示计时器
        if (this.showTimer) {
            clearTimeout(this.showTimer);
            this.showTimer = null;
        }

        // 延迟隐藏
        if (this.visible) {
            this.hideTimer = setTimeout(() => {
                this.hide();
            }, this.hideDelay);
        }
    }

    /**
     * 显示 tooltip
     */
    show(target, content) {
        // 如果已有 tooltip 且是同一个元素，直接返回
        if (this.currentTooltip && this.currentTarget === target) {
            return;
        }

        // 如果已有 tooltip，先隐藏
        if (this.currentTooltip) {
            this.currentTooltip.remove();
        }

        this.currentTarget = target;

        // 创建 tooltip 元素
        const tooltip = document.createElement('div');
        tooltip.className = 'el-tooltip';
        tooltip.textContent = content;
        document.body.appendChild(tooltip);

        this.currentTooltip = tooltip;

        // 计算位置
        this.position(tooltip, target);

        // 显示
        this.visible = true;
        tooltip.classList.add('el-tooltip--visible');
    }

    /**
     * 隐藏 tooltip
     */
    hide() {
        if (!this.currentTooltip) return;

        const tooltip = this.currentTooltip;
        tooltip.classList.remove('el-tooltip--visible');
        this.visible = false;

        // 动画结束后移除
        setTimeout(() => {
            if (tooltip.parentNode) {
                tooltip.parentNode.removeChild(tooltip);
            }
        }, 150);

        this.currentTooltip = null;
        this.currentTarget = null;
    }

    /**
     * 计算并设置位置（智能定位）
     */
    position(tooltip, target) {
        const targetRect = target.getBoundingClientRect();
        const tooltipRect = tooltip.getBoundingClientRect();
        const viewportWidth = window.innerWidth;
        const viewportHeight = window.innerHeight;
        const gap = 12;

        // 计算各个方向的位置和箭头位置
        const positions = {
            top: {
                top: targetRect.top - tooltipRect.height - gap,
                left: targetRect.left + (targetRect.width - tooltipRect.width) / 2,
                arrow: targetRect.left + targetRect.width / 2
            },
            bottom: {
                top: targetRect.bottom + gap,
                left: targetRect.left + (targetRect.width - tooltipRect.width) / 2,
                arrow: targetRect.left + targetRect.width / 2
            },
            left: {
                top: targetRect.top + (targetRect.height - tooltipRect.height) / 2,
                left: targetRect.left - tooltipRect.width - gap,
                arrow: targetRect.top + targetRect.height / 2
            },
            right: {
                top: targetRect.top + (targetRect.height - tooltipRect.height) / 2,
                left: targetRect.right + gap,
                arrow: targetRect.top + targetRect.height / 2
            }
        };

        // 优先顺序：上 > 下 > 右 > 左
        const preferredOrder = ['top', 'bottom', 'right', 'left'];
        let bestPosition = 'top';
        let bestScore = -1;

        for (const [name, pos] of Object.entries(positions)) {
            let score = 0;

            // 检查是否在视口内（留出 10px 边距）
            const margin = 10;
            const fitsHorizontally = pos.left >= margin && pos.left + tooltipRect.width <= viewportWidth - margin;
            const fitsVertically = pos.top >= margin && pos.top + tooltipRect.height <= viewportHeight - margin;

            if (fitsVertically && fitsHorizontally) {
                score += 100;
            } else if (fitsVertically) {
                score += 50;
            } else if (fitsHorizontally) {
                score += 25;
            }

            // 方向优先级
            if (name === 'top') score += 4;
            if (name === 'bottom') score += 3;
            if (name === 'right') score += 2;
            if (name === 'left') score += 1;

            if (score > bestScore) {
                bestScore = score;
                bestPosition = name;
            }
        }

        // 应用位置
        const pos = positions[bestPosition];
        tooltip.style.top = `${pos.top}px`;
        tooltip.style.left = `${pos.left}px`;

        // 设置方向类和箭头位置
        tooltip.className = 'el-tooltip';
        tooltip.classList.add(`el-tooltip--${bestPosition}`);
        tooltip.classList.add('el-tooltip--visible');

        // 设置箭头位置
        const arrow = document.createElement('div');
        arrow.className = 'el-tooltip__arrow';
        tooltip.appendChild(arrow);

        // 根据方向设置箭头位置
        if (bestPosition === 'top' || bestPosition === 'bottom') {
            arrow.style.left = `${pos.arrow - targetRect.left}px`;
        } else {
            arrow.style.top = `${pos.arrow - targetRect.top}px`;
        }
    }
}

// 创建单例
const tooltip = new Tooltip();

export default tooltip;