/** 服务配置（镜像 config.ServerConfig） */
export interface AdminConfig {
  port: number
  username: string
  password: string // GET 时为空，保存时空=不改
  path: string
  domain: string
  ssl_enabled: boolean
  require_password_change: boolean
}

export interface DirectoriesConfig {
  sites: string
  ftp: string
  backup: string
}

export interface ShareConfig {
  allowed_dirs: string[]
}

export interface BackupConfig {
  auto_enabled: boolean
  schedule: 'daily' | 'weekly' | 'monthly'
  retention: number
  items: string[]
}

export interface LogConfig {
  retention_days: number
  max_size_mb: number
  compress_days: number
  cleanup_hour: number
  level: string
  levels?: Record<string, string>
}

export interface AutoStartConfig {
  enabled: boolean
}

export interface ServerConfig {
  name: string
  timezone: string
  admin: AdminConfig
  directories: DirectoriesConfig
  share: ShareConfig
  backup: BackupConfig
  log: LogConfig
  auto_start: AutoStartConfig
}
