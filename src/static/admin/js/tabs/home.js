/**
 * 首页模块 - 系统监控仪表盘（宝塔风格）
 *
 * 显示系统状态：CPU、内存、硬盘、负载
 */

import { BaseTab } from './BaseTab.js';
import { formatUptime, formatStorage, animateValue } from '../core/utils.js';
import { showSkeleton, hideSkeleton } from '../components/skeleton.js';

// 配置
const REFRESH_INTERVAL = 5000;
const SYNC_INTERVAL = 60000;

class HomeTab extends BaseTab {
    constructor(deps) {
        super(deps, 'home');
        this.serverStartTime = null;
        this.timers = { uptime: null, refresh: null, sync: null };
        this.currentData = {};
    }

    onInit() {
        console.log('初始化首页仪表盘...');
        
        this.bindEvents();
        this.initCollapsible();
        this.startTimers();
        
        // 监听状态加载
        this.events.on('status:loaded', (data) => {
            if (data.server_start_time) {
                this.serverStartTime = data.server_start_time;
                this.updateUptime();
            }
        });

        // 页面可见性变化
        document.addEventListener('visibilitychange', () => {
            if (!document.hidden) {
                this.refresh();
                this.updateUptime();
            }
        });
    }

    bindEvents() {
        // 刷新
        this.$('#refresh-status')?.addEventListener('click', () => {
            this.api.clearCache('/api/system/status');
            this.refresh();
        });

        // 释放内存
        this.$('#free-memory-btn')?.addEventListener('click', (e) => {
            e.stopPropagation();
            this.freeMemory();
        });
        this.$('#tooltip-free-memory-btn')?.addEventListener('click', (e) => {
            e.stopPropagation();
            this.freeMemory();
        });

        // 清理
        this.$('#cleanup-btn')?.addEventListener('click', (e) => {
            e.stopPropagation();
            this.scanCleanup();
        });
        this.$('#tooltip-cleanup-btn')?.addEventListener('click', (e) => {
            e.stopPropagation();
            this.scanCleanup();
        });

        // 确认清理
        this.$('#confirm-cleanup-btn')?.addEventListener('click', () => this.executeCleanup());
    }

    async onLoad() {
        // 使用缓存 API
        const data = await this.api.getJSON('/api/system/status');
        if (!data) return;

        this.currentData = data;

        // 更新启动时间
        if (data.server_start_time) {
            this.serverStartTime = data.server_start_time;
            this.updateUptime();
        }

        // 更新各指标
        this.updateLoad(data);
        this.updateCPU(data);
        this.updateMemory(data);
        this.updateDisk(data);
        this.updateStatusBar(data);
    }

    // ========== 更新显示 ==========

    updateRingChart(id, percent) {
        const fill = document.getElementById(`${id}-ring-fill`);
        const text = document.getElementById(`${id}-percent`);

        if (fill) {
            const circumference = 2 * Math.PI * 40;
            fill.style.strokeDashoffset = circumference - (percent / 100) * circumference;
            fill.classList.remove('warning', 'danger');
            if (percent > 80) fill.classList.add('danger');
            else if (percent > 50) fill.classList.add('warning');
        }

        if (text) animateValue(text, Math.round(percent));
    }

    updateLoad(data) {
        const load1m = data.load_avg?.[0] || 0;
        const cores = data.cpu_cores || 4;
        const percent = Math.round((load1m / cores) * 100);
        const status = percent < 50 ? { text: '运行流畅', cls: 'good' }
                      : percent < 80 ? { text: `运行达到${percent}%`, cls: 'warning' }
                      : { text: '负载较高', cls: 'danger' };

        this.updateRingChart('load', percent);
        this.setText('#load-status-text', status.text);
        this.$('#load-status-text')?.classList.add(status.cls);
        this.setText('#tooltip-load-status', status.text);
        this.setText('#tooltip-load-avg', data.load_avg?.map(v => v?.toFixed(2)).join(' / '));
        this.setText('#tooltip-process-count', `${data.process_active || '--'} / ${data.process_total || '--'}`);
    }

    updateCPU(data) {
        const percent = data.cpu_percent || 0;
        const cores = data.cpu_cores || 4;

        // 格式化 CPU 型号（截断过长的名称）
        let cpuModel = data.cpu_model || 'Unknown CPU';
        // 移除常见冗余后缀
        cpuModel = cpuModel
            .replace(/\(R\)/gi, '')
            .replace(/\(TM\)/gi, '')
            .replace(/ CPU @ .*$/gi, '')
            .replace(/ Processor/gi, '');
        // 截断到 40 字符
        if (cpuModel.length > 40) {
            cpuModel = cpuModel.substring(0, 38) + '...';
        }

        this.updateRingChart('cpu', percent);
        this.setText('#cpu-cores-text', `${cores}核心`);
        this.setText('#tooltip-cpu-status', `占用${Math.round(percent)}%`);
        this.setText('#tooltip-cpu-info', `1 / ${cores} / ${data.cpu_threads || cores * 2}`);
        this.setText('#tooltip-cpu-model span', cpuModel);
    }

