/**
 * 设置页面表单验证模块
 *
 * 验证端口和域名格式
 */

// 验证端口号 (1-65535)
export function validatePort(value) {
    if (!value || value.trim() === '') {
        return { valid: false, message: '端口不能为空' };
    }

    const port = parseInt(value, 10);

    if (isNaN(port)) {
        return { valid: false, message: '请输入有效的端口号' };
    }

    if (port < 1 || port > 65535) {
        return { valid: false, message: '端口必须在 1-65535 之间' };
    }

    return { valid: true, message: '' };
}

// 验证域名或 IP 地址
export function validateDomain(value) {
    // 空值允许（表示不使用域名绑定）
    if (!value || value.trim() === '') {
        return { valid: true, message: '' };
    }

    const domain = value.trim();

    // IP 地址格式
    const ipPattern = /^(\d{1,3}\.){3}\d{1,3}$/;
    if (ipPattern.test(domain)) {
        const parts = domain.split('.');
        for (const part of parts) {
            if (parseInt(part) > 255) {
                return { valid: false, message: 'IP 地址段超出有效范围 (0-255)' };
            }
        }
        return { valid: true, message: '' };
    }

    // 域名格式
    const domainPattern = /^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)+$/;
    if (!domainPattern.test(domain)) {
        return { valid: false, message: '请输入有效的域名或 IP 地址' };
    }

    return { valid: true, message: '' };
}

// 显示字段错误
export function showFieldError(fieldId, message) {
    const input = document.getElementById(fieldId);
    if (!input) return;

    // 清除旧错误
    clearFieldError(fieldId);

    // 添加错误 class
    input.classList.add('input-error');

    // 创建错误提示（放在输入框后面）
    const errorEl = document.createElement('div');
    errorEl.className = 'field-error';
    errorEl.textContent = message;

    // 插入到输入框后面，但在提示文本之前
    input.parentElement.insertBefore(errorEl, input.nextSibling);
}

// 清除字段错误
export function clearFieldError(fieldId) {
    const input = document.getElementById(fieldId);
    if (!input) return;

    // 移除错误 class
    input.classList.remove('input-error');

    // 移除错误提示元素
    const errorEl = input.parentElement?.querySelector('.field-error');
    if (errorEl) {
        errorEl.remove();
    }
}

// 验证所有字段（返回第一个错误）
export function validateAll() {
    const portInput = document.getElementById('admin-port');
    const domainInput = document.getElementById('admin-domain');

    // 验证端口
    if (portInput) {
        const portResult = validatePort(portInput.value);
        if (!portResult.valid) {
            return portResult;
        }
        clearFieldError('admin-port');
    }

    // 验证域名
    if (domainInput) {
        const domainResult = validateDomain(domainInput.value);
        if (!domainResult.valid) {
            return domainResult;
        }
        clearFieldError('admin-domain');
    }

    return { valid: true, message: '' };
}

// 验证配置对象（用于保存前验证）
export function validateConfig(config) {
    const errors = [];
    
    // 验证面板端口
    if (config.Global?.AdminPort) {
        if (config.Global.AdminPort < 1 || config.Global.AdminPort > 65535) {
            errors.push('面板端口必须在 1-65535 之间');
        }
    }
    
    // 验证管理员用户名
    if (config.Admin?.Username !== undefined) {
        if (!config.Admin.Username || config.Admin.Username.trim() === '') {
            errors.push('管理员用户名不能为空');
        }
    }
    
    // 验证日志保留天数
    if (config.Log?.RetentionDays !== undefined) {
        if (config.Log.RetentionDays < 1 || config.Log.RetentionDays > 365) {
            errors.push('日志保留天数必须在 1-365 之间');
        }
    }
    
    // 验证日志文件大小
    if (config.Log?.MaxSizeMB !== undefined) {
        if (config.Log.MaxSizeMB < 1 || config.Log.MaxSizeMB > 1000) {
            errors.push('日志文件大小必须在 1-1000 MB 之间');
        }
    }
    
    return errors;
}

// 创建 debounce 函数
function debounce(fn, delay) {
    let timer;
    return function(...args) {
        clearTimeout(timer);
        timer = setTimeout(() => fn.apply(this, args), delay);
    };
}

// 初始化实时验证
export function initRealtimeValidation() {
    const portInput = document.getElementById('admin-port');
    const domainInput = document.getElementById('admin-domain');

    // 端口实时验证
    if (portInput) {
        const validatePortDebounced = debounce(() => {
            const result = validatePort(portInput.value);
            if (!result.valid) {
                showFieldError('admin-port', result.message);
            } else {
                clearFieldError('admin-port');
            }
        }, 300);

        portInput.addEventListener('input', validatePortDebounced);
    }

    // 域名实时验证
    if (domainInput) {
        const validateDomainDebounced = debounce(() => {
            const result = validateDomain(domainInput.value);
            if (!result.valid) {
                showFieldError('admin-domain', result.message);
            } else {
                clearFieldError('admin-domain');
            }
        }, 500);

        domainInput.addEventListener('input', validateDomainDebounced);
    }
}

// 导出验证器对象
export const settingsValidator = {
    validatePort,
    validateDomain,
    validateAll,
    validate: validateConfig,
    showFieldError,
    clearFieldError,
    initRealtimeValidation
};

export default settingsValidator;