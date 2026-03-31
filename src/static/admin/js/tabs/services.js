/**
 * 服务控制模块
 *
 * 负责 HTTP 和 FTP 服务的启动、停止、重启
 */

import { BaseTab } from './BaseTab.js';

class ServicesTab extends BaseTab {
    constructor(deps) {
        super(deps, 'services');
    }

    onInit() {
        console.log('🔌 初始化服务面板...');
        this.bindEvents();
        
        // 监听服务状态更新
            this.events.on('status:loaded', (data) => {
            this.updateUI(data);
        });
        // FTP 稡块单独触发状态刷新
        this.events.on('ftp:status:loaded', (data) => {
            this.updateUI(data);
        });
    }

    bindEvents() {
        // FTP 控制
        this.$('#ftp-restart-btn')?.addEventListener('click', () => this.control('ftp', 'restart'));
        this.$('#ftp-stop-btn')?.addEventListener('click', () => this.control('ftp', 'stop'));
        this.$('#ftp-start-btn')?.addEventListener('click', () => this.control('ftp', 'start'));
    }

    async onLoad() {
        const response = await this.api.get('/api/system/status');
        if (response?.ok) {
            const data = await this.api.parseJSON(response);
            if (data) {
                this.updateUI(data);
                globalEvents.emit('status:loaded', data);
            }
        }
        // FTP 稡块独立获取状态
        const ftpResponse = await this.api.get('/api/ftp/status');
        if (ftpResponse?.ok) {
            const ftpData = await this.api.parseJSON(ftpResponse);
            if (ftpData) {
                this.updateUI({ ftp_running: ftpData.running, ftp_port: ftpData.port });
            }
        }
    }

    /**
     * 更新 UI
     */
    updateUI(data) {
        if (!data) return;

        // FTP 服务
        this.setText('#ftp-service-port', data.ftp_port);
        this.setText('#ftp-service-root', data.directories?.ftp || './ftp');
        this.setStatusBadge('#ftp-service-status', data.ftp_running);
        this.setStatusBadge('#ftp-service-status-badge', data.ftp_running);
        this.updateControlButtons(data.ftp_running);
    }

    /**
     * 设置文本
     */
    setText(selector, value) {
        const el = this.$(selector);
        if (el && value !== undefined) {
            el.textContent = value;
        }
    }

    /**
     * 更新状态徽章
     */
    setStatusBadge(selector, isRunning) {
        const el = this.$(selector);
        if (el && isRunning !== undefined) {
            el.textContent = isRunning ? '运行中' : '已停止';
            el.className = isRunning ? 'badge badge-success' : 'badge badge-danger';
        }
    }

    /**
     * 更新控制按钮
     */
    updateControlButtons(isRunning) {
        const stopBtn = this.$('#ftp-stop-btn');
        const startBtn = this.$('#ftp-start-btn');

        if (stopBtn) stopBtn.style.display = isRunning ? '' : 'none';
        if (startBtn) startBtn.style.display = isRunning ? 'none' : '';
    }

    /**
     * 控制服务
     */
    async control(service, action) {
        const names = { start: '启动', stop: '停止', restart: '重启' };

        try {
            const response = await this.api.post(`/api/service/${service}/${action}`);
            if (response) {
                const data = await this.api.parseJSON(response);
                this.toast.success(data?.message || `${service.toUpperCase()} 服务${names[action]}成功`);
                await this.refresh();
            }
        } catch (error) {
            this.toast.error(`${service.toUpperCase()} 服务${names[action]}失败: ${error.message}`);
        }
    }
}

// 单例
let instance = null;

export function initServicesTab(deps) {
    if (!instance) {
        instance = new ServicesTab(deps);
        instance.init();
    }
    return instance;
}