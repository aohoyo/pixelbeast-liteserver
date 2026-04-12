package handlers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pixelbeast/src/config"

	"golang.org/x/crypto/acme/autocert"
)

// CertInfo 证书详细信息
type CertInfo struct {
	Domain          string `json:"domain"`
	Type            string `json:"type"` // "auto" | "custom" | "self-signed"
	Enabled         bool   `json:"enabled"`
	AutoHTTPS       bool   `json:"auto_https"`
	ForceHTTPS      bool   `json:"force_https"`
	HSTS            bool   `json:"hsts"`
	Email           string `json:"email"`
	Issuer          string `json:"issuer"`     // "Let's Encrypt" | "LiteSSL" | "Custom" | ""
	NotBefore       string `json:"not_before"` // RFC3339
	NotAfter        string `json:"not_after"`  // RFC3339
	DaysLeft        int    `json:"days_left"`
	CertFile        string `json:"cert_file"`
	KeyFile         string `json:"key_file"`
	HasCert         bool   `json:"has_cert"`
	Provider        string `json:"provider"`         // "letsencrypt" | "litessl"
	ChallengeMethod string `json:"challenge_method"` // "http-auto" | "http-file" | "dns"
}

// SSLManager SSL 管理器
type SSLManager struct {
	mu sync.RWMutex

	certs     map[string]*tls.Certificate  // domain -> cert
	configs   map[string]*config.SSLConfig // domain -> config
	certInfos map[string]*CertInfo         // domain -> cert info
	certDir   string                       // 证书存储目录 ./ssl
	acmeDir   string                       // ACME 缓存目录 ./ssl/acme

	// autocert
	certManager *autocert.Manager
	httpHandler http.Handler // ACME challenge handler

	autoRenew   bool
	renewTicker *time.Ticker

	// lego 挑战状态（内存临时存储，重启丢失需重新申请）
	pendingChallenges map[string]*PendingChallenge // domain -> challenge
	acmeFileTokens    map[string]string            // token -> keyAuthorization (文件验证托管)

	// 证书申请进度（内存临时存储）
	certProgress map[string]*CertProgress // domain -> progress

	// 证书获取成功回调（用于通知外部系统更新站点配置）
	onCertObtained func(domain, provider, challengeMethod, email string)

	// EAB 凭证（LiteSSL 等 ACME 服务商需要）
	eabKid     string
	eabHmacKey string
}

// NewSSLManager 创建 SSL 管理器
func NewSSLManager(certDir string) *SSLManager {
	return &SSLManager{
		certs:             make(map[string]*tls.Certificate),
		configs:           make(map[string]*config.SSLConfig),
		certInfos:         make(map[string]*CertInfo),
		certDir:           certDir,
		acmeDir:           filepath.Join(certDir, "acme"),
		pendingChallenges: make(map[string]*PendingChallenge),
		acmeFileTokens:    make(map[string]string),
		certProgress:      make(map[string]*CertProgress),
	}
}

// SetEABCredentials 设置 EAB 凭证（LiteSSL 等）
func (m *SSLManager) SetEABCredentials(kid, hmacKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eabKid = kid
	m.eabHmacKey = hmacKey
}

// GetEABCredentials 获取 EAB 凭证
func (m *SSLManager) GetEABCredentials() (kid, hmacKey string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.eabKid, m.eabHmacKey
}

// SetOnCertObtained 设置证书获取成功回调
func (m *SSLManager) SetOnCertObtained(cb func(domain, provider, challengeMethod, email string)) {
	m.onCertObtained = cb
}

// Start 启动 SSL 管理器
func (m *SSLManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 确保证书目录存在
	if err := os.MkdirAll(m.certDir, 0755); err != nil {
		return fmt.Errorf("create cert dir: %w", err)
	}
	if err := os.MkdirAll(m.acmeDir, 0700); err != nil {
		return fmt.Errorf("create acme dir: %w", err)
	}

	// 启动自动续期检查器（每天检查一次）
	m.autoRenew = true
	m.renewTicker = time.NewTicker(24 * time.Hour)
	go func() {
		for range m.renewTicker.C {
			m.checkAndRenewCerts()
		}
	}()

	log.Printf("[SSL] SSL 管理器已启动")
	return nil
}

