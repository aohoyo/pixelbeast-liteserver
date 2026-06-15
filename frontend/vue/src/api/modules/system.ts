/** 系统监控 API */
import api from '../client'
import type { SystemStatus, CleanupScanResponse, UpdateInfo } from '@/types/system'

export function getSystemStatus(): Promise<SystemStatus> {
  return api.get<SystemStatus>('/api/system/status')
}

export function freeMemory(): Promise<Record<string, unknown>> {
  return api.post('/api/system/free-memory')
}

export function scanCleanup(): Promise<CleanupScanResponse> {
  return api.get<CleanupScanResponse>('/api/system/cleanup-scan')
}

export function executeCleanup(items: string[]): Promise<void> {
  return api.post('/api/system/cleanup', { items })
}

export function checkUpdate(): Promise<UpdateInfo> {
  return api.get<UpdateInfo>('/api/system/check-update')
}
