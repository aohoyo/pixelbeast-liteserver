// 后端统一响应结构 {code, message, data}
export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data?: T
}

// 业务错误（统一响应里 code !== 200 时抛出）
export class ApiError extends Error {
  code: number
  constructor(code: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.code = code
  }
}
