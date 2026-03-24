/**
 * 网站管理标签页
 */

/**
 * 初始化网站管理标签页
 */
export function initSitesTab({ state, api, toast, events }) {
    console.log('[Sites] 初始化网站管理标签页');

    // 添加站点按钮
    const addSiteBtn = document.getElementById('add-site-btn');
    if (addSiteBtn) {
        addSiteBtn.addEventListener('click', () => showAddSiteModal({ state, api, toast }));
    }

    // 监听标签页切换
    events.match('tab:switch:sites', () => {
        loadSites({ state, api, toast });
    });

    // 监听状态加载完成
    events.match('status:loaded', (data) => {
        const currentTab = state.get('currentTab');
        if (currentTab === 'sites') {
            renderSites(data.sites || [], { state, api, toast });
        }
    });

    // 初始加载
    loadSites({ state, api, toast });
}

/**
 * 加载站点列表
 */
async function loadSites({ state, api, toast }) {
    try {
        const response = await api.get('/api/sites');
        const sites = await api.parseJSON(response);
        renderSites(sites || [], { state, api, toast });
    } catch (error) {
        console.error('[Sites] 加载失败:', error);
        toast.error('加载站点列表失败');
    }
}

/**
 * 渲染站点列表
 */
function renderSites(sites, { state, api, toast }) {
    const container = document.getElementById('sites-list');
    if (!container) return;

    if (sites.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <span class="empty-icon">🌐</span>
                <p>暂无网站</p>
                <button class="btn btn-primary" id="add-first-site">+ 添加第一个网站</button>
            </div>
        `;
        document.getElementById('add-first-site')?.addEventListener('click', () => {
            showAddSiteModal({ state, api, toast });
        });
        return;
    }

    container.innerHTML = sites.map(site => `
        <div class="site-card ${site.enabled ? '' : 'disabled'}" data-site-id="${site.id}">
            <div class="site-card-header">
                <div class="site-card-icon">${site.type === 'static' ? '📄' : '🔄'}</div>
                <div class="site-card-info">
                    <h4>${escapeHtml(site.name)}</h4>
                    <span class="badge ${site.enabled ? 'badge-success' : 'badge-secondary'}">
                        ${site.enabled ? '运行中' : '已停止'}
                    </span>
                </div>
                <div class="site-card-actions">
                    <button class="btn-icon" data-action="toggle" title="${site.enabled ? '停止' : '启动'}">
                        ${site.enabled ? '⏸' : '▶'}
                    </button>
                    <button class="btn-icon" data-action="edit" title="编辑">✏️</button>
                    <button class="btn-icon" data-action="delete" title="删除">🗑</button>
                </div>
            </div>
            <div class="site-card-body">
                <div class="site-info-row">
                    <span class="site-label">类型:</span>
                    <span class="site-value">${site.type === 'static' ? '静态文件' : '反向代理'}</span>
                </div>
                ${site.type === 'static' ? `
                    <div class="site-info-row">
                        <span class="site-label">根目录:</span>
                        <span class="site-value"><code>${escapeHtml(site.root)}</code></span>
                    </div>
                ` : ''}
                ${site.proxy ? `
                    <div class="site-info-row">
                        <span class="site-label">目标:</span>
                        <span class="site-value"><code>${escapeHtml(site.proxy.target)}</code></span>
                    </div>
                ` : ''}
                ${site.domains && site.domains.length > 0 ? `
                    <div class="site-info-row">
                        <span class="site-label">域名:</span>
                        <span class="site-value">${site.domains.map(d => `<code>${escapeHtml(d)}</code>`).join(', ')}</span>
                    </div>
                ` : ''}
                ${site.port > 0 ? `
                    <div class="site-info-row">
                        <span class="site-label">端口:</span>
                        <span class="site-value">${site.port}</span>
                    </div>
                ` : ''}
            </div>
        </div>
    `).join('');

    // 绑定事件
    container.querySelectorAll('.btn-icon').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const card = e.target.closest('.site-card');
            const siteId = card.dataset.siteId;
            const action = e.target.dataset.action;

            switch (action) {
                case 'toggle':
                    toggleSite(siteId, { state, api, toast });
                    break;
                case 'edit':
                    editSite(siteId, { state, api, toast });
                    break;
                case 'delete':
                    deleteSite(siteId, { state, api, toast });
                    break;
            }
        });
    });
}

/**
 * 显示添加站点对话框
 */
function showAddSiteModal({ state, api, toast }) {
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.innerHTML = `
        <div class="modal-content" style="max-width: 500px;">
            <h3>添加网站</h3>
            <form id="add-site-form">
                <div class="form-group">
                    <label>网站名称</label>
                    <input type="text" name="name" required placeholder="例如：我的博客">
                </div>
                <div class="form-group">
                    <label>网站类型</label>
                    <select name="type" required>
                        <option value="static">静态文件</option>
                        <option value="proxy">反向代理</option>
                    </select>
                </div>
                <div class="form-group" id="root-field">
                    <label>根目录</label>
                    <input type="text" name="root" placeholder="./data/sites/mysite">
                </div>
                <div class="form-group" id="proxy-field" style="display: none;">
                    <label>目标 URL</label>
                    <input type="text" name="proxy_target" placeholder="http://localhost:3000">
                </div>
                <div class="form-group">
                    <label>域名（可选，多个用逗号分隔）</label>
                    <input type="text" name="domains" placeholder="example.com,www.example.com">
                </div>
                <div class="form-group">
                    <label>端口（0 = 共享端口）</label>
                    <input type="number" name="port" value="0" min="0" max="65535">
                </div>
                <div class="modal-actions">
                    <button type="button" class="btn modal-cancel">取消</button>
                    <button type="submit" class="btn btn-primary">添加</button>
                </div>
            </form>
        </div>
    `;

    document.body.appendChild(modal);

    // 类型切换
    const typeSelect = modal.querySelector('select[name="type"]');
    const rootField = modal.querySelector('#root-field');
    const proxyField = modal.querySelector('#proxy-field');

    typeSelect.addEventListener('change', (e) => {
        if (e.target.value === 'static') {
            rootField.style.display = '';
            proxyField.style.display = 'none';
        } else {
            rootField.style.display = 'none';
            proxyField.style.display = '';
        }
    });

    // 表单提交
    const form = modal.querySelector('#add-site-form');
    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(form);

        const siteData = {
            name: formData.get('name'),
            type: formData.get('type'),
            enabled: true,
            port: parseInt(formData.get('port')) || 0,
            domains: formData.get('domains') ? formData.get('domains').split(',').map(d => d.trim()) : []
        };

        if (siteData.type === 'static') {
            siteData.root = formData.get('root') || `./data/sites/${generateID()}`;
        } else {
            siteData.proxy = {
                target: formData.get('proxy_target'),
                websocket: true,
                timeout: 30
            };
        }

        try {
            const response = await api.post('/api/sites', siteData);
            await api.parseJSON(response);
            toast.success('网站添加成功');
            modal.remove();
            loadSites({ state, api, toast });
        } catch (error) {
            toast.error(error.message || '添加失败');
        }
    });

    // 取消按钮
    modal.querySelector('.modal-cancel').addEventListener('click', () => modal.remove());
    modal.addEventListener('click', (e) => {
        if (e.target === modal) modal.remove();
    });
}

/**
 * 切换站点状态
 */
async function toggleSite(siteId, { state, api, toast }) {
    try {
        const response = await api.post('/api/sites/toggle', { id: siteId });
        await api.parseJSON(response);
        toast.success('站点状态已更新');
        loadSites({ state, api, toast });
    } catch (error) {
        toast.error('操作失败');
    }
}

/**
 * 编辑站点
 */
function editSite(siteId, { state, api, toast }) {
    // TODO: 实现编辑功能
    toast.info('编辑功能待实现');
}

/**
 * 删除站点
 */
async function deleteSite(siteId, { state, api, toast }) {
    if (!confirm('确定要删除这个网站吗？')) return;

    try {
        const response = await api.delete(`/api/sites/${siteId}`);
        await api.parseJSON(response);
        toast.success('网站已删除');
        loadSites({ state, api, toast });
    } catch (error) {
        toast.error('删除失败');
    }
}

/**
 * 生成随机 ID
 */
function generateID() {
    return Math.random().toString(36).substr(2, 9);
}

/**
 * HTML 转义
 */
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
