/**
 * 设置模块 - 重构版
 *
 * 负责系统设置的加载、编辑、保存
 */

import { BaseTab } from './BaseTab.js';
import { DirectoryPicker } from '../components/directory-picker.js';

class SettingsTab extends BaseTab {
    constructor(deps) {
        super(deps, 'settings');
        this.config = null;
        this.originalConfig = null;
        this.hasChanges = false;
        this.currentTab = 'basic';
        this.directoryPickers = {};
    }

    onInit() {
        console.log('[Settings] 初始化设置面板...');
        this.bindEvents();
        this.initDirectoryPickers();
    }

    /**
     * 绑定事件
     */
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

        // 密码切换显示
        this.$('.password-toggle')?.addEventListener('click', (e) => {
            const input = this.$('#admin-password');
            const eyeOn = e.currentTarget.querySelector('.icon-eye');
            const eyeOff = e.currentTarget.querySelector('.icon-eye-off');
            
            if (input.type === 'password') {
                input.type = 'text';
                eyeOn.style.display = 'none';
                eyeOff.style.display = 'block';
            } else {
                input.type = 'password';
                eyeOn.style.display = 'block';
                eyeOff.style.display = 'none';
            }
        });

        // 监听配置变化
        const inputs = this.$$('.settings-content input, .settings-content select');
        inputs.forEach(input => {
            input.addEventListener('change', () => {
                this.hasChanges = true;
            });
        });
    }

    /**
     * 初始化目录选择器
     */
    initDirectoryPickers() {
        // WEB 根目录
        const webRootContainer = this.$('#web-root-picker');
        if (webRootContainer) {
            this.directoryPickers.webRoot = new DirectoryPicker({
                container: webRootContainer,
                api: this.api,
                apiPath: '/api/files',
                placeholder: './web',
                onChange: () => { this.hasChanges = true; }
            });
        }

        // 备份目录
        const backupDirContainer = this.$('#backup-dir-picker');
        if (backupDirContainer) {
            this.directoryPickers.backup = new DirectoryPicker({
                container: backupDirContainer,
                api: this.api,
                apiPath: '/api/files',
                placeholder: './backups',
                onChange: () => { this.hasChanges = true; }
            });
        }
    }

    /**
     * 切换标签页
     */
    switchTab(tabName) {
        this.currentTab = tabName;

        // 更新按钮状态
        this.$$('.settings-tabs .tab-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.tab === tabName);
        });

        // 更新内容区
        this.$$('.tab-pane').forEach(pane => {
            pane.classList.toggle('active', pane.id === `tab-${tabName}`);
        });
    }

    /**
     * 加载配置
     */
    async onLoad() {
        try {
            const response = await this.api.getJSON('/api/config');
            this.config = response.data;
            this.originalConfig = JSON.parse(JSON.stringify(this.config));
            this.render();
            console.log('[Settings] 配置加载成功', this.config);
        } catch (error) {
            console.error('[Settings] 加载配置失败:', error);
            this.toast.error('加载配置失败：' + error.message);
        }
    }

    /**
     * 渲染配置到表单
     */
    render() {
        if (!this.config) return;

        const { admin, http, log, backup_dir } = this.config;

        // 基础设置
        this.setText('#server-name', 'PixelBeast Server');
        this.setText('#timezone', 'Asia/Shanghai');

        // HTTP 服务
        this.setText('#http-port', http?.port || 8080);
        this.setText('#index-files', http?.index_files?.join(', ') || 'index.html, index.htm');
        this.$('#auto-index').checked = http?.auto_index ?? true;

        // 更新目录选择器
        if (this.directoryPickers.webRoot && http?.root) {
            this.directoryPickers.webRoot.setValue(http.root);
        }

        // 管理面板
        this.setText('#admin-port', admin?.port || 9527);
        this.setText('#admin-path', admin?.path || '/admin');
        this.setText('#admin-username', admin?.username || 'admin');
        this.setText('#admin-password', '');

        // 日志配置
        this.setText('#log-level', log?.level || 'info');
        this.setText('#log-retention', log?.retention_days || 30);
        this.setText('#log-max-size', log?.max_size_mb || 100);
        this.setText('#log-compress', log?.compress_days || 7);
        this.setText('#log-cleanup-hour', log?.cleanup_hour || 3);

        // 备份设置
        this.setText('#backup-dir', backup_dir || './backups');
        if (this.directoryPickers.backup && backup_dir) {
            this.directoryPickers.backup.setValue(backup_dir);
        }
        this.$('#auto-backup').checked = true;

        this.hasChanges = false;
    }

    /**
     * 从表单收集配置
     */
    collectConfig() {
        const config = JSON.parse(JSON.stringify(this.originalConfig));

        // HTTP 服务
        config.http.port = parseInt(this.$('#http-port').value) || 8080;
        config.http.root = this.directoryPickers.webRoot?.getValue() || './web';
        config.http.index_files = (this.$('#index-files').value || 'index.html')
            .split(',')
            .map(s => s.trim())
            .filter(Boolean);
        config.http.auto_index = this.$('#auto-index').checked;

        // 管理面板
        config.admin.port = parseInt(this.$('#admin-port').value) || 9527;
        config.admin.path = this.$('#admin-path').value || '/admin';
        config.admin.username = this.$('#admin-username').value || 'admin';

        const adminPassword = this.$('#admin-password').value;
        if (adminPassword && adminPassword.trim()) {
            config.admin.password = adminPassword;
        }

        // 日志配置
        config.log.level = this.$('#log-level').value || 'info';
        config.log.retention_days = parseInt(this.$('#log-retention').value) || 30;
        config.log.max_size_mb = parseInt(this.$('#log-max-size').value) || 100;
        config.log.compress_days = parseInt(this.$('#log-compress').value) || 7;
        config.log.cleanup_hour = parseInt(this.$('#log-cleanup-hour').value) || 3;

        // 备份设置
        config.backup_dir = this.directoryPickers.backup?.getValue() || './backups';

        return config;
    }

    /**
     * 保存配置
     */
    async save() {
        if (!this.hasChanges && !this.$('#admin-password').value) {
            this.toast.info('没有需要保存的更改');
            return;
        }

        try {
            const config = this.collectConfig();
            await this.api.post('/api/config', config);
            
            this.originalConfig = JSON.parse(JSON.stringify(config));
            this.hasChanges = false;
            
            this.toast.success('配置保存成功');
            console.log('[Settings] 配置保存成功', config);

            // 触发配置更新事件
            this.events.emit('config:updated', config);
        } catch (error) {
            console.error('[Settings] 保存配置失败:', error);
            this.toast.error('保存失败：' + error.message);
        }
    }

    /**
     * 重置配置
     */
    reset() {
        this.dialog.confirm('确定要重置所有配置吗？未保存的更改将丢失。', () => {
            this.render();
            this.toast.success('配置已重置');
        });
    }

    /**
     * 刷新
     */
    async onRefresh() {
        await this.onLoad();
    }

    /**
     * 销毁
     */
    onDestroy() {
        // 清理目录选择器
        Object.values(this.directoryPickers).forEach(picker => {
            if (picker.destroy) picker.destroy();
        });
        this.directoryPickers = {};
    }
}

// 导出单例
export default new SettingsTab({
    api: window.api,
    state: window.state,
    toast: window.toast,
    message: window.message,
    dialog: window.dialog,
    events: window.events
});
