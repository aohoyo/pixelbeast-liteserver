/**
 * 首页模块 - 系统监控仪表盘（宝塔风格）
 *
 * 显示系统状态：CPU、内存、硬盘、负载
 * 卡片布局：左文右图，鼠标悬停显示详情
 */

import { globalEvents } from '../core/events.js';

// 配置
const REFRESH_INTERVAL = 5000;
const SYNC_INTERVAL = 60000;
let serverStartTime = null;
let uptimeTimer = null;
let syncTimer = null;
let refreshTimer = null;

// 缓存数据用于弹窗显示
let currentData = {};

/**
 * 格式化运行时间
 */
function formatUptime(ms) {
    if (typeof ms !== 'number' || isNaN(ms)) return '--';

    const seconds = Math.floor(ms / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days}天${hours % 24}时${minutes % 60}分`;
    if (hours > 0) return `${hours}时${minutes % 60}分`;
    if (minutes > 0) return `${minutes}分${seconds % 60}秒`;
    return `${seconds}秒`;
}

/**
 * 格式化存储大小
 */
function formatStorage(gb) {
    if (gb >= 1000) return `${(gb / 1000).toFixed(1)}TB`;
    if (gb < 1) return `${(gb * 1024).toFixed(0)}MB`;
    return `${gb.toFixed(1)}GB`;
}

/**
 * 数字动画
 */
function animateValue(element, target, suffix = '', duration = 600) {
    if (!element) return;
    const start = parseFloat(element.textContent) || 0;
    const startTime = performance.now();

    function update(currentTime) {
        const elapsed = currentTime - startTime;
        const progress = Math.min(elapsed / duration, 1);
        const easeProgress = 1 - Math.pow(1 - progress, 3);
        const current = start + (target - start) * easeProgress;

        element.textContent = (target % 1 === 0 ? Math.floor(current) : current.toFixed(1)) + suffix;

        if (progress < 1) requestAnimationFrame(update);
        else element.textContent = (target % 1 === 0 ? target : target.toFixed(1)) + suffix;
    }
    requestAnimationFrame(update);
}

/**
 * 更新环形图
 */
function updateRingChart(id, percent) {
    const fill = document.getElementById(`${id}-ring-fill`);
    const text = document.getElementById(`${id}-percent`);

    if (fill) {
        const circumference = 2 * Math.PI * 40;
        const offset = circumference - (percent / 100) * circumference;
        fill.style.strokeDashoffset = offset;

        fill.classList.remove('warning', 'danger');
        if (percent > 80) fill.classList.add('danger');
        else if (percent > 50) fill.classList.add('warning');
    }

    if (text) animateValue(text, Math.round(percent));
}

/**
 * 计算负载百分比和状态
 */
function getLoadStatus(load1m, cpuCores) {
    const percent = Math.round((load1m / cpuCores) * 100);

    if (percent < 50) {
        return { percent, text: '运行流畅', class: 'good' };
    } else if (percent < 80) {
        return { percent, text: `运行达到${percent}%`, class: 'warning' };
    } else {
        return { percent, text: '负载较高', class: 'danger' };
    }
}

/**
 * 更新负载显示
 */
function updateLoadDisplay(data) {
    const load1m = data.load_avg?.[0] || 0;
    const cpuCores = data.cpu_cores || 4;
    const status = getLoadStatus(load1m, cpuCores);

    // 更新环形图
    updateRingChart('load', status.percent);

    // 更新状态文字
    const statusEl = document.getElementById('load-status-text');
    if (statusEl) {
        statusEl.textContent = status.text;
        statusEl.className = 'metric-status ' + status.class;
    }

    // 更新弹窗内容
    const tooltipStatus = document.getElementById('tooltip-load-status');
    if (tooltipStatus) {
        tooltipStatus.textContent = status.text;
        tooltipStatus.className = 'tooltip-status ' + status.class;
    }

    const loadAvgEl = document.getElementById('tooltip-load-avg');
    if (loadAvgEl && data.load_avg) {
        loadAvgEl.textContent = `${data.load_avg[0]?.toFixed(2)} / ${data.load_avg[1]?.toFixed(2)} / ${data.load_avg[2]?.toFixed(2)}`;
    }

    const procCountEl = document.getElementById('tooltip-process-count');
    if (procCountEl) {
        procCountEl.textContent = `${data.process_active || '--'} / ${data.process_total || '--'}`;
    }
}

/**
 * 更新 CPU 显示
 */
function updateCPUDisplay(data) {
    const percent = data.cpu_percent || 0;
    const cores = data.cpu_cores || 4;

    // 更新环形图
    updateRingChart('cpu', percent);

    // 更新核心数
    const coresEl = document.getElementById('cpu-cores-text');
    if (coresEl) coresEl.textContent = `${cores}核心`;

    // 更新弹窗
    const tooltipStatus = document.getElementById('tooltip-cpu-status');
    if (tooltipStatus) tooltipStatus.textContent = `占用${Math.round(percent)}%`;

    const cpuInfoEl = document.getElementById('tooltip-cpu-info');
    if (cpuInfoEl) cpuInfoEl.textContent = `1 / ${cores} / ${data.cpu_threads || cores * 2}`;

    const cpuModelEl = document.getElementById('tooltip-cpu-model');
    if (cpuModelEl) cpuModelEl.querySelector('span').textContent = data.cpu_model || 'Unknown CPU';
}

/**
 * 更新内存显示
 */
function updateMemoryDisplay(data) {
    const percent = data.memory_percent || 0;
    const used = data.memory_used_gb || 0;
    const total = data.memory_total_gb || 1;

    // 更新环形图
    updateRingChart('memory', percent);

    // 更新用量文字
    const usedEl = document.getElementById('memory-used');
    const totalEl = document.getElementById('memory-total');
    if (usedEl) usedEl.textContent = formatStorage(used);
    if (totalEl) totalEl.textContent = formatStorage(total);

    // 更新弹窗
    const tooltipStatus = document.getElementById('tooltip-memory-status');
    if (tooltipStatus) tooltipStatus.textContent = `占用${Math.round(percent)}%`;

    // 内存详情
    const memFree = document.getElementById('tooltip-mem-free');
    const memUsed = document.getElementById('tooltip-mem-used');
    const memTotal = document.getElementById('tooltip-mem-total');
    const memShared = document.getElementById('tooltip-mem-shared');
    const memAvail = document.getElementById('tooltip-mem-available');
    const memBuff = document.getElementById('tooltip-mem-buff-cache');

    if (memFree) memFree.textContent = formatStorage(data.memory_free_gb || 0);
    if (memUsed) memUsed.textContent = formatStorage(used) + ' MB';
    if (memTotal) memTotal.textContent = formatStorage(total) + ' MB';
    if (memShared) memShared.textContent = (data.memory_shared_mb || 0) + ' MB';
    if (memAvail) memAvail.textContent = formatStorage(data.memory_available_gb || 0);
    if (memBuff) memBuff.textContent = data.memory_buff_cache_mb || '--';
}

/**
 * 更新硬盘显示
 */
function updateDiskDisplay(data) {
    const percent = data.disk_percent || 0;
    const used = data.disk_used_gb || 0;
    const total = data.disk_total_gb || 1;

    // 更新环形图
    updateRingChart('disk', percent);

    // 更新用量文字
    const usedEl = document.getElementById('disk-used');
    const totalEl = document.getElementById('disk-total');
    if (usedEl) usedEl.textContent = formatStorage(used);
    if (totalEl) totalEl.textContent = formatStorage(total);

    // 更新弹窗
    const tooltipStatus = document.getElementById('tooltip-disk-status');
    if (tooltipStatus) tooltipStatus.textContent = `容量占用${percent}%`;

    // 硬盘详情
    const diskMount = document.getElementById('tooltip-disk-mount');
    const diskTotal = document.getElementById('tooltip-disk-total');
    const diskFree = document.getElementById('tooltip-disk-free');
    const diskUsed = document.getElementById('tooltip-disk-used');
    const diskFs = document.getElementById('tooltip-disk-fs');
    const diskType = document.getElementById('tooltip-disk-type');

    if (diskMount) diskMount.textContent = data.disk_mount || '/';
    if (diskTotal) diskTotal.textContent = formatStorage(total);
    if (diskFree) diskFree.textContent = formatStorage(data.disk_free_gb || 0);
    if (diskUsed) diskUsed.textContent = formatStorage(used);
    if (diskFs) diskFs.textContent = data.disk_filesystem || '--';
    if (diskType) diskType.textContent = data.disk_type || '--';
}

/**
 * 更新运行时间显示
 */
function updateUptimeDisplay() {
    if (!serverStartTime) return;
    const elapsed = Date.now() - serverStartTime;
    const uptimeEl = document.getElementById('status-uptime');
    if (uptimeEl) uptimeEl.textContent = formatUptime(elapsed);
}

/**
 * 启动运行时间计时器
 */
function startUptimeTimer() {
    if (uptimeTimer) clearInterval(uptimeTimer);
    uptimeTimer = setInterval(updateUptimeDisplay, 1000);
}

/**
 * 加载系统数据
 */
async function loadSystemData({ api, toast }) {
    try {
        const response = await api.get('/api/system/status');
        if (!response || !response.ok) return;

        const data = await api.parseJSON(response);
        if (!data) return;

        currentData = data;

        // 更新启动时间
        if (data.server_start_time) {
            serverStartTime = data.server_start_time;
            updateUptimeDisplay();
        }

        // 更新各指标
        updateLoadDisplay(data);
        updateCPUDisplay(data);
        updateMemoryDisplay(data);
        updateDiskDisplay(data);

        // 更新底部状态栏
        const statusMemEl = document.getElementById('status-memory');
        const statusGorEl = document.getElementById('status-goroutines');
        const statusSysEl = document.getElementById('status-sysinfo');

        if (statusMemEl && data.memory_mb !== undefined) {
            statusMemEl.textContent = `${data.memory_mb.toFixed(1)} MB`;
        }
        if (statusGorEl && data.goroutines !== undefined) {
            statusGorEl.textContent = data.goroutines;
        }
        if (statusSysEl && data.os && data.arch) {
            statusSysEl.textContent = `${data.os} / ${data.arch}`;
        }

    } catch (error) {
        console.error('加载系统数据失败:', error);
    }
}

/**
 * 释放内存
 */
async function freeMemory({ api, toast, dialog }) {
    const btn = document.getElementById('free-memory-btn');
    const tooltipBtn = document.getElementById('tooltip-free-memory-btn');

    if (btn) {
        btn.disabled = true;
        btn.textContent = '释放中...';
    }
    if (tooltipBtn) {
        tooltipBtn.disabled = true;
        tooltipBtn.textContent = '释放中...';
    }

    try {
        const response = await api.post('/api/system/free-memory');
        if (response && response.ok) {
            const data = await api.parseJSON(response);
            const freedSize = data?.freed_mb || 0;

            const freedEl = document.getElementById('freed-memory-size');
            if (freedEl) freedEl.textContent = `${freedSize.toFixed(1)} MB`;

            dialog.show('free-memory-dialog');
            loadSystemData({ api, toast });
        }
    } catch (error) {
        console.error('释放内存失败:', error);
        toast?.error('释放内存失败');
    } finally {
        if (btn) {
            btn.disabled = false;
            btn.textContent = '立即释放';
        }
        if (tooltipBtn) {
            tooltipBtn.disabled = false;
            tooltipBtn.textContent = '立即释放';
        }
    }
}

/**
 * 扫描清理
 */
async function scanCleanup({ api, toast, dialog }) {
    try {
        const response = await api.get('/api/system/cleanup-scan');
        if (response && response.ok) {
            const data = await api.parseJSON(response);

            const logsEl = document.getElementById('cleanup-logs');
            const tempEl = document.getElementById('cleanup-temp');
            const totalEl = document.getElementById('cleanup-total');

            if (logsEl) logsEl.textContent = `${(data?.logs_mb || 0).toFixed(1)} MB`;
            if (tempEl) tempEl.textContent = `${(data?.temp_mb || 0).toFixed(1)} MB`;
            if (totalEl) totalEl.textContent = `${(data?.total_mb || 0).toFixed(1)} MB`;

            dialog.show('cleanup-dialog');
        }
    } catch (error) {
        console.error('扫描清理失败:', error);
        toast?.error('扫描失败');
    }
}

/**
 * 执行清理
 */
async function executeCleanup({ api, toast, dialog }) {
    const btn = document.getElementById('confirm-cleanup-btn');
    if (!btn) return;

    btn.disabled = true;
    btn.textContent = '清理中...';

    try {
        const response = await api.post('/api/system/cleanup');
        if (response && response.ok) {
            const data = await api.parseJSON(response);
            const cleanedSize = data?.cleaned_mb || 0;

            dialog.hide('cleanup-dialog');
            toast?.success(`成功清理 ${cleanedSize.toFixed(1)} MB`);
            loadSystemData({ api, toast });
        }
    } catch (error) {
        console.error('清理失败:', error);
        toast?.error('清理失败');
    } finally {
        btn.disabled = false;
        btn.textContent = '立即清理';
    }
}

/**
 * 初始化可折叠区块
 */
function initCollapsible() {
    document.querySelectorAll('[data-collapsible]').forEach(header => {
        header.addEventListener('click', () => {
            const section = header.parentElement;
            const isCollapsed = section.classList.contains('collapsed');

            if (isCollapsed) {
                section.classList.remove('collapsed');
                header.classList.add('expanded');
            } else {
                section.classList.add('collapsed');
                header.classList.remove('expanded');
            }
        });
    });
}

/**
 * 初始化首页
 */
export function initHomeTab({ state, api, toast, dialog }) {
    console.log('初始化首页仪表盘...');

    // 监听状态加载
    globalEvents.on('status:loaded', (data) => {
        if (data.server_start_time) {
            serverStartTime = data.server_start_time;
            updateUptimeDisplay();
        }
    });

    // 监听标签页切换
    globalEvents.on('tab:switch:home', () => {
        loadSystemData({ api, toast });
    });

    // 刷新按钮
    const refreshBtn = document.getElementById('refresh-status');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', () => {
            loadSystemData({ api, toast });
        });
    }

    // 释放内存按钮（卡片上 + 弹窗内）
    const freeMemBtn = document.getElementById('free-memory-btn');
    const tooltipFreeBtn = document.getElementById('tooltip-free-memory-btn');

    if (freeMemBtn) {
        freeMemBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            freeMemory({ api, toast, dialog });
        });
    }
    if (tooltipFreeBtn) {
        tooltipFreeBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            freeMemory({ api, toast, dialog });
        });
    }

    // 清理按钮
    const cleanupBtn = document.getElementById('cleanup-btn');
    const tooltipCleanupBtn = document.getElementById('tooltip-cleanup-btn');

    if (cleanupBtn) {
        cleanupBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            scanCleanup({ api, toast, dialog });
        });
    }
    if (tooltipCleanupBtn) {
        tooltipCleanupBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            scanCleanup({ api, toast, dialog });
        });
    }

    // 确认清理
    const confirmCleanupBtn = document.getElementById('confirm-cleanup-btn');
    if (confirmCleanupBtn) {
        confirmCleanupBtn.addEventListener('click', () => {
            executeCleanup({ api, toast, dialog });
        });
    }

    // 初始化可折叠区块
    initCollapsible();

    // 启动定时器
    refreshTimer = setInterval(() => loadSystemData({ api, toast }), REFRESH_INTERVAL);
    syncTimer = setInterval(() => loadSystemData({ api, toast }), SYNC_INTERVAL);
    startUptimeTimer();

    // 页面可见性变化
    document.addEventListener('visibilitychange', () => {
        if (!document.hidden) {
            loadSystemData({ api, toast });
            updateUptimeDisplay();
        }
    });

    // 初始加载
    loadSystemData({ api, toast });
}

/**
 * 清理资源
 */
export function cleanupHomeTab() {
    if (uptimeTimer) clearInterval(uptimeTimer);
    if (syncTimer) clearInterval(syncTimer);
    if (refreshTimer) clearInterval(refreshTimer);
}
