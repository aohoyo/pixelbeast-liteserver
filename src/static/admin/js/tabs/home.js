/**
 * 首页模块 - 系统监控仪表盘（宝塔风格）
 *
 * 显示系统状态：CPU、内存、磁盘、负载
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
        this.netHistory = { sent: [], recv: [] };
        this.diskIOHistory = { write: [], read: [] };
        this.NET_HISTORY_MAX = 30;
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
        this.updateNetwork(data);
        this.updateDiskIO(data);

        // 更新服务状态（使用已加载的数据）
        this.updateServices(data);
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
        const percent = Math.min(Math.round((load1m / cores) * 100), 100);
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

        // 渲染核心使用率
        const coreList = this.$('#core-usage-list');
        if (coreList && Array.isArray(data.cpu_per_core) && data.cpu_per_core.length > 0) {
            coreList.innerHTML = data.cpu_per_core.map((v, i) => {
                const p = Math.round(v);
                const cls = p > 80 ? 'danger' : p > 50 ? 'warning' : '';
                return `<div class="core-usage-item">
                    <span class="core-usage-label">${i}</span>
                    <div class="core-usage-bar"><div class="core-usage-bar-fill ${cls}" style="width:${p}%"></div></div>
                    <span class="core-usage-value">${p}%</span>
                </div>`;
            }).join('');
        }
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
        this.setText('#tooltip-mem-swap',
            data.swap_total_gb > 0
                ? `${formatStorage(data.swap_used_gb)} / ${formatStorage(data.swap_total_gb)} (${Math.round(data.swap_percent || 0)}%)`
                : '未启用');
    }

    updateDisk(data) {
        const percent = data.disk_percent || 0;
        const used = data.disk_used_gb || 0;
        const total = data.disk_total_gb || 1;

        this.updateRingChart('disk', percent);
        this.setText('#disk-used', formatStorage(used));
        this.setText('#disk-total', formatStorage(total));
        this.setText('#tooltip-disk-status', `容量占用${Math.round(percent)}%`);

        // 渲染磁盘列表
        const diskList = this.$('#disk-list');
        if (diskList && Array.isArray(data.disks) && data.disks.length > 0) {
            diskList.innerHTML = data.disks.map(d => {
                const p = Math.round(d.percent || 0);
                const barCls = p > 80 ? 'danger' : p > 50 ? 'warning' : '';
                const primary = d.mount === (data.disk_mount || '/') ? ' <span class="disk-primary-badge">主</span>' : '';
                return `
                    <div class="disk-item">
                        <div class="disk-item-header">
                            <span class="disk-item-mount">${d.mount}${primary}</span>
                            <span class="disk-item-percent">${p}%</span>
                        </div>
                        <div class="disk-item-bar">
                            <div class="disk-item-bar-fill ${barCls}" style="width:${p}%"></div>
                        </div>
                        <div class="disk-item-detail">
                            <span>${formatStorage(d.used_gb || 0)} / ${formatStorage(d.total_gb || 0)}</span>
                            <span>${d.fstype || '--'} · ${d.device || '--'}</span>
                        </div>
                    </div>`;
            }).join('');
        }
    }

    updateNetwork(data) {
        const sentRate = data.net_sent_rate_kb || 0;
        const recvRate = data.net_recv_rate_kb || 0;
        const totalSent = data.net_total_sent_gb || 0;
        const totalRecv = data.net_total_recv_gb || 0;

        const formatRate = (kb) => {
            if (kb >= 1048576) return (kb / 1048576).toFixed(2) + ' GB/s';
            if (kb >= 1024) return (kb / 1024).toFixed(1) + ' MB/s';
            if (kb >= 1) return kb.toFixed(1) + ' KB/s';
            return '0 B/s';
        };

        const formatTotal = (gb) => {
            if (gb >= 1024) return (gb / 1024).toFixed(1) + ' TB';
            if (gb >= 1) return gb.toFixed(2) + ' GB';
            return (gb * 1024).toFixed(0) + ' MB';
        };

        this.setText('#network-sent-rate', formatRate(sentRate));
        this.setText('#network-recv-rate', formatRate(recvRate));
        this.setText('#network-total-sent', formatTotal(totalSent));
        this.setText('#network-total-recv', formatTotal(totalRecv));

        // 记录历史
        this.netHistory.sent.push(sentRate);
        this.netHistory.recv.push(recvRate);
        if (this.netHistory.sent.length > this.NET_HISTORY_MAX) {
            this.netHistory.sent.shift();
            this.netHistory.recv.shift();
        }

        // 绘制折线图
        this.drawChart('network-canvas', this.netHistory.sent, this.netHistory.recv,
            '#3b82f6', 'rgba(59,130,246,0.15)', '#22c55e', 'rgba(34,197,94,0.15)');
    }

    drawChart(canvasId, data1, data2, color1, fill1, color2, fill2) {
        const canvas = document.getElementById(canvasId);
        if (!canvas) return;

        // 设置 canvas 实际像素尺寸
        const rect = canvas.parentElement.getBoundingClientRect();
        if (rect.width > 0 && rect.height > 0) {
            canvas.width = rect.width * (window.devicePixelRatio || 1);
            canvas.height = rect.height * (window.devicePixelRatio || 1);
        }

        const ctx = canvas.getContext('2d');
        const W = canvas.width;
        const H = canvas.height;

        ctx.clearRect(0, 0, W, H);
        if (data1.length < 2) return;

        const maxVal = Math.max(...data1, ...data2, 1) * 1.2;
        const stepX = W / (this.NET_HISTORY_MAX - 1);

        const drawLine = (data, stroke, fill) => {
            ctx.beginPath();
            data.forEach((v, i) => {
                const x = i * stepX;
                const y = H - (v / maxVal) * H;
                if (i === 0) ctx.moveTo(x, y);
                else ctx.lineTo(x, y);
            });
            ctx.strokeStyle = stroke;
            ctx.lineWidth = 1.5 * (window.devicePixelRatio || 1);
            ctx.stroke();

            ctx.lineTo((data.length - 1) * stepX, H);
            ctx.lineTo(0, H);
            ctx.closePath();
            const grad = ctx.createLinearGradient(0, 0, 0, H);
            grad.addColorStop(0, fill);
            grad.addColorStop(1, 'transparent');
            ctx.fillStyle = grad;
            ctx.fill();
        };

        drawLine(data2, color2, fill2);
        drawLine(data1, color1, fill1);
    }

    updateDiskIO(data) {
        const writeRate = data.diskio_speed_write_kb || 0;
        const readRate = data.diskio_speed_read_kb || 0;
        const iops = data.diskio_iops || 0;
        const latency = data.diskio_latency_ms || 0;

        const formatRate = (kb) => {
            if (kb >= 1048576) return (kb / 1048576).toFixed(2) + ' GB/s';
            if (kb >= 1024) return (kb / 1024).toFixed(1) + ' MB/s';
            if (kb >= 1) return kb.toFixed(1) + ' KB/s';
            return (kb * 1024).toFixed(0) + ' B/s';
        };

        this.setText('#diskio-read-rate', formatRate(readRate));
        this.setText('#diskio-write-rate', formatRate(writeRate));
        this.setText('#diskio-iops', iops > 0 ? iops.toFixed(0) : '--');
        this.setText('#diskio-latency', latency > 0 ? latency.toFixed(1) + ' ms' : '--');

        // 记录历史
        this.diskIOHistory.write.push(writeRate);
        this.diskIOHistory.read.push(readRate);
        if (this.diskIOHistory.write.length > this.NET_HISTORY_MAX) {
            this.diskIOHistory.write.shift();
            this.diskIOHistory.read.shift();
        }

        // 绘制折线图
        this.drawChart('diskio-canvas', this.diskIOHistory.write, this.diskIOHistory.read,
            '#f97316', 'rgba(249,115,22,0.15)', '#a78bfa', 'rgba(167,139,250,0.15)');
    }

    updateServices(data) {
        if (!data) return;

        this.updateServiceCard('admin', data.admin_running, `端口 :${data.admin_port}`);
        this.updateServiceCard('sites', data.sites_running,
            data.sites_count > 0 ? `${data.sites_count} 个站点` : '无站点');
        this.updateServiceCard('ftp', data.ftp_running,
            data.ftp_port ? `端口 :${data.ftp_port}` : '未配置');
    }

    updateServiceCard(service, running, detail) {
        const card = this.$(`#service-${service}`);
        const statusEl = card?.querySelector('.status-text');
        const portEl = card?.querySelector('.service-status-port');

        if (card) {
            card.classList.remove('running', 'stopped');
            card.classList.add(running ? 'running' : 'stopped');
        }
        if (statusEl) statusEl.textContent = running ? '运行中' : '已停止';
        if (portEl) portEl.innerHTML = detail;
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