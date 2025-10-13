// 工具函数
function showMessage(message, type = 'info') {
    const messageEl = document.getElementById('message');
    messageEl.textContent = message;
    messageEl.className = `message ${type}`;
    messageEl.classList.add('show');
    
    setTimeout(() => {
        messageEl.classList.remove('show');
    }, 5000);
}

function setButtonLoading(button, loading) {
    if (loading) {
        button.disabled = true;
        button.classList.add('loading');
        button.dataset.originalText = button.textContent;
        button.textContent = button.textContent.replace(/登录|注册/, loading === 'login' ? '登录中...' : '注册中...');
    } else {
        button.disabled = false;
        button.classList.remove('loading');
        if (button.dataset.originalText) {
            button.textContent = button.dataset.originalText;
        }
    }
}

// API 请求函数
async function apiRequest(url, data) {
    try {
        const response = await fetch(url, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(data)
        });

        const result = await response.json();
        
        if (!response.ok) {
            throw new Error(result.error || result.details || '请求失败');
        }
        
        return result;
    } catch (error) {
        throw new Error(error.message || '网络请求失败');
    }
}

// 处理认证成功
function handleAuthSuccess(tokens) {
    // 无论什么情况都先存储 tokens 到 cookie
    setCookie('access_token', tokens.access_token, 7); // 7天过期
    setCookie('refresh_token', tokens.refresh_token, 7); // 7天过期
    
    // 检查是否来自 /auth/local
    const authSource = getCookie('auth_source');
    
    if (authSource === 'local') {
        // 清除 auth_source cookie
        document.cookie = 'auth_source=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
        
        // 跳转到 /auth/local/callback 并携带 tokens
        const callbackUrl = `/auth/local/callback?access_token=${encodeURIComponent(tokens.access_token)}&refresh_token=${encodeURIComponent(tokens.refresh_token)}`;
        showMessage('认证成功！正在跳转...', 'success');
        setTimeout(() => {
            window.location.href = callbackUrl;
        }, 1500);
    } else {
        showMessage('认证成功！正在跳转...', 'success');
        
        // 跳转到主页或仪表板
        setTimeout(() => {
            window.location.href = '/dashboard'; // 可以根据需要修改跳转地址
        }, 1500);
    }
}

// 获取 cookie 的辅助函数
function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
    return null;
}

// 设置 cookie 的辅助函数
function setCookie(name, value, days) {
    let expires = "";
    if (days) {
        const date = new Date();
        date.setTime(date.getTime() + (days * 24 * 60 * 60 * 1000));
        expires = "; expires=" + date.toUTCString();
    }
    document.cookie = name + "=" + (value || "") + expires + "; path=/";
}

// 登录表单处理
document.addEventListener('DOMContentLoaded', function() {
    const loginForm = document.getElementById('loginForm');
    const registerForm = document.getElementById('registerForm');

    // 登录表单
    if (loginForm) {
        loginForm.addEventListener('submit', async function(e) {
            e.preventDefault();
            
            const submitBtn = this.querySelector('button[type="submit"]');
            const formData = new FormData(this);
            
            const data = {
                email: formData.get('email'),
                password: formData.get('password')
            };

            // 验证表单
            if (!data.email || !data.password) {
                showMessage('请填写所有必填字段', 'error');
                return;
            }

            try {
                setButtonLoading(submitBtn, 'login');
                const result = await apiRequest('/auth/local/login', data);
                handleAuthSuccess(result);
            } catch (error) {
                showMessage(error.message, 'error');
            } finally {
                setButtonLoading(submitBtn, false);
            }
        });
    }

    // 注册表单
    if (registerForm) {
        registerForm.addEventListener('submit', async function(e) {
            e.preventDefault();
            
            const submitBtn = this.querySelector('button[type="submit"]');
            const formData = new FormData(this);
            
            const data = {
                name: formData.get('name'),
                email: formData.get('email'),
                password: formData.get('password')
            };

            const confirmPassword = formData.get('confirmPassword');

            // 验证表单
            if (!data.name || !data.email || !data.password || !confirmPassword) {
                showMessage('请填写所有必填字段', 'error');
                return;
            }

            if (data.password !== confirmPassword) {
                showMessage('两次输入的密码不一致', 'error');
                return;
            }

            if (data.password.length < 6) {
                showMessage('密码至少需要6位字符', 'error');
                return;
            }

            try {
                setButtonLoading(submitBtn, 'register');
                const result = await apiRequest('/auth/local/register', data);
                handleAuthSuccess(result);
            } catch (error) {
                showMessage(error.message, 'error');
            } finally {
                setButtonLoading(submitBtn, false);
            }
        });
    }

    // 检查URL参数中的错误信息（用于OAuth回调错误处理）
    const urlParams = new URLSearchParams(window.location.search);
    const error = urlParams.get('error');
    const errorDescription = urlParams.get('error_description');
    
    if (error) {
        showMessage(errorDescription || '认证失败', 'error');
    }

    // 检查URL参数中的token（用于OAuth回调成功处理）
    const accessToken = urlParams.get('access_token');
    const refreshToken = urlParams.get('refresh_token');
    
    if (accessToken && refreshToken) {
        handleAuthSuccess({
            access_token: accessToken,
            refresh_token: refreshToken
        });
    }
});