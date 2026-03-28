/**
 * 设置验证器
 *
 * 提供配置字段的验证规则
 */

export const settingsValidator = {
    /**
     * 验证端口号
     */
    port(value) {
        const port = parseInt(value);
        if (isNaN(port) || port < 1 || port > 65535) {
            return { valid: false, message: '端口必须在 1-65535 之间' };
        }
        return { valid: true };
    },

    /**
     * 验证路径
     */
    path(value) {
        if (!value || value.trim() === '') {
            return { valid: false, message: '路径不能为空' };
        }
        return { valid: true };
    },

    /**
     * 验证用户名
     */
    username(value) {
        if (!value || value.trim().length < 3) {
            return { valid: false, message: '用户名至少 3 个字符' };
        }
        return { valid: true };
    },

    /**
     * 验证密码
     */
    password(value) {
        if (value && value.length < 6) {
            return { valid: false, message: '密码至少 6 个字符' };
        }
        return { valid: true };
    },

    /**
     * 验证数字范围
     */
    range(value, min, max) {
        const num = parseInt(value);
        if (isNaN(num) || num < min || num > max) {
            return { valid: false, message: `值必须在 ${min}-${max} 之间` };
        }
        return { valid: true };
    }
};
