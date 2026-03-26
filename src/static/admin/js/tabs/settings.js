/**
 * 设置模块
 *
 * 负责系统设置的加载、编辑、保存
 */

import { BaseTab } from './BaseTab.js';
import { settingsValidator } from './settings-validator.js';

class SettingsTab extends BaseTab {
    constructor(deps) {
        super(deps, 'settings');
        this.config = null;
        this.hasChanges = false;
        this.currentTab = 'panel';
    }

    onInit() {
        console.log('初始化设置面板...');
        this.bindEvents();
        this.initValidation();
    }

    async onLoad() {
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

        // 更新内容显示
        this.$$('.settings-content').forEach(c => c.classList.remove('active'));
        this.$(`#${tabName}-settings`)?.classList.add('active');
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

        // HTTP
        this.setInputValue('http-enabled', this.config.http?.enabled);
        this.setInputValue('http-port', this.config.http?.port);
        this.setInputValue('http-root', this.config.http?.root);

        // FTP
        this.setInputValue('ftp-enabled', this.config.ftp?.enabled);
        this.setInputValue('ftp-port', this.config.ftp?.port);
        this.setInputValue('ftp-root', this.config.ftp?.root);

        // Admin
        this.setInputValue('admin-username', this.config.admin?.username);
        this.setInputValue('admin-password', this.config.admin?.password);
        this.setInputValue('admin-port', this.config.admin?.port);

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
        return {
            http: {
                enabled: this.getInputValue('http-enabled'),
                port: parseInt(this.getInputValue('http-port')) || 8080,
                root: this.getInputValue('http-root') || './web'
            },
            ftp: {
                enabled: this.getInputValue('ftp-enabled'),
                port: parseInt(this.getInputValue('ftp-port')) || 2121,
                root: this.getInputValue('ftp-root') || './ftp'
            },
            admin: {
                username: this.getInputValue('admin-username') || 'admin',
                password: this.getInputValue('admin-password') || 'admin123',
                port: parseInt(this.getInputValue('admin-port')) || 9527
            }
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