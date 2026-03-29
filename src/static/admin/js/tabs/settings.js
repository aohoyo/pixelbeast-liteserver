/**
 * 设置模块
 *
 * 负责系统设置的加载、编辑、保存
 */

import { BaseTab } from './BaseTab.js';

class SettingsTab extends BaseTab {
    constructor(deps) {
        super(deps, 'settings');
        this.config = null;
        this.originalConfig = null;
        this.hasChanges = false;
    }

    onInit() {
        console.log('[Settings] 初始化设置面板...');
        this.initNumberInputs();
        this.bindEvents();
        this.startClock();
    }

    initNumberInputs() {
        this.$$('.number-input-wrapper').forEach(wrapper => {
            const input = wrapper.querySelector('input[type="number"]');
            const upBtn = wrapper.querySelector('[data-action="up"]');
            const downBtn = wrapper.querySelector('[data-action="down"]');

            if (!input) return;

            const step = parseInt(input.dataset.step) || 1;
            const min = input.min !== '' ? parseInt(input.min) : null;
            const max = input.max !== '' ? parseInt(input.max) : null;

            const updateValue = (delta) => {
                let value = parseInt(input.value) || 0;
                value += delta;
                if (min !== null && value < min) value = min;
                if (max !== null && value > max) value = max;
                input.value = value;
                input.dispatchEvent(new Event('input', { bubbles: true }));
            };

            upBtn?.addEventListener('click', () => updateValue(step));
            downBtn?.addEventListener('click', () => updateValue(-step));

            input.addEventListener('keydown', (e) => {
                if (e.key === 'ArrowUp') {
                    e.preventDefault();
                    updateValue(step);
                } else if (e.key === 'ArrowDown') {
                    e.preventDefault();
                    updateValue(-step);
                }
            });
        });
    }

