/** 站点相关类型 —— 镜像 backend/internal/config/config.go 的 SiteConfig */

export type SiteType = 'static' | 'proxy'

/** 反向代理配置 */
export interface ProxyConfig {
  target: string
  strip_prefix?: string
  websocket?: boolean
  timeout?: number
}

/** SSL 配置 */
export interface SSLConfig {
  enabled: boolean
  auto_https?: boolean
  email?: string
  cert_file?: string
  key_file?: string
  force_https?: boolean
  hsts?: boolean
  provider?: string // "letsencrypt" | "litessl"
  challenge_method?: string // "http-auto" | "http-file" | "dns"
  dns_provider?: string
}

/** 站点（API 响应字段，与 backend siteToMap 对齐） */
export interface Site {
  id: string
  name: string
  enabled: boolean
  type: SiteType
  port: number
  domain: string[]
  root?: string
  index_files: string[]
  auto_index: boolean
  proxy?: ProxyConfig | null
  ssl?: SSLConfig | null
  created_at: string
  updated_at: string
}

/** 站点服务状态 */
export interface SitesServiceStatus {
  running: boolean
  port: number
}
