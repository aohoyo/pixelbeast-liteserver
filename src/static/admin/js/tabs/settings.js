/**
 * 设置模块
 *
 * 负责系统设置的加载、编辑、保存
 */

import { BaseTab } from './BaseTab.js';
import { openFileBrowser } from '../components/file-browser/index.js';

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

        // 目录选择器
        this.$$('.directory-picker-btn').forEach(btn => {
            btn.addEventListener('click', () => this.openDirPicker(btn.dataset.dir));
        });

        // 创建备份按钮
        this.$('#btn-create-backup')?.addEventListener('click', () => this.createBackup());
        this.$('#btn-refresh-backup')?.addEventListener('click', () => this.loadBackups());

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

        // 备份列表延迟加载
        if (tabName === 'backup' && !this._backupLoaded) {
            this.loadBackups();
            this._backupLoaded = true;
        }
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

        const { admin, directories, log, backup } = this.config;

        // 基础设置
        this.setValue('#server-name', admin?.name || 'PixelBeast Server');
        this.setValue('#timezone', 'Asia/Shanghai');

        // 目录设置
        this.setValue('#dir-sites', directories?.sites || './sites');
        this.setValue('#dir-ftp', directories?.ftp || './ftp');
        this.setValue('#dir-backup', directories?.backup || './backups');

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
        this.setChecked('#backup-enabled', backup?.auto_enabled ?? false);
        const items = backup?.items || ['config', 'sites', 'ftp'];
        this.setChecked('#backup-item-config', items.includes('config'));
        this.setChecked('#backup-item-sites', items.includes('sites'));
        this.setChecked('#backup-item-ftp', items.includes('ftp'));
        this.setValue('#backup-schedule', backup?.schedule || 'daily');
        this.setValue('#backup-retention', backup?.retention || 3);

        this.hasChanges = false;
    }

    async openDirPicker(inputId) {
        const input = this.$(`#${inputId}`);
        if (!input) return;
        try {
            const selected = await openFileBrowser({
                title: '选择目录',
                selectMode: 'folder',
                root: input.value || '.',
                api: this.api,
            });
            if (selected) {
                input.value = selected;
                input.dispatchEvent(new Event('change', { bubbles: true }));
            }
        } catch (e) {
            // 用户取消或关闭
        }
    }


    // ========== 备份管理 ==========

    async loadBackups() {
        const container = this.$('#backup-list');
        if (!container) return;
        try {
            const data = await this.api.getJSON('/api/backups');
            const backups = data?.backups || [];
            if (backups.length === 0) {
                container.innerHTML = '<div class="backup-empty" style="text-align: center; padding: 32px; color: var(--text-muted, #78716c); font-size: 14px;">暂无备份文件</div>';
                return;
            }
            container.innerHTML = `
                <div style="display: grid; gap: 8px;">
                    ${backups.map(b => `
                        <div class="backup-item" style="display: flex; align-items: center; justify-content: space-between; padding: 12px 16px; background: var(--bg-elevated, #1c1917); border-radius: 8px; border: 1px solid var(--border, #44403c);">
                            <div style="display: flex; align-items: center; gap: 12px; flex: 1; min-width: 0;">
                                <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:20px;height:20px;color:var(--warning, #fbbf24);flex-shrink:0;"><path d="M21 8v13H3V8"/><path d="M1 3h22v5H1z"/><path d="M10 12h4"/></svg>
                                <div style="min-width: 0;">
                                    <div style="font-size: 14px; color: var(--text, #fafaf9); overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">${this.escapeHtml(b.name)}</div>
                                    <div style="font-size: 12px; color: var(--text-muted, #78716c);">${b.modified} · ${this.formatSize(b.size)}</div>
                                </div>
                            <div style="display: flex; gap: 6px; flex-shrink: 0;">
                                <button class="btn btn-sm backup-download-btn" data-name="${this.escapeHtml(b.name)}" title="下载" style="display:flex;align-items:center;justify-content:center;width:32px;height:32px;">
                                    <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:16px;height:16px;"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                                </button>
                                <button class="btn btn-sm backup-restore-btn" data-name="${this.escapeHtml(b.name)}" title="恢复" style="display:flex;align-items:center;justify-content:center;width:32px;height:32px;">
                                    <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:16px;height:16px;"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>
                                </button>
                                <button class="btn btn-sm backup-delete-btn" data-name="${this.escapeHtml(b.name)}" style="color:var(--danger, #ef4444);" title="删除">
                                    <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:16px;height:16px;"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                                </button>
                            </div>
                            </div>
                        </div>
                    `).join('')}
                </div>
            `;
            // Bind events
            container.querySelectorAll('.backup-download-btn').forEach(btn => {
                btn.addEventListener('click', () => this.downloadBackup(btn.dataset.name));
            });
            container.querySelectorAll('.backup-restore-btn').forEach(btn => {
                btn.addEventListener('click', () => this.restoreBackup(btn.dataset.name));
            });
            container.querySelectorAll('.backup-delete-btn').forEach(btn => {
                btn.addEventListener('click', () => this.deleteBackup(btn.dataset.name));
            });
        } catch (error) {
            console.error('[Settings] 加载备份列表失败:', error);
        }
    }

    async createBackup() {
        const btn = this.$('#btn-create-backup');
        if (btn) btn.disabled = true;
        try {
            const data = await this.api.postJSON('/api/backups/create');
            this.toast?.success(data?.message || '备份创建成功');
            this.loadBackups();
        } catch (error) {
            this.toast?.error('创建备份失败：' + error.message);
        } finally {
            if (btn) btn.disabled = false;
        }
    }

    downloadBackup(name) {
        const link = document.createElement('a');
        link.href = `/api/backups/download?name=${encodeURIComponent(name)}`;
        link.download = name;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
    }

    async restoreBackup(name) {
        this.dialog?.confirm(`确定要从备份 "${name}" 恢复吗？当前配置将被覆盖。`, async () => {
            try {
                const data = await this.api.postJSON('/api/backups/restore', { name });
                this.toast?.success(data?.message || '备份恢复成功');
                // Reload config after restore
                await this.onLoad();
            } catch (error) {
                this.toast?.error('恢复失败：' + error.message);
            }
        });
    }

    async deleteBackup(name) {
        this.dialog?.confirm(`确定要删除备份 "${name}" 吗？`, async () => {
            try {
                await this.api.post('/api/backups/delete', { name });
                this.toast?.success('备份已删除');
                this.loadBackups();
            } catch (error) {
                this.toast?.error('删除失败：' + error.message);
            }
        });
    }

    formatSize(bytes) {
        if (!bytes || bytes === 0) return '0 B';
        const units = ['B', 'KB', 'MB', 'GB'];
        let i = 0;
        let size = bytes;
        while (size >= 1024 && i < units.length - 1) {
            size /= 1024;
            i++;
        }
        return size.toFixed(i === 0 ? 0 : 1) + ' ' + units[i];
    }

    escapeHtml(str) {
        if (!str) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    collectConfig() {
        const config = JSON.parse(JSON.stringify(this.originalConfig));

        // Directories
        config.directories = {
            sites: this.getValue('#dir-sites') || './sites',
            ftp: this.getValue('#dir-ftp') || './ftp',
            backup: this.getValue('#dir-backup') || './backups',
        };

        // Backup
        config.backup = {
            auto_enabled: this.isChecked('#backup-enabled'),
            items: [
                ...(this.isChecked('#backup-item-config') ? ['config'] : []),
                ...(this.isChecked('#backup-item-sites') ? ['sites'] : []),
                ...(this.isChecked('#backup-item-ftp') ? ['ftp'] : []),
            ],
            schedule: this.getValue('#backup-schedule') || 'daily',
            retention: this.getInt('#backup-retention', 3),
        };

        // Admin
        config.admin.name = this.getValue('#server-name') || 'PixelBeast Server';
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
            const data = await this.api.postJSON('/api/system/time/sync');
            if (data?.updated) {
                this.toast?.success(data.message || '系统时间已校正');
            } else {
                this.toast?.success(data?.message || '系统时间已同步');
            }
            // 直接用 sync 返回的时间数据刷新时钟
            if (data?.unix_milli) {
                this._serverTime = data.unix_milli;
                this._serverTimeFetched = Date.now();
                this.updateClockDisplay();
            }
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