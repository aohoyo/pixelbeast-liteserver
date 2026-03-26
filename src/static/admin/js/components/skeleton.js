/**
 * Skeleton - 骨架屏组件
 *
 * 用于加载状态的占位显示，提升用户体验
 */

/**
 * 生成骨架屏 HTML
 * @param {string} type - 骨架屏类型
 * @param {Object} options - 选项
 * @returns {string} HTML 字符串
 */
export function skeleton(type, options = {}) {
    const { count = 3, animated = true } = options;
    const cls = animated ? 'skeleton skeleton-animated' : 'skeleton';

    switch (type) {
        case 'text':
            return `<div class="${cls}-text" style="width: ${options.width || '100%'}"></div>`;

        case 'avatar':
            return `<div class="${cls}-avatar" style="width: ${options.size || '40px'}; height: ${options.size || '40px'}"></div>`;

        case 'card':
            return `
                <div class="skeleton-card">
                    <div class="${cls}" style="height: 200px; border-radius: var(--radius-lg) var(--radius-lg) 0 0;"></div>
                    <div class="skeleton-card-body">
                        <div class="${cls}-text" style="width: 60%; height: 20px;"></div>
                        <div class="${cls}-text" style="width: 80%; height: 16px; margin-top: 8px;"></div>
                    </div>
                </div>
            `;

        case 'table':
            return `
                <div class="skeleton-table">
                    <div class="skeleton-table-header">
                        ${Array(count).fill(`<div class="${cls}-text" style="width: 100px;"></div>`).join('')}
                    </div>
                    ${Array(5).fill(`
                        <div class="skeleton-table-row">
                            ${Array(count).fill(`<div class="${cls}-text"></div>`).join('')}
                        </div>
                    `).join('')}
                </div>
            `;

        case 'list':
            return `
                <div class="skeleton-list">
                    ${Array(options.count || 5).fill(`
                        <div class="skeleton-list-item">
                            ${options.avatar ? `<div class="${cls}-avatar"></div>` : ''}
                            <div class="skeleton-list-content">
                                <div class="${cls}-text" style="width: 60%;"></div>
                                <div class="${cls}-text" style="width: 40%; margin-top: 8px;"></div>
                            </div>
                        </div>
                    `).join('')}
                </div>
            `;

        case 'metric':
            return `
                <div class="skeleton-metric">
                    <div class="${cls}-text" style="width: 80px; height: 14px;"></div>
                    <div class="${cls}-text" style="width: 120px; height: 36px; margin-top: 8px;"></div>
                    <div class="skeleton-metric-chart"></div>
                </div>
            `;

        default:
            return `<div class="${cls}" style="width: ${options.width || '100%'}; height: ${options.height || '20px'};"></div>`;
    }
}

/**
 * 显示骨架屏
 * @param {HTMLElement|string} container - 容器元素或选择器
 * @param {string} type - 骨架屏类型
 * @param {Object} options - 选项
 */
export function showSkeleton(container, type, options = {}) {
    const el = typeof container === 'string' ? document.querySelector(container) : container;
    if (el) {
        el.innerHTML = skeleton(type, options);
        el.setAttribute('data-skeleton', 'true');
    }
}

/**
 * 隐藏骨架屏
 * @param {HTMLElement|string} container - 容器元素或选择器
 */
export function hideSkeleton(container) {
    const el = typeof container === 'string' ? document.querySelector(container) : container;
    if (el) {
        el.removeAttribute('data-skeleton');
    }
}

/**
 * 骨架屏加载器
 * @param {HTMLElement|string} container - 容器
 * @param {Function} loader - 加载函数
 * @param {string} type - 骨架屏类型
 * @param {Object} options - 选项
 */
export async function withSkeleton(container, loader, type, options = {}) {
    showSkeleton(container, type, options);
    try {
        const result = await loader();
        hideSkeleton(container);
        return result;
    } catch (error) {
        hideSkeleton(container);
        throw error;
    }
}

export default { skeleton, showSkeleton, hideSkeleton, withSkeleton };