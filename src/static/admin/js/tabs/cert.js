/**
 * 证书管理模块
 *
 * SSL 证书列表、申请、上传、续期、删除
 * 支持 Let's Encrypt，HTTP/DNS/文件验证
 * DNS 服务商管理（CRUD），步骤式证书申请（实时日志轮询）
 */

import { BaseTab } from './BaseTab.js';
import { escapeHtml } from '../core/utils.js';

class CertTab extends BaseTab {
    constructor(deps) {
        super(deps, 'cert');
        this.certs = [];
        this.dnsProviders = [];
        this._pendingDomain = null;
        this._pollTimer = null;
        this._deployDomain = null;
        this._deploySites = [];
    }

    onInit() {
        this.bindEvents();
        this.bindDNSProviderEvents();
    }

    // ========== 事件绑定 ==========

    bindEvents() {
        // 标签页切换
        this.$$('.cert-tabs .tab-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                this.switchCertTab(e.currentTarget.dataset.certTab);
            });
        });

        // 申请证书按钮
        this.$('#cert-request-btn')?.addEventListener('click', () => this.showRequestModal());
        this.$('#cert-request-confirm')?.addEventListener('click', () => this.requestCert());
        this.$('#cert-request-close')?.addEventListener('click', () => this.closeRequestModal());
        this.$('#cert-request-cancel')?.addEventListener('click', () => this.closeRequestModal());

        // 上传证书按钮
        this.$('#cert-upload-btn')?.addEventListener('click', () => this.showUploadModal());
        this.$('#cert-upload-confirm')?.addEventListener('click', () => this.uploadCert());
        this.$('#cert-upload-close')?.addEventListener('click', () => this.hideModal('cert-upload-modal'));
        this.$('#cert-upload-cancel')?.addEventListener('click', () => this.hideModal('cert-upload-modal'));

        // 文件选择区域
        this.bindUploadZone('cert-file-zone', 'cert-file-input', 'cert-file-text');
        this.bindUploadZone('key-file-zone', 'key-file-input', 'key-file-text');

        // 验证方式切换
        this.$('#cert-request-challenge')?.addEventListener('change', (e) => this.onChallengeChange(e.target.value));

        // DNS 服务商切换（申请表单内）
        this.$('#cert-dns-provider-select')?.addEventListener('change', (e) => this.onRequestDNSProviderChange(e.target.value));

        // DNS 验证按钮
        this.$('#cert-dns-verify-btn')?.addEventListener('click', () => this.completeDNSChallenge());

        // 文件验证按钮
        this.$('#cert-file-verify-btn')?.addEventListener('click', () => this.completeFileChallenge());

        // 域名输入自动检测通配符
        this.$('#cert-request-domain')?.addEventListener('input', (e) => {
            const domain = e.target.value.trim();
            if (domain.startsWith('*.')) {
                const challenge = this.$('#cert-request-challenge');
                if (challenge && challenge.value !== 'dns') {
                    challenge.value = 'dns';
                    this.onChallengeChange('dns');
                    this.toast?.info('通配符域名自动切换为 DNS 验证');
                }
            }
        });

        // 复制按钮
        document.querySelectorAll('.cert-copy-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                const targetId = btn.dataset.copy;
                const el = this.$(`#${targetId}`);
                if (el) {
                    navigator.clipboard.writeText(el.textContent).then(() => {
                        this.toast?.success('已复制到剪贴板');
                    });
                }
            });
        });

        // 部署到站点
        this.$('#cert-deploy-close')?.addEventListener('click', () => this.hideModal('cert-deploy-modal'));
        this.$('#cert-deploy-cancel')?.addEventListener('click', () => this.hideModal('cert-deploy-modal'));
        this.$('#cert-deploy-confirm')?.addEventListener('click', () => this.deployCertToSites());
        this.$('#cert-deploy-modal')?.querySelector('.modal-overlay')?.addEventListener('click', () => this.hideModal('cert-deploy-modal'));
    }

    bindDNSProviderEvents() {
        // DNS 服务商 CRUD
        this.$('#dns-provider-add-btn')?.addEventListener('click', () => this.showAddDNSProviderModal());
        this.$('#dns-provider-modal-close')?.addEventListener('click', () => this.hideModal('dns-provider-modal'));
        this.$('#dns-provider-modal-cancel')?.addEventListener('click', () => this.hideModal('dns-provider-modal'));
        this.$('#dns-provider-modal-save')?.addEventListener('click', () => this.saveDNSProvider());
        this.$('#dns-provider-modal-test')?.addEventListener('click', () => this.testDNSProviderModal());

        // DNS 服务商类型切换（模态框内）
        this.$('#dns-provider-type')?.addEventListener('change', (e) => this.onDNSModalTypeChange(e.target.value));

        // 密码显示/隐藏切换
        this.$$('.input-toggle-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                const input = this.$(`#${btn.dataset.toggle}`);
                if (!input) return;
                const eye = btn.querySelector('.icon-eye');
                const eyeOff = btn.querySelector('.icon-eye-off');
                if (input.type === 'password') {
                    input.type = 'text';
                    if (eye) eye.style.display = 'none';
                    if (eyeOff) eyeOff.style.display = 'block';
                } else {
                    input.type = 'password';
                    if (eye) eye.style.display = 'block';
                    if (eyeOff) eyeOff.style.display = 'none';
                }
            });
        });
    }

    bindUploadZone(zoneId, inputId, textId) {
        const zone = this.$(`#${zoneId}`);
        const input = this.$(`#${inputId}`);
        if (!zone || !input) return;

        zone.addEventListener('click', () => input.click());
        zone.addEventListener('dragover', (e) => { e.preventDefault(); zone.classList.add('dragover'); });
        zone.addEventListener('dragleave', () => zone.classList.remove('dragover'));
        zone.addEventListener('drop', (e) => {
            e.preventDefault();
            zone.classList.remove('dragover');
            if (e.dataTransfer.files.length > 0) {
                input.files = e.dataTransfer.files;
                this.onFileSelected(input, textId, zone);
            }
        });
        input.addEventListener('change', () => this.onFileSelected(input, textId, zone));
    }

    onFileSelected(input, textId, zone) {
        const file = input.files[0];
        if (file) {
            this.setText(`#${textId}`, file.name);
            zone?.classList.add('file-selected');
        }
    }

    // ========== 生命周期 ==========

    async onLoad() {
        await Promise.all([
            this.loadCerts(),
            this.loadDNSProviders()
        ]);
    }

    async onRefresh() {
        await Promise.all([
            this.loadCerts(),
            this.loadDNSProviders()
        ]);
    }

    // ========== 证书列表 ==========

    async loadCerts() {
        try {
            const data = await this.api.getJSON('/api/certs');
            this.certs = Array.isArray(data) ? data : [];
        } catch {
            this.certs = [];
        }
        this.renderCertList();
        this.renderSummary();
    }

    renderSummary() {
        const total = this.certs.length;
        const expiring = this.certs.filter(c => c.has_cert && c.days_left >= 0 && c.days_left < 30).length;
        const valid = this.certs.filter(c => c.has_cert && c.days_left >= 30).length;

        this.setText('#cert-total-count', total);
        this.setText('#cert-valid-count', valid);
        this.setText('#cert-expiring-count', expiring);
    }

    renderCertList() {
        const list = this.$('#cert-list');
        if (!list) return;

        if (this.certs.length === 0) {
            list.innerHTML = `
                <div class="cert-empty">
                    <div class="cert-empty-icon">🔒</div>
                    <p>暂无证书</p>
                    <p style="font-size:12px;margin-top:4px">点击"申请证书"自动获取免费证书，或"上传证书"使用自定义证书</p>
                </div>`;
            return;
        }

        list.innerHTML = this.certs.map(cert => this.renderCertItem(cert)).join('');

        // 绑定操作按钮事件
        list.querySelectorAll('[data-action]').forEach(btn => {
            btn.addEventListener('click', () => {
                const domain = btn.dataset.domain;
                const action = btn.dataset.action;
                if (action === 'renew') this.renewCert(domain);
                else if (action === 'delete') this.deleteCert(domain);
                else if (action === 'deploy') this.showDeployModal(domain);
            });
        });
    }

    renderCertItem(cert) {
        const badgeClass = this.getBadgeClass(cert);
        const badgeText = this.getBadgeText(cert);
        const daysClass = this.getDaysClass(cert.days_left);
        const daysText = cert.has_cert ? `${cert.days_left} 天` : '-';
        const challengeIcon = this.getChallengeIcon(cert.challenge_method);

        return `
            <div class="cert-item">
                <div class="cert-item-info">
                    <div class="cert-item-domain">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:16px;height:16px;flex-shrink:0"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path></svg>
                        ${escapeHtml(cert.domain)}
                        ${challengeIcon}
                    </div>
                    <div class="cert-item-meta">
                        ${cert.has_cert ? `
                            <span>签发者: ${escapeHtml(cert.issuer || '未知')}</span>
                            <span>有效期: ${this.formatDate(cert.not_before)} ~ ${this.formatDate(cert.not_after)}</span>
                            <span class="${daysClass}">剩余 ${daysText}</span>
                        ` : `
                            <span style="color:var(--text-muted)">证书未安装</span>
                        `}
                    </div>
                </div>
                <div class="cert-item-actions">
                    <span class="cert-badge ${badgeClass}">${badgeText}</span>
                    ${cert.has_cert ? `<button class="btn btn-sm" data-action="deploy" data-domain="${escapeHtml(cert.domain)}">部署到站点</button>` : ''}
                    ${cert.has_cert && cert.auto_https ? `<button class="btn btn-sm" data-action="renew" data-domain="${escapeHtml(cert.domain)}">续期</button>` : ''}
                    <button class="btn btn-sm btn-danger" data-action="delete" data-domain="${escapeHtml(cert.domain)}">删除</button>
                </div>
            </div>`;
    }

    getChallengeIcon(method) {
        switch (method) {
            case 'dns':
                return '<span class="cert-challenge-icon" title="DNS 验证">DNS</span>';
            case 'http-file':
                return '<span class="cert-challenge-icon" title="文件验证">FILE</span>';
            case 'http-auto':
                return '<span class="cert-challenge-icon" title="HTTP 验证">HTTP</span>';
            default:
                return '';
        }
    }

    getBadgeClass(cert) {
        if (!cert.has_cert) return 'cert-badge-selfsigned';
        if (cert.days_left < 0) return 'cert-badge-expired';
        if (cert.days_left < 30) return 'cert-badge-expiring';
        if (cert.type === 'auto') return 'cert-badge-auto';
        if (cert.type === 'custom') return 'cert-badge-custom';
        return 'cert-badge-selfsigned';
    }

    getBadgeText(cert) {
        if (!cert.has_cert) return '未安装';
        if (cert.days_left < 0) return '已过期';
        if (cert.days_left < 30) return '即将过期';
        if (cert.type === 'auto') return "Let's Encrypt";
        if (cert.type === 'custom') return '自定义';
        return '自签名';
    }

    getDaysClass(days) {
        if (days < 0) return 'days-danger';
        if (days < 30) return 'days-warn';
        return 'days-safe';
    }

    formatDate(dateStr) {
        if (!dateStr) return '-';
        try {
            const d = new Date(dateStr);
            return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
        } catch {
            return dateStr;
        }
    }

    // ========== DNS 服务商管理 ==========

    async loadDNSProviders() {
        try {
            const data = await this.api.getJSON('/api/certs/dns-providers');
            this.dnsProviders = Array.isArray(data) ? data : [];
        } catch {
            this.dnsProviders = [];
        }
        this.renderDNSProviders();
    }

    renderDNSProviders() {
        const list = this.$('#dns-provider-list');
        if (!list) return;

        if (this.dnsProviders.length === 0) {
            list.innerHTML = `
                <div class="cert-empty" style="padding:var(--space-lg) 0">
                    <p style="font-size:13px;color:var(--text-muted)">暂无 DNS 服务商配置，添加后可快速申请 DNS 验证证书</p>
                </div>`;
            return;
        }

        const typeLabels = { alidns: '阿里云 DNS', tencentcloud: '腾讯云 DNSPod', baota: '宝塔域名' };
        const typeColors = { alidns: 'badge-primary', tencentcloud: 'badge-info', baota: 'badge-success' };

        list.innerHTML = `
            <div class="dns-list">
                ${this.dnsProviders.map(p => `
                    <div class="dns-provider-item" data-id="${escapeHtml(String(p.id))}">
                        <div class="dns-provider-info">
                            <div class="dns-provider-name">${escapeHtml(p.name)}</div>
                            <span class="badge ${typeColors[p.type] || 'badge-primary'}">${typeLabels[p.type] || p.type}</span>
                        </div>
                        <div class="dns-provider-actions">
                            <button class="btn btn-sm" data-dns-action="test" data-dns-id="${escapeHtml(String(p.id))}" title="测试连接">测试</button>
                            <button class="btn btn-sm" data-dns-action="edit" data-dns-id="${escapeHtml(String(p.id))}" title="编辑">编辑</button>
                            <button class="btn btn-sm btn-danger" data-dns-action="delete" data-dns-id="${escapeHtml(String(p.id))}" title="删除">删除</button>
                        </div>
                    </div>
                `).join('')}
            </div>
        `;

        // 绑定操作按钮
        list.querySelectorAll('[data-dns-action]').forEach(btn => {
            btn.addEventListener('click', () => {
                const action = btn.dataset.dnsAction;
                const id = btn.dataset.dnsId;
                if (action === 'edit') {
                    // 从内存数据取，不从 HTML 属性取，避免编码问题
                    const provider = this.dnsProviders.find(p => String(p.id) === String(id));
                    if (provider) this.showEditDNSProviderModal(provider);
                } else if (action === 'delete') {
                    this.deleteDNSProvider(id);
                } else if (action === 'test') {
                    this.testDNSProvider(id);
                }
            });
        });
    }

    showAddDNSProviderModal() {
        const modal = this.$('#dns-provider-modal');
        if (!modal) return;

        this.setText('#dns-provider-modal-title', '添加 DNS 服务商');
        this.$('#dns-provider-edit-id') && (this.$('#dns-provider-edit-id').value = '');
        this.$('#dns-provider-name') && (this.$('#dns-provider-name').value = '');
        this.$('#dns-provider-type') && (this.$('#dns-provider-type').value = 'alidns');

        // 清空凭证字段
        this.clearDNSModalCreds();

        // 显示默认类型凭证表单
        this.onDNSModalTypeChange('alidns');
        modal.classList.add('active');
    }

    async showEditDNSProviderModal(provider) {
        const modal = this.$('#dns-provider-modal');
        if (!modal) return;

        this.setText('#dns-provider-modal-title', '编辑 DNS 服务商');
        this.$('#dns-provider-edit-id') && (this.$('#dns-provider-edit-id').value = provider.id || '');
        this.$('#dns-provider-name') && (this.$('#dns-provider-name').value = provider.name || '');
        this.$('#dns-provider-type') && (this.$('#dns-provider-type').value = provider.type || 'alidns');

        // 清空凭证字段
        this.clearDNSModalCreds();

        // 显示对应类型凭证表单
        this.onDNSModalTypeChange(provider.type);
        modal.classList.add('active');

        // 异步获取明文凭证并回填
        try {
            const data = await this.api.getJSON('/api/certs/dns-providers/' + encodeURIComponent(provider.id) + '/credentials');
            const creds = data.credentials || {};
            switch (provider.type) {
                case 'alidns':
                    if (creds.access_key) this.setCredentialValue('dns-modal-alidns-key', creds.access_key);
                    if (creds.secret_key) this.setCredentialValue('dns-modal-alidns-secret', creds.secret_key);
                    break;
                case 'tencentcloud':
                    if (creds.secret_id) this.setCredentialValue('dns-modal-tencent-id', creds.secret_id);
                    if (creds.secret_key) this.setCredentialValue('dns-modal-tencent-key', creds.secret_key);
                    break;
                case 'baota':
                    if (creds.account_id) this.setCredentialValue('dns-modal-baota-account', creds.account_id);
                    if (creds.access_key) this.setCredentialValue('dns-modal-baota-ak', creds.access_key);
                    if (creds.secret_key) this.setCredentialValue('dns-modal-baota-sk', creds.secret_key);
                    if (creds.domain_id) this.setCredentialValue('dns-modal-baota-domain-id', creds.domain_id);
                    break;
            }
        } catch (e) {
            // 获取失败时用脱敏值作 placeholder 提示
            const mc = provider.masked_creds || {};
            switch (provider.type) {
                case 'alidns':
                    if (mc.access_key) this.setPlaceholder('dns-modal-alidns-key', '当前值: ' + mc.access_key);
                    if (mc.secret_key) this.setPlaceholder('dns-modal-alidns-secret', '当前值: ' + mc.secret_key);
                    break;
                case 'tencentcloud':
                    if (mc.secret_id) this.setPlaceholder('dns-modal-tencent-id', '当前值: ' + mc.secret_id);
                    if (mc.secret_key) this.setPlaceholder('dns-modal-tencent-key', '当前值: ' + mc.secret_key);
                    break;
                case 'baota':
                    if (mc.account_id) this.setPlaceholder('dns-modal-baota-account', '当前值: ' + mc.account_id);
                    if (mc.access_key) this.setPlaceholder('dns-modal-baota-ak', '当前值: ' + mc.access_key);
                    if (mc.secret_key) this.setPlaceholder('dns-modal-baota-sk', '当前值: ' + mc.secret_key);
                    if (mc.domain_id) this.setPlaceholder('dns-modal-baota-domain-id', '当前值: ' + mc.domain_id);
                    break;
            }
        }
    }

    clearDNSModalCreds() {
        const fields = [
            'dns-modal-alidns-key', 'dns-modal-alidns-secret',
            'dns-modal-tencent-id', 'dns-modal-tencent-key',
            'dns-modal-baota-account', 'dns-modal-baota-ak', 'dns-modal-baota-sk', 'dns-modal-baota-domain-id'
        ];
        fields.forEach(id => {
            const el = this.$(`#${id}`);
            if (el) el.value = '';
        });
    }

    setCredentialValue(id, value) {
        const el = this.$(`#${id}`);
        if (el) {
            el.value = value;
            el.placeholder = '';
        }
    }

    setPlaceholder(id, text) {
        const el = this.$(`#${id}`);
        if (el) el.placeholder = text;
    }

    onDNSModalTypeChange(type) {
        // 隐藏所有凭证区域
        const sections = ['dns-modal-alidns', 'dns-modal-tencent', 'dns-modal-baota'];
        sections.forEach(id => {
            const el = this.$(`#${id}`);
            if (el) el.style.display = 'none';
        });

        // 显示对应类型
        const target = this.$(`#dns-modal-${type === 'tencentcloud' ? 'tencent' : type}`);
        if (target) target.style.display = 'block';
    }

    async saveDNSProvider() {
        const editId = this.$('#dns-provider-edit-id')?.value;
        const name = this.$('#dns-provider-name')?.value.trim();
        const type = this.$('#dns-provider-type')?.value;

        if (!name) {
            this.toast?.warning('请输入服务商名称');
            return;
        }

        const credentials = this.getDNSModalCredentials(type);

        try {
            if (editId) {
                // 更新
                await this.api.post('/api/certs/dns-providers/' + encodeURIComponent(editId), {
                    name, type, credentials
                });
                this.toast?.success('DNS 服务商已更新');
            } else {
                // 新建
                await this.api.post('/api/certs/dns-providers', {
                    name, type, credentials
                });
                this.toast?.success('DNS 服务商已添加');
            }
            this.hideModal('dns-provider-modal');
            await this.loadDNSProviders();
        } catch (error) {
            this.toast?.error('保存失败: ' + (error.message || '未知错误'));
        }
    }

    getDNSModalCredentials(type) {
        const creds = {};
        switch (type) {
            case 'alidns':
                creds.access_key = this.$('#dns-modal-alidns-key')?.value || '';
                creds.secret_key = this.$('#dns-modal-alidns-secret')?.value || '';
                break;
            case 'tencentcloud':
                creds.secret_id = this.$('#dns-modal-tencent-id')?.value || '';
                creds.secret_key = this.$('#dns-modal-tencent-key')?.value || '';
                break;
            case 'baota':
                creds.account_id = this.$('#dns-modal-baota-account')?.value || '';
                creds.access_key = this.$('#dns-modal-baota-ak')?.value || '';
                creds.secret_key = this.$('#dns-modal-baota-sk')?.value || '';
                creds.domain_id = this.$('#dns-modal-baota-domain-id')?.value || '';
                break;
        }
        return creds;
    }

    async deleteDNSProvider(id) {
        const item = this.$(`.dns-provider-item[data-id="${id}"]`);
        const provider = this.dnsProviders.find(p => String(p.id) === String(id));
        const name = provider ? provider.name : id;

        if (!confirm(`确定删除 DNS 服务商「${name}」？删除后不可恢复。`)) return;

        const btn = item?.querySelector('[data-dns-action="delete"]');
        if (btn) { btn.disabled = true; btn.textContent = '删除中...'; }

        try {
            await this.api.delete('/api/certs/dns-providers/' + encodeURIComponent(id));
            this.toast?.success(`DNS 服务商「${name}」已删除`);
            await this.loadDNSProviders();
        } catch (error) {
            this.toast?.error('删除失败: ' + (error.message || '未知错误'));
            if (btn) { btn.disabled = false; btn.textContent = '删除'; }
        }
    }

    async testDNSProvider(id) {
        const item = this.$(`.dns-provider-item[data-id="${id}"]`);
        const btn = item?.querySelector('[data-dns-action="test"]');
        if (btn) { btn.disabled = true; btn.textContent = '测试中...'; }

        this.clearTestStatus(item);

        try {
            const data = await this.api.postJSON('/api/certs/dns-providers/' + encodeURIComponent(id) + '/test');
            if (data && data.success) {
                this.showTestStatus(item, 'success', '连接成功');
                this.toast?.success(data.message || '连接测试成功');
            } else {
                this.showTestStatus(item, 'fail', '连接失败');
                this.toast?.error(data?.message || '连接测试失败');
            }
        } catch (error) {
            this.showTestStatus(item, 'fail', '连接失败');
            this.toast?.error('测试失败: ' + (error.message || '未知错误'));
        } finally {
            if (btn) { btn.disabled = false; btn.textContent = '测试'; }
        }
    }

    showTestStatus(item, status, text) {
        if (!item) return;
        this.clearTestStatus(item);
        const nameEl = item.querySelector('.dns-provider-name');
        if (!nameEl) return;
        const el = document.createElement('span');
        el.className = `dns-test-status ${status}`;
        el.textContent = text;
        nameEl.appendChild(el);
    }

    clearTestStatus(row) {
        if (!row) return;
        row.querySelectorAll('.dns-test-status').forEach(el => el.remove());
    }

    // ========== 提供商/验证方式切换 ==========

    onChallengeChange(method) {
        // 显示/隐藏 DNS 服务商区域
        const dnsSection = this.$('#cert-dns-provider-section');
        if (dnsSection) {
            dnsSection.style.display = method === 'dns' ? 'block' : 'none';
            // 当切换到 DNS 验证时，刷新已保存服务商下拉列表
            if (method === 'dns') {
                this.populateSavedDNSProviders();
            }
        }

        // 隐藏验证步骤面板
        const dnsStep = this.$('#cert-dns-step');
        const fileStep = this.$('#cert-file-step');
        if (dnsStep) dnsStep.style.display = 'none';
        if (fileStep) fileStep.style.display = 'none';

        // 更新确认按钮文本
        const confirmBtn = this.$('#cert-request-confirm');
        if (confirmBtn) {
            switch (method) {
                case 'dns': confirmBtn.textContent = '获取 DNS 记录'; break;
                case 'http-file': confirmBtn.textContent = '准备验证文件'; break;
                default: confirmBtn.textContent = '提交申请'; break;
            }
        }

        this.updateNotes();
    }

    /**
     * 申请表单内 DNS 服务商选择变化
     */
    onRequestDNSProviderChange(value) {
        const savedHint = this.$('#cert-dns-saved-hint');
        if (value.startsWith('saved_')) {
            if (savedHint) savedHint.style.display = 'block';
        } else {
            if (savedHint) savedHint.style.display = 'none';
        }
    }

    /**
     * 将已保存的 DNS 服务商填充到申请表单的下拉框中
     */
    populateSavedDNSProviders() {
        const select = this.$('#cert-dns-provider-select');
        if (!select) return;

        // 清除所有选项
        select.innerHTML = '';

        const noSavedHint = this.$('#cert-dns-no-saved');

        if (this.dnsProviders.length > 0) {
            if (noSavedHint) noSavedHint.style.display = 'none';

            // 添加已保存的服务商
            this.dnsProviders.forEach(p => {
                const opt = document.createElement('option');
                opt.value = `saved_${p.id}`;
                opt.textContent = p.name;
                select.appendChild(opt);
            });

            // 默认选第一个
            select.selectedIndex = 0;
            this.onRequestDNSProviderChange(select.value);
        } else {
            if (noSavedHint) noSavedHint.style.display = 'block';
            // 添加占位选项
            const placeholder = document.createElement('option');
            placeholder.value = '';
            placeholder.textContent = '请先添加 DNS 服务商';
            placeholder.disabled = true;
            placeholder.selected = true;
            select.appendChild(placeholder);
        }
    }

    updateNotes() {
        const method = this.$('#cert-request-challenge')?.value || 'http-auto';
        const noteList = this.$('#cert-note-list');
        if (!noteList) return;

        const notes = [];
        switch (method) {
            case 'http-auto':
                notes.push('服务器需要开放端口 80（用于 ACME HTTP 验证）');
                notes.push('域名需要正确解析到本服务器');
                break;
            case 'http-file':
                notes.push('验证文件将自动托管，需要服务器开放端口 80');
                notes.push('域名需要正确解析到本服务器');
                notes.push('验证过程完全自动，无需手动操作');
                break;
            case 'dns':
                notes.push('支持通配符域名如 *.example.com');
                notes.push('需要已保存的 DNS 服务商配置');
                notes.push('验证过程全自动，无需手动操作');
                notes.push('DNS 传播可能需要几分钟');
                break;
        }
        notes.push('证书将自动续期（到期前 30 天）');
        noteList.innerHTML = notes.map(n => `<li>${n}</li>`).join('');
    }

    // ========== 步骤指示器 ==========

    /**
     * 更新步骤指示器
     * @param {number} step - 当前步骤 (1-5)
     * @param {string} status - 'active' | 'done' | 'error'
     */
    updateStepIndicator(step, status) {
        const stepsBar = this.$('#cert-steps-bar');
        if (!stepsBar) return;

        const steps = stepsBar.querySelectorAll('.cert-step');
        steps.forEach(el => {
            const s = parseInt(el.dataset.step);
            el.classList.remove('active', 'done', 'error');

            if (s < step) {
                el.classList.add('done');
            } else if (s === step) {
                el.classList.add(status);
            }
        });
    }

    resetStepIndicator() {
        const stepsBar = this.$('#cert-steps-bar');
        if (!stepsBar) return;
        stepsBar.querySelectorAll('.cert-step').forEach(el => {
            el.classList.remove('active', 'done', 'error');
        });
    }

    // ========== 日志面板 ==========

    /**
     * 显示日志面板，隐藏表单
     */
    showLogPanel() {
        const form = this.$('#cert-step-form');
        const logPanel = this.$('#cert-log-panel');
        const footer = this.$('#cert-request-footer');

        if (form) form.style.display = 'none';
        if (logPanel) logPanel.style.display = 'block';
        if (footer) footer.style.display = 'none';
    }

    /**
     * 隐藏日志面板，恢复表单
     */
    hideLogPanel() {
        const form = this.$('#cert-step-form');
        const logPanel = this.$('#cert-log-panel');
        const footer = this.$('#cert-request-footer');

        if (form) form.style.display = 'block';
        if (logPanel) logPanel.style.display = 'none';
        if (footer) footer.style.display = 'flex';
    }

    /**
     * 追加一条日志到日志面板
     */
    appendLog(time, message, level) {
        const container = this.$('#cert-log-content');
        if (!container) return;

        const colorMap = {
            info: 'var(--text-secondary)',
            success: 'var(--success, #22c55e)',
            error: 'var(--danger, #ef4444)',
            warn: 'var(--warning, #fbbf24)'
        };
        const color = colorMap[level] || colorMap.info;

        const line = document.createElement('div');
        line.className = 'cert-log-line';
        line.innerHTML = `<span class="cert-log-time" style="color:var(--text-muted)">${escapeHtml(time)}</span> <span class="cert-log-msg" style="color:${color}">${escapeHtml(message)}</span>`;
        container.appendChild(line);

        // 自动滚动到底部
        container.scrollTop = container.scrollHeight;
    }

    /**
     * 清空日志面板
     */
    clearLogs() {
        const container = this.$('#cert-log-content');
        if (container) container.innerHTML = '';
    }

    // ========== 日志轮询 ==========

    /**
     * 开始轮询证书申请进度（自适应间隔：2s → 5s）
     */
    startLogPolling(domain) {
        this.stopLogPolling();
        this._polledLogCount = 0;
        this._pollRound = 0;
        this._pollInterval = 2000;
        this._scheduleNextPoll(domain);
    }

    /**
     * 调度下一次轮询
     */
    _scheduleNextPoll(domain) {
        this._pollTimer = setTimeout(async () => {
            await this.pollProgress(domain);
            if (this._pollTimer !== null) {
                this._pollRound++;
                // 逐步增加轮询间隔: 2s → 3s → 4s → 5s（上限）
                if (this._pollRound > 4 && this._pollInterval < 5000) {
                    this._pollInterval += 1000;
                }
                this._scheduleNextPoll(domain);
            }
        }, this._pollInterval);
    }

    /**
     * 停止轮询
     */
    stopLogPolling() {
        if (this._pollTimer) {
            clearTimeout(this._pollTimer);
            this._pollTimer = null;
        }
    }

    /**
     * 轮询单次进度
     */
    async pollProgress(domain) {
        try {
            const data = await this.api.getJSON('/api/certs/progress/' + encodeURIComponent(domain));
            if (!data) return;

            // 更新步骤指示器
            if (data.step) {
                const status = data.status === 'error' ? 'error' : 'active';
                this.updateStepIndicator(data.step, status);
            }

            // 只追加新日志（按已处理条数去重）
            if (Array.isArray(data.logs)) {
                const offset = this._polledLogCount || 0;
                const newLogs = data.logs.slice(offset);
                newLogs.forEach(log => {
                    this.appendLog(log.time || '', log.message || '', log.level || 'info');
                });
                this._polledLogCount = data.logs.length;
            }

            // 检查是否完成
            if (data.status === 'success') {
                this.stopLogPolling();
                this.updateStepIndicator(5, 'done');
                this.appendLog(new Date().toLocaleTimeString(), '证书申请成功！', 'success');
                this.toast?.success('证书申请成功');
                // 延迟关闭申请弹窗，然后弹出部署到站点对话框
                setTimeout(() => {
                    this.hideModal('cert-request-modal');
                    this.loadCerts().then(() => {
                        this.showDeployModal(domain);
                    });
                }, 1500);
            } else if (data.status === 'error') {
                this.stopLogPolling();
                this.appendLog(new Date().toLocaleTimeString(), data.step_text || '申请失败', 'error');
                this.toast?.error('证书申请失败: ' + (data.step_text || '未知错误'));
            }
        } catch (error) {
            // 网络错误不中断轮询，等待下次
        }
    }

    // ========== 申请证书 ==========

    showRequestModal() {
        const modal = this.$('#cert-request-modal');
        if (modal) {
            // 重置表单
            this.$('#cert-request-domain') && (this.$('#cert-request-domain').value = '');
            this.$('#cert-request-email') && (this.$('#cert-request-email').value = '');

            this.$('#cert-request-challenge') && (this.$('#cert-request-challenge').value = 'http-auto');
            this.$('#cert-dns-provider-select') && (this.$('#cert-dns-provider-select').value = 'manual');

            // 显示表单，隐藏日志面板
            const form = this.$('#cert-step-form');
            const logPanel = this.$('#cert-log-panel');
            const footer = this.$('#cert-request-footer');
            const dnsStep = this.$('#cert-dns-step');
            const fileStep = this.$('#cert-file-step');
            const dnsSection = this.$('#cert-dns-provider-section');
            if (form) form.style.display = 'block';
            if (logPanel) logPanel.style.display = 'none';
            if (footer) footer.style.display = 'flex';
            if (dnsStep) dnsStep.style.display = 'none';
            if (fileStep) fileStep.style.display = 'none';
            if (dnsSection) dnsSection.style.display = 'none';

            // 重置 DNS 提示区域
            const savedHint = this.$('#cert-dns-saved-hint');
            if (savedHint) savedHint.style.display = 'none';

            // 重置按钮文本
            const confirmBtn = this.$('#cert-request-confirm');
            if (confirmBtn) confirmBtn.textContent = '提交申请';

            // 重置步骤指示器
            this.resetStepIndicator();
            this.clearLogs();

            // 填充已保存服务商
            this.populateSavedDNSProviders();

            this._pendingDomain = null;
            this.updateNotes();
            modal.classList.add('active');
        }
    }

    /**
     * 关闭申请模态框，停止轮询
     */
    closeRequestModal() {
        this.stopLogPolling();
        this.hideModal('cert-request-modal');
    }

    async requestCert() {
        const domain = this.$('#cert-request-domain')?.value.trim();
        const email = this.$('#cert-request-email')?.value.trim();
        const challenge = this.$('#cert-request-challenge')?.value || 'http-auto';

        if (!domain) {
            this.toast?.warning('请输入域名');
            return;
        }

        try {
            if (challenge === 'http-auto') {
                // HTTP 自动验证：提交后进入日志轮询模式
                this.showLogPanel();
                this.clearLogs();
                this.resetStepIndicator();
                this.appendLog(new Date().toLocaleTimeString(), '正在提交申请请求...', 'info');

                await this.api.post('/api/certs/request', {
                    domain, email, challenge_method: challenge
                });

                this.appendLog(new Date().toLocaleTimeString(), '申请已提交，等待处理...', 'info');
                this.startLogPolling(domain);
                return;
            }

            if (challenge === 'dns') {
                const dnsProviderValue = this.$('#cert-dns-provider-select')?.value || '';
                if (!dnsProviderValue || !dnsProviderValue.startsWith('saved_')) {
                    this.toast?.warning('请先添加 DNS 服务商');
                    return;
                }

                const providerId = dnsProviderValue.replace('saved_', '');

                // 进入日志面板模式
                this.showLogPanel();
                this.clearLogs();
                this.resetStepIndicator();
                this.appendLog(new Date().toLocaleTimeString(), '正在提交 DNS 验证申请...', 'info');

                await this.api.post('/api/certs/dns-prepare', {
                    domain, email,
                    dns_provider: 'saved',
                    dns_provider_id: providerId
                });

                this.appendLog(new Date().toLocaleTimeString(), 'DNS 记录将自动添加，等待验证...', 'info');

                // 开始轮询进度
                this.startLogPolling(domain);
                return;
            }

            if (challenge === 'http-file') {
                // 文件验证
                this.showLogPanel();
                this.clearLogs();
                this.resetStepIndicator();
                this.appendLog(new Date().toLocaleTimeString(), '正在准备验证文件...', 'info');

                const data = await this.api.postJSON('/api/certs/file-prepare', { domain, email });
                if (data) {
                    this.showFileStep(data);
                    // 同步后端进度到步骤指示器
                    this.updateStepIndicator(3, 'active');
                    this.appendLog(new Date().toLocaleTimeString(), '验证文件已准备，请放置文件后点击验证按钮', 'info');
                }
                return;
            }
        } catch (error) {
            this.toast?.error('操作失败: ' + (error.message || '未知错误'));
            this.appendLog(new Date().toLocaleTimeString(), '操作失败: ' + (error.message || '未知错误'), 'error');
        }
    }

    showDNSStep(data) {
        this.setText('#cert-dns-host', data.fqdn || '-');
        this.setText('#cert-dns-value', data.value || '-');

        // 在日志面板内显示 DNS 步骤
        const dnsStep = this.$('#cert-dns-step');
        if (dnsStep) dnsStep.style.display = 'block';

        this._pendingDomain = this.$('#cert-request-domain')?.value.trim();
    }

    async completeDNSChallenge() {
        const domain = this._pendingDomain;
        if (!domain) {
            this.toast?.warning('域名信息丢失，请重新申请');
            return;
        }

        const btn = this.$('#cert-dns-verify-btn');
        if (btn) { btn.disabled = true; btn.textContent = '验证中...'; }

        this.appendLog(new Date().toLocaleTimeString(), '正在验证 DNS 记录...', 'info');

        try {
            await this.api.post('/api/certs/dns-complete', { domain });
            this.appendLog(new Date().toLocaleTimeString(), 'DNS 验证已提交，正在后台执行...', 'info');
            // 立即开始轮询进度（后端异步执行验证）
            this.startLogPolling(domain);
        } catch (error) {
            this.toast?.error('DNS 验证失败: ' + (error.message || '未知错误'));
            this.appendLog(new Date().toLocaleTimeString(), 'DNS 验证失败: ' + (error.message || '未知错误'), 'error');
            if (btn) { btn.disabled = false; btn.textContent = '我已添加 DNS 记录，开始验证'; }
        }
    }

    showFileStep(data) {
        this.setText('#cert-file-path', data.url_path || '-');
        this.setText('#cert-file-content', data.key_auth || '-');

        // 在日志面板内显示文件验证步骤
        const fileStep = this.$('#cert-file-step');
        if (fileStep) fileStep.style.display = 'block';

        this._pendingDomain = this.$('#cert-request-domain')?.value.trim();
    }

    async completeFileChallenge() {
        const domain = this._pendingDomain;
        if (!domain) {
            this.toast?.warning('域名信息丢失，请重新申请');
            return;
        }

        const btn = this.$('#cert-file-verify-btn');
        if (btn) { btn.disabled = true; btn.textContent = '验证中...'; }

        this.appendLog(new Date().toLocaleTimeString(), '正在验证文件...', 'info');

        try {
            await this.api.post('/api/certs/file-complete', { domain });
            this.appendLog(new Date().toLocaleTimeString(), '文件验证已提交，等待结果...', 'info');
            // 开始轮询获取最终结果
            this.startLogPolling(domain);
        } catch (error) {
            this.toast?.error('文件验证失败: ' + (error.message || '未知错误'));
            this.appendLog(new Date().toLocaleTimeString(), '文件验证失败: ' + (error.message || '未知错误'), 'error');
            if (btn) { btn.disabled = false; btn.textContent = '开始验证并获取证书'; }
        }
    }

    // ========== 上传证书 ==========

    showUploadModal() {
        const modal = this.$('#cert-upload-modal');
        if (modal) {
            this.$('#cert-upload-domain') && (this.$('#cert-upload-domain').value = '');
            this.$('#cert-file-input') && (this.$('#cert-file-input').value = '');
            this.$('#key-file-input') && (this.$('#key-file-input').value = '');
            this.setText('#cert-file-text', '点击选择证书文件');
            this.setText('#key-file-text', '点击选择私钥文件');
            this.$('#cert-file-zone')?.classList.remove('file-selected');
            this.$('#key-file-zone')?.classList.remove('file-selected');
            modal.classList.add('active');
        }
    }

    async uploadCert() {
        const domain = this.$('#cert-upload-domain')?.value.trim();
        const certInput = this.$('#cert-file-input');
        const keyInput = this.$('#key-file-input');

        if (!domain) {
            this.toast?.warning('请输入域名');
            return;
        }
        if (!certInput?.files?.length) {
            this.toast?.warning('请选择证书文件');
            return;
        }
        if (!keyInput?.files?.length) {
            this.toast?.warning('请选择私钥文件');
            return;
        }

        const formData = new FormData();
        formData.append('domain', domain);
        formData.append('cert_file', certInput.files[0]);
        formData.append('key_file', keyInput.files[0]);

        try {
            await this.api.upload('/api/certs/upload', formData);
            this.toast?.success('证书上传成功');
            this.hideModal('cert-upload-modal');
            await this.loadCerts();
        } catch (error) {
            this.toast?.error('上传失败: ' + (error.message || '未知错误'));
        }
    }

    // ========== 续期 / 删除 ==========

    async renewCert(domain) {
        try {
            await this.api.post('/api/certs/renew', { domain });
            this.toast?.success('续期已触发');
            await this.loadCerts();
        } catch (error) {
            this.toast?.error('续期失败: ' + (error.message || '未知错误'));
        }
    }

    async deleteCert(domain) {
        if (!confirm(`确定删除证书 ${domain}？`)) return;

        try {
            await this.api.post('/api/certs/delete', { domain });
            this.toast?.success('证书已删除');
            await this.loadCerts();
        } catch (error) {
            this.toast?.error('删除失败: ' + (error.message || '未知错误'));
        }
    }

    // ========== 部署到站点 ==========

    async showDeployModal(domain) {
        this._deployDomain = domain;
        this.setText('#cert-deploy-domain-text', domain);

        // 加载站点列表
        const listEl = this.$('#cert-deploy-site-list');
        if (listEl) {
            listEl.innerHTML = '<div style="padding:var(--space-lg) 0;text-align:center;color:var(--text-muted)">加载中...</div>';
        }

        try {
            const sites = await this.api.getJSON('/api/sites') || [];
            this._deploySites = sites;

            if (sites.length === 0) {
                if (listEl) {
                    listEl.innerHTML = '<div style="padding:var(--space-lg) 0;text-align:center;color:var(--text-muted)">暂无站点，请先创建站点</div>';
                }
            } else {
                // 筛选域名匹配的站点用于预选
                const certDomain = domain.replace(/^\*\./, '');
                listEl.innerHTML = sites.map(site => {
                    const domains = site.domain || [];
                    const match = domains.some(d => {
                        const sd = d.replace(/^\*\./, '');
                        return sd === certDomain || certDomain.endsWith('.' + sd) || sd.endsWith('.' + certDomain);
                    });
                    const domainStr = domains.length > 0 ? domains.join(', ') : '-';
                    const checked = match ? 'checked' : '';
                    const sslBadge = site.ssl?.enabled
                        ? '<span style="font-size:11px;color:var(--success);margin-left:4px">SSL</span>'
                        : '';
                    return `
                        <label style="display:flex;align-items:center;gap:8px;padding:8px 12px;border-bottom:1px solid var(--border);cursor:pointer;">
                            <input type="checkbox" class="cert-deploy-site-check" value="${escapeHtml(site.id)}" ${checked}>
                            <span style="flex:1;font-size:13px">${escapeHtml(site.name)}${sslBadge}</span>
                            <span style="font-size:12px;color:var(--text-muted)">${escapeHtml(domainStr)}</span>
                        </label>
                    `;
                }).join('');
            }
        } catch (error) {
            if (listEl) {
                listEl.innerHTML = '<div style="padding:var(--space-lg) 0;text-align:center;color:var(--danger)">加载站点失败</div>';
            }
        }

        this.$('#cert-deploy-modal')?.classList.add('active');
    }

    async deployCertToSites() {
        const domain = this._deployDomain;
        if (!domain) return;

        const checks = this.$$('.cert-deploy-site-check:checked');
        const siteIds = Array.from(checks).map(el => el.value);
        if (siteIds.length === 0) {
            this.toast?.warning('请选择至少一个站点');
            return;
        }

        const confirmBtn = this.$('#cert-deploy-confirm');
        if (confirmBtn) { confirmBtn.disabled = true; confirmBtn.textContent = '部署中...'; }

        try {
            const data = await this.api.postJSON('/api/certs/deploy', { domain, site_ids: siteIds });
            this.toast?.success(`证书已部署到 ${data?.deployed || siteIds.length} 个站点`);
            this.hideModal('cert-deploy-modal');
            this.api.clearCache('/api/sites');
            await this.loadCerts();
        } catch (error) {
            this.toast?.error('部署失败: ' + (error.message || '未知错误'));
        } finally {
            if (confirmBtn) { confirmBtn.disabled = false; confirmBtn.textContent = '部署'; }
        }
    }

    // ========== 标签页切换 ==========

    switchCertTab(tabName) {
        this.$$('.cert-tabs .tab-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.certTab === tabName);
        });
        this.$$('.cert-tab-pane').forEach(pane => {
            pane.classList.toggle('active', pane.id === `cert-tab-${tabName}`);
        });
    }

    // ========== 模态框测试按钮 ==========

    async testDNSProviderModal() {
        const editId = this.$('#dns-provider-edit-id')?.value;
        if (editId) {
            // 编辑模式：使用已有的 test API（后端有完整凭证）
            await this.testDNSProvider(editId);
            return;
        }

        // 新建模式：直接测试凭证
        const type = this.$('#dns-provider-type')?.value;
        const credentials = this.getDNSModalCredentials(type);
        const hasCreds = Object.values(credentials).some(v => v.trim() !== '');
        if (!hasCreds) {
            this.toast?.warning('请先填写凭证信息');
            return;
        }

        const btn = this.$('#dns-provider-modal-test');
        if (btn) { btn.disabled = true; btn.textContent = '测试中...'; }

        try {
            const data = await this.api.postJSON('/api/certs/dns-providers-test', {
                type, credentials
            });
            if (data && data.success) {
                this.toast?.success(data.message || '连接测试成功');
            } else {
                this.toast?.error(data?.message || '连接测试失败');
            }
        } catch (error) {
            this.toast?.error('测试失败: ' + (error.message || '未知错误'));
        } finally {
            if (btn) { btn.disabled = false; btn.textContent = '测试连接'; }
        }
    }

    // ========== 清理 ==========

    onDestroy() {
        this.stopLogPolling();
    }
}

// 单例
let instance = null;

export function initCertTab(deps) {
    if (!instance) {
        instance = new CertTab(deps);
        instance.init();
    }
    return instance;
}

export default CertTab;
