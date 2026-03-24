/**
 * FTP 服务管理模块
 *
 * 负责 FTP 服务器的启停控制、用户管理
 */

import { globalEvents } from '../core/events.js';

// 存储依赖以便在闭包函数中使用
let deps = null;

/**
 * 初始化 FTP 标签页
 * @param {Object} dependencies - 依赖注入 { state, api, toast }
 */
export function initFtpTab({ state, api, toast }) {
    console.log('🔌 初始化 FTP 服务标签页...');

    // 保存依赖
    deps = { state, api, toast };

    // 绑定事件
    bindEvents();

    // 监听标签页切换
    globalEvents.match('tab:switch:ftp', () => {
        loadFtpStatus();
    });

    // 监听状态更新
    globalEvents.match('status:loaded', (event, data) => {
        updateFtpStatus(data);
    });
}

/**
 * 绑定事件监听器
 */
function bindEvents() {
    if (!deps) return;
    const { api, toast } = deps;

    // FTP 启动按钮
    const startBtn = document.getElementById('ftp-start-btn');
    startBtn?.addEventListener('click', async () => {
        try {
            const response = await api.post('/api/service/ftp/start');
            const data = await api.parseJSON(response);
            if (data.message) {
                toast.success(data.message);
            } else {
                toast.success('FTP 服务已启动');
            }
            loadFtpStatus();
        } catch (error) {
            toast.error('启动失败: ' + error.message);
        }
    });

    // FTP 停止按钮
    const stopBtn = document.getElementById('ftp-stop-btn');
    stopBtn?.addEventListener('click', async () => {
        try {
            const response = await api.post('/api/service/ftp/stop');
            const data = await api.parseJSON(response);
            if (data.message) {
                toast.success(data.message);
            } else {
                toast.success('FTP 服务已停止');
            }
            loadFtpStatus();
        } catch (error) {
            toast.error('停止失败: ' + error.message);
        }
    });

    // FTP 重启按钮
    const restartBtn = document.getElementById('ftp-restart-btn');
    restartBtn?.addEventListener('click', async () => {
        try {
            const response = await api.post('/api/service/ftp/restart');
            const data = await api.parseJSON(response);
            if (data.message) {
                toast.success(data.message);
            } else {
                toast.success('FTP 服务已重启');
            }
            loadFtpStatus();
        } catch (error) {
            toast.error('重启失败: ' + error.message);
        }
    });

    // 添加用户按钮
    const addUserBtn = document.getElementById('add-ftp-user-btn');
    addUserBtn?.addEventListener('click', () => {
        toast.info('添加用户功能开发中...');
    });
}

/**
 * 加载 FTP 状态
 */
async function loadFtpStatus() {
    if (!deps) return;
    const { api } = deps;

    try {
        const response = await api.get('/api/status');
        const data = await api.parseJSON(response);
        if (data && data.ftp_running !== undefined) {
            updateFtpStatus(data);
        }
    } catch (error) {
        console.error('加载 FTP 状态失败:', error);
    }
}

/**
 * 更新 FTP 状态显示
 * @param {Object} data - 状态数据
 */
function updateFtpStatus(data) {
    const statusBadge = document.getElementById('ftp-service-status-badge');
    const statusText = document.getElementById('ftp-service-status-text');
    const portDisplay = document.getElementById('ftp-service-port');
    const rootDisplay = document.getElementById('ftp-service-root');

    if (!statusBadge && !statusText) return;

    const isRunning = data.ftp_running;

    // 更新徽章
    if (isRunning) {
        statusBadge.className = 'badge badge-success';
        statusBadge.textContent = '运行中';
    } else {
        statusBadge.className = 'badge badge-danger';
        statusBadge.textContent = '已停止';
    }

    // 更新文本
    if (statusText) {
        statusText.textContent = isRunning ? '运行中' : '已停止';
    }

    // 更新端口和根目录
    if (portDisplay && data.ftp_port) {
        portDisplay.textContent = data.ftp_port;
    }
    if (rootDisplay && data.ftp_root) {
        rootDisplay.textContent = data.ftp_root;
    }
}