    bindEvents() {
        // 标签页切换
        this.$$('.settings-tabs .tab-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const tab = e.currentTarget.dataset.tab;
                this.switchTab(tab);
            });
        });

        // 保存按钮
        this.$('#btn-save')?.addEventListener('click', () => this.save());

        // 重置按钮
        this.$('#btn-reset')?.addEventListener('click', () => this.reset());

        // 同步时间按钮
        this.$('#btn-sync-time')?.addEventListener('click', () => this.syncTime());

        // 密码切换显示
        this.$('#toggle-password')?.addEventListener('click', () => {
            const input = this.$('#admin-password');
            const btn = this.$('#toggle-password');
            if (input && btn) {
                const isPassword = input.type === 'password';
                input.type = isPassword ? 'text' : 'password';
                btn.innerHTML = isPassword
                    ? '<svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>'
                    : '<svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>';
            }
        });

        // 监听配置变化
        this.$$('.settings-content input, .settings-content select').forEach(input => {
            input.addEventListener('change', () => {
                this.hasChanges = true;
            });
        });
    }

    switchTab(tabName) {
        // 更新按钮状态
        this.$$('.settings-tabs .tab-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.tab === tabName);
        });

        // 更新内容区
        this.$$('.tab-pane').forEach(pane => {
            pane.classList.toggle('active', pane.id === `tab-${tabName}`);
        });
    }

    async onLoad() {
        try {
            const config = await this.api.getJSON('/api/config');
            this.config = config;
            this.originalConfig = JSON.parse(JSON.stringify(this.config));
            this.render();
            console.log('[Settings] 配置加载成功', this.config);
        } catch (error) {
            console.error('[Settings] 加载配置失败:', error);
            this.toast?.error('加载配置失败：' + error.message);
        }
    }

    render() {
        if (!this.config) return;

        const { admin, http, log, backup_dir } = this.config;

        // 基础设置
        this.setValue('#server-name', 'PixelBeast Server');
        this.setValue('#timezone', 'Asia/Shanghai');

        // HTTP 服务
        this.setValue('#http-port', http?.port || 3080);
        this.setValue('#http-root', http?.root || './web');

        // 管理面板
        this.setValue('#admin-port', admin?.port || 9527);
        this.setValue('#admin-path', admin?.path || '/admin');
        this.setValue('#admin-username', admin?.username || 'admin');
        this.setValue('#admin-password', '');
        this.setValue('#admin-bind-domain', admin?.bind_domain || '');
        this.setChecked('#admin-ssl-enabled', admin?.ssl_enabled ?? false);

        // 日志配置
        this.setValue('#log-level', log?.level || 'info');
        this.setValue('#log-retention', log?.retention_days || 30);
        this.setValue('#log-max-size', log?.max_size_mb || 100);
        this.setValue('#log-compress', log?.compress_days || 7);
        this.setValue('#log-cleanup-hour', log?.cleanup_hour || 3);

        // 备份设置
        this.setValue('#backup-dir', backup_dir || './backups');
        this.setChecked('#auto-backup', true);

        // FTP 配置
        const ftp = this.config.ftp || {};
        this.setChecked('#ftp-enabled', ftp.enabled ?? false);
        this.setValue('#ftp-port', ftp.port || 2121);
        this.setValue('#ftp-root', ftp.root || './ftp');

        this.hasChanges = false;
    }

    collectConfig() {
        const config = JSON.parse(JSON.stringify(this.originalConfig));

        // HTTP
        config.http.port = this.getInt('#http-port', 3080);
        config.http.root = this.getValue('#http-root') || './web';

        // Admin
        config.admin.port = this.getInt('#admin-port', 9527);
        config.admin.path = this.getValue('#admin-path') || '/admin';
        config.admin.username = this.getValue('#admin-username') || 'admin';
        const pwd = this.getValue('#admin-password');
        if (pwd && pwd.trim()) {
            config.admin.password = pwd;
        }

        // 域名绑定 & SSL
        config.admin.bind_domain = this.getValue('#admin-bind-domain') || '';
        config.admin.ssl_enabled = this.isChecked('#admin-ssl-enabled');

        // Log
        config.log.level = this.getValue('#log-level') || 'info';
        config.log.retention_days = this.getInt('#log-retention', 30);
        config.log.max_size_mb = this.getInt('#log-max-size', 100);
        config.log.compress_days = this.getInt('#log-compress', 7);
        config.log.cleanup_hour = this.getInt('#log-cleanup-hour', 3);

        // Backup
        config.backup_dir = this.getValue('#backup-dir') || './backups';

        // FTP
        config.ftp.enabled = this.isChecked('#ftp-enabled');
        config.ftp.port = this.getInt('#ftp-port', 2121);
        config.ftp.root = this.getValue('#ftp-root') || './ftp';

        return config;
    }

    async save() {
        if (!this.hasChanges && !this.getValue('#admin-password')) {
            this.toast?.info('没有需要保存的更改');
            return;
        }

        try {
            const config = this.collectConfig();
            await this.api.post('/api/config/save', config);
            this.originalConfig = JSON.parse(JSON.stringify(config));
            this.hasChanges = false;
            this.toast?.success('配置保存成功');
            this.events.emit('config:updated', config);
        } catch (error) {
            console.error('[Settings] 保存失败:', error);
            this.toast?.error('保存失败：' + error.message);
        }
    }

    async reset() {
        this.dialog?.confirm('确定要重置所有配置吗？未保存的更改将丢失。', async () => {
            try {
                const data = await this.api.postJSON('/api/config/reset');
                this.config = data;
                this.originalConfig = JSON.parse(JSON.stringify(this.config));
                this.render();
                this.toast?.success('配置已重置');
                this.events.emit('config:updated', this.config);
            } catch (error) {
                console.error('[Settings] 重置失败:', error);
                this.toast?.error('重置失败：' + error.message);
            }
        });
    }

    async onRefresh() {
        await this.onLoad();
    }

    onDestroy() {
        if (this._clockTimer) {
            clearInterval(this._clockTimer);
            this._clockTimer = null;
        }
    }

    startClock() {
        // 只启动时钟更新，不自动同步 NTP
        this._serverTime = Date.now();
        this._serverTimeFetched = Date.now();
        this.updateClockDisplay();
        this._clockTimer = setInterval(() => {
            this.updateClockDisplay();
        }, 1000);
    }

    async syncTime() {
        const btn = this.$('#btn-sync-time');
        const icon = btn?.querySelector('.icon');
        if (btn) btn.disabled = true;
        if (icon) icon.classList.add('spinning');

        try {
            // POST 同步系统时间（会尝试修改系统时钟）
            const data = await this.api.postJSON('/api/system/time/sync');
            if (data?.updated) {
                this.toast?.success(data.message || '系统时间已校正');
            } else {
                this.toast?.success(data?.message || '系统时间已同步');
            }
            // 刷新时钟显示
            const time = await this.api.getJSON('/api/system/time');
            this._serverTime = time.unix_milli;
            this._serverTimeFetched = Date.now();
            this.updateClockDisplay();
        } catch (error) {
            console.error('[Settings] 同步时间失败:', error);
            this.toast?.error('同步时间失败：' + (error.message || '请检查权限'));
        } finally {
            if (icon) icon.classList.remove('spinning');
            if (btn) btn.disabled = false;
        }
    }

    updateClockDisplay() {
        const el = this.$('#system-time');
        if (!el) return;

        let now;
        if (this._serverTime && this._serverTimeFetched) {
            const elapsed = Date.now() - this._serverTimeFetched;
            now = new Date(this._serverTime + elapsed);
        } else {
            now = new Date();
        }

        const timezone = this.getValue('#timezone') || 'Asia/Shanghai';
        try {
            el.textContent = now.toLocaleString('zh-CN', {
                timeZone: timezone,
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit',
                hour12: false
            });
        } catch {
            el.textContent = now.toLocaleString('zh-CN', { hour12: false });
        }
    }

    // ========== 工具方法 ==========

    getValue(selector) {
        const el = this.$(selector);
        return el?.value || '';
    }

    setValue(selector, value) {
        const el = this.$(selector);
        if (el) {
            el.value = value;
        }
    }

    getInt(selector, defaultVal) {
        const val = parseInt(this.getValue(selector));
        return isNaN(val) ? defaultVal : val;
    }

    isChecked(selector) {
        const el = this.$(selector);
        return el?.checked ?? false;
    }

    setChecked(selector, checked) {
        const el = this.$(selector);
        if (el) {
            el.checked = checked;
        }
    }
}

let instance = null;

export function initSettingsTab(deps) {
    if (!instance) {
        instance = new SettingsTab(deps);
        instance.init();
    }
    return instance;
}

export function cleanupSettingsTab() {
    instance?.destroy();
    instance = null;
}