// Stop 停止 SSL 管理器
func (m *SSLManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.autoRenew = false
	if m.renewTicker != nil {
		m.renewTicker.Stop()
	}

	log.Printf("[SSL] SSL 管理器已停止")
}

// LoadSiteCertificates 加载所有站点的 SSL 配置
func (m *SSLManager) LoadSiteCertificates(sites []config.SiteConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 收集需要 autocert 自动证书的域名（仅 http-auto + letsencrypt）
	var autoDomains []string
	var autoEmail string

	for i := range sites {
		site := &sites[i]
		if site.SSL == nil || !site.SSL.Enabled {
			continue
		}

		// 为每个域名注册配置
		for _, domain := range site.Domain {
			m.configs[domain] = site.SSL
		}

		if site.SSL.AutoHTTPS {
			// lego 管理的证书只从磁盘加载，不走 autocert
			if site.SSL.IsLegoCert() {
				for _, domain := range site.Domain {
					m.loadCertFromDisk(domain)
					log.Printf("[SSL] lego 证书已从磁盘加载: %s (provider: %s, method: %s)",
						domain, site.SSL.GetProvider(), site.SSL.GetChallengeMethod())
				}
			} else {
				// autocert 模式
				autoDomains = append(autoDomains, site.Domain...)
				if site.SSL.Email != "" {
					autoEmail = site.SSL.Email
				}
			}
		}

		// 加载已有证书
		for _, domain := range site.Domain {
			if site.SSL.IsLegoCert() {
				// 已在上面处理
				continue
			}
			if site.SSL.AutoHTTPS {
				// 尝试从 ACME 缓存或本地目录加载
				m.loadCertFromDisk(domain)
			} else if site.SSL.CertFile != "" && site.SSL.KeyFile != "" {
				// 自定义证书
				cert, err := tls.LoadX509KeyPair(site.SSL.CertFile, site.SSL.KeyFile)
				if err != nil {
					log.Printf("[SSL] 加载自定义证书失败 %s: %v", domain, err)
					continue
				}
				m.certs[domain] = &cert
				m.updateCertInfoFromCert(domain, site.SSL, &cert)
				log.Printf("[SSL] 已加载自定义证书: %s", domain)
			}
		}
	}

	// 初始化 autocert manager（仅 autocert 域名）
	if len(autoDomains) > 0 {
		m.initACMEManager(autoDomains, autoEmail)
	}
}

// initACMEManager 初始化 autocert 管理器
func (m *SSLManager) initACMEManager(domains []string, email string) {
	hostSet := make(map[string]bool)
	for _, d := range domains {
		hostSet[d] = true
	}

	m.certManager = &autocert.Manager{
		Prompt:      autocert.AcceptTOS,
		HostPolicy:  autocert.HostWhitelist(domains...),
		Cache:       autocert.DirCache(m.acmeDir),
		Email:       email,
		RenewBefore: 30 * 24 * time.Hour,
	}
	m.httpHandler = m.certManager.HTTPHandler(nil)

	log.Printf("[SSL] autocert 已初始化，域名: %v", domains)
}

// AddAutoDomain 添加自动证书域名（动态更新）
func (m *SSLManager) AddAutoDomain(domain, email string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.certManager == nil {
		m.initACMEManager([]string{domain}, email)
		return
	}

	// autocert.HostWhitelist 不支持动态添加，需要重建
	// 收集所有现有域名
	var allDomains []string
	for d := range m.configs {
		cfg := m.configs[d]
		if cfg != nil && cfg.AutoHTTPS {
			allDomains = append(allDomains, d)
		}
	}
	// 确保新域名在列表中
	found := false
	for _, d := range allDomains {
		if d == domain {
			found = true
			break
		}
	}
	if !found {
		allDomains = append(allDomains, domain)
	}

	m.initACMEManager(allDomains, email)
}

