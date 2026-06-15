/**
 * 组件加载器
 * 负责动态加载 HTML 组件并插入页面
 */

// 组件映射表
const COMPONENTS = {
    home: 'components/home-section.html',
    sites: 'components/sites-section.html',
    ftp: 'components/ftp-section.html',
    files: 'components/files-section.html',
    terminal: 'components/terminal-section.html',
    settings: 'components/settings-section.html',
    cert: 'components/cert-section.html',
    logs: 'components/logs-section.html',
    modal: 'components/modal.html'
};

// 组件缓存
const componentCache = new Map();

/**
 * 加载单个组件
 * @param {string} name - 组件名称
 * @returns {Promise<string>} HTML 内容
 */
export async function loadComponent(name) {
    // 检查缓存
    if (componentCache.has(name)) {
        return componentCache.get(name);
    }

    const url = COMPONENTS[name];
    if (!url) {
        throw new Error(`组件 "${name}" 不存在`);
    }

    try {
        const response = await fetch(url);
        if (!response.ok) {
            throw new Error(`加载组件失败: ${response.status}`);
        }
        const html = await response.text();
        componentCache.set(name, html);
        return html;
    } catch (error) {
        console.error(`加载组件 "${name}" 失败:`, error);
        throw error;
    }
}

/**
 * 加载所有内容区域组件
 */
export async function loadContentSections() {
    const contentArea = document.querySelector('.content-area');
    if (!contentArea) {
        throw new Error('未找到 .content-area 容器');
    }

    // 清空原有的"加载中..."提示
    contentArea.innerHTML = '';

    // 并行加载各组件，按顺序插入 DOM
    const sections = ['home', 'sites', 'ftp', 'files', 'terminal', 'settings', 'cert', 'logs'];

    const results = await Promise.allSettled(sections.map(name => loadComponent(name)));

    for (let i = 0; i < sections.length; i++) {
        const name = sections[i];
        const result = results[i];
        const section = document.createElement('section');
        section.id = name;
        section.className = 'tab-content';
        if (name === 'home') {
            section.classList.add('active');
        }

        if (result.status === 'fulfilled') {
            section.innerHTML = result.value;
        } else {
            console.error(`加载组件 "${name}" 失败:`, result.reason);
            section.innerHTML = `<div class="card"><p class="loading">加载失败: ${result.reason?.message || '未知错误'}</p></div>`;
        }
        contentArea.appendChild(section);
    }
}

/**
 * 加载模态框组件
 */
export async function loadModal() {
    try {
        const html = await loadComponent('modal');
        document.body.insertAdjacentHTML('beforeend', html);
    } catch (error) {
        console.error('加载模态框失败:', error);
    }
}

/**
 * 预加载所有组件（可选，用于提升性能）
 */
export async function preloadComponents() {
    const promises = Object.keys(COMPONENTS).map(name => loadComponent(name));
    await Promise.allSettled(promises);
}

/**
 * 清除组件缓存
 */
export function clearComponentCache() {
    componentCache.clear();
}
