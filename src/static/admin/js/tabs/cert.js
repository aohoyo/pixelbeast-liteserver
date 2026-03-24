/**
 * 证书管理模块
 *
 * 负责 SSL 证书的管理、申请、续签
 */

import { globalEvents } from '../core/events.js';

// 存储依赖以便在闭包函数中使用
let deps = null;

/**
 * 初始化证书标签页
 * @param {Object} dependencies - 依赖注入 { api, toast }
 */
export function initCertTab({ api, toast }) {
    console.log('🔐 初始化证书标签页...');

    // 保存依赖
    deps = { api, toast };

    // 绑定事件
    bindEvents();

    // 监听标签页切换
    globalEvents.match('tab:switch:cert', () => {
        loadCerts();
    });
}

/**
 * 绑定事件监听器
 */
function bindEvents() {
    if (!deps) return;
    const { toast } = deps;

    // 添加证书按钮
    const addBtn = document.getElementById('add-cert-btn');
    addBtn?.addEventListener('click', () => {
        toast.info('证书申请功能开发中...');
    });
}

/**
 * 加载证书列表
 */
async function loadCerts() {
    if (!deps) return;
    const { toast } = deps;

    const certList = document.getElementById('cert-list');
    if (!certList) return;

    try {
        // TODO: 调用 API 获取证书列表
        // const response = await api.get('/api/certs');
        // const certs = await api.parseJSON(response);

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
    } catch (error) {
        console.error('加载证书失败:', error);
        toast.error('加载证书失败');
    }
}
