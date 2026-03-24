/**
 * 服务控制模块
 *
 * 负责 HTTP 和 FTP 服务的启动、停止、重启
 */

import { globalEvents } from '../core/events.js';

/**
 * 初始化服务面板
 * @param {Object} dependencies - 依赖注入 { api, toast }
 */
export function initServicesTab({ api, toast }) {
    console.log('🔌 初始化服务面板...');

    // 绑定按钮事件
    bindEvents();

    // 监听服务状态更新
    globalEvents.on('status:loaded', (data) => {
        updateServicesUI(data);
    });

    // 监听标签页切换
    globalEvents.on('tab:switch:services', () => {
        loadServicesStatus();
    });

    // 监听标签页切换（通配符）
    globalEvents.match('tab:switch:*', (_event, data) => {
        if (data && data.tabName === 'services') {
            loadServicesStatus();
        }
    });

    /**
     * 绑定事件监听器
     */
    function bindEvents() {
        // FTP 重启
        const restartBtn = document.getElementById('ftp-restart-btn');
        if (restartBtn) {
            restartBtn.addEventListener('click', () => controlService('ftp', 'restart'));
        }

        // FTP 停止
        const stopBtn = document.getElementById('ftp-stop-btn');
        if (stopBtn) {
            stopBtn.addEventListener('click', () => controlService('ftp', 'stop'));
        }

        // FTP 启动
        const startBtn = document.getElementById('ftp-start-btn');
        if (startBtn) {
            startBtn.addEventListener('click', () => controlService('ftp', 'start'));
        }
    }

    /**
     * 加载服务状态
     */
    async function loadServicesStatus() {
        try {
            const response = await api.get('/api/status');
            if (response && response.ok) {
                const data = await api.parseJSON(response);
                updateServicesUI(data);
            }
        } catch (error) {
            console.error('加载服务状态失败:', error);
        }
    }

    /**
     * 更新服务 UI
     * @param {Object} data - 状态数据（扁平结构）
     */
    function updateServicesUI(data) {
        if (!data) return;

        // HTTP 服务
        const httpPortEl = document.getElementById('http-service-port');
        const httpStatusEl = document.getElementById('http-service-status');
        const httpRootEl = document.getElementById('http-service-root');

        if (httpPortEl && data.http_port) {
            httpPortEl.textContent = data.http_port;
        }
        if (httpStatusEl && data.http_running !== undefined) {
            updateStatusBadge(httpStatusEl, data.http_running);
        }
        if (httpRootEl && data.http_root) {
            httpRootEl.textContent = data.http_root;
        }

        // FTP 服务
        const ftpPortEl = document.getElementById('ftp-service-port');
        const ftpStatusEl = document.getElementById('ftp-service-status');
        const ftpStatusBadge = document.getElementById('ftp-service-status-badge');
        const ftpRootEl = document.getElementById('ftp-service-root');

        if (ftpPortEl && data.ftp_port) {
            ftpPortEl.textContent = data.ftp_port;
        }
        if (ftpStatusEl && data.ftp_running !== undefined) {
            updateStatusBadge(ftpStatusEl, data.ftp_running);
            updateControlButtons(data.ftp_running);
        }
        if (ftpStatusBadge && data.ftp_running !== undefined) {
            updateStatusBadge(ftpStatusBadge, data.ftp_running);
        }
        if (ftpRootEl && data.ftp_root) {
            ftpRootEl.textContent = data.ftp_root;
        }
    }

    /**
     * 更新状态徽章
     * @param {HTMLElement} element - 徽章元素
     * @param {boolean} isRunning - 是否运行中
     */
    function updateStatusBadge(element, isRunning) {
        element.textContent = isRunning ? '运行中' : '已停止';
        element.className = isRunning ? 'badge badge-success' : 'badge badge-danger';
    }

    /**
     * 更新控制按钮显示状态
     * @param {boolean} isRunning - 是否运行中
     */
    function updateControlButtons(isRunning) {
        const stopBtn = document.getElementById('ftp-stop-btn');
        const startBtn = document.getElementById('ftp-start-btn');

        if (isRunning) {
            if (stopBtn) stopBtn.style.removeProperty('display');
            if (startBtn) startBtn.style.display = 'none';
        } else {
            if (stopBtn) stopBtn.style.display = 'none';
            if (startBtn) startBtn.style.removeProperty('display');
        }
    }

    /**
     * 控制服务
     * @param {string} service - 服务名称 ('http' | 'ftp')
     * @param {string} action - 操作 ('start' | 'stop' | 'restart')
     */
    async function controlService(service, action) {
        const actionNames = {
            start: '启动',
            stop: '停止',
            restart: '重启'
        };

        try {
            const response = await api.post(`/api/service/${service}/${action}`);
            if (response) {
                // 统一响应格式处理
                const data = await api.parseJSON(response);
                if (data && data.message) {
                    toast.success(data.message);
                } else {
                    toast.success(`${service.toUpperCase()} 服务${actionNames[action]}成功`);
                }
                await loadServicesStatus();
            }
        } catch (error) {
            console.error(`服务${action}失败:`, error);
            toast.error(`${service.toUpperCase()} 服务${actionNames[action]}失败: ${error.message}`);
        }
    }

    // 初始加载
    loadServicesStatus();
}
