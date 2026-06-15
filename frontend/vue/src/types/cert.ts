/** SSL 证书信息 */
export interface CertInfo {
  domain: string
  type: 'auto' | 'custom' | 'self-signed'
  enabled: boolean
  auto_https: boolean
  force_https: boolean
  hsts: boolean
  email: string
  issuer: string
  not_before: string
  not_after: string
  days_left: number
  cert_file: string
  key_file: string
  has_cert: boolean
  provider: string
  challenge_method: string
}

/** 证书申请进度（实时日志） */
export interface CertProgress {
  domain: string
  step: number
  step_text: string
  status: 'running' | 'success' | 'error' | 'waiting'
  logs: { time: string; message: string; level: string }[]
}

/** DNS 服务商 */
export interface DNSProvider {
  id: string
  name: string
  type: 'alidns' | 'tencentcloud' | 'baota'
  masked_creds: Record<string, string>
  created_at: string
  updated_at: string
}
