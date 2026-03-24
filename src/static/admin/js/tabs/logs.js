/**
 * 日志查看模块
 *
 * 负责访问日志和错误日志的查看、刷新、清空
 */

import { globalEvents } from '../core/events.js';

/**
 * 日志分类配置
 */
const LOG_TYPES = {
    http: ['access', 'error'],
    ftp: ['access', 'error'],
    panel: ['access', 'api', 'auth']
};

const LOG_TYPE_LABELS = {
    'http-access': 'HTTP 访问',
    'http-error': 'HTTP 错误',
    'ftp-access': 'FTP 操作',
    'ftp-error': 'FTP 错误',
    'panel-access': '面板访问',
    'panel-api': 'API 调用',
    'panel-auth': '认证日志'
};

/**
 * 初始化日志面板
 * @param {Object} dependencies - 依赖注入 { api, toast }
 */
export function initLogsTab({ api, toast }) {
    console.log('📋 初始化日志面板...');

    // 当前日志分类和类型
    let currentCategory = 'http';
    let currentType = 'access';
    let autoScroll = true;
    let refreshTimer = null;

    // 绑定事件
    bindEvents();

    // 监听标签页切换到日志
    globalEvents.match('tab:switch:*', (_event, data) => {
        if (data && data.tabName === 'logs') {
            loadLogs();
            startAutoRefresh();
        }
    });

    /**
     * 绑定事件监听器
     */
    function bindEvents() {
        // 日志分类切换
        const logCategorySelect = document.getElementById('log-category');
        logCategorySelect?.addEventListener('change', (e) => {
            currentCategory = e.target.value;
            updateLogTypeOptions();
            loadLogs();
        });

        // 日志类型切换
        const logTypeSelect = document.getElementById('log-type');
        logTypeSelect?.addEventListener('change', (e) => {
            currentType = e.target.value;
            loadLogs();
        });

        // 刷新按钮
        const refreshBtn = document.getElementById('refresh-logs');
        refreshBtn?.addEventListener('click', () => {
            loadLogs();
        });

        // 清空按钮
        const clearBtn = document.getElementById('clear-logs');
        clearBtn?.addEventListener('click', async () => {
            const label = LOG_TYPE_LABELS[`${currentCategory}-${currentType}`] || `${currentCategory}-${currentType}`;
            if (confirm(`确定要清空${label}日志吗？`)) {
                await clearLogs();
            }
        });

        // 自动滚动
        const autoScrollCheckbox = document.getElementById('auto-scroll');
        autoScrollCheckbox?.addEventListener('change', (e) => {
            autoScroll = e.target.checked;
        });

        // 初始化日志类型选项
        updateLogTypeOptions();
    }

    /**
     * 更新日志类型选项
     */
    function updateLogTypeOptions() {
        const logTypeSelect = document.getElementById('log-type');
        if (!logTypeSelect) return;

        // 清空现有选项
        logTypeSelect.innerHTML = '';

        // 获取当前分类的日志类型
        const types = LOG_TYPES[currentCategory] || ['access'];

        // 添加新选项
        types.forEach(type => {
            const option = document.createElement('option');
            option.value = type;
            option.textContent = LOG_TYPE_LABELS[`${currentCategory}-${type}`] || type;
            logTypeSelect.appendChild(option);
        });

        // 设置默认选中第一个
        currentType = types[0];
        logTypeSelect.value = currentType;
    }

    /**
     * 加载日志
     */
    async function loadLogs() {
        const logViewer = document.getElementById('log-viewer');
        if (logViewer) {
            logViewer.textContent = '加载中...';
        }

        try {
            const response = await api.get(`/api/logs?category=${currentCategory}&type=${currentType}&lines=100`);
            if (response && response.ok) {
                const data = await api.parseJSON(response);
                // 处理不同的响应格式：数组用换行连接，字符串直接使用
                let logsText = '';
                if (Array.isArray(data.logs)) {
                    logsText = data.logs.join('\n');
                } else {
                    logsText = data.logs || data.log || data || '';
                }
                renderLogs(String(logsText));
            } else {
                toast.error('加载日志失败');
                if (logViewer) {
                    logViewer.textContent = '加载失败';
                }
            }
        } catch (error) {
            console.error('加载日志失败:', error);
            toast.error('加载日志失败: ' + error.message);
            if (logViewer) {
                logViewer.textContent = '加载失败: ' + error.message;
            }
        }
    }

    /**
     * 渲染日志内容
     * @param {string} logs - 日志内容
     */
    function renderLogs(logs) {
        const logViewer = document.getElementById('log-viewer');
        if (!logViewer) return;

        if (!logs || logs.trim() === '') {
            logViewer.textContent = '(空)';
            return;
        }

        logViewer.textContent = logs;

        // 自动滚动到底部
        if (autoScroll) {
            logViewer.scrollTop = logViewer.scrollHeight;
        }
    }

    /**
     * 清空日志
     */
    async function clearLogs() {
        try {
            const response = await api.post('/api/logs/clear', { category: currentCategory, type: currentType });
            if (response) {
                const data = await api.parseJSON(response);
                if (data && data.message) {
                    toast.success(data.message);
                } else {
                    toast.success('日志已清空');
                }
                loadLogs();
            }
        } catch (error) {
            console.error('清空日志失败:', error);
            toast.error('清空日志失败: ' + error.message);
        }
    }

    /**
     * 开始自动刷新
     */
    function startAutoRefresh() {
        stopAutoRefresh();
        refreshTimer = setInterval(() => {
            loadLogs();
        }, 5000); // 5秒刷新一次
    }

    /**
     * 停止自动刷新
     */
    function stopAutoRefresh() {
        if (refreshTimer) {
            clearInterval(refreshTimer);
            refreshTimer = null;
        }
    }

    // 页面卸载时清理
    window.addEventListener('beforeunload', () => {
        stopAutoRefresh();
    });
}
