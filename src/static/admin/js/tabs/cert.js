/**
 * 证书管理模块
 *
 * 负责 SSL 证书的管理、申请、续签
 */

import { BaseTab } from './BaseTab.js';

class CertTab extends BaseTab {
    constructor(deps) {
        super(deps, 'cert');
    }

    onInit() {
        console.log('🔐 初始化证书标签页...');
        this.bindEvents();
    }

    bindEvents() {
        // 添加证书按钮
        const addBtn = document.getElementById('add-cert-btn');
        addBtn?.addEventListener('click', () => {
            this.toast.info('证书申请功能开发中...');
        });
    }

    async onLoad() {
        const certList = document.getElementById('cert-list');
        if (!certList) return;

        // TODO: 调用 API 获取证书列表
        // const response = await this.api.get('/api/certs');
        // const certs = await this.api.parseJSON(response);

        // 暂时显示默认证书
        certList.innerHTML = `
            <div class="cert-item">
                <div class="cert-info">
                    <h4>默认证书</h4>
                    <p class="cert-domain">localhost</p>
                    <p class="cert-expire">自签名证书</p>
                </div>
                <div class="cert-actions">
                    <span class="badge badge-warning">自签名</span>
                </div>
            </div>
        `;
    }
}

// 单例
let instance = null;

/**
 * 初始化证书标签页
 */
export function initCertTab(deps) {
    if (!instance) {
        instance = new CertTab(deps);
        instance.init();
    }
    return instance;
}