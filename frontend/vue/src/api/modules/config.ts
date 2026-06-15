/** 配置管理 API */
import api from '../client'
import type { ServerConfig } from '@/types/config'

export function getConfig(): Promise<ServerConfig> {
  return api.get<ServerConfig>('/api/config')
}

export function saveConfig(data: ServerConfig): Promise<void> {
  return api.post('/api/config/save', data)
}

export function resetConfig(): Promise<ServerConfig> {
  return api.post<ServerConfig>('/api/config/reset')
}

export function getLogConfig() {
  return api.get('/api/logs/config')
}

export function saveLogConfig(data: { retention_days?: number; max_size_mb?: number; compress_days?: number; level?: string }) {
  return api.post('/api/logs/config', data)
}