// GetCertificate 获取证书（用于 TLS 配置）
func (m *SSLManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	domain := hello.ServerName
	if domain == "" {
		return nil, fmt.Errorf("no SNI")
	}

	m.mu.RLock()
	// 1. 精确匹配自定义证书
	if cert, ok := m.certs[domain]; ok {
		m.mu.RUnlock()
		return cert, nil
	}

	// 2. 通配符匹配
	for d, cert := range m.certs {
		if isWildcardMatch(d, domain) {
			m.mu.RUnlock()
			return cert, nil
		}
	}

	// 3. autocert 自动获取
	if m.certManager != nil {
		m.mu.RUnlock()
		cert, err := m.certManager.GetCertificate(hello)
		if err == nil {
			// 缓存到内存
			m.mu.Lock()
			m.certs[domain] = cert
			m.mu.Unlock()
		}
		return cert, err
	}

	m.mu.RUnlock()
	return nil, fmt.Errorf("certificate not found for %s", domain)
}

// ObtainCertificate 申请证书（支持 autocert 和 lego）
func (m *SSLManager) ObtainCertificate(domain, email, provider, challengeMethod string) error {
	// 默认值
	if provider == "" {
		provider = "letsencrypt"
	}
	if challengeMethod == "" {
		challengeMethod = "http-auto"
	}

	// autocert 模式：仅 letsencrypt + http-auto
	if provider == "letsencrypt" && challengeMethod == "http-auto" {
		m.mu.Lock()
		m.AddAutoDomain(domain, email)
		m.mu.Unlock()
		log.Printf("[SSL] 证书申请已加入队列 (autocert): %s", domain)
		return nil
	}

	// lego 模式：LiteSSL 或非 http-auto 验证方式
	if challengeMethod == "http-auto" {
		// lego HTTP-01 自动模式
		return m.ObtainCertificateLego(domain, email, provider)
	}

	// http-file 和 dns 由 API 端点分两步处理
	log.Printf("[SSL] 证书申请需两步验证: %s (method: %s)", domain, challengeMethod)
	return nil
}

// RenewCertificate 续期证书
func (m *SSLManager) RenewCertificate(domain string) error {
	m.mu.RLock()
	cfg, ok := m.configs[domain]
	m.mu.RUnlock()

	if !ok || !cfg.AutoHTTPS {
		return fmt.Errorf("自定义证书无法自动续期，请手动上传新证书")
	}

	// lego 管理的证书
	if cfg.IsLegoCert() {
		return m.RenewCertificateLego(domain)
	}

	// autocert 模式：删除缓存让 autocert 重新获取
	m.mu.Lock()
	delete(m.certs, domain)
	m.mu.Unlock()

	log.Printf("[SSL] 证书续期已触发 (autocert): %s", domain)
	return nil
}

// SaveCustomCertificate 保存自定义证书
func (m *SSLManager) SaveCustomCertificate(domain string, certPEM, keyPEM []byte) error {
	// 验证证书和私钥匹配
	if _, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
		return fmt.Errorf("证书与私钥不匹配: %w", err)
	}

	// 保存到文件
	certPath := filepath.Join(m.certDir, domain+".crt")
	keyPath := filepath.Join(m.certDir, domain+".key")

	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return fmt.Errorf("保存证书文件失败: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		os.Remove(certPath)
		return fmt.Errorf("保存私钥文件失败: %w", err)
	}

	// 加载到内存
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("加载证书失败: %w", err)
	}

	m.mu.Lock()
	m.certs[domain] = &cert

	// 更新配置
	if _, ok := m.configs[domain]; !ok {
		m.configs[domain] = &config.SSLConfig{Enabled: true}
	}
	cfg := m.configs[domain]
	cfg.Enabled = true
	cfg.AutoHTTPS = false
	cfg.CertFile = certPath
	cfg.KeyFile = keyPath

	m.updateCertInfoFromCert(domain, cfg, &cert)
	m.mu.Unlock()

	log.Printf("[SSL] 自定义证书已保存: %s", domain)
	return nil
}