    updateMemory(data) {
        const percent = data.memory_percent || 0;
        const used = data.memory_used_gb || 0;
        const total = data.memory_total_gb || 1;

        this.updateRingChart('memory', percent);
        this.setText('#memory-used', formatStorage(used));
        this.setText('#memory-total', formatStorage(total));
        this.setText('#tooltip-memory-status', `占用${Math.round(percent)}%`);
        this.setText('#tooltip-mem-free', formatStorage(data.memory_free_gb || 0));
        this.setText('#tooltip-mem-used', formatStorage(used) + ' MB');
        this.setText('#tooltip-mem-total', formatStorage(total) + ' MB');
        this.setText('#tooltip-mem-shared', (data.memory_shared_mb || 0) + ' MB');
        this.setText('#tooltip-mem-available', formatStorage(data.memory_available_gb || 0));
        this.setText('#tooltip-mem-buff-cache', data.memory_buff_cache_mb || '--');
    }

    updateDisk(data) {
        const percent = data.disk_percent || 0;
        const used = data.disk_used_gb || 0;
        const total = data.disk_total_gb || 1;

        this.updateRingChart('disk', percent);
        this.setText('#disk-used', formatStorage(used));
        this.setText('#disk-total', formatStorage(total));
        this.setText('#tooltip-disk-status', `容量占用${percent}%`);
        this.setText('#tooltip-disk-mount', data.disk_mount || '/');
        this.setText('#tooltip-disk-total', formatStorage(total));
        this.setText('#tooltip-disk-free', formatStorage(data.disk_free_gb || 0));
        this.setText('#tooltip-disk-used', formatStorage(used));
        this.setText('#tooltip-disk-fs', data.disk_filesystem || '--');
        this.setText('#tooltip-disk-type', data.disk_type || '--');
    }

    updateStatusBar(data) {
        this.setText('#status-memory', data.memory_mb?.toFixed(1) + ' MB');
        this.setText('#status-goroutines', data.goroutines);
        this.setText('#status-sysinfo', `${data.os} / ${data.arch}`);
    }

    updateUptime() {
        if (!this.serverStartTime) return;
        const elapsed = Date.now() - this.serverStartTime;
        this.setText('#status-uptime', formatUptime(elapsed));
    }

    // ========== 操作 ==========

    async freeMemory() {
        const btns = [this.$('#free-memory-btn'), this.$('#tooltip-free-memory-btn')];
        btns.forEach(btn => { if (btn) { btn.disabled = true; btn.textContent = '释放中...'; } });

        try {
            const response = await this.api.post('/api/system/free-memory');
            if (response?.ok) {
                const data = await this.api.parseJSON(response);
                this.setText('#freed-memory-size', `${(data?.freed_mb || 0).toFixed(1)} MB`);
                this.dialog.show('free-memory-dialog');
                await this.refresh();
            }
        } catch (error) {
            this.toast.error('释放内存失败');
        } finally {
            btns.forEach(btn => { if (btn) { btn.disabled = false; btn.textContent = '立即释放'; } });
        }
    }

    async scanCleanup() {
        try {
            const response = await this.api.get('/api/system/cleanup-scan');
            if (response?.ok) {
                const data = await this.api.parseJSON(response);
                this.setText('#cleanup-logs', `${(data?.logs_mb || 0).toFixed(1)} MB`);
                this.setText('#cleanup-temp', `${(data?.temp_mb || 0).toFixed(1)} MB`);
                this.setText('#cleanup-total', `${(data?.total_mb || 0).toFixed(1)} MB`);
                this.dialog.show('cleanup-dialog');
            }
        } catch (error) {
            this.toast.error('扫描失败');
        }
    }

    async executeCleanup() {
        const btn = this.$('#confirm-cleanup-btn');
        if (btn) { btn.disabled = true; btn.textContent = '清理中...'; }

        try {
            const response = await this.api.post('/api/system/cleanup');
            if (response?.ok) {
                const data = await this.api.parseJSON(response);
                this.dialog.hide('cleanup-dialog');
                this.toast.success(`成功清理 ${(data?.cleaned_mb || 0).toFixed(1)} MB`);
                await this.refresh();
            }
        } catch (error) {
            this.toast.error('清理失败');
        } finally {
            if (btn) { btn.disabled = false; btn.textContent = '立即清理'; }
        }
    }

    // ========== 工具 ==========

    initCollapsible() {
        document.querySelectorAll('[data-collapsible]').forEach(header => {
            header.addEventListener('click', () => {
                const section = header.parentElement;
                section.classList.toggle('collapsed');
                header.classList.toggle('expanded');
            });
        });
    }

    startTimers() {
        this.timers.uptime = setInterval(() => this.updateUptime(), 1000);
        this.timers.refresh = setInterval(() => this.refresh(), REFRESH_INTERVAL);
        this.timers.sync = setInterval(() => this.refresh(), SYNC_INTERVAL);
    }

    stopTimers() {
        Object.values(this.timers).forEach(t => t && clearInterval(t));
    }

    // ========== 骨架屏 ==========

    showMetricsSkeleton() {
        // 使用 skeleton.js 的 showSkeleton 函数
        showSkeleton('.metrics-grid', 'metric', { count: 4 });
    }

    onDestroy() {
        this.stopTimers();
    }
}

// 单例
let instance = null;

export function initHomeTab(deps) {
    if (!instance) {
        instance = new HomeTab(deps);
        instance.init();
    }
    return instance;
}

export function cleanupHomeTab() {
    instance?.destroy();
}