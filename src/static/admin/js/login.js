/**
 * 登录页面脚本
 */

(function() {
    const loginForm = document.getElementById('loginForm');
    const loginBtn = document.getElementById('loginBtn');
    const errorMessage = document.getElementById('errorMessage');
    const warningBox = document.getElementById('warningBox');

    // 检查是否有警告信息
    const urlParams = new URLSearchParams(window.location.search);
    const warning = urlParams.get('warning');
    if (warning) {
        warningBox.textContent = decodeURIComponent(warning);
        warningBox.classList.add('show');
    }

    // 登录函数
    async function doLogin() {
        loginBtn.disabled = true;
        errorMessage.classList.remove('show');

        // 手动获取表单值
        const username = document.getElementById('username').value.trim();
        const password = document.getElementById('password').value;

        try {
            const formData = new URLSearchParams();
            formData.append('username', username);
            formData.append('password', password);

            const response = await fetch('api/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: formData.toString()
            });

            const data = await response.json();

            // 新格式：{code: 200, message: "success"} 或 {code: 401, message: "...", data: {remaining: N}}
            if (data.code === 200) {
                window.location.href = './';
            } else {
                // 显示错误信息，如果有 remaining 则显示剩余次数
                let msg = data.message || '登录失败';
                if (data.data && data.data.remaining !== undefined) {
                    msg += ` (剩余尝试次数: ${data.data.remaining})`;
                }
                errorMessage.textContent = msg;
                errorMessage.classList.add('show');
                loginBtn.disabled = false;
            }
        } catch (error) {
            errorMessage.textContent = '网络错误，请重试';
            errorMessage.classList.add('show');
            loginBtn.disabled = false;
        }
    }

    // 回车键提交
    loginForm.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            doLogin();
        }
    });

    // 暴露登录函数到全局
    window.doLogin = doLogin;
})();
