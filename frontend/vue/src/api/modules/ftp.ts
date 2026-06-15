/** FTP 管理 API */
import api from '../client'
import type { FtpUser, FtpUsersResponse, FtpStatus } from '@/types/ftp'

export function listFtpUsers(): Promise<FtpUsersResponse> {
  return api.get<FtpUsersResponse>('/api/ftp/users')
}

export function addFtpUser(data: Partial<FtpUser> & { password: string }): Promise<void> {
  return api.post('/api/ftp/users/add', data)
}

export function updateFtpUser(username: string, data: Partial<FtpUser>): Promise<void> {
  return api.put(`/api/ftp/users/${username}`, data)
}

export function deleteFtpUser(username: string, deleteFiles = false): Promise<void> {
  return api.delete(`/api/ftp/users/${encodeURIComponent(username)}?deleteFiles=${deleteFiles}`)
}

export function toggleFtpUser(username: string, enabled: boolean): Promise<void> {
  return api.post('/api/ftp/users/toggle', { username, enabled })
}

export function batchFtpUsers(action: string, usernames: string[]): Promise<void> {
  return api.post('/api/ftp/users/batch', { action, usernames })
}

export function getFtpStatus(): Promise<FtpStatus> {
  return api.get<FtpStatus>('/api/ftp/status')
}

export function saveFtpPort(port: number): Promise<void> {
  return api.post('/api/ftp/port', { port })
}

// 服务控制
export const toggleFtpService = () => api.post('/api/service/ftp/toggle')
export const startFtpService = () => api.post('/api/service/ftp/start')
export const stopFtpService = () => api.post('/api/service/ftp/stop')
export const restartFtpService = () => api.post('/api/service/ftp/restart')
export const reloadFtpConfig = () => api.post('/api/service/ftp/reload')

export type { FtpUser, FtpStatus }