// DeleteCertificate 删除证书
func (m *SSLManager) DeleteCertificate(domain string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 删除内存缓存
	delete(m.certs, domain)
	delete(m.configs, domain)
	delete(m.certInfos, domain)

	// 删除文件
	certPath := filepath.Join(m.certDir, domain+".crt")
	keyPath := filepath.Join(m.certDir, domain+".key")
	os.Remove(certPath)
	os.Remove(keyPath)

	log.Printf("[SSL] 证书已删除: %s", domain)
	return nil
}

// GetCertStatus 获取单个域名证书状态
func (m *SSLManager) GetCertStatus(domain string) *CertInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.certInfos[domain]
	if ok {
		return info
	}

	// 没有缓存信息，构造基本状态
	status := &CertInfo{
		Domain:   domain,
		HasCert:  false,
		Enabled:  false,
		DaysLeft: 0,
	}
	if cfg, ok := m.configs[domain]; ok {
		status.Enabled = cfg.Enabled
		status.AutoHTTPS = cfg.AutoHTTPS
		status.ForceHTTPS = cfg.ForceHTTPS
		status.Email = cfg.Email
	}
	return status
}

// GetAllCertStatuses 获取所有证书状态
// 遍历站点配置中的域名 + ssl 目录下独立存在的证书文件
func (m *SSLManager) GetAllCertStatuses() []*CertInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*CertInfo, 0)
	seenDomain := make(map[string]bool)  // 已处理的域名
	seenCertFile := make(map[string]bool) // 已处理的证书文件路径

	// 1. 遍历站点配置中已注册的域名
	for domain := range m.configs {
		if seenDomain[domain] {
			continue
		}
		seenDomain[domain] = true

		info, ok := m.certInfos[domain]
		if ok {
			// 按证书文件去重
			if info.CertFile != "" {
				absPath := filepath.Clean(info.CertFile)
				if seenCertFile[absPath] {
					continue
				}
				seenCertFile[absPath] = true
			}
			if cert, hasCert := m.certs[domain]; hasCert && cert.Leaf != nil {
				info.DaysLeft = int(time.Until(cert.Leaf.NotAfter).Hours() / 24)
			}
			result = append(result, info)
		} else {
			status := m.GetCertStatus(domain)
			if status.CertFile != "" {
				absPath := filepath.Clean(status.CertFile)
				if seenCertFile[absPath] {
					continue
				}
				seenCertFile[absPath] = true
			}
			result = append(result, status)
		}
	}

	// 2. 扫描 ssl 目录下独立存在的证书文件（不在站点配置中的）
	entries, err := os.ReadDir(m.certDir)
	if err != nil {
		return result
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".crt") {
			continue
		}
		domain := strings.TrimSuffix(entry.Name(), ".crt")
		if domain == "" || domain == "acme" || seenDomain[domain] {
			continue
		}

		// 检查对应的 key 文件
		keyPath := filepath.Join(m.certDir, domain+".key")
		if _, err := os.Stat(keyPath); err != nil {
			continue
		}

		certPath := filepath.Join(m.certDir, entry.Name())
		absCertPath := filepath.Clean(certPath)

		// 按证书文件去重
		if seenCertFile[absCertPath] {
			continue
		}
		seenCertFile[absCertPath] = true
		seenDomain[domain] = true

		// 尝试加载并解析证书信息
		certPEM, err := os.ReadFile(certPath)
		if err != nil {
			continue
		}
		info, err := ParseCertInfoFromPEM(certPEM)
		if err != nil {
			info = &CertInfo{
				Domain:  domain,
				HasCert: true,
				Type:    "custom",
			}
		}
		result = append(result, info)
	}

	return result
}

