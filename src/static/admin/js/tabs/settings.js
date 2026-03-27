/**
 * 设置模块
 *
 * 负责系统设置的加载、编辑、保存
 */

import { BaseTab } from './BaseTab.js';
import { settingsValidator } from './settings-validator.js';
import { DirectoryPicker } from '../components/directory-picker.js';

class SettingsTab extends BaseTab {
    constructor(deps) {
        super(deps, 'settings');
        this.config = null;
        this.hasChanges = false;
        this.currentTab = 'panel';
        this.directoryPickers = {};
    }

    onInit() {
        console.log('初始化设置面板...');
        this.bindEvents();
        this.initValidation();
    }

    initDirectoryPickers() {
        // 防止重复初始化
        if (this.directoryPickersInitialized) {
            return;
        }
        
        console.log('[Settings] 初始化目录选择器...');
        
        // WEB 根目录选择器
        const webRootContainer = this.$('#web-root-picker');
        if (webRootContainer) {
            webRootContainer.innerHTML = '';
            this.directoryPickers.webRoot = new DirectoryPicker({
                container: webRootContainer,
                api: this.api,
                apiPath: '/api/files',
                placeholder: './web',
                onChange: (path) => { this.hasChanges = true; }
            });
            console.log('[Settings] WEB 根目录选择器已创建', webRootContainer.innerHTML);
        } else {
            console.warn('[Settings] WEB 根目录容器未找到');
        }

        // FTP 根目录选择器
        const ftpRootContainer = this.$('#ftp-root-picker');
        if (ftpRootContainer) {
            ftpRootContainer.innerHTML = '';
            this.directoryPickers.ftpRoot = new DirectoryPicker({
                container: ftpRootContainer,
                api: this.api,
                apiPath: '/api/files',
                placeholder: './ftp',
                onChange: (path) => { this.hasChanges = true; }
            });
            console.log('[Settings] FTP 根目录选择器已创建', ftpRootContainer.innerHTML);
        } else {
            console.warn('[Settings] FTP 根目录容器未找到');
        }

        // 备份目录选择器
        const backupDirContainer = this.$('#backup-dir-picker');
        if (backupDirContainer) {
            backupDirContainer.innerHTML = '';
            this.directoryPickers.backupDir = new DirectoryPicker({
                container: backupDirContainer,
                api: this.api,
                apiPath: '/api/files',
                placeholder: './backups',
                onChange: (path) => { this.hasChanges = true; }
            });
            console.log('[Settings] 备份目录选择器已创建', backupDirContainer.innerHTML);
        } else {
            console.warn('[Settings] 备份目录容器未找到');
        }
        
        this.directoryPickersInitialized = true;
    }

    async onLoad() {
        // 先初始化目录选择器（组件已加载到 DOM）
        this.initDirectoryPickers();
        await this.loadConfig();
    }

    bindEvents() {
        // Tab 切换
        this.$$('.settings-tab').forEach(tab => {
            tab.addEventListener('click', () => {
                this.switchTab(tab.dataset.tab);
            });
        });

        // 保存
        this.$('#save-settings')?.addEventListener('click', () => this.save());

        // 恢复默认
        this.$('#reset-settings')?.addEventListener('click', () => {
            this.dialog.confirm('确定要恢复默认设置吗？', () => this.reset());
        });

        // 密码显示/隐藏
        this.$('#toggle-password')?.addEventListener('click', () => {
            const input = this.$('#admin-password');
            if (input) {
                input.type = input.type === 'password' ? 'text' : 'password';
            }
        });

        // 生成密码
        this.$('#generate-password')?.addEventListener('click', () => {
            this.$('#admin-password').value = this.generatePassword();
        });

        // 表单变化追踪
        this.$$('.settings-form input, .settings-form select').forEach(input => {
            input.addEventListener('change', () => { this.hasChanges = true; });
        });

        // 端口输入数字控制
        this.$$('.port-input').forEach(input => {
            input.addEventListener('input', (e) => {
                e.target.value = e.target.value.replace(/\D/g, '');
            });
        });
    }

    initValidation() {
        settingsValidator.initRealtimeValidation();
    }

    switchTab(tabName) {
        this.currentTab = tabName;

        // 更新 Tab 样式
        this.$$('.settings-tab').forEach(t => t.classList.remove('active'));
        this.$(`.settings-tab[data-tab="${tabName}"]`)?.classList.add('active');

        // 更新内容显示（ID 格式：settings-{tabName}）
        this.$$('.settings-content').forEach(c => c.classList.remove('active'));
        this.$(`#settings-${tabName}`)?.classList.add('active');
        
        // 切换到目录设置时，确保目录选择器已初始化
        if (tabName === 'directory' && !this.directoryPickers.webRoot) {
            this.initDirectoryPickers();
        }
    }

