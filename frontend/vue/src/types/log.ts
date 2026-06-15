/** 日志文件信息 */
export interface LogFileInfo {
  name: string
  category: 'http' | 'ftp' | 'panel'
  type: string
  size: number
  modified_at: string
  compressed: boolean
}

export interface LogEntry {
  timestamp: string
  level?: string
  message: string
  raw: string
}

export interface LogReadResponse {
  entries: LogEntry[]
  total: number
  file: string
}

export interface LogStats {
  category: string
  type: string
  count: number
  errors: number
  warnings: number
}
