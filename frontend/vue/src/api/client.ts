/**
 * API 客户端 —— 复刻后端约定（后端零改动，全部由前端适配）
 *
 * 后端契约（来自 backend/internal/panel/response.go + handler.go）：
 * - 统一响应 {code:200, message, data}，成功 code===200
 * - CSRF：登录后调 /api/system/status 返回 data.csrf_token，
 *         后续非 GET 请求需带 X-CSRF-Token 头
 * - 认证：Cookie session（credentials: 'include'），401 表示未登录
 * - 登录锁定：5 次失败返回 429
 */
import axios, {
  type AxiosInstance,
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
} from 'axios'
import { ElMessage } from 'element-plus'
import type { ApiResponse } from './types'
import { ApiError } from './types'

// CSRF token（模块级，自动管理）
let csrfToken: string | null = null

export function setCSRFToken(token: string | null) {
  csrfToken = token
}

export function getCSRFToken(): string | null {
  return csrfToken
}

// 认证失效回调（由 auth store 注入，避免循环依赖）
let onUnauthorized: (() => void) | null = null

export function setUnauthorizedHandler(handler: () => void) {
  onUnauthorized = handler
}

// 推导管理面板基础路径：后端安全入口要求所有请求带 adminPath 前缀
// SPA 运行在 /admin/ 下，window.location.pathname 的首段即为 adminPath
function getAdminBase(): string {
  const m = window.location.pathname.match(/^(\/[^/]+)/)
  return m ? m[1] : ''
}
const ADMIN_BASE = getAdminBase()

// 创建 axios 实例
const client: AxiosInstance = axios.create({
  // baseURL 为 adminPath：所有 /api/* 请求实际发往 /admin/api/*
  // dev 模式 Vite proxy 转发 /admin/api → 后端；prod 同源直连
  baseURL: ADMIN_BASE,
  withCredentials: true, // 携带 session cookie
  timeout: 30_000,
})

// 请求拦截器：非 GET 请求自动注入 CSRF token
client.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const method = (config.method || 'get').toLowerCase()
  if (method !== 'get' && method !== 'head' && csrfToken) {
    config.headers.set('X-CSRF-Token', csrfToken)
  }
  return config
})

// 响应拦截器：解包统一响应 + 自动提取 CSRF token + 错误处理
client.interceptors.response.use(
  (response) => {
    const body = response.data as ApiResponse

    // 自动提取 CSRF token（后端在部分响应的 data.csrf_token 中下发）
    if (body?.code === 200 && body.data && typeof body.data === 'object') {
      const maybeToken = (body.data as { csrf_token?: unknown }).csrf_token
      if (typeof maybeToken === 'string' && maybeToken) {
        csrfToken = maybeToken
      }
    }

    // 解包：成功返回 data 字段，失败抛 ApiError
    if (body && typeof body === 'object' && 'code' in body) {
      if (body.code === 200) {
        // 成功：返回 data（可能为 undefined，如仅 message 的响应）
        return body.data as never
      }
      // 业务错误
      throw new ApiError(body.code, body.message || '请求失败')
    }

    // 非统一格式（如文件下载的原始响应），原样返回
    return response.data as never
  },
  (error) => {
    // HTTP 层错误（非 2xx）
    if (error.response?.status === 401) {
      onUnauthorized?.()
      throw new ApiError(401, '未登录或会话已过期')
    }
    if (error.response?.status === 429) {
      ElMessage.error('登录失败次数过多，请稍后再试')
      throw new ApiError(429, '请求过多')
    }
    // 统一响应格式的 HTTP 错误（如 400/403/500）
    const body = error.response?.data as ApiResponse | undefined
    if (body && typeof body === 'object' && 'code' in body) {
      throw new ApiError(body.code, body.message || `请求失败 (${body.code})`)
    }
    // 网络/超时等
    const msg = error.code === 'ECONNABORTED' ? '请求超时' : '网络错误'
    ElMessage.error(msg)
    throw new ApiError(0, msg)
  },
)

// 对外暴露的便捷方法 —— 返回已解包的 data
export const api = {
  get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return client.get(url, config) as unknown as Promise<T>
  },
  post<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    return client.post(url, data, config) as unknown as Promise<T>
  },
  put<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    return client.put(url, data, config) as unknown as Promise<T>
  },
  delete<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return client.delete(url, config) as unknown as Promise<T>
  },
  // 原始 axios 实例（文件下载等需要原始 Response 的场景）
  raw: client,
}

export default api
