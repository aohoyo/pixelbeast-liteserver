/** SSL 证书管理 API */
import api from '../client'
import type { CertInfo, CertProgress, DNSProvider } from '@/types/cert'

export function listCerts(): Promise<CertInfo[]> {
  return api.get<CertInfo[]>('/api/certs')
}

/** HTTP-01 自动申请（autocert） */
export function requestCert(domain: string, email: string, challengeMethod = 'http-auto'): Promise<void> {
  return api.post('/api/certs/request', { domain, email, challenge_method: challengeMethod })
}

export function renewCert(domain: string): Promise<void> {
  return api.post('/api/certs/renew', { domain })
}

export function deleteCert(domain: string): Promise<void> {
  return api.post('/api/certs/delete', { domain })
}

export function deployCert(domain: string, siteIds: string[]): Promise<{ deployed: number }> {
  return api.post('/api/certs/deploy', { domain, site_ids: siteIds })
}

/** 粘贴证书 */
export function pasteCert(domain: string, certPem: string, keyPem: string): Promise<CertInfo> {
  return api.post('/api/certs/paste', { domain, cert_pem: certPem, key_pem: keyPem })
}

/** 上传证书（multipart） */
export function uploadCert(domain: string, certFile: File, keyFile: File): Promise<void> {
  const form = new FormData()
  form.append('domain', domain)
  form.append('cert_file', certFile)
  form.append('key_file', keyFile)
  return api.raw.post('/api/certs/upload', form).then((res: { data: { code: number; message: string } }) => {
    if (res.data.code !== 200) throw new Error(res.data.message)
  })
}

/** 获取申请进度（轮询） */
export function getCertProgress(domain: string): Promise<CertProgress | null> {
  return api.get<CertProgress | null>(`/api/certs/progress/${encodeURIComponent(domain)}`)
}

// DNS-01 流程
export function dnsPrepare(data: {
  domain: string
  email: string
  dns_provider?: string
  dns_provider_id?: string
  dns_credentials?: Record<string, string>
}): Promise<{ fqdn: string; value: string; record_type: string }> {
  return api.post('/api/certs/dns-prepare', data)
}

export function dnsComplete(domain: string): Promise<void> {
  return api.post('/api/certs/dns-complete', { domain })
}

// HTTP-01 文件验证流程
export function filePrepare(domain: string, email: string): Promise<{
  token: string
  key_auth: string
  url_path: string
  verify_url: string
}> {
  return api.post('/api/certs/file-prepare', { domain, email })
}

export function fileComplete(domain: string): Promise<void> {
  return api.post('/api/certs/file-complete', { domain })
}

// DNS 服务商管理
export function listDNSProviders(): Promise<DNSProvider[]> {
  return api.get<DNSProvider[]>('/api/certs/dns-providers')
}

export function addDNSProvider(data: {
  name: string
  type: string
  credentials: Record<string, string>
}): Promise<{ id: string }> {
  return api.post('/api/certs/dns-providers', data)
}

export function deleteDNSProvider(id: string): Promise<void> {
  return api.delete(`/api/certs/dns-providers/${id}`)
}

export function testDNSProvider(data: {
  type: string
  credentials: Record<string, string>
}): Promise<{ success: boolean; message: string }> {
  return api.post('/api/certs/dns-providers-test', data)
}