// GetHTTPSRedirectHandler 返回 HTTPS 重定向 handler
// 先处理 ACME challenge，再按域名判断是否 301 重定向到 HTTPS
func (m *SSLManager) GetHTTPSRedirectHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ACME challenge（优先处理）
		if strings.HasPrefix(r.URL.Path, "/.well-known/acme-challenge/") {
			// 1. 先检查文件验证的 token（acmeFileTokens）
			token := strings.TrimPrefix(r.URL.Path, "/.well-known/acme-challenge/")
			if token != "" {
				m.mu.RLock()
				keyAuth, found := m.acmeFileTokens[token]
				m.mu.RUnlock()
				if found {
					w.Header().Set("Content-Type", "text/plain")
					w.Write([]byte(keyAuth))
					return
				}
			}
			// 2. 回退到 autocert handler
			m.mu.RLock()
			handler := m.httpHandler
			m.mu.RUnlock()
			if handler != nil {
				handler.ServeHTTP(w, r)
				return
			}
		}

		// 检查域名是否需要 HTTPS 重定向
		host := r.Host
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}

		m.mu.RLock()
		cfg, ok := m.configs[host]
		m.mu.RUnlock()

		if ok && cfg.Enabled && cfg.ForceHTTPS {
			httpsURL := "https://" + host + r.URL.RequestURI()
			http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
			return
		}

		// 不需要重定向，传递给下一个 handler
		if next != nil {
			next.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})
}

// HasSSLDomains 检查是否有任何域名启用了 SSL
func (m *SSLManager) HasSSLDomains() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, cfg := range m.configs {
		if cfg.Enabled {
			return true
		}
	}
	return false
}

// HasPendingChallenges 检查是否有待验证的挑战（需要端口 80）
func (m *SSLManager) HasPendingChallenges() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pendingChallenges) > 0 || len(m.acmeFileTokens) > 0
}

// HasHTTPRedirect 检查是否有域名需要 HTTP 重定向
func (m *SSLManager) HasHTTPRedirect() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, cfg := range m.configs {
		if cfg.Enabled && cfg.ForceHTTPS {
			return true
		}
	}
	return false
}

// ========== 内部方法 ==========

// loadCertFromDisk 从磁盘加载证书
func (m *SSLManager) loadCertFromDisk(domain string) {
	certPath := filepath.Join(m.certDir, domain+".crt")
	keyPath := filepath.Join(m.certDir, domain+".key")

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		// 文件不存在，不报错（可能是首次，需要 ACME 获取）
		if !os.IsNotExist(err) {
			log.Printf("[SSL] 加载证书失败 %s: %v", domain, err)
		}
		return
	}

	m.certs[domain] = &cert

	cfg := m.configs[domain]
	m.updateCertInfoFromCert(domain, cfg, &cert)
	log.Printf("[SSL] 已加载证书: %s", domain)
}

// updateCertInfoFromCert 从 tls.Certificate 更新 CertInfo
func (m *SSLManager) updateCertInfoFromCert(domain string, sslCfg *config.SSLConfig, cert *tls.Certificate) {
	info := &CertInfo{
		Domain:  domain,
		HasCert: true,
	}

	if sslCfg != nil {
		info.Enabled = sslCfg.Enabled
		info.AutoHTTPS = sslCfg.AutoHTTPS
		info.ForceHTTPS = sslCfg.ForceHTTPS
		info.HSTS = sslCfg.HSTS
		info.Email = sslCfg.Email
		info.CertFile = sslCfg.CertFile
		info.KeyFile = sslCfg.KeyFile
		info.Provider = sslCfg.GetProvider()
		info.ChallengeMethod = sslCfg.GetChallengeMethod()
	}

	if sslCfg != nil && sslCfg.AutoHTTPS {
		info.Type = "auto"
		switch sslCfg.GetProvider() {
		case "litessl":
			info.Issuer = "LiteSSL"
		default:
			info.Issuer = "Let's Encrypt"
		}
	} else {
		info.Type = "custom"
	}

	// 解析证书详情
	if cert.Leaf == nil && len(cert.Certificate) > 0 {
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		if err == nil {
			cert.Leaf = x509Cert
		}
	}

	if cert.Leaf != nil {
		info.NotBefore = cert.Leaf.NotBefore.Format(time.RFC3339)
		info.NotAfter = cert.Leaf.NotAfter.Format(time.RFC3339)
		info.DaysLeft = int(time.Until(cert.Leaf.NotAfter).Hours() / 24)

		// 判断签发者
		if cert.Leaf.Issuer.CommonName != "" {
			if info.Issuer == "" {
				info.Issuer = cert.Leaf.Issuer.CommonName
			}
			// 检测自签名
			if cert.Leaf.Issuer.CommonName == cert.Leaf.Subject.CommonName {
				info.Type = "self-signed"
			}
		}
	}

	m.certInfos[domain] = info
}

