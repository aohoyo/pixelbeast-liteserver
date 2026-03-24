/**
 * 首页模块
 *
 * 负责显示系统状态：内存、协程数、运行时间等
 */

import { globalEvents } from '../core/events.js';

// 内存最大值（用于进度条）
const MAX_MEMORY_MB = 1024; // 1GB
const MAX_GOROUTINES = 500; // 协程参考值

/**
 * 格式化运行时间（毫秒转换为时分秒）
 * @param {number} ms - 毫秒
 * @returns {object} 包含天、时、分、秒的对象
 */
function formatUptime(ms) {
    if (typeof ms !== 'number' || isNaN(ms)) {
        return { days: 0, hours: 0, minutes: 0, seconds: 0, startTime: '--' };
    }

    const seconds = Math.floor(ms / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    return {
        days: days,
        hours: hours % 24,
        minutes: minutes % 60,
        seconds: seconds % 60,
        // 计算启动时间
        startTime: new Date(Date.now() - ms).toLocaleString('zh-CN', {
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit'
        })
    };
}

/**
 * 数字滚动动画
 * @param {HTMLElement} element - 目标元素
 * @param {number} target - 目标值
 * @param {number} duration - 动画时长（毫秒）
 */
function animateNumber(element, target, duration = 600) {
    if (!element) return;

    const start = 0;
    const startTime = performance.now();
    const isFloat = target % 1 !== 0;

    function update(currentTime) {
        const elapsed = currentTime - startTime;
        const progress = Math.min(elapsed / duration, 1);
        const easeProgress = 1 - Math.pow(1 - progress, 3); // ease-out

        const current = start + (target - start) * easeProgress;

        if (isFloat) {
            element.textContent = current.toFixed(1);
        } else {
            element.textContent = Math.floor(current);
        }

        if (progress < 1) {
            requestAnimationFrame(update);
        } else {
            element.textContent = isFloat ? target.toFixed(1) : target;
        }
    }

    requestAnimationFrame(update);
}

/**
 * 初始化状态面板
 * @param {Object} dependencies - 依赖注入 { state, api, toast }
 */
export function initHomeTab({ state, api, toast }) {
    console.log('📊 初始化状态面板...');

    // 监听状态加载事件
    globalEvents.on('status:loaded', (data) => {
        updateStatusUI(data);
    });

    // 监听标签页切换
    globalEvents.on('tab:switch:status', () => {
        loadStatusData();
    });

    /**
     * 加载状态数据
     */
    async function loadStatusData() {
        try {
            const response = await api.get('/api/status');
            if (response && response.ok) {
                const data = await api.parseJSON(response);
                updateStatusUI(data);
            }
        } catch (error) {
            console.error('加载状态失败:', error);
        }
    }

    /**
     * 更新状态 UI
     * @param {Object} data - 状态数据
     */
    function updateStatusUI(data) {
        // 进程内存
        const memoryValueEl = document.getElementById('memory-value');
        const memoryFillEl = document.getElementById('memory-fill');
        const memoryPercentEl = document.getElementById('memory-percent');
        if (memoryValueEl && data.memory_mb !== undefined) {
            const memValue = parseFloat(data.memory_mb);
            animateNumber(memoryValueEl, memValue, 600);

            // 更新进度条
            if (memoryFillEl) {
                const percent = Math.min((memValue / MAX_MEMORY_MB) * 100, 100);
                memoryFillEl.style.width = `${percent}%`;
            }

            // 更新百分比
            if (memoryPercentEl) {
                const percent = ((memValue / MAX_MEMORY_MB) * 100).toFixed(1);
                memoryPercentEl.textContent = `${percent}%`;
            }
        }

        // 协程数量
        const goroutineValueEl = document.getElementById('goroutine-value');
        const goroutineFillEl = document.getElementById('goroutine-fill');
        const goroutineStatusEl = document.getElementById('goroutine-status');
        if (goroutineValueEl && data.goroutines !== undefined) {
            animateNumber(goroutineValueEl, data.goroutines, 500);

            // 更新进度条
            if (goroutineFillEl) {
                const percent = Math.min((data.goroutines / MAX_GOROUTINES) * 100, 100);
                goroutineFillEl.style.width = `${percent}%`;
            }

            // 更新状态
            if (goroutineStatusEl) {
                goroutineStatusEl.className = 'metric-status';
                if (data.goroutines < 300) {
                    goroutineStatusEl.textContent = '正常';
                    goroutineStatusEl.classList.add('good');
                } else if (data.goroutines < 500) {
                    goroutineStatusEl.textContent = '较高';
                    goroutineStatusEl.classList.add('warning');
                } else {
                    goroutineStatusEl.textContent = '过高';
                    goroutineStatusEl.classList.add('danger');
                }
            }
        }

        // 运行时间
        if (data.uptime !== undefined) {
            updateUptimeDisplay(data.uptime);
        }

        // 系统信息
        const sysinfoEl = document.getElementById('sysinfo-value');
        if (sysinfoEl) {
            const sysInfo = [
                data.os || '',
                data.arch || ''
            ].filter(Boolean).join(' / ');
            sysinfoEl.textContent = sysInfo || '--';
        }

        // 服务状态
        updateServiceStatus(data);
    }

    /**
     * 更新运行时间显示
     * @param {number} ms - 运行时间（毫秒）
     */
    function updateUptimeDisplay(ms) {
        const uptime = formatUptime(ms);

        // 更新各段数值
        const segments = ['days', 'hours', 'minutes', 'seconds'];
        segments.forEach(segment => {
            const el = document.getElementById(`uptime-${segment}`);
            if (el) {
                const valueEl = el.querySelector('.segment-value');
                if (valueEl) {
                    animateNumber(valueEl, uptime[segment], 400);
                }
            }
        });

        // 更新启动时间
        const startTimeEl = document.getElementById('uptime-start-time');
        if (startTimeEl) {
            startTimeEl.textContent = uptime.startTime;
        }
    }

    /**
     * 更新服务状态
     * @param {Object} data - 完整状态数据
     */
    function updateServiceStatus(data) {
        // HTTP 服务状态
        const httpPortEl = document.getElementById('http-port-display');
        const httpCard = document.getElementById('http-service-card');
        const httpStatusText = document.getElementById('http-status-text');

        if (httpPortEl && data.http_port !== undefined) {
            httpPortEl.textContent = data.http_port;
        }
        if (httpCard && data.http_running !== undefined) {
            httpCard.className = 'service-status-card ' + (data.http_running ? 'running' : 'stopped');
        }
        if (httpStatusText && data.http_running !== undefined) {
            httpStatusText.textContent = data.http_running ? '运行中' : '已停止';
        }

        // FTP 服务状态
        const ftpPortEl = document.getElementById('ftp-port-display');
        const ftpCard = document.getElementById('ftp-service-card');
        const ftpStatusText = document.getElementById('ftp-status-text');

        if (ftpPortEl && data.ftp_port !== undefined) {
            ftpPortEl.textContent = data.ftp_port;
        }
        if (ftpCard && data.ftp_running !== undefined) {
            ftpCard.className = 'service-status-card ' + (data.ftp_running ? 'running' : 'stopped');
        }
        if (ftpStatusText && data.ftp_running !== undefined) {
            ftpStatusText.textContent = data.ftp_running ? '运行中' : '已停止';
        }
    }

    // 刷新按钮
    const refreshBtn = document.getElementById('refresh-status');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', () => {
            loadStatusData();
        });
    }

    // 初始加载
    loadStatusData();
}
