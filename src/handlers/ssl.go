package handlers

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pixelbeast/src/config"
)

// SSLManager SSL 管理器
type SSLManager struct {
	mu          sync.RWMutex
	certs       map[string]*tls.Certificate // domain -> cert
	configs     map[string]*config.SSLConfig // domain -> config
	certDir     string                      // 证书存储目录
	autoRenew   bool
	renewTicker *time.Ticker
}

// NewSSLManager 创建 SSL 管理器
func NewSSLManager(certDir string) *SSLManager {
	return &SSLManager{
		certs:   make(map[string]*tls.Certificate),
		configs: make(map[string]*config.SSLConfig),
		certDir: certDir,
	}
}

// Start 启动 SSL 管理器
func (m *SSLManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 确保证书目录存在
	if err := os.MkdirAll(m.certDir, 0755); err != nil {
		return fmt.Errorf("create cert dir: %w", err)
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

// LoadCertificate 加载证书
func (m *SSLManager) LoadCertificate(domain string, cfg *config.SSLConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 保存配置
	m.configs[domain] = cfg

	// 如果禁用 SSL，移除证书
	if !cfg.Enabled {
		delete(m.certs, domain)
		return nil
	}

	var cert tls.Certificate
	var err error

	// 加载证书文件
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err = tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return fmt.Errorf("load certificate files: %w", err)
		}
	} else {
		// 尝试从默认位置加载
		certPath := filepath.Join(m.certDir, domain+".crt")
		keyPath := filepath.Join(m.certDir, domain+".key")

		cert, err = tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			// 如果是 AutoHTTPS 模式，尝试申请证书
			if cfg.AutoHTTPS {
				log.Printf("[SSL] 证书不存在，将自动申请: %s", domain)
				return nil // 稍后申请
			}
			return fmt.Errorf("load default certificate: %w", err)
		}
	}

	m.certs[domain] = &cert
	log.Printf("[SSL] 已加载证书: %s", domain)
	return nil
}

// GetCertificate 获取证书（用于 TLS 配置）
func (m *SSLManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	domain := hello.ServerName
	if domain == "" {
		return nil, fmt.Errorf("no SNI")
	}

	// 精确匹配
	if cert, ok := m.certs[domain]; ok {
		return cert, nil
	}

	// 通配符匹配
	for d, cert := range m.certs {
		if isWildcardMatch(d, domain) {
			return cert, nil
		}
	}

	return nil, fmt.Errorf("certificate not found for %s", domain)
}

// GetCertificateByDomain 根据域名获取证书
func (m *SSLManager) GetCertificateByDomain(domain string) (*tls.Certificate, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cert, ok := m.certs[domain]
	return cert, ok
}

// checkAndRenewCerts 检查并续期证书
func (m *SSLManager) checkAndRenewCerts() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for domain, cert := range m.certs {
		// 解析证书过期时间
		if cert.Leaf == nil {
			// 解析 PEM 证书
			if len(cert.Certificate) > 0 {
				x509Cert, err := parseX509Cert(cert.Certificate[0])
				if err != nil {
					log.Printf("[SSL] 解析证书失败: %s, %v", domain, err)
					continue
				}
				cert.Leaf = x509Cert
			} else {
				continue
			}
		}

		// 检查是否需要续期（30天内过期）
		if time.Until(cert.Leaf.NotAfter) < 30*24*time.Hour {
			log.Printf("[SSL] 证书即将过期，尝试续期: %s", domain)
			// TODO: 实现 Let's Encrypt 续期
		}
	}
}

// ObtainCertificate 申请证书（Let's Encrypt）
func (m *SSLManager) ObtainCertificate(domain, email string) error {
	// TODO: 实现 Let's Encrypt 证书申请
	log.Printf("[SSL] 申请证书: %s (邮箱: %s)", domain, email)
	return fmt.Errorf("Let's Encrypt 证书申请功能待实现")
}

// RenewCertificate 续期证书
func (m *SSLManager) RenewCertificate(domain string) error {
	// TODO: 实现 Let's Encrypt 证书续期
	log.Printf("[SSL] 续期证书: %s", domain)
	return fmt.Errorf("Let's Encrypt 证书续期功能待实现")
}

// isWildcardMatch 检查通配符匹配
func isWildcardMatch(pattern, domain string) bool {
	if !strings.HasPrefix(pattern, "*.") {
		return false
	}

	suffix := pattern[2:]
	return strings.HasSuffix(domain, suffix)
}

// parseX509Cert 解析 X.509 证书
func parseX509Cert(data []byte) (*x509.Certificate, error) {
	return x509.ParseCertificate(data)
}

// GetCertStatus 获取证书状态
func (m *SSLManager) GetCertStatus(domain string) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := map[string]interface{}{
		"domain":    domain,
		"enabled":   false,
		"has_cert":  false,
		"auto":      false,
		"expires":   nil,
		"days_left": 0,
	}

	cfg, ok := m.configs[domain]
	if ok {
		status["enabled"] = cfg.Enabled
		status["auto"] = cfg.AutoHTTPS
	}

	cert, ok := m.certs[domain]
	if ok && cert.Leaf != nil {
		status["has_cert"] = true
		status["expires"] = cert.Leaf.NotAfter
		status["days_left"] = int(time.Until(cert.Leaf.NotAfter).Hours() / 24)
	}

	return status
}
