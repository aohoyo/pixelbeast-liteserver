/** 系统监控状态（GET /api/system/status 返回） */
export interface DiskPartition {
  mount: string
  device: string
  fstype: string
  total_gb: number
  used_gb: number
  free_gb: number
  percent: number
}

export interface SystemStatus {
  cpu_percent: number
  cpu_cores: number
  cpu_threads: number
  cpu_model: string
  cpu_history: number[]
  cpu_per_core: number[]
  memory_percent: number
  memory_used_gb: number
  memory_total_gb: number
  memory_free_gb: number
  memory_available_gb: number
  swap_percent: number
  swap_used_gb: number
  swap_total_gb: number
  disk_percent: number
  disk_used_gb: number
  disk_total_gb: number
  disk_mount: string
  disks: DiskPartition[]
  load_avg: number[]
  process_active: number
  process_total: number
  server_uptime_ms: number
  system_uptime_ms: number
  net_sent_rate_kb: number
  net_recv_rate_kb: number
  net_total_sent_gb: number
  net_total_recv_gb: number
  diskio_speed_write_kb: number
  diskio_speed_read_kb: number
  diskio_iops: number
  diskio_latency_ms: number
  sites_running: boolean
  sites_enabled: number
  sites_count: number
  ftp_running: boolean
  ftp_users: number
  os: string
  arch: string
  os_name: string
  os_name_short: string
  kernel: string
  hostname: string
  csrf_token?: string
}

export interface CleanupItem {
  name: string
  desc: string
  size_mb: number
  count: number
  path: string
  cleanable: boolean
  need_admin: boolean
}

export interface CleanupScanResponse {
  items: CleanupItem[]
  total_mb: number
  message?: string
}

export interface UpdateInfo {
  current_version: string
  latest_version: string
  has_update: boolean
  changelog?: string
  download_url?: string
  message: string
}