    async loadConfig() {
        try {
            this.config = await this.api.getJSON('/api/config');
            this.populateForm();
        } catch (error) {
            this.toast.error('加载配置失败: ' + error.message);
        }
    }

    populateForm() {
        if (!this.config) return;

        // Admin
        this.setInputValue('admin-username', this.config.admin?.username);
        this.setInputValue('admin-password', this.config.admin?.password);
        this.setInputValue('admin-port', this.config.admin?.port);
        this.setInputValue('admin-path', this.config.admin?.path);

        // FTP
        this.directoryPickers.ftpRoot?.setValue(this.config.ftp?.root || './ftp');

        // 日志设置
        const log = this.config.log || {};
        this.setInputValue('log-retention-days', log.retention_days || 30);
        this.setInputValue('log-max-size-mb', log.max_size_mb || 100);
        this.setInputValue('log-compress-days', log.compress_days || 7);
        this.setInputValue('log-cleanup-hour', log.cleanup_hour || 3);
        this.setInputValue('log-level', log.level || 'info');
        
        // 分类日志级别
        if (log.levels) {
            this.setInputValue('log-level-http', log.levels.http || '');
            this.setInputValue('log-level-ftp', log.levels.ftp || '');
            this.setInputValue('log-level-panel', log.levels.panel || '');
        }

        // 目录设置
        this.directoryPickers.webRoot?.setValue(this.config.http?.root || './web');
        this.directoryPickers.backupDir?.setValue(this.config.backup_dir || './backups');

        this.hasChanges = false;
    }

    setInputValue(id, value) {
        const el = this.$(`#${id}`);
        if (!el) return;

        if (el.type === 'checkbox') {
            el.checked = !!value;
        } else {
            el.value = value ?? '';
        }
    }

    getInputValue(id) {
        const el = this.$(`#${id}`);
        if (!el) return null;

        if (el.type === 'checkbox') {
            return el.checked;
        }
        return el.value;
    }

    collectFormData() {
        // 收集分类日志级别
        const logLevels = {};
        const httpLevel = this.getInputValue('log-level-http');
        const ftpLevel = this.getInputValue('log-level-ftp');
        const panelLevel = this.getInputValue('log-level-panel');
        if (httpLevel) logLevels.http = httpLevel;
        if (ftpLevel) logLevels.ftp = ftpLevel;
        if (panelLevel) logLevels.panel = panelLevel;

        return {
            http: {
                enabled: true,
                port: 8080,
                root: this.directoryPickers.webRoot?.getValue() || './web'
            },
            ftp: {
                enabled: this.config?.ftp?.enabled || false,
                port: this.config?.ftp?.port || 2121,
                root: this.directoryPickers.ftpRoot?.getValue() || './ftp'
            },
            admin: {
                username: this.getInputValue('admin-username') || 'admin',
                password: this.getInputValue('admin-password') || '',
                port: parseInt(this.getInputValue('admin-port')) || 9527,
                path: this.getInputValue('admin-path') || '/admin'
            },
            log: {
                retention_days: parseInt(this.getInputValue('log-retention-days')) || 30,
                max_size_mb: parseInt(this.getInputValue('log-max-size-mb')) || 100,
                compress_days: parseInt(this.getInputValue('log-compress-days')) || 7,
                cleanup_hour: parseInt(this.getInputValue('log-cleanup-hour')) || 3,
                level: this.getInputValue('log-level') || 'info',
                levels: logLevels
            },
            backup_dir: this.directoryPickers.backupDir?.getValue() || './backups'
        };
    }

    async save() {
        const data = this.collectFormData();

        // 验证
        const errors = settingsValidator.validate(data);
        if (errors.length > 0) {
            this.toast.error(errors[0]);
            return;
        }

        try {
            await this.api.post('/api/config', data);
            this.toast.success('设置已保存');
            this.hasChanges = false;
            await this.loadConfig();
        } catch (error) {
            this.toast.error('保存失败: ' + error.message);
        }
    }

    async reset() {
        try {
            await this.api.post('/api/config/reset');
            this.toast.success('已恢复默认设置');
            await this.loadConfig();
        } catch (error) {
            this.toast.error('重置失败: ' + error.message);
        }
    }

    generatePassword(length = 16) {
        const charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
        let password = '';
        for (let i = 0; i < length; i++) {
            password += charset.charAt(Math.floor(Math.random() * charset.length));
        }
        return password;
    }

    // 提示未保存
    onDestroy() {
        if (this.hasChanges) {
            // 可以提示用户保存
        }
    }
}

// 单例
let instance = null;

export function initSettingsTab(deps) {
    if (!instance) {
        instance = new SettingsTab(deps);
        instance.init();
    }
    return instance;
}