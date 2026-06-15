/** 备份管理 API */
import api from '../client'
import type { BackupListResponse } from '@/types/backup'

export function listBackups(): Promise<BackupListResponse> {
  return api.get<BackupListResponse>('/api/backups')
}

export function createBackup(): Promise<void> {
  return api.post('/api/backups/create')
}

export function deleteBackup(name: string): Promise<void> {
  return api.post('/api/backups/delete', { name })
}

export function restoreBackup(name: string): Promise<void> {
  return api.post('/api/backups/restore', { name })
}

// 下载：返回原始 blob
export function downloadBackupUrl(name: string): string {
  return `/api/backups/download?name=${encodeURIComponent(name)}`
}
