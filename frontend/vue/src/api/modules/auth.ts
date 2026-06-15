/** 认证相关 API */
import api from '../client'

export interface LoginResponse {
  // 后端登录成功仅返回 message，无 data
}

/** 登录 —— 后端返回 {code:200, message:"登录成功"}，设置 session cookie */
export function login(username: string, password: string): Promise<LoginResponse> {
  return api.post<LoginResponse>('/api/login', { username, password })
}

/** 登出 */
export function logout(): Promise<void> {
  return api.post('/api/logout')
}

/**
 * 初始化 CSRF token —— 登录成功后必须调用一次。
 * 后端在 /api/system/status 的 data.csrf_token 中下发首个 CSRF token。
 */
export async function initCSRF(): Promise<void> {
  await api.get('/api/system/status')
  // CSRF token 已被响应拦截器自动提取到模块级变量
}
