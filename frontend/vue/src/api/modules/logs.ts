/** 日志管理 API */
import api from '../client'
import type { LogFileInfo, LogReadResponse, LogStats } from '@/types/log'

export interface LogReadParams {
  category?: string
  type?: string
  date?: string
  search?: string
  level?: string
  limit?: number
}

export function listLogs(): Promise<LogFileInfo[]> {
  return api.get<LogFileInfo[]>('/api/logs')
}

export function readLogs(params: LogReadParams): Promise<LogReadResponse> {
  return api.get<LogReadResponse>('/api/logs/read', { params })
}

export function getLogsStats(params: LogReadParams): Promise<LogStats[]> {
  return api.get<LogStats[]>('/api/logs/stats', { params })
}

export function clearLog(category: string, type: string): Promise<void> {
  return api.post(`/api/logs/clear?category=${category}&type=${type}`)
}

export function bulkClearLogs(data: { category?: string; before_date?: string; compressed?: boolean }): Promise<void> {
  return api.post('/api/logs/bulk-clear', data)
}