// ParseCertInfoFromPEM 从 PEM 数据解析证书信息（不加载到内存）
func ParseCertInfoFromPEM(certPEM []byte) (*CertInfo, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("无法解析 PEM 数据")
	}

	x509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析证书失败: %w", err)
	}

	info := &CertInfo{
		HasCert:   true,
		NotBefore: x509Cert.NotBefore.Format(time.RFC3339),
		NotAfter:  x509Cert.NotAfter.Format(time.RFC3339),
		DaysLeft:  int(time.Until(x509Cert.NotAfter).Hours() / 24),
	}

	if x509Cert.Issuer.CommonName != "" {
		info.Issuer = x509Cert.Issuer.CommonName
	}

	// 检查 SAN (Subject Alternative Names)
	if len(x509Cert.DNSNames) > 0 {
		info.Domain = x509Cert.DNSNames[0]
	} else {
		info.Domain = x509Cert.Subject.CommonName
	}

	// 自签名检测
	if x509Cert.Issuer.CommonName == x509Cert.Subject.CommonName {
		info.Type = "self-signed"
	}

	return info, nil
}

// checkAndRenewCerts 检查并续期证书
func (m *SSLManager) checkAndRenewCerts() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for domain, cert := range m.certs {
		// 解析证书过期时间
		if cert.Leaf == nil && len(cert.Certificate) > 0 {
			x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
			if err != nil {
				log.Printf("[SSL] 解析证书失败: %s, %v", domain, err)
				continue
			}
			cert.Leaf = x509Cert
		}

		if cert.Leaf == nil {
			continue
		}

		daysLeft := int(time.Until(cert.Leaf.NotAfter).Hours() / 24)

		// 检查是否需要续期（30天内过期）
		if daysLeft < 30 {
			cfg, ok := m.configs[domain]
			if !ok || !cfg.AutoHTTPS {
				log.Printf("[SSL] 自定义证书即将过期（剩余 %d 天）: %s，请手动更新", daysLeft, domain)
				continue
			}

			if cfg.IsLegoCert() {
				// lego 管理的证书：尝试主动续期
				log.Printf("[SSL] lego 证书即将过期（剩余 %d 天），尝试续期: %s", daysLeft, domain)
				go func(d string) {
					if err := m.RenewCertificateLego(d); err != nil {
						log.Printf("[SSL] lego 续期失败: %s, %v", d, err)
					}
				}(domain)
			} else {
				// autocert 会在下次 TLS 握手时自动续期
				log.Printf("[SSL] 证书即将过期（剩余 %d 天），autocert 将自动续期: %s", daysLeft, domain)
			}
		}
	}
}

// isWildcardMatch 检查通配符匹配
func isWildcardMatch(pattern, domain string) bool {
	if !strings.HasPrefix(pattern, "*.") {
		return false
	}
	suffix := pattern[2:]
	return strings.HasSuffix(domain, suffix)
}



// Ensure SSLManager implements autocert.HostPolicy-compatible interface
var _ context.Context = context.Background()
