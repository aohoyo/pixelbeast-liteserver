/** FTP 用户（注意：API 用 camelCase，非 config 的 snake_case） */
export interface FtpUser {
  username: string
  password: string // 列表返回 "••••••••"
  rootPath: string
  status: 'enabled' | 'disabled'
  quota: number // MB, 0=无限
  usedSpace: number // bytes
  expiryDays: number
  expiryDate: string
  remark: string
  speedLimit: number // KB/s 下载
  bandwidth: number // KB/s 上传
  maxConnections: number
  maxFiles: number
  maxFileSize: number // MB
}

export interface FtpUsersResponse {
  users: FtpUser[]
  total: number
}

export interface FtpStatus {
  running: boolean
  port: number
}
