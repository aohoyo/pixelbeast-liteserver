/** 站点管理 API —— 对齐 backend/internal/panel/api_site.go */
import api from '../client'
import type { Site, SiteType, SitesServiceStatus } from '@/types/site'

/** 站点列表（含 root 实际路径） */
export function listSites(): Promise<Site[]> {
  return api.get<Site[]>('/api/sites')
}

/** 创建站点（POST /api/sites） */
export function createSite(data: Partial<Site>): Promise<Site> {
  return api.post<Site>('/api/sites', data)
}

/** 更新站点（PUT /api/sites/{id}） */
export function updateSite(id: string, data: Partial<Site>): Promise<void> {
  return api.put(`/api/sites/${id}`, data)
}

/** 删除站点（DELETE /api/sites/{id}） */
export function deleteSite(id: string): Promise<void> {
  return api.delete(`/api/sites/${id}`)
}

/** 切换启用/禁用 */
export function toggleSite(id: string, enabled: boolean): Promise<void> {
  return api.post('/api/sites/toggle', { id, enabled })
}

/** 启动站点 */
export function startSite(id: string): Promise<void> {
  return api.post('/api/sites/start', { id })
}

/** 停止站点 */
export function stopSite(id: string): Promise<void> {
  return api.post('/api/sites/stop', { id })
}

/** 重启站点 */
export function restartSite(id: string): Promise<void> {
  return api.post('/api/sites/restart', { id })
}

/** 批量操作 action: enable | disable | delete */
export function batchSites(action: string, ids: string[]): Promise<void> {
  return api.post('/api/sites/batch', { action, ids })
}

/** 站点服务状态 */
export function getSitesStatus(): Promise<SitesServiceStatus> {
  return api.get<SitesServiceStatus>('/api/sites/status')
}

// 站点服务控制（全局，非单站点）
export function startSitesService(): Promise<void> {
  return api.post('/api/service/sites/start')
}
export function stopSitesService(): Promise<void> {
  return api.post('/api/service/sites/stop')
}
export function restartSitesService(): Promise<void> {
  return api.post('/api/service/sites/restart')
}
export function reloadSitesConfig(): Promise<void> {
  return api.post('/api/service/sites/reload')
}

export type { Site, SiteType }
